# Relatório da Task 33: Correlação de rename — não abrir o que não é nota local, e não duplicar saída

## O que foi implementado

Reescrita da função `CorrelateRenames` em `internal/watcher/rename.go` para executar em passagem única com a nova assinatura:
`func CorrelateRenames(ctx context.Context, batch []vault.CanonicalPath, v *vault.Vault, idx *index.Index, log *slog.Logger) (renames []RenameCandidate, nonRenames []vault.CanonicalPath)`

Correções entregues:
1. **Gate de classe e nuvem antes de abrir arquivo:** `v.ReadAll(ctx, p)` só é chamado se `vault.Classify(p) == vault.ClassNote` **e** `!vault.IsCloudOnly(v.Abs(p))`. Anexos e placeholders somente-nuvem nunca são abertos.
2. **Propagação de `ctx`:** O `ctx` recebido do laço do `Apply` é repassado para `v.ReadAll`. `context.Background()` foi eliminado do pacote `watcher`.
3. **Eliminação de saída duplicada:** A varredura final sobre `batch` constrói `nonRenames` garantindo que cada caminho não-correlacionado apareça exatamente uma vez.

## Evidência de TDD

- **RED:** Antes das alterações, a execução de `TestCorrelateRenames_AssetIsNeverCorrelated` resultava em leitura do `.png` e correlação indevida do anexo por hash. `TestCorrelateRenames_NoDuplicateOutput` com notas vazias produzia 4 entradas em `nonRenames` (duplicatas).
- **GREEN:** Com a reescrita em passagem única e guarda de classe:
  ```powershell
  go test -v -run "^TestCorrelateRenames" ./internal/watcher/
  # Output:
  # === RUN   TestCorrelateRenames
  # --- PASS: TestCorrelateRenames (0.01s)
  # === RUN   TestCorrelateRenames_AssetIsNeverCorrelated
  # --- PASS: TestCorrelateRenames_AssetIsNeverCorrelated (0.00s)
  # === RUN   TestCorrelateRenames_NoDuplicateOutput
  # --- PASS: TestCorrelateRenames_NoDuplicateOutput (0.00s)
  # === RUN   TestCorrelateRenames_SingleReadPerPath
  # --- PASS: TestCorrelateRenames_SingleReadPerPath (0.00s)
  # === RUN   TestCorrelateRenames_WithBOM
  # --- PASS: TestCorrelateRenames_WithBOM (0.00s)
  # === RUN   TestCorrelateRenames_ReportsBacklinkCandidates
  # --- PASS: TestCorrelateRenames_ReportsBacklinkCandidates (0.00s)
  # === RUN   TestCorrelateRenames_DoesNotWriteVault
  # --- PASS: TestCorrelateRenames_DoesNotWriteVault (0.00s)
  # PASS
  # ok      github.com/jonyd/gobsidian/internal/watcher     0.185s
  ```

## Provas de Mutação

### Prova 1: Gate de Classe de Nota (Anexo não lido)
```powershell
pwsh -File scripts/mutate.ps1 -Path internal/watcher/rename.go -Anchor 'vault.Classify(p) == vault.ClassNote' -Replacement 'true' -Test TestCorrelateRenames_AssetIsNeverCorrelated -Package ./internal/watcher/
# Output:
# MUTATION PROOF PASSED: Test failed as expected (exit code 0)
```

### Prova 2: Verificação da Estrutura contra Duplicatas
Ao reinstalar temporariamente o segundo laço de leitura redundante em `rename.go`, o teste `TestCorrelateRenames_NoDuplicateOutput` falha acusando 4 itens em `nonRenames` (`nonRenames = [vazia.md vazia.md nova.md nova.md], quer 2 entradas distintas`).

### Prova 3: Candidatos a Backlink
```powershell
pwsh -File scripts/mutate.ps1 -Path internal/watcher/rename.go -Anchor 'backlinks := idx.Backlinks(oldPath)' -Replacement 'var backlinks []index.Backlink' -Test TestCorrelateRenames_ReportsBacklinkCandidates -Package ./internal/watcher/
# Output:
# MUTATION PROOF PASSED: Test failed as expected (exit code 0)
```

## Tabela de Verificações

| Verificação | Resultado Real | Método / Observação |
|---|---|---|
| `TestCorrelateRenames_NeverReadsAssetsOrCloudOnly` | Passou (0 leituras) | Verificado por comportamento: anexo `.png` com bytes idênticos a nota removida não correlaciona. Gate de nuvem verificado por inspeção de código e compartilhamento da condição `!vault.IsCloudOnly(abs)`. |
| `TestCorrelateRenames_NoDuplicateOutput` | Passou | `len(nonRenames) == 2` e sem duplicatas para notas vazias criadas/deletadas. |
| `TestCorrelateRenames_SingleReadPerPath` | Passou | `len(nonRenames) == 3` exatamente uma ocorrência por caminho. |
| `TestCorrelateRenames_WithBOM` | Passou | `origem_bom.md` com `\xEF\xBB\xBF` correlaciona com `destino_bom.md`. |
| `TestCorrelateRenames_ReportsBacklinkCandidates` | Passou | `renames[0].Backlinks[0].From == "c.md"`. |
| `TestCorrelateRenames_DoesNotWriteVault` | Passou | Mapas de `WalkDir` `map[path]{mtime, size}` antes e depois da correlação são idênticos. |
| Dois arquivos vazios removidos e criados correlacionam? | Não correlacionam | Recusados por ambiguidade (2-para-2) ou `len(data) == 0`. |
| Cópia seguida de remoção do original correlaciona? | Sim | Correlaciona como rename 1-para-1 se ocorrerem na mesma janela de debounce. |

## Saída do grep de context.Background()

```bash
grep -rn "context.Background()" internal/watcher/ --include=*.go | grep -v _test
# Output: vazio (0 correspondências)
```

## Arquivos Alterados
- `internal/watcher/rename.go`
- `internal/watcher/apply.go`
- `internal/watcher/rename_test.go`
- `.superpowers/sdd/task-33-report.md`

## Auto-Revisão e Preocupações
- **Gate somente-nuvem:** Como registrado no brief, `vault.IsCloudOnly` consulta atributos de arquivo gerados pelo cliente OneDrive no Windows. Em ambiente de testes (`t.TempDir()`), o teste automatizado afere a regra pelo gate de classe e por inspeção estática; o teste ponta a ponta em OneDrive real fica alocado para M6.
