# Review — Task 182 (d91b2fb..8e684d0)

Revisor: agente `rev-182`. Somente leitura: nao rodei teste, nao toquei working
tree, index nem HEAD. Tudo abaixo saiu do diff e de leituras pontuais de codigo
fora dele, cada uma nomeada no risco que ela serve.

## Veredictos

- **Spec compliance:** ✅ — os dez passos foram cumpridos; o unico desvio (Step 6)
  e justificado e verificado.
- **Code quality:** ❌ **Needs fixes** — duas consequencias do `Resolved = origin`
  quebram `note_move` e deixam `Resolved` apontando para caminho morto depois de
  `index.MoveNote`. Nenhuma das duas aparece no diff; as duas sao alcancadas por
  ele.

---

## Spec compliance, passo a passo

| Step | Situacao | Nota |
|---|---|---|
| 1. Teste do parser | **met** | `internal/parser/links_anchor_test.go`, os seis casos do brief, literais. Ganhou um setimo teste (`TestMarkdownLinkNaoSeparaNoPercent23`) que o brief nao pedia e que cobre a regra de ORDEM. |
| 2. Ver falhar | **met** | RED colado, com os cinco subcasos que tinham de falhar e o `sem ancora fica igual` passando — o formato que so sai de uma execucao real. |
| 3. Implementar no parser | **met** | `splitAnchor` em `ast.go:100`; `splitWikilink` passou a chama-la (`ext_wikilink.go`), sobrou com `\|` e TrimSpace. **Uma separacao de `#` no pacote inteiro** — `grep` por `IndexByte(s, '#')` devolve so `ast.go:101`. `Raw` intacto nos dois ramos. `types.go` diz "vale para os TRES Kind". |
| 4. Teste do indice | **met** | `resolve_anchor_test.go` cobre a tabela do brief nas cinco linhas, na ordem, mais o contrapeso `[x]()` -> `LinkTargetMissing`. |
| 5. Ver falhar | **met** | RED colado; so os quatro de alvo vazio falham, e o relatorio explica por que o link 0 ja passava (Step 3 o consertou). Coerente. |
| 6. Implementar no indice | **deviated, justificado** | Ver Risco 1. `IndexCacheParserVersion = 2` com o comentario exato que o brief pediu (`persist.go:39-49`). |
| 7. Writer | **met** | `anchorMarkdown` (`linkrewrite.go:130`) nos dois ramos Markdown; teste com `#Sec`, embed e `#Com%20Espaco`. |
| 8. Golden e suite | **met (por alegacao falsificavel)** | Nao rodei. A alegacao "nenhum golden mudou" vem com o mecanismo (`TestGolden` sem `-update`), um `grep` que mostra por que nao havia como mudar, e `git status --porcelain testdata/` vazio. E o formato certo de alegacao. |
| 9. Prova de mutacao | **met, com sobra** | As tres do brief, no passado, com saida real e `restaurado byte a byte (SHA-256 confere)`. A quarta — inverter a ordem `PercentDecode`/split — e a que eu teria pedido: as tres primeiras ficam verdes com a ordem trocada, e o implementador diz isso explicitamente. |
| 10. Gate e commit | **met** | `verify.ps1` EXIT=0, tail com as etapas 8-14. Mensagem em Conventional Commits, em ingles, igual a do brief. Ver F5 sobre o hook. |

Regras globais: `stdout` intocado; nenhum `net/*`; nenhum tipo de SDK MCP fora de
`mcpsrv`; nenhum `helpers.go`; nenhuma aresta de import nova (o diff nao adiciona
import nenhum — `strings` ja estava nos tres arquivos); comentarios em portugues
sem acento, no tom dos vizinhos; nenhum numero nao medido (os 267/372 vem do
brief e sao repetidos como medidos naquele dia, nao inventados aqui).

---

## Findings

### F1 — `note_move` FALHA numa nota que contem link so de ancora — blocking

`internal/service/write.go:481-524` e `624-632`, via
`internal/index/backlinks.go:19`.

