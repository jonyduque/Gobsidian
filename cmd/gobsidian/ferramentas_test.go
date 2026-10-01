package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/jonyduque/Gobsidian/internal/boot"
	"github.com/jonyduque/Gobsidian/internal/config"
	"github.com/jonyduque/Gobsidian/internal/console"
	"github.com/jonyduque/Gobsidian/internal/mcpsrv"
)

// cofreDeFerramentas e o cofre de fixture dos testes das tools: duas notas,
// uma tag aninhada, um link quebrado e uma ancora.
func cofreDeFerramentas(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	notas := map[string]string{
		"Civil/Dolo.md": "---\ntags: [penal, direito/civil]\n---\n# Dolo\n## Conceito\nVontade livre e consciente. Ver [[Culpa]] e [[Nada]].\n^bloco1\n",
		"Culpa.md":      "# Culpa\nNegligencia e imprudencia. [[Dolo#Conceito]]\n",
	}
	for rel, conteudo := range notas {
		full := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(conteudo), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// rodar executa a arvore inteira, como o binario, e devolve stdout, stderr e
// o erro.
func rodar(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	root := newRootCmd()
	var out, errOut bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errOut)
	root.SetIn(strings.NewReader(""))
	root.SetArgs(args)
	err := root.Execute()
	return out.String(), errOut.String(), err
}

// argumentosDeCofre sao --vault e --cache-dir de cada chamada.
func argumentosDeCofre(t *testing.T, cofre string) []string {
	return []string{"--vault", cofre, "--cache-dir", t.TempDir()}
}

// comandoDa acha o comando de uma tool na arvore.
func comandoDa(t *testing.T, root *cobra.Command, f ferramenta) *cobra.Command {
	t.Helper()
	caminho := strings.Fields(f.caminhoDoComando())
	c, _, err := root.Find(caminho)
	if err != nil || c == nil || c.Name() != f.nome {
		t.Fatalf("comando %q ausente da arvore (err=%v)", f.caminhoDoComando(), err)
	}
	return c
}

// TestCadaToolTemComando e o gate de cobertura do desenho: toda tool do
// servidor tem comando, todo comando da tabela e uma tool, e todo parametro
// do schema virou flag ou posicional. Tool nova sem comando, ou parametro novo
// sem flag, quebra aqui -- e a invariante que vale um gate.
func TestCadaToolTemComando(t *testing.T) {
	esquemas, err := mcpsrv.EsquemasDeEntrada()
	if err != nil {
		t.Fatal(err)
	}
	naTabela := map[string]ferramenta{}
	for _, f := range ferramentas {
		naTabela[f.tool] = f
	}
	noServidor := map[string]bool{}
	root := newRootCmd()
	for _, e := range esquemas {
		noServidor[e.Nome] = true
		f, ok := naTabela[e.Nome]
		if !ok {
			t.Errorf("a tool %s nao tem comando na tabela de ferramentas.go", e.Nome)
			continue
		}
		cmd := comandoDa(t, root, f)
		cobertos := map[string]bool{}
		for _, p := range f.posicionais {
			cobertos[p] = true
		}
		if f.plural != "" {
			cobertos[f.plural] = true
		}
		for _, p := range e.Parametros {
			if cobertos[p.Nome] {
				continue
			}
			if cmd.Flags().Lookup(nomeDaFlag(p.Nome)) == nil {
				t.Errorf("%s: o parametro %q (tipo %s) nao virou flag nem posicional", f.caminhoDoComando(), p.Nome, p.Tipo)
			}
		}
		for _, p := range f.posicionais {
			if !temParametro(e, p) {
				t.Errorf("%s: o posicional %q nao e parametro da tool", f.caminhoDoComando(), p)
			}
		}
	}
	for _, f := range ferramentas {
		if !noServidor[f.tool] {
			t.Errorf("a tabela cita %s, que o servidor nao registra", f.tool)
		}
	}
}

func temParametro(e mcpsrv.EsquemaDeTool, nome string) bool {
	for _, p := range e.Parametros {
		if p.Nome == nome {
			return true
		}
	}
	return false
}

// TestFlagsDeControleSoOndeValem: flag declarada e ignorada e pior que flag
// ausente (achado 5.8). --max-results so em search, que e a unica tool com
// teto; --read-only so nas de escrita.
func TestFlagsDeControleSoOndeValem(t *testing.T) {
	esquemas, _ := mcpsrv.EsquemasDeEntrada()
	escrita := map[string]bool{}
	for _, e := range esquemas {
		escrita[e.Nome] = e.Escrita
	}
	root := newRootCmd()
	for _, f := range ferramentas {
		cmd := comandoDa(t, root, f)
		if tem := cmd.Flags().Lookup("max-results") != nil; tem != (f.tool == "vault_search") {
			t.Errorf("%s: --max-results declarada = %v", f.caminhoDoComando(), tem)
		}
		if tem := cmd.Flags().Lookup("read-only") != nil; tem != escrita[f.tool] {
			t.Errorf("%s: --read-only declarada = %v, escrita = %v", f.caminhoDoComando(), tem, escrita[f.tool])
		}
	}
}

// TestCadaToolTemFormatador: tool sem formatador cai no JSON indentado, o que
// funciona -- e e exatamente por funcionar que o esquecimento passaria.
func TestCadaToolTemFormatador(t *testing.T) {
	esquemas, _ := mcpsrv.EsquemasDeEntrada()
	for _, e := range esquemas {
		if _, ok := formatadores[e.Nome]; !ok {
			t.Errorf("a tool %s nao tem formatador em ferramentas_texto.go", e.Nome)
		}
	}
}

// casoDeParidade e uma chamada: a linha de comando e a entrada que a tool deve
// receber por ela.
type casoDeParidade struct {
	tool    string
	args    []string
	entrada map[string]any
}

var casosDeParidade = []casoDeParidade{
	{"note_read", []string{"note", "read", "Civil/Dolo.md", "--heading", "Conceito"}, map[string]any{"path": "Civil/Dolo.md", "heading": "Conceito"}},
	{"note_list", []string{"note", "list", "--tags", "penal", "--limit", "5"}, map[string]any{"tags": []any{"penal"}, "limit": 5}},
	{"note_outline", []string{"note", "outline", "Civil/Dolo.md"}, map[string]any{"path": "Civil/Dolo.md"}},
	{"note_metadata", []string{"note", "metadata", "Culpa.md", "--include", "links,headings"}, map[string]any{"path": "Culpa.md", "include": []any{"links", "headings"}}},
	{"note_create", []string{"note", "create", "Nova.md", "--content", "# Nova\n", "--dry-run"}, map[string]any{"path": "Nova.md", "content": "# Nova\n", "dry_run": true}},
	{"note_append", []string{"note", "append", "Culpa.md", "--content", "mais", "--dry-run"}, map[string]any{"path": "Culpa.md", "content": "mais", "dry_run": true}},
	{"note_patch", []string{"note", "patch", "Civil/Dolo.md", "--heading", "Conceito", "--content", "Outro.", "--dry-run"}, map[string]any{"path": "Civil/Dolo.md", "heading": "Conceito", "content": "Outro.", "dry_run": true}},
	{"note_move", []string{"note", "move", "Culpa.md", "Civil/Culpa.md", "--dry-run"}, map[string]any{"from": "Culpa.md", "to": "Civil/Culpa.md", "dry_run": true}},
	{"note_delete", []string{"note", "delete", "Culpa.md", "--dry-run"}, map[string]any{"path": "Culpa.md", "dry_run": true}},
	{"vault_stats", []string{"stats"}, map[string]any{}},
	{"vault_search", []string{"search", "vontade"}, map[string]any{"query": "vontade"}},
	{"vault_broken_links", []string{"broken-links", "--state", "target_missing"}, map[string]any{"state": "target_missing"}},
	{"tag_list", []string{"tag", "list", "--hierarchical"}, map[string]any{"hierarchical": true}},
	{"link_graph", []string{"graph", "Culpa.md", "--depth", "2"}, map[string]any{"path": "Culpa.md", "depth": 2}},
}

// TestParidadeComATool: para cada tool, o comando com --json devolve o MESMO
// JSON que a chamada feita direto pelo transporte em memoria com a entrada
// esperada. O que isto prova e o mapeamento de posicionais e flags para a
// entrada da tool -- o resto e a propria tool nos dois lados.
func TestParidadeComATool(t *testing.T) {
	cobertas := map[string]bool{}
	for _, c := range casosDeParidade {
		cobertas[c.tool] = true
		t.Run(c.tool, func(t *testing.T) {
			cofre := cofreDeFerramentas(t)
			cache := t.TempDir()
			// Aquece o cache antes das duas chamadas: indice construido e
			// indice lido do cache diferem em vault_stats.generation (2 contra
			// 0, medido em 2026-09-27), e a primeira chamada construiria
			// enquanto a segunda leria o que ela gravou. Os dois lados partem
			// do mesmo estado.
			if _, errOut, err := rodar(t, "stats", "--vault", cofre, "--cache-dir", cache, "--json"); err != nil {
				t.Fatalf("aquecendo o cache: %v\n%s", err, errOut)
			}
			out, errOut, err := rodar(t, append(c.args, "--vault", cofre, "--cache-dir", cache, "--json")...)
			if err != nil {
				t.Fatalf("comando: %v\nstderr: %s", err, errOut)
			}

			cfg, err := config.Load(config.Flags{VaultPath: cofre, CacheDir: cache})
			if err != nil {
				t.Fatal(err)
			}
			log := slog.New(slog.NewTextHandler(io.Discard, nil))
			svc, fechar, err := boot.AbrirServicoDeCLI(context.Background(), cfg, log)
			if err != nil {
				t.Fatal(err)
			}
			defer fechar()
			direto, err := mcpsrv.ChamarLocal(context.Background(), svc, cfg, log, c.tool, c.entrada)
			if err != nil {
				t.Fatalf("chamada direta: %v", err)
			}

			var pelaCLI, pelaTool any
			if err := json.Unmarshal([]byte(out), &pelaCLI); err != nil {
				t.Fatalf("saida do comando nao e JSON: %v\n%s", err, out)
			}
			if err := json.Unmarshal(direto, &pelaTool); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(pelaCLI, pelaTool) {
				t.Errorf("CLI e tool divergem.\nCLI:  %s\ntool: %s", strings.TrimSpace(out), direto)
			}
		})
	}
	for _, f := range ferramentas {
		if !cobertas[f.tool] {
			t.Errorf("%s sem caso de paridade", f.tool)
		}
	}
}

// TestMontarEntrada: so a flag passada viaja, a flag vence --args, e varios
// posicionais de note read viram paths.
func TestMontarEntrada(t *testing.T) {
	root := newRootCmd()
	var leitura ferramenta
	for _, f := range ferramentas {
		if f.tool == "note_read" {
			leitura = f
		}
	}
	cmd := comandoDa(t, root, leitura)
	if err := cmd.Flags().Parse([]string{"--heading", "Conceito", "--args", `{"heading":"Outro","max_bytes":10}`}); err != nil {
		t.Fatal(err)
	}

	var params []*parametroDeFlag
	esquemas, _ := mcpsrv.EsquemasDeEntrada()
	for _, e := range esquemas {
		if e.Nome == "note_read" {
			// Registra num comando novo so para ter os leitores das flags;
			// os valores vem de cmd, que foi o que recebeu o Parse.
			params = registrarParametros(&cobra.Command{}, e, map[string]bool{"path": true, "paths": true})
		}
	}
	for _, p := range params {
		p := p
		fl := cmd.Flags().Lookup(p.flag)
		p.ler = func() (any, error) { return fl.Value.String(), nil }
	}

	entrada, err := montarEntrada(cmd, leitura, `{"heading":"Outro","max_bytes":10}`, []string{"a.md", "b.md"}, params)
	if err != nil {
		t.Fatal(err)
	}
	if entrada["heading"] != "Conceito" {
		t.Errorf("heading = %v: a flag passada tem de vencer --args", entrada["heading"])
	}
	if _, ok := entrada["include_frontmatter"]; ok {
		t.Errorf("include_frontmatter viajou sem ter sido passada: %v", entrada)
	}
	if entrada["max_bytes"] != float64(10) {
		t.Errorf("max_bytes = %v: o que so --args traz tem de chegar", entrada["max_bytes"])
	}
	if paths, ok := entrada["paths"].([]any); !ok || len(paths) != 2 {
		t.Errorf("paths = %v: dois posicionais viram paths", entrada["paths"])
	}
	if _, ok := entrada["path"]; ok {
		t.Errorf("path e paths juntos: %v", entrada)
	}
}

// TestConteudoPeloStdin: --content - le o stdin.
func TestConteudoPeloStdin(t *testing.T) {
	cofre := cofreDeFerramentas(t)
	root := newRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetIn(strings.NewReader("# Veio do stdin\n"))
	root.SetArgs(append([]string{"note", "create", "Nova.md", "--content", "-", "--json"}, argumentosDeCofre(t, cofre)...))
	if err := root.Execute(); err != nil {
		t.Fatalf("%v\n%s", err, out.String())
	}
	b, err := os.ReadFile(filepath.Join(cofre, "Nova.md"))
	if err != nil {
		t.Fatalf("a nota nao foi criada: %v\n%s", err, out.String())
	}
	if !strings.Contains(string(b), "# Veio do stdin") {
		t.Errorf("conteudo = %q", b)
	}
}

// TestGobsidianVaultValeNaCLIENaoNoServe: as duas metades do desenho.
func TestGobsidianVaultValeNaCLIENaoNoServe(t *testing.T) {
	cofre := cofreDeFerramentas(t)
	t.Setenv(varDoCofrePadrao, cofre)

	out, errOut, err := rodar(t, "note", "list", "--cache-dir", t.TempDir(), "--json")
	if err != nil {
		t.Fatalf("CLI com %s e sem --vault falhou: %v\n%s", varDoCofrePadrao, err, errOut)
	}
	if !strings.Contains(out, "Culpa.md") {
		t.Errorf("a CLI nao abriu o cofre da variavel:\n%s", out)
	}

	// A metade do servidor olha a flag e NAO executa o comando: com a regra
	// quebrada, executar serve serviria o cofre de verdade -- a ponte sobe um
	// daemon com o executavel do teste, que roda os testes de novo. Medido em
	// 2026-10-01 sob mutacao: 18 processos gobsidian.test em 10 s.
	for _, c := range []*cobra.Command{newServeCmd(), newDaemonCmd()} {
		fl := c.Flags().Lookup("vault")
		if fl == nil {
			t.Fatalf("%s sem --vault", c.Name())
		}
		if fl.DefValue != "" || fl.Value.String() != "" {
			t.Errorf("%s: --vault vale %q com %s definida; um host serviria o cofre da variavel em silencio", c.Name(), fl.Value.String(), varDoCofrePadrao)
		}
	}
}

// TestCodigosDeSaida: uma linha por situacao da tabela do desenho.
func TestCodigosDeSaida(t *testing.T) {
	cofre := cofreDeFerramentas(t)
	casos := []struct {
		nome   string
		args   []string
		codigo int
	}{
		{"sucesso", []string{"stats", "--json"}, 0},
		{"erro da tool", []string{"note", "metadata", "nao-existe.md", "--json"}, saidaErro},
		{"flag desconhecida", []string{"stats", "--nao-existe"}, saidaUso},
		{"json e texto juntos", []string{"stats", "--json", "--texto"}, saidaUso},
		{"args invalido", []string{"stats", "--args", "{nao e json"}, saidaUso},
		// note read aceita N posicionais e passa por outro ramo da validacao;
		// outline e move sao os de contagem fixa.
		{"posicional faltando (varios)", []string{"note", "read"}, saidaUso},
		{"posicional faltando", []string{"note", "outline"}, saidaUso},
		{"posicional sobrando", []string{"note", "move", "a.md", "b.md", "c.md"}, saidaUso},
		{"escrita em somente leitura", []string{"note", "delete", "Culpa.md", "--read-only"}, saidaUso},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			_, _, err := rodar(t, append(c.args, argumentosDeCofre(t, cofre)...)...)
			codigo := 0
			if err != nil {
				codigo, _ = codigoDeSaida(err)
			}
			if codigo != c.codigo {
				t.Errorf("codigo = %d, quer %d (err: %v)", codigo, c.codigo, err)
			}
		})
	}

	t.Run("cofre que nao resolve", func(t *testing.T) {
		_, _, err := rodar(t, "stats", "--vault", "cofre-que-nao-existe-em-lugar-nenhum", "--cache-dir", t.TempDir())
		if codigo, _ := codigoDeSaida(err); err == nil || codigo != saidaUso {
			t.Errorf("err = %v, codigo = %d; quer uso (%d)", err, codigo, saidaUso)
		}
	})
}

