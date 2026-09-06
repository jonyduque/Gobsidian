# Revisão da Task 169 — `2307cf3..383fd90`, 11 commits

Revisor: rev-169. Somente leitura da árvore; nada foi editado além deste
arquivo. As mutações que eu precisei rodar foram feitas numa **cópia** do
código em `%TEMP%\claude\...\scratchpad\mut169` (só `go.mod`, `go.sum`,
`internal/`, `cmd/`, `testdata/`), nunca no repositório.

## Progresso

Duas horas medidas, e só duas: rodei `date +%H:%M` no começo e no fim. As
etapas intermediárias **não têm hora medida** e por isso vão sem hora — inventar
uma progressão seria exatamente o defeito que o relatório desta task confessa.

- **05:28 [relógio]** — Li o brief, o relatório, `papeis/revisor.md`,
  `papeis/desempenho.md` e o pacote de revisão inteiro.
- Conferi a lista de commits, o `--stat` do golden em `6227757` e o `git diff`
  do golden entre `4f3eb4c` e `6227757`.
- Reextraí o grafo de imports de produção com `GOOS=windows go list` e comparei
  com o bloco novo do `CLAUDE.md`.
- `go build ./...`, `go vet` em windows/linux/darwin, `gofmt -l`,
  `go clean -testcache` e `go test -race` em `index`, `writer`, `service` e
  `mcpsrv`.
- Reproduzi **todas** as nove tabelas de `benchstat` a partir dos arquivos de
  amostra brutos, e reproduzi os ganhos de fora com `-count=3` contra os
  binários `antes_*`.
- Provei a não-vacuidade do golden **eu mesmo**, com duas mutações, e a do teste
  de NFC/NFD com uma terceira.
- Medi o custo da preocupação 1 do implementador (chave de trava), que o
  relatório declara "não medido".
- **05:49 [relógio]** — Escrevendo este arquivo.

---

## 1. Evidência de `benchstat`

Todas as tabelas do relatório foram **reproduzidas por mim** a partir dos
arquivos de amostra em `%LOCALAPPDATA%\gobsidian-bench\2026-09-02\`, rodando
`benchstat` de novo. Contagem de amostras por arquivo, medida com
`grep -c '^Benchmark'` (arquivos com dois benchmarks têm 14 linhas para 7
amostras por benchmark):

```
t169_i1  : 7/7      t169_i2 : 14/14    t169_i3 : 14/14    t169_i3b: 12/12
t169_i4  : 12/12    t169_i4h: 7/7      t169_i5 : 7/7      t169_i6 : 7/7
t169_i7  : 7/7      t169_i9 : 14/14
```

Todo lado tem **≥ 7 amostras**. Item 1, verbatim da minha própria execução:

```
LinkGraphBothDepth2-12   8.746µ ± 3%   3.271µ ± 16%  -62.60% (p=0.001 n=7)   sec/op
LinkGraphBothDepth2-12  2.055Ki ± 0%   1.180Ki ± 0%  -42.59% (p=0.001 n=7)   B/op
LinkGraphBothDepth2-12    41.00 ± 0%     11.00 ± 0%  -73.17% (p=0.001 n=7)   allocs/op
```

Os três números citados na prosa do relatório conferem com as tabelas coladas:
`sec/op −62,60%` (item 1), `−29,79%` (item 3, `TagListHierarquico` sec/op),
`−75,12% B/op` (item 3). Nenhum número do relatório apareceu sem tabela.

Não me contentei com o `benchstat` sobre os arquivos gravados — rodei os
binários de novo, `-count=3`, para ver se a ordem de grandeza é a mesma:

```
$ antes_service.test.exe  -test.bench LinkGraphBothDepth2 -test.count=3
BenchmarkLinkGraphBothDepth2-12   130119   8988 ns/op   2104 B/op   41 allocs/op
BenchmarkLinkGraphBothDepth2-12   143653   8685 ns/op   2104 B/op   41 allocs/op
BenchmarkLinkGraphBothDepth2-12   132880   8816 ns/op   2104 B/op   41 allocs/op

