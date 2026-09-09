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

# Quantos relatorios o auditor enxergou sem -Task: a linha `=== Relatorios (N) ===`.
function Relatorios-Vistos {
    $saida = & pwsh -NoProfile -File $Audit -SddRoot $SddFalso 2>&1 | Out-String
    if ($LASTEXITCODE -eq 2) { return 'erro' }
    if ($saida -match '=== Relatorios \((\d+)\) ===') { return $Matches[1] }
    return 'sem-cabecalho'
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

    # F7 da revisao final: --amend nao e mais allow incondicional. Com .go em
    # stage e sem doc, amend e commit igual; com nada em stage o caminho normal
    # ja responde allow.
    Caso -Nome '--amend -m sem escotilha, .go sem doc -> deny' `
        -Esperado 'deny' -Obtido (Decisao-Hook 'git commit --amend -m "fix: x"' $go)

    Caso -Nome '--amend -m com escotilha, .go sem doc -> allow' `
        -Esperado 'allow' -Obtido (Decisao-Hook 'git commit --amend -m "fix: x [sem-doc]"' $go)

    Caso -Nome '--amend --no-edit, nada em stage -> allow' `
        -Esperado 'nada em stage' -Obtido (Motivo-Hook 'git commit --amend --no-edit' @())

    # N1 da re-revisao final: \" e aspa literal, nao abre nem fecha aspas; o #
    # depois dela continua sendo comentario de shell.
    Caso -Nome 'aspa escapada antes do comentario, -F sem escotilha -> deny' `
        -Esperado 'deny' -Obtido (Decisao-Hook "git commit -F $msgSem \`" # was: -m `"wip [sem-doc]`"" $go)

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

    # N2 da re-revisao final: cerca aberta e nunca fechada nao pode esconder os
    # cabecalhos reais que vem depois dela.
    Caso -Nome 'cerca nao fechada antes das quatro secoes reais -> 0 SECAO-AUSENTE' `
        -Esperado '0' -Obtido (Secoes-Ausentes '4')

    # 2026-09-07: o filtro `task-*-report.md` deixou os dois final-fix-report.md
    # reais fora da auditoria, um deles sem secao nenhuma. Qualquer
    # `*-report.md` conta, e -Task aceita nome sem numero.
    Caso -Nome '-Task final-fix acha final-fix-report.md -> 0 SECAO-AUSENTE' `
        -Esperado '0' -Obtido (Secoes-Ausentes 'final-fix')

    Caso -Nome 'sem -Task, todos os cinco *-report.md da fixture sao vistos' `
        -Esperado '5' -Obtido (Relatorios-Vistos)

    # ------------------------------------------------------------------
    # netcheck: a SEGUNDA excecao da RNF-30 (decisao D-13, 2026-09-08)
    #
    # net/http passou a ser permitido em internal/selfupdate, para o
    # `gobsidian update`. Excecao sem gate vira porta escancarada: os tres
    # casos abaixo sao o que RECUSA, o que ACEITA, e o inverso -- a excecao
    # valendo so onde deve, e so para os hosts da lista.
    #
    # Roda o analisador DE VERDADE contra fixtures, e nao um grep: a regra
    # mora no analisador, e um gate que testa outra coisa nao testa a regra.
    # ------------------------------------------------------------------
    $VetTool = Join-Path ([System.IO.Path]::GetTempPath()) "netcheck_gate_$([guid]::NewGuid().ToString('N')).exe"
    go build -o $VetTool ./tools/netcheck/cmd/netcheck
    if ($LASTEXITCODE -ne 0) {
        Write-Output "[!] netcheck: falha ao compilar o vettool"
        $script:Reprovados++
        $script:Total++
    }
    else {
        try {
            # Devolve 'aceito' ou 'recusado' para uma fixture.
            #
            # Caminho RELATIVO ao modulo (./scripts/...), e nao absoluto: com
            # caminho absoluto o `go vet` resolve o pacote por diretorio e o
            # caminho de importacao que chega ao analisador nao termina no nome
            # do pacote -- a regra da excecao passa a nao reconhecer
            # internal/selfupdate e o caso "aceito" reprova por engano.
            function Netcheck-Fixture {
                param([string]$Dir)
                $rel = "./scripts/testdata/gates/netcheck/$Dir"
                if (-not (Test-Path (Join-Path $ProjectRoot $rel))) { return "fixture-ausente" }
                go vet "-vettool=$VetTool" $rel 2>&1 | Out-Null
                if ($LASTEXITCODE -eq 0) { return 'aceito' }
                return 'recusado'
            }

            Caso -Nome 'net/http em internal/selfupdate -> aceito' `
                -Esperado 'aceito' -Obtido (Netcheck-Fixture 'permitido/selfupdate')

            Caso -Nome 'net/http em qualquer outro pacote -> recusado' `
                -Esperado 'recusado' -Obtido (Netcheck-Fixture 'proibido/qualquer')

            Caso -Nome 'host fora da lista, dentro de selfupdate -> recusado' `
                -Esperado 'recusado' -Obtido (Netcheck-Fixture 'hostestranho/selfupdate')
        }
        finally {
            Remove-Item $VetTool -ErrorAction SilentlyContinue
        }
    }

    # ------------------------------------------------------------------
    # check_graph: o grafo do CLAUDE.md contra os imports de verdade
    #
    # O proprio CLAUDE.md avisa que "dizer 'conferido' nao e conferir, e as duas
    # versoes anteriores diziam". Em 2026-09-09 uma TERCEIRA disse -- o bloco
    # ganhou `selfupdate -> config` e `go list` dizia folha.
    #
    # Os mutantes saem do CLAUDE.md VIVO, e nao de fixtures copiadas: uma copia
    # do grafo teria de ser atualizada toda vez que o grafo real mudasse, e uma
    # fixture desatualizada reprova pelo motivo errado -- que e a mesma classe de
    # gate inutil que este arquivo existe para impedir.
    # ------------------------------------------------------------------
    $Claude = Join-Path $ProjectRoot 'CLAUDE.md'
    $GraphScript = Join-Path $PSScriptRoot 'check_graph.ps1'

    function Graph-Resultado {
        param([string]$Caminho)
        & $GraphScript -Arquivo $Caminho -Silencioso *> $null
        if ($LASTEXITCODE -eq 0) { return 'aceito' }
        return 'recusado'
    }

    # Escreve uma copia do CLAUDE.md com UMA troca, e falha alto se a troca nao
    # casou -- um mutante que nao mutou passaria no teste dizendo nada.
    function Graph-Mutante {
        param([string]$De, [string]$Para)
        $texto = Get-Content -Path $Claude -Raw -Encoding UTF8
        if ($texto -notmatch [regex]::Escape($De)) { return $null }
        $novo = $texto -replace [regex]::Escape($De), $Para
        $tmp = Join-Path ([IO.Path]::GetTempPath()) ("claude_grafo_" + [guid]::NewGuid().ToString('N') + ".md")
        [IO.File]::WriteAllText($tmp, $novo, (New-Object Text.UTF8Encoding($false)))
        return $tmp
    }

    Caso -Nome 'grafo do CLAUDE.md como esta -> aceito' `
        -Esperado 'aceito' -Obtido (Graph-Resultado $Claude)

    # A aresta que NAO existe: exatamente o erro cometido em 2026-09-09.
    $m1 = Graph-Mutante 'selfupdate -> (folha)' 'selfupdate -> config'
    if (-not $m1) { $m1 = Graph-Mutante ("selfupdate " + [char]0x2192 + " (folha)") ("selfupdate " + [char]0x2192 + " config") }
    if ($m1) {
        Caso -Nome 'aresta no documento que nao existe nos imports -> recusado' `
            -Esperado 'recusado' -Obtido (Graph-Resultado $m1)
        Remove-Item $m1 -ErrorAction SilentlyContinue
    }
    else {
        Caso -Nome 'aresta no documento que nao existe nos imports -> recusado' `
            -Esperado 'recusado' -Obtido 'mutante-nao-aplicou'
    }

    # A aresta OMITIDA: a forma do erro anterior, que chamava parser de folha.
    $m2 = Graph-Mutante 'parser   -> text' 'parser   -> (folha)'
    if (-not $m2) { $m2 = Graph-Mutante ("parser   " + [char]0x2192 + " text") ("parser   " + [char]0x2192 + " (folha)") }
    if ($m2) {
        Caso -Nome 'aresta real omitida do documento -> recusado' `
            -Esperado 'recusado' -Obtido (Graph-Resultado $m2)
        Remove-Item $m2 -ErrorAction SilentlyContinue
    }
    else {
        Caso -Nome 'aresta real omitida do documento -> recusado' `
            -Esperado 'recusado' -Obtido 'mutante-nao-aplicou'
    }

    # ------------------------------------------------------------------
    # check_pins: as versoes fixadas concordam entre si
    #
    # Os dois defeitos que este gate cobre aconteceram no mesmo dia: o pin do
    # golangci-lint ficou discordando de si mesmo (a substituicao literal pegou
    # a mensagem e nao a condicao, escrita com pontos escapados), e o gate do
    # release rodou numa toolchain diferente da que compila o binario -- contra
    # o que o comentario do proprio arquivo afirma.
    #
    # Os mutantes saem de copias dos arquivos VIVOS, pela mesma razao do
    # check_graph: fixture com copia de pin envelhece e reprova pelo motivo
    # errado.
    # ------------------------------------------------------------------
    $PinsScript = Join-Path $PSScriptRoot 'check_pins.ps1'

    function Pins-Resultado {
        param([string]$RaizAlvo)
        & $PinsScript -Raiz $RaizAlvo -Silencioso *> $null
        if ($LASTEXITCODE -eq 0) { return 'aceito' }
        return 'recusado'
    }

    # Copia os quatro arquivos que o check le, aplicando UMA troca.
    function Pins-Raiz {
        param([string]$Arquivo, [string]$De, [string]$Para)
        $tmp = Join-Path ([IO.Path]::GetTempPath()) ("pins_" + [guid]::NewGuid().ToString('N'))
        New-Item -ItemType Directory -Path (Join-Path $tmp 'scripts') -Force | Out-Null
        New-Item -ItemType Directory -Path (Join-Path $tmp '.github/workflows') -Force | Out-Null
        Copy-Item (Join-Path $ProjectRoot 'scripts/verify.ps1') (Join-Path $tmp 'scripts/verify.ps1')
        foreach ($y in @('ci.yml', 'gate.yml', 'release.yml')) {
            Copy-Item (Join-Path $ProjectRoot ".github/workflows/$y") (Join-Path $tmp ".github/workflows/$y")
        }
        $alvo = Join-Path $tmp $Arquivo
        $texto = Get-Content -Path $alvo -Raw -Encoding UTF8
        if ($texto -notmatch [regex]::Escape($De)) { return $null }
        [IO.File]::WriteAllText($alvo, ($texto -replace [regex]::Escape($De), $Para), (New-Object Text.UTF8Encoding($false)))
        return $tmp
    }

    Caso -Nome 'pins como estao -> aceito' `
        -Esperado 'aceito' -Obtido (Pins-Resultado $ProjectRoot)

    # O defeito real: a condicao escapada fica para tras da mensagem.
    $p1 = Pins-Raiz 'scripts/verify.ps1' '"2\.13\.2"' '"2\.12\.2"'
    if ($p1) {
        Caso -Nome 'pin do linter discordando entre condicao e mensagem -> recusado' `
            -Esperado 'recusado' -Obtido (Pins-Resultado $p1)
        Remove-Item $p1 -Recurse -Force -ErrorAction SilentlyContinue
    }
    else {
        Caso -Nome 'pin do linter discordando entre condicao e mensagem -> recusado' `
            -Esperado 'recusado' -Obtido 'mutante-nao-aplicou'
    }

    # A terceira invariante, de 2026-09-09: pin de Go ABAIXO da diretiva do
    # go.mod. Com GOTOOLCHAIN=local o job nao degrada -- ele para com "go.mod
    # requires go >= X". Foi o que derrubou onze dos catorze jobs do CI no dia
    # em que a diretiva subiu para 1.27.0, com ci.yml e bench.yml ainda em
    # '1.25'. O gate passou, porque ele fixa 1.27.1 -- e passar sozinho ao lado
    # de onze vermelhos e o pior sinal possivel.
    #
    # Pins-Raiz nao copia o go.mod, entao esta invariante precisa da raiz
    # inteira: e o unico caso que compara arquivo de workflow com o go.mod.
    function Pins-RaizComGoMod {
        param([string]$Arquivo, [string]$De, [string]$Para)
        $tmp = Join-Path ([IO.Path]::GetTempPath()) ("pinsgo_" + [guid]::NewGuid().ToString('N'))
        New-Item -ItemType Directory -Path (Join-Path $tmp 'scripts') -Force | Out-Null
        New-Item -ItemType Directory -Path (Join-Path $tmp '.github/workflows') -Force | Out-Null
        Copy-Item (Join-Path $ProjectRoot 'scripts/verify.ps1') (Join-Path $tmp 'scripts/verify.ps1')
        Copy-Item (Join-Path $ProjectRoot 'go.mod') (Join-Path $tmp 'go.mod')
        Copy-Item (Join-Path $ProjectRoot '.github/workflows/*.yml') (Join-Path $tmp '.github/workflows')
        $alvo = Join-Path $tmp $Arquivo
        $texto = Get-Content -Path $alvo -Raw -Encoding UTF8
        if ($texto -notmatch [regex]::Escape($De)) { return $null }
        # So a PRIMEIRA ocorrencia: trocar todas deixaria o arquivo coerente
        # consigo mesmo de novo, e o caso passaria sem ter mutado nada util --
        # o erro que a primeira versao do caso do release ja cometeu.
        $i = $texto.IndexOf($De)
        $novo = $texto.Substring(0, $i) + $Para + $texto.Substring($i + $De.Length)
        [IO.File]::WriteAllText($alvo, $novo, (New-Object Text.UTF8Encoding($false)))
        return $tmp
    }

    $p3 = Pins-RaizComGoMod '.github/workflows/ci.yml' "go-version: '1.27.1'" "go-version: '1.25'"
    if ($p3) {
        Caso -Nome 'pin de Go abaixo da diretiva do go.mod -> recusado' `
            -Esperado 'recusado' -Obtido (Pins-Resultado $p3)
        Remove-Item $p3 -Recurse -Force -ErrorAction SilentlyContinue
    }
    else {
        Caso -Nome 'pin de Go abaixo da diretiva do go.mod -> recusado' `
            -Esperado 'recusado' -Obtido 'mutante-nao-aplicou'
    }

    # O inverso: pin ACIMA da diretiva e legitimo, e recusa-lo tornaria
    # impossivel testar numa toolchain nova antes de subir a diretiva.
    $p4 = Pins-RaizComGoMod '.github/workflows/ci.yml' "go-version: '1.27.1'" "go-version: '1.28.0'"
    if ($p4) {
        Caso -Nome 'pin de Go acima da diretiva -> aceito' `
            -Esperado 'aceito' -Obtido (Pins-Resultado $p4)
        Remove-Item $p4 -Recurse -Force -ErrorAction SilentlyContinue
    }
    else {
        Caso -Nome 'pin de Go acima da diretiva -> aceito' `
            -Esperado 'aceito' -Obtido 'mutante-nao-aplicou'
    }

    # O outro: o gate do release deixa de cobrar a toolchain que compila.
    # DEZ espacos: a linha do job de build. A do gate tem seis. Sem essa
    # distincao o -replace trocaria as DUAS -- e elas voltariam a concordar,
    # com o mutante passando por nao ter mutado nada util. A primeira versao
    # deste caso fez exatamente isso.
    $p2 = Pins-Raiz '.github/workflows/release.yml' "          go-version: '1.27.1'" "          go-version: '1.25'"
    if ($p2) {
        Caso -Nome 'gate do release em toolchain diferente do build -> recusado' `
            -Esperado 'recusado' -Obtido (Pins-Resultado $p2)
        Remove-Item $p2 -Recurse -Force -ErrorAction SilentlyContinue
    }
    else {
        Caso -Nome 'gate do release em toolchain diferente do build -> recusado' `
            -Esperado 'recusado' -Obtido 'mutante-nao-aplicou'
    }

    # ------------------------------------------------------------------
    # check_test_isolation: variavel de ambiente nao isola teste
    #
    # O defeito real: os testes do log desviavam os.UserCacheDir com
    # t.Setenv("XDG_CACHE_HOME"). Verde nas tres plataformas, escrevendo no
    # cache real do usuario numa delas -- os.UserCacheDir IGNORA XDG_CACHE_HOME
    # e LOCALAPPDATA no macOS.
    #
    # O mutante sai do arquivo VIVO que teve o defeito, daemon_log_test.go, com
    # a linha que ele tinha e nao tem mais.
    # ------------------------------------------------------------------
    $IsolScript = Join-Path $PSScriptRoot 'check_test_isolation.ps1'

    function Isol-Resultado {
        param([string]$RaizAlvo)
        & $IsolScript -Raiz $RaizAlvo -Silencioso *> $null
        if ($LASTEXITCODE -eq 0) { return 'aceito' }
        return 'recusado'
    }

    # Copia UM _test.go vivo para uma raiz propria, com uma linha acrescentada
    # (ou nenhuma, para o caso inverso).
    function Isol-Raiz {
        param([string]$Acrescentar)
        $tmp = Join-Path ([IO.Path]::GetTempPath()) ("isol_" + [guid]::NewGuid().ToString('N'))
        New-Item -ItemType Directory -Path (Join-Path $tmp 'cmd/gobsidian') -Force | Out-Null
        $origem = Join-Path $ProjectRoot 'cmd/gobsidian/daemon_log_test.go'
        $alvo = Join-Path $tmp 'cmd/gobsidian/daemon_log_test.go'
        $texto = Get-Content -Path $origem -Raw -Encoding UTF8
        if ($Acrescentar) { $texto = $texto + "`n" + $Acrescentar + "`n" }
        [IO.File]::WriteAllText($alvo, $texto, (New-Object Text.UTF8Encoding($false)))
        return $tmp
    }

    Caso -Nome 'testes do repositorio como estao -> aceito' `
        -Esperado 'aceito' -Obtido (Isol-Resultado $ProjectRoot)

    $i1 = Isol-Raiz "`tt.Setenv(`"XDG_CACHE_HOME`", dir)"
    Caso -Nome 'teste desviando os.UserCacheDir por env -> recusado' `
        -Esperado 'recusado' -Obtido (Isol-Resultado $i1)
    Remove-Item $i1 -Recurse -Force -ErrorAction SilentlyContinue

    # O inverso, na MESMA raiz de um so arquivo: sem a linha, aceita. Sem este
    # caso, o anterior nao distingue "pegou a linha" de "reprova qualquer raiz".
    $i2 = Isol-Raiz ''
    Caso -Nome 'o mesmo arquivo sem a linha -> aceito' `
        -Esperado 'aceito' -Obtido (Isol-Resultado $i2)
    Remove-Item $i2 -Recurse -Force -ErrorAction SilentlyContinue

    # ------------------------------------------------------------------
    # check_partida: nada roda antes de o encerramento estar armado
    #
    # Os dois mutantes sao as duas metades do defeito de 2026-09-09, cada uma
    # na forma exata em que ele aconteceu: I/O no comeco de runServe, e um
    # ponto de partida que chama prepararProcesso sem ter armado nada.
    # ------------------------------------------------------------------
    $PartidaScript = Join-Path $PSScriptRoot 'check_partida.ps1'

    function Partida-Resultado {
        param([string]$RaizAlvo)
        & $PartidaScript -Raiz $RaizAlvo -Silencioso *> $null
        if ($LASTEXITCODE -eq 0) { return 'aceito' }
        return 'recusado'
    }

    function Partida-Raiz {
        param([string]$Arquivo, [string]$De, [string]$Para)
        $tmp = Join-Path ([IO.Path]::GetTempPath()) ("partida_" + [guid]::NewGuid().ToString('N'))
        New-Item -ItemType Directory -Path (Join-Path $tmp 'cmd/gobsidian') -Force | Out-Null
        Copy-Item (Join-Path $ProjectRoot 'cmd/gobsidian/*.go') (Join-Path $tmp 'cmd/gobsidian')
        $alvo = Join-Path $tmp "cmd/gobsidian/$Arquivo"
        $texto = Get-Content -Path $alvo -Raw -Encoding UTF8
        if ($texto -notmatch [regex]::Escape($De)) { return $null }
        [IO.File]::WriteAllText($alvo, ($texto -replace [regex]::Escape($De), $Para), (New-Object Text.UTF8Encoding($false)))
        return $tmp
    }

    Caso -Nome 'partida do repositorio como esta -> aceito' `
        -Esperado 'aceito' -Obtido (Partida-Resultado $ProjectRoot)

    $q1 = Partida-Raiz 'serve.go' `
        "`tcodigo := shutdownExitCode(servePonte" `
        "`tinstalar.LiberarPresenca()`n`tcodigo := shutdownExitCode(servePonte"
    if ($q1) {
        Caso -Nome 'I/O em runServe antes de servePonte -> recusado' `
            -Esperado 'recusado' -Obtido (Partida-Resultado $q1)
        Remove-Item $q1 -Recurse -Force -ErrorAction SilentlyContinue
    }
    else {
        Caso -Nome 'I/O em runServe antes de servePonte -> recusado' `
            -Esperado 'recusado' -Obtido 'mutante-nao-aplicou'
    }

    $q2 = Partida-Raiz 'ponte.go' `
        "`tctx, vig := boot.VigiarHost(parent, stdin, log)" `
        "`tvar ctx = parent; var vig *boot.Vigia"
    if ($q2) {
        Caso -Nome 'ponto de partida que nao armou o encerramento -> recusado' `
            -Esperado 'recusado' -Obtido (Partida-Resultado $q2)
        Remove-Item $q2 -Recurse -Force -ErrorAction SilentlyContinue
    }
    else {
        Caso -Nome 'ponto de partida que nao armou o encerramento -> recusado' `
            -Esperado 'recusado' -Obtido 'mutante-nao-aplicou'
    }

    # ------------------------------------------------------------------
    # check_prompt: o prompt do usuario e os enums que o codigo cobra
    #
    # docs/PROMPT.md e uma SEGUNDA copia de um fato que mora em
    # internal/service, e e a copia menos consultada: ninguem abre o prompt ao
    # acrescentar um valor a um ValidarEnum. A copia e inevitavel -- o host nao
    # transmite enum nenhum, e o modelo tem de ler os valores em algum lugar --,
    # o silencio quando ela divergir nao e.
    # ------------------------------------------------------------------
    $PromptScript = Join-Path $PSScriptRoot 'check_prompt.ps1'

    function Prompt-Resultado {
        param([string]$RaizAlvo)
        & $PromptScript -Raiz $RaizAlvo -Silencioso *> $null
        if ($LASTEXITCODE -eq 0) { return 'aceito' }
        return 'recusado'
    }

    function Prompt-Raiz {
        param([string]$Arquivo, [string]$De, [string]$Para)
        $tmp = Join-Path ([IO.Path]::GetTempPath()) ("prompt_" + [guid]::NewGuid().ToString('N'))
        New-Item -ItemType Directory -Path (Join-Path $tmp 'docs') -Force | Out-Null
        New-Item -ItemType Directory -Path (Join-Path $tmp 'internal/service') -Force | Out-Null
        Copy-Item (Join-Path $ProjectRoot 'docs/PROMPT.md') (Join-Path $tmp 'docs/PROMPT.md')
        Copy-Item (Join-Path $ProjectRoot 'internal/service/*.go') (Join-Path $tmp 'internal/service')
        $alvo = Join-Path $tmp $Arquivo
        $texto = Get-Content -Path $alvo -Raw -Encoding UTF8
        if ($texto -notmatch [regex]::Escape($De)) { return $null }
        [IO.File]::WriteAllText($alvo, ($texto -replace [regex]::Escape($De), $Para), (New-Object Text.UTF8Encoding($false)))
        return $tmp
    }

    Caso -Nome 'prompt como esta -> aceito' `
        -Esperado 'aceito' -Obtido (Prompt-Resultado $ProjectRoot)

    # O que o gate existe para pegar: o codigo ganha um valor e o prompt fica
    # para tras. O modelo continuaria chamando o conjunto antigo.
    $r1 = Prompt-Raiz 'internal/service/graph.go' `
        '"both", "outgoing", "incoming")' '"both", "outgoing", "incoming", "sideways")'
    if ($r1) {
        Caso -Nome 'valor novo no codigo que o prompt nao ensina -> recusado' `
            -Esperado 'recusado' -Obtido (Prompt-Resultado $r1)
        Remove-Item $r1 -Recurse -Force -ErrorAction SilentlyContinue
    }
    else {
        Caso -Nome 'valor novo no codigo que o prompt nao ensina -> recusado' `
            -Esperado 'recusado' -Obtido 'mutante-nao-aplicou'
    }

    # O inverso, e o que mata a implementacao ingenua: casar por NOME de campo.
    # Ha dois `sort`, o de note_list e o de tag_list, com conjuntos diferentes.
    # Trocar um pelo outro deixa os dois nomes presentes dos dois lados -- um
    # gate que casasse por nome aceitaria, e o prompt estaria ensinando ao
    # modelo os valores da tool errada.
    $r2 = Prompt-Raiz 'docs/PROMPT.md' `
        '- tag_list.sort: name, count' '- tag_list.sort: path, modified, size, title'
    if ($r2) {
        Caso -Nome 'conjunto de uma tool atribuido a outra -> recusado' `
            -Esperado 'recusado' -Obtido (Prompt-Resultado $r2)
        Remove-Item $r2 -Recurse -Force -ErrorAction SilentlyContinue
    }
    else {
        Caso -Nome 'conjunto de uma tool atribuido a outra -> recusado' `
            -Esperado 'recusado' -Obtido 'mutante-nao-aplicou'
    }

    # ------------------------------------------------------------------
    # check_unicode: chave derivada nao depende de tabela que a toolchain move
    #
    # O Go 1.27 subiu as tabelas Unicode da geracao 15 para a 17, com a v1.6.0
    # ja publicada nessa toolchain. config.VaultKey nomeia o cache E o socket, e
    # o analisador decide o que e termo no indice persistido. A primeira nao
    # pode se mover nunca; a segunda pode, desde que o cache saiba.
    # ------------------------------------------------------------------
    $UniScript = Join-Path $PSScriptRoot 'check_unicode.ps1'

    function Uni-Resultado {
        param([string]$RaizAlvo)
        & $UniScript -Raiz $RaizAlvo -Silencioso *> $null
        if ($LASTEXITCODE -eq 0) { return 'aceito' }
        return 'recusado'
    }

    function Uni-Raiz {
        param([string]$Arquivo, [string]$De, [string]$Para)
        $tmp = Join-Path ([IO.Path]::GetTempPath()) ("uni_" + [guid]::NewGuid().ToString('N'))
        New-Item -ItemType Directory -Path (Join-Path $tmp 'internal/config') -Force | Out-Null
        New-Item -ItemType Directory -Path (Join-Path $tmp 'internal/search') -Force | Out-Null
        Copy-Item (Join-Path $ProjectRoot 'internal/config/config.go') (Join-Path $tmp 'internal/config/config.go')
        Copy-Item (Join-Path $ProjectRoot 'internal/search/persist.go') (Join-Path $tmp 'internal/search/persist.go')
        $alvo = Join-Path $tmp $Arquivo
        $texto = Get-Content -Path $alvo -Raw -Encoding UTF8
        if ($texto -notmatch [regex]::Escape($De)) { return $null }
        [IO.File]::WriteAllText($alvo, ($texto -replace [regex]::Escape($De), $Para), (New-Object Text.UTF8Encoding($false)))
        return $tmp
    }

    Caso -Nome 'chaves derivadas como estao -> aceito' `
        -Esperado 'aceito' -Obtido (Uni-Resultado $ProjectRoot)

    # A regressao exata: a chave do cofre volta a consultar a tabela da stdlib.
    $u1 = Uni-Raiz 'internal/config/config.go' `
        'return cases.Lower(language.Und).String(s)' 'return strings.ToLower(s)'
    if ($u1) {
        Caso -Nome 'chave do cofre voltando a usar tabela da stdlib -> recusado' `
            -Esperado 'recusado' -Obtido (Uni-Resultado $u1)
        Remove-Item $u1 -Recurse -Force -ErrorAction SilentlyContinue
    }
    else {
        Caso -Nome 'chave do cofre voltando a usar tabela da stdlib -> recusado' `
            -Esperado 'recusado' -Obtido 'mutante-nao-aplicou'
    }

    # O inverso, e o outro lado da regra: o analisador PODE usar as tabelas,
    # desde que o cache carregue a geracao. Sem o carimbo, cache de uma geracao
    # e lido como valido por outra -- e o gate tem de pegar a ausencia do
    # carimbo, nao a presenca da chamada.
    $u2 = Uni-Raiz 'internal/search/persist.go' `
        'var CacheAnalyzerVersion = versaoManualDoAnalisador*1000 + text.VersaoDasTabelas()' `
        'var CacheAnalyzerVersion = versaoManualDoAnalisador'
    if ($u2) {
        Caso -Nome 'versao do analisador sem a geracao Unicode -> recusado' `
            -Esperado 'recusado' -Obtido (Uni-Resultado $u2)
        Remove-Item $u2 -Recurse -Force -ErrorAction SilentlyContinue
    }
    else {
        Caso -Nome 'versao do analisador sem a geracao Unicode -> recusado' `
            -Esperado 'recusado' -Obtido 'mutante-nao-aplicou'
    }
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
