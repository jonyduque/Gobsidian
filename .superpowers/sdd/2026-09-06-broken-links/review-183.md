# Revisão — Task 183 (`vault_broken_links`)

**Base:** c987ff1 **Head:** 5f9ed57
Somente leitura sobre o checkout; nenhum comando git rodado, nenhum teste
executado, árvore/índice/HEAD intocados.

## Veredictos

- **Conformidade com o brief:** ✅ os oito passos entregues como especificados.
- **Qualidade de código:** **Aprovado** — nenhum achado blocking. Dois
  should-fix, ambos de documentação, de uma linha cada.

## Conformidade, passo a passo

| Passo | | Evidência |
|---|---|---|
| 1. Teste do service (falha antes) | ✅ | `broken_test.go` cobre a matriz inteira do brief e ainda acrescenta "prefixo que não casa nada" e "offset além do fim" |
| 2. RED de compilação | ✅ | saída real: `broken_test.go:42:19: svc.BrokenLinks undefined` — a linha 42 é de fato a primeira chamada, e as colunas 19 e 36 batem com a posição dos identificadores no arquivo entregue |
| 3. `internal/service/broken.go` | ✅ | percorre `NotePaths()`, filtra, conta, ordena, pagina; comentário de abertura diz o que `vault_stats` conta e por que `external` fica de fora, como pedido |
| 4. GREEN com `-race` | ✅ | saída colada, sete subtestes nomeados |
| 5. Registro em `mcpsrv` | ✅ | `vaultBrokenLinksInput` com os quatro campos; lista de tools do teste atualizada; caso novo `vault_broken_links valid` com `state: "anchor_missing"` afirmando `"total":1` |
| 6. `check_tool_params` e docs | ✅ | os quatro campos aparecem no relatório do script; `TOOLS.md`, `README.md`, `ESTRUTURA.md` e wiki editados |
| 7. Provas de mutação | ✅ | cinco (quatro do brief + prefixo), no passado, com saída colada — ver "Autenticidade" abaixo |
| 8. Gate e commit | ✅ | `verify.ps1` com as 14 etapas, 6 pulados iguais aos de antes |

### Autenticidade das provas de mutação

Confiro as provas contra o arquivo entregue, não contra a alegação. As quatro
linhas citadas nas saídas de falha caem exatamente sobre as asserções certas do
`broken_test.go` como ele está no commit:

- prova 1 → `broken_test.go:47` = `t.Fatalf("Total = %d, queria 3: %+v", ...)` ✅
- prova 2 → `broken_test.go:180` = `t.Errorf("len(Links) = %d, queria %d: o limite absurdo...")` ✅
- prova 3 → `broken_test.go:61` = o `t.Errorf` da comparação de ordem ✅
- prova 4 → `broken_test.go:148` = `t.Fatal("state invalido foi ACEITO...")` ✅

E o RED do passo 2 traz coluna de compilador (`42:19`, `42:36`) que só sai de um
`go build` real. As provas são reais.

## Achados

### 1. `docs/wiki/concepts/os-dois-indices.md:32-33` — should-fix

O denominador subiu e o numerador não: a frase virou "o índice de metadados
sozinho já sustenta **11 das 14** tools; só `vault_search` precisa do outro".
`service.BrokenLinks` lê só `s.index` — nunca `s.inv` —, e a tabela da própria
página, editada nesta tarefa, marca `vault_broken_links` como "Toca o disco?
não". A tool nova pertence ao numerador.

Pior: com a cláusula "só `vault_search` precisa do outro", a aritmética não
fecha nem antes nem depois — 14 tools menos uma dá 13, não 11. A inconsistência
é anterior a esta tarefa, mas quem mexeu na frase passa a respondê-la, e a
Concern 3 do próprio relatório invoca exatamente esta regra ("número errado em
documento normativo é a classe de defeito que este projeto já pagou") para
justificar os outros cinco arquivos.

**Correção:** subir o numerador para 12 (o mínimo que este diff deve), ou
derivar o número de verdade e escrever o derivado. Se a intenção da frase é
"não precisa do índice invertido", o número é 13 das 14.

### 2. `docs/TOOLS.md:13` — should-fix

O preâmbulo **Limites** enumera nominalmente quem tem qual teto: "`vault_search`:
padrão 20, teto embutido 200 [...]; `note_list` e `link_graph`: padrão 100, teto
500". `vault_broken_links` entra nessa segunda classe (`ComTeto`, portanto
`LimitePadrao` 100 e `LimiteTeto` 500) e não foi acrescentado à lista. Lista
normativa que enumera tools e deixa uma de fora é a mesma classe do achado que a
seção nova evita corretamente no seu próprio corpo.

