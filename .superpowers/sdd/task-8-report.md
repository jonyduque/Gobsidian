# Task 8 Report: Varredura, EOL, BOM, caminho longo e placeholder de nuvem

## O que foi implementado

Todos os arquivos pedidos pelo brief, em `internal/vault/`:

- `eol.go` — `EOLStyle`, `DetectEOL`, `StripBOM`, `AddBOM`, `NormalizeEOL`.
- `longpath_windows.go` / `longpath_other.go` — `LongPath`, atrás de build tags.
- `cloud_windows.go` / `cloud_other.go` — `IsCloudOnly`, atrás de build tags.
- `vault.go` — `Vault`, `New`, `Root`, `Abs`, `Open`, `ReadRange`, `ReadAll`.
- `walk.go` — `Entry`, `excludedDirs`, `assetExts`, `isNoise`, `(*Vault).Walk`.
- `eol_test.go`, `walk_test.go` — testes do brief, copiados literalmente.
- `context_test.go` — dois testes extras que o brief não escreveu (ver seção de checks abaixo), cobrindo cancelamento de contexto.

Todo o código de implementação foi copiado do brief **sem nenhuma correção mecânica** — compilou e passou de primeira. Nada precisou ser ajustado.

## TDD Evidence

**RED** — `go test ./internal/vault/ -v` antes de criar os arquivos de implementação (só com `eol_test.go` e `walk_test.go` presentes):

```
# github.com/jonyd/gobsidian/internal/vault_test [github.com/jonyd/gobsidian/internal/vault.test]
internal\vault\eol_test.go:14:14: undefined: vault.EOLStyle
internal\vault\eol_test.go:16:34: undefined: vault.EOLLF
internal\vault\eol_test.go:17:42: undefined: vault.EOLCRLF
...
internal\vault\eol_test.go:26:20: undefined: vault.DetectEOL
internal\vault\eol_test.go:36:21: undefined: vault.StripBOM
FAIL	github.com/jonyd/gobsidian/internal/vault [build failed]
FAIL
```

Falha esperada: os símbolos do pacote ainda não existiam (nenhum arquivo de implementação criado ainda). `walk_test.go` também falharia da mesma forma por `vault.New` indefinido, mas o build já aborta nos símbolos de `eol_test.go` primeiro.

**GREEN** — `go test -race ./internal/vault/ -v` depois de implementar todos os arquivos:

```
=== RUN   TestDetectEOL
    --- PASS (todas as 6 subcases)
=== RUN   TestStripBOM
--- PASS: TestStripBOM (0.00s)
=== RUN   TestResolveConfinement ... (Task 7, 21 subcases)
--- PASS: TestResolveConfinement
=== RUN   TestCanonicalizeRejectsStandalone ... (Task 7, 4 subcases)
--- PASS
=== RUN   TestResolveRejectsSiblingWithSharedPrefix
--- PASS
=== RUN   TestCanonicalizeUsesForwardSlashes
--- PASS
=== RUN   TestWalkExcludesAndClassifies
--- PASS: TestWalkExcludesAndClassifies (0.02s)
=== RUN   TestNewRejectsMissingRoot
--- PASS: TestNewRejectsMissingRoot (0.00s)
=== RUN   TestReadRange
--- PASS: TestReadRange (0.01s)
PASS
ok  	github.com/jonyd/gobsidian/internal/vault	1.522s
```

Todos os testes da Task 7 (`path_test.go`) continuam passando — nenhum arquivo dessa task foi tocado.

## Os cinco checks extras — resultado real de cada um

