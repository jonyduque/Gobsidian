# Encerramento com guarda-chuva, e logs que dizem quem falou — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Nenhuma espera de encerramento fica sem orçamento, e todo log do daemon diz qual processo e qual versão o escreveu — para que o defeito medido em 2026-09-07 (daemon vivo 20 h depois de pedir para encerrar) não possa se repetir em silêncio.

**Architecture:** Um guarda-chuva **único**, armado quando o context raiz é cancelado, cobre `lifecycle.Shutdown` **e** as três esperas que hoje não têm orçamento nenhum (`wg.Wait` dentro de `daemon.Run`, `lc.Wait` e `c.Esperar`). Ele mora em `internal/lifecycle`, ao lado da guarda que já existe, e é armado por `cmd/gobsidian/runDaemon`. Nada de novo pacote. Em paralelo, o logger do daemon ganha `pid` e `versao`, o fallback para o modo em processo sobe de `INFO` para `WARN` com `motivo=`, e `ipc.cleanupSocketFile` ganha plano B e passa a relatar o estado que encontrou.

**Tech Stack:** Go 1.25.0 (piso do `go.mod`, forçado pelo pin do `go-sdk@v1.5.0`), `log/slog`, cobra. Sem dependência nova.

**Spec:** [`docs/superpowers/specs/2026-09-08-instalador-e-encerramento-design.md`](../specs/2026-09-08-instalador-e-encerramento-design.md), commit `4390a1b`. Este plano implementa as seções **5 (Logs)**, **6 (Anteparo)** e a mitigação de §2.2 na tabela de riscos. O instalador (D-01 a D-11) é o **segundo** plano e não é tocado aqui.

Medições que este plano cita e que **nenhuma task pode inventar** (todas de 2026-09-08, máquina do dono, Windows 11 10.0.26220, Go 1.26.5):

- Daemon PID 42856: `encerramento solicitado reason=idle` às 2026-09-07T19:30:36, **nenhuma** linha `daemon encerrado`, processo vivo 20 h depois com 274 MB, **0 s de CPU em 3 s** de amostragem, 26 threads em `Wait,UserRequest`.
- `lifecycle.Shutdown` tem guarda de `os.Exit(1)` em 6 s (`internal/lifecycle/shutdown.go:48`). O processo estar vivo prova que o travamento está **fora** dela.
- `scripts/test_orphans.ps1:6-9` fixa `$SettleMs = 8000` com o comentário *"8s > o guarda-chuva de 6s de lifecycle.Shutdown. (...) Se este numero mudar, o de la mudou primeiro."* — por isso o guarda-chuva deste plano é **6 s**, e não 8.
- `ipc.cleanupSocketFile` falhou 20+ vezes desde 2026-09-01 com `The file cannot be accessed by the system` (1920). Dois cenários foram reproduzidos e **nenhum** produz o par observado (dial 10022 / remove 1920); o comentário de `errnoDe` em `cmd/gobsidian/ponte.go` já registra que o 10022 de campo "nenhum dos reprodutores conhecidos explica".
- Dois processos serviram o cofre `Estudo` ao mesmo tempo (PIDs 7920 e 42628), provado por cada rename aparecer duas vezes no log com milissegundos diferentes (`10:48:20.137` e `10:48:20.152`).

## Global Constraints

- CLAUDE.md inteiro vale. Em especial: **nunca** `git checkout`, `git restore`, `git stash`, `git clean`, `git reset`; **nunca** `go mod tidy`; commit só por caminho explícito (`git add <arquivo>`), nunca `git add -A` — há trabalho do dono não commitado em `test-vault/`, `.claude/skills/`, `Resume-Claude.ps1` e `.superpowers/sdd/2026-07-25-gobsidian-v01/progress.md`.
- **stdout pertence ao JSON-RPC.** Todo log vai para stderr. Nenhuma task deste plano pode acrescentar `fmt.Print*` em código alcançável de `serve`.
- **Só campos novos nos logs.** Nenhuma linha existente muda de texto: `scripts/measure.ps1` faz parsing de `servidor pronto` e da regex `index_ms=(\d+)`, e `scripts/test_orphans.ps1` lê `reason=`. Mudar texto quebra os dois gates.
- **Nenhum tipo do SDK MCP cruza para fora de `internal/mcpsrv`.**
- **Uma conta por regra.** O guarda-chuva é um só, e o orçamento de 6 s tem um só lugar de definição.
- **Código de plataforma atrás de build tag, em arquivo separado.** Nunca `if runtime.GOOS ==`.
- Saída de console em ASCII puro: `[OK]`, `[*]`, `[!]`, `[i]`, `[...]`.
- Sem `helpers.go`, `utils.go`, `common.go`.
- Não escreva número que não mediu. Se não mediu, escreva **"não medido"**.
- **Um teste que não pode falhar é pior que teste ausente.** Antes de dizer que testou: apague a regra, rode, confirme que um teste nomeia a falha, restaure. Prova de mutação por `scripts/mutate.ps1`, com a saída colada no relatório — no passado, não no condicional.
- `pwsh -File scripts/verify.ps1` verde antes de qualquer commit.
- Commits em Conventional Commits, em inglês, via arquivo (`git commit -F <arquivo>`). Trailers obrigatórios: `Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>` e `Claude-Session: https://claude.ai/code/session_01P5wkw6PAdBFzF3uB1w1jNj`.
- Depois de editar qualquer `.md`: `python -c "open('<arquivo>',encoding='utf-8').read()" && echo "[OK] UTF-8 valido"`.

