# Task 180 - uma chave de tag (caixa, NFC, sem `#`), hierárquica em `note_list`, `vault_search` e `tag_list`

## Status

DONE_WITH_CONCERNS

Tudo o que o brief pede está entregue e medido. As duas ressalvas estão em
`## Concerns`: uma tag inline em NFD é cortada pelo **parser** antes de chegar
ao índice (defeito anterior a esta Task, fora do alcance dela), e
`parser.dedupeTags` continua sendo uma segunda conta de "mesma tag" que não
pode chamar `index.ChaveDeTag` sem ciclo de import.

A rodada de correção 1 sobre a revisão `review-180.md` está entregue em
`aa8ec0b` e descrita em `## Fix round 1`, no fim deste relatório: N1, N2, N4,
N6 e N9 corrigidos, N3 e N5 corrigidos no texto, N7 e N8 sem ação por decisão
do orquestrador.

## Commits

| SHA | Assunto |
|---|---|
| `937e54c` | `test(service): benchmark vault_search with a tags filter [sem-doc]` |
| `18d9da4` | `feat(index)!: one tag key - case, NFC, no hash - hierarchical in note_list, vault_search and tag_list` |
| `c08c70c` | `docs(sugestoes): track the parser's NFD tag truncation as B20` |
| `aa8ec0b` | `fix(index): tag_list.prefix is documented as the string prefix it is; the prefix folding gets its test` |

O ` [sem-doc]` no commit 1 é a convenção já usada em `git log` para commit só de
teste — o hook de documentação recusa o assunto sem ele.

Arquivos do commit 2 (`git diff --cached --name-only`, por caminhos explícitos,
nunca `git add -A`):

```
docs/ESTADO.md
docs/TOOLS.md
internal/index/chave.go
internal/index/index.go
internal/index/persist_test.go
internal/index/query.go
internal/index/tag_chave_test.go
internal/index/update.go
internal/service/graph.go
internal/service/search.go
internal/service/tags_contrato_test.go
```

`testdata/tag_list_hierarquico.json` **não** está na lista, de propósito — ver
`## Step 6`.

O commit 3 é posterior à decisão do orquestrador de manter `internal/parser`
fora desta Task, e contém um arquivo só, `docs/SUGESTOES.md`: a entrada **B20**
na lista aberta, para o seguimento ficar rastreado. Sem código. `check_doc_refs`
e `check_readme_anchors` verdes sobre ele.

## Progresso

- 13:35 - li o brief, `CLAUDE.md`, `docs/papeis/desempenho.md` e a árvore.
- 13:40 - Step 1: `internal/service/bench_tags_test.go`, benchmark roda, `verify.ps1 -SkipCross -SkipNet` verde, commit 1 (`937e54c`).
- 13:42 - binários `antes180_service.test.exe` e `antes180_index.test.exe` construídos do commit 1, ANTES de qualquer alteração de produto.
- 13:43 - Step 2: `tag_chave_test.go` criado; RED de compilação.
- 13:44 - Steps 3 e 4 aplicados.
- 13:45 - achado fora do brief: o parser inline TRUNCA tag em NFD. Fixture do NFD movida para o frontmatter. Orquestrador avisado por mensagem.
- 13:46 - `go test -race ./internal/index/` verde.
- 13:50 - Step 5: `search.go` e `graph.go`; `tags_contrato_test.go`.
- 13:52 - Step 7 (primeira rodada): o meu próprio `TestVaultSearchTagsNFD` SOBREVIVEU à mutação de `ParaNFC` (exit 1). Fixture corrigida: as duas grafias em notas separadas.
- 13:54 - Step 6: `persist_test.go` ganhou a nota acentuada que faz a comparação de `Tags("", 0)` poder falhar.
- 13:56 - `go test -race` verde em index, service, mcpsrv e watcher; golden inalterado.
- 13:58 - Step 8 (index): **regressão** em `ListPorTag`, +16,08% tempo e +40,31% B/op.
- 14:00 - regressão diagnosticada e corrigida no `candidatosPorTagLocked`.
- 14:04 - `benchstat` do index refeito: `ListPorTag` de volta a `~`.
- 14:08 - `benchstat` do service acusou `NoteListPorTag` +24,15% B/op — era binário `depois180_service` **velho**, de 13:57, anterior à correção.
- 14:15 - service reconstruído e remedido: `NoteListPorTag` `~`, B/op idêntico.
- 14:16 - `docs/TOOLS.md` e `docs/ESTADO.md`.
- 14:22 - `verify.ps1` COMPLETO (14 etapas) verde; commit 2 (`18d9da4`).
- 14:31 - decisão do orquestrador sobre o parser recebida; **B20** escrito na lista aberta de `docs/SUGESTOES.md` e a seção `## Concerns` apontando para ele.
- 14:35 - `check_doc_refs` e `check_readme_anchors` verdes sobre o `SUGESTOES.md` editado; commit 3 (`c08c70c`), só `docs/SUGESTOES.md`.
- 14:40 - revisão `review-180.md` lida (N1–N9 e `## Commands I ran`). Rodada de correção 1 começada.
- 14:46 - N1: `docs/TOOLS.md` reescrito no `prefix` de `tag_list` e o parágrafo novo que separa `prefix` do filtro `tags`; a descrição do schema em `internal/mcpsrv/tools_read.go:389` dizia "subárvore" e foi alinhada.
- 14:47 - N6 e N9: `docs/ESTADO.md` (B20 no lugar de "Sem tarefa aberta", e a contagem do subcomando `index`) e `docs/TOOLS.md:216` (`tags` do `note_list` mantém a grafia).
- 14:49 - N4: asserção de `Tags(proj)` reescrita por lista ordenada; N2: `TestTagListDobraOPrefixoNosDoisRamos` no service e a checagem de prefixo acentuado no index.
- 14:51 - as quatro mutações (três de N2 + a de N4) deram EXIT=0.
- 14:57 - `verify.ps1 -SkipCross -SkipNet` verde, 11 etapas, 6 pulados conhecidos.
- 14:58 - N3 e N5 corrigidos no texto deste relatório; commit 4 (`aa8ec0b`).
- 14:59 - `## Fix round 1` escrito com as quatro saídas de mutação; `audit_reports.ps1 180` sem achado neste relatório.

