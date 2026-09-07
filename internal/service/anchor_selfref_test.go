package service_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jonyd/gobsidian/internal/service"
)

// Desde 2026-09-06 um link so de ancora resolve para a PROPRIA nota, e com isso
// a nota virou backlink de si mesma. note_move colhia esse backlink como se
// fosse um citante a reescrever: movia o corpo primeiro e so depois lia o
// caminho ANTIGO para reescrever os links, o que dava ENOENT e devolvia
// CodeInternal com o move ja pela metade — corpo movido, citantes intactos.
//
// O discriminante e Target == "", e NAO "o citante e a propria nota":
// auto-referencia com alvo escrito precisa continuar sendo reescrita. Este
// teste nao consegue fixar essa segunda metade — ver o LIMITE abaixo —, entao
// ela fica com ref.md, que prova ao menos que o guarda nao desligou a reescrita
// de citante nenhum.
//
// LIMITE MEDIDO em 2026-09-06: uma nota que cita a si mesma com alvo ESCRITO
// ("[[a]]" dentro de a.md) derruba note_move pelo mesmo ENOENT, e isso e
// PRE-EXISTENTE — independe da Task 182, porque nada nela toca link com alvo.
// Sondado neste mesmo tree com uma nota cujo unico conteudo era "Veja [[a]].":
// `MoveNote: lendo nota "a.md": ... The system cannot find the file specified`.
// Por isso a nota movida aqui nao contem "[[a]]": o teste falharia por um
// defeito que nao e desta tarefa. Registrado no relatorio.
func TestMoveNote_LinkSoDeAncoraNaoQuebraOMove(t *testing.T) {
	origem := "# Topo\n\nVeja [x](#Topo) e [[#Topo]] e ![[#Topo]].\n"
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

	// As tres formas de ancora saem intactas: mover a nota nao muda para onde
	// elas apontam, e escrever um alvo ali inventaria "[[b#Topo]]" onde o autor
	// escreveu "[[#Topo]]".
	if string(lido) != origem {
		t.Errorf("o corpo mudou no move:\n got %q\nwant %q", lido, origem)
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
	if res.LinksUpdated != 1 {
		t.Errorf("LinksUpdated = %d, quer 1 (so o de ref.md, nunca os tres de ancora)", res.LinksUpdated)
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
