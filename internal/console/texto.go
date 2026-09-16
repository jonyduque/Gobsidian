package console

import "github.com/jonyduque/Gobsidian/internal/text"

// adaptarTexto devolve o texto como ele sai NESTE console.
//
// # Por que existe
//
// A regra do projeto era escrever tudo sem acento, e a razao esta em estilo.go:
// um console em CP-850 renderiza "permissão" como "permissÒo". O custo era que
// toda mensagem do produto saia errada em portugues -- "diretorio", "versao",
// "permissao" -- inclusive nos consoles que aguentam UTF-8, que sao a maioria
// hoje (Windows Terminal, VS Code, qualquer terminal de Unix com locale UTF-8).
//
// A decisao aqui e a MESMA que a dos glifos da moldura, e pelo mesmo motivo:
// medir em vez de supor. Onde a codificacao aguenta, o texto sai como foi
// escrito; onde nao aguenta, ele perde o acento -- que e degradacao legivel, e
// nao lixo. Ver modoUnicode.
//
// Tirar acento e text.RemoveAccents, a mesma conta do indice e da chave de
// cofre, e nao uma tabela local: duas tabelas divergem, e a menos consultada e
// a que fica errada. E a unica aresta deste pacote, e ela esta justificada no
// grafo do CLAUDE.md.
//
// Idempotente de proposito: a moldura adapta as linhas antes de MEDIR a
// largura, e quem imprime adapta de novo na hora de escrever. Adaptar duas
// vezes o mesmo texto devolve o mesmo texto.
func adaptarTexto(s string) string {
	if modoUnicode() {
		return s
	}
	return text.RemoveAccents(s)
}
