package orchestrator

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mortimus/forge/internal/clients/jules"
	"github.com/mortimus/forge/internal/mocks"
	"github.com/mortimus/forge/internal/persistence"
	"github.com/mortimus/forge/internal/stats"
)

func TestOrchestrator_PersistenceIntegration(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "forge_test_integ")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	statePath := filepath.Join(tmpDir, "state.json")
	
	cfg := createTestConfig()
	// Set CheckInterval to expected value
	cfg.CheckIntervalSeconds = 1
	
	repoName := cfg.Repositories[0].GithubRepo

	// --- Phase 1: Run Orchestrator, Create Session, Save State ---
	pm1 := persistence.NewManager(statePath)
	stats1 := stats.New()
	jMock1 := &mocks.JulesMock{}
	// ghMock1 not needed since run() isn't called, only internal state setup

	// Mocking behavior for Phase 1
	jMock1.ListSourcesFunc = func(ctx context.Context) ([]jules.Source, error) {
		return []jules.Source{{
			Name: "source/1",
			GithubRepo: struct {
				Owner string `json:"owner"`
				Repo  string `json:"repo"`
			}{Owner: "owner", Repo: "repo"},
		}}, nil
	}
	// Simulate existing work to trigger session creation ... actually let's just manually inject a session 
	// because Run() is hard to stop deterministically without complex signaling in test.
	// But we want to test loadState/saveState logic which is private.
	// So we will instantiate Orchestrator, injecting dependencies, and call public methods if possible 
	// or rely on internal state modification if we are in the same package (which we are).

	orch1 := New(cfg, jMock1, stats1, pm1)
	
	// Manually inject a session into orch1 to simulate active work
	orch1.activeSessions["session-xyz"] = &ActiveSession{
		ID:        "session-xyz",
		Repo:      repoName,
		Type:      TypeGapAnalysis,
		StartTime: time.Now(),
		State:     "RUNNING",
	}
	// Manually set a repo state
    // We need to initialize activeRepos first.
	orch1.activeRepos = make(map[string]*RepoContext)
	orch1.activeRepos[repoName] = &RepoContext{
		Config:     cfg.Repositories[0],
		DailyCount: 42,
		LastReset:  time.Now(),
	}

	// Trigger Save (using the private method since we are in same package)
	orch1.saveState()

	// Verify file exists
	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		t.Fatalf("State file was not created at %s", statePath)
	}

	// --- Phase 2: Create NEW Orchestrator, Load State ---
	pm2 := persistence.NewManager(statePath)
	stats2 := stats.New()
	jMock2 := &mocks.JulesMock{}
	
	jMock2.ListSourcesFunc = func(ctx context.Context) ([]jules.Source, error) {
		return []jules.Source{{
			Name: "source/1",
			GithubRepo: struct {
				Owner string `json:"owner"`
				Repo  string `json:"repo"`
			}{Owner: "owner", Repo: "repo"},
		}}, nil
	}
	
	orch2 := New(cfg, jMock2, stats2, pm2)
	
	// We need to initialize activeRepos in orch2 so loadState has somewhere to put data
	// resolveSourceName needs to work
	orch2.activeRepos = make(map[string]*RepoContext)
	orch2.activeRepos[repoName] = &RepoContext{
		Config: cfg.Repositories[0],
        // DailyCount should be 0 initially
		DailyCount: 0, 
	}

	// Trigger Load (private method)
	orch2.loadState()

	// Verify Session Restored
	if len(orch2.activeSessions) != 1 {
		t.Errorf("Expected 1 active session, got %d", len(orch2.activeSessions))
	}
	if sess, ok := orch2.activeSessions["session-xyz"]; !ok {
		t.Error("Expected session-xyz to be loaded")
	} else {
		if sess.Repo != repoName {
			t.Errorf("Expected repo %s, got %s", repoName, sess.Repo)
		}
	}

	// Verify Repo State Restored
	if rc, ok := orch2.activeRepos[repoName]; !ok {
		t.Error("Repo context missing")
	} else {
		if rc.DailyCount != 42 {
			t.Errorf("Expected DailyCount 42, got %d", rc.DailyCount)
		}
	}
}
