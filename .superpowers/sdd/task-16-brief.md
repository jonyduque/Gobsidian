### Task 16: Extensão goldmark — tags inline hierárquicas

**Files:**
- Create: `internal/parser/ext_tag.go` (substitui o stub), `internal/parser/ext_tag_test.go`

**Interfaces:**
- Consumes: nada
- Produces: `parser.TagExtension`, `parser.TagNode{Name string}`

- [ ] **Step 1: Escrever o teste**

```go
func TestInlineTags(t *testing.T) {
	src := "Nota sobre #civil e #civil/obrigacoes e #proc-civil.\n"

	note, err := parser.Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	want := []string{"civil", "civil/obrigacoes", "proc-civil"}
	if len(note.Tags) != len(want) {
		t.Fatalf("tags = %v, quer %v", note.Tags, want)
	}
	for i := range want {
		if note.Tags[i] != want[i] {
			t.Errorf("tags[%d] = %q, quer %q", i, note.Tags[i], want[i])
		}
	}
}

func TestTagRejections(t *testing.T) {
	tests := []struct{ name, in string }{
		{"heading nao e tag", "# Titulo\n"},
		{"so digitos nao e tag", "item #123\n"},
		{"dentro de codigo", "```\n#civil\n```\n"},
		{"codigo inline", "use `#civil` aqui\n"},
		{"cerquilha isolada", "a # b\n"},
		{"dentro de url", "veja https://x.com/a#secao\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			note, err := parser.Parse([]byte(tt.in))
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if len(note.Tags) != 0 {
				t.Errorf("tags = %v, quer nenhuma", note.Tags)
			}
		})
	}
}

func TestTagsFromFrontmatterMerge(t *testing.T) {
	src := "---\ntags:\n  - civil\n  - penal\n---\nTexto com #civil e #tributario.\n"

	note, err := parser.Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	// civil aparece nas duas fontes e conta uma vez so.
	want := map[string]bool{"civil": true, "penal": true, "tributario": true}
	if len(note.Tags) != len(want) {
		t.Fatalf("tags = %v, quer %d unicas", note.Tags, len(want))
	}
	for _, tag := range note.Tags {
		if !want[tag] {
			t.Errorf("tag inesperada: %q", tag)
		}
	}
}
```

- [ ] **Step 2: Rodar para confirmar que falha**

Run: `go test ./internal/parser/ -run "TestInlineTags|TestTag" -v`
Esperado: FAIL.

- [ ] **Step 3: Implementar**

`internal/parser/ext_tag.go`, mesma forma das anteriores, com `Trigger() []byte{'#'}` e prioridade 120. Regras, uma por caso de teste:

1. O `#` precisa ser precedido por início de linha, espaço ou pontuação de abertura. `a#b` e `url#secao` não são tags.
2. O nome aceita letras, dígitos, `-`, `_` e `/`. Precisa conter **pelo menos uma letra**: `#123` não é tag no Obsidian.
3. `/` produz hierarquia e é preservado literalmente. `civil/obrigacoes` é uma tag, não duas.
4. Um `#` seguido de espaço não é tag — é heading (tratado por `ExtractHeadings`) ou texto.
5. Contexto de código é responsabilidade do goldmark.

- [ ] **Step 4: Rodar para confirmar que passa**

Run: `go test -race ./internal/parser/ -v`
Esperado: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/parser
git commit -m "feat(parser): hierarchical inline tags merged with frontmatter tags"
```

---

