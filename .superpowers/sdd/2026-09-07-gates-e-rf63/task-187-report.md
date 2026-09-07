## Progresso

- 10:27 -- Início. Lido o brief inteiro, o hook atual (`scripts/pre_commit_docs.ps1`), `verify.ps1` em torno da linha 253, `CLAUDE.md` em torno da linha 207, `docs/papeis/documentador.md` em torno da linha 143. Confirmado com `grep -n 'check_readme_anchors' docs/OPERACAO.md` que o arquivo NÃO lista as etapas do `verify.ps1` (grep sem match) -- registrando "não lista", não vou inventar seção lá. Confirmado com `git grep -n 'pre_commit_docs' -- '*.md' '*.ps1' '*.json'` que nenhuma chamada real usa `-Mensagem`; as menções em relatórios antigos são narrativas do bypass antigo (linha de comando), não invocações do parâmetro. Criado `scripts/testdata/gates/`.
- 10:28 -- Step 1: `scripts/check_gates.ps1` e as duas fixtures escritos, verbatim do brief (com o try/catch da regra R1 do orquestrador em `Decisao-Hook`, checando `$LASTEXITCODE` e envolvendo o `ConvertFrom-Json`).
- 10:29 -- Step 2 (RED): `pwsh -File scripts/check_gates.ps1` antes de tocar o hook. 9 de 9 `[!]`, `EXIT=1`, todos com "obtido 'erro'" (hook nao conhece `-Comando`/`-EmStage`). Saida colada em `### RED` abaixo.
- 10:30-10:32 -- Step 3: hook `scripts/pre_commit_docs.ps1` alterado -- param `-Comando`/`-EmStage` no lugar de `-Mensagem`, funcao `Extrair-Mensagem`, leitura de `$Comando` em vez de `$Mensagem`, `$emStage` condicional a `-Simular`, texto do `deny` e do `.DESCRIPTION` atualizados, tudo verbatim do brief.
- 10:33 -- Step 4 (GREEN), primeira tentativa: 8 de 9 `[OK]`, 1 `[!]` (`-m sem escotilha, .go com doc -> allow`, obtido `erro`). Investigado: `pwsh -File $Hook -EmStage $EmStage` com `$EmStage` de 2 elementos nao chega como array do outro lado do processo -- `-File` repassa argumentos como argv cru, entao so o primeiro elemento faz bind em `-EmStage` e o segundo vira parametro posicional sem casa, e o hook (com `[CmdletBinding()]`) reprova com "A positional parameter cannot be found that accepts argument 'docs/TOOLS.md'." Reproduzido isolado em script minimo (`[string[]]$Foo` + `-Foo "a" "b"` via `pwsh -File` -> mesmo erro; `-Foo a,b` sem espaco -> vira string literal "a,b", nao array). Corrigido `Decisao-Hook` em `check_gates.ps1`: em vez de `-File`, monta um `& '<hook>' -Simular -Comando '<comando>' -EmStage @('<a>','<b>')` como texto e roda via `pwsh -NoProfile -EncodedCommand <base64>` -- assim quem le a linha e o parser do PowerShell (que entende `@(...)` como array de verdade), nao o passa-argumento cru do `-File`. Validado isolado antes de aplicar (script de teste em scratchpad, removido depois). Isso e um desvio do codigo literal do brief para `Decisao-Hook` (o `& pwsh -NoProfile -File $Hook -Simular -Comando $Comando -EmStage $EmStage` do brief tem esse bug de binding, medido, nao suposto); a assinatura, os nomes `Caso`/`$Fixtures`/`$ProjectRoot` e o contrato de saida ficaram identicos.
- 10:34 -- Step 4 (GREEN), segunda tentativa, com a correcao: 9 de 9 `[OK]`, `EXIT=0`. Saida colada em `### GREEN` abaixo.
- 10:34-10:35 -- Step 5 (prova de mutacao): revertida a mao a linha `$mensagem = Extrair-Mensagem $comando` para `$mensagem = $comando`. Rodado `check_gates.ps1`: 4 de 9 casos reprovaram (`EXIT=1`), incluindo os tres que o brief nomeou ("escotilha em comentario de shell", "escotilha fora das aspas", "-F inexistente") mais um quarto ("escotilha no arquivo de -F -> allow" tambem regride, porque com a mutacao a mensagem nunca e extraida do arquivo de `-F`). Linha restaurada a mao. Rodado de novo: 9 de 9 `[OK]`, `EXIT=0`. `git diff --stat -- scripts/pre_commit_docs.ps1` depois de restaurar mostra soh o diff esperado do Step 3 (42 insercoes, 9 remocoes) -- nao sobrou residuo da mutacao.
- 10:35 -- Step 6: `Invoke-Step "check_gates"` adicionado em `scripts/verify.ps1` logo apos `check_readme_anchors` (linha 253 original), verbatim do brief.
- 10:35 -- Step 7: `CLAUDE.md:207` ("14 etapas" -> "15 etapas") e a frase da lista de coberturas (acrescentado `check_gates` ao fim, junto de `check_readme_anchors`); `docs/papeis/documentador.md:143` (acrescentada a frase sobre a escotilha valer na mensagem, nao na linha de comando). `docs/OPERACAO.md`: `grep -n 'check_readme_anchors' docs/OPERACAO.md` sem match -- **o arquivo nao lista as etapas do verify.ps1**, entao nao editei nada la (registrando "nao lista", conforme o brief instrui para esse caso). UTF-8 validado nos dois arquivos efetivamente editados (`CLAUDE.md`, `docs/papeis/documentador.md`) -- `docs/OPERACAO.md` nao foi tocado, entao nao ha o que validar nele.
- 10:36-10:42 -- Step 8: `pwsh -File scripts/verify.ps1 -SkipCross -SkipNet` em primeiro plano. `EXIT=0`. Conferido que a conta de "15 etapas" bate: `grep -n "Invoke-Step \"" scripts/verify.ps1` lista 14 chamadas de `Invoke-Step` (incluindo `check_gates` recem-acrescentada) mais o passo "contagem de testes pulados" (linha 130), que usa o mesmo contador `$script:StepNumber` mas nao passa por `Invoke-Step` -- total 15 quando nada e pulado. Na rodada com `-SkipCross -SkipNet`, 3 dessas 15 (vet linux, vet darwin, check_net) viraram aviso `[i] ... pulado`, entao o numero visto no ultimo passo foi 12 (`check_gates`), consistente.
- 10:43 -- Step 9: staged exatamente os 7 arquivos com mudanca real (`scripts/pre_commit_docs.ps1`, `scripts/check_gates.ps1`, as duas fixtures, `scripts/verify.ps1`, `CLAUDE.md`, `docs/papeis/documentador.md`) mais `docs/OPERACAO.md` (sem diff, `git add` nele nao mudou nada em stage -- confirmado com `git diff --stat -- docs/OPERACAO.md` vazio antes de commitar). Nenhum arquivo do dono (test-vault/, .claude/skills/, Resume-Claude.ps1, progress.md de 2026-07-25) tocado. `git commit -F .superpowers/sdd/2026-09-07-gates-e-rf63/commit-187.txt` -> commit `55d2599`. O hook novo rodou nesta chamada (via `.claude/settings.json`), e a chamada de `git commit` no Bash nao devolveu um bloqueio -- ou seja, o hook decidiu `allow`. O JSON do hook em si nao apareceu na saida do Bash (o PreToolUse roda no nivel do harness, fora do stdout do comando git), entao nao tenho o JSON literal para colar; o que tenho e o raciocinio determinista: nenhum dos 7 arquivos em stage e `.go`, entao `$codigo.Count -eq 0` e a decisao teria que ser "allow" / "nenhum .go de producao em stage" -- e foi isso que o commit fez (nao foi bloqueado).

