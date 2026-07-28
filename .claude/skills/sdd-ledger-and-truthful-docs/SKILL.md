---
name: sdd-ledger-and-truthful-docs
description: Guidelines for maintaining task ledger progress (.superpowers/sdd/progress.md via scripts/sdd.ps1), validating subagent deliverables, and enforcing zero-hallucination rules for benchmark metrics and release claims.
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

---

## 2. Zero-Hallucination Rule for Benchmarks and Release Claims

When editing `docs/OPERACAO.md`, `README.md`, or release notes:

- **No Placeholder/Example Copying**: NEVER copy prompt example text (e.g. `"ex: 408ms"`, `"tende a ~30-45 MB"`) and present it as actual benchmark results.
- **Empirical Evidence Required**: Benchmark tables in `docs/OPERACAO.md` MUST reflect actual execution data from a measured run over a synthetic 5,000-note vault (`Measure-Command { ... }`).
- **Unmeasured Metric Declaration**: If a metric has not been benchmarked yet, it MUST be explicitly listed as `"Pendente de medição empírica em cofre de 5.000 notas"`.
- **Release Tag Verification**: Do NOT write `"v0.1 publicada"` in `README.md` unless the git tag `v0.1.0` has actually been created (`git tag -a v0.1.0`) and verified with `git tag -l`.
