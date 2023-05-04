package server

import (
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewListener(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Test Server"))
	})

	config := &EndpointConfig{":1331"}
	ctx := context.TODO()

	listener, _ := NewListener(ctx, config, handler)
	go listener.Start(ctx)

	client := &http.Client{}
	resp, err := client.Get("http://localhost:1331")
	if err != nil {
		t.Errorf("Error: %v", err)
	}
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, "Test Server", string(body))
}
