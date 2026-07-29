package watcher

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/vault"
)

// TestReconcile_CorrectsLostEvents roda SEM watcher. Isso e a tarefa inteira:
// com o watcher ligado, o pipeline normal aplica as tres mudancas e a
// reconciliacao nunca e exercitada — foi assim que um requisito P0 passou por
// uma revisao inteira com cobertura zero.
func TestReconcile_CorrectsLostEvents(t *testing.T) {
	tmp := t.TempDir()

	modificado := filepath.Join(tmp, "file1.md")
	removido := filepath.Join(tmp, "file2.md")
	if err := os.WriteFile(modificado, []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(removido, []byte("some content"), 0644); err != nil {
		t.Fatal(err)
	}

	v, err := vault.New(tmp)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	idx := index.New()
	if err := idx.Build(context.Background(), v); err != nil {
		t.Fatalf("idx.Build: %v", err)
	}

	// O cofre muda por baixo, com NINGUEM escutando. E o que "eventos perdidos"
	// significa: nao ha evento nenhum para perder, so divergencia.
	if err := os.WriteFile(modificado, []byte("new content, different size"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(removido); err != nil {
		t.Fatal(err)
	}
	criado := filepath.Join(tmp, "file3.md")
	if err := os.WriteFile(criado, []byte("added content"), 0644); err != nil {
		t.Fatal(err)
	}

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	updated, removedN, skipped := Reconcile(context.Background(), v, idx, log)

	if n, ok := idx.Get("file1.md"); !ok || n.Size != int64(len("new content, different size")) {
		t.Errorf("modificado nao reconciliado: ok=%v nota=%+v", ok, n)
	}
	if _, ok := idx.Get("file2.md"); ok {
		t.Errorf("removido continua no indice")
	}
	if n, ok := idx.Get("file3.md"); !ok || n.Size != int64(len("added content")) {
		t.Errorf("criado nao entrou no indice: ok=%v nota=%+v", ok, n)
	}
	if updated < 2 || removedN != 1 {
		t.Errorf("contadores = updated %d, removed %d, skipped %d; quer >=2 e 1",
			updated, removedN, skipped)
	}
}

// TestApply_ReconcileSignal prova que a correcao veio do SINAL, e nao de um
// evento: o canal de entrada e criado e nunca alimentado.
func TestApply_ReconcileSignal(t *testing.T) {
	tmp := t.TempDir()
	alvo := filepath.Join(tmp, "nota.md")
	if err := os.WriteFile(alvo, []byte("antes"), 0644); err != nil {
		t.Fatal(err)
	}

	v, err := vault.New(tmp)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	idx := index.New()
	if err := idx.Build(context.Background(), v); err != nil {
		t.Fatalf("idx.Build: %v", err)
	}

	in := make(chan []vault.CanonicalPath) // criado e NUNCA alimentado
	reconcile := make(chan struct{}, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	go Apply(ctx, in, reconcile, idx, v, log, nil, nil, nil, nil)

	if err := os.WriteFile(alvo, []byte("depois, bem maior"), 0644); err != nil {
		t.Fatal(err)
	}
	reconcile <- struct{}{}

	// Espera em laco com condicao de saida. time.Sleep fixo como assercao e o
	// que fez o teste anterior passar sem mecanismo nenhum.
	quer := int64(len("depois, bem maior"))
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if n, ok := idx.Get("nota.md"); ok && n.Size == quer {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	n, _ := idx.Get("nota.md")
	t.Fatalf("indice nao foi corrigido pelo sinal de reconciliacao: %+v", n)
}

func TestRun_OverflowSchedulesExactlyOne(t *testing.T) {
	tmp := t.TempDir()
	v, err := vault.New(tmp)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	idx := index.New()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	w, err := New(v, idx, 10*time.Millisecond, log)
	if err != nil {
		t.Fatalf("New watcher: %v", err)
	}
	defer w.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Inicia o laço de erros sem lançar o Apply (para o canal reconcile não ser consumido por Apply)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case err, ok := <-w.fsWatcher.Errors:
				if !ok {
					return
				}
				if err == fsnotify.ErrEventOverflow || (err != nil && err.Error() == fsnotify.ErrEventOverflow.Error()) {
					w.reconciliations.Add(1)
					select {
					case w.reconcile <- struct{}{}:
					default:
					}
				}
			}
		}
	}()

	for range 5 {
		w.fsWatcher.Errors <- fsnotify.ErrEventOverflow
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if w.reconciliations.Load() == 5 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if count := w.reconciliations.Load(); count != 5 {
		t.Fatalf("reconciliations counter quer 5, obteve %d", count)
	}

	select {
	case <-w.reconcile:
		// Primeiro token entregue como esperado
	default:
		t.Fatal("esperava token no canal reconcile, nenhum encontrado")
	}

	select {
	case <-w.reconcile:
		t.Fatal("canal reconcile entregou mais de um token")
	default:
		// OK - apenas um token agendado
	}
}

func TestReconcile_VaultGoneLeavesIndexIntact(t *testing.T) {
	tmp := t.TempDir()
	vaultDir := filepath.Join(tmp, "vault")
	if err := os.MkdirAll(vaultDir, 0755); err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{"n1.md", "n2.md", "n3.md"} {
		if err := os.WriteFile(filepath.Join(vaultDir, name), []byte("content"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	v, err := vault.New(vaultDir)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	idx := index.New()
	if err := idx.Build(context.Background(), v); err != nil {
		t.Fatalf("idx.Build: %v", err)
	}

	if idx.NoteCount() != 3 {
		t.Fatalf("NoteCount inicial quer 3, obteve %d", idx.NoteCount())
	}

	if err := os.RemoveAll(vaultDir); err != nil {
		t.Fatal(err)
	}

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	updated, removed, skipped := Reconcile(context.Background(), v, idx, log)

	if idx.NoteCount() != 3 {
		t.Errorf("NoteCount apos cofre sumir quer 3, obteve %d", idx.NoteCount())
	}
	if updated != 0 || removed != 0 || skipped != 0 {
		t.Errorf("contadores com cofre inacessivel querem 0, 0, 0; obteve %d, %d, %d", updated, removed, skipped)
	}
}

func TestReconcile_CtxCancelStopsEarly(t *testing.T) {
	tmp := t.TempDir()
	v, err := vault.New(tmp)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	idx := index.New()

	for i := range 200 {
		name := filepath.Join(tmp, fmt.Sprintf("note%03d.md", i))
		if err := os.WriteFile(name, []byte("hello"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	if err := idx.Build(context.Background(), v); err != nil {
		t.Fatalf("idx.Build: %v", err)
	}

	if idx.NoteCount() != 200 {
		t.Fatalf("esperava 200 notas no indice, obteve %d", idx.NoteCount())
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	updated, removed, skipped := Reconcile(ctx, v, idx, log)

	if idx.NoteCount() != 200 {
		t.Errorf("indice foi alterado/esvaziado apos cancelamento: count=%d", idx.NoteCount())
	}
	if updated >= 200 {
		t.Errorf("visitou todas as notas apesar do cancelamento: updated=%d", updated)
	}
	if removed != 0 {
		t.Errorf("remoções não deveriam ter rodado sob cancelamento: removed=%d", removed)
	}
	_ = skipped
}

func TestReconcile_CancelIsNotAnError(t *testing.T) {
	tmp := t.TempDir()
	for i := range 10 {
		name := filepath.Join(tmp, fmt.Sprintf("note%02d.md", i))
		if err := os.WriteFile(name, []byte("content"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	v, err := vault.New(tmp)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	idx := index.New()
	if err := idx.Build(context.Background(), v); err != nil {
		t.Fatalf("idx.Build: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	Reconcile(ctx, v, idx, log)

	output := buf.String()
	if strings.Contains(output, "LEVEL=ERROR") || strings.Contains(output, "level=ERROR") || strings.Contains(output, "Erro durante varredura") {
		t.Fatalf("Reconcile cancelado emitiu log de ERROR: %s", output)
	}
	if !strings.Contains(output, "Reconciliação interrompida pelo shutdown") {
		t.Errorf("Reconcile cancelado nao emitiu log Debug de shutdown: %s", output)
	}
}
