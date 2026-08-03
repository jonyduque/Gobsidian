package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jonyd/gobsidian/internal/service"
)

// TestSearchComIndiceEmConstrucaoNaoMenteZeroResultados fixa o contrato que
// torna a construção assíncrona do índice invertido segura.
//
// O boot do servidor tokenizava o cofre inteiro ANTES de anunciar as tools.
// Num cofre real de 109 MB isso levou 220 s, e o Claude Code desiste do
// handshake em 30 s — matando o processo antes de o cache ser gravado, de modo
// que a tentativa seguinte recomeçava do zero. Toda tentativa falhava pelo
// mesmo motivo, para sempre.
//
// A saída é construir em segundo plano. O perigo dessa saída é este teste: com
// o índice pela metade, `vault_search` acharia menos notas do que existem e
// devolveria isso como resposta legítima. "Nenhum resultado" e "ainda não sei"
// são respostas diferentes, e o modelo do outro lado não tem como distinguir
// uma da outra se as duas chegarem como uma lista vazia.
func TestSearchComIndiceEmConstrucaoNaoMenteZeroResultados(t *testing.T) {
	svc, _, _, inv := createSearchService(t, map[string]string{
		"a.md": "# Prescrição\n\nTexto sobre prescrição.\n",
	})

	// Com o índice pronto, a consulta acha a nota. É a linha de base: sem ela,
	// um erro na montagem do cenário passaria por "índice em construção".
	res, err := svc.Search(context.Background(), service.SearchOptions{Query: "prescricao"})
	if err != nil {
		t.Fatalf("linha de base: %v", err)
	}
	if len(res.Results) != 1 {
		t.Fatalf("linha de base: %d resultados, quer 1", len(res.Results))
	}

	// Agora o mesmo índice, declarado em construção.
	inv.MarkBuilding()

	_, err = svc.Search(context.Background(), service.SearchOptions{Query: "prescricao"})
	if err == nil {
		t.Fatal("busca com índice em construção devolveu err == nil; " +
			"uma lista vazia é indistinguível de 'não achei nada'")
	}

	var serr *service.Error
	if !errors.As(err, &serr) {
		t.Fatalf("erro = %T (%v), quer *service.Error", err, err)
	}
	if serr.Code != service.CodeIndexBuilding {
		t.Errorf("Code = %q, quer %q", serr.Code, service.CodeIndexBuilding)
	}

	// Voltando a pronto, a busca funciona de novo: o estado é reversível e não
	// deixa o servidor travado.
	inv.MarkReady()
	res, err = svc.Search(context.Background(), service.SearchOptions{Query: "prescricao"})
	if err != nil {
		t.Fatalf("depois de MarkReady: %v", err)
	}
	if len(res.Results) != 1 {
		t.Fatalf("depois de MarkReady: %d resultados, quer 1", len(res.Results))
	}
}
