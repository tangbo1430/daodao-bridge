#!/bin/sh
set -eu

BASE_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
ZIP_FILE="$BASE_DIR/YCH-IM-deploy.zip"
APP_DIR="$BASE_DIR/YCH-IM-deploy"
UNPACK_DIR="$BASE_DIR/.deploy-unpack"

cleanup() {
  rm -rf "$UNPACK_DIR"
}
trap cleanup EXIT INT TERM

if ! command -v docker >/dev/null 2>&1 || ! docker compose version >/dev/null 2>&1; then
  echo "Docker Engine and Docker Compose Plugin are required."
  exit 1
fi
if [ ! -f "$ZIP_FILE" ]; then
  echo "Missing $ZIP_FILE"
  exit 1
fi
if [ ! -f "$BASE_DIR/.env" ]; then
  echo "Copy .env.example to .env and fill in the required values."
  exit 1
fi
if grep -q 'CHANGE_ME' "$BASE_DIR/.env"; then
  echo "Replace all CHANGE_ME values in .env before deployment."
  exit 1
fi

rm -rf "$UNPACK_DIR"
mkdir -p "$UNPACK_DIR"
if ! unzip -q "$ZIP_FILE" -d "$UNPACK_DIR"; then
  echo "Failed to unpack $ZIP_FILE. Upload the latest Linux-compatible deployment ZIP."
  exit 1
fi
if [ ! -f "$UNPACK_DIR/YCH-IM-deploy/docker-compose.yml" ]; then
  echo "Invalid deployment package: YCH-IM-deploy/docker-compose.yml is missing."
  exit 1
fi

rm -rf "$APP_DIR"
mv "$UNPACK_DIR/YCH-IM-deploy" "$APP_DIR"
cp "$BASE_DIR/.env" "$APP_DIR/.env"
mkdir -p "$APP_DIR/backend/runtime"
chmod -R 0777 "$APP_DIR/backend/runtime"

cd "$APP_DIR"
docker compose pull
docker compose build --pull
docker compose up -d --force-recreate
sleep 8
docker compose ps
echo "Deployment completed: https://im.awhaha.com and https://admin.awhaha.com"
