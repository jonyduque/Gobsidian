package instalar

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jonyd/gobsidian/internal/config"
	"github.com/jonyd/gobsidian/internal/daemon"
	"github.com/jonyd/gobsidian/internal/ipc"
	"github.com/jonyd/gobsidian/internal/search"
)

// TetoDoLog e o tamanho a partir do qual um log e rotacionado, e nao apagado.
// Mesmo valor que cmd/gobsidian usa ao abrir o log do daemon; aqui ele alcanca
// os logs de cofres que nao estao rodando agora.
const TetoDoLog = 5 << 20

// Relatorio e o que a limpeza encontrou -- e, se aplicar for verdadeiro, o que
// ela fez.
//
// Listas de caminhos, e nao contagens: o usuario vai ler isto antes de
// autorizar, e "12 arquivos" nao permite discordar de nenhum deles.
type Relatorio struct {
	Locks            []string
	Sockets          []string
	Presencas        []string
	Caches           []string
	LogsRotacionados []string
	Bytes            int64
	// NaoRemovidos nomeia o que a limpeza QUIS remover e nao conseguiu. Sem
	// isso, uma limpeza que falha em tudo produz a mesma saida de uma que nao
	// tinha nada a fazer -- a distincao que o achado P11 registra para a
	// varredura de temporarios.
	NaoRemovidos []string
}

// Vazio diz se nao ha nada a fazer.
func (r Relatorio) Vazio() bool {
	return len(r.Locks)+len(r.Sockets)+len(r.Presencas)+len(r.Caches)+len(r.LogsRotacionados) == 0
}

// Limpar percorre o diretorio de runtime e a raiz do cache e remove SO o que e
// comprovadamente orfao (decisao D-05 do dono, 2026-09-08).
//
// Com aplicar=false ela nao toca em nada: e o modo que `doctor` usa para
// mostrar o que existe. Com aplicar=true e o que `install` e `update` fazem, e
// eles rodam sob a trava global, com nenhum processo do produto no ar.
//
// # A regra, e por que ela e mecanica
//
// Nada aqui pergunta "isto parece velho?". Cada remocao tem uma prova:
//
//	.lock, .presenca   a trava esta LIVRE  (daemon.TravaEmUso)
//	.sock              ninguem escuta      (ipc.AlguemEscuta)
//	cache de cofre     o cofre do cabecalho do cache NAO existe mais
//	.log               nunca e apagado -- passa do teto, e rotacionado
//
// Cache de cofre EXISTENTE nunca e tocado. Reconstruir custou 3021 ms no cofre
// de referencia do dono (3 257 notas, medido em 2026-09-07), e apagar cache de
// quem esta usando o produto e o tipo de "limpeza" que so cria trabalho.
//
// # Remover um .lock aqui nao contradiz internal/daemon/trava.go
//
// Aquele comentario diz que o arquivo de trava NUNCA e removido, e a razao e
// concreta: entre remover e recriar, qualquer um entra, e era dai que vinham as
// corridas do esquema antigo. Isso vale durante a OPERACAO. Aqui e outra
// situacao: a limpeza roda sob a trava global de instalacao, com nenhum
// processo do produto no ar, e so sobre travas que ela mesma acabou de
// verificar como livres. Um .lock recriado na proxima partida e o
// comportamento normal.
func Limpar(runtimeDir, cacheRaiz string, aplicar bool) (Relatorio, error) {
	var r Relatorio

	cofrePorChave, err := cofresConhecidos(cacheRaiz)
	if err != nil {
		return r, err
	}

	if err := limparRuntime(runtimeDir, cofrePorChave, aplicar, &r); err != nil {
		return r, err
	}
	if err := limparCaches(cacheRaiz, cofrePorChave, aplicar, &r); err != nil {
		return r, err
	}

	sort.Strings(r.Locks)
	sort.Strings(r.Sockets)
	sort.Strings(r.Presencas)
	sort.Strings(r.Caches)
	sort.Strings(r.LogsRotacionados)
	sort.Strings(r.NaoRemovidos)
	return r, nil
}

// estadoDoCofre e o que se sabe sobre o cofre por tras de uma chave.
type estadoDoCofre int

const (
	// cofreDesconhecido: nao ha cache para essa chave, entao nada nesta
	// maquina diz de qual cofre ela e. Nao da para provar que sumiu.
	cofreDesconhecido estadoDoCofre = iota
	cofreVivo
	cofreSumido
)

// cofresConhecidos mapeia chave -> estado, lendo o cabecalho de cada cache.
//
// O diretorio de cache e nomeado pelo HASH do caminho do cofre
// (config.VaultKey), e hash nao volta para o caminho. Quem guarda o caminho e o
// cabecalho do proprio cache (search.CacheHeader.VaultPath), desde sempre --
// so faltava alguem le-lo.
func cofresConhecidos(cacheRaiz string) (map[string]estadoDoCofre, error) {
	porChave := map[string]estadoDoCofre{}

	entradas, err := os.ReadDir(cacheRaiz)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return porChave, nil
		}
		return nil, fmt.Errorf("lendo raiz do cache %s: %w", cacheRaiz, err)
	}

	for _, e := range entradas {
		if !e.IsDir() {
			continue
		}
		h, err := search.LerCabecalhoDoCache(filepath.Join(cacheRaiz, e.Name()))
		if err != nil || h.VaultPath == "" {
			// Cache ausente, de outra versao ou ilegivel: nao prova nada sobre
			// o cofre. Desconhecido, e portanto intocado.
			porChave[e.Name()] = cofreDesconhecido
			continue
		}
		if _, err := os.Stat(h.VaultPath); err == nil {
			porChave[e.Name()] = cofreVivo
		} else if errors.Is(err, fs.ErrNotExist) {
			porChave[e.Name()] = cofreSumido
		} else {
			// Existe mas nao da para consultar (permissao, unidade de rede
			// fora do ar). Nao poder perguntar nao autoriza a concluir.
			porChave[e.Name()] = cofreDesconhecido
		}
	}
	return porChave, nil
}

