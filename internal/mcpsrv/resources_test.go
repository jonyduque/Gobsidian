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
	if len(res.Resources) != 3 {
		t.Fatalf("resources = %d, quer 3: %+v", len(res.Resources), res.Resources)
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

// Acima de 200 notas, a maioria NAO e publicada como resource concreto: quem
// atende essas leituras e o template `gobsidian:///{+path}`.
//
// O `+` e o operador de expansao reservada do RFC 6570. Sem ele a barra do
// caminho seria escapada na expansao, e o template deixaria de casar com
// qualquer nota fora da raiz — falha que nao aparece em cofre pequeno, porque
// ali todas as notas cabem na lista concreta.
//
// Um cofre de estudo com centenas de notas e o alvo deste produto, entao este
// caminho e o comum, nao o excepcional.
func TestReadingANoteBeyondThePublishedLimitUsesTheTemplate(t *testing.T) {
	root := t.TempDir()
	for i := range 210 {
		writeFile(t, root, fmt.Sprintf("Pasta com espaco/Nota %03d.md", i),
			fmt.Sprintf("# Nota %03d\n\nCorpo da nota %03d.\n", i, i))
	}

	srv := newTestServerWithIndex(t, root)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	go func() { _ = srv.Connect(ctx, serverTransport) }()

	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "0.0.1"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer func() { _ = session.Close() }()

	list, err := session.ListResources(ctx, nil)
	if err != nil {
		t.Fatalf("ListResources: %v", err)
	}
	if len(list.Resources) != 200 {
		t.Fatalf("resources = %d, quer 200 - o limite e o que torna este teste util", len(list.Resources))
	}

	publicadas := make(map[string]bool, len(list.Resources))
	for _, r := range list.Resources {
		publicadas[r.URI] = true
	}

	// Encontra uma nota que existe no cofre e NAO foi publicada.
	var alvo string
	for i := range 210 {
		uri := fmt.Sprintf("gobsidian:///Pasta%%20com%%20espaco/Nota%%20%03d.md", i)
		if !publicadas[uri] {
			alvo = uri
			break
		}
	}
	if alvo == "" {
		t.Fatal("todas as 210 notas foram publicadas - o limite de 200 nao esta valendo")
	}

	read, err := session.ReadResource(ctx, &mcp.ReadResourceParams{URI: alvo})
	if err != nil {
		t.Fatalf("ReadResource(%q) pelo template: %v", alvo, err)
	}
	if len(read.Contents) != 1 {
		t.Fatalf("contents = %d, quer 1", len(read.Contents))
	}
	if !strings.Contains(read.Contents[0].Text, "Corpo da nota") {
		t.Errorf("conteudo devolvido nao e o da nota: %q", read.Contents[0].Text)
	}
}
