package service

import (
	"context"
	"testing"
)

const notaComTudo = "---\ntitle: Origem\nautor: fulano\n---\n" +
	"# Origem\n\n" +
	"campo:: valor inline\n\n" +
	"## Prescricao\n\n" +
	"Ver [[Alvo|o alvo]] e tambem [[Alvo#Secao]].\n\n" +
	"## Honorarios\n\n" +
	"Nada aqui.\n"

func servicoComOrigemEAlvo(t *testing.T) *Service {
	t.Helper()
	root := t.TempDir()
	writeFile(t, root, "Origem.md", notaComTudo)
	writeFile(t, root, "Alvo.md", "# Alvo\n\n## Secao\n\ntexto\n")
	return newTestService(t, root)
}

// TestMetadataHeadingsTrazNivelSlugEOffsets é o achado M3.
//
// `docs/TOOLS.md` promete que `headings` traz "nível, texto, slug e offsets — o
// que permite planejar uma leitura ou uma escrita seletiva antes de fazê-la". O
// código devolvia `[]string` com só o texto: sem slug não dá para montar a
// âncora, sem offsets não dá para planejar leitura seletiva. O campo existia e
// respondia menos do que o contrato dizia.
func TestMetadataHeadingsTrazNivelSlugEOffsets(t *testing.T) {
	svc := servicoComOrigemEAlvo(t)

	res, err := svc.NoteMetadata(context.Background(), MetadataRequest{
		Path: "Origem.md", Include: []string{"headings"},
	})
	if err != nil {
		t.Fatalf("NoteMetadata: %v", err)
	}
	if len(res.Headings) < 3 {
		t.Fatalf("headings = %d, queria ao menos 3", len(res.Headings))
	}

	var viuNivel2 bool
	for _, h := range res.Headings {
		if h.Text == "" {
			t.Errorf("heading sem texto: %+v", h)
		}
		if h.Slug == "" {
			t.Errorf("heading %q sem slug: sem ele o cliente nao consegue montar a ancora", h.Text)
		}
		if h.Level < 1 {
			t.Errorf("heading %q com nivel %d", h.Text, h.Level)
		}
		if h.End <= h.Start {
			t.Errorf("heading %q com offsets %d..%d: sem eles nao da para planejar leitura seletiva",
				h.Text, h.Start, h.End)
		}
		if h.Level == 2 {
			viuNivel2 = true
		}
	}
	// Sem isto o teste passaria numa nota de um heading só, sem exercitar o
	// nível — que é justamente o campo que distingue seção de subseção.
	if !viuNivel2 {
		t.Error("nenhum heading de nivel 2: o cenario nao exercita a variacao de nivel")
	}
}

// TestMetadataHonraInlineFields é o achado M4: `inline_fields` estava no enum do
// schema, era aceito, e o código o descartava.
//
// Schema que promete e código que ignora é pior que parâmetro ausente — o modelo
// do outro lado lê o schema justamente para decidir o que pedir, e não tem como
// saber que o pedido não fez nada.
func TestMetadataHonraInlineFields(t *testing.T) {
	svc := servicoComOrigemEAlvo(t)

	res, err := svc.NoteMetadata(context.Background(), MetadataRequest{
		Path: "Origem.md", Include: []string{"inline_fields"},
	})
	if err != nil {
		t.Fatalf("NoteMetadata: %v", err)
	}
	if len(res.InlineFields) == 0 {
		t.Fatal("inline_fields veio vazio: o parametro do schema continua sendo descartado")
	}
	if v := res.InlineFields["campo"]; len(v) == 0 || v[0] != "valor inline" {
		t.Errorf("inline_fields[campo] = %v, queria [\"valor inline\"]", v)
	}

	// E o contrapeso: sem pedir, não vem. Um campo que aparece sempre não estaria
	// honrando o `include` — estaria ignorando ele na outra direção.
	semPedir, err := svc.NoteMetadata(context.Background(), MetadataRequest{
		Path: "Origem.md", Include: []string{"tags"},
	})
	if err != nil {
		t.Fatalf("NoteMetadata: %v", err)
	}
	if len(semPedir.InlineFields) != 0 {
		t.Errorf("inline_fields veio sem ser pedido: %v", semPedir.InlineFields)
	}
}

