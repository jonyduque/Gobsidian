// Package instalar e o instalador do produto, dentro do proprio produto.
//
// Ate 2026-09-08 a instalacao morava em `install.ps1` (729 linhas) e
// `installer/install.js` (1 009) -- a MESMA logica escrita duas vezes, em duas
// linguagens, sem um unico teste. E era o codigo que encerra os processos do
// usuario para poder substituir o binario.
//
// Aqui ela e uma conta so, na linguagem que o resto do produto ja testa. Ver
// docs/superpowers/specs/2026-09-08-instalador-e-encerramento-design.md,
// decisoes D-01 a D-11.
package instalar

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jonyd/gobsidian/internal/daemon"
)

// SufixoDePresenca nomeia os arquivos que este pacote interpreta. O diretorio
// de runtime tambem guarda `.lock`, `.sock` e `.sock.log`, e contar qualquer um
// deles como processo faria o instalador anunciar processos que nao existem --
// e pedir para mata-los.
const SufixoDePresenca = ".presenca"

// Presenca e um processo do produto que estava vivo quando alguem perguntou.
//
// Arquivo nao vai para o JSON: ele e o caminho de onde a presenca foi lida, e
// gravar o proprio caminho dentro do arquivo seria uma segunda conta do mesmo
// fato -- que diverge no dia em que o diretorio de runtime mudar.
type Presenca struct {
	PID     int    `json:"pid"`
	Cofre   string `json:"cofre"`
	Papel   string `json:"papel"`
	Versao  string `json:"versao"`
	Arquivo string `json:"-"`
}

// Registrar anuncia este processo enquanto ele viver.
//
// # Por que trava de kernel, e nao enumeracao de processos
//
// Go nao tem listagem de processos portavel, e o CLAUDE.md proibe
// `if runtime.GOOS ==` em logica compartilhada: responder "quem esta rodando?"
// pelo sistema operacional custaria codigo de plataforma em tres variantes,
// para uma pergunta que o produto ja sabe responder de outro jeito.
//
// internal/daemon/trava.go ja resolveu a parte dificil em 2026-08-31, e o
// comentario de la explica o que se ganhou: com a trava do kernel nao existe
// arquivo obsoleto, nem PID a parsear, nem sondagem de vitalidade. Um processo
// que segura a trava do proprio arquivo enquanto vive e, ao mesmo tempo,
// DETECTAVEL (a trava esta tomada) e LEGIVEL (o conteudo diz quem ele e). E
// legivel porque aquela trava cobre a faixa em 1<<62, longe dos dados,
// escolhida assim justamente para outro processo poder ler o arquivo.
//
// Morrer solta a trava, pelo kernel. Nao ha caminho de recuperacao a escrever,
// e portanto nao ha a classe de corrida inteira.
//
// O nome leva o PID porque um processo serve um cofre: dois processos nunca
// disputam o mesmo arquivo, e um PID reciclado apenas reaproveita o arquivo de
// um morto, que e o comportamento desejado.
func Registrar(runtimeDir, cofre, papel, versao string) (liberar func(), err error) {
	if err := os.MkdirAll(runtimeDir, 0o700); err != nil {
		return nil, fmt.Errorf("criando diretorio de runtime: %w", err)
	}

	caminho := CaminhoDePresenca(runtimeDir, papel, os.Getpid())
	trava, tomou, err := daemon.TentarTravar(caminho)
	if err != nil {
		return nil, fmt.Errorf("travando presenca: %w", err)
	}
	if !tomou {
		// So acontece se outro processo VIVO tiver exatamente este PID, o que
		// o sistema operacional nao permite. Se acontecer, nao registrar e
		// melhor que registrar por cima de alguem.
		return nil, fmt.Errorf("presenca %s ja esta travada por outro processo", caminho)
	}

	conteudo, err := json.Marshal(Presenca{
		PID:    os.Getpid(),
		Cofre:  cofre,
		Papel:  papel,
		Versao: versao,
	})
	if err != nil {
		trava.Liberar()
		return nil, fmt.Errorf("serializando presenca: %w", err)
	}
	if err := trava.Gravar(conteudo); err != nil {
		trava.Liberar()
		return nil, fmt.Errorf("gravando presenca: %w", err)
	}

	// A liberacao solta a trava E REMOVE o arquivo.
	//
	// Aqui remover e seguro, e a diferenca em relacao a internal/daemon/trava.go
	// -- que nunca remove -- e concreta: aquele arquivo tem nome FIXO por cofre,
	// e entre remover e recriar qualquer um entra, que era a origem das corridas
	// do esquema antigo. Este tem o PID no nome: ninguem mais disputa este
	// caminho, nunca.
	//
	// E remover importa. Medido em 2026-09-09, rodando o gate de orfaos com 100
	// ciclos: sem esta remocao o diretorio de runtime foi de 6 para 130 arquivos
	// de presenca -- a mesma forma do lixo de 960 `.lock` que a limpeza deste
	// mesmo pacote existe para varrer. Criar a segunda versao do problema que
	// se acabou de consertar seria dificil de defender.
	//
	// A ordem e obrigatoria: soltar a trava (que fecha o descritor) ANTES de
	// remover. O Windows recusa apagar um arquivo que o proprio processo mantem
	// aberto -- foi assim que um teste deste plano passou pelo motivo errado.
	//
	// Morte abrupta nao roda isto, e nao precisa: o arquivo fica sem trava, e e
	// exatamente o que Limpar reconhece como orfao.
	return func() {
		trava.Liberar()
		_ = os.Remove(caminho)
	}, nil
}

