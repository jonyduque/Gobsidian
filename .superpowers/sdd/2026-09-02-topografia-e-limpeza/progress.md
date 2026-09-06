# Ponteiro — este plano NÃO tem ledger próprio

O ledger deste projeto é **um só**, e fica em:

    .superpowers/sdd/2026-07-25-gobsidian-v01/progress.md

As Tasks 146 a 181 (plano `docs/superpowers/plans/2026-09-02-topografia-e-limpeza.md`)
são registradas **lá**, na mesma numeração contínua das Tasks 1 a 145.

`sdd.ps1 status` resolve o plano mais recente e avisa que o ledger está ausente
aqui. O aviso é esperado; este arquivo existe para responder a ele.

**Por que um só.** Este projeto já teve dois ledgers e eles divergiram: um tinha
16 tarefas e o outro tinha 6, e ninguém percebeu até alguém re-executar trabalho
pronto. A regra registrada no `CLAUDE.md` saiu daquele episódio. Um ledger por
plano recria exatamente a condição — a numeração é contínua, o histórico é o
mesmo, e a próxima sessão não tem contexto, tem o ledger.

Os outros artefatos por plano (`task-N-base.txt`, `review-*.diff`) ficam aqui
mesmo: são por tarefa e descartáveis. O ledger não é.

- Task 176: impl-176 reported DONE at 11:04; commits 022d2ef (feat(cli): index and inspect load from the cache) and 8a11cec (refactor(cli): register shared vault flags once [sem-doc]); verify.ps1 green twice (full before commit 1, -SkipCross -SkipNet after commit 2); mutation proof exit 0. Review package review-6b70721..8a11cec.diff; rev-176 (Sonnet) dispatched 11:08.
- Task 176 review (rev-176, Sonnet, review-176.md): spec APPROVED, quality APPROVED, 0 blocking, nits N1 (index.go/inspect.go repeat the vault.New+loggerDeCLI+AbrirIndice+MarshalIndent block; pre-existing pattern) and N2 (OPERACAO.md sentence lists six subcommands then four flags). Reviewer re-ran the mutation proof (exit 0) and verify -SkipCross -SkipNet (green).
- Ruling: N1 parked, no extraction now -- three copies is the codebase's file-per-operation convention and an `abrirCLI` would be the fourth helper-shaped thing this plan adds; revisit if a fourth CLI reader appears -- cost if wrong: three copies of one block drift.
- Ruling: N2 left as written -- the sentence corrects itself in the same clause; folded into the final doc pass list instead of a fix round -- cost if wrong: one reader misreads doctor as having --cache-dir.
- Task 176: complete (022d2ef, 8a11cec)
