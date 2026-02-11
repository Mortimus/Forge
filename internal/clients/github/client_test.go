package github

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/google/go-github/v69/github"
)

// setupMockServer creates a test server and a github client configured to use it.
func setupMockServer() (*http.ServeMux, *httptest.Server, *Client) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)

	// github.NewClient returns a client pointing to api.github.com
	// We need to overwrite BaseURL to point to server.URL

	client := github.NewClient(nil)
	url, _ := url.Parse(server.URL + "/")
	client.BaseURL = url
	client.UploadURL = url

	c := &Client{
		client: client,
		owner:  "owner",
		repo:   "repo",
		bp:     NewBackpressure(1 * time.Millisecond),
	}

	return mux, server, c
}

func TestNewClient(t *testing.T) {
	ctx := context.Background()

	// Valid
	c, err := NewClient(ctx, "token", "owner/repo", 1*time.Millisecond)
	if err != nil {
		t.Errorf("NewClient valid failed: %v", err)
	}
	if c.owner != "owner" || c.repo != "repo" {
		t.Errorf("NewClient owner/repo mismatch: %s/%s", c.owner, c.repo)
	}

	// Invalid
	_, err = NewClient(ctx, "token", "invalid", 1*time.Millisecond)
	if err == nil {
		t.Error("NewClient invalid expected error, got nil")
	}
}

func TestListOpenIssuesByLabel(t *testing.T) {
	mux, server, client := setupMockServer()
	defer server.Close()

	mux.HandleFunc("/repos/owner/repo/issues", func(w http.ResponseWriter, r *http.Request) {
		if r.FormValue("state") != "open" {
			t.Errorf("state = %v, want open", r.FormValue("state"))
		}
		if r.FormValue("labels") != "bug" {
			t.Errorf("labels = %v, want bug", r.FormValue("labels"))
		}
		fmt.Fprint(w, `[{"number":1, "title":"Test Issue"}]`)
	})

	issues, err := client.ListOpenIssuesByLabel(context.Background(), "bug")
	if err != nil {
		t.Fatalf("ListOpenIssuesByLabel returned error: %v", err)
	}
	if len(issues) != 1 {
		t.Errorf("expected 1 issue, got %d", len(issues))
	}
	if issues[0].GetTitle() != "Test Issue" {
		t.Errorf("expected title 'Test Issue', got %s", issues[0].GetTitle())
	}
}

func TestListOpenIssuesByLabel_Error(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	// Create a client with the mock server's URL
	// Note: NewClient expects a token and owner/repo, but for this error test,
	// we're primarily testing the HTTP interaction, so a dummy token/repo is fine.
	// We also need to manually set the BaseURL for the underlying github client.
	c, err := NewClient(context.Background(), "token", "owner/repo", 1*time.Millisecond)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	c.client.BaseURL, _ = url.Parse(server.URL + "/")

	mux.HandleFunc("/repos/owner/repo/issues", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	_, err = c.ListOpenIssuesByLabel(context.Background(), "bug")
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestListOpenPullRequests(t *testing.T) {
	mux, server, client := setupMockServer()
	defer server.Close()

	mux.HandleFunc("/repos/owner/repo/pulls", func(w http.ResponseWriter, r *http.Request) {
		if r.FormValue("state") != "open" {
			t.Errorf("state = %v, want open", r.FormValue("state"))
		}
		fmt.Fprint(w, `[{"number":2, "title":"Test PR"}]`)
	})

	prs, err := client.ListOpenPullRequests(context.Background())
	if err != nil {
		t.Fatalf("ListOpenPullRequests returned error: %v", err)
	}
	if len(prs) != 1 {
		t.Errorf("got %d prs, want 1", len(prs))
	}
	if prs[0].GetNumber() != 2 {
		t.Errorf("got PR number %d, want 2", prs[0].GetNumber())
	}
}

func TestCreateIssue(t *testing.T) {
	mux, server, client := setupMockServer()
	defer server.Close()

	mux.HandleFunc("/repos/owner/repo/issues", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Method = %v, want POST", r.Method)
		}
		// Could decode body and check fields but simplified for now
		fmt.Fprint(w, `{"number":3, "title":"New Issue"}`)
	})

	issue, err := client.CreateIssue(context.Background(), "New Issue", "Body", []string{"label"})
	if err != nil {
		t.Fatalf("CreateIssue returned error: %v", err)
	}
	if issue.GetNumber() != 3 {
		t.Errorf("got issue number %d, want 3", issue.GetNumber())
	}
}

func TestAddLabel(t *testing.T) {
	mux, server, client := setupMockServer()
	defer server.Close()

	mux.HandleFunc("/repos/owner/repo/issues/1/labels", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Method = %v, want POST", r.Method)
		}
		fmt.Fprint(w, `[]`)
	})

	err := client.AddLabel(context.Background(), 1, []string{"label"})
	if err != nil {
		t.Fatalf("AddLabel returned error: %v", err)
	}
}

