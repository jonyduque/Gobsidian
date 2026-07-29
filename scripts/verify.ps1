#Requires -Version 7.0
<#
.SYNOPSIS
    Bateria de verificacao obrigatoria antes de qualquer commit.

.DESCRIPTION
    Roda, em ordem e parando no primeiro erro: build, testes com detector de
    corrida, vet nos tres alvos, gofmt, e a verificacao de RNF-30.

    Existe porque a lista de comandos espalhada pela documentacao convida a
    rodar tres dos cinco. Um comando so nao deixa escolher qual pular.

    O -race nao e opcional: a versao sem ele nao conta como suite verde neste
    projeto. Varios pacotes coordenam goroutines, e uma corrida so aparece la.

.PARAMETER SkipCross
    Pula o vet cruzado de linux e darwin. Use apenas em iteracao rapida sobre
    codigo que nao tem build tag; o gate completo roda os tres.

.PARAMETER SkipNet
    Pula a verificacao de rede. Use apenas quando nenhum import mudou.

.EXAMPLE
    .\scripts\verify.ps1
    .\scripts\verify.ps1 -SkipCross
#>
[CmdletBinding()]
param(
    [switch]$SkipCross,
    [switch]$SkipNet
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$ProjectRoot = Split-Path -Parent $PSScriptRoot
Push-Location $ProjectRoot

$Failed = @()
$StepNumber = 0

function Invoke-Step {
    param(
        [string]$Name,
        [scriptblock]$Body,
        [switch]$FailIfOutput
    )

    $script:StepNumber++
    Write-Output "[...] $script:StepNumber. $Name"

    $Output = & $Body 2>&1
    $ExitCode = $LASTEXITCODE

    # gofmt sai com zero e lista os arquivos mal formatados na saida. Sem
    # -FailIfOutput ele passaria despercebido.
    $HasOutput = $FailIfOutput -and $Output
    $Broke = ($ExitCode -ne 0) -or $HasOutput

    if ($Broke) {
        $script:Failed += $Name
        Write-Warning "[!] $Name"
        if ($Output) { $Output | ForEach-Object { Write-Output "     $_" } }
    }
    else {
        Write-Output "[OK] $Name"
    }
}

# Os alvos sao os NOSSOS pacotes, nao "./...". Plugins de skills despejam
# assets na raiz do modulo — hoje agent/, com .go de exemplo cujos imports nao
# resolvem — e "./..." varre isso junto: build, test, vet, gofmt e check_net
# reprovavam as SETE etapas por causa de codigo que nao e deste projeto.
#
# Restringir nao afrouxa o gate: internal/ e cmd/ sao o modulo inteiro que nos
# escrevemos, e o codigo que vazar para fora deles nao teria onde ser
# importado. Se um dia a raiz voltar a ter .go nosso, acrescente-o aqui.
$Alvos = @("./internal/...", "./cmd/...")

# gofmt nao entende padrao de pacote, so caminho de diretorio.
$DirsFmt = @("internal", "cmd", "scripts") | Where-Object { Test-Path $_ }

Invoke-Step "go build" { go build @Alvos }

Invoke-Step "go test -race" { go test -race @Alvos }

Invoke-Step "go vet (windows)" { go vet @Alvos }

if (-not $SkipCross) {
    Invoke-Step "go vet (linux)" {
        $env:GOOS = "linux"
        try { go vet @Alvos } finally { Remove-Item Env:\GOOS -ErrorAction SilentlyContinue }
    }
    Invoke-Step "go vet (darwin)" {
        $env:GOOS = "darwin"
        try { go vet @Alvos } finally { Remove-Item Env:\GOOS -ErrorAction SilentlyContinue }
    }
}
else {
    Write-Output "[i] vet cruzado pulado (-SkipCross)"
}

Invoke-Step "gofmt" { gofmt -l @DirsFmt } -FailIfOutput

if (-not $SkipNet) {
    Invoke-Step "check_net (RNF-30)" { & (Join-Path $PSScriptRoot "check_net.ps1") }
}
else {
    Write-Output "[i] check_net pulado (-SkipNet)"
}

Pop-Location

Write-Output ""
if ($Failed.Count -gt 0) {
    Write-Warning "[!] $($Failed.Count) etapa(s) reprovada(s):"
    $Failed | ForEach-Object { Write-Output "    $_" }
    exit 1
}

Write-Output "[OK] Bateria completa. Pode commitar."
exit 0
