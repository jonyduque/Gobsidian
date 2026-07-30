package mcpsrv_test

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/jonyd/gobsidian/internal/config"
	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/mcpsrv"
	"github.com/jonyd/gobsidian/internal/search"
	"github.com/jonyd/gobsidian/internal/service"
	"github.com/jonyd/gobsidian/internal/vault"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func newTestServerWithConfig(t *testing.T, root string, cfg config.Config) *mcpsrv.Server {
	t.Helper()
	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	idx := index.New()
	if err := idx.Build(context.Background(), v); err != nil {
		t.Fatalf("idx.Build: %v", err)
	}
	inv := search.NewInverted()
	svc := service.New(v, idx, inv, nil, service.Options{ReadOnly: cfg.ReadOnly})
	return mcpsrv.New(context.Background(), svc, cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func TestListTools_ReadOnlyTrue(t *testing.T) {
	cfg := config.Defaults()
	cfg.ReadOnly = true
	srv := newTestServerWithConfig(t, t.TempDir(), cfg)

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

	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}

	toolNames := make(map[string]bool)
	for _, tool := range tools.Tools {
		toolNames[tool.Name] = true
	}

	// Tools de escrita devem estar ausentes
	writeTools := []string{"note_create", "note_append", "note_patch"}
	for _, wt := range writeTools {
		if toolNames[wt] {
			t.Errorf("tool de escrita %q NAO deveria estar presente sob --read-only", wt)
		}
	}

	// Tools de leitura devem continuar presentes
	readTools := []string{"vault_stats", "note_read", "vault_search"}
	for _, rt := range readTools {
		if !toolNames[rt] {
			t.Errorf("tool de leitura %q deveria estar presente", rt)
		}
	}
}

func TestListTools_ReadOnlyFalse(t *testing.T) {
	cfg := config.Defaults()
	cfg.ReadOnly = false
	srv := newTestServerWithConfig(t, t.TempDir(), cfg)

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

	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}

	toolNames := make(map[string]bool)
	for _, tool := range tools.Tools {
		toolNames[tool.Name] = true
	}

	// Tools de escrita devem estar presentes quando ReadOnly = false
	writeTools := []string{"note_create", "note_append", "note_patch"}
	for _, wt := range writeTools {
		if !toolNames[wt] {
			t.Errorf("tool de escrita %q deveria estar presente sem --read-only", wt)
		}
	}
}

func TestWatcherActiveUnderReadOnly(t *testing.T) {
	root := t.TempDir()
	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	idx := index.New()
	_ = idx.Build(context.Background(), v)
	inv := search.NewInverted()

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg := config.Defaults()
	cfg.ReadOnly = true
	svc := service.New(v, idx, inv, dummyWatchStats{}, service.Options{ReadOnly: true})
	srv := mcpsrv.New(context.Background(), svc, cfg, log)

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

	out, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "vault_stats",
		Arguments: map[string]any{"include_runtime": true},
	})
	if err != nil {
		t.Fatalf("CallTool vault_stats: %v", err)
	}
	if out.IsError {
		t.Fatalf("CallTool devolveu erro: %v", out)
	}
}