---

## File Structure

| Arquivo | Responsabilidade | Task |
|---|---|---|
| `internal/lifecycle/guarda.go` (novo) | `ArmarGuardaChuva` e `OrcamentoDeEncerramento`: o relógio único que cobre o encerramento inteiro | 189 |
| `internal/lifecycle/guarda_test.go` (novo) | prova que o guarda-chuva dispara, via subprocesso (é `os.Exit`) | 189 |
| `cmd/gobsidian/daemon.go` | arma o guarda-chuva; logger com `pid` e `versao`; rotação do log | 189, 191, 194 |
| `cmd/gobsidian/daemon_log_test.go` | testes do logger | 191, 194 |
| `cmd/gobsidian/ponte.go` | fallback em processo vira `WARN` com `motivo=` | 190 |
| `cmd/gobsidian/ponte_test.go` | prova que os três casos têm motivos distintos | 190 |
| `internal/ipc/ipc.go` | `cleanupSocketFile` com plano B e relato do estado | 192 |
| `internal/ipc/ipc_test.go` | prova do ramo de falha e do plano B | 192 |
| — | o modo e o dono do cache já estão em `doctor` (`daemon.go:107`, `checks.go:212`); a enumeração de processos vai para o plano 2 | 193 |
| `docs/ESTADO.md` | registra a dívida aberta: a causa de §2.1 não foi encontrada | 194 |
| `docs/OPERACAO.md` | descreve o guarda-chuva e os campos novos de log | 194 |

---

### Task 189: guarda-chuva único de encerramento

**Files:**
- Create: `internal/lifecycle/guarda.go`
- Create: `internal/lifecycle/guarda_test.go`
- Modify: `cmd/gobsidian/daemon.go` (dentro de `runDaemon`)

**Interfaces:**
- Consumes: `context.Context`, `*slog.Logger`.
- Produces: `lifecycle.OrcamentoDeEncerramento` (`time.Duration`, 6 s) e `lifecycle.ArmarGuardaChuva(ctx context.Context, log *slog.Logger, orcamento time.Duration) (desarmar func())`. A Task 194 documenta os dois.

- [ ] **Step 1: Escrever o teste que falha.** Ele roda o guarda-chuva num **subprocesso**, porque a ação é `os.Exit(1)` e um teste não pode matar o próprio binário de teste. O padrão de re-execução (`GOBSIDIAN_TESTE_GUARDA=1` + `-test.run`) é o mesmo que o harness de órfãos usa por fora.

