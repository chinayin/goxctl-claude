#!/bin/sh
# shellcheck disable=SC3043
# goxctl-claude installer — installs the goxctl core (if missing) and the claude
# extension from prebuilt binaries. No Go required.
# Usage: curl -sSfL https://raw.githubusercontent.com/chinayin/goxctl-claude/main/install.sh | sh [-s -- [options]]
set -u

CORE_REPO="chinayin/goxctl"
EXT_REPO="chinayin/goxctl-claude"
INSTALL_DIR="${GOXCTL_BIN_DIR:-/usr/local/bin}"

# --- terminal detection ---

_use_color=false
if [ -t 2 ]; then
	if [ "${TERM+set}" = 'set' ]; then
		case "$TERM" in
			xterm* | rxvt* | urxvt* | linux* | vt* | screen* | tmux*) _use_color=true ;;
		esac
	fi
fi

info() {
	if $_use_color; then printf '\033[32minfo\033[0m: %s\n' "$1" >&2; else printf 'info: %s\n' "$1" >&2; fi
}
warn() {
	if $_use_color; then printf '\033[33mwarn\033[0m: %s\n' "$1" >&2; else printf 'warn: %s\n' "$1" >&2; fi
}
err() {
	if $_use_color; then printf '\033[31merror\033[0m: %s\n' "$1" >&2; else printf 'error: %s\n' "$1" >&2; fi
}

need_cmd() {
	if ! command -v "$1" > /dev/null 2>&1; then
		err "required command not found: $1"
		exit 1
	fi
}

# --- downloader (curl/wget with TLS enforcement) ---

_downloader=""

detect_downloader() {
	if command -v curl > /dev/null 2>&1; then
		_downloader=curl
	elif command -v wget > /dev/null 2>&1; then
		_downloader=wget
	else
		err "either curl or wget is required"
		exit 1
	fi
}

download() { # url output
	if [ "$_downloader" = curl ]; then
		curl --proto '=https' --tlsv1.2 -sSfL "$1" -o "$2"
	else
		wget --https-only --secure-protocol=TLSv1_2 -q "$1" -O "$2"
	fi
}

# install_binary src dst —— 安装到默认在 PATH 的目录；目标不可写时回退 sudo。
install_binary() {
	local _src="$1" _dst="$2" _dir
	_dir=$(dirname "$_dst")
	if mkdir -p "$_dir" 2> /dev/null && [ -w "$_dir" ]; then
		mv -f "$_src" "$_dst"
		chmod 755 "$_dst"
	elif command -v sudo > /dev/null 2>&1; then
		info "installing to $_dir (requires sudo)"
		sudo mkdir -p "$_dir"
		sudo mv -f "$_src" "$_dst"
		sudo chmod 755 "$_dst"
	else
		err "cannot write to $_dir and sudo not found; set GOXCTL_BIN_DIR to a writable dir"
		exit 1
	fi
}

# --- platform detection ---

get_target() {
	local _os _arch
	_os="$(uname -s)"
	case "$_os" in
		Darwin) _os=darwin ;;
		Linux) _os=linux ;;
		*) err "unsupported OS: $_os (only macOS/Linux are supported)"; exit 1 ;;
	esac
	_arch="$(uname -m)"
	case "$_arch" in
		x86_64 | x86-64 | x64 | amd64) _arch=amd64 ;;
		aarch64 | arm64) _arch=arm64 ;;
		*) err "unsupported architecture: $_arch (only amd64/arm64)"; exit 1 ;;
	esac
	echo "${_os}_${_arch}"
}

sha256_of() { # file -> sha
	if command -v sha256sum > /dev/null 2>&1; then
		sha256sum "$1" | awk '{print $1}'
	else
		shasum -a 256 "$1" | awk '{print $1}'
	fi
}

# install_core downloads the latest goxctl core binary into INSTALL_DIR.
# Relies on the global _tmpdir created in main().
install_core() {
	local _target _base _asset _want _got
	_target=$(get_target)
	_base="https://github.com/${CORE_REPO}/releases/latest/download"
	_asset="goxctl_${_target}.tar.gz"

	info "downloading goxctl core (${_asset})..."
	if ! download "${_base}/${_asset}" "${_tmpdir}/${_asset}"; then
		err "download failed: ${_base}/${_asset}"
		exit 1
	fi
	if ! download "${_base}/checksums.txt" "${_tmpdir}/core-checksums.txt"; then
		err "failed to download core checksums.txt"
		exit 1
	fi

	info "verifying core checksum..."
	_want=$(awk -v f="$_asset" '$2 == f {print $1}' "${_tmpdir}/core-checksums.txt")
	_got=$(sha256_of "${_tmpdir}/${_asset}")
	if [ -z "$_want" ] || [ "$_want" != "$_got" ]; then
		err "checksum verification failed for ${_asset}"
		exit 1
	fi

	tar -xzf "${_tmpdir}/${_asset}" -C "${_tmpdir}" goxctl
	install_binary "${_tmpdir}/goxctl" "${INSTALL_DIR}/goxctl"
	info "installed core: ${INSTALL_DIR}/goxctl"
}

usage() {
	printf '%s\n' \
		"goxctl-claude installer" \
		"" \
		"Usage:" \
		"  curl -sSfL .../install.sh | sh -s -- [options]" \
		"" \
		"Options:" \
		"  --version=VER   Extension version to install (default: latest)" \
		"  --proxy=URL     HTTPS proxy for downloads" \
		"  --dir=PATH      Install directory for the core (default: /usr/local/bin)" \
		"  --help          Show this help"
}

main() {
	detect_downloader
	need_cmd uname
	need_cmd mktemp
	need_cmd tar
	need_cmd awk

	local _version="" _proxy=""
	for arg in "$@"; do
		case "$arg" in
			--version=*) _version="${arg#*=}" ;;
			--proxy=*) _proxy="${arg#*=}" ;;
			--dir=*) INSTALL_DIR="${arg#*=}" ;;
			--help | -h) usage; exit 0 ;;
			*) warn "unknown argument: $arg" ;;
		esac
	done

	if [ -n "$_proxy" ]; then
		export https_proxy="$_proxy" http_proxy="$_proxy" HTTPS_PROXY="$_proxy" HTTP_PROXY="$_proxy"
		info "using proxy: $_proxy"
	fi

	# _tmpdir must be global: the EXIT trap fires after main returns.
	_tmpdir=$(mktemp -d) || { err "failed to create temp directory"; exit 1; }
	trap 'if [ -n "${_tmpdir:-}" ]; then rm -rf "$_tmpdir"; fi' EXIT INT TERM

	# core (always latest; the extension is the versioned part)
	if command -v goxctl > /dev/null 2>&1 || [ -x "${INSTALL_DIR}/goxctl" ]; then
		info "goxctl core already present"
	else
		install_core
	fi
	local _goxctl
	_goxctl="$(command -v goxctl 2> /dev/null || echo "${INSTALL_DIR}/goxctl")"

	# extension (installed by the core, which prefers prebuilt binaries)
	local _extver=latest
	if [ -n "$_version" ]; then
		case "$_version" in v*) _extver="$_version" ;; *) _extver="v${_version}" ;; esac
	fi
	info "installing goxctl-claude extension (${_extver})..."
	"$_goxctl" extension install "$EXT_REPO" "$_extver"

	case ":${PATH}:" in
		*":${INSTALL_DIR}:"*) ;;
		*) warn "add to PATH: export PATH=\"${INSTALL_DIR}:\$PATH\"" ;;
	esac
	info "done — try: goxctl claude --help"
}

main "$@"
