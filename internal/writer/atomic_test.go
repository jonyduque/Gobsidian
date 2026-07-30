package writer_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
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
	orfaos := 0
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

		// Um processo morto NAO roda defer, entao deixar temporario aqui e o
		// comportamento esperado — nao defeito. O que se afirma e que a
		// recuperacao os remove.
		//
		// A versao anterior deste bloco chamava CleanStaleTempFiles e DEPOIS
		// afirmava que nao havia sobras: a linha de cima garantia a de baixo, e
		// a assercao nao podia falhar. Medido: com a limpeza neutralizada, este
		// teste reprova — havia temporarios reais sendo mascarados.
		antesDaVarredura, _ := filepath.Glob(filepath.Join(dir, writer.TempFilePrefix+"*"))
		orfaos += len(antesDaVarredura)

		removidos, err := writer.SweepStaleTempFiles(context.Background(), dir)
		if err != nil {
			t.Fatalf("iteracao %d: SweepStaleTempFiles: %v", i, err)
		}
		if removidos != len(antesDaVarredura) {
			t.Errorf("iteracao %d: varredura removeu %d de %d temporarios",
				i, removidos, len(antesDaVarredura))
		}
		if sobras, _ := filepath.Glob(filepath.Join(dir, writer.TempFilePrefix+"*")); len(sobras) > 0 {
			t.Errorf("iteracao %d: temporario sobrou DEPOIS da varredura: %v", i, sobras)
		}
	}

	if corrompidas > 0 {
		t.Fatalf("RNF-11 REPROVADO: %d de %d iteracoes corromperam a nota", corrompidas, iteracoes)
	}

	// O numero de orfaos e informacao, nao falha: ele mede quantas iteracoes
	// morreram DEPOIS de criar o temporario e ANTES do rename, que e exatamente
	// a janela que a escrita atomica existe para cobrir. Zero orfaos em 1.000
	// iteracoes significaria que o crash nunca caiu nessa janela, e ai o teste
	// nao teria exercitado o que promete.
	t.Logf("RNF-11: 0 corrompidas em %d iteracoes; %d temporarios orfaos varridos",
		iteracoes, orfaos)
	if orfaos == 0 {
		t.Error("nenhum temporario orfao em 1.000 crashes: o crash nunca caiu " +
			"entre criar o temporario e renomear, entao a atomicidade nao foi exercitada")
	}
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

// TestWriteAtomicConcurrentSameDirectory e o teste de regressao do defeito que
// a revisao do M4 achou: WriteAtomic chamava CleanStaleTempFiles(dir) no
// inicio, e o glob apaga TODOS os temporarios do diretorio — inclusive o de
// outra escrita em voo. A trava do writer e por CAMINHO de proposito, entao
// duas notas na mesma pasta escrevem em paralelo, e o recurso compartilhado
// (o diretorio) nao esta coberto por ela.
//
// ATENCAO AO LER UM VERDE AQUI NO WINDOWS: os.Remove sobre arquivo com handle
// aberto falha com sharing violation, e o erro era engolido — a corrida ficava
// mascarada. Em Linux e macOS o unlink sucede por semantica POSIX, a outra
// escrita segue gravando num inode desvinculado, e o rename falha com ENOENT.
// Este teste so reprova de verdade nos dois ultimos; no Windows ele documenta a
// intencao e guarda contra alguem devolver a varredura para dentro da escrita.
func TestWriteAtomicConcurrentSameDirectory(t *testing.T) {
	dir := t.TempDir()
	const escritores = 8
	const porEscritor = 25

	var wg sync.WaitGroup
	erros := make(chan error, escritores*porEscritor)

	for w := 0; w < escritores; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			alvo := filepath.Join(dir, fmt.Sprintf("nota%02d.md", w))
			esperado := []byte(fmt.Sprintf("conteudo do escritor %02d\r\n", w))
			for i := 0; i < porEscritor; i++ {
				if err := writer.WriteAtomic(alvo, esperado); err != nil {
					erros <- fmt.Errorf("escritor %d, iteracao %d: %w", w, i, err)
					return
				}
			}
		}(w)
	}
	wg.Wait()
	close(erros)

	for err := range erros {
		t.Errorf("escrita concorrente na mesma pasta falhou: %v", err)
	}

	// Cada alvo tem de ter exatamente o conteudo do seu escritor. Uma escrita
	// que perdeu o temporario para a varredura de outra deixaria o alvo com o
	// conteudo antigo — ou inexistente.
	for w := 0; w < escritores; w++ {
		alvo := filepath.Join(dir, fmt.Sprintf("nota%02d.md", w))
		esperado := fmt.Sprintf("conteudo do escritor %02d\r\n", w)
		lido, err := os.ReadFile(alvo)
		if err != nil {
			t.Errorf("escritor %d: alvo ausente: %v", w, err)
			continue
		}
		if string(lido) != esperado {
			t.Errorf("escritor %d: lido %q, quer %q", w, lido, esperado)
		}
	}

	// E nenhum temporario pode sobrar: aqui nenhum processo morreu, entao o
	// defer de cada WriteAtomic removeu o seu.
	if sobras, _ := filepath.Glob(filepath.Join(dir, writer.TempFilePrefix+"*")); len(sobras) > 0 {
		t.Errorf("temporario sobrou apos escritas bem-sucedidas: %v", sobras)
	}
}
