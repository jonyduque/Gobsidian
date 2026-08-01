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

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jonyd/gobsidian/internal/config"
	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/mcpsrv"
	"github.com/jonyd/gobsidian/internal/search"
	"github.com/jonyd/gobsidian/internal/service"
	"github.com/jonyd/gobsidian/internal/vault"
	"github.com/jonyd/gobsidian/internal/watcher"
)

// Este arquivo cobre a composicao que a Task 65 pediu e que nao foi escrita:
// mover uma nota COM O SERVIDOR RODANDO e confirmar que vault_search e
// link_graph passam a responder pelo caminho novo.
//
// A lacuna era real e nao era teorica. As pecas existiam e cada uma tinha
// teste seu: note_move renomeia no disco e reescreve os links; o watcher
// correlaciona rename por xxhash; index.MoveNote transporta a entrada. Nenhum
// teste ligava as tres. E o ponto de emenda e justamente onde o defeito cabe,
// porque NENHUMA tool de escrita atualiza o indice direto — quem atualiza e o
// watcher, por evento. Se a correlacao de rename nao disparar, o disco fica
// certo e o indice fica apontando para um arquivo que nao existe mais, com
// todas as tools de leitura respondendo com confianca a partir dele.
//
// Por isso o servidor aqui e montado como cmd/gobsidian monta: vault, indice,
// invertido, watcher rodando, servico e servidor MCP sobre o mesmo idx e o
// mesmo inv. Um teste que construisse o indice depois do move mediria o
// index.Build, nao o watcher.

// servidorComWatcher monta a pilha inteira e devolve a sessao MCP ja
// conectada, com o watcher rodando sobre o mesmo indice que o servico le.
func servidorComWatcher(t *testing.T, root string, debounce time.Duration) (*mcp.ClientSession, context.Context) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	v, err := vault.New(root)
	if err != nil {
		cancel()
		t.Fatalf("vault.New: %v", err)
	}

	idx := index.New()
	if err := idx.Build(ctx, v); err != nil {
		cancel()
		t.Fatalf("idx.Build: %v", err)
	}

	inv := search.NewInverted()
	for _, p := range idx.NotePaths() {
		data, err := v.ReadAll(ctx, p)
		if err != nil {
			cancel()
			t.Fatalf("ReadAll(%s): %v", p, err)
		}
		body, _ := vault.StripBOM(data)
		inv.Add(string(p), search.Analyze(string(body)))
	}

	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	w, err := watcher.New(v, idx, inv, debounce, log)
	if err != nil {
		cancel()
		t.Fatalf("watcher.New: %v", err)
	}

	parou := make(chan struct{})
	go func() {
		defer close(parou)
		_ = w.Run(ctx)
	}()

	// Um unico Cleanup, em ordem fixa: cancela, ESPERA Run sair, so entao
	// fecha. Fechar antes de Run sair puxa o fsnotify debaixo da goroutine que
	// ainda le dele, e no Windows deixa handle de diretorio preso — ver
	// TestWatcher_CloseReleasesHandles.
	t.Cleanup(func() {
		cancel()
		<-parou
		_ = w.Close()
	})

	svc := service.New(v, idx, inv, nil, service.Options{})
	srv := mcpsrv.New(ctx, svc, config.Defaults(), log)

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

// chamaTool chama a tool e decodifica o StructuredContent em destino.
func chamaTool(ctx context.Context, t *testing.T, session *mcp.ClientSession, nome string, args map[string]any, destino any) {
	t.Helper()

	res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: nome, Arguments: args})
	if err != nil {
		t.Fatalf("%s: %v", nome, err)
	}
	if res.IsError {
		t.Fatalf("%s devolveu erro: %+v", nome, res.Content)
	}
	bruto, err := json.Marshal(res.StructuredContent)
	if err != nil {
		t.Fatalf("%s: resposta nao serializa: %v", nome, err)
	}
	if err := json.Unmarshal(bruto, destino); err != nil {
		t.Fatalf("%s: resposta ilegivel: %v\n%s", nome, err, bruto)
	}
}

// esperaIndice repete fn ate ela devolver nil ou o prazo acabar, e devolve o
// ultimo motivo. O watcher e assincrono: sem espera o teste mediria a corrida
// entre a chamada da tool e o evento, e passaria ou falharia por tempo.
//
// O prazo e generoso de proposito. Um teste de composicao que pisca por 200 ms
// de folga vira um teste que alguem desliga.
func esperaIndice(t *testing.T, prazo time.Duration, fn func() error) error {
	t.Helper()

	limite := time.Now().Add(prazo)
	var ultimo error
	for time.Now().Before(limite) {
		ultimo = fn()
		if ultimo == nil {
			return nil
		}
		time.Sleep(25 * time.Millisecond)
	}
	return ultimo
}

