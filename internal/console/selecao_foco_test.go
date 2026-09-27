package console

import (
	"bytes"
	"strings"
	"testing"
)

// TestCaixasMarcadaFocoEVazia: desenho do dono em 2026-09-27 -- ■ marcada,
// ▣ em foco, ▢ vazia. A linha em foco mostra o glifo de foco qualquer que seja
// o estado dela.
func TestCaixasMarcadaFocoEVazia(t *testing.T) {
	t.Setenv(VarDeEstilo, "1")
	forcarGlifos = &glifosUnicode
	defer func() { forcarGlifos = nil }()

	var buf bytes.Buffer
	opcoes := []Opcao{{Rotulo: "Estudo"}, {Rotulo: "Oral"}, {Rotulo: "Revisão"}}
	desenhar(NewPlain(&buf), opcoes, []bool{true, true, false}, 0, "Quais?", false)
	saida := buf.String()

	for _, quer := range []string{"▸ ▣ Estudo", "  ■ Oral", "  ▢ Revisão"} {
		if !strings.Contains(saida, quer) {
			t.Errorf("faltou %q:\n%s", quer, saida)
		}
	}
	if strings.Contains(saida, "[■]") || strings.Contains(saida, "[▢]") {
		t.Errorf("a caixa voltou a sair entre colchetes:\n%s", saida)
	}
}

// TestCaixaEmFocoNoASCIIMostraOEstado: o conjunto ASCII nao tem glifo de foco,
// e sem cor o glifo e o que diz se a linha esta marcada.
func TestCaixaEmFocoNoASCIIMostraOEstado(t *testing.T) {
	t.Setenv(VarDeEstilo, "0")
	forcarGlifos = &glifosASCII
	defer func() { forcarGlifos = nil }()

	var buf bytes.Buffer
	opcoes := []Opcao{{Rotulo: "Estudo"}, {Rotulo: "Oral"}}
	desenhar(NewPlain(&buf), opcoes, []bool{true, false}, 0, "Quais?", false)
	saida := buf.String()
	if !strings.Contains(saida, "> [x] Estudo") || !strings.Contains(saida, "  [ ] Oral") {
		t.Errorf("a linha em foco perdeu o estado no ASCII:\n%s", saida)
	}
}

// TestRodapeDasTeclasEmNegritoEItalico: a tecla sai entre colchetes e o que
// ela faz, escrito. Sem cor, a marcacao some e as palavras ficam.
func TestRodapeDasTeclasEmNegritoEItalico(t *testing.T) {
	t.Setenv(VarDeEstilo, "1")
	forcarGlifos = &glifosUnicode
	defer func() { forcarGlifos = nil }()

	var buf bytes.Buffer
	desenhar(NewPlain(&buf), []Opcao{{Rotulo: "a"}}, []bool{false}, 0, "Quais?", false)
	saida := buf.String()
	for _, quer := range []string{"[↑]/[↓] mover", "[ espaço ] marcar", "[a] todos", "[⏎ enter] confirmar"} {
		if !strings.Contains(saida, quer) {
			t.Errorf("rodape sem %q:\n%s", quer, saida)
		}
	}
	if strings.Contains(saida, "*") {
		t.Errorf("o rodape deixou a marcacao crua:\n%s", saida)
	}
}
