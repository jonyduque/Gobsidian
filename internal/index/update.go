package index

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/jonyd/gobsidian/internal/parser"
	"github.com/jonyd/gobsidian/internal/vault"
)

// Replace reindexa um unico caminho: remove as contribuicoes antigas, le e
// parseia de novo, reinsere, e reprocessa os links afetados.
//
// E o ponto de chegada do watcher. Arquivo que sumiu entre o evento e o
// Stat nao e erro: a nota sai do indice e os links para ela passam a
// quebrados.
func (ix *Index) Replace(ctx context.Context, v *vault.Vault, path vault.CanonicalPath) error {
	ix.mu.Lock()
	defer ix.mu.Unlock()

	ix.removeContributionsLocked(path)

	atomic.AddUint64(&ix.generation, 1)

	abs := v.Abs(path)
	info, err := os.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			// Nota sumiu entre o evento e o Stat. Reprocessar todos os
			// links e o que marca como quebrados os que apontavam para ela.
			ix.reprocessLinksLocked()
			return nil
		}
		return err
	}

	isNote := strings.HasSuffix(strings.ToLower(abs), ".md")
	entry := vault.Entry{
		Path:      path,
		Size:      info.Size(),
		ModTime:   info.ModTime(),
		IsNote:    isNote,
		CloudOnly: vault.IsCloudOnly(abs),
	}

	if !entry.IsNote || entry.CloudOnly {
		a := &Asset{
			Path:    entry.Path,
			Size:    entry.Size,
			ModTime: entry.ModTime,
		}
		ix.assets[entry.Path] = a
		lower := strings.ToLower(string(entry.Path))
		ix.lowerPath[lower] = entry.Path
		base := vault.CanonicalPath(filepath.ToSlash(filepath.Base(string(entry.Path))))
		ix.byName[string(base)] = append(ix.byName[string(base)], entry.Path)
		ix.reprocessLinksLocked()
		return nil
	}

	data, err := v.ReadAll(ctx, entry.Path)
	if err != nil {
		return err
	}

	body, hadBOM := vault.StripBOM(data)
	note, err := parser.Parse(body)
	if err != nil {
		return err
	}
	if hadBOM {
		note.ShiftOffsets(int64(vault.BOMLen))
	}

	n := &Note{
		Path:        entry.Path,
		Title:       note.Title,
		Size:        entry.Size,
		ModTime:     entry.ModTime,
		Hash:        xxhash.Sum64(data),
		EOL:         vault.DetectEOL(data),
		BOM:         hadBOM,
		CloudOnly:   entry.CloudOnly,
		Frontmatter: note.Frontmatter,
		Tags:        note.Tags,
		Aliases:     note.Aliases,
		Headings:    note.Headings,
		Blocks:      note.Blocks,
		Inline:      note.Inline,
	}

	for _, l := range note.Links {
		n.Links = append(n.Links, ResolvedLink{Link: l})
	}
	ix.notes[entry.Path] = n

	lower := strings.ToLower(string(entry.Path))
	ix.lowerPath[lower] = entry.Path

	base := vault.CanonicalPath(filepath.ToSlash(filepath.Base(string(entry.Path))))
	ix.byName[string(base)] = append(ix.byName[string(base)], entry.Path)

	for _, alias := range note.Aliases {
		key := aliasKey(alias)
		ix.byAlias[key] = append(ix.byAlias[key], entry.Path)
	}
	for _, tag := range note.Tags {
		ix.tags[tag] = append(ix.tags[tag], entry.Path)
	}

	// Resolve os links da nota inserida
	ix.resolveLinksForNoteLocked(n)

	// Adiciona backlinks da nota inserida
	for _, l := range n.Links {
		if l.Resolved != "" {
			bl := Backlink{
				From:    path,
				Anchor:  l.Anchor,
				Alias:   l.Alias,
				Context: "",
				Kind:    l.Kind,
			}
			ix.backlinks[l.Resolved] = append(ix.backlinks[l.Resolved], bl)
		}
	}

	// Um alvo recem-criado pode fazer links passarem a resolver
	ix.reprocessLinksLocked()
	return nil
}

