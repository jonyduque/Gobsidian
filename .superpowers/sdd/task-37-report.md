# Relatório de Execução - Task 37

- **Status**: DONE
- **Commit**: (Pendente de gravação após auditoria) - `feat(watcher): per-reason drop counters, coalesced count, and a real active flag`

---

## 1. Evidência de TDD

### Red
Falha inicial nos testes devido às alterações na assinatura de `Debounce`, `Apply` e na estrutura `WatchCounters`:
```text
internal\watcher\debounce_test.go:27:47: not enough arguments in call to Debounce
internal\watcher\apply_test.go:65:53: not enough arguments in call to watcher.Apply
tools_read_test.go:279: CallTool: type: <invalid reflect.Value> has type "null", want "object"
```

### Green
Saída real de `pwsh -File scripts/verify.ps1`:
```text
Carregado em 380ms
Carregado em 366ms
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

## 2. As 7 Provas de Mutação

| Contador | O que foi mutado | Teste que reprovou | Saída do Teste | Restauro |
| --- | --- | --- | --- | --- |
| `droppedChmod` | `w.droppedChmod.Add(1)` -> `_ = 1` | `TestCounters_DropReasons` | `FAIL: TestCounters_DropReasons (1.08s) counters_test.go:177: chmod drop count = 0, want 1` | [OK] |
| `droppedOutsideVault` | `w.droppedOutsideVault.Add(1)` -> `_ = 1` | `TestCounters_DropReasons` | `FAIL: TestCounters_DropReasons (1.08s) counters_test.go:180: outside_vault drop count = 0, want 1` | [OK] |
| `droppedExcluded` | `w.droppedExcluded.Add(1)` -> `_ = 1` | `TestCounters_DropReasons` | `FAIL: TestCounters_DropReasons (1.08s) counters_test.go:183: excluded drop count = 0, want 1` | [OK] |
| `droppedUnknownOp` | `w.droppedUnknownOp.Add(1)` -> `_ = 1` | `TestCounters_DropReasons` | `FAIL: TestCounters_DropReasons (1.08s) counters_test.go:186: unknown_op drop count = 0, want 1` | [OK] |
| `events_coalesced` | `coalesced.Add(1)` -> `_ = 1` | `TestCounters_Coalesced` | `FAIL: TestCounters_Coalesced (0.58s) counters_test.go:233: EventsCoalesced = 0, want 1` | [OK] |
| `reconciled_updated` | `reconciledUpdated.Add(int64(up))` -> `_ = up` | `TestCounters_ReconciledUpdatedAndRemoved` | `FAIL: TestCounters_ReconciledUpdatedAndRemoved (1.10s) counters_test.go:257: ReconciledUpdated = 0, want 1` | [OK] |
| `reconciled_removed` | `reconciledRemoved.Add(int64(rem))` -> `_ = rem` | `TestCounters_ReconciledUpdatedAndRemoved` | `FAIL: TestCounters_ReconciledUpdatedAndRemoved (1.12s) counters_test.go:275: ReconciledRemoved = 0, want 1` | [OK] |

---

## 3. Verificações do Brief

- **Saída de `go list -deps ./internal/watcher | Select-String service`**:
```text
(saída vazia - internal/watcher não importa internal/service)
```

- **Chamadores de `service.New`**:
```text
cmd/gobsidian/serve.go:116: svc := service.New(v, idx, watcherStats{w: w}, service.Options{...})
internal/mcpsrv/server_test.go:26: svc := service.New(v, idx, nil, service.Options{})
internal/mcpsrv/tools_read_test.go:33: svc := service.New(v, idx, nil, service.Options{})
internal/mcpsrv/tools_read_test.go:259: svc := service.New(v, idx, dummyWatchStats{}, service.Options{})
```

- **Diff em `docs/TOOLS.md`**:
```diff
- Com `include_runtime`: `runtime` (RSS, goroutines, gc) e objeto `watcher` (ausente se desligado) com os campos: `active`, `events_received`, `events_dropped`, `events_processed`, `events_skipped`, `reconciliations`.
+ Com `include_runtime`: `runtime` (RSS, goroutines, gc) e objeto `watcher` (ausente se desligado) com os campos: `active`, `events_received`, `events_dropped`, `events_dropped_by_reason`, `events_coalesced`, `events_processed`, `events_skipped`, `reconciliations`, `reconciled_updated`, `reconciled_removed`.
```

- **`active` real após cancelamento de `ctx`**: Provado no teste `TestCounters_ActiveState` em `internal/watcher/counters_test.go` (retorna `false` após `cancel()`).

---

## 4. O que ficou de fora

Nada. Todos os requisitos da Task 37 foram implementados e testados.

---

## 5. `git status --porcelain`

```text
 M .superpowers/sdd/2026-07-25-gobsidian-v01/progress.md
 M cmd/gobsidian/serve.go
 M docs/TOOLS.md
 M internal/mcpsrv/tools_read_test.go
 M internal/service/service.go
 M internal/watcher/apply.go
 M internal/watcher/apply_test.go
 M internal/watcher/counters.go
 M internal/watcher/counters_test.go
 M internal/watcher/debounce.go
 M internal/watcher/debounce_test.go
 M internal/watcher/filter.go
 M internal/watcher/filter_test.go
 M internal/watcher/overflow_test.go
 M internal/watcher/watcher.go
```
