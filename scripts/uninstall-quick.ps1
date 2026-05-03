<#
.SYNOPSIS
    Quick uninstall of 1mcp.in (no admin required)
#>

$ErrorActionPreference = "Continue"

Write-Host "Quick uninstalling 1mcp.in..." -ForegroundColor Cyan

# Stop processes
Write-Host "Stopping processes..." -ForegroundColor Yellow
Get-Process -Name "mach1", "mach1ctl", "mcpapiserver" -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
Start-Sleep -Seconds 1

# Remove project binaries
$projectBin = "C:\projects\work\1Mcp\bin"
if (Test-Path $projectBin) {
    Write-Host "Removing $projectBin..." -ForegroundColor Gray
    Remove-Item -Path $projectBin -Recurse -Force -ErrorAction SilentlyContinue
}

# Remove AppData Local
$localBin = "$env:LOCALAPPDATA\1mcp"
if (Test-Path $localBin) {
    Write-Host "Removing $localBin..." -ForegroundColor Gray
    Remove-Item -Path $localBin -Recurse -Force -ErrorAction SilentlyContinue
}

# Remove AppData Roaming
$roamingData = "$env:APPDATA\Mach1"
if (Test-Path $roamingData) {
    Write-Host "Removing $roamingData..." -ForegroundColor Gray
    Remove-Item -Path $roamingData -Recurse -Force -ErrorAction SilentlyContinue
}

# Remove environment variable
if ([Environment]::GetEnvironmentVariable("MACH1_HTTP_TOKEN", "User")) {
    [Environment]::SetEnvironmentVariable("MACH1_HTTP_TOKEN", $null, "User")
    Write-Host "Removed MACH1_HTTP_TOKEN" -ForegroundColor Gray
}

# Remove from PATH
$userPath = [Environment]::GetEnvironmentVariable("PATH", "User")
if ($userPath) {
    $pathEntries = $userPath -split ";" | Where-Object { $_ -ne "" }
    $newPath = ($pathEntries | Where-Object { $_ -notlike "*mach1*" -and $_ -notlike "*1mcp*" }) -join ";"
    if ($newPath -ne $userPath) {
        [Environment]::SetEnvironmentVariable("PATH", $newPath, "User")
        Write-Host "Removed from PATH" -ForegroundColor Gray
    }
}

Write-Host ""
Write-Host "[OK] Quick uninstall complete." -ForegroundColor Green
Write-Host "  Note: Run uninstall.ps1 as Administrator for 100% cleanup." -ForegroundColor Yellow
