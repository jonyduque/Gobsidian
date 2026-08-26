package search

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jonyd/gobsidian/internal/vault"
)

// termoSonda é uma palavra que não aparece em nenhum outro lugar do corpus de
// teste. Se ela chegar ao índice, o arquivo foi lido — não há outra forma de
// ela entrar.
const termoSonda = "xyzzyplugh"

// TestUpdateNaoLeAnexo cobre a regra fechada "anexo é indexado por nome, NUNCA
// lido".
//
// O anexo deste teste tem conteúdo que TOKENIZARIA: texto ASCII com uma palavra
// única. Um anexo de bytes binários passaria a asserção por acidente, porque
// não produziria token nenhum de qualquer jeito — e o teste estaria medindo o
// conteúdo, não a guarda.
//
// A regra vale porque Inverted.Update é a função que abre o arquivo. Até
// 2026-08-26 ela guardava apenas contra placeholder de nuvem; anexo local
// seguia para os.ReadFile e tokenização. O dano não é só I/O: binário no índice
// infla DocCount e docLengths, que é o DIVISOR da normalização por tamanho do
// BM25 — ou seja, ranking errado — e o índice construído no boot (só notas)
// deixa de responder igual ao mantido por eventos, que é a família do defeito
// de DocLength que este projeto já pagou uma vez.
func TestUpdateNaoLeAnexo(t *testing.T) {
	dir := t.TempDir()
	anexo := filepath.Join(dir, "imagem.png")
	if err := os.WriteFile(anexo, []byte("# titulo\n\n"+termoSonda+" conteudo legivel.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	v, err := vault.New(dir)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	ix := NewInverted()

	if err := ix.Update(context.Background(), v, vault.CanonicalPath("imagem.png")); err != nil {
		t.Fatalf("Update: %v", err)
	}

	// 1. O anexo TEM de contar como coberto. Fora de docLengths ele nao entra
	//    em DocCount, o cabecalho do cache declara menos entradas do que o
	//    indice de metadados enxerga, e todo boot conclui "cache parcial" e
	//    regrava o cache inteiro — a armadilha irma da nota sem token nenhum.
	if !ix.HasDoc("imagem.png") {
		t.Error("o anexo nao ficou coberto; o cache concluira 'parcial' em todo boot")
	}

	// 2. E NAO pode ter sido lido.
	if postings := ix.Postings(termoSonda); len(postings) != 0 {
		t.Errorf("o anexo foi LIDO: o termo-sonda %q entrou no indice (%d postings)",
			termoSonda, len(postings))
	}
	if n := ix.DocLength("imagem.png"); n != 0 {
		t.Errorf("DocLength do anexo = %d, queria 0: bytes de anexo entraram no divisor do BM25", n)
	}
}

// TestUpdateNaoLeArquivoExcluido confirma que ruído e diretório excluído seguem
// o mesmo caminho do anexo. Sem isto, a guarda de classe poderia ter sido
// escrita como "se não for nota, mas for anexo" e deixar o resto passando.
func TestUpdateNaoLeArquivoExcluido(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".obsidian"), 0o755); err != nil {
		t.Fatal(err)
	}
	ruido := filepath.Join(dir, ".obsidian", "workspace.json")
	if err := os.WriteFile(ruido, []byte(termoSonda+" configuracao\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	v, err := vault.New(dir)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	ix := NewInverted()

	if err := ix.Update(context.Background(), v, vault.CanonicalPath(".obsidian/workspace.json")); err != nil {
		t.Fatalf("Update: %v", err)
	}

	if postings := ix.Postings(termoSonda); len(postings) != 0 {
		t.Errorf("arquivo excluido foi LIDO: %q entrou no indice", termoSonda)
	}
}

// TestUpdateLeNota é o contrapeso, e existe para que a correção não possa ser
// "não leia nada".
//
// Sem ele, uma guarda escrita larga demais passaria nos dois testes acima e
// desligaria a busca inteira em silêncio — trocando um defeito por outro muito
// pior.
func TestUpdateLeNota(t *testing.T) {
	dir := t.TempDir()
	nota := filepath.Join(dir, "nota.md")
	if err := os.WriteFile(nota, []byte("# Titulo\n\n"+termoSonda+" no corpo.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	v, err := vault.New(dir)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	ix := NewInverted()

	if err := ix.Update(context.Background(), v, vault.CanonicalPath("nota.md")); err != nil {
		t.Fatalf("Update: %v", err)
	}

	if postings := ix.Postings(termoSonda); len(postings) == 0 {
		t.Errorf("a nota NAO foi lida: %q nao entrou no indice; a guarda esta larga demais", termoSonda)
	}
	if n := ix.DocLength("nota.md"); n == 0 {
		t.Error("DocLength da nota = 0; a nota nao foi tokenizada")
	}
}
