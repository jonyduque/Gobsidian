# Task 171 — relatorio

## Progresso

- 06:54 — inicio; brief, `CLAUDE.md`, `implementador.md` e `golang-refactoring` lidos; base `8a70044` confirmada.
- 06:57 — Steps 1-3: seis arquivos movidos com `git mv`, pacotes trocados, `ReplaceFile` extraida com o callback, `WriteAtomic` virou o wrapper, `writer/atomic.go` reescrito so com os quatro encaminhadores `Deprecated`. `go build ./...` limpo.
- 06:58 — Steps 4-5: `walk.go` usa `TempFilePrefix`, comentario de `longpath_windows.go` corrigido, dois testes novos escritos; as tres provas de mutacao sairam 0 (coladas abaixo).
- 07:00 — `go test -race -count=3 ./internal/vault/ ./internal/writer/` verde.
- 07:05 — Step 6: `verify.ps1` **REPROVOU** em 2 das 14 etapas (`golangci-lint` Windows e Linux), com SA1019 nos chamadores que a PR1 nao pode tocar. Docs e `CLAUDE.md` ja editados e validados em UTF-8.
- 07:10 — BLOCKED enviado ao orquestrador com as quatro opcoes e a recomendacao. Nada commitado; tudo no working tree, com os caminhos explicitos ja em stage.
- 07:12 — `audit_reports.ps1 171`: dois achados neste relatorio (secoes de TDD), respondidos na secao "TDD — RED e GREEN"; os outros 14 sao do ledger antigo e nao vem desta tarefa. Relatorio validado em UTF-8.
- 07:20 — Ruling do orquestrador (opcao 1) aplicado: os quatro encaminhadores perderam o marcador `// Deprecated:` e ganharam a prosa "Encaminhador transitorio para vault.X; a Task 172 migra os chamadores e o remove". `ESTRUTURA.md` e `ARCHITECTURE.md` acompanharam. Build limpo, `gofmt` limpo, UTF-8 validado.
- 07:25 — `verify.ps1` **VERDE nas 14 etapas**: `[OK] Bateria completa. Pode commitar.`
- 07:25 — commit `4d7f397` com os 13 caminhos explicitos, via `git commit -F`. Nada mais entrou: o `progress.md` modificado do dono e os 100+ nao rastreados ficaram fora.

## Status

**DONE** — `4d7f397`, com `verify.ps1` verde nas 14 etapas.

Steps 1 a 6 completos. Houve um bloqueio no Step 6, resolvido por ruling do
orquestrador; o registro abaixo fica porque a proxima pessoa que marcar um
encaminhador como `Deprecated` neste repositorio vai bater na mesma parede.

### Desvio do brief, por ruling do orquestrador

O brief especifica, na secao **Interfaces**, o marcador
`// Deprecated: use vault.X. Removido na Task 172.` em cada encaminhador. **Esse
marcador nao foi usado.** No lugar dele, cada encaminhador leva a prosa
`// Encaminhador transitorio para vault.X; a Task 172 migra os chamadores e o
remove.`, com a wording ditada pelo orquestrador.

O motivo esta abaixo. As outras tres saidas consideradas — fundir 171 com 172,
excluir SA1019 no `.golangci.yml`, ou `//nolint` nos seis chamadores — foram
recusadas: a primeira destroi a divisibilidade da PR, e as duas ultimas exigem
mexer em arquivo que a PR1 nao pode tocar. Os chamadores continuam intactos,
`.golangci.yml` continua intacto, e nao ha nenhum `//nolint` novo.

### O bloqueio que originou o desvio

Antes do ruling, `verify.ps1` ficava 12/14. As etapas 9 e 10 (`golangci-lint`,
Windows e Linux) reprovavam com 5 achados SA1019, todos nos arquivos de chamador
que a PR1 nao pode editar:

```
internal\service\write.go:161:12: SA1019: writer.WriteAtomic is deprecated: use vault.WriteAtomic. Removido na Task 172. (staticcheck)
	if err := writer.WriteAtomic(ctx, absPath, []byte(fullContent)); err != nil {
	          ^
internal\service\write.go:260:12: SA1019: writer.WriteAtomic is deprecated: use vault.WriteAtomic. Removido na Task 172. (staticcheck)
internal\service\write.go:389:12: SA1019: writer.WriteAtomic is deprecated: use vault.WriteAtomic. Removido na Task 172. (staticcheck)
cmd\gobsidian\servico.go:73:7: SA1019: writer.SweepResult is deprecated: use vault.SweepResult. Removido na Task 172. (staticcheck)
cmd\gobsidian\servico.go:78:13: SA1019: writer.SweepStaleTempFiles is deprecated: use vault.SweepStaleTempFiles. Removido na Task 172. (staticcheck)
5 issues:
* staticcheck: 5
```

