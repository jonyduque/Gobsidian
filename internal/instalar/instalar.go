package instalar

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/jonyd/gobsidian/internal/hosts"
)

// Sistema sao as operacoes que tocam a maquina do usuario.
//
// Elas ficam numa struct de funcoes para que a SEQUENCIA seja testavel sem
// encerrar processo nenhum e sem mover arquivo nenhum. Os 1 738 linhas de
// instalador que este pacote substitui nao tinham teste, e sao justamente o
// codigo que mata processos do usuario -- essa combinacao e o que este desenho
// existe para desfazer.
type Sistema struct {
	// Encerrar mata um processo pelo PID.
	Encerrar func(pid int) error
	// Confirmar pergunta ao usuario. Devolve false quando ele recusa, e recusa
	// ABORTA a instalacao (decisao D-06).
	Confirmar func(pergunta string, itens []string) bool
	// AjustarPath acrescenta um diretorio ao PATH do usuario.
	//
	// Injetavel porque a implementacao real escreve no REGISTRO do Windows
	// (HKCU\Environment) ou no ~/.profile: um teste que a chamasse de verdade
	// alteraria o ambiente de quem roda a suite. Nulo usa AdicionarAoPath.
	AjustarPath func(dir string) (bool, error)

	// Agora existe para o teste nao depender do relogio.
	Agora func() time.Time
}

// SistemaReal e o Sistema que age de verdade.
func SistemaReal(confirmar func(string, []string) bool) Sistema {
	return Sistema{
		Encerrar: func(pid int) error {
			p, err := os.FindProcess(pid)
			if err != nil {
				return err
			}
			return p.Kill()
		},
		Confirmar:   confirmar,
		AjustarPath: AdicionarAoPath,
		Agora:       time.Now,
	}
}

// Opcoes e o que o usuario pediu.
type Opcoes struct {
	// Origem e o executavel a instalar. Vazio usa os.Executable() -- o caso
	// normal, em que o binario baixado se instala.
	Origem string
	// Destino e o diretorio de instalacao. Vazio usa DiretorioPadrao().
	Destino string

	Versao   string
	Cofre    string
	Hosts    []string // chaves; vazio significa "os detectados"
	ReadOnly bool
	SemPath  bool

	// Ambiente permite ao teste montar um mundo. Zero-valor usa o real.
	Ambiente *hosts.Ambiente
	// RuntimeDir e CacheRaiz idem; vazios usam os reais.
	RuntimeDir string
	CacheRaiz  string
}

// Resultado e o que aconteceu, para o comando imprimir.
type Resultado struct {
	Binario     string
	VersaoHash  string
	PathMudou   bool
	Encerrados  []Presenca
	HostsOK     map[string]string // chave -> aviso
	HostsFalhos map[string]string // chave -> erro
	Limpeza     Relatorio
	// ChavesMigradas lista os diretorios de cache cujo nome ficou para tras da
	// conta de config.VaultKey. Vazio e o caso normal -- ver MigrarChaves.
	ChavesMigradas []MigracaoDeChave
}

// ErrRecusado indica que o usuario disse nao. Nao e falha: e decisao dele.
var ErrRecusado = errors.New("instalacao recusada pelo usuario")

