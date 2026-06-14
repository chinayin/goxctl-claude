package claude

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"gopkg.in/yaml.v3"
)

// LoadLock 从指定路径读取 lock；不存在返回 ErrLockNotFound。
func LoadLock(path string) (*Lock, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrLockNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("claude: read lock %q: %w", path, err)
	}

	var l Lock
	if err := yaml.Unmarshal(data, &l); err != nil {
		return nil, fmt.Errorf("claude: parse lock %q: %w", path, err)
	}
	return &l, nil
}

// SaveLock 将 lock 写入指定路径。
func SaveLock(path string, l *Lock) error {
	data, err := yaml.Marshal(l)
	if err != nil {
		return fmt.Errorf("claude: marshal lock: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("claude: write lock %q: %w", path, err)
	}
	return nil
}

// ComputeDigest 计算受管文件集合在 target 目录下的整体摘要。
// 文件按相对路径排序后，依次混入「路径 + 内容」，因此与传入顺序无关、稳定可复算。
func ComputeDigest(target string, managed []string) (string, error) {
	files := slices.Clone(managed)
	slices.Sort(files)

	h := sha256.New()
	for _, rel := range files {
		content, err := os.ReadFile(filepath.Join(target, rel))
		if err != nil {
			return "", fmt.Errorf("claude: read managed file %q: %w", rel, err)
		}
		fmt.Fprintf(h, "%s\n", rel)
		h.Write(content)
	}
	return fmt.Sprintf("sha256:%x", h.Sum(nil)), nil
}

// VerifyDigest 校验 target 下受管文件的当前摘要是否等于 want；不等返回 ErrDigestMismatch。
func VerifyDigest(target string, managed []string, want string) error {
	got, err := ComputeDigest(target, managed)
	if err != nil {
		return err
	}
	if got != want {
		return fmt.Errorf("%w: want %s, got %s", ErrDigestMismatch, want, got)
	}
	return nil
}
