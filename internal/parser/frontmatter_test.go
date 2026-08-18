package parser_test

import (
	"strings"
	"testing"
	"time"

	"github.com/jonyd/gobsidian/internal/parser"
)

func TestSplitFrontmatter(t *testing.T) {
	tests := []struct {
		name       string
		in         string
		wantFM     string
		wantBody   string
		wantOffset int64
	}{
		{
			name:       "presente",
			in:         "---\ntitle: A\n---\n# Corpo\n",
			wantFM:     "title: A\n",
			wantBody:   "# Corpo\n",
			wantOffset: 17,
		},
		{
			name:       "ausente",
			in:         "# Corpo\n",
			wantFM:     "",
			wantBody:   "# Corpo\n",
			wantOffset: 0,
		},
		{
			name:       "tres tracos no meio nao conta",
			in:         "# Corpo\n---\nnao e frontmatter\n",
			wantFM:     "",
			wantBody:   "# Corpo\n---\nnao e frontmatter\n",
			wantOffset: 0,
		},
		{
			name:       "delimitador nao fechado",
			in:         "---\ntitle: A\n# Corpo\n",
			wantFM:     "",
			wantBody:   "---\ntitle: A\n# Corpo\n",
			wantOffset: 0,
		},
		{
			name:       "frontmatter vazio",
			in:         "---\n---\n# Corpo\n",
			wantFM:     "",
			wantBody:   "# Corpo\n",
			wantOffset: 8,
		},
		// CRLF e onde a aritmetica e menos obvia: advance conta o \r e o
		// TrimRight nao, entao o CR precisa ser contabilizado certo em tres
		// lugares. Cofre em Windows e o caso normal deste produto, e uma
		// regressao aqui desloca todo heading, bloco e link de toda nota.
		{
			name:       "CRLF",
			in:         "---\r\ntitle: A\r\n---\r\n# Corpo\r\n",
			wantFM:     "title: A\r\n",
			wantBody:   "# Corpo\r\n",
			wantOffset: 20,
		},
		{
			name:       "CRLF com frontmatter vazio",
			in:         "---\r\n---\r\n# Corpo\r\n",
			wantFM:     "",
			wantBody:   "# Corpo\r\n",
			wantOffset: 10,
		},
		// Corpo vazio no fim exato do buffer. E a forma que produz panic de
		// indice em qualquer bug de fatiamento rio abaixo, entao vale fixar
		// que o offset e igual ao comprimento total.
		{
			name:       "so frontmatter, sem corpo",
			in:         "---\ntitle: A\n---\n",
			wantFM:     "title: A\n",
			wantBody:   "",
			wantOffset: 17,
		},
		{
			name:       "sem newline final",
			in:         "---\ntitle: A\n---",
			wantFM:     "title: A\n",
			wantBody:   "",
			wantOffset: 16,
		},
		{
			name:       "delimitador de abertura no fim do arquivo",
			in:         "---\n",
			wantFM:     "",
			wantBody:   "---\n",
			wantOffset: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fm, body, off := parser.SplitFrontmatter([]byte(tt.in))
			if string(fm) != tt.wantFM {
				t.Errorf("fm = %q, quer %q", fm, tt.wantFM)
			}
			if string(body) != tt.wantBody {
				t.Errorf("body = %q, quer %q", body, tt.wantBody)
			}
			// Sem guarda de `!= 0`: zero e a resposta CERTA em quatro destes casos,
			// e pular a asercao neles deixa passar um bug que devolve zero
			// sempre. bodyOffset e o que mapeia posicao no corpo de volta para
			// posicao no arquivo — errado aqui, todo offset de heading, bloco e
			// link do produto inteiro sai deslocado.
			if off != tt.wantOffset {
				t.Errorf("offset = %d, quer %d", off, tt.wantOffset)
			}
		})
	}
}

