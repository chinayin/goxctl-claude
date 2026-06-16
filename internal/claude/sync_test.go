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
func fakeGitHub(t *testing.T, latestTag string, byTag map[string]map[string]string) *httptest.Server {
	t.Helper()
	responses := map[string][]byte{}
	if latestTag != "" {
		responses["/repos/o/r/releases/latest"] = []byte(`{"tag_name":"` + latestTag + `"}`)
	}
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
	srv := fakeGitHub(t, "", map[string]map[string]string{
		"v1.0.0": {"steering/rules.md": "RULES-v1", "steering/cli.md": "CLI"},
	})
	defer srv.Close()
	dir := t.TempDir()
	s := NewSyncer(dir, NewFetcher(WithAPIBase(srv.URL)))
	ctx := context.Background()

	// Act：add
	_, err := s.Add(ctx, "github.com/o/r", "v1.0.0", []string{"steering/"}, "")
	require.NoError(t, err)

	// Assert：文件展平落到 .kiro/steering，lock 正确
	got, err := os.ReadFile(filepath.Join(dir, DefaultTarget, "rules.md"))
	require.NoError(t, err)
	assert.Equal(t, "RULES-v1", string(got))

	lock, err := LoadLock(filepath.Join(dir, LockFile))
	require.NoError(t, err)
	assert.Equal(t, "sha-v1.0.0", lock.Commit)
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

func TestSyncer_Add_NoVersion_UsesLatest(t *testing.T) {
	srv := fakeGitHub(t, "v2.0.0", map[string]map[string]string{
		"v2.0.0": {"steering/rules.md": "RULES-v2"},
	})
	defer srv.Close()
	dir := t.TempDir()
	s := NewSyncer(dir, NewFetcher(WithAPIBase(srv.URL)))

	// 不传 version → 解析最新 release 并钉住具体 tag
	_, err := s.Add(context.Background(), "github.com/o/r", "", []string{"steering/"}, "")
	require.NoError(t, err)

	m, _, err := s.Status()
	require.NoError(t, err)
	assert.Equal(t, "v2.0.0", m.Version)
	lock, err := LoadLock(filepath.Join(dir, LockFile))
	require.NoError(t, err)
	assert.Equal(t, "sha-v2.0.0", lock.Commit)
}

func TestSyncer_Update_UpgradesAndCleansStale(t *testing.T) {
	// Arrange：v1 有 rules+cli，v2 删掉 cli、改 rules
	srv := fakeGitHub(t, "", map[string]map[string]string{
		"v1.0.0": {"steering/rules.md": "RULES-v1", "steering/cli.md": "CLI"},
		"v2.0.0": {"steering/rules.md": "RULES-v2"},
	})
	defer srv.Close()
	dir := t.TempDir()
	s := NewSyncer(dir, NewFetcher(WithAPIBase(srv.URL)))
	ctx := context.Background()
	_, err := s.Add(ctx, "github.com/o/r", "v1.0.0", []string{"steering/"}, "")
	require.NoError(t, err)

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
	srv := fakeGitHub(t, "", map[string]map[string]string{
		"v1.0.0": {"steering/rules.md": "RULES"},
		"v2.0.0": {"steering/rules.md": "RULES-v2"},
	})
	defer srv.Close()
	dir := t.TempDir()
	s := NewSyncer(dir, NewFetcher(WithAPIBase(srv.URL)))
	ctx := context.Background()
	_, err := s.Add(ctx, "github.com/o/r", "v1.0.0", []string{"steering/"}, "")
	require.NoError(t, err)

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

func TestSyncer_Add_GeneratesEntrypoint(t *testing.T) {
	// tarball 含仓库根 CLAUDE.template.md，随规范一起拉取并生成项目 CLAUDE.md
	srv := fakeGitHub(t, "", map[string]map[string]string{
		"v1.0.0": {"steering/rules.md": "R", "CLAUDE.template.md": "ENTRY @import"},
	})
	defer srv.Close()
	dir := t.TempDir()
	s := NewSyncer(dir, NewFetcher(WithAPIBase(srv.URL)))

	created, err := s.Add(context.Background(), "github.com/o/r", "v1.0.0", []string{"steering/"}, "")
	require.NoError(t, err)
	assert.True(t, created)

	b, err := os.ReadFile(filepath.Join(dir, ClaudeMdFile))
	require.NoError(t, err)
	assert.Equal(t, "ENTRY @import", string(b))

	// CLAUDE.md 不纳入受管
	lock, err := LoadLock(filepath.Join(dir, LockFile))
	require.NoError(t, err)
	assert.Equal(t, []string{"rules.md"}, lock.Managed)
}

func TestSyncer_Update_NoArg_UsesLatest(t *testing.T) {
	srv := fakeGitHub(t, "v2.0.0", map[string]map[string]string{
		"v1.0.0": {"steering/rules.md": "R1"},
		"v2.0.0": {"steering/rules.md": "R2"},
	})
	defer srv.Close()
	dir := t.TempDir()
	s := NewSyncer(dir, NewFetcher(WithAPIBase(srv.URL)))
	ctx := context.Background()
	_, err := s.Add(ctx, "github.com/o/r", "v1.0.0", []string{"steering/"}, "")
	require.NoError(t, err)

	// update 无参 → 升级到最新（v2.0.0）
	require.NoError(t, s.Update(ctx, ""))

	m, _, err := s.Status()
	require.NoError(t, err)
	assert.Equal(t, "v2.0.0", m.Version)
	got, err := os.ReadFile(filepath.Join(dir, DefaultTarget, "rules.md"))
	require.NoError(t, err)
	assert.Equal(t, "R2", string(got))
}

// fakeGitHubTarballFails 解析 sha 正常，但 tarball 下载返回 500。
func fakeGitHubTarballFails(t *testing.T, tag string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/o/r/commits/" + tag:
			_, _ = w.Write([]byte("sha-" + tag))
		default: // tarball 及其它
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
}

func TestSyncer_Add_FailedPull_IsRetryable(t *testing.T) {
	// Arrange：tarball 下载失败的 server
	failSrv := fakeGitHubTarballFails(t, "v1.0.0")
	defer failSrv.Close()
	dir := t.TempDir()
	s := NewSyncer(dir, NewFetcher(WithAPIBase(failSrv.URL)))
	ctx := context.Background()

	// Act：add 应返回错误
	_, err := s.Add(ctx, "github.com/o/r", "v1.0.0", []string{"steering/"}, "")
	require.Error(t, err)

	// Assert：manifest 不应写入磁盘（失败后可直接重试 add）
	_, loadErr := LoadManifest(filepath.Join(dir, ManifestFile))
	require.ErrorIs(t, loadErr, ErrManifestNotFound, "pull 失败后 manifest 不应存在")

	// Act：换用正常 server，重试 add 应成功
	okSrv := fakeGitHub(t, "", map[string]map[string]string{
		"v1.0.0": {"steering/rules.md": "RULES-v1", "steering/cli.md": "CLI"},
	})
	defer okSrv.Close()
	s2 := NewSyncer(dir, NewFetcher(WithAPIBase(okSrv.URL)))
	_, err = s2.Add(ctx, "github.com/o/r", "v1.0.0", []string{"steering/"}, "")
	require.NoError(t, err, "重试 add 应成功")

	got, err := os.ReadFile(filepath.Join(dir, DefaultTarget, "rules.md"))
	require.NoError(t, err)
	assert.Equal(t, "RULES-v1", string(got))
}

func TestSyncer_Update_FailedDownload_KeepsFiles(t *testing.T) {
	// Arrange：先用正常 server 做 v1.0.0 的 add
	okSrv := fakeGitHub(t, "", map[string]map[string]string{
		"v1.0.0": {"steering/rules.md": "RULES-v1", "steering/cli.md": "CLI"},
	})
	defer okSrv.Close()
	dir := t.TempDir()
	s := NewSyncer(dir, NewFetcher(WithAPIBase(okSrv.URL)))
	ctx := context.Background()
	_, err := s.Add(ctx, "github.com/o/r", "v1.0.0", []string{"steering/"}, "")
	require.NoError(t, err)

	// 确认 v1 文件已落盘
	require.FileExists(t, filepath.Join(dir, DefaultTarget, "rules.md"))
	require.FileExists(t, filepath.Join(dir, DefaultTarget, "cli.md"))

	// Act：用下载失败的 server 尝试升级到 v2.0.0
	failSrv := fakeGitHubTarballFails(t, "v2.0.0")
	defer failSrv.Close()
	s2 := NewSyncer(dir, NewFetcher(WithAPIBase(failSrv.URL)))
	err = s2.Update(ctx, "v2.0.0")
	require.Error(t, err)

	// Assert：v1 的文件仍然存在，未被清理
	gotRules, err := os.ReadFile(filepath.Join(dir, DefaultTarget, "rules.md"))
	require.NoError(t, err)
	assert.Equal(t, "RULES-v1", string(gotRules), "下载失败后 rules.md 应保持 v1 内容")

	_, err = os.Stat(filepath.Join(dir, DefaultTarget, "cli.md"))
	assert.False(t, os.IsNotExist(err), "下载失败后 cli.md 应仍然存在")

	// Assert：Check 对 v1 lock 仍然通过（文件未被篡改）
	sCheck := NewSyncer(dir, NewFetcher(WithAPIBase(okSrv.URL)))
	require.NoError(t, sCheck.Check(), "v1 lock 的 Check 应通过")
}
