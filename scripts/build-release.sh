#!/bin/sh
# Cross-compile both CLIs and package them into dist/.
# Usage: scripts/build-release.sh [version]   (version defaults to "dev")
set -eu

version="${1:-dev}"
# A tag may contain '/' (e.g. release/v1.0.0) and still match the
# workflow's v* trigger. Keep the original string for the embedded
# version, but never let it become a path separator in archive names.
filever=$(printf '%s' "$version" | tr '/' '-')
root=$(cd "$(dirname "$0")/.." && pwd)
cd "$root"

targets="linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64"
cmds="html2pdf md2pdf"

rm -rf dist
mkdir -p dist
dist="$root/dist"

for target in $targets; do
	os=${target%/*}
	arch=${target#*/}
	name="html2pdf-go_${filever}_${os}_${arch}"
	ext=""
	[ "$os" = windows ] && ext=".exe"

	stage=$(mktemp -d)
	for c in $cmds; do
		echo "build $c ($os/$arch)"
		CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" go build -trimpath \
			-ldflags "-s -w -X main.version=$version" \
			-o "$stage/$c$ext" "./cmd/$c"
	done
	cp README.md README-ja.md LICENSE "$stage/"

	if [ "$os" = windows ]; then
		(cd "$stage" && zip -q -r "$dist/$name.zip" .)
	else
		tar -C "$stage" -czf "$dist/$name.tar.gz" .
	fi
	rm -rf "$stage"
done

cd "$dist"
if command -v sha256sum >/dev/null 2>&1; then
	sha256sum *.tar.gz *.zip > checksums.txt
else
	shasum -a 256 *.tar.gz *.zip > checksums.txt
fi
echo "done: $dist"
