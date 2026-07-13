package config

import (
	"path/filepath"
	"testing"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("FORTYHOURS_CONFIG", filepath.Join(dir, "config.yaml"))
	t.Setenv(EnvAPIToken, "")
	t.Setenv(EnvOrgID, "")

	cfg := &Config{
		PersonID:         "123",
		PersonEmail:      "me@example.com",
		SickEvent:        "Sick",
		PTOEvent:         "PTO",
		DailyGoalMinutes: 480,
		Autofill: []AutofillProject{
			{Project: "dreamfi", ServiceID: "111", Hours: 7},
			{Project: "internal", ServiceID: "222", Hours: 1},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.PersonID != cfg.PersonID || got.PersonEmail != cfg.PersonEmail {
		t.Errorf("person mismatch: got %+v", got)
	}
	if got.SickEvent != "Sick" || got.PTOEvent != "PTO" {
		t.Errorf("event mismatch: got %+v", got)
	}
	if len(got.Autofill) != 2 || got.Autofill[0].Project != "dreamfi" || got.Autofill[0].Hours != 7 {
		t.Errorf("autofill mismatch: got %+v", got.Autofill)
	}
}

func TestLoadMissingFileIsNotAnError(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("FORTYHOURS_CONFIG", filepath.Join(dir, "does-not-exist.yaml"))
	t.Setenv(EnvAPIToken, "")
	t.Setenv(EnvOrgID, "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.DailyGoalMinutes != DefaultDailyGoalMinutes {
		t.Errorf("DailyGoalMinutes = %d, want default %d", cfg.DailyGoalMinutes, DefaultDailyGoalMinutes)
	}
}

func TestEnvVarsOverrideConfigFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("FORTYHOURS_CONFIG", filepath.Join(dir, "config.yaml"))

	cfg := &Config{APIToken: "file-token", OrgID: "file-org"}
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	t.Setenv(EnvAPIToken, "env-token")
	t.Setenv(EnvOrgID, "env-org")

	got, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.APIToken != "env-token" {
		t.Errorf("APIToken = %q, want env-token to win over file", got.APIToken)
	}
	if got.OrgID != "env-org" {
		t.Errorf("OrgID = %q, want env-org to win over file", got.OrgID)
	}
}