## Status

DONE_WITH_CONCERNS

## Commits

- `55d2599` -- fix(gates): pre_commit_docs reads the hatch from the commit message, not the command line
  7 files changed, 181 insertions(+), 12 deletions(-): CLAUDE.md, docs/papeis/documentador.md, scripts/check_gates.ps1 (novo), scripts/pre_commit_docs.ps1, scripts/testdata/gates/msg-com-sem-doc.txt (novo), scripts/testdata/gates/msg-sem-escotilha.txt (novo), scripts/verify.ps1.

### RED — check_gates before the fix

Comando: `pwsh -File scripts/check_gates.ps1; echo EXIT=$?`, rodado antes do Step 3 (hook ainda sem `-Comando`/`-EmStage`).

```
Carregado em 525ms
=== pre_commit_docs.ps1 ===
pre_commit_docs.ps1: A parameter cannot be found that matches parameter name 'Comando'.
[!] escotilha em comentario de shell, mensagem sem ela -> deny: esperado 'deny', obtido 'erro'
pre_commit_docs.ps1: A parameter cannot be found that matches parameter name 'Comando'.
[!] escotilha no arquivo de -F -> allow: esperado 'allow', obtido 'erro'
pre_commit_docs.ps1: A parameter cannot be found that matches parameter name 'Comando'.
[!] escotilha em -m entre aspas duplas -> allow: esperado 'allow', obtido 'erro'
pre_commit_docs.ps1: A parameter cannot be found that matches parameter name 'Comando'.
[!] escotilha em -m entre aspas simples -> allow: esperado 'allow', obtido 'erro'
pre_commit_docs.ps1: A parameter cannot be found that matches parameter name 'Comando'.
[!] escotilha fora das aspas de -m -> deny: esperado 'deny', obtido 'erro'
pre_commit_docs.ps1: A parameter cannot be found that matches parameter name 'Comando'.
[!] -m sem escotilha, .go sem doc -> deny: esperado 'deny', obtido 'erro'
pre_commit_docs.ps1: A parameter cannot be found that matches parameter name 'Comando'.
[!] -m sem escotilha, .go com doc -> allow: esperado 'allow', obtido 'erro'
pre_commit_docs.ps1: A parameter cannot be found that matches parameter name 'Comando'.
[!] -F arquivo inexistente, escotilha no comando -> deny: esperado 'deny', obtido 'erro'
pre_commit_docs.ps1: A parameter cannot be found that matches parameter name 'Comando'.
[!] sem -m nem -F (editor), .go sem doc -> deny: esperado 'deny', obtido 'erro'

[!] check_gates: 9 de 9 casos reprovados
EXIT=1
```

