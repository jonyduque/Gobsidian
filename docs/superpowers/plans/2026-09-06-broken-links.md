# Broken links: ancoras e a tool de cofre inteiro

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `broken_links` deixa de contar link de ancora como alvo ausente, e o cofre inteiro ganha uma tool que lista cada link quebrado com origem, alvo e contexto.

**Architecture:** O parser passa a separar `#ancora` tambem em link Markdown (uma conta: a mesma que o wikilink ja usa); o indice resolve alvo vazio com ancora para a propria nota de origem e entao roda a checagem de ancora que ja existe. A tool nova e leitura pura do indice — `service.BrokenLinks` percorre `NotePaths()` e filtra `LinkTargetMissing`/`LinkAnchorMissing`, com o mesmo `ComTeto` e o mesmo par `Total`/limite de `note_list`; `mcpsrv` so traduz.

**Tech Stack:** Go 1.25+, goldmark, go-sdk MCP v1.5.0 (pinado), PowerShell 7 para os gates.

**Spec:** Pedido do dono em 2026-09-06 (sessao): "Verifique se sao reportados broken links que correspondem a sites. Se o caso, retire os links de sites de broken links. Implemente uma ferramenta para mostrar todos broken links de um cofre inteiro. Ao final, revise a documentacao, faca um build novo e publique um release, comittando tudo." Medicao previa do orquestrador (2026-09-06, quatro cofres reais): URL com esquema ja e `external` e nao entra em `broken_links`; **zero** alvos quebrados contem `://`, `www.`, `.com`, `.br`, `.org`. Os falsos positivos reais sao tres formas de link de ancora: `[x](nota.md#h)` (target `nota.md#h`, sem split), `[x](#h)` e `[[#h]]` (mesma nota). Estudo: 267 alvos comecando com `#` e 10 vazios; Oral: 372 com `#`.

## Global Constraints

- CLAUDE.md inteiro vale, em especial: stdout e do JSON-RPC; nenhum `net/*`; nenhum tipo do SDK MCP fora de `internal/mcpsrv`; uma conta por regra; sem `helpers.go`/`utils.go`/`common.go`; aresta nova de import precisa de justificativa (este plano criou uma: `service → text`, Task 183, justificada em CLAUDE.md).
- Nunca `git checkout`, `git restore`, `git stash`, `git clean`, `git reset`. Nunca `go mod tidy`.
- Commit so por caminho explicito. Nunca `git add -A` nem `git add .`. Ha trabalho do dono nao commitado em `test-vault/`, `.claude/skills/`, `Resume-Claude.ps1`.
- Todo teste novo vem com prova de mutacao no passado, com a saida colada no relatorio (`docs/papeis/testador.md`).
- Numero nao medido se escreve "nao medido".
- `pwsh -File scripts/verify.ps1` verde antes de qualquer commit de codigo. Rodar em FOREGROUND. Nunca rodar `scripts/test_orphans.ps1`.
- Conventional Commits em ingles; mensagem via arquivo e `git commit -F`. Trailers: `Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>` e `Claude-Session: https://claude.ai/code/session_01P5wkw6PAdBFzF3uB1w1jNj`.
- Saida de console ASCII: `[OK]`, `[*]`, `[!]`, `[i]`, `[...]`.
- Depois de editar `.md`: `python -c "open('<arquivo>',encoding='utf-8').read()"`.
- Cache: `internal/index/persist.go` tem `IndexCacheParserVersion`; mudanca no parser sobe esse numero (e so ele). `ResolvedLink.State/Resolved/Via` sao recalculados no load, nao persistidos.

---

### Task 182: Link de ancora nao e alvo ausente

**Files:**
- Modify: `internal/parser/ast.go` (`collect`, ramos `*gast.Link` e `*gast.Image`)
- Modify: `internal/parser/types.go` (comentario de `Link.Anchor`)
- Modify: `internal/index/resolve.go` (`resolveTarget` e o ponto que chama `resolveAnchor`)
- Modify: `internal/index/persist.go` (`IndexCacheParserVersion = 2`, com o motivo no comentario)
- Modify: `internal/writer/linkrewrite.go` (`BuildLinkText`, ramos `LinkMarkdown` e `LinkEmbed` em forma Markdown)
- Test: `internal/parser/links_anchor_test.go` (novo), `internal/index/resolve_anchor_test.go` (novo), `internal/writer/linkrewrite_test.go` (caso novo)
- Golden: `testdata/parser/*.golden` — se algum mudar, e porque agora separa ancora; regenerar pelo mecanismo que os testes do parser ja usam (ver como `parser` compara golden antes de tocar) e conferir o diff a olho.

