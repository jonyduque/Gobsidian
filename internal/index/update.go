package index

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
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

	// chaves acumula, ao longo da funcao, tudo que este Replace pode ter
	// afetado: o que a versao antiga desta entrada citava (devolvido por
	// removeContributionsLocked) mais o que a versao nova passa a citar. E o
	// que reprocessLinksDirigidoLocked usa no final para achar SO as notas
	// que citam algum desses nomes — em vez de varrer o cofre inteiro a
	// cada evento do watcher (RNF-06, Task 86).
	chaves := ix.removeContributionsLocked(path)

	atomic.AddUint64(&ix.generation, 1)

	abs := v.Abs(path)
	info, err := os.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			// Nota sumiu entre o evento e o Stat. Reprocessar so quem citava
			// o nome dela — o que marca como quebrados os links que
			// apontavam para ela.
			ix.reprocessLinksDirigidoLocked(chaves)
			return nil
		}
		return err
	}

	// IsNote sai de vault.Classify, a mesma funcao que vault.Walk consulta na
	// varredura. Derivar aqui por sufixo ".md" era uma segunda copia da regra,
	// que discordava da primeira em nome de ruido do editor (`~$nota.md`).
	entry := vault.Entry{
		Path:      path,
		Size:      info.Size(),
		ModTime:   info.ModTime(),
		IsNote:    vault.Classify(path) == vault.ClassNote,
		CloudOnly: vault.IsCloudOnly(abs),
	}

	classificacao := classificar(entry)
	if classificacao == classeAnexo {
		ix.publishAssetLocked(anexoDaEntrada(entry))
		ix.reprocessLinksDirigidoLocked(chaves)
		return nil
	}
	if classificacao == classeNotaSemLeitura {
		// Placeholder de nuvem: entra como NOTA, com o que a varredura de
		// diretorio ja sabia, sem abrir o arquivo. Build sempre fez assim; era
		// Replace que o registrava como Asset, de modo que idx.Get devolvia
		// false para um `.md` conforme qual construcao tivesse rodado.
		ix.publishNoteLocked(notaSemLeitura(entry))
		ix.reprocessLinksDirigidoLocked(chaves)
		return nil
	}

	data, err := v.ReadAll(ctx, entry.Path)
	if err != nil {
		return err
	}

	body, hadBOM := vault.StripBOM(data)
	note := parser.Parse(body)
	if hadBOM {
		note.ShiftOffsets(int64(vault.BOMLen))
	}

	n := &Note{
		Path:           entry.Path,
		Title:          note.Title,
		TitleNorm:      normalizeTitleForNote(note.Title),
		Size:           entry.Size,
		ModTime:        entry.ModTime,
		Hash:           xxhash.Sum64(data),
		EOL:            vault.DetectEOL(data),
		BOM:            hadBOM,
		CloudOnly:      entry.CloudOnly,
		Frontmatter:    note.Frontmatter,
		FrontmatterErr: note.FrontmatterErr,
		Tags:           note.Tags,
		Aliases:        note.Aliases,
		Headings:       note.Headings,
		Blocks:         note.Blocks,
		Inline:         note.Inline,
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

	// A nota nova pode citar nomes diferentes da versao antiga (link
	// editado, alias adicionado ou removido) — registra o que ela cita AGORA
	// e acumula as chaves que a MUDANCA DE IDENTIDADE dela afeta em quem
	// cita ela (nome de arquivo inalterado aqui, mas alias pode ter mudado).
	ix.registrarCitantesLocked(entry.Path, n.Links)
	chaves = append(chaves, chavesDaNota(n)...)

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

	// Um alvo recem-criado pode fazer links passarem a resolver — so nas
	// notas que citam algum dos nomes em chaves.
	ix.reprocessLinksDirigidoLocked(chaves)
	return nil
}

// Remove tira o caminho do indice e marca como quebrados os links que
// apontavam para ele.
func (ix *Index) Remove(path vault.CanonicalPath) {
	ix.mu.Lock()
	defer ix.mu.Unlock()

	chaves := ix.removeContributionsLocked(path)
	atomic.AddUint64(&ix.generation, 1)

	// Reprocessar so quem citava o nome da nota removida — o que marca como
	// quebrados os links que apontavam para ela.
	ix.reprocessLinksDirigidoLocked(chaves)
}

