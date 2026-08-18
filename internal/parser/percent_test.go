package parser

import "testing"

// A tabela de casos de borda e o que impede a proxima pessoa de trocar a
// implementacao a mao por net/url — que resolveria o mesmo problema e
// derrubaria a garantia de ausencia de rede verificada no CI.
//
// Tem dois consumidores: os destinos de link Markdown, aqui, e as URIs de
// resource em mcpsrv. Um caso que sair desta tabela sai dos dois.
func TestPercentDecode(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"sem escape passa direto", "Civil/PONTO 03.md", "Civil/PONTO 03.md"},
		{"espaco", "Civil/PONTO%2003.md", "Civil/PONTO 03.md"},
		{"minusculas no escape", "a%2fb", "a/b"},
		{"maiusculas no escape", "a%2Fb", "a/b"},

		// UTF-8 multibyte chega como varios %XX seguidos. Decodificar por byte
		// e o que remonta o caractere; decodificar por rune quebraria aqui.
		{"cedilha", "Acao%20e%20Reac%CC%A7ao.md", "Acao e Reaçao.md"},
		{"acento precomposto", "Cap%C3%ADtulo.md", "Capítulo.md"},

		// Percent que nao inicia escape valido e texto do usuario, nao erro.
		// Recusar o link ou inventar um byte seria pior que preservar.
		{"percent solto", "50% de desconto.md", "50% de desconto.md"},
		{"escape invalido", "a%ZZb", "a%ZZb"},
		{"percent no fim", "arquivo%", "arquivo%"},
		{"percent truncado", "arquivo%2", "arquivo%2"},
		{"percent literal escapado", "50%25.md", "50%.md"},

		{"vazio", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PercentDecode(tt.in); got != tt.want {
				t.Errorf("PercentDecode(%q) = %q, quer %q", tt.in, got, tt.want)
			}
		})
	}
}

// A decodificacao vale para link Markdown e nao para wikilink. No Obsidian
// [[a%20b]] procura um arquivo literalmente chamado "a%20b" — decodificar ali
// faria o wikilink apontar para outro arquivo.
//
// E Raw fica byte-exato nos dois casos: note_move reescreve a partir dele, e
// normalizar ali trocaria o texto que o usuario escreveu.
func TestPercentDecodeAppliesOnlyToMarkdownTargets(t *testing.T) {
	note := Parse([]byte("[texto](Civil/PONTO%2003.md) e [[Civil/PONTO%2003]]\n"))
	if len(note.Links) != 2 {
		t.Fatalf("links = %d, quer 2: %+v", len(note.Links), note.Links)
	}

	var md, wiki Link
	for _, l := range note.Links {
		switch l.Kind {
		case LinkMarkdown:
			md = l
		case LinkWiki:
			wiki = l
		}
	}

	if md.Target != "Civil/PONTO 03.md" {
		t.Errorf("markdown Target = %q, quer decodificado", md.Target)
	}
	if md.Raw != "Civil/PONTO%2003.md" {
		t.Errorf("markdown Raw = %q, quer byte-exato — note_move reescreve a partir dele", md.Raw)
	}
	if wiki.Target != "Civil/PONTO%2003" {
		t.Errorf("wikilink Target = %q, quer NAO decodificado", wiki.Target)
	}
}
