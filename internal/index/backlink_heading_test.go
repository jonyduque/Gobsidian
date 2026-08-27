package index_test

import (
	"testing"
)

// notaComSecoes põe as referências em seções DIFERENTES, o que é o ponto: o
// heading só informa alguma coisa se ele variar com a posição do link.
const notaComSecoes = "# Origem\n\n" +
	"Introducao sem referencia nenhuma.\n\n" +
	"## Prescricao\n\n" +
	"O prazo foi contado a partir da citacao, conforme [[Alvo]].\n\n" +
	"## Honorarios\n\n" +
	"A fixacao seguiu o criterio de [[Alvo]] para a base de calculo.\n"

// TestBacklinkTrazHeadingDaSecao fixa o pedido do dono: além do contexto curto,
// saber em QUE seção da nota de origem a referência está.
//
// O heading custa zero byte de cache — `Note.Headings` já é persistido e
// `Link.Start` vem da mesma origem de offset. Se alguém "otimizar" isso
// guardando o título por link, este teste continua passando e o cache incha; o
// que protege contra isso é o teste de tamanho, não este.
func TestBacklinkTrazHeadingDaSecao(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "Origem.md", notaComSecoes)
	writeFile(t, root, "Alvo.md", "# Alvo\n")

	bls := montarIndice(t, root).Backlinks("Alvo.md")
	if len(bls) != 2 {
		t.Fatalf("backlinks = %d, queria 2", len(bls))
	}

	vistos := map[string]bool{}
	for _, bl := range bls {
		if bl.Heading == "" {
			t.Errorf("backlink de %s sem heading; contexto=%q", bl.From, bl.Context)
		}
		vistos[bl.Heading] = true
	}
	for _, quero := range []string{"Prescricao", "Honorarios"} {
		if !vistos[quero] {
			t.Errorf("nenhum backlink na secao %q; vistos=%v", quero, vistos)
		}
	}

	// Dois headings distintos, não o mesmo repetido: heading igual para os dois
	// significaria que o cálculo ignora a posição do link — pegando sempre o
	// primeiro, ou sempre o último.
	if len(vistos) != 2 {
		t.Errorf("os dois backlinks caíram no MESMO heading %v: o calculo ignora a posicao do link", vistos)
	}
}

// TestHeadingVazioAntesDoPrimeiroTitulo fixa a borda honesta: referência que não
// está sob heading nenhum devolve "", e não o título de uma seção que vem
// DEPOIS dela — que seria pior que vazio, porque pareceria informação.
func TestHeadingVazioAntesDoPrimeiroTitulo(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "Origem.md",
		"Uma linha solta citando [[Alvo]] antes de qualquer titulo.\n\n## Depois\n\nTexto.\n")
	writeFile(t, root, "Alvo.md", "# Alvo\n")

	bls := montarIndice(t, root).Backlinks("Alvo.md")
	if len(bls) != 1 {
		t.Fatalf("backlinks = %d, queria 1", len(bls))
	}
	if bls[0].Heading != "" {
		t.Errorf("Heading = %q, queria vazio: a referencia esta ANTES do primeiro titulo, "+
			"e devolver %q seria apontar uma secao que vem depois dela", bls[0].Heading, bls[0].Heading)
	}
}

// TestContextoCurtoNaoEstouraOOrcamento fixa o corte para 40 bytes de cada lado.
// Sem isto, contextoBytes voltaria a 80 numa edição futura sem nada reprovando —
// e foi 80 que atravessou o teto do RNF-02 no cofre local.
func TestContextoCurtoNaoEstouraOOrcamento(t *testing.T) {
	root := t.TempDir()
	corpo := "# T\n\n" + longo(300) + " [[Alvo]] " + longo(300) + "\n"
	writeFile(t, root, "Origem.md", corpo)
	writeFile(t, root, "Alvo.md", "# Alvo\n")

	bls := montarIndice(t, root).Backlinks("Alvo.md")
	if len(bls) != 1 {
		t.Fatalf("backlinks = %d, queria 1", len(bls))
	}
	// 40 de cada lado + o proprio "[[Alvo]]" (8) + folga para o alinhamento de
	// runa. Um teto frouxo ainda reprova o 80, que era 160+span.
	const teto = 40 + 40 + 8 + 8
	if n := len(bls[0].Context); n > teto {
		t.Errorf("contexto tem %d bytes, teto %d: contextoBytes voltou a crescer\ncontexto=%q",
			n, teto, bls[0].Context)
	}
	if len(bls[0].Context) == 0 {
		t.Error("contexto vazio: o corte comeu tudo")
	}
}

func longo(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = 'x'
		if i%8 == 7 {
			b[i] = ' '
		}
	}
	return string(b)
}
