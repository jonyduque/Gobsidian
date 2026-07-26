package mcpsrv

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// guard embrulha um handler de tool de modo que um panic vire resultado de
// erro, com stack trace em stderr. RNF-13: falha de uma tool jamais derruba
// o servidor. E o que distingue um servidor robusto de um que exige reiniciar
// o Claude Desktop toda vez que um caminho invalido e passado.
func guard[In, Out any](
	log *slog.Logger,
	name string,
	fn func(context.Context, *mcp.CallToolRequest, In) (*mcp.CallToolResult, Out, error),
) func(context.Context, *mcp.CallToolRequest, In) (*mcp.CallToolResult, Out, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, in In) (res *mcp.CallToolResult, out Out, err error) {
		defer func() {
			if r := recover(); r != nil {
				log.Error("panic em handler de tool",
					"tool", name,
					"panic", fmt.Sprint(r),
					"stack", string(debug.Stack()))
				res = nil
				out = *new(Out)
				err = fmt.Errorf("%s: falha interna em %s; detalhes registrados em stderr", codeInternal, name)
			}
		}()
		return fn(ctx, req, in)
	}
}
