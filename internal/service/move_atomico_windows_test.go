//go:build windows

package service_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/windows"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/service"
	"github.com/jonyd/gobsidian/internal/vault"
)

// montaCofreParaMove cria um cofre com a nota a mover e uma nota que a cita.
func montaCofreParaMove(t *testing.T) (*service.Service, string) {
	t.Helper()
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "origem.md"),
		[]byte("# Origem\n\ncorpo da nota.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "citante.md"),
		[]byte("# Citante\n\nver [[origem]] aqui.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	v, err := vault.New(dir)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	idx := index.New()
	if err := idx.Build(context.Background(), v); err != nil {
		t.Fatalf("idx.Build: %v", err)
	}
	return service.New(v, idx, nil, nil, service.Options{}), dir
}

// TestMoveNaoReportaSucessoComNotaDuplicada cobre o A1 pelo caminho que o
// produz em campo.
//
// `_ = os.Remove(absFrom)` descartava o erro. No Windows, um arquivo com handle
// aberto por outro processo recusa remoção — e o Obsidian mantendo a nota
// aberta é rotina, não caso de borda. O resultado era **sucesso reportado com a
// nota existindo nos DOIS caminhos**.
//
// O teste segura um handle de verdade, em vez de injetar um dublê: é a mesma
// condição do sistema operacional que o usuário encontra.
func TestMoveNaoReportaSucessoComNotaDuplicada(t *testing.T) {
	svc, dir := montaCofreParaMove(t)
	origem := filepath.Join(dir, "origem.md")

	// Handle aberto SEM compartilhar exclusão: e o que faz os.Remove falhar.
	trava, err := os.Open(origem)
	if err != nil {
		t.Fatalf("abrindo a nota para travar: %v", err)
	}
	defer func() { _ = trava.Close() }()

	res, err := svc.MoveNote(context.Background(), service.MoveNoteRequest{
		From:        "origem.md",
		To:          "destino.md",
		UpdateLinks: true,
	})

	_, errOrigem := os.Stat(origem)
	_, errDestino := os.Stat(filepath.Join(dir, "destino.md"))
	origemExiste := errOrigem == nil
	destinoExiste := errDestino == nil

	if origemExiste && destinoExiste && err == nil {
		t.Errorf("SUCESSO reportado com a nota DUPLICADA: origem e destino existem "+
			"e MoveNote devolveu nil (res=%+v)", res)
	}
	if origemExiste && destinoExiste && err != nil {
		t.Logf("estado duplicado, mas o erro foi reportado: %v", err)
	}
	if err == nil && origemExiste {
		t.Error("MoveNote devolveu nil mas a origem continua no disco")
	}
}

// TestMoveNaoReescreveCitantesAntesDeMoverOCorpo cobre a segunda metade do A1.
//
// Os citantes eram reescritos ANTES de a nota se mover. Falhando a movimentação,
// os links já estavam PERSISTIDOS EM DISCO apontando para um destino que não
// existe — sem compensação, e com o `raw` original em mãos no loop.
//
// O cenário precisa falhar DEPOIS do laço de citantes, e por isso trava a
// LEITURA da origem com acesso exclusivo: `os.ReadFile(absFrom)` acontece
// depois das reescritas. Ocupar o destino não serviria — a validação de
// destino existente falha antes, e o teste passaria sem exercitar nada.
//
// O acesso pedido importa, e foi MEDIDO em 2026-08-26 nesta máquina:
//
//	GENERIC_READ        + share=0 -> os.ReadFile OK,   os.Remove ERRO
//	GENERIC_READ|WRITE  + share=0 -> os.ReadFile ERRO, os.Remove ERRO
//
// Só o segundo bloqueia a leitura, que é o que este cenário precisa. O primeiro
// é o que o outro teste deste arquivo usa, porque lá o alvo é o `os.Remove`.
func TestMoveNaoReescreveCitantesAntesDeMoverOCorpo(t *testing.T) {
	svc, dir := montaCofreParaMove(t)

	p, err := windows.UTF16PtrFromString(filepath.Join(dir, "origem.md"))
	if err != nil {
		t.Fatal(err)
	}
	h, err := windows.CreateFile(p, windows.GENERIC_READ|windows.GENERIC_WRITE,
		0 /* sem compartilhamento */, nil,
		windows.OPEN_EXISTING, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		t.Skipf("nao foi possivel travar a origem com acesso exclusivo: %v", err)
	}
	defer func() { _ = windows.CloseHandle(h) }()

	_, err = svc.MoveNote(context.Background(), service.MoveNoteRequest{
		From:        "origem.md",
		To:          "destino.md",
		UpdateLinks: true,
	})
	if err == nil {
		t.Fatal("MoveNote devolveu nil com a origem travada para leitura")
	}

	citante, errLer := os.ReadFile(filepath.Join(dir, "citante.md"))
	if errLer != nil {
		t.Fatalf("lendo citante: %v", errLer)
	}
	if !strings.Contains(string(citante), "[[origem]]") {
		t.Errorf("o citante foi reescrito ANTES de o corpo mover, e a movimentacao "+
			"falhou: o link agora aponta para um destino inexistente.\ncitante = %q",
			string(citante))
	}
}
