package search

import (
	"bytes"
	"encoding/binary"
	"errors"
	"math"
	"strings"
	"testing"
)

// TestLimitesCabemEmInt32 é o achado B1 pela metade que o compilador já pega.
//
// `termIni`, `postIni` e `postPath` são `[]int32` e guardam ÍNDICES dentro das
// fatias que estes tetos dimensionam. Um teto acima de `math.MaxInt32` não
// produz erro: produz `int32(kPos)` estourando em SILÊNCIO, com o índice
// apontando para outro trecho do arquivo — o cache decodifica "com sucesso" e
// serve as posições de outro termo.
//
// A guarda de verdade é a de compilação em persist_codec.go, porque teste que
// ninguém rodou não impede o commit. Este teste existe para NOMEAR a regra: quem
// vier atrás e ler só os testes precisa encontrar a razão escrita.
func TestLimitesCabemEmInt32(t *testing.T) {
	for _, c := range []struct {
		nome  string
		valor int
	}{
		{"limiteCaminhos", limiteCaminhos},
		{"limiteTermos", limiteTermos},
		{"limitePostings", limitePostings},
		{"limitePosicoes", limitePosicoes},
	} {
		if c.valor > math.MaxInt32 {
			t.Errorf("%s = %d, acima de math.MaxInt32 (%d): os indices sao int32 e vao estourar em silencio",
				c.nome, c.valor, math.MaxInt32)
		}
	}
}

// TestLimitePosicoesNaoPedeMemoriaAbsurda fixa a outra metade do B1: o teto
// dimensiona um `make` que acontece ANTES de qualquer verificação do corpo.
//
// Com o teto antigo de 4 bilhões, `make([]TokenPosition, totPos)` — 16 bytes por
// posição — pedia 64 GB a partir de um cabeçalho adulterado, e o processo morria
// por OOM em vez de devolver "cache corrompido" e reconstruir. O cofre de
// referência tem 17,8 milhões de posições.
func TestLimitePosicoesNaoPedeMemoriaAbsurda(t *testing.T) {
	const bytesPorPosicao = 16 // TokenPosition{Start, End int64}
	const tetoDeAlocacao = 4 << 30

	if pedido := limitePosicoes * bytesPorPosicao; pedido > tetoDeAlocacao {
		t.Errorf("limitePosicoes = %d pede %d bytes (%.1f GB) num unico make, acima do teto de %.0f GB\n"+
			"esse make roda a partir do CABECALHO, antes de o corpo ser conferido",
			limitePosicoes, pedido, float64(pedido)/(1<<30), float64(tetoDeAlocacao)/(1<<30))
	}
}

// TestCodecRecusaTotalDePosicoesAbsurdo exercita o caminho de verdade: um
// cabeçalho adulterado tem de ser RECUSADO, e não alocado.
//
// Sem o teto certo este teste não falha — ele trava a máquina, que é exatamente
// o modo de falha que o achado descreve.
func TestCodecRecusaTotalDePosicoesAbsurdo(t *testing.T) {
	var buf bytes.Buffer
	termos := map[string]map[string][]TokenPosition{
		"aaa": {"n.md": {{Start: 0, End: 1}}},
	}
	if err := escreveCache(&buf, CacheHeader{FormatVersion: CacheFormatVersion, NoteCount: 1}, termos, map[string]int{"n.md": 1}); err != nil {
		t.Fatalf("escreveCache: %v", err)
	}
	b := buf.Bytes()

	// totPos é o varint imediatamente antes de totPost... na ordem do formato
	// (ver o mapa no topo de persist_codec.go) totPost vem antes de totPos, e
	// os dois vêm antes da tabela de caminhos. Ambos valem 1 neste cache
	// minúsculo, então cada um ocupa um byte só: localizar por valor seria
	// ambíguo. Reescrevemos o arquivo inteiro trocando o segundo "1" varint
	// depois do cabeçalho por um varint gigante.
	orig := append([]byte(nil), b...)
	pos := indiceDoTotalDePosicoes(t, orig)

	var grande [binary.MaxVarintLen64]byte
	n := binary.PutUvarint(grande[:], 3_000_000_000) // acima de MaxInt32, abaixo do teto antigo
	adulterado := make([]byte, 0, len(orig)+n)
	adulterado = append(adulterado, orig[:pos]...)
	adulterado = append(adulterado, grande[:n]...)
	adulterado = append(adulterado, orig[pos+1:]...)

	_, _, err := leCache(adulterado)
	if err == nil {
		t.Fatal("cabecalho declarando 3 bilhoes de posicoes foi ACEITO")
	}
	if !errors.Is(err, ErrCacheCorrupted) {
		t.Fatalf("err = %v, quer ErrCacheCorrupted", err)
	}
	// A mensagem tem de nomear o campo adulterado. Sem isto, uma mudança de
	// layout faria indiceDoTotalDePosicoes apontar para OUTRO varint, o cache
	// seria recusado por outro motivo, e o teste continuaria verde — cobrindo
	// uma regra que ele deixou de exercitar.
	if !strings.Contains(err.Error(), "totalPosicoes") {
		t.Errorf("err = %q, nao nomeia totalPosicoes: o teste provavelmente adulterou outro campo", err)
	}
}

// indiceDoTotalDePosicoes acha o byte de totPos varrendo o cabeçalho do jeito
// que o decodificador o lê. Duplicar o layout num offset fixo faria o teste
// mentir silenciosamente na próxima mudança de formato.
func indiceDoTotalDePosicoes(t *testing.T, b []byte) int {
	t.Helper()
	// leCache lê: magic, versao, ... e depois totCaminhos, totTermos, totPost,
	// totPos. Em vez de recontar isso aqui, achamos o "aaa" e voltamos: entre o
	// fim do cabeçalho e o termo estão a tabela de caminhos e nTermos.
	i := bytes.Index(b, []byte("n.md"))
	if i < 6 {
		t.Fatalf("caminho nao encontrado no cache (i=%d)", i)
	}
	// Os quatro totais são varints de 1 byte neste cache mínimo, imediatamente
	// antes da tabela de caminhos, que começa com o comprimento de "n.md" (4).
	// Andamos para trás até o byte que vale 4 (o comprimento) e pegamos o
	// quarto byte antes dele.
	compr := i - 1
	if b[compr] != byte(len("n.md")) {
		t.Fatalf("layout inesperado: b[%d] = %d, queria %d", compr, b[compr], len("n.md"))
	}
	// nCaminhos, totTermos, totPost, totPos ficam antes; totPos é o último.
	tot := compr - 2
	if tot < 0 {
		t.Fatalf("offset negativo (%d)", tot)
	}
	return tot
}