| # | Check | Método | Resultado |
|---|-------|--------|-----------|
| 1 | Contexto cancelado interrompe a varredura de fato | Escrevi `context_test.go` com dois testes permanentes: (a) contexto já cancelado antes de `Walk` começar, 300 arquivos no cofre; (b) `cancel()` disparado de dentro do callback após 10 chamadas. | (a) `Walk` retorna `context.Canceled`, callback nunca é invocado (`visited == 0`) — a checagem `ctx.Err()` roda antes até da primeira entrada (a raiz do próprio diretório). (b) `Walk` retorna `context.Canceled` e para com `visited == 10` de 300 possíveis — confirma que a checagem dentro do `WalkDirFunc` interrompe a recursão de fato, não apenas observa e ignora. **Comportamento correto, verificado.** |
| 2 | `ReadRange` nas bordas | Probes ad hoc (removidos depois): `start==end`, `end` além do EOF, `start` além do EOF, `start` negativo, em arquivo de 10 bytes ("0123456789"). | `(5,5)` → `""`, sem erro. `(8,100)` → `"89"`, sem erro (o `io.EOF` de `ReadAt` é engolido pelo `if err != nil && err != io.EOF`, e o slice é cortado por `buf[:n]`). `(100,200)` → `""`, sem erro (n=0, mesmo tratamento de EOF). `(-3,5)` → erro `"lendo \"A.md\" em -3..5: readat ...: negative offset"`, sem panic — o próprio `os.File.ReadAt` rejeita o offset negativo e o erro é envolvido corretamente pela mensagem do brief. **Nenhuma borda panica ou devolve dado incorreto; nenhum teste novo necessário.** |
| 3 | `NormalizeEOL` é idempotente | Probe ad hoc: conteúdo misto `"a\r\nb\nc\r\nd\n"`, aplicado duas vezes para `EOLLF` e para `EOLCRLF`. | `EOLLF`: `"a\nb\nc\nd\n"` nas duas aplicações, idêntico. `EOLCRLF`: `"a\r\nb\r\nc\r\nd\r\n"` nas duas aplicações, idêntico. **Idempotente nos dois estilos, inclusive com entrada mista. Nenhum gap.** |
| 4 | `DetectEOL` com CR solto (Mac clássico) | Probe ad hoc: `"a\rb\rc\r"` (sem nenhum `\n`). | Devolve `EOLLF`. É o resultado do algoritmo do brief: ele só conta `\r\n` e `\n`; um CR sozinho não é reconhecido como quebra de linha nenhuma, então `crlf=0` e `lf=0`, e o desempate cai no default `EOLLF`. Isso é sensato dado o desenho do tipo `EOLStyle` (só modela LF/CRLF, não um terceiro estilo CR-only) — o efeito prático é que `NormalizeEOL` sobre conteúdo novo escrito nesse arquivo usaria LF, e o conteúdo antigo em CR-solto nunca é tocado porque `NormalizeEOL` só processa o texto novo, não reescreve o arquivo inteiro. **Comportamento é uma consequência direta e documentada do desenho de dois estilos, não um bug. Nenhum teste novo necessário.** |
| 5 | Diretório ilegível não aborta a varredura | Tentei três abordagens para forçar um erro real de leitura de diretório no Windows: (a) `os.Chmod(dir, 0o000)`; (b) `icacls <dir> /deny "user:(RX)"`; (c) junction quebrada apontando para destino inexistente. | (a) e (b) não impediram a leitura — o diretório continuou sendo listado normalmente (`notes=[Bad/B.md Good/A.md Good2/C.md]`), mesmo com a ACE de DENY explícita aplicada e confirmada via `icacls` (a saída mostrou `maquina de referencia\jonyd:(DENY)(RX)`); a leitura passou mesmo assim, provavelmente por causa de algo no modelo de token/sandbox deste ambiente que não estou conseguindo isolar rapidamente. (c) falhou ao criar (mklink recusou o destino inexistente). **Inconclusivo neste ambiente** — não consegui reproduzir empiricamente um diretório genuinamente ilegível no Windows sandboxed deste shell. O caminho de código (`fs.SkipDir` retornado quando `err != nil` e `d.IsDir()`, dentro do próprio `WalkDirFunc` passado a `filepath.WalkDir`) é uma interação direta e documentada do contrato do `filepath.WalkDir` da stdlib — meu grau de confiança nele é alto por revisão de código, mas não pude observá-lo em execução real. Reporto como não verificado empiricamente, e não escrevi um teste que dependeria dessa quirk de ambiente. |

## A entrada silenciosamente ignorada — decisão

O `Walk` do brief faz `return nil` quando `Canonicalize` rejeita uma entrada lida do disco (por exemplo, um arquivo cujo nome termina em ponto ou espaço no Windows). Decidi **manter exatamente como o brief escreveu, sem adicionar nenhum parâmetro de logger nem qualquer efeito colateral global** (nada de `log.Println`, `slog`, contador em variável de pacote, etc.). Razões:

1. A assinatura da interface que a Task 8 se compromete a produzir é fixa: `(*Vault).Walk(ctx, func(Entry) error) error`. Tarefas 9+ (parser, índice, busca, escrita) dependem dessa assinatura exata. Acrescentar um parâmetro de logger ou um canal de avisos mudaria o contrato que o resto do projeto consome, e o brief não pediu isso.
2. Transformar o erro de `Canonicalize` em um erro que aborta o `Walk` inteiro é pior do que silenciar: um único nome de arquivo irrepresentável (`Nota .md`, por exemplo) derrubaria a indexação do cofre inteiro — exatamente o cenário que o comentário do brief já descarta.
3. Um efeito colateral não solicitado (escrever em stderr, incrementar uma variável global) seria disciplina violada: "anything built the brief did not ask for?" — eu não deveria introduzir observabilidade ad hoc numa camada que nem tem um logger injetado em nenhum outro lugar do pacote.

Isso não significa que o problema desapareceu: **uma nota que o `Canonicalize` rejeita é uma nota que o usuário não consegue alcançar nem diagnosticar**, como o enunciado da task corretamente aponta. A camada certa para tornar isso observável é a que vier depois e consumir `Walk` — por exemplo, o índice (Task 9+) pode contar quantas vezes o callback foi invocado versus quantas entradas o `WalkDir` subjacente visitou, ou a ferramenta `vault_stats` pode expor um contador de "entradas ignoradas por nome inválido". Documentei essa lacuna aqui para que a próxima task saiba que ela existe e não fica presumindo que toda entrada do disco vira uma `Entry`.

## Arquivos alterados/criados

- `internal/vault/eol.go` (novo)
- `internal/vault/eol_test.go` (novo)
- `internal/vault/longpath_windows.go` (novo)
- `internal/vault/longpath_other.go` (novo)
- `internal/vault/cloud_windows.go` (novo)
- `internal/vault/cloud_other.go` (novo)
- `internal/vault/vault.go` (novo)
- `internal/vault/walk.go` (novo)
- `internal/vault/walk_test.go` (novo)
- `internal/vault/context_test.go` (novo, além do brief — testes de cancelamento de contexto)

Nenhum arquivo da Task 7 (`path.go`, `path_windows.go`, `path_other.go`, `path_test.go`) foi modificado.

## Self-review

- **Completude:** toda função/tipo listado na seção "Produces" do brief existe: `Vault`, `Root()`, `Open`, `ReadRange`, `ReadAll`, `New`, `Entry{Path,Size,ModTime,IsNote,CloudOnly}`, `(*Vault).Walk`, `DetectEOL`, `StripBOM`, `LongPath`. Os dois pares de arquivos com build tag (`longpath_*`, `cloud_*`) existem e são mutuamente exclusivos (`windows` / `!windows`).
- **Correção:** nada abre anexo nem placeholder de nuvem durante a varredura. `Walk` só chama `d.Info()` (stat via a entrada de diretório já aberta pelo `WalkDir`, não um `Open`) e `IsCloudOnly(abs)`, que por sua vez só chama `windows.GetFileAttributes` — não há `os.Open`/`os.ReadFile` em nenhum ponto do caminho de anexo. Confirmei lendo `cloud_windows.go` linha por linha: o comentário do próprio brief já é preciso ("abrir é exatamente o que dispara a hidratação") e o código bate com isso.
- **Disciplina:** não adicionei nada que o brief não pediu, com uma exceção deliberada e sinalizada: `context_test.go`, porque a instrução da task pedia explicitamente "Write a test" para o check de cancelamento de contexto. Os outros quatro checks foram só sondados e reportados, sem gerar teste permanente, porque nenhum revelou um gap real (fronteiras de `ReadRange` seguras, `NormalizeEOL` idempotente, CR solto tem comportamento sensato e documentado, diretório ilegível ficou inconclusivo por limitação de ambiente, não por defeito de código).
- **Testes:** rodei a suíte inteira várias vezes com `-race`; saída limpa, sem warnings. Os probes ad hoc que criei para os checks 2–5 foram copiados para um arquivo temporário fora do repositório (`internal/vault/zzprobe*_test.go`), rodados, e depois **removidos** — não sobraram no diretório do pacote. Os diretórios temporários criados para testar `icacls`/junction (`mktemp -d`, fora do repo) foram removidos ao final (`icacls /reset` + `rm -rf`). `git status --porcelain` mostra só os arquivos novos pretendidos, nenhum artefato de teste esquecido.

## Correções mecânicas ao código do brief

