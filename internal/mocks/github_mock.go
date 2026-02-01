package mocks

import (
	"context"

	ghlib "github.com/google/go-github/v69/github"
	"github.com/mortimus/forge/internal/clients/github"
)

// GithubMock is a mock implementation of github.ClientInterface
type GithubMock struct {
	ListOpenIssuesByLabelFunc func(ctx context.Context, label string) ([]*ghlib.Issue, error)
	ListOpenPullRequestsFunc  func(ctx context.Context) ([]*ghlib.PullRequest, error)
	CreateIssueFunc           func(ctx context.Context, title, body string, labels []string) (*ghlib.Issue, error)
	AddLabelFunc              func(ctx context.Context, number int, labels []string) error
	SpecHasGapAnalysisFunc    func(ctx context.Context, specID int, gapLabel string) bool
	GetIssueContentFunc       func(ctx context.Context, number int) (string, error)
	CommentOnIssueFunc        func(ctx context.Context, number int, body string) error
	CloseIssueFunc            func(ctx context.Context, number int) error
	MergePRFunc               func(ctx context.Context, number int) error
	GetPRStatusFunc           func(ctx context.Context, ref string) (string, error)
	GetPRFunc                 func(ctx context.Context, number int) (*ghlib.PullRequest, error)
	GetFileContentFunc        func(ctx context.Context, path string) (string, error)
	ListFilesFunc             func(ctx context.Context, path string) ([]string, error)
	DeleteBranchFunc          func(ctx context.Context, branchName string) error
	DeleteFileFunc            func(ctx context.Context, path, message string) error
}

var _ github.ClientInterface = (*GithubMock)(nil)

func (m *GithubMock) ListOpenIssuesByLabel(ctx context.Context, label string) ([]*ghlib.Issue, error) {
	if m.ListOpenIssuesByLabelFunc != nil {
		return m.ListOpenIssuesByLabelFunc(ctx, label)
	}
	return nil, nil
}

func (m *GithubMock) ListOpenPullRequests(ctx context.Context) ([]*ghlib.PullRequest, error) {
	if m.ListOpenPullRequestsFunc != nil {
		return m.ListOpenPullRequestsFunc(ctx)
	}
	return nil, nil
}

func (m *GithubMock) CreateIssue(ctx context.Context, title, body string, labels []string) (*ghlib.Issue, error) {
	if m.CreateIssueFunc != nil {
		return m.CreateIssueFunc(ctx, title, body, labels)
	}
	return &ghlib.Issue{}, nil
}

func (m *GithubMock) AddLabel(ctx context.Context, number int, labels []string) error {
	if m.AddLabelFunc != nil {
		return m.AddLabelFunc(ctx, number, labels)
	}
	return nil
}

func (m *GithubMock) SpecHasGapAnalysis(ctx context.Context, specID int, gapLabel string) bool {
	if m.SpecHasGapAnalysisFunc != nil {
		return m.SpecHasGapAnalysisFunc(ctx, specID, gapLabel)
	}
	return false
}

func (m *GithubMock) GetIssueContent(ctx context.Context, number int) (string, error) {
	if m.GetIssueContentFunc != nil {
		return m.GetIssueContentFunc(ctx, number)
	}
	return "", nil
}

func (m *GithubMock) CommentOnIssue(ctx context.Context, number int, body string) error {
	if m.CommentOnIssueFunc != nil {
		return m.CommentOnIssueFunc(ctx, number, body)
	}
	return nil
}

func (m *GithubMock) CloseIssue(ctx context.Context, number int) error {
	if m.CloseIssueFunc != nil {
		return m.CloseIssueFunc(ctx, number)
	}
	return nil
}

func (m *GithubMock) MergePR(ctx context.Context, number int) error {
	if m.MergePRFunc != nil {
		return m.MergePRFunc(ctx, number)
	}
	return nil
}

func (m *GithubMock) GetPRStatus(ctx context.Context, ref string) (string, error) {
	if m.GetPRStatusFunc != nil {
		return m.GetPRStatusFunc(ctx, ref)
	}
	return "", nil
}

func (m *GithubMock) GetPR(ctx context.Context, number int) (*ghlib.PullRequest, error) {
	if m.GetPRFunc != nil {
		return m.GetPRFunc(ctx, number)
	}
	return &ghlib.PullRequest{}, nil
}

func (m *GithubMock) GetFileContent(ctx context.Context, path string) (string, error) {
	if m.GetFileContentFunc != nil {
		return m.GetFileContentFunc(ctx, path)
	}
	return "", nil
}

func (m *GithubMock) ListFiles(ctx context.Context, path string) ([]string, error) {
	if m.ListFilesFunc != nil {
		return m.ListFilesFunc(ctx, path)
	}
	return nil, nil
}

func (m *GithubMock) DeleteBranch(ctx context.Context, branchName string) error {
	if m.DeleteBranchFunc != nil {
		return m.DeleteBranchFunc(ctx, branchName)
	}
	return nil
}

func (m *GithubMock) DeleteFile(ctx context.Context, path, message string) error {
	if m.DeleteFileFunc != nil {
		return m.DeleteFileFunc(ctx, path, message)
	}
	return nil
}
