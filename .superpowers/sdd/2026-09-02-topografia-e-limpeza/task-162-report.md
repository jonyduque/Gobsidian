# Task 162 — Confundidores: corpus do ranking, contexto de backlink, ordem de mapa, `idx == nil`

## Progresso

- 00:15 — inicio; HEAD `a313f32` (nao `6c1f217`; o lead informou um SHA anterior ao ultimo commit de docs da Task 161).
- 00:19 — leitura dos sitios concluida: `ranking_golden_test.go`, `backlink_contexto_test.go`, `backlink_heading_test.go`, `persist_test.go`, `bm25_test.go`, `bm25.go`, `contexto_link.go`, `backlinks.go`. Goldens estao em `internal/service/testdata/ranking/`, nao em `testdata/ranking/` na raiz como o brief diz.
- 00:24 — Step 1 pronto: gerador reescrito, seis `.tsv` regenerados, mutacao do peso de heading contra `TestRankingGolden` com exit 0.
- 00:25 — Steps 2 e 3 prontos: contexto de backlink afirma vizinhanca (mutacao `Context: l.Raw` reprovou os dois testes); `persist_test.go` ganhou segunda origem e ordenacao (sem ordenar, reprova 20/20).
- 00:30 — Step 4 pronto: `TestBM25WeightBody` virou `TestBM25PesoDeCorpoEOMenorDosTres` com indice real; duas mutacoes com exit 0. `gofmt -l internal/` vazio; os tres pacotes passam.
- 00:36 — `verify.ps1` reprovou em `golangci-lint` (QF1012, `WriteString(fmt.Sprintf(...))` em duas linhas novas do gerador). Trocado por `fmt.Fprintf(&corpo, ...)`.
- 00:40 — `verify.ps1` verde (exit 0). Commit `d12f84f`.

Fix round 1 (R=1), review `review-162.md`:

- 01:03 — leitura de B1–N3.
- 01:05 — B1, N1, N2 reescritos nas duas docstrings; N3 aplicado (`20 + ((i*7+29)%60) + i/60`); seis `.tsv` regenerados com `-update`; `TestRankingGolden` verde, `conferirDiscriminacao` passa.
- 01:06 — mutacao do peso de heading re-rodada sobre o corpus novo: exit 0.
- 01:11 — `verify.ps1` verde (exit 0).
- 01:12 — commit da rodada. O SHA nao aparece aqui de proposito: o relatorio entra no proprio commit, entao um SHA escrito nele so poderia estar errado. O SHA vai na mensagem ao lead e em `git log`.

## Status

**DONE.** SHA `d12f84f`.

## Sitio a sitio

| Sitio | Confundidor | Conserto | Prova |
|---|---|---|---|
| `internal/service/ranking_golden_test.go` (gerador + goldens) | 299 de 300 notas identicas; toda consulta empatava e o `.tsv` congelava o desempate por caminho, nao o ranking | gerador com comprimento, frequencia e presenca de termo variando por formula deterministica; `conferirDiscriminacao` reprova corpus que empata nos dois primeiros | `mutate.ps1` peso de heading -> peso de corpo, `TestRankingGolden`, exit 0 |
| `internal/index/backlink_contexto_test.go` | uma implementacao que devolvesse `link.Raw` passava as quatro assercoes | assercao por grafia de link, de uma palavra ANTES e uma DEPOIS do link, ambas fora dos colchetes | `mutate.ps1` `Context: contextoDoLink(body, l)` -> `Context: l.Raw`, exit 0 |
| `internal/index/backlink_heading_test.go` (`TestContextoNaoEstouraOOrcamento`) | so havia teto (176 bytes) e `len != 0`; `link.Raw` = 8 bytes passava nos dois | fixture ganhou `esquerda` e `direita` coladas ao link; teste afirma que as duas chegam ao contexto | mesma mutacao acima, exit 0 |
| `internal/index/persist_test.go` | `reflect.DeepEqual` sobre fatia vinda de iteracao de mapa, com uma origem so — afirmava sequencia, media conjunto, e nao podia divergir | segunda origem (`Penal/Citante.md`), checagem de fixture exigindo 2 origens distintas, e ordenacao das duas fatias antes do `DeepEqual` | com a ordenacao removida, `-count=20` reprova (saida colada abaixo) |
| `internal/index/classify_test.go` | `ordenarBacklinks` ja existia no pacote de teste, com chave parcial `(From, Anchor, Alias)` | reforcada para chave TOTAL (`+ Heading, Context, Kind`), em vez de criar uma segunda funcao — "uma conta por regra" | build (a duplicata reprovava a compilacao, foi o que a revelou) |
| `internal/search/bm25_test.go` (`TestBM25WeightBody`) | `idx == nil` faz `pesoDeCampo` devolver `WeightBody` na primeira linha; o teste nomeava peso de campo e exercitava o ramo sem campo | virou `TestBM25PesoDeCorpoEOMenorDosTres`: indice real, tres notas com o MESMO multiconjunto de tokens, afirma titulo > heading > corpo | duas `mutate.ps1` (peso de heading e peso de titulo), exit 0 nas duas |

