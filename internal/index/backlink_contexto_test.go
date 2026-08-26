package index_test

import (
	"context"
	"strings"
	"testing"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/vault"
)

// origemComQuatroGrafias exercita as QUATRO sintaxes de link. A sondagem de
// 2026-08-26 mostrou que todas carregam span correto — o comentário de
// parser/types.go que dizia o contrário estava velho (achado B14), e o
// contexto depende justamente desses offsets.
const origemComQuatroGrafias = "# Origem\n\n" +
	"O acordao do RESP 1234 foi superado pelo entendimento de [[Alvo]] quanto a prescricao.\n\n" +
	"Ver tambem o resumo em [o alvo](Alvo.md) para o historico.\n"

func montarIndice(t *testing.T, root string) *index.Index {
	t.Helper()
	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	idx := index.New()
	if err := idx.Build(context.Background(), v); err != nil {
		t.Fatalf("Build: %v", err)
	}
	return idx
}

// TestBacklinkTrazContexto é o achado A8: `Backlink.Context` existia no tipo,
// era documentado como "texto ao redor da referência", e as TRÊS construções
// de Backlink escreviam `Context: ""`. O campo chegava vazio ao host em toda
// resposta de `vault_backlinks`, sempre — um campo de API com valor fixo.
func TestBacklinkTrazContexto(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "Origem.md", origemComQuatroGrafias)
	writeFile(t, root, "Alvo.md", "# Alvo\n")

	bls := montarIndice(t, root).Backlinks("Alvo.md")
	if len(bls) != 2 {
		t.Fatalf("backlinks = %d, queria 2 (wikilink e markdown)", len(bls))
	}

	for _, bl := range bls {
		if bl.Context == "" {
			t.Errorf("kind=%v: Context vazio — o campo continua sendo promessa nao cumprida", bl.Kind)
			continue
		}
		// O contexto tem de vir da VIZINHANÇA da referência, não ser um
		// pedaço qualquer da nota: é isso que decide se o backlink interessa
		// sem abrir a origem.
		if !strings.Contains(bl.Context, "Alvo") {
			t.Errorf("kind=%v: Context %q nao contem a propria referencia", bl.Kind, bl.Context)
		}
		if strings.ContainsAny(bl.Context, "\r\n") {
			t.Errorf("kind=%v: Context %q tem quebra de linha", bl.Kind, bl.Context)
		}
	}

	// A referência wikilink está no meio de uma frase sobre prescrição; a
	// markdown, numa frase sobre histórico. Contextos iguais significariam que
	// o recorte ignora a posição do link — o defeito que pareceria corrigido.
	if bls[0].Context == bls[1].Context {
		t.Errorf("os dois backlinks tem o MESMO contexto %q: o recorte ignora a posicao do link", bls[0].Context)
	}

	var achouPrescricao bool
	for _, bl := range bls {
		if strings.Contains(bl.Context, "prescricao") {
			achouPrescricao = true
		}
	}
	if !achouPrescricao {
		t.Errorf("nenhum contexto trouxe a vizinhanca da referencia wikilink; contextos=%q, %q",
			bls[0].Context, bls[1].Context)
	}
}

// TestContextoSobreviveAoCacheDeMetadados é o achado A8 pela outra metade, e a
// prova do B11 ao mesmo tempo.
//
// O contexto NÃO é recalculável a partir do índice — recortá-lo de novo exigiria
// reler o cofre no boot, que é o custo que o cache existe para não pagar. Então
// ele é persistido, e persistir exigiu subir o formato de 2 para 3.
//
// **Era esse bump que tornava o B11 perigoso.** Até 2026-08-26 duas constantes
// independentes guardavam o mesmo portão — `IndexCacheFormatVersion` conferida
// em `LoadIndexCache` e `indexCacheCodecVers` conferida no cabeçalho pelo
// decodificador. Subir uma sem a outra não quebra build nem teste: faz o leitor
// recusar todo save que o próprio processo acabou de gravar, com reconstrução
// completa a cada boot e nenhum log dizendo por quê. Este round-trip é o teste
// que teria pego isso, e não existia.
func TestContextoSobreviveAoCacheDeMetadados(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "Origem.md", origemComQuatroGrafias)
	writeFile(t, root, "Alvo.md", "# Alvo\n")

	idx := montarIndice(t, root)
	antes := idx.Backlinks("Alvo.md")
	if len(antes) == 0 {
		t.Fatal("sem backlinks antes do cache: o cenario nao exercita nada")
	}

	ctx := context.Background()
	cacheDir := t.TempDir()
	if err := index.SaveIndexCache(ctx, cacheDir, root, idx); err != nil {
		t.Fatalf("SaveIndexCache: %v", err)
	}

	// Recusar aqui é o sintoma do B11: o processo grava e não consegue reler.
	carregado, _, err := index.LoadIndexCache(ctx, cacheDir, root)
	if err != nil {
		t.Fatalf("LoadIndexCache recusou um cache que ESTE processo acabou de gravar: %v\n"+
			"e o sintoma do B11 — as duas constantes de versao divergiram", err)
	}

	depois := carregado.Backlinks("Alvo.md")
	if len(depois) != len(antes) {
		t.Fatalf("backlinks depois do cache = %d, antes = %d", len(depois), len(antes))
	}
	for i := range antes {
		if depois[i].Context != antes[i].Context {
			t.Errorf("backlink %d: contexto %q depois do cache, %q antes\n"+
				"o cache degrada a resposta: mesma pergunta, respostas diferentes conforme houve boot frio",
				i, depois[i].Context, antes[i].Context)
		}
	}
}
