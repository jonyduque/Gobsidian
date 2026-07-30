package service_test

import (
	"context"
	"testing"

	"github.com/jonyd/gobsidian/internal/service"
)

func TestMoveNote_BrokenAnchorsReportedOnlyWhenMissing(t *testing.T) {
	files := map[string]string{
		"alvo.md": "# CabecalhoValido\nConteudo da nota",
		"ref.md":  "Link com ancora quebrada [[alvo#Inexistente]] e link com ancora valida [[alvo#CabecalhoValido]]",
	}

	svc, _, _, _ := createMoveService(t, files)

	res, err := svc.MoveNote(context.Background(), service.MoveNoteRequest{
		From:          "alvo.md",
		To:            "Novo/alvo.md",
		UpdateLinks:   true,
		CreateFolders: true,
	})
	if err != nil {
		t.Fatalf("MoveNote: %v", err)
	}

	if len(res.BrokenAnchors) != 1 {
		t.Fatalf("len(BrokenAnchors) = %d; quer 1 (apenas a ancora inexistente)", len(res.BrokenAnchors))
	}

	expected := service.BrokenAnchor{
		From:   "ref.md",
		To:     "Novo/alvo.md",
		Anchor: "Inexistente",
	}

	if res.BrokenAnchors[0] != expected {
		t.Errorf("BrokenAnchors[0] = %+v; quer %+v", res.BrokenAnchors[0], expected)
	}
}

func TestDeleteNote_BrokenAnchorsReportedOnDeletion(t *testing.T) {
	files := map[string]string{
		"alvo.md": "# SecaoExistente\nConteudo",
		"ref.md":  "Link apontando para [[alvo#SecaoExistente]]",
	}

	svc, _, _, _ := createDeleteService(t, files)

	res, err := svc.DeleteNote(context.Background(), service.DeleteNoteRequest{
		Path:              "alvo.md",
		ToTrash:           true,
		ReportBrokenLinks: true,
	})
	if err != nil {
		t.Fatalf("DeleteNote: %v", err)
	}

	if len(res.BrokenAnchors) != 1 {
		t.Fatalf("len(BrokenAnchors) = %d; quer 1", len(res.BrokenAnchors))
	}

	expected := service.BrokenAnchor{
		From:   "ref.md",
		To:     "alvo.md",
		Anchor: "SecaoExistente",
	}

	if res.BrokenAnchors[0] != expected {
		t.Errorf("BrokenAnchors[0] = %+v; quer %+v", res.BrokenAnchors[0], expected)
	}
}
