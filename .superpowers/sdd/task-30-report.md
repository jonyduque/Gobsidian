# Relatório de Execução - Task 30: Recuperação de `ErrEventOverflow` por reconciliação

- **Status**: DONE
- **Commit**: `5afbcce` (Com testes de reconciliação determinísticos atualizados na Task 34)

---

## Evidência TDD (RED e GREEN)

### RED
Testes da versão inicial deixavam o watcher rodando, fazendo com que o pipeline normal absorvesse os eventos antes da reconciliação.

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

Prova real produzida na Task 34 (com mutação do corpo de `Reconcile` para no-op sem watcher rodando):
```text
[...] Mutando internal/watcher/overflow.go
      - func Reconcile(...) (updated, removed, skipped int) {
      + func Reconcile(...) (updated, removed, skipped int) { return 0, 0, 0 }

[...] go test -race -run TestReconcile_CorrectsLostEvents ./internal/watcher/
----------------------------------------------------------------------
--- FAIL: TestReconcile_CorrectsLostEvents (0.04s)
    overflow_test.go:42: updated = 0, want 1
    overflow_test.go:45: removed = 0, want 1
FAIL
FAIL	github.com/jonyd/gobsidian/internal/watcher	1.050s
FAIL
----------------------------------------------------------------------
[OK] internal/watcher/overflow.go restaurado byte a byte (SHA-256 confere).

[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
```

---

## Verificações Adicionais

- **Eventos genuinamente perdidos:** `TestReconcile_CorrectsLostEvents` (escrito na Task 34) roda **sem watcher** e desliga o caminho normal para provar isoladamente a mecânica de fallback.
- **Cancelamento (ctx):** `TestReconcile_CtxCancelStopsEarly` confirma que `context.Canceled` interrompe a varredura sem registrar log de erro alarmante.
- **Overflow durante Reconciliação:** Garantido por envio com default discard em canal bufferizado de tamanho 1 (`case w.reconcile <- struct{}{}: default:`).
- **Cofre Inacessível:** `TestReconcile_VaultGoneLeavesIndexIntact` confirma que erro na raiz do cofre preserva o índice intacto sem apagar as notas existentes.
- **Plataformas (macOS / BSD):** O backend `kqueue` do fsnotify v1.10.1 nunca emite `ErrEventOverflow`. Lacuna documentada em `ARCHITECTURE.md` §5.3 e `WINDOWS.md`.

---

## O que ficou de fora

Nada. A cobertura de mutação foi reescrita e verificada deterministicamente.

---

## `git status --porcelain`

```text
 M .superpowers/sdd/task-30-report.md
```
