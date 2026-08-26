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
