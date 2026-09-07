#Requires -Version 7.0
<#
.SYNOPSIS
    Antes de um `git commit`, confere se a documentacao acompanhou o codigo.

.DESCRIPTION
    Roda como hook PreToolUse do Claude Code, filtrado para `git commit*`. Le o
    JSON do hook em stdin, inspeciona o que esta EM STAGE e devolve uma decisao.

    Existe porque neste projeto a documentacao e contrato, nao enfeite. Tres
    defeitos ja custaram caro por divergencia entre codigo e doc:

      - `note_list` declarava `fields` no schema e o descartava. Quem lia o
        schema nao tinha como saber que o pedido nao fazia nada.
      - `alias_collisions` era `Collisions: 0` literal, e aparecia na resposta.
      - RNF-32 esteve publicado como "Atingido" enquanto metade dele — symlink
        de arquivo — nunca funcionou nem teve teste.

    QUEM ELE INTERROMPE, e por que isso mudou em 2026-08-26. A primeira versao
    devolvia `ask`, e o resultado era o hook PERGUNTANDO AO USUARIO sobre um
    commit que o modelo fez. Alvo errado: o usuario nao e quem esqueceu a
    documentacao, e uma pergunta a cada commit vira ruido que se aprende a
    aprovar sem ler — que e o mesmo modo de falha de um gate que reprova
    aleatoriamente.

    Agora ele devolve `deny`, e o motivo volta para o MODELO, que corrige e
    tenta de novo. Do ponto de vista do usuario e automatico: nenhum prompt.

    ESCOTILHA. `[sem-doc]` na mensagem do commit passa direto. Existe porque
    gate sem saida legitima ensina a contornar o gate — e porque ha commits em
    que documentacao de fato nao se aplica (revert, ajuste de formatacao,
    correcao de teste que nao muda contrato). Usar a escotilha e uma decisao
    consciente e visivel na mensagem, que e exatamente o que se quer. A
    escotilha vale na mensagem (-m ou arquivo de -F). Ate 2026-09-07 valia em
    qualquer lugar da linha de comando, inclusive num comentario de shell, e
    por isso nao valia nada.

.NOTES
    Saida: JSON de hook em stdout. Falha do proprio script NAO bloqueia o
    commit: um hook quebrado nao pode virar um repositorio travado.
#>
[CmdletBinding()]
param(
    # Para teste: pula a leitura de stdin. -Comando e a linha que o hook
    # receberia em tool_input.command; -EmStage substitui `git diff --cached`.
    [switch]$Simular,
    [string]$Comando = "",
    [string[]]$EmStage = @()
)

$ErrorActionPreference = "Stop"

function Emitir($decisao, $motivo, $aviso = $null) {
    $saida = @{
        hookSpecificOutput = @{
            hookEventName            = "PreToolUse"
            permissionDecision       = $decisao
            permissionDecisionReason = $motivo
        }
    }
    if ($aviso) { $saida.systemMessage = $aviso }
    $saida | ConvertTo-Json -Depth 5 -Compress
    exit 0
}

