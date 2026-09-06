# Revisão da Task 180 — uma chave de tag (caixa, NFC, sem `#`)

Revisor: agente `rev-180`. Pacote revisado: `4ee5db8..18d9da4` (`937e54c` bench,
`18d9da4` feat). Árvore no estado do commit `18d9da4` durante toda a execução dos
comandos abaixo — as mutações foram restauradas byte a byte (SHA-256 conferido
pelo próprio `mutate.ps1`) e `git status -- internal/` volta vazio no fim.

**Nota de concorrência:** durante esta revisão, às 14:29–14:30 de 2026-09-06, um
terceiro commit entrou no repositório — `c08c70c docs(sugestoes): track the
parser's NFD tag truncation as B20`. Ele está **fora** do intervalo revisado e é
tratado em `## Rulings check`, R1.

---

## Spec compliance

| Step | Veredito | Evidência |
|---|---|---|
| 1 — commit 1 só com o bench | ✅ | `937e54c` traz só `internal/service/bench_tags_test.go`. `antes180_index.test.exe` e `antes180_service.test.exe` datam de **13:42:23–13:42:25**, e o primeiro arquivo de produto (Step 2/3) é de 13:43+ — os binários "antes" são de fato anteriores a qualquer alteração de produto. |
| 2 — RED de compilação | ✅ (aceito por consistência) | Não reproduzível a posteriori sem desfazer o commit; `ChaveDeTag` e `PathsComTags` são símbolos novos em `18d9da4` (`git diff` confirma que nascem aqui), então o FAIL de compilação colado é o único resultado possível. |
| 3 — a chave e os três pontos de escrita | ✅ | `grep "ix\.tags\|\.tags\["` em produção: **um único** ponto que escreve (`internal/index/index.go:205`, dentro de `publishNoteLocked`), e os pontos de remoção/rename em `internal/index/update.go:217,225,227` e `:557` — os quatro derivam a chave por `ChaveDeTag`. `Tags` (`query.go:184`) dobra o prefixo pela mesma função. |
| 4 — `PathsComTags` / `candidatosPorTagLocked` | ✅ | `internal/index/query.go:239-303`. A linha-âncora `if k == tk \|\| strings.HasPrefix(k, tk+"/") {` (`query.go:252`) existe exatamente uma vez e reprovou sob mutação (saída em `## Commands I ran`, m2). |
| 5 — `service`: `vault_search` e `tag_list` | ✅ com ressalva | `search.go:238` resolve `porTag` uma vez; `search.go:449` faz `slices.BinarySearch`; `graph.go:392` e `:410` passam por `index.ChaveDeTag`. `matchesSearchFilters` tem **um só** chamador (`search.go:265`), e o caminho `searchMetadataOnly` (`search.go:392`) resolve tags por `index.List` → `coletarLocked` → `candidatosPorTagLocked`, ou seja, a mesma conta. Ressalva: ver N2. |
| 6 — golden e `Tags()` no reload | ✅ com ressalva | `git diff 4ee5db8..18d9da4 -- testdata/` volta **vazio**; a explicação do relatório confere (R2 abaixo). Ressalva sobre a asserção do `persist_test.go`: N3. |
| 7 — provas de mutação | ✅ | As duas exigidas pelo brief reproduzidas por mim, `EXIT=0` nas duas, saídas idênticas às do relatório. |
| 8 — `benchstat` | ✅ | Reproduzi `benchstat` sobre os `.txt` crus: as tabelas batem **número a número** com o relatório e com `docs/ESTADO.md`. 7 rodadas confirmadas por contagem de linhas (`^Benchmark`): index 14 = 2 benchmarks × 7; service 35 = 5 × 7. A regressão de `ListPorTag` foi diagnosticada e corrigida antes do commit, e os binários "depois" (14:03 index, 14:11 service) são posteriores à correção (14:00), coerentes com as medições (14:04 e 14:14). |
| 9 — documentação e commit 2 | ⚠️ **parcial** | `note_list.tags` (`TOOLS.md:203`) e `vault_search.tags` (`:52`) descrevem **exatamente** o que o código faz. `tag_list.prefix` (`:297`) **não** — ver N1. `verify.ps1` completo verde: `verify180.txt` confere com o relatório linha a linha. |

**Restrições globais:**

- *"Uma conta por regra"* — ✅ **cumprida dentro de `index` e `service`.** Nenhum
  `ToLower`/`ParaNFC` sobre tag sobrou fora de `ChaveDeTag` nesses dois pacotes
  (grep colado abaixo). Os `ToLower` remanescentes em `query.go` dobram `mode`,
  `Order` e `Sort`, não tags.
- *"Aresta nova precisa de justificativa"* — ✅ grafo re-extraído:
  `index → parser, text, vault`; `service → index, parser, search, vault,
  writer`. Nenhuma aresta nova; `service` continua **não** importando `text`.
- *"Não escreva número que você não mediu"* — ✅ todos os números de
  `docs/ESTADO.md` foram reproduzidos por mim a partir dos `.txt` crus.
