package parser

import (
	"unicode"
	"unicode/utf8"

	"github.com/yuin/goldmark"
	gast "github.com/yuin/goldmark/ast"
	gparser "github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

var kindTag = gast.NewNodeKind("Tag")

// TagNode e o no de AST para uma tag inline "#tag" ou "#tag/subtag"
// hierarquica.
type TagNode struct {
	gast.BaseInline

	Name string
}

func (n *TagNode) Kind() gast.NodeKind { return kindTag }

func (n *TagNode) Dump(src []byte, level int) {
	gast.DumpHelper(n, src, level, map[string]string{"Name": n.Name}, nil)
}

// tagNameChar e o alfabeto aceito no corpo de uma tag: letras e digitos
// Unicode (para "#Maiúscula" e "#ação"), hifen e underscore, e '/' para
// hierarquia (regra 3 — "civil/obrigacoes" e uma tag, nao duas).
func tagNameChar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' || r == '/'
}

// tagPrecedingOK diz se o caractere que vem antes do '#' autoriza uma tag
// ali: inicio de linha ou de documento (o goldmark reporta isso como '\n'),
// espaco, ou pontuacao de abertura. Sem essa checagem, "url#secao" e
// "a#b" produzem tags que nao existem — o falso positivo de maior volume
// possivel, porque qualquer URL com fragmento no cofre vira uma tag.
//
// "Pontuacao de abertura" cobre parenteses/colchetes/chaves e variantes
// Unicode (categoria Ps), aspas curvas e guillemets de abertura (categoria
// Pi), e pontuacao de travessao — hifen, en dash, em dash (categoria Pd).
// Aspa dupla reta e aspa simples reta entram a parte: sao categoria Po no
// Unicode, mas abrem citacao no uso comum.
func tagPrecedingOK(r rune) bool {
	if r == '\n' || unicode.IsSpace(r) {
		return true
	}
	if r == '"' || r == '\'' {
		return true
	}
	return unicode.In(r, unicode.Ps, unicode.Pi, unicode.Pd)
}

type tagParser struct{}

// Trigger dispara este parser em cada '#' oferecido pelo goldmark. O
// goldmark ja suprime o gatilho dentro de blocos de codigo cercados,
// indentados, spans de codigo inline e blocos/comentarios HTML — nao ha
// estado de fence para rastrear aqui (regra 5).
func (p *tagParser) Trigger() []byte { return []byte{'#'} }

func (p *tagParser) Parse(_ gast.Node, block text.Reader, _ gparser.Context) gast.Node {
	line, _ := block.PeekLine()
	if len(line) == 0 || line[0] != '#' {
		return nil
	}

	// Regra 1: o '#' precisa ser precedido por inicio de linha, espaco ou
	// pontuacao de abertura.
	if !tagPrecedingOK(block.PrecendingCharacter()) {
		return nil
	}

	rest := line[1:]
	width := 0
	hasLetter := false
	for width < len(rest) {
		r, size := utf8.DecodeRune(rest[width:])
		if r == utf8.RuneError && size <= 1 {
			break
		}
		if !tagNameChar(r) {
			break
		}
		if unicode.IsLetter(r) {
			hasLetter = true
		}
		width += size
	}

	// Uma ou mais barras no FIM do nome nao fecham segmento de hierarquia
	// nenhum — "#tag/" e a tag "tag", nao "tag/" com um segmento vazio. A
	// barra sobra como texto literal depois da tag.
	trimmed := width
	for trimmed > 0 && rest[trimmed-1] == '/' {
		trimmed--
	}

	// Regra 2: precisa conter ao menos uma letra. "#123" nao e tag no
	// Obsidian — e referencia de issue, numero de titulo, nota de rodape.
	// Regra 4: '#' seguido de espaco (ou de nada que pertenca ao alfabeto)
	// nao produz nome nenhum aqui, e cai neste mesmo caso.
	if trimmed == 0 || !hasLetter {
		return nil
	}

	node := &TagNode{Name: string(rest[:trimmed])}

	block.Advance(1 + trimmed)
	return node
}

// TagExtension registra o inline parser no goldmark.
type TagExtension struct{}

func (TagExtension) Extend(md goldmark.Markdown) {
	md.Parser().AddOptions(
		gparser.WithInlineParsers(
			// Prioridade sem concorrencia conhecida com wikilink (150) ou
			// block-id (100): '#' nao aparece no gatilho de nenhum dos dois.
			util.Prioritized(&tagParser{}, 120),
		),
	)
}
