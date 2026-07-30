### Task 56: `internal/writer/diff.go` — Myers sobre linhas

#### Onde isto encaixa

`dry_run` (RF-37) devolve o diff sem tocar o disco. Cerca de 150 linhas, **sem dependência** — é decisão fechada: uma biblioteca de diff traria mais superfície que o problema pede.

#### A armadilha

**Diff sobre linhas precisa saber o que é uma linha, e CRLF é a pergunta.** Dividir por `\n` deixa `\r` no fim de cada linha e todo diff em arquivo Windows vira "toda linha mudou". Decida se a comparação normaliza o EOL, documente, e **teste com CRLF**.

E o diff é para leitura humana e de modelo, não para aplicar: não invente formato próprio. Use unificado, com contagem de linhas de contexto configurável ou fixa e documentada.

#### Verificações além dos passos

- Arquivo CRLF contra ele mesmo produz diff **vazio**? É o teste que pega a divisão por `\n`.
- Inserção no meio, remoção no meio, e substituição produzem o diff mínimo? Compare com o esperado escrito à mão.
- Arquivo vazio contra não vazio, e o contrário?
- Linha final sem `\n` — o caso que quase toda implementação erra?
- Quantas alocações num diff de 1.000 linhas com 10 mudanças? Medido, ou **"não medido"**.

**Prova de mutação obrigatória:** troque a normalização de EOL por divisão crua em `\n` e confirme que o teste de CRLF reprova.

**Files:** Create `internal/writer/diff.go`, `diff_test.go`
**Commit:** `feat(writer): line-based Myers diff`

---

