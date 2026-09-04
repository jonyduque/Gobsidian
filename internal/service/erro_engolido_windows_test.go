//go:build windows

package service_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/service"
	"github.com/jonyd/gobsidian/internal/vault"
	"github.com/jonyd/gobsidian/internal/vaulttest"
)

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

	// Guarda da montagem, e so ela: o handle aberto sem compartilhar exclusao
	// faz o os.Remove falhar, entao a origem TEM de continuar no disco. Com
	// isso afirmado aqui, as duas assercoes seguintes ficam incondicionais —
	// enquanto elas eram guardadas por `errOrigem == nil && err == nil && ...`,
	// um cenario que nao se montasse passava calado, que e o defeito que esta
	// tarefa remove.
	if _, errOrigem := os.Stat(origem); errOrigem != nil {
		t.Fatalf("cenario invalido: a origem sumiu do disco apesar do handle aberto: %v", errOrigem)
	}
	if err == nil {
		t.Fatalf("DeleteNote devolveu nil com a nota ainda no caminho original: ela existe na "+
			"lixeira E em %q (res=%+v)", origem, res)
	}
	// Desde 2026-09-02 o erro vem de moverCorpo, a conta unica do move, e
	// nomeia o destino pelo caminho (".trash/origem.md") em vez da palavra
	// "lixeira". A garantia e a mesma — o erro diz onde a copia esta.
	if !strings.Contains(strings.ToLower(err.Error()), ".trash") {
		t.Errorf("o erro nao explica que a copia na lixeira existe: %v", err)
	}
}

// TestDeleteNoteToTrashNaoBaixaPlaceholder: com o rename recusado, o fallback
// de copia de moverCorpo faria os.ReadFile num placeholder de nuvem — o
// download sincrono que a regra "somente-nuvem nunca e aberto" proibe.
func TestDeleteNoteToTrashNaoBaixaPlaceholder(t *testing.T) {
	root := t.TempDir()
	caminho := filepath.Join(root, "nuvem.md")
	if err := os.WriteFile(caminho, []byte("# Nuvem\n\ncorpo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	vaulttest.MarcarSomenteNuvem(t, caminho)

	v, err := vault.New(root)
	if err != nil {
		t.Fatal(err)
	}
	idx := index.New()
	if err := idx.Build(context.Background(), v); err != nil {
		t.Fatal(err)
	}
	if n, ok := idx.Get("nuvem.md"); !ok || !n.CloudOnly {
		t.Fatal("a nota nao ficou CloudOnly; o atributo nao pegou")
	}
	// Handle exclusivo: os.Rename recusa, e o fallback de copia e forcado.
	vaulttest.TravarExclusivo(t, caminho)
	svc := service.New(v, idx, nil, nil, service.Options{})

	_, err = svc.DeleteNote(context.Background(), service.DeleteNoteRequest{Path: "nuvem.md", ToTrash: true})
	if err == nil {
		t.Fatal("DeleteNote to_trash devolveu sucesso sobre um placeholder com rename recusado: o fallback leu o arquivo")
	}
	if got := service.CodeOf(err); got != service.CodeCloudOnlyFile {
		t.Fatalf("codigo = %s, quer %s: %v", got, service.CodeCloudOnlyFile, err)
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
	vaulttest.TravarExclusivo(t, filepath.Join(dir, "origem.md"))

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
