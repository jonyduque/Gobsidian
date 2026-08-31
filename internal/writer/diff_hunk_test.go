package writer_test

import (
	"strings"
	"testing"

	"github.com/jonyd/gobsidian/internal/writer"
)

// TestCabecalhoDeHunkComComprimentoZero é o achado B8.
//
// O formato unified define que, para um lado de comprimento 0, o número inicial
// é a linha ANTES do ponto de inserção. Inserir numa nota vazia produz
// `@@ -0,0 +1,1 @@`; até 2026-08-28 saía `@@ -1,0 +1,1 @@`, que o GNU `patch`
// recusa como cabeçalho inválido.
//
// Isso importa porque `dry_run` existe para o cliente poder **aplicar** ou
// revisar o que viu: um diff legível na tela e não aplicável é meio contrato.
func TestCabecalhoDeHunkComComprimentoZero(t *testing.T) {
	casos := []struct {
		nome string
		a, b string
		quer string
	}{
		{"insercao em nota vazia", "", "primeira\n", "@@ -0,0 +1,1 @@"},
		{"remocao ate esvaziar", "unica\n", "", "@@ -1,1 +0,0 @@"},
		{"insercao no topo nao e zero", "linha1\nlinha2\n", "nova\nlinha1\nlinha2\n", "@@ -1,2 +1,3 @@"},
		{"remocao no topo nao e zero", "topo\nresto\n", "resto\n", "@@ -1,2 +1,1 @@"},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			d := writer.UnifiedDiff("a.md", "b.md", c.a, c.b, 3)
			var cab string
			for _, l := range strings.Split(d, "\n") {
				if strings.HasPrefix(l, "@@") {
					cab = l
					break
				}
			}
			if cab != c.quer {
				t.Errorf("cabecalho = %q, queria %q\ndiff:\n%s", cab, c.quer, d)
			}
		})
	}
}
