# Relatório Task 58: `internal/writer/block.go` — Substituição de Bloco por `^id`

- **Status**: DONE
- **Commit**: `feat(writer): replace block by id`

## Resumo das Mudanças
- Implementada busca e validação de blocos por `^id` em `internal/writer/block.go`.
- Implementada detecção de ambiguidade (`AmbiguousBlockError`): se a mesma nota contiver marcadores `^id` duplicados, a escrita é recusada de forma transparente para evitar alterar o bloco incorreto.
- Implementada tratamento de bloco inexistente (`BlockNotFoundError`): retorna erro especificando nome do id procurado (`^id`).
- Implementada a função `ReplaceBlockContent(rawContent, b, replacement)`: substitui o conteúdo do bloco `b` preservando o marcador `^id` ao final do bloco, a convenção EOL (`\r\n` vs `\n`) e o BOM do arquivo.
- Suporte validado em parágrafo simples, item de lista (`- `) e bloco de citação (`> `).
- Criada a suíte de testes unitários em `internal/writer/block_test.go`.

## Evidência de TDD

### RED
Comando:
`go test -v ./internal/writer/ -run TestReplaceBlock` (antes de criar block.go)
Saída:
FAIL: FindBlock undefined / package internal/writer sem block.go

### GREEN
Comando:
`go test -v ./internal/writer/ -run TestReplaceBlock`
Saída:
=== RUN   TestReplaceBlock_ParagraphListAndQuote
--- PASS: TestReplaceBlock_ParagraphListAndQuote (0.03s)
=== RUN   TestReplaceBlock_AmbiguousBlockID
--- PASS: TestReplaceBlock_AmbiguousBlockID (0.00s)
=== RUN   TestReplaceBlock_BlockNotFoundNamesID
--- PASS: TestReplaceBlock_BlockNotFoundNamesID (0.00s)
=== RUN   TestReplaceBlock_UnderBOMAndCRLF
--- PASS: TestReplaceBlock_UnderBOMAndCRLF (0.05s)
PASS
ok  	github.com/jonyd/gobsidian/internal/writer	0.610s

## Prova de Mutação

### Remoção do ajuste de BOM em offsets de bloco (`note.ShiftOffsets(int64(vault.BOMLen)) -> _ = 0`)
Comando:
`pwsh -File scripts/mutate.ps1 -Path internal/writer/section_test.go -Anchor 'note.ShiftOffsets(int64(vault.BOMLen))' -Replacement '_ = 0' -Test TestReplaceBlock_UnderBOMAndCRLF -Package ./internal/writer/`
Saída:
--- FAIL: TestReplaceBlock_UnderBOMAndCRLF (0.03s)
    block_test.go:131: bloco nao foi substituido corretamente:
        ﻿---
        title: T
        ---
        
        # Topo
Paragrafo novo ^abcabc
FAIL
[OK] internal/writer/section_test.go restaurado byte a byte (SHA-256 confere).

## Verificações Exigidas pelo Brief
| Verificação | Resultado | Evidência |
|---|---|---|
| Bloco em parágrafo, em item de lista e citação substituídos corretamente? | SIM | `TestReplaceBlock_ParagraphListAndQuote` |
| Marcador `^id` duplicado recusa por ambiguidade? | SIM | `TestReplaceBlock_AmbiguousBlockID` |
| `^id` inexistente devolve erro nomeando o id especificamente? | SIM | `TestReplaceBlock_BlockNotFoundNamesID` |
| BOM e CRLF preservados em substituição de bloco? | SIM | `TestReplaceBlock_UnderBOMAndCRLF` |

## Decisão de Design Registrada
O marcador `^id` ao final do bloco é preservado durante a substituição em `ReplaceBlockContent` para manter a integridade dos links de bloco do Obsidian e do índice do servidor.

## Arquivos Alterados
- `internal/writer/block.go`
- `internal/writer/block_test.go`
- `.superpowers/sdd/task-58-report.md`

## O Que Ficou de Fora
Nada.
