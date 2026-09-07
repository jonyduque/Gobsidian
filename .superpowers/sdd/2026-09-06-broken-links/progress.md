# SDD ledger — plan: docs/superpowers/plans/2026-09-06-broken-links.md

BASE at start: d91b2fb (2026-09-06 20:20)

## Pre-flight scan
| par | produz / consome | achado |
|---|---|---|
| 182 -> 183 | `parser.Link.Anchor` para LinkMarkdown; indice resolve `Target==""&&Anchor!=""` para origem / 183 filtra por State | consistente; 183 so ve estados, nao depende da forma |
| 182 -> 184 | `IndexCacheParserVersion=2` / ESTADO.md registra | consistente |
| 183 -> 184 | tool nova em TOOLS.md/README / 184 revisa | 183 ja escreve TOOLS.md; 184 so revisa — sem conflito de arquivo (sequencial) |
| 182 interno | teste parser exige split ANTES do PercentDecode; teste writer exige ancora reposta | consistente |
| 183 interno | `Total` antes de offset/limit; teto via ComTeto; teste do teto com LimiteTeto+1 | consistente com note_list |

Ruling: sites com esquema ja sao `external` (medido, 0 em 4 cofres) — nenhuma mudanca; `www.x` sem esquema segue Obsidian (interno) — nao muda. Custo se errado: um tipo de link raro segue contado.
Ruling: tool e MCP (`vault_broken_links`), sem subcomando CLI — "ferramenta" no vocabulario do projeto e tool MCP; CLI pode vir depois se pedido. Custo: dono queria CLI -> uma task extra.
Ruling: release = tag `v1.5.0` (feat -> minor), push da tag pelo orquestrador ao final, autorizado pelo pedido "publique um release".
Task 182: dispatched impl-182 (opus) at 20:21, BASE d91b2fb
Task 182: DONE by impl-182 at 20:43 — commit 8e684d0, verify.ps1 verde, 4 provas de mutacao (exit 0). Relatorio: task-182-report.md. Desvio: resolveTarget ganhou o parametro anchor (3 chamadores) em vez de a condicao ir para o chamador — eram tres chamadores, e a copia tripla e o defeito que a task corrige.
Task 182: impl DONE_WITH_CONCERNS, commit 8e684d0; reviewer rev-182 (opus) dispatched 20:47. Note for next briefs: pre_commit_docs hook reads [sem-doc] from the COMMAND text, not the -F file.
Task 182: review rev-182 at 20:55 — spec OK, quality NEEDS FIXES: F1 note_move half-done on self-link (blocking), F2 MoveNote leaves Resolved=oldPath (blocking), F3 note_delete lists itself (should-fix), F4 report text, F5 pre_commit_docs hook greps the command (process — PARKED, own ticket). Fix round 1 -> impl-182 resumed.
Task 182 fix round 1: DONE by impl-182 at 21:08 — commit c987ff1 (F1, F2, F3 + nits F6/F7/strings.Cut + texto do F4). verify.ps1 verde, 3 provas de mutacao (exit 0). Achado NOVO, medido: auto-referencia com alvo escrito ("[[a]]" dentro de a.md) ja derrubava note_move com ENOENT ANTES da Task 182 — merece task propria.
Task 182: fix round 1 DONE by impl-182 — commit c987ff1, verify verde, 3 provas de mutacao. Re-review rerev-182 (sonnet) dispatched.
Ruling: defeito PRE-EXISTENTE achado na rodada 1 — `[[a]]` dentro de a.md derruba note_move com ENOENT (o citante e a propria nota movida; moverCorpo renomeia antes da reescrita). Entra como Task 185 (pequena) antes do release, porque o release vai sair com note_move e o defeito e reproduzido com um teste ja escrito pela metade. Custo se errado: uma task a mais.
Task 182: complete at 21:15 — commits 8e684d0, c987ff1; re-review r1: all findings addressed (review-182-r1.md). Parked: F5 hook greps command; F8 trailing '#'; TOOLS.md source==target line -> Task 184.
Task 183: dispatched impl-183 (opus) at 21:15, BASE c987ff1
Task 183: DONE_WITH_CONCERNS by impl-183 at 21:35 — commit 5f9ed57, verify verde, 5 provas de mutacao. Ruling: prefix e prefixo de string (precedente tag_list.prefix, documentado) — nao muda; custo se errado: um filtro que casa 'subtotal' com 'sub', contornavel com 'sub/'. Reviewer rev-183 (opus) dispatched.
Task 183: complete at 21:41 — commit 5f9ed57; review rev-183: spec OK, quality Approved; 2 should-fix de doc (os-dois-indices.md numerador 11/14; TOOLS.md Limites nao lista vault_broken_links) + nit (TOOLS.md:307 target vazio no self-anchor). Ruling: os tres vao para a Task 184 (revisao de docs), sem rodada de correcao — mesmos arquivos, mesmo papel. Custo se errado: nenhum; 184 e obrigatoria.
Task 185: dispatched impl-185 (sonnet) at 21:41, BASE 5f9ed57
Task 185: DONE by impl-185 at 21:59 — commit 3ff26ef, verify verde, prova de mutacao (ENOENT). Reviewer rev-185 (opus) dispatched.
Task 185: review rev-185 at 22:04 — spec OK, Approved; should-fix 1 (TOOLS.md:496 'a origem nao entra' agora falso) -> Task 184; should-fix 2 (comentario do teste nomeia discriminante errado) -> fix round 1 impl-185; nit ARMADILHAS.md mecanismo 'renomeia antes, le o caminho antigo' -> Task 184.
Task 185: complete at 22:06 — commits 3ff26ef, b34814a (fix round 1: comentario do teste; diff conferido pelo orquestrador, so comentario).
Task 184: dispatched impl-184 (opus) at 22:06, BASE b34814a. Carried: TOOLS.md:13 Limites lista vault_broken_links; TOOLS.md:307 target vazio no self-anchor; TOOLS.md:496 note_move 'a origem nao entra' falso; wiki os-dois-indices numerador; ARMADILHAS.md mecanismo renomeia-antes-le-caminho-antigo; F5 hook pre_commit_docs grepa o comando (registrar em ESTADO.md como divida).
Task 184: DONE by impl-184 at 22:22 — commit 399d0a6, 10 docs. Os 6 achados carregados fechados + ARCHITECTURE 3.6/5.6, ESTRUTURA release.yml, ESTADO marco 182-185 e 2 dividas (hook pre_commit_docs; `[x](b.md#)` parqueado). Gates: check_doc_refs EXIT=0, check_readme_anchors EXIT=0, UTF-8 nos 10, verify.ps1 -SkipCross -SkipNet EXIT=0. Build: v1.4.1-112-g399d0a6-dirty (`-dirty` = trabalho nao commitado do dono em test-vault/ e .claude/, mais o plano nao versionado). Achado NOVO: ESTADO.md dizia "formato do cache de metadados e o 3" e o codigo diz 5 (persist.go:38) — corrigido. Nao commitado de proposito: docs/superpowers/plans/2026-09-06-broken-links.md continua UNTRACKED. Tag e push: orquestrador.
Task 184: commit 2 at 22:29 — ed24393, so docs/ESTADO.md. Achado NOVO no proprio instrumento: `scripts/audit_reports.ps1:113-118` procura as 4 secoes obrigatorias por PALAVRA SOLTA no corpo (red|green|muta|verifica), entao a frase "nao ha ciclo RED/GREEN nem prova de mutacao" satisfaz tres delas e os SECAO-AUSENTE somem de um relatorio que nao tem nenhuma. Mesma classe do F5 (hook pre_commit_docs). Registrado como divida aberta em ESTADO.md, NAO corrigido (mudar gate e task de codigo). TAG VAI EM ed24393, nao em 399d0a6.
Task 184: DONE by impl-184 at 22:30 — commits 399d0a6, ed24393. Concerns: audit_reports.ps1 word-match (divida registrada); write.go:545 comentario 'a origem nao entra em diffs' falso -> final fix; PRD RF-63 so wikilink (dono decide); plano untracked -> orquestrador commita. Final review dispatched.
Final review: dispatched rev-final (opus) at 22:33, BASE d91b2fb HEAD ed24393, package review-d91b2fb..ed24393.diff, output review-final.md. Named risk: write.go:545 dry_run comment.
Final review: CHANGES_REQUESTED at 22:41 — 0 Blocking, 2 Important (F1 write.go:545 comment false; F2 CLAUDE.md graph missing service->text), 2 Nit (F3 broken.go comparator comment; F4 [x](b.md#) already parked). Ruling: all three text-only, one fix dispatch (fix-final, sonnet) at 22:41, BASE ed24393, brief final-fix-brief.md.
Final fix: DONE by fix-final at 22:49 — commit 92b6af5 (CLAUDE.md grafo service->text + justificativa; write.go comentario dry-run; broken.go comentario comparador; plano linha 15). gofmt limpo, go test service ok, verify.ps1 -SkipCross -SkipNet EXIT=0, UTF-8 ok. Scoped re-review at 22:50: orquestrador leu o diff inteiro (30+/10-, so comentarios e docs), F1/F2/F3 fechados conforme brief; F4 permanece parqueado em ESTADO.md. Final review: CLEAN.
Owner decisions pending (surface in report): PRD RF-63 wording (wikilink-only example); www.x sem esquema fica interno (paridade Obsidian); [x](b.md#) parqueado; hooks pre_commit_docs/audit_reports word-match (divida em ESTADO.md).
Release: tag v1.5.0 on the SDD-artifacts commit; push master + tag; verify Actions run.
