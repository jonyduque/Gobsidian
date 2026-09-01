package parser

import (
	"bytes"
	"strconv"
	"strings"
)

// Candidate e uma estrutura APARENTE de uma nota: algo que se parece com um
// titulo sem ser um heading Markdown.
//
// Existe porque parseATXHeading so aceita ATX (`#`), e nota convertida de PDF,
// DOCX ou EPUB nao tem nenhum: ela marca titulo com paragrafo em negrito —
// `**13.1.10 Substituicao de candidatos**` — ou com setext. Num cofre de estudo
// isso e a maioria das notas, e nelas note_read por heading, note_patch por
// secao, ancora de wikilink e o peso de heading do BM25 nao funcionam.
//
// Candidato NUNCA vira Heading, nem entra no indice, nem no cache. Uma tool que
// afirma estrutura que o arquivo nao tem e pior que uma que nao responde: o
// cliente age sobre a afirmacao. Por isso a resposta separa os dois e diz qual
// e qual.
type Candidate struct {
	// Kind e "strong_paragraph" ou "setext".
	Kind string `json:"kind"`
	Text string `json:"text"`
	// Level e PONTEIRO porque ausente e diferente de zero: candidato sem
	// numeracao hierarquica nao tem nivel que se possa afirmar, e um zero
	// literal ali mentiria dizendo "nivel zero".
	Level *int `json:"level,omitempty"`
	// Start e End sao offsets ABSOLUTOS, na mesma coordenada que
	// note_read(offset=) aceita — ou a tool devolve numeros bonitos e inuteis.
	Start int64 `json:"start"`
	End   int64 `json:"end"`
}

// comprimentoMinimoDeTitulo e o piso de CARACTERES para um texto poder ser
// titulo. Runes, e nao bytes: `Nao.` tem 4 caracteres e 5 bytes.
//
// Cinco, e o numero saiu de medicao, nao de gosto. Nos tres cofres reais do dono
// em 2026-09-01, o ruido da deteccao tinha um nome so: `**V:**`, o marcador de
// VERSO de flashcard. Ele fica sozinho na linha -- o par dele, `**F:**`, escapa
// porque a pergunta vem na mesma linha -- e respondia por 14.940 dos 14.940
// candidatos do cofre Revisao e por 14.961 dos 22.180 do TJSP.
//
// O piso de 5 descarta, nos tres cofres: 100% dos candidatos de Revisao, 68% dos
// do TJSP, e 1,6% dos do Estudo -- e os 91 do Estudo sao `Nao.`, `SIM.`, `22`,
// tambem respostas de flashcard. Nenhuma secao legitima medida se perde.
//
// Dois filtros que foram MEDIDOS E RECUSADOS, para nao voltarem:
//
//   - "termina com dois-pontos" descartaria 222 candidatos legitimos do Estudo,
//     como `Jurisprudencia:` e `Constituicao Federal de 1988:`, que sao rotulos
//     de subsecao de verdade.
//   - "repetido em muitas notas" marcaria 28,7% do Estudo, e o que ele marca sao
//     `1. Objetividade juridica` (152x), `3. Sujeito ativo` (127x) -- a estrutura
//     de sete partes que um tratado de direito penal repete a cada crime.
//     Repeticao ali e sinal de livro bem estruturado, nao de ruido.
const comprimentoMinimoDeTitulo = 5

// ehTextoDeTitulo e a conta UNICA do piso, e vale para as duas formas: um setext
// de duas letras nao e mais titulo que um negrito de duas letras.
func ehTextoDeTitulo(t string) bool {
	return len([]rune(strings.TrimSpace(t))) >= comprimentoMinimoDeTitulo
}

// nivelDeCandidatoSemNumero e o nivel de trabalho de um candidato sem numeracao
// no calculo de End.
//
// Seis, e nao zero: um candidato sem numero e tratado como o MAIS PROFUNDO, e
// entao qualquer candidato numerado o fecha, enquanto ele nao fecha nenhuma
// secao numerada. Com zero seria o contrario — "**Introducao**" engoliria
// "**13.1 Algo**" que vem depois, porque 2 <= 0 e falso.
const nivelDeCandidatoSemNumero = 6

