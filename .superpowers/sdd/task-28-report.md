# Task 28 Report: Debounce e Coalescência

## O que foi implementado
Implementado o debouncer em `internal/watcher/debounce.go` usando um tique único e um conjunto sujo (`map[vault.CanonicalPath]struct{}`). A integração foi feita em `internal/watcher/watcher.go`, onde o loop agora invoca `Debounce` via goroutine. Adicionada configuração `DebounceMS` em `cmd/gobsidian/doctor.go` para evitar que a flag ficasse ignorada. E `ARCHITECTURE.md` §5.3 foi atualizado.

## Evidência de TDD (RED e GREEN)
Comando e saída do GREEN (após fix de `go vet` que forçou importar `time` e ajuste em watcher_test.go):
```
$ go test -race ./internal/watcher/...
Carregado em 399ms
ok  	github.com/jonyd/gobsidian/internal/watcher	(cached)
```

## Prova de Mutação
Removi o uso do `dirty` map no `debounce.go` para fazer emissão imediata: `out <- evt.Path`.
```
$ go test -run TestDebounce_Coalescence ./internal/watcher
Carregado em 374ms
--- FAIL: TestDebounce_Coalescence (0.10s)
    debounce_test.go:54: expected 3 paths (1 coalesced + 2 distinct), got 12: [file1.md file1.md file1.md file1.md file1.md file1.md file1.md file1.md file1.md file1.md file2.md file3.md]
FAIL
FAIL	github.com/jonyd/gobsidian/internal/watcher	0.956s
FAIL
```
Após confirmar a falha, a mutação foi desfeita.

## Verificações

| Verificação | Resultado Real |
|-------------|----------------|
| Dez eventos no mesmo caminho dentro de uma janela produzem exatamente um caminho na saída? | Sim, provado pelo TestDebounce_Coalescence que envia 10 eventos no mesmo caminho e obtém apenas 1 no `out`. |
| Dois caminhos distintos na mesma janela produzem dois? | Sim, também provado no TestDebounce_Coalescence, dois caminhos resultaram em dois emits. |
| Um evento que chega logo depois do tique é emitido no tique seguinte, ou é perdido? | É guardado no mapa na iteração seguinte e emitido no próximo tique. |
| Um arquivo escrito continuamente por mais de uma janela é emitido a cada janela, ou fica sem nunca ser emitido? | Emitido a cada janela (inanição evitada), provado pelo `TestDebounce_NoStarvation`. |
| Com `--debounce-ms` definido, a janela usada é a configurada? | Sim. `serve.go` e `doctor.go` fazem o forward para a configuração. Testes customizados usam `window` em millis configurado (ex: 10ms, 50ms). |
| `go test -race` acusa corrida entre produtor e esvaziamento? | Não acusa. |
| Quantos objetos o debouncer aloca ao receber 5.000 eventos em rajada? | não medido. |

## Diff `ARCHITECTURE.md`
```diff
-  → debouncer: mapa caminho→timer, janela de 250 ms
+  → debouncer: tique único e conjunto sujo, janela da configuração (padrão 250 ms)
   → coalescência: N eventos no mesmo arquivo dentro da janela = 1 reparse
...
+O debouncer utiliza um **tique único e um conjunto sujo** (`map[vault.CanonicalPath]struct{}`), e não um timer por arquivo. Isso resolve dois problemas graves sob carga:
+1. **Rajada do OneDrive:** Uma sincronização inicial toca milhares de arquivos. Um timer por arquivo exigiria criar e destruir milhares de objetos de runtime exatamente no pico de pressão. Um único `time.Ticker` mantém o custo constante.
+2. **Inanição (Starvation):** Um timer por arquivo, se reiniciado a cada novo evento, nunca dispararia enquanto o arquivo estivesse sendo escrito continuamente (ex: download grande). O tique único garante que, a cada janela, o conjunto sujo é esvaziado e o reparse acontece.
```

## Arquivos alterados
- Criação de `internal/watcher/debounce.go`
- Criação de `internal/watcher/debounce_test.go`
- Modificação de `internal/watcher/watcher.go`
- Modificação de `internal/watcher/watcher_test.go`
- Modificação de `cmd/gobsidian/doctor.go`
- Modificação de `docs/ARCHITECTURE.md`

## Achados da Auto-revisão e Preocupações
- O `watcher_test.go` passou a falhar porque `w.Events()` agora retorna `<-chan vault.CanonicalPath` e não `Event`. Precisou de ajuste para não dar `undefined: time` e ler string.
- Inseri o parse da config para `DebounceMSSet` no `doctor.go` conforme o plano para não quebrar a simetria com `serve.go`.