func TestSpecHasGapAnalysis(t *testing.T) {
	mux, server, client := setupMockServer()
	defer server.Close()

	mux.HandleFunc("/search/issues", func(w http.ResponseWriter, r *http.Request) {
		q := r.FormValue("q")
		expected := `repo:owner/repo label:gap-analysis "derived from Issue #123"`
		if q != expected {
			t.Errorf("q = %v, want %v", q, expected)
		}
		fmt.Fprint(w, `{"total_count": 1, "items": [{}]}`)
	})

	got := client.SpecHasGapAnalysis(context.Background(), 123, "gap-analysis")
	if !got {
		t.Error("SpecHasGapAnalysis = false, want true")
	}
}

func TestGetIssueContent(t *testing.T) {
	mux, server, client := setupMockServer()
	defer server.Close()

	mux.HandleFunc("/repos/owner/repo/issues/1", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"number":1, "body":"content"}`)
	})

	body, err := client.GetIssueContent(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetIssueContent returned error: %v", err)
	}
	if body != "content" {
		t.Errorf("body = %v, want content", body)
	}
}

func TestCommentOnIssue(t *testing.T) {
	mux, server, client := setupMockServer()
	defer server.Close()

	mux.HandleFunc("/repos/owner/repo/issues/1/comments", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Method = %v, want POST", r.Method)
		}
		fmt.Fprint(w, `{}`)
	})

	err := client.CommentOnIssue(context.Background(), 1, "comment")
	if err != nil {
		t.Fatalf("CommentOnIssue returned error: %v", err)
	}
}

func TestCloseIssue(t *testing.T) {
	mux, server, client := setupMockServer()
	defer server.Close()

	mux.HandleFunc("/repos/owner/repo/issues/1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PATCH" {
			t.Errorf("Method = %v, want PATCH", r.Method)
		}
		fmt.Fprint(w, `{}`)
	})

	err := client.CloseIssue(context.Background(), 1)
	if err != nil {
		t.Fatalf("CloseIssue returned error: %v", err)
	}
}

func TestMergePR(t *testing.T) {
	mux, server, client := setupMockServer()
	defer server.Close()

	mux.HandleFunc("/repos/owner/repo/pulls/1/merge", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("Method = %v, want PUT", r.Method)
		}
		fmt.Fprint(w, `{"merged": true}`)
	})

	err := client.MergePR(context.Background(), 1)
	if err != nil {
		t.Fatalf("MergePR returned error: %v", err)
	}
}

func TestGetPRStatus(t *testing.T) {
	mux, server, client := setupMockServer()
	defer server.Close()

	mux.HandleFunc("/repos/owner/repo/commits/sha/status", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"state": "success"}`)
	})

	status, err := client.GetPRStatus(context.Background(), "sha")
	if err != nil {
		t.Fatalf("GetPRStatus returned error: %v", err)
	}
	if status != "success" {
		t.Errorf("status = %v, want success", status)
	}
}

func TestGetPR(t *testing.T) {
	mux, server, client := setupMockServer()
	defer server.Close()

	mux.HandleFunc("/repos/owner/repo/pulls/1", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"number": 1}`)
	})

	pr, err := client.GetPR(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetPR returned error: %v", err)
	}
	if pr.GetNumber() != 1 {
		t.Errorf("pr number = %d, want 1", pr.GetNumber())
	}
}

func TestGetFileContent(t *testing.T) {
	mux, server, client := setupMockServer()
	defer server.Close()

	mux.HandleFunc("/repos/owner/repo/contents/file.txt", func(w http.ResponseWriter, r *http.Request) {
		// Response for GetContents is usually base64 encoded content or raw if media type set,
		// but go-github handles decoding. The mock needs to return JSON that matches structure.
		// simplified:
		fmt.Fprint(w, `{"type":"file", "encoding":"base64", "content":"aGVsbG8="}`) // hello
	})

	content, err := client.GetFileContent(context.Background(), "file.txt")
	if err != nil {
		t.Fatalf("GetFileContent returned error: %v", err)
	}
	if content != "hello" {
		t.Errorf("content = %v, want hello", content)
	}

	// Test Not Found
	mux.HandleFunc("/repos/owner/repo/contents/missing", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	})
	_, err = client.GetFileContent(context.Background(), "missing")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestListFiles(t *testing.T) {
	mux, server, client := setupMockServer()
	defer server.Close()

	mux.HandleFunc("/repos/owner/repo/contents/dir", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[
			{"name":"file1.txt", "type":"file"},
			{"name":"subdir", "type":"dir"}
		]`)
	})

	files, err := client.ListFiles(context.Background(), "dir")
	if err != nil {
		t.Fatalf("ListFiles returned error: %v", err)
	}
	expected := []string{"file1.txt"}
	if !reflect.DeepEqual(files, expected) {
		t.Errorf("files = %v, want %v", files, expected)
	}
}

