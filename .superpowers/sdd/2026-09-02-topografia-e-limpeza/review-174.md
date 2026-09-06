# Revisão — Task 174 (`internal/boot`), commit `eac13c9` sobre `7119645`

Revisão **somente leitura**: o gate de órfãos estava rodando nesta máquina
(`orphans-174.log`, lançado 09:35), então nada que compile ou levante processo
foi executado. Ferramentas usadas: `git show`, `diff`, `grep`, `gofmt -l`,
`go list`. Onde a evidência depende de compilar ou testar, ela é do relatório e
está marcada como tal.

---

## Spec compliance

**Veredito: ✅ PASS.**

Requisito por requisito, com a evidência que eu mesmo produzi:

| Requisito do brief | Evidência |
|---|---|
| Pacote `boot` com `doc.go`, `indice.go`, `busca.go`, `montar.go` | `git show --name-status eac13c9`: os quatro em `A`, mais `A internal/boot/{busca,indice,montar}_test.go` |
| `git mv` dos dois testes, `git rm servico.go` | `R095 cmd/gobsidian/boot_indice_busca_windows_test.go → internal/boot/busca_windows_test.go`, `R095 …/inverted_cache_state_test.go → …/estado_do_cache_test.go`, `R077 cmd/gobsidian/servico.go → internal/boot/montar.go` — renames reconhecidos pelo git, não delete+add |
| Assinaturas exatas que as Tasks 175–177 consomem | `AbrirIndice` (`indice.go:14`), `PrepararBusca` (`busca.go:113`), `Componentes` + `Esperar` (`montar.go:23,34`), `Montar` (`montar.go:53`) — idênticas ao bloco do brief |
| `boot` importa **exatamente** `config index search service vault watcher` | `go list -f '{{.Imports}}' ./internal/boot/` rodado por mim: as seis, e só. Nenhum `mcpsrv`, `lifecycle`, `ipc`, `daemon`, `doctor`, `net` |
| Nenhum pacote de domínio importa `boot` | `grep -rn "internal/boot" --include=*.go .` fora do próprio pacote devolve **duas** linhas: `cmd/gobsidian/daemon.go:17` e `cmd/gobsidian/serve.go:10` |
| Log só pelo `*slog.Logger` recebido | `grep -rn "fmt\.Print" internal/boot/` → vazio |
| Linha `servidor pronto` com as mesmas chaves | `montar.go:235-241`: `vault, read_only, notes, assets, index_ms, index_origin`, mesma mensagem, mesma ordem. `scripts/measure.ps1:191,196,199` casa `servidor pronto`, `index_ms=(\d+)` e `index_origin=(\w+)` — os três continuam casando, e os literais seguem `"cache"`/`"build"` (`indice.go:15,20`) |
| `Esperar()` onde estava `wg.Wait()`, depois de `lifecycle.Shutdown` | `serve.go`: `lifecycle.Shutdown(...)` → `lc.Wait()` → `c.Esperar()`; `daemon.go`: `lifecycle.Shutdown(...)` → `lc.Wait()` → `c.Esperar()`. Nos dois o `Esperar()` substitui o `montado.wg.Wait()` **na mesma posição** — verificado no diff `-U10`, hunks `@@ -437,26 +156,26 @@` e `@@ -148,38 +149,38 @@` |
| Código de plataforma atrás de build tag | `busca_windows_test.go` mantém `//go:build windows` na primeira linha; nenhum `runtime.GOOS` no pacote |
| `t.TempDir()` para cofre e `CacheDir` | `indice_test.go:19,32` — `raiz := t.TempDir()` e `CacheDir: t.TempDir()`; nenhum teste toca o cache padrão do usuário |
| Âncora da mutação intacta | `grep -n "ctx.Err() != nil" internal/boot/busca.go` → **uma** ocorrência, `busca.go:201`, dentro do `for _, p := range caminhos` de `construirBusca`, imediatamente antes do comentário `// Sai sem gravar, de proposito.` Como `mutate.ps1` recusa âncora ambígua, a unicidade é a garantia de alvo |
| Prova de mutação no passado, com saída, exit 0 | Relatório §Step 4: `--- FAIL: TestPrepararBuscaCtxCanceladoNaoMarcaPronta … busca_test.go:79` e `EXIT=0`. Tempo verbal correto, saída colada. Não re-executada aqui (rodaria `go test`) |
| Commit em Conventional Commits, inglês, staging por caminho | `refactor(boot): assemble vault, index, search and service in one testable package`; 15 arquivos, todos da lista do brief + `docs/OPERACAO.md`. Nada da árvore não-commitada do dono entrou |
| Docs: CLAUDE.md árvore + grafo, ESTRUTURA, ARCHITECTURE | Todos editados. Ver **N1–N5** para o que ficou errado |

