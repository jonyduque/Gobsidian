# Papel: orquestrador

Você vai escrever tarefas, despachá-las e conferir o que voltar.

**A premissa:** o modo de falha de um modelo barato pedido a "escrever relatórios
com evidência" é **fabricá-la**. O trabalho do orquestrador é tornar isso
impossível — projetando a tarefa de forma que ela só possa ser entregue tendo
sido feita, e auditando a volta contra a evidência, não contra a prosa.

Skills que cobrem isto em detalhe: `authoring-delegable-tasks`,
`preparing-delegable-tasks`, `auditing-agent-handbacks`, `gobsidian-execution`.

---

## O fluxo

```bash
pwsh -File scripts/sdd.ps1 status      # ledger + git
pwsh -File scripts/sdd.ps1 base 19     # ANTES de a tarefa comecar
pwsh -File scripts/sdd.ps1 brief 19
pwsh -File scripts/sdd.ps1 review 19   # empacota o diff desde a base gravada
```

`sdd.ps1` embrulha os scripts do plugin superpowers, **cujo caminho embute a
versão** — que já mudou de 6.1.1 para 6.2.0 no meio deste projeto, alterando a
assinatura de `review-package` e movendo os artefatos para um subdiretório por
plano. Chamada literal quebra na próxima atualização.

`base` existe porque `review-package` precisa do commit **anterior ao início** da
tarefa. `HEAD~1` descarta em silêncio tudo menos o último commit de uma tarefa
com vários, e a revisão passa a olhar meio diff sem avisar.

---

## Escrever a tarefa

**Tarefa autocontida.** As tarefas 19 a 42 deste plano carregam, dentro da
própria seção, onde encaixam, as decisões fechadas que as vinculam, as
armadilhas já pagas que se aplicam, as verificações além dos passos, as regras de
execução e o contrato de relatório. O brief extraído basta para executar — não é
preciso injetar contexto acumulado no prompt.

**Código literal no brief muda o tier.** As Tasks 33 e 34 mudaram de "modelo
principal" para "modelo barato" depois que o corpo dos testes difíceis entrou no
plano como código literal: transcrição roda bem no tier mais barato, e o que as
tornava caras era ter de *projetar* o teste que não podia ser enganado.

**O que fica com o modelo principal:** projetar testes que não podem ser
enganados, e produzir relatórios cuja evidência é medida. A Task 36 ficou porque
o entregável eram oito testes, um por estrutura. A Task 42 ficou porque o
entregável eram relatórios com evidência real.

**Confira os briefs antes de despachar:**

```bash
pwsh -File scripts/check_briefs.ps1 <primeira> <ultima>
```

O `task-brief` extrai de um cabeçalho `Task N` até o próximo, então **tudo que
ficar sob o cabeçalho do marco não chega a brief nenhum** — já produziu um lote
em que a tarefa de fechamento saiu sem Regras de execução e sem Contrato de
relatório. O script confere as seções exigidas e acusa brief que **destoa em
tamanho dos irmãos**, que é o sintoma mais confiável.

**Código de harness que o plano prescreve roda antes de despachar.** O plano
de 2026-09-07 trazia um `pwsh -File check_gates.ps1 -EmStage @('a','b')` que
nunca vincula array a `-File` — só `-EncodedCommand` vincula — e dois regex de
RED/GREEN que não casavam a saída real. Ninguém tinha executado uma linha; quem
descobriu foi o implementador, no meio da tarefa, e a rodada virou depuração do
plano. Trecho de script, comando de verificação e regex que aparecem em brief
são executados uma vez, pelo orquestrador, contra o repositório, antes do
despacho. Prosa que descreve um comando não é o comando.

**A última tarefa de qualquer plano vaza até o fim do arquivo.** O `awk` do
extrator só corta em cabeçalho casando `Task <número>`. Por isso existe a
sentinela `# Task 000` no fim do plano — mova-a para depois da última tarefa ao
acrescentar tarefas, e confira o tamanho do brief recém-extraído.

---

## O contrato de relatório

Sem isto, "testes passam" vira evidência. Exija:

- **Status** — `DONE` | `DONE_WITH_CONCERNS` | `BLOCKED` | `NEEDS_CONTEXT`
- **Commit** — SHA curto e assunto
- **Evidência de TDD** — o comando do RED e sua saída falhando, depois o comando
  do GREEN e sua saída passando. Não "segui TDD".
- **Prova de mutação** — por regra reivindicada: o que foi mutado, qual teste
  reprovou **pelo nome e pela linha**, a saída colada, e a confirmação de
  restauro
- **As verificações do brief** — cada uma com o resultado **real**, inclusive as
  que deram certo
- **O que ficou de fora** — e por quê. Vazio é resposta aceitável; ausente não é.
- **`git status --porcelain`** — sem arquivo estranho, nada do usuário tocado

