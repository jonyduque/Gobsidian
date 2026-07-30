### Task 59: Tools `note_create`, `note_append`, `note_patch`

RF-30, RF-31, RF-32, RF-37. **`dry_run` e `expected_hash` em todas.**

#### O que já está fechado

`docs/TOOLS.md` define os schemas. **O contrato existe antes do código** — leia-o e implemente o que está lá; se algo divergir, corrija o doc na mesma tarefa e diga o quê.

- **`note_create` falha se o caminho já existir** (RF-30). Sobrescrever silenciosamente é perda de dado.
- **`expected_hash` é o `xxhash` sobre os bytes crus**, igual a `index.Note.Hash`, calculado **antes** de `vault.StripBOM`. Hash que não casa é conflito: recuse, e devolva o hash atual para o chamador poder reler.
- **`dry_run` devolve o diff e não toca o disco.** Verifique que não toca — mtime do arquivo antes e depois.

#### As duas armadilhas que já aconteceram aqui

**Schema que promete e código que ignora.** `note_list` declarava `fields` e o descartava. Para **cada** parâmetro de cada uma das três tools: um teste que o exercita e afirma que ele mudou a resposta. Liste-os no relatório com o nome do teste ao lado.

**Handler que devolve `error` Go faz o SDK montar `IsError` sem `StructuredContent`.** Devolver resultado de erro com a saída zerada manda `{"ok":false,...}` junto, e o cliente não distingue falha de "nada a fazer" no canal que ele lê primeiro.

#### Verificações além dos passos

- `dry_run: true` deixa o mtime do arquivo **inalterado**? Compare, não confie.
- `expected_hash` errado recusa **e** devolve o hash atual?
- `expected_hash` omitido: escreve, ou recusa? **Decida e documente** — as duas são defensáveis, a ambígua não.
- `note_create` sobre caminho existente falha com erro que diz qual caminho?
- `note_create` cria diretórios intermediários, ou recusa? Decida e documente.
- Caminho fora do cofre é recusado nas três? É a mesma camada de confinamento; confirme que ela está no caminho de escrita.
- Cada parâmetro de cada tool tem teste que prova que age. Liste todos.

**Prova de mutação obrigatória:** desligue a checagem de `expected_hash`; desligue o curto-circuito de `dry_run`. Confirme que um teste nomeado reprova em cada caso.

**Files:** Modify `internal/service/`, `internal/mcpsrv/`, `docs/TOOLS.md` se divergir
**Commit:** `feat(mcpsrv): write tools with dry_run and expected_hash`

---

