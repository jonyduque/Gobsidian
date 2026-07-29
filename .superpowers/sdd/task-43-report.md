# Relatório Task 43: Frontmatter — delimitador de fechamento com espaço no fim

- **Status**: DONE
- **Commit**: `fix(parser): accept trailing whitespace on the frontmatter delimiter`

## Resumo das Mudanças
- Atualizado `internal/parser/frontmatter.go` para utilizar `bytes.TrimRight(line, " \t\r")` tanto no delimitador de abertura quanto no delimitador de fechamento de frontmatter.
- Aceitos espaços em branco e tabulações ao fim das linhas delimitadoras `---`, preservando o cálculo exato do `bodyOffset`.
- Adicionados testes em `internal/parser/frontmatter_test.go`.

## Evidência de TDD
### RED (com o teste novo sem a correção em frontmatter.go)
Comando:
`go test -v ./internal/parser/ -run TestFrontmatterClosingDelimiterWithTrailingSpace`
Saída:
--- FAIL: TestFrontmatterClosingDelimiterWithTrailingSpace (0.00s)
    frontmatter_test.go:168: tags = [], quer [civil] — o bloco YAML virou corpo
FAIL

### GREEN (com a correção aplicada)
Comando:
`go test -v ./internal/parser/ -run TestFrontmatterClosingDelimiterWithTrailingSpace`
Saída:
=== RUN   TestFrontmatterClosingDelimiterWithTrailingSpace
--- PASS: TestFrontmatterClosingDelimiterWithTrailingSpace (0.00s)
PASS

## Prova de Mutação
Comando:
`pwsh -File scripts/mutate.ps1 -Path internal/parser/frontmatter.go -Anchor 'if bytes.Equal(bytes.TrimRight(line, " \t\r"), fmDelim) {' -Replacement 'if bytes.Equal(bytes.TrimRight(line, "\r"), fmDelim) {' -Test TestFrontmatterClosingDelimiterWithTrailingSpace -Package ./internal/parser/`

Saída:
[...] Mutando internal/parser/frontmatter.go
      - if bytes.Equal(bytes.TrimRight(line, " \t\r"), fmDelim) {
      + if bytes.Equal(bytes.TrimRight(line, "\r"), fmDelim) {

[...] go test -race -run TestFrontmatterClosingDelimiterWithTrailingSpace ./internal/parser/
----------------------------------------------------------------------
--- FAIL: TestFrontmatterClosingDelimiterWithTrailingSpace (0.00s)
    frontmatter_test.go:168: tags = [], quer [civil] — o bloco YAML virou corpo
FAIL
FAIL	github.com/jonyd/gobsidian/internal/parser	0.878s
FAIL
----------------------------------------------------------------------
[OK] internal/parser/frontmatter.go restaurado byte a byte (SHA-256 confere).

[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.

## Verificações Exigidas pelo Brief
| Verificação | Resultado | Evidência |
|---|---|---|
| Espaço na linha de abertura aceito? | SIM | Testado por `TestFrontmatterOpeningDelimiterWithTrailingSpace` |
| Tabulação no fim aceita? | SIM | Decidido aceitar `" \t\r"`, testado por `TestFrontmatterDelimiterWithTrailingTabs` |
| `--- x` continua rejeitado? | SIM | Testado por `TestFrontmatterDelimiterWithExtraTextRejected` |
| Nota com BOM e espaço produz offset certo? | SIM | `vault.StripBOM` remove BOM de 3 bytes antes do parse; `ShiftOffsets(3)` preserva o offset correto |
| Goldens alterados | 0 goldens alterados | Nenhuma das 48 fixtures existentes em `testdata/parser/` possuía espaço/tab no delimitador |

## Arquivos Alterados
- `internal/parser/frontmatter.go`
- `internal/parser/frontmatter_test.go`
- `.superpowers/sdd/task-43-report.md`

## git status --porcelain
```
 M internal/parser/frontmatter.go
 M internal/parser/frontmatter_test.go
?? .superpowers/sdd/task-43-report.md
```

## O Que Ficou de Fora
Nada.
