package claude

import (
	"fmt"
	"os"
	"path/filepath"
)

// ClaudeMdFile 是项目入口文件名，承载 always-on 注入链（@import .kiro/steering/*）。
const ClaudeMdFile = "CLAUDE.md"

// ensureEntrypoint 在项目根写入 CLAUDE.md 入口，仅当其不存在且提供了模板内容。
//
// content 为随规范一起拉取的 CLAUDE 模板（与 steering 同版本，见 extractTarball）；为空表示
// 该版本未附带模板，跳过。CLAUDE.md 含项目自有内容（Project context），故作为一次性脚手架：
// 不纳入受管，check / remove 不触碰它。返回是否本次新建。
func (s *Syncer) ensureEntrypoint(content string) (bool, error) {
	if content == "" {
		return false, nil
	}
	path := filepath.Join(s.dir, ClaudeMdFile)
	switch _, err := os.Stat(path); {
	case err == nil:
		return false, nil // 已存在，跳过（不覆盖项目自有内容）
	case !os.IsNotExist(err):
		return false, fmt.Errorf("claude: stat %s: %w", ClaudeMdFile, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return false, fmt.Errorf("claude: write %s: %w", ClaudeMdFile, err)
	}
	return true, nil
}
