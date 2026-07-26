package doctor_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jonyd/gobsidian/internal/config"
	"github.com/jonyd/gobsidian/internal/doctor"
)

func TestRunFlagsMissingVault(t *testing.T) {
	cfg := config.Defaults()
	cfg.VaultPath = filepath.Join(t.TempDir(), "nao-existe")

	results := doctor.Run(context.Background(), cfg)

	if doctor.ExitCode(results) == 0 {
		t.Fatal("doctor deveria falhar com raiz inexistente")
	}
	if !hasFailure(results, "raiz do cofre") {
		t.Errorf("nenhuma verificacao de raiz falhou: %+v", results)
	}

	// A verificacao de raiz e "halting": sua falha deve impedir toda e
	// qualquer verificacao seguinte de rodar, nao apenas marca-las como
	// derivadas. Sem raiz nenhuma outra verificacao tem informacao nova a
	// acrescentar, e o mecanismo de halt existe exatamente para nao produzir
	// ruido nesse caso. Testado aqui diretamente (contagem de resultados) em
	// vez de com uma raiz existente-mas-ilegivel: nao foi possivel tornar um
	// diretorio ilegivel neste ambiente (Windows, conta com privilegios de
	// administrador/dono) nem com chmod nem com icacls /deny — a ACL de
	// permissao explicita e sobrescrita pelas ACEs herdadas de Full Control do
	// dono, entao os.ReadDir e os.Stat continuam enxergando o diretorio.
	if len(results) != 1 {
		t.Errorf("raiz inexistente deveria parar a execucao apos 1 resultado, obteve %d: %+v", len(results), results)
	}
}

func TestRunWarnsWithoutObsidianDir(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "A.md"), []byte("# A\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cfg := config.Defaults()
	cfg.VaultPath = root

	results := doctor.Run(context.Background(), cfg)

	// Ausencia de .obsidian/ e aviso, nao falha: o produto funciona sobre
	// qualquer pasta de Markdown, e forcar a presenca seria arbitrario.
	if doctor.ExitCode(results) != 0 {
		t.Errorf("doctor falhou em cofre valido sem .obsidian: %+v", results)
	}
	if !hasStatus(results, ".obsidian", doctor.StatusWarn) {
		t.Errorf("esperava aviso sobre .obsidian ausente: %+v", results)
	}
}

func hasFailure(results []doctor.Result, substr string) bool {
	return hasStatus(results, substr, doctor.StatusFail)
}

func hasStatus(results []doctor.Result, substr string, want doctor.Status) bool {
	for _, r := range results {
		if r.Status == want && contains(r.Name, substr) {
			return true
		}
	}
	return false
}

func contains(haystack, needle string) bool {
	return len(needle) == 0 || len(haystack) >= len(needle) &&
		(haystack == needle || indexOf(haystack, needle) >= 0)
}

func indexOf(h, n string) int {
	for i := 0; i+len(n) <= len(h); i++ {
		if h[i:i+len(n)] == n {
			return i
		}
	}
	return -1
}
