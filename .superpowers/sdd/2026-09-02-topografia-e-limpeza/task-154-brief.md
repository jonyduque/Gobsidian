### Task 154: Apagar `doctor.Status.Marker` (1.9)

`Marker()` (`doctor/doctor.go:25-35`) não tem chamador de produção — o
marcador impresso vem de `internal/console`. Só `doctor_extra_test.go:20-42` o
usa, e testa uma função que nada usa.

**Files:**
- Modify: `internal/doctor/doctor.go:25-35` (apagar o método e o comentário)
- Modify: `internal/doctor/doctor_extra_test.go:20-42` (apagar `TestStatusMarkerDistinctPerStatus`)

- [ ] **Step 1: Confirmar que não há chamador**

Run: gopls references em `Marker` (ou `grep -rn "\.Marker()" --include=*.go .`)
Expected: só o teste.

- [ ] **Step 2: Apagar os dois. `go build ./... && go vet ./internal/doctor/...`. Gate. Commit.**

```bash
git add internal/doctor/doctor.go internal/doctor/doctor_extra_test.go
git commit -m "refactor(doctor): drop Status.Marker, the console package owns the markers"
```

#### Verificações
Além dos passos:
1. `gopls` references em `Marker` antes de apagar: só o teste. Cole a saída.
2. `internal/console` continua sendo quem imprime os marcadores — nenhum marcador ASCII foi reintroduzido no `doctor`.
3. `go vet ./internal/doctor/...` e `golangci-lint run ./internal/doctor/...` limpos (apagar código pode deixar import órfão).

#### Regras de execução
- Gate: `pwsh -File scripts/verify.ps1` verde antes do commit, com a contagem de etapas colada no relatório. `-SkipCross -SkipNet` só para iterar.
- Nunca `git checkout`, `git restore`, `git stash`, `git clean` nem `git reset`. Há trabalho não commitado no repositório (`test-vault/`, `.claude/skills/`, `Resume-Claude.ps1`): `git diff <caminho>` antes de `git add <caminho>`; nunca `git add -A`.
- Nunca `go mod tidy`.
- Nunca despache subagentes. Nunca mate processo por nome (`Stop-Process -Name`); só por PID que você lançou.
- Referências e renames por gopls (LSP), não por grep. `grep` só para confirmar tags e reflection.
- Se um teste falhar por motivo que este brief não explica, **pare e reporte `BLOCKED`**; não ajuste a expectativa para o código passar.
- Este commit é de UMA categoria (o prefixo do assunto diz qual). Se você se pegar corrigindo outra coisa no caminho, anote em "O que ficou de fora" e não corrija.

#### Comando de mutação
Esta tarefa não tem prova de mutação: apaga código sem chamador; a prova é o build verde sem ele.

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

