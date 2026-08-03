package main

import (
	"errors"
	"testing"

	"github.com/jonyd/gobsidian/internal/search"
)

// TestInvertedCacheState fixa a regra que impede um cache PARCIAL de passar por
// completo.
//
// A construcao do indice de busca passou a rodar em segundo plano e a gravar
// parciais, para que uma morte no meio nao jogue fora minutos de trabalho.
// LoadInvertedCache nao confere cobertura — so versao de formato, de parser, de
// analisador e o caminho do cofre. Sem esta regra, o parcial voltaria do disco
// marcado como pronto e a busca serviria menos notas do que existem, sem nada
// no retorno dizendo isso.
func TestInvertedCacheState(t *testing.T) {
	casos := []struct {
		nome        string
		hdr         *search.CacheHeader
		err         error
		noteCount   int
		querPronta  bool
		querRetomar bool
	}{
		{
			nome:      "sem cache no disco",
			hdr:       nil,
			err:       search.ErrCacheNotFound,
			noteCount: 100,
		},
		{
			nome:      "cache corrompido",
			hdr:       nil,
			err:       search.ErrCacheCorrupted,
			noteCount: 100,
		},
		{
			nome:      "versao divergente traz cabecalho, mas nao serve",
			hdr:       &search.CacheHeader{NoteCount: 100},
			err:       search.ErrCacheVersionMismatch,
			noteCount: 100,
		},
		{
			nome:       "cobertura exata",
			hdr:        &search.CacheHeader{NoteCount: 100},
			noteCount:  100,
			querPronta: true,
		},
		{
			nome:        "parcial: menos notas no cache do que no cofre",
			hdr:         &search.CacheHeader{NoteCount: 399},
			noteCount:   3153,
			querRetomar: true,
		},
		{
			nome:        "parcial por uma nota so",
			hdr:         &search.CacheHeader{NoteCount: 99},
			noteCount:   100,
			querRetomar: true,
		},
		{
			// Notas apagadas do cofre deixam entradas velhas no cache. Ele cobre
			// tudo o que existe hoje, e as sobras nao vazam: a busca so devolve
			// o que o indice de metadados confirma.
			nome:       "cache maior que o cofre, apos apagar notas",
			hdr:        &search.CacheHeader{NoteCount: 3153},
			noteCount:  3000,
			querPronta: true,
		},
		{
			nome:       "cofre vazio",
			hdr:        &search.CacheHeader{NoteCount: 0},
			noteCount:  0,
			querPronta: true,
		},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			pronta, retomar := invertedCacheState(c.hdr, c.err, c.noteCount)
			if pronta != c.querPronta {
				t.Errorf("pronta = %v, quer %v", pronta, c.querPronta)
			}
			if retomar != c.querRetomar {
				t.Errorf("retomar = %v, quer %v", retomar, c.querRetomar)
			}
			if pronta && retomar {
				t.Error("pronta e retomar ao mesmo tempo: os dois estados sao exclusivos")
			}
		})
	}
}

// TestInvertedCacheStateErroNuncaEPronta e a guarda contra a regressao que
// custa mais caro aqui: qualquer erro de leitura tem de resultar em
// reconstrucao, nunca em "pronta".
func TestInvertedCacheStateErroNuncaEPronta(t *testing.T) {
	erros := []error{
		search.ErrCacheNotFound,
		search.ErrCacheCorrupted,
		search.ErrCacheVersionMismatch,
		errors.New("erro qualquer de I/O"),
	}
	for _, e := range erros {
		// Cabecalho generoso de proposito: mesmo dizendo cobrir o cofre inteiro,
		// um cache que falhou ao carregar nao pode ser dado como pronto.
		pronta, _ := invertedCacheState(&search.CacheHeader{NoteCount: 999999}, e, 100)
		if pronta {
			t.Errorf("erro %v resultou em pronta = true", e)
		}
	}
}
