package index

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/jonyd/gobsidian/internal/vault"
)

func TestChaveDeTagDobraCaixaHashENFC(t *testing.T) {
	// "Ação" em NFD: A + c + cedilha combinante + a + til combinante + o
	nfd := "Ação"
	casos := map[string]string{
		"#Projeto":    "projeto",
		"Projeto/Sub": "projeto/sub",
		"#" + nfd:     "ação",
		"ação":        "ação",
	}
	for in, quer := range casos {
		if got := ChaveDeTag(in); got != quer {
			t.Errorf("ChaveDeTag(%q) = %q, quero %q", in, got, quer)
		}
	}
}

func TestListPorTagCasaSubtagENFD(t *testing.T) {
	root := t.TempDir()
	escreve := func(nome, corpo string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, nome), []byte(corpo), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	escreve("a.md", "# A\n\n#Projeto/Alpha\n")
	escreve("b.md", "# B\n\n#projeto\n")
	escreve("c.md", "# C\n\n#outra\n")
	// A grafia NFD entra pelo FRONTMATTER, e nao por tag inline, e a razao
	// esta medida: o parser inline corta a tag no primeiro sinal combinante
	// — tagNameChar (parser/ext_tag.go) aceita letra, digito, hifen,
	// underscore e barra, e nao unicode.Mn —, entao "#Ação" em NFD indexa
	// como a tag "Ac". E defeito do parser, anterior a esta chave e fora do
	// alcance dela: nenhuma dobra de chave conserta uma tag que ja chegou
	// cortada. Pelo frontmatter o YAML entrega a string inteira, que e onde
	// a forma Unicode de fato varia entre um cofre escrito no macOS e um
	// escrito no Windows.
	escreve("d.md", "---\ntags: [\"Ação\"]\n---\n# D\n\ncorpo\n") // NFD
	v, err := vault.New(root)
	if err != nil {
		t.Fatal(err)
	}
	ix := New()
	if err := ix.Build(context.Background(), v); err != nil {
		t.Fatal(err)
	}
	got := ix.PathsComTags([]string{"#PROJETO"}, "all")
	if quer := []vault.CanonicalPath{"a.md", "b.md"}; !slices.Equal(got, quer) {
		t.Fatalf("PathsComTags(#PROJETO) = %v, quero %v", got, quer)
	}
	got = ix.PathsComTags([]string{"ação"}, "all") // pedido em NFC, nota em NFD
	if quer := []vault.CanonicalPath{"d.md"}; !slices.Equal(got, quer) {
		t.Fatalf("PathsComTags(ação) = %v, quero %v", got, quer)
	}
	got = ix.PathsComTags([]string{"projeto/alpha", "outra"}, "any")
	if quer := []vault.CanonicalPath{"a.md", "c.md"}; !slices.Equal(got, quer) {
		t.Fatalf("PathsComTags(any) = %v, quero %v", got, quer)
	}
	if got := ix.PathsComTags([]string{"projeto/alpha", "outra"}, "all"); len(got) != 0 {
		t.Fatalf("PathsComTags(all, disjuntas) = %v, quero vazio", got)
	}
	// Os dois nomes, cada um conferido por si. A forma anterior era
	// `len(tags) != 2 || tags[0].Tag != "projeto" && tags[1].Tag != "projeto"`,
	// e `&&` liga mais forte que `||`: bastava UM dos dois ser "projeto" para
	// passar, e "projeto/alpha" nunca era conferido.
	tags := ix.Tags("proj", 0)
	nomes := make([]string, 0, len(tags))
	for _, tc := range tags {
		nomes = append(nomes, tc.Tag)
	}
	slices.Sort(nomes)
	if !slices.Equal(nomes, []string{"projeto", "projeto/alpha"}) {
		t.Fatalf("Tags(proj) = %v, quero projeto e projeto/alpha dobradas", tags)
	}

	// O PREFIXO passa pela mesma conta da chave: '#' fora, caixa baixa, NFC.
	// Prefixo ASCII minusculo nao prova isso — em "proj", ChaveDeTag e o
	// strings.ToLower que ela substituiu devolvem a mesma coisa. Estes dois
	// separam as duas: um traz o '#' e a maiuscula, o outro o sinal
	// combinante.
	for _, prefixo := range []string{"#AÇ", "aç"} {
		acentuadas := ix.Tags(prefixo, 0)
		if len(acentuadas) != 1 || acentuadas[0].Tag != "ação" {
			t.Fatalf("Tags(%q) = %v, quero a entrada ação dobrada", prefixo, acentuadas)
		}
	}
}
