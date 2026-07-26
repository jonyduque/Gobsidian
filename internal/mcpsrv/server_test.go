package mcpsrv_test

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jonyd/gobsidian/internal/config"
	"github.com/jonyd/gobsidian/internal/mcpsrv"
	"github.com/jonyd/gobsidian/internal/service"
	"github.com/jonyd/gobsidian/internal/vault"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

func newTestServer(t *testing.T, root string) *mcpsrv.Server {
	t.Helper()

	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	svc := service.New(v, nil, service.Options{})
	cfg := config.Defaults()

	return mcpsrv.New(svc, cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func TestServerAnswersInitializeAndListsTools(t *testing.T) {
	srv := newTestServer(t, t.TempDir())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Transporte em memoria: os dois lados falam JSON-RPC sem tocar o disco
	// nem o processo. E o mesmo caminho de codigo do stdio.
	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	go func() { _ = srv.Connect(ctx, serverTransport) }()

	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "0.0.1"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer func() { _ = session.Close() }()

	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}

	found := false
	for _, tool := range tools.Tools {
		if tool.Name == "vault_stats" {
			found = true
		}
	}
	if !found {
		t.Fatal("vault_stats nao esta na lista de tools")
	}
}

func TestVaultStatsCountsNotes(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "A.md", "# A\n")
	writeFile(t, root, "sub/B.md", "# B\n")

	srv := newTestServer(t, root)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	go func() { _ = srv.Connect(ctx, serverTransport) }()

	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "0.0.1"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer func() { _ = session.Close() }()

	res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "vault_stats"})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if res.IsError {
		t.Fatalf("vault_stats devolveu erro: %+v", res.Content)
	}
}

// Uma tool que entra em panic nao pode derrubar o servidor (RNF-13).
func TestPanicInHandlerBecomesToolError(t *testing.T) {
	srv := newTestServer(t, t.TempDir())
	srv.RegisterPanicProbeForTest()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	go func() { _ = srv.Connect(ctx, serverTransport) }()

	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "0.0.1"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer func() { _ = session.Close() }()

	res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "panic_probe"})
	if err != nil {
		t.Fatalf("CallTool devolveu erro de transporte, quer erro de tool: %v", err)
	}
	if !res.IsError {
		t.Fatal("panic deveria virar resultado de erro de tool")
	}

	// O servidor continua respondendo depois do panic.
	if _, err := session.ListTools(ctx, nil); err != nil {
		t.Fatalf("servidor caiu apos panic em handler: %v", err)
	}
}
