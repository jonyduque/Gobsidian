// Command sondahost mede o que um host MCP deixa um processo filho fazer com
// socket AF_UNIX, trava de kernel e diretorio novo -- as tres perguntas que
// decidem para onde o socket do daemon vai (plano
// docs/superpowers/plans/2026-09-11-resources-arvore-e-sdk.md, Parte G1).
//
// Nao e produto e nao entra no release. Existe porque o defeito que ela mede
// NAO se reproduz fora do host: em 2026-09-14 um processo Medium criado pelo
// Agendador de Tarefas, com e sem a identidade do pacote do Claude, conectava e
// removia socket normalmente, e so um servidor iniciado PELO PROPRIO Claude
// Desktop mostrou dial 10022 e lstat 1920 em %LOCALAPPDATA%.
//
// Uso, nesta ordem:
//
//  1. go build -o <exe> ./tools/sondahost
//  2. <exe> preparar   -- da shell; segura um socket em cada diretorio candidato
//     e uma trava compartilhada. Deixe rodando.
//  3. registre <exe>, sem argumentos, como servidor MCP no host e reinicie o
//     host. O processo que o host cria sonda e depois serve MCP sem tools.
//  4. <exe> ler        -- da shell; imprime o resultado e diz onde o diretorio
//     novo foi parar.
//  5. <exe> limpar     -- apaga o que a sonda criou.
//
// O resultado vai para stderr (o host grava no log do servidor) e para
// %USERPROFILE%\.gobsidian-sondahost\resultado.txt: fora de %LOCALAPPDATA%,
// onde a escrita do processo do host foi medida como real.
package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/jonyduque/Gobsidian/internal/daemon"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	nomeCruzado    = "sondahost-cruzado.sock"
	nomeTravaComum = "sondahost-compartilhada.lock"
	prefixoNovo    = "gobsidian-sondahost-novo-"
	marcaDiretorio = "diretorio-novo: "
)

// candidato e um diretorio onde o socket do daemon poderia morar.
type candidato struct {
	nome string
	dir  string
}

// ambiente resolve os caminhos a partir do perfil do processo que roda. Os dois
// lados -- preparar e o processo do host -- chegam aos mesmos caminhos porque
// herdam as mesmas variaveis de perfil (medido em 2026-09-14: LOCALAPPDATA,
// USERPROFILE e TEMP identicos dentro e fora do Claude Desktop).
type ambiente struct {
	perfil     string
	localApp   string
	base       string
	candidatos []candidato
	travaComum string
}

func novoAmbiente() (ambiente, error) {
	perfil, err := os.UserHomeDir()
	if err != nil {
		return ambiente{}, fmt.Errorf("resolvendo o perfil: %w", err)
	}
	localApp, err := os.UserCacheDir()
	if err != nil {
		return ambiente{}, fmt.Errorf("resolvendo LOCALAPPDATA: %w", err)
	}
	return ambiente{
		perfil:   perfil,
		localApp: localApp,
		base:     filepath.Join(perfil, ".gobsidian-sondahost"),
		candidatos: []candidato{
			{"perfil/.gobsidian-sondahost/run (recomendado em G2)", filepath.Join(perfil, ".gobsidian-sondahost", "run")},
			{"LOCALAPPDATA/Temp/gobsidian-sondahost/run (alternativa em G2)", filepath.Join(localApp, "Temp", "gobsidian-sondahost", "run")},
			{"LOCALAPPDATA/gobsidian/run (o diretorio de hoje, controle)", filepath.Join(localApp, "gobsidian", "run")},
		},
		travaComum: filepath.Join(localApp, "gobsidian", "run", nomeTravaComum),
	}, nil
}

func main() {
	amb, err := novoAmbiente()
	if err != nil {
		fmt.Fprintln(os.Stderr, "[!]", err)
		os.Exit(1)
	}
	modo := ""
	if len(os.Args) > 1 {
		modo = os.Args[1]
	}
	switch modo {
	case "":
		os.Exit(sondarComoHost(amb))
	case "preparar":
		os.Exit(preparar(amb))
	case "ler":
		os.Exit(ler(amb))
	case "limpar":
		os.Exit(limpar(amb))
	default:
		fmt.Fprintf(os.Stderr, "[!] modo desconhecido %q: use preparar, ler, limpar, ou nenhum (processo do host)\n", modo)
		os.Exit(2)
	}
}

// errno descreve um erro com o numero do sistema quando ha um, porque 10022 e
// 1920 sao exatamente o que esta sonda procura.
func errno(err error) string {
	if err == nil {
		return "ok"
	}
	var en syscall.Errno
	if errors.As(err, &en) {
		return fmt.Sprintf("errno=%d (%v)", uint(en), err)
	}
	return err.Error()
}

