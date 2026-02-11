package jules

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// Backpressure handles API rate limiting and circuit breaking for Jules.
type Backpressure struct {
	mu           sync.RWMutex
	consecutiveErr int
	brokenUntil  time.Time
	retryAfter   time.Time
	minInterval  time.Duration
}

// NewBackpressure creates a new Backpressure handler.
func NewBackpressure(minInterval time.Duration) *Backpressure {
	return &Backpressure{
		minInterval: minInterval,
	}
}

// Wait blocks until the next request is allowed.
func (b *Backpressure) Wait(ctx context.Context) error {
	b.mu.RLock()
	waitDuration := b.calculateWait()
	b.mu.RUnlock()

	if waitDuration > 0 {
		timer := time.NewTimer(waitDuration)
		defer timer.Stop()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
		}
	}

	return nil
}

func (b *Backpressure) calculateWait() time.Duration {
	now := time.Now()

	// 1. Check Circuit Breaker
	if b.brokenUntil.After(now) {
		return b.brokenUntil.Sub(now)
	}

	// 2. Check Retry-After
	if b.retryAfter.After(now) {
		return b.retryAfter.Sub(now)
	}

	// 3. Fallback to min interval
	return b.minInterval
}

// HandleResponse updates the backpressure state based on the API response headers.
func (b *Backpressure) HandleResponse(resp *http.Response, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if err != nil {
		b.consecutiveErr++
		if b.consecutiveErr >= 3 {
			b.brokenUntil = time.Now().Add(5 * time.Second)
		}
	} else {
		b.consecutiveErr = 0
		b.brokenUntil = time.Time{}

		if resp.StatusCode == http.StatusTooManyRequests {
			if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
				if seconds, err := strconv.Atoi(retryAfter); err == nil {
					b.retryAfter = time.Now().Add(time.Duration(seconds) * time.Second)
				}
			} else {
				// Default backoff if header missing
				b.retryAfter = time.Now().Add(1 * time.Second)
			}
		}
	}
}

// IsBroken returns true if the circuit breaker is active.
func (b *Backpressure) IsBroken() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.brokenUntil.After(time.Now())
}
