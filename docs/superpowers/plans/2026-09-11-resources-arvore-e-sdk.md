# Resources como árvore, busca no picker, e o SDK despinado

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** fazer o cofre inteiro aparecer no host, organizável em árvore pelo cliente, buscável enquanto se digita, e com nota e pasta visualmente distintas — decidindo cada item contra o que o protocolo realmente permite, e não contra o que seria conveniente.

**Origem.** O dono viu o menu de connector do Claude Desktop listando notas soltas e perguntou se as pastas podiam virar submenus, com um primeiro item representando a pasta inteira. A resposta curta é que **submenu é decisão do cliente** — nenhuma versão do protocolo tem campo de pai/filho. A resposta útil é que há quatro coisas que o servidor pode fazer para que um cliente que queira árvore consiga montá-la, e uma quinta que resolve melhor o problema real.

**O problema real, medido em 2026-09-11** contra os quatro cofres configurados nesta máquina:

| Cofre | Notas | Pastas | Profundidade máx. | Publicado hoje |
|---|---|---|---|---|
| Estudo | 3.313 | 108 | 5 | 200 (6,0%) |
| Revisão | 1.275 | 26 | 1 | 200 (15,7%) |
| Jurisprudência | 1.254 | 30 | 4 | 200 (15,9%) |
| Oral | 79 | 13 | 3 | 79 (100%) |

`internal/mcpsrv/resources.go:65` pede `Limit: 200, Sort: "modified"`. Em Estudo, **94% do cofre nunca aparece** — e não por limite do protocolo: a paginação de `resources/list` é do SDK e já funciona.

---

## Parte 0 — despinar o SDK (pré-requisito de nada, viabilizador de tudo)

"Despinar" aqui não é tirar a versão do `go.mod` — módulo Go sempre grava versão exata, e a mitigação de risco do PRD (§ tabela de riscos) é "fixar versão exata **e** isolar atrás de camada de adaptação". A camada é `internal/mcpsrv` e continua. O que muda é **qual** versão está fixada.

**Medido em 2026-09-11**, com `go.mod`/`go.sum` copiados para o scratchpad antes e restaurados depois:

```
go get github.com/modelcontextprotocol/go-sdk/mcp@v1.7.0
go: upgraded github.com/google/jsonschema-go v0.4.2 => v0.4.3
go: upgraded github.com/modelcontextprotocol/go-sdk v1.5.0 => v1.7.0
go: added golang.org/x/time v0.15.0

go build ./...   -> exit 0, nenhuma saída
go vet ./...     -> exit 0, nenhuma saída
go test ./internal/mcpsrv/... ./internal/daemon/... ./internal/boot/...
ok  internal/mcpsrv  3.615s
ok  internal/daemon  5.138s
ok  internal/boot    1.459s
```

**Zero quebra de compilação.** A camada de adaptação pagou o que prometia pagar.

O que v1.7.0 traz e v1.5.0 não tem:

| | v1.5.0 | v1.7.0 |
|---|---|---|
| `latestProtocolVersion` | `2025-11-25` | `2026-07-28` |
| Versões negociáveis | 4 | 5 (ganha `2024-11-05` explícito) |
| `resultType` / MRTR | não | `mcp/mrtr.go` |
| `ttlMs` / `cacheScope` | não | `mcp/cache.go`, `Cacheable` embutido nos resultados |
| `subscriptions/listen` | não | sim |
| `Resource.Icons` | sim | sim |
| `ServerOptions.PageSize` | sim (default 1000) | sim (default 1000) |
| `ServerOptions.CompletionHandler` | sim | sim |

**O que isso NÃO obriga.** `latestProtocolVersion` é o que o SDK manda como *cliente*. Como servidor, ele responde a versão que o cliente pediu, se suportada (`shared.go`). Subir o SDK **não** força `2026-07-28` em ninguém: dá a capacidade de atendê-la quando o Claude Desktop pedir. RNF-24 precisa ser reescrito para dizer isso, porque hoje ele diz "fixada em 2025-11-25" como se fosse escolha nossa a cada requisição.

**O que isso obriga a revisar.** `2026-07-28` deprecia Roots, Sampling e Logging. Varredura feita: **nenhum arquivo de produção usa nenhum dos três**. O grep por `Roots|Sampling|LoggingLevel|mcp.Log` em `internal/` e `cmd/` volta vazio.

