package github

import (
	"context"

	"github.com/google/go-github/v69/github"
)

// ClientInterface defines the methods required for interacting with GitHub.
// This interface allows for mocking in tests.
type ClientInterface interface {
	ListOpenIssuesByLabel(ctx context.Context, label string) ([]*github.Issue, error)
	ListOpenPullRequests(ctx context.Context) ([]*github.PullRequest, error)
	CreateIssue(ctx context.Context, title, body string, labels []string) (*github.Issue, error)
	AddLabel(ctx context.Context, number int, labels []string) error
	SpecHasGapAnalysis(ctx context.Context, specID int, gapLabel string) bool
	GetIssueContent(ctx context.Context, number int) (string, error)
	CommentOnIssue(ctx context.Context, number int, body string) error
	CloseIssue(ctx context.Context, number int) error
	MergePR(ctx context.Context, number int) error
	GetPRStatus(ctx context.Context, ref string) (string, error)
	GetPR(ctx context.Context, number int) (*github.PullRequest, error)
	GetFileContent(ctx context.Context, path string) (string, error)
	ListFiles(ctx context.Context, path string) ([]string, error)
	DeleteBranch(ctx context.Context, branchName string) error
	DeleteFile(ctx context.Context, path, message string) error
}
