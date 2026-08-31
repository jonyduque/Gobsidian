package parser

import "testing"

// TestSpanDeLinkMarkdown é o achado M10, e os dois casos produziam offset
// PLAUSÍVEL E ERRADO — que é pior que `offsetUnknown`, porque quem reescreve
// confia nele.
//
// O span é o que `note_move` e a reescrita de link usam para fatiar o arquivo.
// Um span do link errado corrompe a nota do usuário.
func TestSpanDeLinkMarkdown(t *testing.T) {
	casos := []struct {
		nome  string
		corpo string
		// spans esperados, na ordem em que os links saem do parser.
		spans []string
	}{
		{
			// Caso 1 do M10. Sondado antes da correção: os DOIS links recebiam
			// o span "[](um.md)" — o `strings.Index` do corpo inteiro achava a
			// primeira ocorrência de "[](" para os dois.
			nome:  "dois links de texto vazio",
			corpo: "primeiro [](um.md) e segundo [](dois.md) fim\n",
			spans: []string{"[](um.md)", "[](dois.md)"},
		},
		{
			// Caso 2 do M10. Sondado antes: o link para dest.md recebia
			// "[alt](img.png)" — truncado e SOBREPOSTO ao span do embed.
			nome:  "imagem aninhada dentro de link",
			corpo: "antes [![alt](img.png)](dest.md) depois\n",
			spans: []string{"[![alt](img.png)](dest.md)", "![alt](img.png)"},
		},
		{
			nome:  "link comum nao regride",
			corpo: "texto [rotulo](alvo.md) fim\n",
			spans: []string{"[rotulo](alvo.md)"},
		},
		{
			nome:  "embed comum nao regride",
			corpo: "texto ![alt](img.png) fim\n",
			spans: []string{"![alt](img.png)"},
		},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			links := Parse([]byte(c.corpo)).Links
			if len(links) != len(c.spans) {
				t.Fatalf("links = %d, queria %d: %+v", len(links), len(c.spans), links)
			}
			// Ordena por Kind não: a ordem do parser é a que o teste fixa, e se
			// ela mudar isso também é informação.
			for i, l := range links {
				if l.Start < 0 || l.End > int64(len(c.corpo)) || l.End <= l.Start {
					t.Errorf("link %d (%s): offsets %d..%d fora do corpo", i, l.Target, l.Start, l.End)
					continue
				}
				got := c.corpo[l.Start:l.End]
				if got != c.spans[i] {
					t.Errorf("link %d (target=%q): span = %q, queria %q\n"+
						"o span e o que note_move e a reescrita usam para fatiar o arquivo",
						i, l.Target, got, c.spans[i])
				}
			}
		})
	}
}

// TestSpanAmbiguoDevolveDesconhecido fixa a borda honesta da correção.
//
// Quando o MESMO destino aparece duas vezes com texto vazio, a âncora pelo
// destino volta a ser ambígua. A resposta certa é `offsetUnknown` — offset
// ausente é honesto, offset plausível e errado não é, e foi exatamente isso que
// o M10 produzia.
func TestSpanAmbiguoDevolveDesconhecido(t *testing.T) {
	corpo := "um [](mesmo.md) e outro [](mesmo.md) fim\n"
	links := Parse([]byte(corpo)).Links
	if len(links) != 2 {
		t.Fatalf("links = %d, queria 2", len(links))
	}
	for i, l := range links {
		if l.Start != offsetUnknown {
			got := ""
			if l.Start >= 0 && l.End <= int64(len(corpo)) && l.End > l.Start {
				got = corpo[l.Start:l.End]
			}
			t.Errorf("link %d: Start = %d (span %q), queria offsetUnknown\n"+
				"dois destinos iguais com texto vazio nao sao distinguiveis; "+
				"chutar um deles e o defeito que o M10 descreve", i, l.Start, got)
		}
	}
}
