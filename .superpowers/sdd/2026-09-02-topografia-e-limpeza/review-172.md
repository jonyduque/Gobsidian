# Revisão da Task 172 — `ae619cf` sobre `011042f`

## Spec compliance

**Veredito: APROVADO COM RESSALVA** — todos os passos do brief foram entregues e
verificados; o que falta é a sincronização da documentação normativa que a
própria exclusão do arquivo tornou falsa (N1).

- **Step 1 — chamadores migram.** OK. `internal/service/write.go`: 5 chamadas
  `writer.WriteAtomic` → `vault.WriteAtomic` (`:160,:259,:388,:639,:845`).
  `cmd/gobsidian/servico.go`: `SweepResult`, `SweepStaleTempFiles` e os dois
  comentários migrados; import `writer` removido — `go list` confirma que
  `cmd/gobsidian` não o lista mais. O import de `writer` em `service` permanece
  (PathLocker, seções, diff), como o brief mandava.
- **Steps 2 e 3 — caches via `ReplaceFile`.** OK,
  `internal/index/persist.go:113-118` e `internal/search/persist.go:69-82`,
  ambos idênticos ao brief. A ordem `promover → exportar → gravar` está correta e
  o comentário nomeia o mecanismo real. Nenhum buffer fica pendente quando o
  `Sync()` roda: `escreveIndexCache` faz `Flush()` e só então escreve o CRC direto
  no `w` (`internal/index/persist_codec.go:389-397`), e `escreveCache` termina em
  `return e.w.Flush()` (`internal/search/persist_codec.go:341`).
- **Prova de inversão.** GENUÍNA. O FAIL colado só pode ter saído do código novo:
  encadeia `gravando cache de busca em %q` (wrapper novo, `persist.go:80`) com
  `falha ao renomear %q para %q apos 10 tentativas` (`vault/atomic.go:220`, que
  não existia no caminho antigo), e o temporário citado é
  `.gobsidian-tmp-1341878549` — o padrão `TempFilePrefix+"*"` do `ReplaceFile`,
  não o `.gobsidian-tmp-cache-*.gob` da rotina antiga. `persist_test.go:107` é de
  fato o `t.Fatalf` de `TestSaveOverwritesMappedCache`.
- **Step 4 — encaminhadores apagados.** OK. `internal/writer/atomic.go` deletado;
  `grep` dos quatro símbolos volta vazio (rodei, exit 1). Testes do `writer`
  migrados para `vault.WriteAtomic`.
- **N2 e N5 da revisão da 171.** OK: o grep de `.gobsidian-tmp-` devolve exatamente
  `internal/vault/atomic.go:14` e as duas fixtures de `walk_test.go`; a docstring
  do callback (`internal/vault/atomic.go:106-109`) diz as duas coisas — não
  fechar/renomear o `*os.File`, e esvaziar buffer antes de retornar.
- **Step 5 — benchstat.** OK. n=7 intercalado, base de `011042f`, números
  publicados em `docs/ESTADO.md:114,:120` e no bloco novo `:131-150`, byte a byte
  iguais aos do corpo do commit. A leitura é honesta: o invertido está registrado
  como "sem diferença estatisticamente significativa (p=0,097)" em vez de forçado
  para uma alta. Nenhum número não medido.
- **Formato de cache.** INALTERADO: `git diff 011042f..ae619cf` nos dois
  `persist.go` não toca nenhuma linha com `FormatVersion`.
- **Grafo de imports.** Sem aresta nova; nenhum dos quatro pacotes ganhou
  `writer`, e `writer` continua importando `vault` por outros arquivos, então a
  linha do `CLAUDE.md` segue válida. Saída em "Verified myself".
- **Higiene do commit.** OK: 9 caminhos, todos do escopo; `test-vault/`,
  `.claude/skills/`, `Resume-Claude.ps1` e o ledger ficaram de fora.
  Conventional Commit em inglês. Sem `runtime.GOOS` novo, sem
  helpers/utils/common, sem print em stdout, sem import de `net`.
- **Honestidade do relatório.** OK. Timestamps do Progresso batem com o commit
  (08:09 × `Sun Sep 6 08:09:56 2026`). Os três "Concerns" são divulgações reais —
  um deles vira N3. A última linha do `verify.ps1` é a exigida; não a reconferi
  (instrução de não rodar o gate), e nada no repositório a contradiz.

