package doctor

import (
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jonyd/gobsidian/internal/config"
)

// TestClasseDoCaminhoDoSocketDistingueOsEstados e o teste que a investigacao de
// 2026-08-26 exigiu.
//
// A mensagem de erro do dial NAO distingue estes estados: medido no Windows,
// arquivo comum, socket orfao de dono morto a forca e caminho inexistente
// devolvem os TRES o mesmo 10061, enquanto diretorio devolve 10022. Um
// diagnostico que so repetisse o erro do sistema seria inutil exatamente no
// caso que apareceu em producao.
//
// Por isso cada estado tem um caso proprio, e cada caso afirma o TEXTO, nao so
// o status: quem le o relatorio precisa saber o que remover.
func TestClasseDoCaminhoDoSocketDistingueOsEstados(t *testing.T) {
	dir := t.TempDir()

	ausente := filepath.Join(dir, "ausente.sock")

	comum := filepath.Join(dir, "comum.sock")
	if err := os.WriteFile(comum, nil, 0o600); err != nil {
		t.Fatalf("criando arquivo comum: %v", err)
	}

	diretorio := filepath.Join(dir, "diretorio.sock")
	if err := os.Mkdir(diretorio, 0o700); err != nil {
		t.Fatalf("criando diretorio: %v", err)
	}

	// Socket de verdade: bind e mantem o listener aberto ate o fim do teste.
	// Fechar removeria o arquivo (o Go desvincula no Close), e o caso deixaria
	// de existir.
	socketReal := filepath.Join(dir, "real.sock")
	ln, err := net.Listen("unix", socketReal)
	if err != nil {
		t.Skipf("socket unix indisponivel nesta maquina: %v", err)
	}
	defer func() { _ = ln.Close() }()

	casos := []struct {
		nome     string
		path     string
		esperado string
	}{
		{"ausente", ausente, "ausente"},
		{"arquivo comum", comum, "arquivo comum"},
		{"diretorio", diretorio, "DIRETORIO"},
		{"socket real", socketReal, "socket"},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			got := classeDoCaminhoDoSocket(c.path)
			if !strings.Contains(got, c.esperado) {
				t.Errorf("classe de %s = %q, queria conter %q", c.nome, got, c.esperado)
			}
		})
	}

	// A afirmacao que da sentido as outras: diretorio e arquivo comum NAO
	// podem produzir o mesmo texto. Se produzirem, o relatorio perdeu
	// justamente a distincao que o dial nao dava.
	if classeDoCaminhoDoSocket(diretorio) == classeDoCaminhoDoSocket(comum) {
		t.Error("diretorio e arquivo comum produziram a MESMA classe; a distincao sumiu")
	}
}

// TestVizinhosParecidosAchaGrafiaComAcento reproduz o defeito de campo: o
// config do host apontava para "Jurisprudencia" e no disco so existia
// "Jurisprudencia" com acento. As duas grafias produzem VaultKey diferente,
// logo socket, cache e daemon proprios, e o servidor morria na partida sem que
// nada dissesse o motivo -- por dois dias.
func TestVizinhosParecidosAchaGrafiaComAcento(t *testing.T) {
	dir := t.TempDir()
	comAcento := filepath.Join(dir, "Jurisprudência")
	if err := os.Mkdir(comAcento, 0o700); err != nil {
		t.Fatalf("criando diretorio com acento: %v", err)
	}

	semAcento := filepath.Join(dir, "Jurisprudencia")
	vizinhos := vizinhosParecidos(semAcento)

	if len(vizinhos) != 1 || vizinhos[0] != "Jurisprudência" {
		t.Errorf("vizinhos de %q = %v; queria [Jurisprudência]", semAcento, vizinhos)
	}
}

// TestCheckRootExistsApontaAGrafiaDoDisco reproduz o defeito de campo dentro da
// verificacao que de fato roda. A dica mora em checkRootExists porque ela e
// HALTING: uma checagem posterior seria abortada justamente no caso para o qual
// existiria -- foi o que aconteceu na primeira tentativa desta tarefa.
func TestCheckRootExistsApontaAGrafiaDoDisco(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "Revisão"), 0o700); err != nil {
		t.Fatalf("criando diretorio: %v", err)
	}

	r := checkRootExists(t.Context(), config.Config{VaultPath: filepath.Join(dir, "Revisao")})

	if r.Status != StatusFail {
		t.Errorf("status = %v, queria StatusFail", r.Status)
	}
	if !strings.Contains(r.Detail, "grafia diferente") {
		t.Errorf("detalhe nao oferece a grafia do disco: %q", r.Detail)
	}
	if !strings.Contains(r.Detail, "Revisão") {
		t.Errorf("detalhe nao nomeia o vizinho real: %q", r.Detail)
	}
}
