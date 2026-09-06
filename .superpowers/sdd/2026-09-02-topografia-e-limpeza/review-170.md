# Revisão da Task 170 — otimizações medidas

Pacote revisado: `review-a6521d1..6ec12da.diff` (2 commits de código).
Durante a revisão o implementador acrescentou dois commits de documentação
(`314d930`, `dcae200`), que commitam o próprio relatório; `git diff
a6521d1..HEAD --stat` passou de 2 para 3 arquivos por causa disso e continua
tocando só `internal/search/inverted.go`, `internal/writer/linkrewrite.go` e
`task-170-report.md`.

## Progresso

- 06:30 — início; `git log`/`git diff --stat` da base `a6521d1`; brief lido.
- 06:31 — relatório lido; `git status --short internal/` vazio.
- 06:32 — `review-a6521d1..6ec12da.diff` lido inteiro; `internal/search/inverted.go`
  lido em `Add`/`addTermPositionLocked`/`Remove`/`sombrearLocked`/`removeLocked`
  (136–261), `ExportForCache`/`AdotarDe`/`newInvertedFromSoA` (630–827);
  `internal/writer/linkrewrite.go` lido inteiro.
- 06:33 — `go test -race -count=1` nos quatro pacotes: verde.
- 06:34 — perfil de cobertura de `internal/search`; `git log` de novo (HEAD
  tinha andado para `dcae200`).
