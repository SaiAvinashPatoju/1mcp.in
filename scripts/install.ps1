[CmdletBinding()]
param(
    [string]$Version = $env:MACH1_VERSION,
    [string]$InstallDir = $env:MACH1_INSTALL_DIR,
    [string]$Owner = "SaiAvinashPatoju",
    [string]$Repo = "1mcp.in"
)

$ErrorActionPreference = "Stop"

# 1. Check Node.js
if (-not (Get-Command "npm" -ErrorAction SilentlyContinue)) {
    Write-Host "Node.js (npm) not found. Installing via winget..."
    winget install OpenJS.NodeJS
    Write-Host "Node.js installed. You may need to restart your terminal later for paths to fully apply."
}

# 2. Check uv
if (-not (Get-Command "uv" -ErrorAction SilentlyContinue)) {
    Write-Host "uv not found. Installing via astral.sh..."
    Invoke-RestMethod -Uri "https://astral.sh/uv/install.ps1" | Invoke-Expression
}

if (-not $Version) { $Version = "latest" }
if (-not $InstallDir) { $InstallDir = Join-Path $env:LOCALAPPDATA "1mcp\bin" }

$arch = if ([Environment]::Is64BitOperatingSystem) { "amd64" } else { throw "Unsupported architecture" }
$asset = "mach1-windows-$arch.zip"
$api = if ($Version -eq "latest") {
    "https://api.github.com/repos/$Owner/$Repo/releases/latest"
} else {
    "https://api.github.com/repos/$Owner/$Repo/releases/tags/$Version"
}

$release = Invoke-RestMethod -Uri $api -Headers @{ "User-Agent" = "1mcp-installer" }
$download = ($release.assets | Where-Object { $_.name -eq $asset } | Select-Object -First 1).browser_download_url
if (-not $download) { throw "Could not find release asset $asset" }

New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
$tmp = Join-Path ([IO.Path]::GetTempPath()) ([IO.Path]::GetRandomFileName())
New-Item -ItemType Directory -Force -Path $tmp | Out-Null
try {
    $zip = Join-Path $tmp $asset
    Invoke-WebRequest -Uri $download -OutFile $zip
    Expand-Archive -Path $zip -DestinationPath $InstallDir -Force
}
finally {
    Remove-Item -Recurse -Force $tmp -ErrorAction SilentlyContinue
}

$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if (($userPath -split ';') -notcontains $InstallDir) {
    [Environment]::SetEnvironmentVariable("Path", "$InstallDir;$userPath", "User")
    $env:Path = "$InstallDir;$env:Path"
    Write-Host "Added $InstallDir to user PATH"
}

# 3. Add mach1ctl temporarily for injecting rules
$env:Path = "$InstallDir;$env:Path"

# 4. Inject rules for AI clients
Write-Host "Injecting rule files for AI clients..."
& "$InstallDir\mach1ctl.exe" inject-rules vscode -ErrorAction SilentlyContinue
& "$InstallDir\mach1ctl.exe" inject-rules cursor -ErrorAction SilentlyContinue
& "$InstallDir\mach1ctl.exe" inject-rules windsurf -ErrorAction SilentlyContinue

Write-Host "1mcp.in installed in $InstallDir"
Write-Host "Run `"mach1ctl start`" to launch 1mcp.in"