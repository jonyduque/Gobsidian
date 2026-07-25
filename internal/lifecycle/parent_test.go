package lifecycle_test

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/jonyd/gobsidian/internal/lifecycle"
)

func TestParentGoneCancelsContext(t *testing.T) {
	// Um processo de vida longa serve de "pai" sintetico. Ele precisa estar
	// VIVO quando New e chamado: parentIdentity captura a identidade inicial
	// no startup, e se essa captura falhar a vigilia se desabilita em vez de
	// disparar — comportamento correto para um PID que nunca foi observavel,
	// e que tornaria este teste vacuo se o processo ja estivesse morto aqui.
	// Sem o wrapper "cmd /c": com ele, Kill() mata apenas o cmd.exe e deixa
	// ping.exe orfao rodando por ate ~29s — no proprio teste de um mecanismo
	// anti-orfao. Invocar ping.exe diretamente faz do PID matado o pinger de
	// verdade.
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("ping", "-n", "30", "127.0.0.1")
	} else {
		cmd = exec.Command("sleep", "30")
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	pid := cmd.Process.Pid

	pr, pw := io.Pipe()
	t.Cleanup(func() { _ = pw.Close() })

	ctx, lc := lifecycle.New(context.Background(), lifecycle.Options{
		Stdin:               pr,
		ParentPID:           pid,
		ParentCheckInterval: 50 * time.Millisecond,
	})

	// Agora que a identidade inicial foi capturada, mate o pai.
	if err := cmd.Process.Kill(); err != nil {
		t.Fatalf("Kill: %v", err)
	}
	_ = cmd.Wait() // libera o handle; sem isso o PID continua consultavel

	select {
	case <-ctx.Done():
	case <-time.After(3 * time.Second):
		t.Fatal("context nao foi cancelado apos o pai morrer")
	}

	if got := lc.Reason(); got != "parent-gone" {
		t.Errorf("Reason() = %q, quer %q", got, "parent-gone")
	}

	// Guardado com timeout pelo mesmo motivo que em TestLiveParentKeepsContextAlive:
	// numa regressao, um Wait() nu trava o binario de teste ate o -timeout do
	// go test estourar, em vez de falhar de forma limpa e localizada aqui.
	done := make(chan struct{})
	go func() { lc.Wait(); close(done) }()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Wait() nao retornou apos o cancelamento do context")
	}
}

func TestLiveParentKeepsContextAlive(t *testing.T) {
	pr, pw := io.Pipe()
	t.Cleanup(func() { _ = pw.Close() })

	// Um logger proprio, em vez do sink padrao (io.Discard), da ao teste um
	// sinal positivo de que a vigilia esta de fato rodando e comparando
	// identidade, nao apenas silenciosamente desabilitada. Se a captura
	// inicial de parentIdentity falhasse, watchParent registraria o aviso
	// "vigilia do processo pai desabilitada" e retornaria antes de wg.Add(1)
	// — nenhuma goroutine rodaria, o context nunca seria cancelado, e a
	// asserção de "ainda vivo apos 500ms" abaixo passaria pelo motivo
	// errado. Checar a ausencia do aviso distingue "vigiando e batendo
	// identidade" de "nunca comecou a vigiar".
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, nil))

	// O proprio processo de teste como pai: esta vivo por definicao.
	parent, cancelParent := context.WithCancel(context.Background())
	ctx, lc := lifecycle.New(parent, lifecycle.Options{
		Stdin:               pr,
		ParentPID:           os.Getpid(),
		ParentCheckInterval: 50 * time.Millisecond,
		Logger:              logger,
	})

	select {
	case <-ctx.Done():
		t.Fatal("context cancelado com o pai vivo")
	case <-time.After(500 * time.Millisecond):
	}

	if strings.Contains(logBuf.String(), "vigilia do processo pai desabilitada") {
		t.Errorf("vigilia se desabilitou na captura inicial; log: %s", logBuf.String())
	}

	// As duas assercoes acima ja foram feitas. O cancelamento abaixo existe
	// para desenrolar as goroutines: tanto o vigia de sinais quanto o do pai
	// bloqueiam em ctx.Done(), entao Wait() so retorna depois que o context
	// morre. Chamar Wait() sem cancelar antes trava para sempre.
	cancelParent()

	done := make(chan struct{})
	go func() { lc.Wait(); close(done) }()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Wait() nao retornou apos o cancelamento do context")
	}
}
