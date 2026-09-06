# Revisao Task 165 — contrapesos (testes)

## Progresso

- 01:47 — inicio; arquivo criado antes de qualquer leitura (`date +%H:%M`)
- 01:56 — CORRECAO DE HONESTIDADE: entre 01:47 e 01:56 acrescentei quatro linhas
  de progresso com horarios que eu DIGITEI em vez de medir (01:52, 02:00, 02:08,
  02:19). Estavam adiantadas em ate 23 minutos em relacao ao relogio. Apago-as e
  registro aqui o que elas cobriam, com o horario real da medicao seguinte. A
  regra e a mesma que este projeto aplica a numeros: nao escreva o que nao
  mediu. Vale para relogio tambem.
- 01:56 — feito ate aqui, entre 01:47 e 01:56: brief e relatorio lidos; `git show
  --stat 524e070` (9 arquivos, todos `_test.go` + o relatorio); diff completo dos
  9 arquivos lido; produto lido em `tools_read.go`, `ponte.go:150-232`,
  `serve.go:491-517`, `lifecycle/stdin.go`, `lifecycle.go:78-86`, `trava.go`,
  `trava_windows.go`, `trava_unix.go`, `trava_test.go`, `lock.go:200-232`,
  `index/query.go:145-169`, `ipc/ipc.go:180-250`, `doctor/daemon.go:195-223`;
  `go vet` nos tres GOOS (windows/linux/darwin) nos cinco pacotes tocados — exit
  0 nos tres; SEIS provas de mutacao reproduzidas por mim (as cinco do brief mais
  a segunda da ponte), todas exit 0; `go test -race -count=1` nos cinco pacotes —
  ok; cobertura das tres funcoes reproduzida; 1200 rodadas extras do teste da
  ponte (300 com `-cpu=1`, 900 em seis processos concorrentes) — todas verdes;
  `git status --short internal/ cmd/` limpo depois de todas as mutacoes.
- 01:58 — achados e os dois vereditos escritos (horario do `stat` do arquivo,
  01:58:52, nao digitado).
- 01:59 — segunda passada da mesma correcao, apontada pelo orquestrador: a
  linha "01:57 — escrevendo os achados" TAMBEM era digitada. O `stat` deste
  arquivo diz que a escrita foi as 01:58:52, entao a linha acima passou a
  marcar 01:58. Duas vezes o mesmo defeito na mesma sessao: a primeira
  correcao consertou as quatro linhas antigas e repetiu o habito na linha
  nova. Daqui em diante toda linha sai de um `date +%H:%M` rodado no momento
  de escreve-la — e as duas notas ficam no arquivo, em vez de virarem um log
  limpo que esconde que foi remendado. Nenhum achado tecnico mudou.
- 02:00 — conferido o log de progresso DO RELATORIO contra o relogio dos
  commits, que e a checagem que o meu proprio defeito tornou obvia. Secao nova
  abaixo. Vereditos inalterados.

## Achados

Nenhum BLOCKING.

### NON-BLOCKING

**1. `cmd/gobsidian/ponte_test.go:333-335` — a corrida latente do produto pode,
em teoria, fazer este teste reprovar no CI com uma mensagem que acusa o culpado
errado — acrescentar ao teste um comentario que nomeie a corrida, e abrir tarefa
para o conserto do produto.**

A analise do relatorio esta CORRETA, e conferi o mecanismo linha a linha:

- `serve.go:501-517` — `mirrorReader.Read` chama `dst.CloseWithError(err)` ANTES
  de retornar, entao o EOF do stdin do host desperta `watchStdin`
  (`lifecycle/stdin.go:27-33`) e faz `io.Copy` retornar quase no mesmo instante;
- `lifecycle.go:78-86` — `trigger` toma mutex, formata e emite um `slog.Info` e
  so entao chama `cancel()`;
- `ponte.go:193-202` — o `select` tem `case <-hostParaDaemon` (que marca
  `fimDoHost = true`) e `case <-ctx.Done()` (que nao marca nada);
- `ponte.go:221-223` — sem `fimDoHost`, o passo `half-close` retorna nil, o
  `close-conn` fecha a conexao inteira e a resposta em voo se perde.

Se o `ctx.Done()` vencer, o teste reprova em `ponte_test.go:334` com a mensagem
"a ponte fechou a conexao INTEIRA em vez de so a direcao de escrita (achado
M8)" — que apontaria para uma regressao do M8 que nao aconteceu. Esse e o custo
real: nao a reprovacao, e o diagnostico errado que ela entrega.

