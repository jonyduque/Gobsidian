package search

import (
	"bytes"
	"sort"
	"testing"
)

// viaCache grava o índice, relê e devolve o índice montado sobre a base
// achatada — o mesmo caminho que o servidor usa ao encontrar cache válido.
func viaCache(t *testing.T, origem *Inverted) *Inverted {
	t.Helper()
	termos, comp := origem.ExportForCache()
	var buf bytes.Buffer
	h := CacheHeader{
		FormatVersion:   CacheFormatVersion,
		ParserVersion:   CacheParserVersion,
		AnalyzerVersion: CacheAnalyzerVersion,
		NoteCount:       origem.DocCount(),
	}
	if err := escreveCache(&buf, h, termos, comp); err != nil {
		t.Fatalf("escreveCache: %v", err)
	}
	_, base, err := leCache(buf.Bytes())
	if err != nil {
		t.Fatalf("leCache: %v", err)
	}
	return newInvertedFromSoA(base)
}

// todosOsTermos devolve todo termo que o índice pode conhecer, vivo ou não.
// Usado só pelo comparador — a lista precisa ser ampla de propósito, para que
// um termo que deveria ter sumido seja consultado em vez de ignorado.
func todosOsTermos(ix *Inverted) []string {
	vistos := map[string]bool{}
	for termo := range ix.terms {
		vistos[termo] = true
	}
	if ix.base != nil {
		for _, termo := range ix.base.termos {
			vistos[termo] = true
		}
	}
	out := make([]string, 0, len(vistos))
	for termo := range vistos {
		out = append(out, termo)
	}
	sort.Strings(out)
	return out
}

// exigeEquivalentes compara dois índices por tudo que é observável de fora.
//
// Este é o teste central da estrutura em duas camadas. Ele não confere valores
// escritos à mão: confere que o índice montado sobre o cache mais o delta
// responde EXATAMENTE o que responde um índice construído do zero com o mesmo
// conteúdo. Valor escrito à mão eu erraria junto com o código; a construção do
// zero é o caminho que já estava certo antes de a base achatada existir.
func exigeEquivalentes(t *testing.T, quer, tem *Inverted, caminhos []string) {
	t.Helper()

	if a, b := quer.DocCount(), tem.DocCount(); a != b {
		t.Errorf("DocCount = %d, quer %d", b, a)
	}
	if a, b := quer.TermCount(), tem.TermCount(); a != b {
		t.Errorf("TermCount = %d, quer %d", b, a)
	}

	for _, path := range caminhos {
		if a, b := quer.HasDoc(path), tem.HasDoc(path); a != b {
			t.Errorf("HasDoc(%q) = %t, quer %t", path, b, a)
		}
		if a, b := quer.DocLength(path), tem.DocLength(path); a != b {
			t.Errorf("DocLength(%q) = %d, quer %d", path, b, a)
		}
	}

	termos := todosOsTermos(quer)
	for _, extra := range todosOsTermos(tem) {
		if sort.SearchStrings(termos, extra) >= len(termos) || termos[sort.SearchStrings(termos, extra)] != extra {
			termos = append(termos, extra)
			sort.Strings(termos)
		}
	}

	for _, termo := range termos {
		if a, b := quer.HasTerm(termo), tem.HasTerm(termo); a != b {
			t.Errorf("HasTerm(%q) = %t, quer %t", termo, b, a)
		}

		pa, pb := quer.Postings(termo), tem.Postings(termo)
		if len(pa) != len(pb) {
			t.Errorf("Postings(%q): %d postings, quer %d", termo, len(pb), len(pa))
			continue
		}
		if !sort.SliceIsSorted(pb, func(i, j int) bool { return pb[i].Path < pb[j].Path }) {
			t.Errorf("Postings(%q) fora de ordem", termo)
		}
		for i := range pa {
			if pa[i].Path != pb[i].Path {
				t.Errorf("Postings(%q)[%d].Path = %q, quer %q", termo, i, pb[i].Path, pa[i].Path)
				continue
			}
			if pa[i].Frequency != pb[i].Frequency {
				t.Errorf("Postings(%q)[%d].Frequency = %d, quer %d",
					termo, i, pb[i].Frequency, pa[i].Frequency)
			}
			if len(pa[i].Positions) != len(pb[i].Positions) {
				t.Errorf("Postings(%q)/%q: %d posições, quer %d",
					termo, pa[i].Path, len(pb[i].Positions), len(pa[i].Positions))
				continue
			}
			for k := range pa[i].Positions {
				if pa[i].Positions[k] != pb[i].Positions[k] {
					t.Errorf("Postings(%q)/%q[%d] = %+v, quer %+v",
						termo, pa[i].Path, k, pb[i].Positions[k], pa[i].Positions[k])
				}
			}
		}

		for _, path := range caminhos {
			qa, qb := quer.Positions(termo, path), tem.Positions(termo, path)
			if len(qa) != len(qb) {
				t.Errorf("Positions(%q, %q): %d, quer %d", termo, path, len(qb), len(qa))
				continue
			}
			for k := range qa {
				if qa[k] != qb[k] {
					t.Errorf("Positions(%q, %q)[%d] = %+v, quer %+v", termo, path, k, qb[k], qa[k])
				}
			}
		}
	}
}

