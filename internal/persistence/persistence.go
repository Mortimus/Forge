package persistence

import (
	"encoding/json"
	"os"
	"sync"
)

// SessionMetadata tracks information about an active Jules session.
type SessionMetadata struct {
	ID        string `json:"id"`
	Repo      string `json:"repo"`   // Repository Name
	Type      string `json:"type"`   // GapAnalysis, Resolution, or Unknown
	PRURL     string `json:"pr_url"` // Associated Pull Request URL
	StartTime string `json:"start_time"`
}

// RepoState tracks the daily usage limits for a specific repository.
type RepoState struct {
	DailyCount int    `json:"daily_count"`
	LastReset  string `json:"last_reset"` // RFC3339 timestamp
}

// State represents the persistent data for the Forge service.
type State struct {
	LifetimeSessions int                  `json:"lifetime_sessions"`
	ActiveSessions   map[string]SessionMetadata `json:"active_sessions"`
	Repositories     map[string]RepoState       `json:"repositories"` // Key: Repo Name (Owner/Repo)
	LastRepo         string                     `json:"last_repo"`
	LastSessionID    string                     `json:"last_session_id"`
	LastPR           string                     `json:"last_pr"`
}

// Manager handles saving and loading the service state.
type Manager struct {
	path string
	mu   sync.RWMutex
}

// NewManager creates a new State Manager.
func NewManager(path string) *Manager {
	return &Manager{
		path: path,
	}
}

// Load reads the state from disk.
func (m *Manager) Load() (*State, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	data, err := os.ReadFile(m.path)
	if err != nil {
		if os.IsNotExist(err) {
			return &State{
				ActiveSessions: make(map[string]SessionMetadata),
			}, nil
		}
		return nil, err
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	if state.ActiveSessions == nil {
		state.ActiveSessions = make(map[string]SessionMetadata)
	}
	return &state, nil
}

// Save writes the state to disk.
func (m *Manager) Save(state *State) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(m.path, data, 0644)
}
