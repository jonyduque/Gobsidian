package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jonyduque/Gobsidian/internal/boot"
	"github.com/jonyduque/Gobsidian/internal/config"
	"github.com/jonyduque/Gobsidian/internal/console"
	"github.com/jonyduque/Gobsidian/internal/mcpsrv"
	"github.com/jonyduque/Gobsidian/internal/textos"
)

// ferramenta liga uma tool do servidor MCP a um comando de CLI.
//
// # Por que a CLI chama a tool, e nao o servico
//
// Um comando que chamasse service direto teria de repetir o que o handler da
// tool faz -- os padroes de cada parametro, a validacao de schema do SDK, o
// codigo de erro. Seriam duas contas da mesma entrada, e a da CLI divergiria
// na primeira tool que ganhasse um padrao novo. Aqui o comando monta os
// argumentos e chama a tool pelo transporte em memoria (mcpsrv.ChamarLocal): o
// que sai e o que o host recebe.
type ferramenta struct {
	tool   string
	grupo  string // "" na raiz, ou o comando pai ("note", "tag")
	nome   string
	resumo string
	// posicionais sao parametros da tool que vao pela linha de comando sem
	// flag, na ordem.
	posicionais []string
	// plural, quando nao vazio, faz o ULTIMO posicional aceitar varios valores:
	// um vira o parametro singular, dois ou mais viram este. E note_read, cujo
	// path e paths sao mutuamente exclusivos.
	plural string
	// comTeto diz que a tool obedece --max-results. So vault_search obedece
	// (service.Search); declarar a flag nas outras seria prometer um teto que
	// ninguem aplica -- o achado 5.8, que tirou --max-results de index e
	// inspect.
	comTeto bool
}

// ferramentas e a tabela do desenho (docs/superpowers/specs/
// 2026-09-25-cli-das-tools-design.md). TestCadaToolTemComando prova que ela
// e a lista de tools do servidor, nos dois sentidos.
var ferramentas = []ferramenta{
	{tool: "note_read", grupo: "note", nome: "read", resumo: textos.ResumoNoteRead, posicionais: []string{"path"}, plural: "paths"},
	{tool: "note_list", grupo: "note", nome: "list", resumo: textos.ResumoNoteList},
	{tool: "note_outline", grupo: "note", nome: "outline", resumo: textos.ResumoNoteOutline, posicionais: []string{"path"}},
	{tool: "note_metadata", grupo: "note", nome: "metadata", resumo: textos.ResumoNoteMetadata, posicionais: []string{"path"}},
	{tool: "note_create", grupo: "note", nome: "create", resumo: textos.ResumoNoteCreate, posicionais: []string{"path"}},
	{tool: "note_append", grupo: "note", nome: "append", resumo: textos.ResumoNoteAppend, posicionais: []string{"path"}},
	{tool: "note_patch", grupo: "note", nome: "patch", resumo: textos.ResumoNotePatch, posicionais: []string{"path"}},
	{tool: "note_move", grupo: "note", nome: "move", resumo: textos.ResumoNoteMove, posicionais: []string{"from", "to"}},
	{tool: "note_delete", grupo: "note", nome: "delete", resumo: textos.ResumoNoteDelete, posicionais: []string{"path"}},
	{tool: "vault_stats", nome: "stats", resumo: textos.ResumoStats},
	{tool: "vault_search", nome: "search", resumo: textos.ResumoSearch, posicionais: []string{"query"}, comTeto: true},
	{tool: "vault_broken_links", nome: "broken-links", resumo: textos.ResumoBrokenLinks},
	{tool: "tag_list", grupo: "tag", nome: "list", resumo: textos.ResumoTagList},
	{tool: "link_graph", nome: "graph", resumo: textos.ResumoGraph, posicionais: []string{"path"}},
}

// gruposDeFerramentas sao os comandos pais, que so agrupam.
var gruposDeFerramentas = map[string]string{
	"note": textos.ResumoGrupoNote,
	"tag":  textos.ResumoGrupoTag,
}

// montarFerramentas e falso em `serve` e `daemon` (main.go): ler os esquemas
// custa ~30 ms medidos em 2026-09-27, e um servidor que o host abre a cada
// sessao nao tem por que pagar pela arvore de comandos de outra pessoa.
var montarFerramentas = true

