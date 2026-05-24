package preferences

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const DefaultCloseBehavior = "prompt"

type Preferences struct {
	CloseBehavior string `json:"close_behavior"`
}

func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".ocgt", "preferences.json"), nil
}

func Load(path string) (Preferences, error) {
	if strings.TrimSpace(path) == "" {
		var err error
		path, err = DefaultPath()
		if err != nil {
			return Preferences{}, fmt.Errorf("failed to determine preferences path: %w", err)
		}
	}

	prefs := Preferences{CloseBehavior: DefaultCloseBehavior}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return prefs, nil
		}
		return Preferences{}, fmt.Errorf("failed to read preferences file %q: %w", path, err)
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return prefs, nil
	}
	if err := json.Unmarshal(data, &prefs); err != nil {
		return Preferences{}, fmt.Errorf("failed to parse preferences file %q: %w", path, err)
	}
	prefs.applyDefaults()
	return prefs, prefs.Validate()
}

func (p Preferences) Save(path string) error {
	if strings.TrimSpace(path) == "" {
		var err error
		path, err = DefaultPath()
		if err != nil {
			return err
		}
	}
	p.applyDefaults()
	if err := p.Validate(); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o600)
}

func (p *Preferences) applyDefaults() {
	if p.CloseBehavior == "" {
		p.CloseBehavior = DefaultCloseBehavior
	}
}

func (p Preferences) Validate() error {
	if !IsValidCloseBehavior(p.CloseBehavior) {
		return fmt.Errorf("invalid close_behavior %q, must be 'prompt', 'minimize', or 'exit'", p.CloseBehavior)
	}
	return nil
}

func IsValidCloseBehavior(value string) bool {
	switch value {
	case "prompt", "minimize", "exit":
		return true
	default:
		return false
	}
}