**Interfaces:**
- Consumes: `parser.Link{Target, Anchor, Kind, Raw}`, `index.resolveTarget(target, origin)`, `index.resolveAnchor`, `index.LinkState`.
- Produces: `parser.Link.Anchor` preenchido para `LinkMarkdown` e para `LinkEmbed` em forma `![]()`; `Target` desses links sem o `#...`. `index` resolve `Target == "" && Anchor != ""` para a nota de origem, estado `LinkOK` ou `LinkAnchorMissing`. Task 183 depende dos dois.

- [ ] **Step 1: Teste do parser (falha antes)**

`internal/parser/links_anchor_test.go`:

```go
package parser

import "testing"

// Link Markdown com ancora: o parser separava so o wikilink, e o indice
// procurava a nota "b.md#Sec" — que nao existe — e contava alvo ausente.
// Medido em 2026-09-06 em cofres reais: 267 alvos comecando com "#" em um
// cofre, 372 em outro, todos contados como broken_links.
func TestMarkdownLinkSeparaAncora(t *testing.T) {
	casos := []struct {
		nome, src, target, anchor string
	}{
		{"nota e heading", "[x](b.md#Sec)", "b.md", "Sec"},
		{"so ancora", "[x](#Topo)", "", "Topo"},
		{"ancora percent-encoded", "[x](b.md#Se%C3%A7%C3%A3o)", "b.md", "Seção"},
		{"embed markdown", "![x](b.md#Sec)", "b.md", "Sec"},
		{"sem ancora fica igual", "[x](b.md)", "b.md", ""},
		{"URL com fragmento: Target sem fragmento, esquema intacto", "[x](https://ex.com/p#f)", "https://ex.com/p", "f"},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			doc := Parse([]byte(c.src))
			if len(doc.Links) != 1 {
				t.Fatalf("links = %d, queria 1", len(doc.Links))
			}
			l := doc.Links[0]
			if l.Target != c.target || l.Anchor != c.anchor {
				t.Errorf("Target=%q Anchor=%q, queria %q/%q", l.Target, l.Anchor, c.target, c.anchor)
			}
		})
	}
}
```

Ajuste o nome da funcao de parse (`Parse`) ao que `internal/parser` exporta — leia `parser.go` antes; o corpo do teste nao muda. O caso da URL com fragmento fixa a regra: o split acontece antes de o indice decidir `external`, e `external` continua sendo decidido por `hasURIScheme(Target)`, que nao depende do fragmento.

- [ ] **Step 2: Rodar e ver falhar**

Run: `go test ./internal/parser -run TestMarkdownLinkSeparaAncora -v`
Expected: FAIL nos casos com ancora (`Target="b.md#Sec" Anchor=""`).

- [ ] **Step 3: Implementar no parser**

Em `internal/parser/ast.go`, nos ramos `*gast.Link` e `*gast.Image`: separar o destino em `target` e `anchor` no PRIMEIRO `#` **antes** de `PercentDecode` — `%23` decodificado nao pode virar separador —, e decodificar as duas partes. Uma conta so: extraia a separacao que `splitWikilink` faz do `#` para uma funcao pequena reutilizada pelos dois caminhos, se a de `ext_wikilink.go` puder ser reaproveitada sem carregar a logica de `|`; se nao puder, a funcao nova fica em `ast.go` e `splitWikilink` passa a chama-la. Nao deixe duas separacoes de `#`.

`Raw` continua o destino inteiro, como esta (o writer usa `Raw` para decidir encoding).

Atualize o comentario de `Link.Anchor` em `types.go` para dizer que vale para os tres tipos.

- [ ] **Step 4: Teste do indice (falha antes)**

`internal/index/resolve_anchor_test.go` — monte um cofre temporario com `a.md` contendo `# Topo` e as formas (`[x](b.md#Sec)`, `[x](#Topo)`, `[[#Topo]]`, `[x](#Nada)`, `[[#Nada]]`) e `b.md` com `# Sec`. Use o mesmo apoio que os testes de `resolve_test.go` ja usam para construir o indice. Afirme, por link, `Resolved` e `State`:

