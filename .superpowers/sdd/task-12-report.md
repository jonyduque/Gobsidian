# Task 12 report

## Fix pass — review findings

Scope: `internal/parser/frontmatter.go`, `frontmatter_test.go`, `slug_test.go`, `types.go`. All snippets transcribed verbatim from `docs/superpowers/plans/2026-07-25-gobsidian-v01.md`, section "Task 12: Tipos do parser, frontmatter e slug".

### Changes made

1. **Finding 1** — rewrote `SplitFrontmatter`'s doc comment in `frontmatter.go` to state the offset is relative to the slice received (not the file), and to spell out how to map it to a file position when a BOM was stripped (add `len(bom)` when `vault.StripBOM` reported `true`). Function body unchanged.
2. **Finding 2** — added `CRLF` and `CRLF com frontmatter vazio` rows to `TestSplitFrontmatter` in `frontmatter_test.go`.
3. **Finding 3** — added `so frontmatter, sem corpo`, `sem newline final`, and `delimitador de abertura no fim do arquivo` rows to the same table.
4. **Finding 4** — added the `{"!!!", ""}` row to `TestSlug` in `slug_test.go`.
5. **Finding 5** — expanded the `Heading.Start` doc comment in `types.go` to state the offset origin (same buffer `SplitFrontmatter` received, with `bodyOffset` already added by `Parse`) and note `Block.Start`/`Link.Start` follow the same origin.

Findings left open per instructions (Task 25 parity checklist, not touched): closing delimiter with trailing space (`--- `), duplicate frontmatter key behaviour vs. Obsidian.

### Verification commands and output

**1. `go test -race ./internal/parser/ -v`**

```
=== RUN   TestSplitFrontmatter
=== RUN   TestSplitFrontmatter/presente
=== RUN   TestSplitFrontmatter/ausente
=== RUN   TestSplitFrontmatter/tres_tracos_no_meio_nao_conta
=== RUN   TestSplitFrontmatter/delimitador_nao_fechado
=== RUN   TestSplitFrontmatter/frontmatter_vazio
=== RUN   TestSplitFrontmatter/CRLF
=== RUN   TestSplitFrontmatter/CRLF_com_frontmatter_vazio
=== RUN   TestSplitFrontmatter/so_frontmatter,_sem_corpo
=== RUN   TestSplitFrontmatter/sem_newline_final
=== RUN   TestSplitFrontmatter/delimitador_de_abertura_no_fim_do_arquivo
--- PASS: TestSplitFrontmatter (0.00s)
    --- PASS: TestSplitFrontmatter/presente (0.00s)
    --- PASS: TestSplitFrontmatter/ausente (0.00s)
    --- PASS: TestSplitFrontmatter/tres_tracos_no_meio_nao_conta (0.00s)
    --- PASS: TestSplitFrontmatter/delimitador_nao_fechado (0.00s)
    --- PASS: TestSplitFrontmatter/frontmatter_vazio (0.00s)
    --- PASS: TestSplitFrontmatter/CRLF (0.00s)
    --- PASS: TestSplitFrontmatter/CRLF_com_frontmatter_vazio (0.00s)
    --- PASS: TestSplitFrontmatter/so_frontmatter,_sem_corpo (0.00s)
    --- PASS: TestSplitFrontmatter/sem_newline_final (0.00s)
    --- PASS: TestSplitFrontmatter/delimitador_de_abertura_no_fim_do_arquivo (0.00s)
=== RUN   TestDecodeFrontmatterPreservesTypes
--- PASS: TestDecodeFrontmatterPreservesTypes (0.00s)
=== RUN   TestDecodeFrontmatterMalformedReturnsError
--- PASS: TestDecodeFrontmatterMalformedReturnsError (0.00s)
=== RUN   TestSlug
--- PASS: TestSlug (0.00s)
PASS
ok  	github.com/jonyd/gobsidian/internal/parser	2.019s
```

**2. Mutation proof — `SplitFrontmatter` forced to return `0` unconditionally as its third value**

Applied by renaming the original function body to `splitFrontmatterReal` and making `SplitFrontmatter` call it, discard its offset, and return `0`:

```go
func SplitFrontmatter(data []byte) ([]byte, []byte, int64) {
	fm, body, _ := splitFrontmatterReal(data)
	return fm, body, 0
}

func splitFrontmatterReal(data []byte) ([]byte, []byte, int64) {
	if !bytes.HasPrefix(data, fmDelim) {
	... (rest unchanged)
```

`go test -race ./internal/parser/ -v -run TestSplitFrontmatter`:

```
=== RUN   TestSplitFrontmatter
=== RUN   TestSplitFrontmatter/presente
    frontmatter_test.go:112: offset = 0, quer 17
=== RUN   TestSplitFrontmatter/ausente
=== RUN   TestSplitFrontmatter/tres_tracos_no_meio_nao_conta
=== RUN   TestSplitFrontmatter/delimitador_nao_fechado
=== RUN   TestSplitFrontmatter/frontmatter_vazio
    frontmatter_test.go:112: offset = 0, quer 8
=== RUN   TestSplitFrontmatter/CRLF
    frontmatter_test.go:112: offset = 0, quer 20
=== RUN   TestSplitFrontmatter/CRLF_com_frontmatter_vazio
    frontmatter_test.go:112: offset = 0, quer 10
=== RUN   TestSplitFrontmatter/so_frontmatter,_sem_corpo
    frontmatter_test.go:112: offset = 0, quer 17
=== RUN   TestSplitFrontmatter/sem_newline_final
    frontmatter_test.go:112: offset = 0, quer 16
=== RUN   TestSplitFrontmatter/delimitador_de_abertura_no_fim_do_arquivo
--- FAIL: TestSplitFrontmatter (0.00s)
    --- FAIL: TestSplitFrontmatter/presente (0.00s)
    --- PASS: TestSplitFrontmatter/ausente (0.00s)
    --- PASS: TestSplitFrontmatter/tres_tracos_no_meio_nao_conta (0.00s)
    --- PASS: TestSplitFrontmatter/delimitador_nao_fechado (0.00s)
    --- FAIL: TestSplitFrontmatter/frontmatter_vazio (0.00s)
    --- FAIL: TestSplitFrontmatter/CRLF (0.00s)
    --- FAIL: TestSplitFrontmatter/CRLF_com_frontmatter_vazio (0.00s)
    --- FAIL: TestSplitFrontmatter/so_frontmatter,_sem_corpo (0.00s)
    --- FAIL: TestSplitFrontmatter/sem_newline_final (0.00s)
    --- PASS: TestSplitFrontmatter/delimitador_de_abertura_no_fim_do_arquivo (0.00s)
FAIL
FAIL	github.com/jonyd/gobsidian/internal/parser	1.014s
FAIL
```

Confirms the new `CRLF`, `CRLF com frontmatter vazio`, `so frontmatter, sem corpo`, and `sem newline final` rows fail under the mutation, alongside the pre-existing `presente` and `frontmatter vazio` rows. `ausente`, `tres tracos no meio nao conta`, `delimitador nao fechado`, and `delimitador de abertura no fim do arquivo` correctly stay green — their expected offset genuinely is `0`.

Restored `frontmatter.go` from a pre-mutation backup copy (`cp`/`rm`), then re-ran the full suite clean (`go clean -testcache && go test -race ./internal/parser/ -v`) — all pass again, output identical to the first run above.

`git diff internal/parser/frontmatter.go` after restore shows only the intended doc-comment change (the `splitFrontmatterReal` mutation is fully gone):

```
diff --git a/internal/parser/frontmatter.go b/internal/parser/frontmatter.go
index ac73c79..e3fba36 100644
--- a/internal/parser/frontmatter.go
+++ b/internal/parser/frontmatter.go
@@ -10,8 +10,16 @@ import (
 var fmDelim = []byte("---")
 
 // SplitFrontmatter separa o bloco YAML do corpo, devolvendo tambem o offset
-// em que o corpo comeca no arquivo original. O offset e o que mantem todos os
-// offsets de heading e de bloco corretos em relacao ao arquivo, nao ao corpo.
+// em que o corpo comeca. O offset e o que mantem os offsets de heading e de
+// bloco corretos em relacao ao inicio do buffer, nao ao corpo.
+//
+// O offset e relativo AO SLICE RECEBIDO, nao ao arquivo em disco. A distincao
+// so importa quando ha BOM: como esta funcao exige entrada ja sem BOM, o
+// buffer e tres bytes mais curto que o arquivo. Para converter um offset daqui
+// em posicao no arquivo, quem tiver o arquivo precisa somar len(bom) quando
+// vault.StripBOM tiver reportado true. Nao somar produz deslocamento de
+// exatamente tres bytes em toda leitura de secao — silencioso, e so em notas
+// com BOM.
 //
 // Exige entrada ja sem BOM UTF-8. Quem produz essa garantia e
 // vault.StripBOM — este pacote nao a chama, para nao duplicar logica que ja
```

**3. `go vet ./...`, `gofmt -l .`, `GOOS=linux go vet ./...`, `GOOS=darwin go vet ./...`**

All four ran clean — zero output from each.

**4. `git status --porcelain`**

```
 M internal/parser/frontmatter.go
 M internal/parser/frontmatter_test.go
 M internal/parser/slug_test.go
 M internal/parser/types.go
```

No stray files; only the four files in scope are modified.
