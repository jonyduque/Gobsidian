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

// ErrPathNotFound sinaliza que a entrada nao casou com nota nem anexo nenhum.
//
// Era um errors.New("not found") anonimo ate 2026-08-27 (achado M2). Sem
// sentinela o chamador nao tinha como distinguir "nao existe" de qualquer outra
// falha, e dois deles — LinkGraph e NoteMetadata — respondiam
// PATH_OUTSIDE_VAULT para nota inexistente. O host le esse codigo como
// "tentativa de sair do cofre", que e um erro de SEGURANCA: a resposta acusava
// o cliente de algo que ele nao fez.
//
// ResolvePath NAO verifica confinamento — ela so procura no indice. Entao
// "nao encontrado" e a unica coisa que ela pode afirmar aqui.
var ErrPathNotFound = errors.New("path not found")

// nomeChave normaliza um alvo de link — ou um nome derivado de caminho ou
// alias — para a chave usada no indice reverso citantesPorNome. Toma so o
// ultimo segmento do caminho, sem sufixo .md, em minusculas.
//
// E uma SUPERAPROXIMACAO deliberada. resolveByName casa nome exato (sensivel
// a caixa, com .md so para nota) e resolveByAlias casa por aliasKey (so
// minusculas, sem separar diretorio). Colapsar as duas regras na mesma chave
// normalizada pode fazer um link ser marcado para reprocessar A MAIS —
// nunca a menos. Falso positivo aqui e barato (reprocessa um link que nao
// mudou); falso negativo e o defeito que este indice existe para nunca
// reintroduzir — ver o comentario de citantesPorNome em index.go e a
// armadilha de aliasKey em alias.go: chave crua num lado e normalizada no
// outro é exatamente como [[STJ]] continuou resolvendo, com state=ok, para
// uma nota ja removida.
func nomeChave(s string) string {
	s = filepath.ToSlash(s)
	if i := strings.LastIndex(s, "/"); i >= 0 {
		s = s[i+1:]
	}
	s = chaveDeCaminho(s)
	return strings.TrimSuffix(s, ".md")
}

// chavesDaNota devolve as chaves de citantesPorNome que uma nota afeta ao
// ser inserida, alterada ou removida: o proprio nome de arquivo e cada
// alias declarado.
//
// Alias conta — decisao fechada da Task 86: criar uma nota com
// "aliases: [STJ]" tem de afetar todo "[[STJ]]" do cofre, exatamente como
// conta na resolucao de verdade (resolveByAlias, acima). Sem este loop, uma
// nota citada so por alias nunca dispara o reprocessamento de quem a cita.
func chavesDaNota(n *Note) []string {
	chaves := []string{nomeChave(string(n.Path))}
	for _, alias := range n.Aliases {
		chaves = append(chaves, nomeChave(alias))
	}
	return chaves
}

