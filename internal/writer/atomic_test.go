package writer_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/jonyd/gobsidian/internal/writer"
)

func TestMain(m *testing.M) {
	if os.Getenv("GO_TEST_HELPER_PROCESS") == "1" {
		targetPath := os.Getenv("GO_TEST_TARGET_PATH")
		data := []byte(os.Getenv("GO_TEST_DATA"))
		if targetPath != "" {
			_ = writer.WriteAtomic(targetPath, data)
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func rodarEscritorEMatar(t *testing.T, alvo string, novo []byte, matarEm time.Duration) {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^$")
	cmd.Env = append(os.Environ(),
		"GO_TEST_HELPER_PROCESS=1",
		"GO_TEST_TARGET_PATH="+alvo,
		"GO_TEST_DATA="+string(novo),
	)

	if err := cmd.Start(); err != nil {
		t.Fatalf("cmd.Start: %v", err)
	}

	if matarEm > 0 {
		time.Sleep(matarEm)
		_ = cmd.Process.Kill()
	}
	_ = cmd.Wait()
}

// TestRNF11NoCorruptionUnder1000Crashes e o criterio de bloqueio do M4 (RNF-11).
// Executa 1.000 iteracoes com crash induzido por mata de processo em pontos pseudoaleatorios.
func TestRNF11NoCorruptionUnder1000Crashes(t *testing.T) {
	if testing.Short() {
		t.Skip("1000 iteracoes; roda no gate, nao no ciclo curto")
	}

	const iteracoes = 1000
	antigo := []byte("# Antes\r\n\r\nconteudo original preservado\r\n")
	novo := []byte("# Depois\r\n\r\nconteudo novo completo\r\n")

	corrompidas := 0
	for i := 0; i < iteracoes; i++ {
		dir := t.TempDir()
		alvo := filepath.Join(dir, "nota.md")
		if err := os.WriteFile(alvo, antigo, 0644); err != nil {
			t.Fatal(err)
		}

		matarEm := time.Duration(i%40) * time.Millisecond
		rodarEscritorEMatar(t, alvo, novo, matarEm)

		lido, err := os.ReadFile(alvo)
		if err != nil {
			t.Fatalf("iteracao %d: alvo desapareceu: %v", i, err)
		}

		if !bytes.Equal(lido, antigo) && !bytes.Equal(lido, novo) {
			corrompidas++
			t.Errorf("iteracao %d: estado intermediario (%d bytes): %q",
				i, len(lido), string(lido[:min(80, len(lido))]))
		}

		// Limpa qualquer temporario orfao deixado pelo processo morto antes do rename
		writer.CleanStaleTempFiles(dir)

		sobras, _ := filepath.Glob(filepath.Join(dir, writer.TempFilePrefix+"*"))
		if len(sobras) > 0 {
			t.Errorf("iteracao %d: temporario sobrou: %v", i, sobras)
		}
	}

	if corrompidas > 0 {
		t.Fatalf("RNF-11 REPROVADO: %d de %d iteracoes corromperam a nota", corrompidas, iteracoes)
	}

	t.Logf("RNF-11: 0 corrompidas em %d iteracoes", iteracoes)
}

func TestWriteAtomic_TempInSameDir(t *testing.T) {
	dir := t.TempDir()
	alvo := filepath.Join(dir, "nota.md")
	data := []byte("conteudo de teste")

	if err := writer.WriteAtomic(alvo, data); err != nil {
		t.Fatalf("WriteAtomic: %v", err)
	}

	lido, err := os.ReadFile(alvo)
	if err != nil {
		t.Fatalf("os.ReadFile: %v", err)
	}
	if !bytes.Equal(lido, data) {
		t.Errorf("lido = %q, quer %q", lido, data)
	}
}

func TestWriteAtomic_CreatesTempInTargetDir(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "subfolder")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	alvo := filepath.Join(dir, "nota.md")
	data := bytes.Repeat([]byte("1234567890\n"), 200000)

	done := make(chan error, 1)
	go func() {
		done <- writer.WriteAtomic(alvo, data)
	}()

	foundInDir := false
	for start := time.Now(); time.Since(start) < 2*time.Second; {
		matches, _ := filepath.Glob(filepath.Join(dir, writer.TempFilePrefix+"*"))
		if len(matches) > 0 {
			foundInDir = true
			break
		}
		time.Sleep(1 * time.Millisecond)
	}

	err := <-done
	if err != nil {
		t.Fatalf("WriteAtomic: %v", err)
	}

	if !foundInDir {
		t.Fatalf("temporario nao foi criado no mesmo diretorio do alvo (%s), criando risco de rename nao-atomico entre volumes", dir)
	}
}

func TestWriteAtomic_PreservesBOMAndCRLF(t *testing.T) {
	dir := t.TempDir()
	alvo := filepath.Join(dir, "nota_bom.md")
	// BOM + CRLF
	data := []byte("\xef\xbb\xbf# Nota com BOM\r\n\r\nLinha 1\r\nLinha 2\r\n")

	if err := writer.WriteAtomic(alvo, data); err != nil {
		t.Fatalf("WriteAtomic: %v", err)
	}

	lido, err := os.ReadFile(alvo)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !bytes.Equal(lido, data) {
		t.Errorf("BOM/CRLF nao foram preservados byte a byte")
	}
}

func TestWriteAtomic_RenameRetryOnLock(t *testing.T) {
	dir := t.TempDir()
	alvo := filepath.Join(dir, "bloqueado.md")

	if err := os.WriteFile(alvo, []byte("inicial"), 0644); err != nil {
		t.Fatal(err)
	}

	// Abre um handle em modo exclusivo temporario simulando antivirus / indexador
	f, err := os.OpenFile(alvo, os.O_RDWR, 0644)
	if err != nil {
		t.Fatal(err)
	}

	done := make(chan error, 1)
	go func() {
		done <- writer.WriteAtomic(alvo, []byte("substituido"))
	}()

	// Solta o bloqueio apos 30ms para exercitar o retry do rename
	time.Sleep(30 * time.Millisecond)
	_ = f.Close()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("WriteAtomic falhou apos soltar o lock: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout aguardando WriteAtomic")
	}

	lido, _ := os.ReadFile(alvo)
	if string(lido) != "substituido" {
		t.Errorf("conteudo lido = %q, quer 'substituido'", string(lido))
	}
}
