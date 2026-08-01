package writer_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/jonyd/gobsidian/internal/writer"
)

// avisoPronto e o que o processo filho imprime imediatamente antes de chamar
// WriteAtomic. E o sincronismo que torna este teste um teste — ver
// rodarEscritorEMatar.
const avisoPronto = "pronto\n"

func TestMain(m *testing.M) {
	if os.Getenv("GO_TEST_HELPER_PROCESS") == "1" {
		targetPath := os.Getenv("GO_TEST_TARGET_PATH")
		data := []byte(os.Getenv("GO_TEST_DATA"))
		if targetPath != "" {
			// Avisa o pai que o init acabou e a escrita comeca AGORA.
			//
			// Escrever em stdout aqui nao fere a regra de stdout do projeto:
			// isto e um processo auxiliar de teste, nao um servidor MCP, e o
			// unico leitor do outro lado e o processo pai deste teste.
			fmt.Print(avisoPronto)
			_ = os.Stdout.Sync()

			_ = writer.WriteAtomic(targetPath, data)
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}

// rodarEscritorEMatar roda o escritor num processo filho e o mata `jitter`
// depois de o filho AVISAR que vai escrever.
//
// O aviso e a correcao de um defeito que tornava este teste vazio. A versao
// anterior matava num ponto fixo do relogio — 0 a 39 ms contados do
// cmd.Start() — e o filho e o proprio binario de teste: antes de chegar em
// WriteAtomic ele paga criacao de processo, init do runtime de Go e init do
// framework de teste. Medido em 2026-08-01 no maquina de referencia (12 nucleos, Windows
// 11), uma escrita NAO interrompida, do Start ao Wait:
//
//	sem -race:  min 43,3 ms   mediana 47,2 ms
//	com -race:  min 1,057 s   mediana 1,077 s
//
// A janela de 39 ms cabia inteira dentro do init. As mortes caiam antes de o
// temporario existir — sob -race, que e o unico modo em que verify.ps1 roda
// os testes, em 100% das 1.000 iteracoes. O teste reportava "0 corrompidas em
// 1.000 iteracoes" sem nunca ter escrito um byte, e "0 corrompidas" e o
// criterio de bloqueio do M4 (RNF-11).
//
// Quem denunciou foi a guarda de `orfaos == 0`, escrita junto com o teste e
// correta desde entao: ela se recusa a reportar cobertura que nao houve. Ela
// so nao disparava porque, sem -race, umas poucas iteracoes por rodada
// chegavam a escrever por sorte de escalonamento.
//
// Cronometrar desde o Start nao tem conserto por ajuste de constante: o init
// domina o relogio e muda com a maquina, com o modo e com a versao do Go.
// Sincronizar com a escrita tira o init da conta.
func rodarEscritorEMatar(t *testing.T, alvo string, novo []byte, jitter time.Duration) {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^$")
	cmd.Env = append(os.Environ(),
		"GO_TEST_HELPER_PROCESS=1",
		"GO_TEST_TARGET_PATH="+alvo,
		"GO_TEST_DATA="+string(novo),
	)

	saida, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("StdoutPipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("cmd.Start: %v", err)
	}

	// Espera o aviso. Se o filho morrer antes de avisar, ReadFull devolve erro
	// e a iteracao simplesmente nao exercita a janela — nao e falha, e o
	// contador de orfaos e quem diz se sobrou exercicio suficiente.
	buf := make([]byte, len(avisoPronto))
	avisou := false
	if _, err := io.ReadFull(saida, buf); err == nil && string(buf) == avisoPronto {
		avisou = true
	}

	if avisou {
		if jitter > 0 {
			time.Sleep(jitter)
		}
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

	// As iteracoes rodam em paralelo porque o custo de cada uma e o init do
	// processo filho — 1,08 s sob -race —, nao o trabalho que se quer medir.
	// Sequencial, as 1.000 levariam ~18 minutos, e um gate de 18 minutos e um
	// gate que alguem pula. Cada iteracao tem diretorio proprio: o unico
	// estado compartilhado sao os contadores, sob mutex.
	trabalhadores := min(runtime.NumCPU(), 12)

	var mu sync.Mutex
	corrompidas := 0
	orfaos := 0

	var wg sync.WaitGroup
	fila := make(chan int)

	trabalho := func(i int) {
		dir := t.TempDir()
		alvo := filepath.Join(dir, "nota.md")
		if err := os.WriteFile(alvo, antigo, 0644); err != nil {
			t.Errorf("iteracao %d: preparo: %v", i, err)
			return
		}

		// A janela e contada a partir do AVISO, nao do Start: 0 a 9,95 ms em
		// passos de 50 us. WriteAtomic cria o temporario, escreve, faz Sync e
		// renomeia — no Windows o Sync e a parte lenta, e e dentro dele que a
		// maior parte das mortes precisa cair para o teste medir atomicidade.
		rodarEscritorEMatar(t, alvo, novo, time.Duration(i%200)*50*time.Microsecond)

		lido, err := os.ReadFile(alvo)
		if err != nil {
			t.Errorf("iteracao %d: alvo desapareceu: %v", i, err)
			return
		}

		if !bytes.Equal(lido, antigo) && !bytes.Equal(lido, novo) {
			mu.Lock()
			corrompidas++
			mu.Unlock()
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
		mu.Lock()
		orfaos += len(antesDaVarredura)
		mu.Unlock()

		// t.Fatalf so pode ser chamado da goroutine do teste; aqui e Errorf.
		removidos, err := writer.SweepStaleTempFiles(context.Background(), dir)
		if err != nil {
			t.Errorf("iteracao %d: SweepStaleTempFiles: %v", i, err)
			return
		}
		if removidos != len(antesDaVarredura) {
			t.Errorf("iteracao %d: varredura removeu %d de %d temporarios",
				i, removidos, len(antesDaVarredura))
		}
		if sobras, _ := filepath.Glob(filepath.Join(dir, writer.TempFilePrefix+"*")); len(sobras) > 0 {
			t.Errorf("iteracao %d: temporario sobrou DEPOIS da varredura: %v", i, sobras)
		}
	}

	inicio := time.Now()
	for range trabalhadores {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range fila {
				trabalho(i)
			}
		}()
	}
	for i := range iteracoes {
		fila <- i
	}
	close(fila)
	wg.Wait()
	decorrido := time.Since(inicio)

	if corrompidas > 0 {
		t.Fatalf("RNF-11 REPROVADO: %d de %d iteracoes corromperam a nota", corrompidas, iteracoes)
	}

	// O numero de orfaos e informacao, nao falha: ele mede quantas iteracoes
	// morreram DEPOIS de criar o temporario e ANTES do rename, que e exatamente
	// a janela que a escrita atomica existe para cobrir. Zero orfaos em 1.000
	// iteracoes significaria que o crash nunca caiu nessa janela, e ai o teste
	// nao teria exercitado o que promete.
	t.Logf("RNF-11: 0 corrompidas em %d iteracoes; %d temporarios orfaos varridos "+
		"(%d trabalhadores, %v)", iteracoes, orfaos, trabalhadores, decorrido)
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
