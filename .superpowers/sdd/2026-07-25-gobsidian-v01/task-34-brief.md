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
- **Provas de mutação obrigatórias.** Use `scripts/mutate.ps1` — saída `0` é o que você quer, saída `1` significa que a regra continua sem cobertura. Cole a saída real das duas.

```bash
# 1. o reconciliador inteiro. Hoje, ANTES desta tarefa, esta mutacao sai 1:
#    o teste passa sem reconciliador nenhum. Depois dela tem que sair 0.
pwsh -File scripts/mutate.ps1 -Path internal/watcher/apply.go `
  -Anchor 'Reconcile(ctx, v, idx, log)' -Replacement '_ = Reconcile' `
  -Test TestReconcile_CorrectsLostEvents -Package ./internal/watcher/

# 2. o cancelamento tratado como erro
pwsh -File scripts/mutate.ps1 -Path internal/watcher/overflow.go `
  -Anchor 'errors.Is(err, context.Canceled)' -Replacement 'false' `
  -Test TestReconcile_CancelIsNotAnError -Package ./internal/watcher/
```

Rode a mutação 1 **antes** de escrever os testes novos, e cole essa saída também: ela é a linha de base que mostra que o teste antigo não conseguia reprovar. Sem esse antes, o depois não prova que a cobertura mudou.

#### O código dos dois testes que sustentam esta tarefa

Transcreva. Os outros quatro são diretos; estes dois carregam a decisão inteira, e a entrega anterior errou exatamente aqui — deixando o watcher rodando.

```go
// TestReconcile_CorrectsLostEvents roda SEM watcher. Isso e a tarefa inteira:
// com o watcher ligado, o pipeline normal aplica as tres mudancas e a
// reconciliacao nunca e exercitada — foi assim que um requisito P0 passou por
// uma revisao inteira com cobertura zero.
func TestReconcile_CorrectsLostEvents(t *testing.T) {
	tmp := t.TempDir()

	modificado := filepath.Join(tmp, "file1.md")
	removido := filepath.Join(tmp, "file2.md")
	if err := os.WriteFile(modificado, []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(removido, []byte("some content"), 0644); err != nil {
		t.Fatal(err)
	}

	v, err := vault.New(tmp)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	idx := index.New()
	if err := idx.Build(context.Background(), v); err != nil {
		t.Fatalf("idx.Build: %v", err)
	}

	// O cofre muda por baixo, com NINGUEM escutando. E o que "eventos perdidos"
	// significa: nao ha evento nenhum para perder, so divergencia.
	if err := os.WriteFile(modificado, []byte("new content, different size"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(removido); err != nil {
		t.Fatal(err)
	}
	criado := filepath.Join(tmp, "file3.md")
	if err := os.WriteFile(criado, []byte("added content"), 0644); err != nil {
		t.Fatal(err)
	}

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	updated, removedN, skipped := Reconcile(context.Background(), v, idx, log)

	if n, ok := idx.Get("file1.md"); !ok || n.Size != int64(len("new content, different size")) {
		t.Errorf("modificado nao reconciliado: ok=%v nota=%+v", ok, n)
	}
	if _, ok := idx.Get("file2.md"); ok {
		t.Errorf("removido continua no indice")
	}
	if n, ok := idx.Get("file3.md"); !ok || n.Size != int64(len("added content")) {
		t.Errorf("criado nao entrou no indice: ok=%v nota=%+v", ok, n)
	}
	if updated < 2 || removedN != 1 {
		t.Errorf("contadores = updated %d, removed %d, skipped %d; quer >=2 e 1",
			updated, removedN, skipped)
	}
}

// TestApply_ReconcileSignal prova que a correcao veio do SINAL, e nao de um
// evento: o canal de entrada e criado e nunca alimentado.
func TestApply_ReconcileSignal(t *testing.T) {
	tmp := t.TempDir()
	alvo := filepath.Join(tmp, "nota.md")
	if err := os.WriteFile(alvo, []byte("antes"), 0644); err != nil {
		t.Fatal(err)
	}

	v, err := vault.New(tmp)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	idx := index.New()
	if err := idx.Build(context.Background(), v); err != nil {
		t.Fatalf("idx.Build: %v", err)
	}

	in := make(chan []vault.CanonicalPath)  // criado e NUNCA alimentado
	reconcile := make(chan struct{}, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	go Apply(ctx, in, reconcile, idx, v, log /* ...contadores conforme a assinatura atual */)

	if err := os.WriteFile(alvo, []byte("depois, bem maior"), 0644); err != nil {
		t.Fatal(err)
	}
	reconcile <- struct{}{}

	// Espera em laco com condicao de saida. time.Sleep fixo como assercao e o
	// que fez o teste anterior passar sem mecanismo nenhum.
	quer := int64(len("depois, bem maior"))
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if n, ok := idx.Get("nota.md"); ok && n.Size == quer {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	n, _ := idx.Get("nota.md")
	t.Fatalf("indice nao foi corrigido pelo sinal de reconciliacao: %+v", n)
}
```

Os dois ficam no pacote **interno** (`package watcher`), porque chamam `Reconcile` e `Apply` diretamente — é o que `overflow_test.go` já faz hoje. Importe `context`, `io`, `log/slog`, `os`, `path/filepath`, `testing`, `time`, mais `index` e `vault`. Ajuste a chamada de `Apply` à assinatura de contadores que a Task 32 deixou.
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