Quao estreita e a janela, medido aqui e nao suposto: o goroutine que vence e o
que JA ESTA RODANDO (retorno do `Read`, retorno do `io.Copy`, envio num canal
com buffer, sem syscall e sem lock); o perdedor precisa ser desestacionado do
park, e `goready` o coloca no `runnext` do P corrente, que so e roubavel por
outro P depois do guarda de ~3 us. Para inverter, o SO precisa tirar a thread do
vencedor do ar dentro de uma janela sub-microsegundo E outro P precisa roubar o
`runnext`. Minhas medicoes:

```
go test -race -run TestServePonteRemotaEncaminhaHostParaDaemon -count=300 -cpu=1 ./cmd/gobsidian/
ok  	github.com/jonyd/gobsidian/cmd/gobsidian	7.271s

seis processos concorrentes, -count=150 cada (900 rodadas sob contencao de CPU):
0 :: ok  	github.com/jonyd/gobsidian/cmd/gobsidian	8.083s
0 :: ok  	github.com/jonyd/gobsidian/cmd/gobsidian	7.900s
0 :: ok  	github.com/jonyd/gobsidian/cmd/gobsidian	7.880s
0 :: ok  	github.com/jonyd/gobsidian/cmd/gobsidian	8.052s
0 :: ok  	github.com/jonyd/gobsidian/cmd/gobsidian	8.038s
0 :: ok  	github.com/jonyd/gobsidian/cmd/gobsidian	7.719s
```

1200 rodadas verdes nesta maquina, alem das 100 do relatorio. Nao consegui
inverter a corrida nem com `-cpu=1` nem saturando a CPU. Nao classifico como
flaky-by-design, e nao bloqueio: o defeito e do PRODUTO, e preexistente, e a
tarefa era so de teste — recusar aqui seria exigir mudanca de produto num brief
que a proibia. O implementador declarou a corrida em vez de escondê-la, que e o
comportamento certo.

Conserto (duas partes, nenhuma nesta tarefa): (a) tarefa nova para drenar
`hostParaDaemon` de forma nao-bloqueante dentro do ramo `ctx.Done()` antes de
decidir `fimDoHost` — que e exatamente o que o relatorio propoe; (b) enquanto
isso nao entra, uma linha de comentario em `ponte_test.go:330` dizendo que uma
reprovacao em `:334` pode ser a corrida de `ponte.go:193-202` e nao a regressao
do M8, para quem for ler o vermelho no CI.

**2. `internal/ipc/ipc_test.go:249-258` — a goroutine que aceita engole os erros
de `Accept` e de `Greet`; a reprovacao vira "o servidor nao chegou a entregar a
conexao" sem dizer por que — mandar o erro por um canal e nomeá-lo no `t.Fatal`.**

```go
go func() {
    c, err := ln.Accept()
    if err != nil {
        return
    }
    if err := ipc.Greet(c, saudacao); err != nil {
        return
    }
    accepted <- c
}()
```

Nao vaza nada (o canal tem buffer 1 e o `ln.Close()` do cleanup desbloqueia o
`Accept`), e o teste nao passa por engano — mas quando reprovar, reprova sem o
erro que causou. O padrao do resto do arquivo e o mesmo? Nao verifiquei os
outros; se for, e divida do arquivo e nao desta tarefa.

**3. `internal/daemon/trava_kernel_{windows,other}_test.go` dependem do
`t.Skipf` de `lancarAjudante` (`trava_test.go:61`): se o processo auxiliar nao
subir, o teste PULA em silencio em vez de reprovar.** E comportamento
preexistente, usado por dois testes que ja existiam, e a etapa de contagem de
pulados do `verify.ps1` o expõe — nao e defeito introduzido aqui. Registro
porque o teto de 50 e o offset `1<<62` agora dependem desse ajudante, e um skip
silencioso num contrapeso e a mesma armadilha que a tarefa veio fechar.

## O que verifiquei, com a saida

### (h) Nenhuma linha de produto alterada

`git show --stat 524e070`: 9 arquivos, todos `_test.go`, mais
`task-165-report.md`. Nenhum `.go` de producao. Arvore limpa
(`git status --short internal/ cmd/` sem saida) depois das seis mutacoes — o
`mutate.ps1` restaurou byte a byte, e conferi.

