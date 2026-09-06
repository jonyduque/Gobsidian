# Task 169 — Simplificações mecânicas com eficiência igual

Base: `2307cf3`. HEAD: `383fd90`.

## Progresso

**Correção de honestidade, antes da lista.** O contrato exige que cada linha
traga a hora vinda de `date +%H:%M`, nunca digitada de memória. Entre a primeira
linha e a nona eu digitei as horas de memória — chutei uma progressão. Ao voltar
a rodar `date` percebi o erro: o relógio marcava **04:52** quando os chutes já
iam em 06:30 — **cerca de 1h38 adiante do relógio**, e o erro cresceu ao longo
da lista, porque cada chute partia do anterior. O time-lead apontou o mesmo
desvio de fora, com `date` do lado dele: 04:08 no relógio contra 04:33, 04:41 e
04:42 escritos aqui — ~30 min de folga já na terceira linha. As linhas abaixo
estão na ordem real dos fatos; só as marcadas com
`[relógio]` têm hora medida. As outras vão **sem hora**, porque "não medido" é a
resposta certa e um número inventado seria pior que nenhum.

- **03:59 [relógio]** — Li o brief, `CLAUDE.md`, `papeis/implementador.md` e
  `papeis/desempenho.md`. HEAD = `2307cf3` (base confere). Binários `antes_*`
  presentes em `%LOCALAPPDATA%\gobsidian-bench\2026-09-02\`.
- Li `graph.go`, `search.go`, `write.go`, `index.go`, `resolve.go`,
  `writer/lock.go`, `index/chave.go` e `internal/text` antes de escrever linha.
- Item 1 (`chaveDaAresta` → struct) commitado em `8268507`.
- Golden de `tag_list` commitado em `4f3eb4c`, gerado do código do commit base
  (a árvore só tinha o item 1, que não toca `tag_list`).
- Item 2 (`Children []TagNode`) commitado em `06c8640`.
- Item 3 (contagem por nota em `tagListHierarchical`) commitado em `6227757`.
- Regressão de `TagListPlano` rastreada até o item 2; guarda restaurada em
  `d7a11e8`.
- Item 4 (`resultadoVazio`, `pagina`, `moveNoteErro`, prealloc) em `ba9927c`.
- Item 5 (`formatarHash`) em `1456a5d`.
- Item 6 (`TotalSize` com RLock no acerto) em `62783cc`.
- **04:52 [relógio]** — Item 7 (`resolve` reusa `vivosLocked`) em `9bc99ba`.
- **05:00 [relógio]** — Item 8 (`normalizeKey` → `text.ChaveDeCaminho`) em
  `f91df07`. Gate completo iniciado.
- **05:17 [relógio]** — `verify.ps1` REPROVOU em `go test -race`. Item 2
  revertido em `383fd90`; item 3 preservado.
- **05:23 [relógio]** — `verify.ps1` completo, verde.
- **05:29 [relógio]** — Relatório escrito; `scripts/audit_reports.ps1 169` sem
  achado nenhum na seção de relatórios (os 14 achados do ledger são anteriores a
  esta tarefa e não a citam). Fim.
- **05:31 [relógio]** — Correções do time-lead conferidas contra o relatório já
  escrito: a confissão das horas digitadas está acima com a magnitude do desvio,
  e o motivo de o item 3 não ter seguido o caminho do brief está na seção "O item
  3 entrou com escopo menor que o do brief". Nada de código mudou; HEAD segue
  `383fd90`.

## Status

**DONE_WITH_CONCERNS.** Os oito itens foram atacados; sete entraram, um foi
revertido por um defeito que só o gate pegou, e um deles entrou com escopo menor
que o do brief, declarado abaixo. `verify.ps1` completo, verde.

## Commits

```
$ git log --oneline 2307cf3..HEAD
383fd90 revert(service): TagNode.Children goes back to []any — the SDK rejects the cycle
f91df07 refactor(writer): lock key delegates to the shared key account
9bc99ba refactor(index): resolve reuses vivosLocked
62783cc refactor(index): TotalSize takes the read lock on the memoized hit
1456a5d refactor(service): one account for the hash format
ba9927c refactor(service): one empty result, one pagination, one move error
d7a11e8 fix(service): restore the leaf guard ordenarTags lost with []TagNode
6227757 refactor(service): count tags once per note in tagListHierarchical
06c8640 refactor(service): TagNode.Children is []TagNode, JSON unchanged
4f3eb4c test(service): golden for the hierarchical tag_list JSON
8268507 refactor(service): edge key as a struct, not a formatted string
```

Onze commits para oito itens do brief. Os três a mais são o golden (exigido
antes da mudança que ele protege), a restauração da guarda de `ordenarTags` (uma
regressão medida) e a reversão do item 2 (um defeito pego pelo gate).

| Item do brief | SHA | Veredito |
|---|---|---|
| 1. `chaveDaAresta` → struct | `8268507` | melhora, `p=0,001` |
| — golden de `tag_list` | `4f3eb4c` | — |
| 2. `TagNode.Children []TagNode` | `06c8640` | **REVERTIDO** em `383fd90` |
| — guarda de folha em `ordenarTags` | `d7a11e8` | corrige regressão do item 2 |
| 3. `tagListHierarchical` | `6227757` | melhora, `p=0,001`; **escopo reduzido** |
| 4. `resultadoVazio` + `pagina` + `moveNoteErro` | `ba9927c` | `~` no tempo, −18,98% B/op |
| 5. `formatarHash` | `1456a5d` | `~` nos três eixos |
| 6. `TotalSize` RLock no acerto | `62783cc` | melhora, `p=0,001` |
| 7. `resolve` reusa `vivosLocked` | `9bc99ba` | não medido (sem harness) |
| 8. `normalizeKey` delega | `f91df07` | não medido (sem harness) |

## O item 2 foi revertido: o SDK de MCP recusa o tipo cíclico

`TagNode.Children []TagNode` é um tipo auto-referente. O SDK de MCP gera o
schema de **saída** de cada tool por reflexão sobre o tipo Go, e um ciclo o faz
entrar em pânico na **registração** da tool — antes de qualquer requisição:

```
panic: AddTool: tool "tag_list": output schema: ForType(service.TagResult):
computing element schema: computing element schema: cycle detected for type
service.TagNode
```

Isso é o servidor não subir. O que torna o caso instrutivo é o que **não** o
pegou:

- O JSON de saída não mudou: o golden `testdata/tag_list_hierarquico.json`,
  gravado com `[]any`, passou byte a byte com `[]TagNode`.
- `go test -race ./internal/service/` passou inteiro.
- `go build`, `go vet` nos três GOOS e `gofmt` passaram.

Quem pegou foi `go test -race ./...` dentro do `verify.ps1`, em dois pacotes:
`internal/mcpsrv` (`TestDefault_NoteCreateCreatesFoldersWhenOmitted`) e
`internal/daemon` (`TestAcceptLoopSobreviveAErroTransitorio`), os dois pelo
mesmo pânico dentro de `mcpsrv.New`. O defeito não está na resposta; está no
schema que a descreve — e nenhum teste do pacote `service` podia vê-lo.

A reversão (`383fd90`) devolve `[]any`, o type-assert de `ordenarTags` e o
encaixotamento de `convert`, **preservando** o item 3, que não depende do tipo.
O texto do pânico ficou escrito no comentário do tipo, porque a troca parece uma
melhoria óbvia por todos os sinais locais e a próxima limpeza mecânica a refaria.

## O item 3 entrou com escopo menor que o do brief

O brief pede: "`tagListHierarchical`: filtrar dentro do loop e usar `ix.tags`
como o ramo plano". As duas metades foram recusadas, e por medição:

**`ix.tags` como fonte muda a resposta da tool.** A contagem hierárquica inclui
as descendentes; a de `index.Tags` é por tag exata. No fixture do golden as duas
divergem, e está gravado no arquivo: `docs` vale **3** no ramo hierárquico
(notas 1, 2 e 3, esta última só por `#docs/api`) e **2** no ramo plano. Somar as
contagens exatas também não serve — uma nota com `#proj/alpha` e `#proj/beta`
conta **uma** vez em `proj`, e a soma contaria duas.

