### Task 158: Serviço sem índice devolve `VAULT_UNAVAILABLE`, não `fmt.Errorf` (1.13 + `outline`)

`graph.go:87,:303,:455,:556,:663` fazem `fmt.Errorf("index not available")` —
erro sem código, que `mcpsrv` traduz para `INTERNAL`. `outline.go:71` devolve
`CodeInternal` para nota ilegível quando `read.go:399` devolve
`CodeVaultUnavailable` para a mesma condição. Mesmo fato, dois códigos.

**Files:**
- Modify: `internal/service/graph.go` (cinco sítios), `internal/service/outline.go:71`
- Test: `internal/service/errors_test.go` (acrescentar)

- [ ] **Step 1: Teste**

```go
func TestServicoSemIndiceDevolveVaultUnavailable(t *testing.T) {
	root := t.TempDir()
	v, err := vault.New(root)
	if err != nil {
		t.Fatal(err)
	}
	svc := New(v, nil, nil, nil, Options{})
	ctx := context.Background()

	chamadas := map[string]func() error{
		"LinkGraph": func() error { _, e := svc.LinkGraph(ctx, GraphRequest{Path: "a.md"}); return e },
		"TagList":   func() error { _, e := svc.TagList(ctx, TagRequest{}); return e },
		"NoteList":  func() error { _, e := svc.NoteList(ctx, ListRequest{}); return e },
		"Metadata":  func() error { _, e := svc.NoteMetadata(ctx, MetadataRequest{Path: "a.md"}); return e },
		"Stats":     func() error { _, e := svc.VaultStats(ctx, StatsRequest{}); return e },
	}
	for nome, f := range chamadas {
		err := f()
		if err == nil {
			t.Errorf("%s sem indice devolveu nil", nome)
			continue
		}
		if got := CodeOf(err); got != CodeVaultUnavailable {
			t.Errorf("%s sem indice: codigo = %s, quer %s (%v)", nome, got, CodeVaultUnavailable, err)
		}
	}
}
```

Confira os nomes reais dos métodos e dos tipos de request em `graph.go` (`grep -n "^func (s \*Service)" internal/service/graph.go`) e ajuste.

- [ ] **Step 2: Rodar; cinco falhas com `codigo = INTERNAL`**

- [ ] **Step 3: Cinco sítios**: `Errorf(CodeVaultUnavailable, "indice indisponivel")`. Se `fmt` ficar sem uso em `graph.go`, tire o import. `outline.go:71`: `Errorf(CodeVaultUnavailable, "lendo nota %q: %v", req.Path, err)`.

- [ ] **Step 4: Rodar; passa. Gate. Commit.**

```bash
git add internal/service/graph.go internal/service/outline.go internal/service/errors_test.go
git commit -m "fix(service): a missing index is VAULT_UNAVAILABLE everywhere, and outline agrees with read"
```

#### Verificações
Além dos passos:
1. `grep -rn 'fmt.Errorf("index not available")' internal/service` vazio.
2. `grep -n "CodeInternal" internal/service/outline.go` não lista mais a linha do `ReadAll`.
3. `docs/TOOLS.md`: onde `VAULT_UNAVAILABLE` é descrito, a frase cobre "índice indisponível"; a tabela de erros de `note_outline` não promete `INTERNAL` para nota ilegível.

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
pwsh -File scripts/mutate.ps1 -Path internal/service/graph.go `
  -Anchor 'return nil, Errorf(CodeVaultUnavailable, "indice indisponivel")' `
  -Replacement 'return nil, fmt.Errorf("indice indisponivel")' `
  -Test TestServicoSemIndiceDevolveVaultUnavailable -Package ./internal/service/
```
(Com cinco sítios idênticos a âncora é ambígua — `EXIT=2`. Nesse caso mute UM sítio à mão, cole o diff, rode, cole a falha nomeando o método, restaure.)

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

## Fase 2 — Testes: infraestrutura comum, testes que não podem falhar, confundidores, tempo e contrapesos (Tasks 159–165)

A Fase 2 não muda produto. Ela existe porque a Fase 3 (código morto, simplificações,
otimizações) e a Fase 4 (movimentos entre pacotes) só são seguras sobre uma rede de
testes que **falha quando deve**. Cada tarefa desta fase termina com uma prova de que
o teste novo (ou consertado) nomeia a falha quando a regra é apagada.

Modelo por tarefa: 159 Opus (Windows, handles, ciclos de import); 160 Haiku
(apagar e substituir com texto completo); 161 Opus (julgamento sobre 13 sítios);
162 Sonnet; 163 Sonnet; 164 Opus (sincronização de watcher sob `-race`);
165 Opus (kernel lock, half-close do ponte).

