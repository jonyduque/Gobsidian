package parser

import (
	"github.com/yuin/goldmark"
	gast "github.com/yuin/goldmark/ast"
)

// TagNode e um stub. Task 16 substitui este arquivo por uma extensao que
// reconhece "#tag" e "#tag/subtag" hierarquicos no corpo.
type TagNode struct {
	gast.BaseInline

	Name string
}

var kindTag = gast.NewNodeKind("Tag")

func (n *TagNode) Kind() gast.NodeKind { return kindTag }

func (n *TagNode) Dump(src []byte, level int) {
	gast.DumpHelper(n, src, level, map[string]string{"Name": n.Name}, nil)
}

// TagExtension e um stub sem efeito, so para o pacote compilar antes da
// Task 16.
type TagExtension struct{}

func (TagExtension) Extend(goldmark.Markdown) {}
