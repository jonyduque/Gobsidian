//go:build windows

package daemon

import (
	"os"
	"strconv"
	"strings"
	"testing"

	"golang.org/x/sys/windows"
)

// deslocamentoEsperadoDaTrava repete, como LITERAL, o offset que
// faixaDaTrava calcula em trava_windows.go.
//
// A repeticao e o ponto do teste, nao um descuido de "uma conta por regra":
// se ele chamasse faixaDaTrava(), produto e teste andariam juntos e trocar o
// offset nao seria detectado por ninguem. Aqui o teste e o CONTRATO, e o
// contrato tem de ser escrito uma vez de fora.
//
// O valor importa porque a trava do Windows e por faixa de bytes e a faixa
// travada recusa leitura alheia com ERROR_LOCK_VIOLATION. `internal/doctor`
// (daemon.go:217) faz `os.ReadFile` no arquivo de trava para mostrar o PID de
// quem a detem; com a faixa no byte 0, essa leitura passaria a falhar em toda
// maquina Windows com um daemon vivo. 1<<62 e uma regiao que nenhum arquivo
// real alcanca.
const deslocamentoEsperadoDaTrava = uint64(1) << 62

// TestTravaDoKernelEntreProcessos prende tres coisas que so um SEGUNDO
// PROCESSO consegue observar, e que por isso nenhum teste in-process cobria:
//
//  1. TravaEmUso responde true enquanto outro processo detem a trava, e false
//     depois que ele morre — a pergunta que `doctor` faz.
//  2. A faixa travada e exatamente 1<<62: um LockFileEx de fora nesse offset
//     TEM de falhar enquanto o dono vive. Trocar o offset no produto faz este
//     LockFileEx passar, e este teste reprova.
//  3. O byte 0 continua LEGIVEL de fora — e o PID que `doctor` mostra. E a
//     metade que explica por que o offset e longe: com a trava no byte 0 a
//     leitura viria com "another process has locked a portion of the file".
//
// Trocar LockFileEx por sync.Mutex passaria em (1) dentro de um processo so;
// aqui nao passa, porque o dono da trava e um processo separado.
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

	sobreposto := windows.Overlapped{
		Offset:     uint32(deslocamentoEsperadoDaTrava & 0xFFFFFFFF),
		OffsetHigh: uint32(deslocamentoEsperadoDaTrava >> 32),
	}
	err = windows.LockFileEx(
		windows.Handle(f.Fd()),
		windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY,
		0, 1, 0, &sobreposto,
	)
	if err == nil {
		_ = windows.UnlockFileEx(windows.Handle(f.Fd()), 0, 1, 0, &sobreposto)
		t.Fatalf("consegui travar o offset %#x com o dono vivo: o produto nao esta travando essa faixa. "+
			"Se o offset mudou, confira que a nova faixa nao alcanca o byte 0 — "+
			"internal/doctor le o PID desse arquivo e uma faixa que cubra o byte 0 quebra essa leitura",
			deslocamentoEsperadoDaTrava)
	}

	// A outra metade do offset: o conteudo segue legivel de fora. Sem esta
	// assercao, mover a trava para o byte 0 continuaria passando em tudo
	// acima — e quebraria o doctor em producao.
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
