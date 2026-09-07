# Revisao final do ramo — Tasks 182-185

BASE `d91b2fb` -> HEAD `ed24393`, 7 commits. Revisao SOMENTE LEITURA: nenhum
arquivo do repositorio foi criado, alterado ou apagado por esta revisao, exceto
este relatorio. Nenhum `git checkout/restore/stash/clean/reset`, nenhum
`go mod tidy`, nenhum `scripts/test_orphans.ps1`, nenhum subagente.

## Progresso

- 22:33 inicio; listagem do pacote de revisao e leitura do plano
  `docs/superpowers/plans/2026-09-06-broken-links.md` (Global Constraints e
  Tasks 182-185).
- 22:33 `go build ./...` e `go vet ./...` — os dois exit 0.
- 22:34 `go test` dos cinco pacotes tocados — 5/5 ok. Grep de segunda separacao
  de `#` — uma so ocorrencia.
- 22:34 leitura de `internal/parser/ast.go` (`splitAnchor`),
  `internal/parser/ext_wikilink.go` (`splitWikilink`) e
  `internal/service/broken.go`.
- 22:35 `GOOS=windows go list` dos cinco pacotes; comparacao com o grafo de
  `CLAUDE.md`. Leitura das 7 mensagens de commit.
- 22:35 leitura de `internal/index/resolve.go` (`resolveTarget` e os tres
  chamadores), `internal/index/persist.go` (as duas versoes e o caminho de
  load), `internal/writer/linkrewrite.go` (`BuildLinkText`, `anchorMarkdown`).
- 22:36 leitura de `internal/service/write.go:470-700` (`MoveNote` completo,
  ramo `DryRun`, laco de reescrita, `DeleteNote`) — o KNOWN RISK.
- 22:36 leitura dos quatro relatorios de revisao anteriores; verificacao, um a
  um, de que as correcoes pedidas entraram.
- 22:37 contagem real das tools registradas; conferencia de numeros em
  `docs/TOOLS.md`, `docs/ESTADO.md`, `docs/wiki/`, `README.md`.
- 22:37 `gofmt -l`, grep de `net/*`, `check_tool_params.ps1`,
  `check_doc_refs.ps1`, `check_readme_anchors.ps1` — todos verdes.
- 22:38 leitura dos testes novos (`broken_test.go`, `tools_read_test.go`,
  diffs de `parser`, `writer`, `service/write.go`); redacao do relatorio.
- 22:38 fim.

## Plan alignment

| Item do plano | Estado |
|---|---|
| T182 — `splitAnchor` unica, antes do percent-decode, compartilhada com `splitWikilink` | **Feito.** `internal/parser/ast.go:100-105`; `splitWikilink` (`ext_wikilink.go:120-127`) a chama e fica so com o `\|` e o aparo de espaco. |
| T182 — `resolveTarget` recebe a ancora; alvo vazio + ancora resolve para a origem | **Feito.** `internal/index/resolve.go:114-132`, `ViaPath`, `LinkOK`, e cai na checagem de ancora do chamador. Os TRES chamadores passam a ancora (`resolve.go:97`, `update.go:271`, `update.go:431`) — a condicao nao foi copiada. |
| T182 — `BuildLinkText` reemite a ancora | **Feito.** `internal/writer/linkrewrite.go:95-99` + `anchorMarkdown:130-136`, com o mesmo `encodeMarkdownTarget` do alvo. |
| T182 — `IndexCacheParserVersion = 2` | **Feito e efetivo.** `persist.go:49`; o load compara em `persist.go:161` e devolve `ErrIndexCacheVersionMismatch`. `IndexCacheFormatVersion` continua 5, corretamente (o layout nao mudou). |
| T183 — `service.BrokenLinks` com `Total` antes de offset/limit, ordem `Source`+`Start`, `external` fora, `ComTeto`, `ValidarEnum` | **Feito.** `internal/service/broken.go`. `total := len(achados)` acontece depois dos dois filtros e antes da pagina (`broken.go:126`). |
| T183 — registro em `mcpsrv`, tool 13 -> 14 | **Feito.** `tools_read.go:289-311`; contagem real de registros: 14 nomes unicos (13 em `tools_read.go`/`tools_write.go` + `vault_stats` em `server.go`). |
| T183 — schema x comportamento | **Confere.** As quatro descricoes de `vaultBrokenLinksInput` (`tools_read.go:413-418`) dizem o que o codigo faz, inclusive a semantica incomoda de `prefix` (prefixo de STRING, `sub` casa `subtotal.md`) e `total` antes de offset/limit. |
| T184 — docs | **Feito**, com um desvio: ver F2. |
| T185 — `caminhoAtual` como decisao unica | **Feito.** `write.go:645-657`; a variavel serve trava, leitura e escrita (`:660`, `:661`, `:674`, `:680`). |
| Plano: "este plano nao cria nenhuma [aresta de import]" | **Divergiu.** `service -> text` e nova. Ver F2. |

