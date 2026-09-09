package lifecycle

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime"
	"runtime/pprof"
	"sync"
	"time"
)

// OrcamentoDeEncerramento e o teto do encerramento INTEIRO -- nao de uma etapa.
//
// 6 s, e o numero nao e livre. scripts/test_orphans.ps1:6-9 mede com uma janela
// de $SettleMs = 8000 e o comentario dela fixa a relacao: "8s > o guarda-chuva
// de 6s de lifecycle.Shutdown. (...) Se este numero mudar, o de la mudou
// primeiro." A janela do harness tem de ser MAIOR que o guarda-chuva do
// produto; adotar 8 s aqui igualaria os dois e o processo sairia exatamente
// quando o gate para de esperar.
//
// O mesmo valor que Shutdown ja usa como hardLimit, e de proposito: sao o mesmo
// relogio com alcances diferentes, nao duas politicas.
const OrcamentoDeEncerramento = 6 * time.Second

// ArmarGuardaChuva cobre o encerramento INTEIRO com um relogio so.
//
// Shutdown ja tinha guarda propria (shutdown.go), e ela cobre apenas o corpo
// dela. Medido em 2026-09-07 na maquina do dono: o daemon PID 42856 pediu
// encerramento por ociosidade as 19:30:36, nunca registrou "daemon encerrado" e
// seguiu vivo 20 h, com 274 MB residentes, 0 s de CPU em 3 s de amostragem e 26
// threads em Wait,UserRequest. Como o processo continuou VIVO, o travamento
// estava fora de Shutdown -- que teria feito os.Exit(1) em 6 s. Sobram as tres
// esperas que nao tinham orcamento nenhum: wg.Wait dentro de daemon.Run,
// lc.Wait e c.Esperar.
//
// Qual das tres NAO foi determinado. O binario e compilado com -s -w
// (scripts/build.ps1 e .github/workflows/release.yml) e `dlv attach` responde
// "could not find goroutine array". A decisao do dono em 2026-09-08 foi
// anteparo em vez de caca, e e por isso que este guarda-chuva e GENERICO: ele
// cobre o intervalo, nao uma causa. Se a causa aparecer depois, ele continua
// certo.
//
// O relogio so comeca a contar quando ctx e cancelado, porque e ai que o
// encerramento comeca. Um guarda-chuva que contasse desde o armamento mataria
// todo daemon ocioso, que e o oposto do que ele existe para fazer.
//
// desarmar e idempotente: quem chama o registra com defer E o chama no caminho
// feliz, e fechar um canal duas vezes entra em panic.
func ArmarGuardaChuva(ctx context.Context, log *slog.Logger, orcamento time.Duration) (desarmar func()) {
	pronto := make(chan struct{})
	var uma sync.Once

	go func() {
		// Duas esperas em serie, e nao um select unico: a primeira e "o
		// encerramento comecou?", a segunda e "ele terminou a tempo?". Um
		// select unico com time.After armaria o relogio antes do cancelamento.
		select {
		case <-pronto:
			return
		case <-ctx.Done():
		}

		select {
		case <-pronto:
		case <-time.After(orcamento):
			// Mesmo verbo e mesma saida da guarda de Shutdown: quem le o log
			// nao precisa aprender duas linguagens para o mesmo evento.
			log.Error("encerramento travou alem do guarda-chuva", "orcamento", orcamento)
			DespejarPilhas(os.Stderr)
			os.Exit(1)
		}
	}()

	return func() { uma.Do(func() { close(pronto) }) }
}

// TetoDoDespejo limita o que DespejarPilhas escreve. Um encerramento travado ja
// e um dia ruim; um dump de megabytes no stderr de um host MCP e outro.
const TetoDoDespejo = 4 << 20

// DespejarPilhas escreve o estado de TODAS as goroutines, e depois o perfil
// goroutineleak, em w.
//
// # Por que isto existe
//
// Em 2026-09-07 o daemon PID 42856 pediu encerramento por ociosidade as
// 19:30:36, nunca registrou "daemon encerrado" e seguiu vivo 20 h, com 274 MB
// residentes e 26 threads em Wait,UserRequest. As candidatas eram tres --
// wg.Wait dentro de daemon.Run, lc.Wait e c.Esperar -- e QUAL delas nunca foi
// determinado: o binario e compilado com -s -w (scripts/build.ps1 e
// release.yml) e `dlv attach` responde "could not find goroutine array".
//
// O guarda-chuva sabia que o encerramento travou, dizia isso no log e saia sem
// olhar. A resposta estava na memoria do processo o tempo todo. Um dump EM
// PROCESSO nao se importa com -s -w, que e exatamente o que derrotou o
// depurador.
//
// # As duas metades, e por que nenhuma substitui a outra
//
// runtime.Stack diz ONDE cada goroutine estava. Com a diretiva `go 1.27` no
// go.mod, o cabecalho de cada uma traz os rotulos de runtime/pprof, entao a
// espera aparece pelo NOME que lifecycle.Esperar registra -- ver o pprof.Do
// em espera.go.
//
// O perfil goroutineleak (GA no Go 1.27) da o veredito: goroutine bloqueada em
// primitiva INALCANCAVEL, que e vazamento provado. Ele nao cobre o outro caso:
// espera bloqueada em primitiva alcancavel e nunca sinalizada nao e vazamento
// pela definicao dele, e e o que o dump mostra. Prometer que o perfil sozinho
// responde repetiria o erro de afirmar sem medir.
//
// # stderr direto, e nao slog
//
// Uma pilha de goroutines dentro de um atributo de slog vira uma linha unica
// com a quebra escapada, ilegivel justamente quando mais se precisa dela.
// stdout continua sendo do JSON-RPC; stderr e onde o log ja mora.
func DespejarPilhas(w io.Writer) {
	buf := make([]byte, 64<<10)
	for {
		n := runtime.Stack(buf, true)
		if n < len(buf) {
			buf = buf[:n]
			break
		}
		if len(buf) >= TetoDoDespejo {
			// No teto o dump sai truncado, e sai assim de proposito: pilha
			// parcial responde mais que pilha nenhuma.
			break
		}
		buf = make([]byte, min(len(buf)*2, TetoDoDespejo))
	}

	_, _ = fmt.Fprintf(w, "\n=== goroutines no estouro do guarda-chuva ===\n%s\n", buf)

	// Lookup devolve nil se a toolchain nao tiver o perfil. Um encerramento
	// travado nao pode virar panic dentro do diagnostico do encerramento.
	if p := pprof.Lookup("goroutineleak"); p != nil {
		_, _ = fmt.Fprintf(w, "=== goroutineleak (%d) ===\n", p.Count())
		if err := p.WriteTo(w, 1); err != nil {
			_, _ = fmt.Fprintf(w, "(perfil goroutineleak falhou: %v)\n", err)
		}
	}
}
