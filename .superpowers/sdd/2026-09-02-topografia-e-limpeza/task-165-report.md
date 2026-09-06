# Task 165 — relatorio

## Progresso

- 01:26 Step 1 (limite 50 aceito) — INICIADO / 01:27 OK, mutacao exit 0
- 01:27 Step 2 (ponte host->daemon + half-close) — INICIADO / 01:31 OK, duas mutacoes exit 0
- 01:31 Step 3 (trava do kernel entre processos, TravaEmUso, EscutarComLock) — INICIADO / 01:34 OK, mutacao exit 0
- 01:34 Step 4 (AliasCollisions) — INICIADO / 01:35 OK, mutacao exit 0
- 01:35 Step 5 (handshake ReadOnly=true) — INICIADO / 01:36 OK, mutacao exit 0
- 01:37 Step 6 (cobertura, verify, commit) — INICIADO / 01:44 OK

## Status

**CONCLUIDA.** Cinco contrapesos escritos, cinco provas de mutacao com exit 0,
`verify.ps1` verde. Nenhuma linha de produto alterada — os cinco arquivos de
produto que aparecem nas provas foram mutados por `scripts/mutate.ps1` e
restaurados byte a byte por ele (SHA-256 conferido em cada uma das saidas
abaixo).

SHA: `PREENCHER_NO_COMMIT`

`test_orphans.ps1` NAO foi rodado aqui: o orquestrador o roda destacado, porque
ele nao cabe no teto de tempo de chamada desta sessao. Duracao dele: nao medida
por mim.

## RED e GREEN

Esta tarefa nao escreveu produto, entao o RED de TDD (teste antes do codigo)
nao se aplica ao pe da letra. O equivalente aqui, e mais forte, e a prova de
mutacao: cada contrapeso e RED contra o produto com a regra removida, e GREEN
contra o produto intacto. As duas metades estao coladas:

- **RED** — as cinco saidas de `mutate.ps1` da secao seguinte, todas com
  `FAIL` e `EXITCODE=0`, cada uma nomeando a linha do teste que acusou.
- **GREEN** — `verify.ps1` verde (etapa 2, `go test -race`, cobre os cinco
  testes novos) e as rodadas isoladas antes de cada mutacao:

```
go test -race -run 'TestNoteReadAceitaLoteNoTeto|TestNoteReadRecusaLoteAcimaDoTeto' -count=1 ./internal/mcpsrv/
ok  	github.com/jonyd/gobsidian/internal/mcpsrv	1.425s

go test -race -run TestServePonteRemotaEncaminhaHostParaDaemon -count=100 ./cmd/gobsidian/
ok  	github.com/jonyd/gobsidian/cmd/gobsidian	4.891s

go test -race -run 'TestTravaDoKernelEntreProcessos|TestEscutarComLockAbreOSocketEUmSoOuvinte' -count=1 -v ./internal/daemon/
--- PASS: TestTravaDoKernelEntreProcessos (0.08s)
--- PASS: TestEscutarComLockAbreOSocketEUmSoOuvinte (0.01s)
ok  	github.com/jonyd/gobsidian/internal/daemon	2.096s

go test -race -run TestAliasCollisions -count=1 -v ./internal/index/
--- PASS: TestAliasCollisions (0.03s)
--- PASS: TestAliasCollisionsZeroSemDuplicata (0.02s)
ok  	github.com/jonyd/gobsidian/internal/index	1.760s

go test -race -run TestDialAndHandshakeReadOnlyCombinando -count=1 -v ./internal/ipc/
--- PASS: TestDialAndHandshakeReadOnlyCombinando (0.01s)
ok  	github.com/jonyd/gobsidian/internal/ipc	1.515s
```

O `-count=100` da ponte nao e enfeite: ver a observacao sobre a corrida latente
no fim deste relatorio.

## Arquivos

Criados:

- `internal/mcpsrv/export_test.go`
- `internal/daemon/trava_kernel_windows_test.go`
- `internal/daemon/trava_kernel_other_test.go`

