#Requires -Version 7.0
<#
.SYNOPSIS
    Audita relatorios de tarefa e o ledger contra as formas de mentira que ja
    passaram por revisao neste projeto.

.DESCRIPTION
    Oito tarefas foram entregues como concluidas sem terem sido, e a auditoria
    que descobriu isso foi manual. Cada checagem abaixo corresponde a um caso
    real, nao a uma preocupacao hipotetica:

    1. HEDGE — `docs/OPERACAO.md` trouxe uma tabela de "Resultado da Medicao"
       com "ex: 408ms em teste local" e "tende a ficar ~30-45 MB". O primeiro
       e exemplo, o segundo e expectativa; nenhum e medicao. Alvo nao medido
       apresentado como resultado e ficcao com aparencia de tabela.

    2. MUTACAO NO CONDICIONAL — os relatorios das Tasks 30 e 31 escreveram
       "Se removermos X, o teste falha". Isso nao e prova. E a da Task 30
       estava factualmente errada: o reconciliador foi removido e a suite
       continuou verde, deixando um requisito P0 com cobertura zero.

    3. SECAO AUSENTE — o relatorio da Task 29 tinha 1.148 bytes, sem RED, sem
       GREEN, e respondia uma verificacao com "coberta implicitamente".

    4. SHA FANTASMA — o ledger registrou a Task 31 no commit 14210ee, que nao
       existe no repositorio. Um ledger que aponta para o vazio e pior que um
       ledger desatualizado, porque parece preciso.

    5. RELATORIO PROMETIDO E AUSENTE — ledger citando tarefa completa cujo
       relatorio nao esta no disco.

    O script NAO julga conteudo. Ele localiza as frases e os SHAs para uma
    pessoa conferir. Um achado pode ser legitimo — "nao medido" ao lado de uma
    linha de tabela e a resposta certa, e o script vai apontar a linha mesmo
    assim. O que ele impede e a frase passar sem ninguem olhar.

    CODIGO DE SAIDA:
      0 -> nada a conferir
      1 -> ha achados; a lista esta na saida
      2 -> o script nao conseguiu rodar

.EXAMPLE
    pwsh -File scripts/audit_reports.ps1
    pwsh -File scripts/audit_reports.ps1 -Task 33
