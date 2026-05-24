package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethan-blue/open-code-go-tools/internal/config"
	"github.com/ethan-blue/open-code-go-tools/internal/preferences"
	"github.com/ethan-blue/open-code-go-tools/internal/proxy"
	"github.com/getlantern/systray"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx        context.Context
	srv        *proxy.Server
	cancelFunc context.CancelFunc
	actionCh   chan trayAction
	quitCh     chan struct{} // signals menu click listener goroutine to exit

	// Systray menu items
	mShow     *systray.MenuItem
	mHide     *systray.MenuItem
	mSettings *systray.MenuItem
	mAbout    *systray.MenuItem
	mQuit     *systray.MenuItem

	// Allows explicit quit actions to bypass the close-to-tray prompt.
	forceQuit     atomic.Bool
	setupTrayOnce sync.Once
	exitOnce      sync.Once
}

type trayAction int

const (
	trayActionShow trayAction = iota + 1
	trayActionHide
	trayActionSettings
	trayActionAbout
	trayActionQuit
)

// NewApp creates a new App struct instance
func NewApp() *App {
	return &App{actionCh: make(chan trayAction, 16), quitCh: make(chan struct{})}
}

//go:embed build/appicon.png
var appIconPng []byte

//go:embed build/windows/icon.ico
var appIconIco []byte

func (a *App) setupSystray() {
	a.setupTrayOnce.Do(func() {
		onReady := func() {
			if runtime.GOOS == "windows" {
				systray.SetIcon(appIconIco)
			} else {
				systray.SetIcon(appIconPng)
			}

			// Use permanent bilingual titles to ensure 100% stability on Windows and avoid Win32 menu leaks
			systray.SetTitle("ocgt")
			systray.SetTooltip("ocgt 控制面板 / Control Panel")

			a.mShow = systray.AddMenuItem("显示控制面板 / Show Panel", "显示主窗口 / Show Main Window")
			a.mHide = systray.AddMenuItem("隐藏控制面板 / Hide Panel", "隐藏主窗口 / Hide to Tray")
			a.mSettings = systray.AddMenuItem("打开设置 / Open Settings", "打开设置页面 / Open Settings Page")
			a.mAbout = systray.AddMenuItem("关于 ocgt / About", "关于此程序 / About App")
			systray.AddSeparator()
			a.mQuit = systray.AddMenuItem("退出程序 / Quit", "彻底退出代理服务 / Quit Application")

			go func() {
				for {
					select {
					case <-a.mShow.ClickedCh:
						a.enqueueTrayAction(trayActionShow)
					case <-a.mHide.ClickedCh:
						a.enqueueTrayAction(trayActionHide)
					case <-a.mSettings.ClickedCh:
						a.enqueueTrayAction(trayActionSettings)
					case <-a.mAbout.ClickedCh:
						a.enqueueTrayAction(trayActionAbout)
					case <-a.mQuit.ClickedCh:
						a.enqueueTrayAction(trayActionQuit)
					case <-a.quitCh:
						return
					}
				}
			}()
		}

		onExit := func() {}
		if runtime.GOOS == "windows" {
			// On Windows, systray.Register() must own the calling thread.
			// Wails already owns the main thread, so we use Run() which creates
			// its own dedicated OS thread with LockOSThread internally.
			// This completely prevents the right-click menu deadlock from thread contention.
			go systray.Run(onReady, onExit)
			return
		}
		go systray.Run(onReady, onExit)
	})
}

func (a *App) showMainWindow() {
	if a.ctx == nil {
		return
	}
	wailsruntime.WindowShow(a.ctx)
	if wailsruntime.WindowIsMinimised(a.ctx) {
		wailsruntime.WindowUnminimise(a.ctx)
	}
	wailsruntime.WindowCenter(a.ctx)
}

func (a *App) hideMainWindow() {
	if a.ctx == nil {
		return
	}
	wailsruntime.WindowMinimise(a.ctx)
	wailsruntime.WindowHide(a.ctx)
}

func (a *App) enqueueTrayAction(action trayAction) {
	select {
	case a.actionCh <- action:
	default:
		go func() { a.actionCh <- action }()
	}
}

