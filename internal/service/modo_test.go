package service

import (
	"context"
	"testing"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/vault"
)

// TestModoNuncaSaiVazio: um campo que as vezes some faz quem le acreditar que a
// informacao nao existe, quando ela so nao foi preenchida.
//
// A pergunta que este campo responde -- "quem esta servindo este cofre, e onde
// ele escreve?" -- e a que a investigacao de 2026-09-08 nao conseguiu fazer ao
// produto. Dois processos serviam o cofre Estudo e gravavam o MESMO
// inverted_cache.gob, e descobrir isso exigiu comparar milissegundos entre
// linhas de log duplicadas.
func TestModoNuncaSaiVazio(t *testing.T) {
	casos := []struct {
		declarado string
		esperado  string
	}{
		{ModoDaemon, ModoDaemon},
		{ModoEmProcesso, ModoEmProcesso},
		{"", ModoDesconhecido},
	}
	for _, c := range casos {
		s := &Service{opts: Options{Modo: c.declarado}}
		if got := s.modo(); got != c.esperado {
			t.Errorf("modo(%q) = %q, esperado %q", c.declarado, got, c.esperado)
		}
	}
}

// TestOsTresModosSaoDistintos: se duas constantes colidirem, `doctor` e
// vault_stats passam a dizer a mesma coisa para estados diferentes.
func TestOsTresModosSaoDistintos(t *testing.T) {
	vistos := map[string]bool{}
	for _, m := range []string{ModoDaemon, ModoEmProcesso, ModoDesconhecido} {
		if m == "" {
			t.Error("uma das constantes de modo esta vazia")
		}
		if vistos[m] {
			t.Errorf("constante de modo duplicada: %q", m)
		}
		vistos[m] = true
	}
}

// TestVaultStatsReportaModoECacheDir prova que os campos CHEGAM AO RETORNO, e
// nao apenas que a funcao que os calcula existe.
//
// A distincao nao e academica: a primeira versao deste arquivo testava so
// s.modo(), e a mutacao que trocava `Modo: s.modo()` por `Modo: ""` dentro de
// VaultStats passava. Um campo declarado que ninguem preenche e pior que um
// campo ausente, porque quem le acredita -- e docs/TOOLS.md ja diz isso sobre
// esta mesma struct.
func TestVaultStatsReportaModoECacheDir(t *testing.T) {
	root := t.TempDir()

	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	ix := index.New()
	if err := ix.Build(context.Background(), v); err != nil {
		t.Fatalf("index.Build: %v", err)
	}

	svc := New(v, ix, nil, nil, Options{Modo: ModoDaemon, CacheDir: `C:/cache/abc`})

	st, err := svc.VaultStats(context.Background(), StatsRequest{IncludeRuntime: true})
	if err != nil {
		t.Fatalf("VaultStats: %v", err)
	}
	if st.Runtime == nil {
		t.Fatal("include_runtime nao devolveu o bloco runtime")
	}
	if st.Runtime.Modo != ModoDaemon {
		t.Errorf("Modo = %q, esperado %q -- quem diagnostica nao sabe quem esta servindo", st.Runtime.Modo, ModoDaemon)
	}
	if st.Runtime.CacheDir != `C:/cache/abc` {
		t.Errorf("CacheDir = %q, esperado o diretorio declarado -- duas instancias com o mesmo cache_dir sao duas instancias disputando os mesmos arquivos", st.Runtime.CacheDir)
	}
}

// TestVaultStatsSemRuntimeNaoInventaModo: sem include_runtime o bloco inteiro
// nao vem, e isso e contrato -- o campo custa duas chamadas ao runtime e so
// interessa a quem esta diagnosticando.
func TestVaultStatsSemRuntimeNaoInventaModo(t *testing.T) {
	root := t.TempDir()
	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	ix := index.New()
	if err := ix.Build(context.Background(), v); err != nil {
		t.Fatalf("index.Build: %v", err)
	}
	svc := New(v, ix, nil, nil, Options{Modo: ModoDaemon})

	st, err := svc.VaultStats(context.Background(), StatsRequest{})
	if err != nil {
		t.Fatalf("VaultStats: %v", err)
	}
	if st.Runtime != nil {
		t.Errorf("runtime veio sem include_runtime: %+v", st.Runtime)
	}
}
