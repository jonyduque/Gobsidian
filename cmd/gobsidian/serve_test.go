package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"testing"
	"time"
)

// boundedWait e o prazo usado para esperar por um lado do pipe num teste.
// Uma regressao real (por exemplo, apagar o CloseWithError que propaga EOF)
// nao deve travar o binario de teste ate o timeout padrao de 10 minutos do
// "go test ./..." sem -timeout: precisa reprovar em milissegundos, com uma
// mensagem que diz o que faltou.
const boundedWait = 2 * time.Second

// eofReader devolve os bytes de data, em quantas chamadas a Read forem
// necessarias para o buffer do chamador, e so entao io.EOF. Ao contrario de
// um copy-e-sent-incondicional, ele nao descarta o resto dos dados quando o
// buffer do chamador e menor que len(data) — importante porque
// TestMirrorReaderBrokenMirrorDoesNotPoisonRead depende de duas leituras
// verem, cada uma, um pedaco diferente.
type eofReader struct {
	data []byte
	pos  int
}

func (r *eofReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

// chunkReader devolve uma fatia de data por chamada a Read, imitando uma
// origem que entrega bytes em pedacos distintos (como um stdin real, em
// leituras sucessivas) em vez de tudo de uma vez.
type chunkReader struct {
	chunks [][]byte
	idx    int
}

func (r *chunkReader) Read(p []byte) (int, error) {
	if r.idx >= len(r.chunks) {
		return 0, io.EOF
	}
	n := copy(p, r.chunks[r.idx])
	r.idx++
	return n, nil
}

// spyDst implementa mirrorDst e conta quantas vezes Write foi chamado, para
// que o teste do latch broken possa afirmar que uma segunda tentativa de
// escrita NAO aconteceu — coisa que o valor de retorno de Read sozinho nao
// expoe, porque e o mesmo com ou sem a guarda !m.broken.
type spyDst struct {
	writes int
	fail   bool
}

func (s *spyDst) Write(p []byte) (int, error) {
	s.writes++
	if s.fail {
		return 0, io.ErrClosedPipe
	}
	return len(p), nil
}

func (s *spyDst) CloseWithError(error) error { return nil }

func TestMirrorReaderCopiesToMirror(t *testing.T) {
	src := &eofReader{data: []byte("ola mundo")}
	pr, pw := io.Pipe()
	m := &mirrorReader{src: src, dst: pw}

	// Fechar as duas pontas no fim do teste desbloqueia qualquer goroutine
	// abaixo que ainda esteja presa em Read/Write caso o teste reprove pelo
	// timeout: sem isso, uma mutacao que apaga a copia deixaria uma
	// goroutine vazando ate o processo de teste inteiro terminar.
	t.Cleanup(func() {
		_ = pr.Close()
		_ = pw.Close()
	})

	// A leitura da mirrorReader precisa acontecer concorrentemente com a
	// leitura do lado espelhado: io.Pipe nao tem buffer, entao o Write
	// dentro de Read bloqueia ate alguem ler do outro lado.
	type readResult struct {
		n   int
		err error
	}
	readDone := make(chan readResult, 1)
	buf := make([]byte, 64)
	go func() {
		n, err := m.Read(buf)
		readDone <- readResult{n, err}
	}()

	mirrored := make([]byte, len("ola mundo"))
	mirrorDone := make(chan error, 1)
	go func() {
		_, err := io.ReadFull(pr, mirrored)
		mirrorDone <- err
	}()

	// Acha 1: sem limite de tempo aqui, uma mutacao que apaga o bloco de
	// escrita no espelho pendura o io.ReadFull para sempre e o "go test
	// -race ./..." do projeto (sem -timeout) so reprova depois de 10
	// minutos, derrubando o binario do pacote inteiro com um dump de
	// goroutines em vez de uma falha nomeada.
	select {
	case err := <-mirrorDone:
		if err != nil {
			t.Fatalf("leitura do espelho falhou: %v", err)
		}
	case <-time.After(boundedWait):
		t.Fatalf("leitura do espelho nao chegou em %s: mirrorReader parou de copiar para o espelho", boundedWait)
	}

	var res readResult
	select {
	case res = <-readDone:
	case <-time.After(boundedWait):
		t.Fatalf("Read da origem nao retornou em %s", boundedWait)
	}

	if res.err != nil {
		t.Fatalf("Read da origem devolveu erro inesperado: %v", res.err)
	}
	if got := string(buf[:res.n]); got != "ola mundo" {
		t.Fatalf("Read devolveu %q, esperado %q", got, "ola mundo")
	}
	if !bytes.Equal(mirrored, []byte("ola mundo")) {
		t.Fatalf("espelho recebeu %q, esperado %q", mirrored, "ola mundo")
	}
}

func TestMirrorReaderPropagatesEOF(t *testing.T) {
	src := &eofReader{} // origem ja no fim: nenhum dado, pos 0 >= len(data) 0
	pr, pw := io.Pipe()
	m := &mirrorReader{src: src, dst: pw}

	t.Cleanup(func() {
		_ = pr.Close()
		_ = pw.Close()
	})

	n, err := m.Read(make([]byte, 8))
	if n != 0 {
		t.Fatalf("Read devolveu %d bytes no EOF, esperado 0", n)
	}
	if !errors.Is(err, io.EOF) {
		t.Fatalf("Read devolveu err=%v, esperado io.EOF", err)
	}

	// E o comportamento que io.TeeReader nao tem: sem o CloseWithError do
	// mirrorReader, o lado de leitura do pipe nunca veria o fim, e o monitor
	// de stdin do lifecycle ficaria inerte esperando um EOF que nunca chega.
	//
	// Acha 1: a leitura abaixo roda numa goroutine com prazo, nao inline —
	// sem CloseWithError ela pendura, e sem o limite de tempo essa pendura
	// so aparece depois do timeout padrao de 10 minutos do "go test
	// -race ./..." sem -timeout, matando o binario do pacote inteiro.
	type readResult struct {
		n   int
		err error
	}
	rc := make(chan readResult, 1)
	go func() {
		n, err := pr.Read(make([]byte, 8))
		rc <- readResult{n, err}
	}()

	select {
	case res := <-rc:
		if !errors.Is(res.err, io.EOF) {
			t.Fatalf("lado de leitura do pipe devolveu err=%v, esperado io.EOF", res.err)
		}
	case <-time.After(boundedWait):
		t.Fatalf("lado de leitura do pipe nao recebeu EOF em %s: mirrorReader nao fechou dst", boundedWait)
	}
}

func TestMirrorReaderBrokenMirrorDoesNotPoisonRead(t *testing.T) {
	// Duas leituras da origem, em dois pedacos: a primeira aciona o latch
	// broken (o espelho falha), a segunda precisa provar que o latch
	// realmente impediu uma nova tentativa de escrita — nao so que o campo
	// ficou true.
	src := &chunkReader{chunks: [][]byte{[]byte("dados"), []byte("mais")}}
	dst := &spyDst{fail: true}
	m := &mirrorReader{src: src, dst: dst}

	n, err := m.Read(make([]byte, 64))
	if err != nil {
		t.Fatalf("Read devolveu err=%v, esperado nil (erro e da origem, nao do espelho)", err)
	}
	if n != len("dados") {
		t.Fatalf("Read devolveu n=%d, esperado %d", n, len("dados"))
	}
	if !m.broken {
		t.Fatal("mirrorReader.broken deveria estar true apos escrita falhar no espelho")
	}
	if dst.writes != 1 {
		t.Fatalf("espelho recebeu %d tentativas de escrita apos a primeira leitura, esperado 1", dst.writes)
	}

	// Segunda leitura: dados novos chegam (n > 0), mas o espelho ja esta
	// broken. Se a guarda !m.broken fosse removida, esta leitura tentaria
	// escrever de novo no espelho e dst.writes subiria para 2 — e isso, nao
	// o valor de retorno de Read, e o que este teste trava.
	n2, err2 := m.Read(make([]byte, 64))
	if err2 != nil {
		t.Fatalf("segunda leitura devolveu err=%v, esperado nil", err2)
	}
	if n2 != len("mais") {
		t.Fatalf("segunda leitura devolveu n=%d, esperado %d", n2, len("mais"))
	}
	if dst.writes != 1 {
		t.Fatalf("espelho recebeu %d tentativas de escrita apos ficar broken, esperado continuar em 1", dst.writes)
	}

	// Terceira leitura: a origem chegou ao fim. Confirma que o espelho
	// quebrado nao deixou a leitura principal travada nem incorreta.
	n3, err3 := m.Read(make([]byte, 64))
	if n3 != 0 || !errors.Is(err3, io.EOF) {
		t.Fatalf("terceira leitura devolveu (%d, %v), esperado (0, io.EOF)", n3, err3)
	}
}

func TestShutdownExitCode(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
	}{
		{"nil", nil, 0},
		{"context.Canceled", context.Canceled, 0},
		{"erro embrulhado com context.Canceled", fmt.Errorf("serve: %w", context.Canceled), 0},
		{"io.EOF", io.EOF, 0},
		{"erro embrulhado com io.EOF", fmt.Errorf("sdk: %w", io.EOF), 0},
		{"io.ErrClosedPipe", io.ErrClosedPipe, 0},
		{"erro embrulhado com io.ErrClosedPipe", fmt.Errorf("sdk: %w", io.ErrClosedPipe), 0},
		{"erro real", errors.New("falha real"), 1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := shutdownExitCode(tc.err); got != tc.want {
				t.Fatalf("shutdownExitCode(%v) = %d, esperado %d", tc.err, got, tc.want)
			}
		})
	}
}
