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
	SendMessageFunc   func(ctx context.Context, sessionName, message string) error
	ApprovePlanFunc   func(ctx context.Context, sessionName string) error
	DeleteSessionFunc func(ctx context.Context, sessionName string) error
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

func (m *JulesMock) SendMessage(ctx context.Context, sessionName, message string) error {
	if m.SendMessageFunc != nil {
		return m.SendMessageFunc(ctx, sessionName, message)
	}
	return nil
}

func (m *JulesMock) ApprovePlan(ctx context.Context, sessionName string) error {
	if m.ApprovePlanFunc != nil {
		return m.ApprovePlanFunc(ctx, sessionName)
	}
	return nil
}

func (m *JulesMock) DeleteSession(ctx context.Context, sessionName string) error {
	if m.DeleteSessionFunc != nil {
		return m.DeleteSessionFunc(ctx, sessionName)
	}
	return nil
}
