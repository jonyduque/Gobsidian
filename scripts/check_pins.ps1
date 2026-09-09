#Requires -Version 7.0
<#
.SYNOPSIS
    Prova que as versoes fixadas concordam entre si.

.DESCRIPTION
    Duas invariantes que existiam so como intencao, e as duas quebraram em
    2026-09-09.

    1. A VERSAO DO golangci-lint E A MESMA EM TODA PARTE.

       Ela aparece em quatro lugares: a condicao e a mensagem dentro de
       verify.ps1, os dois jobs de lint do ci.yml, e o default do gate.yml. Ao
       mover o pin de v2.12.2 para v2.13.2, uma substituicao literal pegou a
       mensagem e NAO pegou a condicao, escrita com pontos escapados. O
       resultado: o erro dizia "fora da versao fixada (v2.13.2)" enquanto
       exigia a v2.12.2. Um pin que discorda de si mesmo e pior que pin nenhum
       -- ele afirma um numero e cobra outro.

    2. O GATE DO RELEASE RODA NA TOOLCHAIN QUE PRODUZ O BINARIO.

       release.yml fixa go-version em dois lugares: o gate e o job que compila
       os binarios publicados. O comentario de la diz, em letra, que o gate
       "tem de cobrar a toolchain que de fato produz o artefato" -- e nada
       garantia isso. As duas ja divergiram nesta sessao, quando o gate ficou
       no piso do go.mod e o build subiu para 1.27.1.

    COMENTARIO NAO E PIN. Os dois arquivos citam versoes antigas de proposito
    -- a v1.64.8 que recusava um go.mod declarando 1.25.0, a v2.12.2 que
    entrava em panic com a toolchain 1.27 -- e essas mencoes sao a historia que
    justifica a conferencia. Um gate que as tratasse como pin reprovaria para
    sempre, e um gate que reprova sempre e desligado na primeira semana.

.NOTES
    Saida em ASCII puro. Sai 1 se algum pin discordar.

    -Raiz existe para o check_gates.ps1 rodar contra copias mutadas, em vez de
    contra o repositorio.
#>
[CmdletBinding()]
param(
    [string]$Raiz,
    [switch]$Silencioso
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

if (-not $Raiz) { $Raiz = Split-Path -Parent $PSScriptRoot }

$Verify = Join-Path $Raiz 'scripts/verify.ps1'
$CI = Join-Path $Raiz '.github/workflows/ci.yml'
$Gate = Join-Path $Raiz '.github/workflows/gate.yml'
$Release = Join-Path $Raiz '.github/workflows/release.yml'

foreach ($f in @($Verify, $CI, $Gate, $Release)) {
    if (-not (Test-Path $f)) {
        Write-Output "[!] arquivo ausente: $f"
        exit 1
    }
}

# SemComentarios devolve so as linhas de codigo de um arquivo.
function SemComentarios {
    param([string]$Caminho)
    $linhas = Get-Content -Path $Caminho -Encoding UTF8 | Where-Object { $_ -notmatch '^\s*#' }
    return ($linhas -join "`n")
}

$Problemas = @()

# ----------------------------------------------------------------------
# 1. golangci-lint: um numero so
# ----------------------------------------------------------------------
$Achados = @{}

function Registrar-Versao {
    param([string]$Versao, [string]$Onde)
    if (-not $Achados.ContainsKey($Versao)) { $Achados[$Versao] = @() }
    $Achados[$Versao] += $Onde
}

$CodigoVerify = SemComentarios $Verify

# A condicao, com pontos escapados. E a forma que escapou da substituicao
# literal e deixou mensagem e condicao discordando.
foreach ($m in [regex]::Matches($CodigoVerify, '"(\d+)\\\.(\d+)\\\.(\d+)"')) {
    Registrar-Versao "v$($m.Groups[1].Value).$($m.Groups[2].Value).$($m.Groups[3].Value)" 'verify.ps1 (condicao, escapada)'
}
# A mensagem do throw.
foreach ($m in [regex]::Matches($CodigoVerify, 'v(\d+)\.(\d+)\.(\d+)')) {
    Registrar-Versao "v$($m.Groups[1].Value).$($m.Groups[2].Value).$($m.Groups[3].Value)" 'verify.ps1 (mensagem)'
}

foreach ($par in @(@{ f = $CI; n = 'ci.yml' }, @{ f = $Gate; n = 'gate.yml' })) {
    $codigo = SemComentarios $par.f
    foreach ($m in [regex]::Matches($codigo, "(?m)(?:version:\s*|default:\s*')(v\d+\.\d+\.\d+)")) {
        Registrar-Versao $m.Groups[1].Value $par.n
    }
}

$Versoes = @($Achados.Keys | Sort-Object)
if ($Versoes.Count -eq 0) {
    $Problemas += "nao achei nenhuma versao de golangci-lint fixada -- o pin sumiu?"
}
elseif ($Versoes.Count -gt 1) {
    $Problemas += "o pin do golangci-lint discorda de si mesmo:"
    foreach ($v in $Versoes) {
        $onde = @($Achados[$v] | Sort-Object -Unique) -join ', '
        $Problemas += "    $v  em  $onde"
    }
}

# ----------------------------------------------------------------------
# 2. release.yml: gate e build na MESMA toolchain
# ----------------------------------------------------------------------
$CodigoRelease = SemComentarios $Release
$Gos = @()
foreach ($m in [regex]::Matches($CodigoRelease, "(?m)go-version:\s*'([^']+)'")) {
    $Gos += $m.Groups[1].Value
}

if ($Gos.Count -lt 2) {
    $Problemas += "release.yml deveria fixar go-version em dois lugares (gate e build); achei $($Gos.Count)"
}
else {
    $unicas = @($Gos | Sort-Object -Unique)
    if ($unicas.Count -gt 1) {
        $Problemas += "release.yml usa toolchains diferentes no gate e no build: $($unicas -join ', ')"
        $Problemas += "    o gate tem de cobrar a toolchain que produz o artefato -- e o proprio comentario de la que diz isso"
    }
}

if ($Problemas.Count -gt 0) {
    Write-Output "[!] pins inconsistentes:"
    $Problemas | ForEach-Object { Write-Output "     $_" }
    exit 1
}

if (-not $Silencioso) {
    Write-Output "[OK] pins concordam: golangci-lint $($Versoes[0]); release cobra e compila com a mesma toolchain ($($Gos[0]))"
}
exit 0
