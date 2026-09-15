package instalar

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/jonyduque/Gobsidian/internal/hosts"
)

// TestEntradasParaCofresNuncaRepeteChave e H1: os dois pares medidos em
// 2026-09-14 davam a mesma chave, e o segundo cofre sobrescrevia o primeiro no
// config do host.
func TestEntradasParaCofresNuncaRepeteChave(t *testing.T) {
	pares := [][]string{
		{"C:/A/Revis" + string(rune(0x00E3)) + "o", "C:/B/Revisao"},
		{"C:/A/Estudo", "C:/B/Estudo"},
	}
	for _, par := range pares {
		entradas := EntradasParaCofres("gobsidian.exe", par, false)
		if len(entradas) != 2 {
			t.Fatalf("EntradasParaCofres(%v) = %d entradas", par, len(entradas))
		}
		if entradas[0].Chave == entradas[1].Chave {
			t.Fatalf("EntradasParaCofres(%v): as duas entradas com a chave %q", par, entradas[0].Chave)
		}
		for _, e := range entradas {
			if !hosts.EhNossa(e.Chave) {
				t.Errorf("chave %q deixou de ser reconhecida como nossa", e.Chave)
			}
		}

		// E o config do host guarda as duas.
		arquivo := filepath.Join(t.TempDir(), "config.json")
		if err := hosts.FundirVarias(arquivo, entradas); err != nil {
			t.Fatalf("FundirVarias: %v", err)
		}
		lidas, err := hosts.LerEntradas(arquivo)
		if err != nil {
			t.Fatalf("LerEntradas: %v", err)
		}
		if len(lidas) != 2 {
			t.Fatalf("o config do host ficou com %d entradas para %v, esperado 2", len(lidas), par)
		}
	}

	// O inverso: nomes diferentes nao ganham sufixo.
	entradas := EntradasParaCofres("gobsidian.exe", []string{"C:/A/Estudo", "C:/A/Oral"}, false)
	if entradas[0].Chave != "gobsidian-estudo" || entradas[1].Chave != "gobsidian-oral" {
		t.Fatalf("chaves sem colisao mudaram: %q, %q", entradas[0].Chave, entradas[1].Chave)
	}
}

// TestChaveDeCofreComNomeNaoLatino e H2.
func TestChaveDeCofreComNomeNaoLatino(t *testing.T) {
	grego := "C:/X/" + string([]rune{0x03A9, 0x03BC, 0x03AD, 0x03B3, 0x03B1})
	japones := "C:/X/" + string([]rune{0x65E5, 0x672C, 0x8A9E})
	kg, kj := ChaveDeCofre(grego), ChaveDeCofre(japones)
	for _, k := range []string{kg, kj} {
		if k == hosts.ChaveDoServidor {
			t.Fatalf("nome nao latino virou a chave do cofre unico: %q", k)
		}
		if !strings.HasPrefix(k, hosts.PrefixoDeCofre) || len(k) == len(hosts.PrefixoDeCofre) {
			t.Fatalf("chave %q sem o prefixo de cofre ou sem sufixo", k)
		}
	}
	if kg == kj {
		t.Fatalf("dois cofres de nome nao latino com a mesma chave %q", kg)
	}
	if ChaveDeCofre(grego) != kg {
		t.Fatal("a chave do mesmo cofre mudou entre duas chamadas")
	}

	casos := map[string]string{
		"C:/X/S" + string(rune(0x00F8)) + "ren":                "gobsidian-soren",
		"C:/X/Stra" + string(rune(0x00DF)) + "e":               "gobsidian-strasse",
		"C:/X/" + string(rune(0x00C6)) + "sir":                 "gobsidian-aesir",
		"C:/X/" + string(rune(0x0141)) + "odz":                 "gobsidian-lodz",
		"C:/X/A" + string([]rune{0x00E7, 0x00E3}) + "o Direta": "gobsidian-acao-direta",
	}
	for caminho, quer := range casos {
		if got := ChaveDeCofre(caminho); got != quer {
			t.Errorf("ChaveDeCofre(%q) = %q, esperado %q", caminho, got, quer)
		}
	}
}