$ depois169_service.test.exe -test.bench 'LinkGraphBothDepth2|TagList' -test.count=3
BenchmarkLinkGraphBothDepth2-12   456193   3239 ns/op   1208 B/op     11 allocs/op
BenchmarkLinkGraphBothDepth2-12   355113   3373 ns/op   1208 B/op     11 allocs/op
BenchmarkLinkGraphBothDepth2-12   468422   3243 ns/op   1208 B/op     11 allocs/op
BenchmarkTagListHierarquico-12       398   2716310 ns/op   268496 B/op   9998 allocs/op
BenchmarkTagListHierarquico-12       477   2595541 ns/op   268496 B/op   9998 allocs/op
BenchmarkTagListHierarquico-12       472   2602657 ns/op   268496 B/op   9998 allocs/op

$ antes_index.test.exe / depois169_index.test.exe -test.bench TotalSizeRepetido -test.count=3
antes :  39.52 / 38.27 / 45.12 ns/op
depois:  21.41 / 24.54 / 21.82 ns/op
```

Bate com as tabelas (2104→1208 B/op, 41→11 allocs, 1053,9Ki→262,2Ki = 268 496 B,
10 879→9 998 allocs, ~40 ns→~22 ns). Nada aqui é número escrito de memória.

**A regressão e a volta ao `~`.** Reproduzidas por mim dos arquivos brutos:

```
i3   TagListPlano-12  19.62µ ± 16%  21.04µ ± 22%  +7.24% (p=0.038 n=7)   ← regressão do item 2
i3b  TagListPlano-12  19.43µ ±  1%  20.44µ ±  4%  +5.22% (p=0.000 n=12)  ← isolada, variância apertada
i4   TagListPlano-12  19.93µ ±  5%  20.47µ ±  5%       ~ (p=0.198 n=12)  ← depois de d7a11e8
```

Nenhuma linha de nenhuma das nove tabelas tem piora com `p < 0,05` no estado
final da árvore.

**Sequência do i9 ("estado final").** Os arquivos `t169_i9_*` são de 05:16:48 e
05:16:59, e o commit da reversão é de 05:17:21 — a medição precede o commit. Não
é um número medido na árvore errada: a mensagem de `383fd90` diz "com a árvore
neste estado" e o binário `depois169_service.test.exe` é de 05:14, isto é, foi
construído da árvore **já revertida** e ainda não commitada. A ordem
medir→commitar é a certa; só vale saber que o arquivo é mais velho que o SHA.

Itens 7 e 8 estão declarados **não medidos**, com o motivo (nenhum benchmark da
bateria passa pelo ramo de alias nem pelo `PathLocker`). É a resposta certa.

---

## 2. Golden `testdata/tag_list_hierarquico.json`

**Criado de código-base.** `git diff --stat 2307cf3 4f3eb4c` toca três arquivos,
e o diff de `graph.go` entre os dois commits **não tem uma única linha com
`tag`** — o item 1 mexe só em `chaveDaAresta`. Logo o golden foi gravado com o
comportamento de `tag_list` do commit base, como o relatório afirma.

**Insersão pura no commit do item 3.** Conferido nos dois comandos que o
orquestrador pediu:

```
$ git show --stat 6227757 -- testdata/tag_list_hierarquico.json
 testdata/tag_list_hierarquico.json | 33 +++++++++++++++++++++++++++++++++
 1 file changed, 33 insertions(+)