- [x] 0.1 — `go get github.com/modelcontextprotocol/go-sdk/mcp@v1.7.0`. **Nunca `go mod tidy`** (CLAUDE.md). Conferir que `golang.org/x/time` entrou como indirect e que `netcheck` continua verde — ele analisa o nosso código, não o do SDK, mas a asserção do PRD §6.4 sobre `net/http` chegar transitivamente precisa continuar verdadeira e auditável.
- [x] 0.2 — Reescrever RNF-24 (`docs/PRD.md:371,375`) e D6 (`docs/PRD.md:500`): a versão-alvo continua `2025-11-25` porque é o que os hosts instalados negociam, mas o SDK agora **suporta** `2026-07-28` e a negociação é dele. Q2 (`docs/PRD.md:513`) tem metade do gatilho cumprida — "o SDK Go marcá-la como estável" — e a outra metade não: o Claude Desktop instalado ainda não a negocia. Registrar isso, não fingir que fechou.
- [x] 0.3 — Atualizar `docs/ARCHITECTURE.md` §2.3, que hoje afirma "última revisão estável com suporte pleno no SDK Go oficial". Deixou de ser verdade.
- [x] 0.4 — Atualizar a frase do `CLAUDE.md` sobre `go mod tidy`, que cita "o pin do SDK MCP, que é decisão fechada (PRD D6)". A decisão não é o número; é que existe um número e que ele muda por decisão, não por `tidy`.
- [x] 0.5 — `pwsh -File scripts/verify.ps1` verde. As 22 etapas, não um subconjunto.

---

## Parte A — publicar o cofre inteiro, com pastas

### A1. Tirar o teto de 200

O teto não tem flag e não tem motivo declarado além de "é caro para o host" (`docs/TOOLS.md:564`). Medido em 2026-09-11, contra o SDK v1.7.0, registrando resources com nome e URI de tamanho realista:

```
n=200   AddResource total=523µs    por_item=2.613µs  heap_delta=60 KB
n=3421  AddResource total=7.488ms  por_item=2.188µs  heap_delta=907 KB
n=5988  AddResource total=9.589ms  por_item=1.601µs  heap_delta=1574 KB
```

`n=3421` é Estudo inteiro com as 108 pastas. **7,5 ms e 0,9 MB** no boot. Contra os tetos de RNF-01/RNF-07, é ruído.

O custo que sobra é o da **listagem**, e esse é do cliente: `resources/list` com `PageSize` 1000 devolve ~1000 entradas por página, 4 páginas para Estudo. Um cliente que siga o cursor vê tudo; um que ignore vê a primeira página.

**Aqui está o risco honesto, e ele muda com a ordenação.** Hoje a lista é `Sort: "modified", Order: "desc"` — as 200 mais recentes. Publicar tudo ordenado por **caminho** é o que a árvore precisa, mas se o cliente não paginar, a primeira página passa a ser "as 1000 primeiras em ordem alfabética" em vez de "as 200 mais recentes". Para quem não pagina, isso é pior. **Medido em 2026-09-14 (E1): o Claude Desktop 1.52386.6 segue `nextCursor` e carrega todas as páginas em sequência.** A1 entra como está, com `PageSize` 1000.

- [x] A1.1 — Medir, antes de mudar código, se o host segue o cursor (ver E1). O resultado é a entrada desta decisão.
- [ ] A1.2 — Substituir a chamada `ListNotes(Limit: 200, Sort: "modified")` por uma que devolva o cofre inteiro ordenado por caminho. **Não dá para fazer com `ListNotes`:** `service.ComTeto` grampeia todo `limit` em `LimiteTeto = 500` (`internal/service/errors.go:137`), e esse teto é contrato de tool declarado no schema (achado B4). Laçar com `Offset` funcionaria, ao custo de 7 ordenações completas do cofre. Entra método novo na fachada — ver A2.
- [ ] A1.3 — Teste: cofre de N notas publica N resources, e a N-ésima é legível pela URI publicada. O teste atual (`resources_test.go:99`) prova o **contrário** — que uma nota além do limite só é alcançável pelo template. Ele não some: vira o teste do template, com o nome dizendo isso.

### A2. Um método de fachada para a árvore, e por que não é `ListNotes`

