package index

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/cespare/xxhash/v2"
	"github.com/jonyd/gobsidian/internal/parser"
	"github.com/jonyd/gobsidian/internal/vault"
)

func (ix *Index) Replace(ctx context.Context, v *vault.Vault, path vault.CanonicalPath) error {
	ix.mu.Lock()
	defer ix.mu.Unlock()

	ix.removeContributionsLocked(path)

	atomic.AddUint64(&ix.generation, 1)

	abs := v.Abs(path)
	info, err := os.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			ix.markLinksToTargetMissingLocked(path)
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
		ix.byAlias[alias] = append(ix.byAlias[alias], entry.Path)
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

func (ix *Index) Remove(path vault.CanonicalPath) {
	ix.mu.Lock()
	defer ix.mu.Unlock()

	ix.removeContributionsLocked(path)
	atomic.AddUint64(&ix.generation, 1)

	ix.markLinksToTargetMissingLocked(path)
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
			al := ix.byAlias[alias]
			filteredAl := make([]vault.CanonicalPath, 0, len(al))
			for _, p := range al {
				if p != path {
					filteredAl = append(filteredAl, p)
				}
			}
			if len(filteredAl) == 0 {
				delete(ix.byAlias, alias)
			} else {
				ix.byAlias[alias] = filteredAl
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

func (ix *Index) reprocessLinksLocked() {
	// Reavalia todos os links em todas as notas.
	// E mais simples e garante consistencia.
	for _, n := range ix.notes {
		for i := range n.Links {
			resolved, via, state := ix.resolveTarget(n.Links[i].Target, n.Path)

			oldResolved := n.Links[i].Resolved

			n.Links[i].Resolved = resolved
			n.Links[i].Via = via
			n.Links[i].State = state

			if state == LinkOK {
				ix.resolveAnchor(&n.Links[i])
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

func (ix *Index) markLinksToTargetMissingLocked(path vault.CanonicalPath) {
	// Links que apontavam para path agora estao quebrados.
	// reprocessLinksLocked() cuida disso se iterarmos todas as notas.
	ix.reprocessLinksLocked()
}