```

`git diff 4f3eb4c 6227757 -- testdata/...` é **um único hunk contíguo** (`@@
-138,6 +138,39 @@`), só linhas `+`, inserindo a variante
`hierarquico_prefixo_no_meio` entre `hierarquico_prefixo_proj` e
`hierarquico_min_count_2`. Zero remoções ⇒ as cinco variantes originais são byte
a byte as mesmas. Confirmado.

**Não-vacuidade — provada por mim, não lida no relatório.** Duas mutações, na
cópia do código:

Mutação A, `Children: anyChildren` → `Children: nil` em `convert`
(`internal/service/graph.go:457`):

```
$ go test ./internal/service/ -run TestTagListGolden -count=1
--- FAIL: TestTagListGolden (0.01s)
    tag_list_golden_test.go:109: o JSON de tag_list mudou
        --- esperado ---
        ... "tag": "docs", "count": 3, "children": [ { "tag": "docs/api", ...
```

Mutação B — a alternativa que o brief mandava e o implementador recusou
(filtrar o prefixo já na varredura das notas), inserida depois de
`tagClean := strings.TrimPrefix(tag, "#")`:

```go
if req.Prefix != "" && !strings.HasPrefix(strings.ToLower(tagClean), strings.ToLower(req.Prefix)) {
    continue
}
```

```
$ go test ./internal/service/ -run TestTagListGolden -count=1
--- FAIL: TestTagListGolden (0.01s)
```

Ou seja: o golden pega **as duas** coisas — a perda dos filhos e a contagem do
ancestral que a leitura alternativa do brief apagaria. Restaurado o arquivo, ele
passa; e passa `-count=6`, então é determinístico apesar de `convert` iterar
mapa (a ordenação em `ordenarTags` é total nos dois modos).

A comparação é byte a byte (`if string(got) != string(want)`), sobre um cofre
escrito pelo próprio teste a partir de literais.

---

## 3. Reversão do item 2 (`383fd90`)

**A árvore volta ao equivalente de antes de `06c8640`.** O diff
`2307cf3..383fd90` de `graph.go`, com comentários filtrados, mostra em `TagNode`
e em `ordenarTags` **nenhuma linha de código alterada** — só o comentário novo
no tipo e o comentário da guarda. Isto é: o campo é `[]any` de novo, o
type-assert de `ordenarTags` é o mesmo, o encaixotamento de `convert` é o mesmo,
e a guarda `if len(tags[i].Children) > 0` é literalmente a do código base. Não
há "correção de folha" pendurada que precise ser reavaliada contra `[]any`: ela
é o código base restaurado.

**Nenhum resto de `[]TagNode`.** `grep -rn "\[\]TagNode" --include=*.go .`
devolve nove ocorrências, todas legítimas e todas pré-existentes ao 169
(`Tags []TagNode` do `TagResult`, a assinatura de `ordenarTags`, `childList`,
`nodes`, o `convert`) mais duas citações em comentário que explicam por que o
campo **não** é `[]TagNode`.

**O texto do pânico está no relatório e no código**, verbatim:
`panic: AddTool: tool "tag_list": output schema: ForType(service.TagResult):
computing element schema: computing element schema: cycle detected for type
service.TagNode`. Ele está gravado no comentário de `TagNode`
(`internal/service/graph.go:272-285`), que é onde a próxima limpeza mecânica vai
olhar — decisão certa.

`go test -race ./internal/mcpsrv/` passa no estado final (saída em §7).

---

## 4. Item 3 (`6227757`) — a alegação do implementador é **verdadeira**

O brief mandava "usar `ix.tags` como o ramo plano". O implementador recusou
dizendo que isso muda a resposta da tool. Conferido no código e no dado:

- `internal/index/query.go:173-192` — `Tags(prefix, minCount)` conta
  `len(ix.tags[t])` por tag **exata**. `ix.tags` é povoado em
  `internal/index/index.go:194-196`, um append por tag literal da nota, sem
  rollup de ancestral.
- `internal/service/graph.go:344-346` — o ramo plano é exatamente esse
  `s.index.Tags(...)`.
- `internal/service/graph.go:355-360` (o ramo hierárquico) conta uma nota em
  **todos os prefixos** da tag dela.

O que difere não é frontmatter vs inline, e não é "por nota vs por ocorrência" —
os dois ramos são por nota distinta. O que difere é o **rollup de
descendentes**: no hierárquico uma nota marcada só com `#docs/api` conta em
`docs`; no plano, não. Medido no próprio golden:

```
hierarquico_nome  [('docs', 3), ('proj', 4), ('zeta', 1)]
plano_contagem    [('docs', 2), ('proj/alpha/um', 2), ('proj/beta', 2),
                   ('docs/api', 1), ('proj/alpha', 1), ('proj/alpha/dois', 1), ('zeta', 1)]
```

`docs` vale 3 num ramo e 2 no outro. Somar as contagens exatas também não
serviria: uma nota com `#proj/alpha` e `#proj/beta` conta **uma** vez em `proj`
e a soma contaria duas. Além disso `ix.tags` é campo privado de `index` — de
`service` a única porta é `Tags()`, que já entrega a contagem exata.

A outra metade ("filtrar dentro do loop") também está corretamente recusada, e
não por opinião: minha mutação B acima mostra o golden pegando o efeito
(`proj` cairia de 4 para 2 com `prefix="proj/al"`).

**Veredito: o brief estava errado nesse item, e o escopo reduzido é o certo.**
Ele não encolheu em silêncio — está na mensagem do commit, no relatório e agora
travado por uma variante nova do golden.

O que entrou (dedupe por `map[string]bool` de rascunho + `map[string]int` no
lugar de um `map[CanonicalPath]bool` alocado por tag) preserva a semântica:
`NotePaths()` vem de `for p := range ix.notes` (`query.go:136-146`), portanto
sem repetição, e o `clear(vistas)` por nota faz a dedupe que o mapa por tag
fazia.

---

## 5. Item 4 (`ba9927c`)

- `resultadoVazio`: os **três** literais foram substituídos
  (`search.go:222` era o de zero tokens, `:276` e `:404` os de offset além do
  total). `grep -n "SnippetCharsEfetivo"` em produção devolve quatro sítios: a
  declaração do campo, `resultadoVazio` e os **dois retornos não vazios**, que
  não são duplicação do mesmo fato. Nenhum resto.
- Semântica preservada: os três passam `Total` com o mesmo valor de antes (0 no
  primeiro, `total` nos outros dois), `Truncated: false`, e as fatias continuam
  **não-nil** (`[]SearchHit{}`), que é o que impede `null` no JSON.
- `pagina(offset, limit, total)`: `fim = offset+limit; if fim < total → (fim,
  true); senão (total, false)` é linha a linha o que os dois blocos faziam,
  inclusive no caso `limit == 0`.
- `moveNoteErro`: os cinco sítios foram substituídos. Os dois primeiros
  (MkdirAll e `moverCorpo`) não citavam `Rewritten`/`LinksUpdated`; agora citam,
  mas a closure lê `rewrittenList == nil` e `linksUpdatedCount == 0` nesse
  ponto — os mesmos valores zero. Os códigos de erro são os mesmos objetos `err`
  passados como argumento; nenhum foi reescrito.
- `make([]hitComNota, 0, len(rawHits))`: o laço só filtra, o teto é exato.

---

## 6. Item 5 (`1456a5d`)

```
$ grep -rn "%016x" --include=*.go .
./internal/service/write.go:29:  return fmt.Sprintf("%016x", h)          ← formatarHash
./internal/service/write_test.go:243
./internal/service/write_test.go:384
```

Um único sítio de produção, dentro de `formatarHash`. `hashDoConteudo` a chama
(`write.go:33`), e os três `fmt.Sprintf("%016x", n.Hash)` de `read.go:404`,
`graph.go:481` e `graph.go:583` viraram `formatarHash(...)`. `fmt` saiu do
import de `read.go`. Os dois restos em `write_test.go` são expectativas de teste
computadas por fora, o que é o certo — um teste que chama a função sob teste
para calcular o esperado não testa nada.

---

## 7. Item 6 (`62783cc`) — `TotalSize`

O double-check está **correto**:

- A memorização é invalidada só por `generation`, que é `atomic.AddUint64` em
  cinco sítios (`index.go:153`, `update.go:82,166,178,490`). Não existe caminho
  que ponha `tamanhoValido = false` — a validade é a igualdade de geração.
- O acerto lê `tamanhoValido/tamanhoGeracao/tamanhoTotal` sob `RLock` e copia o
  `int64` para uma local **antes** do `RUnlock`. Sem leitura fora do lock.
- No miss, a segunda conferência sob `Lock` usa a **mesma** `geracao` capturada
  no início, e é ela quem decide — é o que evita recalcular por cima de um
  preenchimento concorrente.
- Valor obsoleto: um leitor que capture a geração `G` e só depois tome o `RLock`
  pode devolver o total de `G` mesmo com o índice já em `G+1`. Isso é um
  instantâneo consistente de um estado que existiu, e é **idêntico ao
  comportamento do código base**, que também lia `generation` antes de travar. A
  mudança não introduz obsolescência nova.
- O caso feio que sobra é benigno: se a geração avançou entre o `RUnlock` e o
  `Lock`, o total recalculado (do estado atual) é gravado com o rótulo `G`,
  antigo. Como a geração só cresce, ninguém lê esse rótulo com sucesso depois —
  o custo é um recálculo a mais, nunca um número errado. Também é o
  comportamento do código base.

```
$ go clean -testcache && go test -race ./internal/index/ ./internal/writer/ ./internal/service/ ./internal/mcpsrv/
ok  github.com/jonyd/gobsidian/internal/index    3.371s
ok  github.com/jonyd/gobsidian/internal/writer  17.034s
ok  github.com/jonyd/gobsidian/internal/service 46.671s
ok  github.com/jonyd/gobsidian/internal/mcpsrv   5.889s
```

---

## 8. Item 7 (`9bc99ba`) — `vivosLocked`

O lock **está tomado no sítio da chamada**: `ResolvePath` abre com
`ix.mu.RLock(); defer ix.mu.RUnlock()` (`resolve.go:368-369`) e o ramo de alias
é `resolve.go:417-418`, dentro do mesmo escopo. `vivosLocked`
(`resolve.go:249`) documenta "Exige ix.mu ja tomado" e o outro chamador do mesmo
corpo (`resolve.go:381`) está sob o mesmo lock.

Diferença de comportamento: a cópia removida aceitava só `ix.notes`;
`vivosLocked` aceita `ix.notes` **ou** `ix.assets`. Hoje isso é inalcançável —
`byAlias` só recebe `n.Path` de nota (`index.go:208`, dentro de
`publishNoteLocked`) e caminho de nota termina em `.md`. Ver achado N4.

---

## 9. Item 8 (`f91df07`) — `normalizeKey`

**Executado como o orquestrador decidiu.** `text.ChaveDeCaminho` guarda
`strings.ToLower(ParaNFC(filepath.ToSlash(path)))`; `index.chaveDeCaminho`
delega; `writer.normalizeKey` a chama.

**`text` continua folha**, conferido e não afirmado:

```
$ GOOS=windows go list -f '{{.ImportPath}}: {{.Imports}}' ./internal/text/
internal/text: [golang.org/x/text/runes golang.org/x/text/transform
                golang.org/x/text/unicode/norm path/filepath strings sync unicode]
```

Nenhum import de `internal/*`. `path/filepath` é stdlib.

**O bloco do grafo do `CLAUDE.md` confere com os imports**, reextraído por mim
pacote a pacote:

```
config:      console:     lifecycle:   text:        vault:
parser:  text                    ipc:     config
writer:  parser text vault       index:   parser text vault
search:  index parser text vault watcher: index search vault
service: index parser search vault writer
mcpsrv:  config index parser service vault
daemon:  config ipc mcpsrv       doctor:  config daemon ipc vault
vaulttest: vault
```

Linha por linha igual ao bloco novo, inclusive `writer → parser, text, vault`.
A justificativa da aresta ("uma conta por regra", com o defeito concreto) está
escrita logo abaixo do bloco, no mesmo commit.

**O RED do relatório é verdadeiro.** Reproduzi na cópia, revertendo
`normalizeKey` para `strings.ToLower`:

```
=== RUN   TestChaveDaTravaUneAsDuasGrafias
    lock_unicode_test.go:36: chaves diferentes para a mesma nota:
         NFC -> "ação.md"
         NFD -> "ação.md"
    lock_unicode_test.go:60: a grafia NFD pegou a trava enquanto a NFC a segurava; sao a mesma nota
--- FAIL: TestChaveDaTravaUneAsDuasGrafias (0.00s)
```

Mesmas duas asserções, mesmas linhas 36 e 60 do relatório. O teste cobra a conta
**e** a porta pública (`Lock` de uma grafia bloqueia a outra), e ainda começa
conferindo que as duas constantes diferem — sem isso ele passaria vazio.

**Preocupação 1 do implementador: minha resposta é "não precisa de benchmark", e
agora com número.** Medi na cópia, comparando a conta antiga com a nova sobre um
caminho típico de cofre (`Projetos/Direito/Acordaos/2026/Acordao Relevante.md`),
`-count=5`:

```
BenchmarkChaveAntiga-12   179.2 / 174.1 / 171.3 / 162.1 / 159.9 ns/op   64 B/op  1 allocs/op
BenchmarkChaveNova-12     213.2 / 220.4 / 257.1 / 257.9 / 214.0 ns/op   64 B/op  1 allocs/op
```

~+60 ns e **zero alocação a mais** por aquisição de trava. `PathLocker.Lock` é
tomado uma vez por chamada de tool de escrita (e uma por nota citante no
`MoveNote`), e cada uma dessas travas envolve `ReadFile` + `WriteAtomic` com
`sync` — centenas de microssegundos a milissegundos. Sessenta nanossegundos é
ordem de 10⁻⁴ do que a trava protege. **"Não medido" era aceitável; agora está
medido e o veredito é: não abra tarefa de benchmark para isto.**

---

## 10. Honestidade dos timestamps

O relatório abre com a correção, em texto claro, antes da lista: as horas das
primeiras linhas foram **digitadas de memória**, o relógio marcava 04:52 quando
os chutes já iam em 06:30, e as linhas inventadas foram **removidas** em vez de
corrigidas para um palpite melhor. Só as marcadas `[relógio]` têm hora. É
exatamente a conduta que a regra pede ("não medido" é a resposta certa), e as
horas dos commits (`git log --date=format:%H:%M:%S`) corroboram a ordem narrada.
Sem achado.

## 11. Dica do `gopls modernize` em `internal/index/resolve.go:313`

**Não é linha nova desta task.** `git log -L 308,318:internal/index/resolve.go`
aponta `da1a382` (2026-07-28, "feat(index): obsidian-order link resolution").
O `if len(originParts) < maxLen { maxLen = ... }` é pré-existente, fora do
diff `2307cf3..383fd90`. Não é achado da 169; se virar tarefa, é de limpeza
mecânica geral.

---

## 12. Restrições globais

| Restrição | Estado |
|---|---|
| JSON de saída inalterado | golden byte a byte + mutação; `[]any` restaurado |
| Ordem de resultados inalterada | ordenações não tocadas (ver N3, pré-existente) |
| Códigos de erro inalterados | `moveNoteErro` recebe o `err` pronto |
| Uma conta por regra | `deAresta`, `resultadoVazio`, `pagina`, `moveNoteErro`, `formatarHash`, `ChaveDeCaminho` |
| Nenhum número não medido | nove tabelas reproduzidas; itens 7 e 8 declarados não medidos |
| stdout / JSON-RPC | nenhum `fmt.Print*` novo |
| Nenhum `net/*` | `text` e `writer` sem import de rede |
| Tipos do SDK fora de `mcpsrv` | nenhum; a reversão é justamente por causa da fronteira |
| Sem `helpers.go`/`utils.go`/`common.go` | nenhum arquivo novo desse tipo |
| Aresta nova só `writer → text` | conferido com `go list` |
| Conventional Commits em inglês | 11 assuntos, todos conformes |
| `verify.ps1` verde | última linha colada; **não reexecutei** (proibido nesta revisão). Rodei por fora: `go build ./...`, `gofmt -l` (vazio), `go vet` em windows/linux/darwin (limpo), `go test -race` em quatro pacotes (verde) |
| Árvore sem sobra não commitada | `git status --porcelain` não mostra nenhum `.go` modificado |

---

## Achados

**N1 — should-fix — `docs/ESTRUTURA.md:183-192`.** A árvore de `testdata/` é
declarada autoritativa e lista `parser/`, `vault_small/` e `parity/`. O golden
novo, `testdata/tag_list_hierarquico.json`, é o **único arquivo solto na raiz de
`testdata/`** (`git ls-files testdata | awk -F/ 'NF==2'` devolve só ele) e não
aparece na árvore. `CLAUDE.md` manda a mudança atualizar `docs/` no mesmo PR.
Uma linha resolve. O caminho veio do brief, então a escolha do lugar não é
defeito do implementador — a documentação não acompanhada é.

**N2 — nit — `06c8640`, `4f3eb4c..d7a11e8`.** Três commits no meio do intervalo
produzem uma árvore em que `mcpsrv.New` entra em pânico na registração da tool,
isto é, o servidor não sobe. O estado final está correto e a reversão é a
conduta certa (a regra proíbe `reset`), mas `git bisect` nesse intervalo vai
encontrar commits que não sobem. Vale a nota no ledger, não uma mudança.

**N3 — nit — `internal/service/graph.go:248-256`.** A ordenação das arestas
compara só `Source`, `Target` e `Kind`, com `slices.SortFunc`, que **não é
estável**. Duas arestas que diferem apenas em `Alias`/`Anchor` — exatamente as
que a chave passou a distinguir — empatam, e a ordem relativa delas no JSON
varia entre execuções. **Isso é anterior à 169** (a chave já carregava alias e
âncora antes), e o item 1 não piorou nada; mas ele encostou no ponto, então
registro: o desempate por `Alias` e `Anchor` fecharia a não-determinação. Fora do
escopo desta task.

**N4 — nit — `internal/index/resolve.go:410-421`.** O item 7 troca um filtro
"só nota" por `vivosLocked`, que aceita nota **ou anexo**. Hoje é inalcançável, e
concordo com o raciocínio do comentário — mas ele repousa em dois invariantes
que nenhum teste guarda: que `byAlias` só recebe caminho de nota
(`index.go:208`) e que a remoção limpa `byAlias` antes de tirar o caminho de
`notes` (`update.go:234-244`). Se um dia um anexo ganhar alias, este ramo passa a
resolver alias para anexo em silêncio. Comentário já registra a diferença;
sugestão é só de teste, se e quando alguém mexer em alias de anexo.

**N5 — nit — `task-169-report.md:163-165`.** A saída do FAIL do golden cita
`tag_list_golden_test.go:102`; no HEAD o `t.Errorf` está em `:109`. Não é
evidência falsa — conferi que em `4f3eb4c` o arquivo tinha 104 linhas, então
`:102` é coerente com a prova ter sido rodada naquele commit, e a mesma mutação
falha hoje em `:109` (reproduzi). Só falta o relatório dizer **em que commit** a
prova rodou; uma linha resolveria a dúvida sem obrigar o revisor a datar o
arquivo.

Nenhum achado bloqueante.

---

## Veredito

O relatório é honesto no ponto em que mais teria a ganhar mentindo: ele declara
uma regressão que ele mesmo causou, mede-a duas vezes, desfaz, e declara uma
reversão inteira de item por um defeito que só o gate pegou. As nove tabelas de
`benchstat` reproduzem exatamente dos arquivos brutos, os ganhos reproduzem
rodando os binários de novo, o golden é não-vacuo por duas mutações minhas, o
RED do teste de NFC/NFD reproduz linha por linha, e o grafo do `CLAUDE.md`
confere com `go list`. A única coisa que o brief pedia e não foi feita é a que o
brief pedia errado, e isso está provado com dado, não com argumento.

**Spec: APPROVED**
**Quality: APPROVED**
