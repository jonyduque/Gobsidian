package service_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/search"
	"github.com/jonyd/gobsidian/internal/service"
	"github.com/jonyd/gobsidian/internal/vault"
)

func createMoveService(t *testing.T, files map[string]string) (*service.Service, *vault.Vault, *index.Index, string) {
	t.Helper()
	root := t.TempDir()

	for relPath, content := range files {
		full := filepath.Join(root, filepath.FromSlash(relPath))
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		if err := os.WriteFile(full, []byte(content), 0644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
	}

	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}

	idx := index.New()
	if err := idx.Build(context.Background(), v); err != nil {
		t.Fatalf("idx.Build: %v", err)
	}

	inv := search.NewInverted()
	svc := service.New(v, idx, inv, nil, service.Options{})
	return svc, v, idx, root
}

func TestNoteMovePartialFailureReportsWhatWasApplied(t *testing.T) {
	// b.md fica numa subpasta de proposito. Em Unix o que impede a escrita e a
	// permissao do DIRETORIO, e travar a raiz inteira travaria tambem a.md —
	// que precisa ser reescrita com sucesso para o cenario ser "falha PARCIAL"
	// e nao "falha total". As chaves sao ordenadas em write.go:528, entao
	// "a.md" e reescrita antes de "sub/b.md".
	files := map[string]string{
		"alvo.md":  "# Alvo\nConteudo",
		"a.md":     "Link para [[alvo]]",
		"sub/b.md": "Link para [[alvo]]",
	}

	svc, _, _, root := createMoveService(t, files)

	// A regra de qual objeto travar muda com a plataforma; ver
	// inescrivel_windows_test.go e inescrivel_unix_test.go.
	tornaInescrivel(t, filepath.Join(root, "sub"), "b.md")

	res, err := svc.MoveNote(context.Background(), service.MoveNoteRequest{
		From:          "alvo.md",
		To:            "Novo/alvo.md",
		UpdateLinks:   true,
		CreateFolders: true,
	})

	if err == nil {
		t.Fatal("esperava falha parcial; o cenario tornou uma nota inescrivel")
	}

	// 1. O relatorio deve listar a.md que foi reescrita antes da falha
	if len(res.Rewritten) == 0 {
		t.Errorf("Rewritten esta vazio; esperava registrar [a.md] antes da falha")
	}

	// 2. O CORPO JA SE MOVEU, e isso mudou em 2026-08-26.
	//
	// Ate entao este teste exigia o oposto — "o alvo.md NAO pode ter se
	// movido" —, porque os citantes eram reescritos ANTES do corpo. Essa ordem
	// tinha um custo pior, medido em teste: quando a movimentacao falhava, os
	// citantes ja estavam gravados apontando para um destino que nunca
	// existiu, e a nota continuava na origem. Todo link quebrado, e nenhum
	// deles apontando para nada.
	//
	// A inversao (A1) troca quem fica inconsistente. Agora o corpo se move
	// primeiro, e uma falha de citante deixa APENAS os citantes ainda nao
	// reescritos apontando para o caminho antigo — um subconjunto, visivel, e
	// recuperavel repetindo a atualizacao de links. A nota esta onde o usuario
	// pediu.
	//
	// A garantia antiga ("se nao der para terminar, nao mova") foi trocada de
	// propósito, nao perdida por descuido. O preco esta registrado em
	// docs/SUGESTOES.md, na analise de alternativas do A1.
	if _, err := os.Stat(filepath.Join(root, "Novo", "alvo.md")); err != nil {
		t.Errorf("o corpo NAO se moveu apesar de a falha ser posterior a ele: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "alvo.md")); err == nil {
		t.Error("a nota ficou nos DOIS caminhos: o corpo moveu e a origem sobreviveu")
	}

	// 3. O citante que NAO foi reescrito continua apontando para o caminho
	//    antigo — inconsistencia visivel, que e o ponto da troca.
	b, errB := os.ReadFile(filepath.Join(root, "sub", "b.md"))
	if errB == nil && !strings.Contains(string(b), "[[alvo]]") {
		t.Errorf("sub/b.md foi reescrita apesar de o cenario torna-la inescrivel: %q", string(b))
	}
}

func TestMoveNote_DryRunLeavesMtimeIntact(t *testing.T) {
	files := map[string]string{
		"alvo.md": "Alvo",
		"ref.md":  "Ref para [[alvo]]",
	}

	svc, _, _, root := createMoveService(t, files)

	alvoPath := filepath.Join(root, "alvo.md")
	refPath := filepath.Join(root, "ref.md")

	infoBeforeAlvo, _ := os.Stat(alvoPath)
	infoBeforeRef, _ := os.Stat(refPath)

	time.Sleep(10 * time.Millisecond)

	res, err := svc.MoveNote(context.Background(), service.MoveNoteRequest{
		From:          "alvo.md",
		To:            "Novo/alvo.md",
		UpdateLinks:   true,
		CreateFolders: true,
		DryRun:        true,
	})
	if err != nil {
		t.Fatalf("MoveNote dry_run: %v", err)
	}

	if !res.DryRun {
		t.Error("esperado DryRun=true no resultado")
	}
	if len(res.Diffs) == 0 {
		t.Error("esperava diffs populados em dry_run")
	}

	infoAfterAlvo, _ := os.Stat(alvoPath)
	infoAfterRef, _ := os.Stat(refPath)

	if !infoBeforeAlvo.ModTime().Equal(infoAfterAlvo.ModTime()) {
		t.Error("mtime de alvo.md foi alterado durante dry_run")
	}
	if !infoBeforeRef.ModTime().Equal(infoAfterRef.ModTime()) {
		t.Error("mtime de ref.md foi alterado durante dry_run")
	}
}

func TestMoveNote_UpdateLinksFalse(t *testing.T) {
	files := map[string]string{
		"alvo.md": "Alvo",
		"ref.md":  "Ref para [[alvo]]",
	}

	svc, _, _, root := createMoveService(t, files)
	refPath := filepath.Join(root, "ref.md")

	res, err := svc.MoveNote(context.Background(), service.MoveNoteRequest{
		From:        "alvo.md",
		To:          "destino.md",
		UpdateLinks: false,
	})
	if err != nil {
		t.Fatalf("MoveNote update_links=false: %v", err)
	}

	if res.LinksUpdated != 0 || len(res.Rewritten) != 0 {
		t.Errorf("esperava 0 links atualizados, obtido %d", res.LinksUpdated)
	}

	refRaw, _ := os.ReadFile(refPath)
	if string(refRaw) != "Ref para [[alvo]]" {
		t.Errorf("ref.md foi alterado apesar de update_links=false: %s", refRaw)
	}
}

func TestMoveNote_CreateFoldersFalseMissingDir(t *testing.T) {
	files := map[string]string{
		"alvo.md": "Alvo",
	}

	svc, _, _, _ := createMoveService(t, files)

	_, err := svc.MoveNote(context.Background(), service.MoveNoteRequest{
		From:          "alvo.md",
		To:            "PastaInexistente/alvo.md",
		CreateFolders: false,
	})

	if err == nil {
		t.Fatal("esperava erro ao mover para pasta inexistente com create_folders=false")
	}

	var codeErr *service.Error
	if errors.As(err, &codeErr) {
		if codeErr.Code != service.CodeFolderNotFound {
			t.Errorf("esperava CodeFolderNotFound, obtido %v", codeErr.Code)
		}
	}
}

func TestMoveNote_OutsideVaultAndAlreadyExists(t *testing.T) {
	files := map[string]string{
		"a.md": "Nota A",
		"b.md": "Nota B",
	}

	svc, _, _, _ := createMoveService(t, files)

	// Destino fora do cofre
	_, err := svc.MoveNote(context.Background(), service.MoveNoteRequest{
		From: "a.md",
		To:   "../fora.md",
	})
	if err == nil {
		t.Error("esperava erro para destino fora do cofre")
	}

	// Destino ja existe
	_, err = svc.MoveNote(context.Background(), service.MoveNoteRequest{
		From: "a.md",
		To:   "b.md",
	})
	if err == nil {
		t.Error("esperava erro para destino existente b.md")
	}
}

func TestMoveNote_PreservesAliasAndAnchor(t *testing.T) {
	files := map[string]string{
		"origem.md": "# Origem",
		"ref.md":    "Link [[origem#Secao|Alias]] e [[origem#^bloco]]",
	}

	svc, _, _, root := createMoveService(t, files)

	res, err := svc.MoveNote(context.Background(), service.MoveNoteRequest{
		From:          "origem.md",
		To:            "Pasta/destino.md",
		UpdateLinks:   true,
		CreateFolders: true,
	})
	if err != nil {
		t.Fatalf("MoveNote: %v", err)
	}

	if res.LinksUpdated != 2 {
		t.Errorf("esperava 2 links atualizados, obtido %d", res.LinksUpdated)
	}

	refPath := filepath.Join(root, "ref.md")
	refRaw, _ := os.ReadFile(refPath)
	want := "Link [[destino#Secao|Alias]] e [[destino#^bloco]]"
	if string(refRaw) != want {
		t.Errorf("obtido %q, quer %q", string(refRaw), want)
	}
}

// TestMoveNote_HappyPathActuallyMovesTheFile e o teste que faltava ao M5.
//
// Medido na revisao de 2026-07-30: neutralizando a escrita do arquivo de
// destino dentro de MoveNote, a suite INTEIRA de internal/service continuava
// verde. Os testes de move cobriam dry-run, update_links:false,
// create_folders:false, caminho fora do cofre, preservacao de alias e ancora, e
// falha parcial — e nenhum afirmava que a nota chega ao caminho novo.
//
// O caminho feliz da ferramenta mais perigosa do marco estava sem verificacao:
// note_move reescreve links em dezenas de notas que o usuario nao pediu para
// tocar, e um move que reescreve tudo e nao move o arquivo deixa o cofre com
// dezenas de links apontando para um caminho vazio.
func TestMoveNote_HappyPathActuallyMovesTheFile(t *testing.T) {
	conteudo := "---\ntags: [civil]\n---\r\n\r\n# Alvo\r\n\r\nconteudo que precisa chegar inteiro\r\n"
	svc, _, idx, root := createMoveService(t, map[string]string{
		"alvo.md": conteudo,
		"ref.md":  "aponta para [[alvo]] e para [texto](alvo.md)\n",
	})

	// CreateFolders explicito: no nivel do SERVICO o zero-value false e
	// legitimo, porque e uma API Go. Quem aplica o "default: true" do
	// docs/TOOLS.md e a camada da tool, em internal/mcpsrv — e foi la que a
	// revisao de 2026-07-30 achou tres campos com bool simples onde o schema
	// promete true.
	res, err := svc.MoveNote(context.Background(), service.MoveNoteRequest{
		From:          "alvo.md",
		To:            "Novo/alvo.md",
		UpdateLinks:   true,
		CreateFolders: true,
	})
	if err != nil {
		t.Fatalf("MoveNote: %v", err)
	}

	// 1. O arquivo esta no lugar novo, com o conteudo byte a byte. CRLF e
	//    frontmatter incluidos: RF-38 vale no move como vale na escrita.
	destino := filepath.Join(root, "Novo", "alvo.md")
	lido, err := os.ReadFile(destino)
	if err != nil {
		t.Fatalf("a nota nao chegou ao caminho novo: %v", err)
	}
	if string(lido) != conteudo {
		t.Errorf("conteudo no destino difere do original:\n got %q\nwant %q", lido, conteudo)
	}

	// 2. E saiu do lugar antigo. Um "move" que copia deixa duas notas, e o
	//    cofre passa a ter duplicata que o indice reporta como duas.
	if _, err := os.Stat(filepath.Join(root, "alvo.md")); !os.IsNotExist(err) {
		t.Errorf("a nota continua no caminho antigo (err=%v)", err)
	}

	// 3. O retorno diz o caminho novo — o chamador usa isso para a proxima
	//    chamada, e um campo errado aqui manda o modelo para o lugar errado.
	if res.To != "Novo/alvo.md" {
		t.Errorf("res.To = %q, quer %q", res.To, "Novo/alvo.md")
	}

	// NAO se afirma o indice aqui, e a razao importa: NENHUMA tool de escrita
	// deste projeto atualiza o indice diretamente — todas leem dele e deixam a
	// atualizacao para o watcher, que ve a propria escrita chegar como evento.
	// E consistente entre CreateNote, AppendNote, PatchNote, MoveNote e
	// DeleteNote, e nao e defeito.
	//
	// O que NAO esta verificado em lugar nenhum e a COMPOSICAO: com o servidor
	// rodando, note_move remove o arquivo antigo e escreve um novo com conteudo
	// identico, o que o watcher tem de correlacionar como rename e aplicar via
	// index.MoveNote. O brief da Task 65 pedia esse teste ponta a ponta
	// ("mover com o servidor rodando e confirmar que vault_search e link_graph
	// refletem o caminho novo") e ele nao foi escrito. Registrado na revisao de
	// 2026-07-30; fechar isso e tarefa, nao assercao para este teste de unidade.
	_ = idx
}