func limparRuntime(runtimeDir string, cofres map[string]estadoDoCofre, aplicar bool, r *Relatorio) error {
	entradas, err := os.ReadDir(runtimeDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("lendo diretorio de runtime %s: %w", runtimeDir, err)
	}

	for _, e := range entradas {
		if e.IsDir() {
			continue
		}
		nome := e.Name()
		caminho := filepath.Join(runtimeDir, nome)

		switch {
		case strings.HasSuffix(nome, ".log"):
			rotacionarSePassar(caminho, aplicar, r)

		case strings.HasSuffix(nome, SufixoDePresenca):
			// Presenca de processo morto e SEMPRE orfa: nenhum processo a
			// segura, e ela e recriada a cada partida. Nao depende do cofre.
			if emUso, err := daemon.TravaEmUso(caminho); err != nil || emUso {
				continue
			}
			remover(caminho, aplicar, &r.Presencas, r)

		case daemon.EhArquivoDeTrava(nome):
			if emUso, err := daemon.TravaEmUso(caminho); err != nil || emUso {
				continue
			}
			if cofres[chaveDoNome(nome)] == cofreVivo {
				continue
			}
			remover(caminho, aplicar, &r.Locks, r)

		case strings.HasSuffix(nome, ".sock"):
			if ipc.AlguemEscuta(caminho) {
				continue
			}
			if cofres[chaveDoNome(nome)] == cofreVivo {
				continue
			}
			remover(caminho, aplicar, &r.Sockets, r)
		}
	}
	return nil
}

// chaveDoNome extrai a chave do cofre do nome de um arquivo do diretorio de
// runtime. Os nomes sao "<chave>.sock", "<chave>.sock.lock",
// "<chave>.sock.listen.lock" e "<chave>.sock.log" -- todos derivados do MESMO
// caminho de socket (ipc.SocketPath), entao a chave e sempre o primeiro
// segmento.
func chaveDoNome(nome string) string {
	if i := strings.IndexByte(nome, '.'); i > 0 {
		return nome[:i]
	}
	return nome
}

func limparCaches(cacheRaiz string, cofres map[string]estadoDoCofre, aplicar bool, r *Relatorio) error {
	for chave, estado := range cofres {
		if estado != cofreSumido {
			continue
		}
		dir := filepath.Join(cacheRaiz, chave)
		tamanho := tamanhoDaArvore(dir)
		if !aplicar {
			r.Caches = append(r.Caches, dir)
			r.Bytes += tamanho
			continue
		}
		if err := os.RemoveAll(dir); err != nil {
			r.NaoRemovidos = append(r.NaoRemovidos, fmt.Sprintf("%s: %v", dir, err))
			continue
		}
		r.Caches = append(r.Caches, dir)
		r.Bytes += tamanho
	}
	return nil
}

func remover(caminho string, aplicar bool, lista *[]string, r *Relatorio) {
	fi, err := os.Stat(caminho)
	var tamanho int64
	if err == nil {
		tamanho = fi.Size()
	}
	if !aplicar {
		*lista = append(*lista, caminho)
		r.Bytes += tamanho
		return
	}
	if err := os.Remove(caminho); err != nil {
		r.NaoRemovidos = append(r.NaoRemovidos, fmt.Sprintf("%s: %v", caminho, err))
		return
	}
	*lista = append(*lista, caminho)
	r.Bytes += tamanho
}

// rotacionarSePassar guarda o log como ".1" quando ele passa do teto. Nunca
// apaga: o log e a unica memoria do daemon, e a investigacao de 2026-09-08
// dependeu de linhas de 2026-08-24.
func rotacionarSePassar(caminho string, aplicar bool, r *Relatorio) {
	if strings.HasSuffix(caminho, ".log.1") {
		return
	}
	fi, err := os.Stat(caminho)
	if err != nil || fi.Size() < TetoDoLog {
		return
	}
	if !aplicar {
		r.LogsRotacionados = append(r.LogsRotacionados, caminho)
		return
	}
	if err := os.Rename(caminho, caminho+".1"); err != nil {
		r.NaoRemovidos = append(r.NaoRemovidos, fmt.Sprintf("%s: %v", caminho, err))
		return
	}
	r.LogsRotacionados = append(r.LogsRotacionados, caminho)
}

func tamanhoDaArvore(dir string) int64 {
	var total int64
	_ = filepath.WalkDir(dir, func(_ string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil //nolint:nilerr // subarvore ilegivel nao invalida a soma das outras
		}
		if fi, err := d.Info(); err == nil {
			total += fi.Size()
		}
		return nil
	})
	return total
}

// RaizDoCache delega para config: uma conta por regra. A limpeza precisa da
// raiz que contem um subdiretorio por cofre, e quem a define e config.
func RaizDoCache() string { return config.RaizDoCache() }