```go
package lifecycle

import (
	"context"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// TestGuardaChuvaMataEncerramentoPendurado cobre o defeito medido em
// 2026-09-07: o daemon PID 42856 registrou "encerramento solicitado
// reason=idle" as 19:30:36 e seguiu VIVO 20 h, com 0 s de CPU em 3 s de
// amostragem. lifecycle.Shutdown tem guarda propria, e o processo estar vivo
// prova que o travamento estava FORA dela -- numa das tres esperas que nao
// tinham orcamento nenhum.
//
// Roda em subprocesso porque a acao e os.Exit(1).
func TestGuardaChuvaMataEncerramentoPendurado(t *testing.T) {
	if os.Getenv("GOBSIDIAN_TESTE_GUARDA") == "1" {
		ctx, cancel := context.WithCancel(context.Background())
		log := slog.New(slog.NewTextHandler(os.Stderr, nil))
		_ = ArmarGuardaChuva(ctx, log, 50*time.Millisecond)
		cancel()
		// Espera que NUNCA termina: e exatamente a forma do defeito.
		select {}
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestGuardaChuvaMataEncerramentoPendurado")
	cmd.Env = append(os.Environ(), "GOBSIDIAN_TESTE_GUARDA=1")
	saida, err := cmd.CombinedOutput()

	if err == nil {
		t.Fatal("o subprocesso saiu com sucesso; o guarda-chuva nao disparou")
	}
	if ee, ok := err.(*exec.ExitError); !ok || ee.ExitCode() != 1 {
		t.Fatalf("saida esperada 1, veio %v", err)
	}
	if !strings.Contains(string(saida), "encerramento travou") {
		t.Fatalf("o log nao nomeia a causa; saida:\n%s", saida)
	}
}

// TestGuardaChuvaDesarmadoNaoMata prova o outro lado: um encerramento que
// termina dentro do orcamento nao pode derrubar o processo. Sem este caso, um
// guarda-chuva que matasse SEMPRE passaria no teste acima.
func TestGuardaChuvaDesarmadoNaoMata(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	desarmar := ArmarGuardaChuva(ctx, slog.New(slog.NewTextHandler(io.Discard, nil)), 20*time.Millisecond)
	cancel()
	desarmar()
	time.Sleep(80 * time.Millisecond) // passa do orcamento de proposito
	// Chegar aqui ja e a assercao: o processo nao morreu.
}

// TestGuardaChuvaDesarmarEhIdempotente: runDaemon chama desarmar por defer E
// no caminho feliz; fechar um canal duas vezes entra em panic.
func TestGuardaChuvaDesarmarEhIdempotente(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	desarmar := ArmarGuardaChuva(ctx, slog.New(slog.NewTextHandler(io.Discard, nil)), time.Hour)
	desarmar()
	desarmar()
}
```

- [ ] **Step 2: Rodar e ver falhar.**

Run: `go test ./internal/lifecycle/ -run TestGuardaChuva -v`
Expected: FAIL — `undefined: ArmarGuardaChuva`.

- [ ] **Step 3: Implementar o mínimo.** Criar `internal/lifecycle/guarda.go`:

```go
package lifecycle

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"time"
)

// OrcamentoDeEncerramento e o teto do encerramento INTEIRO -- nao de uma
// etapa.
//
// 6 s, e o numero nao e livre. scripts/test_orphans.ps1:6-9 mede com uma
// janela de $SettleMs = 8000 e o comentario dela fixa a relacao: "8s > o
// guarda-chuva de 6s de lifecycle.Shutdown. (...) Se este numero mudar, o de
// la mudou primeiro." A janela do harness tem de ser MAIOR que o guarda-chuva
// do produto; adotar 8 s aqui igualaria os dois e o processo sairia
// exatamente quando o gate para de esperar.
const OrcamentoDeEncerramento = 6 * time.Second

// ArmarGuardaChuva cobre o encerramento INTEIRO com um relogio so.
//
// Shutdown ja tinha guarda propria, e ela cobria apenas o corpo dela. Medido
// em 2026-09-07: o daemon PID 42856 pediu encerramento por ociosidade as
// 19:30:36 e seguiu vivo 20 h, com 0 s de CPU em 3 s de amostragem e 26
// threads em espera. Como o processo continuou vivo, o travamento estava fora
// de Shutdown -- em wg.Wait (internal/daemon/daemon.go), lc.Wait ou
// c.Esperar, nenhuma das tres com orcamento.
//
// Qual das tres NAO foi determinado: o binario e compilado com -s -w e
// `dlv attach` responde "could not find goroutine array". A decisao do dono
// foi anteparo em vez de caca (D-12 da spec) -- e por isso o guarda-chuva e
// generico: ele cobre o intervalo, nao uma causa.
//
// O relogio so comeca a contar quando ctx e cancelado, porque e ai que o
// encerramento comeca. Antes disso o processo pode viver o quanto quiser.
//
// desarmar e idempotente: runDaemon o chama por defer e tambem no caminho
// feliz, e fechar um canal duas vezes entra em panic.
func ArmarGuardaChuva(ctx context.Context, log *slog.Logger, orcamento time.Duration) (desarmar func()) {
	pronto := make(chan struct{})
	var uma sync.Once

	go func() {
		select {
		case <-pronto:
			return
		case <-ctx.Done():
		}
		select {
		case <-pronto:
		case <-time.After(orcamento):
			// O mesmo texto e a mesma saida da guarda de Shutdown: quem le o
			// log nao precisa aprender duas linguagens para o mesmo evento.
			log.Error("encerramento travou alem do guarda-chuva", "orcamento", orcamento)
			os.Exit(1)
		}
	}()

	return func() { uma.Do(func() { close(pronto) }) }
}
```

- [ ] **Step 4: Rodar e ver passar.**

Run: `go test ./internal/lifecycle/ -run TestGuardaChuva -v`
Expected: PASS nos três.

- [ ] **Step 5: Armar em `runDaemon`.** Em `cmd/gobsidian/daemon.go`, logo depois de `ctx, lc := lifecycle.New(parent, ...)`:

```go
	// O guarda-chuva cobre TUDO daqui para a frente: Shutdown, lc.Wait e
	// c.Esperar. Armado aqui, e nao antes de Shutdown, porque wg.Wait mora
	// DENTRO de d.Run -- um guarda armado depois de Run nunca alcancaria a
	// espera que mais provavelmente pendurou o PID 42856.
	desarmarGuarda := lifecycle.ArmarGuardaChuva(ctx, log, lifecycle.OrcamentoDeEncerramento)
	defer desarmarGuarda()
```

E, depois de `c.Esperar()` e antes de `log.Info("daemon encerrado", ...)`:

```go
	desarmarGuarda()
```

- [ ] **Step 6: Prova de mutação.** Apagar o corpo de `ArmarGuardaChuva` (devolver `func(){}` sem goroutine), rodar `go test ./internal/lifecycle/ -run TestGuardaChuva`, confirmar que `TestGuardaChuvaMataEncerramentoPendurado` **falha nomeando** "o guarda-chuva nao disparou", restaurar. Colar a saída no relatório.

- [ ] **Step 7: Gate e commit.**

```bash
pwsh -File scripts/verify.ps1
git add internal/lifecycle/guarda.go internal/lifecycle/guarda_test.go cmd/gobsidian/daemon.go
git commit -F <arquivo-de-mensagem>
```

---

### Task 190: o fallback para o modo em processo deixa de ser silencioso

**Files:**
- Modify: `cmd/gobsidian/ponte.go:78-107` (os três pontos de queda em `servePonte`)
- Modify: `cmd/gobsidian/ponte_test.go`

**Interfaces:**
- Consumes: nada novo.
- Produces: nada novo. Só nível e campos de log.

**Contexto que a task não pode reinventar:** os três pontos de queda logam hoje em `INFO` e dão linhas diferentes com a mesma gravidade. Um deles ficou 20 h ligado sem ninguém ver (PID 42628, medido 2026-09-08), e `docs/OPERACAO.md` já registra que a mesma classe de silêncio custou um marco inteiro desligado em produção sem ninguém perceber.

- [ ] **Step 1: Escrever o teste que falha.** Acrescentar a `cmd/gobsidian/ponte_test.go`:

```go
// TestFallbackEmProcessoEhWarnComMotivo cobre o silencio medido em
// 2026-09-08: o PID 42628 serviu em modo degradado por 20 h, gravando no
// MESMO cache de busca que o daemon, e a unica pista era uma linha INFO.
//
// Nivel WARN porque INFO e o que o modo normal ja usa: quem filtra por
// gravidade nao tinha como separar "estou no caminho bom" de "cai para o
// caminho ruim". E motivo= porque os tres casos sao defeitos DIFERENTES com
// consertos diferentes.
func TestFallbackEmProcessoEhWarnComMotivo(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	cfg := config.Config{
		VaultPath: t.TempDir(),
		CacheDir:  t.TempDir(),
		LogLevel:  slog.LevelDebug,
	}

	// Sem daemon algum no caminho: EnsureStarted vai falhar e servePonte tem
	// de cair para o modo em processo pelo motivo "daemon-nao-subiu".
	original := iniciarDaemonFn
	iniciarDaemonFn = func(config.Config) error { return errors.New("recusado pelo teste") }
	defer func() { iniciarDaemonFn = original }()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = servePonte(ctx, cfg, log)

	saida := buf.String()
	if !strings.Contains(saida, "level=WARN") {
		t.Errorf("a queda para o modo em processo saiu sem WARN:\n%s", saida)
	}
	if !strings.Contains(saida, "motivo=") {
		t.Errorf("a queda saiu sem motivo=:\n%s", saida)
	}
}
```

- [ ] **Step 2: Rodar e ver falhar.**

Run: `go test ./cmd/gobsidian/ -run TestFallbackEmProcessoEhWarnComMotivo -v`
Expected: FAIL — "saiu sem WARN".

- [ ] **Step 3: Implementar.** Trocar os três `log.Info` de queda por `log.Warn`, cada um com seu `motivo`, **sem mudar o texto das mensagens** (a regra global vale):

| Linha atual | Motivo |
|---|---|
| `"nao foi possivel iniciar o daemon; servindo em processo"` | `daemon-nao-subiu` |
| `"daemon nao respondeu apos iniciar; servindo em processo"` | `daemon-mudo` |
| `"GOBSIDIAN_NO_DAEMON definida; servindo em processo sem tentar o daemon"` | continua `INFO` — é escolha explícita do usuário, não degradação |