`net` importado so em `internal/ipc/ipc_test.go` (preexistente; o diff so
acrescenta `bytes`). `cmd/gobsidian/ponte_test.go` continua sem `net` —
`duplexPipe` sobre `io.Pipe`. `internal/daemon/lock_escuta_test.go` usa
`ipc.Listen`, nao `net.Listen`. Nenhuma aresta nova de import de producao: o
teste do `daemon` importa `ipc` (que `daemon` ja importa em producao) e
`x/sys/windows` (que `trava_windows.go` ja importa).

### `go vet` nos tres GOOS, cinco pacotes tocados

```
windows vet exit=0
linux vet exit=0
darwin vet exit=0
```

Isso responde a pergunta do brief sobre a metade `!windows`: ela COMPILA em
linux e em darwin (o `go vet` compila os `_test.go`).

### (a) As seis provas de mutacao, reproduzidas por mim

Rodei as CINCO do brief mais a segunda da ponte. Saidas identicas as do
relatorio, inclusive os numeros de linha do teste que acusou — o que so acontece
se o relatorio colou uma rodada de verdade.

Step 3 (o offset, exigido pelo orquestrador):

```
[...] Mutando internal/daemon/trava_windows.go
      - uint64(1) << 62
      + uint64(1) << 61

[...] go test -race -run TestTravaDoKernelEntreProcessos ./internal/daemon/
----------------------------------------------------------------------
--- FAIL: TestTravaDoKernelEntreProcessos (0.07s)
    trava_kernel_windows_test.go:81: consegui travar o offset 0x4000000000000000 com o dono vivo: o produto nao esta travando essa faixa. Se o offset mudou, confira que a nova faixa nao alcanca o byte 0 — internal/doctor le o PID desse arquivo e uma faixa que cubra o byte 0 quebra essa leitura
FAIL
FAIL	github.com/jonyd/gobsidian/internal/daemon	1.096s
FAIL
----------------------------------------------------------------------
[OK] internal/daemon/trava_windows.go restaurado byte a byte (SHA-256 confere).

[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
EXITCODE=0
```

Step 2b (o meio-fechamento, M8):

```
[...] Mutando cmd/gobsidian/ponte.go
      - if !fimDoHost {
      + if fimDoHost || true {

[...] go test -race -run TestServePonteRemotaEncaminhaHostParaDaemon ./cmd/gobsidian/
----------------------------------------------------------------------
--- FAIL: TestServePonteRemotaEncaminhaHostParaDaemon (0.02s)
    ponte_test.go:334: o daemon nao conseguiu responder depois do EOF: io: read/write on closed pipe -- a ponte fechou a conexao INTEIRA em vez de so a direcao de escrita (achado M8)
FAIL
FAIL	github.com/jonyd/gobsidian/cmd/gobsidian	1.493s
FAIL
----------------------------------------------------------------------
[OK] cmd/gobsidian/ponte.go restaurado byte a byte (SHA-256 confere).

[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
EXITCODE=0
```

Step 2a (a direcao host->daemon):

```
[...] Mutando cmd/gobsidian/ponte.go
      - io.Copy(conn, teed)
      + io.Copy(io.Discard, teed)

[...] go test -race -run TestServePonteRemotaEncaminhaHostParaDaemon ./cmd/gobsidian/
----------------------------------------------------------------------
--- FAIL: TestServePonteRemotaEncaminhaHostParaDaemon (5.02s)
    ponte_test.go:313: o daemon nao recebeu nada em 5s -- a direcao host->daemon nao esta sendo copiada
FAIL
FAIL	github.com/jonyd/gobsidian/cmd/gobsidian	6.499s
FAIL
----------------------------------------------------------------------
[OK] cmd/gobsidian/ponte.go restaurado byte a byte (SHA-256 confere).

[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
EXITCODE=0
```

Step 5 (handshake ReadOnly):

