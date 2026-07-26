package vault

import (
	"errors"
	"fmt"
	"testing"
)

// TestRecordSkipCapsSamplesButNotCount is a whitebox permanent test for
// recordSkip/SkippedEntries, the fallback Finding 2 (fix pass 2) asks for in
// case the Windows Canonicalize-rejection scenario proves too brittle to run
// reliably in CI. It exercises three properties directly, without needing a
// real filesystem edge case:
//
//  1. The count (skipped.Load()) keeps incrementing past maxSkippedSamples —
//     it is not capped, only the sample slice is.
//  2. The sample slice stops growing at maxSkippedSamples.
//  3. SkippedEntries returns a copy, not an alias to the internal slice: a
//     caller mutating the returned slice must not corrupt v.skippedSamples.
func TestRecordSkipCapsSamplesButNotCount(t *testing.T) {
	v := &Vault{}

	total := maxSkippedSamples + 5
	for i := 0; i < total; i++ {
		v.recordSkip(fmt.Sprintf("caminho-%d", i), errors.New("motivo"))
	}

	count, samples := v.SkippedEntries()
	if count != int64(total) {
		t.Errorf("count = %d, quer %d — o contador nao deveria ser limitado pelo teto de amostras", count, total)
	}
	if len(samples) != maxSkippedSamples {
		t.Errorf("len(samples) = %d, quer %d (maxSkippedSamples)", len(samples), maxSkippedSamples)
	}

	// A amostra guardada deve conter as primeiras entradas, nao as ultimas —
	// recordSkip para de anexar assim que o teto e atingido.
	if len(samples) > 0 && samples[0] != "caminho-0: motivo" {
		t.Errorf("samples[0] = %q, quer conter a primeira entrada registrada", samples[0])
	}

	// SkippedEntries deve devolver uma copia: mutar o slice devolvido nao
	// pode corromper v.skippedSamples para a proxima chamada.
	if len(samples) > 0 {
		samples[0] = "adulterado"
	}
	_, samplesAgain := v.SkippedEntries()
	if len(samplesAgain) > 0 && samplesAgain[0] == "adulterado" {
		t.Fatal("SkippedEntries devolveu um alias do slice interno, nao uma copia — mutar o retorno corrompeu o estado do Vault")
	}

	// Uma chamada adicional depois do teto continua incrementando o
	// contador, mas nao a amostra.
	v.recordSkip("mais-um", errors.New("motivo"))
	count, samples = v.SkippedEntries()
	if count != int64(total)+1 {
		t.Errorf("count apos mais um recordSkip = %d, quer %d", count, total+1)
	}
	if len(samples) != maxSkippedSamples {
		t.Errorf("len(samples) apos o teto continua crescendo: %d, quer %d", len(samples), maxSkippedSamples)
	}
}
