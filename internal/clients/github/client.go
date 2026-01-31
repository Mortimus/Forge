// Package github provides a wrapper around the go-github client for interacting with the GitHub API.
package github

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/v69/github"
	"golang.org/x/oauth2"
)

// Client encapsulates the GitHub API client and repository context.
type Client struct {
	client *github.Client
	owner  string
	repo   string
}

// NewClient creates a new GitHub client for specific repository.
// token is the GitHub Personal Access Token.
// repoFullName is the repository in "owner/repo" format.
func NewClient(ctx context.Context, token, repoFullName string) (*Client, error) {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	ghClient := github.NewClient(tc)

	parts := strings.Split(repoFullName, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid repo name: %s (expected owner/name)", repoFullName)
	}

	return &Client{
		client: ghClient,
		owner:  parts[0],
		repo:   parts[1],
	}, nil
}

// ListOpenIssuesByLabel returns all open issues with the given label
func (c *Client) ListOpenIssuesByLabel(ctx context.Context, label string) ([]*github.Issue, error) {
	opts := &github.IssueListByRepoOptions{
		State:       "open",
		Labels:      []string{label},
		ListOptions: github.ListOptions{PerPage: 100},
	}

	var allIssues []*github.Issue
	for {
		issues, resp, err := c.client.Issues.ListByRepo(ctx, c.owner, c.repo, opts)
		if err != nil {
			return nil, err
		}
		allIssues = append(allIssues, issues...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return allIssues, nil
}

// CreateIssue creates a new issue
func (c *Client) CreateIssue(ctx context.Context, title, body string, labels []string) (*github.Issue, error) {
	req := &github.IssueRequest{
		Title:  &title,
		Body:   &body,
		Labels: &labels,
	}
	issue, _, err := c.client.Issues.Create(ctx, c.owner, c.repo, req)
	return issue, err
}

// AddLabel adds labels to an issue
func (c *Client) AddLabel(ctx context.Context, number int, labels []string) error {
	_, _, err := c.client.Issues.AddLabelsToIssue(ctx, c.owner, c.repo, number, labels)
	return err
}

// SpecHasGapAnalysis checks if a Spec #ID has a corresponding Gap Analysis issue
func (c *Client) SpecHasGapAnalysis(ctx context.Context, specID int, gapLabel string) bool {
	// Query: repo:owner/repo label:gap-analysis "derived from Issue #ID"
	query := fmt.Sprintf("repo:%s/%s label:%s \"derived from Issue #%d\"", c.owner, c.repo, gapLabel, specID)
	result, _, err := c.client.Search.Issues(ctx, query, &github.SearchOptions{})
	if err != nil {
		// Log error? For now assume false to be safe, or true to avoid dupes? 
		// Safe to return false (might duplicate), but strict to return true (might stall)?
		// Let's print invalid query error if needed handling.
		return false
	}
	return result.Total != nil && *result.Total > 0
}

// GetIssueContent returns the body of a specific issue
func (c *Client) GetIssueContent(ctx context.Context, number int) (string, error) {
	issue, _, err := c.client.Issues.Get(ctx, c.owner, c.repo, number)
	if err != nil {
		return "", err
	}
	if issue.Body == nil {
		return "", nil
	}
	return *issue.Body, nil
}

// CommentOnIssue adds a comment to an issue
func (c *Client) CommentOnIssue(ctx context.Context, number int, body string) error {
	comment := &github.IssueComment{Body: &body}
	_, _, err := c.client.Issues.CreateComment(ctx, c.owner, c.repo, number, comment)
	return err
}

// CloseIssue closes an issue
func (c *Client) CloseIssue(ctx context.Context, number int) error {
	state := "closed"
	req := &github.IssueRequest{State: &state}
	_, _, err := c.client.Issues.Edit(ctx, c.owner, c.repo, number, req)
	return err
}

// MergePR merges a pull request
func (c *Client) MergePR(ctx context.Context, number int) error {
	_, _, err := c.client.PullRequests.Merge(ctx, c.owner, c.repo, number, "Automated merge by Ralph", nil)
	return err
}

// GetPRStatus checks the combined status of a Ref (Head SHA of PR)
func (c *Client) GetPRStatus(ctx context.Context, ref string) (string, error) {
	// check-runs are better for Actions
	// But simple implementation: use combined status
	status, _, err := c.client.Repositories.GetCombinedStatus(ctx, c.owner, c.repo, ref, nil)
	if err != nil {
		return "", err
	}
	return *status.State, nil
}

// GetPR returns a PR to get its Head SHA
func (c *Client) GetPR(ctx context.Context, number int) (*github.PullRequest, error) {
	pr, _, err := c.client.PullRequests.Get(ctx, c.owner, c.repo, number)
	return pr, err
}

// GetFileContent reads a file from the repository (default branch)
func (c *Client) GetFileContent(ctx context.Context, path string) (string, error) {
	content, _, _, err := c.client.Repositories.GetContents(ctx, c.owner, c.repo, path, nil)
	if err != nil {
		return "", err
	}
	if content == nil {
		return "", fmt.Errorf("file not found or is a directory: %s", path)
	}
	return content.GetContent()
}

// ListFiles returns a list of files in a directory
func (c *Client) ListFiles(ctx context.Context, path string) ([]string, error) {
	_, dirContent, _, err := c.client.Repositories.GetContents(ctx, c.owner, c.repo, path, nil)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, file := range dirContent {
		if file.GetType() == "file" {
			files = append(files, file.GetName())
		}
	}
	return files, nil
}