// TestSegundaChamadaLeOCache: o que index_origem_test.go e search_cache_test.go
// provavam sobre os comandos que sairam. A primeira chamada constroi e grava
// os dois caches; a segunda le os dois. Um comando que abrisse o servico sem
// cache-dir, ou que nao gravasse, pagaria a varredura a cada chamada -- e a
// CLI existe para ser chamada muitas vezes.
func TestSegundaChamadaLeOCache(t *testing.T) {
	cofre := cofreDeFerramentas(t)
	cache := t.TempDir()
	roda := func() string {
		out, errOut, err := rodar(t, "search", "vontade", "--vault", cofre, "--cache-dir", cache, "--log-level", "info", "--json")
		if err != nil {
			t.Fatalf("search: %v\n%s", err, errOut)
		}
		if !strings.Contains(out, `"Civil/Dolo.md"`) {
			t.Fatalf("stdout sem o resultado:\n%s", out)
		}
		return errOut
	}
	primeira := roda()
	for _, q := range []string{"origem=build", "origem=construcao"} {
		if !strings.Contains(primeira, q) {
			t.Errorf("primeira chamada sem %q:\n%s", q, primeira)
		}
	}
	segunda := roda()
	if strings.Count(segunda, "origem=cache") != 2 {
		t.Errorf("segunda chamada devia ler os dois caches (metadados e busca):\n%s", segunda)
	}
}

