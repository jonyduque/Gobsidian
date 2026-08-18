package parser_test

import (
	"testing"

	"github.com/jonyd/gobsidian/internal/parser"
)

func TestInlineTags(t *testing.T) {
	src := "Nota sobre #civil e #civil/obrigacoes e #proc-civil.\n"

	note := parser.Parse([]byte(src))

	want := []string{"civil", "civil/obrigacoes", "proc-civil"}
	if len(note.Tags) != len(want) {
		t.Fatalf("tags = %v, quer %v", note.Tags, want)
	}
	for i := range want {
		if note.Tags[i] != want[i] {
			t.Errorf("tags[%d] = %q, quer %q", i, note.Tags[i], want[i])
		}
	}
}

func TestTagRejections(t *testing.T) {
	tests := []struct{ name, in string }{
		{"heading nao e tag", "# Titulo\n"},
		{"so digitos nao e tag", "item #123\n"},
		{"dentro de codigo", "```\n#civil\n```\n"},
		{"codigo inline", "use `#civil` aqui\n"},
		{"cerquilha isolada", "a # b\n"},
		{"dentro de url", "veja https://x.com/a#secao\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			note := parser.Parse([]byte(tt.in))
			if len(note.Tags) != 0 {
				t.Errorf("tags = %v, quer nenhuma", note.Tags)
			}
		})
	}
}

func TestTagsFromFrontmatterMerge(t *testing.T) {
	src := "---\ntags:\n  - civil\n  - penal\n---\nTexto com #civil e #tributario.\n"

	note := parser.Parse([]byte(src))

	// civil aparece nas duas fontes e conta uma vez so.
	want := map[string]bool{"civil": true, "penal": true, "tributario": true}
	if len(note.Tags) != len(want) {
		t.Fatalf("tags = %v, quer %d unicas", note.Tags, len(want))
	}
	for _, tag := range note.Tags {
		if !want[tag] {
			t.Errorf("tag inesperada: %q", tag)
		}
	}
}

// TestTagPunctuationBoundary confirma que pontuacao de fim de frase nao entra
// no nome da tag: e alfabeto de tagNameChar que decide onde o nome acaba, nao
// o fim da frase.
func TestTagPunctuationBoundary(t *testing.T) {
	tests := []struct{ name, in, want string }{
		{"ponto final", "veja #tag.\n", "tag"},
		{"virgula", "veja #tag, depois\n", "tag"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			note := parser.Parse([]byte(tt.in))
			if len(note.Tags) != 1 || note.Tags[0] != tt.want {
				t.Errorf("tags = %v, quer [%q]", note.Tags, tt.want)
			}
		})
	}
}

// TestTagTrailingSlash confirma que uma barra no fim nao produz um segmento
// de hierarquia vazio: "#tag/" e a tag "tag", nao "tag/".
func TestTagTrailingSlash(t *testing.T) {
	note := parser.Parse([]byte("veja #tag/ depois\n"))
	if len(note.Tags) != 1 || note.Tags[0] != "tag" {
		t.Errorf("tags = %v, quer [tag]", note.Tags)
	}
}

// TestTagEmptyHierarchySegments confirma que segmento vazio de hierarquia
// nao chega em Tags: barra dupla interna e barra inicial sao colapsadas do
// mesmo jeito que a barra final ja era (TestTagTrailingSlash). Sem isto,
// "#a//b" e "#/civil" produzem segmento vazio ao serem divididos por '/' por
// tag_list (hierarchical:true), e todas as tags com segmento vazio na mesma
// profundidade colidem num so no sem nome.
func TestTagEmptyHierarchySegments(t *testing.T) {
	tests := []struct{ name, in, want string }{
		{"barra dupla interna", "veja #a//b depois\n", "a/b"},
		{"barra inicial", "veja #/civil depois\n", "civil"},
		{"barra dupla e final compostas", "veja #a//b/c depois\n", "a/b/c"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			note := parser.Parse([]byte(tt.in))
			if len(note.Tags) != 1 || note.Tags[0] != tt.want {
				t.Errorf("tags = %v, quer [%q]", note.Tags, tt.want)
			}
		})
	}
}

// TestTagCharset confirma hifen, underscore, maiuscula e letras acentuadas.
func TestTagCharset(t *testing.T) {
	tests := []struct{ name, in, want string }{
		{"hifen", "#tag-com-hifen\n", "tag-com-hifen"},
		{"underscore", "#tag_com_underscore\n", "tag_com_underscore"},
		{"maiuscula", "#Maiuscula\n", "Maiuscula"},
		{"acentuada", "#Maiúscula\n", "Maiúscula"},
		{"cedilha e til", "#ação\n", "ação"},
		{"digito e letra", "#1a\n", "1a"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			note := parser.Parse([]byte(tt.in))
			if len(note.Tags) != 1 || note.Tags[0] != tt.want {
				t.Errorf("tags = %v, quer [%q]", note.Tags, tt.want)
			}
		})
	}
}

// TestTagOpeningPunctuation confirma o conjunto de pontuacao de abertura que
// autoriza uma tag: parenteses, colchetes, aspas retas, aspa curva de
// abertura e travessao.
func TestTagOpeningPunctuation(t *testing.T) {
	tests := []struct{ name, in, want string }{
		{"parenteses", "(#tag)\n", "tag"},
		{"colchetes", "[#tag]\n", "tag"},
		{"aspas retas", "\"#tag\"\n", "tag"},
		{"aspa curva de abertura", "“#tag”\n", "tag"},
		{"travessao colado", "texto—#tag\n", "tag"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			note := parser.Parse([]byte(tt.in))
			if len(note.Tags) != 1 || note.Tags[0] != tt.want {
				t.Errorf("tags = %v, quer [%q]", note.Tags, tt.want)
			}
		})
	}
}

