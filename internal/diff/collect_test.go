package diff

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePushInfo_Normal(t *testing.T) {
	input := "refs/heads/main abc123 refs/heads/main def456\n"
	infos, err := ParsePushInfo(strings.NewReader(input))
	require.NoError(t, err)
	require.Len(t, infos, 1)
	assert.Equal(t, "refs/heads/main", infos[0].LocalRef)
	assert.Equal(t, "abc123", infos[0].LocalSHA)
	assert.Equal(t, "refs/heads/main", infos[0].RemoteRef)
	assert.Equal(t, "def456", infos[0].RemoteSHA)
	assert.False(t, infos[0].IsNewBranch())
	assert.False(t, infos[0].IsDeletePush())
}

func TestParsePushInfo_NewBranch(t *testing.T) {
	input := "refs/heads/feature abc123 refs/heads/feature " + zeroSHA + "\n"
	infos, err := ParsePushInfo(strings.NewReader(input))
	require.NoError(t, err)
	require.Len(t, infos, 1)
	assert.True(t, infos[0].IsNewBranch())
	assert.False(t, infos[0].IsDeletePush())
}

func TestParsePushInfo_DeletePush(t *testing.T) {
	input := "refs/heads/feature " + zeroSHA + " refs/heads/feature def456\n"
	infos, err := ParsePushInfo(strings.NewReader(input))
	require.NoError(t, err)
	require.Len(t, infos, 1)
	assert.False(t, infos[0].IsNewBranch())
	assert.True(t, infos[0].IsDeletePush())
}

func TestParsePushInfo_MultiRef(t *testing.T) {
	input := "refs/heads/main a1 refs/heads/main b1\nrefs/heads/dev a2 refs/heads/dev b2\n"
	infos, err := ParsePushInfo(strings.NewReader(input))
	require.NoError(t, err)
	require.Len(t, infos, 2)
}

func TestParsePushInfo_BlankLines(t *testing.T) {
	input := "\nrefs/heads/main abc123 refs/heads/main def456\n\n"
	infos, err := ParsePushInfo(strings.NewReader(input))
	require.NoError(t, err)
	require.Len(t, infos, 1)
}

func TestParsePushInfo_InvalidLine(t *testing.T) {
	input := "only-two fields\n"
	_, err := ParsePushInfo(strings.NewReader(input))
	require.Error(t, err)
}

func TestMergeBaseWith_NoValidCandidates(t *testing.T) {
	// Non-existent refs must all fail, returning an empty string (triggers tip-commit fallback).
	result := mergeBaseWith(context.Background(), "HEAD", []string{"origin/nonexistent-ref-xyzzy"})
	assert.Equal(t, "", result)
}

func TestMergeBaseWith_EmptyCandidates(t *testing.T) {
	result := mergeBaseWith(context.Background(), "HEAD", nil)
	assert.Equal(t, "", result)
}

func TestIsGitRepo(t *testing.T) {
	// Current working directory inside the test is the package dir which is inside a git repo.
	assert.True(t, IsGitRepo("."))
	assert.False(t, IsGitRepo("/tmp"))
}