Nenhuma. Todo o código de `eol.go`, `longpath_windows.go`, `longpath_other.go`, `cloud_windows.go`, `cloud_other.go`, `vault.go`, `walk.go`, `eol_test.go` e `walk_test.go` foi copiado literalmente do brief e compilou/passou de primeira.

## Preocupações

- O check 5 (diretório ilegível) ficou **inconclusivo empiricamente** neste ambiente — não é uma falha do código, é uma limitação de sandbox/permissões que não consegui contornar em tempo razoável (chmod e icacls deny não restringiram a leitura do diretório de teste; motivo exato não identificado, possivelmente relacionado ao token do processo neste shell). Reportado com transparência acima; não escrevi um teste que dependeria de uma condição de ambiente que não pude reproduzir de forma confiável.
- `golang.org/x/sys` está listado como `// indirect` em `go.mod`. `cloud_windows.go` o importa diretamente. Isso compila e passa em todos os `go vet` (nativo Windows, `GOOS=linux`, `GOOS=darwin`) sem problema — Go não exige que uma dependência usada diretamente esteja marcada `// indirect` versus direta para compilar, só `go mod tidy` corrigiria a anotação, e a instrução foi explicitamente não rodar `go mod tidy`/`go get`. Sinalizo isso caso uma task futura rode `go mod tidy` e o comentário mude — não é um problema funcional agora.

## Fix pass — review findings

Commit base: `1b7ece9`. Scope: `internal/vault/vault.go`, `internal/vault/walk.go`, `internal/vault/eol.go`, new `internal/vault/longpath_windows_test.go` + `internal/vault/longpath_other_test.go`, `internal/vault/walk_test.go`. `path.go`, `path_windows.go`, `path_other.go`, `path_test.go` (Task 7) untouched.

### Finding 1 — missing vault reports as empty vault

Transcribed from the plan (`docs/superpowers/plans/2026-07-25-gobsidian-v01.md`, Task 8, Step 6): `Vault` gained a `walkRoot` field set to `LongPath(abs)` in `New`. `Walk` now runs `filepath.WalkDir(v.walkRoot, ...)` and calls `Canonicalize(v.walkRoot, abs)` — both against the same root, so `filepath.Rel` never compares mismatched forms. The error branch now checks `d == nil` first and returns `fmt.Errorf("varrendo a raiz do cofre %q: %w", v.root, err)` instead of falling through to `return nil`.

One deliberate deviation from the literal plan text: the plan keeps `abs == v.root` in the directory branch that skips reprocessing the walk root itself. Since `filepath.WalkDir` now runs against `v.walkRoot`, that comparison is only true when `walkRoot == root` (no long-path prefix in effect). The plan's code and this code are behaviorally identical in every case tested — the root's own base name is never a key of `excludedDirs`, so a missed match just falls through to `return nil` either way — but comparing `abs` against a variable it structurally cannot equal, two lines away from where `v.walkRoot` is used for everything else, is the kind of inconsistency a reviewer flags on sight. Changed it to `abs == v.walkRoot`.

### Finding 2 — dropped entries leave no trace

Transcribed from the plan: `skipped atomic.Int64`, `skippedMu sync.Mutex`, `skippedSamples []string` on `Vault`; `maxSkippedSamples = 50`; `recordSkip(abs string, cause error)`; exported `SkippedEntries() (int64, []string)`. `Walk` now calls `v.recordSkip` on directory-error skip, file-error skip, `Canonicalize` rejection, and `d.Info()` failure — every `return nil` / `return fs.SkipDir` in the error paths that used to be silent now records first.

### Finding 3 — LongPath unused in traversal, untested

`Walk` traverses `v.walkRoot` (see Finding 1) instead of `v.root`, so the walk itself now benefits from `\\?\` prefixing the same way `New`'s `os.Stat`, `Abs`, and `IsCloudOnly` already did.

Wrote `internal/vault/longpath_windows_test.go` (`//go:build windows`) and `internal/vault/longpath_other_test.go` (`//go:build !windows`) — see "Prove finding 3" below for the case list and results.

### Minor findings