Commits: os 7 estao em Conventional Commits, em ingles, com os dois trailers
exigidos, e cada corpo nomeia o mecanismo do defeito. Nenhum numero sem
medicao: `ESTADO.md` escreve explicitamente "**nao re-medido**" para a queda
de `broken_links` nos cofres reais.

## Findings

### F1 — Important — `internal/service/write.go:545-549`

**O que.** O comentario do ramo `if req.DryRun {` afirma:

> "A origem nao entra em diffs: mover nao altera o conteudo dela, e UnifiedDiff
> de um texto contra ele mesmo e `""` — um item vazio que dizia 'esta nota nao
> muda' sobre a nota que muda de lugar."

As duas afirmacoes sao falsas desde a Task 185 quando a nota movida cita a si
mesma com **alvo escrito**.

**O que o codigo faz de verdade nesse caso** (rastreado, nao suposto):

1. `Backlinks(canonicalFrom)` devolve a propria nota como citante, porque ela
   se cita. O guarda novo de `write.go:505` so descarta `rl.Target == ""` — a
   auto-referencia **so de ancora**. `[[a]]` dentro de `a.md` tem alvo escrito e
   **passa**.
2. Logo `affectedNotes[canonicalFrom]` existe e `totalLinks` ja conta esse link
   — e `totalLinks` e calculado **antes** do `if req.DryRun`, entao
   `LinksUpdated` do dry-run tambem o conta (`write.go:574`).
3. O laco do dry-run (`write.go:556-570`) itera `affectedNotes`, que contem
   `canonicalFrom`, le `s.vault.Abs(canonicalFrom)` — que no dry-run ainda
   existe, nada foi movido — e grava `diffs["a.md"]`. **A origem entra em
   diffs**, sob o caminho ANTIGO.
4. O diff nao e necessariamente vazio. Ele e vazio so quando `newTarget` ==
   alvo escrito, que e o caso do wikilink por nome-base (`[[a]]` -> `newTarget`
   = `toBaseName` = `"a"`). Para um link Markdown, `newTarget =
   string(canonicalTo)`: `[x](a.md)` dentro de `a.md` movida para `sub/a.md`
   produz `[x](sub/a.md)` e um diff **nao vazio** em `diffs["a.md"]`.

**Por que importa.** `docs/TOOLS.md`, secao `note_move`, ja foi corrigida na
Task 184 e hoje descreve exatamente o comportamento certo, incluindo a
assimetria (real: `rewritten` sob o caminho NOVO; dry-run: `diffs` sob o
ANTIGO). O comentario do codigo e o unico lugar que ainda afirma o contrario da
normativa, dentro da propria funcao que produz o comportamento. Regra do
projeto: comentario nao mente, e "nao afirme estado que voce nao verificou". Um
comentario falso ao lado do codigo certo e pior que a ausencia dele, porque a
proxima pessoa a mexer no ramo `DryRun` vai confiar nele.

**Correcao sugerida.** Reescrever o comentario para separar os dois papeis da
origem, que hoje ele funde:

- como **nota movida**, ela nao ganha entrada propria em `diffs` — o move nao
  reescreve o corpo dela por si, e um item vazio diria "esta nota nao muda"
  sobre a nota que muda de lugar; a leitura de `absFrom` continua sendo so a
  validacao de "origem ilegivel falha aqui, e nao so na execucao real";
- como **citante de si mesma com alvo escrito**, ela entra normalmente, sob o
  caminho ANTIGO (o que existe no disco durante o dry-run) — ao contrario da
  execucao real, que a reporta sob o NOVO —, e conta em `links_updated` nos
  dois. Uma auto-referencia so de ancora nao entra em nenhum dos dois, por
  causa do guarda de `write.go:505`.

Nenhuma mudanca de comportamento e necessaria: o codigo esta certo e a
normativa esta certa; e o comentario que ficou para tras.

