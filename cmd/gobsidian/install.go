// install.go implementa `gobsidian install`, `gobsidian path` e
// `gobsidian vaults` -- o instalador que ate 2026-09-08 vivia em `install.ps1`
// (729 linhas) e `installer/install.js` (1 009), a mesma logica escrita duas
// vezes e sem um unico teste.
//
// Os tres comandos imprimem em STDOUT de proposito, como `doctor` e `version`:
// sao comandos de CLI, nao servidores. Nenhum JSON-RPC trafega aqui.
package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/jonyd/gobsidian/internal/console"
	"github.com/jonyd/gobsidian/internal/hosts"
	"github.com/jonyd/gobsidian/internal/instalar"
	"github.com/spf13/cobra"
)

// opcoesDeInstalacao sao as flags compartilhadas por `install` e `vaults`.
type opcoesDeInstalacao struct {
	vault      string
	installDir string
	hostsCSV   string
	sim        bool
	readOnly   bool
	semPath    bool
}

// registrarFlagsDeInstalacao poe as flags e as variaveis de ambiente
// equivalentes -- as mesmas que `install.ps1` ja aceitava, para nao quebrar
// quem automatiza.
func registrarFlagsDeInstalacao(cmd *cobra.Command, o *opcoesDeInstalacao) {
	cmd.Flags().StringVar(&o.vault, "vault", os.Getenv("GOBSIDIAN_VAULT"),
		"cofre a servir (padrao: perguntar, lendo o registro do Obsidian)")
	cmd.Flags().StringVar(&o.installDir, "install-dir", os.Getenv("GOBSIDIAN_INSTALL_DIR"),
		"onde por o binario (padrao: "+instalar.DiretorioPadrao()+")")
	cmd.Flags().StringVar(&o.hostsCSV, "hosts", "",
		"hosts a configurar, separados por virgula ("+strings.Join(hosts.Chaves(), ", ")+"); 'none' nao configura nenhum")
	cmd.Flags().BoolVar(&o.sim, "yes", false,
		"nao pergunta nada: instala, ajusta o PATH e configura os hosts detectados")
	cmd.Flags().BoolVar(&o.readOnly, "read-only", false, "registra o servidor com --read-only")
	cmd.Flags().BoolVar(&o.semPath, "no-path", false, "nao mexe no PATH")
}

func newInstallCmd() *cobra.Command {
	var o opcoesDeInstalacao

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Instala o gobsidian, ajusta o PATH e configura os hosts de IA",
		Long: "Instala o executavel no perfil do usuario -- nunca pede elevacao --, " +
			"pergunta se deve acrescentar o diretorio ao PATH e registrar o servidor " +
			"nos hosts de IA detectados, e limpa lixo comprovadamente orfao de execucoes anteriores.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return rodarInstalacao(cmd.Context(), cmd, &o, "")
		},
	}
	registrarFlagsDeInstalacao(cmd, &o)
	return cmd
}

// rodarInstalacao e o corpo compartilhado por `install` e pela autoinstalacao
// sem argumentos (decisao D-11). origem vazia significa "o executavel corrente".
func rodarInstalacao(ctx context.Context, cmd *cobra.Command, o *opcoesDeInstalacao, origem string) error {
	con := console.New(cmd.OutOrStdout())
	entrada := bufio.NewReader(cmd.InOrStdin())

	cofre, err := escolherCofre(con, entrada, o)
	if err != nil {
		return err
	}

	chaves, err := escolherHosts(con, entrada, o)
	if err != nil {
		return err
	}

	sis := instalar.SistemaReal(func(pergunta string, itens []string) bool {
		return confirmar(con, entrada, o.sim, pergunta, itens)
	})

	con.Step("Instalando")
	r, err := instalar.Instalar(ctx, sis, instalar.Opcoes{
		Origem:   origem,
		Destino:  o.installDir,
		Versao:   version,
		Cofre:    cofre,
		Hosts:    chaves,
		ReadOnly: o.readOnly,
		SemPath:  o.semPath,
	})
	if errors.Is(err, instalar.ErrRecusado) {
		con.Warn("Instalacao cancelada; nada foi alterado")
		return nil
	}
	if err != nil {
		return err
	}

	imprimirResumo(con, r, cofre)
	return nil
}

