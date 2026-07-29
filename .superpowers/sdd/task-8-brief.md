### Task 8: Varredura, EOL, BOM, caminho longo e placeholder de nuvem

**Files:**
- Create: `internal/vault/vault.go`, `internal/vault/walk.go`, `internal/vault/eol.go`
- Create: `internal/vault/longpath_windows.go`, `internal/vault/longpath_other.go`
- Create: `internal/vault/cloud_windows.go`, `internal/vault/cloud_other.go`
- Create: `internal/vault/walk_test.go`, `internal/vault/eol_test.go`

**Interfaces:**
- Consumes: `vault.CanonicalPath`, `vault.Canonicalize` da Task 7
- Produces: `vault.Vault` com `Root() string`, `Open(CanonicalPath) (*os.File, error)`, `ReadRange(ctx, CanonicalPath, start, end int64) ([]byte, error)`, `ReadAll(ctx, CanonicalPath) ([]byte, error)`; `vault.New(root string) (*Vault, error)`; `vault.Entry{Path CanonicalPath, Size int64, ModTime time.Time, IsNote bool, CloudOnly bool}`; `(*Vault).Walk(ctx, func(Entry) error) error`; `vault.DetectEOL([]byte) EOLStyle`; `vault.StripBOM([]byte) (body []byte, hadBOM bool)`; `vault.LongPath(abs string) string`

- [ ] **Step 1: Escrever o teste de EOL e BOM**

`internal/vault/eol_test.go`:

```go
package vault_test

import (
	"bytes"
	"testing"

	"github.com/jonyd/gobsidian/internal/vault"
)

func TestDetectEOL(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want vault.EOLStyle
	}{
		{"lf puro", "a\nb\nc\n", vault.EOLLF},
		{"crlf puro", "a\r\nb\r\nc\r\n", vault.EOLCRLF},
		{"misto majoritariamente crlf", "a\r\nb\r\nc\n", vault.EOLCRLF},
		{"misto majoritariamente lf", "a\nb\nc\r\n", vault.EOLLF},
		{"sem quebra alguma", "linha unica", vault.EOLLF},
		{"vazio", "", vault.EOLLF},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := vault.DetectEOL([]byte(tt.in)); got != tt.want {
				t.Errorf("DetectEOL(%q) = %v, quer %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestStripBOM(t *testing.T) {
	withBOM := append([]byte{0xEF, 0xBB, 0xBF}, []byte("# Titulo\n")...)

	body, had := vault.StripBOM(withBOM)
	if !had {
		t.Error("StripBOM nao detectou o BOM")
	}
	if !bytes.Equal(body, []byte("# Titulo\n")) {
		t.Errorf("body = %q, quer %q", body, "# Titulo\n")
	}

	body, had = vault.StripBOM([]byte("# Titulo\n"))
	if had {
		t.Error("StripBOM reportou BOM inexistente")
	}
	if !bytes.Equal(body, []byte("# Titulo\n")) {
		t.Errorf("body = %q, inalterado esperado", body)
	}
}
```

**Por que o BOM importa.** Sem removê-lo na leitura, o primeiro heading do arquivo deixa de ser reconhecido — os três bytes ficam antes do `#` e o goldmark não vê um heading. Sem preservá-lo na escrita, cada gravação produz um diff espúrio em cofre versionado.

- [ ] **Step 2: Escrever o teste de varredura**

`internal/vault/walk_test.go`:

