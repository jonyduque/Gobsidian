package writer_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/jonyd/gobsidian/internal/parser"
	"github.com/jonyd/gobsidian/internal/writer"
)

// BenchmarkRewriteLinksMuitos mede o laco de RewriteLinks, que hoje realoca o
// buffer inteiro a cada substituicao (O(n*m)). Nota com 200 links, todos
// reescritos, e o caso de note_move numa nota-indice.
func BenchmarkRewriteLinksMuitos(b *testing.B) {
	var src bytes.Buffer
	for i := range 200 {
		fmt.Fprintf(&src, "Paragrafo %d com texto de enchimento suficiente para parecer uma nota. Veja [[Nota_%d]] agora.\n\n", i, i)
	}
	pn := parser.Parse(src.Bytes())
	if len(pn.Links) != 200 {
		b.Fatalf("links = %d, quer 200", len(pn.Links))
	}
	reps := make([]writer.LinkReplacement, len(pn.Links))
	for i, l := range pn.Links {
		reps[i] = writer.LinkReplacement{Link: l, NewTarget: fmt.Sprintf("Pasta/Nova_%d", i)}
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		out, err := writer.RewriteLinks(src.Bytes(), reps)
		if err != nil || len(out) <= src.Len() {
			b.Fatalf("RewriteLinks: %v, %d bytes", err, len(out))
		}
	}
}
