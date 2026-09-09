package instalar

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jonyd/gobsidian/internal/hosts"
)

// mundoDeTeste desvia TUDO que a instalacao tocaria para diretorios
// temporarios: runtime, cache, manifesto, destino do binario e hosts.
//
// A instalacao real encerra processos, escreve no registro do Windows e edita
// arquivos de configuracao que o usuario escreveu a mao. Um teste que fizesse
// qualquer uma dessas coisas na maquina de quem roda a suite seria pior que
// nenhum teste -- e e por isso que Sistema e Opcoes carregam pontos de
// injecao em vez de chamarem o mundo direto.
type mundoDeTeste struct {
	runtimeDir string
	cacheRaiz  string
	destino    string
	origem     string
	amb        hosts.Ambiente

	perguntas  []string
	respostaOK bool
	mortos     []int
	pathPedido string
}

func novoMundo(t *testing.T) *mundoDeTeste {
	t.Helper()
	raiz := t.TempDir()

	// O manifesto mora sob RaizDoCache(). Desviar por VARIAVEL DE AMBIENTE nao
	// funciona nas tres plataformas: no macOS, os.UserCacheDir devolve
	// $HOME/Library/Caches e ignora XDG_CACHE_HOME -- e o CI de 2026-09-09
	// mostrou o resultado, com os testes gravando no cache real do usuario e um
	// enxergando o manifesto do outro. A raiz e injetada direto.
	cacheDeTeste := filepath.Join(raiz, "cache")
	original := raizDoCache
	raizDoCache = func() string { return cacheDeTeste }
	t.Cleanup(func() { raizDoCache = original })

	m := &mundoDeTeste{
		runtimeDir: filepath.Join(raiz, "run"),
		cacheRaiz:  filepath.Join(raiz, "cacheraiz"),
		destino:    filepath.Join(raiz, "programas", "gobsidian"),
		origem:     filepath.Join(raiz, "baixado", NomeDoExecutavel),
		respostaOK: true,
	}
	if err := os.MkdirAll(filepath.Dir(m.origem), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(m.origem, []byte("binario novo, de mentira"), 0o700); err != nil {
		t.Fatal(err)
	}
	// Ambiente sem host nenhum instalado: a configuracao de hosts tem teste
	// proprio em internal/hosts, e aqui o que se prova e a SEQUENCIA.
	m.amb = hosts.Ambiente{
		Home:       filepath.Join(raiz, "home"),
		Existe:     func(string) bool { return false },
		TemComando: func(string) bool { return false },
		Rodar:      func(string, ...string) error { return nil },
	}
	return m
}

func (m *mundoDeTeste) sistema() Sistema {
	return Sistema{
		Encerrar: func(pid int) error {
			m.mortos = append(m.mortos, pid)
			return nil
		},
		Confirmar: func(pergunta string, itens []string) bool {
			m.perguntas = append(m.perguntas, pergunta+" || "+strings.Join(itens, " | "))
			return m.respostaOK
		},
		AjustarPath: func(dir string) (bool, error) {
			m.pathPedido = dir
			return true, nil
		},
		Agora: func() time.Time { return time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC) },
	}
}

func (m *mundoDeTeste) opcoes() Opcoes {
	return Opcoes{
		Origem:     m.origem,
		Destino:    m.destino,
		Versao:     "v9.9.9",
		Cofre:      `C:\Cofre`,
		Ambiente:   &m.amb,
		RuntimeDir: m.runtimeDir,
		CacheRaiz:  m.cacheRaiz,
	}
}

// TestInstalarSequenciaCompleta prova que os sete passos aconteceram, e na
// ordem que importa.
func TestInstalarSequenciaCompleta(t *testing.T) {
	m := novoMundo(t)

	r, err := Instalar(context.Background(), m.sistema(), m.opcoes())
	if err != nil {
		t.Fatalf("Instalar() error = %v", err)
	}

	binario := filepath.Join(m.destino, NomeDoExecutavel)
	if r.Binario != binario {
		t.Errorf("Binario = %q, esperado %q", r.Binario, binario)
	}
	if _, err := os.Stat(binario); err != nil {
		t.Fatalf("o binario nao foi instalado: %v", err)
	}
	if r.VersaoHash == "" {
		t.Error("o hash do binario instalado nao foi calculado")
	}
	if m.pathPedido != m.destino {
		t.Errorf("PATH ajustado para %q, esperado %q", m.pathPedido, m.destino)
	}

	mani, err := LerManifesto()
	if err != nil {
		t.Fatalf("LerManifesto() error = %v", err)
	}
	if mani.Binario != binario || mani.Versao != "v9.9.9" || mani.Hash != r.VersaoHash {
		t.Errorf("manifesto incompleto: %+v", mani)
	}
	if mani.PathAdicionado != m.destino {
		t.Errorf("o manifesto nao registrou o PATH: %+v", mani)
	}
	if !mani.Em.Equal(time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)) {
		t.Errorf("o manifesto nao usou o relogio injetado: %v", mani.Em)
	}
}

