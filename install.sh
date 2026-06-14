#!/bin/sh
# 安装 goxctl 核心 + goxctl-claude 扩展，零 Go 依赖（下载预编译二进制）。
# 用法：curl -sSfL https://raw.githubusercontent.com/chinayin/goxctl-claude/main/install.sh | sh [-s -- <version>]
set -eu

CORE_REPO="chinayin/goxctl"
EXT_REPO="chinayin/goxctl-claude"
INSTALL_DIR="${GOXCTL_BIN_DIR:-$HOME/.goxctl/bin}"
VERSION="${1:-latest}"

# 平台探测（仅支持 macOS / Linux，amd64 / arm64）
os=$(uname -s | tr '[:upper:]' '[:lower:]')
arch=$(uname -m)
case "$arch" in
	x86_64 | amd64) arch=amd64 ;;
	aarch64 | arm64) arch=arm64 ;;
	*) echo "不支持的架构: $arch" >&2; exit 1 ;;
esac
case "$os" in
	darwin | linux) ;;
	*) echo "不支持的系统: $os（install.sh 仅支持 macOS/Linux）" >&2; exit 1 ;;
esac

fetch() { # url dest
	if command -v curl >/dev/null 2>&1; then
		curl -sSfL "$1" -o "$2"
	else
		wget -qO "$2" "$1"
	fi
}

sha256_of() { # file -> sha
	if command -v sha256sum >/dev/null 2>&1; then
		sha256sum "$1" | awk '{print $1}'
	else
		shasum -a 256 "$1" | awk '{print $1}'
	fi
}

asset_url() { # repo asset -> url
	if [ "$VERSION" = latest ]; then
		echo "https://github.com/$1/releases/latest/download/$2"
	else
		echo "https://github.com/$1/releases/download/$VERSION/$2"
	fi
}

install_core() {
	asset="goxctl_${os}_${arch}.tar.gz"
	tmp=$(mktemp -d)
	echo "下载 goxctl 核心 ($VERSION, $os/$arch) ..."
	fetch "$(asset_url "$CORE_REPO" "$asset")" "$tmp/$asset"
	fetch "$(asset_url "$CORE_REPO" checksums.txt)" "$tmp/checksums.txt"

	want=$(grep " $asset\$" "$tmp/checksums.txt" | awk '{print $1}')
	got=$(sha256_of "$tmp/$asset")
	if [ -z "$want" ] || [ "$want" != "$got" ]; then
		echo "校验失败: $asset" >&2
		exit 1
	fi

	mkdir -p "$INSTALL_DIR"
	tar -xzf "$tmp/$asset" -C "$INSTALL_DIR" goxctl
	chmod +x "$INSTALL_DIR/goxctl"
	rm -rf "$tmp"
}

if ! command -v goxctl >/dev/null 2>&1 && [ ! -x "$INSTALL_DIR/goxctl" ]; then
	install_core
fi
GOXCTL="$(command -v goxctl 2>/dev/null || echo "$INSTALL_DIR/goxctl")"

echo "安装 goxctl-claude 扩展 ($VERSION) ..."
"$GOXCTL" extension install "$EXT_REPO" "$VERSION"

echo
echo "完成。若 goxctl 不在 PATH，请加入："
echo "  export PATH=\"$INSTALL_DIR:\$PATH\""
echo "试试：goxctl claude --help"
