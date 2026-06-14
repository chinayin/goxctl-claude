# goxctl-claude

团队 AI 协作配置（steering / CLAUDE.md）的版本化同步工具，是 [`goxctl`](https://github.com/chinayin/goxctl) 的 `claude` 扩展。

它从规范源仓库按 git tag 同步 steering 文件到项目本地（默认 `.kiro/steering`），
供 Kiro 与 Claude Code 共用；版本锁定在 `.goxctl-claude.lock`。

## 用法

通常作为 goxctl 子命令使用：

```bash
goxctl claude add chinayin/goxctl-claude --version v1.0.0   # 首次添加并拉取（source 可简写，默认 github.com）
goxctl claude update                                                   # 拉到锁定版本（恢复/校正）
goxctl claude update v1.1.0                                            # 升级到指定版本
goxctl claude list                                                     # 查看源/版本/受管文件
goxctl claude check                                                    # 校验本地与 lock 一致（CI 用）
goxctl claude remove                                                   # 移除受管文件与 manifest/lock
```

私有源通过 `GH_TOKEN` / `GITHUB_TOKEN` 环境变量鉴权。

## 设计

- **版本锁定**：`.goxctl-claude.lock` 只锚定 commit sha（唯一完整性锚点）+ 自动生成的受管文件列表 + 整体 digest，不逐文件记 hash。
- **部分托管**：只管同步来的受管文件，项目自有的 steering 文件原样保留。
- **单一源**：本地与 CI 共用同一份 lock；`check` 防漂移/手改。

详见 [goxctl 架构设计](https://github.com/chinayin/goxctl/blob/main/docs/GOXCTL_ARCHITECTURE.md)。
