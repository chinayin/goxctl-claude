package claude

import (
	"fmt"
	"strings"
)

// defaultHost 是 source 简写（owner/repo）缺省的代码托管主机名。
const defaultHost = "github.com"

// RepoRef 是解析后的 GitHub 仓库引用。
type RepoRef struct {
	Owner string
	Repo  string
}

// parseSource 解析形如 "github.com/owner/repo" 的源地址（容忍 https:// 前缀与首尾斜杠）。
func parseSource(source string) (RepoRef, error) {
	s := strings.TrimPrefix(source, "https://")
	s = strings.TrimPrefix(s, defaultHost+"/")
	s = strings.Trim(s, "/")

	parts := strings.Split(s, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return RepoRef{}, fmt.Errorf("claude: invalid source %q, want %s/owner/repo", source, defaultHost)
	}
	return RepoRef{Owner: parts[0], Repo: parts[1]}, nil
}
