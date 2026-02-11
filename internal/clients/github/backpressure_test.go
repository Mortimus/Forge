package github

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/go-github/v69/github"
)

func TestBackpressure_Wait(t *testing.T) {
	bp := NewBackpressure(10 * time.Millisecond)

	// 1. Initial wait (minInterval)
	start := time.Now()
	err := bp.Wait(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	elapsed := time.Since(start)
	if elapsed < 10*time.Millisecond {
		t.Errorf("expected wait >= 10ms, got %v", elapsed)
	}

	// 2. Rate Limit Hit
	bp.HandleResponse(&github.Response{
		Response: &http.Response{Header: http.Header{}},
		Rate: github.Rate{
			Remaining: 0,
			Reset:     github.Timestamp{Time: time.Now().Add(50 * time.Millisecond)},
		},
	}, nil)

	start = time.Now()
	err = bp.Wait(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	elapsed = time.Since(start)
	if elapsed < 50*time.Millisecond {
		t.Errorf("expected wait >= 50ms, got %v", elapsed)
	}

	// 3. Circuit Breaker
	testErr := fmt.Errorf("internal server error")
	bp.HandleResponse(nil, testErr)
	bp.HandleResponse(nil, testErr)
	bp.HandleResponse(nil, testErr)

	if !bp.IsBroken() {
		t.Error("expected circuit breaker to be broken")
	}

	start = time.Now()
	err = bp.Wait(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	elapsed = time.Since(start)
	if elapsed < 4*time.Second { // Should wait for ~5s
		t.Errorf("expected circuit breaker wait, got %v", elapsed)
	}
}
