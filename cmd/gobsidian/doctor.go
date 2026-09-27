package main

import (
	"github.com/jonyduque/Gobsidian/internal/textos"
	"os"

	"errors"
	"fmt"
	"github.com/jonyduque/Gobsidian/internal/config"
	"github.com/jonyduque/Gobsidian/internal/console"
	"github.com/jonyduque/Gobsidian/internal/doctor"
	"github.com/jonyduque/Gobsidian/internal/hosts"
	"github.com/jonyduque/Gobsidian/internal/instalar"
	"github.com/spf13/cobra"
	"path/filepath"
	"sort"
	"strings"
)

func newDoctorCmd() *cobra.Command {
	var flags config.Flags

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: textos.ResumoDoctor,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Sem esta linha, --read-only nao chega a Config e a verificacao
			// de permissao de escrita roda mesmo quando o usuario pediu para
			// nao rodar. Toda chamada a config.Load precisa preencher os
			// companheiros das flags que o comando expoe.
			flags.ReadOnlySet = cmd.Flags().Changed("read-only")
			flags.MaxResultsSet = cmd.Flags().Changed("max-results")

			cfg, err := carregarConfig(flags)
			if err != nil {
				return err
			}

			results := doctor.Run(cmd.Context(), cfg)

			// doctor imprime em stdout de proposito: e um comando de CLI,
			// nao um servidor. Nenhum JSON-RPC trafega aqui.
			//
			// O erro de escrita e descartado explicitamente, e nao por
			// esquecimento: se o proprio relatorio nao sai, nao sobra canal
			// para reclamar disso.
			// A cor sai do writer do comando, nao de os.Stdout: quem faz
			// `gobsidian doctor > relatorio.txt` recebe um arquivo limpo.
			con := console.New(cmd.OutOrStdout())
			con.Titulo("%s", textos.DoctorTitulo)
			relatarVerificacoes(con, results)

			relatarProcessosELixo(con, corrigir)

			code := doctor.ExitCode(results)
			if code != 0 {
				con.Err("%s", textos.DoctorFalhasAcima)
				os.Exit(code)
			}
			con.OK("%s", textos.DoctorAmbienteApto)
			return nil
		},
	}

	flagsDeCofre(cmd, &flags)
	cmd.Flags().BoolVar(&flags.ReadOnly, "read-only", false, textos.FlagDoctorReadOnly)
	cmd.Flags().IntVar(&flags.MaxResults, "max-results", 0, textos.FlagMaxResults)
	cmd.Flags().BoolVar(&corrigir, "fix", false,
		textos.FlagDoctorFix)

	return cmd
}

// corrigir e a flag --fix. Fora do config.Flags de proposito: ela nao configura
// o produto, so decide se este comando escreve.
var corrigir bool

