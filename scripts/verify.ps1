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

    # Pula o golangci-lint. Para iteracao rapida; o gate roda tudo.
    [switch]$SkipLint,
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
#
# tools/ entrou depois da Task 74. Ate ela, tools/netcheck era um analisador que
# ninguem chamava; a partir dela, check_net.ps1 o compila e o roda como vettool,
# e o gate do RNF-30 passou a depender dele. Fora de $Alvos, o analisador de que
# o gate depende era o unico codigo Go do repositorio que o gate nao compilava,
# nao vetava, nao lintava e cujo teste nunca rodava.
$Alvos = @("./internal/...", "./cmd/...", "./tools/...")

# gofmt nao entende padrao de pacote, so caminho de diretorio.
$DirsFmt = @("internal", "cmd", "scripts", "tools") | Where-Object { Test-Path $_ }

Invoke-Step "go build" { go build @Alvos }

Invoke-Step "go test -race" { go test -race @Alvos }

# O teto do RNF-04 so e cobrado SEM -race: o detector multiplica a latencia por
# 2 a 6, e comparar esse numero com 100 ms nao diria nada. Como a etapa acima e
# a unica que roda testes, e ela roda com -race, o teto do RNF-04 nao era
# cobrado pelo gate em lugar nenhum — ficava por conta de quem, um dia,
# rodasse `go test` na mao. Quem rodava na mao rodava com a maquina carregada,
# e o resultado era ver o teto piscar fora do gate e nunca dentro.
#
# Sao ~15 s. O -count=1 e obrigatorio: cache de teste devolveria o verde da
# rodada anterior, e uma medicao em cache nao mediu nada.
#
# TestRNF04SnippetConcurrencyLimit200 entrou aqui depois da revisao da Task 72.
# Ele nasceu com a mesma guarda `!raceEnabled` dos demais testes de tempo, mas
# rodava SO na etapa 2, que usa -race — de modo que a assercao era compilada
# fora em todo lugar onde o teste rodava, e o teto de 60 ms nao era cobrado por
# ninguem. E o mesmo defeito que o commit 67016de fechou para o teto do RNF-04;
# todo teto novo entra nesta lista no mesmo commit em que nasce.
Invoke-Step "go test (RNF-04, sem -race)" {
    go test -count=1 -run "TestRNF04VaultSearchLatencyP95|TestRNF04SnippetConcurrencyLimit200" ./internal/service/
}

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

# O lint entra na bateria porque ficar de fora dela custou 22 achados. `go vet`
# NAO pega errcheck, e as Tasks 33-42 fecharam com 18 errcheck e 4 revive sem
# que nada no loop por tarefa reportasse — dez relatorios, nenhum mencionando
# ter rodado o linter. Enquanto era um comando separado, era um comando que
# alguem esquecia.
#
# A versao e conferida antes: o go.mod declara go 1.25.0, e um golangci-lint
# compilado com Go mais antigo recusa o config ANTES de analisar linha nenhuma,
# saindo com erro que nao diz que o problema e a versao. Sem a conferencia, um
# zero local nao diz nada sobre o CI, que fixa a v2.12.2.
if (-not $SkipLint) {
    Invoke-Step "golangci-lint" {
        $Ver = (golangci-lint version 2>&1) -join " "
        if ($Ver -notmatch "2\.12\.2") {
            throw "golangci-lint fora da versao fixada pelo CI (v2.12.2): $Ver"
        }
        golangci-lint run @Alvos
    }
}
else {
    Write-Output "[i] lint pulado (-SkipLint)"
}

if (-not $SkipNet) {
    Invoke-Step "check_net (RNF-30)" { & (Join-Path $PSScriptRoot "check_net.ps1") }
}
else {
    Write-Output "[i] check_net pulado (-SkipNet)"
}

Invoke-Step "check_tool_params" { & (Join-Path $PSScriptRoot "check_tool_params.ps1") }

Pop-Location

Write-Output ""
if ($Failed.Count -gt 0) {
    Write-Warning "[!] $($Failed.Count) etapa(s) reprovada(s):"
    $Failed | ForEach-Object { Write-Output "    $_" }
    exit 1
}

Write-Output "[OK] Bateria completa. Pode commitar."
exit 0
