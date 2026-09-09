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

    3. TODO PIN DE GO SATISFAZ A DIRETIVA DO go.mod.

       Os workflows fixam a versao do Go com `go-version:` e o setup-go
       exporta GOTOOLCHAIN=local -- de proposito, para que a toolchain
       instalada seja a que compila, e nao uma que o go.mod baixe por conta.
       O preco disso e que um pin ABAIXO da diretiva nao degrada: ele para o
       job com "go.mod requires go >= X (running Y; GOTOOLCHAIN=local)".

       Aconteceu em 2026-09-09, no mesmo dia em que a diretiva subiu para
       1.27.0: ci.yml e bench.yml continuaram fixando '1.25' em dez lugares, e
       o CI caiu em ONZE dos catorze jobs. O unico que passou foi o gate, que
       fixa 1.27.1 -- e passar deu a impressao errada, porque ele e justamente
       quem roda o verify inteiro. Um gate verde ao lado de onze jobs vermelhos
       e o pior sinal possivel.

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
$GoMod = Join-Path $Raiz 'go.mod'

foreach ($f in @($Verify, $CI, $Gate, $Release, $GoMod)) {
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

# ----------------------------------------------------------------------
# 3. todo pin de Go satisfaz a diretiva do go.mod
# ----------------------------------------------------------------------
#
# Versao como numero comparavel: '1.27.1' -> 1.027001. Partes ausentes valem
# zero, entao '1.25' e 1.25.0 -- que e como o proprio Go a le.
function ComoNumero {
    param([string]$V)
    $p = @($V -split '\.')
    $n = 0.0
    for ($i = 0; $i -lt 3; $i++) {
        $parte = 0
        if ($i -lt $p.Count) { [void][int]::TryParse($p[$i], [ref]$parte) }
        $n += $parte / [math]::Pow(1000, $i)
    }
    return $n
}

$TextoGoMod = Get-Content -Path $GoMod -Raw -Encoding UTF8
$mDiretiva = [regex]::Match($TextoGoMod, '(?m)^go\s+(?<v>\d+(\.\d+)*)\s*$')
if (-not $mDiretiva.Success) {
    $Problemas += "nao achei a diretiva 'go' no go.mod"
}
else {
    $diretiva = $mDiretiva.Groups['v'].Value
    $alvo = ComoNumero $diretiva
    foreach ($y in @('ci.yml', 'gate.yml', 'release.yml', 'bench.yml')) {
        $caminho = Join-Path $Raiz ".github/workflows/$y"
        if (-not (Test-Path $caminho)) { continue }
        $codigo = SemComentarios $caminho
        foreach ($m in [regex]::Matches($codigo, "(?m)(?:go-version:\s*|default:\s*)'(?<v>\d+(\.\d+)*)'")) {
            $v = $m.Groups['v'].Value
            if ((ComoNumero $v) -lt $alvo) {
                $Problemas += "$y fixa Go $v, abaixo da diretiva go $diretiva do go.mod"
                $Problemas += "    com GOTOOLCHAIN=local o job nao degrada: ele para com 'go.mod requires go >= $diretiva'"
            }
        }
    }
}

if ($Problemas.Count -gt 0) {
    Write-Output "[!] pins inconsistentes:"
    $Problemas | ForEach-Object { Write-Output "     $_" }
    exit 1
}

if (-not $Silencioso) {
    Write-Output "[OK] pins concordam: golangci-lint $($Versoes[0]); release cobra e compila com a mesma toolchain ($($Gos[0])); todo pin de Go satisfaz a diretiva go $diretiva"
}
exit 0
