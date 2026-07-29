### Task 41: O teste ponta a ponta que prova que o M2 existe

#### Onde isto encaixa

A Task 29 pede, textualmente: *"escreva uma nota nova no cofre com o servidor rodando, espere uma janela, e chame `vault_stats`. A contagem subiu? **Este é o teste que prova que o M2 existe**; sem ele, tudo acima é maquinário sem consumidor."*

Ele não foi escrito. `internal/watcher/burst_test.go` afere `idx.NoteCount()`, que é uma camada abaixo do pedido — não prova que o índice que o watcher atualiza é o mesmo que a tool serve. O relatório da Task 29 apresentou o teste de rajada como cumprimento desta verificação.

#### O que já está fechado e vincula esta tarefa

- **Nenhum tipo do SDK de MCP cruza para fora de `internal/mcpsrv`.** O teste roda contra o serviço ou contra o handler, não subindo processo.
- **Modo somente-leitura não desliga o watcher.** Se o teste configurar `--read-only`, a contagem ainda tem que subir.
- **Determinismo sob paralelismo.** `go test -race`, e espera em laço com condição, nunca `time.Sleep` fixo como afirmação.

#### O que implementar

`TestVaultStatsReflectsWatcherUpdate`, em `internal/mcpsrv` ou `internal/service` — onde o handler de `vault_stats` for exercitável sem subir processo. Diga qual escolheu e por quê.

1. Montar cofre temporário, `index.Build`, subir o watcher com janela curta apontando para **o mesmo `*index.Index`** que o serviço recebe.
2. Chamar `vault_stats` e guardar a contagem de notas.
3. Escrever uma nota nova no disco.
4. Esperar em laço até a contagem mudar, com timeout.
5. Afirmar `notas + 1`.
6. Apagar a nota, esperar, afirmar que voltou ao valor original.

#### Armadilhas já pagas neste projeto que se aplicam aqui

- **Um teste que não pode falhar é pior que teste ausente.** Se o teste medir só depois do `Build`, ele passa sem watcher nenhum. **Prova de mutação obrigatória:** desligue a chamada a `idx.Replace` em `internal/watcher/apply.go`, confirme que este teste reprova, restaure. Sem essa mutação você não sabe se está medindo o `Build` do boot.
- **Handler que devolve `error` Go faz o SDK montar `IsError` sem `StructuredContent`.** Se o teste for pelo handler, afirme o conteúdo estruturado, não só a ausência de erro.
- **Verificar conteúdo, não presença.** Afirmar que o campo existe não é afirmar que ele mudou.

#### Verificações além dos passos

- A contagem sobe depois de criar a nota? Em quanto tempo? Um número medido.
- A contagem volta depois de apagar?
- O teste passa com `--read-only` ligado? (O watcher não pode ser desligado por ele.)
- `go test -race` acusa corrida entre a escrita do watcher e a leitura da tool?

#### Regras de execução

- **O plano é a fonte.** Se um teste falhar por motivo que a seção não explica, **pare e reporte** — pode ser defeito real de wiring, e nesse caso a correção é do código.
- **Nunca rode `git checkout`, `git restore`, `git stash`, `git clean` ou `git reset`.**
- **`go mod tidy` está proibido.**
- **Verde obrigatório antes do commit:** `go test -race ./...`, `go vet ./...`, `gofmt -l .`, `GOOS=linux go vet ./...`, `GOOS=darwin go vet ./...`.
- Commits em Conventional Commits, em inglês.

#### Contrato de relatório

Grave em `.superpowers/sdd/task-41-report.md`: o que implementou; onde pôs o teste e por quê; evidência de TDD (RED e GREEN); a prova de mutação com saída colada; o tempo medido até a contagem subir; a tabela de verificações; arquivos alterados; e preocupações.

Responda com no máximo 15 linhas, no formato acima.

**Files:**
- Create ou Modify: teste em `internal/mcpsrv/` ou `internal/service/`

**Commit:** `test(mcpsrv): vault_stats reflects a note created while the server runs`

---

