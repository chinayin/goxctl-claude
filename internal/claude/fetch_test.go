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

func TestFetcher_ResolveTag(t *testing.T) {
	// Arrange
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/repos/chinayin/goxctl-claude/commits/v1.0.0", r.URL.Path)
		assert.Equal(t, "application/vnd.github.sha", r.Header.Get("Accept"))
		_, _ = w.Write([]byte("9f3a2c1ddeadbeef"))
	}))
	defer srv.Close()
	f := NewFetcher(WithAPIBase(srv.URL))

	// Act
	sha, err := f.ResolveTag(context.Background(), RepoRef{"chinayin", "goxctl-claude"}, "v1.0.0")

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "9f3a2c1ddeadbeef", sha)
}

func TestFetcher_ResolveTag_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	f := NewFetcher(WithAPIBase(srv.URL))

	_, err := f.ResolveTag(context.Background(), RepoRef{"chinayin", "goxctl-claude"}, "v9.9.9")
	require.Error(t, err)
}

func TestFetcher_ResolveLatest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/repos/chinayin/goxctl-claude/releases/latest", r.URL.Path)
		_, _ = w.Write([]byte(`{"tag_name":"v3.1.0"}`))
	}))
	defer srv.Close()
	f := NewFetcher(WithAPIBase(srv.URL))

	tag, err := f.ResolveLatest(context.Background(), RepoRef{"chinayin", "goxctl-claude"})
	require.NoError(t, err)
	assert.Equal(t, "v3.1.0", tag)
}

func TestFetcher_ResolveLatest_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	f := NewFetcher(WithAPIBase(srv.URL))

	_, err := f.ResolveLatest(context.Background(), RepoRef{"chinayin", "goxctl-claude"})
	require.Error(t, err)
}

func TestFetcher_DownloadTarball_ThenExtract(t *testing.T) {
	// Arrange：mock tarball 端点返回模拟 GitHub tarball
	body, err := io.ReadAll(makeTarball(t, "goxctl-claude-1.0.0", map[string]string{
		"steering/rules.md": "RULES",
		"README.md":         "README",
	}))
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/repos/chinayin/goxctl-claude/tarball/v1.0.0", r.URL.Path)
		_, _ = w.Write(body)
	}))
	defer srv.Close()
	f := NewFetcher(WithAPIBase(srv.URL))

	// Act：下载 → 解压
	rc, err := f.DownloadTarball(context.Background(), RepoRef{"chinayin", "goxctl-claude"}, "v1.0.0")
	require.NoError(t, err)
	defer rc.Close()

	target := t.TempDir()
	managed, _, err := extractTarball(rc, []string{"steering/"}, target)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, []string{"rules.md"}, managed)
	got, err := os.ReadFile(filepath.Join(target, "rules.md"))
	require.NoError(t, err)
	assert.Equal(t, "RULES", string(got))
}

func TestFetcher_WithToken_SetsAuthHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer secret-token", r.Header.Get("Authorization"))
		_, _ = w.Write([]byte("sha"))
	}))
	defer srv.Close()
	f := NewFetcher(WithAPIBase(srv.URL), WithToken("secret-token"))

	_, err := f.ResolveTag(context.Background(), RepoRef{"o", "r"}, "v1")
	require.NoError(t, err)
}
