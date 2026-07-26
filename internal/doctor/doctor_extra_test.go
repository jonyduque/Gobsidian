package doctor_test

// Testes adicionais aos dois exigidos pela Task 10, cobrindo comportamentos
// que a brief pede para confirmar mas cujos testes nao escreve: ExitCode com
// slice vazio, propagacao de cancelamento de contexto para as verificacoes
// que varrem o cofre, limpeza do arquivo temporario da verificacao de
// escrita, e a medida correta da verificacao de comprimento de caminho.

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/jonyd/gobsidian/internal/config"
	"github.com/jonyd/gobsidian/internal/doctor"
)

func TestExitCodeEmptyResults(t *testing.T) {
	if code := doctor.ExitCode(nil); code != 0 {
		t.Errorf("ExitCode(nil) = %d, esperava 0", code)
	}
	if code := doctor.ExitCode([]doctor.Result{}); code != 0 {
		t.Errorf("ExitCode([]Result{}) = %d, esperava 0", code)
	}
}

// TestRunContextCancelledStopsWalk confirma que um contexto ja cancelado
// impede as verificacoes que varrem o cofre de completar a varredura. Sem
// isso, um cofre grande sincronizado na nuvem faria doctor travar em vez de
// respeitar o cancelamento do chamador.
func TestRunContextCancelledStopsWalk(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"A.md", "B.md", "C.md", "D.md", "E.md"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("# nota\n"), 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
	}

	cfg := config.Defaults()
	cfg.VaultPath = root

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancelado antes mesmo de Run comecar

	results := doctor.Run(ctx, cfg)

	if doctor.ExitCode(results) != 0 {
		t.Errorf("cancelamento de contexto nao deveria virar falha bloqueante: %+v", results)
	}

	found := false
	for _, r := range results {
		if r.Name != "contagem de notas" {
			continue
		}
		found = true
		if !strings.Contains(r.Detail, "interrompida") {
			t.Errorf("contagem de notas deveria reportar varredura interrompida com contexto cancelado, recebi: %+v", r)
		}
		if strings.Contains(r.Detail, "5 notas") {
			t.Errorf("contagem de notas completou a varredura apesar do contexto cancelado: %+v", r)
		}
	}
	if !found {
		t.Fatal("verificacao de contagem de notas nao encontrada nos resultados")
	}
}

// TestCheckWritableNoLeftoverTempFile confirma que a verificacao de escrita
// nao deixa nenhum arquivo temporario para tras na raiz do cofre.
func TestCheckWritableNoLeftoverTempFile(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "A.md"), []byte("# A\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cfg := config.Defaults()
	cfg.VaultPath = root

	_ = doctor.Run(context.Background(), cfg)

	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, e := range entries {
		if strings.Contains(strings.ToLower(e.Name()), "gobsidian-doctor") {
			t.Errorf("arquivo temporario da verificacao de escrita nao foi limpo: %s", e.Name())
		}
	}
}

// TestCheckLongestPathMeasuresAbsolutePath confirma que a verificacao de
// comprimento de caminho mede o caminho absoluto completo (o que um syscall
// receberia), nao apenas a porcao relativa a raiz do cofre — caso contrario
// o limiar de 240 caracteres, que existe por causa do MAX_PATH absoluto do
// Windows, nunca dispararia para um cofre com nomes de arquivo curtos mas
// localizado em um caminho profundo.
func TestCheckLongestPathMeasuresAbsolutePath(t *testing.T) {
	root := t.TempDir()
	const name = "A.md"
	if err := os.WriteFile(filepath.Join(root, name), []byte("# A\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cfg := config.Defaults()
	cfg.VaultPath = root

	results := doctor.Run(context.Background(), cfg)

	wantLen := len(filepath.Join(root, name))
	relLen := len(name)

	for _, r := range results {
		if r.Name != "comprimento de caminho" {
			continue
		}
		wantSubstr := strconv.Itoa(wantLen)
		if !strings.Contains(r.Detail, wantSubstr) {
			t.Errorf("esperava o comprimento do caminho absoluto (%d) no detalhe, recebi %q", wantLen, r.Detail)
		}
		relSubstr := strconv.Itoa(relLen)
		if wantSubstr != relSubstr && strings.Contains(r.Detail, relSubstr) && !strings.Contains(r.Detail, wantSubstr) {
			t.Errorf("detalhe parece medir o caminho relativo (%d), nao o absoluto (%d): %q", relLen, wantLen, r.Detail)
		}
		return
	}
	t.Fatal("verificacao de comprimento de caminho nao encontrada nos resultados")
}
