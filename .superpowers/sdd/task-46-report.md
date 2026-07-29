# Relatório Task 46: `internal/search/bm25.go` — ranking

- **Status**: DONE
- **Commit**: `feat(search): BM25 ranking with field weights`

## Resumo das Mudanças
- Criado `internal/search/bm25.go` com cálculo de ranking BM25.
- Definidas constantes `ParamK1 = 1.2`, `ParamB = 0.75`, `WeightTitle = 3.0`, `WeightHeadings = 2.0`, `WeightBody = 1.0`, `WeightRaw = 1.5`, `WeightReduced = 1.0`.
- Implementada classificação de campo (`getFieldWeight`) com base em título, headings e corpo.
- Garantido desempate determinístico por `Path` em caso de scores iguais.
- Criada suíte de testes em `internal/search/bm25_test.go`.

## Evidência de TDD
### RED
Comando:
`go test -v ./internal/search/ -run TestBM25FieldWeightsAreApplied`
Saída:
--- FAIL: TestBM25FieldWeightsAreApplied (0.00s)
    bm25_test.go:68: titulo=0.2228 corpo=0.2228 — o peso 3x do titulo nao esta sendo aplicado
FAIL

### GREEN
Comando:
`go test -v ./internal/search/...`
Saída:
=== RUN   TestAnalyzerDualIndexing
--- PASS: TestAnalyzerDualIndexing (0.00s)
=== RUN   TestNormalizeAccentsAndCase
--- PASS: TestNormalizeAccentsAndCase (0.00s)
=== RUN   TestLegalTermsDistinctiveness
--- PASS: TestLegalTermsDistinctiveness (0.00s)
=== RUN   TestPunctuationHyphenNumbers
--- PASS: TestPunctuationHyphenNumbers (0.00s)
=== RUN   TestBOMStrippedTextOffsets
--- PASS: TestBOMStrippedTextOffsets (0.00s)
=== RUN   TestBM25FieldWeightsAreApplied
--- PASS: TestBM25FieldWeightsAreApplied (0.01s)
=== RUN   TestBM25WeightTitle
--- PASS: TestBM25WeightTitle (0.01s)
=== RUN   TestBM25WeightHeadings
--- PASS: TestBM25WeightHeadings (0.01s)
=== RUN   TestBM25WeightBody
--- PASS: TestBM25WeightBody (0.00s)
=== RUN   TestBM25ParamK1
--- PASS: TestBM25ParamK1 (0.00s)
=== RUN   TestBM25ParamB
--- PASS: TestBM25ParamB (0.00s)
=== RUN   TestBM25RawVsReduced
--- PASS: TestBM25RawVsReduced (0.00s)
=== RUN   TestBM25FrequentTermNoNaN
--- PASS: TestBM25FrequentTermNoNaN (0.00s)
=== RUN   TestBM25DeterministicTieBreaking
--- PASS: TestBM25DeterministicTieBreaking (0.00s)
PASS
ok  	github.com/jonyd/gobsidian/internal/search	2.661s

## Provas de Mutação (5 Constantes Obrigatórias)

### 1. `ParamK1 = 1.2`
Comando: `pwsh -File scripts/mutate.ps1 -Path internal/search/bm25.go -Anchor 'ParamK1        = 1.2' -Replacement 'ParamK1        = 10.0' -Test TestBM25ParamK1 -Package ./internal/search/`
Saída:
[...] Mutando internal/search/bm25.go
      - ParamK1        = 1.2
      + ParamK1        = 10.0
--- FAIL: TestBM25ParamK1 (0.00s)
    bm25_test.go:154: ParamK1 saturação de frequencia fora do esperado: ratio = 3.2857
FAIL
[OK] internal/search/bm25.go restaurado byte a byte (SHA-256 confere).

### 2. `ParamB = 0.75`
Comando: `pwsh -File scripts/mutate.ps1 -Path internal/search/bm25.go -Anchor 'ParamB         = 0.75' -Replacement 'ParamB         = 0.0' -Test TestBM25ParamB -Package ./internal/search/`
Saída:
[...] Mutando internal/search/bm25.go
      - ParamB         = 0.75
      + ParamB         = 0.0
