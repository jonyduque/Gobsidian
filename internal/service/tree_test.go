package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func escreverEmPasta(t *testing.T, root, name, content string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("criando pasta de %s: %v", name, err)
	}
	writeFile(t, root, filepath.FromSlash(name), content)
}

// O cofre cobre os quatro formatos que A2.3 exige: pasta de um nivel so,
// cinco niveis (e o que o cofre Estudo tem), pasta sem nota direta com
// subpasta cheia, e nome com acento e espaco.
func montarCofreDeArvore(t *testing.T) *Service {
	t.Helper()
	root := t.TempDir()
	escreverEmPasta(t, root, "Raiz.md", "# Raiz\n")
	escreverEmPasta(t, root, "Solta/Uma.md", "# Uma\n")
	escreverEmPasta(t, root, "A/B/C/D/E/Funda.md", "# Funda\n")
	escreverEmPasta(t, root, "Vazia/Cheia/Um.md", "# Um\n")
	escreverEmPasta(t, root, "Vazia/Cheia/Dois.md", "# Dois\n")
	escreverEmPasta(t, root, "Direito Penal/Ação/Dolo.md", "---\ntitle: Dolo eventual\n---\ncorpo\n")
	// Anexo: o indice o conhece, mas a arvore e de NOTAS. Uma pasta que so
	// tem anexo nao entra, porque nao ha nota para o cliente abrir nela.
	escreverEmPasta(t, root, "Anexos/img.png", "png")
	return newTestService(t, root)
}

func pastasPorCaminho(r TreeResult) map[string]FolderItem {
	m := make(map[string]FolderItem, len(r.Folders))
	for _, f := range r.Folders {
		m[f.Path] = f
	}
	return m
}

func TestVaultTreeDerivaAsPastasDasNotas(t *testing.T) {
	svc := montarCofreDeArvore(t)

	res, err := svc.VaultTree(context.Background(), TreeRequest{Recursive: true})
	if err != nil {
		t.Fatalf("VaultTree: %v", err)
	}

	if res.Folder.Path != "" || res.Folder.NotesRecursive != 6 || res.Folder.Notes != 1 {
		t.Errorf("raiz = %+v, esperado caminho vazio, 1 nota direta e 6 no total", res.Folder)
	}

	want := map[string]FolderItem{
		"A":                  {Path: "A", Parent: "", Notes: 0, NotesRecursive: 1},
		"A/B":                {Path: "A/B", Parent: "A", Notes: 0, NotesRecursive: 1},
		"A/B/C":              {Path: "A/B/C", Parent: "A/B", Notes: 0, NotesRecursive: 1},
		"A/B/C/D":            {Path: "A/B/C/D", Parent: "A/B/C", Notes: 0, NotesRecursive: 1},
		"A/B/C/D/E":          {Path: "A/B/C/D/E", Parent: "A/B/C/D", Notes: 1, NotesRecursive: 1},
		"Direito Penal":      {Path: "Direito Penal", Parent: "", Notes: 0, NotesRecursive: 1},
		"Direito Penal/Ação": {Path: "Direito Penal/Ação", Parent: "Direito Penal", Notes: 1, NotesRecursive: 1},
		"Solta":              {Path: "Solta", Parent: "", Notes: 1, NotesRecursive: 1},
		"Vazia":              {Path: "Vazia", Parent: "", Notes: 0, NotesRecursive: 2},
		"Vazia/Cheia":        {Path: "Vazia/Cheia", Parent: "Vazia", Notes: 2, NotesRecursive: 2},
	}
	got := pastasPorCaminho(res)
	if len(got) != len(want) {
		t.Errorf("pastas = %d, esperado %d: %+v", len(got), len(want), res.Folders)
	}
	for k, w := range want {
		if g, ok := got[k]; !ok {
			t.Errorf("pasta %q ausente", k)
		} else if g != w {
			t.Errorf("pasta %q = %+v, esperado %+v", k, g, w)
		}
	}
	if _, ok := got["Anexos"]; ok {
		t.Error("pasta so com anexo entrou na arvore de notas")
	}

	for i := 1; i < len(res.Folders); i++ {
		if res.Folders[i-1].Path >= res.Folders[i].Path {
			t.Errorf("pastas fora de ordem: %q antes de %q", res.Folders[i-1].Path, res.Folders[i].Path)
		}
	}
	if len(res.Notes) != 6 {
		t.Fatalf("notas = %d, esperado 6: %+v", len(res.Notes), res.Notes)
	}
	for i := 1; i < len(res.Notes); i++ {
		if res.Notes[i-1].Path >= res.Notes[i].Path {
			t.Errorf("notas fora de ordem: %q antes de %q", res.Notes[i-1].Path, res.Notes[i].Path)
		}
	}
	for _, n := range res.Notes {
		if n.Path == "Direito Penal/Ação/Dolo.md" {
			if n.Title != "Dolo eventual" || n.Folder != "Direito Penal/Ação" {
				t.Errorf("nota com titulo de frontmatter = %+v", n)
			}
		}
		if n.Path == "Raiz.md" && n.Folder != "" {
			t.Errorf("nota da raiz com pasta %q", n.Folder)
		}
	}
}

