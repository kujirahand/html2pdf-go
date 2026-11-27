#!/bin/sh
# Install the latest (or a given) html2pdf-go release for Linux / macOS.
#
#   curl -fsSL https://raw.githubusercontent.com/kujirahand/html2pdf-go/main/scripts/install.sh | sh
#   curl -fsSL .../install.sh | sh -s -- v0.1.0
#
# Environment:
#   HTML2PDF_VERSION      version tag to install (default: latest)
#   HTML2PDF_INSTALL_DIR  install directory (default: $HOME/.local/bin)
set -eu

REPO="kujirahand/html2pdf-go"

err() {
	echo "install.sh: $*" >&2
	exit 1
}

have() {
	command -v "$1" >/dev/null 2>&1
}

# fetch URL [OUTPUT]: write to OUTPUT, or to stdout when OUTPUT is omitted.
fetch() {
	if have curl; then
		if [ $# -ge 2 ]; then
			curl -fsSL -o "$2" "$1"
		else
			curl -fsSL "$1"
		fi
	elif have wget; then
		if [ $# -ge 2 ]; then
			wget -q -O "$2" "$1"
		else
			wget -q -O - "$1"
		fi
	else
		err "curl or wget is required"
	fi
}

have tar || err "tar is required"

version="${1:-${HTML2PDF_VERSION:-latest}}"

case "$(uname -s)" in
Linux) os=linux ;;
Darwin) os=darwin ;;
*) err "unsupported OS: $(uname -s) (Windows users should use install.ps1)" ;;
esac

case "$(uname -m)" in
x86_64 | amd64) arch=amd64 ;;
arm64 | aarch64) arch=arm64 ;;
*) err "unsupported architecture: $(uname -m)" ;;
esac

if [ "$version" = "latest" ]; then
	tag=$(fetch "https://api.github.com/repos/$REPO/releases/latest" |
		grep '"tag_name"' | head -n 1 |
		sed 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/') ||
		err "failed to query the latest release"
	[ -n "$tag" ] || err "could not determine the latest release tag"
else
	tag="$version"
fi

archive="html2pdf-go_${tag}_${os}_${arch}.tar.gz"
base="https://github.com/$REPO/releases/download/$tag"

tmp=$(mktemp -d) || err "failed to create a temporary directory"
trap 'rm -rf "$tmp"' EXIT INT TERM

echo "install.sh: downloading $archive"
fetch "$base/$archive" "$tmp/$archive" || err "failed to download $base/$archive"
fetch "$base/checksums.txt" "$tmp/checksums.txt" || err "failed to download $base/checksums.txt"

expected=$(awk -v f="$archive" '$2 == f || $2 == "*" f { print $1; exit }' "$tmp/checksums.txt")
[ -n "$expected" ] || err "$archive not found in checksums.txt"

if have sha256sum; then
	actual=$(sha256sum "$tmp/$archive" | awk '{ print $1 }')
elif have shasum; then
	actual=$(shasum -a 256 "$tmp/$archive" | awk '{ print $1 }')
else
	err "sha256sum or shasum is required"
fi

if [ "$expected" != "$actual" ]; then
	err "checksum mismatch for $archive (expected $expected, got $actual); nothing was installed"
fi
echo "install.sh: checksum OK"

mkdir -p "$tmp/extract"
tar -xzf "$tmp/$archive" -C "$tmp/extract" || err "failed to extract $archive"

dir="${HTML2PDF_INSTALL_DIR:-$HOME/.local/bin}"
mkdir -p "$dir" || err "cannot create $dir"

for bin in html2pdf md2pdf; do
	src=$(find "$tmp/extract" -type f -name "$bin" | head -n 1)
	[ -n "$src" ] || err "$bin not found in $archive"
	cp "$src" "$dir/$bin" || err "cannot write $dir/$bin"
	chmod +x "$dir/$bin"
	if [ "$os" = darwin ]; then
		xattr -d com.apple.quarantine "$dir/$bin" >/dev/null 2>&1 || true
	fi
done

echo "install.sh: installed html2pdf and md2pdf to $dir"
"$dir/html2pdf" -version 2>/dev/null || echo "install.sh: installed $tag"

case ":$PATH:" in
*":$dir:"*) ;;
*) echo "install.sh: $dir is not in your PATH. Add it with: export PATH=\"$dir:\$PATH\"" ;;
esac
