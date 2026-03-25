package ollama

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient_EmptyBaseURL(t *testing.T) {
	_, err := NewClient("  ", time.Second)
	require.Error(t, err)
}

func TestClient_Chat_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/chat", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		resp := ChatResponse{
			Model: "m",
			Message: ChatMessage{
				Role:    "assistant",
				Content: `{"ok":true}`,
			},
			Done: true,
		}
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(resp))
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient(srv.URL, 5*time.Second)
	require.NoError(t, err)

	out, err := c.Chat(context.Background(), &ChatRequest{
		Model:    "m",
		Messages: []ChatMessage{{Role: "user", Content: "hi"}},
		Stream:   false,
	})
	require.NoError(t, err)
	assert.Equal(t, `{"ok":true}`, out.Message.Content)
}

func TestClient_Chat_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "nope", http.StatusServiceUnavailable)
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient(srv.URL, 5*time.Second)
	require.NoError(t, err)

	_, err = c.Chat(context.Background(), &ChatRequest{Model: "m", Stream: false})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnavailable)
}

func TestClient_Chat_ContextCancel(t *testing.T) {
	block := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		<-block
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	t.Cleanup(func() { close(block) })

	c, err := NewClient(srv.URL, 60*time.Second)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = c.Chat(ctx, &ChatRequest{Model: "m", Stream: false})
	require.ErrorIs(t, err, context.Canceled)
}
