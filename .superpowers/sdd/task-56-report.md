# Relatório Task 56: `internal/writer/diff.go` — Myers sobre Linhas

- **Status**: DONE
- **Commit**: `feat(writer): line-based Myers diff`

## Resumo das Mudanças
- Implementado o algoritmo Myers de diff baseado em linhas em Go puro sem dependências externas em `internal/writer/diff.go` (~160 linhas).
- Implementada normalização de fins de linha (`\r\n` e `\n`) na divisão de linhas (`splitLines`), garantindo que arquivos CRLF e LF equivalentes resultem em diff vazio (evitando o falso positivo clássico do Windows onde cada linha é marcada como alterada por causa de `\r`).
- Criada a função `UnifiedDiff(aName, bName, aText, bText string, contextLines int) string` gerando diffs no formato unificado padronizado (patch format) com contagem de contexto.
- Criada a suíte de testes unitários em `internal/writer/diff_test.go` cobrindo equivalência de CRLF, inserção/deleção no meio de texto, arquivos vazios, última linha sem `\n` e medição de alocações.

## Evidência de TDD

### RED
Comando:
`go test -v ./internal/writer/ -run TestDiff` (antes de criar diff.go)
Saída:
FAIL: UnifiedDiff undefined / package internal/writer sem diff.go

### GREEN
Comando:
`go test -v ./internal/writer/ -run TestDiff`
Saída:
=== RUN   TestDiff_CRLFIdenticalProducesEmpty
--- PASS: TestDiff_CRLFIdenticalProducesEmpty (0.00s)
=== RUN   TestDiff_InsertionAndDeletionMiddle
--- PASS: TestDiff_InsertionAndDeletionMiddle (0.00s)
=== RUN   TestDiff_EmptyAgainstNonEmpty
--- PASS: TestDiff_EmptyAgainstNonEmpty (0.00s)
=== RUN   TestDiff_NoTrailingNewlineLine
--- PASS: TestDiff_NoTrailingNewlineLine (0.00s)
=== RUN   TestDiff_AllocationsAndPerformance
    diff_test.go:89: Medicao de Alocacoes: 270 alocacoes para diff de 1000 linhas com 10 alteracoes
--- PASS: TestDiff_AllocationsAndPerformance (0.01s)
PASS
ok  	github.com/jonyd/gobsidian/internal/writer	0.580s

## Prova de Mutação

### Normalização de EOL (`strings.ReplaceAll(text, "\r\n", "\n") -> text`)
Comando:
`pwsh -File scripts/mutate.ps1 -Path internal/writer/diff.go -Anchor 'normalized := strings.ReplaceAll(text, "\r\n", "\n")' -Replacement 'normalized := text' -Test TestDiff_CRLFIdenticalProducesEmpty -Package ./internal/writer/`
Saída:
--- FAIL: TestDiff_CRLFIdenticalProducesEmpty (0.00s)
    diff_test.go:22: CRLF contra LF equivalente produziu diff nao-vazio:
        --- a.md
        +++ b.md
        @@ -1,8 +1,4 @@
         # Titulo
         
        -
        -
         linha1
        -
         linha2
        -
FAIL
[OK] internal/writer/diff.go restaurado byte a byte (SHA-256 confere).

## Verificações Exigidas pelo Brief
| Verificação | Resultado | Evidência |
|---|---|---|
| Arquivo CRLF contra ele mesmo e contra LF produz diff vazio? | SIM | `TestDiff_CRLFIdenticalProducesEmpty` |
| Inserção e remoção no meio produzem o diff unificado correto? | SIM | `TestDiff_InsertionAndDeletionMiddle` |
| Arquivo vazio contra não vazio e vice-versa funcionam? | SIM | `TestDiff_EmptyAgainstNonEmpty` |
| Linha final sem `\n` é tratada corretamente? | SIM | `TestDiff_NoTrailingNewlineLine` |
| Quantas alocações em 1.000 linhas com 10 mudanças? | 270 | `TestDiff_AllocationsAndPerformance` (`270 alocações`) |

## Arquivos Alterados
- `internal/writer/diff.go`
- `internal/writer/diff_test.go`
- `.superpowers/sdd/task-56-report.md`

## O Que Ficou de Fora
Nada.
