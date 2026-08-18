//go:build windows

package vault_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jonyd/gobsidian/internal/vault"
)

// TestWalkRecordsSkipOnCanonicalizeRejection is the permanent regression test
// for Finding 2 (fix pass 2): the RecordSkip-on-Canonicalize-rejection path
// was previously verified only with a throwaway test file, deleted after the
// run. Nothing in the committed suite caught a regression to a silent drop.
//
// The scenario: a note that genuinely exists on disk, that filepath.WalkDir
// reaches, but whose parent directory name ends in a dot — a form Win32
// normally strips silently at open time. That normalization only happens on
// the traditional (non-\\?\) path parsing route, so a vault root padded past
// longPathThreshold (which makes Vault.walkRoot carry the \\?\ prefix per
// LongPath) lets WalkDir enumerate and descend into the trailing-dot
// directory without the OS ever trimming it away — and lets Canonicalize see
// (and reject) the trailing-dot component for what it is.
//
// Creating the trailing-dot directory itself requires the same \\?\ bypass.
// If directory/file creation is refused in this environment for any reason,
// the test skips rather than failing: what is under test is Walk's response
// to a Canonicalize rejection, not this environment's tolerance for
// trailing-dot names.
func TestWalkRecordsSkipOnCanonicalizeRejection(t *testing.T) {
	base := t.TempDir()

	// Pad the root past longPathThreshold (240 bytes) so Vault.walkRoot
	// carries the \\?\ prefix and WalkDir can descend into "Pasta ." without
	// Win32 trimming the trailing dot away first.
	padding := strings.Repeat("a", 220)
	root := filepath.Join(base, padding)
	if err := os.MkdirAll(vault.LongPath(root), 0o755); err != nil {
		t.Skipf("nao foi possivel criar raiz com padding para exceder o limiar de caminho longo: %v", err)
	}

	dotDir := filepath.Join(root, "Pasta .")
	if err := os.Mkdir(vault.LongPath(dotDir), 0o755); err != nil {
		t.Skipf("filesystem/SO recusou diretorio terminado em ponto, mesmo via \\\\?\\: %v", err)
	}

	notePath := filepath.Join(dotDir, "Nota.md")
	if err := os.WriteFile(vault.LongPath(notePath), []byte("# nota\n"), 0o644); err != nil {
		t.Skipf("filesystem/SO recusou arquivo dentro de diretorio terminado em ponto: %v", err)
	}

	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	var visited int
	err = v.Walk(context.Background(), func(vault.Entry) error {
		visited++
		return nil
	})
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}
	if visited != 0 {
		t.Fatalf("visited = %d, quer 0 — Nota.md dentro de \"Pasta .\" deveria ser rejeitada por Canonicalize, nunca chegar ao callback", visited)
	}

	count, samples := v.SkippedEntries()
	if count == 0 {
		t.Fatal("SkippedEntries count = 0, quer >0 apos a rejeicao de Canonicalize")
	}
	found := false
	for _, s := range samples {
		if strings.Contains(s, "Nota.md") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("SkippedEntries samples = %v, quer uma amostra citando Nota.md", samples)
	}
}