func TestDecodeFrontmatterPreservesTypes(t *testing.T) {
	fm := []byte("titulo: Ponto 3\nnumero: 42\nativo: true\ntags:\n  - civil\n  - obrigacoes\naliases: [P3, Ponto III]\ndata: 2026-07-25\n")

	got, err := parser.DecodeFrontmatter(fm)
	if err != nil {
		t.Fatalf("DecodeFrontmatter: %v", err)
	}

	if got["titulo"] != "Ponto 3" {
		t.Errorf("titulo = %v (%T), quer string", got["titulo"], got["titulo"])
	}
	if n, ok := got["numero"].(int); !ok || n != 42 {
		t.Errorf("numero = %v (%T), quer int 42", got["numero"], got["numero"])
	}
	if b, ok := got["ativo"].(bool); !ok || !b {
		t.Errorf("ativo = %v (%T), quer bool true", got["ativo"], got["ativo"])
	}
	if tags, ok := got["tags"].([]any); !ok || len(tags) != 2 {
		t.Errorf("tags = %v (%T), quer lista de 2", got["tags"], got["tags"])
	}
	if aliases, ok := got["aliases"].([]any); !ok || len(aliases) != 2 {
		t.Errorf("aliases = %v (%T), quer lista de 2", got["aliases"], got["aliases"])
	}
	// A data e o tipo mais facil de perder: yaml.v3 devolve time.Time, e
	// qualquer camada que "normalize" para string quebra comparacao de
	// intervalo em M3 sem nenhum teste reclamar. O fixture ja carregava a
	// data desde o inicio; faltava conferir.
	if d, ok := got["data"].(time.Time); !ok || d.Format("2006-01-02") != "2026-07-25" {
		t.Errorf("data = %v (%T), quer time.Time 2026-07-25", got["data"], got["data"])
	}
}

// Frontmatter malformado nao pode derrubar o parse: a nota ainda tem corpo,
// headings e links uteis. O erro e reportado e o resto segue.
func TestDecodeFrontmatterMalformedReturnsError(t *testing.T) {
	if _, err := parser.DecodeFrontmatter([]byte("a: [1, 2\nb: :\n")); err == nil {
		t.Fatal("frontmatter malformado deveria devolver erro")
	}
}

func TestFrontmatterClosingDelimiterWithTrailingSpace(t *testing.T) {
	raw := []byte("---\ntitle: Prescricao\ntags: [civil]\n--- \n\n# Corpo\n\ntexto\n")

	note := parser.Parse(raw)

	if len(note.Tags) != 1 || note.Tags[0] != "civil" {
		t.Fatalf("tags = %v, quer [civil] — o bloco YAML virou corpo", note.Tags)
	}
	if note.Frontmatter == nil || note.Frontmatter["title"] != "Prescricao" {
		t.Errorf("frontmatter = %v, quer title=Prescricao", note.Frontmatter)
	}

	_, _, bodyOffset := parser.SplitFrontmatter(raw)
	corpo := string(raw[bodyOffset:])
	if !strings.HasPrefix(corpo, "\n# Corpo") {
		t.Errorf("BodyOffset=%d aponta para %q; quer o inicio do corpo real",
			bodyOffset, corpo[:min(20, len(corpo))])
	}
}

func TestFrontmatterOpeningDelimiterWithTrailingSpace(t *testing.T) {
	raw := []byte("--- \ntitle: Abastecimento\n---\n# Corpo\n")
	note := parser.Parse(raw)
	if note.Frontmatter == nil || note.Frontmatter["title"] != "Abastecimento" {
		t.Errorf("frontmatter = %v, quer title=Abastecimento", note.Frontmatter)
	}
}

func TestFrontmatterDelimiterWithTrailingTabs(t *testing.T) {
	raw := []byte("---\t\ntitle: Tabulacao\n---\t\n# Corpo\n")
	note := parser.Parse(raw)
	if note.Frontmatter == nil || note.Frontmatter["title"] != "Tabulacao" {
		t.Errorf("frontmatter = %v, quer title=Tabulacao", note.Frontmatter)
	}
}

func TestFrontmatterDelimiterWithExtraTextRejected(t *testing.T) {
	raw := []byte("--- extra\ntitle: Rejeitado\n---\n# Corpo\n")
	fm, body, off := parser.SplitFrontmatter(raw)
	if fm != nil {
		t.Errorf("fm = %q, quer nil para delimitador invalido", fm)
	}
	if off != 0 {
		t.Errorf("offset = %d, quer 0", off)
	}
	if string(body) != string(raw) {
		t.Errorf("body = %q, quer raw inteiro", body)
	}
}
