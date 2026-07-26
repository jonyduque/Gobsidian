package mcpsrv

import (
	"errors"
	"testing"

	"github.com/jonyd/gobsidian/internal/service"
)

// TestToolErrPreservesUnwrap trava a regressao do achado 2: toolErr precisa
// envolver o erro de dominio com %w, nao apenas formatar seu texto com %s. Sem
// isso, errors.Is/errors.As nao alcancam o *service.Error do outro lado da
// fronteira MCP — a unica travessia do dominio para o transporte.
func TestToolErrPreservesUnwrap(t *testing.T) {
	domainErr := service.Errorf(service.CodeNoteNotFound, "nota nao encontrada: %s", "foo.md")

	wrapped := toolErr(domainErr)

	if !errors.Is(wrapped, domainErr) {
		t.Fatal("errors.Is nao encontrou o *service.Error original atraves de toolErr; %w foi perdido")
	}

	var asErr *service.Error
	if !errors.As(wrapped, &asErr) {
		t.Fatal("errors.As nao encontrou o *service.Error original atraves de toolErr")
	}
	if asErr.Code != service.CodeNoteNotFound {
		t.Errorf("Code = %q, want %q", asErr.Code, service.CodeNoteNotFound)
	}
}
