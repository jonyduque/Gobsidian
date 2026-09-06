# Task 170 — Otimizações medidas: só entra o que o `benchstat` aprova

## Progresso

- 05:53 — início; lidos `CLAUDE.md`, `docs/papeis/implementador.md`, `docs/papeis/desempenho.md`, o brief. Base `a6521d1`.
- 05:56 — mapeados os cinco candidatos por símbolo (os números de linha do brief são anteriores às Tasks 166–169). Cofre de bench presente: 5000 `.md` em `%TEMP%\vault_5000`. `benchstat` em `/c/Users/jonyd/go/bin/benchstat`.
- 05:59 — candidato 1 (`removeLocked`, índice reverso doc→termos no delta): medido 7×2 intercalado. `-70,64%` sec/op, `p=0,001`, `B/op +8,33%`. **Entra.** `go test -race ./internal/search/` verde. Commit `8e0670c`.
- 06:01 — candidato 2 (`bytes.Index`/`bytes.Count` em `findMarkdownLinkSpan`): perfil de cobertura sob o próprio benchmark mostra o bloco com contagem **0**; medido mesmo assim, `p=0,620`, `allocs/op` idêntico. **Não entra**, revertido à mão (`git diff` de `internal/parser` vazio).
- 06:04 — candidato 3 (`RewriteLinks` em uma passada, `Grow` único): `-96,00%` sec/op, `-98,61%` B/op, `p=0,001`. **Entra.** `go test -race ./internal/writer/` e `go test ./internal/service/` (os dois chamadores) verdes. Commit `6ec12da`.
- 06:12 — candidato 4 (variante sem `Normalize` para termo já normalizado): sec/op `~` (`p=0,128`); B/op `-0,27%` e allocs/op `-0,36%` com `p=0,001`. **Não entra** — ver a decisão registrada abaixo. Revertido à mão; `git status --porcelain internal/` vazio.
- 06:17 — candidato 5 (içamento da normalização das tags para fora do laço): perfil de cobertura sob o próprio benchmark mostra o bloco de tags com contagem **0**; medido mesmo assim, `~` nos três (`p=0,710` / `0,902` / `1,000`). **Não entra**, revertido à mão; árvore de `internal/` limpa.
- 06:17 — início do gate: `go test -race` nos quatro pacotes, depois `verify.ps1` completo.
- 06:22 — `go test -race -count=1` verde nos quatro pacotes.
- 06:27 — `verify.ps1` completo verde: `[OK] Bateria completa. Pode commitar.`
- 06:29 — relatório escrito; encoding UTF-8 validado.
- 06:30 — `audit_reports.ps1 170` rodado, os dois achados contra este relatório respondidos na seção Verificações. Relatório commitado em `314d930`.
- 06:31 — fim. Árvore de `internal/`, `cmd/` e `docs/` limpa; nenhum arquivo do dono tocado.

---

## Status

