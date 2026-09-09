#Requires -Version 7.0
<#
.SYNOPSIS
    Prova que os conjuntos fechados citados em docs/PROMPT.md sao os que o
    codigo cobra.

.DESCRIPTION
    docs/PROMPT.md existe por causa de um buraco documentado em TOOLS.md: o
    host recebe so `type` e `description` de cada tool, entao NENHUM enum
    chega ao modelo. O servidor cobra os conjuntos assim mesmo, com
    INVALID_ARGUMENT. O prompt e o unico lugar de onde o modelo pode aprende-los.

    Isso faz do prompt uma SEGUNDA copia de um fato que ja mora em
    internal/service -- exatamente o que o CLAUDE.md proibe, porque "duas
    copias do mesmo fato divergem e a menos consultada e a que fica errada". E
    a menos consultada seria esta: ninguem abre o prompt ao acrescentar um
    valor a um ValidarEnum.

    A copia e inevitavel (o modelo precisa ler os valores em algum lugar); o
    silencio quando ela divergir, nao. Este script le os dois lados:

      - do codigo, toda chamada ValidarEnum("campo", v, padrao, "a", "b", ...)
        sob internal/service, incluindo as que quebram em varias linhas, e
        resolvendo a unica que passa a lista por variavel (CamposDeMetadata);
      - do prompt, cada linha "- tool.campo: a, b, c" do bloco de conjuntos
        fechados.

    Compara os CONJUNTOS DE VALORES, nao os nomes de campo: ha dois `sort`
    diferentes (note_list e tag_list), e casar por nome faria um cobrir o
    outro.

.NOTES
    Saida em ASCII puro. Sai 1 se um lado tiver conjunto que o outro nao tem.

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

$Prompt = Join-Path $Raiz 'docs/PROMPT.md'
$DirServico = Join-Path $Raiz 'internal/service'

foreach ($alvo in @($Prompt, $DirServico)) {
    if (-not (Test-Path $alvo)) {
        Write-Output "[!] ausente: $alvo"
        exit 1
    }
}

# Chave canonica de um conjunto: valores ordenados, minusculos, unidos por
# virgula. E o que permite comparar os dois lados sem depender do nome do campo
# nem da ordem em que foram escritos.
function Chave {
    param([string[]]$Valores)
    return (($Valores | ForEach-Object { $_.Trim().ToLowerInvariant() } | Where-Object { $_ } | Sort-Object -Unique) -join ',')
}

# ----------------------------------------------------------------------
# 1. O que o CODIGO cobra
# ----------------------------------------------------------------------
$NoCodigo = @{}

$FontesGo = @(Get-ChildItem -Path $DirServico -Filter '*.go' -File | Where-Object { $_.Name -notlike '*_test.go' })
$TextoGo = ($FontesGo | ForEach-Object { Get-Content -Path $_.FullName -Raw -Encoding UTF8 }) -join "`n"

# A lista literal pode continuar na linha seguinte (PatchNote faz isso), entao
# a captura vai ate o parenteses que fecha, com (?s).
foreach ($m in [regex]::Matches($TextoGo, '(?s)ValidarEnum\(\s*"(?<campo>[a-z_]+)"\s*,(?<resto>.*?)\)')) {
    $resto = $m.Groups['resto'].Value
    if ($resto -match 'CamposDeMetadata') {
        # A unica que passa a lista por variavel. Resolve na declaracao, em vez
        # de repetir os sete valores aqui -- repeti-los seria uma TERCEIRA
        # copia, criada pelo gate que existe para impedir a segunda.
        if ($TextoGo -match 'CamposDeMetadata\s*=\s*\[\]string\{(?<itens>[^}]*)\}') {
            $vals = [regex]::Matches($Matches['itens'], '"([^"]+)"') | ForEach-Object { $_.Groups[1].Value }
            $NoCodigo[(Chave $vals)] = "$($m.Groups['campo'].Value) (CamposDeMetadata)"
        }
        continue
    }
    # Os literais depois do padrao. O primeiro literal do resto e o PADRAO e
    # nao um valor aceito quando ele nao se repete na lista -- mas todos os
    # padroes deste codigo sao um dos aceitos, ou string vazia, entao tomar
    # todos os literais nao-vazios e correto e nao inventa valor.
    $vals = [regex]::Matches($resto, '"([^"]*)"') | ForEach-Object { $_.Groups[1].Value } | Where-Object { $_ }
    if ($vals.Count -lt 2) { continue }
    $NoCodigo[(Chave $vals)] = $m.Groups['campo'].Value
}

if ($NoCodigo.Count -eq 0) {
    Write-Output "[!] nao achei nenhuma chamada ValidarEnum em internal/service -- os conjuntos fechados sumiram?"
    exit 1
}

# ----------------------------------------------------------------------
# 2. O que o PROMPT afirma
# ----------------------------------------------------------------------
$NoPrompt = @{}

foreach ($linha in (Get-Content -Path $Prompt -Encoding UTF8)) {
    if ($linha -notmatch '^- (?<tool>[a-z_]+)\.(?<campo>[a-z_]+):\s*(?<vals>.+)$') { continue }
    $vals = $Matches['vals'] -split ',' | ForEach-Object { $_.Trim() }
    $NoPrompt[(Chave $vals)] = "$($Matches['tool']).$($Matches['campo'])"
}

if ($NoPrompt.Count -eq 0) {
    Write-Output "[!] docs/PROMPT.md nao lista nenhum conjunto fechado -- o bloco sumiu ou mudou de forma"
    exit 1
}

# ----------------------------------------------------------------------
# 3. Comparar
# ----------------------------------------------------------------------
$Problemas = @()

foreach ($k in ($NoCodigo.Keys | Sort-Object)) {
    if (-not $NoPrompt.ContainsKey($k)) {
        $Problemas += "o codigo cobra $($NoCodigo[$k]) = { $k } e o prompt nao ensina esse conjunto"
    }
}
foreach ($k in ($NoPrompt.Keys | Sort-Object)) {
    if (-not $NoCodigo.ContainsKey($k)) {
        $Problemas += "o prompt ensina $($NoPrompt[$k]) = { $k } e nenhum ValidarEnum cobra esse conjunto"
    }
}

if ($Problemas.Count -gt 0) {
    Write-Output "[!] docs/PROMPT.md e internal/service discordam:"
    $Problemas | ForEach-Object { Write-Output "     $_" }
    Write-Output ""
    Write-Output "     O host nao transmite enum nenhum (TOOLS.md, 'Schemas servidos'): o que"
    Write-Output "     estiver errado no prompt e o que o modelo vai tentar chamar."
    exit 1
}

if (-not $Silencioso) {
    Write-Output "[OK] os $($NoCodigo.Count) conjuntos fechados do prompt sao os que internal/service cobra"
}
exit 0
