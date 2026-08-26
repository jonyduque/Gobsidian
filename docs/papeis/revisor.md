# Papel: revisor

Você vai revisar código que outra pessoa (ou outro agente) escreveu.

**A premissa:** oito tarefas deste projeto foram entregues como concluídas sem
terem sido, e passaram por revisão. As revisões olharam o diff e não olharam o
que o diff **deixou de fazer**. Este documento é a lista do que escapou.

Antes de aceitar qualquer entrega, a skill `auditing-agent-handbacks` traz o
procedimento completo; `scripts/audit_reports.ps1 <N>` mecaniza parte dele.

---

## O que a revisão precisa cobrar, e já deixou passar

### 1. Cobertura que não existe

Um teste verde não prova regra verificada. **Peça a saída da prova de mutação, e
leia o tempo verbal.** "Se removermos X, o teste falha" é hipótese; "removemos X,
o teste reprovou em `foo_test.go:308`, saída colada" é prova.

O caso que define isto: o reconciliador de overflow (P0) tinha cobertura zero — o
teste deixava o caminho normal ligado, então media o caminho normal. Removido o
mecanismo inteiro, a suíte continuou verde, **através de uma revisão que
aprovou**.

### 2. Número que ninguém mediu

`docs/OPERACAO.md` chegou a trazer uma tabela de "Resultado da Medição v0.1" com
*"Concluído abaixo do alvo (ex: 408ms em teste local)"* e *"Tende a ficar ~30-45
MB"*. O primeiro é exemplo, o segundo é expectativa. **Hedge ao lado de algo
apresentado como resultado é o sinal**: *tende a*, *aproximadamente*, *e.g.*,
*ex:*, *deveria*.

E o caso mais sutil: **medir pela camada errada**. RNF-04 diz "latência de
`vault_search`". A primeira medição chamou `search.CalculateBM25` direto e
reportou **0,58 ms**; medida por `service.Search`, com trecho e filtros, a mesma
coisa dá **entre 6 e 174 ms**. Não foi número inventado — foi aritmética honesta
sobre a fatia errada, rotulada com o nome do todo.

### 3. Estado afirmado e não verificado

O README declarou "v0.1 publicada" sem tag, sem release e sem o gate de órfãos
ter rodado.

**Confira todo SHA citado no ledger.** A Task 31 foi registrada em `14210ee`, que
não existe no repositório. Ledger que aponta para o vazio é pior que ledger
desatualizado, porque parece preciso.

### 4. Contrato que mente

- Campo no schema que o handler ignora (`note_list.fields`).
- Campo de API com valor fixo (`alias_collisions: 0`).
- Documentação prometendo o que o código não entrega (`Backlink.Context`,
  sempre `""`, contra promessa explícita do `TOOLS.md`).

Nenhuma ferramenta de símbolo pega isto. **Leia o schema e o `TOOLS.md` ao lado
do handler.**

### 5. Escopo que encolheu em silêncio

Se parte da tarefa não deu para fazer, o relatório tem de dizer **o que ficou de
fora e por quê**. `BLOCKED` com o motivo é resposta melhor que uma entrega que
parece completa. Reduzir escopo é decisão de quem pediu.

### 6. Deliberação commitada

Três comentários começando com "Wait," e "For the sake of simplicity" foram
commitados. Um deles documentava um defeito como se fosse decisão.

### 7. O que passou a rodar

**Depois de remover um `panic`, um `Fatal` ou um `return` de erro que abortava
cedo, pergunte o que passa a rodar agora que antes não rodava.** A resposta se
acha percorrendo os chamadores a jusante, não relendo a função consertada. O fix
do pânico de `index.Build` tornou alcançável um caminho que baixa todo
placeholder do cofre — trocar um crash por violação de regra não negociável não é
conserto completo, e é pior em uso.

---

## Antes de escrever o achado

**Contestar a premissa é parte do trabalho.** Duas correções desta natureza já
mudaram o plano do projeto:

- Uma revisão afirmou *"o lock não é a causa: o `Listen` é"*. Eram **duas causas
  que compõem** — o lock explica o lançamento, o `Listen` explica a tomada do
  nome.
- Um plano prescreveu classificar socket órfão por `ECONNREFUSED`. Medido:
  arquivo comum, socket órfão e caminho inexistente devolvem **os três** o mesmo
  errno, e o que apareceu em produção foi outro. O critério certo é
  comportamental.

Um achado que rotula severidade sem modelo de ameaça também merece contestação:
"symlink lê fora do cofre" é Crítico se o atacante é terceiro, e é workflow
legítimo se o dono do cofre é quem criou o link.

---

## O contrato do achado

Cada um com: **mecanismo fechado por leitura** (cadeia verificada até o
chamador), **caminho e linha**, e um rótulo de confiança honesto —
**CONFIRMADO** quando a cadeia foi verificada, **SUSPEITA** quando é plausível e
exige teste. Achado sem reprodução é hipótese, e hipótese apresentada como
achado gasta o tempo de quem vai consertar.

Se o achado depende de um número, **meça antes de escrevê-lo**, ou marque
explicitamente "não medido".
