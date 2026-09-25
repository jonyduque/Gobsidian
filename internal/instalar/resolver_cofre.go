package instalar

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/jonyduque/Gobsidian/internal/text"
	"github.com/jonyduque/Gobsidian/internal/textos"
)

// ErroDeCofre e a recusa de um --vault dado pelo nome. Carrega os candidatos
// para quem quiser mostra-los de outro jeito; a mensagem ja os lista.
type ErroDeCofre struct {
	Valor string
	// Candidatos sao os cofres do registro que casaram o nome: mais de um e
	// ambiguidade; um so, com NoDiretorio preenchido, e conflito com o cwd.
	Candidatos []string
	// NoDiretorio e a pasta homonima no diretorio corrente, quando ela e o
	// cofre do registro apontam para lugares diferentes.
	NoDiretorio string
	// Conhecidos sao os nomes do registro, quando nada casou.
	Conhecidos []string
	// SemRegistro diz que o obsidian.json nao existe.
	SemRegistro bool
	Registro    string
}

func (e *ErroDeCofre) Error() string {
	switch {
	case len(e.Candidatos) > 1:
		return fmt.Sprintf(textos.ErroCofreAmbiguo, e.Valor, strings.Join(e.Candidatos, ", "))
	case e.NoDiretorio != "":
		return fmt.Sprintf(textos.ErroCofreNomeEPasta, e.Valor, e.Candidatos[0], e.NoDiretorio)
	case e.SemRegistro:
		return fmt.Sprintf(textos.ErroCofreSemRegistro, e.Valor, e.Registro)
	}
	conhecidos := textos.CofreNenhumConhecido
	if len(e.Conhecidos) > 0 {
		conhecidos = strings.Join(e.Conhecidos, ", ")
	}
	return fmt.Sprintf(textos.ErroCofreDesconhecido, e.Valor, conhecidos)
}

// ResolverCofre decide se o valor de --vault e um caminho ou o nome de um
// cofre do Obsidian, e devolve o caminho. porNome diz qual dos dois foi.
//
// A regra, fechada na Parte J do plano de 2026-09-11:
//
//  1. Valor com separador, absoluto, com unidade, ou comecando por "." ou "~"
//     e caminho, devolvido intacto. Nunca consulta o registro.
//  2. Palavra solta e nome: casa o nome da pasta de cada cofre do registro
//     pela chave de caminho do indice (NFC + caixa). Sem tirar acento:
//     "Revisao" e "Revisão" podem ser dois cofres, e o servidor nao escolhe.
//  3. Nome que casa um cofre e tambem existe como pasta em cwd, em outro
//     lugar, e erro: um host MCP roda com diretorio corrente arbitrario, e
//     escolher em silencio e o defeito.
//  4. Nome que casa mais de um cofre e erro com os caminhos.
//  5. Nome sem cofre que existe como pasta em cwd e caminho, como antes.
//  6. Nome que nao casa nada e erro com os nomes conhecidos, ou dizendo que
//     o registro nao existe.
//
// cwd vazio desliga as regras 3 e 5: quem le a entrada de um host nao sabe
// em que diretorio o host a roda.
//
// A resolucao mora aqui, e nao em config, porque config e folha e ler o
// registro e a conta de CofresDoObsidian.
func ResolverCofre(valor, registro, cwd string) (caminho string, porNome bool, err error) {
	if valor == "" || pareceCaminho(valor) {
		return valor, false, nil
	}

	local := ""
	if cwd != "" {
		if p := filepath.Join(cwd, valor); ehDiretorio(p) {
			local = p
		}
	}

	cofres, errReg := CofresDoObsidian(registro)
	if errReg != nil {
		if local != "" {
			return valor, false, nil
		}
		return "", false, errReg
	}

	var casam, conhecidos []string
	for _, g := range CofresPorNome(cofres) {
		conhecidos = append(conhecidos, g.Nome)
		if g.Chave == text.ChaveDeCaminho(valor) {
			casam = g.Caminhos
		}
	}

	switch {
	case len(casam) > 1:
		return "", false, &ErroDeCofre{Valor: valor, Candidatos: casam}
	case len(casam) == 1:
		if local != "" && !mesmoLugar(local, casam[0]) {
			return "", false, &ErroDeCofre{Valor: valor, Candidatos: casam, NoDiretorio: local}
		}
		return casam[0], true, nil
	case local != "":
		return valor, false, nil
	}

	e := &ErroDeCofre{Valor: valor, Registro: registro}
	if _, err := os.Stat(registro); registro == "" || errors.Is(err, fs.ErrNotExist) {
		e.SemRegistro = true
		return "", false, e
	}
	e.Conhecidos = conhecidos
	return "", false, e
}

// NomeDeCofre agrupa os cofres do registro que respondem pelo mesmo nome.
type NomeDeCofre struct {
	// Nome e a grafia da pasta do primeiro caminho do grupo.
	Nome string
	// Chave e o nome pela chave de caminho do indice: NFC e caixa.
	Chave string
	// Caminhos, ordenados. Mais de um e um nome ambiguo.
	Caminhos []string
}

// CofresPorNome e a conta unica de "que nome responde por que cofre": o nome
// e a pasta do cofre, comparada pela chave de caminho do indice. ResolverCofre
// e a completacao de --vault usam esta, para que um nome que a completacao
// oferece seja sempre um nome que --vault aceita. Ordenado por Nome.
func CofresPorNome(cofres []Cofre) []NomeDeCofre {
	porChave := map[string]*NomeDeCofre{}
	for _, c := range cofres {
		nome := filepath.Base(c.Caminho)
		chave := text.ChaveDeCaminho(nome)
		g, ok := porChave[chave]
		if !ok {
			g = &NomeDeCofre{Nome: nome, Chave: chave}
			porChave[chave] = g
		}
		g.Caminhos = append(g.Caminhos, c.Caminho)
	}
	grupos := make([]NomeDeCofre, 0, len(porChave))
	for _, g := range porChave {
		slices.Sort(g.Caminhos)
		g.Nome = filepath.Base(g.Caminhos[0])
		grupos = append(grupos, *g)
	}
	slices.SortFunc(grupos, func(a, b NomeDeCofre) int { return strings.Compare(a.Nome, b.Nome) })
	return grupos
}

// pareceCaminho e a regra 1: o que o usuario so escreveria querendo um
// caminho.
func pareceCaminho(v string) bool {
	return strings.ContainsAny(v, `/\`) ||
		filepath.IsAbs(v) ||
		filepath.VolumeName(v) != "" ||
		strings.HasPrefix(v, ".") ||
		strings.HasPrefix(v, "~")
}

func ehDiretorio(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.IsDir()
}

// mesmoLugar compara dois caminhos pela chave de caminho do indice, depois de
// torna-los absolutos: no Windows, caixa e forma Unicode nao separam pastas.
func mesmoLugar(a, b string) bool {
	return text.ChaveDeCaminho(CaminhoCanonicoDeCofre(filepath.Clean(a))) ==
		text.ChaveDeCaminho(CaminhoCanonicoDeCofre(filepath.Clean(b)))
}
