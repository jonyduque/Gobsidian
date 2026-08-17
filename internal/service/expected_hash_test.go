package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// TestExpectedHashPegaEdicaoExternaAindaNaoIndexada e o cenario para o qual o
// campo foi escrito, e o unico que os testes de hoje nao cobrem.
//
// O truque e NAO deixar o indice ver a edicao: escrever no disco por baixo e nao
// reindexar. Um teste que reindexa antes de chamar a escrita mede o caso facil,
// em que disco e indice concordam — e esse ja passava com o codigo defeituoso.
func TestExpectedHashPegaEdicaoExternaAindaNaoIndexada(t *testing.T) {
	casos := []struct {
		nome    string
		chamada func(svc *Service, hash string) error
	}{
		{
			nome: "note_append",
			chamada: func(svc *Service, hash string) error {
				_, err := svc.AppendNote(context.Background(), AppendNoteRequest{
					Path: "nota.md", Content: "acrescentado\n", ExpectedHash: hash,
				})
				return err
			},
		},
		{
			nome: "note_patch",
			chamada: func(svc *Service, hash string) error {
				_, err := svc.PatchNote(context.Background(), PatchNoteRequest{
					Path: "nota.md", Heading: "Secao", Mode: "replace_section",
					Content: "trocado\n", ExpectedHash: hash,
				})
				return err
			},
		},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			root := t.TempDir()
			writeFile(t, root, "nota.md", "# Secao\n\noriginal\n")
			svc := newTestService(t, root)

			// O hash que o cliente teria em maos: o do estado indexado.
			antes, err := svc.ReadNote(context.Background(), ReadRequest{Path: "nota.md"})
			if err != nil {
				t.Fatalf("ReadNote inicial: %v", err)
			}
			hashDoCliente := antes.Hash
			if hashDoCliente == "" {
				t.Fatal("ReadNote devolveu Hash vazio; o teste nao tem o que comparar")
			}

			// Edicao externa DIRETO no disco, sem reindexar. E a janela do debounce.
			const editado = "# Secao\n\neditado por fora\n"
			abs := filepath.Join(root, "nota.md")
			if err := os.WriteFile(abs, []byte(editado), 0644); err != nil {
				t.Fatalf("escrevendo por fora: %v", err)
			}

			// Confere que a CONDICAO se montou: o indice ainda tem o hash velho.
			// Sem esta guarda, um teste que reindexou por acidente passaria por
			// engano — a licao da Task 103.
			depois, err := svc.ReadNote(context.Background(), ReadRequest{Path: "nota.md"})
			if err != nil {
				t.Fatalf("ReadNote apos edicao externa: %v", err)
			}
			if depois.Hash != hashDoCliente {
				t.Fatalf("o indice ja absorveu a edicao (%s -> %s); a condicao "+
					"deste teste nao se montou", hashDoCliente, depois.Hash)
			}

			err = c.chamada(svc, hashDoCliente)
			if err == nil {
				t.Fatal("a escrita aceitou expected_hash obsoleto e sobrescreveu a " +
					"edicao externa — o controle otimista falhou no seu unico caso")
			}
			if got := CodeOf(err); got != CodeHashMismatch {
				t.Fatalf("codigo = %v, quer %v", got, CodeHashMismatch)
			}

			// A prova de que nada foi gravado. Sem ela, um erro devolvido DEPOIS
			// da escrita passaria.
			atual, err := os.ReadFile(abs)
			if err != nil {
				t.Fatalf("relendo: %v", err)
			}
			if string(atual) != editado {
				t.Fatalf("o arquivo foi alterado apesar do erro:\n%q", atual)
			}
		})
	}
}

// TestExpectedHashCorretoAindaPassa e o controle. Sem ele, uma implementacao que
// recusasse TODA escrita com expected_hash passaria no teste acima.
func TestExpectedHashCorretoAindaPassa(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "nota.md", "# Secao\n\noriginal\n")
	svc := newTestService(t, root)

	res, err := svc.ReadNote(context.Background(), ReadRequest{Path: "nota.md"})
	if err != nil {
		t.Fatalf("ReadNote: %v", err)
	}
	if _, err := svc.AppendNote(context.Background(), AppendNoteRequest{
		Path: "nota.md", Content: "acrescentado\n", ExpectedHash: res.Hash,
	}); err != nil {
		t.Fatalf("AppendNote com expected_hash CORRETO foi recusado: %v", err)
	}
}
