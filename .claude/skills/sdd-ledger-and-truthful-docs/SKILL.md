---
name: sdd-ledger-and-truthful-docs
description: Maintain the task ledger via scripts/sdd.ps1, validate what a subagent hands back with scripts/audit_reports.ps1, and enforce zero-hallucination rules for benchmark metrics and release claims. Use before accepting any task as done, before writing a completion claim into the ledger, README or docs/OPERACAO.md, and whenever a report cites a commit SHA or a measured number.
---

# SDD Ledger & Truthful Documentation

This skill establishes strict controls for task tracking and documentation claims in Gobsidian.

## 1. Ledger Maintenance via `scripts/sdd.ps1`

- **Single Source of Truth**: The active progress ledger lives at `.superpowers/sdd/2026-07-25-gobsidian-v01/progress.md`.
- **Mandatory Workflow Steps**:
  1. Before starting a task: `pwsh -File scripts/sdd.ps1 base <N>`
  2. Extract task brief: `pwsh -File scripts/sdd.ps1 brief <N>`
  3. Package diff for review: `pwsh -File scripts/sdd.ps1 review <N>`
  4. Record task completion in `progress.md` with commit range and review notes.
- **Orchestrator Responsibility**: The main agent MUST NOT rely solely on subagent summary messages (`Status: DONE`). The main agent MUST verify that:
  - Deliverable files exist on disk.
  - The task report exists in `.superpowers/sdd/task-<N>-report.md`.
  - The ledger in `.superpowers/sdd/2026-07-25-gobsidian-v01/progress.md` is updated.

- **`.superpowers/` é versionado.** O ledger é a única coisa que atravessa sessões; um ledger que existe só na cópia de trabalho se perde junto com ela. Se artefatos novos pararem de aparecer em `git status`, o culpado é o `.gitignore` que o plugin recria em `.superpowers/sdd/` — `sdd.ps1` o apaga a cada chamada. Ver a skill `windows-toolchain-traps`.

- **O arquivo `task-<N>-base.txt` fica sujo de propósito.** Commitá-lo sozinho move o HEAD e torna a base defasada; regravar recursa. O primeiro commit da tarefa o recolhe.

---

## 1.1 Auditar antes de aceitar: `scripts/audit_reports.ps1`

```bash
pwsh -File scripts/audit_reports.ps1 33     # so a Task 33
pwsh -File scripts/audit_reports.ps1        # varredura completa
```

Sai `1` quando há achados. Ele procura as formas de mentira que **já passaram por revisão** neste projeto, e cada regra tem um caso real por trás:

| Regra | Caso real |
|---|---|
| `MUTACAO-CONDICIONAL` | "Se removermos X, o teste falha" nas Tasks 30 e 31 — e a da 30 estava errada: o reconciliador foi removido e a suíte continuou verde |
| `NAO-RESPOSTA` | "covered implicitly by the stat checks" na Task 29 |
| `HEDGE` | "ex: 408ms em teste local" numa tabela chamada "Resultado da Medição" |
| `SHA-FANTASMA` | ledger apontando a Task 31 para `14210ee`, inexistente no repositório |
| `RELATORIO-AUSENTE` | tarefa marcada completa sem relatório no disco |
| `SECAO-AUSENTE` / `CURTO` | relatório da Task 29 com 1.148 bytes, sem RED nem GREEN |

Ele **não julga conteúdo** — localiza a frase para uma pessoa conferir. `"não medido"` não é sinalizado: é a resposta certa quando não houve medição.

**Confira todo SHA que o ledger cita.** `git cat-file -t <sha>` tem que responder `commit`. Um ledger que aponta para o vazio é pior que um desatualizado, porque parece preciso.

**Prova de mutação é obrigatória e tem forma fixa** — o que foi mutado, qual teste reprovou por nome, e a saída colada. Use `scripts/mutate.ps1`; a skill `mutation-proof-discipline` tem o contrato dos códigos de saída.

---

## 2. Zero-Hallucination Rule for Benchmarks and Release Claims

When editing `docs/OPERACAO.md`, `README.md`, or release notes:

- **No Placeholder/Example Copying**: NEVER copy prompt example text (e.g. `"ex: 408ms"`, `"tende a ~30-45 MB"`) and present it as actual benchmark results.
- **Empirical Evidence Required**: Benchmark tables in `docs/OPERACAO.md` MUST reflect actual execution data from a measured run over a synthetic 5,000-note vault (`Measure-Command { ... }`).
- **Unmeasured Metric Declaration**: If a metric has not been benchmarked yet, it MUST be explicitly listed as `"Pendente de medição empírica em cofre de 5.000 notas"`.
- **Release Tag Verification**: Do NOT write `"v0.1 publicada"` in `README.md` unless the git tag `v0.1.0` has actually been created (`git tag -a v0.1.0`) and verified with `git tag -l`.
