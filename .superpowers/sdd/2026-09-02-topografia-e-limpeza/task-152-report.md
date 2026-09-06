# Task 152 — Relatório

## Status
DONE

## Commit
`a637b7f` fix(service): note_move dry-run drops the empty origin diff and fails on an unreadable referrer

## Evidência de TDD

**RED** — testes escritos, rodados ANTES da correção (write.go ainda com `continue` e diff vazio da origem):

```
$ go test ./internal/service/ -run 'TestMoveNoteDryRunNaoFabricaDiffVazioDaOrigem|TestMoveNoteDryRunNaoEngoleReferenciadoraIlegivel' -v
=== RUN   TestMoveNoteDryRunNaoFabricaDiffVazioDaOrigem
    move_test.go:360: a origem entrou em diffs com "": um item vazio diz que a nota nao muda
--- FAIL: TestMoveNoteDryRunNaoFabricaDiffVazioDaOrigem (0.07s)
=== RUN   TestMoveNoteDryRunNaoEngoleReferenciadoraIlegivel
    move_test.go:386: dry-run devolveu sucesso com uma referenciadora ilegivel; diffs=map[origem.md:]
--- FAIL: TestMoveNoteDryRunNaoEngoleReferenciadoraIlegivel (0.10s)
FAIL
FAIL	github.com/jonyd/gobsidian/internal/service	1.959s
FAIL
```

Ambos os testes reprovam pelo motivo esperado: o primeiro por a origem entrar
em `diffs` com `""`; o segundo por o dry-run devolver sucesso com a
referenciadora ilegível engolida por `continue`.

**GREEN** — depois de reescrever o bloco do dry-run em `write.go:505-543`:

```
$ go test ./internal/service/ -run 'TestMoveNoteDryRunNaoFabricaDiffVazioDaOrigem|TestMoveNoteDryRunNaoEngoleReferenciadoraIlegivel|TestMoveDryRunNaoApresentaDiffVazioComoResultado|TestMoveNote_|TestNoteMovePartialFailureReportsWhatWasApplied' -v
=== RUN   TestMoveNote_BrokenAnchorsReportedOnlyWhenMissing
--- PASS: TestMoveNote_BrokenAnchorsReportedOnlyWhenMissing (0.02s)
=== RUN   TestMoveDryRunNaoApresentaDiffVazioComoResultado
--- PASS: TestMoveDryRunNaoApresentaDiffVazioComoResultado (0.02s)
=== RUN   TestNoteMovePartialFailureReportsWhatWasApplied
--- PASS: TestNoteMovePartialFailureReportsWhatWasApplied (0.14s)
=== RUN   TestMoveNote_DryRunLeavesMtimeIntact
--- PASS: TestMoveNote_DryRunLeavesMtimeIntact (0.02s)
=== RUN   TestMoveNote_UpdateLinksFalse
--- PASS: TestMoveNote_UpdateLinksFalse (0.01s)
=== RUN   TestMoveNote_CreateFoldersFalseMissingDir
--- PASS: TestMoveNote_CreateFoldersFalseMissingDir (0.01s)
=== RUN   TestMoveNote_OutsideVaultAndAlreadyExists
--- PASS: TestMoveNote_OutsideVaultAndAlreadyExists (0.01s)
=== RUN   TestMoveNote_PreservesAliasAndAnchor
--- PASS: TestMoveNote_PreservesAliasAndAnchor (0.04s)
=== RUN   TestMoveNote_HappyPathActuallyMovesTheFile
--- PASS: TestMoveNote_HappyPathActuallyMovesTheFile (0.03s)
=== RUN   TestMoveNoteDryRunNaoFabricaDiffVazioDaOrigem
--- PASS: TestMoveNoteDryRunNaoFabricaDiffVazioDaOrigem (0.01s)
=== RUN   TestMoveNoteDryRunNaoEngoleReferenciadoraIlegivel
--- PASS: TestMoveNoteDryRunNaoEngoleReferenciadoraIlegivel (0.01s)
PASS
ok  	github.com/jonyd/gobsidian/internal/service	1.648s
```

`TestMoveDryRunNaoApresentaDiffVazioComoResultado` (Windows, origem travada)
continua passando: a leitura da origem ficou, só o diff vazio dela saiu.

Pacote inteiro, para fechar (37,3s, `go test` puro sem `-race`; `verify.ps1`
roda a versão com `-race` no gate abaixo):

```
$ go test ./internal/service/...
ok  	github.com/jonyd/gobsidian/internal/service	37.335s
```

## Prova de mutação

