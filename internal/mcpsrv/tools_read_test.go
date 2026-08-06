package mcpsrv_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jonyd/gobsidian/internal/config"
	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/mcpsrv"
	"github.com/jonyd/gobsidian/internal/search"
	"github.com/jonyd/gobsidian/internal/service"
	"github.com/jonyd/gobsidian/internal/vault"
	"github.com/jonyd/gobsidian/internal/watcher"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func newTestServerWithIndex(t *testing.T, root string) *mcpsrv.Server {
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
	for _, p := range idx.NotePaths() {
		if data, err := v.ReadAll(context.Background(), p); err == nil {
			body, _ := vault.StripBOM(data)
			inv.Add(string(p), search.Analyze(string(body)))
		}
	}

	svc := service.New(v, idx, inv, nil, service.Options{})
	cfg := config.Defaults()

	return mcpsrv.New(context.Background(), svc, cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func TestReadTools(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "A.md", "---\ntags: [a]\n---\n# A\nText A\n[[B]]\n")
	writeFile(t, root, "B.md", "---\ntags: [b]\n---\n# B\nText B\n")

	srv := newTestServerWithIndex(t, root)

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

	t.Run("Tools registered", func(t *testing.T) {
		tools, err := session.ListTools(ctx, nil)
		if err != nil {
			t.Fatalf("ListTools: %v", err)
		}
		wantTools := []string{"vault_search", "note_read", "note_list", "note_metadata", "link_graph", "tag_list"}
		found := make(map[string]bool)
		for _, tool := range tools.Tools {
			found[tool.Name] = true
		}
		for _, wt := range wantTools {
			if !found[wt] {
				t.Errorf("Tool %s not found", wt)
			}
		}
	})

	tests := []struct {
		name    string
		tool    string
		args    map[string]interface{}
		wantErr bool
		errCode string
	}{
		{
			name: "vault_search valid",
			tool: "vault_search",
			args: map[string]interface{}{"query": "Text"},
		},
		{
			name: "note_read valid",
			tool: "note_read",
			args: map[string]interface{}{"path": "A.md"},
		},
		{
			name:    "note_read invalid",
			tool:    "note_read",
			args:    map[string]interface{}{"path": "Missing.md"},
			wantErr: true,
			errCode: "NOTE_NOT_FOUND",
		},
		{
			name: "note_list valid",
			tool: "note_list",
			args: map[string]interface{}{},
		},
		{
			name: "note_metadata valid",
			tool: "note_metadata",
			args: map[string]interface{}{"path": "A.md"},
		},
		{
			name: "link_graph valid",
			tool: "link_graph",
			args: map[string]interface{}{"path": "A.md"},
		},
		{
			name: "tag_list valid",
			tool: "tag_list",
			args: map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := json.Marshal(tt.args)
			if err != nil {
				t.Fatalf("json.Marshal(%+v): %v", tt.args, err)
			}
			var args map[string]any
			if err := json.Unmarshal(b, &args); err != nil {
				t.Fatalf("json.Unmarshal: %v", err)
			}

			res, err := session.CallTool(ctx, &mcp.CallToolParams{
				Name:      tt.tool,
				Arguments: args,
			})
			if err != nil {
				t.Fatalf("CallTool transport err: %v", err)
			}
			if tt.wantErr {
				if !res.IsError {
					t.Fatalf("Expected IsError=true, got false. content: %v", res.Content)
				}
				if res.StructuredContent != nil {
					t.Errorf("Expected nil StructuredContent with IsError, got %v", res.StructuredContent)
				}
				msg := ""
				if len(res.Content) > 0 {
					b, _ := json.Marshal(res.Content[0])
					var obj map[string]interface{}
					if err := json.Unmarshal(b, &obj); err == nil {
						if text, ok := obj["text"].(string); ok {
							msg = text
						}
					}
				}
				if !strings.HasPrefix(msg, tt.errCode) {
					t.Errorf("Expected error code %s prefix, got %q", tt.errCode, msg)
				}
			} else {
				if res.IsError {
					t.Fatalf("Expected IsError=false, got true. content: %v", res.Content)
				}
				if res.StructuredContent == nil {
					t.Errorf("Expected non-nil StructuredContent")
				}
			}
		})
	}

	t.Run("Server alive after error", func(t *testing.T) {
		_, err := session.ListTools(ctx, nil)
		if err != nil {
			t.Fatalf("Server died: %v", err)
		}
	})
}

