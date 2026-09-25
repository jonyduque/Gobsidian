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
	"github.com/jonyduque/Gobsidian/internal/textos"
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
		textos.FlagInstallVault)
	cmd.Flags().StringVar(&o.installDir, "install-dir", os.Getenv("GOBSIDIAN_INSTALL_DIR"),
		textos.FlagInstallDir+instalar.DiretorioPadrao()+")")
	cmd.Flags().StringVar(&o.hostsCSV, "hosts", "",
		textos.FlagInstallHosts+strings.Join(hosts.Chaves(), ", ")+textos.FlagInstallHostsFim)
	cmd.Flags().BoolVar(&o.sim, "yes", false,
		textos.FlagInstallYes)
	cmd.Flags().BoolVar(&o.readOnly, "read-only", false, textos.FlagInstallReadOnly)
	cmd.Flags().BoolVar(&o.semPath, "no-path", false, textos.FlagInstallNoPath)
}

func newInstallCmd() *cobra.Command {
	var o opcoesDeInstalacao

	cmd := &cobra.Command{
		Use:   "install",
		Short: textos.ResumoInstall,
		Long:  textos.DescricaoInstall,
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
		con.Warn("%s", textos.InstallSemTerminal)
		con.Detail("%s", textos.InstallSemTerminalDica)
		o.sim = true
	}

	e, err := escolherConfiguracao(con, entrada, o)
	if err != nil {
		return err
	}
	cofres, chaves := e.cofres, e.hosts

	sis := instalar.SistemaReal(
		func(pergunta string, itens []string) bool {
			return confirmar(con, entrada, o.sim, pergunta, itens)
		},
		// O que o usuario ve entre a ultima pergunta e o resumo. Cada passo
		// aparece ANTES de comecar, para que o que travar tenha dito que
		// comecou -- a mesma regra de lifecycle.Esperar.
		func(nome string) { con.Passo("%s", nome) },
	)

	con.Titulo("%s", textos.InstallTitulo)
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
		con.Warn("%s", textos.InstallCancelado)
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
// escolha e o que install e vaults decidiram sobre cofres e hosts.
type escolha struct {
	cofres []string
	// hosts segue a convencao de instalar.Opcoes.Hosts: nulo e "detecte",
	// vazio e "nenhum".
	hosts []string
	// manter diz que o usuario escolheu manter a configuracao atual: nada
	// nos hosts muda.
	manter bool
}

// configuracaoAtualFn e escolherHostsFn existem para o teste da sequencia nao
// ler nem reescrever os hosts de verdade da maquina -- o mesmo motivo de
// rodarInstalacaoFn em main.go.
var (
	configuracaoAtualFn = func() []string {
		return instalar.ConfiguracaoAtual(hosts.AmbienteReal(), instalar.CaminhoDoRegistroDoObsidian())
	}
	escolherHostsFn = escolherHosts
)

// escolherConfiguracao faz as duas perguntas na ordem, e e quem garante que
// "manter" responde as duas.
//
// Medido pelo dono em 2026-09-25, com `gobsidian vaults`: Sim em "Manter esta
// configuracao?" e a lista de hosts apareceu mesmo assim, porque o manter so
// valia para os cofres. Com --yes era pior: a fatia nula de hosts fazia
// instalar.Instalar configurar TODOS os detectados. Manter e nao alterar nada:
// os hosts saem vazios e NAO nulos.
func escolherConfiguracao(con *console.Stream, entrada *bufio.Reader, o *opcoesDeInstalacao) (escolha, error) {
	cofres, manter, err := escolherCofres(con, entrada, o)
	if err != nil {
		return escolha{}, err
	}
	if manter {
		return escolha{cofres: cofres, hosts: []string{}, manter: true}, nil
	}
	chaves, err := escolherHostsFn(con, entrada, o)
	if err != nil {
		return escolha{}, err
	}
	return escolha{cofres: cofres, hosts: chaves}, nil
}

// escolherCofres devolve os cofres escolhidos e se o usuario manteve a
// configuracao atual.
//
// "Manter" so e oferecido quando a linha de comando nao pediu nada: com
// --vault ou --hosts, quem chamou ja disse o que quer configurar.
func escolherCofres(con *console.Stream, entrada *bufio.Reader, o *opcoesDeInstalacao) ([]string, bool, error) {
	if o.vault != "" {
		return []string{o.vault}, false, nil
	}

	jaConfigurados := configuracaoAtualFn()

	// Manter o que ja existe e a resposta mais provavel de quem roda o
	// instalador de novo -- e a unica que nao mexe em nada.
	if len(jaConfigurados) > 0 && o.hostsCSV == "" {
		corpos := make([]string, 0, len(jaConfigurados))
		for _, c := range jaConfigurados {
			corpos = append(corpos, "  "+c)
		}
		con.Bloco(textos.InstallConfigAtual, corpos, "")
		if o.sim || simOuNao(con, entrada, textos.InstallManterConfig, true) {
			return jaConfigurados, true, nil
		}
	}

	doObsidian, err := instalar.CofresDoObsidian(instalar.CaminhoDoRegistroDoObsidian())
	if err != nil {
		con.Warn(textos.InstallRegistroIlegivel, err)
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
		return nil, false, errors.New(textos.ErroSemCofre)
	}
	if o.sim {
		// Sem interacao, a escolha e o que ja estava, ou o primeiro cofre.
		if len(jaConfigurados) > 0 {
			return jaConfigurados, false, nil
		}
		con.Info(textos.InstallCofreEscolhido, itens[0].caminho)
		return []string{itens[0].caminho}, false, nil
	}

	opcoes := make([]console.Opcao, 0, len(itens))
	for _, it := range itens {
		opcoes = append(opcoes, console.Opcao{Rotulo: it.caminho, Nota: it.nota, Marcada: it.marcado})
	}

	indices, err := console.Selecionar(con, arquivoDaEntrada(), textos.InstallQuaisCofres, opcoes)
	switch {
	case err == nil:
	case errors.Is(err, console.ErrCancelado):
		return nil, false, errors.New(textos.ErroSelecaoCancelada)
	case errors.Is(err, console.ErrSemTerminal):
		// Sem terminal de verdade (pipe, IDE, CI): a MESMA pergunta, digitada.
		con.Titulo("%s", textos.InstallQuaisCofres)
		for i, it := range itens {
			con.Detail(textos.InstallItemNumerado, i+1, it.caminho, it.nota)
		}
		resposta := perguntar(con, entrada, textos.InstallEscolhaDigitada, "")
		indices, err = console.SelecionarDigitando(resposta, opcoes)
		if err != nil {
			return nil, false, err
		}
	default:
		return nil, false, err
	}

	var saida []string
	for _, i := range indices {
		saida = append(saida, itens[i].caminho)
	}
	return saida, false, nil
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
				return nil, fmt.Errorf(textos.ErroHostDesconhecido, c, strings.Join(hosts.Chaves(), ", "))
			}
			chaves = append(chaves, c)
		}
		return chaves, nil
	}

	detectados := hosts.Detectar(hosts.AmbienteReal())
	if len(detectados) == 0 {
		con.Info("%s", textos.InstallSemHosts)
		return []string{}, nil
	}

	if o.sim {
		nomes := make([]string, 0, len(detectados))
		for _, h := range detectados {
			nomes = append(nomes, "  "+h.Nome)
		}
		con.Bloco(textos.InstallHostsEncontrados, nomes, "")
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

	indices, err := console.Selecionar(con, arquivoDaEntrada(), textos.InstallQuaisHosts, opcoes)
	switch {
	case err == nil:
	case errors.Is(err, console.ErrCancelado):
		return nil, errors.New(textos.ErroSelecaoCancelada)
	case errors.Is(err, console.ErrSemTerminal):
		con.Titulo("%s", textos.InstallQuaisHosts)
		for i, h := range detectados {
			con.Detail(textos.InstallItemHost, i+1, h.Nome, h.Chave)
		}
		resposta := perguntar(con, entrada, textos.InstallEscolhaDigitada, "*")
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
	con.OK("%s", textos.InstallConcluido)
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
	add(textos.ResumoBinario, r.Binario)
	if len(cofres) == 0 {
		add("%s", textos.ResumoCofres)
	}
	for _, c := range cofres {
		add(textos.ResumoCofre, c)
	}
	if r.PathMudou {
		add(textos.ResumoPATH, instalar.AvisoDePath())
	}
	for _, p := range r.Encerrados {
		add(textos.ResumoEncerrado, p.PID, p.Papel)
	}
	for chave, aviso := range r.HostsOK {
		add(textos.ResumoHostOK, chave, aviso)
	}
	for chave, erro := range r.HostsFalhos {
		add(textos.ResumoHostFalha, chave, erro)
	}
	if !r.Limpeza.Vazio() {
		// O que foi removido ganha cor; o que era zero fica apagado. Mesma
		// regra do bloco de lixo do `doctor`: num resumo em que quase tudo é
		// zero, pintar todos os números não destaca nenhum.
		n := func(q int) string {
			if q == 0 {
				return con.Dim("0")
			}
			return con.Amarelo(fmt.Sprintf("%d", q))
		}
		add(textos.ResumoLimpeza,
			n(len(r.Limpeza.Locks)), n(len(r.Limpeza.Sockets)), n(len(r.Limpeza.Presencas)),
			n(len(r.Limpeza.Caches)), r.Limpeza.Bytes/1024)
	}
	return l
}

func newPathCmd() *cobra.Command {
	var adicionar, remover bool

	cmd := &cobra.Command{
		Use:   "path",
		Short: textos.ResumoPath,
		RunE: func(cmd *cobra.Command, _ []string) error {
			con := console.New(cmd.OutOrStdout())
			if adicionar == remover {
				return errors.New(textos.ErroAddOuRemove)
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
				con.OK("%s", textos.PathJaEstava)
				con.Detail("%s", dir)
				return nil
			}
			con.OK("%s", textos.PathAtualizado)
			con.Detail("%s", dir)
			con.Detail("%s", instalar.AvisoDePath())
			return nil
		},
	}
	cmd.Flags().BoolVar(&adicionar, "add", false, textos.FlagPathAdd)
	cmd.Flags().BoolVar(&remover, "remove", false, textos.FlagPathRemove)
	return cmd
}

func newVaultsCmd() *cobra.Command {
	var o opcoesDeInstalacao

	cmd := &cobra.Command{
		Use:   "vaults",
		Short: textos.ResumoVaults,
		RunE: func(cmd *cobra.Command, _ []string) error {
			con := console.New(cmd.OutOrStdout())
			entrada := bufio.NewReader(cmd.InOrStdin())

			m, err := instalar.LerManifesto()
			if err != nil {
				return fmt.Errorf(textos.ErroSemManifesto, err)
			}

			e, err := escolherConfiguracao(con, entrada, &o)
			if err != nil {
				return err
			}
			if e.manter {
				con.OK("%s", textos.VaultsMantido)
				return nil
			}
			cofres, chaves := e.cofres, e.hosts

			ok, falhos := instalar.ConfigurarHosts(m.Binario, cofres, o.readOnly, chaves)
			con.OK("%s", textos.InstallHostsConfigurados)
			if len(cofres) == 0 {
				con.Detail("%s", textos.InstallSemCofreNaLista)
			}
			for _, c := range cofres {
				con.Detail(textos.InstallCofreNaLista, c)
			}
			for chave, aviso := range ok {
				con.Detail(textos.InstallHostNaLista, chave, aviso)
			}
			for chave, erro := range falhos {
				con.Warn(textos.InstallHostFalhou, chave)
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

// simOuNao pergunta com os BOTOES de console.Confirmar e, sem terminal, com a
// mesma pergunta digitada.
//
// Ate 2026-09-16 so existia a forma digitada: o usuario lia "S/n" e apertava
// Enter sem ver o que estava escolhendo. O modal mostra a resposta corrente na
// tela, que e o que se confirma.
func simOuNao(con *console.Stream, entrada *bufio.Reader, pergunta string, padrao bool) bool {
	return decidir(con, entrada, pergunta, nil, padrao)
}

// decidir e a conta unica das duas formas da mesma pergunta. Cancelar (Esc, q,
// Ctrl-C) e NAO: desistir de uma pergunta nunca autoriza o lado que muda o
// sistema.
func decidir(con *console.Stream, entrada *bufio.Reader, pergunta string, itens []string, padrao bool) bool {
	resposta, err := console.Confirmar(con, arquivoDaEntrada(), pergunta, itens, padrao)
	switch {
	case err == nil:
		return resposta
	case errors.Is(err, console.ErrCancelado):
		con.Resposta("cancelado")
		return false
	case errors.Is(err, console.ErrSemTerminal):
		for _, i := range itens {
			con.Detail("%s", i)
		}
		return simOuNaoDigitado(con, entrada, pergunta, padrao)
	default:
		return false
	}
}

func simOuNaoDigitado(con *console.Stream, entrada *bufio.Reader, pergunta string, padrao bool) bool {
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
	if sim {
		con.Warn("%s", pergunta)
		for _, i := range itens {
			con.Detail("%s", i)
		}
		con.Detail("%s", textos.InstallYesEncerrando)
		return true
	}
	// Os itens vao DENTRO da moldura da pergunta, e nao impressos antes dela:
	// a lista de processos e o que se esta decidindo, e uma lista que rolou
	// para fora da tela nao permite discordar de nenhum deles.
	return decidir(con, entrada, pergunta, itens, false)
}

// arquivoDaEntrada devolve a entrada padrao como *os.File, que e o que o modo
// bruto do terminal exige.
//
// cmd.InOrStdin() e um io.Reader e serve para os testes injetarem texto; o modo
// bruto precisa do descritor de verdade. Quando a entrada nao e o stdin real, a
// selecao cai sozinha no caminho digitado -- entrarNoModoBruto devolve
// ErrSemTerminal e quem chama trata.
func arquivoDaEntrada() *os.File { return os.Stdin }