// Remove tira o caminho do indice e marca como quebrados os links que
// apontavam para ele.
func (ix *Index) Remove(path vault.CanonicalPath) {
	ix.mu.Lock()
	defer ix.mu.Unlock()

	ix.removeContributionsLocked(path)
	atomic.AddUint64(&ix.generation, 1)

	// Reprocessar todos os links e o que marca como quebrados os que
	// apontavam para a nota removida.
	ix.reprocessLinksLocked()
}

func (ix *Index) removeContributionsLocked(path vault.CanonicalPath) {
	oldNote, hadNote := ix.notes[path]
	if hadNote {
		for _, l := range oldNote.Links {
			if l.Resolved != "" {
				bls := ix.backlinks[l.Resolved]
				filtered := make([]Backlink, 0, len(bls))
				for _, bl := range bls {
					if bl.From != path {
						filtered = append(filtered, bl)
					}
				}
				if len(filtered) == 0 {
					delete(ix.backlinks, l.Resolved)
				} else {
					ix.backlinks[l.Resolved] = filtered
				}
			}
		}

		for _, tag := range oldNote.Tags {
			paths := ix.tags[tag]
			filtered := make([]vault.CanonicalPath, 0, len(paths))
			for _, p := range paths {
				if p != path {
					filtered = append(filtered, p)
				}
			}
			if len(filtered) == 0 {
				delete(ix.tags, tag)
			} else {
				ix.tags[tag] = filtered
			}
		}

		lower := strings.ToLower(string(path))
		delete(ix.lowerPath, lower)

		base := vault.CanonicalPath(filepath.ToSlash(filepath.Base(string(path))))
		names := ix.byName[string(base)]
		filteredNames := make([]vault.CanonicalPath, 0, len(names))
		for _, p := range names {
			if p != path {
				filteredNames = append(filteredNames, p)
			}
		}
		if len(filteredNames) == 0 {
			delete(ix.byName, string(base))
		} else {
			ix.byName[string(base)] = filteredNames
		}

		for _, alias := range oldNote.Aliases {
			key := aliasKey(alias)
			al := ix.byAlias[key]
			filteredAl := make([]vault.CanonicalPath, 0, len(al))
			for _, p := range al {
				if p != path {
					filteredAl = append(filteredAl, p)
				}
			}
			if len(filteredAl) == 0 {
				delete(ix.byAlias, key)
			} else {
				ix.byAlias[key] = filteredAl
			}
		}
	}

	_, hadAsset := ix.assets[path]
	if hadAsset {
		lower := strings.ToLower(string(path))
		delete(ix.lowerPath, lower)

		base := vault.CanonicalPath(filepath.ToSlash(filepath.Base(string(path))))
		names := ix.byName[string(base)]
		filteredNames := make([]vault.CanonicalPath, 0, len(names))
		for _, p := range names {
			if p != path {
				filteredNames = append(filteredNames, p)
			}
		}
		if len(filteredNames) == 0 {
			delete(ix.byName, string(base))
		} else {
			ix.byName[string(base)] = filteredNames
		}
	}

	delete(ix.notes, path)
	delete(ix.assets, path)
}

func (ix *Index) resolveLinksForNoteLocked(n *Note) {
	for i := range n.Links {
		resolved, via, state := ix.resolveTarget(n.Links[i].Target, n.Path)
		n.Links[i].Resolved = resolved
		n.Links[i].Via = via
		n.Links[i].State = state

		if state == LinkOK {
			ix.resolveAnchor(&n.Links[i])
		}
	}
}