### F2 — Important — `CLAUDE.md:123` (grafo de dependencias)

**O que.** `internal/service/broken.go:10` importa
`github.com/jonyd/gobsidian/internal/text`. Medido:

```
GOOS=windows go list -f '{{.ImportPath}} {{.Imports}}' ./internal/service
-> ... internal/index internal/parser internal/search internal/text internal/vault internal/writer ...
```

O grafo de `CLAUDE.md:123` continua lendo:

```
service  → index, parser, search, vault, writer
```

Que a aresta e **nova neste ramo** tambem foi medido:
`git grep -l 'internal/text' d91b2fb -- 'internal/service/*.go'` volta vazio.

**Por que importa.** Tres regras do projeto convergem aqui:

- "**Aresta nova precisa de justificativa**" — a aresta nao tem uma escrita.
- O bloco do grafo afirma sobre si mesmo que foi "**re-extraido dos imports de
  producao em 2026-09-06**" — a data de hoje. Um leitor confia que a linha do
  `service` foi conferida hoje, e ela esta errada hoje. O paragrafo abaixo do
  bloco diz, com todas as letras, "dizer 'conferido' nao e conferir, e as duas
  versoes anteriores diziam"; esta e a terceira.
- O plano deste marco, em Global Constraints, afirma "**este plano nao cria
  nenhuma** [aresta]". Criou uma, e a divergencia entre plano e codigo nao foi
  registrada.

A aresta em si esta **certa** e nao deve ser removida: `text.ChaveDeCaminho` e
justamente a conta unica de chave de caminho (a mesma que o `writer` passou a
usar na Task 169), `text` e folha, e o grafo continua aciclico. O defeito e
documental, e o custo do conserto e uma linha mais uma frase.

**Correcao sugerida.** Em `CLAUDE.md`: `service  → index, parser, search, text,
vault, writer`, com uma justificativa no mesmo estilo das de `writer → text` e
`boot → lifecycle` — algo como: *`service → text` e de 2026-09-06 (Task 183); o
filtro `prefix` de `vault_broken_links` compara caminho de origem, e comparar
caminho e `text.ChaveDeCaminho` — a mesma conta do `writer` e do `index`.
Inventar uma comparacao local faria "Sub/" e "sub/" serem duas pastas aqui e
uma la.* Atualizar tambem a nota do plano, se ele ainda for editado.

### F3 — Nit — `internal/service/broken.go:118-124`

O achado 3 da `review-183.md` pedia uma frase no comentario do comparador
dizendo que a ordem por `Source` e **herdada** de `index.NotePaths()` (que ja
devolve `slices.Sort(paths)`) e que o comparador a repete de proposito, para a
garantia ser local em vez de depender de um invariante de outro pacote. O
comentario atual explica **por que** a ordem tem de ser deterministica, mas nao
diz isso. Nenhum teste consegue discriminar a perna `Source`, e o relatorio da
Task 183 provou so a perna `start`. Nao e defeito e nao pede codigo novo — uma
oracao fecha.

### F4 — Nit (registrado, sem acao pedida) — ancora vazia

`[x](b.md#)` perde o `#` na reescrita (`splitAnchor` devolve `("b.md", "")` e
`anchorMarkdown` devolve `""`). Ja esta registrado como divida aberta em
`docs/ESTADO.md`, com a decisao explicita de parquear porque a forma nunca foi
medida em cofre real. Confiro que a divida existe e que o texto e honesto sobre
o que nao foi medido; nada a fazer neste ramo.

**Nenhum achado Blocking.**

## Verified prior-review fixes

