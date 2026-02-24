#!/usr/bin/env bash
set -euo pipefail

BIN="${HOME}/.local/bin"
SHARE="${HOME}/.local/share"

install -Dm755 goop                                    "$BIN/goop"
install -Dm644 org.codeberg.sigterm-de.goop.desktop    "$SHARE/applications/org.codeberg.sigterm-de.goop.desktop"
install -Dm644 goop.png                                "$SHARE/icons/hicolor/256x256/apps/org.codeberg.sigterm-de.goop.png"
install -Dm644 goop-drop.svg                           "$SHARE/icons/hicolor/scalable/apps/org.codeberg.sigterm-de.goop.svg"
update-desktop-database "$SHARE/applications"          2>/dev/null || true
gtk-update-icon-cache -t "$SHARE/icons/hicolor"        2>/dev/null || true

echo "goop installed → $BIN/goop"
