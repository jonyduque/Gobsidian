package mcpsrv

import (
	"github.com/jonyd/gobsidian/internal/service"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const codeInternal = service.CodeInternal

// errorResult monta o resultado de erro no formato que o host entende, com
// codigo legivel por maquina no inicio da mensagem.
func errorResult(code, message string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{
			&mcp.TextContent{Text: code + ": " + message},
		},
	}
}

// toolError traduz um erro de dominio em resultado MCP. Erros nunca sobem
// como erro de protocolo: o cliente precisa poder ler a mensagem e se corrigir.
func toolError(err error) *mcp.CallToolResult {
	return errorResult(string(service.CodeOf(err)), err.Error())
}
