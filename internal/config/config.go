package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	DefaultListen                = "127.0.0.1:8787"
	DefaultUpstream              = "https://opencode.ai/zen/go"
	DefaultRequestTimeoutSeconds = 300
)

type Config struct {
	Listen                string             `json:"listen"`
	Upstream              string             `json:"upstream"`
	RequestTimeoutSeconds int                `json:"request_timeout_seconds,omitempty"`
	ActiveProfile         string             `json:"active_profile"`
	Profiles              map[string]Profile `json:"profiles"`
	LocalAuthToken        string             `json:"local_auth_token,omitempty"`        // Optional local auth token for proxy access
	MaxConcurrentRequests int                `json:"max_concurrent_requests,omitempty"` // Optional concurrent request limit
}

type Profile struct {
	APIKeyEnv     string            `json:"api_key_env"`
	APIKey        string            `json:"api_key,omitempty"`
	DefaultModel  string            `json:"default_model,omitempty"`
	ModelAliases  map[string]string `json:"model_aliases,omitempty"`
	MessageModels []string          `json:"message_models,omitempty"`
	Headers       map[string]string `json:"headers,omitempty"`
}

func DefaultPath() (string, error) {
	if p := strings.TrimSpace(os.Getenv("OCGT_CONFIG")); p != "" {
		return p, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".ocgt", "config.json"), nil
}

func Example() Config {
	return Config{
		Listen:                DefaultListen,
		Upstream:              DefaultUpstream,
		RequestTimeoutSeconds: DefaultRequestTimeoutSeconds,
		ActiveProfile:         "opencode-go",
		Profiles: map[string]Profile{
			"opencode-go": {
				APIKeyEnv:    "OPENCODE_GO_API_KEY",
				DefaultModel: "kimi-k2.6",
				ModelAliases: map[string]string{
					"deepseek": "deepseek-v4-pro",
					"flash":    "deepseek-v4-flash",
					"glm":      "glm-5.1",
					"glm5":     "glm-5",
					"hy3":      "hy3-preview",
					"kimi":     "kimi-k2.6",
					"kimi25":   "kimi-k2.5",
					"mimo":     "mimo-v2.5-pro",
					"mimo25":   "mimo-v2.5",
					"minimax":  "minimax-m2.7",
					"opus":     "kimi-k2.6",
					"qwen":     "qwen3.6-plus",
					"qwen35":   "qwen3.5-plus",
					"sonnet":   "qwen3.6-plus",
					"haiku":    "deepseek-v4-flash",
				},
				MessageModels: []string{"minimax-m2.5", "minimax-m2.7"},
			},
		},
	}
}

func Load(path string) (Config, error) {
	if strings.TrimSpace(path) == "" {
		var err error
		path, err = DefaultPath()
		if err != nil {
			return Config{}, err
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	cfg.applyDefaults()
	return cfg, cfg.Validate()
}

func WriteExample(path string, overwrite bool) (string, error) {
	if strings.TrimSpace(path) == "" {
		var err error
		path, err = DefaultPath()
		if err != nil {
			return "", err
		}
	}
	if !overwrite {
		if _, err := os.Stat(path); err == nil {
			return path, fmt.Errorf("config already exists: %s", path)
		} else if !errors.Is(err, os.ErrNotExist) {
			return path, err
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return path, err
	}
	data, err := json.MarshalIndent(Example(), "", "  ")
	if err != nil {
		return path, err
	}
	return path, os.WriteFile(path, append(data, '\n'), 0o600)
}

func (c Config) Save(path string) error {
	if strings.TrimSpace(path) == "" {
		var err error
		path, err = DefaultPath()
		if err != nil {
			return err
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o600)
}

func (c *Config) applyDefaults() {
	if c.Listen == "" {
		c.Listen = DefaultListen
	}
	if c.Upstream == "" {
		c.Upstream = DefaultUpstream
	}
	if c.RequestTimeoutSeconds == 0 {
		c.RequestTimeoutSeconds = DefaultRequestTimeoutSeconds
	}
	if c.Profiles == nil {
		c.Profiles = map[string]Profile{}
	}
	if c.ActiveProfile == "" && len(c.Profiles) == 1 {
		for name := range c.Profiles {
			c.ActiveProfile = name
		}
	}
}

func (c Config) Validate() error {
	if _, err := url.ParseRequestURI(c.Upstream); err != nil {
		return fmt.Errorf("invalid upstream %q: %w", c.Upstream, err)
	}
	if c.RequestTimeoutSeconds < 1 || c.RequestTimeoutSeconds > 3600 {
		return fmt.Errorf("request_timeout_seconds must be between 1 and 3600, got %d", c.RequestTimeoutSeconds)
	}
	if len(c.Profiles) == 0 {
		return errors.New("at least one profile is required")
	}
	if _, ok := c.Profiles[c.ActiveProfile]; !ok {
		return fmt.Errorf("active profile %q does not exist", c.ActiveProfile)
	}
	return nil
}

func (c Config) RequestTimeout() time.Duration {
	seconds := c.RequestTimeoutSeconds
	if seconds == 0 {
		seconds = DefaultRequestTimeoutSeconds
	}
	return time.Duration(seconds) * time.Second
}

// WarnIfNoAPIKey checks if the active profile has an API key and returns a warning message if not.
func (c Config) WarnIfNoAPIKey() string {
	profile, name, err := c.Profile("")
	if err != nil {
		return ""
	}
	if profile.APIKeyValue() == "" {
		return fmt.Sprintf("profile %q has no API key (set %s or configure api_key)", name, profile.APIKeyEnv)
	}
	return ""
}

func (c Config) Profile(name string) (Profile, string, error) {
	if strings.TrimSpace(name) == "" {
		name = c.ActiveProfile
	}
	p, ok := c.Profiles[name]
	if !ok {
		return Profile{}, name, fmt.Errorf("profile %q does not exist", name)
	}
	return p, name, nil
}

func (p Profile) APIKeyValue() string {
	if p.APIKey != "" {
		return p.APIKey
	}
	if p.APIKeyEnv != "" {
		return os.Getenv(p.APIKeyEnv)
	}
	return ""
}

func (p Profile) ResolveModel(model string) string {
	if model == "" {
		model = p.DefaultModel
	}
	model = strings.TrimSpace(model)
	if p.ModelAliases != nil {
		if mapped := p.ModelAliases[model]; mapped != "" {
			return mapped
		}
	}
	lower := strings.ToLower(model)
	switch {
	case strings.Contains(lower, "opus"):
		return p.resolveAliasOrDefault("opus")
	case strings.Contains(lower, "sonnet"):
		return p.resolveAliasOrDefault("sonnet")
	case strings.Contains(lower, "haiku"):
		return p.resolveAliasOrDefault("haiku")
	}
	return model
}

func (p Profile) resolveAliasOrDefault(alias string) string {
	if p.ModelAliases != nil {
		if mapped := p.ModelAliases[alias]; mapped != "" {
			return mapped
		}
	}
	return p.DefaultModel
}

func (p Profile) UsesMessagesEndpoint(model string) bool {
	model = p.ResolveModel(model)
	for _, candidate := range p.MessageModels {
		if strings.EqualFold(candidate, model) {
			return true
		}
	}
	return false
}
