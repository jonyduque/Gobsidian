# Relatório Task 47: `internal/search/snippet.go` — trecho com destaque

- **Status**: DONE
- **Commit**: `feat(search): snippets with term highlight`

## Resumo das Mudanças
- Criado `internal/search/snippet.go` com recortador de trecho `GenerateSnippet`.
- Implementado suporte a janelas configuráveis (`maxChars` default 240, max 1000).
- Aplicado ajuste obrigatório de BOM (+3 bytes) para alinhar os offsets do parser/Inverted com o conteúdo do arquivo em disco em chamadas a `v.ReadRange`.
- Garantido que notas somente-nuvem (`CloudOnly`) retornem trecho vazio sem efetuar leitura em disco.
- Adicionada verificação de integridade UTF-8 para evitar recortar no meio de caracteres multibyte acentuados.
- Criada suíte de testes em `internal/search/snippet_test.go`.

## Evidência de TDD
### RED
Comando:
`go test -v ./internal/search/ -run TestSnippetOffsetsAlignWithDiskBytesUnderBOM`
Saída:
--- FAIL: TestSnippetOffsetsAlignWithDiskBytesUnderBOM (0.01s)
    snippet_test.go:58: destaque = "ao intercorre", quer "intercorrente"
FAIL

### GREEN
Comando:
`go test -v ./internal/search/...`
Saída:
=== RUN   TestSnippetOffsetsAlignWithDiskBytesUnderBOM
--- PASS: TestSnippetOffsetsAlignWithDiskBytesUnderBOM (0.03s)
=== RUN   TestSnippetOffsetsWithoutBOM
--- PASS: TestSnippetOffsetsWithoutBOM (0.02s)
=== RUN   TestSnippetMaxCharsRespected
--- PASS: TestSnippetMaxCharsRespected (0.03s)
=== RUN   TestSnippetTermAtEdges
--- PASS: TestSnippetTermAtEdges (0.02s)
=== RUN   TestSnippetAccentedWordUTF8Safety
--- PASS: TestSnippetAccentedWordUTF8Safety (0.02s)
PASS
ok  	github.com/jonyd/gobsidian/internal/search	3.147s

## Prova de Mutação (Ajuste de BOM)

Comando:
`pwsh -File scripts/mutate.ps1 -Path internal/search/snippet.go -Anchor 'bomOffset = int64(vault.BOMLen)' -Replacement 'bomOffset = 0' -Test TestSnippetOffsetsAlignWithDiskBytesUnderBOM -Package ./internal/search/`

Saída:
[...] Mutando internal/search/snippet.go
      - bomOffset = int64(vault.BOMLen)
      + bomOffset = 0

[...] go test -race -run TestSnippetOffsetsAlignWithDiskBytesUnderBOM ./internal/search/
----------------------------------------------------------------------
--- FAIL: TestSnippetOffsetsAlignWithDiskBytesUnderBOM (0.01s)
    snippet_test.go:58: destaque = "ao intercorre", quer "intercorrente"
FAIL
FAIL	github.com/jonyd/gobsidian/internal/search	0.794s
FAIL
----------------------------------------------------------------------
[OK] internal/search/snippet.go restaurado byte a byte (SHA-256 confere).
[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.

## Verificações Exigidas pelo Brief
| Verificação | Resultado | Evidência |
|---|---|---|
| Mesmos testes sem BOM continuam passando? | SIM | Testado por `TestSnippetOffsetsWithoutBOM` |
| `maxChars` é respeitado (default 240 / teto 1000)? | SIM | Testado por `TestSnippetMaxCharsRespected` |
| Termo no primeiro e último byte do arquivo? | SIM | Testado por `TestSnippetTermAtEdges` sem estouro de slice |
| Nota somente-nuvem? | SIM | Retorna `Snippet{}` imediato sem chamada a `ReadRange` |
| Recorte em palavra acentuada? | SIM | Testado por `TestSnippetAccentedWordUTF8Safety` e `utf8.ValidString` |

## Arquivos Alterados
- `internal/search/snippet.go`
- `internal/search/snippet_test.go`
- `.superpowers/sdd/task-47-report.md`

## git status --porcelain
```
?? internal/search/snippet.go
?? internal/search/snippet_test.go
?? .superpowers/sdd/task-47-report.md
```

## O Que Ficou de Fora
Nada.