// relatarProcessosELixo responde as duas perguntas que a investigacao de
// 2026-09-08 nao conseguiu responder com o produto na mao.
//
// A primeira: QUEM esta servindo agora? Naquele dia havia dois processos
// servindo o cofre Estudo e gravando o mesmo inverted_cache.gob, e descobrir
// isso exigiu comparar milissegundos entre linhas de log duplicadas.
//
// A segunda: o que sobrou de lixo? O diretorio de runtime tinha 960 arquivos
// .lock e 3 .sock de cofres que nao existem mais.
//
// Sem --fix nada e removido: ver e autorizar sao decisoes diferentes, e um
// diagnostico que muda o sistema so por ter sido consultado nao e diagnostico.
//
// Esta funcao mora no COMANDO, e nao em internal/doctor, para nao criar aresta
// de internal/doctor para internal/instalar. doctor responde sobre o AMBIENTE;
// quem sabe de processos e de lixo do produto e o instalador.
func relatarProcessosELixo(con *console.Stream, aplicar bool) {
	runtimeDir, err := instalar.DiretorioDeRuntime()
	if err != nil {
		con.Warn(textos.DoctorRuntimeIndisponivel, err)
		return
	}

	vivos, err := instalar.VivosComAnterior(runtimeDir)
	switch {
	case err != nil:
		con.Warn(textos.DoctorProcessosErro, err)
	case len(vivos) == 0:
		con.OK("%s", textos.DoctorProcessos)
		con.Detail("%s", textos.DoctorNenhumProcesso)
	default:
		con.OK(textos.DoctorProcessosContagem, len(vivos))
		con.Bloco("", linhasPorCofre(con, vivos), textos.DoctorProcessosRodape)
		// Dois GRAVADORES do mesmo cofre gravam o mesmo cache de busca: o estado
		// medido em 2026-09-08. Ponte nao grava, e ate 2026-09-14 este aviso a
		// contava -- mandava encerrar as pontes do Antigravity, que estavam
		// certas. Ver instalar.GravaCache.
		for _, s := range analisarGravadores(vivos) {
			if len(s.Gravadores) > 1 {
				con.Warn(textos.DoctorGravadoresDuplos, len(s.Gravadores))
				con.Detail(textos.DoctorGravadoresDetalhe, s.Cofre, listarPIDs(s.Gravadores))
			}
			if len(s.SemModo) > 0 {
				con.Detail(textos.DoctorSemModoDetalhe, s.Cofre, listarPIDs(s.SemModo))
			}
		}
	}

	relatarProcessosSemPresenca(con, vivos)
	relatarHostsDeOutroBinario(con, vivos)

	// Chaves de cache que ficaram para tras da conta de config.VaultKey.
	//
	// SEMPRE em simulacao aqui, mesmo com -Aplicar: renomear um diretorio de
	// cache enquanto um daemon o mapeia com mmap e pedir problema, e `doctor`
	// roda com o produto no ar. Quem renomeia e `install`/`update`, sob a
	// trava global e com nenhum processo vivo.
	if migradas, err := instalar.MigrarChaves(instalar.RaizDoCache(), false); err != nil {
		con.Warn(textos.DoctorChavesErro, err)
	} else if len(migradas) > 0 {
		con.Warn(textos.DoctorChavesSuperada, len(migradas))
		linhas := make([]string, 0, len(migradas))
		for _, m := range migradas {
			linhas = append(linhas, fmt.Sprintf("  %s -> %s  %s", m.De, m.Para, con.Dim(m.Cofre)))
		}
		con.Bloco("", linhas, textos.DoctorChavesRodape)
	}

	r, err := instalar.Limpar(runtimeDir, instalar.RaizDoCache(), aplicar)
	if err != nil {
		con.Warn(textos.DoctorLixoErro, err)
		return
	}
	if r.Vazio() {
		con.OK("%s", textos.DoctorLixoNenhum)
		con.Detail("%s", textos.DoctorLixoNadaARemover)
		return
	}

	verbo := textos.DoctorLixoRemovivel
	if aplicar {
		verbo = textos.DoctorLixoRemovido
	}
	// O número pintado é o que EXISTE; zero fica apagado. Num bloco em que
	// quase tudo é zero, pintar todos os números esconderia o único que
	// importa.
	numero := func(n int) string {
		if n == 0 {
			return con.Dim("0")
		}
		return con.Amarelo(fmt.Sprintf("%d", n))
	}
	con.Warn("%s", textos.DoctorLixoTitulo)
	con.Campos("", []console.Campo{
		{Chave: textos.DoctorLixoTravas, Valor: numero(len(r.Locks))},
		{Chave: textos.DoctorLixoSockets, Valor: numero(len(r.Sockets))},
		{Chave: textos.DoctorLixoPresencas, Valor: numero(len(r.Presencas))},
		{Chave: textos.DoctorLixoCaches, Valor: numero(len(r.Caches)), Nota: textos.DoctorLixoCachesNota},
		{Chave: textos.DoctorLixoLogs, Valor: numero(len(r.LogsRotacionados))},
		{Chave: textos.DoctorLixoTotal, Valor: fmt.Sprintf(textos.DoctorLixoKB, r.Bytes/1024), Nota: verbo},
	})
	if !aplicar {
		con.Detail("%s", textos.DoctorLixoComoLimpar)
	}
	for _, falha := range r.NaoRemovidos {
		con.Detail(textos.DoctorLixoNaoRemovido, falha)
	}
}

// situacaoDoCofre separa, para um cofre, quem grava o cache de quem nao se
// sabe.
type situacaoDoCofre struct {
	Cofre      string
	Gravadores []instalar.Presenca
	SemModo    []instalar.Presenca
}

