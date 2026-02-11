package jules

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestListSources(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1alpha/sources" {
			t.Errorf("Path = %v, want /v1alpha/sources", r.URL.Path)
		}
		if r.Header.Get("X-Goog-Api-Key") != "key" {
			t.Errorf("ApiKey header missing or invalid")
		}
		fmt.Fprint(w, `{"sources":[{"name":"source1", "id":"1"}]}`)
	}))
	defer server.Close()

	c := NewClient("key", 1*time.Millisecond)
	c.BaseURL = server.URL

	sources, err := c.ListSources(context.Background())
	if err != nil {
		t.Fatalf("ListSources failed: %v", err)
	}
	if len(sources) != 1 {
		t.Errorf("want 1 source, got %d", len(sources))
	}
	if sources[0].Name != "source1" {
		t.Errorf("want source1, got %s", sources[0].Name)
	}
}

func TestCreateSession(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Method = %v, want POST", r.Method)
		}
		if r.URL.Path != "/v1alpha/sessions" {
			t.Errorf("Path = %v, want /v1alpha/sessions", r.URL.Path)
		}
		fmt.Fprint(w, `{"name":"sessions/123"}`)
	}))
	defer server.Close()

	c := NewClient("key", 1*time.Millisecond)
	c.BaseURL = server.URL

	sess, err := c.CreateSession(context.Background(), "title", "prompt", "source", "main")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}
	if sess.Name != "sessions/123" {
		t.Errorf("want sessions/123, got %s", sess.Name)
	}
}

func TestGetSession(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1alpha/sessions/123" {
			t.Errorf("Path = %v, want /v1alpha/sessions/123", r.URL.Path)
		}
		fmt.Fprint(w, `{"name":"sessions/123", "state":"RUNNING"}`)
	}))
	defer server.Close()

	c := NewClient("key", 1*time.Millisecond)
	c.BaseURL = server.URL

	sess, err := c.GetSession(context.Background(), "sessions/123")
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}
	if sess.State != "RUNNING" {
		t.Errorf("want RUNNING, got %s", sess.State)
	}
}

func TestListSources_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	c := NewClient("key", 1*time.Millisecond)
	c.BaseURL = server.URL

	_, err := c.ListSources(context.Background())
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestCreateSession_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	c := NewClient("key", 1*time.Millisecond)
	c.BaseURL = server.URL

	_, err := c.CreateSession(context.Background(), "t", "p", "s", "b")
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestGetSession_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	c := NewClient("key", 1*time.Millisecond)
	c.BaseURL = server.URL

	_, err := c.GetSession(context.Background(), "s")
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestGetSession_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	c := NewClient("key", 1*time.Millisecond)
	c.BaseURL = server.URL

	_, err := c.GetSession(context.Background(), "s")
	if err == nil {
		t.Error("expected error for HTTP 500, got nil")
	}
}

func TestGetSession_MalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"status": invalid}`)
	}))
	defer server.Close()

	c := NewClient("key", 1*time.Millisecond)
	c.BaseURL = server.URL

	_, err := c.GetSession(context.Background(), "s")
	if err == nil {
		t.Error("expected error for malformed JSON, got nil")
	}
}

func TestCreateSession_MalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"name": invalid}`)
	}))
	defer server.Close()

	c := NewClient("key", 1*time.Millisecond)
	c.BaseURL = server.URL

	_, err := c.CreateSession(context.Background(), "t", "p", "s", "b")
	if err == nil {
		t.Error("expected error for malformed JSON, got nil")
	}
}

func TestListSources_MalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"sources": [invalid]}`)
	}))
	defer server.Close()

	c := NewClient("key", 1*time.Millisecond)
	c.BaseURL = server.URL

	_, err := c.ListSources(context.Background())
	if err == nil {
		t.Error("expected error for malformed JSON, got nil")
	}
}

func TestCreateSession_RequestError(t *testing.T) {
	c := NewClient("key", 1*time.Millisecond)
	c.BaseURL = "http://invalid-url"
	_, err := c.CreateSession(context.Background(), "t", "p", "s", "b")
	if err == nil {
		t.Error("expected error for invalid URL, got nil")
	}
}

func TestGetSession_RequestError(t *testing.T) {
	c := NewClient("key", 1*time.Millisecond)
	c.BaseURL = "http://invalid-url"
	_, err := c.GetSession(context.Background(), "s")
	if err == nil {
		t.Error("expected error for invalid URL, got nil")
	}
}
func TestDeleteSession(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	c := NewClient("test-api-key", 1*time.Millisecond)
	c.BaseURL = server.URL

	sessionName := "sessions/123"

	mux.HandleFunc("/v1alpha/sessions/123", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("Method = %v, want DELETE", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	err := c.DeleteSession(context.Background(), sessionName)
	if err != nil {
		t.Fatalf("DeleteSession returned error: %v", err)
	}
}

func TestBackpressure_SmartRateLimit(t *testing.T) {
	interval := 100 * time.Millisecond
	bp := NewBackpressure(interval)

	// First request should not wait
	start := time.Now()
	err := bp.Wait(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	wait1 := time.Since(start)
	if wait1 > 50*time.Millisecond {
		t.Errorf("First Wait() took %v, want < 50ms", wait1)
	}

	// Second request should wait
	start = time.Now()
	err = bp.Wait(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	wait2 := time.Since(start)
	if wait2 < interval {
		t.Errorf("Second Wait() took %v, want >= %v", wait2, interval)
	}
}

func TestClient_RetryOn429(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	c := NewClient("test-api-key", 1*time.Millisecond)
	c.BaseURL = server.URL

	attempts := 0
	mux.HandleFunc("/v1alpha/sessions", func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts <= 2 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Write([]byte(`{"sessions": []}`))
	})

	_, err := c.ListSessions(context.Background())
	if err != nil {
		t.Fatalf("ListSessions failed: %v", err)
	}

	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}
