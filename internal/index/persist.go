package index

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/jonyd/gobsidian/internal/vault"
)

// Constantes de versionamento do cache do índice de metadados.
//
// docs/PRD.md Q3 (fechada em 2026-07-29) decidiu persistir OS DOIS caches —
// index_cache.gob (aqui) e inverted_cache.gob (internal/search) — e só o
// segundo existia até esta tarefa. A anotação em PRD.md registrava a
// lacuna; ela é fechada junto com este arquivo.
const (
	// IndexCacheFormatVersion muda quando o layout serializado muda.
	//
	// 2 -> 3 em 2026-08-26: cada link passou a gravar o Context recortado do
	// corpo (achado A8). Cache da versao anterior e recusado e reconstruido,
	// que e o comportamento certo: carrega-lo daria backlinks com contexto
	// vazio para sempre, sem nada indicando por que.
	IndexCacheFormatVersion = 3
	// IndexCacheParserVersion muda quando o parser passa a produzir
	// estrutura diferente para a mesma entrada. Sem isto, corrigir um bug de
	// parsing e reiniciar carregaria de volta o índice errado: mtime e
	// tamanho dos arquivos não mudaram, então VerifyFreshness não veria
	// motivo pra descartar o cache, e o bug pareceria não corrigido. Mesmo
	// raciocínio de ARCHITECTURE.md §6.3 para o parser_version do cache de
	// busca.
	IndexCacheParserVersion = 1
)

// indexCacheFileName é o nome fixo dentro de cfg.CacheDir — o mesmo
// diretório do inverted_cache.gob, arquivo irmão.
const indexCacheFileName = "index_cache.gob"

// Erros exportados para operações de persistência do cache de metadados.
var (
	// ErrIndexCacheNotFound indica que o arquivo de cache não existe.
	ErrIndexCacheNotFound = errors.New("index cache file not found")
	// ErrIndexCacheVersionMismatch indica incompatibilidade de formato, de
	// parser ou de caminho do cofre.
	ErrIndexCacheVersionMismatch = errors.New("index cache version mismatch")
	// ErrIndexCacheCorrupted indica erro de decodificação ou checksum que
	// não bate.
	ErrIndexCacheCorrupted = errors.New("index cache file corrupted")
	// ErrIndexCachePartial indica que o cabeçalho promete mais do que o
	// corpo do arquivo trouxe. É a regra que o cache de busca aprendeu na
	// marra (ver o comentário de invertedCacheState em cmd/gobsidian/serve.go):
	// LoadInvertedCache conferia versão e não contagem, e um cache parcial
	// passava por completo.
	ErrIndexCachePartial = errors.New("index cache incomplete")
)

// CacheHeader guarda as versões e a cobertura declarada do cache.
type CacheHeader struct {
	FormatVersion int
	ParserVersion int
	VaultPath     string
	NoteCount     int
	AssetCount    int
}

// WriteIndexCacheForTest escreve um cache no formato corrente com o
// cabeçalho que o teste mandar — inclusive um cabeçalho que MENTE sobre a
// cobertura, para exercitar a checagem de LoadIndexCache sem depender de
// conseguir produzir um cache parcial de verdade (SaveIndexCache sempre
// grava um cabeçalho fiel ao que exportou).
func WriteIndexCacheForTest(w io.Writer, h CacheHeader, notes []*Note, assets []*Asset) error {
	return escreveIndexCache(w, h, notes, assets)
}

// SaveIndexCache salva o índice de metadados em disco atomicamente:
// temporário no mesmo diretório, depois rename — o mesmo padrão de
// SaveInvertedCache.
func SaveIndexCache(ctx context.Context, cacheDir, vaultPath string, ix *Index) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if cacheDir == "" || ix == nil {
		return nil
	}

	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("criando cacheDir: %w", err)
	}

	ix.mu.RLock()
	notes := make([]*Note, 0, len(ix.notes))
	for _, n := range ix.notes {
		notes = append(notes, n)
	}
	assets := make([]*Asset, 0, len(ix.assets))
	for _, a := range ix.assets {
		assets = append(assets, a)
	}
	ix.mu.RUnlock()

	header := CacheHeader{
		FormatVersion: IndexCacheFormatVersion,
		ParserVersion: IndexCacheParserVersion,
		VaultPath:     vaultPath,
		NoteCount:     len(notes),
		AssetCount:    len(assets),
	}

	tmpFile, err := os.CreateTemp(cacheDir, ".gobsidian-tmp-index-cache-*.gob")
	if err != nil {
		return fmt.Errorf("criando arquivo temporario de cache de indice: %w", err)
	}
	tmpPath := tmpFile.Name()

	if err := escreveIndexCache(tmpFile, header, notes, assets); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("codificando cache de indice: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("fechando arquivo temporario de cache de indice: %w", err)
	}

	finalPath := filepath.Join(cacheDir, indexCacheFileName)
	if err := os.Rename(tmpPath, finalPath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("renomeando cache de indice temporario para %q: %w", finalPath, err)
	}

	return nil
}