| link | Resolved | State |
|---|---|---|
| `[x](b.md#Sec)` | `b.md` | `LinkOK` |
| `[x](#Topo)` | `a.md` | `LinkOK` |
| `[[#Topo]]` | `a.md` | `LinkOK` |
| `[x](#Nada)` | `a.md` | `LinkAnchorMissing` |
| `[[#Nada]]` | `a.md` | `LinkAnchorMissing` |

E o contrapeso: `[[]]` nao existe (o parser devolve nil), mas `[x]()` — target vazio e ancora vazia — continua `LinkTargetMissing`. Inclua-o.

- [ ] **Step 5: Rodar e ver falhar**

Run: `go test ./internal/index -run TestAncoraNaMesmaNota -v` (nome do teste a sua escolha, consistente com o arquivo)
Expected: FAIL — `[x](#Topo)` e `[[#Topo]]` com `State=target_missing`.

- [ ] **Step 6: Implementar no indice**

Em `resolveTarget` (`internal/index/resolve.go`): o ramo `target == ""` passa a distinguir. Como `resolveTarget` nao recebe a ancora, a decisao mora no chamador (a funcao que percorre os links e chama `resolveAnchor`): se `link.Target == "" && link.Anchor != ""`, `Resolved = origin`, `Via = ViaPath` (ou a constante que melhor descreva "a propria nota" — se nenhuma servir, NAO invente estado novo sem comentar por que), `State = LinkOK`, e cai na checagem de ancora normal. Escreva o comentario com o defeito: as tres formas contavam como alvo ausente.

Suba `IndexCacheParserVersion` para `2` em `persist.go` com uma linha no comentario: "1 -> 2 em 2026-09-06: link Markdown passou a separar ancora; cache antigo carregaria `Target="b.md#Sec"`."

- [ ] **Step 7: Writer preserva a ancora do link Markdown**

`BuildLinkText` para `LinkMarkdown` e para `LinkEmbed` em forma `![]()` hoje formata so `newTarget`; com a ancora separada, um `note_move` apagaria o `#Sec` do link reescrito. Acrescente caso em `internal/writer/linkrewrite_test.go`: link original `[x](b.md#Sec)`, novo alvo `c.md`, esperado `[x](c.md#Sec)`; e `![x](b.md#Sec)` -> `![x](c.md#Sec)`. Rode, veja falhar, corrija em `BuildLinkText` (a ancora vai depois do alvo codificado, com o mesmo encoding que o alvo usa se ela tiver espaco — confira o que `encodeMarkdownTarget` faz e reutilize), rode, veja passar.

- [ ] **Step 8: Golden e suite inteira**

Run: `go test ./internal/parser/... ./internal/index/... ./internal/writer/... ./internal/service/... -race`
Se golden do parser mudar, regenere pelo mecanismo existente e leia o diff: so linhas de `Target`/`Anchor` de links Markdown podem mudar.

- [ ] **Step 9: Prova de mutacao (tres, no passado, saida colada)**

1. Remover o split em `ast.go` -> `TestMarkdownLinkSeparaAncora` falha.
2. Remover o ramo `Target == "" && Anchor != ""` no indice -> teste da Task falha em `[[#Topo]]`.
3. Remover a ancora de `BuildLinkText` -> caso novo do writer falha.
Restaure cada uma; cole as tres saidas no relatorio.

- [ ] **Step 10: Gate e commit**

Run: `pwsh -File scripts/verify.ps1` (completo, foreground).
Commit (caminhos explicitos, `git commit -F`):
`fix(links): anchor links resolve to their note instead of counting as missing targets`

---

### Task 183: Tool `vault_broken_links`

**Files:**
- Create: `internal/service/broken.go`
- Create: `internal/service/broken_test.go`
- Modify: `internal/mcpsrv/tools_read.go` (registro da tool, struct de input `vaultBrokenLinksInput`)
- Modify: `internal/mcpsrv/*_test.go` onde a lista de tools e afirmada (procure por `"tag_list"` nos testes do pacote)
- Modify: `docs/TOOLS.md` (secao nova, mesmo formato das outras), `README.md` (secao "MCP tools", tabela), `docs/wiki/features/tools-mcp.md`, `docs/PRD.md` (se houver tabela de tools/RF), `docs/ESTRUTURA.md` (se listar arquivos de `service`)
- Verify: `pwsh -File scripts/check_tool_params.ps1` e `pwsh -File scripts/check_readme_anchors.ps1`

