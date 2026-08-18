package service

import (
	"context"
	"testing"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/vault"
)

// TestNotaComFrontmatterQuebradoNaoSomeEmSilencio cobre as tres pontas de uma
// vez: o campo chega ao indice, note_metadata o expoe, e vault_stats o CONTA.
//
// A asserção sobre o contador e a que importa: sem ela, uma nota quebrada em mil
// e indistinguivel de zero notas quebradas, que e o estado de hoje.
func TestNotaComFrontmatterQuebradoNaoSomeEmSilencio(t *testing.T) {
	// YAML invalido de verdade: lista aberta que a linha seguinte nao fecha.
	const quebrada = "---\ntags: [a, b\ntitulo: sem fechar\n---\n\n# Corpo\n\ntexto util\n"
	const boa = "---\ntags: [x]\n---\n\n# Boa\n\ntexto\n"

	root := t.TempDir()
	writeFile(t, root, "quebrada.md", quebrada)
	writeFile(t, root, "boa.md", boa)
	svc := newTestService(t, root)

	md, err := svc.NoteMetadata(context.Background(), MetadataRequest{Path: "quebrada.md"})
	if err != nil {
		t.Fatalf("NoteMetadata: %v", err)
	}
	if md.FrontmatterErr == "" {
		t.Fatal("frontmatter malformado nao chegou a note_metadata: a nota perdeu " +
			"tags, aliases e titulo em silencio")
	}

	// O corpo continua util — frontmatter quebrado NAO invalida a nota, e o
	// comentario de parser.Parse:29 diz exatamente isso.
	if len(md.Headings) == 0 {
		t.Fatal("os headings do corpo sumiram junto com o frontmatter")
	}

	// A nota boa nao pode ganhar o campo.
	boaMD, err := svc.NoteMetadata(context.Background(), MetadataRequest{Path: "boa.md"})
	if err != nil {
		t.Fatalf("NoteMetadata(boa): %v", err)
	}
	if boaMD.FrontmatterErr != "" {
		t.Fatalf("nota com frontmatter valido ganhou FrontmatterErr=%q", boaMD.FrontmatterErr)
	}

	// Verifica que Replace (caminho do watcher/update) tambem propaga FrontmatterErr.
	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	ix, ok := svc.index.(*index.Index)
	if !ok {
		t.Fatal("svc.index nao e *index.Index")
	}
	if err := ix.Replace(context.Background(), v, "quebrada.md"); err != nil {
		t.Fatalf("Replace: %v", err)
	}
	mdAfterReplace, err := svc.NoteMetadata(context.Background(), MetadataRequest{Path: "quebrada.md"})
	if err != nil {
		t.Fatalf("NoteMetadata after Replace: %v", err)
	}
	if mdAfterReplace.FrontmatterErr == "" {
		t.Fatal("frontmatter malformado nao chegou a note_metadata apos Replace (update.go)")
	}

	st, err := svc.VaultStats(context.Background(), StatsRequest{IncludeHealth: true})
	if err != nil {
		t.Fatalf("VaultStats: %v", err)
	}
	if st.FrontmatterErrors == nil {
		t.Fatal("vault_stats nao reporta o contador de frontmatter quebrado")
	}
	if *st.FrontmatterErrors != 1 {
		t.Fatalf("vault_stats conta %d notas com frontmatter quebrado, quer 1",
			*st.FrontmatterErrors)
	}
}

// TestContadorDeFrontmatterZeraQuandoNaoHaQuebrada e o controle: sem ele, um
// contador que devolvesse 1 fixo passaria no teste acima.
func TestContadorDeFrontmatterZeraQuandoNaoHaQuebrada(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "boa.md", "---\ntags: [x]\n---\n\n# Boa\n\ntexto\n")
	svc := newTestService(t, root)

	st, err := svc.VaultStats(context.Background(), StatsRequest{IncludeHealth: true})
	if err != nil {
		t.Fatalf("VaultStats: %v", err)
	}
	if st.FrontmatterErrors == nil {
		t.Fatal("com include_health o contador tem de existir, mesmo sendo zero")
	}
	if *st.FrontmatterErrors != 0 {
		t.Fatalf("contador = %d num cofre sem nota quebrada", *st.FrontmatterErrors)
	}
}

// TestContadorDeFrontmatterAusenteSemIncludeHealth e a outra metade da mesma
// regra, e ela ja esta escrita em StatsResult: nil = nao pedido, &0 = pedido e
// nao ha nenhum. Um zero que significa as duas coisas nao informa nada.
func TestContadorDeFrontmatterAusenteSemIncludeHealth(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "boa.md", "---\ntags: [x]\n---\n\n# Boa\n\ntexto\n")
	svc := newTestService(t, root)

	st, err := svc.VaultStats(context.Background(), StatsRequest{})
	if err != nil {
		t.Fatalf("VaultStats: %v", err)
	}
	if st.FrontmatterErrors != nil {
		t.Fatalf("sem include_health o contador veio preenchido (%d): o cliente "+
			"nao distingue `nao ha` de `nao perguntei`", *st.FrontmatterErrors)
	}
}
