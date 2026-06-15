package claude

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSyncer_ensureEntrypoint(t *testing.T) {
	dir := t.TempDir()
	s := NewSyncer(dir, nil) // 不依赖 fetcher
	path := filepath.Join(dir, ClaudeMdFile)

	// 空模板内容 → 不生成
	created, err := s.ensureEntrypoint("")
	require.NoError(t, err)
	assert.False(t, created)
	_, statErr := os.Stat(path)
	assert.True(t, os.IsNotExist(statErr))

	// 有内容且不存在 → 写入
	created, err = s.ensureEntrypoint("HELLO\n@.kiro/steering/karpathy-guidelines.md\n")
	require.NoError(t, err)
	assert.True(t, created)
	b, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(b), "@.kiro/steering/karpathy-guidelines.md")

	// 已存在 → 跳过，不覆盖项目自有内容
	created, err = s.ensureEntrypoint("SHOULD NOT OVERWRITE")
	require.NoError(t, err)
	assert.False(t, created)
	b, err = os.ReadFile(path)
	require.NoError(t, err)
	assert.NotContains(t, string(b), "SHOULD NOT OVERWRITE")
}