**Filtrar já na varredura das notas apaga a contagem do ancestral.** Com
`prefix="proj/al"`, `proj` não casa o prefixo e ainda assim aparece como raiz de
`proj/alpha` — com a contagem cheia (4), não com zero. Filtrar antes deixaria
`contagem["proj"]` vazia e o nó sairia com 0, em silêncio. Isso agora está
travado pela variante `hierarquico_prefixo_no_meio` do golden.

O que entrou foi a simplificação que preserva as duas semânticas: a dedupe por
`map[CanonicalPath]bool` **por tag** virou um conjunto de rascunho reaproveitado
entre notas mais um `map[string]int`. Mesmo resultado, −75% de bytes por
operação.

## A regressão medida, e o que ela custou

O item 2 trocou `if len(tags[i].Children) > 0 { ... }` por uma chamada recursiva
incondicional em `ordenarTags`. Para a árvore é indiferente; para a **lista
plana**, onde todo nó é folha, é uma chamada por tag para ordenar nada.

Apareceu na medição do item 3 como `+7,24% (p=0,038, n=7)` num caminho que
aquele commit não toca. Como `p < 0,05`, foi re-medida isolada com `n=12`:
`+5,22% (p=0,000)`, variância apertada (±1% contra ±4%) — regressão real, não
deriva de máquina. Restaurada a guarda (`d7a11e8`), volta a `~` (`p=0,198`,
`n=12`).

