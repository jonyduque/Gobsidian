package instalar

import (
	"os"
	"path/filepath"
	"testing"
)

// TestTravaGlobalVisivelParaQuemPergunta e o contrato entre o instalador e os
// servidores: enquanto a instalacao roda, ninguem sobe.
//
// Sem isso, o host MCP respawna o servidor no MEIO da troca do binario e duas
// versoes voltam a conviver -- o defeito que encerrar todos os processos existe
// para fechar. Medido em 2026-09-07: o host reconectou 15 s depois da
// desconexao, sozinho.
func TestTravaGlobalVisivelParaQuemPergunta(t *testing.T) {
	dir := t.TempDir()

	if InstalacaoEmCurso(dir) {
		t.Fatal("InstalacaoEmCurso() = true sem instalacao nenhuma")
	}

	liberar, err := TomarTravaGlobal(dir)
	if err != nil {
		t.Fatalf("TomarTravaGlobal() error = %v", err)
	}

	if !InstalacaoEmCurso(dir) {
		t.Error("InstalacaoEmCurso() = false com a trava tomada; um serve subiria no meio da troca")
	}

	liberar()

	if InstalacaoEmCurso(dir) {
		t.Error("InstalacaoEmCurso() = true depois de liberar; todo serve seguinte ficaria de fora")
	}
}

// TestTravaGlobalNaoRemoveOArquivo: o arquivo persiste entre execucoes, de
// proposito. Remover era a origem de toda corrida do esquema antigo -- entre o
// remover e o recriar, qualquer um entrava (internal/daemon/trava.go).
func TestTravaGlobalNaoRemoveOArquivo(t *testing.T) {
	dir := t.TempDir()
	liberar, err := TomarTravaGlobal(dir)
	if err != nil {
		t.Fatalf("TomarTravaGlobal() error = %v", err)
	}
	liberar()

	caminho := filepath.Join(dir, NomeDaTravaGlobal)
	if _, err := os.Stat(caminho); err != nil {
		t.Fatalf("o arquivo da trava sumiu depois de liberar: %v", err)
	}

	// E tomar de novo continua funcionando: quem decide a posse e a trava, nao
	// a existencia.
	liberar2, err := TomarTravaGlobal(dir)
	if err != nil {
		t.Fatalf("segunda TomarTravaGlobal() error = %v", err)
	}
	liberar2()
}

// TestTravaGlobalRecusaSegundaInstalacao: duas instalacoes simultaneas trocando
// o mesmo binario e o unico jeito de acabar com um executavel meio escrito.
func TestTravaGlobalRecusaSegundaInstalacao(t *testing.T) {
	dir := t.TempDir()

	liberar, err := TomarTravaGlobal(dir)
	if err != nil {
		t.Fatalf("TomarTravaGlobal() error = %v", err)
	}
	defer liberar()

	if _, err := TomarTravaGlobal(dir); err == nil {
		t.Fatal("a segunda TomarTravaGlobal() foi aceita; duas instalacoes trocariam o binario juntas")
	}
}
