#Requires -Version 7.0
<#
.SYNOPSIS
    Prova a invariante da partida: nada roda antes de os mecanismos de
    encerramento estarem armados.

.DESCRIPTION
    Em 2026-09-09 esta invariante existia so como frase num comentario, e foi
    quebrada pelo autor da frase. A trava de instalacao e o registro de
    presenca -- criar diretorio, abrir arquivo, pedir trava do kernel, gravar
    JSON, fsync -- foram parar no comeco de runServe, ANTES de boot.VigiarHost,
    que e onde lifecycle.New instala o tratador de sinal. Um sinal que
    chegasse nessa janela nao tinha tratador: o processo morria pela acao
    padrao, sem registrar "reason=".

    O CI mediu: o cenario `signal` do gate de orfaos reprovou com "2 de 100
    ciclos encerraram sem registrar reason=", nas duas rodadas, com o mesmo job
    verde no commit anterior (26bb00d). O harness manda o sinal ~50-150 ms
    depois de lancar o processo, e o I/O acrescentado cabia dentro disso num
    runner carregado. A correcao foi mover as duas para prepararProcesso,
    chamada depois de VigiarHost nos tres pontos de partida.

    Duas invariantes, uma para cada metade do defeito:

    A. QUEM CHAMA prepararProcesso JA ARMOU O ENCERRAMENTO. Em toda funcao que
       chama prepararProcesso, uma chamada a boot.VigiarHost ou a lifecycle.New
       vem antes, na mesma funcao. Sao os tres pontos de partida --
       serveEmProcesso, servePonteRemota e o daemon --, e um quarto que
       aparecesse sem isso teria a mesma janela.

    B. runServe NAO FAZ NADA ANTES DE servePonte. E o lugar exato onde o I/O
       foi parar, e a unica coisa que pode rodar la e o logger: qualquer outra
       linha esta, por construcao, antes de o tratador de sinal existir. Quem
       precisa de I/O na partida faz depois de VigiarHost.

    A invariante B recusa ate calculo puro de proposito. "So uma leitura de
    env" e "so uma resolucao de caminho" foi como o I/O entrou -- ninguem
    acrescenta uma linha achando que ela e cara. Precisando mesmo, o caminho e
    mover para prepararProcesso, ou acrescentar o caso aqui com a razao escrita.

.NOTES
    Saida em ASCII puro. Sai 1 se qualquer invariante quebrar.

    Le o texto, nao a AST: as duas invariantes sao sobre ORDEM de linhas dentro
    de um corpo de funcao, o gofmt e etapa do proprio verify.ps1 e garante que
    a chave que fecha uma funcao de topo esta na coluna 0.

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

$Dir = Join-Path $Raiz 'cmd/gobsidian'
if (-not (Test-Path $Dir)) {
    Write-Output "[!] diretorio ausente: $Dir"
    exit 1
}

# Corpos devolve, por arquivo, cada funcao de topo como
# { Nome; Arquivo; Linhas } -- ja sem comentario, e com o numero de linha
# original preservado para a mensagem.
function Corpos {
    param([string]$Caminho)
    $todas = Get-Content -Path $Caminho -Encoding UTF8
    $saida = @()
    $atual = $null
    for ($i = 0; $i -lt $todas.Count; $i++) {
        $l = $todas[$i]
        if ($null -eq $atual) {
            if ($l -match '^func\s+(?:\([^)]*\)\s*)?([A-Za-z0-9_]+)\s*\(') {
                $atual = [pscustomobject]@{ Nome = $Matches[1]; Arquivo = $Caminho; Linhas = @() }
            }
            continue
        }
        if ($l -match '^\}') {
            $saida += $atual
            $atual = $null
            continue
        }
        # Comentario de linha inteira sai; comentario no fim da linha tambem,
        # senao "// ... prepararProcesso ..." conta como chamada.
        $limpa = ($l -replace '//.*$', '')
        if ($limpa.Trim()) {
            $atual.Linhas += [pscustomobject]@{ N = $i + 1; Texto = $limpa }
        }
    }
    return $saida
}

$Problemas = @()
$Funcoes = @()
foreach ($arq in (Get-ChildItem -Path $Dir -Filter '*.go' -File)) {
    if ($arq.Name -like '*_test.go') { continue }
    $Funcoes += Corpos $arq.FullName
}

if ($Funcoes.Count -eq 0) {
    Write-Output "[!] nao achei nenhuma funcao em $Dir"
    exit 1
}

# ----------------------------------------------------------------------
# A. quem chama prepararProcesso ja armou o encerramento
# ----------------------------------------------------------------------
$Armadores = 'boot\.VigiarHost\(|lifecycle\.New\('
$Chamadores = 0

foreach ($f in $Funcoes) {
    if ($f.Nome -eq 'prepararProcesso') { continue }
    $chamada = $f.Linhas | Where-Object { $_.Texto -match 'prepararProcesso\(' } | Select-Object -First 1
    if (-not $chamada) { continue }
    $Chamadores++
    $armou = $f.Linhas | Where-Object { $_.N -lt $chamada.N -and $_.Texto -match $Armadores } | Select-Object -First 1
    if (-not $armou) {
        $rel = (Split-Path -Leaf $f.Arquivo)
        $Problemas += "${rel}:$($chamada.N): $($f.Nome) chama prepararProcesso sem ter armado o encerramento antes"
        $Problemas += "    boot.VigiarHost ou lifecycle.New tem de vir antes, na mesma funcao"
    }
}

if ($Chamadores -eq 0) {
    $Problemas += "ninguem chama prepararProcesso -- a trava de instalacao e a presenca sumiram da partida?"
}

# ----------------------------------------------------------------------
# B. runServe nao faz nada antes de servePonte
# ----------------------------------------------------------------------
$RunServe = $Funcoes | Where-Object { $_.Nome -eq 'runServe' } | Select-Object -First 1
if (-not $RunServe) {
    $Problemas += "nao achei runServe em cmd/gobsidian -- foi renomeada?"
}
else {
    $ponte = $RunServe.Linhas | Where-Object { $_.Texto -match 'servePonte\(' } | Select-Object -First 1
    if (-not $ponte) {
        $Problemas += "runServe nao chama servePonte -- a partida mudou de forma e esta invariante precisa ser reescrita"
    }
    else {
        # A unica linha permitida antes: a criacao do logger. Denylist
        # envelhece; o que interessa nao e QUAL chamada e cara, e que nenhuma
        # roda ali.
        foreach ($l in $RunServe.Linhas) {
            if ($l.N -ge $ponte.N) { continue }
            if ($l.Texto -match '^\s*log\s*:?=\s*slog\.') { continue }
            $Problemas += "serve.go:$($l.N): roda antes de servePonte, e o tratador de sinal ainda nao existe:"
            $Problemas += "        $($l.Texto.Trim())"
        }
    }
}

if ($Problemas.Count -gt 0) {
    Write-Output "[!] invariante da partida quebrada:"
    $Problemas | ForEach-Object { Write-Output "     $_" }
    Write-Output ""
    Write-Output "     NADA roda antes de os mecanismos de encerramento estarem armados."
    Write-Output "     Quem precisa de I/O na partida faz depois de VigiarHost, em"
    Write-Output "     prepararProcesso -- ver o comentario dela em cmd/gobsidian/serve.go."
    exit 1
}

if (-not $Silencioso) {
    Write-Output "[OK] partida: $Chamadores pontos armam o encerramento antes de prepararProcesso; runServe so cria o logger"
}
exit 0
