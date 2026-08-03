# Relatório de Execução - Task 29

- **Status**: DONE
- **Commit**: `e2c38e1` (Atualizado com evidências reais do M2.1)

---

## Evidência de TDD

### RED
Execução inicial sem a verificação de `mtime` e `size` em `internal/watcher/apply.go` processava o arquivo sem incrementar o contador `skipped`.

### GREEN
Saída real de `pwsh -File scripts/verify.ps1`:
```text
Carregado em 374ms
Carregado em 339ms
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

## Prova de Mutação

Mutação em `internal/watcher/apply.go` (desativando o atalho de mtime+size):
```text
[...] Mutando internal/watcher/apply.go
      - if info.ModTime() == note.ModTime && info.Size() == note.Size
      + if false

[...] go test -race -run TestApply ./internal/watcher/
----------------------------------------------------------------------
--- FAIL: TestApply (0.05s)
    apply_test.go:68: skipped = 0, want 1
FAIL
FAIL	github.com/jonyd/gobsidian/internal/watcher	1.120s
FAIL
----------------------------------------------------------------------
[OK] internal/watcher/apply.go restaurado byte a byte (SHA-256 confere).

[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
```

---

## Verificações do Brief

- **Zero reparses em arquivo inalterado**: Provado via `TestApply` em `apply_test.go` (valida `skipped == 1` sem reler o conteúdo da nota).
- **Arquivo novo processado**: Provado via `TestApply` e `TestWatcher_Burst` (500 notas criadas em rajada são indexadas).
- **Arquivo removido processado**: Provado via `TestApply` (valida remoção do índice) e `TestVaultStatsReflectsWatcherUpdate`.
- **Proteção contra erro de permissão**: O comando `os.Stat` captura erros de I/O/permissão e zera `ModTime` (`time.Time{}`), forçando reindexação segura sem pânico ou travamento do loop.
- **Resiliência a erro de parse**: Erro retornado por `parser.Parse` não aborta a varredura nem o watcher; o arquivo é mantido no mapa com metadados básicos.
- **Consistência em rajada de 500 arquivos**: `TestWatcher_Burst` em `burst_test.go` confirma que todas as 500 notas foram indexadas no repositório.

---

## O que ficou de fora

Nada. Todos os itens da Task 29 foram auditados e documentados.

---

## `git status --porcelain`

```text
 M .superpowers/sdd/task-29-report.md
```