| Achado anterior | Estado agora | Onde conferi |
|---|---|---|
| 182 F1 — `note_move` falha com link so de ancora | Corrigido | Guarda `rl.Target == ""` em `write.go:505`, com o mecanismo no comentario |
| 182 F2 — `Resolved` preso ao caminho antigo apos `index.MoveNote` | Corrigido | `internal/index/update.go:507-516`, reescrita de `Resolved` logo apos a copia da nota |
| 182 F3 — `note_delete` acusa a propria nota | Corrigido | `write.go:730-739`, guarda `bl.From == canonical` com o motivo (e por que aqui o discriminante e outro que o de `MoveNote`) |
| 182 F6/F7 | Corrigidos na rodada r1 (fora do escopo deste diff) | `review-182-r1.md` |
| 182 F8 — `[x](b.md#)` | Parqueado com decisao registrada | `docs/ESTADO.md`, dividas abertas |
| 183 #1 — "11 das 14 tools" nao fecha | Corrigido | `docs/wiki/concepts/os-dois-indices.md:32-33` diz **13 das 14** e, logo abaixo, deriva o numero (`s.inverted` aparece so em `search.go`) e nomeia o erro anterior |
| 183 #2 — preambulo **Limites** sem a tool nova | Corrigido | `docs/TOOLS.md:13`: "`note_list`, `link_graph` e `vault_broken_links`: padrao 100, teto 500" |
| 183 #3 — frase sobre a perna `Source` do comparador | **Nao feito** | F3 acima |
| 183 #4 — `target` vazio na auto-ancora quebrada | Corrigido | `docs/TOOLS.md:309`, paragrafo dedicado + a terceira linha do exemplo de retorno |
| 185 #1 — `TOOLS.md` dizia que a origem nunca entra | Corrigido na doc | `docs/TOOLS.md`, `note_move`, paragrafo "A origem entra quando ela cita a si mesma com alvo escrito", com a assimetria dry-run/real explicita. **Mas ver F1: o comentario do codigo nao acompanhou** |
| 185 #2 — comentario nomeava a assercao errada como discriminante | Corrigido | commit `b34814a`; o corpo do commit explica que quem discrimina e `err == nil` + `LinksUpdated == 1`, e que o `Resolved` fica como guarda de regressao do F2 |
| 185 #3 — acento isolado em arquivo sem acentos | Corrigido | `grep -n '[aaaaeeiooouuc com acento]' internal/service/anchor_selfref_test.go` volta vazio |
| 185 #4 — `ARMADILHAS.md` sem o mecanismo | Corrigido | `docs/ARMADILHAS.md:161`, com a generalizacao "se a operacao renomeia, a lista de alvos montada antes do rename nao sobrevive a ele" |

## Verification output

Tudo em foreground, na arvore como esta.

```
$ go build ./...        -> BUILD_EXIT=0   (sem saida)
$ go vet ./...          -> VET_EXIT=0     (sem saida)
```

```
$ go test ./internal/parser/ ./internal/index/ ./internal/writer/ ./internal/service/ ./internal/mcpsrv/ -count=1
ok  github.com/jonyd/gobsidian/internal/parser   1.068s
ok  github.com/jonyd/gobsidian/internal/index    3.196s
ok  github.com/jonyd/gobsidian/internal/writer   1.697s
ok  github.com/jonyd/gobsidian/internal/service  19.793s
ok  github.com/jonyd/gobsidian/internal/mcpsrv   3.648s
```

```
$ GOOS=windows go list -f '{{.ImportPath}} {{.Imports}}' ./internal/service ./internal/index ./internal/parser ./internal/writer ./internal/mcpsrv
service  -> index, parser, search, TEXT, vault, writer   <- 'text' NAO esta no grafo de CLAUDE.md:123 (F2)
index    -> parser, text, vault                          <- confere
parser   -> text                                         <- confere
writer   -> parser, text, vault                          <- confere
mcpsrv   -> config, index, parser, service, vault        <- confere
```

(imports de stdlib e de terceiros omitidos da tabela; nenhum pacote novo de
terceiros entrou.)

```
$ grep -rn 'Cut(.*"#"' internal/ --include=*.go
internal/parser/ast.go:103:     target, anchor, _ = strings.Cut(s, "#")

$ grep -rn 'Index(.*"#"|IndexByte(.*'#'|SplitN(.*"#")' internal/ --include=*.go
(vazio)
```

Uma separacao de `#` no repositorio inteiro. `splitWikilink` a chama; os ramos
`*gast.Link` e `*gast.Image` de `collect` a chamam. Nenhuma segunda conta entrou
em `index`, `writer` ou `service`.

```
$ gofmt -l ./cmd ./internal ./tools
(vazio)

$ grep -rn '"net"|"net/http"|net/url' internal/ cmd/ --include=*.go | grep -v _test
internal/daemon/daemon.go:22:   "net"        <- IPC local, autorizado (Task 90)
internal/daemon/lock.go:13:     "net"        <- idem
internal/ipc/ipc.go:20:         "net"        <- idem
internal/parser/ast.go:231:     comentario explicando por que net/url NAO e importado
internal/parser/ast.go:233:     idem
```

