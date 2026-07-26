package vault_test

import (
	"errors"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/jonyd/gobsidian/internal/vault"
)

// testRoot devolve uma raiz sintetica valida na plataforma corrente. Nenhuma
// funcao deste pacote toca o disco, entao a raiz nao precisa existir — mas
// precisa ter a FORMA certa: "C:/cofre" e um caminho relativo com dois-pontos
// no Linux, e testar contra ele nao testa nada.
func testRoot() string {
	if runtime.GOOS == "windows" {
		return filepath.FromSlash("C:/cofre")
	}
	return filepath.FromSlash("/cofre")
}

func TestResolveConfinement(t *testing.T) {
	root := testRoot()

	tests := []struct {
		name        string
		input       string
		wantErr     error // sentinela esperada; nil significa sucesso
		want        vault.CanonicalPath
		windowsOnly bool
	}{
		{name: "simples", input: "Civil/PONTO 03.md", want: "Civil/PONTO 03.md"},
		{name: "ponto inicial", input: "./Civil/PONTO 03.md", want: "Civil/PONTO 03.md"},
		{name: "duplo ponto interno", input: "Civil/../Penal/A.md", want: "Penal/A.md"},
		{name: "componente vazio colapsa", input: "Civil//PONTO 03.md", want: "Civil/PONTO 03.md"},
		{name: "ponto interno colapsa", input: "Civil/./PONTO 03.md", want: "Civil/PONTO 03.md"},

		// A conversao de barra invertida so vale no Windows: em Unix a barra
		// invertida e um byte legitimo de nome de arquivo, e converte-la
		// corromperia nomes reais.
		{name: "barra invertida", input: `Civil\PONTO 03.md`, want: "Civil/PONTO 03.md", windowsOnly: true},

		{name: "escapa do cofre", input: "../outro/A.md", wantErr: vault.ErrOutsideVault},
		{name: "escapa com muitos niveis", input: "a/b/../../../fora.md", wantErr: vault.ErrOutsideVault},
		{name: "duplo ponto sozinho", input: "..", wantErr: vault.ErrOutsideVault},
		{name: "absoluto unix rejeitado", input: "/etc/passwd", wantErr: vault.ErrAbsolutePath},
		{name: "vazio rejeitado", input: "", wantErr: vault.ErrEmptyPath},
		{name: "so separadores", input: "/", wantErr: vault.ErrAbsolutePath},

		// Letra de drive que sobrevive ao Clean. Uma checagem que olha apenas
		// os dois primeiros bytes da entrada crua nao ve esta forma.
		{name: "drive apos ponto inicial", input: "./C:/Windows/win.ini", wantErr: vault.ErrAbsolutePath, windowsOnly: true},
		{name: "absoluto com drive", input: "C:/cofre/A.md", wantErr: vault.ErrAbsolutePath, windowsOnly: true},

		// Nomes de dispositivo. Nenhum contem "..", entao a comparacao por
		// componente os aceita — e o sistema operacional os resolve para fora
		// do sistema de arquivos.
		{name: "dispositivo NUL", input: "NUL", wantErr: vault.ErrInvalidPath, windowsOnly: true},
		{name: "dispositivo em subpasta", input: "Civil/NUL", wantErr: vault.ErrInvalidPath, windowsOnly: true},
		{name: "dispositivo COM1", input: "COM1", wantErr: vault.ErrInvalidPath, windowsOnly: true},
		{name: "dispositivo CON", input: "CON", wantErr: vault.ErrInvalidPath, windowsOnly: true},

		// Identidade canonica unica por arquivo.
		{name: "componente termina em ponto", input: "Civil/A.md.", wantErr: vault.ErrInvalidPath, windowsOnly: true},
		{name: "componente termina em espaco", input: "Civil/A.md ", wantErr: vault.ErrInvalidPath, windowsOnly: true},
		{name: "pasta termina em ponto", input: "Civil./A.md", wantErr: vault.ErrInvalidPath, windowsOnly: true},

		{name: "byte nulo", input: "Civil/\x00A.md", wantErr: vault.ErrInvalidPath},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.windowsOnly && runtime.GOOS != "windows" {
				t.Skip("comportamento especifico do Windows")
			}

			abs, canon, err := vault.Resolve(root, tt.input)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("Resolve(%q) = %q, quer erro %v", tt.input, canon, tt.wantErr)
				}
				// A sentinela importa: cada uma vira um codigo de erro MCP
				// diferente, e um teste que so verifica "err != nil" nao
				// notaria as quatro colapsando em uma.
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("Resolve(%q) erro = %v, quer %v", tt.input, err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("Resolve(%q) error = %v", tt.input, err)
			}
			if canon != tt.want {
				t.Errorf("Resolve(%q) = %q, quer %q", tt.input, canon, tt.want)
			}

			// abs e o valor que vai para os.Open. Sem esta assercao a
			// implementacao poderia devolver qualquer caminho do disco e
			// todos os casos continuariam passando.
			wantAbs := filepath.Join(root, filepath.FromSlash(string(tt.want)))
			if abs != wantAbs {
				t.Errorf("Resolve(%q) abs = %q, quer %q", tt.input, abs, wantAbs)
			}
		})
	}
}