// presencaViva guarda a trava do processo corrente pela vida dele inteira.
//
// Nao e cache nem conveniencia: e o que impede o COLETOR DE LIXO de soltar a
// trava com o processo ainda vivo. A trava e um *os.File por baixo, e o Go
// registra um finalizador que FECHA o descritor quando o valor deixa de ser
// alcancavel -- e fechar o descritor solta a trava do kernel. Uma presenca
// registrada e depois esquecida some sozinha, em momento imprevisivel, e o
// instalador passaria a nao ver um processo que esta rodando.
var presencaViva func()

// RegistrarAteMorrer registra a presenca deste processo e a mantem ate ele
// morrer, sem devolver nada para o chamador guardar.
//
// Existe porque `runServe` termina em os.Exit por desenho (o codigo de saida e
// a decisao dele, nao o que o cobra derivaria de um error), e `defer` nao roda
// depois de os.Exit -- o golangci-lint acusou exatamente isso. Aqui nao ha o
// que deferir: quem solta a trava e o KERNEL, quando o processo morre, que e o
// mecanismo inteiro deste desenho (ver internal/daemon/trava.go).
//
// Erro nao e propagado de proposito: presenca e diagnostico. Um diretorio de
// runtime inacessivel nao pode impedir o produto de servir um cofre.
func RegistrarAteMorrer(runtimeDir, cofre, papel, versao string) error {
	liberar, err := Registrar(runtimeDir, cofre, papel, versao)
	if err != nil {
		return err
	}
	presencaViva = liberar
	return nil
}

// LiberarPresenca solta a presenca deste processo e remove o arquivo dela.
//
// Producao CHAMA isto: `serve` antes do os.Exit e `daemon` por defer. Sem essas
// duas chamadas o arquivo fica para sempre, e o gate de orfaos mediu o custo em
// 2026-09-09 -- 100 ciclos levaram o diretorio de runtime de 6 para 130
// arquivos de presenca, a mesma forma do lixo de 960 `.lock` que a limpeza
// deste pacote existe para varrer.
//
// Morte abrupta nao chega aqui, e nao precisa: o arquivo fica sem trava, que e
// o que Limpar reconhece como orfao.
//
// Idempotente: chamar duas vezes nao pode entrar em panic, pela mesma razao que
// o desarmar de lifecycle.ArmarGuardaChuva.
func LiberarPresenca() {
	if presencaViva == nil {
		return
	}
	presencaViva()
	presencaViva = nil
}

