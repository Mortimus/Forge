package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	// Create a temporary config file
	content := `
jules_api_key: "test_key"
debug: true
repositories:
  - github_repo: "owner/repo"
    github_pat: "test_pat"
`
	tmpfile, err := os.CreateTemp("", "config_*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name()) // clean up

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(tmpfile.Name())
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}

	if cfg.JulesAPIKey != "test_key" {
		t.Errorf("JulesAPIKey = %v, want %v", cfg.JulesAPIKey, "test_key")
	}
	if !cfg.Debug {
		t.Error("Debug = false, want true")
	}
	if len(cfg.Repositories) != 1 {
		t.Fatalf("Repositories count = %d, want 1", len(cfg.Repositories))
	}
	if cfg.Repositories[0].GithubRepo != "owner/repo" {
		t.Errorf("Repo = %v, want owner/repo", cfg.Repositories[0].GithubRepo)
	}
	if cfg.Repositories[0].GithubPAT != "test_pat" {
		t.Errorf("PAT = %v, want test_pat", cfg.Repositories[0].GithubPAT)
	}
}

func TestLoad_MissingRequired(t *testing.T) {
	content := `
debug: true
repositories: []
`
	tmpfile, err := os.CreateTemp("", "config_invalid_*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()

	_, err = Load(tmpfile.Name())
	if err == nil {
		t.Error("Load() expected error for missing required, got nil")
	}
}

func TestCheckInterval(t *testing.T) {
	cfg := &Config{CheckIntervalSeconds: 30}
	expected := 30 * time.Second
	if got := cfg.CheckInterval(); got != expected {
		t.Errorf("CheckInterval() = %v, want %v", got, expected)
	}
}
