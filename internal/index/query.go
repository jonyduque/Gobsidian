package index

import (
	"path"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/jonyd/gobsidian/internal/text"
	"github.com/jonyd/gobsidian/internal/vault"
)

// Query e o filtro de note_list: pasta, tags, campos de frontmatter,
// ordenacao e limite. Os criterios se combinam por E.
type Query struct {
	Folder         string
	Glob           string
	Tags           []string
	TagMode        string // "all" ou "any"
	Frontmatter    map[string]any
	ModifiedAfter  *time.Time
	ModifiedBefore *time.Time
	Recursive      bool
	Sort           string
	Order          string
	Limit          int
	Offset         int
}

// TagCount e uma tag com o numero de notas que a usam.
type TagCount struct {
	Tag   string
	Count int
}

func normalizeString(s string) string {
	return text.Normalize(s)
}

func getNestedValue(m map[string]any, key string) (any, bool) {
	parts := strings.Split(key, ".")
	var current any = m
	for i, part := range parts {
		if curMap, ok := current.(map[string]any); ok {
			val, exists := curMap[part]
			if !exists {
				return nil, false
			}
			if i == len(parts)-1 {
				return val, true
			}
			current = val
		} else {
			return nil, false
		}
	}
	return nil, false
}

func matchFrontmatterValue(val any, expected any) bool {
	if expected == nil {
		return true // presenca casou
	}

	switch e := expected.(type) {
	case []any: // expected is list, we need ALL to match
		vSlice, ok := val.([]any)
		if !ok {
			return false
		}
		for _, eItem := range e {
			itemMatched := false
			for _, vItem := range vSlice {
				if matchFrontmatterValue(vItem, eItem) {
					itemMatched = true
					break
				}
			}
			if !itemMatched {
				return false
			}
		}
		return true

	case string:
		eNorm := normalizeString(e)
		switch v := val.(type) {
		case string:
			return normalizeString(v) == eNorm
		case []any:
			for _, item := range v {
				if strItem, ok := item.(string); ok {
					if normalizeString(strItem) == eNorm {
						return true
					}
				}
			}
		}

	case time.Time:
		switch v := val.(type) {
		case time.Time:
			return v.Equal(e)
		case string:
			return v == e.Format("2006-01-02") || v == e.Format(time.RFC3339)
		}

	default: // Outros tipos (int, bool)
		switch v := val.(type) {
		case []any:
			for _, item := range v {
				if reflect.DeepEqual(item, e) {
					return true
				}
			}
		default:
			return reflect.DeepEqual(v, e)
		}
	}
	return false
}

// NotePaths devolve apenas caminhos de NOTA.
//
// Paths() devolve notas e anexos, mas Get() so resolve notas — quem iterar um
// e chamar o outro estoura em qualquer cofre com anexo. Foi o que aconteceu:
// acrescentar um unico .png ao cofre de TestBacklinkInvariantUnderMutation
// derrubava o teste com desreferencia de ponteiro nulo, e ele so passava
// porque a fixture nao tinha anexo.
//
// Quem quer percorrer notas usa esta. Paths() continua existindo para quem
// precisa do conjunto inteiro, e agora tem um par que diz o que faz.
func (ix *Index) NotePaths() []vault.CanonicalPath {
	ix.mu.RLock()
	defer ix.mu.RUnlock()

	paths := make([]vault.CanonicalPath, 0, len(ix.notes))
	for p := range ix.notes {
		paths = append(paths, p)
	}
	slices.Sort(paths)
	return paths
}

// AliasCollisions conta aliases declarados por mais de uma nota.
//
// Colisao de alias nao tem resposta correta: a resolucao escolhe pelo mesmo
// desempate de proximidade dos homonimos, mas o autor provavelmente nao quis
// que dois arquivos respondessem pelo mesmo nome. Contar e o que torna isso
// diagnosticavel em vez de silencioso.
//
// Antes disto o campo alias_collisions de vault_stats era zero literal no
// codigo — aparecia na resposta e mentia sempre, o que e pior que nao existir,
// porque quem le acredita.
func (ix *Index) AliasCollisions() int {
	ix.mu.RLock()
	defer ix.mu.RUnlock()

	n := 0
	for _, paths := range ix.byAlias {
		if len(paths) > 1 {
			n++
		}
	}
	return n
}

// Tags devolve as tags do cofre com suas contagens, filtradas por prefixo e
// por contagem minima.
func (ix *Index) Tags(prefix string, minCount int) []TagCount {
	ix.mu.RLock()
	defer ix.mu.RUnlock()

	var result []TagCount
	prefix = strings.ToLower(prefix)

	for t, paths := range ix.tags {
		if len(paths) >= minCount && strings.HasPrefix(strings.ToLower(t), prefix) {
			result = append(result, TagCount{Tag: t, Count: len(paths)})
		}
	}
	slices.SortFunc(result, func(a, b TagCount) int {
		if a.Count != b.Count {
			return b.Count - a.Count
		}
		return strings.Compare(a.Tag, b.Tag)
	})
	return result
}

