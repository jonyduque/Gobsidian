### Task 13: Headings com offsets de seção

**Files:**
- Create: `internal/parser/headings.go`, `internal/parser/headings_test.go`

**Interfaces:**
- Consumes: `parser.Heading`, `parser.Slug` (Task 12)
- Produces: `parser.ExtractHeadings(body []byte, bodyOffset int64) []Heading`

- [ ] **Step 1: Escrever o teste**

`internal/parser/headings_test.go`:

```go
package parser_test

import (
	"testing"

	"github.com/jonyd/gobsidian/internal/parser"
)

func TestExtractHeadingsSectionBoundaries(t *testing.T) {
	body := "# Titulo\ntexto a\n\n## Cap 1\ntexto b\n\n### Sub\ntexto c\n\n## Cap 2\ntexto d\n"

	hs := parser.ExtractHeadings([]byte(body), 0)

	if len(hs) != 4 {
		t.Fatalf("headings = %d, quer 4: %+v", len(hs), hs)
	}

	// "## Cap 1" termina onde "## Cap 2" comeca — inclui "### Sub", que e
	// subsecao dele, e NAO vai ate o fim do arquivo.
	cap1 := hs[1]
	if cap1.Text != "Cap 1" || cap1.Level != 2 {
		t.Fatalf("hs[1] = %+v", cap1)
	}
	cap2Start := hs[3].Start
	if cap1.End != cap2Start {
		t.Errorf("Cap 1 End = %d, quer %d (inicio de Cap 2)", cap1.End, cap2Start)
	}

	// "### Sub" termina no proximo heading de nivel <= 3, que e "## Cap 2".
	if hs[2].End != cap2Start {
		t.Errorf("Sub End = %d, quer %d", hs[2].End, cap2Start)
	}

	// O ultimo heading vai ate o fim.
	if hs[3].End != int64(len(body)) {
		t.Errorf("Cap 2 End = %d, quer %d", hs[3].End, len(body))
	}

	// BodyStart pula a linha do titulo.
	if got := body[cap1.BodyStart:cap1.End]; got != "texto b\n\n### Sub\ntexto c\n\n" {
		t.Errorf("corpo de Cap 1 = %q", got)
	}
}

func TestExtractHeadingsRespectsBodyOffset(t *testing.T) {
	body := "# Titulo\n"
	hs := parser.ExtractHeadings([]byte(body), 100)

	if len(hs) != 1 {
		t.Fatalf("headings = %d, quer 1", len(hs))
	}
	if hs[0].Start != 100 {
		t.Errorf("Start = %d, quer 100 — offsets sao relativos ao arquivo", hs[0].Start)
	}
}

func TestExtractHeadingsIgnoresCodeBlocks(t *testing.T) {
	body := "# Real\n\n```\n# Nao e heading\n```\n\n## Tambem real\n"

	hs := parser.ExtractHeadings([]byte(body), 0)

	if len(hs) != 2 {
		t.Fatalf("headings = %d, quer 2: %+v", len(hs), hs)
	}
	if hs[1].Text != "Tambem real" {
		t.Errorf("hs[1].Text = %q", hs[1].Text)
	}
}

func TestExtractHeadingsCRLF(t *testing.T) {
	body := "# A\r\ntexto\r\n## B\r\n"

	hs := parser.ExtractHeadings([]byte(body), 0)

	if len(hs) != 2 {
		t.Fatalf("headings = %d, quer 2", len(hs))
	}
	if hs[0].Text != "A" || hs[1].Text != "B" {
		t.Errorf("textos = %q, %q — CR nao foi removido", hs[0].Text, hs[1].Text)
	}
}
```

- [ ] **Step 2: Rodar para confirmar que falha**

Run: `go test ./internal/parser/ -run TestExtractHeadings -v`
Esperado: FAIL — `undefined: parser.ExtractHeadings`.

- [ ] **Step 3: Implementar**

`internal/parser/headings.go`:

```go
package parser

import (
	"bytes"
	"strings"
)

// ExtractHeadings percorre o corpo linha a linha e devolve a hierarquia com
// os offsets de secao ja calculados.
//
// A varredura e propria em vez de vir da AST do goldmark por um motivo:
// precisamos do offset de FIM de secao, que e uma propriedade da hierarquia,
// nao do no. O goldmark da a posicao de cada heading; o fim de uma secao e o
// inicio do proximo heading de nivel menor ou igual, e calcular isso exige
// uma passada com pilha de qualquer forma.
func ExtractHeadings(body []byte, bodyOffset int64) []Heading {
	var out []Heading

	inFence := false
	var fenceMarker string

	pos := int64(0)
	for len(body) > 0 {
		nl := bytes.IndexByte(body, '\n')
		var line []byte
		var advance int64
		if nl < 0 {
			line, advance = body, int64(len(body))
		} else {
			line, advance = body[:nl], int64(nl)+1
		}

		trimmed := bytes.TrimRight(line, "\r")
		text := string(trimmed)

		// Blocos de codigo cercados. Um '#' dentro de um deles nao e heading,
		// e ignorar isso e a forma mais rapida de produzir hierarquia falsa.
		if marker := fenceMarkerOf(text); marker != "" {
			if !inFence {
				inFence, fenceMarker = true, marker
			} else if strings.HasPrefix(strings.TrimSpace(text), fenceMarker) {
				inFence, fenceMarker = false, ""
			}
			pos += advance
			body = body[advance:]
			continue
		}

		if !inFence {
			if level, title, ok := parseATXHeading(text); ok {
				out = append(out, Heading{
					Level:     level,
					Text:      title,
					Slug:      Slug(title),
					Start:     bodyOffset + pos,
					BodyStart: bodyOffset + pos + advance,
				})
			}
		}

		pos += advance
		body = body[advance:]
	}

	total := bodyOffset + pos
	closeSections(out, total)
	return out
}

// closeSections preenche End: o inicio do proximo heading de nivel menor ou
// igual, ou o fim do arquivo.
func closeSections(hs []Heading, total int64) {
	for i := range hs {
		hs[i].End = total
		for j := i + 1; j < len(hs); j++ {
			if hs[j].Level <= hs[i].Level {
				hs[i].End = hs[j].Start
				break
			}
		}
	}
}

func parseATXHeading(line string) (int, string, bool) {
	// Ate tres espacos de indentacao sao permitidos pelo CommonMark.
	trimmed := strings.TrimLeft(line, " ")
	if len(line)-len(trimmed) > 3 {
		return 0, "", false
	}

	level := 0
	for level < len(trimmed) && trimmed[level] == '#' {
		level++
	}
	if level == 0 || level > 6 {
		return 0, "", false
	}
	if level < len(trimmed) && trimmed[level] != ' ' && trimmed[level] != '\t' {
		return 0, "", false
	}

	title := strings.TrimSpace(trimmed[level:])
	// Fechamento opcional: "## Titulo ##".
	title = strings.TrimRight(title, "#")
	return level, strings.TrimSpace(title), true
}

func fenceMarkerOf(line string) string {
	t := strings.TrimSpace(line)
	switch {
	case strings.HasPrefix(t, "```"):
		return "```"
	case strings.HasPrefix(t, "~~~"):
		return "~~~"
	}
	return ""
}
```

- [ ] **Step 4: Rodar para confirmar que passa**

Run: `go test -race ./internal/parser/ -v`
Esperado: PASS, quatro testes de heading.

- [ ] **Step 5: Commit**

```bash
git add internal/parser
git commit -m "feat(parser): heading hierarchy with section end offsets"
```

---

