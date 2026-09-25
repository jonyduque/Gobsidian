package mcpsrv_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// O panic original, contra um cofre de verdade:
//
//	panic: parse "gobsidian://test vault/Origem.md": invalid character " " in host name
//
// Concatenar "gobsidian://" com o caminho canonico faz o PRIMEIRO SEGMENTO do
// caminho virar autoridade da URI, e autoridade nao aceita espaco. O servidor
// morria dentro de AddResource, no boot, antes de anunciar uma unica tool.
//
// TestResources ja exercitava esse caminho e passava: o cofre dele tem uma nota
// chamada "A.md". Espaco em nome de pasta ou de nota e o caso comum num cofre
// do Obsidian, e era o unico caso que faltava.
//
// Este teste falha com panic, nao com asercao, se a construcao voltar atras.
func TestResourceRegistrationSurvivesPathsWithSpaces(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "Minha nota.md", "# Minha nota\n")
	writeFile(t, root, "test vault/Origem.md", "# Origem\n")
	writeFile(t, root, "Civil/PONTO 03.md", "# Ponto 3\n\nCorpo do ponto.\n")

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
	// Tres notas, duas pastas ("test vault" e "Civil") e a raiz.
	if len(res.Resources) != 6 {
		t.Fatalf("resources = %d, quer 6: %+v", len(res.Resources), res.Resources)
	}

	for _, r := range res.Resources {
		if strings.Contains(r.URI, " ") {
			t.Errorf("URI com espaco cru: %q - e o caractere que torna a URI improcessavel", r.URI)
		}
		if !strings.HasPrefix(r.URI, "gobsidian:///") {
			t.Errorf("URI %q nao comeca com gobsidian:/// - com duas barras o primeiro segmento vira host", r.URI)
		}
	}

	// Publicar a URI certa nao basta: o handler precisa conseguir voltar dela
	// ao caminho canonico. Se a volta nao fechar, o sintoma e "nota nao
	// encontrada" para exatamente as notas cujo nome precisou de escape.
	var alvo string
	for _, r := range res.Resources {
		if strings.Contains(r.URI, "PONTO") {
			alvo = r.URI
		}
	}
	if alvo == "" {
		t.Fatalf("nota com espaco no nome nao foi publicada: %+v", res.Resources)
	}

	read, err := session.ReadResource(ctx, &mcp.ReadResourceParams{URI: alvo})
	if err != nil {
		t.Fatalf("ReadResource(%q): %v", alvo, err)
	}
	if len(read.Contents) != 1 {
		t.Fatalf("contents = %d, quer 1", len(read.Contents))
	}
	if !strings.Contains(read.Contents[0].Text, "Corpo do ponto.") {
		t.Errorf("conteudo devolvido nao e o da nota: %q", read.Contents[0].Text)
	}
}

// listarTudo segue nextCursor ate o fim, como o Claude Desktop faz (medido em
// 2026-09-14, E1). Sem isso o teste veria so a primeira pagina do SDK.
func listarTudo(ctx context.Context, t *testing.T, session *mcp.ClientSession) []*mcp.Resource {
	t.Helper()
	var todos []*mcp.Resource
	params := &mcp.ListResourcesParams{}
	for paginas := 0; ; paginas++ {
		if paginas > 50 {
			t.Fatal("mais de 50 paginas: o cursor nao termina")
		}
		res, err := session.ListResources(ctx, params)
		if err != nil {
			t.Fatalf("ListResources: %v", err)
		}
		todos = append(todos, res.Resources...)
		if res.NextCursor == "" {
			return todos
		}
		params = &mcp.ListResourcesParams{Cursor: res.NextCursor}
	}
}

