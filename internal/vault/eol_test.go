package vault_test

import (
	"bytes"
	"testing"

	"github.com/jonyd/gobsidian/internal/vault"
)

func TestDetectEOL(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want vault.EOLStyle
	}{
		{"lf puro", "a\nb\nc\n", vault.EOLLF},
		{"crlf puro", "a\r\nb\r\nc\r\n", vault.EOLCRLF},
		{"misto majoritariamente crlf", "a\r\nb\r\nc\n", vault.EOLCRLF},
		{"misto majoritariamente lf", "a\nb\nc\r\n", vault.EOLLF},
		{"sem quebra alguma", "linha unica", vault.EOLLF},
		{"vazio", "", vault.EOLLF},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := vault.DetectEOL([]byte(tt.in)); got != tt.want {
				t.Errorf("DetectEOL(%q) = %v, quer %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestStripBOM(t *testing.T) {
	withBOM := append([]byte{0xEF, 0xBB, 0xBF}, []byte("# Titulo\n")...)

	body, had := vault.StripBOM(withBOM)
	if !had {
		t.Error("StripBOM nao detectou o BOM")
	}
	if !bytes.Equal(body, []byte("# Titulo\n")) {
		t.Errorf("body = %q, quer %q", body, "# Titulo\n")
	}

	body, had = vault.StripBOM([]byte("# Titulo\n"))
	if had {
		t.Error("StripBOM reportou BOM inexistente")
	}
	if !bytes.Equal(body, []byte("# Titulo\n")) {
		t.Errorf("body = %q, inalterado esperado", body)
	}
}
