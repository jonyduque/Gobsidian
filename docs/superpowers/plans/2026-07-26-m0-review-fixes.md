# gobsidian — Correções da revisão tardia de M0

Plano de remediação para os achados da revisão das Tasks 9, 10 e 11 do plano
`docs/superpowers/plans/2026-07-25-gobsidian-v01.md`. Aquelas três tasks
fecharam sem revisão fresca (Task 9 e Task 10 no último fix pass, Task 11
inteira) e o ledger registrou a dívida; esta é a revisão que faltava,
convertida em trabalho.

Nada aqui é funcionalidade nova. Cada task fecha um achado nomeado, e a
numeração `R<n>` existe para não colidir com as Tasks 1–26 do plano da v0.1.

## Global Constraints

Valem as mesmas do plano da v0.1, e estas três governam este plano em
particular:

1. **stdout pertence ao JSON-RPC.** Todo log vai para stderr via `log/slog`.
   `doctor` e `version` imprimem em stdout de propósito — são comandos de CLI.
2. **Saída de console em ASCII puro:** `[OK]`, `[*]`, `[!]`, `[i]`, `[...]`.
3. **Nenhum teste pode passar sem poder falhar.** Todo teste adicionado aqui
   trava um comportamento que hoje está destravado; o implementador prova isso
   mutando o código sob teste e mostrando o teste reprovando. Um teste que não
   reprova sob mutação é pior que teste nenhum, porque reporta cobertura que
   não existe.

Comandos de verificação (a versão sem `-race` não conta):

```bash
go test -race ./...
go vet ./...
gofmt -l .
GOOS=linux go vet ./... && GOOS=darwin go vet ./...
pwsh -File scripts/check_net.ps1
pwsh -File scripts/build.ps1
pwsh -File scripts/test_orphans.ps1 -Cycles 10
```

**Nunca rode `go mod tidy`.** Deps fixadas sem importador ainda seriam removidas,
junto com o pin do SDK MCP (PRD D6).

## Achados que este plano NÃO fecha

Registrados para a revisão final não os redescobrir como novos:

- **T11-2 — o fix do `exited` só tem cobertura unitária.** No harness atual
  `stdin-eof` vence todo ciclo, então o ramo `b.exited` de `sameProcess` nunca
  é exercitado ponta a ponta. Fechar isso exige um cenário em que o stdin fica
  aberto e o pai morre, que é escopo de M6. A lacuna já está no ledger.
