param()

$ErrorActionPreference = "Stop"

# Ensure we run from the repo/backend directory (where this script lives)
$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Definition
Set-Location $scriptDir

if (-not (Test-Path -Path "tmp")) {
    New-Item -ItemType Directory -Path "tmp" | Out-Null
}

Write-Host "Building backend executable to tmp\main.exe..."

try {
    # Build the server package (cmd/server) into tmp/main.exe
    & go build -o tmp/main.exe ./cmd/server
    Write-Host "Build succeeded.\n"
} catch {
    Write-Error "Build failed: $_"
    exit 1
}
