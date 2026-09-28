package main

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jonyduque/Gobsidian/internal/console"
	"github.com/jonyduque/Gobsidian/internal/textos"
)

// Os codigos de saida dos comandos de CLI. Um script distingue "a tool
// respondeu com erro" de "chamei errado" pelo codigo, sem ler a mensagem.
const (
	saidaErro = 1 // a operacao falhou: erro da tool, do disco, do sistema
	saidaUso  = 2 // o comando foi chamado errado, ou o cofre nao resolve
)

// erroComCodigo carrega o codigo de saida junto do erro.
//
// impresso diz que o comando ja escreveu a falha -- no JSON de erro em stdout,
// por exemplo -- e main so precisa sair com o codigo, sem repetir a linha.
type erroComCodigo struct {
	codigo   int
	err      error
	impresso bool
}

func (e *erroComCodigo) Error() string { return e.err.Error() }
func (e *erroComCodigo) Unwrap() error { return e.err }

// erroDeUso e o erro de quem chamou o comando errado: codigo 2.
func erroDeUso(err error) error {
	if err == nil {
		return nil
	}
	var ec *erroComCodigo
	if errors.As(err, &ec) {
		return err
	}
	return &erroComCodigo{codigo: saidaUso, err: err}
}

// codigoDeSaida decide com que codigo o processo sai, e se main ainda precisa
// imprimir a falha.
func codigoDeSaida(err error) (codigo int, imprimir bool) {
	var ec *erroComCodigo
	if errors.As(err, &ec) {
		return ec.codigo, !ec.impresso
	}
	return saidaErro, true
}

// argsDeUso embrulha um validador de posicionais do cobra: posicional faltando
// ou sobrando e uso errado, codigo 2.
func argsDeUso(v cobra.PositionalArgs) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		return erroDeUso(v(cmd, args))
	}
}

// semArgumentoDeUso e cobra.NoArgs com o codigo de uso.
var semArgumentoDeUso = argsDeUso(cobra.NoArgs)

// opcoesDeSaida sao --json e --texto, que forcam uma das duas formas.
//
// Sem nenhuma das duas, a forma sai do destino: terminal recebe texto para
// ler, pipe ou arquivo recebe JSON numa linha -- o que um script passa a jq.
// A pergunta e se o destino e um TERMINAL, e nao se ha cor: NO_COLOR desliga
// a cor e nao transforma o terminal num pipe.
type opcoesDeSaida struct {
	json  bool
	texto bool
}

func (o *opcoesDeSaida) registrar(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&o.json, "json", false, textos.FlagJSON)
	cmd.Flags().BoolVar(&o.texto, "texto", false, textos.FlagTexto)
}

func (o *opcoesDeSaida) emJSON(cmd *cobra.Command) (bool, error) {
	switch {
	case o.json && o.texto:
		return false, erroDeUso(errors.New(textos.ErroJSONETexto))
	case o.json:
		return true, nil
	case o.texto:
		return false, nil
	}
	return !console.EhTerminal(cmd.OutOrStdout()), nil
}

// erroDeFlag e o SetFlagErrorFunc da raiz: flag desconhecida ou valor
// invalido sao uso errado.
func erroDeFlag(_ *cobra.Command, err error) error {
	return erroDeUso(fmt.Errorf("%w", err))
}
