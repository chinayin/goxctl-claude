package claude

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
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
		"steering/rules.md":  "RULES",
		"steering/cli.md":    "CLI",
		"README.md":          "README",
		"CLAUDE.template.md": "TPL",
	})
	target := t.TempDir()

	// Act
	managed, tmpl, err := extractTarball(tb, []string{"steering/"}, target)

	// Assert：只取 steering/，前缀剥离后展平到 target；CLAUDE 模板内容返回但不落盘
	require.NoError(t, err)
	assert.Equal(t, []string{"cli.md", "rules.md"}, managed)
	assert.Equal(t, "TPL", tmpl)

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

// makeTarballWithEntries 构造一个模拟 GitHub tarball，支持自定义每个条目的 mode。
func makeTarballWithEntries(t *testing.T, top string, entries []tar.Header) io.Reader {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for _, hdr := range entries {
		hdr.Name = top + "/" + hdr.Name
		require.NoError(t, tw.WriteHeader(&hdr))
		if hdr.Size > 0 {
			// 用指定大小填充零字节内容
			_, err := io.Copy(tw, io.LimitReader(strings.NewReader(strings.Repeat("x", int(hdr.Size))), hdr.Size))
			require.NoError(t, err)
		}
	}
	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())
	return &buf
}

func TestExtractTarball_PathTraversalRejected(t *testing.T) {
	// Arrange：归档条目路径含 .. 试图逃逸 target（tar-slip）
	tb := makeTarballWithEntries(t, "repo-1.0.0", []tar.Header{
		{Name: "steering/../../escape.md", Mode: 0o644, Size: 4, Typeflag: tar.TypeReg},
	})
	target := t.TempDir()
	parentDir := filepath.Dir(target)

	// Act
	_, _, err := extractTarball(tb, []string{"steering/"}, target)

	// Assert：返回错误且逃逸文件不存在于 target 父目录
	require.Error(t, err)
	_, statErr := os.Stat(filepath.Join(parentDir, "escape.md"))
	assert.True(t, os.IsNotExist(statErr), "逃逸文件不应被写入 target 父目录")
}

func TestExtractTarball_OversizedFileRejected(t *testing.T) {
	// Arrange：steering/ 下一个超过 maxFileSize 的条目
	oversized := maxFileSize + 1
	tb := makeTarballWithEntries(t, "repo-1.0.0", []tar.Header{
		{Name: "steering/big.md", Mode: 0o644, Size: int64(oversized), Typeflag: tar.TypeReg},
	})
	target := t.TempDir()

	// Act
	_, _, err := extractTarball(tb, []string{"steering/"}, target)

	// Assert：返回提及体积超限的错误
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds")
}

func TestExtractTarball_FileModeForced0644(t *testing.T) {
	// Arrange：归档条目 mode 为 0o777，提取后应被强制为 0o644
	tb := makeTarballWithEntries(t, "repo-1.0.0", []tar.Header{
		{Name: "steering/x.md", Mode: 0o777, Size: 4, Typeflag: tar.TypeReg},
	})
	target := t.TempDir()

	// Act
	_, _, err := extractTarball(tb, []string{"steering/"}, target)

	// Assert：文件权限为 0o644，不继承归档中的 0o777
	require.NoError(t, err)
	fi, err := os.Stat(filepath.Join(target, "x.md"))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o644), fi.Mode().Perm(), "文件权限应强制为 0o644")
}
