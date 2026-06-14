#!/bin/sh
# 安装 goxctl-claude 扩展：若缺 goxctl 核心则先装核心，再用核心把扩展装到 ~/.goxctl/extensions。
# 用法：curl -sSfL https://raw.githubusercontent.com/chinayin/goxctl-claude/main/install.sh | sh [-s -- <version>]
set -eu

GOXCTL_MODULE="github.com/chinayin/goxctl"
CLAUDE_MODULE="github.com/chinayin/goxctl-claude"
VERSION="${1:-latest}"

GOBIN_DIR="$(go env GOBIN)"
[ -n "$GOBIN_DIR" ] || GOBIN_DIR="$(go env GOPATH)/bin"
GOXCTL_BIN="$GOBIN_DIR/goxctl"

if ! command -v goxctl >/dev/null 2>&1 && [ ! -x "$GOXCTL_BIN" ]; then
	echo "goxctl core not found, installing ${GOXCTL_MODULE}@latest ..."
	go install "${GOXCTL_MODULE}@latest"
fi

# 优先用 PATH 上的 goxctl，否则用 GOBIN 里的
GOXCTL="$(command -v goxctl 2>/dev/null || echo "$GOXCTL_BIN")"

echo "Installing goxctl-claude extension (${VERSION}) ..."
"$GOXCTL" extension install "${CLAUDE_MODULE}" "${VERSION}"

echo "Done. Try: goxctl claude --help"
