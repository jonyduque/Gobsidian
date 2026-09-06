package service_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/jonyd/gobsidian/internal/service"
)

// O golden de tag_list existe para uma refatoracao especifica: TagNode.Children
// deixou de ser []any e passou a ser []TagNode. O tipo do campo e invisivel para
// quem consome a tool — o que ele consome e o JSON —, e "o JSON nao muda" e
// exatamente o tipo de afirmacao que se faz sem conferir. Aqui ela e conferida
// byte a byte.
//
// O cofre e escrito pelo proprio teste, e nao lido de testdata/: o que precisa
// ser deterministico e o JSON, e um cofre montado a partir de literais nao pode
// divergir do golden por um arquivo que alguem editou sem reparar.

// cofreDeTagsHierarquicas escreve um cofre com tags em tres niveis, prefixos
// compartilhados e um empate de contagem.
//
// O empate (proj/alpha e docs, ambos em 3 notas) nao e decorativo: com
// sort=count ele e o unico caso que exercita o desempate por nome, e um golden
// sem empate passaria mesmo que o desempate sumisse.
func cofreDeTagsHierarquicas(t *testing.T) string {
	t.Helper()

	notas := map[string]string{
		"nota-1.md": "# Um\n\nCorpo da primeira nota. #proj/alpha/um #proj/beta #docs\n",
		"nota-2.md": "# Dois\n\nCorpo da segunda nota. #proj/alpha/um #proj/alpha/dois #docs\n",
		"nota-3.md": "# Tres\n\nCorpo da terceira nota. #proj/alpha #docs/api\n",
		"nota-4.md": "# Quatro\n\nCorpo da quarta nota. #proj/beta #zeta\n",
	}

	root := t.TempDir()
	for nome, corpo := range notas {
		if err := os.WriteFile(filepath.Join(root, nome), []byte(corpo), 0o644); err != nil {
			t.Fatalf("escrevendo %s: %v", nome, err)
		}
	}
	return root
}

// variantesDeTagList sao as consultas que o golden cobre. A ordem e fixa porque
// ela e a ordem do JSON gravado.
var variantesDeTagList = []struct {
	Nome string             `json:"nome"`
	Req  service.TagRequest `json:"requisicao"`
}{
	{"hierarquico_nome", service.TagRequest{Sort: "name", Hierarchical: true}},
	{"hierarquico_contagem", service.TagRequest{Sort: "count", Hierarchical: true}},
	{"hierarquico_prefixo_proj", service.TagRequest{Sort: "name", Hierarchical: true, Prefix: "proj"}},
	{"hierarquico_min_count_2", service.TagRequest{Sort: "count", Hierarchical: true, MinCount: 2}},
	{"plano_contagem", service.TagRequest{Sort: "count"}},
}

func TestTagListGolden(t *testing.T) {
	svc := servicoDoCofre(t, cofreDeTagsHierarquicas(t))

	type entrada struct {
		Nome      string             `json:"nome"`
		Requisica service.TagRequest `json:"requisicao"`
		Resultado service.TagResult  `json:"resultado"`
	}

	saida := make([]entrada, 0, len(variantesDeTagList))
	for _, v := range variantesDeTagList {
		res, err := svc.TagList(context.Background(), v.Req)
		if err != nil {
			t.Fatalf("TagList(%s): %v", v.Nome, err)
		}
		if len(res.Tags) == 0 {
			t.Fatalf("TagList(%s) devolveu zero tags; um golden vazio nao cobre nada", v.Nome)
		}
		saida = append(saida, entrada{Nome: v.Nome, Requisica: v.Req, Resultado: res})
	}

	got, err := json.MarshalIndent(saida, "", "  ")
	if err != nil {
		t.Fatalf("serializando: %v", err)
	}
	got = append(got, '\n')

	goldenPath := filepath.Join("..", "..", "testdata", "tag_list_hierarquico.json")

	if *atualizaGolden {
		if err := os.WriteFile(goldenPath, got, 0o644); err != nil {
			t.Fatalf("gravando golden: %v", err)
		}
		return
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("lendo golden (rode com -update para criar): %v", err)
	}
	if string(got) != string(want) {
		t.Errorf("o JSON de tag_list mudou\n--- esperado ---\n%s\n--- obtido ---\n%s", want, got)
	}
}