// Pedida uma pasta sem Recursive, a resposta traz os FILHOS diretos: as
// subpastas e as notas dela, nunca os netos. E o que o read de um resource de
// pasta mostra.
func TestVaultTreeDeUmaPastaTrazSoOsFilhos(t *testing.T) {
	svc := montarCofreDeArvore(t)

	res, err := svc.VaultTree(context.Background(), TreeRequest{Folder: "Vazia"})
	if err != nil {
		t.Fatalf("VaultTree: %v", err)
	}
	if res.Folder.Path != "Vazia" || res.Folder.NotesRecursive != 2 {
		t.Errorf("pasta pedida = %+v", res.Folder)
	}
	if len(res.Folders) != 1 || res.Folders[0].Path != "Vazia/Cheia" {
		t.Errorf("subpastas = %+v, esperado so Vazia/Cheia", res.Folders)
	}
	if len(res.Notes) != 0 {
		t.Errorf("notas diretas = %+v, esperado nenhuma - as duas sao netas", res.Notes)
	}

	raiz, err := svc.VaultTree(context.Background(), TreeRequest{})
	if err != nil {
		t.Fatalf("VaultTree da raiz: %v", err)
	}
	if len(raiz.Notes) != 1 || raiz.Notes[0].Path != "Raiz.md" {
		t.Errorf("notas diretas da raiz = %+v", raiz.Notes)
	}
	if len(raiz.Folders) != 4 {
		t.Errorf("subpastas da raiz = %+v, esperado A, Direito Penal, Solta, Vazia", raiz.Folders)
	}
}

// A pasta pedida casa pela mesma chave de caminho que o indice usa: caixa e
// forma Unicode nao separam, e a barra final que a URI de pasta carrega tambem
// nao.
func TestVaultTreeCasaAPastaPelaChaveDeCaminho(t *testing.T) {
	svc := montarCofreDeArvore(t)

	for _, pedido := range []string{"direito penal/ação", "Direito Penal/Ação/", "Direito Penal/Ação"} {
		res, err := svc.VaultTree(context.Background(), TreeRequest{Folder: pedido})
		if err != nil {
			t.Errorf("VaultTree(%q): %v", pedido, err)
			continue
		}
		if res.Folder.Path != "Direito Penal/Ação" || len(res.Notes) != 1 {
			t.Errorf("VaultTree(%q) = %+v", pedido, res)
		}
	}
}

func TestVaultTreePastaInexistenteEErroDeDominio(t *testing.T) {
	svc := montarCofreDeArvore(t)

	for _, pedido := range []string{"Nada", "Anexos"} {
		_, err := svc.VaultTree(context.Background(), TreeRequest{Folder: pedido})
		if !errors.Is(err, &Error{Code: CodeFolderNotFound}) {
			t.Errorf("VaultTree(%q) err = %v, esperado FOLDER_NOT_FOUND", pedido, err)
		}
	}
}

func TestVaultTreeDeCofreVazio(t *testing.T) {
	svc := newTestService(t, t.TempDir())

	res, err := svc.VaultTree(context.Background(), TreeRequest{Recursive: true})
	if err != nil {
		t.Fatalf("VaultTree de cofre vazio: %v", err)
	}
	if len(res.Folders) != 0 || len(res.Notes) != 0 || res.Folder.NotesRecursive != 0 {
		t.Errorf("cofre vazio = %+v", res)
	}
}
