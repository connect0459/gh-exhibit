// Package clock implements repositories.Clock against the operating
// system's own wall clock.
package clock

import (
	"time"

	"github.com/connect0459/gh-exhibit/internal/domain/repositories"
)

// systemClock implements repositories.Clock via time.Now.
type systemClock struct{}

// NewClock builds a repositories.Clock backed by the operating system's
// wall clock.
func NewClock() repositories.Clock {
	return systemClock{}
}

// Now implements repositories.Clock.
func (systemClock) Now() time.Time {
	return time.Now()
}