### Desvio do brief, declarado

O Step 4 do brief manda "afirmar `scoreHeading > scoreBody`" e nomeia o teste
`TestBM25PesoDeHeading`. Escrito ao pe da letra, isso seria **duplicata exata**
de `TestBM25WeightHeadings` (`bm25_test.go:94`), que ja monta indice real com
duas notas de mesmo multiconjunto de tokens e afirma exatamente isso. O que
faltava era um teste que **`TestBM25WeightBody` pudesse honrar**: que
`WeightBody` e o PISO da escala. Por isso a assercao e a escada inteira
(titulo > heading > corpo) em um unico fixture de tres notas, e o nome e
`TestBM25PesoDeCorpoEOMenorDosTres`. As duas mutacoes exigidas foram rodadas
contra esse nome, ambas com exit 0.

Nota sobre `frase-exata`: e a unica consulta que casa **uma** nota entre 300, e
casar uma so e a propriedade que aquele golden existe para congelar. Por isso
`conferirDiscriminacao` trata esse nome a parte, e afirma `len(got) == 1` em vez
de comparar os dois primeiros. As outras cinco consultas passam pela checagem de
discriminacao completa.

## `head -2` dos seis `.tsv` regenerados

```
-- com-acento.tsv (20 linhas)
pasta00/n0000.md	1.716162
pasta09/n0189.md	1.705403
-- dois-termos.tsv (20 linhas)
pasta00/n0000.md	3.997986
pasta05/n0105.md	3.784420
-- frase-exata.tsv (1 linha)
pasta00/n0150.md	6.504890
-- so-em-heading.tsv (20 linhas)
pasta00/n0000.md	4.072235
pasta04/n0044.md	3.994505
-- so-no-titulo.tsv (6 linhas)
pasta05/n0035.md	7.022125
pasta05/n0045.md	6.880230
-- termo-amplo.tsv (50 linhas)
pasta03/n0043.md	0.003212
pasta03/n0103.md	0.003206
```

Em todos, a primeira linha tem score estritamente maior que a segunda —
`frase-exata` nao tem segunda linha, e a razao esta acima.

`so-no-titulo.tsv` inteiro, que e onde o contraste titulo-vs-corpo aparece: as
cinco notas com "intercorrente" no titulo vem antes da unica que so o tem no
corpo.

```
pasta05/n0035.md	7.022125
pasta05/n0045.md	6.880230
pasta05/n0005.md	6.598737
pasta05/n0015.md	6.419046
pasta05/n0025.md	6.326284
pasta00/n0250.md	5.049972
```

### Uma tentativa que reprovou, e por que

A primeira versao do gerador usou o enchimento literal do brief,
`20 + (i*7)%60`. Reprovou:

```
=== RUN   TestRankingGolden/termo-amplo
    ranking_golden_test.go:262: termo-amplo: os dois primeiros empatam (0.003205); o corpus nao discrimina
```

