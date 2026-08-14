package gemini

import (
	"context"
	"sync"
	"time"
)

// Sliding window limiter. Callers over the limit wait for a slot,
// nothing is ever dropped, see docs/tech-stack.md.
type rateLimiter struct {
	mu     sync.Mutex
	limit  int
	window time.Duration
	starts []time.Time
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	return &rateLimiter{limit: limit, window: window}
}

// wait blocks until the caller may start a request or the context ends.
func (l *rateLimiter) wait(ctx context.Context) error {
	for {
		l.mu.Lock()
		now := time.Now()
		keep := l.starts[:0]
		for _, t := range l.starts {
			if now.Sub(t) < l.window {
				keep = append(keep, t)
			}
		}
		l.starts = keep
		if len(l.starts) < l.limit {
			l.starts = append(l.starts, now)
			l.mu.Unlock()
			return nil
		}
		wakeAt := l.starts[0].Add(l.window)
		l.mu.Unlock()

		timer := time.NewTimer(time.Until(wakeAt))
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}