```go
package vault_test

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/jonyd/gobsidian/internal/vault"
)

func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

func TestWalkExcludesAndClassifies(t *testing.T) {
	root := t.TempDir()

	writeFile(t, root, "Civil/PONTO 03.md", "# A\n")
	writeFile(t, root, "Penal/B.md", "# B\n")
	writeFile(t, root, "Anexos/diagrama.png", "\x89PNG")
	writeFile(t, root, ".obsidian/workspace.json", "{}")
	writeFile(t, root, ".git/config", "[core]")
	writeFile(t, root, ".trash/velha.md", "# velha\n")
	writeFile(t, root, "desktop.ini", "[.ShellClassInfo]")
	writeFile(t, root, "~$temp.md", "lixo")
	writeFile(t, root, "notas.txt", "nao e nota")

	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	var notes, assets []string
	err = v.Walk(context.Background(), func(e vault.Entry) error {
		if e.IsNote {
			notes = append(notes, string(e.Path))
		} else {
			assets = append(assets, string(e.Path))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}

	sort.Strings(notes)
	sort.Strings(assets)

	wantNotes := []string{"Civil/PONTO 03.md", "Penal/B.md"}
	if len(notes) != len(wantNotes) {
		t.Fatalf("notas = %v, quer %v", notes, wantNotes)
	}
	for i := range wantNotes {
		if notes[i] != wantNotes[i] {
			t.Errorf("notas[%d] = %q, quer %q", i, notes[i], wantNotes[i])
		}
	}

	wantAssets := []string{"Anexos/diagrama.png"}
	if len(assets) != len(wantAssets) || assets[0] != wantAssets[0] {
		t.Errorf("anexos = %v, quer %v", assets, wantAssets)
	}
}

func TestNewRejectsMissingRoot(t *testing.T) {
	if _, err := vault.New(filepath.Join(t.TempDir(), "nao-existe")); err == nil {
		t.Fatal("New com raiz inexistente deveria falhar")
	}
}

func TestReadRange(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "A.md", "0123456789")

	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	got, err := v.ReadRange(context.Background(), "A.md", 2, 5)
	if err != nil {
		t.Fatalf("ReadRange: %v", err)
	}
	if string(got) != "234" {
		t.Errorf("ReadRange = %q, quer %q", got, "234")
	}
}
```

- [ ] **Step 3: Rodar para confirmar que falha**

Run: `go test ./internal/vault/ -v`
Esperado: FAIL — `undefined: vault.New`.

- [ ] **Step 4: Implementar EOL e BOM**

`internal/vault/eol.go`:

```go
package vault

import "bytes"

type EOLStyle int

const (
	EOLLF EOLStyle = iota
	EOLCRLF
)

func (e EOLStyle) String() string {
	if e == EOLCRLF {
		return "CRLF"
	}
	return "LF"
}

// Bytes devolve a sequencia de quebra de linha do estilo.
func (e EOLStyle) Bytes() []byte {
	if e == EOLCRLF {
		return []byte("\r\n")
	}
	return []byte("\n")
}

var bom = []byte{0xEF, 0xBB, 0xBF}

// DetectEOL devolve o estilo predominante do arquivo. Arquivos com mistura
// existem, e converter o arquivo inteiro para um estilo so seria reescrever
// o que o usuario nao pediu — a escrita normaliza apenas o conteudo novo.
func DetectEOL(data []byte) EOLStyle {
	crlf := bytes.Count(data, []byte("\r\n"))
	lf := bytes.Count(data, []byte("\n")) - crlf

	if crlf > lf {
		return EOLCRLF
	}
	return EOLLF
}

// StripBOM remove o marcador UTF-8 do inicio, reportando se ele estava la.
func StripBOM(data []byte) ([]byte, bool) {
	if bytes.HasPrefix(data, bom) {
		return data[len(bom):], true
	}
	return data, false
}

// AddBOM reintroduz o marcador. Usado na escrita, quando o arquivo original
// o tinha.
func AddBOM(data []byte) []byte {
	if bytes.HasPrefix(data, bom) {
		return data
	}
	return append(append([]byte{}, bom...), data...)
}

// NormalizeEOL converte o conteudo novo para o estilo do arquivo alvo.
func NormalizeEOL(data []byte, style EOLStyle) []byte {
	flat := bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
	if style == EOLLF {
		return flat
	}
	return bytes.ReplaceAll(flat, []byte("\n"), []byte("\r\n"))
}
```

