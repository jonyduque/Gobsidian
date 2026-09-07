# Final fix round — brief (Tasks 182–185 branch, HEAD ed24393)

Three text-only fixes. NO behavior change. NO new tests. Do not touch other files.

## F1 — `internal/service/write.go:544-549` — comment in `if req.DryRun {` is false

Current comment says "A origem nao entra em diffs: mover nao altera o conteudo dela, e UnifiedDiff de um texto contra ele mesmo e "" ...". Since Task 185 a moved note that cites ITSELF with a written target (e.g. `[[a]]` or `[x](a.md)` inside `a.md`) DOES enter `diffs`, keyed by the OLD path (the file that exists on disk during dry-run), and is counted in `links_updated` (totalLinks is computed before the branch). For a Markdown self-link the diff is non-empty (`[x](a.md)` -> `[x](sub/a.md)`). An anchor-only self-reference (`[[#h]]`, `[x](#h)`) enters neither, because of the `rl.Target == ""` guard earlier in the function (~line 505).

Rewrite the comment (Portuguese without accents, like neighbors) to separate the origin's two roles:
- as the MOVED note, it gets no entry of its own in `diffs` — the move does not rewrite its body by itself, and an empty item would say "this note does not change" about the note that changes location; the `os.ReadFile(absFrom)` below remains only the "unreadable origin fails here, not just in real execution" check;
- as a CITER of itself with a written target, it enters `diffs` normally under the OLD path (what exists on disk during dry-run) — unlike real execution, which reports it in `rewritten` under the NEW path — and counts in `links_updated` in both modes. Anchor-only self-references enter neither (guard at the top of the citer loop).

`docs/TOOLS.md` note_move section already describes this correctly; do not change it.

## F2 — `CLAUDE.md:123` — dependency graph missing `service → text`

`internal/service/broken.go` imports `internal/text` (`text.ChaveDeCaminho` for the `prefix` filter). Measured: `GOOS=windows go list -f '{{.Imports}}' ./internal/service` lists internal/text. New in this branch: `git grep -l 'internal/text' d91b2fb -- 'internal/service/*.go'` is empty.

Edit the graph line to: `service  → index, parser, search, text, vault, writer`.
Add a justification paragraph right after the `writer → text` paragraph (before the "Quatro arestas existem **só em teste**" paragraph), same style, e.g.:

`service → text` é de 2026-09-06 (Task 183): o filtro `prefix` de `vault_broken_links` compara caminho de origem com o prefixo pedido, e comparar caminho é `text.ChaveDeCaminho` — a mesma conta do `writer` (Task 169) e do `index`. Uma comparação local faria `Sub/` e `sub/` serem duas pastas aqui e uma lá. `text` continua folha; o grafo continua acíclico.

Also update the sentence in the intro of the graph block that says "Duas linhas mudaram desde 2026-09-02: a do `writer` e a do `boot`" to mention that the `service` line changed on 2026-09-06 (Task 183) too. Keep accents as CLAUDE.md uses them. Validate: `python -c "open('CLAUDE.md',encoding='utf-8').read()" && echo "[OK] UTF-8 valido"`.

Also `docs/superpowers/plans/2026-09-06-broken-links.md:15` says "(este plano nao cria nenhuma)". Change to "(este plano criou uma: `service → text`, Task 183, justificada em CLAUDE.md)". That file is UNTRACKED — edit it but do NOT `git add` it; the orchestrator commits it.

## F3 — `internal/service/broken.go:~122` — comparator comment, one sentence

Add to the "Ordem deterministica" comment one sentence saying the `Source` order is INHERITED from `index.NotePaths()` (already sorted) and the comparator repeats it on purpose so the guarantee is local instead of depending on another package's invariant.

## Verification and commit

1. `gofmt -l internal/service` must print nothing. `go build ./... && go vet ./internal/service/` .
2. `go test ./internal/service/ -count=1` (paste the summary line).
3. `pwsh -File scripts/verify.ps1 -SkipCross -SkipNet` must end EXIT=0 (paste the last 5 lines). Comment/doc-only change; full cross/net gate already ran on this branch.
4. `python -c "open('CLAUDE.md',encoding='utf-8').read()"` and same for the plan file.
5. Commit ONLY by explicit path: `git add internal/service/write.go internal/service/broken.go CLAUDE.md` (NOT the plan, NOT anything else, never `git add -A`/`.`). Write the message to `.superpowers/sdd/2026-09-06-broken-links/final-fix-commit-msg.txt` and commit with `git commit -F <that file>`. Message (Conventional Commits, English):

```
docs: the dry-run comment and the graph catch up with the code

note_move's dry-run comment said the origin never enters diffs; since a
moved note that cites itself is rewritten, it does, under the old path.
The dependency graph in CLAUDE.md claimed to be re-extracted today and
missed service -> text, which vault_broken_links' prefix filter added.
The order comparator in broken.go now says its Source leg is inherited
from index.NotePaths and repeated on purpose.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01P5wkw6PAdBFzF3uB1w1jNj
```

A pre-commit hook may demand docs for code changes; this commit touches CLAUDE.md so it should pass. If the hook blocks, paste the exact hook output in the report and return BLOCKED — do not bypass hooks.

## Report
Write `.superpowers/sdd/2026-09-06-broken-links/final-fix-report.md` with `## Progresso` (real `date +%H:%M` per step), the diff (`git show --stat HEAD` + `git show HEAD -- internal/ CLAUDE.md`), pasted verification output, and the commit SHA. Return to the orchestrator ONLY: status (DONE/BLOCKED), commit SHA, one line.
