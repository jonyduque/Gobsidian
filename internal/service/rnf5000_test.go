package service_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"testing"
	"time"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/search"
	"github.com/jonyd/gobsidian/internal/service"
	"github.com/jonyd/gobsidian/internal/vault"
)

// Os tetos deste arquivo vem de docs/PRD.md secao 6.1, e de lugar nenhum mais.
//
//   - RNF-01, PRD.md:288 — indexacao a frio: alvo <= 3 s, limite de falha 6 s.
//   - RNF-07, PRD.md:294 — heap vivo em repouso: alvo <= 8 MB + 32 KB x notas,
//     limite de falha 2x o alvo. Para as 5.000 notas deste cofre o alvo da
//     8 + 32 x 5000 / 1024 = 164,25 MB. A conta e a mesma que
//     scripts/measure.ps1:223-227 faz, com as mesmas constantes.
//
// RESSALVA QUE ESTE TESTE NAO PODE DEIXAR DE FAZER: o cofre que gen_vault.ps1
// produz tem as 5.000 notas do cofre de referencia do PRD, mas nao os 50 MB —
// medido em 2026-09-04, 5.050 arquivos e 1,4 MB. Medido no mesmo dia nesta
// maquina, a indexacao sai em ~370 ms de mediana (medido em 2026-09-04:
// mediana de 371,5 ms) contra o limite de 6 s, e o heap vivo em ~12,5 MB
// contra o limite de 328,5 MB. Os tetos aqui sao
// anteparo contra regressao CATASTROFICA e nada mais; verde neste teste NAO
// autoriza escrever que o RNF-01 ou o RNF-07 estao atingidos. Quem responde
// isso e `scripts/measure.ps1` contra um cofre real, e o resultado mora em
// docs/OPERACAO.md.
const (
	rnf01Alvo   = 3 * time.Second
	rnf01Limite = 6 * time.Second

	// Uma conta so para o tamanho do cofre: o teto do RNF-07 escala com ele, e
	// as guardas de contagem falam do mesmo numero.
	rnf5000Notas = 5000

	rnf07BaseMB    = 8.0
	rnf07KBPorNota = 32.0
	rnf07AlvoMB    = rnf07BaseMB + rnf07KBPorNota*rnf5000Notas/1024
	rnf07LimiteMB  = 2 * rnf07AlvoMB
)

func getVault5000Path(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(os.TempDir(), "vault_5000")
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Skipf("cofre de 5000 notas nao encontrado em %s; gere com gen_vault.ps1", dir)
	}
	return dir
}

