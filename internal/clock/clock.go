package clock

import "time"

type Clock interface {
	Now() time.Time
	Since(time.Time) time.Duration
}

type RealClock struct{}

func NewRealClock() Clock {
	return RealClock{}
}

func (c RealClock) Now() time.Time {
	return time.Now()
}

func (c RealClock) Since(t time.Time) time.Duration {
	return time.Since(t)
}