- *"Um teste que não pode falhar é pior que teste ausente"* — ⚠️ **parcial**:
  duas mutações do brief reprovam; mas três linhas que a Task introduziu ou
  mudou sobrevivem à mutação (N2).
- *"Schema que promete e código que ignora"* — ❌ em um dos três pontos: N1.

---

## Quality

### N1 — `docs/TOOLS.md:297`: o contrato de `tag_list.prefix` afirma uma regra que o código não implementa. CONFIRMADO. Severidade: **média**

O texto novo diz: *"O prefixo casa a si mesmo e suas subtags"*. O código faz
**prefixo de string sobre a chave dobrada**, que não é a mesma coisa, e erra nas
duas direções:

- **Casa mais do que promete.** `internal/index/query.go:187` (ramo plano) e
  `internal/service/graph.go:412` (ramo hierárquico) fazem
  `strings.HasPrefix(t, prefixo)` — sem `+"/"`. Prova que não é hipótese: o
  golden **commitado** `testdata/tag_list_hierarquico.json`, variante
  `hierarquico_prefixo_no_meio`, tem `prefix: "proj/al"` e devolve `proj/alpha`
  — que não é `proj/al` nem uma subtag de `proj/al`. E o comentário de
  `internal/service/tag_list_golden_test.go:57-63` diz que isso é **deliberado**
  (fixado na Task 169: *"Prefixo que corta NO MEIO de um caminho de tag"*).
- **Não casa a si mesmo.** O exemplo do próprio schema é `'civil/'`;
  `strings.HasPrefix("civil", "civil/")` é `false`, então o prefixo `civil/`
  **não** devolve a tag `civil`.

Isto é a categoria 4 de `docs/papeis/revisor.md` ("Contrato que mente"), numa
tarefa cujo objetivo declarado é justamente fazer as três tools concordarem numa
regra documentada — e num commit marcado `feat!`, que é o que o consumidor lê
para saber o que mudou. Note que `note_list.tags` e `vault_search.tags` estão
**certos**: lá o código é `k == tk || strings.HasPrefix(k, tk+"/")`
(`query.go:252`), que é literalmente "a si mesma e suas subtags". O defeito é só
na terceira.

Atenuante, e é relevante para quem for corrigir: a frase veio **verbatim do
bloco Interfaces do brief**, que a prescreveu para os três pontos. Mas a
contradição com o golden da Task 169 é visível de dentro da tarefa — o relatório
inclusive discute esse mesmo fixture no `## Step 6` sem notar o conflito. O
caminho certo era o que o implementador usou para o parser: medir e avisar o
orquestrador.

**O que conserta.** Uma de duas, e a escolha é do dono porque muda comportamento
fixado de propósito:
(a) reescrever a descrição para o que o código faz — *"casa toda tag cuja chave
dobrada comece pelo prefixo; a comparação é por prefixo de string, não por
segmento (`civil/` não devolve `civil`)"*; ou
(b) tornar o casamento hierárquico também aqui (`t == p || HasPrefix(t, p+"/")`)
e regenerar o golden, aceitando que `hierarquico_prefixo_no_meio` muda.

### N2 — a dobra (`#`, caixa, NFC) do prefixo de `tag_list` está escrita, não verificada, nos **dois** ramos. CONFIRMADO. Severidade: **média**

`index.ChaveDeTag` no prefixo é a linha que faz `tag_list` honrar a metade do
contrato que **está** correta em `TOOLS.md` (`'#'` opcional, insensível a caixa e
a forma Unicode). Nenhum teste do repositório a segura. Três mutações, todas
rodando a suíte **inteira** do pacote (`-run .`), todas sobrevividas:

| Linha mutada | Substituição | Suíte | Resultado |
|---|---|---|---|
| `internal/index/query.go:184` | `prefix = strings.ToLower(prefix)` | `./internal/index/` completa | **EXIT=1** (passou) |
| `internal/index/query.go:184` | `prefix = strings.ToLower(prefix)` | `./internal/service/` completa | **EXIT=1** (passou) |
| `internal/service/graph.go:410` | `prefixo := strings.ToLower(req.Prefix)` | `./internal/service/` completa | **EXIT=1** (passou) |

Saídas em `## Commands I ran`, m4–m6. A causa é o fixture: **todo** prefixo
usado em teste é ASCII minúsculo (`"proj"`, `"proj/al"`, `"aç"`, `""`), onde
`ChaveDeTag` e `ToLower` são a mesma função. A prova 6 do relatório mutou a
linha de **contagem** (`graph.go:392`), não a de **prefixo** (`graph.go:410`) —
são duas regras, e só uma tem prova.

**O que conserta.** Um caso em `internal/service/tags_contrato_test.go` chamando
`TagList` com `Prefix: "#AÇ"` e com a forma NFD, nos dois ramos
(`Hierarchical: false` e `true`), esperando a mesma entrada `ação` com count 2.
Custa quatro linhas e mata as três mutações acima.

