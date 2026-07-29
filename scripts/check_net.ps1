Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$ModulePath = go list -m 2>$null
if ($LASTEXITCODE -ne 0 -or -not $ModulePath) {
    Write-Warning "[!] 'go list -m' falhou; nao foi possivel determinar o modulo"
    exit 1
}

# Alvos explicitos, nao "./...": plugins de skills despejam .go de exemplo na
# raiz do modulo, e o `go list` inteiro falha por causa de imports que nao
# resolvem — a verificacao entao NAO RODA e avisa, o que e um gate que deixou
# de gatear. internal/ e cmd/ sao exatamente os pacotes que a regra cobre.
$Rows = go list -f '{{.ImportPath}}|{{join .Imports ","}}' ./internal/... ./cmd/...
if ($LASTEXITCODE -ne 0 -or -not $Rows) {
    Write-Warning "[!] 'go list ./...' falhou ou nao retornou pacotes; verificacao nao executada"
    exit 1
}

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
Write-Output "[OK] Nenhum pacote de internal/ ou cmd/ importa rede"
