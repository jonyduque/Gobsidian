package service_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jonyduque/Gobsidian/internal/index"
	"github.com/jonyduque/Gobsidian/internal/search"
	"github.com/jonyduque/Gobsidian/internal/service"
	"github.com/jonyduque/Gobsidian/internal/vault"
)

func createDeleteService(t *testing.T, files map[string]string) (*service.Service, *vault.Vault, *index.Index, string) {
	t.Helper()
	root := t.TempDir()

	for relPath, content := range files {
		full := filepath.Join(root, filepath.FromSlash(relPath))
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		if err := os.WriteFile(full, []byte(content), 0644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
	}

	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}

	idx := index.New()
	if err := idx.Build(context.Background(), v); err != nil {
		t.Fatalf("idx.Build: %v", err)
	}

	inv := search.NewInverted()
	svc := service.New(v, idx, inv, nil, service.Options{})
	return svc, v, idx, root
}

func TestDeleteNote_ReportBrokenLinksBeforeDeletion(t *testing.T) {
	files := map[string]string{
		"alvo.md": "# Alvo",
		"ref.md":  "Link para [[alvo]]",
	}

	svc, _, _, root := createDeleteService(t, files)

	res, err := svc.DeleteNote(context.Background(), service.DeleteNoteRequest{
		Path:              "alvo.md",
		ToTrash:           true,
		ReportBrokenLinks: true,
	})
	if err != nil {
		t.Fatalf("DeleteNote: %v", err)
	}

	if len(res.BrokenLinks) != 1 || res.BrokenLinks[0] != "ref.md" {
		t.Errorf("BrokenLinks = %v; quer [ref.md]", res.BrokenLinks)
	}

	if _, err := os.Stat(filepath.Join(root, "alvo.md")); err == nil {
		t.Error("alvo.md ainda existe no caminho original apos DeleteNote")
	}

	if _, err := os.Stat(filepath.Join(root, ".trash", "alvo.md")); err != nil {
		t.Errorf("alvo.md nao foi encontrado em .trash/: %v", err)
	}
}

func TestDeleteNote_ToTrashFalseDefiniteDelete(t *testing.T) {
	files := map[string]string{
		"alvo.md": "Conteudo",
	}

	svc, _, _, root := createDeleteService(t, files)

	res, err := svc.DeleteNote(context.Background(), service.DeleteNoteRequest{
		Path:    "alvo.md",
		ToTrash: false,
	})
	if err != nil {
		t.Fatalf("DeleteNote to_trash=false: %v", err)
	}

	if !res.Deleted || res.MovedToTrash {
		t.Errorf("res = %+v; quer Deleted=true, MovedToTrash=false", res)
	}

	if _, err := os.Stat(filepath.Join(root, "alvo.md")); err == nil {
		t.Error("alvo.md ainda existe apos exclusao definitiva")
	}

	if _, err := os.Stat(filepath.Join(root, ".trash", "alvo.md")); err == nil {
		t.Error("alvo.md foi encontrado em .trash/ mas a exclusao era definitiva")
	}
}

func TestDeleteNote_TrashNameCollision(t *testing.T) {
	files := map[string]string{
		"a.md": "Versao 1",
	}

	svc, v, idx, root := createDeleteService(t, files)

	// Primeira exclusao
	res1, err := svc.DeleteNote(context.Background(), service.DeleteNoteRequest{
		Path:    "a.md",
		ToTrash: true,
	})
	if err != nil {
		t.Fatalf("Primeira exclusao: %v", err)
	}

	// Recria a.md com outro conteudo
	if err := os.WriteFile(filepath.Join(root, "a.md"), []byte("Versao 2"), 0644); err != nil {
		t.Fatalf("WriteFile a.md: %v", err)
	}
	_ = idx.Build(context.Background(), v)

	// Havia um time.Sleep(10ms) aqui, e ele nao esperava por nada. A hipotese
	// implicita era que os dois destinos na lixeira se distinguem pelo sufixo
	// `_<UnixNano>` e que duas chamadas rapidas demais receberiam o mesmo
	// numero. Nao e como funciona: `.trash/a.md` nao existe na PRIMEIRA
	// exclusao, entao res1 fica sem sufixo nenhum; so a segunda encontra o nome
	// ocupado e ganha o sufixo. Os dois caminhos diferem por CONSTRUCAO, e nao
	// por resolucao de relogio. O sleep so somava 10 ms a suite.

	// Segunda exclusao com mesmo nome
	res2, err := svc.DeleteNote(context.Background(), service.DeleteNoteRequest{
		Path:    "a.md",
		ToTrash: true,
	})
	if err != nil {
		t.Fatalf("Segunda exclusao: %v", err)
	}

	if res1.TrashPath == res2.TrashPath {
		t.Errorf("colisao de nomes na lixeira: res1=%q, res2=%q", res1.TrashPath, res2.TrashPath)
	}

	// Conteudo, e nao so existencia: dois arquivos na lixeira com o mesmo corpo
	// significariam que a segunda exclusao sobrescreveu a primeira e o caminho
	// devolvido mentiu. Ler os dois e o que distingue "nao colidiu" de "colidiu
	// e ninguem viu".
	for _, caso := range []struct{ caminho, quer string }{
		{res1.TrashPath, "Versao 1"},
		{res2.TrashPath, "Versao 2"},
	} {
		b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(caso.caminho)))
		if err != nil {
			t.Errorf("arquivo da lixeira %q sumiu: %v", caso.caminho, err)
			continue
		}
		if string(b) != caso.quer {
			t.Errorf("lixeira %q tem %q, quer %q: uma exclusao sobrescreveu a outra",
				caso.caminho, string(b), caso.quer)
		}
	}
}