Modificados:

- `internal/mcpsrv/tools_read_test.go` (`TestNoteReadAceitaLoteNoTeto`)
- `cmd/gobsidian/ponte_test.go` (`TestServePonteRemotaEncaminhaHostParaDaemon`)
- `internal/daemon/lock_escuta_test.go` (`TestEscutarComLockAbreOSocketEUmSoOuvinte`)
- `internal/index/alias_test.go` (`TestAliasCollisions`, `TestAliasCollisionsZeroSemDuplicata`)
- `internal/ipc/ipc_test.go` (`TestDialAndHandshakeReadOnlyCombinando`)

## As cinco provas de mutacao, como rodaram

### Step 1 — o teto de 50 caminhos, lado ACEITO

```
pwsh -File scripts/mutate.ps1 -Path internal/mcpsrv/tools_read.go `
  -Anchor "len(req.Paths) > maxPathsPorLote" `
  -Replacement "len(req.Paths) >= maxPathsPorLote" `
  -Test "TestNoteReadAceitaLoteNoTeto|TestNoteReadRecusaLoteAcimaDoTeto" `
  -Package ./internal/mcpsrv/
```

```
[...] Mutando internal/mcpsrv/tools_read.go
      - len(req.Paths) > maxPathsPorLote
      + len(req.Paths) >= maxPathsPorLote

[...] go test -race -run TestNoteReadAceitaLoteNoTeto|TestNoteReadRecusaLoteAcimaDoTeto ./internal/mcpsrv/
----------------------------------------------------------------------
--- FAIL: TestNoteReadAceitaLoteNoTeto (0.15s)
    tools_read_test.go:410: 50 caminhos e o teto, nao acima dele: INVALID_ARGUMENT: paths tem 50 itens; o maximo por chamada e 50
FAIL
FAIL	github.com/jonyd/gobsidian/internal/mcpsrv	1.425s
FAIL
----------------------------------------------------------------------
[OK] internal/mcpsrv/tools_read.go restaurado byte a byte (SHA-256 confere).

[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
EXITCODE=0
```

O teste da recusa (51 caminhos) continua verde sob esta mutacao — e por isso
ele sozinho nunca prendeu o valor do teto.

### Step 2a — a direcao host -> daemon

```
pwsh -File scripts/mutate.ps1 -Path cmd/gobsidian/ponte.go `
  -Anchor "io.Copy(conn, teed)" -Replacement "io.Copy(io.Discard, teed)" `
  -Test TestServePonteRemotaEncaminhaHostParaDaemon -Package ./cmd/gobsidian/
```

```
[...] Mutando cmd/gobsidian/ponte.go
      - io.Copy(conn, teed)
      + io.Copy(io.Discard, teed)

[...] go test -race -run TestServePonteRemotaEncaminhaHostParaDaemon ./cmd/gobsidian/
----------------------------------------------------------------------
--- FAIL: TestServePonteRemotaEncaminhaHostParaDaemon (5.02s)
    ponte_test.go:313: o daemon nao recebeu nada em 5s -- a direcao host->daemon nao esta sendo copiada
FAIL
FAIL	github.com/jonyd/gobsidian/cmd/gobsidian	6.420s
FAIL
----------------------------------------------------------------------
[OK] cmd/gobsidian/ponte.go restaurado byte a byte (SHA-256 confere).

[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
EXITCODE=0
```

### Step 2b — o meio-fechamento (achado M8)

```
pwsh -File scripts/mutate.ps1 -Path cmd/gobsidian/ponte.go `
  -Anchor "if !fimDoHost {" -Replacement "if fimDoHost || true {" `
  -Test TestServePonteRemotaEncaminhaHostParaDaemon -Package ./cmd/gobsidian/
```

