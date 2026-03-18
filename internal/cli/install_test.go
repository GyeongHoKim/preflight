package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteHookScript_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	err := WriteHookScript(dir, false)
	require.NoError(t, err)

	hookPath := filepath.Join(dir, "pre-push")
	data, err := os.ReadFile(hookPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "preflight run")

	info, err := os.Stat(hookPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o755), info.Mode().Perm())
}

func TestWriteHookScript_ExistingManaged_Idempotent(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, WriteHookScript(dir, false))
	require.NoError(t, WriteHookScript(dir, false)) // idempotent
}

func TestWriteHookScript_UnmanagedHook_NoForce(t *testing.T) {
	dir := t.TempDir()
	hookPath := filepath.Join(dir, "pre-push")
	require.NoError(t, os.WriteFile(hookPath, []byte("#!/bin/sh\necho hello\n"), 0o755))

	err := WriteHookScript(dir, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "existing hook found")
	assert.Contains(t, err.Error(), "--force")
}

func TestWriteHookScript_UnmanagedHook_WithForce(t *testing.T) {
	dir := t.TempDir()
	hookPath := filepath.Join(dir, "pre-push")
	require.NoError(t, os.WriteFile(hookPath, []byte("#!/bin/sh\necho hello\n"), 0o755))

	err := WriteHookScript(dir, true)
	require.NoError(t, err)

	data, err := os.ReadFile(hookPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "preflight run")
}

func TestIsManagedHook(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, WriteHookScript(dir, false))
	hookPath := filepath.Join(dir, "pre-push")
	assert.True(t, IsManagedHook(hookPath))
}

func TestIsManagedHook_Unmanaged(t *testing.T) {
	dir := t.TempDir()
	hookPath := filepath.Join(dir, "pre-push")
	require.NoError(t, os.WriteFile(hookPath, []byte("#!/bin/sh\necho hello\n"), 0o755))
	assert.False(t, IsManagedHook(hookPath))
}

func TestIsManagedHook_Missing(t *testing.T) {
	assert.False(t, IsManagedHook("/nonexistent/pre-push"))
}
