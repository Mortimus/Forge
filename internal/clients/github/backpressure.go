package github

import (
	"context"
	"sync"
	"time"

	"github.com/google/go-github/v69/github"
)

// Backpressure handles API rate limiting and circuit breaking for GitHub.
type Backpressure struct {
	mu           sync.RWMutex
	rate         *github.Rate
	resetTime    time.Time
	consecutiveErr int
	brokenUntil  time.Time
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

	// 2. Check Rate Limit
	if b.rate != nil && b.rate.Remaining == 0 {
		if b.rate.Reset.After(now) {
			return b.rate.Reset.Sub(now)
		}
	}

	// 3. Fallback to min interval (jittered if we want)
	// For simplicity, we just use the minInterval
	return b.minInterval
}

// HandleResponse updates the backpressure state based on the API response.
func (b *Backpressure) HandleResponse(resp *github.Response, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if err != nil {
		b.consecutiveErr++
		// If we get 3 errors in a row, break for 5 seconds
		if b.consecutiveErr >= 3 {
			b.brokenUntil = time.Now().Add(5 * time.Second)
		}
		// If it's a RateLimitError, go-github should've set response
	} else {
		b.consecutiveErr = 0
		b.brokenUntil = time.Time{}
	}

	if resp != nil {
		b.rate = &resp.Rate
		// Handle Retry-After if present (RateLimitError might have it)
		// go-github Rate struct already captures Reset time which is usually what GitHub uses.
	}
}

// IsBroken returns true if the circuit breaker is active.
func (b *Backpressure) IsBroken() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.brokenUntil.After(time.Now())
}
