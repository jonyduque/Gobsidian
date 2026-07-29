### Task 17: Extensão goldmark — campos inline do Dataview

**Files:**
- Create: `internal/parser/ext_inline_field.go` (substitui o stub), `internal/parser/ext_inline_field_test.go`

**Interfaces:**
- Consumes: nada
- Produces: `parser.InlineFieldExtension`, `parser.InlineFieldNode{Key, Value string}`

- [ ] **Step 1: Escrever o teste**

```go
func TestInlineFields(t *testing.T) {
	src := "autor:: Fulano\nano:: 2026\n\n[status:: revisado]\n\nnao e campo: valor\n"

	note, err := parser.Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if got := note.Inline["autor"]; len(got) != 1 || got[0] != "Fulano" {
		t.Errorf("autor = %v, quer [Fulano]", got)
	}
	if got := note.Inline["ano"]; len(got) != 1 || got[0] != "2026" {
		t.Errorf("ano = %v, quer [2026]", got)
	}
	if got := note.Inline["status"]; len(got) != 1 || got[0] != "revisado" {
		t.Errorf("status = %v, quer [revisado] — a forma entre colchetes conta", got)
	}
	if _, ok := note.Inline["nao e campo"]; ok {
		t.Error("dois-pontos simples nao e campo inline")
	}
}

func TestInlineFieldRepeatedKey(t *testing.T) {
	src := "tema:: prescricao\ntema:: decadencia\n"

	note, err := parser.Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got := note.Inline["tema"]; len(got) != 2 {
		t.Fatalf("tema = %v, quer duas ocorrencias", got)
	}
}

func TestInlineFieldNotInCode(t *testing.T) {
	note, err := parser.Parse([]byte("```\nautor:: Fulano\n```\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(note.Inline) != 0 {
		t.Errorf("inline = %v, quer vazio", note.Inline)
	}
}
```

- [ ] **Step 2: Rodar para confirmar que falha**

Run: `go test ./internal/parser/ -run TestInlineField -v`
Esperado: FAIL.

- [ ] **Step 3: Implementar**

`internal/parser/ext_inline_field.go`, com `Trigger() []byte{':', '['}` e prioridade 130. Regras:

1. Exatamente **dois** `:` seguidos. Um só é pontuação normal e não pode virar campo — é o caso mais comum de falso positivo.
2. A chave é o texto imediatamente à esquerda, até o início da linha ou até `[`. Aceita letras, dígitos, espaço, `-` e `_`.
3. O valor vai até o fim da linha, ou até `]` na forma entre colchetes.
4. Chave repetida acumula em lista — é por isso que `Inline` é `map[string][]string` e não `map[string]string`.
5. RF-18 é P1: se esta extensão atrasar, ela pode ser adiada sem bloquear a v0.1. Nesse caso, registre `InlineFieldExtension` como no-op e mantenha os testes marcados com `t.Skip`.

- [ ] **Step 4: Rodar para confirmar que passa**

Run: `go test -race ./internal/parser/ -v`
Esperado: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/parser
git commit -m "feat(parser): dataview inline fields with repeated-key accumulation"
```

---