## Step 2 - RED

```
$ go test ./internal/index/ -run 'TestChaveDeTag|TestListPorTag'
# github.com/jonyd/gobsidian/internal/index [github.com/jonyd/gobsidian/internal/index.test]
internal\index\tag_chave_test.go:23:13: undefined: ChaveDeTag
internal\index\tag_chave_test.go:49:12: ix.PathsComTags undefined (type *Index has no field or method PathsComTags)
internal\index\tag_chave_test.go:53:11: ix.PathsComTags undefined (type *Index has no field or method PathsComTags)
internal\index\tag_chave_test.go:57:11: ix.PathsComTags undefined (type *Index has no field or method PathsComTags)
internal\index\tag_chave_test.go:61:15: ix.PathsComTags undefined (type *Index has no field or method PathsComTags)
FAIL	github.com/jonyd/gobsidian/internal/index [build failed]
FAIL
```

## Steps 3 e 4 - GREEN

```
$ go test ./internal/index/ -run 'TestChaveDeTag|TestListPorTag' -v
=== RUN   TestChaveDeTagDobraCaixaHashENFC
--- PASS: TestChaveDeTagDobraCaixaHashENFC (0.00s)
=== RUN   TestListPorTagCasaSubtagENFD
--- PASS: TestListPorTagCasaSubtagENFD (0.01s)
PASS
ok  	github.com/jonyd/gobsidian/internal/index	0.576s

$ go test -race ./internal/index/
ok  	github.com/jonyd/gobsidian/internal/index	3.325s
```

## Step 5 - GREEN

```
$ go test ./internal/service/ -run 'TestVaultSearchTags|TestTagList' -v
=== RUN   TestTagListOrdenacao
--- PASS: TestTagListOrdenacao (0.01s)
=== RUN   TestTagListHierarquico
--- PASS: TestTagListHierarquico (0.01s)
=== RUN   TestVaultSearchTags
--- PASS: TestVaultSearchTags (0.03s)
=== RUN   TestTagListGolden
--- PASS: TestTagListGolden (0.01s)
=== RUN   TestVaultSearchTagsCasaSubtag
--- PASS: TestVaultSearchTagsCasaSubtag (0.03s)
=== RUN   TestVaultSearchTagsNFD
--- PASS: TestVaultSearchTagsNFD (0.03s)
=== RUN   TestTagListDevolveFormaDobrada
--- PASS: TestTagListDevolveFormaDobrada (0.03s)
=== RUN   TestTagListHierarquicoDobraGrafias
--- PASS: TestTagListHierarquicoDobraGrafias (0.03s)
PASS
ok  	github.com/jonyd/gobsidian/internal/service	0.800s
```

Suíte completa dos pacotes tocados, com `-race`:

```
$ go test -race ./internal/index/ ./internal/service/ ./internal/mcpsrv/ ./internal/watcher/
ok  	github.com/jonyd/gobsidian/internal/index	3.744s
ok  	github.com/jonyd/gobsidian/internal/service	38.326s
ok  	github.com/jonyd/gobsidian/internal/mcpsrv	9.905s
ok  	github.com/jonyd/gobsidian/internal/watcher	11.295s
```

### Duas divergências deliberadas em relação ao código literal do brief

**1. `tags_contrato_test.go` é `package service_test` e usa `createSearchService`,
não `package service` com `newTestService`.** `newTestService`
(`internal/service/read_test.go:22`) injeta `inverted = nil` — o comentário dele
diz isso: "os testes de leitura não tocam busca". Com o índice invertido nulo,
`svc.Search` casa zero resultados sempre, e um teste de filtro de tag que corre
no caminho vazio passa por não haver o que filtrar: não pode falhar.
`createSearchService` (`internal/service/search_test.go:21`) monta o invertido e
ainda o faz vir do CACHE, que é o ramo que o servidor usa.

**2. As duas grafias de "Acao" estão em notas SEPARADAS, não na mesma nota.**
Com as duas juntas, um pedido em NFC casava pela grafia NFC e o teste passava
mesmo com a normalização Unicode removida. Isso não é hipótese: está medido
abaixo, em `## Step 7`, com `exit 1`.

## Step 6 - o golden e o `Tags()` no reload

**O golden `testdata/tag_list_hierarquico.json` não mudou, e não foi
regenerado.** `git diff --stat testdata/` volta vazio e `TestTagListGolden`
passa sem `-update`. O motivo é o fixture: `cofreDeTagsHierarquicas`
(`internal/service/tag_list_golden_test.go:29`) usa `proj/alpha/um`,
`proj/beta`, `docs`, `proj/alpha/dois`, `proj/alpha`, `docs/api` e `zeta` —
todas já minúsculas, ASCII puro e sem `#`. `ChaveDeTag` é a identidade sobre
cada uma delas, então nem as chaves nem as contagens se movem. O brief previa
que o golden falharia; ele não falhou, e a explicação é essa. Nenhuma contagem
caiu, porque nenhuma linha mudou.

**`persist_test.go`: a asserção já existia; o que faltava era um corpus que a
fizesse morder.** A comparação pedida pelo brief está em
`internal/index/persist_test.go:182` desde antes desta Task
(`lido.Tags("", 0)` contra `fresco.Tags("", 0)` por `reflect.DeepEqual`). Só que
o corpus tinha uma única tag, `direito`, já minúscula e já em NFC: a comparação
passava com qualquer chave. Acrescentei `Acentuada.md` com
`tags: ["Ação", "ação", Direito]` — três grafias que dobram para duas chaves,
sendo que duas delas caem na MESMA chave dentro da MESMA nota, que é o caso que
a deduplicação de `publishNoteLocked` existe para tratar.

