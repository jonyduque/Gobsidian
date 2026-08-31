package index

import (
	"path/filepath"
	"strings"

	"github.com/jonyd/gobsidian/internal/text"
)

// As chaves derivadas do indice moram todas aqui, e cada uma tem UMA conta.
//
// Ate 2026-08-31 eram quatro derivacoes espalhadas — publishNameLocked,
// aliasKey, nomeChave e ResolvePath — e as quatro aplicavam so caixa. Nenhuma
// aplicava normalizacao Unicode, enquanto text.Normalize e parser.Slug
// aplicavam. As duas convencoes conviviam no mesmo produto sem que nada dissesse
// onde uma acabava.
//
// O sintoma: `Capítulo` gravado em NFD (o que um cofre sincronizado com macOS
// produz) e pedido em NFC (o que um cliente Windows envia) sao strings
// diferentes para um mapa de Go, e ResolvePath devolvia ErrPathNotFound para uma
// nota que existe. Este e um cofre em portugues, onde acento e a regra.
//
// A normalizacao vale para a CHAVE, nunca para o CanonicalPath guardado: o
// caminho gravado continua sendo a grafia do disco, senao o servidor passa a
// abrir arquivo que nao existe.

// chaveDeCaminho e a chave insensivel a caixa de um caminho inteiro — a de
// lowerPath, escrita por publicarNomeLocked e lida por ResolvePath.
func chaveDeCaminho(path string) string {
	return strings.ToLower(text.ParaNFC(filepath.ToSlash(path)))
}

// chaveDeNomeDeArquivo e a chave de byName. Ela baixa a caixa desde 2026-08-31.
//
// Ate ali, guardava e lia a base CRUA, e o efeito era uma incoerencia visivel:
// `ResolvePath("pasta/ACORDAO.MD")` resolvia, porque caminho completo passa por
// lowerPath, e `ResolvePath("acordao")` NAO resolvia, porque nome nu passa por
// aqui. Duas portas para a mesma pergunta, respondendo diferente.
//
// Baixar a caixa aqui so e seguro porque byName e uma LISTA e ResolvePath conta
// os candidatos: onde duas notas passam a colidir na chave, a resposta e
// ErrAmbiguousPath, e nao uma das duas escolhida em silencio.
func chaveDeNomeDeArquivo(base string) string {
	return strings.ToLower(text.ParaNFC(base))
}

// aliasKey normaliza a chave de byAlias. Toda escrita e toda leitura passam
// por aqui: o boot indexava minusculo e Replace indexava cru, e a entrada
// que Remove nao encontrava sobrevivia apontando para uma nota deletada.
func aliasKey(alias string) string {
	return strings.ToLower(text.ParaNFC(alias))
}
