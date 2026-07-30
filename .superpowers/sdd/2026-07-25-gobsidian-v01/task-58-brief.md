### Task 58: `internal/writer/block.go` — `replace_block` por `^id`

RF-33, P1.

#### O que já está fechado

`internal/parser` já extrai block ids e `index.Note.Blocks` já guarda `Start` e `End`. Não reimplemente a extração. **Os offsets têm a mesma questão de BOM da Task 57.**

#### A armadilha

**Block id não é único por projeto do Obsidian, mas o usuário assume que é.** Dois `^abc` na mesma nota: recuse por ambiguidade, como a Task 57 faz com slug. E `^abc` no fim de uma linha de lista pertence ao item, não ao parágrafo — o intervalo que o parser guarda é a fonte, não sua intuição sobre onde o bloco termina.

#### Verificações além dos passos

- Bloco em parágrafo, em item de lista, e em bloco de citação: os três?
- `^id` duplicado recusa por ambiguidade?
- `^id` inexistente devolve erro que **nomeia o id**, não "não encontrado"?
- BOM, CRLF, e a variante sem nenhum dos dois?
- O `^id` sobrevive à substituição, ou é para ser reescrito junto? **Decida e diga.**

**Prova de mutação obrigatória:** aponte a substituição para `Block.Start` sem o ajuste de BOM e confirme que um teste nomeado reprova.

**Files:** Create `internal/writer/block.go`, `block_test.go`
**Commit:** `feat(writer): replace block by id`

---