E o caso de versão divergente, que hoje se confunde com os outros: quando `startErr` casar `ipc.ErrVersionMismatch` ou `ipc.ErrConfigMismatch`, o motivo é `versao-divergente` ou `config-divergente`.

```go
	if startErr := daemon.EnsureStarted(ctx, cfg, daemonStartTimeout, iniciar); startErr != nil {
		log.Warn("nao foi possivel iniciar o daemon; servindo em processo",
			"motivo", motivoDaQueda(startErr),
			"err", startErr, "errno", errnoDe(startErr))
		return serveEmProcesso(ctx, cfg, log)
	}
```

```go
// motivoDaQueda classifica a queda para o modo em processo.
//
// Os tres casos sao defeitos diferentes com consertos diferentes, e ate
// 2026-09-08 davam a mesma linha INFO. Versao divergente, em particular, nao
// e transitoria: ela vai se repetir em toda partida ate alguem reinstalar, e
// e o unico caso em que o daemon do outro lado esta VIVO e saudavel.
func motivoDaQueda(err error) string {
	switch {
	case errors.Is(err, ipc.ErrVersionMismatch):
		return "versao-divergente"
	case errors.Is(err, ipc.ErrConfigMismatch):
		return "config-divergente"
	default:
		return "daemon-nao-subiu"
	}
}
```

- [ ] **Step 4: Rodar e ver passar.**

Run: `go test ./cmd/gobsidian/ -run TestFallback -v`
Expected: PASS.

- [ ] **Step 5: Confirmar que os testes vizinhos continuam válidos.** `TestPonteCaiParaModoEmProcesso` e `TestServePonteVersaoDiferenteCaiParaProcesso` já existem e afirmam a queda; rodar os dois e conferir que nenhum dependia do nível `INFO`.

Run: `go test ./cmd/gobsidian/ -run TestServePonte -v`

- [ ] **Step 6: Prova de mutação.** Trocar `log.Warn` de volta por `log.Info` num dos três pontos, rodar, confirmar que o teste nomeia a falha, restaurar. Colar a saída.

- [ ] **Step 7: Gate e commit.**

---

### Task 191: `pid` e `versao` em todo log do daemon

**Files:**
- Modify: `cmd/gobsidian/daemon.go` (`novoLoggerDoDaemon`)
- Modify: `cmd/gobsidian/daemon_log_test.go`

**Interfaces:**
- Consumes: a variável de pacote `version` de `cmd/gobsidian/main.go:23`.
- Produces: nada novo.

**Contexto:** o arquivo de log é **um só por cofre** e recebe append de N instâncias ao longo de meses (medido: 727 261 bytes). Determinar qual processo escreveu cada linha, durante a investigação de 2026-09-08, exigiu cruzar mtime de arquivo com `StartTime` de processo — e ainda ficou ambíguo.

- [ ] **Step 1: Escrever o teste que falha.**

```go
// TestLoggerDoDaemonCarimbaPidEVersao: o arquivo de log e unico por cofre e
// recebe append de N instancias. Em 2026-09-08, descobrir quem escreveu cada
// linha custou cruzar mtime de arquivo com StartTime de processo, e o
// resultado ainda ficou ambiguo.
func TestLoggerDoDaemonCarimbaPidEVersao(t *testing.T) {
	cofre := t.TempDir()
	t.Setenv("LOCALAPPDATA", t.TempDir()) // desvia do diretorio real do usuario

	log, fechar, err := novoLoggerDoDaemon(cofre, slog.LevelInfo)
	if err != nil {
		t.Fatalf("novoLoggerDoDaemon: %v", err)
	}
	log.Info("linha de teste")
	if err := fechar(); err != nil {
		t.Fatalf("fechar: %v", err)
	}

	caminho, err := daemonLogPath(cofre)
	if err != nil {
		t.Fatalf("daemonLogPath: %v", err)
	}
	conteudo, err := os.ReadFile(caminho)
	if err != nil {
		t.Fatalf("lendo o log: %v", err)
	}

	linha := string(conteudo)
	esperado := "pid=" + strconv.Itoa(os.Getpid())
	if !strings.Contains(linha, esperado) {
		t.Errorf("log sem %s:\n%s", esperado, linha)
	}
	if !strings.Contains(linha, "versao=") {
		t.Errorf("log sem versao=:\n%s", linha)
	}
}
```

- [ ] **Step 2: Rodar e ver falhar.**

Run: `go test ./cmd/gobsidian/ -run TestLoggerDoDaemonCarimba -v`
Expected: FAIL — "log sem pid=".

- [ ] **Step 3: Implementar.** Em `novoLoggerDoDaemon`, trocar a última linha:

