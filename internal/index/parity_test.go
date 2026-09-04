package index

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"testing"

	"github.com/jonyd/gobsidian/internal/parser"
	"github.com/jonyd/gobsidian/internal/vault"
)

type RefHeading struct {
	Level   int    `json:"level"`
	Heading string `json:"heading"`
}

type RefLink struct {
	Link        string  `json:"link"`
	DisplayText *string `json:"displayText"`
	Resolved    *string `json:"resolved"`
}

type RefEmbed struct {
	Link     string  `json:"link"`
	Resolved *string `json:"resolved"`
}

type RefNote struct {
	Headings        []RefHeading `json:"headings"`
	Tags            []string     `json:"tags"`
	FrontmatterTags []string     `json:"frontmatterTags"`
	Aliases         []string     `json:"aliases"`
	Blocks          []string     `json:"blocks"`
	Links           []RefLink    `json:"links"`
	Embeds          []RefEmbed   `json:"embeds"`
}

// Reference e a referencia carregada, junto do grafo resolvido quando o dumper
// que a gerou soube produzi-lo.
type Reference struct {
	Schema          int
	Notes           map[string]RefNote
	ResolvedLinks   map[string]map[string]int
	UnresolvedLinks map[string]map[string]int
}

// HasGraph diz se a referencia carrega o grafo resolvido do proprio Obsidian.
//
// So o dumper schema 2 produz isso. O schema 1 registrava resolucao com
// getFirstLinkpathDest, que nao consulta aliases — comparar contra ele faria
// cada alias virar divergencia falsa, e a reacao natural seria quebrar o nosso
// resolvedor para casar com o instrumento. Por isso a comparacao de grafo pula
// em vez de rodar contra uma referencia que mede errado.
func (r Reference) HasGraph() bool {
	return r.Schema >= 2 && len(r.ResolvedLinks) > 0
}

type dumpV2 struct {
	Schema          int                       `json:"schema"`
	Notes           map[string]RefNote        `json:"notes"`
	ResolvedLinks   map[string]map[string]int `json:"resolvedLinks"`
	UnresolvedLinks map[string]map[string]int `json:"unresolvedLinks"`
}

func loadReference(t *testing.T, path string) Reference {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile(%q): %v", path, err)
	}

	// Schema 2 primeiro. O schema 1 era um mapa plano de caminho para
	// metadados, entao o campo "schema" ausente e o que distingue os dois.
	var v2 dumpV2
	if err := json.Unmarshal(raw, &v2); err == nil && v2.Schema >= 2 {
		return Reference(v2)
	}

	var flat map[string]RefNote
	if err := json.Unmarshal(raw, &flat); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	return Reference{Schema: 1, Notes: flat}
}

// assertGraphMatches compara o grafo RESOLVIDO, nao apenas a presenca dos links.
//
// A presenca dizia pouco: um link pode estar registrado dos dois lados e apontar
// para notas diferentes. O grafo e o objeto da metrica do PRD §7.
//
// A comparacao e assimetrica de proposito, como o resto: toda aresta que o
// Obsidian resolveu precisa existir do nosso lado. O inverso nao e exigido —
// resolvemos coisas que ele nao expoe, como o estado de ancora quebrada.
func assertGraphMatches(t *testing.T, idx *Index, ref Reference) {
	t.Helper()

	for origem, alvos := range ref.ResolvedLinks {
		n, ok := idx.Get(vault.CanonicalPath(origem))
		if !ok {
			t.Errorf("%s: origem do grafo ausente do nosso indice", origem)
			continue
		}

		// A contagem importa, nao so a presenca do alvo. Sem ela, uma regressao
		// que desvia UM link entre varios para o mesmo destino fica invisivel:
		// inverter o desempate de colisao manda [[PONTO 03]] para Penal/, e o
		// alvo Civil/ continua presente pelos outros cinco links. Provado por
		// mutacao — a versao anterior desta comparacao passava.
		nossos := make(map[string]int, len(n.Links))
		for _, l := range n.Links {
			if l.Resolved != "" {
				nossos[string(l.Resolved)]++
			}
		}

		for alvo, querAoMenos := range alvos {
			if nossos[alvo] < querAoMenos {
				t.Errorf("%s: o Obsidian resolve %d aresta(s) para %q e nos resolvemos %d; nosso grafo daqui: %v",
					origem, querAoMenos, alvo, nossos[alvo], contagemOf(nossos))
			}
		}
	}
}

