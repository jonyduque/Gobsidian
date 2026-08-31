package service

import (
	"context"
	"strings"
	"testing"
)

// ptrDe existe porque os campos do objeto por item sao PONTEIROS, e escrever
// &valor em literal de struct nao compila para constante.
//
// Nome com sufixo de proposito: um helper chamado `ptr` no pacote de teste
// colide no dia em que a implementacao criar o dela.
func ptrDe[T any](v T) *T { return &v }

// TestReadNotesObjetoPorItemSobrepoeOTopo prova as duas metades do contrato: o
// item sem objeto HERDA os campos de topo, e o item com objeto sobrepoe SO o
// campo que ele traz.
//
// O erro previsivel aqui e sobrepor o REGISTRO INTEIRO em vez de campo a campo —
// e o sintoma seria max_bytes do topo sumindo no item que so pediu heading.
// A ultima asserção e a que pega isso; sem ela o teste passa com a versao errada.
func TestReadNotesObjetoPorItemSobrepoeOTopo(t *testing.T) {
	const secoes = "# Alfa\n\nconteudo de alfa bem mais longo que dez bytes\n\n" +
		"# Beta\n\nconteudo de beta bem mais longo que dez bytes\n"

	root := t.TempDir()
	writeFile(t, root, "a.md", secoes)
	writeFile(t, root, "b.md", secoes)
	svc := newTestService(t, root)

	out := svc.ReadNotes(context.Background(), ReadBatchRequest{
		Heading:  "Alfa", // padrao do topo
		MaxBytes: 10,     // padrao do topo
		Alvos: []ReadAlvo{
			{Path: "a.md"},                         // herda os dois
			{Path: "b.md", Heading: ptrDe("Beta")}, // sobrepoe SO o heading
		},
	})

	if len(out.Items) != 2 {
		t.Fatalf("items = %d, quer 2", len(out.Items))
	}
	for i, it := range out.Items {
		if it.Err != nil {
			t.Fatalf("item %d devolveu erro: %v", i, it.Err)
		}
	}
	if out.Items[0].Section == nil || out.Items[0].Section.Text != "Alfa" {
		t.Fatalf("item 0 nao herdou o heading do topo: %+v", out.Items[0].Section)
	}
	if out.Items[1].Section == nil || out.Items[1].Section.Text != "Beta" {
		t.Fatalf("item 1 nao sobrepos o heading: %+v", out.Items[1].Section)
	}
	if !out.Items[1].Truncated {
		t.Fatal("item 1 perdeu max_bytes=10 do topo ao sobrepor o heading — " +
			"a sobreposicao trocou o registro inteiro em vez de um campo")
	}
}

// TestReadNotesZeroExplicitoNaoEOmissao guarda a armadilha de ReadOnlySet e
// DebounceMSSet, um nivel abaixo: {"path":"a.md","max_bytes":0} e um pedido
// DIFERENTE de {"path":"a.md"}, e um campo nao-ponteiro nao distingue os dois.
func TestReadNotesZeroExplicitoNaoEOmissao(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "a.md", "corpo com bastante texto para ser cortado no meio\n")
	svc := newTestService(t, root)

	out := svc.ReadNotes(context.Background(), ReadBatchRequest{
		MaxBytes: 5,
		Alvos: []ReadAlvo{
			{Path: "a.md"},                     // herda 5
			{Path: "a.md", MaxBytes: ptrDe(0)}, // ZERO explicito: sem teto
		},
	})
	if len(out.Items[0].Content) != 5 {
		t.Fatalf("item 0: %d bytes, quer 5 (herdado do topo)", len(out.Items[0].Content))
	}
	if len(out.Items[1].Content) == 5 {
		t.Fatal("item 1: max_bytes=0 explicito foi tratado como omissao e herdou 5")
	}
}

// TestReadNotesItemInvalidoDizQualItem cobre a regra D-R-3 no nivel do item.
// "paths invalido" nao ajuda quem mandou seis capitulos numa chamada.
func TestReadNotesItemInvalidoDizQualItem(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "a.md", "# Alfa\n\ntexto\n")
	svc := newTestService(t, root)

	out := svc.ReadNotes(context.Background(), ReadBatchRequest{
		Alvos: []ReadAlvo{
			{Path: "a.md"},
			{Path: "a.md", Heading: ptrDe("Alfa"), Offset: ptrDe(int64(3))},
		},
	})
	if out.Items[0].Err != nil {
		t.Fatalf("item 0, que e valido, falhou: %v", out.Items[0].Err)
	}
	if out.Items[1].Err == nil {
		t.Fatal("item com offset E heading foi aceito")
	}
	if !strings.Contains(out.Items[1].Err.Error(), "1") {
		t.Errorf("o erro nao identifica o indice do item: %v", out.Items[1].Err)
	}
}

// TestReadNotesFormaAntigaContinuaValendo e o controle de compatibilidade: a
// lista so de strings, que e o que todo cliente manda hoje, nao pode quebrar.
func TestReadNotesFormaAntigaContinuaValendo(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "a.md", "alfa\n")
	writeFile(t, root, "b.md", "beta\n")
	svc := newTestService(t, root)

	out := svc.ReadNotes(context.Background(), ReadBatchRequest{
		Alvos: []ReadAlvo{{Path: "a.md"}, {Path: "b.md"}},
	})
	if len(out.Items) != 2 || out.Items[0].Err != nil || out.Items[1].Err != nil {
		t.Fatalf("lote simples quebrou: %+v", out.Items)
	}
}
