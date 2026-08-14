package gemini

import (
	"context"
	"testing"
	"time"
)

func TestRateLimiterQueuesInsteadOfDropping(t *testing.T) {
	l := newRateLimiter(2, 150*time.Millisecond)
	ctx := context.Background()

	start := time.Now()
	for range 2 {
		if err := l.wait(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if elapsed := time.Since(start); elapsed > 50*time.Millisecond {
		t.Fatalf("first %d calls should not wait, took %v", 2, elapsed)
	}
	if err := l.wait(ctx); err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(start); elapsed < 100*time.Millisecond {
		t.Fatalf("third call should have queued for a slot, took only %v", elapsed)
	}
}

func TestRateLimiterHonorsCancel(t *testing.T) {
	l := newRateLimiter(1, time.Hour)
	if err := l.wait(context.Background()); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if err := l.wait(ctx); err == nil {
		t.Fatal("blocked wait must fail when the context ends")
	}
}
