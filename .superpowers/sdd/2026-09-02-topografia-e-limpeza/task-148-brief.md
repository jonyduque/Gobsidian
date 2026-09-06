### Task 148: `note_read` em lote carrega `section_synthetic` como a leitura simples (1.2)

`ReadResult.SectionSynthetic` existe desde a promoção de candidatos; `ReadNoteItem`
(o item do lote) não tem o campo, e `ReadNotes` não o copia. Um lote que lê seis
capítulos por candidato devolve seis `section` sem dizer que são palpite —
exatamente a afirmação de estrutura que `TOOLS.md:110` diz que a tool não faz.

**Files:**
- Modify: `internal/service/read.go:148-158` (`ReadNoteItem`), `:162-171` (`readNoteItemWire`), `:181-193` (`MarshalJSON`), `:218-226` (`ReadNotes`)
- Modify: `docs/TOOLS.md` (parágrafo do retorno em lote de `note_read` — acrescentar `section_synthetic` à lista de campos por item)
- Test: `internal/service/lote_por_item_test.go` (acrescentar)

**Interfaces:**
- Produces: `ReadNoteItem.SectionSynthetic bool` com tag `json:"section_synthetic,omitempty"`.

- [ ] **Step 1: Teste que falha**

Ao fim de `internal/service/lote_por_item_test.go`:

```go
// TestReadNotesLotePropagaSectionSynthetic: o item do lote nasceu sem o campo
// que a leitura simples ja tinha. Sem ele, seis secoes por candidato numa
// chamada so chegam como estrutura afirmada.
func TestReadNotesLotePropagaSectionSynthetic(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "conv.md",
		"**13 Registro**\n\ntexto do capitulo\n\n**13.1 Substituicao**\n\no texto da subsecao\n")
	writeFile(t, root, "real.md",
		"# 13 Registro\n\ntexto do capitulo\n\n## 13.1 Substituicao\n\no texto da subsecao\n")
	svc := newTestService(t, root)

	out := svc.ReadNotes(context.Background(), ReadBatchRequest{
		Heading: "13.1 Substituicao",
		Alvos:   []ReadAlvo{{Path: "conv.md"}, {Path: "real.md"}},
	})
	if len(out.Items) != 2 {
		t.Fatalf("items = %d, quer 2", len(out.Items))
	}
	for i, it := range out.Items {
		if it.Err != nil {
			t.Fatalf("item %d: %v", i, it.Err)
		}
	}
	if !out.Items[0].SectionSynthetic {
		t.Error("conv.md: secao veio de candidato e o item do lote nao diz section_synthetic")
	}
	if out.Items[1].SectionSynthetic {
		t.Error("real.md: heading ATX de verdade marcado como sintetico")
	}

	b, err := json.Marshal(out.Items[0])
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"section_synthetic":true`) {
		t.Errorf("MarshalJSON nao serializa o campo: %s", b)
	}
}
```

Acrescente `"encoding/json"` aos imports do arquivo.

- [ ] **Step 2: Rodar; falha de compilação**

Run: `go test ./internal/service -run TestReadNotesLotePropagaSectionSynthetic`
Expected: `out.Items[0].SectionSynthetic undefined`.

- [ ] **Step 3: Campo nos dois structs, cópia nos dois lugares**

Em `ReadNoteItem` e em `readNoteItemWire`, depois de `Section`:

```go
	SectionSynthetic bool `json:"section_synthetic,omitempty"`
```

Em `MarshalJSON`, depois de `Section: i.Section,`:

```go
		SectionSynthetic: i.SectionSynthetic,
```

Em `ReadNotes`, depois de `Section: res.Section,`:

```go
			SectionSynthetic: res.SectionSynthetic,
