#!/usr/bin/env bash
# Cloud Agent start: ensure a virtual X display is available so the Wails desktop app
# (built binary or `wails dev`) can run and be screenshotted headlessly.
# Idempotent and best-effort: never fails the boot. Uses setsid so the display server
# detaches from this script's session and keeps running after start.sh returns.
set -uo pipefail

DISPLAY_NUM=":99"
LOG=/tmp/xvfb.log

if command -v xdpyinfo >/dev/null 2>&1 && xdpyinfo -display "$DISPLAY_NUM" >/dev/null 2>&1; then
  echo "Xvfb already running on ${DISPLAY_NUM}"
elif command -v Xvfb >/dev/null 2>&1; then
  echo "Starting Xvfb on ${DISPLAY_NUM} (log: ${LOG})"
  # setsid + nohup + </dev/null detaches Xvfb so it survives after start.sh exits.
  setsid nohup Xvfb "$DISPLAY_NUM" -screen 0 1600x1000x24 -nolisten tcp >"$LOG" 2>&1 </dev/null &
  for _ in 1 2 3 4 5 6 7 8; do
    if command -v xdpyinfo >/dev/null 2>&1 && xdpyinfo -display "$DISPLAY_NUM" >/dev/null 2>&1; then
      echo "Xvfb is up on ${DISPLAY_NUM}"
      break
    fi
    sleep 1
  done
else
  echo "Xvfb not installed; skipping virtual display startup" >&2
fi

echo "To run the app headlessly: DISPLAY=${DISPLAY_NUM} WEBKIT_DISABLE_COMPOSITING_MODE=1 WEBKIT_DISABLE_DMABUF_RENDERER=1 <app-or-wails-dev>"
exit 0
