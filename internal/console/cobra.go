package console

import (
	"regexp"
	"strings"

	"github.com/spf13/cobra"
)

// reFlag casa uma linha de flag do cobra: indentacao, a forma curta e/ou
// longa, o espacamento de alinhamento, e a descricao. So o segundo grupo e
// realcado -- realcar a descricao junto tira o contraste que justifica
// realcar qualquer coisa.
var reFlag = regexp.MustCompile(`^(\s*)(-\S+(?:,\s*--\S+)?|--\S+)(\s+)(.*)$`)

// helpTemplate e o template de ajuda do cobra com os realces aplicados.
//
// As funcoes que ele chama estao registradas por SetupHelp e decidem sobre cor
// no momento da chamada, olhando a saida real do comando. Um template nao pode
// levar a decisao embutida: o mesmo binario imprime ajuda num terminal e num
// pipe, e quem redireciona `gobsidian --help > ajuda.txt` quer o arquivo
// limpo.
const helpTemplate = `
{{if .HasParent}}{{else if .Short}}{{tituloForte .Short}}

{{end}}{{if .Runnable}}{{secao "Uso:"}}
  {{destaque .UseLine}}

{{end}}{{if .HasAvailableSubCommands}}{{secao "Comandos disponiveis:"}}
{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}  {{forte (rpad .Name .NamePadding)}}  {{leve .Short}}
{{end}}{{end}}
{{end}}{{if .HasAvailableLocalFlags}}{{secao "Flags:"}}
{{.LocalFlags.FlagUsages | realcaFlags | trimTrailingWhitespaces}}
{{end}}{{if .HasAvailableSubCommands}}
Use {{destaque (printf "\"%s [comando] --help\"" .CommandPath)}} para detalhes de um comando.
{{end}}
`

// SetupHelp instala o template de ajuda formatado na arvore de comandos.
//
// A decisao de cor sai de saidaDeAjuda(cmd), nao de os.Stdout: o cobra escreve
// a ajuda no writer configurado no comando, e em teste esse writer e um
// buffer. Amarrar a decisao a os.Stdout faria o teste receber sequencias ANSI
// que ele nao pediu, e -- pior -- faria uma ajuda redirecionada sair suja
// enquanto o teste passava.
func SetupHelp(root *cobra.Command) {
	registraFuncoes(root)
	root.SetHelpTemplate(helpTemplate)
	traduzComandosNativos(root)
}

// saidaDeAjuda devolve um Stream para onde a ajuda de root vai de fato sair.
func saidaDeAjuda(root *cobra.Command) *Stream {
	return New(root.OutOrStdout())
}

func registraFuncoes(root *cobra.Command) {
	estilo := func(fn func(*Stream, string) string) func(string) string {
		return func(s string) string { return fn(saidaDeAjuda(root), s) }
	}

	cobra.AddTemplateFunc("tituloForte", estilo(func(s *Stream, t string) string {
		return s.style(t, codeBold, codeCyan)
	}))
	cobra.AddTemplateFunc("secao", estilo(func(s *Stream, t string) string {
		return s.style(t, codeBold, codeYellow)
	}))
	cobra.AddTemplateFunc("destaque", estilo(func(s *Stream, t string) string {
		return s.style(t, codeCyan)
	}))
	cobra.AddTemplateFunc("forte", estilo(func(s *Stream, t string) string {
		return s.Bold(t)
	}))
	cobra.AddTemplateFunc("leve", estilo(func(s *Stream, t string) string {
		return s.Dim(t)
	}))

	cobra.AddTemplateFunc("realcaFlags", func(usos string) string {
		s := saidaDeAjuda(root)
		if !s.Colored() {
			return usos
		}
		linhas := strings.Split(usos, "\n")
		for i, linha := range linhas {
			m := reFlag.FindStringSubmatch(linha)
			if len(m) != 5 {
				continue
			}
			linhas[i] = m[1] + s.Bold(m[2]) + m[3] + m[4]
		}
		return strings.Join(linhas, "\n")
	})
}

// traduzComandosNativos poe em portugues os dois comandos que o cobra cria
// sozinho. Eles so existem depois de InitDefault*, que o cobra chamaria mais
// tarde -- forcar aqui e o que torna possivel alcanca-los.
func traduzComandosNativos(root *cobra.Command) {
	root.InitDefaultHelpCmd()
	root.InitDefaultCompletionCmd()

	for _, cmd := range root.Commands() {
		switch cmd.Name() {
		case "help":
			cmd.Short = "Exibe a ajuda sobre qualquer comando"
		case "completion":
			cmd.Short = "Gera o script de autocompletar do shell"
		}
	}

	root.InitDefaultHelpFlag()
	if f := root.Flags().Lookup("help"); f != nil {
		f.Usage = "Exibe a ajuda do " + root.Name()
	}
}
