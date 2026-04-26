# Build all OneMCP binaries into ./bin
# Usage:  pwsh ./scripts/build.ps1
[CmdletBinding()]
param(
    [string]$OutDir
)

$ErrorActionPreference = "Stop"
$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$repoRoot = Resolve-Path (Join-Path $scriptDir "..")
$serviceDir = Join-Path $repoRoot "services\central-mcp"
if (-not $OutDir) { $OutDir = Join-Path $repoRoot "bin" }
if (-not (Test-Path $OutDir)) { New-Item -ItemType Directory -Force -Path $OutDir | Out-Null }
$OutDir = (Resolve-Path $OutDir).Path

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    throw "Go toolchain not found in PATH. Install Go 1.22+ and re-run."
}

Push-Location $serviceDir
try {
    Write-Host "go mod tidy"
    go mod tidy
    if ($LASTEXITCODE -ne 0) { throw "go mod tidy failed" }

    $cmds = @("centralmcpd", "onemcpctl", "onemcpe2e", "stubmcp")
    foreach ($c in $cmds) {
        $out = Join-Path $OutDir "$c.exe"
        Write-Host "go build -> $out"
        go build -trimpath -ldflags "-s -w" -o $out ".\cmd\$c"
        if ($LASTEXITCODE -ne 0) { throw "build $c failed" }
    }

    Write-Host "go vet"
    go vet ./...
    if ($LASTEXITCODE -ne 0) { throw "go vet reported issues" }

    Write-Host "go test"
    go test ./...
    if ($LASTEXITCODE -ne 0) { throw "go test failures" }
}
finally {
    Pop-Location
}

Write-Host "`nBuild complete. Binaries in: $OutDir"
Get-ChildItem $OutDir -Filter *.exe | Format-Table Name, Length, LastWriteTime