// TestLinkGraphTrazDistanciaEDetalheDaAresta é o achado M5.
//
// `docs/TOOLS.md` promete nós com "caminho, título, distância da origem" e
// arestas com "origem, destino, tipo, alias, âncora, resolvido". Vinham só
// caminho/título e origem/destino/tipo.
func TestLinkGraphTrazDistanciaEDetalheDaAresta(t *testing.T) {
	svc := servicoComOrigemEAlvo(t)

	res, err := svc.LinkGraph(context.Background(), GraphRequest{
		Path: "Origem.md", Depth: 2, Direction: "outgoing",
	})
	if err != nil {
		t.Fatalf("LinkGraph: %v", err)
	}

	distancias := map[string]int{}
	for _, n := range res.Nodes {
		distancias[n.Path] = n.Distance
	}
	if d, ok := distancias["Origem.md"]; !ok || d != 0 {
		t.Errorf("distancia da propria origem = %d (presente=%v), queria 0", d, ok)
	}
	if d, ok := distancias["Alvo.md"]; !ok || d != 1 {
		t.Errorf("distancia de Alvo.md = %d (presente=%v), queria 1", d, ok)
	}

	// Duas referências distintas para o MESMO alvo: uma com alias, outra com
	// âncora. Antes a chave da aresta era só origem+destino, então elas
	// colapsavam numa só — e publicar a âncora de uma delas seria escolher
	// arbitrariamente qual das duas contar.
	var comAlias, comAncora bool
	for _, e := range res.Edges {
		if e.Source != "Origem.md" || e.Target != "Alvo.md" {
			continue
		}
		if !e.Resolved {
			t.Errorf("aresta para alvo existente marcada como nao resolvida: %+v", e)
		}
		if e.Alias == "o alvo" {
			comAlias = true
		}
		if e.Anchor == "Secao" {
			comAncora = true
		}
	}
	if !comAlias {
		t.Error("nenhuma aresta trouxe o alias da referencia")
	}
	if !comAncora {
		t.Error("nenhuma aresta trouxe a ancora da referencia")
	}
	if !comAlias || !comAncora {
		t.Logf("arestas devolvidas: %+v", res.Edges)
	}
}

// TestNotaInexistenteNaoAcusaSaidaDoCofre é o achado M2, e é o mais sério do
// grupo porque a resposta errada é uma ACUSAÇÃO.
//
// `LinkGraph` e `NoteMetadata` respondiam `PATH_OUTSIDE_VAULT` para qualquer
// falha de `ResolvePath`, inclusive nota que simplesmente não existe. O host lê
// esse código como tentativa de escapar do cofre: errar um nome de nota passava
// a acusar o cliente de algo que ele não fez.
func TestNotaInexistenteNaoAcusaSaidaDoCofre(t *testing.T) {
	svc := servicoComOrigemEAlvo(t)
	ctx := context.Background()

	casos := []struct {
		nome string
		erro func() error
	}{
		{"NoteMetadata", func() error {
			_, err := svc.NoteMetadata(ctx, MetadataRequest{Path: "NaoExiste.md"})
			return err
		}},
		{"LinkGraph", func() error {
			_, err := svc.LinkGraph(ctx, GraphRequest{Path: "NaoExiste.md"})
			return err
		}},
		{"ReadNote", func() error {
			_, err := svc.ReadNote(ctx, ReadRequest{Path: "NaoExiste.md"})
			return err
		}},
	}

	for _, c := range casos {
		err := c.erro()
		if err == nil {
			t.Errorf("%s: nota inexistente foi aceita", c.nome)
			continue
		}
		if got := CodeOf(err); got != CodeNoteNotFound {
			t.Errorf("%s: codigo = %s, queria %s\nerro: %v", c.nome, got, CodeNoteNotFound, err)
		}
	}

	// O contrapeso: caminho que de fato tenta sair do cofre CONTINUA sendo
	// classificado como tal. Sem isto, "corrigir" o M2 seria só apagar o sinal.
	if _, err := svc.NoteMetadata(ctx, MetadataRequest{Path: "../fora.md"}); err == nil {
		t.Error("caminho com ../ foi aceito")
	} else if got := CodeOf(err); got != CodePathOutsideVault {
		t.Errorf("codigo para ../ = %s, queria %s", got, CodePathOutsideVault)
	}
}
