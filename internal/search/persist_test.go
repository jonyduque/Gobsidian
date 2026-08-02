package search_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/search"
	"github.com/jonyd/gobsidian/internal/vault"
)

func TestSaveAndLoadInvertedCache(t *testing.T) {
	cacheDir := t.TempDir()
	vaultPath := t.TempDir()

	inv := search.NewInverted()
	inv.Add("a.md", search.Analyze("# Prescrição\n\nTexto sobre prescrição.\n"))
	inv.Add("b.md", search.Analyze("# Outra\n\nTexto sem o termo.\n"))

	ctx := context.Background()
	if err := search.SaveInvertedCache(ctx, cacheDir, vaultPath, inv); err != nil {
		t.Fatalf("SaveInvertedCache: %v", err)
	}

	loaded, header, err := search.LoadInvertedCache(ctx, cacheDir, vaultPath)
	if err != nil {
		t.Fatalf("LoadInvertedCache: %v", err)
	}
	if header.NoteCount != 2 {
		t.Errorf("header.NoteCount = %d, want 2", header.NoteCount)
	}

	postings := loaded.Postings("prescricao")
	if len(postings) != 1 || postings[0].Path != "a.md" {
		t.Errorf("Get(prescricao) = %+v, want a.md", postings)
	}
}

// Prova de mutação: desligar a checagem de analyzer_version faz este teste reprovar.
func TestCacheAnalyzerVersionMismatchDiscardsCache(t *testing.T) {
	cacheDir := t.TempDir()
	vaultPath := t.TempDir()

	inv := search.NewInverted()
	inv.Add("a.md", search.Analyze("termo"))

	ctx := context.Background()
	if err := search.SaveInvertedCache(ctx, cacheDir, vaultPath, inv); err != nil {
		t.Fatalf("SaveInvertedCache: %v", err)
	}

	cachePath := filepath.Join(cacheDir, "inverted_cache.gob")
	data, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	badData := search.InvertedCacheData{
		Header: search.CacheHeader{
			FormatVersion:   search.CacheFormatVersion,
			ParserVersion:   search.CacheParserVersion,
			AnalyzerVersion: 999, // Versão mutada
			VaultPath:       vaultPath,
			NoteCount:       1,
		},
		Terms: inv.ExportTermsForCache(),
	}

	f, err := os.Create(cachePath)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	enc := search.NewEncoderForTest(f)
	if err := enc.Encode(badData); err != nil {
		_ = f.Close()
		t.Fatalf("Encode: %v", err)
	}
	_ = f.Close()

	_, _, err = search.LoadInvertedCache(ctx, cacheDir, vaultPath)
	if err == nil {
		t.Fatal("LoadInvertedCache deveria falhar com versão de analisador diferente (999)")
	}
	if !errors.Is(err, search.ErrCacheVersionMismatch) {
		t.Fatalf("err = %v, want ErrCacheVersionMismatch", err)
	}

	_ = data
}