// TestInstalarLiberaATravaGlobalNoFim: uma trava esquecida tomada impede TODO
// `serve` seguinte de subir -- o defeito de 2026-08-13, que desligou o daemon
// por tres dias.
func TestInstalarLiberaATravaGlobalNoFim(t *testing.T) {
	m := novoMundo(t)
	if _, err := Instalar(context.Background(), m.sistema(), m.opcoes()); err != nil {
		t.Fatalf("Instalar() error = %v", err)
	}
	if InstalacaoEmCurso(m.runtimeDir) {
		t.Fatal("a trava global continuou tomada depois da instalacao; nenhum serve subiria mais")
	}
}

// TestInstalarPerguntaAntesDeEncerrarERecusaAborta e a decisao D-06 do dono.
//
// Encerrar sessao MCP sem avisar derruba trabalho em curso, e o usuario nao tem
// como saber que foi o instalador -- e exatamente o que aconteceu em
// 2026-09-07, quando a desconexao que abriu esta investigacao veio do proprio
// instalador.
func TestInstalarPerguntaAntesDeEncerrarERecusaAborta(t *testing.T) {
	m := novoMundo(t)

	// Um processo "vivo" de verdade: presenca com a trava tomada.
	liberar, err := Registrar(m.runtimeDir, `C:\Outro`, "serve", "v1")
	if err != nil {
		t.Fatalf("Registrar() error = %v", err)
	}
	defer liberar()

	m.respostaOK = false
	_, err = Instalar(context.Background(), m.sistema(), m.opcoes())
	if !errors.Is(err, ErrRecusado) {
		t.Fatalf("Instalar() com recusa devolveu %v, esperado ErrRecusado", err)
	}

	if len(m.perguntas) != 1 {
		t.Fatalf("esperado exatamente uma pergunta, veio %v", m.perguntas)
	}
	// A lista precisa trazer PID e cofre: "3 processos" nao permite discordar
	// de nenhum deles.
	if !strings.Contains(m.perguntas[0], "pid ") || !strings.Contains(m.perguntas[0], `C:\Outro`) {
		t.Errorf("a pergunta nao mostra PID e cofre: %q", m.perguntas[0])
	}
	if len(m.mortos) != 0 {
		t.Errorf("matou processo depois de o usuario recusar: %v", m.mortos)
	}
	if _, err := os.Stat(filepath.Join(m.destino, NomeDoExecutavel)); !os.IsNotExist(err) {
		t.Error("instalou o binario mesmo com a recusa")
	}
	if InstalacaoEmCurso(m.runtimeDir) {
		t.Error("a trava global ficou tomada depois do aborto")
	}
}

// TestInstalarEncerraQuandoAutorizado e o outro lado: sem ele, um instalador
// que NUNCA encerrasse passaria no teste acima.
func TestInstalarEncerraQuandoAutorizado(t *testing.T) {
	m := novoMundo(t)
	liberar, err := Registrar(m.runtimeDir, `C:\Outro`, "daemon", "v1")
	if err != nil {
		t.Fatalf("Registrar() error = %v", err)
	}
	defer liberar()

	r, err := Instalar(context.Background(), m.sistema(), m.opcoes())
	if err != nil {
		t.Fatalf("Instalar() error = %v", err)
	}
	if len(m.mortos) != 1 || m.mortos[0] != os.Getpid() {
		t.Fatalf("encerrou %v, esperado o PID registrado (%d)", m.mortos, os.Getpid())
	}
	if len(r.Encerrados) != 1 || r.Encerrados[0].Papel != "daemon" {
		t.Errorf("o resultado nao registrou quem foi encerrado: %+v", r.Encerrados)
	}
}