```go
	// pid e versao em TODA linha, e nao so na de partida.
	//
	// O arquivo e unico por cofre e recebe append de N instancias ao longo de
	// meses -- 727 261 bytes na maquina do dono em 2026-09-08. Sem estes dois
	// campos, descobrir qual processo escreveu uma linha exige cruzar mtime de
	// arquivo com StartTime de processo, e o resultado fica ambiguo quando duas
	// instancias convivem, que foi exatamente o caso investigado.
	log := slog.New(slog.NewTextHandler(io.MultiWriter(f, os.Stderr), &slog.HandlerOptions{Level: level})).
		With("pid", os.Getpid(), "versao", version)
	return log, f.Close, nil
```

- [ ] **Step 4: Rodar e ver passar.**

Run: `go test ./cmd/gobsidian/ -run TestLoggerDoDaemonCarimba -v`
Expected: PASS.

- [ ] **Step 5: Confirmar que os gates de log continuam lendo o que liam.** `pid` e `versao` são campos **acrescentados**; nenhuma mensagem muda. Provar:

```bash
pwsh -File scripts/test_orphans.ps1 -Cycles 5 -Scenario daemon-idle
```
Expected: `[OK] Nenhum daemon orfao em 5 ciclos`.

- [ ] **Step 6: Prova de mutação.** Remover o `.With(...)`, rodar, confirmar a falha nomeada, restaurar. Colar a saída.

- [ ] **Step 7: Gate e commit.**

---

### Task 192: `cleanupSocketFile` ganha plano B e passa a relatar o que encontrou

**Files:**
- Modify: `internal/ipc/ipc.go:375-386`
- Modify: `internal/ipc/ipc_test.go`

**Interfaces:**
- Consumes: nada novo.
- Produces: a variável de pacote `removerArquivo = os.Remove`, trocável em teste — o mesmo padrão de `iniciarDaemonFn` em `cmd/gobsidian/ponte.go`.

**Contexto:** falhou 20+ vezes em produção desde 2026-09-01 com erro 1920. O comentário atual afirma que a função é independente de plataforma "porque `os.Remove` se comporta igual nas três" — e produção mostra um terceiro estado no Windows que nenhum reprodutor conhecido explica. A task **não** deve fingir que descobriu a causa: ela adiciona uma saída e melhora o relato.

- [ ] **Step 1: Escrever os testes que falham.**

```go
// TestCleanupSocketFilePlanoB cobre as 20+ falhas medidas em producao desde
// 2026-09-01: os.Remove devolveu "The file cannot be accessed by the system"
// (1920) e o daemon nao subiu, derrubando toda ponte para o modo em processo.
//
// O mecanismo desse estado NAO foi reproduzido -- dois cenarios foram testados
// em 2026-09-08 e nenhum produz o par observado (dial 10022 / remove 1920). O
// que esta task fecha nao e a causa, e a AUSENCIA DE SAIDA: um caminho
// ocupado por um arquivo que nao se apaga ainda pode ser liberado por rename,
// que o Windows permite ate sobre executavel em uso (medido).
func TestCleanupSocketFilePlanoB(t *testing.T) {
	dir := t.TempDir()
	caminho := filepath.Join(dir, "p.sock")
	if err := os.WriteFile(caminho, nil, 0o600); err != nil {
		t.Fatal(err)
	}

	original := removerArquivo
	removerArquivo = func(string) error { return errors.New("The file cannot be accessed by the system.") }
	defer func() { removerArquivo = original }()

	if err := cleanupSocketFile(caminho); err != nil {
		t.Fatalf("plano B nao liberou o caminho: %v", err)
	}
	if _, err := os.Lstat(caminho); !os.IsNotExist(err) {
		t.Fatalf("o caminho continua ocupado: %v", err)
	}
}

// TestCleanupSocketFileRelataOEstado: quando NEM o plano B funciona, o erro
// tem de dizer o que havia no caminho. Ate 2026-09-08 ele dizia so o texto do
// sistema, e quem lia nao sabia se havia arquivo, diretorio ou nada.
func TestCleanupSocketFileRelataOEstado(t *testing.T) {
	dir := t.TempDir()
	caminho := filepath.Join(dir, "sub") // diretorio, e nao arquivo
	if err := os.Mkdir(caminho, 0o700); err != nil {
		t.Fatal(err)
	}

	original := removerArquivo
	removerArquivo = func(string) error { return errors.New("recusado") }
	defer func() { removerArquivo = original }()

	// Sem permissao de escrita no pai o rename tambem falha; num diretorio
	// temporario comum ele funcionaria, entao o alvo aqui e o RELATO.
	err := cleanupSocketFile(caminho)
	if err == nil {
		return // plano B deu conta; nada a relatar
	}
	if !strings.Contains(err.Error(), "estado do caminho") {
		t.Fatalf("o erro nao relata o estado encontrado: %v", err)
	}
}

// TestCleanupSocketFileAusenteNaoEhErro preserva o contrato que ja existia.
func TestCleanupSocketFileAusenteNaoEhErro(t *testing.T) {
	if err := cleanupSocketFile(filepath.Join(t.TempDir(), "nao-existe")); err != nil {
		t.Fatalf("ausente virou erro: %v", err)
	}
}
```

