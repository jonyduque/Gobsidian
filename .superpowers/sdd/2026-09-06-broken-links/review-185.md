# Revisao da Task 185 — `note_move` de uma nota que cita a si mesma

Base `5f9ed57` -> head `3ff26ef`. Pacote lido: `review-5f9ed57..3ff26ef.diff`,
`internal/service/write.go` (`MoveNote` inteiro, `moverCorpo`),
`internal/service/anchor_selfref_test.go`, `docs/TOOLS.md` §`note_move`,
`.superpowers/sdd/2026-09-06-broken-links/progress.md`.
Sem comandos git, sem re-rodar suite, sem subagentes. Nao rodei teste focado:
a duvida que teria motivado um (o teste discrimina?) foi resolvida pela propria
saida de mutacao colada — ver Achado 2.

---

## Spec compliance

| Step | Veredito | Evidencia |
|---|---|---|
| 1. Teste que falha antes | ✅ | `TestMoveNote_NotaQueCitaASiMesma` em `anchor_selfref_test.go:94-157`: `a.md` = `# A\n\nVeja [[a]].\n`, `MoveNote` -> `sub/a.md`, sem erro, `sub/a.md` existe, `a.md` nao existe, `Resolved` afirmado pelo indice. Todas as afirmacoes que o brief pediu estao la. Ressalva no Achado 2: a que o brief chamou de discriminante nao e a que discrimina. |
| 2. Rodar e ver falhar | ✅ | ENOENT colado no relatorio para os dois testes. Confere com o codigo atual: `anchor_selfref_test.go:107` e exatamente o `t.Fatalf("MoveNote: %v", err)` do teste novo, e `:45` cai no mesmo ponto do teste estendido. Numeros de linha e texto batem com a arvore — a saida nao foi remontada. |
| 3. Corrigir com uma conta so | ✅ | `write.go:655-658` define `caminhoAtual`; `write.go:660,661,665,671,676,680` a usam para trava, `Abs`, leitura, escrita, as tres mensagens de erro e a lista `Rewritten`. Nenhum segundo `if` decide o mesmo. Comentario com o defeito em `write.go:644-654`. |
| 4. Rodar e ver passar + discriminante do F1 | ✅ | GREEN colado para os tres testes. O discriminante que a revisao 182 pediu esta fechado: `anchor_selfref_test.go:60-61` poe `[[a]]` no corpo de `a.md` e afirma `[[b]]` no corpo lido de `sub/b.md` (`:92-94`), mais `LinksUpdated == 2` (`:78-80`). Essa e a metade que a rodada 1 nao podia fixar, e agora esta fixada por texto, nao por indice. |
| 5. Prova de mutacao | ✅ | Ancora `if refPath == canonicalFrom {` e unica em `write.go` (a unica outra ocorrencia da expressao, `:648`, esta dentro de comentario e sem o `if `/`{`). Saida real colada, no passado, com o ENOENT nomeando o teste e a linha 107, restauracao com SHA-256 conferido, EXIT=0. |
| 6. Gate e commit | ✅ (conforme evidencia colada; nao re-rodei) | `verify.ps1` 14/14 colado com EXIT=0 e os 6 skips nomeados um a um — sao os mesmos de ambiente do Windows ja conhecidos, nenhum novo. Mensagem de commit em `task-185-commit-msg.txt` e identica a exigida pelo brief. Ledger: `progress.md:29` registra DONE + SHA antes do relatorio dizer que acabou. |

**Spec compliance: ✅** — os seis steps cumpridos.

## Code quality

**Approved com should-fix** — nenhum achado bloqueante. Dois should-fix (um de
doc que diverge do comportamento novo, um de comentario que afirma o que nao se
verifica) e dois nits.

---

## Achados

### 1. `docs/TOOLS.md:496` diz que a origem nunca entra no retorno; agora ela entra — should-fix

`docs/TOOLS.md:496`, sobre `note_move`:

> "…`diffs`: mapa caminho → diff unificado, uma entrada por referenciadora que
> seria reescrita. **A origem não entra: mover não altera o conteúdo dela.**"

Depois desta task a frase e falsa nos dois lados do retorno quando a nota cita a
si mesma com alvo escrito:

- execucao real: `write.go:680` acrescenta `caminhoAtual` — ou seja, `sub/a.md` —
  a `Rewritten`, e `LinksUpdated` conta o link dela. A origem **entra**, e mover
  **alterou** o conteudo dela.
- `dry_run`: `write.go:557-570` itera `affectedNotes`, que ja contem
  `canonicalFrom` nesse caso, e produz `diffs["a.md"]` com um diff nao vazio.

