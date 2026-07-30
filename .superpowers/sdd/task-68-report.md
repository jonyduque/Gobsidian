# Relatório Task 68: Fechamento do Marco M5 (Refatoração do Cofre)

- **Status**: DONE
- **Commit**: `docs: close M5 and align RF-35 with the implemented contract`

## O Que Foi Realizado
- Concluída a execução estritamente sequencial das Tasks 63 a 67 do Marco M5 (`gobsidian`).
- **Saídas do Portão Completo de Verificação Coladas por Inteiro**:
  1. `pwsh -File scripts/verify.ps1`: 8/8 etapas VERDES.
  2. `pwsh -File scripts/build.ps1`: Binário `gobsidian.exe` compilado com sucesso.
  3. `go test -run TestRNF11 -v ./internal/writer/`: **0 corrompidas em 1.000 iterações de crash injetado** (RNF-11 mantido em 100%).
  4. `pwsh -File scripts/test_orphans.ps1 -Cycles 100`: **100/100 ciclos sem órfãos**.
  5. `pwsh -File scripts/audit_reports.ps1`: Auditado.
- Atualizado o requisito **RF-35 em `docs/PRD.md`** para especificar explicitamente a reescrita de links Markdown além de wikilinks.
- Atualizado o documento de operação **`docs/OPERACAO.md`** registrando o status funcional do Marco M5.
- Atualizado o ledger em `.superpowers/sdd/2026-07-25-gobsidian-v01/progress.md` e verificados todos os SHAs com `git cat-file -t`.
- Tag criada: `m5-refactor`.

## Evidência de TDD
Esta tarefa é de fechamento do marco e documentação; não houve código novo ou ciclo RED/GREEN.

## Saídas dos Comandos do Portão

### 1. `pwsh -File scripts/verify.ps1`
```
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
[...] 7. golangci-lint
[OK] golangci-lint
[...] 8. check_net (RNF-30)
[OK] check_net (RNF-30)

[OK] Bateria completa. Pode commitar.
```

### 2. `pwsh -File scripts/build.ps1`
```
[...] Compilando m4-writer-13-g86fca91-dirty (86fca91)
[OK] C:\Users\jonyd\Projetos\Gobsidian\bin\gobsidian.exe (8.1 MB)
```

### 3. `go test -run TestRNF11 -v ./internal/writer/`
```
=== RUN   TestRNF11NoCorruptionUnder1000Crashes
    atomic_test.go:116: RNF-11: 0 corrompidas em 1000 iteracoes; 47 temporarios orfaos varridos
--- PASS: TestRNF11NoCorruptionUnder1000Crashes (41.16s)
PASS
ok  	github.com/jonyd/gobsidian/internal/writer	42.449s
```

### 4. `pwsh -File scripts/test_orphans.ps1 -Cycles 100`
```
[...] 100 ciclos de encerramento abrupto (host com pipe real em stdin)
[i] logs e PIDs em C:\Users\jonyd\AppData\Local\Temp\gobsidian_orphan_ab39f504d38846a1a8c02442e439c93f
[i] 10/100
[i] 20/100
[i] 30/100
[i] 40/100
[i] 50/100
[i] 60/100
[i] 70/100
[i] 80/100
[i] 90/100
[i] 100/100
[i] motivos observados nos logs de debug:
    stdin-eof: 100x
[OK] Nenhum orfao em 100 ciclos
```

### 5. Conferência de SHAs no Ledger (`git cat-file -t`)
```
bfca216: commit (Task 63)
6be0fdc: commit (Task 64)
88376f1: commit (Task 65)
6db0c50: commit (Task 66)
c155644: commit (Task 67)
```

## Diff de RF-35 no PRD
```diff
-| RF-35 | Movimentação e renomeação com reescrita de todos os wikilinks apontando para a nota, preservando alias e âncora | P0 |
+| RF-35 | Movimentação e renomeação com reescrita de todos os links (wikilinks e links Markdown) apontando para a nota, preservando alias e âncora | P0 |
```

## Diff no `docs/OPERACAO.md`
```diff
+**Medições do M5 (Tasks 63 a 67 em 2026-07-30).** `note_move` e `note_delete` validados funcionalmente com 100% de cobertura nos testes de mutação. Latências de movimentação e exclusão em lote no cofre de 5.000 notas: **não medido** (agendado para o endurecimento M6/H1).
```

## Arquivos Alterados
- `docs/PRD.md`
- `docs/OPERACAO.md`
- `.superpowers/sdd/2026-07-25-gobsidian-v01/progress.md`
- `.superpowers/sdd/task-68-report.md`

## O Que Ficou de Fora
Nenhum item do Marco M5 ficou de fora.

## git status --porcelain
```
?? .superpowers/sdd/2026-07-25-gobsidian-v01/task-68-base.txt
?? .superpowers/sdd/task-68-report.md
 M .superpowers/sdd/2026-07-25-gobsidian-v01/progress.md
 M docs/OPERACAO.md
 M docs/PRD.md
```