**Correção:** `` `note_list`, `link_graph` e `vault_broken_links`: padrão 100, teto 500 ``.

### 3. `internal/service/broken.go:629-634` — nit

A perna `Source` do comparador é inobservável: `index.NotePaths()`
(`internal/index/query.go:136-146`) já devolve `slices.Sort(paths)`, então os
achados chegam ao `SortFunc` agrupados e ordenados por origem. Nenhum teste
consegue discriminá-la, e a prova de mutação 3 cobre só a perna `start`. Não é
defeito — manter o comparador completo torna a garantia local em vez de
dependente de um invariante de outro pacote —, mas o relatório diz "trocar a
ordenação" e provou metade dela. Vale a frase no comentário dizendo que a ordem
por `Source` é herdada de `NotePaths` e o comparador a repete de propósito.

### 4. `docs/TOOLS.md:307` — nit

Para uma auto-âncora quebrada (`[x](#Nada)`, `[[#Nada]]`) o `Target` sai vazio:
`resolveTarget` devolve `(origin, ViaPath, LinkOK)` para alvo vazio com âncora
(`internal/index/resolve.go:128-130`) e a checagem de âncora do chamador rebaixa
para `LinkAnchorMissing` sem preencher `Target`. O próprio teste reconhece o caso
(`if l.Target == "" && l.Anchor == "Topo"`). A descrição do retorno diz que
`target` é "a grafia do alvo, sem o `#`" e não avisa que ele vem vazio nessa
linha — quem consome a lista para corrigir links vai ler `"target":""` sem
explicação. Uma oração resolve.

### 5. `internal/service/broken.go:600-604` — informativo, fora do escopo

`s.index.Get(p)` devolve `*Note` sob `RLock` e o solta ao retornar
(`internal/index/index.go:66-71`); `n.Links` é lido fora da trava. É exatamente
o que `VaultStats` faz (`internal/service/graph.go:765-785`) e o que `LinkGraph`
faz — mesmo acessor, mesmo padrão. Não introduzido aqui e não é achado desta
tarefa; registro porque o risco 1 pergunta.

## Riscos nomeados

**1. `service.BrokenLinks`.** Tudo confere. `total := len(achados)` acontece
depois dos dois filtros e antes de `offset`/`limit`. A ordem é
(`Source`, `start`), determinística. `LinkExternal` e `LinkOK` ficam de fora pelo
whitelist explícito de `LinkTargetMissing`/`LinkAnchorMissing` — e a prova de
mutação 1 mostra o `https://ex.com` entrando quando a regra cai, com
`State:external` na saída. `ValidarEnum` é o mesmo caminho de `direction` e
devolve `CodeInvalidArgument` (`internal/service/errors.go:165-173`). `Offset`
além do fim: `if offset < total` deixa `pagina` nil, `make([]BrokenLink, 0)`
serializa `[]` e não `null`; e como `offset+limit` só é avaliado dentro daquele
ramo, um `offset` absurdo não estoura a soma. `ComTeto` aplicado, com o cofre de
`LimiteTeto+1` que torna o clamp observável — o mesmo desenho de
`TestNoteListAplicaOTetoDeLimit`, inclusive a asserção de controle. Sobre a
trava: usa o mesmo acessor que `VaultStats` (achado 5).

**2. Handler `mcpsrv`.** Os quatro campos são lidos (`in.State`, `in.Prefix`,
`in.Limit`, `in.Offset`), os dois ponteiros via `valorOuZero`, que devolve 0 e
deixa o padrão para o service — a única conta de cada padrão, como manda o
comentário da Task 168. `toolErr(err)` no erro. Os números da descrição batem:
`LimitePadrao = 100`, `LimiteTeto = 500` (`internal/service/errors.go:136-137`),
e a tag diz "Padrão 100, teto 500". Nenhum tipo do SDK cruza a fronteira: o
handler traduz para `service.BrokenLinksRequest` e devolve
`service.BrokenLinksResult`. Nenhuma aresta de import nova — `broken.go` importa
`index` e `text`, ambos já no grafo de `service`.