A metade do `dry_run` e pre-existente (o dry-run nunca move, entao nunca deu o
ENOENT e sempre listou a origem); a metade da execucao real **e desta task**:
antes, esse caminho devolvia `CodeInternal` e nunca chegava a reportar nada.
`note_move` passou de "erra" para "sucede reportando a origem", e isso e mudanca
de comportamento visivel ao host, com a regra do CLAUDE.md ("mudanca de
comportamento atualiza `docs/` no mesmo PR") apontando para ela.

Nao chamo de bloqueante porque a Task 184 do mesmo marco e a revisao de
documentacao e ainda nao rodou. **Fix:** acrescentar a excecao ao inventario da
Task 184 — a frase vira algo como "a origem so entra quando ela propria cita a
nota movida por alvo escrito; nesse caso o caminho reportado e o NOVO". Se a
Task 184 fechar sem isso, o release sai com `TOOLS.md` afirmando o contrario do
que o codigo faz.

### 2. `internal/service/anchor_selfref_test.go:91-93` afirma qual asserção discrimina, e nao e essa — should-fix

O comentario de topo do teste novo diz:

> "Por isso a asserção que discrimina esta correcao nao pode ser textual: e o
> Resolved do link, lido do INDICE…"

A asserção do `Resolved` **nao discrimina esta correcao**, por duas razoes
independentes:

1. `[[a]]` resolve por nome-base. Depois de `idx.MoveNote`, `Resolved` aponta
   para `sub/a.md` esteja o corpo reescrito ou nao — o texto do link e identico
   nos dois mundos. A asserção passa com e sem o fix.
2. Sem o fix ela nem e alcancada: `MoveNote` devolve o ENOENT e o teste morre em
   `:107`. **A propria saida da prova de mutacao mostra isso** — a falha colada e
   `anchor_selfref_test.go:107`, o `t.Fatalf("MoveNote: %v", err)`, nunca `:147`,
   o `Resolved`.

O que de fato discrimina e o par `err == nil` (`:106-108`) mais
`res.LinksUpdated == 1` (`:154-156`) — este ultimo e o bom, porque separa o fix
certo ("redireciona para `canonicalTo`") do fix errado que tambem faria o teste
passar ("pula o citante quando ele e a propria nota"): com o `continue`,
`LinksUpdated` seria 0 e o teste reprovaria. Vale dizer isso no comentario, que
e conhecimento util; o que nao vale e a frase atual, que nomeia como
discriminante uma asserção que nao discrimina. Regra do CLAUDE.md: "nao afirme
estado que voce nao verificou".

**Fix:** reescrever as tres linhas — a asserção do indice guarda a regressao do
F2 (`Resolved` preso ao caminho antigo), e quem discrimina esta correcao e o
`MoveNote` sem erro mais `LinksUpdated == 1`.

### 3. `internal/service/anchor_selfref_test.go:91` — "asserção" com cedilha e til num arquivo sem acentos — nit

Unico caractere acentuado introduzido pelo diff nos dois arquivos (conferido:
`grep -n '[áàâãéêíóôõúüç…]'` nos dois devolve so essa linha e um `cabeçalho`
pre-existente em `write.go:267`). O arquivo inteiro escreve `ancora`,
`correcao`, `referencia` sem acento. **Fix:** `assercao`.

### 4. `docs/ARMADILHAS.md` nao ganhou o mecanismo — nit

"Renomeia primeiro, le o caminho antigo depois" e "a nota e sua propria citante"
sao exatamente a forma de defeito que o arquivo coleciona, e este ja custou uma
task inteira. O brief nao pediu (Step 6 e so gate e commit), entao nao conta
contra a spec. Se a Task 184 abrir `ARMADILHAS.md`, e o lugar barato de
registrar.

---

## Riscos nomeados

### Risco 1 — `caminhoAtual`: uma decisao so, e a trava e no arquivo certo, sem re-trava

`caminhoAtual` (`write.go:655-658`) e usada nos tres pontos que importam e em
mais dois: `s.locker.Lock` (`:660`), `s.vault.Abs` (`:661`, que serve leitura e
escrita pelo mesmo `absRef`), as tres mensagens de erro (`:665,671,676`) e
`Rewritten` (`:680`). Nao ha um segundo `if` decidindo a mesma coisa — o achado
que o brief queria evitar nao aconteceu. A trava e sobre `canonicalTo`, que e
onde o arquivo esta: `moverCorpo` ja o renomeou em `:637`.

Sobre duplo-trancamento: `PathLocker` **nao e reentrante** (dito em
`write.go:781`), entao a pergunta e real — e a resposta e nao. `moverCorpo`
(`write.go:860-873`) adquire as travas de `de` e `para` ordenadas por chave e as
libera por `defer` ao **retornar**, em `:637-639`, antes de o laco comecar. Na
iteracao o laco pega uma trava por vez e a libera em todos os quatro caminhos de
saida (`:664,670,675,679`) antes da proxima. Nenhum ponto de `MoveNote` segura
`canonicalFrom` ou `canonicalTo` enquanto o laco pede `canonicalTo`. Ordem de
aquisicao tambem nao gera AB-BA: uma trava de cada vez nao tem ordem para
inverter.

O que existe — e ja existia antes do fix — e uma **janela** entre `moverCorpo`
soltar as travas e o laco pegar a de `canonicalTo`: outro escritor pode entrar
ali. O fix nao a alarga nem a estreita; e a mesma janela que qualquer citante
ja tinha. Fora de escopo, e nao piorou.

### Risco 2 — atomicidade na falha: o estado resultante e melhor que o de antes, nao pior

Se a reescrita do corpo da propria nota movida falhar depois do rename:
`vault.WriteAtomic` e temp+sync+rename, entao a falha deixa `sub/a.md` com o
conteudo **integro** que o rename colocou la — nunca truncado, nunca meio
gravado. O que sobra e a auto-referencia sem reescrever, e `moveNoteErro`
(`write.go:604-612`) devolve o parcial com `Rewritten`/`LinksUpdated` do
instante da falha. Isso e exatamente o trade-off ja escrito e assumido em
`write.go:626-636`: sobra link apontando para o nome antigo, que e visivel e
recuperavel.

Comparado com antes: antes esse item **sempre** falhava, com ENOENT, e o
`return` abortava o laco inteiro — todo citante que ordenasse depois da chave da
origem (`affectedKeys` e ordenado alfabeticamente em `:588-590`) ficava sem
reescrever, num cofre em que o corpo ja tinha se movido. O fix nao so remove a
falha certa como destrava os citantes que vinham depois dela. O pior caso novo e
estritamente menos grave que o antigo caso unico.

Um detalhe honesto: quando o nome-base nao muda (`a.md` -> `sub/a.md`), o
`WriteAtomic` da origem grava bytes identicos aos que ja estao la — toca mtime e
acorda o watcher sem que nada tenha mudado. Nao e regressao desta task: qualquer
citante cujo `[[a]]` seja reescrito para `[[a]]` ja fazia isso, e e o caso comum
de mover uma nota de pasta sem renomear.

### Risco 3 — o teste novo exercita o `service`, e discrimina; mas nao pela asserção que o comentario aponta

Sim, exercita `service.MoveNote`: `anchor_selfref_test.go:100-105` chama
`svc.MoveNote`, e e essa chamada que quebrava. O `idx.MoveNote` + `idx.Get`
(`:134-152`) vem **depois**, sobre o resultado, para afirmar `Resolved` pelo
indice como o brief pediu, e a justificativa dada no relatorio esta correta:
nenhuma tool de escrita deste projeto atualiza o indice, entao sem essa chamada
o `idx` do teste ficaria parado no estado pre-move. Nao e "o teste so exercita o
indice" — seria achado, e nao e o caso.

O que nao se sustenta e a afirmacao de qual asserção discrimina; ver Achado 2. O
teste esta correto e falha pelo motivo certo (a mutacao prova). O comentario e
que descreve mal o proprio teste.

### Risco 4 — prova de mutacao

Real, no passado, com saida colada e coerente com a arvore: ancora unica, teste
nomeado, linha 107 batendo com o `t.Fatalf` que existe hoje nessa linha, "restaurado
byte a byte (SHA-256 confere)", EXIT=0. E a mutacao escolhida (`&& false`)
desliga precisamente a regra nova, nao um efeito colateral dela. Nada a apontar.

### Risco 5 — comentario que reivindica limitacao ja resolvida

Limpo no codigo. O bloco `LIMITE MEDIDO em 2026-09-06` foi removido do topo de
`anchor_selfref_test.go` e reescrito para dizer que a Task 185 fechou o laco;
`grep` por `LIMITE MEDIDO` na arvore inteira nao devolve nada, e o comentario do
guarda em `write.go:499-510` ja falava do discriminante certo (`Target == ""`) e
continua valido. As mencoes remanescentes ao defeito como "pre-existente" estao
so no plano e no ledger, que sao documentos historicos e devem mesmo registrar
o que se sabia na epoca.

O contrario, porem, aparece: o comentario novo de `:91-93` **cria** uma
afirmacao errada onde o antigo dizia a verdade sobre um limite real. Achado 2.