func contagemOf(m map[string]int) []string {
	out := make([]string, 0, len(m))
	for k, n := range m {
		out = append(out, fmt.Sprintf("%s x%d", k, n))
	}
	sort.Strings(out)
	return out
}

// conjuntosIguais compara os DOIS sentidos.
//
// Ate 2026-09-04 cada comparacao daqui iterava so `want`, isto e, perguntava
// "todo item da referencia existe no nosso indice?" — e uma referencia com
// listas vazias corria zero comparacoes e reportava paridade. O sentido que
// faltava e o que pega o excesso: um heading que inventamos, uma tag que o
// Obsidian nao registra, um link a mais. Ordenar antes de comparar e o que
// torna a igualdade uma pergunta sobre CONJUNTO, e nao sobre a ordem em que o
// parser ou o dumper listaram.
func conjuntosIguais(t *testing.T, path, campo string, got, want []string) {
	t.Helper()
	g := slices.Clone(got)
	w := slices.Clone(want)
	slices.Sort(g)
	slices.Sort(w)
	if !slices.Equal(g, w) {
		t.Errorf("%s: %s divergem\n  nosso indice: %v\n  referencia:   %v", path, campo, g, w)
	}
}

func headingsComparaveis(got []parser.Heading) []string {
	out := make([]string, 0, len(got))
	for _, g := range got {
		out = append(out, fmt.Sprintf("h%d %s", g.Level, g.Text))
	}
	return out
}

func headingsDaReferencia(want []RefHeading) []string {
	out := make([]string, 0, len(want))
	for _, w := range want {
		out = append(out, fmt.Sprintf("h%d %s", w.Level, w.Heading))
	}
	return out
}

func blocosComparaveis(got []parser.Block) []string {
	out := make([]string, 0, len(got))
	for _, g := range got {
		out = append(out, g.ID)
	}
	return out
}

// linkKey reconstroi a forma que o Obsidian guarda, para que a comparacao
// compare a mesma coisa dos dois lados.
//
// O Obsidian registra o alvo COM a ancora junto — "Civil/PONTO 03#Cap 1" —
// enquanto nos separamos em Target e Anchor, que e a representacao melhor
// para resolver e reescrever. Comparar Target contra o link do Obsidian faz
// cada link com ancora divergir, em qualquer cofre, e os achados verdadeiros
// afogam no ruido. Foi assim que a primeira rodada de paridade reportou tres
// divergencias falsas ao lado de uma real.
//
// Anchor nao carrega o "#" (splitWikilink o remove), entao ele volta aqui.
// Ancora de bloco ja vem com o "^", que faz parte do identificador.
func linkKey(l ResolvedLink) string {
	if l.Anchor == "" {
		return l.Target
	}
	return l.Target + "#" + l.Anchor
}

func assertLinksMatch(t *testing.T, path string, gotLinks []ResolvedLink, wantLinks []RefLink, wantEmbeds []RefEmbed) {
	t.Helper()

	var gotNormais, gotEmbeds []string
	for _, g := range gotLinks {
		// URL externa fica de fora dos DOIS lados. O Obsidian nao a registra
		// como link do cofre — esta e a mesma constatacao que note.go:23-33
		// documenta em producao, e o proprio corpus a confirma:
		// Neoconstitucionalismo.md tem tres markdown-links http(s) que a
		// referencia nao lista, ao lado de chapter09.xhtml e chapter17.xhtml,
		// que ela lista. Comparar universos diferentes faria toda URL virar
		// divergencia falsa; comparar o mesmo universo nos dois sentidos e o
		// que da valor a igualdade.
		if g.State == LinkExternal {
			continue
		}
		if g.Kind == parser.LinkEmbed {
			gotEmbeds = append(gotEmbeds, linkKey(g))
			continue
		}
		gotNormais = append(gotNormais, linkKey(g))
	}

	querNormais := make([]string, 0, len(wantLinks))
	for _, w := range wantLinks {
		querNormais = append(querNormais, w.Link)
	}
	querEmbeds := make([]string, 0, len(wantEmbeds))
	for _, w := range wantEmbeds {
		querEmbeds = append(querEmbeds, w.Link)
	}

	conjuntosIguais(t, path, "links", gotNormais, querNormais)
	conjuntosIguais(t, path, "embeds", gotEmbeds, querEmbeds)
}