- [ ] **Step 5: Implementar caminho longo e detecção de nuvem**

`internal/vault/longpath_windows.go`:

```go
//go:build windows

package vault

import (
	"path/filepath"
	"strings"
)

// longPathThreshold e conservador de proposito: o limite e 260, e a folga
// cobre o nome do arquivo temporario que a escrita atomica cria ao lado do
// alvo.
const longPathThreshold = 240

// LongPath prefixa com \\?\ quando o caminho se aproxima de MAX_PATH.
//
// Restricoes do prefixo, que o chamador precisa ter respeitado antes: exige
// caminho absoluto, exige separador "\", e nao aceita "." nem "..".
func LongPath(abs string) string {
	if len(abs) < longPathThreshold {
		return abs
	}
	if strings.HasPrefix(abs, `\\?\`) {
		return abs
	}
	clean := filepath.Clean(abs)
	if strings.HasPrefix(clean, `\\`) {
		return `\\?\UNC\` + strings.TrimPrefix(clean, `\\`)
	}
	return `\\?\` + clean
}
```

`internal/vault/longpath_other.go`:

```go
//go:build !windows

package vault

// LongPath e identidade fora do Windows: nao ha MAX_PATH a contornar.
func LongPath(abs string) string { return abs }
```

`internal/vault/cloud_windows.go`:

```go
//go:build windows

package vault

import "golang.org/x/sys/windows"

// Atributos que indicam arquivo nao hidratado pelo sincronizador de nuvem.
// Ler um arquivo assim dispara download sincrono, que pode levar segundos ou
// falhar sem conexao — e uma indexacao ingenua forcaria o download do cofre
// inteiro no boot.
const (
	attrRecallOnDataAccess = 0x00400000
	attrRecallOnOpen       = 0x00040000
)

// IsCloudOnly consulta os atributos sem abrir o arquivo, o que e o ponto:
// abrir e exatamente o que dispara a hidratacao.
func IsCloudOnly(abs string) bool {
	p, err := windows.UTF16PtrFromString(LongPath(abs))
	if err != nil {
		return false
	}
	attrs, err := windows.GetFileAttributes(p)
	if err != nil {
		return false
	}
	const offline = windows.FILE_ATTRIBUTE_OFFLINE
	return attrs&(attrRecallOnDataAccess|attrRecallOnOpen|offline) != 0
}
```

`internal/vault/cloud_other.go`:

```go
//go:build !windows

package vault

// IsCloudOnly e sempre falso fora do Windows. Dropbox e Google Drive tem
// mecanismos equivalentes em macOS, mas a deteccao deles nao e por atributo
// de arquivo e fica fora da v1.
func IsCloudOnly(string) bool { return false }
```

- [ ] **Step 6: Implementar `Vault` e a varredura**

`internal/vault/vault.go`:

```go
package vault

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type Vault struct {
	root string
}

func New(root string) (*Vault, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolvendo raiz do cofre %q: %w", root, err)
	}
	info, err := os.Stat(LongPath(abs))
	if err != nil {
		return nil, fmt.Errorf("raiz do cofre inacessivel %q: %w", abs, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("raiz do cofre nao e diretorio: %q", abs)
	}
	return &Vault{root: abs}, nil
}

func (v *Vault) Root() string { return v.root }

// Abs devolve o caminho absoluto de um caminho canonico, ja preparado para
// chamadas do sistema.
func (v *Vault) Abs(p CanonicalPath) string {
	return LongPath(filepath.Join(v.root, filepath.FromSlash(string(p))))
}

func (v *Vault) Open(p CanonicalPath) (*os.File, error) {
	f, err := os.Open(v.Abs(p))
	if err != nil {
		return nil, fmt.Errorf("abrindo %q: %w", p, err)
	}
	return f, nil
}

// ReadRange le apenas a faixa pedida. Uma secao de 2 KB em uma nota de 500 KB
// custa 2 KB — e a razao de o indice guardar offsets em vez de conteudo.
func (v *Vault) ReadRange(ctx context.Context, p CanonicalPath, start, end int64) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if end < start {
		return nil, fmt.Errorf("faixa invalida em %q: %d..%d", p, start, end)
	}

	f, err := v.Open(p)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	buf := make([]byte, end-start)
	n, err := f.ReadAt(buf, start)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("lendo %q em %d..%d: %w", p, start, end, err)
	}
	return buf[:n], nil
}

func (v *Vault) ReadAll(ctx context.Context, p CanonicalPath) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(v.Abs(p))
	if err != nil {
		return nil, fmt.Errorf("lendo %q: %w", p, err)
	}
	return data, nil
}
```

