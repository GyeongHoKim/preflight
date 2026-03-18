package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeYAML(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

func TestLoad_Defaults(t *testing.T) {
	cfg, err := Load("", "")
	require.NoError(t, err)
	assert.Equal(t, "auto", cfg.Provider)
	assert.Equal(t, "critical", cfg.BlockOn)
	assert.Equal(t, 60*time.Second, cfg.Timeout)
	assert.Equal(t, 524288, cfg.MaxDiffBytes)
}

func TestLoad_MissingFile(t *testing.T) {
	cfg, err := Load("/nonexistent/path/preflight.yml", "/also/nonexistent.yml")
	require.NoError(t, err)
	assert.Equal(t, "auto", cfg.Provider)
}

func TestLoad_ValidYAML(t *testing.T) {
	dir := t.TempDir()
	path := writeYAML(t, dir, "preflight.yml", `
provider: claude
block_on: warning
timeout: 30s
max_diff_bytes: 1024
`)
	cfg, err := Load(path, "")
	require.NoError(t, err)
	assert.Equal(t, "claude", cfg.Provider)
	assert.Equal(t, "warning", cfg.BlockOn)
	assert.Equal(t, 30*time.Second, cfg.Timeout)
	assert.Equal(t, 1024, cfg.MaxDiffBytes)
}

func TestLoad_ProjectOverridesGlobal(t *testing.T) {
	dir := t.TempDir()
	globalPath := writeYAML(t, dir, "global.yml", `
provider: gemini
block_on: critical
`)
	projectPath := writeYAML(t, dir, "project.yml", `
provider: claude
`)
	cfg, err := Load(projectPath, globalPath)
	require.NoError(t, err)
	assert.Equal(t, "claude", cfg.Provider)
	assert.Equal(t, "critical", cfg.BlockOn)
}

func TestLoad_InvalidProvider(t *testing.T) {
	dir := t.TempDir()
	path := writeYAML(t, dir, "preflight.yml", `provider: invalid`)
	_, err := Load(path, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid provider")
}

func TestLoad_InvalidBlockOn(t *testing.T) {
	dir := t.TempDir()
	path := writeYAML(t, dir, "preflight.yml", `block_on: none`)
	_, err := Load(path, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid block_on")
}