func TestParityWithObsidian(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "parity", "vault")
	refPath := filepath.Join("..", "..", "testdata", "parity", "metadata.json")

	// UM unico motivo para pular: o corpus nao existe nesta maquina. Ele e
	// gerado por um plugin de desenvolvimento do Obsidian (tools/parity-dumper),
	// que nem todo checkout roda.
	//
	// Todo o resto e FALHA, e nao skip. Ate 2026-09-04 um corpus presente e
	// incompleto — diretorio vazio, referencia sem notas — pulava, e o
	// `verify.ps1` ficava verde sem paridade nenhuma. Corpus ausente e um fato
	// do ambiente; corpus presente e quebrado e um defeito.
	if _, err := os.Stat(root); errors.Is(err, fs.ErrNotExist) {
		t.Skipf("corpus de paridade ausente em %s; gere com tools/parity-dumper (ver o README de la)", root)
	}

	notes, _ := filepath.Glob(filepath.Join(root, "*.md"))
	sub, _ := filepath.Glob(filepath.Join(root, "*", "*.md"))
	if len(notes)+len(sub) == 0 {
		t.Fatalf("%s existe mas nao tem nota nenhuma; o corpus de paridade esta incompleto", root)
	}

	ref := loadReference(t, refPath)
	if len(ref.Notes) == 0 {
		t.Fatalf("referencia %s sem notas; o dump nao rodou — ver tools/parity-dumper/README.md", refPath)
	}

	// Guarda de referencia vazia, por CATEGORIA.
	//
	// `len(ref.Notes) > 0` nao basta: uma referencia com sete notas de listas
	// todas vazias faz cada comparacao rodar sobre nada e o teste reporta
	// paridade que ninguem verificou. O corpus de paridade existe justamente
	// para exercitar as sete categorias abaixo; se uma delas zerar, o dump
	// perdeu a metade que ela cobria, e isso e um defeito do instrumento — que
	// e a unica coisa que este teste nao consegue detectar por comparacao.
	var nHeadings, nTags, nFmTags, nBlocos, nLinks, nEmbeds int
	for _, w := range ref.Notes {
		nHeadings += len(w.Headings)
		nTags += len(w.Tags)
		nFmTags += len(w.FrontmatterTags)
		nBlocos += len(w.Blocks)
		nLinks += len(w.Links)
		nEmbeds += len(w.Embeds)
	}
	// A lista e exatamente a das categorias que o laco abaixo COMPARA. Guardar
	// uma categoria que nada compara afirmaria sobre o corpus, nao sobre a
	// cobertura — `aliases` fica de fora por isso.
	for _, c := range []struct {
		nome string
		n    int
	}{
		{"headings", nHeadings}, {"tags", nTags}, {"frontmatterTags", nFmTags},
		{"blocks", nBlocos}, {"links", nLinks}, {"embeds", nEmbeds},
	} {
		if c.n == 0 {
			t.Fatalf("referencia %s nao tem nenhum item da categoria %q; o dump esta incompleto e a comparacao dessa categoria rodaria vazia",
				refPath, c.nome)
		}
	}

	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	idx := New()
	if err := idx.Build(context.Background(), v); err != nil {
		t.Fatalf("Build: %v", err)
	}

	for path, want := range ref.Notes {
		note, ok := idx.Get(vault.CanonicalPath(path))
		if !ok {
			t.Errorf("%s: ausente do nosso indice", path)
			continue
		}
		conjuntosIguais(t, path, "headings", headingsComparaveis(note.Headings), headingsDaReferencia(want.Headings))
		// O Obsidian separa a tag do corpo (`tags`) da tag do frontmatter
		// (`frontmatterTags`); nos guardamos as duas numa lista so. A uniao e o
		// que faz os dois lados falarem do mesmo conjunto — sem ela,
		// Apelidada.md (tags [], frontmatterTags [civil, civil/obrigacoes])
		// divergiria de um indice correto.
		conjuntosIguais(t, path, "tags", note.Tags, append(slices.Clone(want.Tags), want.FrontmatterTags...))
		conjuntosIguais(t, path, "blocks", blocosComparaveis(note.Blocks), want.Blocks)
		assertLinksMatch(t, path, note.Links, want.Links, want.Embeds)
	}

	// O grafo resolvido e o objeto da metrica; a presenca dos links, so o
	// primeiro passo. Uma referencia do dumper antigo nao consegue sustentar
	// esta comparacao — ver Reference.HasGraph.
	if !ref.HasGraph() {
		t.Logf("referencia no schema %d: comparacao de GRAFO pulada. "+
			"Recompile tools/parity-dumper/ e rode o dump de novo para verificar resolucao, "+
			"nao apenas presenca de link.", ref.Schema)
		return
	}
	assertGraphMatches(t, idx, ref)
}
