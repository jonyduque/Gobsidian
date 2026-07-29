package search

import (
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Constantes de versionamento de cache (Task 49).
const (
	CacheFormatVersion   = 1
	CacheParserVersion   = 1
	CacheAnalyzerVersion = 1
)

// Erros exportados para operações de persistência de cache.
var (
	// ErrCacheNotFound indica que o arquivo de cache não existe.
	ErrCacheNotFound = errors.New("cache file not found")
	// ErrCacheVersionMismatch indica incompatibilidade de versão de formato, parser ou analisador.
	ErrCacheVersionMismatch = errors.New("cache version mismatch")
	// ErrCacheCorrupted indica erro de decodificação ou corrupção no arquivo de cache.
	ErrCacheCorrupted = errors.New("cache file corrupted")
)

// NewEncoderForTest expõe um gob.Encoder para suíte de testes de mutação.
func NewEncoderForTest(w interface{ Write([]byte) (int, error) }) *gob.Encoder {
	return gob.NewEncoder(w)
}

// CacheHeader guarda as versões e metadados de integridade do cache.
type CacheHeader struct {
	FormatVersion   int
	ParserVersion   int
	AnalyzerVersion int
	VaultPath       string
	NoteCount       int
}

// InvertedCacheData é a estrutura serializada do índice invertido.
type InvertedCacheData struct {
	Header CacheHeader
	Terms  map[string]map[string][]TokenPosition
}

// SaveInvertedCache salva o índice invertido em disco atomicamente.
func SaveInvertedCache(ctx context.Context, cacheDir string, vaultPath string, inv *Inverted) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if cacheDir == "" || inv == nil {
		return nil
	}

	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("criando cacheDir: %w", err)
	}

	header := CacheHeader{
		FormatVersion:   CacheFormatVersion,
		ParserVersion:   CacheParserVersion,
		AnalyzerVersion: CacheAnalyzerVersion,
		VaultPath:       vaultPath,
		NoteCount:       inv.DocCount(),
	}

	data := InvertedCacheData{
		Header: header,
		Terms:  inv.ExportTermsForCache(),
	}

	tmpFile, err := os.CreateTemp(cacheDir, ".gobsidian-tmp-cache-*.gob")
	if err != nil {
		return fmt.Errorf("criando arquivo temporário de cache: %w", err)
	}
	tmpPath := tmpFile.Name()

	enc := gob.NewEncoder(tmpFile)
	if err := enc.Encode(data); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("codificando gob de cache: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("fechando arquivo temporário de cache: %w", err)
	}

	finalPath := filepath.Join(cacheDir, "inverted_cache.gob")
	if err := os.Rename(tmpPath, finalPath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("renomeando cache temporário para %q: %w", finalPath, err)
	}

	return nil
}

// LoadInvertedCache lê o índice invertido do disco e valida seu cabeçalho.
func LoadInvertedCache(ctx context.Context, cacheDir string, vaultPath string) (*Inverted, *CacheHeader, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	if cacheDir == "" {
		return nil, nil, ErrCacheNotFound
	}

	finalPath := filepath.Join(cacheDir, "inverted_cache.gob")
	f, err := os.Open(finalPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil, ErrCacheNotFound
		}
		return nil, nil, fmt.Errorf("abrindo cache %q: %w", finalPath, err)
	}
	defer func() { _ = f.Close() }()

	var data InvertedCacheData
	dec := gob.NewDecoder(f)
	if err := dec.Decode(&data); err != nil {
		return nil, nil, ErrCacheCorrupted
	}

	h := data.Header
	if h.FormatVersion != CacheFormatVersion || h.ParserVersion != CacheParserVersion || h.AnalyzerVersion != CacheAnalyzerVersion || (vaultPath != "" && h.VaultPath != vaultPath) {
		return nil, &h, ErrCacheVersionMismatch
	}

	inv := NewInvertedFromCache(data.Terms)
	return inv, &h, nil
}
