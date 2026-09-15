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
