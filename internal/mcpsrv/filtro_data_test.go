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

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jonyd/gobsidian/internal/config"
	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/mcpsrv"
	"github.com/jonyd/gobsidian/internal/search"
	"github.com/jonyd/gobsidian/internal/service"
	"github.com/jonyd/gobsidian/internal/vault"
)

func sessaoComBusca(t *testing.T, root string) (*mcp.ClientSession, context.Context) {
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
		data, err := v.ReadAll(context.Background(), p)
		if err != nil {
			t.Fatalf("ReadAll %s: %v", p, err)
		}
		body, _ := vault.StripBOM(data)
		inv.Add(string(p), search.Analyze(string(body)))
	}
	svc := service.New(v, idx, inv, nil, service.Options{})
	srv := mcpsrv.New(context.Background(), svc, config.Defaults(),
		slog.New(slog.NewTextHandler(io.Discard, nil)))

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	t.Cleanup(cancel)
	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	go func() { _ = srv.Connect(ctx, serverTransport) }()
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "0.0.1"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })
	return session, ctx
}

func TestFiltroDeDataInvalidoNaoViraSilencio(t *testing.T) {
	root := t.TempDir()
	for _, nome := range []string{"a.md", "b.md", "c.md"} {
		if err := os.WriteFile(filepath.Join(root, nome),
			[]byte("# Nota\n\npalavra comum aqui\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	session, ctx := sessaoComBusca(t, root)

	buscar := func(t *testing.T, extra map[string]any) (int, bool) {
		t.Helper()
		args := map[string]any{"query": "comum"}
		for k, v := range extra {
			args[k] = v
		}
		res, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name: "vault_search", Arguments: args,
		})
		if err != nil {
			t.Fatalf("CallTool: %v", err)
		}
		if res.IsError {
			return -1, true
		}
		var out struct {
			Hits []struct {
				Path string `json:"path"`
			} `json:"hits"`
			Results []struct {
				Path string `json:"path"`
			} `json:"results"`
		}
		bruto, err := json.Marshal(res.StructuredContent)
		if err != nil {
			t.Fatalf("resposta nao serializa: %v", err)
		}
		if err := json.Unmarshal(bruto, &out); err != nil {
			t.Fatalf("resposta ilegivel: %v\n%s", err, bruto)
		}
		n := len(out.Hits)
		if n == 0 && len(out.Results) > 0 {
			n = len(out.Results)
		}
		return n, false
	}

	if n, _ := buscar(t, nil); n != 3 {
		t.Fatalf("sem filtro: hits = %d, quer 3", n)
	}

	if n, _ := buscar(t, map[string]any{"modified_after": "2100-01-01T00:00:00Z"}); n != 0 {
		t.Fatalf("modified_after no futuro: hits = %d, quer 0 — o filtro nao e aplicado", n)
	}

	for _, forma := range []string{"2000-01-01", "2000-01-01T00:00:00Z"} {
		t.Run("aceita "+forma, func(t *testing.T) {
			n, erro := buscar(t, map[string]any{"modified_after": forma})
			if erro {
				t.Fatalf("forma valida %q foi rejeitada", forma)
			}
			if n != 3 {
				t.Fatalf("hits = %d, quer 3", n)
			}
		})
	}

	t.Run("recusa data invalida", func(t *testing.T) {
		n, erro := buscar(t, map[string]any{"modified_after": "ontem"})
		if !erro {
			t.Fatalf("modified_after=%q foi aceito e devolveu %d hits: o filtro "+
				"sumiu e a busca respondeu como se filtrada", "ontem", n)
		}
	})

	t.Run("recusa modified_before invalido", func(t *testing.T) {
		if n, erro := buscar(t, map[string]any{"modified_before": "semana passada"}); !erro {
			t.Fatalf("modified_before invalido foi aceito e devolveu %d hits", n)
		}
	})
}
