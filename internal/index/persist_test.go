package index_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/vault"
)

func TestSaveAndLoadIndexCache(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "A.md", "# A\n\n[[B]]\n")
	writeFile(t, root, "B.md", "# B\n")
	writeFile(t, root, "img.png", "img")

	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	idx := index.New()
	if err := idx.Build(context.Background(), v); err != nil {
		t.Fatalf("Build: %v", err)
	}

	cacheDir := t.TempDir()
	ctx := context.Background()
	if err := index.SaveIndexCache(ctx, cacheDir, root, idx); err != nil {
		t.Fatalf("SaveIndexCache: %v", err)
	}

	loaded, hdr, err := index.LoadIndexCache(ctx, cacheDir, root)
	if err != nil {
		t.Fatalf("LoadIndexCache: %v", err)
	}
	if hdr.NoteCount != 2 {
		t.Errorf("hdr.NoteCount = %d, quer 2", hdr.NoteCount)
	}
	if hdr.AssetCount != 1 {
		t.Errorf("hdr.AssetCount = %d, quer 1", hdr.AssetCount)
	}
	if loaded.NoteCount() != 2 || loaded.AssetCount() != 1 {
		t.Errorf("indice carregado = %d notas / %d anexos, quer 2/1", loaded.NoteCount(), loaded.AssetCount())
	}

	n, ok := loaded.Get("A.md")
	if !ok {
		t.Fatal("A.md nao entrou no indice carregado")
	}
	if len(n.Links) != 1 || n.Links[0].Resolved != "B.md" {
		t.Errorf("link de A.md nao resolveu apos o load: %+v", n.Links)
	}
}

