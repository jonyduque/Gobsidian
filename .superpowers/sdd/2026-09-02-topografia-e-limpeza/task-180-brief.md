### Task 180: b — uma chave de tag (caixa, NFC, sem `#`), hierárquica em `note_list`, `vault_search` e `tag_list`

**Files:**
- Create: `internal/service/bench_tags_test.go` (`BenchmarkSearchFiltroTags`) — **commit 1, só teste**
- Modify: `internal/index/chave.go` (`ChaveDeTag`), `internal/index/index.go:179-180`, `internal/index/update.go:216-226,:556` (chave de `ix.tags`)
- Modify: `internal/index/query.go:173-190` (`Tags`), `:226-290` (`coletarLocked` passo 1 → `candidatosPorTagLocked`; `PathsComTags` exportado)
- Modify: `internal/service/search.go:213,:238,:441-453` (`porTag` calculado uma vez; `matchesSearchFilters` faz `BinarySearch`)
- Modify: `internal/service/graph.go` (`tagListHierarchical` da Task 169: prefixo via `ChaveDeTag`; chaves já dobradas)
- Modify: `internal/index/persist_test.go` (`TestIndiceDeMetadadosRecarregadoEIdentico` compara `Tags()`)
- Create: `internal/index/tag_chave_test.go`, `internal/service/tags_contrato_test.go`
- Modify: `testdata/tag_list_hierarquico.json` (golden da Task 169, regenerado), `docs/TOOLS.md:50,:201-202` e seção de `tag_list`, `docs/ESTADO.md`

**Interfaces:**
- Consumes: `text.ParaNFC` (o `index` já importa `text`); `aliasKey` como modelo (`chave.go:50`); `TagNode.Children []TagNode` e `tagListHierarchical` (Task 169); `porFrontmatter`/`casamFrontmatter` como modelo do filtro resolvido uma vez (`search.go:213,:488`).
- Produces:

```go
// ChaveDeTag e a UNICA conta da chave de tag: sem '#', NFC, minuscula.
// Exportada porque service compara tags e nao importa text (folha do grafo:
// service -> index, nao service -> text).
func ChaveDeTag(tag string) string

// PathsComTags devolve, ordenados, os caminhos das notas que casam as tags
// em mode ("all" | "any"), com a regra hierarquica: a tag pedida casa a si
// mesma e qualquer subtag ("projeto" casa "projeto/x"). Nil para len(tags)==0.
func (ix *Index) PathsComTags(tags []string, mode string) []vault.CanonicalPath
```

Regra do contrato (para `TOOLS.md`): em `note_list.tags`, `vault_search.tags` e `tag_list.prefix`, a tag pedida casa a si mesma e suas subtags; `#` inicial é opcional; comparação insensível a caixa e a forma Unicode (NFC). `tag_list` devolve a forma **dobrada** (`#Ação` e `#ação` viram uma entrada `ação` com a soma das contagens). `note_metadata.tags` continua devolvendo a grafia original da nota.

- [ ] **Step 1: Commit 1 — o bench que vai ser comparado**

`internal/service/bench_tags_test.go`:

```go
package service_test

import (
	"testing"

	"github.com/jonyd/gobsidian/internal/service"
)

// BenchmarkSearchFiltroTags mede a busca com filtro de tag: hoje o filtro
// baixa a caixa de cada tag de cada resultado por consulta; depois da
// Task 180 resolve o conjunto uma vez e faz busca binaria por resultado.
func BenchmarkSearchFiltroTags(b *testing.B) {
	svc := benchServicoDeCache(b)
	benchBusca(b, svc, service.SearchOptions{
		Query: "nota",
		Limit: 200,
		Tags:  []string{"golang"},
	}, 1)
}
```

O último argumento de `benchBusca` é o mínimo de resultados esperado (ver `BenchmarkSearchFiltroFrontmatter`, `bench_cache_test.go:100`); se a assinatura real for outra, seguir a real. Run: `go test ./internal/service/ -run xxx -bench SearchFiltroTags -benchtime 3x` — Expected: roda.

