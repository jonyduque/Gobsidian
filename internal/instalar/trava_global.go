package instalar

import (
	"fmt"
	"path/filepath"

	"github.com/jonyd/gobsidian/internal/daemon"
	"github.com/jonyd/gobsidian/internal/ipc"
)

// NomeDaTravaGlobal e o arquivo, no diretorio de runtime, que diz "ha uma
// instalacao em curso". Nome fixo e nao derivado de cofre: a instalacao e da
// MAQUINA, nao de um cofre -- ela troca um binario que todos os cofres usam.
const NomeDaTravaGlobal = "instalacao.lock"

// DiretorioDeRuntime e o diretorio onde socket, travas e presencas moram.
//
// Delega para ipc.RuntimeDir em vez de recalcular: uma conta por regra. Duas
// contas do mesmo caminho concordam por coincidencia ate uma delas mudar
// sozinha -- a licao que config.VaultKey registra sob o nome byAlias.
func DiretorioDeRuntime() (string, error) {
	return ipc.RuntimeDir()
}

// TomarTravaGlobal impede que qualquer `serve` ou `daemon` suba enquanto a
// instalacao acontece.
//
// # Por que ela e o mecanismo, e nao a rede de seguranca
//
// A decisao do dono (D-06/D-07 da spec) e encerrar TODOS os processos antes de
// trocar o binario, porque a alternativa -- conviver com duas versoes -- exige
// manter tres invariantes simultaneas, e invariante e o que quebra em silencio.
//
// So que um host MCP respawna o servidor sozinho, e rapido: medido em
// 2026-09-07, o host reconectou 15 s depois da desconexao (17:45:15 -> daemon
// novo as 17:45:30). Sem esta trava, o respawn cai no MEIO da troca do binario
// e duas versoes voltam a conviver -- exatamente o defeito que encerrar todo
// mundo existe para fechar.
//
// Trava de kernel, e nao arquivo-sentinela: se o instalador morrer no meio, o
// kernel solta, e a proxima partida de `serve` nao fica presa para sempre atras
// de um arquivo que ninguem vai remover. Foi esse o defeito de 2026-08-13, que
// desligou o daemon por tres dias (docs/OPERACAO.md).
func TomarTravaGlobal(runtimeDir string) (liberar func(), err error) {
	caminho := filepath.Join(runtimeDir, NomeDaTravaGlobal)
	trava, tomou, err := daemon.TentarTravar(caminho)
	if err != nil {
		return nil, fmt.Errorf("travando instalacao: %w", err)
	}
	if !tomou {
		return nil, fmt.Errorf("ja ha uma instalacao em curso (%s)", caminho)
	}
	return trava.Liberar, nil
}

// InstalacaoEmCurso e a pergunta que `serve` e `daemon` fazem ao subir.
//
// Erro ao perguntar devolve false, e nao erro: nao poder consultar a trava nao
// pode impedir o servidor de servir. O pior caso de um false errado e uma
// sessao que sobe durante a troca -- que o encerramento do instalador resolve.
// O pior caso de um erro propagado e o produto nao subir por causa do
// instalador, que e pior.
func InstalacaoEmCurso(runtimeDir string) bool {
	caminho := filepath.Join(runtimeDir, NomeDaTravaGlobal)
	emUso, err := daemon.TravaEmUso(caminho)
	if err != nil {
		return false
	}
	return emUso
}
