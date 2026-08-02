# Relatório Task 73: `bench.yml` no CI com detecção de regressão

**Data:** 2026-08-02
**Base:** `f068f7a`
**Commits:** `4304bea` (workflow, comparador, benchmarks), `580c0aa` (primeira
referência), `f9beed8` (referência refeita sobre mediana de três rodadas).

---

## O que foi entregue

- `.github/workflows/bench.yml` — gera o cofre determinístico, roda os
  benchmarks, compara com a referência versionada.
- `scripts/bench_compare.ps1` — o comparador. Reprova regressão acima de 20%,
  avisa em melhora acima de 20%, e reprova benchmark que está na referência e
  não apareceu na saída.
- `internal/service/bench_test.go` — seis benchmarks. **Não existia benchmark
  nenhum no repositório**; um workflow de bench sem benchmark é um gate que não
  pode falhar, então eles fazem parte da tarefa.
- `docs/bench-baseline.json` — a referência, com runner, commit e nota de como
  foi obtida.

## As duas decisões fechadas, e onde elas aparecem no código

**A referência é arquivo versionado, não a execução anterior.** Está em
`docs/bench-baseline.json` e só muda por `-UpdateBaseline` + commit. O
comparador nunca escreve sozinho durante a comparação. O motivo está no cabeçalho
do script: 5% por semana nunca dispara um gate de 20%, e em dez semanas dobrou.

**O número do CI não substitui a medição local.** O JSON nomeia o runner
(`ubuntu-latest, 2 vCPU, GOMAXPROCS=2`) e o script imprime, ao gravar, que isto
não substitui `docs/OPERACAO.md`. Os números das duas fontes não são
comparáveis: o runner tem 2 núcleos, a máquina de medição tem 12.

## Comparação por benchmark, nunca agregada

Cada benchmark tem sua linha e seu delta. A saída real de uma rodada limpa:

```
    benchmark                           referencia        medido     delta
    BenchmarkIndexBuild                244,692,285   241,830,882     -1.2%
    BenchmarkInvertedLoad               89,720,000    86,476,008     -3.6%
    BenchmarkSearchTermoAmplo          118,947,918   107,175,360     -9.9%
    BenchmarkSearchDoisTermos           21,850,238    18,770,056    -14.1%
    BenchmarkSearchFraseExata           78,001,305    71,275,452     -8.6%
[OK] nenhum benchmark regrediu acima de 20%.
```

## Prova de que o gate dispara — no CI, não só localmente

`scripts/mutate.ps1` não serve aqui: o alvo não é teste Go. A prova é a injeção
descrita no brief. Injetei `time.Sleep(120 * time.Millisecond)` em
`service.Search`, num branch `bench-gate-proof`, e deixei o workflow rodar.

**Run 30762162779, conclusão `failure`:**

```
[i] referencia: docs/bench-baseline.json (runner: ubuntu-latest (GitHub-hosted, 2 vCPU), go 1.25, GOMAXPROCS=2), tolerancia +20%.
    benchmark                           referencia        medido     delta
    BenchmarkIndexBuild                244,692,285   149,012,656    -39.1%
    BenchmarkInvertedLoad               89,720,000    62,148,876    -30.7%
    BenchmarkSearchTermoAmplo          118,947,918   204,002,994    +71.5%
    BenchmarkSearchDoisTermos           21,850,238   135,800,055   +521.5%
    BenchmarkSearchFraseExata           78,001,305   176,261,663   +126.0%

    Melhora acima da tolerancia nao reprova, mas merece um olhar:

WARNING: [!] 3 regressao(oes) acima de 20%:
    BenchmarkSearchTermoAmplo: 118,947,918 -> 204,002,994 ns/op (+71.5%, acima da tolerancia de 20%)
    BenchmarkSearchDoisTermos: 21,850,238 -> 135,800,055 ns/op (+521.5%, acima da tolerancia de 20%)
    BenchmarkSearchFraseExata: 78,001,305 -> 176,261,663 ns/op (+126.0%, acima da tolerancia de 20%)
```

A mesma rodada exercitou o caminho de **aviso de melhora**: `IndexBuild` e
`InvertedLoad` não passam por `Search`, saíram mais rápidos naquela instância de
runner, e o comparador avisou sem reprovar. Branch e worktree removidos depois.

Localmente, a mesma injeção com 50 ms, contra uma referência local:

