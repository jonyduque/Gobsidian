package index

import (
	"testing"

	"github.com/jonyd/gobsidian/internal/text"
)

// TestNormalizeStringEquivaleATextNormalize e o que autoriza apagar a segunda
// implementacao. Conferir por LEITURA que as duas fazem a mesma coisa nao basta:
// a chain e igual hoje e a divergencia apareceria na proxima edicao de uma delas.
func TestNormalizeStringEquivaleATextNormalize(t *testing.T) {
	entradas := []string{
		"", "a", "A", "ÁÉÍÓÚÃÕÇ", "Prescrição Intercorrente",
		"Notas sobre C#", "Artigo 5º — parágrafo único",
		"MAIÚSCULA com Acento", "sem acento nenhum aqui",
		"Cap\u00edtulo", "Capi\u0301tulo", // NFC e NFD do mesmo texto
		"emoji \U0001F600 no meio", "  espaços  nas  bordas  ",
	}
	for _, e := range entradas {
		if got, quer := normalizeString(e), text.Normalize(e); got != quer {
			t.Errorf("normalizeString(%q) = %q, text.Normalize da %q", e, got, quer)
		}
	}
}
