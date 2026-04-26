# Run the full E2E pass against the in-tree stub MCP (no npm/python/docker
# required). Builds binaries first if missing.
[CmdletBinding()]
param(
    [string]$OutDir,
    [string]$Report
)

$ErrorActionPreference = "Stop"
$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$repoRoot = Resolve-Path (Join-Path $scriptDir "..")
if (-not $OutDir) { $OutDir = Join-Path $repoRoot "bin" }
if (-not $Report) { $Report = Join-Path $repoRoot "e2e-report.md" }
if (-not (Test-Path $OutDir)) { New-Item -ItemType Directory -Force -Path $OutDir | Out-Null }
$OutDir = (Resolve-Path $OutDir).Path

$central = Join-Path $OutDir "centralmcpd.exe"
$e2e     = Join-Path $OutDir "onemcpe2e.exe"
$stub    = Join-Path $OutDir "stubmcp.exe"

if (-not (Test-Path $central) -or -not (Test-Path $e2e) -or -not (Test-Path $stub)) {
    & (Join-Path $PSScriptRoot "build.ps1") -OutDir $OutDir
}

$env:STUBMCP_BIN = $stub
$config = Join-Path $repoRoot "services\central-mcp\test\e2e\config.stub.json"
$smokes = Join-Path $repoRoot "services\central-mcp\test\e2e\smokes.stub.json"

Write-Host "Running E2E with stub MCP..."
& $e2e --bin $central --config $config --smoke $smokes --out $Report
$exit = $LASTEXITCODE

Write-Host "`n----- $Report -----"
Get-Content $Report
Write-Host "----- end -----`n"

exit $exit
