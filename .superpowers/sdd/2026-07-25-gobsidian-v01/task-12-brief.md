### Task 12: Tipos do parser, frontmatter e slug

**Files:**
- Create: `internal/parser/types.go`, `internal/parser/frontmatter.go`, `internal/parser/slug.go`
- Create: `internal/parser/frontmatter_test.go`, `internal/parser/slug_test.go`

**Interfaces:**
- Consumes: nada
- Produces: `parser.ParsedNote`, `parser.Heading`, `parser.Block`, `parser.Link`, `parser.LinkKind` (`LinkWiki`, `LinkEmbed`, `LinkMarkdown`); `parser.SplitFrontmatter(data []byte) (fm []byte, body []byte, bodyOffset int64)`; `parser.DecodeFrontmatter(fm []byte) (map[string]any, error)`; `parser.Slug(text string) string`

- [ ] **Step 1: Escrever os testes**

`internal/parser/slug_test.go`:

```go
package parser_test

import (
	"testing"

	"github.com/jonyd/gobsidian/internal/parser"
)

// O slug existe para casar o texto do heading com a ancora de um wikilink.
// [[nota#Capitulo 118]] tem que achar "## Capítulo 118".
func TestSlug(t *testing.T) {
	tests := []struct{ in, want string }{
		{"Capítulo 118", "capitulo 118"},
		{"CAPÍTULO 118", "capitulo 118"},
		{"  Capitulo   118  ", "capitulo 118"},
		{"Ação & Reação", "acao reacao"},
		{"Art. 1.234 — CPC", "art 1234 cpc"},
		{"", ""},
	}

	for _, tt := range tests {
		if got := parser.Slug(tt.in); got != tt.want {
			t.Errorf("Slug(%q) = %q, quer %q", tt.in, got, tt.want)
		}
	}
}
```

`internal/parser/frontmatter_test.go`:

```go
package parser_test

import (
	"testing"

	"github.com/jonyd/gobsidian/internal/parser"
)

func TestSplitFrontmatter(t *testing.T) {
	tests := []struct {
		name       string
		in         string
		wantFM     string
		wantBody   string
		wantOffset int64
	}{
		{
			name:       "presente",
			in:         "---\ntitle: A\n---\n# Corpo\n",
			wantFM:     "title: A\n",
			wantBody:   "# Corpo\n",
			wantOffset: 17,
		},
		{
			name:     "ausente",
			in:       "# Corpo\n",
			wantFM:   "",
			wantBody: "# Corpo\n",
		},
		{
			name:     "tres tracos no meio nao conta",
			in:       "# Corpo\n---\nnao e frontmatter\n",
			wantFM:   "",
			wantBody: "# Corpo\n---\nnao e frontmatter\n",
		},
		{
			name:     "delimitador nao fechado",
			in:       "---\ntitle: A\n# Corpo\n",
			wantFM:   "",
			wantBody: "---\ntitle: A\n# Corpo\n",
		},
		{
			name:     "frontmatter vazio",
			in:       "---\n---\n# Corpo\n",
			wantFM:   "",
			wantBody: "# Corpo\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fm, body, off := parser.SplitFrontmatter([]byte(tt.in))
			if string(fm) != tt.wantFM {
				t.Errorf("fm = %q, quer %q", fm, tt.wantFM)
			}
			if string(body) != tt.wantBody {
				t.Errorf("body = %q, quer %q", body, tt.wantBody)
			}
			if tt.wantOffset != 0 && off != tt.wantOffset {
				t.Errorf("offset = %d, quer %d", off, tt.wantOffset)
			}
		})
	}
}

func TestDecodeFrontmatterPreservesTypes(t *testing.T) {
	fm := []byte("titulo: Ponto 3\nnumero: 42\nativo: true\ntags:\n  - civil\n  - obrigacoes\naliases: [P3, Ponto III]\ndata: 2026-07-25\n")

	got, err := parser.DecodeFrontmatter(fm)
	if err != nil {
		t.Fatalf("DecodeFrontmatter: %v", err)
	}

	if got["titulo"] != "Ponto 3" {
		t.Errorf("titulo = %v (%T), quer string", got["titulo"], got["titulo"])
	}
	if n, ok := got["numero"].(int); !ok || n != 42 {
		t.Errorf("numero = %v (%T), quer int 42", got["numero"], got["numero"])
	}
	if b, ok := got["ativo"].(bool); !ok || !b {
		t.Errorf("ativo = %v (%T), quer bool true", got["ativo"], got["ativo"])
	}
	if tags, ok := got["tags"].([]any); !ok || len(tags) != 2 {
		t.Errorf("tags = %v (%T), quer lista de 2", got["tags"], got["tags"])
	}
	if aliases, ok := got["aliases"].([]any); !ok || len(aliases) != 2 {
		t.Errorf("aliases = %v (%T), quer lista de 2", got["aliases"], got["aliases"])
	}
}

// Frontmatter malformado nao pode derrubar o parse: a nota ainda tem corpo,
// headings e links uteis. O erro e reportado e o resto segue.
func TestDecodeFrontmatterMalformedReturnsError(t *testing.T) {
	if _, err := parser.DecodeFrontmatter([]byte("a: [1, 2\nb: :\n")); err == nil {
		t.Fatal("frontmatter malformado deveria devolver erro")
	}
}
```

