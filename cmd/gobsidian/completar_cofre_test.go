package main

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/jonyduque/Gobsidian/internal/instalar"
	"github.com/jonyduque/Gobsidian/internal/textos"
)

func TestCompletacaoDeVaultOfereceONomeSemAmbiguidade(t *testing.T) {
	aEstudo := filepath.Join("A", "Estudo")
	aOral := filepath.Join("A", "Oral")
	bOral := filepath.Join("B", "Oral")
	cofres := []instalar.Cofre{
		{Caminho: aEstudo, Aberto: true},
		{Caminho: aOral},
		{Caminho: bOral},
	}
	got := paresDeCofre(cofres)
	quer := []string{
		"Estudo", aEstudo,
		aEstudo, textos.CompletarCofreAberto,
		aOral, textos.CompletarCofre,
		bOral, textos.CompletarCofre,
	}
	if !slices.Equal(got, quer) {
		t.Errorf("pares = %q\nquer    %q", got, quer)
	}
}