// TestTagAdjacentHash confirma que "#tag#outra" produz uma unica tag: a
// segunda cerquilha e precedida por uma letra, o que a regra 1 recusa.
func TestTagAdjacentHash(t *testing.T) {
	note := parser.Parse([]byte("#tag#outra\n"))
	if len(note.Tags) != 1 || note.Tags[0] != "tag" {
		t.Errorf("tags = %v, quer [tag]", note.Tags)
	}
}

// TestTagDoubleHash confirma que "##tag" nao produz tag nenhuma: a primeira
// cerquilha nao tem alfabeto de tag depois dela (a segunda cerquilha nao
// pertence ao alfabeto), e a segunda e precedida por uma cerquilha, que nao
// e inicio de linha, espaco, nem pontuacao de abertura.
func TestTagDoubleHash(t *testing.T) {
	note := parser.Parse([]byte("##tag\n"))
	if len(note.Tags) != 0 {
		t.Errorf("tags = %v, quer nenhuma", note.Tags)
	}
}

// TestTagLoneHash confirma que uma cerquilha sozinha, ou seguida so de
// pontuacao, nao produz tag.
func TestTagLoneHash(t *testing.T) {
	tests := []struct{ name, in string }{
		{"cerquilha no fim da linha", "termina em #\n"},
		{"cerquilha seguida de pontuacao", "vazio #! aqui\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			note := parser.Parse([]byte(tt.in))
			if len(note.Tags) != 0 {
				t.Errorf("tags = %v, quer nenhuma", note.Tags)
			}
		})
	}
}

// TestTagInsideWikilinkAlias confirma que uma tag dentro do alias de um
// wikilink nao e extraida como tag independente: o parser de wikilink ja
// consumiu o intervalo inteiro antes do parser de tag ver o '#'.
func TestTagInsideWikilinkAlias(t *testing.T) {
	note := parser.Parse([]byte("[[nota|#tag]]\n"))
	if len(note.Tags) != 0 {
		t.Errorf("tags = %v, quer nenhuma (a tag esta dentro do wikilink)", note.Tags)
	}
	if len(note.Links) != 1 || note.Links[0].Alias != "#tag" {
		t.Errorf("links = %+v, quer alias #tag", note.Links)
	}
}

// TestTagInHeading confirma que uma tag no fim de uma linha de heading e
// extraida normalmente: o goldmark analisa o texto do heading como inline
// igual a qualquer paragrafo.
func TestTagInHeading(t *testing.T) {
	note := parser.Parse([]byte("## Titulo #tag\n"))
	if len(note.Tags) != 1 || note.Tags[0] != "tag" {
		t.Errorf("tags = %v, quer [tag]", note.Tags)
	}
}

// TestTagAdjacentToEmphasisMarkers confirma que '*', '_' e '~' autorizam uma
// tag logo depois deles: "**#civil**", "*#civil*", "_#civil_" e "~~#civil~~"
// sao idioma comum no Obsidian (tag em negrito/italico/riscado), que a
// aplicacao real indexa. Sem isto a tag desaparece de tag_list, do filtro
// tags e do indice inteiro, porque esses tres caracteres sao categoria
// Unicode Po/Pc — nao pontuacao de abertura — e ficavam de fora da checagem.
func TestTagAdjacentToEmphasisMarkers(t *testing.T) {
	tests := []struct{ name, in, want string }{
		{"negrito", "**#civil**\n", "civil"},
		{"italico asterisco", "*#civil*\n", "civil"},
		// O '_' de fechamento entra no NOME, nao so na checagem de abertura:
		// '_' ja fazia parte do alfabeto de tagNameChar antes desta correcao
		// (para "#tag_com_underscore"), entao um '_' colado no fim e
		// consumido pela mesma regra que aceita "#tag_com_underscore" — nao
		// ha como distinguir "underscore de tag" de "underscore de italico"
		// olhando so pro caractere. Ambiguidade pre-existente do alfabeto,
		// ortogonal ao que esta checagem (regra 1, caractere PRECEDENTE)
		// resolve.
		{"italico underscore", "_#civil_\n", "civil_"},
		{"riscado", "~~#civil~~\n", "civil"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			note := parser.Parse([]byte(tt.in))
			if len(note.Tags) != 1 || note.Tags[0] != tt.want {
				t.Errorf("tags = %v, quer [%q]", note.Tags, tt.want)
			}
		})
	}
}

// TestTagRealURLs confirma que fragmentos de URLs realistas nao viram tags,
// para tres formatos comuns de URL com fragmento.
func TestTagRealURLs(t *testing.T) {
	tests := []struct{ name, in string }{
		{"wikipedia com parenteses", "https://en.wikipedia.org/wiki/Go_(programming_language)#Concurrency\n"},
		{"github readme", "https://github.com/user/repo#readme\n"},
		{"docs python", "https://docs.python.org/3/library/re.html#re.match\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			note := parser.Parse([]byte(tt.in))
			if len(note.Tags) != 0 {
				t.Errorf("tags = %v, quer nenhuma", note.Tags)
			}
		})
	}
}