`write.go:640` e `write.go:846` nao aparecem porque o `max-same-issues` padrao do
golangci-lint corta a terceira repeticao da mesma mensagem — nao porque estejam
limpos.

Era estrutural, nao defeito do codigo: `staticcheck` esta ligado em
`.golangci.yml`, entao "marcar o encaminhador como `Deprecated`" e "deixar os
chamadores para a Task 172" nao podiam valer ao mesmo tempo com o gate verde. O
brief exigia os dois. O proprio `writer/atomic.go` carrega hoje um comentario
explicando por que os marcadores nao estao la, para que ninguem os "conserte" de
volta.

## `git diff --stat -M 8a70044..HEAD`

```
 CLAUDE.md                                          |   7 +-
 docs/ARCHITECTURE.md                               |   6 +-
 docs/ESTRUTURA.md                                  |  10 +-
 internal/vault/atomic.go                           | 226 +++++++++++++++++++++
 internal/{writer => vault}/atomic_test.go          |  66 ++++--
 internal/{writer => vault}/durabilidade_test.go    |  10 +-
 internal/vault/longpath_windows.go                 |   2 +-
 .../sweep_profundo_windows_test.go                 |   8 +-
 internal/{writer => vault}/syncdir_unix.go         |   2 +-
 internal/{writer => vault}/syncdir_windows.go      |   2 +-
 internal/vault/walk.go                             |   2 +-
 internal/vault/walk_test.go                        |  33 +++
 internal/writer/atomic.go                          | 225 ++++----------------
 13 files changed, 376 insertions(+), 223 deletions(-)
```

O proprio `git commit` reportou cinco dos seis como rename, com a similaridade:

```
 13 files changed, 376 insertions(+), 223 deletions(-)
 create mode 100644 internal/vault/atomic.go
 rename internal/{writer => vault}/atomic_test.go (84%)
 rename internal/{writer => vault}/durabilidade_test.go (88%)
 rename internal/{writer => vault}/sweep_profundo_windows_test.go (92%)
 rename internal/{writer => vault}/syncdir_unix.go (97%)
 rename internal/{writer => vault}/syncdir_windows.go (98%)
```

Cinco dos seis aparecem como rename. `atomic.go` **nao pode** aparecer com `-M`
sozinho, e isso e consequencia do desenho da tarefa, nao de um `git mv` mal
feito: o Step 3 recria `internal/writer/atomic.go` com os encaminhadores, entao
o caminho de origem continua existindo e o `-M` nao tem um delete para parear.
Com deteccao de copia os seis aparecem, e o delta de 35 linhas do `atomic.go` e
exatamente a extracao de `ReplaceFile`:

```
$ git diff --stat -M -C --find-copies-harder 8a70044..HEAD
 CLAUDE.md                                          |   7 +-
 docs/ARCHITECTURE.md                               |   6 +-
 docs/ESTRUTURA.md                                  |  10 +-
 internal/{writer => vault}/atomic.go               |  35 +++-
 internal/{writer => vault}/atomic_test.go          |  66 ++++--
 internal/{writer => vault}/durabilidade_test.go    |  10 +-
 internal/vault/longpath_windows.go                 |   2 +-
 .../sweep_profundo_windows_test.go                 |   8 +-
 internal/{writer => vault}/syncdir_unix.go         |   2 +-
 internal/{writer => vault}/syncdir_windows.go      |   2 +-
 internal/vault/walk.go                             |   2 +-
 internal/vault/walk_test.go                        |  33 +++
 internal/writer/atomic.go                          | 225 ++++-----------------
 13 files changed, 178 insertions(+), 230 deletions(-)
```

## Grep de chamadores — inalterado

```
$ grep -rn "writer\.WriteAtomic\|writer\.SweepStaleTempFiles" --include=*.go internal cmd | grep -v _test
internal/service/write.go:161:	if err := writer.WriteAtomic(ctx, absPath, []byte(fullContent)); err != nil {
internal/service/write.go:260:	if err := writer.WriteAtomic(ctx, absPath, proposed); err != nil {
internal/service/write.go:389:	if err := writer.WriteAtomic(ctx, absPath, proposed); err != nil {
internal/service/write.go:640:		if err := writer.WriteAtomic(ctx, absRef, rewritten); err != nil {
internal/service/write.go:846:	if err := writer.WriteAtomic(ctx, absTo, fromRaw); err != nil {
cmd/gobsidian/servico.go:78:		r, err := writer.SweepStaleTempFiles(ctx, cfg.VaultPath)
cmd/gobsidian/servico.go:103:	// writer.SweepStaleTempFiles.
```

Sete linhas: as **seis chamadas** intocadas mais o comentario de
`servico.go:103`, que fica para a Task 172. Eram oito antes; a oitava era o
comentario de `vault/longpath_windows.go:26`, que o brief mandou atualizar para
`SweepStaleTempFiles` e por isso saiu do grep.