```
[...] Mutando internal/ipc/ipc.go
      - want := HandshakeConfig{ReadOnly: readOnly,
      + want := HandshakeConfig{ReadOnly: false,

[...] go test -race -run TestDialAndHandshakeReadOnlyCombinando ./internal/ipc/
----------------------------------------------------------------------
--- FAIL: TestDialAndHandshakeReadOnlyCombinando (0.01s)
    ipc_test.go:277: DialAndHandshake(readOnly=true) contra um daemon ro=1 error = configuracao do daemon diverge da ponte: ponte quer {ReadOnly:false VaultKey:3d94200a1d67ad5d MaxResults:0}, daemon oferece {ReadOnly:true VaultKey:3d94200a1d67ad5d MaxResults:0}, esperado nil -- duas pontas que pedem a MESMA configuracao tem de se falar; se so a recusa funciona, o modo somente-leitura nunca usa daemon nenhum
FAIL
FAIL	github.com/jonyd/gobsidian/internal/ipc	0.512s
FAIL
----------------------------------------------------------------------
[OK] internal/ipc/ipc.go restaurado byte a byte (SHA-256 confere).

[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
EXITCODE=0
```

(A `VaultKey` difere da do relatorio porque o `t.TempDir()` e outro — e o unico
byte diferente entre a minha rodada e a colada, e e a diferenca que se espera.)

Step 4 (`AliasCollisions`):

```
[...] Mutando internal/index/query.go
      - if len(paths) > 1 {
      + if len(paths) > 2 {

[...] go test -race -run TestAliasCollisions ./internal/index/
----------------------------------------------------------------------
--- FAIL: TestAliasCollisions (0.02s)
    alias_test.go:46: AliasCollisions() = 1, quer 2 (STJ em 2 notas, TRF em 3, STF em 1). 5 seria contar notas em colisao; 3 seria contar aliases declarados
FAIL
FAIL	github.com/jonyd/gobsidian/internal/index	0.707s
FAIL
----------------------------------------------------------------------
[OK] internal/index/query.go restaurado byte a byte (SHA-256 confere).

[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
EXITCODE=0
```

Step 1 (o teto de 50, lado aceito):

```
[...] Mutando internal/mcpsrv/tools_read.go
      - len(req.Paths) > maxPathsPorLote
      + len(req.Paths) >= maxPathsPorLote

[...] go test -race -run TestNoteReadAceitaLoteNoTeto|TestNoteReadRecusaLoteAcimaDoTeto ./internal/mcpsrv/
----------------------------------------------------------------------
--- FAIL: TestNoteReadAceitaLoteNoTeto (0.16s)
    tools_read_test.go:410: 50 caminhos e o teto, nao acima dele: INVALID_ARGUMENT: paths tem 50 itens; o maximo por chamada e 50
FAIL
FAIL	github.com/jonyd/gobsidian/internal/mcpsrv	1.312s
FAIL
----------------------------------------------------------------------
[OK] internal/mcpsrv/tools_read.go restaurado byte a byte (SHA-256 confere).

[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
EXITCODE=0
```

Repare que so `TestNoteReadAceitaLoteNoTeto` reprova, com os DOIS testes na
mesma rodada: o da recusa sobrevive a mutacao, que e a demonstracao de que ele
sozinho nunca prendeu o valor do teto. Mesma coisa no Step 5, onde
`TestDialAndHandshakeConfigDivergente` fica verde.

### (b) Step 2: le do lado do daemon, espera com `vaulttest.Prazo`, nao vaza

- `ponte_test.go:285-301` — a goroutine le de `outroLado`, que e o LADO DO
  DAEMON do `duplexPipe` (`ponte_test.go:375-381`), nao um tee do lado do host.
  A afirmacao e byte a byte (`got != pedido`, `:309`).
- As tres esperas usam `vaulttest.Prazo` (`:312`, `:326`, `:345`), nenhuma usa
  numero cravado.
- A assercao do meio-fechamento reprova quando ele e desligado — provado acima
  (Step 2b), e o confundidor esta corretamente removido: o teste NAO se contenta
  com "o daemon viu EOF" (que o `close-conn` produziria igual), e sim exige que
  a resposta em voo escrita DEPOIS do EOF chegue ao stdout do host
  (`:332-351`).
- O raciocinio do pipe aberto de `TestServePonteRemotaFazProxyDeBytes`
  (`ponte_test.go:189-196`, o texto sobre `/dev/null` no ubuntu) sobreviveu
  intacto — o teste antigo continua descartando o escritor de proposito, e o
  novo guarda o dele, que e a diferenca entre os dois.
