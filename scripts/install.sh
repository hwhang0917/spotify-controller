#!/bin/sh
# Installs vibe-music from a GitHub release for the current machine.
#
#   curl -fsSL https://raw.githubusercontent.com/hwhang0917/vibe-music/master/scripts/install.sh | sh
#
# Env:
#   VIBE_MUSIC_VERSION  release to install, e.g. 0.0.1 (default: latest)
#   INSTALL_DIR         where to put it (Linux: ~/.local/bin, macOS: ~/Applications)
set -eu

REPO="${VIBE_MUSIC_REPO:-hwhang0917/vibe-music}"
VERSION="${VIBE_MUSIC_VERSION:-latest}"

# Asset names come from the build matrix in .github/workflows/release.yml.
os=$(uname -s)
arch=$(uname -m)
case "$os/$arch" in
  Linux/x86_64)  asset=vibe-music-linux-amd64.tar.gz; dir="${INSTALL_DIR:-$HOME/.local/bin}" ;;
  Darwin/x86_64) asset=vibe-music-darwin-amd64.zip;   dir="${INSTALL_DIR:-$HOME/Applications}" ;;
  *)
    echo "vibe-music: unsupported platform $os/$arch; releases cover linux/amd64 and darwin/amd64 only." >&2
    echo "Windows users: download vibe-music-windows-amd64.exe from https://github.com/$REPO/releases" >&2
    exit 1 ;;
esac

if [ "$VERSION" = latest ]; then
  url="https://github.com/$REPO/releases/latest/download/$asset"
else
  url="https://github.com/$REPO/releases/download/v${VERSION#v}/$asset"
fi

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

echo "Downloading $url"
curl -fsSL --retry 3 -o "$tmp/$asset" "$url" || { echo "vibe-music: download failed: $url" >&2; exit 1; }

mkdir -p "$dir"
case "$asset" in
  *.tar.gz)
    tar xzf "$tmp/$asset" -C "$dir"
    chmod +x "$dir/vibe-music"
    echo "Installed $dir/vibe-music"
    case ":$PATH:" in *":$dir:"*) ;; *) echo "note: $dir is not on your PATH" ;; esac ;;
  *.zip)
    rm -rf "$dir/vibe-music.app"
    ditto -x -k "$tmp/$asset" "$dir"
    echo "Installed $dir/vibe-music.app" ;;
esac
