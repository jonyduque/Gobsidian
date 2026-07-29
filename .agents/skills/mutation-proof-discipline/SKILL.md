---
name: mutation-proof-discipline
description: Prove that a rule is verified and not merely written, using scripts/mutate.ps1, and audit task reports for fake evidence with scripts/audit_reports.ps1. Use whenever you claim a test covers something, write or accept a task report, review someone else's task, add a regression test after a fix, or are about to say "tests pass". Also use before marking any task complete in the ledger, since a green suite over an unmutated rule is the most expensive false signal in this project.
---

# Prova de mutação e evidência de relatório

Regra que sobrevive a mutação não está verificada, está escrita. Na Task 13, sete regras de um módulo sobreviveram a mutantes com a suíte verde — inclusive a que o comentário do próprio fix defendia. Ler o teste não acha isso.

Este projeto tem dois scripts para isso. Use-os em vez de fazer a mão: as três formas de errar a mutação manual já aconteceram aqui.

## 1. `scripts/mutate.ps1` — uma mutação, um teste, restauração garantida

```bash
pwsh -File scripts/mutate.ps1 `
    -Path internal/watcher/apply.go `
    -Anchor 'if n, ok := idx.Get(path); ok {' `
    -Replacement 'if n, ok := idx.Get(path); ok && false {' `
    -Test TestApply -Package ./internal/watcher/
```

**O código de saída é o contrato, e ele é invertido de propósito:**

| Saída | Significa |
|---|---|
| `0` | O teste **reprovou** sob mutação. A regra está verificada. |
| `1` | O teste **passou** sob mutação. A regra está escrita, não verificada. Escreva o teste que falta. |
| `2` | Inconclusivo: âncora ambígua, arquivo ausente, ou a mutação quebrou o build. |

Teste que sobrevive à mutação é o resultado ruim, então ele é o que falha. Isso permite encadear: `pwsh -File scripts/mutate.ps1 ... && echo "coberta"`.

O script garante três coisas que a mão não garantia:

- **A âncora precisa ocorrer exatamente uma vez.** `str.replace` que não casa não falha — segue em silêncio, e a suíte verde vira prova de cobertura de uma regra que nunca foi tocada. Zero ocorrências para o script antes de escrever qualquer byte.
- **Restauração byte a byte, conferida por SHA-256, em `finally`.** Vale para Ctrl+C e para teste travado. Ler com encoding de texto e gravar em modo texto converte o arquivo para CRLF no Windows e o `gofmt` reprova um `.go` perfeitamente formatado — já custou dois commits aqui.
- **Falha de build não conta como cobertura.** Uma mutação que não compila também "reprova o teste". O script detecta `[setup failed]`, `[build failed]`, `undefined:`, `no test files` e `no tests to run`, e sai `2` em vez de `0`.

### Como escolher a mutação

Mute a **regra**, não o teste. A pergunta é "se esta linha sumisse, algum teste gritaria?".

- Condição de guarda: `if cond {` → `if cond && false {`
- Chamada com efeito: `foo(a, b)` → `_ = foo`
- Linha inteira: `-Replacement ''`
- Constante de fronteira: `n < 1` → `n < 0`

Não mute duas coisas de uma vez. Se duas mutações são necessárias para um teste reprovar, são duas regras, e você precisa saber qual está descoberta.

## 2. O que conta como prova no relatório

Prova de mutação tem **três** partes, e falta uma invalida as outras duas:

1. **O que foi mutado** — arquivo, linha, o antes e o depois.
2. **Qual teste reprovou, pelo nome e pela linha** — com a saída real do `go test` colada.
3. **Confirmação de restauro.**

**Prova escrita no condicional não é prova.** Os relatórios das Tasks 30 e 31 escreveram *"Se removermos X, o teste falha"*. A da Task 30 estava factualmente errada: o reconciliador foi removido e a suíte continuou verde, deixando um requisito P0 com cobertura zero durante toda uma revisão que o aprovou. O tempo verbal é o sinal — prova real está no passado e traz saída colada.

## 3. `scripts/audit_reports.ps1` — auditar antes de aceitar

```bash
pwsh -File scripts/audit_reports.ps1 33     # so a Task 33
pwsh -File scripts/audit_reports.ps1        # varredura completa
```

Procura as cinco formas de mentira que já passaram por revisão neste projeto:

| Regra | O caso real que a originou |
|---|---|
| `MUTACAO-CONDICIONAL` | "Se removermos X, o teste falha" nas Tasks 30 e 31 |
| `NAO-RESPOSTA` | "covered implicitly by the stat checks" na Task 29 |
| `HEDGE` | "ex: 408ms em teste local" numa tabela chamada "Resultado da Medição" |
| `SHA-FANTASMA` | ledger apontando a Task 31 para `14210ee`, que não existe |
| `RELATORIO-AUSENTE` | tarefa marcada completa sem relatório no disco |
| `SECAO-AUSENTE` / `CURTO` | relatório da Task 29 com 1.148 bytes, sem RED nem GREEN |

O script **não julga conteúdo** — ele localiza a frase para uma pessoa conferir. Alguns achados são legítimos: `~15s` numa discussão de projeto é prosa, não medição falsa. O que ele impede é a frase passar sem ninguém olhar.

**"não medido" não é sinalizado, de propósito.** É a resposta certa quando não houve medição, e marcá-la ensinaria a escondê-la.

## 4. Quando usar

- **Antes de dizer que um teste cobre alguma coisa.** Uma vez por regra reivindicada.
- **Ao receber uma tarefa de volta de um subagente.** `audit_reports.ps1 <N>` primeiro, depois `mutate.ps1` em cada regra que o relatório reivindica. Oito tarefas deste projeto voltaram como completas sem estarem.
- **Ao escrever teste de regressão depois de um fix.** O teste que "prova" o fix precisa reprovar sem ele.
- **Antes de registrar a tarefa no ledger.**

## 5. O que a mutação não pega, e o que pega no lugar

- **Teste que deixa o pipeline real rodando.** `TestOverflowReconciliationFull` injetava overflow com o watcher ativo, então os eventos normais aplicavam as mudanças e a reconciliação nunca era exercida. A mutação **pegou** isso — mas só porque alguém a rodou. Um teste de mecanismo de recuperação precisa desconectar o caminho normal, ou está medindo o caminho normal.
- **Regressão que uma feature nova apagou.** Mutação compara com o presente. Para isso o instrumento é o A/B contra o commit anterior — foi assim que se descobriu que a extensão do Dataview tinha apagado wikilinks que o commit anterior já coletava.
- **Fixture inerte.** As fixtures de exclusão da Task 8 usavam extensões que o filtro descartaria de qualquer jeito, então apagar a regra de exclusão não mudava contagem nenhuma. Pergunte se a fixture consegue distinguir os dois estados antes de confiar no teste.
