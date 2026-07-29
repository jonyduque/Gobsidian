# Relatório da Task 35: Chave de `byAlias` — normalizar para que `Remove` limpe o que `Build` escreveu

## O que foi implementado

1. **Centralização da chave de `byAlias`:**
   - Criada a função helper `aliasKey(alias string) string { return strings.ToLower(alias) }` em `internal/index/alias.go`.
2. **Atualização de todos os acessos a `byAlias`:**
   - `alias.go:19` (`buildAliasMap`): Atualizado para `aliasKey(alias)`.
   - `resolve.go:161` (`resolveByAlias`): Atualizado para `aliasKey(target)`.
   - `update.go:108` (`Replace`): Atualizado para `aliasKey(alias)`.
   - `update.go:203, 211, 213` (`removeContributionsLocked`): Atualizado para `aliasKey(alias)`.
   - `update.go:355` (`MoveNote`): Atualizado para `aliasKey(alias)`.
3. **Inclusão do teste com alias maiúsculo (`STJ`):**
   - Criado `TestAliasSurvivesReplaceAndRemove` em `internal/index/alias_test.go`.

## Evidência de TDD

- **RED:** Ao reverter `update.go` para inserir a chave sem normalizar (`ix.byAlias[alias]`), a execução de `Replace("a.md")` insere a chave crua `STJ`. A chamada subsequente a `Remove("a.md")` busca e deleta `byAlias["STJ"]` (que não existia), deixando a chave normalizada `stj` intacta no mapa. O teste reprova no passo 3 acusando `resolved="a.md"` em vez de `""`.
- **GREEN:**
  ```powershell
  go test -v -run "^TestAliasSurvivesReplaceAndRemove$" ./internal/index/
  # Output:
  # === RUN   TestAliasSurvivesReplaceAndRemove
  # --- PASS: TestAliasSurvivesReplaceAndRemove (0.00s)
  # PASS
  # ok      github.com/jonyd/gobsidian/internal/index       0.178s
  ```

## Provas de Mutação

### Prova 1: Reversão da Normalização no Replace
```powershell
pwsh -File scripts/mutate.ps1 -Path internal/index/update.go -Anchor 'key := aliasKey(alias)' -Replacement 'key := alias' -Test TestAliasSurvivesReplaceAndRemove -Package ./internal/index/
# Output:
# MUTATION PROOF PASSED: Test failed as expected (exit code 0)
```

## Tabela de Verificações

| Verificação | Resultado Real | Método / Observação |
|---|---|---|
| `TestAliasSurvivesReplaceAndRemove` | Passou | Testado com alias `STJ` maiúsculo; `Replace` e `Remove` mantêm e limpam a chave |
| Prova de mutação em `update.go` | Passou (exit 0) | Teste reprovou ao mutar `key := aliasKey(alias)` para `key := alias` |
| Paridade do M1 mantida | 100% equivalente | Todos os testes do pacote `index` passaram sem alteração de arestas |
| `grep -rn "byAlias" internal/index/` | 5 acessos | Todos usam `aliasKey(...)` |

## Saída do grep de byAlias em internal/index

```bash
grep -rn "byAlias" internal/index/ --include=*.go
# Output:
# internal/index/alias.go:19:			key := aliasKey(alias)
# internal/index/alias.go:20:			ix.byAlias[key] = append(ix.byAlias[key], n.Path)
# internal/index/index.go:50:		byAlias:                  make(map[string][]vault.CanonicalPath),
# internal/index/resolve.go:162:	paths, ok := ix.byAlias[key]
# internal/index/update.go:108:		key := aliasKey(alias)
# internal/index/update.go:109:		ix.byAlias[key] = append(ix.byAlias[key], entry.Path)
# internal/index/update.go:203:			key := aliasKey(alias)
# internal/index/update.go:204:			al := ix.byAlias[key]
# internal/index/update.go:209:				delete(ix.byAlias, key)
# internal/index/update.go:211:				ix.byAlias[key] = filteredAl
# internal/index/update.go:350:		key := aliasKey(alias)
# internal/index/update.go:351:		al := ix.byAlias[key]
```

## Arquivos Alterados
- `internal/index/alias.go`
- `internal/index/resolve.go`
- `internal/index/update.go`
- `internal/index/alias_test.go`
- `.superpowers/sdd/task-35-report.md`

## Auto-Revisão e Preocupações
- Nenhuma. Todos os 5 pontos de acesso a `byAlias` foram normalizados com `aliasKey(...)`.
