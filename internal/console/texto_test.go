package console

import (
	"bytes"
	"strings"
	"testing"
)

// TestTextoSaiComAcentoOndeOConsoleAguenta: ate 2026-09-16 toda mensagem do
// produto era escrita sem acento, porque um console em CP-850 renderiza
// "permissão" como lixo. A decisao passou a ser medida, como a dos glifos.
func TestTextoSaiComAcentoOndeOConsoleAguenta(t *testing.T) {
	t.Setenv(VarDeEstilo, "1")

	var buf bytes.Buffer
	s := NewPlain(&buf)
	s.OK("permissão de escrita")
	s.Detail("diretório de cache")
	s.Line("versão %s", "v1.9.0")

	got := buf.String()
	for _, quer := range []string{"permissão de escrita", "diretório de cache", "versão v1.9.0"} {
		if !strings.Contains(got, quer) {
			t.Errorf("saida sem %q:\n%s", quer, got)
		}
	}
}

// TestTextoPerdeAcentoOndeOConsoleNaoAguenta e o outro lado, e e o que impede
// a volta do lixo: no console que nao aguenta, o acento CAI em vez de virar
// "permissÒo".
func TestTextoPerdeAcentoOndeOConsoleNaoAguenta(t *testing.T) {
	t.Setenv(VarDeEstilo, "0")

	var buf bytes.Buffer
	s := NewPlain(&buf)
	s.OK("permissão de escrita")
	s.Detail("diretório de cache")

	got := buf.String()
	for _, quer := range []string{"permissao de escrita", "diretorio de cache"} {
		if !strings.Contains(got, quer) {
			t.Errorf("saida sem %q:\n%s", quer, got)
		}
	}
	for _, r := range got {
		if r > 0x7f {
			t.Fatalf("saida com caractere fora do ASCII (%q) no console que nao aguenta:\n%s", r, got)
		}
	}
}

// TestMolduraAdaptaTituloERodape: Moldura DEVOLVE linhas, e quem as escreve
// (a lista de selecao, o modal de botoes) nao passa por Line -- entao a
// adaptacao tem de acontecer aqui dentro.
//
// A primeira redacao adaptava titulo e rodape DEPOIS de montar as bordas com
// eles: o texto acentuado ia para a tela, e nenhum teste via, porque os que
// existiam imprimiam por Bloco, que passa por Line. Quem achou foi o
// ineffassign do golangci-lint.
func TestMolduraAdaptaTituloERodape(t *testing.T) {
	t.Setenv(VarDeEstilo, "0")
	forcarGlifos = &glifosASCII
	defer func() { forcarGlifos = nil }()

	linhas := NewPlain(nil).Moldura("Diagnóstico", []string{"  permissão"}, "ação")
	for i, l := range linhas {
		for _, r := range l {
			if r > 0x7f {
				t.Fatalf("linha %d da moldura com %q, fora do ASCII, no console que nao aguenta:\n%s",
					i, r, strings.Join(linhas, "\n"))
			}
		}
	}
	if !strings.Contains(linhas[0], "Diagnostico") {
		t.Errorf("o titulo sumiu da borda de cima: %q", linhas[0])
	}
	if !strings.Contains(linhas[len(linhas)-1], "acao") {
		t.Errorf("o rodape sumiu da borda de baixo: %q", linhas[len(linhas)-1])
	}
}

// TestMolduraFechaComTextoAcentuado: a moldura mede a largura das linhas, e
// medir uma grafia para escrever outra deixa a borda torta -- por isso o texto
// e adaptado ANTES da medida.
func TestMolduraFechaComTextoAcentuado(t *testing.T) {
	for _, caso := range []struct{ nome, unicode string }{
		{"com acento", "1"},
		{"sem acento", "0"},
	} {
		t.Run(caso.nome, func(t *testing.T) {
			t.Setenv(VarDeEstilo, caso.unicode)
			forcarGlifos = &glifosASCII
			defer func() { forcarGlifos = nil }()

			var buf bytes.Buffer
			s := NewPlain(&buf)
			s.Bloco("Diagnóstico", []string{"  permissão de leitura", "  espaço em disco"}, "ação")

			linhas := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
			quer := colunasNaTela(linhas[0])
			for i, l := range linhas {
				if got := colunasNaTela(l); got != quer {
					t.Errorf("linha %d com %d colunas na tela, a borda tem %d:\n%s", i, got, quer, buf.String())
				}
			}
		})
	}
}
