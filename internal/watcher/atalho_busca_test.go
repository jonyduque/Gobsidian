package watcher

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/search"
	"github.com/jonyd/gobsidian/internal/vault"
)

// TestAtalhoDoApplyConsultaOIndiceDeBusca cobre a assimetria A7.
//
// O atalho de mtime/tamanho do Apply consultava SÓ idx.Get. A reconciliação por
// overflow já tinha aprendido a condição certa — overflow.go:58 exige
// `ok && (inv == nil || inv.HasDoc(...))` — e o Apply não a espelhou. São duas
// cópias de uma regra, e a que está errada é a MAIS usada.
//
// Consequência: um único `searchInv.Update` falho (que só produz log.Warn)
// deixa os metadados em dia e a posting ausente. A partir daí, TODO evento com
// mtime e tamanho iguais cai no `continue` e o índice de busca nunca se
// recompõe. E o OneDrive re-emite eventos de arquivos intocados como rotina,
// então a condição de disparo não é rara.
//
// O teste monta exatamente esse estado — metadados presentes, posting ausente,
// arquivo intocado — e afirma que o índice de busca se recompõe.
func TestAtalhoDoApplyConsultaOIndiceDeBusca(t *testing.T) {
	dir := t.TempDir()
	nota := filepath.Join(dir, "nota.md")
	const termo = "palavradetestesingular"
	if err := os.WriteFile(nota, []byte("# Titulo\n\n"+termo+" no corpo.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	v, err := vault.New(dir)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	idx := index.New()
	if err := idx.Build(context.Background(), v); err != nil {
		t.Fatalf("idx.Build: %v", err)
	}
	inv := search.NewInverted()
	canon := vault.CanonicalPath("nota.md")
	if err := inv.Update(context.Background(), v, canon); err != nil {
		t.Fatalf("inv.Update: %v", err)
	}

	// Guarda de cenário: sem isto, a asserção final poderia passar num estado
	// em que nada nunca esteve indexado, e o teste não mediria nada.
	if !inv.HasDoc("nota.md") {
		t.Fatal("cenario invalido: a nota nem chegou ao indice de busca")
	}

	// O estado que um Update falho produz: metadados em dia, posting ausente,
	// arquivo INTOCADO (mtime e tamanho iguais).
	inv.Remove("nota.md")
	if inv.HasDoc("nota.md") {
		t.Fatal("cenario invalido: Remove nao tirou a posting")
	}

	entrada := make(chan []vault.CanonicalPath, 1)
	reconcile := make(chan struct{})
	var processed, skipped, recUp, recRem atomic.Int64
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go Apply(ctx, entrada, reconcile, idx, v, log, &processed, &skipped, &recUp, &recRem, inv)

	entrada <- []vault.CanonicalPath{canon}

	limite := time.Now().Add(5 * time.Second)
	for time.Now().Before(limite) {
		if inv.HasDoc("nota.md") {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Errorf("o indice de busca NAO se recompos: o atalho de mtime/tamanho pulou o "+
		"evento sem consultar HasDoc (skipped=%d, processed=%d)",
		skipped.Load(), processed.Load())
}

// TestAtalhoDoApplyAindaPulaQuandoTudoEstaEmDia é o contrapeso.
//
// Sem ele, a correção poderia ser "nunca pular", e o atalho — que existe para
// não reindexar a cada evento espúrio do OneDrive — deixaria de funcionar em
// silêncio, trocando um defeito por um custo.
func TestAtalhoDoApplyAindaPulaQuandoTudoEstaEmDia(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "nota.md"), []byte("# T\n\ncorpo.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	v, err := vault.New(dir)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	idx := index.New()
	if err := idx.Build(context.Background(), v); err != nil {
		t.Fatalf("idx.Build: %v", err)
	}
	inv := search.NewInverted()
	canon := vault.CanonicalPath("nota.md")
	if err := inv.Update(context.Background(), v, canon); err != nil {
		t.Fatalf("inv.Update: %v", err)
	}

	entrada := make(chan []vault.CanonicalPath, 1)
	reconcile := make(chan struct{})
	var processed, skipped, recUp, recRem atomic.Int64
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go Apply(ctx, entrada, reconcile, idx, v, log, &processed, &skipped, &recUp, &recRem, inv)

	entrada <- []vault.CanonicalPath{canon}

	limite := time.Now().Add(5 * time.Second)
	for time.Now().Before(limite) {
		if skipped.Load() > 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Error("o atalho deixou de pular um evento espurio com tudo em dia; " +
		"a guarda ficou larga demais e o OneDrive passa a reindexar sem motivo")
}