```bash
git add internal/service/bench_tags_test.go
git commit -m "test(service): benchmark vault_search with a tags filter"
```

Depois do commit: `go test -c -o %LOCALAPPDATA%\gobsidian-bench\2026-09-02\antes180_service.test.exe ./internal/service/` e `go test -c -o ...\antes180_index.test.exe ./internal/index/` — são os binários "antes" desta Task (os da Baseline não têm este bench).

- [ ] **Step 2: Testes do índice — falham por compilação e por comportamento**

`internal/index/tag_chave_test.go` (pacote `index`):

```go
package index

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/jonyd/gobsidian/internal/vault"
)

func TestChaveDeTagDobraCaixaHashENFC(t *testing.T) {
	// "Ação" em NFD: A + c + cedilha combinante + a + til combinante + o
	nfd := "Ação"
	casos := map[string]string{
		"#Projeto":    "projeto",
		"Projeto/Sub": "projeto/sub",
		"#" + nfd:     "ação",
		"ação":        "ação",
	}
	for in, quer := range casos {
		if got := ChaveDeTag(in); got != quer {
			t.Errorf("ChaveDeTag(%q) = %q, quero %q", in, got, quer)
		}
	}
}

func TestListPorTagCasaSubtagENFD(t *testing.T) {
	root := t.TempDir()
	escreve := func(nome, corpo string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, nome), []byte(corpo), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	escreve("a.md", "# A\n\n#Projeto/Alpha\n")
	escreve("b.md", "# B\n\n#projeto\n")
	escreve("c.md", "# C\n\n#outra\n")
	escreve("d.md", "# D\n\n#Ação\n") // NFD
	v, err := vault.New(root)
	if err != nil {
		t.Fatal(err)
	}
	ix := New()
	if err := ix.Build(context.Background(), v); err != nil {
		t.Fatal(err)
	}
	got := ix.PathsComTags([]string{"#PROJETO"}, "all")
	if quer := []vault.CanonicalPath{"a.md", "b.md"}; !slices.Equal(got, quer) {
		t.Fatalf("PathsComTags(#PROJETO) = %v, quero %v", got, quer)
	}
	got = ix.PathsComTags([]string{"ação"}, "all") // pedido em NFC, nota em NFD
	if quer := []vault.CanonicalPath{"d.md"}; !slices.Equal(got, quer) {
		t.Fatalf("PathsComTags(ação) = %v, quero %v", got, quer)
	}
	got = ix.PathsComTags([]string{"projeto/alpha", "outra"}, "any")
	if quer := []vault.CanonicalPath{"a.md", "c.md"}; !slices.Equal(got, quer) {
		t.Fatalf("PathsComTags(any) = %v, quero %v", got, quer)
	}
	if got := ix.PathsComTags([]string{"projeto/alpha", "outra"}, "all"); len(got) != 0 {
		t.Fatalf("PathsComTags(all, disjuntas) = %v, quero vazio", got)
	}
	tags := ix.Tags("proj", 0)
	if len(tags) != 2 || tags[0].Tag != "projeto" && tags[1].Tag != "projeto" {
		t.Fatalf("Tags(proj) = %v, quero projeto e projeto/alpha dobradas", tags)
	}
}
```

Run: `go test ./internal/index/ -run 'TestChaveDeTag|TestListPorTag'` — Expected: FAIL de compilação (`ChaveDeTag`, `PathsComTags` não existem).

- [ ] **Step 3: A chave, os três pontos de escrita, e `Tags`**

`internal/index/chave.go`, junto de `aliasKey`:

```go
// ChaveDeTag e a chave de ix.tags e a forma que tag_list devolve. Tres pontos
// escreviam a chave crua (boot, remocao, rename) e tres leitores baixavam a
// caixa cada um do seu jeito: Tags so ToLower no prefixo, coletarLocked ToLower
// nos dois lados a cada comparacao, service TrimPrefix('#') + ToLower. Nenhum
// deles normalizava Unicode, e "#Ação" digitado num Mac (NFD) nao casava o
// mesmo "#Ação" digitado no Windows (NFC). Uma conta, aqui.
func ChaveDeTag(tag string) string {
	return strings.ToLower(text.ParaNFC(strings.TrimPrefix(tag, "#")))
}
```

