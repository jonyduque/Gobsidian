package instalar

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jonyduque/Gobsidian/internal/config"
)

// raizComCache monta uma raiz de cache com UM diretorio, nomeado como o teste
// mandar -- inclusive com um nome que nao e a chave do cofre, que e o caso
// inteiro desta migracao.
func raizComCache(t *testing.T, nomeDoDir string, cofreExiste bool) (raiz, cofre string) {
	t.Helper()
	base := t.TempDir()
	raiz = filepath.Join(base, "cache")
	cofre = filepath.Join(base, "Cofre")
	if cofreExiste {
		if err := os.MkdirAll(cofre, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	escreverCache(t, filepath.Join(raiz, nomeDoDir), cofre)
	return raiz, cofre
}

func TestMigrarChavesNaoTocaNoQueJaEstaCerto(t *testing.T) {
	base := t.TempDir()
	raiz := filepath.Join(base, "cache")
	cofre := filepath.Join(base, "Cofre")
	if err := os.MkdirAll(cofre, 0o700); err != nil {
		t.Fatal(err)
	}
	escreverCache(t, filepath.Join(raiz, config.VaultKey(cofre)), cofre)

	migradas, err := MigrarChaves(raiz, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(migradas) != 0 {
		t.Errorf("MigrarChaves devolveu %+v, queria nada: a chave ja e a certa", migradas)
	}
}

func TestMigrarChavesRenomeiaChaveSuperada(t *testing.T) {
	raiz, cofre := raizComCache(t, "chaveantiga", true)
	quer := config.VaultKey(cofre)

	migradas, err := MigrarChaves(raiz, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(migradas) != 1 || !migradas[0].Feito {
		t.Fatalf("MigrarChaves = %+v, queria uma migracao feita", migradas)
	}
	if migradas[0].De != "chaveantiga" || migradas[0].Para != quer {
		t.Errorf("migracao = %s -> %s, queria chaveantiga -> %s", migradas[0].De, migradas[0].Para, quer)
	}
	if _, err := os.Stat(filepath.Join(raiz, quer)); err != nil {
		t.Errorf("o diretorio nao esta sob a chave nova: %v", err)
	}
	if _, err := os.Stat(filepath.Join(raiz, "chaveantiga")); err == nil {
		t.Errorf("o diretorio antigo continua la: renomear nao e copiar")
	}
}

// Sem aplicar nada muda. E o modo que `doctor` usa, e ele roda com o produto
// no ar -- renomear cache que alguem mapeia com mmap e pedir problema.
func TestMigrarChavesSemAplicarNaoRenomeia(t *testing.T) {
	raiz, _ := raizComCache(t, "chaveantiga", true)

	migradas, err := MigrarChaves(raiz, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(migradas) != 1 || migradas[0].Feito {
		t.Fatalf("MigrarChaves = %+v, queria uma migracao relatada e NAO feita", migradas)
	}
	if _, err := os.Stat(filepath.Join(raiz, "chaveantiga")); err != nil {
		t.Errorf("o diretorio antigo sumiu numa simulacao: %v", err)
	}
}

// Destino ocupado: nao sobrescreve, e diz que nao sobrescreveu. Sem o Motivo,
// uma migracao que nao pode agir teria a mesma saida de uma que agiu.
func TestMigrarChavesNaoSobrescreveDestinoOcupado(t *testing.T) {
	base := t.TempDir()
	raiz := filepath.Join(base, "cache")
	cofre := filepath.Join(base, "Cofre")
	if err := os.MkdirAll(cofre, 0o700); err != nil {
		t.Fatal(err)
	}
	escreverCache(t, filepath.Join(raiz, "chaveantiga"), cofre)
	escreverCache(t, filepath.Join(raiz, config.VaultKey(cofre)), cofre)

	migradas, err := MigrarChaves(raiz, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(migradas) != 1 || migradas[0].Feito {
		t.Fatalf("MigrarChaves = %+v, queria uma migracao recusada", migradas)
	}
	if migradas[0].Motivo == "" {
		t.Error("migracao recusada sem motivo: o relatorio nao distingue recusa de sucesso")
	}
	if _, err := os.Stat(filepath.Join(raiz, "chaveantiga")); err != nil {
		t.Errorf("o diretorio antigo foi tocado apesar do destino ocupado: %v", err)
	}
}

// Cofre sumido e trabalho da limpeza, sob a regra dela. Renomear lixo so o
// deixa com nome novo.
func TestMigrarChavesNaoRenomeiaCacheDeCofreSumido(t *testing.T) {
	raiz, _ := raizComCache(t, "chaveantiga", false)

	migradas, err := MigrarChaves(raiz, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(migradas) != 1 || migradas[0].Feito {
		t.Fatalf("MigrarChaves = %+v, queria uma migracao recusada", migradas)
	}
	if _, err := os.Stat(filepath.Join(raiz, "chaveantiga")); err != nil {
		t.Errorf("o diretorio de um cofre sumido foi renomeado: %v", err)
	}
}

func TestMigrarChavesRaizAusenteNaoEhErro(t *testing.T) {
	migradas, err := MigrarChaves(filepath.Join(t.TempDir(), "nao-existe"), true)
	if err != nil {
		t.Fatalf("raiz ausente virou erro: %v", err)
	}
	if len(migradas) != 0 {
		t.Errorf("MigrarChaves = %+v, queria nada", migradas)
	}
}
