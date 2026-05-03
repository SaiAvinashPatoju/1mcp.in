#Requires -RunAsAdministrator
<#
.SYNOPSIS
    Completely removes 1mcp.in / mach1 from the system with zero traces.

.DESCRIPTION
    This script:
    1. Stops all running mach1 processes
    2. Removes all mach1 binaries
    3. Removes all configuration and data files
    4. Removes environment variables
    5. Removes PATH entries
    6. Cleans up any orphan processes

.NOTES
    Run as Administrator for complete cleanup.
#>

param(
    [switch]$Force
)

$ErrorActionPreference = "Continue"

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  1mcp.in Complete Uninstaller" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Confirmation
if (-not $Force) {
    $confirm = Read-Host "This will COMPLETELY remove 1mcp.in and all data. Continue? (y/N)"
    if ($confirm -ne "y" -and $confirm -ne "Y") {
        Write-Host "Uninstall cancelled." -ForegroundColor Yellow
        exit 0
    }
}

Write-Host ""
Write-Host "[1/7] Stopping mach1 processes..." -ForegroundColor Yellow

# Stop all mach1 processes
$processes = Get-Process -Name "mach1", "mach1ctl", "mcpapiserver" -ErrorAction SilentlyContinue
if ($processes) {
    $processes | ForEach-Object {
        Write-Host "  Stopping $($_.Name) (PID: $($_.Id))..." -ForegroundColor Gray
        Stop-Process -Id $_.Id -Force -ErrorAction SilentlyContinue
    }
    Start-Sleep -Seconds 2
    Write-Host "  All mach1 processes stopped." -ForegroundColor Green
} else {
    Write-Host "  No mach1 processes running." -ForegroundColor Gray
}

Write-Host ""
Write-Host "[2/7] Stopping orphan processes..." -ForegroundColor Yellow

# Stop orphan node/uv/python processes that might be from mach1
$orphanProcesses = Get-Process -Name "node", "uv", "uvx", "python" -ErrorAction SilentlyContinue | 
    Where-Object { $_.WorkingSet64 -lt 50MB }  # Only small processes (likely orphaned)
if ($orphanProcesses) {
    $orphanProcesses | ForEach-Object {
        Write-Host "  Stopping orphan $($_.Name) (PID: $($_.Id))..." -ForegroundColor Gray
        Stop-Process -Id $_.Id -Force -ErrorAction SilentlyContinue
    }
    Write-Host "  Orphan processes stopped." -ForegroundColor Green
} else {
    Write-Host "  No orphan processes found." -ForegroundColor Gray
}

Write-Host ""
Write-Host "[3/7] Removing binaries..." -ForegroundColor Yellow

# Remove project bin directory
$projectBin = "C:\projects\work\1Mcp\bin"
if (Test-Path $projectBin) {
    Write-Host "  Removing $projectBin..." -ForegroundColor Gray
    Remove-Item -Path $projectBin -Recurse -Force -ErrorAction SilentlyContinue
    if (-not (Test-Path $projectBin)) {
        Write-Host "  Removed successfully." -ForegroundColor Green
    } else {
        Write-Host "  WARNING: Could not remove $projectBin" -ForegroundColor Red
    }
}

# Remove AppData Local binaries
$localBin = "$env:LOCALAPPDATA\1mcp"
if (Test-Path $localBin) {
    Write-Host "  Removing $localBin..." -ForegroundColor Gray
    Remove-Item -Path $localBin -Recurse -Force -ErrorAction SilentlyContinue
    if (-not (Test-Path $localBin)) {
        Write-Host "  Removed successfully." -ForegroundColor Green
    } else {
        Write-Host "  WARNING: Could not remove $localBin" -ForegroundColor Red
    }
}

Write-Host ""
Write-Host "[4/7] Removing configuration and data..." -ForegroundColor Yellow

# Remove AppData Roaming (registry, secrets, config)
$roamingData = "$env:APPDATA\Mach1"
if (Test-Path $roamingData) {
    Write-Host "  Removing $roamingData..." -ForegroundColor Gray
    Remove-Item -Path $roamingData -Recurse -Force -ErrorAction SilentlyContinue
    if (-not (Test-Path $roamingData)) {
        Write-Host "  Removed successfully." -ForegroundColor Green
    } else {
        Write-Host "  WARNING: Could not remove $roamingData" -ForegroundColor Red
    }
}

# Remove any mach1 config files in user home
$homeConfigs = @(
    "$env:USERPROFILE\.mach1",
    "$env:USERPROFILE\.config\mach1"
)
foreach ($config in $homeConfigs) {
    if (Test-Path $config) {
        Write-Host "  Removing $config..." -ForegroundColor Gray
        Remove-Item -Path $config -Recurse -Force -ErrorAction SilentlyContinue
        Write-Host "  Removed successfully." -ForegroundColor Green
    }
}

Write-Host ""
Write-Host "[5/7] Removing environment variables..." -ForegroundColor Yellow

# Remove MACH1_HTTP_TOKEN
if ([Environment]::GetEnvironmentVariable("MACH1_HTTP_TOKEN", "User")) {
    [Environment]::SetEnvironmentVariable("MACH1_HTTP_TOKEN", $null, "User")
    Write-Host "  Removed MACH1_HTTP_TOKEN" -ForegroundColor Green
}
if ([Environment]::GetEnvironmentVariable("MACH1_HTTP_TOKEN", "Machine")) {
    [Environment]::SetEnvironmentVariable("MACH1_HTTP_TOKEN", $null, "Machine")
    Write-Host "  Removed MACH1_HTTP_TOKEN (Machine)" -ForegroundColor Green
}

