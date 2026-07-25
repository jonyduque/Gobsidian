Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$Rows = go list -f '{{.ImportPath}}|{{join .Imports ","}}' ./...
$Offenders = @()

foreach ($Row in $Rows) {
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
