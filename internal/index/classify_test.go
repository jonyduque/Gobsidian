package index_test

import (
	"context"
	"reflect"
	"slices"
	"sort"
	"testing"
	"time"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/vault"
)

// construirPorBuild monta o indice pelo caminho do boot: uma varredura so.
func construirPorBuild(t *testing.T, root string) *index.Index {
	t.Helper()
	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	ix := index.New()
	if err := ix.Build(context.Background(), v); err != nil {
		t.Fatalf("Build: %v", err)
	}
	return ix
}

// construirPorReplace monta o MESMO cofre pelo caminho do watcher: um indice
// vazio e um Replace por arquivo, na ordem em que a varredura os entrega.
//
// E a segunda construcao do indice. Ela existe no produto (todo evento do
// watcher passa por Replace) e ate agora ninguem tinha comparado o que ela
// produz com o que Build produz.
func construirPorReplace(t *testing.T, root string) *index.Index {
	t.Helper()
	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}

	var caminhos []vault.CanonicalPath
	if err := v.Walk(context.Background(), func(e vault.Entry) error {
		caminhos = append(caminhos, e.Path)
		return nil
	}); err != nil {
		t.Fatalf("Walk: %v", err)
	}
	slices.Sort(caminhos)

	ix := index.New()
	for _, p := range caminhos {
		if err := ix.Replace(context.Background(), v, p); err != nil {
			t.Fatalf("Replace %s: %v", p, err)
		}
	}
	return ix
}

// compararNotasCampoACampo percorre os campos de index.Note por reflexao, e
// nao por uma lista escrita a mao, de proposito: campo novo entra na comparacao
// sozinho. Conferir valores escritos a mao em cada lado nao pega divergencia
// nenhuma — foi comparando as duas construcoes campo a campo que a divergencia
// de DocLength entre indice construido e indice recarregado apareceu.
func compararNotasCampoACampo(t *testing.T, caminho vault.CanonicalPath, doBuild, doReplace *index.Note) {
	t.Helper()

	vb := reflect.ValueOf(*doBuild)
	vr := reflect.ValueOf(*doReplace)
	tipo := vb.Type()

	for i := range tipo.NumField() {
		nome := tipo.Field(i).Name
		x := vb.Field(i).Interface()
		y := vr.Field(i).Interface()

		// time.Time carrega relogio monotonico e ponteiro de fuso; DeepEqual
		// compara os dois e acusaria diferenca onde o instante e o mesmo.
		if tb, ok := x.(time.Time); ok {
			if !tb.Equal(y.(time.Time)) {
				t.Errorf("%s: campo %s diverge — Build=%v, Replace=%v", caminho, nome, tb, y)
			}
			continue
		}

		if !reflect.DeepEqual(x, y) {
			t.Errorf("%s: campo %s diverge — Build=%#v, Replace=%#v", caminho, nome, x, y)
		}
	}
}

func ordenarBacklinks(bls []index.Backlink) []index.Backlink {
	ordenados := append([]index.Backlink(nil), bls...)
	sort.Slice(ordenados, func(i, j int) bool {
		if ordenados[i].From != ordenados[j].From {
			return ordenados[i].From < ordenados[j].From
		}
		if ordenados[i].Anchor != ordenados[j].Anchor {
			return ordenados[i].Anchor < ordenados[j].Anchor
		}
		return ordenados[i].Alias < ordenados[j].Alias
	})
	return ordenados
}

// compararConstrucoes e a assercao central da Task 95: as duas construcoes do
// indice tem de responder igual para a mesma entrada.
//
// Nao afirma valores; afirma IGUALDADE entre as duas. Um teste que conferisse
// valores escritos a mao em cada lado passaria feliz com as duas erradas do
// mesmo jeito, e passou a existir exatamente porque as duas estavam certas de
// jeitos diferentes: Build registrava o placeholder de nuvem como Note e
// Replace o registrava como Asset.
func compararConstrucoes(t *testing.T, root string) {
	t.Helper()

	doBuild := construirPorBuild(t, root)
	doReplace := construirPorReplace(t, root)

	if got, want := doReplace.NoteCount(), doBuild.NoteCount(); got != want {
		t.Errorf("NoteCount: Build=%d, Replace=%d", want, got)
	}
	if got, want := doReplace.AssetCount(), doBuild.AssetCount(); got != want {
		t.Errorf("AssetCount: Build=%d, Replace=%d", want, got)
	}
	if got, want := doReplace.TotalSize(), doBuild.TotalSize(); got != want {
		t.Errorf("TotalSize: Build=%d, Replace=%d", want, got)
	}

	caminhosBuild := doBuild.Paths()
	caminhosReplace := doReplace.Paths()
	if !reflect.DeepEqual(caminhosBuild, caminhosReplace) {
		t.Errorf("Paths diverge — Build=%v, Replace=%v", caminhosBuild, caminhosReplace)
	}

	// Guarda da montagem: um cofre vazio faria todas as comparacoes acima
	// passarem sem comparar nada.
	if len(caminhosBuild) == 0 {
		t.Fatal("o cofre de comparacao esta vazio; nada foi comparado")
	}

	for _, p := range caminhosBuild {
		nb, okB := doBuild.Get(p)
		nr, okR := doReplace.Get(p)

		// A propria divergencia da Task 95 aparece aqui: um `.md`
		// somente-nuvem dava ok=true por Build e ok=false por Replace.
		if okB != okR {
			t.Errorf("%s: Get devolve %v por Build e %v por Replace — mesma entrada, duas respostas", p, okB, okR)
			continue
		}
		if okB {
			compararNotasCampoACampo(t, p, nb, nr)
		}

		blB := ordenarBacklinks(doBuild.Backlinks(p))
		blR := ordenarBacklinks(doReplace.Backlinks(p))
		if !reflect.DeepEqual(blB, blR) {
			t.Errorf("%s: backlinks divergem — Build=%v, Replace=%v", p, blB, blR)
		}
	}

	if !reflect.DeepEqual(doBuild.Tags("", 0), doReplace.Tags("", 0)) {
		t.Errorf("Tags divergem — Build=%v, Replace=%v", doBuild.Tags("", 0), doReplace.Tags("", 0))
	}
}

func TestConstrucoesDoIndiceConcordamCampoACampo(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "Civil/PONTO 03.md", "---\naliases: [P3]\ntags: [civil, prova]\n---\n# Ponto 3\n\nVer [[Penal/A]] e [[Sumiu]].\n")
	writeFile(t, root, "Penal/A.md", "# A\n\n![[diagrama.png]]\n\n#golang\n")
	writeFile(t, root, "Penal/B.md", "# B\n\nVolta para [[P3]].\n")
	writeFile(t, root, "Anexos/diagrama.png", "\x89PNG")

	compararConstrucoes(t, root)
}