# Corta a linha de comando ate o SEGMENTO do ultimo `git commit` nela: comeca
# no ultimo `git commit` que aparecer e termina no primeiro `#`, `&&`, `;` ou
# `|` que estiver FORA de aspas (percorrido caractere a caractere, com estado
# de aspas simples/duplas -- um `#` dentro de aspas e texto de mensagem, nao
# comentario). Existe porque a revisao final da Task 187 mediu dois bypasses
# na mesma familia:
#   - um `#` de comentario de shell com `-m "... [sem-doc]"` DEPOIS dele, onde
#     a mensagem de verdade (via -F) nao tinha a escotilha;
#   - dois `git commit` encadeados com `&&`, onde a escotilha do primeiro
#     cobria o segundo, que e o commit que de fato acontece por ultimo.
# So o ultimo segmento e o que sera de fato commitado; qualquer coisa antes
# dele -- inclusive um commit anterior na mesma linha -- nao e a mensagem.
function Segmento-Commit([string]$linha) {
    $ms = [regex]::Matches($linha, 'git\s+commit')
    if ($ms.Count -eq 0) { return $linha }
    $inicio = $ms[$ms.Count - 1].Index
    $aspaS = $false
    $aspaD = $false
    $fim = $linha.Length
    for ($i = $inicio; $i -lt $linha.Length; $i++) {
        $c = $linha[$i]
        if ($c -eq "'" -and -not $aspaD) { $aspaS = -not $aspaS; continue }
        if ($c -eq '"' -and -not $aspaS) { $aspaD = -not $aspaD; continue }
        if ($aspaS -or $aspaD) { continue }
        if ($c -eq '#' -or $c -eq ';' -or $c -eq '|') { $fim = $i; break }
        if ($c -eq '&' -and ($i + 1) -lt $linha.Length -and $linha[$i + 1] -eq '&') { $fim = $i; break }
    }
    return $linha.Substring($inicio, $fim - $inicio)
}

# A escotilha vale na MENSAGEM, nunca na linha de comando. Ate 2026-09-07 o
# hook procurava [sem-doc] no texto do comando inteiro, e um comentario de
# shell (`git commit -F msg.txt # [sem-doc]`) o satisfazia com a mensagem
# dizendo outra coisa — foi assim que os commits de 2026-09-06 passaram, por
# instrucao do brief, sem que ninguem quisesse burlar nada. Gate que le a
# escotilha fora do lugar onde ela fica visivel no historico nao gateia.
#
# Le -m/--message= (aspas duplas, simples ou sem aspas, repetidos) e o
# arquivo de -F/--file=. Sem nenhum dos dois — commit que abriria editor — a
# mensagem e desconhecida e a escotilha nao vale.
#
# O grupo de -m aceita flags curtas agrupadas (`-am`, `-aem`): o lookbehind
# barra um `-` precedido de letra/digito/traco, que e o que evita casar o
# segundo `-` de `--amend` como se fosse um `-m` isolado (revisao final, F3).
function Extrair-Mensagem([string]$linha) {
    $linha = Segmento-Commit $linha
    $partes = [System.Collections.Generic.List[string]]::new()
    $reM = '(?:(?<![\w-])-[a-zA-Z]*m|--message)(?:=|\s+)(?:"([^"]*)"|''([^'']*)''|(\S+))'
    foreach ($m in [regex]::Matches($linha, $reM)) {
        $texto = @($m.Groups[1].Value, $m.Groups[2].Value, $m.Groups[3].Value) | Where-Object { $_ } | Select-Object -First 1
        if ($texto) { $partes.Add($texto) }
    }
    $reF = '(?:-F|--file)(?:=|\s+)(?:"([^"]*)"|''([^'']*)''|(\S+))'
    foreach ($m in [regex]::Matches($linha, $reF)) {
        $arquivo = @($m.Groups[1].Value, $m.Groups[2].Value, $m.Groups[3].Value) | Where-Object { $_ } | Select-Object -First 1
        if ($arquivo -and (Test-Path -LiteralPath $arquivo -PathType Leaf)) {
            $partes.Add((Get-Content -LiteralPath $arquivo -Raw -Encoding utf8))
        }
    }
    return ($partes -join "`n")
}