`ListNotes` é a tool barata: teto de 500, projeção `ListItem` com hash, tags e frontmatter. Publicar resources não precisa de nada disso e precisa de duas coisas que ela não dá — **todas** as notas e **as pastas**.

Pastas não existem no domínio hoje. `index.Query.Folder` filtra por prefixo, mas nada calcula o **conjunto** de pastas. Derivar isso dentro de `mcpsrv` seria pôr conta de domínio na camada de adaptação, que é justamente o que `ARCHITECTURE` §2.3 diz que ela não faz.

Proposta: `service.VaultTree(ctx, TreeRequest) (TreeResult, error)`, devolvendo pastas e notas ordenadas por caminho, sem teto, documentado como **a conta única do conjunto de pastas**. Quando um dia existir uma tool `vault_folders` ou um `gobsidian index --tree`, eles chamam esta — e não uma segunda derivação que diverge no primeiro caminho com barra dupla.

- [ ] A2.1 — `TreeResult` com `Folders []FolderItem` (caminho, contagem de notas diretas, contagem recursiva) e `Notes []TreeNote` (caminho, título, tamanho, modificado). Sem hash, sem tags: não é a projeção da tool.
- [ ] A2.2 — Derivar o conjunto de pastas dos caminhos das notas, **não** do disco: o índice é a verdade e já respeita as exclusões (`.obsidian`, `.trash`). Uma varredura paralela do FS reintroduziria a divergência que `vault.Walk` existe para não ter.
- [ ] A2.3 — Teste com caminho de uma pasta só, caminho aninhado em 5 níveis (é o que Estudo tem), pasta sem nota direta mas com subpasta cheia, e nome com acento e espaço.
- [ ] A2.4 — Atualizar o grafo do `CLAUDE.md` **só se** uma aresta nova aparecer. Não deve: `service` já importa `index` e `vault`.

### A3. Pasta como resource, e o "primeiro item" que o dono pediu

O pedido original era um item representando a pasta inteira, como primeiro filho dela. Isso é exatamente um resource de pasta, e a spec `2026-07-28` o nomeia:

> Servers **MAY** return multiple resource contents in response to a single `resources/read` request. For example, a server could return the contents of several files when a **directory resource** is read.

Forma:

| | Nota | Pasta |
|---|---|---|
| URI | `gobsidian:///Direito/Penal/Dolo.md` | `gobsidian:///Direito/Penal/` |
| `Name` | `Direito/Penal/Dolo` | `Direito/Penal/` |
| `Title` | `Dolo` | `Penal` |
| `MIMEType` | `text/markdown` | `inode/directory` |

**A barra final é o discriminador, e é uma conta só.** `pathFromResourceURI` continua devolvendo o caminho; uma função `ehPastaURI(uri) bool` decide o ramo, e o handler despacha. Nota com barra final é impossível no FS, então o discriminador não tem caso ambíguo.

**O read de pasta devolve UM `ResourceContents`, não os filhos inteiros.** A spec permite devolver o conteúdo de vários arquivos; ler a raiz de Estudo assim seriam 3.313 notas numa resposta. O que a pasta devolve é um índice em Markdown: contagem, subpastas e notas diretas, cada uma como link `gobsidian:///...`. Barato, previsível, e é o que um "primeiro item da pasta" deveria mostrar.

- [ ] A3.1 — `ehPastaURI` + despacho no handler. Uma conta.
- [ ] A3.2 — Read de pasta devolvendo o índice em Markdown. Incluir a raiz (`gobsidian:///`), que é o "todos os itens do cofre".
- [ ] A3.3 — Teste: read de pasta lista os filhos diretos e **não** os netos; read de pasta vazia não dá erro; read de pasta inexistente dá erro de domínio, não panic.
- [ ] A3.4 — `resources_test.go` ganha o caso de pasta com espaço e acento, pelo mesmo motivo que `TestResourceRegistrationSurvivesPathsWithSpaces` existe: foi um panic no boot que derrubou o servidor antes de anunciar uma tool.

### A4. `Name` com caminho, `Title` com a folha

É o que torna uma lista plana legível e uma árvore montável. `Name` carrega o caminho relativo inteiro (o cliente agrupa e o filtro do `@` casa por pasta); `Title` carrega só a folha (o rótulo do nó).

