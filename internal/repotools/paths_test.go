package repotools

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPathRules_ResolveReadable(t *testing.T) {
	root := t.TempDir()
	rules := PathRules{
		RepoRoot:     root,
		DenyPatterns: []string{".env*"},
	}

	abs, rel, err := rules.ResolveReadable("foo.go")
	require.NoError(t, err)
	assert.Equal(t, "foo.go", rel)
	assert.Equal(t, filepath.Join(root, "foo.go"), abs)

	_, _, err = rules.ResolveReadable("../outside")
	require.ErrorIs(t, err, ErrOutsideRepo)

	_, _, err = rules.ResolveReadable(".git/config")
	require.ErrorIs(t, err, ErrDeniedPath)
}

func TestPathRules_AllowPrefixes(t *testing.T) {
	root := t.TempDir()
	rules := PathRules{
		RepoRoot:      root,
		AllowPrefixes: []string{"internal/"},
	}

	_, _, err := rules.ResolveReadable("internal/x.go")
	require.NoError(t, err)

	_, _, err = rules.ResolveReadable("cmd/main.go")
	require.ErrorIs(t, err, ErrDeniedPath)
}
