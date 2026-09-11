// update.go implementa `gobsidian update`: confere a versao publicada, baixa,
// PROVA o SHA-256 e troca o binario.
//
// Este e o unico comando que alcanca internal/selfupdate, e portanto o unico
// caminho por onde o produto fala com a rede (PRD 6.4, segunda excecao da
// RNF-30). `serve` e `daemon` nao o importam.
package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/jonyduque/Gobsidian/internal/console"
	"github.com/jonyduque/Gobsidian/internal/instalar"
	"github.com/jonyduque/Gobsidian/internal/selfupdate"
	"github.com/spf13/cobra"
)

// repositorio e de onde os releases vem. Constante literal, e nao flag: um
// repositorio configuravel transformaria `update` num downloader generico, e a
// excecao da RNF-30 e para atualizar ESTE produto.
const repositorio = "jonyduque/Gobsidian"

func newUpdateCmd() *cobra.Command {
	var apenasConferir bool
	var sim bool

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Atualiza o gobsidian para a ultima versao publicada",
		Long: "Consulta a versao publicada, baixa o binario da plataforma corrente, " +
			"CONFERE o SHA-256 publicado e -- so entao -- encerra os processos em execucao " +
			"e troca o binario. Divergencia de soma aborta sem instalar nada.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return rodarUpdate(cmd.Context(), cmd, apenasConferir, sim)
		},
	}
	cmd.Flags().BoolVar(&apenasConferir, "check", false,
		"so diz se ha versao nova, sem baixar nem instalar")
	cmd.Flags().BoolVar(&sim, "yes", false,
		"nao pergunta antes de encerrar os processos em execucao")
	return cmd
}

func rodarUpdate(ctx context.Context, cmd *cobra.Command, apenasConferir, sim bool) error {
	con := console.New(cmd.OutOrStdout())
	entrada := bufio.NewReader(cmd.InOrStdin())

	con.Step("Consultando a ultima versao publicada")
	transporte := selfupdate.TransporteHTTP{}
	release, err := selfupdate.UltimaVersao(ctx, transporte, repositorio)
	if err != nil {
		return fmt.Errorf("consultando releases: %w", err)
	}

	con.Info("instalada: %s", version)
	con.Info("publicada: %s", release.Tag)

	// selfupdate.PrecisaAtualizar, e nao `release.Tag == version`: a comparacao
	// por igualdade dizia que um build local -- carimbado pelo `git describe` de
	// scripts/build.ps1 como "v1.5.1-20-gcf8991a-dirty" -- estava DESATUALIZADO
	// em relacao a v1.5.1, e `update` faria downgrade. Encontrado rodando
	// `update --check` de verdade em 2026-09-09.
	if !selfupdate.PrecisaAtualizar(version, release.Tag) {
		con.OK("Ja esta na ultima versao")
		return nil
	}
	if apenasConferir {
		con.Warn("Ha versao nova: %s", release.Tag)
		con.Detail("rode `gobsidian update` para instalar")
		return nil
	}

	ativo := nomeDoAtivoDaPlataforma()
	if _, ok := release.Ativos[ativo]; !ok {
		return fmt.Errorf("o release %s nao publica %q (plataforma %s/%s)",
			release.Tag, ativo, runtime.GOOS, runtime.GOARCH)
	}

	// Baixa para um TEMPORARIO e confere ANTES de encerrar processo nenhum.
	//
	// A ordem importa: se o download ou a soma falharem, o usuario continua
	// com a versao que tinha e nenhuma sessao foi derrubada. Encerrar primeiro
	// e baixar depois trocaria uma atualizacao que falha por um ambiente
	// quebrado.
	tmp, err := os.MkdirTemp("", "gobsidian-update-*")
	if err != nil {
		return fmt.Errorf("criando diretorio temporario: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmp) }()

	baixado := filepath.Join(tmp, instalar.NomeDoExecutavel)
	con.Step("Baixando %s e conferindo o SHA-256", ativo)
	if err := selfupdate.Baixar(ctx, transporte, release, ativo, baixado); err != nil {
		if errors.Is(err, selfupdate.ErrHashDivergente) {
			con.Err("O binario baixado NAO confere com a soma publicada")
			con.Detail("nada foi instalado; sua instalacao continua intacta")
		}
		return err
	}
	con.OK("SHA-256 confere")

	sis := instalar.SistemaReal(
		func(pergunta string, itens []string) bool {
			return confirmar(con, entrada, sim, pergunta, itens)
		},
		func(nome string) { con.Passo("%s", nome) },
	)

	// Destino e cofre saem do MANIFESTO: `update` nao pergunta de novo o que
	// `install` ja perguntou. Sem manifesto nao ha o que atualizar.
	m, err := instalar.LerManifesto()
	if err != nil {
		return fmt.Errorf("%w -- rode `gobsidian install` primeiro", err)
	}

	con.Step("Trocando o binario")
	r, err := instalar.Instalar(ctx, sis, instalar.Opcoes{
		Origem:  baixado,
		Destino: filepath.Dir(m.Binario),
		Versao:  release.Tag,
		Cofre:   m.Cofre,
		Hosts:   m.Hosts,
		SemPath: m.PathAdicionado == "",
	})
	if errors.Is(err, instalar.ErrRecusado) {
		con.Warn("Atualizacao cancelada; nada foi alterado")
		return nil
	}
	if err != nil {
		return err
	}

	con.OK("Atualizado para %s", release.Tag)
	imprimirResumo(con, r, cofresDoManifesto(m))
	con.Detail("os hosts reiniciam o servidor sozinhos; nao ha o que fazer a mao")
	return nil
}

// nomeDoAtivoDaPlataforma e o nome do arquivo publicado para esta plataforma.
//
// Os nomes seguem o que .github/workflows/release.yml gera. Eles NAO sao
// derivados de runtime.GOOS solto: um par nao publicado tem de dar erro com
// nome, e nao um 404 sem explicacao.
func nomeDoAtivoDaPlataforma() string {
	switch runtime.GOOS + "/" + runtime.GOARCH {
	case "windows/amd64":
		return "gobsidian-windows-amd64.exe"
	case "darwin/arm64":
		return "gobsidian-darwin-arm64"
	case "linux/amd64":
		return "gobsidian-linux-amd64"
	default:
		return fmt.Sprintf("gobsidian-%s-%s", runtime.GOOS, runtime.GOARCH)
	}
}

// cofresDoManifesto le a lista de cofres do manifesto, aceitando o formato
// antigo de um cofre so.
//
// O manifesto de antes de 2026-09-09 traz `cofre`; o de agora traz `cofres`. Um
// update nao pode falhar por causa de um manifesto que ele mesmo escreveu na
// versao anterior.
func cofresDoManifesto(m instalar.Manifesto) []string {
	if len(m.Cofres) > 0 {
		return m.Cofres
	}
	if m.Cofre != "" {
		return []string{m.Cofre}
	}
	return nil
}