**Movimento, não reescrita — verificado mecanicamente, não a olho.** Extraí
`7119645:cmd/gobsidian/serve.go` e `7119645:cmd/gobsidian/servico.go`, apliquei
só os renomes da tabela do relatório e diferenciei contra os arquivos novos:

- `carregarIndiceDoCache` (34 linhas, comentário incluído): **diff vazio**.
- `busca.go` (238 linhas): duas diferenças, ambas em comentário, nenhuma em
  código executável — `runServe` → `Montar` em `busca.go:111` (a ordem que o
  comentário descreve passou mesmo a ser imposta em `Montar`) e a reescrita do
  Concern 3 em `busca.go:195-197`. Tudo o mais, byte a byte.
- `Montar` vs `construirServico` (197 linhas): duas diferenças, as duas
  intencionais e previstas pelo brief — o `if/else` de cache virou a chamada a
  `AbrirIndice` (`montar.go:87-90`) e o `var wg sync.WaitGroup` virou
  `c.espera` (`montar.go:198-202`). O resto — a goroutine da varredura, o
  `JOIN`, o `inv.MarkBuilding()`, o `watcher.New(…, time.Duration(cfg.DebounceMS)*time.Millisecond, …)`,
  o `opts.CarregarBusca`, a ordem `PrepararBusca` → `w.Run` na goroutine de
  fundo, e o bloco `servidor pronto` — verbatim.

**Semântica de goroutine e ctx de `PrepararBusca`/`construirBusca`:** inalterada.
O `defer devolveMemoriaTransitoria(log)` continua no topo de `PrepararBusca`
(cobrindo o `return` do caminho de cache completo, que é onde as medições foram
feitas); o `return` sem gravar sob `ctx.Err() != nil` continua no mesmo ponto do
laço; a gravação periódica por `intervaloDeGravacao` continua depois do
`inv.Update`. Diff vazio nesses trechos.

**`sync.WaitGroup` nunca copiado:** `Componentes` é sempre manipulado por
ponteiro — `Montar` devolve `*Componentes` (`montar.go:53,243`), `Esperar` tem
receptor ponteiro (`montar.go:34`), e os dois chamadores guardam `c` como
ponteiro. `go vet`/`copylocks` (etapas 5–7 do gate) transformaria uma cópia
futura em erro de gate.

**Dispensa em `docs/OPERACAO.md:1525` — legítima, e conferi o fato, não só a
forma.** A frase registra que o segundo binário foi compilado com
`SnippetCacheEntries` em zero. `service.Options.SnippetCacheEntries` é
`internal/service/service.go:50`, e o literal `service.Options{…}` que um
experimento desses editaria é o de `montar.go:166-169` — que era mesmo
`cmd/gobsidian/servico.go`. Então "hoje `internal/boot/montar.go`" é verdadeiro,
o comentário de dispensa está **na mesma linha** da referência e traz o motivo
obrigatório, e a história não foi reescrita. Único reparo de forma: o comentário
HTML caiu no meio da frase ("a edição `<!-- … -->` foi aplicada"), o que não
afeta a renderização mas atrapalha quem lê o Markdown cru.

**Escopo:** não encolheu em silêncio. `test_orphans.ps1` (Step 5) está declarado
como escopo do orquestrador, com a frase certa — "não há quatro `[OK]` para colar
aqui" — em vez de um número inventado. E o wall clock de `servidor pronto`
(Step 7) está marcado **"Não medi"**, com o motivo (o `measure.ps1` não emite
esse número). As duas são exatamente a resposta que o `CLAUDE.md` pede.

