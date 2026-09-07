#Requires -Version 7.0
<#
.SYNOPSIS
    Prova que os gates recusam o bypass que ja passou por eles.

.DESCRIPTION
    Dois gates deste projeto casavam a PALAVRA e nao a evidencia, e foram
    contornados sem intencao de contornar (docs/ESTADO.md, dividas de
    2026-09-06):

      - pre_commit_docs.ps1 procurava [sem-doc] na LINHA DE COMANDO do git
        commit, e um comentario de shell (`# [sem-doc]`) o satisfazia com a
        mensagem dizendo outra coisa.
      - audit_reports.ps1 procurava as secoes obrigatorias do relatorio por
        palavra solta no corpo, e "nao ha ciclo RED/GREEN nem prova de
        mutacao" satisfazia tres delas.

    Cada caso aqui e um bypass conhecido com a decisao que o gate DEVE tomar.
    Roda como etapa do verify.ps1: gate que volta a aceitar o bypass reprova
    o gate inteiro.

.NOTES
    Saida em ASCII puro. Sai 1 se qualquer caso reprovar.
#>
[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$ProjectRoot = Split-Path -Parent $PSScriptRoot
$Fixtures = Join-Path $PSScriptRoot 'testdata\gates'
$Hook = Join-Path $PSScriptRoot 'pre_commit_docs.ps1'

$script:Total = 0
$script:Reprovados = 0

function Caso {
    param([string]$Nome, [string]$Esperado, [string]$Obtido)
    $script:Total++
    if ($Esperado -eq $Obtido) {
        Write-Output "[OK] $Nome"
    }
    else {
        $script:Reprovados++
        Write-Output "[!] ${Nome}: esperado '$Esperado', obtido '$Obtido'"
    }
}

# Decisao do hook em modo simulado. O hook imprime o JSON de hook em stdout;
# so a decisao interessa aqui.
#
# Nao chama "pwsh -File $Hook -EmStage $EmStage" direto: -File repassa os
# argumentos como argv cru (medido nesta tarefa), entao um -EmStage com mais
# de um elemento vira um parametro posicional sobrando e o hook reprova com
# erro de bind -- nao com a decisao do caso. -EncodedCommand roda o mesmo
# script, mas o array chega como array de verdade porque quem le a linha e o
# parser do PowerShell, nao o passa-argumento do -File.
function Decisao-Hook {
    param([string]$Comando, [string[]]$EmStage)
    try {
        $comandoLiteral = "'" + ($Comando -replace "'", "''") + "'"
        $emStageLiteral = ($EmStage | ForEach-Object { "'" + ($_ -replace "'", "''") + "'" }) -join ','
        $hookLiteral = "'" + ($Hook -replace "'", "''") + "'"
        $script = "& $hookLiteral -Simular -Comando $comandoLiteral -EmStage @($emStageLiteral)"
        $encoded = [Convert]::ToBase64String([System.Text.Encoding]::Unicode.GetBytes($script))
        $json = & pwsh -NoProfile -EncodedCommand $encoded
        if ($LASTEXITCODE -ne 0) { return 'erro' }
        return ($json | ConvertFrom-Json).hookSpecificOutput.permissionDecision
    }
    catch {
        return 'erro'
    }
}

# Conta quantos SECAO-AUSENTE o auditor emite para um relatorio.
function Secoes-Ausentes([string]$Task) {
    $saida = & pwsh -NoProfile -File $Audit -Task $Task -SddRoot $SddFalso 2>&1 | Out-String
    return ([regex]::Matches($saida, 'SECAO-AUSENTE')).Count.ToString()
}

Push-Location $ProjectRoot
try {
    Write-Output "=== pre_commit_docs.ps1 ==="
    $go = @('internal/index/resolve.go')
    $goEDoc = @('internal/index/resolve.go', 'docs/TOOLS.md')
    $msgSem = 'scripts/testdata/gates/msg-sem-escotilha.txt'
    $msgCom = 'scripts/testdata/gates/msg-com-sem-doc.txt'

    # O bypass real de 2026-09-06: a escotilha num comentario de shell, a
    # mensagem (via -F) sem ela.
    Caso -Nome 'escotilha em comentario de shell, mensagem sem ela -> deny' `
        -Esperado 'deny' -Obtido (Decisao-Hook "git commit -F $msgSem # [sem-doc]" $go)

    Caso -Nome 'escotilha no arquivo de -F -> allow' `
        -Esperado 'allow' -Obtido (Decisao-Hook "git commit -F $msgCom" $go)

    Caso -Nome 'escotilha em -m entre aspas duplas -> allow' `
        -Esperado 'allow' -Obtido (Decisao-Hook 'git commit -m "fix: x [sem-doc]"' $go)

    Caso -Nome 'escotilha em -m entre aspas simples -> allow' `
        -Esperado 'allow' -Obtido (Decisao-Hook "git commit -m 'fix: x [sem-doc]'" $go)

    Caso -Nome 'escotilha fora das aspas de -m -> deny' `
        -Esperado 'deny' -Obtido (Decisao-Hook 'git commit -m "fix: x" # [sem-doc]' $go)

    Caso -Nome '-m sem escotilha, .go sem doc -> deny' `
        -Esperado 'deny' -Obtido (Decisao-Hook 'git commit -m "fix: x"' $go)

    Caso -Nome '-m sem escotilha, .go com doc -> allow' `
        -Esperado 'allow' -Obtido (Decisao-Hook 'git commit -m "fix: x"' $goEDoc)

    Caso -Nome '-F arquivo inexistente, escotilha no comando -> deny' `
        -Esperado 'deny' -Obtido (Decisao-Hook 'git commit -F nao-existe.txt [sem-doc]' $go)

    Caso -Nome 'sem -m nem -F (editor), .go sem doc -> deny' `
        -Esperado 'deny' -Obtido (Decisao-Hook 'git commit' $go)

    Write-Output ""
    Write-Output "=== audit_reports.ps1 ==="
    $Audit = Join-Path $PSScriptRoot 'audit_reports.ps1'
    $SddFalso = Join-Path $Fixtures 'sdd-falso'

    # O bypass conhecido: prosa que NEGA ter RED/GREEN/mutacao satisfazia
    # tres das quatro palavras. Com cabecalho exigido, faltam as quatro.
    Caso -Nome 'prosa que nomeia as secoes sem te-las -> 4 SECAO-AUSENTE' `
        -Esperado '4' -Obtido (Secoes-Ausentes '1')

    Caso -Nome 'quatro secoes em cabecalho -> 0 SECAO-AUSENTE' `
        -Esperado '0' -Obtido (Secoes-Ausentes '2')
}
finally {
    Pop-Location
}

Write-Output ""
if ($script:Reprovados -gt 0) {
    Write-Output "[!] check_gates: $script:Reprovados de $script:Total casos reprovados"
    exit 1
}
Write-Output "[OK] check_gates: $script:Total casos"
exit 0
