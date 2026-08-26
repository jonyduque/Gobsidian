//go:build windows

package service_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/windows"

	"github.com/jonyd/gobsidian/internal/service"
)

// travaExclusiva impede leitura E remoção do arquivo, como um aplicativo que o
// mantém aberto.
//
// O acesso pedido importa, e foi medido em 2026-08-26: `GENERIC_READ` com
// `share=0` bloqueia só o remove; `GENERIC_READ|GENERIC_WRITE` bloqueia os
// dois. Aqui queremos os dois.
func travaExclusiva(t *testing.T, abs string) {
	t.Helper()
	p, err := windows.UTF16PtrFromString(abs)
	if err != nil {
		t.Fatal(err)
	}
	h, err := windows.CreateFile(p, windows.GENERIC_READ|windows.GENERIC_WRITE, 0, nil,
		windows.OPEN_EXISTING, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		t.Skipf("nao foi possivel travar %q com acesso exclusivo: %v", abs, err)
	}
	t.Cleanup(func() { _ = windows.CloseHandle(h) })
}

// TestDeleteToTrashNaoMenteQuandoORemoveFalha cobre o B5.
//
// `_ = os.Remove(absPath)` descartava o erro depois de copiar para a lixeira.
// Nota travada pelo Obsidian ⇒ `Deleted: true` com a nota existindo em DOIS
// lugares: o caminho original e a lixeira. É a mesma família do A1, no outro
// caminho de escrita.
func TestDeleteToTrashNaoMenteQuandoORemoveFalha(t *testing.T) {
	svc, dir := montaCofreParaMove(t)
	origem := filepath.Join(dir, "origem.md")

	// Basta bloquear o remove; a leitura precisa funcionar para a copia
	// chegar a lixeira, que e o cenario que interessa.
	f, err := os.Open(origem)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()

	res, err := svc.DeleteNote(context.Background(), service.DeleteNoteRequest{
		Path:    "origem.md",
		ToTrash: true,
	})

	_, errOrigem := os.Stat(origem)
	if errOrigem == nil && err == nil && res.Deleted {
		t.Errorf("Deleted=true com a nota ainda no caminho original: ela existe na "+
			"lixeira E em %q (res=%+v)", origem, res)
	}
	if err != nil && !strings.Contains(strings.ToLower(err.Error()), "lixeira") {
		t.Errorf("o erro nao explica que a copia na lixeira existe: %v", err)
	}
}

// TestMoveDryRunNaoApresentaDiffVazioComoResultado cobre o B17.
//
// `fromRaw, _ := os.ReadFile(absFrom)` engolia o erro de leitura, e o dry-run
// seguia produzindo um diff de `""` contra `""` — vazio, mas apresentado como
// resultado legítimo. Quem lê um dry-run vazio conclui que a operação não muda
// nada, que é o oposto do que aconteceria.
func TestMoveDryRunNaoApresentaDiffVazioComoResultado(t *testing.T) {
	svc, dir := montaCofreParaMove(t)
	travaExclusiva(t, filepath.Join(dir, "origem.md"))

	res, err := svc.MoveNote(context.Background(), service.MoveNoteRequest{
		From:        "origem.md",
		To:          "destino.md",
		UpdateLinks: true,
		DryRun:      true,
	})
	if err == nil {
		t.Errorf("dry-run devolveu sucesso com a origem ilegivel; diffs=%v", res.Diffs)
	}
}