// preparar segura, da shell, o lado "outro processo" de cada medicao cruzada.
func preparar(amb ambiente) int {
	fmt.Printf("[i] contexto deste processo: %s\n", contexto())
	if err := os.MkdirAll(amb.base, 0o700); err != nil {
		fmt.Printf("[!] criando %s: %v\n", amb.base, err)
		return 1
	}

	var ouvintes []net.Listener
	for _, c := range amb.candidatos {
		if err := os.MkdirAll(c.dir, 0o700); err != nil {
			fmt.Printf("[!] %s: criando diretorio: %v\n", c.nome, err)
			continue
		}
		caminho := filepath.Join(c.dir, nomeCruzado)
		_ = os.Remove(caminho)
		ln, err := net.Listen("unix", caminho)
		if err != nil {
			fmt.Printf("[!] %s: listen %s\n", c.nome, errno(err))
			continue
		}
		go aceitarEFechar(ln)
		ouvintes = append(ouvintes, ln)
		fmt.Printf("[OK] %s: escutando em %s\n", c.nome, caminho)
	}

	trava, tomou, err := daemon.TentarTravar(amb.travaComum)
	switch {
	case err != nil:
		fmt.Printf("[!] trava compartilhada %s: %v\n", amb.travaComum, err)
	case !tomou:
		fmt.Printf("[!] trava compartilhada %s ja estava tomada por outro processo\n", amb.travaComum)
	default:
		fmt.Printf("[OK] trava compartilhada tomada: %s\n", amb.travaComum)
	}

	pronto := filepath.Join(amb.base, "pronto.txt")
	if err := os.WriteFile(pronto, []byte(fmt.Sprintf("pid=%d desde=%s\n", os.Getpid(), time.Now().Format(time.RFC3339))), 0o600); err != nil {
		fmt.Printf("[!] gravando %s: %v\n", pronto, err)
	}
	fmt.Println("[i] preparado. Reinicie o host agora; Ctrl+C encerra esta sonda (ou 60 min).")

	ctx, parar := signal.NotifyContext(context.Background(), os.Interrupt)
	defer parar()
	select {
	case <-ctx.Done():
	case <-time.After(60 * time.Minute):
	}

	for _, ln := range ouvintes {
		_ = ln.Close()
	}
	if trava != nil {
		trava.Liberar()
		_ = os.Remove(amb.travaComum)
	}
	_ = os.Remove(pronto)
	fmt.Println("[OK] sonda preparada encerrada")
	return 0
}

func aceitarEFechar(ln net.Listener) {
	for {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		_ = c.Close()
	}
}

// saida grava cada linha em stderr e no arquivo de resultado.
type saida struct{ f *os.File }

func (s saida) linha(format string, a ...any) {
	texto := "[sondahost] " + fmt.Sprintf(format, a...)
	fmt.Fprintln(os.Stderr, texto)
	if s.f != nil {
		_, _ = fmt.Fprintln(s.f, texto)
	}
}

