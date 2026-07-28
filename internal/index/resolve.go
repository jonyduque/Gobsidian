package index

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/jonyd/gobsidian/internal/vault"
)

// ErrAmbiguousPath sinaliza que a entrada casou com mais de um arquivo e
// nenhuma regra de desempate decidiu. Devolver um dos candidatos em silencio
// seria pior: o cliente leria a nota errada acreditando ter lido a certa.
var ErrAmbiguousPath = errors.New("ambiguous path")

func (ix *Index) resolveAllLinks() {
	ix.mu.Lock()
	defer ix.mu.Unlock()

	for _, n := range ix.notes {
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
}

func (ix *Index) resolveTarget(target string, origin vault.CanonicalPath) (vault.CanonicalPath, ResolveVia, LinkState) {
	if target == "" {
		return "", ViaNone, LinkTargetMissing
	}

	// Alvo com esquema de URI nunca foi para o cofre. Precisa sair antes de
	// qualquer tentativa de resolucao, senao vira link quebrado e polui a
	// contagem de saude.
	if hasURIScheme(target) {
		return "", ViaNone, LinkExternal
	}

	// 1. Caminho explicito
	if strings.Contains(target, "/") {
		if path, ok := ix.resolveExplicit(target); ok {
			return path, ViaPath, LinkOK
		}
	}

	// 2. Nome de arquivo de nota
	if path, ok := ix.resolveByName(target, true, origin); ok {
		return path, ViaName, LinkOK
	}

	// 3. Nome de arquivo de anexo
	if path, ok := ix.resolveByName(target, false, origin); ok {
		return path, ViaAsset, LinkOK
	}

	// 4. Alias
	if path, ok := ix.resolveByAlias(target, origin); ok {
		return path, ViaAlias, LinkOK
	}

	return "", ViaNone, LinkTargetMissing
}

// hasURIScheme reconhece "esquema:" no inicio do alvo, conforme a RFC 3986:
// uma letra seguida de letras, digitos, "+", "-" ou ".", terminando em ":".
//
// Cobre http, https, mailto, ftp, obsidian e qualquer outro sem precisar de
// lista. Uma lista fixa envelheceria e deixaria passar exatamente o esquema
// que ninguem previu.
//
// Nao confunde com caminho do cofre: um alvo interno e relativo a raiz, e
// validateLocal ja rejeita forma enraizada antes de chegar aqui. E um nome de
// arquivo com dois-pontos nao passa, porque o esquema exige comecar por letra
// e nao conter barra antes do dois-pontos.
func hasURIScheme(target string) bool {
	for i := 0; i < len(target); i++ {
		c := target[i]
		switch {
		case c == ':':
			// Esquema vazio nao e esquema.
			return i > 0
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z':
			continue
		case c >= '0' && c <= '9', c == '+', c == '-', c == '.':
			// Digito e pontuacao so valem depois da primeira letra.
			if i == 0 {
				return false
			}
		default:
			return false
		}
	}
	return false
}

func (ix *Index) resolveExplicit(target string) (vault.CanonicalPath, bool) {
	p := vault.CanonicalPath(filepath.ToSlash(target))
	if _, ok := ix.notes[p]; ok {
		return p, true
	}
	if _, ok := ix.assets[p]; ok {
		return p, true
	}

	pMd := vault.CanonicalPath(filepath.ToSlash(target + ".md"))
	if _, ok := ix.notes[pMd]; ok {
		return pMd, true
	}

	return "", false
}

func (ix *Index) resolveByName(target string, isNote bool, origin vault.CanonicalPath) (vault.CanonicalPath, bool) {
	var name string
	if isNote {
		if strings.HasSuffix(target, ".md") {
			name = target
		} else {
			name = target + ".md"
		}
	} else {
		name = target // assets require explicit extension
	}

	paths, ok := ix.byName[name]
	if !ok || len(paths) == 0 {
		return "", false
	}

	var valid []vault.CanonicalPath
	for _, p := range paths {
		if isNote {
			if _, ok := ix.notes[p]; ok {
				valid = append(valid, p)
			}
		} else {
			if _, ok := ix.assets[p]; ok {
				valid = append(valid, p)
			}
		}
	}

	if len(valid) == 0 {
		return "", false
	}
	if len(valid) == 1 {
		return valid[0], true
	}

	return ix.tiebreak(valid, origin), true
}

func (ix *Index) resolveByAlias(target string, origin vault.CanonicalPath) (vault.CanonicalPath, bool) {
	lower := strings.ToLower(target)
	paths, ok := ix.byAlias[lower]
	if !ok || len(paths) == 0 {
		return "", false
	}

	if len(paths) == 1 {
		return paths[0], true
	}

	return ix.tiebreak(paths, origin), true
}

func (ix *Index) tiebreak(candidates []vault.CanonicalPath, origin vault.CanonicalPath) vault.CanonicalPath {
	originDir := filepath.ToSlash(filepath.Dir(string(origin)))
	originParts := strings.Split(originDir, "/")
	if originDir == "." {
		originParts = nil
	}

	var best vault.CanonicalPath
	bestScore := -1

	for _, p := range candidates {
		dir := filepath.ToSlash(filepath.Dir(string(p)))
		parts := strings.Split(dir, "/")
		if dir == "." {
			parts = nil
		}

		score := 0
		maxLen := len(parts)
		if len(originParts) < maxLen {
			maxLen = len(originParts)
		}
		for i := 0; i < maxLen; i++ {
			if parts[i] == originParts[i] {
				score++
			} else {
				break
			}
		}

		if score > bestScore {
			bestScore = score
			best = p
		} else if score == bestScore {
			if string(p) < string(best) {
				best = p
			}
		}
	}

	return best
}

// ResolvePath traduz o que o cliente escreveu — caminho, nome de arquivo ou
// alias — no caminho canonico da nota. Devolve ErrAmbiguousPath quando mais de
// um arquivo casa e o desempate nao decide.
func (ix *Index) ResolvePath(input string) (vault.CanonicalPath, error) {
	ix.mu.RLock()
	defer ix.mu.RUnlock()

	p := vault.CanonicalPath(filepath.ToSlash(input))
	if _, ok := ix.notes[p]; ok {
		return p, nil
	}
	if _, ok := ix.assets[p]; ok {
		return p, nil
	}

	lower := strings.ToLower(filepath.ToSlash(input))

	var matches []vault.CanonicalPath
	for lowerPath, realPath := range ix.lowerPath {
		if lowerPath == lower {
			matches = append(matches, realPath)
		}
	}

	if len(matches) == 0 {
		return "", errors.New("not found")
	}

	if len(matches) > 1 {
		return "", ErrAmbiguousPath
	}

	return matches[0], nil
}
