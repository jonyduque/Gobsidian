### Task 14: Extensão goldmark — wikilinks e embeds

**Files:**
- Create: `internal/parser/ext_wikilink.go`, `internal/parser/ext_wikilink_test.go`

**Interfaces:**
- Consumes: `parser.Link`, `parser.LinkKind` (Task 12)
- Produces: `parser.WikilinkExtension` (implementa `goldmark.Extender`); nó AST `parser.WikilinkNode` com campos `Target`, `Alias`, `Anchor`, `Kind`, `Raw`, `Start`, `End`

- [ ] **Step 1: Escrever o teste**

`internal/parser/ext_wikilink_test.go`:

```go
package parser_test

import (
	"testing"

	"github.com/jonyd/gobsidian/internal/parser"
)

func TestWikilinkForms(t *testing.T) {
	tests := []struct {
		name   string
		in     string
		target string
		alias  string
		anchor string
		kind   parser.LinkKind
	}{
		{"simples", "[[nota]]", "nota", "", "", parser.LinkWiki},
		{"com alias", "[[nota|apelido]]", "nota", "apelido", "", parser.LinkWiki},
		{"com heading", "[[nota#Cap 1]]", "nota", "", "Cap 1", parser.LinkWiki},
		{"com bloco", "[[nota#^abc123]]", "nota", "", "^abc123", parser.LinkWiki},
		{"heading e alias", "[[nota#Cap 1|Cap um]]", "nota", "Cap um", "Cap 1", parser.LinkWiki},
		{"caminho", "[[Civil/PONTO 03]]", "Civil/PONTO 03", "", "", parser.LinkWiki},
		{"embed", "![[nota]]", "nota", "", "", parser.LinkEmbed},
		{"embed de imagem", "![[diagrama.png]]", "diagrama.png", "", "", parser.LinkEmbed},
		{"embed de secao", "![[nota#Cap 1]]", "nota", "", "Cap 1", parser.LinkEmbed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			note, err := parser.Parse([]byte("texto " + tt.in + " texto\n"))
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if len(note.Links) != 1 {
				t.Fatalf("links = %d, quer 1: %+v", len(note.Links), note.Links)
			}

			got := note.Links[0]
			if got.Target != tt.target {
				t.Errorf("Target = %q, quer %q", got.Target, tt.target)
			}
			if got.Alias != tt.alias {
				t.Errorf("Alias = %q, quer %q", got.Alias, tt.alias)
			}
			if got.Anchor != tt.anchor {
				t.Errorf("Anchor = %q, quer %q", got.Anchor, tt.anchor)
			}
			if got.Kind != tt.kind {
				t.Errorf("Kind = %v, quer %v", got.Kind, tt.kind)
			}
			if got.Raw != tt.in {
				t.Errorf("Raw = %q, quer %q — a forma original tem que ser preservada", got.Raw, tt.in)
			}
		})
	}
}

// RF-17: a diferenca entre um grafo de links correto e um grafo plausivel.
func TestWikilinkSuppressedInCode(t *testing.T) {
	tests := []struct {
		name string
		in   string
	}{
		{"bloco cercado", "```\n[[nao e link]]\n```\n"},
		{"bloco cercado com linguagem", "```go\n// [[nao e link]]\n```\n"},
		{"codigo inline", "use `[[nao e link]]` aqui\n"},
		{"bloco indentado", "    [[nao e link]]\n"},
		{"escapado", "\\[\\[nao e link\\]\\]\n"},
		{"colchete literal simples", "isto [nao] e link\n"},
		{"til cercado", "~~~\n[[nao e link]]\n~~~\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			note, err := parser.Parse([]byte(tt.in))
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if len(note.Links) != 0 {
				t.Errorf("links = %+v, quer nenhum", note.Links)
			}
		})
	}
}

func TestWikilinkOffsetsPointAtSource(t *testing.T) {
	src := "abc [[nota]] def\n"

	note, err := parser.Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(note.Links) != 1 {
		t.Fatalf("links = %d, quer 1", len(note.Links))
	}

	l := note.Links[0]
	if got := src[l.Start:l.End]; got != "[[nota]]" {
		t.Errorf("src[%d:%d] = %q, quer %q", l.Start, l.End, got, "[[nota]]")
	}
}
```

**Por que isto justifica goldmark.** Os sete casos de `TestWikilinkSuppressedInCode` são exatamente onde extração por regex quebra. Uma extensão que participa do parse real não precisa rastrear contexto de código: dentro de um bloco de código o parser não está em contexto inline, e o wikilink simplesmente não é oferecido ao nosso inline parser.

- [ ] **Step 2: Rodar para confirmar que falha**

Run: `go test ./internal/parser/ -run TestWikilink -v`
Esperado: FAIL — `undefined: parser.Parse`.

- [ ] **Step 3: Implementar o nó e o inline parser**

`internal/parser/ext_wikilink.go`:

```go
package parser

