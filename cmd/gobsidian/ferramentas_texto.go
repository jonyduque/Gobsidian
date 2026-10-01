package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/jonyduque/Gobsidian/internal/console"
	"github.com/jonyduque/Gobsidian/internal/textos"
)

// formatador escreve o resultado de uma tool para uma pessoa ler.
//
// Cada um decodifica o JSON da tool numa visao propria, com so os campos que
// mostra -- e nao nos tipos de service. O JSON e o contrato publico
// (docs/TOOLS.md); os tipos de service sao detalhe interno, e amarrar a tela a
// eles faria uma renomeacao interna quebrar a CLI.
//
// Conteudo de nota e dado, e sai como esta: sem marcacao desenhada e sem
// adaptacao de acento -- um "**" no corpo da nota nao e marcacao do produto.
type formatador func(con *console.Stream, w io.Writer, raw json.RawMessage) error

var formatadores = map[string]formatador{
	"note_read":          formatarNoteRead,
	"note_list":          formatarNoteList,
	"note_outline":       formatarNoteOutline,
	"note_metadata":      formatarNoteMetadata,
	"note_create":        formatarNoteCreate,
	"note_append":        formatarNoteAppend,
	"note_patch":         formatarNotePatch,
	"note_move":          formatarNoteMove,
	"note_delete":        formatarNoteDelete,
	"vault_stats":        formatarStats,
	"vault_search":       formatarSearch,
	"vault_broken_links": formatarBrokenLinks,
	"tag_list":           formatarTagList,
	"link_graph":         formatarGraph,
}

// formatar escolhe o formatador da tool. Tool sem formatador sai em JSON
// indentado -- e TestCadaToolTemFormatador reprova, porque uma tool nova sem
// formatador e um esquecimento, nao uma decisao.
func formatar(w io.Writer, tool string, raw json.RawMessage) error {
	con := console.New(w)
	f, ok := formatadores[tool]
	if !ok {
		var b bytes.Buffer
		if err := json.Indent(&b, raw, "", "  "); err != nil {
			return err
		}
		_, _ = fmt.Fprintln(w, b.String())
		return nil
	}
	return f(con, w, raw)
}

func decodificar(raw json.RawMessage, v any) error {
	return json.Unmarshal(raw, v)
}

// --- note_read -----------------------------------------------------------

