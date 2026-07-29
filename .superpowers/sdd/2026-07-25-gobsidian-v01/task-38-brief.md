### Task 38: `--debounce-ms` recusa zero na configuração

Implementa a decisão fechada 1. Move a recusa para onde ela pertence e apaga o fallback silencioso.

#### Onde isto encaixa

`config` decidiu, com comentário e dois testes nomeados, que zero era valor válido significando "sem debounce". O watcher desfez isso dois pacotes depois. A decisão nova é recusar zero — e o ponto de recusa é a configuração, não o consumidor.

#### A evidência medida do defeito

`internal/watcher/debounce.go:16-18`:

```go
if window <= 0 {
	window = 250 * time.Millisecond // default
}
```

`config.DefaultDebounceMS = 250` já existe em `internal/config/defaults.go:8`, e `internal/config/config_test.go` tem dois casos nomeados garantindo que o zero explícito sobrevive ao `Load`. O fallback recodifica a constante e apaga uma escolha do usuário — é a armadilha catalogada de "flag que não distingue omitida de definida com zero", consumada no consumidor em vez de no parser.

#### O que implementar

**1. `internal/config/config.go:153-161`** — inverta a regra e o comentário:

```go
// validateDebounceMS aplica a mesma regra de aceitacao independente da
// origem do valor (flag ou env). Zero e recusado: sem coalescencia cada
// evento vira um lote de um caminho so, e a correlacao de rename — que
// exige uma remocao E uma criacao no MESMO lote — para de detectar
// qualquer rename. Servidor sem debounce nao e configuracao que se possa
// pedir por engano, entao a recusa mora aqui e nao no watcher.
func validateDebounceMS(n int) error {
	if n < 1 {
		return fmt.Errorf("valor invalido %d (use um inteiro >= 1)", n)
	}
	return nil
}
```

**2. `internal/config/config.go:145`** — a mensagem de `parseDebounceMS` passa a dizer `use um inteiro >= 1`.

**3. `internal/config/config_test.go:93-104`** — os casos `"debounce explicit zero reachable from env"` e `"...from flag"` viram casos de recusa: `wantErr: true`, renomeados para `"debounce zero rejected from env"` e `"debounce zero rejected from flag"`. **Não apague os casos.** O zero continua sendo o valor que precisa de teste nomeado; só mudou de veredito.

**4. `internal/watcher/debounce.go:16-18`** — apague o bloco inteiro. Com a config recusando `< 1`, a janela chega sempre positiva e `time.NewTicker` nunca vê duração não positiva. **Não acrescente defesa em profundidade aqui:** um segundo fallback silencioso é exatamente o que estamos removendo.

**5. Docs.** `docs/WINDOWS.md` §4.2 e a nota de `docs/TOOLS.md` sobre `--debounce-ms` passam a dizer que o menor valor aceito é 1.

#### Armadilhas já pagas neste projeto que se aplicam aqui

- **Flag booleana ou inteira não distingue "omitida" de "definida com zero".** `DebounceMSSet` continua sendo quem faz essa distinção, e continua tendo que ser preenchido com `cmd.Flags().Changed("debounce-ms")` em **todos** os subcomandos. Confirme em `serve` **e** em `doctor` — esquecer em um faz a flag virar no-op silencioso.
- **`str.replace` que não casa não falha.** Se editar por script, ponha `assert` do texto-âncora antes de substituir e confira o resultado no disco depois.

#### Verificações além dos passos

- **Prova de mutação obrigatória:** volte `n < 1` para `n < 0`, confirme que `TestLoad/debounce_zero_rejected_from_flag` reprova, restaure. Cole a saída.
- `gobsidian serve --debounce-ms 0` sai com erro e mensagem legível? Cole a saída real do comando.
- `GOBSIDIAN_DEBOUNCE_MS=0 gobsidian serve` sai com erro? Cole a saída real.
- `gobsidian serve --debounce-ms 1` sobe normalmente? Cole a saída.
- `grep -rn "DebounceMSSet" --include=*.go .` — os dois subcomandos preenchem?
- Sobrou alguma constante `250` fora de `internal/config/defaults.go`? Cole a saída de `grep -rn "250" --include=*.go internal/ cmd/`.

#### Regras de execução

- **O plano é a fonte.** Se um teste falhar por motivo que a seção não explica, **pare e reporte**.
- **Nunca rode `git checkout`, `git restore`, `git stash`, `git clean` ou `git reset`.**
- **`go mod tidy` está proibido.**
- **Ao editar arquivo por script, leia *e* grave com `newline=""`.**
- **Verde obrigatório antes do commit:** `go test -race ./...`, `go vet ./...`, `gofmt -l .`, `GOOS=linux go vet ./...`, `GOOS=darwin go vet ./...`.
- Commits em Conventional Commits, em inglês.

#### Contrato de relatório

Grave em `.superpowers/sdd/task-38-report.md`: o que implementou; evidência de TDD (RED e GREEN); a prova de mutação com saída colada; a saída real dos quatro comandos acima; a saída dos dois `grep`; arquivos alterados; achados da auto-revisão; e preocupações.

Responda com no máximo 15 linhas, no formato acima.

**Files:**
- Modify: `internal/config/config.go`, `internal/config/config_test.go`
- Modify: `internal/watcher/debounce.go`
- Modify: `docs/WINDOWS.md`, `docs/TOOLS.md`

**Commit:** `fix(config): reject debounce-ms below 1 instead of silently defaulting`

---

