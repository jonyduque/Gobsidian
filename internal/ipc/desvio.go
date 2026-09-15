package ipc

import (
	"fmt"
	"os"
)

// desvioDoRuntime, quando nao vazio, toma o lugar do diretorio de runtime do
// sistema. So RodarComRuntimeIsolado o escreve, e so TestMain chama
// RodarComRuntimeIsolado: check_test_isolation.ps1 recusa a chamada fora de
// arquivo _test.go.
//
// E escrito uma vez, antes de m.Run, e NUNCA zerado. A primeira redacao zerava
// no fim, supondo que nenhuma goroutine de teste sobrevivesse a m.Run; o -race
// do verify.ps1 mostrou o contrario em 2026-09-14: TestEnsureStartedPerdedor
// NuncaChamaIniciar (internal/daemon/lock_test.go) deixa uma goroutine discando
// o socket depois de a suite terminar, e ela leu esta variavel enquanto o defer
// a escrevia. Zerar nao compra nada -- o processo sai com os.Exit logo depois --,
// e nao zerar tira a escrita concorrente de existir.
var desvioDoRuntime string

// RuntimeDir devolve o diretorio onde moram socket, travas, log do daemon e
// presenca. A conta de producao e runtimeDirDoSistema, uma por plataforma.
func RuntimeDir() (string, error) {
	if desvioDoRuntime != "" {
		return desvioDoRuntime, nil
	}
	return runtimeDirDoSistema()
}

// DiretorioDeRuntimeAntigo devolve onde versoes ate 2026-09-14 guardavam o
// runtime, ou vazio quando nao ha transicao nesta plataforma.
//
// Com o desvio dos testes armado, devolve vazio: o diretorio antigo e o do
// usuario, e um teste que o lesse ou varresse sairia do isolamento que
// RodarComRuntimeIsolado existe para garantir.
func DiretorioDeRuntimeAntigo() string {
	if desvioDoRuntime != "" {
		return ""
	}
	return runtimeDirAntigoDoSistema()
}

// executorDeTestes e a parte de *testing.M que importa aqui. Receber a
// interface, e nao *testing.M, mantem o pacote testing fora do binario.
type executorDeTestes interface{ Run() int }

// RodarComRuntimeIsolado roda a suite de um pacote com o diretorio de runtime
// num temporario proprio, e apaga o temporario no fim. O uso, e o unico:
//
//	func TestMain(m *testing.M) { os.Exit(ipc.RodarComRuntimeIsolado(m)) }
//
// # O defeito que isto conserta
//
// Medido em 2026-09-14: `go test -count=1` sobre ipc, daemon, doctor, instalar
// e cmd/gobsidian levou %LOCALAPPDATA%\gobsidian\run de 116 para 132 arquivos,
// e nada foi removido. Rodando a suite inteira contra uma isca, o que caiu nela
// foram 15 travas de cofres de t.TempDir(), `instalacao.lock` e um arquivo de
// presenca. Cada teste que abre socket nomeia socket, trava e log pela chave
// do cofre, e o diretorio que resolvia esses caminhos era o do usuario.
//
// # Por que variavel de pacote, e nao variavel de ambiente
//
// A regra de docs/ARMADILHAS.md: variavel de ambiente dentro do teste finge
// isolar. os.UserCacheDir respeita LOCALAPPDATA no Windows e ignora qualquer
// variavel no macOS. O que isola e uma raiz trocada por variavel de pacote --
// o mesmo padrao de instalar.raizDoCache. A diferenca e que esta cruza
// pacotes: daemon, doctor, instalar e cmd/gobsidian chegam ao diretorio de
// runtime atraves de ipc, e nenhum deles consegue trocar uma variavel nao
// exportada daqui.
//
// # Por que o prefixo curto
//
// O caminho do socket e diretorio + chave de 16 caracteres + ".sock", e o
// AF_UNIX limita o caminho a pouco mais de 100 bytes. No macOS o TMPDIR ja
// ocupa cerca de 50. "gobs-rt-" deixa folga; um nome descritivo nao deixaria.
func RodarComRuntimeIsolado(m executorDeTestes) int {
	dir, err := os.MkdirTemp("", "gobs-rt-")
	if err != nil {
		fmt.Fprintf(os.Stderr, "criando diretorio de runtime isolado para os testes: %v\n", err)
		return 1
	}
	desvioDoRuntime = dir
	codigo := m.Run()
	// Best-effort: um listener que um teste esqueceu aberto segura o arquivo no
	// Windows, e uma goroutine que sobreviveu a m.Run pode recriar o diretorio.
	// O que sobrar fica no TEMP, e nao no perfil do usuario.
	_ = os.RemoveAll(dir)
	return codigo
}
