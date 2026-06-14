package claude

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeGitHub 按 tag 返回 commit sha 与对应内容的 tarball。
// 响应在 handler 外预构造，避免在 http goroutine 内调用 require（testifylint go-require）。
func fakeGitHub(t *testing.T, byTag map[string]map[string]string) *httptest.Server {
	t.Helper()
	responses := map[string][]byte{}
	for tag, files := range byTag {
		responses["/repos/o/r/commits/"+tag] = []byte("sha-" + tag)
		body, err := io.ReadAll(makeTarball(t, "r-"+tag, files))
		require.NoError(t, err)
		responses["/repos/o/r/tarball/"+tag] = body
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := responses[r.URL.Path]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write(body)
	}))
}

func TestSyncer_Add_Check_Remove(t *testing.T) {
	// Arrange
	srv := fakeGitHub(t, map[string]map[string]string{
		"v1.0.0": {"steering/rules.md": "RULES-v1", "steering/cli.md": "CLI"},
	})
	defer srv.Close()
	dir := t.TempDir()
	s := NewSyncer(dir, NewFetcher(WithAPIBase(srv.URL)))
	ctx := context.Background()

	// Act：add
	require.NoError(t, s.Add(ctx, "github.com/o/r", "v1.0.0", []string{"steering/"}, ""))

	// Assert：文件展平落到 .kiro/steering，lock 正确
	got, err := os.ReadFile(filepath.Join(dir, DefaultTarget, "rules.md"))
	require.NoError(t, err)
	assert.Equal(t, "RULES-v1", string(got))

	lock, err := LoadLock(filepath.Join(dir, LockFile))
	require.NoError(t, err)
	assert.Equal(t, "sha-v1.0.0", lock.Resolved)
	assert.Equal(t, []string{"cli.md", "rules.md"}, lock.Managed)

	// Check 通过；篡改后失败
	require.NoError(t, s.Check())
	require.NoError(t, os.WriteFile(filepath.Join(dir, DefaultTarget, "rules.md"), []byte("hand-edited"), 0o600))
	require.ErrorIs(t, s.Check(), ErrDigestMismatch)

	// Remove：受管文件与 manifest/lock 删除
	require.NoError(t, s.Remove())
	_, err = os.Stat(filepath.Join(dir, DefaultTarget, "cli.md"))
	assert.True(t, os.IsNotExist(err))
	_, err = LoadManifest(filepath.Join(dir, ManifestFile))
	require.ErrorIs(t, err, ErrManifestNotFound)
}

func TestSyncer_Update_UpgradesAndCleansStale(t *testing.T) {
	// Arrange：v1 有 rules+cli，v2 删掉 cli、改 rules
	srv := fakeGitHub(t, map[string]map[string]string{
		"v1.0.0": {"steering/rules.md": "RULES-v1", "steering/cli.md": "CLI"},
		"v2.0.0": {"steering/rules.md": "RULES-v2"},
	})
	defer srv.Close()
	dir := t.TempDir()
	s := NewSyncer(dir, NewFetcher(WithAPIBase(srv.URL)))
	ctx := context.Background()
	require.NoError(t, s.Add(ctx, "github.com/o/r", "v1.0.0", []string{"steering/"}, ""))

	// Act：升级到 v2.0.0
	require.NoError(t, s.Update(ctx, "v2.0.0"))

	// Assert：rules 更新、cli 被清理（部分托管：旧受管文件移除）
	got, err := os.ReadFile(filepath.Join(dir, DefaultTarget, "rules.md"))
	require.NoError(t, err)
	assert.Equal(t, "RULES-v2", string(got))
	_, err = os.Stat(filepath.Join(dir, DefaultTarget, "cli.md"))
	assert.True(t, os.IsNotExist(err), "v2 不含 cli.md，应被清理")

	lock, err := LoadLock(filepath.Join(dir, LockFile))
	require.NoError(t, err)
	assert.Equal(t, "v2.0.0", lock.Version)
	assert.Equal(t, []string{"rules.md"}, lock.Managed)
}

func TestSyncer_Update_PreservesProjectOwnedFiles(t *testing.T) {
	// Arrange
	srv := fakeGitHub(t, map[string]map[string]string{
		"v1.0.0": {"steering/rules.md": "RULES"},
		"v2.0.0": {"steering/rules.md": "RULES-v2"},
	})
	defer srv.Close()
	dir := t.TempDir()
	s := NewSyncer(dir, NewFetcher(WithAPIBase(srv.URL)))
	ctx := context.Background()
	require.NoError(t, s.Add(ctx, "github.com/o/r", "v1.0.0", []string{"steering/"}, ""))

	// 项目自有的 steering 文件（非受管）
	own := filepath.Join(dir, DefaultTarget, "project-specific.md")
	require.NoError(t, os.WriteFile(own, []byte("MINE"), 0o600))

	// Act：升级
	require.NoError(t, s.Update(ctx, "v2.0.0"))

	// Assert：项目自有文件原样保留
	got, err := os.ReadFile(own)
	require.NoError(t, err)
	assert.Equal(t, "MINE", string(got))
}
