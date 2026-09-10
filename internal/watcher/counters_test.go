package watcher

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/jonyduque/Gobsidian/internal/index"
	"github.com/jonyduque/Gobsidian/internal/vault"
	"github.com/jonyduque/Gobsidian/internal/vaulttest"
)

// bufferDeLog acumula o que o watcher registrou, para o teste despejar quando
// falhar.
//
// io.Discard estava aqui antes, e foi o que tornou um estouro de prazo
// indiagnosticavel em 2026-08-26: o teste dizia "nunca chegou ao indice" e
// TODOS os avisos do watcher — falha ao adicionar watch, entrada ilegivel,
// varredura interrompida — tinham sido jogados fora. Um teste que descarta o
// log do componente que ele testa so consegue dizer QUE falhou.
//
// Protegido por mutex porque o watcher registra a partir de varias goroutines
// enquanto o teste le.
type bufferDeLog struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *bufferDeLog) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *bufferDeLog) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// diagnostico despeja os contadores do watcher numa falha.
//
// A mensagem antiga imprimia so "eventos=N" e descartava o resto, entao um
// estouro de prazo em 2026-08-26 nao dizia QUAL das etapas parou. Cada campo
// aqui separa uma causa diferente:
//
//	Active=false           o laco de Run saiu; nada mais sera processado
//	EventsReceived=0       o watch nunca viu o rename (registro ou fsnotify)
//	EventsReceived=1       so a criacao do diretorio chegou; a varredura falhou
//	EventsProcessed=0      a varredura emitiu, mas Apply nao indexou
//	DroppedByReason        o filtro comeu os eventos, e diz por que motivo
//	Reconciliations>0      houve overflow do buffer do SO
func diagnostico(w *Watcher) string {
	s := w.Stats()
	return fmt.Sprintf("  contadores: Active=%v Received=%d Dropped=%d(%v) "+
		"Coalesced=%d Processed=%d Skipped=%d Reconciliations=%d",
		s.Active, s.EventsReceived, s.EventsDropped, s.DroppedByReason,
		s.EventsCoalesced, s.EventsProcessed, s.EventsSkipped, s.Reconciliations)
}