func TestTruncatedCacheRefused(t *testing.T) {
	cacheDir := t.TempDir()
	vaultPath := t.TempDir()

	cachePath := filepath.Join(cacheDir, "inverted_cache.gob")
	if err := os.WriteFile(cachePath, []byte("tru"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, _, err := search.LoadInvertedCache(context.Background(), cacheDir, vaultPath)
	if err == nil {
		t.Fatal("LoadInvertedCache deveria falhar com arquivo truncado")
	}
	if !errors.Is(err, search.ErrCacheCorrupted) {
		t.Fatalf("err = %v, want ErrCacheCorrupted", err)
	}
}

func TestEmptyVaultCacheDistinguishableFromMissing(t *testing.T) {
	cacheDir := t.TempDir()
	vaultPath := t.TempDir()

	// 1. Cache inexistente
	_, _, err := search.LoadInvertedCache(context.Background(), cacheDir, vaultPath)
	if !errors.Is(err, search.ErrCacheNotFound) {
		t.Fatalf("Cache inexistente = %v, want ErrCacheNotFound", err)
	}

	// 2. Cache de cofre vazio (0 notas)
	inv := search.NewInverted()
	if err := search.SaveInvertedCache(context.Background(), cacheDir, vaultPath, inv); err != nil {
		t.Fatalf("SaveInvertedCache: %v", err)
	}

	loaded, header, err := search.LoadInvertedCache(context.Background(), cacheDir, vaultPath)
	if err != nil {
		t.Fatalf("Cache de cofre vazio falhou: %v", err)
	}
	if header.NoteCount != 0 {
		t.Errorf("header.NoteCount = %d, want 0", header.NoteCount)
	}
	if loaded.DocCount() != 0 {
		t.Errorf("loaded.DocCount = %d, want 0", loaded.DocCount())
	}
}

func TestCacheOutsideVault(t *testing.T) {
	vaultPath := t.TempDir()
	cacheDir := t.TempDir()

	cClean := filepath.Clean(cacheDir)
	vClean := filepath.Clean(vaultPath)
	if strings.HasPrefix(cClean, vClean) {
		t.Fatalf("cacheDir %q está dentro de vaultPath %q", cacheDir, vaultPath)
	}
}

// geraCorpus cria N notas com caminhos DISTINTOS e conteúdo distinto.
func geraCorpus(t *testing.T, n int) (*vault.Vault, *index.Index, *search.Inverted, string) {
	t.Helper()
	root := t.TempDir()
	for i := 0; i < n; i++ {
		dir := filepath.Join(root, fmt.Sprintf("pasta%02d", i%10))
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		corpo := fmt.Sprintf("---\ntags: [t%d]\n---\n\n# Nota %d\n\nprescricao intercorrente termo%d civil direito processo\n", i%7, i, i)
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("nota%04d.md", i)), []byte(corpo), 0644); err != nil {
			t.Fatal(err)
		}
	}
	v, err := vault.New(root)
	if err != nil {
		t.Fatal(err)
	}
	idx := index.New()
	if err := idx.Build(context.Background(), v); err != nil {
		t.Fatal(err)
	}
	inv := search.NewInverted()
	for _, p := range idx.NotePaths() {
		data, err := v.ReadAll(context.Background(), p)
		if err != nil {
			t.Fatalf("corpus ilegivel em %s: %v", p, err)
		}
		body, _ := vault.StripBOM(data)
		inv.Add(string(p), search.Analyze(string(body)))
	}

	if got := idx.NoteCount(); got != n {
		t.Fatalf("corpus tem %d notas, quer %d", got, n)
	}
	return v, idx, inv, root
}

func TestQ3PerformanceMeasurement(t *testing.T) {
	const n = 500
	v, idx, inv, _ := geraCorpus(t, n)
	cacheDir := t.TempDir()
	vaultPath := v.Root()
	ctx := context.Background()

	if err := search.SaveInvertedCache(ctx, cacheDir, vaultPath, inv); err != nil {
		t.Fatalf("SaveInvertedCache: %v", err)
	}

	startLoad := time.Now()
	loadedInv, header, err := search.LoadInvertedCache(ctx, cacheDir, vaultPath)
	if err != nil {
		t.Fatalf("LoadInvertedCache: %v", err)
	}
	loadDur := time.Since(startLoad)

	if header.NoteCount != n {
		t.Fatalf("header.NoteCount = %d, quer %d", header.NoteCount, n)
	}
	if loadedInv.DocCount() != n {
		t.Fatalf("loadedInv.DocCount() = %d, quer %d", loadedInv.DocCount(), n)
	}

	startRebuild := time.Now()
	rebuiltInv := search.NewInverted()
	for _, p := range idx.NotePaths() {
		data, err := v.ReadAll(ctx, p)
		if err != nil {
			t.Fatalf("ReadAll: %v", err)
		}
		body, _ := vault.StripBOM(data)
		rebuiltInv.Add(string(p), search.Analyze(string(body)))
	}
	rebuildDur := time.Since(startRebuild)

	if rebuiltInv.DocCount() != n {
		t.Fatalf("rebuiltInv.DocCount() = %d, quer %d", rebuiltInv.DocCount(), n)
	}

	t.Logf("Q3 Medição em %d notas distintas:", n)
	t.Logf("  (a) LoadInvertedCache (disco): %v", loadDur)
	t.Logf("  (b) Reconstruir Invertido (metadados): %v", rebuildDur)
}

