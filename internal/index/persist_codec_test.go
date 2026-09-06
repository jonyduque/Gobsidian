package index

import (
	"bufio"
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jonyd/gobsidian/internal/parser"
)

func roundTrip(t *testing.T, v any) any {
	t.Helper()
	var buf bytes.Buffer
	bw := bufio.NewWriter(&buf)
	e := &escritor{w: bw}
	e.value(v)
	if e.err != nil {
		t.Fatalf("escrevendo %#v: %v", v, e.err)
	}
	if err := bw.Flush(); err != nil {
		t.Fatal(err)
	}
	l := &leitor{b: buf.Bytes()}
	got := l.value(0)
	if l.err != nil {
		t.Fatalf("lendo %#v: %v", v, l.err)
	}
	if l.i != len(l.b) {
		t.Fatalf("sobraram %d bytes depois de ler %#v", len(l.b)-l.i, v)
	}
	return got
}

func TestCodecValorRoundTripPorTipo(t *testing.T) {
	zona := time.FixedZone("X", -3*3600)
	casos := []any{
		nil, true, false, int(-7), int64(1 << 40), uint64(1<<63 + 5), float64(2.5),
		"acao", "", time.Date(2026, 9, 2, 10, 0, 0, 123, zona),
		// []any(nil) e um []any TIPADO com valor nil — distinto do `nil` solto
		// acima (que vira valNil) e de []any{} abaixo (que vira valSlice de
		// tamanho 0). Cobre o par valSliceNil/valSlice (escritor :294-299,
		// leitor :659-660), que ficava a 0% de cobertura antes desta rodada
		// (achado N1 da revisao da Task 181). O escritor grava valSliceNil e o
		// leitor devolve []any(nil): reflect.DeepEqual distingue nil de vazio
		// para slice, entao o round-trip abaixo prova a distincao real.
		[]any(nil),
		[]any{int(1), "dois", nil}, []any{}, map[string]any{"k": "v", "n": int(3)}, map[string]any{},
		[]any{map[string]any{"a": []any{int(1)}}},
	}
	for _, in := range casos {
		got := roundTrip(t, in)
		if in == nil {
			if got != nil {
				t.Errorf("nil virou %#v", got)
			}
			continue
		}
		if reflect.TypeOf(got) != reflect.TypeOf(in) {
			t.Errorf("%#v (%T) voltou como %T", in, in, got)
			continue
		}
		if tm, ok := in.(time.Time); ok {
			if !tm.Equal(got.(time.Time)) {
				t.Errorf("time %v voltou %v", tm, got)
			}
			continue
		}
		if !reflect.DeepEqual(got, in) {
			t.Errorf("%#v voltou %#v", in, got)
		}
	}
}

func TestCodecStrSliceNilDistintoDeVazio(t *testing.T) {
	for _, in := range [][]string{nil, {}} {
		var buf bytes.Buffer
		bw := bufio.NewWriter(&buf)
		e := &escritor{w: bw}
		e.strSlice(in)
		if e.err != nil {
			t.Fatal(e.err)
		}
		if err := bw.Flush(); err != nil {
			t.Fatal(err)
		}
		l := &leitor{b: buf.Bytes()}
		got := l.strSlice("teste")
		if l.err != nil {
			t.Fatal(l.err)
		}
		if (got == nil) != (in == nil) {
			t.Errorf("strSlice %#v voltou %#v (nil-ness diferente)", in, got)
		}
	}
}

func TestCodecTimeBlobPreservaZona(t *testing.T) {
	in := time.Date(2026, 9, 2, 10, 0, 0, 7, time.FixedZone("Y", 5*3600+1800))
	var buf bytes.Buffer
	bw := bufio.NewWriter(&buf)
	e := &escritor{w: bw}
	e.timeBlob(in)
	if e.err != nil {
		t.Fatal(e.err)
	}
	if err := bw.Flush(); err != nil {
		t.Fatal(err)
	}
	l := &leitor{b: buf.Bytes()}
	got := l.timeBlob("teste")
	if l.err != nil {
		t.Fatal(l.err)
	}
	if !got.Equal(in) {
		t.Fatalf("%v voltou %v", in, got)
	}
	_, off1 := in.Zone()
	_, off2 := got.Zone()
	if off1 != off2 {
		t.Fatalf("offset de zona %d voltou %d", off1, off2)
	}
}

func TestCodecValorTipoNaoSuportadoFalha(t *testing.T) {
	var buf bytes.Buffer
	bw := bufio.NewWriter(&buf)
	e := &escritor{w: bw}
	e.value(struct{}{})
	if e.err == nil || !strings.Contains(e.err.Error(), "tipo nao suportado") {
		t.Fatalf("struct{}{} devia falhar com \"tipo nao suportado\", tenho %v", e.err)
	}
}

