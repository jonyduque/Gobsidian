package instalar

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

// Cofre e um cofre que o Obsidian conhece.
type Cofre struct {
	Caminho string
	Aberto  bool
}

// registroDoObsidian espelha o obsidian.json, reduzido ao que importa.
type registroDoObsidian struct {
	Vaults map[string]struct {
		Path string `json:"path"`
		Open bool   `json:"open"`
		TS   int64  `json:"ts"`
	} `json:"vaults"`
}

// CofresDoObsidian le o registro que o proprio Obsidian mantem.
//
// Ler dali evita pedir ao usuario um caminho que ele teria de ir buscar -- o
// instalador antigo ja fazia isso, e o comentario dele dizia exatamente isso.
//
// Cofre cujo diretorio nao existe mais e DESCARTADO: o registro do Obsidian
// guarda cofres que o usuario ja apagou, e oferecer um deles faria a instalacao
// terminar apontando para o nada.
func CofresDoObsidian(caminhoDoRegistro string) ([]Cofre, error) {
	b, err := os.ReadFile(caminhoDoRegistro)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil // Obsidian nao instalado: nao e erro.
		}
		return nil, fmt.Errorf("lendo %s: %w", caminhoDoRegistro, err)
	}

	var reg registroDoObsidian
	if err := json.Unmarshal(b, &reg); err != nil {
		return nil, fmt.Errorf("%s nao e um JSON valido: %w", caminhoDoRegistro, err)
	}

	var cofres []Cofre
	for _, v := range reg.Vaults {
		if v.Path == "" {
			continue
		}
		if fi, err := os.Stat(v.Path); err != nil || !fi.IsDir() {
			continue
		}
		cofres = append(cofres, Cofre{Caminho: v.Path, Aberto: v.Open})
	}

	// Ordem estavel: a lista vai para a tela e o usuario escolhe por numero.
	// Ordem que muda entre execucoes faz alguem escolher o cofre errado. O
	// aberto vem primeiro porque e o palpite mais provavel.
	sort.SliceStable(cofres, func(i, j int) bool {
		if cofres[i].Aberto != cofres[j].Aberto {
			return cofres[i].Aberto
		}
		return cofres[i].Caminho < cofres[j].Caminho
	})
	return cofres, nil
}

// CaminhoDoRegistroDoObsidian e onde o Obsidian guarda a lista de cofres.
func CaminhoDoRegistroDoObsidian() string {
	return filepath.Join(diretorioDeConfigDoObsidian(), "obsidian.json")
}

// TerminalInterativo diz se ha um humano do outro lado.
//
// E a guarda da decisao D-11: `gobsidian` sem argumentos so se autoinstala num
// terminal. Sem ela, um host MCP que invocasse o binario sem argumento --
// nenhum faz hoje, mas o custo de supor que nunca farao e uma instalacao
// disparada por engano no meio de uma sessao.
//
// O criterio e o stdin ser um DISPOSITIVO DE CARACTERE. Redirecionado de
// arquivo ou de pipe -- que e como todo host MCP o entrega -- ele nao e.
func TerminalInterativo() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}
