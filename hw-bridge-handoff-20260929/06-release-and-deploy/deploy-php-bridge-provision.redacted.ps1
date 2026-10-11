param(
    [string]$ServerHost = "23.251.34.150",
    [string]$ServerUser = "root",
    [string]$SourceRoot = (Join-Path $PSScriptRoot '..\03-php-deployment\backend'),
    [switch]$PackageOnly,
    [string]$SshKeyPath = $(if ($env:HW_BRIDGE_SSH_KEY) { $env:HW_BRIDGE_SSH_KEY } else { "" })
)

$ErrorActionPreference = "Stop"
$sshOptions = @()
if ($SshKeyPath -and (Test-Path -LiteralPath $SshKeyPath)) { $sshOptions = @("-i", $SshKeyPath) }
$sourceRoot = (Resolve-Path -LiteralPath $SourceRoot).Path
$bundle = Join-Path $PSScriptRoot "php-bridge-provision.tar.gz"
$stage = Join-Path $env:TEMP ("php-bridge-patch-" + [guid]::NewGuid().ToString("N"))

if (-not (Test-Path (Join-Path $sourceRoot "application\api\model\User.php"))) {
    throw "PHP source not found: $sourceRoot"
}
if (-not (Get-Command tar -ErrorAction SilentlyContinue)) {
    throw "Windows tar.exe is required."
}

New-Item -ItemType Directory -Force (Join-Path $stage "application\api\model") | Out-Null
New-Item -ItemType Directory -Force (Join-Path $stage "application\imaccess\service") | Out-Null
New-Item -ItemType Directory -Force (Join-Path $stage "application\imaccess\controller") | Out-Null
New-Item -ItemType Directory -Force (Join-Path $stage "application\imcallback\handler") | Out-Null
New-Item -ItemType Directory -Force (Join-Path $stage "route") | Out-Null
New-Item -ItemType Directory -Force (Join-Path $stage "config") | Out-Null
Copy-Item (Join-Path $sourceRoot "application\api\model\User.php") (Join-Path $stage "application\api\model\User.php")
Copy-Item (Join-Path $sourceRoot "application\api\model\GroupV.php") (Join-Path $stage "application\api\model\GroupV.php")
Copy-Item (Join-Path $sourceRoot "application\imcallback\handler\HandlerRegistry.php") (Join-Path $stage "application\imcallback\handler\HandlerRegistry.php")
Copy-Item (Join-Path $sourceRoot "application\imaccess\service\BridgeService.php") (Join-Path $stage "application\imaccess\service\BridgeService.php")
Copy-Item (Join-Path $sourceRoot "application\imaccess\service\MediaSourceService.php") (Join-Path $stage "application\imaccess\service\MediaSourceService.php")
Copy-Item (Join-Path $sourceRoot "application\imaccess\controller\Bridge.php") (Join-Path $stage "application\imaccess\controller\Bridge.php")
Copy-Item (Join-Path $sourceRoot "application\imaccess\controller\BridgeMedia.php") (Join-Path $stage "application\imaccess\controller\BridgeMedia.php")
Copy-Item (Join-Path $sourceRoot "route\route.php") (Join-Path $stage "route\route.php")
Copy-Item (Join-Path $sourceRoot "config\im_bridge.php") (Join-Path $stage "config\im_bridge.php")
Copy-Item (Join-Path $sourceRoot "application\imaccess\service\BlocklistService.php") (Join-Path $stage "application\imaccess\service\BlocklistService.php")
Copy-Item (Join-Path $sourceRoot "application\api\model\UserTencent.php") (Join-Path $stage "application\api\model\UserTencent.php")
foreach ($relative in @('application/api/controller/Login.php','application/imaccess/service/RegistrationService.php','application/imaccess/controller/BridgeInternal.php','application/imcallback/service/WorkerService.php')) {
    $destination = Join-Path $stage $relative
    New-Item -ItemType Directory -Force (Split-Path $destination -Parent) | Out-Null
    Copy-Item -LiteralPath (Join-Path $sourceRoot $relative) -Destination $destination
}

