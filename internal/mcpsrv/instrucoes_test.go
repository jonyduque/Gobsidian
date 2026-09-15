package mcpsrv_test

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jonyduque/Gobsidian/internal/config"
	"github.com/jonyduque/Gobsidian/internal/mcpsrv"
	"github.com/jonyduque/Gobsidian/internal/service"
	"github.com/jonyduque/Gobsidian/internal/vault"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// instrucoesDoInitialize sobe o servidor com o cofre e o modo pedidos e devolve
// o campo instructions do resultado do initialize, como o host o recebe.
func instrucoesDoInitialize(t *testing.T, root string, somenteLeitura bool) string {
	t.Helper()
	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	cfg := config.Defaults()
	cfg.VaultPath = root
	cfg.ReadOnly = somenteLeitura
	srv := mcpsrv.New(context.Background(), service.New(v, nil, nil, nil, service.Options{}), cfg,
		slog.New(slog.NewTextHandler(io.Discard, nil)))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	go func() { _ = srv.Connect(ctx, serverTransport) }()

	session, err := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "0.0.1"}, nil).Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })
	return session.InitializeResult().Instructions
}

// TestInitializeLevaAsInstrucoesDoServidor: ate 2026-09-15 o servidor mandava
// instructions vazio, e o usuario colava docs/PROMPT.md a mao em cada host. Sem
// o texto, o modelo nao sabe os conjuntos fechados -- o schema servido nao
// carrega enum (TOOLS.md, "Schemas servidos") -- nem que expected_hash e a
// unica defesa contra sobrescrever o que o Obsidian acabou de gravar.
func TestInitializeLevaAsInstrucoesDoServidor(t *testing.T) {
	root := filepath.Join(t.TempDir(), "Estudo")
	writeFile(t, root, "A.md", "# A\n")

	got := instrucoesDoInitialize(t, root, false)

	for _, trecho := range []string{
		"Cofre: Estudo\n",
		"- note_list.sort: path, modified, size, title\n",
		"expected_hash",
		"INVALID_ARGUMENT",
	} {
		if !strings.Contains(got, trecho) {
			t.Errorf("instructions sem %q:\n%s", trecho, got)
		}
	}
	if strings.Contains(got, "\r") {
		t.Error("instructions com CR: o arquivo embutido veio com CRLF do checkout e o texto nao foi normalizado")
	}
	if strings.Contains(got, "SOMENTE LEITURA") {
		t.Errorf("cofre com escrita recebeu o aviso de somente-leitura:\n%s", got)
	}
}

// TestInstrucoesDizemSomenteLeituraSoQuandoE: a frase fixa antiga ("o cofre
// pode estar em modo somente-leitura") valia para todo cofre e nao dizia nada
// sobre este. Montada por sessao, ela so aparece quando e verdade.
func TestInstrucoesDizemSomenteLeituraSoQuandoE(t *testing.T) {
	root := filepath.Join(t.TempDir(), "Oral")
	writeFile(t, root, "A.md", "# A\n")

	got := instrucoesDoInitialize(t, root, true)
	if !strings.Contains(got, "SOMENTE LEITURA") {
		t.Fatalf("cofre somente-leitura sem o aviso:\n%s", got)
	}
	if strings.Index(got, "SOMENTE LEITURA") > strings.Index(got, "CAMINHOS") {
		t.Errorf("o aviso de somente-leitura precisa vir antes das regras de uso:\n%s", got)
	}
}
