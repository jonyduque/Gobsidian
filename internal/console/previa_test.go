package console

import (
	"bytes"
	"strings"
	"testing"
)

// previa desenha a conversa inteira num buffer, para o teste poder MOSTRAR sem
// poluir a saida de `go test ./...`.
//
// Escrever direto em os.Stdout imprimiria a moldura em toda rodada da bateria,
// inclusive no verify -- e um teste que existe para ser lido nao pode atrapalhar
// os que existem para reprovar. Com t.Log a previa so aparece com -v.
//
// A cor e forcada com NewPlain: o buffer nao e terminal, e sem isso a previa
// sairia sem nenhum realce justamente onde se quer conferir o realce. O que ela
// mostra e a ESTRUTURA; as cores estao em console.go, nomeadas por papel.
func previa(t *testing.T, g Glifos) string {
	t.Helper()
	forcarGlifos = &g
	defer func() { forcarGlifos = nil }()

	var buf bytes.Buffer
	s := NewPlain(&buf)

	s.Titulo("Configuracao atual")
	s.Bloco("", []string{`  C:\Users\jonyd\Documents\Estudo`}, "")
	s.Pergunta("Manter esta configuracao?", "S/n")
	s.Line("")
	s.Resposta("nao")

	opcoes := []Opcao{
		{Rotulo: `C:\Users\jonyd\Documents\Estudo`, Nota: "(ja configurado)", Marcada: true},
		{Rotulo: `C:\Users\jonyd\Documents\Trabalho`, Nota: "(aberto agora)"},
		{Rotulo: `D:\Cofres\Ação Direta`},
	}
	desenhar(s, opcoes, []bool{true, false, true}, 1, "Quais cofres configurar?", false)

	s.Titulo("Instalando")
	s.OK("Instalado")
	s.Bloco("", []string{
		`  binario  C:\Users\jonyd\AppData\Local\Programs\gobsidian\gobsidian.exe`,
		`  cofre    C:\Users\jonyd\Documents\Estudo`,
		"  claude-desktop   Reinicie o Claude Desktop para carregar o servidor.",
	}, "")
	return buf.String()
}

// TestPreviaDaConversa nao asserta desenho: ele o mostra, e confere a unica
// coisa que da para afirmar sem congelar a arte -- que nenhuma moldura ficou
// torta. Rode com -v para ver.
func TestPreviaDaConversa(t *testing.T) {
	for _, caso := range []struct {
		nome   string
		glifos Glifos
	}{
		{"unicode", glifosUnicode},
		{"ascii", glifosASCII},
	} {
		t.Run(caso.nome, func(t *testing.T) {
			saida := previa(t, caso.glifos)
			t.Logf("\n%s", saida)

			// As molduras da previa: toda linha que comeca com uma barra ou um
			// canto pertence a alguma delas, e todas as de UM bloco tem a mesma
			// largura. Conferir bloco a bloco exigiria conhecer o desenho; o
			// que da para afirmar aqui e que nenhuma linha de moldura ficou
			// vazia ou sem fechar.
			for i, l := range strings.Split(saida, "\n") {
				if l == "" {
					continue
				}
				comecaMoldura := strings.HasPrefix(l, caso.glifos.CantoSupEsq) ||
					strings.HasPrefix(l, caso.glifos.Vertical)
				if !comecaMoldura {
					continue
				}
				fim := caso.glifos.CantoSupDir
				if strings.HasPrefix(l, caso.glifos.Vertical) {
					fim = caso.glifos.Vertical
				}
				if !strings.HasSuffix(l, fim) && !strings.HasSuffix(l, caso.glifos.CantoInfDir) {
					t.Errorf("linha %d da moldura nao fecha: %q", i, l)
				}
			}
		})
	}
}