**Mecanismo, verificado lendo os tres sitios:**

1. `buildBacklinks` (`backlinks.go:19`) filtra so por `l.Resolved != ""`. Nao ha
   filtro de auto-referencia. Com o diff, `a.md` contendo `[[#Topo]]` produz
   `ix.backlinks["a.md"] = [{From: "a.md", ...}]`.
2. `MoveNote` (`write.go:477`) faz `backlinks := s.index.Backlinks(canonicalFrom)`
   e entra no laco com `bl.From == canonicalFrom`. Em `write.go:494` a condicao
   `rl.Resolved == canonicalFrom` casa o proprio link de ancora, e
   `affectedNotes["a.md"]` fica populado.
3. `write.go:620` move o corpo (`moverCorpo` -> `os.Rename`, `write.go:830`).
   **`a.md` deixa de existir.**
4. `write.go:624-632` percorre `affectedKeys`, que agora inclui `a.md`, e faz
   `os.ReadFile(s.vault.Abs("a.md"))` -> ENOENT ->
   `moveNoteErro(CodeInternal, "lendo nota %q")`.

Resultado: **`note_move` de qualquer nota que contenha `[x](#Topo)` ou
`[[#Topo]]` devolve erro interno depois de a nota ja ter sido movida**, deixando
o move pela metade (corpo movido, citantes nao reescritos). E no `DryRun`
(`write.go:539-553`) a origem entra em `diffs`, contradizendo o invariante que o
comentario de `write.go:528` afirma ("A origem nao entra em diffs"), e
`LinksUpdated` conta um link que nao precisava de reescrita nenhuma.

**Honestidade sobre a novidade:** o mecanismo e PRE-EXISTENTE — `a.md` contendo
`[[a]]` ja resolvia para si mesma e ja caia nele. O que o diff faz e trocar a
forma rara (auto-referencia por nome) pela forma comum: sao os 267 e 372 links
que o proprio brief mediu. Chamo de blocking porque a tarefa existe para
consertar exatamente esses links, e ela os torna capazes de derrubar `note_move`.

**Correcao concreta.** Em `write.go:493`, pular o link que nao tem alvo para
reescrever:

```go
for _, rl := range refNote.Links {
    if rl.Resolved != canonicalFrom {
        continue
    }
    // Link so de ancora aponta para a PROPRIA nota e nao tem alvo escrito:
    // mover a nota nao muda para onde ele aponta, e formatar um alvo aqui
    // inventaria "[[b#Topo]]" onde o autor escreveu "[[#Topo]]".
    if rl.Target == "" {
        continue
    }
    ...
}
```

Isso NAO pode virar `if bl.From == canonicalFrom { continue }`: `[[a]]` dentro de
`a.md` e auto-referencia com alvo escrito, e essa **precisa** ser reescrita no
move. `rl.Target == ""` e o discriminante certo.

Regressao que prova: `note_move` de uma nota com `[x](#Topo)`, esperando sucesso e
o arquivo de destino com o `[x](#Topo)` intacto. Hoje ela falha com
`CodeInternal`.

### F2 — depois de `index.MoveNote`, o link de ancora fica `LinkOK` apontando para caminho que nao esta mais no indice — blocking

`internal/index/update.go:530-531`, `538-548`, `576-605`.

**Mecanismo, verificado lendo a funcao inteira:**

- `update.go:530-531` publica sob `newPath` e **apaga `ix.notes[oldPath]`**.
- Passo 6 (`update.go:577-593`) pega `ix.backlinks[oldPath]`, que agora contem a
  entrada de si mesma `{From: oldPath}`, move o balde para `newPath` e apaga o de
  `oldPath`. Ao tentar corrigir `Resolved`, faz `ix.notes[bl.From]` com
  `bl.From == oldPath` — ja apagado no passo anterior — e **nao entra no `if`**.
  O `Resolved` do link de ancora continua `oldPath`.
