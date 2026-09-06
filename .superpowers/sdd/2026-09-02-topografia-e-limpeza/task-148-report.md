# Task 148 — Relatório

## Status
DONE

## Commit
`1a22508` — `fix(service): batch note_read carries section_synthetic like the single read does`

## Evidência de TDD

**RED**

```
$ go test ./internal/service -run TestReadNotesLotePropagaSectionSynthetic
# github.com/jonyd/gobsidian/internal/service [github.com/jonyd/gobsidian/internal/service.test]
internal\service\lote_por_item_test.go:148:19: out.Items[0].SectionSynthetic undefined (type ReadNoteItem has no field or method SectionSynthetic)
internal\service\lote_por_item_test.go:151:18: out.Items[1].SectionSynthetic undefined (type ReadNoteItem has no field or method SectionSynthetic)
FAIL	github.com/jonyd/gobsidian/internal/service [build failed]
FAIL
```

**GREEN**

```
$ go test ./internal/service -run TestReadNotesLotePropagaSectionSynthetic -v
=== RUN   TestReadNotesLotePropagaSectionSynthetic
--- PASS: TestReadNotesLotePropagaSectionSynthetic (0.02s)
PASS
ok  	github.com/jonyd/gobsidian/internal/service	0.840s
```

## Prova de mutação

Duas cópias do campo, duas provas — uma por local onde `SectionSynthetic` é
copiado (o brief pedia as duas: `ReadNotes` e `MarshalJSON`).

**Mutação 1 — `ReadNotes` (linha `SectionSynthetic: res.SectionSynthetic,`)**

```
$ pwsh -File scripts/mutate.ps1 -Path internal/service/read.go `
  -Anchor 'SectionSynthetic: res.SectionSynthetic,' -Replacement '' `
  -Test TestReadNotesLotePropagaSectionSynthetic -Package ./internal/service/
```

```
[...] Mutando internal/service/read.go
      - SectionSynthetic: res.SectionSynthetic,
      +

[...] go test -race -run TestReadNotesLotePropagaSectionSynthetic ./internal/service/
----------------------------------------------------------------------
--- FAIL: TestReadNotesLotePropagaSectionSynthetic (0.01s)
    lote_por_item_test.go:149: conv.md: secao veio de candidato e o item do lote nao diz section_synthetic
    lote_por_item_test.go:160: MarshalJSON nao serializa o campo: {"path":"conv.md","content":"**13.1 Substituicao**\n\no texto da subsecao\n","hash":"70823967892670c3","section":{"level":2,"text":"13.1 Substituicao","slug":"131 substituicao","start":36,"end":79,"body_start":0},"total_size":79}
FAIL
FAIL	github.com/jonyd/gobsidian/internal/service	0.946s
FAIL
----------------------------------------------------------------------
[OK] internal/service/read.go restaurado byte a byte (SHA-256 confere).

[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
```
`EXIT=0`. Teste reprova nomeando `conv.md` na linha 149, exatamente a asserção
que a linha mutada existe para satisfazer. Restauro confirmado pelo próprio
script (SHA-256) e por `git diff --stat internal/service/read.go` limpo em
seguida (diff voltou a mostrar só a mudança real, não a mutação).

**Mutação 2 — `MarshalJSON` (linha `SectionSynthetic: i.SectionSynthetic,`)**

```
$ pwsh -File scripts/mutate.ps1 -Path internal/service/read.go `
  -Anchor 'SectionSynthetic: i.SectionSynthetic,' -Replacement '' `
  -Test TestReadNotesLotePropagaSectionSynthetic -Package ./internal/service/
```

```
[...] Mutando internal/service/read.go
      - SectionSynthetic: i.SectionSynthetic,
      +

[...] go test -race -run TestReadNotesLotePropagaSectionSynthetic ./internal/service/
----------------------------------------------------------------------
--- FAIL: TestReadNotesLotePropagaSectionSynthetic (0.01s)
    lote_por_item_test.go:160: MarshalJSON nao serializa o campo: {"path":"conv.md","content":"**13.1 Substituicao**\n\no texto da subsecao\n","hash":"70823967892670c3","section":{"level":2,"text":"13.1 Substituicao","slug":"131 substituicao","start":36,"end":79,"body_start":0},"total_size":79}
FAIL
FAIL	github.com/jonyd/gobsidian/internal/service	0.902s
FAIL
----------------------------------------------------------------------
[OK] internal/service/read.go restaurado byte a byte (SHA-256 confere).

[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
```
`EXIT=0`. Teste reprova na asserção de `MarshalJSON` (linha 160), a mutação
correta pega exatamente a regra que ela protege. Restauro confirmado.

Depois das duas mutações, `git diff --stat internal/service/read.go` mostrou
34 inserções / 30 deleções — exatamente a mudança real (o `gofmt` de
realinhamento de tags conta como troca de linha inteira), nada da mutação
sobrou.

## As verificações do brief

1. **`readNoteItemWire` e `ReadNoteItem` com a MESMA tag JSON; JSON idêntico
   entre leitura simples e em lote para a mesma nota.** Confirmado com um
   teste temporário (`TestZZZTempVerify148CompareSingleBatch`, escrito,
   rodado e apagado — não faz parte do commit):

   Simples:
   ```
   {"content":"**13.1 Substituicao**\n\no texto da subsecao\n","hash":"70823967892670c3","section":{"level":2,"text":"13.1 Substituicao","slug":"131 substituicao","start":36,"end":79,"body_start":0},"total_size":79,"section_synthetic":true}
   ```
   Em lote (mesmo item):
   ```
   {"path":"conv.md","content":"**13.1 Substituicao**\n\no texto da subsecao\n","hash":"70823967892670c3","section":{"level":2,"text":"13.1 Substituicao","slug":"131 substituicao","start":36,"end":79,"body_start":0},"section_synthetic":true,"total_size":79}
   ```
   `section_synthetic:true` presente nos dois, com o mesmo `section` e
   `hash`; a única diferença é o campo `path`, que só o item de lote carrega
   (esperado — identifica de qual entrada de `Alvos` o item veio).

