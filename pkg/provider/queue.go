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
	"errors"
	"sync"
)

var errJobQueueFull = errors.New("job queue is full")

// jobQueue is the concurrent job queue.
type jobQueue struct {
	// jobCh is the main event channel listening for jobs.
	jobCh chan func()

	// stopCh aborts the processing loop.
	stopCh chan struct{}

	// wg controls the number of consumers/workers.
	wg sync.WaitGroup
}

// newJobQueue creates a new queue and starts the processing loop.
func newJobQueue(size, maxConcurrency int) *jobQueue {
	q := &jobQueue{
		jobCh:  make(chan func(), size),
		stopCh: make(chan struct{}),
	}
	q.wg.Add(maxConcurrency)
	for i := 0; i < maxConcurrency; i++ {
		go q.listen()
	}
	return q
}

// dispatch enqueues a job for execution.
func (q *jobQueue) dispatch(job func()) error {
	select {
	case q.jobCh <- job:
		return nil
	default:
		return errJobQueueFull
	}
}

// stops aborts the processing loop.
func (q *jobQueue) stop() {
	close(q.stopCh)
	q.wg.Wait()
}

// listen starts the job processing loop.
func (q *jobQueue) listen() {
	defer q.wg.Done()
	for {
		select {
		case job := <-q.jobCh:
			job() // execute job.
		case <-q.stopCh:
			return
		}
	}
}