// resolveAllLinks resolve os links de TODAS as notas.
//
// Escreve por CÓPIA, via mutarNotaLocked, e não no lugar.
//
// Até 2026-08-28 ela mutava `n.Links[i]` direto na Note já publicada no mapa
// (achado B12). Sob o lock de escrita, o que é seguro contra outra escrita — mas
// não contra LEITURA: `idx.Get` devolve o ponteiro sob RLock e quem chamou lê os
// campos depois de soltar o lock. É exatamente a corrida que o detector já
// apontou uma vez em `MoveNote`, e que a disciplina de cópia existe para evitar.
//
// Era "seguro por ORDEM DE CHAMADAS, não por construção": os dois chamadores —
// `Build` e `LoadIndexCache` — rodam antes de o índice ser visível a qualquer
// consulta. Invariante que depende de ninguém mudar a ordem de chamada não
// sobrevive à próxima pessoa; agora ela não é mais necessária.
func (ix *Index) resolveAllLinks() {
	ix.mu.Lock()
	defer ix.mu.Unlock()

	for path, atual := range ix.notes {
		// Nota sem link nenhum nao precisa de copia: nada seria escrito nela.
		if len(atual.Links) == 0 {
			continue
		}
		// Uma cópia por nota, no máximo: mutarNotaLocked devolve a mesma
		// enquanto o acumulador não for nil.
		var copia *Note
		alvo := ix.mutarNotaLocked(path, &copia)

		for i := range alvo.Links {
			resolved, via, state := ix.resolveTarget(alvo.Links[i].Target, alvo.Path)
			alvo.Links[i].Resolved = resolved
			alvo.Links[i].Via = via
			alvo.Links[i].State = state

			if state == LinkOK {
				ix.resolveAnchor(&alvo.Links[i])
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
	name := nomeDeArquivo(target, isNote)

	valid := ix.candidatosPorNomeLocked(name, isNote)
	if len(valid) == 0 {
		return "", false
	}
	if len(valid) == 1 {
		return valid[0], true
	}

	return ix.tiebreak(valid, origin), true
}

// candidatosPorNomeLocked devolve TODOS os caminhos vivos que casam um nome de
// arquivo, sem desempatar.
//
// Separada de resolveByName porque os dois chamadores querem coisas diferentes
// da mesma busca. O wikilink QUER o desempate por proximidade — e o
// comportamento documentado de `[[nota]]` num cofre com homonimos. Uma chamada
// de tool nao tem nota de origem para medir proximidade, e escolher um dos
// candidatos ali seria devolver um valor arbitrario com cara de resposta.
// ResolvePath usa esta lista para poder dizer "ambiguo" em vez de chutar.
//
// Exige ix.mu ja tomado.
func (ix *Index) candidatosPorNomeLocked(name string, isNote bool) []vault.CanonicalPath {
	paths, ok := ix.byName[chaveDeNomeDeArquivo(name)]
	if !ok || len(paths) == 0 {
		return nil
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
	return valid
}

// nomeDeArquivo aplica a mesma normalizacao que resolveByName usa: nota ganha
// ".md" quando nao o tem; anexo exige extensao explicita. Uma conta so.
func nomeDeArquivo(target string, isNote bool) string {
	if !isNote {
		return target
	}
	if strings.HasSuffix(target, ".md") {
		return target
	}
	return target + ".md"
}

func (ix *Index) resolveByAlias(target string, origin vault.CanonicalPath) (vault.CanonicalPath, bool) {
	key := aliasKey(target)
	paths, ok := ix.byAlias[key]
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
// alias — no caminho canonico da nota.
//
// # As tres formas, nesta ordem
//
// Ate 2026-08-28 o comentario prometia as tres e o corpo fazia DUAS: caminho
// exato e caminho insensivel a maiusculas (achado M6). Nome de arquivo e alias
// nao resolviam, embora `resolveTarget` — que serve os wikilinks — soubesse
// faze-lo desde sempre. O resultado era que `[[STJ]]` numa nota resolvia e
// `note_read` com "STJ" nao, apesar de esta ser a porta UNICA de todas as tools.
//
// A ordem importa e e do mais especifico para o menos: caminho exato nunca pode
// perder para um alias homonimo.
//
// # ErrAmbiguousPath so aparece onde a ambiguidade existe
//
// O ramo insensivel a maiusculas varria o mapa INTEIRO comparando chave por
// chave, e devolvia ErrAmbiguousPath se achasse duas — inalcancavel, porque
// `lowerPath` tem chave unica por construcao (`publishNameLocked`). Agora e um
// lookup direto, e o erro de ambiguidade vive onde ela e real: nome e alias, que
// podem casar mais de uma nota e passam pelo desempate de `tiebreak`.
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

	// Insensivel a maiusculas: lookup direto, nao varredura.
	if canonico, ok := ix.lowerPath[chaveDeCaminho(input)]; ok {
		return canonico, nil
	}

	// Nome de arquivo, com ou sem ".md" — e depois anexo, que exige extensao.
	//
	// Aqui NAO ha desempate por proximidade: a chamada de tool nao tem nota de
	// origem, e escolher um entre homonimos devolveria um caminho arbitrario com
	// cara de resposta. Dois candidatos e ambiguidade de verdade, e e isto que
	// mantem ErrAmbiguousPath vivo depois que a varredura insensivel a
	// maiusculas — onde ele era inalcancavel — deixou de existir.
	for _, isNote := range []bool{true, false} {
		candidatos := ix.candidatosPorNomeLocked(nomeDeArquivo(input, isNote), isNote)
		switch len(candidatos) {
		case 0:
			// segue para a proxima forma
		case 1:
			return candidatos[0], nil
		default:
			return "", ErrAmbiguousPath
		}
	}

	// Alias do frontmatter, com o mesmo criterio.
	if paths, ok := ix.byAlias[aliasKey(input)]; ok {
		var vivos []vault.CanonicalPath
		for _, p := range paths {
			if _, existe := ix.notes[p]; existe {
				vivos = append(vivos, p)
			}
		}
		switch len(vivos) {
		case 0:
		case 1:
			return vivos[0], nil
		default:
			return "", ErrAmbiguousPath
		}
	}

	return "", ErrPathNotFound
}