// analisarGravadores agrupa as presencas por cofre, em ordem de cofre para a
// saida nao mudar de uma rodada para outra. Chave e o caminho como o processo
// o registrou; comparacao de caminho nao entra aqui porque quem escreveu os
// dois lados foi o mesmo produto, na mesma maquina.
func analisarGravadores(vivos []instalar.Presenca) []situacaoDoCofre {
	porCofre := map[string]*situacaoDoCofre{}
	var ordem []string
	for _, p := range vivos {
		if p.Cofre == "" {
			continue
		}
		s, ok := porCofre[p.Cofre]
		if !ok {
			s = &situacaoDoCofre{Cofre: p.Cofre}
			porCofre[p.Cofre] = s
			ordem = append(ordem, p.Cofre)
		}
		switch {
		case p.Modo == "":
			s.SemModo = append(s.SemModo, p)
		case instalar.GravaCache(p.Modo):
			s.Gravadores = append(s.Gravadores, p)
		}
	}
	sort.Strings(ordem)
	saida := make([]situacaoDoCofre, 0, len(ordem))
	for _, c := range ordem {
		saida = append(saida, *porCofre[c])
	}
	return saida
}

// relatarVerificacoes imprime o relatório do `doctor` em SEÇÕES.
//
// As dezesseis linhas saíam numa lista única, e quem abre o `doctor` já está
// procurando uma coisa: o cofre não abre, ou o daemon não responde. A seção diz
// onde olhar antes de o olho percorrer tudo.
//
// O detalhe de uma linha OK entra na PRÓPRIA linha, apagado, quando é curto: um
// relatório em que cada verificação ocupa duas linhas tem o dobro do tamanho e
// a mesma informação. Aviso e falha mantêm o detalhe embaixo, com o destaque
// que a indentação dá -- eles são o que alguém veio ler.
func relatarVerificacoes(con *console.Stream, results []doctor.Result) {
	var ok, avisos, falhas int
	for _, g := range []string{doctor.GrupoCofre, doctor.GrupoCache, doctor.GrupoDaemon, doctor.GrupoSO} {
		var doGrupo []doctor.Result
		for _, r := range results {
			if r.Grupo == g {
				doGrupo = append(doGrupo, r)
			}
		}
		if len(doGrupo) == 0 {
			continue
		}
		con.Titulo("%s", g)
		for _, r := range doGrupo {
			nome := r.Name
			detalheAbaixo := r.Detail
			if r.Status == doctor.StatusOK && cabeNaLinha(r.Detail) {
				nome += "  " + con.Apagado(r.Detail)
				detalheAbaixo = ""
			}
			switch r.Status {
			case doctor.StatusOK:
				ok++
				con.OK("%s", nome)
			case doctor.StatusWarn:
				avisos++
				con.Warn("%s", nome)
			default:
				falhas++
				con.Err("%s", nome)
			}
			if detalheAbaixo != "" {
				con.Detail("%s", detalheAbaixo)
			}
		}
	}

	con.Line("")
	// Onde o console aguenta emoji, cada numero e seguido do proprio marcador
	// do estado -- o mesmo que abriu as linhas acima. Onde nao aguenta, aviso e
	// falha teriam o mesmo "[!]", e a contagem sai por extenso.
	okS, okP := textos.DoctorResumoOK, textos.DoctorResumoOK
	avS, avP := textos.DoctorResumoAviso, textos.DoctorResumoAvisos
	faS, faP := textos.DoctorResumoFalha, textos.DoctorResumoFalhas
	if console.EmojiNaSaida() {
		m := console.MarcadoresDaSaida()
		okS, okP = strings.TrimSpace(m.OK), strings.TrimSpace(m.OK)
		avS, avP = strings.TrimSpace(m.Aviso), strings.TrimSpace(m.Aviso)
		faS, faP = strings.TrimSpace(m.Erro), strings.TrimSpace(m.Erro)
	}
	con.Info(textos.DoctorResumo, ok+avisos+falhas,
		contagem(con, ok, okS, okP, con.Verde),
		contagem(con, avisos, avS, avP, con.Amarelo),
		contagem(con, falhas, faS, faP, con.Vermelho))
}

// contagem escreve "N rótulo" e PINTA o número quando ele não é zero.
//
// Zero é a resposta boa para aviso e falha, e uma linha em que todo número tem
// cor não destaca nada: o que precisa saltar aos olhos é o que existe. Zero sai
// apagado, e o singular e o plural saem certos.
func contagem(con *console.Stream, n int, singular, plural string, cor func(string) string) string {
	texto := fmt.Sprintf("%d %s", n, plural)
	if n == 1 {
		texto = fmt.Sprintf("%d %s", n, singular)
	}
	if n == 0 {
		return con.Dim(texto)
	}
	return cor(texto)
}