A leitura da dobra em ação está na saída da mutação do codec, em `## Step 7`: o
lado `construido` aparece como `[{direito 2} {ação 1}]` — `Direito` e `direito`
fundidos em duas notas, e as duas grafias de "Acao" da mesma nota contando
**uma** vez.

**Correção (N3 da revisão).** A frase anterior aqui dizia que foi essa asserção
que passou a morder o caminho de reload, e isso é mais forte do que o que ela
faz. A comparação `lido.Tags("",0)` × `fresco.Tags("",0)` é **simétrica**: os
dois lados publicam por `publishNoteLocked`, então uma `ChaveDeTag` errada
apareceria igual nos dois e ela não a veria. O que o corpus acentuado fez foi
passar a exercitar a dobra; a asserção que de fato segura o round-trip é a
comparação campo a campo pré-existente de `persist_test.go:199`, e é ela que
reprova junto na mutação do codec.

`IndexCacheFormatVersion` não muda: o cache guarda `Note.Tags` cru e o reload
republica por `publishNoteLocked`.

## Step 7 - provas de mutação

Seis mutações, todas com `exit 0` (o teste REPROVOU sob mutação). As duas
primeiras são as que o brief pede; as outras quatro cobrem o que o brief não
pedia e que eu não podia afirmar sem medir.

### 1. `chave.go`, `TrimPrefix('#')` (brief)

```
$ pwsh -File scripts/mutate.ps1 -Path internal/index/chave.go -Anchor 'strings.TrimPrefix(tag, "#")' -Replacement 'tag' -Test TestChaveDeTagDobraCaixaHashENFC -Package ./internal/index/
[...] Mutando internal/index/chave.go
      - strings.TrimPrefix(tag, "#")
      + tag

[...] go test -race -run TestChaveDeTagDobraCaixaHashENFC ./internal/index/
----------------------------------------------------------------------
--- FAIL: TestChaveDeTagDobraCaixaHashENFC (0.00s)
    tag_chave_test.go:24: ChaveDeTag("#Ação") = "#ação", quero "ação"
    tag_chave_test.go:24: ChaveDeTag("#Projeto") = "#projeto", quero "projeto"
FAIL
FAIL	github.com/jonyd/gobsidian/internal/index	0.661s
FAIL
----------------------------------------------------------------------
[OK] internal/index/chave.go restaurado byte a byte (SHA-256 confere).
[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
EXIT=0
```

### 2. `query.go`, a linha do casamento hierárquico (brief)

```
$ pwsh -File scripts/mutate.ps1 -Path internal/index/query.go -Anchor 'if k == tk || strings.HasPrefix(k, tk+"/") {' -Replacement 'if k == tk {' -Test TestListPorTagCasaSubtagENFD -Package ./internal/index/
[...] Mutando internal/index/query.go
      - if k == tk || strings.HasPrefix(k, tk+"/") {
      + if k == tk {

[...] go test -race -run TestListPorTagCasaSubtagENFD ./internal/index/
----------------------------------------------------------------------
--- FAIL: TestListPorTagCasaSubtagENFD (0.02s)
    tag_chave_test.go:60: PathsComTags(#PROJETO) = [b.md], quero [a.md b.md]
FAIL
FAIL	github.com/jonyd/gobsidian/internal/index	0.727s
FAIL
----------------------------------------------------------------------
[OK] internal/index/query.go restaurado byte a byte (SHA-256 confere).
[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
EXIT=0
```

### 3. `chave.go`, `ParaNFC` contra o índice

```
$ pwsh -File scripts/mutate.ps1 -Path internal/index/chave.go -Anchor 'return strings.ToLower(text.ParaNFC(strings.TrimPrefix(tag, "#")))' -Replacement 'return strings.ToLower(strings.TrimPrefix(tag, "#"))' -Test TestListPorTagCasaSubtagENFD -Package ./internal/index/
----------------------------------------------------------------------
--- FAIL: TestListPorTagCasaSubtagENFD (0.01s)
    tag_chave_test.go:64: PathsComTags(ação) = [], quero [d.md]
FAIL
FAIL	github.com/jonyd/gobsidian/internal/index	0.682s
FAIL
----------------------------------------------------------------------
[OK] internal/index/chave.go restaurado byte a byte (SHA-256 confere).
[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
EXIT=0
```

### 4. `chave.go`, `ParaNFC` contra as três tools do service

Esta é a que reprovou o **meu próprio teste** na primeira tentativa. Com as
duas grafias na mesma nota (a forma do brief), a saída foi:

```
[...] go test -race -run TestVaultSearchTagsNFD ./internal/service/
----------------------------------------------------------------------
ok  	github.com/jonyd/gobsidian/internal/service	1.776s
----------------------------------------------------------------------
[!] O teste PASSOU com a regra mutada.
    TestVaultSearchTagsNFD nao consegue reprovar sem essa regra: ela esta escrita, nao verificada.
EXIT=1
```

Depois de separar as grafias em duas notas:

```
$ pwsh -File scripts/mutate.ps1 -Path internal/index/chave.go -Anchor 'return strings.ToLower(text.ParaNFC(strings.TrimPrefix(tag, "#")))' -Replacement 'return strings.ToLower(strings.TrimPrefix(tag, "#"))' -Test 'TestVaultSearchTagsNFD|TestTagListDevolveFormaDobrada|TestTagListHierarquicoDobraGrafias' -Package ./internal/service/
----------------------------------------------------------------------
--- FAIL: TestVaultSearchTagsNFD (0.03s)
    tags_contrato_test.go:65: tags=[ação]: [{Path:e.md ...}], quero d.md e e.md — a mesma tag em NFD e em NFC
--- FAIL: TestTagListDevolveFormaDobrada (0.03s)
    tags_contrato_test.go:76: tag_list(aç) = [{Tag:ação Count:1 Children:[]}], quero UMA entrada ação com count 2 (duas notas, duas grafias)
--- FAIL: TestTagListHierarquicoDobraGrafias (0.03s)
    tags_contrato_test.go:89: tag_list(aç, hierarquico) = [{Tag:ação Count:1 Children:[]}], quero UMA entrada ação com count 2
FAIL
FAIL	github.com/jonyd/gobsidian/internal/service	0.852s
----------------------------------------------------------------------
[OK] internal/index/chave.go restaurado byte a byte (SHA-256 confere).
[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
EXIT=0
```

