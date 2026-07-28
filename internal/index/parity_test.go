package index

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
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
		return Reference{
			Schema:          v2.Schema,
			Notes:           v2.Notes,
			ResolvedLinks:   v2.ResolvedLinks,
			UnresolvedLinks: v2.UnresolvedLinks,
		}
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

		nossos := make(map[string]bool, len(n.Links))
		for _, l := range n.Links {
			if l.Resolved != "" {
				nossos[string(l.Resolved)] = true
			}
		}

		for alvo := range alvos {
			if !nossos[alvo] {
				t.Errorf("%s: o Obsidian resolve uma aresta para %q e nos nao; resolvemos %v",
					origem, alvo, keysOf(nossos))
			}
		}
	}
}

func keysOf(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func assertHeadingsContain(t *testing.T, path string, got []parser.Heading, want []RefHeading) {
	t.Helper()
	for _, w := range want {
		found := false
		for _, g := range got {
			if g.Text == w.Heading && g.Level == w.Level {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("%s: heading (level %d) %q ausente no nosso índice", path, w.Level, w.Heading)
		}
	}
}

func assertTagsContain(t *testing.T, path string, got []string, want []string) {
	t.Helper()
	for _, w := range want {
		found := false
		for _, g := range got {
			if g == w {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("%s: tag %q ausente no nosso índice", path, w)
		}
	}
}

func assertBlocksContain(t *testing.T, path string, got []parser.Block, want []string) {
	t.Helper()
	for _, w := range want {
		found := false
		for _, g := range got {
			if g.ID == w {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("%s: bloco %q ausente no nosso índice", path, w)
		}
	}
}

// linkKey reconstroi a forma que o Obsidian guarda, para que a comparacao
// compare a mesma coisa dos dois lados.
//
// O Obsidian registra o alvo COM a ancora junto — "Civil/PONTO 03#Cap 1" —
// enquanto nos separamos em Target e Anchor, que e a representacao melhor
// para resolver e reescrever. Comparar Target contra o link do Obsidian faz
// TODO link com ancora divergir, em qualquer cofre, e os achados verdadeiros
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
	for _, w := range wantLinks {
		found := false
		for _, g := range gotLinks {
			if linkKey(g) == w.Link && g.Kind != parser.LinkEmbed {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("%s: link %q ausente no nosso índice", path, w.Link)
		}
	}
	for _, w := range wantEmbeds {
		found := false
		for _, g := range gotLinks {
			if linkKey(g) == w.Link && g.Kind == parser.LinkEmbed {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("%s: embed %q ausente no nosso índice", path, w.Link)
		}
	}
}

func TestParityWithObsidian(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "parity", "vault")
	refPath := filepath.Join("..", "..", "testdata", "parity", "metadata.json")

	// Checar CONTEUDO, nao existencia. Um diretorio vazio e um metadata.json
	// com "{}" fazem o laco de comparacao nao executar nenhuma vez, e o teste
	// passa afirmando uma paridade que ninguem verificou.
	notes, _ := filepath.Glob(filepath.Join(root, "*.md"))
	sub, _ := filepath.Glob(filepath.Join(root, "*", "*.md"))
	if len(notes)+len(sub) == 0 {
		t.Skip("corpus de paridade vazio; ver tools/parity-dumper/README.md")
	}

	ref := loadReference(t, refPath)
	if len(ref.Notes) == 0 {
		t.Skip("referencia de paridade vazia; rode o plugin dumper — ver tools/parity-dumper/README.md")
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
		assertHeadingsContain(t, path, note.Headings, want.Headings)
		assertTagsContain(t, path, note.Tags, want.Tags)
		assertBlocksContain(t, path, note.Blocks, want.Blocks)
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
