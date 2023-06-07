// MIT License
//
// Copyright (c) 2023 kache.io
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

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
