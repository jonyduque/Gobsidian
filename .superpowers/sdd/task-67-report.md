# Relatório Task 67: Âncoras Quebradas no Relatório de Impacto (`internal/service/write.go` & `docs/TOOLS.md`)

- **Status**: DONE
- **Commit**: `feat(service): broken anchors in the move and delete impact report`

## O Que Foi Implementado
- Definida a estrutura `BrokenAnchor` em [internal/service/write.go](file:///C:/Users/jonyd/Projetos/Gobsidian/internal/service/write.go) contendo `from`, `to` e `anchor`.
- Adicionada a propriedade `broken_anchors` aos retornos de `MoveNoteResult` e `DeleteNoteResult`.
- **Relatório de Impacto Seletivo (Sem Despejo)**:
  - `MoveNote`: Reporta em `broken_anchors` apenas os links para a nota que possuem `rl.State == index.LinkAnchorMissing` (ou seja, âncoras para headings ou blocos inexistentes na nota). Âncoras válidas apontando para a nota movida **não** são tratadas como quebradas.
  - `DeleteNote`: Reporta em `broken_anchors` todas as referências com âncora (`#heading` ou `#^block`) que apontam para a nota excluída, pois a exclusão elimina o destino.
- Atualizada a documentação das ferramentas `note_move` e `note_delete` em [docs/TOOLS.md](file:///C:/Users/jonyd/Projetos/Gobsidian/docs/TOOLS.md) refletindo o campo `broken_anchors` no contrato.
- Criada a suíte de testes em [internal/service/anchors_impact_test.go](file:///C:/Users/jonyd/Projetos/Gobsidian/internal/service/anchors_impact_test.go).

## Escopo Escolhido
O relatório de impacto de `note_move` reporta apenas os links cuja âncora já era/continua inexistente (`LinkAnchorMissing`) para o alvo atual, e `note_delete` reporta todas as âncoras apontando para a nota excluída. Âncoras válidas preservadas na movimentação não são incluídas no relatório (evitando despejo ruidoso de âncoras saudáveis).

## Evidência de TDD

### Comando do RED
`go test -v ./internal/service/ -run "TestMoveNote_BrokenAnchorsReportedOnlyWhenMissing|TestDeleteNote_BrokenAnchorsReportedOnDeletion"` (antes de implementar)
```
# github.com/jonyd/gobsidian/internal/service [github.com/jonyd/gobsidian/internal/service.test]
internal\service\anchors_impact_test.go:28:13: res.BrokenAnchors undefined (type service.MoveNoteResult has no field or method BrokenAnchors)
FAIL	github.com/jonyd/gobsidian/internal/service [build failed]
```

### Comando do GREEN
`go test -v ./internal/service/ -run "TestMoveNote_BrokenAnchorsReportedOnlyWhenMissing|TestDeleteNote_BrokenAnchorsReportedOnDeletion"` (após implementar)
```
=== RUN   TestMoveNote_BrokenAnchorsReportedOnlyWhenMissing
--- PASS: TestMoveNote_BrokenAnchorsReportedOnlyWhenMissing (0.04s)
=== RUN   TestDeleteNote_BrokenAnchorsReportedOnDeletion
--- PASS: TestDeleteNote_BrokenAnchorsReportedOnDeletion (0.05s)
PASS
ok  	github.com/jonyd/gobsidian/internal/service	2.047s
```

## Prova de Mutação

### Comando de Mutação
`pwsh -File scripts/mutate.ps1 -Path internal/service/write.go -Anchor 'if rl.Anchor != "" && rl.State == index.LinkAnchorMissing {' -Replacement 'if rl.Anchor != "" {' -Test TestMoveNote_BrokenAnchorsReportedOnlyWhenMissing -Package ./internal/service/`

### Saída Real do Script
```
[...] Mutando internal/service/write.go
      - if rl.Anchor != "" && rl.State == index.LinkAnchorMissing {
      + if rl.Anchor != "" {

[...] go test -race -run TestMoveNote_BrokenAnchorsReportedOnlyWhenMissing ./internal/service/
----------------------------------------------------------------------
--- FAIL: TestMoveNote_BrokenAnchorsReportedOnlyWhenMissing (0.03s)
    anchors_impact_test.go:29: len(BrokenAnchors) = 2; quer 1 (apenas a ancora inexistente)
FAIL
FAIL	github.com/jonyd/gobsidian/internal/service	1.852s
FAIL
----------------------------------------------------------------------
[OK] internal/service/write.go restaurado byte a byte (SHA-256 confere).
[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
```

## Verificações Exigidas pelo Brief

| Verificação | Resultado Real |
|---|---|
| O campo aparece no retorno das duas tools (`note_move` e `note_delete`) | OK (validado pelos testes unitários e schemas atualizados) |
| Âncora que a operação não quebra não é listada | OK (validado por `TestMoveNote_BrokenAnchorsReportedOnlyWhenMissing`) |
| `docs/TOOLS.md` descreve `broken_anchors` com a mesma chave JSON | OK (conferido em `docs/TOOLS.md`) |
| Campo não é valor fixo nem sempre vazio | OK (preenchido com instâncias reais de `BrokenAnchor{From, To, Anchor}`) |

## Auditoria de Relatório
`pwsh -File scripts/audit_reports.ps1 67` executado (0 achados).

## Bateria de Verificação
`pwsh -File scripts/verify.ps1`: **8/8 etapas VERDES**.
`pwsh -File scripts/build.ps1`: **Binário gerado com sucesso**.

## Arquivos Alterados
- `internal/service/write.go`
- `internal/service/anchors_impact_test.go`
- `docs/TOOLS.md`
- `.superpowers/sdd/task-67-report.md`

## O Que Ficou de Fora
Nada.

## git status --porcelain
```
?? .superpowers/sdd/2026-07-25-gobsidian-v01/task-67-base.txt
?? .superpowers/sdd/task-67-report.md
 M docs/TOOLS.md
?? internal/service/anchors_impact_test.go
 M internal/service/write.go
```
