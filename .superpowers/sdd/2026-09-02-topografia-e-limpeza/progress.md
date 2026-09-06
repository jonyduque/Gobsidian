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
- Task 177 dispatched to impl-177 (Opus) at 11:20, base 5e89964 (task-177-base.txt). Corrections carried: serve.go anchors :88-:231, ponte.go :155-:241, daemon.go :148-:179, Step.Fn takes ctx, serve_test already uses vaulttest.Prazo (no boundedWait), logSilencioso is in boot_test not boot. Orchestrator runs the orphans gate after the commit (rebuild first, no tree edits during the run).
- Task 177: impl-177 reported DONE_WITH_CONCERNS at 11:35; commit b52b441 (14 files, +451 -318); verify.ps1 full green on 2nd try (1st failed contextcheck x3 with the brief's Vigia.Ctx field); two mutation proofs exit 0; go list shows config index lifecycle search service vault watcher.
- Ruling: accept VigiarHost(parent, stdin, log) (context.Context, *Vigia) instead of the brief's Vigia.Ctx field -- contextcheck rejects ctx stored in a struct and lifecycle.New already returns (ctx, *Lifecycle); a //nolint would silence a legitimate warning -- cost if wrong: one extra return value at three call sites.
- Order: review (rev-177, Opus) before the orphans gate; a fix round would force a rerun. Gate runs detached after the review is clean, with scripts/build.ps1 first and no tree edits during the run.
- Task 177 review (rev-177, Opus, review-177.md): spec APPROVED, quality APPROVED, 0 blocking; N1 should-fix (PassoFecharEspelho no-op survives the suite, measured exit 1 on anchor 'return v.pw.Close()'), N2 nit (docs/wiki/flows/encerramento.md source_paths still name serve.go).
- Task 177 fix round 1 dispatched to impl-177 at 11:47: add TestPassoFecharEspelhoFechaOEspelho with mutation proof, test-only commit.
- Ruling: N2 parked to the final doc pass (wiki carries source_commit and the brief did not ask for it) -- cost if wrong: one stale wiki page until the pass.
- Task 177 fix round 1: ff8f234 test(boot): the close-pipe step must actually reach the stdin watch [sem-doc]; mutation on 'return v.pw.Close()' now exit 0; verify -SkipCross -SkipNet green. Orchestrator re-reviewed the 34-line test diff itself (Reason string checked against lifecycle/stdin.go:32).
- Orphans gate for 177 launched detached 11:52 (PID 33428, all scenarios, 100 cycles, log orphans-177.log) on bin built at ff8f234. No tree edits until it ends.
- Orphans gate on ff8f234 (11:52-12:29, orphans-177.log): stdin-eof 100/100 (stdin-eof 100x), parent-death 100/100, signal 100/100 (signal 100x), daemon-idle 100/100 (reason=idle). Final line: '[OK] Nenhum orfao em 100 ciclos nos quatro mecanismos'.
- Task 177: complete (b52b441, ff8f234)