`index.go:179-180`: `k := ChaveDeTag(t); ix.tags[k] = append(ix.tags[k], n.Path)`. `update.go:216-226`: `paths := ix.tags[ChaveDeTag(tag)]` e `delete(ix.tags, ChaveDeTag(tag))` / `ix.tags[ChaveDeTag(tag)] = filtered`. `update.go:556`: `paths := ix.tags[ChaveDeTag(tag)]`. `Note.Tags` continua com a grafia original — só a chave do mapa dobra. Como o mesmo caminho pode entrar duas vezes na lista quando a nota tem `#Ação` e `#ação` (o parser mantém as duas grafias), `publishNoteLocked` deduplica: `if len(ix.tags[k]) == 0 || ix.tags[k][len(ix.tags[k])-1] != n.Path`.

`Tags` (`query.go:173-190`): `prefix = ChaveDeTag(prefix)` e `strings.HasPrefix(t, prefix)` direto (a chave já está dobrada); `TagCount{Tag: t, …}` devolve a chave.

- [ ] **Step 4: `PathsComTags` — a conta única do casamento**

Extrair o passo 1 de `coletarLocked` (`query.go:232-284`) para:

```go
// candidatosPorTagLocked e o passo 1 de coletarLocked e o corpo de
// PathsComTags. Exige ix.mu ja travado para leitura.
func (ix *Index) candidatosPorTagLocked(tags []string, mode string) []vault.CanonicalPath {
	casam := func(pedida string) []vault.CanonicalPath {
		tk := ChaveDeTag(pedida)
		var m []vault.CanonicalPath
		for k, v := range ix.tags {
			if k == tk || strings.HasPrefix(k, tk+"/") {
				m = append(m, v...)
			}
		}
		slices.Sort(m)
		return slices.Compact(m)
	}
	if strings.ToLower(mode) == "any" {
		var todos []vault.CanonicalPath
		for _, t := range tags {
			todos = append(todos, casam(t)...)
		}
		slices.Sort(todos)
		return slices.Compact(todos)
	}
	var cand []vault.CanonicalPath
	for i, t := range tags {
		m := casam(t)
		if i == 0 {
			cand = m
			continue
		}
		cand = slices.DeleteFunc(cand, func(c vault.CanonicalPath) bool {
			_, ok := slices.BinarySearch(m, c)
			return !ok
		})
		if len(cand) == 0 {
			break
		}
	}
	return cand
}

func (ix *Index) PathsComTags(tags []string, mode string) []vault.CanonicalPath {
	if len(tags) == 0 {
		return nil
	}
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	return ix.candidatosPorTagLocked(tags, mode)
}
```

Em `coletarLocked`, o passo 1 vira `if len(q.Tags) > 0 { candidates = ix.candidatosPorTagLocked(q.Tags, q.TagMode) } else { … }` — o `tagMode == ""` → `"all"` fica coberto pelo `!= "any"`. A linha `if k == tk || strings.HasPrefix(k, tk+"/") {` fica **exatamente** assim: âncora de mutação.

Run: `go test -race ./internal/index/` — Expected: PASS, inclusive os do Step 2.

- [ ] **Step 5: `service` — `vault_search` e `tag_list`**

`internal/service/search.go`: junto de `porFrontmatter := s.casamFrontmatter(opts)` (`:213`), `porTag := s.index.PathsComTags(opts.Tags, "all")`; `matchesSearchFilters(note, opts, porFrontmatter, porTag)`; dentro (`:441-453`), o bloco de tags vira:

