package claude

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// claudeTemplateInTarball 是仓库内 CLAUDE.md 模板的路径（剥离顶层目录后）。
// 它随规范一起按 tag 拉取，内容返回给调用方用于生成项目入口，不写入 target。
const claudeTemplateInTarball = "CLAUDE.template.md"

const (
	// maxFileSize 是单个受管文件解压后的体积上限（防解压炸弹）。
	maxFileSize = 10 << 20 // 10MB
	// maxTotalSize 是单次解压所有受管文件的总体积上限。
	maxTotalSize = 50 << 20 // 50MB
)

// extractTarball 从 GitHub tarball 流中提取匹配 paths 的文件到 target，
// 并顺带返回仓库根 CLAUDE.template.md 的内容（若存在）。
//
// GitHub tarball 顶层有一个 "{repo}-{ref}/" 包裹目录，先剥离；paths 中的目录前缀
// （如 "steering/"）也剥离，内容展平到 target 下。返回写入的受管文件相对 target 的路径（已排序）
// 与 CLAUDE 模板内容。
func extractTarball(r io.Reader, paths []string, target string) (managed []string, claudeTemplate string, err error) {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return nil, "", fmt.Errorf("claude: open gzip: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	var total int64
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, "", fmt.Errorf("claude: read tar: %w", err)
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}

		rel := stripTopDir(hdr.Name)

		// CLAUDE 模板随规范一起拉取，内容返回（不写入 target，由 ensureEntrypoint 决定是否落地）
		if rel == claudeTemplateInTarball {
			claudeTemplate, err = readTemplate(tr)
			if err != nil {
				return nil, "", err
			}
			continue
		}

		sub, ok := matchPath(rel, paths)
		if !ok {
			continue
		}
		written, err := writeManaged(target, sub, tr)
		if err != nil {
			return nil, "", err
		}
		total += written
		if total > maxTotalSize {
			return nil, "", fmt.Errorf("claude: extracted size exceeds %d bytes", maxTotalSize)
		}
		managed = append(managed, sub)
	}

	slices.Sort(managed)
	return managed, claudeTemplate, nil
}

// readTemplate 读取归档中的 CLAUDE 模板内容，限制体积防解压炸弹。
func readTemplate(r io.Reader) (string, error) {
	b, err := io.ReadAll(io.LimitReader(r, maxFileSize+1))
	if err == nil && len(b) > maxFileSize {
		return "", fmt.Errorf("claude: %s exceeds %d bytes", claudeTemplateInTarball, maxFileSize)
	}
	if err != nil {
		return "", fmt.Errorf("claude: read template: %w", err)
	}
	return string(b), nil
}

// writeManaged 将归档条目安全写入 target/sub，返回写入字节数。
func writeManaged(target, sub string, r io.Reader) (int64, error) {
	dst, err := safeJoin(target, sub)
	if err != nil {
		return 0, err
	}
	if err := writeFile(dst, r); err != nil {
		return 0, err
	}
	fi, err := os.Stat(dst)
	if err != nil {
		return 0, fmt.Errorf("claude: stat %q: %w", dst, err)
	}
	return fi.Size(), nil
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

// safeJoin 把 sub 拼到 base 下，并确保结果不逃逸 base（防 tar-slip 路径穿越）。
func safeJoin(base, sub string) (string, error) {
	dst := filepath.Clean(filepath.Join(base, sub))
	if dst != base && !strings.HasPrefix(dst, base+string(os.PathSeparator)) {
		return "", fmt.Errorf("claude: unsafe path %q escapes target", sub)
	}
	return dst, nil
}

func writeFile(dst string, r io.Reader) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("claude: mkdir %q: %w", filepath.Dir(dst), err)
	}
	f, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644) // 固定权限，不信任归档
	if err != nil {
		return fmt.Errorf("claude: create %q: %w", dst, err)
	}
	defer f.Close()

	// 限制单文件体积，防解压炸弹；CopyN 复制至多 maxFileSize+1 字节以探测超限
	n, err := io.CopyN(f, r, maxFileSize+1)
	if err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("claude: write %q: %w", dst, err)
	}
	if n > maxFileSize {
		return fmt.Errorf("claude: %q exceeds %d bytes", dst, maxFileSize)
	}
	return nil
}
