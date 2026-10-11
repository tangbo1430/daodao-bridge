#!/bin/sh
set -eu
umask 000

runtime=/var/www/app/runtime
uploads=/var/www/app/public/uploads
mkdir -p "$runtime/cache" "$runtime/log" "$runtime/temp"
mkdir -p "$uploads"

index=0
while [ "$index" -lt 256 ]; do
  prefix=$(printf '%02x' "$index")
  mkdir -p "$runtime/cache/$prefix"
  index=$((index + 1))
done

chmod -R a+rwX "$runtime"
chmod -R a+rwX "$uploads"
php /opt/app-preflight.php
exec /usr/local/bin/docker-php-entrypoint "$@"
