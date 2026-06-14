package claude

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// makeTarball 构造一个模拟 GitHub tarball（含顶层包裹目录）。
func makeTarball(t *testing.T, top string, files map[string]string) io.Reader {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, content := range files {
		full := top + "/" + name
		require.NoError(t, tw.WriteHeader(&tar.Header{
			Name: full, Mode: 0o644, Size: int64(len(content)), Typeflag: tar.TypeReg,
		}))
		_, err := tw.Write([]byte(content))
		require.NoError(t, err)
	}
	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())
	return &buf
}

func TestExtractTarball_FlattensPathsAndFilters(t *testing.T) {
	// Arrange：steering/ 下两个文件 + 一个仓库根文件（不应被取）
	tb := makeTarball(t, "goxctl-claude-1.0.0", map[string]string{
		"steering/rules.md": "RULES",
		"steering/cli.md":   "CLI",
		"README.md":         "README",
	})
	target := t.TempDir()

	// Act
	managed, err := extractTarball(tb, []string{"steering/"}, target)

	// Assert：只取 steering/，前缀剥离后展平到 target
	require.NoError(t, err)
	assert.Equal(t, []string{"cli.md", "rules.md"}, managed)

	got, err := os.ReadFile(filepath.Join(target, "rules.md"))
	require.NoError(t, err)
	assert.Equal(t, "RULES", string(got))

	_, err = os.Stat(filepath.Join(target, "README.md"))
	assert.True(t, os.IsNotExist(err), "仓库根文件不应被同步")
}

func TestStripTopDir(t *testing.T) {
	assert.Equal(t, "steering/rules.md", stripTopDir("goxctl-claude-1.0.0/steering/rules.md"))
	assert.Equal(t, "steering/rules.md", stripTopDir("./goxctl-claude-1.0.0/steering/rules.md"))
	assert.Empty(t, stripTopDir("toplevelonly"))
}
