package claude

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

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

// Add 首次添加规范源：写 manifest 后立即拉取。已初始化则报错。
func (s *Syncer) Add(ctx context.Context, source, version string, paths []string, target string) error {
	if _, err := LoadManifest(s.manifestPath()); err == nil {
		return fmt.Errorf("claude: already initialized (%s exists)", ManifestFile)
	}
	if target == "" {
		target = DefaultTarget
	}
	if len(paths) == 0 {
		paths = []string{"steering/"}
	}

	m := &Manifest{Source: source, Version: version, Paths: paths, Target: target}
	if err := SaveManifest(s.manifestPath(), m); err != nil {
		return err
	}
	return s.pull(ctx, m, version)
}

// Update 按 manifest 拉取：version 为空=拉到 manifest 锁定版本（恢复/校正）；
// 非空=升级到该版本并改写 manifest。
func (s *Syncer) Update(ctx context.Context, version string) error {
	m, err := LoadManifest(s.manifestPath())
	if err != nil {
		return err
	}
	if version != "" && version != m.Version {
		m.Version = version
		if err := SaveManifest(s.manifestPath(), m); err != nil {
			return err
		}
	}
	return s.pull(ctx, m, m.Version)
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
func (s *Syncer) pull(ctx context.Context, m *Manifest, version string) error {
	ref, err := parseSource(m.Source)
	if err != nil {
		return err
	}

	sha, err := s.fetcher.ResolveTag(ctx, ref, version)
	if err != nil {
		return err
	}

	target := filepath.Join(s.dir, m.Target)

	// 部分托管：只清理上一版 lock 记录的受管文件，不动项目自有 steering
	if old, lockErr := LoadLock(s.lockPath()); lockErr == nil {
		if err := removeManaged(target, old.Managed); err != nil {
			return err
		}
	}

	rc, err := s.fetcher.DownloadTarball(ctx, ref, version)
	if err != nil {
		return err
	}
	defer rc.Close()

	managed, err := extractTarball(rc, m.Paths, target)
	if err != nil {
		return err
	}
	if len(managed) == 0 {
		return fmt.Errorf("claude: no files matched paths %v in %s@%s", m.Paths, m.Source, version)
	}

	digest, err := ComputeDigest(target, managed)
	if err != nil {
		return err
	}

	return SaveLock(s.lockPath(), &Lock{
		Source:   m.Source,
		Version:  version,
		Resolved: sha,
		Managed:  managed,
		Digest:   digest,
	})
}

func removeManaged(target string, managed []string) error {
	for _, rel := range managed {
		if err := os.Remove(filepath.Join(target, rel)); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("claude: remove managed %q: %w", rel, err)
		}
	}
	return nil
}