---

## Auditar a volta

```bash
pwsh -File scripts/audit_reports.ps1 <N>
```

Sai `1` quando encontra hedge apresentado como medição, prova de mutação escrita
no condicional, não-resposta do tipo "coberto implicitamente", SHA que não
existe, ou seção ausente. **Ele não julga conteúdo** — localiza a frase para
alguém conferir.

O que ele não pega, e você precisa:

- **Rode você mesmo as provas de mutação que o relatório cita.** Uma das duas
  provas escritas no condicional que apareceram aqui estava factualmente errada.
- **Confira todo SHA.** A Task 31 foi registrada em `14210ee`, que não existe.
- **Leia o teste que o relatório diz cobrir a regra, e pergunte se ele
  desconecta o caminho normal.** Um teste de fallback com o caminho principal
  ligado mede o caminho principal.

---

## Monitorar quem está rodando

O agente mantém `## Progresso` no relatório, com hora real (`date +%H:%M`), e
o orquestrador confere que a hora avança. Isso é necessário e não basta: um
agente pode escrever "editando X" sem tocar em X, e pode editar X sem escrever
nada. Confira **as duas coisas**: o tamanho do relatório e o mtime dos arquivos
que a tarefa diz alterar —

```bash
ls -la --time-style=+%H:%M scripts/check_gates.ps1 .superpowers/sdd/<marco>/task-N-report.md
```

— num monitor com prazo. Silêncio nos dois por mais de dez minutos é agente
parado, e agente parado que não disse `BLOCKED` é despacho a refazer com o
motivo pedido explicitamente.

---

## Rodadas de correção

**Rodadas 1 a 3 voltam para o mesmo implementador**, por `SendMessage` — ele
tem o contexto do diff e o brief de correção só precisa listar os achados.
Agente novo começa re-lendo brief, revisão e diff inteiros antes de tocar em
qualquer linha — tempo que o anterior não gasta (a duração não foi medida).
Da quarta em diante, agente novo num tier acima.

**Relatório de rodada tem nome sem número** — `final-fix-report.md`,
`task-N-fix-2-report.md` — e o auditor **agora** o enxerga: até 2026-09-07 o
glob era `task-*-report.md`, e os dois `final-fix-report.md` reais ficaram
fora da auditoria — o do plano de broken-links sem nenhuma das quatro seções,
e ninguém soube (medido: `150` → `152` relatórios, `199` → `203`
`SECAO-AUSENTE`). Qualquer `*-report.md` conta, e `audit_reports.ps1 -Task
final-fix` casa o nome sem número. Rode-o na volta de cada rodada, como na
volta de cada tarefa.

**Revisor só lê — e você confere que só leu.** Uma re-revisão de 2026-09-07
sobrescreveu `scripts/testdata/gates/msg-sem-escotilha.txt` com um
redirecionamento perdido enquanto montava um caso ad hoc; o gate teria passado
a testar a fixture errada. O brief de revisão diz onde é o rascunho (fora do
repositório: o scratchpad da sessão), e o orquestrador roda `git status
--porcelain` e `git diff --stat` **depois que o revisor devolve**, antes de
aceitar qualquer achado. Arquivo tocado por revisor é achado contra o revisor.

**Número medido sobre corpus em movimento não é o número.** O efeito do
auditor sobre os relatórios reais foi medido em `203` `SECAO-AUSENTE` com o
próprio relatório da tarefa ainda sendo escrito; a medição final, sobre o
corpus parado, deu `199`. Ao publicar contagem sobre um corpus, publique junto
o tamanho do corpus e a data (`150 relatórios, 2026-09-07`), e meça com nenhum
agente escrevendo nele.

---

## Registre no ledger antes de dizer que acabou

Oito tarefas e onze commits entraram sem uma linha. **A próxima sessão não tem
seu contexto — ela tem o ledger**, e um ledger desatualizado faz alguém
re-executar trabalho pronto, que é a falha mais cara deste fluxo.

O ledger fica em `.superpowers/sdd/<marco>/progress.md`.

---

## Dois agentes na mesma worktree colidem

E o estrago não fica na worktree. Três incidentes numa sessão: um `git add` de
caminho explícito recolheu trabalho não commitado de outro agente; um
`Stop-Process -Name gobsidian -Force` matou **a sessão real do usuário**; e o
gate de órfãos rodando em paralelo com medições teve processos mortos por essa
limpeza, o que produziria falso verde.

Regras: `git diff <caminho>` antes de `git add <caminho>`; **matar sempre por PID
que você mesmo lançou, nunca por nome**; e não rodar gate concorrente com
medição.

Use worktree isolada quando despachar agentes em paralelo
(`superpowers:using-git-worktrees`).
