package search_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/search"
	"github.com/jonyd/gobsidian/internal/vault"
)

// Os testes deste arquivo usam a mesma sonda: reescrever o arquivo no disco SEM
// reconstruir o índice de metadados.
//
// Com a chave inalterada, o disco e o cache passam a discordar, e o texto
// devolvido diz qual dos dois respondeu — "veio do cache" vira uma afirmação
// observável em vez de uma inferência sobre tempo. É por isso que cada versão
// de conteúdo tem o MESMO comprimento e o termo no MESMO offset: se o texto ao
// redor mudasse de tamanho, a janela mudaria junto e o teste passaria a medir
// duas coisas.
const (
	conteudoA = "alvo aaaaaaaaaaaaaaaaaaaa"
	conteudoB = "alvo bbbbbbbbbbbbbbbbbbbb"
)

// cofreComNota monta cofre, índice e invertido para uma nota só.
func cofreComNota(t *testing.T, nome, conteudo string) (*vault.Vault, *index.Index, *search.Inverted, string) {
	t.Helper()
	root := t.TempDir()
	caminho := filepath.Join(root, nome)
	if err := os.WriteFile(caminho, []byte(conteudo), 0o644); err != nil {
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
	ix := search.NewInverted()
	ix.Add(nome, search.Analyze(conteudo))
	return v, idx, ix, caminho
}

func trechoCom(t *testing.T, v *vault.Vault, ix *search.Inverted, idx *index.Index, nome, termo string, c *search.SnippetCache) search.Snippet {
	t.Helper()
	s, err := search.GenerateSnippet(context.Background(), v, ix, idx, nome, []string{termo}, 240, c)
	if err != nil {
		t.Fatalf("GenerateSnippet: %v", err)
	}
	return s
}

func TestSnippetCacheAcertoNaoTocaNoDisco(t *testing.T) {
	v, idx, ix, caminho := cofreComNota(t, "nota.md", conteudoA)
	cache := search.NewSnippetCache(16)

	primeiro := trechoCom(t, v, ix, idx, "nota.md", "alvo", cache)
	if !strings.Contains(primeiro.Text, "aaaa") {
		t.Fatalf("primeiro trecho = %q; nao recortou o conteudo original", primeiro.Text)
	}
	if cache.Len() != 1 {
		t.Fatalf("cache.Len() = %d, quer 1; o trecho nao foi guardado", cache.Len())
	}

	// Disco muda, índice não: a chave continua a mesma, e só o cache pode
	// responder com o conteúdo antigo.
	if err := os.WriteFile(caminho, []byte(conteudoB), 0o644); err != nil {
		t.Fatal(err)
	}

	segundo := trechoCom(t, v, ix, idx, "nota.md", "alvo", cache)
	if segundo.Text != primeiro.Text {
		t.Fatalf("segundo trecho = %q, quer %q; a chave nao mudou, logo houve leitura de disco "+
			"onde devia haver acerto de cache", segundo.Text, primeiro.Text)
	}
}

// TestSnippetCacheNaoServeTrechoDeConteudoAntigo é o teste da regra D1: o hash
// da nota faz parte da chave.
//
// A montagem isola o hash de propósito. Caminho, termo, offsets da ocorrência e
// maxChars são idênticos entre as duas chamadas — as duas versões do arquivo
// têm o mesmo comprimento e o termo no mesmo lugar. O ÚNICO campo da chave que
// muda é o hash. Tire o hash da chave e a segunda chamada devolve o trecho da
// primeira, que é o modo de falha que este cache poderia introduzir.
func TestSnippetCacheNaoServeTrechoDeConteudoAntigo(t *testing.T) {
	v, idxAntigo, ix, caminho := cofreComNota(t, "nota.md", conteudoA)
	cache := search.NewSnippetCache(16)

	antes := trechoCom(t, v, ix, idxAntigo, "nota.md", "alvo", cache)
	if !strings.Contains(antes.Text, "aaaa") {
		t.Fatalf("trecho antes da edicao = %q; nao recortou o conteudo original", antes.Text)
	}

	if err := os.WriteFile(caminho, []byte(conteudoB), 0o644); err != nil {
		t.Fatal(err)
	}
	idxNovo := index.New()
	if err := idxNovo.Build(context.Background(), v); err != nil {
		t.Fatal(err)
	}

	// Guarda da montagem. Sem hash diferente, este teste não afirma nada sobre
	// a chave — passaria com qualquer implementação.
	nAntigo, ok := idxAntigo.Get("nota.md")
	if !ok {
		t.Fatal("nota sumiu do indice antigo")
	}
	nNovo, ok := idxNovo.Get("nota.md")
	if !ok {
		t.Fatal("nota sumiu do indice novo")
	}
	if nAntigo.Hash == nNovo.Hash {
		t.Fatalf("os dois indices tem o mesmo hash (%d); a edicao nao chegou ao indice "+
			"e o teste mediria a montagem, nao a chave", nAntigo.Hash)
	}

	depois := trechoCom(t, v, ix, idxNovo, "nota.md", "alvo", cache)

	if strings.Contains(depois.Text, "aaaa") {
		t.Fatalf("trecho obsoleto servido do cache: %q ainda traz o conteudo anterior a edicao; "+
			"o hash da nota nao esta na chave do cache", depois.Text)
	}
	if !strings.Contains(depois.Text, "bbbb") {
		t.Fatalf("trecho depois da edicao = %q, quer o conteudo novo", depois.Text)
	}
	// Os offsets iguais confirmam que o hash foi a única diferença entre as duas
	// chaves; offsets diferentes tornariam a passagem acima acidental.
	if antes.HighlightStart != depois.HighlightStart || antes.HighlightEnd != depois.HighlightEnd {
		t.Fatalf("destaque mudou de %d..%d para %d..%d; a chave mudou por outro campo "+
			"alem do hash e o teste nao isola a regra",
			antes.HighlightStart, antes.HighlightEnd, depois.HighlightStart, depois.HighlightEnd)
	}
}

// TestSnippetCacheNaoGuardaNotaSemHash cobre a regra D2 nos seus dois casos:
// índice ausente e nota que não está nele. Sem hash não existe chave de
// invalidação, e uma entrada assim sobreviveria a qualquer edição.
func TestSnippetCacheNaoGuardaNotaSemHash(t *testing.T) {
	t.Run("indice ausente", func(t *testing.T) {
		v, _, ix, _ := cofreComNota(t, "nota.md", conteudoA)
		cache := search.NewSnippetCache(16)

		trecho := trechoCom(t, v, ix, nil, "nota.md", "alvo", cache)
		if !strings.Contains(trecho.Text, "aaaa") {
			t.Fatalf("trecho = %q; sem indice o recorte ainda tem de sair do disco", trecho.Text)
		}
		if cache.Len() != 0 {
			t.Fatalf("cache.Len() = %d, quer 0; nota sem hash foi guardada", cache.Len())
		}
	})

	t.Run("nota fora do indice", func(t *testing.T) {
		v, idx, ix, caminho := cofreComNota(t, "nota.md", conteudoA)
		cache := search.NewSnippetCache(16)

		// Nota que nasceu depois do Build: existe no disco e no invertido, não
		// no índice de metadados.
		nova := filepath.Join(filepath.Dir(caminho), "nova.md")
		if err := os.WriteFile(nova, []byte(conteudoB), 0o644); err != nil {
			t.Fatal(err)
		}
		ix.Add("nova.md", search.Analyze(conteudoB))
		if _, ok := idx.Get("nova.md"); ok {
			t.Fatal("a nota nova entrou no indice; a montagem nao cobre o caso pretendido")
		}

		trecho := trechoCom(t, v, ix, idx, "nova.md", "alvo", cache)
		if !strings.Contains(trecho.Text, "bbbb") {
			t.Fatalf("trecho = %q; a nota fora do indice ainda tem de ser recortada", trecho.Text)
		}
		if cache.Len() != 0 {
			t.Fatalf("cache.Len() = %d, quer 0; nota fora do indice foi guardada", cache.Len())
		}
	})
}

func TestSnippetCacheNaoGuardaQuandoNenhumTermoOcorre(t *testing.T) {
	v, idx, ix, _ := cofreComNota(t, "nota.md", conteudoA)
	cache := search.NewSnippetCache(16)

	trecho := trechoCom(t, v, ix, idx, "nota.md", "inexistente", cache)
	if trecho.Text != "" {
		t.Fatalf("trecho = %q, quer vazio", trecho.Text)
	}
	if cache.Len() != 0 {
		t.Fatalf("cache.Len() = %d, quer 0; trecho vazio despejaria trecho util", cache.Len())
	}
}

func TestSnippetCacheDespejaMenosUsadoRecentemente(t *testing.T) {
	root := t.TempDir()
	nomes := []string{"a.md", "b.md", "c.md"}
	for _, n := range nomes {
		if err := os.WriteFile(filepath.Join(root, n), []byte(conteudoA), 0o644); err != nil {
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
	ix := search.NewInverted()
	for _, n := range nomes {
		ix.Add(n, search.Analyze(conteudoA))
	}

	cache := search.NewSnippetCache(2)
	for _, n := range nomes {
		trechoCom(t, v, ix, idx, n, "alvo", cache)
	}
	if cache.Len() != 2 {
		t.Fatalf("cache.Len() = %d, quer 2; o teto nao foi respeitado", cache.Len())
	}

	// b.md entrou depois de a.md e ainda deve estar guardado: o disco muda, a
	// chave não, e o texto antigo só pode vir do cache.
	if err := os.WriteFile(filepath.Join(root, "b.md"), []byte(conteudoB), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := trechoCom(t, v, ix, idx, "b.md", "alvo", cache); !strings.Contains(got.Text, "aaaa") {
		t.Fatalf("b.md = %q; foi relido do disco, logo a entrada mais recente foi despejada", got.Text)
	}

	// a.md foi o primeiro a entrar num cache de duas entradas: tem de ter sido
	// despejado, e a releitura traz o conteúdo novo.
	if err := os.WriteFile(filepath.Join(root, "a.md"), []byte(conteudoB), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := trechoCom(t, v, ix, idx, "a.md", "alvo", cache); !strings.Contains(got.Text, "bbbb") {
		t.Fatalf("a.md = %q; a entrada mais velha sobreviveu ao teto de 2", got.Text)
	}
}

func TestSnippetCacheNiloEDesligado(t *testing.T) {
	v, idx, ix, caminho := cofreComNota(t, "nota.md", conteudoA)

	if c := search.NewSnippetCache(0); c != nil {
		t.Fatalf("NewSnippetCache(0) = %v, quer nil; teto zero e a forma de desligar", c)
	}

	primeiro := trechoCom(t, v, ix, idx, "nota.md", "alvo", nil)
	if !strings.Contains(primeiro.Text, "aaaa") {
		t.Fatalf("trecho = %q; o caminho sem cache tem de recortar do disco", primeiro.Text)
	}

	if err := os.WriteFile(caminho, []byte(conteudoB), 0o644); err != nil {
		t.Fatal(err)
	}
	segundo := trechoCom(t, v, ix, idx, "nota.md", "alvo", nil)
	if !strings.Contains(segundo.Text, "bbbb") {
		t.Fatalf("trecho = %q; com cache nil a segunda chamada tem de reler o disco", segundo.Text)
	}
}
