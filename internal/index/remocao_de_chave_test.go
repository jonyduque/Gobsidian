package index

import (
	"context"
	"testing"

	"github.com/jonyd/gobsidian/internal/vault"
)

// TestRemoverNomeNaoDeixaChaveOrfa cobre o que a resolucao NAO cobre.
//
// `vivosLocked` filtra caminho ja removido na hora de responder, entao uma
// entrada orfa em lowerPath nao produz resposta errada — o teste de colisao
// passa mesmo sem a remocao. O que ela evita e VAZAMENTO: num daemon de vida
// longa, cada nota apagada ou renomeada deixaria uma entrada para sempre, e o
// indice derivado cresceria sem limite enquanto o cofre encolhe.
//
// Por isso a asserção e sobre o MAPA, e nao sobre o que ResolvePath devolve.
func TestRemoverNomeNaoDeixaChaveOrfa(t *testing.T) {
	root := t.TempDir()
	writeFileHelper(t, root, "pasta/Nota.md", "# N\n")

	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	ix := New()
	if err := ix.Build(context.Background(), v); err != nil {
		t.Fatalf("Build: %v", err)
	}

	const chaveCaminho = "pasta/nota.md"
	const chaveNome = "nota.md"
	if len(ix.lowerPath[chaveCaminho]) != 1 {
		t.Fatalf("lowerPath[%q] = %v; a fixture nao monta o teste", chaveCaminho, ix.lowerPath[chaveCaminho])
	}
	if len(ix.byName[chaveNome]) != 1 {
		t.Fatalf("byName[%q] = %v; a fixture nao monta o teste", chaveNome, ix.byName[chaveNome])
	}

	ix.Remove("pasta/Nota.md")

	if _, existe := ix.lowerPath[chaveCaminho]; existe {
		t.Errorf("lowerPath[%q] sobreviveu a remocao: %v\n"+
			"chave orfa nao devolve resposta errada, mas vaza num daemon de vida longa",
			chaveCaminho, ix.lowerPath[chaveCaminho])
	}
	if _, existe := ix.byName[chaveNome]; existe {
		t.Errorf("byName[%q] sobreviveu a remocao: %v", chaveNome, ix.byName[chaveNome])
	}
}

// TestRemoverNomeMantemAChaveQuandoSobraAlguem e o contrapeso: apagar a chave
// inteira em vez de filtrar a lista derrubaria a nota que ficou.
func TestRemoverNomeMantemAChaveQuandoSobraAlguem(t *testing.T) {
	root := t.TempDir()
	writeFileHelper(t, root, "civil/Nota.md", "# C\n")
	writeFileHelper(t, root, "penal/Nota.md", "# P\n")

	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	ix := New()
	if err := ix.Build(context.Background(), v); err != nil {
		t.Fatalf("Build: %v", err)
	}

	const chaveNome = "nota.md"
	if len(ix.byName[chaveNome]) != 2 {
		t.Fatalf("byName[%q] = %v, queria 2", chaveNome, ix.byName[chaveNome])
	}

	ix.Remove("civil/Nota.md")

	restantes := ix.byName[chaveNome]
	if len(restantes) != 1 || restantes[0] != vault.CanonicalPath("penal/Nota.md") {
		t.Errorf("byName[%q] = %v, queria so penal/Nota.md", chaveNome, restantes)
	}
}
