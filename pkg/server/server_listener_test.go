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

package server

import (
	"context"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/kacheio/kache/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewListener(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Test Server"))
	})

	listener, err := NewListener(context.Background(), &config.Listener{
		Addr: ":1337",
	}, handler)
	require.NoError(t, err)

	go listener.Start(context.Background())
	t.Cleanup(func() { listener.Shutdown(context.Background()) })

	resp, err := http.Get("http://localhost:1337")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, "Test Server", string(body))
}

func TestShutdown(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Test Server"))
	})

	ln, err := NewListener(context.Background(), &config.Listener{
		Addr: ":1337",
	}, handler)
	require.NoError(t, err)

	go ln.Start(context.Background())
	t.Cleanup(func() { ln.Shutdown(context.Background()) })

	go ln.Shutdown(context.Background())

	// Ensure new connections are refused after shutdown.
	isClosed := false
	for i := 0; i < 10; i++ {
		conn, err := net.Dial("tcp", ln.listener.Addr().String())
		if err == nil {
			_ = conn.Close()
			time.Sleep(100 * time.Microsecond)
			continue
		}
		if !strings.Contains(err.Error(), "connection refused") && !strings.Contains(err.Error(), "connection reset by peer") {
			t.Fatalf("unexpected error: \ngot:  %v\nwant: %v", err.Error(), "'connection refused' or 'connection reset by peer'")
		}
		isClosed = true
		break
	}
	if !isClosed {
		t.Fatal("Listener failed to shut down.")
	}
}
