package service_test

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/search"
	"github.com/jonyd/gobsidian/internal/service"
	"github.com/jonyd/gobsidian/internal/vault"
)

// TestRNF03_RNF05_RNF06 mede os três RNFs de latência que a tabela de
// docs/OPERACAO.md trazia como "não medido" até o fechamento do M6:
//
//	RNF-03  note_read p95                      alvo 15 ms, degradado 50 ms
//	RNF-05  note_list com filtro de metadados   alvo 10 ms, degradado 30 ms
//	RNF-06  reindexação de arquivo único        alvo 20 ms, degradado 100 ms
//
// Mede na escala de 5.000 notas, que é onde os números significam alguma coisa:
// a 7 notas qualquer um deles sai abaixo da resolução do relógio, e um teto com
// 1000x de folga não pode falhar.
//
// Os tetos NÃO são cobrados sob -race, pela razão de sempre — ver
// raceflag_on_test.go. A medição é registrada nos dois modos.
func TestRNF03_RNF05_RNF06(t *testing.T) {
	vaultDir := getVault5000Path(t)

	v, err := vault.New(vaultDir)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	idx := index.New()
	if err := idx.Build(context.Background(), v); err != nil {
		t.Fatalf("idx.Build: %v", err)
	}
	if n := idx.NoteCount(); n < 5000 {
		t.Fatalf("NoteCount = %d, quer 5000; medir noutra escala responde outra pergunta", n)
	}
	inv := search.NewInverted()
	svc := service.New(v, idx, inv, nil, service.Options{})
	ctx := context.Background()

	caminhos := idx.NotePaths()

	// lote e quantas operacoes entram em cada amostra cronometrada.
	//
	// note_read e note_list custam centenas de MICROsegundos, e o relogio do
	// Windows anda de ~0,5 ms em ~0,5 ms: cronometrar uma operacao por vez dava
	// "mediana 0s", e um percentil sobre zeros nao guarda nada. A guarda de
	// mediana zerada acusou isso na primeira medicao — a correcao e medir um
	// lote e dividir, nao afrouxar a guarda.
	// alvo e o numero do RNF; teto e a linha de DEGRADADO que o proprio PRD
	// define e que o gate cobra. Os dois sao registrados sempre.
	//
	// A distincao existe por causa do RNF-06, que nao e atingido a 5.000 notas:
	// mediana de 20,3 ms contra alvo de 20 ms. Cobrar o alvo deixaria a suite
	// permanentemente vermelha; ignora-lo esconderia a lacuna. O log traz os
	// dois numeros, docs/OPERACAO.md registra "NAO ATINGIDO", e o gate guarda a
	// unica coisa que ainda pode piorar sem ninguem ver: a linha de degradado.
	medir := func(nome string, alvo, teto time.Duration, n, lote int, fn func(i int) error) {
		t.Helper()
		durs := make([]time.Duration, n)
		for i := range n {
			ini := time.Now()
			for k := range lote {
				if err := fn(i*lote + k); err != nil {
					t.Fatalf("%s: %v", nome, err)
				}
			}
			durs[i] = time.Since(ini) / time.Duration(lote)
		}
		sort.Slice(durs, func(a, b int) bool { return durs[a] < durs[b] })
		mediana := durs[n/2]
		p95 := durs[int(float64(n)*0.95)]
		situacao := "ATINGIDO"
		if p95 > alvo {
			situacao = "NAO ATINGIDO"
		}
		t.Logf("  %-42s mediana %-12v p95 %-12v alvo %-8v degradado %-8v %s",
			nome, mediana, p95, alvo, teto, situacao)

		// Mediana zerada significa que o relógio não andou, e um percentil sobre
		// zeros não guarda nada.
		if mediana == 0 {
			t.Errorf("%s: mediana = 0s, a medição não mediu nada", nome)
		}
		if !raceEnabled && p95 > teto {
			t.Errorf("%s: p95 = %v excede a linha de DEGRADADO de %v (alvo do RNF: %v)",
				nome, p95, teto, alvo)
		}
	}

	t.Logf("RNF-03, RNF-05 e RNF-06 em %d notas", idx.NoteCount())

	// RNF-03: note_read. Percorre caminhos diferentes a cada iteração para não
	// medir o cache de disco de um arquivo só.
	medir("RNF-03 note_read", 15*time.Millisecond, 50*time.Millisecond, 40, 50, func(i int) error {
		_, err := svc.ReadNote(ctx, service.ReadRequest{Path: string(caminhos[i%len(caminhos)])})
		return err
	})

	// RNF-05: note_list COM filtro de metadados — o alvo do RNF é esse, e listar
	// sem filtro mede outra coisa.
	medir("RNF-05 note_list (filtro de tag)", 10*time.Millisecond, 30*time.Millisecond, 40, 20, func(int) error {
		_, err := svc.ListNotes(ctx, service.ListRequest{
			Query: index.Query{Tags: []string{"golang"}, TagMode: "any", Recursive: true},
		})
		return err
	})

	// RNF-06: reindexação de arquivo único, que é o caminho que o watcher
	// percorre a cada evento.
	medir("RNF-06 index.Replace (arquivo unico)", 20*time.Millisecond, 100*time.Millisecond, 60, 1, func(i int) error {
		return idx.Replace(ctx, v, caminhos[i%len(caminhos)])
	})
}
