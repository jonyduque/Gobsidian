package daemon

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPistaDoLogDistingueAusenteDeMorteNaMontagem e o teste que sustenta a
// mensagem de prazo estourado.
//
// Sao dois defeitos DIFERENTES com o mesmo sintoma para quem so olha o socket,
// e ambos ocorreram na maquina do dono em 2026-08-26:
//
//   - log AUSENTE: o processo do daemon nao chegou a escrever nada. O problema
//     esta no spawn, nao no cofre. Foi o caso das tres sessoes que pagaram 10 s
//     em toda partida durante dias.
//   - log com "daemon iniciado" e MAIS NADA: ele subiu e morreu na montagem do
//     servico. Foi o caso dos dois cofres com caminho inexistente.
//
// Sem a distincao, "socket do daemon nao respondeu em 10s" culpa o socket --
// o unico lugar onde a resposta NAO esta.
func TestPistaDoLogDistingueAusenteDeMorteNaMontagem(t *testing.T) {
	// Cofre ficticio: nao precisa existir, so precisa gerar uma chave estavel.
	cofre := filepath.Join(t.TempDir(), "cofre-de-teste")

	path, err := CaminhoDoLog(cofre)
	if err != nil {
		t.Fatalf("CaminhoDoLog: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("criando diretorio de runtime: %v", err)
	}
	// O log deste cofre ficticio nao pertence a ninguem mais; remover no fim
	// evita deixar residuo no diretorio de runtime real do usuario.
	t.Cleanup(func() { _ = os.Remove(path) })
	_ = os.Remove(path)

	t.Run("ausente", func(t *testing.T) {
		pista := pistaDoLog(cofre)
		if !strings.Contains(pista, "nao existe") {
			t.Errorf("pista = %q; queria dizer que o log nao existe", pista)
		}
		if !strings.Contains(pista, "spawn") {
			t.Errorf("pista = %q; queria apontar o spawn como suspeito", pista)
		}
	})

	t.Run("morreu na montagem", func(t *testing.T) {
		conteudo := `time=2026-08-24T12:56:46.418-03:00 level=INFO msg="daemon iniciado" vault=X` + "\n"
		if err := os.WriteFile(path, []byte(conteudo), 0o600); err != nil {
			t.Fatalf("escrevendo log: %v", err)
		}
		pista := pistaDoLog(cofre)
		if strings.Contains(pista, "nao existe") {
			t.Errorf("pista = %q; o log EXISTE neste caso", pista)
		}
		if !strings.Contains(pista, "daemon iniciado") {
			t.Errorf("pista = %q; queria trazer a ultima linha do log", pista)
		}
	})
}

// TestUltimasLinhasDoLogDevolveOFim confere o recorte e o sinal de existencia,
// que e o que separa os dois diagnosticos acima.
func TestUltimasLinhasDoLogDevolveOFim(t *testing.T) {
	cofre := filepath.Join(t.TempDir(), "outro-cofre")
	path, err := CaminhoDoLog(cofre)
	if err != nil {
		t.Fatalf("CaminhoDoLog: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("criando diretorio de runtime: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(path) })

	if _, existe := UltimasLinhasDoLog(cofre, 3); existe {
		t.Fatal("existe = true para log que ainda nao foi criado")
	}

	if err := os.WriteFile(path, []byte("a\nb\n\nc\nd\n"), 0o600); err != nil {
		t.Fatalf("escrevendo log: %v", err)
	}

	linhas, existe := UltimasLinhasDoLog(cofre, 3)
	if !existe {
		t.Fatal("existe = false para log presente")
	}
	querido := []string{"b", "c", "d"}
	if len(linhas) != len(querido) {
		t.Fatalf("linhas = %v, queria %v", linhas, querido)
	}
	for i := range querido {
		if linhas[i] != querido[i] {
			t.Errorf("linha %d = %q, queria %q", i, linhas[i], querido[i])
		}
	}
}
