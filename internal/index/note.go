package index

import (
	"time"

	"github.com/jonyd/gobsidian/internal/parser"
	"github.com/jonyd/gobsidian/internal/vault"
)

type LinkState int

const (
	LinkOK LinkState = iota
	LinkTargetMissing
	LinkAnchorMissing
)

func (s LinkState) String() string {
	switch s {
	case LinkTargetMissing:
		return "target_missing"
	case LinkAnchorMissing:
		return "anchor_missing"
	default:
		return "ok"
	}
}

type ResolveVia int

const (
	ViaNone ResolveVia = iota
	ViaPath
	ViaName
	ViaAsset
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
}

type Note struct {
	Path    vault.CanonicalPath
	Title   string
	Size    int64
	ModTime time.Time
	// Hash e xxhash do conteudo BRUTO do arquivo, com frontmatter e BOM.
	// E o valor exposto como "hash" e aceito em expected_hash.
	Hash uint64
	EOL  vault.EOLStyle
	BOM  bool
	// CloudOnly marca placeholder do OneDrive: indexado por metadados de
	// diretorio, sem leitura de conteudo.
	CloudOnly bool

	Frontmatter map[string]any
	Tags        []string
	Aliases     []string
	Headings    []parser.Heading
	Blocks      []parser.Block
	Links       []ResolvedLink
	Inline      map[string][]string
}

type Asset struct {
	Path    vault.CanonicalPath
	Size    int64
	ModTime time.Time
}

type Backlink struct {
	From    vault.CanonicalPath
	Anchor  string
	Alias   string
	Context string // texto ao redor da referencia
	Kind    parser.LinkKind
}
