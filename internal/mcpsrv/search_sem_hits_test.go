package mcpsrv_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestVaultSearchNaoDevolveHits(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.md"), []byte("# A\n\npalavra comum\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	session, ctx := sessaoComBusca(t, root)
	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "vault_search", Arguments: map[string]any{"query": "comum"},
	})
	if err != nil || res.IsError {
		t.Fatalf("CallTool: err=%v isError=%v", err, res.IsError)
	}
	bruto, err := json.Marshal(res.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]json.RawMessage
	if err := json.Unmarshal(bruto, &out); err != nil {
		t.Fatalf("resposta ilegivel: %v\n%s", err, bruto)
	}
	if _, tem := out["hits"]; tem {
		t.Fatalf("a resposta ainda traz \"hits\":\n%s", bruto)
	}
	var results []json.RawMessage
	if err := json.Unmarshal(out["results"], &results); err != nil || len(results) != 1 {
		t.Fatalf("results = %s (err=%v), quero 1 item", out["results"], err)
	}
}