func TestScale5000_RNF01_RNF02_RNF07_RNF04(t *testing.T) {
	vaultDir := getVault5000Path(t)

	// 1. RNF-01: Indexação a frio (5 execuções)
	var rnf01Times []time.Duration
	for i := 0; i < 5; i++ {
		v, err := vault.New(vaultDir)
		if err != nil {
			t.Fatalf("vault.New: %v", err)
		}
		idx := index.New()
		start := time.Now()
		if err := idx.Build(context.Background(), v); err != nil {
			t.Fatalf("idx.Build: %v", err)
		}
		dur := time.Since(start)
		if idx.NoteCount() != rnf5000Notas {
			t.Fatalf("NoteCount = %d, quer %d", idx.NoteCount(), rnf5000Notas)
		}
		rnf01Times = append(rnf01Times, dur)
	}

	sort.Slice(rnf01Times, func(i, j int) bool { return rnf01Times[i] < rnf01Times[j] })
	t.Logf("=== RNF-01 (Indexacao a Frio 5.000 notas) ===")
	for i, d := range rnf01Times {
		t.Logf("  Rodada %d: %v", i+1, d)
	}
	t.Logf("  Min: %v, Mediana: %v, Max: %v (alvo %v, limite de falha %v)",
		rnf01Times[0], rnf01Times[2], rnf01Times[4], rnf01Alvo, rnf01Limite)

	// Teto, e nao so `Logf`. Ate 2026-09-04 este teste inteiro so imprimia:
	// mediu o RNF-01 cinco vezes e nao tinha como reprovar por nenhum valor.
	//
	// Cobrado sobre a MEDIANA, e nao sobre o maximo: cinco amostras numa
	// maquina de desenvolvimento com outros processos vivos produzem um
	// outlier alto que nao diz nada sobre o produto — a primeira rodada ainda
	// paga a leitura fria de disco do sistema operacional (o mesmo efeito que
	// PRD.md registra nas partidas de 2026-08-06: 7736 ms na primeira e
	// 852-901 ms nas quatro seguintes).
	//
	// Guarda `!raceEnabled` pelo motivo de sempre: o detector multiplica a
	// latencia por 2x-6x, e comparar esse numero com um teto do produto
	// reprovaria por motivo que nao e do produto.
	if !raceEnabled && rnf01Times[2] > rnf01Limite {
		t.Errorf("RNF-01: mediana de %v excede o limite de falha de %v (alvo %v, PRD.md:288)",
			rnf01Times[2], rnf01Limite, rnf01Alvo)
	}

	// 2. RNF-02: Boot com cache válido (5 execuções)
	v, _ := vault.New(vaultDir)
	idx := index.New()
	_ = idx.Build(context.Background(), v)
	inv := search.NewInverted()
	for _, p := range idx.NotePaths() {
		_ = inv.Update(context.Background(), v, p)
	}
	cacheDir := t.TempDir()
	if err := search.SaveInvertedCache(context.Background(), cacheDir, vaultDir, inv); err != nil {
		t.Fatalf("SaveInvertedCache: %v", err)
	}

	var rnf02Times []time.Duration
	for i := 0; i < 5; i++ {
		start := time.Now()
		loaded, _, err := search.LoadInvertedCache(context.Background(), cacheDir, vaultDir)
		if err != nil {
			t.Fatalf("LoadInvertedCache: %v", err)
		}
		dur := time.Since(start)
		rnf02Times = append(rnf02Times, dur)
		// Task 89: fecha a arena mapeada (se houver) ANTES da proxima volta —
		// medido o tempo de carga, o mapeamento nao serve mais para nada, e
		// cinco deles abertos ao mesmo tempo sobre o mesmo arquivo fariam o
		// t.TempDir() do fim do teste recusar apagar o diretorio no Windows.
		_ = loaded.Close()
	}
	sort.Slice(rnf02Times, func(i, j int) bool { return rnf02Times[i] < rnf02Times[j] })
	t.Logf("=== RNF-02 (Boot com Cache Valido 5.000 notas) ===")
	for i, d := range rnf02Times {
		t.Logf("  Rodada %d: %v", i+1, d)
	}
	t.Logf("  Min: %v, Mediana: %v, Max: %v", rnf02Times[0], rnf02Times[2], rnf02Times[4])

	// 3. RNF-07: heap vivo em repouso.
	//
	// O rotulo dizia "RSS em repouso", que e a redacao ANTERIOR a 2026-08-28 —
	// PRD.md:298-307 conta por que a metrica deixou de ser RSS: o RSS segue a
	// meta de heap do GC, e medido em 2026-08-27 ele chegou a inverter de
	// sinal entre dois binarios. `HeapAlloc` logo depois de um `runtime.GC()`
	// e a aproximacao de heap vivo que este processo consegue dar.
	var mem runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&mem)
	heapVivo := float64(mem.HeapAlloc) / 1024 / 1024
	t.Logf("=== RNF-07 (heap vivo em repouso, 5.000 notas) ===")
	t.Logf("  HeapAlloc apos GC: %.2f MB (alvo <= %.2f MB, limite de falha %.2f MB)",
		heapVivo, rnf07AlvoMB, rnf07LimiteMB)
	t.Logf("  diagnostico, NAO e o requisito: Alloc %.2f MB, Sys %.2f MB",
		float64(mem.Alloc)/1024/1024, float64(mem.Sys)/1024/1024)

	// O teto vale nos DOIS modos: o detector de corrida mexe em tempo, nao no
	// tamanho do heap vivo, entao aqui nao ha motivo para a guarda
	// `!raceEnabled` que os tetos de latencia usam.
	if heapVivo > rnf07LimiteMB {
		t.Errorf("RNF-07: heap vivo de %.2f MB excede o limite de falha de %.2f MB (alvo %.2f MB, PRD.md:294)",
			heapVivo, rnf07LimiteMB, rnf07AlvoMB)
	}

	// 4. RNF-04: Latência de vault_search p95 a 5.000 notas
	//
	// O serviço é montado com o índice VINDO DO CACHE, não com inv.
	//
	// Inverted.Postings tem dois ramos. Índice construído do zero (inv):
	// base == nil, tudo vive no delta em mapas, e a função ORDENA. Índice
	// carregado do cache: base != nil e delta vazio, e ela devolve a fatia do
	// base sem ordenar. O servidor em produção sempre carrega do cache, e
	// medido no mesmo cofre e na mesma consulta os dois ramos diferem por 4,4x
	// (BenchmarkSearchLimit200 174.791.983 ns/op contra
	// BenchmarkSearchLimit200Cache 39.565.533 ns/op). Até 2026-08-12 este bloco
	// media inv, isto é, o ramo que o servidor não executa.
	//
	// Instância própria, viva até o fim do teste: a do laço de RNF-02 é
	// fechada a cada volta de propósito (ver o comentário lá).
	doCache, _, err := search.LoadInvertedCache(context.Background(), cacheDir, vaultDir)
	if err != nil {
		t.Fatalf("LoadInvertedCache para o RNF-04: %v", err)
	}
	defer func() { _ = doCache.Close() }()

	// Guarda de ramo, no espírito da de bench_cache_test.go. Um cache recusado
	// em silêncio — troca de formato, versão de analisador, caminho de cofre
	// diferente — faria este bloco medir exatamente o ramo que ele existe para
	// NÃO medir, e ninguém notaria.
	//
	// Pergunta pelo RAMO com VindoDoCache, e nao por contagem. A versao anterior
	// deste comentario argumentava que DocCount distinguia os ramos por deducao
	// sobre newInvertedFromSoA — deducao correta e frágil, e que rederivava pela
	// terceira vez uma resposta que agora tem uma funcao so. A contagem fica
	// logo abaixo, medindo outra coisa: que o corpus veio inteiro.
	if doCache == nil {
		t.Fatal("LoadInvertedCache devolveu cache nulo; sem base o RNF-04 mediria o ramo do delta")
	}
	if !doCache.VindoDoCache() {
		t.Fatal("o indice nao veio do cache; o RNF-04 mediria o ramo do delta")
	}
	if doCache.DocCount() < rnf5000Notas {
		t.Fatalf("o indice vindo do cache tem %d documentos, quer >= %d; "+
			"o cache foi recusado e o RNF-04 mediria o ramo do delta", doCache.DocCount(), rnf5000Notas)
	}

	// Cache de trecho DESLIGADO: o laço abaixo repete cada consulta 30 vezes, e
	// com o cache ligado 29 dessas 30 acertariam. O p95 cairia para o valor da
	// consulta repetida, que nenhum usuário vê na primeira busca — seria um RNF
	// declarado atingido por medir outra coisa.
	svc := service.New(v, idx, doCache, nil, service.Options{SnippetCacheEntries: &semCacheDeTrecho})
	t.Logf("=== RNF-04 (Latencia vault_search p95 5.000 notas, indice vindo do CACHE) ===")

	// Tres destas oito consultas casavam ZERO documentos no cofre que
	// gen_vault.ps1 produz, e o teste so imprimia — media a latencia de nao
	// achar nada e chamava isso de RNF-04. bench_test.go:173-178 registra que
	// a mesma troca de corpus quebrou o benchmark em 2026-09-01; aqui nao
	// quebrou nada porque nada era afirmado.
	//
	// Trocadas, e cada troca conferida DUAS vezes no cofre em
	// %TEMP%\vault_5000 (5.000 notas .md; a semente que o gerou NAO foi
	// verificada — o diretorio foi reaproveitado, nao regenerado, entao nao da
	// para afirmar que veio de `-Seed 42`), nesta maquina, em 2026-09-04: por
	// `grep -ril` (arquivos que contem o termo, de 5.000) e pelo `Total` que a
	// propria busca devolve — os dois numeros abaixo, nessa ordem:
	//
	//   nota ................................. grep 5000  / busca 5000
	//   decisao reconheceu ................... grep 1224 e 1224 / busca 1224
	//   Acentuada ............................ grep 1214  / busca 1214
	//   nota + pasta Projetos ................ 1165 .md sob Projetos/ / busca 1165
	//   nota + tag golang .................... grep 608   / busca 608
	//   "contra a decisao que reconheceu" .... grep 1224  / busca 1224
	//   acordao firmou ....................... grep 1234 e 1234 / busca 1234
	//
	// E as tres que sairam, medidas do mesmo jeito e no mesmo cofre:
	//
	//   servidor ... 8 arquivos, e nos OITO como pedaco de palavra inventada
	//                ("imservidorio", "extraservidorio"); como TOKEN, zero
	//   algoritmo 0, BM25 0, pesos 0
	//   comportamento 0, watcher 0
	//
	// Cuidado ao trocar de novo: a consulta de varios termos e OU, nao E, e
	// basta UM termo casar. "xyzzy-inexistente" parece nao existir e casa 1323
	// notas, porque "inexistente" esta em 1323 delas — foi assim que a
	// primeira tentativa de provar a guarda abaixo passou verde. Um termo que
	// de fato nao existe aqui: "zarabatana".
	queries := []struct {
		nome string
		opts service.SearchOptions
	}{
		{"termo amplo, limit default", service.SearchOptions{Query: "nota"}},
		{"dois termos", service.SearchOptions{Query: "decisao reconheceu"}},
		{"termo seletivo", service.SearchOptions{Query: "Acentuada"}},
		{"filtro de pasta", service.SearchOptions{Query: "nota", Folder: "Projetos"}},
		{"filtro de tag", service.SearchOptions{Query: "nota", Tags: []string{"golang"}}},
		{"frase exata", service.SearchOptions{Query: `"contra a decisao que reconheceu"`}},
		{"trecho maximo", service.SearchOptions{Query: "acordao firmou", SnippetChars: 1000}},
		{"limit maximo do schema", service.SearchOptions{Query: "nota", Limit: 200}},
	}

	// Guarda de corpus, ANTES de medir. Fatal, e nao Skip: um corpus que nao
	// serve e defeito de ambiente que o gate tem de ver. Um Skip aqui devolve
	// exatamente a situacao que esta tarefa veio consertar — verde sem
	// medicao.
	for _, q := range queries {
		res, err := svc.Search(context.Background(), q.opts)
		if err != nil {
			t.Fatalf("guarda de corpus, consulta %q (%s): %v", q.opts.Query, q.nome, err)
		}
		if len(res.Results) == 0 {
			t.Fatalf("a consulta %q (%s) nao casa nada no corpus %s; o corpus mudou ou a consulta esta errada",
				q.opts.Query, q.nome, vaultDir)
		}
		t.Logf("  guarda de corpus: %-30s %d resultados (total %d)", q.nome, len(res.Results), res.Total)
	}

	for _, q := range queries {
		var lats []time.Duration
		for k := 0; k < 30; k++ {
			st := time.Now()
			_, err := svc.Search(context.Background(), q.opts)
			if err != nil {
				t.Fatalf("Search err: %v", err)
			}
			lats = append(lats, time.Since(st))
		}
		sort.Slice(lats, func(i, j int) bool { return lats[i] < lats[j] })
		mediana := lats[15]
		p95 := lats[int(float64(len(lats))*0.95)-1]
		t.Logf("  %-30s mediana %-10v p95 %-10v", q.nome, mediana, p95)
	}
}
