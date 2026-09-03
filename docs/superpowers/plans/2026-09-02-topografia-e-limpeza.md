# Topografia e limpeza — plano de implementação

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fechar os treze defeitos, os testes que não podem falhar e o código
morto encontrados na análise de 2026-09-02; depois mover as partes que estão no
pacote errado (`WriteAtomic` → `vault`, boot → `internal/boot`), apagar a
interface `service.Index`, e aplicar as duas mudanças de contrato decididas
(`hits` sai; `tags` vira uma conta hierárquica nos três consumidores) — sem
perder um nanossegundo medido.

**Architecture:** Cada Task é um commit de UMA categoria — comportamental
(fecha defeito, muda contrato) ou estrutural (move, apaga, simplifica) — nunca
as duas. A ordem é: defeitos → testes → código morto → simplificações →
topografia → contrato. Move cross-package usa forwarder temporário
(`writer.WriteAtomic` chama `vault.WriteAtomic` por um PR) para que nenhum
commit precise tocar todos os call sites. Toda simplificação que toca caminho
quente tem `benchstat` antes/depois contra os binários da Task 146.

**Tech Stack:** Go 1.26.5, gopls (Rename/References), benchstat,
`scripts/verify.ps1`, `scripts/test_orphans.ps1`, `scripts/mutate.ps1`.

**Spec:** a análise consolidada e as decisões do dono estão na conversa de
2026-09-02 e resumidas em "Decisões do dono" abaixo; a normativa continua sendo
`docs/PRD.md`, `docs/ARCHITECTURE.md`, `docs/TOOLS.md`. Quando este plano e
`TOOLS.md` divergirem depois de uma Task de contrato, é `TOOLS.md` que foi
atualizado no mesmo commit — releia-o.

**Numeração:** as Tasks continuam a numeração do ledger único
(`.superpowers/sdd/2026-07-25-gobsidian-v01/progress.md`), que parou em 145 (as Tasks 140–145 de 2026-09-01 não têm plano próprio).
Este plano vai de **Task 146 a Task 181**. O diretório
`.superpowers/sdd/2026-09-02-topografia-e-limpeza/` guarda só o `progress.md`
ponteiro (copie o de `2026-08-16-revisao-fixes/`), briefs e `review-*.diff`.

## Decisões do dono (2026-09-02)

| # | Decisão | Onde no plano |
|---|---|---|
| (a) | `hits` sai de `vault_search`; fica `results` | Task 179 |
| (b) | filtro `tags` hierárquico + strip `#` + caixa + NFC nos três (`note_list`, `vault_search`, `tag_list`), uma conta só | Task 180 |
| (c1) | `WriteAtomic` + `SweepStaleTempFiles` + prefixo temp vão para `internal/vault`; caches usam; nenhuma aresta nova | Tasks 171–172 |
| (c2) | `internal/bincodec` adiado; 1.12 fecha com alias agora; teste white-box do codec do `index` é o pré-requisito e entra por último | Tasks 157, 181 |
| (d) | apagar `service.Index`; `*index.Index` direto | Task 173 |
| (e) | `internal/boot` em dois PRs: montagem pura, depois ciclo de vida | Tasks 174–177 |

**Sub-decisão pendente (B.1), a confirmar antes da Task 180:** com a chave de
tag dobrada, `tag_list` passa a devolver a forma dobrada: `#Ação` sai como
`ação` (NFC, minúscula, sem `#`; acento fica — a chave NÃO remove acento, só
normaliza a forma). A alternativa — guardar "primeira grafia vista" — é
dependente da ordem do worker pool no `Build`, logo não determinística entre
dois boots do mesmo cofre. O plano assume a forma dobrada.

## Global Constraints

Valem para TODAS as Tasks. Copiadas de `CLAUDE.md`; a redação de lá vence.

- `pwsh -File scripts/verify.ps1` verde antes de **cada** commit. `-SkipCross -SkipNet` só para iterar; o commit exige o gate completo.
- Nunca `git checkout`, `git restore`, `git stash`, `git clean`, `git reset`. Há trabalho não commitado no repositório o tempo todo (`test-vault/`, `.claude/skills/`, `Resume-Claude.ps1`). Commite por caminho explícito (`git add <arquivo>`), nunca `git add -A`.
- Nunca `go mod tidy`. Piso `go 1.25.0`; SDK MCP fixado (PRD D6).
- stdout pertence ao JSON-RPC; log via `log/slog` em stderr. `doctor`, `version`, `search`, `index`, `inspect` imprimem em stdout de propósito, com comentário dizendo isso.
- Nenhum pacote sob `internal/` ou `cmd/` importa `net/*`, exceto `net.Dial`/`net.Listen` com a rede literal `"unix"` em `internal/ipc`.
- Nenhum tipo do SDK MCP fora de `internal/mcpsrv`.
- Aresta nova no grafo de imports precisa de justificativa escrita no commit; folha não ganha import. Grafo de produção em `CLAUDE.md`; conferir com `go list -f '{{.Imports}}' ./internal/<pkg>` **antes e depois** de cada Task estrutural.
- Uma conta por regra: toda chave derivada passa por uma função só.
- Código de plataforma atrás de build tag em arquivo separado; nunca `if runtime.GOOS ==` em lógica compartilhada.
- Console em ASCII puro: `[OK]`, `[*]`, `[!]`, `[i]`, `[...]`.
- Sem `helpers.go`, `utils.go`, `common.go`. (`internal/vaulttest` é pacote nomeado pela função, permitido.)
- Commits em Conventional Commits, em inglês. Uma categoria por commit: `fix:`/`feat:` (comportamental) ou `refactor:`/`test:`/`docs:` (estrutural), nunca os dois no mesmo commit.
- Não escreva número que não mediu; escreva **"não medido"**. Prova de mutação é saída colada, no passado.
- Um teste que não pode falhar é pior que teste ausente: antes de dizer que testou, apague a regra, rode, confirme que um teste nomeia a falha, restaure — e cole a saída no relatório.
- Escopo não encolhe em silêncio: `BLOCKED` com motivo.
- Registre no ledger antes de dizer que acabou. O relatório é o entregável.
- Medição de desempenho segue `docs/papeis/desempenho.md`: dois binários compilados ANTES de medir, execuções intercaladas, ≥7 repetições, `benchstat`, cofre `$env:TEMP\vault_5000` gerado por `scripts/gen_vault.ps1 -Notes 5000 -Seed 42` (o gerador atual, commit 3363445 — um cofre gerado antes de 2026-09-01 faz `BenchmarkSearchDoisTermos` e `FraseExata` casarem zero e FALHAREM; apague e regenere). Snippet cache desligado. Sem `-race` em bench.
- "Use servidores LSP quando estiver programando" (instrução global do dono): referências e renames por gopls, não por grep.

## Baseline medida — 2026-09-02, HEAD 6c5d1f1

Máquina: i7-10750H, windows/amd64, Go 1.26.5. Binários e saídas brutas em
`%LOCALAPPDATA%\gobsidian-bench\2026-09-02\` (`antes_<pkg>.test.exe`,
`antes_<pkg>_run<N>.txt`). São os binários "antes" de TODAS as comparações
deste plano — não os recompile; o `benchstat` compara contra eles.

`go test -run '^$' -bench . -benchmem -count=5` por pacote (**5 amostras** —
abaixo das 7 que `desempenho.md` exige, então esta tabela é REFERÊNCIA, não
veredito; toda comparação deste plano roda o binário `antes_*` de novo,
intercalado com o `depois_*`, com `-count=7` ou mais, e é o `benchstat` dessa
rodada que decide). Mediana das 5, via `benchstat antes_all.txt`:

| Benchmark (pacote) | sec/op | B/op | allocs/op | Task que compara |
|---|---|---|---|---|
| `LinkGraphBothDepth2` (service) | 15,21 µs | 2,06 KiB | 41 | 169 |
| `TagListPlano` (service) | 31,78 µs | 13,4 KiB | 9 | 169, 180 |
| `TagListHierarquico` (service) | 6,163 ms | 1,03 MiB | 10 880 | 169, 180 |
| `NoteListPorTag` (service) | 403,8 µs | 41,5 KiB | 210 | 180 |
| `SearchLimit200Cache` (service) | 14,58 ms | 2,38 MiB | 10 170 | 169, 173, 179 |
| `SearchTermoAmploCache` (service) | 7,324 ms | 1,94 MiB | 5 594 | 173, 179 |
| `SearchFiltroFrontmatter` (service) | 23,20 ms | 4,31 MiB | 30 120 | 180 |
| `SearchLimit200CacheTrechoRepetido` (service) | 7,270 ms | 2,11 MiB | 7 654 | 179 |
| `IndexBuild` (service) | 352,4 ms | 95,6 MiB | 918 400 | 170, 173 |
| `InvertedLoad` (service) | 17,50 ms | 3,51 MiB | 46 380 | 172 |
| `SearchTermoAmplo` / `DoisTermos` / `FraseExata` / `Limit200` (service) | 8,705 / 3,658 / 23,27 / 15,41 ms | — | — | 173 |
| `SaveIndexCacheReal` / `ComFsync` (index) | 21,76 / 41,87 ms | 1,17 MiB | 6 088 | 172 |
| `TagsSemPrefixo` (index) | 20,02 µs | 7,35 KiB | 8 | 180 |
| `ListPorTag` (index) | 856,1 µs | 119 KiB | 12 | 180 |
| `BuildComHub` (index) | 115,4 ms | 12,95 MiB | 128 400 | 170 |
| `TotalSizeRepetido` (index) | 5,610 µs | 0 | 0 | 169 |
| `SaveInvertedCacheReal` / `ComFsync` (search) | 227,3 / 259,1 ms | 24,95 MiB | 255 000 | 172 |
| `EscreveCache` (search) | 24,56 ms | 1,04 MiB | 822 | 172 |
| `InvertedUpdateLote` (search) | 6,549 s | 105 MiB | 565 200 | 170 |
| `RewriteLinksMuitos` (writer) | 1,496 ms | 3,98 MiB | 795 | 170 |
| `ParseNotaLonga` (parser) | 2,611 ms | 1,19 MiB | 9 247 | 170 |
| `DetectCandidatesNotaLonga` (parser) | 309,9 µs | 127 KiB | 1 301 | 170 |

Saída completa do `benchstat`: `%LOCALAPPDATA%\gobsidian-bench\2026-09-02\antes_all.txt`
(concatenação dos cinco `antes_<pkg>_run*.txt` válidos; `antes_service_run1.txt`
NÃO entra — rodou contra o cofre velho e tem dois FAIL).

Fatos medidos sem benchmark:

- `vault_5000`: 112 tags distintas, 0 hierárquicas. Cofres reais do dono: **não medido** neste plano.
- Cobertura por função (`go test -coverprofile`, 2026-09-02): `construirServico` 6,5 %, `carregarIndiceDoCache` 0 %, `prepararIndiceDeBusca` 0 %, `runServe` 0 %, `serveEmProcesso` 20,6 %, `buildInvertedIndex` 60,7 %; `WriteAtomic` 71,1 %, `SweepStaleTempFiles` 62,5 %, `CleanStaleTempFiles` 0 %; `SaveIndexCache` 64,5 %, `SaveInvertedCache` 56,0 %; `Tags` 91,7 %, `coletarLocked` 87,6 %, `tagListHierarchical` 94,1 %, `LinkGraph` 78,1 %; `Search` 96 %; `mcpsrv.Server.Serve` 0 %, `Close` 0 %; `doctor.checkDaemonLog` 44,4 %, `checkLocksDeDaemon` 58,6 %.
- Tempo de suíte (`go test ./... -count=1 -cover`): service 51,7 s, writer 32,5 s, search 31,1 s, watcher 26,2 s, index 22,3 s, vault 22,0 s, doctor 19,4 s, mcpsrv 17,0 s.
- Raio de explosão de `service.Index` (gopls references): `Get` 13, `ResolvePath` 8, `Backlinks` 5, `List` 4, `NotePaths` 2, `TotalSize`/`Tags`/`NoteCount`/`Generation`/`AssetCount`/`AliasCollisions` 1 cada, `Paths` **0**.
- Raio de `writer.WriteAtomic`: 6 sítios em `internal/service/write.go` (`:149,:248,:378,:619,:751,:832`) + 3 arquivos de teste; `SweepStaleTempFiles`: 1 sítio (`cmd/gobsidian/servico.go:78`).
- Prefixo `.gobsidian-tmp-` em 4 literais: `writer/atomic.go:14` (constante), `vault/walk.go:74`, `search/persist.go:87`, `index/persist.go:124`.
- `hits` × `results`: `search --json --limit 200 --vault vault_5000 "execucao"` (binário de 6c5d1f1): `hits` e `results` são 200 itens e **byte a byte iguais** (`hits == results` → `True` em Python). JSON compacto: 195 481 bytes com `hits`, 97 787 sem — **50,0 % do payload é a cópia**. Arquivo indentado: 216 304 bytes.
- CLI a frio × `serve` com cache: cofre `vault_5000`, mesmo binário. `search` a frio (Build + `inv.Update` serial por nota): **10 520 / 10 819 / 11 046 ms** de parede em 3 execuções. `inspect` a frio (só Build): **770 / 767 / 739 ms**. `serve` em processo (`GOBSIDIAN_NO_DAEMON=1 --eager-search`) com cache quente, 5 execuções: `index_ms` **101–123**, índice de busca `duracao_ms` **13–20**, parede boot→saída **475–528 ms**. Ou seja: a CLI de busca paga ~10 s que o `serve` não paga; adotar o cache na CLI (Task 175) tem teto de ganho medido, não estimado.

---

## Fase 0 — a régua

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

### Task 147: Falha de `ReadDir` na raiz varrida é falha da raiz (1.1)

`filepath.WalkDir(dir, fn)` chama `fn` para a raiz DUAS vezes quando `Lstat`
passa mas `ReadDir` falha: a primeira com `d != nil, err == nil`, a segunda com
`d != nil, caminho == dir, err != nil`. Os dois callbacks que tratam "falha da
raiz" só reconhecem `d == nil` (`Lstat` falhou), então uma raiz que EXISTE mas
não pode ser LIDA vira "entrada ilegível", é engolida, e a varredura devolve
sucesso com zero entradas — no `vault.Walk` isso é "cofre vazio"; no
`watcher.varreDiretorioNovo` é "diretório novo sem nada dentro".

Provado nesta máquina em 2026-09-02 com um handle exclusivo
(`CreateFile(..., dwShareMode=0, FILE_FLAG_BACKUP_SEMANTICS)`) sobre o
diretório: `os.Lstat(dir)` → `nil`; `os.ReadDir(dir)` → `The process cannot
access the file because it is being used by another process`. É exatamente o
cenário do antivírus segurando a pasta recém-movida.

**Files:**
- Modify: `internal/vault/walk.go:123-146` (callback de `Walk`)
- Modify: `internal/watcher/watcher.go:228-252` (callback de `varreDiretorioNovo`)
- Test: `internal/vault/walk_raiz_test.go` (novo, puro)
- Test: `internal/vault/walk_raiz_windows_test.go` (novo, `//go:build windows`)
- Test: `internal/watcher/varredura_raiz_test.go` (novo, `//go:build !windows`)

**Interfaces:**
- Produces: `func FalhaNaRaiz(raiz, caminho string, d fs.DirEntry) bool` em `internal/vault/walk.go`, exportada — a ÚNICA conta de "este erro do WalkDir é da raiz". `watcher` já importa `vault`; nenhuma aresta nova.

- [ ] **Step 1: Teste puro da conta, em `internal/vault/walk_raiz_test.go`**

```go
package vault

import (
	"io/fs"
	"testing"
)

// entradaFalsa satisfaz fs.DirEntry sem tocar o disco. So Name e usado
// pela conta; o resto existe para compilar.
type entradaFalsa struct{ nome string }

func (e entradaFalsa) Name() string               { return e.nome }
func (e entradaFalsa) IsDir() bool                { return true }
func (e entradaFalsa) Type() fs.FileMode          { return fs.ModeDir }
func (e entradaFalsa) Info() (fs.FileInfo, error) { return nil, fs.ErrNotExist }

func TestFalhaNaRaizReconheceAsDuasFormasDoWalkDir(t *testing.T) {
	const raiz = `C:\cofre`
	casos := []struct {
		nome    string
		caminho string
		d       fs.DirEntry
		quer    bool
	}{
		{"Lstat da raiz falhou: d == nil", raiz, nil, true},
		{"ReadDir da raiz falhou: d != nil, caminho == raiz", raiz, entradaFalsa{"cofre"}, true},
		{"entrada comum ilegivel", `C:\cofre\sub`, entradaFalsa{"sub"}, false},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			if got := FalhaNaRaiz(raiz, c.caminho, c.d); got != c.quer {
				t.Fatalf("FalhaNaRaiz(%q, %q, %v) = %v, quer %v", raiz, c.caminho, c.d, got, c.quer)
			}
		})
	}
}
```

- [ ] **Step 2: Rodar; deve falhar por símbolo ausente**

Run: `go test ./internal/vault -run TestFalhaNaRaiz`
Expected: `undefined: FalhaNaRaiz`.

- [ ] **Step 3: Escrever a conta e usá-la em `Walk`**

Em `internal/vault/walk.go`, antes de `Walk`:

```go
// FalhaNaRaiz diz se um erro entregue pelo callback de filepath.WalkDir e da
// PROPRIA raiz varrida, e nao de uma entrada dentro dela.
//
// WalkDir tem duas formas de falhar na raiz, e so uma delas vem com d == nil:
//
//   - Lstat(raiz) falhou: um unico callback, d == nil.
//   - Lstat passou e ReadDir(raiz) falhou: DOIS callbacks — o primeiro normal,
//     o segundo com d != nil, caminho == raiz e o erro do ReadDir.
//
// Tratar so d == nil deixa a segunda forma passar como "entrada ilegivel":
// engolida, logada, e a varredura devolve sucesso com zero entradas. Um
// diretorio que existe mas nao pode ser lido (antivirus segurando a pasta,
// no Windows) virava cofre vazio. E a unica conta dessa distincao — Walk e
// watcher.varreDiretorioNovo usam a mesma.
func FalhaNaRaiz(raiz, caminho string, d fs.DirEntry) bool {
	return d == nil || caminho == raiz
}
```

E no callback de `Walk`, troque `if d == nil {` por `if FalhaNaRaiz(v.walkRoot, abs, d) {` — o comentário acima do `if` ganha a frase "Ver FalhaNaRaiz para a segunda forma, que ReadDir produz." e perde a afirmação de que `d == nil` é a única.

- [ ] **Step 4: Usar a conta em `varreDiretorioNovo`**

Em `internal/watcher/watcher.go:245`, troque `if d == nil {` por `if vault.FalhaNaRaiz(dir, caminho, d) {`. O comentário de :230-244 fica; acrescente uma linha: "A segunda forma — ReadDir da raiz falhou, d != nil — e a que vault.FalhaNaRaiz cobre."

- [ ] **Step 5: Teste no watcher, `internal/watcher/varredura_raiz_test.go`**

O que se prova aqui é que o watcher USA a conta. A versão portátil tira a permissão de leitura por `chmod`, que no Windows não faz nada — por isso o build tag:

```go
//go:build !windows

package watcher

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestVarreDiretorioNovoNaoEngoleRaizIlegivel(t *testing.T) {
	w, cancel, root, _ := setupTestWatcher(t)
	defer cancel()
	dir := filepath.Join(root, "chegou")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a.md"), []byte("# a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
	if _, err := os.ReadDir(dir); err == nil {
		t.Skip("ReadDir nao falhou com 0o000 (root?); o cenario nao se reproduz aqui")
	}
	err := w.varreDiretorioNovo(context.Background(), dir)
	if err == nil {
		t.Fatal("varreDiretorioNovo devolveu nil para uma raiz que ReadDir nao consegue ler: a varredura reportou sucesso com zero entradas")
	}
}
```

`setupTestWatcher` é o construtor de `internal/watcher/counters_test.go:68` — devolve `(*Watcher, context.CancelFunc, string, *index.Index)`, com o watcher já rodando sobre um `t.TempDir()`. O evento `Create` do `MkdirAll` vai disparar `varreDiretorioNovo` pelo `Run` também; não importa — o teste chama a função diretamente e julga só o retorno dela.

- [ ] **Step 6: Teste Windows em `internal/vault/walk_raiz_windows_test.go`**

```go
//go:build windows

package vault

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/windows"
)

// travarDiretorioExclusivo abre dir com dwShareMode = 0. Enquanto o handle
// vive, ReadDir(dir) falha com ERROR_SHARING_VIOLATION e Lstat(dir) passa —
// a segunda forma de falha da raiz que FalhaNaRaiz existe para reconhecer.
// Provado nesta maquina em 2026-09-02; o teste confere de novo e pula, com o
// motivo, se o SO desta vez deixar ler.
func travarDiretorioExclusivo(t *testing.T, dir string) {
	t.Helper()
	p, err := windows.UTF16PtrFromString(dir)
	if err != nil {
		t.Fatal(err)
	}
	h, err := windows.CreateFile(p, windows.GENERIC_READ|windows.GENERIC_WRITE, 0, nil,
		windows.OPEN_EXISTING, windows.FILE_FLAG_BACKUP_SEMANTICS, 0)
	if err != nil {
		t.Fatalf("CreateFile exclusivo em %q: %v", dir, err)
	}
	t.Cleanup(func() { _ = windows.CloseHandle(h) })
	if _, err := os.ReadDir(dir); err == nil {
		t.Skip("handle exclusivo NAO impediu ReadDir nesta maquina; o cenario nao se reproduz")
	}
}

func TestWalkNaoEngoleRaizQueExisteMasNaoLe(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.md"), []byte("# a"), 0o644); err != nil {
		t.Fatal(err)
	}
	v, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	travarDiretorioExclusivo(t, root)

	var vistos int
	err = v.Walk(context.Background(), func(Entry) error { vistos++; return nil })
	if err == nil {
		t.Fatalf("Walk devolveu nil com %d entradas para uma raiz que ReadDir nao le: cofre inacessivel virou cofre vazio", vistos)
	}
}
```

`golang.org/x/sys` já está no `go.mod` (`v0.47.0`, indireto); se `go build` reclamar de `go.sum`, `go get golang.org/x/sys/windows@v0.47.0` — caminho do pacote, nunca `go mod tidy`.

- [ ] **Step 7: Rodar os três; ver o do Windows FALHAR antes do fix**

Run: `go test ./internal/vault ./internal/watcher -run 'FalhaNaRaiz|RaizQueExiste|RaizIlegivel' -v`
Expected ANTES do Step 3/4: `TestWalkNaoEngoleRaizQueExisteMasNaoLe` FAIL com "Walk devolveu nil com 0 entradas". Se der SKIP, registre o motivo e a Task fica `BLOCKED`: o cenário não se reproduz e não há prova. DEPOIS: PASS. O teste `!windows` não roda nesta máquina; o gate só o compila (`go vet` com `GOOS=linux`) — diga isso no relatório em vez de "testes passam".

- [ ] **Step 8: Prova de mutação**

Troque `return d == nil || caminho == raiz` por `return d == nil`, rode o Step 7, cole a saída (`TestFalhaNaRaiz.../ReadDir_da_raiz_falhou` e o Windows devem falhar), restaure.

- [ ] **Step 9: Gate e commit**

```bash
pwsh -File scripts/verify.ps1
git add internal/vault/walk.go internal/vault/walk_raiz_test.go internal/vault/walk_raiz_windows_test.go internal/watcher/watcher.go internal/watcher/varredura_raiz_test.go
git commit -m "fix(vault): a ReadDir failure on the scanned root is a root failure, not an unreadable entry"
```

(Acrescente `go.sum` ao `git add` só se ele mudou.)

#### Verificações
Além dos passos:
1. `walk.go` e `watcher.go` chamam a MESMA função (`vault.FalhaNaRaiz`); `grep -rn "d == nil" internal/vault internal/watcher` não devolve mais nenhuma decisão de raiz feita à mão.
2. `internal/watcher/watcher.go:63` (varredura inicial, `path == root`) NÃO foi tocado nesta Task — ele já estava certo; se você o migrou para `FalhaNaRaiz`, diga no relatório, e é aceitável desde que o comportamento seja idêntico.
3. O teste Windows (`walk_raiz_windows_test.go`) reprova ANTES do fix com a mensagem que nomeia a raiz — cole a saída do RED.
4. `go list -f '{{.Imports}}' ./internal/watcher` continua sem aresta nova (`vault` já era importado).

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
pwsh -File scripts/mutate.ps1 -Path internal/vault/walk.go `
  -Anchor 'return d == nil || caminho == raiz' `
  -Replacement 'return d == nil' `
  -Test TestFalhaNaRaiz -Package ./internal/vault/
```
E a segunda, no consumidor:
```bash
pwsh -File scripts/mutate.ps1 -Path internal/watcher/watcher.go `
  -Anchor 'vault.FalhaNaRaiz(dir, caminho, d)' `
  -Replacement 'd == nil' `
  -Test TestVarreDiretorioNovoNaoEngoleRaizIlegivel -Package ./internal/watcher/
```
(Este segundo só roda fora de Windows — `//go:build !windows`. Em Windows, faça a mutação à mão e rode `go test ./internal/vault -run TestWalkNaoEngoleRaizQueExisteMasNaoLe`, colando o diff e a saída.)

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

### Task 151: Lixeira move o arquivo em vez de copiar; a cópia de fallback recusa placeholder de nuvem (1.3)

`DeleteNote` com `to_trash` faz `ReadFile` + `WriteAtomic` + `Remove`
(`write.go:748-770`) — três operações e um arquivo inteiro na memória para o
que `os.Rename` faz numa. `moverCorpo` (`:805-841`) já é a conta de "mover
uma nota conferindo todos os erros"; a lixeira não a usa. E `moverCorpo`, no
fallback de cópia, faz `os.ReadFile` sem consultar `CloudOnly` — abrir um
placeholder do OneDrive dispara download síncrono, que é a regra "arquivo
somente-nuvem nunca é aberto" do `CLAUDE.md`, e "quem roda antes do guarda
precisa do mesmo guarda".

`PathLocker` NÃO é reentrante (`writer/lock.go:37-49`: `entry.mu.Lock()` de
novo na mesma goroutine trava para sempre). `DeleteNote` trava `canonical` em
`:713` e `moverCorpo` trava `de` e `para`; portanto a lixeira precisa chamar
`moverCorpo` SEM a trava de `DeleteNote` — a trava de `moverCorpo` cobre os
dois caminhos.

**Files:**
- Modify: `internal/service/write.go:700-777` (`DeleteNote`, ramo `ToTrash`) e `:826-841` (`moverCorpo`, fallback)
- Test: `internal/service/delete_test.go` (acrescentar teste portátil)
- Test: `internal/service/erro_engolido_windows_test.go` (acrescentar teste Windows)

- [ ] **Step 1: Teste portátil — a lixeira é um rename, não uma cópia**

Como saber que foi rename e não cópia+remove sem olhar a implementação? Um
arquivo grande em cópia custa tempo e memória, mas isso não é asserção. O que é
observável: rename preserva o mtime do arquivo; `WriteAtomic` (escreve, `sync`,
`rename` do temporário) produz um arquivo com mtime NOVO. Fixe um mtime antigo,
mova para a lixeira, confira que o mtime sobreviveu.

Em `internal/service/delete_test.go`:

```go
func TestDeleteNoteToTrashMoveSemCopiar(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "a.md", "# A\n\ncorpo\n")
	antigo := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)
	if err := os.Chtimes(filepath.Join(root, "a.md"), antigo, antigo); err != nil {
		t.Fatal(err)
	}
	svc := newTestService(t, root)

	res, err := svc.DeleteNote(context.Background(), DeleteNoteRequest{Path: "a.md", ToTrash: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.TrashPath == "" {
		t.Fatalf("sem TrashPath: %+v", res)
	}
	fi, err := os.Stat(filepath.Join(root, filepath.FromSlash(res.TrashPath)))
	if err != nil {
		t.Fatalf("a nota nao esta na lixeira: %v", err)
	}
	if !fi.ModTime().Equal(antigo) {
		t.Fatalf("mtime da copia na lixeira = %v, quer %v: a lixeira COPIOU em vez de mover", fi.ModTime(), antigo)
	}
	if _, err := os.Stat(filepath.Join(root, "a.md")); !os.IsNotExist(err) {
		t.Fatalf("a origem ainda existe (err=%v)", err)
	}
}
```

Acrescente `"os"`, `"path/filepath"`, `"time"` aos imports se faltarem.

- [ ] **Step 2: Rodar; DEVE falhar com "a lixeira COPIOU em vez de mover". Se passar sem o fix, a prova está errada (o `WriteAtomic` preservou mtime?) — pare, verifique com `os.Stat` antes/depois num teste descartável, e troque a asserção por outra que distinga rename de cópia (ex.: `os.SameFile` num handle aberto antes do move, no Windows). Não avance com um teste que já passa.**

- [ ] **Step 3: Teste Windows — a lixeira não abre placeholder**

Em `internal/service/erro_engolido_windows_test.go` (já é `//go:build windows`, pacote `service_test`, e tem `travaExclusiva`):

```go
// TestDeleteNoteToTrashNaoBaixaPlaceholder: com o rename recusado, o fallback
// de copia de moverCorpo faria os.ReadFile num placeholder de nuvem — o
// download sincrono que a regra "somente-nuvem nunca e aberto" proibe.
func TestDeleteNoteToTrashNaoBaixaPlaceholder(t *testing.T) {
	root := t.TempDir()
	caminho := filepath.Join(root, "nuvem.md")
	if err := os.WriteFile(caminho, []byte("# Nuvem\n\ncorpo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err := windows.UTF16PtrFromString(vault.LongPath(caminho))
	if err != nil {
		t.Fatal(err)
	}
	if err := windows.SetFileAttributes(p, windows.FILE_ATTRIBUTE_OFFLINE); err != nil {
		t.Skipf("nao foi possivel marcar FILE_ATTRIBUTE_OFFLINE: %v", err)
	}
	t.Cleanup(func() { _ = windows.SetFileAttributes(p, windows.FILE_ATTRIBUTE_NORMAL) })

	v, err := vault.New(root)
	if err != nil {
		t.Fatal(err)
	}
	idx := index.New()
	if err := idx.Build(context.Background(), v); err != nil {
		t.Fatal(err)
	}
	if n, ok := idx.Get("nuvem.md"); !ok || !n.CloudOnly {
		t.Fatal("a nota nao ficou CloudOnly; o atributo nao pegou")
	}
	// Handle exclusivo: os.Rename recusa, e o fallback de copia e forcado.
	travaExclusiva(t, caminho)
	svc := service.New(v, idx, nil, nil, service.Options{})

	_, err = svc.DeleteNote(context.Background(), service.DeleteNoteRequest{Path: "nuvem.md", ToTrash: true})
	if err == nil {
		t.Fatal("DeleteNote to_trash devolveu sucesso sobre um placeholder com rename recusado: o fallback leu o arquivo")
	}
	if got := service.CodeOf(err); got != service.CodeCloudOnlyFile {
		t.Fatalf("codigo = %s, quer %s: %v", got, service.CodeCloudOnlyFile, err)
	}
}
```

Acrescente `"github.com/jonyd/gobsidian/internal/index"` e `".../internal/vault"` aos imports do arquivo.

- [ ] **Step 4: Rodar; falha (código diferente de CLOUD_ONLY_FILE, ou sucesso)**

- [ ] **Step 5: Reescrever o ramo `ToTrash` de `DeleteNote`**

Substitua de `unlock := s.locker.Lock(canonical)` (`:713`) até o fim do ramo `if req.ToTrash { ... }` (`:777`) por:

```go
	absPath := s.vault.Abs(canonical)
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return DeleteNoteResult{}, Errorf(CodeNoteNotFound, "nota %q nao encontrada no disco", req.Path)
	}

	if req.ToTrash {
		// Sem s.locker.Lock aqui: PathLocker nao e reentrante, e moverCorpo
		// trava origem E destino em ordem global. A lixeira e um move como
		// qualquer outro — ate 2026-09-02 era ReadFile + WriteAtomic +
		// Remove, tres operacoes e o arquivo inteiro em memoria para o que
		// os.Rename faz numa, e sem o guarda de placeholder que moverCorpo
		// ganhou junto com esta mudanca.
		trashRel, absTrash, err := s.destinoNaLixeira(canonical)
		if err != nil {
			return DeleteNoteResult{}, err
		}
		if err := os.MkdirAll(filepath.Dir(absTrash), 0755); err != nil {
			return DeleteNoteResult{}, Errorf(CodeInternal, "criando diretorio lixeira: %v", err)
		}
		if err := s.moverCorpo(ctx, canonical, vault.CanonicalPath(trashRel), absTrash); err != nil {
			return DeleteNoteResult{}, err
		}
		return DeleteNoteResult{
			Path:          string(canonical),
			Deleted:       true,
			MovedToTrash:  true,
			TrashPath:     trashRel,
			BrokenLinks:   brokenLinks,
			BrokenAnchors: brokenAnchors,
		}, nil
	}

	unlock := s.locker.Lock(canonical)
	defer unlock()
```

