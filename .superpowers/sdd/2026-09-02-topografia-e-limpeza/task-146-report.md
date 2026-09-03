# Task 146 — Relatório

## Status
DONE

## Commit
`d5f784b` — `test(bench): add analysis benchmarks and publish the 2026-09-02 baseline`

## Evidência de TDD
Não se aplica. Esta tarefa versiona benchmarks já escritos e mede hoje (não
escreve regra nova nem teste que valida comportamento) — não há RED/GREEN.

## Prova de mutação
Não se aplica. O brief é explícito: "Esta tarefa não tem prova de mutação:
ela versiona benchmarks e documenta uma medição, não entrega regra nova."

## Step 1 — cinco arquivos compilam e rodam uma iteração

`vault_5000` já existia em `$env:TEMP\vault_5000` (não precisou regenerar).

Comando:
```
go test ./internal/service ./internal/index ./internal/search ./internal/writer ./internal/parser -run XXX_NENHUM -bench 'LinkGraphBoth|TagList|NoteListPorTag|SaveIndexCacheReal|TagsSemPrefixo|SaveInvertedCacheReal|RewriteLinksMuitos|NotaLonga' -benchtime 1x
```

Saída (resumida, cinco `ok`, zero `FAIL`, zero `SKIP`):
```
pkg: github.com/jonyd/gobsidian/internal/service
BenchmarkLinkGraphBothDepth2-12    	       1	     57400 ns/op	    2112 B/op	      41 allocs/op
BenchmarkTagListPlano-12           	       1	     91500 ns/op	   13672 B/op	       9 allocs/op
BenchmarkTagListHierarquico-12     	       1	   8161200 ns/op	 1079240 B/op	   10879 allocs/op
BenchmarkNoteListPorTag-12         	       1	    632200 ns/op	   44448 B/op	     214 allocs/op
PASS
ok  	github.com/jonyd/gobsidian/internal/service	38.836s

pkg: github.com/jonyd/gobsidian/internal/index
BenchmarkSaveIndexCacheReal-12            	       1	  42147100 ns/op	 1258792 B/op	    6701 allocs/op
BenchmarkSaveIndexCacheRealComFsync-12    	       1	  42518600 ns/op	 1223728 B/op	    6095 allocs/op
BenchmarkTagsSemPrefixo-12                	       1	     55400 ns/op	    7528 B/op	       8 allocs/op
PASS
ok  	github.com/jonyd/gobsidian/internal/index	1.846s

pkg: github.com/jonyd/gobsidian/internal/search
BenchmarkSaveInvertedCacheReal-12            	       1	 227306100 ns/op	26160832 B/op	  254973 allocs/op
BenchmarkSaveInvertedCacheRealComFsync-12    	       1	 377707700 ns/op	26161248 B/op	  254972 allocs/op
PASS
ok  	github.com/jonyd/gobsidian/internal/search	22.912s

pkg: github.com/jonyd/gobsidian/internal/writer
BenchmarkRewriteLinksMuitos-12    	       1	   4171400 ns/op	 4176272 B/op	     802 allocs/op
PASS
ok  	github.com/jonyd/gobsidian/internal/writer	0.748s

pkg: github.com/jonyd/gobsidian/internal/parser
BenchmarkParseNotaLonga-12               	       1	   3123900 ns/op	   9.24 MB/s	 1248256 B/op	    9348 allocs/op
BenchmarkDetectCandidatesNotaLonga-12    	       1	    228500 ns/op	 126.35 MB/s	  129760 B/op	    1301 allocs/op
PASS
ok  	github.com/jonyd/gobsidian/internal/parser	0.621s
```
(Os `ns/op` de uma única iteração de `-benchtime 1x` divergem em ordem de
grandeza das medianas de 5 amostras da tabela publicada — esperado, é ruído
de uma iteração só; a tabela vem da medição de hoje mais cedo com `-count=5`
guardada em `%LOCALAPPDATA%\gobsidian-bench\2026-09-02\antes_all.txt`.)

## Step 2 — tabela colada em `docs/ESTADO.md`

Subseção `### Baseline dos benchmarks de análise — 2026-09-02, HEAD 6c5d1f1`
adicionada dentro de `## Marcos`, logo depois do trecho de memória do M7 (a
outra tabela de medição na mesma seção) e antes do `---` que fecha `## Marcos`.
Colada sem a coluna "Task que compara", mantendo a frase das 5 amostras e a
referência aos binários em `%LOCALAPPDATA%\gobsidian-bench\2026-09-02\`.

## Step 3 — gofmt, vet, gate

`gofmt -l ./internal` → saída vazia.

`pwsh -File scripts/verify.ps1` → verde, 13/13 etapas:
```
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
[OK] check_tool_params
[...] 12. check_doc_refs
[OK] check_doc_refs
[...] 13. check_readme_anchors
[OK] check_readme_anchors

