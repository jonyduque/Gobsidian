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
	"strings"

	"github.com/jonyduque/Gobsidian/internal/console"
	"github.com/jonyduque/Gobsidian/internal/hosts"
	"github.com/jonyduque/Gobsidian/internal/instalar"
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

	// Entrada que NAO e terminal nao pode ser lida como resposta.
	//
	// Medido em 2026-09-11, com o bootstrap de nushell rodando de um cano
	// (`http get .../install.nu | nu --stdin -c $in`): o processo filho herda o
	// stdin do CANO, que ainda carrega o resto do texto do script, e o
	// instalador leu as linhas do proprio script como respostas do usuario. A
	// saida do dono mostrou "escolha invalida: #" -- uma linha de comentario do
	// install.nu virando escolha de cofre -- e uma pergunta de sim/nao
	// respondida sozinha.
	//
	// Nao basta tratar EOF: o problema e haver bytes que NAO sao resposta. A
	// unica pergunta segura aqui e "ha alguem do outro lado?", e a resposta e
	// TerminalInterativo.
	if !o.sim && !terminalInterativoFn() {
		con.Warn("a entrada nao e um terminal: usando as respostas padrao")
		con.Detail("para escolher os cofres, rode `gobsidian install` num terminal")
		o.sim = true
	}

	cofres, err := escolherCofres(con, entrada, o)
	if err != nil {
		return err
	}

	chaves, err := escolherHosts(con, entrada, o)
	if err != nil {
		return err
	}

	sis := instalar.SistemaReal(
		func(pergunta string, itens []string) bool {
			return confirmar(con, entrada, o.sim, pergunta, itens)
		},
		// O que o usuario ve entre a ultima pergunta e o resumo. Cada passo
		// aparece ANTES de comecar, para que o que travar tenha dito que
		// comecou -- a mesma regra de lifecycle.Esperar.
		func(nome string) { con.Passo("%s", nome) },
	)

	con.Titulo("Instalando")
	r, err := instalar.Instalar(ctx, sis, instalar.Opcoes{
		Origem:      origem,
		Destino:     o.installDir,
		Versao:      version,
		Cofre:       primeiro(cofres),
		CofresExtra: resto(cofres),
		Hosts:       chaves,
		ReadOnly:    o.readOnly,
		SemPath:     o.semPath,
	})
	if errors.Is(err, instalar.ErrRecusado) {
		con.Warn("Instalacao cancelada; nada foi alterado")
		return nil
	}
	if err != nil {
		return err
	}

	imprimirResumo(con, r, cofres)
	return nil
}

// primeiro e resto quebram a lista de cofres no formato que Opcoes carrega.
// Existem como funcoes nomeadas para o chamador nao repetir o cuidado com a
// lista vazia em dois lugares.
func primeiro(cofres []string) string {
	if len(cofres) == 0 {
		return ""
	}
	return cofres[0]
}

func resto(cofres []string) []string {
	if len(cofres) <= 1 {
		return nil
	}
	return cofres[1:]
}

