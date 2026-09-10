package instalar

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// NomeDoManifesto e o arquivo que registra o que a instalacao fez.
//
// Ele mora na RAIZ DO CACHE, e nao ao lado do binario, e a razao e a pergunta
// que ele precisa responder: "este executavel que esta rodando agora esta
// instalado?" (decisao D-11). Um manifesto ao lado do binario so seria
// encontrado por quem ja sabe onde o binario foi instalado -- exatamente o que
// nao se sabe quando o usuario baixa o executavel e clica nele.
const NomeDoManifesto = "instalacao.json"

// Manifesto e o que a instalacao deixou para trás.
//
// Sem ele, `update` adivinha o que reconfigurar e a deteccao de D-11 nao tem
// resposta mecanica. Com ele, `update` sabe dizer "voce instalou para 3 hosts,
// vou reconfigurar os 3".
type Manifesto struct {
	Binario        string   `json:"binario"`
	Versao         string   `json:"versao"`
	Hash           string   `json:"hash"`
	PathAdicionado string   `json:"path_adicionado,omitempty"`
	Hosts          []string `json:"hosts,omitempty"`
	// Cofre e o formato antigo, de um cofre so. Continua sendo LIDO para que
	// um update a partir de uma instalacao anterior nao perca a configuracao.
	Cofre string `json:"cofre,omitempty"`
	// Cofres e o formato atual, desde 2026-09-09.
	Cofres []string  `json:"cofres,omitempty"`
	Em     time.Time `json:"em"`
}

// CaminhoDoManifesto e a conta unica do caminho.
func CaminhoDoManifesto() string {
	return filepath.Join(RaizDoCache(), NomeDoManifesto)
}

// ErrSemManifesto indica que nunca houve instalacao nesta maquina.
//
// Sentinela: "nunca instalou" e "nao consegui ler" pedem respostas diferentes
// -- a primeira leva a autoinstalacao, a segunda a uma mensagem de erro.
var ErrSemManifesto = errors.New("nao ha manifesto de instalacao")

// LerManifesto devolve o manifesto da instalacao corrente.
func LerManifesto() (Manifesto, error) {
	b, err := os.ReadFile(CaminhoDoManifesto())
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Manifesto{}, ErrSemManifesto
		}
		return Manifesto{}, fmt.Errorf("lendo manifesto: %w", err)
	}
	var m Manifesto
	if err := json.Unmarshal(b, &m); err != nil {
		return Manifesto{}, fmt.Errorf("manifesto ilegivel em %s: %w", CaminhoDoManifesto(), err)
	}
	return m, nil
}

// GravarManifesto substitui o manifesto.
func GravarManifesto(m Manifesto) error {
	caminho := CaminhoDoManifesto()
	if err := os.MkdirAll(filepath.Dir(caminho), 0o700); err != nil {
		return fmt.Errorf("criando diretorio do manifesto: %w", err)
	}
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("serializando manifesto: %w", err)
	}
	if err := os.WriteFile(caminho, append(b, '\n'), 0o600); err != nil {
		return fmt.Errorf("gravando manifesto: %w", err)
	}
	return nil
}

// EstaInstalado responde a pergunta da decisao D-11: o executavel em curso e
// uma instalacao, ou e um download solto que alguem acabou de rodar?
//
// A resposta e MECANICA: existe manifesto E o executavel corrente esta no
// caminho que ele registra. Comparar caminho e text.ChaveDeCaminho no resto do
// produto; aqui a comparacao e por os.SameFile, que e mais forte -- ela
// atravessa link, junction e diferenca de grafia sem depender de normalizacao.
//
// Erro ao perguntar devolve (false, err) e quem chama decide. Nao se
// autoinstala por nao ter conseguido ler algo.
func EstaInstalado() (bool, Manifesto, error) {
	m, err := LerManifesto()
	if err != nil {
		return false, Manifesto{}, err
	}
	if m.Binario == "" {
		return false, m, nil
	}

	atual, err := os.Executable()
	if err != nil {
		return false, m, fmt.Errorf("resolvendo o executavel corrente: %w", err)
	}

	infoAtual, err := os.Stat(atual)
	if err != nil {
		return false, m, fmt.Errorf("consultando o executavel corrente: %w", err)
	}
	infoManifesto, err := os.Stat(m.Binario)
	if err != nil {
		// O manifesto aponta para um binario que nao existe mais: a instalacao
		// foi removida a mao. Nao instalado, e sem erro -- o estado e
		// conhecido, so nao e o esperado.
		return false, m, nil
	}
	return os.SameFile(infoAtual, infoManifesto), m, nil
}