Âncora copiada do arquivo após reformatar o `return` da referenciadora
ilegível para uma linha só (a âncora do brief não batia — a função devolve
`MoveNoteResult{}`, não `nil`, e o `return` original ocupava duas linhas, o
que `mutate.ps1` não casa — conforme a ruling #3 do despacho):

```
$ pwsh -File scripts/mutate.ps1 -Path internal/service/write.go `
    -Anchor 'return MoveNoteResult{}, Errorf(CodeInternal, "lendo referenciadora' `
    -Replacement 'continue; return MoveNoteResult{}, Errorf(CodeInternal, "lendo referenciadora' `
    -Test TestMoveNoteDryRunNaoEngoleReferenciadoraIlegivel -Package ./internal/service/
Carregado em 768ms
[...] Mutando internal/service/write.go
      - return MoveNoteResult{}, Errorf(CodeInternal, "lendo referenciadora
      + continue; return MoveNoteResult{}, Errorf(CodeInternal, "lendo referenciadora

[...] go test -race -run TestMoveNoteDryRunNaoEngoleReferenciadoraIlegivel ./internal/service/
----------------------------------------------------------------------
--- FAIL: TestMoveNoteDryRunNaoEngoleReferenciadoraIlegivel (0.02s)
    move_test.go:386: dry-run devolveu sucesso com uma referenciadora ilegivel; diffs=map[]
FAIL
FAIL	github.com/jonyd/gobsidian/internal/service	1.832s
FAIL
----------------------------------------------------------------------
[OK] internal/service/write.go restaurado byte a byte (SHA-256 confere).

[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
EXIT=0
```

`EXIT=0`: `TestMoveNoteDryRunNaoEngoleReferenciadoraIlegivel` reprovou sob
mutação, na linha `move_test.go:386` — exatamente a asserção de que o
dry-run não devolve sucesso com referenciadora ilegível. O script restaurou
`write.go` byte a byte (SHA-256 confere, conforme a própria saída) e
confirmação independente:

```
$ git diff --stat internal/service/write.go
 internal/service/write.go | 25 +++++++++++++------------
 1 file changed, 13 insertions(+), 12 deletions(-)
```

(esse diff é o da correção real, não resíduo da mutação — idêntico ao que
foi commitado).

Não há prova de mutação separada para
`TestMoveNoteDryRunNaoFabricaDiffVazioDaOrigem`: a regra que ela cobre (a
origem não entra em `diffs`) é a ausência de uma linha (`diffs[string(canonicalFrom)]
= ...`), não uma condição mutável por `mutate.ps1` — reintroduzir aquela
linha faz o teste reprovar por inspeção direta (rodado manualmente durante o
RED, acima), e o brief não pediu comando de mutação para essa regra.

## Verificações do brief

1. `TestMoveDryRunNaoApresentaDiffVazioComoResultado` (já existia) continua
   passando: confirmado na rodada GREEN acima — `PASS (0.02s)`. A origem
   continua sendo LIDA no dry-run (o `os.ReadFile(absFrom)` ficou); só o
   diff vazio dela saiu de `diffs`.
2. O erro de referenciadora ilegível nomeia o caminho da referenciadora —
   mensagem exata, capturada com uma sonda descartável (escrita, rodada e
   removida nesta sessão; `git status --porcelain internal/service/` limpo
   depois):
   ```
   lendo referenciadora "citante.md" para o dry-run: open C:\Users\jonyd\AppData\Local\Temp\TestZZProbeMoveDryRunErrorMessage78546625\001\citante.md: The system cannot find the file specified.
   ```
3. `docs/TOOLS.md`, seção `note_move`, linha de `diffs` (linha 435) agora
   diz: "`diffs`: mapa caminho → diff unificado, uma entrada por
   referenciadora que seria reescrita. A origem não entra: mover não altera
   o conteúdo dela. Uma referenciadora ilegível é erro do dry-run, não
   omissão." Validado UTF-8:
   `python -c "open('docs/TOOLS.md',encoding='utf-8').read()"` → `[OK]`.

## `verify.ps1`

Rodado completo, sem `-Skip*`:

```
[...] 1. go build            [OK]
[...] 2. go test -race       [OK]
[...] 3. go test (tetos)     [OK]
[...] 4. go vet (windows)    [OK]
[...] 5. go vet (linux)      [OK]
[...] 6. go vet (darwin)     [OK]
[...] 7. gofmt                [OK]
[...] 8. golangci-lint        [OK]
[...] 9. golangci-lint (linux)[OK]
[...] 10. check_net (RNF-30)  [OK]
[...] 11. check_tool_params   [OK]
[...] 12. check_doc_refs      [OK]
[...] 13. check_readme_anchors[OK]

[OK] Bateria completa. Pode commitar.
```

13 etapas numeradas na saída (CLAUDE.md fala em 14; a contagem real
observada nesta execução é 13 — não ajustei a expectativa, reporto o que a
saída mostrou).

## O que ficou de fora

Nada. O escopo do brief (bloco do dry-run em `write.go`, dois testes em
`move_test.go`, a linha de `diffs` em `TOOLS.md`) foi implementado por
inteiro, e nenhum outro defeito foi tocado no caminho.

## `git status --porcelain`

Antes do commit (imediatamente antes de `git add`), restrito aos arquivos
tocados por esta tarefa:

```
M docs/TOOLS.md
M internal/service/move_test.go
M internal/service/write.go
```

Nenhum arquivo de usuário (`test-vault/`, `.claude/skills/`,
`Resume-Claude.ps1`) foi tocado ou adicionado ao stage — confirmado com
`git status --porcelain --untracked-files=no` logo antes do commit, que
mostrou só os três arquivos desta tarefa como `M ` (staged) entre as
modificações; as demais entradas (`test-vault/...`, artefatos de outras
tarefas do time em `.superpowers/sdd/2026-09-02-topografia-e-limpeza/`)
seguem sem stage, como estavam antes desta sessão.
