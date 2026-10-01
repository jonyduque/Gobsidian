package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jonyduque/Gobsidian/internal/config"
	"github.com/jonyduque/Gobsidian/internal/mcpsrv"
	"github.com/jonyduque/Gobsidian/internal/textos"
)

// parametroDeFlag e um parametro da tool registrado como flag, com o jeito de
// ler o valor passado.
type parametroDeFlag struct {
	param mcpsrv.Parametro
	flag  string
	ler   func() (any, error)
}

// nomeDaFlag e o nome do parametro com hifen no lugar de sublinhado:
// heading_level vira --heading-level. Uma conta so, que o teste de cobertura
// tambem usa.
func nomeDaFlag(parametro string) string {
	return strings.ReplaceAll(parametro, "_", "-")
}

// registrarParametros transforma cada parametro do schema numa flag, pelo
// tipo:
//
//	string            texto
//	integer           inteiro
//	number            numero
//	boolean           booleano
//	array de string   repetivel: --tags a --tags b, ou --tags a,b
//	object            repetivel: --frontmatter chave=valor
//
// Parametro de tipo que nao cabe numa flag fica sem ela, e o teste de
// cobertura reprova: a saida para ele e --args, mas isso tem de ser decidido,
// e nao acontecer em silencio.
func registrarParametros(cmd *cobra.Command, e mcpsrv.EsquemaDeTool, pular map[string]bool) []*parametroDeFlag {
	var saida []*parametroDeFlag
	for _, p := range e.Parametros {
		if pular[p.Nome] {
			continue
		}
		pf := &parametroDeFlag{param: p, flag: nomeDaFlag(p.Nome)}
		uso := descricaoDaFlag(p)
		fs := cmd.Flags()

		switch p.Tipo {
		case "string":
			var v string
			fs.StringVar(&v, pf.flag, "", uso)
			pf.ler = func() (any, error) { return v, nil }
		case "integer":
			var v int64
			fs.Int64Var(&v, pf.flag, 0, uso)
			pf.ler = func() (any, error) { return v, nil }
		case "number":
			var v float64
			fs.Float64Var(&v, pf.flag, 0, uso)
			pf.ler = func() (any, error) { return v, nil }
		case "boolean":
			var v bool
			fs.BoolVar(&v, pf.flag, false, uso)
			pf.ler = func() (any, error) { return v, nil }
		case "array":
			if p.TipoDoItem != "string" {
				continue
			}
			var v []string
			fs.StringSliceVar(&v, pf.flag, nil, uso)
			pf.ler = func() (any, error) {
				lista := make([]any, 0, len(v))
				for _, s := range v {
					lista = append(lista, s)
				}
				return lista, nil
			}
		case "object":
			var v []string
			fs.StringArrayVar(&v, pf.flag, nil, uso+textos.FlagFrontmatter)
			pf.ler = func() (any, error) { return paresChaveValor(v) }
		default:
			continue
		}
		saida = append(saida, pf)
	}
	return saida
}

// descricaoDaFlag e a descricao do parametro no schema -- o mesmo texto que o
// modelo le --, com os valores aceitos quando ha enum.
func descricaoDaFlag(p mcpsrv.Parametro) string {
	uso := p.Descricao
	if uso == "" {
		uso = fmt.Sprintf(textos.FlagSemDescricao, p.Nome)
	}
	if len(p.Enum) > 0 {
		valores := make([]string, 0, len(p.Enum))
		for _, v := range p.Enum {
			valores = append(valores, "`"+v+"`")
		}
		uso += " (" + strings.Join(valores, ", ") + ")"
	}
	switch {
	case p.Nome == "content":
		uso += textos.FlagContentStdin
	case p.ItemAceitaObjeto:
		uso += textos.FlagPathsComoArgs
	}
	return uso
}

// paresChaveValor le --frontmatter chave=valor. O valor vai como texto: um
// valor tipado (lista, numero) vai por --args.
func paresChaveValor(pares []string) (map[string]any, error) {
	m := make(map[string]any, len(pares))
	for _, par := range pares {
		chave, valor, ok := strings.Cut(par, "=")
		if !ok || chave == "" {
			return nil, fmt.Errorf(textos.ErroFrontmatterFlag, par)
		}
		m[chave] = valor
	}
	return m, nil
}

// flagsDeCofreDaCLI e flagsDeCofre com GOBSIDIAN_VAULT como padrao de --vault.
//
// So para comando de CLI. `serve` e `daemon` registram flagsDeCofre, sem o
// padrao: um host MCP herda o ambiente do usuario, e um --vault esquecido no
// config do host serviria em silencio o cofre da variavel -- e dois hosts sem
// --vault serviriam o mesmo cofre por acidente.
// TestGobsidianVaultValeNaCLIENaoNoServe prova as duas metades.
func flagsDeCofreDaCLI(cmd *cobra.Command, f *config.Flags) {
	flagsDeCofre(cmd, f)
	if padrao := os.Getenv(varDoCofrePadrao); padrao != "" {
		fl := cmd.Flags().Lookup("vault")
		fl.DefValue = padrao
		_ = fl.Value.Set(padrao)
	}
}

// varDoCofrePadrao e a variavel do cofre padrao da CLI.
const varDoCofrePadrao = "GOBSIDIAN_VAULT"
