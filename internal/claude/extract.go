package claude

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// extractTarball 从 GitHub tarball 流中提取匹配 paths 的文件到 target。
//
// GitHub tarball 顶层有一个 "{repo}-{ref}/" 包裹目录，先剥离；paths 中的目录前缀
// （如 "steering/"）也剥离，内容展平到 target 下。返回写入的受管文件相对 target 的路径，已排序。
func extractTarball(r io.Reader, paths []string, target string) ([]string, error) {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("claude: open gzip: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	var managed []string
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("claude: read tar: %w", err)
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}

		rel := stripTopDir(hdr.Name)
		sub, ok := matchPath(rel, paths)
		if !ok {
			continue
		}

		if err := writeFile(filepath.Join(target, sub), tr, hdr.FileInfo().Mode().Perm()); err != nil {
			return nil, err
		}
		managed = append(managed, sub)
	}

	slices.Sort(managed)
	return managed, nil
}

// stripTopDir 去掉 GitHub tarball 的顶层包裹目录（第一段路径）。
func stripTopDir(name string) string {
	name = strings.TrimPrefix(name, "./")
	if i := strings.IndexByte(name, '/'); i >= 0 {
		return name[i+1:]
	}
	return ""
}

// matchPath 若 rel 落在某个 path 前缀下，返回剥离前缀后的子路径。
func matchPath(rel string, paths []string) (string, bool) {
	for _, p := range paths {
		p = strings.Trim(p, "/")
		if p == "" {
			continue
		}
		prefix := p + "/"
		if strings.HasPrefix(rel, prefix) {
			return rel[len(prefix):], true
		}
	}
	return "", false
}

func writeFile(dst string, r io.Reader, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("claude: mkdir %q: %w", filepath.Dir(dst), err)
	}
	f, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("claude: create %q: %w", dst, err)
	}
	defer f.Close()

	if _, err := io.Copy(f, r); err != nil {
		return fmt.Errorf("claude: write %q: %w", dst, err)
	}
	return nil
}
