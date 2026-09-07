# Revisao final — 20f97d5..f3e3b15 (Tasks 186, 187, 188)

Revisor: agente `rev-final-gates`. Somente leitura: nada foi editado, nada foi
commitado. Os desvios ja adjudicados (`-EncodedCommand`, `\bred\b`,
`OPERACAO.md` intocado, 79 -> 203) nao sao re-reportados como defeito; F8 toca
o ultimo apenas para propor que o numero ganhe a data e o tamanho do corpus.

## Verdict

APPROVE_WITH_NITS

A mudanca faz o que promete e e **estritamente mais forte** que o que
substituiu, nos dois gates. O passo 15 esta ligado de verdade (medido, ver
`## Verified`), as provas de mutacao dos dois relatorios sao reais e a conta de
"15 etapas" bate. As quatro descobertas Minor abaixo sao buracos residuais, nao
regressoes: F1/F2 sao a mesma classe do bypass fechado, sobrevivendo numa forma
mais estreita; F4 e um caso de teste que nao consegue falhar sozinho; F5 e um
falso-positivo de "secao presente". Nenhuma bloqueia o merge; F1, F4 e F5
merecem uma tarefa de acompanhamento.

## Findings

### F1 — `Extrair-Mensagem` le a linha inteira, entao um `-m` dentro de um comentario de shell volta a valer

**Local:** `scripts/pre_commit_docs.ps1:76-91` (funcao), `:106-109` (uso).
**Severidade:** Minor.

`[regex]::Matches($linha, $reM)` varre a linha de comando **inteira**, sem
ancorar em `git commit` e sem parar no `#` que abre um comentario de shell. Logo
qualquer texto com a forma `-m "..."` em qualquer lugar da linha e aceito como
"a mensagem".

Cenario medido nesta revisao (hook chamado em `-Simular`, `-EmStage
internal/index/resolve.go`):

```
allow  <- git commit -F scripts/testdata/gates/msg-sem-escotilha.txt # was: -m "wip [sem-doc]"
```

A mensagem efetivamente commitada e `msg-sem-escotilha.txt`, que **nao** tem a
escotilha; o `allow` veio do texto dentro do comentario.

**Isto importa para o proposito deste hook?** Importa, e e a unica descoberta
sobre a qual eu diria isso. O gate nao e fronteira de seguranca — ninguem aqui
esta tentando burlar nada —, mas a razao de ser da Task 187 esta escrita no
proprio comentario do hook (`:66-71`): a escotilha tem de ficar **visivel no
historico**. Texto que mora num comentario de shell nunca entra no historico.
O bypass de 2026-09-06 aconteceu porque um brief mandou anexar `# [sem-doc]` a
linha; um brief que mandasse anexar `# retry de: git commit -m "... [sem-doc]"`
passaria pelo gate novo do mesmo jeito, com a mesma inocencia.

**Conserto sugerido:** recortar a linha antes de extrair — do token `git commit`
ate o primeiro `#` fora de aspas (e ate `&&`, `;`, `|`) — e so entao rodar os
dois regex sobre o pedaco restante.

### F2 — dois `git commit` na mesma linha: a escotilha do primeiro cobre o segundo

**Local:** `scripts/pre_commit_docs.ps1:76-91`.
**Severidade:** Minor.

Consequencia direta de F1, mas com um cenario proprio e mais plausivel que o
comentario:

```
allow  <- git commit -m "docs: a [sem-doc]" && git commit -m "feat: b"
```

Os dois `-m` sao concatenados em `$partes`, a escotilha do primeiro e achada, e
o hook devolve `allow` para uma chamada que faz **dois** commits — o segundo sem
escotilha nenhuma. O hook recebe a linha inteira em `tool_input.command`, entao
essa e a forma que o harness realmente entrega quando o modelo encadeia commits.

**Conserto sugerido:** o mesmo de F1 — considerar apenas o segmento do **ultimo**
`git commit` da linha.

### F3 — formas de `git commit` que o extrator nao le (todas falham fechadas)

**Local:** `scripts/pre_commit_docs.ps1:78` (`$reM`), `:83` (`$reF`); texto do
deny em `:163-166`.
**Severidade:** Minor.

Medido, mesma bancada:

```
deny   <- git commit -am "fix: x [sem-doc]"            (flags curtas coladas: nao ha "-m" na linha)
deny   <- git commit -m "fix: say \"hi\" [sem-doc]"    (aspa escapada trunca o grupo "([^"]*)")
deny   <- git commit -F - <<EOF ... [sem-doc] ... EOF  (heredoc: "-" nao e Leaf)
deny   <- git commit -C HEAD                           (mensagem reusada, invisivel ao hook)
deny   <- git commit --fixup=abc1234
deny   <- git commit --amend --no-edit
deny   <- cd scripts && git commit -F testdata/gates/msg-com-sem-doc.txt   (-F relativo a cwd errada)
deny   <- git commit -F C:\...\msg com espaco.txt      (caminho com espaco SEM aspas; com aspas -> allow)
```

