package index

import (
	"cmp"
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
//
// A tag devolvida e a CHAVE, ja dobrada por ChaveDeTag — minuscula, NFC e sem
// '#'. Duas grafias que so diferem em caixa ou em forma Unicode sao uma
// entrada so, com a soma das contagens. Ate a Task 180 a chave era a grafia
// crua da nota e o prefixo era comparado com um ToLower de cada lado a cada
// leitura: a mesma tag aparecia duas vezes na lista.
func (ix *Index) Tags(prefix string, minCount int) []TagCount {
	ix.mu.RLock()
	defer ix.mu.RUnlock()

	var result []TagCount
	prefix = ChaveDeTag(prefix)

	for t, paths := range ix.tags {
		if len(paths) >= minCount && strings.HasPrefix(t, prefix) {
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
	// O LOCK cobre so a coleta. A ordenacao e a paginacao rodam fora dele.
	//
	// Ate 2026-08-28 o RLock ia da primeira linha ao return, entao o sort — que
	// e O(n log n) e o trecho mais caro da funcao — rodava com o indice travado
	// para escrita, prolongando a retencao contra o watcher (achado P4).
	//
	// E seguro porque os `*Note` sao publicados por COPIA: um ponteiro obtido
	// sob o RLock continua apontando para uma nota coerente depois de solta-lo,
	// e e a mesma garantia de que service.Search ja depende ao ler note.Path
	// fora do lock.
	notes := ix.coletarLocked(q)

	sortAndPaginate(notes, q)
	total := len(notes)
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

// candidatosPorTagLocked e o passo 1 de coletarLocked e o corpo de
// PathsComTags. Exige ix.mu ja travado para leitura.
//
// A regra e hierarquica: a tag pedida casa a si mesma e qualquer subtag —
// "projeto" casa "projeto/x". As duas pontas passam por ChaveDeTag, entao
// caixa, forma Unicode e '#' inicial nao separam o que e a mesma tag.
//
// mode vazio vale "all": e o default de note_list, e o `!= "any"` abaixo o
// cobre sem um segundo lugar que decida isso.
func (ix *Index) candidatosPorTagLocked(tags []string, mode string) []vault.CanonicalPath {
	// "any" acumula direto no destino e ordena UMA vez no fim; "all" precisa de
	// cada conjunto ordenado a parte, porque a intersecao e por busca binaria.
	//
	// Por isso o casamento anexa a uma fatia do chamador em vez de devolver uma
	// pronta: a versao que sempre devolvia ordenada e compactada fazia "any"
	// pagar um sort e uma copia por tag para depois ordenar tudo de novo.
	// Medido em BenchmarkListPorTag (3.000 notas, uma tag, tag_mode=any):
	// +16,08% de tempo (p=0,026, n=7) e +40,31% de B/op (p=0,001) contra o
	// codigo anterior a esta Task.
	casam := func(pedida string, dst []vault.CanonicalPath) []vault.CanonicalPath {
		tk := ChaveDeTag(pedida)
		for k, v := range ix.tags {
			if k == tk || strings.HasPrefix(k, tk+"/") {
				dst = append(dst, v...)
			}
		}
		return dst
	}
	ordenado := func(pedida string) []vault.CanonicalPath {
		m := casam(pedida, nil)
		slices.Sort(m)
		return slices.Compact(m)
	}
	if strings.ToLower(mode) == "any" {
		var todos []vault.CanonicalPath
		for _, t := range tags {
			todos = casam(t, todos)
		}
		slices.Sort(todos)
		return slices.Compact(todos)
	}
	var cand []vault.CanonicalPath
	for i, t := range tags {
		m := ordenado(t)
		if i == 0 {
			cand = m
			continue
		}
		cand = slices.DeleteFunc(cand, func(c vault.CanonicalPath) bool {
			_, ok := slices.BinarySearch(m, c)
			return !ok
		})
		if len(cand) == 0 {
			break
		}
	}
	return cand
}

// PathsComTags devolve, ordenados, os caminhos das notas que casam as tags em
// mode ("all" | "any"), com a regra hierarquica. Nil para len(tags) == 0 —
// "sem filtro de tag", que nao e a mesma resposta que "nenhuma nota casou".
//
// Existe para que vault_search resolva o filtro de tag UMA vez por consulta,
// pela mesma conta que note_list ja usava, em vez de re-derivar a comparacao
// por resultado.
func (ix *Index) PathsComTags(tags []string, mode string) []vault.CanonicalPath {
	if len(tags) == 0 {
		return nil
	}
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	return ix.candidatosPorTagLocked(tags, mode)
}

// coletarLocked reune os candidatos que passam pelos filtros. Toma o RLock e o
// solta ao devolver: nada aqui ordena nem pagina.
func (ix *Index) coletarLocked(q Query) []*Note {
	ix.mu.RLock()
	defer ix.mu.RUnlock()

	var candidates []vault.CanonicalPath

	// 1. Tags
	if len(q.Tags) > 0 {
		candidates = ix.candidatosPorTagLocked(q.Tags, q.TagMode)
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
	return notes
}

// sortAndPaginate ordena a lista fora do lock do indice.
func sortAndPaginate(notes []*Note, q Query) {
	if q.Sort != "" {
		order := strings.ToLower(q.Order)
		// ICADO do comparador (achado P4). `q.Order` ja estava fora; `q.Sort`
		// nao, e ToLower alocava e percorria a string uma vez por COMPARACAO —
		// O(n log n) vezes por consulta, sempre com o mesmo resultado.
		criterio := strings.ToLower(q.Sort)
		slices.SortStableFunc(notes, func(a, b *Note) int {
			ordem := 0
			switch criterio {
			case "title":
				ordem = strings.Compare(a.Title, b.Title)
			case "path":
				ordem = strings.Compare(string(a.Path), string(b.Path))
			case "modified":
				ordem = a.ModTime.Compare(b.ModTime)
			case "size":
				// cmp.Compare, e nao int(a.Size-b.Size): a subtracao transborda
				// em build 32-bit e devolve a ordem invertida em silencio.
				ordem = cmp.Compare(a.Size, b.Size)
			}
			if ordem == 0 {
				ordem = strings.Compare(string(a.Path), string(b.Path))
			}
			if order == "desc" {
				return -ordem
			}
			return ordem
		})
	} else {
		slices.SortStableFunc(notes, func(a, b *Note) int {
			return strings.Compare(string(a.Path), string(b.Path))
		})
	}

}
