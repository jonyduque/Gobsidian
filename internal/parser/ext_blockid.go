package parser

import (
	"github.com/yuin/goldmark"
	gast "github.com/yuin/goldmark/ast"
)

// BlockIDNode e um stub. Task 15 substitui este arquivo por uma extensao que
// reconhece marcadores "^id" no fim de linha.
type BlockIDNode struct {
	gast.BaseInline

	ID    string
	Start int64
	End   int64
}

var kindBlockID = gast.NewNodeKind("BlockID")

func (n *BlockIDNode) Kind() gast.NodeKind { return kindBlockID }

func (n *BlockIDNode) Dump(src []byte, level int) {
	gast.DumpHelper(n, src, level, map[string]string{"ID": n.ID}, nil)
}

// BlockIDExtension e um stub sem efeito, so para o pacote compilar antes da
// Task 15.
type BlockIDExtension struct{}

func (BlockIDExtension) Extend(goldmark.Markdown) {}
