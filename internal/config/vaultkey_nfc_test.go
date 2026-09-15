package config_test

import (
	"testing"

	"github.com/jonyduque/Gobsidian/internal/config"
)

// TestVaultKeyIgualEmNFCeNFD e H3.
//
// Os caminhos sao montados por code point, e nao escritos no fonte, para que
// nenhum editor normalize a grafia sem avisar e o teste passe a comparar a
// mesma string com ela mesma.
func TestVaultKeyIgualEmNFCeNFD(t *testing.T) {
	nfc := "C:/Users/jonyd/Obsidian/Revis" + string(rune(0x00E3)) + "o"
	nfd := "C:/Users/jonyd/Obsidian/Revisa" + string(rune(0x0303)) + "o"
	if nfc == nfd {
		t.Fatal("as duas grafias sairam iguais: o teste nao compara nada")
	}

	// A chave NFC foi medida em 2026-09-14, antes da mudanca. Caminho ja em NFC
	// nao pode mudar de chave: o cache e o socket de quem ja roda dependem dela.
	const querNFC = "eda87fbb16003550"
	if got := config.VaultKey(nfc); got != querNFC {
		t.Fatalf("VaultKey(NFC) = %s, esperado %s -- caminho em NFC mudou de chave", got, querNFC)
	}
	if got := config.VaultKey(nfd); got != querNFC {
		t.Fatalf("VaultKey(NFD) = %s, esperado %s -- a mesma pasta com duas chaves", got, querNFC)
	}
}