## `vault` continua folha

```
$ go list -f '{{.Imports}}' ./internal/vault/
[bytes context errors fmt golang.org/x/sys/windows io io/fs log/slog os path path/filepath strings sync sync/atomic syscall time]
```

Nenhum `internal/*`. `atomic.go` e os dois `syncdir_*` so trazem stdlib, e
`golang.org/x/sys/windows` ja estava la antes desta tarefa.

## Provas de mutacao

### Step 4 — a constante do prefixo e uma conta so

```
$ pwsh -File scripts/mutate.ps1 -Path internal/vault/atomic.go -Anchor 'TempFilePrefix = ".gobsidian-tmp-"' -Replacement 'TempFilePrefix = ".gobsidian-tmq-"' -Test TestWalkIgnoraTemporarioDeBinarioAntigo -Package ./internal/vault/
[...] Mutando internal/vault/atomic.go
      - TempFilePrefix = ".gobsidian-tmp-"
      + TempFilePrefix = ".gobsidian-tmq-"

[...] go test -race -run TestWalkIgnoraTemporarioDeBinarioAntigo ./internal/vault/
----------------------------------------------------------------------
--- FAIL: TestWalkIgnoraTemporarioDeBinarioAntigo (0.01s)
    walk_test.go:138: o temporario entrou no walk: [.gobsidian-tmp-abc123.md a.md]
FAIL
FAIL	github.com/jonyd/gobsidian/internal/vault	0.513s
FAIL
----------------------------------------------------------------------
[OK] internal/vault/atomic.go restaurado byte a byte (SHA-256 confere).

[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
EXIT=0
```

**Desvio do brief, deliberado.** O brief escreve a fixture como
`.gobsidian-tmp-abc123`, sem extensao. Esse nome nao prova nada: `filepath.Ext`
devolve `.gobsidian-tmp-abc123` inteiro, que nao e `.md` nem esta em
`assetExts`, entao `Classify` o exclui pelo filtro de extensao com ou sem
`isNoise` — o teste passaria **tambem com a constante mutada**, e a prova de
mutacao teria saido 1. A fixture virou `.gobsidian-tmp-abc123.md`, que so e
excluida pelo prefixo, e e a mesma forma que `TestWalkExcludesAndClassifies` ja
usava na linha 57 pelo mesmo motivo. O comentario do teste registra isso.

Tambem adaptei a assinatura, como o brief pediu: `vault.New(root)` seguido de
`v.Walk(ctx, func(e vault.Entry) error)`, e nao o `vault.Walk(ctx, root, fn)` do
brief, que nao existe.

### Step 5 — o temporario nao fica para tras quando o callback falha

```
$ pwsh -File scripts/mutate.ps1 -Path internal/vault/atomic.go -Anchor 'cleanup := true' -Replacement 'cleanup := false' -Test TestReplaceFileCallbackFalhaNaoTocaOAlvo -Package ./internal/vault/
[...] Mutando internal/vault/atomic.go
      - cleanup := true
      + cleanup := false

[...] go test -race -run TestReplaceFileCallbackFalhaNaoTocaOAlvo ./internal/vault/
----------------------------------------------------------------------
--- FAIL: TestReplaceFileCallbackFalhaNaoTocaOAlvo (1.62s)
    atomic_test.go:263: temporario ficou para tras: [C:\Users\jonyd\AppData\Local\Temp\TestReplaceFileCallbackFalhaNaoTocaOAlvo2446720163\001\.gobsidian-tmp-2434599411]
    testing.go:1464: TempDir RemoveAll cleanup: unlinkat C:\Users\jonyd\AppData\Local\Temp\TestReplaceFileCallbackFalhaNaoTocaOAlvo2446720163\001\.gobsidian-tmp-2434599411: The process cannot access the file because it is being used by another process.
FAIL
FAIL	github.com/jonyd/gobsidian/internal/vault	2.145s
FAIL
----------------------------------------------------------------------
[OK] internal/vault/atomic.go restaurado byte a byte (SHA-256 confere).

[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
EXIT=0
```

### Step 5 — o erro do callback chega embrulhado

