package parser_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/jonyd/gobsidian/internal/parser"
)

func notaLonga() []byte {
	var src bytes.Buffer
	for i := range 300 {
		fmt.Fprintf(&src, "**%d.%d Secao em negrito**\n\nTexto com [[Wiki_%d]] e [md](Nota%d.md) e #tag%d e campo:: valor.\n\n", i/10, i%10, i, i, i%7)
		if i%10 == 0 {
			src.WriteString("```go\nfunc x() {}\n```\n\n")
		}
	}
	return src.Bytes()
}

func BenchmarkParseNotaLonga(b *testing.B) {
	src := notaLonga()
	b.SetBytes(int64(len(src)))
	b.ReportAllocs()
	for b.Loop() {
		pn := parser.Parse(src)
		if len(pn.Links) < 600 {
			b.Fatalf("links = %d", len(pn.Links))
		}
	}
}

func BenchmarkDetectCandidatesNotaLonga(b *testing.B) {
	src := notaLonga()
	b.SetBytes(int64(len(src)))
	b.ReportAllocs()
	for b.Loop() {
		if cs := parser.DetectCandidates(src, 0); len(cs) != 300 {
			b.Fatalf("candidatos = %d", len(cs))
		}
	}
}
