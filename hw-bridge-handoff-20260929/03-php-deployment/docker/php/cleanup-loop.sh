#!/bin/sh
set -eu
while true; do
  php application/imcallback/scripts/purge.php || true
  sleep 86400
done
