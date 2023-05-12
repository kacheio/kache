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

	listener, err := NewListener(context.Background(), &config.EndpointConfig{
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

	ln, err := NewListener(context.Background(), &config.EndpointConfig{
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