### N3 — `internal/index/persist_test.go:183`: a asserção nova não é independente. CONFIRMADO. Severidade: **baixa**

A comparação `lido.Tags("",0)` × `fresco.Tags("",0)` é **simétrica**: os dois
lados publicam por `publishNoteLocked` (reload em `internal/index/persist.go:164`,
fresco via `insert`), então ela não consegue detectar uma `ChaveDeTag` errada —
qualquer chave errada aparece igual nos dois. A única mutação que ela pega
(codec largando `Note.Tags`) já é pega pela comparação campo a campo
**pré-existente** de `persist_test.go:199`: na minha execução (m3) as duas linhas
reprovam juntas. Não é defeito — a nota `Acentuada.md` é um bom corpus e a saída
`construido = [{direito 2} {ação 1}]` é uma leitura útil da dobra. Mas a
afirmação do relatório de que foi essa asserção que passou a "morder" o caminho
de reload é mais forte do que o que ela faz.

**O que conserta.** Ou asserir as chaves de `ix.tags` diretamente (algo que
diverge quando só um dos dois caminhos dobra), ou trocar a frase do relatório
por "o corpus acentuado passou a exercitar a dobra; a asserção que a segura é a
comparação campo a campo de `:199`".

### N4 — `internal/index/tag_chave_test.go:70`: precedência de operador enfraquece a asserção. CONFIRMADO. Severidade: **baixa**

```go
if len(tags) != 2 || tags[0].Tag != "projeto" && tags[1].Tag != "projeto" {
```

`&&` liga mais forte que `||`, então isto é
`len != 2 || (tags[0] != "projeto" && tags[1] != "projeto")`: passa sempre que
**qualquer um** dos dois for `"projeto"`, e **nunca** confere que
`projeto/alpha` está lá — que é o que a mensagem de falha diz querer
(*"quero projeto e projeto/alpha dobradas"*). Veio verbatim do brief.

**O que conserta.** `slices.ContainsFunc` para os dois nomes esperados, ou duas
comparações explícitas.

### N5 — o relatório declara "o único item com p < 0,05" e a própria tabela dele mostra dois. CONFIRMADO. Severidade: **baixa**

`## Step 8` do relatório: *"O único item com p < 0,05 que sobrou:
`SearchFiltroTags`, B/op"*. A tabela colada logo acima traz também
`SearchFiltroFrontmatter-12  allocs/op  -0.00% (p=0.049 n=7)`. Reproduzi e o
`p=0.049` está lá. **Não há nada a investigar** — é −0,00 % sobre 30,12k, uma
melhora dentro do ruído, e o brief só manda investigar *piora*. O achado é sobre
a frase, não sobre o número: num relatório cuja tese é "só o que foi medido", uma
afirmação de exaustividade contrariada pela tabela da linha de cima é o padrão
que `CLAUDE.md` chama de "não afirme estado que você não verificou".

### N6 — `docs/ESTADO.md`: "Sem tarefa aberta." ficou desatualizado. CONFIRMADO. Severidade: **baixa (nit)**

O item novo de `ESTADO.md` sobre a truncagem NFD do parser fecha com *"Sem
tarefa aberta."*. Isso era verdade em `18d9da4` e deixou de ser em `c08c70c`,
que criou o **B20** em `docs/SUGESTOES.md`. Duas cópias do mesmo fato, e a menos
consultada é a que fica errada — que é exatamente o que o `CLAUDE.md` diz.

**O que conserta.** Trocar a frase por "aberta como **B20** em
`docs/SUGESTOES.md`".

### N7 — `porTag` é um retrato tomado fora do laço. CONFIRMADO (latente). Severidade: **baixa, informativa**

`internal/service/search.go:238` resolve `porTag` sob o `RLock` e o solta; as
`*index.Note` do laço são buscadas uma a uma em `:260`, cada uma tomando o
`RLock` de novo. Entre os dois, o watcher pode reindexar: abre-se uma janela em
que o filtro de tag e a nota discordam (nota que acabou de ganhar a tag não casa;
nota que acabou de perdê-la ainda casa). **Não é regressão** — `porFrontmatter`
(`:231`) já tem exatamente essa propriedade desde a Task que a extraiu do laço, e
foi aceita. Registro só para que seja decisão e não acidente.

### N8 — `TOOLS.md:331`: `vault_stats` promete contagens que `StatsResult` não tem. CONFIRMADO. Severidade: **baixa, pré-existente**

*"Contagem de notas, tamanho total, contagem de links, contagem de tags"* —
`service.StatsResult` (`internal/service/graph.go:714-735`) tem `notes`,
`assets`, `total_size`, saúde, `alias_collisions`, `generation`, `runtime`,
`watcher`. Não há campo de links nem de tags. **Anterior a esta Task** e fora do
escopo dela; anotado porque fica a dois parágrafos do texto que a Task editou e
porque cai na mesma categoria de N1.

### N9 — duas saídas de tag não dobradas ficaram sem menção na documentação. CONFIRMADO. Severidade: **baixa**

