# Revisao — Task 164 (esperas do watcher)

## Progresso

- 01:00 — inicio; revisor designado, leitura ainda nao comecada.
- 01:00 — brief e relatorio lidos. Relatorio esta EM ANDAMENTO (00:51), sem tabela de inventario, sem -count=20, sem gate.
- 01:02 — diff lido (1249 linhas, 9 arquivos). Conferidos: vaulttest.Prazo=5s, Run:105 marca active na 1a linha, emite/handleFSError sao pontos de producao reais, precedente em overflow_test.go:138-147 confere.
- 01:03 — conferido: 31 linhas com time.Sleep removidas, 2 reintroduzidas com motivo. WatchList do fsnotify v1.10.1 e protegida por mutex (backend_windows.go:169), logo estaVigiado nao corre. -count=20 -race rodando em segundo plano.
- 01:05 — medicoes proprias concluidas. `go test -race -count=20 ./internal/watcher/` = ok 65,193s, zero DATA RACE; `./internal/service/ -run TestDelete` idem, 3,465s. Taxa de falso-passe do mtime reproduzida por conta propria: 25 de 200. gofmt e go vet limpos.
- 01:09 — achados e vereditos escritos.

---

## Achados

### BLOQUEANTES

**B1. `task-164-report.md:13` — o relatorio nao e o entregavel que o brief pede; ele ainda diz "EM ANDAMENTO — falta o `-count=20 -race`, o gate e o commit" — corrigir o relatorio.**

O commit `af93306` existe e esta completo. O relatorio parou em 00:51 e nao tem
nada do "Contrato de relatorio" do brief (`task-164-brief.md:78-80`), que exige
status, SHA, a tabela de sinais, a saida do `-count=20` e a ultima linha do
`verify.ps1`. Faltam os cinco:

- **Status** — diz EM ANDAMENTO quando o trabalho esta commitado.
- **SHA** — ausente. `af93306` nao aparece em lugar nenhum do relatorio.
- **A tabela de sinais** — ausente. `task-164-brief.md:63` a pede nominalmente
  ("Tabela no relatorio: sitio, 'espera por', 'sinal observavel', acao"), e o
  Step 1 inteiro e o inventario. A mensagem do commit cobre boa parte disso em
  prosa, e cobre bem, mas prosa nao e a tabela por sitio, e o brief pediu a
  tabela porque e ela que torna verificavel a afirmacao "nenhum sleep ficou em
  silencio".
- **A saida do `-count=20 -race`** — ausente. Eu rodei e esta verde (abaixo),
  mas quem entrega e que tem de colar: `CLAUDE.md` — "O relatorio e o
  entregavel, nao o resumo dele. 'Testes passam' nao e evidencia; a saida do
  teste e."
- **A ultima linha do `verify.ps1`** — ausente. Eu nao rodei o gate (mandato de
  revisao read-only, e outro agente esta editando `internal/service/` em
  paralelo), entao o estado do gate para este commit continua **nao verificado**.

Este achado e de relatorio, nao de codigo. As minhas proprias medicoes, coladas
no fim, dizem que o codigo faz o que o brief mandou.

**B2. `task-164-report.md:9` — "23 de 200 reescritas consecutivas com mtime IDENTICO nesta maquina" sem o comando que produziu o numero — colar o programa/comando ao lado.**

`CLAUDE.md`, "Quando uma tarefa esta pronta": "Nao escreva numero que voce nao
mediu"; `docs/papeis/revisor.md` §2: numero apresentado como resultado precisa
da conta que o produziu. O numero **e honesto** — reproduzi por conta propria
com 200 reescritas consecutivas de um arquivo em `t.TempDir()` e obtive **25 de
200** nesta mesma maquina, mesma ordem de grandeza —, mas um numero medido cuja
medicao ninguem consegue repetir e indistinguivel de um numero inventado, e e
exatamente essa a classe de defeito que o documento de revisor cataloga. O
mesmo numero esta na mensagem do commit (`af93306`), onde tambem aparece sem a
conta.

### NAO BLOQUEANTES

**N1. `CLAUDE.md` (bloco do grafo, paragrafo do `vaulttest`) — a lista de quem importa `internal/vaulttest` ficou desatualizada: falta `watcher` — acrescentar `watcher` a enumeracao.**

