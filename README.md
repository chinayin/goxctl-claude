# goxctl-claude

团队 AI 协作规范的版本化同步**工具**，是 [`goxctl`](https://github.com/chinayin/goxctl) 的 `claude` 扩展。

规范数据（steering + CLAUDE 模板）放在独立仓库 [`gox-claude-standards`](https://github.com/chinayin/gox-claude-standards)，
有自己的版本线；本工具只负责按 tag 拉取它到项目本地（默认 `.kiro/steering`，供 Kiro 与 Claude Code 共用），
并在项目根生成通用的 `CLAUDE.md` 入口；版本锁定在 `.gox-claude.lock`。

> **两个独立版本**：工具版本（本仓库 tag，装在 `~/.gox`）与规范版本（`gox-claude-standards` 的 tag，记在项目 lock）互不绑定。

## 安装

无需 Go 环境（macOS / Linux，amd64 / arm64）：

```bash
curl -sSfL https://github.com/chinayin/goxctl-claude/releases/latest/download/install.sh | sh
```

脚本会下载 goxctl 核心（若缺，装到 `/usr/local/bin`，默认在 PATH）与本扩展二进制（装到 `~/.gox/extensions`，由核心管理），全程不依赖 Go。

## 用法

通常作为 goxctl 子命令使用：

```bash
goxctl claude add                       # 首次添加：默认源 gox-claude-standards，不传 --version 取最新并钉住
goxctl claude add <owner>/<repo>        # 指定其它规范源
goxctl claude add --version v0.1.0      # 钉到指定版本
goxctl claude update                    # 升级到最新 release
goxctl claude update v0.1.0             # 切到指定版本
goxctl claude list                      # 查看源/版本/受管文件
goxctl claude check                     # 校验本地与 lock 一致（CI 用）
goxctl claude remove                    # 移除受管文件与 manifest/lock
```

`add` 不带 source 时默认拉团队规范源 `chinayin/gox-claude-standards`。私有源通过 `GH_TOKEN` / `GITHUB_TOKEN` 环境变量鉴权。

## CLAUDE.md 入口（通用，非 Go 专属）

`add` 会在项目根生成 `CLAUDE.md`（仅当不存在；已有则原样保留）。它是**语言 / 技术栈无关的团队顶层入口**：

- **Always on**：只 `@import` 通用行为准则（`karpathy-guidelines`）。
- **领域规范条件化**：仅在「Go 项目」一节引用 `rules.md` 与 cli/config/db/scaffold —— 非 Go 项目不会被 Go 规范注入。
- `## Project context` 留给项目自有内容（本地维护，不同步、不受管）。

## 设计

- **版本锁定**：`.gox-claude.lock` 只锚定 commit sha（唯一完整性锚点）+ 自动生成的受管文件列表 + 整体 digest，不逐文件记 hash。
- **部分托管**：只管同步来的受管文件（`.kiro/steering` 下）；`CLAUDE.md` 与项目自有 steering 文件**不受管**，`check` / `remove` 不触碰。
- **单一源**：本地与 CI 共用同一份 lock；`check` 防漂移/手改。

详见 [goxctl 架构设计](https://github.com/chinayin/goxctl/blob/main/docs/GOXCTL_ARCHITECTURE.md)。
