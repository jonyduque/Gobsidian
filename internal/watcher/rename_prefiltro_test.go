package watcher_test

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/vault"
	"github.com/jonyd/gobsidian/internal/watcher"
)

// cofreComRename monta o cenário: uma nota já indexada some do disco, e uma
// nota nova aparece com o MESMO conteúdo — um rename. Junto vêm `ruido`
// arquivos novos de tamanhos diferentes, que o pré-filtro do achado P9 não
// precisa ler.
func cofreComRename(t testing.TB, ruido int) (*vault.Vault, *index.Index, []vault.CanonicalPath) {
	t.Helper()
	root := t.TempDir()
	const conteudo = "---\ntitle: Movida\n---\n\ncorpo estavel da nota movida.\n"

	esc := func(nome, corpo string) {
		if err := os.WriteFile(filepath.Join(root, nome), []byte(corpo), 0644); err != nil {
			t.Fatalf("escrevendo %s: %v", nome, err)
		}
	}
	esc("origem.md", conteudo)
	for i := range ruido {
		// Tamanhos DIFERENTES do da nota movida: nenhum pode ser o outro lado
		// do rename, e o pré-filtro dispensa ler todos eles.
		esc(fmt.Sprintf("ruido%04d.md", i), conteudo+fmt.Sprintf("%*s", i+1, "x"))
	}

	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	idx := index.New()
	if err := idx.Build(context.Background(), v); err != nil {
		t.Fatalf("Build: %v", err)
	}

	// Agora aplica o rename no disco: origem some, destino aparece.
	if err := os.Remove(filepath.Join(root, "origem.md")); err != nil {
		t.Fatalf("removendo origem: %v", err)
	}
	esc("destino.md", conteudo)

	lote := []vault.CanonicalPath{"origem.md", "destino.md"}
	for i := range ruido {
		lote = append(lote, vault.CanonicalPath(fmt.Sprintf("ruido%04d.md", i)))
	}
	return v, idx, lote
}

// TestPreFiltroPorTamanhoNaoPerdeRename é o achado P9, e o risco da correção é
// maior que o do defeito: um pré-filtro cedo demais deixa de detectar um rename
// de verdade, e aí os backlinks da nota movida quebram em silêncio.
//
// O filtro é EXATO, não heurístico — hashes iguais exigem bytes iguais, que
// exigem tamanho igual — e este teste é o que garante que ele continua assim.
func TestPreFiltroPorTamanhoNaoPerdeRename(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	v, idx, lote := cofreComRename(t, 20)

	renames, naoRenames := watcher.CorrelateRenames(context.Background(), lote, v, idx, log)

	if len(renames) != 1 {
		t.Fatalf("renames = %d, queria 1: o pre-filtro descartou o rename de verdade\n%+v", len(renames), renames)
	}
	if renames[0].From != "origem.md" || renames[0].To != "destino.md" {
		t.Errorf("rename = %s -> %s, queria origem.md -> destino.md", renames[0].From, renames[0].To)
	}
	// E o ruído continua saindo como não-rename, não sumindo do lote.
	if len(naoRenames) != 20 {
		t.Errorf("naoRenames = %d, queria 20: o pre-filtro nao pode ENGOLIR os candidatos que descarta", len(naoRenames))
	}
}

// BenchmarkCorrelateRenamesComRuido mede o que o P9 nomeia: sem o pré-filtro,
// TODA nota adicionada do lote é lida do disco quando há qualquer remoção.
func BenchmarkCorrelateRenamesComRuido(b *testing.B) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	v, idx, lote := cofreComRename(b, 300)
	ctx := context.Background()
	b.ResetTimer()
	for b.Loop() {
		if r, _ := watcher.CorrelateRenames(ctx, lote, v, idx, log); len(r) != 1 {
			b.Fatalf("renames = %d, queria 1", len(r))
		}
	}
}
