package instalar

import "errors"

// Os modos de um processo registrado na presenca. Uma conta so: quem registra
// (cmd/gobsidian, no ponto em que o modo ja esta decidido) e quem le (o
// `doctor`) usam estas constantes.
const (
	ModoDaemon     = "daemon"
	ModoPonte      = "ponte"
	ModoEmProcesso = "em-processo"
)

// GravaCache diz se um processo neste modo grava o cache do cofre.
//
// Ponte nao grava: ela so copia bytes entre o host e o daemon. Em 2026-09-14 o
// aviso de duplicidade do `doctor` contava todo `serve` como gravador e mandava
// encerrar os extras -- inclusive as pontes do Antigravity, de 24 MB cada, que
// estavam certas. Os gravadores de verdade eram o daemon e os servidores em
// processo do Claude Desktop, de 42 a 153 MB.
func GravaCache(modo string) bool {
	return modo == ModoDaemon || modo == ModoEmProcesso
}

// ProcessoDoSistema e um processo com o nome do executavel do produto, visto
// pelo sistema operacional e nao pela presenca.
type ProcessoDoSistema struct {
	PID        int
	Executavel string
}

// ErrProcessosNaoVerificados diz que esta plataforma nao lista processos. O
// `doctor` escreve isso em vez de fingir que nao ha nenhum.
var ErrProcessosNaoVerificados = errors.New("listagem de processos nao verificada nesta plataforma")

// SemPresenca devolve os processos do sistema que nao registraram presenca,
// tirando o proprio processo que pergunta -- o `doctor` tambem e um
// gobsidian.exe.
func SemPresenca(processos []ProcessoDoSistema, vivos []Presenca, proprio int) []ProcessoDoSistema {
	registrados := make(map[int]bool, len(vivos))
	for _, p := range vivos {
		registrados[p.PID] = true
	}
	var saida []ProcessoDoSistema
	for _, p := range processos {
		if p.PID == proprio || registrados[p.PID] {
			continue
		}
		saida = append(saida, p)
	}
	return saida
}