**Medição do Step 7:** a execução descartável (394 ms, cache de página frio) foi
**mantida na tabela** e o descarte foi justificado antes de ser feito, com o n=3
declarado e a conclusão limitada ao que n=3 sustenta ("nada indica que tenha
ficado mais rápido"). É o oposto do padrão que `revisor.md` §2 nomeia.

---

## Code quality

**Veredito: ✅ PASS.** Nenhum achado bloqueante. Os cinco achados que valem
conserto são todos de documentação, e três deles são afirmações **falsas** sobre
o estado atual da árvore — a classe que o `CLAUDE.md` proíbe por nome ("Não
afirme estado que você não verificou"). O código movido está verbatim, `gofmt -l`
sobre `internal/boot` e `cmd/gobsidian` volta vazio, e o grafo de dependências
que os documentos declaram bate com o `go list` que eu rodei.

---

## Achados

### N1 — `docs`: "CLI chamam" é falso — should-fix

`CLAUDE.md` (árvore, linha do `boot/`): «monta cofre, índice, busca, watcher e
Service; **serve, daemon e CLI chamam**».
`docs/ESTRUTURA.md:153`: «a sequência de boot que **serve, daemon e CLI**
compartilham».
`internal/boot/doc.go:3`: «Existe porque serve, daemon e **os subcomandos de
CLI** precisam da mesma montagem».

**Mecanismo, fechado por leitura — CONFIRMADO.** `grep -rn "internal/boot"
--include=*.go .` devolve dois importadores: `cmd/gobsidian/serve.go:10` e
`cmd/gobsidian/daemon.go:17`. Os subcomandos de CLI continuam montando o índice
por conta própria: `cmd/gobsidian/index.go:42`, `cmd/gobsidian/inspect.go:47` e
`cmd/gobsidian/search.go:38`, os três com `idx := index.New()`. Nenhum deles
chama `boot.Montar` nem `boot.AbrirIndice`.

**A falha que causa:** a árvore do `CLAUDE.md` é lida como estado corrente — é o
que a próxima sessão tem no lugar do contexto. Quem for fazer a Task 175/176
lendo essa linha vai procurar a chamada da CLI para `boot` e não vai achar, ou
pior, vai supor que a unificação já foi feita.

**Correção:** em `CLAUDE.md` e `ESTRUTURA.md:153`, «serve e daemon chamam» /
«que serve e daemon compartilham». Em `doc.go`, ou o mesmo corte, ou marcar
explicitamente como intenção («os subcomandos de CLI ainda montam o índice por
conta própria»). A linha veio do brief, mas o brief não pode autorizar uma
afirmação de estado falsa.

### N2 — `docs/ARCHITECTURE.md:568` ainda nomeia `construirServico` — should-fix

> O daemon monta `index`, `watcher` e o serviço de domínio uma única vez
> (`construirServico`, compartilhada com o boot em processo para as duas
> sequências nunca divergirem) e aceita N conexões sobre o socket.

**CONFIRMADO.** `construirServico` deixou de existir neste commit.
`ARCHITECTURE.md` **está** na lista de **Files** do brief e foi editado (§2.1 e
o novo §2.14) — esta ocorrência, na §7.5, passou batida.

**Por que o gate não pegou:** `check_doc_refs` resolve **caminhos de arquivo**;
`construirServico` é identificador Go, e nenhuma etapa do `verify.ps1` confere
identificador citado em `.md`. O gate verde não é evidência aqui.

**Correção:** `boot.Montar` (`internal/boot`).

### N3 — `docs/ARCHITECTURE.md` §2.14 repete um defeito já registrado — should-fix

O parágrafo novo afirma, sem qualificar:

> A ordem não é livre: `watcher.New` registra os watches **antes** da construção
> do índice de busca, e `w.Run` só consome a fila **depois** dela […]

**CONFIRMADO.** Isso vale **só no modo eager**. No modo padrão — preguiçoso —
`Montar` dispara `w.Run(ctx)` imediatamente (`montar.go:200-221`: o
`PrepararBusca` está dentro de `if cfg.EagerSearch`), e a carga da busca só
acontece na primeira `vault_search`, via `opts.CarregarBusca`
(`montar.go:187-193`). O próprio comentário logo acima, em `montar.go:179-186`,
diz o contrário do §2.14: «Como o watcher.Run comeca no boot […] e nao espera
este carregamento».

**O agravante:** essa generalização é **o achado A6 da auditoria de 2026-08-25**,
registrado em `docs/SUGESTOES.md:229` («comentário de `servico.go:104-107`
descreve apenas o modo eager») e listado como item 3 da lista de correções em
`docs/SUGESTOES.md:777`. O defeito foi corrigido no comentário do código e
reintroduzido no documento normativo.

**Correção:** qualificar — «no modo eager … ; no modo padrão `w.Run` começa
antes, e a adoção do cache acontece na primeira busca, onde `AdotarDe` recusa o
cache se o watcher já escreveu».

### N4 — `docs/ARCHITECTURE.md` §2.14: "o fluxo de §5.1 inteiro" — nit

> `boot.Montar` executa o fluxo de §5.1 inteiro

**CONFIRMADO.** §5.1 (`ARCHITECTURE.md:321-338`) tem oito passos. O passo 1
(analisar flags, resolver o caminho do cofre) é de `config`/`cmd`; o passo 2
(iniciar lifecycle) e o passo 8 (iniciar o servidor MCP em stdio) são
justamente os dois que `boot` **não** faz — o que o próprio §2.14 declara duas
frases depois («`boot` não importa `mcpsrv` nem `lifecycle`»). A seção
contradiz a si mesma.

**Correção:** «executa os passos 3 a 7 de §5.1».

### N5 — `docs/ESTRUTURA.md:235`: o segundo sítio do Step 6 não foi atualizado — should-fix

> `cmd/gobsidian` não contém lógica de domínio. Analisa flags, monta
> configuração, **constrói o `Service`** e delega.

**CONFIRMADO.** Depois deste commit quem constrói o `*service.Service` é
`boot.Montar` (`montar.go:196`); `cmd/gobsidian` recebe `c.Service` pronto
(`serve.go:110`, `daemon.go:170`). O brief pedia este sítio explicitamente —
Step 6, «`docs/ESTRUTURA.md:12-18`: apagar `servico.go` […]; **`:216` idem**».
Com a deriva de ~10 linhas que o orquestrador já registrou, `:216` é a linha
235 de hoje; só o primeiro sítio (a árvore, linha 17) foi atendido.

**Correção:** «Analisa flags, monta configuração, pede a montagem a
`internal/boot` e delega».

### N6 — `montar_test.go`: o teste não distingue `Esperar` de um no-op — nit (recomendo **park**)

`TestMontarDevolveServicoPronto` (`internal/boot/montar_test.go:12-35`) cancela o
ctx, fecha o watcher e afirma que `c.Esperar()` volta antes de
`vaulttest.Prazo`. **Apague `c.espera.Add(1)` e `defer c.espera.Done()`
(`montar.go:200,202`) e o teste continua passando** — `Wait()` sobre um
WaitGroup zerado volta na hora. Ele fixa "Esperar não trava", não "Esperar
espera", e é justamente a espera que impede o órfão que o `test_orphans.ps1`
caça.

Sendo honesto sobre o custo: fixar de verdade exige um sinal observável de que a
goroutine terminou, e a produção hoje não expõe nenhum que não seja sujeito a
corrida com o `cancel()` deste teste (`c.Inverted.Building()` depois do cancel
pode ser `true` ou `false` legitimamente). O teste veio literal do brief e o
gate de órfãos cobre a ponta a ponta. **Recomendo parquear** e resolver junto de
uma Task que possa tocar a produção — não vale inventar um teste flaky agora.

### N7 — enumeração incompleta no Concern 5 — nit

O Concern 5 lista os comentários de outros pacotes que ainda citam
`cmd/gobsidian/serve.go`, e o orquestrador já o parqueou para a passada de
documentação. Falta um sítio na lista, e é o mais visível de todos:
**`internal/daemon/daemon.go:40`** — «monta esse Server uma unica vez, via
`construirServico`, antes de chamar». Citar símbolo apagado no comentário do
pacote que o `cmd/gobsidian/daemon.go` serve é o pior lugar para isso ficar.
Acrescentar à lista parqueada para a limpeza não passar por cima.

### N8 — `docs/ESTADO.md` cita os nomes antigos — nit, nenhuma ação

`ESTADO.md:155` («Cobertura por função, 2026-09-02»: `construirServico`,
`carregarIndiceDoCache`, `prepararIndiceDeBusca`, `buildInvertedIndex`) e
`ESTADO.md:158` (`cmd/gobsidian/servico.go:78`). São medições **datadas**, num
documento que é história por definição — mesma natureza da dispensa de
`OPERACAO.md`, e reescrevê-las seria reescrever a história. Registrado só para
que a próxima leitura não os reencontre como se fossem achado novo.

---

## Concerns judged

| # | O que é | Julgamento |
|---|---|---|
| 1 | `AbrirIndice` mantém o `return nil, err` cru e o texto `"falha ao salvar cache de indice de metadados"` do original, contra o código ilustrativo do brief | **accept** — decisão já tomada pelo orquestrador, e a verificação mecânica confirma que é isso mesmo que está no commit (`indice.go:19-22`, diff vazio contra o original) |
| 2 | `(ver Task 177 …)` removido do `doc.go` | **accept** — ruling do orquestrador; e uma referência a Task inexistente é deliberação no código |
| 3 | Comentário passou a nomear `serveEmProcesso` em vez de `runServe` | **accept** — ruling do orquestrador. Conferido: é a **única** outra alteração de comentário em `busca.go`, e o fato descrito continua verdadeiro (`serve.go` chama `c.Esperar()` depois de `lifecycle.Shutdown`) |
| 4 | `TestInvertedCacheState` e `TestBuildInvertedIndexNaoAbrePlaceholderDeNuvem` não renomeados | **park** — o argumento é bom (os dois nomes são a rastreabilidade de provas de mutação em `task-128-report.md` e `2026-08-16-revisao-fixes.md`), o brief não pede, e renomear sem nota nos dois documentos custa mais do que resolve |
| 5 | Comentários `.go` de outros pacotes citando `cmd/gobsidian/serve.go` | **park** (já parqueado pelo orquestrador) — **mas a lista está incompleta**: ver **N7**, `internal/daemon/daemon.go:40` |
| 6 | `docs/wiki/` desatualizado | **park** — wiki é derivada e tem `status: stale` próprio; foi corretamente registrado em vez de silenciado |
| 7 | `§2.14` fora de ordem de propósito | **accept** — a justificativa (renumerar §2.2–§2.13 quebra as referências cruzadas) está escrita no próprio parágrafo, que é o que impede alguém de "consertar" sem ver o custo. Ortogonal a **N3** e **N4**, que são sobre o *conteúdo* do §2.14 |
| 8 | `Componentes` guarda `sync.WaitGroup` por valor | **accept** — é o que o brief especifica e está correto; conferi que nada copia a struct, e `copylocks` no `go vet` dos três GOOS é a proteção automática que o concern descreve |

---

## Verdict

**APROVADO com should-fix.** Nenhum bloqueante. A extração é o que o brief pediu:
corpos verbatim (verificado por `diff`, não por leitura), grafo de dependências
exatamente as seis arestas que `cmd/gobsidian` já tinha (verificado por
`go list`), `Esperar()` na mesma posição que o `wg.Wait()` ocupava nos dois
chamadores, linha `servidor pronto` intacta para o `measure.ps1`, prova de
mutação no passado com saída colada sobre uma âncora que confirmei única e no
laço certo, e um relatório que escreve "não medi" onde não mediu.

O que precisa de conserto é **só documentação**, e três dos cinco itens são
afirmações falsas sobre o estado da árvore: **N1** (a CLI não chama `boot`),
**N2** (`ARCHITECTURE.md:568` nomeia símbolo apagado, num arquivo que o brief
mandava editar), **N3** (§2.14 reintroduz o achado A6 da auditoria), **N5**
(`ESTRUTURA.md:235`, o `:216` do Step 6, ficou para trás). **N4** é nit no mesmo
parágrafo. **N6** e **N7** recomendo parquear, **N8** não pede ação.

Sugestão de despacho: N1–N5 são cinco edições de uma linha cada, num commit
`docs:` único. Não justificam re-abrir a Task 174 no código.