// Ate 2026-09-25 so as 200 notas mais recentes eram publicadas; no cofre
// Estudo, 94% das notas nunca apareciam no menu. Agora o cofre inteiro e
// publicado, e 1.010 notas passam da pagina de 1.000 do SDK: o teste so fecha
// se a paginacao entregar a ultima.
func TestResourcesPublicaOCofreInteiro(t *testing.T) {
	root := t.TempDir()
	const total = 1010
	for i := range total {
		writeFile(t, root, fmt.Sprintf("Pasta com espaco/Nota %04d.md", i),
			fmt.Sprintf("# Nota %04d\n\nCorpo da nota %04d.\n", i, i))
	}

	srv := newTestServerWithIndex(t, root)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	session := connectTestSession(ctx, t, srv)

	todos := listarTudo(ctx, t, session)
	notas := 0
	for _, r := range todos {
		if r.MIMEType == "text/markdown" {
			notas++
		}
	}
	if notas != total {
		t.Fatalf("notas publicadas = %d, quer %d (resources = %d)", notas, total, len(todos))
	}

	ultima := "gobsidian:///Pasta%20com%20espaco/Nota%201009.md"
	if _, ok := porURI(todos)[ultima]; !ok {
		t.Fatalf("%s nao publicada", ultima)
	}
	read, err := session.ReadResource(ctx, &mcp.ReadResourceParams{URI: ultima})
	if err != nil {
		t.Fatalf("ReadResource(%q): %v", ultima, err)
	}
	if !strings.Contains(read.Contents[0].Text, "Corpo da nota 1009.") {
		t.Errorf("conteudo devolvido nao e o da nota: %q", read.Contents[0].Text)
	}
}

// O template `gobsidian:///{+path}` continua atendendo a URI que o cliente
// monta sozinho e que nao esta na lista — aqui, com outra caixa.
//
// O `+` e o operador de expansao reservada do RFC 6570. Sem ele a barra do
// caminho seria escapada na expansao, e o template deixaria de casar com
// qualquer nota fora da raiz.
func TestTemplateLeURIMontadaPeloCliente(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "Pasta com espaco/Nota 001.md", "# Nota 001\n\nCorpo da nota 001.\n")

	srv := newTestServerWithIndex(t, root)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	session := connectTestSession(ctx, t, srv)

	uri := "gobsidian:///pasta%20com%20espaco/nota%20001.md"
	if _, ok := porURI(listarTudo(ctx, t, session))[uri]; ok {
		t.Fatalf("%q foi publicada; o teste precisa de uma URI que so o template atende", uri)
	}
	read, err := session.ReadResource(ctx, &mcp.ReadResourceParams{URI: uri})
	if err != nil {
		t.Fatalf("ReadResource(%q) pelo template: %v", uri, err)
	}
	if !strings.Contains(read.Contents[0].Text, "Corpo da nota 001.") {
		t.Errorf("conteudo devolvido nao e o da nota: %q", read.Contents[0].Text)
	}
}

func porURI(todos []*mcp.Resource) map[string]*mcp.Resource {
	m := make(map[string]*mcp.Resource, len(todos))
	for _, r := range todos {
		m[r.URI] = r
	}
	return m
}

// Name carrega o caminho (e por ele que a busca do Desktop casa); Title
// carrega a folha primeiro e o contexto depois, porque o Desktop mostra so o
// Title e corta pelo fim (E4). A pasta se distingue pelo texto, porque o
// Desktop nao desenha icone (E2).
func TestResourceNameECaminhoTitleEFolha(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "Direito/Penal/Dolo.md", "---\ntitle: Dolo eventual\n---\ncorpo\n")
	writeFile(t, root, "Direito/Penal/Culpa.md", "sem titulo nenhum\n")
	writeFile(t, root, "Raiz.md", "# Raiz\n")

	srv := newTestServerWithIndex(t, root)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	session := connectTestSession(ctx, t, srv)
	m := porURI(listarTudo(ctx, t, session))

	casos := []struct {
		uri, name, title, mime string
	}{
		{"gobsidian:///Direito/Penal/Dolo.md", "Direito/Penal/Dolo", "Dolo eventual · Direito/Penal", "text/markdown"},
		{"gobsidian:///Direito/Penal/Culpa.md", "Direito/Penal/Culpa", "Culpa · Direito/Penal", "text/markdown"},
		{"gobsidian:///Raiz.md", "Raiz", "Raiz", "text/markdown"},
		{"gobsidian:///Direito/Penal/", "Direito/Penal/", "📁 Penal · Direito", "inode/directory"},
		{"gobsidian:///Direito/", "Direito/", "📁 Direito", "inode/directory"},
		{"gobsidian:///", "/", "📁 Cofre inteiro", "inode/directory"},
	}
	for _, c := range casos {
		r, ok := m[c.uri]
		if !ok {
			t.Errorf("%s nao publicado", c.uri)
			continue
		}
		if r.Name != c.name || r.Title != c.title || r.MIMEType != c.mime {
			t.Errorf("%s: Name=%q Title=%q MIME=%q, quer %q %q %q", c.uri, r.Name, r.Title, r.MIMEType, c.name, c.title, c.mime)
		}
	}
	if len(m) != len(casos) {
		t.Errorf("resources = %d, quer %d: %v", len(m), len(casos), m)
	}
}