O parágrafo novo de `TOOLS.md:308-311` diz que `note_metadata.tags` mantém a
grafia original. Duas outras saídas também mantêm, e não são citadas:
`note_list` devolve `tags` cru (`internal/service/graph.go:541`, documentado em
`TOOLS.md:216` só como "`tags`"), e o subcomando CLI `index` reporta
`len(idx.Tags("", 1))` (`cmd/gobsidian/index.go:53`), cuja contagem de tags
distintas **cai** quando grafias se fundem — número visível ao usuário que muda
com esta Task e não aparece em lugar nenhum da documentação.

---

## Rulings check

**R1 — `internal/parser` intocado; a truncagem NFD vira entrada em `SUGESTOES.md`.**

- `internal/parser` intocado: ✅ `git diff --name-only 4ee5db8..18d9da4 --
  internal/parser/ testdata/` volta **vazio**.
- Fixtures NFD vindo do frontmatter: ✅ `tag_chave_test.go:36` (`d.md` com
  `tags: ["Ação"]`) e `tags_contrato_test.go` (`d.md`/`e.md` idem), ambos com o
  comentário explicando por quê.
- Entrada em `SUGESTOES.md`: ❌ **no intervalo revisado** — `docs/SUGESTOES.md`
  não está entre os 12 arquivos de `18d9da4`, e `git show 18d9da4:docs/SUGESTOES.md
  | grep -c B20` devolve `0`. ✅ **fora dele**: enquanto eu revisava, entrou
  `c08c70c docs(sugestoes): track the parser's NFD tag truncation as B20`
  (mtime do arquivo: 2026-09-06 14:29:23; commit ~14:30). Li o B20: cita
  `internal/parser/ext_tag.go:36`, dá o corpo medido, a chave `"ac"` resultante,
  o motivo de a fixture ter migrado para o frontmatter e o custo da correção
  (passe de golden + paridade). Segue o estilo do B19. **A ruling está cumprida,
  mas por um commit que não faz parte do pacote de revisão** — quem fechar a
  tarefa no ledger precisa registrar os **três** SHAs, não dois. Resta N6, a
  frase de `ESTADO.md` que ficou contradizendo o B20.

**R2 — `testdata/tag_list_hierarquico.json` inalterado.**

✅ Verificado, e a explicação do relatório está correta. `cofreDeTagsHierarquicas`
(`internal/service/tag_list_golden_test.go:29-46`) usa `proj/alpha/um`,
`proj/beta`, `docs`, `proj/alpha/dois`, `proj/alpha`, `docs/api`, `zeta` — todas
minúsculas, ASCII puro, sem `#` no texto da tag. `ChaveDeTag` é a identidade
sobre cada uma, logo nem chave nem contagem se movem, e `TestTagListGolden`
passa sem `-update`. A previsão do brief (o golden falharia) era o que estava
errado, não a entrega.

**Consequência, e ela é a que o orquestrador pediu que eu dissesse em voz alta:**
o caminho do golden **não exercita dobra nenhuma** — nem de caixa, nem de forma
Unicode, nem de `#`. Nem o fixture (que não tem grafia variante) nem as seis
variantes de consulta (cujos prefixos são todos minúsculos ASCII) chegam perto
disso. Isso é lacuna de cobertura do golden, **não** defeito desta Task — mas é a
mesma lacuna que N2 mede com mutação, e as duas se fecham com o mesmo teste.

**R3 — as três divergências declaradas em Concerns §3.**

1. **`tags_contrato_test.go` em `package service_test` com `createSearchService`
   em vez de `package service` com `newTestService` — ✅ correta, e a razão está
   certa.** Conferi `internal/service/read_test.go`: `newTestService` monta o
   serviço para testes de leitura e não o índice invertido. Com `inverted` nulo,
   `Search` devolve zero resultados sempre, e um teste de filtro que corre no
   caminho vazio passa por não haver o que filtrar — é literalmente o defeito de
   `docs/papeis/revisor.md` §1 (o reconciliador que media o caminho normal).
   Prova de que o serviço escolhido está no ramo certo: a mutação m5 do relatório
   (`slices.BinarySearch(...); ok && false`) faz `TestVaultSearchTagsCasaSubtag`
   devolver **4** resultados, isto é, os quatro hits realmente chegaram ao filtro.
2. **As duas grafias de "Ação" em notas separadas — ✅ correta, e é a decisão que
   mais me convenceu do relatório.** A forma do brief (as duas na mesma nota)
   sobreviveu à mutação de `ParaNFC` com `EXIT=1` colado, porque o pedido em NFC
   casava pela grafia NFC da própria nota. Separando (`d.md` só NFD maiúscula,
   `e.md` só NFC minúscula), só a dobra reúne as duas, e a mutação reprova
   (`EXIT=0`, prova 4). Isso é o `EXIT=1` sendo usado como instrumento em vez de
   escondido — que é o comportamento que este projeto quer ver.
