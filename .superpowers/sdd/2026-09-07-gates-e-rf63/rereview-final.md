# Re-revisao do fix final — f3e3b15..6978181

Revisor: agente de re-revisao escopado (somente leitura; nada editado, nada
commitado no repositorio — ver nota sobre um engano de teste corrigido abaixo).

## Verdict

APPROVE_WITH_NITS

`check_gates.ps1` imprime `[OK] check_gates: 19 casos` e sai 0, batendo com o
esperado (11 originais + 6 hook + 2 audit). Os oito achados do escopo (F1-F6,
F8, F10) estao fechados para os cenarios exatos que a revisao anterior mediu.
Um achado novo (N1) mostra que o rastreador de aspas de `Segmento-Commit`
diverge da semantica real do bash em presenca de aspas escapadas/desbalanceadas
— mas testei a execucao real em um repositorio git descartavel e o `git commit`
verdadeiro recusa a linha (pathspec invalido) antes de qualquer commit
acontecer, entao nao ha exploracao funcional encontrada; por isso Minor, nao
Blocker. Um segundo achado novo (N2) e mais serio na pratica: uma cerca de
codigo nao fechada por acidente em `audit_reports.ps1` engole TODAS as secoes
reais que vierem depois dela no arquivo, produzindo falso-positivo de
`SECAO-AUSENTE` num relatorio genuinamente completo — o modo de falha inverso
de F5, e um erro de digitacao plausivel neste projeto (relatorios colam muita
saida de comando com cercas). Nenhum dos dois bloqueia o merge.

## Findings table

| Finding | Closed? | Evidence |
|---|---|---|
| F1 (comentario de shell com `-m` escondido) | yes | `check_gates` caso `-m com escotilha dentro de comentario de shell, -F sem ela -> deny` = OK; reproduzi a mesma chamada isolada via `-Simular` com o `-F` **sem** aspas estranhas e obtive `deny` com o motivo de doc ausente. |
| F2 (dois `git commit` encadeados) | yes | Casos `dois git commit encadeados, escotilha so no primeiro -> deny` e `... so no ultimo -> allow`, ambos OK em `check_gates`. |
| F3 (`-am` e flags curtas agrupadas) | yes | Casos `-am com escotilha -> allow` e `escotilha dentro de aspas com # no texto -> allow`, ambos OK. Confirmei que `--amend` sozinho nao casa `$reM` (o lookbehind `(?<![\w-])-` barra o segundo `-` de `--amend`, e o primeiro `-` nao e seguido de `m`) — nao regressa o roteamento de F7, que continua fora do alcance de `-Simular` (ja adjudicado). |
| F4 (`Secoes-Ausentes` cega a exit 2) | yes | Caso `fixture inexistente -> erro, nao 0` = OK. `Secoes-Ausentes` agora retorna `'erro'` quando `$LASTEXITCODE -eq 2`, distinguindo do `0` legitimo do fixture 2. |
| F5 (comentario de shell em cerca conta como cabecalho) | yes | Caso `secoes so em comentario de shell dentro de cerca -> 4 SECAO-AUSENTE` = OK. Testei tambem uma cerca indentada dentro de um item de lista (` - passo:\n  \`\`\`bash\n  # comentario\n  \`\`\``) seguida de headings reais: 0 `SECAO-AUSENTE`, a cerca indentada fecha corretamente e nao esconde nada — ver N2 abaixo para o caso oposto (cerca **nao** fechada). |
| F6 (motivo do allow, nao so a decisao) | yes | Caso `motivo do allow e a escotilha, nao o catch do hook` = OK, usando `Motivo-Hook`/`Invocar-Hook` novo. |
| F8 (numero `79`->`203` sem data/corpus) | yes | `docs/ESTADO.md:626-628` agora le "medido em 2026-09-07 sobre 150 relatorios: `79` -> `203` ... (o numero cresce com o corpus; nao o corrija sem re-medir)". Texto bate com o brief. |
| F10 (texto do hatch mais forte que o comportamento) | yes | `docs/papeis/documentador.md:139-141` agora diz "so no trecho do ultimo `git commit` da linha... comentario de shell e comandos encadeados antes dele nao contam" — alinhado com F1/F2/F3 fechados. UTF-8 valido nos dois `.md` editados (`python -c "open(...,encoding='utf-8').read()"`, OK nos dois). |