// O read de pasta devolve UM conteudo, um indice em Markdown dos filhos
// diretos — nunca os netos, e nunca o texto das notas: a raiz de Estudo seria
// 3.313 notas numa resposta.
func TestReadDePastaListaOsFilhosDiretos(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "Direito/Penal/Dolo.md", "# Dolo\n\nTEXTO-DA-NOTA\n")
	writeFile(t, root, "Direito/Penal/Sub/Neto.md", "# Neto\n")
	writeFile(t, root, "Direito/Civil.md", "# Civil\n")

	srv := newTestServerWithIndex(t, root)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	session := connectTestSession(ctx, t, srv)

	read, err := session.ReadResource(ctx, &mcp.ReadResourceParams{URI: "gobsidian:///Direito/Penal/"})
	if err != nil {
		t.Fatalf("ReadResource de pasta: %v", err)
	}
	if len(read.Contents) != 1 {
		t.Fatalf("contents = %d, quer 1", len(read.Contents))
	}
	c := read.Contents[0]
	if c.MIMEType != "text/markdown" || c.URI != "gobsidian:///Direito/Penal/" {
		t.Errorf("conteudo com MIME %q e URI %q", c.MIMEType, c.URI)
	}
	for _, quer := range []string{"gobsidian:///Direito/Penal/Dolo.md", "gobsidian:///Direito/Penal/Sub/", "2 notas"} {
		if !strings.Contains(c.Text, quer) {
			t.Errorf("indice da pasta sem %q:\n%s", quer, c.Text)
		}
	}
	for _, nao := range []string{"Neto.md", "Civil.md", "TEXTO-DA-NOTA"} {
		if strings.Contains(c.Text, nao) {
			t.Errorf("indice da pasta traz %q, que nao e filho direto:\n%s", nao, c.Text)
		}
	}

	raiz, err := session.ReadResource(ctx, &mcp.ReadResourceParams{URI: "gobsidian:///"})
	if err != nil {
		t.Fatalf("ReadResource da raiz: %v", err)
	}
	if !strings.Contains(raiz.Contents[0].Text, "gobsidian:///Direito/") || strings.Contains(raiz.Contents[0].Text, "Civil.md") {
		t.Errorf("indice da raiz:\n%s", raiz.Contents[0].Text)
	}

	if _, err := session.ReadResource(ctx, &mcp.ReadResourceParams{URI: "gobsidian:///Nada/"}); err == nil {
		t.Error("read de pasta inexistente nao deu erro")
	}
}

func TestReadDaRaizDeCofreVazioNaoDaErro(t *testing.T) {
	srv := newTestServerWithIndex(t, t.TempDir())
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	session := connectTestSession(ctx, t, srv)

	read, err := session.ReadResource(ctx, &mcp.ReadResourceParams{URI: "gobsidian:///"})
	if err != nil {
		t.Fatalf("ReadResource da raiz vazia: %v", err)
	}
	if len(read.Contents) != 1 {
		t.Fatalf("contents = %d, quer 1", len(read.Contents))
	}
}

// Pasta com espaco e acento: o mesmo motivo de
// TestResourceRegistrationSurvivesPathsWithSpaces, que foi um panic no boot.
func TestResourceDePastaComEspacoEAcento(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "Ação Penal/Nota.md", "# Nota\n")

	srv := newTestServerWithIndex(t, root)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	session := connectTestSession(ctx, t, srv)

	uri := "gobsidian:///A%C3%A7%C3%A3o%20Penal/"
	if _, ok := porURI(listarTudo(ctx, t, session))[uri]; !ok {
		t.Fatalf("%s nao publicado", uri)
	}
	read, err := session.ReadResource(ctx, &mcp.ReadResourceParams{URI: uri})
	if err != nil {
		t.Fatalf("ReadResource(%q): %v", uri, err)
	}
	if !strings.Contains(read.Contents[0].Text, "gobsidian:///A%C3%A7%C3%A3o%20Penal/Nota.md") {
		t.Errorf("indice sem a nota:\n%s", read.Contents[0].Text)
	}
}
