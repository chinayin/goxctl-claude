---
inclusion: fileMatch
fileMatchPattern: "{Makefile,.gitignore,.editorconfig,.golangci-lint-version,.github/workflows/*.yml}"
---

# Project Scaffold Standards

Every new Go project must include these root-level files. Use these templates directly.

## .editorconfig

```editorconfig
root = true

[*]
charset = utf-8
end_of_line = lf
insert_final_newline = true
trim_trailing_whitespace = true
indent_style = space
indent_size = 4

[*.go]
indent_style = tab

[*.{yaml,yml,toml}]
indent_size = 2

[*.md]
trim_trailing_whitespace = false

[Makefile]
indent_style = tab
```

## .gitignore

```gitignore
# Build output (含 make lint 下载的 golangci-lint 二进制)
bin/

# Local config overrides
config/*.local.yaml
.env

# macOS
.DS_Store

# IDE
.idea/
.vscode/
*.swp
*.swo

# Kiro
.kiro/
```

## .golangci-lint-version

锁定本项目使用的 golangci-lint 版本，作为**单一版本源**（本地与 CI 共用）。内容为一行版本号：

```
v2.12.2
```

- 本地 `make lint` 据此自动下载锁定版本到 `./bin`，不依赖 brew / 全局安装
- CI 用 `golangci-lint-action` 的 `version-file` 读取它
- 升级 linter 只改这一个文件，本地和 CI 同步跟随
- 注意：golangci-lint 官方预编译二进制按 release 时的 Go 版本编译，其版本若**低于** go.mod 的 `go` 指令会拒绝运行，所以锁定版本要选用 ≥ go.mod 版本编译的发行版

## Makefile

Required targets (variable declarations are project-specific, not part of this standard):

```makefile
.PHONY: help build run clean test lint lint-fix fmt tidy check install-tools ensure-lint

# golangci-lint 版本锁定（随项目走，升级改 .golangci-lint-version）
GOLANGCI_VERSION := $(shell cat .golangci-lint-version)
GOLANGCI := ./bin/golangci-lint

help: ## Show help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  make %-13s - %s\n", $$1, $$2}'

build: clean ## Compile
	@go build -ldflags "$(LDFLAGS)" -o bin/$(APP) ./cmd/$(APP)

run: build ## Run
	@./bin/$(APP) --help

clean: ## Clean build output
	@rm -rf bin/

ensure-lint:
	@if [ ! -x "$(GOLANGCI)" ] || ! "$(GOLANGCI)" version 2>/dev/null | grep -q "$(GOLANGCI_VERSION:v%=%)"; then \
		curl -sSfL https://golangci-lint.run/install.sh | sh -s -- -b ./bin "$(GOLANGCI_VERSION)"; \
	fi

install-tools: ensure-lint ## Install pinned golangci-lint to ./bin
	@echo "golangci-lint $(GOLANGCI_VERSION) ready at $(GOLANGCI)"

test: ## Run tests
	@go test -v -race -count=1 -timeout 120s ./...

lint: ensure-lint ## Run linter (pinned version)
	@$(GOLANGCI) run ./...

lint-fix: ensure-lint ## Run linter with auto-fix
	@$(GOLANGCI) run --fix ./...

fmt: lint-fix ## Format code (lint + fix)
	@echo "Code formatted!"

check: fmt lint test ## Full local check (run before commit)
	@echo "All checks passed!"
```

### Makefile Rules

- Default target is `help`, auto-generated from `## comments`
- Every target must have `## comment` describing its purpose (internal targets like `ensure-lint` omit it to stay out of help)
- Prefix commands with `@` to suppress echo
- `build` depends on `clean` for clean output
- Tests must include `-race` and `-timeout`
- `check` is pre-commit full validation (format + lint + test)
- **Lint 用项目锁定版本**：`lint`/`lint-fix` 依赖 `ensure-lint`，从 `.golangci-lint-version` 取版本、下载到 `./bin`，**绝不直接调全局 `golangci-lint`**（避免团队成员版本不一致）
- Unified linter: golangci-lint v2
- Optional additions: `ci-check` (CI pipeline, no fmt), `docker-up/down/logs`

## CI Workflows

CI 与本地共用版本源——**Go 版本读 `go.mod`，golangci-lint 版本读 `.golangci-lint-version`，workflow 内不写死任何版本**。

`.github/workflows/lint.yml`:

```yaml
name: Lint

on:
  push:
    branches:
      - main
    tags:
      - 'v*'
  pull_request:

permissions:
  checks: write

jobs:
  golangci:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout code
        uses: actions/checkout@v6

      - name: Set up Go
        uses: actions/setup-go@v6
        with:
          go-version-file: go.mod

      - name: Run golangci-lint
        uses: golangci/golangci-lint-action@v9
        with:
          version-file: .golangci-lint-version
```

`test.yml` / `build.yml` 同样用 `actions/setup-go` 的 `go-version-file: go.mod`，不再用 matrix 写死 go 版本（除非真要测多版本）。

### CI Rules

- 监听分支用仓库默认分支（统一 `main`）；workflow 的 `branches` 必须与默认分支一致，否则 push 不触发
- Go 版本单一源 `go.mod`，golangci-lint 版本单一源 `.golangci-lint-version`
- Actions 固定大版本（`@v6`/`@v9`），不用浮动 ref