// TestInstalarSemPathNaoTocaNoPath: `--no-path` e uma promessa, e uma promessa
// sem teste e uma flag decorativa.
func TestInstalarSemPathNaoTocaNoPath(t *testing.T) {
	m := novoMundo(t)
	o := m.opcoes()
	o.SemPath = true

	r, err := Instalar(context.Background(), m.sistema(), o)
	if err != nil {
		t.Fatalf("Instalar() error = %v", err)
	}
	if m.pathPedido != "" {
		t.Errorf("--no-path mexeu no PATH: %q", m.pathPedido)
	}
	if r.PathMudou {
		t.Error("--no-path relatou mudanca de PATH")
	}
	mani, err := LerManifesto()
	if err != nil {
		t.Fatal(err)
	}
	if mani.PathAdicionado != "" {
		t.Errorf("o manifesto registrou PATH que nao foi adicionado: %+v", mani)
	}
}

// TestInstalarTrocaBinarioAntigoPorRename cobre a medicao de 2026-09-08:
// sobrescrever um executavel em uso falha, renomear funciona. Como
// `gobsidian update` E o executavel a substituir, este e o unico caminho.
func TestInstalarTrocaBinarioAntigoPorRename(t *testing.T) {
	m := novoMundo(t)
	binario := filepath.Join(m.destino, NomeDoExecutavel)
	if err := os.MkdirAll(m.destino, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(binario, []byte("binario ANTIGO"), 0o700); err != nil {
		t.Fatal(err)
	}

	if _, err := Instalar(context.Background(), m.sistema(), m.opcoes()); err != nil {
		t.Fatalf("Instalar() error = %v", err)
	}

	novo, err := os.ReadFile(binario)
	if err != nil {
		t.Fatal(err)
	}
	if string(novo) != "binario novo, de mentira" {
		t.Fatalf("o binario nao foi substituido: %q", novo)
	}
}

// TestEstaInstaladoResponsePelaIdentidadeDoArquivo e a deteccao da decisao
// D-11: `gobsidian` sem argumentos so se autoinstala se NAO estiver instalado.
//
// A comparacao e os.SameFile, e nao texto: link, junction e diferenca de grafia
// apontam para o mesmo arquivo, e um instalador que se reinstalasse por causa
// de uma barra invertida seria pior que nenhum.
func TestEstaInstaladoResponsePelaIdentidadeDoArquivo(t *testing.T) {
	m := novoMundo(t)

	// Antes de instalar: nao ha manifesto.
	instalado, _, err := EstaInstalado()
	if !errors.Is(err, ErrSemManifesto) {
		t.Fatalf("EstaInstalado() sem manifesto devolveu (%v, %v), esperado ErrSemManifesto", instalado, err)
	}

	if _, err := Instalar(context.Background(), m.sistema(), m.opcoes()); err != nil {
		t.Fatalf("Instalar() error = %v", err)
	}

	// Depois: o manifesto existe, mas o executavel CORRENTE e o binario de
	// teste, nao o instalado. A resposta certa e "nao instalado" -- e e o que
	// faz um gobsidian.exe baixado no Downloads se reconhecer como avulso.
	instalado, mani, err := EstaInstalado()
	if err != nil {
		t.Fatalf("EstaInstalado() error = %v", err)
	}
	if instalado {
		t.Error("o binario de teste se declarou instalado")
	}
	if mani.Binario == "" {
		t.Error("EstaInstalado() nao devolveu o manifesto que leu")
	}
}

// TestConfigurarHostsDistingueNenhumDeDetectar: `--hosts none` tem de
// configurar ZERO hosts. Com uma condicao `len(o.Hosts) > 0`, a fatia vazia
// cairia na deteccao e `none` configuraria tudo -- o oposto do que ele diz.
func TestConfigurarHostsDistingueNenhumDeDetectar(t *testing.T) {
	var detectou bool
	amb := hosts.Ambiente{
		Home: t.TempDir(),
		Existe: func(string) bool {
			detectou = true
			return false
		},
		TemComando: func(string) bool {
			detectou = true
			return false
		},
		Rodar: func(string, ...string) error { return nil },
	}

	ok, falhos := configurarHosts(Opcoes{Hosts: []string{}, Ambiente: &amb}, "bin")
	if len(ok) != 0 || len(falhos) != 0 {
		t.Errorf("--hosts none configurou algo: ok=%v falhos=%v", ok, falhos)
	}
	if detectou {
		t.Error("--hosts none disparou a deteccao de hosts")
	}
}
