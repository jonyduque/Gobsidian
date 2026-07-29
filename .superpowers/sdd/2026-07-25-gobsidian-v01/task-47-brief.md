### Task 47: `internal/search/snippet.go` — trecho com destaque

RF-22, P0. É a tarefa de offset deste marco, e offset é o que este projeto já errou mais de uma vez.

#### Onde isto encaixa

Recorta, do arquivo em disco, o pedaço em volta das ocorrências que a Task 45 guardou. `snippet_chars` já está no schema de `docs/TOOLS.md` com default 240 e máximo 1000.

#### A armadilha que domina esta tarefa

**Offset do parser é relativo ao corpo depois de `vault.StripBOM`; o arquivo em disco tem o BOM.** Ler o disco com um offset do parser, numa nota com BOM, recorta 3 bytes deslocado — e o sintoma é um trecho que começa no meio de uma palavra, que parece erro de recorte e não erro de offset.

`internal/parser/types.go` já carrega `ShiftOffsets`, e `internal/index` já o chama quando `hadBOM`. Confira o que o `index.Note` guarda: se os offsets já vêm deslocados, somar de novo erra na direção oposta. **Invoque a skill `preventing-false-pass-and-offset-bugs` antes de escrever a primeira linha.**

**Teste em memória não prova isto.** Afirmar que a estrutura do trecho existe não prova que `vault.ReadRange` traz os bytes certos. O teste tem de ler **do disco**, de um arquivo real, e comparar a fatia exata.

#### O que já está fechado

- **`vault.ReadRange` já existe** e é quem lê intervalo sem carregar o arquivo inteiro. Não reimplemente leitura.
- **Arquivo somente-nuvem não é aberto.** Um resultado de busca sobre nota não hidratada não pode disparar download. Se não der para recortar sem abrir, devolva o trecho vazio e **conte**, não force a leitura.
- **`ctx` onde bloqueia.** Ler intervalo de arquivo bloqueia.
- **Colisão de slug é aceita.** Duas seções com o mesmo slug produzem duas ocorrências distintas, cada uma com seu offset. `matched_headings` traz o **texto** do heading, não o slug.

#### O teste que sustenta a tarefa

```go
func TestSnippetOffsetsAlignWithDiskBytesUnderBOM(t *testing.T) {
	root := t.TempDir()
	// BOM + frontmatter + corpo. Os tres deslocamentos que este projeto ja
	// errou, no mesmo arquivo, de proposito.
	raw := []byte("\xEF\xBB\xBF---\ntitle: T\n---\n\n# Secao\n\nA prescricao intercorrente corre.\n")
	caminho := filepath.Join(root, "nota.md")
	if err := os.WriteFile(caminho, raw, 0644); err != nil {
		t.Fatal(err)
	}

	v, err := vault.New(root)
	if err != nil {
		t.Fatal(err)
	}
	idx := index.New()
	if err := idx.Build(context.Background(), v); err != nil {
		t.Fatal(err)
	}

	trecho := snippetPara(t, v, idx, "nota.md", "intercorrente", 240)

	// A assercao e sobre BYTES DO DISCO, nao sobre a estrutura em memoria.
	if !strings.Contains(trecho.Text, "intercorrente") {
		t.Fatalf("trecho = %q; nao contem o termo — offset deslocado pelo BOM "+
			"ou pelo frontmatter", trecho.Text)
	}
	if strings.ContainsRune(trecho.Text, '�') {
		t.Errorf("trecho tem byte invalido: recorte caiu no meio de um caractere")
	}
	// O destaque tem de apontar para o termo, nao para a vizinhanca.
	if got := trecho.Text[trecho.HighlightStart:trecho.HighlightEnd]; got != "intercorrente" {
		t.Errorf("destaque = %q, quer %q", got, "intercorrente")
	}
}
```

#### Verificações além dos passos

- O mesmo teste **sem** BOM continua passando? Os dois casos, não um.
- `snippet_chars` é respeitado? Meça o comprimento; o schema promete 240 por default e 1000 no máximo, e **schema que promete e código que ignora é pior que parâmetro ausente**.
- Termo no primeiro byte do arquivo, e termo no último: o recorte não estoura?
- Duas ocorrências próximas produzem um trecho ou dois? Decida, documente, teste.
- Nota somente-nuvem: nenhum download disparado. Confirme como verificou.
- Recorte em palavra acentuada não parte caractere? É a razão de o teste procurar `�`.

**Prova de mutação obrigatória:** remova o ajuste de BOM e confirme que o teste reprova. Se ele **não** reprovar, o teste está lendo memória e não disco — conserte o teste antes de seguir.

#### Regras de execução e contrato de relatório

Idênticos aos da Task 43. Relatório em `.superpowers/sdd/task-47-report.md`.

**Files:** Create `internal/search/snippet.go`, `internal/search/snippet_test.go`
**Commit:** `feat(search): snippets with term highlight`

---

