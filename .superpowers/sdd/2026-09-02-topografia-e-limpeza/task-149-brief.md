### Task 149: `heading` + `block_id` juntos é `INVALID_ARGUMENT`, não `INTERNAL` (1.6)

`write.go:262` devolve `CodeInternal` para um pedido malformado do cliente.
`INTERNAL` diz ao host que o servidor quebrou — e o host tenta de novo em vez
de corrigir o pedido. É o mesmo achado B4 que já corrigiu o `default` do
`switch` de `mode`, três telas abaixo.

**Files:**
- Modify: `internal/service/write.go:262`
- Test: `internal/service/limites_enums_test.go` (acrescentar)

- [ ] **Step 1: Teste**

```go
func TestPatchNoteHeadingEBlockIDEhInvalidArgument(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "a.md", "# H\n\ntexto ^b1\n")
	svc := newTestService(t, root)

	_, err := svc.PatchNote(context.Background(), PatchNoteRequest{
		Path: "a.md", Heading: "H", BlockID: "b1", Content: "x",
	})
	if err == nil {
		t.Fatal("heading e block_id juntos foram aceitos")
	}
	if got := CodeOf(err); got != CodeInvalidArgument {
		t.Errorf("codigo = %s, queria %s: INTERNAL manda o host tentar de novo o mesmo pedido\nerro: %v",
			got, CodeInvalidArgument, err)
	}
}
```

- [ ] **Step 2: Rodar; falha com `codigo = INTERNAL`**

- [ ] **Step 3: Trocar o código em `write.go:262`**

```go
	if req.Heading != "" && req.BlockID != "" {
		return PatchNoteResult{}, Errorf(CodeInvalidArgument, "heading e block_id sao mutuamente exclusivos em note_patch")
	}
```

- [ ] **Step 4: Rodar; passa. Gate. Commit.**

```bash
git add internal/service/write.go internal/service/limites_enums_test.go
git commit -m "fix(service): heading plus block_id in note_patch is the client's mistake, not the server's"
```

#### Verificações
Além dos passos:
1. A mensagem de erro continua a mesma; só o código mudou. `grep -n "CodeInternal" internal/service/write.go` não lista mais a linha do `heading`+`block_id`.
2. `docs/TOOLS.md` na tabela de erros de `note_patch` diz `INVALID_ARGUMENT` para o par.

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
  -Anchor 'CodeInvalidArgument, "heading e block_id' `
  -Replacement 'CodeInternal, "heading e block_id' `
  -Test TestPatchNoteHeadingEBlockIDEhInvalidArgument -Package ./internal/service/
```
(Copie o texto da mensagem do arquivo; o fragmento acima é aproximado.)

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

