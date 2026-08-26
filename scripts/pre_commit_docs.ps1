#Requires -Version 7.0
<#
.SYNOPSIS
    Antes de um `git commit`, confere se a documentacao acompanhou o codigo.

.DESCRIPTION
    Roda como hook PreToolUse do Claude Code, filtrado para `Bash(git commit*)`
    e `PowerShell(git commit*)`. Le o JSON do hook em stdin, inspeciona o que
    esta EM STAGE e devolve uma decisao de permissao.

    Existe porque neste projeto a documentacao nao e enfeite -- ela e o contrato.
    Tres defeitos ja custaram caro exatamente por divergencia entre codigo e doc:

      - `note_list` declarava `fields` no schema e o descartava. Quem lia o
        schema nao tinha como saber que o pedido nao fazia nada.
      - `alias_collisions` era `Collisions: 0` literal, e aparecia na resposta.
      - O README declarou "v0.1 publicada" sem tag, sem release e sem o gate de
        orfaos ter rodado.

    A decisao e `ask`, nao `deny`. Gate que reprova sozinho e sem recurso ensina
    a contornar o gate -- e a licao de -MaxNaoMedidosPct no test_orphans.ps1.
    Aqui quem decide e a pessoa, com a lista do que provavelmente ficou para
    tras na tela.

.NOTES
    Saida: JSON de hook em stdout. Nunca escreve em stdout fora desse JSON.
    Falha do proprio script NAO bloqueia o commit (sai permitindo): um hook
    quebrado nao pode virar um repositorio travado.
#>
[CmdletBinding()]
param(
    # Para teste manual: pula a leitura de stdin e assume que e um git commit.
    [switch]$Simular
)

$ErrorActionPreference = "Stop"

function Emitir($decisao, $motivo) {
    $obj = @{
        hookSpecificOutput = @{
            hookEventName            = "PreToolUse"
            permissionDecision       = $decisao
            permissionDecisionReason = $motivo
        }
    }
    $obj | ConvertTo-Json -Depth 5 -Compress
    exit 0
}

try {
    if (-not $Simular) {
        $bruto = [Console]::In.ReadToEnd()
        if ([string]::IsNullOrWhiteSpace($bruto)) { Emitir "allow" "sem payload" }
        $entrada = $bruto | ConvertFrom-Json
        $comando = $entrada.tool_input.command
        if ($comando -notmatch 'git\s+commit') { Emitir "allow" "nao e git commit" }
        # `git commit --amend` de mensagem, e commits so de doc, nao interessam.
        if ($comando -match '--amend' -and $comando -notmatch '--no-edit') {
            Emitir "allow" "amend de mensagem"
        }
    }

    $emStage = @(git diff --cached --name-only 2>$null | Where-Object { $_ })
    if ($emStage.Count -eq 0) { Emitir "allow" "nada em stage" }

    # Codigo de producao: o que muda comportamento e contrato.
    $codigo = @($emStage | Where-Object {
            $_ -like "internal/*.go" -or $_ -like "cmd/*.go" -or
            $_ -like "internal/*/*.go" -or $_ -like "cmd/*/*.go"
        } | Where-Object { $_ -notlike "*_test.go" })

    if ($codigo.Count -eq 0) { Emitir "allow" "nenhum .go de producao em stage" }

    $docs = @($emStage | Where-Object {
            $_ -like "docs/*" -or $_ -eq "CLAUDE.md" -or $_ -eq "AGENTS.md" -or
            $_ -eq "README.md" -or $_ -like ".superpowers/sdd/*"
        })

    # Contrato de tool mexido exige TOOLS.md, mesmo que outra doc tenha entrado.
    $tools = @($codigo | Where-Object { $_ -like "*mcpsrv/tools_*" -or $_ -like "*internal/service/*" })
    $toolsDoc = $emStage -contains "docs/TOOLS.md"

    $pendencias = [System.Collections.Generic.List[string]]::new()
    if ($docs.Count -eq 0) {
        $pendencias.Add("nenhum arquivo de documentacao em stage")
    }
    if ($tools.Count -gt 0 -and -not $toolsDoc) {
        $pendencias.Add("mexeu em superficie de tool ($($tools -join ', ')) e docs/TOOLS.md NAO esta em stage")
    }
    $ledger = @($emStage | Where-Object { $_ -like ".superpowers/sdd/*progress.md" })
    if ($ledger.Count -eq 0) {
        $pendencias.Add("o ledger (progress.md) nao esta em stage -- a proxima sessao le ele, nao o seu contexto")
    }

    if ($pendencias.Count -eq 0) {
        Emitir "allow" "codigo e documentacao entraram juntos"
    }

    $texto = @(
        "Este commit muda codigo de producao e a documentacao pode ter ficado para tras.",
        "",
        "Arquivos .go de producao em stage ($($codigo.Count)):",
        ($codigo | Select-Object -First 8 | ForEach-Object { "  - $_" })
        if ($codigo.Count -gt 8) { "  ... e mais $($codigo.Count - 8)" }
        "",
        "Pendencias:",
        ($pendencias | ForEach-Object { "  [!] $_" }),
        "",
        "Antes de confirmar, pergunte-se:",
        "  - mudou contrato de tool?      -> docs/TOOLS.md",
        "  - mudou regra ou armadilha?    -> CLAUDE.md / docs/ARMADILHAS.md",
        "  - mudou camada ou decisao?     -> docs/ARCHITECTURE.md (AD-xx)",
        "  - mudou requisito ou medicao?  -> docs/OPERACAO.md (numero MEDIDO, nunca estimado)",
        "  - terminou uma tarefa?         -> o ledger em .superpowers/sdd/<marco>/progress.md",
        "",
        "Se a resposta for 'nada disso se aplica', aprove e siga."
    ) | Where-Object { $null -ne $_ } | ForEach-Object { $_ }

    Emitir "ask" (($texto | Out-String).TrimEnd())
}
catch {
    # Hook quebrado nao trava repositorio.
    Emitir "allow" "pre_commit_docs.ps1 falhou: $($_.Exception.Message)"
}
