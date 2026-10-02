package console

import (
	"bytes"
	"strings"
	"testing"
)

// semBordas tira a moldura de uma linha ASCII e devolve o conteudo interno.
func semBordas(l string) string {
	l = strings.TrimSuffix(strings.TrimPrefix(l, "|"), "|")
	return l
}

// TestMolduraQuebraLinhaLongaSemPerderTexto: linha mais larga que a moldura e
// QUEBRADA, e nao cortada. Medido em 2026-10-01 no `graph` do dono: os caminhos
// das notas saiam com "…" no lugar do fim -- e o fim do caminho e o nome da
// nota, que era justamente o que se queria ler.
func TestMolduraQuebraLinhaLongaSemPerderTexto(t *testing.T) {
	forcarGlifos = &glifosASCII
	defer func() { forcarGlifos = nil }()

	longo := "  Pontos 192/Ponto 02/Direito Processual Penal e Legislação Especial/Juizados Especiais Criminais (Lei nº 9.099 - 95).md  1"
	var buf bytes.Buffer
	NewPlain(&buf).Bloco("41 notas", []string{longo, "  curta.md  1"}, "")

	linhas := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if strings.Contains(buf.String(), GlifosDaSaida().Reticencia) {
		t.Errorf("a linha foi cortada com reticência:\n%s", buf.String())
	}
	largura := larguraVisivel(linhas[0])
	var texto []string
	for i, l := range linhas {
		if larguraVisivel(l) != largura {
			t.Errorf("linha %d com %d colunas, a borda tem %d:\n%s", i, larguraVisivel(l), largura, buf.String())
		}
		if i == 0 || i == len(linhas)-1 {
			continue
		}
		texto = append(texto, strings.TrimSpace(semBordas(l)))
	}
	junto := strings.Join(texto, " ")
	for _, parte := range strings.Fields(longo) {
		if !strings.Contains(junto, parte) {
			t.Errorf("a parte %q da linha longa sumiu:\n%s", parte, buf.String())
		}
	}
	if len(linhas) < 5 {
		t.Errorf("esperava a linha longa quebrada em mais de uma (bordas + 2 + curta), tenho %d linhas:\n%s", len(linhas), buf.String())
	}
}

// TestQuebrarRepoeACorNaLinhaSeguinte: quebrar no meio de um trecho colorido
// fecha a cor no fim da linha e a reabre na seguinte -- senao a cor vaza para a
// borda, ou some do resto do trecho.
func TestQuebrarRepoeACorNaLinhaSeguinte(t *testing.T) {
	dim := "\x1b[2m"
	reset := "\x1b[0m"
	c := "  " + dim + strings.Repeat("palavra ", 12) + reset
	pedacos := quebrar(c, 30)
	if len(pedacos) < 2 {
		t.Fatalf("esperava quebra, tenho %q", pedacos)
	}
	for i, p := range pedacos {
		if larguraVisivel(p) > 30 {
			t.Errorf("pedaço %d com %d colunas, teto 30: %q", i, larguraVisivel(p), p)
		}
		if i < len(pedacos)-1 && !strings.HasSuffix(p, reset) {
			t.Errorf("pedaço %d não fecha a cor: %q", i, p)
		}
		if i > 0 && !strings.Contains(p, dim) {
			t.Errorf("pedaço %d não reabre a cor: %q", i, p)
		}
	}
}

// TestMolduraNaoCortaTituloLongo: o titulo do `graph` traz o caminho da nota de
// origem, e ele saia cortado na borda ("... Especial - V"). O que nao cabe na
// borda desce para as primeiras linhas do corpo.
func TestMolduraNaoCortaTituloLongo(t *testing.T) {
	forcarGlifos = &glifosASCII
	defer func() { forcarGlifos = nil }()

	titulo := "39 notas e 284 links a partir de Resumos/PENAL/Legislação Penal Especial - Victor Eduardo Rios Gonçalves.md"
	var buf bytes.Buffer
	NewPlain(&buf).Bloco(titulo, []string{"  a.md  1"}, "")

	saida := buf.String()
	if strings.Contains(saida, GlifosDaSaida().Reticencia) {
		t.Errorf("o título foi cortado:\n%s", saida)
	}
	plano := strings.Join(strings.Fields(strings.NewReplacer("|", " ", "+", " ", "-", " ").Replace(adaptarTexto(saida))), " ")
	for _, parte := range strings.Fields(adaptarTexto(titulo)) {
		if strings.Trim(parte, "-") == "" {
			continue
		}
		if !strings.Contains(plano, strings.ReplaceAll(parte, "-", " ")) {
			t.Errorf("a parte %q do título sumiu:\n%s", parte, saida)
		}
	}
	linhas := strings.Split(strings.TrimRight(saida, "\n"), "\n")
	for i, l := range linhas {
		if larguraVisivel(l) != larguraVisivel(linhas[0]) {
			t.Errorf("linha %d com %d colunas, a borda tem %d:\n%s", i, larguraVisivel(l), larguraVisivel(linhas[0]), saida)
		}
	}
}
