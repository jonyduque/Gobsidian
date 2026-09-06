# Task 149 Report: `heading` + `block_id` juntos é `INVALID_ARGUMENT`, não `INTERNAL`

**Status:** DONE

**Commit:** `4a06a24` — fix(service): heading plus block_id in note_patch is the client's mistake, not the server's

---

## Evidência de TDD

### RED Phase — Teste falha

```bash
go test -v ./internal/service -run TestPatchNoteHeadingEBlockIDEhInvalidArgument
```

**Saída falhando (RED):**
```
=== RUN   TestPatchNoteHeadingEBlockIDEhInvalidArgument
    limites_enums_test.go:63: codigo = INTERNAL, queria INVALID_ARGUMENT: INTERNAL manda o host tentar de novo o mesmo pedido
        erro: heading e block_id sao mutuamente exclusivos em note_patch
--- FAIL: TestPatchNoteHeadingEBlockIDEhInvalidArgument (0.01s)
FAIL
FAIL	github.com/jonyd/gobsidian/internal/service	0.842s
FAIL
```

### GREEN Phase — Teste passa

Após alterar `internal/service/write.go:262` de `CodeInternal` para `CodeInvalidArgument`:

```bash
go test -v ./internal/service -run TestPatchNoteHeadingEBlockIDEhInvalidArgument
```

**Saída passando (GREEN):**
```
=== RUN   TestPatchNoteHeadingEBlockIDEhInvalidArgument
--- PASS: TestPatchNoteHeadingEBlockIDEhInvalidArgument (0.01s)
PASS
ok  	github.com/jonyd/gobsidian/internal/service	0.815s
```

---

## Prova de Mutação

**Comando:**
```bash
pwsh -File scripts/mutate.ps1 -Path internal/service/write.go `
  -Anchor 'CodeInvalidArgument, "heading e block_id' `
  -Replacement 'CodeInternal, "heading e block_id' `
  -Test TestPatchNoteHeadingEBlockIDEhInvalidArgument -Package ./internal/service/
```

**Saída da mutação:**
```
Carregado em 464ms
[...] Mutando internal/service/write.go
      - CodeInvalidArgument, "heading e block_id
      + CodeInternal, "heading e block_id

[...] go test -race -run TestPatchNoteHeadingEBlockIDEhInvalidArgument ./internal/service/
----------------------------------------------------------------------
--- FAIL: TestPatchNoteHeadingEBlockIDEhInvalidArgument (0.01s)
    limites_enums_test.go:63: codigo = INTERNAL, queria INVALID_ARGUMENT: INTERNAL manda o host tentar de novo o mesmo pedido
        erro: heading e block_id sao mutuamente exclusivos em note_patch
FAIL
FAIL	github.com/jonyd/gobsidian/internal/service	1.100s
FAIL
----------------------------------------------------------------------
[OK] internal/service/write.go restaurado byte a byte (SHA-256 confere).

[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
[i] Cole a saida acima no relatorio. Prova de mutacao no condicional
    ('se removermos X, o teste falha') nao conta: uma delas ja estava errada.
```

**Resultado:** EXIT=0 (teste reprovou sob mutação, regra está verificada)

---

## Verificações do Brief

### Verificação 1: Mensagem de erro permanece, apenas o código mudou
**Comando:** `grep -n "CodeInternal" internal/service/write.go`

**Resultado:** ✓ PASSOU
- Antes: linha 262 tinha `CodeInternal, "heading e block_id..."`
- Depois: linha 262 tem `CodeInvalidArgument, "heading e block_id..."`
- `grep` não encontra mais `CodeInternal` para o caso heading+block_id
- Mensagem de erro permanece idêntica: `"heading e block_id sao mutuamente exclusivos em note_patch"`

### Verificação 2: `docs/TOOLS.md` documenta o erro
**Resultado:** ✓ PASSOU
- Adicionada sentença ao primeiro parágrafo **Notas** de `## note_patch`:
  - Antes: `**Notas.** \`replace_section\` preserva a linha do heading e substitui apenas o conteúdo abaixo dela, incluindo subseções. \`replace_heading_and_section\` substitui também a linha do heading.`
  - Depois: `**Notas.** \`replace_section\` preserva a linha do heading e substitui apenas o conteúdo abaixo dela, incluindo subseções. \`replace_heading_and_section\` substitui também a linha do heading. \`heading\` e \`block_id\` juntos são \`INVALID_ARGUMENT\`.`

---

## Validações Adicionais

**Build:** ✓ PASSOU
```bash
go build ./cmd/gobsidian
```
Build successful

**Encoding:** ✓ PASSOU
```bash
python -c "open(r'C:\Users\jonyd\Projetos\Gobsidian\docs\TOOLS.md',encoding='utf-8').read()"
[OK] UTF-8 valido
```

---

## verify.ps1

**Saída completa do gate (13 verificações):**
```
Carregado em 504ms
[...] 1. go build
[OK] go build
[...] 2. go test -race
[OK] go test -race
[...] 3. go test (tetos de latencia, sem -race)
[OK] go test (tetos de latencia, sem -race)
[...] 4. go vet (windows)
[OK] go vet (windows)
[...] 5. go vet (linux)
[OK] go vet (linux)
[...] 6. go vet (darwin)
[OK] go vet (darwin)
[...] 7. gofmt
[OK] gofmt
[...] 8. golangci-lint
[OK] golangci-lint
[...] 9. golangci-lint (linux)
[OK] golangci-lint (linux)
[...] 10. check_net (RNF-30)
[OK] check_net (RNF-30)
[...] 11. check_tool_params
Carregamento: 1322,7 ms 

[OK] check_tool_params
[...] 12. check_doc_refs
[OK] check_doc_refs
[...] 13. check_readme_anchors
[OK] check_readme_anchors

[OK] Bateria completa. Pode commitar.

[exited with code 0]
```

**Resultado:** ✓ PASSOU — 13 verificações, todas verdes.

---

## O que ficou de fora

Nada. Todos os passos do brief foram executados:
- ✓ Step 1: Teste adicionado
- ✓ Step 2: Teste falha com `codigo = INTERNAL`
- ✓ Step 3: Código alterado em `write.go:262`
- ✓ Step 4: Teste passa, gate pronto
- ✓ Verificação 1: Mensagem de erro permanece, apenas código mudou
- ✓ Verificação 2: `docs/TOOLS.md` documenta o erro
- ✓ Mutação prova corregida

---

## Git Status

**Após commit (as três arquivos foram commitadas com sucesso):**
```
M .superpowers/sdd/2026-07-25-gobsidian-v01/progress.md
 M test-vault/test vault/.obsidian/community-plugins.json
 D test-vault/test vault/.obsidian/plugins/parity-dumper/main.js
 D test-vault/test vault/.obsidian/plugins/parity-dumper/manifest.json
 M test-vault/test vault/.obsidian/workspace.json
?? .claude/skills/troglodita-commit/
?? .claude/skills/troglodita-help/
?? .claude/skills/troglodita-review/
?? .claude/skills/troglodita/
?? Resume-Claude.ps1
```

Confirmação:
- ✓ `internal/service/write.go` — commitada
- ✓ `internal/service/limites_enums_test.go` — commitada
- ✓ `docs/TOOLS.md` — commitada
- ✓ Nenhum arquivo do usuário foi tocado (test-vault/, .claude/skills/, Resume-Claude.ps1, progress.md permanecem como estavam)
- ✓ Commit SHA: `4a06a24`

---
