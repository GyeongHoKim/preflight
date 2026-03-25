package diff

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTopLevel(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, exec.Command("git", "init", dir).Run())

	sub := filepath.Join(dir, "pkg", "sub")
	require.NoError(t, os.MkdirAll(sub, 0o755))

	root, err := TopLevel(sub)
	require.NoError(t, err)
	require.Equal(t, dir, root)
}