// removeContributionsLocked tira path dos indices derivados e devolve as
// chaves de citantesPorNome que a remocao pode ter afetado: o proprio nome
// do arquivo e, se era uma nota, cada alias que ela declarava. E o chamador
// (Replace ou Remove) quem decide o que fazer com essas chaves — aqui so se
// apaga o que path contribuia.
func (ix *Index) removeContributionsLocked(path vault.CanonicalPath) []string {
	var chaves []string

	oldNote, hadNote := ix.notes[path]
	if hadNote {
		chaves = chavesDaNota(oldNote)
		ix.desregistrarCitantesLocked(path, oldNote.Links)

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

	if chaves == nil {
		// Nem nota nem anexo: path pode nao ter existido no indice (Replace
		// chama isto incondicionalmente, antes de saber se ha algo novo pra
		// por no lugar). O proprio nome ainda e uma chave valida — algo pode
		// citar esse nome sem que exista arquivo nenhum com ele.
		chaves = []string{nomeChave(string(path))}
	}

	return chaves
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
// entao publicam. Quem nao obedecia era MoveNote e o reprocessamento de
// links, que ate a Task 86 rodava em TODO evento do watcher, sobre TODAS as
// notas: era a corrida mais larga das duas.
//
// Toda escrita em nota publicada passa por aqui. Nao e para consertar as duas
// que estavam erradas: e para a proxima nao ter onde nascer.
//
// A copia e preguicosa de proposito. reprocessLinksDirigidoLocked visita so
// as notas que citam algum nome afetado, e quase nenhuma delas muda de fato;
// copiar todas trocaria uma corrida por lixo proporcional ao numero de
// citantes a cada evento.
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

// citantesAtuais devolve, sem copiar, a lista ja registrada para um alvo de
// link — leitura auxiliar de registrarCitantesLocked, com o parametro
// nomeado diferente de proposito: a prova de mutacao da Task 86 muta a linha
// de escrita abaixo, e se esta leitura usasse o mesmo texto a ancora casaria
// duas vezes e mutate.ps1 recusaria por ambiguidade.
func (ix *Index) citantesAtuais(nome string) []vault.CanonicalPath {
	return ix.citantesPorNome[nomeChave(nome)]
}

// registrarCitantesLocked publica os alvos dos links de uma nota no indice
// reverso citantesPorNome — o que permite, quando outra entrada muda,
// encontrar so as notas que citam aquele nome sem varrer o cofre inteiro
// (RNF-06, Task 86).
//
// A chave e SEMPRE nomeChave(alvo), nunca o alvo cru. Mesma disciplina de
// aliasKey em alias.go: toda escrita passa por uma unica funcao de
// normalizacao, ou a proxima diverge exatamente como byAlias divergiu —
// [[STJ]] continuando a resolver, com state=ok, para uma nota ja removida.
func (ix *Index) registrarCitantesLocked(path vault.CanonicalPath, links []ResolvedLink) {
	for _, l := range links {
		alvo := l.Target
		if alvo == "" {
			continue
		}
		if slices.Contains(ix.citantesAtuais(alvo), path) {
			continue
		}
		ix.citantesPorNome[nomeChave(alvo)] = append(ix.citantesAtuais(alvo), path)
	}
}

// desregistrarCitantesLocked e o par de registrarCitantesLocked: tira path
// de citantesPorNome sob a chave de cada alvo que ela citava. Chamado antes
// de remover ou substituir uma nota, para que ela pare de aparecer como
// citante de nomes que nao cita mais.
func (ix *Index) desregistrarCitantesLocked(path vault.CanonicalPath, links []ResolvedLink) {
	vistas := make(map[string]bool, len(links))
	for _, l := range links {
		if l.Target == "" {
			continue
		}
		chave := nomeChave(l.Target)
		if vistas[chave] {
			continue
		}
		vistas[chave] = true

		lista := ix.citantesPorNome[chave]
		filtrada := make([]vault.CanonicalPath, 0, len(lista))
		for _, p := range lista {
			if p != path {
				filtrada = append(filtrada, p)
			}
		}
		if len(filtrada) == 0 {
			delete(ix.citantesPorNome, chave)
		} else {
			ix.citantesPorNome[chave] = filtrada
		}
	}
}

// reprocessLinksDirigidoLocked reavalia SO os links das notas que citam ao
// menos uma das chaves — descobertas por citantesPorNome, o indice reverso
// mantido a cada escrita. Substituiu o percurso completo do cofre que rodava
// em todo evento do watcher, sobre todas as notas (RNF-06, Task 86: 20,35 ms
// contra 20 ms de teto para reindexar UM arquivo de um cofre de 5.000).
//
// O caminho global — reavaliar toda nota, sempre — nao sobrevive no
// produto: fica so no teste, como referencia para o diferencial que prova
// esta funcao equivalente a ele (TestReresolucaoDirigidaIgualAGlobal).
func (ix *Index) reprocessLinksDirigidoLocked(chaves []string) {
	vistos := make(map[vault.CanonicalPath]bool)
	for _, chave := range chaves {
		for _, path := range ix.citantesPorNome[chave] {
			if vistos[path] {
				continue
			}
			vistos[path] = true
			ix.reprocessNoteLinksLocked(path)
		}
	}
}

// reprocessNoteLinksLocked reavalia os links de UMA nota: mesmo corpo que o
// antigo percurso global fazia por nota, restrito a um caminho so.
func (ix *Index) reprocessNoteLinksLocked(path vault.CanonicalPath) {
	n, ok := ix.notes[path]
	if !ok {
		return
	}

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

	// 1b. Atualizar citantesPorNome PARA OS LINKS DA PROPRIA NOTA MOVIDA —
	// o alvo de cada link nao muda (o conteudo nao foi relido), so o
	// caminho de QUEM cita. Sem isto a nota movida some do indice reverso
	// como citante, e o proximo evento que tocar um dos alvos dela nao a
	// encontraria para reprocessar.
	for _, l := range n.Links {
		if l.Target == "" {
			continue
		}
		lista := ix.citantesPorNome[nomeChave(l.Target)]
		for i, p := range lista {
			if p == oldPath {
				lista[i] = newPath
			}
		}
	}

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

	// 8. Reprocessar os links garante que links recém-quebrados ou
	// recém-resolvidos sejam notados — so nas notas que citam o nome antigo,
	// o nome novo (podem ser o mesmo, se so a pasta mudou) ou algum alias.
	// nomeChave(oldPath) tem de entrar mesmo quando so a pasta muda: o nome
	// de arquivo pode ser identico e chavesDaNota(n), que usa n.Path == newPath,
	// nao repete essa chave por si so — repetir aqui e barato e nunca errado.
	chaves := append([]string{nomeChave(string(oldPath))}, chavesDaNota(n)...)
	ix.reprocessLinksDirigidoLocked(chaves)
}
