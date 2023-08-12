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

package cluster

import (
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// Message wraps a HTTP request that gets processed by the queue.
type Message struct {
	Request *http.Request
	Attempt int
}

// RequestQueueOpts holds the request queue settings.
type RequestQueueOpts struct {
	// Size is the size of the request queue.
	Size int

	// MaxWorkers is maximum amount of workers accessing the queue concurrently.
	MaxWorkers int

	// MaxRetries indicate how often a failed request should be retried.
	MaxRetries int

	// Backoff is a simple backoff duration; nothing fancy, no exponential, no jitter.
	Backoff time.Duration
}

// RequestQueue is a simple request queue processing messages with simple retry.
// It is used to broadcast messages in a cluster.
type RequestQueue struct {
	Options RequestQueueOpts

	Queue chan Message

	done chan bool
	wg   sync.WaitGroup
}

// NewRequestQueue creates a simple request queue.
func NewRequestQueue(opts RequestQueueOpts) *RequestQueue {
	q := &RequestQueue{
		Options: opts,
		Queue:   make(chan Message, opts.Size),
		done:    make(chan bool),
	}

	q.wg.Add(opts.MaxWorkers)
	for i := 0; i < opts.MaxWorkers; i++ {
		go q.process()
	}

	return q
}

// Stop stops the request queue.
func (q *RequestQueue) Stop() {
	close(q.done)
	q.wg.Wait()
}

func (q *RequestQueue) process() {
	defer q.wg.Done()

	client := &http.Client{}
	client.Transport = http.DefaultTransport

	for {
		select {
		case msg := <-q.Queue:
			resp, err := client.Do(msg.Request)
			if err != nil {
				log.Error().Err(err).
					Str("url", msg.Request.RequestURI).
					Msg("Error signaling node")
				q.retry(msg)
			}
			if resp.StatusCode >= 400 && resp.StatusCode <= 599 {
				log.Error().Err(err).
					Str("url", resp.Request.RequestURI).Int("status", resp.StatusCode).
					Msg("Error signaling node")
				q.retry(msg)
			}
			if resp != nil {
				// discard and close body to reuse connection.
				_, _ = io.Copy(io.Discard, resp.Body)
				_ = resp.Body.Close()
			}
			log.Debug().Str("url", resp.Request.RequestURI).Int("status", resp.StatusCode).
				Msg("Message sent")

		case <-q.done:
			return
		}
	}
}

func (q *RequestQueue) retry(msg Message) {
	msg.Attempt++
	if msg.Attempt < q.Options.MaxRetries {
		go func() {
			log.Debug().Str("url", msg.Request.RequestURI).Int("attempt", msg.Attempt).
				Msgf("Message retry in %v", q.Options.Backoff)
			time.Sleep(q.Options.Backoff)
			q.Queue <- msg
		}()
	}
}
