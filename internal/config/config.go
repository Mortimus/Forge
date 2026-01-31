// Package config handles the loading and validation of application configuration.
package config

import (
	"time"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

// Config holds all configuration parameters for the application.
// Values are loaded from environment variables.
type Config struct {
	// Jules Configuration

	// JulesAPIKey is the API Key for Google Jules.
	JulesAPIKey         string `envconfig:"JULES_API_KEY" required:"true"`
	// MaxJulesSessions is the maximum number of concurrent sessions (default 1).
	MaxJulesSessions    int    `envconfig:"MAX_JULES_SESSIONS" default:"1"`
	// MaxSessionsPerDay is the safety limit for daily sessions (default 100).
	MaxSessionsPerDay   int    `envconfig:"MAX_SESSIONS_PER_DAY" default:"100"`

	// AutoMerge controls if PRs are merged automatically (default true).
	AutoMerge           bool   `envconfig:"AUTO_MERGE" default:"true"`

	// GitHub Configuration

	// GithubPAT is the Personal Access Token for GitHub API access.
	GithubPAT               string `envconfig:"GITHUB_PAT" required:"true"`
	// GithubRepo is the target repository in "owner/repo" format.
	GithubRepo              string `envconfig:"GITHUB_REPO" required:"true"`

	// File Path Configuration (In-Repo)

	// AgentsPromptPath is the path to the context memory file in the repo.
	AgentsPromptPath string `envconfig:"AGENTS_PROMPT_PATH" default:"AGENTS.md"`
	// ImplPlanPath is the path to the implementation plan file in the repo.
	ImplPlanPath     string `envconfig:"IMPL_PLAN_PATH" default:"IMPLEMENTATION_PLAN.md"`
	// SpecPath is the directory path for specification files in the repo.
	SpecPath         string `envconfig:"SPEC_PATH" default:"docs/spec"`
	// SystemPromptPath is the path to the system prompt file in the repo.
	SystemPromptPath string `envconfig:"SYSTEM_PROMPT_PATH" default:"SYSTEM_PROMPT.md"`

	// Prompt Template Overrides (Local Filesystem)

	// GapAnalysisTemplatePath is an optional path to a local file overriding the embedded gap analysis prompt.
	GapAnalysisTemplatePath string `envconfig:"GAP_ANALYSIS_TEMPLATE_PATH"`
	// ResolutionTemplatePath is an optional path to a local file overriding the embedded resolution prompt.
	ResolutionTemplatePath  string `envconfig:"RESOLUTION_TEMPLATE_PATH"`

	// Application Configuration

	// CheckIntervalSeconds is the frequency in seconds to check for new work.
	CheckIntervalSeconds int `envconfig:"CHECK_INTERVAL_SECONDS" default:"60"`
}

// Load reads configuration from .env files and environment variables.
// It returns an error if required fields are missing.
func Load() (*Config, error) {
	// Load .env file if it exists, ignore error if it doesn't (production might use real env vars)
	_ = godotenv.Load()

	var cfg Config
	err := envconfig.Process("", &cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}

// CheckInterval returns the CheckIntervalSeconds as a time.Duration.
func (c *Config) CheckInterval() time.Duration {
	return time.Duration(c.CheckIntervalSeconds) * time.Second
}