### 5. `search.go`, o filtro de tag de `vault_search`

A primeira forma da mutação (`-Replacement 'if false {'`) deu `EXIT=2`, e o
script estava certo: ela deixava `slices` importado e sem uso, e uma mutação
que quebra a compilação não prova cobertura nenhuma. Forma válida:

```
$ pwsh -File scripts/mutate.ps1 -Path internal/service/search.go -Anchor 'if _, ok := slices.BinarySearch(porTag, note.Path); !ok {' -Replacement 'if _, ok := slices.BinarySearch(porTag, note.Path); ok && false {' -Test TestVaultSearchTagsCasaSubtag -Package ./internal/service/
----------------------------------------------------------------------
--- FAIL: TestVaultSearchTagsCasaSubtag (0.03s)
    tags_contrato_test.go:42: tags=[#PROJETO]: 4 resultados, quero 2 (projeto e projeto/alpha): [b.md, c.md, a.md, d.md]
FAIL
FAIL	github.com/jonyd/gobsidian/internal/service	0.752s
----------------------------------------------------------------------
[OK] internal/service/search.go restaurado byte a byte (SHA-256 confere).
[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
EXIT=0
```

### 6. `graph.go`, a dobra no ramo hierárquico

```
$ pwsh -File scripts/mutate.ps1 -Path internal/service/graph.go -Anchor 'parts := strings.Split(index.ChaveDeTag(tag), "/")' -Replacement 'parts := strings.Split(strings.TrimPrefix(tag, "#"), "/")' -Test TestTagListHierarquicoDobraGrafias -Package ./internal/service/
----------------------------------------------------------------------
--- FAIL: TestTagListHierarquicoDobraGrafias (0.03s)
    tags_contrato_test.go:89: tag_list(aç, hierarquico) = [{Tag:ação Count:1 Children:[]}], quero UMA entrada ação com count 2
FAIL
FAIL	github.com/jonyd/gobsidian/internal/service	0.783s
----------------------------------------------------------------------
[OK] internal/service/graph.go restaurado byte a byte (SHA-256 confere).
[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
EXIT=0
```

### 7. `persist_codec.go`, o `Tags` no round-trip do cache

Prova de que a comparação de `Tags("", 0)` do `persist_test.go` pode falhar — e,
de quebra, a leitura da dobra em ação no lado `construido`:

```
$ pwsh -File scripts/mutate.ps1 -Path internal/index/persist_codec.go -Anchor 'e.strSlice(n.Tags)' -Replacement 'e.strSlice(nil)' -Test TestIndiceDeMetadadosRecarregadoEIdentico -Package ./internal/index/
----------------------------------------------------------------------
--- FAIL: TestIndiceDeMetadadosRecarregadoEIdentico (0.08s)
    persist_test.go:183: Tags("", 0) recarregado = [], construido = [{direito 2} {ação 1}]
    persist_test.go:199: Get(Acentuada.md) divergiu campo a campo:
        fresco      = ... Tags:[Ação Direito ação] ...
        recarregado = ... Tags:[] ...
FAIL
FAIL	github.com/jonyd/gobsidian/internal/index	0.838s
----------------------------------------------------------------------
[OK] internal/index/persist_codec.go restaurado byte a byte (SHA-256 confere).
[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
EXIT=0
```

`construido = [{direito 2} {ação 1}]` é a dobra medida: `Direito` (de
`Acentuada.md`) e `direito` (de `Civil/PONTO 03.md`) numa chave com contagem 2,
e as duas grafias de "Acao" da mesma nota numa chave com contagem **1** — a
deduplicação de `publishNoteLocked` fazendo o trabalho dela.

## Step 8 - benchstat

Protocolo de `docs/papeis/desempenho.md`: **7 rodadas alternadas de uma execução
de cada braço**, não `-count=7` por braço (rodar uma batelada e depois a outra
mede a deriva da máquina; já produziu dois números retratados neste projeto).
`-benchmem`. Braço "antes" = binários construídos do commit `937e54c`, ANTES de
qualquer alteração de produto.

### index

```
                  │ t180b_index_antes.txt │      t180b_index_depois.txt        │
                  │        sec/op         │   sec/op     vs base               │
TagsSemPrefixo-12            18.11µ ±  3%   15.85µ ± 12%  -12.47% (p=0.001 n=7)
ListPorTag-12                631.3µ ±  2%   635.2µ ± 10%        ~ (p=1.000 n=7)
geomean                      106.9µ         100.3µ         -6.16%

                  │        B/op           │    B/op      vs base               │
TagsSemPrefixo-12           7.352Ki ±  0%  7.352Ki ± 0%         ~ (p=1.000 n=7) ¹
ListPorTag-12               119.1Ki ±  0%  119.1Ki ± 0%         ~ (p=1.000 n=7) ¹
geomean                     29.59Ki        29.59Ki        +0.00%

                  │      allocs/op        │  allocs/op   vs base               │
TagsSemPrefixo-12             8.000 ±  0%    8.000 ± 0%         ~ (p=1.000 n=7) ¹
ListPorTag-12                 12.00 ±  0%    12.00 ± 0%         ~ (p=1.000 n=7) ¹
geomean                       9.798          9.798         +0.00%
¹ all samples are equal
```

### service