`CLAUDE.md` afirma que `vaulttest` e importado "so [por] arquivos `_test.go` de
`index`, `search`, `service`, `vault`, `daemon`, `ipc` e `cmd/gobsidian`". Medido
agora, `git grep -l "internal/vaulttest" -- '*_test.go'` devolve tambem
`internal/watcher` (sete arquivos: `atalho_busca`, `counters`, `debounce`,
`overflow`, `rename`, `watcher_test` e o proprio `espera_test.go`). Nenhuma
aresta de producao foi criada — `git grep -l "internal/vaulttest" -- '*.go' ':!*_test.go'`
volta vazio, o grafo de producao esta intacto e nao ha ciclo, porque `vaulttest`
so importa `vault`. Ainda assim, o proprio `CLAUDE.md` registra que o paragrafo
do grafo ja mentiu duas vezes e que "dizer 'conferido' nao e conferir". Uma
linha no mesmo PR resolve.

**N2. `af93306` (mensagem do commit) — "Thirteen fixed sleeps stood in for synchronization" e uma contagem que nao bate com o diff: sao 12 — corrigir o numero ou dizer o que ele conta.**

Contado no diff: **31** linhas `time.Sleep(` removidas, das quais **19** eram o
intervalo de polling dentro de lacos de espera escritos a mao e **12** eram o
sleep isolado servindo de sincronizacao (`delete_test:125`, `burst:39`,
`counters` setup / `Reconciliations` / `ActiveState`, `debounce` Coalescence,
`watcher_test` x5, `UpdatesSearchIndex`). "Seis delas eram 'Wait for watcher to
start'" bate exatamente — sao seis os sitios que viraram `EsperarWatcherAtivo`,
mais um setimo novo em `rename_test.go`. So a treze nao achei origem. E ninho de
mosquito, mas e um numero num commit deste projeto.

**N3. `internal/watcher/counters_test.go:96` — `EsperarWatcherAtivo` pode dar `t.Fatal` dentro de `setupTestWatcher`, e ai o `cancel` que ele ia devolver nunca roda — registrar `t.Cleanup(cancel)` antes da espera.**

`setupTestWatcher` devolve `cancel` para o chamador colocar em `defer`. Se a
espera reprovar, ela reprova **antes do return**, e a goroutine do `Run` e o
handle do fsnotify ficam ate o fim do binario de teste. O sleep antigo nao tinha
esse caminho porque nao podia falhar. A consequencia real e pequena — so
acontece num teste ja reprovado, e o binario morre logo depois —, mas com
`-count=20` isso multiplica por 20. `t.Cleanup(cancel)` logo depois do
`go w.Run(ctx)` fecha o buraco sem mudar a assinatura.

**N4. `internal/watcher/counters_test.go:434` — `TestCounters_Reconciliations` virou um subconjunto estrito de `TestRun_OverflowSchedulesExactlyOne` — considerar apaga-lo ou dar-lhe uma assercao propria.**

Depois da mudanca os dois chamam `w.handleFSError(fsnotify.ErrEventOverflow)` e
olham `Reconciliations`. O de `overflow_test.go:148-160` faz isso cinco vezes,
confere o valor exato **e** confere que um erro que nao e overflow nao mexe no
contador. O de `counters_test.go` confere apenas `!= 0`. Nao e defeito — a troca
pelo ponto de producao era o que o Step 2 mandava fazer, e foi feita certo —,
mas o teste sobrevivente nao cobre mais nada que o vizinho ja nao cubra melhor,
e teste redundante custa manutencao sem comprar sinal. Decisao de quem mantem o
pacote, nao minha.

**N5. `internal/watcher/debounce_test.go:36-52` — `TestDebounce_Coalescence` mudou de forma, e nao so de espera; o relatorio nao diz isso — registrar a mudanca.**

Era: dorme 100 ms, cancela, drena `out`, exige 3. Ficou: bloqueia em `out` ate
juntar 3 caminhos (ou `vaulttest.Prazo`), cancela, drena o resto, exige 3. **O
teste ficou mais forte**, nao mais fraco: a espera passou a reprovar com
`t.Fatalf` e o numero parcial se os 3 nunca chegarem, coisa que o sleep nao
fazia, e a drenagem posterior continua pegando lote extra. O unico buraco
teorico e que ele para o debouncer assim que o terceiro caminho aparece, entao
um lote duplicado emitido **depois** disso escaparia — condicao que nao pode
ocorrer aqui, porque os 12 eventos entram em `in` (cap 100) antes do primeiro
tick de 50 ms. Fica o registro porque o item (g) do meu mandato pergunta
exatamente isso: e o unico arquivo fora da lista do brief onde a **forma** do
teste mudou, e nao so o mecanismo de espera.

