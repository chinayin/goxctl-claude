// Package claude 实现团队 AI 协作配置（steering / CLAUDE.md 模板）的版本化同步。
//
// 它从规范源仓库按 git tag 拉取受管文件到项目本地（默认 .kiro/steering），
// 供 Kiro 与 Claude Code 共用；版本锁定在 .gox-claude.lock。
//
// 完整性以 git commit sha 为唯一锚点（内容寻址），lock 不逐文件记 hash，
// 仅记录自动生成的受管文件列表与一个整体 digest，供离线 check 防本地篡改。
package claude
