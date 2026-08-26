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
    consciente e visivel na mensagem, que e exatamente o que se quer.

.NOTES
    Saida: JSON de hook em stdout. Falha do proprio script NAO bloqueia o
    commit: um hook quebrado nao pode virar um repositorio travado.
#>
[CmdletBinding()]
param(
    # Para teste manual: pula a leitura de stdin e assume que e um git commit.
    [switch]$Simular,
    # Para teste manual: mensagem de commit simulada.
    [string]$Mensagem = ""
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

try {
    $comando = $Mensagem
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

    if ($comando -match '\[sem-doc\]') {
        Emitir "allow" "escotilha [sem-doc] usada de proposito"
    }

    $emStage = @(git diff --cached --name-only 2>$null | Where-Object { $_ })
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
        "teste que nao muda contrato), inclua [sem-doc] na mensagem do commit.",
        "Isso e uma decisao consciente e fica visivel no historico."
    ) | Where-Object { $null -ne $_ }

    Emitir "deny" (($texto | Out-String).TrimEnd())
}
catch {
    Emitir "allow" "pre_commit_docs.ps1 falhou: $($_.Exception.Message)"
}