// TestBM25KernelLatency guarda o NÚCLEO do ranqueamento contra regressão
// algorítmica. Não é a medição do RNF-04, e este teste chamava-se
// TestRNF04SearchLatencyPercentile enquanto era o oposto disso:
//
//   - Media CalculateBM25 direto, deixando de fora parsing da consulta,
//     filtros, limit/offset e a leitura de disco de cada trecho — que é a
//     parte cara. O RNF-04 fala de `vault_search`, e `vault_search` é a
//     pilha inteira.
//   - Consultava `termo%d`, que existe em 5 das 500 notas. Com 5 postings o
//     BM25 devolve antes de o relógio andar: a saída era "Mínimo: 0s,
//     Mediana: 0s, p95: 1,02 ms" contra um teto de 100 ms — 98x de folga,
//     mediana abaixo da resolução do relógio. Não podia falhar.
//   - E era a única das duas asserções que o gate rodava, porque a medição
//     de verdade fica em internal/service e é pulada sob -race, que é o
//     único modo em que verify.ps1 roda os testes. O gate cobrava a
//     tautologia e pulava a medição.
//
// A medição do RNF-04 é TestRNF04VaultSearchLatencyP95, em
// internal/service — por formato de consulta, através de service.Search.
// Ela agora é cobrada pelo gate na etapa "go test (medições de latência)".
//
// Aqui a consulta é "prescricao", que o corpus põe em TODAS as 500 notas: o
// kernel percorre 500 postings, e o número mede alguma coisa: a mediana sai
// de 0s para 4,1 ms.
//
// O teto vem da medição, não do RNF: 80 ms contra um p95 de 8,1 ms medido em
// 2026-08-01 no maquina de referencia (12 núcleos, Windows 11), sem -race. Folga larga de
// propósito — o defeito que este teste existe para pegar é ordem de grandeza,
// não porcento: a varredura linear que a Task 61 pagou custava 174 ms.
//
// O teto NÃO é cobrado sob -race, e a primeira versão deste teste errou nisso.
// Ela dimensionava 80 ms como "4,5x de folga sobre os 17,9 ms medidos com
// -race" na máquina de desenvolvimento. No runner compartilhado do CI, com
// -race, o mesmo teste mediu mediana de 26,6 ms e p95 de 107,1 ms e reprovou
// sem regressão nenhuma — o número tinha vindo da máquina errada. A medição
// continua sendo registrada nos dois modos; só o teto deixa de valer onde o
// número não é comparável. Quem o cobra é a etapa de verify.ps1 que roda sem
// -race.
func TestBM25KernelLatency(t *testing.T) {
	const corpusSize = 500
	const numQueries = 200
	const teto = 80 * time.Millisecond

	_, idx, inv, _ := geraCorpus(t, corpusSize)

	// "prescricao" está em todas as notas de geraCorpus. Confirmado antes de
	// medir: se o termo sumir do corpus, esta medição volta a ser sobre um
	// dicionário dando miss, e ninguém percebe pelo verde.
	tokens := search.Analyze("prescricao")
	if got := len(search.CalculateBM25(tokens, inv, idx)); got != corpusSize {
		t.Fatalf("a consulta casou %d notas de %d; com poucas postings a medição "+
			"não mede o kernel", got, corpusSize)
	}

	durations := make([]time.Duration, numQueries)
	for i := 0; i < numQueries; i++ {
		start := time.Now()
		_ = search.CalculateBM25(tokens, inv, idx)
		durations[i] = time.Since(start)
	}

	sort.Slice(durations, func(i, j int) bool {
		return durations[i] < durations[j]
	})

	minDur := durations[0]
	medianDur := durations[numQueries/2]
	p95Dur := durations[int(float64(numQueries)*0.95)]

	t.Logf("Kernel BM25 (%d consultas de %d postings em %d notas):", numQueries, corpusSize, corpusSize)
	t.Logf("  Mínimo:  %v", minDur)
	t.Logf("  Mediana: %v", medianDur)
	t.Logf("  p95:     %v  (teto deste teste: %v)", p95Dur, teto)

	// Mediana em zero significa que o relógio não andou, e um percentil sobre
	// zeros não guarda nada. Vale como falha por si só.
	if medianDur == 0 {
		t.Errorf("mediana = 0s: a consulta é seletiva demais para medir o kernel")
	}
	if !raceEnabled && p95Dur > teto {
		t.Errorf("p95 %v excede o teto de %v deste teste (regressão algorítmica no "+
			"kernel; o RNF-04 é medido em TestRNF04VaultSearchLatencyP95)", p95Dur, teto)
	}
}
