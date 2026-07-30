# Relatório Task 65: Tool `note_move` com Reescrita Fiel de Links (`internal/service/write.go` & `internal/mcpsrv/tools_write.go`)

- **Status**: DONE
- **Commit**: `feat(mcpsrv): note_move with faithful link rewriting`

## O Que Foi Implementado
- Implementado o método `MoveNote` em [internal/service/write.go](file:///C:/Users/jonyd/Projetos/Gobsidian/internal/service/write.go) com suporte completo a `from`, `to`, `update_links`, `create_folders` e `dry_run`.
- **Ordem Estrita de Operações (Movimentação por Último)**:
  1. Valida permissão `--read-only` e confinamento ao cofre de `from` e `to`.
  2. Coleta backlinks e mapeia substituições de link (`LinkReplacement`) com base em `idx.Backlinks(canonicalFrom)`.
  3. Se `dry_run: true`, calcula os diffs de todas as notas sem tocar o disco (preservando o `mtime`).
  4. Reescreve atomicamente cada nota afetada uma a uma (`WriteAtomic` + `PathLocker`). Se ocorrer erro em uma nota, a execução para e o erro reporta exatamente as notas reescritas até aquele momento.
  5. **Move o arquivo por último** (`WriteAtomic` de `from` para `to` + remoção do arquivo de origem).
- Registrada a ferramenta MCP `note_move` em [internal/mcpsrv/tools_write.go](file:///C:/Users/jonyd/Projetos/Gobsidian/internal/mcpsrv/tools_write.go). Sob `--read-only`, a ferramenta fica **ausente** de `ListTools`.
- Criada a suíte de testes em [internal/service/move_test.go](file:///C:/Users/jonyd/Projetos/Gobsidian/internal/service/move_test.go), incluindo o teste obrigatório `TestNoteMovePartialFailureReportsWhatWasApplied`.

## Evidência de TDD

### Comando do RED
`go test -v ./internal/service/ -run TestNoteMove` (antes de implementar `MoveNote`)
```
# github.com/jonyd/gobsidian/internal/service [github.com/jonyd/gobsidian/internal/service.test]
internal\service\move_test.go:38:13: svc.MoveNote undefined (type *service.Service has no field or method MoveNote)
FAIL	github.com/jonyd/gobsidian/internal/service [build failed]
```

### Comando do GREEN
`go test -v ./internal/service/ -run "TestNoteMove|TestMoveNote"` (após implementar `MoveNote`)
```
=== RUN   TestNoteMovePartialFailureReportsWhatWasApplied
--- PASS: TestNoteMovePartialFailureReportsWhatWasApplied (0.13s)
=== RUN   TestMoveNote_DryRunLeavesMtimeIntact
--- PASS: TestMoveNote_DryRunLeavesMtimeIntact (0.02s)
=== RUN   TestMoveNote_UpdateLinksFalse
--- PASS: TestMoveNote_UpdateLinksFalse (0.03s)
=== RUN   TestMoveNote_CreateFoldersFalseMissingDir
--- PASS: TestMoveNote_CreateFoldersFalseMissingDir (0.01s)
=== RUN   TestMoveNote_OutsideVaultAndAlreadyExists
--- PASS: TestMoveNote_OutsideVaultAndAlreadyExists (0.01s)
=== RUN   TestMoveNote_PreservesAliasAndAnchor
--- PASS: TestMoveNote_PreservesAliasAndAnchor (0.02s)
PASS
ok  	github.com/jonyd/gobsidian/internal/service	0.854s
```

## Prova de Mutação

### Comando de Mutação
`pwsh -File scripts/mutate.ps1 -Path internal/service/write.go -Anchor '// Sort keys of affectedNotes alphabetically for deterministic processing order' -Replacement '_ = os.MkdirAll(dirTo, 0755); _ = os.Rename(s.vault.Abs(canonicalFrom), absTo); // Sort keys of affectedNotes alphabetically for deterministic processing order' -Test TestNoteMovePartialFailureReportsWhatWasApplied -Package ./internal/service/`

### Saída Real do Script
```
[...] Mutando internal/service/write.go
      - // Sort keys of affectedNotes alphabetically for deterministic processing order
      + _ = os.MkdirAll(dirTo, 0755); _ = os.Rename(s.vault.Abs(canonicalFrom), absTo); // Sort keys of affectedNotes alphabetically for deterministic processing order

[...] go test -race -run TestNoteMovePartialFailureReportsWhatWasApplied ./internal/service/
----------------------------------------------------------------------
--- FAIL: TestNoteMovePartialFailureReportsWhatWasApplied (0.15s)
    move_test.go:80: o alvo.md saiu do lugar apesar da falha: GetFileAttributesEx C:\Users\jonyd\AppData\Local\Temp\TestNoteMovePartialFailureReportsWhatWasApplied3875304988\001\alvo.md: The system cannot find the file specified.
    move_test.go:83: o alvo foi movido para Novo/alvo.md apesar da falha antes do passo de move
FAIL
FAIL	github.com/jonyd/gobsidian/internal/service	1.078s
FAIL
----------------------------------------------------------------------
[OK] internal/service/write.go restaurado byte a byte (SHA-256 confere).
[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
```

## Verificações Exigidas pelo Brief

| Verificação | Resultado Real |
|---|---|
| `dry_run: true` deixa o `mtime` de todas as notas afetadas inalterado | OK (validado por `TestMoveNote_DryRunLeavesMtimeIntact` comparando `ModTime` antes e depois) |
| `update_links: false` move o arquivo sem alterar referências | OK (validado por `TestMoveNote_UpdateLinksFalse`) |
| `create_folders: false` recusa pasta inexistente com erro | OK (validado por `TestMoveNote_CreateFoldersFalseMissingDir` retornando `CodeFolderNotFound`) |
| `to` fora do cofre ou existente é recusado | OK (validado por `TestMoveNote_OutsideVaultAndAlreadyExists` retornando `CodePathOutsideVault` / `CodeNoteExists`) |
| Mover nota sem backlinks funciona com 0 reescritas | OK (`LinksUpdated: 0`, `Rewritten: nil`) |
| Alias e âncora preservados no cofre real | OK (validado por `TestMoveNote_PreservesAliasAndAnchor`) |
| Ponta a ponta: `ListTools` sob `--read-only` exclui `note_move` | OK (validado em `TestListTools_ReadOnlyTrue` e `TestListTools_ReadOnlyFalse`) |
| Mover nota para dentro de `.trash/` | Permitido: funciona como movimentação de pasta normal no cofre. |

## Auditoria de Relatório
`pwsh -File scripts/audit_reports.ps1 65` executado (0 achados).

## Bateria de Verificação
`pwsh -File scripts/verify.ps1`: **8/8 etapas VERDES**.
`pwsh -File scripts/build.ps1`: **Binário gerado com sucesso**.

## Arquivos Alterados
- `internal/service/write.go`
- `internal/service/move_test.go`
- `internal/mcpsrv/tools_write.go`
- `internal/mcpsrv/tools_write_test.go`
- `.superpowers/sdd/task-65-report.md`

## O Que Ficou de Fora
Nada.

## git status --porcelain
```
?? .superpowers/sdd/2026-07-25-gobsidian-v01/task-65-base.txt
?? .superpowers/sdd/task-65-report.md
 M internal/mcpsrv/tools_write.go
 M internal/mcpsrv/tools_write_test.go
?? internal/service/move_test.go
 M internal/service/write.go
```