// TestIndiceDeMetadadosRecarregadoEIdentico compara os dois caminhos de
// construcao campo a campo, em vez de conferir valores escritos a mao. Valor
// escrito a mao codifica o mesmo engano do codigo; o caminho de construcao do
// zero e o que ja estava certo.
//
// O corpus cobre, de proposito, cada armadilha ja paga citada no brief:
//   - "Civil/PONTO 03.md" tem alias (P3), frontmatter com int/bool/lista, e
//     um link com ancora quebrada (para "Origem.md#NaoExiste") ao lado de um
//     link plano — cobre nota-com-alias e nota-com-ancora-quebrada, e prova
//     que Resolved/Via/State recalculados apos o load batem com os do
//     indice fresco mesmo sem serem persistidos (ver escritor.links).
//   - "Origem.md" recebe os dois links de PONTO 03 — cobre nota-com-backlink,
//     com DOIS backlinks do mesmo caminho de origem.
//   - "Vazia.md" tem zero bytes — cobre a nota vazia que "nunca contava como
//     coberta" na versao anterior do cache de busca; aqui exercita nil vs
//     slice/mapa vazio em quase todo campo de Note ao mesmo tempo.
//   - "Anexos/diagrama.png" cobre o anexo.
//   - ResolvePath("civil/ponto 03.md") contra "Civil/PONTO 03.md" cobre o
//     nome que colide em caixa.
func TestIndiceDeMetadadosRecarregadoEIdentico(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "Civil/PONTO 03.md", "---\n"+
		"aliases: [P3]\n"+
		"tags: [direito]\n"+
		"numero: 42\n"+
		"ativo: true\n"+
		"---\n"+
		"# Ponto 3\n\n"+
		"Ver [[Origem]] e [[Origem#NaoExiste]].\n")
	writeFile(t, root, "Origem.md", "# Origem\n\nConteudo qualquer.\n")
	writeFile(t, root, "Vazia.md", "")
	writeFile(t, root, "Anexos/diagrama.png", "\x89PNG")

	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	fresco := index.New()
	if err := fresco.Build(context.Background(), v); err != nil {
		t.Fatalf("Build: %v", err)
	}

	// A ancora quebrada e o alias precisam estar no corpus ANTES da
	// comparacao — checagem de sanidade do proprio fixture, nao do codec.
	pontoFresco, ok := fresco.Get("Civil/PONTO 03.md")
	if !ok {
		t.Fatal("fixture: Civil/PONTO 03.md nao entrou no indice fresco")
	}
	if len(pontoFresco.Aliases) != 1 || pontoFresco.Aliases[0] != "P3" {
		t.Fatalf("fixture: alias nao capturado: %v", pontoFresco.Aliases)
	}
	achouAncoraQuebrada := false
	for _, l := range pontoFresco.Links {
		if l.State == index.LinkAnchorMissing {
			achouAncoraQuebrada = true
		}
	}
	if !achouAncoraQuebrada {
		t.Fatal("fixture: nenhum link com ancora quebrada no indice fresco")
	}
	if len(fresco.Backlinks("Origem.md")) != 2 {
		t.Fatalf("fixture: Origem.md deveria ter 2 backlinks, tem %d", len(fresco.Backlinks("Origem.md")))
	}

	cacheDir := t.TempDir()
	ctx := context.Background()
	if err := index.SaveIndexCache(ctx, cacheDir, root, fresco); err != nil {
		t.Fatalf("SaveIndexCache: %v", err)
	}
	lido, hdr, err := index.LoadIndexCache(ctx, cacheDir, root)
	if err != nil {
		t.Fatalf("LoadIndexCache: %v", err)
	}

	// Campo a campo: Paths, NoteCount, AssetCount, TotalSize, Tags("").
	if got, want := lido.Paths(), fresco.Paths(); !reflect.DeepEqual(got, want) {
		t.Errorf("Paths() recarregado = %v, construido = %v", got, want)
	}
	if got, want := lido.NoteCount(), fresco.NoteCount(); got != want {
		t.Errorf("NoteCount() recarregado = %d, construido = %d", got, want)
	}
	if got, want := lido.AssetCount(), fresco.AssetCount(); got != want {
		t.Errorf("AssetCount() recarregado = %d, construido = %d", got, want)
	}
	if got, want := lido.TotalSize(), fresco.TotalSize(); got != want {
		t.Errorf("TotalSize() recarregado = %d, construido = %d", got, want)
	}
	if got, want := lido.Tags("", 0), fresco.Tags("", 0); !reflect.DeepEqual(got, want) {
		t.Errorf("Tags(\"\", 0) recarregado = %v, construido = %v", got, want)
	}
	if hdr.NoteCount != fresco.NoteCount() {
		t.Errorf("hdr.NoteCount = %d, quer %d — o boot compara isto com a varredura", hdr.NoteCount, fresco.NoteCount())
	}

	// Por caminho: Get, Backlinks, ResolvePath do nome curto.
	for _, p := range fresco.Paths() {
		freNote, freOk := fresco.Get(p)
		loNote, loOk := lido.Get(p)
		if freOk != loOk {
			t.Errorf("Get(%s): presenca divergiu — fresco=%v, recarregado=%v", p, freOk, loOk)
			continue
		}
		if freOk {
			if !reflect.DeepEqual(freNote, loNote) {
				t.Errorf("Get(%s) divergiu campo a campo:\nfresco     = %+v\nrecarregado = %+v", p, freNote, loNote)
			}
		}

		freBL := fresco.Backlinks(p)
		loBL := lido.Backlinks(p)
		if !reflect.DeepEqual(freBL, loBL) {
			t.Errorf("Backlinks(%s) divergiu: fresco=%+v, recarregado=%+v", p, freBL, loBL)
		}
	}

	// ResolvePath do nome curto, dos dois indices.
	for _, input := range []string{"PONTO 03.md", "Origem.md", "Vazia.md", "diagrama.png"} {
		freP, freErr := fresco.ResolvePath(input)
		loP, loErr := lido.ResolvePath(input)
		if (freErr == nil) != (loErr == nil) || freP != loP {
			t.Errorf("ResolvePath(%q): fresco=(%q,%v), recarregado=(%q,%v)", input, freP, freErr, loP, loErr)
		}
	}

	// Nome que colide em caixa: entrada em minusculo tem de resolver para a
	// grafia do disco nos dois indices, identicamente.
	freP, freErr := fresco.ResolvePath("civil/ponto 03.md")
	loP, loErr := lido.ResolvePath("civil/ponto 03.md")
	if freErr != nil || loErr != nil {
		t.Fatalf("ResolvePath case-insensitive falhou: fresco=%v, recarregado=%v", freErr, loErr)
	}
	if freP != "Civil/PONTO 03.md" || loP != "Civil/PONTO 03.md" {
		t.Errorf("ResolvePath case-insensitive = fresco:%q recarregado:%q, quer Civil/PONTO 03.md", freP, loP)
	}
	if freP != loP {
		t.Errorf("ResolvePath case-insensitive divergiu entre os dois indices: %q vs %q", freP, loP)
	}

	// A nota vazia especificamente: tem de estar coberta nos dois lados, com
	// campos nil (nao fatia/mapa vazio) preservados.
	vaziaFresco, ok := fresco.Get("Vazia.md")
	if !ok {
		t.Fatal("fixture: Vazia.md nao entrou no indice fresco")
	}
	vaziaLida, ok := lido.Get("Vazia.md")
	if !ok {
		t.Fatal("Vazia.md sumiu no indice recarregado — a nota vazia nao contou como coberta")
	}
	if !reflect.DeepEqual(vaziaFresco, vaziaLida) {
		t.Errorf("Vazia.md divergiu:\nfresco     = %+v\nrecarregado = %+v", vaziaFresco, vaziaLida)
	}
	if vaziaFresco.Frontmatter != nil || vaziaLida.Frontmatter != nil {
		t.Errorf("Vazia.md deveria ter Frontmatter nil nos dois lados: fresco=%v, recarregado=%v",
			vaziaFresco.Frontmatter, vaziaLida.Frontmatter)
	}
}

