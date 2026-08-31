package index_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/vault"
)

// TestResolvePathNomeNuIgnoraCaixa fecha uma incoerencia: `pasta/ACORDAO.MD`
// resolvia e `acordao` nao. As duas fazem a MESMA pergunta por portas
// diferentes — lowerPath baixava a caixa, byName comparava a base crua.
func TestResolvePathNomeNuIgnoraCaixa(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "pasta/Acordao.md", "# A\n")

	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	ix := index.New()
	if err := ix.Build(context.Background(), v); err != nil {
		t.Fatalf("Build: %v", err)
	}

	// O caminho completo em caixa trocada e o CONTRAPESO: ele ja funcionava, e
	// a correcao do nome nu nao pode te-lo quebrado.
	for _, entrada := range []string{
		"pasta/ACORDAO.MD", "PASTA/acordao.md",
		"acordao", "ACORDAO", "acordao.md", "AcOrDaO.Md",
	} {
		got, err := ix.ResolvePath(entrada)
		if err != nil {
			t.Errorf("ResolvePath(%q) = erro %v; a nota existe", entrada, err)
			continue
		}
		if got != "pasta/Acordao.md" {
			t.Errorf("ResolvePath(%q) = %q, queria pasta/Acordao.md", entrada, got)
		}
	}
}

// TestResolvePathColisaoDeChaveENomeada e o preco da metade acima: baixar a
// caixa cria colisoes novas, e colisao tem de virar ErrAmbiguousPath — nunca
// uma das duas escolhida em silencio. Pastas distintas porque o NTFS nao aceita
// as duas grafias na mesma.
func TestResolvePathColisaoDeChaveENomeada(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "civil/Prescricao.md", "# Civil\n")
	writeFile(t, root, "penal/prescricao.md", "# Penal\n")

	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	ix := index.New()
	if err := ix.Build(context.Background(), v); err != nil {
		t.Fatalf("Build: %v", err)
	}

	if _, err := ix.ResolvePath("prescricao"); !errors.Is(err, index.ErrAmbiguousPath) {
		t.Errorf("nome nu com duas notas so diferindo na caixa: err = %v, queria ErrAmbiguousPath\n"+
			"escolher uma das duas devolve caminho arbitrario com cara de resposta", err)
	}

	// E o caminho exato continua desempatando: ele nunca perde para o nome.
	for _, c := range []struct{ entrada, quer string }{
		{"civil/Prescricao.md", "civil/Prescricao.md"},
		{"penal/prescricao.md", "penal/prescricao.md"},
		{"CIVIL/PRESCRICAO.MD", "civil/Prescricao.md"},
	} {
		got, err := ix.ResolvePath(c.entrada)
		if err != nil || string(got) != c.quer {
			t.Errorf("ResolvePath(%q) = (%q, %v), queria %q", c.entrada, got, err, c.quer)
		}
	}
}

// TestColisaoDeNormalizacaoNaMesmaPasta cobre as duas metades do mapa de lista.
//
// `Capítulo.md` em NFC e em NFD sao DOIS arquivos no NTFS — 12 e 13 bytes — e
// UMA chave no indice. Com valor unico, a segunda tomava o lugar da primeira e
// remover qualquer uma apagava a entrada da outra.
//
// As duas ficam na MESMA pasta de proposito: em pastas diferentes as chaves
// diferem pelo diretorio e nao ha colisao. Uma versao anterior errou nisso e
// passou com a regra mutada.
func TestColisaoDeNormalizacaoNaMesmaPasta(t *testing.T) {
	const nfc = "Cap\u00edtulo.md"  // í precomposto
	const nfd = "Capi\u0301tulo.md" // i + acento combinante

	root := t.TempDir()
	writeFile(t, root, "livro/"+nfc, "# NFC"+"\n")
	writeFile(t, root, "livro/"+nfd, "# NFD"+"\n")

	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	ix := index.New()
	if err := ix.Build(context.Background(), v); err != nil {
		t.Fatalf("Build: %v", err)
	}

	// Controle: sao duas notas distintas para o indice, nao uma.
	if _, ok := ix.Get(vault.CanonicalPath("livro/" + nfc)); !ok {
		t.Fatal("a nota em NFC nao entrou no indice; a fixture nao monta o teste")
	}
	if _, ok := ix.Get(vault.CanonicalPath("livro/" + nfd)); !ok {
		t.Fatal("a nota em NFD nao entrou no indice; a fixture nao monta o teste")
	}

	// Metade 1: a chave colidida NAO escolhe uma das duas em silencio.
	//
	// A entrada e o caminho completo em caixa trocada, que erra o mapa exato de
	// proposito: assim a pergunta so pode ser respondida por lowerPath.
	if _, err := ix.ResolvePath("LIVRO/" + nfc); !errors.Is(err, index.ErrAmbiguousPath) {
		t.Errorf("duas notas na mesma chave: err = %v, queria ErrAmbiguousPath"+"\n"+
			"devolver uma delas e um caminho arbitrario com cara de resposta", err)
	}

	// Metade 2: removida uma, a outra continua resolvivel pela MESMA chave.
	// Com o mapa de valor unico, as duas dividiam uma entrada e a remocao a
	// apagava inteira.
	ix.Remove(vault.CanonicalPath("livro/" + nfc))

	got, err := ix.ResolvePath("LIVRO/" + nfc)
	if err != nil {
		t.Fatalf("removi a nota em NFC e a em NFD parou de resolver: %v"+"\n"+
			"as duas dividiam uma entrada de lowerPath", err)
	}
	if string(got) != "livro/"+nfd {
		t.Errorf("resolveu para %q, queria livro/%q", got, nfd)
	}
}