// Instalar executa a sequencia inteira, sob a trava global.
//
// A ordem NAO e livre, e cada passo depende do anterior:
//
//  1. trava global      a partir daqui nenhum serve ou daemon sobe
//  2. listar e perguntar   D-06: PID e cofre na tela, recusa aborta
//  3. encerrar          o que estava rodando com o binario antigo
//  4. limpar            lixo comprovadamente orfao (D-05)
//  5. instalar binario  por rename, porque o executavel corrente pode ser o
//     proprio arquivo a substituir (medido em 2026-09-08:
//     sobrescrever falha, renomear funciona)
//  6. PATH e hosts
//  7. manifesto
//
// A trava e solta pelo defer, inclusive nos ramos de erro: uma trava esquecida
// tomada impede todo `serve` seguinte de subir.
func Instalar(ctx context.Context, sis Sistema, o Opcoes) (Resultado, error) {
	var r Resultado

	runtimeDir := o.RuntimeDir
	if runtimeDir == "" {
		dir, err := DiretorioDeRuntime()
		if err != nil {
			return r, fmt.Errorf("resolvendo o diretorio de runtime: %w", err)
		}
		runtimeDir = dir
	}
	cacheRaiz := o.CacheRaiz
	if cacheRaiz == "" {
		cacheRaiz = RaizDoCache()
	}

	liberarTrava, err := TomarTravaGlobal(runtimeDir)
	if err != nil {
		return r, err
	}
	defer liberarTrava()

	// 2 e 3: quem esta rodando, e o aval para encerrar.
	if err := encerrarProcessos(sis, runtimeDir, &r); err != nil {
		return r, err
	}

	// 4: limpeza. Roda AQUI -- com a trava tomada e ninguem rodando -- e nao
	// antes: e a unica janela em que remover uma trava livre nao corre com
	// alguem que esta prestes a toma-la.
	limpeza, err := Limpar(runtimeDir, cacheRaiz, true)
	if err != nil {
		return r, fmt.Errorf("limpando: %w", err)
	}
	r.Limpeza = limpeza

	// 4b: chaves de cache que ficaram para tras. Mesma janela da limpeza, e
	// pela mesma razao: renomear diretorio de cache exige que ninguem o esteja
	// mapeando. Ver MigrarChaves para o defeito que a originou -- a conta de
	// config.VaultKey deixou de depender de tabela Unicode da toolchain, e um
	// punhado de cofres muda de chave por isso.
	migradas, err := MigrarChaves(cacheRaiz, true)
	if err != nil {
		return r, fmt.Errorf("migrando chaves de cache: %w", err)
	}
	r.ChavesMigradas = migradas

	// 5: o binario.
	binario, hash, err := instalarBinario(o)
	if err != nil {
		return r, err
	}
	r.Binario = binario
	r.VersaoHash = hash

	// 6: PATH.
	if !o.SemPath {
		mudou, err := sis.ajustarPath(filepath.Dir(binario))
		if err != nil {
			return r, fmt.Errorf("ajustando o PATH: %w", err)
		}
		r.PathMudou = mudou
	}

	// 6: hosts.
	r.HostsOK, r.HostsFalhos = configurarHosts(o, binario)

	// 7: manifesto.
	m := Manifesto{
		Binario: binario,
		Versao:  o.Versao,
		Hash:    hash,
		Cofre:   o.Cofre,
		Em:      sis.agora(),
	}
	if r.PathMudou {
		m.PathAdicionado = filepath.Dir(binario)
	}
	for chave := range r.HostsOK {
		m.Hosts = append(m.Hosts, chave)
	}
	if err := GravarManifesto(m); err != nil {
		return r, err
	}

	return r, ctx.Err()
}

func (s Sistema) ajustarPath(dir string) (bool, error) {
	if s.AjustarPath != nil {
		return s.AjustarPath(dir)
	}
	return AdicionarAoPath(dir)
}

func (s Sistema) agora() time.Time {
	if s.Agora != nil {
		return s.Agora()
	}
	return time.Now()
}

// encerrarProcessos lista quem esta rodando, pergunta, e encerra.
//
// Perguntar ANTES e a metade que o instalador antigo aprendeu na pratica:
// encerrar sessao MCP sem avisar derruba trabalho em curso, e o usuario nao tem
// como saber que foi o instalador (docs/OPERACAO.md). A lista traz PID e cofre
// porque "3 processos" nao permite discordar de nenhum deles.
func encerrarProcessos(sis Sistema, runtimeDir string, r *Resultado) error {
	vivos, err := Vivos(runtimeDir)
	if err != nil {
		return fmt.Errorf("listando processos: %w", err)
	}
	// O proprio instalador nao esta na lista: ele nao registra presenca.
	if len(vivos) == 0 {
		return nil
	}

	itens := make([]string, 0, len(vivos))
	for _, p := range vivos {
		itens = append(itens, fmt.Sprintf("pid %d  %s  %s", p.PID, p.Papel, p.Cofre))
	}
	if sis.Confirmar != nil && !sis.Confirmar("Encerrar estes processos do gobsidian para trocar o binario?", itens) {
		return ErrRecusado
	}

	for _, p := range vivos {
		if p.PID <= 0 {
			continue
		}
		if err := sis.Encerrar(p.PID); err != nil {
			// Processo que ja morreu entre a listagem e aqui nao e falha.
			continue
		}
		r.Encerrados = append(r.Encerrados, p)
	}
	return nil
}

// instalarBinario poe o executavel no destino e devolve o caminho e o SHA-256.
//
// A troca e por RENAME, e nao e otimizacao. Medido em 2026-09-08 nesta
// plataforma: sobrescrever um .exe em execucao falha com "Device or resource
// busy"; renomea-lo funciona, o processo antigo segue vivo lendo do arquivo
// renomeado, e apagar o renomeado funciona mesmo com ele rodando. Como
// `gobsidian update` E o executavel que precisa substituir, renomear e o unico
// caminho.
func instalarBinario(o Opcoes) (caminho, hash string, err error) {
	origem := o.Origem
	if origem == "" {
		exe, err := os.Executable()
		if err != nil {
			return "", "", fmt.Errorf("resolvendo o executavel corrente: %w", err)
		}
		origem = exe
	}
	destinoDir := o.Destino
	if destinoDir == "" {
		destinoDir = DiretorioPadrao()
	}
	destino := filepath.Join(destinoDir, NomeDoExecutavel)

	if err := os.MkdirAll(destinoDir, 0o700); err != nil {
		return "", "", fmt.Errorf("criando %s: %w", destinoDir, err)
	}

	// Ja e o mesmo arquivo: nada a copiar. Acontece quando `install` roda a
	// partir do binario ja instalado.
	if mesmoArquivo(origem, destino) {
		h, err := somaDoArquivo(destino)
		return destino, h, err
	}

	// Tira o antigo do caminho ANTES de escrever. Se ele estiver em execucao,
	// sobrescrever falharia; renomear nao.
	if _, err := os.Stat(destino); err == nil {
		antigo := fmt.Sprintf("%s.antigo-%d", destino, time.Now().UnixNano())
		if err := os.Rename(destino, antigo); err != nil {
			return "", "", fmt.Errorf("tirando o binario antigo do caminho: %w", err)
		}
		// Best-effort: se o antigo ainda estiver mapeado, ele fica e a limpeza
		// da proxima instalacao o alcanca. O que NAO pode e bloquear a troca.
		_ = os.Remove(antigo)
	}

	if err := copiarArquivo(origem, destino); err != nil {
		return "", "", err
	}
	h, err := somaDoArquivo(destino)
	return destino, h, err
}

