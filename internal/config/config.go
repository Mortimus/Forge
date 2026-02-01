package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// RepositoryConfig holds configuration for a specific repository.
type RepositoryConfig struct {
	GithubRepo string `yaml:"github_repo"`
	GithubPAT  string `yaml:"github_pat"`
	AutoMerge  bool   `yaml:"auto_merge"`

	// In-Repo Paths
	AgentsPromptPath string `yaml:"agents_prompt_path"`
	ImplPlanPath     string `yaml:"impl_plan_path"`
	SpecPath         string `yaml:"spec_path"`
	SystemPromptPath string `yaml:"system_prompt_path"`

	// Local Template Overrides
	GapAnalysisTemplatePath string `yaml:"gap_analysis_template_path"`
	ResolutionTemplatePath  string `yaml:"resolution_template_path"`
}

// Config holds all configuration parameters for the application.
type Config struct {
	JulesAPIKey          string             `yaml:"jules_api_key"`
	MaxSessionsPerDay    int                `yaml:"max_sessions_per_day"`
	Debug                bool               `yaml:"debug"`
	CheckIntervalSeconds int                `yaml:"check_interval_seconds"`
	StateFilePath        string             `yaml:"state_file_path"`
	Repositories         []RepositoryConfig `yaml:"repositories"`
}

func (c *Config) SetDefaults() {
	if c.MaxSessionsPerDay == 0 {
		c.MaxSessionsPerDay = 100
	}
	if c.CheckIntervalSeconds == 0 {
		c.CheckIntervalSeconds = 20
	}
    if c.StateFilePath == "" {
        c.StateFilePath = "forge_state.json"
    }
	for i := range c.Repositories {
		if c.Repositories[i].AgentsPromptPath == "" {
			c.Repositories[i].AgentsPromptPath = "AGENTS.md"
		}
		if c.Repositories[i].ImplPlanPath == "" {
			c.Repositories[i].ImplPlanPath = "IMPLEMENTATION_PLAN.md"
		}
		if c.Repositories[i].SpecPath == "" {
			c.Repositories[i].SpecPath = "docs/spec"
		}
		if c.Repositories[i].SystemPromptPath == "" {
			c.Repositories[i].SystemPromptPath = "SYSTEM_PROMPT.md"
		}
	}
}

// Load reads configuration from the specified YAML file.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	cfg.SetDefaults()
	
	// Validation
	if cfg.JulesAPIKey == "" {
		return nil, fmt.Errorf("jules_api_key is required")
	}
	if len(cfg.Repositories) == 0 {
		return nil, fmt.Errorf("at least one repository must be configured")
	}
	for _, repo := range cfg.Repositories {
		if repo.GithubRepo == "" {
			return nil, fmt.Errorf("github_repo is required for all repositories")
		}
		if repo.GithubPAT == "" {
			return nil, fmt.Errorf("github_pat is required for %s", repo.GithubRepo) // Be careful logging PATs? No, just saying it's required for the repo.
		}
	}

	return &cfg, nil
}

// CheckInterval returns the CheckIntervalSeconds as a time.Duration.
func (c *Config) CheckInterval() time.Duration {
	return time.Duration(c.CheckIntervalSeconds) * time.Second
}