func (a *App) actionLoop() {
	for action := range a.actionCh {
		switch action {
		case trayActionShow:
			a.showMainWindow()
		case trayActionHide:
			a.hideMainWindow()
		case trayActionSettings:
			a.showMainWindow()
			if a.ctx != nil {
				wailsruntime.EventsEmit(a.ctx, "nav-to-settings")
			}
		case trayActionAbout:
			if a.ctx != nil {
				wailsruntime.EventsEmit(a.ctx, "show-about-dialog")
			}
		case trayActionQuit:
			a.exitNow()
		}
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	go a.actionLoop()

	// Start Go proxy server in the background!
	go func() {
		// 1. Auto-init config if it doesn't exist
		defaultPath, err := config.DefaultPath()
		if err == nil {
			if _, err := os.Stat(defaultPath); os.IsNotExist(err) {
				_, _ = config.WriteExample("", false)
			}
		}

		// 2. Load config
		cfg, err := config.Load("")
		if err != nil {
			log.Println("[GUI proxy] config load error:", err)
			return
		}

		// 3. Create server
		srv, err := proxy.New(cfg)
		if err != nil {
			log.Println("[GUI proxy] server creation error:", err)
			return
		}
		srv.SetConfigPath(defaultPath)
		a.srv = srv

		// 4. Listen and Serve with cancellation context
		proxyCtx, cancel := context.WithCancel(context.Background())
		a.cancelFunc = cancel

		log.Println("[GUI proxy] starting background proxy server on http://" + cfg.Listen)
		if err := srv.ListenAndServe(proxyCtx); err != nil {
			log.Println("[GUI proxy] server stopped:", err)
		}
	}()
}

// domReady is called when the frontend DOM is fully loaded and ready.
func (a *App) domReady(ctx context.Context) {
	a.ctx = ctx
	// Force the main window to be shown, unminimized, centered and focused on startup
	a.showMainWindow()

	// Initialize the system tray after the Wails WebView2 is fully loaded.
	// A short delay prevents Windows message pump race conditions on startup.
	// Note: setupSystray uses systray.Run() which manages its own dedicated
	// OS thread — no LockOSThread needed here.
	go func() {
		time.Sleep(500 * time.Millisecond)
		a.setupSystray()
	}()
}

// shutdown is called when the app closes
func (a *App) shutdown(ctx context.Context) {
	// Signal menu click listener goroutine to exit
	close(a.quitCh)
	// systray.Quit must be called exactly once during app teardown.
	// Do not call it anywhere else (e.g., beforeClose) to avoid Win32 message-pump deadlocks.
	systray.Quit()
	if a.cancelFunc != nil {
		log.Println("[GUI proxy] shutting down background proxy server...")
		a.cancelFunc()
	}
}

// GetListenAddress returns the actual proxy listen address dynamically
// GetLocalToken returns the local auth token for API requests.
func (a *App) GetLocalToken() string {
	if a.srv == nil {
		return ""
	}
	return a.srv.LocalToken()
}

func (a *App) GetListenAddress() string {
	if a.srv != nil {
		return a.srv.ListenAddress()
	}
	// Try loading config to get the address if server is not fully initialized yet
	cfg, err := config.Load("")
	if err == nil && cfg.Listen != "" {
		return cfg.Listen
	}
	return "127.0.0.1:8787" // default fallback
}

func (a *App) localProxyAuthToken() string {
	if token := a.GetLocalToken(); token != "" {
		return token
	}
	cfg, err := config.Load("")
	if err == nil {
		return cfg.LocalAuthToken
	}
	return ""
}

// SaveProfileConfig saves API key, model aliases, timeout, and thinking settings.
func (a *App) SaveProfileConfig(profileName, apiKey, defaultModel, sonnetAlias, haikuAlias, opusAlias, timeoutSeconds, thinkingBudgetTokens string) string {
	// 1. Resolve path
	path, err := config.DefaultPath()
	if err != nil {
		return "resolve path error: " + err.Error()
	}

	// 2. Load config
	cfg, err := config.Load(path)
	if err != nil {
		return "load error: " + err.Error()
	}

	// 3. Find and update profile
	p, ok := cfg.Profiles[profileName]
	if !ok {
		return "profile not found: " + profileName
	}

	if apiKey != "" && !isMaskedAPIKey(apiKey) {
		p.APIKey = apiKey
	}
	p.DefaultModel = defaultModel
	if p.ModelAliases == nil {
		p.ModelAliases = make(map[string]string)
	}
	p.ModelAliases["sonnet"] = sonnetAlias
	p.ModelAliases["haiku"] = haikuAlias
	p.ModelAliases["opus"] = opusAlias
	cfg.Profiles[profileName] = p
	if timeoutSeconds != "" {
		timeout, err := strconv.Atoi(timeoutSeconds)
		if err != nil {
			return "request timeout must be a number of seconds"
		}
		cfg.RequestTimeoutSeconds = timeout
	}
	if thinkingBudgetTokens != "" {
		budget, err := strconv.Atoi(thinkingBudgetTokens)
		if err != nil {
			return "thinking budget must be a number of tokens"
		}
		cfg.MaxThinkingBudgetTokens = budget
	}
	if err := cfg.Validate(); err != nil {
		return "validation error: " + err.Error()
	}

	// 4. Save config
	if err := cfg.Save(path); err != nil {
		return "save error: " + err.Error()
	}

	// 5. Update server config in-memory if running
	if a.srv != nil {
		a.srv.ApplyConfig(cfg)
	}
	if err := syncClaudeSettings(a.claudeCodeEnv()); err != nil {
		return "sync Claude settings error: " + err.Error()
	}

	return "success"
}

// InstallClaudeUserEnv persists Claude Code environment variables for new shells.
func (a *App) InstallClaudeUserEnv() string {
	env := a.claudeCodeEnv()

	for _, name := range legacyClaudeEnvNames() {
		if err := unsetUserEnvironment(name); err != nil {
			return "unset " + name + " error: " + err.Error()
		}
	}
	for name, value := range env {
		if err := setUserEnvironment(name, value); err != nil {
			return "set " + name + " error: " + err.Error()
		}
	}
	if err := syncClaudeSettings(env); err != nil {
		return "sync Claude settings error: " + err.Error()
	}
	return "success"
}

func (a *App) claudeCodeEnv() map[string]string {
	listenAddr := a.GetListenAddress()
	activeProfile := "opencode-go"
	thinkingBudget := config.DefaultMaxThinkingBudgetTokens

	path, err := config.DefaultPath()
	if err == nil {
		cfg, err := config.Load(path)
		if err == nil {
			activeProfile = cfg.ActiveProfile
			thinkingBudget = cfg.ThinkingBudgetTokens()
		}
	}

	env := map[string]string{
		"ANTHROPIC_BASE_URL":       "http://" + listenAddr,
		"ANTHROPIC_API_KEY":        "ocgt-local-proxy",
		"ANTHROPIC_CUSTOM_HEADERS": "X-Ocgt-Profile: " + activeProfile,
		"OCGT_PROFILE":             activeProfile,
	}
	if token := a.localProxyAuthToken(); token != "" {
		env["ANTHROPIC_AUTH_TOKEN"] = token
	}
	applyClaudeThinkingEnv(env, thinkingBudget)
	return env
}

// sanitizeEnvValue validates that a value is safe to pass as an environment variable.
// It only allows alphanumeric characters, dash, underscore, dot, colon, slash, and space.
func sanitizeEnvValue(value, name string) error {
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '-' || r == '_' || r == '.' || r == ':' || r == '/' || r == ' ':
		default:
			return fmt.Errorf("invalid character %q in %s", r, name)
		}
	}
	return nil
}