3. **A forma de `candidatosPorTagLocked` (`casam(pedida, dst)` + `ordenado()`) —
   ✅ correta, medida, e não encolhe o escopo.** A forma do brief regredia
   `ListPorTag` em +16,08 % de tempo e +40,31 % de B/op porque obrigava
   `tag_mode=any` a pagar `sort`+cópia por tag para reordenar tudo no fim. A
   forma entregue mantém a **linha-âncora de mutação intacta e única**
   (`query.go:252`), o que preserva a prova que o brief pedia — e eu reproduzi
   essa prova depois da mudança (m2). O invariante que a `BinarySearch` do
   `service` depende continua valendo nos dois ramos: `any` faz
   `Sort`+`Compact` no fim (`:268-269`); `all` começa em `ordenado(t)`, já
   ordenado, e `slices.DeleteFunc` preserva a ordem relativa — resposta ao item
   (d): **sim, `porTag` é garantidamente ordenada e compactada**, por
   construção, nos dois modos, e o doc-comment de `PathsComTags` (`:289`) diz
   isso explicitamente.

---

## Commands I ran

Todos com `HEAD = 18d9da4` (os commits `c08c70c` que entrou depois só toca
`docs/SUGESTOES.md` e não afeta nada abaixo).

### (a) suíte com `-race`

```
$ go test -race -count=1 ./internal/index/ ./internal/service/ ./internal/mcpsrv/
ok  	github.com/jonyd/gobsidian/internal/index	4.083s
ok  	github.com/jonyd/gobsidian/internal/service	37.333s
ok  	github.com/jonyd/gobsidian/internal/mcpsrv	9.365s
EXIT=0
```

Os testes novos, nomeados:

```
$ go test -count=1 ./internal/index/ -run 'TestChaveDeTag|TestListPorTag' -v
--- PASS: TestChaveDeTagDobraCaixaHashENFC (0.00s)
--- PASS: TestListPorTagCasaSubtagENFD (0.01s)
ok  	github.com/jonyd/gobsidian/internal/index	0.633s

$ go test -count=1 ./internal/service/ -run 'TestVaultSearchTags|TestTagList' -v
--- PASS: TestTagListOrdenacao (0.01s)
--- PASS: TestTagListHierarquico (0.01s)
--- PASS: TestVaultSearchTags (0.02s)
--- PASS: TestTagListGolden (0.01s)
--- PASS: TestVaultSearchTagsCasaSubtag (0.02s)
--- PASS: TestVaultSearchTagsNFD (0.02s)
--- PASS: TestTagListDevolveFormaDobrada (0.03s)
--- PASS: TestTagListHierarquicoDobraGrafias (0.03s)
ok  	github.com/jonyd/gobsidian/internal/service	0.823s
```

`TestVaultSearchTags` é o teste **anterior** de filtro de tag, não modificado por
esta Task: ele continua verde sob a semântica nova, ou seja, a mudança
`BREAKING` é um superconjunto para o fixture dele.

### (b) `ToLower`/NFC sobre tag fora de `ChaveDeTag`

```
$ grep -rn "ToLower\|ParaNFC\|norm\.NF" --include=*.go internal/index internal/service | grep -v "_test.go"
internal/index/chave.go:45:	return strings.ToLower(text.ParaNFC(base))          <- chaveDeNomeDeArquivo
internal/index/chave.go:58:	return strings.ToLower(text.ParaNFC(strings.TrimPrefix(tag, "#")))   <- ChaveDeTag
internal/index/chave.go:65:	return strings.ToLower(text.ParaNFC(alias))         <- aliasKey
internal/index/query.go:263:	if strings.ToLower(mode) == "any" {                 <- modo, nao tag
internal/index/query.go:424:		order := strings.ToLower(q.Order)                 <- ordenacao
internal/index/query.go:428:		criterio := strings.ToLower(q.Sort)               <- criterio
(as demais ocorrencias sao comentarios; chave.go:50-51 e graph.go:409 sao doc)
```

Nenhuma dobra de tag fora de `ChaveDeTag` em `internal/index` e
`internal/service`. Fora desses dois pacotes sobra `internal/parser/ast.go:158`
(`dedupeTags`), que é a Concern §2 do relatório e agora o B20 do
`SUGESTOES.md` — fora do alcance por ciclo de import (`index → parser`).

Grafo de imports re-extraído, para a regra da aresta:

```
$ GOOS=windows go list -f '{{.ImportPath}}: {{join .Imports " "}}' ./internal/index/ ./internal/service/
internal/index:   ... internal/parser internal/text internal/vault ...
internal/service: ... internal/index internal/parser internal/search internal/vault internal/writer ...
```

Nenhuma aresta nova; `service` **não** importa `text`.

### (c) `Tags()` de um índice recarregado do cache × fresco

Mutação assimétrica no codec (a única classe que essa comparação pode pegar —
ver N3):