```
                           │ t180b_service_antes.txt │    t180b_service_depois.txt        │
                           │         sec/op          │   sec/op     vs base               │
TagListPlano-12                       19.72µ ± 13%    17.63µ ±  7%  -10.62% (p=0.001 n=7)
TagListHierarquico-12                 2.750m ± 21%    3.134m ± 39%        ~ (p=0.097 n=7)
NoteListPorTag-12                     235.9µ ± 52%    235.6µ ± 13%        ~ (p=0.535 n=7)
SearchFiltroFrontmatter-12            13.04m ± 50%    13.67m ±  3%        ~ (p=0.128 n=7)
SearchFiltroTags-12                   13.36m ±  3%    12.85m ±  8%        ~ (p=0.073 n=7)
geomean                               1.174m          1.180m         +0.50%

                           │          B/op           │    B/op      vs base               │
TagListPlano-12                      13.35Ki ±  0%   13.35Ki ± 0%         ~ (p=1.000 n=7) ¹
TagListHierarquico-12                262.2Ki ±  0%   262.2Ki ± 0%         ~ (p=1.000 n=7) ¹
NoteListPorTag-12                    41.41Ki ±  0%   41.41Ki ± 0%         ~ (p=0.266 n=7)
SearchFiltroFrontmatter-12           4.427Mi ±  0%   4.426Mi ± 0%         ~ (p=0.073 n=7)
SearchFiltroTags-12                  1.926Mi ±  0%   1.935Mi ± 0%    +0.48% (p=0.001 n=7)
geomean                              264.6Ki         264.8Ki        +0.09%

                           │        allocs/op        │  allocs/op   vs base               │
TagListPlano-12                        9.000 ±  0%     9.000 ± 0%         ~ (p=1.000 n=7) ¹
TagListHierarquico-12                 9.998k ±  0%    9.998k ± 0%         ~ (p=1.000 n=7) ¹
NoteListPorTag-12                      210.0 ±  0%     210.0 ± 0%         ~ (p=1.000 n=7) ¹
SearchFiltroFrontmatter-12            30.12k ±  0%    30.12k ± 0%    -0.00% (p=0.049 n=7)
SearchFiltroTags-12                   10.15k ±  0%    10.15k ± 0%         ~ (p=0.244 n=7)
geomean                               1.420k          1.420k        +0.01%
¹ all samples are equal
```

### A regressão que apareceu, e o que ela era

A **primeira** forma de `candidatosPorTagLocked` — a do brief, com um `casam`
que sempre devolve a fatia já ordenada e compactada — regrediu `ListPorTag`:

```
ListPorTag-12   636.6µ ± 29%  739.0µ ± 6%  +16.08% (p=0.026 n=7)   sec/op
ListPorTag-12   119.1Ki ± 0%  167.1Ki ± 0% +40.31% (p=0.001 n=7)   B/op
ListPorTag-12    12.00 ± 0%    13.00 ± 0%   +8.33% (p=0.001 n=7)   allocs/op
```

Causa: `tag_mode=any` ordena e compacta **uma vez no fim**, sobre o acumulado.
Fazer `casam` ordenar e compactar por tag obrigava `any` a pagar um `sort` e uma
**cópia** por tag para depois ordenar tudo de novo — a fatia extra é o `+1
alloc` e os `+48 KiB`. Correção: o casamento anexa à fatia do chamador
(`casam(pedida, dst)`), `any` acumula direto e ordena no fim, e `all` usa um
`ordenado()` que embrulha o mesmo `casam` com `Sort`+`Compact`, porque a
interseção é por busca binária e precisa de cada conjunto ordenado à parte.
**A linha-âncora de mutação (`if k == tk || strings.HasPrefix(k, tk+"/") {`)
continua existindo exatamente uma vez** — a prova 2 acima é posterior à
correção. Depois dela, `ListPorTag` voltou a `~` com B/op e allocs/op
**idênticos** ao braço "antes".

Um segundo susto foi erro meu de instrumento, não do código: a rodada de
`benchstat` do service às 14:08 acusou `NoteListPorTag` +24,15% de B/op porque
eu havia reconstruído só o `depois180_index.test.exe` depois da correção — o
binário do service ainda era o de 13:57, com a versão que regride. Refeito com o
binário certo, `NoteListPorTag` dá `~` com B/op idêntico.

### Os dois itens com p < 0,05, e o único que pedia investigação

São **dois** na tabela acima, e a frase anterior desta seção dizia "o único",
contrariada pela linha de cima dela (N5 da revisão). O segundo é
`SearchFiltroFrontmatter-12 allocs/op -0.00% (p=0.049 n=7)`: uma **melhora** de
−0,00 % sobre 30,12k, ruído em torno de zero, e o brief manda investigar piora.
Nada a investigar ali. O que pedia investigação, e teve:

`SearchFiltroTags`, B/op +0,48% (1.926Mi -> 1.935Mi, p=0.001). Investigado antes
do commit, e o número que decide é o `allocs/op`: **idêntico nos dois braços**
(10.15k, e 10150 exatos numa corrida de N fixo). Ou seja, não é alocação nova em
volume, é uma alocação a mais de ~9,6 KiB por consulta: o conjunto ordenado que
`index.PathsComTags` resolve UMA vez e que **escapa** (é devolvido), no lugar do
`map[string]bool` por resultado, que **não escapava** e saía da pilha. O tempo
não piorou (`~`, p=0.073, tendendo a melhor). É o preço da semântica correta,
medido, e está publicado em `docs/ESTADO.md`.

## Verificações do brief

`grep` exigido — vazio, como pedido:

```
$ grep -rn "strings.ToLower(t)\|strings.ToLower(k)\|ToLower(reqTag)" internal/index/query.go internal/service/search.go internal/service/graph.go
$ echo $?
1
```

Todo `ToLower` que sobrou nesses três arquivos, e o que cada um dobra:

```
internal/index/query.go:177:  (comentário)
internal/index/query.go:251:  strings.ToLower(mode)      -> "any"/"all", não é tag
internal/index/query.go:412:  strings.ToLower(q.Order)   -> "asc"/"desc"
internal/index/query.go:416:  strings.ToLower(q.Sort)    -> critério de ordenação
internal/service/graph.go:409: (comentário)
```

Todo `ToLower` sobre tag em produção, no repositório inteiro:

```
$ grep -rn "ToLower" --include=*.go internal/ cmd/ | grep -v _test | grep -i "tag"
internal/doctor/checks_windows.go:17:  (comentário sobre build tag, não sobre tag de nota)
internal/index/chave.go:50:            (comentário)
internal/index/chave.go:58:            return strings.ToLower(text.ParaNFC(strings.TrimPrefix(tag, "#")))
internal/parser/ast.go:158:            key := strings.ToLower(tag)   <- ver ## Concerns
```

Grafo de imports, re-extraído: nenhuma aresta nova, e `service` continua **não**
importando `text`.

```
$ GOOS=windows go list -f '{{.ImportPath}} -> {{.Imports}}' ./internal/index/ ./internal/service/
internal/index   -> [... internal/parser internal/text internal/vault ...]
internal/service -> [... internal/index internal/parser internal/search internal/vault internal/writer ...]
```

## `verify.ps1` COMPLETO (14 etapas)

```
[...] 1. go build
[OK] go build
[...] 2. go test -race
[OK] go test -race
[...] 3. contagem de testes pulados
[!] 6 testes pulados
     --- SKIP: TestAjudanteSeguraTrava (0.00s)
     --- SKIP: TestListenRestringePermissaoUnix (0.00s)
     --- SKIP: TestSignalCancelsContext (0.10s)
     --- SKIP: TestPerfilDeHeapServindo (0.00s)
     --- SKIP: TestWriteAtomicPreservaOModoDoAlvo (0.00s)
     --- SKIP: TestNew_FailsOnUnwatchablePath (0.00s)
[...] 4. go test (tetos de latencia, sem -race)
[OK] go test (tetos de latencia, sem -race)
[...] 5. go vet (windows)
[OK] go vet (windows)
[...] 6. go vet (linux)
[OK] go vet (linux)
[...] 7. go vet (darwin)
[OK] go vet (darwin)
[...] 8. gofmt
[OK] gofmt
[...] 9. golangci-lint
[OK] golangci-lint
[...] 10. golangci-lint (linux)
[OK] golangci-lint (linux)
[...] 11. check_net (RNF-30)
[OK] check_net (RNF-30)
[...] 12. check_tool_params
[OK] check_tool_params
[...] 13. check_doc_refs
[OK] check_doc_refs
[...] 14. check_readme_anchors
[OK] check_readme_anchors

[OK] Bateria completa. Pode commitar.
```

Os 6 pulados são os de sempre (elevação de privilégio, socket Unix, sinal,
perfil de heap, modo de arquivo POSIX, caminho não-vigiável); nenhum deles é de
tag e nenhum entrou nesta Task. Saída completa em
`%LOCALAPPDATA%\gobsidian-bench\2026-09-02\verify180.txt`.

O commit 1 rodou com `-SkipCross -SkipNet`, como o brief autoriza; o commit 2
rodou o gate completo.

## Concerns

### 1. Tag inline em NFD é cortada pelo parser, antes de chegar ao índice

Medido, com `ChaveDeTag` já em pé:

```
corpo da nota: "# D\n\n#Ação\n"     (NFD: A c U+0327 a U+0303 o)
n.Tags     = ["Ac"]
chave      = "ac"
```

`parser.tagNameChar` (`internal/parser/ext_tag.go:36`) aceita
`unicode.IsLetter`, `unicode.IsDigit`, `-`, `_` e `/`, e **não** `unicode.Mn`,
então o parser inline para no sinal combinante. É anterior a esta Task e fora do
alcance dela: nenhuma dobra de chave conserta uma tag que já chegou cortada.
Não mexi em `internal/parser` — não está nos arquivos do brief e mudar o
alfabeto do parser mexe com os goldens dele. Avisei o orquestrador por mensagem
assim que medi, e registrei o fato em `docs/ESTADO.md`. O orquestrador
decidiu manter `internal/parser` fora desta Task e mandou registrar o
seguimento na lista aberta de `docs/SUGESTOES.md`: está lá como **B20**, com
a mesma medição e a mesma referência de linha.

Consequência prática: hoje um `#Ação` digitado num macOS indexa em silêncio como
a tag `Ac`. **Recomendo uma tarefa própria** para `tagNameChar` (acrescentar
`unicode.Mn`), com passe de golden do parser e conferência de paridade contra o
Obsidian.

Isso **não** torna falsa a redação que entrou em `docs/TOOLS.md`: o que está
escrito lá é que a **comparação** ignora caixa e forma Unicode, e ela ignora — o
pedido em NFD casa uma nota gravada em NFC e vice-versa, provado em
`## Step 7`, prova 4. O que está quebrado a montante é o **reconhecimento** de
uma tag inline em NFD. Pelo frontmatter, que é onde a forma Unicode de fato
varia entre um cofre escrito no macOS e um escrito no Windows, o caminho inteiro
funciona.

### 2. `parser.dedupeTags` é uma segunda conta de "mesma tag"

`internal/parser/ast.go:158` decide, com `strings.ToLower(tag)` e sem NFC, quais
grafias sobrevivem em `Note.Tags` dentro de uma nota. É a mesma pergunta que
`ChaveDeTag` responde, respondida em outro lugar e com outra regra — a definição
de "uma conta por regra" que o `CLAUDE.md` cobra.

Não corrigi, e a razão é estrutural: `parser` não pode importar `index` (o grafo
tem `index -> parser`; a volta seria ciclo). O caminho existe e tem precedente
nesta base: mover a conta para `internal/text` e fazer `index.ChaveDeTag`
delegar, exatamente como `chaveDeCaminho` delega para `text.ChaveDeCaminho`
desde a Task 169 — `parser` já importa `text`. Não fiz porque expande o escopo
para dois pacotes fora do brief e muda o que `note_metadata.tags` devolve.

**Não há divergência de comportamento observável hoje**, e isso está verificado:
duas grafias que diferem só em caixa são reduzidas a uma por `dedupeTags`; duas
que diferem só em forma Unicode sobrevivem as duas e são reduzidas a uma pela
deduplicação de `publishNoteLocked`. Os dois caminhos chegam a **uma** entrada em
`ix.tags` — é o que a saída da prova 7 mostra (`{ação 1}` para a nota com as duas
grafias). A diferença é só qual grafia `note_metadata` exibe.

