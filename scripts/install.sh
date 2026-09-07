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
    # The binary links WebKitGTK 4.1 dynamically; say so now rather than at launch.
    missing=$(ldd "$dir/vibe-music" | awk '/not found/ {print $1}')
    if [ -n "$missing" ]; then
      echo "vibe-music: installed $dir/vibe-music, but these shared libraries are missing:" >&2
      echo "$missing" | sed 's/^/  /' >&2
      echo "Install WebKitGTK 4.1 and rerun vibe-music:" >&2
      echo "  Debian/Ubuntu: sudo apt install libwebkit2gtk-4.1-0" >&2
      echo "  Fedora:        sudo dnf install webkit2gtk4.1" >&2
      echo "  Arch:          sudo pacman -S webkit2gtk-4.1" >&2
      exit 1
    fi
    echo "Installed $dir/vibe-music"
    case ":$PATH:" in *":$dir:"*) ;; *) echo "note: $dir is not on your PATH" ;; esac
    # Desktop entry so app launchers (GNOME, KDE, wlroots menus) can start it.
    data="${XDG_DATA_HOME:-$HOME/.local/share}"
    mkdir -p "$data/applications" "$data/vibe-music"
    curl -fsSL -o "$data/vibe-music/icon.png" "https://raw.githubusercontent.com/$REPO/master/build/appicon.png" \
      || echo "note: icon download failed; the launcher entry will have no icon" >&2
    cat > "$data/applications/vibe-music.desktop" <<DESKTOP
[Desktop Entry]
Type=Application
Name=vibe-music
Comment=Office jukebox with votes
Exec="$dir/vibe-music"
Icon=$data/vibe-music/icon.png
Terminal=false
Categories=AudioVideo;Audio;Player;
StartupWMClass=vibe-music
DESKTOP
    command -v update-desktop-database >/dev/null 2>&1 && update-desktop-database "$data/applications" 2>/dev/null
    echo "Desktop entry: $data/applications/vibe-music.desktop" ;;
  *.zip)
    rm -rf "$dir/vibe-music.app"
    ditto -x -k "$tmp/$asset" "$dir"
    echo "Installed $dir/vibe-music.app" ;;
esac