try {
    $comando = $Comando
    if (-not $Simular) {
        $bruto = [Console]::In.ReadToEnd()
        if ([string]::IsNullOrWhiteSpace($bruto)) { Emitir "allow" "sem payload" }
        $entrada = $bruto | ConvertFrom-Json
        $comando = $entrada.tool_input.command
        if ($comando -notmatch 'git\s+commit') { Emitir "allow" "nao e git commit" }
        if ($comando -match '--amend' -and $comando -notmatch '--no-edit') {
            Emitir "allow" "amend de mensagem"
        }
    }

    $mensagem = Extrair-Mensagem $comando
    if ($mensagem -match '\[sem-doc\]') {
        Emitir "allow" "escotilha [sem-doc] na mensagem do commit"
    }

    $emStage = if ($Simular) { @($EmStage) } else { @(git diff --cached --name-only 2>$null | Where-Object { $_ }) }
    if ($emStage.Count -eq 0) { Emitir "allow" "nada em stage" }

    $codigo = @($emStage | Where-Object {
            $_ -like "internal/*.go" -or $_ -like "cmd/*.go" -or
            $_ -like "internal/*/*.go" -or $_ -like "cmd/*/*.go"
        } | Where-Object { $_ -notlike "*_test.go" })

    if ($codigo.Count -eq 0) { Emitir "allow" "nenhum .go de producao em stage" }

    $docs = @($emStage | Where-Object {
            $_ -like "docs/*" -or $_ -eq "CLAUDE.md" -or $_ -eq "AGENTS.md" -or $_ -eq "README.md"
        })
    $tools = @($codigo | Where-Object { $_ -like "*mcpsrv/tools_*" })
    $toolsDoc = $emStage -contains "docs/TOOLS.md"
    $ledger = @($emStage | Where-Object { $_ -like "*progress.md" })

    # Bloqueantes: o codigo mudou e a documentacao que o descreve nao veio junto.
    $bloqueios = [System.Collections.Generic.List[string]]::new()
    if ($docs.Count -eq 0) {
        $bloqueios.Add("$($codigo.Count) arquivo(s) .go de producao em stage e NENHUM arquivo de documentacao")
    }
    if ($tools.Count -gt 0 -and -not $toolsDoc) {
        $bloqueios.Add("a superficie de tool mudou ($($tools -join ', ')) e docs/TOOLS.md NAO esta em stage -- schema e documentacao sao um contrato so")
    }

    if ($bloqueios.Count -eq 0) {
        # O ledger avisa, nunca bloqueia: nem todo commit fecha uma tarefa, e um
        # bloqueio aqui obrigaria a inventar linha de ledger a cada commit
        # intermediario -- que e pior que ledger ausente, porque vira ruido.
        if ($ledger.Count -eq 0) {
            Emitir "allow" "documentacao acompanhou o codigo" `
                "[i] Lembrete: o ledger (progress.md) nao esta neste commit. Se ele fecha uma tarefa, registre antes de dizer que acabou."
        }
        Emitir "allow" "codigo, documentacao e ledger entraram juntos"
    }

    $texto = @(
        "COMMIT BLOQUEADO: codigo de producao mudou e a documentacao correspondente nao entrou.",
        "",
        ($bloqueios | ForEach-Object { "  [!] $_" }),
        "",
        "Arquivos .go de producao em stage:",
        ($codigo | Select-Object -First 8 | ForEach-Object { "  - $_" }),
        "",
        "Antes de tentar de novo, decida qual se aplica:",
        "  - mudou contrato de tool?      -> docs/TOOLS.md",
        "  - mudou regra ou armadilha?    -> docs/ARMADILHAS.md",
        "  - mudou camada ou decisao?     -> docs/ARCHITECTURE.md (AD-xx)",
        "  - mudou requisito ou medicao?  -> docs/OPERACAO.md (numero MEDIDO, nunca estimado)",
        "  - fechou uma tarefa?           -> o ledger em .superpowers/sdd/<marco>/progress.md",
        "",
        "Se documentacao genuinamente NAO se aplica (revert, formatacao, ajuste de",
        "teste que nao muda contrato), inclua [sem-doc] na MENSAGEM do commit (-m ou",
        "arquivo de -F). Na linha de comando fora da mensagem ele nao vale.",
        "O hook so le -m/--message= e o arquivo de -F/--file=; -C, --fixup, heredoc e",
        "caminho sem aspas nao sao lidos e caem aqui.",
        "Isso e uma decisao consciente e fica visivel no historico."
    ) | Where-Object { $null -ne $_ }

    Emitir "deny" (($texto | Out-String).TrimEnd())
}
catch {
    Emitir "allow" "pre_commit_docs.ps1 falhou: $($_.Exception.Message)"
}