```
[...] Mutando cmd/gobsidian/ponte.go
      - if !fimDoHost {
      + if fimDoHost || true {

[...] go test -race -run TestServePonteRemotaEncaminhaHostParaDaemon ./cmd/gobsidian/
----------------------------------------------------------------------
--- FAIL: TestServePonteRemotaEncaminhaHostParaDaemon (0.02s)
    ponte_test.go:334: o daemon nao conseguiu responder depois do EOF: io: read/write on closed pipe -- a ponte fechou a conexao INTEIRA em vez de so a direcao de escrita (achado M8)
FAIL
FAIL	github.com/jonyd/gobsidian/cmd/gobsidian	1.499s
FAIL
----------------------------------------------------------------------
[OK] cmd/gobsidian/ponte.go restaurado byte a byte (SHA-256 confere).

[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
EXITCODE=0
```

`if fimDoHost || true` e nao `if true`: com `if true` a variavel `fimDoHost`
deixa de ser lida em qualquer lugar e o pacote nao compila
("declared and not used"), e mutacao que quebra o build e INCONCLUSIVA, nao
prova. A forma acima mantem a leitura e desliga so a regra.

**O confundidor que este teste teve de apagar:** afirmar apenas que o daemon ve
EOF nao distingue meio-fechamento de fechamento inteiro — sem o `half-close`, o
passo `close-conn` fecha a conexao e o daemon ve EOF do mesmo jeito. O que
separa os dois e a resposta em voo DEPOIS do EOF: com o meio-fechamento ela
chega ao stdout do host; sem ele a escrita do daemon morre em pipe fechado, que
e a linha de falha acima.

### Step 3 — o offset 1<<62 da trava do Windows

Ancora diferente da do brief: o codigo escreve `const deslocamento = uint64(1) << 62`,
entao `"1 << 62"` nao casa (ha um `)` no meio) e `mutate.ps1` sairia 2.

```
pwsh -File scripts/mutate.ps1 -Path internal/daemon/trava_windows.go `
  -Anchor "uint64(1) << 62" -Replacement "uint64(1) << 61" `
  -Test TestTravaDoKernelEntreProcessos -Package ./internal/daemon/
```

```
[...] Mutando internal/daemon/trava_windows.go
      - uint64(1) << 62
      + uint64(1) << 61

[...] go test -race -run TestTravaDoKernelEntreProcessos ./internal/daemon/
----------------------------------------------------------------------
--- FAIL: TestTravaDoKernelEntreProcessos (0.07s)
    trava_kernel_windows_test.go:81: consegui travar o offset 0x4000000000000000 com o dono vivo: o produto nao esta travando essa faixa. Se o offset mudou, confira que a nova faixa nao alcanca o byte 0 — internal/doctor le o PID desse arquivo e uma faixa que cubra o byte 0 quebra essa leitura
FAIL
FAIL	github.com/jonyd/gobsidian/internal/daemon	1.101s
FAIL
----------------------------------------------------------------------
[OK] internal/daemon/trava_windows.go restaurado byte a byte (SHA-256 confere).

[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
EXITCODE=0
```

O teste so consegue reprovar porque escreve o offset como LITERAL. Se chamasse
`faixaDaTrava()`, produto e teste andariam juntos sob a mutacao e ela
sobreviveria — o comentario no topo do arquivo registra isso, porque a
repeticao parece violar "uma conta por regra" e nao viola: aqui o teste E o
contrato, e contrato se escreve de fora.

### Step 4 — `AliasCollisions`

```
pwsh -File scripts/mutate.ps1 -Path internal/index/query.go `
  -Anchor "if len(paths) > 1 {" -Replacement "if len(paths) > 2 {" `
  -Test TestAliasCollisions -Package ./internal/index/
```

```
[...] Mutando internal/index/query.go
      - if len(paths) > 1 {
      + if len(paths) > 2 {

[...] go test -race -run TestAliasCollisions ./internal/index/
----------------------------------------------------------------------
--- FAIL: TestAliasCollisions (0.02s)
    alias_test.go:46: AliasCollisions() = 1, quer 2 (STJ em 2 notas, TRF em 3, STF em 1). 5 seria contar notas em colisao; 3 seria contar aliases declarados
FAIL
FAIL	github.com/jonyd/gobsidian/internal/index	0.744s
FAIL
----------------------------------------------------------------------
[OK] internal/index/query.go restaurado byte a byte (SHA-256 confere).

[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
EXITCODE=0
```

