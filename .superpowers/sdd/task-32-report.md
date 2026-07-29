# Task 32: Mutation Proofs

## Mutation: events_received
```
--- FAIL: TestCounters_EventsReceived (1.08s)
    counters_test.go:59: EventsReceived = 0, want > 0
FAIL
FAIL	github.com/jonyd/gobsidian/internal/watcher	1.662s
FAIL
```

## Mutation: events_dropped
```
--- FAIL: TestCounters_EventsDropped (1.08s)
    counters_test.go:80: EventsDropped = 0, want > 0
FAIL
FAIL	github.com/jonyd/gobsidian/internal/watcher	1.663s
FAIL
```

## Mutation: reconciliations
```
time=2026-07-28T22:46:23.827-03:00 level=WARN msg="Overflow de fsnotify detectado, reconciliação agendada"
time=2026-07-28T22:46:23.827-03:00 level=INFO msg="Iniciando reconciliação completa do cofre devido a overflow"
time=2026-07-28T22:46:23.848-03:00 level=WARN msg="Reconciliação concluída" updated=0 removed=0 skipped=0
--- FAIL: TestCounters_Reconciliations (0.16s)
    counters_test.go:158: Reconciliations = 0, want > 0
FAIL
FAIL	github.com/jonyd/gobsidian/internal/watcher	0.720s
FAIL
```

## Mutation: events_processed
```
--- FAIL: TestCounters_EventsProcessed (1.08s)
    counters_test.go:101: EventsProcessed = 0, want > 0
FAIL
FAIL	github.com/jonyd/gobsidian/internal/watcher	2.588s
FAIL
```

## Mutation: events_skipped
```
--- FAIL: TestCounters_EventsSkipped (1.10s)
    counters_test.go:141: EventsSkipped = 0, want > 0
FAIL
FAIL	github.com/jonyd/gobsidian/internal/watcher	2.611s
FAIL
```
