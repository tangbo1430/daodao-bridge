param(
    [Parameter(Mandatory=$false)][string]$ServerHost,
    [string]$ServerUser = "root"
)

$ErrorActionPreference = "Stop"
$bundle = "F:\IM\release\bridge-server-bundle-v6.zip"
if (-not (Test-Path $bundle)) { throw "Bundle not found: $bundle" }
if ([string]::IsNullOrWhiteSpace($ServerHost)) { $ServerHost = Read-Host "Server IP or hostname" }

$remoteBundle = "/root/IM/bridge-server-bundle.zip"
Write-Host "Uploading bridge bundle..."
scp -q $bundle "$ServerUser@$ServerHost`:$remoteBundle"
if ($LASTEXITCODE -ne 0) { throw "scp failed" }

$remoteScript = @'
set -euo pipefail
DEPLOY=/root/IM/YCH-IM-deploy
RUNTIME=/root/IM/bridge-runtime
STAMP=$(date +%Y%m%d%H%M%S)
BACKUP=/root/IM/backup-bridge-$STAMP
mkdir -p "$BACKUP"
cp -a "$DEPLOY/backend/application/imcallback/service/WorkerService.php" "$BACKUP/" 2>/dev/null || true
cp -a "$DEPLOY/backend/application/imcallback/service/BridgeDispatchService.php" "$BACKUP/" 2>/dev/null || true
cp -a "$DEPLOY/backend/config/im_bridge.php" "$BACKUP/" 2>/dev/null || true
cp -a "$DEPLOY/backend/application/imaccess/service/BridgeService.php" "$BACKUP/" 2>/dev/null || true
cp -a "$DEPLOY/backend/application/imaccess/controller/Bridge.php" "$BACKUP/" 2>/dev/null || true
cp -a "$DEPLOY/backend/route/route.php" "$BACKUP/" 2>/dev/null || true
cp -a "$DEPLOY/.env" "$BACKUP/env" 2>/dev/null || true
cp -a "$DEPLOY/docker-compose.yml" "$BACKUP/" 2>/dev/null || true
rm -rf "$RUNTIME"
mkdir -p "$RUNTIME"
unzip -q -o /root/IM/bridge-server-bundle.zip -d "$RUNTIME"
cp "$RUNTIME/php-patch/application/imcallback/service/BridgeDispatchService.php" "$DEPLOY/backend/application/imcallback/service/"
cp "$RUNTIME/php-patch/application/imcallback/service/WorkerService.php" "$DEPLOY/backend/application/imcallback/service/"
cp "$RUNTIME/php-patch/config/im_bridge.php" "$DEPLOY/backend/config/"
mkdir -p "$DEPLOY/backend/application/imaccess/service" "$DEPLOY/backend/application/imaccess/controller"
cp "$RUNTIME/php-patch/application/imaccess/service/BridgeService.php" "$DEPLOY/backend/application/imaccess/service/"
cp "$RUNTIME/php-patch/application/imaccess/controller/Bridge.php" "$DEPLOY/backend/application/imaccess/controller/"
cp "$RUNTIME/php-patch/route/route.php" "$DEPLOY/backend/route/"
cp "$RUNTIME/php-patch/database/07_im_bridge_delivery.sql" "$DEPLOY/database/migrations/"
if ! grep -q '^BRIDGE_SHARED_SECRET=' "$DEPLOY/.env"; then
  SECRET=$(openssl rand -hex 32)
  DBPASS=$(openssl rand -hex 16)
  DBROOT=$(openssl rand -hex 16)
  cat >> "$DEPLOY/.env" <<EOF
