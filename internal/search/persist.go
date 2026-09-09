package search

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/jonyd/gobsidian/internal/vault"
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

// CacheHeader guarda as versões e metadados de integridade do cache.
type CacheHeader struct {
	FormatVersion   int
	ParserVersion   int
	AnalyzerVersion int
	VaultPath       string
	NoteCount       int
}

// NomeDoArquivoDeCache e o nome do arquivo do indice invertido dentro do
// cache-dir. Ate 2026-09-08 o literal aparecia duas vezes neste mesmo arquivo,
// e a limpeza do instalador seria a terceira.
const NomeDoArquivoDeCache = "inverted_cache.gob"

// LerCabecalhoDoCache responde "de qual cofre e este cache?" sem decodificar o
// corpo.
//
// A limpeza do instalador (internal/instalar) precisa disso para decidir se um
// diretorio de cache ficou orfao: o diretorio e nomeado pelo HASH do caminho do
// cofre (config.VaultKey), e hash nao volta para o caminho. O cabecalho ja
// guarda VaultPath desde sempre -- ver CacheHeader --, entao a resposta existe
// e so faltava alcanca-la.
//
// Le um PREFIXO do arquivo, e nao o arquivo: o cache do cofre de referencia do
// dono tem 66 MB, e carregar tudo para ler um campo seria absurdo numa varredura
// que roda uma vez por diretorio. Se o prefixo nao bastar -- caminho de cofre
// patologicamente longo --, ele releva e tenta o arquivo inteiro, porque
// "cabecalho truncado" e "arquivo corrompido" nao podem virar a mesma resposta.
func LerCabecalhoDoCache(cacheDir string) (CacheHeader, error) {
	caminho := filepath.Join(cacheDir, NomeDoArquivoDeCache)

	f, err := os.Open(caminho)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return CacheHeader{}, ErrCacheNotFound
		}
		return CacheHeader{}, fmt.Errorf("abrindo cache %q: %w", caminho, err)
	}
	defer func() { _ = f.Close() }()

	prefixo := make([]byte, prefixoDoCabecalho)
	n, err := io.ReadFull(f, prefixo)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		return CacheHeader{}, fmt.Errorf("lendo cache %q: %w", caminho, err)
	}
	prefixo = prefixo[:n]

	h, err := decodificaCabecalho(&leitor{b: prefixo})
	if err == nil {
		return h, nil
	}
	if errors.Is(err, ErrCacheVersionMismatch) || n < prefixoDoCabecalho {
		// O prefixo era o arquivo inteiro, ou a versao nao bate: reler nao muda
		// nada.
		return h, err
	}

	dados, lerErr := os.ReadFile(caminho)
	if lerErr != nil {
		return CacheHeader{}, fmt.Errorf("relendo cache %q: %w", caminho, lerErr)
	}
	return decodificaCabecalho(&leitor{b: dados})
}

// prefixoDoCabecalho e quanto se le antes de decodificar o cabecalho. 64 KiB
// cobre com folga assinatura, tres versoes, o caminho do cofre e a contagem de
// notas -- o caminho mais longo medido na maquina do dono em 2026-09-08 tinha
// 334 caracteres.
const prefixoDoCabecalho = 64 << 10

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

	// Promove a arena ANTES de exportar e gravar: ReplaceFile faz o rename
	// por dentro, e o rename falha no Windows enquanto o alvo esta mapeado
	// (ver promoverArenaSePresente, em mmap.go). ExportForCache ja copia as
	// posicoes, entao a ordem aqui e sobre o rename, nao sobre aliasing.
	promoverArenaSePresente(inv)
	termos, docLengths := inv.ExportForCache()

	finalPath := filepath.Join(cacheDir, NomeDoArquivoDeCache)
	if err := vault.ReplaceFile(ctx, finalPath, func(f *os.File) error {
		return escreveCache(f, header, termos, docLengths)
	}); err != nil {
		return fmt.Errorf("gravando cache de busca em %q: %w", finalPath, err)
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

	finalPath := filepath.Join(cacheDir, NomeDoArquivoDeCache)

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
