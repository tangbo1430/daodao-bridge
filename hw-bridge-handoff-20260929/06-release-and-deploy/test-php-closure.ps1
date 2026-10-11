param([string]$SshKeyPath='C:/Users/Administrator/.ssh/hw_bridge_20260929')
$ErrorActionPreference='Stop'
& "$PSScriptRoot/deploy-php-bridge-provision.redacted.ps1" -PackageOnly
scp -i $SshKeyPath "$PSScriptRoot/php-bridge-provision.tar.gz" root@23.251.34.150:/tmp/php-closure-check.tar.gz
scp -i $SshKeyPath "$PSScriptRoot/../03-php-deployment/tests/*.php" root@23.251.34.150:/tmp/
$script=@'
set -euo pipefail
mkdir -p /tmp/php-closure-check
tar -xzf /tmp/php-closure-check.tar.gz -C /tmp/php-closure-check
cd /root/IM/YCH-IM-deploy
docker compose cp /tmp/php-closure-check php:/tmp/
while read -r hash relative; do docker compose exec -T php php -l < "/tmp/php-closure-check/$relative"; done < /tmp/php-closure-check/SHA256SUMS
for test in registration_transaction registration_product bridge_facade group_creation_route account_deletion bridge_media_stream media_source message_state_callbacks; do
 docker compose cp "/tmp/${test}_test.php" "php:/tmp/${test}_test.php"
 docker compose exec -T -e GROUP_MODEL_FILE=/tmp/php-closure-check/application/api/model/GroupV.php -e REGISTRATION_USER_FILE=/tmp/php-closure-check/application/api/model/User.php -e REGISTRATION_SERVICE_FILE=/tmp/php-closure-check/application/imaccess/service/RegistrationService.php -e BRIDGE_SERVICE_FILE=/tmp/php-closure-check/application/imaccess/service/BridgeService.php -e MEDIA_SOURCE_FILE=/tmp/php-closure-check/application/imaccess/service/MediaSourceService.php -e BRIDGE_MEDIA_FILE=/tmp/php-closure-check/application/imaccess/controller/BridgeMedia.php php php "/tmp/${test}_test.php" < /dev/null
done
echo PHP_CLOSURE_CHECK_OK
'@
$encoded=[Convert]::ToBase64String([Text.Encoding]::UTF8.GetBytes(($script -replace "`r`n","`n")))
ssh -T -i $SshKeyPath root@23.251.34.150 "echo $encoded | base64 -d | bash"
if($LASTEXITCODE -ne 0){throw 'PHP closure verification failed'}