`(i*7)%60` tem periodo 60 e o corpus tem 300 notas, entao cada comprimento saia
repetido cinco vezes; e como 3, 4 e 5 dividem 60, essas cinco notas tambem
concordavam em tf de "nota", em "prescricao" e em "execucao" — ficavam
equivalentes para a consulta `termo-amplo`. O enchimento e
`20 + (i*7)%60 + i/60`, com o motivo escrito na docstring da `corpusGolden`. A
propria verificacao que o brief pediu foi o que pegou isso.

## Prova de mutacao — Step 1 (peso de heading, contra o golden)

```
pwsh -File scripts/mutate.ps1 -Path internal/search/bm25.go `
  -Anchor 'return WeightHeadings' -Replacement 'return WeightBody' `
  -Test TestRankingGolden -Package ./internal/service/
```

```
[...] Mutando internal/search/bm25.go
      - return WeightHeadings
      + return WeightBody

[...] go test -race -run TestRankingGolden ./internal/service/
----------------------------------------------------------------------
--- FAIL: TestRankingGolden (0.92s)
    --- FAIL: TestRankingGolden/so-em-heading (0.00s)
        ranking_golden_test.go:300: ranking mudou.
            --- quer ---
            pasta00/n0000.md	4.072235
            pasta04/n0044.md	3.994505
            pasta01/n0121.md	3.969250
            [... 17 linhas ...]

            --- tem ---
            pasta00/n0000.md	3.352347
            pasta04/n0044.md	3.248276
            pasta01/n0121.md	3.215007
            [... 17 linhas ...]

            Golden que muda exige explicacao escrita. NAO regenere para fazer passar.
FAIL
FAIL	github.com/jonyd/gobsidian/internal/service	1.623s
FAIL
----------------------------------------------------------------------
[OK] internal/search/bm25.go restaurado byte a byte (SHA-256 confere).

[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
EXIT=0
```

Saida completa em
`<scratchpad>/mut-step1.txt`. Antes do conserto do corpus esta mesma mutacao
deixava os seis `.tsv` intactos — era o defeito que a tarefa nomeia.

## Prova de mutacao — Step 2 (contexto vira `link.Raw`)

```
pwsh -File scripts/mutate.ps1 -Path internal/index/contexto_link.go `
  -Anchor 'Context: contextoDoLink(body, l),' -Replacement 'Context: l.Raw,' `
  -Test 'TestBacklinkTrazContexto|TestContextoNaoEstouraOOrcamento' -Package ./internal/index/
```

```
[...] Mutando internal/index/contexto_link.go
      - Context: contextoDoLink(body, l),
      + Context: l.Raw,

[...] go test -race -run TestBacklinkTrazContexto|TestContextoNaoEstouraOOrcamento ./internal/index/
----------------------------------------------------------------------
--- FAIL: TestBacklinkTrazContexto (0.01s)
    backlink_contexto_test.go:94: kind=wikilink: Context "[[Alvo]]" nao traz "acordao", que esta na linha do link e FORA dele — o recorte nao esta trazendo a vizinhanca
    backlink_contexto_test.go:94: kind=wikilink: Context "[[Alvo]]" nao traz "prescricao", que esta na linha do link e FORA dele — o recorte nao esta trazendo a vizinhanca
    backlink_contexto_test.go:94: kind=markdown: Context "Alvo.md" nao traz "resumo", que esta na linha do link e FORA dele — o recorte nao esta trazendo a vizinhanca
    backlink_contexto_test.go:94: kind=markdown: Context "Alvo.md" nao traz "historico", que esta na linha do link e FORA dele — o recorte nao esta trazendo a vizinhanca
--- FAIL: TestContextoNaoEstouraOOrcamento (0.01s)
    backlink_heading_test.go:106: contexto "[[Alvo]]" nao traz "esquerda", que esta colada ao link e FORA dele: o corte comeu a vizinhanca, ou o contexto virou o proprio link
    backlink_heading_test.go:106: contexto "[[Alvo]]" nao traz "direita", que esta colada ao link e FORA dele: o corte comeu a vizinhanca, ou o contexto virou o proprio link
FAIL
FAIL	github.com/jonyd/gobsidian/internal/index	0.992s
FAIL
----------------------------------------------------------------------
[OK] internal/index/contexto_link.go restaurado byte a byte (SHA-256 confere).

[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
EXIT=0
```

