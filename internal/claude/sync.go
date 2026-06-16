package claude

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/chinayin/goxctl-claude/internal/ui"
)

// maxTarballBytes 是下载的 tarball（压缩态）体积上限，防止异常巨大的响应耗尽内存。
const maxTarballBytes = 50 << 20 // 50MB

// Syncer 在某个项目目录下执行规范同步（add/update/check/remove）。
type Syncer struct {
	dir     string // 项目根目录
	fetcher *Fetcher
}

// NewSyncer 创建 Syncer。
func NewSyncer(dir string, fetcher *Fetcher) *Syncer {
	return &Syncer{dir: dir, fetcher: fetcher}
}

func (s *Syncer) manifestPath() string { return filepath.Join(s.dir, ManifestFile) }
func (s *Syncer) lockPath() string     { return filepath.Join(s.dir, LockFile) }

// Add 首次添加规范源：写 manifest 后立即拉取，并生成 CLAUDE.md 入口（仅当不存在）。
// 已初始化则报错。返回是否本次新建了 CLAUDE.md。
func (s *Syncer) Add(ctx context.Context, source, version string, paths []string, target string) (bool, error) {
	if _, err := LoadManifest(s.manifestPath()); err == nil {
		return false, fmt.Errorf("claude: already initialized (%s exists)", ManifestFile)
	}
	if target == "" {
		target = DefaultTarget
	}
	if len(paths) == 0 {
		paths = []string{"steering/"}
	}

	// 未指定版本：解析最新 release tag 并钉住具体版本（保持可复现，非 rolling latest）
	if version == "" {
		ref, err := parseSource(source)
		if err != nil {
			return false, err
		}
		version, err = s.fetcher.ResolveLatest(ctx, ref)
		if err != nil {
			return false, err
		}
	}

	m := &Manifest{Source: source, Version: version, Paths: paths, Target: target}
	tmpl, err := s.pull(ctx, m, version)
	if err != nil {
		return false, err // pull 失败：不留下 manifest，add 可直接重试
	}
	if err := SaveManifest(s.manifestPath(), m); err != nil {
		return false, err
	}
	// CLAUDE.md 模板随规范一起拉取（与 steering 同版本），据此生成项目入口
	return s.ensureEntrypoint(tmpl)
}

// Update 升级规范：version 为空=升级到最新 release；非空=切到指定版本。
// 两种情况都会改写 manifest 与 lock（steering 跟随项目进 git，新 clone 自带文件，无需“恢复”语义）。
func (s *Syncer) Update(ctx context.Context, version string) error {
	m, err := LoadManifest(s.manifestPath())
	if err != nil {
		return err
	}

	// 未指定版本：升级到最新 release
	if version == "" {
		ref, err := parseSource(m.Source)
		if err != nil {
			return err
		}
		version, err = s.fetcher.ResolveLatest(ctx, ref)
		if err != nil {
			return err
		}
	}

	if version != m.Version {
		m.Version = version
		if err := SaveManifest(s.manifestPath(), m); err != nil {
			return err
		}
	}
	_, err = s.pull(ctx, m, version)
	return err
}

// Check 校验本地受管文件与 lock 一致（CI 防漂移/手改）。
func (s *Syncer) Check() error {
	l, err := LoadLock(s.lockPath())
	if err != nil {
		return err
	}
	m, err := LoadManifest(s.manifestPath())
	if err != nil {
		return err
	}
	return VerifyDigest(filepath.Join(s.dir, m.Target), l.Managed, l.Digest)
}

// Remove 移除受管文件并删除 manifest/lock；不碰项目自有文件。
func (s *Syncer) Remove() error {
	m, errM := LoadManifest(s.manifestPath())
	l, errL := LoadLock(s.lockPath())
	if errM == nil && errL == nil {
		if err := removeManaged(filepath.Join(s.dir, m.Target), l.Managed); err != nil {
			return err
		}
	}
	_ = os.Remove(s.lockPath())
	_ = os.Remove(s.manifestPath())
	return nil
}

// Status 返回当前 manifest 与 lock（lock 可能尚不存在，返回 nil）。
func (s *Syncer) Status() (*Manifest, *Lock, error) {
	m, err := LoadManifest(s.manifestPath())
	if err != nil {
		return nil, nil, err
	}
	l, _ := LoadLock(s.lockPath())
	return m, l, nil
}

// pull 执行实际拉取：resolve → 清旧受管 → download → extract → 写 lock。
// 返回随规范一起拉取的 CLAUDE 模板内容（可能为空，供调用方决定是否生成入口）。
func (s *Syncer) pull(ctx context.Context, m *Manifest, version string) (string, error) {
	ref, err := parseSource(m.Source)
	if err != nil {
		return "", err
	}

	sha, err := s.fetcher.ResolveTag(ctx, ref, version)
	if err != nil {
		return "", err
	}
	ui.Stepf(os.Stdout, "Pulling %s %s...", m.Source, version)

	// 先把 tarball 完整下载到内存（有界）：网络失败发生在删除旧文件之前，失败即无副作用
	rc, err := s.fetcher.DownloadTarball(ctx, ref, version)
	if err != nil {
		return "", err
	}
	buf, err := io.ReadAll(io.LimitReader(rc, maxTarballBytes+1))
	_ = rc.Close()
	if err != nil {
		return "", fmt.Errorf("claude: download %s@%s: %w", m.Source, version, err)
	}
	if int64(len(buf)) > maxTarballBytes {
		return "", fmt.Errorf("claude: tarball %s@%s exceeds %d bytes", m.Source, version, maxTarballBytes)
	}

	target := filepath.Join(s.dir, m.Target)

	// 部分托管：下载成功后才清理上一版 lock 记录的受管文件
	if old, lockErr := LoadLock(s.lockPath()); lockErr == nil {
		if err := removeManaged(target, old.Managed); err != nil {
			return "", err
		}
	}

	managed, claudeTemplate, err := extractTarball(bytes.NewReader(buf), m.Paths, target)
	if err != nil {
		return "", err
	}
	if len(managed) == 0 {
		return "", fmt.Errorf("claude: no files matched paths %v in %s@%s", m.Paths, m.Source, version)
	}

	digest, err := ComputeDigest(target, managed)
	if err != nil {
		return "", err
	}

	if err := SaveLock(s.lockPath(), &Lock{
		Source:   m.Source,
		Version:  version,
		Resolved: sha,
		Managed:  managed,
		Digest:   digest,
	}); err != nil {
		return "", err
	}
	return claudeTemplate, nil
}

func removeManaged(target string, managed []string) error {
	for _, rel := range managed {
		if err := os.Remove(filepath.Join(target, rel)); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("claude: remove managed %q: %w", rel, err)
		}
	}
	return nil
}
