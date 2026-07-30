### Task 67: Âncoras quebradas no relatório de impacto

R4. **Nada novo a implementar** — `broken_anchors` já existe em `vault_stats` desde o M1. Esta tarefa o expõe no relatório de impacto de `note_move` e `note_delete`.

#### O que vincula esta tarefa

- **Nada novo a implementar:** `broken_anchors` existe em `vault_stats` desde o M1. Esta tarefa o expõe no relatório de impacto.
- **Campo de API com valor fixo mente sempre.** `alias_collisions` era `Collisions: 0` literal. Um teste que produz a situação e afirma o conteúdo, não que o campo existe.
- **`docs/TOOLS.md` faz parte da entrega.** Campo na resposta que não está no contrato é campo em que ninguém pode confiar.

#### Por que é uma tarefa e não uma linha

Mover uma nota **não** quebra a âncora de um link que aponta para ela — o heading continua lá. Mas `note_patch` e `replace_block` do M4 **quebram**, e o relatório de impacto é o lugar onde o usuário veria isso antes de aceitar.

**Decida o escopo e diga qual escolheu:** o relatório de impacto cobre só os links que a operação atual quebra, ou também os já quebrados na nota afetada? O segundo é mais útil e mais ruidoso.

#### Verificações além dos passos

- O campo aparece no retorno das duas tools?
- Uma âncora que a operação **não** quebra não é listada? É o teste que separa "relatório" de "despejo".
- `docs/TOOLS.md` descreve o campo com o mesmo nome JSON que o código emite? Compare contra a saída real.
- Campo com valor sempre vazio? **Campo de API com valor fixo mente sempre** — um teste que produz a situação e afirma o conteúdo.

#### Regras de execução

Valem para toda tarefa deste plano e não são negociáveis.

- **O plano é a fonte.** Transcreva o desenho desta seção; não improvise uma variante. Se um teste falhar por motivo que a seção não explica, **pare e reporte** — não ajuste a expectativa para o código passar.
- **Nunca rode `git checkout`, `git restore`, `git stash`, `git clean` ou `git reset`.** **`go mod tidy` está proibido.**
- **Toda escrita no cofre usa `writer.WriteAtomic`.** Nunca abra o arquivo do usuário para escrita direta. EOL e BOM preservados byte a byte (RF-38).
- **Confinamento vale para origem E destino.** `vault.Resolve` e `Canonicalize` são as mesmas duas camadas; escrita não ganha caminho novo.
- **`dry_run` não toca o disco.** Prove por mtime, não por intenção.
- **Verde obrigatório antes do commit:** `pwsh -File scripts/verify.ps1`, que agora inclui o `golangci-lint` como oitava etapa.
- Commits em Conventional Commits, em inglês. Sem `helpers.go`, `utils.go` ou `common.go`.

#### Contrato de relatório

Grave em `.superpowers/sdd/task-67-report.md`: o que implementou; evidência de TDD (comando e saída do RED, comando e saída do GREEN); a prova de mutação com a **saída real colada**, nomeando o teste que reprovou; a tabela de verificações acima com o resultado **real** de cada uma, inclusive as que deram certo; arquivos alterados; o que ficou de fora e por quê; e `git status --porcelain`.

**Não escreva número que você não mediu** — escreva "não medido". **Prova de mutação no condicional não é prova:** use `scripts/mutate.ps1` e cole o que ele imprimiu. Rode `pwsh -File scripts/audit_reports.ps1 67` antes de entregar.

Responda com no máximo 15 linhas: status (`DONE` | `DONE_WITH_CONCERNS` | `BLOCKED` | `NEEDS_CONTEXT`), commit criado, resumo de teste em uma linha, e preocupações.

**Files:** Modify `internal/service/`, `docs/TOOLS.md`
**Commit:** `feat(service): broken anchors in the move and delete impact report`

---