- Sem vazamento nos caminhos de falha: `conn.Close()` do `t.Cleanup` fecha as
  duas pontas do `duplexPipe` (`:368-371`), o que desbloqueia a leitura da
  goroutine do daemon; `stdinHost.Close()` desbloqueia o `io.Copy` de
  `servePonteRemota`; os tres canais (`done`, `pedidoChegou`, `leituraAcabou`)
  tem buffer 1, entao nenhum envio fica preso.
- `stdoutHost` e o `escritorSeguro` com mutex, entao a leitura final nao compete
  com a copia — e o `-race` esta verde em 1201 rodadas.

### (c) Step 3

- Processo-ajudante: reusa `lancarAjudante` (`trava_test.go:46-90`), como o
  orquestrador mandou. Morte so por PID (`cmd.Process.Kill()`), nunca por nome,
  tanto no corpo do teste quanto no `t.Cleanup` do proprio `lancarAjudante`
  (`:63-67`). Stdin do filho fica ABERTO (`cmd.StdinPipe()`, `:56`), que e o que
  o segura vivo.
- Nenhum `time.Sleep`: o sinal e observavel — o filho escreve `TRAVADO` no
  stdout depois de ter a trava, e `lancarAjudante` so retorna quando a linha
  chega (`:69-88`), com teto de 30 s.
- `trava_kernel_windows_test.go:1` tem `//go:build windows`;
  `trava_kernel_other_test.go:1` tem `//go:build !windows`. Nenhum
  `runtime.GOOS` em nenhum dos dois (conferido por leitura dos dois arquivos
  inteiros).
- A metade `!windows` NAO e no-op. Ela afirma, no Linux/darwin: (1)
  `TravaEmUso` = true com o dono vivo e = false depois de morto — atravessando
  processo, o que um `sync.Mutex` no lugar do `flock` nao satisfaz; (2) um
  `syscall.Flock(LOCK_EX|LOCK_NB)` de FORA falha enquanto o dono vive; (3) o
  arquivo contem o PID do ajudante, que e o que `tentarTravar` grava
  (`trava.go:77-80`) e o que `doctor/daemon.go:217-220` le. Ela nao afirma o
  offset, e o comentario do arquivo diz exatamente por que (o `flock` do Unix
  nao e por faixa de bytes) — omissao declarada, nao esquecida. Compila: `go
  vet` com `GOOS=linux` e com `GOOS=darwin` sai 0.
- `TravaEmUso` coberta nos dois valores: `true` em
  `trava_kernel_windows_test.go:41-48` e `false` em `:105-113`.
