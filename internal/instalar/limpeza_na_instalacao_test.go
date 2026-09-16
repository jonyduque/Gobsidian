package instalar

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jonyduque/Gobsidian/internal/config"
)

// TestInstalarLimpaOLixoDeExecucoesAnteriores prova o passo 4 da sequencia
// DENTRO da instalacao, e nao so em Limpar isolada.
//
// A diferenca importa porque e a pergunta que o dono fez em 2026-09-16: a
// instalacao limpa o lixo antigo? O teste de Limpar prova a regra; este prova
// que `install` e `update` a executam -- e que o relatorio devolve o que foi
// removido, que e o que a tela mostra.
//
// A limpeza roda sob a trava global e depois de encerrar processos: e a unica
// janela em que remover uma trava livre nao corre com quem esta prestes a
// toma-la.
//
// Os diretorios de cache usam a chave REAL (config.VaultKey), e nao um nome
// qualquer: o passo seguinte da instalacao e MigrarChaves, que renomeia o cache
// cuja chave ficou para tras. Com "vivo" e "sumido" como nomes, a primeira
// redacao deste teste viu o cache do cofre vivo "sumir" -- ele tinha sido
// renomeado para a chave certa, e nao removido.
func TestInstalarLimpaOLixoDeExecucoesAnteriores(t *testing.T) {
	m := novoMundo(t)
	if err := os.MkdirAll(m.runtimeDir, 0o700); err != nil {
		t.Fatal(err)
	}

	// Lixo de um cofre que nao existe mais: cada arquivo tem a prova que a
	// regra exige -- trava livre, ninguem escutando, cofre sumido.
	cofreSumido := filepath.Join(t.TempDir(), "cofre-que-foi-apagado")
	chaveSumida := config.VaultKey(cofreSumido)
	escreverCache(t, filepath.Join(m.cacheRaiz, chaveSumida), cofreSumido)
	escrever(t, filepath.Join(m.runtimeDir, chaveSumida+".sock"), "")
	escrever(t, filepath.Join(m.runtimeDir, chaveSumida+".sock.lock"), "4242\n")

	// E o inverso, que impede uma "limpeza" que apaga tudo de passar: o cofre
	// deste existe.
	cofreVivo := t.TempDir()
	chaveViva := config.VaultKey(cofreVivo)
	escreverCache(t, filepath.Join(m.cacheRaiz, chaveViva), cofreVivo)
	escrever(t, filepath.Join(m.runtimeDir, chaveViva+".sock.lock"), "4242\n")

	r, err := Instalar(context.Background(), m.sistema(), m.opcoes())
	if err != nil {
		t.Fatalf("Instalar() error = %v", err)
	}

	if r.Limpeza.Vazio() {
		t.Fatalf("a instalacao nao relatou limpeza nenhuma: %+v", r.Limpeza)
	}
	for _, nome := range []string{chaveSumida + ".sock", chaveSumida + ".sock.lock"} {
		if _, err := os.Stat(filepath.Join(m.runtimeDir, nome)); !os.IsNotExist(err) {
			t.Errorf("%s sobreviveu a instalacao; era lixo comprovado", nome)
		}
	}
	if _, err := os.Stat(filepath.Join(m.cacheRaiz, chaveSumida)); !os.IsNotExist(err) {
		t.Error("o cache do cofre inexistente sobreviveu a instalacao")
	}
	if _, err := os.Stat(filepath.Join(m.runtimeDir, chaveViva+".sock.lock")); err != nil {
		t.Errorf("a trava do cofre que existe foi removida: %v", err)
	}
	if _, err := os.Stat(filepath.Join(m.cacheRaiz, chaveViva)); err != nil {
		t.Errorf("o cache do cofre que existe foi removido: %v", err)
	}
}