- `ReadRange`: added `maxReadRangeBytes = 64 << 20` (64 MiB) next to `maxSkippedSamples` in `vault.go`, and a check `if end-start > maxReadRangeBytes` before the allocation, returning a wrapped error naming the requested range and the cap. Added `TestReadRangeRejectsHugeRange` to `walk_test.go` (requests `0..1<<40` and asserts an error).
- `DetectEOL` doc comment: added a paragraph stating that a lone CR (classic Mac) has no representation in `EOLStyle`, resolves to `EOLLF`, and that this is a consequence of the two-value type, not a bug.
- `TestWalkExcludesAndClassifies`: fixture extended to write a file under every entry of `excludedDirs` (added `.stfolder/syncthing.db` alongside the pre-existing `.obsidian`, `.git`, `.trash`) and to exercise every branch of `isNoise` (added `.~lock.Penal.md#`, `Thumbs.db`, `.DS_Store`, `rascunho.tmp`, `.gobsidian-tmp-abc123.md`, alongside the pre-existing `~$temp.md` and `desktop.ini`). Expected notes/assets counts unchanged, since none of the added fixtures are notes or assets.

### Command: go test -race ./internal/vault/ -v

PASS, all subtests including Task 7's `path_test.go` suite (`TestResolveConfinement` 21 subcases, `TestCanonicalizeRejectsStandalone`, `TestResolveRejectsSiblingWithSharedPrefix`, `TestCanonicalizeUsesForwardSlashes`), Task 8's own suite (`TestDetectEOL` 6 subcases, `TestStripBOM`, `TestLongPathWindows` 7 subcases, `TestWalkExcludesAndClassifies`, `TestNewRejectsMissingRoot`, `TestReadRange`, `TestReadRangeRejectsHugeRange`), and `context_test.go`'s cancellation tests. Output pristine, no warnings.

```
ok  	github.com/jonyd/gobsidian/internal/vault	3.282s
```

### Prove finding 1 is closed

Built a `Vault` for a fresh `os.MkdirTemp` directory containing one note, then `os.RemoveAll`'d the directory, then called `Walk`. Verified with a throwaway in-package test (`internal/vault/zzverify_test.go`, deleted immediately after the run — `git status --porcelain` confirmed no trace left):

```
FINDING1: walkErr == nil: false
FINDING1: visited: 0
FINDING1: error text: varrendo a raiz do cofre "C:\Users\jonyd\AppData\Local\Temp\gobsidian-finding1-2860707886": GetFileAttributesEx C:\Users\jonyd\AppData\Local\Temp\gobsidian-finding1-2860707886: The system cannot find the file specified.
```

`Walk` now returns a non-nil, wrapped error naming the vault root instead of nil-with-zero-entries.

### Prove finding 2 is closed

Could create a trailing-dot filename on disk — using the `\\?\` prefix, which bypasses the Win32 CreateFile normalization that ordinarily strips a trailing dot/space before the file is ever created.

Getting the specific Canonicalize-rejection route (rather than the pre-existing "unreadable directory" branch) took two attempts, both run as a throwaway in-package test (`internal/vault/zzverify2_test.go`, deleted after — confirmed by `git status --porcelain` below):

- First attempt (short vault root): created a directory named `Pasta .` via `\\?\`, with a normal `Nota.md` inside. `SkippedEntries()` did go non-zero (count 1), but the sample was an "open ...: The system cannot find the file specified" error — Win32 auto-trims the trailing dot when opening the directory too, so `filepath.WalkDir` (using the plain, short, unprefixed root) never lists what is inside; `Nota.md` is skipped via the generic directory-error branch, never reaching `Canonicalize`.
- Second attempt (root padded to 256 bytes, past `longPathThreshold`): with `Vault.walkRoot` carrying the `\\?\` prefix, `filepath.WalkDir` can enumerate and descend into `Pasta .` correctly (the prefix disables Win32's auto-trim), reaches `Nota.md` inside it as an ordinary note candidate, and `Canonicalize` itself rejects it:

```
FINDING2: raiz de teste tem 256 bytes
FINDING2: entrada no disco: "Pasta ."
FINDING2: entries reached callback: 0
FINDING2: SkippedEntries count: 1
FINDING2: sample: \\?\C:\Users\jonyd\...\padding-dir-...\Pasta .\Nota.md: caminho malformado: componente "Pasta ." termina em ponto ou espaco, o que o Windows remove ao abrir e faria o mesmo arquivo ter mais de um caminho canonico
```

This is the exact scenario Finding 2 describes: a note (`Nota.md`) that exists on disk, is genuinely reachable by the walk, and is rejected by `Canonicalize`'s Windows dot/space rule — now recorded by name instead of silently dropped.

### Prove finding 3 is closed

`internal/vault/longpath_windows_test.go` (`//go:build windows`), `TestLongPathWindows`, 7 table cases, all PASS:

| Case | Input | Result |
|---|---|---|
| short path returned unchanged | `C:\Users\jonyd\Documents\note.md` (33 bytes) | unchanged |
| just below threshold (239) not prefixed | `C:\` + 236x`a` (239 bytes) | unchanged |
| at threshold (240) gains `\\?\` | `C:\` + 237x`a` (240 bytes) | `\\?\C:\` + 237x`a` |
| above threshold gains prefix | `C:\` + 300x`a` | `\\?\C:\` + 300x`a` |
| already-prefixed path not double-prefixed | `\\?\C:\` + 300x`a` | unchanged |
| UNC becomes `\\?\UNC\`, no triple backslash | `\\server\share\` + 230x`a` | `\\?\UNC\server\share\` + 230x`a` |
| forward slashes and `.`/`..` absent from output | `C:/Users/jonyd/../jonyd/AppData/Local/` + 230x`a` | `\\?\C:\Users\jonyd\AppData\Local\` + 230x`a` |

Every case also asserted the output contains no forward slash, no `\..\` segment, and no triple backslash after the UNC prefix.

`internal/vault/longpath_other_test.go` (`//go:build !windows`), `TestLongPathOther`, 2 table cases, both asserting identity (short path; a ~400-byte path built from 200 repeated `a/` segments) — not run natively in this Windows session (build-tag excluded), but compiled clean under `GOOS=linux go vet ./...` and `GOOS=darwin go vet ./...`.

### go vet / gofmt / cross-GOOS vet

```
go vet ./internal/vault/...        -> clean
go vet ./...                       -> clean
gofmt -l .                         -> no output (nothing to format)
GOOS=linux go vet ./...            -> clean
GOOS=darwin go vet ./...           -> clean
```

### git status --porcelain

```
 M internal/vault/eol.go
 M internal/vault/vault.go
 M internal/vault/walk.go
 M internal/vault/walk_test.go
?? internal/vault/longpath_other_test.go
?? internal/vault/longpath_windows_test.go
```

No stray files, no leftover temp directories (all zzverify*_test.go scratch files and os.MkdirTemp directories created during verification were removed after each check).

## Fix pass 2 — inert fixtures and regression tests

