//go:build windows

package daemon

import "golang.org/x/sys/windows"

// pidVivo diz se o processo de PID `pid` ainda esta em execucao.
//
// OpenProcess devolver sucesso NAO e prova de vida, e essa e a armadilha que
// este projeto ja pagou: o Windows nao libera o PID enquanto houver handle
// aberto para o processo morto, e o proprio ato de consultar mantem um. Um
// processo terminado continua respondendo a OpenProcess e devolvendo o mesmo
// creation time por muito tempo depois de morrer — foi o que deixou 5 de 5
// orfaos no primeiro teste de vigilia do pai (ver internal/lifecycle,
// parent_windows.go, e a secao correspondente do CLAUDE.md).
//
// exitTime nao-zero e a resposta: ele so e preenchido quando o processo
// termina.
//
// Nao reusa lifecycle.parentIdentity de proposito: aquela funcao responde
// outra pergunta — "o pai continua sendo o MESMO processo" — e no Unix ela se
// apoia em os.Getppid(), que nao faz sentido para um PID arbitrario lido de um
// arquivo.
func pidVivo(pid int) bool {
	if pid <= 0 {
		return false
	}
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		// Nao existe, ou nao ha permissao para consultar. Sem permissao, o
		// conservador seria assumir vivo — mas o lock e criado pelo MESMO
		// usuario que le, num diretorio 0700, entao "nao consigo consultar"
		// aqui significa "nao existe" na pratica.
		return false
	}
	defer func() { _ = windows.CloseHandle(h) }()

	var creation, exit, kernel, user windows.Filetime
	if err := windows.GetProcessTimes(h, &creation, &exit, &kernel, &user); err != nil {
		return false
	}
	return exit.HighDateTime == 0 && exit.LowDateTime == 0
}