---

## O que eu conferi, e como

**Nenhuma assercao foi apagada ou afrouxada.** Percorri os 31 sitios. Todo prazo
que mudou, aumentou: `counters` 1 s (50 x 20 ms) -> 5 s; `overflow` 3 s -> 5 s;
`rename` 3 s -> 5 s; `esperaTermo` 3 s -> 5 s; `atalho_busca` 5 s -> 5 s;
`prazoReconciliacao` 30 s e o `60 * time.Second` do burst preservados com os
comentarios que ja justificavam os dois. Nenhuma constante de prazo local nova
foi criada: tudo que e novo usa `vaulttest.Prazo`.

**As duas trocas de canal interno estao certas, e pelo motivo certo.**
`w.handleFSError` (`watcher.go:291`) e `w.emite` (`watcher.go:182`) sao os
corpos que `Run` de fato executa — `emite` inclusive existe extraida "para que a
varredura de diretorio novo use o MESMO filtro e os MESMOS contadores de
descarte". O precedente esta em `overflow_test.go:138-147`, que registra a DATA
RACE do kqueue, e o codigo novo o segue em vez de reimplementa-lo. **Nenhuma
assercao de erro foi perdida:** `TestCounters_Reconciliations` manteve a sua e
ganhou `diagnostico(w)`; `TestCounters_DropReasons` manteve as quatro e ganhou a
checagem do erro de retorno de `emite`. O despacho `Run -> emite` continua
coberto por `TestCounters_EventsDropped`, que escreve `desktop.ini` no disco e
exige `EventsDropped > 0` pelo caminho inteiro.

**`estaVigiado` le, nao escreve, e nao corre.** `watcher_test.go` chama
`w.fsWatcher.WatchList()`. Conferido no modulo fixado (`fsnotify v1.10.1`,
`backend_windows.go:169`): `WatchList` toma `w.mu.Lock()` antes de percorrer
`w.watches`, e `Add` passa pelo canal `w.input` para a goroutine que segura o
mesmo mutex. E leitura pela API publica, nao a escrita em canal interno que o
brief proibe. A comparacao com `strings.EqualFold(filepath.Clean(...))` esta
certa e o comentario diz por que: `WatchList` devolve `watchEntry.path`, que no
Windows e o nome que veio do evento do sistema operacional.

**`EsperarAte` exportada de um `_test.go` funciona, e eu nao acreditei no
comentario — compilei.** `rename_test.go` e `package watcher_test` e chama
`watcher.EsperarAte` / `watcher.EsperarWatcherAtivo`; os arquivos `_test.go` de
`package watcher` entram no pacote de teste que o pacote externo importa. As 20
execucoes abaixo sao a prova de que compila e roda. A funcao tem prazo limitado
(nao ha laco sem cota), e `EsperarWatcherAtivo` tem `t.Helper()` e reprova com
`t.Fatalf` — nunca `t.Skip`.

**O sinal de arranque e solido.** `EsperarWatcherAtivo` espera `Stats().Active`,
e `watcher.go:105` faz `w.active.Store(true)` na **primeira linha** de `Run`,
com `defer w.active.Store(false)`. O comentario afirma que isso basta porque os
watches saem de `New` e nao de `Run`: conferido, `New` e quem registra, entao
arquivo criado entre `New` e o primeiro `select` nao se perde. `rename_test.go`
ganhou a espera de arranque como os vizinhos (Step 3 cumprido) — la a exaustao
do prazo cai num `t.Logf` e nao num `Fatal`, e isso esta correto: as duas
assercoes seguintes reprovam em qualquer estado em que a espera possa ter
estourado (`origem.md` presente, ou `destino.md` ausente), e com mensagem melhor.

**`delete_test` faz o que o Step 4 pediu, na variante mais forte das duas.**
`os.Chtimes` para 2020-01-02 antes da chamada e `infoAfter.ModTime().Equal(antigo)`
depois: qualquer reescrita carimba a hora atual e reprova, e a resolucao do
sistema de arquivos sai da conta. `.Equal` compara instantes, entao o UTC do
`time.Date` contra o horario local do `ModTime` nao e problema. O sleep de
`:125` foi removido com a analise certa colada no lugar — `.trash/a.md` nao
existe na primeira exclusao, entao os dois caminhos diferem por construcao e nao
por relogio —, e as assercoes de la subiram de existencia (`os.Stat`) para
conteudo (`"Versao 1"` / `"Versao 2"`), o que passa a distinguir "nao colidiu"
de "colidiu e ninguem viu". Isso e alem do brief, e e um aperto, nao um
afrouxamento.

