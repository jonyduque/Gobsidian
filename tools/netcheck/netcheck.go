// Package netcheck implementa a verificação de RNF-30: nenhum socket do
// produto sai da máquina. Ver PRD §6.4 para a formulação completa da
// garantia, reaberta DUAS vezes — em 2026-08-05 para o IPC local via socket
// Unix, e em 2026-09-08 para o `gobsidian update` (decisão D-13 do dono).
//
// A segunda exceção é estreita, e as três coisas que a estreitam são
// verificadas aqui: `net/http` só pode ser importado por
// `internal/selfupdate`; dentro dele, toda URL escrita no código tem de
// apontar para um host da lista abaixo; e nenhum outro pacote ganha nada.
// O que este analisador NÃO consegue provar — uma URL montada em tempo de
// execução — é recusado por uma guarda no próprio pacote, com teste que a
// exercita: análise estática e guarda de runtime cobrem metades diferentes,
// e dizer que uma cobre a outra seria mentira.
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
	"go/token"
	"go/types"
	"strconv"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// pacoteDaExcecao e o UNICO pacote do produto que pode importar net/http.
// Sufixo, e nao caminho completo, porque o mesmo analisador roda sobre o
// modulo real e sobre as fixtures do gate (scripts/testdata/gates/netcheck).
const pacoteDaExcecao = "internal/selfupdate"

// hostsPermitidos sao os hosts que `gobsidian update` pode alcancar. Lista
// fechada: um host novo exige tocar aqui, e tocar aqui exige explicar por que
// no PRD. Sem ela, a excecao seria "internal/selfupdate pode tudo".
//
// internal/selfupdate carrega a MESMA lista, cobrada em tempo de execucao
// sobre URL que chega da API. As duas existem porque cobrem metades
// diferentes, e um teste daquele pacote prova que elas nao divergiram.
var hostsPermitidos = []string{
	"https://api.github.com",
	"https://objects.githubusercontent.com",
	"https://github.com",
}

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
			if path == "net/http" && ehPacoteDaExcecao(pass) {
				continue
			}
			pass.Reportf(imp.Pos(), "pacote de rede proibido: %s", path)
		}

		// Dentro da excecao, toda URL escrita no codigo tem de apontar para um
		// host da lista. Fora dela este laco nao encontra nada, porque nenhum
		// outro pacote chega a importar net/http.
		//
		// Arquivo _test.go fica de fora, e a razao nao e conveniencia: o teste
		// que prova que um host proibido E RECUSADO precisa escrever um host
		// proibido. Cobrar a regra ali tornaria impossivel provar a regra --
		// e a garantia da RNF-30 e sobre o binario ENTREGUE, que nao contem
		// arquivo de teste. A metade dinamica continua coberta por
		// selfupdate.ValidarHost, com teste.
		//
		// A regra de IMPORTACAO, acima, continua valendo tambem em teste: os
		// _test.go deste pacote nao importam net, e ha um comentario la
		// dizendo por que.
		if ehPacoteDaExcecao(pass) && !ehArquivoDeTeste(pass, file) {
			for _, lit := range literaisDeURL(file) {
				if hostPermitido(lit.valor) {
					continue
				}
				pass.Reportf(lit.pos, "host proibido em %s: %q -- so %s sao permitidos (PRD 6.4)",
					pacoteDaExcecao, lit.valor, strings.Join(hostsPermitidos, ", "))
			}
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

// ehPacoteDaExcecao diz se o pacote sob analise e internal/selfupdate.
func ehPacoteDaExcecao(pass *analysis.Pass) bool {
	caminho := pass.Pkg.Path()
	return caminho == pacoteDaExcecao || strings.HasSuffix(caminho, "/"+pacoteDaExcecao) ||
		strings.HasSuffix(caminho, "/selfupdate")
}

// literalDeURL e um literal de string do arquivo que contem um esquema.
type literalDeURL struct {
	valor string
	pos   token.Pos
}

// literaisDeURL colhe todo literal de string com "://".
//
// Literal, e nao expressao constante: o que se quer impedir e alguem ESCREVER
// um host novo no codigo. Concatenacao de constantes (hostDaAPI + "/caminho")
// e vista pelos pedacos, que e o suficiente -- o pedaco com o esquema e o que
// carrega o host.
func literaisDeURL(file *ast.File) []literalDeURL {
	var achados []literalDeURL
	ast.Inspect(file, func(n ast.Node) bool {
		lit, ok := n.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return true
		}
		valor, err := strconv.Unquote(lit.Value)
		if err != nil || !strings.Contains(valor, "://") {
			return true
		}
		achados = append(achados, literalDeURL{valor: valor, pos: lit.Pos()})
		return true
	})
	return achados
}

// hostPermitido confere o PREFIXO: a lista guarda esquema e host, e o caminho
// depois deles nao muda para onde a conexao vai.
func hostPermitido(url string) bool {
	for _, h := range hostsPermitidos {
		if url == h || strings.HasPrefix(url, h+"/") {
			return true
		}
	}
	return false
}

// ehArquivoDeTeste diz se o arquivo termina em _test.go.
func ehArquivoDeTeste(pass *analysis.Pass, file *ast.File) bool {
	nome := pass.Fset.Position(file.Pos()).Filename
	return strings.HasSuffix(nome, "_test.go")
}
