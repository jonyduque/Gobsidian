package instalar

import (
	"github.com/jonyduque/Gobsidian/internal/ipc"
)

// diretorioAntigoFn e ipc.DiretorioDeRuntimeAntigo numa variavel, para o teste
// apontar a transicao para um diretorio temporario. Producao nunca a troca.
var diretorioAntigoFn = ipc.DiretorioDeRuntimeAntigo

// DaemonAnteriorDoCofre diz se ha um daemon de versao anterior servindo cofre
// a partir do diretorio de runtime ANTIGO, e o PID dele.
//
// # Por que presenca, e nao trava
//
// Um daemon em regime nao segura trava de inicializacao nem de escuta: as duas
// so ficam tomadas enquanto ele sobe. O que ele segura a vida inteira e a
// presenca (`daemon.<pid>.presenca`), e as versoes desde 2026-09-09 a gravam.
// Encontrar uma viva para o mesmo cofre no diretorio antigo e o sinal de que
// subir um daemon no diretorio novo poria dois processos gravando o mesmo
// cache -- o incidente de 2026-09-08 por outra porta.
func DaemonAnteriorDoCofre(cofre string) (pid int, ok bool) {
	antigo := diretorioAntigoFn()
	if antigo == "" {
		return 0, false
	}
	return daemonNoDiretorio(antigo, cofre)
}

// VivosComAnterior devolve as presencas vivas do diretorio de runtime e, na
// transicao, tambem as do diretorio antigo.
//
// Medido em 2026-09-14, com o `doctor` do binario novo: os processos v1.8.1 em
// execucao registram presenca no diretorio antigo, e o `doctor` listava todos
// como "sem presenca" -- sem cofre, sem modo, sem versao, e sem entrar no aviso
// de gravadores duplicados. Quem pergunta "quem esta rodando?" precisa olhar
// os dois lugares enquanto houver processo de versao anterior.
//
// Erro ao ler o diretorio antigo volta junto com o que foi lido do novo: quem
// chama decide se relata; nao ler um lugar nao apaga o outro.
func VivosComAnterior(runtimeDir string) ([]Presenca, error) {
	vivos, err := Vivos(runtimeDir)
	if err != nil {
		return nil, err
	}
	antigo := diretorioAntigoFn()
	if antigo == "" || mesmoDiretorio(antigo, runtimeDir) {
		return vivos, nil
	}
	anteriores, err := Vivos(antigo)
	return append(vivos, anteriores...), err
}

func daemonNoDiretorio(dir, cofre string) (pid int, ok bool) {
	vivos, err := Vivos(dir)
	if err != nil {
		return 0, false
	}
	for _, p := range vivos {
		if p.Papel == "daemon" && mesmoDiretorio(p.Cofre, cofre) {
			return p.PID, true
		}
	}
	return 0, false
}
