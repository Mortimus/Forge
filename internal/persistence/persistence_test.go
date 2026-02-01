package persistence

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestManager_SaveAndLoad(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "forge_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	statePath := filepath.Join(tmpDir, "state.json")
	manager := NewManager(statePath)

	// Create a dummy state
	state := &State{
		LifetimeSessions: 10,
		ActiveSessions: map[string]SessionMetadata{
			"session-1": {
				ID:        "session-1",
				Repo:      "owner/repo",
				Type:      "GAP_ANALYSIS",
				StartTime: time.Now().Format(time.RFC3339),
			},
		},
		Repositories: map[string]RepoState{
			"owner/repo": {
				DailyCount: 5,
				LastReset:  time.Now().Format(time.RFC3339),
			},
		},
		LastRepo: "owner/repo",
	}

	// Save
	if err := manager.Save(state); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Load
	loadedState, err := manager.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Verify
	if loadedState.LifetimeSessions != 10 {
		t.Errorf("LifetimeSessions = %d, want 10", loadedState.LifetimeSessions)
	}
	if len(loadedState.ActiveSessions) != 1 {
		t.Errorf("ActiveSessions count = %d, want 1", len(loadedState.ActiveSessions))
	}
	if len(loadedState.Repositories) != 1 {
		t.Errorf("Repositories count = %d, want 1", len(loadedState.Repositories))
	}
	
	repoState, ok := loadedState.Repositories["owner/repo"]
	if !ok {
		t.Error("Repo state not found")
	}
	if repoState.DailyCount != 5 {
		t.Errorf("DailyCount = %d, want 5", repoState.DailyCount)
	}
}

func TestManager_LoadNonExistent(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "forge_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	statePath := filepath.Join(tmpDir, "nonexistent.json")
	manager := NewManager(statePath)

	state, err := manager.Load()
	if err != nil {
		t.Fatalf("Load failed for nonexistent file: %v", err)
	}

	if state == nil {
		t.Fatal("State should not be nil")
	}
	if len(state.ActiveSessions) != 0 {
		t.Error("ActiveSessions should be empty")
	}
}
