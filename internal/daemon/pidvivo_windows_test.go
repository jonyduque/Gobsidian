//go:build windows

package daemon

import (
	"os/exec"
	"testing"

	"golang.org/x/sys/windows"
)

// TestPidVivoDetectaMorteComHandleAberto e o teste da regra que so o Windows
// tem, e a unica montagem em que ela pode ser observada.
//
// O Windows nao libera o PID enquanto houver handle aberto para o processo
// morto. Se o teste chamar cmd.Wait(), o Go FECHA o handle, o PID e liberado, e
// OpenProcess passa a falhar sozinho — pidVivo devolveria false pelo motivo
// errado, e a checagem de exitTime nunca seria exercitada. Foi exatamente isso
// que aconteceu na primeira versao deste teste: a mutacao que apagava o exitTime
// PASSOU, e a regra ficou escrita sem estar verificada.
//
// Aqui o Wait e adiado de proposito. Com o handle do Go ainda aberto,
// OpenProcess SUCEDE sobre um processo morto, e so exitTime distingue.
func TestPidVivoDetectaMorteComHandleAberto(t *testing.T) {
	cmd := exec.Command("cmd", "/c", "exit")
	if err := cmd.Start(); err != nil {
		t.Skipf("nao foi possivel lancar processo auxiliar: %v", err)
	}
	pid := cmd.Process.Pid
	// Wait so no fim: e ele que solta o handle do Go.
	t.Cleanup(func() { _ = cmd.Wait() })

	// Handle proprio, so para esperar o termino sem soltar o do Go.
	h, err := windows.OpenProcess(windows.SYNCHRONIZE|windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		t.Skipf("nao foi possivel abrir o processo %d: %v", pid, err)
	}
	defer func() { _ = windows.CloseHandle(h) }()

	if _, err := windows.WaitForSingleObject(h, 10_000); err != nil {
		t.Fatalf("esperando o processo %d terminar: %v", pid, err)
	}

	// Guarda da montagem, e ela e o teste inteiro: se OpenProcess ja falhasse
	// aqui, pidVivo devolveria false sem consultar exitTime, e este teste
	// passaria sem afirmar nada sobre a regra.
	h2, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		t.Skipf("o PID %d ja foi liberado; a condicao que este teste precisa "+
			"(processo morto ainda consultavel) nao se montou", pid)
	}
	_ = windows.CloseHandle(h2)

	if pidVivo(pid) {
		t.Fatalf("pidVivo(%d) = true para um processo JA MORTO que ainda responde "+
			"a OpenProcess; e o defeito que deixou 5 de 5 orfaos na vigilia do pai, "+
			"e aqui faria uma ponte respeitar para sempre o lock de um processo morto", pid)
	}
}
