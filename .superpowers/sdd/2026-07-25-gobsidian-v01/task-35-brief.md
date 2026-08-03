### Task 35: Chave de `byAlias` — normalizar para que `Remove` limpe o que `Build` escreveu

Critical do índice, ativado pelo watcher. Corrige link que resolve para nota deletada.

#### Onde isto encaixa

`internal/index` é M1 e foi revisado. O defeito estava lá desde a Task 19, mas `Replace` e `Remove` só passaram a ser alcançáveis com o watcher do M2 — antes do M2 o índice só era construído por `Build` no boot, e por isso a divergência nunca se manifestava.

#### A evidência medida do defeito

Sonda executada contra o commit `bce5fa1`:

```
after Build:            b.md link resolved="a.md" state=ok
after Replace(a.md):    b.md link resolved="a.md" state=ok
after Remove(a.md):     b.md link resolved="a.md" state=ok   (a.md indexed=false)
```

O link `[[STJ]]` de `b.md` continua resolvendo para `a.md` com estado `ok` depois de `a.md` sair do índice. O servidor reporta com confiança um link válido para uma nota que não existe.

Causa: a chave de `byAlias` diverge por caixa entre quem escreve e quem lê.

| Local | Chave usada hoje |
|---|---|
| `internal/index/alias.go:19` (build do boot) | `strings.ToLower(alias)` |
| `internal/index/resolve.go:162` (leitura) | `strings.ToLower(target)` |
| `internal/index/update.go:108` (`Replace`) | `alias` cru |
| `internal/index/update.go:203, 211, 213` (`removeContributionsLocked`) | `alias` cru |
| `internal/index/update.go:355` (`MoveNote`) | `strings.ToLower(alias)` |

O parser guarda o alias exatamente como veio do frontmatter (`internal/parser/parser.go:35`, sem normalização). Então `STJ` entra minúsculo no boot e cru pelo watcher; o `Remove` procura `byAlias["STJ"]`, não encontra, e a entrada `byAlias["stj"]` sobrevive apontando para uma nota que já saiu.

#### O que implementar

Acrescente em `internal/index/alias.go`, junto do índice que a consome:

```go
// aliasKey normaliza a chave de byAlias. Toda escrita e toda leitura passam
// por aqui: o boot indexava minusculo e Replace indexava cru, e a entrada
// que Remove nao encontrava sobrevivia apontando para uma nota deletada.
func aliasKey(alias string) string { return strings.ToLower(alias) }
```

Troque os **cinco** pontos da tabela por `aliasKey(...)` — inclusive os dois que já estão corretos. O objetivo não é só consertar os três errados: é tornar a próxima divergência impossível de introduzir sem tocar na função.

#### Armadilhas já pagas neste projeto que se aplicam aqui

- **Um teste que não pode falhar é pior que teste ausente.** Um teste de alias escrito com alias todo minúsculo passa com o bug intacto. **Use alias com maiúscula.**
- **Feature P1 não tem direito de apagar dado P0.** A normalização não pode mudar a contagem de arestas da paridade do M1, que é a métrica de sucesso mais forte do PRD. Rode a suíte de `internal/index` inteira e compare.

#### Verificações além dos passos

- `TestAliasSurvivesReplaceAndRemove` em `internal/index/alias_test.go`: cofre com `a.md` (`aliases: [STJ]`) e `b.md` (`[[STJ]]`); `Build`; afirmar `resolved == "a.md"`; `Replace(a.md)`; afirmar ainda `"a.md"`; `Remove("a.md")`; afirmar `resolved == ""` e o estado de alvo ausente.
- **Prova de mutação obrigatória:** reverter **só** `update.go:108` para `ix.byAlias[alias]`, confirmar que `TestAliasSurvivesReplaceAndRemove` reprova, restaurar. Cole a saída.
- A suíte de paridade do M1 continua com o mesmo número de arestas resolvidas? Cole o antes e o depois.
- Existe algum outro acesso a `ix.byAlias` fora dos cinco pontos da tabela? Cole a saída de `grep -rn "byAlias" internal/index/ --include=*.go`.

#### Regras de execução

- **O plano é a fonte.** Se um teste falhar por motivo que a seção não explica, **pare e reporte**.
- **Nunca rode `git checkout`, `git restore`, `git stash`, `git clean` ou `git reset`.**
- **`go mod tidy` está proibido.**
- **Verde obrigatório antes do commit:** `go test -race ./...`, `go vet ./...`, `gofmt -l .`, `GOOS=linux go vet ./...`, `GOOS=darwin go vet ./...`.
- Commits em Conventional Commits, em inglês.

#### Contrato de relatório

Grave em `.superpowers/sdd/task-35-report.md`: o que implementou; evidência de TDD (RED e GREEN); a prova de mutação com saída colada; a saída do `grep` de `byAlias`; a comparação de arestas da paridade; arquivos alterados; achados da auto-revisão; e preocupações.

Responda com no máximo 15 linhas, no formato acima.

**Files:**
- Modify: `internal/index/alias.go` (`aliasKey`), `internal/index/update.go` (quatro pontos)
- Create: `internal/index/alias_test.go` ou acrescentar ao teste de alias existente

**Commit:** `fix(index): normalize byAlias keys so Remove clears what Build wrote`

---

