package elevenlabs

import (
	"fmt"
	"sync/atomic"
)

// Budget counts characters before every call so a bug can never burn
// the monthly credits, see the trap table in docs/tech-stack.md.

const (
	MaxCallChars  = 2500
	MaxTotalChars = 50000
)

type Budget struct {
	used atomic.Int64
}

// Allow reserves n characters or explains why it cannot.
func (b *Budget) Allow(n int) error {
	if n <= 0 {
		return fmt.Errorf("empty text")
	}
	if n > MaxCallChars {
		return fmt.Errorf("text is %d chars, the per call limit is %d", n, MaxCallChars)
	}
	used := b.used.Add(int64(n))
	if used > MaxTotalChars {
		b.used.Add(int64(-n))
		return fmt.Errorf("budget spent: %d of %d chars used this run", used-int64(n), MaxTotalChars)
	}
	return nil
}
