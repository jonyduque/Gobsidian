### Task 37: Contadores por motivo, `active` que pode ser falso, e a camada de volta ao lugar

Implementa a decisão fechada 3 e corrige dois achados da revisão da Task 32.

#### Onde isto encaixa

A Task 32 publicou os contadores em `vault_stats` e fechou o M2. A revisão encontrou que o campo mais diagnóstico é um agregado de quatro causas distintas, que `active` é constante, e que o tipo dos contadores inverteu a direção das camadas.

#### A evidência medida dos defeitos

**Agregação.** `docs/TOOLS.md:239` fixa o objeto `watcher` em seis campos e descreve `events_dropped` como "os irrelevantes ou ocultos" — um número só. O plano da Task 27 exige contagem **por motivo**, e o parágrafo de `TOOLS.md` vende esse número como *"a instrumentação principal para diagnosticar cofres em pastas sincronizadas"*. `events_dropped` alto por `chmod` significa "OneDrive normal, ignore"; por `outside_vault` significa "a raiz é um link e o confinamento está recusando eventos"; por `excluded` significa "atividade em `.obsidian/`". Três diagnósticos num número só. Falta também o contador de coalescidos, que a Task 28 declara como entrega.

**`active` é campo de valor fixo.** `internal/watcher/counters.go:8` devolve `Active: true` literal, e `internal/service/graph.go:395` só inclui o objeto quando `s.watcher != nil`. O campo é `true` em 100% das respostas que o contêm. É o defeito do `alias_collisions: 0` outra vez: aparece na resposta e não pode estar errado porque não pode variar. Watcher que morreu no `Run` continua reportando `active: true`.

**Inversão de camada.** `internal/watcher/counters.go:3` importa `internal/service`. `docs/ARCHITECTURE.md:40` é explícito: *"`internal/service` é o único ponto que enxerga todos os subsistemas. As camadas abaixo dele não se conhecem."* A interface `WatchStats` foi declarada em `service` justamente para a dependência apontar num sentido só, e o tipo `WatchCounters` morando lá obriga o watcher a importar de volta quem deveria consumi-lo.

#### O que implementar

**1. O tipo volta para o watcher.** Em `internal/watcher/counters.go`:

```go
// Counters e o retrato dos contadores do watcher. Mora aqui, e nao em
// internal/service: as camadas abaixo do servico nao o conhecem
// (ARCHITECTURE.md §1). Quem casa os dois lados e o adaptador em
// cmd/gobsidian/serve.go.
type Counters struct {
	Active            bool
	EventsReceived    int64
	EventsDropped     int64            // soma de DroppedByReason
	DroppedByReason   map[string]int64 // chmod, outside_vault, excluded, unknown_op
	EventsCoalesced   int64
	EventsProcessed   int64
	EventsSkipped     int64
	Reconciliations   int64
	ReconciledUpdated int64
	ReconciledRemoved int64
}
```

`internal/watcher` deixa de importar `internal/service`. O adaptador que satisfaz `service.WatchStats` vai para `cmd/gobsidian/serve.go`, que é onde os subsistemas já se encontram. Confirme com `go list -deps ./internal/watcher | grep service` — a saída tem que ser vazia.

**2. `Active` real.** Um `atomic.Bool` no `Watcher`, ligado na primeira linha de `Run` e desligado num `defer`. `active: false` passa a significar "o subsistema existe e caiu", que é o evento que mais importa diagnosticar.

**3. Motivos de descarte.** `filter` em `internal/watcher/filter.go` passa a devolver o motivo junto com o booleano. Os quatro motivos são fechados e não podem ser inventados outros sem mudar o plano: `chmod`, `outside_vault`, `excluded`, `unknown_op`. **O incremento acontece onde a decisão é tomada**, dentro do `filter` ou imediatamente no chamador a partir do motivo devolvido — nunca por reconstrução da causa depois.

**4. Coalescidos.** Em `internal/watcher/debounce.go`, incremente a cada evento que chega num caminho **já presente** no conjunto sujo. Esse é o número que significa "trabalho evitado". A diferença entre tamanho do conjunto e tamanho do lote **não** é esse número.

**5. Corrigidos por reconciliação.** Ligue `ReconciledUpdated` e `ReconciledRemoved` ao retorno de `Reconcile` que a Task 34 introduziu.

