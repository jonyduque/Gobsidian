package mcpsrv

import (
	"testing"

	"github.com/jonyd/gobsidian/internal/vault"
)

// Este teste existe por um panic no boot, nao por hipotese.
//
//	panic: parse "gobsidian://test vault/Origem.md": invalid character " " in host name
//
// A forma anterior concatenava "gobsidian://" com o caminho canonico, o que faz
// o PRIMEIRO SEGMENTO do caminho virar a autoridade da URI. Um espaco e ilegal
// em nome de host, entao qualquer nota na raiz do cofre com espaco no nome
// derrubava o servidor em AddResource — antes de servir a primeira requisicao,
// com o processo saindo sem que nenhuma tool tivesse sido anunciada.
//
// Nome de pasta com espaco e o caso comum num cofre do Obsidian, nao o exotico.
func TestResourceURIIsParseableForOrdinaryVaultPaths(t *testing.T) {
	casos := []struct {
		nome string
		path vault.CanonicalPath
		want string
	}{
		{"raiz sem espaco", "Origem.md", "gobsidian:///Origem.md"},
		{"raiz com espaco", "Minha nota.md", "gobsidian:///Minha%20nota.md"},
		{"subpasta com espaco", "test vault/Origem.md", "gobsidian:///test%20vault/Origem.md"},
		{"espaco no arquivo", "Civil/PONTO 03.md", "gobsidian:///Civil/PONTO%2003.md"},

		// A barra separa segmentos e NAO pode ser escapada: escapa-la
		// transformaria a hierarquia num nome unico e o resource deixaria de
		// casar com o caminho canonico que o handler resolve.
		{"barras preservadas", "a/b/c.md", "gobsidian:///a/b/c.md"},

		// Percent literal precisa virar %25, senao a decodificacao do outro
		// lado inventa um byte que o usuario nao escreveu.
		{"percent literal", "50% de desconto.md", "gobsidian:///50%25%20de%20desconto.md"},

		// # e ? delimitam fragmento e query. Sem escape, tudo depois deles
		// deixa de fazer parte do caminho.
		{"cerquilha", "Nota #1.md", "gobsidian:///Nota%20%231.md"},
		{"interrogacao", "E agora?.md", "gobsidian:///E%20agora%3F.md"},

		// Acentos sao comuns em cofre em portugues. UTF-8 escapado byte a byte.
		{"acento", "Capítulo.md", "gobsidian:///Cap%C3%ADtulo.md"},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			got := resourceURI(c.path)
			if got != c.want {
				t.Errorf("resourceURI(%q) = %q, quer %q", c.path, got, c.want)
			}
		})
	}
}

// A ida e a volta precisam fechar, senao o resource anunciado nao e o resource
// que o handler consegue ler — e o sintoma seria "nota nao encontrada" para
// exatamente as notas cujo nome exigiu escape.
func TestResourceURIRoundTrips(t *testing.T) {
	paths := []vault.CanonicalPath{
		"Origem.md",
		"Minha nota.md",
		"test vault/Civil/PONTO 03.md",
		"50% de desconto.md",
		"Nota #1.md",
		"Capítulo.md",
		"a/b/c.md",
	}

	for _, p := range paths {
		t.Run(string(p), func(t *testing.T) {
			uri := resourceURI(p)
			got, err := pathFromResourceURI(uri)
			if err != nil {
				t.Fatalf("pathFromResourceURI(%q): %v", uri, err)
			}
			if got != string(p) {
				t.Errorf("volta de %q deu %q, quer %q", uri, got, p)
			}
		})
	}
}

// A forma de duas barras e o que a documentacao descrevia e o que um host pode
// ter construido a partir dela. Aceitar na leitura custa uma linha; recusar
// transforma um documento desatualizado em nota inalcancavel.
func TestPathFromResourceURIAcceptsTheLegacyTwoSlashForm(t *testing.T) {
	got, err := pathFromResourceURI("gobsidian://Civil/PONTO%2003.md")
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if got != "Civil/PONTO 03.md" {
		t.Errorf("got %q, quer %q", got, "Civil/PONTO 03.md")
	}
}

func TestPathFromResourceURIRejectsForeignScheme(t *testing.T) {
	if _, err := pathFromResourceURI("obsidian://open?vault=X"); err == nil {
		t.Error("quer erro para esquema estrangeiro, veio nil")
	}
}
