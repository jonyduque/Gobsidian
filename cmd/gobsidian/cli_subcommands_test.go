package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

func createTestVault(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	files := map[string]string{
		"nota1.md": "# Titulo 1\nTexto da nota 1 #tag1\nLink para [[nota2#Secao]]",
		"nota2.md": "# Titulo 2\n## Secao\nTexto da nota 2 #tag2",
	}

	for rel, content := range files {
		full := filepath.Join(dir, rel)
		if err := os.WriteFile(full, []byte(content), 0644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
	}

	return dir
}

func TestIndexCmd_StdoutAndJSON(t *testing.T) {
	vaultDir := createTestVault(t)

	// Mode 1: Non-JSON
	cmd := newIndexCmd()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--vault", vaultDir, "--cache-dir", t.TempDir()})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("index Execute: %v", err)
	}

	if stderr.Len() > 0 {
		t.Errorf("stderr NAO deveria conter nada, continha: %q", stderr.String())
	}
	outStr := stdout.String()
	if outStr == "" {
		t.Error("stdout deveria conter o resumo da indexacao")
	}

	// Mode 2: --json
	cmdJSON := newIndexCmd()
	var stdoutJSON, stderrJSON bytes.Buffer
	cmdJSON.SetOut(&stdoutJSON)
	cmdJSON.SetErr(&stderrJSON)
	cmdJSON.SetArgs([]string{"--vault", vaultDir, "--cache-dir", t.TempDir(), "--json"})

	if err := cmdJSON.Execute(); err != nil {
		t.Fatalf("index --json Execute: %v", err)
	}

	if stderrJSON.Len() > 0 {
		t.Errorf("stderr NAO deveria conter nada, continha: %q", stderrJSON.String())
	}

	var data indexSummaryJSON
	if err := json.Unmarshal(stdoutJSON.Bytes(), &data); err != nil {
		t.Fatalf("saida de --json NAO e JSON valido: %v; stdout=%q", err, stdoutJSON.String())
	}

	if data.Notes != 2 {
		t.Errorf("data.Notes = %d; quer 2", data.Notes)
	}
}

func TestSearchCmd_StdoutAndJSON(t *testing.T) {
	vaultDir := createTestVault(t)

	// Mode 1: Non-JSON
	cmd := newSearchCmd()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--vault", vaultDir, "--cache-dir", t.TempDir(), "nota"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("search Execute: %v", err)
	}

	if stderr.Len() > 0 {
		t.Errorf("stderr NAO deveria conter nada, continha: %q", stderr.String())
	}
	if stdout.Len() == 0 {
		t.Error("stdout deveria conter os resultados da busca")
	}

	// Mode 2: --json
	cmdJSON := newSearchCmd()
	var stdoutJSON, stderrJSON bytes.Buffer
	cmdJSON.SetOut(&stdoutJSON)
	cmdJSON.SetErr(&stderrJSON)
	cmdJSON.SetArgs([]string{"--vault", vaultDir, "--cache-dir", t.TempDir(), "nota", "--json"})

	if err := cmdJSON.Execute(); err != nil {
		t.Fatalf("search --json Execute: %v", err)
	}

	if stderrJSON.Len() > 0 {
		t.Errorf("stderr NAO deveria conter nada, continha: %q", stderrJSON.String())
	}

	var data map[string]any
	if err := json.Unmarshal(stdoutJSON.Bytes(), &data); err != nil {
		t.Fatalf("saida de --json NAO e JSON valido: %v; stdout=%q", err, stdoutJSON.String())
	}
}

func TestInspectCmd_StdoutAndJSON(t *testing.T) {
	vaultDir := createTestVault(t)

	// Mode 1: Non-JSON
	cmd := newInspectCmd()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--vault", vaultDir, "--cache-dir", t.TempDir(), "nota2.md"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("inspect Execute: %v", err)
	}

	if stderr.Len() > 0 {
		t.Errorf("stderr NAO deveria conter nada, continha: %q", stderr.String())
	}
	if stdout.Len() == 0 {
		t.Error("stdout deveria conter a inspecao da nota")
	}

	// Mode 2: --json
	cmdJSON := newInspectCmd()
	var stdoutJSON, stderrJSON bytes.Buffer
	cmdJSON.SetOut(&stdoutJSON)
	cmdJSON.SetErr(&stderrJSON)
	cmdJSON.SetArgs([]string{"--vault", vaultDir, "--cache-dir", t.TempDir(), "nota2.md", "--json"})

	if err := cmdJSON.Execute(); err != nil {
		t.Fatalf("inspect --json Execute: %v", err)
	}

	if stderrJSON.Len() > 0 {
		t.Errorf("stderr NAO deveria conter nada, continha: %q", stderrJSON.String())
	}

	var data inspectResultJSON
	if err := json.Unmarshal(stdoutJSON.Bytes(), &data); err != nil {
		t.Fatalf("saida de --json NAO e JSON valido: %v; stdout=%q", err, stdoutJSON.String())
	}

	if data.Path != "nota2.md" {
		t.Errorf("data.Path = %q; quer nota2.md", data.Path)
	}
	if len(data.Backlinks) != 1 || data.Backlinks[0] != "nota1.md" {
		t.Errorf("data.Backlinks = %v; quer [nota1.md]", data.Backlinks)
	}
}

// TestSearchCLIRespeitaMaxResults trava a regressao do achado 1.11:
// cfg.MaxResults era lido e validado por config.Load mas descartado antes de
// chegar em service.Options — a flag --max-results de "search" prometia um
// teto que o servico nunca aplicava.
func TestSearchCLIRespeitaMaxResults(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < 5; i++ {
		p := filepath.Join(root, fmt.Sprintf("n%d.md", i))
		if err := os.WriteFile(p, []byte("# Nota\n\npalavra unica aqui\n"), 0644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
	}
	var out bytes.Buffer
	cmd := newSearchCmd()
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--vault", root, "--cache-dir", t.TempDir(), "--json", "--max-results", "2", "palavra"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	var res struct {
		Results []json.RawMessage `json:"results"`
	}
	if err := json.Unmarshal(out.Bytes(), &res); err != nil {
		t.Fatalf("saida nao e JSON: %v\n%s", err, out.String())
	}
	if len(res.Results) != 2 {
		t.Fatalf("results = %d com --max-results 2: a flag e lida e descartada", len(res.Results))
	}
}

// TestIndexEInspectNaoAceitamFlagsQueIgnoram trava a regressao do achado
// 5.8: index e inspect declaravam --read-only, --debounce-ms e --max-results
// sem observar nenhuma delas — nem escrevem, nem observam, nem buscam.
// "Schema que promete e codigo que ignora e pior que parametro ausente."
func TestIndexEInspectNaoAceitamFlagsQueIgnoram(t *testing.T) {
	for _, tc := range []struct {
		nome string
		cmd  func() *cobra.Command
	}{{"index", newIndexCmd}, {"inspect", newInspectCmd}} {
		for _, flag := range []string{"read-only", "debounce-ms", "max-results"} {
			if tc.cmd().Flags().Lookup(flag) != nil {
				t.Errorf("%s declara --%s e nao a usa", tc.nome, flag)
			}
		}
	}
	for _, flag := range []string{"read-only", "debounce-ms"} {
		if newSearchCmd().Flags().Lookup(flag) != nil {
			t.Errorf("search declara --%s e nao a usa", flag)
		}
	}
}