// TestBaseMaisDeltaEquivaleAConstruidoDoZero é o teste central da estrutura em
// duas camadas.
//
// Cobre as quatro coisas que a sobreposição pode errar, e todas erram em
// silêncio:
//
//   - nota EDITADA: o base ainda tem a versão da partida. Se ela vazar, a busca
//     devolve o texto antigo de uma nota que o usuário acabou de mudar.
//   - nota REMOVIDA: idem, e ainda conta em DocCount.
//   - termo que só existia na nota removida: precisa sumir de HasTerm e de
//     TermCount, que na versão em mapa sumia sozinho ao apagar a entrada.
//   - nota NOVA: não está no base, e o caminho de leitura não pode concluir
//     "não está no base, logo não existe".
func TestBaseMaisDeltaEquivaleAConstruidoDoZero(t *testing.T) {
	const (
		textoA    = "prescricao intercorrente execucao fiscal"
		textoBOld = "exclusivo do arquivo b antigo"
		textoBNew = "prescricao revisada no arquivo b"
		textoC    = "termoexclusivodoc unico"
		textoD    = "civil processo prescricao"
		textoE    = "novissimo arquivo com prescricao"
	)
	caminhos := []string{"a.md", "b.md", "c.md", "d.md", "e.md", "vazia.md", "inexistente.md"}

	// Estado da partida, gravado no cache.
	naPartida := NewInverted()
	naPartida.Add("a.md", Analyze(textoA))
	naPartida.Add("b.md", Analyze(textoBOld))
	naPartida.Add("c.md", Analyze(textoC))
	naPartida.Add("d.md", Analyze(textoD))
	naPartida.Add("vazia.md", Analyze(""))

	comBase := viaCache(t, naPartida)
	if comBase.base == nil {
		t.Fatal("o indice de teste nao ficou com base achatada; o teste mediria o caminho errado")
	}

	// As mesmas mudanças nos dois.
	aplica := func(ix *Inverted) {
		ix.Add("b.md", Analyze(textoBNew)) // editada
		ix.Remove("c.md")                  // removida
		ix.Add("e.md", Analyze(textoE))    // nova
	}
	aplica(comBase)

	doZero := NewInverted()
	doZero.Add("a.md", Analyze(textoA))
	doZero.Add("d.md", Analyze(textoD))
	doZero.Add("vazia.md", Analyze(""))
	aplica(doZero)
	if doZero.base != nil {
		t.Fatal("o indice de referencia nao pode ter base; ele e o caminho de construcao do zero")
	}

	exigeEquivalentes(t, doZero, comBase, caminhos)

	// Sondas explícitas para o caso de o comparador um dia enfraquecer.
	if comBase.HasTerm("termoexclusivodoc") {
		t.Error("termo da nota removida sobreviveu: o base foi consultado para um caminho sombreado")
	}
	for _, p := range comBase.Postings("arquivo") {
		if p.Path == "b.md" && len(p.Positions) > 0 {
			for _, pos := range comBase.Positions("antigo", "b.md") {
				t.Errorf("conteudo antigo de b.md vazou do base: %+v", pos)
			}
		}
	}
}

// TestSombrearDuasVezesNaoDescontaDuasVezes: editar a mesma nota várias vezes é
// o caso NORMAL do watcher, e cada edição chama sombrearLocked. Descontar duas
// vezes zeraria postsVivasPorTermo de termos ainda vivos, e HasTerm passaria a
// responder "não existe" para palavras que existem.
func TestSombrearDuasVezesNaoDescontaDuasVezes(t *testing.T) {
	naPartida := NewInverted()
	naPartida.Add("a.md", Analyze("prescricao intercorrente"))
	naPartida.Add("b.md", Analyze("prescricao civil"))

	ix := viaCache(t, naPartida)
	for i := 0; i < 5; i++ {
		ix.Add("a.md", Analyze("prescricao intercorrente"))
	}

	if ix.sombraNoBase != 1 {
		t.Errorf("sombraNoBase = %d, quer 1 apos cinco edicoes do mesmo caminho", ix.sombraNoBase)
	}
	if !ix.HasTerm("civil") {
		t.Error("HasTerm(civil) = false; b.md nunca foi tocada")
	}
	if got := ix.DocCount(); got != 2 {
		t.Errorf("DocCount = %d, quer 2", got)
	}
	for t2 := range ix.base.termos {
		if ix.postsVivasPorTermo[t2] < 0 {
			t.Fatalf("postsVivasPorTermo[%d] = %d, negativo",
				t2, ix.postsVivasPorTermo[t2])
		}
	}
}

// TestExportForCacheMesclaAsDuasCamadas guarda o caminho de RETOMADA: cache
// parcial adotado como base, construção completa o resto, e a gravação seguinte
// precisa conter as duas coisas. Gravar só o delta perderia tudo que veio do
// cache, e o boot seguinte reconstruiria o cofre inteiro sem nada indicando o
// motivo.
func TestExportForCacheMesclaAsDuasCamadas(t *testing.T) {
	naPartida := NewInverted()
	naPartida.Add("a.md", Analyze("prescricao intercorrente"))
	naPartida.Add("b.md", Analyze("civil processo"))

	ix := viaCache(t, naPartida)
	ix.Add("c.md", Analyze("novo arquivo"))
	ix.Remove("b.md")

	segundaVolta := viaCache(t, ix)
	exigeEquivalentes(t, ix, segundaVolta, []string{"a.md", "b.md", "c.md"})

	if segundaVolta.HasTerm("civil") {
		t.Error("termo da nota removida foi regravado no cache")
	}
	if !segundaVolta.HasTerm("prescricao") {
		t.Error("termo que so existia no base sumiu ao regravar o cache")
	}
	if !segundaVolta.HasTerm("novo") {
		t.Error("termo que so existia no delta sumiu ao regravar o cache")
	}
}