// TestFerramentaForaDoTerminalSaiEmJSON: sem --json nem --texto, quem decide e
// o destino. Um buffer nao e terminal, entao o script recebe JSON numa linha
// -- sem precisar saber da flag.
func TestFerramentaForaDoTerminalSaiEmJSON(t *testing.T) {
	cofre := cofreDeFerramentas(t)
	out, errOut, err := rodar(t, append([]string{"note", "list"}, argumentosDeCofre(t, cofre)...)...)
	if err != nil {
		t.Fatalf("%v\n%s", err, errOut)
	}
	var v map[string]any
	if err := json.Unmarshal([]byte(out), &v); err != nil {
		t.Fatalf("fora do terminal a saida devia ser JSON: %v\n%s", err, out)
	}
	if n := strings.Count(strings.TrimRight(out, "\n"), "\n"); n != 0 {
		t.Errorf("JSON em %d linhas; quer uma so:\n%s", n+1, out)
	}
}

// TestSearchRespeitaMaxResults: --max-results e declarada so em search
// (TestFlagsDeControleSoOndeValem); aqui, que ela e aplicada.
func TestSearchRespeitaMaxResults(t *testing.T) {
	cofre := t.TempDir()
	for i := range 5 {
		if err := os.WriteFile(filepath.Join(cofre, "n"+string(rune('0'+i))+".md"), []byte("# Nota\n\npalavra unica aqui\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	out, errOut, err := rodar(t, append([]string{"search", "palavra", "--max-results", "2", "--json"}, argumentosDeCofre(t, cofre)...)...)
	if err != nil {
		t.Fatalf("%v\n%s", err, errOut)
	}
	var r struct {
		Results []json.RawMessage `json:"results"`
	}
	if err := json.Unmarshal([]byte(out), &r); err != nil {
		t.Fatalf("saida nao e JSON: %v\n%s", err, out)
	}
	if len(r.Results) != 2 {
		t.Errorf("results = %d com --max-results 2: a flag e lida e descartada", len(r.Results))
	}
}

// TestErroDaToolEmJSONVaiParaOStdout: o script le stdout; o erro precisa
// estar la, com o codigo, e nada antes nem depois.
func TestErroDaToolEmJSONVaiParaOStdout(t *testing.T) {
	cofre := cofreDeFerramentas(t)
	out, _, err := rodar(t, append([]string{"note", "metadata", "nao-existe.md", "--json"}, argumentosDeCofre(t, cofre)...)...)
	if err == nil {
		t.Fatal("esperava erro")
	}
	var r struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if jerr := json.Unmarshal([]byte(out), &r); jerr != nil || r.Error.Code != "NOTE_NOT_FOUND" {
		t.Errorf("stdout = %q (%v); quer {\"error\":{\"code\":\"NOTE_NOT_FOUND\"...}}", out, jerr)
	}
}

// TestFormatadores: um caso por tool, com o modo de saida fixado -- teste que
// afirma desenho fixa o modo (licao de 2026-09-16).
func TestFormatadores(t *testing.T) {
	t.Setenv(console.VarDeEstilo, "0")
	casos := map[string]struct {
		args []string
		quer []string
	}{
		"note_read":          {[]string{"note", "read", "Civil/Dolo.md", "--heading", "Conceito"}, []string{"## Conceito", "Vontade livre"}},
		"note_list":          {[]string{"note", "list"}, []string{"2 de 2 notas.", "Civil/Dolo.md", "#penal"}},
		"note_outline":       {[]string{"note", "outline", "Civil/Dolo.md"}, []string{"Dolo", "Conceito"}},
		"note_metadata":      {[]string{"note", "metadata", "Civil/Dolo.md"}, []string{"Civil/Dolo.md", "Dolo", "penal"}},
		"note_create":        {[]string{"note", "create", "Nova.md", "--content", "x", "--dry-run"}, []string{"Simulacao"}},
		"note_append":        {[]string{"note", "append", "Culpa.md", "--content", "x"}, []string{"Conteudo acrescentado: Culpa.md."}},
		"note_patch":         {[]string{"note", "patch", "Civil/Dolo.md", "--heading", "Conceito", "--content", "Outro."}, []string{"Trecho substituido: Civil/Dolo.md."}},
		"note_move":          {[]string{"note", "move", "Culpa.md", "Civil/Culpa.md"}, []string{"Nota movida: Culpa.md -> Civil/Culpa.md."}},
		"note_delete":        {[]string{"note", "delete", "Culpa.md", "--to-trash=false"}, []string{"Nota excluida: Culpa.md."}},
		"vault_stats":        {[]string{"stats"}, []string{"Notas", "2", "Links quebrados"}},
		"vault_search":       {[]string{"search", "vontade"}, []string{"1 de 1 notas.", "Civil/Dolo.md"}},
		"vault_broken_links": {[]string{"broken-links"}, []string{"1 de 1 links quebrados.", "Civil/Dolo.md -> Nada"}},
		"tag_list":           {[]string{"tag", "list"}, []string{"#penal", "#direito/civil"}},
		"link_graph":         {[]string{"graph", "Culpa.md"}, []string{"a partir de Culpa.md", "Civil/Dolo.md"}},
	}
	for _, f := range ferramentas {
		c, ok := casos[f.tool]
		if !ok {
			t.Errorf("%s sem caso de formatador", f.tool)
			continue
		}
		t.Run(f.tool, func(t *testing.T) {
			cofre := cofreDeFerramentas(t)
			out, errOut, err := rodar(t, append(c.args, append(argumentosDeCofre(t, cofre), "--texto")...)...)
			if err != nil {
				t.Fatalf("%v\n%s", err, errOut)
			}
			for _, q := range c.quer {
				if !strings.Contains(out, q) {
					t.Errorf("faltou %q:\n%s", q, out)
				}
			}
			if strings.HasPrefix(strings.TrimSpace(out), "{") {
				t.Errorf("--texto saiu em JSON:\n%s", out)
			}
		})
	}
}
