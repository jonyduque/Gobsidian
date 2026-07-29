### Task 18: Golden files do parser

**Files:**
- Create: `internal/parser/golden_test.go`
- Create: `testdata/parser/**` (pares `.md` de entrada e `.json` esperado)

**Interfaces:**
- Consumes: `parser.Parse` (Tasks 12–17)
- Produces: suíte regenerável com `go test ./internal/parser -update`

- [ ] **Step 1: Escrever o harness**

`internal/parser/golden_test.go`:

```go
package parser_test

import (
	"encoding/json"
	"flag"
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
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
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
```

Adicione `"io/fs"` se o compilador reclamar da assinatura de `WalkDir`; a entrada correta é `fs.DirEntry`.

- [ ] **Step 2: Criar o corpus**

Um arquivo `.md` por caso, nas pastas de `docs/ESTRUTURA.md`. Mínimo exigido:

| Pasta | Arquivos |
|---|---|
| `wikilinks/` | `simples.md`, `alias.md`, `heading.md`, `bloco.md`, `heading_alias.md`, `caminho.md`, `embed.md`, `embed_imagem.md`, `multiplos.md` |
| `codeblocks/` | `cercado.md`, `cercado_lang.md`, `inline.md`, `indentado.md`, `til.md`, `escapado.md`, `colchete_literal.md` — **todos devem produzir zero links** |
| `frontmatter/` | `completo.md`, `vazio.md`, `ausente.md`, `malformado.md`, `tags_string.md`, `tags_lista.md`, `aliases.md`, `nao_fechado.md` |
| `headings/` | `hierarquia.md`, `acentos.md`, `fechamento.md`, `duplicados.md`, `nivel_pulado.md` |
| `blocks/` | `simples.md`, `multiplos.md`, `em_lista.md`, `invalido.md` |
| `edge/` | `vazio.md`, `sem_newline_final.md`, `crlf.md`, `crlf_misto.md`, `bom.md`, `so_frontmatter.md`, `nota_gigante.md` |

`edge/bom.md` precisa ser gravado com os três bytes `EF BB BF` no início:

```powershell
$Bytes = [byte[]](0xEF,0xBB,0xBF) + [System.Text.Encoding]::UTF8.GetBytes("# Titulo`n`nTexto.`n")
[System.IO.File]::WriteAllBytes("testdata\parser\edge\bom.md", $Bytes)
```

`edge/crlf.md` e `crlf_misto.md` precisam de CRLF real, não convertido pelo Git. Adicione ao repositório:

`.gitattributes`:

```
testdata/parser/edge/crlf.md       -text
testdata/parser/edge/crlf_misto.md -text
testdata/parser/edge/bom.md        -text binary
```

Sem isso, `core.autocrlf` no Windows normaliza os arquivos no checkout e os casos que eles existem para testar deixam de existir.

- [ ] **Step 3: Gerar os golden files**

Run: `go test ./internal/parser -update`
Depois: **leia cada `.json` gerado.** O `-update` grava o que o código produz, não o que está certo; aceitar sem ler transforma o teste em tautologia. Cada arquivo em `codeblocks/` deve ter `links` ausente ou vazio.

- [ ] **Step 4: Rodar a suíte**

Run: `go test -race ./internal/parser/ -v`
Esperado: PASS em todos os casos.

- [ ] **Step 5: Commit**

```bash
git add internal/parser testdata/parser .gitattributes
git commit -m "test(parser): golden file corpus covering documented edge cases"
```

---

