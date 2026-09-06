package boot

import "io"

// mirrorDst e o subconjunto de *io.PipeWriter que mirrorReader usa. Extrair
// para interface nao muda nada em producao — pw continua sendo um
// *io.PipeWriter de verdade — mas deixa o teste trocar o espelho por um
// dublê que conta tentativas de escrita. Sem isso nao ha como observar, de
// fora, que a guarda !m.broken impediu uma segunda escrita: o retorno de
// Read e o mesmo com ou sem a guarda, so o numero de chamadas ao espelho
// difere.
type mirrorDst interface {
	io.Writer
	CloseWithError(error) error
}

// mirrorReader espelha o que le para dst e, crucialmente, propaga o fim da
// leitura fechando dst. E o que faz o EOF do host chegar ao monitor de stdin
// do lifecycle — io.TeeReader nao serve aqui porque so copia bytes, e EOF nao
// e um byte.
type mirrorReader struct {
	src    io.Reader
	dst    mirrorDst
	broken bool // espelho desistiu; a leitura principal segue intacta
}

func (m *mirrorReader) Read(p []byte) (int, error) {
	n, err := m.src.Read(p)

	// O espelho e auxiliar: existe so para o lifecycle enxergar o EOF. Se a
	// escrita nele falhar, o JSON-RPC continua — devolver o erro da escrita
	// no lugar do resultado da leitura injetaria uma falha inventada em uma
	// sessao saudavel, e o cliente veria a conexao morrer sem motivo.
	if n > 0 && !m.broken {
		if _, werr := m.dst.Write(p[:n]); werr != nil {
			m.broken = true
		}
	}

	if err != nil {
		_ = m.dst.CloseWithError(err)
	}
	return n, err
}
