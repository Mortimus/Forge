package mocks

import (
	"context"

	"github.com/mortimus/forge/internal/clients/jules"
)

// JulesMock is a mock implementation of jules.ClientInterface
type JulesMock struct {
	ListSourcesFunc   func(ctx context.Context) ([]jules.Source, error)
	CreateSessionFunc func(ctx context.Context, title, prompt, sourceName, branch string) (*jules.Session, error)
	GetSessionFunc    func(ctx context.Context, sessionName string) (*jules.Session, error)
	ListSessionsFunc  func(ctx context.Context) ([]jules.Session, error)
}

var _ jules.ClientInterface = (*JulesMock)(nil)

func (m *JulesMock) ListSources(ctx context.Context) ([]jules.Source, error) {
	if m.ListSourcesFunc != nil {
		return m.ListSourcesFunc(ctx)
	}
	return nil, nil
}

func (m *JulesMock) CreateSession(ctx context.Context, title, prompt, sourceName, branch string) (*jules.Session, error) {
	if m.CreateSessionFunc != nil {
		return m.CreateSessionFunc(ctx, title, prompt, sourceName, branch)
	}
	return &jules.Session{}, nil
}

func (m *JulesMock) GetSession(ctx context.Context, sessionName string) (*jules.Session, error) {
	if m.GetSessionFunc != nil {
		return m.GetSessionFunc(ctx, sessionName)
	}
	return &jules.Session{}, nil
}

func (m *JulesMock) ListSessions(ctx context.Context) ([]jules.Session, error) {
	if m.ListSessionsFunc != nil {
		return m.ListSessionsFunc(ctx)
	}
	return nil, nil
}
