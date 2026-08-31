package index_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/vault"
)

// TestResolvePathNomeNuIgnoraCaixa fecha a incoerencia que a sonda de 2026-08-31
// achou: `pasta/ACORDAO.MD` resolvia e `acordao` nao.
//
// As duas entradas fazem a MESMA pergunta — "qual nota e essa?" — por portas
// diferentes: caminho completo passa por lowerPath, nome nu passa por byName. A
// primeira baixava a caixa desde sempre; a segunda comparava a base crua.
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

// TestResolvePathColisaoDeChaveENomeada e a outra metade: baixar a caixa da
// chave de byName cria colisoes que antes nao existiam, e colisao tem de virar
// ErrAmbiguousPath — nunca uma das duas escolhida em silencio.
//
// Duas notas com o mesmo nome em pastas diferentes ja colidiam; o caso NOVO e
// caixa diferente. No NTFS os dois arquivos nao podem existir na MESMA pasta,
// entao a fixture os poe em pastas distintas, que e o cenario real de um cofre
// com "Civil/Prescricao.md" e "Penal/prescricao.md".
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
// `Capítulo.md` em NFC e em NFD sao DOIS arquivos no NTFS — conferido em
// 2026-08-31: 12 e 13 bytes, na mesma pasta — e UMA chave no indice. Com
// lowerPath sendo map[string]CanonicalPath, a segunda nota tomava o lugar da
// primeira em silencio, e remover qualquer uma apagava a entrada da outra.
//
// A fixture poe as duas na MESMA pasta de proposito: em pastas diferentes as
// chaves diferem pelo diretorio e a colisao nao existe. Uma versao anterior
// deste teste errou exatamente nisso e passou com a regra mutada.
//
// Medido nos quatro cofres reais: zero ocorrencias, todos NTFS e todos em NFC.
// Isto guarda o dia em que um deles abrir num Mac.
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