`internal/vault/walk.go`:

```go
package vault

import (
	"context"
	"io/fs"
	"path/filepath"
	"strings"
	"time"
)

type Entry struct {
	Path      CanonicalPath
	Size      int64
	ModTime   time.Time
	IsNote    bool
	CloudOnly bool
}

// excludedDirs sao podados na varredura, nunca descendidos.
var excludedDirs = map[string]bool{
	".obsidian": true,
	".git":      true,
	".trash":    true,
	".stfolder": true,
}

// assetExts sao indexados por nome, nunca lidos (PRD RF-60). Sem eles, todo
// embed de imagem vira link quebrado e vault_stats fica dominado por falsos
// positivos.
var assetExts = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true,
	".webp": true, ".svg": true, ".bmp": true,
	".pdf": true, ".canvas": true,
	".mp3": true, ".wav": true, ".m4a": true, ".ogg": true,
	".mp4": true, ".webm": true, ".mov": true,
}

// Arquivos de controle de sincronizador e de editor, que geram evento e nunca
// interessam (WINDOWS.md §1.4).
func isNoise(name string) bool {
	lower := strings.ToLower(name)
	switch {
	case strings.HasPrefix(name, "~$"):
		return true
	case strings.HasPrefix(lower, ".~lock."):
		return true
	case lower == "desktop.ini", lower == "thumbs.db", lower == ".ds_store":
		return true
	case strings.HasSuffix(lower, ".tmp"):
		return true
	case strings.HasPrefix(name, ".gobsidian-tmp-"):
		return true
	}
	return false
}

// Walk percorre o cofre aplicando as exclusoes e classificando cada arquivo
// como nota ou anexo. Arquivos que nao sao nem um nem outro sao ignorados.
func (v *Vault) Walk(ctx context.Context, fn func(Entry) error) error {
	return filepath.WalkDir(v.root, func(abs string, d fs.DirEntry, err error) error {
		if err != nil {
			// Um diretorio ilegivel nao derruba a varredura inteira. O cofre
			// e do usuario, e uma pasta com ACL estranha e problema local.
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}

		name := d.Name()

		if d.IsDir() {
			if abs == v.root {
				return nil
			}
			if excludedDirs[strings.ToLower(name)] {
				return fs.SkipDir
			}
			return nil
		}

		if isNoise(name) {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(name))
		isNote := ext == ".md"
		isAsset := assetExts[ext]
		if !isNote && !isAsset {
			return nil
		}

		canon, cErr := Canonicalize(v.root, abs)
		if cErr != nil {
			return nil
		}
		info, iErr := d.Info()
		if iErr != nil {
			return nil
		}

		return fn(Entry{
			Path:      canon,
			Size:      info.Size(),
			ModTime:   info.ModTime(),
			IsNote:    isNote,
			CloudOnly: IsCloudOnly(abs),
		})
	})
}
```

- [ ] **Step 7: Rodar para confirmar que passa**

Run: `go test -race ./internal/vault/ -v`
Esperado: PASS. `TestWalkExcludesAndClassifies` deve ver exatamente duas notas e um anexo.

- [ ] **Step 8: Commit**

```bash
git add internal/vault
git commit -m "feat(vault): walk with exclusions, EOL and BOM detection, long paths, cloud placeholders"
```

---