type itemLido struct {
	Path       string `json:"path"`
	Content    string `json:"content"`
	Truncated  bool   `json:"truncated"`
	TotalSize  int64  `json:"total_size"`
	NextOffset *int64 `json:"next_offset"`
	Error      *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func formatarNoteRead(con *console.Stream, w io.Writer, raw json.RawMessage) error {
	var lote struct {
		Items []itemLido `json:"items"`
	}
	if err := decodificar(raw, &lote); err == nil && lote.Items != nil {
		for i, it := range lote.Items {
			if i > 0 {
				_, _ = fmt.Fprintln(w)
			}
			if it.Error != nil {
				con.Err(textos.TextoLoteItemErro, it.Path, it.Error.Message, it.Error.Code)
				continue
			}
			con.Titulo("%s", it.Path)
			escreverConteudo(con, w, it)
		}
		return nil
	}
	var um itemLido
	if err := decodificar(raw, &um); err != nil {
		return err
	}
	escreverConteudo(con, w, um)
	return nil
}

func escreverConteudo(con *console.Stream, w io.Writer, it itemLido) {
	_, _ = io.WriteString(w, it.Content)
	if !strings.HasSuffix(it.Content, "\n") {
		_, _ = fmt.Fprintln(w)
	}
	if it.Truncated && it.NextOffset != nil {
		con.Warn(textos.TextoLidoCortado, *it.NextOffset, it.TotalSize, *it.NextOffset)
	}
}

// --- note_list -----------------------------------------------------------

func formatarNoteList(con *console.Stream, _ io.Writer, raw json.RawMessage) error {
	var r struct {
		Notes []struct {
			Path  string   `json:"path"`
			Title string   `json:"title"`
			Tags  []string `json:"tags"`
		} `json:"notes"`
		Total     int  `json:"total"`
		Truncated bool `json:"truncated"`
	}
	if err := decodificar(raw, &r); err != nil {
		return err
	}
	if len(r.Notes) == 0 {
		con.Info("%s", textos.TextoSemNotas)
		return nil
	}
	linhas := make([]string, 0, len(r.Notes))
	for _, n := range r.Notes {
		linha := "  " + n.Path
		if len(n.Tags) > 0 {
			linha += "  " + con.Dim("#"+strings.Join(n.Tags, " #"))
		}
		linhas = append(linhas, linha)
	}
	con.Bloco(fmt.Sprintf(textos.TextoNotasTitulo, len(r.Notes), r.Total), linhas, "")
	avisarCorte(con, r.Truncated)
	return nil
}

func avisarCorte(con *console.Stream, cortado bool) {
	if cortado {
		con.Warn("%s", textos.TextoCortado)
	}
}

// --- note_outline --------------------------------------------------------

type heading struct {
	Level int    `json:"level"`
	Text  string `json:"text"`
}

func formatarNoteOutline(con *console.Stream, _ io.Writer, raw json.RawMessage) error {
	var r struct {
		Path       string    `json:"path"`
		Headings   []heading `json:"headings"`
		Candidates []struct {
			Text string `json:"text"`
			Kind string `json:"kind"`
		} `json:"candidates"`
		Truncated bool `json:"truncated"`
	}
	if err := decodificar(raw, &r); err != nil {
		return err
	}
	linhas := make([]string, 0, len(r.Headings))
	for _, h := range r.Headings {
		linhas = append(linhas, "  "+strings.Repeat("  ", max(h.Level-1, 0))+h.Text)
	}
	if len(r.Candidates) > 0 {
		linhas = append(linhas, "", "  "+con.Dim(textos.TextoCandidatos))
		for _, c := range r.Candidates {
			linhas = append(linhas, "  "+c.Text+"  "+con.Dim(c.Kind))
		}
	}
	con.Bloco(fmt.Sprintf(textos.TextoOutlineTitulo, r.Path, len(r.Headings), len(r.Candidates)), linhas, "")
	avisarCorte(con, r.Truncated)
	return nil
}

// --- note_metadata -------------------------------------------------------

func formatarNoteMetadata(con *console.Stream, _ io.Writer, raw json.RawMessage) error {
	var r struct {
		Path        string         `json:"path"`
		Title       string         `json:"title"`
		Hash        string         `json:"hash"`
		Frontmatter map[string]any `json:"frontmatter"`
		Tags        []string       `json:"tags"`
		Aliases     []string       `json:"aliases"`
		Headings    []heading      `json:"headings"`
		Blocks      []string       `json:"blocks"`
		Links       []struct {
			Target string `json:"target"`
		} `json:"links"`
		Backlinks []struct {
			Source string `json:"source"`
		} `json:"backlinks"`
	}
	if err := decodificar(raw, &r); err != nil {
		return err
	}
	campos := []console.Campo{
		console.Campof(textos.CampoTitulo, "%s", r.Title),
		{Chave: "*Hash*", Valor: r.Hash},
	}
	if len(r.Frontmatter) > 0 {
		chaves := make([]string, 0, len(r.Frontmatter))
		for k := range r.Frontmatter {
			chaves = append(chaves, k)
		}
		sort.Strings(chaves)
		campos = append(campos, console.Campof(textos.CampoFrontmatter, "%s", strings.Join(chaves, ", ")))
	}
	if len(r.Tags) > 0 {
		campos = append(campos, console.Campof(textos.CampoTags, "%s", strings.Join(r.Tags, ", ")))
	}
	if len(r.Aliases) > 0 {
		campos = append(campos, console.Campof(textos.CampoAliases, "%s", strings.Join(r.Aliases, ", ")))
	}
	campos = append(campos, console.Campof(textos.CampoHeadings, "%d", len(r.Headings)))
	if len(r.Blocks) > 0 {
		campos = append(campos, console.Campof(textos.CampoBlocos, "%d", len(r.Blocks)))
	}
	campos = append(campos,
		console.Campof(textos.CampoLinksSaida, "%d", len(r.Links)),
		console.Campof(textos.CampoBacklinks, "%d", len(r.Backlinks)),
	)
	con.Campos(r.Path, campos)
	return nil
}

// --- escrita -------------------------------------------------------------

// diffDaSimulacao imprime o diff de um --dry-run como esta: e dado.
func diffDaSimulacao(con *console.Stream, w io.Writer, diff string) {
	con.Info("%s", textos.TextoSimulacao)
	if diff != "" {
		_, _ = io.WriteString(w, diff)
		if !strings.HasSuffix(diff, "\n") {
			_, _ = fmt.Fprintln(w)
		}
	}
}

type resultadoDeEscrita struct {
	Path     string `json:"path"`
	Diff     string `json:"diff"`
	Hash     string `json:"hash"`
	Created  *bool  `json:"created"`
	Appended *bool  `json:"appended"`
	Patched  *bool  `json:"patched"`
}

func (r resultadoDeEscrita) feito() bool {
	for _, b := range []*bool{r.Created, r.Appended, r.Patched} {
		if b != nil {
			return *b
		}
	}
	return false
}

func formatarEscrita(texto string) formatador {
	return func(con *console.Stream, w io.Writer, raw json.RawMessage) error {
		var r resultadoDeEscrita
		if err := decodificar(raw, &r); err != nil {
			return err
		}
		if !r.feito() {
			diffDaSimulacao(con, w, r.Diff)
			return nil
		}
		con.OK(texto, r.Path)
		if r.Hash != "" {
			con.Detail(textos.TextoHash, r.Hash)
		}
		return nil
	}
}

var (
	formatarNoteCreate = formatarEscrita(textos.TextoCriada)
	formatarNoteAppend = formatarEscrita(textos.TextoAcrescentada)
	formatarNotePatch  = formatarEscrita(textos.TextoSubstituida)
)

func formatarNoteMove(con *console.Stream, w io.Writer, raw json.RawMessage) error {
	var r struct {
		From         string            `json:"from"`
		To           string            `json:"to"`
		Rewritten    []string          `json:"rewritten"`
		LinksUpdated int               `json:"links_updated"`
		DryRun       bool              `json:"dry_run"`
		Diffs        map[string]string `json:"diffs"`
	}
	if err := decodificar(raw, &r); err != nil {
		return err
	}
	if r.DryRun {
		chaves := make([]string, 0, len(r.Diffs))
		for k := range r.Diffs {
			chaves = append(chaves, k)
		}
		sort.Strings(chaves)
		var todos strings.Builder
		for _, k := range chaves {
			todos.WriteString(r.Diffs[k])
		}
		diffDaSimulacao(con, w, todos.String())
		return nil
	}
	con.OK(textos.TextoMovida, r.From, r.To, r.LinksUpdated, len(r.Rewritten))
	for _, n := range r.Rewritten {
		con.Detail("%s", n)
	}
	return nil
}

func formatarNoteDelete(con *console.Stream, _ io.Writer, raw json.RawMessage) error {
	var r struct {
		Path         string   `json:"path"`
		Deleted      bool     `json:"deleted"`
		MovedToTrash bool     `json:"moved_to_trash"`
		TrashPath    string   `json:"trash_path"`
		BrokenLinks  []string `json:"broken_links"`
		DryRun       bool     `json:"dry_run"`
	}
	if err := decodificar(raw, &r); err != nil {
		return err
	}
	switch {
	case r.DryRun:
		con.Info("%s", textos.TextoSimulacao)
	case r.MovedToTrash:
		con.OK(textos.TextoParaLixeira, r.TrashPath)
	default:
		con.OK(textos.TextoExcluida, r.Path)
	}
	if len(r.BrokenLinks) > 0 {
		con.Warn(textos.TextoLinksQuebrar, strings.Join(r.BrokenLinks, ", "))
	}
	return nil
}

// --- vault_stats ---------------------------------------------------------

func formatarStats(con *console.Stream, _ io.Writer, raw json.RawMessage) error {
	var r struct {
		Notes             int   `json:"notes"`
		Assets            int   `json:"assets"`
		TotalSize         int64 `json:"total_size"`
		Orphans           *int  `json:"orphans"`
		BrokenLinks       *int  `json:"broken_links"`
		BrokenAnchors     *int  `json:"broken_anchors"`
		FrontmatterErrors *int  `json:"frontmatter_errors"`
		Collisions        int   `json:"alias_collisions"`
	}
	if err := decodificar(raw, &r); err != nil {
		return err
	}
	campos := []console.Campo{
		console.Campof(textos.CampoNotas, "%d", r.Notes),
		console.Campof(textos.CampoAnexos, "%d", r.Assets),
		{Chave: textos.CampoTamanho, Valor: fmt.Sprintf("%d", r.TotalSize), Nota: textos.NotaBytes},
	}
	opcional := func(chave string, n *int) {
		if n != nil {
			campos = append(campos, console.Campof(chave, "%d", *n))
		}
	}
	opcional(textos.CampoOrfas, r.Orphans)
	opcional(textos.CampoLinksQuebrados, r.BrokenLinks)
	opcional(textos.CampoAncorasQuebradas, r.BrokenAnchors)
	opcional(textos.CampoErrosFrontmatter, r.FrontmatterErrors)
	campos = append(campos, console.Campof(textos.CampoColisoes, "%d", r.Collisions))
	con.Campos(textos.TextoStatsTitulo, campos)
	return nil
}

// --- vault_search --------------------------------------------------------

func formatarSearch(con *console.Stream, _ io.Writer, raw json.RawMessage) error {
	var r struct {
		Results []struct {
			Path    string  `json:"path"`
			Score   float64 `json:"score"`
			Snippet string  `json:"snippet"`
		} `json:"results"`
		Total     int  `json:"total"`
		Truncated bool `json:"truncated"`
	}
	if err := decodificar(raw, &r); err != nil {
		return err
	}
	if len(r.Results) == 0 {
		con.Info("%s", textos.TextoSemNotas)
		return nil
	}
	// Uma lista numa moldura so: um bloco por resultado seria uma moldura a
	// cada duas linhas.
	corpos := make([]string, 0, 2*len(r.Results))
	for _, m := range r.Results {
		corpos = append(corpos, fmt.Sprintf("  %s  %s", m.Path, con.Dim(fmt.Sprintf("%.2f", m.Score))))
		if m.Snippet != "" {
			corpos = append(corpos, "    "+con.Dim("... "+m.Snippet+" ..."))
		}
	}
	con.Bloco(fmt.Sprintf(textos.TextoNotasTitulo, len(r.Results), r.Total), corpos, "")
	avisarCorte(con, r.Truncated)
	return nil
}

// --- vault_broken_links --------------------------------------------------

func formatarBrokenLinks(con *console.Stream, _ io.Writer, raw json.RawMessage) error {
	var r struct {
		Links []struct {
			Source string `json:"source"`
			Target string `json:"target"`
			Anchor string `json:"anchor"`
			State  string `json:"state"`
		} `json:"links"`
		Total     int  `json:"total"`
		Truncated bool `json:"truncated"`
	}
	if err := decodificar(raw, &r); err != nil {
		return err
	}
	if len(r.Links) == 0 {
		con.OK("%s", textos.TextoSemQuebrados)
		return nil
	}
	linhas := make([]string, 0, len(r.Links))
	for _, l := range r.Links {
		alvo := l.Target
		if l.Anchor != "" {
			alvo += "#" + l.Anchor
		}
		linhas = append(linhas, "  "+l.Source+" -> "+alvo+"  "+con.Dim(l.State))
	}
	con.Bloco(fmt.Sprintf(textos.TextoQuebrados, len(r.Links), r.Total), linhas, "")
	avisarCorte(con, r.Truncated)
	return nil
}

// --- tag_list ------------------------------------------------------------

type noDeTag struct {
	Tag      string    `json:"tag"`
	Count    int       `json:"count"`
	Children []noDeTag `json:"children"`
}

func formatarTagList(con *console.Stream, _ io.Writer, raw json.RawMessage) error {
	var r struct {
		Tags []noDeTag `json:"tags"`
	}
	if err := decodificar(raw, &r); err != nil {
		return err
	}
	if len(r.Tags) == 0 {
		con.Info("%s", textos.TextoSemTags)
		return nil
	}
	var linhas []string
	var andar func(ns []noDeTag, nivel int)
	andar = func(ns []noDeTag, nivel int) {
		for _, n := range ns {
			linhas = append(linhas, fmt.Sprintf("  %s#%s  %s", strings.Repeat("  ", nivel), n.Tag, con.Dim(fmt.Sprintf("%d", n.Count))))
			andar(n.Children, nivel+1)
		}
	}
	andar(r.Tags, 0)
	con.Bloco(fmt.Sprintf(textos.TextoTagsTitulo, len(r.Tags)), linhas, "")
	return nil
}

// --- link_graph ----------------------------------------------------------

func formatarGraph(con *console.Stream, _ io.Writer, raw json.RawMessage) error {
	var r struct {
		Nodes []struct {
			Path     string `json:"path"`
			Distance int    `json:"distance"`
		} `json:"nodes"`
		Edges []struct {
			Source   string `json:"source"`
			Target   string `json:"target"`
			Resolved bool   `json:"resolved"`
		} `json:"edges"`
		Truncated bool `json:"truncated"`
	}
	if err := decodificar(raw, &r); err != nil {
		return err
	}
	centro := ""
	linhas := make([]string, 0, len(r.Nodes)+len(r.Edges)+1)
	for _, n := range r.Nodes {
		if n.Distance == 0 {
			centro = n.Path
		}
		linhas = append(linhas, fmt.Sprintf("  %s%s  %s", strings.Repeat("  ", n.Distance), n.Path, con.Dim(fmt.Sprintf("%d", n.Distance))))
	}
	if len(r.Edges) > 0 {
		linhas = append(linhas, "")
		for _, e := range r.Edges {
			seta := " -> "
			if !e.Resolved {
				seta = " -x "
			}
			linhas = append(linhas, "  "+con.Dim(e.Source+seta+e.Target))
		}
	}
	con.Bloco(fmt.Sprintf(textos.TextoGrafoTitulo, len(r.Nodes), len(r.Edges), centro), linhas, "")
	avisarCorte(con, r.Truncated)
	return nil
}
