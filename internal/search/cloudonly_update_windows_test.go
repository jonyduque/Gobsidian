//go:build windows

package search_test

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"golang.org/x/sys/windows"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/search"
	"github.com/jonyd/gobsidian/internal/vault"
)

// conteudoDaNuvem tem termos proprios, que nao aparecem em nenhuma outra nota
// dos cofres deste arquivo: se algum deles for indexado, foi porque o
// placeholder foi aberto.
const conteudoDaNuvem = "# Titulo da nuvem\n\nsesquipedaliano hidratado indevidamente\n"

// marcarSomenteNuvem poe FILE_ATTRIBUTE_OFFLINE e restaura no fim.
//
// FILE_ATTRIBUTE_RECALL_ON_DATA_ACCESS nao e gravavel por SetFileAttributes — e
// o motivo de TestReadNoteCloudOnlyFails estar pulado —, e vault.IsCloudOnly
// aceita os dois. So Windows porque o atributo e do NTFS.
func marcarSomenteNuvem(t *testing.T, abs string) {
	t.Helper()
	p, err := windows.UTF16PtrFromString(vault.LongPath(abs))
	if err != nil {
		t.Fatal(err)
	}
	if err := windows.SetFileAttributes(p, windows.FILE_ATTRIBUTE_OFFLINE); err != nil {
		t.Skipf("nao foi possivel marcar FILE_ATTRIBUTE_OFFLINE: %v", err)
	}
	t.Cleanup(func() {
		_ = windows.SetFileAttributes(p, windows.FILE_ATTRIBUTE_NORMAL)
	})
}

// travarExclusivo segura um handle exclusivo sobre o arquivo e CONFERE que ele
// barra uma leitura, devolvendo antes de qualquer assercao depender disso.
//
// Leitura E escrita: medido nesta maquina, um handle exclusivo que pede so
// GENERIC_READ nao barra o os.ReadFile. A conferencia existe porque uma trava
// que nao trava tornaria vazia toda prova de "nao abriu".
func travarExclusivo(t *testing.T, abs string) {
	t.Helper()
	p, err := windows.UTF16PtrFromString(vault.LongPath(abs))
	if err != nil {
		t.Fatal(err)
	}
	h, err := windows.CreateFile(p, windows.GENERIC_READ|windows.GENERIC_WRITE, 0, nil,
		windows.OPEN_EXISTING, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		t.Skipf("nao foi possivel abrir o arquivo em modo exclusivo: %v", err)
	}
	t.Cleanup(func() { _ = windows.CloseHandle(h) })

	if _, err := os.ReadFile(abs); err == nil {
		t.Fatal("o handle exclusivo nao barrou a leitura; a prova de 'nao abriu' seria vazia")
	}
}

