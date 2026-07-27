package vault

import "bytes"

// EOLStyle e a convencao de fim de linha de uma nota. O produto preserva a
// que encontrou: reescrever uma nota inteira so porque ela usava CRLF gera
// um diff que o usuario nao pediu e que polui o historico do cofre dele.
type EOLStyle int

// LF e o zero deliberadamente: e o default para nota nova e para arquivo
// onde nao ha CRLF que indique o contrario.
const (
	EOLLF EOLStyle = iota
	EOLCRLF
)

func (e EOLStyle) String() string {
	if e == EOLCRLF {
		return "CRLF"
	}
	return "LF"
}

// Bytes devolve a sequencia de quebra de linha do estilo.
func (e EOLStyle) Bytes() []byte {
	if e == EOLCRLF {
		return []byte("\r\n")
	}
	return []byte("\n")
}

var bom = []byte{0xEF, 0xBB, 0xBF}

// DetectEOL devolve o estilo predominante do arquivo. Arquivos com mistura
// existem, e converter o arquivo inteiro para um estilo so seria reescrever
// o que o usuario nao pediu — a escrita normaliza apenas o conteudo novo.
//
// CR solto (Mac classico, pre-OS X) nao tem representacao em EOLStyle: como
// o tipo so modela LF e CRLF, um arquivo sem nenhum "\r\n" nem "\n" conta
// zero quebras dos dois tipos e cai no default, EOLLF. Isso e consequencia
// direta do desenho de dois valores, nao um bug — e o conteudo antigo em CR
// solto nunca e reescrito, porque NormalizeEOL so processa o texto novo.
func DetectEOL(data []byte) EOLStyle {
	crlf := bytes.Count(data, []byte("\r\n"))
	lf := bytes.Count(data, []byte("\n")) - crlf

	if crlf > lf {
		return EOLCRLF
	}
	return EOLLF
}

// StripBOM remove o marcador UTF-8 do inicio, reportando se ele estava la.
func StripBOM(data []byte) ([]byte, bool) {
	if bytes.HasPrefix(data, bom) {
		return data[len(bom):], true
	}
	return data, false
}

// AddBOM reintroduz o marcador. Usado na escrita, quando o arquivo original
// o tinha.
func AddBOM(data []byte) []byte {
	if bytes.HasPrefix(data, bom) {
		return data
	}
	return append(append([]byte{}, bom...), data...)
}

// NormalizeEOL converte o conteudo novo para o estilo do arquivo alvo.
func NormalizeEOL(data []byte, style EOLStyle) []byte {
	flat := bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
	if style == EOLLF {
		return flat
	}
	return bytes.ReplaceAll(flat, []byte("\n"), []byte("\r\n"))
}
