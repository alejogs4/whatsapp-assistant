package services

import "time"

// Clock abstracts wall-clock time for deterministic unit testing.
type Clock interface {
	Now() time.Time
}

// RealClock returns time.Now().UTC().
type RealClock struct{}

func (RealClock) Now() time.Time { return time.Now().UTC() }
