package search

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// gravaCacheDeTeste escreve um cache válido do formato corrente em caminho, a
// partir de um Inverted qualquer.
func gravaCacheDeTeste(t *testing.T, caminho string, origem *Inverted, vaultPath string) {
	t.Helper()
	termos, comp := origem.ExportForCache()
	f, err := os.Create(caminho)
	if err != nil {
		t.Fatalf("os.Create: %v", err)
	}
	h := CacheHeader{
		FormatVersion:   CacheFormatVersion,
		ParserVersion:   CacheParserVersion,
		AnalyzerVersion: CacheAnalyzerVersion,
		VaultPath:       vaultPath,
		NoteCount:       origem.DocCount(),
	}
	if err := escreveCache(f, h, termos, comp); err != nil {
		_ = f.Close()
		t.Fatalf("escreveCache: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

// TestRecusaMapearCacheDentroDoCofre confere a guarda que impede mapear um
// arquivo de cache que more DENTRO do cofre.
//
// O cofre de referência fica em OneDrive, e um arquivo mapeado que o
// sincronizador mexe embaixo dos pés é uma classe de falha que este projeto
// ainda não pagou (ver CLAUDE.md, "O cofre fica em OneDrive"). O cache mora
// fora do cofre por decisão de configuração — tentaAbrirArena confere isso em
// tempo de execução, e não confia que a configuração está certa: um
// --cache-dir apontado para dentro do cofre por engano não pode resultar em
// mapear um arquivo que o OneDrive pode reescrever por baixo.
//
// Prova de mutação: `pwsh -File scripts/mutate.ps1 -Path internal/search/mmap.go
// -Anchor 'if dentroDoCofre(caminhoCache, vaultPath) {' -Replacement 'if false {'
// -Test TestRecusaMapearCacheDentroDoCofre -Package ./internal/search/`
func TestRecusaMapearCacheDentroDoCofre(t *testing.T) {
	vaultPath := t.TempDir()
	cacheDir := filepath.Join(vaultPath, ".gobsidian-cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		t.Fatal(err)
	}
	cachePath := filepath.Join(cacheDir, "inverted_cache.gob")

	inv := NewInverted()
	inv.Add("a.md", Analyze("prescricao intercorrente"))
	gravaCacheDeTeste(t, cachePath, inv, vaultPath)

	arena, ok := tentaAbrirArena(cachePath, vaultPath)
	if ok {
		if arena != nil {
			_ = arena.fechar()
		}
		t.Fatal("tentaAbrirArena mapeou um cache dentro do cofre; a guarda nao disparou")
	}
	if arena != nil {
		t.Fatal("tentaAbrirArena devolveu uma arena nao-nil junto de ok=false")
	}
}

// TestMapeiaForaDoCofre é o controle positivo de
// TestRecusaMapearCacheDentroDoCofre: o mesmo cache, num diretório que NÃO
// está dentro do cofre, tem de mapear com sucesso. Sem este teste, a guarda
// acima poderia estar recusando TUDO — inclusive o caminho que deveria
// funcionar — e passaria do mesmo jeito.
func TestMapeiaForaDoCofre(t *testing.T) {
	vaultPath := t.TempDir()
	cacheDir := t.TempDir()
	cachePath := filepath.Join(cacheDir, "inverted_cache.gob")

	inv := NewInverted()
	inv.Add("a.md", Analyze("prescricao intercorrente civil"))
	inv.Add("b.md", Analyze("processo civil recurso"))
	gravaCacheDeTeste(t, cachePath, inv, vaultPath)

	arena, ok := tentaAbrirArena(cachePath, vaultPath)
	if !ok {
		t.Fatal("tentaAbrirArena recusou um cache fora do cofre")
	}
	defer func() { _ = arena.fechar() }()

	if len(arena.pos) == 0 {
		t.Fatal("arena.pos vazia; o corpus de teste tem posicoes")
	}
}

// TestArenaMapeadaIdenticaANaoMapeada é o teste central desta tarefa: o
// índice montado sobre a arena mapeada (leCacheComArena) tem de responder
// EXATAMENTE igual ao montado sobre a decodificação integral (leCache), para
// o mesmo arquivo.
//
// Guarda o mesmo defeito que TestIndiceRecarregadoEIdenticoAoConstruido guarda
// para "construído do zero" vs. "recarregado do cache": aqui as duas
// CONSTRUÇÕES A PARTIR DO CACHE (com e sem arena) têm de concordar campo a
// campo. Comparar contra um valor escrito à mão erraria junto com o código;
// comparar as duas decodificações não.
func TestArenaMapeadaIdenticaANaoMapeada(t *testing.T) {
	vaultPath := t.TempDir()
	cacheDir := t.TempDir()
	cachePath := filepath.Join(cacheDir, "inverted_cache.gob")

	origem := NewInverted()
	origem.Add("a.md", Analyze("prescricao intercorrente execucao fiscal civil"))
	origem.Add("b.md", Analyze("recurso extraordinario processo civil prescricao"))
	origem.Add("c.md", Analyze("termoexclusivo unico"))
	origem.Add("vazia.md", Analyze(""))
	gravaCacheDeTeste(t, cachePath, origem, vaultPath)

	dados, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatal(err)
	}
	_, semArena, err := decodificaCache(dados, nil)
	if err != nil {
		t.Fatalf("decodificaCache sem arena: %v", err)
	}

	arena, ok := tentaAbrirArena(cachePath, vaultPath)
	if !ok {
		t.Fatal("tentaAbrirArena recusou um cache valido fora do cofre")
	}
	defer func() { _ = arena.fechar() }()
	_, comArena, err := decodificaCache(arena.dados, arena.pos)
	if err != nil {
		t.Fatalf("decodificaCache com arena: %v", err)
	}

	ixSemArena := newInvertedFromSoA(semArena)
	ixComArena := newInvertedFromSoA(comArena)

	exigeEquivalentes(t, ixSemArena, ixComArena, []string{"a.md", "b.md", "c.md", "vazia.md"})
}

// TestArenaRecusaContagemDivergente confere que uma arena cujo tamanho não
// bate com o totPos do cabeçalho é recusada — "não mapeia lixo" — em vez de
// produzir uma base cujos limites mentem sobre o que ela contém.
func TestArenaRecusaContagemDivergente(t *testing.T) {
	origem := NewInverted()
	origem.Add("a.md", Analyze("prescricao intercorrente"))
	termos, comp := origem.ExportForCache()

	var buf bytes.Buffer
	h := CacheHeader{FormatVersion: CacheFormatVersion, ParserVersion: CacheParserVersion,
		AnalyzerVersion: CacheAnalyzerVersion, NoteCount: 1}
	if err := escreveCache(&buf, h, termos, comp); err != nil {
		t.Fatalf("escreveCache: %v", err)
	}

	// Arena com uma posicao a menos do que o cabecalho declara.
	arenaCurta := make([]TokenPosition, 0)
	_, _, err := decodificaCache(buf.Bytes(), arenaCurta)
	if err == nil {
		t.Fatal("decodificaCache aceitou arena com contagem divergente do totPos declarado")
	}
}

// TestFooterAusenteNaoImpedeCarga confere que um arquivo sem rodapé (cache de
// um formato anterior ao Task 89, ou corrompido especificamente no rodapé) não
// crasha tentaAbrirArena — só recusa mapear, e quem chama cai para a
// decodificação integral.
func TestFooterAusenteNaoImpedeCarga(t *testing.T) {
	vaultPath := t.TempDir()
	cacheDir := t.TempDir()
	cachePath := filepath.Join(cacheDir, "inverted_cache.gob")

	inv := NewInverted()
	inv.Add("a.md", Analyze("prescricao"))
	gravaCacheDeTeste(t, cachePath, inv, vaultPath)

	// Corrompe so a assinatura do rodape (os primeiros 8 dos ultimos 24
	// bytes), deixando o resto do arquivo intacto.
	dados, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatal(err)
	}
	corrompido := make([]byte, len(dados))
	copy(corrompido, dados)
	for i := len(corrompido) - footerBytes; i < len(corrompido)-footerBytes+8; i++ {
		corrompido[i] = 0xFF
	}
	if err := os.WriteFile(cachePath, corrompido, 0644); err != nil {
		t.Fatal(err)
	}

	arena, ok := tentaAbrirArena(cachePath, vaultPath)
	if ok {
		_ = arena.fechar()
		t.Fatal("tentaAbrirArena mapeou um arquivo com rodape corrompido")
	}

	// O corpo em varint continua intacto: a decodificacao integral tem de
	// funcionar normalmente a partir daqui.
	h, base, err := leCache(corrompido)
	if err != nil {
		t.Fatalf("leCache falhou com rodape corrompido, mas corpo intacto: %v", err)
	}
	if h.NoteCount != 1 || len(base.caminhos) != 1 {
		t.Fatalf("decodificacao integral nao recuperou o conteudo: %+v", h)
	}
}