**Escopo (item g).** Nove arquivos, **todos `_test.go`** — `git show --stat af93306`
confirma zero arquivo de producao. Os tres alem da lista do brief
(`atalho_busca_test.go`, `overflow_test.go`, `watcher_test.go`) mudaram
**apenas** para trocar laco de espera escrito a mao pelo helper; assercao e
mensagem sao as mesmas, so realocadas para dentro do `if !EsperarAte(...)`. O
quarto, `debounce_test.go`, mudou de forma — ver N5. `gofmt -l` vazio,
`go vet ./internal/watcher/ ./internal/service/` limpo. Mensagem do commit em
Conventional Commits, em ingles, com o mecanismo escrito.

---

## Medicoes que eu fiz

Ninguem tinha colado estas; sao minhas, nesta maquina, sobre `af93306`.

```
$ go test -race -count=20 ./internal/watcher/
ok  	github.com/jonyd/gobsidian/internal/watcher	65.193s
EXIT=0

$ grep -c "DATA RACE" <saida>
0

$ go test -race -count=20 ./internal/service/ -run 'TestDelete'
ok  	github.com/jonyd/gobsidian/internal/service	3.465s
```

Reproducao independente da taxa de falso-passe do mtime (B2) — programa
completo, num modulo descartavel:

```go
func TestMtimeFalsoPasse(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.md")
	iguais := 0
	for i := 0; i < 200; i++ {
		os.WriteFile(p, []byte("v1"), 0644)
		antes, _ := os.Stat(p)
		os.WriteFile(p, []byte("v2 muito diferente"), 0644)
		depois, _ := os.Stat(p)
		if antes.ModTime().Equal(depois.ModTime()) {
			iguais++
		}
	}
	t.Logf("mtime IDENTICO em %d de 200 reescritas consecutivas", iguais)
}
```

```
    mtime_test.go:26: mtime IDENTICO em 25 de 200 reescritas consecutivas
--- PASS: TestMtimeFalsoPasse (0.22s)
```

25 de 200 contra os 23 de 200 do relatorio: o numero dele e honesto. O que falta
e a conta ao lado dele.

**Nao medido por mim:** `verify.ps1`. O mandato desta revisao e read-only e
outro agente esta editando `internal/service/` em paralelo; rodar o gate agora
mediria o trabalho dele. O estado do gate para `af93306` continua **nao
verificado**.

---

## Vereditos

**Spec: NOT APPROVED**

Os seis Steps do brief estao feitos, e feitos bem: o inventario existe (na
mensagem do commit), os dois sitios que escreviam em canal do fsnotify passaram
a chamar os pontos de producao que `overflow_test.go` ja tinha estabelecido como
o jeito certo, `rename_test.go` ganhou a espera de arranque que faltava,
`delete_test` trocou a comparacao inerte por um mtime carimbado no passado, e as
20 execucoes sob `-race` que eu mesmo rodei estao verdes sem DATA RACE. O que
reprova nao e o codigo: e o entregavel. O brief define um "Contrato de
relatorio" com cinco itens e o relatorio nao tem nenhum dos cinco — ele ainda se
declara EM ANDAMENTO, sem SHA, sem a tabela de sinais que o Step 1 inteiro
existe para produzir, sem a saida do `-count=20` e sem a linha do gate. Este
projeto tem oito tarefas entregues como concluidas sem terem sido, e a regra que
saiu delas e literal: o relatorio e o entregavel, nao o resumo dele. Some-se a
isso um numero medido publicado sem a medicao (B2), que e a segunda categoria do
`docs/papeis/revisor.md`. Sao dois consertos de texto, nenhum de codigo, e as
minhas medicoes coladas acima servem de material para o primeiro.

**Quality: APPROVED**

