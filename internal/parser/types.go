// Package parser transforma bytes de uma nota em estrutura. E puro: sem I/O,
// sem estado, sem conhecimento do indice ou do cofre. Recebe []byte, devolve
// ParsedNote. Isso o torna trivialmente testavel por golden file e
// trivialmente paralelizavel.
package parser

// LinkKind distingue as tres sintaxes de link do Obsidian: wikilink, embed
// (wikilink com "!" na frente) e link markdown ([texto](alvo)). A distincao
// importa porque reescrita e resolucao tratam cada uma diferente.
type LinkKind int

// Valores de LinkKind. LinkWiki e o zero-value porque e o caso mais comum em
// notas do Obsidian.
const (
	LinkWiki LinkKind = iota
	LinkEmbed
	LinkMarkdown
)

func (k LinkKind) String() string {
	switch k {
	case LinkEmbed:
		return "embed"
	case LinkMarkdown:
		return "markdown"
	default:
		return "wikilink"
	}
}

// Heading e uma secao do documento delimitada por uma linha "#"..."######".
type Heading struct {
	Level int    `json:"level"`
	Text  string `json:"text"`
	Slug  string `json:"slug"`
	// Start e o offset do '#', relativo ao mesmo buffer que SplitFrontmatter
	// recebeu — ja com o bodyOffset somado por Parse. End e o offset do fim da
	// secao: o inicio do proximo heading de nivel menor ou igual, ou o fim do
	// buffer. Block.Start e Link.Start seguem a mesma origem.
	//
	// A origem precisa estar escrita aqui porque um ParsedNote sozinho nao
	// permite descobri-la: quem so recebe a struct nao tem como saber se os
	// offsets sao do corpo ou do buffer inteiro.
	Start int64 `json:"start"`
	End   int64 `json:"end"`
	// BodyStart e o offset logo apos a linha do heading. E o que
	// replace_section usa: preserva o titulo e substitui so o que vem abaixo.
	BodyStart int64 `json:"body_start"`
}

// Block e um bloco identificado por um marcador "^id" no fim de linha.
type Block struct {
	ID    string `json:"id"` // sem o '^'
	Start int64  `json:"start"`
	End   int64  `json:"end"`
}

// Link e uma referencia a outra nota ou a um recurso, na sintaxe wikilink,
// embed ou markdown.
type Link struct {
	Raw    string   `json:"raw"` // texto original, para reescrita fiel
	Target string   `json:"target"`
	Alias  string   `json:"alias,omitempty"`
	Anchor string   `json:"anchor,omitempty"` // heading ou ^bloco
	Kind   LinkKind `json:"kind"`
	Start  int64    `json:"start"`
	End    int64    `json:"end"`
}

// ParsedNote e a estrutura extraida dos bytes de uma nota: frontmatter,
// headings, blocos e links, prontos para o indice consumir sem reparsear.
type ParsedNote struct {
	Frontmatter map[string]any      `json:"frontmatter,omitempty"`
	Tags        []string            `json:"tags,omitempty"`
	Aliases     []string            `json:"aliases,omitempty"`
	Headings    []Heading           `json:"headings,omitempty"`
	Blocks      []Block             `json:"blocks,omitempty"`
	Links       []Link              `json:"links,omitempty"`
	Inline      map[string][]string `json:"inline,omitempty"`
	// Title vem do frontmatter, do primeiro H1, ou fica vazio para que o
	// chamador use o nome do arquivo. O parser nao conhece o nome do arquivo.
	Title string `json:"title,omitempty"`
	// FrontmatterErr registra frontmatter malformado sem abortar o parse.
	FrontmatterErr string `json:"frontmatter_err,omitempty"`
}
