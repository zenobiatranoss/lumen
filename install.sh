#!/usr/bin/env bash
set -e
cd "$(dirname "$0")"
command -v go >/dev/null || { echo "Go is not installed: sudo apt install golang"; exit 1; }
go build -trimpath -ldflags "-s -w" -o lumen .
install -Dm755 lumen "$HOME/.local/bin/lumen"
install -Dm644 packaging/window.py "$HOME/.local/share/lumen/window.py"
python3 "$HOME/.local/share/lumen/window.py" --check 2>/dev/null || echo "Native window needs: sudo apt install python3-gi gir1.2-webkit2-4.1 (falling back to Chromium until then)"
install -Dm644 packaging/lumen.svg "$HOME/.local/share/icons/hicolor/scalable/apps/lumen.svg"
install -Dm644 web/icon.png "$HOME/.local/share/icons/hicolor/256x256/apps/lumen.png"
mkdir -p "$HOME/.local/share/applications"
cat > "$HOME/.local/share/applications/lumen.desktop" <<DESKTOP
[Desktop Entry]
Type=Application
Name=Lumen
Comment=Calm image-first file browser
Exec=$HOME/.local/bin/lumen %f
Icon=lumen
StartupWMClass=lumen
Terminal=false
Categories=Graphics;Utility;FileManager;
MimeType=inode/directory;image/jpeg;image/png;image/webp;image/gif;video/mp4;
DESKTOP
update-desktop-database "$HOME/.local/share/applications" 2>/dev/null || true
gtk-update-icon-cache -f -t "$HOME/.local/share/icons/hicolor" 2>/dev/null || true
kbuildsycoca6 >/dev/null 2>&1 || kbuildsycoca5 >/dev/null 2>&1 || true
echo "Installed. Run: lumen   (or lumen ~/Pictures)"
case ":$PATH:" in *":$HOME/.local/bin:"*) ;; *) echo "Add ~/.local/bin to your PATH to run it from the terminal." ;; esac