// LaunchClaudeTerminal spawns a new terminal window preconfigured with the Claude Code proxy environment
func (a *App) LaunchClaudeTerminal(shell string, lang string) string {
	listenAddr := a.GetListenAddress()
	activeProfile := "opencode-go"
	defaultModel := "kimi-k2.6"
	thinkingBudget := config.DefaultMaxThinkingBudgetTokens

	// Try loading from config to get the latest
	path, err := config.DefaultPath()
	if err == nil {
		cfg, err := config.Load(path)
		if err == nil {
			activeProfile = cfg.ActiveProfile
			thinkingBudget = cfg.ThinkingBudgetTokens()
			if p, ok := cfg.Profiles[activeProfile]; ok {
				if p.DefaultModel != "" {
					defaultModel = p.DefaultModel
				}
			}
		}
	}

	// Validate inputs to prevent command injection
	if err := sanitizeEnvValue(activeProfile, "profile name"); err != nil {
		return "invalid profile name: " + err.Error()
	}
	if err := sanitizeEnvValue(defaultModel, "model name"); err != nil {
		return "invalid model name: " + err.Error()
	}
	thinkingEnv := map[string]string{}
	applyClaudeThinkingEnv(thinkingEnv, thinkingBudget)
	thinkingTokenValue := thinkingEnv["MAX_THINKING_TOKENS"]
	disableThinking := thinkingEnv["CLAUDE_CODE_DISABLE_THINKING"] == "1"
	localAuthToken := a.localProxyAuthToken()

	baseURL := "http://" + listenAddr

	// Localized greeting strings
	welcomeTitle := "[ocgt] Claude Code proxy terminal successfully launched!"
	proxyLabel := "Current Proxy: "
	modelLabel := "Current Model: "
	actionHint := "Please type 'claude' below to start coding:"
	if lang == "zh" {
		welcomeTitle = "[ocgt] Claude Code 代理终端已成功拉起！"
		proxyLabel = "当前代理: "
		modelLabel = "当前模型: "
		actionHint = "请在下方直接输入: claude"
	}

	// Sanitize ALL dynamic values before any shell interpolation
	if err := sanitizeEnvValue(listenAddr, "listen address"); err != nil {
		return "invalid listen address: " + err.Error()
	}
	if err := sanitizeEnvValue(thinkingTokenValue, "thinking token value"); err != nil {
		return "invalid thinking token: " + err.Error()
	}
	if localAuthToken != "" {
		if err := sanitizeEnvValue(localAuthToken, "local auth token"); err != nil {
			return "invalid local auth token: " + err.Error()
		}
	}

	switch runtime.GOOS {
	case "windows":
		// SECURITY: Env vars are passed via cmd.Env (child process inherits them).
		// Shell scripts reference $env:VAR (PowerShell) or %VAR% (CMD) instead of
		// interpolating values into strings — prevents command injection.
		env := []string{
			fmt.Sprintf("ANTHROPIC_BASE_URL=%s", baseURL),
			"ANTHROPIC_API_KEY=ocgt-local-proxy",
			fmt.Sprintf("ANTHROPIC_CUSTOM_HEADERS=X-Ocgt-Profile: %s", activeProfile),
			fmt.Sprintf("MAX_THINKING_TOKENS=%s", thinkingTokenValue),
			fmt.Sprintf("OCGT_DEFAULT_MODEL=%s", defaultModel),
		}
		if localAuthToken != "" {
			env = append(env, fmt.Sprintf("ANTHROPIC_AUTH_TOKEN=%s", localAuthToken))
		}
		if disableThinking {
			env = append(env, "CLAUDE_CODE_DISABLE_THINKING=1")
		}

		if shell == "cmd" {
			// CMD: Use %VAR% expansion — env vars already set via cmd.Env
			cmd := exec.Command("cmd.exe", "/c", "start", "cmd.exe", "/k",
				"echo =========================================================&& "+
					"echo  "+welcomeTitle+"&& "+
					"echo  "+proxyLabel+"%ANTHROPIC_BASE_URL%&& "+
					"echo  "+modelLabel+"%OCGT_DEFAULT_MODEL% (proxy fallback)&& "+
					"echo  "+actionHint+"&& "+
					"echo =========================================================&& echo.")
			cmd.Env = mergedClaudeProcessEnv(env, disableThinking)
			if err := cmd.Run(); err != nil {
				return "launch cmd error: " + err.Error()
			}
		} else {
			// PowerShell: Use $env:VAR references — values already in process env
			psScript := "Remove-Item Env:ANTHROPIC_MODEL -ErrorAction SilentlyContinue; " +
				powershellThinkingDisableScript(disableThinking) +
				"Clear-Host; " +
				"Write-Host '=========================================================' -ForegroundColor Cyan; " +
				"Write-Host ('  " + welcomeTitle + "') -ForegroundColor Green; " +
				"Write-Host ('  " + proxyLabel + "' + $env:ANTHROPIC_BASE_URL) -ForegroundColor Gray; " +
				"Write-Host ('  " + modelLabel + "' + $env:OCGT_DEFAULT_MODEL + ' (proxy fallback)') -ForegroundColor Gray; " +
				"Write-Host ('  " + actionHint + "') -ForegroundColor Green; " +
				"Write-Host '=========================================================' -ForegroundColor Cyan; " +
				"Write-Host ''"
			cmd := exec.Command("powershell.exe", "-NoExit", "-Command", psScript)
			cmd.Env = mergedClaudeProcessEnv(env, disableThinking)
			if err := cmd.Start(); err != nil {
				return "launch powershell error: " + err.Error()
			}
		}
		return "success"
	case "darwin":
		// macOS: Terminal.app doesn't inherit our env, so use export commands
		// but with all dynamic values validated above
		authScript := "unset ANTHROPIC_AUTH_TOKEN && "
		if localAuthToken != "" {
			authScript = fmt.Sprintf("export ANTHROPIC_AUTH_TOKEN='%s' && ", localAuthToken)
		}
		script := fmt.Sprintf(
			`tell application "Terminal" to do script "unset ANTHROPIC_MODEL && export ANTHROPIC_BASE_URL='%s' && export ANTHROPIC_API_KEY='ocgt-local-proxy' && %sexport ANTHROPIC_CUSTOM_HEADERS='X-Ocgt-Profile: %s' && export MAX_THINKING_TOKENS='%s' && export OCGT_DEFAULT_MODEL='%s' && %sclear && echo '=========================================================' && echo ' %s' && echo ' %s$ANTHROPIC_BASE_URL' && echo ' %s$OCGT_DEFAULT_MODEL (proxy fallback)' && echo ' %s' && echo '=========================================================' && echo ''"`,
			baseURL, authScript, activeProfile, thinkingTokenValue, defaultModel, shellThinkingDisableScript(disableThinking),
			welcomeTitle, proxyLabel, modelLabel, actionHint)
		cmd := exec.Command("osascript", "-e", script)
		if err := cmd.Run(); err != nil {
			return "launch terminal error: " + err.Error()
		}
		return "success"
	default:
		return "unsupported operating system for automatic terminal launch"
	}
}