9 de 9 `[!]`, `EXIT=1` -- como o brief previa.

### GREEN — check_gates after the fix

Duas rodadas. A primeira, logo apos o Step 3 verbatim do brief, reprovou 1 caso por um bug de binding de array entre processos, achado nesta tarefa e nao previsto no brief (mecanismo e correcao em `## Concerns`). A segunda, com `Decisao-Hook` corrigido em `check_gates.ps1`, passou 9/9.

Primeira rodada (achado do bug, 8/9):

```
Carregado em 577ms
=== pre_commit_docs.ps1 ===
[OK] escotilha em comentario de shell, mensagem sem ela -> deny
[OK] escotilha no arquivo de -F -> allow
[OK] escotilha em -m entre aspas duplas -> allow
[OK] escotilha em -m entre aspas simples -> allow
[OK] escotilha fora das aspas de -m -> deny
[OK] -m sem escotilha, .go sem doc -> deny
pre_commit_docs.ps1: A positional parameter cannot be found that accepts argument 'docs/TOOLS.md'.
[!] -m sem escotilha, .go com doc -> allow: esperado 'allow', obtido 'erro'
[OK] -F arquivo inexistente, escotilha no comando -> deny
[OK] sem -m nem -F (editor), .go sem doc -> deny

[!] check_gates: 1 de 9 casos reprovados
EXIT=1
```

Segunda rodada, apos corrigir `Decisao-Hook`:

```
Carregado em 417ms
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

[OK] check_gates: 9 casos
EXIT=0
```

## Mutation proofs

Revertida a mao a linha `$mensagem = Extrair-Mensagem $comando` em `scripts/pre_commit_docs.ps1` para `$mensagem = $comando`.

Rodada com a mutacao (`pwsh -File scripts/check_gates.ps1; echo EXIT=$?`):

```
Carregado em 408ms
=== pre_commit_docs.ps1 ===
[!] escotilha em comentario de shell, mensagem sem ela -> deny: esperado 'deny', obtido 'allow'
[!] escotilha no arquivo de -F -> allow: esperado 'allow', obtido 'deny'
[OK] escotilha em -m entre aspas duplas -> allow
[OK] escotilha em -m entre aspas simples -> allow
[!] escotilha fora das aspas de -m -> deny: esperado 'deny', obtido 'allow'
[OK] -m sem escotilha, .go sem doc -> deny
[OK] -m sem escotilha, .go com doc -> allow
[!] -F arquivo inexistente, escotilha no comando -> deny: esperado 'deny', obtido 'allow'
[OK] sem -m nem -F (editor), .go sem doc -> deny

[!] check_gates: 4 de 9 casos reprovados
EXIT=1
```

4 de 9 reprovaram: os tres que o brief nomeou (comentario de shell, fora das aspas, `-F` inexistente) mais um quarto ("escotilha no arquivo de -F -> allow", que tambem regride porque, com a mutacao, a mensagem nunca e extraida do arquivo de `-F`). `EXIT=1`, como esperado.

Linha restaurada a mao para `$mensagem = Extrair-Mensagem $comando`. Rodada apos restaurar:

```
Carregado em 352ms
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

[OK] check_gates: 9 casos
EXIT=0
```

`EXIT=0`. `git diff --stat -- scripts/pre_commit_docs.ps1` depois de restaurar: `1 file changed, 42 insertions(+), 9 deletions(-)` -- so o diff esperado do Step 3, nenhum residuo da mutacao.

## Verification

`git grep -n 'pre_commit_docs' -- '*.md' '*.ps1' '*.json'` (confirmando que ninguem chama `-Mensagem`): as ocorrencias sao (a) `.claude/settings.json` chamando o hook sem esse parametro (via stdin, modo real, invocacao inalterada), (b) narrativa em relatorios/ledger antigos descrevendo o BYPASS antigo por linha de comando (nao uma chamada com `-Mensagem`), e (c) a propria mensagem de erro dentro do hook. Reconferido especificamente com `git grep -n "pre_commit_docs.ps1 -Mensagem"` -- zero ocorrencias. Nenhum caller real usa o parametro removido.

