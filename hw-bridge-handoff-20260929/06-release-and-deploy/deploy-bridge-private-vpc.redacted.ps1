param(
    [string]$ServerHost = "101.32.11.27",
    [string]$ServerUser = "ubuntu",
    [switch]$PackageOnly,
    [switch]$DeployExisting,
    [string]$SourceRoot = (Join-Path $PSScriptRoot '..\01-bridge\tencent-tangseng-bridge'),
    [string]$GoExecutable = 'go',
    [string]$SshKeyPath = $(if ($env:HW_BRIDGE_SSH_KEY) { $env:HW_BRIDGE_SSH_KEY } else { "" })
)

$ErrorActionPreference = "Stop"
$sshOptions = @()
if ($SshKeyPath) {
    try {
        if (Test-Path -LiteralPath $SshKeyPath -ErrorAction Stop) {
            $sshOptions = @("-i", $SshKeyPath)
        }
    } catch {
        # An inaccessible optional key should not prevent ssh-agent or the
        # user's default key from being used.
    }
}
$project = (Resolve-Path -LiteralPath $SourceRoot).Path
$bundle = Join-Path $PSScriptRoot "hw-bridge-private-vpc.tar.gz"
$stage = Join-Path $env:TEMP ("hw-bridge-bundle-" + [guid]::NewGuid().ToString("N"))

function Remove-Utf8Bom([string]$Path) {
    $content = [System.IO.File]::ReadAllText($Path) -replace "`r`n", "`n"
    [System.IO.File]::WriteAllText($Path, $content, (New-Object System.Text.UTF8Encoding($false)))
}

if (-not (Test-Path (Join-Path $project "go.mod"))) {
    throw "Bridge source was not found: $project"
}
if (-not (Get-Command $GoExecutable -ErrorAction SilentlyContinue)) {
    throw "Go 1.22 is required to build the Bridge binary."
}
if (-not (Get-Command tar -ErrorAction SilentlyContinue)) {
    throw "Windows tar is required. Windows 10/11 includes tar.exe."
}

