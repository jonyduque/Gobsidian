### Task 152: Dry-run de `note_move` não fabrica diff vazio da origem nem engole referenciadora ilegível (1.4)

`write.go:518`: `diffs[origem] = UnifiedDiff(from, to, raw, raw, 3)` —
`UnifiedDiff` devolve `""` para textos iguais (`diff.go:166`), então a origem
SEMPRE entra em `diffs` com um diff vazio: um item que diz "esta nota não muda"
sobre a nota que vai mudar de lugar. `:522-531`: `continue` num `ReadFile` ou
`RewriteLinks` que falhou — a referenciadora some do dry-run em silêncio, e
quem lê conclui que ela não seria tocada, quando na execução real ela seria (ou
a execução real falharia).

**Files:**
- Modify: `internal/service/write.go:506-545`
- Modify: `docs/TOOLS.md` (contrato de `diffs` em `note_move`: "um diff por referenciadora reescrita; a origem não entra — mover não altera o conteúdo dela")
- Test: `internal/service/move_test.go` (acrescentar dois testes)

- [ ] **Step 1: Testes**

```go
func TestMoveNoteDryRunNaoFabricaDiffVazioDaOrigem(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "origem.md", "# Origem\n\ncorpo\n")
	writeFile(t, root, "citante.md", "ver [[origem]]\n")
	svc := newTestService(t, root)

	res, err := svc.MoveNote(context.Background(), MoveNoteRequest{
		From: "origem.md", To: "destino.md", UpdateLinks: true, DryRun: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if d, ok := res.Diffs["origem.md"]; ok {
		t.Fatalf("a origem entrou em diffs com %q: um item vazio diz que a nota nao muda", d)
	}
	if d := res.Diffs["citante.md"]; !strings.Contains(d, "destino") {
		t.Fatalf("a referenciadora nao tem diff util: %q", d)
	}
}

func TestMoveNoteDryRunNaoEngoleReferenciadoraIlegivel(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "origem.md", "# Origem\n\ncorpo\n")
	writeFile(t, root, "citante.md", "ver [[origem]]\n")
	svc := newTestService(t, root)
	// Indexada, depois apagada do disco: ReadFile falha no dry-run.
	if err := os.Remove(filepath.Join(root, "citante.md")); err != nil {
		t.Fatal(err)
	}

	res, err := svc.MoveNote(context.Background(), MoveNoteRequest{
		From: "origem.md", To: "destino.md", UpdateLinks: true, DryRun: true,
	})
	if err == nil {
		t.Fatalf("dry-run devolveu sucesso com uma referenciadora ilegivel; diffs=%v", res.Diffs)
	}
	if !strings.Contains(err.Error(), "citante.md") {
		t.Fatalf("o erro nao nomeia a referenciadora: %v", err)
	}
}
```

- [ ] **Step 2: Rodar; o primeiro falha em "a origem entrou em diffs", o segundo em "devolveu sucesso"**

- [ ] **Step 3: Reescrever o bloco do dry-run**

Substitua `write.go:506-545` (de `if req.DryRun {` até o `return` do dry-run) por:

```go
	if req.DryRun {
		// A origem nao entra em diffs: mover nao altera o conteudo dela, e
		// UnifiedDiff de um texto contra ele mesmo e "" — um item vazio que
		// dizia "esta nota nao muda" sobre a nota que muda de lugar. A
		// leitura continua para que uma origem ilegivel falhe aqui, e nao so
		// na execucao real.
		absFrom := s.vault.Abs(canonicalFrom)
		if _, err := os.ReadFile(absFrom); err != nil {
			return MoveNoteResult{}, Errorf(CodeInternal,
				"lendo nota de origem %q para o dry-run: %v", canonicalFrom, err)
		}

		diffs := make(map[string]string, len(affectedNotes))
		for refPath, replacements := range affectedNotes {
			raw, err := os.ReadFile(s.vault.Abs(refPath))
			if err != nil {
				// Ate 2026-09-02 era `continue`: a referenciadora sumia do
				// dry-run e quem lia concluia que ela nao seria tocada.
				return MoveNoteResult{}, Errorf(CodeInternal,
					"lendo referenciadora %q para o dry-run: %v", refPath, err)
			}
			rewritten, err := writer.RewriteLinks(raw, replacements)
			if err != nil {
				return MoveNoteResult{}, Errorf(CodeInternal,
					"reescrevendo links de %q para o dry-run: %v", refPath, err)
			}
			diffs[string(refPath)] = writer.UnifiedDiff(string(refPath), string(refPath), string(raw), string(rewritten), 3)
		}

		return MoveNoteResult{
			From:          string(canonicalFrom),
			To:            string(canonicalTo),
			Rewritten:     nil,
			LinksUpdated:  totalLinks,
			BrokenAnchors: brokenAnchors,
			DryRun:        true,
			Diffs:         diffs,
		}, nil
	}
```

- [ ] **Step 4: Rodar `go test ./internal/service -run 'MoveNote|MoveDryRun|Move_' -v`; os dois novos passam; `TestMoveDryRunNaoApresentaDiffVazioComoResultado` (Windows, origem travada) continua passando porque a leitura da origem ficou.**

- [ ] **Step 5: `TOOLS.md`** — na seção de `note_move`, a linha de `diffs` passa a: "`diffs`: mapa caminho → diff unificado, uma entrada por referenciadora que seria reescrita. A origem não entra: mover não altera o conteúdo dela. Uma referenciadora ilegível é erro do dry-run, não omissão."

- [ ] **Step 6: Gate e commit**

```bash
git add internal/service/write.go internal/service/move_test.go docs/TOOLS.md
git commit -m "fix(service): note_move dry-run drops the empty origin diff and fails on an unreadable referrer"
```

#### Verificações
Além dos passos:
1. `TestMoveDryRunNaoApresentaDiffVazioComoResultado` (já existe) continua passando: a origem continua sendo LIDA no dry-run; só o diff vazio dela sai.
2. O erro de referenciadora ilegível nomeia o caminho da referenciadora — cole a mensagem.
3. `docs/TOOLS.md` em `note_move`: `diffs` no dry-run lista só referenciadoras com mudança; origem não aparece.

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
pwsh -File scripts/mutate.ps1 -Path internal/service/write.go `
  -Anchor 'return nil, Errorf(CodeInternal, "lendo referenciadora' `
  -Replacement 'continue; return nil, Errorf(CodeInternal, "lendo referenciadora' `
  -Test TestMoveNoteDryRunNaoEngoleReferenciadoraIlegivel -Package ./internal/service/
```
(Copie a mensagem exata do arquivo; se `continue` não compilar no ponto, troque por `_ = err; continue` num `if` equivalente e cole o diff.)

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