// DetectCandidates acha as estruturas aparentes de body, com offsets deslocados
// por bodyOffset (o BOM, quando ha).
//
// A maquina de cercas e a MESMA de ExtractHeadings — openFence e closesFence.
// Duas maquinas de cerca divergem, e a divergencia aparece como hierarquia
// falsa dentro de bloco de codigo, que e o pior lugar para ela aparecer numa
// nota que documenta Markdown.
func DetectCandidates(body []byte, bodyOffset int64) []Candidate {
	var out []Candidate

	inFence := false
	var fence fenceInfo

	// A linha anterior fica guardada porque setext so se reconhece na linha
	// SEGUINTE: o titulo e a linha de cima, e o `===` de baixo e o que diz que
	// ela era um titulo.
	var anteriorTexto string
	var anteriorInicio int64
	anteriorValida := false

	pos := int64(0)
	for len(body) > 0 {
		nl := bytes.IndexByte(body, '\n')
		var line []byte
		var advance int64
		if nl < 0 {
			line, advance = body, int64(len(body))
		} else {
			line, advance = body[:nl], int64(nl)+1
		}

		texto := string(bytes.TrimRight(line, "\r"))
		inicioDaLinha := bodyOffset + pos

		if inFence {
			if closesFence(texto, fence) {
				inFence = false
			}
			anteriorValida = false
			pos += advance
			body = body[advance:]
			continue
		}

		if f, ok := openFence(texto); ok {
			inFence, fence = true, f
			anteriorValida = false
			pos += advance
			body = body[advance:]
			continue
		}

		// Setext: a linha de sublinhado transforma a ANTERIOR em titulo.
		if anteriorValida {
			if nivel, ok := sublinhadoSetext(texto); ok && ehTextoDeTitulo(anteriorTexto) {
				n := nivel
				out = append(out, Candidate{
					Kind:  "setext",
					Text:  strings.TrimSpace(anteriorTexto),
					Level: &n,
					Start: anteriorInicio,
				})
				anteriorValida = false
				pos += advance
				body = body[advance:]
				continue
			}
		}

		if titulo, ok := paragrafoEmNegrito(texto); ok && ehTextoDeTitulo(titulo) {
			c := Candidate{
				Kind:  "strong_paragraph",
				Text:  titulo,
				Start: inicioDaLinha,
			}
			if n, ok := nivelPorNumeracao(titulo); ok {
				c.Level = &n
			}
			out = append(out, c)
			anteriorValida = false
			pos += advance
			body = body[advance:]
			continue
		}

		anteriorTexto = texto
		anteriorInicio = inicioDaLinha
		anteriorValida = strings.TrimSpace(texto) != "" && !ehLinhaDeATX(texto)

		pos += advance
		body = body[advance:]
	}

	fecharCandidatos(out, bodyOffset+pos)
	return out
}

// fecharCandidatos preenche End reusando closeSections, a MESMA regra que
// fecha heading ATX. Reimplementa-la aqui seria a segunda conta da mesma
// pergunta, e as duas divergiriam no primeiro caso de borda.
func fecharCandidatos(cs []Candidate, total int64) {
	if len(cs) == 0 {
		return
	}
	hs := make([]Heading, len(cs))
	for i, c := range cs {
		nivel := nivelDeCandidatoSemNumero
		if c.Level != nil {
			nivel = *c.Level
		}
		hs[i] = Heading{Level: nivel, Start: c.Start}
	}
	closeSections(hs, total)
	for i := range cs {
		cs[i].End = hs[i].End
	}
}

// paragrafoEmNegrito reconhece uma linha que e SO um trecho em negrito —
// `**titulo**` ou `__titulo__`, sozinho na linha.
//
// Sozinho na linha e a regra inteira: negrito no meio de um paragrafo e enfase,
// nao titulo, e aceita-lo encheria a resposta de ruido que o cliente teria de
// filtrar. Tres asteriscos de cada lado (negrito + italico) tambem contam.
func paragrafoEmNegrito(linha string) (string, bool) {
	t := strings.TrimSpace(linha)
	for _, marca := range []string{"***", "**", "__"} {
		if len(t) > 2*len(marca) && strings.HasPrefix(t, marca) && strings.HasSuffix(t, marca) {
			miolo := strings.TrimSpace(t[len(marca) : len(t)-len(marca)])
			// A marca nao pode reaparecer no miolo: `**a** e **b**` sao duas
			// enfases num paragrafo, nao um titulo.
			if miolo == "" || strings.Contains(miolo, marca) {
				return "", false
			}
			return miolo, true
		}
	}
	return "", false
}

// sublinhadoSetext reconhece `===` (nivel 1) ou `---` (nivel 2).
func sublinhadoSetext(linha string) (int, bool) {
	t := strings.TrimSpace(linha)
	if t == "" {
		return 0, false
	}
	c := t[0]
	if c != '=' && c != '-' {
		return 0, false
	}
	for i := 0; i < len(t); i++ {
		if t[i] != c {
			return 0, false
		}
	}
	if c == '=' {
		return 1, true
	}
	return 2, true
}

// ehLinhaDeATX evita que um heading ATX de verdade vire titulo de setext por
// causa de um `---` logo abaixo dele.
func ehLinhaDeATX(linha string) bool {
	_, _, ok := parseATXHeading(linha)
	return ok
}

// nivelPorNumeracao deriva o nivel da PROFUNDIDADE da numeracao: `13` da 1,
// `13.1` da 2, `13.1.10` da 3.
//
// Sem numeracao nao ha nivel a afirmar, e a funcao diz isso devolvendo false em
// vez de um zero que o chamador teria de reinterpretar.
func nivelPorNumeracao(texto string) (int, bool) {
	campo := texto
	if i := strings.IndexAny(campo, " \t"); i >= 0 {
		campo = campo[:i]
	}
	campo = strings.TrimRight(campo, ".)-—:")
	if campo == "" {
		return 0, false
	}
	partes := strings.Split(campo, ".")
	for _, p := range partes {
		if p == "" {
			return 0, false
		}
		if _, err := strconv.Atoi(p); err != nil {
			return 0, false
		}
	}
	return len(partes), true
}