func escreverNota(t *testing.T, root, rel, conteudo string) string {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(full, []byte(conteudo), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return full
}

// cofreComPlaceholder monta o cofre usado pelos testes de contagem: duas notas
// comuns e um placeholder de nuvem.
//
// O placeholder fica sob handle exclusivo durante o teste inteiro, e nao so nos
// testes que afirmam "nao abriu". Duas razoes, e a segunda e a que importa:
//
//  1. Nenhum destes testes tem direito de abrir o placeholder. Com a trava, uma
//     regressao que o abrisse falha aqui tambem, e nao so no teste dedicado.
//  2. Numa rodada de 2026-08-12, os dois testes que usam esta funcao reprovaram
//     na guarda de montagem enquanto o teste que segura a trava passou. Nao foi
//     reproduzido depois — 20 execucoes no mesmo processo e 25 em processos
//     separados, zero falhas — e a causa NAO foi identificada. A trava fecha a
//     janela em que qualquer outro processo poderia mexer no arquivo, e as
//     mensagens da guarda abaixo separam as causas para que a proxima
//     ocorrencia diga qual foi, em vez de acusar o atributo por padrao.
func cofreComPlaceholder(t *testing.T) (*vault.Vault, *index.Index) {
	t.Helper()
	root := t.TempDir()
	escreverNota(t, root, "comum.md", "# Comum\n\ncorpo de uma nota comum\n")
	escreverNota(t, root, "outra.md", "# Outra\n\noutro corpo qualquer\n")
	abs := escreverNota(t, root, "nuvem.md", conteudoDaNuvem)
	marcarSomenteNuvem(t, abs)
	travarExclusivo(t, abs)

	v, err := vault.New(root)
	if err != nil {
		t.Fatal(err)
	}
	idx := index.New()
	if err := idx.Build(context.Background(), v); err != nil {
		t.Fatalf("Build: %v", err)
	}

	// Guarda da montagem: sem o atributo pegando, todos os testes deste arquivo
	// passariam a exercitar uma nota comum.
	//
	// As duas causas sao separadas de proposito. Juntas numa condicao so, a
	// mensagem culpa o atributo quando o problema pode ser a nota nem estar no
	// indice — que e falha de outra coisa, e mandaria quem for depurar para o
	// lugar errado.
	n, ok := idx.Get("nuvem.md")
	if !ok {
		t.Fatalf("nuvem.md nao esta no indice de metadados (notas=%d, anexos=%d); "+
			"IsCloudOnly no disco = %v", idx.NoteCount(), idx.AssetCount(), vault.IsCloudOnly(abs))
	}
	if !n.CloudOnly {
		t.Fatalf("a nota nao ficou marcada CloudOnly; IsCloudOnly no disco = %v",
			vault.IsCloudOnly(abs))
	}
	return v, idx
}

// construirComoOBoot tokeniza o cofre exatamente como buildInvertedIndex faz:
// um Update por caminho de idx.NotePaths().
func construirComoOBoot(t *testing.T, v *vault.Vault, idx *index.Index) *search.Inverted {
	t.Helper()
	inv := search.NewInverted()
	for _, p := range idx.NotePaths() {
		if err := inv.Update(context.Background(), v, p); err != nil {
			t.Fatalf("inv.Update %s: %v", p, err)
		}
	}
	return inv
}

// TestUpdateNaoAbreNotaSomenteNuvem prova a regra nao negociavel pelo caminho
// que a violava: Inverted.Update chamava os.ReadFile sem consultar
// vault.IsCloudOnly, e o laco de boot o chama para TODA nota do cofre.
//
// A prova de que o arquivo nao foi aberto e mecanica: o teste segura um handle
// exclusivo e confere que ele barra uma leitura ANTES de afirmar. Com a trava
// provada, um Update que abrisse o arquivo devolveria erro de
// compartilhamento; ele devolvendo nil e a afirmacao.
func TestUpdateNaoAbreNotaSomenteNuvem(t *testing.T) {
	root := t.TempDir()
	abs := escreverNota(t, root, "nuvem.md", conteudoDaNuvem)
	marcarSomenteNuvem(t, abs)
	travarExclusivo(t, abs)

	v, err := vault.New(root)
	if err != nil {
		t.Fatal(err)
	}

	inv := search.NewInverted()
	if err := inv.Update(context.Background(), v, "nuvem.md"); err != nil {
		t.Fatalf("Update: %v — o arquivo foi ABERTO, e abrir um placeholder "+
			"dispara download sincrono", err)
	}

	// Entra coberta e vazia: pular sem registrar nao e a mesma coisa que
	// registrar sem conteudo, e a diferenca custa uma regravacao de cache por
	// partida (ver TestPlaceholderNaoFazOBootDeclararCacheParcial).
	if !inv.HasDoc("nuvem.md") {
		t.Fatal("HasDoc = false: a nota somente-nuvem nao entrou no indice, e o " +
			"boot a releria a cada retomada sem nunca conta-la como coberta")
	}
	if n := inv.DocLength("nuvem.md"); n != 0 {
		t.Fatalf("DocLength = %d, quer 0; o placeholder foi tokenizado", n)
	}
	for _, termo := range []string{"sesquipedaliano", "hidratado", "titulo", "nuvem"} {
		if posts := inv.Postings(termo); len(posts) != 0 {
			t.Fatalf("Postings(%q) = %v, quer vazio; o conteudo do disco entrou no indice", termo, posts)
		}
	}
}

// TestPlaceholderNaoFazOBootDeclararCacheParcial afirma sobre o que o BOOT
// compara, e nao sobre um campo isolado.
//
// invertedCacheState confronta hdr.NoteCount — que e inv.DocCount() no momento
// da gravacao — com idx.NoteCount(). Menor significa "cache parcial": o boot
// retoma a construcao e regrava o cache inteiro, em toda partida, para sempre.
// Conferir DocLength == 0 nao pega essa regressao; comparar as duas contagens
// que o boot compara, pega.
func TestPlaceholderNaoFazOBootDeclararCacheParcial(t *testing.T) {
	v, idx := cofreComPlaceholder(t)
	inv := construirComoOBoot(t, v, idx)

	if inv.DocCount() != idx.NoteCount() {
		t.Fatalf("DocCount do indice de busca = %d, notas no indice de metadados = %d; "+
			"o boot concluiria cache parcial e regravaria o cache inteiro a cada partida",
			inv.DocCount(), idx.NoteCount())
	}

	cacheDir := t.TempDir()
	if err := search.SaveInvertedCache(context.Background(), cacheDir, string(v.Root()), inv); err != nil {
		t.Fatalf("SaveInvertedCache: %v", err)
	}
	doCache, hdr, err := search.LoadInvertedCache(context.Background(), cacheDir, string(v.Root()))
	if err != nil {
		t.Fatalf("LoadInvertedCache: %v", err)
	}
	defer func() { _ = doCache.Close() }()

	// A comparacao literal de invertedCacheState: hdr.NoteCount >= noteCount
	// significa cache pronto; menor significa retomar.
	if hdr.NoteCount < idx.NoteCount() {
		t.Fatalf("cabecalho do cache declara %d notas, o indice de metadados enxerga %d; "+
			"invertedCacheState devolveria retomar=true e o cache seria regravado em todo boot",
			hdr.NoteCount, idx.NoteCount())
	}
}

// TestIdaEVoltaPeloCacheComPlaceholder confere que indice recem-construido e
// indice recarregado respondem IGUAL para o cofre com placeholder.
//
// Nao afirma valores: afirma igualdade entre as duas construcoes. E o formato
// que pegou a divergencia de DocLength entre construido e recarregado, quando o
// mesmo cofre ranqueava diferente conforme o servidor tivesse acabado de
// indexar ou de ler o cache.
func TestIdaEVoltaPeloCacheComPlaceholder(t *testing.T) {
	v, idx := cofreComPlaceholder(t)
	construido := construirComoOBoot(t, v, idx)

	cacheDir := t.TempDir()
	if err := search.SaveInvertedCache(context.Background(), cacheDir, string(v.Root()), construido); err != nil {
		t.Fatalf("SaveInvertedCache: %v", err)
	}
	recarregado, _, err := search.LoadInvertedCache(context.Background(), cacheDir, string(v.Root()))
	if err != nil {
		t.Fatalf("LoadInvertedCache: %v", err)
	}
	defer func() { _ = recarregado.Close() }()

	if got, want := recarregado.DocCount(), construido.DocCount(); got != want {
		t.Errorf("DocCount: construido=%d, recarregado=%d", want, got)
	}
	if got, want := recarregado.TermCount(), construido.TermCount(); got != want {
		t.Errorf("TermCount: construido=%d, recarregado=%d", want, got)
	}

	caminhosC := construido.DocPaths()
	caminhosR := recarregado.DocPaths()
	sort.Strings(caminhosC)
	sort.Strings(caminhosR)
	if !reflect.DeepEqual(caminhosC, caminhosR) {
		t.Fatalf("DocPaths divergem — construido=%v, recarregado=%v", caminhosC, caminhosR)
	}
	// Guarda da montagem: cofre vazio faria tudo acima passar sem comparar nada.
	if len(caminhosC) != 3 {
		t.Fatalf("o cofre de comparacao tem %d caminhos, quer 3", len(caminhosC))
	}

	for _, p := range caminhosC {
		if got, want := recarregado.HasDoc(p), construido.HasDoc(p); got != want {
			t.Errorf("%s: HasDoc construido=%v, recarregado=%v", p, want, got)
		}
		if got, want := recarregado.DocLength(p), construido.DocLength(p); got != want {
			t.Errorf("%s: DocLength construido=%d, recarregado=%d", p, want, got)
		}
	}

	// Termos das notas comuns, mais os do placeholder: os primeiros para provar
	// que a comparacao alcanca postings de verdade, os segundos para que uma
	// hidratacao em qualquer um dos dois lados apareca aqui.
	for _, termo := range []string{"comum", "corpo", "outra", "outro", "sesquipedaliano", "hidratado", "nuvem"} {
		pc := construido.Postings(termo)
		pr := recarregado.Postings(termo)
		sort.Slice(pc, func(i, j int) bool { return pc[i].Path < pc[j].Path })
		sort.Slice(pr, func(i, j int) bool { return pr[i].Path < pr[j].Path })
		if !reflect.DeepEqual(pc, pr) {
			t.Errorf("Postings(%q) divergem — construido=%v, recarregado=%v", termo, pc, pr)
		}
	}
}
