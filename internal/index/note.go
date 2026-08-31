package index

import (
	"time"

	"github.com/jonyd/gobsidian/internal/parser"
	"github.com/jonyd/gobsidian/internal/text"
	"github.com/jonyd/gobsidian/internal/vault"
)

// LinkState diz o que aconteceu ao tentar resolver um link.
type LinkState int

const (
	// LinkOK resolveu para uma nota ou anexo do cofre.
	LinkOK LinkState = iota
	// LinkTargetMissing deveria resolver para algo do cofre e nao resolveu.
	// E o que conta como link quebrado.
	LinkTargetMissing
	// LinkAnchorMissing resolveu a nota, mas nao o heading ou bloco citado
	// depois do #. A nota existe; a ancora dentro dela, nao.
	LinkAnchorMissing
	// LinkExternal e um alvo que nunca foi para o cofre: tem esquema de URI.
	//
	// Sem este estado, toda URL numa nota entrava como link quebrado, e a
	// contagem de links quebrados — que o PRD chama de principal sinal de
	// saude do cofre — afogava em falso positivo. Num cofre de estudo com
	// centenas de referencias externas o numero deixava de ter uso.
	//
	// Confirmado contra o metadataCache real: o Obsidian nao registra URL
	// externa nem em resolvedLinks nem em unresolvedLinks. Para ele, nao e
	// link do cofre.
	LinkExternal
)

func (s LinkState) String() string {
	switch s {
	case LinkTargetMissing:
		return "target_missing"
	case LinkAnchorMissing:
		return "anchor_missing"
	case LinkExternal:
		return "external"
	default:
		return "ok"
	}
}

// ResolveVia registra QUAL regra resolveu o link. Duas notas com o mesmo
// nome em pastas diferentes tornam a regra que venceu uma informacao util
// para diagnosticar um alvo inesperado.
type ResolveVia int

const (
	// ViaNone significa que nenhuma regra resolveu o link.
	ViaNone ResolveVia = iota
	// ViaPath resolveu porque o alvo ja era um caminho do cofre.
	ViaPath
	// ViaName resolveu pelo nome do arquivo, sem o caminho.
	ViaName
	// ViaAsset resolveu para um anexo, nao para uma nota.
	ViaAsset
	// ViaAlias resolveu por um alias declarado no frontmatter. E uma
	// divergencia deliberada do Obsidian, que NAO resolve links por alias.
	ViaAlias
)

func (v ResolveVia) String() string {
	switch v {
	case ViaPath:
		return "path"
	case ViaName:
		return "name"
	case ViaAsset:
		return "asset"
	case ViaAlias:
		return "alias"
	default:
		return ""
	}
}

// ResolvedLink e um parser.Link mais o resultado da resolucao, que depende do
// cofre inteiro e por isso nao pode ser feita no parser.
type ResolvedLink struct {
	parser.Link
	Resolved vault.CanonicalPath
	Via      ResolveVia
	State    LinkState
	// Context e o texto ao redor da referencia, recortado do corpo no momento
	// em que a nota foi lida. Diferente de Resolved/Via/State — que o codec do
	// cache NAO persiste porque sao recalculados na carga — o contexto nao e
	// derivavel do que o indice guarda: recalcula-lo exigiria reler o cofre no
	// boot, que e exatamente o que o cache existe para evitar. Por isso ele e
	// persistido, e por isso o formato do cache subiu para 3. Ver
	// contexto_link.go.
	Context string
}

// Note e uma nota indexada: metadados e offsets, sem o corpo.
// Ler o corpo custa uma ida ao disco, de proposito.
type Note struct {
	Path      vault.CanonicalPath
	Title     string
	TitleNorm string
	Size      int64
	ModTime   time.Time
	// Hash e xxhash do conteudo BRUTO do arquivo, com frontmatter e BOM.
	// E o valor exposto como "hash" e aceito em expected_hash.
	Hash uint64
	EOL  vault.EOLStyle
	BOM  bool
	// CloudOnly marca placeholder do OneDrive: indexado por metadados de
	// diretorio, sem leitura de conteudo.
	CloudOnly bool

	Frontmatter    map[string]any
	FrontmatterErr string
	Tags           []string
	Aliases        []string
	Headings       []parser.Heading
	Blocks         []parser.Block
	Links          []ResolvedLink
	Inline         map[string][]string
}

// Asset e um anexo. Nao tem hash nem conteudo: ele e indexado por nome e
// nunca aberto, porque abri-lo dispararia download de arquivo somente-nuvem e
// porque o cofre so precisa saber que ele existe para nao chamar de quebrado
// o embed que aponta para ele.
type Asset struct {
	Path    vault.CanonicalPath
	Size    int64
	ModTime time.Time
}

// Backlink e uma referencia CHEGANDO numa nota, com o contexto de texto ao
// redor — o backlink sem contexto obriga a abrir a nota de origem para saber
// se a referencia interessa.
type Backlink struct {
	From    vault.CanonicalPath
	Anchor  string
	Alias   string
	Context string // texto ao redor da referencia
	// Heading e o titulo da secao da nota de origem em que a referencia esta.
	// Derivado dos headings da propria Note na hora de montar o backlink, nao
	// guardado por link: custa zero byte de cache e evita uma segunda conta do
	// mesmo dado. Ver headingDoLink.
	Heading string
	Kind    parser.LinkKind
}

// backlinkDe monta o Backlink que corresponde a um link ja resolvido.
//
// E o UNICO lugar que faz essa conversao. Havia TRES — backlinks.go:20,
// update.go:154 e update.go:494 — e as tres escreviam `Context: ""` a mao. Foi
// assim que o campo ficou tres anos prometido no tipo e vazio na resposta: nao
// havia um lugar onde escrever o valor, havia tres para esquecer.
func backlinkDe(de *Note, l ResolvedLink) Backlink {
	return Backlink{
		From:    de.Path,
		Anchor:  l.Anchor,
		Alias:   l.Alias,
		Context: l.Context,
		Heading: headingDoLink(de.Headings, l.Start),
		Kind:    l.Kind,
	}
}

// normalizeTitleForNote produz o campo TitleNorm de uma Note. Única função
// que o calcula; todas as construções de Note chamam ela.
//
// Havia aqui um `_ = text.Normalize("")` com o comentário "garante que text é
// usado; mutação deve remover text.Normalize" — um andaime de prova de mutação
// rodando EM PRODUÇÃO, uma vez por nota (achado B10). Uma mutação que deixa o
// import órfão se resolve na MUTAÇÃO, mantendo o pacote vivo do outro lado
// (`_ = text.Normalize` na própria substituição), e não deixando trabalho morto
// no caminho quente do índice.
func normalizeTitleForNote(title string) string {
	return text.Normalize(title)
}
