package console

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/jonyduque/Gobsidian/internal/textos"
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
//
// Os rotulos ("Uso:", "Flags:") vem de textos por funcao do template, e nao
// escritos aqui: o template e codigo, e a redacao mora num arquivo so.
const helpTemplate = `
{{if .HasParent}}{{else if .Short}}{{tituloForte .Short}}

{{end}}{{if .Runnable}}{{subtitulo txtUso}}
  {{destaque .UseLine}}

{{end}}{{if .HasAvailableSubCommands}}{{subtitulo txtComandos}}
{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}  {{forte (rpad .Name .NamePadding)}}  {{leve .Short}}
{{end}}{{end}}
{{end}}{{if .HasAvailableLocalFlags}}{{subtitulo txtFlags}}
{{.LocalFlags.FlagUsages | realcaFlags | trimTrailingWhitespaces}}
{{end}}{{if .HasAvailableSubCommands}}
{{detalhes .CommandPath}}
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
	// estilo desenha a marcacao do texto DENTRO do papel de cor: um resumo de
	// comando com "**gobsidian**" sai apagado, com a palavra em negrito, e o
	// apagado religado depois dela.
	estilo := func(codes []string) func(string) string {
		return func(t string) string {
			s := saidaDeAjuda(root)
			return s.style(s.marcar(adaptarTexto(t), codes...), codes...)
		}
	}

	// Os mesmos papeis de cor da interface interativa, e nao um segundo
	// conjunto: a ajuda e a instalacao aparecem na mesma tela, e duas paletas
	// fariam "titulo" ter duas cores dependendo de qual comando imprimiu.
	cobra.AddTemplateFunc("tituloForte", estilo(corTitulo))
	cobra.AddTemplateFunc("subtitulo", estilo(corDestaque))
	cobra.AddTemplateFunc("destaque", estilo(corPergunta))
	cobra.AddTemplateFunc("forte", estilo([]string{codeBold}))
	cobra.AddTemplateFunc("leve", estilo([]string{codeDim}))

	cobra.AddTemplateFunc("txtUso", func() string { return textos.AjudaUso })
	cobra.AddTemplateFunc("txtComandos", func() string { return textos.AjudaComandos })
	cobra.AddTemplateFunc("txtFlags", func() string { return textos.AjudaFlags })
	cobra.AddTemplateFunc("detalhes", func(caminho string) string {
		return saidaDeAjuda(root).Marcado(fmt.Sprintf(textos.AjudaDetalhes, caminho))
	})

	// A descricao de cada flag leva marcacao, e ela e desenhada com ou sem
	// cor: sem cor, "*PATH*" precisa virar "PATH", e nao sair com os
	// asteriscos. So o nome da flag ganha negrito, e so com cor.
	cobra.AddTemplateFunc("realcaFlags", func(usos string) string {
		s := saidaDeAjuda(root)
		linhas := strings.Split(usos, "\n")
		for i, linha := range linhas {
			m := reFlag.FindStringSubmatch(linha)
			if len(m) != 5 {
				continue
			}
			linhas[i] = m[1] + s.Bold(m[2]) + m[3] + s.Marcado(m[4])
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
			cmd.Short = textos.ResumoHelp
		case "completion":
			cmd.Short = textos.ResumoCompletion
		}
	}

	root.InitDefaultHelpFlag()
	if f := root.Flags().Lookup("help"); f != nil {
		f.Usage = fmt.Sprintf(textos.FlagHelp, root.Name())
	}
}

// TirarMarcacaoDaArvore tira negrito e italico do resumo de cada comando e da
// descricao de cada flag.
//
// Existe para o autocompletar: o shell mostra esses textos ao lado de cada
// comando e de cada flag, e quem os escreve e o cobra, direto, sem passar por
// um Stream -- "**gobsidian**" apareceria com os asteriscos. Quem chama so o
// faz quando o comando pedido e o de completar; a ajuda continua desenhando a
// marcacao.
func TirarMarcacaoDaArvore(cmd *cobra.Command) {
	cmd.Short = SemMarcacao(cmd.Short)
	cmd.Long = SemMarcacao(cmd.Long)
	cmd.Flags().VisitAll(func(f *pflag.Flag) { f.Usage = SemMarcacao(f.Usage) })
	cmd.PersistentFlags().VisitAll(func(f *pflag.Flag) { f.Usage = SemMarcacao(f.Usage) })
	for _, c := range cmd.Commands() {
		TirarMarcacaoDaArvore(c)
	}
}
