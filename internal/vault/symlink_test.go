package vault_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jonyd/gobsidian/internal/vault"
)

// criaSymlink cria um link simbolico e devolve false quando o sistema recusa
// por PERMISSAO.
//
// A distincao importa. No Windows, criar link simbolico exige privilegio
// elevado ou o Modo de Desenvolvedor ligado, e numa maquina comum a chamada
// falha. Pular nesse caso e legitimo; pular em silencio, nao — um t.Skip
// permanente vira cobertura fantasma, e este projeto ja teve tres testes que
// reportavam cobertura inexistente. Por isso a mensagem do skip diz que foi
// permissao, e QUALQUER outro erro e falha.
func criaSymlink(t *testing.T, alvo, link string) bool {
	t.Helper()
	err := os.Symlink(alvo, link)
	if err == nil {
		return true
	}
	if os.IsPermission(err) || errosDePrivilegioDeSymlink(err) {
		t.Skipf("RNF-32 NAO VERIFICADO nesta maquina: criacao de link simbolico "+
			"recusada por PERMISSAO (%v). No Windows exige privilegio elevado ou "+
			"Modo de Desenvolvedor. Rode elevado para exercitar este teste.", err)
		return false
	}
	t.Fatalf("os.Symlink falhou por motivo que NAO e permissao: %v", err)
	return false
}

// TestWalkNaoSegueSymlink fixa o RNF-32: vault.Walk não atravessa link
// simbólico.
//
// Até 2026-08-02 a propriedade valia "por construção" — `filepath.WalkDir` não
// segue links — e tinha apenas teste indireto na Task 7. Propriedade que vale
// por construção continua valendo até alguém trocar a construção; o que este
// teste fixa é que trocar `WalkDir` por algo que siga links passa a quebrar
// aqui, em vez de abrir uma varredura para fora do cofre em silêncio.
//
// O cenário é o que importa: o alvo do link fica FORA da raiz e contém uma
// nota. Se a varredura seguisse o link, essa nota apareceria — e o
// confinamento de caminho, que é a garantia inteira, teria sido furado por uma
// camada que ninguém suspeitava.
func TestWalkNaoSegueSymlink(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "cofre")
	fora := filepath.Join(base, "fora")

	for _, d := range []string{root, fora} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatalf("MkdirAll %s: %v", d, err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "dentro.md"), []byte("# Dentro"), 0o644); err != nil {
		t.Fatalf("WriteFile dentro.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(fora, "vazada.md"), []byte("# Vazada"), 0o644); err != nil {
		t.Fatalf("WriteFile vazada.md: %v", err)
	}

	if !criaSymlink(t, fora, filepath.Join(root, "atalho")) {
		return
	}

	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}

	var vistos []string
	if err := v.Walk(context.Background(), func(e vault.Entry) error {
		vistos = append(vistos, string(e.Path))
		return nil
	}); err != nil {
		t.Fatalf("Walk: %v", err)
	}

	// Guarda de cenario: se dentro.md nao aparecer, a varredura nao andou e a
	// ausencia de vazada.md nao prova nada.
	var achouDentro bool
	for _, p := range vistos {
		if p == "dentro.md" {
			achouDentro = true
		}
		if filepath.Base(p) == "vazada.md" {
			t.Errorf("Walk atravessou o link simbolico e entregou %q — RNF-32 violado: "+
				"a varredura saiu da raiz do cofre", p)
		}
	}
	if !achouDentro {
		t.Fatalf("Walk nao entregou dentro.md; entradas vistas: %v — o cenario nao "+
			"exercitou nada e a ausencia de vazada.md nao significa nada", vistos)
	}
}
