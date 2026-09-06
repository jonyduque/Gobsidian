package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func cofreComUmaNota(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "n.md"), []byte("---\nk: v\n---\n# N\n\n#tag texto\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestNoteMetadataIncludeInvalidoEInvalidArgument(t *testing.T) {
	svc := newTestService(t, cofreComUmaNota(t))
	_, err := svc.NoteMetadata(context.Background(), MetadataRequest{
		Path: "n.md", Include: []string{"headers"},
	})
	var se *Error
	if !errors.As(err, &se) || se.Code != CodeInvalidArgument {
		t.Fatalf("include=[\"headers\"]: quero INVALID_ARGUMENT, tenho %v", err)
	}
}

func TestNoteMetadataIncludeVazioEInvalidArgument(t *testing.T) {
	svc := newTestService(t, cofreComUmaNota(t))
	_, err := svc.NoteMetadata(context.Background(), MetadataRequest{
		Path: "n.md", Include: []string{""},
	})
	var se *Error
	if !errors.As(err, &se) || se.Code != CodeInvalidArgument {
		t.Fatalf("include=[\"\"]: quero INVALID_ARGUMENT, tenho %v", err)
	}
}

func TestNoteMetadataIncludeValidoContinuaAceito(t *testing.T) {
	svc := newTestService(t, cofreComUmaNota(t))
	res, err := svc.NoteMetadata(context.Background(), MetadataRequest{
		Path: "n.md", Include: []string{"tags", "inline_fields"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Tags) != 1 {
		t.Fatalf("tags = %v, quero [tag]", res.Tags)
	}
}