--- FAIL: TestBM25ParamB (0.00s)
    bm25_test.go:174: ParamB razao de penalidade = 1.0000, quer > 1.3
FAIL
[OK] internal/search/bm25.go restaurado byte a byte (SHA-256 confere).

### 3. `WeightTitle = 3.0`
Comando: `pwsh -File scripts/mutate.ps1 -Path internal/search/bm25.go -Anchor 'WeightTitle    = 3.0' -Replacement 'WeightTitle    = 1.0' -Test TestBM25WeightTitle -Package ./internal/search/`
Saída:
[...] Mutando internal/search/bm25.go
      - WeightTitle    = 3.0
      + WeightTitle    = 1.0
--- FAIL: TestBM25WeightTitle (0.01s)
    bm25_test.go:88: WeightTitle falhou: res = [{Path:c.md Score:0.22283745830372229} {Path:t.md Score:0.22283745830372229}]
FAIL
[OK] internal/search/bm25.go restaurado byte a byte (SHA-256 confere).

### 4. `WeightHeadings = 2.0`
Comando: `pwsh -File scripts/mutate.ps1 -Path internal/search/bm25.go -Anchor 'WeightHeadings = 2.0' -Replacement 'WeightHeadings = 0.5' -Test TestBM25WeightHeadings -Package ./internal/search/`
Saída:
[...] Mutando internal/search/bm25.go
      - WeightHeadings = 2.0
      + WeightHeadings = 0.5
--- FAIL: TestBM25WeightHeadings (0.01s)
    bm25_test.go:113: WeightHeadings falhou: scoreH=0.1600, scoreC=0.2173
FAIL
[OK] internal/search/bm25.go restaurado byte a byte (SHA-256 confere).

### 5. `WeightBody = 1.0`
Comando: `pwsh -File scripts/mutate.ps1 -Path internal/search/bm25.go -Anchor 'WeightBody     = 1.0' -Replacement 'WeightBody     = 0.0' -Test TestBM25WeightBody -Package ./internal/search/`
Saída:
[...] Mutando internal/search/bm25.go
      - WeightBody     = 1.0
      + WeightBody     = 0.0
--- FAIL: TestBM25WeightBody (0.00s)
    bm25_test.go:124: WeightBody falhou: []
FAIL
[OK] internal/search/bm25.go restaurado byte a byte (SHA-256 confere).

## Tabela de Mutações
| Constante | Valor Mutado | Teste que Reprovou | Resultado da Mutação |
|---|---|---|---|
| `ParamK1` | `10.0` | `TestBM25ParamK1` | FAIL (ratio = 3.2857) |
| `ParamB` | `0.0` | `TestBM25ParamB` | FAIL (ratio = 1.0000) |
| `WeightTitle` | `1.0` | `TestBM25WeightTitle` | FAIL (empate 0.2228) |
| `WeightHeadings` | `0.5` | `TestBM25WeightHeadings` | FAIL (scoreH < scoreC) |
| `WeightBody` | `0.0` | `TestBM25WeightBody` | FAIL (res = []) |

## Verificações Exigidas pelo Brief
| Verificação | Resultado | Evidência |
|---|---|---|
| Forma crua pontua acima da reduzida? | SIM | `nota_raw.md` (0.3166) > `nota_red.md` (0.2228) em `TestBM25RawVsReduced` |
| Termo em todas as notas ordena com utilidade? | SIM | IDF suave RSJ mantém pontuação sem NaN ou 0 global |
| Nota vazia/sem termo produz score 0 e não NaN? | SIM | Testado por `TestBM25FrequentTermNoNaN` e verificações em `CalculateBM25` |
| Desempate de score idêntico é determinístico? | SIM | Ordena por `Score` decrescente e `Path` crescente, testado por `TestBM25DeterministicTieBreaking` |

## Arquivos Alterados
- `internal/search/bm25.go`
- `internal/search/bm25_test.go`
- `.superpowers/sdd/task-46-report.md`

## git status --porcelain
```
?? internal/search/bm25.go
?? internal/search/bm25_test.go
?? .superpowers/sdd/task-46-report.md
```

## O Que Ficou de Fora
Nada.
