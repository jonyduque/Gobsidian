package main

import (
	"os"

	"fmt"
	"github.com/jonyd/gobsidian/internal/config"
	"github.com/jonyd/gobsidian/internal/console"
	"github.com/jonyd/gobsidian/internal/doctor"
	"github.com/jonyd/gobsidian/internal/instalar"
	"github.com/spf13/cobra"
)

func newDoctorCmd() *cobra.Command {
	var flags config.Flags

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Diagnostica o ambiente: permissoes, OneDrive, MAX_PATH, casing",
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Sem esta linha, --read-only nao chega a Config e a verificacao
			// de permissao de escrita roda mesmo quando o usuario pediu para
			// nao rodar. Toda chamada a config.Load precisa preencher os
			// companheiros das flags que o comando expoe.
			flags.ReadOnlySet = cmd.Flags().Changed("read-only")
			flags.MaxResultsSet = cmd.Flags().Changed("max-results")

			cfg, err := config.Load(flags)
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
			con.Titulo("Diagnostico do ambiente")
			for _, r := range results {
				switch r.Status {
				case doctor.StatusOK:
					con.OK("%s", r.Name)
				case doctor.StatusWarn:
					con.Warn("%s", r.Name)
				default:
					con.Err("%s", r.Name)
				}
				if r.Detail != "" {
					con.Detail("%s", r.Detail)
				}
			}

			relatarProcessosELixo(con, corrigir)

			code := doctor.ExitCode(results)
			if code != 0 {
				con.Err("Ha falhas bloqueantes acima")
				os.Exit(code)
			}
			con.OK("Ambiente apto")
			return nil
		},
	}

	flagsDeCofre(cmd, &flags)
	cmd.Flags().BoolVar(&flags.ReadOnly, "read-only", false, "nao verifica permissao de escrita")
	cmd.Flags().IntVar(&flags.MaxResults, "max-results", 0, "teto de resultados por consulta")
	cmd.Flags().BoolVar(&corrigir, "fix", false,
		"alem de diagnosticar, remove o lixo comprovadamente orfao do diretorio de runtime e do cache")

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
		con.Warn("processos e lixo: diretorio de runtime indisponivel (%v)", err)
		return
	}

	vivos, err := instalar.Vivos(runtimeDir)
	switch {
	case err != nil:
		con.Warn("processos do gobsidian: %v", err)
	case len(vivos) == 0:
		con.OK("processos do gobsidian")
		con.Detail("nenhum rodando")
	default:
		con.OK("processos do gobsidian")
		linhas := make([]string, 0, len(vivos))
		for _, p := range vivos {
			linhas = append(linhas, fmt.Sprintf("  pid %-7d %-8s %-12s %s", p.PID, p.Papel, p.Versao, p.Cofre))
		}
		con.Bloco("", linhas, "")
		// Dois processos servindo o MESMO cofre gravam o mesmo cache de busca.
		// E o estado medido em 2026-09-08, e ate hoje ele so aparecia por
		// comparacao de milissegundos entre linhas de log.
		for cofre, n := range contarPorCofre(vivos) {
			if n > 1 {
				con.Warn("%d processos servem o mesmo cofre", n)
				con.Detail("%s -- eles gravam o MESMO cache de busca; encerre os extras", cofre)
			}
		}
	}

	// Chaves de cache que ficaram para tras da conta de config.VaultKey.
	//
	// SEMPRE em simulacao aqui, mesmo com -Aplicar: renomear um diretorio de
	// cache enquanto um daemon o mapeia com mmap e pedir problema, e `doctor`
	// roda com o produto no ar. Quem renomeia e `install`/`update`, sob a
	// trava global e com nenhum processo vivo.
	if migradas, err := instalar.MigrarChaves(instalar.RaizDoCache(), false); err != nil {
		con.Warn("chaves de cache: %v", err)
	} else if len(migradas) > 0 {
		con.Warn("%d cache(s) sob chave superada", len(migradas))
		linhas := make([]string, 0, len(migradas))
		for _, m := range migradas {
			linhas = append(linhas, fmt.Sprintf("  %s -> %s  %s", m.De, m.Para, con.Dim(m.Cofre)))
		}
		con.Bloco("", linhas, "rode `gobsidian update` para renomear com tudo encerrado")
	}

	r, err := instalar.Limpar(runtimeDir, instalar.RaizDoCache(), aplicar)
	if err != nil {
		con.Warn("lixo do diretorio de runtime: %v", err)
		return
	}
	if r.Vazio() {
		con.OK("lixo de execucoes anteriores")
		con.Detail("nada a remover")
		return
	}

	verbo := "removivel"
	if aplicar {
		verbo = "removido"
	}
	con.Warn("lixo de execucoes anteriores")
	con.Campos("", []console.Campo{
		console.Campof("travas", "%d", len(r.Locks)),
		console.Campof("sockets", "%d", len(r.Sockets)),
		console.Campof("presencas", "%d", len(r.Presencas)),
		{Chave: "caches", Valor: fmt.Sprintf("%d", len(r.Caches)), Nota: "de cofre inexistente"},
		console.Campof("logs rotacionados", "%d", len(r.LogsRotacionados)),
		{Chave: "total", Valor: fmt.Sprintf("%d KB", r.Bytes/1024), Nota: verbo},
	})
	if !aplicar {
		con.Detail("rode `gobsidian doctor --fix` para remover")
	}
	for _, falha := range r.NaoRemovidos {
		con.Detail("nao removido: %s", falha)
	}
}

// contarPorCofre agrupa presencas por cofre. Chave e o caminho como o processo
// o registrou; comparacao de caminho nao entra aqui porque quem escreveu os
// dois lados foi o mesmo produto, na mesma maquina.
func contarPorCofre(vivos []instalar.Presenca) map[string]int {
	porCofre := map[string]int{}
	for _, p := range vivos {
		if p.Cofre == "" {
			continue
		}
		porCofre[p.Cofre]++
	}
	return porCofre
}
