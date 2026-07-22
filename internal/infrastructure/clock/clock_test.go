package clock

import (
	"testing"
	"time"
)

func TestNewClock_NowReturnsTheCurrentWallClockTime(t *testing.T) {
	c := NewClock()

	before := time.Now()
	got := c.Now()
	after := time.Now()

	if got.Before(before) || got.After(after) {
		t.Fatalf("Now() = %v, want a time between %v and %v", got, before, after)
	}
}
