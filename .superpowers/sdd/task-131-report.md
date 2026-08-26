# Task 131 — A1 + B5 + B17 + B6: escrita que não reporta o que não fez

**Status:** DONE_WITH_CONCERNS — **um contrato foi invertido de propósito, e um
teste existente guardava o antigo.** Ver "A troca que esta tarefa fez".
**Commit:** `539d25c` — `fix(watcher,service): close A7, A1, B5, B17 and B6`

---

## O mecanismo, reconferido no código

Em `internal/service/write.go`, a ordem era: `WriteAtomic(absRef, ...)` para cada
citante (`:581`), depois `WriteAtomic(absTo, fromRaw)` (`:626`), depois
`_ = os.Remove(absFrom)` (`:636`).

Duas consequências independentes:

1. O erro do remove era **descartado**. No Windows, arquivo com handle aberto
   por outro processo recusa remoção — e o Obsidian segurando a nota aberta é
   rotina. Resultado: **sucesso reportado com a nota nos dois caminhos**.
2. Os citantes eram reescritos **antes** de o corpo se mover. Falhando a
   movimentação, os links já estavam persistidos em disco apontando para um
   destino que nunca existiu.

---

## Reprodução com condição real do sistema, não com dublê

Segurar um handle é o que o Obsidian de fato faz. Qual acesso produz qual efeito
foi **medido**, porque a primeira tentativa não bloqueava a leitura:

```
GENERIC_READ        + share=0 -> os.ReadFile OK,   os.Remove ERRO
GENERIC_READ|WRITE  + share=0 -> os.ReadFile ERRO, os.Remove ERRO
```

O número está no comentário dos dois testes, porque a escolha do flag é o que
faz cada cenário existir.

### RED

```
=== RUN   TestMoveNaoReportaSucessoComNotaDuplicada
    move_atomico_windows_test.go:77: SUCESSO reportado com a nota DUPLICADA:
    origem e destino existem e MoveNote devolveu nil
    (res={From:origem.md To:destino.md Rewritten:[citante.md] LinksUpdated:1 ...})
    move_atomico_windows_test.go:84: MoveNote devolveu nil mas a origem continua no disco
--- FAIL

=== RUN   TestMoveNaoReescreveCitantesAntesDeMoverOCorpo
        citante = "# Citante\n\nver [[destino]] aqui.\n"
--- FAIL
```

O segundo mostra o dano literal: o link foi reescrito para `[[destino]]` e a
nota nunca se moveu.

**Um cenário inicial foi descartado por medir a coisa errada.** A primeira
versão ocupava o destino com um diretório — mas isso falha na validação de
"destino já existe", **antes** do laço de citantes, e o teste passava sem
exercitar nada.

### GREEN

```
--- PASS: TestMoveNaoReportaSucessoComNotaDuplicada (0.02s)
--- PASS: TestMoveNaoReescreveCitantesAntesDeMoverOCorpo (0.01s)
--- PASS: TestDeleteToTrashNaoMenteQuandoORemoveFalha (0.02s)
--- PASS: TestMoveDryRunNaoApresentaDiffVazioComoResultado (0.01s)
```

---

## A troca que esta tarefa fez

`moverCorpo` move o corpo primeiro, via `os.Rename` quando possível (atômico no
mesmo volume, então "duplicada" deixa de ser um estado alcançável), com fallback
copia-e-remove **conferindo o remove**.

**Isso inverte quem fica inconsistente numa falha parcial**, e não é ganho de
graça:

| | falha depois de… | quem fica inconsistente |
|---|---|---|
| ordem antiga | citantes gravados, corpo falha | **todos** os links apontam para destino inexistente; nota na origem |
| ordem nova | corpo movido, um citante falha | **apenas** os citantes ainda não reescritos apontam para o caminho antigo |

A ordem nova é melhor — link obsoleto é visível e recuperável repetindo a
atualização; nota duplicada é silenciosa — mas a garantia antiga era real:
*"se não der para terminar, não mova"*.

`TestNoteMovePartialFailureReportsWhatWasApplied` **afirmava essa garantia
antiga**. Ele foi reescrito para o contrato novo, com o motivo e a troca no
lugar. **Não foi ajustado para a suíte passar**: a inversão foi apresentada como
o preço da Alternativa 1 em `docs/SUGESTOES.md` e escolhida pelo dono antes de
qualquer código.

---

## Duas regressões minhas, pegas pela suíte existente

Nenhuma das duas apareceu nos meus testes novos. Registradas porque a lição é
essa:

1. **A criação do diretório de destino continuava depois do move.** Com a nova
   ordem, `WriteAtomic` falhava em `criando temporario em <dir>: The system
   cannot find the path specified` — quatro testes de move acusaram.
2. **O contrato acima**, que só um teste existente guardava.

Se eu tivesse confiado apenas nos testes que escrevi, as duas teriam passado.

---

## B5, B17 e B6

- **B5** — `note_delete to_trash` descartava o erro do remove. RED:
  `Deleted:true` com a nota na lixeira **e** no caminho original. Agora devolve
  `CodeFileLocked` com mensagem dizendo que a cópia na lixeira existe — sem
  isso, quem recebe o erro não sabe se precisa limpar algo.
- **B17** — o dry-run do move engolia o erro de leitura (`fromRaw, _ :=`) e
  apresentava **diff vazio como resultado**. Quem lê um dry-run vazio conclui
  que a operação não muda nada.
- **B6** — as travas do move agora são adquiridas em ordem de chave
  normalizada, não `from→to`. O deadlock AB-BA entre dois moves opostos deixa de
  ser possível, ao custo de uma comparação de string.

---

## Provas de mutação

Três regras, três `EXIT=0`:

1. `-Anchor 'if err := os.Remove(absFrom); err != nil {'` → `&& false`
   (A1, erro do remove conferido)
2. `-Anchor 'if err := s.moverCorpo(canonicalFrom, canonicalTo, absTo); err != nil {'`
   → `if err := error(nil); err != nil {` (A1, corpo antes dos citantes)
3. `-Anchor 'return DeleteNoteResult{}, Errorf(CodeFileLocked,'` → devolve
   sucesso (B5)

A terceira saiu **`EXIT=2`** na primeira tentativa: a âncora
`if err := os.Remove(absPath); err != nil {` ocorre **2×** no arquivo. Ampliada
com a linha vizinha até ficar única, como o próprio script instrui.

---

## Verificações

- `pwsh -File scripts/verify.ps1`: **14 de 14 [OK]**.
- Testes de move verdes em `internal/service` **e** `internal/mcpsrv` (o E2E).

---

## O que ficou de fora

**O rollback automático dos citantes não foi implementado.** Ele está descrito
como Alternativa 2 em `docs/SUGESTOES.md`, com o motivo de não ser padrão: o
próprio rollback pode falhar, e aí o estado é pior que o de qualquer uma das
alternativas — parcialmente revertido, sem registro. Fica como decisão do dono,
separada.