func TestListOpenPullRequests_Error(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	c, _ := NewClient(context.Background(), "token", "owner/repo", 1*time.Millisecond)
	c.client.BaseURL, _ = url.Parse(server.URL + "/")

	mux.HandleFunc("/repos/owner/repo/pulls", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	_, err := c.ListOpenPullRequests(context.Background())
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestCreateIssue_Error(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	c, _ := NewClient(context.Background(), "token", "owner/repo", 1*time.Millisecond)
	c.client.BaseURL, _ = url.Parse(server.URL + "/")

	mux.HandleFunc("/repos/owner/repo/issues", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	_, err := c.CreateIssue(context.Background(), "Title", "Body", nil)
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestAddLabel_Error(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	c, _ := NewClient(context.Background(), "token", "owner/repo", 1*time.Millisecond)
	c.client.BaseURL, _ = url.Parse(server.URL + "/")

	mux.HandleFunc("/repos/owner/repo/issues/1/labels", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	err := c.AddLabel(context.Background(), 1, []string{"label"})
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestGetIssueContent_Error(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	c, _ := NewClient(context.Background(), "token", "owner/repo", 1*time.Millisecond)
	c.client.BaseURL, _ = url.Parse(server.URL + "/")

	mux.HandleFunc("/repos/owner/repo/issues/1", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	_, err := c.GetIssueContent(context.Background(), 1)
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestCommentOnIssue_Error(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	c, _ := NewClient(context.Background(), "token", "owner/repo", 1*time.Millisecond)
	c.client.BaseURL, _ = url.Parse(server.URL + "/")

	mux.HandleFunc("/repos/owner/repo/issues/1/comments", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	err := c.CommentOnIssue(context.Background(), 1, "comment")
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestCloseIssue_Error(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	c, _ := NewClient(context.Background(), "token", "owner/repo", 1*time.Millisecond)
	c.client.BaseURL, _ = url.Parse(server.URL + "/")

	mux.HandleFunc("/repos/owner/repo/issues/1", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	err := c.CloseIssue(context.Background(), 1)
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestMergePR_Error(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	c, _ := NewClient(context.Background(), "token", "owner/repo", 1*time.Millisecond)
	c.client.BaseURL, _ = url.Parse(server.URL + "/")

	mux.HandleFunc("/repos/owner/repo/pulls/1/merge", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	err := c.MergePR(context.Background(), 1)
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestGetPRStatus_Error(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	c, _ := NewClient(context.Background(), "token", "owner/repo", 1*time.Millisecond)
	c.client.BaseURL, _ = url.Parse(server.URL + "/")

	mux.HandleFunc("/repos/owner/repo/commits/sha/status", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	_, err := c.GetPRStatus(context.Background(), "sha")
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestGetPR_Error(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	c, _ := NewClient(context.Background(), "token", "owner/repo", 1*time.Millisecond)
	c.client.BaseURL, _ = url.Parse(server.URL + "/")

	mux.HandleFunc("/repos/owner/repo/pulls/1", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	_, err := c.GetPR(context.Background(), 1)
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestGetFileContent_Error(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	c, _ := NewClient(context.Background(), "token", "owner/repo", 1*time.Millisecond)
	c.client.BaseURL, _ = url.Parse(server.URL + "/")

	mux.HandleFunc("/repos/owner/repo/contents/file", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	_, err := c.GetFileContent(context.Background(), "file")
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestListFiles_Error(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	c, _ := NewClient(context.Background(), "token", "owner/repo", 1*time.Millisecond)
	c.client.BaseURL, _ = url.Parse(server.URL + "/")

	mux.HandleFunc("/repos/owner/repo/contents/dir", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	_, err := c.ListFiles(context.Background(), "dir")
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestNewClient_InvalidRepo(t *testing.T) {
	_, err := NewClient(context.Background(), "token", "invalid", 1*time.Millisecond)
	if err == nil {
		t.Error("expected error for invalid repo format, got nil")
	}
}

func TestSpecHasGapAnalysis_Error(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	c, _ := NewClient(context.Background(), "token", "owner/repo", 1*time.Millisecond)
	c.client.BaseURL, _ = url.Parse(server.URL + "/")

	mux.HandleFunc("/search/issues", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	exists := c.SpecHasGapAnalysis(context.Background(), 1, "label")
	if exists {
		t.Error("expected false on API error")
	}
}

func TestGetIssueContent_NilBody(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	c, _ := NewClient(context.Background(), "token", "owner/repo", 1*time.Millisecond)
	c.client.BaseURL, _ = url.Parse(server.URL + "/")

	mux.HandleFunc("/repos/owner/repo/issues/1", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"number":1, "body":null}`)
	})

	content, err := c.GetIssueContent(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if content != "" {
		t.Errorf("expected empty content for nil body, got %q", content)
	}
}

func TestGetFileContent_Nil(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	c, _ := NewClient(context.Background(), "token", "owner/repo", 1*time.Millisecond)
	c.client.BaseURL, _ = url.Parse(server.URL + "/")

	// Mocking GetContents to return something that results in nil RepositoryContent
	// go-github returns a slice for directories and a single object for files.
	// If it's a directory, the single object return is nil.
	mux.HandleFunc("/repos/owner/repo/contents/path", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[{"name":"file"}]`) // Returns directoy list
	})
}

func TestGetFileContent_NilExplicit(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	c, _ := NewClient(context.Background(), "token", "owner/repo", 1*time.Millisecond)
	c.client.BaseURL, _ = url.Parse(server.URL + "/")

	// This results in nil RepositoryContent because it's a directory
	mux.HandleFunc("/repos/owner/repo/contents/path", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[{"name":"file"}]`)
	})

	_, err := c.GetFileContent(context.Background(), "path")
	if err == nil {
		t.Error("expected error for nil content, got nil")
	}
}

func TestListFiles_WithDirs(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	c, _ := NewClient(context.Background(), "token", "owner/repo", 1*time.Millisecond)
	c.client.BaseURL, _ = url.Parse(server.URL + "/")

	mux.HandleFunc("/repos/owner/repo/contents/dir", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[
			{"name":"file.txt", "type":"file"},
			{"name":"subdir", "type":"dir"}
		]`)
	})

	files, err := c.ListFiles(context.Background(), "dir")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0] != "file.txt" {
		t.Errorf("expected only file.txt, got %v", files)
	}
}

func TestSpecHasGapAnalysis_NilTotal(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	c, _ := NewClient(context.Background(), "token", "owner/repo", 1*time.Millisecond)
	c.client.BaseURL, _ = url.Parse(server.URL + "/")

	mux.HandleFunc("/search/issues", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"total_count": null}`)
	})

	exists := c.SpecHasGapAnalysis(context.Background(), 1, "label")
	if exists {
		t.Error("expected false for nil total_count")
	}
}

func TestListOpenPullRequests_Paging(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	c, _ := NewClient(context.Background(), "token", "owner/repo", 1*time.Millisecond)
	c.client.BaseURL, _ = url.Parse(server.URL + "/")

	count := 0
	mux.HandleFunc("/repos/owner/repo/pulls", func(w http.ResponseWriter, r *http.Request) {
		if count == 0 {
			w.Header().Set("Link", fmt.Sprintf(`<%s/repos/owner/repo/pulls?page=2>; rel="next"`, server.URL))
			fmt.Fprint(w, `[{"number":1}]`)
			count++
		} else {
			fmt.Fprint(w, `[{"number":2}]`)
		}
	})

	prs, err := c.ListOpenPullRequests(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(prs) != 2 {
		t.Errorf("expected 2 PRs, got %d", len(prs))
	}
}

func TestListOpenIssuesByLabel_Paging(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	c, _ := NewClient(context.Background(), "token", "owner/repo", 1*time.Millisecond)
	c.client.BaseURL, _ = url.Parse(server.URL + "/")

	count := 0
	mux.HandleFunc("/repos/owner/repo/issues", func(w http.ResponseWriter, r *http.Request) {
		if count == 0 {
			w.Header().Set("Link", fmt.Sprintf(`<%s/repos/owner/repo/issues?page=2>; rel="next"`, server.URL))
			fmt.Fprint(w, `[{"number":1}]`)
			count++
		} else {
			fmt.Fprint(w, `[{"number":2}]`)
		}
	})

	issues, err := c.ListOpenIssuesByLabel(context.Background(), "label")
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 2 {
		t.Errorf("expected 2 issues, got %d", len(issues))
	}
}

func TestListOpenIssuesByLabel_ErrorPaging(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	c, _ := NewClient(context.Background(), "token", "owner/repo", 1*time.Millisecond)
	c.client.BaseURL, _ = url.Parse(server.URL + "/")

	count := 0
	mux.HandleFunc("/repos/owner/repo/issues", func(w http.ResponseWriter, r *http.Request) {
		if count == 0 {
			w.Header().Set("Link", fmt.Sprintf(`<%s/repos/owner/repo/issues?page=2>; rel="next"`, server.URL))
			fmt.Fprint(w, `[{"number":1}]`)
			count++
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
	})

	_, err := c.ListOpenIssuesByLabel(context.Background(), "label")
	if err == nil {
		t.Error("expected error on second page, got nil")
	}
}