func TestResources(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "A.md", "# A")

	srv := newTestServerWithIndex(t, root)

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

	res, err := session.ListResources(ctx, nil)
	if err != nil {
		t.Fatalf("ListResources: %v", err)
	}

	if len(res.Resources) == 0 {
		t.Fatalf("No resources listed")
	}

	found := false
	for _, r := range res.Resources {
		if r.URI == "gobsidian:///A.md" {
			found = true
			if r.MIMEType != "text/markdown" {
				t.Errorf("MimeType = %s, want text/markdown", r.MIMEType)
			}
			if r.Name != "A" && r.Name != "A.md" {
				t.Errorf("Name = %s, want A", r.Name)
			}
		}
	}
	if !found {
		t.Errorf("Resource gobsidian:///A.md not found in list")
	}

	readRes, err := session.ReadResource(ctx, &mcp.ReadResourceParams{URI: "gobsidian:///A.md"})
	if err != nil {
		t.Fatalf("ReadResource: %v", err)
	}

	if len(readRes.Contents) == 0 {
		t.Fatalf("ReadResource returned empty contents")
	}
	content := readRes.Contents[0]
	if content.MIMEType != "text/markdown" {
		t.Errorf("ReadResource MimeType = %s, want text/markdown", content.MIMEType)
	}
	if content.URI != "gobsidian:///A.md" {
		t.Errorf("ReadResource Uri = %s, want gobsidian:///A.md", content.URI)
	}
}

// firstText extrai o texto do primeiro bloco de Content de um resultado de
// tool. E o mesmo jeito que TestReadTools usa inline para ler o codigo de
// erro — extraido aqui porque os testes de note_read em lote precisam dele
// mais de uma vez.
func firstText(t *testing.T, res *mcp.CallToolResult) string {
	t.Helper()
	if res == nil || len(res.Content) == 0 {
		return ""
	}
	b, err := json.Marshal(res.Content[0])
	if err != nil {
		t.Fatalf("marshal Content[0]: %v", err)
	}
	var obj map[string]interface{}
	if err := json.Unmarshal(b, &obj); err != nil {
		t.Fatalf("unmarshal Content[0]: %v", err)
	}
	text, _ := obj["text"].(string)
	return text
}

func connectTestSession(ctx context.Context, t *testing.T, srv *mcpsrv.Server) *mcp.ClientSession {
	t.Helper()
	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	go func() { _ = srv.Connect(ctx, serverTransport) }()

	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "0.0.1"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })
	return session
}

// TestNoteReadPathAndPathsMutuallyExclusive prova a decisao 1 do brief da
// Task 84: path e paths preenchidos ao mesmo tempo e erro de validacao, e
// NAO uma precedencia silenciosa que decidiria pelo cliente qual dos dois
// vale.
func TestNoteReadPathAndPathsMutuallyExclusive(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "A.md", "# A\n")
	srv := newTestServerWithIndex(t, root)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	session := connectTestSession(ctx, t, srv)

	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "note_read",
		Arguments: map[string]any{"path": "A.md", "paths": []string{"A.md"}},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !res.IsError {
		t.Fatalf("path e paths juntos deveria ser erro de validacao")
	}
	msg := firstText(t, res)
	if !strings.HasPrefix(msg, "INVALID_ARGUMENT") {
		t.Errorf("mensagem = %q, quer prefixo INVALID_ARGUMENT", msg)
	}
	if res.StructuredContent == nil {
		t.Errorf("StructuredContent nulo — erro de validacao de lote precisa devolver Out preenchido")
	}
}

