package mcpsrv_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/jonyd/gobsidian/internal/config"
	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/mcpsrv"
	"github.com/jonyd/gobsidian/internal/service"
	"github.com/jonyd/gobsidian/internal/vault"
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

	svc := service.New(v, idx, nil, service.Options{})
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
		wantTools := []string{"note_read", "note_list", "note_metadata", "link_graph", "tag_list"}
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
				// Verify errCode is in the message
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

	// Server continues responding
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

// Dummy WatchStats for testing
type dummyWatchStats struct{}

func (d dummyWatchStats) Stats() service.WatchCounters {
	return service.WatchCounters{
		Active:          true,
		EventsReceived:  10,
		EventsDropped:   2,
		EventsProcessed: 8,
		EventsSkipped:   1,
		Reconciliations: 0,
	}
}

func TestVaultStatsWithWatcher(t *testing.T) {
	root := t.TempDir()

	v, _ := vault.New(root)
	idx := index.New()
	idx.Build(context.Background(), v)

	// Inject dummy watcher
	svc := service.New(v, idx, dummyWatchStats{}, service.Options{})
	cfg := config.Defaults()
	srv := mcpsrv.New(context.Background(), svc, cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	go func() { _ = srv.Connect(ctx, serverTransport) }()

	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "0.0.1"}, nil)
	session, _ := client.Connect(ctx, clientTransport, nil)
	defer session.Close()

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
		json.Unmarshal(b, &out)

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
		json.Unmarshal(b, &out)

		if _, ok := out["watcher"]; ok {
			t.Fatal("watcher block should be missing when include_runtime is false")
		}
	})
}
