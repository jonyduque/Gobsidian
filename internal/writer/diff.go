package writer

import (
	"fmt"
	"strings"
)

// EditOp representa a operacao de edicao em um diff.
type EditOp int

const (
	// OpEqual indica linha sem alteracao.
	OpEqual EditOp = iota
	// OpInsert indica linha inserida.
	OpInsert
	// OpDelete indica linha removida.
	OpDelete
)

// Edit representa uma linha anotada no diff.
type Edit struct {
	Op   EditOp
	Text string
}

// splitLines divide o texto em linhas normalizando EOL (\r\n e \n).
// Remove os caracteres \r finais de modo que arquivos CRLF e LF sejam comparados uniformemente.
func splitLines(text string) []string {
	if text == "" {
		return nil
	}

	// Normaliza EOL: converte \r\n para \n antes de splitting
	normalized := strings.ReplaceAll(text, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")

	lines := strings.Split(normalized, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" && strings.HasSuffix(normalized, "\n") {
		lines = lines[:len(lines)-1]
	}
	return lines
}

// DiffLines calcula a lista de operacoes de edicao (Equal, Insert, Delete) usando o algoritmo de Myers.
func DiffLines(aText, bText string) []Edit {
	a := splitLines(aText)
	b := splitLines(bText)
	return myersDiff(a, b)
}

func myersDiff(a, b []string) []Edit {
	N := len(a)
	M := len(b)

	if N == 0 && M == 0 {
		return nil
	}
	if N == 0 {
		edits := make([]Edit, M)
		for i, line := range b {
			edits[i] = Edit{Op: OpInsert, Text: line}
		}
		return edits
	}
	if M == 0 {
		edits := make([]Edit, N)
		for i, line := range a {
			edits[i] = Edit{Op: OpDelete, Text: line}
		}
		return edits
	}

	maxLimit := N + M
	v := make(map[int]int)
	v[1] = 0

	trace := make([]map[int]int, 0, maxLimit+1)

	for D := 0; D <= maxLimit; D++ {
		vCopy := make(map[int]int, len(v))
		for k, val := range v {
			vCopy[k] = val
		}
		trace = append(trace, vCopy)

		for k := -D; k <= D; k += 2 {
			var x int
			if k == -D || (k != D && v[k-1] < v[k+1]) {
				x = v[k+1]
			} else {
				x = v[k-1] + 1
			}
			y := x - k

			for x < N && y < M && a[x] == b[y] {
				x++
				y++
			}
			v[k] = x

			if x >= N && y >= M {
				return backtrackTrace(a, b, trace, D)
			}
		}
	}
	return nil
}

func backtrackTrace(a, b []string, trace []map[int]int, maxD int) []Edit {
	x := len(a)
	y := len(b)

	var reverse []Edit

	for d := maxD; d > 0; d-- {
		v := trace[d]
		k := x - y

		var prevK int
		if k == -d || (k != d && v[k-1] < v[k+1]) {
			prevK = k + 1
		} else {
			prevK = k - 1
		}

		prevX := v[prevK]
		prevY := prevX - prevK

		for x > prevX && y > prevY {
			x--
			y--
			reverse = append(reverse, Edit{Op: OpEqual, Text: a[x]})
		}

		if x == prevX {
			y--
			reverse = append(reverse, Edit{Op: OpInsert, Text: b[y]})
		} else {
			x--
			reverse = append(reverse, Edit{Op: OpDelete, Text: a[x]})
		}
	}

	for x > 0 && y > 0 {
		x--
		y--
		reverse = append(reverse, Edit{Op: OpEqual, Text: a[x]})
	}

	edits := make([]Edit, len(reverse))
	for i, e := range reverse {
		edits[len(reverse)-1-i] = e
	}
	return edits
}

type hunk struct {
	aStart, aLen int
	bStart, bLen int
	lines        []string
}

// UnifiedDiff gera a representacao de diff unificado (estilo patch / git diff) entre dois textos.
// Se os textos forem identicos (ou equivalentes sob normalizacao EOL), retorna string vazia.
func UnifiedDiff(aName, bName, aText, bText string, contextLines int) string {
	if aText == bText {
		return ""
	}

	aLines := splitLines(aText)
	bLines := splitLines(bText)
	if len(aLines) == len(bLines) {
		equal := true
		for i := range aLines {
			if aLines[i] != bLines[i] {
				equal = false
				break
			}
		}
		if equal {
			return ""
		}
	}

	edits := DiffLines(aText, bText)
	if len(edits) == 0 {
		return ""
	}

	hunks := buildHunks(edits, contextLines)
	if len(hunks) == 0 {
		return ""
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "--- %s\n", aName)
	fmt.Fprintf(&sb, "+++ %s\n", bName)

	for _, h := range hunks {
		// Comprimento ZERO exige inicio ZERO — nao o numero da linha seguinte.
		//
		// O formato unified define, para um lado de comprimento 0, que o inicio
		// e a linha ANTES do ponto de insercao. Numa insercao em arquivo vazio
		// isso da "@@ -0,0 +1,1 @@"; ate 2026-08-28 saia "@@ -1,0 +1,1 @@"
		// (achado B8), que o GNU patch recusa como cabecalho invalido. O diff
		// era legivel na tela e nao aplicavel — e dry_run existe justamente para
		// o cliente poder aplicar ou revisar o que viu.
		fmt.Fprintf(&sb, "@@ -%d,%d +%d,%d @@\n",
			inicioDeHunk(h.aStart, h.aLen), h.aLen,
			inicioDeHunk(h.bStart, h.bLen), h.bLen)
		for _, line := range h.lines {
			sb.WriteString(line)
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

func buildHunks(edits []Edit, contextLines int) []hunk {
	if len(edits) == 0 {
		return nil
	}

	type annotatedEdit struct {
		edit Edit
		aIdx int
		bIdx int
	}

	var annotated []annotatedEdit
	aCur, bCur := 1, 1
	for _, e := range edits {
		ann := annotatedEdit{edit: e, aIdx: aCur, bIdx: bCur}
		annotated = append(annotated, ann)
		switch e.Op {
		case OpEqual:
			aCur++
			bCur++
		case OpDelete:
			aCur++
		case OpInsert:
			bCur++
		}
	}

	var hunks []hunk
	var currentLines []string
	var hAStart, hBStart int
	var hALen, hBLen int

	inHunk := false

	for i, ann := range annotated {
		isChange := (ann.edit.Op != OpEqual)

		if !isChange {
			hasPrevChange := false
			for j := max(0, i-contextLines); j < i; j++ {
				if annotated[j].edit.Op != OpEqual {
					hasPrevChange = true
					break
				}
			}
			hasNextChange := false
			for j := i + 1; j <= min(len(annotated)-1, i+contextLines); j++ {
				if annotated[j].edit.Op != OpEqual {
					hasNextChange = true
					break
				}
			}

			if !hasPrevChange && !hasNextChange {
				if inHunk {
					hunks = append(hunks, hunk{aStart: hAStart, aLen: hALen, bStart: hBStart, bLen: hBLen, lines: currentLines})
					inHunk = false
					currentLines = nil
				}
				continue
			}
		}

		if !inHunk {
			inHunk = true
			hAStart = ann.aIdx
			hBStart = ann.bIdx
			hALen = 0
			hBLen = 0
			currentLines = nil
		}

		prefix := " "
		switch ann.edit.Op {
		case OpDelete:
			prefix = "-"
			hALen++
		case OpInsert:
			prefix = "+"
			hBLen++
		default:
			hALen++
			hBLen++
		}
		currentLines = append(currentLines, prefix+ann.edit.Text)
	}

	if inHunk {
		hunks = append(hunks, hunk{aStart: hAStart, aLen: hALen, bStart: hBStart, bLen: hBLen, lines: currentLines})
	}

	return hunks
}

// inicioDeHunk devolve o numero de linha que o cabecalho de hunk deve trazer.
//
// Uma conta so para os dois lados do "@@": lado com comprimento zero comeca em
// zero, os demais na propria linha. Duplicar a regra nos dois argumentos do
// Fprintf e como o achado B8 poderia voltar pela metade.
func inicioDeHunk(inicio, comprimento int) int {
	if comprimento == 0 {
		return inicio - 1
	}
	return inicio
}
