package writer_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/jonyduque/Gobsidian/internal/writer"
)

func TestDiff_CRLFIdenticalProducesEmpty(t *testing.T) {
	crlf := "# Titulo\r\n\r\nlinha1\r\nlinha2\r\n"
	lf := "# Titulo\n\nlinha1\nlinha2\n"

	diff1 := writer.UnifiedDiff("a.md", "b.md", crlf, crlf, 3)
	if diff1 != "" {
		t.Fatalf("CRLF contra ele mesmo produziu diff nao-vazio:\n%s", diff1)
	}

	diff2 := writer.UnifiedDiff("a.md", "b.md", crlf, lf, 3)
	if diff2 != "" {
		t.Fatalf("CRLF contra LF equivalente produziu diff nao-vazio:\n%s", diff2)
	}
}

func TestDiff_InsertionAndDeletionMiddle(t *testing.T) {
	a := "linha1\nlinha2\nlinha3\n"
	b := "linha1\nlinha_modificada\nlinha3\nlinha4\n"

	diff := writer.UnifiedDiff("a.md", "b.md", a, b, 3)
	if diff == "" {
		t.Fatal("esperava diff nao-vazio")
	}

	if !strings.Contains(diff, "-linha2") {
		t.Errorf("diff nao contem delecao de linha2:\n%s", diff)
	}
	if !strings.Contains(diff, "+linha_modificada") {
		t.Errorf("diff nao contem insercao de linha_modificada:\n%s", diff)
	}
	if !strings.Contains(diff, "+linha4") {
		t.Errorf("diff nao contem insercao de linha4:\n%s", diff)
	}
}

func TestDiff_EmptyAgainstNonEmpty(t *testing.T) {
	a := ""
	b := "linha1\nlinha2\n"

	diff := writer.UnifiedDiff("a.md", "b.md", a, b, 3)
	if !strings.Contains(diff, "+linha1") || !strings.Contains(diff, "+linha2") {
		t.Fatalf("diff de vazio para nao-vazio incorreto:\n%s", diff)
	}

	diffReverse := writer.UnifiedDiff("b.md", "a.md", b, a, 3)
	if !strings.Contains(diffReverse, "-linha1") || !strings.Contains(diffReverse, "-linha2") {
		t.Fatalf("diff de nao-vazio para vazio incorreto:\n%s", diffReverse)
	}
}

func TestDiff_NoTrailingNewlineLine(t *testing.T) {
	a := "linha1\nlinha2"
	b := "linha1\nlinha2_modificada"

	diff := writer.UnifiedDiff("a.md", "b.md", a, b, 3)
	if !strings.Contains(diff, "-linha2") || !strings.Contains(diff, "+linha2_modificada") {
		t.Fatalf("diff de linha final sem newline falhou:\n%s", diff)
	}
}

func TestDiff_AllocationsAndPerformance(t *testing.T) {
	var aLines, bLines []string
	for i := 0; i < 1000; i++ {
		line := fmt.Sprintf("linha de conteudo numero %d no arquivo de teste", i)
		aLines = append(aLines, line)
		if i%100 == 0 {
			bLines = append(bLines, line+" (modificada)")
		} else {
			bLines = append(bLines, line)
		}
	}
	aText := strings.Join(aLines, "\n")
	bText := strings.Join(bLines, "\n")

	allocs := testing.AllocsPerRun(10, func() {
		_ = writer.UnifiedDiff("a.md", "b.md", aText, bText, 3)
	})

	t.Logf("Medicao de Alocacoes: %.0f alocacoes para diff de 1000 linhas com 10 alteracoes (orcamento %d)",
		allocs, orcamentoAlocacoesDiff)

	// Orcamento, e nao so `Logf`. Ate 2026-09-04 este teste media e imprimia:
	// uma regressao que triplicasse as alocacoes saia no `-v` e passava verde.
	//
	// Ver orcamentoAlocacoesDiff para as catorze medicoes que produziram o
	// numero, e por que o detector de corrida obriga a contar as duas series.
	if allocs > orcamentoAlocacoesDiff {
		t.Errorf("UnifiedDiff fez %.0f alocacoes, orcamento %d (maximo medido em 2026-09-04: 273, x1,5)\n"+
			"um diff de 1000 linhas com 10 alteracoes nao deveria alocar mais do que isso",
			allocs, orcamentoAlocacoesDiff)
	}
}

// orcamentoAlocacoesDiff e o teto de alocacoes de UnifiedDiff no cenario de
// TestDiff_AllocationsAndPerformance: 1000 linhas com 10 alteracoes.
//
// O numero vem de MEDICAO, nao de estimativa. Catorze rodadas em 2026-09-04, na
// maquina NB-JONY (Intel i7-10750H, 12 nucleos, go1.26.5 windows/amd64):
//
//	go test ./internal/writer/ -run TestDiff_Alloc... -count=7
//	    259, 259, 259, 259, 259, 259, 259
//	go test -race ./internal/writer/ -run TestDiff_Alloc... -count=7
//	    273, 273, 272, 270, 272, 271, 273
//
// As DUAS series importam porque `verify.ps1` roda a suite com `-race`, e o
// detector aloca por conta propria: e a serie de 273 que o gate cobra. Um
// orcamento tirado so da serie sem `-race` seria um numero que ninguem mede.
//
// Orcamento = 273 (o maximo das catorze) x 1,5 arredondado para cima = 410. A
// folga de 50% absorve mudanca de versao do runtime e de biblioteca; ela nao
// absorve regressao de algoritmo, que e o que este teste guarda.
//
// `AllocsPerRun` conta alocacoes, nao tempo, e por isso o numero nao varia com
// a carga da maquina — as sete rodadas sem `-race` sairam identicas.
const orcamentoAlocacoesDiff = 410
