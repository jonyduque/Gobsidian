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

