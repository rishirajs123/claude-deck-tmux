#!/usr/bin/env bash
# One-line installer for ClaudeDeck (macOS).
#   curl -fsSL https://raw.githubusercontent.com/tachodril/claude-deck/main/install.sh | bash
#   ... | bash -s -- --service     # also install the always-on launchd agent
#
# Wrapped in main() so a partial download from curl|bash never executes.
set -euo pipefail
main() {
  REPO="tachodril/claude-deck"
  INSTALL_DIR="$HOME/.local/bin"
  BASE_URL="${CD_DOWNLOAD_URL:-https://github.com/${REPO}/releases/latest/download}"
  WITH_SERVICE=false
  for a in "$@"; do [ "$a" = "--service" ] && WITH_SERVICE=true; done

  case "$(uname -s)" in Darwin) ;; *) echo "error: ClaudeDeck currently supports macOS only." >&2; exit 1 ;; esac
  case "$(uname -m)" in
    arm64) ARCH=arm64 ;;
    x86_64) ARCH=amd64 ;;
    *) echo "error: unsupported arch $(uname -m)" >&2; exit 1 ;;
  esac

  ARCHIVE="claude-deck-darwin-${ARCH}.tar.gz"
  URL="${BASE_URL}/${ARCHIVE}"
  TMP="$(mktemp -d)"; trap 'rm -rf "$TMP"' EXIT

  echo "downloading $ARCHIVE …"
  curl -fSL --progress-bar -o "$TMP/$ARCHIVE" "$URL"

  if curl -fsSL -o "$TMP/checksums.txt" "${BASE_URL}/checksums.txt" 2>/dev/null; then
    EXP="$(grep "$ARCHIVE" "$TMP/checksums.txt" | awk '{print $1}')"
    ACT="$(shasum -a 256 "$TMP/$ARCHIVE" | awk '{print $1}')"
    if [ -n "$EXP" ] && [ "$EXP" != "$ACT" ]; then
      echo "error: checksum mismatch" >&2; exit 1
    fi
    [ -n "$EXP" ] && echo "checksum verified."
  fi

  tar -xzf "$TMP/$ARCHIVE" -C "$TMP"
  mkdir -p "$INSTALL_DIR"
  install -m 0755 "$TMP/claude-deck" "$INSTALL_DIR/claude-deck"
  xattr -d com.apple.quarantine "$INSTALL_DIR/claude-deck" 2>/dev/null || true
  codesign --sign - --force "$INSTALL_DIR/claude-deck" 2>/dev/null || true

  echo "✓ installed: $("$INSTALL_DIR/claude-deck" --version)"
  echo "  → $INSTALL_DIR/claude-deck"
  case ":$PATH:" in *":$INSTALL_DIR:"*) ;; *) echo "  NOTE: add to PATH → echo 'export PATH=\"$INSTALL_DIR:\$PATH\"' >> ~/.zshrc" ;; esac

  if [ "$WITH_SERVICE" = true ]; then
    echo "installing launchd agent…"
    LABEL="com.claudedeck.agent"; PLIST="$HOME/Library/LaunchAgents/$LABEL.plist"; LOG="$HOME/Library/Logs/claude-deck.log"
    mkdir -p "$HOME/Library/LaunchAgents" "$(dirname "$LOG")"
    cat > "$PLIST" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
  <key>Label</key><string>$LABEL</string>
  <key>ProgramArguments</key><array><string>$INSTALL_DIR/claude-deck</string><string>--port</string><string>7420</string><string>--no-open</string></array>
  <key>RunAtLoad</key><true/><key>KeepAlive</key><true/>
  <key>EnvironmentVariables</key><dict><key>PATH</key><string>/usr/bin:/bin:/usr/sbin:/sbin:/usr/local/bin</string></dict>
  <key>StandardOutPath</key><string>$LOG</string><key>StandardErrorPath</key><string>$LOG</string>
</dict></plist>
EOF
    launchctl unload "$PLIST" 2>/dev/null || true
    launchctl load -w "$PLIST"
    echo "✓ always-on at http://localhost:7420"
  else
    echo "run it:          claude-deck"
    echo "keep it running: claude-deck --install-service   (auto-starts at login)"
  fi
}
main "$@"
