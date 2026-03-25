package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/GyeongHoKim/preflight/internal/config"
	"github.com/GyeongHoKim/preflight/internal/ollama"
	"github.com/GyeongHoKim/preflight/internal/review"
	"github.com/GyeongHoKim/preflight/internal/review/reviewtest"
)

func TestOllamaRunner_Run_FinalJSON(t *testing.T) {
	inner := reviewtest.CanonicalJSON("ok", false, nil, review.VerdictCorrect, 0.9)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := ollama.ChatResponse{
			Model: "m",
			Message: ollama.ChatMessage{
				Role:    "assistant",
				Content: string(inner),
			},
			Done: true,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(srv.Close)

	dir := t.TempDir()
	cfg := &config.Config{
		Provider:     "ollama",
		BlockOn:      "critical",
		Timeout:      10 * time.Second,
		MaxDiffBytes: 1024,
		Ollama: config.OllamaConfig{
			BaseURL:          srv.URL,
			Model:            "m",
			MaxToolTurns:     3,
			MaxReadBytes:     1024,
			MaxListEntries:   10,
			MaxSearchMatches: 10,
		},
	}
	require.NoError(t, config.Validate(cfg))

	r := NewOllamaRunner(cfg, dir, review.SystemPrompt(""), review.Schema())
	out, err := r.Run(context.Background(), []byte("diff --git"))
	require.NoError(t, err)
	assert.JSONEq(t, string(inner), string(out.Stdout))
}

func TestOllamaRunner_Run_Unavailable(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	srv.Close()

	dir := t.TempDir()
	cfg := &config.Config{
		Provider:     "ollama",
		BlockOn:      "critical",
		Timeout:      10 * time.Second,
		MaxDiffBytes: 1024,
		Ollama: config.OllamaConfig{
			BaseURL:          srv.URL,
			Model:            "m",
			MaxToolTurns:     1,
			MaxReadBytes:     1024,
			MaxListEntries:   10,
			MaxSearchMatches: 10,
		},
	}
	require.NoError(t, config.Validate(cfg))

	r := NewOllamaRunner(cfg, dir, review.SystemPrompt(""), review.Schema())
	_, err := r.Run(context.Background(), []byte("diff"))
	require.Error(t, err)
	assert.ErrorIs(t, err, ollama.ErrUnavailable)
}
