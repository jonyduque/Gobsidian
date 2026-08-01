Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$ProjectRoot = Split-Path -Parent $PSScriptRoot
Push-Location $ProjectRoot

$TempBin = Join-Path $env:TEMP "netcheck_$([guid]::NewGuid().ToString("N")).exe"

try {
    # 1. Compila o netcheck como vettool
    go build -o $TempBin ./tools/netcheck/cmd/netcheck
    if ($LASTEXITCODE -ne 0) {
        Write-Warning "[!] Falha ao compilar tools/netcheck/cmd/netcheck"
        exit 1
    }

    # 2. Executa a analise semantica com go vet para os 3 alvos GOOS
    $Alvos = @("./internal/...", "./cmd/...")
    $OSList = @("windows", "linux", "darwin")

    foreach ($targetOS in $OSList) {
        $env:GOOS = $targetOS
        try {
            $Out = go vet "-vettool=$TempBin" @Alvos 2>&1
            if ($LASTEXITCODE -ne 0) {
                Write-Warning "[!] netcheck reprovou em GOOS=$targetOS"
                if ($Out) { $Out | ForEach-Object { Write-Output "     $_" } }
                exit 1
            }
        }
        finally {
            Remove-Item Env:\GOOS -ErrorAction SilentlyContinue
        }
    }

    # 3. Mantem a verificacao textual de retrocompatibilidade
    $ModulePath = go list -m 2>$null
    if ($LASTEXITCODE -eq 0 -and $ModulePath) {
        $Rows = go list -f '{{.ImportPath}}|{{join .Imports ","}}' ./internal/... ./cmd/...
        if ($LASTEXITCODE -eq 0 -and $Rows) {
            $Scoped = $Rows | Where-Object {
                $Pkg = ($_ -split "\|", 2)[0]
                $Pkg -eq "$ModulePath/internal" -or $Pkg -like "$ModulePath/internal/*" -or
                $Pkg -eq "$ModulePath/cmd" -or $Pkg -like "$ModulePath/cmd/*"
            }

            $Offenders = @()
            foreach ($Row in $Scoped) {
                $Parts = $Row -split "\|", 2
                $Pkg = $Parts[0]

                $Imports = @()
                if ($Parts.Count -gt 1 -and $Parts[1]) { $Imports = $Parts[1] -split "," }

                $Net = $Imports | Where-Object { $_ -eq "net" -or $_ -like "net/*" }
                if ($Net) { $Offenders += "$Pkg -> $($Net -join ', ')" }
            }

            if ($Offenders) {
                Write-Warning "[!] Pacote do produto importando rede:"
                $Offenders | ForEach-Object { Write-Output "    $_" }
                exit 1
            }
        }
    }

    Write-Output "[OK] Nenhum pacote de internal/ ou cmd/ importa rede (verificado via netcheck vettool em windows, linux, darwin)"
}
finally {
    if (Test-Path $TempBin) {
        Remove-Item $TempBin -Force -ErrorAction SilentlyContinue
    }
    Pop-Location
}
