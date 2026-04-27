package clock

import "time"

// Clock provides UTC time operations.
// This abstraction enables testing with fixed times.
type Clock interface {
	Now() time.Time
}

// RealClock returns the current UTC time.
type RealClock struct{}

func New() Clock {
	return &RealClock{}
}

func (c *RealClock) Now() time.Time {
	return time.Now().UTC()
}

// MockClock is used for testing with fixed times.
type MockClock struct {
	FixedTime time.Time
}

func NewMock(t time.Time) *MockClock {
	return &MockClock{FixedTime: t.UTC()}
}

func (c *MockClock) Now() time.Time {
	return c.FixedTime
}

func (c *MockClock) Advance(d time.Duration) {
	c.FixedTime = c.FixedTime.Add(d)
}

func (c *MockClock) Set(t time.Time) {
	c.FixedTime = t.UTC()
}