**DONE_WITH_CONCERNS.** Dois dos cinco candidatos entraram, cada um com `p=0,001`;
três saem como "sem ganho medido". A ressalva é o candidato 4 — ver
[Concerns](#concerns).

| # | Candidato | Bench | Veredito | SHA |
|---|---|---|---|---|
| 1 | `removeLocked`: índice reverso doc→termos no delta | `InvertedUpdateLote` | **entrou** — sec/op `-70,64%`, `p=0,001` | `8e0670c` |
| 2 | `bytes.Index`/`bytes.Count` em `findMarkdownLinkSpan` | `ParseNotaLonga` | sem ganho medido (`p=0,620`) | — |
| 3 | `RewriteLinks` em uma passada, `Grow` único | `RewriteLinksMuitos` | **entrou** — sec/op `-96,00%`, `p=0,001` | `6ec12da` |
| 4 | Variante sem `Normalize` para termo já normalizado | `SearchTermoAmploCache` | sem ganho medido em sec/op (`p=0,128`) | — |
| 5 | Içamento da normalização das tags para fora do laço | `SearchFiltroFrontmatter` | sem ganho medido (`p=0,710`) | — |

`git log --oneline a6521d1..HEAD`:

```
6ec12da perf(writer): rewrite every link in one pass instead of one buffer per link
8e0670c perf(search): reverse doc->terms map makes delta removal O(terms of doc)
```

---

## Método de medição

Para **cada** candidato os dois binários foram construídos da MESMA árvore, o
`base170` antes de aplicar a edição daquele candidato e o `depois` logo após —
nunca contra os `antes_*.test.exe` de 2026-09-02, que carregam as mudanças da
Task 169 e misturariam o ganho dela com o desta tarefa.

Sete amostras por lado, **intercaladas** uma de cada por vez
(`-run '^$' -bench '<nome>' -benchmem -count=1`, redirecionado para arquivo e
concatenado), pelo motivo do `docs/papeis/desempenho.md`: bateladas separadas
medem a deriva da máquina junto com a diferença entre os braços, e isso já
publicou dois números errados neste projeto. Nada mais rodou durante as
medições — nenhum gate concorrente, nenhum `test_orphans.ps1`.

Cofre: `%TEMP%\vault_5000`, 5000 `.md` (`find ... -name '*.md' | wc -l` = 5000).
Binários e amostras brutas em `%LOCALAPPDATA%\gobsidian-bench\2026-09-06\`
(`c<N>_base.txt`, `c<N>_depois.txt`).

`CacheFormatVersion` está intacto: nenhum candidato precisou de mudança de
formato, e o único que mexe em estrutura de dados (candidato 1) mexe só no
**delta**, que é reconstruído a partir de `Add` e nunca é serializado.

---

## Candidato 1 — `removeLocked`: índice reverso doc→termos no delta — **ENTROU** (`8e0670c`)

`removeLocked` varria TODO termo do delta para apagar um caminho. No laço de
construção do zero — um `Add` por nota, que é o que `buildInvertedIndex` faz —
isso é quadrático: a n-ésima nota percorre os termos que as n-1 anteriores
acumularam só para não achar nada, porque caminho novo ainda não está em termo
nenhum.

`termosDoDoc map[string][]string` é o índice reverso do delta, escrito em
`addTermPositionLocked` exatamente quando o par (termo, path) aparece pela
primeira vez e apagado inteiro em `removeLocked`. Invariante:
`terms[t][p]` existe se e somente se `t` está em `termosDoDoc[p]`. Também é
transferido e zerado em `AdotarDe`, que é o outro lugar que move o delta
inteiro. `NewInverted` é o único construtor de `*Inverted` no produto
(`grep -rn "Inverted{" internal/ --include=*.go | grep -v _test.go` devolve só
a linha dele), então não há caminho que produza o mapa faltando.

**Memória é o custo, e está medido:** `+8,33%` B/op e `+5,83%` allocs/op — os
cabeçalhos de string de uma entrada por par (doc, termo).

```
goos: windows
goarch: amd64
pkg: github.com/jonyd/gobsidian/internal/search
cpu: Intel(R) Core(TM) i7-10750H CPU @ 2.60GHz
                      │ c1_base.txt │           c1_depois.txt           │
                      │   sec/op    │   sec/op    vs base               │
InvertedUpdateLote-12    3.936 ± 1%   1.156 ± 5%  -70.64% (p=0.001 n=7)

                      │ c1_base.txt  │           c1_depois.txt            │
                      │     B/op     │     B/op      vs base              │
InvertedUpdateLote-12   105.0Mi ± 0%   113.8Mi ± 0%  +8.33% (p=0.001 n=7)

                      │ c1_base.txt │           c1_depois.txt           │
                      │  allocs/op  │  allocs/op   vs base              │
InvertedUpdateLote-12   565.3k ± 0%   598.2k ± 0%  +5.83% (p=0.001 n=7)
```

Nota sobre o baseline do brief: ele cita **6,549 s** para `InvertedUpdateLote`.
Esta máquina mediu **3,936 s** no mesmo código, mesmo cofre. Não tentei
reconciliar os dois — o que decide a entrada é a comparação lado a lado feita
aqui, e o número do brief não é reusável entre máquinas.

---

## Candidato 2 — `bytes.Index`/`bytes.Count` no parser — **SEM GANHO MEDIDO**

O alvo é o ramo `if startIdx == -1` de `findMarkdownLinkSpan`
(`internal/parser/ast.go`), que trata link Markdown de texto vazio —
`"[](alvo.md)"` — e faz `strings.Index(string(body), alvo)` seguido de
`strings.Count(string(body), alvo)`, copiando o corpo inteiro da nota duas
vezes. A edição trocava as duas por `bytes.Index`/`bytes.Count` sobre `body`
direto, com o alvo montado em um `append` só.

**O benchmark designado não executa esse ramo.** Perfil de cobertura tirado sob
o próprio `BenchmarkParseNotaLonga`
(`go test -run '^$' -bench '^BenchmarkParseNotaLonga$' -benchtime=1x -coverprofile=...`):

```
github.com/jonyd/gobsidian/internal/parser/ast.go:322.2,322.20   1 1
github.com/jonyd/gobsidian/internal/parser/ast.go:322.20,337.58  3 0   <- corpo do `if startIdx == -1`
github.com/jonyd/gobsidian/internal/parser/ast.go:337.58,339.15  2 0
```

O bloco que contém as duas linhas tem contagem **0**. As notas do benchmark
usam `[md](NotaN.md)`, com texto de link, então `startIdx` sempre é resolvido
pelo prefixo. O `benchstat` confirma de outro jeito: `allocs/op` **idêntico nos
14 runs** (`all samples are equal`), que é o que se espera de código não
executado.

```
goos: windows
goarch: amd64
pkg: github.com/jonyd/gobsidian/internal/parser
cpu: Intel(R) Core(TM) i7-10750H CPU @ 2.60GHz
                  │ c2_base.txt  │         c2_depois.txt         │
                  │    sec/op    │    sec/op     vs base         │
ParseNotaLonga-12   2.114m ± 24%   2.086m ± 10%  ~ (p=0.620 n=7)

                  │  c2_base.txt  │         c2_depois.txt         │
                  │      B/s      │     B/s       vs base         │
ParseNotaLonga-12   13.03Mi ± 19%   13.20Mi ± 9%  ~ (p=0.620 n=7)

                  │ c2_base.txt  │         c2_depois.txt         │
                  │     B/op     │     B/op      vs base         │
ParseNotaLonga-12   1.188Mi ± 0%   1.188Mi ± 0%  ~ (p=0.731 n=7)

                  │ c2_base.txt │         c2_depois.txt          │
                  │  allocs/op  │  allocs/op   vs base           │
ParseNotaLonga-12   9.247k ± 0%   9.247k ± 0%  ~ (p=1.000 n=7) ¹
¹ all samples are equal
```

Nenhum benchmark do repositório cobre link de texto vazio — o outro bench do
pacote, `DetectCandidatesNotaLonga`, nem toca em link. Não escrevi um: o brief
autoriza criar benchmark faltante só para o candidato 1, e escrever um
benchmark novo para justificar uma otimização é exatamente a ordem invertida.
Se a otimização interessar, ela precisa primeiro de um caso que a exercite.

---

## Candidato 3 — `RewriteLinks` em uma passada — **ENTROU** (`6ec12da`)

O laço copiava `src` inteiro e depois realocava e recopiava o buffer inteiro a
cada substituição: no caso que o benchmark modela — nota-índice com 200 links,
todos reescritos — são 200 cópias da nota, O(n·m).

A passada única só é possível porque `work` já está ordenada por `Start`
DECRESCENTE e já foi provada sem sobreposição algumas linhas acima; percorrida
ao contrário sai crescente, e cada substituição consome o trecho intacto entre
o cursor e o `Start` do próximo link. O tamanho final é conhecido antes do
primeiro byte, então `bytes.Buffer.Grow` aloca uma vez. `BuildLinkText`
continua rodando exatamente uma vez por link; o que mudou é que o resultado é
reusado em vez de descartado.

Correção coberta por `TestRewriteLinks_*` em
`internal/writer/linkrewrite_test.go`, que comparam a saída **byte a byte**
com um `want` literal, incluindo um caso de 3 links, um de 2 (Markdown +
embed) e um com BOM e CRLF; mais os dois chamadores em
`internal/service/write.go:547` e `:634`, cujo pacote roda verde.

```
goos: windows
goarch: amd64
pkg: github.com/jonyd/gobsidian/internal/writer
cpu: Intel(R) Core(TM) i7-10750H CPU @ 2.60GHz
                      │  c3_base.txt  │           c3_depois.txt            │
                      │    sec/op     │   sec/op     vs base               │
RewriteLinksMuitos-12   1356.53µ ± 4%   54.19µ ± 9%  -96.00% (p=0.001 n=7)

                      │  c3_base.txt   │            c3_depois.txt            │
                      │      B/op      │     B/op      vs base               │
RewriteLinksMuitos-12   4073.38Ki ± 0%   56.68Ki ± 0%  -98.61% (p=0.001 n=7)

                      │ c3_base.txt │           c3_depois.txt           │
                      │  allocs/op  │ allocs/op   vs base               │
RewriteLinksMuitos-12    795.0 ± 0%   596.0 ± 0%  -25.03% (p=0.001 n=7)
```

---

## Candidato 4 — variante sem `Normalize` — **SEM GANHO MEDIDO EM `sec/op`**

A edição acrescentava `PostingsDeTermoNormalizado` e
`PositionsDeTermoNormalizado` a `Inverted`, com `Postings`/`Positions`
delegando depois de `Normalize`, e trocava os três chamadores que **já têm o
termo normalizado**: `bm25.go` (`m.term` é `qTok.Raw`/`qTok.Reduced`),
`snippet.go` (`queryTerms.termos` sai de `NovosTermosDeTrecho`, que só guarda
`tok.Raw` e `tok.Reduced`) e `service/search.go` (`tok.Raw`). A pré-condição
foi conferida nas três origens, não presumida.

`sec/op` deu `~` (`p=0,128`). B/op e allocs/op melhoram de verdade e com
significância — `-0,27%` e `-0,36%`, `p=0,001` —, mas são **20 alocações de
5583**, e a métrica que o RNF-04 nomeia é latência de `vault_search`, não
alocação. Não entrou: ver [Concerns](#concerns).

```
goos: windows
goarch: amd64
pkg: github.com/jonyd/gobsidian/internal/service
cpu: Intel(R) Core(TM) i7-10750H CPU @ 2.60GHz
                         │ c4_base.txt │         c4_depois.txt         │
                         │   sec/op    │    sec/op     vs base         │
SearchTermoAmploCache-12   5.331m ± 2%   5.206m ± 16%  ~ (p=0.128 n=7)

                         │ c4_base.txt  │           c4_depois.txt            │
                         │     B/op     │     B/op      vs base              │
SearchTermoAmploCache-12   1.495Mi ± 0%   1.491Mi ± 0%  -0.27% (p=0.001 n=7)

                         │ c4_base.txt │           c4_depois.txt           │
                         │  allocs/op  │  allocs/op   vs base              │
SearchTermoAmploCache-12   5.583k ± 0%   5.563k ± 0%  -0.36% (p=0.001 n=7)
```

---

## Candidato 5 — içamento da normalização das tags — **SEM GANHO MEDIDO**

Conforme a decisão do orquestrador, foi feito **só o içamento**: uma função
`tagsLimpas(opts)` resolvendo `strings.TrimPrefix(strings.ToLower(reqTag), "#")`
uma vez, ao lado de `casamFrontmatter`, com o resultado passado a
`matchesSearchFilters`. A comparação em si ficou intacta para a Task 180.

**O benchmark designado não executa esse ramo.** `BenchmarkSearchFiltroFrontmatter`
passa `Frontmatter`, e nenhum `Tags`. Perfil de cobertura sob o próprio
benchmark:

```
github.com/jonyd/gobsidian/internal/service/search.go:441.2,441.24  1 1   <- o `if len(opts.Tags) > 0`
github.com/jonyd/gobsidian/internal/service/search.go:441.24,443.31 2 0   <- o corpo dele
github.com/jonyd/gobsidian/internal/service/search.go:443.31,445.4  1 0
github.com/jonyd/gobsidian/internal/service/search.go:446.3,446.36  1 0
```

A condição é avaliada (contagem 1) e o corpo nunca roda (contagem 0). O
`benchstat` confirma: `allocs/op` idêntico nos 14 runs.

```
goos: windows
goarch: amd64
pkg: github.com/jonyd/gobsidian/internal/service
cpu: Intel(R) Core(TM) i7-10750H CPU @ 2.60GHz
                           │ c5_base.txt  │         c5_depois.txt         │
                           │    sec/op    │    sec/op     vs base         │
SearchFiltroFrontmatter-12   13.96m ± 34%   15.25m ± 13%  ~ (p=0.710 n=7)

                           │ c5_base.txt  │         c5_depois.txt         │
                           │     B/op     │     B/op      vs base         │
SearchFiltroFrontmatter-12   4.427Mi ± 0%   4.427Mi ± 0%  ~ (p=0.902 n=7)

                           │ c5_base.txt │        c5_depois.txt         │
                           │  allocs/op  │  allocs/op   vs base         │
SearchFiltroFrontmatter-12   30.12k ± 0%   30.12k ± 0%  ~ (p=1.000 n=7)
```

O brief pareia este candidato com um benchmark que não passa por ele. Nenhum
outro benchmark do repositório usa `SearchOptions.Tags`. Como a Task 180 vai
reescrever esse filtro pela chave de tag e vai precisar de um caso que o
exercite de qualquer forma, o içamento cabe lá, junto com o benchmark que o
mede — e não aqui, sem instrumento.

---

## Verificações

`go test -race -count=1 ./internal/search/ ./internal/parser/ ./internal/writer/ ./internal/service/`
(sem cache, na árvore final):

```
ok  	github.com/jonyd/gobsidian/internal/search	17.679s
ok  	github.com/jonyd/gobsidian/internal/parser	1.844s
ok  	github.com/jonyd/gobsidian/internal/writer	23.933s
ok  	github.com/jonyd/gobsidian/internal/service	48.465s
```

`pwsh -File scripts/verify.ps1` — 14 etapas, todas `[OK]`. A etapa 3 informa
sem reprovar: 6 testes pulados, os mesmos de sempre (condições de ambiente do
Windows e Unix). Última linha:

```
[OK] Bateria completa. Pode commitar.
```

`docs/bench-baseline.json` não contém entrada para `InvertedUpdateLote` nem
para `RewriteLinksMuitos` (`grep` nos dois nomes volta vazio), então os dois
ganhos não deixam número obsoleto lá.

Não rodei `scripts/test_orphans.ps1` — proibido pelo despacho.

Prova de mutação: esta tarefa não tem, por decisão do brief — a regra provada é
de desempenho e a prova é o `benchstat`. A correção do candidato 1 é coberta
por `TestIndiceRecarregadoEIdenticoAoConstruido`; a do candidato 3, pelos
`TestRewriteLinks_*` com comparação byte a byte.

`pwsh -File scripts/audit_reports.ps1 170` aponta dois achados neste relatório,
`[SECAO-AUSENTE] sem secao de TDD/RED` e `.../TDD/GREEN`. São esperados e ficam
assim: o brief dispensa a prova de mutação por escrito, e esta tarefa não
adiciona comportamento — as duas mudanças que entraram preservam comportamento e
são cobertas por testes que já existiam. Os outros 14 achados do mesmo comando
são do ledger de `2026-07-25-gobsidian-v01`, anteriores a esta tarefa e fora
dela.

O ledger não foi tocado: o despacho o lista entre os arquivos do dono que o
implementador não mexe. O registro da Task 170 cabe ao orquestrador.

---

## Concerns

**1. O candidato 4 tem melhora significativa em memória e nenhuma em latência,
e eu decidi que não entra.** É a única decisão desta tarefa que não se resolve
sozinha pela regra. `sec/op` é `~` (`p=0,128`); B/op e allocs/op melhoram com
`p=0,001`, mas em `0,27%` / `0,36%`. O preço é permanente: dois métodos
exportados a mais em `Inverted`, cuja pré-condição — termo já normalizado — é
silenciosa quando violada (o chamador recebe zero resultados, não um erro).
Apliquei o precedente da Task 82 registrado em `docs/papeis/desempenho.md`
("mudança sem ganho significativo é dívida pura") e a redação do Step 1 do
brief, que ancora a decisão em `sec/op`. Os números estão acima, revertidos e
reproduzíveis, se o orquestrador quiser decidir diferente.

**2. Dois dos cinco candidatos do brief apontam para código que o benchmark
pareado não executa** (candidatos 2 e 5), provado por perfil de cobertura em
ambos os casos. Não é um erro de medição desta tarefa, é um pareamento do brief
que não fecha — vale para quem escrever os próximos briefs de desempenho.

**3. O baseline citado no brief para `InvertedUpdateLote` (6,549 s) não
reproduz nesta máquina** (3,936 s no mesmo código e no mesmo cofre). Não
investiguei. Nenhuma conclusão desta tarefa depende disso, porque toda decisão
saiu de comparação lado a lado feita aqui — mas um baseline de brief que não
reproduz é um número que a próxima tarefa vai tentar usar.
</content>