**6. Publicação.** `internal/service/service.go` — `WatchCounters` ganha os campos com as tags JSON: `events_dropped_by_reason`, `events_coalesced`, `reconciled_updated`, `reconciled_removed`. **`events_dropped` continua sendo a soma**, para não quebrar quem já lê o campo.

**7. `docs/TOOLS.md` faz parte da entrega.** A lista de campos da §`vault_stats` passa a incluir os quatro novos, e o parágrafo de "Notas" ganha o que cada motivo significa operacionalmente:

> `events_dropped_by_reason` desdobra o total porque as causas pedem ações diferentes: `chmod` alto é OneDrive em operação normal e pode ser ignorado; `outside_vault` alto indica que a raiz do cofre é um link e o confinamento está recusando eventos; `excluded` alto indica atividade em `.obsidian/` ou `.git/`; `unknown_op` alto indica evento que o filtro não soube classificar e merece `--log-level debug`.

#### Os quatro pontos onde esta tarefa vai dar errado, com o código que evita cada um

Esta é a tarefa mais larga do bloco — oito arquivos, três pacotes, uma corrida de dados e um golden. Os quatro pontos abaixo não são estilo; são os lugares onde um erro compila, passa nos testes e mente na resposta.

**1. `map` de contadores não é seguro para leitura concorrente, e `atomic` não conserta isso.** Os contadores são escritos na goroutine do watcher e lidos na que atende MCP. Um `map[string]int64` compartilhado, mesmo que só cresça, é corrida de dados — `go test -race` acusa, e um `atomic.Int64` **dentro** do map não ajuda, porque o mapa em si é a estrutura compartilhada. Guarde quatro atomics separados e monte o mapa **dentro** do `Stats()`, na goroutine que pergunta:

```go
type Watcher struct {
	// ...
	droppedChmod        atomic.Int64
	droppedOutsideVault atomic.Int64
	droppedExcluded     atomic.Int64
	droppedUnknownOp    atomic.Int64
}

func (w *Watcher) Stats() Counters {
	// O mapa nasce aqui, por chamada. Um mapa compartilhado entre a goroutine
	// que incrementa e a que le e corrida de dados mesmo que so cresca, e
	// atomic no valor nao protege a estrutura.
	porMotivo := map[string]int64{
		"chmod":         w.droppedChmod.Load(),
		"outside_vault": w.droppedOutsideVault.Load(),
		"excluded":      w.droppedExcluded.Load(),
		"unknown_op":    w.droppedUnknownOp.Load(),
	}
	var soma int64
	for _, n := range porMotivo {
		soma += n
	}
	return Counters{
		Active:          w.active.Load(),
		EventsDropped:   soma, // sempre derivada, nunca contada em paralelo
		DroppedByReason: porMotivo,
		// ...
	}
}
```

`EventsDropped` é **derivada da soma**, nunca um quinto contador incrementado em paralelo. Dois contadores para a mesma coisa divergem, e o dia em que divergirem ninguém vai saber qual acreditar.

**2. O motivo tem que ser devolvido pelo `filter`, não reconstruído depois.** Reconstruir a causa no chamador é como o contador agregado nasce de novo — e erra, porque a informação que distingue os motivos já foi descartada.

```go
// DropReason e fechado: quatro valores, e nenhum outro entra sem mudar o plano.
type DropReason string

const (
	DropChmod        DropReason = "chmod"
	DropOutsideVault DropReason = "outside_vault"
	DropExcluded     DropReason = "excluded"
	DropUnknownOp    DropReason = "unknown_op"
)

// filter devolve o motivo junto com a decisao. Quando ok e true, reason nao
// tem significado — nao a use.
func filter(e fsnotify.Event, root string, log *slog.Logger) (Event, bool, DropReason)
```

**3. `active` precisa ser desligado num `defer`, no topo do `Run`.** Ligar no fim ou desligar só no caminho feliz devolve o campo ao estado de hoje: `true` sempre, inclusive depois de o watcher morrer.

```go
func (w *Watcher) Run(ctx context.Context) error {
	w.active.Store(true)
	defer w.active.Store(false) // vale para ctx cancelado, canal fechado e panic
	// ...
}
```

