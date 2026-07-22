package repositories

import "time"

// Clock abstracts the wall clock so the application layer's dependency on
// "the current time" stays testable — production code is wired to the real
// clock, tests inject a fixed one.
type Clock interface {
	// Now returns the current time.
	Now() time.Time
}