// CaminhoDePresenca e a conta unica do nome do arquivo. Registrar o escreve e a
// limpeza o reconhece; duas contas do mesmo nome concordam por coincidencia ate
// uma delas mudar sozinha (a licao do byAlias, em config.VaultKey).
func CaminhoDePresenca(runtimeDir, papel string, pid int) string {
	return filepath.Join(runtimeDir, fmt.Sprintf("%s.%d%s", papel, pid, SufixoDePresenca))
}

// Vivos lista os processos do produto que estao rodando agora.
//
// Diretorio ausente devolve lista vazia, e nao erro: "ninguem rodando" e a
// resposta certa para quem nunca rodou nada, e tanto `doctor` quanto o
// instalador perguntam antes de existir diretorio.
//
// Um arquivo de presenca cuja trava esta LIVRE nao conta. O arquivo sobrevive
// ao processo de proposito -- e o mesmo desenho de trava.go, que nunca remove o
// lock porque remover era a origem de toda corrida. Quem decide e a trava,
// nunca a existencia: em 2026-09-08 o diretorio de runtime do dono tinha 960
// arquivos orfaos, e contar arquivos teria reportado 960 processos.
func Vivos(runtimeDir string) ([]Presenca, error) {
	presencas, _, err := lerPresencas(runtimeDir)
	return presencas, err
}

// PresencasOrfas lista os arquivos de presenca cuja trava esta livre -- lixo
// que a limpeza pode remover com prova, e nao por heuristica.
func PresencasOrfas(runtimeDir string) ([]string, error) {
	_, orfas, err := lerPresencas(runtimeDir)
	return orfas, err
}

// lerPresencas percorre o diretorio uma vez e responde as duas perguntas.
//
// Uma passada, e nao duas: entre uma varredura e outra um processo pode nascer
// ou morrer, e a limpeza decidiria remover um arquivo que acabou de ganhar
// dono. Perguntar as duas coisas no mesmo passo nao elimina a corrida -- nada
// elimina --, mas a estreita para o intervalo entre duas linhas.
func lerPresencas(runtimeDir string) (vivos []Presenca, orfas []string, err error) {
	entradas, err := os.ReadDir(runtimeDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("lendo diretorio de runtime %s: %w", runtimeDir, err)
	}

	for _, e := range entradas {
		if e.IsDir() || !strings.HasSuffix(e.Name(), SufixoDePresenca) {
			continue
		}
		caminho := filepath.Join(runtimeDir, e.Name())

		emUso, err := daemon.TravaEmUso(caminho)
		if err != nil {
			// Nao poder perguntar nao autoriza a concluir. Tratar como VIVO e
			// o lado conservador: no maximo o instalador mostra um processo a
			// mais para o usuario decidir, em vez de a limpeza apagar o
			// arquivo de alguem que esta rodando.
			vivos = append(vivos, Presenca{Arquivo: caminho})
			continue
		}
		if !emUso {
			orfas = append(orfas, caminho)
			continue
		}

		p, err := lerPresenca(caminho)
		if err != nil {
			// Travado, logo vivo -- mesmo que o conteudo esteja ilegivel. O
			// conteudo e diagnostico; a trava e a prova.
			vivos = append(vivos, Presenca{Arquivo: caminho})
			continue
		}
		vivos = append(vivos, p)
	}

	// Ordem estavel: a saida vai para a tela do usuario, que vai ler a lista
	// duas vezes -- antes e depois de autorizar o encerramento.
	sort.Slice(vivos, func(i, j int) bool { return vivos[i].Arquivo < vivos[j].Arquivo })
	sort.Strings(orfas)
	return vivos, orfas, nil
}

func lerPresenca(caminho string) (Presenca, error) {
	b, err := os.ReadFile(caminho)
	if err != nil {
		return Presenca{}, err
	}
	var p Presenca
	if err := json.Unmarshal(b, &p); err != nil {
		return Presenca{}, err
	}
	p.Arquivo = caminho
	return p, nil
}
