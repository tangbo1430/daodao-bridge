#!/usr/bin/env sh
set -eu

: "${CALLBACK_URL:?CALLBACK_URL is required}"
: "${IM_CALLBACK_TOKEN:?IM_CALLBACK_TOKEN is required}"

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
FIXTURE="$SCRIPT_DIR/fixtures/c2c_after_send.json"

curl --fail-with-body \
  --silent \
  --show-error \
  --request POST \
  --header 'Content-Type: application/json' \
  --data-binary "@$FIXTURE" \
  "${CALLBACK_URL}?token=${IM_CALLBACK_TOKEN}"

printf '\n'
