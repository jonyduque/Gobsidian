# AGENTS.md

Este projeto mantém suas instruções em [`CLAUDE.md`](CLAUDE.md), que serve
qualquer agente — não só o Claude Code. **Leia-o primeiro.**

Ele é curto de propósito: traz o índice da documentação, a estrutura do projeto,
as regras que não se negociam e o que significa uma tarefa estar pronta. O resto
está distribuído por papel.

---

## Se você foi despachado para executar uma tarefa numerada

O seu brief é a fonte. Ele fica em
`.superpowers/sdd/<marco>/task-<N>-brief.md` e é autocontido. **Leia o brief
inteiro antes de escrever qualquer linha.** Onde o brief e a documentação
falarem do mesmo assunto, o brief vence.

Descubra o marco corrente com:

```bash
pwsh -File scripts/sdd.ps1 status
```

Não o deduza de memória: o caminho do marco muda a cada plano, e chutá-lo já
produziu brief extraído do plano errado.

---

## Vá direto ao papel

| Você vai… | Leia |
|---|---|
| escrever código | [`docs/papeis/implementador.md`](docs/papeis/implementador.md) |
| revisar código de outro | [`docs/papeis/revisor.md`](docs/papeis/revisor.md) |
| escrever ou avaliar teste | [`docs/papeis/testador.md`](docs/papeis/testador.md) |
| otimizar desempenho | [`docs/papeis/desempenho.md`](docs/papeis/desempenho.md) |
| escrever documentação | [`docs/papeis/documentador.md`](docs/papeis/documentador.md) |
| escrever, despachar ou auditar tarefas | [`docs/papeis/orquestrador.md`](docs/papeis/orquestrador.md) |

E, antes de commitar, releia [`docs/ARMADILHAS.md`](docs/ARMADILHAS.md).

---

## O mínimo, se você não vai ler mais nada

- `pwsh -File scripts/verify.ps1` verde antes de qualquer commit.
- Nunca `git checkout`, `git restore`, `git stash`, `git clean` nem `git reset`.
- Nunca `go mod tidy`.
- stdout pertence ao JSON-RPC; todo log vai para stderr.
- Não escreva número que você não mediu.
- Escopo não encolhe em silêncio: diga o que ficou de fora e por quê.
