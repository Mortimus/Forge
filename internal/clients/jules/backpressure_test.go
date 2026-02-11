package jules

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestBackpressure_Wait(t *testing.T) {
	bp := NewBackpressure(10 * time.Millisecond)

	// 1. Initial wait
	start := time.Now()
	err := bp.Wait(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	elapsed := time.Since(start)
	if elapsed < 10*time.Millisecond {
		t.Errorf("expected wait >= 10ms, got %v", elapsed)
	}

	// 2. Retry-After Header
	bp.HandleResponse(&http.Response{
		StatusCode: http.StatusTooManyRequests,
		Header: http.Header{
			"Retry-After": []string{"1"}, // 1 second
		},
	}, nil)

	start = time.Now()
	err = bp.Wait(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	elapsed = time.Since(start)
	if elapsed < 1*time.Second {
		t.Errorf("expected wait >= 1s, got %v", elapsed)
	}

	// 3. Circuit Breaker
	bp.HandleResponse(nil, http.ErrHandlerTimeout)
	bp.HandleResponse(nil, http.ErrHandlerTimeout)
	bp.HandleResponse(nil, http.ErrHandlerTimeout)

	if !bp.IsBroken() {
		t.Error("expected broken state")
	}

	start = time.Now()
	bp.Wait(context.Background())
	elapsed = time.Since(start)
	if elapsed < 4*time.Second {
		t.Errorf("expected circuit breaker wait, got %v", elapsed)
	}
}