// cabeNaLinha decide se um detalhe pode ser colado ao nome da verificação. Uma
// linha só, e curta: o log do daemon traz três linhas próprias, e o caminho de
// um cache passa da largura da moldura.
func cabeNaLinha(detalhe string) bool {
	return detalhe != "" && !strings.Contains(detalhe, "\n") && len([]rune(detalhe)) <= 56
}

// linhasPorCofre resume os processos vivos: uma linha por cofre, com quantos
// são de cada modo.
//
// Medido em 2026-09-15 na máquina do dono: 28 processos, 28 linhas, quatro
// cofres. A pergunta que alguém faz olhando isso ("quantos servem o Estudo, e
// algum grava o cache duas vezes?") exigia contar à mão.
func linhasPorCofre(con *console.Stream, vivos []instalar.Presenca) []string {
	type resumo struct {
		porModo  map[string]int
		versoes  map[string]bool
		total    int
		gravando int
	}
	porCofre := map[string]*resumo{}
	var ordem []string
	for _, p := range vivos {
		cofre := p.Cofre
		if cofre == "" {
			cofre = textos.DoctorSemCofre
		}
		r, ok := porCofre[cofre]
		if !ok {
			r = &resumo{porModo: map[string]int{}, versoes: map[string]bool{}}
			porCofre[cofre] = r
			ordem = append(ordem, cofre)
		}
		modo := p.Modo
		if modo == "" {
			modo = textos.DoctorModoNaoRegistrado
		}
		r.porModo[modo]++
		r.total++
		if p.Versao != "" {
			r.versoes[p.Versao] = true
		}
		if instalar.GravaCache(modo) {
			r.gravando++
		}
	}
	sort.Strings(ordem)

	linhas := make([]string, 0, len(ordem))
	for _, cofre := range ordem {
		r := porCofre[cofre]
		modos := make([]string, 0, len(r.porModo))
		for m := range r.porModo {
			modos = append(modos, m)
		}
		sort.Strings(modos)
		// Mais de um processo do mesmo modo no mesmo cofre é o que se procura
		// aqui -- dois gravadores gravam o mesmo cache --, então é esse número
		// que ganha cor. Um só é o estado normal e sai sem destaque.
		partes := make([]string, 0, len(modos))
		for _, m := range modos {
			n := r.porModo[m]
			contador := fmt.Sprintf("%d", n)
			if n > 1 {
				contador = con.Amarelo(contador)
			}
			partes = append(partes, contador+" "+con.Marcado(rotuloDeModo(m, n)))
		}
		versoes := make([]string, 0, len(r.versoes))
		for v := range r.versoes {
			versoes = append(versoes, v)
		}
		sort.Strings(versoes)
		linhas = append(linhas,
			fmt.Sprintf("  %-24s %-34s %s",
				filepath.Base(cofre), strings.Join(partes, ", "), con.Dim(strings.Join(versoes, " "))))
	}
	return linhas
}

// rotuloDeModo escreve o modo como ele aparece na tela, no singular ou no
// plural. "2 ponte" era o que a primeira redação mostrava, e a linha existe
// justamente para ser lida de relance.
//
// "em-processo" é a constante que a presença grava; na tela ele vira
// "servidor em processo", que é o que a palavra significa para quem lê.
func rotuloDeModo(modo string, n int) string {
	switch modo {
	case instalar.ModoDaemon:
		return plural(n, textos.DoctorModoDaemon, textos.DoctorModoDaemons)
	case instalar.ModoPonte:
		return plural(n, textos.DoctorModoPonte, textos.DoctorModoPontes)
	case instalar.ModoEmProcesso:
		return plural(n, textos.DoctorModoEmProcesso, textos.DoctorModoEmProcessos)
	default:
		return modo
	}
}

func plural(n int, singular, plural string) string {
	if n == 1 {
		return singular
	}
	return plural
}

// listarNumeros escreve uma lista de PIDs para caber numa linha de moldura.
func listarNumeros(pids []int) string {
	partes := make([]string, 0, len(pids))
	for _, p := range pids {
		partes = append(partes, fmt.Sprintf("%d", p))
	}
	return fmt.Sprintf(textos.DoctorPids, strings.Join(partes, ", "))
}