## Prova — Step 3 (ordem de mapa)

Nao ha regra de produto a mutar; o que se prova e que o **fixture** agora expoe a
ordem do mapa. Com a chamada a `ordenarBacklinks` removida das duas linhas de
`persist_test.go`, `-count=20`:

```
$ go test ./internal/index/ -run 'TestIndiceDeMetadadosRecarregadoEIdentico' -count=20
--- FAIL: TestIndiceDeMetadadosRecarregadoEIdentico (0.05s)
    persist_test.go:196: Backlinks(Origem.md) divergiu:
      fresco=[{From:Penal/Citante.md ...} {From:Civil/PONTO 03.md Anchor: ...} {From:Civil/PONTO 03.md Anchor:NaoExiste ...}],
      recarregado=[{From:Civil/PONTO 03.md Anchor: ...} {From:Civil/PONTO 03.md Anchor:NaoExiste ...} {From:Penal/Citante.md ...}]
    (as 20 execucoes reprovaram)
```

Com a ordenacao no lugar, `-count=20` da `ok`:

```
$ go test ./internal/index/ -run 'TestIndiceDeMetadadosRecarregadoEIdentico' -count=20
ok  	github.com/jonyd/gobsidian/internal/index	1.155s
```

Antes desta tarefa a mesma remocao **nao** reprovava: com uma origem so, nao
havia ordem para divergir.

## Prova de mutacao — Step 4 (escala de pesos de campo)

```
pwsh -File scripts/mutate.ps1 -Path internal/search/bm25.go `
  -Anchor 'return WeightHeadings' -Replacement 'return WeightBody' `
  -Test TestBM25PesoDeCorpoEOMenorDosTres -Package ./internal/search/
```

```
[...] Mutando internal/search/bm25.go
      - return WeightHeadings
      + return WeightBody

[...] go test -race -run TestBM25PesoDeCorpoEOMenorDosTres ./internal/search/
----------------------------------------------------------------------
--- FAIL: TestBM25PesoDeCorpoEOMenorDosTres (0.01s)
    bm25_test.go:173: heading=0.163205 nao ficou acima de corpo=0.163205: WeightHeadings nao esta separando de WeightBody
FAIL
FAIL	github.com/jonyd/gobsidian/internal/search	0.650s
FAIL
----------------------------------------------------------------------
[OK] internal/search/bm25.go restaurado byte a byte (SHA-256 confere).

[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
EXIT=0
```

```
pwsh -File scripts/mutate.ps1 -Path internal/search/bm25.go `
  -Anchor 'return WeightTitle' -Replacement 'return WeightBody' `
  -Test TestBM25PesoDeCorpoEOMenorDosTres -Package ./internal/search/
```

```
[...] Mutando internal/search/bm25.go
      - return WeightTitle
      + return WeightBody

[...] go test -race -run TestBM25PesoDeCorpoEOMenorDosTres ./internal/search/
----------------------------------------------------------------------
--- FAIL: TestBM25PesoDeCorpoEOMenorDosTres (0.01s)
    bm25_test.go:169: titulo=0.163205 nao ficou acima de heading=0.209835: WeightTitle nao esta separando de WeightHeadings
FAIL
FAIL	github.com/jonyd/gobsidian/internal/search	0.687s
FAIL
----------------------------------------------------------------------
[OK] internal/search/bm25.go restaurado byte a byte (SHA-256 confere).

[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
EXIT2=0
```

O teste antigo (`TestBM25WeightBody`, com `idx == nil`) **sobrevivia** as duas:
com indice nulo, `pesoDeCampo` devolve `WeightBody` antes de olhar campo algum,
e a mutacao nao alcanca o caminho exercitado.