func TestE2E_NoteMoveIsReflectedBySearchAndGraph(t *testing.T) {
	root := t.TempDir()

	// "zarabatana" nao aparece em mais nenhuma nota: se a busca por ela
	// devolver um caminho, e o da nota movida, e nao ha como o teste passar por
	// casar outro documento.
	writeFile(t, root, "origem/alvo.md",
		"---\ntags: [juridico]\n---\n\n# Alvo\n\nzarabatana e a palavra sonda desta nota.\n")
	writeFile(t, root, "aponta.md",
		"# Aponta\n\nEsta nota cita [[alvo]] e nada mais.\n")

	session, ctx := servidorComWatcher(t, root, 20*time.Millisecond)

	// Estado inicial: a busca acha a nota no caminho ANTIGO. Sem esta
	// afirmacao, um corpus que nunca indexou passaria pelo resto do teste
	// vacuamente — o "sumiu do caminho antigo" seria verdade porque nunca
	// esteve la.
	var antes struct {
		Results []struct {
			Path string `json:"path"`
		} `json:"results"`
	}
	chamaTool(ctx, t, session, "vault_search", map[string]any{"query": "zarabatana"}, &antes)
	if len(antes.Results) != 1 || antes.Results[0].Path != "origem/alvo.md" {
		t.Fatalf("estado inicial errado: vault_search por zarabatana deu %+v, "+
			"quer exatamente [origem/alvo.md]", antes.Results)
	}

	// O move, pela tool, com o servidor no ar.
	var move struct {
		To           string `json:"to"`
		LinksUpdated int    `json:"links_updated"`
	}
	chamaTool(ctx, t, session, "note_move", map[string]any{
		"from": "origem/alvo.md",
		"to":   "destino/renomeada.md",
	}, &move)

	// Disco primeiro: se o rename nao aconteceu, o resto do teste estaria
	// medindo outra coisa.
	if _, err := os.Stat(filepath.Join(root, "destino", "renomeada.md")); err != nil {
		t.Fatalf("a nota nao esta no caminho novo: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "origem", "alvo.md")); !os.IsNotExist(err) {
		t.Errorf("a nota continua no caminho antigo (Stat err = %v)", err)
	}

	// O link em aponta.md foi reescrito para o nome novo. E isso que torna o
	// link_graph adiante uma pergunta com resposta: com [[alvo]] intacto, o
	// link resolveria por nome e o grafo pareceria certo sem o indice ter
	// seguido o move.
	apontaDepois, err := os.ReadFile(filepath.Join(root, "aponta.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(apontaDepois), "[[renomeada]]") {
		t.Errorf("o link em aponta.md nao foi reescrito:\n%s", apontaDepois)
	}

	// AQUI e a emenda que nao tinha teste: o watcher precisa correlacionar o
	// rename e mover a entrada no indice. Nenhuma tool de escrita faz isso.
	if err := esperaIndice(t, 10*time.Second, func() error {
		var depois struct {
			Results []struct {
				Path string `json:"path"`
			} `json:"results"`
		}
		chamaTool(ctx, t, session, "vault_search", map[string]any{"query": "zarabatana"}, &depois)
		if len(depois.Results) != 1 {
			return fmt.Errorf("vault_search devolveu %d resultados, quer 1: %+v", len(depois.Results), depois.Results)
		}
		if depois.Results[0].Path != "destino/renomeada.md" {
			return fmt.Errorf("vault_search ainda responde por %q; o indice nao seguiu o move",
				depois.Results[0].Path)
		}
		return nil
	}); err != nil {
		t.Fatalf("vault_search nao refletiu o caminho novo em 10s: %v", err)
	}

	// E o grafo: aponta.md tem de aparecer como vizinho do caminho NOVO.
	if err := esperaIndice(t, 10*time.Second, func() error {
		var grafo struct {
			Nodes []struct {
				Path string `json:"path"`
			} `json:"nodes"`
			Edges []struct {
				Source string `json:"source"`
				Target string `json:"target"`
			} `json:"edges"`
		}
		chamaTool(ctx, t, session, "link_graph",
			map[string]any{"path": "destino/renomeada.md"}, &grafo)

		temAresta := false
		for _, e := range grafo.Edges {
			if e.Source == "aponta.md" && e.Target == "destino/renomeada.md" {
				temAresta = true
			}
			// O caminho antigo nao pode sobreviver em ponta nenhuma: entrada
			// velha que fica e exatamente o defeito que a divergencia de caixa
			// em byAlias produziu, com [[STJ]] resolvendo para nota removida.
			if e.Source == "origem/alvo.md" || e.Target == "origem/alvo.md" {
				return fmt.Errorf("o caminho antigo sobrevive numa aresta: %s -> %s", e.Source, e.Target)
			}
		}
		if !temAresta {
			return fmt.Errorf("link_graph nao traz aponta.md -> destino/renomeada.md: nodes=%+v edges=%+v",
				grafo.Nodes, grafo.Edges)
		}
		return nil
	}); err != nil {
		t.Fatalf("link_graph nao refletiu o caminho novo em 10s: %v", err)
	}
}