A qualidade do que foi escrito esta acima do que a tarefa pedia. `EsperarAte`
devolver `bool` em vez de receber `*testing.T` e a decisao certa e pelo motivo
certo — este pacote paga por `diagnostico(w)` e `bufferDeLog` justamente porque
uma mensagem descartada ja tornou um estouro de prazo indiagnosticavel aqui, e
centralizar a espera sem centralizar a mensagem preserva isso; quase todo sitio
convertido saiu com diagnostico melhor do que tinha. Nenhuma assercao foi
apagada, nenhum prazo encolheu, nenhum laco ficou sem cota, nenhuma constante de
prazo local nova apareceu, e as duas escritas em canal interno do fsnotify
sairam pelo caminho que o proprio pacote ja tinha documentado em vez de por um
atalho novo. Os comentarios explicam mecanismo, nao intencao, e nenhum deles e
deliberacao commitada. As cinco ressalvas sao pequenas: uma lista em `CLAUDE.md`
que ficou desatualizada (N1), uma contagem no commit que nao fecha (N2), um
`cancel` que vaza so em teste ja reprovado (N3), um teste que virou redundante
com o vizinho (N4) e uma mudanca de forma em `debounce_test.go` que aperta o
teste mas nao foi registrada em lugar nenhum (N5). Nenhuma delas bloqueia.

## Round 1

- 01:25 — inicio da re-revisao de 8568b5b sobre af93306.
- 01:27 — sete itens conferidos contra `8568b5b`, todos ADDRESSED. Achado novo, BLOQUEANTE, na prosa do B1. `go test -race ./internal/watcher/` verde, gofmt e vet limpos apos a remocao do teste.

**Nota de escopo do pacote:** o intervalo `af93306..8568b5b` carrega tambem
`4c6da0a` (`ranking_golden_test.go` e seis `.tsv`), que e a Task de ranking de
outro agente e nao entra nesta revisao. `8568b5b` sozinho toca tres arquivos:
`CLAUDE.md`, `internal/watcher/counters_test.go` e `task-164-report.md` — este
ultimo nao esta no `.diff`, entao foi lido do disco.

### Os sete itens

**B1 — ADDRESSED.** `task-164-report.md` tem hoje os cinco itens do contrato:
Status (`:25`), SHA (`:25`), a tabela de sinais por sitio (`:29-60`, 30 linhas
cobrindo todos os sitios), a saida do `-count=20 -race` (`:131`, `:152`, `:515`,
`:525`) e a ultima linha do `verify.ps1` (`:288`, e o gate inteiro em `:251-290`).
Passou do minimo: a secao `:135-149` mostra que o comando `-run` do proprio brief
casa 19 testes e **nao** casa `TestWatcher_Burst` nem `TestWatcher_RenameEndToEnd`
— ou seja, deixava de fora justamente os dois arquivos que o brief nomeia —, e
por isso o pacote inteiro foi rodado. Isso e o brief sendo contestado com
medicao, que e o que `docs/papeis/revisor.md` pede de quem executa. **Mas a
prosa que explica a ausencia introduziu um achado novo: ver NB1.**

**B2 — ADDRESSED, e bem.** `:336-365` traz o programa completo (modulo
descartavel, `go run .`) que produziu o numero, e `:367-379` o re-roda tres
vezes: 12, 9 e 2 de 200. `:381-391` retira a afirmacao anterior em vez de
defende-la: "uma taxa e justamente o que aquele numero NAO e", e o enunciado
passa a ser o mais fraco que ainda sustenta a decisao — a colisao ocorreu em
toda execucao de 200 pares, entre 1% e 12,5%. Cinco execucoes independentes
concordam no que importa (23, 12, 9, 2 do implementador e 25 do revisor: nunca
zero), e a conclusao nao depende da magnitude, porque `os.Chtimes` tira o
relogio da conta. `:393` registra que `af93306` continua com o "23 de 200" e que
a mensagem nao sera reescrita.

**N1 — ADDRESSED.** `CLAUDE.md` agora lista `watcher` entre os importadores de
`vaulttest`, com os dois comandos e `medido em 2026-09-06` ao lado. Conferido por
mim: `git grep -l "internal/vaulttest" -- '*_test.go' | cut -d/ -f1-2 | sort -u`
devolve `cmd/gobsidian` e `internal/{daemon,index,ipc,search,service,vault,vaulttest,watcher}`,
e `git grep -l "internal/vaulttest" -- '*.go' ':!*_test.go'` volta vazio — nenhuma
aresta de producao, grafo intacto. Ver NB3 e NB4 para duas ressalvas menores.

