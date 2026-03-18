package orchestrator

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	ghlib "github.com/google/go-github/v69/github"
	"github.com/mortimus/forge/internal/clients/jules"
	"github.com/mortimus/forge/internal/config"
	"github.com/mortimus/forge/internal/mocks"
	"github.com/mortimus/forge/internal/persistence"
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

func TestGlobalRateLimit(t *testing.T) {
	cfg := createTestConfig()
	cfg.MaxSessionsPerDay = 1 // Limit to 1
	jMock := &mocks.JulesMock{}
	s := stats.New()

	// Mock Persistence
	pm, _ := createTempPersistence(t)

	o := New(cfg, jMock, s, pm)

	// 1. First check -> OK
	if err := o.checkRateLimit(); err != nil {
		t.Errorf("First check failed: %v", err)
	}

	// 2. Record one session
	o.recordSessionStart()

	// 3. Second check -> Should Fail
	if err := o.checkRateLimit(); err == nil {
		t.Error("Second check succeeded, expected rate limit error")
	}
}
func TestHandleCompletion_Cleanup(t *testing.T) {
	tests := []struct {
		name               string
		autoDeleteSessions bool
		wantDeleted        bool
	}{
		{
			name:               "AutoDelete Disabled (Default)",
			autoDeleteSessions: false,
			wantDeleted:        false,
		},
		{
			name:               "AutoDelete Enabled",
			autoDeleteSessions: true,
			wantDeleted:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := createTestConfig()
			cfg.AutoDeleteSessions = tt.autoDeleteSessions
			jMock := &mocks.JulesMock{}
			ghMock := &mocks.GithubMock{}
			stats := stats.New()
			o := New(cfg, jMock, stats, nil)

			rc := &RepoContext{
				Config: cfg.Repositories[0],
				GH:     ghMock,
			}

			sessionDeleted := false
			jMock.DeleteSessionFunc = func(ctx context.Context, sessionName string) error {
				if sessionName == "session1" {
					sessionDeleted = true
				}
				return nil
			}

			jMock.GetSessionFunc = func(ctx context.Context, sessionName string) (*jules.Session, error) {
				return &jules.Session{Name: "session1", State: "SUCCEEDED"}, nil
			}

			sess := &ActiveSession{ID: "session1", State: "SUCCEEDED"}
			o.handleCompletion(context.Background(), rc, sess)

			if sessionDeleted != tt.wantDeleted {
				t.Errorf("sessionDeleted = %v, want %v", sessionDeleted, tt.wantDeleted)
			}
		})
	}
}

func createTempPersistence(t *testing.T) (*persistence.Manager, string) {
	tmpFile, err := os.CreateTemp("", "state.json")
	if err != nil {
		t.Fatal(err)
	}
	tmpFile.Close() // Close so manager can read/write

	// Initialize empty state
	pm := persistence.NewManager(tmpFile.Name())
	state := &persistence.State{}
	pm.Save(state)
	
	// Cleanup
	t.Cleanup(func() {
		os.Remove(tmpFile.Name())
	})

	return pm, tmpFile.Name()
}

func TestHandleAutomatedInteractions_AwaitingUserFeedback(t *testing.T) {
	cfg := createTestConfig()
	cfg.Repositories[0].ProceedTemplatePath = "PROCEED.md"
	jMock := &mocks.JulesMock{}
	ghMock := &mocks.GithubMock{}
	s := stats.New()
	o := New(cfg, jMock, s, nil)

	rc := &RepoContext{
		Config: cfg.Repositories[0],
		GH:     ghMock,
	}

	ghMock.GetFileContentFunc = func(ctx context.Context, path string) (string, error) {
		if path == "PROCEED.md" {
			return "Proceed with the test", nil
		}
		return "", errors.New("not found")
	}

	sentMsg := ""
	jMock.SendMessageFunc = func(ctx context.Context, sessionID, message string) error {
		sentMsg = message
		return nil
	}

	sess := &ActiveSession{
		ID:    "session1",
		State: "AWAITING_USER_FEEDBACK",
	}

	o.handleAutomatedInteractions(context.Background(), rc, sess)

	if sentMsg != "Proceed with the test" {
		t.Errorf("Expected sent message to be 'Proceed with the test', got '%s'", sentMsg)
	}
	if sess.LastAutomatedState != "AWAITING_USER_FEEDBACK" {
		t.Errorf("Expected LastAutomatedState to be 'AWAITING_USER_FEEDBACK', got '%s'", sess.LastAutomatedState)
	}
}
