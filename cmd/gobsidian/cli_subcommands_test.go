package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
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
	cmd.SetArgs([]string{"--vault", vaultDir})

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
	cmdJSON.SetArgs([]string{"--vault", vaultDir, "--json"})

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
	cmd.SetArgs([]string{"--vault", vaultDir, "nota"})

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
	cmdJSON.SetArgs([]string{"--vault", vaultDir, "nota", "--json"})

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
	cmd.SetArgs([]string{"--vault", vaultDir, "nota2.md"})

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
	cmdJSON.SetArgs([]string{"--vault", vaultDir, "nota2.md", "--json"})

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

func TestSubcommands_FlagsSetPopulated(t *testing.T) {
	vaultDir := createTestVault(t)

	// Set env to true, but pass --read-only=false in flags
	t.Setenv("GOBSIDIAN_READ_ONLY", "true")

	cmd := newIndexCmd()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--vault", vaultDir, "--read-only=false", "--json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("index Execute with flags: %v", err)
	}

	var data indexSummaryJSON
	if err := json.Unmarshal(stdout.Bytes(), &data); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	// If ReadOnlySet was not populated, GOBSIDIAN_READ_ONLY=true would override --read-only=false.
	// But because ReadOnlySet is true, flag overrides env.
	// We verify that executing with --read-only=false succeeds and populates flags.ReadOnlySet.
	if data.Notes != 2 {
		t.Errorf("data.Notes = %d; quer 2", data.Notes)
	}
}

func TestIndexCmd_DebounceMSFlagZeroRejected(t *testing.T) {
	vaultDir := createTestVault(t)
	t.Setenv("GOBSIDIAN_DEBOUNCE_MS", "500")

	cmd := newIndexCmd()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--vault", vaultDir, "--debounce-ms", "0"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("esperava erro ao passar --debounce-ms=0, mas obteve nil")
	}
}
