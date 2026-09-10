package boot

import (
	"context"
	"errors"
	"log/slog"
	"runtime/debug"
	"time"

	"github.com/jonyduque/Gobsidian/internal/config"
	"github.com/jonyduque/Gobsidian/internal/index"
	"github.com/jonyduque/Gobsidian/internal/search"
	"github.com/jonyduque/Gobsidian/internal/vault"
)

// intervaloDeGravacao limita QUANTO TRABALHO se perde num encerramento
// abrupto durante a construcao.
//
// O que se quer limitar e tempo, nao contagem de notas: uma nota de 200 bytes
// e uma de 200 KB custam ordens de grandeza diferentes, e um contador de notas
// da garantias diferentes em cofres diferentes.
//
// A gravacao NAO e incremental por dentro — cada uma serializa o cache inteiro.
// Gravar a cada 250 notas custou +39% no tempo total de construcao (303 s
// contra 219 s), medido quando o cache do cofre de referencia tinha 472 MB no
// formato gob. Com o formato binario ele passou a 66 MB, entao esse custo so
// pode ter caido — mas nao foi medido de novo, e o intervalo de 60 s nao foi
// reajustado por causa disso.
//
// 60 s porque esta gravacao e a UNICA rede: o encerramento nao grava parcial
// (ver o caminho de cancelamento em construirBusca), entao o que se perde e
// o trabalho desde a ultima passagem por aqui.
const intervaloDeGravacao = 60 * time.Second

// estadoDoCache decide o que fazer com o que veio do disco.
//
// Existe separada porque e a regra que impede o defeito mais caro desta area:
// LoadInvertedCache confere versao de formato, de parser, de analisador e o
// caminho do cofre — NAO confere cobertura. Como a construcao grava parciais,
// aceitar o cache sem olhar a contagem faria a busca servir um indice
// incompleto como se fosse completo, e o cliente receberia menos notas do que
// existem sem nada indicando isso.
//
// pronta  -> o cache cobre o cofre inteiro
// retomar -> o cache e utilizavel mas incompleto; serve de ponto de partida
//
// A comparacao e >= e nao ==: notas APAGADAS do cofre deixam entradas velhas no
// cache, e nesse caso ele cobre tudo o que existe. As sobras nao vazam para o
// resultado porque a busca so devolve o que o indice de metadados confirma.
func estadoDoCache(hdr *search.CacheHeader, err error, noteCount int) (pronta, retomar bool) {
	if err != nil || hdr == nil {
		return false, false
	}
	if hdr.NoteCount >= noteCount {
		return true, false
	}
	return false, true
}

// devolveMemoriaTransitoria devolve ao sistema o que a montagem do indice usou e
// nao precisa mais.
//
// Montar o indice do cofre de referencia aloca 737 MB para deixar ~500 MB
// vivos. A diferenca e transitoria — a fatia com o arquivo inteiro (70 MB) e a
// tabela de faixas da arena (~120 MB) viram lixo assim que o indice fica
// montado — mas o Go devolve essas paginas ao sistema no tempo dele, e ate la
// elas aparecem no RSS de um servidor que ficara horas em repouso ao lado do
// editor do usuario.
//
// Medido no cofre de referencia, RSS em repouso 22 s depois da partida, tres
// partidas por braco:
//
//	sem esta chamada  782,8  783,0  779,5 MB
//	com esta chamada  627,4  587,1  588,6 MB
//
// Cerca de -195 MB na mediana, com as distribuicoes sem sobreposicao. RNF-07
// mede exatamente isto. Custo medido: 67, 70 e 73 ms.
//
// Roda DEPOIS de o indice ficar pronto e ANTES de w.Run, porque forca um ciclo
// completo de GC. Os eventos do periodo nao se perdem: watcher.New ja registrou
// os watches e eles ficam enfileirados no fsnotify.
//
// Chamada por `defer` e nao no fim do corpo: o caminho de cache completo — o
// mais rapido e o mais comum — sai por um `return` antes do fim, e e nele que
// ha mais memoria transitoria para devolver. As medicoes acima sao desse
// caminho.
//
// GOGC ficou como esta. GOGC=400 durante a carga mediu -5,8% na mediana do
// carregamento em 12 partidas por braco, e o U de Mann-Whitney deu 88 contra
// uma regiao critica de 37/107 — nao significativo. E o RSS em repouso ficou
// IGUAL ou pior (829,5 / 794,2 / 789,1 MB), porque um alvo de heap maior e
// exatamente isso.
func devolveMemoriaTransitoria(log *slog.Logger) {
	inicio := time.Now()
	debug.FreeOSMemory()
	log.Debug("memoria transitoria da montagem do indice devolvida",
		"duracao_ms", time.Since(inicio).Milliseconds())
}

