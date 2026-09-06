package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSearchCLISegundaExecucaoUsaCache(t *testing.T) {
	cofre := t.TempDir()
	if err := os.WriteFile(filepath.Join(cofre, "n.md"), []byte("# N\n\nexecucao do teste\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cache := t.TempDir()
	roda := func() string {
		var stdout, stderr bytes.Buffer
		cmd := newSearchCmd()
		cmd.SetOut(&stdout)
		cmd.SetErr(&stderr)
		cmd.SetArgs([]string{"--vault", cofre, "--cache-dir", cache, "--log-level", "info", "--json", "execucao"})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("search: %v\nstderr:\n%s", err, stderr.String())
		}
		if !strings.Contains(stdout.String(), `"n.md"`) {
			t.Fatalf("stdout sem o resultado esperado:\n%s", stdout.String())
		}
		return stderr.String()
	}
	primeira := roda()
	if !strings.Contains(primeira, "origem=construcao") {
		t.Fatalf("primeira execucao devia construir o indice de busca:\n%s", primeira)
	}
	segunda := roda()
	if !strings.Contains(segunda, "origem=cache") {
		t.Fatalf("segunda execucao devia adotar o cache de busca:\n%s", segunda)
	}
}
