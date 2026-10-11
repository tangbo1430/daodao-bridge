param([string]$SshKeyPath='C:/Users/Administrator/.ssh/hw_bridge_20260929')
$ErrorActionPreference='Stop'
scp -i $SshKeyPath "$PSScriptRoot/verify-php-registration.php" root@23.251.34.150:/tmp/verify-php-registration.php
if ($LASTEXITCODE -ne 0) {throw 'Fixture upload failed'}
ssh -T -i $SshKeyPath root@23.251.34.150 'cd /root/IM/YCH-IM-deploy && docker compose cp /tmp/verify-php-registration.php php:/tmp/verify-php-registration.php && docker compose exec -T php php /tmp/verify-php-registration.php < /dev/null'
if($LASTEXITCODE -ne 0){throw 'Native registration transaction validation failed'}
