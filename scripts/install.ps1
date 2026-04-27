[CmdletBinding()]
param(
    [string]$Version = $env:ONEMCP_VERSION,
    [string]$InstallDir = $env:ONEMCP_INSTALL_DIR,
    [string]$Owner = "SaiAvinashPatoju",
    [string]$Repo = "1mcp.in"
)

$ErrorActionPreference = "Stop"
if (-not $Version) { $Version = "latest" }
if (-not $InstallDir) { $InstallDir = Join-Path $env:LOCALAPPDATA "1mcp\bin" }

$arch = if ([Environment]::Is64BitOperatingSystem) { "amd64" } else { throw "Unsupported architecture" }
$asset = "onemcp-windows-$arch.zip"
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

Write-Host "1mcp installed in $InstallDir"
Write-Host "Run `"onemcpctl start`" to launch 1mcp"