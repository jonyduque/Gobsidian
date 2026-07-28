package index

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
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

func loadReference(t *testing.T, path string) map[string]RefNote {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("os.Open(%q): %v", path, err)
	}
	defer f.Close()

	var ref map[string]RefNote
	if err := json.NewDecoder(f).Decode(&ref); err != nil {
		t.Fatalf("json.Decode: %v", err)
	}
	return ref
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

func assertLinksMatch(t *testing.T, path string, gotLinks []ResolvedLink, wantLinks []RefLink, wantEmbeds []RefEmbed) {
	t.Helper()
	for _, w := range wantLinks {
		found := false
		for _, g := range gotLinks {
			if g.Target == w.Link && g.Kind != parser.LinkEmbed {
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
			if g.Target == w.Link && g.Kind == parser.LinkEmbed {
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
	if _, err := os.Stat(root); os.IsNotExist(err) {
		t.Skip("corpus de paridade ausente; ver tools/parity-dumper/README.md")
	}

	ref := loadReference(t, filepath.Join("..", "..", "testdata", "parity", "metadata.json"))

	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	idx := New()
	if err := idx.Build(context.Background(), v); err != nil {
		t.Fatalf("Build: %v", err)
	}

	for path, want := range ref {
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
}