## `verify.ps1` — completo, exit 0

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
     --- SKIP: TestNew_FailsOnUnwatchablePath (0.01s)
     --- SKIP: TestWriteAtomicPreservaOModoDoAlvo (0.00s)
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
EXIT=0
```

Os 6 pulados sao os mesmos de antes desta tarefa (etapa informativa, nao
reprova).

## `git diff --stat` do commit `d12f84f`

```
 internal/index/backlink_contexto_test.go           |  35 ++++-
 internal/index/backlink_heading_test.go            |  16 +-
 internal/index/classify_test.go                    |  33 +++--
 internal/index/persist_test.go                     |  36 ++++-
 internal/search/bm25_test.go                       |  54 +++++--
 internal/service/ranking_golden_test.go            | 165 +++++++++++++++++----
 internal/service/testdata/ranking/com-acento.tsv   |  40 ++---
 internal/service/testdata/ranking/dois-termos.tsv  |  40 ++---
 internal/service/testdata/ranking/frase-exata.tsv  |   2 +-
 .../service/testdata/ranking/so-em-heading.tsv     |  40 ++---
 internal/service/testdata/ranking/so-no-titulo.tsv |  12 +-
 internal/service/testdata/ranking/termo-amplo.tsv  | 100 ++++++-------
 12 files changed, 396 insertions(+), 177 deletions(-)
```

So `_test.go` e `testdata/`. Nenhum arquivo de produto no commit — as tres
mutacoes foram aplicadas e restauradas byte a byte por `mutate.ps1`, com
SHA-256 conferido em cada uma.

`internal/index/classify_test.go` esta no commit e nao no brief: foi onde a
funcao `ordenarBacklinks` ja morava. Criar uma segunda em `persist_test.go`
quebrou o build do pacote de teste (`ordenarBacklinks redeclared`), e a
alternativa certa era reforcar a existente, nao duplicar.


---

## Fix round 1 — correcao de uma afirmacao falsa deste relatorio e do commit `d12f84f`

**A frase estava errada, e foi escrita como se tivesse sido medida.** Este
relatorio dizia, na secao da prova de mutacao do Step 1:

> Antes do conserto do corpus esta mesma mutacao deixava os seis `.tsv` intactos.

A mensagem do commit `d12f84f` diz o equivalente ("Erasing the heading weight
changed no byte of any .tsv"), e as duas docstrings de
`internal/service/ranking_golden_test.go` (`:88` e a de `conferirDiscriminacao`)
diziam "nao mudava um byte de nenhum `.tsv`".

**Isso e falso.** O revisor reproduziu o corpus ANTIGO com `go test -overlay`,
recuperando o `ranking_golden_test.go` de `a313f32` contra um `bm25.go` com
`return WeightHeadings` trocado por `return WeightBody`: **dois dos seis `.tsv`
mudavam — `so-em-heading.tsv` e `dois-termos.tsv`, 40 linhas ao todo — e o
`TestRankingGolden` antigo REPROVAVA com a mutacao.** `dois-termos` entrava
porque o heading antigo era `## Execução fiscal` e "execucao" e um dos dois
termos daquela consulta.

O que era verdade, e continua sendo o achado da tarefa, e que a **ordem** nao
mudava: as 20 linhas saiam na mesmissima sequencia nas duas execucoes. O golden
antigo congelava o desempate por caminho, nao o ranking — mas nao era inerte ao
peso de campo. Eu nao medi essa frase antes de escreve-la em tres lugares
permanentes; foi inferida de "as 300 notas sao identicas, entao o peso e fator
comum", que explica a ordem e nao explica a coluna de score.

A mensagem de `d12f84f` nao pode ser reescrita sem reescrever o commit, e o
commit ja esta na `master`. **Este paragrafo e a correcao dela.** As duas
docstrings foram reescritas para o enunciado verdadeiro, e vao no commit desta
rodada.

### N1 — `:139-142`

"As demais notas trazem só a forma reduzida, no corpo" era falso: a forma
reduzida `prescricao` so entra quando `i%3 == 0`. Trocado por "As notas com
i%3 == 0 trazem a forma reduzida no corpo; é o contraste que `com-acento` mede."

### N2 — `:106-108`

Acrescentada a frase pedida: `WeightHeadings` e fator comum as notas que casam
`fiscal` (o termo ocorre exatamente uma vez, sempre num heading), entao a ORDEM
daquele golden e decidida pela normalizacao por comprimento e o peso aparece na
coluna de score. A saida da mutacao re-rodada abaixo confirma: `quer` e `tem`
trazem a mesma sequencia de 20 caminhos, so com scores diferentes.

