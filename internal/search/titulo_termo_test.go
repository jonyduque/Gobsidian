package search

import "testing"

// TestTituloContemTermo cobre a regra do achado P2 isolada.
//
// Os titulos vao NORMALIZADOS porque e o TitleNorm que a funcao recebe — o
// indice ja o calcula e guarda, e reaproveita-lo e o que torna a checagem
// barata o bastante para rodar por documento candidato.
//
// Ela precisa de um teste de UNIDADE, e não só do de integração, porque o
// frontmatter é tokenizado junto com o corpo: um título "Ar puro" injeta uma
// ocorrência de "ar" nos tokens do documento, e a nota pontua mais mesmo se o
// peso de campo tiver sido apagado. O teste de integração passou a cobrir só a
// metade que ele consegue isolar — pedaço de palavra NÃO vale título —, e esta
// aqui cobre a regra inteira.
func TestTituloContemTermo(t *testing.T) {
	casos := []struct {
		nome   string
		titulo string
		termo  string
		quer   bool
		porque string
	}{
		{
			nome: "termo inteiro", titulo: "ar puro", termo: "ar", quer: true,
			porque: "o termo E um token do titulo",
		},
		{
			nome: "pedaco de palavra", titulo: "barragem", termo: "ar", quer: false,
			porque: "o defeito original: substring dava peso de TITULO a \"Barragem\" na busca \"ar\"",
		},
		{
			// Sondado em 2026-08-28: Analyze("processos") da Raw="processos",
			// Reduced="processo". A reducao e o SINGULAR, nao um radical — o
			// analisador nunca produz "process". CalculateBM25 pontua os dois
			// termos, entao a consulta "processos" chega aqui tambem como
			// "processo", e e essa forma que casa o titulo.
			nome: "forma reduzida da consulta", titulo: "processo civil", termo: "processo", quer: true,
			porque: "a busca por \"processos\" reduz para \"processo\", e casar o titulo por ela e desejado",
		},
		{
			// LIMITE CONHECIDO, e medido. Titulo no plural com consulta no
			// singular nao casa: "processo" nao termina em fronteira dentro de
			// "processos". Uma versao anterior cobria este caso tokenizando o
			// titulo com Analyze — estava certa e custou +38% na busca
			// (BenchmarkSearchLimit200Cache, 20,3 -> 27,9 ms, faixas
			// disjuntas), porque Analyze roda por documento candidato por
			// termo. O caminho INVERSO, que e o comum, continua casando: ver o
			// caso acima.
			nome: "titulo plural com consulta singular NAO casa", titulo: "processos civis", termo: "processo", quer: false,
			porque: "limite aceito da varredura por fronteira; cobri-lo exigiria tokens do titulo no indice, e index nao pode importar search",
		},
		{
			nome: "radical inventado nao casa", titulo: "processo civil", termo: "process", quer: false,
			porque: "\"process\" e pedaco de \"processo\"; o analisador nunca produz esse termo",
		},
		{
			nome: "prefixo nao basta", titulo: "processo civil", termo: "proc", quer: false,
			porque: "prefixo tambem e pedaco de palavra",
		},
		{
			nome: "sufixo nao basta", titulo: "barragem", termo: "gem", quer: false,
			porque: "sufixo tambem e pedaco de palavra",
		},
		{
			nome: "segundo token", titulo: "processo civil", termo: "civil", quer: true,
			porque: "a regra vale para qualquer token, nao so o primeiro",
		},
		{
			nome: "titulo vazio", titulo: "", termo: "ar", quer: false,
			porque: "nota sem titulo nao pode casar nada",
		},
		{
			nome: "termo vazio", titulo: "ar puro", termo: "", quer: false,
			porque: "termo vazio casaria tudo",
		},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			if got := tituloContemTermo(c.titulo, c.termo); got != c.quer {
				t.Errorf("tituloContemTermo(%q, %q) = %v, queria %v\n%s",
					c.titulo, c.termo, got, c.quer, c.porque)
			}
		})
	}
}
