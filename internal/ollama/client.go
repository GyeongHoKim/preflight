package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ErrUnavailable wraps transport or HTTP failures for the Ollama client so callers
// can classify fail-open behavior with errors.Is.
var ErrUnavailable = errors.New("ollama: unavailable")

// Client posts JSON to a normalized Ollama base URL (no trailing slash).
type Client struct {
	baseURL string
	http    *http.Client
}

// NewClient returns a client for baseURL (e.g. http://localhost:11434). The URL is
// trimmed of trailing slashes.
func NewClient(baseURL string, timeout time.Duration) (*Client, error) {
	u := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if u == "" {
		return nil, fmt.Errorf("ollama: empty base URL")
	}
	hc := &http.Client{}
	if timeout > 0 {
		hc.Timeout = timeout
	}
	return &Client{
		baseURL: u,
		http:    hc,
	}, nil
}

// Chat performs POST /api/chat with stream disabled and decodes the JSON response.
func (c *Client) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("ollama: nil request")
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("ollama: encode request: %w", err)
	}

	url := c.baseURL + "/api/chat"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("%w: build request: %w", ErrUnavailable, err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		// Propagate explicit context cancellation only; other errors (including
		// dial failures and i/o timeouts) map to ErrUnavailable for fail-open.
		if errors.Is(err, context.Canceled) {
			return nil, err
		}
		return nil, fmt.Errorf("%w: %w", ErrUnavailable, err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: read body: %w", ErrUnavailable, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%w: HTTP %d: %s", ErrUnavailable, resp.StatusCode, bytes.TrimSpace(respBody))
	}

	var out ChatResponse
	if err := json.Unmarshal(respBody, &out); err != nil {
		return nil, fmt.Errorf("%w: decode response: %w", ErrUnavailable, err)
	}
	return &out, nil
}
