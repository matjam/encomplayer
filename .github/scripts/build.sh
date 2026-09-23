#!/usr/bin/env bash
#
# Builds a static binary and packages it for one platform.
#
# Usage: build.sh <goos> <goarch> <version>
# Writes dist/encomplayer_<version>_<goos>_<goarch>.{tar.gz,zip} and prints
# the archive path.

set -euo pipefail

goos="${1:?goos}" goarch="${2:?goarch}" version="${3:?version}"
name="encomplayer_${version#v}_${goos}_${goarch}"
stage="dist/$name"
bin="encomplayer"
[[ "$goos" == windows ]] && bin="encomplayer.exe"

rm -rf "$stage"
mkdir -p "$stage"
CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" go build \
	-trimpath \
	-ldflags "-s -w -X main.version=$version" \
	-o "$stage/$bin" \
	./cmd/encomplayer
cp README.md "$stage/"

if [[ "$goos" == windows ]]; then
	(cd dist && zip -qr "$name.zip" "$name")
	echo "dist/$name.zip"
else
	tar -C dist -czf "dist/$name.tar.gz" "$name"
	echo "dist/$name.tar.gz"
fi
