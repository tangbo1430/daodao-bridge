#!/bin/sh
set -eu

cd "$(dirname "$0")"

if ! command -v docker >/dev/null 2>&1; then
  echo "Docker is not installed. Install Docker Engine and Compose plugin first."
  exit 1
fi
if ! docker compose version >/dev/null 2>&1; then
  echo "Docker Compose plugin is not installed."
  exit 1
fi
if [ ! -f .env ]; then
  cp .env.example .env
  echo "Created .env. Fill in all CHANGE_ME values, then run ./starh.sh again."
  exit 1
fi
if grep -q 'CHANGE_ME' .env; then
  echo "Replace all CHANGE_ME values in .env before deployment."
  exit 1
fi
if [ ! -f database/init/00_legacy.sql ] && [ ! -f database/init/00_legacy.sql.gz ]; then
  echo "Place the existing business database backup at database/init/00_legacy.sql(.gz)."
  exit 1
fi

mkdir -p backend/runtime
chmod -R 0777 backend/runtime

docker compose pull
docker compose build --pull
docker compose up -d

echo "Waiting for services..."
sleep 8
docker compose ps
echo "Deployment completed: https://im.awhaha.com and https://admin.awhaha.com"
