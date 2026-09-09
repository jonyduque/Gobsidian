//go:build !windows

package daemon

import (
	"os"
	"strconv"
	"strings"
	"syscall"
	"testing"
)

// TestTravaDoKernelEntreProcessos e o irmao de trava_kernel_windows_test.go
// para flock: sem faixa de bytes, e portanto sem a metade do offset.
//
// Prende as duas que valem nos dois sistemas:
//
//  1. TravaEmUso responde true enquanto outro processo detem a trava, e false
//     depois que ele morre — a pergunta que `doctor` faz.
//  2. Um flock exclusivo de FORA falha enquanto o dono vive. Trocar o flock
//     por um sync.Mutex passaria em (1) dentro de um processo so; aqui nao,
//     porque o dono e um processo separado.
//
// O byte 0 tambem e lido aqui, mas por outro motivo: no Unix o flock nao
// bloqueia leitura nenhuma, entao a assercao nao prende o desenho da trava —
// prende que TentarTravar de fato grava o PID depois de tomar a trava, que e
// o que `doctor` mostra.
func TestTravaDoKernelEntreProcessos(t *testing.T) {
	cofre := t.TempDir()
	path, err := lockPath(cofre)
	if err != nil {
		t.Fatalf("lockPath: %v", err)
	}

	// lancarAjudante (trava_test.go) so volta depois que o filho confirma que
	// esta com a trava em maos, e o t.Cleanup dele mata o PID que ELE lancou.
	cmd := lancarAjudante(t, path)

	emUso, err := TravaEmUso(path)
	if err != nil {
		t.Fatalf("TravaEmUso com o dono vivo: %v", err)
	}
	if !emUso {
		t.Fatal("TravaEmUso disse livre com outro processo segurando a trava; " +
			"doctor reportaria 'nenhuma trava em uso' com o daemon rodando")
	}

	f, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		t.Fatalf("abrindo o arquivo de trava de fora: %v", err)
	}
	t.Cleanup(func() { _ = f.Close() })

	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err == nil {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		t.Fatal("consegui o flock exclusivo com o dono vivo: a exclusao nao esta no kernel")
	}

	dados, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile no arquivo de trava com o dono vivo: %v — "+
			"e exatamente a leitura que internal/doctor faz para mostrar o PID", err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(dados)))
	if err != nil {
		t.Fatalf("conteudo do arquivo de trava = %q, esperado o PID do dono: %v", dados, err)
	}
	if pid != cmd.Process.Pid {
		t.Errorf("PID no arquivo de trava = %d, quer %d (o auxiliar que segura)", pid, cmd.Process.Pid)
	}

	if err := cmd.Process.Kill(); err != nil {
		t.Fatalf("matando o auxiliar: %v", err)
	}
	_, _ = cmd.Process.Wait()

	emUso, err = TravaEmUso(path)
	if err != nil {
		t.Fatalf("TravaEmUso depois da morte do dono: %v", err)
	}
	if emUso {
		t.Fatal("TravaEmUso continuou dizendo 'em uso' depois da morte do dono; " +
			"doctor acusaria um daemon que nao existe, e toda ponte seguinte serviria em processo")
	}
}