- Passo 7 (`update.go:596-605`) tentaria consertar o `From` do balde, mas le
  `ix.backlinks[l.Resolved]` = `ix.backlinks[oldPath]`, que o passo 6 acabou de
  deletar. Nao conserta nada.
- Passo 8 (`update.go:613-614`) reprocessa por `citantesPorNome`, e
  `update.go:538-541` pula explicitamente `l.Target == ""` ao reindexar o citante.
  O link de ancora **nao tem chave por onde ser encontrado**, entao o
  reprocessamento dirigido nao o alcanca.

Estado final: `n.Links[i].State == LinkOK` com `Resolved == oldPath`, um caminho
que nao esta em `ix.notes` nem em `ix.assets`; e `ix.backlinks[newPath]` carrega
um `Backlink{From: oldPath}` de uma nota que nao existe. `service.LinkGraph`
(`graph.go:164-179`) emite essa aresta com `Resolved: true`.

Isto **e novo**, e nao herdado: para `[[a]]` o passo 8 alcanca a nota (o link tem
`Target != ""`, entao ele ESTA em `citantesPorNome`, atualizado em
`update.go:542-547`) e o estado se auto-corrige. Para `Target == ""` nao ha essa
saida. E e exatamente a classe que o proprio pacote nomeia como a que existe para
nunca reintroduzir — o comentario de `nomeChave` (`resolve.go:36-42`) descreve o
defeito `[[STJ]]` nestes termos: link com `state=ok` resolvendo para nota
removida.

**Correcao concreta.** Em `update.go`, logo depois de `n := &movida`
(linha 505) — `movida.Links` ja e copia privada, entao escrever nela e seguro:

```go
// Link so de ancora resolve para a PROPRIA nota. Mover a nota move o alvo, e
// nenhum dos passos abaixo alcanca este link: o passo 6 procura a origem em
// ix.notes[oldPath], ja apagado, e o passo 8 acha o citante por
// citantesPorNome, onde alvo vazio nao entra.
for i := range n.Links {
    if n.Links[i].Resolved == oldPath {
        n.Links[i].Resolved = newPath
    }
}
```

Regressao que prova: `MoveNote` numa nota com `[[#Topo]]` e depois afirmar
`Resolved == newPath` e `idx.Backlinks(newPath)[0].From == newPath`. Serve de
molde `TestMoveNote_UpdatesOutgoingBacklinks` (`move_test.go:241`).

### F3 — `note_delete --report_broken_links` acusa a propria nota — should-fix