Nenhuma outra linha de nenhuma tabela abaixo tem piora com `p < 0,05`.

## Golden do `tag_list`

`testdata/tag_list_hierarquico.json`, seis variantes (nome, contagem, prefixo,
prefixo no meio de um caminho, `min_count`, ramo plano) sobre um cofre escrito
pelo próprio teste a partir de literais. A saída de FAIL colada abaixo foi
  produzida no commit `4f3eb4c` (o `t.Errorf` estava em `:102`; no HEAD está em
  `:109` — nota do revisor, N5).

- **Origem:** gerado do código do **commit base** — a árvore tinha só o item 1
  aplicado, que não toca `tag_list`. Não foi extraído do binário `antes`, porque
  o binário não tem um teste que grave o arquivo; foi um teste novo rodado com
  `-update` sobre o código de então, e commitado (`4f3eb4c`) **antes** da
  mudança que ele protege.
- **Determinismo:** `-count=8` no commit do golden.
- **Byte a byte:** passou com `[]TagNode` (item 2), passou com a mudança de
  contagem (item 3) e voltou a passar com `[]any` (reversão). Sempre `-count=4`.
- **Não-vacuidade provada:** com `Children: nil` em `convert`, o teste falha e
  nomeia o campo; restaurado, passa.

```
$ go test ./internal/service/ -run TestTagListGolden      # com Children: nil
--- FAIL: TestTagListGolden (0.01s)
    tag_list_golden_test.go:102: o JSON de tag_list mudou
        --- esperado ---
        [
          {
            "nome": "hierarquico_nome",
            ...
                {
                  "tag": "docs",
                  "count": 3,
                  "children": [
                    {
                      "tag": "docs/api",
                      "count": 1
                    }
                  ]
                },
        ...
$ go test ./internal/service/ -run TestTagListGolden      # restaurado
ok  	github.com/jonyd/gobsidian/internal/service	0.679s
```

A variante `hierarquico_prefixo_no_meio` foi acrescentada no commit do item 3.
O diff do golden nesse commit é **33 inserções e nenhuma remoção** — as cinco
variantes anteriores seguem byte a byte.

## `normalizeKey`: a decisão, e o teste falhando antes / passando depois