// escolherCofres resolve QUAIS cofres servir.
//
// # A ordem das perguntas, e por que ela mudou
//
// Ate 2026-09-09 a primeira coisa que o instalador fazia era listar os cofres
// do Obsidian e pedir um numero. Isso pressupunha duas coisas erradas: que nao
// havia nada configurado, e que a resposta era exatamente um. Quem ja tinha
// dois cofres registrados reconfigurava os dois para nao perder um, e quem nao
// queria nenhum nao tinha como dizer.
//
// Agora a sequencia e: ler o que ja esta configurado, oferecer MANTER, e so
// entao mostrar a lista -- com os cofres ja configurados marcados.
//
// # Por que caixas
//
// Numero digitado aceita um. Caixas aceitam 0..N, que e a forma real da
// pergunta. Ver console.Selecionar; sem terminal, cai na versao digitada.
func escolherCofres(con *console.Stream, entrada *bufio.Reader, o *opcoesDeInstalacao) ([]string, error) {
	if o.vault != "" {
		return []string{o.vault}, nil
	}

	jaConfigurados := instalar.ConfiguracaoAtual(hosts.AmbienteReal())

	// Manter o que ja existe e a resposta mais provavel de quem roda o
	// instalador de novo -- e a unica que nao mexe em nada.
	if len(jaConfigurados) > 0 {
		corpos := make([]string, 0, len(jaConfigurados))
		for _, c := range jaConfigurados {
			corpos = append(corpos, "  "+c)
		}
		con.Bloco("Configuracao atual", corpos, "")
		if o.sim {
			return jaConfigurados, nil
		}
		if simOuNao(con, entrada, "Manter esta configuracao?", true) {
			return jaConfigurados, nil
		}
	}

	doObsidian, err := instalar.CofresDoObsidian(instalar.CaminhoDoRegistroDoObsidian())
	if err != nil {
		con.Warn("nao foi possivel ler o registro de cofres do Obsidian: %v", err)
	}

	// A lista soma os cofres do Obsidian com os que JA estao configurados e nao
	// aparecem la. Um cofre configurado a mao, ou aberto por um Obsidian que
	// nao e este, sumiria da lista -- e sumir da lista aqui significa ser
	// desconfigurado.
	type item struct {
		caminho string
		nota    string
		marcado bool
	}
	// A comparacao e sobre o caminho CANONICO dos dois lados. O registro do
	// Obsidian devolve contrabarra e o config de um host guarda a grafia que
	// foi passada na hora de configurar -- em 2026-09-11 o dono viu cada um dos
	// seus quatro cofres DUAS vezes por causa disso, um com "\\" e outro com
	// "/", nenhum reconhecido como o outro.
	var itens []item
	visto := map[string]bool{}
	for _, c := range doObsidian {
		canonico := instalar.CaminhoCanonicoDeCofre(c.Caminho)
		configurado := contem(jaConfigurados, canonico)
		nota := ""
		if c.Aberto {
			nota = "(aberto agora)"
		}
		if configurado {
			nota = "(ja configurado)"
		}
		itens = append(itens, item{canonico, nota, configurado})
		visto[canonico] = true
	}
	for _, c := range jaConfigurados {
		if !visto[c] {
			itens = append(itens, item{c, "(ja configurado, fora do Obsidian)", true})
		}
	}

	if len(itens) == 0 {
		return nil, errors.New("nenhum cofre encontrado; passe --vault com o caminho do cofre")
	}
	if o.sim {
		// Sem interacao, a escolha e o que ja estava, ou o primeiro cofre.
		if len(jaConfigurados) > 0 {
			return jaConfigurados, nil
		}
		con.Info("cofre: %s", itens[0].caminho)
		return []string{itens[0].caminho}, nil
	}

	opcoes := make([]console.Opcao, 0, len(itens))
	for _, it := range itens {
		opcoes = append(opcoes, console.Opcao{Rotulo: it.caminho, Nota: it.nota, Marcada: it.marcado})
	}

	indices, err := console.Selecionar(con, arquivoDaEntrada(), "Quais cofres configurar?", opcoes)
	switch {
	case err == nil:
	case errors.Is(err, console.ErrCancelado):
		return nil, errors.New("selecao cancelada; nada foi alterado")
	case errors.Is(err, console.ErrSemTerminal):
		// Sem terminal de verdade (pipe, IDE, CI): a MESMA pergunta, digitada.
		con.Titulo("Quais cofres configurar?")
		for i, it := range itens {
			con.Detail("%d) %s  %s", i+1, it.caminho, it.nota)
		}
		resposta := perguntar(con, entrada, "Numeros separados por espaco, * para todos, vazio para nenhum", "")
		indices, err = console.SelecionarDigitando(resposta, opcoes)
		if err != nil {
			return nil, err
		}
	default:
		return nil, err
	}

	var saida []string
	for _, i := range indices {
		saida = append(saida, itens[i].caminho)
	}
	return saida, nil
}

func contem(lista []string, alvo string) bool {
	for _, x := range lista {
		if x == alvo {
			return true
		}
	}
	return false
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

	if o.sim {
		nomes := make([]string, 0, len(detectados))
		for _, h := range detectados {
			nomes = append(nomes, "  "+h.Nome)
		}
		con.Bloco("Hosts de IA encontrados", nomes, "")
		return nil, nil
	}

	// Caixas, como na lista de cofres, e pela mesma razao: a pergunta real e
	// QUAIS, e nao "todos ou nenhum". Quem tem seis hosts instalados e quer
	// configurar dois nao tinha como dizer isso -- respondia "nao" e ficava sem
	// nenhum.
	//
	// Todos marcados por padrao: foram DETECTADOS, entao querer todos e a
	// resposta provavel, e desmarcar e mais barato que marcar seis.
	opcoes := make([]console.Opcao, 0, len(detectados))
	for _, h := range detectados {
		opcoes = append(opcoes, console.Opcao{Rotulo: h.Nome, Nota: h.Chave, Marcada: true})
	}

	indices, err := console.Selecionar(con, arquivoDaEntrada(), "Em quais hosts registrar?", opcoes)
	switch {
	case err == nil:
	case errors.Is(err, console.ErrCancelado):
		return nil, errors.New("selecao cancelada; nada foi alterado")
	case errors.Is(err, console.ErrSemTerminal):
		con.Titulo("Em quais hosts registrar?")
		for i, h := range detectados {
			con.Detail("%d) %s  (%s)", i+1, h.Nome, h.Chave)
		}
		resposta := perguntar(con, entrada, "Numeros separados por espaco, * para todos, vazio para nenhum", "*")
		indices, err = console.SelecionarDigitando(resposta, opcoes)
		if err != nil {
			return nil, err
		}
	default:
		return nil, err
	}

	// Fatia VAZIA e nao nula: "nenhum host", e nao "detecte voce". A distincao
	// esta em configurarHosts, e trocar uma pela outra faria "nenhum"
	// configurar tudo.
	chaves := []string{}
	for _, i := range indices {
		chaves = append(chaves, detectados[i].Chave)
	}
	return chaves, nil
}