```
    BenchmarkSearchDoisTermos           28.931.800    72.356.533   +150,1%
    BenchmarkSearchLimit200            192.855.267   261.560.067    +35,6%
GATE_EXIT=1
```

E a reprovação por benchmark ausente, que é o modo mais barato de ter um gate
que nunca falha:

```
    BenchmarkSearchLimit200            192.855.267       AUSENTE         -
WARNING: [!] 1 regressao(oes) acima de 20%:
    BenchmarkSearchLimit200 esta na referencia e NAO foi medido
GATE_EXIT=1
```

## Prova de que ele não dispara à toa — e o que ela achou

Duas rodadas limpas contra a referência final, sem mudança de código:
**30761791215** (push) e **30761852416** (workflow_dispatch), ambas `success`.

**A primeira tentativa não passou nesse teste, e isso mudou a entrega.** Com uma
referência de amostra única, duas rodadas limpas deram:

| rodada | `BenchmarkSearchFraseExata` |
|---|---|
| 30761611392 | **+19,3%** |
| 30761677179 | **+18,8%** |

Contra um gate de 20%. Um gate a 0,7 ponto de reprovar numa árvore inalterada é
um gate que alguém desliga. A causa era a amostra, não a tolerância: a primeira
rodada mediu `FraseExata` em 65,6 ms e as duas seguintes em 78,3 e 78,0 ms.

A referência passou a ser a **mediana de três rodadas**. Contra ela, o pior
desvio positivo entre os seis benchmarks caiu para **+9,6%**, cerca de metade do
gate. `bench_compare.ps1` ganhou `-Nota`, e o JSON registra como o número foi
obtido — "mediana de três" e "uma amostra" se comportam de forma diferente num
runner compartilhado, e a diferença não é visível nos números.

**Variância medida no runner, três rodadas limpas:**

| benchmark | mín | mediana | máx |
|---|---|---|---|
| `IndexBuild` | 188.779.853 | 244.692.285 | 254.752.121 |
| `InvertedLoad` | 75.187.691 | 89.720.000 | 93.821.832 |
| `SearchTermoAmplo` | 107.150.484 | 118.947.918 | 130.371.924 |
| `SearchDoisTermos` | 20.896.127 | 21.850.238 | 22.848.446 |
| `SearchFraseExata` | 65.631.449 | 78.001.305 | 78.304.583 |
| `SearchLimit200` | 359.119.565 | 373.350.430 | 375.080.084 |

## `golangci-lint` segue fixado

`.github/workflows/ci.yml` mantém `golangci/golangci-lint-action@v8` com
`version: v2.12.2` nos dois jobs (`lint` e `lint-windows`). Não foi tocado.
`verify.ps1` confere a versão antes de confiar num zero local.

## Guardas contra benchmark que para de medir

- Cada benchmark de busca afirma um mínimo de resultados. Consulta que deixa de
  casar mede o caminho vazio e ficaria dramaticamente mais rápida — apareceria
  como melhora de 90% e o gate só avisaria.
- `benchServico` recusa corpus com menos de 5.000 notas. Cofre truncado produz
  benchmark rápido e verde.
- Benchmark que pula (cofre ausente) não emite linha `ns/op`, e o comparador
  trata ausência como reprovação.

## Bateria

```
[OK] Bateria completa. Pode commitar.
VERIFY_EXIT=0
```

CI completo verde em `30761531221`: `fmt`, `lint`, `lint-windows`, `netcheck`,
`orphans`, e `test` nos três sistemas.

## O que ficou de fora, e um achado fora do escopo

**Fora do escopo, corrigido antes porque bloqueava esta tarefa:** o CI estava
**vermelho desde 2026-07-28**, em todos os pushes. Um gate novo num CI vermelho
é um gate que ninguém lê, e a exigência "confirme verde nas duas" não seria
distinguível do vermelho existente. Foram cinco defeitos, em `45ac834` e
`7023b25`, todos invisíveis para `verify.ps1` por ele rodar só em Windows. Estão
detalhados nas mensagens dos dois commits; o mais grave é que
`TestRun_OverflowSchedulesExactlyOne` carregava uma **cópia do corpo de
tratamento de overflow dentro do próprio teste** e afirmava sobre a
reimplementação — apagar o tratamento de produção o deixava verde.

**Ficou de fora:** o `bench.yml` roda em um runner só (`ubuntu-latest`). Medir
nos três sistemas daria três referências e três gates; não está no brief e
triplicaria o tempo de CI. Registrado como escolha, não como esquecimento.
