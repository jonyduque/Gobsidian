package mcpsrv_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"path/filepath"
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

func setupMCPServer(t *testing.T) (*mcp.ClientSession, func()) {
	t.Helper()
	dir := t.TempDir()

	files := map[string]string{
		"note_a.md": "---\ntitle: Note A\ntags: [tagA]\n---\n# Heading A\n\nLink to [[note_b]] and embed ![[note_c]] and broken [[missing_note]].\n",
		"note_b.md": "---\ntitle: Note B\n---\nBacklink to [[note_a]].\n",
		"note_c.md": "---\ntitle: Note C\n---\nTarget of embed.\n",
	}

	for rel, content := range files {
		full := filepath.Join(dir, rel)
		if err := os.WriteFile(full, []byte(content), 0644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
	}

	v, err := vault.New(dir)
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
	srv := mcpsrv.New(context.Background(), svc, cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	go func() { _ = srv.Connect(ctx, serverTransport) }()

	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "0.0.1"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		cancel()
		t.Fatalf("Connect: %v", err)
	}

	cleanup := func() {
		_ = session.Close()
		cancel()
	}

	return session, cleanup
}

func TestNoteMetadata_IncludeParameter(t *testing.T) {
	session, cleanup := setupMCPServer(t)
	defer cleanup()

	ctx := context.Background()

	// Default behavior (include omitted)
	resDefault, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "note_metadata",
		Arguments: map[string]any{
			"path": "note_a.md",
		},
	})
	if err != nil || resDefault.IsError {
		t.Fatalf("CallTool default err: %v, content: %v", err, resDefault.Content)
	}
	var dataDefault struct {
		Tags     []string `json:"tags"`
		Headings []string `json:"headings"`
		Links    []any    `json:"links"`
	}
	bDefault, _ := json.Marshal(resDefault.StructuredContent)
	_ = json.Unmarshal(bDefault, &dataDefault)

	if len(dataDefault.Tags) == 0 {
		t.Error("esperava tags no retorno default")
	}
	if len(dataDefault.Headings) == 0 {
		t.Error("esperava headings no retorno default")
	}

	// Non-default behavior (include = ["tags"])
	resTags, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "note_metadata",
		Arguments: map[string]any{
			"path":    "note_a.md",
			"include": []string{"tags"},
		},
	})
	if err != nil || resTags.IsError {
		t.Fatalf("CallTool include tags err: %v, content: %v", err, resTags.Content)
	}
	var dataTags struct {
		Tags        []string       `json:"tags"`
		Headings    []string       `json:"headings"`
		Links       []any          `json:"links"`
		Frontmatter map[string]any `json:"frontmatter"`
	}
	bTags, _ := json.Marshal(resTags.StructuredContent)
	_ = json.Unmarshal(bTags, &dataTags)

	if len(dataTags.Tags) == 0 {
		t.Error("esperava tags quando include=['tags']")
	}
	if len(dataTags.Headings) != 0 {
		t.Errorf("headings deveria ser vazio quando include=['tags'], obteve: %v", dataTags.Headings)
	}
	if len(dataTags.Links) != 0 {
		t.Errorf("links deveria ser vazio quando include=['tags'], obteve: %v", dataTags.Links)
	}
	if len(dataTags.Frontmatter) != 0 {
		t.Errorf("frontmatter deveria ser vazio quando include=['tags'], obteve: %v", dataTags.Frontmatter)
	}
}

func TestLinkGraph_DirectionParameter(t *testing.T) {
	session, cleanup := setupMCPServer(t)
	defer cleanup()

	ctx := context.Background()

	// Non-default behavior (direction = "outgoing")
	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "link_graph",
		Arguments: map[string]any{
			"path":      "note_a.md",
			"direction": "outgoing",
		},
	})
	if err != nil || res.IsError {
		t.Fatalf("CallTool direction outgoing err: %v, content: %v", err, res.Content)
	}

	var data struct {
		Edges []struct {
			Source string `json:"source"`
			Target string `json:"target"`
		} `json:"edges"`
	}
	b, _ := json.Marshal(res.StructuredContent)
	_ = json.Unmarshal(b, &data)

	for _, e := range data.Edges {
		if e.Target == "note_a.md" {
			t.Errorf("nao esperava incoming edge quando direction='outgoing', encontrou: %+v", e)
		}
	}
}

func TestLinkGraph_IncludeBrokenParameter(t *testing.T) {
	session, cleanup := setupMCPServer(t)
	defer cleanup()

	ctx := context.Background()

	// Non-default behavior (include_broken = false)
	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "link_graph",
		Arguments: map[string]any{
			"path":           "note_a.md",
			"include_broken": false,
		},
	})
	if err != nil || res.IsError {
		t.Fatalf("CallTool include_broken=false err: %v, content: %v", err, res.Content)
	}

	var data struct {
		Edges []struct {
			Target string `json:"target"`
		} `json:"edges"`
	}
	b, _ := json.Marshal(res.StructuredContent)
	_ = json.Unmarshal(b, &data)

	for _, e := range data.Edges {
		if e.Target == "missing_note.md" || e.Target == "missing_note" {
			t.Errorf("nao esperava aresta quebrada quando include_broken=false, obteve: %+v", e)
		}
	}
}

func TestLinkGraph_IncludeEmbedsParameter(t *testing.T) {
	session, cleanup := setupMCPServer(t)
	defer cleanup()

	ctx := context.Background()

	// Non-default behavior (include_embeds = false)
	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "link_graph",
		Arguments: map[string]any{
			"path":           "note_a.md",
			"include_embeds": false,
		},
	})
	if err != nil || res.IsError {
		t.Fatalf("CallTool include_embeds=false err: %v, content: %v", err, res.Content)
	}

	var data struct {
		Edges []struct {
			Target string `json:"target"`
		} `json:"edges"`
	}
	b, _ := json.Marshal(res.StructuredContent)
	_ = json.Unmarshal(b, &data)

	for _, e := range data.Edges {
		if e.Target == "note_c.md" {
			t.Errorf("nao esperava aresta de embed quando include_embeds=false, obteve: %+v", e)
		}
	}
}
