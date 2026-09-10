package config_test

import (
	"testing"

	"github.com/jonyduque/Gobsidian/internal/config"
)

// TestVaultKeyDobraCaixaInclusiveForaDoASCII e a propriedade que a primeira
// redacao desta mudanca quebrava.
//
// O sistema de arquivos do Windows e insensivel a caixa, e config.Load so
// chama filepath.Abs -- que nao conserta a grafia para a do disco. Duas
// grafias do MESMO cofre precisam dar a MESMA chave, ou viram dois diretorios
// de cache e dois caminhos de socket, e dois processos servindo um cofre so.
//
// Prova de mutacao: trocar caixaEstavel por um rebaixamento de A-Z faz os dois
// ultimos casos reprovarem nomeando as duas chaves.
func TestVaultKeyDobraCaixaInclusiveForaDoASCII(t *testing.T) {
	casos := []struct {
		nome string
		a, b string
	}{
		{"ascii", "C:/Users/x/COFRE", "C:/Users/x/cofre"},
		{"acento maiusculo", "C:/ÁREA/Cofre", "C:/área/Cofre"},
		{"cedilha e til", "C:/AÇÃO/x", "C:/ação/x"},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			if ka, kb := config.VaultKey(c.a), config.VaultKey(c.b); ka != kb {
				t.Errorf("VaultKey(%q) = %s e VaultKey(%q) = %s: o mesmo cofre com duas chaves",
					c.a, ka, c.b, kb)
			}
		})
	}
}

// TestVaultKeyNaoSeMexeuParaCaminhoLatino trava o valor de uma chave concreta.
//
// A troca de strings.ToLower (tabela da stdlib, que a toolchain move) por
// cases.Lower do x/text (tabela de modulo fixado) foi medida em 2026-09-09
// antes de entrar: para caminho latino, INCLUSIVE acentuado, as duas contas
// dao o mesmo hash -- "C:/Users/x/Cofre", "C:/ÁREA/Cofre",
// "C:/Área de Trabalho/Cofre" e "C:/AÇÃO/x" nao mudaram de chave. Diferem so
// onde o x/text aplica casing especial que o ToLower simples nao aplica:
// sigma final grego e I com ponto turco.
//
// Este golden existe para que a proxima mudanca nesta conta seja BARULHENTA.
// Chave que nomeia socket nao se muda por acidente.
func TestVaultKeyNaoSeMexeuParaCaminhoLatino(t *testing.T) {
	const caminho = "C:/Users/x/Cofre"
	const quer = "1f3bc9ee1596843f"
	if got := config.VaultKey(caminho); got != quer {
		t.Errorf("VaultKey(%q) = %s, quer %s.\n"+
			"A conta da chave mudou. Se foi de proposito, instalar.MigrarChaves "+
			"conserta o diretorio de cache pelo cabecalho -- mas o socket de um "+
			"processo ja no ar nao se conserta, entao a troca so e segura passando "+
			"pelo instalador, que encerra tudo.", caminho, got, quer)
	}
}
