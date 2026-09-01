package service

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// notaConvertida monta a fixture que as fixtures existentes NAO cobrem: nota sem
// heading ATX nenhum, com titulo em negrito, CRLF e bloco de codigo cercado.
//
// As fixtures deste projeto sao notas Obsidian idiomaticas, e nelas o produto
// inteiro funciona. "Cofre Obsidian idiomatico" nunca foi escrito como premissa,
// e por isso nunca foi questionado — ate uma nota de 255 KB convertida de livro
// derrubar a sessao real de 2026-08-15.
func notaConvertida() string {
	return "**13 Registro de candidatura**\r\n" +
		"\r\n" +
		"texto de abertura do capitulo\r\n" +
		"\r\n" +
		"**13.1.10 Substituicao de candidatos**\r\n" +
		"\r\n" +
		"o texto da secao de substituicao\r\n" +
		"\r\n" +
		"```\r\n" +
		"**isto esta dentro de bloco de codigo**\r\n" +
		"```\r\n"
}

// TestOutlineNaoConfundeCandidatoComHeading e a asserção central da tool: ela
// pode ERRAR na deteccao de candidato sem causar dano, mas nao pode apresentar
// candidato como heading.
func TestOutlineNaoConfundeCandidatoComHeading(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "conv.md", notaConvertida())
	svc := newTestService(t, root)

	out, err := svc.Outline(context.Background(), OutlineRequest{Path: "conv.md"})
	if err != nil {
		t.Fatalf("Outline: %v", err)
	}
	if len(out.Headings) != 0 {
		t.Fatalf("nota sem heading ATX devolveu %d headings: %+v",
			len(out.Headings), out.Headings)
	}
	if len(out.Candidates) != 2 {
		t.Fatalf("quer 2 candidatos (o de dentro da cerca nao conta), tem %d: %+v",
			len(out.Candidates), out.Candidates)
	}
	if out.Candidates[0].Level == nil || *out.Candidates[0].Level != 1 {
		t.Fatalf("13 devia dar nivel 1, deu %v", out.Candidates[0].Level)
	}
	if out.Candidates[1].Level == nil || *out.Candidates[1].Level != 3 {
		t.Fatalf("13.1.10 devia dar nivel 3, deu %v", out.Candidates[1].Level)
	}
}

// TestOutlineOffsetAlimentaNoteRead prova que os dois lados falam a mesma
// coordenada. Sem isto, a tool devolve numeros bonitos e inuteis.
func TestOutlineOffsetAlimentaNoteRead(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "conv.md", notaConvertida())
	svc := newTestService(t, root)

	out, err := svc.Outline(context.Background(), OutlineRequest{Path: "conv.md"})
	if err != nil {
		t.Fatalf("Outline: %v", err)
	}
	c := out.Candidates[1]
	lido, err := svc.ReadNote(context.Background(), ReadRequest{
		Path: "conv.md", Offset: &c.Start, MaxBytes: int(c.End - c.Start),
	})
	if err != nil {
		t.Fatalf("ReadNote(offset=%d): %v", c.Start, err)
	}
	if !strings.HasPrefix(lido.Content, "**13.1.10") {
		t.Fatalf("ler no offset do candidato deu %q", lido.Content)
	}
	if !strings.Contains(lido.Content, "o texto da secao de substituicao") {
		t.Fatal("o recorte parou antes do corpo da secao: End nao fecha onde devia")
	}
}