if (!$DeployExisting) {
New-Item -ItemType Directory -Force $stage | Out-Null
try {
    Write-Host "Building Linux Bridge binary..."
    $oldGoCache = $env:GOCACHE
    if (!$env:GOCACHE) { $env:GOCACHE = Join-Path $stage "gocache" }
    New-Item -ItemType Directory -Force $env:GOCACHE | Out-Null
    Push-Location $project
    try {
        $env:GOOS = "linux"
        $env:GOARCH = "amd64"
        $env:CGO_ENABLED = "0"
        & $GoExecutable build -trimpath -ldflags "-s -w" -o (Join-Path $stage "bridge") ./cmd/bridge
        if ($LASTEXITCODE -ne 0) { throw "go build failed" }
    } finally {
        Pop-Location
        Remove-Item Env:GOOS -ErrorAction SilentlyContinue
        Remove-Item Env:GOARCH -ErrorAction SilentlyContinue
        Remove-Item Env:CGO_ENABLED -ErrorAction SilentlyContinue
        $env:GOCACHE = $oldGoCache
    }

    Copy-Item -LiteralPath (Join-Path $project 'deploy/Dockerfile.runtime') -Destination (Join-Path $stage 'Dockerfile')
    Remove-Utf8Bom (Join-Path $stage "Dockerfile")

    @'
services:
  bridge:
    build:
      context: .
      dockerfile: Dockerfile
    env_file: .env
    ports:
      - "8090:8090"
      - "8443:8090"
    depends_on:
      mysql:
        condition: service_healthy
      redis:
        condition: service_healthy
    restart: unless-stopped
  mysql:
    image: mysql:8.0
    environment:
      MYSQL_DATABASE: bridge
      MYSQL_USER: bridge
      MYSQL_PASSWORD: ${BRIDGE_DB_PASSWORD}
      MYSQL_ROOT_PASSWORD: ${BRIDGE_DB_ROOT_PASSWORD}
    volumes:
      - bridge_mysql:/var/lib/mysql
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost"]
      interval: 5s
      timeout: 3s
      retries: 30
    restart: unless-stopped
  redis:
    image: redis:7-alpine
    volumes:
      - bridge_redis:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 3s
      retries: 30
    restart: unless-stopped
volumes:
  bridge_mysql:
  bridge_redis:
'@ | Set-Content -Encoding UTF8 (Join-Path $stage "docker-compose.yml")
    Remove-Utf8Bom (Join-Path $stage "docker-compose.yml")

    @'
BRIDGE_LISTEN_ADDR=:8090
BRIDGE_SHARED_SECRET=__BRIDGE_SHARED_SECRET__
BRIDGE_STORE=mysql
BRIDGE_AUTO_MIGRATE=true
BRIDGE_DB_PASSWORD=__BRIDGE_DB_PASSWORD__
BRIDGE_DB_ROOT_PASSWORD=__BRIDGE_DB_ROOT_PASSWORD__
BRIDGE_MYSQL_DSN=bridge:__BRIDGE_DB_PASSWORD__@tcp(mysql:3306)/bridge?parseTime=true&charset=utf8mb4
BRIDGE_REDIS_URL=redis://redis:6379/0
BRIDGE_MAX_ATTEMPTS=5
BRIDGE_WORKER_INTERVAL_MS=250
BRIDGE_LEADER_ENABLED=false
BRIDGE_PEER_ID=
BRIDGE_ENCRYPTION_KEY=__BRIDGE_ENCRYPTION_KEY__

TENCENT_SDK_APP_ID=20044723
TENCENT_SECRET_KEY=__SET_TENCENT_SECRET_KEY__
TENCENT_ADMIN_ACCOUNT=administrator
TENCENT_REST_BASE_URL=https://adminapisgp.im.qcloud.com
TENCENT_USERSIG_TTL=86400

TANGSENG_BASE_URL=http://10.0.0.100:8090
TANGSENG_LOGIN_ENDPOINT=/v1/user/usernamelogin
TANGSENG_USER_ENDPOINT=/v1/user/usernameregister
TANGSENG_WS_URL=ws://10.0.0.100:5200/
TANGSENG_USERNAME=
TANGSENG_PASSWORD=
TANGSENG_DEVICE_ID=
TANGSENG_SHADOW_PASSWORD=__SET_TANGSENG_SHADOW_PASSWORD__
TANGSENG_LEGACY_SHADOW_PASSWORD=__SET_TANGSENG_LEGACY_SHADOW_PASSWORD__
TANGSENG_INVITE_CODE=__SET_TANGSENG_INVITE_CODE__
TANGSENG_MANUFACTURER=bridge
'@ | Set-Content -Encoding UTF8 (Join-Path $stage ".env.template")
    Remove-Utf8Bom (Join-Path $stage ".env.template")

    @'
This bundle is for the independent hw-bridge CVM in the TangSeng VPC.
It does not modify the TangSeng server or the existing Tencent PHP server.
The 8443 mapping is plain HTTP for the first internal test. Put it behind
TLS before production PHP traffic is enabled.
'@ | Set-Content -Encoding UTF8 (Join-Path $stage "README.txt")
    Remove-Utf8Bom (Join-Path $stage "README.txt")

    if (Test-Path $bundle) { Remove-Item -Force $bundle }
    Write-Host "Packaging deployment bundle..."
    Copy-Item -LiteralPath (Join-Path $project 'migrations') -Destination $stage -Recurse
    $sourcePrefix = (Get-Item -LiteralPath $project).FullName.TrimEnd('\') + '\'
    $sourceLines = Get-ChildItem -LiteralPath $project -File -Recurse | Where-Object { $_.Extension -in @('.go','.sql') -or $_.Name -in @('go.mod','go.sum') } | Sort-Object FullName | ForEach-Object {
        (Get-FileHash -LiteralPath $_.FullName -Algorithm SHA256).Hash.ToLowerInvariant() + '  ' + $_.FullName.Substring($sourcePrefix.Length).Replace('\','/')
    }
    [IO.File]::WriteAllText((Join-Path $stage 'SOURCE_SHA256SUMS'), (($sourceLines -join "`n") + "`n"), (New-Object Text.UTF8Encoding($false)))
    $stagePrefix = (Get-Item -LiteralPath $stage).FullName.TrimEnd('\') + '\'
    $hashFiles = @('bridge','Dockerfile','docker-compose.yml','.env.template','README.txt','SOURCE_SHA256SUMS') | ForEach-Object { Get-Item -LiteralPath (Join-Path $stage $_) }
    $hashFiles += Get-ChildItem -LiteralPath (Join-Path $stage 'migrations') -File -Recurse
    $hashLines = $hashFiles | ForEach-Object { (Get-FileHash -LiteralPath $_.FullName -Algorithm SHA256).Hash.ToLowerInvariant() + '  ' + $_.FullName.Substring($stagePrefix.Length).Replace('\','/') }
    [IO.File]::WriteAllText((Join-Path $stage 'SHA256SUMS'), (($hashLines -join "`n") + "`n"), (New-Object Text.UTF8Encoding($false)))
    tar -czf $bundle -C $stage bridge Dockerfile docker-compose.yml .env.template README.txt migrations SOURCE_SHA256SUMS SHA256SUMS
    if ($LASTEXITCODE -ne 0) { throw "tar packaging failed" }
} finally {
    $resolvedStage = [IO.Path]::GetFullPath($stage)
    $tempRoot = [IO.Path]::GetFullPath($env:TEMP).TrimEnd('\') + '\'
    if (!$resolvedStage.StartsWith($tempRoot, [StringComparison]::OrdinalIgnoreCase)) { throw 'Unsafe staging path' }
    if (Test-Path -LiteralPath $resolvedStage) { Remove-Item -LiteralPath $resolvedStage -Recurse -Force }
}
} elseif (!(Test-Path -LiteralPath $bundle)) { throw 'Prepared Bridge bundle not found' }

if ($PackageOnly) {
    Write-Host "Bundle created: $bundle"
    exit 0
}

$remoteScript = @'
set -euo pipefail
INSTALL=/opt/hw-bridge
ARCHIVE=/tmp/hw-bridge-private-vpc.tar.gz
STAMP=$(date +%Y%m%d%H%M%S)
BACKUP=/opt/hw-bridge.backup.$STAMP
STAGE=$(mktemp -d /opt/hw-bridge-stage.XXXXXX)
trap 'rm -rf "$STAGE"' EXIT
[ -f "$INSTALL/.env" ] && [ -f "$INSTALL/docker-compose.yml" ] || { echo 'Existing Bridge deployment required'; exit 20; }
if grep -Eq '^[A-Z0-9_]+=(__SET_|__BRIDGE_)' "$INSTALL/.env"; then echo 'Unresolved environment placeholders'; exit 21; fi
curl -fsS --max-time 8 http://10.0.0.100:8090/v1/ping >/dev/null
mkdir -m 700 "$BACKUP"
cp -a "$INSTALL/bridge" "$INSTALL/Dockerfile" "$INSTALL/docker-compose.yml" "$INSTALL/.env" "$BACKUP/"
tar -xzf "$ARCHIVE" -C "$STAGE"
(cd "$STAGE" && sha256sum -c SHA256SUMS)
cd "$INSTALL"
# Schema additions preserve compatibility with the backed-up binary.
for migration in "$STAGE"/migrations/*.sql; do
 docker compose exec -T mysql sh -c 'export MYSQL_PWD="$MYSQL_PASSWORD"; exec mysql -u"$MYSQL_USER" "$MYSQL_DATABASE"' < "$migration"
done
cp "$STAGE/bridge" "$INSTALL/bridge.next"
chmod 755 "$INSTALL/bridge.next"
mv "$INSTALL/bridge.next" "$INSTALL/bridge"
cp "$STAGE/Dockerfile" "$INSTALL/Dockerfile"
rollback() {
 cp "$BACKUP/bridge" "$INSTALL/bridge"
 cp "$BACKUP/Dockerfile" "$INSTALL/Dockerfile"
 docker compose up -d --build --no-deps bridge
 echo "BRIDGE_ROLLBACK_APPLIED backup=$BACKUP"
}
if ! docker compose up -d --build --no-deps bridge; then rollback; exit 30; fi
healthy=false
for i in $(seq 1 30); do
 if curl -fsS --max-time 3 http://127.0.0.1:8090/readyz >/dev/null; then healthy=true; break; fi
 sleep 2
done
if [ "$healthy" != true ]; then rollback; exit 31; fi
cmp -s "$INSTALL/.env" "$BACKUP/.env" || { echo 'ENV_CHANGED'; exit 32; }
sha256sum "$INSTALL/bridge"
docker compose ps bridge
printf 'HW_BRIDGE_DEPLOY_OK backup=%s\n' "$BACKUP"
'@

$encoded = [Convert]::ToBase64String([Text.Encoding]::UTF8.GetBytes(($remoteScript -replace "`r`n", "`n")))
Write-Host "Uploading bundle to $ServerUser@$ServerHost..."
scp @sshOptions $bundle "$ServerUser@$ServerHost`:$('/tmp/hw-bridge-private-vpc.tar.gz')"
if ($LASTEXITCODE -ne 0) { throw "scp failed" }
Write-Host "Installing on the new CVM..."
ssh @sshOptions -T "$ServerUser@$ServerHost" "echo $encoded | base64 -d | sudo -n bash"
if ($LASTEXITCODE -ne 0) { throw "remote deploy failed" }
Write-Host "Deployment completed. The existing Tencent and TangSeng services were not changed."