// acrescentarFerramentas pendura um comando por tool na raiz.
//
// Falha aqui e erro de PROGRAMACAO -- os esquemas vem de tipos estaticos --, e
// o mesmo contrato de mcpsrv.New, que ja entra em panico por schema invalido.
func acrescentarFerramentas(root *cobra.Command) {
	esquemas, err := mcpsrv.EsquemasDeEntrada()
	if err != nil {
		panic(fmt.Sprintf("esquemas das tools: %v", err))
	}
	porNome := map[string]mcpsrv.EsquemaDeTool{}
	for _, e := range esquemas {
		porNome[e.Nome] = e
	}

	pais := map[string]*cobra.Command{}
	for _, f := range ferramentas {
		e, ok := porNome[f.tool]
		if !ok {
			// A tabela cita uma tool que o servidor nao registra. O teste de
			// cobertura nomeia o caso; aqui o comando simplesmente nao existe.
			continue
		}
		pai := root
		if f.grupo != "" {
			if pais[f.grupo] == nil {
				pais[f.grupo] = &cobra.Command{Use: f.grupo, Short: gruposDeFerramentas[f.grupo], Args: semArgumentoDeUso}
				root.AddCommand(pais[f.grupo])
			}
			pai = pais[f.grupo]
		}
		pai.AddCommand(comandoDeFerramenta(f, e))
	}
}

// caminhoDoComando e como o usuario chama a tool: "note read", "stats".
func (f ferramenta) caminhoDoComando() string {
	return strings.TrimSpace(f.grupo + " " + f.nome)
}

func (f ferramenta) uso() string {
	u := f.nome
	for i, p := range f.posicionais {
		if f.plural != "" && i == len(f.posicionais)-1 {
			u += " <" + p + ">..."
			continue
		}
		u += " <" + p + ">"
	}
	return u
}

// validarPosicionais: com --args, a entrada pode vir inteira no JSON, e os
// posicionais passam a ser opcionais.
func (f ferramenta) validarPosicionais(cmd *cobra.Command, args []string) error {
	n := len(f.posicionais)
	minimo := n
	if cmd.Flags().Changed("args") {
		minimo = 0
	}
	switch {
	case f.plural != "":
		if len(args) < minimo {
			return erroDeUso(cobra.MinimumNArgs(minimo)(cmd, args))
		}
		return nil
	default:
		return erroDeUso(cobra.RangeArgs(minimo, n)(cmd, args))
	}
}

// comandoDeFerramenta monta o comando de uma tool a partir do schema dela.
func comandoDeFerramenta(f ferramenta, e mcpsrv.EsquemaDeTool) *cobra.Command {
	var (
		flags    config.Flags
		saida    opcoesDeSaida
		argsJSON string
	)

	cmd := &cobra.Command{
		Use:   f.uso(),
		Short: f.resumo,
		Args:  f.validarPosicionais,
	}

	pular := map[string]bool{}
	for _, p := range f.posicionais {
		pular[p] = true
	}
	if f.plural != "" {
		pular[f.plural] = true
	}
	parametros := registrarParametros(cmd, e, pular)

	flagsDeCofreDaCLI(cmd, &flags)
	flagsDeCache(cmd, &flags)
	if f.comTeto {
		cmd.Flags().IntVar(&flags.MaxResults, "max-results", 0, textos.FlagMaxResults)
	}
	// --read-only so onde ele muda alguma coisa: numa tool de leitura ele nao
	// teria o que recusar. GOBSIDIAN_READ_ONLY vale nas duas, por config.Load.
	if e.Escrita {
		cmd.Flags().BoolVar(&flags.ReadOnly, "read-only", false, textos.FlagReadOnly)
	}
	cmd.Flags().StringVar(&argsJSON, "args", "", textos.FlagArgs)
	saida.registrar(cmd)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		emJSON, err := saida.emJSON(cmd)
		if err != nil {
			return err
		}
		entrada, err := montarEntrada(cmd, f, argsJSON, args, parametros)
		if err != nil {
			return err
		}

		flags.ReadOnlySet = cmd.Flags().Changed("read-only")
		flags.MaxResultsSet = cmd.Flags().Changed("max-results")
		cfg, err := carregarConfig(flags)
		if err != nil {
			return err
		}
		// A tool de escrita nao existe no modo somente leitura -- o servidor
		// nem a registra. Dizer isso e melhor que "tool desconhecida".
		if e.Escrita && cfg.ReadOnly {
			return erroDeUso(fmt.Errorf(textos.ErroToolSomenteLeitura, f.caminhoDoComando()))
		}

		log := loggerDeCLI(cmd, cfg)
		svc, fechar, err := boot.AbrirServicoDeCLI(cmd.Context(), cfg, log)
		if err != nil {
			return err
		}
		defer fechar()

		raw, err := mcpsrv.ChamarLocal(cmd.Context(), svc, cfg, log, f.tool, entrada)
		return imprimirResultado(cmd, f.tool, emJSON, raw, err)
	}
	return cmd
}

