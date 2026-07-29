### Task 15: Extensão goldmark — identificadores de bloco

**Files:**
- Create: `internal/parser/ext_blockid.go` (substitui o stub), `internal/parser/ext_blockid_test.go`

**Interfaces:**
- Consumes: `parser.Block` (Task 12)
- Produces: `parser.BlockIDExtension`, `parser.BlockIDNode{ID string, Start, End int64}`

- [ ] **Step 1: Escrever o teste**

```go
package parser_test

import (
	"testing"

	"github.com/jonyd/gobsidian/internal/parser"
)

func TestBlockIDExtraction(t *testing.T) {
	src := "Primeiro paragrafo. ^abc123\n\nSegundo paragrafo.\n\nTerceiro. ^def456\n"

	note, err := parser.Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(note.Blocks) != 2 {
		t.Fatalf("blocos = %d, quer 2: %+v", len(note.Blocks), note.Blocks)
	}
	if note.Blocks[0].ID != "abc123" {
		t.Errorf("ID = %q, quer %q — o circunflexo nao faz parte do id", note.Blocks[0].ID, "abc123")
	}

	// Start e End delimitam o BLOCO, nao o marcador: e o que note_read com
	// block_id devolve, e o que note_patch com replace_block substitui.
	got := src[note.Blocks[0].Start:note.Blocks[0].End]
	if got != "Primeiro paragrafo. ^abc123" {
		t.Errorf("bloco = %q", got)
	}
}

func TestBlockIDRejectsNonTerminal(t *testing.T) {
	tests := []struct{ name, in string }{
		{"no meio da linha", "texto ^abc123 mais texto\n"},
		{"dentro de codigo", "```\ntexto ^abc123\n```\n"},
		{"circunflexo sozinho", "texto ^\n"},
		{"caracteres invalidos", "texto ^abc def\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			note, err := parser.Parse([]byte(tt.in))
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if len(note.Blocks) != 0 {
				t.Errorf("blocos = %+v, quer nenhum", note.Blocks)
			}
		})
	}
}
```

- [ ] **Step 2: Rodar para confirmar que falha**

Run: `go test ./internal/parser/ -run TestBlockID -v`
Esperado: FAIL — o stub não produz nós.

- [ ] **Step 3: Implementar**

`internal/parser/ext_blockid.go` segue a mesma forma da Task 14: um `gast.NewNodeKind("BlockID")`, um inline parser com `Trigger() []byte{'^'}`, e um `Extend` que registra com `util.Prioritized(&blockIDParser{}, 100)`.

Regras que o parser precisa aplicar, cada uma correspondendo a um caso do teste:

1. O `^` precisa estar **no fim da linha** — nada além de espaço em branco depois do id. Um `^` no meio é texto literal.
2. O id aceita apenas `[A-Za-z0-9-]`. Espaço encerra o candidato e o invalida.
3. `Start` é o offset do início do **bloco pai** (parágrafo, item de lista), não do `^`. Obtenha-o de `parent.Lines().At(0).Start`; `End` é o fim do marcador.
4. Dentro de bloco de código o parser nunca é oferecido — o goldmark cuida disso, que é o motivo de a extensão existir em vez de uma regex.

- [ ] **Step 4: Rodar para confirmar que passa**

Run: `go test -race ./internal/parser/ -run TestBlockID -v`
Esperado: PASS, cinco subcasos.

- [ ] **Step 5: Commit**

```bash
git add internal/parser
git commit -m "feat(parser): block id extraction with block-level offsets"
```

---

