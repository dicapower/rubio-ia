#!/bin/sh
# installs or updates rubio from the latest GitHub release.
#
#   curl -fsSL https://raw.githubusercontent.com/dicapower/rubio-ia/main/install.sh | sh
#
# Run it again to upgrade. It verifies the SHA-256 checksum of the download and
# never uses sudo. Settings (environment variables):
#   RUBIO_VERSION      tag to install, e.g. v0.1.0 (default: latest release)
#   RUBIO_INSTALL_DIR  folder for the binary (default: where `rubio` already is,
#                     else /usr/local/bin if writable, else ~/.local/bin)
set -eu

REPO="dicapower/rubio-ia"

say() { printf '%s\n' "$*"; }
die() { printf 'rubio install: %s\n' "$*" >&2; exit 1; }
have() { command -v "$1" >/dev/null 2>&1; }

have curl || die "curl is required"
have tar || die "tar is required"

case "$(uname -s)" in
  Linux) os=linux ;;
  Darwin) os=darwin ;;
  *) die "unsupported system $(uname -s). On Windows use Scoop (see the README) or download the .zip from https://github.com/$REPO/releases" ;;
esac
case "$(uname -m)" in
  x86_64|amd64) arch=amd64 ;;
  arm64|aarch64) arch=arm64 ;;
  *) die "unsupported architecture $(uname -m)" ;;
esac

tag="${RUBIO_VERSION:-}"
if [ -z "$tag" ]; then
  url=$(curl -fsSLI -o /dev/null -w '%{url_effective}' "https://github.com/$REPO/releases/latest") \
    || die "could not reach GitHub"
  tag="${url##*/}"
fi
case "$tag" in v[0-9]*) ;; *) die "could not work out the latest version (got '$tag')" ;; esac

name="rubio-$tag-$os-$arch"
base="https://github.com/$REPO/releases/download/$tag"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT INT TERM

say "Downloading rubio $tag ($os/$arch)..."
curl -fsSL -o "$tmp/$name.tar.gz" "$base/$name.tar.gz" \
  || die "download failed: $base/$name.tar.gz (is $tag a published release with binaries?)"
curl -fsSL -o "$tmp/SHA256SUMS" "$base/SHA256SUMS" || die "could not download SHA256SUMS"

want=$(grep "  $name.tar.gz\$" "$tmp/SHA256SUMS" | cut -d' ' -f1)
[ -n "$want" ] || die "$name.tar.gz is not listed in SHA256SUMS"
if have sha256sum; then got=$(sha256sum "$tmp/$name.tar.gz" | cut -d' ' -f1)
elif have shasum; then got=$(shasum -a 256 "$tmp/$name.tar.gz" | cut -d' ' -f1)
else die "sha256sum or shasum is required to verify the download"; fi
[ "$got" = "$want" ] || die "checksum mismatch, nothing installed"

tar -xzf "$tmp/$name.tar.gz" -C "$tmp"
[ -x "$tmp/$name/rubio" ] || die "the archive does not contain the rubio binary"

# Where to put it.
dir="${RUBIO_INSTALL_DIR:-}"
if [ -z "$dir" ] && have rubio; then
  existing=$(command -v rubio)
  d=$(dirname "$existing")
  if [ -w "$d" ]; then dir="$d"; else
    say "Found rubio in $d, which you cannot write to. Installing in a user folder instead."
  fi
fi
if [ -z "$dir" ]; then
  if [ -d /usr/local/bin ] && [ -w /usr/local/bin ]; then dir=/usr/local/bin
  else dir="$HOME/.local/bin"; fi
fi
mkdir -p "$dir" || die "cannot create $dir"
[ -w "$dir" ] || die "cannot write to $dir (set RUBIO_INSTALL_DIR to a folder you own)"

# Replace atomically so a running agent keeps its old copy until it restarts.
cp "$tmp/$name/rubio" "$dir/.rubio.new" && chmod 755 "$dir/.rubio.new" && mv -f "$dir/.rubio.new" "$dir/rubio"

say "Installed $("$dir/rubio" version) in $dir"
case ":$PATH:" in
  *":$dir:"*) ;;
  *) say ""
     say "$dir is not in your PATH. Add it with:"
     say "  echo 'export PATH=\"$dir:\$PATH\"' >> ~/.zshrc && . ~/.zshrc    (use ~/.bashrc for bash)" ;;
esac
say ""
say "Next: rubio setup (first time), then restart your agent. Then run: rubio setup, then rubio doctor"
