<#
.SYNOPSIS
    Starts all database servers for benchmarking: GTSDB, VictoriaMetrics, InfluxDB, NSQ.
.DESCRIPTION
    Kills any existing instances, cleans old data, starts fresh servers, and
    initializes InfluxDB with the required org/bucket/token.
    Run this before executing benchmark suite.
.NOTES
    Assumes server binaries exist on the user's Desktop.
#>

$ErrorActionPreference = "Stop"
$Desktop = "$env:USERPROFILE\Desktop"
$Temp = $env:TEMP
$RepoRoot = "C:\Repos\gtsdb"

# ── Helper: wait for a TCP port to be listening ──
function Wait-Port {
    param($Port, $TimeoutSeconds = 10)
    $elapsed = 0
    while ($elapsed -lt $TimeoutSeconds) {
        $conn = Get-NetTCPConnection -LocalPort $Port -ErrorAction SilentlyContinue
        if ($conn -and $conn.State -eq "Listen") { return $true }
        Start-Sleep -Seconds 1
        $elapsed++
    }
    return $false
}

# ── Kill existing instances ──
Write-Host "=== Killing existing server instances ===" -ForegroundColor Cyan
@("gtsdb", "victoria-metrics-windows-amd64-prod", "influxd", "nsqd", "nsqlookupd") | ForEach-Object {
    Get-Process -Name $_ -ErrorAction SilentlyContinue | Stop-Process -Force
}
Start-Sleep -Seconds 2

# ── Clean data directories ──
Write-Host "=== Cleaning data directories ===" -ForegroundColor Cyan
Remove-Item -Recurse -Force "$RepoRoot\data"             -ErrorAction SilentlyContinue
Remove-Item -Recurse -Force "$Temp\vm-data"               -ErrorAction SilentlyContinue
Remove-Item -Recurse -Force "$Temp\influxdb-data"         -ErrorAction SilentlyContinue
Remove-Item -Force "$RepoRoot\benchmark-repo\nsqd.dat"    -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force "$Temp\vm-data"               | Out-Null
New-Item -ItemType Directory -Force "$Temp\influxdb-data\engine"  | Out-Null

# ═══════════════════════════════════════════════════════════════════════
# 1. GTSDB (with compression)
# ═══════════════════════════════════════════════════════════════════════
Write-Host "=== Starting GTSDB (compression enabled) ===" -ForegroundColor Green
$pg = Start-Process -FilePath "$Desktop\gtsdb.exe" `
    -WorkingDirectory $RepoRoot `
    -ArgumentList "bench.ini" `
    -PassThru -NoNewWindow `
    -RedirectStandardOutput "$Temp\gtsdb_stdout.log" `
    -RedirectStandardError "$Temp\gtsdb_stderr.log"
if (-not (Wait-Port 5555 5)) { Write-Warning "GTSDB port 5555 not ready" }

# ═══════════════════════════════════════════════════════════════════════
# 2. VictoriaMetrics
# ═══════════════════════════════════════════════════════════════════════
Write-Host "=== Starting VictoriaMetrics ===" -ForegroundColor Green
$pv = Start-Process -FilePath "$Desktop\victoria-metrics-windows-amd64-prod.exe" `
    -ArgumentList "-storageDataPath","$Temp\vm-data","-retentionPeriod","1200","-httpListenAddr",":8428" `
    -PassThru -NoNewWindow `
    -RedirectStandardOutput "$Temp\vm_stdout.log" `
    -RedirectStandardError "$Temp\vm_stderr.log"
if (-not (Wait-Port 8428 10)) { Write-Warning "VM port 8428 not ready" }

# ═══════════════════════════════════════════════════════════════════════
# 3. InfluxDB
# ═══════════════════════════════════════════════════════════════════════
Write-Host "=== Starting InfluxDB ===" -ForegroundColor Green
$pi = Start-Process -FilePath "$Desktop\influxdb2-2.9.1-windows_amd64\influxd.exe" `
    -ArgumentList "--store=memory","--engine-path=$Temp\influxdb-data\engine","--bolt-path=$Temp\influxdb-data\influxd.bolt","--reporting-disabled" `
    -PassThru -NoNewWindow `
    -RedirectStandardOutput "$Temp\influx_stdout.log" `
    -RedirectStandardError "$Temp\influx_stderr.log"
if (-not (Wait-Port 8086 10)) { Write-Warning "InfluxDB port 8086 not ready" }

# Setup InfluxDB (first-run initialization)
Write-Host "=== Initializing InfluxDB (org/bucket/token) ===" -ForegroundColor Yellow
$body = @{
    username = "admin"
    password = "password123"
    org      = "bench"
    bucket   = "bench"
    token    = "bench-token-123"
} | ConvertTo-Json
try {
    Invoke-RestMethod -Uri "http://localhost:8086/api/v2/setup" `
        -Method POST -Body $body -ContentType "application/json" -ErrorAction Stop | Out-Null
    Write-Host "InfluxDB initialized successfully" -ForegroundColor Green
} catch {
    Write-Host "InfluxDB setup: $($_.Exception.Message)" -ForegroundColor DarkYellow
}

# ═══════════════════════════════════════════════════════════════════════
# 4. NSQ (nsqlookupd + nsqd)
# ═══════════════════════════════════════════════════════════════════════
Write-Host "=== Starting NSQ ===" -ForegroundColor Green
$pnl = Start-Process -FilePath "$Desktop\nsq-1.3.0.windows-amd64.go1.21.5\bin\nsqlookupd.exe" `
    -PassThru -NoNewWindow `
    -RedirectStandardOutput "$Temp\nsqlookupd.log" `
    -RedirectStandardError "$Temp\nsqlookupd_err.log"
if (-not (Wait-Port 4160 5)) { Write-Warning "NSQ lookupd port 4160 not ready" }

$pnq = Start-Process -FilePath "$Desktop\nsq-1.3.0.windows-amd64.go1.21.5\bin\nsqd.exe" `
    -ArgumentList "--lookupd-tcp-address=127.0.0.1:4160" `
    -PassThru -NoNewWindow `
    -RedirectStandardOutput "$Temp\nsqd.log" `
    -RedirectStandardError "$Temp\nsqd_err.log"
if (-not (Wait-Port 4150 5)) { Write-Warning "NSQ port 4150 not ready" }

# ═══════════════════════════════════════════════════════════════════════
# Summary
# ═══════════════════════════════════════════════════════════════════════
Write-Host "`n=== All servers started ===" -ForegroundColor Cyan
Write-Host "  GTSDB     : PID $($pg.Id)   :5555 (TCP)  :5556 (HTTP)" -ForegroundColor Green
Write-Host "  VM        : PID $($pv.Id)   :8428" -ForegroundColor Green
Write-Host "  InfluxDB  : PID $($pi.Id)   :8086" -ForegroundColor Green
Write-Host "  NSQ       : lookupd PID $($pnl.Id) :4160, nsqd PID $($pnq.Id) :4150" -ForegroundColor Green
Write-Host "`nNow run benchmarks:"
Write-Host "  cd C:\Repos\gtsdb\benchmark-repo"
Write-Host '  $env:INFLUX_TOKEN="bench-token-123"'
Write-Host '  go run . "--db=gtsdb,influx,vm,nsq" "--count=3000" "--sensors=5" "--runs=2" "--warmup=200" "--format=json" "all"'
