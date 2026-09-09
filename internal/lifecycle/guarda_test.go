package lifecycle

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"os/exec"
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
