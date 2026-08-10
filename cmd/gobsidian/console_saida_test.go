package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/jonyd/gobsidian/internal/console"
)

// esc abre toda sequencia ANSI. Afirmar sobre o byte, e nao sobre as
// sequencias que o pacote emite hoje, e o que mantem estes testes valendo
// quando alguem acrescentar uma cor nova.
const esc = "\x1b"

// TestAjudaRedirecionadaNaoSaiFormatada cobre `gobsidian --help > ajuda.txt`.
//
// O template de ajuda chama funcoes que decidem sobre cor lendo o writer do
// comando. Se alguma delas passar a olhar os.Stdout, este teste continua
// verde num terminal e o arquivo de quem redireciona sai cheio de escape --
// por isso a assercao e sobre o BUFFER, que nunca e terminal.
func TestAjudaRedirecionadaNaoSaiFormatada(t *testing.T) {
	// A arvore real, pelos mesmos motivos de TestServeNaoEscreveNoStdout. A
	// primeira versao deste teste montava comandos de mentira sem Run, e o
	// cobra nao lista comando nao-executavel: o teste reprovou por defeito do
	// proprio fixture, afirmando que o template tinha comido o conteudo.
	casos := []struct {
		nome    string
		args    []string
		trechos []string
	}{
		{
			nome:    "raiz",
			args:    []string{"--help"},
			trechos: []string{"Comandos disponiveis:", "doctor", "serve", "Flags:", "--help"},
		},
		{
			// Subcomando e executavel, entao aqui a secao de uso aparece.
			nome:    "subcomando",
			args:    []string{"doctor", "--help"},
			trechos: []string{"Uso:", "gobsidian doctor", "--vault"},
		},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			root := newRootCmd()
			var out bytes.Buffer
			root.SetOut(&out)
			root.SetErr(&out)

			root.SetArgs(c.args)
			if err := root.Execute(); err != nil {
				t.Fatalf("Execute: %v", err)
			}

			got := out.String()
			if got == "" {
				t.Fatal("ajuda saiu vazia: assercao sobre saida vazia passa sempre e nao cobre nada")
			}
			if strings.Contains(got, esc) {
				t.Errorf("ajuda redirecionada contem sequencia ANSI:\n%q", got)
			}
			// O realce nao pode ter comido o conteudo junto com a cor.
			for _, trecho := range c.trechos {
				if !strings.Contains(got, trecho) {
					t.Errorf("ajuda perdeu o trecho %q:\n%s", trecho, got)
				}
			}
		})
	}
}

// TestSaidaDeErroRedirecionadaNaoSaiFormatada cobre `gobsidian ... 2> erro.txt`.
func TestSaidaDeErroRedirecionadaNaoSaiFormatada(t *testing.T) {
	var buf bytes.Buffer
	console.New(&buf).Err("cofre inacessivel: %s", "raiz nao existe")

	got := buf.String()
	if !strings.Contains(got, "[!] cofre inacessivel: raiz nao existe") {
		t.Errorf("mensagem de erro perdeu conteudo ou marcador: %q", got)
	}
	if strings.Contains(got, esc) {
		t.Errorf("erro redirecionado contem sequencia ANSI: %q", got)
	}
}

// TestServeNaoConstroiConsoleNoStdout e a guarda da regra que mais custa caro
// aqui: stdout pertence ao JSON-RPC.
//
// Uma sequencia ANSI escrita no stdout de `serve` corrompe a sessao do mesmo
// jeito que um fmt.Println, e com o mesmo sintoma -- o servidor some do host
// sem erro nenhum. O teste percorre a arvore de comandos e afirma que `serve`
// e `daemon` NAO tem, eles proprios, nada que formate saida: os dois escrevem
// diagnostico por slog no stderr, e nunca pelo console.
//
// A assercao e sobre o writer que o cobra entrega ao comando. Se alguem ligar
// um console ao stdout de serve, o writer deixa de ser o buffer deste teste e
// a saida capturada passa a conter escape.
func TestServeNaoEscreveNoStdout(t *testing.T) {
	for _, nome := range []string{"serve", "daemon"} {
		t.Run(nome, func(t *testing.T) {
			// newRootCmd, e nao um root montado aqui: SilenceUsage vive nele,
			// e um root construido a mao sem essa flag despeja o texto de uso
			// na saida e faz este teste afirmar sobre um programa que nao e o
			// que se distribui. Foi exatamente o que aconteceu na primeira
			// versao deste teste.
			root := newRootCmd()

			var stdout bytes.Buffer
			root.SetOut(&stdout)
			root.SetErr(&bytes.Buffer{})

			// --vault ausente: o comando reprova na configuracao, antes de
			// abrir cofre ou socket. O que interessa e que o caminho de erro
			// tambem nao escreve nada em stdout.
			root.SetArgs([]string{nome})
			_ = root.Execute()

			if got := stdout.String(); got != "" {
				t.Errorf("%s escreveu em stdout, que pertence ao JSON-RPC: %q", nome, got)
			}
		})
	}
}