2. **`docs/TOOLS.md` descreve `section_synthetic` no retorno em lote.**
   Editado o parágrafo "Retorno com `paths`" (linha 115) acrescentando
   `section_synthetic` à lista de campos de sucesso e a frase "`section_synthetic:
   true` acompanha toda seção vinda de candidato" — a mesma frase da leitura
   simples (linha 110).
   `pwsh -File scripts/check_doc_refs.ps1` → `EXIT=0`, "[OK] nenhum token
   entre crases parece citar artefato ausente do codigo."

3. **`go test ./internal/service -run 'ReadNote|Lote' -v` sem falha.**
   24 testes rodados, 23 PASS + 1 SKIP (`TestReadNoteCloudOnlyFails`, exige
   `FILE_ATTRIBUTE_RECALL_ON_DATA_ACCESS`, pré-existente e sem relação com
   esta mudança). Nenhuma falha.

## `verify.ps1`

```
[...] 1. go build       [OK]
[...] 2. go test -race  [OK]
[...] 3. go test (tetos de latencia, sem -race)  [OK]
[...] 4. go vet (windows)  [OK]
[...] 5. go vet (linux)    [OK]
[...] 6. go vet (darwin)   [OK]
[...] 7. gofmt             [OK]
[...] 8. golangci-lint          [OK]
[...] 9. golangci-lint (linux)  [OK]
[...] 10. check_net (RNF-30)      [OK]
[...] 11. check_tool_params       [OK]
[...] 12. check_doc_refs          [OK]
[...] 13. check_readme_anchors    [OK]

[OK] Bateria completa. Pode commitar.
```
13 etapas, todas `[OK]`. `GATE_EXIT=0`.

## O que ficou de fora

- Não usei `gopls`/MCP LSP para as referências, como o `implementador.md`
  pede: as ferramentas `mcp__gopls__*` não estavam carregáveis nesta sessão
  (`ToolSearch` não encontrou nenhuma). Usei `go build`, `go vet` e `grep`
  como substituto — suficiente aqui porque a mudança é aditiva (novo campo em
  três structs já existentes, sem renomear nada), mas registro a divergência
  em vez de afirmar que rodei o que não rodei.
- `.superpowers/sdd/2026-07-25-gobsidian-v01/progress.md` está modificado na
  árvore de trabalho (22 linhas adicionadas) desde antes desta tarefa
  começar — não fui eu que o toquei, e a instrução de despacho proíbe
  explicitamente mexer nele. Deixei fora do commit e fora do `git add`;
  reporto para o orquestrador decidir o que fazer com ele.
- Nenhum outro escopo foi cortado. As mudanças em `internal/service/read.go`,
  `internal/service/lote_por_item_test.go` e `docs/TOOLS.md` cobrem
  exatamente o que o brief descreveu nos Steps 1–5.

## `git status --porcelain` (após o commit)

```
 M .superpowers/sdd/2026-07-25-gobsidian-v01/progress.md
 M "test-vault/test vault/.obsidian/community-plugins.json"
 D "test-vault/test vault/.obsidian/plugins/parity-dumper/main.js"
 D "test-vault/test vault/.obsidian/plugins/parity-dumper/manifest.json"
 M "test-vault/test vault/.obsidian/workspace.json"
?? .claude/skills/troglodita-commit/
?? .claude/skills/troglodita-help/
?? .claude/skills/troglodita-review/
?? .claude/skills/troglodita/
?? .superpowers/sdd/2026-09-02-topografia-e-limpeza/review-40521d8..3dfcf8e.diff
?? .superpowers/sdd/2026-09-02-topografia-e-limpeza/task-147-base.txt
?? .superpowers/sdd/2026-09-02-topografia-e-limpeza/task-147-brief.md
?? .superpowers/sdd/2026-09-02-topografia-e-limpeza/task-147-report.md
?? .superpowers/sdd/2026-09-02-topografia-e-limpeza/task-148-base.txt
?? .superpowers/sdd/2026-09-02-topografia-e-limpeza/task-148-brief.md
?? Resume-Claude.ps1
?? "test-vault/test vault/.obsidian/community-plugins (conflito 01M08RSCY410E88M0516VB4K9R 2026-08-17 22h52).json"
?? "test-vault/test vault/.obsidian/core-plugins (conflito 01M08RSCY410E88M0516VB4K9R 2026-08-17 22h52).json"
?? "test-vault/test vault/.obsidian/plugins/gosync/"
?? "test-vault/test vault/A\303\247\303\243o.md"
?? "test-vault/test vault/Pasted image 20260814221454.png"
?? "test-vault/test vault/Sem t\303\255tulo.md"
?? "test-vault/test vault/main.js"
?? "test-vault/test vault/manifest.json"
```
Todos pré-existentes ao início desta tarefa (nenhum arquivo do usuário
tocado); `task-148-report.md` e `task-148-base.txt` são artefatos do próprio
despacho, não desta implementação.