// TestOutlineOffsetSobreviveAoBOM e o controle do teste acima. Os offsets do
// candidato saem de bytes lidos do disco; se o BOM nao for contado, todos eles
// erram por tres bytes — e tres bytes bastam para o recorte comecar no meio do
// asterisco.
func TestOutlineOffsetSobreviveAoBOM(t *testing.T) {
	root := t.TempDir()
	comBOM := append([]byte{0xEF, 0xBB, 0xBF}, []byte(notaConvertida())...)
	if err := os.WriteFile(filepath.Join(root, "bom.md"), comBOM, 0o644); err != nil {
		t.Fatal(err)
	}
	svc := newTestService(t, root)

	out, err := svc.Outline(context.Background(), OutlineRequest{Path: "bom.md"})
	if err != nil {
		t.Fatalf("Outline: %v", err)
	}
	if len(out.Candidates) != 2 {
		t.Fatalf("candidatos = %d, quer 2", len(out.Candidates))
	}
	c := out.Candidates[1]
	lido, err := svc.ReadNote(context.Background(), ReadRequest{
		Path: "bom.md", Offset: &c.Start, MaxBytes: 20,
	})
	if err != nil {
		t.Fatalf("ReadNote(offset=%d): %v", c.Start, err)
	}
	if !strings.HasPrefix(lido.Content, "**13.1.10") {
		t.Fatalf("com BOM, ler no offset do candidato deu %q", lido.Content)
	}
}

// TestOutlineSeparaEstruturaDePalpite exercita a nota que tem as DUAS coisas.
// Sem ela, os testes acima passariam com uma implementacao que jogasse tudo no
// mesmo campo desde que a nota so tivesse uma das duas formas.
func TestOutlineSeparaEstruturaDePalpite(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "misto.md", "# Titulo real\n\ntexto\n\n**14.2 Palpite**\n\nmais texto\n")
	svc := newTestService(t, root)

	out, err := svc.Outline(context.Background(), OutlineRequest{Path: "misto.md"})
	if err != nil {
		t.Fatalf("Outline: %v", err)
	}
	if len(out.Headings) != 1 || out.Headings[0].Text != "Titulo real" {
		t.Fatalf("headings = %+v, quer so o ATX", out.Headings)
	}
	if len(out.Candidates) != 1 || out.Candidates[0].Text != "14.2 Palpite" {
		t.Fatalf("candidates = %+v, quer so o negrito", out.Candidates)
	}
	for _, h := range out.Headings {
		for _, c := range out.Candidates {
			if h.Text == c.Text {
				t.Fatalf("%q aparece nos dois campos", h.Text)
			}
		}
	}
}

// TestOutlineTetoNaoESilencioso: cortar sem dizer e o defeito, nao o corte.
func TestOutlineTetoNaoESilencioso(t *testing.T) {
	root := t.TempDir()
	var b strings.Builder
	for i := range 10 {
		b.WriteString("**")
		b.WriteString(string(rune('a' + i)))
		b.WriteString(" secao**\n\ntexto\n\n")
	}
	writeFile(t, root, "muitos.md", b.String())
	svc := newTestService(t, root)

	out, err := svc.Outline(context.Background(), OutlineRequest{Path: "muitos.md", MaxCandidates: 3})
	if err != nil {
		t.Fatalf("Outline: %v", err)
	}
	if len(out.Candidates) != 3 {
		t.Fatalf("candidatos = %d, quer 3 (o teto pedido)", len(out.Candidates))
	}
	if !out.Truncated {
		t.Fatal("cortou de 10 para 3 e devolveu truncated=false")
	}

	// Contrapeso: sem corte, truncated tem de ser falso — senao o campo esta
	// preso em verdadeiro e nao informa nada.
	out2, err := svc.Outline(context.Background(), OutlineRequest{Path: "muitos.md"})
	if err != nil {
		t.Fatalf("Outline: %v", err)
	}
	if out2.Truncated {
		t.Fatalf("10 candidatos sob o padrao de %d e truncated=true", CandidatosPadrao)
	}
}