**Interfaces:**
- Consumes (Task 182): `index.ResolvedLink{State, Resolved, Context, Link{Target, Anchor, Kind, Alias, Start}}`, `s.index.NotePaths()`, o acessor de nota que `VaultStats` usa, `ComTeto`, `LimitePadrao`, `LimiteTeto`, `ValidarEnum`, `CodeInvalidArgument`.
- Produces:

```go
// BrokenLinksRequest sao os parametros de vault_broken_links.
type BrokenLinksRequest struct {
	// State filtra: "target_missing", "anchor_missing" ou "" (os dois).
	State string `json:"state"`
	// Prefix restringe a origem a um caminho canonico (pasta ou nota). Vazio = cofre inteiro.
	Prefix string `json:"prefix"`
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
}

// BrokenLink e um link sem alvo ou sem ancora, com o bastante para achar e corrigir.
type BrokenLink struct {
	Source  string `json:"source"`
	Target  string `json:"target"`
	Anchor  string `json:"anchor,omitempty"`
	Alias   string `json:"alias,omitempty"`
	Kind    string `json:"kind"`  // link.Kind.String()
	State   string `json:"state"` // "target_missing" | "anchor_missing"
	Context string `json:"context,omitempty"`
}

// BrokenLinksResult: Total e a contagem ANTES de offset/limit, como em note_list.
type BrokenLinksResult struct {
	Links []BrokenLink `json:"links"`
	Total int          `json:"total"`
}

func (s *Service) BrokenLinks(_ context.Context, req BrokenLinksRequest) (BrokenLinksResult, error)
```

Regras: ordenacao deterministica por `Source` e depois `Start`; `LinkExternal` nunca entra; `State` invalido -> `CodeInvalidArgument` via `ValidarEnum` (o mesmo caminho de `direction`); `Limit` passa por `ComTeto`; `Offset` negativo vira 0; `Prefix` compara pela mesma chave que o indice usa para caminho (`text.ChaveDeCaminho` ou o que `NotePaths` devolve — nao invente comparacao nova; um `Prefix` que nao casa nada devolve `Total: 0`, nao erro). `Total` e a contagem apos o filtro de `State` e `Prefix`, antes de `Offset`/`Limit`.

- [ ] **Step 1: Teste do service (falha antes)**

`internal/service/broken_test.go`, com `newTestService`/`writeFile` de `limites_enums_test.go`: cofre com `a.md` (`# Topo`, `[[b]]`, `[[nada]]`, `[[b#Nada]]`, `[x](#Topo)`, `[s](https://ex.com)`), `sub/c.md` (`[[outra]]`), `b.md` (`# Sec`). Afirme:
- sem filtro: `Total == 3`, ordem `a.md`(`nada`), `a.md`(`b#Nada`), `sub/c.md`(`outra`); `https://ex.com` e `[x](#Topo)` ausentes;
- `State: "anchor_missing"`: so `b#Nada`, `State == "anchor_missing"`, `Anchor == "Nada"`;
- `Prefix: "sub"`: so `outra`, `Total == 1`;
- `Limit: 1, Offset: 1`: um item, `Total == 3`;
- `State: "quebrado"`: erro com `CodeOf(err) == CodeInvalidArgument`;
- `Limit: 100000` num cofre com `LimiteTeto+1` links quebrados: `len(Links) == LimiteTeto` e `Total == LimiteTeto+1` (o mesmo desenho de `TestNoteListAplicaOTetoDeLimit`: sem o cofre acima do teto, o clamp e inobservavel).

- [ ] **Step 2: Rodar e ver falhar**

Run: `go test ./internal/service -run TestBrokenLinks -v`
Expected: FAIL de compilacao (`BrokenLinks` indefinido).

- [ ] **Step 3: Implementar `internal/service/broken.go`**

Percorra `s.index.NotePaths()` em ordem (ordene se `NotePaths` nao garantir), para cada nota os `ResolvedLinks` na ordem de `Start`, filtre por estado e prefixo, conte `Total`, aplique offset e limite. Comentario de abertura: o que `vault_stats` conta, esta tool lista — e o motivo de `external` ficar de fora e o mesmo de `graph.go`.

- [ ] **Step 4: Rodar e ver passar**