func TestCodecValorProfundidadeAlemDoLimiteERecusada(t *testing.T) {
	var v any = "folha"
	for i := 0; i < limiteValorProfund+6; i++ {
		v = []any{v}
	}
	var buf bytes.Buffer
	bw := bufio.NewWriter(&buf)
	e := &escritor{w: bw}
	e.value(v)
	if e.err == nil {
		// O escritor nao limita profundidade (so serializa); e o leitor que tem
		// de recusar, que e o caminho testado abaixo.
		if err := bw.Flush(); err != nil {
			t.Fatal(err)
		}
		l := &leitor{b: buf.Bytes()}
		_ = l.value(0)
		if l.err == nil || !errors.Is(l.err, ErrIndexCacheCorrupted) || !strings.Contains(l.err.Error(), "profundidade") {
			t.Fatalf("profundidade %d devia ser recusada na leitura, tenho %v", limiteValorProfund+6, l.err)
		}
		return
	}
	if !strings.Contains(e.err.Error(), "profundidade") {
		t.Fatalf("escritor recusou por outro motivo: %v", e.err)
	}
}

// TestCodecTagDesconhecidaERecusada: a leitura da tag de valor
// (leitor.value, persist_codec.go:638) passa por l.uvarint(uint64(valMap),
// "tipo de valor") — o MESMO uvarint que aplica o teto valMap como LIMITE.
// Uma tag = valMap+1 nunca alcanca o `default` do switch (:684): ela e
// recusada antes, dentro de leitor.uvarint, com a mensagem "acima do
// limite" — nao "tag de valor desconhecida". O branch default do switch e
// codigo morto para qualquer tag > valMap; so seria alcancavel por uma tag
// dentro de [0, valMap] que o switch nao cobrisse, e o switch cobre as
// doze tags inteiras (valNil..valMap). Achado durante a Task 181; nao e
// corrigido aqui (so teste/doc) — ver docs/SUGESTOES.md B21.
func TestCodecTagDesconhecidaERecusada(t *testing.T) {
	l := &leitor{b: []byte{valMap + 1}}
	_ = l.value(0)
	if l.err == nil || !errors.Is(l.err, ErrIndexCacheCorrupted) || !strings.Contains(l.err.Error(), "acima do limite") {
		t.Fatalf("tag %d devia ser recusada pelo teto do uvarint, tenho %v", valMap+1, l.err)
	}
}

func TestCodecStringAcimaDoLimiteERecusada(t *testing.T) {
	var buf bytes.Buffer
	bw := bufio.NewWriter(&buf)
	e := &escritor{w: bw}
	e.uvarint(uint64(limiteString + 1))
	if e.err != nil {
		t.Fatal(e.err)
	}
	if err := bw.Flush(); err != nil {
		t.Fatal(err)
	}
	l := &leitor{b: buf.Bytes()}
	_ = l.str("teste")
	if l.err == nil || !strings.Contains(l.err.Error(), "acima do limite") {
		t.Fatalf("string de %d bytes devia ser recusada, tenho %v", limiteString+1, l.err)
	}
}

func TestCodecNotaTruncadaEmCadaByteERecusada(t *testing.T) {
	n := &Note{
		Path: "pasta/nota.md", Title: "Nota", Hash: 123,
		Tags: []string{"a", "b/c"}, Aliases: []string{"x"},
		Frontmatter: map[string]any{"k": "v", "n": int64(2), "lista": []any{"a", int(1)}},
		ModTime:     time.Now(),
		Headings: []parser.Heading{
			{Level: 1, Text: "titulo", Slug: "titulo", Start: 0, End: 10, BodyStart: 11},
		},
		Blocks: []parser.Block{
			{ID: "bloco1", Start: 20, End: 30},
		},
		Links: []ResolvedLink{
			{
				Link: parser.Link{
					Raw: "[[alvo]]", Target: "alvo", Alias: "", Anchor: "",
					Kind: parser.LinkWiki, Start: 40, End: 48,
				},
				Context: "contexto do link",
			},
		},
		Inline: map[string][]string{"campo": {"valor1", "valor2"}},
	}
	var buf bytes.Buffer
	bw := bufio.NewWriter(&buf)
	e := &escritor{w: bw}
	e.note(n)
	if e.err != nil {
		t.Fatal(e.err)
	}
	if err := bw.Flush(); err != nil {
		t.Fatal(err)
	}
	b := buf.Bytes()
	for i := 0; i < len(b); i++ {
		l := &leitor{b: b[:i]}
		_ = l.note()
		if l.err == nil {
			t.Fatalf("prefixo de %d/%d bytes foi aceito como nota completa", i, len(b))
		}
		if !errors.Is(l.err, ErrIndexCacheCorrupted) {
			t.Fatalf("prefixo de %d/%d bytes falhou sem ErrIndexCacheCorrupted: %v", i, len(b), l.err)
		}
	}
	l := &leitor{b: b}
	got := l.note()
	if l.err != nil || got == nil || got.Path != n.Path {
		t.Fatalf("a nota inteira devia ler: err=%v got=%+v", l.err, got)
	}
}
