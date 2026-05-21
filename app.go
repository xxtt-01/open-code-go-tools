package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/ethan-blue/open-code-go-tools/internal/config"
	"github.com/ethan-blue/open-code-go-tools/internal/proxy"
)

// App struct
type App struct {
	ctx        context.Context
	srv        *proxy.Server
	cancelFunc context.CancelFunc
}

// NewApp creates a new App struct instance
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

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

// shutdown is called when the app closes
func (a *App) shutdown(ctx context.Context) {
	if a.cancelFunc != nil {
		log.Println("[GUI proxy] shutting down background proxy server...")
		a.cancelFunc()
	}
}

// GetListenAddress returns the actual proxy listen address dynamically
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

// SaveProfileConfig saves API key, default model, aliases, and timeout settings.
func (a *App) SaveProfileConfig(profileName, apiKey, defaultModel, sonnetAlias, haikuAlias, opusAlias, timeoutSeconds string) string {
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

	if apiKey != "" {
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

	return "success"
}

// InstallClaudeUserEnv persists Claude Code environment variables for new shells.
func (a *App) InstallClaudeUserEnv() string {
	listenAddr := a.GetListenAddress()
	activeProfile := "opencode-go"
	defaultModel := "kimi-k2.6"

	path, err := config.DefaultPath()
	if err == nil {
		cfg, err := config.Load(path)
		if err == nil {
			activeProfile = cfg.ActiveProfile
			if p, ok := cfg.Profiles[activeProfile]; ok && p.DefaultModel != "" {
				defaultModel = p.DefaultModel
			}
		}
	}

	env := map[string]string{
		"ANTHROPIC_BASE_URL":                         "http://" + listenAddr,
		"ANTHROPIC_API_KEY":                          "ocgt-local-proxy",
		"ANTHROPIC_CUSTOM_HEADERS":                   "X-Ocgt-Profile: " + activeProfile,
		"CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS":     "1",
		"CLAUDE_CODE_ENABLE_GATEWAY_MODEL_DISCOVERY": "1",
		"ANTHROPIC_MODEL":                            defaultModel,
		"OCGT_PROFILE":                               activeProfile,
	}

	if err := unsetUserEnvironment("ANTHROPIC_AUTH_TOKEN"); err != nil {
		return "unset ANTHROPIC_AUTH_TOKEN error: " + err.Error()
	}
	for name, value := range env {
		if err := setUserEnvironment(name, value); err != nil {
			return "set " + name + " error: " + err.Error()
		}
	}
	return "success"
}

// LaunchClaudeTerminal spawns a new terminal window preconfigured with the Claude Code proxy environment
func (a *App) LaunchClaudeTerminal(shell string) string {
	listenAddr := a.GetListenAddress()
	activeProfile := "opencode-go"
	defaultModel := "kimi-k2.6"

	// Try loading from config to get the latest
	path, err := config.DefaultPath()
	if err == nil {
		cfg, err := config.Load(path)
		if err == nil {
			activeProfile = cfg.ActiveProfile
			if p, ok := cfg.Profiles[activeProfile]; ok {
				if p.DefaultModel != "" {
					defaultModel = p.DefaultModel
				}
			}
		}
	}

	baseURL := "http://" + listenAddr

	switch runtime.GOOS {
	case "windows":
		if shell == "cmd" {
			// Launch CMD terminal
			cmd := exec.Command("cmd.exe", "/c", "start", "cmd.exe", "/k",
				fmt.Sprintf("set ANTHROPIC_BASE_URL=%s&& set ANTHROPIC_API_KEY=ocgt-local-proxy&& set ANTHROPIC_CUSTOM_HEADERS=X-Ocgt-Profile:%s&& set CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS=1&& set CLAUDE_CODE_ENABLE_GATEWAY_MODEL_DISCOVERY=1&& set ANTHROPIC_MODEL=%s&& echo =========================================================&& echo  [ocgt] Claude Code 代理终端已成功拉起！&& echo  当前代理: %s&& echo  当前模型: %s&& echo  请在下方直接输入: claude&& echo =========================================================&& echo.",
					baseURL, activeProfile, defaultModel, baseURL, defaultModel))
			if err := cmd.Run(); err != nil {
				return "launch cmd error: " + err.Error()
			}
		} else {
			// Launch PowerShell terminal
			psScript := fmt.Sprintf(
				"$env:ANTHROPIC_BASE_URL='%s'; $env:ANTHROPIC_API_KEY='ocgt-local-proxy'; $env:ANTHROPIC_CUSTOM_HEADERS='X-Ocgt-Profile: %s'; $env:CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS='1'; $env:CLAUDE_CODE_ENABLE_GATEWAY_MODEL_DISCOVERY='1'; $env:ANTHROPIC_MODEL='%s'; Clear-Host; Write-Host '=========================================================' -ForegroundColor Cyan; Write-Host ' [ocgt] Claude Code 代理终端已成功拉起！' -ForegroundColor Green; Write-Host ' 当前代理: %s' -ForegroundColor Gray; Write-Host ' 当前模型: %s' -ForegroundColor Gray; Write-Host ' 请在下方直接输入: claude' -ForegroundColor Green; Write-Host '=========================================================' -ForegroundColor Cyan; Write-Host ''",
				baseURL, activeProfile, defaultModel, baseURL, defaultModel)
			cmd := exec.Command("powershell.exe", "-NoExit", "-Command", psScript)
			if err := cmd.Start(); err != nil {
				return "launch powershell error: " + err.Error()
			}
		}
		return "success"
	case "darwin":
		// MacOS support (Terminal.app)
		script := fmt.Sprintf(
			"tell application \"Terminal\" to do script \"export ANTHROPIC_BASE_URL='%s' && export ANTHROPIC_API_KEY='ocgt-local-proxy' && export ANTHROPIC_CUSTOM_HEADERS='X-Ocgt-Profile: %s' && export CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS='1' && export CLAUDE_CODE_ENABLE_GATEWAY_MODEL_DISCOVERY='1' && export ANTHROPIC_MODEL='%s' && clear && echo '=========================================================' && echo ' [ocgt] Claude Code 代理终端已成功拉起！' && echo ' 当前代理: %s' && echo ' 当前模型: %s' && echo ' 请在下方直接输入: claude' && echo '=========================================================' && echo ''\"",
			baseURL, activeProfile, defaultModel, baseURL, defaultModel)
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
