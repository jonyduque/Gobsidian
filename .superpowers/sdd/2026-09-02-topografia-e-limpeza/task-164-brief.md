### Task 164: Sleep como única sincronização nos testes do watcher; mtime em `delete_test`

**Files:**
- Modify: `internal/watcher/counters_test.go:206-227,:265`
- Modify: `internal/watcher/debounce_test.go:43-46`
- Modify: `internal/watcher/burst_test.go:39`
- Modify: `internal/watcher/rename_test.go:430-437`
- Modify: `internal/service/delete_test.go:148`

**Interfaces:**
- Consumes: o mecanismo de espera que `overflow_test.go` usa (o arquivo registra em `:140-145` por que escrever em `fsWatcher.Errors` foi removido — DATA RACE em kqueue). Ler antes de tocar `counters_test.go`.

- [ ] **Step 1: Inventariar o que cada sleep espera**

Para cada sítio, escrever no relatório: "espera por X; o sinal observável de X é Y". Exemplos de Y que existem no pacote: contador exposto pelo watcher (`Stats()`), canal de eventos aplicados, `idx.Get` do caminho. Se não houver sinal observável, o sleep fica **e o comentário diz por quê** — mas o sleep passa a ser um `esperarAte(t, cond, vaulttest.Prazo)` com polling de 10 ms, não um valor fixo.

`esperarAte` — se o pacote já tem um (procurar `waitFor`, `eventually`, `esperar`), usar; senão criar em `internal/watcher/espera_test.go`:

```go
// esperarAte faz polling ate cond ser verdadeira ou o prazo estourar. Um
// sleep fixo e a pior das duas coisas: lento quando a condicao ja vale, e
// falso quando a maquina esta carregada.
func esperarAte(t *testing.T, cond func() bool, prazo time.Duration) {
	t.Helper()
	fim := time.Now().Add(prazo)
	for time.Now().Before(fim) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("condicao nao valeu em %v", prazo)
}
```

- [ ] **Step 2: `counters_test.go:206-227` — não escrever em `fsWatcher.Errors`**

O teste escreve em `fsWatcher.Errors` para simular erro — `overflow_test.go:140-145` registra que isso é DATA RACE em kqueue e foi removido lá. Trocar pelo mecanismo que `overflow_test.go` usa hoje (ler). Se não houver como injetar erro sem tocar o canal, apagar as asserções de erro e registrar no relatório.

- [ ] **Step 3: `rename_test.go:430-437` — espera de arranque**

Único teste do pacote sem espera de arranque do watcher (os outros usam `esperarPronto` ou equivalente — ler um vizinho e copiar). Acrescentar.

- [ ] **Step 4: `delete_test.go:148` — mtime**

Compara mtime antes/depois sem sleep; em NTFS com resolução de 100 ns pode até funcionar, mas o teste é inerte se o valor for igual. Trocar por `os.Chtimes` explícito com `-2 s` antes da operação e afirmar `depois.After(antes)`; ou, se o que se quer provar é "o arquivo foi reescrito", afirmar sobre o conteúdo. Ler o teste e escolher; dizer qual no relatório.

- [ ] **Step 5: Rodar 20 vezes sob `-race`**

Run: `go test -race -count=20 ./internal/watcher/ ./internal/service/ -run 'TestCounters|TestDebounce|TestBurst|TestRename|TestDelete' 2>&1 | tail -5` — Expected: `ok` 20 vezes, sem `DATA RACE`.

- [ ] **Step 6: Gate e commit**

Run: `pwsh -File scripts/verify.ps1` — Expected: verde.

```bash
git add internal/watcher/counters_test.go internal/watcher/debounce_test.go internal/watcher/burst_test.go internal/watcher/rename_test.go internal/watcher/espera_test.go internal/service/delete_test.go
git commit -m "test(watcher): wait on observable signals instead of fixed sleeps; explicit mtime in delete test"
```

#### Verificações

- Tabela no relatório: sítio, "espera por", "sinal observável", ação.
- `grep -n "time.Sleep" internal/watcher/*_test.go` — cada ocorrência restante tem comentário na linha de cima dizendo por que não há sinal observável.
- `-count=20 -race` colado.
- `verify.ps1` verde.

#### Regras de execução

- Nunca `git checkout/restore/stash/clean/reset`.
- Nenhuma escrita em `fsWatcher.Errors` ou outro canal interno do fsnotify.
- Nenhuma alteração em `internal/watcher/*.go` de produto — se um sinal observável precisar existir e não existe, `BLOCKED` com o sinal proposto; a próxima sessão decide.

#### Comando de mutação

Esta tarefa não tem prova de mutação por `mutate.ps1`: ela troca sincronização de teste, não regra de produto. A prova é o `-count=20 -race` verde e o inventário de sinais.

#### Contrato de relatório

`task-164-report.md`: status, SHA, a tabela de sinais, a saída do `-count=20`, última linha do `verify.ps1`.

