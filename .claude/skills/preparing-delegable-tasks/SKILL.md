---
name: preparing-delegable-tasks
description: Author a milestone's tasks so a cheap model can execute them without you — analyse what the last batch got wrong, fix the instrument before writing, write self-contained sections with literal test bodies, define what proves each task done, and end with the delegation prompt. Use whenever asked to prepare a milestone, plan the next tasks, write briefs for delegation, or set up a batch for subagents. Also use before re-dispatching a milestone that came back with defects, since the first phase is what stops the same class from escaping twice.
---

# Preparar tarefas para delegação

Cinco fases, em ordem. **A primeira não é opcional e é a que mais paga:** escrever tarefas sem olhar o que a última leva errou reproduz a mesma classe de defeito com números diferentes.

O produto final é sempre o mesmo: seções autocontidas no plano, briefs conferidos, e **o prompt de delegação pronto para colar**. Terminar sem o prompt é entregar metade.

---

## Fase 1 — Analisar a leva anterior antes de escrever qualquer linha

Não comece pela tarefa nova. Comece perguntando **o que escapou, e por quê o instrumento não pegou.**

```bash
pwsh -File scripts/sdd.ps1 status
pwsh -File scripts/audit_reports.ps1
pwsh -File scripts/verify.ps1
golangci-lint run ./internal/... ./cmd/...
```

Para cada defeito que a revisão anterior encontrou, responda **duas** perguntas:

1. **Quem o pegou — o gate ou uma pessoa?** Se foi pessoa, há uma lacuna de instrumento, e ela vai deixar passar o próximo também.
2. **De que classe ele é?** Escreva a classe, não o caso. "Faltou `errcheck` no `apply.go`" é caso; "o linter não estava no gate" é classe.

### As perguntas que já encontraram defeito neste projeto

- **A prova de mutação foi escolhida para passar?** Mutar `b = 0.75` para `0.0` cruza fronteira qualitativa e sempre reprova; mutar para `0.5` sobrevive. Os cinco parâmetros do BM25 estavam "provados" com valores extremos, e nenhum teste fixava valor nenhum.
- **A medição mediu o que o rótulo diz?** `0,58 ms` para RNF-04 era aritmética honesta sobre `CalculateBM25` direto — sem filtros, sem trecho, com consulta que casava uma nota em quinhentas. Pela camada que o requisito nomeia, o mesmo dá 174 ms. **Aritmética correta sobre a coisa errada é o defeito mais difícil de pegar em revisão**, porque tudo no relatório está tecnicamente certo.
- **O corpus é do tamanho que o rótulo diz?** Um laço inseriu cem vezes o **mesmo caminho** e o log imprimiu `Notes: 1` ao lado de `"100 notas"`. O número foi para o PRD como decisão fechada.
- **Alguma asserção é garantida pela linha acima dela?** O teste do RNF-11 chamava a limpeza e **depois** afirmava não haver sobras. Com a limpeza desligada, o teste reprova — havia sobras reais mascaradas.
- **O gate parou de gatear em silêncio?** O `audit_reports.ps1` deixou de casar cinco linhas do ledger quando o formato mudou, e as três checagens de SHA simplesmente não rodaram. O `check_net` já tinha feito o mesmo: com `go list ./...` quebrado, avisava que não rodou e saía verde.
- **Você afirmou algo sem conferir?** Acusei `watcher` importando `search` como violação de camada; `ARCHITECTURE.md` §5.3 **especifica** essa chamada. Dois achados meus estavam errados. Confira a fonte antes de escrever o achado — e mais ainda antes de escrever a tarefa que o "corrige".

---

## Fase 2 — Consertar o instrumento antes de escrever a tarefa

**Se um defeito escapou porque o instrumento não olhava, conserte o instrumento primeiro.** Escrever a tarefa antes é garantir que a próxima leva escape do mesmo jeito.

Onde cada lição mora:

| Tipo de lição | Vai para |
|---|---|
| Etapa de verificação que faltava | `scripts/verify.ps1` |
| Forma de evidência falsa em relatório ou ledger | `scripts/audit_reports.ps1` |
| Brief que não chega inteiro ao executor | `scripts/check_briefs.ps1` |
| O que "pronto" significa | `AGENTS.md` §4 |
| Regra irreversível que um import falho não pode custar | `GEMINI.md`, inline |
| Técnica reutilizável | uma skill |
| Decisão que vincula uma tarefa só | a seção dela no plano |

**Prove que o instrumento agora pega.** Mesma disciplina da prova de mutação: adultere o artefato, confirme que a ferramenta acusa, restaure. Trocar um SHA do ledger por `deadbee` e ver `SHA-FANTASMA` disparar é a diferença entre "acrescentei a regra" e "a regra funciona".

Um checador que dispara em prosa legítima vira ruído e para de ser lido — o que é pior que não existir. A primeira versão do `check_briefs.ps1` acusou uma linha que **negava** ter placeholders. Rode contra o repositório inteiro e olhe o volume antes de aceitar a regra.

---

## Fase 3 — Escrever as seções

### A armadilha que domina esta fase

**O `task-brief` extrai de `### Task N` até o próximo `###`. Tudo sob o cabeçalho do marco (`## M5`) não chega a brief nenhum.**

O M5 foi escrito com quatro decisões fechadas e seis regras compartilhadas no preâmbulo. Os briefs saíram com 22 a 71 linhas; o da Task 67 tinha **zero** das quatro, e nenhuma das seis seções tinha Regras de execução ou Contrato de relatório. Foi pego por leitura, um minuto antes do despacho.

**Repita, dentro de cada seção, o que vincula aquela tarefa.** Duplicação entre seções é o ponto: a seção é a unidade que viaja.

