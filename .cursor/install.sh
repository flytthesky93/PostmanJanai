#!/usr/bin/env bash
# Cloud Agent install: idempotent bootstrap for the PostmanJanai Wails desktop app.
# Runs after the repository is checked out. Safe to re-run.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

WAILS_VERSION="v2.11.0"

echo "==> Installing system libraries (GTK3 + WebKit2GTK 4.1 for Wails on Linux)"
export DEBIAN_FRONTEND=noninteractive
sudo apt-get update -y
# libgtk-3-dev + libwebkit2gtk-4.1-dev: native deps Wails links against on Linux.
# xvfb / dbus-x11 / imagemagick / mesa: run and screenshot the GUI headlessly.
sudo apt-get install -y --no-install-recommends \
  build-essential \
  pkg-config \
  libgtk-3-dev \
  libwebkit2gtk-4.1-dev \
  xvfb \
  dbus-x11 \
  x11-utils \
  xdotool \
  imagemagick \
  libgl1-mesa-dri

echo "==> Installing Wails CLI ${WAILS_VERSION}"
go install "github.com/wailsapp/wails/v2/cmd/wails@${WAILS_VERSION}"

echo "==> Downloading Go modules"
go mod download

echo "==> Installing frontend dependencies (npm ci)"
cd "$REPO_ROOT/frontend"
npm ci

echo "==> Building frontend (produces frontend/dist embedded by main.go)"
npm run build

echo "==> Install complete"