Run: `go test ./internal/service -run TestBrokenLinks -v -race`
Expected: PASS.

- [ ] **Step 5: Registrar em `mcpsrv`**

Em `tools_read.go`, ao lado de `link_graph`:

```go
type vaultBrokenLinksInput struct {
	State  string `json:"state,omitempty" jsonschema:"Filtra por estado: target_missing ou anchor_missing. Omitido devolve os dois."`
	Prefix string `json:"prefix,omitempty" jsonschema:"Restringe a origem a uma pasta ou nota (caminho relativo ao cofre)."`
	Limit  *int   `json:"limit,omitempty" jsonschema:"Maximo de links devolvidos (padrao 100, teto 500)."`
	Offset *int   `json:"offset,omitempty" jsonschema:"Quantos links pular, para paginar."`
}
```

Copie o estilo das tags `jsonschema` que `linkGraphInput` usa (leia-as; se as descricoes la nao usam esse formato, siga o delas). Handler: `valorOuZero` para `Limit`/`Offset`, repassa cru para `s.svc.BrokenLinks`, `toolErr(err)`. `Name: "vault_broken_links"`, `Description: "Todos os links quebrados do cofre: alvo ausente ou ancora ausente, com origem e contexto."`. Confira que os valores de `LimitePadrao`/`LimiteTeto` batem com os numeros na descricao — se nao forem 100/500, escreva os reais.

Teste em `mcpsrv`: onde os testes do pacote listam tools (procure `"tag_list"`), acrescente `vault_broken_links`. Se houver teste que chama tools por nome com um cofre, acrescente uma chamada com `state: "anchor_missing"` e afirme que a resposta tem `total`.

- [ ] **Step 6: check_tool_params e docs**

Run: `pwsh -File scripts/check_tool_params.ps1` — precisa passar (os quatro campos sao lidos no handler como `in.State`, `in.Prefix`, `in.Limit`, `in.Offset`, e no dominio como `.State`, `.Prefix`, `.Limit`, `.Offset`).

`docs/TOOLS.md`: secao `vault_broken_links` no formato das vizinhas (schema, retorno com exemplo, erros: `INVALID_ARGUMENT` para `state` fora do enum). `README.md` secao MCP tools: uma linha na tabela. `docs/wiki/features/tools-mcp.md`: uma linha. Rode `pwsh -File scripts/check_readme_anchors.ps1` e `pwsh -File scripts/check_doc_refs.ps1`. Valide UTF-8 de cada `.md` editado.

- [ ] **Step 7: Prova de mutacao (no passado, saida colada)**

1. Trocar o filtro para deixar `LinkExternal` passar -> teste sem filtro falha (`Total == 4`).
2. Remover o `ComTeto` -> teste do teto falha.
3. Trocar a ordenacao -> teste de ordem falha.
4. Remover `ValidarEnum` -> teste de `state` invalido falha.

- [ ] **Step 8: Gate e commit**

Run: `pwsh -File scripts/verify.ps1`.
Commit: `feat(tools): vault_broken_links lists every missing target and anchor in the vault`

---

### Task 184: Revisao de documentacao e preparo do release

**Files:**
- Modify: `docs/ESTADO.md` (marco: 2026-09-06, Tasks 182–183, `IndexCacheParserVersion` 2), `docs/OPERACAO.md` (se listar tools ou versao de cache), `docs/ARCHITECTURE.md` (se descrever resolucao de link/ancora — o texto tem de dizer que ancora vale para os tres tipos), `docs/TOOLS.md` (revisao: `vault_stats.broken_links` descreve o que conta; `link_graph.include_broken` cita a tool nova como o caminho de cofre inteiro), `docs/wiki/*` paginas que descrevem parser de links ou vault_stats (marque `status: stale` so se nao corrigir), `docs/ESTRUTURA.md` (`.github/workflows/release.yml` esta ausente da arvore — inclua; `internal/service/broken.go`)
- Verify: `pwsh -File scripts/check_doc_refs.ps1`, `pwsh -File scripts/check_readme_anchors.ps1`, UTF-8 de cada `.md`

**Interfaces:** nenhuma; le o diff das Tasks 182–183 (`git log -p <BASE>..HEAD`, BASE informado no despacho).