```
$ pwsh -File scripts/mutate.ps1 -Path internal/index/persist_codec.go -Anchor 'e.strSlice(n.Tags)' -Replacement 'e.strSlice(nil)' -Test TestIndiceDeMetadadosRecarregadoEIdentico -Package ./internal/index/
--- FAIL: TestIndiceDeMetadadosRecarregadoEIdentico (0.05s)
    persist_test.go:183: Tags("", 0) recarregado = [], construido = [{direito 2} {ação 1}]
    persist_test.go:199: Get(Acentuada.md) divergiu campo a campo:
        fresco      = ... Frontmatter:map[tags:[Ação ação Direito]] ... Tags:[Ação Direito ação] ...
        recarregado = ... Frontmatter:map[tags:[Ação ação Direito]] ... Tags:[] ...
    persist_test.go:199: Get(Civil/PONTO 03.md) divergiu campo a campo: ... Tags:[direito] × Tags:[] ...
FAIL	github.com/jonyd/gobsidian/internal/index	0.705s
[OK] internal/index/persist_codec.go restaurado byte a byte (SHA-256 confere).
[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
EXIT=0
```

`construido = [{direito 2} {ação 1}]` é a dobra medida por mim, e não citada do
relatório: `Direito` (de `Acentuada.md`) e `direito` (de `Civil/PONTO 03.md`)
numa chave com contagem 2, e as duas grafias de "Ação" **da mesma nota** numa
chave com contagem **1** — a deduplicação de `publishNoteLocked` funcionando. As
linhas `:183` e `:199` reprovam **juntas**, que é a base de N3.

### (d) `porTag` está ordenada?

**Sim, por construção, nos dois modos.** Leitura fechada em
`internal/index/query.go:239-303`:

- `any` (`:263-270`): acumula sem ordenar e faz `slices.Sort` + `slices.Compact`
  **uma vez** no fim → ordenada e compactada.
- `all` (`:271-286`): `cand` começa em `ordenado(t)` (`:258-262`, `Sort` +
  `Compact`); as iterações seguintes só aplicam `slices.DeleteFunc`, que
  **preserva a ordem relativa** → segue ordenada e compactada.
- `PathsComTags` (`:296`) devolve `nil` para `len(tags)==0`, e o
  doc-comment (`:289`) promete "ordenados" — a promessa e o código conferem.
- Quem consome: `internal/service/search.go:449`, `slices.BinarySearch`, e o
  guarda `len(opts.Tags) > 0` em `:448` impede a `BinarySearch` de decidir
  qualquer coisa quando `porTag` é `nil` por ausência de filtro.
- `vault_search` sempre pede `"all"` (`search.go:238`), o que é o certo: a tool
  não tem parâmetro `tag_mode` e a descrição do schema diz "TODAS as tags".
- O caminho de consulta vazia (`searchMetadataOnly`, `search.go:392-401`) não
  passa por `matchesSearchFilters` — resolve tags por `index.List` →
  `coletarLocked` → `candidatosPorTagLocked`, a **mesma** conta. Confirmado que
  `matchesSearchFilters` tem um só chamador (`grep`: `search.go:265`).

### (e) mudanças de comportamento além do contrato do brief

- `TagCount.Tag` dobrado: **documentado** (`TOOLS.md:308-311`), e afeta
  `tag_list` nos dois ramos.
- `note_metadata.tags`: **inalterado** (`graph.go:668`, `res.Tags = n.Tags`) e
  dito na documentação.
- `note_list` result `tags` (`graph.go:541`) e `cmd/gobsidian/index.go:53`:
  **não documentados** → N9.
- `Tags(prefix)` agora aceita `#` e NFD no prefixo (antes só `ToLower`): é
  ampliação, documentada — mas sem teste que a segure → N2.
- `vault_stats`: não devolve contagem de tags apesar do que `TOOLS.md:331` diz →
  N8, pré-existente.

### (f) caminho de remoção usa a mesma chave da inserção?

**Sim.** Único ponto de escrita:

```
$ grep -rn "ix\.tags\[" --include=*.go internal/index/ | grep -v _test
internal/index/index.go:204:		if len(ix.tags[k]) == 0 || ix.tags[k][len(ix.tags[k])-1] != n.Path {
internal/index/index.go:205:			ix.tags[k] = append(ix.tags[k], n.Path)      <- k := ChaveDeTag(t)
internal/index/update.go:217:			paths := ix.tags[chave]                      <- chave := ChaveDeTag(tag)
internal/index/update.go:225:				delete(ix.tags, chave)
internal/index/update.go:227:				ix.tags[chave] = filtered
internal/index/update.go:557:		paths := ix.tags[ChaveDeTag(tag)]            <- MoveNote
```

Inserção e remoção derivam a chave pela mesma função. Percorri o caso que a
deduplicação cria — nota com duas grafias que dobram numa chave só: a remoção
itera `oldNote.Tags` e visita a chave duas vezes; a primeira filtra o `path` (a
entrada única) e a segunda encontra `paths` sem ele, resultando em `delete` de
chave já ausente ou reescrita idêntica. Inócuo nos dois ramos. `MoveNote` idem:
a segunda visita não acha `oldPath` e não faz nada.

