package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"sort"
	"strings"
	"time"

	"github.com/ethan-blue/open-code-go-tools/internal/config"
	"github.com/ethan-blue/open-code-go-tools/internal/proxy"
)

var version = "0.1.8"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		usage()
		return nil
	}
	switch args[0] {
	case "init":
		return cmdInit(args[1:])
	case "serve":
		return cmdServe(args[1:])
	case "profiles":
		return cmdProfiles(args[1:])
	case "models":
		return cmdModels(args[1:])
	case "claude-env":
		return cmdClaudeEnv(args[1:])
	case "ccswitch":
		return cmdCCSwitch(args[1:])
	case "key":
		return cmdKey(args[1:])
	case "version":
		fmt.Println(version)
		return nil
	case "help", "-h", "--help":
		usage()
		return nil
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func cmdInit(args []string) error {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	path := fs.String("config", "", "config path")
	force := fs.Bool("force", false, "overwrite existing config")
	if err := fs.Parse(args); err != nil {
		return err
	}
	written, err := config.WriteExample(*path, *force)
	if err != nil {
		return err
	}
	fmt.Println(written)
	return nil
}

func cmdServe(args []string) error {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	configPath := fs.String("config", "", "config path")
	profileName := fs.String("profile", "", "profile name")
	listen := fs.String("listen", "", "listen address")
	if err := fs.Parse(args); err != nil {
		return err
	}
	cfg, err := loadConfig(*configPath)
	if err != nil {
		return err
	}
	if *profileName != "" {
		cfg.ActiveProfile = *profileName
	} else if envProfile := strings.TrimSpace(os.Getenv("OCGT_PROFILE")); envProfile != "" {
		cfg.ActiveProfile = envProfile
	}
	if *listen != "" {
		cfg.Listen = *listen
	}
	if err := cfg.Validate(); err != nil {
		return err
	}
	srv, err := proxy.New(cfg)
	if err != nil {
		return err
	}
	resolvedPath := *configPath
	if strings.TrimSpace(resolvedPath) == "" {
		resolvedPath, _ = config.DefaultPath()
	}
	srv.SetConfigPath(resolvedPath)

	if warn := cfg.WarnIfNoAPIKey(); warn != "" {
		fmt.Fprintln(os.Stderr, "warning:", warn)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	return srv.ListenAndServe(ctx)
}

func cmdProfiles(args []string) error {
	fs := flag.NewFlagSet("profiles", flag.ExitOnError)
	configPath := fs.String("config", "", "config path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	cfg, err := loadConfig(*configPath)
	if err != nil {
		return err
	}
	names := make([]string, 0, len(cfg.Profiles))
	for name := range cfg.Profiles {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		marker := " "
		if name == cfg.ActiveProfile {
			marker = "*"
		}
		p := cfg.Profiles[name]
		fmt.Printf("%s %s default_model=%s api_key_env=%s\n", marker, name, p.DefaultModel, p.APIKeyEnv)
	}
	return nil
}

func cmdModels(args []string) error {
	fs := flag.NewFlagSet("models", flag.ExitOnError)
	configPath := fs.String("config", "", "config path")
	profileName := fs.String("profile", "", "profile name")
	remote := fs.Bool("remote", false, "query upstream /v1/models")
	if err := fs.Parse(args); err != nil {
		return err
	}
	cfg, profile, _, err := selectedProfile(*configPath, *profileName)
	if err != nil {
		return err
	}
	if *remote {
		data, err := fetchRemoteModels(context.Background(), cfg.Upstream, profile)
		if err != nil {
			if len(data) > 0 {
				fmt.Print(string(data))
			}
			return err
		}
		fmt.Print(string(data))
		return nil
	}
	printAliases(profile)
	return nil
}

func cmdClaudeEnv(args []string) error {
	fs := flag.NewFlagSet("claude-env", flag.ExitOnError)
	configPath := fs.String("config", "", "config path")
	profileName := fs.String("profile", "", "profile name")
	baseURL := fs.String("base-url", "http://127.0.0.1:8787", "local proxy base URL")
	shell := fs.String("shell", "powershell", "powershell, bash, or cmd")
	if err := fs.Parse(args); err != nil {
		return err
	}
	_, profile, name, err := selectedProfile(*configPath, *profileName)
	if err != nil {
		return err
	}
	env := map[string]string{
		"ANTHROPIC_BASE_URL":       *baseURL,
		"ANTHROPIC_API_KEY":        "ocgt-local-proxy",
		"ANTHROPIC_CUSTOM_HEADERS": "X-Ocgt-Profile: " + name,
		"OCGT_PROFILE":             name,
	}
	if profile.DefaultModel != "" {
		env["ANTHROPIC_MODEL"] = profile.DefaultModel
	}
	printEnv(env, *shell)
	return nil
}

func cmdCCSwitch(args []string) error {
	fs := flag.NewFlagSet("ccswitch", flag.ExitOnError)
	configPath := fs.String("config", "", "config path")
	profileName := fs.String("profile", "", "profile name")
	baseURL := fs.String("base-url", "http://127.0.0.1:8787", "local proxy base URL")
	if err := fs.Parse(args); err != nil {
		return err
	}
	_, profile, name, err := selectedProfile(*configPath, *profileName)
	if err != nil {
		return err
	}
	payload := map[string]any{
		"name":    "ocgt-" + name,
		"type":    "anthropic",
		"baseURL": *baseURL,
		"apiKey":  "ocgt-local-proxy",
		"model":   profile.DefaultModel,
		"headers": map[string]string{
			"X-Ocgt-Profile": name,
		},
	}
	out, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}

func cmdKey(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: ocgt key set <opencode-go-key> | ocgt key show")
	}
	switch args[0] {
	case "set":
		if len(args) < 2 || strings.TrimSpace(args[1]) == "" {
			return errors.New("usage: ocgt key set <opencode-go-key>")
		}
		return setUserEnv("OPENCODE_GO_API_KEY", strings.TrimSpace(args[1]))
	case "show":
		key := os.Getenv("OPENCODE_GO_API_KEY")
		if key == "" {
			fmt.Println("OPENCODE_GO_API_KEY is not set in this process")
			return nil
		}
		fmt.Println(maskKey(key))
		return nil
	default:
		return fmt.Errorf("unknown key command %q", args[0])
	}
}

func selectedProfile(configPath, profileName string) (config.Config, config.Profile, string, error) {
	cfg, err := loadConfig(configPath)
	if err != nil {
		return config.Config{}, config.Profile{}, "", err
	}
	profile, name, err := cfg.Profile(profileName)
	if err != nil {
		return config.Config{}, config.Profile{}, "", err
	}
	return cfg, profile, name, nil
}

func loadConfig(path string) (config.Config, error) {
	cfg, err := config.Load(path)
	if err == nil {
		return cfg, nil
	}
	defaultPath, pathErr := config.DefaultPath()
	if pathErr == nil && (path == "" || path == defaultPath) && os.IsNotExist(err) {
		return config.Config{}, fmt.Errorf("config not found; run `ocgt init` first: %s", defaultPath)
	}
	return config.Config{}, err
}

func printAliases(profile config.Profile) {
	if profile.DefaultModel != "" {
		fmt.Printf("* default -> %s\n", profile.ResolveModel(profile.DefaultModel))
	}
	keys := make([]string, 0, len(profile.ModelAliases))
	for k := range profile.ModelAliases {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		route := "chat/completions"
		if profile.UsesMessagesEndpoint(k) {
			route = "messages"
		}
		fmt.Printf("  %s -> %s [%s]\n", k, profile.ModelAliases[k], route)
	}
}

func setUserEnv(name, value string) error {
	if err := os.Setenv(name, value); err != nil {
		return err
	}
	if isWindows() {
		if err := setWindowsUserEnv(name, value); err != nil {
			return err
		}
		fmt.Printf("%s saved to Windows user environment. Restart PowerShell or run `$env:%s = [Environment]::GetEnvironmentVariable(%q, 'User')` for the current window.\n", name, name, name)
		return nil
	}
	fmt.Printf("%s set for this process. Add it to your shell profile to persist it.\n", name)
	return nil
}

func isWindows() bool {
	return os.PathSeparator == '\\'
}

func setWindowsUserEnv(name, value string) error {
	return runPowerShellNoProfile("[Environment]::SetEnvironmentVariable($args[0], $args[1], 'User')", name, value)
}

func runPowerShellNoProfile(script string, args ...string) error {
	cmdArgs := append([]string{"-NoProfile", "-NonInteractive", "-Command", "& { " + script + " }", "--"}, args...)
	cmd := exec.Command("powershell", cmdArgs...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("powershell failed: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func maskKey(key string) string {
	if len(key) <= 4 {
		return strings.Repeat("*", len(key))
	}
	return "****" + key[len(key)-4:] + fmt.Sprintf(" (%d chars)", len(key))
}

func fetchRemoteModels(ctx context.Context, upstream string, profile config.Profile) ([]byte, error) {
	target := strings.TrimRight(upstream, "/") + "/v1/models"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, err
	}
	if key := profile.APIKeyValue(); key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	// Limit response body to 1MB
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return data, fmt.Errorf("OpenCode Go returned %s", resp.Status)
	}
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return data, nil
	}
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return data, nil
	}
	return append(out, '\n'), nil
}

func printEnv(env map[string]string, shell string) {
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v := env[k]
		switch strings.ToLower(shell) {
		case "bash", "sh", "zsh":
			fmt.Printf("export %s=%q\n", k, v)
		case "cmd":
			fmt.Printf("set %s=%s\n", k, v)
		default:
			fmt.Printf("$env:%s = %q\n", k, v)
		}
	}
}

type multiFlag []string

func (m *multiFlag) String() string {
	return strings.Join(*m, ",")
}

func (m *multiFlag) Set(v string) error {
	*m = append(*m, v)
	return nil
}

func usage() {
	fmt.Print(`ocgt - official Claude API proxy for Claude Code and CC Switch

Commands:
  init          create ~/.ocgt/config.json
  serve         run the local proxy
  profiles      list configured profiles
  models        show local aliases or query official /v1/models with --remote
  claude-env    print environment variables for Claude Code
  ccswitch      print a CC Switch-friendly provider JSON snippet
  key           save or show OPENCODE_GO_API_KEY
  version       print version

Typical flow:
  ocgt init
  ocgt serve
  ocgt claude-env
`)
}