Hoje `Name` recebe `n.Title` com fallback para o caminho (`resources.go:72-76`) e `Title` fica vazio — ou seja, a informação hierárquica é jogada fora exatamente no campo que a carregaria.

- [ ] A4.1 — Trocar o preenchimento. Fallback quando o título é vazio continua sendo a folha do caminho, não o caminho inteiro.
- [ ] A4.2 — Teste: nota com título de frontmatter diferente do nome do arquivo publica `Title` = título e `Name` = caminho.

### A5. Ícones — e a extensão da RNF-30 que eles exigem

`Resource.Icons []Icon` existe em v1.5.0 e v1.7.0. `Icon.Source` aceita URL http(s) **ou** data URI.

**URL http é rede.** A documentação do Claude Desktop confirma que ele busca ícones de connector pela rede e que `disableNonessentialServices: true` os bloqueia — "connectors show without icons". Um `src` remoto faria cada abertura do menu sair da máquina, o que é a RNF-30 sendo contornada pelo cliente a nosso pedido.

Portanto: **todo ícone deste servidor é `data:image/svg+xml;base64,...` embutido no binário.** Dois desenhos (nota, pasta), cada um com variante `light` e `dark` via `Icon.Theme`.

- [ ] A5.1 — `internal/mcpsrv/icones.go` com os quatro `data:` URIs como constantes, e comentário dizendo por que não são URL.
- [ ] A5.2 — **Gate `check_icones.ps1`**: nenhum `Icon{...Source:...}` em `internal/` pode ter Source que não comece com `data:`. É a RNF-30 aplicada a um caminho que o `netcheck` não enxerga, porque não é import de `net`.
- [ ] A5.3 — Três casos em `check_gates.ps1` (o que recusa, o que aceita, o inverso), conforme a regra do `CLAUDE.md`.
- [x] A5.4 — **Medido em 2026-09-14: o Claude Desktop 1.52386.6 não desenha `icons` de resource** (E2). Se não desenhar, o custo afundado é ~2 KB de binário e o gate continua valendo para quando desenhar.

---

## Parte B — busca no picker (`completion/complete`)

É o item de maior retorno, e o que de fato resolve "3.313 notas num menu".

`ServerOptions.CompletionHandler` atende `completion/complete`. Com `ref: {type: "ref/resource", uri: "gobsidian:///{+path}"}`, o host manda o que o usuário digitou e devolvemos caminhos casados.

Spec `2026-07-28`, limites textuais: **máximo 100 itens por resposta**; `total` e `hasMore` opcionais; erro `-32602` para ref desconhecida.

`mcp.NewServer` hoje recebe `nil` como options (`server.go:38`). Passa a receber `&mcp.ServerOptions{...}`.

- [ ] B1.1 — `CompletionHandler` despachando por `Ref.Type`. `ref/prompt` não existe aqui: devolver `-32602` nomeando a ref, não uma lista vazia — lista vazia mente dizendo "nenhum resultado".
- [ ] B1.2 — Casamento sobre o índice em memória, pelo caminho. Reusar a normalização que já existe (`text.ChaveDeCaminho`) — **uma conta por regra**; um `strings.Contains` local trataria `Sub/` e `sub/` como duas pastas aqui e uma no `index`.
- [ ] B1.3 — Cortar em 100, preencher `Total` com o número real de casamentos e `HasMore` quando cortou. `Total` fixo ou omitido quando há corte é "campo de API com valor fixo mente sempre".
- [ ] B1.4 — Teste: 150 casamentos devolvem 100 valores, `HasMore` verdadeiro e `Total` 150. Ref desconhecida devolve erro, não vazio.
- [x] B1.5 — **Medido em 2026-09-14: o Claude Desktop 1.52386.6 não emite `completion/complete`** (E3).

---

## Parte C — paginação de resposta: o que existe e o que não existe

O dono pediu "paginação em respostas, não só de recursos". A página `2026-07-28/server/utilities/pagination` lista, textualmente, as operações que paginam:

> * `resources/list` * `resources/templates/list` * `prompts/list` * `tools/list`

`tools/call` **não está na lista**, em nenhuma versão. Conferido na `2026-07-28/server/tools` da mesma revisão: o único `nextCursor` ali é o de `tools/list`; o resultado de `tools/call` carrega `resultType` (`complete` | `input_required`), que é o MRTR — pedir mais **entrada**, não entregar mais **saída**.