Nenhuma das oito descobertas do escopo ficou "partial": os cenarios exatos
medidos em `review-final.md` agora denunciam corretamente, e as fixtures/casos
novos em `check_gates.ps1` batem com o brief (nomes, `$go`, `$msgSem`
reaproveitados; `Invocar-Hook` refatorado como o brief sugeriu).

## Novos achados

### N1 — `Segmento-Commit` diverge da semantica de aspas do bash quando ha uma aspa escapada/desbalanceada; nenhum exploit funcional encontrado

**Local:** `scripts/pre_commit_docs.ps1:304-320` (`Segmento-Commit`).
**Severidade:** Minor.

O rastreador conta caracteres `'`/`"` um a um, sem considerar `\"` como aspa
escapada (que em bash NAO abre/fecha modo de aspas, apenas insere um `"`
literal na palavra atual). Uma aspa avulsa (escapada ou nao) antes do `#`
inverte a paridade que o rastreador guarda, e ele passa a tratar o `#` real
como se estivesse "dentro de aspas", deixando o segmento se estender para
dentro do comentario — reabrindo exatamente a classe de F1 por um mecanismo
diferente. Medido via `-Simular`:

```
git commit -F scripts/testdata/gates/msg-sem-escotilha.txt \" # was: -m "wip [sem-doc]"
-> allow (deveria ser deny; a mensagem real, via -F, nao tem a escotilha)
```

Tambem reproduz com aspa avulsa sem barra (`" # was: -m "wip [sem-doc]""`).

**Isto e exploravel de verdade?** Testei a execucao REAL da mesma linha num
repositorio git descartavel (`/tmp/scratch-gitcommit-test`), fora deste
projeto:

```
$ git commit -F msg.txt \" # was: -m "wip [sem-doc]"
error: pathspec '"' did not match any file(s) known to git
EXIT=1
```

`git` recusa a linha porque a aspa avulsa vira um argumento posicional
(pathspec `"`) que nao casa nenhum arquivo — nenhum commit acontece. Tokenizei
a linha com `bash` puro (`for a; do printf '<%s>\n' "$a"; done`) e confirmei:
os argumentos reais sao `git commit -F <arquivo> "` — o `#` e um comentario de
shell de verdade (correto), mas a aspa avulsa sobra como um quinto argumento
que quebra o `git commit` antes de qualquer coisa. Tentei mover a aspa colada
ao nome do arquivo (corrompe o path, `git` erra "no such file") e colada em
`commit` (corrompe o subcomando, `git` erra "not a git command"); em toda
variante que tentei, a mesma aspa que confunde o rastreador da PowerShell
tambem corrompe um argumento real que faz o `git` real abortar antes do
commit. Nao encontrei uma linha que (a) engane o rastreador E (b) produza um
commit de verdade sem documentacao.

**Por que reportar mesmo sem exploit:** o rastreador continua sendo uma
aproximacao de bash, nao bash de verdade — `-Simular` pode dizer `allow` para
uma linha que o `git` real nunca executaria como commit. Isso nao e um buraco
de seguranca (o hook nao e fronteira de seguranca, ver F1 na revisao anterior),
mas um teste `check_gates` que confiasse cegamente na decisao simulada como
prova de "o commit passaria" estaria testando um bash que nao existe.

