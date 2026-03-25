package repotools

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecutor_ListReadSearch(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "a.go"), []byte("package main\n"), 0o600))
	sub := filepath.Join(root, "pkg")
	require.NoError(t, os.Mkdir(sub, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(sub, "b.go"), []byte("hello world\n"), 0o600))

	ex := NewExecutor(root, nil, nil, 100, 4096, 50)

	listOut, err := ex.ListFiles(context.Background(), []byte(`{"prefix":"pkg"}`))
	require.NoError(t, err)
	assert.Contains(t, listOut, "pkg/b.go")

	readOut, err := ex.ReadFile([]byte(`{"path":"pkg/b.go"}`))
	require.NoError(t, err)
	assert.Contains(t, readOut, "hello world")

	searchOut, err := ex.SearchRepo(context.Background(), []byte(`{"pattern":"world","path_prefix":"pkg"}`))
	require.NoError(t, err)
	var sr struct {
		Matches []struct {
			Path string `json:"path"`
			Line int    `json:"line"`
		} `json:"matches"`
	}
	require.NoError(t, json.Unmarshal([]byte(searchOut), &sr))
	require.Len(t, sr.Matches, 1)
	assert.Equal(t, "pkg/b.go", sr.Matches[0].Path)
}

func TestExecutor_DispatchUnknown(t *testing.T) {
	ex := NewExecutor(t.TempDir(), nil, nil, 10, 100, 10)
	_, err := ex.Dispatch(context.Background(), "nope", []byte(`{}`))
	require.Error(t, err)
}

func TestExecutor_GitContext(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "a.go"), []byte("package main\n"), 0o600))
	runGit(t, root, "init")
	runGit(t, root, "add", "a.go")
	runGitWithIdentity(t, root, "commit", "-m", "init a.go")

	ex := NewExecutor(root, nil, nil, 10, 100, 10)

	out, err := ex.Dispatch(context.Background(), "git_context", []byte(`{"mode":"log","limit":1}`))
	require.NoError(t, err)
	assert.Contains(t, out, "init a.go")

	out, err = ex.Dispatch(context.Background(), "git_context", []byte(`{"mode":"log_file","path":"a.go","limit":1}`))
	require.NoError(t, err)
	assert.Contains(t, out, "a.go")
	assert.Contains(t, out, "init a.go")
}

func TestExecutor_GitContext_DisallowedMode(t *testing.T) {
	ex := NewExecutor(t.TempDir(), nil, nil, 10, 100, 10)
	out, err := ex.Dispatch(context.Background(), "git_context", []byte(`{"mode":"show"}`))
	require.NoError(t, err)
	assert.Contains(t, out, "mode must be one of")
}

func runGit(t *testing.T, root string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
}

func runGitWithIdentity(t *testing.T, root string, args ...string) {
	t.Helper()
	base := []string{
		"-C", root,
		"-c", "user.name=preflight-test",
		"-c", "user.email=preflight@test.local",
		"-c", "commit.gpgsign=false",
	}
	cmd := exec.Command("git", append(base, args...)...)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
}