`internal/service/write.go:693-714`. Com o self-backlink, apagar `a.md` que
contem `[[#Topo]]` coloca `"a.md"` em `brokenLinks` (a lista de "notas cujos
links quebram") e um `BrokenAnchor{From: "a.md", To: "a.md", Anchor: "Topo"}`.
Reportar a nota apagada como vitima da propria exclusao e resposta errada, ainda
que nao destrutiva.

Correcao: `if bl.From == canonical { continue }` no topo do laco de
`write.go:695` — aqui o discriminante `bl.From == canonical` **e** o certo, ao
contrario de F1: nenhum link dentro da nota apagada sobrevive a exclusao dela,
entao nao ha nada a reportar sobre ele.

### F4 — a alegacao "parity_test.go ficou verde, logo nao ha divergencia" e verdadeira mas o instrumento nao ve esta direcao — should-fix (no relatorio, nao no codigo)

`internal/index/parity_test.go:101-129`. `assertGraphMatches` itera
`ref.ResolvedLinks` e exige `nossos[alvo] >= querAoMenos`. A comparacao e
**assimetrica de proposito** (o proprio comentario, linhas 98-100, diz isso). Uma
aresta a MAIS do nosso lado — que e exatamente o que o self-backlink cria — nao
tem como reprovar esse teste.

Entao a frase do relatorio ("A suite inteira de `service` e `index` ficou verde,
incluindo `parity_test.go`, entao **nao ha divergencia medida**") esta
literalmente correta e induz ao contrario do que quer dizer: o teste nao mediu
nada nessa direcao, e nao podia. O relatorio deveria dizer "o instrumento de
paridade e assimetrico e nao consegue ver aresta a mais". Vale corrigir o texto
antes de a Task 183 se apoiar nele.

### F5 — o hook `pre_commit_docs.ps1` foi satisfeito por um comentario de shell — should-fix (processo, nao codigo)

O relatorio (linhas 30-33) conta que o hook procura `[sem-doc]` no **comando**, e
que a segunda tentativa passou por declarar a escotilha num comentario de shell.
Nao foi `--no-verify`, o `[sem-doc]` esta de fato no corpo da mensagem, e o
implementador **declarou** o que fez — isso e o comportamento certo. Mas o achado
e do hook: se ele le a linha de comando, qualquer comentario o satisfaz, e ele
nao esta gateando nada. Isso merece ticket proprio, fora desta tarefa.

### F6 — `v, _ := vault.New(root)` engole o erro — nit

`internal/index/resolve_anchor_test.go:147`. Se `vault.New` falhar, `v` e nil e a
falha aparece como panico ou como um `Build` estranho, nao como a mensagem que a
diz. `v, err := vault.New(root); if err != nil { t.Fatalf(...) }`, como o resto
dos testes do pacote.

### F7 — `![[#Topo]]` esta no comentario e nao esta no teste — nit

`internal/index/resolve.go:116` afirma as TRES formas, incluindo `![[#Topo]]`. O
teste cobre duas (`[x](#Topo)`, `[[#Topo]]`). O caminho de codigo e o mesmo
(`target == "" && anchor != ""`), entao o risco e baixo, mas a casa nao deixa
comentario afirmar o que nenhum teste fixa. Uma linha na tabela resolve.

### F8 — `[x](b.md#)` perde o `#` na reescrita — nit

`splitAnchor` (`ast.go:100`) devolve `("b.md", "")` para `"b.md#"`, e
`anchorMarkdown` (`linkrewrite.go:130`) devolve `""` para ancora vazia. O `#`
final some numa reescrita de `note_move`. Fidelidade minima, forma rarissima; nao
proponho mudar, so registro para nao ser redescoberto como surpresa.

---

## Riscos nomeados

**1. `anchor` como parametro de `resolveTarget` em vez de decisao no chamador.**
Li os tres chamadores. `resolveAllLinks` (`resolve.go:97`),
`resolveLinksForNoteLocked` (`update.go:271`) e `reprocessNoteLinksLocked`
(`update.go:431`) passam `Links[i].Anchor` e, **os tres**, chamam
`ix.resolveAnchor(...)` quando `state == LinkOK` (`resolve.go:102-104`,
`update.go:265-267`, `update.go:443-445`). Comportamento identico nos tres. O
desvio e solido e eu o aprovaria mesmo se o brief nao tivesse aberto a porta: a
condicao escrita tres vezes e a forma exata do defeito que a Task 182 conserta
(uma separacao de `#` que existia num lugar so). Confirmei ainda que o caminho
incremental resolve a ancora contra os headings CERTOS: `Replace` publica a nota
(`update.go:98`) antes de `resolveLinksForNoteLocked` (`update.go:100`), entao
`resolveAnchor` acha `ix.notes[origin]` ja atualizada.

**2. Self-backlink.** O que o codigo faz agora, dito sem rodeio:
`buildBacklinks` (`backlinks.go:12-24`) filtra so `l.Resolved != ""`, sem
excecao para auto-referencia, entao `a.md` com `[[#Topo]]` **passa a ser backlink
de si mesma**, e `note_backlinks` (`graph.go:684`) a lista. Isso vale tambem para
`[[#Nada]]`, que fica `LinkAnchorMissing` com `Resolved = a.md`.

Sobre paridade com o Obsidian eu **nao medi nada** e nao vou afirmar: o corpus de
paridade nao consegue ver aresta a mais (F4), entao a suite verde nao e evidencia
aqui nem contra nem a favor. Se `resolvedLinks` de fato registra o self-link,
a escolha e paridade — mas o painel de backlinks do Obsidian e outra coisa, e
`note_backlinks` e a tool que se parece com o painel.

Sobre invariantes quebrados, achei dois, e sao F1 e F2. Em `link_graph`
especificamente NAO ha quebra estrutural: a aresta `a.md -> a.md` e deduplicada
por `edgesMap[deAresta(a)]` (`graph.go:174` e `223`), e nenhum dos dois ramos
enfileira o no porque `visited[curr.Path]` ja e true (`graph.go:141-144`, `176`,
`225`) — nao ha laco infinito nem distancia corrompida. O que muda e a forma do
payload: passa a existir aresta com `source == target`, que nenhum cliente via
antes. Vale uma linha em `docs/TOOLS.md` quando a Task 183/184 mexer nele.

**3. Split antes de `PercentDecode`.** Verificado no diff, nos dois ramos:
`ast.go` faz `target, anchor := splitAnchor(string(node.Destination))` e so
entao `Target: PercentDecode(target)`, `Anchor: PercentDecode(anchor)` — ancora
percent-decodificada, como o brief exige, e `%23` nunca vira separador. A regra
esta fixada por `TestMarkdownLinkNaoSeparaNoPercent23` **e** pela quarta prova de
mutacao, que inverteu a ordem e viu o teste reprovar (`Target="C" Anchor=".md"`).
E a unica das quatro provas que cobre a ordem; as outras tres ficariam verdes com
ela trocada, e o relatorio diz isso.

**4. `BuildLinkText`.** `LinkMarkdown` (`linkrewrite.go:105-107`) e o ramo
Markdown de `LinkEmbed` (`linkrewrite.go:101-103`) reemitem
`anchorMarkdown(orig.Anchor, orig.Raw)`, que usa `encodeMarkdownTarget` — o mesmo
encoding do alvo, verificado nas duas direcoes do heuristico: `%20` em `origRaw`
ou espaco cru na ancora produzem `%20`. O embed em forma wiki **nao regrediu**: a
condicao `HasPrefix(orig.Raw, "!") && HasPrefix(TrimPrefix(orig.Raw,"!"), "[[")`
(`linkrewrite.go:99`) nao foi tocada e continua desviando para `formatWikiLink`,
que ja emitia `#ancora`; o ramo Markdown novo so e alcancado quando essa condicao
falha. `TestRewriteLinks_PreservesSyntaxAndEmbed` (que fixa `![[imagem_nova.png]]`)
continua no arquivo e passa na saida colada.

**5. Cache.** `IndexCacheParserVersion` 1 -> 2 com o motivo, e `persist.go` nao
tem outra hunk no diff. `IndexCacheFormatVersion` continua 5, que e o certo: o
layout nao mudou — `Anchor` ja existia em `parser.Link` e ja era codificado, o
que mudou foi o VALOR para a mesma entrada. Conferi tambem o cache irmao:
`search.CacheParserVersion` continua 1 e **nao precisa subir** — `grep` por
`\.Links|parser\.Link` em `internal/search` nao devolve nada, entao o indice
invertido nao guarda estrutura de link.

**6. Diagnosticos do gopls.** Nenhum dos dois justifica uma rodada por si.
`ast.go:101`: `strings.Cut` e de fato exatamente equivalente aqui — ela devolve
`(s, "", false)` quando nao acha, que e o mesmo que o `return s, ""` da linha 103
—, entao `t, a, _ := strings.Cut(s, "#"); return t, a` e simplificacao limpa;
vale pegar carona na rodada que F1 e F2 obrigam, nao antes.
`linkrewrite.go:75` esta em `RewriteLinks`, **fora do diff** (a primeira hunk do
arquivo comeca na linha 86): e codigo pre-existente e nao e desta tarefa. O
`golangci-lint` verde nas duas plataformas ja disse que nenhum dos dois e gate.
