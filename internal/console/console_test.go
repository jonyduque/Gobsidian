package console_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jonyd/gobsidian/internal/console"
)

// esc e o byte que abre toda sequencia ANSI. Procurar por ele e o unico jeito
// honesto de afirmar "nao ha formatacao aqui": procurar pelas sequencias
// especificas que o pacote emite passaria a valer nada no dia em que alguem
// acrescentasse uma cor nova.
const esc = "\x1b"

func TestBufferNuncaRecebeANSI(t *testing.T) {
	// Um bytes.Buffer nao e terminal. Se este teste falhar, toda saida
	// redirecionada para arquivo ou pipe esta saindo suja.
	var buf bytes.Buffer
	con := console.New(&buf)

	if con.Colored() {
		t.Fatal("Colored() = true para um bytes.Buffer; nada que nao seja *os.File pode receber cor")
	}

	con.OK("tudo certo")
	con.Warn("cuidado")
	con.Err("falhou")
	con.Info("detalhe")
	con.Item("item")
	con.Step("etapa")
	con.Detail("indentado")
	con.Line("solto")

	if got := buf.String(); strings.Contains(got, esc) {
		t.Errorf("saida para buffer contem sequencia ANSI:\n%q", got)
	}
}

func TestMarcadoresContinuamEmASCII(t *testing.T) {
	// A regra do projeto: [OK], [!], [i], [*], [...] em ASCII puro, porque um
	// console PowerShell em CP-850 renderiza o resto como lixo. A cor SOMA ao
	// marcador; se ela for descartada, a informacao tem de sobreviver.
	var buf bytes.Buffer
	con := console.New(&buf)

	con.OK("a")
	con.Warn("b")
	con.Err("c")
	con.Info("d")
	con.Item("e")
	con.Step("f")

	got := buf.String()
	for _, marcador := range []string{"[OK] a", "[!] b", "[!] c", "[i] d", "[*] e", "[...] f"} {
		if !strings.Contains(got, marcador) {
			t.Errorf("marcador %q ausente da saida:\n%s", marcador, got)
		}
	}
	for i := 0; i < len(got); i++ {
		if got[i] > 0x7F {
			t.Fatalf("byte nao-ASCII na posicao %d da saida: %q", i, got)
		}
	}
}

func TestArquivoComumNaoRecebeCor(t *testing.T) {
	// `gobsidian doctor > relatorio.txt` grava num arquivo comum, que nao e
	// dispositivo de caractere. Sem esta regra o arquivo sai cheio de escape.
	f, err := os.Create(filepath.Join(t.TempDir(), "relatorio.txt"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()

	if console.SupportsColor(f) {
		t.Error("SupportsColor = true para arquivo comum em disco")
	}
}

func TestEstilosSaoNoOpSemCor(t *testing.T) {
	// Bold/Dim/Italic tem de devolver o texto intacto quando nao ha cor --
	// nao o texto com sequencia vazia, nem o texto truncado.
	var buf bytes.Buffer
	con := console.New(&buf)

	for _, caso := range []struct {
		nome string
		got  string
	}{
		{"Bold", con.Bold("texto")},
		{"Dim", con.Dim("texto")},
		{"Italic", con.Italic("texto")},
	} {
		if caso.got != "texto" {
			t.Errorf("%s sem cor = %q, quer %q", caso.nome, caso.got, "texto")
		}
	}
}

func TestNewPlainNuncaColore(t *testing.T) {
	// NewPlain existe para saida destinada a outro programa. Ele nao consulta
	// ambiente nenhum: mesmo num terminal de verdade, com cor habilitada,
	// devolve texto cru.
	f, err := os.Create(filepath.Join(t.TempDir(), "x.txt"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()

	if console.NewPlain(f).Colored() {
		t.Error("NewPlain devolveu Stream colorido")
	}
}