Então paginação de resposta de tool continua sendo conta nossa, no payload, e já é: `limit`/`offset` com teto único em `service.ComTeto`. O que falta não é protocolo, é honestidade de contrato — várias tools cortam sem dizer que cortaram.

- [ ] C1.1 — Varrer `docs/TOOLS.md` e o código: toda tool que aplica `ComTeto` declara `total` e um sinal de "há mais"? Onde não declara, declarar. Um resultado cortado sem sinal é indistinguível de um resultado completo, que é a mesma classe do achado B4.
- [ ] C1.2 — Corrigir `docs/TOOLS.md:564`, que diz "A listagem de resources é paginada e serve o índice em memória (...) limite fixo de 200 (`resources.go:66`)". Depois da Parte A, as duas metades estão erradas: a paginação é do SDK (`PageSize`, default 1000) e o limite de 200 deixou de existir. Doc que aponta linha de código é doc que envelhece em silêncio — apontar comportamento, não linha.
- [ ] C1.3 — Registrar em `docs/ESTADO.md` o fato medido: `tools/call` não pagina em nenhuma versão do protocolo, inclusive `2026-07-28`. É a pergunta que já foi feita duas vezes.

---

## Parte D — o que este plano recusa, com o motivo

**D1. Hierarquia por "typed links" — não existe caminho.** `ResourceLink` é tipo de *Content* (`mcp/content.go:127`) e só cabe em `CallToolResult.Content`. O retorno de `resources/read` é `[]*ResourceContents` — `URI`, `MIMEType`, `Text`, `Blob`, `Meta` — em v1.5.0 **e** em v1.7.0 (`content.go:302`). Não há onde pendurar um link tipado num resource. A hierarquia sai pela URI (Parte A) ou não sai.

**D2. Lista de resources que muda conforme a navegação — agora é proibido por escrito.** A `2026-07-28` acrescentou a frase que fecha o desenho "entrar numa pasta e relistar":

> the set of resources ... **MUST NOT** vary per-connection or as a side effect of other requests on the connection.

**D3. Tornar o MCP stateless — não se aplica, e `ARCHITECTURE` §2.3 já explicava por quê.** O `Stateless` do SDK é campo de `StreamableHTTPOptions` (`mcp/streamable.go:133`): transporte HTTP. Aqui é stdio, e RNF-30 barra abrir HTTP. A statelessness de `2026-07-28` "existe para deploys remotos com balanceamento de carga"; para um servidor stdio local ela não compra nada. E o estado que este produto tem não é de protocolo: é o índice em memória que o `daemon` mantém entre conexões — de propósito, que é o motivo de o daemon existir.

**D4. `https://claude.com/docs/llms-full.txt` não tem nada sobre isto.** Baixado e varrido em 2026-09-11: 3.113.119 bytes, **zero** ocorrências de `resources/list`, `resources/read`, `resources/templates`, "MCP resource", "expose resources" ou `@server:`. É documentação de produto — Claude Tag, diretório de connectors, configuração do Desktop, MCP Apps. O único achado com efeito aqui é o do bloqueio de favicon, que já está citado em A5.

---

## Parte E — o que medir, porque três decisões dependem de comportamento do cliente

Três itens deste plano estão escritos com "não medido" e uma alternativa. Medir é uma sessão do MCP Inspector ou um cliente de teste, não adivinhação.

- [x] E1 — O host segue `nextCursor` em `resources/list`? **Sim** — três páginas pedidas em sequência, 2.625 entradas em 2,5 s. A1 publica tudo com `PageSize` 1000; o risco "quem não pagina vê só as alfabéticas" não se aplica ao Desktop.
- [x] E2 — O host desenha `Resource.Icons`? **Não** — nota e pasta com o mesmo ícone genérico. A5 é investimento adiantado.
- [x] E3 — O host emite `completion/complete` para `ref/resource`? **Não** — nenhuma chamada, mesmo com `completions` declarado e busca digitada. A busca é do cliente. A Parte B é recurso adormecido para o Desktop.
- [x] E4 — O host agrupa por `Name` com barras, ou mostra a string crua? **Nenhum dos dois: mostra só o `Title`, em lista plana**, e a caixa "Procurar" filtra por texto fora do `Title` (`Sub B/Nota 01` achou `Nota 010`…`018`). A4 muda — ver abaixo.
- [x] E5 — Qual versão de protocolo o Claude Desktop instalado negocia hoje? **`2025-11-25`** (`app_version 1.52386.6`, clientes `claude-ai 0.1.0` e `local-agent-mode-*`). A metade da Q2 que depende do host não disparou.