import (
	"bytes"

	"github.com/yuin/goldmark"
	gast "github.com/yuin/goldmark/ast"
	gparser "github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

var KindWikilink = gast.NewNodeKind("Wikilink")

type WikilinkNode struct {
	gast.BaseInline

	Target string
	Alias  string
	Anchor string
	Kind   LinkKind
	Raw    string
	Start  int64
	End    int64
}

func (n *WikilinkNode) Kind() gast.NodeKind { return KindWikilink }

func (n *WikilinkNode) Dump(src []byte, level int) {
	gast.DumpHelper(n, src, level, map[string]string{
		"Target": n.Target,
		"Alias":  n.Alias,
		"Anchor": n.Anchor,
		"Raw":    n.Raw,
	}, nil)
}

type wikilinkParser struct{}

// Trigger diz ao goldmark em quais bytes oferecer este parser. '[' cobre
// [[...]]; '!' cobre ![[...]].
func (p *wikilinkParser) Trigger() []byte { return []byte{'[', '!'} }

func (p *wikilinkParser) Parse(_ gast.Node, block text.Reader, _ gparser.Context) gast.Node {
	line, seg := block.PeekLine()

	embed := false
	offset := 0
	if len(line) > 0 && line[0] == '!' {
		embed = true
		offset = 1
	}
	if len(line) < offset+4 || line[offset] != '[' || line[offset+1] != '[' {
		return nil
	}

	close := bytes.Index(line[offset+2:], []byte("]]"))
	if close < 0 {
		return nil
	}
	inner := line[offset+2 : offset+2+close]

	// Um wikilink nao atravessa linha, e nao contem "[[" aninhado.
	if bytes.ContainsAny(inner, "\n") || bytes.Contains(inner, []byte("[[")) {
		return nil
	}

	total := offset + 2 + close + 2
	raw := string(line[:total])

	target, anchor, alias := splitWikilink(string(inner))
	if target == "" && anchor == "" {
		return nil
	}

	node := &WikilinkNode{
		Target: target,
		Alias:  alias,
		Anchor: anchor,
		Raw:    raw,
		Start:  int64(seg.Start),
		End:    int64(seg.Start + total),
	}
	node.Kind = LinkWiki
	if embed {
		node.Kind = LinkEmbed
	}

	block.Advance(total)
	return node
}

// splitWikilink reparte "alvo#ancora|alias" nas tres partes. A ordem importa:
// o alias e sempre o ultimo, e a ancora vem antes dele.
func splitWikilink(inner string) (target, anchor, alias string) {
	if i := bytes.IndexByte([]byte(inner), '|'); i >= 0 {
		alias = inner[i+1:]
		inner = inner[:i]
	}
	if i := bytes.IndexByte([]byte(inner), '#'); i >= 0 {
		anchor = inner[i+1:]
		inner = inner[:i]
	}
	return trimSpace(inner), trimSpace(anchor), trimSpace(alias)
}

func trimSpace(s string) string { return string(bytes.TrimSpace([]byte(s))) }

// WikilinkExtension registra o inline parser no goldmark.
type WikilinkExtension struct{}

func (WikilinkExtension) Extend(md goldmark.Markdown) {
	md.Parser().AddOptions(
		gparser.WithInlineParsers(
			// Prioridade abaixo da do link padrao do CommonMark (200), para
			// que "[[x]]" seja oferecido a nos antes de virar dois links
			// aninhados.
			util.Prioritized(&wikilinkParser{}, 150),
		),
	)
}
```

**Sobre a prioridade 150.** Números menores são tentados primeiro no goldmark. O link padrão do CommonMark está em 200; entrar antes dele é o que impede `[[nota]]` de ser interpretado como um link contendo outro link. Se o teste `TestWikilinkForms` produzir dois links em vez de um, é a prioridade que está errada.

- [ ] **Step 4: Implementar `Parse` e a travessia da AST**

`internal/parser/ast.go` e `internal/parser/parser.go` são criados aqui em forma mínima — só wikilinks — e completados nas Tasks 15 a 17.

`internal/parser/parser.go`:

```go
package parser

import (
	"github.com/yuin/goldmark"
	gtext "github.com/yuin/goldmark/text"
)

// md e construido uma vez: goldmark.Markdown e seguro para uso concorrente
// apos a construcao, e o worker pool de indexacao depende disso.
var md = goldmark.New(
	goldmark.WithExtensions(
		WikilinkExtension{},
		BlockIDExtension{},
		TagExtension{},
		InlineFieldExtension{},
	),
)

// Parse transforma os bytes de uma nota em ParsedNote. Puro: nao toca disco,
// nao guarda estado, nao conhece o caminho do arquivo.
func Parse(data []byte) (*ParsedNote, error) {
	fm, body, bodyOffset := SplitFrontmatter(data)

	note := &ParsedNote{}

	if len(fm) > 0 {
		decoded, err := DecodeFrontmatter(fm)
		if err != nil {
			// Frontmatter quebrado nao invalida a nota: o corpo continua
			// tendo headings e links uteis, e recusar tudo seria desproporcional.
			note.FrontmatterErr = err.Error()
		} else {
			note.Frontmatter = decoded
			note.Tags = append(note.Tags, tagsFromFrontmatter(decoded)...)
			note.Aliases = aliasesFromFrontmatter(decoded)
			note.Title = titleFromFrontmatter(decoded)
		}
	}

	note.Headings = ExtractHeadings(body, bodyOffset)
	if note.Title == "" {
		note.Title = firstH1(note.Headings)
	}

	// O reader recebe o CORPO, e todos os offsets do goldmark sao relativos a
	// ele. bodyOffset e somado na coleta.
	doc := md.Parser().Parse(gtext.NewReader(body))
	collect(doc, body, bodyOffset, note)

	dedupeTags(note)
	return note, nil
}

func firstH1(hs []Heading) string {
	for _, h := range hs {
		if h.Level == 1 {
			return h.Text
		}
	}
	return ""
}
```

`internal/parser/ast.go`:

```go
package parser

import (
	gast "github.com/yuin/goldmark/ast"
)

// collect percorre a AST uma unica vez e distribui cada no de interesse para
// o campo correspondente de ParsedNote.
func collect(doc gast.Node, body []byte, bodyOffset int64, note *ParsedNote) {
	_ = gast.Walk(doc, func(n gast.Node, entering bool) (gast.WalkStatus, error) {
		if !entering {
			return gast.WalkContinue, nil
		}

		switch node := n.(type) {
		case *WikilinkNode:
			note.Links = append(note.Links, Link{
				Raw:    node.Raw,
				Target: node.Target,
				Alias:  node.Alias,
				Anchor: node.Anchor,
				Kind:   node.Kind,
				Start:  bodyOffset + node.Start,
				End:    bodyOffset + node.End,
			})

		case *gast.Link:
			// Link Markdown padrao. So interessa quando aponta para dentro do
			// cofre; a decisao de o que e interno cabe ao indice, entao aqui
			// registramos tudo e deixamos a resolucao filtrar.
			note.Links = append(note.Links, Link{
				Raw:    string(node.Destination),
				Target: string(node.Destination),
				Alias:  string(node.Text(body)),
				Kind:   LinkMarkdown,
			})

		case *BlockIDNode:
			note.Blocks = append(note.Blocks, Block{
				ID:    node.ID,
				Start: bodyOffset + node.Start,
				End:   bodyOffset + node.End,
			})

		case *TagNode:
			note.Tags = append(note.Tags, node.Name)

		case *InlineFieldNode:
			if note.Inline == nil {
				note.Inline = map[string][]string{}
			}
			note.Inline[node.Key] = append(note.Inline[node.Key], node.Value)
		}

		return gast.WalkContinue, nil
	})
}
```

Adicione também em `ast.go` as funções `tagsFromFrontmatter`, `aliasesFromFrontmatter`, `titleFromFrontmatter` e `dedupeTags`:

- `tagsFromFrontmatter` aceita `tags` e `tag`, cada um como string única, string com vírgulas, ou lista. Cada valor entra sem o `#` inicial.
- `aliasesFromFrontmatter` aceita `aliases` e `alias`, nas mesmas formas. É o insumo de RF-62.
- `titleFromFrontmatter` lê a chave `title`, apenas se for string.
- `dedupeTags` ordena e remove duplicatas de `note.Tags`, preservando a grafia da primeira ocorrência.

- [ ] **Step 5: Criar stubs para as extensões seguintes**

Para compilar antes das Tasks 15 a 17, crie `BlockIDExtension`, `TagExtension`, `InlineFieldExtension` como structs vazios com `Extend(goldmark.Markdown) {}`, e `BlockIDNode`, `TagNode`, `InlineFieldNode` como structs mínimos embutindo `gast.BaseInline`. Serão substituídos.

- [ ] **Step 6: Rodar para confirmar que passa**

Run: `go test -race ./internal/parser/ -run TestWikilink -v`
Esperado: PASS, treze subcasos.

- [ ] **Step 7: Commit**

```bash
git add internal/parser
git commit -m "feat(parser): goldmark inline parser for wikilinks and embeds"
```

---

