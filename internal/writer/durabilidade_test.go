package writer_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/jonyd/gobsidian/internal/writer"
)

// TestWriteAtomicPreservaOModoDoAlvo é metade do achado M12.
//
// `os.CreateTemp` cria com 0600, e o rename leva esse modo para o alvo. Uma nota
// que era 0644 vira 0600: a escrita diz que preservou o conteúdo e muda a
// permissão pelas costas. Em cofre compartilhado por grupo, a nota some para os
// outros.
//
// No Windows o runtime do Go só mapeia o bit de somente-leitura, então o teste
// pula — mas a regra vale, porque o cofre pode estar num share lido de Linux.
func TestWriteAtomicPreservaOModoDoAlvo(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("o runtime do Go so mapeia o bit de somente-leitura no Windows; a regra vale, o teste nao a exercita aqui")
	}
	dir := t.TempDir()
	alvo := filepath.Join(dir, "nota.md")
	if err := os.WriteFile(alvo, []byte("antes\n"), 0644); err != nil {
		t.Fatalf("preparando o alvo: %v", err)
	}
	// Sem isto o teste compararia 0644 com 0644 por coincidência do umask.
	if err := os.Chmod(alvo, 0644); err != nil {
		t.Fatalf("chmod: %v", err)
	}

	if err := writer.WriteAtomic(context.Background(), alvo, []byte("depois\n")); err != nil {
		t.Fatalf("WriteAtomic: %v", err)
	}

	info, err := os.Stat(alvo)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if got := info.Mode().Perm(); got != 0644 {
		t.Errorf("modo depois da escrita = %04o, era 0644 antes\n"+
			"o temporario nasce 0600 e o rename levou esse modo para o alvo", got)
	}
}

// TestWriteAtomicRespeitaCancelamento é o achado M13.
//
// O laço de rename dorme até 100 ms tentando de novo, e não havia `ctx` nenhum
// na assinatura: a regra desta base é "ctx onde há espera real". Um contexto já
// cancelado tem de voltar sem escrever.
func TestWriteAtomicRespeitaCancelamento(t *testing.T) {
	dir := t.TempDir()
	alvo := filepath.Join(dir, "nota.md")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	inicio := time.Now()
	err := writer.WriteAtomic(ctx, alvo, []byte("conteudo\n"))
	if err == nil {
		t.Fatal("contexto cancelado foi ignorado: a escrita seguiu")
	}
	if decorrido := time.Since(inicio); decorrido > time.Second {
		t.Errorf("demorou %s para desistir de um contexto ja cancelado", decorrido)
	}

	// E o contrapeso: não pode ter deixado nada para trás.
	if _, err := os.Stat(alvo); err == nil {
		t.Error("a nota foi criada apesar do cancelamento")
	}
	sobras, _ := filepath.Glob(filepath.Join(dir, writer.TempFilePrefix+"*"))
	if len(sobras) > 0 {
		t.Errorf("temporario sobrou depois do cancelamento: %v", sobras)
	}
}
