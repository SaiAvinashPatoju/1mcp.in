<#
.SYNOPSIS
    Verifies that 1mcp.in has been completely removed
#>

Write-Host "Verifying 1mcp.in removal..." -ForegroundColor Cyan
Write-Host ""

$clean = $true

# Check binaries
Write-Host "[1/5] Checking binaries..." -ForegroundColor Yellow
$binPaths = @(
    "C:\projects\work\1Mcp\bin\mach1.exe",
    "C:\projects\work\1Mcp\bin\mach1ctl.exe",
    "$env:LOCALAPPDATA\1mcp\bin\mach1.exe",
    "$env:LOCALAPPDATA\1mcp\bin\mach1ctl.exe"
)
foreach ($path in $binPaths) {
    if (Test-Path $path) {
        Write-Host "  ✗ Found: $path" -ForegroundColor Red
        $clean = $false
    }
}
if ($clean) {
    Write-Host "  ✓ No binaries found" -ForegroundColor Green
}

# Check data directories
Write-Host ""
Write-Host "[2/5] Checking data directories..." -ForegroundColor Yellow
$dataPaths = @(
    "$env:APPDATA\Mach1",
    "$env:LOCALAPPDATA\1mcp",
    "$env:USERPROFILE\.mach1",
    "$env:USERPROFILE\.config\mach1"
)
foreach ($path in $dataPaths) {
    if (Test-Path $path) {
        Write-Host "  ✗ Found: $path" -ForegroundColor Red
        $clean = $false
    }
}
if ($clean) {
    Write-Host "  ✓ No data directories found" -ForegroundColor Green
}

# Check environment variables
Write-Host ""
Write-Host "[3/5] Checking environment variables..." -ForegroundColor Yellow
$envVars = @("MACH1_HTTP_TOKEN", "MACH1_HOME", "MACH1_DATA_DIR", "MACH1_CATALOG", "MACH1_DB_PATH")
foreach ($var in $envVars) {
    $userVal = [Environment]::GetEnvironmentVariable($var, "User")
    $machineVal = [Environment]::GetEnvironmentVariable($var, "Machine")
    if ($userVal -or $machineVal) {
        Write-Host "  ✗ Found: $var" -ForegroundColor Red
        $clean = $false
    }
}
if ($clean) {
    Write-Host "  ✓ No environment variables found" -ForegroundColor Green
}

# Check PATH
Write-Host ""
Write-Host "[4/5] Checking PATH..." -ForegroundColor Yellow
$userPath = [Environment]::GetEnvironmentVariable("PATH", "User")
$machinePath = [Environment]::GetEnvironmentVariable("PATH", "Machine")
$hasMach1InPath = $false
if ($userPath -and ($userPath -like "*mach1*" -or $userPath -like "*1mcp*")) {
    Write-Host "  ✗ Found in User PATH" -ForegroundColor Red
    $hasMach1InPath = $true
}
if ($machinePath -and ($machinePath -like "*mach1*" -or $machinePath -like "*1mcp*")) {
    Write-Host "  ✗ Found in Machine PATH" -ForegroundColor Red
    $hasMach1InPath = $true
}
if (-not $hasMach1InPath) {
    Write-Host "  ✓ Not in PATH" -ForegroundColor Green
} else {
    $clean = $false
}

# Check processes
Write-Host ""
Write-Host "[5/5] Checking processes..." -ForegroundColor Yellow
$processes = Get-Process -Name "mach1", "mach1ctl", "mcpapiserver" -ErrorAction SilentlyContinue
if ($processes) {
    Write-Host "  ✗ Found running processes:" -ForegroundColor Red
    $processes | ForEach-Object { Write-Host "    - $($_.Name) (PID: $($_.Id))" -ForegroundColor Red }
    $clean = $false
} else {
    Write-Host "  ✓ No processes running" -ForegroundColor Green
}

# Summary
Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
if ($clean) {
    Write-Host "✓ VERIFIED: 1mcp.in completely removed" -ForegroundColor Green
    Write-Host ""
    Write-Host "System is clean. You can now reinstall." -ForegroundColor Cyan
} else {
    Write-Host "⚠ WARNING: Some items still exist" -ForegroundColor Yellow
    Write-Host ""
    Write-Host "Run uninstall.ps1 as Administrator to remove everything." -ForegroundColor Yellow
}
Write-Host "========================================" -ForegroundColor Cyan
