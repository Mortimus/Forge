// Package jules provides a client for interacting with the Google Jules API.
package jules

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// BaseURL is the default endpoint for the Jules API.
const DefaultBaseURL = "https://jules.googleapis.com"

// Client is a wrapper for the Jules API HTTP client.
type Client struct {
	BaseURL    string
	apiKey     string
	httpClient *http.Client
	limiter    *time.Ticker
}

// NewClient creates a new Jules API client with the provided API key.
func NewClient(apiKey string, interval time.Duration) *Client {
	if interval == 0 {
		interval = 60 * time.Second
	}
	return &Client{
		BaseURL: DefaultBaseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		limiter: time.NewTicker(interval),
	}
}

// Data Types

// Source represents a Jules data source (e.g. a GitHub repository).
type Source struct {
	Name       string `json:"name"`
	ID         string `json:"id"`
	GithubRepo struct {
		Owner string `json:"owner"`
		Repo  string `json:"repo"`
	} `json:"githubRepo"`
}

// ListSourcesResponse is the API response for listing sources.
type ListSourcesResponse struct {
	Sources       []Source `json:"sources"`
	NextPageToken string   `json:"nextPageToken"`
}

// ListSessionsResponse is the API response for listing sessions.
type ListSessionsResponse struct {
	Sessions      []Session `json:"sessions"`
	NextPageToken string    `json:"nextPageToken"`
}

// CreateSessionRequest is the payload for creating a new session.
type CreateSessionRequest struct {
	Title          string        `json:"title"`
	Prompt         string        `json:"prompt"`
	SourceContext  SourceContext `json:"sourceContext"`
	AutomationMode string        `json:"automationMode"`
}

// SourceContext defines the context for a session.
type SourceContext struct {
	Source            string            `json:"source"`
	GithubRepoContext GithubRepoContext `json:"githubRepoContext"`
}

// GithubRepoContext defines the branch context for a GitHub repo.
type GithubRepoContext struct {
	StartingBranch string `json:"startingBranch"`
}

// Session represents a Jules coding session.
type Session struct {
	Name          string          `json:"name"` // Session ID basically
	State         string          `json:"state"`
	SourceContext SourceContext   `json:"sourceContext"`
	Outputs       []SessionOutput `json:"outputs"`
}

// SessionOutput represents an output artifact from a session.
type SessionOutput struct {
	PullRequest *PullRequestOutput `json:"pullRequest,omitempty"`
}

// PullRequestOutput contains details about a created PR.
type PullRequestOutput struct {
	URL   string `json:"url"`
	Title string `json:"title"`
}

// Methods

func (c *Client) do(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewBuffer(jsonBody)
	}

	url := fmt.Sprintf("%s%s", c.BaseURL, path)
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-Goog-Api-Key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	// Wait for rate limit
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-c.limiter.C:
		// Proceed
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("api error %d: %s", resp.StatusCode, string(respBody))
	}

	return io.ReadAll(resp.Body)
}

// ListSources retrieves all available data sources (repositories) from the Jules API.
// It handles pagination automatically.
func (c *Client) ListSources(ctx context.Context) ([]Source, error) {
	var allSources []Source
	pageToken := ""

	for {
		path := "/v1alpha/sources"
		if pageToken != "" {
			path = fmt.Sprintf("%s?pageToken=%s", path, pageToken)
		}

		respBytes, err := c.do(ctx, "GET", path, nil)
		if err != nil {
			return nil, err
		}

		var resp ListSourcesResponse
		if err := json.Unmarshal(respBytes, &resp); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}

		allSources = append(allSources, resp.Sources...)
		pageToken = resp.NextPageToken

		if pageToken == "" {
			break
		}
	}

	return allSources, nil
}

// CreateSession initializes a new Jules session with the given title, prompt, source, and branch.
// It uses "AUTO_CREATE_PR" mode by default.
func (c *Client) CreateSession(ctx context.Context, title, prompt, sourceName, branch string) (*Session, error) {
	req := CreateSessionRequest{
		Title:  title,
		Prompt: prompt,
		SourceContext: SourceContext{
			Source: sourceName,
			GithubRepoContext: GithubRepoContext{
				StartingBranch: branch,
			},
		},
		AutomationMode: "AUTO_CREATE_PR",
	}

	respBytes, err := c.do(ctx, "POST", "/v1alpha/sessions", req)
	if err != nil {
		return nil, err
	}

	var session Session
	if err := json.Unmarshal(respBytes, &session); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &session, nil
}

// GetSession retrieves details for a specific session by its resource name.
func (c *Client) GetSession(ctx context.Context, sessionName string) (*Session, error) {
	// sessionName is full resource name e.g. "sessions/123"
	path := fmt.Sprintf("/v1alpha/%s", sessionName)
	respBytes, err := c.do(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var session Session
	if err := json.Unmarshal(respBytes, &session); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &session, nil
}

// ListSessions retrieves all sessions for the authenticated user, handling pagination.
func (c *Client) ListSessions(ctx context.Context) ([]Session, error) {
	var allSessions []Session
	pageToken := ""

	for {
		path := "/v1alpha/sessions"
		if pageToken != "" {
			path = fmt.Sprintf("%s?pageToken=%s", path, pageToken)
		}

		respBody, err := c.do(ctx, "GET", path, nil)
		if err != nil {
			return nil, err
		}

		var resp ListSessionsResponse
		if err := json.Unmarshal(respBody, &resp); err != nil {
			return nil, fmt.Errorf("failed to unmarshal sessions: %w", err)
		}

		allSessions = append(allSessions, resp.Sessions...)
		pageToken = resp.NextPageToken

		if pageToken == "" {
			break
		}
	}

	return allSessions, nil
}