func unsetUserEnvironment(name string) error {
	if err := os.Unsetenv(name); err != nil {
		return err
	}
	switch runtime.GOOS {
	case "windows":
		return unsetWindowsUserEnvironment(name)
	case "darwin":
		return nil
	default:
		return nil
	}
}

func legacyClaudeEnvNames() []string {
	return []string{
		"ANTHROPIC_AUTH_TOKEN",
		"ANTHROPIC_DEFAULT_SONNET_MODEL_NAME",
		"ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME",
		"ANTHROPIC_DEFAULT_OPUS_MODEL_NAME",
		"ANTHROPIC_DEFAULT_SONNET_MODEL",
		"ANTHROPIC_DEFAULT_HAIKU_MODEL",
		"ANTHROPIC_DEFAULT_OPUS_MODEL",
		"ANTHROPIC_MODEL",
		"CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS",
		"CLAUDE_CODE_ENABLE_GATEWAY_MODEL_DISCOVERY",
		"CLAUDE_CODE_DISABLE_THINKING",
	}
}

func applyClaudeThinkingEnv(env map[string]string, budgetTokens int) {
	if budgetTokens < 0 {
		env["MAX_THINKING_TOKENS"] = "0"
		env["CLAUDE_CODE_DISABLE_THINKING"] = "1"
		return
	}
	if budgetTokens == 0 {
		budgetTokens = config.DefaultMaxThinkingBudgetTokens
	}
	env["MAX_THINKING_TOKENS"] = strconv.Itoa(budgetTokens)
}

