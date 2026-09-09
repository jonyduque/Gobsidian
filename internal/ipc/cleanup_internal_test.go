// cleanup_internal_test.go testa cleanupSocketFile de DENTRO do pacote: as
// variaveis removerArquivo e renomearArquivo nao sao exportadas, e o teste
// precisa troca-las para alcancar o ramo de falha que producao exibiu 20+ vezes
// e que nenhum reprodutor conhecido produz. ipc_test.go fica no pacote externo
// (ipc_test), que so enxerga a superficie publica.

package ipc

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// textoDeCampo1920 e a mensagem EXATA que o Windows devolveu em producao, 20+
// vezes desde 2026-09-01, para ERROR_CANT_ACCESS_FILE (1920). Fica numa
// constante, e nao inline, por duas razoes: ela e o DADO do defeito -- copiar o
// texto de campo e o que torna o teste reconhecivel para quem o reencontrar no
// log --, e escrita como literal dentro de errors.New ela reprova em revive
// (error-strings: maiuscula inicial e ponto final), que e uma regra sobre erros
// que NOS escrevemos, nao sobre os que reproduzimos.
const textoDeCampo1920 = "The file cannot be accessed by the system."

// TestCleanupSocketFilePlanoB cobre as 20+ falhas medidas em producao desde
// 2026-09-01 no log do cofre Estudo: os.Remove devolveu "The file cannot be
// accessed by the system" (ERROR_CANT_ACCESS_FILE, 1920), o daemon nao subiu, e
// TODA ponte do cofre caiu para o modo em processo -- o estado em que duas
// instancias gravam o MESMO cache de busca.
//
// O mecanismo desse estado NAO foi reproduzido. Dois cenarios foram medidos em
// 2026-09-08 e nenhum produz o par observado em campo (dial 10022 / remove
// 1920):
//
//	listener fechado limpo   dial 10061   arquivo ja nao existe (Close desvincula)
//	processo morto a forca   dial 10061   os.Remove com SUCESSO
//
// Por isso o conserto nao trata errno -- classificar por numero e o que o
// comentario de Listen ja proibe. E uma SAIDA: se o caminho nao se apaga, tira
// o arquivo do caminho por rename, que o Windows permite ate sobre executavel
// em uso (medido em 2026-09-08).
func TestCleanupSocketFilePlanoB(t *testing.T) {
	caminho := filepath.Join(t.TempDir(), "p.sock")
	if err := os.WriteFile(caminho, nil, 0o600); err != nil {
		t.Fatal(err)
	}

	originalRemover := removerArquivo
	removerArquivo = func(string) error { return errors.New(textoDeCampo1920) }
	t.Cleanup(func() { removerArquivo = originalRemover })

	if err := cleanupSocketFile(caminho); err != nil {
		t.Fatalf("o plano B nao liberou o caminho, entao o daemon nao sobe: %v", err)
	}
	if _, err := os.Lstat(caminho); !os.IsNotExist(err) {
		t.Fatalf("o caminho continua ocupado depois do plano B: %v", err)
	}
}

// TestCleanupSocketFileRelataOEstado: quando NEM o plano B funciona, o erro tem
// de dizer o que havia no caminho. Ate 2026-09-08 ele devolvia so o texto do
// sistema, e quem lia nao sabia se havia arquivo, diretorio ou nada -- as tres
// coisas produzem mensagens de sistema parecidas no Windows.
//
// As duas operacoes sao forcadas a falhar de proposito. Uma versao anterior
// deste teste usava um diretorio de verdade e deixava o rename acontecer; como
// o rename FUNCIONA num diretorio temporario, o teste saia cedo e nao afirmava
// nada -- um teste que nao pode falhar.
func TestCleanupSocketFileRelataOEstado(t *testing.T) {
	caminho := filepath.Join(t.TempDir(), "p.sock")
	if err := os.WriteFile(caminho, []byte("doze bytes!!"), 0o600); err != nil {
		t.Fatal(err)
	}

	originalRemover, originalRenomear := removerArquivo, renomearArquivo
	removerArquivo = func(string) error { return errors.New("recusado pelo teste") }
	renomearArquivo = func(string, string) error { return errors.New("rename tambem recusado") }
	t.Cleanup(func() {
		removerArquivo, renomearArquivo = originalRemover, originalRenomear
	})

	err := cleanupSocketFile(caminho)
	if err == nil {
		t.Fatal("as duas operacoes falharam e cleanupSocketFile devolveu nil")
	}
	if !strings.Contains(err.Error(), "recusado pelo teste") {
		t.Errorf("o erro perdeu a causa original: %v", err)
	}
	if !strings.Contains(err.Error(), "estado do caminho") {
		t.Errorf("o erro nao relata o estado encontrado, que e o que faltava em campo: %v", err)
	}
	if !strings.Contains(err.Error(), "tamanho=12") {
		t.Errorf("o relato nao descreve o arquivo que estava la: %v", err)
	}
}

// TestCleanupSocketFileAusenteNaoEhErro preserva o contrato que ja existia: um
// caminho livre e o caso normal, nao uma falha.
func TestCleanupSocketFileAusenteNaoEhErro(t *testing.T) {
	if err := cleanupSocketFile(filepath.Join(t.TempDir(), "nao-existe")); err != nil {
		t.Fatalf("caminho ausente virou erro: %v", err)
	}
}