Commit base: `3c3beb0`. Scope: `internal/vault/vault.go`, `internal/vault/walk.go`, `internal/vault/walk_test.go`, new `internal/vault/skip_internal_test.go` + `internal/vault/walk_windows_test.go`. `internal/vault/eol.go` untouched (Finding 5's `DetectEOL` transcription gap turned out to already match disk — the lone-CR paragraph existed in code, just not in the plan). `path.go`, `path_windows.go`, `path_other.go`, `path_test.go` (Task 7) untouched.

### Finding 1 — inert exclusion fixtures

The fixture in `TestWalkExcludesAndClassifies` was extended to write a `.md` file inside every `excludedDirs` entry (`.obsidian/Nota.md`, `.git/Nota.md`, `.trash/velha.md` — already present — `.stfolder/Nota.md`) so pruning each directory has an observable effect: if the prune is removed, a note that would otherwise be indexed leaks through. The original per-directory example files (`workspace.json`, `config`, `syncthing.db`) were kept alongside for realism but don't count toward coverage — none has a note/asset extension.

For `isNoise`, `.~lock.Penal.md#` was renamed to `.~lock.Penal.md` (dropped the trailing `#`) so `filepath.Ext` returns `.md` instead of `.md#` — with the `#`, the extension filter alone already rejected the file and the `isNoise` branch was never exercised. `~$temp.md` and `.gobsidian-tmp-abc123.md` already had the right shape and needed no change.

`desktop.ini`, `Thumbs.db`, `.DS_Store` (fixed filenames, no `.md`/asset extension possible) and the `*.tmp` suffix rule (`.tmp` is not in `assetExts`) cannot be made to bite by any choice of fixture — the extension filter (`if !isNote && !isAsset { return nil }`) already excludes them regardless of `isNoise`. Marked defensive with a comment directly above each case in `walk.go`, explaining why no fixture can prove them, instead of contorting the fixture to fake coverage. The fixture files for these four are still written (documents that real vaults contain them) but a code comment in the test now says explicitly they don't prove anything.

### Mutation table — proof per rule

Each rule below was disabled directly in `walk.go` (map entry removed, or `switch` case deleted/commented), `go test -run TestWalkExcludesAndClassifies` run, result recorded, then the exact original text restored via `Edit` and reverified with `git diff internal/vault/walk.go` (confirmed only the two new defensive-branch comments remained as a diff against `HEAD`, no accidental drift) before moving to the next rule.

| # | Rule removed | Result | Verdict |
|---|---|---|---|
| 1 | `excludedDirs[".obsidian"]` | FAIL — `notas = [.obsidian/Nota.md Civil/PONTO 03.md Penal/B.md]` | **Covered** |
| 2 | `excludedDirs[".git"]` | FAIL — `notas = [.git/Nota.md Civil/PONTO 03.md Penal/B.md]` | **Covered** |
| 3 | `excludedDirs[".trash"]` | FAIL — `notas = [.trash/velha.md Civil/PONTO 03.md Penal/B.md]` | **Covered** |
| 4 | `excludedDirs[".stfolder"]` | FAIL — `notas = [.stfolder/Nota.md Civil/PONTO 03.md Penal/B.md]` | **Covered** |
| 5 | `isNoise` `~$` prefix case | FAIL — `notas = [Civil/PONTO 03.md Penal/B.md ~$temp.md]` | **Covered** |
| 6 | `isNoise` `.~lock.` prefix case | FAIL — `notas = [.~lock.Penal.md Civil/PONTO 03.md Penal/B.md]` | **Covered** |
| 7 | `isNoise` `desktop.ini`/`thumbs.db`/`.ds_store` case | PASS — unchanged | **Defensive, not alcancavel por efeito** |
| 8 | `isNoise` `*.tmp` suffix case | PASS — unchanged | **Defensive, not alcancavel por efeito** |
| 9 | `isNoise` `.gobsidian-tmp-` prefix case | FAIL — `notas = [.gobsidian-tmp-abc123.md Civil/PONTO 03.md Penal/B.md]` | **Covered** |

7 of 9 rules are genuinely covered — deleting the branch changes the note count and the test fails. 2 (`desktop.ini`/`Thumbs.db`/`.DS_Store`, `*.tmp`) are confirmed unreachable by effect: the extension filter downstream already excludes them, with or without `isNoise` catching them first. Both are now documented as belt-and-braces in `walk.go`, not claimed as tested.

### Finding 2 — permanent regression tests for the two prior fixes

- **Missing root:** `TestWalkFailsWhenRootRemoved` in `walk_test.go` — builds a `Vault` for a real `t.TempDir()`, `os.RemoveAll`s it, calls `Walk`, asserts a non-nil error and that the callback is never invoked.
- **Skip recorded:** two tests, since the Windows scenario carried real risk of being environment-brittle:
  - `internal/vault/walk_windows_test.go` (`//go:build windows`), `TestWalkRecordsSkipOnCanonicalizeRejection`: reproduces the exact scenario verified ad hoc in fix pass 1 — pads the vault root past `longPathThreshold` (220 `a`s under `t.TempDir()`) so `Vault.walkRoot` carries the `\\?\` prefix, creates a directory literally named `Pasta .` and a `Nota.md` inside it via `vault.LongPath`-prefixed `os.Mkdir`/`os.WriteFile` (bypassing Win32's silent trailing-dot trim), then asserts `Walk` reaches zero entries via the callback and `SkippedEntries()` returns a non-zero count with a sample naming `Nota.md`. Each creation step (`MkdirAll`, `Mkdir`, `WriteFile`) is individually `t.Skip`-guarded so the test degrades to a skip, not a failure, if this environment ever refuses trailing-dot names.
  - `internal/vault/skip_internal_test.go` (`package vault`, whitebox), `TestRecordSkipCapsSamplesButNotCount`: the portable fallback the finding asked for regardless — calls `recordSkip` `maxSkippedSamples+5` times directly and asserts (a) the count is not capped, (b) the sample slice caps at `maxSkippedSamples` and keeps the earliest entries, (c) `SkippedEntries()` returns a copy — mutating the returned slice does not corrupt `v.skippedSamples` on a subsequent call, and (d) one more `recordSkip` after the cap still increments the count without growing the sample slice.

Ran both on this machine (real Windows, not WSL): both PASS, the Windows one did not need to skip.

### Finding 3 — `ReadRange` integer overflow

Added `if start < 0 { ... }` before the `end < start` check in `vault.go`. With `start` guaranteed `>= 0` and `end` guaranteed `>= start`, `end-start` is bounded to `[0, math.MaxInt64]` and cannot wrap — the existing `> maxReadRangeBytes` check downstream is now safe. Added `TestReadRangeRejectsOverflowPair` to `walk_test.go`: calls `ReadRange(ctx, "A.md", math.MinInt64, math.MaxInt64)` and asserts an error, not a panic.

### Finding 4 — `SkippedEntries` doc

Moved the cumulative-across-walks semantics from the private `skipped` field comment onto the exported `SkippedEntries` method doc comment in `vault.go`, so `go doc` shows it.

### Finding 5 — plan drift

Updated `docs/superpowers/plans/2026-07-25-gobsidian-v01.md`'s Task 8 section:

- `eol.go` snippet: added the lone-CR paragraph to `DetectEOL`'s doc comment (code already had it from fix pass 1; only the plan was missing it).
- `vault.go` snippet: added `maxReadRangeBytes` constant, the `start < 0` overflow guard in `ReadRange`, and the cumulative-behaviour paragraph on `SkippedEntries`'s doc comment.
- `walk.go` snippet: added the two defensive-branch comments in `isNoise`.
- `walk_test.go` snippet: replaced with the current fixture (per-`excludedDirs`-entry `.md` files, renamed lock-file fixture, defensive-branch fixtures kept but documented as non-probative) and the three new tests (`TestWalkFailsWhenRootRemoved`, `TestReadRangeRejectsHugeRange`, `TestReadRangeRejectsOverflowPair`). Added a short note after the snippet pointing at `skip_internal_test.go` and `walk_windows_test.go`, which live outside this snippet as separate files.
- Found and fixed one more pre-existing drift while diffing snippets against disk, unrelated to this pass's two named gaps: the plan's `walk.go` snippet was missing the `fmt` import (needed for `fmt.Errorf` in the root-walk-error branch added in fix pass 1) and had a comment (`// Comparar com walkRoot, nao com root...`) that was never actually added to the code file, only to the plan, in the `docs:` commit that preceded this pass. Removed the phantom comment from the plan and added the missing import, so the plan now byte-matches the disk file rather than describing code that doesn't exist.

Verified all four snippets (`vault.go`, `walk.go`, `eol.go`, `walk_test.go`) byte-match their disk files via `diff` after editing (fence-line offset aside).

### go test -race ./internal/vault/ -v

```
--- PASS: TestRecordSkipCapsSamplesButNotCount (0.00s)
--- PASS: TestWalkStopsOnPreCancelledContext (0.57s)
--- PASS: TestWalkStopsMidwayOnCancel (0.58s)
--- PASS: TestDetectEOL (0.00s)
--- PASS: TestStripBOM (0.00s)
--- PASS: TestLongPathWindows (0.00s)
--- PASS: TestResolveConfinement (0.00s)
--- PASS: TestCanonicalizeRejectsStandalone (0.00s)
--- PASS: TestResolveRejectsSiblingWithSharedPrefix (0.00s)
--- PASS: TestCanonicalizeUsesForwardSlashes (0.00s)
--- PASS: TestWalkExcludesAndClassifies (0.03s)
--- PASS: TestNewRejectsMissingRoot (0.00s)
--- PASS: TestWalkFailsWhenRootRemoved (0.00s)
--- PASS: TestReadRange (0.01s)
--- PASS: TestReadRangeRejectsHugeRange (0.00s)
--- PASS: TestReadRangeRejectsOverflowPair (0.00s)
--- PASS: TestWalkRecordsSkipOnCanonicalizeRejection (0.01s)
PASS
ok  	github.com/jonyd/gobsidian/internal/vault	(cached)
```

Output pristine, no warnings, `-race` clean.

### go vet / gofmt / cross-GOOS vet

```
go vet ./...             -> clean
gofmt -l .                -> no output (nothing to format)
GOOS=linux go vet ./...   -> clean
GOOS=darwin go vet ./...  -> clean
```

### git status --porcelain

```
 M docs/superpowers/plans/2026-07-25-gobsidian-v01.md
 M internal/vault/vault.go
 M internal/vault/walk.go
 M internal/vault/walk_test.go
?? internal/vault/skip_internal_test.go
?? internal/vault/walk_windows_test.go
```

No stray files, no leftover temp directories, no `zzverify` files.