IM_BRIDGE_ENABLED=true
IM_BRIDGE_URL=http://host.docker.internal:8090
IM_BRIDGE_PATH=/cb/tim
IM_BRIDGE_SHARED_SECRET=$SECRET
BRIDGE_SHARED_SECRET=$SECRET
BRIDGE_STORE=mysql
BRIDGE_AUTO_MIGRATE=true
BRIDGE_DB_PASSWORD=$DBPASS
BRIDGE_DB_ROOT_PASSWORD=$DBROOT
TANGSENG_BASE_URL=http://10.0.0.100:8090
TANGSENG_LOGIN_ENDPOINT=/v1/user/usernamelogin
TANGSENG_WS_URL=ws://10.0.0.100:5200
TANGSENG_SHADOW_PASSWORD=__SET_TANGSENG_SHADOW_PASSWORD__
TANGSENG_MANUFACTURER=bridge
BRIDGE_PEER_ID=64833
EOF
fi
grep -q '^IM_BRIDGE_TEST_USERS=' "$DEPLOY/.env" || echo 'IM_BRIDGE_TEST_USERS=64833' >> "$DEPLOY/.env"
grep -q '^IM_BRIDGE_PEER_ID=' "$DEPLOY/.env" || echo 'IM_BRIDGE_PEER_ID=64833' >> "$DEPLOY/.env"
SECRET=$(grep '^BRIDGE_SHARED_SECRET=' "$DEPLOY/.env" | tail -1 | cut -d= -f2-)
DBPASS=$(grep '^BRIDGE_DB_PASSWORD=' "$DEPLOY/.env" | tail -1 | cut -d= -f2-)
DBROOT=$(grep '^BRIDGE_DB_ROOT_PASSWORD=' "$DEPLOY/.env" | tail -1 | cut -d= -f2-)
cat > "$RUNTIME/.env.bridge" <<EOF
BRIDGE_LISTEN_ADDR=:8090
BRIDGE_SHARED_SECRET=$SECRET
BRIDGE_STORE=mysql
BRIDGE_AUTO_MIGRATE=true
BRIDGE_DB_PASSWORD=$DBPASS
BRIDGE_DB_ROOT_PASSWORD=$DBROOT
TENCENT_SDK_APP_ID=$(grep '^IM_SDK_APP_ID=' "$DEPLOY/.env" | tail -1 | cut -d= -f2-)
TENCENT_SECRET_KEY=$(grep '^IM_SDK_SECRET_KEY=' "$DEPLOY/.env" | tail -1 | cut -d= -f2-)
TENCENT_ADMIN_ACCOUNT=$(grep '^IM_ADMIN_ACCOUNT=' "$DEPLOY/.env" | tail -1 | cut -d= -f2-)
TENCENT_REST_BASE_URL=https://adminapisgp.im.qcloud.com
TANGSENG_BASE_URL=http://10.0.0.100:8090
TANGSENG_LOGIN_ENDPOINT=/v1/user/usernamelogin
TANGSENG_WS_URL=ws://10.0.0.100:5200
TANGSENG_SHADOW_PASSWORD=__SET_TANGSENG_SHADOW_PASSWORD__
TANGSENG_MANUFACTURER=bridge
BRIDGE_PEER_ID=64833
EOF
cat > "$DEPLOY/docker-compose.override.yml" <<EOF
services:
  php:
    extra_hosts:
      - "host.docker.internal:host-gateway"
  worker:
    extra_hosts:
      - "host.docker.internal:host-gateway"
EOF
cd "$RUNTIME"
docker compose --env-file .env.bridge -f docker-compose.bridge.yml up -d --build
cd "$DEPLOY"
docker compose run --rm migrate
docker compose up -d --build php worker
curl -fsS http://127.0.0.1:8090/healthz
printf '\nDEPLOY_OK backup=%s\n' "$BACKUP"
'@

$encoded = [Convert]::ToBase64String([Text.Encoding]::UTF8.GetBytes($remoteScript))
ssh "$ServerUser@$ServerHost" "echo $encoded | base64 -d | bash"
if ($LASTEXITCODE -ne 0) { throw "remote deploy failed" }
Write-Host "Deployment completed. Original files are backed up on the server."