- **T10-3 — `longPathThreshold` duplicado entre `internal/doctor` e
  `internal/vault`.** Deliberado e documentado no código: os dois medem
  caminhos diferentes (com e sem o prefixo `\\?\`). Fica como está.

---

### Task R1: `doctor` — falha de varredura não pode virar aviso

**Achados:** T10-1 (Important), T10-2 (Minor).

**Files:**
- Modify: `internal/doctor/checks.go`
- Modify: `internal/doctor/checks_windows.go`
- Modify: `internal/doctor/doctor_extra_test.go`

**Interfaces:**
- Consumes: `vault.Walk`, que já distingue os dois desfechos na origem —
  cancelamento devolve `ctx.Err()` cru, falha na raiz devolve
  `fmt.Errorf("varrendo a raiz do cofre %q: %w", ...)`
- Produces: `scanStatus(scan vaultScan, name string) (Result, bool)`, o helper
  único que os seis consumidores de `vaultScan` usam para reportar erro de
  varredura

**O defeito:** `scanVault` colapsa dois desfechos distintos em `scan.err`, e
todos os seis consumidores (`checkNoteCount`, `checkLongestPath`,
`checkLongPathsEnabled`, `checkCloudOnlyFiles`, `checkCasingCollisions`) mapeiam
qualquer valor não-nulo para `StatusWarn`. Cancelamento de contexto e cofre
inacessível produzem a mesma resposta, e `ExitCode` devolve 0 nos dois.

Um cofre inacessível — unidade removível desconectada, share de rede caído,
pasta que o cliente de nuvem moveu — faz `doctor` imprimir `[*]` e sair 0.
`doctor` é o comando que alguém roda quando o produto já não funciona, e o
exit code é o que um script de setup gateia. Cofre inacessível e cofre vazio
não podem produzir a mesma resposta; é a mesma regra que `vault.Walk` já
respeita em `d == nil`.

- [ ] **Step 1: Adicionar o helper de classificação em `checks.go`**

Logo abaixo da declaração de `vaultScan`:

```go
// scanStatus traduz o erro de uma varredura interrompida em Result, e é o
// único ponto onde essa tradução acontece — os seis checks que consomem
// vaultScan chamam este helper em vez de montar o Result cada um.
//
// A distinção é o ponto todo. Cancelamento veio de quem chamou: o usuário
// deu Ctrl-C, ou o comando de cima desistiu. Não é problema do ambiente e
// não deve virar código de saída não-zero. Qualquer outro erro veio do
// cofre — vault.Walk só o produz quando falha na própria raiz, que significa
// unidade desconectada, share caído ou pasta movida pelo cliente de nuvem.
// Reportar isso como aviso faz doctor sair 0 sobre um cofre que o servidor
// não conseguirá abrir, que é exatamente a confusão que este comando existe
// para desfazer.
//
// O bool devolvido é "houve erro": false significa que o chamador segue com
// os dados do scan.
func scanStatus(scan vaultScan, name string) (Result, bool) {
	if scan.err == nil {
		return Result{}, false
	}
	if errors.Is(scan.err, context.Canceled) || errors.Is(scan.err, context.DeadlineExceeded) {
		return Result{
			Name:   name,
			Status: StatusWarn,
			Detail: fmt.Sprintf("varredura interrompida: %v", scan.err),
		}, true
	}
	return Result{
		Name:   name,
		Status: StatusFail,
		Detail: fmt.Sprintf("cofre inacessivel durante a varredura: %v", scan.err),
	}, true
}
```

`checks.go` passa a importar `context` (já importa `errors` e `fmt`).

- [ ] **Step 2: Trocar os cinco consumidores para o helper**

Em `checks.go`, `checkNoteCount` e `checkLongestPath`; em
`checks_windows.go`, `checkLongPathsEnabled`, `checkCloudOnlyFiles` e
`checkCasingCollisions`. Cada um substitui o bloco

```go
	if scan.err != nil {
		return Result{Name: name, Status: StatusWarn, Detail: fmt.Sprintf("varredura interrompida: %v", scan.err)}
	}
```

por

```go
	if res, failed := scanStatus(scan, name); failed {
		return res
	}
```

`checks_windows.go` pode ficar sem uso de `fmt` depois disso — confira e
remova o import se for o caso; `gofmt -l .` e `go vet ./...` pegam o resto.

- [ ] **Step 3: `checkCacheDir` passa por `vault.LongPath`**

`checks.go`, dentro de `checkCacheDir`: `os.MkdirAll(cfg.CacheDir, 0o755)`
vira `os.MkdirAll(vault.LongPath(cfg.CacheDir), 0o755)`. É o único ponto do
pacote que toca o sistema de arquivos sem passar por `LongPath`, e
`--cache-dir` aceita caminho arbitrário. O `Detail` do resultado continua
imprimindo `cfg.CacheDir`, não a forma com prefixo: quem lê o relatório
precisa do caminho que ele reconhece.

- [ ] **Step 4: Corrigir o teste que trava o comportamento errado**

`TestRunContextCancelledStopsWalk` afirma `ExitCode(results) != 0` como
falha, e isso continua correto — cancelamento não é falha bloqueante. O teste
fica, com o comentário estendido para dizer **por que** cancelamento é o caso
que não falha, em contraste com o novo teste abaixo.

- [ ] **Step 5: Teste novo — cofre que some durante a varredura**

Em `doctor_extra_test.go`. O cofre precisa passar por `checkRootExists` e
`checkReadable` e falhar depois, dentro de `vault.Walk`. Um diretório que é
removido entre as duas fases é a forma direta:

```go
// TestRunFailsWhenVaultVanishesMidRun confirma que um cofre que fica
// inacessivel produz falha bloqueante, nao aviso. Cofre inacessivel e cofre
// vazio nao podem produzir a mesma resposta — e o exit code e a unica parte
// do relatorio que um script de setup consegue ler.
func TestRunFailsWhenVaultVanishesMidRun(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "cofre")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatalf("Mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "A.md"), []byte("# A\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cfg := config.Defaults()
	cfg.VaultPath = root

	// Remove a raiz depois que os checks bloqueantes ja a viram. Run chama
	// scanVault depois deles, e vault.Walk falha em d == nil.
	if err := os.RemoveAll(root); err != nil {
		t.Fatalf("RemoveAll: %v", err)
	}

	results := doctor.Run(context.Background(), cfg)

	if doctor.ExitCode(results) == 0 {
		t.Errorf("cofre inacessivel deveria produzir exit code nao-zero: %+v", results)
	}
}
```

Este teste, como escrito, remove a raiz **antes** de `Run` começar, então
`checkRootExists` já falha e o halt dispara — o exit code fica não-zero pelo
caminho antigo, e o teste passa sem exercitar `scanStatus`. Isso não serve.
Torne o cenário fiel: o implementador deve fazer a raiz sobreviver aos dois
checks bloqueantes e falhar só na varredura. Duas formas aceitáveis:

1. **Teste de unidade sobre `scanStatus`** (preferido, determinístico):
   chame `scanStatus` diretamente com `vaultScan{err: errors.New("varrendo a
   raiz do cofre: acesso negado")}` e com `vaultScan{err: context.Canceled}`,
   e afirme `StatusFail` no primeiro e `StatusWarn` no segundo. Exige que
   `scanStatus` seja testável do pacote — teste interno (`package doctor`) em
   arquivo próprio, já que os testes atuais são externos (`package doctor_test`).
2. **Teste de integração** com uma raiz que existe e é listável mas cuja
   varredura falha. Se você encontrar uma forma robusta e sem privilégio no
   Windows, use-a; caso contrário, fique com (1) e não invente um cenário
   frágil.

Escolha uma, implemente-a de verdade, e **prove que ela pode falhar**:
reverta `scanStatus` para devolver `StatusWarn` sempre, mostre o teste
reprovando, e restaure. O relatório precisa conter o comando e a saída dos
dois estados.

- [ ] **Step 6: Verificar**

```bash
go test -race ./internal/doctor/... && go vet ./... && gofmt -l .
GOOS=linux go vet ./... && GOOS=darwin go vet ./...
```

- [ ] **Step 7: Commit**

```
fix(doctor): fail on an inaccessible vault instead of warning

scanVault collapsed context cancellation and a genuinely unreachable vault
into one error value, and all six consumers mapped it to StatusWarn -- so a
disconnected drive or a dropped network share printed [*] and exited 0. The
exit code is the only part of the report a setup script can read.
```

---

### Task R2: harness de órfãos — gatear no mecanismo, não só na ausência de órfão

**Achados:** T11-1 (Important), T11-3 (Minor), T11-4 (Minor).

**Files:**
- Modify: `scripts/test_orphans.ps1`

**Interfaces:**
- Consumes: os logs de debug por ciclo, onde o lifecycle registra `reason=<mecanismo>`
- Produces: exit 1 quando qualquer ciclo não registrou um `reason=`, mesmo com zero órfãos

**O defeito:** hoje

```powershell
$Failed = ($Survivors -gt 0) -or ($LaunchFailures -gt 0) -or ($KillFailures -gt 0)
```

A ausência de `reason=` em todos os logs produz só um `Write-Warning`, que não
mexe no exit code. Cenário de falha concreto: uma regressão faz o servidor sair
sozinho pouco depois do startup — parse de config quebrado, panic depois do PID
ter sido escrito. A checagem de PID em ~0 ms passa, o host é morto, os 2 s de
espera passam, o filho já morreu por conta própria. Zero órfãos, exit 0,
`[OK] Nenhum orfao em 100 ciclos` — e nenhum mecanismo de encerramento
disparou. É a mesma classe do defeito de aspas que o antecessor já encontrou
neste script: verde provando nada.

- [ ] **Step 1: Contar `reason=` por ciclo, não só no agregado**

O bloco de amostragem atual varre todos os logs de uma vez e agrupa. Troque-o
por uma contagem que sabe **quantos ciclos** registraram um motivo, mantendo
o agrupamento para o diagnóstico humano:

```powershell
# Cada ciclo tem seu proprio log. Um ciclo sem "reason=" e um ciclo em que
# nenhum mecanismo de encerramento disparou — o filho morreu por outro
# motivo, ou nao chegou a viver. Contar o agregado nao basta: 100 motivos
# vindos de 50 ciclos passaria despercebido.
$CyclesWithReason = 0
$Reasons = @()

foreach ($Log in @(Get-ChildItem -Path $WorkDir -Filter "cycle_*.log" -ErrorAction SilentlyContinue)) {
    # @(...) e obrigatorio: sem correspondencia o pipeline devolve $null, e
    # sob Set-StrictMode ler .Count em $null e erro fatal — o proprio
    # diagnostico da falha mascararia a falha original.
    $Found = @(Select-String -Path $Log.FullName -Pattern 'reason=(\S+)' -ErrorAction SilentlyContinue |
        ForEach-Object { $_.Matches[0].Groups[1].Value })
    if ($Found.Count -gt 0) {
        $CyclesWithReason++
        $Reasons += $Found
    }
}

$MeasuredCycles = $Cycles - $LaunchFailures
$ReasonlessCycles = $MeasuredCycles - $CyclesWithReason

if ($Reasons.Count -gt 0) {
    Write-Output "[i] motivos observados nos logs de debug:"
    $Reasons | Group-Object | Sort-Object Count -Descending | ForEach-Object {
        Write-Output ("    {0}: {1}x" -f $_.Name, $_.Count)
    }
}
```

`$MeasuredCycles` desconta os ciclos que já foram contados como
`LaunchFailures`: um ciclo que nunca lançou nada não tem motivo para
registrar, e cobrá-lo duas vezes só embaralha o diagnóstico.

- [ ] **Step 2: Fazer a ausência de motivo reprovar**

```powershell
if ($ReasonlessCycles -gt 0) {
    Write-Warning "[!] FALHA: $ReasonlessCycles de $MeasuredCycles ciclo(s) encerraram sem registrar 'reason=' - nenhum mecanismo de encerramento disparou, e zero orfaos nao prova nada nesse caso"
}

$Failed = ($Survivors -gt 0) -or ($LaunchFailures -gt 0) -or ($KillFailures -gt 0) -or ($ReasonlessCycles -gt 0)
```

O bloco de `Write-Warning` por categoria e o `exit 1` final já existem;
`$ReasonlessCycles` entra na mesma forma que os outros três contadores.

- [ ] **Step 3: `$SettleMs` acima do orçamento de encerramento**

`param()`: `[int]$SettleMs = 2000` vira `[int]$SettleMs = 8000`.

`lifecycle.Shutdown` roda com guarda-chuva de 6 s (3 s para a etapa in-flight,
500 ms para o close-pipe, e o `hardLimit` de 6 s cobrindo tudo). Esperar 2 s
conta um encerramento correto porém lento como órfão. É falso-positivo, não
falso-negativo — não esconde defeito — mas queima uma sessão de depuração, e o
gate precisa medir o que o código promete. Comente o vínculo:

```powershell
    # 8s > o guarda-chuva de 6s de lifecycle.Shutdown. Esperar menos que o
    # orcamento que o proprio codigo se da conta encerramento lento como
    # orfao. Se este numero mudar, o de la mudou primeiro.
    [int]$SettleMs = 8000
```

- [ ] **Step 4: Restringir a varredura final aos filhos deste script**

Hoje:

```powershell
$Remaining = @(Get-Process -Name "gobsidian" -ErrorAction SilentlyContinue)
```

Isso mata todo `gobsidian.exe` da máquina e conta cada um como sobrevivente —
inclusive o `gobsidian serve` que o desenvolvedor deixou rodando noutro
terminal, que vira uma rodada vermelha e um servidor perdido.

Acumule os PIDs que este script mediu e restrinja a varredura a eles:

```powershell
$LaunchedPids = @()
```

no topo, `$LaunchedPids += $ServerPid` logo depois de o PID ser validado, e no
fim:

```powershell
# Restrito aos PIDs que ESTE script lancou. Varrer por nome mataria um
# "gobsidian serve" que o desenvolvedor deixou aberto noutro terminal e o
# contaria como orfao — rodada vermelha por um processo que ninguem pediu
# para medir.
$Remaining = @($LaunchedPids | ForEach-Object {
    Get-Process -Id $_ -ErrorAction SilentlyContinue
} | Where-Object { $_ })

if ($Remaining.Count -gt 0) {
    Write-Warning "[!] $($Remaining.Count) processo(s) gobsidian lancado(s) por este script ainda em execucao"
    $Remaining | ForEach-Object { Stop-Process -Id $_.Id -Force -ErrorAction SilentlyContinue }
    $Survivors += $Remaining.Count
}
```

O comentário que justifica a varredura final (um host morto entre o poll do
PID e o `taskkill`) continua válido e deve ser preservado.

- [ ] **Step 5: Provar que o gate novo pode reprovar**

Um gate que não reprova é exatamente o defeito que esta task fecha, então a
prova não é opcional. Rode as duas metades:

```powershell
pwsh -File scripts/build.ps1
pwsh -File scripts/test_orphans.ps1 -Cycles 10
```

Esperado: `[OK] Nenhum orfao em 10 ciclos`, e a linha de motivos mostrando
`stdin-eof: 10x`.

Depois, mutação: mude o padrão de busca de `'reason=(\S+)'` para algo que não
existe nos logs (`'motivo_inexistente=(\S+)'`), rode de novo com `-Cycles 3`, e
mostre a saída reprovando com `FALHA: 3 de 3 ciclo(s) encerraram sem registrar
'reason='` e exit 1. Restaure o padrão. O relatório precisa das duas saídas.

Mutar o padrão em vez do servidor é deliberado: prova que o gate observa o
sinal, sem exigir uma regressão real no lifecycle.

- [ ] **Step 6: Commit**

```
test(lifecycle): fail the orphan gate when no shutdown mechanism fires

The harness printed the observed reason= values but never gated on them, so
a server that died on its own shortly after startup produced zero orphans and
a green run with no shutdown mechanism ever firing -- the same
green-proving-nothing class as the quoting bug this script already had once.
Also widen the settle window past Shutdown's own 6s umbrella and scope the
final sweep to PIDs this script launched.
```

---

### Task R3: `cmd/gobsidian` — travar o comportamento de encerramento com testes

**Achados:** T9-1 (Important), T9-2 (Minor), T9-3 (Minor), T9-4 (Minor).

**Files:**
- Modify: `cmd/gobsidian/serve.go`
- Create: `cmd/gobsidian/serve_test.go`

**Interfaces:**
- Produces: `shutdownExitCode(err error) int`, a decisão de código de saída
  extraída para uma função pura e testável
- Produces: `mirrorReader` continua onde está, agora com testes

**O defeito:** `go test ./...` imprime
`? github.com/jonyd/gobsidian/cmd/gobsidian [no test files]`. Os dois
comportamentos que o último fix pass da Task 9 introduziu vivem só aqui:

- `mirrorReader.broken` — falha de escrita no espelho não pode virar erro da
  leitura principal, senão uma sessão saudável morre por um motivo que o
  cliente não pode agir
- `errors.Is(loopErr, context.Canceled)` → exit 0 — desconexão limpa não é
  falha

Os dois foram fechados por observação manual ("20 sessões limpas exit 0") que
não existe mais em lugar nenhum. Ninguém consegue defender essas duas linhas
numa refatoração futura.

- [ ] **Step 1: Extrair a decisão de exit code**

Em `serve.go`, acima de `runServe`:

```go
// shutdownExitCode traduz o erro final do serve loop em codigo de saida.
//
// Existe como funcao separada porque e a unica parte de runServe que da para
// testar sem levantar um processo: runServe termina em os.Exit por desenho, e
// os.Exit nao volta.
//
// Os tres erros tratados como encerramento normal sao os que aparecem quando
// o host simplesmente vai embora. context.Canceled vem do proprio lifecycle,
// que cancela o contexto quando o stdin fecha, um sinal chega ou o pai morre.
// io.EOF e io.ErrClosedPipe podem vir do SDK, que detecta o fim do stdin por
// conta propria — as duas deteccoes correm, e qual delas vence decide qual
// valor chega aqui. Tratar qualquer uma como falha faz um host supervisor ver
// erro aleatorio a cada desconexao limpa.
func shutdownExitCode(err error) int {
	switch {
	case err == nil:
		return 0
	case errors.Is(err, context.Canceled),
		errors.Is(err, io.EOF),
		errors.Is(err, io.ErrClosedPipe):
		return 0
	default:
		return 1
	}
}
```

O fim de `runServe` passa a ser:

```go
	// runServe nao retorna: termina o processo aqui para que o codigo de saida
	// seja o desta decisao e nao o que cobra derivaria de um error. O return
	// abaixo e inalcancavel e existe so para satisfazer a assinatura que RunE
	// exige.
	os.Exit(shutdownExitCode(loopErr))
	return nil
```

- [ ] **Step 2: Drenar `lateErr` depois de `lc.Wait()`**

Hoje o `select`/`default` que drena `lateErr` roda antes de `lc.Wait()`. Se a
etapa in-flight estourou o orçamento, sua goroutine foi abandonada e pode
escrever no canal depois da drenagem — o erro cai no buffer e ninguém o lê.
Mover a drenagem para depois de `lc.Wait()` dá à goroutine abandonada todo o
tempo de `Wait` para chegar, sem introduzir espera nova.

```go
	lc.Wait()

	// Depois de Wait, nao antes: a etapa in-flight pode ter sido abandonada
	// por estouro de orcamento, e sua goroutine ainda estar a caminho do
	// canal. Wait e o ultimo ponto em que esperar por ela e de graca.
	//
	// A drenagem continua nao-bloqueante. Se mesmo assim nao houver valor, o
	// erro tardio e descartado de proposito: o encerramento ja estourou o
	// orcamento, e travar aqui trocaria um exit code impreciso por um
	// servidor que nao encerra.
	select {
	case err := <-lateErr:
		if loopErr == nil {
			loopErr = err
		}
	default:
	}

	os.Exit(shutdownExitCode(loopErr))
	return nil
```

- [ ] **Step 3: `cmd/gobsidian/serve_test.go`**

Pacote interno (`package main`) — `mirrorReader` e `shutdownExitCode` não são
exportados.

Testes exigidos, todos com nome que diz qual comportamento travam:

1. `TestMirrorReaderCopiesToMirror` — escreve bytes na origem, lê pelo
   `mirrorReader`, confirma que os mesmos bytes chegam do outro lado do pipe.
2. `TestMirrorReaderPropagatesEOF` — a origem chega ao fim; confirma que a
   ponta de leitura do pipe também vê o fim. É o comportamento que
   `io.TeeReader` não tem e que motivou o tipo existir: sem ele o monitor de
   stdin do lifecycle fica inerte.
3. `TestMirrorReaderBrokenMirrorDoesNotPoisonRead` — fecha a ponta de leitura
   do pipe antes de ler, de modo que a escrita no espelho falhe; confirma que
   `Read` devolve os bytes e o erro da **origem**, não o da escrita, e que
   leituras seguintes continuam funcionando. Este é o teste que trava o campo
   `broken`.
4. `TestShutdownExitCode` — tabela: `nil` → 0; `context.Canceled` → 0;
   `fmt.Errorf("...: %w", context.Canceled)` → 0; `io.EOF` → 0;
   `io.ErrClosedPipe` → 0; `errors.New("falha real")` → 1.

O caso do erro embrulhado não é decorativo: é a razão de a função usar
`errors.Is` e não `==`, e o SDK entrega o erro embrulhado.

Para (3), `io.Pipe` com a ponta de leitura fechada faz a escrita devolver
`io.ErrClosedPipe`. Monte a origem com um `io.Reader` que devolve bytes e
depois `io.EOF`.

- [ ] **Step 4: Provar que os testes podem falhar**

Para cada um dos quatro, mute o código sob teste, mostre o teste reprovando,
restaure. Mutações mínimas que provam o ponto:

- (2): remover a chamada `m.dst.CloseWithError(err)`
- (3): trocar `m.broken = true` por `return n, werr`
- (4): trocar `errors.Is` por `==` no caso de `context.Canceled` — só o caso
  embrulhado deve reprovar, e é isso que se quer ver

O relatório precisa do comando e da saída de cada mutação. Um teste que passa
sob a mutação que ele deveria pegar não conta como feito.

- [ ] **Step 5: Verificar**

```bash
go test -race ./... && go vet ./... && gofmt -l .
```

`go test ./...` não pode mais imprimir `[no test files]` para
`cmd/gobsidian`.

- [ ] **Step 6: Commit**

```
test(cmd): lock down mirror semantics and shutdown exit code

cmd/gobsidian had no tests at all, so the two behaviours the Task 9 fix pass
introduced -- a broken mirror must not poison the live stream, and a clean
disconnect must exit 0 -- rested on a manual observation that no longer
exists anywhere. Extract the exit-code decision into a pure function so it is
testable without spawning a process, and widen it to the EOF errors the SDK
can surface when it wins the detection race.
```

---

### Task R4: CI — rodar os checks que o projeto diz que valem

**Achado:** T11-5 (Minor).

**Files:**
- Modify: `.github/workflows/ci.yml`

**O defeito:** o CI roda `go vet` e `go test -race` na matriz e o
golangci-lint só no ubuntu. Três checks que o CLAUDE.md lista como
obrigatórios não rodam em lugar nenhum:

- `gofmt -l .`
- `GOOS=linux go vet ./...` e `GOOS=darwin go vet ./...`
- lint sobre os arquivos com build tag de Windows — o job de lint roda em
  ubuntu, então `parent_windows.go`, `checks_windows.go` e
  `path_windows.go` nunca são analisados

O último importa mais do que parece: `parent_windows.go` é onde vivia o
defeito que deixou 5 de 5 órfãos.

- [ ] **Step 1: `gofmt` e vet cruzado no job `test`**

Rodar `gofmt` na matriz inteira desperdiça três execuções de um check que não
depende de plataforma, e em Windows a lista de arquivos vem com separador
diferente. Ponha os dois num job próprio, em ubuntu:

```yaml
  fmt:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.25'
      - name: gofmt
        run: |
          fmtout=$(gofmt -l .)
          if [ -n "$fmtout" ]; then
            echo "[!] arquivos fora do formato:"
            echo "$fmtout"
            exit 1
          fi
      # go vet nativo ja roda na matriz; estes dois cobrem os arquivos atras
      # de build tag que a matriz do runner atual nao compila.
      - name: vet cruzado
        run: |
          GOOS=linux go vet ./...
          GOOS=darwin go vet ./...
          GOOS=windows go vet ./...
```

`gofmt -l` sai 0 mesmo listando arquivos, e é por isso que o passo testa a
saída em vez do código de retorno — sem isso o check nunca reprova.

- [ ] **Step 2: Lintar os arquivos com build tag de Windows**

Adicione um segundo job de lint em windows-latest:

```yaml
  lint-windows:
    runs-on: windows-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.25'
      # golangci-lint so analisa os arquivos que a plataforma do runner
      # compila. Sem este job, parent_windows.go, checks_windows.go e
      # path_windows.go nunca sao lintados — e parent_windows.go e onde
      # vivia o defeito que deixou 5 de 5 orfaos.
      - uses: golangci/golangci-lint-action@v6
```

- [ ] **Step 3: Verificar localmente**

```bash
gofmt -l .
GOOS=linux go vet ./... && GOOS=darwin go vet ./... && GOOS=windows go vet ./...
```

As três precisam sair limpas. Não dá para executar o workflow localmente;
confira o YAML com `python -c "import yaml,sys; yaml.safe_load(open('.github/workflows/ci.yml'))"`
ou equivalente, e registre a saída no relatório.

- [ ] **Step 4: Commit**

```
ci: run gofmt, cross-target vet, and lint the Windows-tagged files

CLAUDE.md lists gofmt and GOOS=linux/darwin vet as required checks and CI ran
neither. golangci-lint ran only on ubuntu, so every //go:build windows file --
including parent_windows.go, where the defect that left 5 of 5 orphans lived
-- was never linted.
```

---

## Ordem e independência

R1, R2, R3 e R4 tocam arquivos disjuntos e podem ser executadas em qualquer
ordem. R4 é a única que pode reprovar por causa das outras: se R1 ou R3
deixarem algo fora de formato, o job `fmt` pega. Executar R4 por último dá o
sinal mais útil.

## Auto-revisão do plano

- **R1 Step 5 embute um teste que não serve e diz por quê.** Isso é
  deliberado: a forma óbvia do teste (remover a raiz antes de `Run`) passa
  pelo caminho errado e daria falsa cobertura. Deixar o esboço junto do
  motivo pelo qual ele não presta é mais seguro que omiti-lo e ver o
  implementador reinventá-lo.
- **R3 muda `serve.go` além do que os testes exigem** (extração de
  `shutdownExitCode`, reordenação da drenagem). A extração é o que torna o
  comportamento testável sem levantar processo, então não é escopo extra; a
  reordenação é o achado T9-2 e está nomeada.
- **R2 Step 5 muta o script de teste, não o servidor.** Prova que o gate
  observa o sinal sem exigir uma regressão real no lifecycle. É uma prova mais
  fraca que mutar o servidor, e está registrada como tal.
- **Nenhuma task aqui toca `internal/lifecycle`.** O único achado naquele
  pacote (T11-2) não é fechável sem o cenário de M6.
