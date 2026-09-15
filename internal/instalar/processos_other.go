//go:build !windows

package instalar

// ProcessosDoSistema so e implementado no Windows, onde os processos sem
// presenca foram medidos. Aqui a resposta e "nao verificado", nunca "nenhum".
func ProcessosDoSistema() ([]ProcessoDoSistema, error) {
	return nil, ErrProcessosNaoVerificados
}