Semantica confirmada lendo `internal/index/query.go:158`: conta ALIASES
declarados por mais de uma nota, nao notas em colisao. O cofre do teste separa
as tres contagens possiveis (2 / 5 / 3) para que o numero nao possa ser obtido
por engano.

### Step 5 — handshake com `ReadOnly=true` aceito

```
pwsh -File scripts/mutate.ps1 -Path internal/ipc/ipc.go `
  -Anchor "want := HandshakeConfig{ReadOnly: readOnly," `
  -Replacement "want := HandshakeConfig{ReadOnly: false," `
  -Test TestDialAndHandshakeReadOnlyCombinando -Package ./internal/ipc/
```

```
[...] Mutando internal/ipc/ipc.go
      - want := HandshakeConfig{ReadOnly: readOnly,
      + want := HandshakeConfig{ReadOnly: false,

[...] go test -race -run TestDialAndHandshakeReadOnlyCombinando ./internal/ipc/
----------------------------------------------------------------------
--- FAIL: TestDialAndHandshakeReadOnlyCombinando (0.01s)
    ipc_test.go:277: DialAndHandshake(readOnly=true) contra um daemon ro=1 error = configuracao do daemon diverge da ponte: ponte quer {ReadOnly:false VaultKey:4d04a282c27c5716 MaxResults:0}, daemon oferece {ReadOnly:true VaultKey:4d04a282c27c5716 MaxResults:0}, esperado nil -- duas pontas que pedem a MESMA configuracao tem de se falar; se so a recusa funciona, o modo somente-leitura nunca usa daemon nenhum
FAIL
FAIL	github.com/jonyd/gobsidian/internal/ipc	0.460s
FAIL
----------------------------------------------------------------------
[OK] internal/ipc/ipc.go restaurado byte a byte (SHA-256 confere).

[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
EXITCODE=0
```

`TestDialAndHandshakeConfigDivergente` (ipc_test.go:188) continua verde sob esta
mutacao: um daemon que oferece `ro=1` diverge de um `want` falso do mesmo jeito.
So o lado ACEITO pega. E a mesma armadilha do handshake de `max_results`
registrada em `docs/papeis/testador.md`.

## Cobertura das tres funcoes que estavam a 0 %

Antes (medido pelo orquestrador em `8568b5b`):

```
EscutarComLock     0.0%
TravaEmUso         0.0%
AliasCollisions    0.0%
```

Depois — `go test -count=1 -coverprofile=... ./internal/daemon/ ./internal/index/ ./internal/ipc/`
seguido de `go tool cover -func`:

```
github.com/jonyd/gobsidian/internal/daemon/lock.go:223:		EscutarComLock			100.0%
github.com/jonyd/gobsidian/internal/daemon/trava.go:95:		TravaEmUso			85.7%
github.com/jonyd/gobsidian/internal/index/query.go:158:		AliasCollisions			100.0%
```

`TravaEmUso` fica em 85,7 % e nao 100 %: o ramo que sobra e o `return false, err`
de `tentarTravar` falhando (diretorio de runtime inacessivel), que exigiria uma
condicao de ambiente fora do escopo desta tarefa.

Totais dos tres pacotes na mesma rodada: `daemon 81.7%`, `index 83.0%`,
`ipc 76.7%`.

## Gate

```
[OK] Bateria completa. Pode commitar.
```

As 14 etapas passaram. A contagem de pulados ficou em 6, todos pre-existentes —
`TestAjudanteSeguraTrava` esta na lista porque e o corpo do processo auxiliar,
nao um teste, e pula quando a env var nao esta setada (comportamento de sempre).
`trava_kernel_other_test.go` nao pula: ele nem compila no Windows (`//go:build !windows`).

## Desvios do brief, e por que

1. **`internal/mcpsrv/export_test.go` criado** (arquivo nao previsto). O
   orquestrador pediu que o subteste construisse `maxPathsPorLote` caminhos e
   nao um literal 50; `tools_read_test.go` e `package mcpsrv_test` e nao
   enxerga a constante nao exportada. `export_test.go` (`package mcpsrv`, so
   `_test.go`) e o idioma da stdlib para isso e nao existe no binario.
2. **Teste top-level em vez de `t.Run` dentro do teste da recusa.** O brief
   escreveu o contrapeso como subteste; como funcao propria ele pode ser
   nomeado sozinho no `-run` da prova de mutacao, e a prova roda o PAR
   (`TestNoteReadAceitaLoteNoTeto|TestNoteReadRecusaLoteAcimaDoTeto`) para
   mostrar que so um dos dois reprova.
3. **Ancora do Step 3 diferente da do brief** (`uint64(1) << 62` em vez de
   `1 << 62`) — ver a secao do Step 3.
4. **Step 2 provado com `mutate.ps1`, nao "a mao".** O brief pedia duas
   mutacoes manuais; o script faz a mesma mutacao, imprime a saida real e
   restaura conferindo SHA-256 — as tres coisas que o relatorio precisa. Nao ha
   perda de fidelidade: as duas mutacoes sao as que o brief nomeia.
5. **O Step 3 usa o processo-ajudante que ja existia** (`varAjudante`,
   `TestAjudanteSeguraTrava`, `lancarAjudante` em `trava_test.go`) em vez do
   ajudante novo com codigos de saida 3/4/5 que o brief desenhava — instrucao
   explicita do orquestrador, e evita um segundo padrao de processo-ajudante no
   mesmo pacote. Por isso os arquivos novos sao `package daemon` (interno), que
   e o pacote de `trava_test.go`.
6. **`ComLockDeEscuta` nao ganhou teste novo**: `lock_escuta_test.go` ja o cobre
   pelos dois lados (exclusao sob concorrencia e liberacao apos a chamada). O
   que faltava era `EscutarComLock`, que e o que este relatorio acrescenta.
7. **`test_orphans.ps1` nao rodado** — instrucao do orquestrador.

## Observacao — corrida latente no produto, NAO corrigida (fora de escopo)

Escrevendo o teste do Step 2 apareceu o seguinte, em `cmd/gobsidian/ponte.go:193-202`:
no EOF do stdin do host, DOIS caminhos ficam prontos quase ao mesmo tempo —
`hostParaDaemon` (que marca `fimDoHost = true` e habilita o meio-fechamento) e
`ctx.Done()`, porque `mirrorReader` fecha o espelho e o `watchStdin` do
lifecycle cancela o context pelo mesmo EOF. Se o `select` acordar pelo
`ctx.Done()`, `fimDoHost` fica falso e o meio-fechamento do M8 e PULADO; a
resposta em voo se perde exatamente como antes da correcao.

Medido, nao suposto: `go test -race -run TestServePonteRemotaEncaminhaHostParaDaemon -count=100 ./cmd/gobsidian/`
saiu `ok ... 4.891s`, 100/100 verdes — o caminho curto (retorno do `io.Copy` e
envio num canal com buffer) vence o caminho longo (desparque da goroutine do
`watchStdin`, mutex, formatacao do `slog`, `cancel`) de forma consistente nesta
maquina. Entao o teste novo NAO e flaky aqui, e a corrida e latente: uma
maquina mais lenta no lado do `io.Copy`, ou um logger mais barato no lado do
lifecycle, poderia inverte-la.

Nao alterei o produto (a tarefa e so de teste). Se o dono quiser fechar isso, a
forma menor e drenar `hostParaDaemon` de forma nao-bloqueante dentro do ramo
`ctx.Done()` antes de decidir `fimDoHost`.
