package service_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jonyd/gobsidian/internal/service"
	"github.com/jonyd/gobsidian/internal/vault"
)

// Desde 2026-09-06 um link so de ancora resolve para a PROPRIA nota, e com isso
// a nota virou backlink de si mesma. note_move colhia esse backlink como se
// fosse um citante a reescrever: movia o corpo primeiro e so depois lia o
// caminho ANTIGO para reescrever os links, o que dava ENOENT e devolvia
// CodeInternal com o move ja pela metade — corpo movido, citantes intactos.
//
// O discriminante e Target == "", e NAO "o citante e a propria nota":
// auto-referencia com alvo escrito precisa continuar sendo reescrita. Ate a
// Task 185 este teste nao conseguia fixar essa segunda metade, porque
// "[[a]]" dentro de a.md derrubava note_move pelo MESMO ENOENT — pre-existente,
// independente da Task 182, e so exposto por ela como forma comum. A Task 185
// fechou o laco de reescrita para ler e gravar no caminho NOVO quando o
// citante e a propria nota movida, e agora "[[a]]" entra aqui, ao lado das tres
// formas so de ancora, provando as duas metades no mesmo teste: o guarda nao
// desligou a reescrita de citante nenhum (ref.md) E a auto-referencia com alvo
// escrito continua sendo reescrita depois do move (a propria a.md, ja em
// sub/b.md).
func TestMoveNote_LinkSoDeAncoraNaoQuebraOMove(t *testing.T) {
	origem := "# Topo\n\nVeja [x](#Topo) e [[#Topo]] e ![[#Topo]] e [[a]].\n"
	quer := "# Topo\n\nVeja [x](#Topo) e [[#Topo]] e ![[#Topo]] e [[b]].\n"
	svc, _, _, root := createMoveService(t, map[string]string{
		"a.md":   origem,
		"ref.md": "Outra nota aponta para [[a]]\n",
	})

	res, err := svc.MoveNote(context.Background(), service.MoveNoteRequest{
		From:          "a.md",
		To:            "sub/b.md",
		UpdateLinks:   true,
		CreateFolders: true,
	})
	if err != nil {
		t.Fatalf("MoveNote: %v", err)
	}
	if res.To != "sub/b.md" {
		t.Errorf("res.To = %q, quer %q", res.To, "sub/b.md")
	}

	if _, err := os.Stat(filepath.Join(root, "a.md")); !os.IsNotExist(err) {
		t.Errorf("a nota continua no caminho antigo (err=%v)", err)
	}

	lido, err := os.ReadFile(filepath.Join(root, "sub", "b.md"))
	if err != nil {
		t.Fatalf("a nota nao chegou ao caminho novo: %v", err)
	}

	// As tres formas de ancora saem intactas — mover a nota nao muda para onde
	// elas apontam, e escrever um alvo ali inventaria "[[b#Topo]]" onde o autor
	// escreveu "[[#Topo]]" — e "[[a]]" sai reescrito para "[[b]]": e
	// auto-referencia com ALVO escrito, e essa precisa acompanhar o move. A
	// leitura veio do caminho NOVO (sub/b.md), que e o ponto que ENOENT antes.
	if string(lido) != quer {
		t.Errorf("o corpo apos o move:\n got %q\nwant %q", lido, quer)
	}

	// E o citante de verdade FOI reescrito — o guarda nao pode ter desligado a
	// reescrita de quem realmente aponta para a nota movida.
	refRaw, err := os.ReadFile(filepath.Join(root, "ref.md"))
	if err != nil {
		t.Fatalf("lendo ref.md: %v", err)
	}
	if querRef := "Outra nota aponta para [[b]]\n"; string(refRaw) != querRef {
		t.Errorf("ref.md = %q, quer %q", refRaw, querRef)
	}
	if res.LinksUpdated != 2 {
		t.Errorf("LinksUpdated = %d, quer 2 (ref.md e a propria a.md; nunca os tres de ancora)", res.LinksUpdated)
	}
}

