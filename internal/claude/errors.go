package claude

import "errors"

var (
	// ErrManifestNotFound 表示项目未初始化（无 manifest）。
	ErrManifestNotFound = errors.New("claude: manifest not found")
	// ErrLockNotFound 表示尚未同步（无 lock）。
	ErrLockNotFound = errors.New("claude: lock not found")
	// ErrDigestMismatch 表示本地受管文件与 lock 记录不一致。
	ErrDigestMismatch = errors.New("claude: managed files digest mismatch")
)
