package parser

import (
	"github.com/yuin/goldmark"
	gast "github.com/yuin/goldmark/ast"
	gparser "github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

var kindBlockID = gast.NewNodeKind("BlockID")

// BlockIDNode e o no de AST para um marcador "^id" no fim de linha, que
// ancora um wikilink "[[nota#^id]]" a um bloco especifico.
type BlockIDNode struct {
	gast.BaseInline

	ID string
	// Start e o offset do inicio do BLOCO pai (paragrafo, item de lista,
	// linha de citacao) — nao do '^'. End e o fim do marcador "^id", sem os
	// espacos em branco que possam seguir. E o par que note_read com
	// block_id devolve e que note_patch com replace_block substitui;
	// devolver so o marcador tornaria as duas tools inuteis.
	Start int64
	End   int64
}

// Kind identifica o no de block id para o goldmark.
func (n *BlockIDNode) Kind() gast.NodeKind { return kindBlockID }

// Dump imprime o no na arvore de depuracao do goldmark.
func (n *BlockIDNode) Dump(src []byte, level int) {
	gast.DumpHelper(n, src, level, map[string]string{"ID": n.ID}, nil)
}

// blockIDChar e o alfabeto aceito para um id: letras, digitos e hifen. Ver
// blockIDParser.Parse.
func blockIDChar(b byte) bool {
	return (b >= 'a' && b <= 'z') ||
		(b >= 'A' && b <= 'Z') ||
		(b >= '0' && b <= '9') ||
		b == '-'
}

type blockIDParser struct{}

// Trigger dispara este parser em cada '^' oferecido pelo goldmark. O
// goldmark ja suprime o gatilho dentro de blocos de codigo cercados,
// indentados, spans de codigo inline e blocos/comentarios HTML — nao ha
// estado de fence para rastrear aqui.
func (p *blockIDParser) Trigger() []byte { return []byte{'^'} }

func (p *blockIDParser) Parse(parent gast.Node, block text.Reader, _ gparser.Context) gast.Node {
	line, segment := block.PeekLine()
	if len(line) == 0 || line[0] != '^' {
		return nil
	}

	i := 1
	for i < len(line) && blockIDChar(line[i]) {
		i++
	}
	id := string(line[1:i])
	if id == "" {
		// "^" sozinho, ou seguido de um caractere fora do alfabeto — nao ha
		// candidato a id.
		return nil
	}

	// Regra 1: o marcador precisa estar no fim da linha. Qualquer coisa
	// depois do id que nao seja espaco/tab/quebra de linha significa que o
	// "^" era texto literal no meio da linha, nao um marcador de bloco.
	for _, b := range line[i:] {
		if b != ' ' && b != '\t' && b != '\r' && b != '\n' {
			return nil
		}
	}

	// Regra 3: Start e o inicio do bloco pai (paragrafo, item de lista, linha
	// de citacao), nao do '^'. O goldmark ja fez o trabalho de achar esse
	// bloco: e o proprio "parent" que o parser de inline recebe, porque a
	// analise de inline roda por bloco-folha (aquele que tem Lines()), e
	// "parent" e sempre esse bloco-folha. Sua primeira linha comeca onde o
	// bloco comeca, com os prefixos de sintaxe (marcador de lista, "> " de
	// citacao) ja descontados pelo parser de bloco correspondente.
	lines := parent.Lines()
	if lines == nil || lines.Len() == 0 {
		// Nao deveria acontecer: o goldmark so chama este Parse quando o
		// bloco pai tem ao menos uma linha (e dela que PeekLine tirou "line"
		// acima). Recusar em vez de indexar fora dos limites.
		return nil
	}

	// Regra 1b: o marcador precisa estar na ULTIMA linha do bloco. Um
	// paragrafo com quebra suave (um unico Enter, que o Markdown nao trata
	// como novo paragrafo) tem varias linhas dentro do MESMO bloco; um
	// "^id" no meio delas nao e um block id no Obsidian — e texto literal.
	// Sem esta checagem, "linha um\nlinha dois ^b\nlinha tres ^c" produziria
	// dois blocos com faixas sobrepostas: Start vem sempre da primeira linha
	// do bloco (regra 3), entao "b" e "c" comecariam no mesmo lugar e so o
	// End diferiria — replace_block em "b" sobrescreveria "linha tres", que
	// pertence a "c".
	lastLine := lines.At(lines.Len() - 1)
	if segment.Start < lastLine.Start || segment.Start >= lastLine.Stop {
		return nil
	}

	start := int64(lines.At(0).Start)

	node := &BlockIDNode{
		ID:    id,
		Start: start,
		End:   int64(segment.Start + i),
	}

	block.Advance(i)
	return node
}

// BlockIDExtension registra o inline parser no goldmark.
type BlockIDExtension struct{}

// Extend registra o parser inline de block id no goldmark.
//
// E parser INLINE, e nao varredura de texto, de proposito: o goldmark nao
// chama parsers inline dentro de bloco de codigo, entao a supressao dentro de
// crase sai de graca em vez de virar mais uma regra para manter.
func (BlockIDExtension) Extend(md goldmark.Markdown) {
	md.Parser().AddOptions(
		gparser.WithInlineParsers(
			// Prioridade sem concorrencia conhecida com wikilink (150) ou o
			// link padrao do CommonMark (200): '^' nao aparece no gatilho de
			// nenhum dos dois.
			util.Prioritized(&blockIDParser{}, 100),
		),
	)
}