### 3. O que eu mudei em relação ao código literal do brief

Três coisas, todas com o motivo medido e nenhuma delas encolhendo o escopo:
o pacote e o helper de `tags_contrato_test.go` (senão o teste corre no caminho
vazio), as duas grafias em notas separadas (senão o teste sobrevive à mutação,
`exit 1` colado), e a forma de `candidatosPorTagLocked` (senão `ListPorTag`
regride 16% / 40%, `benchstat` colado). Detalhe em `## Step 5`, `## Step 7` e
`## Step 8`.

## audit_reports

`pwsh -File scripts/audit_reports.ps1 180`, saída completa. **Zero achados no
relatório desta Task** — o bloco `=== Relatorios (1) ===` vem vazio. Os 14
achados são todos do ledger do marco `2026-07-25-gobsidian-v01`, pré-existentes
e alheios à Task 180; o `EXIT=1` vem deles.

```
=== Relatorios (1) ===

=== Ledger ===
  .superpowers\sdd\2026-07-25-gobsidian-v01\progress.md:9: [SHA-NAO-CONFERE] Task 4 ...
  .superpowers\sdd\2026-07-25-gobsidian-v01\progress.md:11: [SHA-NAO-CONFERE] Task 6 ...
  .superpowers\sdd\2026-07-25-gobsidian-v01\progress.md:284: [RELATORIO-AUSENTE] Task 94 marcada completa e sem task-94-report.md
  .superpowers\sdd\2026-07-25-gobsidian-v01\progress.md:287: [RELATORIO-AUSENTE] Task 95 marcada completa e sem task-95-report.md
  .superpowers\sdd\2026-07-25-gobsidian-v01\progress.md:290: [RELATORIO-AUSENTE] Task 96 marcada completa e sem task-96-report.md
  .superpowers\sdd\2026-07-25-gobsidian-v01\progress.md:315: [RELATORIO-AUSENTE] Task 103 marcada completa e sem task-103-report.md
  .superpowers\sdd\2026-07-25-gobsidian-v01\progress.md:318: [RELATORIO-AUSENTE] Task 102 marcada completa e sem task-102-report.md
  .superpowers\sdd\2026-07-25-gobsidian-v01\progress.md:321: [RELATORIO-AUSENTE] Task 101 marcada completa e sem task-101-report.md
  .superpowers\sdd\2026-07-25-gobsidian-v01\progress.md:324: [RELATORIO-AUSENTE] Task 100 marcada completa e sem task-100-report.md
  .superpowers\sdd\2026-07-25-gobsidian-v01\progress.md:327: [RELATORIO-AUSENTE] Task 99 marcada completa e sem task-99-report.md
  .superpowers\sdd\2026-07-25-gobsidian-v01\progress.md:330: [RELATORIO-AUSENTE] Task 98 marcada completa e sem task-98-report.md
  .superpowers\sdd\2026-07-25-gobsidian-v01\progress.md:333: [RELATORIO-AUSENTE] Task 97 marcada completa e sem task-97-report.md
  .superpowers\sdd\2026-07-25-gobsidian-v01\progress.md:988: [SHA-FANTASMA] deadbee nao existe no repositorio
  .superpowers\sdd\2026-07-25-gobsidian-v01\progress.md:5158: [SHA-NAO-CONFERE] Task 153 ...

[!] 14 achado(s). Nenhum e automaticamente um defeito — cada um e
    uma frase ou um SHA que precisa de uma pessoa confirmando.
[i] Onde nao houve medicao, 'nao medido' e a resposta certa e nao e sinalizada.

EXIT=1
```

---

## Fix round 1

Sobre a revisão `review-180.md` (veredito CHANGES_REQUIRED). Commit `aa8ec0b`,
cinco arquivos por caminho explícito:

```
docs/ESTADO.md
docs/TOOLS.md
internal/index/tag_chave_test.go
internal/mcpsrv/tools_read.go
internal/service/tags_contrato_test.go
```

`git status --porcelain -- internal/ docs/ testdata/` volta vazio depois do
commit. **Nenhuma mudança de comportamento**: o golden não foi tocado, a linha
de casamento hierárquico do filtro `tags` não foi tocada, e nenhum arquivo de
produção mudou exceto a string de descrição do schema em `mcpsrv`.

### N1 — o contrato de `tag_list.prefix`, opção (a): o doc segue o código

O texto antigo (`O prefixo casa a si mesmo e suas subtags`) veio verbatim do
brief e descrevia a regra do filtro `tags`, não a de `prefix`. `prefix` é
prefixo de string sobre a chave dobrada, o que erra nas duas direções — casa
`proj/alpha` para `proj/al`, e não casa `civil` para `civil/` — e é
deliberado desde a Task 169 (autocompletar), fixado no golden. O novo texto
diz isso com os dois exemplos, e ganhou um parágrafo que separa as duas
operações:

> **`prefix` não é o filtro `tags`.** São duas operações diferentes, de
> propósito. O `tags` de `note_list` e de `vault_search` casa por SEGMENTO […]
> O `prefix` daqui é autocompletar: prefixo de string sobre a chave dobrada […]

A metade correta do contrato — `#` opcional, insensível a caixa e a forma
Unicode — ficou como estava.

O `grep` que a decisão pedia: `internal/mcpsrv` não usa a palavra "subtags",
mas dizia a mesma coisa em uma palavra —
`internal/mcpsrv/tools_read.go:389`, `jsonschema:"Restringe a uma subárvore,
ex.: 'civil/'"`. Duas cópias do mesmo fato, e a que o host lê era a mais
errada das duas (o exemplo `civil/` é justamente o caso que não devolve
`civil`). Alinhada para a mesma redação. É a única linha de produção do commit.

### N2 — a dobra do prefixo, agora verificada nos dois ramos

O buraco era o fixture: `"proj"`, `"proj/al"`, `"aç"`, `""` — todo prefixo de
teste era ASCII minúsculo, onde `ChaveDeTag` e `strings.ToLower` são a mesma
função. Dois prefixos que as separam:

- `"#AÇ"` — carrega o `#` e a caixa. `ToLower` deixa `"#aç"`, que não é
  prefixo de `"ação"`.
- `"ac" + U+0327` (NFD) — carrega o sinal combinante. `ToLower` não normaliza,
  e a chave está em NFC.

Entraram em dois lugares, porque as três mutações da revisão rodam em dois
pacotes: `TestTagListDobraOPrefixoNosDoisRamos`
(`internal/service/tags_contrato_test.go`, tabela × `Hierarchical` false e
true) e um laço sobre `ix.Tags(prefixo, 0)` dentro de
`TestListPorTagCasaSubtagENFD` (`internal/index/tag_chave_test.go`) — sem este
segundo, a mutação de `query.go` rodada contra `./internal/index/` continuaria
sobrevivendo.

As três mutações da revisão, agora:

```
$ pwsh -File scripts/mutate.ps1 -Path internal/index/query.go -Anchor 'prefix = ChaveDeTag(prefix)' -Replacement 'prefix = strings.ToLower(prefix)' -Test '.' -Package ./internal/index/
FAIL	github.com/jonyd/gobsidian/internal/index	2.088s
FAIL
----------------------------------------------------------------------
[OK] internal/index/query.go restaurado byte a byte (SHA-256 confere).

[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
EXIT=0

$ pwsh -File scripts/mutate.ps1 -Path internal/index/query.go -Anchor 'prefix = ChaveDeTag(prefix)' -Replacement 'prefix = strings.ToLower(prefix)' -Test '.' -Package ./internal/service/
----------------------------------------------------------------------
[OK] internal/index/query.go restaurado byte a byte (SHA-256 confere).

[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
EXIT=0

$ pwsh -File scripts/mutate.ps1 -Path internal/service/graph.go -Anchor 'prefixo := index.ChaveDeTag(req.Prefix)' -Replacement 'prefixo := strings.ToLower(req.Prefix)' -Test '.' -Package ./internal/service/
FAIL	github.com/jonyd/gobsidian/internal/service	33.386s
FAIL
----------------------------------------------------------------------
[OK] internal/service/graph.go restaurado byte a byte (SHA-256 confere).

[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
EXIT=0
```

### N4 — a precedência, e a prova de que a asserção agora confere os dois nomes

A forma antiga era `len(tags) != 2 || tags[0].Tag != "projeto" && tags[1].Tag
!= "projeto"`, que é `len != 2 || (a != x && b != x)`: bastava UM dos dois ser
`projeto`, e `projeto/alpha` nunca era conferido. Agora os nomes saem para uma
fatia, são ordenados e comparados com `[]string{"projeto", "projeto/alpha"}`.

A prova pedida — fazer `Tags` largar `projeto/alpha` e mostrar a reprovação:

```
$ pwsh -File scripts/mutate.ps1 -Path internal/index/query.go -Anchor 'if len(paths) >= minCount && strings.HasPrefix(t, prefix) {' -Replacement 'if len(paths) >= minCount && strings.HasPrefix(t, prefix) && !strings.Contains(t, "/") {' -Test 'TestListPorTagCasaSubtagENFD' -Package ./internal/index/
----------------------------------------------------------------------
--- FAIL: TestListPorTagCasaSubtagENFD (0.01s)
    tag_chave_test.go:84: Tags(proj) = [{projeto 1}], quero projeto e projeto/alpha dobradas
FAIL
FAIL	github.com/jonyd/gobsidian/internal/index	0.620s
FAIL
----------------------------------------------------------------------
[OK] internal/index/query.go restaurado byte a byte (SHA-256 confere).

[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
EXIT=0
```

A linha de falha nomeia `[{projeto 1}]` contra os dois esperados: é
exatamente o caso que a forma antiga deixava passar.

### N6 — `ESTADO.md`

A frase `Sem tarefa aberta.` virou **Aberta como B20 em `docs/SUGESTOES.md`**.

### N9 — as duas saídas não dobradas

`docs/TOOLS.md`, retorno de `note_list`: `tags` vem com a grafia original da
nota, como `note_metadata.tags`. `docs/ESTADO.md`, entrada da Task 180: o
subcomando `index` da CLI reporta `len(idx.Tags("", 1))`, que agora conta
chaves dobradas, então a contagem de tags distintas cai onde grafias se
fundem. De quanto num cofre real: **não medido** — não rodei o subcomando
contra `vault_5000` nem contra cofre do dono nesta rodada.

### N3 e N5 — corrigidos no texto deste relatório

N3 em `## Step 6`: a comparação `lido.Tags("",0)` × `fresco.Tags("",0)` é
simétrica e não pega chave errada; quem segura o round-trip é a comparação
campo a campo de `persist_test.go:199`. N5 em `## Step 8`: são dois itens com
p < 0,05, e o segundo (`SearchFiltroFrontmatter`, allocs/op −0,00 %, p=0.049)
é melhora dentro do ruído, sem nada a investigar. Nenhuma mudança de código
por causa desses dois.

### N7 e N8 — sem ação

Registrados pelo orquestrador. N7 (`porTag` como retrato fora do laço) tem a
mesma propriedade de `porFrontmatter`, que já era aceita; N8 (`vault_stats`
prometendo contagens que `StatsResult` não tem) é anterior a esta Task.

### Gate

`pwsh -File scripts/verify.ps1 -SkipCross -SkipNet` — doc e teste, mais uma
string de schema:

```
[...] 9. check_tool_params
[OK] check_tool_params
[...] 10. check_doc_refs
[OK] check_doc_refs
[...] 11. check_readme_anchors
[OK] check_readme_anchors

[OK] Bateria completa. Pode commitar.
```

Os 6 pulados são os conhecidos (`TestAjudanteSeguraTrava`,
`TestListenRestringePermissaoUnix`, `TestSignalCancelsContext`,
`TestPerfilDeHeapServindo`, `TestWriteAtomicPreservaOModoDoAlvo`,
`TestNew_FailsOnUnwatchablePath`).
