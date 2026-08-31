// Package service e a fachada unica sobre os subsistemas. Cada tool MCP
// corresponde a um metodo aqui. Nenhum tipo do SDK de MCP atravessa esta
// fronteira: o pacote fala Go de dominio, e a traducao acontece em mcpsrv.
package service

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/jonyd/gobsidian/internal/index"
)

// Code e o codigo legivel por maquina devolvido ao cliente. A tabela completa
// esta em docs/TOOLS.md.
type Code string

// A tabela de codigos. Sao contrato publico: o cliente ramifica em cima
// deles, entao renomear um e quebra de compatibilidade, nao refatoracao.
// A tabela completa, com o que cada um significa para quem chama, esta em
// docs/TOOLS.md.
const (
	CodePathOutsideVault Code = "PATH_OUTSIDE_VAULT"
	CodeNoteNotFound     Code = "NOTE_NOT_FOUND"
	CodeNoteExists       Code = "NOTE_ALREADY_EXISTS"
	CodeAmbiguousPath    Code = "AMBIGUOUS_PATH"
	CodeHeadingNotFound  Code = "HEADING_NOT_FOUND"
	CodeAmbiguousHeading Code = "AMBIGUOUS_HEADING"
	CodeBlockNotFound    Code = "BLOCK_NOT_FOUND"
	CodeAmbiguousBlock   Code = "AMBIGUOUS_BLOCK"
	CodeFolderNotFound   Code = "FOLDER_NOT_FOUND"
	CodeHashMismatch     Code = "HASH_MISMATCH"
	CodeFileLocked       Code = "FILE_LOCKED"
	CodeCloudOnlyFile    Code = "CLOUD_ONLY_FILE"
	CodePathTooLong      Code = "PATH_TOO_LONG"
	CodeReadOnlyMode     Code = "READ_ONLY_MODE"
	// CodeInvalidArgument: a combinacao de parametros recebida nao tem
	// interpretacao unica (dois campos mutuamente exclusivos preenchidos) ou
	// viola um teto declarado no schema (lote maior que o limite). Nao e
	// PATH_OUTSIDE_VAULT nem NOTE_NOT_FOUND porque a falha esta na chamada em
	// si, antes de qualquer nome de arquivo ser resolvido.
	CodeInvalidArgument Code = "INVALID_ARGUMENT"
	// CodeIndexBuilding: o indice invertido ainda nao cobre o cofre inteiro.
	// E um erro, e nao uma lista vazia, porque "ainda nao sei" e "nao achei
	// nada" pedem acoes diferentes de quem chama: uma manda tentar de novo,
	// a outra manda mudar a consulta.
	CodeIndexBuilding    Code = "INDEX_BUILDING"
	CodeVaultUnavailable Code = "VAULT_UNAVAILABLE"
	CodeInternal         Code = "INTERNAL"
)

// Error carrega codigo e mensagem acionavel. A mensagem e lida por um modelo
// de linguagem que precisa decidir o que fazer em seguida: "heading nao
// encontrado" gera uma rodada extra de chamadas, enquanto a mesma mensagem
// listando os headings disponiveis permite que o cliente se corrija sozinho.
type Error struct {
	Code    Code
	Message string
	Err     error
}

func (e *Error) Error() string { return e.Message }
func (e *Error) Unwrap() error { return e.Err }

// Is permite errors.Is(err, &service.Error{Code: service.CodeNoteNotFound})
// sem exigir mensagem ou causa identicas — o codigo e o unico campo que
// identifica a categoria do erro para quem chama.
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	if !ok || t == nil {
		return false
	}
	return e.Code == t.Code
}

// Errorf monta um Error sem causa subjacente, para a falha que se origina
// aqui mesmo. Quando houver um erro a preservar, use Wrap.
func Errorf(code Code, format string, args ...any) *Error {
	return &Error{Code: code, Message: sprintf(format, args...)}
}

// Wrap monta um Error preservando a causa, que fica alcancavel por
// errors.Is e errors.As. A mensagem e lida por um modelo de linguagem que
// precisa decidir o proximo passo, entao ela deve dizer o que fazer, nao so
// o que falhou.
func Wrap(code Code, err error, format string, args ...any) *Error {
	return &Error{Code: code, Message: sprintf(format, args...), Err: err}
}