### O que a Parte E mudou no plano (2026-09-14)

Medições completas em `docs/ESTADO.md`, seção "Claude Desktop como host MCP".

1. **A4 muda de forma.** O Desktop mostra **só o `Title`**, então `Title` = folha deixa cem `Nota 001` indistinguíveis — e o menu tem ~300 px, que corta pelo fim. A proposta passa a ser **folha primeiro, contexto depois**: `Dolo · Direito/Penal`. O corte preserva o que identifica a nota. `Name` continua com o caminho, porque é por ele (ou pela URI) que a busca do Desktop casa.
2. **A pasta precisa se distinguir no texto, porque o ícone não aparece.** Proposta: `Title` de pasta com prefixo `📁` — é o que o próprio exemplo da spec faz em `"title": "📁 Project Files"`. **Não medido:** se o menu renderiza o emoji.
3. **A5 desce para opcional.** O gate de `data:` continua valendo para o dia em que algum host desenhar, mas não há retorno visível hoje. Decisão do dono.
4. **A Parte B desce para opcional.** O Desktop não emite `completion/complete`; a busca que o usuário vê já funciona no cliente, desde que o `Name` carregue o caminho. Continua valendo para hosts que emitirem — **não medido** em nenhum outro.
5. **A1 fica como está.** O Desktop pagina sozinho e carrega tudo.

Registrar cada resposta em `docs/ESTADO.md` com a data e a versão do host. "O cliente não faz X" envelhece; "o cliente na versão V, em D, não fazia X" não.

---

## Parte F — achado lateral: `ServerOptions.Instructions`

Existe desde v1.5.0 (`server.go:62`) e nunca foi usado: `mcp.NewServer` recebe `nil`. O campo vai no resultado do `initialize` e é o texto que o host apresenta ao modelo como instrução do servidor.

`docs/PROMPT.md` existe justamente porque o usuário precisa colar um prompt padrão à mão. Parte dele — a parte que descreve o cofre e como usar as tools, não as preferências do usuário — poderia chegar sozinha.

Não entra junto com o resto: é mudança de contrato de conteúdo, não de protocolo, e merece decisão própria do dono. Fica registrado porque a Parte B já vai abrir `ServerOptions`, e o campo está ao lado.

- [ ] F1 — Levar a proposta ao dono, com o texto exato que iria em `Instructions` e o que ficaria em `docs/PROMPT.md`. **Não implementar sem decisão.**

---

## Ordem de execução

1. **Parte 0** — o bump. Isolado, medido, verificável sozinho. Um commit.
2. **Parte E** — as medições. Não custam código e três decisões dependem delas.
3. **A2** — `VaultTree` na fachada. Testável sem tocar em MCP.
4. **A1, A3, A4, A5** — a publicação. Um commit por item; A5 traz o gate.
5. **Parte B** — completion.
6. **Parte C** — o contrato e a documentação.

`verify.ps1` verde antes de cada commit, as 22 etapas. `docs/` e este plano atualizados no mesmo commit que muda comportamento — plano e código não divergem.

## Riscos

| Risco | Probabilidade | Impacto | Mitigação |
|---|---|---|---|
| Host não pagina `resources/list` e passa a ver 1000 alfabéticas em vez de 200 recentes | Média | Médio | E1 antes de A1; se não paginar, reduzir `PageSize` e ordenar de modo que a primeira página seja útil |
| `2026-07-28` chega por negociação e exercita caminho novo do SDK sem teste nosso | Baixa | Médio | Testes de `mcpsrv` rodam contra a versão que o SDK negocia por padrão; acrescentar caso explícito fixando a versão |
| `VaultTree` diverge de `vault.Walk` no conjunto de exclusões | Baixa | Alto | A2.2 obriga derivar do índice, nunca do disco |
| Ícone `data:` grande demais inflar cada entrada de `resources/list` | Baixa | Baixo | SVG de uma figura, medido antes de entrar; teto declarado no gate de A5 |
