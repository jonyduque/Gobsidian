package search_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/jonyd/gobsidian/internal/search"
)

func TestInvertedRemoveLeavesNoEmptyPosting(t *testing.T) {
	ix := search.NewInverted()
	ix.Add("a.md", search.Analyze("prescricao intercorrente"))
	ix.Add("b.md", search.Analyze("prescricao civil"))

	ix.Remove("a.md")

	// "prescricao" continua, porque b.md ainda o tem.
	if got := ix.Postings("prescricao"); len(got) != 1 || got[0].Path != "b.md" {
		t.Errorf("postings de termo compartilhado = %+v, quer so b.md", got)
	}
	// "intercorrente" so existia em a.md: o TERMO tem de sair do dicionario,
	// nao ficar com lista vazia. Lista vazia ocupa espaco, entra na contagem
	// de termos, e nunca corresponde a nada.
	if ix.HasTerm("intercorrente") {
		t.Errorf("termo orfao continua no dicionario com posting list vazia")
	}
	if n := ix.TermCount(); n != 2 { // prescricao, civil
		t.Errorf("TermCount = %d, quer 2 — termo orfao esta sendo contado", n)
	}
}

func TestInvertedDualIndexingPostingsCount(t *testing.T) {
	ix := search.NewInverted()
	// "prescrições" gera raw: prescricoes, reduced: prescricao
	ix.Add("nota1.md", search.Analyze("prescrições"))

	// Consultar a forma crua ("prescricoes") deve retornar 1 posting
	pRaw := ix.Postings("prescricoes")
	if len(pRaw) != 1 || pRaw[0].Path != "nota1.md" {
		t.Errorf("Postings(prescricoes) = %+v, quer 1 posting em nota1.md", pRaw)
	}

	// Consultar a forma reduzida ("prescricao") deve retornar 1 posting
	pRed := ix.Postings("prescricao")
	if len(pRed) != 1 || pRed[0].Path != "nota1.md" {
		t.Errorf("Postings(prescricao) = %+v, quer 1 posting em nota1.md", pRed)
	}
}

func TestInvertedReindexSameNoteNoDuplicates(t *testing.T) {
	ix := search.NewInverted()
	tokens := search.Analyze("teste de reindexacao")

	ix.Add("doc.md", tokens)
	ix.Add("doc.md", tokens) // Reindexa a mesma nota

	p := ix.Postings("teste")
	if len(p) != 1 {
		t.Fatalf("len(Postings) = %d, quer 1", len(p))
	}
	if p[0].Frequency != 1 {
		t.Errorf("Frequency = %d, quer 1 (reindexacao duplica posicoes se nao limpar)", p[0].Frequency)
	}
}

func TestInvertedConcurrencyRace(t *testing.T) {
	ix := search.NewInverted()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var wg sync.WaitGroup

	// Writer goroutine (watcher thread)
	wg.Add(1)
	go func() {
		defer wg.Done()
		i := 0
		for {
			select {
			case <-ctx.Done():
				return
			default:
				path := fmt.Sprintf("note_%d.md", i%10)
				ix.Add(path, search.Analyze("prescricao civil usucapiao"))
				if i%3 == 0 {
					ix.Remove(path)
				}
				i++
			}
		}
	}()

	// Reader goroutines (MCP threads)
	for r := 0; r < 4; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					_ = ix.Postings("prescricao")
					_ = ix.HasTerm("usucapiao")
					_ = ix.TermCount()
					_ = ix.DocCount()
				}
			}
		}()
	}

	wg.Wait()

	if ix.DocCount() < 0 {
		t.Errorf("DocCount negativo apos concorrencia: %d", ix.DocCount())
	}
	if ix.TermCount() == 0 {
		t.Errorf("TermCount zerado apos escritas concorrentes")
	}
	for _, p := range ix.Postings("prescricao") {
		if p.Path == "" {
			t.Errorf("postings com caminho vazio encontrado: %+v", p)
		}
	}
}

func TestInvertedRemoveAndRecreateNoResidue(t *testing.T) {
	ix := search.NewInverted()

	ix.Add("antigo.md", search.Analyze("termo1 termo2"))
	ix.Remove("antigo.md")
	ix.Add("antigo.md", search.Analyze("termo3"))

	if ix.HasTerm("termo1") || ix.HasTerm("termo2") {
		t.Errorf("termos da nota antiga vazaram apos recriacao: t1=%v, t2=%v",
			ix.HasTerm("termo1"), ix.HasTerm("termo2"))
	}
	if !ix.HasTerm("termo3") {
		t.Errorf("termo3 da nova nota nao foi indexado")
	}
}

func TestAdotarDeMoveConteudo(t *testing.T) {
	origem := search.NewInverted()
	origem.Add("a.md", search.Analyze("prescricao intercorrente"))

	destino := search.NewInverted()
	if err := destino.AdotarDe(origem); err != nil {
		t.Fatalf("AdotarDe: %v", err)
	}

	if got := destino.DocCount(); got != 1 {
		t.Errorf("destino.DocCount() = %d, quer 1", got)
	}
	if !destino.HasTerm("prescricao") {
		t.Error("destino nao ficou com o termo adotado")
	}
	// A origem fica vazia e USAVEL: mapa nil aqui viraria panic num caminho
	// que so acontece por engano, que e justamente quando ninguem quer um
	// panic.
	if got := origem.DocCount(); got != 0 {
		t.Errorf("origem.DocCount() = %d, quer 0", got)
	}
	origem.Add("b.md", search.Analyze("outra"))
	if got := origem.DocCount(); got != 1 {
		t.Errorf("origem inutilizavel apos a adocao: DocCount() = %d", got)
	}
}

// TestAdotarDeRecusaIndiceComConteudo guarda a perda silenciosa que a adocao
// tardia do cache introduz.
//
// O cache passou a ser carregado fora do caminho de boot, entao o watcher e o
// servico ja tem o ponteiro do indice quando o conteudo chega. Se um Add tiver
// chegado antes, aplicar o cache por cima apagaria a edicao — e o sintoma seria
// uma nota editada durante o boot sumir da busca ate o proximo reinicio, sem
// erro nenhum. A ordem em runServe impede isso hoje; este teste garante que uma
// mudanca de ordem falhe alto em vez de perder dado.
func TestAdotarDeRecusaIndiceComConteudo(t *testing.T) {
	origem := search.NewInverted()
	origem.Add("do-cache.md", search.Analyze("prescricao"))

	destino := search.NewInverted()
	destino.Add("editada-durante-o-boot.md", search.Analyze("intercorrente"))

	err := destino.AdotarDe(origem)
	if !errors.Is(err, search.ErrIndiceNaoVazio) {
		t.Fatalf("AdotarDe = %v, quer ErrIndiceNaoVazio", err)
	}
	if !destino.HasTerm("intercorrente") {
		t.Error("a escrita anterior foi apagada pela adocao recusada")
	}
	if destino.HasTerm("prescricao") {
		t.Error("a adocao recusada aplicou conteudo mesmo assim")
	}
	if origem.DocCount() != 1 {
		t.Error("a adocao recusada esvaziou a origem")
	}
}