// CodeOf extrai o codigo de um erro, ou INTERNAL se ele nao carregar um.
func CodeOf(err error) Code {
	var e *Error
	if errors.As(err, &e) {
		return e.Code
	}
	return CodeInternal
}

func sprintf(format string, args ...any) string { return fmt.Sprintf(format, args...) }

// ErroDeResolucao traduz a falha de index.ResolvePath no erro de tool certo.
//
// É a ÚNICA função que faz essa classificação. Havia seis chamadores e três
// respostas diferentes para a mesma falha: `read.go` distinguia ambiguidade,
// `../` e não-encontrado; `write.go` distinguia ambiguidade e devolvia
// `NOTE_NOT_FOUND` para o resto; e `graph.go` respondia `PATH_OUTSIDE_VAULT`
// para tudo, inclusive nota inexistente (achado M2).
//
// O terceiro caso não é só inconsistência: `PATH_OUTSIDE_VAULT` é um erro de
// SEGURANÇA, e o host o lê como tentativa de escapar do cofre. Errar um nome de
// nota passava a acusar o cliente de algo que ele não fez.
//
// `ResolvePath` não verifica confinamento — só procura no índice —, então
// "não encontrado" é a única coisa que ela pode afirmar. O `../` continua sendo
// classificado à parte porque é sinal útil para quem depura, e porque perdê-lo
// tornaria a mensagem menos informativa do que já era; mas ele é uma heurística
// sobre o TEXTO da entrada, e não um veredito de confinamento. A guarda de
// verdade mora em `internal/vault`.
func ErroDeResolucao(entrada string, err error) error {
	switch {
	case errors.Is(err, index.ErrAmbiguousPath):
		return Errorf(CodeAmbiguousPath, "caminho %q resolve para mais de um arquivo", entrada)
	case strings.Contains(entrada, "../"):
		return Errorf(CodePathOutsideVault, "caminho %q sai do cofre", entrada)
	default:
		return Errorf(CodeNoteNotFound, "nota %q nao encontrada no indice", entrada)
	}
}

// LimitePadrao e LimiteTeto são os valores que `docs/TOOLS.md` publica para as
// tools de lista. Ficam aqui, e não repetidos em cada tool, porque o teto
// declarado no schema e o teto aplicado no código têm de ser o mesmo número —
// o achado B4 era exatamente os dois divergindo: `note_list` declarava
// `"maximum": 500` e nunca clampava.
const (
	LimitePadrao = 100
	LimiteTeto   = 500
)

// ComTeto aplica o padrão e o teto de um `limit`.
//
// É a ÚNICA conta desse clamp. `link_graph` fazia o dele inline com números
// mágicos e `note_list` não fazia nenhum, apesar de os dois declararem o mesmo
// `"maximum": 500` no schema. Schema que promete um teto e código que não o
// aplica é a mesma classe do achado M4: o cliente lê o schema para decidir.
func ComTeto(pedido int) int {
	if pedido <= 0 {
		return LimitePadrao
	}
	if pedido > LimiteTeto {
		return LimiteTeto
	}
	return pedido
}

// ValidarEnum confere um parâmetro de conjunto fechado.
//
// Vazio significa "não informado" e devolve o padrão — é assim que o CLI e o
// boundary MCP chegam aqui quando o campo foi omitido. Valor FORA do conjunto
// é erro, e não silêncio: até 2026-08-28 `tag_mode`, `sort`, `order` e
// `direction` caíam num `default` do switch e o pedido virava outra coisa sem
// aviso (achado B4). O modelo do outro lado lê o enum do schema para decidir o
// que pedir; se um valor inválido responde como se fosse válido, ele não tem
// como saber que o pedido não fez o que dizia.
func ValidarEnum(campo, valor, padrao string, aceitos ...string) (string, error) {
	if valor == "" {
		return padrao, nil
	}
	if slices.Contains(aceitos, valor) {
		return valor, nil
	}
	return "", Errorf(CodeInvalidArgument, "%s = %q invalido; aceitos: %s",
		campo, valor, strings.Join(aceitos, ", "))
}