// mutarNotaLocked devolve a nota de path pronta para escrita, publicando uma
// COPIA no lugar da entrada atual na primeira chamada e devolvendo a mesma
// copia nas seguintes.
//
// Existe por causa de uma corrida de dados real, encontrada sob -race por
// TestE2E_NoteMoveIsReflectedBySearchAndGraph: index.MoveNote escrevia
// n.Path enquanto service.Search lia note.Path. O mutex do indice nao cobria,
// e nao e bug de trava esquecida — MoveNote segura ix.mu.Lock() do inicio ao
// fim. O mutex protege o MAPA; o *Note para onde a entrada aponta escapa por
// Get e por List, e o chamador le os campos DEPOIS de soltar o RLock. Mutar o
// objeto no lugar e escrever no que outra goroutine esta lendo, com trava ou
// sem.
//
// Dai a invariante: **um *Note publicado em ix.notes e imutavel**. Quem muda
// uma nota troca a entrada do mapa por uma copia. Build e Replace ja
// obedeciam sem que ninguem tivesse escrito a regra — montam Note novo e so
// entao publicam. Quem nao obedecia era MoveNote e reprocessLinksLocked, e
// esta ultima roda em TODO evento do watcher, sobre TODAS as notas: era a
// corrida mais larga das duas.
//
// Toda escrita em nota publicada passa por aqui. Nao e para consertar as duas
// que estavam erradas: e para a proxima nao ter onde nascer.
//
// A copia e preguicosa de proposito. reprocessLinksLocked visita as N notas
// do cofre a cada evento, e quase nenhuma muda; copiar todas trocaria uma
// corrida por lixo proporcional ao cofre a cada salvamento de arquivo.
func (ix *Index) mutarNotaLocked(path vault.CanonicalPath, copia **Note) *Note {
	if *copia != nil {
		return *copia
	}
	atual := ix.notes[path]
	c := *atual
	// Links tem de ser fatia propria: a copia rasa compartilha o array de
	// apoio, e escrever em c.Links[i] escreveria no do original.
	c.Links = append([]ResolvedLink(nil), atual.Links...)
	*copia = &c
	ix.notes[path] = &c
	return &c
}

func (ix *Index) reprocessLinksLocked() {
	// Reavalia todos os links em todas as notas.
	// E mais simples e garante consistencia.
	//
	// Escrever na entrada do mapa durante o range e legal em Go quando a chave
	// ja existe — nenhuma chave e criada aqui.
	for path, n := range ix.notes {
		var copia *Note

		for i := range n.Links {
			resolved, via, state := ix.resolveTarget(n.Links[i].Target, n.Path)

			oldResolved := n.Links[i].Resolved

			// Monta o link novo fora do indice e so publica se ele mudou.
			// ResolvedLink e comparavel — parser.Link tem so escalares e
			// strings — entao a comparacao e a propria condicao de copia.
			novo := n.Links[i]
			novo.Resolved = resolved
			novo.Via = via
			novo.State = state

			if state == LinkOK {
				ix.resolveAnchor(&novo)
			}

			if novo != n.Links[i] {
				ix.mutarNotaLocked(path, &copia).Links[i] = novo
			}

			// Atualizar backlinks se o alvo mudou
			if oldResolved != resolved {
				if oldResolved != "" {
					bls := ix.backlinks[oldResolved]
					filtered := make([]Backlink, 0, len(bls))
					for _, bl := range bls {
						if bl.From != n.Path {
							filtered = append(filtered, bl)
						}
					}
					if len(filtered) == 0 {
						delete(ix.backlinks, oldResolved)
					} else {
						ix.backlinks[oldResolved] = filtered
					}
				}

				if resolved != "" {
					bl := Backlink{
						From:    n.Path,
						Anchor:  n.Links[i].Anchor,
						Alias:   n.Links[i].Alias,
						Context: "",
						Kind:    n.Links[i].Kind,
					}
					ix.backlinks[resolved] = append(ix.backlinks[resolved], bl)
				}
			}
		}
	}
}

