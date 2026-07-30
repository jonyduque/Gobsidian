# Relatório Task 54: `internal/writer/lock.go` — Trava por Caminho

- **Status**: DONE
- **Commit**: `feat(writer): per-path write lock`

## Resumo das Mudanças
- Criada a struct `PathLocker` em `internal/writer/lock.go` oferecendo controle de concorrência por caminho de nota com contagem de referências.
- Implementada a normalização insensível a maiúsculas/minúsculas via `strings.ToLower(string(path))` garantindo que `Civil/A.md` e `civil/a.md` compartilhem a mesma trava em sistemas Windows/canonical.
- Implementada desalocação automática do mapa interno quando o `refCount` chega a 0, evitando vazamento de memória acumulada.
- Criados testes unitários em `internal/writer/lock_test.go` cobrindo prevenção de lost update, suporte a casing insensível, paralelismo entre caminhos distintos e ausência de vazamento de memória.

## Evidência de TDD

### RED
Comando:
`go test -v ./internal/writer/...` (antes de criar a trava)
Saída:
FAIL (Pacote internal/writer não existia).

### GREEN
Comando:
`go test -v ./internal/writer/...`
Saída:
=== RUN   TestPathLocker_SamePathLostUpdate
--- PASS: TestPathLocker_SamePathLostUpdate (0.07s)
=== RUN   TestPathLocker_SamePathCasing
--- PASS: TestPathLocker_SamePathCasing (0.05s)
=== RUN   TestPathLocker_DifferentPathsParallel
--- PASS: TestPathLocker_DifferentPathsParallel (0.00s)
=== RUN   TestPathLocker_NoMemoryLeak
--- PASS: TestPathLocker_NoMemoryLeak (0.09s)
PASS
ok  	github.com/jonyd/gobsidian/internal/writer	0.963s

## Provas de Mutação

### 1. Mutação de Trava Global (`key := normalizeKey(path) -> key := "global"`)
Comando:
`pwsh -File scripts/mutate.ps1 -Path internal/writer/lock.go -Anchor 'key := normalizeKey(path)' -Replacement 'key := "global"' -Test TestPathLocker_DifferentPathsParallel -Package ./internal/writer/`
Saída:
--- FAIL: TestPathLocker_DifferentPathsParallel (1.00s)
    lock_test.go:94: p2 nao adquiriu a trava em caminho diferente (trava global indevida)
FAIL
FAIL	github.com/jonyd/gobsidian/internal/writer	2.161s
FAIL
[OK] internal/writer/lock.go restaurado byte a byte (SHA-256 confere).

### 2. Mutação sem Trava (`entry.mu.Lock() -> _ = entry`)
Comando:
`pwsh -File scripts/mutate.ps1 -Path internal/writer/lock.go -Anchor 'entry.mu.Lock()' -Replacement '_ = entry' -Test TestPathLocker_SamePathLostUpdate -Package ./internal/writer/`
Saída:
WARNING: DATA RACE
Write at 0x00c00000c418 by goroutine 10...
Previous read at 0x00c00000c418 by goroutine 45...
FAIL	github.com/jonyd/gobsidian/internal/writer	0.768s
FAIL
[OK] internal/writer/lock.go restaurado byte a byte (SHA-256 confere).

## Verificações Exigidas pelo Brief
| Verificação | Resultado | Evidência |
|---|---|---|
| Duas escritas concorrentes na mesma nota esperam e preservam dados? | SIM | `TestPathLocker_SamePathLostUpdate` (passed under -race) |
| Duas escritas concorrentes em notas diferentes rodam em paralelo? | SIM | `TestPathLocker_DifferentPathsParallel` (< 10ms) |
| `Civil/A.md` e `civil/a.md` pegam a mesma trava? | SIM | `TestPathLocker_SamePathCasing` |
| Quantas entradas sobram após 100 escritas em caminhos distintos? | 0 | `TestPathLocker_NoMemoryLeak` (`Count() == 0`) |

## Arquivos Alterados
- `internal/writer/lock.go`
- `internal/writer/lock_test.go`
- `.superpowers/sdd/task-54-report.md`

## O Que Ficou de Fora
Nada.
