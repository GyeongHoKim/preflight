package provider

import (
	"context"
	"fmt"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/GyeongHoKim/preflight/internal/ollama"
)

func TestShouldFailOpen(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"ErrProviderNotFound", ErrProviderNotFound, true},
		{"DeadlineExceeded", context.DeadlineExceeded, true},
		{"ExitError", &exec.ExitError{}, true},
		{"Ollama unavailable", fmt.Errorf("wrap: %w", ollama.ErrUnavailable), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ShouldFailOpen(tt.err))
		})
	}
}

func TestDetect_NoneFound(t *testing.T) {
	_, err := Detect([]string{"__nonexistent_binary_xyz__"})
	require.ErrorIs(t, err, ErrProviderNotFound)
}

func TestDetect_Found(t *testing.T) {
	// "sh" is available on all POSIX systems.
	name, err := Detect([]string{"__nonexistent__", "sh"})
	require.NoError(t, err)
	assert.Equal(t, "sh", name)
}

func TestMockRunner(t *testing.T) {
	mock := &MockRunner{Err: ErrProviderNotFound}
	_, err := mock.Run(context.Background(), []byte("diff"))
	require.ErrorIs(t, err, ErrProviderNotFound)
}

func TestDetect_ExplicitNotFound(t *testing.T) {
	_, err := Detect([]string{"__definitely_not_a_real_binary__"})
	assert.ErrorIs(t, err, ErrProviderNotFound)
}

func TestShouldFailOpen_NilIsNotFailOpen(t *testing.T) {
	assert.False(t, ShouldFailOpen(nil))
}
