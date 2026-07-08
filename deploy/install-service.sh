#!/bin/bash
# Install ClaudeDeck as a macOS LaunchAgent: starts at login, restarts on crash,
# survives closing terminals. Idempotent — safe to re-run after a rebuild.
set -euo pipefail

DIR="$(cd "$(dirname "$0")/.." && pwd)"
BIN="$DIR/claude-deck"
PORT="${1:-7420}"
LABEL="com.claudedeck.agent"
PLIST="$HOME/Library/LaunchAgents/$LABEL.plist"
LOG="$HOME/Library/Logs/claude-deck.log"

if [ ! -x "$BIN" ]; then
  echo "building binary…"
  (cd "$DIR" && go build -o claude-deck .)
fi
mkdir -p "$HOME/Library/LaunchAgents" "$(dirname "$LOG")"

cat > "$PLIST" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
  <key>Label</key><string>$LABEL</string>
  <key>ProgramArguments</key><array>
    <string>$BIN</string><string>--port</string><string>$PORT</string>
  </array>
  <key>WorkingDirectory</key><string>$DIR</string>
  <key>RunAtLoad</key><true/>
  <key>KeepAlive</key><true/>
  <key>EnvironmentVariables</key><dict>
    <key>PATH</key><string>/usr/bin:/bin:/usr/sbin:/sbin:/usr/local/bin</string>
  </dict>
  <key>StandardOutPath</key><string>$LOG</string>
  <key>StandardErrorPath</key><string>$LOG</string>
</dict></plist>
EOF

# free the port from any manual (nohup) instance so the agent can bind it
pkill -f "claude-deck --port" 2>/dev/null || true
launchctl unload "$PLIST" 2>/dev/null || true
launchctl load -w "$PLIST"
sleep 1
echo "✓ loaded $LABEL → http://localhost:$PORT"
echo "  logs:      $LOG"
echo "  status:    launchctl list | grep claudedeck"
echo "  restart:   launchctl kickstart -k gui/\$(id -u)/$LABEL"
echo "  uninstall: deploy/uninstall-service.sh"
launchctl list | grep claudedeck || true