A deduplicação de `publishNoteLocked` (comparar com o **último** elemento) é
suficiente e não é frágil: dentro de uma publicação, todo `append` para a chave
`k` vindo daquela nota é o último de `ix.tags[k]`, porque a função roda com
`ix.mu` travado e processa as tags de uma nota só, em laço fechado.

### (g) `IndexCacheFormatVersion` inalterado é correto?

**Sim.** `internal/index/persist.go:38` continua em `5`, e o reload
(`persist.go:163-167`) faz exatamente `idx.publishNoteLocked(n)` nota a nota —
que é o **único** escritor de `ix.tags` (item (f)). O cache guarda `Note.Tags`
cru, então a chave dobrada é **reconstruída** na carga pela mesma função que o
`Build` usa, e nenhum byte do formato serializado muda. Um cache gravado antes
da Task 180 carrega e produz as chaves novas.

### As duas mutações do brief, reproduzidas por mim

**m1 — `chave.go`, `TrimPrefix('#')`:**

```
$ pwsh -File scripts/mutate.ps1 -Path internal/index/chave.go -Anchor 'strings.TrimPrefix(tag, "#")' -Replacement 'tag' -Test TestChaveDeTagDobraCaixaHashENFC -Package ./internal/index/
[...] Mutando internal/index/chave.go
      - strings.TrimPrefix(tag, "#")
      + tag
[...] go test -race -run TestChaveDeTagDobraCaixaHashENFC ./internal/index/
--- FAIL: TestChaveDeTagDobraCaixaHashENFC (0.00s)
    tag_chave_test.go:24: ChaveDeTag("#Ação") = "#ação", quero "ação"
    tag_chave_test.go:24: ChaveDeTag("#Projeto") = "#projeto", quero "projeto"
FAIL	github.com/jonyd/gobsidian/internal/index	0.670s
[OK] internal/index/chave.go restaurado byte a byte (SHA-256 confere).
[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
EXIT=0
```

**m2 — `query.go`, a linha do casamento hierárquico:**

```
$ pwsh -File scripts/mutate.ps1 -Path internal/index/query.go -Anchor 'if k == tk || strings.HasPrefix(k, tk+"/") {' -Replacement 'if k == tk {' -Test TestListPorTagCasaSubtagENFD -Package ./internal/index/
[...] Mutando internal/index/query.go
      - if k == tk || strings.HasPrefix(k, tk+"/") {
      + if k == tk {
[...] go test -race -run TestListPorTagCasaSubtagENFD ./internal/index/
--- FAIL: TestListPorTagCasaSubtagENFD (0.01s)
    tag_chave_test.go:60: PathsComTags(#PROJETO) = [b.md], quero [a.md b.md]
FAIL	github.com/jonyd/gobsidian/internal/index	0.902s
[OK] internal/index/query.go restaurado byte a byte (SHA-256 confere).
[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
EXIT=0
```

Ambas idênticas às do relatório, `EXIT=0`, e a m2 é posterior à mudança de forma
do `candidatosPorTagLocked` — a âncora sobreviveu à correção da regressão.

### As três mutações de N2 (suíte inteira, `-run .`)

```
$ pwsh -File scripts/mutate.ps1 -Path internal/index/query.go -Anchor 'prefix = ChaveDeTag(prefix)' -Replacement 'prefix = strings.ToLower(prefix)' -Test '.' -Package ./internal/index/
[...] go test -race -run . ./internal/index/
ok  	github.com/jonyd/gobsidian/internal/index	3.040s
[OK] internal/index/query.go restaurado byte a byte (SHA-256 confere).
[!] O teste PASSOU com a regra mutada.
EXIT=1

$ pwsh -File scripts/mutate.ps1 -Path internal/index/query.go -Anchor 'prefix = ChaveDeTag(prefix)' -Replacement 'prefix = strings.ToLower(prefix)' -Test '.' -Package ./internal/service/
[...] go test -race -run . ./internal/service/
ok  	github.com/jonyd/gobsidian/internal/service	34.306s
[OK] internal/index/query.go restaurado byte a byte (SHA-256 confere).
[!] O teste PASSOU com a regra mutada.
EXIT=1

$ pwsh -File scripts/mutate.ps1 -Path internal/service/graph.go -Anchor 'prefixo := index.ChaveDeTag(req.Prefix)' -Replacement 'prefixo := strings.ToLower(req.Prefix)' -Test '.' -Package ./internal/service/
[...] go test -race -run . ./internal/service/
ok  	github.com/jonyd/gobsidian/internal/service	34.424s
[OK] internal/service/graph.go restaurado byte a byte (SHA-256 confere).
[!] O teste PASSOU com a regra mutada.
EXIT=1
```

### `benchstat` reproduzido dos `.txt` crus

Rodadas conferidas por contagem de linhas `^Benchmark`: `t180b_index_*` 14 = 2×7;
`t180b_service_*` 35 = 5×7.

