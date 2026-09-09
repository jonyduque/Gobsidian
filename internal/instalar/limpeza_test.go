package instalar

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jonyd/gobsidian/internal/daemon"
	"github.com/jonyd/gobsidian/internal/search"
)

// montarLixeira cria um diretorio de runtime e uma raiz de cache com os quatro
// casos que devem ser limpos E os quatro inversos que NAO devem.
//
// Os inversos sao o que impede uma limpeza que apaga tudo de passar: sem eles,
// `os.RemoveAll(runtimeDir)` seria aprovado por qualquer teste que so olhasse o
// que sumiu.
func montarLixeira(t *testing.T) (runtimeDir, cacheRaiz, cofreVivoPath string) {
	t.Helper()
	runtimeDir = t.TempDir()
	cacheRaiz = t.TempDir()

	cofreVivoPath = t.TempDir() // este existe
	cofreSumidoPath := filepath.Join(t.TempDir(), "cofre-que-foi-apagado")

	escreverCache(t, filepath.Join(cacheRaiz, "vivo"), cofreVivoPath)
	escreverCache(t, filepath.Join(cacheRaiz, "sumido"), cofreSumidoPath)

	// A LIMPAR: pertencem ao cofre que sumiu.
	escrever(t, filepath.Join(runtimeDir, "sumido.sock"), "")
	escrever(t, filepath.Join(runtimeDir, "sumido.sock.lock"), "4242\n")
	escrever(t, filepath.Join(runtimeDir, "sumido.sock.listen.lock"), "4242\n")

	// A PRESERVAR: pertencem ao cofre que existe.
	escrever(t, filepath.Join(runtimeDir, "vivo.sock"), "")
	escrever(t, filepath.Join(runtimeDir, "vivo.sock.lock"), "4242\n")

	// Log pequeno: nunca rotacionado.
	escrever(t, filepath.Join(runtimeDir, "vivo.sock.log"), "pequeno\n")
	// Log grande: rotacionado, nunca apagado.
	escrever(t, filepath.Join(runtimeDir, "sumido.sock.log"), strings.Repeat("x", TetoDoLog+1))

	return runtimeDir, cacheRaiz, cofreVivoPath
}

func escrever(t *testing.T, caminho, conteudo string) {
	t.Helper()
	if err := os.WriteFile(caminho, []byte(conteudo), 0o600); err != nil {
		t.Fatal(err)
	}
}

// TestLimparRemoveSoOComprovadamenteOrfao e a decisao D-05 do dono, virada em
// teste: a regra e mecanica, e o inverso de cada caso esta aqui.
//
// Motivo medido: em 2026-09-08 o diretorio de runtime do dono tinha 960
// arquivos .lock e 3 .sock de cofres que nao existem mais, alem de um log de
// 727 261 bytes sem limite.
func TestLimparRemoveSoOComprovadamenteOrfao(t *testing.T) {
	runtimeDir, cacheRaiz, _ := montarLixeira(t)

	r, err := Limpar(runtimeDir, cacheRaiz, true)
	if err != nil {
		t.Fatalf("Limpar() error = %v", err)
	}
	if len(r.NaoRemovidos) != 0 {
		t.Fatalf("nao removidos: %v", r.NaoRemovidos)
	}

	// Sumiu o do cofre inexistente.
	for _, nome := range []string{"sumido.sock", "sumido.sock.lock", "sumido.sock.listen.lock"} {
		if _, err := os.Stat(filepath.Join(runtimeDir, nome)); !os.IsNotExist(err) {
			t.Errorf("%s sobreviveu; era lixo comprovado", nome)
		}
	}
	if _, err := os.Stat(filepath.Join(cacheRaiz, "sumido")); !os.IsNotExist(err) {
		t.Error("o cache do cofre inexistente sobreviveu")
	}

	// Ficou o do cofre que existe. Apagar cache de quem usa o produto custa
	// reconstrucao: 3021 ms medidos no cofre de referencia do dono.
	for _, nome := range []string{"vivo.sock", "vivo.sock.lock", "vivo.sock.log"} {
		if _, err := os.Stat(filepath.Join(runtimeDir, nome)); err != nil {
			t.Errorf("%s foi removido, mas o cofre dele existe: %v", nome, err)
		}
	}
	if _, err := os.Stat(filepath.Join(cacheRaiz, "vivo")); err != nil {
		t.Errorf("o cache do cofre existente foi removido: %v", err)
	}

	// Log grande rotacionado, NUNCA apagado.
	if _, err := os.Stat(filepath.Join(runtimeDir, "sumido.sock.log.1")); err != nil {
		t.Errorf("o log grande nao foi rotacionado: %v", err)
	}
	if !slices.ContainsFunc(r.LogsRotacionados, func(s string) bool {
		return strings.HasSuffix(s, "sumido.sock.log")
	}) {
		t.Errorf("o relatorio nao menciona a rotacao: %+v", r.LogsRotacionados)
	}
	// Log pequeno intocado.
	if _, err := os.Stat(filepath.Join(runtimeDir, "vivo.sock.log.1")); !os.IsNotExist(err) {
		t.Error("um log de 8 bytes foi rotacionado")
	}
}