- [ ] **Step 2: Rodar para confirmar que falha**

Run: `go test ./internal/parser/ -v`
Esperado: FAIL — `undefined: parser.Slug`.

- [ ] **Step 3: Implementar os tipos**

`internal/parser/types.go`:

```go
// Package parser transforma bytes de uma nota em estrutura. E puro: sem I/O,
// sem estado, sem conhecimento do indice ou do cofre. Recebe []byte, devolve
// ParsedNote. Isso o torna trivialmente testavel por golden file e
// trivialmente paralelizavel.
package parser

type LinkKind int

const (
	LinkWiki LinkKind = iota
	LinkEmbed
	LinkMarkdown
)

func (k LinkKind) String() string {
	switch k {
	case LinkEmbed:
		return "embed"
	case LinkMarkdown:
		return "markdown"
	default:
		return "wikilink"
	}
}

type Heading struct {
	Level int    `json:"level"`
	Text  string `json:"text"`
	Slug  string `json:"slug"`
	// Start e o offset do '#'. End e o offset do fim da secao — o inicio do
	// proximo heading de nivel menor ou igual, ou o fim do arquivo.
	Start int64 `json:"start"`
	End   int64 `json:"end"`
	// BodyStart e o offset logo apos a linha do heading. E o que
	// replace_section usa: preserva o titulo e substitui so o que vem abaixo.
	BodyStart int64 `json:"body_start"`
}

type Block struct {
	ID    string `json:"id"` // sem o '^'
	Start int64  `json:"start"`
	End   int64  `json:"end"`
}

type Link struct {
	Raw    string   `json:"raw"` // texto original, para reescrita fiel
	Target string   `json:"target"`
	Alias  string   `json:"alias,omitempty"`
	Anchor string   `json:"anchor,omitempty"` // heading ou ^bloco
	Kind   LinkKind `json:"kind"`
	Start  int64    `json:"start"`
	End    int64    `json:"end"`
}

type ParsedNote struct {
	Frontmatter map[string]any      `json:"frontmatter,omitempty"`
	Tags        []string            `json:"tags,omitempty"`
	Aliases     []string            `json:"aliases,omitempty"`
	Headings    []Heading           `json:"headings,omitempty"`
	Blocks      []Block             `json:"blocks,omitempty"`
	Links       []Link              `json:"links,omitempty"`
	Inline      map[string][]string `json:"inline,omitempty"`
	// Title vem do frontmatter, do primeiro H1, ou fica vazio para que o
	// chamador use o nome do arquivo. O parser nao conhece o nome do arquivo.
	Title string `json:"title,omitempty"`
	// FrontmatterErr registra frontmatter malformado sem abortar o parse.
	FrontmatterErr string `json:"frontmatter_err,omitempty"`
}
```