`pwsh -File scripts/verify.ps1 -SkipCross -SkipNet` -- ultimas linhas:

```
[...] 11. check_readme_anchors
[OK] check_readme_anchors
[...] 12. check_gates
[OK] check_gates

[OK] Bateria completa. Pode commitar.
EXIT=0
```

`EXIT=0`. A suite tambem reportou "[!] 6 testes pulados" na etapa 3 (contagem de pulados) -- informativo, nao reprova (skips ja documentados: `vaulttest` fora do contexto certo, testes de sinal/perfil dependentes de ambiente), e o gate terminou com `[OK] Bateria completa. Pode commitar.`

UTF-8: `python -c "open('CLAUDE.md',encoding='utf-8').read()"` -> `[OK] CLAUDE.md UTF-8 valido`; `python -c "open('docs/papeis/documentador.md',encoding='utf-8').read()"` -> `[OK] docs/papeis/documentador.md UTF-8 valido`. `docs/OPERACAO.md` nao foi editado (ver `## Concerns`), entao nao ha UTF-8 novo para validar nele.

JSON do hook no commit: nao apareceu no stdout do `git commit` (o PreToolUse do hook roda no nivel do harness que envolve a chamada de Bash, fora do processo `git` em si), entao nao tenho o JSON literal para colar. O que se observa e que o commit NAO foi bloqueado; como nenhum dos 7 arquivos staged e `.go`, a unica decisao possivel do hook (com o codigo atual) e `allow` / "nenhum .go de producao em stage" -- coerente com o resultado observado.

## Concerns

1. **Bug de binding de array encontrado no proprio `Decisao-Hook` do brief, corrigido.** O codigo literal do Step 1 do brief chama `& pwsh -NoProfile -File $Hook -Simular -Comando $Comando -EmStage $EmStage`. Medido (nao suposto): quando `$EmStage` tem 2+ elementos, `pwsh -File` repassa os argumentos como argv cru para o processo filho, e so o PRIMEIRO elemento faz bind no parametro `[string[]]$EmStage`; o segundo vira um argumento posicional sem parametro que o aceite, e o hook (que usa `[CmdletBinding()]`) reprova com "A positional parameter cannot be found that accepts argument '...'" em vez de responder com a decisao do caso. Reproduzi isolado (script minimo com `[string[]]$Foo`, chamado via `pwsh -File` com `-Foo "a" "b"` -> mesmo erro; com `-Foo a,b` sem espaco -> vira string literal `"a,b"`, nao array de 2 elementos). Isso so aparece no caso `-EmStage $goEDoc` (2 elementos); os outros 8 casos usam `$go` (1 elemento) e passavam mesmo com o bug. Corrigi `Decisao-Hook` em `scripts/check_gates.ps1`: em vez de `-File` com argumentos crus, a funcao monta o texto `& '<hook>' -Simular -Comando '<comando>' -EmStage @('<a>','<b>')` (aspas simples escapadas por dobramento) e roda via `pwsh -NoProfile -EncodedCommand <base64>` -- assim quem interpreta a linha e o parser do PowerShell (que entende `@(...)` como array de verdade), nao o passa-argumento cru do `-File`. Validei a tecnica isolada antes de aplicar (script de teste no scratchpad da sessao, removido depois, nao commitado). A assinatura de `Decisao-Hook` (`-Comando`, `-EmStage`), os nomes `Caso`/`$Fixtures`/`$ProjectRoot`/`$Hook` e o contrato de saida de `Caso`/`check_gates` ficaram identicos ao brief -- so o MECANISMO de invocar o processo filho mudou. Reportando para quem revisar e para a Task 188 (que estende este arquivo): se `audit_reports.ps1` tambem for chamado como subprocesso com parametro de array, o mesmo cuidado se aplica.
2. `docs/OPERACAO.md` **nao lista** as etapas do `verify.ps1` (`grep -n 'check_readme_anchors' docs/OPERACAO.md` sem match) -- nao editei o arquivo, conforme o brief instrui para esse caso. Se o dono quiser essa lista la, e uma tarefa nova, nao um ajuste desta.
3. Nao rodei `scripts/test_orphans.ps1` (fora do escopo desta tarefa, e a instrucao foi explicita para nao rodar).
4. Status `DONE_WITH_CONCERNS` por causa do item 1: entrego com o gate verde e a prova de mutacao real, mas o codigo de `check_gates.ps1` que ficou no commit **difere do texto literal do Step 1 do brief** na funcao `Decisao-Hook`, por uma razao medida e documentada, nao por preferencia.
