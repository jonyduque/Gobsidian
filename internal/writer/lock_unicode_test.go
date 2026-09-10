package writer

import (
	"testing"
	"time"

	"github.com/jonyduque/Gobsidian/internal/vault"
)

// As duas grafias do MESMO nome de arquivo. Um cofre sincronizado com macOS
// grava NFD e um cliente Windows manda NFC; sao a mesma nota para quem le e
// strings diferentes para um mapa de Go.
//
// As duas se parecem no editor e e por isso que o teste comeca conferindo que
// elas DIFEREM: um editor que normalize o arquivo ao gravar faria as duas
// virarem a mesma string, e o teste passaria sem cobrir nada.
const (
	acaoNFC vault.CanonicalPath = "Ação.md"   // precompostos
	acaoNFD vault.CanonicalPath = "Ação.md" // c + cedilha, a + til
)

// TestChaveDaTravaUneAsDuasGrafias cobra que a trava de escrita seja UMA para
// as duas grafias do mesmo nome.
//
// Ate a Task 169 normalizeKey so baixava a caixa, enquanto a chave do indice
// (index/chave.go) tambem normalizava para NFC. Duas contas para a mesma
// pergunta: o indice tratava as duas grafias como uma nota, e o locker as
// tratava como dois arquivos — duas escritas concorrentes na MESMA nota
// pegavam travas diferentes e nao se excluiam.
func TestChaveDaTravaUneAsDuasGrafias(t *testing.T) {
	if acaoNFC == acaoNFD {
		t.Fatal("as duas constantes sao a mesma string; o teste nao cobre nada")
	}

	if a, b := normalizeKey(acaoNFC), normalizeKey(acaoNFD); a != b {
		t.Errorf("chaves diferentes para a mesma nota:\n NFC -> %q\n NFD -> %q", a, b)
	}

	// E pela porta publica: com a trava de uma grafia na mao, a outra tem de
	// esperar. Sem isto o teste cobriria so a funcao interna, e a trava e o que
	// o resto do pacote usa.
	l := NewPathLocker()
	destravar := l.Lock(acaoNFC)

	comecou := make(chan struct{})
	pegou := make(chan func(), 1)
	go func() {
		close(comecou)
		pegou <- l.Lock(acaoNFD)
	}()

	// Espera a goroutine estar a ponto de travar, para que uma falha em pegar a
	// trava seja bloqueio de verdade e nao goroutine que ainda nao rodou.
	<-comecou

	select {
	case destravarNFD := <-pegou:
		destravarNFD()
		destravar()
		t.Fatal("a grafia NFD pegou a trava enquanto a NFC a segurava; sao a mesma nota")
	case <-time.After(200 * time.Millisecond):
	}

	destravar()

	select {
	case destravarNFD := <-pegou:
		destravarNFD()
	case <-time.After(5 * time.Second):
		t.Fatal("a grafia NFD nunca pegou a trava depois de a NFC solta-la")
	}

	if n := l.Count(); n != 0 {
		t.Errorf("o registro ficou com %d travas depois de tudo solto; esperado 0", n)
	}
}