// MoveNote atualiza os caminhos de uma nota no índice sem precisar reler do disco.
func (ix *Index) MoveNote(v *vault.Vault, oldPath, newPath vault.CanonicalPath) {
	ix.mu.Lock()
	defer ix.mu.Unlock()

	original, hasNote := ix.notes[oldPath]
	if !hasNote {
		return
	}

	atomic.AddUint64(&ix.generation, 1)

	// 1. Atualizar notas.
	//
	// Copia, nao mutacao no lugar: ver mutarNotaLocked. Era exatamente aqui
	// que o detector de corrida apontava — `n.Path = newPath` contra
	// service.Search lendo note.Path fora do RLock.
	//
	// A copia entra no mapa sob newPath; a entrada de oldPath e apagada logo
	// abaixo. Quem ja segurava o ponteiro antigo continua lendo a nota antiga,
	// coerente, ate largar — que e o contrato de qualquer leitura sem trava.
	movida := *original
	movida.Links = append([]ResolvedLink(nil), original.Links...)
	movida.Path = newPath
	n := &movida

	var info os.FileInfo
	var err error
	if v != nil {
		info, err = os.Stat(v.Abs(newPath))
	} else {
		info, err = os.Stat(string(newPath))
	}
	if err == nil {
		n.ModTime = info.ModTime()
		n.Size = info.Size()
	} else {
		n.ModTime = time.Time{}
		slog.Debug("MoveNote stat failed, zeroing ModTime to force reindex", "path", newPath, "err", err)
	}

	ix.notes[newPath] = n
	delete(ix.notes, oldPath)

	// 2. Atualizar lowerPath
	delete(ix.lowerPath, strings.ToLower(string(oldPath)))
	ix.lowerPath[strings.ToLower(string(newPath))] = newPath

	// 3. Atualizar byName
	oldBase := vault.CanonicalPath(filepath.ToSlash(filepath.Base(string(oldPath))))
	names := ix.byName[string(oldBase)]
	filteredNames := make([]vault.CanonicalPath, 0, len(names))
	for _, p := range names {
		if p != oldPath {
			filteredNames = append(filteredNames, p)
		}
	}
	if len(filteredNames) == 0 {
		delete(ix.byName, string(oldBase))
	} else {
		ix.byName[string(oldBase)] = filteredNames
	}
	newBase := vault.CanonicalPath(filepath.ToSlash(filepath.Base(string(newPath))))
	ix.byName[string(newBase)] = append(ix.byName[string(newBase)], newPath)

	// 4. Atualizar tags
	for _, tag := range n.Tags {
		paths := ix.tags[tag]
		for i, p := range paths {
			if p == oldPath {
				paths[i] = newPath
			}
		}
	}

	// 5. Atualizar byAlias
	for _, alias := range n.Aliases {
		key := aliasKey(alias)
		al := ix.byAlias[key]
		for i, p := range al {
			if p == oldPath {
				al[i] = newPath
			}
		}
	}

	// 6. Atualizar incoming backlinks e a resolução nas notas que apontavam para cá
	if bls, ok := ix.backlinks[oldPath]; ok {
		ix.backlinks[newPath] = bls
		delete(ix.backlinks, oldPath)
		for _, bl := range bls {
			if srcNote, ok := ix.notes[bl.From]; ok {
				// Mesma invariante: a nota que APONTA para a movida tambem
				// esta publicada, e mudar Resolved nela no lugar corre contra
				// quem estiver lendo o grafo.
				var copia *Note
				for i := range srcNote.Links {
					if srcNote.Links[i].Resolved == oldPath {
						ix.mutarNotaLocked(bl.From, &copia).Links[i].Resolved = newPath
					}
				}
			}
		}
	}

	// 7. Atualizar outgoing backlinks
	for _, l := range n.Links {
		if l.Resolved != "" {
			bls := ix.backlinks[l.Resolved]
			for i, bl := range bls {
				if bl.From == oldPath {
					bls[i].From = newPath
				}
			}
		}
	}

	// 8. Reprocessar os links garante que links recém-quebrados ou recém-resolvidos sejam notados
	ix.reprocessLinksLocked()
}
