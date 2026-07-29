### Task 34: Reconciliação — cobertura que pode falhar, e cancelamento que não é erro

Refaz o teste do requisito P0 RF-05, que hoje tem cobertura zero, e corrige dois defeitos em `overflow.go`.

#### Onde isto encaixa

A reconciliação é a resposta ao caso em que eventos foram perdidos e não se sabe quais. É o mecanismo que impede o índice de ficar silenciosamente incorreto, que é o pior estado possível do servidor. Ela existe, funciona, e **não tem um único teste capaz de reprovar se ela parar de funcionar**.

#### A evidência medida do defeito

```
mutei internal/watcher/apply.go:  case <-reconcile: _ = Reconcile
go test -race -run TestOverflowReconciliationFull ./internal/watcher/
ok  github.com/jonyd/gobsidian/internal/watcher  2.823s
```

`internal/watcher/overflow_test.go:51` deixa o watcher rodando e consumindo eventos normalmente, então o pipeline comum aplica as três mudanças dentro dos 200 ms de `time.Sleep` e a reconciliação nunca é exercida. O plano da Task 30 avisava disto textualmente e o aviso foi ignorado.

Dois defeitos adicionais no mesmo arquivo:

- `internal/watcher/overflow.go:47-50` loga `Error` para qualquer retorno de `v.Walk`, inclusive `context.Canceled`. Reconciliação interrompida por shutdown é encerramento normal e vira erro no stderr do operador.
- `Reconcile` não devolve nada. `updated`, `removed` e `skipped` só existem dentro de um `log.Warn`, e o contrato da Task 30 declara "contadores de corrigidos por reconciliação" como entrega.

#### O que já está fechado e vincula esta tarefa

- **`ctx.Canceled` é encerramento normal**, aqui como no serve loop.
- **Falha na raiz não pode virar sucesso vazio.** Se a varredura falhar porque o cofre sumiu, a reconciliação **não** pode concluir "nenhum arquivo existe, remova tudo". O comportamento correto já existe (`overflow.go:47` retorna antes do laço de remoção) e precisa de teste.
- **Escritas no índice são serializadas na goroutine do watcher.** A reconciliação roda na mesma goroutine que aplica mudanças; não a mova para outra.
- **Overflow durante reconciliação agenda uma repetição, não uma cascata.** O canal `reconcile` tem capacidade 1 e o envio é não bloqueante — o mecanismo existe e precisa de teste.
- **`ErrEventOverflow` só é emitido por `backend_inotify.go:398` e `backend_windows.go:582`.** kqueue (macOS, BSD) não emite. Lacuna registrada, não resolvida.

#### O que implementar

**Apague `internal/watcher/overflow_test.go` inteiro**, incluindo o tipo morto `mockFsnotify` (linha 17), declarado e nunca usado. Escreva no lugar:

1. `TestReconcile_CorrectsLostEvents` — **sem watcher nenhum**. `vault.New` + `idx.Build` sobre `file1.md` ("old") e `file2.md`. Depois: reescrever `file1.md` com tamanho diferente, `os.Remove(file2.md)`, criar `file3.md`. Chamar `Reconcile(ctx, v, idx, log)` direto. Afirmar: `file1` com `Size` novo, `file2` ausente de `idx.Get`, `file3` presente com `Size` certo.
2. `TestApply_ReconcileSignal` — canal `in` criado e **nunca alimentado**. `Apply` numa goroutine. Mudar o cofre. Mandar um `struct{}{}` em `reconcile`. Esperar e afirmar o índice corrigido. Prova que a correção veio do sinal, não de evento.
3. `TestRun_OverflowSchedulesExactlyOne` — construir o `Watcher` sem rodar `Apply`; injetar 5 `fsnotify.ErrEventOverflow`; afirmar que o canal `reconcile` entrega **um** token e que o contador de reconciliações vale 5.
4. `TestReconcile_VaultGoneLeavesIndexIntact` — índice sobre 3 notas, raiz do cofre renomeada ou apagada, `Reconcile` chamada; afirmar `idx.NoteCount() == 3` e nenhum `Remove`.
5. `TestReconcile_CtxCancelStopsEarly` — cofre com 200 notas, `ctx` cancelado depois da primeira entrada visitada (contador dentro do callback); afirmar que o número de visitadas é muito menor que 200 e que o índice não foi esvaziado.
6. `TestReconcile_CancelIsNotAnError` — capturar a saída do `slog` num `bytes.Buffer`, cancelar o `ctx` no meio, afirmar que **nenhum** registro de nível `ERROR` foi emitido.

