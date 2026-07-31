package mcpsrv_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jonyd/gobsidian/internal/config"
)

// Os defaults documentados em docs/TOOLS.md so existem se a struct de entrada
// conseguir distinguir "omitido" de "false". Com `bool` simples ela nao
// consegue: o valor omitido chega como false, e o schema que o modelo do outro
// lado leu passa a mentir.
//
// A revisao de 2026-07-30 achou tres campos assim, todos com `"default": true`
// no contrato: note_create.create_folders, note_append.ensure_blank_line e
// vault_stats.include_health. As tools do M5 (note_move, note_delete) ja usavam
// *bool, inclusive no to_trash — o destrutivo estava certo.
//
// Estes testes chamam cada tool SEM o campo e afirmam o comportamento
// documentado. Testar que o campo e aceito nao prova nada: o defeito e
// justamente o caminho em que ele nao e enviado.

func novaSessao(t *testing.T, root string) (*mcp.ClientSession, context.Context) {
	t.Helper()
	srv := newTestServerWithConfig(t, root, config.Defaults())

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

func TestDefault_NoteCreateCreatesFoldersWhenOmitted(t *testing.T) {
	root := t.TempDir()
	session, ctx := novaSessao(t, root)

	// create_folders OMITIDO. docs/TOOLS.md declara "default": true, entao a
	// pasta intermediaria tem de ser criada.
	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "note_create",
		Arguments: map[string]any{"path": "Nova/Pasta/nota.md", "content": "# Nota\n"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if res.IsError {
		t.Fatalf("note_create falhou com create_folders omitido — o default do "+
			"schema nao esta sendo aplicado: %+v", res.Content)
	}
	if _, err := os.Stat(filepath.Join(root, "Nova", "Pasta", "nota.md")); err != nil {
		t.Errorf("a nota nao foi criada na pasta intermediaria: %v", err)
	}
}

func TestDefault_NoteAppendEnsuresBlankLineWhenOmitted(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.md"), []byte("# Titulo\nprimeira linha\n"), 0644); err != nil {
		t.Fatal(err)
	}
	session, ctx := novaSessao(t, root)

	// ensure_blank_line OMITIDO; docs/TOOLS.md declara "default": true.
	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "note_append",
		Arguments: map[string]any{"path": "a.md", "content": "anexado"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if res.IsError {
		t.Fatalf("note_append devolveu erro: %+v", res.Content)
	}

	lido, err := os.ReadFile(filepath.Join(root, "a.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(lido), "\n\nanexado") {
		t.Errorf("sem linha em branco antes do conteudo anexado — o default do "+
			"schema nao foi aplicado:\n%q", string(lido))
	}
}

func TestDefault_VaultStatsIncludesHealthWhenOmitted(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "orfa.md"), []byte("# Orfa\n\nsem backlink\n"), 0644); err != nil {
		t.Fatal(err)
	}
	session, ctx := novaSessao(t, root)

	// include_health OMITIDO; docs/TOOLS.md declara "default": true, entao
	// orphans, broken_links e broken_anchors tem de vir.
	res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "vault_stats"})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if res.IsError {
		t.Fatalf("vault_stats devolveu erro: %+v", res.Content)
	}

	bruto, err := json.Marshal(res.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	// Afirmar o VALOR, nao a presenca do nome. `orphans` e `broken_links` sao
	// int sem omitempty: aparecem no JSON com ou sem include_health, entao um
	// teste que so procura a string do campo NAO PODE FALHAR. A primeira versao
	// deste teste fazia exatamente isso, e a mutacao de `includeHealth := true`
	// para `false` sobrevivia — falso passe escrito dentro do teste que existe
	// para pegar falso passe.
	var resposta struct {
		Orphans int `json:"orphans"`
	}
	if err := json.Unmarshal(bruto, &resposta); err != nil {
		t.Fatalf("resposta ilegivel: %v\n%s", err, bruto)
	}
	// O cofre tem exatamente uma nota, sem backlink: uma orfa.
	if resposta.Orphans != 1 {
		t.Errorf("orphans = %d, quer 1 — com include_health omitido o default "+
			"do schema nao foi aplicado e a contagem veio zerada:\n%s",
			resposta.Orphans, bruto)
	}
}
