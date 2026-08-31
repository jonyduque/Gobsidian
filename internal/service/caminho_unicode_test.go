package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// TestCaminhoUnicodeIdaEVolta e o teste CRUZADO: criar numa forma de
// normalizacao e ler na OUTRA. Criar e ler com a mesma string nao prova nada —
// as duas pontas derivam a chave do mesmo texto.
//
// A reindexacao e um Service NOVO sobre a mesma raiz, nunca time.Sleep
// esperando watcher, que mediria a maquina.
func TestCaminhoUnicodeIdaEVolta(t *testing.T) {
	const nfc = "Cap\u00edtulo I.md"  // í precomposto, U+00ED
	const nfd = "Capi\u0301tulo I.md" // i + acento combinante, U+0301

	casos := []struct {
		nome string
		cria string
		le   string
	}{
		{"travessao", "Nota \u2014 com travessao.md", "Nota \u2014 com travessao.md"},
		{"nfc ida e volta", nfc, nfc},
		{"nfd ida e volta", nfd, nfd},
		{"cria em NFD, le em NFC", nfd, nfc},
		{"cria em NFC, le em NFD", nfc, nfd},
		{"emoji fora do BMP", "Nota \U0001F600.md", "Nota \U0001F600.md"},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			root := t.TempDir()
			svc := newTestService(t, root)

			if _, err := svc.CreateNote(context.Background(), CreateNoteRequest{
				Path:    c.cria,
				Content: "conteudo",
			}); err != nil {
				t.Fatalf("CreateNote(%q): %v", c.cria, err)
			}

			// Reindexa de forma deterministica: Service novo sobre a mesma raiz.
			svc = newTestService(t, root)

			res, err := svc.ReadNote(context.Background(), ReadRequest{Path: c.le})
			if err != nil {
				t.Fatalf("criou %q e ReadNote(%q) falhou: %v", c.cria, c.le, err)
			}
			if res.Content != "conteudo" {
				t.Fatalf("conteudo = %q, quer %q", res.Content, "conteudo")
			}
		})
	}
}

// TestCaminhoUnicodeSobreviveAoMove cobre a segunda superficie pelo mesmo
// mecanismo: note_move constroi CanonicalPath para o DESTINO, e e o outro lugar
// onde a chave nasce.
func TestCaminhoUnicodeSobreviveAoMove(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "origem.md", "conteudo\n")
	svc := newTestService(t, root)

	const destino = "Cap\u00edtulo \u2014 I.md"
	if _, err := svc.MoveNote(context.Background(), MoveNoteRequest{
		From: "origem.md", To: destino,
	}); err != nil {
		t.Fatalf("MoveNote para %q: %v", destino, err)
	}

	svc = newTestService(t, root)
	if _, err := svc.ReadNote(context.Background(), ReadRequest{Path: destino}); err != nil {
		t.Fatalf("moveu para %q e ReadNote falhou: %v", destino, err)
	}
}

// TestCaminhoUnicodePorCaminhoCompletoComHomonimo isola a rota de lowerPath.
//
// ResolvePath tem tres rotas, e na raiz do cofre o nome de arquivo E o caminho
// inteiro — entao byName responde e mascara a regra. Uma prova de mutacao saiu
// EXIT=1 por isso. Com dois homonimos, byName vira ambiguo e sobra o caminho
// completo, que so casa entre formas diferentes se lowerPath normalizar.
func TestCaminhoUnicodePorCaminhoCompletoComHomonimo(t *testing.T) {
	const nfc = "Cap\u00edtulo I.md"
	const nfd = "Capi\u0301tulo I.md"

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "a"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "b"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, root, "a/"+nfd, "conteudo de a\n")
	writeFile(t, root, "b/"+nfd, "conteudo de b\n")
	svc := newTestService(t, root)

	// Controle: o nome nu e ambiguo, entao byName nao pode responder.
	if _, err := svc.ReadNote(context.Background(), ReadRequest{Path: nfc}); err == nil {
		t.Fatal("nome nu com dois homonimos resolveu; a fixture nao monta o teste")
	}

	// A asserção: caminho completo gravado em NFD, pedido em NFC.
	res, err := svc.ReadNote(context.Background(), ReadRequest{Path: "a/" + nfc})
	if err != nil {
		t.Fatalf("gravou %q e ReadNote(%q) falhou: %v\n"+
			"lowerPath nao normaliza: caminho completo e a unica rota aqui", "a/"+nfd, "a/"+nfc, err)
	}
	if res.Content != "conteudo de a\n" {
		t.Fatalf("resolveu para a nota errada: %q", res.Content)
	}
}

// TestCaminhoUnicodePorNomeNuEmSubpasta isola a rota de byName, que e a outra
// metade: aqui o caminho completo NAO pode responder, porque quem pergunta so
// tem o nome do arquivo e a nota vive numa subpasta.
func TestCaminhoUnicodePorNomeNuEmSubpasta(t *testing.T) {
	const nfc = "Cap\u00edtulo I.md"
	const nfd = "Capi\u0301tulo I.md"

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "livro"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, root, "livro/"+nfd, "conteudo\n")
	svc := newTestService(t, root)

	if _, err := svc.ReadNote(context.Background(), ReadRequest{Path: nfc}); err != nil {
		t.Fatalf("gravou livro/%q e ReadNote(%q) falhou: %v\n"+
			"byName nao normaliza: nome nu em subpasta e a unica rota aqui", nfd, nfc, err)
	}
}
