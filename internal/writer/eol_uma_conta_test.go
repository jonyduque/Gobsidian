package writer

import (
	"strings"
	"testing"

	"github.com/jonyd/gobsidian/internal/vault"
)

// TestDetectEOLConcordaComOVault é o achado M14: duas contas do mesmo fato, com
// semânticas DIFERENTES.
//
// `writer.DetectEOL` chamava o arquivo inteiro de CRLF se houvesse QUALQUER
// "\r\n"; `vault.DetectEOL` usa o estilo PREDOMINANTE, e é a resposta dela que o
// índice persiste em `Note.EOL`. Um arquivo com uma linha CRLF e mil LF era LF
// para o índice e CRLF para a escrita — e a edição de seção gravava CRLF num
// arquivo que o resto do sistema tratava como LF.
//
// As duas concordavam na maioria dos arquivos reais, que é o que fez isso
// sobreviver: contas que divergem só na borda são as que ninguém percebe.
func TestDetectEOLConcordaComOVault(t *testing.T) {
	casos := []struct {
		nome     string
		conteudo string
	}{
		{"tudo LF", "a\nb\nc\n"},
		{"tudo CRLF", "a\r\nb\r\nc\r\n"},
		{"uma CRLF no meio de muitas LF", "a\n" + strings.Repeat("linha\n", 50) + "b\r\nc\n"},
		{"uma LF no meio de muitas CRLF", "a\r\n" + strings.Repeat("linha\r\n", 50) + "b\nc\r\n"},
		{"sem quebra nenhuma", "linha unica sem fim"},
		{"vazio", ""},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			daEscrita := DetectEOL([]byte(c.conteudo))
			doIndice := string(vault.DetectEOL([]byte(c.conteudo)).Bytes())
			if daEscrita != doIndice {
				t.Errorf("writer.DetectEOL = %q, vault.DetectEOL = %q\n"+
					"duas contas do mesmo fato divergindo: a escrita grava um estilo e o indice registra outro",
					daEscrita, doIndice)
			}
		})
	}
}

// TestDetectEOLUsaOPredominante nomeia a regra escolhida, e não só a
// concordância. Sem isto, alinhar as duas na regra ERRADA — qualquer CRLF vence
// — passaria no teste acima, porque ele só compara uma com a outra.
func TestDetectEOLUsaOPredominante(t *testing.T) {
	umaCRLFEmMuitasLF := "a\n" + strings.Repeat("linha\n", 50) + "b\r\nc\n"
	if got := DetectEOL([]byte(umaCRLFEmMuitasLF)); got != "\n" {
		t.Errorf("DetectEOL = %q, queria %q: uma unica linha CRLF nao torna o arquivo CRLF", got, "\n")
	}

	umaLFEmMuitasCRLF := "a\r\n" + strings.Repeat("linha\r\n", 50) + "b\nc\r\n"
	if got := DetectEOL([]byte(umaLFEmMuitasCRLF)); got != "\r\n" {
		t.Errorf("DetectEOL = %q, queria %q: o estilo predominante e CRLF", got, "\r\n")
	}
}
