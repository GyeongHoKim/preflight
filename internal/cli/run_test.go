package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildRootCmd(t *testing.T) {
	root := buildRootCmd()
	require.NotNil(t, root)
	assert.Equal(t, "preflight", root.Use)
}

func TestBuildRootCmd_HasSubcommands(t *testing.T) {
	root := buildRootCmd()
	names := make(map[string]bool)
	for _, cmd := range root.Commands() {
		names[cmd.Use] = true
	}
	assert.True(t, names["run"])
	assert.True(t, names["install"])
	assert.True(t, names["uninstall"])
	assert.True(t, names["version"])
}

func TestLoadConfig_ValidDefaults(t *testing.T) {
	// Reset global state before the test.
	prevProvider := globalFlags.provider
	prevConfig := globalFlags.configPath
	defer func() {
		globalFlags.provider = prevProvider
		globalFlags.configPath = prevConfig
	}()

	globalFlags.provider = ""
	globalFlags.configPath = "/nonexistent.yml"

	root := buildRootCmd()
	root.SetArgs([]string{"version"})
	err := root.Execute()
	// With a nonexistent config and no --provider, loadConfig should use defaults.
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, "auto", cfg.Provider)
}
