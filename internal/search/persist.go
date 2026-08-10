package search

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Constantes de versionamento de cache (Task 49).
const (
	// Formato 6: tudo do formato 5 (codec binario proprio, tabela de
	// caminhos, posicoes em varint com delta, totais adiantados, docLengths
	// gravado em vez de derivado, tudo em ordem crescente) MAIS uma segunda
	// secao com as posicoes em formato fixo de 16 bytes, alinhada em 8, e um
	// rodape de 24 bytes no fim do arquivo — para o array de posicoes poder
	// ser mapeado do arquivo em modo leitura em vez de alocado no heap de
	// cada processo (Task 89). O varint continua sendo o que a decodificacao
	// integral usa; a secao fixa so existe para quem mapeia.
	// Ver persist_codec.go para o layout e os numeros que motivaram cada peca.
	CacheFormatVersion   = 6
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

// WriteCacheForTest escreve um cache no formato corrente, com o cabecalho que
// o teste mandar.
//
// Substituiu NewEncoderForTest, que devolvia um gob.Encoder. Depois da troca de
// formato, um arquivo gob deixa de ter a assinatura "GBS2" e e recusado logo na
// primeira leitura — entao o teste de "versao de analisador divergente"
// passaria sem nunca chegar a comparar versao nenhuma. Teste que passa pelo
// motivo errado e pior que teste ausente.
func WriteCacheForTest(
	w io.Writer,
	h CacheHeader,
	termos map[string]map[string][]TokenPosition,
	docLengths map[string]int,
) error {
	return escreveCache(w, h, termos, docLengths)
}

// CacheHeader guarda as versões e metadados de integridade do cache.
type CacheHeader struct {
	FormatVersion   int
	ParserVersion   int
	AnalyzerVersion int
	VaultPath       string
	NoteCount       int
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

	termos, docLengths := inv.ExportForCache()

	tmpFile, err := os.CreateTemp(cacheDir, ".gobsidian-tmp-cache-*.gob")
	if err != nil {
		return fmt.Errorf("criando arquivo temporário de cache: %w", err)
	}
	tmpPath := tmpFile.Name()

	if err := escreveCache(tmpFile, header, termos, docLengths); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("codificando cache: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("fechando arquivo temporário de cache: %w", err)
	}

	finalPath := filepath.Join(cacheDir, "inverted_cache.gob")

	// Promove o array de posições de `inv` para o heap ANTES do rename, se
	// ele vier de uma arena mapeada deste mesmo arquivo. Sem isto, o rename
	// abaixo falharia no Windows sempre que um cache PARCIAL retomado fosse
	// regravado (ver o comentário em promoverArenaSePresente, em mmap.go).
	promoverArenaSePresente(inv)

	if err := os.Rename(tmpPath, finalPath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("renomeando cache temporário para %q: %w", finalPath, err)
	}

	return nil
}

// LoadInvertedCache lê o índice invertido do disco e valida seu cabeçalho.
//
// Tenta mapear a seção fixa de posições do arquivo (Task 89) antes de cair
// para a decodificação integral de sempre. A tentativa é recusada — não é
// erro — quando cacheDir está dentro do cofre (dentroDoCofre), quando o
// arquivo é pequeno demais para ter rodapé, quando o rodapé está ausente ou
// não bate, ou quando a plataforma não conseguiu mapear por qualquer motivo:
// em todos esses casos o caminho de sempre (os.ReadFile + leCache) decide
// sozinho se o arquivo é válido, corrompido ou de versão incompatível.
func LoadInvertedCache(ctx context.Context, cacheDir string, vaultPath string) (*Inverted, *CacheHeader, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	if cacheDir == "" {
		return nil, nil, ErrCacheNotFound
	}

	finalPath := filepath.Join(cacheDir, "inverted_cache.gob")

	if _, err := os.Stat(finalPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil, ErrCacheNotFound
		}
		return nil, nil, fmt.Errorf("consultando cache %q: %w", finalPath, err)
	}

	if arena, ok := tentaAbrirArena(finalPath, vaultPath); ok {
		h, base, err := leCacheComArena(arena.dados, arena.pos)
		if err != nil {
			_ = arena.fechar()
			if errors.Is(err, ErrCacheVersionMismatch) {
				return nil, nil, err
			}
			return nil, nil, ErrCacheCorrupted
		}
		if h.FormatVersion != CacheFormatVersion || h.ParserVersion != CacheParserVersion || h.AnalyzerVersion != CacheAnalyzerVersion || (vaultPath != "" && h.VaultPath != vaultPath) {
			_ = arena.fechar()
			return nil, &h, ErrCacheVersionMismatch
		}
		base.arenaFechar = arena.fechar
		inv := newInvertedFromSoA(base)
		return inv, &h, nil
	}

	// Sem arena: recusada, corrompida/truncada demais para ter rodapé, ou o
	// mapeamento falhou na plataforma. os.ReadFile e nao os.Open: o
	// decodificador trabalha sobre a fatia inteira (ver leitor em
	// persist_codec.go), e ReadFile dimensiona o buffer pelo tamanho
	// declarado no stat, numa alocacao so.
	dados, err := os.ReadFile(finalPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil, ErrCacheNotFound
		}
		return nil, nil, fmt.Errorf("lendo cache %q: %w", finalPath, err)
	}

	h, base, err := leCache(dados)
	if err != nil {
		if errors.Is(err, ErrCacheVersionMismatch) {
			return nil, nil, err
		}
		return nil, nil, ErrCacheCorrupted
	}

	if h.FormatVersion != CacheFormatVersion || h.ParserVersion != CacheParserVersion || h.AnalyzerVersion != CacheAnalyzerVersion || (vaultPath != "" && h.VaultPath != vaultPath) {
		return nil, &h, ErrCacheVersionMismatch
	}

	inv := newInvertedFromSoA(base)
	return inv, &h, nil
}
