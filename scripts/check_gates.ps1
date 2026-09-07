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

# Roda o hook em modo simulado e devolve o objeto hookSpecificOutput inteiro
# (ou $null se o processo falhar). Decisao-Hook e Motivo-Hook leem um campo
# cada um a partir daqui -- refatorado na revisao final (F6) para nao duplicar
# a plumbing de -EncodedCommand entre as duas.
#
# Nao chama "pwsh -File $Hook -EmStage $EmStage" direto: -File repassa os
# argumentos como argv cru (medido nesta tarefa), entao um -EmStage com mais
# de um elemento vira um parametro posicional sobrando e o hook reprova com
# erro de bind -- nao com a decisao do caso. -EncodedCommand roda o mesmo
# script, mas o array chega como array de verdade porque quem le a linha e o
# parser do PowerShell, nao o passa-argumento do -File.
function Invocar-Hook {
    param([string]$Comando, [string[]]$EmStage)
    try {
        $comandoLiteral = "'" + ($Comando -replace "'", "''") + "'"
        $emStageLiteral = ($EmStage | ForEach-Object { "'" + ($_ -replace "'", "''") + "'" }) -join ','
        $hookLiteral = "'" + ($Hook -replace "'", "''") + "'"
        $script = "& $hookLiteral -Simular -Comando $comandoLiteral -EmStage @($emStageLiteral)"
        $encoded = [Convert]::ToBase64String([System.Text.Encoding]::Unicode.GetBytes($script))
        $json = & pwsh -NoProfile -EncodedCommand $encoded
        if ($LASTEXITCODE -ne 0) { return $null }
        return ($json | ConvertFrom-Json).hookSpecificOutput
    }
    catch {
        return $null
    }
}

function Decisao-Hook {
    param([string]$Comando, [string[]]$EmStage)
    $saida = Invocar-Hook $Comando $EmStage
    if ($null -eq $saida) { return 'erro' }
    return $saida.permissionDecision
}

# F6 da revisao final: hook quebrado responde 'allow' de proposito (catch-all
# documentado), e um caso que so olha a decisao nao distingue esse allow do
# allow legitimo da escotilha. Este le o MOTIVO.
function Motivo-Hook {
    param([string]$Comando, [string[]]$EmStage)
    $saida = Invocar-Hook $Comando $EmStage
    if ($null -eq $saida) { return 'erro' }
    return $saida.permissionDecisionReason
}

# Conta quantos SECAO-AUSENTE o auditor emite para um relatorio. F4 da revisao
# final: um exit 2 precoce (raiz ausente, ou nenhum relatorio casou) tambem
# emite zero ocorrencias de SECAO-AUSENTE, e um caso que so conta a string nao
# distingue esse zero vazio do zero legitimo de um relatorio completo.
function Secoes-Ausentes([string]$Task) {
    $saida = & pwsh -NoProfile -File $Audit -Task $Task -SddRoot $SddFalso 2>&1 | Out-String
    if ($LASTEXITCODE -eq 2) { return 'erro' }
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

    # F1 da revisao final: -m dentro de um comentario de shell nao e a mensagem.
    Caso -Nome '-m com escotilha dentro de comentario de shell, -F sem ela -> deny' `
        -Esperado 'deny' -Obtido (Decisao-Hook "git commit -F $msgSem # was: -m `"wip [sem-doc]`"" $go)

    # F2: dois commits na linha -- vale o ultimo, que nao tem escotilha.
    Caso -Nome 'dois git commit encadeados, escotilha so no primeiro -> deny' `
        -Esperado 'deny' -Obtido (Decisao-Hook 'git commit -m "docs: a [sem-doc]" && git commit -m "feat: b"' $go)

    Caso -Nome 'dois git commit encadeados, escotilha so no ultimo -> allow' `
        -Esperado 'allow' -Obtido (Decisao-Hook 'git commit -m "feat: a" && git commit -m "docs: b [sem-doc]"' $go)

    # F3: flags curtas agrupadas.
    Caso -Nome '-am com escotilha -> allow' `
        -Esperado 'allow' -Obtido (Decisao-Hook 'git commit -am "fix: x [sem-doc]"' $go)

    Caso -Nome 'escotilha dentro de aspas com # no texto -> allow' `
        -Esperado 'allow' -Obtido (Decisao-Hook 'git commit -m "fix: issue #12 [sem-doc]"' $go)

    # F6: o motivo do allow tem de ser a escotilha, nao o catch-all do hook.
    Caso -Nome 'motivo do allow e a escotilha, nao o catch do hook' `
        -Esperado 'escotilha [sem-doc] na mensagem do commit' -Obtido (Motivo-Hook 'git commit -m "fix: x [sem-doc]"' $go)

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

    # F5: secoes que so aparecem em comentario de shell dentro de cerca de
    # codigo nao contam -- a cerca inteira e removida antes de casar cabecalho.
    Caso -Nome 'secoes so em comentario de shell dentro de cerca -> 4 SECAO-AUSENTE' `
        -Esperado '4' -Obtido (Secoes-Ausentes '3')

    # F4: fixture inexistente e o auditor nem chega a varrer -- isso e erro,
    # nao "zero achados".
    Caso -Nome 'fixture inexistente -> erro, nao 0' `
        -Esperado 'erro' -Obtido (Secoes-Ausentes '9999')
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