// LoadIndexCache lê o índice de metadados do disco, valida o cabeçalho e
// devolve um *Index totalmente povoado.
//
// byAlias, backlinks e a resolução de cada link NÃO vêm do arquivo: são
// recalculados chamando as MESMAS três funções que Build chama depois de
// indexar — buildAliasMap, resolveAllLinks, buildBacklinks. Persistir esses
// três separadamente seria uma segunda forma de calcular o mesmo dado a
// partir das notas, e a lição registrada em CLAUDE.md é que a forma menos
// usada é a que diverge: foi exatamente assim que byAlias ficou minúsculo
// numa via e cru na outra.
func LoadIndexCache(ctx context.Context, cacheDir, vaultPath string) (*Index, *CacheHeader, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	if cacheDir == "" {
		return nil, nil, ErrIndexCacheNotFound
	}

	finalPath := filepath.Join(cacheDir, indexCacheFileName)
	dados, err := os.ReadFile(finalPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil, ErrIndexCacheNotFound
		}
		return nil, nil, fmt.Errorf("lendo cache de indice %q: %w", finalPath, err)
	}

	h, notes, assets, err := leIndexCache(dados)
	if err != nil {
		if errors.Is(err, ErrIndexCacheVersionMismatch) {
			return nil, &h, err
		}
		return nil, nil, ErrIndexCacheCorrupted
	}

	if h.FormatVersion != IndexCacheFormatVersion || h.ParserVersion != IndexCacheParserVersion ||
		(vaultPath != "" && h.VaultPath != vaultPath) {
		return nil, &h, ErrIndexCacheVersionMismatch
	}

	idx := New()
	idx.mu.Lock()
	for _, n := range notes {
		idx.publishNoteLocked(n)
	}
	for _, a := range assets {
		idx.publishAssetLocked(a)
	}
	idx.mu.Unlock()

	idx.resolveAllLinks()
	idx.buildBacklinks()

	// O cabeçalho confere COBERTURA, não só versão. A lição já paga: um
	// cache parcial passou por completo porque LoadInvertedCache conferia
	// versão e não contagem — o equivalente aqui é qualquer nota que não
	// gerou entrada (nota vazia inclusive) escapando da contagem.
	if h.NoteCount != idx.NoteCount() {
		return nil, &h, fmt.Errorf("%w: cabecalho declara %d notas, corpo trouxe %d",
			ErrIndexCachePartial, h.NoteCount, idx.NoteCount())
	}
	if h.AssetCount != idx.AssetCount() {
		return nil, &h, fmt.Errorf("%w: cabecalho declara %d anexos, corpo trouxe %d",
			ErrIndexCachePartial, h.AssetCount, idx.AssetCount())
	}

	return idx, &h, nil
}

// VerifyFreshness confere se o índice carregado do cache ainda bate com o
// disco: mesma contagem de arquivos, e mesmo tamanho e mtime por arquivo.
// Invalidação é por mtime e tamanho por arquivo — mesmo raciocínio de
// ARCHITECTURE.md §6.3 para o cache de busca. Qualquer divergência devolve
// false; não há reparo arquivo a arquivo aqui, só a decisão entre servir do
// cache ou reconstruir do zero — quem chamou decide.
func (ix *Index) VerifyFreshness(ctx context.Context, v *vault.Vault) (bool, error) {
	ix.mu.RLock()
	notes := make(map[vault.CanonicalPath]*Note, len(ix.notes))
	for p, n := range ix.notes {
		notes[p] = n
	}
	assets := make(map[vault.CanonicalPath]*Asset, len(ix.assets))
	for p, a := range ix.assets {
		assets[p] = a
	}
	ix.mu.RUnlock()

	seen := 0
	fresh := true
	err := v.Walk(ctx, func(e vault.Entry) error {
		seen++
		if e.IsNote {
			n, ok := notes[e.Path]
			if !ok || n.Size != e.Size || !n.ModTime.Equal(e.ModTime) || n.CloudOnly != e.CloudOnly {
				fresh = false
			}
		} else {
			a, ok := assets[e.Path]
			if !ok || a.Size != e.Size || !a.ModTime.Equal(e.ModTime) {
				fresh = false
			}
		}
		return nil
	})
	if err != nil {
		return false, err
	}
	// seen tem de bater com o total indexado: um arquivo NOVO no disco não
	// diverge em tamanho/mtime de nada (não há entrada pra comparar), então
	// só a contagem pega uma adição pura. Notas apagadas do disco, ao
	// contrário, ficam de fora da varredura e por isso não aparecem aqui —
	// mas aí a contagem também diverge, porque seen < len(notes)+len(assets).
	if !fresh || seen != len(notes)+len(assets) {
		return false, nil
	}
	return true, nil
}