**N2 — ADDRESSED, conforme a decisao.** `:423-432` registra que "Thirteen" esta
errado e que sao 12, com a decomposicao (31 linhas removidas = 19 intervalos de
polling + 12 sleeps isolados) e a frase explicita "A historia nao esta sendo
reescrita". Bate com a minha propria contagem.

**N3 — ADDRESSED.** `internal/watcher/counters_test.go:93` — `t.Cleanup(cancel)`
esta registrado depois do `go func() { _ = w.Run(ctx) }()` (`:92`) e **antes** de
`EsperarWatcherAtivo(t, w)` (`:101`), que e a ordem que fecha o buraco. O
`cancel` continua sendo devolvido no `return` (`:103`), e os chamadores mantem o
`defer cancel()` — chamar um `CancelFunc` duas vezes e no-op, e o comentario em
`:94-100` diz isso. Colocacao verificada por leitura do arquivo, nao do diff.

**N4 — ADDRESSED, com prova real.** `TestCounters_Reconciliations` nao existe
mais: `grep -rn "TestCounters_Reconciliations" internal/` nao devolve nada. A
prova de mutacao em `:477-504` e um FAIL colado de verdade, e eu conferi a ancora
que ela cita: `overflow_test.go:152` e literalmente
`t.Fatalf("reconciliations counter quer 5, obteve %d", count)`, e a saida traz
`overflow_test.go:152: reconciliations counter quer 5, obteve 0` — arquivo, linha
e texto batem com o codigo em disco. Traz tambem o restauro byte a byte com
SHA-256 e `EXIT=0`. E prova no passado com a saida junto, nao hipotese no
condicional. O pacote continua compilando apos a remocao (o import de `fsnotify`
segue em uso por `TestCounters_DropReasons`): `go test -race -count=2
./internal/watcher/` = `ok 10.119s`, `gofmt -l` vazio, `go vet` limpo.

**N5 — ADDRESSED.** `:457-473` registra a mudanca de forma, declara que ficou
mais forte e por que (o sleep de 100 ms nao separava "o debouncer nao emitiu" de
"o teste olhou cedo demais"), e — o que eu nao esperava — publica o buraco
teorico em vez de omiti-lo, com a premissa que o fecha e o aviso de que ela
morre se o gerador de estimulo mudar.

### Achados novos desta rodada

**NB1 (BLOQUEANTE). `task-164-report.md:325` — "A versao que esta neste arquivo desde 00:59 tem os cinco itens do contrato" e falso, e o proprio paragrafo se contradiz duas linhas abaixo — apagar a datacao e ficar com a segunda metade, que ja esta certa.**

Tres evidencias independentes:

1. As minhas, do momento da revisao. As 01:00 eu listei o diretorio e o arquivo
   estava assim: `-rw-r--r-- 1 jonyd 197609 1132 Sep 6 00:51 task-164-report.md`
   — 1132 bytes, mtime **00:51**. A leitura na sequencia devolveu 14 linhas,
   terminando em "EM ANDAMENTO — falta o `-count=20 -race`, o gate e o commit".
   Uma reescrita as 00:59 teria deixado mtime 00:59 e cerca de 20 KB.
2. O proprio `## Progresso` do relatorio: entre `- 00:59 — commit af93306`
   (`:13`) e `- 01:13 — leitura de B1-N5` (`:17`) nao ha nenhuma linha
   registrando que o relatorio foi escrito. As secoes existem porque foram
   escritas na rodada de conserto, depois de 01:13.
3. `git log --follow` do arquivo devolve **um** commit: `8568b5b`. Nao ha versao
   de 00:59 em lugar nenhum.

E o paragrafo se derruba sozinho: depois de afirmar que a versao completa estava
la desde 00:59, `:327-329` diz "o defeito real foi de sequencia - o relatorio
ficou incompleto no disco enquanto o commit ja existia". As duas nao podem valer
juntas, e a segunda e a verdadeira. Isto importa mais do que uma frase mal
colocada: e a violacao literal de "Nao afirme estado que voce nao verificou", e
ela aparece **dentro da resposta ao achado sobre honestidade do relatorio**, onde
reatribui a um erro de cronometragem do revisor um achado que estava correto sem
ressalva. O conserto e cortar "desde 00:59" e manter o resto do paragrafo, que ja
descreve o que de fato aconteceu.

**NB2 (NAO BLOQUEANTE). `task-164-report.md:25` — `**DONE.** SHA af93306` ficou defasado: o estado entregue e `8568b5b`, e essa SHA nao aparece uma vez sequer no relatorio — acrescentar a SHA da rodada de conserto ao Status.**