// escolherCofre resolve qual cofre servir.
//
// Le o registro do PROPRIO Obsidian em vez de pedir um caminho que o usuario
// teria de ir buscar. Cofre que nao existe mais nao aparece na lista (ver
// instalar.CofresDoObsidian).
func escolherCofre(con *console.Stream, entrada *bufio.Reader, o *opcoesDeInstalacao) (string, error) {
	if o.vault != "" {
		return o.vault, nil
	}

	cofres, err := instalar.CofresDoObsidian(instalar.CaminhoDoRegistroDoObsidian())
	if err != nil {
		con.Warn("nao foi possivel ler o registro de cofres do Obsidian: %v", err)
	}
	if len(cofres) == 0 {
		return "", errors.New("nenhum cofre encontrado; passe --vault com o caminho do cofre")
	}
	if len(cofres) == 1 || o.sim {
		con.Info("cofre: %s", cofres[0].Caminho)
		return cofres[0].Caminho, nil
	}

	con.Step("Qual cofre?")
	for i, c := range cofres {
		marca := ""
		if c.Aberto {
			marca = "  (aberto agora)"
		}
		con.Detail("%d) %s%s", i+1, c.Caminho, marca)
	}
	escolha := perguntar(con, entrada, fmt.Sprintf("Numero [1-%d]", len(cofres)), "1")
	n, err := strconv.Atoi(strings.TrimSpace(escolha))
	if err != nil || n < 1 || n > len(cofres) {
		return "", fmt.Errorf("escolha invalida: %q", escolha)
	}
	return cofres[n-1].Caminho, nil
}

// escolherHosts resolve em quais hosts registrar.
//
// Devolve nil quando a escolha e "os detectados" -- instalar.Instalar entende
// nil como "detecte voce". "none" devolve uma lista vazia NAO nula, que e o
// jeito de dizer "nenhum" sem cair no caso de nil.
func escolherHosts(con *console.Stream, entrada *bufio.Reader, o *opcoesDeInstalacao) ([]string, error) {
	if o.hostsCSV == "none" {
		return []string{}, nil
	}
	if o.hostsCSV != "" {
		var chaves []string
		for _, c := range strings.Split(o.hostsCSV, ",") {
			c = strings.TrimSpace(c)
			if c == "" {
				continue
			}
			if _, ok := hosts.PorChave(c); !ok {
				return nil, fmt.Errorf("host desconhecido %q; conhecidos: %s", c, strings.Join(hosts.Chaves(), ", "))
			}
			chaves = append(chaves, c)
		}
		return chaves, nil
	}

	detectados := hosts.Detectar(hosts.AmbienteReal())
	if len(detectados) == 0 {
		con.Info("nenhum host de IA conhecido foi detectado")
		return []string{}, nil
	}

	con.Step("Hosts de IA encontrados")
	for _, h := range detectados {
		con.Detail("%s", h.Nome)
	}
	if o.sim {
		return nil, nil
	}
	if !simOuNao(con, entrada, "Registrar o gobsidian nestes hosts?", true) {
		return []string{}, nil
	}
	return nil, nil
}

func imprimirResumo(con *console.Stream, r instalar.Resultado, cofre string) {
	con.OK("Instalado")
	con.Detail("binario  %s", r.Binario)
	con.Detail("cofre    %s", cofre)
	if r.PathMudou {
		con.Detail("PATH     %s", instalar.AvisoDePath())
	}
	for _, p := range r.Encerrados {
		con.Detail("encerrado pid %d (%s)", p.PID, p.Papel)
	}
	for chave, aviso := range r.HostsOK {
		con.Detail("%-16s %s", chave, aviso)
	}
	for chave, erro := range r.HostsFalhos {
		con.Warn("%s nao pode ser configurado", chave)
		con.Detail("%s", erro)
	}
	if !r.Limpeza.Vazio() {
		con.Detail("limpeza  %d trava(s), %d socket(s), %d presenca(s), %d cache(s), %d KB",
			len(r.Limpeza.Locks), len(r.Limpeza.Sockets), len(r.Limpeza.Presencas),
			len(r.Limpeza.Caches), r.Limpeza.Bytes/1024)
	}
}

