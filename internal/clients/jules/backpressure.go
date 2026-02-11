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
	lastReqStart time.Time // When the last request started waiting or was sent
}

// NewBackpressure creates a new Backpressure handler.
func NewBackpressure(minInterval time.Duration) *Backpressure {
	return &Backpressure{
		minInterval: minInterval,
	}
}

// Wait blocks until the next request is allowed.
func (b *Backpressure) Wait(ctx context.Context) error {
	b.mu.Lock() // Writer lock because we'll update lastReqStart
	defer b.mu.Unlock()

	waitDuration := b.calculateWait()

	if waitDuration > 0 {
		timer := time.NewTimer(waitDuration)
		defer timer.Stop()

		b.mu.Unlock() // Release while sleeping
		select {
		case <-ctx.Done():
			b.mu.Lock()
			return ctx.Err()
		case <-timer.C:
		}
		b.mu.Lock()
	}

	b.lastReqStart = time.Now()
	return nil
}

func (b *Backpressure) CalculateWait() time.Duration {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.calculateWait()
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

	// 3. Fallback to min interval (Ensure at least minInterval between starts)
	if !b.lastReqStart.IsZero() {
		elapsed := now.Sub(b.lastReqStart)
		if elapsed < b.minInterval {
			return b.minInterval - elapsed
		}
	}

	return 0
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