// TestCacheDeMetadadosParcialERecusado e a regra que o cache de busca
// aprendeu na marra: LoadInvertedCache conferia versao e nao contagem, e um
// cache parcial passava por completo. Aqui o cabecalho MENTE sobre quantas
// notas o corpo traz, e LoadIndexCache tem de recusar em vez de devolver um
// indice incompleto como se fosse o cofre inteiro.
func TestCacheDeMetadadosParcialERecusado(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "A.md", "# A\n")
	writeFile(t, root, "B.md", "# B\n")

	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	idx := index.New()
	if err := idx.Build(context.Background(), v); err != nil {
		t.Fatalf("Build: %v", err)
	}
	if idx.NoteCount() != 2 {
		t.Fatalf("fixture: quer 2 notas, tem %d", idx.NoteCount())
	}

	var notes []*index.Note
	for _, p := range idx.NotePaths() {
		n, _ := idx.Get(p)
		notes = append(notes, n)
	}

	// Cabecalho declara 5 notas; o corpo so traz as 2 reais. Um cabecalho
	// fiel (o que SaveIndexCache sempre grava) nunca mentiria assim — este
	// arquivo so existe pra provar que LoadIndexCache CONFERE, nao confia.
	cabecalhoMentiroso := index.CacheHeader{
		FormatVersion: index.IndexCacheFormatVersion,
		ParserVersion: index.IndexCacheParserVersion,
		VaultPath:     root,
		NoteCount:     5,
		AssetCount:    0,
	}

	cacheDir := t.TempDir()
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	cachePath := filepath.Join(cacheDir, "index_cache.gob")
	f, err := os.Create(cachePath)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := index.WriteIndexCacheForTest(f, cabecalhoMentiroso, notes, nil); err != nil {
		_ = f.Close()
		t.Fatalf("WriteIndexCacheForTest: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	_, _, err = index.LoadIndexCache(context.Background(), cacheDir, root)
	if err == nil {
		t.Fatal("LoadIndexCache deveria recusar um cabecalho que declara mais notas do que o corpo traz")
	}
	if !errors.Is(err, index.ErrIndexCachePartial) {
		t.Fatalf("err = %v, quer ErrIndexCachePartial", err)
	}
}

func TestIndexCacheTruncatedRefused(t *testing.T) {
	cacheDir := t.TempDir()
	cachePath := filepath.Join(cacheDir, "index_cache.gob")
	if err := os.WriteFile(cachePath, []byte("tru"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, _, err := index.LoadIndexCache(context.Background(), cacheDir, "")
	if err == nil {
		t.Fatal("LoadIndexCache deveria falhar com arquivo truncado")
	}
	if !errors.Is(err, index.ErrIndexCacheCorrupted) {
		t.Fatalf("err = %v, quer ErrIndexCacheCorrupted", err)
	}
}

// TestIndexCacheByteCorruptedRefused corrompe um unico byte no MEIO do
// arquivo — nao no cabecalho, no corpo — e confere que LoadIndexCache
// recusa em vez de decodificar lixo. E o checksum (persist_codec.go) que
// garante isto de forma determinista: sem ele, um byte trocado dentro do
// PAYLOAD de uma string (em vez de um comprimento) nao violaria limite
// nenhum e decodificaria "com sucesso" um titulo ou caminho corrompido.
func TestIndexCacheByteCorruptedRefused(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < 30; i++ {
		writeFile(t, root, corpusNotaPath(i), "---\ntitle: Nota\naliases: [a, b]\n---\n# Titulo\n\nTexto de conteudo variado para engordar o arquivo.\n[[outra]]\n")
	}
	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	idx := index.New()
	if err := idx.Build(context.Background(), v); err != nil {
		t.Fatalf("Build: %v", err)
	}

	cacheDir := t.TempDir()
	ctx := context.Background()
	if err := index.SaveIndexCache(ctx, cacheDir, root, idx); err != nil {
		t.Fatalf("SaveIndexCache: %v", err)
	}

	cachePath := filepath.Join(cacheDir, "index_cache.gob")
	dados, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if len(dados) < 20 {
		t.Fatalf("arquivo pequeno demais pra este teste: %d bytes", len(dados))
	}

	meio := len(dados) / 2
	corrompido := bytes.Clone(dados)
	corrompido[meio] ^= 0xFF
	if err := os.WriteFile(cachePath, corrompido, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, _, err = index.LoadIndexCache(ctx, cacheDir, root)
	if err == nil {
		t.Fatal("LoadIndexCache deveria recusar um arquivo com um byte corrompido no meio")
	}
	if !errors.Is(err, index.ErrIndexCacheCorrupted) {
		t.Fatalf("err = %v, quer ErrIndexCacheCorrupted", err)
	}
}

func corpusNotaPath(i int) string {
	const letras = "abcdefghijklmnopqrstuvwxyz"
	return "nota-" + string(letras[i%len(letras)]) + string(letras[(i/len(letras))%len(letras)]) + ".md"
}

func TestVerifyFreshnessMatchesFreshBuild(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "A.md", "# A\n")
	writeFile(t, root, "B.png", "img")

	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	idx := index.New()
	if err := idx.Build(context.Background(), v); err != nil {
		t.Fatalf("Build: %v", err)
	}

	fresh, err := idx.VerifyFreshness(context.Background(), v)
	if err != nil {
		t.Fatalf("VerifyFreshness: %v", err)
	}
	if !fresh {
		t.Error("VerifyFreshness = false logo apos Build, sem nenhuma mudanca no disco")
	}
}

func TestVerifyFreshnessDetectsModification(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "A.md", "# A\n")

	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	idx := index.New()
	if err := idx.Build(context.Background(), v); err != nil {
		t.Fatalf("Build: %v", err)
	}

	// mtime precisa avancar de verdade: alguns sistemas de arquivos tem
	// resolucao de 1s ou 2s, e escrever de novo no mesmo instante nao muda
	// nada que VerifyFreshness possa detectar.
	novoTempo := time.Now().Add(5 * time.Second)
	abs := filepath.Join(root, "A.md")
	if err := os.WriteFile(abs, []byte("# A mudou, com mais conteudo\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := os.Chtimes(abs, novoTempo, novoTempo); err != nil {
		t.Fatalf("Chtimes: %v", err)
	}

	fresh, err := idx.VerifyFreshness(context.Background(), v)
	if err != nil {
		t.Fatalf("VerifyFreshness: %v", err)
	}
	if fresh {
		t.Error("VerifyFreshness = true depois de A.md mudar de tamanho e mtime")
	}
}

func TestVerifyFreshnessDetectsAddition(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "A.md", "# A\n")

	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	idx := index.New()
	if err := idx.Build(context.Background(), v); err != nil {
		t.Fatalf("Build: %v", err)
	}

	writeFile(t, root, "B.md", "# B, chegou depois do Build\n")

	fresh, err := idx.VerifyFreshness(context.Background(), v)
	if err != nil {
		t.Fatalf("VerifyFreshness: %v", err)
	}
	if fresh {
		t.Error("VerifyFreshness = true com um arquivo novo no disco que o indice nao viu")
	}
}
