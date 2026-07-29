# Relatório de Execução - Task 32: Exposição de métricas do watcher no `vault_stats`

- **Status**: DONE
- **Commit**: `b5b2a9f` (Com métricas desdobradas e adaptadas nas Tasks 32 e 37)

---

## O que foi implementado

Exposição de métricas operacionais do observador de arquivos no endpoint/ferramenta `vault_stats`:
1. Definição da interface `WatcherStats` e struct `WatchCounters` em `internal/service/service.go`.
2. Instrumentação de contadores atômicos em `internal/watcher/watcher.go` (`events_received`, `events_dropped`, `events_processed`, `events_skipped`, `reconciliations`).
3. Adição dos campos do watcher na resposta da ferramenta `vault_stats` em `internal/mcpsrv/tools_read.go` quando `include_runtime: true`.
4. Atualização da documentação em `docs/TOOLS.md`.

---

## Evidência de TDD

### RED
Sem os contadores atômicos e a injeção da interface no serviço, o bloco `watcher` em `vault_stats` retornava nulo ou ausente.

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

### Mutation 1: `events_received`
```text
--- FAIL: TestCounters_EventsReceived (1.08s)
    counters_test.go:59: EventsReceived = 0, want > 0
FAIL
FAIL	github.com/jonyd/gobsidian/internal/watcher	1.662s
FAIL
```

### Mutation 2: `events_dropped`
```text
--- FAIL: TestCounters_EventsDropped (1.08s)
    counters_test.go:80: EventsDropped = 0, want > 0
FAIL
FAIL	github.com/jonyd/gobsidian/internal/watcher	1.663s
FAIL
```

### Mutation 3: `reconciliations`
```text
time=2026-07-28T22:46:23.827-03:00 level=WARN msg="Overflow de fsnotify detectado, reconciliação agendada"
time=2026-07-28T22:46:23.827-03:00 level=INFO msg="Iniciando reconciliação completa do cofre devido a overflow"
time=2026-07-28T22:46:23.848-03:00 level=WARN msg="Reconciliação concluída" updated=0 removed=0 skipped=0
--- FAIL: TestCounters_Reconciliations (0.16s)
    counters_test.go:158: Reconciliations = 0, want > 0
FAIL
FAIL	github.com/jonyd/gobsidian/internal/watcher	0.720s
FAIL
```

### Mutation 4: `events_processed`
```text
--- FAIL: TestCounters_EventsProcessed (1.08s)
    counters_test.go:101: EventsProcessed = 0, want > 0
FAIL
FAIL	github.com/jonyd/gobsidian/internal/watcher	2.588s
FAIL
```

### Mutation 5: `events_skipped`
```text
--- FAIL: TestCounters_EventsSkipped (1.10s)
    counters_test.go:141: EventsSkipped = 0, want > 0
FAIL
FAIL	github.com/jonyd/gobsidian/internal/watcher	2.611s
FAIL
```

---

## Verificações do Brief

- **Exposição de métricas**: `vault_stats` expõe o bloco `watcher` com contadores operacionais reais sob a flag `include_runtime: true`.
- **Desempenho**: O uso de `atomic.Int64` na contabilização do watcher não introduziu locks nem degradação de throughput.
- **Documentação**: `docs/TOOLS.md` atualizado detalhando cada um dos contadores.

---

## O que ficou de fora

Desdobramento por motivo de descarte (`events_dropped_by_reason`) e contadores de reconciliação (`reconciled_updated`, `reconciled_removed`) foram expandidos na Task 37.

---

## `git status --porcelain`

```text
 M .superpowers/sdd/task-32-report.md
```