// TestDeleteNoteToTrashMoveSemCopiar: rename preserva o mtime do arquivo;
// WriteAtomic (escreve, sync, rename do temporario) produz um arquivo com mtime
// NOVO. Fixar um mtime antigo e conferir que ele sobreviveu distingue mover de
// copiar sem olhar a implementacao.
func TestDeleteNoteToTrashMoveSemCopiar(t *testing.T) {
	svc, _, _, root := createDeleteService(t, map[string]string{"a.md": "# A\n\ncorpo\n"})

	antigo := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)
	if err := os.Chtimes(filepath.Join(root, "a.md"), antigo, antigo); err != nil {
		t.Fatal(err)
	}

	res, err := svc.DeleteNote(context.Background(), service.DeleteNoteRequest{Path: "a.md", ToTrash: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.TrashPath == "" {
		t.Fatalf("sem TrashPath: %+v", res)
	}
	fi, err := os.Stat(filepath.Join(root, filepath.FromSlash(res.TrashPath)))
	if err != nil {
		t.Fatalf("a nota nao esta na lixeira: %v", err)
	}
	if !fi.ModTime().Equal(antigo) {
		t.Fatalf("mtime da copia na lixeira = %v, quer %v: a lixeira COPIOU em vez de mover", fi.ModTime(), antigo)
	}
	if _, err := os.Stat(filepath.Join(root, "a.md")); !os.IsNotExist(err) {
		t.Fatalf("a origem ainda existe (err=%v)", err)
	}
}

func TestDeleteNote_DryRunDoesNotDelete(t *testing.T) {
	files := map[string]string{
		"alvo.md": "Conteudo",
		"ref.md":  "Link [[alvo]]",
	}

	svc, _, _, root := createDeleteService(t, files)

	alvoPath := filepath.Join(root, "alvo.md")

	// mtime fixado no PASSADO antes da chamada, em vez de lido antes e comparado
	// depois.
	//
	// A versao anterior tirava um os.Stat, chamava DeleteNote e comparava com um
	// segundo os.Stat, sem nada entre os dois capaz de mover o relogio. Se o
	// dry_run reescrevesse o arquivo dentro do mesmo tique de mtime, os dois
	// valores sairiam iguais e a assercao passaria: um teste que so pode falhar
	// quando a maquina esta lenta o bastante. Com 2020 no mtime, QUALQUER
	// reescrita — que carimba a hora atual — reprova, e a resolucao do sistema
	// de arquivos deixa de fazer parte da conta.
	antigo := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)
	if err := os.Chtimes(alvoPath, antigo, antigo); err != nil {
		t.Fatal(err)
	}

	res, err := svc.DeleteNote(context.Background(), service.DeleteNoteRequest{
		Path:              "alvo.md",
		ToTrash:           true,
		ReportBrokenLinks: true,
		DryRun:            true,
	})
	if err != nil {
		t.Fatalf("DeleteNote dry_run: %v", err)
	}

	if !res.DryRun || res.Deleted {
		t.Errorf("res = %+v; quer DryRun=true, Deleted=false", res)
	}

	if len(res.BrokenLinks) != 1 || res.BrokenLinks[0] != "ref.md" {
		t.Errorf("BrokenLinks em dry_run = %v; quer [ref.md]", res.BrokenLinks)
	}

	infoAfter, err := os.Stat(alvoPath)
	if err != nil {
		t.Fatalf("alvo.md sumiu apos dry_run: %v", err)
	}

	if !infoAfter.ModTime().Equal(antigo) {
		t.Errorf("mtime de alvo.md = %v, quer %v: o dry_run tocou o arquivo",
			infoAfter.ModTime(), antigo)
	}
}
