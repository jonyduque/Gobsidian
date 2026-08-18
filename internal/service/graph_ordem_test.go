package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"testing"
)

// cofreInterligado monta 12 notas com ligacoes cruzadas. O numero importa: com
// duas ou tres, a iteracao de mapa do Go coincide com frequencia alta e o teste
// passa com o codigo desordenado.
func cofreInterligado(t *testing.T) *Service {
	t.Helper()
	root := t.TempDir()
	for i := 0; i < 12; i++ {
		var corpo bytes.Buffer
		fmt.Fprintf(&corpo, "# Nota %02d\n\n", i)
		for j := 0; j < 12; j++ {
			if j != i {
				fmt.Fprintf(&corpo, "liga para [[n%02d]]\n", j)
			}
		}
		writeFile(t, root, fmt.Sprintf("n%02d.md", i), corpo.String())
	}
	return newTestService(t, root)
}

// TestLinkGraphOrdemEstavel roda a MESMA consulta 50 vezes e exige resposta byte
// a byte identica.
//
// Uma chamada so nao pega ordem de mapa. 50 voltas e o que torna a falha
// confiavel; com menos, o teste passa as vezes — que e a pior especie.
func TestLinkGraphOrdemEstavel(t *testing.T) {
	svc := cofreInterligado(t)

	consulta := GraphRequest{Path: "n00.md", Direction: "both", Depth: 2, Limit: 100}

	serializa := func(t *testing.T) []byte {
		t.Helper()
		res, err := svc.LinkGraph(context.Background(), consulta)
		if err != nil {
			t.Fatalf("LinkGraph: %v", err)
		}
		if len(res.Nodes) < 5 || len(res.Edges) < 5 {
			t.Fatalf("grafo pequeno demais para o teste significar algo: %d nos, %d arestas",
				len(res.Nodes), len(res.Edges))
		}
		bruto, err := json.Marshal(res)
		if err != nil {
			t.Fatalf("Marshal: %v", err)
		}
		return bruto
	}

	primeira := serializa(t)
	for i := 1; i <= 50; i++ {
		outra := serializa(t)
		if !bytes.Equal(primeira, outra) {
			t.Fatalf("volta %d devolveu ordem diferente:\n%s\n%s", i, primeira, outra)
		}
	}
}

// TestLinkGraphLimitTemTeto guarda a segunda metade do item 21: `depth` tem teto
// (3) e `limit` nao tem nenhum.
func TestLinkGraphLimitTemTeto(t *testing.T) {
	svc := cofreInterligado(t)

	res, err := svc.LinkGraph(context.Background(), GraphRequest{
		Path: "n00.md", Direction: "both", Depth: 3, Limit: 1000000,
	})
	if err != nil {
		t.Fatalf("LinkGraph: %v", err)
	}
	// O cofre tem 12 notas, entao o teto nao pode ser medido pelo tamanho do
	// resultado. A asserção e sobre o VALOR EFETIVO anunciado na resposta.
	if res.LimitEfetivo == 0 {
		t.Fatal("a resposta nao diz qual limit foi usado")
	}
	if res.LimitEfetivo >= 1000000 {
		t.Fatalf("LimitEfetivo = %d: limit continua sem teto", res.LimitEfetivo)
	}
}
