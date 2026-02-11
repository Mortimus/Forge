// Package github provides a wrapper around the go-github client for interacting with the GitHub API.
package github

import (
	"context"
	"fmt"
	"strings"

	"time"

	"github.com/google/go-github/v69/github"
	"golang.org/x/oauth2"
)

// Client encapsulates the GitHub API client and repository context.
type Client struct {
	client  *github.Client
	owner   string
	repo    string
	bp      *Backpressure
}

// NewClient creates a new GitHub client for specific repository.
// token is the GitHub Personal Access Token.
// repoFullName is the repository in "owner/repo" format.
func NewClient(ctx context.Context, token, repoFullName string, interval time.Duration) (*Client, error) {
	if interval == 0 {
		interval = 1 * time.Second
	}
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
		client:  ghClient,
		owner:   parts[0],
		repo:    parts[1],
		bp:      NewBackpressure(interval),
	}, nil
}

func (c *Client) wait(ctx context.Context) error {
	return c.bp.Wait(ctx)
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
		if err := c.wait(ctx); err != nil {
			return nil, err
		}
		issues, resp, err := c.client.Issues.ListByRepo(ctx, c.owner, c.repo, opts)
		c.bp.HandleResponse(resp, err)
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

// ListOpenPullRequests returns all open pull requests
func (c *Client) ListOpenPullRequests(ctx context.Context) ([]*github.PullRequest, error) {
	opts := &github.PullRequestListOptions{
		State:       "open",
		ListOptions: github.ListOptions{PerPage: 100},
	}
	var allPRs []*github.PullRequest
	for {
		if err := c.wait(ctx); err != nil {
			return nil, err
		}
		prs, resp, err := c.client.PullRequests.List(ctx, c.owner, c.repo, opts)
		c.bp.HandleResponse(resp, err)
		if err != nil {
			return nil, err
		}
		allPRs = append(allPRs, prs...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return allPRs, nil
}

// CreateIssue creates a new issue in the repository.
func (c *Client) CreateIssue(ctx context.Context, title, body string, labels []string) (*github.Issue, error) {
	if err := c.wait(ctx); err != nil {
		return nil, err
	}
	req := &github.IssueRequest{
		Title:  &title,
		Body:   &body,
		Labels: &labels,
	}
	issue, resp, err := c.client.Issues.Create(ctx, c.owner, c.repo, req)
	c.bp.HandleResponse(resp, err)
	return issue, err
}

// AddLabel adds labels to an issue
func (c *Client) AddLabel(ctx context.Context, number int, labels []string) error {
	if err := c.wait(ctx); err != nil {
		return err
	}
	_, resp, err := c.client.Issues.AddLabelsToIssue(ctx, c.owner, c.repo, number, labels)
	c.bp.HandleResponse(resp, err)
	return err
}

// SpecHasGapAnalysis checks if a Spec #ID has a corresponding Gap Analysis issue
func (c *Client) SpecHasGapAnalysis(ctx context.Context, specID int, gapLabel string) bool {
	if err := c.wait(ctx); err != nil {
		return false
	}
	// Query: repo:owner/repo label:gap-analysis "derived from Issue #ID"
	query := fmt.Sprintf("repo:%s/%s label:%s \"derived from Issue #%d\"", c.owner, c.repo, gapLabel, specID)
	result, resp, err := c.client.Search.Issues(ctx, query, &github.SearchOptions{})
	c.bp.HandleResponse(resp, err)
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
	if err := c.wait(ctx); err != nil {
		return "", err
	}
	issue, resp, err := c.client.Issues.Get(ctx, c.owner, c.repo, number)
	c.bp.HandleResponse(resp, err)
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
	if err := c.wait(ctx); err != nil {
		return err
	}
	comment := &github.IssueComment{Body: &body}
	_, resp, err := c.client.Issues.CreateComment(ctx, c.owner, c.repo, number, comment)
	c.bp.HandleResponse(resp, err)
	return err
}

// CloseIssue closes an issue
func (c *Client) CloseIssue(ctx context.Context, number int) error {
	if err := c.wait(ctx); err != nil {
		return err
	}
	state := "closed"
	req := &github.IssueRequest{State: &state}
	_, resp, err := c.client.Issues.Edit(ctx, c.owner, c.repo, number, req)
	c.bp.HandleResponse(resp, err)
	return err
}

// MergePR merges a pull request
func (c *Client) MergePR(ctx context.Context, number int) error {
	if err := c.wait(ctx); err != nil {
		return err
	}
	_, resp, err := c.client.PullRequests.Merge(ctx, c.owner, c.repo, number, "Automated merge by Forge", nil)
	c.bp.HandleResponse(resp, err)
	return err
}

// GetPRStatus checks the combined status of a Ref (Head SHA of PR)
func (c *Client) GetPRStatus(ctx context.Context, ref string) (string, error) {
	if err := c.wait(ctx); err != nil {
		return "", err
	}
	// check-runs are better for Actions
	// But simple implementation: use combined status
	status, resp, err := c.client.Repositories.GetCombinedStatus(ctx, c.owner, c.repo, ref, nil)
	c.bp.HandleResponse(resp, err)
	if err != nil {
		return "", err
	}
	return *status.State, nil
}

// GetPR returns a PR to get its Head SHA
func (c *Client) GetPR(ctx context.Context, number int) (*github.PullRequest, error) {
	if err := c.wait(ctx); err != nil {
		return nil, err
	}
	pr, resp, err := c.client.PullRequests.Get(ctx, c.owner, c.repo, number)
	c.bp.HandleResponse(resp, err)
	return pr, err
}

// GetFileContent reads a file from the repository (default branch)
func (c *Client) GetFileContent(ctx context.Context, path string) (string, error) {
	if err := c.wait(ctx); err != nil {
		return "", err
	}
	content, _, resp, err := c.client.Repositories.GetContents(ctx, c.owner, c.repo, path, nil)
	c.bp.HandleResponse(resp, err)
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
	if err := c.wait(ctx); err != nil {
		return nil, err
	}
	_, dirContent, resp, err := c.client.Repositories.GetContents(ctx, c.owner, c.repo, path, nil)
	c.bp.HandleResponse(resp, err)
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

// DeleteBranch deletes a branch in the repository
func (c *Client) DeleteBranch(ctx context.Context, branchName string) error {
	if err := c.wait(ctx); err != nil {
		return err
	}
	ref := fmt.Sprintf("refs/heads/%s", branchName)
	resp, err := c.client.Git.DeleteRef(ctx, c.owner, c.repo, ref)
	c.bp.HandleResponse(resp, err)
	return err
}

// DeleteFile deletes a file from the repository
func (c *Client) DeleteFile(ctx context.Context, path, message string) error {
	if err := c.wait(ctx); err != nil {
		return err
	}
	// Get the file content to get the SHA
	fileContent, _, resp, err := c.client.Repositories.GetContents(ctx, c.owner, c.repo, path, nil)
	c.bp.HandleResponse(resp, err)
	if err != nil {
		return err
	}
	if fileContent == nil {
		return fmt.Errorf("file not found: %s", path)
	}

	opts := &github.RepositoryContentFileOptions{
		Message: github.String(message),
		SHA:     fileContent.SHA,
	}
	_, resp, err = c.client.Repositories.DeleteFile(ctx, c.owner, c.repo, path, opts)
	c.bp.HandleResponse(resp, err)
	return err
}