- [ ] **Step 2: Rodar e ver falhar.**

Run: `go test ./internal/ipc/ -run TestCleanupSocketFile -v`
Expected: FAIL — `undefined: removerArquivo`.

- [ ] **Step 3: Implementar.**

```go
// removerArquivo e os.Remove, numa variavel para que o teste possa forcar o
// ramo de falha. O padrao e o mesmo de iniciarDaemonFn em
// cmd/gobsidian/ponte.go: producao usa o valor real e ninguem o troca.
var removerArquivo = os.Remove

// cleanupSocketFile libera o caminho do socket de uma partida anterior que
// morreu sem fechar o listener. Ausente nao e erro.
//
// Ate 2026-09-08 era os.Remove puro, e o comentario afirmava que isso bastava
// "porque os.Remove se comporta igual nas tres" plataformas. Producao
// contradisse 20+ vezes desde 2026-09-01, no Windows, com "The file cannot be
// accessed by the system" (ERROR_CANT_ACCESS_FILE, 1920) -- e cada falha
// derrubou toda ponte do cofre para o modo em processo, que e o estado em que
// duas instancias gravam o MESMO cache de busca.
//
// O mecanismo desse estado continua DESCONHECIDO. Dois cenarios foram
// reproduzidos em 2026-09-08 e nenhum produz o par observado em campo:
//
//	listener fechado limpo    dial 10061   arquivo ja nao existe (Close desvincula)
//	processo morto a forca    dial 10061   os.Remove com SUCESSO
//	PRODUCAO                  dial 10022   os.Remove com 1920
//
// Por isso o conserto nao e um tratamento do errno -- classificar por numero
// e o que o comentario de Listen ja proibe, e o que erraria em silencio no
// proximo estado desconhecido. E uma SAIDA: se o caminho nao se apaga, tire o
// arquivo do caminho. Renomear e permitido no Windows onde apagar nao e --
// medido em 2026-09-08, inclusive sobre executavel em uso.
//
// O nome de destino leva timestamp para nunca colidir com uma sobra anterior,
// e a remocao dele e best-effort: se ficar, e lixo inerte fora do caminho que
// importa, e a limpeza do instalador o alcanca depois.
func cleanupSocketFile(path string) error {
	err := removerArquivo(path)
	if err == nil || os.IsNotExist(err) {
		return nil
	}

	desviado := fmt.Sprintf("%s.orfao-%d", path, time.Now().UnixNano())
	if renErr := os.Rename(path, desviado); renErr == nil {
		_ = removerArquivo(desviado)
		return nil
	}

	return fmt.Errorf("%w (estado do caminho: %s)", err, estadoDoCaminho(path))
}

// estadoDoCaminho descreve o que ha em path, para que o erro diga mais que o
// texto do sistema. Ate 2026-09-08 quem lia "The file cannot be accessed by
// the system" nao sabia se havia arquivo, diretorio ou nada.
func estadoDoCaminho(path string) string {
	fi, err := os.Lstat(path)
	if err != nil {
		return fmt.Sprintf("lstat falhou: %v", err)
	}
	return fmt.Sprintf("modo=%s tamanho=%d", fi.Mode(), fi.Size())
}
```

- [ ] **Step 4: Rodar e ver passar.**

Run: `go test ./internal/ipc/ -run TestCleanupSocketFile -v`
Expected: PASS nos três.

- [ ] **Step 5: Confirmar que `Listen` não regrediu.** `TestListenRecusaSocketComDonoVivo` (ou o nome equivalente em `listen_orfao_test.go`) prova que `Listen` não rouba socket de daemon vivo — o plano B **não pode** furar isso, porque ele só roda depois de `alguemEscuta` já ter dito que ninguém escuta.

Run: `go test ./internal/ipc/ ./internal/daemon/ -v`

- [ ] **Step 6: Prova de mutação.** Apagar o bloco do rename, rodar, confirmar que `TestCleanupSocketFilePlanoB` falha nomeando "plano B nao liberou o caminho", restaurar. Colar a saída.

- [ ] **Step 7: Gate e commit.**

---

### Task 193: NÃO EXISTE — medido e descartada, não esquecida

A spec (§5) pede que `doctor` diga o modo e o dono do `--cache-dir`. Ao ler o
pacote antes de escrever a task, os dois já estavam lá:

