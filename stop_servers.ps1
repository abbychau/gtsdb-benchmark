<#
.SYNOPSIS
    Stops all benchmark database server instances.
#>

Write-Host "=== Stopping all server instances ===" -ForegroundColor Cyan
@("gtsdb", "victoria-metrics-windows-amd64-prod", "influxd", "nsqd", "nsqlookupd") | ForEach-Object {
    $p = Get-Process -Name $_ -ErrorAction SilentlyContinue
    if ($p) {
        $p | Stop-Process -Force
        Write-Host "  Killed $_ (PID $($p.Id))" -ForegroundColor DarkYellow
    }
}
Write-Host "=== All servers stopped ===" -ForegroundColor Cyan