## Code quality

**Veredito: BOM.** A migração é mecânica e correta, o comentário da ordem de
promoção explica o mecanismo em vez de repetir o código, e a paridade de erro
está preservada: onde o código antigo fazia `Close`+`Remove` em cada caminho de
falha, o `defer` do `ReplaceFile` faz o mesmo, e cada erro continua embrulhado
com caminho e fase. A troca de "codificando cache" por "escrevendo no temporario
%q" perde o nome da fase e ganha o caminho — empate. `ctx` passou a ser conferido
em pontos que antes não conferia (dentro do laço de rename), o que é ganho.

## Findings

### N1 — blocking — documentação normativa ainda descreve o arquivo apagado

`docs/ESTRUTURA.md:119-121` e `docs/ARCHITECTURE.md:107`.

`ESTRUTURA.md` é a "árvore autoritativa de arquivos" e continua listando
`writer/atomic.go` — "encaminhadores transitórios para vault.WriteAtomic e
vault.SweepStaleTempFiles; somem quando o último chamador migrar" — para um
arquivo que este commit apagou. `ARCHITECTURE.md:107` diz, no presente, "O que
resta em `writer/atomic.go` são encaminhadores transitórios, que somem quando o
último chamador migrar", com o parágrafo inteiro sobre a decisão de não marcar
`// Deprecated:`. Essa era exatamente a frase que a Task 172 fecha.

Não é achado de estilo: o `CLAUDE.md` manda que mudança de nome ou comportamento
atualize `docs/` no mesmo PR, e a normativa é a camada que vence onde divergir de
qualquer outra coisa. Conferi o plano: nenhuma tarefa posterior toca esses dois
trechos (as menções a `ARCHITECTURE.md` nas Tasks 173+ são sobre a camada
`boot`), então ninguém mais vai consertar isso. `check_doc_refs.ps1` não pega —
ele confere o nome-base, e `atomic.go` continua existindo em `vault/`, então o
`verify.ps1` verde é consistente com o defeito.

**Fix:** apagar as três linhas de `writer/atomic.go` da árvore em
`ESTRUTURA.md:119-121`, e em `ARCHITECTURE.md:107` substituir o parágrafo por uma
frase no passado — algo como "A substituição atômica passou para `internal/vault`
(§2.5) na Task 171; os encaminhadores transitórios que ficaram em
`writer/atomic.go` foram removidos na Task 172, quando o último chamador migrou."

### N2 — should-fix — fatos medidos em `ESTADO.md` e `segregacao.md` que este commit falsificou

`docs/ESTADO.md:158-159` e `docs/segregacao.md:85-87`.

`ESTADO.md:159` afirma "Prefixo `.gobsidian-tmp-` em 4 literais:
`writer/atomic.go:14`, `vault/walk.go:74`, `search/persist.go:87`,
`index/persist.go:124`" — três dos quatro deixaram de existir neste commit, e o
grep que eu rodei devolve um só. `:158` fala em "6 sítios" de
`writer.WriteAtomic`; eram 5, e agora zero. O implementador editou `ESTADO.md`
neste mesmo commit e deixou as duas linhas vizinhas intactas. `segregacao.md:85`
ainda intitula uma seção "Troca atômica — três implementações" e argumenta contra
unificar — a unificação é o que as Tasks 171 e 172 fizeram.

**Fix:** as duas linhas de `ESTADO.md` são medições datadas — prefixá-las com
"medido em 2026-09-02, antes das Tasks 171-172" ou trocá-las pela contagem de
hoje. Em `segregacao.md`, a seção 2 precisa de uma linha de fecho: o candidato foi
extraído nas Tasks 171/172 e hoje há uma implementação só, `vault.ReplaceFile`.

### N3 — should-fix — comentários dos próprios benchmarks publicados dizem "SEM fsync"

`internal/index/bench_analise_test.go:27-28,:41-42` e
`internal/search/bench_analise_test.go:41-42,:55-56`.