// montarEntrada junta os argumentos da tool: o JSON de --args por baixo, os
// posicionais e as flags PASSADAS por cima.
//
// So a flag passada entra. Flag omitida deixa a tool aplicar o padrao dela --
// mandar o valor zero de cada flag faria "include_frontmatter" chegar false e
// "limit" chegar 0 em toda chamada, e a CLI divergiria do host justamente nos
// padroes.
func montarEntrada(cmd *cobra.Command, f ferramenta, argsJSON string, args []string, parametros []*parametroDeFlag) (map[string]any, error) {
	entrada := map[string]any{}
	if argsJSON != "" {
		if err := json.Unmarshal([]byte(argsJSON), &entrada); err != nil {
			return nil, erroDeUso(fmt.Errorf(textos.ErroArgsInvalido, err))
		}
		if entrada == nil {
			entrada = map[string]any{}
		}
	}

	for i, nome := range f.posicionais {
		if i >= len(args) {
			break
		}
		if f.plural != "" && i == len(f.posicionais)-1 && len(args)-i > 1 {
			varios := make([]any, 0, len(args)-i)
			for _, a := range args[i:] {
				varios = append(varios, a)
			}
			entrada[f.plural] = varios
			delete(entrada, nome)
			break
		}
		entrada[nome] = args[i]
	}

	for _, p := range parametros {
		if !cmd.Flags().Changed(p.flag) {
			continue
		}
		v, err := p.ler()
		if err != nil {
			return nil, erroDeUso(err)
		}
		entrada[p.param.Nome] = v
	}

	if conteudo, ok := entrada["content"].(string); ok && conteudo == "-" && cmd.Flags().Changed("content") {
		b, err := io.ReadAll(cmd.InOrStdin())
		if err != nil {
			return nil, fmt.Errorf(textos.ErroLendoStdin, err)
		}
		entrada["content"] = string(b)
	}
	return entrada, nil
}

// imprimirResultado escreve o que a tool devolveu na forma pedida.
//
// Erro da tool sai como falha com o codigo de dominio -- no JSON, um objeto
// "error" em stdout, que e onde o script le; no terminal, a linha de falha em
// stderr. Os dois saem com codigo 1.
func imprimirResultado(cmd *cobra.Command, tool string, emJSON bool, raw json.RawMessage, err error) error {
	if err != nil {
		var et *mcpsrv.ErroDeTool
		if !errors.As(err, &et) {
			return err
		}
		if emJSON {
			b, _ := json.Marshal(map[string]any{"error": map[string]string{"code": et.Codigo, "message": et.Mensagem}})
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(b))
		} else {
			console.New(cmd.ErrOrStderr()).Err(textos.ErroDaTool, comMaiuscula(et.Mensagem), et.Codigo)
		}
		return &erroComCodigo{codigo: saidaErro, err: err, impresso: true}
	}

	// Os comandos das tools imprimem em stdout de proposito: sao CLI.
	if emJSON {
		var b bytes.Buffer
		if err := json.Compact(&b, raw); err != nil {
			return err
		}
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), b.String())
		return nil
	}
	return formatar(cmd.OutOrStdout(), tool, raw)
}
