# Relatório de Execução — Task 106

## Status
`DONE`

## Commit
`13cb363` - `feat(service): add offset, total_size and next_offset to note_read (Task 106)`

## Evidência de TDD
RED (asserção falhando antes de implementar):
```
go test -v ./internal/service/... -run "TestReadNote"
--- FAIL: TestReadNoteTruncatedNaoMenteQuandoAFaixaMedeExatamenteMaxBytes (0.01s)
    read_test.go:308: Truncated=false com 511 de 512 bytes devolvidos
```
GREEN (todas as regras aplicadas e verificadas):
```
=== RUN   TestReadNoteTruncatedNaoMenteQuandoAFaixaMedeExatamenteMaxBytes
--- PASS: TestReadNoteTruncatedNaoMenteQuandoAFaixaMedeExatamenteMaxBytes (0.07s)
=== RUN   TestReadNoteOffsetPaginaDoInicioAoFim
--- PASS: TestReadNoteOffsetPaginaDoInicioAoFim (0.07s)
=== RUN   TestReadNoteOffsetMutualExclusion
--- PASS: TestReadNoteOffsetMutualExclusion (0.05s)
=== RUN   TestReadNoteOffsetOutOfBounds
--- PASS: TestReadNoteOffsetOutOfBounds (0.03s)
PASS
```

## Prova de Mutação (`scripts/mutate.ps1`)

1. Mutação 1 (`truncou = true` -> `truncou = false`):
```
pwsh -File scripts/mutate.ps1 -Path internal/service/read.go -Anchor 'truncou = true' -Replacement 'truncou = false' -Test TestReadNoteTruncatedNaoMenteQuandoAFaixaMedeExatamenteMaxBytes -Package ./internal/service/
--- FAIL: TestReadNoteTruncatedNaoMenteQuandoAFaixaMedeExatamenteMaxBytes (0.01s)
    read_test.go:308: Truncated=false com 511 de 512 bytes devolvidos
FAIL
FAIL	github.com/jonyd/gobsidian/internal/service	2.288s
[OK] internal/service/read.go restaurado byte a byte (SHA-256 confere).
[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
EXIT=0
```

2. Mutação 2 (`res.NextOffset = &end` -> `_ = end`):
```
pwsh -File scripts/mutate.ps1 -Path internal/service/read.go -Anchor 'res.NextOffset = &end' -Replacement '_ = end' -Test TestReadNoteOffsetPaginaDoInicioAoFim -Package ./internal/service/
--- FAIL: TestReadNoteOffsetPaginaDoInicioAoFim (0.05s)
    read_test.go:343: concatenacao dos pedacos difere do arquivo: 1000 bytes contra 24000
FAIL
FAIL	github.com/jonyd/gobsidian/internal/service	2.149s
[OK] internal/service/read.go restaurado byte a byte (SHA-256 confere).
[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
EXIT=0
```

## Verificações

- `pwsh -File scripts/verify.ps1`: 13 de 13 [OK].
- `golangci-lint version`: v2.12.2 built with go1.26.4.
- `note_read` em `test-vault/Bem-vindo.md` com `offset: 0` e `max_bytes: 50`:
```json
{
  "content": "Este é o seu novo *Cofre*.\n\nAnote algo, [[crie um",
  "hash": "4bc286598f4a0b8c",
  "truncated": true,
  "total_size": 195,
  "next_offset": 50
}
```

## Diff de `docs/TOOLS.md`

```diff
     "heading_level": { "type": "integer", "minimum": 1, "maximum": 6, "description": "Desambigua quando o mesmo texto aparece em níveis diferentes." },
     "block_id": { "type": "string", "description": "Identificador de bloco, sem o circunflexo." },
+    "offset":   { "type": "integer", "minimum": 0, "description": "Offset de byte a partir do início da nota (byte 0). Mutuamente exclusivo com heading e block_id. Ignora include_frontmatter." },
     "include_frontmatter": { "type": "boolean", "default": true },

-**Retorno com `path`.** `content`, `hash`, `truncated`, e — quando `heading` foi usado — `section` com nível, texto e faixa de offsets. Corresponde a `service.ReadResult`.
+**Retorno com `path`.** `content`, `hash`, `total_size`, `next_offset` (quando houver mais conteúdo), `truncated`, e — quando `heading` foi usado — `section` com nível, texto e faixa de offsets. Corresponde a `service.ReadResult`.

-**Retorno com `paths`.** `items`, uma lista na mesma ordem e do mesmo tamanho de `paths`. Cada item tem `path` e, ou os campos de sucesso (`content`, `hash`, `truncated`, `section`), ou `error` com `code` e `message` — uma nota que falha não derruba as demais e não desaparece da lista: o item aparece na posição de origem, com `error` preenchido. Corresponde a `service.ReadBatchResult`.
+**Retorno com `paths`.** `items`, uma lista na mesma ordem e do mesmo tamanho de `paths`. Cada item tem `path` e, ou os campos de sucesso (`content`, `hash`, `total_size`, `next_offset`, `truncated`, `section`), ou `error` com `code` e `message` — uma nota que falha não derruba as demais e não desaparece da lista: o item aparece na posição de origem, com `error` preenchido. Corresponde a `service.ReadBatchResult`.
```

## O que ficou de fora
Nenhum item do escopo da Task 106 ficou de fora.
