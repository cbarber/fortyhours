// Package config loads and saves fortyhours' persisted settings: Productive
// credentials, the caller's person id, which absence events mean "sick" and
// "pto", and the autofill project/hours defaults discovered by `init`.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// DefaultDailyGoalMinutes is the expected number of minutes worked on a
// weekday (8 hours) used to warn when autofill doesn't add up.
const DefaultDailyGoalMinutes = 8 * 60

// AutofillProject is one line of autofill's default weekday split, e.g.
// "dreamfi: 7h" or "internal: 1h".
type AutofillProject struct {
	Project   string  `yaml:"project"`
	ServiceID string  `yaml:"service_id"`
	Hours     float64 `yaml:"hours"`
}

// Config is fortyhours' persisted configuration.
type Config struct {
	// APIToken and OrgID authenticate every Productive request. Prefer the
	// PRODUCTIVE_API_KEY / PRODUCTIVE_ORG_ID env vars; these fields exist so
	// `init` can persist them for convenience.
	APIToken string `yaml:"api_token,omitempty"`
	OrgID    string `yaml:"organization_id,omitempty"`

	// PersonID is the Productive person id to act as, resolved once during
	// `init` from PersonEmail.
	PersonID    string `yaml:"person_id"`
	PersonEmail string `yaml:"person_email"`

	// SickEvent and PTOEvent are the absence event category names (as
	// returned by `events list`) that `sick` and `pto` book time off
	// against.
	SickEvent string `yaml:"sick_event"`
	PTOEvent  string `yaml:"pto_event"`

	// DailyGoalMinutes is the expected minutes worked per weekday, used to
	// warn (not fail) when a day's entries don't add up.
	DailyGoalMinutes int `yaml:"daily_goal_minutes"`

	// Autofill is the default weekday project/hours split, e.g. dreamfi 7h
	// + internal 1h = 8h/day.
	Autofill []AutofillProject `yaml:"autofill"`
}

// EnvAPIToken, EnvOrgID, and EnvPersonID name the environment variables
// that override (and, when unset, fall back to) the config file's
// credentials and person id.
const (
	EnvAPIToken = "PRODUCTIVE_API_KEY"
	EnvOrgID    = "PRODUCTIVE_ORG_ID"
	EnvPersonID = "PRODUCTIVE_PERSON_ID"
)

// Path returns the config file location: $FORTYHOURS_CONFIG if set,
// otherwise <os.UserConfigDir()>/fortyhours/config.yaml.
func Path() (string, error) {
	if p := os.Getenv("FORTYHOURS_CONFIG"); p != "" {
		return p, nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("config: resolving user config dir: %w", err)
	}
	return filepath.Join(dir, "fortyhours", "config.yaml"), nil
}

// Load reads the config file (if any) and overlays PRODUCTIVE_API_KEY /
// PRODUCTIVE_ORG_ID / PRODUCTIVE_PERSON_ID from the environment. It is not
// an error for the config file to be missing; callers should check whether
// the result is usable (e.g. APIToken/OrgID set) rather than treating a
// missing file as fatal.
func Load() (*Config, error) {
	path, err := Path()
	if err != nil {
		return nil, err
	}

	cfg := &Config{DailyGoalMinutes: DefaultDailyGoalMinutes}
	if data, err := os.ReadFile(path); err == nil {
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("config: parsing %s: %w", path, err)
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("config: reading %s: %w", path, err)
	}

	if v := os.Getenv(EnvAPIToken); v != "" {
		cfg.APIToken = v
	}
	if v := os.Getenv(EnvOrgID); v != "" {
		cfg.OrgID = v
	}
	if v := os.Getenv(EnvPersonID); v != "" {
		cfg.PersonID = v
	}
	if cfg.DailyGoalMinutes == 0 {
		cfg.DailyGoalMinutes = DefaultDailyGoalMinutes
	}
	return cfg, nil
}

// Save writes cfg to its config file, creating the parent directory if
// needed. File permissions are restricted to the owner since APIToken may
// be persisted here.
func (c *Config) Save() error {
	path, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("config: creating config dir: %w", err)
	}
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("config: encoding config: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("config: writing %s: %w", path, err)
	}
	return nil
}
