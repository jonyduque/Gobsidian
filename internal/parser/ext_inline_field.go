package parser

import (
	"github.com/yuin/goldmark"
	gast "github.com/yuin/goldmark/ast"
)

// InlineFieldNode e um stub. Task 17 substitui este arquivo por uma extensao
// que reconhece campos inline do Dataview ("Chave:: valor").
type InlineFieldNode struct {
	gast.BaseInline

	Key   string
	Value string
}

var kindInlineField = gast.NewNodeKind("InlineField")

func (n *InlineFieldNode) Kind() gast.NodeKind { return kindInlineField }

func (n *InlineFieldNode) Dump(src []byte, level int) {
	gast.DumpHelper(n, src, level, map[string]string{"Key": n.Key, "Value": n.Value}, nil)
}

// InlineFieldExtension e um stub sem efeito, so para o pacote compilar antes
// da Task 17.
type InlineFieldExtension struct{}

func (InlineFieldExtension) Extend(goldmark.Markdown) {}