```go
	// porTag e nil quando nao ha filtro de tag; com filtro, e o conjunto
	// ordenado que o indice resolveu UMA vez (index.PathsComTags) — a mesma
	// conta que note_list usa, hierarquia e dobra de caixa incluidas.
	if len(opts.Tags) > 0 {
		if _, ok := slices.BinarySearch(porTag, note.Path); !ok {
			return false
		}
	}
```

`internal/service/graph.go`, `tagListHierarchical` (forma da Task 169): o filtro de prefixo usa `index.ChaveDeTag(req.Prefix)`; as chaves de `s.index.Tags("", 0)` já vêm dobradas, então qualquer `strings.ToLower` restante ali sai. Testes de contrato em `internal/service/tags_contrato_test.go` (pacote `service`, usa `newTestService`):

```go
package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func cofreComTags(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	escreve := func(nome, corpo string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, nome), []byte(corpo), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	escreve("a.md", "# A\n\nnota comum #Projeto/Alpha\n")
	escreve("b.md", "# B\n\nnota comum #projeto\n")
	escreve("c.md", "# C\n\nnota comum #outra\n")
	escreve("d.md", "# D\n\nnota comum #Ação #ação\n") // NFD e NFC
	return root
}

func TestVaultSearchTagsCasaSubtag(t *testing.T) {
	svc := newTestService(t, cofreComTags(t))
	res, err := svc.Search(context.Background(), SearchOptions{Query: "comum", Tags: []string{"#PROJETO"}, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Results) != 2 {
		t.Fatalf("tags=[#PROJETO]: %d resultados, quero 2 (projeto e projeto/alpha): %+v", len(res.Results), res.Results)
	}
}

func TestVaultSearchTagsNFD(t *testing.T) {
	svc := newTestService(t, cofreComTags(t))
	res, err := svc.Search(context.Background(), SearchOptions{Query: "comum", Tags: []string{"ação"}, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Results) != 1 || res.Results[0].Path != "d.md" {
		t.Fatalf("tags=[ação]: %+v, quero so d.md", res.Results)
	}
}

func TestTagListDevolveFormaDobrada(t *testing.T) {
	svc := newTestService(t, cofreComTags(t))
	res, err := svc.TagList(context.Background(), TagListRequest{Prefix: "aç"})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Tags) != 1 || res.Tags[0].Tag != "ação" || res.Tags[0].Count != 1 {
		t.Fatalf("tag_list(aç) = %+v, quero uma entrada ação com count 1 (mesma nota, duas grafias)", res.Tags)
	}
}
```

Os nomes `TagListRequest`, `res.Tags`, `.Tag`, `.Count` são os reais de `graph.go` (conferir; a Task 169 pode ter mudado o tipo do item). `Count` é 1 porque as duas grafias estão na **mesma** nota e a lista de caminhos deduplica; se quiser 2, seriam duas notas.

Run: `go test -race ./internal/service/ ./internal/mcpsrv/` — Expected: PASS nos novos; `TestTagList_Hierarquico` (Task 169) **falha** no golden — é o Step 6.

- [ ] **Step 6: Golden da Task 169 e o `Tags()` no reload do cache**

Regenerar `testdata/tag_list_hierarquico.json` pelo mecanismo da Task 169 (`-update` ou equivalente que ela criou). No relatório, o `git diff` do golden com a explicação linha a linha: o que mudou é caixa/`#`/NFC das chaves e eventuais fusões de entradas — nenhuma contagem pode **cair** sem uma fusão que a explique.

`internal/index/persist_test.go`, `TestIndiceDeMetadadosRecarregadoEIdentico`: acrescentar `if !slices.Equal(recarregado.Tags("", 0), original.Tags("", 0)) { t.Fatalf(...) }` (os nomes das duas variáveis são os do teste). O cache guarda `Note.Tags` cru e o reload passa por `publishNoteLocked`, então a chave dobrada é reconstruída — e o formato do cache **não muda**. Se o teste falhar, a dobra não está no ponto único de publicação.

Run: `go test -race ./internal/index/ ./internal/service/ ./internal/mcpsrv/` — Expected: PASS.

- [ ] **Step 7: Provas de mutação**

