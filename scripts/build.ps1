Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$ProjectRoot = Split-Path -Parent $PSScriptRoot
$OutputDir = Join-Path $ProjectRoot "bin"

if (-not (Test-Path $OutputDir)) {
    New-Item -ItemType Directory -Path $OutputDir | Out-Null
}

$Version = git describe --tags --always --dirty 2>$null
if (-not $Version) { $Version = "dev" }

$Commit = git rev-parse --short HEAD 2>$null
if (-not $Commit) { $Commit = "unknown" }

$BuildDate = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")

$LdFlags = "-s -w -X main.version=$Version -X main.commit=$Commit -X main.buildDate=$BuildDate"
$OutputPath = Join-Path $OutputDir "gobsidian.exe"

Write-Output "[...] Compilando $Version ($Commit)"

$env:CGO_ENABLED = "0"
go build -ldflags $LdFlags -o $OutputPath ".\cmd\gobsidian"

if ($LASTEXITCODE -ne 0) {
    Write-Warning "[!] Falha na compilacao"
    exit 1
}

$SizeBytes = (Get-Item $OutputPath).Length
$SizeMB = [math]::Round($SizeBytes / 1MB, 2)
Write-Output "[OK] $OutputPath ($SizeMB MB)"