// sondarComoHost e o que roda dentro do processo que o host cria.
func sondarComoHost(amb ambiente) int {
	var out saida
	if err := os.MkdirAll(amb.base, 0o700); err == nil {
		f, err := os.OpenFile(filepath.Join(amb.base, "resultado.txt"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
		if err == nil {
			out.f = f
			defer func() { _ = f.Close() }()
		}
	}

	out.linha("inicio pid=%d ppid=%d em %s", os.Getpid(), os.Getppid(), time.Now().Format(time.RFC3339))
	out.linha("contexto: %s", contexto())
	for _, v := range []string{"LOCALAPPDATA", "USERPROFILE", "TEMP"} {
		out.linha("env %s=%s", v, os.Getenv(v))
	}
	if _, err := os.Stat(filepath.Join(amb.base, "pronto.txt")); err != nil {
		out.linha("AVISO: `sondahost preparar` nao estava rodando -- as medicoes cruzadas e a trava compartilhada nao valem")
	}

	for _, c := range amb.candidatos {
		out.linha("%s | proprio: %s", c.nome, sondarProprio(c.dir))
		cruzado := filepath.Join(c.dir, nomeCruzado)
		_, lerr := os.Lstat(cruzado)
		out.linha("%s | cruzado (socket da shell): lstat %s | dial %s", c.nome, errno(lerr), errno(discar(cruzado)))
	}

	out.linha("trava compartilhada com a shell: %s", sondarTravaComum(amb.travaComum))
	out.linha("trava livre (controle do inverso): %s", sondarTravaLivre(filepath.Dir(amb.travaComum)))

	novo := filepath.Join(amb.localApp, fmt.Sprintf("%s%d", prefixoNovo, os.Getpid()))
	merr := os.MkdirAll(novo, 0o700)
	werr := os.WriteFile(filepath.Join(novo, "marca.txt"), []byte("escrito pelo processo do host\n"), 0o600)
	out.linha("%s%s | mkdir %s | escrita %s", marcaDiretorio, novo, errno(merr), errno(werr))
	out.linha("fim")

	s := mcp.NewServer(&mcp.Implementation{Name: "gobsidian-sondahost", Version: "0.0.1"}, nil)
	_ = s.Run(context.Background(), &mcp.StdioTransport{})
	return 0
}

func discar(caminho string) error {
	c, err := net.DialTimeout("unix", caminho, 3*time.Second)
	if c != nil {
		_ = c.Close()
	}
	return err
}

func sondarProprio(dir string) string {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "mkdir " + errno(err)
	}
	caminho := filepath.Join(dir, fmt.Sprintf("sondahost-%d.sock", os.Getpid()))
	_ = os.Remove(caminho)
	ln, err := net.Listen("unix", caminho)
	if err != nil {
		return "listen " + errno(err)
	}
	go aceitarEFechar(ln)
	_, lerr := os.Lstat(caminho)
	derr := discar(caminho)
	_ = ln.Close()
	_ = os.Remove(caminho)
	return fmt.Sprintf("listen ok | lstat %s | dial %s", errno(lerr), errno(derr))
}

// sondarTravaComum tenta a trava que a shell segura. Tomar seria o defeito: dois
// processos donos da mesma trava, cada um achando que e o unico.
func sondarTravaComum(caminho string) string {
	trava, tomou, err := daemon.TentarTravar(caminho)
	switch {
	case err != nil:
		return "erro " + errno(err)
	case tomou:
		trava.Liberar()
		return "TOMADA TAMBEM AQUI -- exclusao mutua entre contextos NAO funciona (ou a shell nao a segurava)"
	default:
		return "recusada, ja tomada por outro processo -- exclusao mutua entre contextos funciona"
	}
}

// sondarTravaLivre e o inverso: uma trava que ninguem segura tem de ser tomada.
// Sem isto, "recusada" acima nao distingue exclusao mutua de trava quebrada.
func sondarTravaLivre(dir string) string {
	caminho := filepath.Join(dir, fmt.Sprintf("sondahost-livre-%d.lock", os.Getpid()))
	trava, tomou, err := daemon.TentarTravar(caminho)
	defer func() { _ = os.Remove(caminho) }()
	switch {
	case err != nil:
		return "erro " + errno(err)
	case !tomou:
		return "RECUSADA sem dono -- a primitiva nao funciona neste contexto"
	default:
		trava.Liberar()
		return "tomada -- a primitiva funciona neste contexto"
	}
}

// ler imprime o resultado e diz, para cada diretorio novo, se ele caiu no
// caminho real ou foi desviado para o LocalCache de algum pacote.
func ler(amb ambiente) int {
	caminho := filepath.Join(amb.base, "resultado.txt")
	f, err := os.Open(caminho)
	if err != nil {
		fmt.Printf("[!] %v -- o host ainda nao iniciou a sonda?\n", err)
		return 1
	}
	defer func() { _ = f.Close() }()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		l := sc.Text()
		fmt.Println(l)
		i := strings.Index(l, marcaDiretorio)
		if i < 0 {
			continue
		}
		novo := strings.TrimSpace(strings.SplitN(l[i+len(marcaDiretorio):], "|", 2)[0])
		_, errReal := os.Stat(filepath.Join(novo, "marca.txt"))
		desviados, _ := filepath.Glob(filepath.Join(amb.localApp, "Packages", "*", "LocalCache", "Local", filepath.Base(novo), "marca.txt"))
		fmt.Printf("     -> caminho real: %s | copia em LocalCache de pacote: %d %v\n", errno(errReal), len(desviados), desviados)
	}
	if err := sc.Err(); err != nil {
		fmt.Printf("[!] lendo %s: %v\n", caminho, err)
		return 1
	}
	return 0
}

// limpar apaga tudo o que preparar e o processo do host criaram.
func limpar(amb ambiente) int {
	alvos := []string{amb.base, filepath.Join(amb.localApp, "Temp", "gobsidian-sondahost")}
	for _, padrao := range []string{
		filepath.Join(amb.localApp, prefixoNovo+"*"),
		filepath.Join(amb.localApp, "Packages", "*", "LocalCache", "Local", prefixoNovo+"*"),
		filepath.Join(amb.localApp, "gobsidian", "run", "sondahost-*"),
	} {
		achados, _ := filepath.Glob(padrao)
		alvos = append(alvos, achados...)
	}
	falhas := 0
	for _, a := range alvos {
		if _, err := os.Lstat(a); err != nil {
			continue
		}
		if err := os.RemoveAll(a); err != nil {
			falhas++
			fmt.Printf("[!] nao removido: %s: %v\n", a, err)
			continue
		}
		fmt.Printf("[OK] removido: %s\n", a)
	}
	if falhas > 0 {
		return 1
	}
	return 0
}
