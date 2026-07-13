<#
.SYNOPSIS
    One-click: build GTSDB, start servers, run benchmarks, generate report.
.DESCRIPTION
    Default: builds GTSDB, restarts all servers, runs full benchmark suite,
    generates markdown report with charts, then STOPS all servers.

    -SkipBuild   : skip GTSDB compilation
    -SkipRestart : skip server restart (assumes running)
    -KeepAlive   : keep servers running after benchmark completes
    -StopOnly    : just stop all servers
    -DB          : comma-separated DB list (default: gtsdb,influx,vm,nsq)
    -Count       : points per write benchmark (default: 3000)
    -Runs        : number of runs (default: 2)

.EXAMPLE
    .\run_benchmark.ps1
    .\run_benchmark.ps1 -SkipBuild -SkipRestart
    .\run_benchmark.ps1 -KeepAlive               # leave servers running
    .\run_benchmark.ps1 -DB "gtsdb,vm" -Count 10000 -Runs 3
#>

param(
    [switch]$SkipBuild,
    [switch]$SkipRestart,
    [switch]$KeepAlive,
    [switch]$StopOnly,
    [int]$Count = 3000,
    [int]$Sensors = 5,
    [int]$Runs = 2,
    [int]$Warmup = 200,
    [string]$DB = "gtsdb,influx,vm,nsq"
)

$BenchDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$RepoRoot = Resolve-Path "$BenchDir\.."
$Desktop = "$env:USERPROFILE\Desktop"
$Timestamp = Get-Date -Format "yyyyMMdd_HHmmss"
$JsonFile = "$env:TEMP\bench_results_$Timestamp.json"
$StdFile = "$env:TEMP\bench_results.json"
$ReportDir = "$BenchDir\report"

if ($StopOnly) {
    & "$BenchDir\stop_servers.ps1"
    exit 0
}

Write-Host "============================================" -ForegroundColor Cyan
Write-Host "  GTSDB Benchmark Suite" -ForegroundColor Cyan
Write-Host "============================================" -ForegroundColor Cyan
Write-Host "  DB:      $DB"
Write-Host "  Count:   $Count  | Sensors: $Sensors"
Write-Host "  Runs:    $Runs    | Warmup:  $Warmup"
Write-Host "============================================" -ForegroundColor Cyan

# 1. Build GTSDB
if (-not $SkipBuild) {
    Write-Host ">>> Step 1: Building GTSDB..." -ForegroundColor Yellow
    Push-Location $RepoRoot
    go build -ldflags="-s -w" -trimpath -o "$Desktop\gtsdb.exe" .
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Build failed!"
        Pop-Location; exit 1
    }
    Pop-Location
    Write-Host "  Build OK" -ForegroundColor Green
} else {
    Write-Host ">>> Step 1: Skipping build (-SkipBuild)" -ForegroundColor DarkGray
}

# 2. Start servers
if (-not $SkipRestart) {
    Write-Host ">>> Step 2: Starting servers..." -ForegroundColor Yellow
    & "$BenchDir\stop_servers.ps1"
    Start-Sleep 2
    & "$BenchDir\start_servers.ps1"
    Start-Sleep 2
} else {
    Write-Host ">>> Step 2: Servers already running (-SkipRestart)" -ForegroundColor DarkGray
}

# 3. Run benchmark
Write-Host ">>> Step 3: Running benchmarks..." -ForegroundColor Yellow
$env:INFLUX_TOKEN = "bench-token-123"

Push-Location $BenchDir
$raw = & go run . "--db=$DB" "--count=$Count" "--sensors=$Sensors" "--runs=$Runs" "--warmup=$Warmup" "--format=json" "all" 2>&1
$exitCode = $LASTEXITCODE
Pop-Location

if ($exitCode -ne 0) {
    Write-Host "  Exit code: $exitCode (partial results may be available)" -ForegroundColor DarkYellow
}

# 4. Extract JSON
Write-Host ">>> Step 4: Extracting JSON..." -ForegroundColor Yellow
$inJson = $false; $jsonLines = @()
foreach ($line in $raw) {
    $t = "$line".Trim()
    if ($t -eq '[' -and -not $inJson) { $inJson = $true }
    if ($inJson) { $jsonLines += "$line"; if ($t -eq ']') { break } }
}

if ($jsonLines.Count -gt 0) {
    $jsonStr = $jsonLines -join "`n"
    try {
        $null = $jsonStr | ConvertFrom-Json
        $utf8 = [System.Text.UTF8Encoding]::new($false)
        [System.IO.File]::WriteAllText($JsonFile, $jsonStr, $utf8)
        [System.IO.File]::WriteAllText($StdFile, $jsonStr, $utf8)
        Write-Host "  Saved to $JsonFile" -ForegroundColor Green
    } catch {
        Write-Warning "  JSON parse failed: $_"
        $raw | Out-File "$JsonFile.raw.txt" -Encoding utf8
    }
} else {
    Write-Warning "  No JSON found - saving raw output"
    $raw | Out-File "$JsonFile.raw.txt" -Encoding utf8
}

# 5. Generate report
Write-Host ">>> Step 5: Generating report..." -ForegroundColor Yellow
$venvPython = "$RepoRoot\.venv\Scripts\python.exe"
if (Test-Path $venvPython) {
    & $venvPython "$BenchDir\generate_report.py"
} else {
    Write-Warning "  Python venv not found - skipping report"
}

# Done
Write-Host "============================================" -ForegroundColor Cyan
Write-Host "  Report: $ReportDir\BENCHMARK_REPORT.md" -ForegroundColor Green
Write-Host "  Charts: $ReportDir\charts\" -ForegroundColor Green
Write-Host "============================================" -ForegroundColor Cyan

# 5.5. Update README.md with latest report
$readmeFile = "$BenchDir\README.md"
$reportFile = "$ReportDir\BENCHMARK_REPORT.md"
if (Test-Path $reportFile) {
    $header = @"
# GTSDB Benchmark

Live benchmark results comparing [GTSDB](https://github.com/abbychau/gtsdb), InfluxDB, VictoriaMetrics, and NSQ.

> Auto-generated by ``run_benchmark.ps1``

---

"@
    $report = Get-Content $reportFile -Raw -Encoding UTF8
    $report = $report -replace '\(charts/', '(report/charts/'
    [System.IO.File]::WriteAllText($readmeFile, $header + $report, [System.Text.UTF8Encoding]::new($false))
    Write-Host "  README: $readmeFile" -ForegroundColor Green
}

# 6. Stop servers (unless -KeepAlive)
if (-not $KeepAlive -and -not $SkipRestart) {
    Write-Host ">>> Step 6: Stopping servers... (use -KeepAlive to skip)" -ForegroundColor Yellow
    & "$BenchDir\stop_servers.ps1"
} elseif ($KeepAlive) {
    Write-Host "  Servers left running (-KeepAlive)" -ForegroundColor DarkYellow
    Write-Host "  Stop:   .\stop_servers.ps1" -ForegroundColor Yellow
}
