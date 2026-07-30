### Task 57: `internal/writer/section.go` — `note_append` e `note_patch` por heading

RF-31 e RF-32. **É a tarefa de offset deste marco.**

#### A armadilha que domina

Os offsets de `Heading.Start` e `Heading.End` do índice são relativos ao corpo **depois** de `vault.StripBOM`, e o arquivo no disco tem o BOM. Escrever usando offset do parser numa nota com BOM corta 3 bytes deslocado — **e aqui isso não produz um trecho estranho, produz uma nota corrompida.**

`internal/index` já chama `ShiftOffsets` quando `hadBOM`. **Confira o que o `index.Note` guarda antes de somar qualquer coisa:** se os offsets já vêm deslocados, somar de novo erra na direção oposta, e o erro é silencioso porque o arquivo continua sendo Markdown válido. Invoque a skill `preventing-false-pass-and-offset-bugs`.

**Substituir sob um heading preserva o heading e as subseções fora do alvo** (RF-32). `## A` com `### A.1` dentro: substituir o conteúdo de `## A` não pode apagar `### A.1` a menos que ela esteja no intervalo — e decidir qual é o intervalo é a parte que erra. `Heading.End` do índice já define isso; use-o, não recalcule.

**Colisão de slug é aceita** (decisão de 2026-07-29). Dois headings que produzem o mesmo slug: `note_patch` tem de **recusar por ambiguidade**, não escolher um. Escolher em silêncio escreve na seção errada, que é perda de dado. Erro de ambiguidade com os dois headings citados é a resposta.

#### O teste que sustenta a tarefa

```go
func TestPatchSectionUnderBOMAndCRLFWritesTheRightBytes(t *testing.T) {
	root := t.TempDir()
	// BOM + CRLF + frontmatter + subsecao: os quatro deslocamentos que este
	// projeto ja pagou, no mesmo arquivo, de proposito.
	raw := []byte("\xEF\xBB\xBF---\r\ntitle: T\r\n---\r\n\r\n" +
		"# Topo\r\n\r\n## Alvo\r\n\r\nconteudo velho\r\n\r\n### Filha\r\n\r\npreservar\r\n\r\n## Depois\r\n\r\nintacto\r\n")
	alvo := filepath.Join(root, "nota.md")
	if err := os.WriteFile(alvo, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	// ... construir vault e index, chamar PatchSection(ctx, v, idx, "nota.md", "Alvo", "conteudo novo") ...

	depois, err := os.ReadFile(alvo)
	if err != nil {
		t.Fatal(err)
	}
	s := string(depois)

	// BOM byte a byte, nao "comeca com BOM".
	if !bytes.HasPrefix(depois, []byte("\xEF\xBB\xBF")) {
		t.Error("BOM perdido na escrita")
	}
	// CRLF preservado: nenhum \n solto. Um LF sozinho aqui e o diff de arquivo
	// inteiro que RF-38 existe para evitar.
	if regexp.MustCompile(`[^\r]\n`).MatchString(s) {
		t.Error("LF solto: o EOL original era CRLF e nao foi preservado")
	}
	// O heading do alvo sobrevive, o conteudo troca.
	if !strings.Contains(s, "## Alvo\r\n") {
		t.Error("o heading do alvo foi apagado")
	}
	if strings.Contains(s, "conteudo velho") {
		t.Error("o conteudo antigo sobreviveu")
	}
	if !strings.Contains(s, "conteudo novo") {
		t.Error("o conteudo novo nao entrou")
	}
	// O que estava FORA do alvo tem de estar intacto — e a subsecao e o caso
	// que erra: ela esta dentro do intervalo do heading pai.
	if !strings.Contains(s, "### Filha") || !strings.Contains(s, "intacto") {
		t.Errorf("conteudo fora do alvo foi destruido:\n%s", s)
	}
}
```

Se a decisão for que `### Filha` **deve** ser substituída junto (porque está no intervalo de `## Alvo`), inverta a asserção **e diga no relatório qual escolheu e por quê**. O que não pode é o teste não dizer.

#### Verificações além dos passos

- O teste acima, e a variante **sem** BOM, e a variante **LF**. Três, não um.
- Heading duplicado por slug: recusa por ambiguidade com os dois citados?
- `note_append` no fim da nota e no fim de uma seção: os dois?
- Anexar em nota sem `\n` final não junta linhas?
- Nota somente-nuvem: a escrita é recusada, ou hidrata? **Decida e diga.**
- O watcher reindexa depois da escrita? Ponta a ponta: escreva pela tool com o servidor rodando e confirme que `vault_search` acha o conteúdo novo.

**Prova de mutação obrigatória:** remova o ajuste de BOM e confirme que o teste com BOM reprova. Remova a preservação de EOL e confirme que o teste de CRLF reprova. **Se algum dos dois não reprovar, o teste está lendo memória em vez de disco** — conserte o teste antes de seguir.

**Files:** Create `internal/writer/section.go`, `section_test.go`
**Commit:** `feat(writer): patch and append by heading, preserving EOL and BOM`

---

