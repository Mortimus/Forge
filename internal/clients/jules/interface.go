package jules

import "context"

// ClientInterface defines the methods required for interacting with Jules.
// This interface allows for mocking in tests.
type ClientInterface interface {
	ListSources(ctx context.Context) ([]Source, error)
	CreateSession(ctx context.Context, title, prompt, sourceName, branch string) (*Session, error)
	GetSession(ctx context.Context, sessionName string) (*Session, error)
	ListSessions(ctx context.Context) ([]Session, error)
}
