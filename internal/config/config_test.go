package config

import "testing"

func TestExampleIsValid(t *testing.T) {
	cfg := Example()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("example config should be valid: %v", err)
	}
}

func TestResolveModel(t *testing.T) {
	profile := Profile{
		DefaultModel: "sonnet",
		ModelAliases: map[string]string{
			"sonnet": "claude-sonnet-4-6",
		},
	}
	if got := profile.ResolveModel(""); got != "claude-sonnet-4-6" {
		t.Fatalf("empty model should resolve through default alias, got %q", got)
	}
	if got := profile.ResolveModel("glm-5.1"); got != "glm-5.1" {
		t.Fatalf("full model names should pass through, got %q", got)
	}
}

func TestExampleIncludesAllDocumentedGoAliases(t *testing.T) {
	profile := Example().Profiles["opencode-go"]
	for alias, want := range map[string]string{
		"glm5":   "glm-5",
		"hy3":    "hy3-preview",
		"kimi25": "kimi-k2.5",
		"mimo25": "mimo-v2.5",
		"qwen35": "qwen3.5-plus",
	} {
		if got := profile.ResolveModel(alias); got != want {
			t.Fatalf("%s resolved to %q, want %q", alias, got, want)
		}
	}
}

func TestResolveClaudeCompatModel(t *testing.T) {
	profile := Profile{
		DefaultModel: "kimi-k2.6",
		ModelAliases: map[string]string{
			"opus":   "kimi-k2.6",
			"sonnet": "qwen3.6-plus",
			"haiku":  "deepseek-v4-flash",
		},
	}
	if got := profile.ResolveModel("claude-opus-4-7-r5"); got != "kimi-k2.6" {
		t.Fatalf("opus compat model resolved to %q", got)
	}
	if got := profile.ResolveModel("claude-sonnet-4-6"); got != "qwen3.6-plus" {
		t.Fatalf("sonnet compat model resolved to %q", got)
	}
	if got := profile.ResolveModel("claude-haiku-4-5"); got != "deepseek-v4-flash" {
		t.Fatalf("haiku compat model resolved to %q", got)
	}
}

func TestWarnIfNoAPIKey(t *testing.T) {
	cfg := Config{
		ActiveProfile: "test",
		Profiles: map[string]Profile{
			"test": {APIKeyEnv: "MISSING_KEY", DefaultModel: "kimi-k2.6"},
		},
	}
	warn := cfg.WarnIfNoAPIKey()
	if warn == "" {
		t.Fatal("expected warning when API key is missing")
	}
}

func TestWarnIfNoAPIKeyPresent(t *testing.T) {
	cfg := Config{
		ActiveProfile: "test",
		Profiles: map[string]Profile{
			"test": {APIKey: "sk-test-key", DefaultModel: "kimi-k2.6"},
		},
	}
	warn := cfg.WarnIfNoAPIKey()
	if warn != "" {
		t.Fatalf("expected no warning when API key is present, got %q", warn)
	}
}

func TestValidateEmptyProfiles(t *testing.T) {
	cfg := Config{Listen: "127.0.0.1:8787", Upstream: "https://opencode.ai/zen/go"}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for empty profiles")
	}
}

func TestValidateInvalidUpstream(t *testing.T) {
	cfg := Config{
		Listen:        "127.0.0.1:8787",
		Upstream:      "not-a-url",
		ActiveProfile: "test",
		Profiles:      map[string]Profile{"test": {DefaultModel: "kimi-k2.6"}},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for invalid upstream")
	}
}

func TestValidateMissingActiveProfile(t *testing.T) {
	cfg := Config{
		Listen:        "127.0.0.1:8787",
		Upstream:      "https://opencode.ai/zen/go",
		ActiveProfile: "missing",
		Profiles:      map[string]Profile{"test": {DefaultModel: "kimi-k2.6"}},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for missing active profile")
	}
}

func TestProfileFromEmptyName(t *testing.T) {
	cfg := Config{
		ActiveProfile: "default",
		Profiles: map[string]Profile{
			"default": {DefaultModel: "kimi-k2.6"},
		},
	}
	p, name, err := cfg.Profile("")
	if err != nil {
		t.Fatal(err)
	}
	if name != "default" {
		t.Fatalf("expected default profile, got %q", name)
	}
	if p.DefaultModel != "kimi-k2.6" {
		t.Fatalf("expected kimi-k2.6, got %q", p.DefaultModel)
	}
}

func TestProfileFromInvalidName(t *testing.T) {
	cfg := Config{
		ActiveProfile: "default",
		Profiles: map[string]Profile{
			"default": {DefaultModel: "kimi-k2.6"},
		},
	}
	_, _, err := cfg.Profile("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent profile")
	}
}

func TestAPIKeyValue(t *testing.T) {
	p := Profile{APIKey: "direct-key", APIKeyEnv: "SOME_ENV_VAR"}
	if got := p.APIKeyValue(); got != "direct-key" {
		t.Fatalf("expected direct-key, got %q", got)
	}
	p = Profile{APIKey: "", APIKeyEnv: ""}
	if got := p.APIKeyValue(); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}
