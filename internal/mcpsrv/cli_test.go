package mcpsrv_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/jonyduque/Gobsidian/internal/config"
	"github.com/jonyduque/Gobsidian/internal/index"
	"github.com/jonyduque/Gobsidian/internal/mcpsrv"
	"github.com/jonyduque/Gobsidian/internal/service"
	"github.com/jonyduque/Gobsidian/internal/vault"
)

// TestEsquemasDeEntradaTrazAsQuatorzeTools: o que a CLI monta sai daqui, e
// uma tool que sumisse da lista sumiria da CLI em silencio.
func TestEsquemasDeEntradaTrazAsQuatorzeTools(t *testing.T) {
	esquemas, err := mcpsrv.EsquemasDeEntrada()
	if err != nil {
		t.Fatalf("EsquemasDeEntrada: %v", err)
	}
	porNome := map[string]mcpsrv.EsquemaDeTool{}
	for _, e := range esquemas {
		porNome[e.Nome] = e
	}
	escrita := map[string]bool{"note_create": true, "note_append": true, "note_patch": true, "note_move": true, "note_delete": true}
	for _, nome := range []string{
		"note_read", "note_list", "note_outline", "note_metadata", "note_create", "note_append",
		"note_patch", "note_move", "note_delete", "vault_stats", "vault_search", "vault_broken_links",
		"tag_list", "link_graph",
	} {
		e, ok := porNome[nome]
		if !ok {
			t.Errorf("%s ausente dos esquemas", nome)
			continue
		}
		if e.Escrita != escrita[nome] {
			t.Errorf("%s: Escrita = %v, quer %v", nome, e.Escrita, escrita[nome])
		}
	}
	if len(esquemas) != 14 {
		t.Errorf("%d esquemas, quer 14", len(esquemas))
	}
}

// TestEsquemaDoNoteReadDizQuePathsAceitaObjeto: o item de paths e string OU
// objeto, e a CLI precisa saber das duas formas para dizer que a segunda vai
// por --args.
func TestEsquemaDoNoteReadDizQuePathsAceitaObjeto(t *testing.T) {
	esquemas, err := mcpsrv.EsquemasDeEntrada()
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range esquemas {
		if e.Nome != "note_read" {
			continue
		}
		for _, p := range e.Parametros {
			switch p.Nome {
			case "paths":
				if p.Tipo != "array" || p.TipoDoItem != "string" || !p.ItemAceitaObjeto {
					t.Errorf("paths = %+v, quer array de string que aceita objeto", p)
				}
			case "max_bytes", "offset":
				if p.Tipo != "integer" {
					t.Errorf("%s: tipo %q, quer integer", p.Nome, p.Tipo)
				}
			case "include_frontmatter":
				if p.Tipo != "boolean" {
					t.Errorf("include_frontmatter: tipo %q, quer boolean", p.Tipo)
				}
			}
		}
		return
	}
	t.Fatal("note_read ausente")
}

// TestLerEsquemasCustaPouco: EsquemasDeEntrada roda na partida de todo
// comando, `serve` inclusive, porque a arvore de comandos e montada antes de
// saber qual foi pedido. Medido em 2026-09-27; o teto e folgado de proposito,
// para pegar regressao de ordem de grandeza, e nao ruido.
func TestLerEsquemasCustaPouco(t *testing.T) {
	// LerEsquemas, e nao EsquemasDeEntrada: esta guarda o resultado por
	// processo, e o teste mediria zero se outro teste tivesse lido antes.
	inicio := time.Now()
	if _, err := mcpsrv.LerEsquemas(); err != nil {
		t.Fatal(err)
	}
	if d := time.Since(inicio); !raceEnabled && d > 500*time.Millisecond {
		t.Errorf("ler os esquemas levou %v", d)
	}
}

func servicoDeTeste(t *testing.T, root string) (*service.Service, config.Config) {
	t.Helper()
	v, err := vault.New(root)
	if err != nil {
		t.Fatal(err)
	}
	idx := index.New()
	if err := idx.Build(context.Background(), v); err != nil {
		t.Fatal(err)
	}
	return service.New(v, idx, nil, nil, service.Options{}), config.Defaults()
}

// TestChamarLocalDevolveOStructuredContent: o que sai e o JSON da tool.
func TestChamarLocalDevolveOStructuredContent(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "A.md", "# A\n")
	writeFile(t, root, "sub/B.md", "# B\n")
	svc, cfg := servicoDeTeste(t, root)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	raw, err := mcpsrv.ChamarLocal(ctx, svc, cfg, slog.New(slog.NewTextHandler(io.Discard, nil)), "vault_stats", map[string]any{})
	if err != nil {
		t.Fatalf("ChamarLocal: %v", err)
	}
	var stats struct {
		Notes int `json:"notes"`
	}
	if err := json.Unmarshal(raw, &stats); err != nil {
		t.Fatalf("JSON: %v\n%s", err, raw)
	}
	if stats.Notes != 2 {
		t.Errorf("notes = %d, quer 2 (%s)", stats.Notes, raw)
	}
}

// TestChamarLocalDevolveOCodigoDoErro: o erro da tool chega com o codigo de
// dominio, que e o que um script compara.
func TestChamarLocalDevolveOCodigoDoErro(t *testing.T) {
	svc, cfg := servicoDeTeste(t, t.TempDir())
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := mcpsrv.ChamarLocal(ctx, svc, cfg, slog.New(slog.NewTextHandler(io.Discard, nil)),
		"note_metadata", map[string]any{"path": "nao-existe.md"})
	var et *mcpsrv.ErroDeTool
	if !errors.As(err, &et) {
		t.Fatalf("err = %v (%T), quer *ErroDeTool", err, err)
	}
	if et.Codigo != string(service.CodeNoteNotFound) {
		t.Errorf("codigo = %q, quer %q (mensagem: %s)", et.Codigo, service.CodeNoteNotFound, et.Mensagem)
	}
}

// TestTodoParametroTemDescricao: a descricao do parametro e o que o modelo le
// no host para decidir o que mandar, e e o texto da flag no --help da CLI. Ate
// 2026-10-01, 19 parametros de 6 tools nao tinham nenhuma, e a ajuda mostrava
// "Parametro depth da tool (ver docs/TOOLS.md)".
func TestTodoParametroTemDescricao(t *testing.T) {
	es, err := mcpsrv.EsquemasDeEntrada()
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range es {
		for _, p := range e.Parametros {
			if strings.TrimSpace(p.Descricao) == "" {
				t.Errorf("%s.%s nao tem descricao no schema", e.Nome, p.Nome)
			}
		}
	}
}
