# gobsidian — contexto do projeto

As instruções completas estão em `AGENTS.md` e são importadas abaixo. **Leia-as antes de agir.**

@./AGENTS.md

---

## Se a importação acima não tiver carregado

Abra `AGENTS.md` na raiz do repositório e leia-o inteiro. Não prossiga sem ele: ele tem as proibições absolutas, o contrato de relatório e as armadilhas de ferramenta deste ambiente.

As quatro regras abaixo estão repetidas aqui **de propósito**, porque violá-las causa dano irreversível e uma importação que falhou em silêncio não pode ser a razão de isso acontecer. Todo o resto está em `AGENTS.md`.

1. **Nunca rode `git checkout`, `git restore`, `git stash`, `git clean` nem `git reset`.** Há trabalho não commitado neste repositório o tempo todo, e um subagente já destruiu trabalho exatamente assim. Para desfazer o que você escreveu, edite de volta ou apague o arquivo específico que você criou.
2. **Nunca rode `go mod tidy`.** Dependências fixadas sem importador seriam removidas, junto com o pin obrigatório do SDK de MCP.
3. **Nunca escreva em stdout a partir de código alcançável de `serve`.** stdout pertence ao JSON-RPC; log vai para stderr via `log/slog`.
4. **Se você foi despachado para uma tarefa numerada, leia o brief inteiro primeiro:** `.superpowers/sdd/2026-07-25-gobsidian-v01/task-<N>-brief.md`. Ele é autocontido. Se algo faltar nele, responda `BLOCKED` — não preencha a lacuna por conta própria.

## Skills e workflows

- `.claude/skills/` — skills do projeto. **Fonte única.** `.agents/` é espelho gerado, não versionado, e não deve ser editado.
- `.claude/workflows/` — workflows.
