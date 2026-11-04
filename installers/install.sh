#!/bin/sh
# Install bibx from GitHub Releases.
#
#   curl -fsSL https://raw.githubusercontent.com/coreofscience/go-bibx/main/installers/install.sh | sh
#
# Environment variables:
#   BIBX_VERSION      release tag to install, e.g. v0.7.0 (default: latest)
#   BIBX_INSTALL_DIR  where to put the binary (default: $HOME/.local/bin)

set -eu

REPO="coreofscience/go-bibx"

say() {
	printf 'bibx-installer: %s\n' "$*" >&2
}

fail() {
	say "error: $*"
	exit 1
}

has() {
	command -v "$1" >/dev/null 2>&1
}

download() {
	# download URL DEST
	if has curl; then
		curl -fsSL --retry 3 -o "$2" "$1"
	elif has wget; then
		wget -q -O "$2" "$1"
	else
		fail "curl or wget is required"
	fi
}

latest_version() {
	url="https://github.com/$REPO/releases/latest"
	if has curl; then
		effective=$(curl -fsSLI -o /dev/null -w '%{url_effective}' "$url")
	else
		effective=$(wget -q -S --spider "$url" 2>&1 | sed -n 's/^ *[Ll]ocation: *//p' | tail -n 1 | tr -d '\r')
	fi
	tag=${effective##*/}
	case "$tag" in
	v*) printf '%s\n' "$tag" ;;
	*) fail "could not determine the latest release (got '$effective')" ;;
	esac
}

sha256_of() {
	if has sha256sum; then
		sha256sum "$1" | cut -d ' ' -f 1
	elif has shasum; then
		shasum -a 256 "$1" | cut -d ' ' -f 1
	fi
}

main() {
	case "$(uname -s)" in
	Linux) os="Linux" ;;
	Darwin) os="Darwin" ;;
	*) fail "unsupported OS '$(uname -s)'; on Windows use install.ps1" ;;
	esac

	case "$(uname -m)" in
	x86_64 | amd64) arch="x86_64" ;;
	aarch64 | arm64) arch="arm64" ;;
	*) fail "unsupported architecture '$(uname -m)'" ;;
	esac

	if [ "$os" = "Darwin" ] && [ "$arch" = "x86_64" ]; then
		fail "Intel Macs are not supported; install with: go install github.com/$REPO/cmd/bibx@latest"
	fi

	version=${BIBX_VERSION:-}
	if [ -z "$version" ]; then
		version=$(latest_version)
	fi
	case "$version" in
	v*) ;;
	*) version="v$version" ;;
	esac

	install_dir=${BIBX_INSTALL_DIR:-"$HOME/.local/bin"}
	archive="go-bibx_${os}_${arch}.tar.gz"
	checksums="go-bibx_${version#v}_checksums.txt"
	base="https://github.com/$REPO/releases/download/$version"

	tmp=$(mktemp -d)
	trap 'rm -rf "$tmp"' EXIT INT TERM

	say "downloading bibx $version for $os/$arch"
	download "$base/$archive" "$tmp/$archive" || fail "could not download $base/$archive"
	download "$base/$checksums" "$tmp/$checksums" || fail "could not download $base/$checksums"

	expected=$(awk -v f="$archive" '$2 == f { print $1 }' "$tmp/$checksums")
	[ -n "$expected" ] || fail "no checksum for $archive in $checksums"
	actual=$(sha256_of "$tmp/$archive")
	if [ -z "$actual" ]; then
		say "warning: sha256sum/shasum not found, skipping checksum verification"
	elif [ "$actual" != "$expected" ]; then
		fail "checksum mismatch for $archive (expected $expected, got $actual)"
	fi

	mkdir -p "$tmp/out"
	tar -xzf "$tmp/$archive" -C "$tmp/out"
	# Releases up to v0.7.0 named the binary go-bibx.
	binary="$tmp/out/bibx"
	[ -f "$binary" ] || binary="$tmp/out/go-bibx"
	[ -f "$binary" ] || fail "no bibx binary in $archive"
	mkdir -p "$install_dir"
	install -m 755 "$binary" "$install_dir/bibx"
	say "installed $("$install_dir/bibx" --version 2>/dev/null || echo bibx) to $install_dir/bibx"

	case ":$PATH:" in
	*":$install_dir:"*) ;;
	*)
		say "$install_dir is not on your PATH; add this to your shell profile:"
		say "  export PATH=\"$install_dir:\$PATH\""
		;;
	esac
}

main "$@"
