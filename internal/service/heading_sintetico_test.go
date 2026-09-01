package service

import (
	"context"
	"strings"
	"testing"
)

// TestReadNotePromoveCandidatoNaLeitura e a forma (b) da alternativa E: ler por
// heading resolve CANDIDATO quando nao ha heading ATX que case.
//
// Antes disto, uma nota convertida de PDF exigia duas chamadas — note_outline
// para achar o offset, note_read para ler ali. Agora resolve em uma.
func TestReadNotePromoveCandidatoNaLeitura(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "conv.md",
		"**13 Registro**\n\ntexto do capitulo\n\n**13.1 Substituicao**\n\no texto da subsecao\n")
	svc := newTestService(t, root)

	res, err := svc.ReadNote(context.Background(), ReadRequest{Path: "conv.md", Heading: "13.1 Substituicao"})
	if err != nil {
		t.Fatalf("ReadNote por candidato: %v", err)
	}
	if !strings.Contains(res.Content, "o texto da subsecao") {
		t.Fatalf("nao devolveu a secao do candidato: %q", res.Content)
	}
	if strings.Contains(res.Content, "texto do capitulo") {
		t.Fatal("devolveu alem da secao; os limites do candidato nao foram respeitados")
	}
	if !res.SectionSynthetic {
		t.Error("section_synthetic falso numa secao que veio de PALPITE — " +
			"sem esse campo a tool afirma estrutura que o arquivo nao tem")
	}
}

// TestReadNoteHeadingRealVenceCandidato e o contrapeso, e sem ele a promocao
// poderia trocar estrutura de verdade por palpite.
func TestReadNoteHeadingRealVenceCandidato(t *testing.T) {
	root := t.TempDir()
	// O MESMO texto aparece como heading ATX e como negrito. O ATX tem de ganhar.
	writeFile(t, root, "misto.md",
		"## Capitulo\n\nconteudo do heading real\n\n**Capitulo**\n\nconteudo do candidato\n")
	svc := newTestService(t, root)

	res, err := svc.ReadNote(context.Background(), ReadRequest{Path: "misto.md", Heading: "Capitulo"})
	if err != nil {
		t.Fatalf("ReadNote: %v", err)
	}
	if !strings.Contains(res.Content, "conteudo do heading real") {
		t.Fatalf("o candidato venceu o heading ATX: %q", res.Content)
	}
	if res.SectionSynthetic {
		t.Error("section_synthetic verdadeiro numa secao que veio de heading ATX")
	}
}

// TestReadNoteCandidatoAmbiguoNaoChuta: dois candidatos com o mesmo texto sao
// ambiguidade, e escolher um devolveria um trecho arbitrario com cara de
// resposta — a mesma regra que ja vale para heading ATX.
func TestReadNoteCandidatoAmbiguoNaoChuta(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "amb.md",
		"**Secao**\n\nprimeira\n\n**Outra**\n\nmeio\n\n**Secao**\n\nsegunda\n")
	svc := newTestService(t, root)

	_, err := svc.ReadNote(context.Background(), ReadRequest{Path: "amb.md", Heading: "Secao"})
	if err == nil {
		t.Fatal("dois candidatos com o mesmo texto e a leitura escolheu um")
	}
	if CodeOf(err) != CodeAmbiguousHeading {
		t.Errorf("codigo = %q, quer AMBIGUOUS_HEADING", CodeOf(err))
	}
}

// TestReadNoteSemCandidatoMantemAMensagemQueEnsina: a nota sem estrutura de
// tipo nenhum continua recebendo a mensagem que aponta note_outline, em vez de
// uma lista de disponiveis vazia.
func TestReadNoteSemCandidatoMantemAMensagemQueEnsina(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "plana.md", "so texto corrido, sem titulo de forma nenhuma\n")
	svc := newTestService(t, root)

	_, err := svc.ReadNote(context.Background(), ReadRequest{Path: "plana.md", Heading: "Qualquer"})
	if err == nil {
		t.Fatal("pedir heading numa nota sem estrutura nenhuma nao falhou")
	}
	if !strings.Contains(err.Error(), "note_outline") {
		t.Errorf("a mensagem perdeu o caminho de saida: %v", err)
	}
}

// TestReadNoteCandidatoPrecisaCasarOTexto e a regra que o teste de ambiguidade
// NAO pega: com tres candidatos, remover a conferencia de texto deixa os tres
// casando e a resposta continua "ambiguo".
//
// Com UM candidato so, remover a conferencia faz a leitura devolver a secao
// dele para QUALQUER heading pedido — inventando uma resposta para uma pergunta
// que nao tem. Uma prova de mutacao saiu EXIT=1 exatamente por essa lacuna.
func TestReadNoteCandidatoPrecisaCasarOTexto(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "um.md", "**Prescricao**\n\nconteudo da unica secao\n")
	svc := newTestService(t, root)

	// Controle: o candidato que EXISTE resolve.
	if _, err := svc.ReadNote(context.Background(), ReadRequest{Path: "um.md", Heading: "Prescricao"}); err != nil {
		t.Fatalf("o candidato existente nao resolveu: %v", err)
	}

	// A asserção: um heading que nao existe nao pode devolver o candidato.
	_, err := svc.ReadNote(context.Background(), ReadRequest{Path: "um.md", Heading: "Decadencia"})
	if err == nil {
		t.Fatal("heading inexistente devolveu a secao do unico candidato; " +
			"a leitura inventou resposta para pergunta que a nota nao responde")
	}
	if CodeOf(err) != CodeHeadingNotFound {
		t.Errorf("codigo = %q, quer HEADING_NOT_FOUND", CodeOf(err))
	}
}