// TestLimparSemAplicarNaoTocaEmNada e o modo que `doctor` usa: relatar sem
// mexer. Sem esta separacao, ver o que ha exigiria autorizar a remocao.
func TestLimparSemAplicarNaoTocaEmNada(t *testing.T) {
	runtimeDir, cacheRaiz, _ := montarLixeira(t)

	r, err := Limpar(runtimeDir, cacheRaiz, false)
	if err != nil {
		t.Fatalf("Limpar() error = %v", err)
	}
	if r.Vazio() {
		t.Fatal("Limpar(aplicar=false) nao encontrou nada; a lixeira tem lixo de proposito")
	}

	for _, nome := range []string{"sumido.sock", "sumido.sock.lock", "sumido.sock.listen.lock", "sumido.sock.log"} {
		if _, err := os.Stat(filepath.Join(runtimeDir, nome)); err != nil {
			t.Errorf("%s sumiu com aplicar=false: %v", nome, err)
		}
	}
	if _, err := os.Stat(filepath.Join(cacheRaiz, "sumido")); err != nil {
		t.Errorf("o cache sumiu com aplicar=false: %v", err)
	}
}

// TestLimparNaoRemoveTravaEmUso: a trava tomada e um processo VIVO. Remover o
// arquivo dele durante uma instalacao seria remover a prova de que ele existe,
// e o instalador deixaria de mostra-lo ao usuario antes de encerrar.
func TestLimparNaoRemoveTravaEmUso(t *testing.T) {
	runtimeDir := t.TempDir()
	cacheRaiz := t.TempDir()

	// Sem cache nenhum, entao a chave e "desconhecida" -- e mesmo assim a
	// trava tomada tem de vencer.
	caminho := filepath.Join(runtimeDir, "abc.sock.lock")
	trava, tomou, err := tentarTravarParaTeste(caminho)
	if err != nil || !tomou {
		t.Fatalf("nao consegui tomar a trava do teste: %v", err)
	}
	defer trava()

	r, err := Limpar(runtimeDir, cacheRaiz, true)
	if err != nil {
		t.Fatalf("Limpar() error = %v", err)
	}
	if len(r.Locks) != 0 {
		t.Errorf("removeu uma trava EM USO: %v", r.Locks)
	}
	if _, err := os.Stat(caminho); err != nil {
		t.Errorf("o arquivo da trava em uso sumiu: %v", err)
	}

	// NaoRemovidos vazio prova que a limpeza nem TENTOU.
	//
	// Sem esta linha o teste passa pelo motivo errado no Windows: o arquivo
	// esta aberto por este processo sem FILE_SHARE_DELETE, entao o os.Remove
	// falha por conta do sistema operacional e o arquivo sobrevive de qualquer
	// jeito. Medido: a mutacao que inverte a guarda `err != nil || emUso` para
	// `&&` passava neste teste antes desta assercao -- um teste que nao podia
	// falhar.
	if len(r.NaoRemovidos) != 0 {
		t.Errorf("a limpeza TENTOU remover uma trava em uso e o sistema operacional a salvou: %v", r.NaoRemovidos)
	}
}

// TestLimparDiretoriosAusentesNaoSaoErro: `doctor` e o instalador perguntam
// antes de o produto ter rodado alguma vez.
func TestLimparDiretoriosAusentesNaoSaoErro(t *testing.T) {
	base := t.TempDir()
	r, err := Limpar(filepath.Join(base, "sem-runtime"), filepath.Join(base, "sem-cache"), true)
	if err != nil {
		t.Fatalf("Limpar() em diretorios ausentes virou erro: %v", err)
	}
	if !r.Vazio() {
		t.Errorf("Limpar() achou algo onde nao ha nada: %+v", r)
	}
}

// escreverCache grava um cache de busca DE VERDADE, com o cabecalho que nomeia
// o cofre.
//
// Nao monta o arquivo a mao de proposito: o formato e de internal/search, e um
// cabecalho escrito aqui seria uma segunda conta do layout -- que concorda com
// a primeira por coincidencia ate uma das duas mudar. SaveInvertedCache e o
// mesmo caminho que producao usa.
func escreverCache(t *testing.T, dir, vaultPath string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := search.SaveInvertedCache(context.Background(), dir, vaultPath, search.NewInverted()); err != nil {
		t.Fatalf("SaveInvertedCache(%s): %v", dir, err)
	}
}

// tentarTravarParaTeste segura uma trava de kernel de verdade, para o teste
// poder provar que a limpeza respeita um processo vivo.
func tentarTravarParaTeste(caminho string) (liberar func(), tomou bool, err error) {
	trava, tomou, err := daemon.TentarTravar(caminho)
	if err != nil || !tomou {
		return func() {}, tomou, err
	}
	return trava.Liberar, true, nil
}