Nenhum import de rede novo. Nenhum tipo do SDK MCP fora de `internal/mcpsrv`
(`grep` de `modelcontextprotocol` fora do pacote: nada; `service/broken.go` fala
so tipos de dominio). Nenhum `helpers.go`/`utils.go`/`common.go` criado. Nenhum
`runtime.GOOS` em logica compartilhada nova.

```
$ pwsh -File scripts/check_tool_params.ps1
[i] 14 structs de entrada, 78 parametros declarados.
    vaultBrokenLinksInput.State  -> tools_read.go (mcpsrv), internal/ (dominio)
    vaultBrokenLinksInput.Prefix -> tools_read.go (mcpsrv), internal/ (dominio)
    vaultBrokenLinksInput.Limit  -> tools_read.go (mcpsrv), internal/ (dominio)
    vaultBrokenLinksInput.Offset -> tools_read.go (mcpsrv), internal/ (dominio)
[OK] todo parametro declarado e lido em algum lugar.
EXIT=0

$ pwsh -File scripts/check_doc_refs.ps1
[i] corpus: 383 arquivos .go.  [i] 37 dispensa(s) em uso
[OK] nenhum token entre crases parece citar artefato ausente do codigo.
EXIT=0

$ pwsh -File scripts/check_readme_anchors.ps1
[i] 11 heading(s), 11 link(s) interno(s).
[OK] toda ancora resolve e toda secao H2 e alcancavel pela navegacao.
EXIT=0
```

Numeros de documentacao conferidos contra o codigo, um a um:

| Alegacao | Fonte no codigo | Confere? |
|---|---|---|
| formato do cache de metadados = **5** | `internal/index/persist.go:38` | sim |
| `IndexCacheParserVersion` = **2** | `internal/index/persist.go:49`, comparado no load em `:161` | sim |
| **14** tools | 14 nomes unicos em `mcp.AddTool` (`tools_read.go` 8, `tools_write.go` 5, `server.go` 1) | sim |
| limite padrao **100**, teto **500** | `ComTeto`/`LimitePadrao`/`LimiteTeto`, ecoados em `TOOLS.md:13`, `TOOLS.md:301` e na tag `jsonschema` | sim |
| **13 das 14** tools dispensam o indice invertido | `s.inverted` so em `internal/service/search.go`; `BrokenLinks` le so `s.index` | sim |
| `kind` = `wikilink` / `embed` / `markdown` | `parser.LinkKind.String()` (`types.go:20-29`) | sim |

Sobre provas de mutacao: nao foram re-executadas aqui, porque executa-las exige
editar a arvore de trabalho e esta revisao e somente leitura. As quatro provas
da Task 183 e a da Task 185 ja foram auditadas nas revisoes por tarefa
(`review-183.md`, secao "Autenticidade das provas de mutacao"; `review-185.md`,
Risco 4), e o commit `b34814a` corrigiu o unico comentario de teste que nomeava
o discriminante errado. O que verifiquei aqui foi a **forma** dos testes novos:
`broken_test.go` tem controle explicito contra falso PASS em dois pontos — a
linha `if res.Total != LimiteTeto+1 { t.Fatalf(...) }` antes de afirmar o clamp
(sem cofre acima do teto o clamp e inobservavel), e as duas assercoes negativas
que impedem "Total == 3" de passar com um filtro errado que deixasse a URL
entrar e derrubasse outro item. `tools_read_test.go` afirma
`"total":1` e `"anchor":"Nada"`, nao apenas a presenca do nome da tool.

## Verdict

**CHANGES_REQUESTED**

Nenhum defeito funcional. Build, vet, os cinco pacotes de teste, `gofmt` e os
tres checadores de doc/params estao verdes; o codigo faz o que o plano pediu, a
separacao de `#` e uma conta so de verdade, o portao de cache e efetivo, e
`docs/TOOLS.md` descreve `note_move` e `vault_broken_links` com precisao
incomum, inclusive a assimetria dry-run/real que a revisao 185 apontou.

O que impede o "aprovado" sao dois textos que ficaram atras do codigo, e as
duas regras que este projeto trata como nao negociaveis sao exatamente essas:
comentario que afirma o contrario do que a funcao faz (F1) e grafo normativo que
diz ter sido conferido hoje e nao foi (F2). Os dois consertos sao um paragrafo e
uma linha; nenhum toca comportamento. Feitos esses, o ramo esta pronto para o
release.
