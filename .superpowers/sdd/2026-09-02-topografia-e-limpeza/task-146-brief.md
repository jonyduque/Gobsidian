### Task 146: Commit dos benchmarks de análise e publicação da baseline

**Files:**
- Commit (já existem, não versionados): `internal/service/bench_analise_test.go`, `internal/index/bench_analise_test.go`, `internal/search/bench_analise_test.go`, `internal/writer/bench_analise_test.go`, `internal/parser/bench_analise_test.go`
- Modify: `docs/ESTADO.md` (seção de medições — acrescentar a tabela de "Baseline medida" deste plano, sem a coluna "Task que compara")

**Interfaces:**
- Produces: os nomes de benchmark que TODAS as Tasks de comparação citam — `BenchmarkLinkGraphBothDepth2`, `BenchmarkTagListPlano`, `BenchmarkTagListHierarquico`, `BenchmarkNoteListPorTag`, `BenchmarkSaveIndexCacheReal[ComFsync]`, `BenchmarkTagsSemPrefixo`, `BenchmarkSaveInvertedCacheReal[ComFsync]`, `BenchmarkRewriteLinksMuitos`, `BenchmarkParseNotaLonga`, `BenchmarkDetectCandidatesNotaLonga`. Não renomeie nenhum: os binários `antes_*` já os têm com esse nome, e `benchstat` casa por nome.

- [ ] **Step 1: Confirmar que os cinco arquivos compilam e rodam uma iteração**

Run: `go test ./internal/service ./internal/index ./internal/search ./internal/writer ./internal/parser -run XXX_NENHUM -bench 'LinkGraphBoth|TagList|NoteListPorTag|SaveIndexCacheReal|TagsSemPrefixo|SaveInvertedCacheReal|RewriteLinksMuitos|NotaLonga' -benchtime 1x`
Expected: cinco `ok`, nenhum `FAIL`, nenhum `SKIP` (o de service exige `$env:TEMP\vault_5000` — se faltar, gere: `pwsh -File scripts/gen_vault.ps1 -Notes 5000 -Seed 42 -Out $env:TEMP\vault_5000`).

- [ ] **Step 2: Copiar a tabela "Baseline medida" para `docs/ESTADO.md`**

Cole a tabela deste plano numa subseção `### Baseline dos benchmarks de análise — 2026-09-02, HEAD 6c5d1f1`, mantendo a frase de que são 5 amostras e a referência aos binários em `%LOCALAPPDATA%\gobsidian-bench\2026-09-02\`. Sem a coluna "Task que compara".

- [ ] **Step 3: gofmt, vet, gate**

Run: `gofmt -l ./internal; pwsh -File scripts/verify.ps1`
Expected: `gofmt -l` vazio; verify verde.

- [ ] **Step 4: Commit**

```bash
git add internal/service/bench_analise_test.go internal/index/bench_analise_test.go internal/search/bench_analise_test.go internal/writer/bench_analise_test.go internal/parser/bench_analise_test.go docs/ESTADO.md
git commit -m "test(bench): add analysis benchmarks and publish the 2026-09-02 baseline"
```

#### Verificações
Além dos passos:
1. Os cinco `bench_analise_test.go` NÃO estão versionados hoje (`git status --porcelain internal/*/bench_analise_test.go` mostra `??`). Depois do commit, `git ls-files internal/*/bench_analise_test.go` lista os cinco.
2. A tabela colada em `docs/ESTADO.md` tem os MESMOS números da seção "Baseline medida" do plano (`docs/superpowers/plans/2026-09-02-topografia-e-limpeza.md`, seção `## Baseline medida`): confira três linhas ao acaso, e cole as três no relatório.
3. `go vet ./internal/...` limpo — os arquivos de bench entram no vet do gate pela primeira vez.
4. Nenhum benchmark foi renomeado (Step "Interfaces").

#### Regras de execução
- Gate: `pwsh -File scripts/verify.ps1` verde antes do commit, com a contagem de etapas colada no relatório. `-SkipCross -SkipNet` só para iterar.
- Nunca `git checkout`, `git restore`, `git stash`, `git clean` nem `git reset`. Há trabalho não commitado no repositório (`test-vault/`, `.claude/skills/`, `Resume-Claude.ps1`): `git diff <caminho>` antes de `git add <caminho>`; nunca `git add -A`.
- Nunca `go mod tidy`.
- Nunca despache subagentes. Nunca mate processo por nome (`Stop-Process -Name`); só por PID que você lançou.
- Referências e renames por gopls (LSP), não por grep. `grep` só para confirmar tags e reflection.
- Se um teste falhar por motivo que este brief não explica, **pare e reporte `BLOCKED`**; não ajuste a expectativa para o código passar.
- Este commit é de UMA categoria (o prefixo do assunto diz qual). Se você se pegar corrigindo outra coisa no caminho, anote em "O que ficou de fora" e não corrija.

#### Comando de mutação
Esta tarefa não tem prova de mutação: ela versiona benchmarks e documenta uma medição, não entrega regra nova.

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

## Fase 1 — defeitos (cada Task é UM `fix:`)