func mergedClaudeProcessEnv(overrides []string, disableThinking bool) []string {
	drop := map[string]bool{
		"ANTHROPIC_AUTH_TOKEN":         true,
		"ANTHROPIC_BASE_URL":           true,
		"ANTHROPIC_API_KEY":            true,
		"ANTHROPIC_CUSTOM_HEADERS":     true,
		"ANTHROPIC_MODEL":              true,
		"MAX_THINKING_TOKENS":          true,
		"CLAUDE_CODE_DISABLE_THINKING": !disableThinking,
	}
	out := make([]string, 0, len(os.Environ())+len(overrides))
	for _, item := range os.Environ() {
		name, _, found := strings.Cut(item, "=")
		if found && drop[name] {
			continue
		}
		out = append(out, item)
	}
	return append(out, overrides...)
}

func powershellThinkingDisableScript(disabled bool) string {
	if disabled {
		return "$env:CLAUDE_CODE_DISABLE_THINKING='1'; "
	}
	return "Remove-Item Env:CLAUDE_CODE_DISABLE_THINKING -ErrorAction SilentlyContinue; "
}

func shellThinkingDisableScript(disabled bool) string {
	if disabled {
		return "export CLAUDE_CODE_DISABLE_THINKING='1' && "
	}
	return "unset CLAUDE_CODE_DISABLE_THINKING && "
}

