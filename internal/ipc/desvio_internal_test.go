package ipc

import (
	"os"
	"runtime"
	"strings"
	"testing"
)

// TestRuntimeDirDoSistemaFicaNoPerfilNoWindows verifica o que da para provar
// sem privilegio administrativo neste ambiente: o diretorio de runtime de
// producao fica dentro do perfil do usuario corrente (%LocalAppData%), cuja
// ACL padrao do Windows ja nega acesso a outros usuarios locais. Isto NAO E o
// mesmo que abrir uma segunda conta de usuario e tentar conectar -- essa prova
// exigiria criar uma conta local, o que requer privilegio administrativo que
// este ambiente de teste nao concede. Ver o relatorio da Task 91 para o
// registro explicito dessa lacuna.
//
// Ate 2026-09-14 este teste abria um socket de verdade via Listen e conferia o
// prefixo do caminho. Isso escrevia uma trava no %LocalAppData% real a cada
// rodada. A garantia e sobre ONDE a conta de producao aponta, e a conta de
// producao e runtimeDirDoSistema: conferir o valor dela prova o mesmo sem
// criar arquivo nenhum.
func TestRuntimeDirDoSistemaFicaNoPerfilNoWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("especifico de Windows -- ver TestListenRestringePermissaoUnix para a garantia equivalente em Unix")
	}

	dir, err := runtimeDirDoSistema()
	if err != nil {
		t.Fatalf("runtimeDirDoSistema() error = %v", err)
	}
	base, err := os.UserCacheDir()
	if err != nil {
		t.Fatalf("UserCacheDir() error = %v", err)
	}
	if !strings.HasPrefix(dir, base) {
		t.Fatalf("diretorio de runtime %s nao esta dentro do perfil do usuario %s", dir, base)
	}
}

// TestSocketDeTesteCaiNoDesvio prova que a suite deste pacote roda isolada: um
// socket aberto por um teste qualquer mora no desvio de RodarComRuntimeIsolado,
// nunca no diretorio do sistema.
//
// Sem este teste, apagar o TestMain deste diretorio deixaria a suite verde e
// escrevendo no perfil do usuario -- que e exatamente o estado anterior a
// 2026-09-14.
func TestSocketDeTesteCaiNoDesvio(t *testing.T) {
	if desvioDoRuntime == "" {
		t.Fatal("a suite de ipc roda sem desvio do diretorio de runtime: testmain_test.go nao chama RodarComRuntimeIsolado")
	}

	ln, path, err := Listen(t.TempDir())
	if err != nil {
		t.Fatalf("Listen() error = %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	if !strings.HasPrefix(path, desvioDoRuntime) {
		t.Fatalf("socket %s fora do desvio %s", path, desvioDoRuntime)
	}
	sistema, err := runtimeDirDoSistema()
	if err != nil {
		t.Fatalf("runtimeDirDoSistema() error = %v", err)
	}
	if strings.HasPrefix(path, sistema) {
		t.Fatalf("socket %s caiu no diretorio do sistema %s", path, sistema)
	}
}
