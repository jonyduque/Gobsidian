# Relatório de Execução - Task 36

- **Status**: DONE
- **Commit**: (Pendente de gravação após auditoria) - `fix(index): MoveNote refreshes stat and matches Remove+Replace`

---

## 1. Evidência de TDD

### Red
Execução do teste de mutação onde a chamada `idx.MoveNote` em `internal/watcher/apply.go` foi substituída por `_ = rename`:
```text
[...] Mutando internal/watcher/apply.go
      - idx.MoveNote(v, rename.From, rename.To)
      + _ = rename

[...] go test -race -run TestWatcher_RenameEndToEnd ./internal/watcher/
----------------------------------------------------------------------
--- FAIL: TestWatcher_RenameEndToEnd (3.03s)
    rename_test.go:434: origem.md ainda no índice após rename end-to-end
    rename_test.go:438: destino.md ausente no índice após rename end-to-end
FAIL
FAIL	github.com/jonyd/gobsidian/internal/watcher	3.882s
FAIL
----------------------------------------------------------------------
[OK] internal/watcher/apply.go restaurado byte a byte (SHA-256 confere).
[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
```

### Green
Saída real de `pwsh -File scripts/verify.ps1`:
```text
Carregado em 574ms
Carregado em 593ms
[...] 1. go build
[OK] go build
[...] 2. go test -race
[OK] go test -race
[...] 3. go vet (windows)
[OK] go vet (windows)
[...] 4. go vet (linux)
[OK] go vet (linux)
[...] 5. go vet (darwin)
[OK] go vet (darwin)
[...] 6. gofmt
[OK] gofmt
[...] 7. check_net (RNF-30)
[OK] check_net (RNF-30)

[OK] Bateria completa. Pode commitar.
```

---

## 2. Prova de Mutação por Estrutura Mutada em MoveNote

Tabela de validação de mutação para cada uma das 8 estruturas modificadas por `MoveNote`:

| Estrutura | O que foi mutado | Teste que reprovou | Saída do Teste | Restauro |
| --- | --- | --- | --- | --- |
| `notes` | `delete(ix.notes, oldPath)` -> `_ = oldPath` | `TestMoveNote_UpdatesNotes` | `FAIL: TestMoveNote_UpdatesNotes (0.01s) move_test.go:135: notes structure survived mutation: a.md still present` | [OK] |
| `lowerPath` | `delete(ix.lowerPath, ...)` -> `_ = oldPath` | `TestMoveNote_UpdatesLowerPath` | `FAIL: TestMoveNote_UpdatesLowerPath (0.01s) move_test.go:153: lowerPath structure survived mutation: a.md -> a.md` | [OK] |
| `byName` | `delete(ix.byName, string(oldBase))` -> `_ = oldBase` | `TestMoveNote_UpdatesByName` | `FAIL: TestMoveNote_UpdatesByName (0.01s) move_test.go:171: byName structure survived mutation: a.md still in byName ([a.md])` | [OK] |
| `tags` | `paths[i] = newPath` -> `paths[i] = oldPath` | `TestMoveNote_UpdatesTags` | `FAIL: TestMoveNote_UpdatesTags (0.01s) move_test.go:190: tags structure survived mutation: tag mylabel -> [a.md]` | [OK] |
| `byAlias` | `al[i] = newPath` -> `al[i] = oldPath` | `TestMoveNote_UpdatesByAlias` | `FAIL: TestMoveNote_UpdatesByAlias (0.02s) move_test.go:206: byAlias structure survived mutation: alias STJ -> [a.md]` | [OK] |
| `incoming backlinks` | `ix.backlinks[newPath] = bls` -> `delete(ix.backlinks, newPath)` | `TestMoveNote_UpdatesIncomingBacklinks` | `FAIL: TestMoveNote_UpdatesIncomingBacklinks (0.01s) move_test.go:225: incoming backlinks: b.md backlinks = []` | [OK] |
| `outgoing backlinks` | `bls[i].From = newPath` -> `bls[i].From = oldPath` | `TestMoveNote_UpdatesOutgoingBacklinks` | `FAIL: TestMoveNote_UpdatesOutgoingBacklinks (0.01s) move_test.go:242: outgoing backlinks survived mutation: c.md backlinks = [{a.md    wikilink}]` | [OK] |
| `reprocessLinksLocked` | `ix.reprocessLinksLocked()` -> `_ = ix` | `TestMoveNote_ReprocessesBrokenLinks` | `FAIL: TestMoveNote_ReprocessesBrokenLinks (0.02s) move_test.go:261: d.md link state = ok, expected LinkTargetMissing` | [OK] |

---

## 3. As Verificações do Brief

- **MoveNote Signature & Stat**: `MoveNote` agora aceita `(v *vault.Vault, oldPath, newPath vault.CanonicalPath)` e executa `os.Stat(v.Abs(newPath))` para atualizar `ModTime` e `Size`. Em caso de falha do `os.Stat`, `ModTime` é zerado para forçar reindexação posterior.
- **TestMoveNote_EquivalentToRemoveReplace**: Passou no comparativo estrutura a estrutura entre um índice que recebeu `MoveNote` e outro que recebeu `Remove` + `Replace`.
- **TestWatcher_RenameEndToEnd**: Passou com o watcher real rodando em background e disparando a correlação de rename.
- **Provas de Mutação**: Todas as 8 mutações em `update.go` e a mutação em `apply.go` resultaram no código de saída `0` com falha nos testes específicos.

---

## 4. O que ficou de fora

Nada. Todos os requisitos da Task 36 foram implementados e testados.

---

## 5. `git status --porcelain`

```text
 M .superpowers/sdd/2026-07-25-gobsidian-v01/progress.md
 M internal/index/resolve_test.go
 M internal/index/update.go
 M internal/watcher/apply.go
 M internal/watcher/rename_test.go
?? internal/index/move_test.go
```