```
$ benchstat t180b_index_antes.txt t180b_index_depois.txt
                  │ t180b_index_antes.txt │       t180b_index_depois.txt        │
                  │        sec/op         │    sec/op     vs base               │
TagsSemPrefixo-12             18.11µ ± 3%   15.85µ ± 12%  -12.47% (p=0.001 n=7)
ListPorTag-12                 631.3µ ± 2%   635.2µ ± 10%        ~ (p=1.000 n=7)
geomean                       106.9µ        100.3µ         -6.16%
   B/op e allocs/op: "~ (p=1.000 n=7) ¹ all samples are equal" nos dois.

$ benchstat t180b_service_antes.txt t180b_service_depois.txt
TagListPlano-12                         19.72µ ± 13%   17.63µ ±  7%  -10.62% (p=0.001 n=7)
TagListHierarquico-12                   2.750m ± 21%   3.134m ± 39%        ~ (p=0.097 n=7)
NoteListPorTag-12                       235.9µ ± 52%   235.6µ ± 13%        ~ (p=0.535 n=7)
SearchFiltroFrontmatter-12              13.04m ± 50%   13.67m ±  3%        ~ (p=0.128 n=7)
SearchFiltroTags-12                     13.36m ±  3%   12.85m ±  8%        ~ (p=0.073 n=7)
B/op:      SearchFiltroTags 1.926Mi -> 1.935Mi  +0.48% (p=0.001 n=7)   (unico que move)
allocs/op: SearchFiltroFrontmatter 30.12k -> 30.12k  -0.00% (p=0.049 n=7)   <- N5
           SearchFiltroTags 10.15k -> 10.15k  ~ (p=0.244 n=7)
```

Todos os números batem com `docs/ESTADO.md` e com o relatório. A conclusão do
relatório sobre o `+0,48 %` de B/op de `SearchFiltroTags` — mesma contagem de
alocações, uma alocação a mais de ~9,6 KiB que **escapa** porque é devolvida,
contra um `map` por resultado que não escapava — é coerente com os dois números
e não é hipótese vestida de medição.

Datas dos artefatos, que fecham o protocolo:

```
antes180_index.test.exe    13:42:25   (commit 1 = 13:40; Step 2 = 13:43)
antes180_service.test.exe  13:42:23
depois180_index.test.exe   14:03:06   -> t180b_index_*.txt   14:04
depois180_service.test.exe 14:11:48   -> t180b_service_*.txt 14:14-14:15
```

Coerente com a narrativa do relatório sobre o binário velho de 13:57 ter causado
o falso `+24,15 %` em `NoteListPorTag` às 14:08.

### `verify.ps1`

Não reexecutei o gate completo (é o que o relatório entrega e o que o commit
exige). Conferi a saída salva contra a colada:
`%LOCALAPPDATA%\gobsidian-bench\2026-09-02\verify180.txt`, 14 etapas, todas
`[OK]`, 6 pulados conhecidos, fecho `[OK] Bateria completa. Pode commitar.` —
idêntica ao relatório, exceto por uma linha de ruído do `check_tool_params`
(`Carregamento: 656,5 ms`) que o relatório omitiu.

### Estado da árvore no fim

```
$ git status --porcelain -- internal/ docs/ testdata/
(vazio)
```

Todas as sete mutações restauradas; nada meu ficou na árvore.

---

## Verdict

**CHANGES_REQUIRED**

O trabalho de engenharia está sólido e a disciplina de medição é acima da média
deste projeto: a conta única existe de fato (um só escritor de `ix.tags`, quatro
derivações da chave por `ChaveDeTag`), o grafo não ganhou aresta, as duas
mutações do brief reprovam quando eu as rodo, os números do `benchstat` saem
iguais dos arquivos crus, e a regressão de `ListPorTag` foi achada, diagnosticada
e corrigida **antes** do commit em vez de explicada depois. As três divergências
do brief são todas melhorias com razão medida, e a de nº 2 usa um `EXIT=1` do
próprio implementador como instrumento.

O que impede o `APPROVED` é N1, e ele é da categoria que este projeto listou como
a que mais escapa da revisão: **um contrato que mente**, escrito num commit
`feat!` cujo propósito declarado é alinhar contratos. `tag_list.prefix` está
documentado como "casa a si mesmo e suas subtags" e o código faz prefixo de
string — o golden commitado do próprio projeto (`prefix: "proj/al"` →
`proj/alpha`) prova a diferença, e o exemplo do próprio schema (`'civil/'`) não
casa a tag `civil`. N2 é o outro lado da mesma linha: a metade do contrato que
**está** correta ali (`#`, caixa, NFC) sobrevive a três mutações contra as suítes
inteiras de `index` e `service`.

Os dois se fecham com pouco: uma frase reescrita em `docs/TOOLS.md:297` (ou a
decisão do dono de mudar o código e regenerar o golden) e um caso de teste com
`Prefix: "#AÇ"` nos dois ramos de `tag_list`. N3–N9 são nits e não precisam
bloquear.
