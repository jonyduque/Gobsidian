### Task 150: `mode` de `note_patch` passa por `ValidarEnum`, e a mensagem lista os modos que existem (1.5 + 5.3)

O `default` do `switch` em `write.go:366-371` diz "aceitos:
replace_heading_and_section, append_to_heading, replace_block, append_to_note".
`append_to_heading` e `append_to_note` NÃO existem como `case`; `replace_section`
existe e não está na lista. A mensagem é o único lugar onde o cliente descobre
os modos, e ela mente. `ValidarEnum` (`errors.go:166`) já é a conta única de
"enum inválido" para as outras tools e produz a lista a partir dos valores
aceitos — impossível divergir.

**Files:**
- Modify: `internal/service/write.go:300-371`
- Modify: `internal/mcpsrv/tools_write.go:35` (descrição do schema: acrescentar "padrao: replace_section, ou replace_block quando block_id vem")
- Modify: `docs/TOOLS.md:403` (o `default` do schema é condicional: diga isso na linha)
- Test: `internal/service/limites_enums_test.go` (acrescentar)

- [ ] **Step 1: Teste**

```go
func TestPatchNoteModeInvalidoListaOsModosQueExistem(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "a.md", "# H\n\ntexto\n")
	svc := newTestService(t, root)

	_, err := svc.PatchNote(context.Background(), PatchNoteRequest{
		Path: "a.md", Heading: "H", Mode: "append_to_heading", Content: "x",
	})
	if err == nil {
		t.Fatal("append_to_heading foi aceito, e nao existe")
	}
	msg := err.Error()
	for _, real := range []string{"replace_section", "replace_heading_and_section", "replace_block"} {
		if !strings.Contains(msg, real) {
			t.Errorf("a mensagem nao lista o modo real %q: %s", real, msg)
		}
	}
	for _, fantasma := range []string{"append_to_heading", "append_to_note"} {
		if strings.Contains(msg, "aceitos") && strings.Contains(msg[strings.Index(msg, "aceitos"):], fantasma) {
			t.Errorf("a mensagem lista o modo fantasma %q como aceito: %s", fantasma, msg)
		}
	}
}
```

- [ ] **Step 2: Rodar; falha em `replace_section` ausente e nos dois fantasmas**

- [ ] **Step 3: Substituir o bloco de default e o `default:` do switch**

Troque `write.go:300-307` por:

```go
	padrao := "replace_section"
	if req.BlockID != "" {
		padrao = "replace_block"
	}
	mode, err := ValidarEnum("mode", req.Mode, padrao,
		"replace_section", "replace_heading_and_section", "replace_block")
	if err != nil {
		return PatchNoteResult{}, err
	}
```

E troque o `default:` do `switch mode` (`:366-371`) por:

```go
	default:
		// Inalcancavel: ValidarEnum ja recusou tudo fora dos tres. Fica como
		// guarda contra um case novo que entre na lista e nao no switch.
		return PatchNoteResult{}, Errorf(CodeInternal, "mode %q passou por ValidarEnum sem case", mode)
```

Se `err` já estiver declarado no escopo (é provável — `canonical, err :=` acima), use `mode, err = ValidarEnum(...)` com `var mode string` antes.

- [ ] **Step 4: Rodar os dois testes de mode (`TestPatchModeInvalidoNaoEErroInterno` e o novo); passam. Prova de mutação: tire `"replace_section"` da lista de `ValidarEnum`, rode, o novo teste nomeia; restaure.**

- [ ] **Step 5: Schema e TOOLS.md**

`tools_write.go:35`: `jsonschema:"replace_section (padrao), replace_heading_and_section ou replace_block (padrao quando block_id vem)"`.
`TOOLS.md:403`: substitua `"default": "replace_section"` por `"default": "replace_section (replace_block quando block_id vem)"` e, nas Notas de `:411`, acrescente: "Qualquer outro valor é `INVALID_ARGUMENT` com a lista dos três."

Run: `pwsh -File scripts/check_tool_params.ps1` (é uma das etapas do gate; roda sozinha para iterar).

- [ ] **Step 6: Gate e commit**

```bash
git add internal/service/write.go internal/service/limites_enums_test.go internal/mcpsrv/tools_write.go docs/TOOLS.md
git commit -m "fix(service): note_patch mode goes through ValidarEnum and the error lists the modes that exist"
```

#### Verificações
Além dos passos:
1. A lista de modos aparece UMA vez em `write.go` (a chamada a `ValidarEnum`); o `default:` do `switch` vira guarda inalcançável com comentário dizendo isso.
2. O schema em `internal/mcpsrv/tools_write.go` e a tabela em `docs/TOOLS.md` listam os mesmos três modos, na mesma ordem. `pwsh -File scripts/check_tool_params.ps1` verde.
3. A mensagem de erro para `mode: "xyz"` cita os três modos — cole-a no relatório.

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
  -Anchor '"replace_section", "replace_heading_and_section", "replace_block")' `
  -Replacement '"replace_section", "replace_heading_and_section", "replace_block", "xyz")' `
  -Test TestPatchNoteModeInvalidoListaOsModosQueExistem -Package ./internal/service/
```

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