E em `internal/watcher/overflow.go`:

```go
if err != nil {
	if errors.Is(err, context.Canceled) {
		log.Debug("Reconciliação interrompida pelo shutdown", "updated", updated, "removed", removed, "skipped", skipped)
		return updated, removed, skipped
	}
	log.Error("Erro durante varredura de reconciliação", "err", err)
	return updated, removed, skipped
}
```

`Reconcile` passa a devolver `(updated, removed, skipped int)`. Atualize a chamada em `apply.go`; a Task 37 é quem liga esses números aos contadores publicados — aqui basta devolvê-los.

Acrescente no topo de `Reconcile` o comentário que amarra a lacuna de plataforma ao ponto de uso:

```go
// Em macOS e BSD esta funcao nunca e chamada: o backend kqueue do fsnotify
// v1.10.1 nao emite ErrEventOverflow (so backend_inotify.go:398 e
// backend_windows.go:582 emitem). La o unico anteparo contra evento perdido
// e a reindexacao no boot. Lacuna registrada, nao resolvida por heuristica.
```

#### Armadilhas já pagas neste projeto que se aplicam aqui

- **Verificar conteúdo, não presença.** Verificar que a reconciliação *rodou* não é verificar que ela *corrigiu*. Afirme o conteúdo do índice depois.
- **Um teste com `time.Sleep` generoso passa mesmo sem o mecanismo.** É exatamente o que aconteceu aqui. Se o teste precisa esperar, espere em laço com condição de saída e afirme a condição, não o tempo.
- **Determinismo sob paralelismo.** `go test -race` sobre teste que escreve no cofre enquanto o watcher roda é o teste que vale.

#### Verificações além dos passos

Faça e **reporte o resultado real de cada uma**:

- Os seis testes acima existem e passam? Liste os seis com o resultado.
- **Prova de mutação obrigatória:** trocar o corpo de `Reconcile` por `return 0, 0, 0`, rodar, confirmar que `TestReconcile_CorrectsLostEvents` **e** `TestApply_ReconcileSignal` reprovam nomeando os três arquivos, restaurar. Cole a saída real.
- **Segunda prova de mutação:** trocar `errors.Is(err, context.Canceled)` por `false`, confirmar que `TestReconcile_CancelIsNotAnError` reprova, restaurar.
- Quanto tempo a reconciliação leva no cofre de teste disponível? Um número medido, com a contagem de notas ao lado. Se não mediu, escreva **"não medido"**.
- O contador de reconciliações incrementa uma vez por overflow, e não uma vez por arquivo reconciliado?

#### Regras de execução

- **O plano é a fonte.** Se um teste falhar por motivo que a seção não explica, **pare e reporte**.
- **Nunca rode `git checkout`, `git restore`, `git stash`, `git clean` ou `git reset`.**
- **`go mod tidy` está proibido.**
- **Ao editar arquivo por script, leia *e* grave com `newline=""`.**
- **Verde obrigatório antes do commit:** `go test -race ./...`, `go vet ./...`, `gofmt -l .`, `GOOS=linux go vet ./...`, `GOOS=darwin go vet ./...`.
- Commits em Conventional Commits, em inglês.

#### Contrato de relatório

Grave em `.superpowers/sdd/task-34-report.md`: o que implementou; evidência de TDD (RED e GREEN, comando e saída); as **duas** provas de mutação com saída real colada; a tabela de verificações; o tempo medido da reconciliação ou "não medido"; arquivos alterados; achados da auto-revisão; e preocupações.

Responda com no máximo 15 linhas, no formato acima.

**Files:**
- Rewrite: `internal/watcher/overflow_test.go`
- Modify: `internal/watcher/overflow.go` (retorno de contadores, `ctx.Canceled`, comentário da lacuna)
- Modify: `internal/watcher/apply.go` (chamada com retorno)

**Interfaces:**
- Produces: `Reconcile(ctx, v, idx, log) (updated, removed, skipped int)`

**Commit:** `test(watcher): reconciliation tests that actually lose events`

---