- [ ] **Step 4: Implementar slug e frontmatter**

`internal/parser/slug.go`:

```go
package parser

import (
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// Slug normaliza um heading para comparacao com a ancora de um wikilink:
// minusculas, sem acentos, sem pontuacao, espacos colapsados.
//
// O Obsidian casa ancora de forma mais permissiva do que igualdade textual,
// e reproduzir isso e o que faz [[nota#Capitulo 118]] encontrar
// "## Capítulo 118" escrito com acento.
func Slug(text string) string {
	stripped, _, err := transform.String(
		transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC),
		text,
	)
	if err != nil {
		stripped = text
	}

	var b strings.Builder
	b.Grow(len(stripped))

	lastSpace := true
	for _, r := range strings.ToLower(stripped) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			lastSpace = false
		case unicode.IsSpace(r):
			if !lastSpace && b.Len() > 0 {
				b.WriteByte(' ')
				lastSpace = true
			}
		default:
			// Pontuacao vira nada, nao vira espaco: "Art. 1.234" precisa
			// virar "art 1234", nao "art 1 234".
		}
	}

	return strings.TrimSpace(b.String())
}
```

`internal/parser/frontmatter.go`:

```go
package parser

import (
	"bytes"
	"fmt"

	"gopkg.in/yaml.v3"
)

var fmDelim = []byte("---")

// SplitFrontmatter separa o bloco YAML do corpo, devolvendo tambem o offset
// em que o corpo comeca no arquivo original. O offset e o que mantem todos os
// offsets de heading e de bloco corretos em relacao ao arquivo, nao ao corpo.
func SplitFrontmatter(data []byte) ([]byte, []byte, int64) {
	if !bytes.HasPrefix(data, fmDelim) {
		return nil, data, 0
	}

	// A primeira linha tem que ser exatamente "---" (com CR opcional).
	firstNL := bytes.IndexByte(data, '\n')
	if firstNL < 0 {
		return nil, data, 0
	}
	first := bytes.TrimRight(data[:firstNL], "\r")
	if !bytes.Equal(first, fmDelim) {
		return nil, data, 0
	}

	rest := data[firstNL+1:]
	offset := int64(firstNL + 1)

	for len(rest) > 0 {
		nl := bytes.IndexByte(rest, '\n')
		var line []byte
		var advance int
		if nl < 0 {
			line, advance = rest, len(rest)
		} else {
			line, advance = rest[:nl], nl+1
		}

		if bytes.Equal(bytes.TrimRight(line, "\r"), fmDelim) {
			fmEnd := int(offset) - (firstNL + 1)
			body := rest[advance:]
			return data[firstNL+1 : firstNL+1+fmEnd], body, offset + int64(advance)
		}

		rest = rest[advance:]
		offset += int64(advance)
	}

	// Delimitador de abertura sem fechamento: nao ha frontmatter, ha um
	// arquivo que comeca com tres tracos.
	return nil, data, 0
}

// DecodeFrontmatter decodifica o YAML preservando os tipos do Obsidian:
// string, numero, booleano, lista e data.
func DecodeFrontmatter(fm []byte) (map[string]any, error) {
	if len(bytes.TrimSpace(fm)) == 0 {
		return nil, nil
	}

	var out map[string]any
	if err := yaml.Unmarshal(fm, &out); err != nil {
		return nil, fmt.Errorf("decodificando frontmatter: %w", err)
	}
	return out, nil
}
```

**Sutileza de offset.** `SplitFrontmatter` devolve o offset do corpo porque todos os offsets em `Heading` e `Block` são relativos ao **arquivo**, não ao corpo. Errar isso faz `note_read` de uma seção devolver bytes deslocados exatamente pelo tamanho do frontmatter — um bug que só aparece em notas que têm frontmatter, ou seja, quase todas.

- [ ] **Step 5: Rodar para confirmar que passa**

Run: `go test -race ./internal/parser/ -v`
Esperado: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/parser
git commit -m "feat(parser): note types, frontmatter split with body offset, anchor slugs"
```

---

