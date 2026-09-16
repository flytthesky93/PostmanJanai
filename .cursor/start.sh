#!/usr/bin/env bash
# Cloud Agent start: ensure a virtual X display + D-Bus session are available so the
# Wails desktop app (built binary or `wails dev`) can run and be screenshotted headlessly.
# Idempotent: skips startup if a display is already serving :99.
set -euo pipefail

DISPLAY_NUM=":99"

if xdpyinfo -display "$DISPLAY_NUM" >/dev/null 2>&1; then
  echo "Xvfb already running on ${DISPLAY_NUM}"
else
  echo "Starting Xvfb on ${DISPLAY_NUM}"
  Xvfb "$DISPLAY_NUM" -screen 0 1600x1000x24 -nolisten tcp >/tmp/xvfb.log 2>&1 &
  # Give Xvfb a moment to come up; do not fail the boot if it is slow.
  for _ in 1 2 3 4 5; do
    if xdpyinfo -display "$DISPLAY_NUM" >/dev/null 2>&1; then break; fi
    sleep 1
  done
fi

echo "Display ready. Use: DISPLAY=${DISPLAY_NUM} WEBKIT_DISABLE_COMPOSITING_MODE=1 WEBKIT_DISABLE_DMABUF_RENDERER=1 <app-or-wails-dev>"