(O `os.Remove` da exclusão definitiva continua logo abaixo, agora sob a trava.)
E extraia a resolução do destino — é o bloco de `:721-741`, sem mudança de lógica:

```go
// destinoNaLixeira resolve .trash/<nome>, com sufixo de timestamp se ja
// houver um arquivo com esse nome la.
func (s *Service) destinoNaLixeira(canonical vault.CanonicalPath) (trashRel, absTrash string, err error) {
	baseName := filepath.Base(string(canonical))
	trashRel = filepath.ToSlash(filepath.Join(".trash", baseName))
	absTrash, _, err = vault.Resolve(s.vault.Root(), trashRel)
	if err != nil {
		return "", "", mapVaultErr(err)
	}
	if _, err := os.Stat(absTrash); err == nil {
		ext := filepath.Ext(baseName)
		stem := strings.TrimSuffix(baseName, ext)
		uniqueName := fmt.Sprintf("%s_%d%s", stem, time.Now().UnixNano(), ext)
		trashRel = filepath.ToSlash(filepath.Join(".trash", uniqueName))
		absTrash, _, err = vault.Resolve(s.vault.Root(), trashRel)
		if err != nil {
			return "", "", mapVaultErr(err)
		}
	}
	return trashRel, absTrash, nil
}
```

O comentário de `:757-765` ("O erro do remove e CONFERIDO...") migra para cima de `moverCorpo`, que já diz o mesmo — apague a cópia.

- [ ] **Step 6: Guarda de placeholder no fallback de `moverCorpo`**

Entre `os.Rename` e `os.ReadFile` (`:826-830`):

```go
	// Volume diferente, ou rename recusado: copia e remove, com o erro do
	// remove CONFERIDO. A copia LE a origem — e ler um placeholder de nuvem
	// dispara download sincrono. O rename de um placeholder nao baixa nada;
	// a copia baixa. Quem roda antes do guarda precisa do mesmo guarda.
	if n, ok := s.index.Get(de); ok && n.CloudOnly {
		return Errorf(CodeCloudOnlyFile,
			"nota %q e somente-nuvem e o rename foi recusado; a copia de fallback a baixaria", de)
	}
	fromRaw, err := os.ReadFile(absFrom)
```

- [ ] **Step 7: Rodar `go test ./internal/service -run 'Delete|Trash|Move' -race -v`**

Expected: os dois novos passam. `TestDeleteToTrashNaoMenteQuandoORemoveFalha`
(`erro_engolido_windows_test.go:41`) vai FALHAR na asserção
`strings.Contains(err.Error(), "lixeira")`: o erro agora vem de `moverCorpo`,
que diz "a nota foi copiada para %q mas a origem %q nao pode ser removida" com
o caminho `.trash/origem.md`. A mensagem não se adapta ao teste; o teste passa
a procurar `.trash` em vez de "lixeira" — é a mesma garantia (o erro diz onde
a cópia está) com a palavra que a conta única usa. Registre a troca no commit.

- [ ] **Step 8: Prova de mutação**

Apague o guarda do Step 6, rode `-run NaoBaixaPlaceholder`, cole a saída (deve
falhar com sucesso indevido ou código INTERNAL), restaure. O portátil já
provou no Step 2 que falhava com a cópia.

- [ ] **Step 9: Gate e commit**

```bash
pwsh -File scripts/verify.ps1
git add internal/service/write.go internal/service/delete_test.go internal/service/erro_engolido_windows_test.go
git commit -m "fix(service): trash moves the note instead of copying it, and the copy fallback refuses cloud placeholders"
```

#### Verificações
Além dos passos:
1. `TestDeleteNoteToTrashMoveSemCopiar` REPROVA antes do fix (mtime muda porque o arquivo foi copiado). Se ele passar antes do fix, o teste não mede o que promete — pare e reporte `BLOCKED` com a saída, não troque a asserção.
2. `PathLocker` não é reentrante: o caminho da lixeira NÃO passa pelo `Lock` do `DeleteNote` e depois pelo de `moverCorpo`. Um deadlock aqui aparece como teste que trava; o `go test` tem `-timeout` padrão de 10 min — use `-timeout 60s` nesta Task.
3. A guarda de `CloudOnly` está no fallback de cópia de `moverCorpo`, e o teste Windows a exercita com o rename FORÇADO a falhar (trava exclusiva). Sem forçar, o rename tem sucesso e o fallback não roda — o teste mediria o caminho principal.
4. `TestDeleteToTrashNaoMenteQuandoORemoveFalha` continua passando com a asserção ajustada para `.trash`.
5. `docs/TOOLS.md` em `note_delete`: `to_trash` move (rename) e descreve o fallback e o erro `CLOUD_ONLY_FILE`.

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
  -Anchor 'if n, ok := s.index.Get(de); ok && n.CloudOnly {' `
  -Replacement 'if false {' `
  -Test TestDeleteNoteToTrashNaoBaixaPlaceholder -Package ./internal/service/
