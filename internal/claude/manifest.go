package claude

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// LoadManifest 从指定路径读取 manifest；不存在返回 ErrManifestNotFound。
func LoadManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrManifestNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("claude: read manifest %q: %w", path, err)
	}

	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("claude: parse manifest %q: %w", path, err)
	}
	if m.Target == "" {
		m.Target = DefaultTarget
	}
	return &m, nil
}

// SaveManifest 将 manifest 写入指定路径。
func SaveManifest(path string, m *Manifest) error {
	data, err := yaml.Marshal(m)
	if err != nil {
		return fmt.Errorf("claude: marshal manifest: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("claude: write manifest %q: %w", path, err)
	}
	return nil
}
