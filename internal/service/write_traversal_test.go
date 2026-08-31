package service

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// travessiaUniversal sao os caminhos que escapam do cofre em QUALQUER sistema.
var travessiaUniversal = []string{
	`../../x.md`,
	`/etc/passwd`,
	"nota\x00.md",
}

// travessiaSoNoWindows sao os caminhos que escapam SO no Windows.
//
// Em Linux e macOS a barra invertida e caractere legal de NOME de arquivo, nao
// separador: `..\x.md` e um unico componente chamado literalmente `..\x.md`, e
// criar isso dentro do cofre e comportamento correto. `COM1` e nome de
// dispositivo so no Windows; em Linux e um nome comum.
var travessiaSoNoWindows = []string{
	`..\..\x.md`,
	`..\x.md`,
	`sub\..\..\x.md`,
	`C:\Windows\Temp\x.md`,
	"COM1",
}

// TestEscritaRecusaTravessiaComSeparadorDoWindows cobre a forma com barra
// invertida, que e a que escapa.
//
// A versao anterior rodava a lista inteira nos tres sistemas, com o comentario
// de que ficava sem `runtime.GOOS` de proposito porque "a asserção 'recusou'
// vale nos dois, e no Linux ela e trivialmente verdadeira". Era o contrario: no
// Linux ela e trivialmente FALSA para os casos de barra invertida, porque ali
// nao ha travessia nenhuma a recusar. O teste passava no Windows e reprovava em
// ubuntu e macos — e so apareceu quando o ramo foi empurrado pela primeira vez,
// porque o CI e o unico Linux deste projeto.
//
// A preocupacao original — "um teste que so roda no Windows e um teste que
// ninguem ve reprovar" — continua legitima e esta respondida de outro jeito: o
// caso nao e PULADO fora do Windows, ele troca de asserção. Onde o nome e legal,
// o produto tem de ACEITA-LO e gravar DENTRO do cofre. Recusar ali seria tornar
// inalcancavel uma nota de nome legitimo, que e a mesma razao pela qual
// `validatePlatformPath` so vale no Windows.
func TestEscritaRecusaTravessiaComSeparadorDoWindows(t *testing.T) {
	for _, c := range travessiaUniversal {
		t.Run("universal/"+c, func(t *testing.T) {
			exigeRecusa(t, c)
		})
	}

	for _, c := range travessiaSoNoWindows {
		t.Run("windows/"+c, func(t *testing.T) {
			if runtime.GOOS == "windows" {
				exigeRecusa(t, c)
				return
			}
			exigeAceitacaoDentroDoCofre(t, c)
		})
	}
}

// exigeRecusa prova que o caminho e recusado E que nada foi escrito fora do
// cofre. A segunda metade importa: um erro devolvido DEPOIS da escrita passaria
// pela primeira.
func exigeRecusa(t *testing.T, caminho string) {
	t.Helper()
	cofreRoot := t.TempDir()
	writeFile(t, cofreRoot, "existente.md", "conteudo")
	svc := newTestService(t, cofreRoot)

	if _, err := svc.CreateNote(context.Background(), CreateNoteRequest{Path: caminho, Content: "x"}); err == nil {
		t.Fatalf("CreateNote(%q) devolveu sucesso", caminho)
	}
	exigeNadaForaDoCofre(t, cofreRoot, caminho)

	// E o mesmo pelo destino de note_move, que constroi o caminho por outra via.
	if _, err := svc.MoveNote(context.Background(), MoveNoteRequest{From: "existente.md", To: caminho}); err == nil {
		t.Fatalf("MoveNote(to=%q) devolveu sucesso", caminho)
	}
	exigeNadaForaDoCofre(t, cofreRoot, caminho)
}

// exigeAceitacaoDentroDoCofre e o contrapeso: onde o nome e legal, recusar e o
// defeito. Sem esta metade, uma implementacao que recusasse barra invertida em
// todo sistema passaria no teste e tornaria inalcancaveis notas de nome legitimo
// num cofre Linux.
func exigeAceitacaoDentroDoCofre(t *testing.T, caminho string) {
	t.Helper()
	cofreRoot := t.TempDir()
	svc := newTestService(t, cofreRoot)

	if _, err := svc.CreateNote(context.Background(), CreateNoteRequest{Path: caminho, Content: "x"}); err != nil {
		t.Fatalf("CreateNote(%q) recusou um nome de arquivo LEGAL em %s: %v",
			caminho, runtime.GOOS, err)
	}
	if _, err := os.Stat(filepath.Join(cofreRoot, caminho)); err != nil {
		t.Fatalf("CreateNote(%q) devolveu sucesso e o arquivo nao esta dentro do cofre: %v",
			caminho, err)
	}
	exigeNadaForaDoCofre(t, cofreRoot, caminho)
}

// exigeNadaForaDoCofre confere que nenhum `.md` apareceu no diretorio ACIMA da
// raiz do cofre — o alvo de toda travessia com `..`.
func exigeNadaForaDoCofre(t *testing.T, cofreRoot, caminho string) {
	t.Helper()
	entradas, err := os.ReadDir(filepath.Dir(cofreRoot))
	if err != nil {
		t.Fatalf("lendo o diretorio acima do cofre: %v", err)
	}
	for _, e := range entradas {
		if strings.HasSuffix(e.Name(), ".md") {
			t.Fatalf("caso %q gravou %q fora do cofre", caminho, e.Name())
		}
	}
}
