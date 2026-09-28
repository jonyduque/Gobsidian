package main

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jonyduque/Gobsidian/internal/console"
	"github.com/jonyduque/Gobsidian/internal/instalar"
	"github.com/jonyduque/Gobsidian/internal/textos"
)

// cofreListado e uma linha de `gobsidian vaults`, e tambem a forma do JSON.
type cofreListado struct {
	Nome        string `json:"name"`
	Caminho     string `json:"path"`
	Aberto      bool   `json:"open"`
	Configurado bool   `json:"configured"`
	NoObsidian  bool   `json:"in_obsidian"`
}

// cofresDoObsidianFn e instalar.CofresDoObsidian numa variavel, pelo mesmo
// motivo de configuracaoAtualFn: o teste nao le o registro da maquina.
var cofresDoObsidianFn = func() ([]instalar.Cofre, error) {
	return instalar.CofresDoObsidian(instalar.CaminhoDoRegistroDoObsidian())
}

// newVaultsCmd lista os cofres: os que o Obsidian conhece e os que estao
// configurados em algum host. Decisao do dono em 2026-09-27 -- `vaults` lista,
// `config` configura.
//
// Nao abre cofre nenhum e nao pede --vault: e a pergunta que se faz ANTES de
// escolher um.
func newVaultsCmd() *cobra.Command {
	var saida opcoesDeSaida

	cmd := &cobra.Command{
		Use:   "vaults",
		Short: textos.ResumoVaults,
		Args:  semArgumentoDeUso,
		RunE: func(cmd *cobra.Command, _ []string) error {
			emJSON, err := saida.emJSON(cmd)
			if err != nil {
				return err
			}

			doObsidian, errRegistro := cofresDoObsidianFn()
			cofres := listarCofres(doObsidian, configuracaoAtualFn())

			// vaults imprime em stdout de proposito: e comando de CLI.
			if emJSON {
				if cofres == nil {
					cofres = []cofreListado{}
				}
				b, err := json.Marshal(cofres)
				if err != nil {
					return err
				}
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(b))
				return nil
			}

			con := console.New(cmd.OutOrStdout())
			if errRegistro != nil {
				con.Warn(textos.InstallRegistroIlegivel, errRegistro)
			}
			if len(cofres) == 0 {
				con.Info("%s", textos.VaultsNenhum)
				return nil
			}
			con.Bloco(textos.VaultsTitulo, linhasDeCofres(con, cofres), textos.VaultsRodape)
			return nil
		},
	}
	saida.registrar(cmd)
	return cmd
}

// listarCofres une os cofres do Obsidian com os configurados nos hosts.
//
// A uniao e pelo caminho CANONICO dos dois lados, a mesma conta de
// escolherCofres: o registro do Obsidian guarda contrabarra e o config de um
// host guarda a grafia de quando foi configurado. Comparar a grafia crua faria
// cada cofre aparecer duas vezes -- foi o que o dono viu na lista do install
// em 2026-09-11. Um cofre configurado que o Obsidian nao conhece entra tambem:
// sumir da lista esconderia exatamente o caso que se quer ver.
func listarCofres(doObsidian []instalar.Cofre, configurados []string) []cofreListado {
	porCaminho := map[string]*cofreListado{}
	var ordem []string
	add := func(caminho string) *cofreListado {
		c := instalar.CaminhoCanonicoDeCofre(caminho)
		if l, ok := porCaminho[c]; ok {
			return l
		}
		l := &cofreListado{Nome: filepath.Base(c), Caminho: c}
		porCaminho[c] = l
		ordem = append(ordem, c)
		return l
	}
	for _, c := range doObsidian {
		l := add(c.Caminho)
		l.NoObsidian = true
		l.Aberto = l.Aberto || c.Aberto
	}
	for _, c := range configurados {
		add(c).Configurado = true
	}

	sort.SliceStable(ordem, func(i, j int) bool {
		return strings.ToLower(porCaminho[ordem[i]].Nome) < strings.ToLower(porCaminho[ordem[j]].Nome)
	})
	lista := make([]cofreListado, 0, len(ordem))
	for _, c := range ordem {
		lista = append(lista, *porCaminho[c])
	}
	if len(lista) == 0 {
		return nil
	}
	return lista
}

// linhasDeCofres monta o corpo da tabela: o nome e o estado numa linha, e o
// caminho apagado na de baixo.
//
// O caminho vai sozinho numa linha porque a moldura corta em 78 colunas, e um
// caminho de cofre passa disso facil: na primeira redacao ele vinha antes do
// estado, e "configurado" sumia cortado no fim da linha.
func linhasDeCofres(con *console.Stream, cofres []cofreListado) []string {
	linhas := make([]string, 0, 2*len(cofres))
	for _, c := range cofres {
		var estado []string
		if c.Aberto {
			estado = append(estado, textos.VaultsAberto)
		}
		if c.Configurado {
			estado = append(estado, con.Verde(textos.VaultsConfigurado))
		}
		if !c.NoObsidian {
			estado = append(estado, con.Amarelo(textos.VaultsForaDoObsidian))
		}
		linha := "  " + con.Bold(c.Nome)
		if len(estado) > 0 {
			linha += "  " + strings.Join(estado, ", ")
		}
		linhas = append(linhas, linha, "    "+con.Dim(c.Caminho))
	}
	return linhas
}