Todas falham **fechadas**: o gate nega, nao libera. Para o proposito deste hook
isso e a direcao certa do erro, e a convencao do projeto (`git commit -F
<arquivo>`, ditada pelo proprio plano) evita quase todas. O incomodo real e o
laco de `-am`: o autor **escreveu** `[sem-doc]` na mensagem, o hook nega, e o
texto do deny (`:163-166`) manda "inclua `[sem-doc]` na MENSAGEM do commit (-m ou
arquivo de -F)" — instrucao que ele ja cumpriu.

**Conserto sugerido:** aceitar o agrupamento de flags curtas trocando `-m` por
`(?<![\w-])-[a-zA-Z]*m` em `$reM`, e acrescentar ao texto do deny que o hook so
consegue ler `-m`/`--message=` e o arquivo de `-F`/`--file=`.

### F4 — o caso "quatro secoes em cabecalho -> 0" passa vazio quando o auditor nem chega a varrer

**Local:** `scripts/check_gates.ps1:113-116` (`Secoes-Ausentes`), usado em
`:157-158`.
**Severidade:** Minor.

`Secoes-Ausentes` conta `SECAO-AUSENTE` em `stdout+stderr` e **ignora o codigo
de saida**. `audit_reports.ps1` tem dois `exit 2` precoces (`:62-65` raiz
inexistente, `:139-142` nenhum relatorio casou) cuja saida nao contem a string —
zero acertos, que e exatamente o valor esperado do caso.

Medido nesta revisao:

```
pwsh -NoProfile -File scripts/audit_reports.ps1 -Task 2 -SddRoot scripts/testdata/gates/nao-existe
exit=2 ; SECAO-AUSENTE matches = 0
[!] Diretorio de artefatos nao encontrado: scripts/testdata/gates/nao-existe

pwsh -NoProfile -File scripts/audit_reports.ps1 -Task 9999 -SddRoot scripts/testdata/gates/sdd-falso
exit=2 ; SECAO-AUSENTE matches = 0
[!] Nenhum relatorio casou com 'task-9999-report.md' em scripts/testdata/gates/sdd-falso
```

Ou seja: apagar ou renomear `task-2-report.md`, mover a pasta de fixtures, ou
quebrar o auditor antes da varredura deixa esse caso **verde**. E o mesmo
acidente que o proprio relatorio da Task 188 registrou no RED (`obtido '0'` lido
como `[OK]`) — mas la foi transitorio, e aqui ficou no codigo entregue.

Atenuante que impede a severidade de subir: o caso irmao espera `'4'`, e uma
quebra sistemica derruba **ele**, entao `check_gates` como um todo nao fica cego.
O que nao existe e um caso capaz de falhar sozinho quando so o fixture 2 some.

**Conserto sugerido:** em `Secoes-Ausentes`, `if ($LASTEXITCODE -eq 2) { return
'erro' }` antes de contar — ou exigir que a saida contenha `=== Relatorios (1) ===`.

### F5 — cabecalho exigido, mas comentario de shell dentro de bloco cercado tambem casa

**Local:** `scripts/audit_reports.ps1:129-134` (`$Required`), aplicado em
`:162-166`.
**Severidade:** Minor.

`(?im)^#{1,6}\s.*muta` (e as tres irmas) casa **qualquer** linha que comece com
`#` seguido de espaco, inclusive um comentario de shell dentro de um bloco
` ```bash `. Relatorios deste projeto colam saida de comando o tempo todo, e
comentario iniciado por `#` e a forma normal de anotar essa saida.

Medido nesta revisao, sobre um corpo cujas unicas linhas iniciadas por `#`
(alem do titulo) estao dentro de uma cerca:

```
## What I ran
```bash
# rodei o gate completo antes de commitar
pwsh -File scripts/verify.ps1
# mutation: apaguei a regra e rodei de novo
# red -> green depois do conserto
```

RED      match=True
GREEN    match=True
Mutacao  match=True
Verif    match=True
```

Quatro secoes "presentes" num relatorio que nao tem nenhuma.

**Isto e regressao?** Nao. Todo texto que casa o padrao novo tambem casava o
antigo (`(?i)muta`), entao o gate so apertou. E um buraco **residual**, e o
julgo Minor por isso — mas e um buraco que a prosa de um relatorio real tropeça
sem querer, que e precisamente o modo de falha que a Task 188 existe para fechar.

