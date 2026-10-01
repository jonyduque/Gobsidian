// Package vazamentotest reprova o binario de teste de um pacote que deixa
// goroutine vazada, pelo perfil goroutineleak do Go 1.27.
//
// So arquivo _test.go o importa, como vaulttest. Nao importa nada do projeto,
// e por isso qualquer pacote pode usa-lo no proprio TestMain sem ciclo -- inclusive
// as folhas (text, vault, lifecycle).
//
// # Por que existe
//
// O perfil reporta goroutine bloqueada numa primitiva de concorrencia que nao
// pode mais ser desbloqueada: um canal, um mutex ou um WaitGroup que nada mais
// alcanca. E a forma do daemon PID 42856, vivo 20 h em 2026-09-07 numa espera
// que nunca voltou. Ate 2026-10-01 o perfil so era lido no estouro do
// guarda-chuva e no doctor -- depois de o defeito chegar a maquina do dono. Aqui
// ele reprova o pacote no CI. Medido ao ligar: 9 de 11 pacotes sem nenhum
// vazamento; internal/watcher com 1 (um teste que nunca fechava o fsnotify) e
// cmd/gobsidian com 7 (o watchStdin, por desenho -- ver as permitidas de la).
//
// Goroutine parada em syscall (Read num arquivo, accept num socket) NAO entra:
// o perfil so ve primitiva de concorrencia. Ele nao substitui o gate de orfaos.
package vazamentotest

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"runtime"
	"runtime/pprof"
	"strconv"
	"strings"
)

// Conferir devolve o codigo de saida do binario de teste: o de m.Run quando ele
// ja reprovou, 1 quando sobrou goroutine vazada fora de permitidas, e 0 quando
// nao. Cada permitida e um trecho da pilha -- tipicamente o nome da funcao que a
// goroutine roda -- e quem a passa diz no proprio TestMain por que ela pode
// ficar.
//
// Uso: os.Exit(vazamentotest.Conferir(m.Run())).
func Conferir(codigo int, permitidas ...string) int {
	return conferirEm(os.Stderr, codigo, permitidas)
}

// conferirEm e Conferir com o destino do relato, para o proprio teste deste
// pacote provar a reprovacao sem imprimir "FAIL" num binario que passou.
func conferirEm(w io.Writer, codigo int, permitidas []string) int {
	if codigo != 0 {
		return codigo
	}
	relato, n := Vazadas(permitidas...)
	if n == 0 {
		return 0
	}
	_, _ = fmt.Fprintf(w, "FAIL: %d goroutine(s) vazada(s) ao fim dos testes (perfil goroutineleak):\n%s\n", n, relato)
	return 1
}

// Vazadas le o perfil goroutineleak e devolve as pilhas que nao casam com
// nenhuma permitida, e quantas goroutines elas somam.
//
// Perfil ausente conta como vazamento: uma trava que some em silencio quando o
// runtime muda e pior que nenhuma.
func Vazadas(permitidas ...string) (relato string, n int) {
	// O perfil roda a deteccao num ciclo de GC; os ciclos antes dele dao tempo
	// as goroutines que acabaram de terminar de sair da contagem.
	for range 2 {
		runtime.GC()
		runtime.Gosched()
	}
	p := pprof.Lookup("goroutineleak")
	if p == nil {
		return "perfil goroutineleak ausente neste runtime", 1
	}
	var b bytes.Buffer
	if err := p.WriteTo(&b, 1); err != nil {
		return "lendo o perfil goroutineleak: " + err.Error(), 1
	}
	return filtrar(&b, permitidas)
}

// filtrar percorre o formato debug=1: uma linha de cabecalho e, depois, um
// bloco por pilha, cada um comecando por "N @ enderecos" e seguido das linhas
// "#\t0x...\tfuncao\tarquivo:linha".
//
// O bloco comeca na linha "N @", e nao depois de uma linha em branco: o
// cabecalho vem colado no primeiro bloco. A primeira redacao separava por
// linha em branco, descartava o primeiro bloco junto com o cabecalho e
// devolvia zero com uma goroutine vazada -- TestVazadasAcusaGoroutinePresa
// reprovou com n=0.
func filtrar(r io.Reader, permitidas []string) (string, int) {
	dados, _ := io.ReadAll(r)
	var blocos []string
	var atual strings.Builder
	for _, linha := range strings.Split(string(dados), "\n") {
		if cabeca, _, ok := strings.Cut(linha, " @ "); ok {
			if _, err := strconv.Atoi(cabeca); err == nil {
				if atual.Len() > 0 {
					blocos = append(blocos, atual.String())
				}
				atual.Reset()
			}
		}
		if atual.Len() > 0 || strings.Contains(linha, " @ ") {
			atual.WriteString(linha)
			atual.WriteString("\n")
		}
	}
	if atual.Len() > 0 {
		blocos = append(blocos, atual.String())
	}

	var saida strings.Builder
	total := 0
	for _, bloco := range blocos {
		cabeca, _, _ := strings.Cut(bloco, " @ ")
		qtd, err := strconv.Atoi(cabeca)
		if err != nil || permitida(bloco, permitidas) {
			continue
		}
		total += qtd
		saida.WriteString(strings.TrimSpace(bloco))
		saida.WriteString("\n\n")
	}
	return saida.String(), total
}

func permitida(bloco string, permitidas []string) bool {
	for _, p := range permitidas {
		if p != "" && strings.Contains(bloco, p) {
			return true
		}
	}
	return false
}