// TestNoteReadRecusaLoteAcimaDoTeto e a prova de mutacao da decisao 4: mais
// de 50 caminhos em paths e recusado. A mutacao do brief troca a condicao do
// teto por "if false" — sem o teto, 51 caminhos passariam e o teste
// (que exige IsError com prefixo INVALID_ARGUMENT) reprovaria.
func TestNoteReadRecusaLoteAcimaDoTeto(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "A.md", "# A\n")
	srv := newTestServerWithIndex(t, root)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	session := connectTestSession(ctx, t, srv)

	paths := make([]string, 51)
	for i := range paths {
		paths[i] = "A.md"
	}

	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "note_read",
		Arguments: map[string]any{"paths": paths},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !res.IsError {
		t.Fatalf("51 paths deveria ser erro; IsError=false")
	}
	msg := firstText(t, res)
	if !strings.HasPrefix(msg, "INVALID_ARGUMENT") {
		t.Errorf("mensagem = %q, quer prefixo INVALID_ARGUMENT", msg)
	}
	if res.StructuredContent == nil {
		t.Errorf("StructuredContent nulo — erro de validacao de lote precisa devolver Out preenchido")
	}
}

// TestNoteReadBatchKeepsFailedItemAtPosition e o teste de ponta a ponta que
// o brief exige: dez caminhos, um deles inexistente — os nove voltam, o
// decimo volta NA POSICAO CERTA com erro, sem derrubar o lote inteiro.
func TestNoteReadBatchKeepsFailedItemAtPosition(t *testing.T) {
	root := t.TempDir()
	var paths []string
	for i := 0; i < 10; i++ {
		name := fmt.Sprintf("N%02d.md", i)
		paths = append(paths, name)
		if i == 5 {
			continue // N05.md nunca e criada -- o item que tem de falhar
		}
		writeFile(t, root, name, fmt.Sprintf("# Nota %02d\n", i))
	}

	srv := newTestServerWithIndex(t, root)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	session := connectTestSession(ctx, t, srv)

	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "note_read",
		Arguments: map[string]any{"paths": paths},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if res.IsError {
		t.Fatalf("lote com uma falha nao pode derrubar o resultado inteiro: %v", res.Content)
	}

	b, err := json.Marshal(res.StructuredContent)
	if err != nil {
		t.Fatalf("marshal StructuredContent: %v", err)
	}
	var out struct {
		Items []struct {
			Path    string `json:"path"`
			Content string `json:"content"`
			Error   *struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		} `json:"items"`
	}
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal StructuredContent: %v", err)
	}

	if len(out.Items) != len(paths) {
		t.Fatalf("len(items) = %d, quer %d — item sumiu em vez de aparecer com erro", len(out.Items), len(paths))
	}
	for i, item := range out.Items {
		if item.Path != paths[i] {
			t.Errorf("items[%d].path = %q, quer %q — posicao nao preservada", i, item.Path, paths[i])
		}
		if i == 5 {
			if item.Error == nil {
				t.Errorf("items[5] deveria ter error (N05.md nao existe)")
			} else if item.Error.Code != "NOTE_NOT_FOUND" {
				t.Errorf("items[5].error.code = %q, quer NOTE_NOT_FOUND", item.Error.Code)
			}
		} else if item.Error != nil {
			t.Errorf("items[%d] nao deveria ter error: %+v", i, item.Error)
		}
	}
}

type dummyWatchStats struct{}

func (d dummyWatchStats) Stats() service.WatchCounters {
	return service.WatchCounters{
		Active:          true,
		EventsReceived:  10,
		EventsDropped:   2,
		DroppedByReason: map[string]int64{"chmod": 0, "outside_vault": 0, "excluded": 2, "unknown_op": 0},
		EventsProcessed: 8,
		EventsSkipped:   1,
		Reconciliations: 0,
	}
}