- [ ] **Step 1: Inventario** — `grep -rn "broken_links\|anchor_missing\|Anchor\|ancora" docs/ README.md` e listar no relatorio cada ocorrencia com "certo / precisa mudar / mudado".
- [ ] **Step 2: Editar** cada lugar listado como "precisa mudar". Sem numero que nao veio da medicao registrada no plano (secao Spec) ou do proprio diff; o resto e "nao medido".
- [ ] **Step 3: Gates de doc** — `check_doc_refs.ps1`, `check_readme_anchors.ps1`, UTF-8.
- [ ] **Step 4: Build** — `pwsh -File scripts/build.ps1`; cole a linha `[...] Compilando <versao> (<commit>)` e a saida de `gobsidian version` do binario gerado no relatorio.
- [ ] **Step 5: Commit** — `docs: anchor links, vault_broken_links, and the release workflow in the tree`. Nao cria tag; a tag e o push sao do orquestrador.

---

### Task 185: `note_move` de uma nota que cita a si mesma pelo nome

Achado na rodada de correcao da Task 182, medido pelo implementador: uma nota `a.md` cujo corpo tem `[[a]]`, movida para `sub/a.md`, derruba `note_move` com `MoveNote: lendo nota "a.md": ... The system cannot find the file specified`. O mecanismo e o mesmo do F1 da revisao 182: a propria nota entra em `affectedNotes` como citante, `moverCorpo` renomeia antes, e o laco de reescrita le o caminho antigo. Pre-existente — nada da Task 182 toca link com alvo escrito —, mas o release sai com `note_move`, e o custo e um move pela metade.

**Files:**
- Modify: `internal/service/write.go` (laco de reescrita de citantes em `MoveNote`, ~linhas 481–524 e 624–632)
- Test: `internal/service/anchor_selfref_test.go` (caso novo ao lado dos da rodada 1)

**Interfaces:** nenhuma nova. Consome `index.Backlinks`, `writer.RewriteLinks`/`BuildLinkText` como hoje.

- [ ] **Step 1: Teste (falha antes)** — `a.md` com `# A

Veja [[a]].
`, `MoveNote` para `sub/a.md`: sem erro; `sub/a.md` existe; `a.md` nao existe; o corpo novo tem um link cujo `Resolved` (via `index`) e `sub/a.md` — afirme pelo indice, nao pela grafia, porque `[[a]]` continua resolvendo por nome e a reescrita pode ou nao mudar o texto.
- [ ] **Step 2: Rodar e ver falhar** — `go test ./internal/service -run TestMoveNote_NotaQueCitaASiMesma -v`; esperado: o ENOENT acima.
- [ ] **Step 3: Corrigir** — no laco de reescrita, quando `bl.From == canonicalFrom`, ler e gravar em `canonicalTo` (o arquivo ja esta la). Uma conta: a decisao de "onde a nota esta agora" fica numa variavel local usada nos dois pontos (leitura e escrita), nao em dois `if`. Comentario com o defeito.
- [ ] **Step 4: Rodar e ver passar**; rode tambem os dois testes da rodada 1 (`TestMoveNote_LinkSoDeAncoraNaoQuebraOMove`, `TestDeleteNote_NaoAcusaAPropriaNotaApagada`) e complete o discriminante que a revisao 182 pediu: o teste do F1 passa a afirmar que `[[a]]` dentro de `a.md` FOI reescrita/continua resolvendo apos o move (a metade que a rodada 1 nao pode fixar).
- [ ] **Step 5: Prova de mutacao** — remover o redirecionamento para `canonicalTo` -> o teste novo falha com o ENOENT. Saida colada.
- [ ] **Step 6: Gate e commit** — `verify.ps1` completo; `fix(write): note_move rewrites the moved note's own self-references at its new path`.

---

## Self-review

- Cobertura do pedido: sites -> medido, nao contam (Spec); falsos positivos de ancora -> Task 182 (e o defeito pre-existente de note_move que ela expos -> Task 185); tool de cofre inteiro -> Task 183; docs, build -> Task 184; release e commit -> orquestrador (tag `v1.5.0`, minor por `feat`).
- Tipos: `BrokenLinksRequest/BrokenLink/BrokenLinksResult` e `vaultBrokenLinksInput` aparecem so na Task 183 e no handler; `parser.Link.Anchor` e o unico contrato que a 182 muda e a 183 consome.
- Placeholders: nenhum "TBD"; os nomes de funcao de parse e de apoio de teste sao "leia antes" por serem conhecidos do repo, com o corpo do teste dado.
