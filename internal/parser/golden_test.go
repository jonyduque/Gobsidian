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

	assertNoOrphanGoldens(t, root, inputs)

	for _, in := range inputs {
		name, _ := filepath.Rel(root, in)
		t.Run(name, func(t *testing.T) {
			src, err := os.ReadFile(in)
			if err != nil {
				t.Fatalf("lendo entrada: %v", err)
			}

			note := parser.Parse(src)

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

// assertNoOrphanGoldens reprova quando existe um .json sem o .md
// correspondente.
//
// A varredura acima enumera entradas, nao goldens, entao apagar um .md deixa
// seu .json orfao e a suite verde: o caso simplesmente para de ser exercitado,
// sem nada dizer. O mesmo vale para um diretorio inteiro de fixtures removido
// por acidente — o unico sinal hoje seria a contagem total chegar a zero.
//
// Cobertura que desaparece em silencio e pior que cobertura ausente, porque
// ninguem vai procurar por ela.
func assertNoOrphanGoldens(t *testing.T, root string, inputs []string) {
	t.Helper()

	expected := make(map[string]bool, len(inputs))
	for _, in := range inputs {
		expected[strings.TrimSuffix(in, ".md")+".json"] = true
	}

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".json") {
			return nil
		}
		if !expected[path] {
			rel, _ := filepath.Rel(root, path)
			t.Errorf("golden orfao: %s nao tem .md correspondente — "+
				"a entrada foi apagada e o caso parou de ser testado em silencio", rel)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("varrendo goldens: %v", err)
	}
}