**Conserto sugerido:** remover os blocos cercados de `$Body` antes de casar
(alternar um flag ao ver `^\s*```` ` `` ` `), como uma passada so antes do
`foreach ($Sec in $Required)`.

### F6 — `Decisao-Hook` le so a decisao, e hook quebrado responde `allow`

**Local:** `scripts/check_gates.ps1:35-52`; catch-all do hook em
`scripts/pre_commit_docs.ps1:171-173`.
**Severidade:** Nit.

O hook tem, de proposito e documentado (`:38-40`), um catch que devolve `allow`
com exit 0 quando ele proprio falha — "hook quebrado nao pode virar repositorio
travado". `Decisao-Hook` le apenas `permissionDecision`, entao um hook quebrado
em qualquer ponto devolve `allow` e os **quatro** casos que esperam `allow`
passam.

Medido: copia do hook no scratchpad com um `throw` injetado no topo do `try`
(nada no repositorio foi tocado):

```
exit=0
{"hookSpecificOutput":{"permissionDecision":"allow","permissionDecisionReason":"pre_commit_docs.ps1 falhou: falha simulada","hookEventName":"PreToolUse"}}
```

Os cinco casos de `deny` continuam reprovando, entao `check_gates` sai 1 e o
gate inteiro reprova — por isso Nit, nao Minor.

**Conserto sugerido:** ao menos um caso conferindo tambem
`permissionDecisionReason -eq 'escotilha [sem-doc] na mensagem do commit'`.

### F7 — `-Simular` pula o roteamento de `--amend`, que continua sendo `allow` incondicional

**Local:** `scripts/pre_commit_docs.ps1:95-104`.
**Severidade:** Nit (comportamento pre-existente, nao introduzido aqui).

Todo o bloco `if (-not $Simular)` — o filtro `git\s+commit` e o desvio de
`--amend` — e inalcancavel pelos casos de `check_gates`, porque eles sempre
passam `-Simular`. Consequencia a registrar: `git commit --amend` **sem**
`--no-edit` devolve `allow` incondicional (`:101-103`), ou seja, e uma passagem
livre pelo gate (staged um `.go` sem doc, `--amend`, pronto). Isso e anterior a
esta mudanca e nao e defeito do diff; anoto para que ninguem confunda "coberto
por check_gates" com "coberto".

**Conserto sugerido:** nenhum agora. Se virar tarefa, o caminho e simular tambem
o payload de stdin (um caso com JSON completo) em vez de so `-Comando`.

### F8 — `79` -> `203` e um retrato de um corpus que ja se mexeu

**Local:** `docs/ESTADO.md:620-621`.
**Severidade:** Nit.

Re-medi hoje com o mesmo comando do relatorio
(`pwsh -NoProfile -File scripts/audit_reports.ps1 2>&1 | grep -c 'SECAO-AUSENTE'`):
**199**, sobre `=== Relatorios (150) ===`. A diferenca de 4 tem explicacao
benigna e verificada: `task-188-report.md` estava incompleto quando o 203 foi
tomado (11:01) e hoje tem as quatro secoes em cabecalho — `audit_reports.ps1
-Task 188` devolve 0 achados de `SECAO-AUSENTE`, e `-Task 187` tambem. Nada
falso foi escrito; o numero era verdadeiro quando medido.

O risco e de leitura: um par de numeros solto sobre um corpus que cresce convida
o proximo leitor a "corrigir" o 203.

**Conserto sugerido:** "medido em 2026-09-07 sobre 150 relatorios: `79` ->
`203`".

### F9 — linha de 357 caracteres num item que quebra em ~80

**Local:** `docs/ESTADO.md:635`.
**Severidade:** Nit.

As linhas 632-639 do mesmo item quebram entre 14 e 84 colunas; a 635 tem 357. O
arquivo ja tem 11 outras linhas longas (114, 197-206), entao nao ha regra
absoluta violada — e o plano ditou a frase como uma linha so. Puramente
cosmetico.

**Conserto sugerido:** quebrar em ~80 quando alguem passar por ali.

### F10 — `documentador.md` promete um pouco mais do que o codigo entrega

**Local:** `docs/papeis/documentador.md:139-141`.
**Severidade:** Nit.

"A escotilha `[sem-doc]` vale na MENSAGEM do commit (`-m` ou o arquivo de `-F`),
**nunca na linha de comando fora dela**" — a segunda metade e mais forte que o
comportamento medido em F1/F2. O resto da mudanca em `documentador.md` (as
quatro secoes so contando em cabecalho, `:48` em diante) confere com
`audit_reports.ps1:129-134`, incluindo o intervalo `#` a `######`.

