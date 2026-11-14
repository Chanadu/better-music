#!/usr/bin/env bash
set -euo pipefail

repo=$(cd "$(dirname "$0")/../.." && pwd)
output="$repo/client/packages"
work=$(mktemp -d)
trap 'rm -rf -- "$work"' EXIT

mkdir -p "$output"
install -m755 "$repo/client/better-music" "$work/better-music-bin"
install -m644 "$repo/deploy/arch/"{PKGBUILD,better-music.desktop} "$work/"
install -m644 "$repo/frontend/static/app-icon.svg" "$repo/LICENSE" "$work/"
cd "$work"
makepkg --force --noconfirm --nodeps --geninteg > checksums
sed -i '/^sha256sums=/d' PKGBUILD
cat checksums >> PKGBUILD
PKGDEST="$output" makepkg --force --noconfirm
