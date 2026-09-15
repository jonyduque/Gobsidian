package doctor

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/jonyduque/Gobsidian/internal/config"
	"github.com/jonyduque/Gobsidian/internal/ipc"
)

func TestCheckDiretorioDeSocketsPassaOndeOSocketConecta(t *testing.T) {
	r := checkDiretorioDeSockets(context.Background(), config.Config{VaultPath: t.TempDir()})
	if r.Status != StatusOK {
		t.Fatalf("status = %v, detalhe %q; esperado OK num diretorio de runtime utilizavel", r.Status, r.Detail)
	}
	// Passar na shell nao prova que passa no host: a linha tem de dizer isso.
	if !strings.Contains(r.Detail, "neste processo") {
		t.Fatalf("detalhe %q nao avisa que a sonda vale so para este processo", r.Detail)
	}
}

func TestCheckDiretorioDeSocketsAvisaOndeOSocketNaoConecta(t *testing.T) {
	original := sondarSocketDoCofre
	t.Cleanup(func() { sondarSocketDoCofre = original })
	sondarSocketDoCofre = func(string) error {
		return fmt.Errorf("%w: simulado", ipc.ErrDiretorioSemSocket)
	}

	r := checkDiretorioDeSockets(context.Background(), config.Config{VaultPath: t.TempDir()})
	if r.Status != StatusWarn {
		t.Fatalf("status = %v, esperado Warn", r.Status)
	}
	if !strings.Contains(r.Detail, "em processo") {
		t.Fatalf("detalhe %q nao diz a consequencia (a ponte serve em processo)", r.Detail)
	}
}