```bash
pwsh -File scripts/check_briefs.ps1 <primeira> <ultima>
```

Sai `1` quando falta seção ou quando um brief destoa em tamanho. **Não substitui ler um brief inteiro** — garante só que nada ficou preso no preâmbulo.

### A forma de cada seção

```
### Task N: <título que diz o que muda, não o arquivo que toca>

#### Onde isto encaixa
#### O que vincula esta tarefa          ← as decisões fechadas que se aplicam A ESTA
#### A evidência medida do defeito       ← só quando corrige algo; comando e saída
#### A decisão que esta tarefa precisa tomar certo
#### Armadilhas já pagas que se aplicam
#### O código dos testes que não são óbvios   ← Go literal
#### Verificações além dos passos
#### Regras de execução
#### Contrato de relatório
**Files:** / **Commit:**
```

### O que transforma tarefa cara em tarefa barata

**Escreva o corpo dos testes difíceis como Go literal.** Foi isso que tirou quatro tarefas do modelo caro no M2.1 e mais três no M3. O que encarece não é transcrever: é **projetar** um teste que não pode ser enganado. Movida a decisão para o plano, ela é revisada uma vez e o executor transcreve.

Escreva literal quando o teste tem sutileza: offset com BOM, crash em subprocesso, corpus que precisa afirmar o próprio tamanho, concorrência. Deixe em prosa quando for direto.

### Pré-comprometa as decisões que seriam tomadas por preferência

Se a tarefa contém uma escolha que dois executores razoáveis fariam diferente, **decida no plano, com o motivo**. Não decidir é delegar a decisão junto com a transcrição, e ela volta como "escolhi X" sem que ninguém tenha comparado.

E quando o plano pré-compromete um **critério**, o executor não pode trocá-lo: a Task 52 mediu certo e decidiu por velocidade quando o critério escrito era custo de versionamento. O defeito não foi a escolha, foi o silêncio sobre haver regra.

### Nomeie a armadilha, com evidência medida

Cada seção deve dizer **o que vai dar errado ali**, e provar que já deu. `"Medido: com o corpo de Reconcile substituído por return, o teste passa em 2,8 s"` vale mais que três parágrafos de cuidado.

### Confira o que você afirma sobre o código

Antes de escrever "isto viola X", abra X. Duas afirmações minhas sobre camadas estavam erradas e a arquitetura especificava o contrário. Use o MCP do `gopls`: `go_symbol_references` antes de dizer quem chama o quê.

---

## Fase 4 — Definir o que prova cada tarefa pronta

Uma tarefa sem critério verificável volta como "implementei". Para cada uma:

**A prova de mutação, com o comando pronto.** Não "prove por mutação" — o comando, com âncora e teste:

```bash
pwsh -File scripts/mutate.ps1 -Path internal/x/y.go `
  -Anchor '<texto exato do arquivo>' -Replacement '<mutação que compila>' `
  -Test TestNome -Package ./internal/x/
```

Saída `0` é o que se quer; `1` significa regra escrita e não verificada; `2` é inconclusivo. **A mutação tem de compilar** — falha de build não é cobertura, e o script sai `2` por isso.

**Mutação por regra, não por tarefa.** Se a tarefa fixa cinco constantes, são cinco mutações com cinco saídas. E **perturbação pequena, não valor extremo**: mutar para o valor degenerado testa que a regra existe, não que ela está certa.

**Quando o alvo não é atingido:** teto medido no teste, lacuna escrita, tarefa registrada. Nunca `t.Skip`, nunca afrouxar o alvo dos outros casos. Alvo não atingido e registrado é informação.

**Meça pela camada que o requisito nomeia**, e afirme o corpus. Um gerador de corpus termina com `if got := idx.NoteCount(); got != n { t.Fatalf(...) }`.

**Asserção de tempo não sobrevive ao `-race`** (2× a 6×). Atrás de constante com build tag, em arquivo separado, com a medição registrada nos dois modos.

**A tarefa de fechamento do marco não tem mutação** — não entrega código. Ela precisa **dizer isso** no contrato de relatório, ou o checador não distingue "não se aplica" de "esqueceu".

---

## Fase 5 — O prompt de delegação

**Sempre apresente o prompt completo ao final.** Ele não é resumo do que você fez: é o que a pessoa cola. Seções obrigatórias:

1. **O que torna este marco diferente** — se pode destruir dados, isso vem primeiro e sozinho.
2. **O que executar**, com a sequência e o motivo dela (quais tarefas tocam os mesmos arquivos).
3. **Estado inicial** — base já gravada, árvore suja por um arquivo de propósito, ledger.
4. **O loop por tarefa** — `base`, `brief`, workflow, validação, `review`, ledger.
5. **Aceitação por tarefa** — o que exigir de cada uma, nomeando o modo de falha barato. É a seção que mais evita retrabalho.
6. **Decisões que não devem ser re-litigadas** — com a justificativa, para o orquestrador recusar sem pesquisar.
7. **Regras para o orquestrador** — revisor também erra; escopo não encolhe em silêncio; confira SHA.
8. **O portão final**, comando a comando.
9. **O que volta para quem pediu** — mover tag, mudar contrato publicado, decisão de projeto.

Diga o custo estimado em invocações, e qual tarefa você não delegaria sem olhar a saída inteira.

---

## Antes de despachar

```bash
pwsh -File scripts/check_briefs.ps1 <primeira> <ultima>
pwsh -File scripts/verify.ps1
pwsh -File scripts/sdd.ps1 base <primeira>
```

E **leia um brief inteiro**, do começo ao fim, como se você fosse executá-lo sem mais nada. Se em algum ponto você precisaria perguntar alguma coisa, a seção não está pronta.

A árvore fica suja por `task-N-base.txt` — é o estado correto, não trabalho pendente.