`BenchmarkSaveIndexCacheReal` — o benchmark cujo número foi publicado em
`ESTADO.md:114` como "com fsync — Task 172" — é documentado logo acima como "mede
a gravacao do cache de indice como ela e hoje: CreateTemp + gob + Close + Rename,
SEM fsync". O par `...ComFsync` diz medir "o custo que a gravacao PASSARIA a
ter". As duas frases estão no presente e ficaram falsas neste commit, e a
primeira contradiz diretamente a nota que este commit escreveu em `ESTADO.md`.
O implementador registrou isto em "Concerns" e não corrigiu por estar fora da
lista de **Files** — a divulgação está certa, mas o defeito segue no repositório
e nenhuma tarefa o cobre.

**Fix:** trocar "como ela e hoje: ... SEM fsync" por "via `vault.ReplaceFile`, com
fsync do arquivo e do diretório (Task 172)", e no par `...ComFsync` registrar que
ele hoje soma um segundo `Sync` por fora e sobrevive só como referência
histórica — ou apagá-lo, se ninguém mais o compara.

### N4 — should-fix — o modo do arquivo de cache mudou de 0600 para 0644 e ninguém disse

`internal/vault/atomic.go:153-156`, alcançado agora por
`internal/index/persist.go:114` e `internal/search/persist.go:77`.

A rotina antiga criava o temporário com `os.CreateTemp` (0600) e renomeava sem
mexer no modo: o cache final ficava 0600. `ReplaceFile` faz `Stat` no alvo e,
quando ele não existe, aplica `0644` — regra escrita para notas do cofre (achado
M12), onde é a coisa certa. O efeito colateral é que `index_cache.gob` e
`inverted_cache.gob` passam a nascer 0644. No Windows o runtime do Go ignora
quase tudo do modo, e é onde este projeto roda; em Unix (que o projeto sustenta —
`syncdir_unix.go`, `restrictPermission` do `ipc`) é um alargamento de permissão
sobre um arquivo que carrega caminhos e texto derivado do cofre. Nenhum teste
fixa o modo do cache e o relatório não nomeia a mudança.

**Fix:** decidir e registrar, não necessariamente mudar código. Se 0644 serve,
uma frase em `docs/ESTADO.md` basta; se não serve, o caminho é `ReplaceFile`
receber o modo em vez de assumir 0644 — mudança de assinatura que vale tarefa
própria, não remendo aqui.

### N5 — nit — a segunda frase do comentário da promoção promete o que `ExportForCache` já garante

`internal/search/persist.go:71-72`. "Exportar depois de promover garante que os
slices gravados nao apontam para o mapeamento fechado" é verdade, mas insinua um
perigo que não existe: `ExportForCache` já copia cada posting
(`internal/search/inverted.go:684-686`, `pCopy := make(...); copy(pCopy, pos)`),
então os slices exportados nunca aliasam a arena, em qualquer ordem. A razão que
de fato prende a ordem é a primeira frase — o rename de dentro do `ReplaceFile`
falha com o alvo mapeado —, e ela está certa.

**Fix:** cortar a última frase, ou trocá-la por "(`ExportForCache` já copia as
posições, então a ordem aqui é sobre o rename, não sobre aliasing)".

## Verified myself

Tudo em primeiro plano, sem `verify.ps1` e sem `test_orphans.ps1`.

