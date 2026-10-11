param(
    [Parameter(Mandatory = $true)]
    [string]$HostName,
    [int[]]$Ports = @(8090, 15200)
)

$ErrorActionPreference = 'Continue'

Write-Host "A+ Phase 0 read-only probe for $HostName"
foreach ($port in $Ports) {
    try {
        $response = Invoke-WebRequest -Uri "http://$HostName`:$port/" -Method Head -TimeoutSec 5 -SkipHttpErrorCheck
        Write-Host "HTTP $port => $($response.StatusCode) $($response.Headers['Server'])"
    } catch {
        Write-Host "HTTP $port => $($_.Exception.Message)"
    }
}

Write-Host "No registration, login, message send, or WebSocket authentication is performed by this probe."

