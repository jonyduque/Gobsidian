// Package search implementa a análise de texto, indexação invertida, pontuação BM25 e busca do cofre.
package search

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/jonyd/gobsidian/internal/text"
)

// Token representa uma ocorrência de termo indexável em um texto.
// Start e End são offsets em BYTES no texto original.
type Token struct {
	Raw     string // Forma crua normalizada: sem acento, em minúsculas
	Reduced string // Forma reduzida (stemming conservador); vazio quando idêntica a Raw ou não aplicável
	Start   int64  // Offset em bytes do início da ocorrência no texto original
	End     int64  // Offset em bytes do fim da ocorrência no texto original
}

// Normalize remove acentos e converte para caixa baixa.
// Delega para text.Normalize que reutiliza transformers para eficiência.
func Normalize(s string) string {
	return text.Normalize(s)
}

// Reduce aplica regras conservadoras de redução para português (plurais e sufixos verbais/nominais).
// Devolve string vazia se a forma reduzida for idêntica à forma crua normalizada.
// Regras aplicadas (8 regras no total):
// 1. -coes / -soes / -zoes -> -cao / -sao / -zao
// 2. -oes -> -ao
// 3. -ais -> -al
// 4. -eis -> -el
// 5. -ois -> -ol
// 6. -uis -> -ul
// 7. -res -> -r
// 8. -s (plural simples de palavras com mais de 3 caracteres, sem terminar em -ss ou -is)
func Reduce(raw string) string {
	if len(raw) <= 3 {
		return ""
	}

	var reduced string
	switch {
	case strings.HasSuffix(raw, "coes"):
		reduced = raw[:len(raw)-4] + "cao"
	case strings.HasSuffix(raw, "soes"):
		reduced = raw[:len(raw)-4] + "sao"
	case strings.HasSuffix(raw, "zoes"):
		reduced = raw[:len(raw)-4] + "zao"
	case strings.HasSuffix(raw, "oes"):
		reduced = raw[:len(raw)-3] + "ao"
	case strings.HasSuffix(raw, "ais"):
		reduced = raw[:len(raw)-3] + "al"
	case strings.HasSuffix(raw, "eis"):
		reduced = raw[:len(raw)-3] + "el"
	case strings.HasSuffix(raw, "ois"):
		reduced = raw[:len(raw)-3] + "ol"
	case strings.HasSuffix(raw, "uis"):
		reduced = raw[:len(raw)-3] + "ul"
	case strings.HasSuffix(raw, "res"):
		reduced = raw[:len(raw)-2]
	case strings.HasSuffix(raw, "s") && !strings.HasSuffix(raw, "ss") && !strings.HasSuffix(raw, "is"):
		reduced = raw[:len(raw)-1]
	default:
		reduced = raw
	}

	if reduced == raw {
		return ""
	}
	return reduced
}

// Analyze tokeniza o texto de entrada extraindo sequências alfanuméricas,
// aplicando normalização e redução conservadora com offsets exatos em bytes.
func Analyze(text string) []Token {
	var tokens []Token
	start := -1
	byteOffset := 0

	emit := func(s, e int) {
		rawNorm := Normalize(text[s:e])
		if len(rawNorm) > 0 {
			red := Reduce(rawNorm)
			tokens = append(tokens, Token{
				Raw:     rawNorm,
				Reduced: red,
				Start:   int64(s),
				End:     int64(e),
			})
		}
	}

	for len(text[byteOffset:]) > 0 {
		r, size := utf8.DecodeRuneInString(text[byteOffset:])
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if start < 0 {
				start = byteOffset
			}
		} else {
			if start >= 0 {
				emit(start, byteOffset)
				start = -1
			}
		}
		byteOffset += size
	}

	if start >= 0 {
		emit(start, byteOffset)
	}

	return tokens
}