// OpenConfigLocation opens the directory containing the config file
func (a *App) OpenConfigLocation() string {
	path, err := config.DefaultPath()
	if err != nil {
		return "resolve path error: " + err.Error()
	}
	dir := filepath.Dir(path)

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("explorer.exe", dir)
	case "darwin":
		cmd = exec.Command("open", dir)
	default:
		return "unsupported operating system"
	}

	if err := cmd.Start(); err != nil {
		return "open error: " + err.Error()
	}
	return "success"
}

func setUserEnvironment(name, value string) error {
	if err := os.Setenv(name, value); err != nil {
		return err
	}
	switch runtime.GOOS {
	case "windows":
		return setWindowsUserEnvironment(name, value)
	case "darwin":
		return nil
	default:
		return nil
	}
}

// claudeSettingsPreserveFields lists top-level keys in ~/.claude/settings.json that
// must survive across tool switches.  Third-party tools like CC-Switch overwrite the
// entire file with only their "env" block, erasing permissions, plugins, etc.  When
// ocgt syncs settings it restores any missing preserve-fields from the last known-good
// backup.
var claudeSettingsPreserveFields = []string{
	"permissions",
	"model",
	"enabledPlugins",
	"statusLine",
	"allowedTools",
}

func syncClaudeSettings(env map[string]string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	settingsPath := filepath.Join(home, ".claude", "settings.json")
	backupPath := filepath.Join(home, ".claude", "settings.json.ocgt-bak")

	settings := map[string]any{}
	if data, err := os.ReadFile(settingsPath); err == nil && len(data) > 0 {
		if err := json.Unmarshal(data, &settings); err != nil {
			return err
		}
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}

	// If a previous ocgt backup exists, restore any preserve-fields that are
	// missing from the current settings (e.g. CC-Switch wiped them).
	if backup, err := os.ReadFile(backupPath); err == nil && len(backup) > 0 {
		bakSettings := map[string]any{}
		if json.Unmarshal(backup, &bakSettings) == nil {
			for _, key := range claudeSettingsPreserveFields {
				if _, exists := settings[key]; !exists {
					if val, ok := bakSettings[key]; ok {
						settings[key] = val
					}
				}
			}
		}
	}

	envMap, _ := settings["env"].(map[string]any)
	if envMap == nil {
		envMap = map[string]any{}
	}
	for _, name := range legacyClaudeEnvNames() {
		delete(envMap, name)
	}

	for key, value := range env {
		envMap[key] = value
	}
	settings["env"] = envMap

	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o700); err != nil {
		return err
	}
	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(settingsPath, append(out, '\n'), 0o600); err != nil {
		return err
	}

	// Keep a backup of the full settings (including preserve-fields) so that a
	// future external overwrite can be repaired on the next sync.
	if err := os.WriteFile(backupPath, append(out, '\n'), 0o600); err != nil {
		log.Printf("ocgt: failed to write settings backup: %v", err)
	}

	return nil
}

