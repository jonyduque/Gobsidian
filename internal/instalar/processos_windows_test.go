//go:build windows

package instalar

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const varAjudanteDorme = "GOBSIDIAN_TESTE_AJUDANTE_DORME"

// TestAjudanteDormeComoGobsidian e o processo filho do teste abaixo: uma copia
// deste binario de teste, renomeada para gobsidian.exe, que so dorme.
func TestAjudanteDormeComoGobsidian(t *testing.T) {
	if os.Getenv(varAjudanteDorme) == "" {
		t.Skip("so roda como processo filho de TestProcessosDoSistemaAchaGobsidianSemPresenca")
	}
	time.Sleep(30 * time.Second)
}

// TestProcessosDoSistemaAchaGobsidianSemPresenca sobe um gobsidian.exe que nao
// registra presenca -- a forma dos processos v1.5.1 medidos em 2026-09-14 -- e
// confere que a listagem o encontra, com o caminho do executavel.
func TestProcessosDoSistemaAchaGobsidianSemPresenca(t *testing.T) {
	origem, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}
	copia := filepath.Join(t.TempDir(), NomeDoExecutavel)
	if err := copiarArquivoDeTeste(origem, copia); err != nil {
		t.Fatalf("copiando o binario de teste: %v", err)
	}

	cmd := exec.Command(copia, "-test.run=^TestAjudanteDormeComoGobsidian$")
	cmd.Env = append(os.Environ(), varAjudanteDorme+"=1")
	if err := cmd.Start(); err != nil {
		t.Fatalf("iniciando o ajudante: %v", err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	})
	pid := cmd.Process.Pid

	var achado *ProcessoDoSistema
	var processos []ProcessoDoSistema
	for prazo := time.Now().Add(10 * time.Second); time.Now().Before(prazo) && achado == nil; time.Sleep(100 * time.Millisecond) {
		processos, err = ProcessosDoSistema()
		if err != nil {
			t.Fatalf("ProcessosDoSistema() error = %v", err)
		}
		for i := range processos {
			if processos[i].PID == pid {
				achado = &processos[i]
			}
		}
	}
	if achado == nil {
		t.Fatalf("o gobsidian.exe de teste (pid %d) nao apareceu na listagem: %+v", pid, processos)
	}
	if !strings.EqualFold(achado.Executavel, copia) {
		t.Fatalf("executavel = %q, esperado %q", achado.Executavel, copia)
	}

	// O inverso: com presenca registrada para o mesmo pid, ele sai da conta.
	for _, p := range SemPresenca(processos, []Presenca{{PID: pid}}, os.Getpid()) {
		if p.PID == pid {
			t.Fatalf("SemPresenca manteve o pid %d, que tem presenca", pid)
		}
	}
}

func copiarArquivoDeTeste(de, para string) error {
	origem, err := os.Open(de)
	if err != nil {
		return err
	}
	defer func() { _ = origem.Close() }()
	destino, err := os.Create(para)
	if err != nil {
		return err
	}
	if _, err := io.Copy(destino, origem); err != nil {
		_ = destino.Close()
		return err
	}
	return destino.Close()
}