func imprimirResumo(con *console.Stream, r instalar.Resultado, cofres []string) {
	con.OK("Instalado")
	con.Bloco("", resumoEmLinhas(con, r, cofres), "")
}

// resumoEmLinhas monta o corpo do bloco final.
//
// Separada de imprimirResumo para o bloco ser montado numa passada so: a
// moldura precisa de TODAS as linhas antes de decidir a largura, e imprimir
// direto (como era ate 2026-09-09) nao permite isso.
func resumoEmLinhas(con *console.Stream, r instalar.Resultado, cofres []string) []string {
	var l []string
	add := func(f string, a ...any) { l = append(l, "  "+fmt.Sprintf(f, a...)) }
	_ = con
	add("binario  %s", r.Binario)
	if len(cofres) == 0 {
		add("cofres   nenhum configurado")
	}
	for _, c := range cofres {
		add("cofre    %s", c)
	}
	if r.PathMudou {
		add("PATH     %s", instalar.AvisoDePath())
	}
	for _, p := range r.Encerrados {
		add("encerrado pid %d (%s)", p.PID, p.Papel)
	}
	for chave, aviso := range r.HostsOK {
		add("%-16s %s", chave, aviso)
	}
	for chave, erro := range r.HostsFalhos {
		add("%-16s FALHOU: %s", chave, erro)
	}
	if !r.Limpeza.Vazio() {
		add("limpeza  %d trava(s), %d socket(s), %d presenca(s), %d cache(s), %d KB",
			len(r.Limpeza.Locks), len(r.Limpeza.Sockets), len(r.Limpeza.Presencas),
			len(r.Limpeza.Caches), r.Limpeza.Bytes/1024)
	}
	return l
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

			cofres, err := escolherCofres(con, entrada, &o)
			if err != nil {
				return err
			}
			chaves, err := escolherHosts(con, entrada, &o)
			if err != nil {
				return err
			}

			ok, falhos := instalar.ConfigurarHosts(m.Binario, cofres, o.readOnly, chaves)
			con.OK("Hosts configurados")
			if len(cofres) == 0 {
				con.Detail("cofres   nenhum configurado")
			}
			for _, c := range cofres {
				con.Detail("cofre    %s", c)
			}
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
	con.Pergunta(pergunta, padrao)
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

// perguntar NAO ecoa a resposta de proposito. Enter sozinho nao deixa nada na
// tela, e o eco resolveria isso -- mas o eco util diz o SIGNIFICADO da
// escolha, e so quem chama sabe traduzir "S/n" em "sim". Ver simOuNao.

func simOuNao(con *console.Stream, entrada *bufio.Reader, pergunta string, padrao bool) bool {
	sufixo := "s/N"
	if padrao {
		sufixo = "S/n"
	}
	// O padrao mostrado e o par inteiro, com a letra maiuscula marcando qual
	// deles o Enter escolhe -- e a convencao que todo instalador de linha de
	// comando usa, e ela cabe no lugar onde Pergunta ja mostra o padrao.
	resposta := strings.ToLower(perguntar(con, entrada, pergunta, sufixo))

	sim := padrao
	if resposta != strings.ToLower(sufixo) {
		// Resposta diferente do proprio sufixo: o usuario digitou algo. Enter
		// sozinho devolve o sufixo e cai no padrao.
		sim = resposta == "s" || resposta == "sim" || resposta == "y" || resposta == "yes"
	}
	con.Resposta(map[bool]string{true: "sim", false: "nao"}[sim])
	return sim
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

// arquivoDaEntrada devolve a entrada padrao como *os.File, que e o que o modo
// bruto do terminal exige.
//
// cmd.InOrStdin() e um io.Reader e serve para os testes injetarem texto; o modo
// bruto precisa do descritor de verdade. Quando a entrada nao e o stdin real, a
// selecao cai sozinha no caminho digitado -- entrarNoModoBruto devolve
// ErrSemTerminal e quem chama trata.
func arquivoDaEntrada() *os.File { return os.Stdin }