**4. `internal/watcher` não pode importar `internal/service`.** O tipo `Counters` mora no `watcher`; quem satisfaz `service.WatchStats` é um adaptador em `cmd/gobsidian/serve.go`, que é onde os subsistemas já se encontram. Confirme com `go list -deps ./internal/watcher | Select-String service` — a saída tem que ser vazia, e ela vai no relatório.

```go
// em cmd/gobsidian/serve.go
type watcherStats struct{ w *watcher.Watcher }

func (a watcherStats) Stats() service.WatchCounters {
	c := a.w.Stats()
	return service.WatchCounters{
		Active:          c.Active,
		EventsReceived:  c.EventsReceived,
		EventsDropped:   c.EventsDropped,
		DroppedByReason: c.DroppedByReason,
		EventsCoalesced: c.EventsCoalesced,
		// ...
	}
}
```

#### Armadilhas já pagas neste projeto que se aplicam aqui

- **Campo de API com valor fixo mente sempre.** É o defeito que esta tarefa corrige; não introduza outro.
- **Contador que nasce agregado nunca vira desagregado depois** — vira uma segunda passagem inteira.
- **`-update` de golden grava o que o código produz, não o que está certo.** O golden de `vault_stats` vai mudar. Depois de gerar, **leia cada `.json` e confira contra o que você esperava antes de rodar**.
- **A armadilha de wiring:** mudar assinatura obriga a atualizar todos os chamadores. `grep -rn "service.New(" --include=*.go .` antes de commitar. Uma mudança de wiring já foi propagada para `serve` e esquecida em `doctor` neste projeto.
- **Determinismo sob paralelismo.** Contadores são escritos na goroutine do watcher e lidos na que atende MCP. `sync/atomic` ou mutex, e prove com `go test -race`. O `map` de `DroppedByReason` **não é** seguro para leitura concorrente: monte uma cópia dentro do `Stats()`, a partir de quatro atomics separados.

#### Verificações além dos passos

- `TestCounters_DropReasons` — um evento de cada motivo pelo `filter`; afirmar que **exatamente** o contador correspondente subiu e os outros três não.
- **Prova de mutação por contador novo:** pare de incrementar cada um dos quatro motivos, mais coalescidos, mais os dois de reconciliação, e confirme que um teste reprova nomeando o contador. Sete mutações, sete saídas coladas.
- `active` fica `false` depois de `Run` retornar? Prove com um teste que cancela o `ctx` e lê `Stats()` depois.
- `go list -deps ./internal/watcher | grep service` devolve vazio? Cole a saída.
- `grep -rn "service.New(" --include=*.go .` — liste todos os chamadores e confirme que todos foram atualizados.
- Os nomes JSON em `docs/TOOLS.md` batem campo a campo com o que o código emite? Compare o doc com a saída real da tool, não de memória.
- `go test -race ./...` acusa corrida na leitura dos contadores durante escrita?

#### Regras de execução

- **O plano é a fonte.** Se um teste falhar por motivo que a seção não explica, **pare e reporte**.
- **Nunca rode `git checkout`, `git restore`, `git stash`, `git clean` ou `git reset`.**
- **`go mod tidy` está proibido.**
- **Ao editar arquivo por script, leia *e* grave com `newline=""`.**
- **Verde obrigatório antes do commit:** `go test -race ./...`, `go vet ./...`, `gofmt -l .`, `GOOS=linux go vet ./...`, `GOOS=darwin go vet ./...`.
- Commits em Conventional Commits, em inglês.

#### Contrato de relatório

Grave em `.superpowers/sdd/task-37-report.md`: o que implementou; evidência de TDD (RED e GREEN); **as sete provas de mutação com saída colada**; a tabela de verificações; a saída de `go list -deps`; a lista de chamadores de `service.New`; o diff de `docs/TOOLS.md`; o diff do golden com a confirmação de que você **leu** cada valor novo; arquivos alterados; achados da auto-revisão; e preocupações.

Responda com no máximo 15 linhas, no formato acima.

**Files:**
- Modify: `internal/watcher/counters.go`, `counters_test.go`, `filter.go`, `filter_test.go`, `debounce.go`, `watcher.go`, `apply.go`
- Modify: `internal/service/service.go`, `internal/service/graph.go`
- Modify: `cmd/gobsidian/serve.go` (adaptador `service.WatchStats`)
- Modify: `docs/TOOLS.md`, golden de `vault_stats`

**Commit:** `feat(watcher): per-reason drop counters, coalesced count, and a real active flag`

---