func newPathCmd() *cobra.Command {
	var adicionar, remover bool

	cmd := &cobra.Command{
		Use:   "path",
		Short: "Acrescenta ou remove o diretorio de instalacao do PATH do usuario",
		RunE: func(cmd *cobra.Command, _ []string) error {
			con := console.New(cmd.OutOrStdout())
			if adicionar == remover {
				return errors.New("escolha exatamente um: --add ou --remove")
			}

			dir := instalar.DiretorioPadrao()
			if m, err := instalar.LerManifesto(); err == nil && m.Binario != "" {
				dir = diretorioDe(m.Binario)
			}

			var mudou bool
			var err error
			if adicionar {
				mudou, err = instalar.AdicionarAoPath(dir)
			} else {
				mudou, err = instalar.RemoverDoPath(dir)
			}
			if err != nil {
				return err
			}
			if !mudou {
				con.OK("PATH ja estava como voce pediu")
				con.Detail("%s", dir)
				return nil
			}
			con.OK("PATH atualizado")
			con.Detail("%s", dir)
			con.Detail("%s", instalar.AvisoDePath())
			return nil
		},
	}
	cmd.Flags().BoolVar(&adicionar, "add", false, "acrescenta o diretorio ao PATH")
	cmd.Flags().BoolVar(&remover, "remove", false, "remove o diretorio do PATH")
	return cmd
}

func newVaultsCmd() *cobra.Command {
	var o opcoesDeInstalacao

	cmd := &cobra.Command{
		Use:   "vaults",
		Short: "Configura os hosts de IA para um cofre, sem reinstalar o binario",
		RunE: func(cmd *cobra.Command, _ []string) error {
			con := console.New(cmd.OutOrStdout())
			entrada := bufio.NewReader(cmd.InOrStdin())

			m, err := instalar.LerManifesto()
			if err != nil {
				return fmt.Errorf("%w -- rode `gobsidian install` primeiro", err)
			}

			cofre, err := escolherCofre(con, entrada, &o)
			if err != nil {
				return err
			}
			chaves, err := escolherHosts(con, entrada, &o)
			if err != nil {
				return err
			}

			ok, falhos := instalar.ConfigurarHosts(m.Binario, cofre, o.readOnly, chaves)
			con.OK("Hosts configurados")
			con.Detail("cofre    %s", cofre)
			for chave, aviso := range ok {
				con.Detail("%-16s %s", chave, aviso)
			}
			for chave, erro := range falhos {
				con.Warn("%s nao pode ser configurado", chave)
				con.Detail("%s", erro)
			}
			return nil
		},
	}
	registrarFlagsDeInstalacao(cmd, &o)
	return cmd
}

// diretorioDe devolve o diretorio de um caminho de arquivo. Existe como funcao
// nomeada para o comando `path` nao importar path/filepath so por uma linha.
func diretorioDe(caminho string) string {
	if i := strings.LastIndexAny(caminho, `/\`); i > 0 {
		return caminho[:i]
	}
	return caminho
}

// perguntar le uma linha, devolvendo o padrao quando o usuario so aperta Enter.
func perguntar(con *console.Stream, entrada *bufio.Reader, pergunta, padrao string) string {
	con.Detail("%s (padrao: %s): ", pergunta, padrao)
	linha, err := entrada.ReadString('\n')
	if err != nil && strings.TrimSpace(linha) == "" {
		return padrao
	}
	linha = strings.TrimSpace(linha)
	if linha == "" {
		return padrao
	}
	return linha
}

func simOuNao(con *console.Stream, entrada *bufio.Reader, pergunta string, padrao bool) bool {
	sufixo := "[s/N]"
	if padrao {
		sufixo = "[S/n]"
	}
	resposta := strings.ToLower(perguntar(con, entrada, pergunta+" "+sufixo, map[bool]string{true: "s", false: "n"}[padrao]))
	return resposta == "s" || resposta == "sim" || resposta == "y" || resposta == "yes"
}

// confirmar e o que instalar.Sistema chama antes de encerrar processos.
//
// Com --yes ele nao pergunta: quem automatiza ja decidiu. Sem --yes, a lista
// aparece inteira -- PID e cofre -- porque "3 processos" nao permite discordar
// de nenhum deles.
func confirmar(con *console.Stream, entrada *bufio.Reader, sim bool, pergunta string, itens []string) bool {
	con.Warn("%s", pergunta)
	for _, i := range itens {
		con.Detail("%s", i)
	}
	if sim {
		con.Detail("--yes: encerrando sem perguntar")
		return true
	}
	return simOuNao(con, entrada, "Encerrar?", false)
}