#>
[CmdletBinding()]
param(
    # Audita so um numero de tarefa. Sem isto, audita todos os relatorios.
    [Parameter(Position = 0)]
    [string]$Task
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$ProjectRoot = Split-Path -Parent $PSScriptRoot
$SddRoot = Join-Path $ProjectRoot '.superpowers\sdd'

if (-not (Test-Path $SddRoot)) {
    Write-Output "[!] Diretorio de artefatos nao encontrado: $SddRoot"
    exit 2
}

$Findings = 0

# Palavras de conteudo de um texto curto. Verbos de commit e ruido de ledger
# saem: sem isso, "fix" e "watcher" fazem qualquer descricao casar com qualquer
# assunto, e a checagem de conferencia para de conferir.
function Get-PalavrasSignificativas {
    param([string]$Texto)
    $Parar = @('the','and','for','with','that','from','into','not','review','approved',
               'complete','completa','commits','commit','fix','feat','test','docs','chore',
               'when','what','its','are','was','has','apos','pass','passes')
    # 'index' e 'watcher' NAO entram na lista: sao os substantivos que carregam
    # o sinal neste projeto, e descarta-los fazia a Task 29 ser acusada por
    # descrever "index application" contra um commit sobre "integrating
    # fsnotify with index".
    return @(($Texto.ToLower() -replace '[^a-z0-9 ]', ' ') -split '\s+' |
        Where-Object { $_.Length -ge 4 -and $Parar -notcontains $_ })
}

function Add-Finding {
    param([string]$File, [int]$Line, [string]$Rule, [string]$Text)
    $script:Findings++
    $Rel = $File.Replace($ProjectRoot, '').TrimStart('\', '/')
    Write-Output ("  {0}:{1}: [{2}] {3}" -f $Rel, $Line, $Rule, $Text.Trim())
}

# ---------------------------------------------------------------------------
# 1 e 2: frases nos relatorios
# ---------------------------------------------------------------------------

# Hedge SO quando esta ao lado de algo que se apresenta como medicao. "e.g."
# em prosa explicativa e legitimo e aparece as dezenas nos relatorios bons; um
# checador que dispara nele vira ruido e para de ser lido, que e pior que nao
# existir. O alvo e a linha que a auditoria pegou de verdade — "ex: 408ms em
# teste local" e "tende a ficar ~30-45 MB" numa tabela chamada "Resultado da
# Medicao". Por isso o hedge so conta com um numero-com-unidade, uma linha de
# tabela, ou a palavra medicao/resultado/benchmark na mesma linha.
$HedgeWord = '(tende a|aproximadamente|e\.g\.|ex:|deveria ser|should be|estima-se|por volta de|cerca de|~\s*\d)'
$ResultShape = '(\d+\s*(ms|s|seg|min|MB|GB|KB|%)|^\s*\||medi(c|ç)[aã]o|medido|resultado|benchmark|tempo de)'
$HedgePattern = "(?i)(?=.*$HedgeWord)(?=.*$ResultShape)"

# "coberta implicitamente" e uma nao-resposta a uma verificacao, independente
# de haver numero na linha. Foi a resposta do relatorio da Task 29 para
# "os.Stat falhando com permissao negada deixa o indice intocado?".
$NonAnswerPattern = '(?i)(implicitamente|implicitly|coberto por consequ|covered by the|por consequencia|deve funcionar|presumo|assumo que)'

# Prova de mutacao escrita no condicional. O tempo verbal e o sinal: prova real
# esta no passado e traz saida colada.
$ConditionalPattern = '(?i)^\s*(se (n[oó]s )?(remov|alter|mud|apag|desativ|comment|troc)\w*|if we (remove|change|disable|comment)|caso (remov|alter)\w*)'

$Required = @(
    @{ Name = 'TDD/RED';     Pattern = '(?i)(^|\W)red(\W|$)' },
    @{ Name = 'TDD/GREEN';   Pattern = '(?i)(^|\W)green(\W|$)' },
    @{ Name = 'Mutacao';     Pattern = '(?i)muta' },
    @{ Name = 'Verificacao'; Pattern = '(?i)verifica' }
)

$Filter = if ($Task) { "task-$Task-report.md" } else { 'task-*-report.md' }
$Reports = @(Get-ChildItem -Path $SddRoot -Filter $Filter -Recurse -File | Sort-Object Name)

if ($Reports.Count -eq 0) {
    Write-Output "[!] Nenhum relatorio casou com '$Filter' em $SddRoot"
    exit 2
}

Write-Output "=== Relatorios ($($Reports.Count)) ==="
foreach ($R in $Reports) {
    $Lines = @(Get-Content -Path $R.FullName -Encoding utf8)
    $Body = $Lines -join "`n"

    for ($i = 0; $i -lt $Lines.Count; $i++) {
        $L = $Lines[$i]
        if ($L -match $HedgePattern) {
            Add-Finding -File $R.FullName -Line ($i + 1) -Rule 'HEDGE' -Text $L
        }
        if ($L -match $NonAnswerPattern) {
            Add-Finding -File $R.FullName -Line ($i + 1) -Rule 'NAO-RESPOSTA' -Text $L
        }
        if ($L -match $ConditionalPattern) {
            Add-Finding -File $R.FullName -Line ($i + 1) -Rule 'MUTACAO-CONDICIONAL' -Text $L
        }
    }

    foreach ($Sec in $Required) {
        if ($Body -notmatch $Sec.Pattern) {
            Add-Finding -File $R.FullName -Line 1 -Rule 'SECAO-AUSENTE' -Text "sem secao de $($Sec.Name)"
        }
    }

    # Um relatorio curto demais nao chega a conter evidencia. O piso vem do
    # menor relatorio deste projeto que continha RED, GREEN e mutacao com saida
    # colada; abaixo disso ninguem colou nada.
    if ($R.Length -lt 2000) {
        Add-Finding -File $R.FullName -Line 1 -Rule 'CURTO' -Text "$($R.Length) bytes — pequeno demais para conter saida de comando colada"
    }
}

# ---------------------------------------------------------------------------
# 4 e 5: ledger
# ---------------------------------------------------------------------------

Write-Output ""
Write-Output "=== Ledger ==="

# SHA pleno -> id da tarefa que o registrou primeiro. Sobrevive entre linhas e
# entre ledgers para que reuso seja detectavel.
$script:ShaDeTarefa = @{}

$Ledgers = @(Get-ChildItem -Path $SddRoot -Filter 'progress.md' -Recurse -File)
foreach ($Ledger in $Ledgers) {
    $Lines = @(Get-Content -Path $Ledger.FullName -Encoding utf8)

    for ($i = 0; $i -lt $Lines.Count; $i++) {
        $L = $Lines[$i]

        # SHAs curtos de 7 a 40 hex. O filtro de palavra evita casar com
        # numeros decimais e com hashes de outros contextos.
        foreach ($M in [regex]::Matches($L, '\b[0-9a-f]{7,40}\b')) {
            $Sha = $M.Value
            # Sequencias so de digitos sao numero, nao SHA.
            if ($Sha -match '^\d+$') { continue }
            & git -C $ProjectRoot cat-file -t $Sha *> $null
            if ($LASTEXITCODE -ne 0) {
                Add-Finding -File $Ledger.FullName -Line ($i + 1) -Rule 'SHA-FANTASMA' -Text "$Sha nao existe no repositorio"
            }
        }

        # SHA que EXISTE nao e SHA CERTO. O ledger ja teve a Task 31 apontando
        # para 14210ee, inexistente — e a "correcao" trocou por 2a59d0c, que
        # existe e e o commit da Task 30. A checagem de existencia passou verde
        # num ledger que mandava duas tarefas para o mesmo lugar, e o trabalho
        # real da 31 (d5d1bf0) deixou de ser referenciado por qualquer linha.
        # As tres checagens abaixo cobrem o que a existencia nao cobre.
        if ($L -match '(?i)^Task\s+([\d/]+)\s*:\s*(complete|completa)\s*\(commits?\s+([0-9a-f]{7,40})(?:\.\.([0-9a-f]{7,40}))?\s*,?\s*([^)]*)\)') {
            $TarefaID  = $Matches[1]
            $De        = $Matches[3]
            $Ate       = if ($Matches[4]) { $Matches[4] } else { $Matches[3] }
            $Descricao = $Matches[5]

            # (a) Intervalo vazio. "A..A" nao contem commit nenhum, e uma tarefa
            # cujo intervalo nao contem trabalho nao esta registrada, esta
            # mencionada.
            $NoIntervalo = @(& git -C $ProjectRoot rev-list "$De..$Ate" 2>$null)
            if ($null -eq $NoIntervalo) { $NoIntervalo = @() }
            if ($LASTEXITCODE -eq 0 -and @($NoIntervalo).Count -eq 0 -and $De -ne $Ate) {
                Add-Finding -File $Ledger.FullName -Line ($i + 1) -Rule 'INTERVALO-VAZIO' `
                    -Text "Task ${TarefaID}: $De..$Ate nao contem commit nenhum"
            }

            # (b) Duas tarefas terminando no MESMO commit. So o fim conta: o
            # inicio de uma tarefa e, por convencao, o fim da anterior — o
            # intervalo e exclusivo na ponta esquerda — entao comparar as duas
            # pontas acusaria toda sequencia contigua, que e justamente a
            # correta. Dois fins iguais e que significam que uma das linhas
            # aponta para o trabalho da outra.
            $FimPleno = (& git -C $ProjectRoot rev-parse $Ate 2>$null)
            if ($LASTEXITCODE -eq 0 -and $FimPleno) {
                if ($script:ShaDeTarefa.ContainsKey($FimPleno) -and
                    $script:ShaDeTarefa[$FimPleno] -ne $TarefaID) {
                    Add-Finding -File $Ledger.FullName -Line ($i + 1) -Rule 'SHA-REUSADO' `
                        -Text ("Task {0} termina em {1}, mesmo commit final da Task {2}" -f `
                            $TarefaID, $Ate, $script:ShaDeTarefa[$FimPleno])
                } else {
                    $script:ShaDeTarefa[$FimPleno] = $TarefaID
                }
            }

            # (c) A descricao da linha e o assunto do commit falam da mesma
            # coisa? Sem NENHUMA palavra significativa em comum, a linha
            # descreve uma tarefa e aponta para outra — que e exatamente o caso
            # da Task 31 hoje: "xxhash rename correlation" apontando para
            # "reconciliation recovery on fsnotify overflow event".
            # Compara com TODOS os assuntos do intervalo, nao so com a ponta: o
            # ultimo commit de uma tarefa e as vezes um gofmt ou um docstring, e
            # exigir que a ponta descreva a tarefa acusaria entregas corretas.
            $Assuntos = @(& git -C $ProjectRoot log --format='%s' "$De..$Ate" 2>$null)
            if (@($Assuntos).Count -eq 0) {
                $Assuntos = @(& git -C $ProjectRoot log -1 --format='%s' $Ate 2>$null)
            }

            # Linhas cuja descricao e so metadado de revisao ("review clean apos
            # 2 fix passes") nao descrevem trabalho e nao tem com o que casar.
            $A = @(Get-PalavrasSignificativas $Descricao)
            $B = @(Get-PalavrasSignificativas ($Assuntos -join ' '))
            if (@($A).Count -ge 3 -and @($B).Count -gt 0) {
                $Comuns = @($A | Where-Object { $B -contains $_ })
                if (@($Comuns).Count -eq 0) {
                    Add-Finding -File $Ledger.FullName -Line ($i + 1) -Rule 'SHA-NAO-CONFERE' `
                        -Text ("Task {0} descreve `"{1}`" mas {2}..{3} contem `"{4}`"" -f `
                            $TarefaID, $Descricao.Trim(), $De, $Ate, ($Assuntos -join ' | '))
                }
            }
        }

        # Tarefa dita completa precisa de relatorio no disco.
        if ($L -match '(?i)^Task\s+(\d+)\s*:\s*(complete|completa)') {
            $N = $Matches[1]
            $Found = Get-ChildItem -Path $SddRoot -Filter "task-$N-report.md" -Recurse -File -ErrorAction SilentlyContinue
            if (-not $Found) {
                Add-Finding -File $Ledger.FullName -Line ($i + 1) -Rule 'RELATORIO-AUSENTE' -Text "Task $N marcada completa e sem task-$N-report.md"
            }
        }
    }
}

# ---------------------------------------------------------------------------

Write-Output ""
if ($Findings -eq 0) {
    Write-Output "[OK] Nada a conferir."
    exit 0
}

Write-Output "[!] $Findings achado(s). Nenhum e automaticamente um defeito — cada um e"
Write-Output "    uma frase ou um SHA que precisa de uma pessoa confirmando."
Write-Output "[i] Onde nao houve medicao, 'nao medido' e a resposta certa e nao e sinalizada."
exit 1