func mesmoArquivo(a, b string) bool {
	ia, err := os.Stat(a)
	if err != nil {
		return false
	}
	ib, err := os.Stat(b)
	if err != nil {
		return false
	}
	return os.SameFile(ia, ib)
}

func copiarArquivo(origem, destino string) error {
	src, err := os.Open(origem)
	if err != nil {
		return fmt.Errorf("abrindo %s: %w", origem, err)
	}
	defer func() { _ = src.Close() }()

	// 0o700: executavel para o dono, e so para ele. Fora do Windows o bit de
	// execucao e o que faz o arquivo poder rodar.
	dst, err := os.OpenFile(destino, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o700)
	if err != nil {
		return fmt.Errorf("criando %s: %w", destino, err)
	}
	if _, err := io.Copy(dst, src); err != nil {
		_ = dst.Close()
		return fmt.Errorf("copiando para %s: %w", destino, err)
	}
	if err := dst.Close(); err != nil {
		return fmt.Errorf("fechando %s: %w", destino, err)
	}
	return nil
}

func somaDoArquivo(caminho string) (string, error) {
	f, err := os.Open(caminho)
	if err != nil {
		return "", fmt.Errorf("abrindo %s para somar: %w", caminho, err)
	}
	defer func() { _ = f.Close() }()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("somando %s: %w", caminho, err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// configurarHosts registra o servidor nos hosts pedidos.
//
// Falha em UM host nao derruba os outros, e por isso ha dois mapas: um host que
// nao esta instalado, ou cujo CLI mudou de opcoes, nao pode impedir os demais
// de serem configurados. O instalador antigo ja tinha essa propriedade, e o
// comentario dele registra o caso concreto (codex, nao verificado).
func configurarHosts(o Opcoes, binario string) (ok, falhos map[string]string) {
	ok, falhos = map[string]string{}, map[string]string{}

	amb := hosts.AmbienteReal()
	if o.Ambiente != nil {
		amb = *o.Ambiente
	}

	args := []string{"serve", "--vault", o.Cofre}
	if o.ReadOnly {
		args = append(args, "--read-only")
	}
	entrada := hosts.Entrada{Command: binario, Args: args}

	// != nil, e nao len() > 0: uma fatia VAZIA significa "nenhum host", e uma
	// fatia NULA significa "detecte voce". Com len() > 0 as duas cairiam na
	// deteccao, e `--hosts none` configuraria tudo -- o oposto do que ele diz.
	//
	// A deteccao so roda no ramo que a USA: ela chama exec.LookPath e percorre
	// diretorios, e fazer isso para descartar o resultado e trabalho que o
	// usuario paga sem receber nada.
	var alvos []hosts.Host
	if o.Hosts == nil {
		alvos = hosts.Detectar(amb)
	} else {
		for _, chave := range o.Hosts {
			h, existe := hosts.PorChave(chave)
			if !existe {
				falhos[chave] = "host desconhecido"
				continue
			}
			alvos = append(alvos, h)
		}
	}

	for _, h := range alvos {
		aviso, err := h.Configurar(amb, entrada)
		if err != nil {
			falhos[h.Chave] = err.Error()
			continue
		}
		ok[h.Chave] = aviso
	}
	return ok, falhos
}

// ConfigurarHosts registra o servidor nos hosts pedidos, sem reinstalar nada.
//
// E o que `gobsidian vaults` chama: trocar de cofre ou acrescentar um host nao
// exige encerrar processo, mexer no PATH nem tocar no binario -- e uma
// instalacao inteira para isso seria desproporcional.
//
// chaves nil significa "os detectados"; uma fatia vazia significa "nenhum".
func ConfigurarHosts(binario, cofre string, readOnly bool, chaves []string) (ok, falhos map[string]string) {
	return configurarHosts(Opcoes{Cofre: cofre, ReadOnly: readOnly, Hosts: chaves}, binario)
}