func TestVaultStatsWithWatcher(t *testing.T) {
	root := t.TempDir()

	v, _ := vault.New(root)
	idx := index.New()
	_ = idx.Build(context.Background(), v)

	svc := service.New(v, idx, nil, dummyWatchStats{}, service.Options{})
	cfg := config.Defaults()
	srv := mcpsrv.New(context.Background(), svc, cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	go func() { _ = srv.Connect(ctx, serverTransport) }()

	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "0.0.1"}, nil)
	session, _ := client.Connect(ctx, clientTransport, nil)
	defer func() { _ = session.Close() }()

	t.Run("include_runtime=true", func(t *testing.T) {
		res, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name:      "vault_stats",
			Arguments: map[string]any{"include_runtime": true},
		})
		if err != nil {
			t.Fatalf("CallTool: %v", err)
		}

		b, _ := json.Marshal(res.StructuredContent)
		var out map[string]interface{}
		_ = json.Unmarshal(b, &out)

		watcherObj, ok := out["watcher"].(map[string]interface{})
		if !ok {
			t.Fatal("watcher block missing or not an object in vault_stats output")
		}

		if watcherObj["active"] != true {
			t.Errorf("watcher.active = %v, want true", watcherObj["active"])
		}
		if watcherObj["events_received"].(float64) != 10 {
			t.Errorf("watcher.events_received = %v, want 10", watcherObj["events_received"])
		}
		if watcherObj["events_dropped"].(float64) != 2 {
			t.Errorf("watcher.events_dropped = %v, want 2", watcherObj["events_dropped"])
		}
		if watcherObj["events_processed"].(float64) != 8 {
			t.Errorf("watcher.events_processed = %v, want 8", watcherObj["events_processed"])
		}
		if watcherObj["events_skipped"].(float64) != 1 {
			t.Errorf("watcher.events_skipped = %v, want 1", watcherObj["events_skipped"])
		}
		if _, ok := watcherObj["reconciliations"]; !ok {
			t.Errorf("watcher.reconciliations missing")
		}
	})

	t.Run("include_runtime=false", func(t *testing.T) {
		res, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name:      "vault_stats",
			Arguments: map[string]any{},
		})
		if err != nil {
			t.Fatalf("CallTool: %v", err)
		}

		b, _ := json.Marshal(res.StructuredContent)
		var out map[string]interface{}
		_ = json.Unmarshal(b, &out)

		if _, ok := out["watcher"]; ok {
			t.Fatal("watcher block should be missing when include_runtime is false")
		}
	})
}

func TestVaultStatsReflectsWatcherUpdate(t *testing.T) {
	root := t.TempDir()
	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}

	idx := index.New()
	if err := idx.Build(context.Background(), v); err != nil {
		t.Fatalf("idx.Build: %v", err)
	}

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	w, err := watcher.New(v, idx, nil, 10*time.Millisecond, log)
	if err != nil {
		t.Fatalf("watcher.New: %v", err)
	}

	runCtx, runCancel := context.WithCancel(context.Background())
	defer runCancel()
	defer func() { _ = w.Close() }()

	go func() {
		_ = w.Run(runCtx)
	}()

	svc := service.New(v, idx, nil, nil, service.Options{ReadOnly: true})
	cfg := config.Defaults()
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

	getNoteCount := func() float64 {
		res, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name: "vault_stats",
		})
		if err != nil {
			t.Fatalf("CallTool vault_stats: %v", err)
		}
		b, _ := json.Marshal(res.StructuredContent)
		var out map[string]interface{}
		_ = json.Unmarshal(b, &out)
		return out["notes"].(float64)
	}

	initialCount := getNoteCount()
	if initialCount != 0 {
		t.Fatalf("initial notes count = %v, want 0", initialCount)
	}

	notePath := filepath.Join(root, "dynamic_note.md")
	if err := os.WriteFile(notePath, []byte("# Dynamic Note\n"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	deadline := time.Now().Add(3 * time.Second)
	foundAdd := false
	startAdd := time.Now()
	for time.Now().Before(deadline) {
		if getNoteCount() == 1 {
			foundAdd = true
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !foundAdd {
		t.Fatalf("vault_stats notes count did not increase to 1 after creating note (took %v)", time.Since(startAdd))
	}

	if err := os.Remove(notePath); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	deadline = time.Now().Add(3 * time.Second)
	foundRemove := false
	startRemove := time.Now()
	for time.Now().Before(deadline) {
		if getNoteCount() == 0 {
			foundRemove = true
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !foundRemove {
		t.Fatalf("vault_stats notes count did not return to 0 after removing note (took %v)", time.Since(startRemove))
	}
}
