package lifecycle

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"runtime"
	"runtime/pprof"
	"strings"
	"testing"
	"time"
)

// envCenarioDaGuarda diz ao processo FILHO qual cenario ele deve encenar.
// Vazia, o processo e o pai e conduz a bateria.
const envCenarioDaGuarda = "GOBSIDIAN_TESTE_GUARDA"

// orcamentoDeTeste e o orcamento que os cenarios usam. Curto de proposito: o
// que se prova aqui e o COMPORTAMENTO do relogio, nao a duracao dele.
const orcamentoDeTeste = 50 * time.Millisecond

// TestGuardaChuva cobre o defeito medido em 2026-09-07 na maquina do dono: o
// daemon PID 42856 registrou "encerramento solicitado reason=idle" as 19:30:36
// e seguiu VIVO 20 h, com 274 MB residentes, 0 s de CPU em 3 s de amostragem e
// 26 threads em Wait,UserRequest. Shutdown ja tinha guarda propria, e o
// processo estar vivo prova que o travamento estava FORA dela.
//
// Os quatro cenarios rodam em SUBPROCESSO, e nao so o primeiro.
//
// O primeiro precisa disso porque a acao e os.Exit(1), e um teste nao pode
// matar o proprio binario de teste. Os outros tres poderiam rodar em processo
// -- mas ali a assercao seria "a funcao de teste chegou ao fim", que e uma
// assercao implicita: nada no codigo NOMEIA o que se esperava. Observando de
// fora, o codigo de saida do processo e a assercao, e ela e a mesma pergunta
// nos quatro casos: o guarda-chuva derrubou este processo, ou nao?
func TestGuardaChuva(t *testing.T) {
	if cenario := os.Getenv(envCenarioDaGuarda); cenario != "" {
		encenarCenario(cenario)
		return
	}

	casos := []struct {
		nome          string
		cenario       string
		saidaEsperada int
		trechoNoLog   string
		porQueImporta string
	}{
		{
			nome:          "encerramento pendurado morre",
			cenario:       "pendurado",
			saidaEsperada: 1,
			trechoNoLog:   "encerramento travou",
			porQueImporta: "e o defeito de 2026-09-07: encerramento pedido, processo vivo para sempre",
		},
		{
			nome:          "desarmado nao morre",
			cenario:       "desarmado",
			saidaEsperada: 0,
			porQueImporta: "sem este caso, um guarda-chuva que matasse SEMPRE passaria no primeiro",
		},
		{
			nome:          "sem cancelamento nao morre",
			cenario:       "sem-cancelamento",
			saidaEsperada: 0,
			porQueImporta: "o relogio so conta a partir do cancelamento; contando desde o armamento, todo daemon ocioso morreria",
		},
		{
			nome:          "desarmar duas vezes nao entra em panic",
			cenario:       "idempotente",
			saidaEsperada: 0,
			porQueImporta: "quem chama registra por defer E pode chamar no caminho feliz; fechar um canal duas vezes entra em panic",
		},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			// Prazo no subprocesso, e nao espera indefinida.
			//
			// Sem ele, um guarda-chuva que NAO dispara deixa o filho preso para
			// sempre, e o teste morre pelo timeout global do `go test` -- dez
			// minutos depois, com "panic: test timed out" no lugar da causa. Um
			// teste que so falha por timeout nao NOMEIA o defeito, e foi
			// exatamente o que a primeira prova de mutacao desta regra mostrou.
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=TestGuardaChuva")
			cmd.Env = append(os.Environ(), envCenarioDaGuarda+"="+caso.cenario)
			saida, err := cmd.CombinedOutput()

			if ctx.Err() != nil {
				t.Fatalf("o subprocesso passou do prazo sem sair (%s). saida:\n%s",
					caso.porQueImporta, saida)
			}

			codigo := 0
			var comSaida *exec.ExitError
			if errors.As(err, &comSaida) {
				codigo = comSaida.ExitCode()
			} else if err != nil {
				t.Fatalf("o subprocesso falhou fora do esperado: %v. saida:\n%s", err, saida)
			}

			if codigo != caso.saidaEsperada {
				t.Fatalf("codigo de saida %d, esperado %d (%s). saida:\n%s",
					codigo, caso.saidaEsperada, caso.porQueImporta, saida)
			}
			if caso.trechoNoLog != "" && !strings.Contains(string(saida), caso.trechoNoLog) {
				t.Fatalf("o log nao contem %q, entao a causa nao chega a quem le. saida:\n%s",
					caso.trechoNoLog, saida)
			}
		})
	}
}

