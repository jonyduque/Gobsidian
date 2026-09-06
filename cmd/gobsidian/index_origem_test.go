package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestIndexCmdJSONTrazOrigem(t *testing.T) {
	cofre := t.TempDir()
	if err := os.WriteFile(filepath.Join(cofre, "n.md"), []byte("# N\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cache := t.TempDir()
	roda := func() string {
		var stdout, stderr bytes.Buffer
		cmd := newIndexCmd()
		cmd.SetOut(&stdout)
		cmd.SetErr(&stderr)
		cmd.SetArgs([]string{"--vault", cofre, "--cache-dir", cache, "--json"})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("index: %v\nstderr:\n%s", err, stderr.String())
		}
		var out struct {
			Origin string `json:"origin"`
			Notes  int    `json:"notes"`
		}
		if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
			t.Fatalf("JSON invalido: %v\n%s", err, stdout.String())
		}
		if out.Notes != 1 {
			t.Fatalf("notes = %d, quero 1", out.Notes)
		}
		return out.Origin
	}
	if o := roda(); o != "build" {
		t.Fatalf("primeira execucao: origin = %q, quero \"build\"", o)
	}
	if o := roda(); o != "cache" {
		t.Fatalf("segunda execucao: origin = %q, quero \"cache\"", o)
	}
}
