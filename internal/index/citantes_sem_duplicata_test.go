package index

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jonyd/gobsidian/internal/vault"
)

// TestCitantesNaoDuplicamApósReplace fixa a invariante que o achado P8 depende
// para poder ser corrigido.
//
// `registrarCitantesLocked` verificava `slices.Contains` na lista inteira de
// citantes de cada alvo, uma vez por link. Para um alvo-hub — a nota que meio
// cofre cita — essa lista cresce com o número de citantes, e a varredura roda
// para cada um deles: soma quadrática no Build.
//
// Tirar o `Contains` só é seguro se um mesmo caminho nunca for registrado duas
// vezes sem passar por `desregistrarCitantesLocked` no meio. Os pontos de
// chamada dizem que sim — `publishNoteLocked` roda no insert do Build e no
// Replace, e o Replace desregistra antes. Isto aqui **verifica** em vez de
// confiar: se a invariante quebrar, a lista ganha duplicata e a nota é
// reprocessada duas vezes a cada mudança do alvo.
//
// O teste está no pacote `index`, e não em `index_test`, porque ele olha o mapa
// interno: o que importa aqui é a estrutura, não a resposta da API.
func TestCitantesNaoDuplicamAposReplace(t *testing.T) {
	root := t.TempDir()
	escrever := func(nome, corpo string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, nome), []byte(corpo), 0644); err != nil {
			t.Fatalf("escrevendo %s: %v", nome, err)
		}
	}
	// Três notas citando o MESMO alvo, e uma delas citando-o duas vezes.
	escrever("Hub.md", "# Hub\n")
	escrever("A.md", "# A\n\n[[Hub]] e de novo [[Hub]]\n")
	escrever("B.md", "# B\n\n[[Hub]]\n")
	escrever("C.md", "# C\n\n[[Hub]]\n")

	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	ix := New()
	if err := ix.Build(context.Background(), v); err != nil {
		t.Fatalf("Build: %v", err)
	}

	conferir := func(momento string) {
		t.Helper()
		ix.mu.RLock()
		defer ix.mu.RUnlock()
		for chave, lista := range ix.citantesPorNome {
			vistos := map[vault.CanonicalPath]int{}
			for _, p := range lista {
				vistos[p]++
			}
			for p, n := range vistos {
				if n > 1 {
					t.Errorf("%s: %q aparece %d vezes como citante de %q\n"+
						"a nota sera reprocessada %d vezes a cada mudanca do alvo", momento, p, n, chave, n)
				}
			}
		}
	}
	conferir("apos Build")

	// Sem isto o teste cobriria só o Build, e o caminho que pode duplicar é
	// justamente o de republicação.
	escrever("A.md", "# A\n\n[[Hub]] agora uma vez so\n")
	if err := ix.Replace(context.Background(), v, "A.md"); err != nil {
		t.Fatalf("Replace: %v", err)
	}
	conferir("apos Replace")

	// E o alvo continua com os três citantes: dedup não pode virar perda.
	ix.mu.RLock()
	n := len(ix.citantesPorNome[nomeChave("Hub")])
	ix.mu.RUnlock()
	if n != 3 {
		t.Errorf("Hub tem %d citantes, queria 3 (A, B, C)", n)
	}
}
