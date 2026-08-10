package search

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"unsafe"
)

// Task 89 — arena de posições mapeada do arquivo.
//
// O array de posições é ~291 MB dos ~382 MB em repouso de uma instância que
// carregou o índice de busca no cofre de referência (18.229.295 posições ×
// 16 bytes). Hoje cada processo aloca a sua cópia no heap. Mapeado do arquivo
// de cache em modo leitura, o cache de páginas do sistema operacional
// compartilha as mesmas páginas físicas entre processos que abrem o mesmo
// arquivo — várias instâncias pagam a arena uma vez, não uma por processo.
//
// O layout do arquivo (a seção fixa e o rodapé) está em persist_codec.go. Este
// arquivo cuida de decidir SE mapear é seguro, e chamar o código específico de
// plataforma (mmap_windows.go, mmap_unix.go) que faz o mapeamento em si.

// arenaPosicoes é o resultado de mapear com sucesso a seção fixa de posições
// de um arquivo de cache.
type arenaPosicoes struct {
	// dados são os bytes crus do arquivo INTEIRO, mapeados. leCacheComArena
	// decodifica a parte estrutural (cabeçalho, termos, caminhos, postings) a
	// partir daqui, exatamente como decodificaria uma fatia vinda de
	// os.ReadFile — só o array de posições é que não vem de um make().
	dados []byte
	// pos é a reinterpretação, via unsafe, da seção fixa dentro de dados.
	pos []TokenPosition
	// fechar desfaz o mapeamento. Específico de plataforma.
	fechar func() error
}

// dentroDoCofre diz se caminhoCache está dentro da árvore de vaultPath.
//
// O cofre de referência fica em OneDrive, e um arquivo mapeado que o
// sincronizador mexe embaixo é uma classe de falha que este projeto ainda não
// pagou (ver CLAUDE.md, "O cofre fica em OneDrive"). O cache mora fora do
// cofre por decisão de configuração — esta função confere isso em tempo de
// execução, e não confia que a configuração está certa: um --cache-dir
// apontado para dentro do cofre por engano nunca deve resultar em mapear um
// arquivo que o OneDrive pode reescrever por baixo dos pés do processo.
func dentroDoCofre(caminhoCache, vaultPath string) bool {
	if vaultPath == "" {
		return false
	}
	cAbs, err1 := filepath.Abs(caminhoCache)
	vAbs, err2 := filepath.Abs(vaultPath)
	if err1 != nil || err2 != nil {
		// Caminho que não resolve: recusa por segurança, não assume nada.
		return true
	}
	rel, err := filepath.Rel(vAbs, cAbs)
	if err != nil {
		// Ex.: unidades diferentes no Windows — não há relação nenhuma entre
		// os dois caminhos, e sem relação não há como provar que está fora.
		return true
	}
	return rel == "." || !strings.HasPrefix(rel, "..")
}

// tentaAbrirArena tenta mapear a seção fixa de posições do arquivo em
// caminhoCache. Devolve ok=false sempre que mapear não for seguro ou não for
// possível — nunca mapeando lixo: quem chama cai para a decodificação
// integral de sempre (os.ReadFile + leCache), que já valida o arquivo por
// conta própria.
func tentaAbrirArena(caminhoCache, vaultPath string) (*arenaPosicoes, bool) {
	if dentroDoCofre(caminhoCache, vaultPath) {
		return nil, false
	}

	f, err := os.Open(caminhoCache)
	if err != nil {
		return nil, false
	}
	fi, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, false
	}
	tamanho := fi.Size()
	if tamanho < footerBytes {
		// Arquivo pequeno demais para ter rodapé — cache de formato anterior,
		// truncado, ou vazio. Não é este código que decide "corrompido": a
		// decodificação integral, que vai rodar em seguida, é quem confere a
		// assinatura e devolve ErrCacheCorrupted se for o caso.
		_ = f.Close()
		return nil, false
	}

	dados, fechar, err := mapearArquivo(f, tamanho)
	if err != nil {
		// mapearArquivo fecha f em caso de erro, nas duas plataformas.
		return nil, false
	}

	offset, count, ok := leRodape(dados)
	if !ok {
		_ = fechar()
		return nil, false
	}
	if count > limitePosicoes {
		_ = fechar()
		return nil, false
	}
	fim := offset + count*16
	if offset%8 != 0 || fim < offset || fim > uint64(len(dados)) {
		_ = fechar()
		return nil, false
	}

	var pos []TokenPosition
	if count > 0 {
		// unsafe.Slice reinterpreta os bytes mapeados como []TokenPosition,
		// sem copiar. Seguro porque: TokenPosition é dois int64 sem padding
		// (16 bytes exatos), o offset dentro do arquivo é múltiplo de 8 por
		// construção (ver o padding em escreveCache), e a base do mapeamento
		// já vem alinhada por página do sistema operacional — então o
		// ponteiro resultante está alinhado em 8 bytes.
		pos = unsafe.Slice((*TokenPosition)(unsafe.Pointer(&dados[offset])), count)
	}

	return &arenaPosicoes{dados: dados, pos: pos, fechar: fechar}, true
}

// leRodape lê os últimos footerBytes de dados e confere a assinatura.
//
// ok=false cobre tanto "não há rodapé" (arquivo de formato anterior, ou sem
// os últimos bytes reservados) quanto "os últimos bytes não são o rodapé"
// (colisão praticamente impossível, mas tratada como ausência e não como
// corrupção — a decodificação integral decide se o arquivo está corrompido).
func leRodape(dados []byte) (offset, count uint64, ok bool) {
	if len(dados) < footerBytes {
		return 0, 0, false
	}
	rodape := dados[len(dados)-footerBytes:]
	var assinatura [8]byte
	copy(assinatura[:], rodape[:8])
	if assinatura != footerMagic {
		return 0, 0, false
	}
	offset = binary.LittleEndian.Uint64(rodape[8:16])
	count = binary.LittleEndian.Uint64(rodape[16:24])
	return offset, count, true
}

// promoverArenaSePresente troca `pos` de uma arena mapeada por uma cópia no
// heap e desfaz o mapeamento, se o base do índice vier de uma arena.
//
// Chamada por SaveInvertedCache ANTES do os.Rename do salvamento atômico: o
// rename falha no Windows (ERROR_SHARING_VIOLATION) se o processo ainda tem o
// arquivo de destino mapeado. Promover para o heap primeiro custa uma cópia —
// ~291 MB no cofre de referência — mas só no caminho raro em que isto pode
// acontecer: cache PARCIAL retomado (ver invertedCacheState em
// cmd/gobsidian/serve.go) e depois regravado por buildInvertedIndex. No
// caminho comum — cache completo, "pronta" — SaveInvertedCache nunca roda de
// novo neste processo depois da carga, então esta função não faz nada.
//
// Sem isto, o rename falharia sempre que o servidor retomasse um cache
// parcial mapeado e depois o regravasse — o sintoma seria "falha ao salvar
// cache invertido de busca" no log a cada encerramento de construção, e o
// cache no disco nunca avançaria além do estado parcial.
func promoverArenaSePresente(inv *Inverted) {
	if inv == nil {
		return
	}
	inv.mu.Lock()
	var fechar func() error
	if inv.base != nil && inv.base.arenaFechar != nil {
		copiado := make([]TokenPosition, len(inv.base.pos))
		copy(copiado, inv.base.pos)
		inv.base.pos = copiado
		fechar = inv.base.arenaFechar
		inv.base.arenaFechar = nil
	}
	inv.mu.Unlock()

	if fechar != nil {
		_ = fechar()
	}
}
