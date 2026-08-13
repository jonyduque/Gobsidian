package search_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
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
	// Task 89: LoadInvertedCache pode ter mapeado o arquivo em vez de
	// decodificá-lo para o heap. Sem fechar, o Windows recusa apagar o
	// arquivo mapeado quando t.TempDir() limpa o diretório no fim do teste.
	t.Cleanup(func() { _ = loaded.Close() })
	if header.NoteCount != 2 {
		t.Errorf("header.NoteCount = %d, want 2", header.NoteCount)
	}

	postings := loaded.Postings("prescricao")
	if len(postings) != 1 || postings[0].Path != "a.md" {
		t.Errorf("Get(prescricao) = %+v, want a.md", postings)
	}
}

// TestSaveOverwritesMappedCache exercita o caminho que a Task 89 identificou
// como o único em que o rename atômico de SaveInvertedCache pode colidir com
// um mapeamento deste MESMO processo: cache PARCIAL retomado — carregado via
// arena mapeada — e depois regravado por quem continua a construção
// (buildInvertedIndex, em cmd/gobsidian/serve.go).
//
// O os.Rename do salvamento atômico falha no Windows
// (ERROR_SHARING_VIOLATION) se o arquivo de destino ainda está mapeado neste
// processo. Antes de confiar que o SaveInvertedCache abaixo tem de dar certo,
// este teste primeiro CONFIRMA o problema que promoverArenaSePresente existe
// para evitar: no Windows, remover o arquivo enquanto `loaded` ainda o tem
// mapeado falha. Em Unix isso não é um problema (unlink de arquivo mapeado é
// uma operação válida), então essa confirmação só roda sob GOOS==windows —
// runtime.GOOS em teste, para pular caso, é aceitável (ver CLAUDE.md).
func TestSaveOverwritesMappedCache(t *testing.T) {
	cacheDir := t.TempDir()
	vaultPath := t.TempDir()
	ctx := context.Background()

	partida := search.NewInverted()
	partida.Add("a.md", search.Analyze("prescricao intercorrente civil"))
	if err := search.SaveInvertedCache(ctx, cacheDir, vaultPath, partida); err != nil {
		t.Fatalf("SaveInvertedCache (partida): %v", err)
	}

	loaded, _, err := search.LoadInvertedCache(ctx, cacheDir, vaultPath)
	if err != nil {
		t.Fatalf("LoadInvertedCache: %v", err)
	}
	// Defensivo: se SaveInvertedCache falhar antes de promover a arena, esta
	// chamada garante que o t.TempDir() do fim do teste ainda consegue
	// limpar o diretorio. Depois de uma regravacao bem-sucedida,
	// promoverArenaSePresente ja fechou a arena e isto vira no-op.
	t.Cleanup(func() { _ = loaded.Close() })

	cachePath := filepath.Join(cacheDir, "inverted_cache.gob")

	if runtime.GOOS == "windows" {
		// Confirma o problema ANTES de confirmar a solução: com o arquivo
		// ainda mapeado por `loaded`, o Windows recusa removê-lo. Se esta
		// asserção um dia parar de disparar, TestSaveOverwritesMappedCache
		// deixou de testar o que diz testar — não há mais nada para
		// promoverArenaSePresente proteger.
		if err := os.Remove(cachePath); err == nil {
			t.Fatal("os.Remove teve sucesso com o cache ainda mapeado; " +
				"o cenario que este teste existe para cobrir nao ocorreu")
		}
	}

	// Simula o resto de buildInvertedIndex retomando a partir do cache
	// parcial: escreve no DELTA do mesmo índice que carregou via arena.
	loaded.Add("b.md", search.Analyze("recurso extraordinario processo civil"))

	// A regravação por cima do arquivo mapeado é o que a Task 89 precisa
	// funcionar sem erro — sem promoverArenaSePresente, este Rename falharia
	// no Windows com o cenário confirmado acima.
	if err := search.SaveInvertedCache(ctx, cacheDir, vaultPath, loaded); err != nil {
		t.Fatalf("SaveInvertedCache por cima do cache mapeado: %v", err)
	}

	reloaded, hdr, err := search.LoadInvertedCache(ctx, cacheDir, vaultPath)
	if err != nil {
		t.Fatalf("LoadInvertedCache apos a regravacao: %v", err)
	}
	t.Cleanup(func() { _ = reloaded.Close() })

	if hdr.NoteCount != 2 {
		t.Errorf("hdr.NoteCount = %d, quer 2 (a.md do cache original + b.md do delta)", hdr.NoteCount)
	}
	if !reloaded.HasDoc("a.md") {
		t.Error("a.md (do cache original, promovido) sumiu apos a regravacao")
	}
	if !reloaded.HasDoc("b.md") {
		t.Error("b.md (do delta escrito depois da carga) sumiu apos a regravacao")
	}
	if postings := reloaded.Postings("civil"); len(postings) != 2 {
		t.Errorf("Postings(civil) = %d notas, quer 2 (civil aparece em a.md e b.md)", len(postings))
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

	badHeader := search.CacheHeader{
		FormatVersion:   search.CacheFormatVersion,
		ParserVersion:   search.CacheParserVersion,
		AnalyzerVersion: 999, // Versão mutada
		VaultPath:       vaultPath,
		NoteCount:       1,
	}

	termos, docLengths := inv.ExportForCache()

	f, err := os.Create(cachePath)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	// Escrito no formato CORRENTE de propósito: o arquivo precisa passar da
	// mágica e do cabeçalho para que a comparação de analyzer_version chegue a
	// acontecer. Com um arquivo de formato antigo, a recusa viria da assinatura
	// e o teste passaria sem exercitar a regra que ele diz cobrir.
	if err := search.WriteCacheForTest(f, badHeader, termos, docLengths); err != nil {
		_ = f.Close()
		t.Fatalf("WriteCacheForTest: %v", err)
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
	t.Cleanup(func() { _ = loaded.Close() })
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
	t.Cleanup(func() { _ = loadedInv.Close() })
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
// 2026-08-01 na maquina de referencia (12 núcleos, Windows 11), sem -race. Folga larga de
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
	const numQueries = 500
	const teto = 80 * time.Millisecond

	v, idx, construido, _ := geraCorpus(t, corpusSize)

	// O kernel e medido sobre o indice VINDO DO CACHE, e nao sobre o construido
	// do zero.
	//
	// CalculateBM25 e o unico consumidor de Inverted.Postings, que e exatamente
	// onde os dois ramos divergem: com base != nil ele devolve a fatia do array
	// achatado; sem base ele materializa o mapa do delta. O servidor sempre
	// carrega do cache, e ate 2026-08-13 este teto de 80 ms cobrava o outro
	// ramo. Mesmo defeito das Tasks 94 e 98, no terceiro teste com teto —
	// achado pela revisao adversarial da 98, que reparou que ele roda na mesma
	// etapa do verify.ps1 que os outros dois.
	ctx := context.Background()
	cacheDir := t.TempDir()
	if err := search.SaveInvertedCache(ctx, cacheDir, v.Root(), construido); err != nil {
		t.Fatalf("SaveInvertedCache: %v", err)
	}
	inv, _, err := search.LoadInvertedCache(ctx, cacheDir, v.Root())
	if err != nil {
		t.Fatalf("LoadInvertedCache: %v", err)
	}
	// Antes de qualquer assercao: t.Fatal com a arena mapeada faz o TempDir
	// falhar por cima e soterra o motivo real sob "RemoveAll cleanup".
	t.Cleanup(func() { _ = inv.Close() })
	if inv == nil || !inv.VindoDoCache() {
		t.Fatal("o indice nao veio do cache; o teto cobraria o ramo que o servidor nao executa")
	}
	if got := inv.DocCount(); got != corpusSize {
		t.Fatalf("o indice vindo do cache tem %d documentos, quer %d", got, corpusSize)
	}

	// "prescricao" está em todas as notas de geraCorpus. Confirmado antes de
	// medir: se o termo sumir do corpus, esta medição volta a ser sobre um
	// dicionário dando miss, e ninguém percebe pelo verde.
	tokens := search.Analyze("prescricao")
	if got := len(search.CalculateBM25(tokens, inv, idx)); got != corpusSize {
		t.Fatalf("a consulta casou %d notas de %d; com poucas postings a medição "+
			"não mede o kernel", got, corpusSize)
	}

	// Medição AMORTIZADA por grupo, e não por chamada.
	//
	// Cronometrar cada chamada individual parou de funcionar quando as
	// otimizações de M7 (pool de normalização, título pré-normalizado,
	// postings buscadas uma vez) deixaram o kernel mais rápido que a
	// granularidade do relógio nesta plataforma: a mediana passou a ler `0s` e
	// o p95 a sair quantizado em múltiplos de milissegundo. O guarda de
	// "mediana = 0" disparava, e a mensagem dele culpava a consulta por ser
	// seletiva demais — quando o teste já afirma, logo acima, que ela casa as
	// 500 notas.
	//
	// Somando um grupo de chamadas e dividindo, cada amostra fica muito acima
	// do tique do relógio e volta a ter significado. É o mesmo remédio que o
	// benchmark de IPC deste projeto precisou pelo mesmo motivo.
	// 10 chamadas por amostra levantam cada medicao bem acima do tique do
	// relogio; 50 amostras dao um p95 que nao e simplesmente o maximo.
	const porGrupo = 10
	const grupos = numQueries / porGrupo

	durations := make([]time.Duration, grupos)
	for g := 0; g < grupos; g++ {
		start := time.Now()
		for i := 0; i < porGrupo; i++ {
			_ = search.CalculateBM25(tokens, inv, idx)
		}
		durations[g] = time.Since(start) / porGrupo
	}

	sort.Slice(durations, func(i, j int) bool {
		return durations[i] < durations[j]
	})

	minDur := durations[0]
	medianDur := durations[grupos/2]
	// Aritmetica inteira: `float64(grupos) * 0.95` e expressao constante, e Go
	// recusa truncar constante para int.
	idxP95 := grupos * 95 / 100
	if idxP95 >= grupos {
		idxP95 = grupos - 1
	}
	p95Dur := durations[idxP95]

	t.Logf("Kernel BM25 (%d consultas em %d grupos de %d, %d postings em %d notas):",
		numQueries, grupos, porGrupo, corpusSize, corpusSize)
	t.Logf("  Mínimo:  %v", minDur)
	t.Logf("  Mediana: %v", medianDur)
	t.Logf("  p95:     %v  (teto deste teste: %v)", p95Dur, teto)

	// Mediana em zero continua sendo falha: com a medição amortizada, ela só
	// pode acontecer se o relógio não andar em 20 chamadas seguidas, e aí a
	// medição inteira não guarda nada. A mensagem antiga culpava a consulta
	// por ser seletiva, o que o teste já descartou na asserção acima.
	if medianDur == 0 {
		t.Errorf("mediana = 0s mesmo amortizando %d chamadas por amostra: "+
			"o relogio desta plataforma nao mede o kernel nesta escala",
			porGrupo)
	}
	if !raceEnabled && p95Dur > teto {
		t.Errorf("p95 %v excede o teto de %v deste teste (regressão algorítmica no "+
			"kernel; o RNF-04 é medido em TestRNF04VaultSearchLatencyP95)", p95Dur, teto)
	}
}

// TestIndiceRecarregadoEIdenticoAoConstruido guarda dois defeitos que estavam
// juntos no mesmo lugar e nao apareciam em teste nenhum.
//
//  1. DocLength divergia. Ele era DERIVADO na leitura, somando o numero de
//     posicoes de cada termo — e um token cuja forma reduzida difere da raiz
//     entra em DUAS postings. Um documento de 5 tokens que todos reduzem media
//     5 recem-construido e 10 recarregado do cache. DocLength e o divisor da
//     normalizacao por tamanho do BM25: o mesmo cofre ranqueava diferente
//     conforme o servidor tivesse acabado de indexar ou de ler o cache.
//
//  2. Nota sem token nenhum sumia. Ela nao entrava em docLengths, entao nao
//     contava em DocCount, entao o cabecalho do cache declarava menos notas do
//     que o indice de metadados via — e o boot achava o cache incompleto TODA
//     vez, reconstruindo e regravando o cache inteiro a cada partida.
//
// O teste compara os dois indices campo a campo em vez de conferir um valor
// esperado escrito a mao: assim ele continua valendo quando o analisador mudar.
func TestIndiceRecarregadoEIdenticoAoConstruido(t *testing.T) {
	cacheDir := t.TempDir()
	vaultPath := t.TempDir()
	ctx := context.Background()

	// "prescricoes" e "intercorrentes" reduzem; "md" nao. A mistura importa:
	// um corpus onde nada reduz passaria mesmo com o bug.
	fresco := search.NewInverted()
	fresco.Add("a.md", search.Analyze("prescricoes intercorrentes execucoes recursos"))
	fresco.Add("b.md", search.Analyze("civil"))
	fresco.Add("vazia.md", search.Analyze(""))

	if err := search.SaveInvertedCache(ctx, cacheDir, vaultPath, fresco); err != nil {
		t.Fatalf("SaveInvertedCache: %v", err)
	}
	lido, hdr, err := search.LoadInvertedCache(ctx, cacheDir, vaultPath)
	if err != nil {
		t.Fatalf("LoadInvertedCache: %v", err)
	}
	t.Cleanup(func() { _ = lido.Close() })

	if lido.DocCount() != fresco.DocCount() {
		t.Errorf("DocCount recarregado = %d, construido = %d", lido.DocCount(), fresco.DocCount())
	}
	if hdr.NoteCount != fresco.DocCount() {
		t.Errorf("hdr.NoteCount = %d, quer %d — o boot compara isto com o indice de metadados",
			hdr.NoteCount, fresco.DocCount())
	}
	for _, path := range []string{"a.md", "b.md", "vazia.md"} {
		if !lido.HasDoc(path) {
			t.Errorf("%s sumiu do cache", path)
		}
		if got, want := lido.DocLength(path), fresco.DocLength(path); got != want {
			t.Errorf("DocLength(%s) recarregado = %d, construido = %d", path, got, want)
		}
	}
	if lido.TermCount() != fresco.TermCount() {
		t.Errorf("TermCount recarregado = %d, construido = %d", lido.TermCount(), fresco.TermCount())
	}
}
