# Ponteiro — este plano NÃO tem ledger próprio

O ledger deste projeto é **um só**, e fica em:

    .superpowers/sdd/2026-07-25-gobsidian-v01/progress.md

As Tasks 104 a 123 (plano `docs/superpowers/plans/2026-08-16-revisao-fixes.md`)
são registradas **lá**, na mesma numeração contínua das Tasks 1 a 103.

`sdd.ps1 status` resolve o plano mais recente e avisa que o ledger está ausente
aqui. O aviso é esperado; este arquivo existe para responder a ele.

**Por que um só.** Este projeto já teve dois ledgers e eles divergiram: um tinha
16 tarefas e o outro tinha 6, e ninguém percebeu até alguém re-executar trabalho
pronto. A regra registrada no `CLAUDE.md` saiu daquele episódio. Um ledger por
plano recria exatamente a condição — a numeração é contínua, o histórico é o
mesmo, e a próxima sessão não tem contexto, tem o ledger.

Os outros artefatos por plano (`task-N-base.txt`, `review-*.diff`) ficam aqui
mesmo: são por tarefa e descartáveis. O ledger não é.