- 06:36 — `benchstat` recalculado dos brutos em
  `%LOCALAPPDATA%\gobsidian-bench\2026-09-06\` para os cinco candidatos;
  re-run `-count=3` de `RewriteLinksMuitos`; `go vet`; corpos dos commits.
- 06:40 — achados e veredito.

Não rodei `scripts/verify.ps1` nem `scripts/test_orphans.ps1` — proibidos pelo
despacho. A linha final do `verify.ps1` fica como afirmação do implementador,
não verificada aqui.

---

## Commit `8e0670c` — `perf(search)`: índice reverso doc→termos no delta

### O que muda

`Inverted` ganha `termosDoDoc map[string][]string` (`inverted.go:95`),
inicializado em `NewInverted` (`:133`), escrito em `addTermPositionLocked`
(`:193-195`), consumido e apagado em `removeLocked` (`:245-255`), transferido e
zerado em `AdotarDe` (`:762`, `:773`).

### Sincronia do mapa reverso — os caminhos, um a um

Verifiquei que **não existe outro escritor**: `grep -n "ix\.terms\|termosDoDoc"
internal/search/*.go` mostra escrita em `ix.terms` só em `inverted.go:184`
(criação do mapa interno) e `:187` (o append da posição), ambos dentro de
`addTermPositionLocked`; apagamento só em `:250` e `:252`, dentro de
`removeLocked`. A conta é única, como manda o CLAUDE.md.

- **Add de doc já presente** — `Add` chama `sombrearLocked` + `removeLocked`
  antes de reindexar (`:153-154`). O `removeLocked` já apaga
  `termosDoDoc[path]` inteiro, então a lista é reconstruída do zero. Sem
  acúmulo.
- **Re-add depois de remove** — mesmo caminho.
  `TestInvertedRemoveAndRecreateNoResidue` (`internal/search/inverted_test.go:142`)
  cobre exatamente isso e passa.
- **Doc que está no base e não no delta** — `removeLocked` lê
  `ix.termosDoDoc[path]` de mapa sem a chave, o que dá `nil`: laço vazio, e os
  dois `delete` são no-op. O base é tratado por `sombrearLocked`, intocado pelo
  diff. `soa_test.go:172,244` cobrem.
- **Remove duplo** — segunda chamada tem `termosDoDoc[path]` ausente: laço
  vazio. Inofensivo.
- **Ponta solta ao contrário** (termo listado no reverso cuja posting já saiu) —
  não é alcançável, porque a única remoção de `ix.terms[term]` acontece dentro
  do mesmo laço que apaga `termosDoDoc[path]`; e `delete(docs, p1)` nunca
  esvazia um termo que outro caminho ainda tenha. O `if !ok { continue }` de
  `:247` cobre a hipótese mesmo assim — ver N2.
- **Duplicata na lista** — `posList, jaTinha := docs[path]` (`:186`) só é falso
  na primeira ocorrência do par (termo, path), porque `docs[path]` só passa a
  existir depois de um `append` que sempre produz `len ≥ 1`. Token com
  `Raw == Reduced` já é filtrado em `Add:174`; token cujo `Reduced` coincide com
  o `Raw` de outro token cai no `jaTinha`. Sem entrada por ocorrência.
- **Compactação/merge do delta no base** — não existe no código. O único
  consumidor do delta inteiro é `ExportForCache` (`:645`), que só **lê**
  `ix.terms`/`ix.docLengths` e não é afetado; e `AdotarDe` (`:739`), que move o
  mapa junto com os outros e zera a origem.
- **Carga de cache** — `newInvertedFromSoA` (`:811`) constrói via
  `NewInverted()`, então `termosDoDoc` nunca é `nil`. O delta nasce vazio e o
  mapa reverso com ele. `NewInverted` é o único construtor: `grep -rn
  "Inverted{" internal/search/*.go` devolve só `inverted.go:130`.
- **Flush/rebuild do delta** — o único "reset" do delta é o lado `outro` de
  `AdotarDe`, e ele reinicializa `termosDoDoc` junto com `terms` e `docLengths`
  (`:771-773`). Não fica mapa reverso órfão apontando para delta zerado.

Invariante afirmada no comentário — `terms[t][p]` existe sse `t ∈
termosDoDoc[p]` — se sustenta nos caminhos acima.

### O laço novo é de fato exercitado (não só "os testes passam")

Perfil de cobertura da suíte de `internal/search`
(`go test -count=1 -coverprofile=... ./internal/search/`), blocos de
`removeLocked`:

```
inverted.go:243.47,245.44 2 1     <- entrada + delete(docLengths)
inverted.go:245.44,247.10 2 1     <- CORPO do laço novo, sobre termosDoDoc
inverted.go:247.10,248.12 1 0     <- o `if !ok { continue }` (ver N2)
inverted.go:250.3,251.21  2 1     <- delete(docs, path)
inverted.go:251.21,253.4  1 1     <- delete(ix.terms, term) quando esvazia
inverted.go:255.2,255.30  1 1     <- delete(ix.termosDoDoc, path)
```

Contagem 1 no corpo do laço: existe teste que remove um caminho **que está no
delta com termos**, que é o caso onde a mudança pode errar.

### A rede de correção não é a que o brief nomeou

O brief e o relatório (`task-170-report.md:319`) apontam
`TestIndiceRecarregadoEIdenticoAoConstruido`
(`internal/search/persist_test.go:500`) como a rede do delta. Li o teste: ele
faz três `Add` e nenhum `Remove`. Passa por `removeLocked` só pelo caminho
trivial de `Add` (lista vazia) e compara `DocCount`, `DocLength` e `TermCount`
depois de um round-trip de cache. **Ele não pode falhar por causa desta
mudança.** Ver N1. A rede real existe e é outra —
`TestInvertedRemoveLeavesNoEmptyPosting` (`inverted_test.go:14`), que verifica
termo compartilhado sobrevivendo, termo órfão saindo do dicionário e
`TermCount` exato depois de um `Remove`; e `TestInvertedRemoveAndRecreateNoResidue`
(`:142`).

### Testes

```
$ go test -race -count=1 ./internal/search/ ./internal/watcher/ ./internal/service/ ./internal/writer/
ok  	github.com/jonyd/gobsidian/internal/search	19.862s
ok  	github.com/jonyd/gobsidian/internal/watcher	17.241s
ok  	github.com/jonyd/gobsidian/internal/service	49.287s
ok  	github.com/jonyd/gobsidian/internal/writer	25.888s
```

`go vet ./internal/search/ ./internal/writer/` — limpo.

### Benchstat

Recalculei dos brutos, não copiei do relatório. `grep -c 'ns/op'`: 7 amostras
em cada um dos dez arquivos (`c1..c5`, base e depois).

```
$ benchstat c1_base.txt c1_depois.txt
InvertedUpdateLote-12    3.936 ± 1%   1.156 ± 5%  -70.64% (p=0.001 n=7)   sec/op
InvertedUpdateLote-12   105.0Mi ± 0%  113.8Mi ± 0%  +8.33% (p=0.001 n=7)  B/op
InvertedUpdateLote-12   565.3k ± 0%   598.2k ± 0%   +5.83% (p=0.001 n=7)  allocs/op
```

Bate byte a byte com a tabela do relatório e com a do corpo do commit. `p=0,001`,
`n=7` por lado, linha de `B/op` presente e o `+8,33%` citado no corpo — o preço
em memória está declarado, não escondido. Os binários da pasta confirmam o
método: `base170_search.test.exe` (05:56) e `depois170_c1_search.test.exe`
(05:57) foram construídos lado a lado, e não são os `antes_*` de 2026-09-02.

Interleaving: os brutos não carregam a ordem em que foram concatenados, então a
intercalação é afirmação do implementador. Os mtimes (`c1_base.txt` e
`c1_depois.txt` ambos 05:58) são consistentes com appends alternados e não com
duas bateladas separadas por minutos. Não é prova; é ausência de contradição.

---

## Commit `6ec12da` — `perf(writer)`: `RewriteLinks` em uma passada

### Saída byte a byte

A função não faz busca de texto: opera só sobre offsets já validados. As duas
versões percorrem a mesma sequência de spans (o `sort` por `Start` decrescente
e a checagem de sobreposição em `linkrewrite.go:43-51` são anteriores ao trecho
mudado e não foram tocados), e `BuildLinkText` roda uma vez por link nas duas.
Percorrer `work` de trás para frente (`:75`) produz `Start` crescente, e cada
volta escreve `src[cursor:Start]` + texto novo, com `cursor = End`. O resto sai
em `out.Write(src[cursor:])`. Caso a caso:

- **Vários links na mesma linha** — nada distingue linha de resto; são só
  offsets. Coberto byte a byte por `TestRewriteLinks_PreservesAliasAndAnchor`
  (`linkrewrite_test.go:11`, 3 links numa linha) e
  `TestRewriteLinks_MultipleOccurrencesInSameNote` (`:64`, 3 links numa linha).
- **Link dentro de cerca de código** — `RewriteLinks` não sabe o que é cerca,
  nem antes nem depois; quem decide os spans é o chamador. O diff não pode
  mudar esse comportamento em nenhuma direção.
- **Spans adjacentes** (`End == Start` do seguinte) — a checagem de
  sobreposição usa `>`, então adjacência é legal, e no laço novo o
  `src[cursor:Start]` daquela volta é fatia vazia. `bytes.Buffer.Write` de fatia
  vazia é no-op. Correto.
- **Link em offset 0** — primeira escrita é `src[0:0]`, vazia. Correto.
- **Link terminando no EOF** — `cursor == len(src)` e o `Write(src[cursor:])`
  final é vazio. Correto.
- **CRLF e BOM** — bytes copiados sem interpretação. Coberto por
  `TestRewriteLinks_PreservesBOMAndEOL` (`:112`), com `want` literal contendo
  `\xef\xbb\xbf` e `\r\n`.
- **Alvo que contém o nome antigo como substring** — irrelevante nas duas
  versões: não há `Index`/`Replace`, só offsets.
- **Spans duplicados** (mesmo `Start`) — continuam recusados por
  `ErrOverlappingLinks`, porque `End > Start` sempre.

Dois pontos que olhei por serem os que poderiam morder:

- `out.Grow(tamanho)` com `tamanho` negativo entraria em pânico. Não é
  alcançável: a validação de limites (`:38`) e a de sobreposição (`:47`) rodam
  **antes**, então `Σ len(span) ≤ len(src)` e `tamanho ≥ Σ len(texto novo) ≥ 0`.
  E ainda que fosse errado, `Grow` é só dica de capacidade — não afeta o
  conteúdo.
- `out.Bytes()` devolve fatia com capacidade sobrando, enquanto o `res` antigo
  vinha com `cap` exato. Os dois chamadores (`internal/service/write.go:547` e
  `:634`) consomem o resultado só para leitura — `string(...)` no dry-run e
  `WriteAtomic` na escrita — e ninguém faz `append` sobre ele. Sem diferença
  observável.

Códigos de erro: `ErrInvalidLinkOffset` e `ErrOverlappingLinks` são levantados
nos mesmos pontos, com o mesmo texto. `TestRewriteLinks_RejectsInvalidOffsets`
(`:91`) passa.

**Sobre "os testes passam" vs. "há teste que fixa a saída":** os cinco
`TestRewriteLinks_*` comparam `string(got)` com um `want` literal. Não é
tautologia — é comparação byte a byte. O que **não** existe é caso com link em
offset 0 ou terminando exatamente no EOF (ver N5).

### Benchstat

```
$ benchstat c3_base.txt c3_depois.txt
RewriteLinksMuitos-12   1356.53µ ± 4%    54.19µ ± 9%   -96.00% (p=0.001 n=7)  sec/op
RewriteLinksMuitos-12   4073.38Ki ± 0%  56.68Ki ± 0%   -98.61% (p=0.001 n=7)  B/op
RewriteLinksMuitos-12     795.0 ± 0%     596.0 ± 0%    -25.03% (p=0.001 n=7)  allocs/op
```

Reproduz dos brutos, 7 por lado, e bate com a tabela do commit.

**Plausibilidade dos −96%/−98,61%:** sim, e a aritmética fecha. O benchmark
(`internal/writer/bench_analise_test.go:15`) monta uma nota de 200 links com
~100 bytes por parágrafo — ~20 KiB. O laço antigo alocava e copiava a nota
inteira uma vez por link: 200 × ~20 KiB ≈ 4 MiB, que é exatamente o `4073.38Ki`
medido no base. O laço novo aloca uma vez, ~57 KiB, que é o `56.68Ki`. A razão
de B/op (≈72×) e a de tempo (25×) são da mesma ordem, com o tempo menor porque
o custo de `BuildLinkText` (196 das 596 alocações restantes) não some.

Re-run independente na árvore atual, `-count=3`:

```
$ go test -run '^$' -bench '^BenchmarkRewriteLinksMuitos$' -benchmem -count=3 ./internal/writer/
BenchmarkRewriteLinksMuitos-12    19378   61959 ns/op   58041 B/op   596 allocs/op
BenchmarkRewriteLinksMuitos-12    22198   57248 ns/op   58040 B/op   596 allocs/op
BenchmarkRewriteLinksMuitos-12    21673   58180 ns/op   58040 B/op   596 allocs/op
```

58040 B = 56,68 KiB e 596 allocs batem com o lado "depois" do relatório na
casa exata. O `ns/op` sai ~7% acima do 54,19 µs medido lá, o que é ruído de
máquina no mesmo intervalo (±9% no lado depois) — não contradição.

---

## Candidatos 2, 4 e 5 — não entraram

Tabela completa colada no relatório para os três, ≥7 amostras por lado. Recalculei
os três dos brutos e os três reproduzem exatamente o que está escrito
(`c2`: `p=0,620` sec/op, `allocs/op` "all samples are equal"; `c4`: `p=0,128`
sec/op, `-0,27%`/`-0,36%` a `p=0,001`; `c5`: `p=0,710`/`0,902`/`1,000`).

**Árvore limpa.** `git diff a6521d1..HEAD --stat` toca três arquivos:
`internal/search/inverted.go`, `internal/writer/linkrewrite.go` e
`task-170-report.md`. `git status --short internal/` não devolve nada. Nenhum
resto de 2, 4 ou 5.

**A alegação de "o benchmark não executa o bloco" (candidatos 2 e 5) confere, e
eu a verifiquei por outro caminho além do perfil de cobertura:**

- Candidato 2: o relatório dá o comando
  (`go test -run '^$' -bench '^BenchmarkParseNotaLonga$' -benchtime=1x
  -coverprofile=...`) e cola os blocos com contagem 0. Independentemente disso,
  li o corpus: `internal/parser/bench_analise_test.go:14` gera
  `[md](Nota%d.md)` — link Markdown **com** texto —, e o ramo alvo
  (`internal/parser/ast.go:322`) é o `if startIdx == -1`, que só roda para
  `"[](alvo.md)"`. O bloco é inalcançável por esse benchmark, ponto.
- Candidato 5: o relatório cola os blocos com contagem 0 mas **não** cola o
  comando (o de c2 está lá, o de c5 é só descrito). Verifiquei na fonte:
  `BenchmarkSearchFiltroFrontmatter` (`internal/service/bench_cache_test.go:100`)
  passa `Frontmatter` e nenhum `Tags`, e o bloco alvo
  (`internal/service/search.go:441`) é guardado por `if len(opts.Tags) > 0`.
  Confere. Fica como observação de forma, não achado.

**Candidato 4 (só memória, `sec/op` `~`): concordo com a decisão de não
entrar.** Duas APIs exportadas a mais cuja pré-condição falha em silêncio, por
20 alocações em 5583, contra a métrica que o RNF-04 nomeia. Precedente da Task
82 aplicado corretamente.

---

## Outras restrições globais

- **`CacheFormatVersion` intacto** — `internal/search/persist.go:22` continua
  `6`, e `persist.go` nem aparece no diff. O mapa reverso vive só no delta, que
  não é serializado; `ExportForCache` não o consulta.
- **Sem `net/*`** — o diff acrescenta um import só, `bytes`, em
  `linkrewrite.go`.
- **Sem tipos do SDK MCP fora de `mcpsrv`** — nada perto disso.
- **Sem aresta de import nova** — `bytes` é stdlib; o grafo de
  `internal/` não muda.
- **Conventional Commits em inglês, com o benchstat no corpo** — os dois
  commits têm mensagem em inglês, escopo correto (`perf(search)`,
  `perf(writer)`) e as três tabelas (`sec/op`, `B/op`, `allocs/op`) coladas no
  corpo, com `p` e `n`. O de `8e0670c` declara o custo de memória e afirma que
  `CacheFormatVersion` não muda — o que confirmei.
- **`docs/bench-baseline.json`** — `grep -c` dos dois nomes devolve 0, então os
  dois ganhos não deixam número obsoleto lá. Confirmado.
- **Baseline do brief que não reproduz** — o relatório registra 3,936 s medidos
  contra os 6,549 s do brief para `InvertedUpdateLote`, e diz que não
  investigou. Correto assim: toda decisão saiu de comparação lado a lado feita
  nesta máquina, e o número do brief não é reusável entre máquinas. **Não é
  achado contra o implementador**; é aviso para quem escrever o próximo brief
  de desempenho — baseline de brief precisa vir com a máquina, ou não vir.
- **Timestamps do Progresso** — batem com o relógio real. `git log` dá
  05:59:27, 06:06:22, 06:31:00 e 06:31:24; o log vai de 05:53 a 06:31, e os
  mtimes dos brutos e binários em `%LOCALAPPDATA%\gobsidian-bench\2026-09-06\`
  (05:56 a 06:17) caem nos intervalos declarados. Uma única linha destoa — N4.

---

### Achados

**N1 — `task-170-report.md:316-319` (e o brief) nomeiam como rede de correção um
teste que não pode falhar com esta mudança. (should-fix)**
`TestIndiceRecarregadoEIdenticoAoConstruido` (`internal/search/persist_test.go:500`)
só faz `Add`; nunca chama `Remove`, então nunca exercita o laço reescrito de
`removeLocked` com `termosDoDoc` não vazio. Apague o corpo novo do laço e esse
teste segue verde. O relatório herdou a frase do brief e a repetiu sem conferir,
que é exatamente "não afirme estado que você não verificou". **Não é
bloqueante** porque a rede real existe, é forte e passa — trocar a citação por
`TestInvertedRemoveLeavesNoEmptyPosting` (`internal/search/inverted_test.go:14`)
e `TestInvertedRemoveAndRecreateNoResidue` (`:142`) resolve, sem tocar em
código. Fica registrado aqui para o ledger, e vale como correção do brief.

**N2 — `internal/search/inverted.go:246-249`: guarda inalcançável e
semanticamente vazia. (nit)**
`docs, ok := ix.terms[term]; if !ok { continue }` tem contagem 0 no perfil de
cobertura, e é equivalente a não ter guarda nenhuma: em Go, `delete` sobre mapa
`nil` é no-op e `len(nil) == 0` faria o `delete(ix.terms, term)` seguinte também
ser no-op. Ou seja, remover as três linhas não muda comportamento em nenhum
caso, alcançável ou não. Mantê-la é defensável como documentação da hipótese;
apagá-la é defensável como "não deixe deliberação no código". Não peço mudança.

**N3 — `internal/search/inverted.go:750`: a guarda de "índice vazio" de
`AdotarDe` não inclui o mapa novo. (nit)**
Ela testa `len(ix.terms) > 0 || len(ix.docLengths) > 0 || ix.base != nil` e não
`len(ix.termosDoDoc) > 0`. Hoje isso é inofensivo — pela invariante,
`termosDoDoc` não vazio implica `terms` não vazio, então a guarda pega o caso —
mas a guarda existe justamente para o mundo em que uma invariante quebrou, e é o
único lugar onde o mapa novo é substituído em bloco. Uma quarta cláusula custa
uma linha.

**N4 — `task-170-report.md:9`: a linha de `06:04` nomeia um SHA que só passou a
existir às `06:06:22`. (nit)**
O commit `6ec12da` tem `CommitDate 2026-09-06 06:06:22 -0300`. O carimbo é de
relógio real (o resto do log bate), mas a linha foi carimbada antes de o SHA que
ela cita existir. Duas linhas — uma para a medição, outra para o commit — evitam
a discrepância.

**N5 — `internal/writer/linkrewrite_test.go`: nenhum caso fixa link em offset 0
nem link terminando exatamente no EOF. (nit)**
São as duas fronteiras que a passada única introduz (`src[0:0]` na primeira
escrita e `src[len:]` na última). Provei por leitura que ambas são fatias vazias
e inofensivas, e o benchmark de 200 links não passa por nenhuma das duas
(a nota começa com "Paragrafo" e termina em `\n\n`). Um teste com
`"[[a]] meio [[b]]"` — link no byte 0 e link colado no fim — fecharia as duas de
uma vez. Sugestão, não exigência.

---

## Veredito

Os dois commits que entraram fazem o que a mensagem diz, preservam
comportamento, e trazem número medido e reproduzível — recalculei os cinco
`benchstat` dos arquivos brutos e todos batem com o que está escrito, o que é a
diferença entre um relatório e uma alegação. Os três candidatos recusados saíram
com tabela completa e árvore limpa, e em dois deles o motivo (o benchmark pareado
não executa o bloco) foi confirmado por mim de forma independente da que o
relatório usou. Os cinco achados são de relatório e de estilo; nenhum toca a
correção do código.

Spec: APPROVED
Quality: APPROVED

---

## Round 1 — re-revisão escopada de `5a1c320`

Escopo: `review-fix1-dcae200..5a1c320.diff` (102 linhas) e a seção
`## Fix round 1` do relatório. Um commit,
`test(search,writer): name the tests that actually guard removeLocked, pin the
rewrite boundaries`, tocando três arquivos: `internal/search/inverted.go`,
`internal/writer/linkrewrite_test.go` e `task-170-report.md`.
`git status --short internal/` vazio.

### Progresso

- 06:51 — `git log`/`--stat` de `dcae200..HEAD`; diff de correção lido inteiro.
- 06:53 — seção `## Fix round 1` lida; âncoras das mutações conferidas na fonte.
- 06:55 — `go test -race -count=1 -run TestRewriteLinks -v ./internal/writer/`;
  linhas citadas nas saídas de mutação conferidas contra
  `internal/search/inverted_test.go`.
- 06:57 — `go test -race -count=1` nos dois pacotes e `go vet`: verde.

### N1 — endereçado

A citação foi corrigida em `task-170-report.md:321-324`
(`TestInvertedRemoveLeavesNoEmptyPosting` + `TestInvertedRemoveAndRecreateNoResidue`)
e o parágrafo **conserva a frase errada e o motivo** em vez de apagá-los
(`:327-332`) — que é a forma certa de corrigir um relatório.

**As três provas de mutação atacam o laço reescrito.** A âncora é
`\t\tdelete(docs, path)\n`, que existe hoje em `internal/search/inverted.go:250`
— dentro do `for _, term := range ix.termosDoDoc[path]` introduzido por
`8e0670c`, e não no laço antigo. Apagá-la é exatamente "remover a remoção que o
índice reverso passou a dirigir".

Não rodei `mutate.ps1` (só edito meu arquivo de revisão), então auditei as
saídas coladas pelo único jeito que não depende de confiança: **os números de
linha que elas citam têm de bater com a fonte.** Batem, os quatro:

| Saída colada | `internal/search/inverted_test.go` |
|---|---|
| `inverted_test.go:23` "postings de termo compartilhado = …, quer so b.md" | linha 23, `t.Errorf("postings de termo compartilhado = %+v, quer so b.md", got)` |
| `inverted_test.go:29` "termo orfao continua no dicionario…" | linha 29, idem |
| `inverted_test.go:32` "TermCount = 3, quer 2 — termo orfao esta sendo contado" | linha 32, `t.Errorf("TermCount = %d, quer 2 — …", n)` |
| `inverted_test.go:150` "termos da nota antiga vazaram apos recriacao: t1=true, t2=true" | linha 150, idem |

E o conteúdo é o previsto pela mutação: sem `delete(docs, path)`, `a.md`
sobrevive nas postings do termo compartilhado, o termo órfão fica no
dicionário e `TermCount` sobe de 2 para 3 — os três erros que a saída mostra,
na ordem. O terceiro caso mostra o teste que o brief nomeava **passando**
(`ok … 1.618s`, `MUTATE_EXIT=1`), com o `mutate.ps1` acusando "esta escrita,
nao verificada". É a prova que N1 pedia, e ela vem no formato que o
`ARMADILHAS.md` exige: no passado, com a saída colada.

### N3 — endereçado

`internal/search/inverted.go:749-754`: quarta cláusula `len(ix.termosDoDoc) > 0`
na guarda e a contagem na mensagem de erro. O comentário registra que ela é
redundante pela invariante e por que existe assim mesmo. O teste que exercita a
recusa (`inverted_test.go:201-203`) compara com `errors.Is`, não com o texto,
então a mensagem mais rica não quebra nada — conferi.

### N4 — endereçado

A linha virou duas (`task-170-report.md:9-10`): medição às 06:04, commit
`6ec12da` às 06:06, que é o `CommitDate` real.

### N5 — endereçado, e o teste faz o que N5 pediu

`TestRewriteLinks_LinkNoInicioENoFim` (`internal/writer/linkrewrite_test.go:52-91`).
Conferi as três coisas que importavam:

1. **Fixa a saída byte a byte** — `want := "[[pasta/um]] meio [[pasta/dois]]"`
   comparado com `string(got)`. Não é asserção de tamanho nem de "contém".
2. **As duas fronteiras estão de fato no caso** — `input := "[[a]] meio [[b]]"`,
   sem texto antes do primeiro link nem depois do último. E o teste **não
   presume** isso: falha com `t.Fatalf` se `note.Links[0].Start != 0` ou
   `note.Links[1].End != len(src)`. Essa guarda é o que impede o teste de ficar
   verde medindo outra coisa caso o parser mude os offsets — era precisamente o
   risco de um teste de fronteira escrito por cima do parser.
3. **Passa**, junto com os cinco anteriores:

```
$ go test -race -count=1 -run 'TestRewriteLinks' -v ./internal/writer/
--- PASS: TestRewriteLinks_PreservesAliasAndAnchor (0.00s)
--- PASS: TestRewriteLinks_PreservesSyntaxAndEmbed (0.00s)
--- PASS: TestRewriteLinks_MultipleOccurrencesInSameNote (0.00s)
--- PASS: TestRewriteLinks_LinkNoInicioENoFim (0.00s)
--- PASS: TestRewriteLinks_RejectsInvalidOffsets (0.00s)
--- PASS: TestRewriteLinks_PreservesBOMAndEOL (0.00s)
ok  	github.com/jonyd/gobsidian/internal/writer	1.654s
```

A mutação colada (`cursor := int64(0)` → `int64(1)`, âncora presente em
`internal/writer/linkrewrite.go:74`) ataca o laço da passada única, e o pânico
`slice bounds out of range [1:0]` em `linkrewrite.go:76` só é alcançável com
`Start == 0` — ou seja, o sintoma exibido é o do caso novo, não o dos antigos.
O relatório **declara por conta própria** que a mesma mutação também derruba os
outros `TestRewriteLinks_*` e que o valor do teste novo é a fronteira, não a
exclusividade. Essa ressalva não foi pedida e é o oposto de inflar a prova.

### N2 — não endereçado, e está certo assim

A rodada deixou o `if !ok { continue }` como estava. Foi nit explicitamente sem
pedido de mudança da minha parte; nada a fazer.

### Fora da lista: o `</content>` no fim do relatório

O implementador achou e removeu sozinho uma linha `</content>` que sobrara do
próprio `Write` no relatório commitado, e registrou o achado como dele
(`task-170-report.md:387`). Conferi: `tail -3` do arquivo termina em prosa, sem
a tag. Achar e declarar o próprio lixo, em vez de esperar o revisor achar, é o
comportamento que este projeto quer.

### Verificações minhas desta rodada

```
$ go test -race -count=1 ./internal/search/ ./internal/writer/
ok  	github.com/jonyd/gobsidian/internal/search	14.960s
ok  	github.com/jonyd/gobsidian/internal/writer	17.848s

$ go vet ./internal/search/ ./internal/writer/
[vet OK]
```

O commit não toca caminho quente — a cláusula nova de `AdotarDe` roda uma vez
por adoção de cache — então não pedi nova medição, e o relatório justifica a
ausência dela pelo mesmo motivo. Continua sem `net/*`, sem tipo do SDK fora de
`mcpsrv`, sem aresta de import nova, `CacheFormatVersion` intocado. A saída do
`verify.ps1 -SkipCross -SkipNet` colada no relatório fica como afirmação do
implementador; o despacho me proíbe de rodá-la.

### Veredito da rodada

Quatro dos cinco achados endereçados, o quinto não exigia ação. As três provas
de mutação são reais, atacam o código reescrito e uma delas demole a afirmação
que o próprio relatório fazia na rodada anterior. O teste de fronteira fixa a
saída byte a byte e se protege contra virar verde por acidente. Nada novo a
apontar.

Spec: APPROVED
Quality: APPROVED
