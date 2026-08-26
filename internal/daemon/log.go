package daemon

import (
	"fmt"
	"os"
	"strings"

	"github.com/jonyd/gobsidian/internal/ipc"
)

// CaminhoDoLog deriva o arquivo de log do daemon do MESMO caminho que
// ipc.SocketPath calcula, trocando só a extensão — do mesmo jeito que lockPath
// deriva o do lock.
//
// Um cofre, um socket, um lock, um log, todos pela MESMA chave
// (config.VaultKey). Esta função existe porque a conta estava em três lugares:
// aqui, em cmd/gobsidian/daemon.go e em internal/doctor. Três contas do mesmo
// valor concordam por coincidência até o dia em que uma muda sozinha — é a
// lição de `byAlias`, e ela já custou caro neste projeto.
func CaminhoDoLog(vaultPath string) (string, error) {
	sock, err := ipc.SocketPath(vaultPath)
	if err != nil {
		return "", err
	}
	return sock + ".log", nil
}

// UltimasLinhasDoLog devolve as n últimas linhas não vazias do log do daemon.
//
// O segundo retorno diz se o arquivo existe. A distinção importa e é o achado
// de 2026-08-26: log AUSENTE significa que o processo do daemon não chegou a
// escrever nada — o problema está no spawn, não no cofre. Log PRESENTE com uma
// linha só ("daemon iniciado" e mais nada) significa que ele subiu e morreu na
// montagem. São dois defeitos diferentes com o mesmo sintoma visível para quem
// só olha o socket, e distingui-los é a diferença entre um diagnóstico e um
// palpite.
func UltimasLinhasDoLog(vaultPath string, n int) (linhas []string, existe bool) {
	path, err := CaminhoDoLog(vaultPath)
	if err != nil {
		return nil, false
	}
	dados, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	for l := range strings.SplitSeq(string(dados), "\n") {
		if l = strings.TrimRight(l, "\r"); strings.TrimSpace(l) != "" {
			linhas = append(linhas, l)
		}
	}
	if len(linhas) > n {
		linhas = linhas[len(linhas)-n:]
	}
	return linhas, true
}

// pistaDoLog resume, em uma linha, o que o log do daemon diz — para colar num
// erro de prazo estourado.
//
// Sem isto, "socket do daemon nao respondeu em 10s" culpa o socket, que é o
// único lugar onde a resposta NÃO está. Medido em 2026-08-26: três sessões
// pagaram esse prazo em toda partida durante dias, e a mensagem nunca apontou
// para o log, onde a causa (ou a ausência dela) estava o tempo todo.
func pistaDoLog(vaultPath string) string {
	linhas, existe := UltimasLinhasDoLog(vaultPath, 1)
	if !existe {
		return "o log do daemon nao existe: o processo nao chegou a escrever nada (falha de spawn, nao do cofre)"
	}
	if len(linhas) == 0 {
		return "o log do daemon esta vazio"
	}
	path, _ := CaminhoDoLog(vaultPath)
	return fmt.Sprintf("ultima linha de %s: %s", path, linhas[len(linhas)-1])
}
