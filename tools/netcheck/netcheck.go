// Package netcheck implementa a verificação de RNF-30: nenhum socket do
// produto sai da máquina. Ver PRD §6.4 para a formulação completa da
// garantia, reaberta em 2026-08-05 para permitir IPC local via socket Unix.
//
// A regra tem duas camadas. A primeira é de importação: nenhum pacote
// `net/*` (net/http incluído) pode ser importado pelo código do produto — só
// o pacote `net` em si é permitido, porque é dele que vêm net.Dial e
// net.Listen. A segunda é de chamada: dentro do pacote `net`, só net.Dial e
// net.Listen são aceitos, e só quando o primeiro argumento (a rede) é a
// constante literal "unix". Rede vinda de variável, de concatenação, ou
// qualquer outro valor não constante é recusada — senão net.Dial(rede,
// endereco) atravessa a regra inteira e a garantia evapora. Qualquer outra
// chamada do pacote net (DialTCP, ListenTCP, DialUDP, LookupHost etc.) é
// proibida por padrão: o par Dial/Listen com "unix" é a única porta aberta.
package netcheck

import (
	"go/ast"
	"go/constant"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// Analyzer e o analisador plugavel em go/analysis. Ele inspeciona os pacotes
// do produto, nao o fecho transitivo: net/http e x/oauth2 chegam pelo SDK de
// MCP, e isso e esperado. A garantia e sobre o que nos escrevemos.
var Analyzer = &analysis.Analyzer{
	Name: "netcheck",
	Doc:  "reporta socket que sai da maquina em codigo do produto",
	Run:  run,
}

func run(pass *analysis.Pass) (any, error) {
	info := pass.TypesInfo

	// ehConstante e valorDe leem o valor resolvido em tempo de compilação do
	// primeiro argumento de net.Dial/net.Listen. Uma expressão não constante
	// (variável, campo, retorno de função, concatenação com algo não
	// constante) não tem Value nenhum em info.Types — é exatamente o caso que
	// a regra tem de recusar, porque o analisador não pode provar o que uma
	// variável vai valer em tempo de execução.
	ehConstante := func(expr ast.Expr) bool {
		tv := info.Types[expr]
		return tv.Value != nil
	}
	valorDe := func(expr ast.Expr) string {
		tv := info.Types[expr]
		if tv.Value == nil {
			return ""
		}
		return constant.StringVal(tv.Value)
	}

	for _, file := range pass.Files {
		for _, imp := range file.Imports {
			path := strings.Trim(imp.Path.Value, `"`)
			if !isSubpacoteDeRede(path) {
				continue
			}
			pass.Reportf(imp.Pos(), "pacote de rede proibido: %s", path)
		}

		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			ident, ok := sel.X.(*ast.Ident)
			if !ok || !isPacoteNet(pass, ident) {
				return true
			}

			nome := sel.Sel.Name
			if nome != "Dial" && nome != "Listen" {
				pass.Reportf(call.Pos(), "chamada de rede proibida: net.%s — so net.Dial e net.Listen com rede \"unix\" sao permitidos", nome)
				return true
			}

			if len(call.Args) == 0 {
				pass.Reportf(call.Pos(), "net.%s sem argumento de rede", nome)
				return true
			}

			arg0 := call.Args[0]
			if !ehConstante(arg0) || valorDe(arg0) != "unix" {
				// arg0, ehConstante e valorDe sao referenciados de novo aqui
				// de proposito, e nao so na condicao acima: a prova de
				// mutacao desta tarefa troca a linha do "if" por "if false {"
				// sem tocar no corpo, e um corpo que nao usasse os tres iria
				// falhar a compilacao por variavel declarada e nao usada —
				// build quebrado nao e cobertura, e o mutate.ps1 sairia
				// inconclusivo em vez de provar a regra.
				pass.Reportf(arg0.Pos(), "rede proibida: net.%s so aceita a constante \"unix\" (constante=%v, valor=%q)", nome, ehConstante(arg0), valorDe(arg0))
			}
			return true
		})
	}
	return nil, nil
}

// isSubpacoteDeRede bane qualquer pacote net/* — net/http incluido. O
// pacote net em si (sem barra) NAO entra aqui: ele e permitido para
// net.Dial/net.Listen, e a chamada e checada em run, nao a importacao.
func isSubpacoteDeRede(path string) bool {
	return strings.HasPrefix(path, "net/")
}

// isPacoteNet confere, via informacao de tipos, que o identificador antes do
// ponto referencia o pacote "net" — nao uma variavel ou campo que por acaso
// se chame "net".
func isPacoteNet(pass *analysis.Pass, ident *ast.Ident) bool {
	obj := pass.TypesInfo.Uses[ident]
	pkgName, ok := obj.(*types.PkgName)
	if !ok {
		return false
	}
	return pkgName.Imported().Path() == "net"
}