// Canonicalize e exportada e a varredura da Task 8 a chama direto, sem passar
// por Resolve. Suas rejeicoes precisam valer sozinhas.
func TestCanonicalizeRejectsStandalone(t *testing.T) {
	root := testRoot()

	tests := []struct {
		name        string
		abs         string
		wantErr     error
		windowsOnly bool
	}{
		{
			name:    "irmao com prefixo compartilhado",
			abs:     filepath.Join(filepath.Dir(root), "cofre-outro", "A.md"),
			wantErr: vault.ErrOutsideVault,
		},
		{
			name:    "acima da raiz",
			abs:     filepath.Dir(root),
			wantErr: vault.ErrOutsideVault,
		},
		{
			name:    "a propria raiz",
			abs:     root,
			wantErr: vault.ErrInvalidPath,
		},
		{
			// Em Unix, "C:Windows" e um nome de diretorio comum e o caminho
			// continua dentro do cofre — aceita-lo la e o comportamento certo.
			name:        "letra de drive dentro da raiz",
			abs:         filepath.Join(root, "C:Windows", "win.ini"),
			wantErr:     vault.ErrAbsolutePath,
			windowsOnly: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.windowsOnly && runtime.GOOS != "windows" {
				t.Skip("comportamento especifico do Windows")
			}

			canon, err := vault.Canonicalize(root, tt.abs)
			if err == nil {
				t.Fatalf("Canonicalize(%q) = %q, quer erro %v", tt.abs, canon, tt.wantErr)
			}
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Canonicalize(%q) erro = %v, quer %v", tt.abs, err, tt.wantErr)
			}
		})
	}
}

// O bug classico: comparacao por prefixo de string aceita um irmao cujo nome
// comeca com o nome da raiz. A comparacao tem que ser por componente.
//
// Chegando por Resolve, este caso e barrado antes de Canonicalize: o `..`
// inicial nao sobrevive a validateLocal. As duas camadas cobrem o mesmo
// ataque de angulos diferentes, e o teste que isola a comparacao por
// componente e TestCanonicalizeRejectsStandalone, que chama Canonicalize
// direto. Ele e o que reprova uma implementacao com strings.HasPrefix; este
// aqui garante que o caminho completo tambem rejeita.
func TestResolveRejectsSiblingWithSharedPrefix(t *testing.T) {
	root := testRoot()
	_, _, err := vault.Resolve(root, "../cofre-outro/A.md")
	if !errors.Is(err, vault.ErrOutsideVault) {
		t.Fatalf("erro = %v, quer ErrOutsideVault", err)
	}
}

func TestCanonicalizeUsesForwardSlashes(t *testing.T) {
	root := testRoot()
	abs := filepath.Join(root, "Civil", "PONTO 03.md")

	got, err := vault.Canonicalize(root, abs)
	if err != nil {
		t.Fatalf("Canonicalize error = %v", err)
	}
	if got != "Civil/PONTO 03.md" {
		t.Errorf("Canonicalize = %q, quer %q", got, "Civil/PONTO 03.md")
	}
}