```
$ grep -rn "writer\.\(WriteAtomic\|SweepStaleTempFiles\|TempFilePrefix\|SweepResult\)" --include=*.go .
exit=1   (vazio)

$ grep -rn '\.gobsidian-tmp-' --include=*.go internal cmd
internal/vault/atomic.go:14:const TempFilePrefix = ".gobsidian-tmp-"
internal/vault/walk_test.go:57:	writeFile(t, root, ".gobsidian-tmp-abc123.md", ...)
internal/vault/walk_test.go:121:	writeFile(t, root, ".gobsidian-tmp-abc123.md", ...)

$ go build ./...   -> exit 0, sem saída
$ go vet ./...     -> exit 0, sem saída

$ go test -race -count=1 ./internal/index/ ./internal/search/ ./internal/writer/ ./internal/vault/
ok  	github.com/jonyd/gobsidian/internal/index	6.311s
ok  	github.com/jonyd/gobsidian/internal/search	19.616s
ok  	github.com/jonyd/gobsidian/internal/writer	2.970s
ok  	github.com/jonyd/gobsidian/internal/vault	29.461s

$ GOOS=windows go list -f '{{.ImportPath}}: {{.Imports}}' ./internal/index/ ./internal/search/ ./internal/service/ ./cmd/gobsidian/ ./internal/writer/
index:   [... parser text vault ...]                sem writer
search:  [... index parser text vault ...]          sem writer
service: [... index parser search vault writer ...] writer esperado (PathLocker/seções/diff)
cmd:     [... mcpsrv search service vault watcher ...]  sem writer
writer:  [bytes errors fmt parser text vault sort strings sync]  writer -> vault segue de pé

$ git show --stat ae619cf   -> 9 arquivos (os 9 do brief), nenhum fora do escopo;
  internal/writer/atomic.go deletado, 50 linhas
$ git status --short -- internal cmd docs   -> vazio

$ git diff 011042f..ae619cf -- internal/index/persist.go internal/search/persist.go | grep FormatVersion
(vazio — versões de formato intactas)
```

## Round 1

| # | Verdict | Evidence |
|---|---|---|
| N1 | addressed | `ESTRUTURA.md:119-121` (writer/atomic.go tree entry) deleted per `diff --git a/docs/ESTRUTURA.md`; `ARCHITECTURE.md:107` rewritten to past tense: "removidos na Task 172, quando o último chamador migrou." |
| N2 | addressed | `ESTADO.md:158-159` both facts prefixed "medido em 2026-09-02, antes das Tasks 171-172" with today's fact appended (0 sítios; one literal `vault/atomic.go:14`). `segregacao.md` gained the closing line asked for: "Extraído nas Tasks 171 e 172: hoje há uma implementação só, `vault.ReplaceFile`" (line ~97). |
| N3 | addressed | `internal/index/bench_analise_test.go:27-28` and `internal/search/bench_analise_test.go:41-42` reworded exactly to the suggested fix ("via `vault.ReplaceFile`, com fsync..."); `...ComFsync` comments now say they add a second, redundant Sync and are kept as historical reference only, per instruction not to delete. |
| N4 | addressed against ruling | No code change, as ruled. `ESTADO.md` gained a paragraph in the cache-format section stating caches now land `0644` via `vault.ReplaceFile`, framed as accepted (not a regression), citing this ruling. |
| N5 | addressed | `internal/search/persist.go:71-72` second sentence replaced with reviewer's exact suggested wording about `ExportForCache` already copying positions. |

Scope: diff touches exactly the 7 files N1-N5 require (`ARCHITECTURE.md`, `ESTADO.md`, `ESTRUTURA.md`, `segregacao.md`, both bench test files, `search/persist.go`). No scope creep.

Checks run: `gofmt -l ./internal` empty. `go vet ./...` clean. All four `.md` files parse as UTF-8.

`grep -n 'writer/atomic.go' docs/*.md CLAUDE.md`: five hits are past-tense/historical (`ARCHITECTURE.md:107`, `ESTADO.md:159`, and the three in `REVISAO-2026-08-15.md`/`SUGESTOES.md`, which are dated historical audit docs describing the pre-171 state). One hit is **not** past tense:

### N6 — nit — `docs/segregacao.md:87` still cites `writer/atomic.go:141` as a live fact

Section "### 2. Troca atômica — três implementações" (title unchanged) opens with
`writer/atomic.go:141`, `index/persist.go:124`, `search/persist.go:87`. and a
present-tense paragraph ("A do `writer` é a cuidadosa...") describing a file
this and the prior commit deleted, plus a now-moot "Recomendação: só junto com
o item 1" for a unification already done. The closing line added for N2 says
the extraction happened, but doesn't retract the stale section above it — the
section now argues against doing something it also says is done.

**Fix (optional, non-blocking):** update the section title (drop "três
implementações") and the opening file-citation line to name `vault/atomic.go`
historically, or fold the whole section into the closing line. Low stakes:
`segregacao.md` is historical, not normative, and the closing line does
disambiguate for a careful reader.

**Verdict: N1, N2, N3, N4, N5 all addressed. N6 is a new nit, non-blocking.**
