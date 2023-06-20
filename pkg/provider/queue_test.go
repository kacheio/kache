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

package provider

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestJobQueueRun(t *testing.T) {
	q := newJobQueue(10, 10)
	defer q.stop()

	var i atomic.Int32
	_ = q.dispatch(func() { i.Add(1) })
	_ = q.dispatch(func() { i.Add(-1) })
	_ = q.dispatch(func() { i.Add(1) })
	_ = q.dispatch(func() { i.Add(-1) })
	_ = q.dispatch(func() { i.Add(1) })

	// Wait for all operations to finish.
	time.Sleep(100 * time.Millisecond)

	require.Equal(t, int32(1), i.Load())
}

func TestJobQueueError(t *testing.T) {
	const queueSize = 10

	q := newJobQueue(queueSize, 1)
	defer q.stop()

	doneCh := make(chan struct{})
	defer func() {
		close(doneCh)
	}()

	// Keep workers busy.
	_ = q.dispatch(func() {
		<-doneCh
	})
	time.Sleep(100 * time.Millisecond)

	// Enqueue jobs until full.
	for i := 0; i < queueSize; i++ {
		require.NoError(t, q.dispatch(func() {}))
	}

	// Overflow queue.
	require.Equal(t, errJobQueueFull, q.dispatch(func() {}))
}