// PrepararBusca carrega o cache e completa o que faltar.
//
// Roda em segundo plano, fora do caminho de boot. Devolve com inv pronto ou com
// o contexto cancelado; em nenhum outro caso o indice fica marcado em
// construcao para sempre.
//
// Precisa rodar ANTES de o watcher comecar a consumir eventos: a adocao do
// cache substitui o conteudo de inv. Ver search.Inverted.AdotarDe.
func PrepararBusca(
	ctx context.Context,
	v *vault.Vault,
	idx *index.Index,
	inv *search.Inverted,
	cfg config.Config,
	log *slog.Logger,
) {
	defer devolveMemoriaTransitoria(log)

	inicio := time.Now()
	doCache, hdr, err := search.LoadInvertedCache(ctx, cfg.CacheDir, cfg.VaultPath)
	pronta, retomar := estadoDoCache(hdr, err, idx.NoteCount())

	// adotado separa "o cache nao servia" de "o cache servia mas nao pode ser
	// aplicado". O segundo caso so acontece se alguem escreveu no indice antes
	// desta funcao, o que a ordem em Montar impede — e se a ordem mudar, o
	// recuo e construir do zero, nunca aplicar por cima.
	adotado := func() bool {
		if err := inv.AdotarDe(doCache); err != nil {
			log.Warn("cache de busca nao aplicado; construindo do zero", "err", err)
			// doCache pode ter uma arena de posicoes mapeada (Task 89) que
			// ninguem mais vai usar, ja que a adocao foi recusada. Sem isto
			// o mapeamento fica aberto ate o processo terminar.
			if cerr := doCache.Close(); cerr != nil {
				log.Warn("falha ao liberar cache de busca nao adotado", "err", cerr)
			}
			return false
		}
		return true
	}

	switch {
	case pronta && adotado():
		inv.MarkReady()
		log.Info("indice de busca pronto",
			"origem", "cache",
			"notas", idx.NoteCount(),
			"duracao_ms", time.Since(inicio).Milliseconds())
		return
	case retomar && adotado():
		log.Info("cache de busca parcial encontrado; retomando",
			"no_cache", hdr.NoteCount, "no_cofre", idx.NoteCount(),
			"carga_ms", time.Since(inicio).Milliseconds())
	default:
		// Cache ausente, de versao incompativel ou corrompido. Os tres levam ao
		// mesmo lugar — construir do zero — e a distincao entre eles ja saiu no
		// log de LoadInvertedCache.
		if err != nil && !errors.Is(err, search.ErrCacheNotFound) {
			log.Warn("cache de busca descartado", "err", err)
		}
	}

	construirBusca(ctx, v, idx, inv, cfg, log)
}

// construirBusca tokeniza o cofre para dentro de inv e o marca pronto.
//
// Roda em segundo plano: ver o comentario no ponto de chamada. Marca pronto
// mesmo com notas individuais falhando — uma nota ilegivel nao pode deixar a
// busca desligada para sempre; ela aparece no log e fica de fora do indice.
func construirBusca(
	ctx context.Context,
	v *vault.Vault,
	idx *index.Index,
	inv *search.Inverted,
	cfg config.Config,
	log *slog.Logger,
) {
	inicio := time.Now()
	caminhos := idx.NotePaths()
	log.Info("construindo indice de busca em segundo plano", "notas", len(caminhos))

	salvar := func(parcial bool) {
		if err := search.SaveInvertedCache(ctx, cfg.CacheDir, cfg.VaultPath, inv); err != nil {
			log.Warn("falha ao salvar cache invertido de busca", "parcial", parcial, "err", err)
		}
	}

	feitas := 0
	jaCobertas := 0
	ultimoSalvo := time.Now()
	for _, p := range caminhos {
		// Retomada: o que o cache parcial ja trouxe nao e relido.
		//
		// HasDoc e nao DocLength > 0: uma nota vazia tem tamanho zero e estar
		// no indice, e a versao antiga a relia a cada retomada sem nunca
		// passar a conta-la como coberta.
		if inv.HasDoc(string(p)) {
			jaCobertas++
			feitas++
			continue
		}
		if ctx.Err() != nil {
			// Sai sem gravar, de proposito.
			//
			// SaveInvertedCache confere o context so na ENTRADA: uma vez comecada,
			// a gravacao nao e interrompivel, e mediu ate 10 s num encerramento
			// real quando o cache tinha 472 MB no formato gob. Hoje ele tem
			// 66 MB; o raciocinio nao depende do numero, so de a gravacao nao
			// ser interrompivel. serveEmProcesso faz a espera das goroutines de
			// fundo DEPOIS de lifecycle.Shutdown, entao essa espera nao passa por
			// orcamento nenhum — e o harness de orfaos conta como orfao o que nao
			// morre em 8 s. Um WithTimeout aqui impediria COMECAR, nunca
			// terminar: seria um orcamento decorativo.
			//
			// O que limita a perda e a gravacao periodica acima. Perder ate um
			// intervalo de trabalho e barato; furar o requisito de bloqueio de
			// release nao e.
			log.Warn("construcao do indice de busca interrompida",
				"feitas", feitas, "de", len(caminhos),
				"perdido_desde_ultima_gravacao_ms", time.Since(ultimoSalvo).Milliseconds())
			return
		}
		// Indexa pelo caminho GUARDADO, nunca lendo aqui.
		//
		// Ate 2026-08-26 este laco fazia v.ReadAll + inv.Add. v.ReadAll e um
		// os.ReadFile puro, sem consulta a CloudOnly, e idx.NotePaths()
		// devolve placeholders — um `.md` somente-nuvem E nota do indice de
		// metadados. O boot escapava assim da guarda inteira que mora em
		// Inverted.Update, e num cofre OneDrive sem cache valido baixava todo
		// placeholder em segundo plano.
		//
		// Havia duas construcoes do mesmo indice, uma com guarda (o CLI, em
		// search.go) e outra sem (esta). Agora ha um caminho so.
		if err := inv.Update(ctx, v, p); err != nil {
			log.Warn("falha ao indexar nota ao construir indice invertido de busca", "path", p, "err", err)
			continue
		}

		feitas++
		if time.Since(ultimoSalvo) >= intervaloDeGravacao {
			salvar(true)
			ultimoSalvo = time.Now()
			log.Debug("cache de busca gravado parcialmente", "feitas", feitas, "de", len(caminhos))
		}
	}

	salvar(false)
	inv.MarkReady()
	log.Info("indice de busca pronto",
		"origem", "construcao",
		"notas", feitas,
		"reaproveitadas_do_cache", jaCobertas,
		"duracao_ms", time.Since(inicio).Milliseconds())
}
