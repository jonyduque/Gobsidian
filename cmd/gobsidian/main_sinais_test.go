package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// TestMainArmaSinaisAntesDeTudo: a captura de sinais tem de ser a PRIMEIRA
// coisa de main para serve e daemon. Um sinal que chegasse antes dela matava o
// processo sem reason= (ver lifecycle.ArmarSinais). A regra de 2026-09-09 --
// nada roda antes de o encerramento estar armado -- tambem estava so num
// comentario, e foi quebrada pelo autor dela no dia seguinte: invariante que
// vale escrever vale um gate.
//
// Le o codigo de main.go: os dois primeiros comandos sao o calculo de
// `servidor` e o if que chama lifecycle.ArmarSinais.
func TestMainArmaSinaisAntesDeTudo(t *testing.T) {
	codigo, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "main.go", codigo, 0)
	if err != nil {
		t.Fatal(err)
	}
	var corpo []ast.Stmt
	for _, d := range f.Decls {
		if fn, ok := d.(*ast.FuncDecl); ok && fn.Name.Name == "main" && fn.Recv == nil {
			corpo = fn.Body.List
		}
	}
	if len(corpo) < 2 {
		t.Fatalf("main com %d comandos; esperava ao menos 2", len(corpo))
	}
	texto := func(s ast.Stmt) string {
		return string(codigo[fset.Position(s.Pos()).Offset:fset.Position(s.End()).Offset])
	}
	primeiro, segundo := texto(corpo[0]), texto(corpo[1])
	if !strings.HasPrefix(primeiro, "servidor :=") || !strings.Contains(primeiro, `"serve"`) || !strings.Contains(primeiro, `"daemon"`) {
		t.Errorf("o primeiro comando de main tem de decidir se o processo e serve ou daemon; e:\n%s", primeiro)
	}
	if _, ok := corpo[1].(*ast.IfStmt); !ok || !strings.Contains(segundo, "lifecycle.ArmarSinais()") {
		t.Errorf("o segundo comando de main tem de ser o if que chama lifecycle.ArmarSinais(); e:\n%s", segundo)
	}
}
