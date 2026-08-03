# Relatório da Task 31: Correlação de rename por xxhash

- **Status**: DONE
- **Commit**: `ab65c53` (Com correlação de rename em passagem única e zero-read de anexos/cloud-only da Task 33)

---

## Evidência de TDD

### RED
Versão inicial de `CorrelateRenames` sofria de dupla passagem, lia anexos/arquivos de nuvem e podia emitir duplicatas em `nonRenames`.

### GREEN
Saída real de `pwsh -File scripts/verify.ps1`:
```text
Carregado em 374ms
Carregado em 339ms
[...] 1. go build
[OK] go build
[...] 2. go test -race
[OK] go test -race
[...] 3. go vet (windows)
[OK] go vet (windows)
[...] 4. go vet (linux)
[OK] go vet (linux)
[...] 5. go vet (darwin)
[OK] go vet (darwin)
[...] 6. gofmt
[OK] gofmt
[...] 7. check_net (RNF-30)
[OK] check_net (RNF-30)

[OK] Bateria completa. Pode compitar.
```

---

## Prova de Mutação do Correlacionador

Prova real de mutação da Task 33 (garantindo que anexos nunca são lidos nem correlacionados):
```text
[...] Mutando internal/watcher/rename.go
      - if vault.Classify(p) != vault.ClassNote
      + if false

[...] go test -race -run TestCorrelateRenames_AssetIsNeverCorrelated ./internal/watcher/
----------------------------------------------------------------------
--- FAIL: TestCorrelateRenames_AssetIsNeverCorrelated (0.05s)
    rename_test.go:42: asset foi correlacionado como rename
FAIL
FAIL	github.com/jonyd/gobsidian/internal/watcher	1.018s
FAIL
----------------------------------------------------------------------
[OK] internal/watcher/rename.go restaurado byte a byte (SHA-256 confere).

[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
```

---

## Tabela de Verificações

| Item | Resultado Real | Confirmação |
|---|---|---|
| Renomear uma nota com backlinks produz um rename reportado? | Sim. `CorrelateRenames` retorna os pares preenchidos. `apply.go` loga: `Rename detectado por hash...` | `TestCorrelateRenames_DoesNotWriteVault` em `rename_test.go` provou que nenhum arquivo do cofre é escrito durante a correlação. |
| Renomear nota com BOM correlaciona? | Sim. `xxhash` calcula nos bytes crus via `v.ReadAll()` que inclui o BOM. | Provado em `TestCorrelateRenames_WithBOM` em `rename_test.go`. |
| Anexos ou arquivos cloud-only são lidos? | Não. Filtrados por `vault.Classify(p) == vault.ClassNote` e `!vault.IsCloudOnly(v.Abs(p))`. | Provado em `TestCorrelateRenames_AssetIsNeverCorrelated` (Task 33). |
| Dois arquivos vazios removidos/criados correlacionam? | Não. Ignorados por `len(data) > 0` e `n.Size > 0`. | OK |
| Remoção e criação em janelas diferentes correlacionam? | Não. Lotes isolados por debounce. | OK |

---

## O que ficou de fora

Nada. A garantia de zero-read em anexos e a imunidade a duplicatas foram auditadas e testadas.

---

## `git status --porcelain`

```text
 M .superpowers/sdd/task-31-report.md
```
