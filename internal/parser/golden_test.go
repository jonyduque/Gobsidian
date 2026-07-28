package parser_test

import (
	"encoding/json"
	"flag"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jonyd/gobsidian/internal/parser"
)

var update = flag.Bool("update", false, "regrava os golden files")

// Golden files tornam aceitar uma mudanca intencional de comportamento uma
// operacao de um comando, e tornam uma regressao acidental imediatamente
// visivel no diff.
func TestGolden(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "parser")

	var inputs []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(path, ".md") {
			inputs = append(inputs, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("varrendo testdata: %v", err)
	}
	if len(inputs) == 0 {
		t.Fatal("nenhum golden file encontrado — testdata/parser esta vazio")
	}

	for _, in := range inputs {
		name, _ := filepath.Rel(root, in)
		t.Run(name, func(t *testing.T) {
			src, err := os.ReadFile(in)
			if err != nil {
				t.Fatalf("lendo entrada: %v", err)
			}

			note, err := parser.Parse(src)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}

			got, err := json.MarshalIndent(note, "", "  ")
			if err != nil {
				t.Fatalf("serializando: %v", err)
			}
			got = append(got, '\n')

			goldenPath := strings.TrimSuffix(in, ".md") + ".json"

			if *update {
				if err := os.WriteFile(goldenPath, got, 0o644); err != nil {
					t.Fatalf("gravando golden: %v", err)
				}
				return
			}

			want, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatalf("lendo golden (rode com -update para criar): %v", err)
			}
			if string(got) != string(want) {
				t.Errorf("divergencia em %s\n--- esperado ---\n%s\n--- obtido ---\n%s", name, want, got)
			}
		})
	}
}