```

`gofmt -w internal/service/read.go` realinha as tags.

- [ ] **Step 4: Rodar; passa. Prova de mutação: apague a linha de `ReadNotes`, rode, o teste nomeia `conv.md`; restaure. Apague a de `MarshalJSON`, rode, o teste nomeia `MarshalJSON`; restaure.**

- [ ] **Step 5: `TOOLS.md`**

No parágrafo do retorno em lote de `note_read` (procure `items` na seção de `note_read`), acrescente `section_synthetic` à lista de campos por item, com a mesma frase da leitura simples: "acompanha toda seção vinda de candidato".

- [ ] **Step 6: Gate e commit**

```bash
pwsh -File scripts/verify.ps1
git add internal/service/read.go internal/service/lote_por_item_test.go docs/TOOLS.md
git commit -m "fix(service): batch note_read carries section_synthetic like the single read does"
```

#### Verificações
Além dos passos:
1. `readNoteItemWire` e `ReadNoteItem` têm o campo com a MESMA tag JSON (`section_synthetic,omitempty`); `note_read` simples e em lote produzem o mesmo JSON para a mesma nota — cole os dois JSONs no relatório.
2. `docs/TOOLS.md` descreve `section_synthetic` no retorno em lote; `pwsh -File scripts/check_doc_refs.ps1` verde.
3. `go test ./internal/service -run 'ReadNote|Lote' -v` sem falha.

#### Regras de execução
- Gate: `pwsh -File scripts/verify.ps1` verde antes do commit, com a contagem de etapas colada no relatório. `-SkipCross -SkipNet` só para iterar.
- Nunca `git checkout`, `git restore`, `git stash`, `git clean` nem `git reset`. Há trabalho não commitado no repositório (`test-vault/`, `.claude/skills/`, `Resume-Claude.ps1`): `git diff <caminho>` antes de `git add <caminho>`; nunca `git add -A`.
- Nunca `go mod tidy`.
- Nunca despache subagentes. Nunca mate processo por nome (`Stop-Process -Name`); só por PID que você lançou.
- Referências e renames por gopls (LSP), não por grep. `grep` só para confirmar tags e reflection.
- Se um teste falhar por motivo que este brief não explica, **pare e reporte `BLOCKED`**; não ajuste a expectativa para o código passar.
- Este commit é de UMA categoria (o prefixo do assunto diz qual). Se você se pegar corrigindo outra coisa no caminho, anote em "O que ficou de fora" e não corrija.

#### Comando de mutação
Copie a âncora **do arquivo**, não de memória — âncora digitada sai `EXIT=2`. `0` = o teste reprovou sob mutação (o que se quer); `1` = a regra está escrita e não verificada; `2` = âncora ambígua ou build quebrado.

```bash
pwsh -File scripts/mutate.ps1 -Path internal/service/read.go `
  -Anchor 'SectionSynthetic: it.SectionSynthetic,' `
  -Replacement '' `
  -Test TestReadNotesLotePropagaSectionSynthetic -Package ./internal/service/
```
(A âncora exata depende de onde a cópia ficou — `MarshalJSON` ou `ReadNotes`. Copie do arquivo.)

#### Contrato de relatório
Escreva o relatório completo no arquivo de relatório indicado no despacho; devolva no chat só status, SHA, uma linha de testes e as preocupações. O relatório traz:
- **Status** — `DONE` | `DONE_WITH_CONCERNS` | `BLOCKED` | `NEEDS_CONTEXT`.
- **Commit** — SHA curto e assunto (`git log -1 --oneline`).
- **Evidência de TDD** — comando do RED com a saída falhando; comando do GREEN com a saída passando. Não "segui TDD".
- **Prova de mutação** — para cada regra reivindicada: o comando `mutate.ps1` (ou a mutação manual, com o diff), qual teste reprovou **pelo nome e pela linha**, a saída colada, e a confirmação de restauro (`git diff --stat` limpo no arquivo mutado).
- **As verificações do brief** — cada uma com o resultado real, inclusive as que deram certo.
- **`verify.ps1`** — a última linha, com a contagem de etapas.
- **O que ficou de fora** — e por quê. Vazio é resposta aceitável; ausente não é.
- **`git status --porcelain`** — colado. Nenhum arquivo do usuário tocado.

---

