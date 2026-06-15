package claude

// Manifest 是项目对规范源与版本的声明（写入 ManifestFile，进 git）。
type Manifest struct {
	Source  string   `yaml:"source"`  // 规范源仓库，如 github.com/chinayin/goxctl-claude
	Version string   `yaml:"version"` // 精确 tag，如 v1.2.0
	Paths   []string `yaml:"paths"`   // 要同步的目录/glob，如 steering/
	Target  string   `yaml:"target"`  // 落地目录，缺省 DefaultTarget
}

// Lock 锁定已同步的规范版本（写入 LockFile，进 git）。
type Lock struct {
	Source   string   `yaml:"source"`
	Version  string   `yaml:"version"`  // 人类可读 tag
	Resolved string   `yaml:"resolved"` // commit sha，唯一完整性锚点
	Managed  []string `yaml:"managed"`  // 受管文件相对路径，update 时自动生成
	Digest   string   `yaml:"digest"`   // 受管文件整体摘要，供离线 check
}

const (
	// DefaultSource 是默认规范源仓库（add 未指定 source 时使用）。
	DefaultSource = "chinayin/gox-claude-standards"
	// DefaultTarget 是规范文件默认落地目录（Kiro 与 Claude Code 共用）。
	DefaultTarget = ".kiro/steering"
	// ManifestFile 是项目声明文件名。
	ManifestFile = ".gox-claude.yaml"
	// LockFile 是版本锁定文件名。
	LockFile = ".gox-claude.lock"
)
