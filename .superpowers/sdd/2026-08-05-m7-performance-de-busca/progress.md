# PONTEIRO — o ledger NAO esta aqui

=== LEDGER MOVIDO — este arquivo e so um ponteiro, nao e estado ===
=== leia .superpowers/sdd/2026-07-25-gobsidian-v01/progress.md ===
=== o projeto esta na Task 97 (2026-08-12), nao na Task 0 ===

As tres linhas acima comecam com "===" de proposito: e o unico padrao que
`sdd.ps1 status` imprime, junto com linhas que comecam com "Task ". Sem elas o
comando mostrava a secao "=== Ledger ===" VAZIA, que se le como "nada
registrado" — o mesmo erro que o ponteiro existe para impedir, com outra cara.

O ledger unico do projeto vive em:

    .superpowers/sdd/2026-07-25-gobsidian-v01/progress.md

Ele cobre a Task 1 ate a mais recente, inclusive as do M7 e as posteriores.

## Por que este arquivo existe

`scripts/sdd.ps1 status` resolve o plano MAIS RECENTE em
`docs/superpowers/plans/`, que hoje e `2026-08-05-m7-performance-de-busca`, e
procura o ledger no diretorio de artefatos correspondente. Sem este arquivo ele
respondia:

    [!] Ledger ausente em .superpowers\sdd\2026-08-05-m7-performance-de-busca\progress.md

Quem rodasse o comando documentado no CLAUDE.md concluiria que nada foi
registrado — com 97 tarefas registradas no outro caminho. Ledger que parece
ausente e pior que ledger desatualizado, porque a resposta obvia a "nao ha
registro" e re-executar trabalho pronto, que e a falha mais cara deste fluxo.

E o mesmo motivo de `.superpowers/sdd/progress.md` ser ponteiro: o plugin
superpowers passou a escrever artefatos por plano na versao 6.2.0, e dois
arquivos com o mesmo proposito ja derivaram neste projeto uma vez — um tinha 16
tarefas e o outro 6.

## Regras

NAO edite este arquivo. NAO o leia como estado. Escreva no ledger unico.

Se um plano novo entrar em `docs/superpowers/plans/`, o diretorio de artefatos
dele vai precisar de um ponteiro igual a este, ou `sdd.ps1 status` volta a
dizer que o ledger sumiu.
