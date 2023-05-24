package clock

import (
	"sync/atomic"
	"time"
)

type (
	// TimeSource is an interface for any entity that
	// provides the current time. Its primarily used
	// to mock out timesources in unit test.
	TimeSource interface {
		Now() time.Time
		Since(t time.Time) time.Duration
	}

	// SystemTime is the real wall-clock time.
	SystemTime struct{}

	// EventTime is the controlled fake time.
	EventTime struct {
		now int64
	}
)

// NewSystemTimeSource returns a real wall clock time source.
func NewSystemTimeSource() *SystemTime {
	return &SystemTime{}
}

// Now returns the real current time.
func (ts *SystemTime) Now() time.Time {
	return time.Now().UTC()
}

// Since returns the time elapsed since t.
func (ts *SystemTime) Since(t time.Time) time.Duration {
	return time.Since(t)
}

// NewEventTimeSource returns a fake controlled time source.
func NewEventTimeSource() *EventTime {
	return &EventTime{}
}

// Now returns the fake current time.
func (ts *EventTime) Now() time.Time {
	return time.Unix(0, atomic.LoadInt64(&ts.now)).UTC()
}

// Since returns the time elapsed since t.
func (ts *EventTime) Since(t time.Time) time.Duration {
	now := time.Unix(0, atomic.LoadInt64(&ts.now))
	return now.Sub(t)
}

// Update sets the fake current time.
func (ts *EventTime) Update(now time.Time) *EventTime {
	atomic.StoreInt64(&ts.now, now.UnixNano())
	return ts
}
