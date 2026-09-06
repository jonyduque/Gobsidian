package mcpsrv

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// MaxPathsPorLote expoe o teto de note_read para os testes de fora do pacote.
// O par de testes do teto (50 aceito, 51 recusado) tem de ser escrito a partir
// da MESMA constante que o produto compara: um literal 50 no teste continua
// verde se alguem mudar o valor do produto, e o par deixa de prender o limite.
// Arquivo _test.go: nada disso existe no binario.
const MaxPathsPorLote = maxPathsPorLote

// RegisterPanicProbeForTest registra uma tool que sempre entra em panic.
// Existe para provar que RNF-13 vale — nao e registrada em producao.
func (s *Server) RegisterPanicProbeForTest() {
	type empty struct{}
	mcp.AddTool(s.mcp,
		&mcp.Tool{Name: "panic_probe", Description: "sonda de teste; entra em panic"},
		guard(s.log, "panic_probe",
			func(context.Context, *mcp.CallToolRequest, empty) (*mcp.CallToolResult, empty, error) {
				panic("sonda")
			}),
	)
}