**3. `docs/TOOLS.md` vs código.** Confiro campo a campo: `state`, `prefix`,
`limit` (default 100, clamp 500), `offset` (negativo vira 0) — todos batem com
`broken.go`. Nomes do exemplo de retorno idênticos às tags JSON de `BrokenLink`
e `BrokenLinksResult`. `kind` documentado como `wikilink`/`embed`/`markdown`
confere com `parser.LinkKind.String()` (`internal/parser/types.go:20-29`).
`INVALID_ARGUMENT` confere. O bloco JSON publica `"enum"` e `"default"` que o
schema realmente servido não carrega — mas isso é a casa: `link_graph.direction`
e `tag_list.sort` fazem o mesmo, e o preâmbulo e `schema_params_test.go:266-271`
registram que o servidor só envia `type`+`description`. Não é achado. A frase
pedida sobre `source == target` está em `TOOLS.md:287`, na seção `link_graph`, e
credita a Task 182. Único furo: o preâmbulo de Limites (achado 2). Ausência de
`truncated` em `BrokenLinksResult` segue o precedente de `note_list`, que o
brief mandou copiar — não conto como achado.

**4. README + wiki.** A contagem está certa: 14 chamadas `mcp.AddTool` em
produção — `server.go` (`vault_stats`) + 8 em `tools_read.go` + 5 em
`tools_write.go`; a nona de `AddTool` no pacote é a `panic_probe` de
`export_test.go`, que não conta. Logo 9 de leitura e 5 de escrita, e a tabela do
wiki lista as nove. "Seis das nove respondem só do índice em memória" bate com
as seis linhas "não" da tabela. Nenhum "13 tools" sobrou em `docs/` ou no README
(grep por variantes numéricas e por extenso). Os textos de link mudaram só o
rótulo — o alvo `features/tools-mcp.md` é o mesmo —, então nenhuma âncora
quebrou, o que `check_readme_anchors` confirma. `source_commit: c987ff1` segue o
que `docs/wiki/_wiki/schema.md:79` define ("commit dos fontes quando a página foi
escrita") e o comprimento de 7 hex já existe em outras páginas — não é achado.
O que sobrou é o numerador do achado 1.

**5. Testes.** A matriz do brief está inteira e cada caso discrimina. Sem filtro:
`Total == 3` com a ordem afirmada item a item, mais duas asserções que nomeiam
o externo e a auto-âncora caso um filtro errado os deixe passar. `anchor_missing`
afirma `State`, `Anchor` **e** `Context` não vazio. Prefixo: `Total == 1` e a
origem certa. `Limit: 1, Offset: 1` afirma que o item é o **segundo** da ordem, o
que também prende a ordenação. `state` inválido afirma o código, não só o erro.
Teto: com a asserção de controle `Total == LimiteTeto+1` antes, sem a qual o
clamp seria inobservável. As cinco mutações reprovam, e as linhas citadas batem
com o arquivo (ver Autenticidade). Nada aqui é teste que não pode falhar.

**6. `prefix` como prefixo de string.** Documentado nos dois lugares que o
cliente lê — a tag `jsonschema` (`tools_read.go:402`) e `TOOLS.md:300` — com o
exemplo explícito `'sub' casa 'sub/c.md' e também 'subtotal.md'`, e a comparação
passa por `text.ChaveDeCaminho`, que é ToSlash + NFC + ToLower
(`internal/text/normalize.go:74-76`), o que sustenta a frase "insensível a caixa
e a forma Unicode (NFC)". Como o orquestrador decidiu manter e a documentação
diz como se comporta, não é achado.

## Concerns do relatório

As seis são escopo, não defeito, e nenhuma esconde entrega parcial. A 3
(crescimento do escopo no wiki) foi a decisão certa e é o que produz o achado 1 —
o rename foi feito, o número que o acompanhava não. A 4 (`PRD.md` intocado) está
correta: não há tabela de tools lá, e reescrever a descrição do marco M2 seria
falsificá-lo. A 5 (`start` fora do retorno) segue o brief; se virar contrato
novo, é tarefa própria.