// TestReadNoteSemHeadingApontaParaOutline fecha o passo 5 do brief da Task 112:
// a nota convertida cai sempre no caso "nenhum heading", e ate agora a resposta
// era uma lista de disponiveis VAZIA, que nao diz por que.
func TestReadNoteSemHeadingApontaParaOutline(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "conv.md", notaConvertida())
	svc := newTestService(t, root)

	_, err := svc.ReadNote(context.Background(), ReadRequest{Path: "conv.md", Heading: "13.1.10"})
	if err == nil {
		t.Fatal("pedir heading numa nota sem heading nenhum nao falhou")
	}
	if !strings.Contains(err.Error(), "note_outline") {
		t.Errorf("a mensagem nao aponta o caminho de saida: %v", err)
	}
}

// TestOutlineListasVaziasNaoSaoNulas foi escrito depois de uma chamada REAL
// contra o test-vault devolver `"headings": null`.
//
// Slice nil vira `null` no JSON, e ai "esta nota nao tem heading nenhum" le
// igual a "nao sei dizer" — que sao exatamente as duas respostas que esta tool
// existe para separar. O teste unitario nao pegava porque compara len(), e
// len(nil) e zero.
func TestOutlineListasVaziasNaoSaoNulas(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "vazia.md", "so texto corrido, sem titulo de forma nenhuma\n")
	svc := newTestService(t, root)

	out, err := svc.Outline(context.Background(), OutlineRequest{Path: "vazia.md"})
	if err != nil {
		t.Fatalf("Outline: %v", err)
	}
	if out.Headings == nil {
		t.Error("Headings nil: serializa como null, e null nao e lista vazia")
	}
	if out.Candidates == nil {
		t.Error("Candidates nil: serializa como null, e null nao e lista vazia")
	}
}

// TestOutlineNaoInventaCandidatoNoFrontmatter fixa o defeito que foi publicado
// na v1.3.0 e na v1.3.1.
//
// O `---` que FECHA o frontmatter e um sublinhado setext valido, e passar o
// arquivo inteiro a DetectCandidates promovia a ultima linha do frontmatter a
// titulo. Medido nos cofres reais em 2026-09-01: um falso por nota com
// frontmatter, 1.274 em 1.275 notas no cofre Revisao.
//
// Nenhum teste desta tool tinha frontmatter, e foi so por isso que passou.
func TestOutlineNaoInventaCandidatoNoFrontmatter(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "fm.md", "---\ntitle: Minha Nota\ntags: [a, b]\n---\n\nTexto corrido, sem titulo nenhum.\n")
	svc := newTestService(t, root)

	out, err := svc.Outline(context.Background(), OutlineRequest{Path: "fm.md"})
	if err != nil {
		t.Fatalf("Outline: %v", err)
	}
	if len(out.Candidates) != 0 {
		t.Fatalf("frontmatter comum produziu %d candidato(s): %+v", len(out.Candidates), out.Candidates)
	}
}

// TestOutlineOffsetCorretoDepoisDoFrontmatter e o contrapeso: pular o
// frontmatter nao pode deslocar os offsets do corpo, senao a correcao troca um
// candidato inventado por um candidato que aponta para o lugar errado.
func TestOutlineOffsetCorretoDepoisDoFrontmatter(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "fm2.md", "---\ntitle: X\n---\n\n**13.1 Secao de verdade**\n\ncorpo da secao\n")
	svc := newTestService(t, root)

	out, err := svc.Outline(context.Background(), OutlineRequest{Path: "fm2.md"})
	if err != nil {
		t.Fatalf("Outline: %v", err)
	}
	if len(out.Candidates) != 1 {
		t.Fatalf("candidatos = %d, quer 1: %+v", len(out.Candidates), out.Candidates)
	}
	c := out.Candidates[0]
	lido, err := svc.ReadNote(context.Background(), ReadRequest{Path: "fm2.md", Offset: &c.Start, MaxBytes: 26})
	if err != nil {
		t.Fatalf("ReadNote(offset=%d): %v", c.Start, err)
	}
	if !strings.HasPrefix(lido.Content, "**13.1") {
		t.Fatalf("ler no offset do candidato deu %q", lido.Content)
	}
}
