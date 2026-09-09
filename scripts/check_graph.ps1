#Requires -Version 7.0
<#
.SYNOPSIS
    Prova que o grafo de dependencias do CLAUDE.md e o que os imports dizem.

.DESCRIPTION
    O CLAUDE.md carrega o grafo de dependencias de `internal/` e afirma, em
    letra, que ele foi "re-extraido dos imports de producao". Ele mesmo avisa
    por que isso precisa de prova:

        "o paragrafo que descreve o grafo nao vale mais que os imports: dizer
         'conferido' nao e conferir, e as duas versoes anteriores diziam."

    Em 2026-09-09 uma TERCEIRA versao disse. O bloco ganhou
    `selfupdate -> config`, e `go list` dizia folha; o erro foi pego por acaso,
    ao rodar o comando por outro motivo. Antes disso, uma versao somava as
    arestas de teste as de producao e atribuia ao `daemon` um conhecimento de
    `service` e `vault` que ele nao tem, e outra omitia `text` inteiro.

    Tres redacoes erradas, tres vezes a mesma causa: a unica coisa que separava
    o documento da verdade era alguem lembrar de conferir.

    Este script confere. Ele le o bloco do CLAUDE.md e o compara com
    `go list -f '{{.Imports}}'` sobre ./internal/..., com GOOS=windows -- o
    mesmo comando que o proprio documento cita, e que NAO enxerga arquivo
    `_test.go`, porque as arestas so de teste ficam fora do grafo de proposito.

.NOTES
    Saida em ASCII puro. Sai 1 se o documento e os imports discordarem.

    -Arquivo e -Bloco existem para o check_gates.ps1 poder rodar este script
    contra fixtures, em vez de contra o repositorio.
#>
[CmdletBinding()]
param(
    [string]$Arquivo,
    [switch]$Silencioso
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$ProjectRoot = Split-Path -Parent $PSScriptRoot
if (-not $Arquivo) { $Arquivo = Join-Path $ProjectRoot 'CLAUDE.md' }

Push-Location $ProjectRoot
try {
    # ------------------------------------------------------------------
    # 1. O que o documento AFIRMA
    # ------------------------------------------------------------------
    #
    # O bloco comeca na linha de folhas (a que termina em "folhas") e vai ate a
    # cerca que a fecha. Ancorar na palavra, e nao num numero de linha: o
    # documento e editado toda semana.
    if (-not (Test-Path $Arquivo)) {
        Write-Output "[!] arquivo nao encontrado: $Arquivo"
        exit 1
    }
    $Linhas = Get-Content -Path $Arquivo -Encoding UTF8

    $Inicio = -1
    for ($i = 0; $i -lt $Linhas.Count; $i++) {
        if ($Linhas[$i] -match '\s+folhas\s*$') { $Inicio = $i; break }
    }
    if ($Inicio -lt 0) {
        Write-Output "[!] nao achei a linha de folhas do grafo em $Arquivo"
        exit 1
    }

    $Declarado = @{}
    $Folhas = @()

    # A linha de folhas: "text  vault  config  console  lifecycle      folhas"
    foreach ($tok in ($Linhas[$Inicio] -split '\s+')) {
        if ($tok -and $tok -ne 'folhas') { $Folhas += $tok }
    }
    foreach ($f in $Folhas) { $Declarado[$f] = @() }

    for ($i = $Inicio + 1; $i -lt $Linhas.Count; $i++) {
        $l = $Linhas[$i]
        if ($l -match '^\s*```') { break }
        # "nome   -> a, b, c"  ou  "nome -> (folha)"
        if ($l -match '^\s*([a-z]+)\s*(?:->|→)\s*(.+?)\s*$') {
            $nome = $Matches[1]
            $resto = $Matches[2]
            if ($resto -match '^\(folha\)$') {
                $Declarado[$nome] = @()
            }
            else {
                $Declarado[$nome] = @($resto -split ',' | ForEach-Object { $_.Trim() } | Where-Object { $_ })
            }
        }
    }

    if ($Declarado.Count -eq 0) {
        Write-Output "[!] o bloco do grafo esta vazio em $Arquivo"
        exit 1
    }

    # ------------------------------------------------------------------
    # 2. O que os IMPORTS dizem
    # ------------------------------------------------------------------
    $ModulePath = go list -m 2>$null
    if ($LASTEXITCODE -ne 0 -or -not $ModulePath) {
        Write-Output "[!] nao consegui resolver o modulo com 'go list -m'"
        exit 1
    }

    $env:GOOS = 'windows'
    try {
        $Rows = go list -f '{{.ImportPath}}|{{join .Imports ","}}' ./internal/... 2>&1
        if ($LASTEXITCODE -ne 0) {
            Write-Output "[!] 'go list' falhou:"
            $Rows | ForEach-Object { Write-Output "     $_" }
            exit 1
        }
    }
    finally {
        Remove-Item Env:\GOOS -ErrorAction SilentlyContinue
    }

    $Real = @{}
    foreach ($Row in $Rows) {
        $Parts = $Row -split '\|', 2
        $Pkg = ($Parts[0] -split '/')[-1]
        $Deps = @()
        if ($Parts.Count -gt 1 -and $Parts[1]) {
            foreach ($imp in ($Parts[1] -split ',')) {
                if ($imp -like "$ModulePath/internal/*") { $Deps += ($imp -split '/')[-1] }
            }
        }
        $Real[$Pkg] = @($Deps | Sort-Object -Unique)
    }

    # ------------------------------------------------------------------
    # 3. Comparar
    # ------------------------------------------------------------------
    #
    # `vaulttest` fica FORA: o CLAUDE.md o documenta em bloco proprio, com a
    # justificativa de que nenhum arquivo de producao o importa. Inclui-lo aqui
    # exigiria que o bloco principal o listasse, contradizendo o documento.
    $Ignorados = @('vaulttest')

    $Problemas = @()

    foreach ($pkg in ($Real.Keys | Sort-Object)) {
        if ($Ignorados -contains $pkg) { continue }
        if (-not $Declarado.ContainsKey($pkg)) {
            $Problemas += "pacote $pkg existe em internal/ e NAO esta no grafo do documento"
            continue
        }
        $d = @($Declarado[$pkg] | Sort-Object -Unique)
        $r = @($Real[$pkg])
        $sobrando = @($d | Where-Object { $r -notcontains $_ })
        $faltando = @($r | Where-Object { $d -notcontains $_ })
        foreach ($x in $sobrando) { $Problemas += "$pkg -> $x esta no documento e NAO nos imports" }
        foreach ($x in $faltando) { $Problemas += "$pkg -> $x esta nos imports e NAO no documento" }
    }

    foreach ($pkg in ($Declarado.Keys | Sort-Object)) {
        if ($Ignorados -contains $pkg) { continue }
        if (-not $Real.ContainsKey($pkg)) {
            $Problemas += "o grafo do documento cita $pkg, que nao existe em internal/"
        }
    }

    if ($Problemas.Count -gt 0) {
        Write-Output "[!] o grafo do CLAUDE.md nao bate com os imports:"
        $Problemas | ForEach-Object { Write-Output "     $_" }
        Write-Output ""
        Write-Output "     Aresta nova precisa de justificativa escrita -- ver as regras no proprio CLAUDE.md."
        exit 1
    }

    if (-not $Silencioso) {
        Write-Output "[OK] o grafo do CLAUDE.md bate com os imports de producao ($($Declarado.Count) pacotes)"
    }
    exit 0
}
finally {
    Pop-Location
}
