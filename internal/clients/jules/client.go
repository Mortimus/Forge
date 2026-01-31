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

// BaseURL is the endpoint for the Jules API.
const BaseURL = "https://jules.googleapis.com"

// Client is a wrapper for the Jules API HTTP client.
type Client struct {
	apiKey     string
	httpClient *http.Client
}

// NewClient creates a new Jules API client with the provided API key.
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Data Types

type Source struct {
	Name       string `json:"name"`
	ID         string `json:"id"`
	GithubRepo struct {
		Owner string `json:"owner"`
		Repo  string `json:"repo"`
	} `json:"githubRepo"`
}

type ListSourcesResponse struct {
	Sources []Source `json:"sources"`
}

type CreateSessionRequest struct {
	Title         string        `json:"title"`
	Prompt        string        `json:"prompt"`
	SourceContext SourceContext `json:"sourceContext"`
	AutomationMode string       `json:"automationMode"`
}

type SourceContext struct {
	Source            string            `json:"source"`
	GithubRepoContext GithubRepoContext `json:"githubRepoContext"`
}

type GithubRepoContext struct {
	StartingBranch string `json:"startingBranch"`
}

type Session struct {
	Name    string         `json:"name"` // Session ID basically
	Status  string         `json:"status"`
	Outputs []SessionOutput `json:"outputs"`
}

type SessionOutput struct {
	PullRequest *PullRequestOutput `json:"pullRequest,omitempty"`
}

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

	url := fmt.Sprintf("%s%s", BaseURL, path)
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-Goog-Api-Key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")

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

func (c *Client) ListSources(ctx context.Context) ([]Source, error) {
	respBytes, err := c.do(ctx, "GET", "/v1alpha/sources", nil)
	if err != nil {
		return nil, err
	}

	var resp ListSourcesResponse
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return resp.Sources, nil
}

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
