package orchestrator

import (
	"context"
	"errors"
	"testing"
	"time"

	ghlib "github.com/google/go-github/v69/github"
	"github.com/mortimus/forge/internal/clients/jules"
	"github.com/mortimus/forge/internal/config"
	"github.com/mortimus/forge/internal/mocks"
	"github.com/mortimus/forge/internal/stats"
)

func createTestConfig() *config.Config {
	return &config.Config{
		Repositories: []config.RepositoryConfig{
			{
				GithubRepo:       "owner/repo",
				GithubPAT:        "pat",
				AgentsPromptPath: "AGENTS.md",
				SystemPromptPath: "SYSTEM.md",
				ImplPlanPath:     "PLAN.md",
				SpecPath:         "specs",
				AutoMerge:        true,
			},
		},
		CheckIntervalSeconds: 1,
		MaxSessionsPerDay:    100,
	}
}

func TestResolveSourceName(t *testing.T) {
	cfg := createTestConfig()
	jMock := &mocks.JulesMock{}
	// ghMock not needed for verify, logic is in resolveSourceName
	stats := stats.New()

	o := New(cfg, jMock, stats, nil)

	// Case 1: Source Found
	jMock.ListSourcesFunc = func(ctx context.Context) ([]jules.Source, error) {
		var s jules.Source
		s.Name = "sources/1"
		s.GithubRepo.Owner = "owner"
		s.GithubRepo.Repo = "repo"
		return []jules.Source{s}, nil
	}

	name, err := o.resolveSourceName(context.Background(), "owner/repo")
	if err != nil {
		t.Errorf("resolveSourceName failed: %v", err)
	}
	if name != "sources/1" {
		t.Errorf("sourceName = %s, want sources/1", name)
	}
}

func TestProcessRepo_FindNewWork(t *testing.T) {
	cfg := createTestConfig()
	jMock := &mocks.JulesMock{}
	ghMock := &mocks.GithubMock{}
	s := stats.New()
	o := New(cfg, jMock, s, nil)

	rc := &RepoContext{
		Config:     cfg.Repositories[0],
		GH:         ghMock,
		SourceName: "source/1",
		LastReset:  time.Now(),
	}

	// Setup: No open PRs, specs exist, no plan -> Trigger Gap Analysis
	ghMock.ListOpenPullRequestsFunc = func(ctx context.Context) ([]*ghlib.PullRequest, error) {
		return []*ghlib.PullRequest{}, nil
	}
	ghMock.ListFilesFunc = func(ctx context.Context, path string) ([]string, error) {
		return []string{"spec.md"}, nil // Found spec
	}
	ghMock.GetFileContentFunc = func(ctx context.Context, path string) (string, error) {
		if path == "specs/spec.md" {
			return "spec", nil
		}
		if path == "AGENTS.md" {
			return "agents", nil
		}
		if path == "SYSTEM.md" {
			return "system", nil
		}
		return "", errors.New("not found") // No plan
	}

	sessionCreated := false
	jMock.CreateSessionFunc = func(ctx context.Context, title, prompt, sourceName, branch string) (*jules.Session, error) {
		sessionCreated = true
		return &jules.Session{Name: "session1"}, nil
	}

	o.processRepo(context.Background(), rc)

	if !sessionCreated {
		t.Error("Gap Analysis session expected but not created")
	}
	if len(o.activeSessions) != 1 {
		t.Error("Session not tracked")
	}
}

func TestProcessRepo_BlockingPR(t *testing.T) {
	cfg := createTestConfig()
	jMock := &mocks.JulesMock{}
	ghMock := &mocks.GithubMock{}
	s := stats.New()
	o := New(cfg, jMock, s, nil)

	rc := &RepoContext{
		Config: cfg.Repositories[0],
		GH:     ghMock,
	}

	ghMock.ListOpenPullRequestsFunc = func(ctx context.Context) ([]*ghlib.PullRequest, error) {
		n := 123
		return []*ghlib.PullRequest{{Number: &n}}, nil
	}

	o.processRepo(context.Background(), rc)

	// Should NOT create session
	if len(o.activeSessions) != 0 {
		t.Error("blocked by PR but started session")
	}
}
