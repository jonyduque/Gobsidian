package parser

import (
	"bytes"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/yuin/goldmark"
	gast "github.com/yuin/goldmark/ast"
	gparser "github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

var kindInlineField = gast.NewNodeKind("InlineField")

// InlineFieldNode e o no de AST para um campo inline do Dataview: "chave::
// valor" ou "[chave:: valor]". O produto nao reimplementa a linguagem de
// consulta do Dataview — so extrai o par chave/valor.
type InlineFieldNode struct {
	gast.BaseInline

	Key   string
	Value string
}

func (n *InlineFieldNode) Kind() gast.NodeKind { return kindInlineField }

func (n *InlineFieldNode) Dump(src []byte, level int) {
	gast.DumpHelper(n, src, level, map[string]string{"Key": n.Key, "Value": n.Value}, nil)
}

// inlineFieldKeyChar e o alfabeto aceito no corpo de uma chave (regra 2):
// letras e digitos Unicode, espaco literal, hifen e underscore. Nao inclui
// '\n' nem outros espacos em branco — sao eles que limitam a chave ao inicio
// da linha.
func inlineFieldKeyChar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == ' ' || r == '-' || r == '_'
}

type inlineFieldParser struct{}

// Trigger dispara em ':' para a forma "chave:: valor" e em '[' para a forma
// "[chave:: valor]". O goldmark ja suprime o gatilho dentro de blocos de
// codigo cercados, indentados, spans de codigo inline e blocos/comentarios
// HTML — nao ha estado de fence para rastrear aqui.
func (p *inlineFieldParser) Trigger() []byte { return []byte{':', '['} }

func (p *inlineFieldParser) Parse(parent gast.Node, block text.Reader, _ gparser.Context) gast.Node {
	line, seg := block.PeekLine()
	if len(line) == 0 {
		return nil
	}

	switch line[0] {
	case ':':
		return p.parseFromColon(parent, block, line, seg)
	case '[':
		return p.parseFromBracket(block, line)
	}
	return nil
}

// parseFromColon trata a forma sem colchetes. A chave ja foi lida pelo
// goldmark como texto comum antes de chegarmos aqui — nao ha como
// "reconsumi-la" adiante, entao olhamos pra tras no source bruto (regra 2:
// "a chave e o texto imediatamente a esquerda").
func (p *inlineFieldParser) parseFromColon(parent gast.Node, block text.Reader, line []byte, seg text.Segment) gast.Node {
	// Regra 1: exatamente dois ':'. Um terceiro logo em seguida desqualifica
	// — e o caso mais comum de falso positivo se nao checarmos.
	if len(line) < 2 || line[1] != ':' {
		return nil
	}
	if len(line) >= 3 && line[2] == ':' {
		return nil
	}

	// O limite de "inicio da linha" (regra 2) precisa vir das Lines() do
	// bloco pai, nao de procurar '\n' no source bruto: o source bruto ainda
	// contem prefixos de sintaxe ("- ", "> ") que Lines() ja descontou. Sem
	// esse limite, o '-' de um marcador de lista — que pertence ao alfabeto
	// de chave — vazaria pra dentro da chave. Ver comentario equivalente em
	// BlockIDNode.Start.
	lines := parent.Lines()
	if lines == nil || lines.Len() == 0 {
		return nil
	}
	pos := seg.Start
	lower := -1
	for i := 0; i < lines.Len(); i++ {
		s := lines.At(i)
		if pos >= s.Start && pos < s.Stop {
			lower = s.Start
			break
		}
	}
	if lower < 0 {
		return nil
	}

	key, ok := scanInlineFieldKeyBackward(block.Source(), lower, pos)
	if !ok {
		return nil
	}

	// Regra 3: o valor vai ate o fim da linha. Isso inclui qualquer "::"
	// seguinte na mesma linha — a forma sem colchetes nao suporta mais de um
	// campo por linha, do mesmo jeito que o Dataview real: so a forma entre
	// colchetes tem terminador proprio.
	value := strings.TrimSpace(string(bytes.TrimRight(line[2:], "\r\n")))

	block.Advance(len(line))
	return &InlineFieldNode{Key: key, Value: value}
}

// scanInlineFieldKeyBackward le pra tras a partir de pos, sem passar de
// lower, acumulando runas do alfabeto de chave. Devolve ok=false quando nao
// ha nenhum caractere de chave imediatamente a esquerda — e o caso de "::
// valor" sem chave nenhuma.
func scanInlineFieldKeyBackward(source []byte, lower, pos int) (string, bool) {
	i := pos
	for i > lower {
		r, size := utf8.DecodeLastRune(source[lower:i])
		if r == utf8.RuneError && size <= 1 {
			break
		}
		if !inlineFieldKeyChar(r) {
			break
		}
		i -= size
	}
	key := strings.TrimSpace(string(source[i:pos]))
	if key == "" {
		return "", false
	}
	return key, true
}

// parseFromBracket trata a forma "[chave:: valor]". Ao contrario da forma
// sem colchetes, aqui tudo — chave, "::", valor e o ']' de fechamento — esta
// a frente do cursor, entao o casamento e so pra frente, no mesmo padrao de
// wikilinkParser e blockIDParser: le a linha inteira, decide de uma vez, e
// so avanca se casar.
func (p *inlineFieldParser) parseFromBracket(block text.Reader, line []byte) gast.Node {
	i := 1
	for i < len(line) {
		r, size := utf8.DecodeRune(line[i:])
		if r == utf8.RuneError && size <= 1 {
			return nil
		}
		if !inlineFieldKeyChar(r) {
			break
		}
		i += size
	}
	key := strings.TrimSpace(string(line[1:i]))
	if key == "" {
		return nil
	}

	// Regra 1 vale aqui tambem: exatamente dois ':' logo apos a chave.
	if i+1 >= len(line) || line[i] != ':' || line[i+1] != ':' {
		return nil
	}
	if i+2 < len(line) && line[i+2] == ':' {
		return nil
	}

	// Regra 3, forma entre colchetes: o valor vai ate o ']'. Sem fechamento
	// nesta linha, isso nao e um campo entre colchetes valido.
	rest := line[i+2:]
	closeIdx := bytes.IndexByte(rest, ']')
	if closeIdx < 0 {
		return nil
	}
	value := strings.TrimSpace(string(rest[:closeIdx]))

	total := i + 2 + closeIdx + 1 // ate e incluindo ']'
	block.Advance(total)
	return &InlineFieldNode{Key: key, Value: value}
}

// InlineFieldExtension registra o inline parser no goldmark.
type InlineFieldExtension struct{}

func (InlineFieldExtension) Extend(md goldmark.Markdown) {
	md.Parser().AddOptions(
		gparser.WithInlineParsers(
			// Prioridade abaixo de wikilink (150) e do link padrao do
			// CommonMark (~200): "[chave:: valor]" precisa ser oferecido a
			// nos antes de qualquer um dos dois tentar interpretar o '['.
			util.Prioritized(&inlineFieldParser{}, 130),
		),
	)
}