// beforeClose is called when the user clicks the 'X' button.
// IMPORTANT: On Windows, calling wailsruntime.MessageDialog inside this callback
// is unreliable and causes deadlocks because the window is mid-close.
// Instead we always prevent the close here, then emit an event to the frontend
// which shows a premium custom HTML modal to let the user decide.
func (a *App) beforeClose(ctx context.Context) bool {
	if a.forceQuit.Load() {
		return false
	}

	prefs, err := preferences.Load("")
	closeBehavior := preferences.DefaultCloseBehavior
	if err == nil {
		closeBehavior = prefs.CloseBehavior
	}

	switch closeBehavior {
	case "exit":
		a.forceQuit.Store(true)
		// Directly exit — no dialog needed
		return false
	case "minimize":
		// Silently hide to tray — no dialog needed
		go func() {
			time.Sleep(50 * time.Millisecond)
			a.enqueueTrayAction(trayActionHide)
		}()
		return true
	default: // "prompt"
		// Emit event to frontend so it can show its own custom HTML modal.
		// This avoids the Windows deadlock caused by OS dialogs inside OnBeforeClose.
		go func() {
			time.Sleep(50 * time.Millisecond)
			wailsruntime.EventsEmit(ctx, "show-close-dialog")
		}()
		return true // prevent OS close; frontend modal will call QuitApp or HideToTray
	}
}

// QuitApp exits the application cleanly. Called from the frontend close-dialog modal.
func (a *App) QuitApp() {
	a.exitNow()
}

func (a *App) exitNow() {
	a.exitOnce.Do(func() {
		a.forceQuit.Store(true)
		if a.ctx != nil {
			wailsruntime.Quit(a.ctx)
			// Hard fallback: if Wails doesn't shut down cleanly within 2 seconds,
			// force-exit to prevent a zombie process. Also clean up the systray
			// icon to avoid orphaned notification area icons on Windows.
			time.AfterFunc(2*time.Second, func() {
				log.Println("[exit] Wails did not shut down in time, forcing exit")
				systray.Quit()
				os.Exit(0)
			})
			return
		}
		a.shutdown(context.Background())
		os.Exit(0)
	})
}

// HideToTray hides the main window to the system tray. Called from the frontend close-dialog modal.
func (a *App) HideToTray() {
	a.forceQuit.Store(false)
	a.enqueueTrayAction(trayActionHide)
}

// ShowAboutDialog shows an about info dialog. Emits an event to the frontend
// so the display runs safely on the Wails JS thread (avoids tray-thread deadlock).
func (a *App) ShowAboutDialog() {
	a.enqueueTrayAction(trayActionAbout)
}

// SavePreferences updates preferences like window close behavior.
func (a *App) SavePreferences(closeBehavior string) string {
	prefs := preferences.Preferences{CloseBehavior: closeBehavior}
	if err := prefs.Validate(); err != nil {
		return "validation error: " + err.Error()
	}
	if err := prefs.Save(""); err != nil {
		return "save error: " + err.Error()
	}
	if err := removeLegacyCloseBehaviorFromConfig(); err != nil {
		log.Println("[GUI preferences] legacy close_behavior cleanup error:", err)
	}
	return "success"
}

// GetPreferences returns GUI-only preferences. These are intentionally kept
// outside the proxy config so CLI/server config stays portable.
func (a *App) GetPreferences() map[string]string {
	prefs, err := preferences.Load("")
	if err != nil {
		log.Println("[GUI preferences] load error:", err)
		prefs = preferences.Preferences{CloseBehavior: preferences.DefaultCloseBehavior}
	}
	return map[string]string{
		"close_behavior": prefs.CloseBehavior,
	}
}

func isMaskedAPIKey(key string) bool {
	return key == "****" || strings.Contains(key, "...")
}

func removeLegacyCloseBehaviorFromConfig() error {
	path, err := config.DefaultPath()
	if err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if _, ok := raw["close_behavior"]; !ok {
		return nil
	}
	delete(raw, "close_behavior")
	out, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(out, '\n'), 0o600)
}
