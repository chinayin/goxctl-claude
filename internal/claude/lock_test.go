package claude

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadLock_NotFound(t *testing.T) {
	_, err := LoadLock(filepath.Join(t.TempDir(), "missing.lock"))
	require.ErrorIs(t, err, ErrLockNotFound)
}

func TestSaveLock_LoadLock_RoundTrip(t *testing.T) {
	// Arrange
	path := filepath.Join(t.TempDir(), LockFile)
	in := &Lock{
		Source:   "github.com/chinayin/goxctl-claude",
		Version:  "v1.1.0",
		Resolved: "9f3a2c1d",
		Managed:  []string{"rules.md", "cli.md"},
		Digest:   "sha256:ab12",
	}

	// Act
	require.NoError(t, SaveLock(path, in))
	out, err := LoadLock(path)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, in, out)
}

func TestComputeDigest_OrderIndependentAndStable(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.md"), []byte("A"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "b.md"), []byte("B"), 0o644))

	// Act：传入顺序不同
	d1, err := ComputeDigest(dir, []string{"a.md", "b.md"})
	require.NoError(t, err)
	d2, err := ComputeDigest(dir, []string{"b.md", "a.md"})
	require.NoError(t, err)

	// Assert：排序后稳定，与顺序无关
	assert.Equal(t, d1, d2)
	assert.True(t, strings.HasPrefix(d1, "sha256:"))
}

func TestComputeDigest_ContentSensitive(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.md"), []byte("A"), 0o644))
	before, err := ComputeDigest(dir, []string{"a.md"})
	require.NoError(t, err)

	// Act：改内容
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.md"), []byte("A-modified"), 0o644))
	after, err := ComputeDigest(dir, []string{"a.md"})
	require.NoError(t, err)

	// Assert
	assert.NotEqual(t, before, after)
}

func TestVerifyDigest(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.md"), []byte("A"), 0o644))
	managed := []string{"a.md"}
	want, err := ComputeDigest(dir, managed)
	require.NoError(t, err)

	// Act & Assert：一致通过
	require.NoError(t, VerifyDigest(dir, managed, want))

	// 篡改后失败
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.md"), []byte("tampered"), 0o644))
	require.ErrorIs(t, VerifyDigest(dir, managed, want), ErrDigestMismatch)
}
