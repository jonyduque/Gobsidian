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
	// Start e o offset do inicio da LINHA do heading, nao do '#'. Um heading
	// aceita ate tres espacos de indentacao, e replace_heading_and_section
	// precisa consumi-los junto — substituir a partir do '#' deixaria a
	// indentacao orfa antes do conteudo novo. Relativo ao mesmo buffer que
	// SplitFrontmatter recebeu, ja com o bodyOffset somado por Parse. End e o
	// offset do fim da secao: o inicio do proximo heading de nivel menor ou
	// igual, ou o fim do buffer. Block.Start e Link.Start seguem a mesma
	// origem.
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
	ID string `json:"id"` // sem o '^'
	// Start e o offset do inicio do BLOCO pai (paragrafo, item de lista,
	// linha de citacao), nao do '^'. Em um item de lista isso cai depois do
	// "- ": replace_block e responsavel so pelo texto, nao por reemitir o
	// marcador de lista nem a indentacao que determina a profundidade do
	// aninhamento.
	//
	// LIMITE CONHECIDO em bloco de varias linhas (paragrafo com quebra
	// suave, ou citacao/item de lista com mais de uma linha): Start..End e
	// uma faixa CONTIGUA no buffer bruto. O prefixo de sintaxe ("> " de
	// citacao, marcador+indentacao de lista) so fica de fora da faixa na
	// PRIMEIRA linha, porque Start vem de Lines().At(0).Start, que so
	// descontou esse prefixo ali. Como a faixa e contigua no buffer, o
	// prefixo de QUALQUER linha de continuacao (a segunda em diante) fica
	// DENTRO dela — nao ha como excluir os dois ao mesmo tempo com um unico
	// intervalo [Start,End). E assimetria inerente a essa escolha, nao um
	// bug: exemplo, "> linha um\n> linha dois ^abc" produz Start=2 End=28,
	// e src[Start:End] == "linha um\n> linha dois ^abc" — o "> " da primeira
	// linha ficou fora, o da segunda ficou dentro. O mesmo vale para
	// indentacao de item de lista em vez de "> ".
	//
	// M4's replace_block, ao escrever conteudo de varias linhas de volta no
	// lugar deste bloco, precisa reemitir esses prefixos de continuacao — o
	// "> " de cada linha de citacao, a indentacao de cada linha de item de
	// lista — porque o range aqui devolvido ja descontou o prefixo da
	// PRIMEIRA linha mas nao o das demais. Escrever de volta sem reemiti-los
	// quebra a sintaxe do bloco.
	Start int64 `json:"start"`
	End   int64 `json:"end"`
}

// offsetUnknown marca um offset que o parser nao conseguiu determinar. Ver o
// comentario de Link.Start.
const offsetUnknown int64 = -1

// Link e uma referencia a outra nota ou a um recurso, na sintaxe wikilink,
// embed ou markdown.
type Link struct {
	Raw    string   `json:"raw"` // texto original, para reescrita fiel
	Target string   `json:"target"`
	Alias  string   `json:"alias,omitempty"`
	Anchor string   `json:"anchor,omitempty"` // heading ou ^bloco
	Kind   LinkKind `json:"kind"`

	// Start e End delimitam o link no buffer, com bodyOffset ja somado.
	//
	// ATENCAO: hoje so sao preenchidos para LinkWiki e LinkEmbed. Para
	// LinkMarkdown e para embeds em grafia Markdown ("![alt](x.png)") ficam no
	// valor sentinela offsetUnknown (-1), porque a AST do goldmark nao entrega
	// o span completo de "[texto](destino)" num unico no. Quem for reescrever
	// um link PRECISA checar Kind antes: reescrever a partir de Start=0 num
	// link Markdown sobrescreveria o inicio da nota.
	//
	// O sentinela e -1, nao zero. Zero e uma posicao legitima — o primeiro
	// byte do buffer — entao usa-lo para "nao sei" repete o erro que
	// config.Flags corrigiu com ReadOnlySet: um valor que nao distingue
	// ausencia de zero. Com -1 um fatiamento estoura alto em vez de
	// sobrescrever o inicio da nota em silencio.
	//
	// Fechar essa lacuna e trabalho de M5 (note_move), que e quem precisa dos
	// offsets. Ate la, Kind e o unico discriminador confiavel.
	Start int64 `json:"start"`
	End   int64 `json:"end"`
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