```powershell
pwsh -File scripts/mutate.ps1 -Path internal/index/chave.go -Anchor 'strings.TrimPrefix(tag, "#")' -Replacement 'tag' -Test TestChaveDeTagDobraCaixaHashENFC -Package ./internal/index/
pwsh -File scripts/mutate.ps1 -Path internal/index/query.go -Anchor 'if k == tk || strings.HasPrefix(k, tk+"/") {' -Replacement 'if k == tk {' -Test TestListPorTagCasaSubtagENFD -Package ./internal/index/
```

Expected: exit 0 nos dois. Colar.

- [ ] **Step 8: `benchstat`**

`antes180_index` × depois: `TagsSemPrefixo`, `ListPorTag`. `antes180_service` × depois: `NoteListPorTag`, `TagListPlano`, `TagListHierarquico`, `SearchFiltroFrontmatter`, `SearchFiltroTags`. 7 intercalados, `-benchmem`. Expected: `SearchFiltroTags` e `ListPorTag` **melhoram** ou `~` (o `ToLower` por comparação sumiu); os demais `~`. Piora com `p < 0.05` em qualquer um: investigar antes de commitar — o suspeito é a deduplicação em `publishNoteLocked`. Colar; publicar em `docs/ESTADO.md`.

- [ ] **Step 9: Documentação e commit 2**

`docs/TOOLS.md:50` (`note_list.tags`), `:201-202` (`vault_search.tags`) e a seção de `tag_list`: a regra do contrato (bloco **Interfaces** acima), palavra por palavra nas três. `docs/ESTADO.md`: nota "chave de tag única desde 2026-09 (Task 180); formato de cache inalterado" e o `benchstat`.

Run: `pwsh -File scripts/verify.ps1` — Expected: verde.

```bash
git add internal/index/chave.go internal/index/index.go internal/index/update.go internal/index/query.go internal/index/tag_chave_test.go internal/index/persist_test.go internal/service/search.go internal/service/graph.go internal/service/tags_contrato_test.go testdata/tag_list_hierarquico.json docs/TOOLS.md docs/ESTADO.md
git commit -m "feat(index)!: one tag key - case, NFC, no hash - hierarchical in note_list, vault_search and tag_list

BREAKING CHANGE: vault_search.tags now matches subtags and ignores case,
Unicode form and a leading '#', like note_list already did; tag_list
returns the folded form (lowercase, NFC, no '#') and merges spellings
that differ only in case or Unicode form."
```

#### Verificações

- Step 2 FAIL de compilação colado; PASS dos Steps 4, 5, 6 colados.
- Dois `mutate.ps1` com exit 0 colados.
- `git diff` do golden com explicação por linha.
- `benchstat` colado e publicado.
- `grep -rn "strings.ToLower(t)\|strings.ToLower(k)\|ToLower(reqTag)" internal/index/query.go internal/service/search.go internal/service/graph.go` — Expected: vazio (nenhuma dobra fora de `ChaveDeTag`).
- `verify.ps1` verde após cada commit.

#### Regras de execução

- Nunca `git checkout/restore/stash/clean/reset`.
- `service` **não** importa `text`; a dobra que ele precisa vem por `index.ChaveDeTag`.
- `Note.Tags` e `note_metadata.tags` mantêm a grafia original; `IndexCacheFormatVersion` **não** muda.
- Commit 1 (bench) antes de qualquer alteração de produto; os binários `antes180_*` são construídos desse commit.
- Toda comparação de tag no produto passa por `ChaveDeTag` ou por `PathsComTags` — inclusive as que já pareciam certas.

#### Comando de mutação

Step 7 (dois `mutate.ps1`: âncora `strings.TrimPrefix(tag, "#")` em `chave.go`; âncora `if k == tk || strings.HasPrefix(k, tk+"/") {` em `query.go`).

#### Contrato de relatório

`task-180-report.md`: status, dois SHAs, saídas dos Steps 2, 4–8, o `git diff` do golden explicado, o `grep` das Verificações, última linha do `verify.ps1`.

---

