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
- Tasks 178+179 batched and dispatched to impl-178 (Sonnet) at 12:36, base 849dbb4 (task-178-base.txt). Corrections carried: graph.go Include :572 / includeSet :618-628, tools_read.go Include :376, search.go Hits :105/:132/:377/:419, ESTADO.md M3 baseline at :160, bench vault %TEMP%/vault_5000, fresh cache dir cache179. Three commits expected.
- Tasks 178+179: impl-178 DONE/DONE at 13:11; commits 364a3df (fix note_metadata include), b6cf8be (docs tools schema), 9803b01 (feat! vault_search results only); verify full green x2, -SkipCross -SkipNet for the doc commit; mutation 178 exit 0; 179 manual reintroduction FAIL pasted; M3 depois measured: indented 108204, compact 95969 (baseline 216304 / 195481, ~-50%); needed --max-results 200 (Task 168 ceiling). New finding B19 written into SUGESTOES.md (heading_level range unenforced; PatchNoteRequest.HeadingLevel dead). Review package review-849dbb4..9803b01.diff; rev-178 (Sonnet) dispatched.
- Tasks 178+179 review (rev-178, Sonnet, review-178.md): spec APPROVED (both), quality APPROVED, 0 blocking; N1 should-fix (include list in graph.go and the mcpsrv jsonschema tag with no sync guard), N2 nit (audit table misses note_read.max_bytes and note_list.offset, neither had a false claim).
- Task 178 fix round 1 dispatched to impl-178 at 13:22: export CamposDeMetadata, reflection test in mcpsrv, two mutation proofs, full verify.
- Ruling: N2 parked -- the two missing rows had no min/max and no false claim; the table is a report artifact, not a deliverable -- cost if wrong: none in code.
- Task 178 fix round 1: 503b94c test(mcpsrv): the include schema must name exactly the values the service accepts [sem-doc]; CamposDeMetadata exported; both mutations (tag side, slice side) exit 0; full verify green. Orchestrator re-reviewed the 90-line diff itself.
- Task 178: complete (364a3df, b6cf8be, 503b94c)
- Task 179: complete (9803b01)
- Task 180 dispatched to impl-180 (Opus), base 4ee5db8, brief task-180-brief.md; anchors re-verified at HEAD (chave.go w/o ChaveDeTag, index.go:195, update.go :216/:226/:556, query.go :173/:226, search.go :76/:390/:436-447, graph.go:368, persist_test.go:89; benchBusca(b,svc,opts,minResultados)). Two commits expected; benchstat 7x interleaved.
- Task 180 commit 1: 937e54c (bench). Ruling: parser NFD truncation (ext_tag.go:36 tagNameChar lacks unicode.Mn; inline #Ação in NFD indexes as "ac") stays OUT of Task 180 — not in the brief's Files, would mix a parser behavior change with goldens/parity into an index-key task; NFD fixture moved to frontmatter, follow-up recorded in SUGESTOES.md — cost if wrong: one more small task later.
- Task 180 implemented: 937e54c (bench), 18d9da4 (feat); report DONE_WITH_CONCERNS; full verify green; benchstat: TagsSemPrefixo -12.5%, TagListPlano -10.6%, rest ~, SearchFiltroTags B/op +0.48% (explained: escaping sorted slice). Golden unchanged (fixture ASCII lowercase). SUGESTOES.md entry for parser NFD not written (went to ESTADO.md). rev-180 (Opus) dispatched on review-4ee5db8..18d9da4.diff.
- Task 180 third commit by impl-180 after my ruling: c08c70c (docs/SUGESTOES.md B20, parser NFD truncation). rev-180 verdict CHANGES_REQUIRED (review-180.md): N1 TOOLS.md:297 says tag_list.prefix matches "itself and subtags", code is string prefix (Task 169 golden `proj/al`→`proj/alpha` deliberate); N2 prefix folding untested (3 mutations survive); N3 persist_test assertion symmetric (report overclaims); N4 tag_chave_test.go:70 precedence; N5 report "only p<0.05" wrong (two rows); N6 ESTADO "Sem tarefa aberta" stale vs B20; N7 porTag snapshot outside loop (same as porFrontmatter, accepted); N8 TOOLS.md:331 vault_stats promises link/tag counts StatsResult lacks (pre-existing); N9 note_list.tags raw and CLI index tag count undocumented.
- Ruling (N1): doc follows code — `tag_list.prefix` is a string prefix over the folded key (autocomplete), distinct from the segment-hierarchical `tags` filter; segment match would make the schema's own `civil/` example return nothing and would undo Task 169's deliberate golden — cost if wrong: one doc sentence plus a golden regen later.
- Ruling (N7): accepted as-is, same property porFrontmatter already has; N8 → owner list / final doc pass.
- Task 180 fix round 1 dispatched to impl-180 (resume): N1 doc, N2 test + 3 mutations, N4, N6, N9, report fixes N3/N5.
- Task 180 fix round 1: aa8ec0b (TOOLS.md prefix contract rewritten to string-prefix + paragraph separating it from the segment `tags` filter; mcpsrv jsonschema aligned; TestTagListDobraOPrefixoNosDoisRamos + accented-prefix check in tag_chave_test; N4 precedence fixed; ESTADO B20 pointer; note_list.tags spelling note; CLI index count "não medido"). Orchestrator re-reviewed the 181-line diff (doc+test+one jsonschema string): three N2 mutations and the N4 mutation exit 0, NFD literals verified as U+0327/U+0303 bytes, verify -SkipCross -SkipNet green. Ruling: scoped re-review by orchestrator instead of a reviewer dispatch — small diff, evidence pasted — cost if wrong: a reviewer catches it in the final whole-branch pass. Orphans gate not run for 180: no lifecycle/daemon/cmd change.
- Task 180: complete (937e54c, 18d9da4, c08c70c, aa8ec0b)
- Task 181 dispatched to impl-181 (Sonnet), base 833906f. Rulings carried: (1) brief's `escritor{w: &buf}` cannot compile — `w` is `*bufio.Writer`; use bufio.NewWriter + Flush — cost if wrong: none, compile error would surface; (2) ESTADO.md has no "dívida 1.12" entry (segregacao numbering) — record coverage under `## Formato de cache` — cost if wrong: one paragraph moved in the final doc pass; (3) a codec behavior the brief's tests assume and the code lacks is a SUGESTOES finding, not a production fix.
- Task 181 implemented: b2561e0 (persist_codec_test.go +239, ESTADO.md +49, SUGESTOES.md +16 = B21: unreachable `default` at persist_codec.go:684). DONE_WITH_CONCERNS. rev-181 (Sonnet) dispatched on review-833906f..b2561e0.diff.
- rev-181 verdict APPROVED_WITH_NITS (review-181.md): N1 typed-nil []any (valSliceNil) branch uncovered, commit message overclaims; N2 truncation test asserts err != nil not errors.Is sentinel. B21 (dead default at persist_codec.go:684) independently confirmed by reviewer (panic mutation survives). Fix round 1 dispatched to impl-181 (resume).
- Task 181 fix round 1: 7d4d684 ([]any(nil) case, valSliceNil mutation exit 0; errors.Is sentinel on every truncated prefix; ESTADO coverage lines re-measured: escritor.value 92.1->97.4, leitor.value 91.7->94.4, package 89.8%). Orchestrator re-reviewed the 23-line diff. Ruling: scoped re-review by orchestrator — two-line test additions with mutation output pasted — cost if wrong: final review catches it.
- Task 181: complete (b2561e0, 7d4d684)
- Phase 5 complete. Next: final whole-branch review (Opus) over 6b70721..HEAD is too wide — plan base is the plan's first task base; use the ledger's Task 146 base. ONE fix dispatch, one scoped re-review, then the final doc pass items.
- 2026-09-06 15:30 Final whole-branch review dispatched to rev-final (Opus), range 6c5d1f1..5c613d5. Package split by area: final-log-stat.txt, final-diff-prod.diff (5858), final-diff-tests.diff (9069), final-diff-docs.diff (6651, excl. OPERACAO/wiki), final-diff-excluded-stat.txt; brief in final-review-brief.md; known/parked list handed over. Output: review-final.md.
- 2026-09-06 15:48 rev-final returned CHANGES_REQUIRED (review-final.md): 0 critical, 3 important (F1 doctor --debounce-ms dead flag; F2 tags jsonschema descriptions missing the Task 180 semantics; F3 segregacao.md candidate 3 still open), 5 nits, plus a stale-references table. Gate verified green by the reviewer; import graph matches CLAUDE.md.
- Ruling: F1 fix as `fix(cli)`; orphans gate NOT required -- doctor is off the serve/daemon shutdown path -- cost if wrong: an orphan scenario changes without the gate noticing (none plausible from a flag removal).
- Ruling: F2 fix as `fix(mcpsrv)` with one small served-schema test on the `subtags` substring -- description is the only prose the host receives (TOOLS.md:802) -- cost if wrong: a test tied to doc prose breaks on the next rewording.
- Ruling: N1 comment follows code (any `.lock`), predicate unchanged -- runtime dir is product-only -- cost if wrong: none observable.
- Ruling: N2 (tag_list min_count default at the MCP edge) PARKED -- no observable difference; owner list -- cost if wrong: one rule applied to 3 of 4 tools.
- Ruling: N3 (wall-clock assertion in TestQ3PerformanceMeasurement) PARKED, owner list -- cost if wrong: one flaky test under load.
- Ruling: N4 (`var cacheMagic`) PARKED, keep -- test pins it -- cost if wrong: a writable format id nobody writes.
- Ruling: N5 comment gets the cache-dir-inside-vault caveat, no code change -- default CacheDir is outside the vault -- cost if wrong: none.
- 2026-09-06 15:55 Final fix round dispatched to impl-final (Sonnet), base 5c613d5, brief task-final-brief.md (Part A = parked doc pass A1-A10 verified against the tree; Part B = review-final findings with rulings). Three commits expected: fix(cli), fix(mcpsrv), docs.
