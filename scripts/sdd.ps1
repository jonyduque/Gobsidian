#Requires -Version 7.0
<#
.SYNOPSIS
    Atalhos para o fluxo de execucao tarefa a tarefa deste projeto.

.DESCRIPTION
    Envolve os dois scripts do plugin superpowers que o fluxo usa o tempo
    todo. Existe por dois motivos:

    O caminho deles embute a versao do plugin (hoje 6.1.1), entao toda
    invocacao literal quebra na proxima atualizacao — no meio de uma
    delegacao, e com mensagem que nao diz o que aconteceu. Aqui a versao e
    descoberta, e a falha diz o que procurar.

    E o `review-package` precisa da BASE certa: o commit ANTES de a tarefa
    comecar. Usar HEAD~1 descarta em silencio tudo menos o ultimo commit de
    uma tarefa com varios, e a revisao passa a olhar meio diff sem avisar.
    O subcomando `base` grava e le esse commit para voce.

.EXAMPLE
    .\scripts\sdd.ps1 base 19        # grava o HEAD atual como base da Task 19
    .\scripts\sdd.ps1 brief 19       # gera o brief autocontido da Task 19
    .\scripts\sdd.ps1 review 19      # empacota o diff desde a base gravada
    .\scripts\sdd.ps1 status         # o que o ledger diz, e onde o git esta
#>
[CmdletBinding()]
param(
    [Parameter(Mandatory, Position = 0)]
    [ValidateSet("brief", "review", "base", "status")]
    [string]$Command,

    [Parameter(Position = 1)]
    [string]$Task
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$ProjectRoot = Split-Path -Parent $PSScriptRoot
$Plan = Join-Path $ProjectRoot "docs\superpowers\plans\2026-07-25-gobsidian-v01.md"
$PlanName = [System.IO.Path]::GetFileNameWithoutExtension($Plan)

# O superpowers passou a escrever artefatos por plano na 6.2.0. O caminho
# plano antigo virou ponteiro depois que os dois ledgers derivaram — um com
# 16 tarefas, outro com 6.
$SddDir = Join-Path $ProjectRoot ".superpowers\sdd\$PlanName"
$Ledger = Join-Path $SddDir "progress.md"

function Get-SuperpowersScripts {
    $Cache = Join-Path $env:USERPROFILE ".claude\plugins\cache\claude-plugins-official\superpowers"
    if (-not (Test-Path $Cache)) {
        throw "Plugin superpowers nao encontrado em $Cache"
    }

    # A versao muda; pegue a mais recente em vez de fixar uma.
    $Version = Get-ChildItem $Cache -Directory |
        Sort-Object Name -Descending |
        Select-Object -First 1

    if (-not $Version) { throw "Nenhuma versao do superpowers em $Cache" }

    $Dir = Join-Path $Version.FullName "skills\subagent-driven-development\scripts"
    if (-not (Test-Path $Dir)) {
        throw "Scripts nao encontrados em $Dir (a estrutura do plugin mudou?)"
    }
    return $Dir
}

function Require-Task {
    if (-not $Task) { throw "Informe o numero da tarefa. Ex.: .\scripts\sdd.ps1 $Command 19" }
}

$BaseFile = Join-Path $SddDir "task-$Task-base.txt"

switch ($Command) {
    "brief" {
        Require-Task
        $Scripts = Get-SuperpowersScripts
        & bash (Join-Path $Scripts "task-brief") $Plan $Task
    }

    "base" {
        Require-Task
        New-Item -ItemType Directory -Force -Path $SddDir | Out-Null
        $Head = (git -C $ProjectRoot rev-parse HEAD).Trim()
        Set-Content -Path $BaseFile -Value $Head -Encoding ascii -NoNewline
        Write-Output "[OK] Base da Task ${Task}: $($Head.Substring(0,7))"
        Write-Output "[i] Rode isto ANTES de o executor comecar."
    }

    "review" {
        Require-Task
        if (-not (Test-Path $BaseFile)) {
            Write-Warning "[!] Base da Task $Task nao registrada."
            Write-Output "    Rode '.\scripts\sdd.ps1 base $Task' antes de a tarefa comecar."
            Write-Output "    Para esta, descubra o commit anterior a mao — nao use HEAD~1:"
            Write-Output "    ele descarta tudo menos o ultimo commit de uma tarefa com varios."
            exit 1
        }
        $Base = (Get-Content $BaseFile -Raw).Trim()
        $Scripts = Get-SuperpowersScripts
        # A assinatura ganhou PLAN_FILE na frente na 6.2.0. Passar na ordem
        # antiga falha com um "usage:" que nao diz que a versao mudou.
        & bash (Join-Path $Scripts "review-package") $Plan $Base "HEAD"
    }

    "status" {
        if (Test-Path $Ledger) {
            Write-Output "=== Ledger ==="
            Select-String -Path $Ledger -Pattern "^Task |^===" | ForEach-Object { $_.Line }
        }
        else {
            Write-Warning "[!] Ledger ausente em $Ledger"
        }
        Write-Output ""
        Write-Output "=== git ==="
        git -C $ProjectRoot log --oneline -3
        $Dirty = git -C $ProjectRoot status --porcelain
        if ($Dirty) {
            Write-Warning "[!] Arvore suja:"
            $Dirty | ForEach-Object { Write-Output "    $_" }
        }
        else {
            Write-Output "[OK] Arvore limpa"
        }
    }
}
