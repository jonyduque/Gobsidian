package service

import (
	"context"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/jonyduque/Gobsidian/internal/index"
	"github.com/jonyduque/Gobsidian/internal/text"
)

// TreeRequest pede a arvore do cofre, ou um pedaco dela.
//
// Folder vazio e a raiz. Sem Recursive, a resposta traz so os filhos diretos
// da pasta pedida; com Recursive, a subarvore inteira. E a mesma semantica de
// index.Query.Folder e Recursive.
type TreeRequest struct {
	Folder    string
	Recursive bool
}

// FolderItem e uma pasta do cofre. Path e Parent vem sem barra final, e a
// raiz e o caminho vazio.
type FolderItem struct {
	Path   string
	Parent string
	// Notes conta as notas diretamente dentro da pasta; NotesRecursive, as da
	// subarvore inteira.
	Notes          int
	NotesRecursive int
}

// TreeNote e a projecao de nota que a arvore precisa: sem hash, tags nem
// frontmatter, porque nao e a projecao de note_list.
type TreeNote struct {
	Path     string
	Folder   string
	Title    string
	Size     int64
	Modified time.Time
}

// TreeResult traz a pasta pedida em Folder, e em Folders e Notes os itens
// abaixo dela, ordenados por caminho.
type TreeResult struct {
	Folder  FolderItem
	Folders []FolderItem
	Notes   []TreeNote
}

// VaultTree e a conta unica do conjunto de pastas do cofre.
//
// As pastas sao derivadas dos caminhos das notas do indice, e nunca do disco:
// o indice ja respeita as exclusoes de vault.Walk (.obsidian, .trash), e uma
// varredura paralela do sistema de arquivos divergiria dele no primeiro caso
// que so uma das duas tratasse. A consequencia e que uma pasta sem nenhuma
// nota na subarvore — vazia, ou so com anexos — nao existe aqui.
//
// Sem teto, ao contrario de ListNotes: quem publica os resources precisa do
// cofre inteiro, e o teto de note_list e contrato de tool, nao desta conta.
//
// Nao recebe ctx util: le so o indice em memoria.
func (s *Service) VaultTree(_ context.Context, req TreeRequest) (TreeResult, error) {
	if s.index == nil {
		return TreeResult{}, Errorf(CodeVaultUnavailable, "índice indisponível")
	}

	notes, _ := s.index.List(index.Query{Sort: "path"})

	pastas := map[string]*FolderItem{"": {}}
	arvore := make([]TreeNote, 0, len(notes))
	for _, n := range notes {
		p := string(n.Path)
		dir := pastaDe(p)
		arvore = append(arvore, TreeNote{
			Path:     p,
			Folder:   dir,
			Title:    n.Title,
			Size:     n.Size,
			Modified: n.ModTime,
		})

		if f := garantirPasta(pastas, dir); f != nil {
			f.Notes++
		}
		for d := dir; ; d = pastaDe(d) {
			pastas[d].NotesRecursive++
			if d == "" {
				break
			}
		}
	}

	alvo, err := acharPasta(pastas, req.Folder)
	if err != nil {
		return TreeResult{}, err
	}

	res := TreeResult{Folder: *pastas[alvo]}
	dentro := func(dir string) bool {
		if !req.Recursive {
			return dir == alvo
		}
		return alvo == "" || dir == alvo || strings.HasPrefix(dir, alvo+"/")
	}
	for _, f := range pastas {
		if f.Path != alvo && dentro(f.Parent) {
			res.Folders = append(res.Folders, *f)
		}
	}
	slices.SortFunc(res.Folders, func(a, b FolderItem) int { return strings.Compare(a.Path, b.Path) })
	for _, n := range arvore {
		if dentro(n.Folder) {
			res.Notes = append(res.Notes, n)
		}
	}
	return res, nil
}

// pastaDe devolve a pasta de um caminho canonico, com "" para a raiz.
func pastaDe(p string) string {
	d := path.Dir(p)
	if d == "." || d == "/" {
		return ""
	}
	return d
}

// garantirPasta registra a pasta e todas as ancestrais que ainda nao estao no
// mapa, e devolve a pasta pedida.
func garantirPasta(pastas map[string]*FolderItem, dir string) *FolderItem {
	for d := dir; ; d = pastaDe(d) {
		if _, ok := pastas[d]; ok {
			break
		}
		pastas[d] = &FolderItem{Path: d, Parent: pastaDe(d)}
	}
	return pastas[dir]
}

// acharPasta casa a pasta pedida pela chave de caminho do indice: caixa, forma
// Unicode e barra final nao separam. A grafia exata vence; se nao houver, e a
// chave casar mais de uma pasta (so possivel num sistema de arquivos que
// distingue caixa), a resposta e ambigua em vez de uma escolha silenciosa.
func acharPasta(pastas map[string]*FolderItem, pedido string) (string, error) {
	pedido = strings.Trim(strings.ReplaceAll(pedido, "\\", "/"), "/")
	if _, ok := pastas[pedido]; ok {
		return pedido, nil
	}

	chave := text.ChaveDeCaminho(pedido)
	var casam []string
	for p := range pastas {
		if text.ChaveDeCaminho(p) == chave {
			casam = append(casam, p)
		}
	}
	switch len(casam) {
	case 1:
		return casam[0], nil
	case 0:
		return "", Errorf(CodeFolderNotFound, "pasta %q não tem nenhuma nota no cofre", pedido)
	}
	slices.Sort(casam)
	return "", Errorf(CodeAmbiguousPath, "pasta %q casa mais de uma pasta: %s", pedido, strings.Join(casam, ", "))
}
