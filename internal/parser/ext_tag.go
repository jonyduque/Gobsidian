package parser

import (
	"strings"
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
// Unicode, mas abrem citacao no uso comum. '*', '_' e '~' tambem entram a
// parte: sao marcadores de enfase/tachado (negrito, italico, ~~riscado~~),
// categoria Po/Pc no Unicode — nao pontuacao de abertura —, mas
// "**#civil**" e idioma comum no Obsidian, que indexa a tag. Nenhuma URL
// tem fragmento precedido por um desses tres, entao admiti-los nao custa
// nada contra o falso positivo que esta funcao existe pra barrar.
//
// Isto NAO e o que impede uma tag dentro do destino de um link Markdown
// valido: "[texto](#introducao)" nao produz tag porque o parser de link do
// CommonMark consome "(#introducao)" inteiro antes de qualquer outro
// parser inline ver aquele '#' — o '(' nunca chega a ser oferecido a
// tagPrecedingOK como caractere precedente, porque o '#' nunca dispara o
// gatilho. A prova de que a protecao e essa, e nao a regra de pontuacao de
// abertura acima (que TAMBEM aceitaria '(' se chegasse a ser consultada): um
// link malformado como "[S] (#introducao)" (espaco entre ']' e '(', que o
// CommonMark nao aceita como link) ainda produz uma tag — o '(' sobra como
// texto solto, tagPrecedingOK e consultada de verdade, e responde "sim".
func tagPrecedingOK(r rune) bool {
	if r == '\n' || unicode.IsSpace(r) {
		return true
	}
	if r == '"' || r == '\'' || r == '*' || r == '_' || r == '~' {
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
	// barra sobra como texto literal depois da tag. So a faixa consumida
	// (Advance) usa este limite; o NOME gravado no no ainda passa por
	// collapseTagSlashes abaixo, que cobre as outras duas posicoes.
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

	// Regra 3: segmento vazio de hierarquia nao pode chegar em Tags. Barra
	// dupla interna ("#a//b") e barra inicial ("#/civil") produzem segmento
	// vazio ao serem divididas por '/' — o mesmo problema que a barra final
	// ja tratava acima, so que nas outras duas posicoes. tag_list com
	// hierarchical:true monta a arvore dividindo por '/', e um segmento
	// vazio vira no sem nome onde toda tag com segmento vazio na mesma
	// profundidade colide num so; o parametro prefix erra a subarvore do
	// mesmo jeito.
	node := &TagNode{Name: collapseTagSlashes(string(rest[:trimmed]))}

	block.Advance(1 + trimmed)
	return node
}

// collapseTagSlashes remove segmentos vazios de hierarquia colapsando
// sequencias de '/' e removendo-as das pontas — ver comentario de regra 3
// em tagParser.Parse.
func collapseTagSlashes(s string) string {
	parts := strings.Split(s, "/")
	out := parts[:0]
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}
	return strings.Join(out, "/")
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