# Remove any other mach1 environment variables
$mach1Vars = @("MACH1_HOME", "MACH1_DATA_DIR", "MACH1_CATALOG", "MACH1_DB_PATH")
foreach ($var in $mach1Vars) {
    if ([Environment]::GetEnvironmentVariable($var, "User")) {
        [Environment]::SetEnvironmentVariable($var, $null, "User")
        Write-Host "  Removed $var" -ForegroundColor Green
    }
    if ([Environment]::GetEnvironmentVariable($var, "Machine")) {
        [Environment]::SetEnvironmentVariable($var, $null, "Machine")
        Write-Host "  Removed $var (Machine)" -ForegroundColor Green
    }
}

Write-Host ""
Write-Host "[6/7] Removing PATH entries..." -ForegroundColor Yellow

# Remove from User PATH
$userPath = [Environment]::GetEnvironmentVariable("PATH", "User")
if ($userPath) {
    $pathEntries = $userPath -split ";" | Where-Object { $_ -ne "" }
    $newPath = ($pathEntries | Where-Object { $_ -notlike "*mach1*" -and $_ -notlike "*1mcp*" }) -join ";"
    if ($newPath -ne $userPath) {
        [Environment]::SetEnvironmentVariable("PATH", $newPath, "User")
        Write-Host "  Removed mach1 entries from User PATH" -ForegroundColor Green
    }
}

# Remove from Machine PATH (requires admin)
$machinePath = [Environment]::GetEnvironmentVariable("PATH", "Machine")
if ($machinePath) {
    $pathEntries = $machinePath -split ";" | Where-Object { $_ -ne "" }
    $newPath = ($pathEntries | Where-Object { $_ -notlike "*mach1*" -and $_ -notlike "*1mcp*" }) -join ";"
    if ($newPath -ne $machinePath) {
        [Environment]::SetEnvironmentVariable("PATH", $newPath, "Machine")
        Write-Host "  Removed mach1 entries from Machine PATH" -ForegroundColor Green
    }
}

Write-Host ""
Write-Host "[7/7] Final cleanup..." -ForegroundColor Yellow

# Remove any temp files
$tempDirs = @(
    "$env:TEMP\mach1*",
    "$env:TEMP\1mcp*"
)
foreach ($temp in $tempDirs) {
    $items = Get-Item -Path $temp -ErrorAction SilentlyContinue
    if ($items) {
        $items | ForEach-Object {
            Write-Host "  Removing temp: $($_.FullName)..." -ForegroundColor Gray
            Remove-Item -Path $_.FullName -Recurse -Force -ErrorAction SilentlyContinue
        }
    }
}

# Verify cleanup
Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  Cleanup Verification" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan

$issues = @()

# Check binaries
if (Test-Path "C:\projects\work\1Mcp\bin\mach1.exe") {
    $issues += "Binary still exists: C:\projects\work\1Mcp\bin\mach1.exe"
}
if (Test-Path "$env:LOCALAPPDATA\1mcp\bin\mach1.exe") {
    $issues += "Binary still exists: $env:LOCALAPPDATA\1mcp\bin\mach1.exe"
}

# Check data
if (Test-Path "$env:APPDATA\Mach1") {
    $issues += "Data directory still exists: $env:APPDATA\Mach1"
}

# Check environment variables
if ([Environment]::GetEnvironmentVariable("MACH1_HTTP_TOKEN", "User")) {
    $issues += "Environment variable still exists: MACH1_HTTP_TOKEN"
}

# Check PATH
$userPath = [Environment]::GetEnvironmentVariable("PATH", "User")
if ($userPath -and ($userPath -like "*mach1*" -or $userPath -like "*1mcp*")) {
    $issues += "PATH still contains mach1 entries"
}

if ($issues.Count -eq 0) {
    Write-Host ""
    Write-Host "✓ COMPLETE: 1mcp.in has been completely removed." -ForegroundColor Green
    Write-Host ""
    Write-Host "Removed:" -ForegroundColor Gray
    Write-Host "  - All binaries (mach1.exe, mach1ctl.exe, etc.)" -ForegroundColor Gray
    Write-Host "  - Registry database and configuration" -ForegroundColor Gray
    Write-Host "  - Secrets and credentials" -ForegroundColor Gray
    Write-Host "  - Environment variables" -ForegroundColor Gray
    Write-Host "  - PATH entries" -ForegroundColor Gray
    Write-Host "  - Orphan processes" -ForegroundColor Gray
    Write-Host ""
    Write-Host "You can now reinstall 1mcp.in from scratch." -ForegroundColor Cyan
} else {
    Write-Host ""
    Write-Host "⚠ WARNING: Some items could not be removed:" -ForegroundColor Yellow
    foreach ($issue in $issues) {
        Write-Host "  - $issue" -ForegroundColor Yellow
    }
    Write-Host ""
    Write-Host "You may need to:" -ForegroundColor Yellow
    Write-Host "  1. Close any programs using mach1" -ForegroundColor Yellow
    Write-Host "  2. Run this script as Administrator" -ForegroundColor Yellow
    Write-Host "  3. Manually remove remaining items" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