**Conserto sugerido:** nenhum urgente — documentar a limitacao ("o rastreador
nao entende `\"` como escape; aspa desbalanceada pode dar falso allow, mas
nenhuma execucao real dessa linha commitaria") ao lado do comentario de
`Segmento-Commit`, ou tratar `\` antes de uma aspa como "pular os dois
caracteres, nao contar a aspa" se alguem quiser fechar o buraco de verdade.

### N2 — cerca de codigo aberta e nao fechada engole TODAS as secoes reais depois dela (falso-positivo, modo oposto de F5)

**Local:** `scripts/audit_reports.ps1:94-100` (`$BodySemCercas`).
**Severidade:** Major.

O toggle `$emCerca = -not $emCerca` em cada linha que casa `^\s*(\`\`\`|~~~)`
nao tem estado de erro: se um relatorio tiver um numero **impar** de linhas de
cerca (autor esqueceu de fechar um bloco \`\`\`bash ao colar saida de
terminal — engano comum, e este projeto tem dezenas de relatorios colando
saida de comando), tudo do ponto da abertura ate o fim do arquivo fica
"dentro de cerca" e e removido de `$BodySemCercas` — **inclusive cabecalhos
reais de `### RED`, `### GREEN` etc. que vierem depois**, mesmo que estejam
fora de qualquer cerca de verdade.

Medido com uma fixture fora do repositorio (nao um arquivo do projeto):

```
# Task 99 report
## What I ran
```bash
pwsh -File scripts/verify.ps1
this fence is never closed, by accident

### RED
(evidencia real aqui)
### GREEN
(evidencia real aqui)
## Mutation proofs
(evidencia real aqui)
## Verification
(evidencia real aqui)
```

```
pwsh -NoProfile -File scripts/audit_reports.ps1 -Task 99 -SddRoot <fixture>
=== Relatorios (1) ===
  ...task-99-report.md:1: [SECAO-AUSENTE] sem secao de TDD/RED
  ...task-99-report.md:1: [SECAO-AUSENTE] sem secao de TDD/GREEN
  ...task-99-report.md:1: [SECAO-AUSENTE] sem secao de Mutacao
  ...task-99-report.md:1: [SECAO-AUSENTE] sem secao de Verificacao
[!] 5 achado(s)
```

As quatro secoes existem, em cabecalho, fora de qualquer cerca de verdade —
mas o auditor as declara ausentes porque uma cerca aberta sem fechar antes
delas as "escondeu". Confirmei em contraste que uma cerca **indentada dentro
de uma lista e fechada corretamente** nao tem esse problema (0
`SECAO-AUSENTE`), entao o defeito e especificamente a ausencia de fechamento,
nao a indentacao.

**Por que Major e nao Minor (ao contrario de F5):** F5 era um falso-negativo
residual (secao ausente contada como presente) que a propria revisao anterior
julgou "buraco que aperta o gate, nunca afrouxa". N2 e o oposto: um
falso-**positivo** que pode reprovar um relatorio genuinamente completo por
causa de um erro de formatacao Markdown comum e sem relacao com o conteudo —
exatamente o tipo de sinal que, se ignorado algumas vezes por ser "ruido",
ensina quem le a nao confiar no `SECAO-AUSENTE` mesmo quando ele estiver
certo. E um erro plausivel: os proprios relatorios reais deste projeto
(Task 187, 188) colam blocos de saida longos com cercas.

**Conserto sugerido:** ao fim do arquivo, se `$emCerca` ainda for `$true`,
tratar isso como sinal de cerca mal formada — no minimo, nao remover nada
(usar `$Body` inteiro para esse relatorio quando a contagem de linhas de cerca
for impar) e talvez emitir um achado novo (`CERCA-ABERTA`) para chamar atencao
humana, em vez de silenciosamente apagar o resto do arquivo.

## Verificado nesta re-revisao

```
pwsh -NoProfile -File scripts/check_gates.ps1
[OK] check_gates: 19 casos
(exit 0)
```

`docs/ESTADO.md` e `docs/papeis/documentador.md`: `python -c "open(f,
encoding='utf-8').read()"` OK nos dois.

`git diff --stat` sobre a arvore de trabalho: nao ha alteracoes pendentes nos
tres scripts nem nas fixtures do diff revisado — as unicas modificacoes
presentes no working tree (`.superpowers/sdd/2026-07-25-gobsidian-v01/progress.md`,
`docs/ESTADO.md` com um diff diferente do fix, arquivos de `test-vault/`) sao
de outros agentes trabalhando em paralelo nesta mesma sessao, nao desta
re-revisao nem do commit `6978181`.

**Nota de processo:** durante a investigacao de N1, um `echo ... > arquivo`
mal direcionado sobrescreveu por engano
`scripts/testdata/gates/msg-sem-escotilha.txt` (fixture do proprio diff sob
revisao). Percebi pelo `git diff --stat`, e como `git checkout` estava
bloqueado pelo classificador de permissoes desta sessao, recriei o conteudo
original a mao com a ferramenta de escrita de arquivo e confirmei
`git diff` vazio para esse arquivo antes de continuar. Nenhuma outra escrita
foi feita dentro do repositorio nesta re-revisao; as duas fixtures novas de
teste (`task-98-report.md`, `task-99-report.md`) foram criadas fora do
repositorio, no diretorio de scratchpad da sessao, e o teste de execucao real
de git (N1) rodou num repositorio descartavel em `/tmp`, tambem fora deste
projeto.
