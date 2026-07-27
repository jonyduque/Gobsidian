package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"testing"
)

// erroredReader devolve os bytes de data e depois io.EOF, imitando um stdin
// que chega ao fim — sem depender de um os.Pipe real para simular o fim da
// origem.
type erroredReader struct {
	data []byte
	sent bool
}

func (r *erroredReader) Read(p []byte) (int, error) {
	if r.sent {
		return 0, io.EOF
	}
	n := copy(p, r.data)
	r.sent = true
	return n, nil
}

func TestMirrorReaderCopiesToMirror(t *testing.T) {
	src := &erroredReader{data: []byte("ola mundo")}
	pr, pw := io.Pipe()
	m := &mirrorReader{src: src, dst: pw}

	// A leitura da mirrorReader precisa acontecer concorrentemente com a
	// leitura do lado espelhado: io.Pipe nao tem buffer, entao o Write dentro
	// de Read bloqueia ate alguem ler do outro lado.
	readDone := make(chan struct{})
	buf := make([]byte, 64)
	var n int
	var err error
	go func() {
		n, err = m.Read(buf)
		close(readDone)
	}()

	mirrored := make([]byte, len("ola mundo"))
	if _, rerr := io.ReadFull(pr, mirrored); rerr != nil {
		t.Fatalf("leitura do espelho falhou: %v", rerr)
	}
	<-readDone

	if err != nil {
		t.Fatalf("Read da origem devolveu erro inesperado: %v", err)
	}
	if got := string(buf[:n]); got != "ola mundo" {
		t.Fatalf("Read devolveu %q, esperado %q", got, "ola mundo")
	}
	if !bytes.Equal(mirrored, []byte("ola mundo")) {
		t.Fatalf("espelho recebeu %q, esperado %q", mirrored, "ola mundo")
	}
}

func TestMirrorReaderPropagatesEOF(t *testing.T) {
	src := &erroredReader{data: nil, sent: true} // origem ja no fim
	pr, pw := io.Pipe()
	m := &mirrorReader{src: src, dst: pw}

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
	_, rerr := pr.Read(make([]byte, 8))
	if !errors.Is(rerr, io.EOF) {
		t.Fatalf("lado de leitura do pipe devolveu err=%v, esperado io.EOF", rerr)
	}
}

func TestMirrorReaderBrokenMirrorDoesNotPoisonRead(t *testing.T) {
	src := &erroredReader{data: []byte("dados")}
	pr, pw := io.Pipe()
	// Fecha a ponta de leitura antes de qualquer Read: a proxima escrita em
	// pw devolve io.ErrClosedPipe, simulando um espelho quebrado.
	if err := pr.Close(); err != nil {
		t.Fatalf("fechar ponta de leitura falhou: %v", err)
	}
	m := &mirrorReader{src: src, dst: pw}

	n, err := m.Read(make([]byte, 64))
	if err != nil {
		t.Fatalf("Read devolveu err=%v, esperado nil (erro e da origem, nao do espelho)", err)
	}
	if got := n; got != len("dados") {
		t.Fatalf("Read devolveu n=%d, esperado %d", got, len("dados"))
	}
	if !m.broken {
		t.Fatal("mirrorReader.broken deveria estar true apos escrita falhar no espelho")
	}

	// Leituras seguintes continuam funcionando: a origem nao foi envenenada
	// pelo espelho quebrado.
	src2Buf := make([]byte, 64)
	n2, err2 := m.Read(src2Buf)
	if n2 != 0 || !errors.Is(err2, io.EOF) {
		t.Fatalf("segunda leitura devolveu (%d, %v), esperado (0, io.EOF)", n2, err2)
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
		{"io.ErrClosedPipe", io.ErrClosedPipe, 0},
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
