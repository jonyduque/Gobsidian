#Requires -Version 7.0
<#
.SYNOPSIS
    Confere que toda ancora interna do README aponta para um heading que
    existe, e que toda secao de nivel 2 e alcancavel pela navegacao do topo.

.DESCRIPTION
    Link interno quebrado nao quebra build nenhum: ele rola a pagina para
    lugar nenhum, e so quem clica descobre. Tres secoes do README ficaram
    sem link de navegacao por um marco inteiro (Daemon, Compatibilidade e
    Desenvolvimento) sem ninguem notar.

    A ancora e gerada pela regra do GitHub, e ela tem duas sutilezas que
    derrubaram as duas primeiras versoes deste script:

      1. O emoji e removido, mas o ESPACO que vinha depois dele permanece --
         e e esse espaco que vira o hifen inicial de "#-visao-geral". Um
         .Trim() antes da troca de espaco por hifen faz o script acusar
         todas as ancoras boas como quebradas.

      2. Emoji como U+2699 vem acompanhado do SELETOR DE VARIACAO U+FE0F,
         que e categoria Mn. Preservar categoria M para nao perder acento
         mantem o seletor tambem, e a ancora sai diferente da que o GitHub
         gera. Normalizar para NFC resolve: acento precomposto passa por
         IsLetterOrDigit, e nao ha marca combinante a preservar.

    CODIGO DE SAIDA:
      0 -> toda ancora resolve e toda secao H2 tem link
      1 -> ha ancora quebrada ou secao orfa
      2 -> o script nao conseguiu rodar

.EXAMPLE
    pwsh -File scripts/check_readme_anchors.ps1
#>
[CmdletBinding()]
param(
    [string]$Path
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$ProjectRoot = Split-Path -Parent $PSScriptRoot
if (-not $Path) { $Path = Join-Path $ProjectRoot 'README.md' }

if (-not (Test-Path $Path)) {
    Write-Warning "[!] arquivo nao encontrado: $Path"
    exit 2
}

function Get-Slug {
    param([string]$Titulo)

    $s = [Text.NormalizationForm]::FormC
    $t = $Titulo.Normalize($s).Trim().ToLowerInvariant()

    $sb = [Text.StringBuilder]::new()
    foreach ($c in $t.ToCharArray()) {
        if ([char]::IsLetterOrDigit($c) -or $c -eq ' ' -or $c -eq '-' -or $c -eq '_') {
            [void]$sb.Append($c)
        }
    }
    # Sem Trim() aqui: o espaco deixado pelo emoji removido e o hifen inicial.
    return $sb.ToString().Replace(' ', '-')
}

$Linhas = @(Get-Content -Path $Path -Encoding utf8)

$Headings = @()
$H2 = @()
foreach ($L in $Linhas) {
    if ($L -match '^(#{2,})\s+(.+)$') {
        $Headings += $Matches[2]
        if ($Matches[1].Length -eq 2) { $H2 += $Matches[2] }
    }
}

$Slugs = [System.Collections.Generic.HashSet[string]]::new(
    [string[]]($Headings | ForEach-Object { Get-Slug $_ }), [System.StringComparer]::Ordinal)

$Links = @()
foreach ($L in $Linhas) {
    foreach ($m in [regex]::Matches($L, '\]\(#([^)]+)\)')) {
        $Links += $m.Groups[1].Value
    }
}

Write-Output "[i] $($Headings.Count) heading(s), $($Links.Count) link(s) interno(s)."

$Falhou = $false

foreach ($a in ($Links | Sort-Object -Unique)) {
    if (-not $Slugs.Contains($a)) {
        Write-Warning "[!] ancora quebrada: #$a nao corresponde a nenhum heading"
        $Falhou = $true
    }
}

$LinksSet = [System.Collections.Generic.HashSet[string]]::new(
    [string[]]$Links, [System.StringComparer]::Ordinal)

foreach ($h in $H2) {
    $slug = Get-Slug $h
    if (-not $LinksSet.Contains($slug)) {
        Write-Warning "[!] secao sem link na navegacao: '$h' (esperava #$slug)"
        $Falhou = $true
    }
}

if ($Falhou) { exit 1 }

Write-Output "[OK] toda ancora resolve e toda secao H2 e alcancavel pela navegacao."
exit 0