// List aplica a consulta e devolve as notas junto do total ANTES do limite —
// sem esse total o cliente nao sabe que existe mais do que ele recebeu.
func (ix *Index) List(q Query) ([]*Note, int) {
	ix.mu.RLock()
	defer ix.mu.RUnlock()

	var candidates []vault.CanonicalPath

	// 1. Tags
	if len(q.Tags) > 0 {
		tagMode := strings.ToLower(q.TagMode)
		if tagMode == "" {
			tagMode = "all"
		}

		if tagMode == "all" {
			var first = true
			for _, t := range q.Tags {
				tLower := strings.ToLower(t)
				var matches []vault.CanonicalPath
				for k, v := range ix.tags {
					kLower := strings.ToLower(k)
					if kLower == tLower || strings.HasPrefix(kLower, tLower+"/") {
						matches = append(matches, v...)
					}
				}

				// Deduplicate matches for this tag
				slices.Sort(matches)
				matches = slices.Compact(matches)

				if first {
					candidates = matches
					first = false
				} else {
					var intersection []vault.CanonicalPath
					for _, c := range candidates {
						if _, found := slices.BinarySearch(matches, c); found {
							intersection = append(intersection, c)
						}
					}
					candidates = intersection
				}
				if len(candidates) == 0 {
					break
				}
			}
		} else { // "any"
			var allMatches []vault.CanonicalPath
			for _, t := range q.Tags {
				tLower := strings.ToLower(t)
				for k, v := range ix.tags {
					kLower := strings.ToLower(k)
					if kLower == tLower || strings.HasPrefix(kLower, tLower+"/") {
						allMatches = append(allMatches, v...)
					}
				}
			}
			slices.Sort(allMatches)
			candidates = slices.Compact(allMatches)
		}
	} else {
		for p := range ix.notes {
			candidates = append(candidates, p)
		}
	}

	// 2. Folder
	if q.Folder != "" {
		folderPath := filepath.ToSlash(q.Folder)
		if folderPath != "" && !strings.HasSuffix(folderPath, "/") {
			folderPath += "/"
		}
		var filtered []vault.CanonicalPath
		for _, c := range candidates {
			cStr := string(c)
			if q.Recursive {
				if folderPath == "/" || strings.HasPrefix(cStr, folderPath) {
					filtered = append(filtered, c)
				}
			} else {
				if folderPath == "/" {
					if !strings.Contains(cStr, "/") {
						filtered = append(filtered, c)
					}
				} else {
					if strings.HasPrefix(cStr, folderPath) {
						rest := cStr[len(folderPath):]
						if !strings.Contains(rest, "/") {
							filtered = append(filtered, c)
						}
					}
				}
			}
		}
		candidates = filtered
	}

	// 3. Glob
	if q.Glob != "" {
		var filtered []vault.CanonicalPath
		for _, c := range candidates {
			match, _ := path.Match(q.Glob, string(c))
			if match {
				filtered = append(filtered, c)
			}
		}
		candidates = filtered
	}

	// 4. Frontmatter
	if len(q.Frontmatter) > 0 {
		var filtered []vault.CanonicalPath
		for _, c := range candidates {
			note := ix.notes[c]
			matchesAll := true
			for k, expected := range q.Frontmatter {
				val, exists := getNestedValue(note.Frontmatter, k)
				if !exists {
					if expected != nil {
						matchesAll = false
						break
					}
					// Campo ausente nunca casa, nem quando o valor pedido e
					// null. A tabela de docs/TOOLS.md diz que null casa com a
					// PRESENCA da chave com qualquer valor — ausencia nao e um
					// valor, e um pedido por "existe com qualquer valor" nao
					// pode ser satisfeito por "nao existe".
					matchesAll = false
					break
				}
				if !matchFrontmatterValue(val, expected) {
					matchesAll = false
					break
				}
			}
			if matchesAll {
				filtered = append(filtered, c)
			}
		}
		candidates = filtered
	}

	// 5. ModifiedAfter & ModifiedBefore
	if q.ModifiedAfter != nil || q.ModifiedBefore != nil {
		var filtered []vault.CanonicalPath
		for _, c := range candidates {
			note := ix.notes[c]
			if q.ModifiedAfter != nil && note.ModTime.Before(*q.ModifiedAfter) {
				continue
			}
			if q.ModifiedBefore != nil && note.ModTime.After(*q.ModifiedBefore) {
				continue
			}
			filtered = append(filtered, c)
		}
		candidates = filtered
	}

	var notes []*Note
	for _, c := range candidates {
		notes = append(notes, ix.notes[c])
	}

	// Sort
	if q.Sort != "" {
		order := strings.ToLower(q.Order)
		slices.SortStableFunc(notes, func(a, b *Note) int {
			cmp := 0
			switch strings.ToLower(q.Sort) {
			case "title":
				cmp = strings.Compare(a.Title, b.Title)
			case "path":
				cmp = strings.Compare(string(a.Path), string(b.Path))
			case "modified":
				cmp = a.ModTime.Compare(b.ModTime)
			case "size":
				cmp = int(a.Size - b.Size)
			}
			if cmp == 0 {
				cmp = strings.Compare(string(a.Path), string(b.Path))
			}
			if order == "desc" {
				return -cmp
			}
			return cmp
		})
	} else {
		slices.SortStableFunc(notes, func(a, b *Note) int {
			return strings.Compare(string(a.Path), string(b.Path))
		})
	}

	total := len(notes)

	// Pagination
	if q.Offset > 0 {
		if q.Offset >= len(notes) {
			notes = nil
		} else {
			notes = notes[q.Offset:]
		}
	}
	if q.Limit > 0 && len(notes) > q.Limit {
		notes = notes[:q.Limit]
	}

	return notes, total
}