// encenarCenario e o corpo do processo FILHO. Cada ramo sai normalmente
// (codigo 0) a menos que o guarda-chuva o derrube (codigo 1).
func encenarCenario(cenario string) {
	silencioso := slog.New(slog.NewTextHandler(io.Discard, nil))

	switch cenario {
	case "pendurado":
		ctx, cancel := context.WithCancel(context.Background())
		// stderr de verdade: o pai confere que a causa aparece no log.
		_ = ArmarGuardaChuva(ctx, slog.New(slog.NewTextHandler(os.Stderr, nil)), orcamentoDeTeste)
		cancel()
		// Espera que NUNCA termina: e exatamente a forma do defeito.
		select {}

	case "desarmado":
		ctx, cancel := context.WithCancel(context.Background())
		desarmar := ArmarGuardaChuva(ctx, silencioso, orcamentoDeTeste)
		cancel()
		desarmar()
		time.Sleep(4 * orcamentoDeTeste) // passa do orcamento de proposito

	case "sem-cancelamento":
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		desarmar := ArmarGuardaChuva(ctx, silencioso, orcamentoDeTeste)
		defer desarmar()
		time.Sleep(6 * orcamentoDeTeste) // seis vezes o orcamento, sem cancelar

	case "idempotente":
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		desarmar := ArmarGuardaChuva(ctx, silencioso, time.Hour)
		desarmar()
		desarmar()
	}
}

// TestEsperarNomeiaAEsperaAntesDeBloquear e a linha que faltou em 2026-09-07.
//
// Sabe-se que o daemon PID 42856 travou numa das tres esperas de encerramento;
// QUAL delas nao foi possivel determinar, porque nenhuma dizia que tinha
// comecado. A assercao que importa e a de ENTRADA: ela e a unica que sobrevive
// a uma espera que nunca volta.
func TestEsperarNomeiaAEsperaAntesDeBloquear(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, nil))

	// A funcao confere o log NO MEIO da espera: e o estado em que um
	// encerramento pendurado deixa o arquivo.
	var durante string
	Esperar(context.Background(), log, "goroutines-de-fundo", func() { durante = buf.String() })

	if !strings.Contains(durante, "goroutines-de-fundo") {
		t.Fatalf("o nome da espera nao estava no log ANTES dela terminar; um travamento nao deixaria rastro:\n%s", durante)
	}
	if !strings.Contains(durante, "esperando no encerramento") {
		t.Errorf("a linha de entrada nao foi escrita:\n%s", durante)
	}

	depois := buf.String()
	if !strings.Contains(depois, "espera concluida") || !strings.Contains(depois, "duracao_ms") {
		t.Errorf("a linha de saida nao traz a duracao:\n%s", depois)
	}
}

