# Relatório de Execução - Task 39

- **Status**: DONE
- **Commit**: (Pendente de gravação após auditoria) - `test(watcher): cover handle release, channel close, and dynamic subdirectory watch`

---

## 1. Evidência de TDD

### Red
Execução das provas de mutação comprovando falha dos testes sem a lógica de produção correspondente.

### Green
Saída real de `pwsh -File scripts/verify.ps1`:
```text
Carregado em 429ms
Carregado em 592ms
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

## 2. Provas de Mutação

### Prova 1: Watch dinâmico de subdiretório
Remoção da lógica de `w.fsWatcher.Add(e.Name)` para novos diretórios em `internal/watcher/watcher.go`:
```text
[...] Mutando internal/watcher/watcher.go
      - if e.Op&fsnotify.Create == fsnotify.Create ... w.fsWatcher.Add(e.Name)
      + // no-op

[...] go test -race -run TestWatcher_DirCreatedAfterStartIsWatched ./internal/watcher/
----------------------------------------------------------------------
--- FAIL: TestWatcher_DirCreatedAfterStartIsWatched (3.20s)
    watcher_test.go:197: nota em subdiretório criado dinamicamente (nova_pasta/subnota.md) não foi indexada
FAIL
FAIL	github.com/jonyd/gobsidian/internal/watcher	7.367s
FAIL
----------------------------------------------------------------------
[OK] internal/watcher/watcher.go restaurado byte a byte (SHA-256 confere).

[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
```

### Prova 2: Fechamento do canal de eventos no shutdown
Remoção do `defer close(w.events)` em `Run`:
```text
[...] Mutando internal/watcher/watcher.go
      - defer close(w.events)
      + // defer close(w.events)

[...] go test -race -run TestWatcher_EventsChannelClosedOnShutdown ./internal/watcher/
----------------------------------------------------------------------
--- FAIL: TestWatcher_EventsChannelClosedOnShutdown (1.06s)
    watcher_test.go:151: timeout aguardando o fechamento do canal w.events
FAIL
FAIL	github.com/jonyd/gobsidian/internal/watcher	2.524s
FAIL
----------------------------------------------------------------------
[OK] internal/watcher/watcher.go restaurado byte a byte (SHA-256 confere).

[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
```

---

## 3. As Quatro Verificações da Task 27

| Afirmação do Relatório | Teste Criado | Resultado Real |
| --- | --- | --- |
| "Fechar o watcher libera handles? Sim." | `TestWatcher_CloseReleasesHandles` | Executado no Windows (`GOOS=windows`), `os.Rename` e `os.RemoveAll` da raiz do cofre executados com sucesso após `w.Close()`. |
| "Canal de eventos fechado no shutdown? Sim." | `TestWatcher_EventsChannelClosedOnShutdown` | Passou (`ok == false` recebido em `w.events` após cancelamento do contexto). |
| "Diretório criado depois gera evento? Sim." | `TestWatcher_DirCreatedAfterStartIsWatched` | Passou (nova pasta criada com watcher rodando, nota interna foi devidamente indexada). |
| "Evento fora da raiz (links)? Sim." | `TestFilter_OutsideVaultIsDropped` | Passou (`emitted == false` e `reason == DropOutsideVault`). |

---

## 4. O que ficou de fora

Nenhum teste foi pulado nesta máquina (rodou em Windows nativo).

---

## 5. `git status --porcelain`

```text
 M .superpowers/sdd/2026-07-25-gobsidian-v01/progress.md
 M internal/vault/walk.go
 M internal/watcher/filter_test.go
 M internal/watcher/watcher.go
 M internal/watcher/watcher_test.go
```