```
E, para a regra do move: mutação manual — troque a chamada a `moverCorpo` no ramo `ToTrash` por uma cópia (`os.ReadFile` + `WriteAtomic`), rode `TestDeleteNoteToTrashMoveSemCopiar`, cole a falha, restaure.

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

### Task 153: `doctor` enxerga a trava de escuta, e os sufixos das travas têm uma conta só (1.7 + 1.8)

`daemon/lock.go:118` deriva `<sock>.lock`; `:188` deriva `<sock>.listen.lock`;
`doctor/daemon.go:203` filtra `HasSuffix(nome, ".sock.lock")` — e
`x.sock.listen.lock` NÃO termina em `.sock.lock`. O `doctor` nunca lista a trava
de escuta, que é justamente a que um daemon morto sem fechar o listener deixa
para trás. `doctor/daemon.go:148` ainda faz `sock + ".log"` por conta própria
quando `daemon.CaminhoDoLog` já existe para isso (é o que o comentário de
`daemon/log.go:11-19` diz que a função veio resolver — e sobrou uma cópia).

**Files:**
- Modify: `internal/daemon/lock.go:113-119`, `:181-189`
- Modify: `internal/doctor/daemon.go:141-148`, `:203`
- Test: `internal/daemon/lock_test.go` (novo ou acrescentar ao existente, pacote `daemon`)
- Test: `internal/doctor/daemon_test.go` (acrescentar, pacote `doctor`)

**Interfaces:**
- Produces: `func EhArquivoDeTrava(nome string) bool` em `internal/daemon/lock.go`, exportada — reconhece os DOIS sufixos. Constantes `sufixoTrava = ".lock"` e `sufixoTravaDeEscuta = ".listen.lock"`, não exportadas.

- [ ] **Step 1: Teste em `daemon`**

```go
func TestEhArquivoDeTravaCobreAsDuasTravas(t *testing.T) {
	sock := "abc.sock"
	casos := map[string]bool{
		sock + sufixoTrava:         true,
		sock + sufixoTravaDeEscuta: true,
		sock:                       false,
		sock + ".log":              false,
		"abc.lock.txt":             false,
	}
	for nome, quer := range casos {
		if got := EhArquivoDeTrava(nome); got != quer {
			t.Errorf("EhArquivoDeTrava(%q) = %v, quer %v", nome, got, quer)
		}
	}
	// As constantes sao o que lockPath e ComLockDeEscuta usam de verdade.
	p, err := lockPath(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if !EhArquivoDeTrava(filepath.Base(p)) {
		t.Fatalf("lockPath produz %q, que EhArquivoDeTrava nao reconhece", p)
	}
}
```

- [ ] **Step 2: Rodar; `undefined: sufixoTrava`**

- [ ] **Step 3: Constantes e conta em `lock.go`**

Acima de `lockPath`:

```go
// Sufixos das duas travas do daemon, derivadas do caminho do socket. O doctor
// lista as travas pelo mesmo par — ate 2026-09-02 ele filtrava por ".sock.lock"
// e a trava de escuta (".sock.listen.lock") era invisivel para ele.
const (
	sufixoTrava         = ".lock"
	sufixoTravaDeEscuta = ".listen.lock"
)

// EhArquivoDeTrava diz se um nome de arquivo no diretorio de runtime e uma
// das duas travas de daemon. E a unica conta; o doctor a consome.
func EhArquivoDeTrava(nome string) bool {
	return strings.HasSuffix(nome, sufixoTrava) // ".listen.lock" tambem termina em ".lock"
}
```

`lockPath` devolve `sock + sufixoTrava`; `ComLockDeEscuta` usa `sock + sufixoTravaDeEscuta`. Acrescente `"strings"` aos imports se faltar.

- [ ] **Step 4: Teste no `doctor`** (pacote `doctor`, em `daemon_test.go`):

```go
func TestCheckLocksDeDaemonEnxergaListenLock(t *testing.T) {
	cfg := config.Config{VaultPath: t.TempDir()}
	err := daemon.ComLockDeEscuta(cfg.VaultPath, func() error {
		r := checkLocksDeDaemon(context.Background(), cfg)
		if !strings.Contains(r.Detail, "listen.lock") {
			t.Errorf("com a trava de escuta tomada, doctor disse: %q", r.Detail)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestCheckDaemonLogUsaACaminhoDoLogDoDaemon(t *testing.T) {
	cfg := config.Config{VaultPath: t.TempDir()}
	esperado, err := daemon.CaminhoDoLog(cfg.VaultPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(esperado), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(esperado, []byte("daemon iniciado\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(esperado) })
	r := checkDaemonLog(context.Background(), cfg)
	if strings.Contains(r.Detail, "ainda nao existe") {
		t.Fatalf("o log existe em %q e o doctor nao o achou: %q", esperado, r.Detail)
	}
}
```

- [ ] **Step 5: `doctor/daemon.go`**

`:141-148`: substitua `sock, err := ipc.SocketPath(...)` + `path := sock + ".log"` por `path, err := daemon.CaminhoDoLog(cfg.VaultPath)` (mesmo tratamento de erro). Se `ipc` ficar sem uso no arquivo, tire o import — confira que `doctor` continua importando `ipc` em outro arquivo (o grafo do `CLAUDE.md` diz `doctor → ipc`; se a aresta sumir de vez, atualize o grafo no mesmo commit).
`:203`: `if e.IsDir() || !daemon.EhArquivoDeTrava(e.Name()) {`.

- [ ] **Step 6: Rodar `go test ./internal/daemon ./internal/doctor -run 'EhArquivoDeTrava|ListenLock|CaminhoDoLog' -v`; passa. Mutação: volte `:203` para `".sock.lock"`, rode, `ListenLock` falha; restaure.**

- [ ] **Step 7: Gate, commit**

```bash
git add internal/daemon/lock.go internal/daemon/lock_test.go internal/doctor/daemon.go internal/doctor/daemon_test.go CLAUDE.md
git commit -m "fix(doctor): list the listen lock too, deriving both lock suffixes and the log path from daemon"
```

(`CLAUDE.md` só entra se o grafo mudou.)

#### Verificações
Além dos passos:
1. Os dois testes do `doctor` usam o diretório de runtime REAL (`ipc.SocketPath` não tem override por env em Windows); o `t.TempDir()` como cofre garante chave única, e o `t.Cleanup` remove o log. Confira que nada ficou em `%LOCALAPPDATA%\gobsidian\run\` com o nome do cofre de teste depois da execução — cole o `ls`.
2. `grep -rn '".lock"\|".listen.lock"\|".log"' internal/daemon internal/doctor` só encontra as constantes em `lock.go` e `log.go`.
3. Se `internal/doctor` deixou de importar `ipc` em todos os arquivos, o grafo em `CLAUDE.md` perde a aresta `doctor → ipc` NO MESMO COMMIT; confira com `go list -f '{{.Imports}}' ./internal/doctor`.

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
pwsh -File scripts/mutate.ps1 -Path internal/doctor/daemon.go `
  -Anchor '!daemon.EhArquivoDeTrava(e.Name())' `
  -Replacement '!strings.HasSuffix(e.Name(), ".sock.lock")' `
  -Test TestCheckLocksDeDaemonEnxergaListenLock -Package ./internal/doctor/
```
(Se `strings` não estiver importado em `daemon.go`, a mutação sai `EXIT=2` por build; use `-Replacement '!(len(e.Name()) > 10 && e.Name()[len(e.Name())-10:] == ".sock.lock")'`.)

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

### Task 155: `ipc.EhDesconexaoLimpa` é a conta única de "o outro lado foi embora" (1.10)

Três lugares decidem se um erro de transporte é encerramento normal:
`cmd/gobsidian/serve.go:73-84` (`shutdownExitCode`: `Canceled`, `EOF`,
`ErrClosedPipe`), `cmd/gobsidian/ponte.go:249-253` (os três + `os.ErrClosed`)
e `internal/daemon/daemon.go:254-258` (os quatro). `shutdownExitCode` não
conhece `os.ErrClosed` — um `serve` em processo que encerra por stdin fechado
pelo SDK com `ErrClosed` sai com código 1 e o host loga falha.

**Files:**
- Create: `internal/ipc/desconexao.go`
- Test: `internal/ipc/desconexao_test.go`
- Modify: `cmd/gobsidian/serve.go:73-84`, `cmd/gobsidian/ponte.go:249-253`, `internal/daemon/daemon.go:254-258`
- Modify: `cmd/gobsidian/serve_test.go:236-246` (acrescentar caso `os.ErrClosed`)

**Interfaces:**
- Produces: `func EhDesconexaoLimpa(err error) bool` em `internal/ipc`. `cmd` e `daemon` já importam `ipc`; nenhuma aresta nova.

- [ ] **Step 1: Teste em `ipc`**

```go
package ipc

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"testing"
)

func TestEhDesconexaoLimpaReconheceAsQuatroFormas(t *testing.T) {
	limpos := []error{nil, context.Canceled, io.EOF, io.ErrClosedPipe, os.ErrClosed,
		fmt.Errorf("copiando: %w", os.ErrClosed)}
	for _, e := range limpos {
		if !EhDesconexaoLimpa(e) {
			t.Errorf("EhDesconexaoLimpa(%v) = false", e)
		}
	}
	sujos := []error{errors.New("falha real"), context.DeadlineExceeded, io.ErrUnexpectedEOF}
	for _, e := range sujos {
		if EhDesconexaoLimpa(e) {
			t.Errorf("EhDesconexaoLimpa(%v) = true", e)
		}
	}
}
```

`DeadlineExceeded` fica de fora de propósito: prazo estourado é falha, não o
outro lado indo embora.

- [ ] **Step 2: `desconexao.go`**

```go
package ipc

import (
	"context"
	"errors"
	"io"
	"os"
)

// EhDesconexaoLimpa diz se um erro devolvido por um loop de transporte
// significa "o outro lado foi embora", e nao falha.
//
// context.Canceled vem do proprio lifecycle; io.EOF e io.ErrClosedPipe sao
// como o SDK reporta o fim do stdin; os.ErrClosed e como o fechamento de um
// pipe ou conn aparece do lado que ainda estava copiando. Tres lugares
// tinham essa lista, e um deles (shutdownExitCode) nao tinha os.ErrClosed —
// um serve que encerrava por essa via saia com codigo 1.
func EhDesconexaoLimpa(err error) bool {
	return err == nil ||
		errors.Is(err, context.Canceled) ||
		errors.Is(err, io.EOF) ||
		errors.Is(err, io.ErrClosedPipe) ||
		errors.Is(err, os.ErrClosed)
}
```

- [ ] **Step 3: Os três consumidores**

`serve.go`:

```go
func shutdownExitCode(err error) int {
	if ipc.EhDesconexaoLimpa(err) {
		return 0
	}
	return 1
}
```

`ponte.go:249-253` → `if !ipc.EhDesconexaoLimpa(loopErr) { return loopErr }` — o comentário de `:242-248` fica, com "ver ipc.EhDesconexaoLimpa" no lugar da lista.
`daemon.go:254-258` → `if !ipc.EhDesconexaoLimpa(err) { d.log.Warn(...) }`.
Tire os imports que ficarem sem uso (`io`, `os`, `context` conforme o arquivo).

- [ ] **Step 4: Caso novo em `serve_test.go`**: `{"os.ErrClosed", os.ErrClosed, 0}` e `{"prazo estourado", context.DeadlineExceeded, 1}`. Rodar: `go test ./cmd/gobsidian ./internal/ipc ./internal/daemon -run 'shutdownExitCode|Desconexao' -v`.

- [ ] **Step 5: Os quatro cenários de encerramento**

Run: `pwsh -File scripts/test_orphans.ps1`
Expected: os quatro `[OK]`. Cole a saída no relatório.

- [ ] **Step 6: Gate, commit**

```bash
git add internal/ipc/desconexao.go internal/ipc/desconexao_test.go cmd/gobsidian/serve.go cmd/gobsidian/ponte.go internal/daemon/daemon.go cmd/gobsidian/serve_test.go
git commit -m "fix(ipc): one account of a clean disconnect, and serve exits 0 on os.ErrClosed like the bridge does"
```

#### Verificações
Além dos passos:
1. Os três consumidores (`serve.go`, `ponte.go`, `daemon/daemon.go`) chamam `ipc.EhDesconexaoLimpa`; `grep -rn "io.ErrClosedPipe" cmd internal` só encontra `internal/ipc/desconexao.go` e testes.
2. `pwsh -File scripts/test_orphans.ps1` — os quatro cenários `[OK]`, saída colada. **Não rode em paralelo com nenhuma medição.**
3. `go list -f '{{.Imports}}' ./cmd/gobsidian ./internal/daemon` — `ipc` já era importado pelos dois; nenhuma aresta nova.

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
pwsh -File scripts/mutate.ps1 -Path internal/ipc/desconexao.go `
  -Anchor 'errors.Is(err, os.ErrClosed)' `
  -Replacement 'false' `
  -Test TestEhDesconexaoLimpaReconheceAsQuatroFormas -Package ./internal/ipc/
```
E no consumidor:
```bash
pwsh -File scripts/mutate.ps1 -Path cmd/gobsidian/serve.go `
  -Anchor 'if ipc.EhDesconexaoLimpa(err) {' `
  -Replacement 'if err == nil {' `
  -Test TestShutdownExitCode -Package ./cmd/gobsidian/
```
(Confira o nome real do teste de tabela em `serve_test.go:236`.)

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

### Task 156: A CLI `search` respeita `--max-results`; `index` e `inspect` param de aceitar flags que ignoram (1.11 + 5.8)

`cmd/gobsidian/search.go:50` monta `service.Options{ReadOnly: cfg.ReadOnly}` —
`cfg.MaxResults` é lido, validado e descartado; a flag `--max-results` do
`search` (`:97`) promete um teto que não aplica. `index.go:91-93` e
`inspect.go:129-131` declaram `--read-only`, `--debounce-ms` e `--max-results`
que nenhum dos dois subcomandos usa: nem escrevem, nem observam, nem buscam.
"Schema que promete e código que ignora é pior que parâmetro ausente".

**Files:**
- Modify: `cmd/gobsidian/search.go:26-28`, `:50`, `:93-94` (tirar `--read-only` e `--debounce-ms`; manter `--max-results`)
- Modify: `cmd/gobsidian/index.go:32-34`, `:91-93`; `cmd/gobsidian/inspect.go:37-39`, `:129-131` (tirar as três)
- Modify: `README.md:196-206` (tabela de flags: dizer quais subcomandos aceitam cada uma)
- Test: `cmd/gobsidian/cli_subcommands_test.go` (acrescentar)

- [ ] **Step 1: Teste**

Veja como `cli_subcommands_test.go` executa um subcomando (procure `newSearchCmd()` ou `SetArgs`); siga o mesmo padrão:

```go
func TestSearchCLIRespeitaMaxResults(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < 5; i++ {
		escreve(t, root, fmt.Sprintf("n%d.md", i), "# Nota\n\npalavra unica aqui\n")
	}
	var out bytes.Buffer
	cmd := newSearchCmd()
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--vault", root, "--json", "--max-results", "2", "palavra"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	var res struct {
		Results []json.RawMessage `json:"results"`
	}
	if err := json.Unmarshal(out.Bytes(), &res); err != nil {
		t.Fatalf("saida nao e JSON: %v\n%s", err, out.String())
	}
	if len(res.Results) != 2 {
		t.Fatalf("results = %d com --max-results 2: a flag e lida e descartada", len(res.Results))
	}
}

func TestIndexEInspectNaoAceitamFlagsQueIgnoram(t *testing.T) {
	for _, tc := range []struct {
		nome string
		cmd  func() *cobra.Command
	}{{"index", newIndexCmd}, {"inspect", newInspectCmd}} {
		for _, flag := range []string{"read-only", "debounce-ms", "max-results"} {
			if tc.cmd().Flags().Lookup(flag) != nil {
				t.Errorf("%s declara --%s e nao a usa", tc.nome, flag)
			}
		}
	}
	for _, flag := range []string{"read-only", "debounce-ms"} {
		if newSearchCmd().Flags().Lookup(flag) != nil {
			t.Errorf("search declara --%s e nao a usa", flag)
		}
	}
}
```

`escreve` é o helper de escrita de arquivo que o arquivo de teste já tem (confira o nome; se não houver, `os.WriteFile` direto). Confira os nomes reais dos construtores (`newIndexCmd`, `newInspectCmd`) com `grep -n "^func new.*Cmd" cmd/gobsidian/*.go`.

- [ ] **Step 2: Rodar; o primeiro falha com `results = 5`, o segundo lista seis flags**

- [ ] **Step 3: `search.go`**

`:50`: `service.New(v, idx, inv, nil, service.Options{ReadOnly: cfg.ReadOnly, MaxResults: cfg.MaxResults})` — confira o nome do campo em `service.Options` (`grep -n "MaxResults" internal/service/service.go`). Apague `:26-27` (`ReadOnlySet`, `DebounceMSSet`) e `:93-94` (as duas flags). `cfg.ReadOnly` segue vindo de `GOBSIDIAN_READ_ONLY` via `config.Load`; a CLI de busca nunca escreve, então `ReadOnly` em `Options` é indiferente — mantenha para não mudar o comportamento neste commit.

- [ ] **Step 4: `index.go` e `inspect.go`**: apague as três linhas `flags.*Set = ...` e as três `cmd.Flags()...` em cada um.

- [ ] **Step 5: `README.md:196-206`**: na tabela de flags, acrescente uma coluna "Subcommands" — `--vault`: all; `--read-only`, `--debounce-ms`, `--cache-dir`, `--eager-search`, `--log-level`: `serve`; `--max-results`: `serve`, `search`; `--follow-symlinks`: all; `--json`, `--limit`: `search` (e `inspect` se tiver `--json`). Confira cada flag contra o código antes de escrever a linha — a tabela é contrato.

Run: `pwsh -File scripts/check_readme_anchors.ps1`.

- [ ] **Step 6: Rodar os dois testes; gate; commit**

```bash
git add cmd/gobsidian/search.go cmd/gobsidian/index.go cmd/gobsidian/inspect.go cmd/gobsidian/cli_subcommands_test.go README.md
git commit -m "fix(cli): search honours --max-results; index and inspect stop declaring flags they ignore"
```

#### Verificações
Além dos passos:
1. `gobsidian search --help`, `index --help`, `inspect --help` — cole os três; `search` mostra `--max-results` e não mostra `--read-only`/`--debounce-ms`; os outros dois não mostram nenhuma das três.
2. `README.md` — a tabela de flags diz, por flag, quais subcomandos a aceitam; `pwsh -File scripts/check_readme_anchors.ps1` verde.
3. `cfg.MaxResults` chega a `service.Options` — o teste prova; e o `--limit` do `search` (se existir) continua funcionando como antes — cole um `search --json --limit 3`.

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
pwsh -File scripts/mutate.ps1 -Path cmd/gobsidian/search.go `
  -Anchor 'MaxResults: cfg.MaxResults' `
  -Replacement 'MaxResults: 0' `
  -Test TestSearchCLIRespeitaMaxResults -Package ./cmd/gobsidian/
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

### Task 157: A versão do cache de busca é uma conta só (1.12, fecha c2 por ora)

`search/persist.go:23` `CacheFormatVersion = 6`; `persist_codec.go:47-48`
`cacheMagic = "GBS6"`, `cacheCodecVers = 6`. Três literais para um número. O
próximo bump de formato que esqueça um deles produz um cache que passa pelo
cabeçalho e falha na decodificação — ou pior, decodifica lixo estruturalmente
válido (é o que o comentário de `:480-484` teme). O dono decidiu (c2) que
`internal/bincodec` fica adiado; o alias fecha o defeito agora.

**Files:**
- Modify: `internal/search/persist_codec.go:46-49`
- Test: `internal/search/persist_codec_test.go` (acrescentar, pacote `search`)

- [ ] **Step 1: Teste**

```go
func TestVersaoDoCacheDeBuscaEUmaConta(t *testing.T) {
	if cacheCodecVers != CacheFormatVersion {
		t.Fatalf("cacheCodecVers = %d, CacheFormatVersion = %d: duas contas", cacheCodecVers, CacheFormatVersion)
	}
	if quer := fmt.Sprintf("GBS%d", CacheFormatVersion); cacheMagic != quer {
		t.Fatalf("cacheMagic = %q, quer %q", cacheMagic, quer)
	}
}
```

- [ ] **Step 2: Rodar; passa (os três valem 6 hoje). Agora a prova: mude `CacheFormatVersion` para 7 em `persist.go`, rode, o teste FALHA nas duas asserções; restaure. Cole a saída.**

- [ ] **Step 3: Alias**

```go
const (
	// Uma conta: CacheFormatVersion e o numero; o magic e o codec derivam dele.
	// Ate 2026-09-02 eram tres literais, e um bump que esquecesse um deles
	// produziria um cache que passa pelo cabecalho e falha — ou decodifica
	// lixo estruturalmente valido — no corpo.
	cacheCodecVers = CacheFormatVersion
)

var cacheMagic = fmt.Sprintf("GBS%d", CacheFormatVersion)
```

`cacheMagic` deixa de ser `const`; os usos (`:178`, `:457-462`) já o tratam como string e compilam. Se `fmt` não estiver importado em `persist_codec.go`, acrescente. Se preferir manter `const` — `cacheMagic = "GBS" + string(rune('0'+CacheFormatVersion))` só vale até 9; o `var` é mais honesto.

- [ ] **Step 4: Repetir o Step 2 (bump para 7 → agora o teste PASSA, porque tudo derivou; e `TestLoadInvertedCache*` devem seguir passando com cache gravado e lido pela mesma versão). Restaure. Rodar `go test ./internal/search`.**

- [ ] **Step 5: Gate, commit**

```bash
git add internal/search/persist_codec.go internal/search/persist_codec_test.go
git commit -m "fix(search): the cache format version is one constant; magic and codec derive from it"
```

#### Verificações
Além dos passos:
1. A prova de mutação desta Task é o bump temporário (Step 2 e Step 4): ANTES do alias, bump para 7 faz o teste reprovar; DEPOIS, bump para 7 faz tudo derivar e o teste passa. Cole as DUAS saídas.
2. `grep -rn "GBS6\|cacheCodecVers = 6" internal/search` vazio depois.
3. Um cache gravado por esta versão e lido por ela mesma continua carregando: `go test ./internal/search -run 'Cache|Persist' -v` verde.

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

Não há `mutate.ps1` aqui: a prova é o bump manual de `CacheFormatVersion` para 7 descrito nos Steps 2 e 4, com as duas saídas coladas e o restauro confirmado por `git diff --stat internal/search/persist.go` vazio.

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

### Task 159: `internal/vaulttest` — handle exclusivo, placeholder somente-nuvem e prazo comum

**Files:**
- Create: `internal/vaulttest/doc.go`
- Create: `internal/vaulttest/exclusivo_windows.go`
- Create: `internal/vaulttest/exclusivo_other.go`
- Create: `internal/vaulttest/somentenuvem_windows.go`
- Create: `internal/vaulttest/somentenuvem_other.go`
- Create: `internal/vaulttest/prazo.go`
- Create: `internal/vaulttest/exclusivo_windows_test.go`
- Modify: `internal/search/cloudonly_update_windows_test.go:44-66` (apagar `travarExclusivo`, usar `vaulttest.TravarExclusivo`)
- Modify: `internal/service/erro_engolido_windows_test.go:23` (apagar `travaExclusiva`; `:60-66` tirar a guarda)
- Modify: `internal/service/move_atomico_windows_test.go:70-85` (tirar a guarda `if origemExiste && destinoExiste && err == nil`)
- Modify: `internal/index/replace_duas_fases_windows_test.go:22` (apagar `travaLeitura`)
- Modify: `internal/index/build_descarte_windows_test.go:10` (apagar `lockFileForTest`)
- Modify: `internal/index/classify_cloudonly_windows_test.go:24` (apagar `marcarSomenteNuvem`)
- Modify: `cmd/gobsidian/boot_indice_busca_windows_test.go:28` (apagar `marcarOffline`)
- Modify: `internal/vault/walk_windows_test.go` (se a Task 147 deixou `travarDiretorioExclusivo` ali: migrar)
- Modify: `internal/daemon/daemon_test.go:23`, `internal/ipc/ipc_test.go:23`, `cmd/gobsidian/serve_test.go:18` (apagar `boundedWait` local, usar `vaulttest.Prazo`)
- Modify: `docs/ESTRUTURA.md` (entrada `internal/vaulttest/`), `docs/papeis/testador.md` (seção "handle exclusivo": apontar para o pacote)
- Modify: `CLAUDE.md` (grafo: `vaulttest → vault`, marcado "só em _test" — é pacote de teste, importado só por `_test.go`)

**Interfaces:**
- Consumes: `vault.LongPath(abs string) string` (`internal/vault/longpath_windows.go:27`).
- Produces (pacote `vaulttest`, importado apenas por arquivos `_test.go` de pacotes externos `*_test` ou `package main`):
  - `func TravarExclusivo(t testing.TB, abs string)` — Windows: `CreateFile(GENERIC_READ|GENERIC_WRITE, share=0)`, `t.Cleanup` fecha, e **prova** que `os.ReadFile(abs)` falha antes de devolver (o modelo de `search/cloudonly_update_windows_test.go:50`). Outros SO: `t.Skip("handle exclusivo e semantica do Windows")`.
  - `func TravarDiretorioExclusivo(t testing.TB, abs string)` — mesma coisa sobre diretório (`FILE_FLAG_BACKUP_SEMANTICS`), prova que `os.ReadDir(abs)` falha. Outros SO: Skip.
  - `func MarcarSomenteNuvem(t testing.TB, abs string)` — Windows: `FILE_ATTRIBUTE_OFFLINE`, cleanup restaura `NORMAL`, e prova `vault.IsCloudOnly` (ou o `Classify` equivalente que o pacote expõe — ver Verificações) devolve verdadeiro antes de devolver. Outros SO: Skip.
  - `const Prazo = 5 * time.Second` — o único `boundedWait`. A Task 165 depende dele.

**Por que um pacote de teste e não cópias:** cinco cópias de "handle exclusivo" e só uma prova que a trava trava (`docs/ARMADILHAS.md` já registra o mecanismo: "um handle exclusivo que pede só GENERIC_READ não barra `os.ReadFile`"). As quatro que não provam deixam `service/erro_engolido_windows_test.go:60-66` e `move_atomico_windows_test.go:70-85` guardarem toda asserção com `if … && err == nil` — se o share mode permitir a leitura, os testes passam sem afirmar nada. Um pacote com a prova dentro do helper faz a asserção incondicional.

**Ciclo de import:** `vaulttest` importa `vault`. Só arquivos `package vault_test` (externo) podem usá-lo dentro de `internal/vault/` — `walk_windows_test.go` é `vault_test`, `cloudonly_info_windows_test.go` é `package vault` e **fica como está**. `vaulttest` **não** importa `index`, `search`, `service`, `parser` — é isto que impede o ciclo nos testes externos desses pacotes. Não é `helpers.go`: o pacote tem um nome de domínio e cada arquivo uma responsabilidade.

- [ ] **Step 1: Criar o pacote com a prova dentro do helper**

`internal/vaulttest/doc.go`:

```go
// Package vaulttest oferece aos testes de outros pacotes as condicoes de
// ambiente que o produto promete respeitar e que so o sistema operacional
// pode criar: um arquivo ou diretorio que NAO pode ser aberto, um arquivo
// somente-nuvem, e o prazo unico de espera dos testes.
//
// Cada helper PROVA a condicao antes de devolver. Um handle exclusivo que nao
// barra a leitura tornaria vazia toda asserção de "nao abriu" — foi o que
// aconteceu quando cinco copias divergiram e so uma conferia.
//
// Importado apenas por arquivos _test.go. Importa vault e mais nada do
// dominio, para nunca fechar ciclo com quem o usa.
package vaulttest
```

`internal/vaulttest/exclusivo_windows.go`:

```go
//go:build windows

package vaulttest

import (
	"os"
	"testing"

	"github.com/jonyd/gobsidian/internal/vault"
	"golang.org/x/sys/windows"
)

// TravarExclusivo segura um handle exclusivo sobre o arquivo ate o fim do
// teste e CONFERE que ele barra os.ReadFile antes de devolver.
//
// Leitura E escrita: um handle exclusivo que pede so GENERIC_READ nao barra
// o os.ReadFile (medido; ver docs/ARMADILHAS.md).
func TravarExclusivo(t testing.TB, abs string) {
	t.Helper()
	h := abrirExclusivo(t, abs, 0)
	t.Cleanup(func() { _ = windows.CloseHandle(h) })
	if _, err := os.ReadFile(abs); err == nil {
		t.Fatalf("vaulttest: o handle exclusivo nao barrou a leitura de %s; a prova de 'nao abriu' seria vazia", abs)
	}
}

// TravarDiretorioExclusivo e o equivalente para diretorio: com share mode 0 o
// os.ReadDir falha, que e a condicao que vault.Walk e o watcher precisam
// distinguir de "diretorio vazio".
func TravarDiretorioExclusivo(t testing.TB, abs string) {
	t.Helper()
	h := abrirExclusivo(t, abs, windows.FILE_FLAG_BACKUP_SEMANTICS)
	t.Cleanup(func() { _ = windows.CloseHandle(h) })
	if _, err := os.ReadDir(abs); err == nil {
		t.Fatalf("vaulttest: o handle exclusivo nao barrou a listagem de %s", abs)
	}
}

func abrirExclusivo(t testing.TB, abs string, flags uint32) windows.Handle {
	t.Helper()
	p, err := windows.UTF16PtrFromString(vault.LongPath(abs))
	if err != nil {
		t.Fatal(err)
	}
	h, err := windows.CreateFile(p, windows.GENERIC_READ|windows.GENERIC_WRITE, 0, nil,
		windows.OPEN_EXISTING, windows.FILE_ATTRIBUTE_NORMAL|flags, 0)
	if err != nil {
		t.Skipf("vaulttest: nao foi possivel abrir %s em modo exclusivo: %v", abs, err)
	}
	return h
}
```

`internal/vaulttest/exclusivo_other.go`:

```go
//go:build !windows

package vaulttest

import "testing"

// TravarExclusivo nao existe fora do Windows: POSIX nao tem share mode. O
// teste que depende dele pula, e o nome do pulo diz por que.
func TravarExclusivo(t testing.TB, _ string) {
	t.Helper()
	t.Skip("handle exclusivo e semantica do Windows")
}

func TravarDiretorioExclusivo(t testing.TB, _ string) {
	t.Helper()
	t.Skip("handle exclusivo e semantica do Windows")
}
```

`internal/vaulttest/somentenuvem_windows.go` — transcrever `marcarSomenteNuvem` de `internal/index/classify_cloudonly_windows_test.go:24-44` (incluindo o comentário sobre `RECALL_ON_DATA_ACCESS` não ser gravável) como `func MarcarSomenteNuvem(t testing.TB, abs string)`, e acrescentar antes do `return` a prova:

```go
	if !vault.IsCloudOnly(abs) {
		t.Fatalf("vaulttest: FILE_ATTRIBUTE_OFFLINE nao fez vault.IsCloudOnly(%s) responder verdadeiro", abs)
	}
```

(Se `vault.IsCloudOnly` tiver assinatura diferente — conferir com `gopls` em `internal/vault/cloudonly*.go` — usar a que existe. **Não** criar função nova em `vault` para isso.)

`internal/vaulttest/somentenuvem_other.go`: `MarcarSomenteNuvem` que faz `t.Skip("FILE_ATTRIBUTE_OFFLINE e atributo NTFS")`.

`internal/vaulttest/prazo.go`:

```go
package vaulttest

import "time"

// Prazo e o unico limite de espera dos testes que aguardam algo assincrono
// (socket, handshake, desligamento). Um defeito real nao pode travar
// "go test -race ./..." ate o timeout de 10 minutos; e tres pacotes com tres
// valores (2 s, 3 s, 5 s) eram tres respostas para a mesma pergunta.
const Prazo = 5 * time.Second
```

- [ ] **Step 2: Teste do próprio pacote (Windows)**

`internal/vaulttest/exclusivo_windows_test.go` (`package vaulttest_test`):

```go
//go:build windows

package vaulttest_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jonyd/gobsidian/internal/vaulttest"
)

// A prova dentro do helper e o que o pacote vende. Este teste confere que ela
// existe: depois de TravarExclusivo, a leitura falha; depois do Cleanup, volta
// a funcionar (o handle foi fechado e nao vazou para o proximo teste).
func TestTravarExclusivoBarraLeituraEDevolveNoCleanup(t *testing.T) {
	abs := filepath.Join(t.TempDir(), "a.md")
	if err := os.WriteFile(abs, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Run("travado", func(t *testing.T) {
		vaulttest.TravarExclusivo(t, abs)
		if _, err := os.ReadFile(abs); err == nil {
			t.Fatal("leitura passou com handle exclusivo aberto")
		}
	})
	if _, err := os.ReadFile(abs); err != nil {
		t.Fatalf("depois do Cleanup a leitura devia voltar: %v", err)
	}
}

func TestTravarDiretorioExclusivoBarraListagem(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "sub")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Run("travado", func(t *testing.T) {
		vaulttest.TravarDiretorioExclusivo(t, dir)
		if _, err := os.ReadDir(dir); err == nil {
			t.Fatal("ReadDir passou com handle exclusivo aberto")
		}
	})
	if _, err := os.ReadDir(dir); err != nil {
		t.Fatalf("depois do Cleanup a listagem devia voltar: %v", err)
	}
}
```

Run: `go test ./internal/vaulttest/ -run 'TestTravar' -v` — Expected: PASS nos dois.

- [ ] **Step 3: Migrar os seis helpers e tirar as guardas**

Para cada arquivo listado em **Files**, apagar o helper local e substituir a chamada por `vaulttest.X`. Nas duas guardas de `service`:

`internal/service/move_atomico_windows_test.go:70-85` — o bloco hoje é `if origemExiste && destinoExiste && err == nil { … asserções … }`. Fica:

```go
	if !origemExiste || !destinoExiste {
		t.Fatalf("o move nao pode ter apagado a origem nem criado o destino com o handle exclusivo aberto: origem=%v destino=%v", origemExiste, destinoExiste)
	}
	if err == nil {
		t.Fatal("MoveNote devia falhar com o destino travado")
	}
	// … asserções que estavam dentro do if, agora incondicionais …
```

`internal/service/erro_engolido_windows_test.go:60-66` — mesma transformação: o `if` que envolve as asserções vira `t.Fatalf` na negação. Ler o teste antes: as asserções que ele guarda são o contrato do teste; nenhuma pode ser perdida.

Run: `go test -race ./internal/... ./cmd/... 2>&1 | tail -30` — Expected: `ok` em todos; os testes Windows que antes podiam passar em silêncio agora afirmam.

- [ ] **Step 4: Prova de que a guarda removida vale**

Em `internal/service/move_atomico_windows_test.go`, trocar temporariamente a chamada `vaulttest.TravarExclusivo(t, destino)` por nada (comentar a linha). Rodar `go test ./internal/service/ -run TestMoveAtomico -v`. Expected: **FAIL** — antes desta tarefa, sem trava o teste passava em silêncio; agora `t.Fatal("MoveNote devia falhar com o destino travado")` nomeia. Restaurar. Colar a saída no relatório.

- [ ] **Step 5: `boundedWait` → `vaulttest.Prazo`**

Em `internal/daemon/daemon_test.go:23`, `internal/ipc/ipc_test.go:23`, `cmd/gobsidian/serve_test.go:18`: apagar a const local e o comentário; `gopls rename` não serve (a const some), então substituir cada uso por `vaulttest.Prazo` e acrescentar o import. `cmd/gobsidian/serve_test.go` usava 2 s: o prazo sobe para 5 s — só afeta o pior caso (falha), não o caso feliz.

Run: `go test -race ./internal/daemon/ ./internal/ipc/ ./cmd/... 2>&1 | tail -5` — Expected: `ok`.

- [ ] **Step 6: Documentação e grafo**

`CLAUDE.md`, bloco "Quatro arestas existem só em teste": acrescentar `vaulttest → vault` na lista de teste e uma linha dizendo que `vaulttest` é importado só por `_test.go`. `docs/ESTRUTURA.md`: entrada `internal/vaulttest/` com uma linha. `docs/papeis/testador.md`: onde fala de handle exclusivo, apontar para `vaulttest.TravarExclusivo` e apagar a receita inline se houver.

Run: `pwsh -File scripts/verify.ps1` — Expected: verde, 14 etapas.

- [ ] **Step 7: Commit**

```bash
git add internal/vaulttest/ internal/search/cloudonly_update_windows_test.go internal/service/erro_engolido_windows_test.go internal/service/move_atomico_windows_test.go internal/index/replace_duas_fases_windows_test.go internal/index/build_descarte_windows_test.go internal/index/classify_cloudonly_windows_test.go cmd/gobsidian/boot_indice_busca_windows_test.go internal/vault/walk_windows_test.go internal/daemon/daemon_test.go internal/ipc/ipc_test.go cmd/gobsidian/serve_test.go docs/ESTRUTURA.md docs/papeis/testador.md CLAUDE.md
git commit -m "test(vaulttest): one exclusive-handle helper that proves it locks, and one wait budget"
```

#### Verificações

- `go list -deps ./internal/vaulttest/ | grep gobsidian` mostra **só** `internal/vault` e `internal/vaulttest` (e `internal/text` se `vault` o importar — conferir; nada mais).
- `grep -rn "CreateFile(" --include=*_test.go internal/ cmd/` devolve zero fora de `internal/vaulttest/`.
- `grep -rn "boundedWait" --include=*_test.go .` devolve zero.
- `grep -rn "FILE_ATTRIBUTE_OFFLINE" --include=*_test.go internal/ cmd/` devolve zero fora de `internal/vaulttest/`.
- Step 4 colado no relatório com o FAIL.
- `verify.ps1` verde; `GOOS=linux go vet ./...` verde (os `_other.go` compilam).

#### Regras de execução

- Nunca `git checkout/restore/stash/clean/reset`. Commit por caminho explícito.
- Sem `helpers.go`/`utils.go`/`common.go`. Um arquivo por responsabilidade.
- `vaulttest` não ganha import de `index`, `search`, `service`, `parser`, `mcpsrv` — se um teste precisar de algo desses, o helper não pertence aqui.
- Código de plataforma atrás de build tag em arquivo separado.
- Não apagar asserção nenhuma dos testes migrados; só a guarda `if` que as tornava condicionais.

#### Comando de mutação

Esta tarefa não tem prova de mutação por `mutate.ps1`: a regra provada é de teste, não de produto, e a prova é o Step 4 (remover a trava, ver o FAIL nomeado, restaurar).

#### Contrato de relatório

`.superpowers/sdd/2026-09-02-topografia-e-limpeza/task-159-report.md`: status, SHA do commit, saída de `go list -deps`, saída do Step 4 (FAIL) e do `verify.ps1` (última linha), lista dos seis helpers apagados com `arquivo:linha` original.

### Task 160: Apagar ou consertar os cinco testes verificados que não podem falhar

**Files:**
- Modify: `internal/watcher/overflow_test.go:219` (`TestReconcile_CtxCancelStopsEarly`)
- Modify: `internal/index/build_test.go:143` (`TestBuildSkipsUnreadableFile` — apagar)
- Delete: `internal/index/normalizacao_equivalente_test.go`
- Modify: `internal/mcpsrv/schema_params_test.go:171,:204,:237`
- Modify: `internal/service/read_test.go:115` (apagar o teste com `t.Skip` incondicional)

**Interfaces:** nenhuma.

Cada sítio foi lido e o mecanismo pelo qual o teste não pode falhar está descrito abaixo. O implementador **confirma** o mecanismo antes de editar (o Step 1 de cada item); se a leitura discordar, `BLOCKED` com o motivo, não edição silenciosa.

- [ ] **Step 1: `overflow_test.go:219` — dar ao cancelamento algo para interromper**

Mecanismo: entre `Build` e `Reconcile` nada muda no cofre; o atalho `overflow.go:58-61` devolve `updated == 0` com ou sem cancelamento, logo `updated >= 200` (a asserção "parou cedo") é inalcançável e o teste passa vazio.

Conserto: criar 300 notas, `Build`, **modificar 300 notas** (reescrever com conteúdo diferente e mtime avançado — `os.Chtimes` +2 s), cancelar o `ctx` **antes** de `Reconcile`, e afirmar `updated < 300`. Com o `ctx` já cancelado, `Reconcile` deve parar no primeiro check; sem o check (mutação), processa as 300.

```go
func TestReconcileCtxCanceladoParaAntesDeProcessarTudo(t *testing.T) {
	root := t.TempDir()
	const n = 300
	for i := range n {
		escreverNota(t, root, fmt.Sprintf("n%03d.md", i), "# v1\n")
	}
	idx := construirIndice(t, root) // o helper que o arquivo ja usa para Build
	depois := time.Now().Add(2 * time.Second)
	for i := range n {
		p := escreverNota(t, root, fmt.Sprintf("n%03d.md", i), "# v2\n")
		if err := os.Chtimes(p, depois, depois); err != nil {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	st := reconcile(ctx, idx, root) // a chamada que o teste original faz
	if st.Updated >= n {
		t.Fatalf("Reconcile processou %d de %d com ctx cancelado antes de comecar; o cancelamento nao e respeitado", st.Updated, n)
	}
}
```

Adaptar os nomes `escreverNota`/`construirIndice`/`reconcile`/`st.Updated` aos que **existem** no arquivo (ler `overflow_test.go:1-60` e o teste original). Apagar `TestReconcile_CtxCancelStopsEarly`.

Prova: `pwsh -File scripts/mutate.ps1 -Path internal/watcher/overflow.go -Anchor "<a linha do select/if ctx.Err() dentro do loop de Reconcile>" -Replacement "<a mesma linha com a condição sempre falsa>" -Test TestReconcileCtxCanceladoParaAntesDeProcessarTudo -Package ./internal/watcher/` — Expected: exit 0.

- [ ] **Step 2: `build_test.go:143` — apagar**

Mecanismo: o teste cria um **diretório** chamado `unreadable.md` esperando que `Build` o pule por erro de leitura; mas `walk.go:152` devolve antes de `Classify` para qualquer diretório — nunca há leitura. O comentário do teste descreve um cenário que não acontece. A cobertura real de "arquivo ilegível é pulado" é `build_descarte_test.go:16` (e a versão Windows com handle exclusivo).

Apagar `TestBuildSkipsUnreadableFile` inteiro (função e comentário). Se algum helper ficar sem uso, apagar também.

- [ ] **Step 3: `normalizacao_equivalente_test.go` — apagar**

Mecanismo: compara `normalizeString` com `text.Normalize`; `query.go:39-41` é `return text.Normalize(s)`. Tautologia. Se, depois de apagar, `normalizeString` ficar sem chamador de teste, tudo bem — é código de produto e tem seus chamadores.

`git rm internal/index/normalizacao_equivalente_test.go`.

- [ ] **Step 4: `schema_params_test.go:171,:204,:237` — checar o erro e o tamanho**

Mecanismo: `json.Unmarshal(…, &got)` com `_` no erro, depois `for _, e := range got.Edges { … }` — um JSON inválido ou uma lista vazia passa sem afirmar nada.

Em cada um dos três sítios:

```go
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("StructuredContent nao e o JSON esperado: %v\n%s", err, raw)
	}
	if len(got.Edges) == 0 {
		t.Fatal("resposta sem arestas; o teste nao exercitou nada")
	}
```

(`got.Edges` é o campo do sítio `:171`; ler os outros dois e usar o campo iterado em cada um.)

Prova por mutação de **teste**: trocar temporariamente `raw` por `[]byte("{}")` num dos três, rodar, ver FAIL em `len(...) == 0`, restaurar. Colar.

- [ ] **Step 5: `read_test.go:115` — apagar**

Mecanismo: `t.Skip` incondicional na primeira linha; a cobertura pretendida (nota somente-nuvem não é aberta) existe em `internal/service/cloudonly_replace_windows_test.go:33` com handle exclusivo. Apagar a função e o comentário.

- [ ] **Step 6: Gate e commit**

Run: `pwsh -File scripts/verify.ps1` — Expected: verde.

```bash
git add internal/watcher/overflow_test.go internal/index/build_test.go internal/index/normalizacao_equivalente_test.go internal/mcpsrv/schema_params_test.go internal/service/read_test.go
git commit -m "test: remove five tests that could not fail, and make the ctx-cancel one able to"
```

#### Verificações

- `go test ./internal/index/ -run 'TestBuildSkipsUnreadableFile|TestNormaliza' -v` imprime `no tests to run`.
- `go test ./internal/service/ -run TestReadNoteCloudOnly -v` não imprime `SKIP` (a função foi apagada, não pulada).
- `mutate.ps1` do Step 1 com exit 0 e a saída colada.
- Step 4 com o FAIL colado.
- `verify.ps1` verde.

#### Regras de execução

- Nunca `git checkout/restore/stash/clean/reset`. `git rm` por caminho explícito é permitido (é o que a tarefa pede).
- Se a leitura de um sítio contradisser o mecanismo descrito, `BLOCKED` naquele item e seguir com os outros — não "consertar" um teste que se entendeu mal.
- Nomes de teste novos em português sem acento, `TestVerboObjeto`, como o resto do repositório.

#### Comando de mutação

`pwsh -File scripts/mutate.ps1 -Path internal/watcher/overflow.go -Anchor "<condição de ctx no loop de Reconcile>" -Replacement "<mesma linha, condição sempre falsa>" -Test TestReconcileCtxCanceladoParaAntesDeProcessarTudo -Package ./internal/watcher/` — o implementador preenche o Anchor lendo `overflow.go`, e cola o comando **como rodou** no relatório. Exit esperado: 0.

#### Contrato de relatório

`task-160-report.md`: status, SHA, para cada um dos cinco itens uma linha "mecanismo confirmado: sim/não — <evidência>", saída do `mutate.ps1`, saída do Step 4, última linha do `verify.ps1`.

### Task 161: Auditoria dos treze testes reportados pelo sweep — cada um vira conserto, apagamento ou "falsifica, mantido"

**Files:**
- Modify (conforme veredito): `internal/mcpsrv/tools_read_test.go:96-133`, `internal/service/limites_enums_test.go:44`, `internal/mcpsrv/server_test.go:116`, `internal/search/bm25_test.go:213`, `internal/search/persist_test.go:232`, `internal/search/pool_test.go:9`, `internal/service/write_test.go:242,:368,:384`, `internal/index/slug_persistido_test.go:84`, `cmd/gobsidian/cli_subcommands_test.go:168`, `cmd/gobsidian/console_saida_test.go:100`, `internal/watcher/filter_test.go:117`, `internal/search/inverted_test.go:118`

**Interfaces:** nenhuma.

Estes treze foram apontados por uma varredura e são coerentes com o código, mas **não foram re-lidos um a um** (o relatório de 2026-09-02 diz isso explicitamente). Esta tarefa é a re-leitura. O produto é uma tabela no relatório com **um veredito por sítio**, e o diff que os vereditos exigem.

- [ ] **Step 1: Para cada sítio, o mesmo protocolo**

1. Ler o teste inteiro e a função de produto que ele exercita.
2. Escrever, em uma frase, **qual regra de produto** o teste afirma.
3. Apagar essa regra no produto (edição temporária: inverter a condição, devolver zero, devolver `nil`), rodar **só aquele teste**, anotar PASS ou FAIL. Restaurar.
4. Veredito:
   - **FAIL sob mutação** → "falsifica, mantido". Sem diff.
   - **PASS sob mutação, e a regra tem outro teste que falsifica** → apagar o teste (dizer qual é o outro).
   - **PASS sob mutação, e a regra não tem outro teste** → consertar: acrescentar a asserção que falta, repetir o passo 3 até FAIL.

Pistas por sítio (a leitura decide, não a pista):

| Sítio | Pista |
|---|---|
| `tools_read_test.go:96-133` | 6 de 7 subtestes só checam `!IsError && StructuredContent != nil`; zero hits passa. Consertar: afirmar sobre o conteúdo (`len(results) > 0`, o path esperado). |
| `limites_enums_test.go:44` | Resultado descartado; o clamp de `MaxResults` é inobservável. Consertar: afirmar `len(res.Results) <= limite`. |
| `server_test.go:116` | Ver o que afirma. |
| `bm25_test.go:213` | Testa filtro de NaN, mas `bm25.go:193` já filtra antes. Provável "apagar" se outro teste cobre `:193`, senão mover a asserção para onde o NaN pode entrar. |
| `persist_test.go:232` | Não chama código de produção. Provável apagar. |
| `pool_test.go:9` | Idem. |
| `write_test.go:242,:368,:384` | Hash esperado calculado a partir do mesmo índice que o produto usa — tautologia. Consertar: hash esperado calculado **do arquivo** (`sha256` do conteúdo lido do disco), não do índice. |
| `slug_persistido_test.go:84` | Ver o que afirma. |
| `cli_subcommands_test.go:168` | Ver o que afirma. |
| `console_saida_test.go:100` | Aborta antes de chegar em `serve`; o que quer provar (stdout limpo?) não é exercitado. Consertar ou apagar. |
| `filter_test.go:117` | `root` hardcoded `C:\` — o próprio arquivo registra 4 dias de CI vermelho por isso. Consertar: `t.TempDir()`. |
| `inverted_test.go:118` | `DocCount() < 0` — `int` sem sinal de negativo possível? Ver o tipo; se `int`, a asserção é vazia. Consertar com o valor esperado exato. |

- [ ] **Step 2: Rodar o pacote de cada arquivo tocado**

Run: `go test -race ./internal/mcpsrv/ ./internal/service/ ./internal/search/ ./internal/index/ ./internal/watcher/ ./cmd/... 2>&1 | tail -10` — Expected: `ok`.

- [ ] **Step 3: Gate e commit**

Run: `pwsh -File scripts/verify.ps1` — Expected: verde.

```bash
git add <cada arquivo tocado, por caminho>
git commit -m "test: audit thirteen sweep-flagged tests; fix the ones that could not fail, drop the redundant ones"
```

#### Verificações

- A tabela do relatório tem **treze linhas** com: sítio, regra afirmada (uma frase), mutação aplicada (uma linha), resultado (PASS/FAIL), veredito, ação.
- Para cada "consertado": saída do teste **falhando** sob a mutação, colada.
- Para cada "apagado": nome do outro teste que falsifica a mesma regra.
- `verify.ps1` verde.

#### Regras de execução

- Nunca `git checkout/restore/stash/clean/reset`. Mutação temporária é edição manual seguida de edição manual de volta; conferir com `git diff` que o produto ficou intacto **antes** do commit (o diff só pode tocar `_test.go`).
- Um veredito por sítio. "Provavelmente" não é veredito.
- Não consertar produto nesta tarefa. Se a mutação revelar um defeito real de produto, registrar no relatório como "defeito encontrado — fora de escopo" e seguir.

#### Comando de mutação

Cada linha da tabela é uma mutação manual; a tarefa não usa `mutate.ps1` porque as mutações são de asserção de teste, não de uma regra única de produto. As saídas coladas são a prova.

#### Contrato de relatório

`task-161-report.md`: status, SHA, a tabela de treze linhas, saídas coladas, `git diff --stat` do commit confirmando só `_test.go`.

### Task 162: Confundidores — corpus do ranking sem empate, contexto de backlink, ordem de mapa, `idx == nil`

**Files:**
- Modify: `internal/service/ranking_golden_test.go:60-110` e `testdata/ranking/*.tsv` (regenerar)
- Modify: `internal/index/backlink_contexto_test.go:66`, `internal/index/backlink_heading_test.go:94`
- Modify: `internal/index/persist_test.go:171`
- Modify: `internal/search/bm25_test.go:127`

**Interfaces:** nenhuma.

- [ ] **Step 1: `ranking_golden_test.go` — corpus com variação, não com um vencedor**

Mecanismo: 299 de 300 notas são idênticas; `n0150` tem uma frase três tokens mais curta e vence os quatro goldens **por ser a mais curta** — inclusive `so-em-heading`, cujo heading é igual nas 300. As linhas 2..N congelam a ordem de inserção. O golden não testa ranking; testa o desempate.

Conserto do gerador (`:60-110`, a função que monta as 300 notas):

- Cada nota `i` recebe: tamanho de corpo `20 + (i*7)%60` tokens de enchimento (varia comprimento); o termo de `termo-amplo` aparece `1 + i%4` vezes; o termo de `dois-termos` só nas notas `i%3 == 0`, segundo termo só em `i%5 == 0`; o termo acentuado só em `i%7 == 0`; o heading de `so-em-heading` só em `i%11 == 0` **e o termo não aparece no corpo dessas**.
- Determinístico (sem `rand`): as fórmulas acima bastam.

Regenerar os quatro `.tsv` com o mecanismo que o teste já tem (`-update` flag ou equivalente — ler o arquivo; se não houver, criar `var atualizar = flag.Bool("update", false, "reescreve os goldens")`).

Verificação de que o novo corpus discrimina: em cada golden, o **primeiro** resultado tem score estritamente maior que o **segundo** (`>`), e os 20 primeiros não são 20 scores iguais. Acrescentar ao teste:

```go
	if len(got) >= 2 && got[0].Score <= got[1].Score {
		t.Fatalf("%s: os dois primeiros empatam (%.6f); o corpus nao discrimina", nome, got[0].Score)
	}
```

Atualizar a docstring `:60-70`: o sintoma diagnosticado ("exatamente 20 linhas") era o desempate de inserção, e a causa era o corpus.

Prova: `mutate.ps1` sobre `internal/search/bm25.go` — Anchor na linha que aplica o peso de heading (`WeightHeading` ou o nome real), Replacement com o peso igual ao de corpo; Test `TestRankingGolden`; Expected exit 0 (o golden `so-em-heading` muda de ordem).

- [ ] **Step 2: `backlink_contexto_test.go:66`, `backlink_heading_test.go:94`**

Mecanismo: uma implementação que devolve só o `Raw` do link como contexto passa, porque o contexto esperado contém o link.

Conserto: afirmar que o contexto contém texto **ao redor** do link que não é o link — escolher no fixture uma palavra que aparece só na linha do link e fora dos colchetes, e afirmar `strings.Contains(ctx, "essa-palavra")`. Nos dois arquivos.

Prova manual: no produto, trocar temporariamente o contexto por `link.Raw`, rodar os dois testes, ver FAIL, restaurar. Colar.

- [ ] **Step 3: `persist_test.go:171` — ordem de mapa**

Mecanismo: `reflect.DeepEqual` sobre slice derivada de iteração de mapa; passa porque há uma origem só.

Conserto: fixture com **duas** origens e comparação por conjunto (ordenar as duas slices com `slices.SortFunc` antes do `DeepEqual`, ou usar `cmp.Diff` com `cmpopts.SortSlices` se `go-cmp` já for dep — conferir `go.mod`; **não** adicionar dep).

- [ ] **Step 4: `bm25_test.go:127` — `idx == nil`**

Mecanismo: com `idx == nil` todo campo é `WeightBody` por definição; o teste de pesos por campo não distingue heading de corpo.

Conserto: montar um índice mínimo (o helper de "vault+Build+Inverted" que o pacote já tem em `persist_test.go:271`, com strip de BOM) com uma nota cujo termo está só no heading e outra só no corpo, e afirmar `scoreHeading > scoreBody`.

Prova: a mesma mutação do Step 1 (peso de heading = peso de corpo) deve fazer este teste falhar também. `mutate.ps1 … -Test TestBM25PesoDeHeading -Package ./internal/search/` — exit 0.

- [ ] **Step 5: Gate e commit**

Run: `pwsh -File scripts/verify.ps1` — Expected: verde.

```bash
git add internal/service/ranking_golden_test.go testdata/ranking/ internal/index/backlink_contexto_test.go internal/index/backlink_heading_test.go internal/index/persist_test.go internal/search/bm25_test.go
git commit -m "test: ranking corpus that discriminates, backlink context beyond the link, map order and nil-index confounders"
```

#### Verificações

- Os quatro `.tsv` regenerados têm primeira linha com score estritamente maior que a segunda (colar `head -2` de cada).
- `mutate.ps1` do Step 1 e do Step 4 com exit 0, saídas coladas.
- Step 2 com o FAIL colado.
- `verify.ps1` verde.

#### Regras de execução

- Nunca `git checkout/restore/stash/clean/reset`.
- Sem dependência nova em `go.mod`; nunca `go mod tidy`.
- Não tocar `internal/search/bm25.go` fora das mutações temporárias; `git diff --stat` do commit só tem `_test.go` e `testdata/`.

#### Comando de mutação

`pwsh -File scripts/mutate.ps1 -Path internal/search/bm25.go -Anchor "<linha do peso de heading>" -Replacement "<mesma linha com peso de corpo>" -Test 'TestRankingGolden|TestBM25PesoDeHeading' -Package ./internal/service/` — e o mesmo com `-Package ./internal/search/`. O implementador cola os dois comandos como rodaram. Exit esperado: 0 nos dois.

#### Contrato de relatório

`task-162-report.md`: status, SHA, `head -2` dos quatro `.tsv`, as duas saídas de `mutate.ps1`, o FAIL do Step 2, `git diff --stat`.

### Task 163: RNF-5000 afirma ou não existe; paridade itera `got` e falha em vez de pular; orçamentos onde só havia `Logf`

**Files:**
- Modify: `internal/service/rnf5000_test.go:85-160`
- Modify: `internal/index/parity_test.go:120-210`
- Modify: `internal/writer/diff_test.go:71`, `internal/search/persist_test.go:281`, `internal/index/movenote_semvault_test.go:77`
- Modify: `scripts/gen_vault.ps1` (só se o Step 1 exigir termos no corpus — ver)

**Interfaces:**
- Consumes: os tetos de RNF-01 e RNF-07 em `docs/PRD.md` (ler os números lá; **não** inventar).

- [ ] **Step 1: `rnf5000_test.go` — cada consulta tem guarda e teto**

Mecanismo: 3 das 8 consultas (`servidor mcp`, `"algoritmo BM25 com pesos"`, `comportamento do watcher`) casam **zero** documentos no `gen_vault.ps1` atual (`grep` = 0), e o teste só faz `t.Logf` — nem teto, nem guarda de corpus. `bench_test.go:173-178` registra que essa mesma troca de corpus quebrou o benchmark em 2026-09-01; o teste RNF não pôde reportar porque não afirma nada.

Conserto:

1. Guarda de corpus, no início: para cada consulta da tabela, rodar e exigir `len(res.Results) > 0`; se zero, `t.Fatalf("a consulta %q nao casa nada no corpus %s; o corpus mudou ou a consulta esta errada", q, caminho)`. **Fatal, não Skip** — um corpus que não serve é defeito de ambiente que o gate deve ver.
2. Para as três consultas sem hits: substituir por três que casam (conferir com `grep -c` no cofre gerado por `gen_vault.ps1 -Notes 5000 -Seed 42`); registrar no comentário da tabela o `grep -c` de cada uma.
3. Teto: `if elapsed > teto { t.Fatalf(...) }` com `teto` = o RNF-01 do `PRD.md` × 2 (folga de máquina), atrás de `//go:build !race` **ou** do mesmo guard que `search`/`service` já usam para tetos (ler `internal/service/latencia_*_test.go` e copiar o padrão). Com `-race` o teste roda a guarda de corpus e pula só o teto — não o teste.
4. RNF-07 (`:91-96`): mesmo tratamento.

- [ ] **Step 2: `parity_test.go:138-205` — iterar `got`, e falhar**

Mecanismo: as asserções iteram `want` (a referência do plugin); uma referência com listas vazias corre zero comparações e reporta paridade. E o teste **pula** quando o corpus de paridade falta, logo `verify.ps1` fica verde sem paridade.

Conserto:

1. Cada bloco de comparação vira comparação de **conjuntos**: `slices.Sort` em `got` e em `want`, `if !slices.Equal(got, want) { t.Errorf("%s: got %v want %v", campo, got, want) }`. Isso cobre os dois sentidos.
2. Guarda de referência vazia: `if len(want.Links)+len(want.Tags)+len(want.Headings) == 0 { t.Fatalf("referencia %s vazia; o dump nao rodou", nome) }` (adaptar aos campos reais).
3. Skip → Fatal quando `testdata/parity/` não existe? **Não** — o corpus é gerado por um plugin de dev do Obsidian e pode não estar em toda máquina. Regra: o teste pula **só** se o diretório não existe, com `t.Skipf` cujo texto diz o caminho; se existe e está incompleto, falha. E `scripts/verify.ps1` ganha uma etapa que **lista** os testes pulados (`go test -v … | grep -c SKIP`) e imprime `[!] N testes pulados` — não falha, mas aparece.

- [ ] **Step 3: Orçamentos onde só havia `Logf`**

- `writer/diff_test.go:71`: `AllocsPerRun` sem orçamento. Medir 7 vezes nesta máquina, anotar o máximo, orçamento = máximo × 1,5 arredondado para cima, `if allocs > orcamento { t.Fatalf(...) }`, com o número medido no comentário e a data.
- `search/persist_test.go:281`, `index/movenote_semvault_test.go:77`: ler o que cada `Logf` imprime; se é uma métrica com RNF, mesmo tratamento; se é diagnóstico sem regra, apagar o `Logf` (ruído no `-v`).

- [ ] **Step 4: Gate e commit**

Run: `pwsh -File scripts/verify.ps1` — Expected: verde, e a etapa nova imprime `[!] N testes pulados`.

```bash
git add internal/service/rnf5000_test.go internal/index/parity_test.go internal/writer/diff_test.go internal/search/persist_test.go internal/index/movenote_semvault_test.go scripts/verify.ps1
git commit -m "test: rnf5000 asserts its corpus and ceilings, parity compares both directions, budgets replace Logf"
```

#### Verificações

- `grep -c` de cada consulta de `rnf5000_test.go` no cofre gerado, colado (8 números, todos > 0).
- Prova da guarda: trocar temporariamente uma consulta por `"xyzzy-inexistente"`, rodar, FAIL nomeando a consulta, restaurar. Colar.
- Prova da paridade: editar temporariamente um `.json` de referência apagando um link, rodar `TestParity…`, FAIL; restaurar (é `testdata/`, `git diff` confere). Colar.
- Os 7 valores medidos de `AllocsPerRun` e o orçamento derivado, no relatório e no comentário do teste.
- `verify.ps1` verde com a contagem de SKIP visível.

#### Regras de execução

- Nunca `git checkout/restore/stash/clean/reset`.
- Nenhum número no teste que não tenha sido medido nesta máquina nesta tarefa; a data e a máquina no comentário.
- Teto de latência atrás do mesmo mecanismo que `search`/`service` já usam (copiar, não inventar).
- Não rodar `test_orphans.ps1` ao mesmo tempo que as medições.

#### Comando de mutação

Esta tarefa não tem prova de mutação por `mutate.ps1`: as regras provadas são guardas de teste, e as provas são as três edições temporárias das Verificações (consulta inexistente, referência mutilada, e — para o orçamento — `orcamento = 0` deve falhar).

#### Contrato de relatório

`task-163-report.md`: status, SHA, os `grep -c`, as três saídas de FAIL, as sete medições, última linha do `verify.ps1` e a linha `[!] N testes pulados`.

### Task 164: Sleep como única sincronização nos testes do watcher; mtime em `delete_test`

**Files:**
- Modify: `internal/watcher/counters_test.go:206-227,:265`
- Modify: `internal/watcher/debounce_test.go:43-46`
- Modify: `internal/watcher/burst_test.go:39`
- Modify: `internal/watcher/rename_test.go:430-437`
- Modify: `internal/service/delete_test.go:148`

**Interfaces:**
- Consumes: o mecanismo de espera que `overflow_test.go` usa (o arquivo registra em `:140-145` por que escrever em `fsWatcher.Errors` foi removido — DATA RACE em kqueue). Ler antes de tocar `counters_test.go`.

- [ ] **Step 1: Inventariar o que cada sleep espera**

Para cada sítio, escrever no relatório: "espera por X; o sinal observável de X é Y". Exemplos de Y que existem no pacote: contador exposto pelo watcher (`Stats()`), canal de eventos aplicados, `idx.Get` do caminho. Se não houver sinal observável, o sleep fica **e o comentário diz por quê** — mas o sleep passa a ser um `esperarAte(t, cond, vaulttest.Prazo)` com polling de 10 ms, não um valor fixo.

`esperarAte` — se o pacote já tem um (procurar `waitFor`, `eventually`, `esperar`), usar; senão criar em `internal/watcher/espera_test.go`:

```go
// esperarAte faz polling ate cond ser verdadeira ou o prazo estourar. Um
// sleep fixo e a pior das duas coisas: lento quando a condicao ja vale, e
// falso quando a maquina esta carregada.
func esperarAte(t *testing.T, cond func() bool, prazo time.Duration) {
	t.Helper()
	fim := time.Now().Add(prazo)
	for time.Now().Before(fim) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("condicao nao valeu em %v", prazo)
}
```

- [ ] **Step 2: `counters_test.go:206-227` — não escrever em `fsWatcher.Errors`**

O teste escreve em `fsWatcher.Errors` para simular erro — `overflow_test.go:140-145` registra que isso é DATA RACE em kqueue e foi removido lá. Trocar pelo mecanismo que `overflow_test.go` usa hoje (ler). Se não houver como injetar erro sem tocar o canal, apagar as asserções de erro e registrar no relatório.

- [ ] **Step 3: `rename_test.go:430-437` — espera de arranque**

Único teste do pacote sem espera de arranque do watcher (os outros usam `esperarPronto` ou equivalente — ler um vizinho e copiar). Acrescentar.

- [ ] **Step 4: `delete_test.go:148` — mtime**

Compara mtime antes/depois sem sleep; em NTFS com resolução de 100 ns pode até funcionar, mas o teste é inerte se o valor for igual. Trocar por `os.Chtimes` explícito com `-2 s` antes da operação e afirmar `depois.After(antes)`; ou, se o que se quer provar é "o arquivo foi reescrito", afirmar sobre o conteúdo. Ler o teste e escolher; dizer qual no relatório.

- [ ] **Step 5: Rodar 20 vezes sob `-race`**

Run: `go test -race -count=20 ./internal/watcher/ ./internal/service/ -run 'TestCounters|TestDebounce|TestBurst|TestRename|TestDelete' 2>&1 | tail -5` — Expected: `ok` 20 vezes, sem `DATA RACE`.

- [ ] **Step 6: Gate e commit**

Run: `pwsh -File scripts/verify.ps1` — Expected: verde.

```bash
git add internal/watcher/counters_test.go internal/watcher/debounce_test.go internal/watcher/burst_test.go internal/watcher/rename_test.go internal/watcher/espera_test.go internal/service/delete_test.go
git commit -m "test(watcher): wait on observable signals instead of fixed sleeps; explicit mtime in delete test"
```

#### Verificações

- Tabela no relatório: sítio, "espera por", "sinal observável", ação.
- `grep -n "time.Sleep" internal/watcher/*_test.go` — cada ocorrência restante tem comentário na linha de cima dizendo por que não há sinal observável.
- `-count=20 -race` colado.
- `verify.ps1` verde.

#### Regras de execução

- Nunca `git checkout/restore/stash/clean/reset`.
- Nenhuma escrita em `fsWatcher.Errors` ou outro canal interno do fsnotify.
- Nenhuma alteração em `internal/watcher/*.go` de produto — se um sinal observável precisar existir e não existe, `BLOCKED` com o sinal proposto; a próxima sessão decide.

#### Comando de mutação

Esta tarefa não tem prova de mutação por `mutate.ps1`: ela troca sincronização de teste, não regra de produto. A prova é o `-count=20 -race` verde e o inventário de sinais.

#### Contrato de relatório

`task-164-report.md`: status, SHA, a tabela de sinais, a saída do `-count=20`, última linha do `verify.ps1`.

### Task 165: Buracos de contrapeso — limite 50 aceito, ponte host→daemon, trava do kernel, `AliasCollisions`

**Files:**
- Modify: `internal/mcpsrv/tools_read_test.go:324` (acrescentar o par: 50 aceito)
- Modify: `cmd/gobsidian/ponte_test.go:195` (host→daemon exercitado; half-close M8)
- Create: `internal/daemon/trava_kernel_windows_test.go`, `internal/daemon/trava_kernel_other_test.go`
- Modify: `internal/daemon/*_test.go` (cobrir `EscutarComLock`, `TravaEmUso`)
- Modify: `internal/index/*_test.go` (cobrir `AliasCollisions`)
- Modify: `internal/ipc/*_test.go` (handshake com `ReadOnly=true`)

**Interfaces:**
- Consumes: `vaulttest.Prazo` (Task 159); `daemon.EhArquivoDeTrava` (Task 153) não é usado aqui; o offset `1<<62` de `internal/daemon/trava_windows.go:19-21`, lido por `internal/doctor/daemon.go:219`.

Cada item é um teste que **afirma o lado aceito** de uma regra que hoje só tem o lado recusado, ou que exercita um caminho que nenhum teste percorre. O critério de pronto de cada um é a mutação nomeada.

- [ ] **Step 1: `tools_read_test.go:324` — 50 aceito**

Hoje: 51 caminhos recusados. Mutação `>` → `>=` no produto sobrevive. Acrescentar o subteste "50 aceito" ao lado:

```go
	t.Run("cinquenta caminhos sao aceitos", func(t *testing.T) {
		paths := make([]string, 50)
		for i := range paths {
			paths[i] = fmt.Sprintf("n%02d.md", i)
		}
		res := chamar(t, sess, "note_metadata", map[string]any{"paths": paths})
		if res.IsError {
			t.Fatalf("50 caminhos e o limite, nao acima dele: %s", textoDoErro(res))
		}
	})
```

(Adaptar `chamar`/`textoDoErro`/o nome da tool ao que `:324` usa.)

Prova: `mutate.ps1 -Path <arquivo do limite> -Anchor "len(req.Paths) > 50" -Replacement "len(req.Paths) >= 50" -Test <nome do teste> -Package ./internal/mcpsrv/` — exit 0.

- [ ] **Step 2: `ponte_test.go:195` — host→daemon**

Hoje: `stdinHost, _ := io.Pipe()` descarta o writer, logo nada flui do host para o daemon; apagar `io.Copy(conn, teed)` (`ponte.go:177-180`) e o half-close M8 (`:198-215`) passa.

Conserto: guardar o writer, escrever um `initialize` JSON-RPC válido nele depois de a ponte estar de pé, e afirmar que o daemon **recebeu** (ler do lado do daemon — o fake que o teste monta — até ver o `"method":"initialize"`, com `vaulttest.Prazo`). Depois fechar o writer e afirmar que o daemon vê EOF dentro do prazo (o half-close M8).

Prova: duas mutações manuais — comentar `io.Copy(conn, teed)`; comentar o half-close — cada uma deve fazer o teste falhar. Colar as duas.

- [ ] **Step 3: `daemon` — trava do kernel entre processos**

Hoje `ComLockDeEscuta` só é testado in-process; trocar `LockFileEx` por `sync.Mutex` passaria. E o offset `1<<62` (`trava_windows.go:19-21`), que `doctor/daemon.go:219` **lê**, não tem teste que o prenda.

`internal/daemon/trava_kernel_windows_test.go` (`package daemon_test`, `//go:build windows`):

1. Criar o arquivo de trava, chamar `daemon.EscutarComLock` (ou a função pública que adquire — ler `trava.go`) no processo de teste.
2. Lançar `os.Executable()` com `-test.run=TestAjudanteTravaEmUso` e env `GOBSIDIAN_TRAVA_AJUDANTE=<caminho>` (padrão de processo-ajudante do `os/exec` da stdlib; o ajudante retorna imediatamente se a env não está setada).
3. O ajudante tenta `daemon.TravaEmUso(caminho)` e sai com código 3 se a trava está em uso, 4 se não.
4. O teste afirma exit 3. Depois libera a trava e relança: exit 4.
5. Offset: o ajudante também tenta `LockFileEx` no offset `1<<62` diretamente (`windows.LockFileEx` com `OverlappedOffset` = os 64 bits do offset) e sai com 5 se **conseguiu** — o que prova que o produto **não** usa esse offset; esperado: falha (o produto está segurando). Se o produto mudar o offset, este teste é o que avisa o `doctor`.

`trava_kernel_other_test.go`: mesmo desenho com `flock` (`unix.Flock`), sem o offset.

Prova: `mutate.ps1 -Path internal/daemon/trava_windows.go -Anchor "1 << 62" -Replacement "1 << 61" -Test TestTravaDoKernelEntreProcessos -Package ./internal/daemon/` — exit 0.

- [ ] **Step 4: `AliasCollisions`**

O campo que substituiu o `0` literal da armadilha está a 0 % de cobertura. Teste em `internal/index/`: duas notas com o mesmo alias no frontmatter; `Build`; afirmar `stats.AliasCollisions == 1` (ou o valor que a semântica de `chave.go` definir — ler e dizer). Mutação: trocar o incremento por nada — exit 0.

- [ ] **Step 5: `ipc` handshake com `ReadOnly=true`**

Toda ponte de teste passa `ReadOnly=false`. Acrescentar um teste que faz o handshake com `ReadOnly=true` e afirma que o servidor do outro lado o recebe assim (ler `internal/ipc/handshake.go` para ver como o campo viaja e onde é observável). Mutação: forçar `ReadOnly=false` no encode — exit 0.

- [ ] **Step 6: Gate, órfãos e commit**

Run: `pwsh -File scripts/verify.ps1` — Expected: verde.
Run: `pwsh -File scripts/test_orphans.ps1` — Expected: os quatro cenários `[OK]` (a ponte foi tocada).

```bash
git add internal/mcpsrv/tools_read_test.go cmd/gobsidian/ponte_test.go internal/daemon/trava_kernel_windows_test.go internal/daemon/trava_kernel_other_test.go internal/daemon/*_test.go internal/index/*_test.go internal/ipc/*_test.go
git commit -m "test: counterweights for the accepted limit, the host-to-daemon bridge, the kernel lock offset, alias collisions and read-only handshake"
```

(Antes do `git add` com glob: `git status --short internal/daemon/ internal/index/ internal/ipc/` e conferir que **só** arquivos desta tarefa aparecem; se houver outro, adicionar por nome.)

#### Verificações

- Cinco saídas de `mutate.ps1` (Steps 1, 3, 4, 5 e — para o Step 2 — as duas mutações manuais) coladas, todas exit 0 / FAIL.
- `go test -cover ./internal/daemon/ ./internal/index/ ./internal/ipc/` antes e depois, colado: `EscutarComLock`, `TravaEmUso`, `AliasCollisions` deixam de estar a 0 % (usar `go tool cover -func` e colar as três linhas).
- `test_orphans.ps1` com os quatro `[OK]`.
- `verify.ps1` verde.

#### Regras de execução

- Nunca `git checkout/restore/stash/clean/reset`.
- Processo-ajudante: matar só o PID que o teste lançou, nunca por nome; `t.Cleanup` com `cmd.Process.Kill()` se ainda vivo.
- Nenhum `net.Listen` fora de `ipc`; o teste do daemon usa `ipc.Listen`.
- Nenhuma alteração de produto. Se um contrapeso revelar defeito, relatório "defeito encontrado — fora de escopo".
- Não rodar `test_orphans.ps1` em paralelo com outra coisa.

#### Comando de mutação

`pwsh -File scripts/mutate.ps1 -Path internal/daemon/trava_windows.go -Anchor "1 << 62" -Replacement "1 << 61" -Test TestTravaDoKernelEntreProcessos -Package ./internal/daemon/` — exit 0. Mais os de Steps 1, 4 e 5, colados como rodaram.

#### Contrato de relatório

`task-165-report.md`: status, SHA, as cinco provas, as três linhas de `cover -func`, saída de `test_orphans.ps1`, última linha do `verify.ps1`.


---

## Fase 3 — Código morto, testes fora do produto, defaults duplicados, simplificações e otimizações medidas (Tasks 166–170)

Nada nesta fase muda contrato. Cada tarefa é estrutural ou é uma otimização com
`benchstat` — nunca as duas no mesmo commit (`golang-refactoring`: move e otimização
em PRs separados). A rede de testes da Fase 2 é o que torna isto seguro.

Modelo por tarefa: 166 Sonnet (deleção guiada por `gopls references`, muitos
arquivos); 167 Haiku (mover três funções para `export_test.go`); 168 Sonnet;
169 Sonnet com revisor Opus (tocam caminho quente e precisam de `benchstat` ~);
170 Opus (otimização medida, decide o que entra).

### Task 166: Código morto — apagar o que ninguém chama, e o doc que promete o que ninguém emite

**Files:**
- Modify: `internal/console/console.go:91` (`Stream.Writer`)
- Modify: `internal/service/service.go:102` (`Inverted()`)
- Modify: `internal/vault/eol.go:71,:79` (`AddBOM`, `NormalizeEOL` — **não** confundir com `writer/section.go:54 NormalizeEOL`, que é viva)
- Modify: `internal/writer/atomic.go:34` (`CleanStaleTempFiles`) e o comentário em `cmd/gobsidian/servico.go:103` que a cita
- Modify: `internal/mcpsrv/server.go:29,:42` (`cfg` escrito e nunca lido)
- Modify: `internal/service/errors.go:36` (`CodePathTooLong`) + `docs/TOOLS.md:503` (5.6)
- Modify: `internal/index/persist_codec.go:639,:682` (guardas `l.uvarint(math.MaxUint64, …)` que nunca disparam)
- Modify: `internal/search/inverted.go:25,:275,:298` (`Posting.Frequency` — só leitores em teste; **não** está no formato de cache)
- Modify: `internal/doctor/doctor.go:61-67` (`halting` constante)
- Modify: `internal/lifecycle/lifecycle.go:28-69` (`Options.ParentCheckInterval` nunca definido em produção — ver Step 2)
- Modify: `internal/doctor/checks.go:277-278` (contadores escritos em toda plataforma, lidos só em `checks_windows.go` — ver Step 2)
- Modify: `internal/service/write.go:423-426` (`vault.Resolve` com três resultados descartados → `if req.To == ""`)
- Modify: os `_test.go` que leem `Posting.Frequency` (encontrar com `gopls references`)

**Interfaces:** nenhuma nova. `writer.CleanStaleTempFiles` some antes da Task 171 mover o resto do arquivo.

- [ ] **Step 1: Para cada símbolo, `gopls references` antes de apagar**

Regra: apagar só o que tem **zero** referências fora da própria declaração, contando `_test.go`. Se um teste referencia, o teste é apagado junto **só se** ele testa exclusivamente o símbolo morto (ler o teste). Registrar no relatório a contagem de referências por símbolo (`gopls references` ou `grep -rn`).

Para `Posting.Frequency`: confirmar com `grep -n Frequency internal/search/persist_codec.go internal/search/soa.go` vazio (não está no formato — apagar o campo não muda o cache). Apagar o campo, os `:275,:298` que o escrevem, e as leituras em teste.

- [ ] **Step 2: Dois casos que não são "apagar"**

`lifecycle.Options.ParentCheckInterval`: nunca definido em produção → o padrão é a única conta. **Não apagar o campo** se um teste o usa para encurtar a vigília (`grep -rn ParentCheckInterval --include=*_test.go`); se nenhum teste o usa, apagar o campo e deixar a constante. Dizer qual foi no relatório.

`doctor/checks.go:277-278`: contadores escritos em toda plataforma e lidos só em `checks_windows.go`. Mover a escrita para o arquivo `_windows.go` que os lê (código de plataforma atrás de build tag). Se isso exigir um ponto de extensão, `BLOCKED` com a proposta — não `if runtime.GOOS`.

- [ ] **Step 3: `write.go:423-426`**

Hoje: `_, _, err := vault.Resolve(root, req.To)` (ou equivalente) só para rejeitar `to: ""`. Trocar por:

```go
	if req.To == "" {
		return MoveNoteResult{}, Errorf(CodeInvalidArgument, "to e obrigatorio")
	}
```

e conferir que a resolução real de `req.To` acontece mais adiante (ela acontece — é o que torna a chamada de `:423` redundante; se **não** acontecer, a chamada não era morta: `BLOCKED`).

Prova: `mutate.ps1 -Path internal/service/write.go -Anchor 'if req.To == ""' -Replacement 'if false && req.To == ""' -Test <teste que já cobre to vazio; se não existir, criar TestMoveNoteSemDestinoEInvalidArgument> -Package ./internal/service/` — exit 0.

- [ ] **Step 4: `TOOLS.md:503` — `PATH_TOO_LONG`**

Retirar a linha da tabela de erros. Se `check_doc_refs.ps1` reclamar de referência solta, é sinal de que outro doc cita — retirar lá também.

- [ ] **Step 5: Gate e commit**

Run: `pwsh -File scripts/verify.ps1` — Expected: verde (o `golangci-lint` com `unused` é quem confirma que nada ficou órfão).

```bash
git add internal/console/console.go internal/service/service.go internal/vault/eol.go internal/writer/atomic.go cmd/gobsidian/servico.go internal/mcpsrv/server.go internal/service/errors.go docs/TOOLS.md internal/index/persist_codec.go internal/search/inverted.go internal/doctor/doctor.go internal/lifecycle/lifecycle.go internal/doctor/checks.go internal/doctor/checks_windows.go internal/service/write.go <testes tocados>
git commit -m "refactor: delete dead code and the error code no path produces"
```

#### Verificações

- Tabela no relatório: símbolo, referências antes (número), ação (apagado / mantido porque …).
- `go build ./... && go vet ./...` nos três GOOS (o `verify.ps1` faz) — `checks_windows.go` compila.
- `mutate.ps1` do Step 3, exit 0, colado.
- `git diff --stat`: só remoção líquida fora de `write.go` e `checks_windows.go`.
- `verify.ps1` verde.

#### Regras de execução

- Nunca `git checkout/restore/stash/clean/reset`.
- Nunca `if runtime.GOOS ==`.
- Um símbolo com referência viva **não** é apagado nesta tarefa — vai para o relatório como "não morto: <quem chama>".
- Sem alteração de comportamento observável além de `to: ""` responder `INVALID_ARGUMENT` (que já respondia, por outro caminho).

#### Comando de mutação

`pwsh -File scripts/mutate.ps1 -Path internal/service/write.go -Anchor 'if req.To == ""' -Replacement 'if false && req.To == ""' -Test TestMoveNoteSemDestinoEInvalidArgument -Package ./internal/service/` — exit 0.

#### Contrato de relatório

`task-166-report.md`: status, SHA, a tabela de símbolos, a saída do `mutate.ps1`, `git diff --stat`, última linha do `verify.ps1`.

### Task 167: `*ForTest` sai do produto para `export_test.go`

**Files:**
- Modify: `internal/index/persist.go:86` (`WriteIndexCacheForTest` → apagar)
- Create: `internal/index/export_test.go`
- Modify: `internal/search/persist.go:46` (`WriteCacheForTest` → apagar)
- Create: `internal/search/export_test.go`
- Modify: `internal/mcpsrv/server.go:123` (`RegisterPanicProbeForTest` → apagar)
- Create: `internal/mcpsrv/export_test.go`

**Interfaces:**
- Consumes: os únicos chamadores são `internal/index/persist_test.go` (`package index_test`), `internal/search/persist_test.go` (`package search_test`), `internal/mcpsrv/server_test.go` (`package mcpsrv_test`) — conferido em 2026-09-02 com `grep -rln`. Um `export_test.go` em `package index` exporta o símbolo **só para os testes do mesmo diretório**, que é exatamente o caso.

- [ ] **Step 1: Um `export_test.go` por pacote**

`internal/index/export_test.go`:

```go
package index

import "io"

// WriteIndexCacheForTest expoe o codificador do cache aos testes externos do
// pacote. Vive aqui, e nao em persist.go, porque um simbolo que so teste chama
// nao pertence ao binario do produto.
func WriteIndexCacheForTest(w io.Writer, h CacheHeader, notes []*Note, assets []*Asset) error {
	return escreveIndexCache(w, h, notes, assets)
}
```

`internal/search/export_test.go` — mesmo desenho sobre `escreveCache` (copiar a assinatura de `persist.go:46`).

`internal/mcpsrv/export_test.go` — mover o corpo de `RegisterPanicProbeForTest` de `server.go:123` para cá, sem alterar.

- [ ] **Step 2: Apagar as três do produto e rodar**

Run: `go test -race ./internal/index/ ./internal/search/ ./internal/mcpsrv/ 2>&1 | tail -5` — Expected: `ok` nos três.
Run: `go build ./... && grep -rn "ForTest" --include=*.go internal cmd | grep -v _test.go` — Expected: vazio.

- [ ] **Step 3: Gate e commit**

Run: `pwsh -File scripts/verify.ps1` — Expected: verde.

```bash
git add internal/index/persist.go internal/index/export_test.go internal/search/persist.go internal/search/export_test.go internal/mcpsrv/server.go internal/mcpsrv/export_test.go
git commit -m "refactor: move test-only exports out of the product into export_test.go"
```

#### Verificações

- `grep -rn "ForTest" --include=*.go internal cmd | grep -v _test.go` vazio, colado.
- `go test -race` dos três pacotes, colado.
- `verify.ps1` verde.

#### Regras de execução

- Nunca `git checkout/restore/stash/clean/reset`.
- Não renomear os símbolos: os testes que os chamam não mudam.
- Se `gopls references` mostrar chamador **fora** do diretório do pacote, `BLOCKED` — `export_test.go` não alcança.

#### Comando de mutação

Esta tarefa não tem prova de mutação: é um move sem regra. A prova é o `grep` vazio e a suíte verde.

#### Contrato de relatório

`task-167-report.md`: status, SHA, o `grep`, a saída dos testes, última linha do `verify.ps1`.

### Task 168: `mcpsrv` para de aplicar defaults que o `service` já aplica

**Files:**
- Modify: `internal/mcpsrv/tools_read.go:27-38,:185-209,:257-265`
- Modify: `internal/mcpsrv/tools_read_test.go` (o teste que prova que o default vem do serviço)

**Interfaces:**
- Consumes: `service.ComTeto(int) int` (`errors.go:147`, padrão `LimitePadrao = 100`, teto 500), `service.ValidarEnum` (`errors.go:166`, vazio → padrão), `service.Search` (`search.go:146-163`: `Limit <= 0 → 20`, `> 200 → 200`, `SnippetChars <= 0 → search.DefaultSnippetChars`), `service.LinkGraph` (`graph.go:101-103`), `service.ListNotes`/`NoteList` (`graph.go:462-470`).

Mecanismo: `tools_read.go` aplica `limit := 20`, `snippetChars := 240`, `offset := 0`, `limit := 100`, `tagMode := "all"`, `sort := "path"`, `order := "asc"`, `depth := 1`, `direction := "both"` antes de chamar o serviço, que aplica os mesmos defaults de novo. Dois lugares com o mesmo número são dois lugares para o número divergir. O `service` é a conta (é ele que a CLI usa); o boundary MCP só desembrulha ponteiros.

- [ ] **Step 1: Tabela dos defaults**

Antes de tocar, montar no relatório: parâmetro, default em `mcpsrv`, default no `service`, default em `TOOLS.md`. Só apagar onde os três coincidem. Onde divergirem: o `TOOLS.md` é o contrato; ajustar o `service` para ele e registrar como "divergência encontrada". Onde o `service` **não** tem default (candidatos: `depth`, `recursive`, `include_broken`, `include_embeds`), o default fica no `mcpsrv` — **não** inventar um no serviço nesta tarefa; anotar.

- [ ] **Step 2: Teste que prova que o default vem do serviço**

Em `tools_read_test.go`, um subteste por tool: chamar `vault_search` sem `limit`/`snippet_chars`, afirmar `effective_limit == 20` e `effective_snippet_chars == 240` na resposta; `note_list` sem `sort`/`order`, afirmar ordem por `path` ascendente; `link_graph` sem `direction`, afirmar que vêm arestas nos dois sentidos (fixture com um link de ida e um de volta).

- [ ] **Step 3: Apagar os blocos**

Os `if in.X != nil { x = *in.X }` viram `x := 0; if in.X != nil { x = *in.X }` (o zero é o que o serviço interpreta como "não informado") — ou, mais simples, passar o ponteiro desreferenciado com `valorOuZero(in.Limit)`:

```go
// valorOuZero desembrulha um parametro opcional. Zero e "nao informado" para
// o service, que aplica o padrao — a UNICA conta de cada padrao.
func valorOuZero(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}
```

Para strings: passar `in.Sort` cru (vazio = padrão em `ValidarEnum`).

Prova de que a conta é uma: `mutate.ps1 -Path internal/service/search.go -Anchor "opts.Limit = 20" -Replacement "opts.Limit = 21" -Test TestVaultSearchDefaultVemDoServico -Package ./internal/mcpsrv/` — exit 0 (antes desta tarefa, o `20` do `mcpsrv` mascarava a mutação).

- [ ] **Step 4: Gate e commit**

Run: `pwsh -File scripts/verify.ps1` — Expected: verde (`check_tool_params` confere schema × código).

```bash
git add internal/mcpsrv/tools_read.go internal/mcpsrv/tools_read_test.go
git commit -m "refactor(mcpsrv): stop re-applying defaults the service owns"
```

#### Verificações

- A tabela do Step 1 no relatório, com uma linha por parâmetro.
- `mutate.ps1` do Step 3, exit 0, colado.
- `wc -l internal/mcpsrv/tools_read.go` antes e depois.
- `verify.ps1` verde.

#### Regras de execução

- Nunca `git checkout/restore/stash/clean/reset`.
- Nenhum default novo no `service` — se falta, fica no `mcpsrv` e vai para o relatório.
- Schema (`jsonschema` tags) não muda nesta tarefa.

#### Comando de mutação

`pwsh -File scripts/mutate.ps1 -Path internal/service/search.go -Anchor "opts.Limit = 20" -Replacement "opts.Limit = 21" -Test TestVaultSearchDefaultVemDoServico -Package ./internal/mcpsrv/` — exit 0.

#### Contrato de relatório

`task-168-report.md`: status, SHA, a tabela, a saída do `mutate.ps1`, `wc -l`, última linha do `verify.ps1`.

### Task 169: Simplificações mecânicas com eficiência igual — cada uma com `benchstat`

**Files:**
- Modify: `internal/service/graph.go:69-75` (`chaveDaAresta` → struct), `:261,:283-294,:396-402` (`TagNode.Children []any` → `[]TagNode`), `:331-366` (`tagListHierarchical` filtra dentro)
- Modify: `internal/service/search.go:192,:251,:393` (`resultadoVazio()`), `:249-269,:392-411` (`pagina()`), `:231` (`make(0, len)`)
- Modify: `internal/service/write.go:561-628` (`MoveNoteResult` de erro 5× → uma função), `:22` `hashDoConteudo` passa a ser usada em `read.go:404`, `graph.go:481,:583`
- Modify: `internal/index/index.go:104-126` (`TotalSize`: `RLock` no hit, upgrade só no miss)
- Modify: `internal/index/resolve.go:410-421` (usar `vivosLocked :249`)
- Modify: `internal/writer/lock.go:14` (`normalizeKey` delega à conta de chave de `index/chave.go` — ver Step 3)

**Interfaces:**
- Consumes: benchmarks da Baseline (`LinkGraphBothDepth2`, `TagListPlano`, `TagListHierarquico`, `SearchLimit200Cache`, `TotalSizeRepetido`) e os binários `antes_*` em `%LOCALAPPDATA%\gobsidian-bench\2026-09-02\`.

Regra da tarefa: cada item é **um commit**; cada commit que toca caminho quente traz `benchstat` com **≥ 7 amostras intercaladas** contra o binário `antes_<pkg>.test.exe` e o veredito é `~` ou melhora. Uma piora com `p < 0.05` reverte o item (novo commit que desfaz, não `git reset`) e vai para o relatório.

- [ ] **Step 1: `chaveDaAresta` → struct**

```go
// chaveDaAresta identifica uma aresta no BFS. Struct, e nao Sprintf: a
// identidade e exata por construcao e nao custa alocacao por aresta.
type chaveDaAresta struct {
	Source, Target, Kind, Alias, Anchor string
}
```

Substituir o `Sprintf` e o `map[string]…` por `map[chaveDaAresta]…`. Bench: `LinkGraphBothDepth2` — esperado: allocs/op cai (41 na baseline); `sec/op` `~` ou melhor.

- [ ] **Step 2: `TagNode.Children []TagNode`**

Trocar `[]any` por `[]TagNode`; remover o type-assert/write-back em `:283-294` e o boxing em `:396-402`. JSON de saída **idêntico** — prova: golden do `tag_list` hierárquico (se não existir, criar `testdata/tag_list_hierarquico.json` a partir do binário `antes`, e o teste compara byte a byte). Bench: `TagListHierarquico` (6,163 ms / 10 880 allocs na baseline).

- [ ] **Step 3: Os demais**

- `tagListHierarchical`: filtrar dentro do loop e usar `ix.tags` como o ramo plano. Bench: `TagListHierarquico`, `TagListPlano`.
- `resultadoVazio()` + `pagina()` em `search.go`; `make([]…, 0, len(rawHits))` em `:231`. Bench: `SearchLimit200Cache`.
- `MoveNoteResult` de erro: `func moveNoteErro(req MoveNoteRequest, err error) (MoveNoteResult, error)`.
- `hashDoConteudo` nos três sítios (`read.go:404`, `graph.go:481,:583`) — atenção: os três formatam `n.Hash` (já `uint64`), e `hashDoConteudo` recebe `[]byte`; criar `func formatarHash(h uint64) string { return fmt.Sprintf("%016x", h) }` e fazer `hashDoConteudo` chamá-la. Uma conta do formato.
- `TotalSize`: `RLock`; se válido, devolve; senão `RUnlock`, `Lock`, **re-conferir** (outro leitor pode ter preenchido), calcular. Bench: `TotalSizeRepetido` (5,610 µs, 0 allocs) — `~`.
- `resolve.go:410-421` → `vivosLocked`.
- `writer/lock.go:14 normalizeKey`: hoje `ToLower` sem NFC. A chave canônica mora em `index/chave.go`, mas `writer` **não** importa `index` (grafo) e não vai importar. Opção que respeita o grafo: `text.ChaveDeCaminho` (se `chave.go` já delega a `text`, usar; se não, `BLOCKED` com a proposta de mover a conta para `text`, que `writer` pode importar — `writer → parser, vault`, e `parser → text`, então `writer → text` é aresta nova **em folha de teste?** Não: `text` é folha, `writer` ganhar `text` é aresta nova no produto e precisa de justificativa no relatório: "uma conta por regra" é a justificativa; o orquestrador decide).

- [ ] **Step 4: `benchstat` final**

Run: `go test -c -o %LOCALAPPDATA%\gobsidian-bench\2026-09-02\depois169_service.test.exe ./internal/service/` (e `index`), depois 7 rodadas intercaladas `antes`/`depois` de `-run '^$' -bench 'LinkGraphBothDepth2|TagList|SearchLimit200Cache' -benchmem -count=1`, concatenar, `benchstat antes.txt depois.txt`. Colar.

- [ ] **Step 5: Gate**

Run: `pwsh -File scripts/verify.ps1` — Expected: verde.

Commits (um por item, mensagens):

```
refactor(service): edge key as a struct, not a formatted string
refactor(service): TagNode.Children is []TagNode, JSON unchanged
refactor(service): filter inside tagListHierarchical and reuse the flat branch
refactor(service): one empty result, one pagination, one move error
refactor(service): one account for the hash format
refactor(index): TotalSize takes the read lock on the memoized hit
refactor(index): resolve reuses vivosLocked
refactor(writer): lock key delegates to the shared key account
```

#### Verificações

- `benchstat` colado com ≥ 7 amostras por lado; nenhuma linha com piora `p < 0.05`.
- Golden do `tag_list` hierárquico byte a byte igual ao do binário `antes`.
- Um commit por item; `git log --oneline <base>..HEAD` colado.
- `verify.ps1` verde.

#### Regras de execução

- Nunca `git checkout/restore/stash/clean/reset`. Reverter = commit novo.
- Não rodar `test_orphans.ps1` nem outra medição durante o `benchstat`.
- Nenhum item muda JSON de saída, ordem de resultados ou código de erro.
- Aresta nova no grafo (`writer → text`) só com a justificativa no relatório e a atualização do bloco do grafo em `CLAUDE.md` no mesmo commit.

#### Comando de mutação

Esta tarefa não tem prova de mutação por `mutate.ps1`: são refatorações que preservam comportamento; a prova é o golden byte a byte e o `benchstat`.

#### Contrato de relatório

`task-169-report.md`: status, SHAs (um por item), `benchstat`, resultado do golden, decisão sobre `normalizeKey`, última linha do `verify.ps1`.

### Task 170: Otimizações medidas — só entra o que o `benchstat` aprova

**Files:**
- Modify (candidatos): `internal/search/inverted.go:214-222` (`removeLocked`: índice reverso doc→termos no delta), `:232,:311,:346` (variante sem `Normalize` para termo já normalizado), `internal/parser/ast.go:336-337` (`bytes.Index`/`bytes.Count`), `internal/writer/linkrewrite.go:52-67` (`bytes.Buffer` com `Grow` único), `internal/service/search.go:443-453` (`EqualFold` + normalização içada)
- Create/Modify: benchmark faltante para `removeLocked` se `update_bench_test.go` não o isolar

**Interfaces:**
- Consumes: Baseline `InvertedUpdateLote` (6,549 s), `RewriteLinksMuitos` (1,496 ms), `ParseNotaLonga` (2,611 ms), `DetectCandidatesNotaLonga` (309,9 µs), `SearchFiltroFrontmatter`, `IndexBuild`, `BuildComHub`; binários `antes_*`.

Regra: **otimização é mudança de comportamento de desempenho, e só entra com número.** Um candidato sem melhora `p < 0.05` **não entra** e vai para o relatório como "sem ganho medido: <benchstat>". Um por commit.

- [ ] **Step 1: `removeLocked` — o único com potencial de ordem de grandeza**

Hoje varre todo termo do delta por remoção: O(termos do delta) por nota removida. `InvertedUpdateLote` na baseline: **6,549 s**. Índice reverso `docTermos map[docID][]termoID` no delta, preenchido no add, consumido no remove. Custo: memória proporcional ao delta (medir `B/op`).

Bench antes/depois de `InvertedUpdateLote`, 7 × 2 intercalados. Se `sec/op` cair com `p < 0.05`: entra, com o `B/op` no commit. Se não: relatório.

- [ ] **Step 2: Os quatro pequenos**

Cada um com o bench da Baseline que o cobre. `bytes.Index` no parser (`ParseNotaLonga`); `bytes.Buffer` no rewrite (`RewriteLinksMuitos`); termo sem re-`Normalize` (`SearchTermoAmploCache`); `EqualFold` no filtro de tags (`SearchFiltroFrontmatter` — **atenção**: a Task 180 vai trocar esse filtro pela chave de tag; se esta tarefa rodar antes, fazer só o içamento da normalização e deixar a comparação para a 180).

- [ ] **Step 3: Gate**

Run: `pwsh -File scripts/verify.ps1` — Expected: verde.

Commits: `perf(search): reverse doc->terms map makes delta removal O(terms of doc)` etc., cada um com o `benchstat` resumido no corpo da mensagem (a linha do bench: antes → depois, p).

#### Verificações

- Um `benchstat` por candidato, ≥ 7 amostras por lado, colado.
- Para cada candidato: "entrou (p=…)" ou "sem ganho medido".
- `go test -race ./internal/search/ ./internal/parser/ ./internal/writer/ ./internal/service/` verde.
- `verify.ps1` verde.

#### Regras de execução

- Nunca `git checkout/restore/stash/clean/reset`.
- Nenhum número no commit ou no relatório que não venha do `benchstat` desta tarefa.
- Não rodar duas medições ao mesmo tempo, nem `test_orphans.ps1`.
- Formato de cache não muda (`CacheFormatVersion` intacto); se a otimização exigir mudar, `BLOCKED`.

#### Comando de mutação

Esta tarefa não tem prova de mutação por `mutate.ps1`: a regra provada é de desempenho e a prova é o `benchstat`. A correção é coberta pela suíte (`TestIndiceRecarregadoEIdenticoAoConstruido` para o delta).

#### Contrato de relatório

`task-170-report.md`: status, SHAs, os `benchstat`, a lista entrou/não entrou, última linha do `verify.ps1`.

---

## Fase 4 — Movimentos entre pacotes (Tasks 171–177)

Todos os movimentos seguem `golang-refactoring`: alias na origem, migração de chamadores, remoção do alias — dois PRs por movimento, nenhum commit com estrutura e comportamento juntos.

Modelo por tarefa: 171 Opus (move cross-package, testes de durabilidade migram); 172 Sonnet com revisor Opus; 173 Sonnet; 174 Opus; 175 Sonnet; 176 Sonnet; 177 Opus.

### Task 171: c1 PR1 — `vault.ReplaceFile`, `vault.WriteAtomic`, `vault.TempFilePrefix`, `vault.SweepStaleTempFiles`; `writer` encaminha

**Files:**
- Create: `internal/vault/atomic.go` (movido de `internal/writer/atomic.go`, menos `CleanStaleTempFiles` já apagada na Task 166)
- Create: `internal/vault/syncdir_unix.go`, `internal/vault/syncdir_windows.go` (movidos)
- Create: `internal/vault/atomic_test.go`, `internal/vault/durabilidade_test.go`, `internal/vault/sweep_profundo_windows_test.go` (movidos de `writer`)
- Modify: `internal/writer/atomic.go` (vira só encaminhadores com alias)
- Modify: `internal/vault/walk.go:74` (literal `".gobsidian-tmp-"` → `TempFilePrefix`)
- Modify: `internal/vault/longpath_windows.go:26` (comentário cita `writer.SweepStaleTempFiles` → `SweepStaleTempFiles`)
- Modify: `docs/ESTRUTURA.md`, `docs/ARCHITECTURE.md` (onde descrevem `writer`/`vault`)

**Interfaces:**
- Produces (pacote `vault`):
  - `const TempFilePrefix = ".gobsidian-tmp-"`
  - `func ReplaceFile(ctx context.Context, targetPath string, escrever func(*os.File) error) error` — temp no mesmo diretório com `TempFilePrefix`, chmod do alvo, chama `escrever(tmp)`, `Sync`, `Close`, rename com retry e `sincronizarDiretorio`. É o corpo atual de `WriteAtomic` com o `tmpFile.Write(data)` substituído por `escrever(tmpFile)`.
  - `func WriteAtomic(ctx context.Context, targetPath string, data []byte) error` — `ReplaceFile(ctx, targetPath, func(f *os.File) error { _, err := f.Write(data); return err })`.
  - `type SweepResult` (movido), `func SweepStaleTempFiles(ctx context.Context, root string) (SweepResult, error)`.
- `writer` durante PR1: `const TempFilePrefix = vault.TempFilePrefix`; `type SweepResult = vault.SweepResult`; `func WriteAtomic(ctx, p, data) error { return vault.WriteAtomic(ctx, p, data) }`; `func SweepStaleTempFiles(ctx, root) (SweepResult, error) { return vault.SweepStaleTempFiles(ctx, root) }` — cada um com `// Deprecated: use vault.X. Removido na Task 172.`

**Por que `ReplaceFile` com callback:** os dois caches (`index/persist.go:124-146`, `search/persist.go:87-116`) codificam **em streaming** para o temporário (`escreveIndexCache(w io.Writer, …)`), não têm `[]byte`. Sem o callback, a Task 172 teria de materializar 25 MiB (`SaveInvertedCacheReal`: 24,95 MiB B/op) para chamar `WriteAtomic`. Com ele, `WriteAtomic` é o caso particular.

**Ruling do orquestrador (registrado no ledger):** os caches passam a ter `Sync()` como as notas — uma conta, um comportamento. Custo medido na Baseline: `SaveIndexCacheReal` 21,76 → 41,87 ms com fsync; `SaveInvertedCacheReal` 227,3 → 259,1 ms. Fora do caminho de consulta; acontece uma vez por construção. Se errado, custa 20 + 32 ms por gravação de cache num cofre de 5 000 notas.

- [ ] **Step 1: Mover com `git mv` e ajustar o pacote**

```bash
git mv internal/writer/atomic.go internal/vault/atomic.go
git mv internal/writer/syncdir_unix.go internal/vault/syncdir_unix.go
git mv internal/writer/syncdir_windows.go internal/vault/syncdir_windows.go
git mv internal/writer/atomic_test.go internal/vault/atomic_test.go
git mv internal/writer/durabilidade_test.go internal/vault/durabilidade_test.go
git mv internal/writer/sweep_profundo_windows_test.go internal/vault/sweep_profundo_windows_test.go
```

Trocar `package writer` → `package vault` (e `writer_test` → `vault_test`) nos seis. Conferir que nenhum dos testes movidos usa símbolo de `writer` que ficou (`PathLocker`, `NormalizeEOL` de `section.go`) — se usar, o teste não migra inteiro: separar o subteste que depende de `writer` num arquivo que fica.

- [ ] **Step 2: Extrair `ReplaceFile`**

Em `internal/vault/atomic.go`, renomear `WriteAtomic` → `ReplaceFile` com o parâmetro `escrever func(*os.File) error`, substituir `tmpFile.Write(data)` por:

```go
	if err := escrever(tmpFile); err != nil {
		return fmt.Errorf("escrevendo no temporario %q: %w", tmpName, err)
	}
```

e criar `WriteAtomic` como o wrapper de quatro linhas. Docstring de `ReplaceFile` explica o callback (os caches em streaming) e mantém os cinco passos numerados que a docstring atual tem.

- [ ] **Step 3: Encaminhadores em `writer`**

Novo `internal/writer/atomic.go` só com os quatro encaminhadores acima e o `// Deprecated`. `writer` já importa `vault` — nenhuma aresta nova.

- [ ] **Step 4: `walk.go:74` e o comentário**

`strings.HasPrefix(name, ".gobsidian-tmp-")` → `strings.HasPrefix(name, TempFilePrefix)`.

Teste que prende a conta, em `internal/vault/walk_test.go`:

```go
// O arquivo e criado com o LITERAL, nao com a constante: e o que o disco tem
// de um binario antigo. Se a constante do filtro mudar, este teste e o que
// avisa que o lixo antigo passaria a entrar no indice.
func TestWalkIgnoraTemporarioDeBinarioAntigo(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".gobsidian-tmp-abc123"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "a.md"), []byte("# a
"), 0o644); err != nil {
		t.Fatal(err)
	}
	var vistos []string
	err := vault.Walk(context.Background(), root, func(rel string, d fs.DirEntry) error {
		vistos = append(vistos, rel)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(vistos) != 1 || vistos[0] != "a.md" {
		t.Fatalf("o temporario entrou no walk: %v", vistos)
	}
}
```

(Adaptar a assinatura de `vault.Walk` à real — ler `walk.go`.) Com a mutação da constante (`tmp` → `tmq`), o filtro deixa de casar o literal e o teste falha.

- [ ] **Step 5: Teste de `ReplaceFile` com callback que falha**

Em `internal/vault/atomic_test.go`:

```go
func TestReplaceFileCallbackFalhaNaoTocaOAlvo(t *testing.T) {
	alvo := filepath.Join(t.TempDir(), "a.md")
	if err := os.WriteFile(alvo, []byte("antes"), 0o644); err != nil {
		t.Fatal(err)
	}
	quero := errors.New("codec falhou")
	err := vault.ReplaceFile(context.Background(), alvo, func(*os.File) error { return quero })
	if !errors.Is(err, quero) {
		t.Fatalf("err = %v, quero %v embrulhado", err, quero)
	}
	got, _ := os.ReadFile(alvo)
	if string(got) != "antes" {
		t.Fatalf("alvo mudou para %q", got)
	}
	restos, _ := filepath.Glob(filepath.Join(filepath.Dir(alvo), vault.TempFilePrefix+"*"))
	if len(restos) != 0 {
		t.Fatalf("temporario ficou para tras: %v", restos)
	}
}
```

- [ ] **Step 6: Gate, órfãos e commit**

Run: `pwsh -File scripts/verify.ps1` — Expected: verde.
Run: `go test -race -count=3 ./internal/vault/ ./internal/writer/` — Expected: `ok` (os 1 000 ciclos de `atomic_test.go` agora em `vault`).

```bash
git add internal/vault/atomic.go internal/vault/syncdir_unix.go internal/vault/syncdir_windows.go internal/vault/atomic_test.go internal/vault/durabilidade_test.go internal/vault/sweep_profundo_windows_test.go internal/writer/atomic.go internal/vault/walk.go internal/vault/longpath_windows.go internal/vault/walk_test.go docs/ESTRUTURA.md docs/ARCHITECTURE.md
git commit -m "refactor(vault): atomic replace, write and temp sweep move to vault; writer forwards"
```

#### Verificações

- `git diff --stat -M <base>..HEAD` mostra os seis arquivos como **rename** (similaridade alta), não delete+add.
- `grep -rn "writer.WriteAtomic\|writer.SweepStaleTempFiles" --include=*.go internal cmd | grep -v _test | wc -l` = 7 (os seis de `service/write.go` + `servico.go:78`) — inalterado nesta PR.
- `mutate.ps1` do Step 4, exit 0, colado.
- Grafo: `go list -f '{{.Imports}}' ./internal/vault/` **não** ganha import de pacote interno (continua folha).
- `verify.ps1` verde; `-count=3` colado.

#### Regras de execução

- Nunca `git checkout/restore/stash/clean/reset`. `git mv` é permitido (é o que a tarefa pede).
- Nenhum chamador de `service` ou `cmd` muda nesta PR — é o que torna a PR2 revisável sozinha.
- `vault` continua folha: nenhum import de `internal/*`.
- Código de plataforma (`syncdir_*`) permanece atrás de build tag.

#### Comando de mutação

`pwsh -File scripts/mutate.ps1 -Path internal/vault/atomic.go -Anchor 'TempFilePrefix = ".gobsidian-tmp-"' -Replacement 'TempFilePrefix = ".gobsidian-tmq-"' -Test TestWalkIgnoraTemporarioDeBinarioAntigo -Package ./internal/vault/` — exit 0.

#### Contrato de relatório

`task-171-report.md`: status, SHA, `git diff --stat -M`, o `grep` com 7, `go list` de `vault`, `mutate.ps1`, última linha do `verify.ps1`.

### Task 172: c1 PR2 — chamadores migram, caches usam `ReplaceFile`, encaminhadores somem

**Files:**
- Modify: `internal/service/write.go:149,:248,:378,:619,:751,:832` (`writer.WriteAtomic` → `vault.WriteAtomic`)
- Modify: `cmd/gobsidian/servico.go:78` (`writer.SweepStaleTempFiles` → `vault.SweepStaleTempFiles`)
- Modify: `internal/index/persist.go:124-146` (`SaveIndexCache` usa `vault.ReplaceFile`)
- Modify: `internal/search/persist.go:87-116` (`SaveInvertedCache` usa `vault.ReplaceFile`; `promoverArenaSePresente(inv)` **antes** de `ExportForCache`)
- Delete: `internal/writer/atomic.go` (os encaminhadores)
- Modify: testes de `service` que referenciam `writer.TempFilePrefix` (encontrar com `gopls references`)
- Modify: `docs/ESTADO.md` (formato de cache: nota de que a gravação agora tem `fsync`)

**Interfaces:**
- Consumes: `vault.ReplaceFile`, `vault.WriteAtomic`, `vault.SweepStaleTempFiles`, `vault.TempFilePrefix` (Task 171).
- Consumes: `TestSaveOverwritesMappedCache` (`internal/search/persist_test.go:65`) — é o teste que prende a ordem `promover → rename`.

- [ ] **Step 1: `gopls rename`? Não — é troca de pacote.** Substituir por `sed` guiado pela lista de **Files** e confirmar com `go build ./...`. Import de `writer` em `service/write.go` permanece (usa `PathLocker`, seções); em `servico.go`, se `writer` ficar sem uso, remover o import.

- [ ] **Step 2: `SaveIndexCache` com `ReplaceFile`**

```go
	finalPath := filepath.Join(cacheDir, indexCacheFileName)
	if err := vault.ReplaceFile(ctx, finalPath, func(f *os.File) error {
		return escreveIndexCache(f, header, notes, assets)
	}); err != nil {
		return fmt.Errorf("gravando cache de indice em %q: %w", finalPath, err)
	}
	return nil
```

`index` já importa `vault` — nenhuma aresta nova.

- [ ] **Step 3: `SaveInvertedCache` com `ReplaceFile`**

A promoção da arena precisa vir **antes** do rename; com `ReplaceFile` o rename está dentro, então a promoção vem antes da chamada inteira — e antes do `ExportForCache`, para que os slices exportados já apontem para o heap:

```go
	// Promove a arena ANTES de exportar e gravar: ReplaceFile faz o rename
	// por dentro, e o rename falha no Windows enquanto o alvo esta mapeado
	// (ver promoverArenaSePresente, em mmap.go). Exportar depois de promover
	// garante que os slices gravados nao apontam para o mapeamento fechado.
	promoverArenaSePresente(inv)
	termos, docLengths := inv.ExportForCache()

	finalPath := filepath.Join(cacheDir, "inverted_cache.gob")
	if err := vault.ReplaceFile(ctx, finalPath, func(f *os.File) error {
		return escreveCache(f, header, termos, docLengths)
	}); err != nil {
		return fmt.Errorf("gravando cache de busca em %q: %w", finalPath, err)
	}
	return nil
```

Run: `go test -race ./internal/search/ -run 'TestSaveOverwritesMappedCache|TestSaveAndLoadInvertedCache|TestIndiceRecarregadoEIdenticoAoConstruido' -v` — Expected: PASS nos três.

Prova de que a ordem importa: mover temporariamente `promoverArenaSePresente(inv)` para **depois** do `ReplaceFile`, rodar `TestSaveOverwritesMappedCache` — Expected: FAIL (o rename falha com o arquivo mapeado). Restaurar. Colar.

- [ ] **Step 4: Apagar os encaminhadores**

`git rm internal/writer/atomic.go`. `go build ./...` — Expected: limpo (ninguém mais usa). `grep -rn "writer\.\(WriteAtomic\|SweepStaleTempFiles\|TempFilePrefix\|SweepResult\)" --include=*.go .` — Expected: vazio.

- [ ] **Step 5: `benchstat` da gravação**

`SaveIndexCacheReal` e `SaveInvertedCacheReal` antes (binário `antes_index`/`antes_search`) × depois, 7 intercalados. Esperado: `sec/op` sobe para perto das linhas `ComFsync` da Baseline (41,87 ms / 259,1 ms) — é o custo decidido no ruling da Task 171; **colar e registrar em `docs/ESTADO.md`** como medição publicada. Se subir **além** disso com `p < 0.05`, algo além do fsync entrou: investigar antes de commitar.

- [ ] **Step 6: Gate, órfãos e commit**

Run: `pwsh -File scripts/verify.ps1` — Expected: verde.
Run: `pwsh -File scripts/test_orphans.ps1` — Expected: quatro `[OK]` (o boot foi tocado).

```bash
git add internal/service/write.go cmd/gobsidian/servico.go internal/index/persist.go internal/search/persist.go internal/writer/atomic.go <testes tocados> docs/ESTADO.md
git commit -m "refactor: callers use vault atomic write; caches replace through vault.ReplaceFile with fsync"
```

#### Verificações

- `grep` do Step 4 vazio, colado.
- FAIL do Step 3 (ordem da promoção) colado.
- `benchstat` do Step 5 colado; números publicados em `ESTADO.md`.
- `go list -f '{{.Imports}}' ./internal/index/ ./internal/search/` não ganha `writer`.
- `test_orphans.ps1` quatro `[OK]`; `verify.ps1` verde.

#### Regras de execução

- Nunca `git checkout/restore/stash/clean/reset`. `git rm` por caminho é permitido.
- Formato de cache (`CacheFormatVersion`, `IndexCacheFormatVersion`) **não** muda: o que muda é como o arquivo chega ao disco, não o que tem dentro. Prova: `TestSaveAndLoadInvertedCache` e o equivalente de `index` carregam um cache gravado pelo binário `antes` (gerar um com `antes_search.test.exe -run TestSaveAndLoad… ` e apontar o teste para ele se houver mecanismo; se não, registrar "compatibilidade inferida do formato intacto, não testada" — honesto).
- Nenhuma aresta nova: `index`, `search`, `service` já importam `vault`.

#### Comando de mutação

Esta tarefa não tem prova de mutação por `mutate.ps1`: a regra provada é a ordem `promover → gravar`, e a prova é a inversão manual do Step 3 com o FAIL colado.

#### Contrato de relatório

`task-172-report.md`: status, SHA, `grep` vazio, FAIL do Step 3, `benchstat`, `go list`, `test_orphans.ps1`, última linha do `verify.ps1`.

---

### Task 173: d — `service` recebe `*index.Index`; a interface `service.Index` some

**Files:**
- Modify: `internal/service/service.go:11-24` (apagar `type Index interface {…}`), `:84` (`New(v *vault.Vault, idx *index.Index, inv *search.Inverted, w WatchStats, opts Options)`), campo `Service.index *index.Index`
- Modify: `internal/service/search.go:202-204` (apagar a asserção `s.index.(*index.Index)`), `:208` e `:298` (passar `s.index` direto a `search.CalculateBM25` e `search.GenerateSnippet`)
- Modify: `docs/ESTADO.md:136` (única menção à interface)
- Test: nenhum novo. Os existentes de `service`, `mcpsrv`, `daemon` e `cmd/gobsidian` já passam `*index.Index` a `New`.

**Interfaces:**
- Consumes: Task 158 (índice `nil` → `VAULT_UNAVAILABLE`): o teste `if s.index == nil` continua verdadeiro com ponteiro tipado — não há o alçapão de interface-com-ponteiro-nil, porque não há mais interface.
- Produces: `service.New(v *vault.Vault, idx *index.Index, inv *search.Inverted, w WatchStats, opts Options) *Service`. Tasks 174–177 chamam esta assinatura.

- [ ] **Step 1: Confirmar que a interface tem uma implementação e nenhum fake**

Run: `gopls references internal/service/service.go:#<offset de Index>` (ou `grep -rn "service\.Index\b\|Index interface" --include=*.go .`).
Expected: definição em `service.go:11`, parâmetro em `New`, campo `index`, asserção em `search.go:202`. Nenhum `type fakeIndex` em `_test.go`. Se aparecer um fake, **parar**: a Task muda de "apagar interface" para "manter interface e apagar asserção", e isso é ruling do orquestrador — reportar `BLOCKED` com o arquivo.

- [ ] **Step 2: Apagar a interface e tipar o campo**

`internal/service/service.go`:

```go
// Service e a fachada das tools sobre o dominio. Recebe o indice concreto:
// a interface que existia aqui tinha uma implementacao e nenhum fake, e
// search.CalculateBM25 e search.GenerateSnippet exigem *index.Index, o que
// obrigava uma assercao de tipo em cada busca.
type Service struct {
	vault    *vault.Vault
	index    *index.Index
	inverted *search.Inverted
	watch    WatchStats
	opts     Options
	// ... demais campos inalterados
}

func New(v *vault.Vault, idx *index.Index, inv *search.Inverted, w WatchStats, opts Options) *Service {
```

Apagar o bloco `type Index interface { … }` inteiro (`:11-24`). O import de `internal/index` já existe no pacote.

- [ ] **Step 3: Remover a asserção em `search.go`**

Substituir `:202-204`:

```go
	var idxImpl *index.Index
	if realIdx, ok := s.index.(*index.Index); ok {
		idxImpl = realIdx
	}
```

por nada, e trocar `idxImpl` por `s.index` nas chamadas `search.CalculateBM25(queryTokens, s.inverted, s.index)` (`:208`) e `search.GenerateSnippet(ctx, s.vault, s.inverted, s.index, …)` (`:298`). Se `idxImpl` aparecer em mais lugares, `gopls references` lista todos — trocar cada um.

Run: `go build ./... && go vet ./...` — Expected: limpo.
Run: `go test -race ./internal/service/ ./internal/mcpsrv/ ./internal/daemon/ ./cmd/...` — Expected: PASS.

- [ ] **Step 4: Prova de que o `nil` da Task 158 ainda é tratado**

Run: `go test -race ./internal/service/ -run 'Nil|Unavailable|Indisponivel' -v` — Expected: os testes da Task 158 passam com a assinatura nova (o nome exato está no relatório da 158; se nenhum casar com o padrão, `grep -n VAULT_UNAVAILABLE internal/service/*_test.go` mostra qual é).

- [ ] **Step 5: `benchstat` — a asserção sumiu; nada mais mudou**

`antes_service.test.exe` (Baseline) × `go test -c -o depois_service.test.exe ./internal/service/`, 7 intercalados, `-bench 'SearchDoisTermos|SearchFraseExata|SearchLimit200Cache|SearchTermoAmploCache' -benchmem`; `IndexBuild` em `antes_index` × `depois_index` (não muda — é a prova de que não muda). Expected: `~` em todos. Colar.

- [ ] **Step 6: Documentação**

`docs/ESTADO.md:136`: trocar a frase que descreve `service.Index` por "`service.New` recebe `*index.Index`; a interface foi removida em 2026-09 (Task 173) — tinha uma implementação e nenhum fake."

- [ ] **Step 7: Gate e commit**

Run: `pwsh -File scripts/verify.ps1` — Expected: verde.

```bash
git add internal/service/service.go internal/service/search.go docs/ESTADO.md
git commit -m "refactor(service): take *index.Index directly; the interface had one implementation and no fake"
```

#### Verificações

- `grep -rn "service\.Index\b\|idxImpl" --include=*.go .` vazio, colado.
- `benchstat` do Step 5 colado — `~` em todos.
- `verify.ps1` verde (última linha colada).

#### Regras de execução

- Nunca `git checkout/restore/stash/clean/reset`.
- Estrutural puro: nenhum comportamento muda, nenhum teste é editado. Se um teste precisar de edição para compilar, é porque existe um fake — voltar ao Step 1.
- `service` **não** ganha import: `index` já era importado.

#### Comando de mutação

Esta tarefa não tem prova de mutação: é estrutural, e a prova é `go build`, `go vet`, os testes da Task 158 e o `benchstat` com `~`.

#### Contrato de relatório

`task-173-report.md`: status, SHA, saída do Step 1, `grep` vazio, saída do Step 4, `benchstat`, última linha do `verify.ps1`.

---

### Task 174: e PR1 — `internal/boot`: abrir índice, preparar busca, montar componentes

**Files:**
- Create: `internal/boot/doc.go`, `internal/boot/indice.go`, `internal/boot/busca.go`, `internal/boot/montar.go`
- Create: `internal/boot/indice_test.go`, `internal/boot/busca_test.go`, `internal/boot/montar_test.go` (pacote `boot_test`)
- Move: `cmd/gobsidian/inverted_cache_state_test.go` → `internal/boot/estado_do_cache_test.go`; `cmd/gobsidian/boot_indice_busca_windows_test.go` → `internal/boot/busca_windows_test.go` (pacote `boot`, caixa-branca)
- Delete: `cmd/gobsidian/servico.go`
- Modify: `cmd/gobsidian/serve.go` (apagar `carregarIndiceDoCache:99`, `invertedSaveInterval`, `invertedCacheState:155`, `devolveMemoriaTransitoria:198`, `prepararIndiceDeBusca:213`, `buildInvertedIndex:274`, `watcherStats`; `serveEmProcesso:381` chama `boot.Montar`)
- Modify: `cmd/gobsidian/daemon.go:150-184` (chama `boot.Montar`)
- Modify: `CLAUDE.md` (árvore + grafo), `docs/ESTRUTURA.md:12-18,:216`, `docs/ARCHITECTURE.md` (camada de montagem)

**Interfaces:**
- Consumes: `service.New(v, idx *index.Index, inv, w, opts)` (Task 173); `vault.SweepStaleTempFiles` (Task 172); `vaulttest.Prazo` e os helpers de SO de `internal/vaulttest` (Task 159); `index.LoadIndexCache`, `index.VerifyFreshness`, `index.SaveIndexCache`; `search.LoadInvertedCache`, `(*Inverted).AdotarDe/MarkBuilding/MarkReady/Building/HasDoc/DocCount`; `watcher.New(v, idx, inv, debounce, log)`, `(*Watcher).Run(ctx)/Close()/Stats()`.
- Produces (Tasks 175–177 dependem destes nomes exatos):

```go
package boot

// AbrirIndice devolve o indice de metadados: do cache, se existir e estiver
// fresco, ou construido do cofre e gravado. origem e "cache" ou "build".
func AbrirIndice(ctx context.Context, v *vault.Vault, cfg config.Config, log *slog.Logger) (idx *index.Index, origem string, err error)

// PrepararBusca deixa inv pronto: adota o cache de busca completo, retoma um
// parcial, ou constroi do indice. Marca Ready ao fim, salvo ctx cancelado.
func PrepararBusca(ctx context.Context, v *vault.Vault, idx *index.Index, inv *search.Inverted, cfg config.Config, log *slog.Logger)

// Componentes e tudo que serve e daemon compartilham depois de montar.
type Componentes struct {
	Vault    *vault.Vault
	Index    *index.Index
	Inverted *search.Inverted
	Watcher  *watcher.Watcher
	Service  *service.Service
	espera   sync.WaitGroup
}

// Esperar bloqueia ate as goroutines de fundo (busca, watcher) terminarem.
func (c *Componentes) Esperar()

// Montar e o corpo do antigo construirServico: cofre, varredura de temporarios,
// indice, busca (eager ou preguicosa), watcher, Service, e o log
// "servidor pronto" que scripts/measure.ps1 le.
func Montar(ctx context.Context, cfg config.Config, log *slog.Logger) (*Componentes, error)
```

Grafo: `boot → config, index, search, service, vault, watcher` — todas arestas que `cmd/gobsidian` já tinha; nenhum pacote de domínio importa `boot`. Em `_test`: `+ vaulttest`.

- [ ] **Step 1: Testes de caracterização primeiro — o pacote ainda não existe, eles falham por compilação**

`internal/boot/indice_test.go`:

```go
package boot_test

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jonyd/gobsidian/internal/boot"
	"github.com/jonyd/gobsidian/internal/config"
	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/vault"
	"github.com/jonyd/gobsidian/internal/vaulttest"
)

func cofreDeTeste(t *testing.T) (*vault.Vault, config.Config) {
	t.Helper()
	raiz := t.TempDir()
	if err := os.WriteFile(filepath.Join(raiz, "a.md"), []byte("# A\n\nliga [[b]] #tag\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(raiz, "b.md"), []byte("# B\n\ntexto\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	v, err := vault.New(raiz)
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{VaultPath: raiz, CacheDir: t.TempDir()}
	return v, cfg
}

func logSilencioso() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestAbrirIndiceSemCacheConstroiEGrava(t *testing.T) {
	v, cfg := cofreDeTeste(t)
	idx, origem, err := boot.AbrirIndice(context.Background(), v, cfg, logSilencioso())
	if err != nil {
		t.Fatal(err)
	}
	if origem != "build" {
		t.Fatalf("origem = %q, quero \"build\"", origem)
	}
	if idx.NoteCount() != 2 {
		t.Fatalf("NoteCount = %d, quero 2", idx.NoteCount())
	}
	if _, _, err := index.LoadIndexCache(context.Background(), cfg.CacheDir, cfg.VaultPath); err != nil {
		t.Fatalf("AbrirIndice devia ter gravado o cache: %v", err)
	}
}

func TestAbrirIndiceComCacheFrescoCarrega(t *testing.T) {
	v, cfg := cofreDeTeste(t)
	if _, _, err := boot.AbrirIndice(context.Background(), v, cfg, logSilencioso()); err != nil {
		t.Fatal(err)
	}
	idx, origem, err := boot.AbrirIndice(context.Background(), v, cfg, logSilencioso())
	if err != nil {
		t.Fatal(err)
	}
	if origem != "cache" {
		t.Fatalf("origem = %q, quero \"cache\"", origem)
	}
	if idx.NoteCount() != 2 {
		t.Fatalf("NoteCount = %d, quero 2", idx.NoteCount())
	}
}

func TestAbrirIndiceComCacheVelhoReconstroi(t *testing.T) {
	v, cfg := cofreDeTeste(t)
	if _, _, err := boot.AbrirIndice(context.Background(), v, cfg, logSilencioso()); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(cfg.VaultPath, "a.md")
	if err := os.WriteFile(p, []byte("# A mudou\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// mtime explicito 2 s a frente: VerifyFreshness compara mtime e tamanho,
	// e dois writes no mesmo tick de relogio podem empatar.
	if err := os.Chtimes(p, time.Now().Add(2*time.Second), time.Now().Add(2*time.Second)); err != nil {
		t.Fatal(err)
	}
	idx, origem, err := boot.AbrirIndice(context.Background(), v, cfg, logSilencioso())
	if err != nil {
		t.Fatal(err)
	}
	if origem != "build" {
		t.Fatalf("origem = %q, quero \"build\" (cache velho)", origem)
	}
	_, cp, err := vault.Resolve(v.Root(), "a.md")
	if err != nil {
		t.Fatal(err)
	}
	n, ok := idx.Get(cp)
	if !ok || n.Title != "A mudou" {
		t.Fatalf("indice reconstruido nao viu a edicao: ok=%v n=%+v", ok, n)
	}
}

func TestAbrirIndiceComCacheCorrompidoReconstroi(t *testing.T) {
	v, cfg := cofreDeTeste(t)
	if _, _, err := boot.AbrirIndice(context.Background(), v, cfg, logSilencioso()); err != nil {
		t.Fatal(err)
	}
	entradas, err := os.ReadDir(cfg.CacheDir)
	if err != nil || len(entradas) != 1 {
		t.Fatalf("esperava um unico arquivo no cache dir, tenho %d (err=%v)", len(entradas), err)
	}
	p := filepath.Join(cfg.CacheDir, entradas[0].Name())
	if err := os.Truncate(p, 16); err != nil {
		t.Fatal(err)
	}
	_, origem, err := boot.AbrirIndice(context.Background(), v, cfg, logSilencioso())
	if err != nil {
		t.Fatal(err)
	}
	if origem != "build" {
		t.Fatalf("origem = %q, quero \"build\" (cache corrompido)", origem)
	}
}
```

Assinaturas reais (conferidas em 2026-09-02): `vault.New(root string, opcoes ...Opcao)`, `vault.Resolve(root, input string) (string, CanonicalPath, error)`, `index.LoadIndexCache(ctx, cacheDir, vaultPath string) (*Index, *CacheHeader, error)`, `index.Note.Title`.

`internal/boot/busca_test.go`:

```go
package boot_test

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/jonyd/gobsidian/internal/boot"
	"github.com/jonyd/gobsidian/internal/search"
)

func TestPrepararBuscaSemCacheConstroiEMarcaPronta(t *testing.T) {
	v, cfg := cofreDeTeste(t)
	idx, _, err := boot.AbrirIndice(context.Background(), v, cfg, logSilencioso())
	if err != nil {
		t.Fatal(err)
	}
	inv := search.NewInverted()
	inv.MarkBuilding()
	boot.PrepararBusca(context.Background(), v, idx, inv, cfg, logSilencioso())
	if inv.Building() {
		t.Fatal("PrepararBusca devia ter marcado Ready")
	}
	for _, p := range idx.NotePaths() {
		if !inv.HasDoc(p) {
			t.Fatalf("nota %q fora do indice de busca", p)
		}
	}
	if _, _, err := search.LoadInvertedCache(context.Background(), cfg.CacheDir, cfg.VaultPath); err != nil {
		t.Fatalf("PrepararBusca devia ter gravado o cache de busca: %v", err)
	}
}

func TestPrepararBuscaComCacheCompletoAdota(t *testing.T) {
	v, cfg := cofreDeTeste(t)
	idx, _, err := boot.AbrirIndice(context.Background(), v, cfg, logSilencioso())
	if err != nil {
		t.Fatal(err)
	}
	primeiro := search.NewInverted()
	primeiro.MarkBuilding()
	boot.PrepararBusca(context.Background(), v, idx, primeiro, cfg, logSilencioso())
	primeiro.Close()

	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	segundo := search.NewInverted()
	segundo.MarkBuilding()
	boot.PrepararBusca(context.Background(), v, idx, segundo, cfg, log)
	defer segundo.Close()
	if segundo.Building() {
		t.Fatal("devia estar pronto")
	}
	if !strings.Contains(buf.String(), "origem=cache") {
		t.Fatalf("log nao diz origem=cache:\n%s", buf.String())
	}
}

func TestPrepararBuscaCtxCanceladoNaoMarcaPronta(t *testing.T) {
	v, cfg := cofreDeTeste(t)
	idx, _, err := boot.AbrirIndice(context.Background(), v, cfg, logSilencioso())
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	inv := search.NewInverted()
	inv.MarkBuilding()
	boot.PrepararBusca(ctx, v, idx, inv, cfg, logSilencioso())
	if !inv.Building() {
		t.Fatal("ctx cancelado antes de construir: o indice de busca nao pode ser marcado Ready")
	}
	if _, _, err := search.LoadInvertedCache(context.Background(), cfg.CacheDir, cfg.VaultPath); err == nil {
		t.Fatal("ctx cancelado: nenhum cache de busca devia ter sido gravado")
	}
}
```

`search.LoadInvertedCache(ctx, cacheDir, vaultPath string) (*Inverted, *CacheHeader, error)` — conferida em 2026-09-02; o teste só usa o `err`. Se o carregamento com sucesso deixar um mapeamento aberto, fechar o `*Inverted` devolvido (`Close`).

`internal/boot/montar_test.go`:

```go
package boot_test

import (
	"context"
	"testing"
	"time"

	"github.com/jonyd/gobsidian/internal/boot"
	"github.com/jonyd/gobsidian/internal/vaulttest"
)

func TestMontarDevolveServicoPronto(t *testing.T) {
	_, cfg := cofreDeTeste(t)
	cfg.EagerSearch = true
	cfg.DebounceMS = 50
	ctx, cancel := context.WithCancel(context.Background())
	c, err := boot.Montar(ctx, cfg, logSilencioso())
	if err != nil {
		t.Fatal(err)
	}
	if c.Service == nil || c.Index == nil || c.Inverted == nil || c.Watcher == nil || c.Vault == nil {
		t.Fatalf("Componentes incompleto: %+v", c)
	}
	if c.Index.NoteCount() != 2 {
		t.Fatalf("NoteCount = %d, quero 2", c.Index.NoteCount())
	}
	cancel()
	_ = c.Watcher.Close()
	feito := make(chan struct{})
	go func() { c.Esperar(); close(feito) }()
	select {
	case <-feito:
	case <-time.After(vaulttest.Prazo):
		t.Fatal("Esperar nao voltou depois de cancelar ctx e fechar o watcher")
	}
}
```

Run: `go test ./internal/boot/` — Expected: FAIL de compilação (`package gobsidian/internal/boot` não existe). Colar a primeira linha.

- [ ] **Step 2: Criar o pacote movendo o código — sem reescrever**

`internal/boot/doc.go`:

```go
// Package boot monta o servidor: cofre, indice de metadados (do cache ou
// construido), indice de busca (adotado do cache, retomado ou construido),
// watcher e Service. Existe porque serve, daemon e os subcomandos de CLI
// precisam da mesma montagem, e ela vivia em cmd/gobsidian, onde nenhum
// teste de pacote a alcancava.
//
// Nao importa mcpsrv nem lifecycle: quem monta nao decide como o host
// conversa nem quando encerra (ver Task 177 para a vigilia do host).
package boot
```

`internal/boot/indice.go` — corpo de `carregarIndiceDoCache` (`serve.go:99`) mais o ramo `else` de `construirServico` (`index.New()+Build+SaveIndexCache`), na forma:

```go
func AbrirIndice(ctx context.Context, v *vault.Vault, cfg config.Config, log *slog.Logger) (*index.Index, string, error) {
	if idx, ok := carregarIndiceDoCache(ctx, v, cfg, log); ok {
		return idx, "cache", nil
	}
	idx := index.New()
	if err := idx.Build(ctx, v); err != nil {
		return nil, "", fmt.Errorf("construindo indice: %w", err)
	}
	if err := index.SaveIndexCache(ctx, cfg.CacheDir, cfg.VaultPath, idx); err != nil {
		// Cache e aceleracao, nao requisito: avisar e seguir com o indice em memoria.
		log.Warn("nao foi possivel gravar o cache de indice", "err", err)
	}
	return idx, "build", nil
}

// carregarIndiceDoCache e o antigo cmd/gobsidian.carregarIndiceDoCache, movido inteiro.
func carregarIndiceDoCache(ctx context.Context, v *vault.Vault, cfg config.Config, log *slog.Logger) (*index.Index, bool) {
	// corpo de serve.go:99-153, verbatim
}
```

As mensagens de log de `carregarIndiceDoCache` (Warn/Info) ficam como estão. Assinaturas reais: `(*Index).Build(ctx, v *vault.Vault) error`, `index.SaveIndexCache(ctx, cacheDir, vaultPath string, ix *Index) error`.

`internal/boot/busca.go` — mover `prepararIndiceDeBusca:213` (renomeada `PrepararBusca`), `buildInvertedIndex:274` (renomeada `construirBusca`, não exportada), `invertedCacheState:155` (renomeada `estadoDoCache`), `invertedSaveInterval` (renomeada `intervaloDeGravacao`) e `devolveMemoriaTransitoria:198`. Corpos verbatim; só o nome muda. O guarda `if ctx.Err() != nil {` dentro de `construirBusca` fica **exatamente** com essa grafia — é a âncora da mutação.

`internal/boot/montar.go` — `Componentes`, `Esperar`, `Montar` com o corpo de `construirServico` (`servico.go:49-247`): `vault.New(cfg.VaultPath, vault.SeguirSymlinks(cfg.FollowSymlinks))`; goroutine de `vault.SweepStaleTempFiles` juntada antes de `watcher.New` com os mesmos `Warn`; `AbrirIndice` no lugar do `if/else` de cache; `indexMS`; `inv := search.NewInverted(); inv.MarkBuilding()`; `watcher.New(v, idx, inv, time.Duration(cfg.DebounceMS)*time.Millisecond, log)` (a conversão que `servico.go` já faz — copiar a expressão real); `service.Options{ReadOnly, MaxResults}` e a closure `CarregarBusca` quando `!cfg.EagerSearch` (a closure chama `PrepararBusca`); `service.New(v, idx, inv, watcherStats{w: w}, opts)`; a goroutine em `c.espera` que, se eager, chama `PrepararBusca` e depois `w.Run(ctx)`; e o log **inalterado**:

```go
	log.Info("servidor pronto",
		"vault", cfg.VaultPath, "read_only", cfg.ReadOnly,
		"notes", idx.NoteCount(), "assets", idx.AssetCount(),
		"index_ms", indexMS, "index_origin", origem)
```

`index_origin` agora vem de `AbrirIndice` ("cache"/"build") — confirmar que os valores anteriores em `servico.go` eram estes dois literais; se eram outros (`"cached"`/`"built"`), manter os antigos: `scripts/measure.ps1` lê esta linha e o texto não muda nesta Task.

`watcherStats` (`serve.go`, fim do arquivo) muda para `montar.go`, não exportado.

- [ ] **Step 3: `serve.go` e `daemon.go` chamam `boot.Montar`**

`serveEmProcesso` (`serve.go:381`): `montado, err := construirServico(ctx, cfg, log)` → `c, err := boot.Montar(ctx, cfg, log)`; `montado.svc` → `c.Service`; `montado.w.Close()` → `c.Watcher.Close()`; `montado.wg.Wait()` → `c.Esperar()`. Idem em `daemon.go:150-184`. `git rm cmd/gobsidian/servico.go`. Apagar de `serve.go` tudo listado em **Files**.

`git mv cmd/gobsidian/inverted_cache_state_test.go internal/boot/estado_do_cache_test.go` (pacote `boot`; `invertedCacheState` → `estadoDoCache`). `git mv cmd/gobsidian/boot_indice_busca_windows_test.go internal/boot/busca_windows_test.go` (pacote `boot`; `buildInvertedIndex` → `construirBusca`; a build tag `//go:build windows` permanece).

Run: `go build ./... && go vet ./...` — Expected: limpo.
Run: `go test -race ./internal/boot/ -v` — Expected: os 8 testes PASS.
Run: `go test -race ./cmd/... ./internal/daemon/ ./internal/mcpsrv/` — Expected: PASS.

- [ ] **Step 4: Prova de mutação**

```powershell
pwsh -File scripts/mutate.ps1 -Path internal/boot/busca.go -Anchor 'if ctx.Err() != nil {' -Replacement 'if false {' -Test TestPrepararBuscaCtxCanceladoNaoMarcaPronta -Package ./internal/boot/
```

Expected: exit 0 (o teste FALHA sob mutação: com o guarda morto, o índice é construído e marcado Ready mesmo com ctx cancelado). Colar a saída.

- [ ] **Step 5: `go list` e órfãos**

Run: `go list -f '{{.ImportPath}} {{.Imports}}' ./internal/boot/ | tr ' ' '\n' | grep gobsidian` — Expected: exatamente `config index search service vault watcher`. Nenhum `mcpsrv`, nenhum `lifecycle`, nenhum `net`.
Run: `go list -f '{{.Imports}}' ./internal/index/ ./internal/search/ ./internal/service/ ./internal/watcher/ | grep -c boot` — Expected: `0`.
Run: `pwsh -File scripts/test_orphans.ps1` — Expected: quatro `[OK]`.

- [ ] **Step 6: Documentação**

- `CLAUDE.md`, árvore: linha `boot/  monta cofre, indice, busca, watcher e Service; serve, daemon e CLI chamam` entre `console/` e `ipc/`; grafo: linha `boot     → config, index, search, service, vault, watcher` depois de `mcpsrv`. Remover `servico.go` da descrição de `cmd/gobsidian/`.
- `docs/ESTRUTURA.md:12-18`: apagar `servico.go`, acrescentar `internal/boot/` com os quatro arquivos; `:216` idem.
- `docs/ARCHITECTURE.md`: um parágrafo na seção de camadas — "montagem" é camada própria acima de `service` e abaixo de `mcpsrv`/`cmd`.

Run: `python -c "open('CLAUDE.md',encoding='utf-8').read()" && echo "[OK] UTF-8 valido"`.

- [ ] **Step 7: Medição — o boot não pode ficar mais lento por ter mudado de pacote**

`scripts/measure.ps1 -Vault <vault_5000>` antes (binário da Baseline/M4) e depois, 3 execuções cada, `index_ms` e wall de `servidor pronto`. Expected: dentro do ruído de M4 (101–123 ms `index_ms` quente). Colar as seis linhas.

- [ ] **Step 8: Gate e commit**

Run: `pwsh -File scripts/verify.ps1` — Expected: verde.

```bash
git add internal/boot/ cmd/gobsidian/serve.go cmd/gobsidian/daemon.go cmd/gobsidian/servico.go cmd/gobsidian/inverted_cache_state_test.go cmd/gobsidian/boot_indice_busca_windows_test.go CLAUDE.md docs/ESTRUTURA.md docs/ARCHITECTURE.md
git commit -m "refactor(boot): assemble vault, index, search and service in one testable package"
```

#### Verificações

- FAIL de compilação do Step 1 colado; 8 PASS do Step 3 colados.
- `mutate.ps1` do Step 4 com exit 0, saída colada.
- `go list` do Step 5 com a lista exata; `test_orphans.ps1` quatro `[OK]`.
- Linha `servidor pronto` com as mesmas chaves de antes (colar uma do `serve` real).
- Medição do Step 7 colada; `verify.ps1` verde.

#### Regras de execução

- Nunca `git checkout/restore/stash/clean/reset`. `git mv`/`git rm` por caminho são permitidos.
- Movimento, não reescrita: corpos verbatim, só nomes e assinaturas mudam. Melhoria que você enxergar vai para o relatório como sugestão, não para o diff.
- `boot` não importa `mcpsrv`, `lifecycle`, `ipc`, `daemon`, `doctor`. Nenhum pacote de domínio importa `boot`.
- Log só por `log *slog.Logger` recebido; nenhum `fmt.Print*`.
- Testes de caracterização usam cofre real em `t.TempDir()` e `CacheDir` em `t.TempDir()` — nunca o cache padrão do usuário.

#### Comando de mutação

Step 4 (`mutate.ps1`, âncora `if ctx.Err() != nil {` em `internal/boot/busca.go`).

#### Contrato de relatório

`task-174-report.md`: status, SHA, saídas dos Steps 1, 3, 4, 5, 7, linha `servidor pronto` real, última linha do `verify.ps1`, e a lista de nomes antigos → novos.

---
### Task 175: `search` de CLI abre índice e busca como o `serve` — e ganha `--cache-dir`/`--log-level`

**Files:**
- Modify: `cmd/gobsidian/search.go:35-49` (troca `index.New()+Build` + laço `inv.Update` por `boot.AbrirIndice` + `boot.PrepararBusca`), `:89-97` (flags `--cache-dir`, `--log-level`)
- Create: `cmd/gobsidian/cli_log.go` (`loggerDeCLI`)
- Modify: `internal/config/config.go:29,:55,:82-95` (`Config.LogLevelExplicito bool`)
- Modify: `cmd/gobsidian/cli_subcommands_test.go` (`TestSearchCmd_StdoutAndJSON:77` passa `--cache-dir` de `t.TempDir()`)
- Create: `cmd/gobsidian/search_cache_test.go` (`TestSearchCLISegundaExecucaoUsaCache`)
- Modify: `README.md` (tabela de flags de `search`), `docs/ESTADO.md` e `docs/OPERACAO.md` (medição M4 depois)

**Interfaces:**
- Consumes: `boot.AbrirIndice(ctx, v, cfg, log) (*index.Index, string, error)`, `boot.PrepararBusca(ctx, v, idx, inv, cfg, log)` (Task 174); `service.New(v, idx *index.Index, inv, nil, opts)` (Task 173); Task 156 já removeu `--read-only`/`--debounce-ms` de `search` e manteve `--max-results`.
- Produces: `func loggerDeCLI(cmd *cobra.Command, cfg config.Config) *slog.Logger` em `cmd/gobsidian/cli_log.go`; `config.Config.LogLevelExplicito bool`. Task 176 usa os dois.

- [ ] **Step 1: `LogLevelExplicito` em `config`**

`internal/config/config.go`, no `Config` (`:55`): campo `LogLevelExplicito bool` com o comentário `// LogLevelExplicito e true quando GOBSIDIAN_LOG_LEVEL ou --log-level foi dado; os subcomandos de CLI so mostram log acima de Warn sem ele.` Em `Load` (`:82-95`), nos dois ramos que atribuem `cfg.LogLevel`, também `cfg.LogLevelExplicito = true`.

Teste em `internal/config/config_test.go` (acrescentar ao arquivo existente):

```go
func TestLoadMarcaLogLevelExplicito(t *testing.T) {
	t.Setenv("GOBSIDIAN_LOG_LEVEL", "")
	cfg, err := Load(Flags{VaultPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.LogLevelExplicito {
		t.Fatal("sem env e sem flag, LogLevelExplicito devia ser false")
	}
	cfg, err = Load(Flags{VaultPath: t.TempDir(), LogLevel: "debug"})
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.LogLevelExplicito || cfg.LogLevel != slog.LevelDebug {
		t.Fatalf("flag dada: Explicito=%v LogLevel=%v", cfg.LogLevelExplicito, cfg.LogLevel)
	}
	t.Setenv("GOBSIDIAN_LOG_LEVEL", "warn")
	cfg, err = Load(Flags{VaultPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.LogLevelExplicito || cfg.LogLevel != slog.LevelWarn {
		t.Fatalf("env dada: Explicito=%v LogLevel=%v", cfg.LogLevelExplicito, cfg.LogLevel)
	}
}
```

Run: `go test ./internal/config/ -run TestLoadMarcaLogLevelExplicito -v` — Expected: FAIL (campo não existe) e depois PASS.

- [ ] **Step 2: `loggerDeCLI`**

`cmd/gobsidian/cli_log.go`:

```go
package main

import (
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/jonyd/gobsidian/internal/config"
)

// loggerDeCLI e o logger dos subcomandos de CLI (search, index, inspect):
// escreve em cmd.ErrOrStderr() para que teste capture, e cala tudo abaixo de
// Warn a menos que o operador tenha pedido um nivel — um subcomando que
// imprime "servidor pronto" a cada chamada polui o terminal de quem so
// queria o resultado.
func loggerDeCLI(cmd *cobra.Command, cfg config.Config) *slog.Logger {
	nivel := slog.LevelWarn
	if cfg.LogLevelExplicito {
		nivel = cfg.LogLevel
	}
	return slog.New(slog.NewTextHandler(cmd.ErrOrStderr(), &slog.HandlerOptions{Level: nivel}))
}
```

- [ ] **Step 3: `search` usa `boot`**

`cmd/gobsidian/search.go`, substituir `:44-49` (o `index.New()`, o `Build` e o laço `inv.Update`) por:

```go
			log := loggerDeCLI(cmd, cfg)
			idx, _, err := boot.AbrirIndice(cmd.Context(), v, cfg, log)
			if err != nil {
				return err
			}
			inv := search.NewInverted()
			inv.MarkBuilding()
			boot.PrepararBusca(cmd.Context(), v, idx, inv, cfg, log)
			defer inv.Close()
```

Flags (`:89-97`): acrescentar

```go
	cmd.Flags().StringVar(&flags.CacheDir, "cache-dir", "", "diretorio do cache de indice")
	cmd.Flags().StringVar(&flags.LogLevel, "log-level", "", "debug, info, warn ou error")
```

Remover o import de `index` se ficar sem uso. `go build ./... && go vet ./...` — Expected: limpo.

- [ ] **Step 4: Testes**

`cli_subcommands_test.go:77 TestSearchCmd_StdoutAndJSON`: acrescentar `"--cache-dir", t.TempDir()` aos `SetArgs` das duas execuções. Sem isso o teste grava em `os.UserCacheDir()/gobsidian/<chave>` — o cache real do usuário para um cofre de teste. A asserção "stderr vazio" continua: `loggerDeCLI` cala Info sem `--log-level`.

`cmd/gobsidian/search_cache_test.go`:

```go
package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSearchCLISegundaExecucaoUsaCache(t *testing.T) {
	cofre := t.TempDir()
	if err := os.WriteFile(filepath.Join(cofre, "n.md"), []byte("# N\n\nexecucao do teste\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cache := t.TempDir()
	roda := func() string {
		var stdout, stderr bytes.Buffer
		cmd := newSearchCmd()
		cmd.SetOut(&stdout)
		cmd.SetErr(&stderr)
		cmd.SetArgs([]string{"--vault", cofre, "--cache-dir", cache, "--log-level", "info", "--json", "execucao"})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("search: %v\nstderr:\n%s", err, stderr.String())
		}
		if !strings.Contains(stdout.String(), `"n.md"`) {
			t.Fatalf("stdout sem o resultado esperado:\n%s", stdout.String())
		}
		return stderr.String()
	}
	primeira := roda()
	if !strings.Contains(primeira, "origem=construcao") {
		t.Fatalf("primeira execucao devia construir o indice de busca:\n%s", primeira)
	}
	segunda := roda()
	if !strings.Contains(segunda, "origem=cache") {
		t.Fatalf("segunda execucao devia adotar o cache de busca:\n%s", segunda)
	}
}
```

`newSearchCmd` é o nome real do construtor em `search.go` (conferir; se for `searchCmd()` ou outro, usar o real). O literal `"n.md"` supõe que o JSON serializa `Path` como caminho canônico relativo — `TestSearchCmd_StdoutAndJSON` já prende esse formato; copiar dele a asserção se for diferente.

Run: `go test -race ./cmd/gobsidian/ -run 'TestSearch' -v` — Expected: PASS nos dois.

- [ ] **Step 5: Prova manual de mutação**

Substituir temporariamente o bloco do Step 3 pelo antigo (`index.New()` + `Build` + laço `inv.Update` + `inv.MarkReady()`), rodar `go test ./cmd/gobsidian/ -run TestSearchCLISegundaExecucaoUsaCache` — Expected: FAIL ("primeira execucao devia construir…" ou "segunda execucao devia adotar…": sem `PrepararBusca` nenhuma linha `origem=` aparece). Restaurar. Colar.

- [ ] **Step 6: Medição M4 — o objetivo da Task**

Binário `antes`: `scripts/build.ps1` no commit da Task 174 (o `search` ainda constrói tudo) → `gobsidian_antes.exe`. Binário `depois`: build deste commit.

Para cada binário, com o cofre `vault_5000` (Baseline) e `--cache-dir` num diretório novo:
- 3 execuções **frias** (apagar o diretório de cache antes de cada uma): `Measure-Command { .\gobsidian_X.exe search --json --limit 200 --vault <vault_5000> --cache-dir <dir> "execucao" }`.
- 3 execuções **quentes** (mesmo diretório, cache já gravado).

Expected: `antes` fria ≈ quente ≈ M4 (10 520 / 10 819 / 11 046 ms); `depois` fria ≈ M4 + custo de gravar os dois caches; `depois` quente perto do `serve` quente de M4 (wall 475–528 ms). Colar as 12 linhas em `docs/ESTADO.md` (medições publicadas) e resumir em `docs/OPERACAO.md` na seção do `search`. Se `depois` quente não ficar abaixo de 1/5 da fria, a Task não entregou o que promete — reportar `DONE_WITH_CONCERNS` com os números.

- [ ] **Step 7: README**

Tabela de flags de `search` (`README.md:196-206`, já editada pela Task 156): acrescentar `--cache-dir` e `--log-level` com a mesma descrição das flags de `serve`; nota de uma linha: "`search` reaproveita o cache de índice e de busca do `serve`; a primeira execução constrói e grava, as seguintes carregam."

- [ ] **Step 8: Gate e commit**

Run: `pwsh -File scripts/verify.ps1` — Expected: verde.

```bash
git add cmd/gobsidian/search.go cmd/gobsidian/cli_log.go cmd/gobsidian/search_cache_test.go cmd/gobsidian/cli_subcommands_test.go internal/config/config.go internal/config/config_test.go README.md docs/ESTADO.md docs/OPERACAO.md
git commit -m "feat(cli): search loads the index and the search cache like serve does"
```

#### Verificações

- Step 1 FAIL→PASS colado; Step 4 PASS colado; Step 5 FAIL colado.
- 12 linhas de medição do Step 6 coladas e publicadas.
- `grep -n "cache-dir" cmd/gobsidian/cli_subcommands_test.go` mostra o `search` com `--cache-dir`.
- `verify.ps1` verde.

#### Regras de execução

- Nunca `git checkout/restore/stash/clean/reset`.
- Nenhum teste de CLI grava fora de `t.TempDir()`: todo `SetArgs` com `--vault` de teste leva `--cache-dir`.
- `search` continua CLI: stdout é do resultado, stderr do log; o comentário existente em `search.go` sobre stdout permanece.
- Não rodar `test_orphans.ps1` em paralelo com a medição do Step 6.

#### Comando de mutação

Esta tarefa não tem prova de mutação por `mutate.ps1`: a prova é a substituição manual do Step 5 com o FAIL colado.

#### Contrato de relatório

`task-175-report.md`: status, SHA, saídas dos Steps 1, 4, 5, 6 (as 12 linhas), última linha do `verify.ps1`.

---

### Task 176: `index` e `inspect` abrem pelo cache; flags de cofre registradas uma vez

**Files:**
- Modify: `cmd/gobsidian/index.go:41-52` (`boot.AbrirIndice`), `indexSummaryJSON` (campo `Origin`), console (`Origem:`), flags
- Modify: `cmd/gobsidian/inspect.go:37-51` (`boot.AbrirIndice`), flags
- Create: `cmd/gobsidian/flags.go` (`flagsDeCofre`, `flagsDeCache`)
- Modify: `cmd/gobsidian/search.go`, `serve.go:47-51`, `daemon.go:56-60`, `doctor.go:67-71` (usam `flagsDeCofre`/`flagsDeCache` em vez de registrar `--vault`, `--follow-symlinks`, `--cache-dir`, `--log-level` à mão)
- Modify: `cmd/gobsidian/cli_subcommands_test.go` (`TestIndexCmd_StdoutAndJSON:30`, `TestInspectCmd_StdoutAndJSON:119` levam `--cache-dir`)
- Create: `cmd/gobsidian/index_origem_test.go`
- Modify: `README.md` (flags de `index`/`inspect`; campo `origin` do `index --json`), `docs/OPERACAO.md` (idem)

**Interfaces:**
- Consumes: `boot.AbrirIndice` (Task 174), `loggerDeCLI` e `Config.LogLevelExplicito` (Task 175). Task 156 já removeu `--read-only/--debounce-ms/--max-results` de `index`/`inspect`.
- Produces: `indexSummaryJSON.Origin string json:"origin"` ("build" | "cache"); `func flagsDeCofre(cmd *cobra.Command, f *config.Flags)` e `func flagsDeCache(cmd *cobra.Command, f *config.Flags)`.

- [ ] **Step 1: Teste da origem — falha porque o campo não existe**

`cmd/gobsidian/index_origem_test.go`:

```go
package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestIndexCmdJSONTrazOrigem(t *testing.T) {
	cofre := t.TempDir()
	if err := os.WriteFile(filepath.Join(cofre, "n.md"), []byte("# N\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cache := t.TempDir()
	roda := func() string {
		var stdout, stderr bytes.Buffer
		cmd := newIndexCmd()
		cmd.SetOut(&stdout)
		cmd.SetErr(&stderr)
		cmd.SetArgs([]string{"--vault", cofre, "--cache-dir", cache, "--json"})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("index: %v\nstderr:\n%s", err, stderr.String())
		}
		var out struct {
			Origin string `json:"origin"`
			Notes  int    `json:"notes"`
		}
		if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
			t.Fatalf("JSON invalido: %v\n%s", err, stdout.String())
		}
		if out.Notes != 1 {
			t.Fatalf("notes = %d, quero 1", out.Notes)
		}
		return out.Origin
	}
	if o := roda(); o != "build" {
		t.Fatalf("primeira execucao: origin = %q, quero \"build\"", o)
	}
	if o := roda(); o != "cache" {
		t.Fatalf("segunda execucao: origin = %q, quero \"cache\"", o)
	}
}
```

`newIndexCmd` e a chave JSON `notes` são as reais de `index.go` (conferir `indexSummaryJSON`; se a tag for outra, usar a outra). Run: `go test ./cmd/gobsidian/ -run TestIndexCmdJSONTrazOrigem` — Expected: FAIL (`origin` vazio).

- [ ] **Step 2: `index` e `inspect` via `boot.AbrirIndice`**

`index.go:41-52`:

```go
			log := loggerDeCLI(cmd, cfg)
			start := time.Now()
			idx, origem, err := boot.AbrirIndice(cmd.Context(), v, cfg, log)
			if err != nil {
				return err
			}
			dur := time.Since(start)
```

`indexSummaryJSON` ganha `Origin string \`json:"origin"\`` preenchido com `origem`; na saída de console, uma linha `con.Item("Origem: %s", origem)` junto das demais. `inspect.go:37-51`: mesma troca (`idx, _, err := boot.AbrirIndice(...)`).

Run: `go test ./cmd/gobsidian/ -run TestIndexCmdJSONTrazOrigem` — Expected: PASS.

- [ ] **Step 3: Flags registradas uma vez**

`cmd/gobsidian/flags.go`:

```go
package main

import (
	"github.com/spf13/cobra"

	"github.com/jonyd/gobsidian/internal/config"
)

// flagsDeCofre registra as flags que todo subcomando que abre um cofre
// aceita. Seis arquivos registravam --vault e --follow-symlinks com o mesmo
// texto; quando o texto mudou, mudou em cinco.
func flagsDeCofre(cmd *cobra.Command, f *config.Flags) {
	cmd.Flags().StringVar(&f.VaultPath, "vault", "", "caminho da raiz do cofre (obrigatorio)")
	cmd.Flags().BoolVar(&f.FollowSymlinks, "follow-symlinks", false,
		"segue symlink dentro do cofre; o padrao recusa, porque o confinamento nao alcanca o alvo")
}

// flagsDeCache registra as flags de quem le ou grava o cache de indice.
func flagsDeCache(cmd *cobra.Command, f *config.Flags) {
	cmd.Flags().StringVar(&f.CacheDir, "cache-dir", "", "diretorio do cache de indice")
	cmd.Flags().StringVar(&f.LogLevel, "log-level", "", "debug, info, warn ou error")
}
```

Trocar os registros em `search.go`, `index.go`, `inspect.go`, `serve.go`, `daemon.go`, `doctor.go` (`doctor` só `flagsDeCofre`). O texto de ajuda é o de `serve.go` hoje; se algum arquivo tinha texto diferente, o de `serve` vence e a diferença vai para o relatório.

Run: `go build ./... && go vet ./...`; `grep -rn '"vault", ""' cmd/gobsidian/*.go` — Expected: só `flags.go`.
Run: `go test -race ./cmd/gobsidian/` — Expected: PASS (`TestSubcommands_FlagsSetPopulated:168` inclusive; se ele testava flags que a Task 156 apagou, já foi ajustado lá).

- [ ] **Step 4: `--cache-dir` nos testes de CLI existentes**

`TestIndexCmd_StdoutAndJSON:30` e `TestInspectCmd_StdoutAndJSON:119`: acrescentar `"--cache-dir", t.TempDir()` a cada `SetArgs`. Motivo idêntico ao da Task 175 — sem isso o teste grava no cache real do usuário.

- [ ] **Step 5: Prova de mutação**

```powershell
pwsh -File scripts/mutate.ps1 -Path cmd/gobsidian/index.go -Anchor 'Origin: origem,' -Replacement 'Origin: "build",' -Test TestIndexCmdJSONTrazOrigem -Package ./cmd/gobsidian/
```

Expected: exit 0 (segunda execução falha: `origin = "build", quero "cache"`). Se o literal da âncora não bater com o código escrito, ajustar a âncora ao texto real — a mutação é "origem fixa", não a grafia.

- [ ] **Step 6: Documentação**

`README.md`: flags de `index` e `inspect` ganham `--cache-dir`/`--log-level`; `index --json` documenta `origin` ("build" na primeira execução, "cache" quando o cache está fresco). `docs/OPERACAO.md`: idem, na seção de `index`.

- [ ] **Step 7: Gate e dois commits**

Run: `pwsh -File scripts/verify.ps1` — Expected: verde.

```bash
git add cmd/gobsidian/index.go cmd/gobsidian/inspect.go cmd/gobsidian/index_origem_test.go cmd/gobsidian/cli_subcommands_test.go README.md docs/OPERACAO.md
git commit -m "feat(cli): index and inspect load from the cache"
git add cmd/gobsidian/flags.go cmd/gobsidian/search.go cmd/gobsidian/index.go cmd/gobsidian/inspect.go cmd/gobsidian/serve.go cmd/gobsidian/daemon.go cmd/gobsidian/doctor.go
git commit -m "refactor(cli): register shared vault flags once"
```

O segundo commit é estrutural puro; se o primeiro já tiver tocado flags em `index.go`/`inspect.go` (registrar `--cache-dir` à mão para o teste passar), tudo bem — o segundo substitui pelo registro comum.

#### Verificações

- Step 1 FAIL→PASS colado; `grep` do Step 3 colado.
- `mutate.ps1` exit 0 colado.
- `git log --oneline -2` mostra os dois commits na ordem.
- `verify.ps1` verde.

#### Regras de execução

- Nunca `git checkout/restore/stash/clean/reset`.
- Todo teste de CLI com `--vault` leva `--cache-dir` de `t.TempDir()`.
- `doctor` não ganha `flagsDeCache`: não lê cache.
- Estrutura (commit 2) separada de comportamento (commit 1).

#### Comando de mutação

Step 5 (`mutate.ps1`, âncora `Origin: origem,` em `cmd/gobsidian/index.go`).

#### Contrato de relatório

`task-176-report.md`: status, dois SHAs, saídas dos Steps 1, 3, 5, última linha do `verify.ps1`, e a lista de textos de ajuda que divergiam (se houver).

---

### Task 177: e PR2 — `boot.VigiarHost` e os passos de shutdown que `serve`, ponte e `daemon` repetem

**Files:**
- Create: `internal/boot/vigia.go` (`Vigia`, `VigiarHost`, `PassoFecharEspelho`), `internal/boot/espelho.go` (`mirrorReader`, `mirrorDst` movidos de `serve.go`), `internal/boot/vigia_test.go`
- Move: os testes `TestMirrorReaderCopiesToMirror:76`, `TestMirrorReaderPropagatesEOF:143`, `TestMirrorReaderBrokenMirrorDoesNotPoisonRead:189` de `cmd/gobsidian/serve_test.go` → `internal/boot/espelho_test.go` (pacote `boot`; `boundedWait` → `vaulttest.Prazo` com `select`)
- Modify: `internal/boot/montar.go` (`(*Componentes).PassoWatcher() lifecycle.Step`)
- Modify: `cmd/gobsidian/serve.go:381-430` (`serveEmProcesso`), `cmd/gobsidian/ponte.go:155-257` (`servePonteRemota`), `cmd/gobsidian/daemon.go:150-184`
- Modify: `CLAUDE.md` (grafo: `boot` ganha `lifecycle`), `docs/ARCHITECTURE.md`

**Interfaces:**
- Consumes: `lifecycle.New(parent, Options{Stdin, ParentPID, Logger}) (ctx, *Lifecycle)`, `lifecycle.ParentPID()`, `lifecycle.Step{Name, Budget, Fn}`, `lifecycle.Shutdown(ctx, log, hardLimit, steps...)`, `(*Lifecycle).Reason()/Wait()`; `boot.Componentes` (Task 174); `vaulttest.Prazo` (Task 159).
- Produces:

```go
// Vigia e o que o processo precisa para saber quando o host foi embora:
// o ctx que cancela, o Lifecycle que explica por que, e o stdin espelhado
// que o servidor MCP le enquanto o lifecycle vigia o EOF.
type Vigia struct {
	Ctx   context.Context
	LC    *lifecycle.Lifecycle
	Stdin io.Reader
	pw    *io.PipeWriter
}

// VigiarHost liga a vigilia: EOF em stdin ou morte do PID pai cancelam Ctx.
func VigiarHost(parent context.Context, stdin io.Reader, log *slog.Logger) *Vigia

// PassoFecharEspelho fecha o pipe para que o leitor de stdin do servidor
// receba EOF; 500 ms de orcamento.
func (v *Vigia) PassoFecharEspelho() lifecycle.Step

// PassoWatcher fecha o watcher; 500 ms de orcamento.
func (c *Componentes) PassoWatcher() lifecycle.Step
```

Aresta nova: `boot → lifecycle`. Justificativa: o mesmo andaime pipe + `mirrorReader` + `lifecycle.New` está em `serve.go:384-391` e em `ponte.go:161-168`, e os passos `close-pipe`/`watcher` estão nos três pontos de saída. `lifecycle` continua folha.

- [ ] **Step 1: Teste da vigília — falha por compilação**

`internal/boot/vigia_test.go`:

```go
package boot_test

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/jonyd/gobsidian/internal/boot"
	"github.com/jonyd/gobsidian/internal/vaulttest"
)

func TestVigiarHostEOFDoStdinCancelaCtx(t *testing.T) {
	pr, pw := io.Pipe()
	v := boot.VigiarHost(context.Background(), pr, logSilencioso())
	// O servidor leria v.Stdin; aqui basta drenar para o espelho nao travar.
	go func() { _, _ = io.Copy(io.Discard, v.Stdin) }()
	if _, err := pw.Write([]byte("{}\n")); err != nil {
		t.Fatal(err)
	}
	_ = pw.Close()
	select {
	case <-v.Ctx.Done():
	case <-time.After(vaulttest.Prazo):
		t.Fatal("EOF em stdin nao cancelou o ctx")
	}
	if r := v.LC.Reason(); r != "stdin-eof" {
		t.Fatalf("Reason = %q, quero \"stdin-eof\"", r)
	}
	passo := v.PassoFecharEspelho()
	if passo.Name != "close-pipe" || passo.Budget != 500*time.Millisecond {
		t.Fatalf("PassoFecharEspelho = {%q %v}", passo.Name, passo.Budget)
	}
	if err := passo.Fn(context.Background()); err != nil {
		t.Fatalf("fechar o espelho: %v", err)
	}
}
```

`logSilencioso` é o de `indice_test.go` (mesmo pacote `boot_test`). A assinatura de `Step.Fn` é a real de `lifecycle` (conferir se recebe `ctx`; ajustar a chamada). Run: `go test ./internal/boot/ -run TestVigiarHostEOFDoStdinCancelaCtx` — Expected: FAIL de compilação.

- [ ] **Step 2: Mover `mirrorReader` e criar `Vigia`**

`internal/boot/espelho.go`: `mirrorReader` e `mirrorDst` de `serve.go`, verbatim, com o comentário de porquê o espelho existe. A linha `_ = m.dst.CloseWithError(err)` fica **exatamente** assim — âncora da mutação.

`internal/boot/vigia.go`:

```go
func VigiarHost(parent context.Context, stdin io.Reader, log *slog.Logger) *Vigia {
	pr, pw := io.Pipe()
	ctx, lc := lifecycle.New(parent, lifecycle.Options{
		Stdin:     pr,
		ParentPID: lifecycle.ParentPID(),
		Logger:    log,
	})
	return &Vigia{
		Ctx:   ctx,
		LC:    lc,
		Stdin: &mirrorReader{src: stdin, dst: pw},
		pw:    pw,
	}
}

func (v *Vigia) PassoFecharEspelho() lifecycle.Step {
	return lifecycle.Step{Name: "close-pipe", Budget: 500 * time.Millisecond, Fn: func(context.Context) error {
		return v.pw.Close()
	}}
}
```

Isto é o que `serve.go:384-391` faz hoje; copiar dali a ordem exata (quem é `src`, quem é `dst`, o que `lifecycle.New` recebe) em vez de confiar no esboço.

`internal/boot/montar.go`:

```go
func (c *Componentes) PassoWatcher() lifecycle.Step {
	return lifecycle.Step{Name: "watcher", Budget: 500 * time.Millisecond, Fn: func(context.Context) error {
		return c.Watcher.Close()
	}}
}
```

Run: `go test -race ./internal/boot/ -v` — Expected: PASS em todos (inclusive os três `TestMirrorReader*` movidos).

- [ ] **Step 3: Os três pontos de saída usam `Vigia`**

`serveEmProcesso` (`serve.go:381`): `vig := boot.VigiarHost(parent, os.Stdin, log)`; `ctx := vig.Ctx`; `c, err := boot.Montar(ctx, cfg, log)`; `mcpsrv.New(ctx, c.Service, cfg, log)` lê de `vig.Stdin` (onde hoje lê de `pr`); `lifecycle.Shutdown(ctx, log, 6*time.Second, <passo in-flight 3 s como hoje>, vig.PassoFecharEspelho(), c.PassoWatcher())`; `vig.LC.Wait(); c.Esperar()`.

`servePonteRemota` (`ponte.go:155`): mesmo andaime; os passos `half-close` (2 s, `conn.CloseWrite`) e `close-conn` ficam onde estão, `close-pipe` vira `vig.PassoFecharEspelho()`.

`daemon.go:150-184`: não tem stdin; continua com `lifecycle.New(parent, Options{Logger})`, e o passo `watcher` vira `c.PassoWatcher()`.

Apagar `mirrorReader`/`mirrorDst` de `serve.go`; `git mv` dos três testes para `internal/boot/espelho_test.go`; `TestShutdownExitCode:235` fica em `serve_test.go`.

Run: `go build ./... && go vet ./... && go test -race ./cmd/... ./internal/boot/` — Expected: PASS.

- [ ] **Step 4: Prova de mutação**

```powershell
pwsh -File scripts/mutate.ps1 -Path internal/boot/espelho.go -Anchor '_ = m.dst.CloseWithError(err)' -Replacement '_ = err' -Test TestMirrorReaderPropagatesEOF -Package ./internal/boot/
```

Expected: exit 0 (sem propagar o EOF ao espelho, o lifecycle nunca vê stdin fechar). Colar.

- [ ] **Step 5: Órfãos — o teste que importa nesta Task**

Run: `pwsh -File scripts/test_orphans.ps1` — Expected: quatro `[OK]`. Colar os quatro. Se um falhar, é esta Task: o andaime mudou de lugar e algo ficou fora de ordem (o `mirrorReader` precisa existir **antes** do `lifecycle.New` receber o `pr`).

- [ ] **Step 6: Grafo e documentação**

Run: `go list -f '{{.Imports}}' ./internal/boot/ | tr ' ' '\n' | grep gobsidian` — Expected: `config index lifecycle search service vault watcher`.
`CLAUDE.md`: linha `boot → config, index, lifecycle, search, service, vault, watcher` e a justificativa da aresta em uma frase (o andaime duplicado). `docs/ARCHITECTURE.md`: parágrafo sobre `VigiarHost` na seção de ciclo de vida.

- [ ] **Step 7: Gate e commit**

Run: `pwsh -File scripts/verify.ps1` — Expected: verde.

```bash
git add internal/boot/ cmd/gobsidian/serve.go cmd/gobsidian/serve_test.go cmd/gobsidian/ponte.go cmd/gobsidian/daemon.go CLAUDE.md docs/ARCHITECTURE.md
git commit -m "refactor(boot): host watch and shutdown steps shared by serve, bridge and daemon"
```

#### Verificações

- Step 1 FAIL→PASS colado; `mutate.ps1` exit 0 colado.
- `test_orphans.ps1` quatro `[OK]` colados.
- `go list` do Step 6 colado.
- `verify.ps1` verde.

#### Regras de execução

- Nunca `git checkout/restore/stash/clean/reset`. `git mv` por caminho é permitido.
- Orçamentos de shutdown (6 s total, 3 s in-flight, 500 ms pipe, 500 ms watcher, 2 s half-close) **não mudam**: são os medidos em `test_orphans.ps1`.
- `boot` não importa `mcpsrv`, `ipc`, `daemon`, `doctor`.
- Não rodar `test_orphans.ps1` em paralelo com qualquer medição.

#### Comando de mutação

Step 4 (`mutate.ps1`, âncora `_ = m.dst.CloseWithError(err)` em `internal/boot/espelho.go`).

#### Contrato de relatório

`task-177-report.md`: status, SHA, saídas dos Steps 1, 4, 5, 6, última linha do `verify.ps1`.

---
## Fase 5 — Contratos e fecho (Tasks 178–181)

Modelos: 178 Sonnet; 179 Sonnet; 180 Opus; 181 Sonnet. A Fase 5 muda o contrato
das tools em três pontos decididos pelo dono (`hits` some; `include` valida;
tags dobram) e fecha a dívida 1.12 com testes de caixa-branca do codec.

### Task 178: `note_metadata` recusa `include` desconhecido; `TOOLS.md` diz o que o schema serve

**Files:**
- Modify: `internal/service/graph.go:567-577` (`includeSet` via `ValidarEnum`), `:519-521` (comentário de `MetadataRequest.Include`)
- Modify: `internal/mcpsrv/tools_read.go:378-381` (tag `jsonschema` de `Include` lista os sete valores)
- Create: `internal/service/metadata_include_test.go`
- Modify: `docs/TOOLS.md:13,:62,:228-238,:481` e cada `minimum`/`maximum` (`:54,:55,:94,:96` e os demais que `grep -n 'minimum\|maximum' docs/TOOLS.md` listar)

**Interfaces:**
- Consumes: `ValidarEnum(campo, valor, padrao string, aceitos ...string) (string, error)` (`errors.go:166`; `""` devolve `padrao`); `Errorf(CodeInvalidArgument, …)`; `newTestService(t, root)` (`read_test.go:22`); a tabela de defaults da Task 168 (para o `:13`).
- Produces: `var camposDeMetadata = []string{"frontmatter", "tags", "headings", "blocks", "links", "backlinks", "inline_fields"}` em `graph.go`.

- [ ] **Step 1: Testes — falham porque hoje qualquer string é aceita**

`internal/service/metadata_include_test.go`:

```go
package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func cofreComUmaNota(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "n.md"), []byte("---\nk: v\n---\n# N\n\n#tag texto\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestNoteMetadataIncludeInvalidoEInvalidArgument(t *testing.T) {
	svc := newTestService(t, cofreComUmaNota(t))
	_, err := svc.NoteMetadata(context.Background(), MetadataRequest{
		Path: "n.md", Include: []string{"headers"},
	})
	var se *Error
	if !errors.As(err, &se) || se.Code != CodeInvalidArgument {
		t.Fatalf("include=[\"headers\"]: quero INVALID_ARGUMENT, tenho %v", err)
	}
}

func TestNoteMetadataIncludeVazioEInvalidArgument(t *testing.T) {
	svc := newTestService(t, cofreComUmaNota(t))
	_, err := svc.NoteMetadata(context.Background(), MetadataRequest{
		Path: "n.md", Include: []string{""},
	})
	var se *Error
	if !errors.As(err, &se) || se.Code != CodeInvalidArgument {
		t.Fatalf("include=[\"\"]: quero INVALID_ARGUMENT, tenho %v", err)
	}
}

func TestNoteMetadataIncludeValidoContinuaAceito(t *testing.T) {
	svc := newTestService(t, cofreComUmaNota(t))
	res, err := svc.NoteMetadata(context.Background(), MetadataRequest{
		Path: "n.md", Include: []string{"tags", "inline_fields"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Tags) != 1 {
		t.Fatalf("tags = %v, quero [tag]", res.Tags)
	}
}
```

`newTestService` está em `read_test.go`, `package service` (caixa-branca) — por isso este arquivo também é `package service`. Run: `go test ./internal/service/ -run TestNoteMetadataInclude -v` — Expected: os dois primeiros FAIL (`err == nil`), o terceiro PASS.

- [ ] **Step 2: A lista, uma vez, e o laço com `ValidarEnum`**

`internal/service/graph.go`, acima de `NoteMetadata`:

```go
// camposDeMetadata sao os valores aceitos em MetadataRequest.Include. A lista
// mora aqui, no service, porque a validacao e o schema (mcpsrv) precisam
// concordar e so um dos dois pode ser a fonte.
var camposDeMetadata = []string{"frontmatter", "tags", "headings", "blocks", "links", "backlinks", "inline_fields"}

// incluidosPorPadrao e o que note_metadata devolve quando include e omitido:
// blocks e inline_fields ficam de fora porque sao os dois campos que crescem
// com o tamanho da nota.
var incluidosPorPadrao = []string{"frontmatter", "tags", "headings", "links", "backlinks"}
```

Substituir `:567-577`:

```go
	includeSet := make(map[string]bool, len(camposDeMetadata))
	if len(req.Include) == 0 {
		for _, c := range incluidosPorPadrao {
			includeSet[c] = true
		}
	} else {
		for _, inc := range req.Include {
			v, err := ValidarEnum("include", inc, "", camposDeMetadata...)
			if err != nil {
				return MetadataResult{}, err
			}
			if v == "" {
				return MetadataResult{}, Errorf(CodeInvalidArgument,
					"include = \"\" invalido; aceitos: %s", strings.Join(camposDeMetadata, ", "))
			}
			includeSet[v] = true
		}
	}
```

`ValidarEnum` devolve o `padrao` (aqui `""`) para valor vazio sem erro — por isso o segundo teste e o `if v == ""`. Run: `go test ./internal/service/ -run 'TestNoteMetadataInclude|TestService_' -v` — Expected: PASS em todos.

`internal/mcpsrv/tools_read.go:380`: `Include []string \`json:"include,omitempty" jsonschema:"campos a devolver; aceitos: frontmatter, tags, headings, blocks, links, backlinks, inline_fields; omitido devolve frontmatter, tags, headings, links, backlinks"\``. Run: `go test ./internal/mcpsrv/ -run TestNoteMetadata_IncludeParameter` — Expected: PASS.

- [ ] **Step 3: Prova de mutação**

```powershell
pwsh -File scripts/mutate.ps1 -Path internal/service/graph.go -Anchor '"backlinks", "inline_fields"}' -Replacement '"backlinks", "inline_fields", "headers"}' -Test TestNoteMetadataIncludeInvalidoEInvalidArgument -Package ./internal/service/
```

Expected: exit 0 (com `"headers"` na lista o teste falha). Colar.

- [ ] **Step 4: Commit 1**

```bash
git add internal/service/graph.go internal/service/metadata_include_test.go internal/mcpsrv/tools_read.go
git commit -m "fix(service): note_metadata rejects unknown include values"
```

- [ ] **Step 5: Auditoria do que `TOOLS.md` promete contra o que o schema serve**

Contexto que o implementador não tem como saber sozinho: `jsonschema-go v0.4.2` (`jsonschema.For[T]`) lê a tag `jsonschema:"…"` **só como `description`**. Nenhum `enum`, `minimum`, `maximum` ou `default` é emitido; a única tool que remenda o schema à mão é `note_read` (`schemaDoNoteRead`, `alvo_note_read.go:71`). Logo toda linha de `TOOLS.md` que mostra `"minimum": 1` ou `"enum": [...]` descreve um schema que o host **não** recebe.

Montar a tabela abaixo (no relatório e, resumida, no topo da seção de contratos de `TOOLS.md`) — uma linha por propriedade numérica ou enumerada de cada tool:

| Tool | Propriedade | `TOOLS.md` diz | Código faz (clamp / rejeita / ignora / onde) | Schema servido |
|---|---|---|---|---|
| vault_search | limit | default 50, min 1, max 500 | clamp a `max_results`, `effective_limit` (`search.go:…`) | só description |
| … | … | … | … | … |

Fonte de "Código faz": `grep -n "Clamp\|clamp\|LimitePadrao\|ValidarEnum" internal/service/*.go` e a tabela de defaults da Task 168. Fonte de "TOOLS.md diz": `grep -n 'default\|minimum\|maximum\|enum' docs/TOOLS.md`.

- [ ] **Step 6: Reescrever o que estava falso**

- `TOOLS.md:13`: "O padrão de `limit` é 20 em `vault_search` e 100 nas demais listas (`LimitePadrao`); o teto é 500" — conferir os dois números na tabela da Task 168 antes de escrever; se divergirem, a tabela vence.
- `TOOLS.md:62`: "e o padrão é 50" → o valor real de `limit` de `vault_search` (20) e do teto `max_results` (o valor de `config.Defaults()`; escrever o número que `grep -n MaxResults internal/config/defaults.go` mostrar).
- `TOOLS.md:481`: "limite configurável (padrão: 200…)" → "limite fixo de 200 (`resources.go:66`); não há flag nem parâmetro que o mude".
- `TOOLS.md:228-238` (schema de `include`): trocar o bloco JSON com `enum`/`default` por prosa: "`include`: lista de strings; aceitos `frontmatter`, `tags`, `headings`, `blocks`, `links`, `backlinks`, `inline_fields`; valor fora da lista devolve `INVALID_ARGUMENT`; omitido devolve `frontmatter`, `tags`, `headings`, `links`, `backlinks`."
- Cada `"minimum"`/`"maximum"` em bloco JSON de schema: remover a chave e dizer em prosa o que o código faz — "valores acima do teto são clampados e `effective_limit` informa o valor usado" onde há clamp; "valores fora da faixa devolvem `INVALID_ARGUMENT`" onde há `ValidarEnum`/rejeição; e onde o código **ignora**, escrever isso e abrir uma linha em `docs/SUGESTOES.md` (não corrigir código nesta Task: é `docs:`).
- Parágrafo novo no topo da seção de contratos: "Os schemas servidos trazem só `type` e `description` (limitação de `jsonschema-go v0.4.2`; a exceção é `note_read`). Limites e enumerações são aplicados pelo servidor, não pelo host: um valor inválido volta como `INVALID_ARGUMENT`, um valor acima do teto volta clampado com `effective_*`."

Run: `pwsh -File scripts/verify.ps1` (inclui `check_tool_params` e `check_doc_refs`) — Expected: verde. Se `check_tool_params` reclamar de uma propriedade que a tabela mostra como "ignora", é achado real — vai para o relatório e para `SUGESTOES.md`, não se apaga a linha do doc.

- [ ] **Step 7: Commit 2**

```bash
git add docs/TOOLS.md docs/SUGESTOES.md
git commit -m "docs(tools): describe the schema the server actually serves; limits and enums are enforced server-side"
```

#### Verificações

- Step 1 FAIL→PASS colado (os dois FAIL e o PASS de controle).
- `mutate.ps1` exit 0 colado.
- Tabela do Step 5 completa no relatório, uma linha por propriedade.
- `grep -c 'minimum\|maximum' docs/TOOLS.md` antes e depois; depois deve ser `0` ou cada ocorrência restante está em prosa, não em bloco JSON — listar.
- `verify.ps1` verde depois de cada commit.

#### Regras de execução

- Nunca `git checkout/restore/stash/clean/reset`.
- Commit 1 é `fix:` (comportamento); commit 2 é `docs:` — nenhum `.go` no segundo.
- Os limites **não** entram no schema servido. Ruling do orquestrador registrado no ledger: a validação do SDK transformaria clamp em erro duro e contradiria `effective_limit`. Se o implementador discordar, escreve no relatório; não remenda `schemaDo*`.
- Não inventar número: cada default e teto escrito em `TOOLS.md` vem de um `grep` colado no relatório.

#### Comando de mutação

Step 3 (`mutate.ps1`, âncora `"backlinks", "inline_fields"}` em `internal/service/graph.go`).

#### Contrato de relatório

`task-178-report.md`: status, dois SHAs, saídas dos Steps 1 e 3, a tabela do Step 5, os `grep` que fundamentam cada número escrito, última linha do `verify.ps1`.

---

### Task 179: a — `vault_search` devolve só `results`; `hits` some

**Files:**
- Modify: `internal/service/search.go:105` (campo `Hits`), `:193-194`, `:252-253`, `:370-371` (atribuições)
- Modify: `internal/service/match_offset_test.go:35,:39,:78-81,:118,:122`, `internal/service/max_results_test.go:27-28,:38`, `internal/mcpsrv/filtro_data_test.go:85-99` (`Hits` → `Results`)
- Create: `internal/mcpsrv/search_sem_hits_test.go`
- Modify: `docs/TOOLS.md:62` ("`results` (e `hits`)" → "`results`"), `docs/ESTADO.md:139` (medição depois), `docs/SUGESTOES.md:1372` (P5 marcado feito)

**Interfaces:**
- Consumes: `sessaoComBusca(t, root)` (`filtro_data_test.go:23`) para abrir uma sessão MCP em memória.
- Produces: `SearchResponse` sem `Hits`. Nada depois desta Task depende de `hits`.

- [ ] **Step 1: O teste do contrato — falha enquanto `hits` existe**

`internal/mcpsrv/search_sem_hits_test.go`:

```go
package mcpsrv_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestVaultSearchNaoDevolveHits(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.md"), []byte("# A\n\npalavra comum\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	session, ctx := sessaoComBusca(t, root)
	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "vault_search", Arguments: map[string]any{"query": "comum"},
	})
	if err != nil || res.IsError {
		t.Fatalf("CallTool: err=%v isError=%v", err, res.IsError)
	}
	bruto, err := json.Marshal(res.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]json.RawMessage
	if err := json.Unmarshal(bruto, &out); err != nil {
		t.Fatalf("resposta ilegivel: %v\n%s", err, bruto)
	}
	if _, tem := out["hits"]; tem {
		t.Fatalf("a resposta ainda traz \"hits\":\n%s", bruto)
	}
	var results []json.RawMessage
	if err := json.Unmarshal(out["results"], &results); err != nil || len(results) != 1 {
		t.Fatalf("results = %s (err=%v), quero 1 item", out["results"], err)
	}
}
```

O import do SDK é o mesmo que `filtro_data_test.go` usa — copiar o caminho de lá. Run: `go test ./internal/mcpsrv/ -run TestVaultSearchNaoDevolveHits` — Expected: FAIL (`a resposta ainda traz "hits"`).

- [ ] **Step 2: Apagar o campo e as atribuições**

`internal/service/search.go:105`: apagar `Hits []SearchHit \`json:"hits,omitempty"\``. Em `:193-194`, `:252-253`, `:370-371`: apagar a linha `Hits: …` (ou `resp.Hits = …`), manter `Results`. `go build ./...` — Expected: os testes que leem `Hits` não compilam — é a lista do Step 3.

- [ ] **Step 3: Testes migram para `Results`**

- `match_offset_test.go:35,:39,:78-81,:118,:122`: `res.Hits` → `res.Results`.
- `max_results_test.go:27-28`: `if len(res.Results) != 5 { t.Fatalf("results = %d, quero 5", len(res.Results)) }`; `:38` idem com o número que o teste esperava para `Hits`.
- `mcpsrv/filtro_data_test.go:85-99`: apagar o campo `Hits` da struct anônima e o `if n == 0 && len(out.Results) > 0`; `n := len(out.Results)`.

Run: `go test -race ./internal/service/ ./internal/mcpsrv/` — Expected: PASS, inclusive `TestVaultSearchNaoDevolveHits`.

- [ ] **Step 4: Prova manual de mutação**

Recolocar temporariamente o campo `Hits` e **uma** atribuição (`:193`), rodar `go test ./internal/mcpsrv/ -run TestVaultSearchNaoDevolveHits` — Expected: FAIL. Remover de novo. Colar. `grep -rn '"hits"\|\.Hits\b' --include=*.go . docs/` — Expected: vazio (fora de `ESTADO.md`, que registra a história).

- [ ] **Step 5: Medição M3 depois**

`search --json --limit 200 --vault <vault_5000> --cache-dir <dir> "execucao"` com o binário deste commit: bytes do arquivo indentado (o que o comando imprime) e do compacto (`python -c "import json,sys;print(len(json.dumps(json.load(open(sys.argv[1])),separators=(',',':'))))" saida.json`). Expected: compacto ≈ 97 787 (Baseline M3 "sem `hits`"), indentado bem abaixo de 216 304. Colar os dois números em `docs/ESTADO.md:139` ao lado dos de antes.

- [ ] **Step 6: Documentação e commit**

`TOOLS.md:62`: "Objeto contendo `results`, `total`, `truncated`, …". `SUGESTOES.md:1372`: acrescentar "— feito em 2026-09 (Task 179)". `ESTADO.md:139`: os números do Step 5.

Run: `pwsh -File scripts/verify.ps1` — Expected: verde.

```bash
git add internal/service/search.go internal/service/match_offset_test.go internal/service/max_results_test.go internal/mcpsrv/filtro_data_test.go internal/mcpsrv/search_sem_hits_test.go docs/TOOLS.md docs/ESTADO.md docs/SUGESTOES.md
git commit -m "feat(service)!: vault_search returns results only

BREAKING CHANGE: the hits field, a byte-for-byte copy of results that
doubled the payload, is gone. Read results."
```

#### Verificações

- Step 1 FAIL→PASS colado; Step 4 FAIL colado; `grep` vazio colado.
- Dois números do Step 5 colados e publicados em `ESTADO.md`.
- `git log -1 --format=%B` mostra o rodapé `BREAKING CHANGE:`.
- `verify.ps1` verde.

#### Regras de execução

- Nunca `git checkout/restore/stash/clean/reset`.
- Nenhum outro campo de `SearchResponse` muda: `total`, `truncated`, `effective_limit`, `effective_snippet_chars`, `unavailable_snippets` ficam.
- Teste que lia `Hits` passa a ler `Results` com a **mesma** asserção — não afrouxar.

#### Comando de mutação

Esta tarefa não tem prova de mutação por `mutate.ps1`: a prova é a reintrodução manual do Step 4 com o FAIL colado.

#### Contrato de relatório

`task-179-report.md`: status, SHA, saídas dos Steps 1, 3, 4, 5, última linha do `verify.ps1`.

---
### Task 180: b — uma chave de tag (caixa, NFC, sem `#`), hierárquica em `note_list`, `vault_search` e `tag_list`

**Files:**
- Create: `internal/service/bench_tags_test.go` (`BenchmarkSearchFiltroTags`) — **commit 1, só teste**
- Modify: `internal/index/chave.go` (`ChaveDeTag`), `internal/index/index.go:179-180`, `internal/index/update.go:216-226,:556` (chave de `ix.tags`)
- Modify: `internal/index/query.go:173-190` (`Tags`), `:226-290` (`coletarLocked` passo 1 → `candidatosPorTagLocked`; `PathsComTags` exportado)
- Modify: `internal/service/search.go:213,:238,:441-453` (`porTag` calculado uma vez; `matchesSearchFilters` faz `BinarySearch`)
- Modify: `internal/service/graph.go` (`tagListHierarchical` da Task 169: prefixo via `ChaveDeTag`; chaves já dobradas)
- Modify: `internal/index/persist_test.go` (`TestIndiceDeMetadadosRecarregadoEIdentico` compara `Tags()`)
- Create: `internal/index/tag_chave_test.go`, `internal/service/tags_contrato_test.go`
- Modify: `testdata/tag_list_hierarquico.json` (golden da Task 169, regenerado), `docs/TOOLS.md:50,:201-202` e seção de `tag_list`, `docs/ESTADO.md`

**Interfaces:**
- Consumes: `text.ParaNFC` (o `index` já importa `text`); `aliasKey` como modelo (`chave.go:50`); `TagNode.Children []TagNode` e `tagListHierarchical` (Task 169); `porFrontmatter`/`casamFrontmatter` como modelo do filtro resolvido uma vez (`search.go:213,:488`).
- Produces:

```go
// ChaveDeTag e a UNICA conta da chave de tag: sem '#', NFC, minuscula.
// Exportada porque service compara tags e nao importa text (folha do grafo:
// service -> index, nao service -> text).
func ChaveDeTag(tag string) string

// PathsComTags devolve, ordenados, os caminhos das notas que casam as tags
// em mode ("all" | "any"), com a regra hierarquica: a tag pedida casa a si
// mesma e qualquer subtag ("projeto" casa "projeto/x"). Nil para len(tags)==0.
func (ix *Index) PathsComTags(tags []string, mode string) []vault.CanonicalPath
```

Regra do contrato (para `TOOLS.md`): em `note_list.tags`, `vault_search.tags` e `tag_list.prefix`, a tag pedida casa a si mesma e suas subtags; `#` inicial é opcional; comparação insensível a caixa e a forma Unicode (NFC). `tag_list` devolve a forma **dobrada** (`#Ação` e `#ação` viram uma entrada `ação` com a soma das contagens). `note_metadata.tags` continua devolvendo a grafia original da nota.

- [ ] **Step 1: Commit 1 — o bench que vai ser comparado**

`internal/service/bench_tags_test.go`:

```go
package service_test

import (
	"testing"

	"github.com/jonyd/gobsidian/internal/service"
)

// BenchmarkSearchFiltroTags mede a busca com filtro de tag: hoje o filtro
// baixa a caixa de cada tag de cada resultado por consulta; depois da
// Task 180 resolve o conjunto uma vez e faz busca binaria por resultado.
func BenchmarkSearchFiltroTags(b *testing.B) {
	svc := benchServicoDeCache(b)
	benchBusca(b, svc, service.SearchOptions{
		Query: "nota",
		Limit: 200,
		Tags:  []string{"golang"},
	}, 1)
}
```

O último argumento de `benchBusca` é o mínimo de resultados esperado (ver `BenchmarkSearchFiltroFrontmatter`, `bench_cache_test.go:100`); se a assinatura real for outra, seguir a real. Run: `go test ./internal/service/ -run xxx -bench SearchFiltroTags -benchtime 3x` — Expected: roda.

```bash
git add internal/service/bench_tags_test.go
git commit -m "test(service): benchmark vault_search with a tags filter"
```

Depois do commit: `go test -c -o %LOCALAPPDATA%\gobsidian-bench\2026-09-02\antes180_service.test.exe ./internal/service/` e `go test -c -o ...\antes180_index.test.exe ./internal/index/` — são os binários "antes" desta Task (os da Baseline não têm este bench).

- [ ] **Step 2: Testes do índice — falham por compilação e por comportamento**

`internal/index/tag_chave_test.go` (pacote `index`):

```go
package index

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/jonyd/gobsidian/internal/vault"
)

func TestChaveDeTagDobraCaixaHashENFC(t *testing.T) {
	// "Ação" em NFD: A + c + cedilha combinante + a + til combinante + o
	nfd := "Ação"
	casos := map[string]string{
		"#Projeto":    "projeto",
		"Projeto/Sub": "projeto/sub",
		"#" + nfd:     "ação",
		"ação":        "ação",
	}
	for in, quer := range casos {
		if got := ChaveDeTag(in); got != quer {
			t.Errorf("ChaveDeTag(%q) = %q, quero %q", in, got, quer)
		}
	}
}

func TestListPorTagCasaSubtagENFD(t *testing.T) {
	root := t.TempDir()
	escreve := func(nome, corpo string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, nome), []byte(corpo), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	escreve("a.md", "# A\n\n#Projeto/Alpha\n")
	escreve("b.md", "# B\n\n#projeto\n")
	escreve("c.md", "# C\n\n#outra\n")
	escreve("d.md", "# D\n\n#Ação\n") // NFD
	v, err := vault.New(root)
	if err != nil {
		t.Fatal(err)
	}
	ix := New()
	if err := ix.Build(context.Background(), v); err != nil {
		t.Fatal(err)
	}
	got := ix.PathsComTags([]string{"#PROJETO"}, "all")
	if quer := []vault.CanonicalPath{"a.md", "b.md"}; !slices.Equal(got, quer) {
		t.Fatalf("PathsComTags(#PROJETO) = %v, quero %v", got, quer)
	}
	got = ix.PathsComTags([]string{"ação"}, "all") // pedido em NFC, nota em NFD
	if quer := []vault.CanonicalPath{"d.md"}; !slices.Equal(got, quer) {
		t.Fatalf("PathsComTags(ação) = %v, quero %v", got, quer)
	}
	got = ix.PathsComTags([]string{"projeto/alpha", "outra"}, "any")
	if quer := []vault.CanonicalPath{"a.md", "c.md"}; !slices.Equal(got, quer) {
		t.Fatalf("PathsComTags(any) = %v, quero %v", got, quer)
	}
	if got := ix.PathsComTags([]string{"projeto/alpha", "outra"}, "all"); len(got) != 0 {
		t.Fatalf("PathsComTags(all, disjuntas) = %v, quero vazio", got)
	}
	tags := ix.Tags("proj", 0)
	if len(tags) != 2 || tags[0].Tag != "projeto" && tags[1].Tag != "projeto" {
		t.Fatalf("Tags(proj) = %v, quero projeto e projeto/alpha dobradas", tags)
	}
}
```

Run: `go test ./internal/index/ -run 'TestChaveDeTag|TestListPorTag'` — Expected: FAIL de compilação (`ChaveDeTag`, `PathsComTags` não existem).

- [ ] **Step 3: A chave, os três pontos de escrita, e `Tags`**

`internal/index/chave.go`, junto de `aliasKey`:

```go
// ChaveDeTag e a chave de ix.tags e a forma que tag_list devolve. Tres pontos
// escreviam a chave crua (boot, remocao, rename) e tres leitores baixavam a
// caixa cada um do seu jeito: Tags so ToLower no prefixo, coletarLocked ToLower
// nos dois lados a cada comparacao, service TrimPrefix('#') + ToLower. Nenhum
// deles normalizava Unicode, e "#Ação" digitado num Mac (NFD) nao casava o
// mesmo "#Ação" digitado no Windows (NFC). Uma conta, aqui.
func ChaveDeTag(tag string) string {
	return strings.ToLower(text.ParaNFC(strings.TrimPrefix(tag, "#")))
}
```

`index.go:179-180`: `k := ChaveDeTag(t); ix.tags[k] = append(ix.tags[k], n.Path)`. `update.go:216-226`: `paths := ix.tags[ChaveDeTag(tag)]` e `delete(ix.tags, ChaveDeTag(tag))` / `ix.tags[ChaveDeTag(tag)] = filtered`. `update.go:556`: `paths := ix.tags[ChaveDeTag(tag)]`. `Note.Tags` continua com a grafia original — só a chave do mapa dobra. Como o mesmo caminho pode entrar duas vezes na lista quando a nota tem `#Ação` e `#ação` (o parser mantém as duas grafias), `publishNoteLocked` deduplica: `if len(ix.tags[k]) == 0 || ix.tags[k][len(ix.tags[k])-1] != n.Path`.

`Tags` (`query.go:173-190`): `prefix = ChaveDeTag(prefix)` e `strings.HasPrefix(t, prefix)` direto (a chave já está dobrada); `TagCount{Tag: t, …}` devolve a chave.

- [ ] **Step 4: `PathsComTags` — a conta única do casamento**

Extrair o passo 1 de `coletarLocked` (`query.go:232-284`) para:

```go
// candidatosPorTagLocked e o passo 1 de coletarLocked e o corpo de
// PathsComTags. Exige ix.mu ja travado para leitura.
func (ix *Index) candidatosPorTagLocked(tags []string, mode string) []vault.CanonicalPath {
	casam := func(pedida string) []vault.CanonicalPath {
		tk := ChaveDeTag(pedida)
		var m []vault.CanonicalPath
		for k, v := range ix.tags {
			if k == tk || strings.HasPrefix(k, tk+"/") {
				m = append(m, v...)
			}
		}
		slices.Sort(m)
		return slices.Compact(m)
	}
	if strings.ToLower(mode) == "any" {
		var todos []vault.CanonicalPath
		for _, t := range tags {
			todos = append(todos, casam(t)...)
		}
		slices.Sort(todos)
		return slices.Compact(todos)
	}
	var cand []vault.CanonicalPath
	for i, t := range tags {
		m := casam(t)
		if i == 0 {
			cand = m
			continue
		}
		cand = slices.DeleteFunc(cand, func(c vault.CanonicalPath) bool {
			_, ok := slices.BinarySearch(m, c)
			return !ok
		})
		if len(cand) == 0 {
			break
		}
	}
	return cand
}

func (ix *Index) PathsComTags(tags []string, mode string) []vault.CanonicalPath {
	if len(tags) == 0 {
		return nil
	}
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	return ix.candidatosPorTagLocked(tags, mode)
}
```

Em `coletarLocked`, o passo 1 vira `if len(q.Tags) > 0 { candidates = ix.candidatosPorTagLocked(q.Tags, q.TagMode) } else { … }` — o `tagMode == ""` → `"all"` fica coberto pelo `!= "any"`. A linha `if k == tk || strings.HasPrefix(k, tk+"/") {` fica **exatamente** assim: âncora de mutação.

Run: `go test -race ./internal/index/` — Expected: PASS, inclusive os do Step 2.

- [ ] **Step 5: `service` — `vault_search` e `tag_list`**

`internal/service/search.go`: junto de `porFrontmatter := s.casamFrontmatter(opts)` (`:213`), `porTag := s.index.PathsComTags(opts.Tags, "all")`; `matchesSearchFilters(note, opts, porFrontmatter, porTag)`; dentro (`:441-453`), o bloco de tags vira:

```go
	// porTag e nil quando nao ha filtro de tag; com filtro, e o conjunto
	// ordenado que o indice resolveu UMA vez (index.PathsComTags) — a mesma
	// conta que note_list usa, hierarquia e dobra de caixa incluidas.
	if len(opts.Tags) > 0 {
		if _, ok := slices.BinarySearch(porTag, note.Path); !ok {
			return false
		}
	}
```

`internal/service/graph.go`, `tagListHierarchical` (forma da Task 169): o filtro de prefixo usa `index.ChaveDeTag(req.Prefix)`; as chaves de `s.index.Tags("", 0)` já vêm dobradas, então qualquer `strings.ToLower` restante ali sai. Testes de contrato em `internal/service/tags_contrato_test.go` (pacote `service`, usa `newTestService`):

```go
package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func cofreComTags(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	escreve := func(nome, corpo string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, nome), []byte(corpo), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	escreve("a.md", "# A\n\nnota comum #Projeto/Alpha\n")
	escreve("b.md", "# B\n\nnota comum #projeto\n")
	escreve("c.md", "# C\n\nnota comum #outra\n")
	escreve("d.md", "# D\n\nnota comum #Ação #ação\n") // NFD e NFC
	return root
}

func TestVaultSearchTagsCasaSubtag(t *testing.T) {
	svc := newTestService(t, cofreComTags(t))
	res, err := svc.Search(context.Background(), SearchOptions{Query: "comum", Tags: []string{"#PROJETO"}, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Results) != 2 {
		t.Fatalf("tags=[#PROJETO]: %d resultados, quero 2 (projeto e projeto/alpha): %+v", len(res.Results), res.Results)
	}
}

func TestVaultSearchTagsNFD(t *testing.T) {
	svc := newTestService(t, cofreComTags(t))
	res, err := svc.Search(context.Background(), SearchOptions{Query: "comum", Tags: []string{"ação"}, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Results) != 1 || res.Results[0].Path != "d.md" {
		t.Fatalf("tags=[ação]: %+v, quero so d.md", res.Results)
	}
}

func TestTagListDevolveFormaDobrada(t *testing.T) {
	svc := newTestService(t, cofreComTags(t))
	res, err := svc.TagList(context.Background(), TagListRequest{Prefix: "aç"})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Tags) != 1 || res.Tags[0].Tag != "ação" || res.Tags[0].Count != 1 {
		t.Fatalf("tag_list(aç) = %+v, quero uma entrada ação com count 1 (mesma nota, duas grafias)", res.Tags)
	}
}
```

Os nomes `TagListRequest`, `res.Tags`, `.Tag`, `.Count` são os reais de `graph.go` (conferir; a Task 169 pode ter mudado o tipo do item). `Count` é 1 porque as duas grafias estão na **mesma** nota e a lista de caminhos deduplica; se quiser 2, seriam duas notas.

Run: `go test -race ./internal/service/ ./internal/mcpsrv/` — Expected: PASS nos novos; `TestTagList_Hierarquico` (Task 169) **falha** no golden — é o Step 6.

- [ ] **Step 6: Golden da Task 169 e o `Tags()` no reload do cache**

Regenerar `testdata/tag_list_hierarquico.json` pelo mecanismo da Task 169 (`-update` ou equivalente que ela criou). No relatório, o `git diff` do golden com a explicação linha a linha: o que mudou é caixa/`#`/NFC das chaves e eventuais fusões de entradas — nenhuma contagem pode **cair** sem uma fusão que a explique.

`internal/index/persist_test.go`, `TestIndiceDeMetadadosRecarregadoEIdentico`: acrescentar `if !slices.Equal(recarregado.Tags("", 0), original.Tags("", 0)) { t.Fatalf(...) }` (os nomes das duas variáveis são os do teste). O cache guarda `Note.Tags` cru e o reload passa por `publishNoteLocked`, então a chave dobrada é reconstruída — e o formato do cache **não muda**. Se o teste falhar, a dobra não está no ponto único de publicação.

Run: `go test -race ./internal/index/ ./internal/service/ ./internal/mcpsrv/` — Expected: PASS.

- [ ] **Step 7: Provas de mutação**

```powershell
pwsh -File scripts/mutate.ps1 -Path internal/index/chave.go -Anchor 'strings.TrimPrefix(tag, "#")' -Replacement 'tag' -Test TestChaveDeTagDobraCaixaHashENFC -Package ./internal/index/
pwsh -File scripts/mutate.ps1 -Path internal/index/query.go -Anchor 'if k == tk || strings.HasPrefix(k, tk+"/") {' -Replacement 'if k == tk {' -Test TestListPorTagCasaSubtagENFD -Package ./internal/index/
```

Expected: exit 0 nos dois. Colar.

- [ ] **Step 8: `benchstat`**

`antes180_index` × depois: `TagsSemPrefixo`, `ListPorTag`. `antes180_service` × depois: `NoteListPorTag`, `TagListPlano`, `TagListHierarquico`, `SearchFiltroFrontmatter`, `SearchFiltroTags`. 7 intercalados, `-benchmem`. Expected: `SearchFiltroTags` e `ListPorTag` **melhoram** ou `~` (o `ToLower` por comparação sumiu); os demais `~`. Piora com `p < 0.05` em qualquer um: investigar antes de commitar — o suspeito é a deduplicação em `publishNoteLocked`. Colar; publicar em `docs/ESTADO.md`.

- [ ] **Step 9: Documentação e commit 2**

`docs/TOOLS.md:50` (`note_list.tags`), `:201-202` (`vault_search.tags`) e a seção de `tag_list`: a regra do contrato (bloco **Interfaces** acima), palavra por palavra nas três. `docs/ESTADO.md`: nota "chave de tag única desde 2026-09 (Task 180); formato de cache inalterado" e o `benchstat`.

Run: `pwsh -File scripts/verify.ps1` — Expected: verde.

```bash
git add internal/index/chave.go internal/index/index.go internal/index/update.go internal/index/query.go internal/index/tag_chave_test.go internal/index/persist_test.go internal/service/search.go internal/service/graph.go internal/service/tags_contrato_test.go testdata/tag_list_hierarquico.json docs/TOOLS.md docs/ESTADO.md
git commit -m "feat(index)!: one tag key - case, NFC, no hash - hierarchical in note_list, vault_search and tag_list

BREAKING CHANGE: vault_search.tags now matches subtags and ignores case,
Unicode form and a leading '#', like note_list already did; tag_list
returns the folded form (lowercase, NFC, no '#') and merges spellings
that differ only in case or Unicode form."
```

#### Verificações

- Step 2 FAIL de compilação colado; PASS dos Steps 4, 5, 6 colados.
- Dois `mutate.ps1` com exit 0 colados.
- `git diff` do golden com explicação por linha.
- `benchstat` colado e publicado.
- `grep -rn "strings.ToLower(t)\|strings.ToLower(k)\|ToLower(reqTag)" internal/index/query.go internal/service/search.go internal/service/graph.go` — Expected: vazio (nenhuma dobra fora de `ChaveDeTag`).
- `verify.ps1` verde após cada commit.

#### Regras de execução

- Nunca `git checkout/restore/stash/clean/reset`.
- `service` **não** importa `text`; a dobra que ele precisa vem por `index.ChaveDeTag`.
- `Note.Tags` e `note_metadata.tags` mantêm a grafia original; `IndexCacheFormatVersion` **não** muda.
- Commit 1 (bench) antes de qualquer alteração de produto; os binários `antes180_*` são construídos desse commit.
- Toda comparação de tag no produto passa por `ChaveDeTag` ou por `PathsComTags` — inclusive as que já pareciam certas.

#### Comando de mutação

Step 7 (dois `mutate.ps1`: âncora `strings.TrimPrefix(tag, "#")` em `chave.go`; âncora `if k == tk || strings.HasPrefix(k, tk+"/") {` em `query.go`).

#### Contrato de relatório

`task-180-report.md`: status, dois SHAs, saídas dos Steps 2, 4–8, o `git diff` do golden explicado, o `grep` das Verificações, última linha do `verify.ps1`.

---

### Task 181: c2 — testes de caixa-branca do codec do cache de metadados

**Files:**
- Create: `internal/index/persist_codec_test.go` (pacote `index`)
- Modify: `docs/ESTADO.md` (dívida 1.12 fechada; cobertura de `persist_codec.go` antes/depois)

**Interfaces:**
- Consumes (todos não exportados, `internal/index/persist_codec.go`): tags `valNil, valBool, valInt, valInt64, valUint64, valFloat64, valString, valTime, valSliceNil, valSlice, valMapNil, valMap`; limites `limiteString`, `limiteValorProfund`; `escritor{w io.Writer}` com métodos `value(any)`, `note(*Note)`, `str`, `strSlice`, `timeBlob`; `leitor{b []byte}` com `value(profundidade int)`, `note()`, `strSlice()`, `timeBlob()` e campo `err`; `ErrIndexCacheCorrupted`. Nomes e aridades exatas: `gopls` sobre `persist_codec.go` — os esboços abaixo usam os nomes do arquivo em 2026-09-02; se um método receber argumento a mais (`leitor.str(oque string)`), passar a string.
- Produces: nada — fecha a dívida 1.12 de `ESTADO.md`.

- [ ] **Step 1: Cobertura antes**

Run: `go test ./internal/index/ -coverprofile=%TEMP%\cov_antes.out && go tool cover -func=%TEMP%\cov_antes.out | grep persist_codec.go` — colar todas as linhas (uma por função).

- [ ] **Step 2: Os testes**

`internal/index/persist_codec_test.go`:

```go
package index

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func roundTrip(t *testing.T, v any) any {
	t.Helper()
	var buf bytes.Buffer
	e := &escritor{w: &buf}
	e.value(v)
	if e.err != nil {
		t.Fatalf("escrevendo %#v: %v", v, e.err)
	}
	l := &leitor{b: buf.Bytes()}
	got := l.value(0)
	if l.err != nil {
		t.Fatalf("lendo %#v: %v", v, l.err)
	}
	if l.i != len(l.b) {
		t.Fatalf("sobraram %d bytes depois de ler %#v", len(l.b)-l.i, v)
	}
	return got
}

func TestCodecValorRoundTripPorTipo(t *testing.T) {
	zona := time.FixedZone("X", -3*3600)
	casos := []any{
		nil, true, false, int(-7), int64(1 << 40), uint64(1<<63 + 5), float64(2.5),
		"ação", "", time.Date(2026, 9, 2, 10, 0, 0, 123, zona),
		[]any{int(1), "dois", nil}, []any{}, map[string]any{"k": "v", "n": int(3)}, map[string]any{},
		[]any{map[string]any{"a": []any{int(1)}}},
	}
	for _, in := range casos {
		got := roundTrip(t, in)
		if in == nil {
			if got != nil {
				t.Errorf("nil virou %#v", got)
			}
			continue
		}
		if reflect.TypeOf(got) != reflect.TypeOf(in) {
			t.Errorf("%#v (%T) voltou como %T", in, in, got)
			continue
		}
		if tm, ok := in.(time.Time); ok {
			if !tm.Equal(got.(time.Time)) {
				t.Errorf("time %v voltou %v", tm, got)
			}
			continue
		}
		if !reflect.DeepEqual(got, in) {
			t.Errorf("%#v voltou %#v", in, got)
		}
	}
}

func TestCodecStrSliceNilDistintoDeVazio(t *testing.T) {
	for _, in := range [][]string{nil, {}} {
		var buf bytes.Buffer
		e := &escritor{w: &buf}
		e.strSlice(in)
		l := &leitor{b: buf.Bytes()}
		got := l.strSlice()
		if l.err != nil {
			t.Fatal(l.err)
		}
		if (got == nil) != (in == nil) {
			t.Errorf("strSlice %#v voltou %#v (nil-ness diferente)", in, got)
		}
	}
}

func TestCodecTimeBlobPreservaZona(t *testing.T) {
	in := time.Date(2026, 9, 2, 10, 0, 0, 7, time.FixedZone("Y", 5*3600+1800))
	var buf bytes.Buffer
	e := &escritor{w: &buf}
	e.timeBlob(in)
	l := &leitor{b: buf.Bytes()}
	got := l.timeBlob()
	if l.err != nil {
		t.Fatal(l.err)
	}
	if !got.Equal(in) {
		t.Fatalf("%v voltou %v", in, got)
	}
	_, off1 := in.Zone()
	_, off2 := got.Zone()
	if off1 != off2 {
		t.Fatalf("offset de zona %d voltou %d", off1, off2)
	}
}

func TestCodecValorTipoNaoSuportadoFalha(t *testing.T) {
	var buf bytes.Buffer
	e := &escritor{w: &buf}
	e.value(struct{}{})
	if e.err == nil || !strings.Contains(e.err.Error(), "tipo nao suportado") {
		t.Fatalf("struct{}{} devia falhar com \"tipo nao suportado\", tenho %v", e.err)
	}
}

func TestCodecValorProfundidadeAlemDoLimiteERecusada(t *testing.T) {
	var v any = "folha"
	for i := 0; i < limiteValorProfund+6; i++ {
		v = []any{v}
	}
	var buf bytes.Buffer
	e := &escritor{w: &buf}
	e.value(v)
	if e.err == nil {
		// O escritor pode nao limitar profundidade; entao o leitor tem de limitar.
		l := &leitor{b: buf.Bytes()}
		_ = l.value(0)
		if l.err == nil || !errors.Is(l.err, ErrIndexCacheCorrupted) || !strings.Contains(l.err.Error(), "profundidade") {
			t.Fatalf("profundidade %d devia ser recusada na leitura, tenho %v", limiteValorProfund+6, l.err)
		}
		return
	}
	if !strings.Contains(e.err.Error(), "profundidade") {
		t.Fatalf("escritor recusou por outro motivo: %v", e.err)
	}
}

func TestCodecTagDesconhecidaERecusada(t *testing.T) {
	l := &leitor{b: []byte{valMap + 1}}
	_ = l.value(0)
	if l.err == nil || !errors.Is(l.err, ErrIndexCacheCorrupted) || !strings.Contains(l.err.Error(), "tag de valor desconhecida") {
		t.Fatalf("tag %d devia ser recusada, tenho %v", valMap+1, l.err)
	}
}

func TestCodecStringAcimaDoLimiteERecusada(t *testing.T) {
	var buf bytes.Buffer
	e := &escritor{w: &buf}
	e.uvarint(uint64(limiteString + 1))
	l := &leitor{b: buf.Bytes()}
	_ = l.str("teste")
	if l.err == nil || !strings.Contains(l.err.Error(), "acima do limite") {
		t.Fatalf("string de %d bytes devia ser recusada, tenho %v", limiteString+1, l.err)
	}
}

func TestCodecNotaTruncadaEmCadaByteERecusada(t *testing.T) {
	n := &Note{
		Path: "pasta/nota.md", Title: "Nota", Hash: "abc",
		Tags: []string{"a", "b/c"}, Aliases: []string{"x"},
		Frontmatter: map[string]any{"k": "v", "n": int64(2), "lista": []any{"a", int(1)}},
		ModTime: time.Now(),
	}
	var buf bytes.Buffer
	e := &escritor{w: &buf}
	e.note(n)
	if e.err != nil {
		t.Fatal(e.err)
	}
	b := buf.Bytes()
	for i := 0; i < len(b); i++ {
		l := &leitor{b: b[:i]}
		_ = l.note()
		if l.err == nil {
			t.Fatalf("prefixo de %d/%d bytes foi aceito como nota completa", i, len(b))
		}
	}
	l := &leitor{b: b}
	got := l.note()
	if l.err != nil || got == nil || got.Path != n.Path {
		t.Fatalf("a nota inteira devia ler: err=%v got=%+v", l.err, got)
	}
}
```

Os campos de `Note` usados (`Path`, `Title`, `Hash`, `Tags`, `Aliases`, `Frontmatter`, `ModTime`) são os de `note.go`; preencher também `Headings`, `Blocks`, `Links`, `InlineFields` com um elemento cada, para que o truncamento percorra os quatro sub-codecs — os tipos exatos estão em `note.go` (`gopls`). Se `escritor.value` não devolver erro para profundidade (só o leitor limita), o teste de profundidade já cobre os dois caminhos.

Run: `go test -race ./internal/index/ -run TestCodec -v` — Expected: PASS em todos. Um FAIL aqui é achado real (o codec aceita o que não devia) — reportar como `DONE_WITH_CONCERNS` com o teste **mantido** e a linha do codec apontada; não afrouxar o teste.

- [ ] **Step 3: Provas de mutação**

```powershell
pwsh -File scripts/mutate.ps1 -Path internal/index/persist_codec.go -Anchor 'if profundidade > limiteValorProfund {' -Replacement 'if false {' -Test TestCodecValorProfundidadeAlemDoLimiteERecusada -Package ./internal/index/
pwsh -File scripts/mutate.ps1 -Path internal/index/persist_codec.go -Anchor 'case valInt64:' -Replacement 'case valInt64 + 100:' -Test TestCodecValorRoundTripPorTipo -Package ./internal/index/
```

Expected: exit 0 nos dois. A grafia exata das âncoras é a do arquivo (`grep -n "limiteValorProfund\|case valInt64" internal/index/persist_codec.go`); ajustar a âncora ao texto real, não a mutação.

- [ ] **Step 4: Cobertura depois e `ESTADO.md`**

Run: `go test ./internal/index/ -coverprofile=%TEMP%\cov_depois.out && go tool cover -func=%TEMP%\cov_depois.out | grep persist_codec.go` — colar. `docs/ESTADO.md`: na dívida 1.12, "fechada em 2026-09 (Task 181): testes de caixa-branca em `persist_codec_test.go`; cobertura de `persist_codec.go` X % → Y %" com os dois números **medidos** no Step 1 e aqui (o total do arquivo é a média ponderada que `go tool cover -func` não dá direto — publicar por função, as linhas coladas, e não inventar uma média).

- [ ] **Step 5: Gate e commit**

Run: `pwsh -File scripts/verify.ps1` — Expected: verde.

```bash
git add internal/index/persist_codec_test.go docs/ESTADO.md
git commit -m "test(index): white-box tests for the metadata cache codec"
```

#### Verificações

- Cobertura antes (Step 1) e depois (Step 4) coladas, por função.
- PASS do Step 2 colado; dois `mutate.ps1` exit 0 colados.
- Nenhum arquivo de produto no commit: `git show --stat HEAD` só com `persist_codec_test.go` e `ESTADO.md`.
- `verify.ps1` verde.

#### Regras de execução

- Nunca `git checkout/restore/stash/clean/reset`.
- Só teste e doc: se um teste revelar defeito no codec, o defeito vai para o relatório e para `SUGESTOES.md`; corrigir é outra Task.
- Nenhum número de cobertura sem a linha do `go tool cover` colada ao lado.

#### Comando de mutação

Step 3 (dois `mutate.ps1` sobre `internal/index/persist_codec.go`).

#### Contrato de relatório

`task-181-report.md`: status, SHA, cobertura antes/depois, saída dos Steps 2 e 3, `git show --stat HEAD`, última linha do `verify.ps1`.

---
# Task 000 — sentinela

Não é tarefa. Existe para que o extrator de briefs pare aqui em vez de vazar até o fim do arquivo. Mova-a para depois da última Task ao acrescentar tarefas.
