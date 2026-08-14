package daemon

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

// escreverLock grava um arquivo de lock com o conteudo dado, no caminho que
// adquirirLock vai consultar para este cofre.
func escreverLock(t *testing.T, cofre, conteudo string) string {
	t.Helper()
	path, err := lockPath(cofre)
	if err != nil {
		t.Fatalf("lockPath: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte(conteudo), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(path) })
	return path
}

// pidMorto devolve um PID que com certeza nao esta em uso: lanca um processo
// trivial e espera ele terminar.
//
// Um numero grande escolhido a dedo (99999) seria um teste que depende de o
// sistema nao ter chegado naquele PID — passa hoje e falha numa maquina com
// uptime alto. Aqui a morte e observada, nao presumida.
func pidMorto(t *testing.T) int {
	t.Helper()
	cmd := comandoTrivial()
	if err := cmd.Start(); err != nil {
		t.Skipf("nao foi possivel lancar processo auxiliar: %v", err)
	}
	pid := cmd.Process.Pid
	_ = cmd.Wait()
	if pidVivo(pid) {
		// No Windows o PID continua consultavel depois da morte; se pidVivo
		// devolver true aqui, e porque a checagem de exitTime nao esta
		// funcionando — e o teste inteiro perderia o sentido.
		t.Fatalf("PID %d continua vivo depois de Wait; pidVivo nao detecta morte", pid)
	}
	return pid
}

// TestAdquirirLockRecuperaLockDeProcessoMorto e o teste da regressao que
// deixou o daemon desligado por tres dias numa maquina real.
func TestAdquirirLockRecuperaLockDeProcessoMorto(t *testing.T) {
	cofre := t.TempDir()
	morto := pidMorto(t)
	path := escreverLock(t, cofre, strconv.Itoa(morto)+"\n")

	adquiriu, liberar, err := adquirirLock(cofre)
	if err != nil {
		t.Fatalf("adquirirLock: %v", err)
	}
	if !adquiriu {
		t.Fatalf("lock do PID morto %d nao foi recuperado; toda ponte seguinte "+
			"concluiria 'outra ja esta subindo o daemon' e serviria em processo", morto)
	}
	defer liberar()

	// O lock agora tem de ser NOSSO: recuperar sem reivindicar deixaria a
	// proxima ponte roubando o lock desta.
	dados, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("lendo lock apos recuperar: %v", err)
	}
	pid, err := strconv.Atoi(string(dados[:len(dados)-1]))
	if err != nil {
		t.Fatalf("lock recuperado tem conteudo ilegivel %q", string(dados))
	}
	if pid != os.Getpid() {
		t.Fatalf("lock recuperado guarda PID %d, quer %d (o nosso)", pid, os.Getpid())
	}
}

// TestAdquirirLockRespeitaLockDeProcessoVivo e o par do teste acima, e o que
// impede a correcao de virar "sempre rouba o lock".
func TestAdquirirLockRespeitaLockDeProcessoVivo(t *testing.T) {
	cofre := t.TempDir()
	// O PID deste proprio processo de teste esta, por construcao, vivo.
	escreverLock(t, cofre, strconv.Itoa(os.Getpid())+"\n")

	adquiriu, liberar, err := adquirirLock(cofre)
	if err != nil {
		t.Fatalf("adquirirLock: %v", err)
	}
	if adquiriu {
		if liberar != nil {
			liberar()
		}
		t.Fatal("lock de processo VIVO foi roubado; duas pontes poderiam lancar " +
			"dois daemons para o mesmo cofre")
	}
}

// TestAdquirirLockRecuperaLockVazio cobre o processo morto ENTRE o O_EXCL e a
// escrita do PID — um arquivo de zero byte que nenhum dono legitimo reivindica
// e que, tratado como vivo, travaria o daemon para sempre.
func TestAdquirirLockRecuperaLockVazio(t *testing.T) {
	for _, caso := range []struct {
		nome     string
		conteudo string
	}{
		{"vazio", ""},
		{"so quebra de linha", "\n"},
		{"lixo", "nao-e-um-pid\n"},
		{"pid invalido", "0\n"},
	} {
		t.Run(caso.nome, func(t *testing.T) {
			cofre := t.TempDir()
			escreverLock(t, cofre, caso.conteudo)

			adquiriu, liberar, err := adquirirLock(cofre)
			if err != nil {
				t.Fatalf("adquirirLock: %v", err)
			}
			if !adquiriu {
				t.Fatalf("lock com conteudo %q nao foi recuperado; sem PID legivel "+
					"nao ha dono a respeitar, e respeita-lo trava o daemon para sempre",
					caso.conteudo)
			}
			liberar()
		})
	}
}