// TestDespejarPilhasNomeiaAEsperaQueTravou e a prova de que a diretiva
// `go 1.27` no go.mod comprou o que se esperava dela.
//
// Em 2026-09-07 o daemon PID 42856 travou numa de tres esperas e QUAL nunca foi
// determinado -- o binario sai com -s -w e `dlv attach` responde "could not
// find goroutine array". lifecycle.Esperar rotula a goroutine com pprof.Do, e a
// partir do Go 1.27 esse rotulo sai no cabecalho de cada goroutine no
// traceback. Este teste segura uma espera e confere que o nome dela aparece.
//
// Prova de mutacao: tirar o pprof.Do de Esperar (chamando fn direto) faz o
// caso reprovar dizendo que "espera-presa" nao esta no dump.
func TestDespejarPilhasNomeiaAEsperaQueTravou(t *testing.T) {
	preso := make(chan struct{})
	entrou := make(chan struct{})
	pronto := make(chan struct{})

	go func() {
		defer close(pronto)
		Esperar(context.Background(), slog.New(slog.NewTextHandler(io.Discard, nil)), "espera-presa", func() {
			close(entrou)
			<-preso
		})
	}()

	<-entrou
	// A goroutine fechou `entrou` DENTRO de fn, mas ainda pode nao ter chegado
	// ao recebimento que a bloqueia. Espera ela aparecer bloqueada no dump.
	var buf bytes.Buffer
	prazo := time.Now().Add(5 * time.Second)
	for {
		buf.Reset()
		DespejarPilhas(&buf)
		if strings.Contains(buf.String(), "espera-presa") || time.Now().After(prazo) {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	saida := buf.String()
	close(preso)
	<-pronto

	if !strings.Contains(saida, "goroutines no estouro do guarda-chuva") {
		t.Errorf("o dump nao traz o cabecalho que o delimita:\n%s", primeirasLinhas(saida, 5))
	}
	if !strings.Contains(saida, "espera-presa") {
		t.Errorf("o dump nao nomeia a espera presa -- o rotulo de pprof nao chegou ao traceback.\n"+
			"Sem ele o dump volta a mostrar endereco, que e o que nao respondeu em 2026-09-07.\n%s",
			primeirasLinhas(saida, 20))
	}
}

// TestDespejarPilhasTrazOPerfilDeVazamento confere a segunda metade do dump. O
// perfil goroutineleak e GA no Go 1.27; se a toolchain nao o tiver, DespejarPilhas
// omite a secao em vez de entrar em panic, e ai este caso e quem avisa.
func TestDespejarPilhasTrazOPerfilDeVazamento(t *testing.T) {
	var buf bytes.Buffer
	DespejarPilhas(&buf)
	if !strings.Contains(buf.String(), "=== goroutineleak (") {
		t.Errorf("o dump nao traz a secao goroutineleak: a toolchain nao tem o perfil, ou ele saiu do codigo")
	}
}

func primeirasLinhas(s string, n int) string {
	linhas := strings.SplitN(s, "\n", n+1)
	if len(linhas) > n {
		linhas = linhas[:n]
	}
	return strings.Join(linhas, "\n")
}

// TestSemGoroutineVazadaDepoisDoCicloDeVida usa o perfil goroutineleak, GA no
// Go 1.27, contra a classe de defeito que originou tudo isto.
//
// O perfil relata goroutine bloqueada em primitiva INALCANCAVEL -- vazamento
// provado, e nao suspeita. Um ciclo completo de lifecycle.New ate Wait nao pode
// deixar nenhuma goroutine deste pacote la dentro.
//
// A assercao olha os QUADROS de internal/lifecycle, e nao a contagem total: o
// binario de teste roda muitos pacotes e uma contagem global seria refem de
// goroutine alheia.
//
// Prova de mutacao, rodada em 2026-09-09 com uma goroutine presa num canal que
// mais ninguem alcanca acrescentada a New:
//
//	--- FAIL: TestSemGoroutineVazadaDepoisDoCicloDeVida (0.03s)
//	    guarda_test.go:303: o ciclo de vida deixou goroutine vazada:
//	    #	0x...	internal/lifecycle.New.func1+0x24	lifecycle.go:68
//
// Nao serve mutar watchSignals tirando o `<-ctx.Done()`: signal.Notify mantem
// o canal alcancavel pelo registro do runtime, entao aquela goroutine ficaria
// presa sem ser VAZAMENTO pela definicao do perfil -- que e exatamente a
// distincao registrada em DespejarPilhas.
func TestSemGoroutineVazadaDepoisDoCicloDeVida(t *testing.T) {
	pr, pw := io.Pipe()
	ctx, lc := New(context.Background(), Options{
		Stdin:               pr,
		ParentPID:           os.Getpid(),
		ParentCheckInterval: 10 * time.Millisecond,
		Logger:              slog.New(slog.NewTextHandler(io.Discard, nil)),
	})

	// EOF em stdin: o mecanismo normal de encerramento de um serve.
	if err := pw.Close(); err != nil {
		t.Fatal(err)
	}
	<-ctx.Done()
	lc.Wait()
	if err := pr.Close(); err != nil {
		t.Fatal(err)
	}

	// O perfil so enxerga o que ficou INALCANCAVEL, e isso depende de uma
	// coleta. Duas, porque a primeira pode apenas tornar alcancavel-para-
	// finalizacao o que a segunda recolhe.
	runtime.GC()
	runtime.GC()

	p := pprof.Lookup("goroutineleak")
	if p == nil {
		t.Skip("toolchain sem o perfil goroutineleak")
	}
	var buf bytes.Buffer
	if err := p.WriteTo(&buf, 1); err != nil {
		t.Fatalf("lendo o perfil: %v", err)
	}
	if strings.Contains(buf.String(), "internal/lifecycle") {
		t.Errorf("o ciclo de vida deixou goroutine vazada:\n%s", buf.String())
	}
}