// TestMoveNote_NotaQueCitaASiMesma cobre o Achado F1 da revisao 182 na forma
// que a revisao chamou de comum: "[[a]]" dentro de a.md, alvo ESCRITO (nao so
// ancora). A nota entra em affectedNotes como sua propria citante; moverCorpo
// renomeia o arquivo primeiro; o laco de reescrita lia o caminho ANTIGO —
// ENOENT, com o corpo ja movido e o link nunca reescrito.
//
// O move aqui preserva o nome-base (a.md -> sub/a.md), entao o texto de
// "[[a]]" NAO muda na reescrita — writer.RewriteLinks escreve o mesmo
// nome-base que ja estava la. Quem discrimina esta correcao e err == nil
// (sem o fix, MoveNote falha em ENOENT logo abaixo, antes de qualquer outra
// assercao) mais LinksUpdated == 1: um fix errado que pulasse o citante
// quando ele e a propria nota movida tambem deixaria err == nil, mas com
// LinksUpdated == 0. A assercao do Resolved, mais abaixo, NAO discrimina
// esta correcao — "[[a]]" resolve por nome-base com ou sem reescrita, entao
// o Resolved seria sub/a.md nos dois mundos — e fica como guarda contra a
// regressao do F2 da revisao 182 (Resolved preso ao caminho ANTIGO depois
// de index.MoveNote).
func TestMoveNote_NotaQueCitaASiMesma(t *testing.T) {
	origem := "# A\n\nVeja [[a]].\n"
	svc, v, idx, root := createMoveService(t, map[string]string{
		"a.md": origem,
	})

	res, err := svc.MoveNote(context.Background(), service.MoveNoteRequest{
		From:          "a.md",
		To:            "sub/a.md",
		UpdateLinks:   true,
		CreateFolders: true,
	})
	if err != nil {
		t.Fatalf("MoveNote: %v", err)
	}
	if res.To != "sub/a.md" {
		t.Errorf("res.To = %q, quer %q", res.To, "sub/a.md")
	}

	if _, err := os.Stat(filepath.Join(root, "a.md")); !os.IsNotExist(err) {
		t.Errorf("a nota continua no caminho antigo (err=%v)", err)
	}

	novoAbs := filepath.Join(root, "sub", "a.md")
	lido, err := os.ReadFile(novoAbs)
	if err != nil {
		t.Fatalf("a nota nao chegou ao caminho novo: %v", err)
	}
	if string(lido) != origem {
		t.Errorf("o corpo mudou no move (nome-base identico, texto nao deveria mudar):\n got %q\nwant %q", lido, origem)
	}

	// O Resolved abaixo NAO discrimina esta correcao — "[[a]]" resolve por
	// nome-base com ou sem reescrita, entao seria sub/a.md nos dois mundos.
	// Ele fica como guarda contra a regressao do F2 (revisao 182): Resolved
	// preso ao caminho ANTIGO depois de index.MoveNote. idx.MoveNote e o que
	// o watcher chamaria em producao ao ver o rename; sem ele o indice desta
	// suite fica parado no estado pre-move (nenhuma tool de escrita atualiza o
	// indice direto — ver o comentario de TestMoveNote_HappyPathActuallyMovesTheFile).
	canonicalFrom := vault.CanonicalPath("a.md")
	canonicalTo := vault.CanonicalPath("sub/a.md")
	idx.MoveNote(v, canonicalFrom, canonicalTo)

	nota, ok := idx.Get(canonicalTo)
	if !ok {
		t.Fatalf("idx.Get(%q): nota nao encontrada apos index.MoveNote", canonicalTo)
	}
	var achou bool
	for _, l := range nota.Links {
		if l.Target != "a" {
			continue
		}
		achou = true
		if l.Resolved != canonicalTo {
			t.Errorf("link [[a]] com Resolved = %q, quer %q", l.Resolved, canonicalTo)
		}
	}
	if !achou {
		t.Fatalf("nenhum link com Target=%q em %q", "a", canonicalTo)
	}

	if res.LinksUpdated != 1 {
		t.Errorf("LinksUpdated = %d, quer 1 (o proprio [[a]])", res.LinksUpdated)
	}
}

// note_delete com report_broken_links listava a propria nota apagada como
// "nota cujos links quebram", e emitia um BrokenAnchor de a.md para a.md. A
// nota deixou de existir; nao ha link dela que sobreviva para quebrar.
func TestDeleteNote_NaoAcusaAPropriaNotaApagada(t *testing.T) {
	svc, _, _, _ := createDeleteService(t, map[string]string{
		"a.md":   "# Topo\n\n[[#Topo]] e [[#Nada]]\n",
		"ref.md": "Link para [[a]]\n",
	})

	res, err := svc.DeleteNote(context.Background(), service.DeleteNoteRequest{
		Path:              "a.md",
		ToTrash:           true,
		ReportBrokenLinks: true,
	})
	if err != nil {
		t.Fatalf("DeleteNote: %v", err)
	}

	for _, p := range res.BrokenLinks {
		if p == "a.md" {
			t.Errorf("BrokenLinks = %v; a propria nota apagada nao pode estar na lista", res.BrokenLinks)
		}
	}
	if len(res.BrokenLinks) != 1 || res.BrokenLinks[0] != "ref.md" {
		t.Errorf("BrokenLinks = %v; quer [ref.md]", res.BrokenLinks)
	}

	for _, ba := range res.BrokenAnchors {
		if ba.From == "a.md" {
			t.Errorf("BrokenAnchors tem %+v; a nota apagada nao aponta mais para nada", ba)
		}
	}
}