```
$ pwsh -File scripts/mutate.ps1 -Path internal/vault/atomic.go -Anchor 'return fmt.Errorf("escrevendo no temporario %q: %w", tmpName, err)' -Replacement 'return fmt.Errorf("escrevendo no temporario %q: %v", tmpName, err)' -Test TestReplaceFileCallbackFalhaNaoTocaOAlvo -Package ./internal/vault/
[...] Mutando internal/vault/atomic.go
      - return fmt.Errorf("escrevendo no temporario %q: %w", tmpName, err)
      + return fmt.Errorf("escrevendo no temporario %q: %v", tmpName, err)

[...] go test -race -run TestReplaceFileCallbackFalhaNaoTocaOAlvo ./internal/vault/
----------------------------------------------------------------------
--- FAIL: TestReplaceFileCallbackFalhaNaoTocaOAlvo (0.01s)
    atomic_test.go:250: err = escrevendo no temporario "C:\\Users\\jonyd\\AppData\\Local\\Temp\\TestReplaceFileCallbackFalhaNaoTocaOAlvo2434244873\\001\\.gobsidian-tmp-772772158": codec falhou, quero codec falhou embrulhado
FAIL
FAIL	github.com/jonyd/gobsidian/internal/vault	0.507s
FAIL
----------------------------------------------------------------------
[OK] internal/vault/atomic.go restaurado byte a byte (SHA-256 confere).

[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
EXIT=0
```

## `go test -race -count=3`

```
$ go test -race -count=3 ./internal/vault/ ./internal/writer/
ok  	github.com/jonyd/gobsidian/internal/vault	45.628s
ok  	github.com/jonyd/gobsidian/internal/writer	4.713s
```

Os 1.000 ciclos de `TestRNF11NoCorruptionUnder1000Crashes` rodam agora em
`vault`, tres vezes, com `-race`.

## `verify.ps1` — ultima linha

Depois do ruling, as 14 etapas passam:

```
[...] 9. golangci-lint
[OK] golangci-lint
[...] 10. golangci-lint (linux)
[OK] golangci-lint (linux)
[...] 11. check_net (RNF-30)
[OK] check_net (RNF-30)
[...] 12. check_tool_params
[OK] check_tool_params
[...] 13. check_doc_refs
[OK] check_doc_refs
[...] 14. check_readme_anchors
[OK] check_readme_anchors

[OK] Bateria completa. Pode commitar.
```

Ultima linha: **`[OK] Bateria completa. Pode commitar.`**

A contagem de pulados da etapa 3 lista `TestWriteAtomicPreservaOModoDoAlvo`, que
pula no Windows por `runtime.GOOS` — e o mesmo skip que ele tinha em `writer`,
pelo mesmo motivo (o runtime do Go so mapeia o bit de somente-leitura), e nao uma
regressao da mudanca de pacote.

## TDD — RED e GREEN

`audit_reports.ps1 171` sinaliza a ausencia destas secoes. A resposta honesta e
que **esta tarefa nao foi test-first, e nem podia ser**: os dois testes novos sao
de regressao sobre codigo que ja existia — um prende a constante que o Step 4
passou a compartilhar, o outro cobre o caminho de erro que o callback do Step 2
criou. Nao houve um RED cronologico "antes da implementacao".

O que substitui o RED aqui, e e mais forte, sao as tres provas de mutacao acima:
cada uma mostra o teste **reprovando** com a regra mutada, com a saida colada e
`EXIT=0` do `mutate.ps1`. Um teste que so passa nao prova nada; esses tres
falham quando a regra some.

O GREEN e o `go test -race -count=3` da secao acima: `ok` nos dois pacotes, tres
repeticoes, com `-race`.

## Notas

- **`TestMain` migrou junto.** `atomic_test.go` define o `TestMain` do processo
  auxiliar, e ele era o unico de `writer_test`. Conferido antes de mover: nao
  havia `TestMain` em `internal/vault` (nem em `package vault`, nem em
  `package vault_test`), entao nao ha colisao. `writer` fica sem `TestMain`, que
  e o padrao.
- **Nenhum subteste precisou ficar para tras.** Os tres arquivos movidos so
  usavam `TempFilePrefix`, `SweepStaleTempFiles` e `WriteAtomic` — todos
  simbolos que migraram. `PathLocker` e `NormalizeEOL` nao aparecem em nenhum
  deles.
- **O commit levou 13 caminhos explicitos.** Nada de `git add -A` nem `.`. O
  `progress.md` modificado e os mais de cem arquivos nao rastreados do dono
  (`test-vault/`, `.claude/skills/`, `Resume-Claude.ps1`, briefs e reviews)
  continuam exatamente onde estavam.
- **`ReplaceFile` ainda nao tem chamador de producao** — e proposital. Ela existe
  para a Task 172, em que os dois caches passam a codificar em streaming direto
  para o temporario. Hoje quem a exercita e `WriteAtomic` (que e o wrapper) e o
  teste do callback que falha.
- **Divergencia pre-existente, nao corrigida por estar fora de escopo:**
  `docs/ARCHITECTURE.md` §5.5 descreve o retry do rename como "backoff
  exponencial, tres tentativas, 50 ms iniciais". O codigo faz 10 tentativas com
  10 ms fixos, e ja fazia antes desta tarefa. Merece tarefa propria.
