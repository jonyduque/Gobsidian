package console

import (
	"bytes"
	"strings"
	"testing"
)

// TestMarcarSemCorTiraEnfaseEMantemCrase: sem cor, negrito e italico somem e
// ficam as palavras; a crase fica, porque e o que ainda separa um comando do
// texto em volta.
func TestMarcarSemCorTiraEnfaseEMantemCrase(t *testing.T) {
	s := NewPlain(&bytes.Buffer{})
	casos := []struct{ entra, sai string }{
		{"Atualiza o **gobsidian**.", "Atualiza o gobsidian."},
		{"Não altera o *PATH*.", "Não altera o PATH."},
		{"Passe `--vault` com o caminho.", "Passe `--vault` com o caminho."},
		{"*Commit*", "Commit"},
		{"Caminho do *socket* do *daemon*.", "Caminho do socket do daemon."},
		{"%s *socket(s)*, %s", "%s socket(s), %s"},
	}
	for _, c := range casos {
		if got := s.marcar(c.entra); got != c.sai {
			t.Errorf("marcar(%q) = %q, quer %q", c.entra, got, c.sai)
		}
	}
}

// TestMarcarNaoInterpretaAsteriscoSolto: "* para todos", "3 * 4" e um asterisco
// sem par nao sao marcacao, e dentro de crases nada e interpretado.
func TestMarcarNaoInterpretaAsteriscoSolto(t *testing.T) {
	for _, s := range []*Stream{NewPlain(&bytes.Buffer{}), NewComCor(&bytes.Buffer{})} {
		for _, texto := range []string{
			"digite * para todos",
			"3 * 4 = 12",
			// Um asterisco seguido de espaco nao abre, mesmo com um fechamento
			// valido adiante: sem a regra, " importante" viraria italico.
			"nota * importante* aqui",
			"um *sem par",
			"termina em *",
		} {
			if got := s.marcar(texto); got != texto {
				t.Errorf("cor=%v: marcar(%q) = %q, devia sair igual", s.color, texto, got)
			}
		}
	}
	if got := NewPlain(&bytes.Buffer{}).marcar("use `*` para todos"); got != "use `*` para todos" {
		t.Errorf("asterisco dentro de crase foi interpretado: %q", got)
	}
}

// TestMarcarComCorDesenhaCadaTrecho: com cor, cada trecho vira a sua
// sequencia, sem os delimitadores.
func TestMarcarComCorDesenhaCadaTrecho(t *testing.T) {
	s := NewComCor(&bytes.Buffer{})
	got := s.marcar("o **gobsidian** e o *PATH* com `--vault`")
	for _, quer := range []string{
		sgr(codeBold) + "gobsidian" + reset,
		sgr(codeItalic) + "PATH" + reset,
		sgr(corCodigo...) + "--vault" + reset,
	} {
		if !strings.Contains(got, quer) {
			t.Errorf("faltou %q em %q", quer, got)
		}
	}
	if strings.ContainsAny(got, "*`") {
		t.Errorf("sobrou delimitador com cor: %q", got)
	}
}

// TestMarcarReligaOEstiloDeFora: o reset de um trecho desliga tambem o estilo
// da linha inteira, e ele precisa ser religado -- sem isso uma linha de Detail
// perde o apagado depois da primeira palavra em italico.
func TestMarcarReligaOEstiloDeFora(t *testing.T) {
	s := NewComCor(&bytes.Buffer{})
	got := s.marcar("log do *daemon* aqui", codeDim)
	if !strings.Contains(got, reset+sgr(codeDim)+" aqui") {
		t.Errorf("o apagado nao foi religado depois do trecho: %q", got)
	}
	if !strings.Contains(got, sgr(codeDim, codeItalic)+"daemon") {
		t.Errorf("o trecho nao carregou o estilo de fora: %q", got)
	}
}

// TestLinhasDoStreamDesenhamAMarcacao: OK, Detail e Line sao o caminho de
// quase todo texto do produto; os tres precisam desenhar.
func TestLinhasDoStreamDesenhamAMarcacao(t *testing.T) {
	t.Setenv(VarDeEstilo, "1")
	var buf bytes.Buffer
	s := NewPlain(&buf)
	s.OK("%s", "Instala o **gobsidian**.")
	s.Detail("%s", "Abra um terminal para o *PATH* valer.")
	s.Line("%s", "Rode `gobsidian install`.")
	saida := buf.String()
	for _, quer := range []string{"Instala o gobsidian.", "para o PATH valer.", "Rode `gobsidian install`."} {
		if !strings.Contains(saida, quer) {
			t.Errorf("faltou %q:\n%s", quer, saida)
		}
	}
	if strings.Contains(saida, "*") {
		t.Errorf("sobrou asterisco:\n%s", saida)
	}
}

// TestMolduraMedeOTituloDepoisDeDesenhar: sem cor, "*Commit*" ocupa seis
// colunas, e nao oito; medir antes de desenhar deixaria a borda torta.
func TestMolduraMedeOTituloDepoisDeDesenhar(t *testing.T) {
	t.Setenv(VarDeEstilo, "0")
	s := NewPlain(&bytes.Buffer{})
	linhas := s.Moldura("**gobsidian** v1", []string{"  x"}, "*rodape*")
	if strings.Contains(strings.Join(linhas, "\n"), "*") {
		t.Fatalf("a moldura deixou asterisco:\n%s", strings.Join(linhas, "\n"))
	}
	largura := len([]rune(linhas[0]))
	for i, l := range linhas {
		if n := len([]rune(l)); n != largura {
			t.Errorf("linha %d com %d colunas, a borda tem %d:\n%s", i, n, largura, strings.Join(linhas, "\n"))
		}
	}
}

// TestSemMarcacaoMantemACrase: e o que vai para o autocompletar do shell.
func TestSemMarcacaoMantemACrase(t *testing.T) {
	if got := SemMarcacao("Instala o **gobsidian** com `--yes`."); got != "Instala o gobsidian com `--yes`." {
		t.Errorf("SemMarcacao = %q", got)
	}
}