func setupTestWatcher(t *testing.T) (*Watcher, context.CancelFunc, string, *index.Index) {
	saida := &bufferDeLog{}
	log := slog.New(slog.NewTextHandler(saida, &slog.HandlerOptions{Level: slog.LevelDebug}))
	t.Cleanup(func() {
		if t.Failed() {
			t.Logf("--- log do watcher ---\n%s--- fim do log ---", saida.String())
		}
	})
	dir := t.TempDir()

	v, err := vault.New(dir)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}

	idx := index.New()

	w, err := New(v, idx, nil, 10*time.Millisecond, log)
	if err != nil {
		t.Fatalf("watcher.New: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() { _ = w.Run(ctx) }()
	// Registrado ANTES da espera, e nao so devolvido ao chamador: se
	// EsperarWatcherAtivo reprovar, ela reprova antes do return, o chamador
	// nunca recebe o cancel para pôr em defer, e a goroutine de Run mais o
	// handle do fsnotify ficam ate o fim do binario de teste — multiplicado
	// por 20 sob `-count=20`. O sleep anterior nao tinha esse caminho porque
	// nao podia falhar. O cancel continua sendo devolvido: chamar duas vezes
	// um CancelFunc e no-op.
	t.Cleanup(cancel)

	EsperarWatcherAtivo(t, w)

	return w, cancel, dir, idx
}

func TestCounters_EventsReceived(t *testing.T) {
	w, cancel, dir, _ := setupTestWatcher(t)
	defer cancel()

	notePath := filepath.Join(dir, "received.md")
	if err := os.WriteFile(notePath, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	if !EsperarAte(vaulttest.Prazo, func() bool { return w.Stats().EventsReceived > 0 }) {
		t.Errorf("EventsReceived = 0 apos %v, want > 0\n%s", vaulttest.Prazo, diagnostico(w))
	}
}

func TestCounters_EventsDropped(t *testing.T) {
	w, cancel, dir, _ := setupTestWatcher(t)
	defer cancel()

	notePath := filepath.Join(dir, "desktop.ini")
	if err := os.WriteFile(notePath, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	if !EsperarAte(vaulttest.Prazo, func() bool { return w.Stats().EventsDropped > 0 }) {
		t.Errorf("EventsDropped = 0 apos %v, want > 0\n%s", vaulttest.Prazo, diagnostico(w))
	}
}

func TestCounters_EventsProcessed(t *testing.T) {
	w, cancel, dir, _ := setupTestWatcher(t)
	defer cancel()

	notePath := filepath.Join(dir, "processed.md")
	if err := os.WriteFile(notePath, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	if !EsperarAte(vaulttest.Prazo, func() bool { return w.Stats().EventsProcessed > 0 }) {
		t.Errorf("EventsProcessed = 0 apos %v, want > 0\n%s", vaulttest.Prazo, diagnostico(w))
	}
}

func TestCounters_EventsSkipped(t *testing.T) {
	w, cancel, dir, _ := setupTestWatcher(t)
	defer cancel()

	notePath := filepath.Join(dir, "skipped.md")
	if err := os.WriteFile(notePath, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	if !EsperarAte(vaulttest.Prazo, func() bool { return w.Stats().EventsProcessed > 0 }) {
		t.Fatalf("EventsProcessed = 0 apos %v, want > 0\n%s", vaulttest.Prazo, diagnostico(w))
	}

	canon, _ := vault.Canonicalize(dir, notePath)
	w.debounced <- []vault.CanonicalPath{canon}

	if !EsperarAte(vaulttest.Prazo, func() bool { return w.Stats().EventsSkipped > 0 }) {
		t.Errorf("EventsSkipped = 0 apos %v, want > 0\n%s", vaulttest.Prazo, diagnostico(w))
	}
}

// TestCounters_DropReasons chama emite, que e o corpo que Run executa para cada
// evento — o mesmo filtro e os mesmos contadores.
//
// A versao anterior escrevia os quatro eventos em `w.fsWatcher.Events` e depois
// esperava em laco o filtro reagir. Escrever num canal interno do fsnotify e a
// corrida que overflow_test.go:138-147 registra; e como emite e sincrona, a
// espera some junto.
func TestCounters_DropReasons(t *testing.T) {
	w, cancel, dir, _ := setupTestWatcher(t)
	defer cancel()

	ctx := context.Background()
	for _, e := range []fsnotify.Event{
		{Name: filepath.Join(dir, "nota.md"), Op: fsnotify.Chmod},        // 1. Chmod
		{Name: "D:\\fora\\nota.md", Op: fsnotify.Write},                  // 2. Outside vault
		{Name: filepath.Join(dir, ".git", "config"), Op: fsnotify.Write}, // 3. Excluded (.git)
		{Name: filepath.Join(dir, "nota.md"), Op: 0},                     // 4. Unknown op
	} {
		if err := w.emite(ctx, e); err != nil {
			t.Fatalf("emite(%+v): %v", e, err)
		}
	}

	st := w.Stats()
	if st.DroppedByReason["chmod"] != 1 {
		t.Errorf("chmod drop count = %d, want 1", st.DroppedByReason["chmod"])
	}
	if st.DroppedByReason["outside_vault"] != 1 {
		t.Errorf("outside_vault drop count = %d, want 1", st.DroppedByReason["outside_vault"])
	}
	if st.DroppedByReason["excluded"] != 1 {
		t.Errorf("excluded drop count = %d, want 1", st.DroppedByReason["excluded"])
	}
	if st.DroppedByReason["unknown_op"] != 1 {
		t.Errorf("unknown_op drop count = %d, want 1", st.DroppedByReason["unknown_op"])
	}
	if st.EventsDropped != 4 {
		t.Errorf("EventsDropped sum = %d, want 4", st.EventsDropped)
	}
}

func TestCounters_ActiveState(t *testing.T) {
	w, cancel, _, _ := setupTestWatcher(t)

	if !w.Stats().Active {
		t.Error("Active = false, want true while running")
	}

	cancel()

	// O sinal observavel do cancelamento e o proprio Active voltando a false —
	// Run o marca no defer, ao sair do laco.
	if !EsperarAte(vaulttest.Prazo, func() bool { return !w.Stats().Active }) {
		t.Errorf("Active = true apos %v do cancel, want false — o laco de Run nao saiu",
			vaulttest.Prazo)
	}
}

func TestCounters_Coalesced(t *testing.T) {
	w, cancel, dir, _ := setupTestWatcher(t)
	defer cancel()

	canon, _ := vault.Canonicalize(dir, filepath.Join(dir, "nota.md"))
	in := make(chan Event, 10)
	out := make(chan []vault.CanonicalPath, 10)

	ctx, cancelDebounce := context.WithCancel(context.Background())
	defer cancelDebounce()

	go Debounce(ctx, in, out, 50*time.Millisecond, slog.Default(), &w.coalesced)

	// Send duplicate events to dirty set
	in <- Event{Path: canon, Op: OpCreate}
	in <- Event{Path: canon, Op: OpWrite}

	if !EsperarAte(vaulttest.Prazo, func() bool { return w.Stats().EventsCoalesced > 0 }) {
		t.Fatalf("EventsCoalesced continuou 0 apos %v: o debouncer nao juntou os "+
			"dois eventos do mesmo caminho", vaulttest.Prazo)
	}
	if got := w.Stats().EventsCoalesced; got != 1 {
		t.Errorf("EventsCoalesced = %d, want 1", got)
	}
}

// setupSoReconciliador monta a pilha com o caminho NORMAL desconectado.
//
// # Por que sem w.Run
//
// A versao anterior deste setup rodava o watcher inteiro, com debounce de
// 10 ms. Apagar o arquivo disparava um evento de verdade, e o caminho normal
// tirava a entrada do indice competindo com o reconciliador: quando ele
// chegava primeiro, Reconcile nao tinha mais o que remover, `rem` vinha 0 e o
// contador nunca subia. O teste passava por quem ganhava a corrida — sob carga
// da suite inteira com -race, o caminho normal ganhava e o teste reprovava com
// "ReconciledRemoved = 0, want 1".
//
// Confirmado por sonda antes de mexer: dando 300 ms de vantagem ao caminho
// normal, reprovava em 3 de 3 execucoes.
//
// Isto e a armadilha que este projeto ja pagou uma vez, em
// TestOverflowReconciliationFull: teste de mecanismo de recuperacao que deixa o
// caminho normal ligado mede o caminho normal. Aqui so o aplicador roda, e a
// entrada de eventos comuns e um canal em que ninguem escreve — o reconciliador
// e o UNICO mecanismo capaz de mexer no indice.
func setupSoReconciliador(t *testing.T) (*Watcher, string) {
	t.Helper()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	dir := t.TempDir()

	v, err := vault.New(dir)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}

	w, err := New(v, index.New(), nil, 10*time.Millisecond, log)
	if err != nil {
		t.Fatalf("watcher.New: %v", err)
	}
	t.Cleanup(func() { _ = w.Close() })

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	// Canal sem escritor nenhum: e ele que garante que nenhum evento de
	// arquivo chegue ao indice por fora da reconciliacao.
	semEventos := make(chan []vault.CanonicalPath)
	go Apply(ctx, semEventos, w.reconcile, w.idx, w.v, w.log,
		&w.processed, &w.skipped, &w.reconciledUpdated, &w.reconciledRemoved, w.inv)

	return w, dir
}

// prazoReconciliacao e quanto se espera o contador subir.
//
// Generoso de proposito. O que o teste mede e que a reconciliacao ACONTECE,
// nao em quanto tempo: sob -race, com a suite inteira disputando a maquina,
// prazo curto mede a carga e nao o mecanismo. Se o mecanismo nao funcionar, o
// teste continua reprovando — so demora mais para dizer isso.
const prazoReconciliacao = 30 * time.Second

// esperaContador repete a leitura ate o contador chegar em `quer` ou o prazo
// acabar, e devolve o ultimo valor visto — e o ultimo valor, e nao um booleano,
// porque a mensagem de falha do chamador o imprime.
func esperaContador(t *testing.T, ler func() int64, quer int64) int64 {
	t.Helper()
	var ultimo int64
	EsperarAte(prazoReconciliacao, func() bool {
		ultimo = ler()
		return ultimo >= quer
	})
	return ultimo
}

func TestCounters_ReconciledUpdatedAndRemoved(t *testing.T) {
	w, dir := setupSoReconciliador(t)
	alvo := filepath.Join(dir, "n1.md")

	if err := os.WriteFile(alvo, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}
	w.reconcile <- struct{}{}
	if got := esperaContador(t, func() int64 { return w.Stats().ReconciledUpdated }, 1); got != 1 {
		t.Fatalf("ReconciledUpdated = %d, want 1", got)
	}

	// Apaga do disco. Sem o caminho normal ligado, so a reconciliacao pode
	// perceber.
	if err := os.Remove(alvo); err != nil {
		t.Fatal(err)
	}
	w.reconcile <- struct{}{}
	if got := esperaContador(t, func() int64 { return w.Stats().ReconciledRemoved }, 1); got != 1 {
		t.Errorf("ReconciledRemoved = %d, want 1", got)
	}

	// Guarda contra a regressao que causou tudo isto: religar o watcher faria
	// EventsReceived subir, e o teste voltaria a medir quem ganha a corrida em
	// vez do reconciliador. w.received so sobe dentro de w.Run.
	if got := w.Stats().EventsReceived; got != 0 {
		t.Errorf("EventsReceived = %d, want 0 — o caminho normal foi religado, "+
			"e este teste voltou a medir uma corrida em vez da reconciliacao", got)
	}
}

// TestPastaQueChegaComArquivosDentro guarda um defeito que tornava notas
// invisiveis para TODAS as tools ate o proximo reinicio.
//
// Um diretorio que chega ao cofre ja com arquivos dentro entrega exatamente UM
// evento — a criacao do proprio diretorio. Os arquivos ja existiam quando o
// watch foi registrado, entao nunca geraram evento nenhum. Medido antes da
// correcao: "eventos recebidos=1, notas no indice=0", sem erro e sem log.
//
// Nao e caso de borda: e o usuario arrastando uma pasta para dentro do cofre.
// E e a mesma falha que fazia note_move perder a nota, porque a tool cria o
// diretorio de destino e renomeia para dentro dele em seguida.
//
// O teste move uma pasta PRONTA para dentro do cofre em vez de criar a pasta e
// depois os arquivos: assim nao ha corrida a ganhar: os arquivos existem antes
// de qualquer watch, sempre.
func TestPastaQueChegaComArquivosDentro(t *testing.T) {
	w, cancel, dir, idx := setupTestWatcher(t)
	defer cancel()

	fora := t.TempDir()
	pronta := filepath.Join(fora, "chegou")
	if err := os.MkdirAll(filepath.Join(pronta, "sub"), 0755); err != nil {
		t.Fatal(err)
	}
	esperados := map[string]string{
		"chegou/a.md":     filepath.Join(pronta, "a.md"),
		"chegou/b.md":     filepath.Join(pronta, "b.md"),
		"chegou/sub/c.md": filepath.Join(pronta, "sub", "c.md"),
	}
	for _, abs := range esperados {
		if err := os.WriteFile(abs, []byte("# X\n\nconteudo.\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Rename(pronta, filepath.Join(dir, "chegou")); err != nil {
		t.Skipf("rename de diretorio nao suportado neste ambiente: %v", err)
	}

	// O resultado da espera nao decide nada: quem reprova, e com o nome da nota
	// que faltou e os contadores, e o laco abaixo. Estourar o prazo aqui so
	// significa "vai reprovar com uma mensagem melhor daqui a uma linha".
	_ = EsperarAte(prazoReconciliacao, func() bool { return idx.NoteCount() >= len(esperados) })

	for rel := range esperados {
		if _, ok := idx.Get(vault.CanonicalPath(rel)); !ok {
			var vistos []string
			for _, p := range idx.NotePaths() {
				vistos = append(vistos, string(p))
			}
			t.Errorf("%s nunca chegou ao indice; indice=%v\n%s",
				rel, vistos, diagnostico(w))
		}
	}

	// A varredura tem de deixar os SUBDIRETORIOS vigiados, e nao so indexar o
	// que achou neles. Sem isto a pasta entra certa e para de acompanhar: uma
	// nota criada depois, dentro de chegou/sub/, nunca apareceria — e nenhuma
	// asserção acima notaria, porque todas falam do estado inicial.
	//
	// Esta metade foi escrita depois: a prova de mutacao mostrou que remover o
	// Add do subdiretorio deixava o teste verde.
	depois := filepath.Join(dir, "chegou", "sub", "d.md")
	if err := os.WriteFile(depois, []byte("# D\n\nnota criada depois.\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if !EsperarAte(prazoReconciliacao, func() bool {
		_, ok := idx.Get(vault.CanonicalPath("chegou/sub/d.md"))
		return ok
	}) {
		t.Errorf("chegou/sub/d.md nao foi indexada: o subdiretorio varrido ficou sem watch\n%s",
			diagnostico(w))
	}
}
