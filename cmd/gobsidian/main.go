// Package main e o binario do gobsidian: `serve` levanta o servidor MCP
// sobre stdio, `doctor` diagnostica o ambiente e `version` imprime a
// identificacao do build.
//
// A divisao entre quem escreve em stdout e quem nao escreve vive aqui e e
// load-bearing: `serve` cede stdout inteiro ao JSON-RPC, enquanto `doctor` e
// `version` sao comandos de CLI e imprimem la de proposito.
package main

import (
	"errors"
	"os"

	"github.com/jonyd/gobsidian/internal/console"
	"github.com/jonyd/gobsidian/internal/instalar"
	"github.com/jonyd/gobsidian/internal/mcpsrv"
	"github.com/spf13/cobra"
)

// Injetados pelo linker. O script de build que os define e trabalho
// pendente da Task 11 (scripts/ hoje so tem check_net.ps1); ate la, builds
// locais caem no fallback "dev".
var (
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
)

func main() {
	// Propaga a versao injetada pelo linker para o handshake MCP — sem isso
	// o cliente ve sempre "dev", mesmo em build de release.
	mcpsrv.Version = version

	root := newRootCmd()

	if err := root.Execute(); err != nil {
		// Erro vai para o stderr, e a decisao de cor sai do PROPRIO stderr.
		// Se stdout estiver redirecionado e o stderr for um terminal, o erro
		// continua colorido -- e o inverso tambem vale.
		console.New(os.Stderr).Err("%v", err)
		os.Exit(1)
	}
}

// newRootCmd monta a arvore de comandos completa.
//
// Existe separada de main() para que o teste exercite a MESMA arvore que o
// binario, e nao uma parecida. SilenceUsage e o exemplo de por que isso
// importa: com ele, um erro de configuracao nao despeja o texto de uso na
// saida -- e um teste que montasse o proprio root sem essa flag afirmaria
// sobre um programa que nao existe.
func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "gobsidian",
		Short:         "Servidor MCP para cofres locais do Obsidian",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(
		newServeCmd(), newDoctorCmd(), newVersionCmd(), newIndexCmd(),
		newSearchCmd(), newInspectCmd(), newDaemonCmd(),
		newInstallCmd(), newUpdateCmd(), newPathCmd(), newVaultsCmd(),
	)

	// Sem argumentos: autoinstala, ou mostra a ajuda (decisao D-11 do dono).
	//
	// O executavel baixado e clicado precisa fazer algo util. Mas a
	// autoinstalacao so dispara com DUAS condicoes juntas: nao estar instalado
	// E haver um terminal do outro lado. A segunda existe porque um host MCP
	// que invocasse o binario sem argumento -- nenhum faz hoje, todos passam
	// `serve --vault` -- dispararia uma instalacao no meio de uma sessao.
	root.RunE = func(cmd *cobra.Command, _ []string) error {
		return semArgumentos(cmd)
	}

	// O nushell entra no `completion` que o cobra cria, e nao ao lado dele: o
	// usuario procura `gobsidian completion <shell>`, e um `completion-nushell`
	// solto na raiz seria o mesmo comando com dois nomes. SetupHelp chama
	// InitDefaultCompletionCmd, entao o pai ja existe quando isto roda.
	console.SetupHelp(root)

	// O carapace acrescenta o nushell e, mais importante, o VALOR de cada flag
	// -- ver completar.go. O `completion nushell` proprio saiu quando ele
	// entrou: duas formas de completar o mesmo shell e a duplicacao que este
	// projeto persegue, e a do carapace completa valor.
	instalarCompletion(root)

	return root
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Imprime versao, commit e data de build",
		Run: func(cmd *cobra.Command, _ []string) {
			con := console.New(cmd.OutOrStdout())
			con.Campos("gobsidian "+version, []console.Campo{
				console.Campof("commit", "%s", commit),
				console.Campof("build", "%s", buildDate),
			})
		},
	}
}

// terminalInterativoFn e instalar.TerminalInterativo numa variavel, para o
// teste poder exercitar os dois lados de D-11 sem um pseudo-terminal. Producao
// nunca a troca -- o mesmo padrao de iniciarDaemonFn em ponte.go.
var terminalInterativoFn = instalar.TerminalInterativo

// rodarInstalacaoFn e o que semArgumentos chama quando decide autoinstalar.
//
// Injetavel, e a razao e um defeito que aconteceu de verdade em 2026-09-08: a
// primeira versao do teste de D-11 chamava a instalacao REAL. Ela leu o
// registro do Obsidian, escolheu um cofre, detectou seis hosts de IA e
// reescreveu a configuracao dos seis para apontar para o BINARIO DE TESTE --
// alem de acrescentar o diretorio ao PATH do usuario. O estrago foi desfeito
// pelos backups que hosts.Fundir grava, mas o teste nunca deveria ter podido
// causa-lo.
//
// O que D-11 decide e SE instala, e e isso que o teste exercita. Que a
// instalacao em si funciona e assunto de internal/instalar, onde tudo que toca
// a maquina e injetado.
var rodarInstalacaoFn = rodarInstalacao

// estaInstaladoFn idem, pelo mesmo motivo: o teste precisa dos quatro
// cruzamentos (instalado x nao, interativo x nao), e tres deles nao dependem de
// instalar nada de verdade.
var estaInstaladoFn = func() (bool, error) {
	instalado, _, err := instalar.EstaInstalado()
	return instalado, err
}

// semArgumentos decide o que `gobsidian` sozinho faz.
//
// A deteccao de "instalado" e mecanica (instalar.EstaInstalado): existe
// manifesto E o executavel corrente e o arquivo que ele registra, comparado por
// os.SameFile -- que atravessa link, junction e diferenca de grafia sem depender
// de normalizacao de caminho.
//
// Qualquer duvida cai na AJUDA, nunca na instalacao. Instalar por engano mexe no
// PATH e na configuracao de hosts do usuario; mostrar ajuda por engano nao custa
// nada.
func semArgumentos(cmd *cobra.Command) error {
	instalado, err := estaInstaladoFn()
	if err != nil && !errors.Is(err, instalar.ErrSemManifesto) {
		return cmd.Help()
	}
	if instalado || !terminalInterativoFn() {
		return cmd.Help()
	}

	con := console.New(cmd.OutOrStdout())
	con.Step("gobsidian ainda nao esta instalado nesta maquina")
	con.Detail("este executavel vai se instalar em %s", instalar.DiretorioPadrao())
	con.Detail("para so ver a ajuda, rode `gobsidian --help`")

	var o opcoesDeInstalacao
	o.vault = os.Getenv("GOBSIDIAN_VAULT")
	o.installDir = os.Getenv("GOBSIDIAN_INSTALL_DIR")
	return rodarInstalacaoFn(cmd.Context(), cmd, &o, "")
}
