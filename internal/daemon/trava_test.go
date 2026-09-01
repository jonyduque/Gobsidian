package daemon

import (
	"bufio"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// varAjudante liga o processo auxiliar. Sem ela, TestAjudanteSeguraTrava e um
// teste que pula; com ela, e o filho que segura a trava.
const varAjudante = "GOBSIDIAN_TESTE_CAMINHO_DA_TRAVA"

// TestAjudanteSeguraTrava NAO e um teste: e o corpo do processo auxiliar que o
// teste de verdade lanca.
//
// Ele existe porque a garantia do flock so pode ser exercitada entre PROCESSOS.
// A versao anterior destes testes simulava a disputa escrevendo um arquivo com
// um PID dentro, o que fazia sentido quando a posse era decidida por PID; com a
// trava do kernel, um arquivo com um numero dentro nao esta travado por
// ninguem, e a simulacao deixaria de medir o que promete.
func TestAjudanteSeguraTrava(t *testing.T) {
	path := os.Getenv(varAjudante)
	if path == "" {
		t.Skip("processo auxiliar; roda so quando o teste pai o invoca")
	}

	trava, tomou, err := tentarTravar(path)
	if err != nil || !tomou {
		_, _ = os.Stdout.WriteString("FALHOU\n")
		os.Exit(1)
	}
	_, _ = os.Stdout.WriteString("TRAVADO\n")

	// Segura ate o pai fechar o stdin ou matar o processo. Nao ha relogio aqui:
	// quem decide quando soltar e o pai.
	_, _ = io.Copy(io.Discard, os.Stdin)
	trava.Liberar()
}

// lancarAjudante sobe o processo auxiliar e so volta quando ele confirma que
// esta com a trava em maos.
func lancarAjudante(t *testing.T, path string) *exec.Cmd {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^TestAjudanteSeguraTrava$", "-test.v")
	cmd.Env = append(os.Environ(), varAjudante+"="+path)

	saida, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("StdoutPipe: %v", err)
	}
	// Stdin precisa ficar ABERTO para o filho nao soltar a trava sozinho.
	entrada, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("StdinPipe: %v", err)
	}
	if err := cmd.Start(); err != nil {
		t.Skipf("nao foi possivel lancar o processo auxiliar: %v", err)
	}
	t.Cleanup(func() {
		_ = entrada.Close()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})

	pronto := make(chan bool, 1)
	go func() {
		leitor := bufio.NewScanner(saida)
		for leitor.Scan() {
			if strings.Contains(leitor.Text(), "TRAVADO") {
				pronto <- true
				return
			}
		}
		pronto <- false
	}()

	select {
	case ok := <-pronto:
		if !ok {
			t.Fatal("o auxiliar nao conseguiu a trava")
		}
	case <-time.After(30 * time.Second):
		t.Fatal("o auxiliar nao confirmou a trava a tempo")
	}
	return cmd
}

// TestTravaDeOutroProcessoNaoERoubada e a metade que impede a correcao de virar
// "sempre toma o lock": com um processo VIVO segurando, ninguem mais entra.
func TestTravaDeOutroProcessoNaoERoubada(t *testing.T) {
	cofre := t.TempDir()
	path, err := lockPath(cofre)
	if err != nil {
		t.Fatalf("lockPath: %v", err)
	}
	lancarAjudante(t, path)

	adquiriu, liberar, err := adquirirLock(cofre)
	if err != nil {
		t.Fatalf("adquirirLock: %v", err)
	}
	if adquiriu {
		if liberar != nil {
			liberar()
		}
		t.Fatal("trava de processo VIVO foi tomada; duas pontes lancariam dois daemons")
	}
}

// TestTravaDeProcessoMortoFicaLivre e a outra metade, e a regressao que deixou
// o daemon desligado por tres dias numa maquina real.
//
// Com a trava do kernel isto nao exige recuperacao nenhuma: o dono morre e a
// trava cai junto. O teste existe para provar que e assim de verdade — e para
// reprovar no dia em que alguem reintroduzir posse por arquivo.
func TestTravaDeProcessoMortoFicaLivre(t *testing.T) {
	cofre := t.TempDir()
	path, err := lockPath(cofre)
	if err != nil {
		t.Fatalf("lockPath: %v", err)
	}
	cmd := lancarAjudante(t, path)

	// Controle: enquanto ele vive, esta ocupada.
	if adquiriu, liberar, _ := adquirirLock(cofre); adquiriu {
		if liberar != nil {
			liberar()
		}
		t.Fatal("a trava do auxiliar vivo nao segurou; o teste nao mede nada")
	}

	if err := cmd.Process.Kill(); err != nil {
		t.Fatalf("matando o auxiliar: %v", err)
	}
	_, _ = cmd.Process.Wait()

	// O arquivo continua LA — nunca e removido — e ainda assim a trava e livre.
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("o arquivo de lock sumiu: %v", err)
	}
	adquiriu, liberar, err := adquirirLock(cofre)
	if err != nil {
		t.Fatalf("adquirirLock depois da morte do dono: %v", err)
	}
	if !adquiriu {
		t.Fatal("o dono morreu e a trava continuou presa; toda ponte seguinte " +
			"concluiria 'outra ja esta subindo o daemon' e serviria em processo")
	}
	liberar()
}
