# Relatório Task 51: Atualização Incremental do Índice de Busca via Watcher

- **Status**: DONE
- **Commit**: `fix(watcher): the inverted index never reached the watcher`

## Resumo das Mudanças
- Removido o parâmetro variádico `inv ...*search.Inverted` de `Apply` em `internal/watcher/apply.go`, tornando `searchInv *search.Inverted` um parâmetro regular e explícito.
- Adicionado a propriedade `inv *search.Inverted` à struct `Watcher` e atualizada a assinatura de `watcher.New` em `internal/watcher/watcher.go`.
- Repassado `w.inv` para `Apply` em `w.Run`, corrigindo o defeito crítico onde o índice invertido era sempre `nil` em execução de produção.
- Conectado `inv` a `watcher.New` em `cmd/gobsidian/serve.go`.
- Atualizados todos os pontos de chamada de `watcher.New` e `Apply` nos testes e comandos.
- Tratados os erros de `searchInv.Update` em `apply.go` com `log.Warn` em vez de engolir com `_ =`.
- Adicionado o teste de integração ponta-a-ponta `TestWatcherUpdatesSearchIndex` em `internal/watcher/watcher_test.go` cobrindo criação e remoção de notas em tempo real.

## Lista de Chamadores de `watcher.New`
Todos os chamadores atualizados:
- `cmd/gobsidian/serve.go:124`
- `internal/mcpsrv/tools_read_test.go:357`
- `internal/watcher/burst_test.go:27`
- `internal/watcher/counters_test.go:28`
- `internal/watcher/overflow_test.go:131`
- `internal/watcher/rename_test.go:421`
- `internal/watcher/watcher_test.go:29`, `99`, `133`, `162`, `212`, `232`

## Saída das Verificações por Grep
Comando: `git grep "watcher.New(" -- "*.go"`
Saída:
```
cmd/gobsidian/serve.go:	w, err := watcher.New(v, idx, inv, time.Duration(cfg.DebounceMS)*time.Millisecond, log)
internal/mcpsrv/tools_read_test.go:	w, err := watcher.New(v, idx, nil, 10*time.Millisecond, log)
internal/watcher/burst_test.go:	w, err := New(v, idx, nil, 10*time.Millisecond, log)
internal/watcher/counters_test.go:	w, err := New(v, idx, nil, 10*time.Millisecond, log)
internal/watcher/overflow_test.go:	w, err := New(v, idx, nil, 10*time.Millisecond, log)
internal/watcher/rename_test.go:	w, err := New(v, idx, nil, 20*time.Millisecond, log)
internal/watcher/watcher.go:func New(v *vault.Vault, idx *index.Index, inv *search.Inverted, debounce time.Duration, log *slog.Logger) (*Watcher, error) {
internal/watcher/watcher_test.go:	w, err := New(v, idx, nil, 10*time.Millisecond, log)
internal/watcher/watcher_test.go:	w, err := New(v, idx, nil, 10*time.Millisecond, log)
internal/watcher/watcher_test.go:	w, err := New(v, idx, nil, 10*time.Millisecond, log)
internal/watcher/watcher_test.go:	w, err := New(v, idx, nil, 10*time.Millisecond, log)
internal/watcher/watcher_test.go:	w, err := New(v, idx, nil, 10*time.Millisecond, log)
internal/watcher/watcher_test.go:	w, err := New(v, idx, inv, 10*time.Millisecond, log)
```

Comando: `git grep "\.\.\.\*search\." -- "*.go"`
Saída: (vazio - parâmetro variádico totalmente eliminado)

## Evidência de TDD
### RED
Comando:
`go test -v ./internal/watcher/ -run TestWatcherUpdatesSearchIndex`
Saída:
--- FAIL: TestWatcherUpdatesSearchIndex (3.11s)
    watcher_test.go:248: apos 3s, "prescricao" em "nova.md": presente=false, quer true — a busca nao acompanhou o cofre
FAIL

### GREEN
Comando:
`go test -v ./internal/watcher/ -run TestWatcherUpdatesSearchIndex`
Saída:
=== RUN   TestWatcherUpdatesSearchIndex
--- PASS: TestWatcherUpdatesSearchIndex (0.15s)
PASS
ok  	github.com/jonyd/gobsidian/internal/watcher	0.531s

## Provas de Mutação

### 1. Repasse do Índice Invertido (`w.inv -> nil`)
Comando:
`pwsh -File scripts/mutate.ps1 -Path internal/watcher/watcher.go -Anchor 'Apply(ctx, w.debounced, w.reconcile, w.idx, w.v, w.log, &w.processed, &w.skipped, &w.reconciledUpdated, &w.reconciledRemoved, w.inv)' -Replacement 'Apply(ctx, w.debounced, w.reconcile, w.idx, w.v, w.log, &w.processed, &w.skipped, &w.reconciledUpdated, &w.reconciledRemoved, nil)' -Test TestWatcherUpdatesSearchIndex -Package ./internal/watcher/`
Saída:
--- FAIL: TestWatcherUpdatesSearchIndex (3.11s)
    watcher_test.go:248: apos 3s, "prescricao" em "nova.md": presente=false, quer true — a busca nao acompanhou o cofre
FAIL
[OK] internal/watcher/watcher.go restaurado byte a byte (SHA-256 confere).

### 2. Remoção no Índice Invertido (`searchInv.Remove -> _ = path`)
Comando:
`pwsh -File scripts/mutate.ps1 -Path internal/watcher/apply.go -Anchor 'searchInv.Remove(string(path))' -Replacement '_ = path' -Test TestWatcherUpdatesSearchIndex -Package ./internal/watcher/`
Saída:
--- FAIL: TestWatcherUpdatesSearchIndex (3.13s)
    watcher_test.go:255: apos 3s, "prescricao" em "nova.md": presente=true, quer false — a busca nao acompanhou o cofre
FAIL
[OK] internal/watcher/apply.go restaurado byte a byte (SHA-256 confere).

## Verificações Exigidas pelo Brief
| Verificação | Resultado | Evidência |
|---|---|---|
| Todos os chamadores de `watcher.New` atualizados? | SIM | Lista de chamadores e `git grep` conferidos |
| Parâmetro variádico `...*search.Inverted` eliminado? | SIM | `git grep "\.\.\.\*search\."` devolve vazio |
| Nota criada fica imediatamente encontrável? | SIM | `TestWatcherUpdatesSearchIndex` (criação) |
| Nota removida deixa de ser encontrável? | SIM | `TestWatcherUpdatesSearchIndex` (remoção) |
| Erros de `searchInv.Update` logados com `Warn`? | SIM | Verificado em `apply.go:63` e `apply.go:101` |

## Arquivos Alterados
- `cmd/gobsidian/serve.go`
- `internal/mcpsrv/tools_read_test.go`
- `internal/watcher/apply.go`
- `internal/watcher/apply_test.go`
- `internal/watcher/burst_test.go`
- `internal/watcher/counters_test.go`
- `internal/watcher/overflow_test.go`
- `internal/watcher/rename_test.go`
- `internal/watcher/watcher.go`
- `internal/watcher/watcher_test.go`
- `.superpowers/sdd/task-51-report.md`

## O Que Ficou de Fora
Nada.
