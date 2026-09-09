package instalar

import (
	"os"
	"path/filepath"
	"testing"
)

// TestPresencaVivoApareceEMortoNao e a razao de existir deste arquivo.
//
// Em 2026-09-08 havia DOIS processos servindo o cofre Estudo e gravando o
// MESMO inverted_cache.gob, e isso so ficou visivel comparando milissegundos
// entre linhas de log duplicadas (10:48:20.137 e 10:48:20.152). Nenhum comando
// do produto respondia "quem esta servindo este cofre agora?".
//
// A resposta NAO passa por enumerar processos do sistema operacional: Go nao
// tem isso de forma portavel, e o projeto proibe `if runtime.GOOS ==` em logica
// compartilhada. Ela reusa a trava de kernel de internal/daemon, que ja resolve
// a parte dificil -- posse decidida pelo kernel, sem PID obsoleto a
// interpretar.
func TestPresencaVivoApareceEMortoNao(t *testing.T) {
	dir := t.TempDir()

	liberar, err := Registrar(dir, `C:\Cofre`, "serve", "v9.9.9")
	if err != nil {
		t.Fatalf("Registrar() error = %v", err)
	}

	vivos, err := Vivos(dir)
	if err != nil {
		t.Fatalf("Vivos() error = %v", err)
	}
	if len(vivos) != 1 {
		t.Fatalf("Vivos() = %d presencas, esperado 1", len(vivos))
	}
	if vivos[0].PID != os.Getpid() {
		t.Errorf("PID = %d, esperado %d", vivos[0].PID, os.Getpid())
	}
	// Conteudo legivel COM a trava tomada: e a razao de internal/daemon travar
	// a faixa em 1<<62, longe dos dados. Se a trava cobrisse o byte 0, isto
	// falharia com "another process has locked a portion of the file".
	if vivos[0].Cofre != `C:\Cofre` || vivos[0].Papel != "serve" || vivos[0].Versao != "v9.9.9" {
		t.Errorf("conteudo ilegivel ou incompleto sob a trava: %+v", vivos[0])
	}

	// Soltar a trava e o que o KERNEL faz sozinho quando o processo morre.
	liberar()

	vivos, err = Vivos(dir)
	if err != nil {
		t.Fatalf("Vivos() error = %v", err)
	}
	if len(vivos) != 0 {
		t.Fatalf("Vivos() = %d presencas depois de liberar, esperado 0: %+v", len(vivos), vivos)
	}
}

// TestPresencaArquivoOrfaoNaoContaComoVivo: o arquivo SOBREVIVE ao processo, de
// proposito -- o mesmo desenho de internal/daemon/trava.go, que nunca remove o
// lock porque remover era a origem de toda corrida. O que decide e a trava,
// nunca a existencia.
//
// Sem este caso, um Vivos() que contasse arquivos passaria no teste acima e
// reportaria como vivos os 960 arquivos orfaos que a maquina do dono tinha em
// 2026-09-08.
func TestPresencaArquivoOrfaoNaoContaComoVivo(t *testing.T) {
	dir := t.TempDir()
	orfao := filepath.Join(dir, "serve.4242"+SufixoDePresenca)
	conteudo := []byte(`{"pid":4242,"cofre":"C:\\Sumido","papel":"serve","versao":"v0.0.1"}`)
	if err := os.WriteFile(orfao, conteudo, 0o600); err != nil {
		t.Fatal(err)
	}

	vivos, err := Vivos(dir)
	if err != nil {
		t.Fatalf("Vivos() error = %v", err)
	}
	if len(vivos) != 0 {
		t.Fatalf("um arquivo de presenca SEM trava contou como processo vivo: %+v", vivos)
	}

	orfaos, err := PresencasOrfas(dir)
	if err != nil {
		t.Fatalf("PresencasOrfas() error = %v", err)
	}
	if len(orfaos) != 1 || orfaos[0] != orfao {
		t.Fatalf("PresencasOrfas() = %v, esperado [%s]", orfaos, orfao)
	}
}

// TestPresencaDiretorioAusenteNaoEhErro: Vivos e chamado por doctor e pelo
// instalador antes de o diretorio de runtime existir. "Ninguem rodando" e a
// resposta certa, nao uma falha.
func TestPresencaDiretorioAusenteNaoEhErro(t *testing.T) {
	vivos, err := Vivos(filepath.Join(t.TempDir(), "nao-existe"))
	if err != nil {
		t.Fatalf("Vivos() em diretorio ausente virou erro: %v", err)
	}
	if len(vivos) != 0 {
		t.Fatalf("Vivos() = %+v, esperado vazio", vivos)
	}
}

// TestPresencaIgnoraOutrosArquivos: o diretorio de runtime tambem guarda
// .lock, .sock e .log. Contar qualquer um deles como presenca faria o
// instalador anunciar processos que nao existem e pedir para mata-los.
func TestPresencaIgnoraOutrosArquivos(t *testing.T) {
	dir := t.TempDir()
	for _, nome := range []string{"a.sock", "a.sock.lock", "a.sock.log", "instalacao.lock"} {
		if err := os.WriteFile(filepath.Join(dir, nome), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	liberar, err := Registrar(dir, `C:\Cofre`, "daemon", "v1")
	if err != nil {
		t.Fatalf("Registrar() error = %v", err)
	}
	defer liberar()

	vivos, err := Vivos(dir)
	if err != nil {
		t.Fatalf("Vivos() error = %v", err)
	}
	if len(vivos) != 1 {
		t.Fatalf("Vivos() = %d, esperado 1 -- arquivos que nao sao presenca foram contados: %+v", len(vivos), vivos)
	}
	if vivos[0].Papel != "daemon" {
		t.Errorf("Papel = %q, esperado daemon", vivos[0].Papel)
	}
}

// TestRegistrarAteMorrerMantemAPresencaViva cobre o caminho que serve e daemon
// usam de verdade.
//
// Ele nao pode usar Registrar + defer: runServe termina em os.Exit, e defer nao
// roda depois disso -- o golangci-lint acusou exatamente esse defeito quando a
// primeira versao tentou. E soltar a referencia tambem nao serve: a trava e um
// *os.File por baixo, e o finalizador do Go FECHA o descritor quando o valor
// deixa de ser alcancavel, o que soltaria a trava com o processo vivo. Por isso
// a trava fica numa variavel de pacote.
func TestRegistrarAteMorrerMantemAPresencaViva(t *testing.T) {
	dir := t.TempDir()
	t.Cleanup(LiberarPresenca)

	if err := RegistrarAteMorrer(dir, `C:\Cofre`, "serve", "v1.2.3"); err != nil {
		t.Fatalf("RegistrarAteMorrer() error = %v", err)
	}

	vivos, err := Vivos(dir)
	if err != nil {
		t.Fatalf("Vivos() error = %v", err)
	}
	if len(vivos) != 1 || vivos[0].Versao != "v1.2.3" {
		t.Fatalf("Vivos() = %+v, esperado uma presenca v1.2.3", vivos)
	}

	LiberarPresenca()

	vivos, err = Vivos(dir)
	if err != nil {
		t.Fatalf("Vivos() error = %v", err)
	}
	if len(vivos) != 0 {
		t.Fatalf("Vivos() = %+v depois de LiberarPresenca(), esperado vazio", vivos)
	}

	// Idempotente, pela mesma razao que o desarmar do guarda-chuva.
	LiberarPresenca()
}
