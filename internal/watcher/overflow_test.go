package watcher

import (
	"bytes"
	"context"
	"errors"
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
	"github.com/jonyd/gobsidian/internal/search"
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
	updated, removedN, skipped := Reconcile(context.Background(), v, idx, invDoCofre(t, v, idx), log)

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
	go Apply(ctx, in, reconcile, idx, v, log, nil, nil, nil, nil, nil)

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

	w, err := New(v, idx, nil, 10*time.Millisecond, log)
	if err != nil {
		t.Fatalf("New watcher: %v", err)
	}
	defer func() { _ = w.Close() }()

	// Chama handleFSError, que é o corpo que Run executa. A versão anterior
	// deste teste injetava o erro em `w.fsWatcher.Errors` e mantinha, dentro do
	// próprio teste, uma CÓPIA do laço de tratamento — contava
	// `reconciliations` e enfileirava em `reconcile` por conta própria. Ele
	// afirmava sobre a reimplementação: apagar o tratamento de overflow da
	// produção o deixava verde. Escrever no canal do fsnotify ainda corria com
	// o backend kqueue, e o detector reprovava em macOS com DATA RACE.
	//
	// Apply não é lançado de propósito: é ele que consome `reconcile`, e o que
	// este teste mede é quantos tokens ficam no canal.
	for range 5 {
		w.handleFSError(fsnotify.ErrEventOverflow)
	}

	if count := w.reconciliations.Load(); count != 5 {
		t.Fatalf("reconciliations counter quer 5, obteve %d", count)
	}

	// Erro que não é overflow não agenda nada. Sem esta asserção, um
	// tratamento que reconciliasse a QUALQUER erro do fsnotify passaria.
	w.handleFSError(errors.New("erro qualquer do fsnotify"))
	if count := w.reconciliations.Load(); count != 5 {
		t.Fatalf("erro comum mexeu no contador: quer 5, obteve %d", count)
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
	updated, removed, skipped := Reconcile(context.Background(), v, idx, invDoCofre(t, v, idx), log)

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
	updated, removed, skipped := Reconcile(ctx, v, idx, invDoCofre(t, v, idx), log)

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

	Reconcile(ctx, v, idx, invDoCofre(t, v, idx), log)

	output := buf.String()
	if strings.Contains(output, "LEVEL=ERROR") || strings.Contains(output, "level=ERROR") || strings.Contains(output, "Erro durante varredura") {
		t.Fatalf("Reconcile cancelado emitiu log de ERROR: %s", output)
	}
	if !strings.Contains(output, "Reconciliação interrompida pelo shutdown") {
		t.Errorf("Reconcile cancelado nao emitiu log Debug de shutdown: %s", output)
	}
}

// invDoCofre monta um indice de busca coerente com o indice de metadados no
// momento da chamada.
//
// Os testes de reconciliacao passavam so o indice de metadados, e por isso nao
// diziam nada sobre a metade que estava quebrada: a reconciliacao reparava os
// metadados e deixava a busca obsoleta para sempre. Passar um indice de busca
// de verdade faz esses testes exercitarem o caminho inteiro.
func invDoCofre(t *testing.T, v *vault.Vault, idx *index.Index) *search.Inverted {
	t.Helper()
	inv := search.NewInverted()
	for _, p := range idx.NotePaths() {
		b, err := v.ReadAll(context.Background(), p)
		if err != nil {
			continue
		}
		corpo, _ := vault.StripBOM(b)
		inv.Add(string(p), search.Analyze(string(corpo)))
	}
	return inv
}

// TestReconcileReparaOIndiceDeBusca guarda o defeito mais caro que a
// reconciliacao ja teve, e que nenhum teste via.
//
// Ela reparava o indice de METADADOS e deixava o de BUSCA obsoleto. Como
// service.Search descarta a posting cujo caminho nao existe nos metadados
// (`s.index.Get` -> `if !ok { continue }`), uma nota MOVIDA durante o overflow
// ficava com os metadados no caminho novo e a posting no antigo: `vault_search`
// devolvia ZERO resultados para uma nota que estava la, para sempre, sem log e
// sem erro. Uma nota CRIADA durante o overflow ficava invisivel pelo mesmo
// motivo, ao contrario.
//
// O teste afirma sobre o que a BUSCA veria, e nao sobre os dois indices em
// separado: e a combinacao dos dois que decide a resposta, e conferir cada um
// isoladamente foi o que deixou isso passar.
func TestReconcileReparaOIndiceDeBusca(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	root := t.TempDir()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(os.MkdirAll(filepath.Join(root, "origem"), 0755))
	must(os.MkdirAll(filepath.Join(root, "destino"), 0755))
	must(os.WriteFile(filepath.Join(root, "origem", "alvo.md"),
		[]byte("# Alvo\n\nzarabatana e a palavra sonda.\n"), 0644))

	v, err := vault.New(root)
	must(err)
	idx := index.New()
	must(idx.Build(context.Background(), v))
	inv := invDoCofre(t, v, idx)

	// visiveis conta o que vault_search devolveria: posting que o indice de
	// metadados confirma.
	visiveis := func(termo string) []string {
		var out []string
		for _, p := range inv.Postings(termo) {
			if _, ok := idx.Get(vault.CanonicalPath(p.Path)); ok {
				out = append(out, p.Path)
			}
		}
		return out
	}

	if got := visiveis("zarabatana"); len(got) != 1 || got[0] != "origem/alvo.md" {
		t.Fatalf("estado inicial errado: %v — sem isto o resto passaria vacuamente", got)
	}

	// O cofre muda com NINGUEM escutando: e o que "eventos perdidos"
	// significa. So a reconciliacao pode consertar.
	must(os.Rename(filepath.Join(root, "origem", "alvo.md"),
		filepath.Join(root, "destino", "renomeada.md")))
	must(os.WriteFile(filepath.Join(root, "destino", "nova.md"),
		[]byte("# Nova\n\nbodoque nasceu durante o overflow.\n"), 0644))

	Reconcile(context.Background(), v, idx, inv, log)

	if got := visiveis("zarabatana"); len(got) != 1 || got[0] != "destino/renomeada.md" {
		t.Errorf("apos a reconciliacao a busca devolveria %v para a nota movida, quer [destino/renomeada.md]", got)
	}
	if got := visiveis("bodoque"); len(got) != 1 || got[0] != "destino/nova.md" {
		t.Errorf("apos a reconciliacao a busca devolveria %v para a nota criada, quer [destino/nova.md]", got)
	}
	if inv.HasDoc("origem/alvo.md") {
		t.Error("o caminho antigo sobreviveu no indice de busca")
	}
}

// TestReconcileNaoPulaNotaAusenteDaBusca guarda o atalho de mtime/tamanho.
//
// Ele fala do indice de METADADOS. Quando a nota esta la e falta na busca —
// que e exatamente a divergencia que esta funcao existe para consertar — pular
// por "ja esta atualizada" fazia o anteparo confirmar o defeito.
func TestReconcileNaoPulaNotaAusenteDaBusca(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "n.md"),
		[]byte("# N\n\nzarabatana.\n"), 0644); err != nil {
		t.Fatal(err)
	}
	v, err := vault.New(root)
	if err != nil {
		t.Fatal(err)
	}
	idx := index.New()
	if err := idx.Build(context.Background(), v); err != nil {
		t.Fatal(err)
	}

	// Metadados em dia, busca vazia: nada mudou no disco, entao mtime e
	// tamanho batem e o atalho dispara.
	inv := search.NewInverted()

	Reconcile(context.Background(), v, idx, inv, log)

	if len(inv.Postings("zarabatana")) != 1 {
		t.Errorf("a nota continua fora do indice de busca apos a reconciliacao: %d postings",
			len(inv.Postings("zarabatana")))
	}
}