- `internal/doctor/daemon.go:107` (`checkDaemonVivo`) já distingue os três
  estados e escreve, em letra: `handshake completo`, `nenhum daemon rodando (a
  ponte servira em processo)`, ou `arquivo existe mas o handshake falhou`.
- `internal/doctor/checks.go:212` (`checkCacheDir`) já imprime o caminho do
  `--cache-dir` do cofre.

Escrever uma checagem nova para os mesmos dois fatos seria **duas contas da
mesma regra** — exatamente o que o CLAUDE.md proíbe, e pior num comando de
diagnóstico, onde duas linhas que discordam levantam dúvida em vez de
resolvê-la.

**O que a spec pede e que de fato falta** é a terceira pergunta: *há mais de um
processo `gobsidian` servindo este cofre agora?* Foi ela que, em 2026-09-08, só
ficou visível pela comparação de milissegundos entre linhas de log duplicadas.
Responder exige **enumeração de processos**, que o plano 2 precisa construir de
qualquer forma (D-06 da spec: listar PID e cofre antes de encerrar).

Portanto ela sai daqui e entra no plano 2, onde vem de graça em vez de exigir
código de plataforma escrito duas vezes. Isto é redução de escopo declarada, não
silenciosa.

---

### Task 194: rotação do log do daemon, e a dívida registrada

**Files:**
- Modify: `cmd/gobsidian/daemon.go` (`novoLoggerDoDaemon`)
- Modify: `cmd/gobsidian/daemon_log_test.go`
- Modify: `docs/ESTADO.md`
- Modify: `docs/OPERACAO.md`

**Interfaces:**
- Consumes: `daemonLogPath`.
- Produces: nada novo.

**Contexto:** 727 261 bytes sem limite, medido em 2026-09-08. O teto é **5 MB**, guardando **um** arquivo anterior (`.sock.log.1`). Rotacionar, nunca apagar: o log é a única memória do daemon, e a investigação de 2026-09-08 dependeu de linhas de 2026-08-24.

- [ ] **Step 1: Escrever o teste que falha.** Gravar um arquivo de log acima do teto, chamar `novoLoggerDoDaemon`, e afirmar que (a) o arquivo corrente ficou pequeno e (b) `.1` existe com o conteúdo antigo.

- [ ] **Step 2: Rodar e ver falhar.**

- [ ] **Step 3: Implementar.** Antes do `os.OpenFile` em `novoLoggerDoDaemon`: se o arquivo existe e passa de 5 MB, `os.Rename(path, path+".1")` — substituindo o `.1` anterior. Erro de rotação **não** pode impedir a abertura do log: loga-se e segue, porque um daemon sem log é pior que um log grande.

- [ ] **Step 4: Rodar e ver passar.**

- [ ] **Step 5: Documentar.** Em `docs/OPERACAO.md`: o guarda-chuva de 6 s da Task 189, os campos `pid`/`versao`, os motivos de queda da Task 190 e a rotação. Em `docs/ESTADO.md`, a dívida aberta, **sem fingir que foi resolvida**:

> **A causa do encerramento pendurado não foi encontrada.** Medido em
> 2026-09-07: o daemon PID 42856 registrou `encerramento solicitado
> reason=idle` às 19:30:36 e seguiu vivo 20 h, com 0 s de CPU em 3 s de
> amostragem. Sabe-se que o travamento está **fora** de `lifecycle.Shutdown`,
> porque a guarda dela teria matado o processo em 6 s. Quais das três esperas
> (`wg.Wait` em `internal/daemon/daemon.go`, `lc.Wait`, `c.Esperar`) travou,
> **não foi determinado**: o binário é compilado com `-s -w` e `dlv attach`
> responde `could not find goroutine array`. A Task 189 pôs anteparo, não
> conserto. Para investigar de novo: compilar sem `-s -w`, instalar, e esperar
> a reprodução.

- [ ] **Step 6: Validar encoding.**

```bash
python -c "open('docs/ESTADO.md',encoding='utf-8').read()" && echo "[OK] UTF-8 valido"
python -c "open('docs/OPERACAO.md',encoding='utf-8').read()" && echo "[OK] UTF-8 valido"
```

- [ ] **Step 7: Gate completo e commit.** `pwsh -File scripts/verify.ps1` **sem** `-Skip*`, e colar as últimas linhas no relatório.

---

## Depois deste plano

O segundo plano — o instalador (D-01 a D-11 da spec) — depende de duas coisas que este entrega: a enumeração de processos que a Task 193 deixou explicitamente de fora, e a limpeza de lixo órfão que reusa `TravaEmUso` e `alguemEscuta`. Nenhuma task deste plano cria pacote novo.