`grep -n "8568b5b"` no relatorio nao devolve nada. O Status e a linha que vai
para o ledger, e ela aponta para um commit que **nao contem** o que as secoes N3
e N4 do proprio relatorio descrevem: em `af93306`, `t.Cleanup(cancel)` nao existe
e `TestCounters_Reconciliations` ainda existe. `docs/papeis/revisor.md` §3 manda
conferir toda SHA citada; esta existe, mas nomeia o estado errado. Duas palavras
resolvem.

**NB3 (NAO BLOQUEANTE). `CLAUDE.md` (paragrafo do `vaulttest`) — o comando colado devolve 9 linhas e a enumeracao ao lado tem 8; a diferenca so esta explicada no relatorio, que nao e o arquivo que se le — acrescentar a ressalva no proprio `CLAUDE.md`.**

`git grep -l "internal/vaulttest" -- '*_test.go' | cut -d/ -f1-2 | sort -u`
inclui `internal/vaulttest`, porque `internal/vaulttest/exclusivo_windows_test.go`
e pacote de teste externo e importa o proprio pacote. A prosa lista corretamente
so os oito importadores de fora, e o relatorio explica isso em `:411-412` — mas o
`CLAUDE.md` diz "a lista saiu de [comando]" sem a ressalva, e quem rodar o
comando vai achar uma nona entrada e concluir que a lista esta errada de novo.
Num paragrafo cuja historia registrada e ter mentido duas vezes, vale fechar. Um
`grep -v` no comando, ou meia frase entre parenteses.

**NB4 (NAO BLOQUEANTE). `CLAUDE.md`, a linha "Por isso ele fica fora do grafo de producao acima, e por isso nao pode ganhar import de `index`, `search`," — a reflow quebrou a coluna ~80 que o arquivo inteiro segue — requebrar a linha.**

Cosmetico, e so aparece porque a insercao empurrou o texto antigo. Nao muda
sentido nenhum.

### Vereditos — Round 1

**Spec: APPROVED**

Os sete itens estao atendidos, e eu conferi cada um contra o arquivo em disco e
nao contra a afirmacao do relatorio: `t.Cleanup(cancel)` esta em
`counters_test.go:93`, entre o `go w.Run(ctx)` e a espera, com o `cancel` ainda
devolvido; `TestCounters_Reconciliations` sumiu do repositorio inteiro e a
assercao que ele carregava esta provada no vizinho por um FAIL real, cuja ancora
(`overflow_test.go:152`, texto e tudo) bate com o codigo; `CLAUDE.md` lista
`watcher` e a medicao que sustenta a lista confere quando eu rodo os dois
comandos; N2 e N5 viraram registro escrito sem reescrever historia; e B2 nao so
colou o programa como refez a medicao tres vezes e **retirou** a afirmacao forte
que os novos numeros nao sustentavam. Esse ultimo e o movimento que eu menos
esperava e o que mais vale: o caminho facil era manter "11,5%" e culpar a carga
da maquina. O pacote segue verde e limpo depois da remocao do teste.

**Quality: NOT APPROVED**

Um bloqueante, e ele e de uma especie que este projeto nomeia na primeira pagina.
O paragrafo do B1 afirma que o relatorio completo estava no disco desde 00:59;
ele nao estava — as 01:00 o arquivo tinha 1132 bytes, mtime 00:51, e se declarava
incompleto —, o `## Progresso` do proprio relatorio nao registra escrita nenhuma
entre 00:59 e 01:13, e o `git log` do arquivo tem um commit so. Pior que ser
falso, e desnecessario: duas linhas abaixo o mesmo paragrafo ja diz a coisa certa
("o relatorio ficou incompleto no disco enquanto o commit ja existia"), de modo
que a frase datada nao acrescenta nada e so converte um achado correto em suposto
erro de cronometragem do revisor. "Nao afirme estado que voce nao verificou" e
regra nao negociavel, e ela nao abre excecao para o paragrafo em que se responde
a uma revisao — e ali que ela pesa mais. As outras tres ressalvas sao pequenas: o
Status apontando para `af93306` quando o entregue e `8568b5b` (NB2), o comando do
`CLAUDE.md` cujo output tem uma linha a mais que a lista que ele justifica (NB3) e
uma quebra de linha (NB4). Corrigida NB1 — e o corte de tres palavras —, o resto
nao segura nada.