[OK] Bateria completa. Pode commitar.
```

## Step 4 — commit

```
git add internal/service/bench_analise_test.go internal/index/bench_analise_test.go internal/search/bench_analise_test.go internal/writer/bench_analise_test.go internal/parser/bench_analise_test.go docs/ESTADO.md
git commit -m "test(bench): add analysis benchmarks and publish the 2026-09-02 baseline"
```
Resultado: `[master d5f784b] test(bench): add analysis benchmarks and publish the 2026-09-02 baseline` — 6 arquivos, 354 inserções, 0 remoções (só criação de arquivo + acréscimo em ESTADO.md).

## Verificações do brief

**1. Os cinco arquivos não estavam versionados antes; estão depois do commit.**

Antes (do despacho, confirmado nesta sessão antes do commit):
```
?? internal/index/bench_analise_test.go
?? internal/parser/bench_analise_test.go
?? internal/search/bench_analise_test.go
?? internal/service/bench_analise_test.go
?? internal/writer/bench_analise_test.go
```

Depois (`git ls-files internal/*/bench_analise_test.go`):
```
internal/index/bench_analise_test.go
internal/parser/bench_analise_test.go
internal/search/bench_analise_test.go
internal/service/bench_analise_test.go
internal/writer/bench_analise_test.go
```

**2. A tabela colada tem os mesmos números da seção "Baseline medida" do plano.**
Três linhas ao acaso, comparando plano vs. `docs/ESTADO.md` (idênticas):

- `| \`TagListHierarquico\` (service) | 6,163 ms | 1,03 MiB | 10 880 |`
- `| \`SaveInvertedCacheReal\` / \`ComFsync\` (search) | 227,3 / 259,1 ms | 24,95 MiB | 255 000 |`
- `| \`ParseNotaLonga\` (parser) | 2,611 ms | 1,19 MiB | 9 247 |`

(Conferido via `git diff 6c5d1f1..HEAD -- docs/ESTADO.md` — colado abaixo — que
mostra a tabela inteira linha a linha idêntica à do plano, menos a coluna
"Task que compara" removida, como pedido.)

**3. `go vet ./internal/...` limpo, com os arquivos de bench entrando pela primeira vez.**
```
$ go vet ./internal/...
(sem saída, exit 0)
```
Também confirmado dentro do gate (etapas 4–6, os três `GOOS`).

**4. Nenhum benchmark foi renomeado.**
Nomes extraídos dos cinco arquivos (`grep -o 'func Benchmark[A-Za-z0-9]*'`):
```
internal/index/bench_analise_test.go:func BenchmarkSaveIndexCacheReal
internal/index/bench_analise_test.go:func BenchmarkSaveIndexCacheRealComFsync
internal/index/bench_analise_test.go:func BenchmarkTagsSemPrefixo
internal/parser/bench_analise_test.go:func BenchmarkDetectCandidatesNotaLonga
internal/parser/bench_analise_test.go:func BenchmarkParseNotaLonga
internal/search/bench_analise_test.go:func BenchmarkSaveInvertedCacheReal
internal/search/bench_analise_test.go:func BenchmarkSaveInvertedCacheRealComFsync
internal/service/bench_analise_test.go:func BenchmarkLinkGraphBothDepth2
internal/service/bench_analise_test.go:func BenchmarkNoteListPorTag
internal/service/bench_analise_test.go:func BenchmarkTagListHierarquico
internal/service/bench_analise_test.go:func BenchmarkTagListPlano
internal/writer/bench_analise_test.go:func BenchmarkRewriteLinksMuitos
```
Todos batem, letra por letra, com a lista em "Produces" do brief.

## verify.ps1 — última linha
```
[OK] Bateria completa. Pode commitar.
```
13 de 13 etapas com `[OK]`.

## O que ficou de fora
Nada. `.superpowers/sdd/2026-07-25-gobsidian-v01/progress.md` apareceu
modificado no `git status` desde antes do início desta tarefa — não é meu, o
brief não o cita entre os arquivos a commitar, e não toquei nele.

## `git status --porcelain` (depois do commit)
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
?? .superpowers/sdd/2026-09-02-topografia-e-limpeza/
?? Resume-Claude.ps1
?? docs/superpowers/plans/2026-09-02-topografia-e-limpeza.md
?? "test-vault/test vault/.obsidian/community-plugins (conflito 01M08RSCY410E88M0516VB4K9R 2026-08-17 22h52).json"
?? "test-vault/test vault/.obsidian/core-plugins (conflito 01M08RSCY410E88M0516VB4K9R 2026-08-17 22h52).json"
?? "test-vault/test vault/.obsidian/plugins/gosync/"
?? "test-vault/test vault/Ação.md"
?? "test-vault/test vault/Pasted image 20260814221454.png"
?? "test-vault/test vault/Sem título.md"
?? "test-vault/test vault/main.js"
?? "test-vault/test vault/manifest.json"
```
Todos esses são arquivos do usuário/de outras tarefas que já estavam
untracked/modificados antes desta sessão (conferido contra o `gitStatus` do
despacho) — nenhum foi tocado por esta tarefa.
