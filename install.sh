#!/bin/sh
#
# Installs EncomPlayer from its GitHub releases.
#
#   curl -fsSL https://raw.githubusercontent.com/matjam/encomplayer/main/install.sh | sh
#
# Environment:
#   BINDIR   install directory (default: ~/.local/bin)
#   VERSION  release tag to install, e.g. v1.1.0 (default: latest)
#
# The archive is checked against the release's SHA256SUMS before anything
# is installed.

set -eu

REPO=matjam/encomplayer
BINDIR=${BINDIR:-$HOME/.local/bin}

die() {
	echo "encomplayer install: $*" >&2
	exit 1
}

need() {
	command -v "$1" >/dev/null 2>&1 || die "$1 is required"
}

need curl
need tar
need uname

case $(uname -s) in
Linux) os=linux ;;
Darwin) os=darwin ;;
*) die "unsupported OS $(uname -s); download a release from https://github.com/$REPO/releases" ;;
esac

case $(uname -m) in
x86_64 | amd64) arch=amd64 ;;
arm64 | aarch64) arch=arm64 ;;
*) die "unsupported CPU $(uname -m)" ;;
esac

if command -v sha256sum >/dev/null 2>&1; then
	sha256() { sha256sum "$1" | cut -d' ' -f1; }
elif command -v shasum >/dev/null 2>&1; then
	sha256() { shasum -a 256 "$1" | cut -d' ' -f1; }
else
	die "sha256sum or shasum is required to verify the download"
fi

# The latest release URL redirects to its tag, which avoids the GitHub API
# and its rate limits.
if [ -z "${VERSION:-}" ]; then
	url=$(curl -fsSLI -o /dev/null -w '%{url_effective}' "https://github.com/$REPO/releases/latest")
	VERSION=${url##*/}
fi
case $VERSION in
v*) ;;
*) VERSION=v$VERSION ;;
esac

file="encomplayer_${VERSION#v}_${os}_${arch}.tar.gz"
base="https://github.com/$REPO/releases/download/$VERSION"

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT INT TERM

echo "Downloading EncomPlayer $VERSION for $os/$arch"
curl -fsSL -o "$tmp/$file" "$base/$file" || die "no $file in release $VERSION"
curl -fsSL -o "$tmp/SHA256SUMS" "$base/SHA256SUMS" || die "release $VERSION has no SHA256SUMS"

want=$(grep " $file\$" "$tmp/SHA256SUMS" | cut -d' ' -f1)
[ -n "$want" ] || die "$file is not listed in SHA256SUMS"
[ "$(sha256 "$tmp/$file")" = "$want" ] || die "checksum mismatch for $file"

tar -xzf "$tmp/$file" -C "$tmp"

# Older releases wrap the binary in a directory; newer ones do not.
bin=$(find "$tmp" -type f -name encomplayer | head -n 1)
[ -n "$bin" ] || die "no encomplayer binary in $file"

mkdir -p "$BINDIR"
install -m 0755 "$bin" "$BINDIR/encomplayer"
echo "Installed $("$BINDIR/encomplayer" --version) to $BINDIR/encomplayer"

case ":$PATH:" in
*":$BINDIR:"*) ;;
*) echo "Add $BINDIR to your PATH to run encomplayer." ;;
esac

if ! command -v ffmpeg >/dev/null 2>&1; then
	echo "Optional: install ffmpeg to play AAC/M4A, ALAC, Opus, WavPack and more."
fi