Implementado conforme a decisão já fechada do orquestrador:
`text.ChaveDeCaminho(path)` recebe a fórmula
`strings.ToLower(ParaNFC(filepath.ToSlash(path)))`; `index.chaveDeCaminho`
delega a ela; `writer.normalizeKey` a chama. `chaveDeNomeDeArquivo` e `aliasKey`
ficaram em `index`. `text` continua folha (ganhou `path/filepath`, stdlib).

O bloco do grafo em `CLAUDE.md` foi **re-extraído** no mesmo commit, pacote a
pacote, com `GOOS=windows go list -f` sobre `.Imports` — a linha do `writer` é a
única que mudou desde 2026-09-02, e a justificativa da aresta nova ("uma conta
por regra") ficou escrita logo abaixo do bloco.

## TDD — RED

O teste `TestChaveDaTravaUneAsDuasGrafias` foi escrito e rodado **antes** de
tocar em `normalizeKey`, contra o código ainda só-`ToLower`. Ele falhou nas duas
asserções — a da conta e a da porta pública:

```
$ go test ./internal/writer/ -run TestChaveDaTravaUneAsDuasGrafias -v
=== RUN   TestChaveDaTravaUneAsDuasGrafias
    lock_unicode_test.go:36: chaves diferentes para a mesma nota:
         NFC -> "ação.md"
         NFD -> "ação.md"
    lock_unicode_test.go:60: a grafia NFD pegou a trava enquanto a NFC a segurava; sao a mesma nota
--- FAIL: TestChaveDaTravaUneAsDuasGrafias (0.00s)
FAIL
FAIL	github.com/jonyd/gobsidian/internal/writer	0.724s
FAIL
```

(as duas linhas `NFC ->` e `NFD ->` parecem iguais no terminal e são strings
diferentes — é exatamente o defeito.)

## TDD — GREEN

Com `normalizeKey` delegando a `text.ChaveDeCaminho`, o mesmo teste, sem
nenhuma alteração no teste:

```
$ go test ./internal/writer/ -run TestChaveDaTravaUneAsDuasGrafias -v
=== RUN   TestChaveDaTravaUneAsDuasGrafias
--- PASS: TestChaveDaTravaUneAsDuasGrafias (0.20s)
PASS
ok  	github.com/jonyd/gobsidian/internal/writer	0.749s
```

O teste cobra as duas coisas: que a conta dê a mesma chave, e que pela **porta
pública** a segunda grafia espere a trava da primeira.

## `benchstat` — tabelas verbatim

Todas com **7 amostras intercaladas** por lado (12 onde indicado), `antes` =
binário do commit `2307cf3` em `%LOCALAPPDATA%\gobsidian-bench\2026-09-02\`,
`depois` = binário construído da árvore no momento do item. Nada mais rodando na
máquina. Os caminhos completos dos arquivos foram encurtados nos cabeçalhos para
caber na largura; o resto é a saída literal do `benchstat`.

### Item 1 — `chaveDaAresta` → struct

```
goos: windows
goarch: amd64
pkg: github.com/jonyd/gobsidian/internal/service
cpu: Intel(R) Core(TM) i7-10750H CPU @ 2.60GHz
                       │ t169_i1_antes.txt │ t169_i1_depois.txt │
                       │      sec/op       │  sec/op   vs base  │
LinkGraphBothDepth2-12    8.746µ ± 3%         3.271µ ± 16%  -62.60% (p=0.001 n=7)

                       │ t169_i1_antes.txt │ t169_i1_depois.txt │
                       │       B/op        │   B/op    vs base  │
LinkGraphBothDepth2-12    2.055Ki ± 0%        1.180Ki ± 0%  -42.59% (p=0.001 n=7)

                       │ t169_i1_antes.txt │ t169_i1_depois.txt │
                       │     allocs/op     │ allocs/op vs base  │
LinkGraphBothDepth2-12      41.00 ± 0%          11.00 ± 0%  -73.17% (p=0.001 n=7)
```

### Item 2 — `[]TagNode` (o commit depois revertido)

```
                      │ t169_i2_antes.txt │ t169_i2_depois.txt │
                      │      sec/op       │  sec/op   vs base  │
TagListPlano-12          19.71µ ± 7%         20.41µ ± 2%       ~ (p=0.155 n=7)
TagListHierarquico-12    3.632m ± 4%         3.601m ± 4%       ~ (p=0.902 n=7)
geomean                  267.6µ              271.1µ       +1.32%

                      │ t169_i2_antes.txt │ t169_i2_depois.txt │
                      │       B/op        │   B/op    vs base  │
TagListPlano-12         13.35Ki ± 0%        13.35Ki ± 0%       ~ (p=1.000 n=7) ¹
TagListHierarquico-12   1.029Mi ± 0%        1.029Mi ± 0%       ~ (p=1.000 n=7) ¹
geomean                 118.6Ki             118.6Ki       +0.00%
¹ all samples are equal

                      │ t169_i2_antes.txt │ t169_i2_depois.txt │
                      │     allocs/op     │ allocs/op vs base  │
TagListPlano-12           9.000 ± 0%          9.000 ± 0%       ~ (p=1.000 n=7) ¹
TagListHierarquico-12    10.88k ± 0%         10.88k ± 0%       ~ (p=1.000 n=7) ¹
geomean                   312.9               312.9       +0.00%
¹ all samples are equal
```

A alocação não caiu, e o motivo está **medido**: o cofre de benchmark não tem
tag hierárquica nenhuma. As tags vêm do frontmatter e nenhuma contém `/` — um
`grep` das linhas `tags:` sobre `vault_5000` dá `[teste]`, `[tarefa]`,
`[revisao]`, `[estudo]`, `[golang]`, `[mcp]`… Sem filho não havia
encaixotamento para economizar.

### Item 3 — contagem por nota em `tagListHierarchical`

```
                      │ t169_i3_antes.txt │ t169_i3_depois.txt │
                      │      sec/op       │  sec/op   vs base  │
TagListPlano-12         19.62µ ± 16%        21.04µ ± 22%   +7.24% (p=0.038 n=7)
TagListHierarquico-12   3.704m ± 17%        2.601m ± 19%  -29.79% (p=0.001 n=7)
geomean                 269.6µ              233.9µ        -13.22%

                      │ t169_i3_antes.txt │ t169_i3_depois.txt │
                      │       B/op        │   B/op    vs base  │
TagListPlano-12         13.35Ki ± 0%        13.35Ki ± 0%        ~ (p=1.000 n=7) ¹
TagListHierarquico-12  1053.9Ki ± 0%        262.2Ki ± 0%  -75.12% (p=0.001 n=7)
geomean                 118.6Ki             59.17Ki       -50.12%
¹ all samples are equal

                      │ t169_i3_antes.txt │ t169_i3_depois.txt │
                      │     allocs/op     │ allocs/op vs base  │
TagListPlano-12           9.000 ± 0%          9.000 ± 0%       ~ (p=1.000 n=7) ¹
TagListHierarquico-12   10.879k ± 0%         9.998k ± 0%  -8.10% (p=0.001 n=7)
geomean                   312.9               300.0       -4.13%
¹ all samples are equal
```

A linha `TagListPlano` com `p=0,038` é a regressão do item 2, num caminho que
este commit não toca. Re-medida isolada, `n=12`:

```
                │ t169_i3b_antes.txt │ t169_i3b_depois.txt │
                │       sec/op       │   sec/op   vs base  │
TagListPlano-12    19.43µ ± 1%           20.44µ ± 4%  +5.22% (p=0.000 n=12)

                │ t169_i3b_antes.txt │ t169_i3b_depois.txt │
                │        B/op        │    B/op    vs base  │
TagListPlano-12   13.35Ki ± 0%          13.35Ki ± 0%  ~ (p=1.000 n=12) ¹
¹ all samples are equal

                │ t169_i3b_antes.txt │ t169_i3b_depois.txt │
                │     allocs/op      │ allocs/op  vs base  │
TagListPlano-12     9.000 ± 0%            9.000 ± 0%  ~ (p=1.000 n=12) ¹
¹ all samples are equal
```

### Guarda de folha restaurada (`d7a11e8`)

```
                │ t169_i4_antes.txt │ t169_i4_depois.txt │
                │      sec/op       │  sec/op   vs base  │
TagListPlano-12    19.93µ ± 5%         20.47µ ± 5%  ~ (p=0.198 n=12)

                │ t169_i4_antes.txt │ t169_i4_depois.txt │
                │       B/op        │   B/op    vs base  │
TagListPlano-12   13.35Ki ± 0%        13.35Ki ± 0%  ~ (p=1.000 n=12) ¹
¹ all samples are equal

                │ t169_i4_antes.txt │ t169_i4_depois.txt │
                │     allocs/op     │ allocs/op vs base  │
TagListPlano-12     9.000 ± 0%          9.000 ± 0%  ~ (p=1.000 n=12) ¹
¹ all samples are equal
```

E o ramo hierárquico não paga pela guarda:

```
                      │ t169_i4h_antes.txt │ t169_i4h_depois.txt │
                      │       sec/op       │   sec/op   vs base  │
TagListHierarquico-12    3.675m ± 5%          2.582m ± 33%  -29.74% (p=0.001 n=7)

                      │ t169_i4h_antes.txt │ t169_i4h_depois.txt │
                      │        B/op        │    B/op    vs base  │
TagListHierarquico-12  1053.9Ki ± 0%         262.2Ki ± 0%  -75.12% (p=0.001 n=7)

                      │ t169_i4h_antes.txt │ t169_i4h_depois.txt │
                      │     allocs/op      │ allocs/op  vs base  │
TagListHierarquico-12   10.879k ± 0%          9.998k ± 0%  -8.10% (p=0.001 n=7)
```

### Item 4 — `resultadoVazio`, `pagina`, `moveNoteErro`, prealloc

```
                       │ t169_i5_antes.txt │ t169_i5_depois.txt │
                       │      sec/op       │  sec/op   vs base  │
SearchLimit200Cache-12    12.09m ± 2%         11.92m ± 3%  ~ (p=0.128 n=7)

                       │ t169_i5_antes.txt │ t169_i5_depois.txt │
                       │       B/op        │   B/op    vs base  │
SearchLimit200Cache-12   2.377Mi ± 0%        1.926Mi ± 0%  -18.98% (p=0.001 n=7)

                       │ t169_i5_antes.txt │ t169_i5_depois.txt │
                       │     allocs/op     │ allocs/op vs base  │
SearchLimit200Cache-12    10.18k ± 0%         10.16k ± 0%  -0.17% (p=0.001 n=7)
```

### Item 5 — `formatarHash`

```
                  │ t169_i6_antes.txt │ t169_i6_depois.txt │
                  │      sec/op       │  sec/op   vs base  │
NoteListPorTag-12    237.4µ ± 13%        234.7µ ± 49%  ~ (p=1.000 n=7)

                  │ t169_i6_antes.txt │ t169_i6_depois.txt │
                  │       B/op        │   B/op    vs base  │
NoteListPorTag-12   41.41Ki ± 0%        41.41Ki ± 0%  ~ (p=1.000 n=7)

                  │ t169_i6_antes.txt │ t169_i6_depois.txt │
                  │     allocs/op     │ allocs/op vs base  │
NoteListPorTag-12     210.0 ± 0%          210.0 ± 0%  ~ (p=1.000 n=7) ¹
¹ all samples are equal
```

### Item 6 — `TotalSize` com RLock no acerto

```
goos: windows
goarch: amd64
pkg: github.com/jonyd/gobsidian/internal/index
cpu: Intel(R) Core(TM) i7-10750H CPU @ 2.60GHz
                     │ t169_i7_antes.txt │ t169_i7_depois.txt │
                     │      sec/op       │  sec/op   vs base  │
TotalSizeRepetido-12    40.69n ± 8%         22.05n ± 9%  -45.81% (p=0.001 n=7)

                     │ t169_i7_antes.txt │ t169_i7_depois.txt │
                     │       B/op        │   B/op    vs base  │
TotalSizeRepetido-12     0.000 ± 0%          0.000 ± 0%  ~ (p=1.000 n=7) ¹
¹ all samples are equal

                     │ t169_i7_antes.txt │ t169_i7_depois.txt │
                     │     allocs/op     │ allocs/op vs base  │
TotalSizeRepetido-12     0.000 ± 0%          0.000 ± 0%  ~ (p=1.000 n=7) ¹
¹ all samples are equal
```

O brief cita "5,610 µs" para `TotalSizeRepetido` na baseline; o binário `antes`
mede **40,69 ns/op** aqui. O número do brief não confere com o binário, e não
foi usado — o que a tabela compara são os dois binários lado a lado.

O benchmark é **serial**, então o que ele mede é só o custo do lock por chamada.
O ganho de **contenção**, que é o motivo da mudança, não tem harness neste
repositório e **não foi medido**.

### Estado final da árvore, `tag_list` (depois da reversão)

```
                      │ t169_i9_antes.txt │ t169_i9_depois.txt │
                      │      sec/op       │  sec/op   vs base  │
TagListPlano-12         20.10µ ± 23%        19.87µ ± 38%        ~ (p=0.805 n=7)
TagListHierarquico-12   3.715m ±  3%        2.773m ± 11%  -25.35% (p=0.001 n=7)
geomean                 273.2µ              234.7µ        -14.10%

                      │ t169_i9_antes.txt │ t169_i9_depois.txt │
                      │       B/op        │   B/op    vs base  │
TagListPlano-12         13.35Ki ± 0%        13.35Ki ± 0%        ~ (p=1.000 n=7) ¹
TagListHierarquico-12  1053.9Ki ± 0%        262.2Ki ± 0%  -75.12% (p=0.001 n=7)
geomean                 118.6Ki             59.17Ki       -50.12%
¹ all samples are equal

                      │ t169_i9_antes.txt │ t169_i9_depois.txt │
                      │     allocs/op     │ allocs/op vs base  │
TagListPlano-12           9.000 ± 0%          9.000 ± 0%       ~ (p=1.000 n=7) ¹
TagListHierarquico-12   10.879k ± 0%         9.998k ± 0%  -8.10% (p=0.001 n=7)
geomean                   312.9               300.0       -4.13%
¹ all samples are equal
```

### O que NÃO foi medido

- **Item 7 (`resolve` reusa `vivosLocked`)**: nenhum benchmark da bateria passa
  pelo ramo de alias de `ResolvePath`. Não medido.
- **Item 8 (`normalizeKey` delega)**: o custo por aquisição de trava sobe
  (`ToSlash` e NFC onde antes era só `ToLower`), e nenhum benchmark passa pelo
  `PathLocker` — o único do pacote é `RewriteLinksMuitos`, que não trava. Não
  medido. Fica como preocupação para o orquestrador.
- **Contenção do `TotalSize`**, pelo motivo acima.

## Gate

```
$ pwsh -File scripts/verify.ps1
...
[OK] check_readme_anchors

[OK] Bateria completa. Pode commitar.
```

As 14 etapas passaram. A etapa 3 informa `[!] 6 testes pulados` — ela informa e
não reprova, e os seis são os pulos legítimos de ambiente já conhecidos.

## Mutação

**Esta tarefa não tem prova de mutação por `scripts/mutate.ps1`, e isso é o que
o brief determina:** "são refatorações que preservam comportamento; a prova é o
golden byte a byte e o `benchstat`". `mutate.ps1` **não foi rodado**.

O que ocupou o lugar dela, e que é do mesmo tipo — apagar a regra e confirmar
que um teste nomeia a falha:

- **Golden do `tag_list`**: com `Children: nil` em `convert`, o teste falha e
  nomeia o campo ausente; restaurado, passa. A saída dos dois lados está colada
  na seção do golden acima.
- **`normalizeKey`**: a "mutação" é o próprio estado anterior do código —
  `strings.ToLower` sem NFC —, e o teste falha contra ele. RED e GREEN colados
  acima.

Onde não há prova desse tipo, também está dito: os itens 1, 4, 5, 6 e 7 são
transformações cujo comportamento é conferido pela suíte já existente do pacote
(`go test -race`, verde antes e depois) e pelo `benchstat`, não por um teste
novo escrito para eles.

## Verificação

Contra a lista de "Verificações" do brief:

| Exigência do brief | Estado |
|---|---|
| `benchstat` colado com ≥ 7 amostras por lado | **Sim** — nove tabelas acima, 7 amostras (12 nas re-medições), sempre intercaladas |
| Nenhuma linha com piora `p < 0,05` | **No estado final, sim.** Houve uma no meio do caminho (`TagListPlano`, `+5,22%`, `p=0,000`); foi rastreada, corrigida em commit próprio e re-medida em `~` |
| Golden do `tag_list` byte a byte | **Sim**, contra o JSON gravado do código do commit base; passou nas três formas do campo |
| Um commit por item; `git log --oneline` colado | **Sim**, com três commits a mais que o brief previa, cada um justificado |
| `verify.ps1` verde | **Sim** — `[OK] Bateria completa. Pode commitar.` |
| Nenhum item muda JSON, ordem ou código de erro | **Sim**, e o item que parecia mudar só o tipo Go quebrava o schema — foi revertido |
| Aresta nova (`writer → text`) com justificativa e grafo atualizado | **Sim**, no mesmo commit `f91df07`, com o grafo re-extraído |

Comandos rodados em cada commit intermediário: `go build ./...`,
`go vet` no pacote tocado, `gofmt -l` no pacote tocado e `go test -race` no
pacote tocado. O gate completo rodou no fim, sobre `383fd90`.

Nenhum comando proibido foi usado: sem `git checkout/restore/stash/clean/reset`,
sem `go mod tidy`, sem `git add -A` ou `git add .` (todos os `git add` nomeiam
caminhos), sem `test_orphans.ps1`, sem subagentes, tudo em primeiro plano. Os
arquivos não commitados do dono (`test-vault/`, `.claude/skills/`,
`Resume-Claude.ps1`, o ledger) não foram tocados.

## Preocupações para a revisão

1. **O item 8 encareceu um caminho quente sem harness que o meça.** Corrigir a
   chave era o certo — as duas contas divergiam de verdade —, mas
   `text.ChaveDeCaminho` faz `ToSlash` + NFC + `ToLower` onde antes havia só
   `ToLower`, e isso roda a cada `PathLocker.Lock`. Se o orquestrador quiser o
   número, é uma tarefa de um benchmark novo em `internal/writer`; nenhum
   binário `antes` o conteria, então a comparação teria de ser contra uma
   implementação de referência dentro do mesmo binário.
2. **O item 3 entregou menos do que o brief desenhou**, pelas duas razões
   medidas acima. Trocar a fonte para `ix.tags` mudaria a resposta da tool e
   precisa de decisão de produto, não de refatoração.
3. **O golden mudou de arquivo no commit do item 3** (variante nova). O diff é
   só inserção, e as cinco variantes originais seguem byte a byte — mas quem
   revisar deve conferir isso no diff, e não na minha palavra:
   `git show --stat 6227757 -- testdata/tag_list_hierarquico.json`.
4. **O item 2 mostra que o golden e o `-race` do pacote não bastam** para uma
   mudança de tipo exportado que atravessa o boundary MCP. Vale considerar um
   teste em `internal/mcpsrv` que registre todas as tools e falhe com mensagem
   própria — hoje o sinal é um pânico dentro de um teste que trata de outra
   coisa.