func listarPIDs(ps []instalar.Presenca) string {
	partes := make([]string, 0, len(ps))
	for _, p := range ps {
		partes = append(partes, fmt.Sprintf(textos.DoctorPid, p.PID))
	}
	return strings.Join(partes, ", ")
}

// relatarProcessosSemPresenca mostra os gobsidian.exe que a presenca nao ve.
//
// Medido em 2026-09-14: cinco processos v1.5.1 do Claude Code serviam cofres
// fora da lista acima, porque a presenca entrou depois deles. A listagem nao
// decide nada; ela so torna visivel quem a presenca nao registra.
func relatarProcessosSemPresenca(con *console.Stream, vivos []instalar.Presenca) {
	processos, err := instalar.ProcessosDoSistema()
	switch {
	case errors.Is(err, instalar.ErrProcessosNaoVerificados):
		con.Detail("%s", textos.DoctorSemPresencaPlataforma)
		return
	case err != nil:
		con.Warn(textos.DoctorSemPresencaErro, err)
		return
	}
	sem := instalar.SemPresenca(processos, vivos, os.Getpid())
	if len(sem) == 0 {
		con.OK("%s", textos.DoctorSemPresencaNenhum)
		con.Detail("%s", textos.DoctorSemPresencaVazio)
		return
	}
	// Agrupado por EXECUTAVEL: cinco processos do mesmo binario antigo sao uma
	// linha e um fato ("o Claude Code ainda roda a v1.5.1"), e nao cinco linhas
	// que o leitor precisa comparar entre si.
	con.Warn(textos.DoctorSemPresencaContagem, len(sem))
	porExecutavel := map[string][]int{}
	var ordem []string
	for _, p := range sem {
		if _, visto := porExecutavel[p.Executavel]; !visto {
			ordem = append(ordem, p.Executavel)
		}
		porExecutavel[p.Executavel] = append(porExecutavel[p.Executavel], p.PID)
	}
	sort.Strings(ordem)
	linhas := make([]string, 0, len(ordem))
	for _, exe := range ordem {
		pids := porExecutavel[exe]
		sort.Ints(pids)
		linhas = append(linhas, fmt.Sprintf("  %-3d %s", len(pids), exe))
		linhas = append(linhas, "      "+con.Dim(listarNumeros(pids)))
	}
	con.Bloco("", linhas, textos.DoctorSemPresencaRodape)
}

// relatarHostsDeOutroBinario mostra as entradas de host que nao rodam o binario
// instalado (G7).
//
// Medido em 2026-09-14: o Claude Code rodava v1.5.1 de C:\Program Files\gobsidian
// enquanto o instalado era v1.8.1, e nada no `doctor` dizia. So relata: quem
// reconfigura e `install`.
func relatarHostsDeOutroBinario(con *console.Stream, vivos []instalar.Presenca) {
	m, err := instalar.LerManifesto()
	switch {
	case errors.Is(err, instalar.ErrSemManifesto) || (err == nil && m.Binario == ""):
		con.Detail("%s", textos.DoctorHostsSemManifesto)
		return
	case err != nil:
		con.Warn(textos.DoctorHostsErro, err)
		return
	}

	outros := instalar.EntradasDeOutroBinario(hosts.AmbienteReal(), m.Binario)
	if len(outros) == 0 {
		con.OK("%s", textos.DoctorHostsOK)
		con.Detail(textos.DoctorHostsDetalhe, m.Binario)
		return
	}

	// Erro na listagem so tira a versao; a entrada continua sendo relatada.
	processos, _ := instalar.ProcessosDoSistema()
	con.Warn(textos.DoctorHostsOutroBinario, len(outros))
	linhas := make([]string, 0, len(outros))
	for _, o := range outros {
		versao := textos.DoctorVersaoNaoMedida
		if o.Executavel == "" {
			versao = textos.DoctorComandoAusente
		} else if v := instalar.VersaoDoExecutavel(o.Executavel, vivos, processos); v != "" {
			versao = v
		}
		linhas = append(linhas, fmt.Sprintf("  %-16s %-24s %-22s %s", o.Host, o.Chave, versao, o.Comando))
	}
	con.Bloco("", linhas, fmt.Sprintf(
		textos.DoctorHostsRodape,
		m.Binario, m.Versao))
}
