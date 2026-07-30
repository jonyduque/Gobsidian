#Requires -Version 7.0
<#
.SYNOPSIS
    Confere se os briefs gerados sao mesmo autocontidos, antes de despachar.

.DESCRIPTION
    O `task-brief` do plugin extrai de "### Task N" ate o proximo "###".
    TUDO que estiver sob o cabecalho do marco — "## M5", por exemplo — nao
    chega a brief nenhum.

    Isso ja aconteceu: o M5 foi escrito com quatro decisoes fechadas e seis
    regras compartilhadas no preambulo do marco, e os briefs sairam com 22 a 71
    linhas, o da Task 67 com ZERO das quatro decisoes e nenhuma das seis secoes
    com Regras de execucao ou Contrato de relatorio. Foi pego por leitura, um
    minuto antes do despacho. Este script existe para nao depender disso.

    O sintoma mais confiavel e o TAMANHO: um brief muito mais curto que os
    irmaos perdeu contexto que os outros tem. Por isso a checagem de tamanho e
    relativa a mediana do lote, e nao um piso fixo.

    CODIGO DE SAIDA:
      0 -> todos os briefs parecem autocontidos
      1 -> ha achados; a lista esta na saida
      2 -> o script nao conseguiu rodar

.EXAMPLE
    pwsh -File scripts/check_briefs.ps1 63 68
    pwsh -File scripts/check_briefs.ps1
#>
[CmdletBinding()]
param(
    # Primeiro e ultimo numero de tarefa do lote. Sem eles, confere todos os
    # briefs do diretorio do plano.
    [Parameter(Position = 0)] [int]$De = 0,
    [Parameter(Position = 1)] [int]$Ate = 0
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$ProjectRoot = Split-Path -Parent $PSScriptRoot
$Plan = Join-Path $ProjectRoot "docs\superpowers\plans\2026-07-25-gobsidian-v01.md"
$PlanName = [System.IO.Path]::GetFileNameWithoutExtension($Plan)
$SddDir = Join-Path $ProjectRoot ".superpowers\sdd\$PlanName"

if (-not (Test-Path $SddDir)) {
    Write-Output "[!] Diretorio de briefs nao encontrado: $SddDir"
    exit 2
}

$Briefs = @(Get-ChildItem -Path $SddDir -Filter 'task-*-brief.md' -File | ForEach-Object {
        if ($_.Name -match 'task-(\d+)-brief\.md') {
            $n = [int]$Matches[1]
            if ($De -eq 0 -or ($n -ge $De -and $n -le $Ate)) {
                [pscustomobject]@{ Num = $n; File = $_ }
            }
        }
    } | Sort-Object Num)

if ($Briefs.Count -eq 0) {
    Write-Output "[!] Nenhum brief no intervalo pedido em $SddDir"
    exit 2
}

# Secoes que todo brief precisa ter. Cada uma esta aqui porque a ausencia dela
# ja produziu uma entrega ruim neste projeto:
#   - sem "Verificacoes", o executor entrega o caminho feliz;
#   - sem "Regras de execucao", ele roda comando proibido ou pula o gate;
#   - sem "Contrato de relatorio", volta um resumo em vez de evidencia;
#   - sem "mutate.ps1", a prova de mutacao vira prosa no condicional.
$Exigidas = @(
    @{ Nome = 'Verificacoes';          Padrao = '(?i)####\s+Verifica' },
    @{ Nome = 'Regras de execucao';    Padrao = '(?i)####\s+Regras de execu' },
    @{ Nome = 'Contrato de relatorio'; Padrao = '(?i)####\s+Contrato de relat' },
    # Tarefa de fechamento de marco nao entrega codigo, entao nao tem o que
    # mutar. Ela DIZ isso no contrato de relatorio, e a frase e o que a
    # distingue de uma tarefa que simplesmente esqueceu a prova.
    @{ Nome = 'Comando de mutacao';    Padrao = '(mutate\.ps1|nao tem prova de muta|n[aã]o tem prova de muta)' },
    @{ Nome = 'Arquivos e commit';     Padrao = '(?m)^\*\*Files:\*\*' }
)

$Achados = 0
function Add-Achado {
    param([int]$Task, [string]$Regra, [string]$Texto)
    $script:Achados++
    Write-Output ("  task-{0,-3} [{1}] {2}" -f $Task, $Regra, $Texto)
}

$Linhas = @{}
foreach ($B in $Briefs) { $Linhas[$B.Num] = (Get-Content -Path $B.File.FullName -Encoding utf8).Count }

# Mediana do lote. Um brief bem abaixo dela perdeu contexto que os irmaos tem —
# e foi exatamente esse o sintoma no M5.
$Ordenadas = @($Linhas.Values | Sort-Object)
$Mediana = $Ordenadas[[int]($Ordenadas.Count / 2)]
$Piso = [int]($Mediana * 0.5)

Write-Output "=== Briefs ($($Briefs.Count)); mediana $Mediana linhas; piso relativo $Piso ==="

foreach ($B in $Briefs) {
    $Corpo = Get-Content -Path $B.File.FullName -Encoding utf8 -Raw

    foreach ($E in $Exigidas) {
        if ($Corpo -notmatch $E.Padrao) {
            Add-Achado -Task $B.Num -Regra 'SECAO-AUSENTE' -Texto "sem $($E.Nome)"
        }
    }

    if ($Linhas[$B.Num] -lt $Piso) {
        Add-Achado -Task $B.Num -Regra 'CURTO' -Texto (
            "$($Linhas[$B.Num]) linhas contra mediana $Mediana — provavelmente " +
            "perdeu o que ficou sob o cabecalho do marco")
    }

    # Marcador de rascunho que sobreviveu ate o brief.
    #
    # So TODO e FIXME. A primeira versao pegava tambem TBD, "preencher" e
    # "a definir", e o primeiro achado foi uma linha que NEGAVA ter
    # placeholders: 'Nenhum passo contem "TBD"'. Um checador que dispara na
    # frase que afirma o contrario do defeito vira ruido, e ruido treina a
    # ignorar a ferramenta.
    foreach ($m in [regex]::Matches($Corpo, '(?m)^(?!\s*(#|\||`)).*\b(TODO|FIXME)\b.*$')) {
        Add-Achado -Task $B.Num -Regra 'RASCUNHO' -Texto $m.Value.Trim()
    }
}

Write-Output ""
if ($Achados -eq 0) {
    Write-Output "[OK] Os $($Briefs.Count) briefs tem as secoes exigidas e nenhum destoa em tamanho."
    Write-Output "[i] Isto NAO garante que o conteudo esteja certo — garante que nada"
    Write-Output "    ficou preso sob o cabecalho do marco. Leia um brief inteiro antes de despachar."
    exit 0
}

Write-Output "[!] $Achados achado(s)."
Write-Output "[i] SECAO-AUSENTE e CURTO quase sempre significam a mesma coisa: o conteudo"
Write-Output "    esta no preambulo do marco, e o extrator so ve de '### Task N' ao proximo"
Write-Output "    '###'. Repita o que vincula a tarefa DENTRO da secao dela."
exit 1
