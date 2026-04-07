#!/usr/bin/env bash
set -euo pipefail

APP_DIR="/home/pi/better-music"

echo "==> updating repo"
cd "$APP_DIR"
git fetch origin
git reset --hard origin/main

echo "==> building backend"
cd "$APP_DIR/backend"
mkdir -p build
/usr/local/go/bin/go mod tidy
/usr/local/go/bin/go build -o build/better-music-backend .

echo "==> building frontend"
cd "$APP_DIR/frontend"
npm install
npm run build

echo "==> restarting services"
sudo systemctl restart better-music-backend
sudo systemctl reload caddy

echo "==> deploy complete"