### N3 — enchimento com fase deslocada

Aplicado o ruling do lead. `20 + (i*7)%60 + i/60` virou
`20 + ((i*7+29)%60) + i/60`. O motivo esta na docstring: sem o `+29`, o minimo
caia em `i == 0`, e `n0000` era ao mesmo tempo a nota unicamente mais curta e
membro de todas as classes de congruencia (`0 % 3 == 0 % 5 == 0 % 7 == 0 % 11`),
vencendo tres dos seis goldens por acumular tudo.

Com a fase deslocada o minimo cai em `i == 13` (`7*13 + 29 = 120 ≡ 0 mod 60`), e
`n0013` nao esta em classe de congruencia nenhuma (`13 % 3 == 1`, `13 % 5 == 3`,
`13 % 7 == 6`, `13 % 11 == 2`): a nota mais curta nao carrega termo especial
nenhum. `n0000` deixou de ser o primeiro de qualquer golden.

`head -3` dos seis `.tsv` depois do `-update`:

```
== com-acento.tsv (20 linhas)
pasta08/n0168.md	1.705403
pasta06/n0126.md	1.688467
pasta04/n0084.md	1.682203
== dois-termos.tsv (20 linhas)
pasta05/n0245.md	3.758047
pasta00/n0280.md	3.699517
pasta00/n0000.md	3.631440
== frase-exata.tsv (1 linhas)
pasta00/n0150.md	5.738904
== so-em-heading.tsv (20 linhas)
pasta06/n0176.md	4.059071
pasta02/n0022.md	4.032995
pasta03/n0253.md	4.032995
== so-no-titulo.tsv (6 linhas)
pasta05/n0005.md	7.070732
pasta05/n0015.md	6.864817
pasta05/n0025.md	6.758831
== termo-amplo.tsv (50 linhas)
pasta09/n0039.md	0.003200
pasta09/n0099.md	0.003189
pasta09/n0159.md	0.003189
```

`conferirDiscriminacao` continua passando nos seis subtestes (`go test
./internal/service/ -run TestRankingGolden` -> `ok ... 1.115s`).

### Mutacao do peso de heading, re-rodada sobre o corpus novo

```
[...] Mutando internal/search/bm25.go
      - 		return WeightHeadings
      + 		return WeightBody

[...] go test -race -run TestRankingGolden ./internal/service/
----------------------------------------------------------------------
--- FAIL: TestRankingGolden (0.92s)
    --- FAIL: TestRankingGolden/so-em-heading (0.00s)
        ranking_golden_test.go:319: ranking mudou.
            --- quer ---
            pasta06/n0176.md	4.059071
            pasta02/n0022.md	4.032995
            pasta03/n0253.md	4.032995
            pasta09/n0099.md	4.007253
            pasta00/n0220.md	3.931960
            pasta07/n0297.md	3.919686
            pasta03/n0143.md	3.907488
            pasta06/n0066.md	3.907488
            pasta03/n0033.md	3.824181
            (...)
            --- tem ---
            pasta06/n0176.md	3.334541
            pasta02/n0022.md	3.299491
            pasta03/n0253.md	3.299491
            pasta09/n0099.md	3.265170
            pasta00/n0220.md	3.166362
            pasta07/n0297.md	3.150473
            pasta03/n0143.md	3.134742
            pasta06/n0066.md	3.134742
            pasta03/n0033.md	3.028876
            (...)

            Golden que muda exige explicacao escrita. NAO regenere para fazer passar.
FAIL
FAIL	github.com/jonyd/gobsidian/internal/service	1.594s
FAIL
----------------------------------------------------------------------
[OK] internal/search/bm25.go restaurado byte a byte (SHA-256 confere).

[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
```

Exit 0 — a regra segue verificada. E a saida colada e ela propria a evidencia do
N2: as 20 linhas de `quer` e de `tem` estao na mesma sequencia; so a coluna de
score mudou.

### `verify.ps1` — fix round 1, exit 0

```
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

Os seis skips sao os mesmos de antes, todos preexistentes.