$checksumLines = Get-ChildItem -LiteralPath $stage -File -Recurse | ForEach-Object {
    $canonicalStage = (Get-Item -LiteralPath $stage).FullName.TrimEnd('\')
    $relative = $_.FullName.Substring($canonicalStage.Length + 1).Replace('\', '/')
    (Get-FileHash -LiteralPath $_.FullName -Algorithm SHA256).Hash.ToLowerInvariant() + '  ' + $relative
}
[IO.File]::WriteAllText((Join-Path $stage 'SHA256SUMS'), (($checksumLines -join "`n") + "`n"), (New-Object Text.UTF8Encoding($false)))

if (Test-Path $bundle) { Remove-Item -Force $bundle }
tar -czf $bundle -C $stage application config route SHA256SUMS
if ($LASTEXITCODE -ne 0) { throw "PHP patch packaging failed" }
$resolvedStage = [IO.Path]::GetFullPath($stage)
$tempRoot = [IO.Path]::GetFullPath($env:TEMP).TrimEnd('\') + '\'
if (!$resolvedStage.StartsWith($tempRoot, [StringComparison]::OrdinalIgnoreCase)) { throw 'Unsafe staging path' }
Remove-Item -LiteralPath $resolvedStage -Recurse -Force

if ($PackageOnly) {
    Write-Host "PHP patch bundle created: $bundle"
    exit 0
}

$remoteScript = @'
set -euo pipefail
DEPLOY=/root/IM/YCH-IM-deploy
PATCH=/tmp/php-bridge-provision.tar.gz
STAMP=$(date +%Y%m%d%H%M%S)
BACKUP=/root/IM/backup-php-bridge-$STAMP
STAGE=$(mktemp -d /tmp/php-bridge-stage.XXXXXX)
trap 'rm -rf "$STAGE"' EXIT
[ -d "$DEPLOY/backend" ] && [ -f "$DEPLOY/.env" ] || exit 20
if ! grep -q '^IM_BRIDGE_SHARED_SECRET=.' "$DEPLOY/.env" || ! grep -q '^IM_BRIDGE_URL=.' "$DEPLOY/.env"; then echo 'Missing existing Bridge configuration'; exit 21; fi
if grep -Eq '^(IM_BRIDGE_SHARED_SECRET|IM_BRIDGE_URL)=(__SET_|__BRIDGE_)' "$DEPLOY/.env"; then echo 'Unresolved environment placeholders'; exit 22; fi
tar -xzf "$PATCH" -C "$STAGE"
(cd "$STAGE" && sha256sum -c SHA256SUMS)
cd "$DEPLOY"
while read -r hash relative; do
 case "$relative" in application/*.php|config/*.php|route/*.php) ;; *) echo 'Unexpected patch path'; exit 23;; esac
 docker compose exec -T php php -l < "$STAGE/$relative"
done < "$STAGE/SHA256SUMS"
mkdir -m 700 "$BACKUP"
cp "$DEPLOY/.env" "$BACKUP/env"
while read -r hash relative; do
 mkdir -p "$BACKUP/$(dirname "$relative")" "$DEPLOY/backend/$(dirname "$relative")"
 if [ -f "$DEPLOY/backend/$relative" ]; then cp -a "$DEPLOY/backend/$relative" "$BACKUP/$relative"; fi
 cp "$STAGE/$relative" "$DEPLOY/backend/$relative"
done < "$STAGE/SHA256SUMS"
cp "$STAGE/SHA256SUMS" "$BACKUP/deployed-SHA256SUMS"
docker compose exec -T php php -r 'require "/var/www/app/thinkphp/base.php"; \think\Container::get("app")->path("/var/www/app/application/")->initialize(); \app\imaccess\service\RegistrationService::install(); echo "REGISTRATION_SCHEMA_OK\n";' < /dev/null
docker compose restart php worker < /dev/null
cmp -s "$DEPLOY/.env" "$BACKUP/env" || { echo 'ENV_CHANGED'; exit 24; }
(cd "$DEPLOY/backend" && sha256sum -c "$STAGE/SHA256SUMS")
docker compose ps php worker
printf 'PHP_BRIDGE_PROVISION_DEPLOY_OK backup=%s\n' "$BACKUP"
'@
$encoded = [Convert]::ToBase64String([Text.Encoding]::UTF8.GetBytes(($remoteScript -replace "`r`n", "`n")))
$runner = if ($ServerUser -eq "root") { "echo $encoded | base64 -d | bash" } else { "sudo -v && echo $encoded | base64 -d | sudo bash" }

Write-Host "Uploading PHP bridge patch to $ServerUser@$ServerHost..."
scp @sshOptions $bundle "$ServerUser@$ServerHost`:/tmp/php-bridge-provision.tar.gz"
if ($LASTEXITCODE -ne 0) { throw "scp failed" }
Write-Host "Applying PHP registration integration..."
ssh @sshOptions -T "$ServerUser@$ServerHost" $runner
if ($LASTEXITCODE -ne 0) { throw "remote PHP deploy failed" }
Write-Host "PHP bridge registration integration deployed."