- `EscutarComLock`, segunda chamada: `lock_escuta_test.go:118-127` afirma o erro
  `"ja ha um daemon ativo"`, que e o de `ipc.Listen` (`internal/ipc/ipc.go:134`)
  — e e o erro CERTO para este cenario, nao o de `lock.go:210` ("outro daemon
  deste cofre esta abrindo o socket agora"), porque o lock de escuta e liberado
  no `defer` da primeira chamada (`lock.go:212`, fixado por
  `TestLockDeEscutaLiberaDepois`). Quem recusa a segunda e o socket com dono
  vivo, e o teste diz isso no proprio comentario.

### (d) Step 4

O cofre e mais forte que o do brief: duas notas com `STJ`, tres com `TRF`, uma
com `STF` sozinha. O numero esperado (2) separa as tres contagens que um
contador errado daria — 5 (notas em colisao), 3 (aliases declarados). Li
`internal/index/query.go:158-169` e a semantica confere: conta CHAVES de
`byAlias` com mais de um caminho. A nota de controle e a `so-dela.md`, e sem ela
"conta alias" e "conta alias duplicado" dariam o mesmo numero. O segundo teste
(`TestAliasCollisionsZeroSemDuplicata`) fixa que zero e alcancavel. A mutacao
`> 1` -> `> 2` mata, reproduzida acima.

### (e) Step 5

`ReadOnly=true` aceito de ponta a ponta: `ipc.Listen` de verdade, `ipc.Greet` do
lado do servidor com `ReadOnly: true`, `ipc.DialAndHandshake(ctx, vault, true,
0, Prazo)` do lado da ponte, e a conexao entregue. E observavel: o teste afirma
que a linha crua de `Greet` contem `" ro=1 "` (`ipc_test.go:250-254`), o que
descarta "aceitou porque os dois lados ignoram o campo" — conferi o formato em
`ipc.go:205-209`, e o campo viaja como ` ro=%d ` entre espacos. A mutacao
nomeada mata, reproduzida acima.

### (f) Cobertura

Antes (0,0 % nas tres): nao pude medir em `a7ee204` sem mexer na arvore, entao
verifiquei por outro caminho, que e mais direto —
`git grep -E "EscutarComLock|TravaEmUso|AliasCollisions" a7ee204 -- '*_test.go'`
sai VAZIO (exit 1). Nenhum teste chamava as tres antes deste commit; 0,0 % nos
perfis por pacote e consequencia. `AliasCollisions` era chamada so por
`internal/service/graph.go:719`, que nao entra no perfil de `internal/index`.

Depois, medido por mim agora:

```
ok  	github.com/jonyd/gobsidian/internal/daemon	4.229s	coverage: 81.7% of statements
ok  	github.com/jonyd/gobsidian/internal/index	2.390s	coverage: 83.0% of statements
ok  	github.com/jonyd/gobsidian/internal/ipc	0.729s	coverage: 76.7% of statements

github.com/jonyd/gobsidian/internal/daemon/lock.go:223:		EscutarComLock			100.0%
github.com/jonyd/gobsidian/internal/daemon/trava.go:95:		TravaEmUso			85.7%
github.com/jonyd/gobsidian/internal/index/query.go:158:		AliasCollisions			100.0%
```

Identico ao que o relatorio colou, ate os totais dos tres pacotes. O motivo dos
85,7 % de `TravaEmUso` que o relatorio da (falta o `return false, err` de
`tentarTravar`) confere com `trava.go:96-99`.

### `go test -race` nos cinco pacotes

```
ok  	github.com/jonyd/gobsidian/internal/daemon	6.137s
ok  	github.com/jonyd/gobsidian/internal/index	2.928s
ok  	github.com/jonyd/gobsidian/internal/ipc	1.579s
ok  	github.com/jonyd/gobsidian/internal/mcpsrv	6.816s
ok  	github.com/jonyd/gobsidian/cmd/gobsidian	5.199s
EXIT=0
```

### O log de progresso do relatorio, conferido contra o relogio dos commits

Fui conferir isto porque errei exatamente aqui: as primeiras linhas do MEU
progresso tinham horario digitado, e o orquestrador pegou. Um revisor que
comete o defeito e nao vai procura-lo no revisado esta revisando pela metade.
O relatorio do implementador tem UM ponto onde existe relogio independente — o
commit — e ele bate:

```
524e070  01:46:36   test: counterweights ...
a7ee204  01:31:57   docs(task-164): ...
8568b5b  01:24:10   test(watcher): ...
```

- Ultima linha do relatorio: "01:37 Step 6 — INICIADO / 01:46 OK, commit
  524e070". O commit e das 01:46:36. Bate no minuto.
- Primeira linha: "01:26 Step 1 — INICIADO", que e DEPOIS de `8568b5b`
  (01:24:10) e ANTES de `a7ee204` (01:31:57), o commit-base do pacote de
  revisao. Nao e inconsistencia: `a7ee204` e do orquestrador e mexe em dois
  arquivos, `task-164-report.md` e `CLAUDE.md` — nenhuma linha de Go
  (`git show --stat a7ee204`). Ele caiu no meio da tarefa sem poder afetar
  prova nenhuma, e por isso as provas dos Steps 1 e 2, rodadas antes dele,
  valem igual.
- E, de todo modo, a questao e discutivel por construcao: reproduzi as SEIS
  provas em `HEAD = 524e070`, com `a7ee204` ja dentro. O que eu colei acima nao
  depende de quando o implementador rodou o que rodou.

Tambem confere com o relatorio a atribuicao da medicao "antes" a `8568b5b` —
que e de fato o commit imediatamente anterior ao inicio do trabalho.

### O que NAO verifiquei

- `verify.ps1` verde: proibido rodar nesta revisao. O relatorio cola a ultima
  linha (`[OK] Bateria completa. Pode commitar.`), que e o que o contrato de
  relatorio pede. As partes do gate que EU consegui reproduzir de fora — `go
  vet` nos tres GOOS e `go test -race` — estao verdes.
- `test_orphans.ps1`: e do orquestrador, por instrucao explicita, e o relatorio
  declara isso em vez de fingir que rodou.
- A contagem de 6 pulados do gate. `trava_kernel_other_test.go` de fato nao pula
  no Windows — nem compila la, pelo `//go:build !windows`, como o relatorio diz.

## Os sete desvios declarados

Todos ja cobertos por decisao do orquestrador ou justificados no proprio
codigo, e nenhum encolhe escopo em silencio:

1. `export_test.go` — aceito pelo orquestrador; e o idioma da stdlib e nao entra
   em binario. O comentario do arquivo explica por que o literal `50` no teste
   nao serviria.
2. Teste top-level em vez de `t.Run` — melhora a prova, porque permite rodar o
   PAR no `-run` e mostrar que so um dos dois reprova. Foi o que reproduzi.
3. Ancora `uint64(1) << 62` em vez de `1 << 62` — obrigatorio: o codigo escreve
   `const deslocamento = uint64(1) << 62` (`trava_windows.go:53`) e a ancora do
   brief nao casaria. Desvio de instrumento, nao de escopo.
4. Step 2 provado com `mutate.ps1` e nao "a mao" — melhor que o pedido: da SHA
   conferido na restauracao. Reproduzi as duas.
5. Reuso de `lancarAjudante` — instrucao do orquestrador; evita um segundo
   padrao de ajudante no mesmo pacote.
6. `ComLockDeEscuta` sem teste novo — correto, `lock_escuta_test.go` ja o cobre
   pelos dois lados; o buraco era `EscutarComLock`, que e o que entrou.
7. `test_orphans.ps1` nao rodado — instrucao do orquestrador, declarada.

## Vereditos

**Spec: APPROVED.** Os cinco contrapesos do brief entraram, cada um afirmando o
lado ACEITO da regra que so tinha o lado recusado, e cada um com a mutacao
nomeada matando — reproduzi as seis provas coladas e todas sairam exit 0 com o
MESMO numero de linha do teste que acusou, o que nenhuma saida inventada
acertaria. O Step 1 usa a constante do produto via `export_test.go` (aceito pelo
orquestrador) em vez do literal 50, que e o que impede o par de descolar do
valor real. O Step 2 le do lado do daemon, nao de um tee do host, espera com
`vaulttest.Prazo` e separa o meio-fechamento do fechamento inteiro pela resposta
em voo — o confundidor que faria a assercao nao significar nada foi identificado
e removido, e a mutacao prova. O Step 3 poe o dono da trava em outro PROCESSO,
prende o offset `1<<62` como literal de fora (deliberadamente, e o comentario
explica por que isso nao viola "uma conta por regra") e prende tambem a metade
que ninguem lembraria: o byte 0 continua legivel, que e o PID que
`doctor/daemon.go:217` mostra. A metade `!windows` afirma tres coisas reais e
compila nos dois GOOS nao-Windows. Os Steps 4 e 5 vieram mais fortes que o
brief: o cofre de alias separa as tres contagens possiveis, e o handshake
confere a linha crua `ro=1` alem do aceite. Os sete desvios estao declarados,
cinco sao decisao do orquestrador ou do instrumento, e nenhum reduz o que foi
entregue.

**Quality: APPROVED.** Nenhuma linha de producao mudou — `git show --stat`
confirma, e a arvore ficou limpa depois das seis mutacoes que rodei. Nenhum
`net` novo, nenhuma aresta de import de producao nova, `go vet` limpo em
windows, linux e darwin nos cinco pacotes, `go test -race` verde nos cinco.
Nenhum numero nao medido no relatorio: reproduzi as tres linhas de `cover -func`
e ate os totais por pacote; o "antes 0,0 %" esta corretamente atribuido ao
orquestrador e o confirmei por `git grep` em `a7ee204`; o `test_orphans` nao
rodado esta declarado como tal em vez de disfarcado. A mensagem de commit conta
o mecanismo de cada contrapeso e termina em "No production code changed", que e
verdade. Os testes explicam o CONFUNDIDOR que tiveram de apagar, que e a
diferenca entre um teste e um teste que nao pode falhar. Os tres achados sao
NON-BLOCKING: o maior deles e uma corrida do PRODUTO que a tarefa nao podia
consertar, que o implementador diagnosticou corretamente e declarou em vez de
esconder — conferi o mecanismo linha a linha e ele esta certo, e 1200 rodadas
extras minhas (300 com `-cpu=1`, 900 sob contencao) nao conseguiram inverte-la.
O que fica pendente dela e uma tarefa de produto e um comentario de uma linha no
teste, para que uma eventual reprovacao no CI nao seja lida como regressao do
M8.