**Conserto sugerido:** casar com F1 — ou consertar o extrator, ou escrever "so
onde o hook consegue ler a mensagem: `-m`/`--message=` e o arquivo de
`-F`/`--file=`".

## Verified

Tudo abaixo foi rodado nesta revisao, em `C:\Users\jonyd\Projetos\Gobsidian`.

**1. `check_gates.ps1` — 11/11, EXIT=0**

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

=== audit_reports.ps1 ===
[OK] prosa que nomeia as secoes sem te-las -> 4 SECAO-AUSENTE
[OK] quatro secoes em cabecalho -> 0 SECAO-AUSENTE

[OK] check_gates: 11 casos
EXIT=0
```

**2. `audit_reports.ps1 -Task 2 -SddRoot scripts/testdata/gates/sdd-falso`**

```
=== Relatorios (1) ===

=== Ledger ===

[OK] Nada a conferir.
EXIT=0
```

O `=== Relatorios (1) ===` confirma que o fixture 2 foi de fato varrido — a
diferenca entre o `0` legitimo e o `0` vazio de F4.

**3. "15 etapas" (CLAUDE.md:207) — confere.** `grep -n 'Invoke-Step "'
scripts/verify.ps1` devolve **14** chamadas (linhas 92, 100, 173, 180, 183, 187,
196, 209, 228, 239, 245, 252, 253, 258) e `verify.ps1:129-130` incrementa
`$script:StepNumber` a mao para "contagem de testes pulados". 14 + 1 = 15.

**4. O passo 15 pode reprovar de verdade.** `Invoke-Step` le `$LASTEXITCODE`
depois de `& $Body` (`verify.ps1:55-61`), e um `.ps1` chamado com `&` propaga o
`exit`:

```
LASTEXITCODE apos & script com exit 1 = 1
```

Somado a prova de mutacao da Task 188 (com `'(?i)muta'` de volta, `check_gates`
sai 1), o passo novo nao e decorativo.

**5. `$Body` e seguro para `(?m)^`, inclusive com CRLF.**
`audit_reports.ps1:146-147`: `Get-Content -Path ... -Encoding utf8` devolve as
linhas **sem** os terminadores (PowerShell 7 corta `\r\n`, `\n` e `\r`), e o
join e literalmente `$Lines -join "`n"`. Nao sobra `\r` no fim de linha nenhuma,
entao `^#{1,6}\s` e `\bred\b` casam igual em arquivo CRLF e LF.

**6. Os outros gates de doc continuam verdes.**

```
check_doc_refs EXIT=0
check_readme_anchors EXIT=0
```

**7. Numeros citados existem onde a citacao diz.** `docs/PRD.md:208` cita "267
delas, outro 372 (medicao em `docs/ESTADO.md`)"; `grep -n '267\|372'
docs/ESTADO.md` -> linhas 224-225: "no cofre *Estudo*, **267 alvos comecando com
`#` e 10 vazios**; no *Oral*, **372 com `#`**".

**8. A afirmacao nova do RF-63 sobre `%23` confere com o codigo.**
`internal/parser/ast.go:97-104`: "Roda ANTES de PercentDecode nos ramos
Markdown, e a ordem e a regra: `%23` e um `#` que faz parte do NOME do arquivo."
O PRD nao inventou comportamento.

**9. Global Constraints do plano.**

- Nenhum `.go` no intervalo: os 12 arquivos do pacote de revisao sao `.md`,
  `.ps1` e `.txt`.
- Conventional Commits em ingles nos tres commits, com **os dois** trailers
  (`Co-Authored-By: Claude Opus 5` e `Claude-Session: ...`) — conferido em
  `git log --format='%H%n%s%n%b' 20f97d5..f3e3b15`.
- Marcadores de console em ASCII puro: `check_gates.ps1` so emite `[OK]` e
  `[!]`; nenhum caractere fora de ASCII em string de saida (os travessoes estao
  em comentarios, como ja acontecia em `audit_reports.ps1`).
- Numero sem medicao: nao achei nenhum. `79`/`203` vem do Step 7 do relatorio
  188 (ver F8), `2420`/`2257` bytes das fixtures vem de `(Get-Item).Length`, e
  os zeros dos cinco cofres vem da secao Spec do plano.

**O que eu NAO rodei:** `scripts/verify.ps1` inteiro (os dois relatorios colam
`EXIT=0` com 15 etapas; rodar de novo custaria minutos e nao mudaria nenhuma
descoberta) e `scripts/test_orphans.ps1` (fora do escopo, e nada neste intervalo
toca processo).
