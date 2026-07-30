### Task 66: Tool `note_delete`

R3 e RF-36.

#### O que vincula esta tarefa

- **`to_trash` é `true` por padrão.** Exclusão definitiva exige passá-lo explicitamente como `false`. Default destrutivo numa tool que um modelo chama sozinho é a diferença entre erro recuperável e dado perdido.
- **`.trash/` já está em `excludedDirs`** (`internal/vault/walk.go:31`). A nota sai do índice pelo watcher, sozinha. **Não acrescente caso especial** — seria código para um problema que não há.
- **`--read-only` remove `note_delete` do `ListTools`.**
- **O relatório prévio é calculado ANTES da exclusão.** Depois, ele lista o estado já quebrado: tecnicamente correto e inútil para decidir.

#### O contrato e o que já está fechado

`docs/TOOLS.md` §`note_delete`: `path`, `to_trash` (default **`true`**), `report_broken_links` (default `true`), `dry_run`.

- **`to_trash` verdadeiro por padrão. Exclusão definitiva exige passá-lo explicitamente como `false`.** Um default destrutivo numa tool que um modelo chama sozinho é a diferença entre erro recuperável e dado perdido.
- **`.trash/` já está em `excludedDirs`.** A nota sai do índice pelo watcher, sozinha. Não escreva caso especial (decisão 4).
- **`report_broken_links` lista as notas que passarão a ter links quebrados** — informação que frequentemente muda a decisão, e é o motivo de o relatório ser prévio.

#### A armadilha

**O relatório prévio tem de ser calculado ANTES da exclusão, e o `dry_run` tem de devolvê-lo sem excluir.** Um relatório calculado depois lista o estado já quebrado — tecnicamente correto e inútil para decidir.

E: **colisão de nome no `.trash/`.** Apagar `a.md`, recriar `a.md`, apagar de novo — o segundo não pode sobrescrever o primeiro na lixeira. Decida o esquema (sufixo, subpasta por data) e **teste os dois apagados**.

#### Verificações além dos passos

- `to_trash` default é `true` — provado por chamada sem o parâmetro, não por leitura do schema.
- `to_trash: false` exclui de verdade?
- Duas exclusões do mesmo nome não se sobrescrevem na lixeira?
- `report_broken_links` lista as notas certas, **antes** de excluir?
- `dry_run` não exclui e devolve o relatório?
- A nota some do índice depois da exclusão, sem caso especial? Ponta a ponta com o watcher.
- Excluir nota **sem** backlinks reporta lista vazia, e não erro?

**Prova de mutação obrigatória:** mova o cálculo do relatório para depois da exclusão e confirme que um teste nomeado reprova.

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

Grave em `.superpowers/sdd/task-66-report.md`: o que implementou; evidência de TDD (comando e saída do RED, comando e saída do GREEN); a prova de mutação com a **saída real colada**, nomeando o teste que reprovou; a tabela de verificações acima com o resultado **real** de cada uma, inclusive as que deram certo; arquivos alterados; o que ficou de fora e por quê; e `git status --porcelain`.

**Não escreva número que você não mediu** — escreva "não medido". **Prova de mutação no condicional não é prova:** use `scripts/mutate.ps1` e cole o que ele imprimiu. Rode `pwsh -File scripts/audit_reports.ps1 66` antes de entregar.

Responda com no máximo 15 linhas: status (`DONE` | `DONE_WITH_CONCERNS` | `BLOCKED` | `NEEDS_CONTEXT`), commit criado, resumo de teste em uma linha, e preocupações.

**Files:** Modify `internal/service/`, `internal/mcpsrv/`, testes
**Commit:** `feat(mcpsrv): note_delete with trash and prior broken-link report`

---

