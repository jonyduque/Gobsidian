# Final-fix report — F1-F6, F8, F10

## Progresso

- 11:21 briefing e review lidos
- 11:29 scripts atuais lidos (pre_commit_docs.ps1, check_gates.ps1, audit_reports.ps1), fixtures e docs localizados
- 11:24 novos casos adicionados a check_gates.ps1 (RED capturado)
- 11:24 fixture scripts/testdata/gates/sdd-falso/task-3-report.md criada (2108 bytes)
- 11:25 pre_commit_docs.ps1 editado (A1 Segmento-Commit, A2 flags agrupadas, A3 texto do deny)
- 11:25 docs/papeis/documentador.md editado (A4)
- 11:25 audit_reports.ps1 editado (C, remocao de cercas)
- 11:25 GREEN confirmado, 19/19 casos
- 11:25 mutation proof 1 (Segmento-Commit) feita e restaurada
- 11:25 mutation proof 2 (BodySemCercas) feita e restaurada
- 11:27 docs/ESTADO.md editado (F8), UTF-8 validado nos dois .md
- 11:27 verify.ps1 -SkipCross -SkipNet rodado, EXIT=0
- 11:34 audit_reports.ps1 -Task 187 e -Task 188 conferidos, 0 SECAO-AUSENTE nos dois
- 11:35 commit 6978181 criado (hook allow, sem .go em stage)
- 11:35 audit_reports.ps1 -Task final-fix -SddRoot .../2026-09-07-gates-e-rf63 -> exit 2 (nao casou); rodado sem -Task, grep por "final-fix" sem saida -- ver Concerns
- 11:51 Round 2 iniciada: rereview-final.md lido, secao N2
- 11:52 fixture task-4-report.md criada (2071 bytes, cerca ```bash aberta e nunca fechada antes das quatro secoes reais)
- 11:52 caso novo adicionado a check_gates.ps1
- 11:52 audit_reports.ps1 corrigido (N2: indice de cerca impar exclui a ultima da alternancia)
- 11:53 RED capturado (trocando temporariamente para a versao pre-N2 do commit 6978181, via git show HEAD:..., so para rodar o caso novo -- nao ficou no working tree)
- 11:53 fix restaurado, GREEN confirmado, 20/20
- 11:54 mutation proof feita e restaurada
- 11:55 verify.ps1 -SkipCross -SkipNet rodado, EXIT=0
- 11:56 audit_reports.ps1 -Task 187/188 e fixtures 1/2/3 reconferidos, inalterados
- 12:03 commit 71a779f criado (hook allow, sem .go em stage)
- 12:27 Round 3 iniciada: final-fix-3-brief.md, secao F7 de review-final.md e secao N1 de rereview-final.md lidas
- 12:29 3 casos F7 e 1 caso N1 adicionados a check_gates.ps1
- 12:30 RED capturado: 23/24 (F7 ja passava -- excecao inalcancavel por -Simular; N1 falhou, obtido 'allow')
- 12:31 pre_commit_docs.ps1 editado (A: excecao de --amend removida; B: `\` antes de aspas tratado como escape em Segmento-Commit)
- 12:31 GREEN confirmado, 24/24
- 12:32 mutation proof 1 (excecao de --amend re-adicionada fora do -Simular) feita e restaurada
- 12:32 mutation proof 2 (branch de escape `\` removido) feita e restaurada
- 12:33 docs/ESTADO.md e docs/papeis/documentador.md editados (F9 rewrap + duas frases), UTF-8 validado nos dois
- 12:33 verify.ps1 -SkipCross -SkipNet rodado, EXIT=0
- 12:34 audit_reports.ps1 -Task 187 e -Task 188 reconferidos, 0 SECAO-AUSENTE nos dois

## Files changed

- `scripts/pre_commit_docs.ps1` — A1 (`Segmento-Commit`), A2 (flags curtas agrupadas em `$reM`), A3 (linha nova no texto de deny)
- `scripts/check_gates.ps1` — B (6 casos novos de hook + `Motivo-Hook`/`Invocar-Hook`), D1 (`Secoes-Ausentes` trata exit 2 como `'erro'`), D2 (2 casos novos de audit)
- `scripts/audit_reports.ps1` — C (`$BodySemCercas` remove blocos cercados antes de casar `$Required`)
- `scripts/testdata/gates/sdd-falso/task-3-report.md` — fixture nova (2108 bytes), secoes so em comentario de shell dentro de cerca ```bash
- `docs/ESTADO.md` — F8 (data e tamanho do corpus na medicao 79->203)
- `docs/papeis/documentador.md` — F10 (texto da escotilha alinhado ao comportamento real)

### RED

Casos novos adicionados a `check_gates.ps1` ANTES de qualquer mudanca de codigo (fixture task-3 ja existia neste ponto, pois e insumo dos casos, nao parte do codigo sob teste). Saida de `pwsh -NoProfile -File scripts/check_gates.ps1`:

```
=== pre_commit_docs.ps1 ===
[OK] escotilha em comentario de shell, mensagem sem ela -> deny
[OK] escotilha no arquivo de -F -> allow
[OK] escotilha em -m entre aspas duplas -> allow
[OK] escotilha em -m entre aspas simples -> allow
[OK] escotilha fora das aspas de -m -> deny
[OK] -m sem escotilha, .go sem doc -> deny
[OK] -m sem escotilha, .go com doc -> allow
[OK] -F arquivo inexistente, escotilha no comando -> deny
[OK] sem -m nem -F (editor), .go sem doc -> deny
[!] -m com escotilha dentro de comentario de shell, -F sem ela -> deny: esperado 'deny', obtido 'allow'
[!] dois git commit encadeados, escotilha so no primeiro -> deny: esperado 'deny', obtido 'allow'
[OK] dois git commit encadeados, escotilha so no ultimo -> allow
[!] -am com escotilha -> allow: esperado 'allow', obtido 'deny'
[OK] escotilha dentro de aspas com # no texto -> allow
[OK] motivo do allow e a escotilha, nao o catch do hook

=== audit_reports.ps1 ===
[OK] prosa que nomeia as secoes sem te-las -> 4 SECAO-AUSENTE
[OK] quatro secoes em cabecalho -> 0 SECAO-AUSENTE
[!] secoes so em comentario de shell dentro de cerca -> 4 SECAO-AUSENTE: esperado '4', obtido '0'
[OK] fixture inexistente -> erro, nao 0

[!] check_gates: 4 de 19 casos reprovados
EXIT=1
```

Quatro casos falharam como esperado: F1 (comentario de shell), F2 (primeiro commit encadeado), F3 (`-am`), F5 (secoes so em cerca). O caso de F4 (`fixture inexistente -> erro`) e o caso F2 "so no ultimo -> allow" ja passavam nesse ponto porque D1 (exit 2 -> erro) e o comportamento antigo do bug ja liam a escotilha do segundo `-m` corretamente por acidente (o codigo antigo escaneava a linha inteira).

### GREEN

Apos as edicoes de A1-A3 (`pre_commit_docs.ps1`) e C (`audit_reports.ps1`):

```
=== pre_commit_docs.ps1 ===
[OK] escotilha em comentario de shell, mensagem sem ela -> deny
[OK] escotilha no arquivo de -F -> allow
[OK] escotilha em -m entre aspas duplas -> allow
[OK] escotilha em -m entre aspas simples -> allow
[OK] escotilha fora das aspas de -m -> deny
[OK] -m sem escotilha, .go sem doc -> deny
[OK] -m sem escotilha, .go com doc -> allow
[OK] -F arquivo inexistente, escotilha no comando -> deny
[OK] sem -m nem -F (editor), .go sem doc -> deny
[OK] -m com escotilha dentro de comentario de shell, -F sem ela -> deny
[OK] dois git commit encadeados, escotilha so no primeiro -> deny
[OK] dois git commit encadeados, escotilha so no ultimo -> allow
[OK] -am com escotilha -> allow
[OK] escotilha dentro de aspas com # no texto -> allow
[OK] motivo do allow e a escotilha, nao o catch do hook

=== audit_reports.ps1 ===
[OK] prosa que nomeia as secoes sem te-las -> 4 SECAO-AUSENTE
[OK] quatro secoes em cabecalho -> 0 SECAO-AUSENTE
[OK] secoes so em comentario de shell dentro de cerca -> 4 SECAO-AUSENTE
[OK] fixture inexistente -> erro, nao 0

[OK] check_gates: 19 casos
EXIT=0
```

19 casos = 11 originais (9 hook + 2 audit) + 6 hook novos (F1, F2 x2, F3 x2, F6) + 2 audit novos (F5, F4) — bate com o esperado.

## Mutation proofs

**Prova 1 — `Segmento-Commit` volta a devolver `$linha` sem cortar.**

Alterado temporariamente para:
```powershell
function Segmento-Commit([string]$linha) {
    return $linha
    $ms = [regex]::Matches($linha, 'git\s+commit')
    ...
```

Saida de `pwsh -NoProfile -File scripts/check_gates.ps1`:
```
[!] -m com escotilha dentro de comentario de shell, -F sem ela -> deny: esperado 'deny', obtido 'allow'
[!] dois git commit encadeados, escotilha so no primeiro -> deny: esperado 'deny', obtido 'allow'
[!] check_gates: 2 de 19 casos reprovados
EXIT=1
```
Exatamente os casos de F1/F2 falharam (o caso "so no ultimo -> allow" continuou OK, como esperado — a escotilha ainda esta visivel na linha inteira nesse caso). Restaurado (removida a linha `return $linha` extra) e reconferido GREEN (19/19, ver acima).

**Prova 2 — `$BodySemCercas = $Body` (sem remover cercas).**

Alterado temporariamente:
```powershell
    $BodySemCercas = $Body
```

Saida:
```
[!] secoes so em comentario de shell dentro de cerca -> 4 SECAO-AUSENTE: esperado '4', obtido '0'
[!] check_gates: 1 de 19 casos reprovados
EXIT=1
```
Exatamente o caso da fixture-3 falhou. Restaurado (`$BodySemCercas = $foraDeCerca -join "`n"`) e reconferido GREEN (19/19).

**`git diff --stat` apos restaurar as duas mutacoes** (confirma que so a mudanca pretendida ficou nos dois scripts):
```
scripts/audit_reports.ps1   | 17 ++++++++++-
scripts/check_gates.ps1     | 69 ++++++++++++++++++++++++++++++++++++++++-----
scripts/pre_commit_docs.ps1 | 39 ++++++++++++++++++++++++-
3 files changed, 116 insertions(+), 9 deletions(-)
```

## Verification

`pwsh -NoProfile -File scripts/verify.ps1 -SkipCross -SkipNet`, cauda da saida:

```
[...] 1. go build
[OK] go build
[...] 2. go test -race
[OK] go test -race
[...] 3. contagem de testes pulados
[!] 6 testes pulados
     --- SKIP: TestAjudanteSeguraTrava (0.00s)
     --- SKIP: TestListenRestringePermissaoUnix (0.00s)
     --- SKIP: TestSignalCancelsContext (0.10s)
     --- SKIP: TestPerfilDeHeapServindo (0.00s)
     --- SKIP: TestWriteAtomicPreservaOModoDoAlvo (0.00s)
     --- SKIP: TestNew_FailsOnUnwatchablePath (0.00s)
[...] 4. go test (tetos de latencia, sem -race)
[OK] go test (tetos de latencia, sem -race)
[...] 5. go vet (windows)
[OK] go vet (windows)
[i] vet cruzado pulado (-SkipCross)
[...] 6. gofmt
[OK] gofmt
[...] 7. golangci-lint
[OK] golangci-lint
[...] 8. golangci-lint (linux)
[OK] golangci-lint (linux)
[i] check_net pulado (-SkipNet)
[...] 9. check_tool_params
[OK] check_tool_params
[...] 10. check_doc_refs
[OK] check_doc_refs
[...] 11. check_readme_anchors
[OK] check_readme_anchors
[...] 12. check_gates
[OK] check_gates

[OK] Bateria completa. Pode commitar.
EXIT=0
```

Os 6 skips sao os conhecidos (condicoes de ambiente Windows/CI, ja documentados em ESTADO.md), nenhum novo.

UTF-8 nos dois `.md` editados:
```
[OK] UTF-8 valido ESTADO.md
[OK] UTF-8 valido documentador.md
```

`audit_reports.ps1 -Task 187` e `-Task 188` continuam 0 `SECAO-AUSENTE` apos a remocao de cercas (prova que a remocao de cercas nao engole cabecalhos reais desses dois relatorios, que colam saida de comando alem de terem `### RED`/`### GREEN`/`## Mutation proofs`/`## Verification` de verdade):

```
$ pwsh -NoProfile -File scripts/audit_reports.ps1 -Task 187 2>&1 | grep -c 'SECAO-AUSENTE'
0
$ pwsh -NoProfile -File scripts/audit_reports.ps1 -Task 188 2>&1 | grep -c 'SECAO-AUSENTE'
0
```

(grep -c devolve exit 1 quando a contagem e 0 por nao ter casado linha nenhuma -- e o exit code do grep, nao do audit_reports.ps1; a contagem impressa, que e o que importa aqui, e 0 nos dois.)

## Concerns

Todas as 8 descobertas do escopo (F1, F2, F3, F4, F5, F6, F8, F10) foram enderecadas; F7 e F9 estao fora de escopo por decisao do brief. `verify.ps1 -SkipCross -SkipNet` verde, `check_gates.ps1` com 19/19, duas provas de mutacao reais coladas acima.

Um ponto sobre o proprio audit do relatorio: `pwsh -File scripts/audit_reports.ps1 -Task final-fix -SddRoot .superpowers/sdd/2026-09-07-gates-e-rf63` devolve exit 2 (`[!] Nenhum relatorio casou com 'task-final-fix-report.md' ...`), porque o filtro do script e `task-$Task-report.md` e este arquivo se chama `final-fix-report.md` (nome dado pelo brief, seguindo o padrao dos outros artefatos da rodada -- `final-fix-brief.md`, `review-final.md`, `final-fix-commit.txt` -- nenhum comeca com `task-`). Rodei sem `-Task` sobre a mesma raiz e grepei por "final-fix": zero linhas, porque o glob `task-*-report.md` tambem nao pega esse nome -- so os tres `task-186/187/188-report.md` da pasta foram varridos. Isso e um descasamento de convencao de nome, nao um SECAO-AUSENTE escondido: este relatorio tem as sete secoes pedidas em cabecalho (`## Progresso`, `## Files changed`, `### RED`, `### GREEN`, `## Mutation proofs`, `## Verification`, `## Concerns`), conferi visualmente. Se o orquestrador quiser o numero automatico, o relatorio precisaria se chamar `task-<algo>-report.md`, o que o brief nao pediu.

## Round 2 (N2)

`rereview-final.md`, secao N2: o toggle de cerca em `$BodySemCercas`
(`scripts/audit_reports.ps1`) nao tinha estado de erro para numero IMPAR de
linhas de cerca. Um `\`\`\`bash` aberto e nunca fechado por acidente (engano
comum ao colar saida de comando longa) fazia tudo do ponto da abertura ate o
fim do arquivo virar "dentro de cerca" -- inclusive `### RED`, `### GREEN`,
`## Mutation proofs` e `## Verification` reais que vinham depois, fora de
qualquer cerca de verdade. O reviewer mediu 4 falsos `SECAO-AUSENTE` num
relatorio genuinamente completo (fixture fora do repositorio).

**Conserto:** antes do loop que alterna `$emCerca`, `scripts/audit_reports.ps1`
agora coleta os INDICES de todas as linhas de cerca num `HashSet[int]`. Se a
contagem for impar, a ULTIMA e removida do conjunto -- ela vira texto comum
em vez de fechar/abrir nada, entao tudo depois dela continua visivel. O loop
so alterna `$emCerca` nas linhas cujo indice sobrou no conjunto.

Fixture nova: `scripts/testdata/gates/sdd-falso/task-4-report.md` (2071
bytes) -- um `\`\`\`bash` aberto e nunca fechado, com saida colada e
comentarios de shell nomeando red/green/mutacao/verificacao DENTRO dele,
seguido das quatro secoes reais em cabecalho fora de qualquer cerca.

### RED

Caso novo adicionado a `check_gates.ps1` primeiro; para capturar a falha
genuina, troquei temporariamente o conteudo de `scripts/audit_reports.ps1`
pela versao do commit `6978181` (`git show HEAD:scripts/audit_reports.ps1`,
sem o conserto de N2 -- nada ficou commitado nem sem stage nesse estado, foi
so para rodar o caso antes do fix existir) e rodei
`pwsh -NoProfile -File scripts/check_gates.ps1`:

```
=== audit_reports.ps1 ===
[OK] prosa que nomeia as secoes sem te-las -> 4 SECAO-AUSENTE
[OK] quatro secoes em cabecalho -> 0 SECAO-AUSENTE
[OK] secoes so em comentario de shell dentro de cerca -> 4 SECAO-AUSENTE
[OK] fixture inexistente -> erro, nao 0
[!] cerca nao fechada antes das quatro secoes reais -> 0 SECAO-AUSENTE: esperado '0', obtido '4'

[!] check_gates: 1 de 20 casos reprovados
EXIT=1
```

Exatamente o caso novo falhou, com o obtido `'4'` que a re-revisao mediu.
Restaurei o `audit_reports.ps1` com o conserto de N2 em seguida.

### GREEN

Apos restaurar o conserto de N2:

```
=== pre_commit_docs.ps1 ===
[OK] escotilha em comentario de shell, mensagem sem ela -> deny
[OK] escotilha no arquivo de -F -> allow
[OK] escotilha em -m entre aspas duplas -> allow
[OK] escotilha em -m entre aspas simples -> allow
[OK] escotilha fora das aspas de -m -> deny
[OK] -m sem escotilha, .go sem doc -> deny
[OK] -m sem escotilha, .go com doc -> allow
[OK] -F arquivo inexistente, escotilha no comando -> deny
[OK] sem -m nem -F (editor), .go sem doc -> deny
[OK] -m com escotilha dentro de comentario de shell, -F sem ela -> deny
[OK] dois git commit encadeados, escotilha so no primeiro -> deny
[OK] dois git commit encadeados, escotilha so no ultimo -> allow
[OK] -am com escotilha -> allow
[OK] escotilha dentro de aspas com # no texto -> allow
[OK] motivo do allow e a escotilha, nao o catch do hook

=== audit_reports.ps1 ===
[OK] prosa que nomeia as secoes sem te-las -> 4 SECAO-AUSENTE
[OK] quatro secoes em cabecalho -> 0 SECAO-AUSENTE
[OK] secoes so em comentario de shell dentro de cerca -> 4 SECAO-AUSENTE
[OK] fixture inexistente -> erro, nao 0
[OK] cerca nao fechada antes das quatro secoes reais -> 0 SECAO-AUSENTE

[OK] check_gates: 20 casos
EXIT=0
```

20 casos = 19 da rodada 1 + 1 caso novo de N2.

### Mutation proof

Alterado temporariamente:
```powershell
    if ($false -and $indicesDeCerca.Count % 2 -ne 0) {
```

Saida de `pwsh -NoProfile -File scripts/check_gates.ps1`:
```
[!] cerca nao fechada antes das quatro secoes reais -> 0 SECAO-AUSENTE: esperado '0', obtido '4'

[!] check_gates: 1 de 20 casos reprovados
EXIT=1
```
Exatamente o caso de N2 falhou, com o mesmo `'4'` do RED. Restaurado
(`if ($indicesDeCerca.Count % 2 -ne 0) {`) e reconferido GREEN (20/20, ver
acima). `git diff --stat` apos restaurar:
```
scripts/audit_reports.ps1 | 26 +++++++++++++++++++++++---
scripts/check_gates.ps1   |  5 +++++
2 files changed, 28 insertions(+), 3 deletions(-)
```

### Verification

`pwsh -NoProfile -File scripts/verify.ps1 -SkipCross -SkipNet`, cauda da saida:

```
[...] 1. go build
[OK] go build
[...] 2. go test -race
[OK] go test -race
[...] 3. contagem de testes pulados
[!] 6 testes pulados
     --- SKIP: TestAjudanteSeguraTrava (0.00s)
     --- SKIP: TestListenRestringePermissaoUnix (0.00s)
     --- SKIP: TestSignalCancelsContext (0.10s)
     --- SKIP: TestPerfilDeHeapServindo (0.00s)
     --- SKIP: TestWriteAtomicPreservaOModoDoAlvo (0.00s)
     --- SKIP: TestNew_FailsOnUnwatchablePath (0.00s)
[...] 4. go test (tetos de latencia, sem -race)
[OK] go test (tetos de latencia, sem -race)
[...] 5. go vet (windows)
[OK] go vet (windows)
[i] vet cruzado pulado (-SkipCross)
[...] 6. gofmt
[OK] gofmt
[...] 7. golangci-lint
[OK] golangci-lint
[...] 8. golangci-lint (linux)
[OK] golangci-lint (linux)
[i] check_net pulado (-SkipNet)
[...] 9. check_tool_params
[OK] check_tool_params
[...] 10. check_doc_refs
[OK] check_doc_refs
[...] 11. check_readme_anchors
[OK] check_readme_anchors
[...] 12. check_gates
[OK] check_gates

[OK] Bateria completa. Pode commitar.
EXIT=0
```

Mesmos 6 skips conhecidos, nenhum novo.

`audit_reports.ps1 -Task 187` e `-Task 188` continuam 0 `SECAO-AUSENTE`; e as
fixtures 1, 2 e 3 continuam com o mesmo resultado da rodada 1 (o conserto de
N2 nao muda comportamento quando a contagem de cercas e par):

```
$ pwsh -NoProfile -File scripts/audit_reports.ps1 -Task 187 2>&1 | grep -c 'SECAO-AUSENTE'
0
$ pwsh -NoProfile -File scripts/audit_reports.ps1 -Task 188 2>&1 | grep -c 'SECAO-AUSENTE'
0
$ pwsh -NoProfile -File scripts/audit_reports.ps1 -Task 1 -SddRoot scripts/testdata/gates/sdd-falso 2>&1 | grep -c 'SECAO-AUSENTE'
4
$ pwsh -NoProfile -File scripts/audit_reports.ps1 -Task 2 -SddRoot scripts/testdata/gates/sdd-falso 2>&1 | grep -c 'SECAO-AUSENTE'
0
$ pwsh -NoProfile -File scripts/audit_reports.ps1 -Task 3 -SddRoot scripts/testdata/gates/sdd-falso 2>&1 | grep -c 'SECAO-AUSENTE'
4
```

Nenhum defeito novo. N1 (aspas nao-bash em `Segmento-Commit`) foi deixado
intocado, como instruido -- ja adjudicado como parked/Minor sem exploit
funcional encontrado pela re-revisao.

## Round 3 (F7, N1, F9)

### RED

3 casos F7 e 1 caso N1 adicionados a `check_gates.ps1` ANTES de qualquer
mudanca de comportamento. Saida completa (`pwsh -NoProfile -File
scripts/check_gates.ps1`), 23 de 24:

```
=== pre_commit_docs.ps1 ===
[OK] escotilha em comentario de shell, mensagem sem ela -> deny
[OK] escotilha no arquivo de -F -> allow
[OK] escotilha em -m entre aspas duplas -> allow
[OK] escotilha em -m entre aspas simples -> allow
[OK] escotilha fora das aspas de -m -> deny
[OK] -m sem escotilha, .go sem doc -> deny
[OK] -m sem escotilha, .go com doc -> allow
[OK] -F arquivo inexistente, escotilha no comando -> deny
[OK] sem -m nem -F (editor), .go sem doc -> deny
[OK] -m com escotilha dentro de comentario de shell, -F sem ela -> deny
[OK] dois git commit encadeados, escotilha so no primeiro -> deny
[OK] dois git commit encadeados, escotilha so no ultimo -> allow
[OK] -am com escotilha -> allow
[OK] escotilha dentro de aspas com # no texto -> allow
[OK] motivo do allow e a escotilha, nao o catch do hook
[OK] --amend -m sem escotilha, .go sem doc -> deny
[OK] --amend -m com escotilha, .go sem doc -> allow
[OK] --amend --no-edit, nada em stage -> allow
[!] aspa escapada antes do comentario, -F sem escotilha -> deny: esperado 'deny', obtido 'allow'

=== audit_reports.ps1 ===
[OK] prosa que nomeia as secoes sem te-las -> 4 SECAO-AUSENTE
[OK] quatro secoes em cabecalho -> 0 SECAO-AUSENTE
[OK] secoes so em comentario de shell dentro de cerca -> 4 SECAO-AUSENTE
[OK] fixture inexistente -> erro, nao 0
[OK] cerca nao fechada antes das quatro secoes reais -> 0 SECAO-AUSENTE

[!] check_gates: 1 de 24 casos reprovados
```

Como o brief previu: os 3 casos de F7 ja passavam nesse RED, porque a
excecao de `--amend` vivia FORA do bloco `if (-not $Simular)` e o harness
sempre chama o hook com `-Simular` -- a excecao era, na pratica,
inalcancavel pelo proprio `check_gates`. O RED real de F7 e esse: um
bypass que nenhum caso automatizado via, nao um caso que falhava. O caso
de N1 falhou como esperado (obtido 'allow'): a aspa escapada `\"`
alternava o estado de aspas do walker, o `#` seguinte era lido como
"dentro de aspas" e nao cortava o segmento, deixando o `-m "wip
[sem-doc]"` dentro do trecho analisado.

### GREEN

Apos (A) remover a excecao de `--amend` em `pre_commit_docs.ps1` e (B)
tratar `\` fora de aspas simples como escape em `Segmento-Commit`, 24 de
24:

```
=== pre_commit_docs.ps1 ===
[OK] escotilha em comentario de shell, mensagem sem ela -> deny
[OK] escotilha no arquivo de -F -> allow
[OK] escotilha em -m entre aspas duplas -> allow
[OK] escotilha em -m entre aspas simples -> allow
[OK] escotilha fora das aspas de -m -> deny
[OK] -m sem escotilha, .go sem doc -> deny
[OK] -m sem escotilha, .go com doc -> allow
[OK] -F arquivo inexistente, escotilha no comando -> deny
[OK] sem -m nem -F (editor), .go sem doc -> deny
[OK] -m com escotilha dentro de comentario de shell, -F sem ela -> deny
[OK] dois git commit encadeados, escotilha so no primeiro -> deny
[OK] dois git commit encadeados, escotilha so no ultimo -> allow
[OK] -am com escotilha -> allow
[OK] escotilha dentro de aspas com # no texto -> allow
[OK] motivo do allow e a escotilha, nao o catch do hook
[OK] --amend -m sem escotilha, .go sem doc -> deny
[OK] --amend -m com escotilha, .go sem doc -> allow
[OK] --amend --no-edit, nada em stage -> allow
[OK] aspa escapada antes do comentario, -F sem escotilha -> deny

=== audit_reports.ps1 ===
[OK] prosa que nomeia as secoes sem te-las -> 4 SECAO-AUSENTE
[OK] quatro secoes em cabecalho -> 0 SECAO-AUSENTE
[OK] secoes so em comentario de shell dentro de cerca -> 4 SECAO-AUSENTE
[OK] fixture inexistente -> erro, nao 0
[OK] cerca nao fechada antes das quatro secoes reais -> 0 SECAO-AUSENTE

[OK] check_gates: 24 casos
```

### Mutation proofs

Duas, ambas feitas e restauradas nesta ordem: rodar `check_gates.ps1`
apos a mutacao, colar a saida, desfazer a mutacao, confirmar `git diff
--stat` limpo antes de seguir.

**Proof 1 (F7): re-adicionar a excecao de `--amend` FORA do `-Simular`.**
Inseri de volta as tres linhas originais logo apos `$comando = $Comando`,
antes do `if (-not $Simular)` -- ou seja, alcancaveis pelo harness desta
vez. Rodei `check_gates.ps1`:

```
[!] --amend -m sem escotilha, .go sem doc -> deny: esperado 'deny', obtido 'allow'
...
[!] check_gates: 1 de 24 casos reprovados
```

O caso `--amend -m sem escotilha, .go sem doc -> deny` falhou (obtido
'allow'), exatamente o bypass que a excecao original permitia quando
havia algo em stage. Removi as tres linhas de novo (voltando ao estado
sem excecao alguma) e reconferi GREEN 24/24.

**Proof 2 (N1): remover o branch de escape `\`.** Apaguei a linha com a
condicao do backslash em `Segmento-Commit` e seu comentario. Rodei
`check_gates.ps1`:

```
[!] aspa escapada antes do comentario, -F sem escotilha -> deny: esperado 'deny', obtido 'allow'
...
[!] check_gates: 1 de 24 casos reprovados
```

O caso de N1 falhou (obtido 'allow') -- sem o escape, a aspa escapada
volta a alternar o estado de aspas e o `#` seguinte fica "dentro de
aspas", deixando o `-m "wip [sem-doc]"` no segmento analisado. Restaurei
o branch e reconferi GREEN 24/24.

`git diff --stat` nos dois scripts, apos restaurar as duas mutacoes,
mostrando so a mudanca pretendida desta rodada:

```
scripts/check_gates.ps1     | 17 +++++++++++++++++
scripts/pre_commit_docs.ps1 | 13 ++++++++++---
2 files changed, 27 insertions(+), 3 deletions(-)
```

### Verification

`pwsh -NoProfile -File scripts/verify.ps1 -SkipCross -SkipNet`, tail:

```
[...] 1. go build
[OK] go build
[...] 2. go test -race
[OK] go test -race
[...] 3. contagem de testes pulados
[!] 6 testes pulados
     --- SKIP: TestAjudanteSeguraTrava (0.00s)
     --- SKIP: TestListenRestringePermissaoUnix (0.00s)
     --- SKIP: TestSignalCancelsContext (0.10s)
     --- SKIP: TestPerfilDeHeapServindo (0.00s)
     --- SKIP: TestWriteAtomicPreservaOModoDoAlvo (0.00s)
     --- SKIP: TestNew_FailsOnUnwatchablePath (0.00s)
[...] 4. go test (tetos de latencia, sem -race)
[OK] go test (tetos de latencia, sem -race)
[...] 5. go vet (windows)
[OK] go vet (windows)
[i] vet cruzado pulado (-SkipCross)
[...] 6. gofmt
[OK] gofmt
[...] 7. golangci-lint
[OK] golangci-lint
[...] 8. golangci-lint (linux)
[OK] golangci-lint (linux)
[i] check_net pulado (-SkipNet)
[...] 9. check_tool_params
[OK] check_tool_params
[...] 10. check_doc_refs
[OK] check_doc_refs
[...] 11. check_readme_anchors
[OK] check_readme_anchors
[...] 12. check_gates
[OK] check_gates

[OK] Bateria completa. Pode commitar.
EXIT=0
```

Mesmos 6 skips conhecidos, nenhum novo.

UTF-8 validado nos dois `.md` tocados nesta rodada:

```
python -c "open('docs/ESTADO.md',encoding='utf-8').read()" && echo "[OK] UTF-8 valido"
[OK] UTF-8 valido
python -c "open('docs/papeis/documentador.md',encoding='utf-8').read()" && echo "[OK] UTF-8 valido"
[OK] UTF-8 valido
```

`audit_reports.ps1 -Task 187` e `-Task 188` continuam 0 `SECAO-AUSENTE`
(a remocao da excecao de `--amend` e o escape de backslash nao tocam
`audit_reports.ps1`, entao isso confirma so ausencia de regressao):

```
pwsh -NoProfile -File scripts/audit_reports.ps1 -Task 187 -SddRoot .superpowers/sdd 2>&1 | grep -c SECAO-AUSENTE
0
pwsh -NoProfile -File scripts/audit_reports.ps1 -Task 188 -SddRoot .superpowers/sdd 2>&1 | grep -c SECAO-AUSENTE
0
```

F9: `docs/ESTADO.md:637` (357 colunas) foi requebrada em 5 linhas de ate
82 colunas, mantendo a indentacao de 2 espacos das vizinhas e nenhuma
palavra alterada (medido com `awk '{ print NR": "length($0) }'
docs/ESTADO.md`: nenhuma linha do arquivo passa de 150 colunas fora do
bloco de tabelas ja existente nas linhas 198-206, que nao foi tocado).

Nenhum achado novo. F7, N1 e F9 fechados -- nenhuma das tres pendencias
parqueadas da revisao final segue aberta.
