### Task 64: `internal/writer/linkrewrite.go`

R1. Reescreve links preservando alias, âncora e a grafia original.

#### O que vincula esta tarefa

- **`Link.Raw` é o insumo, não os campos.** Reconstruir o link a partir de `Target`/`Alias`/`Anchor` perde o que o usuário escreveu.
- **A reescrita passa pelos `Links` da nota de origem, não pelo `Backlink`.** `index.Backlink` tem `From`, `Anchor`, `Alias`, `Context` e `Kind` — **não tem `Raw` nem offsets**. O caminho é `idx.Get(bl.From).Links`, onde cada `ResolvedLink` embute o `parser.Link` inteiro.
- **Reescreva de trás para a frente.** Substituir por offset da esquerda para a direita invalida os offsets seguintes da mesma nota, e produz nota corrompida em vez de erro.
- **Link com `Start == -1` não pode ser reescrito.** Cheque o offset **e** o `Kind`; confiar só no `Kind` quebra no dia em que a Task 63 deixar um caso de fora.

#### O que já está fechado

- **`Link.Raw` é o insumo**, não o alvo reconstruído. Reconstruir `[[alvo|alias#ancora]]` a partir dos campos perde o que o usuário escreveu — espaço interno, grafia de caixa, extensão explícita ou não. `Raw` é o texto original e existe para isto.
- **A reescrita passa pelos `Links` da nota de origem** (decisão 2 acima), não pelo `Backlink`.
- **Reescrever de trás para a frente.** Substituir por offset da esquerda para a direita invalida todos os offsets seguintes da mesma nota. Ordene as substituições por `Start` decrescente. É a forma clássica de errar em reescrita por offset, e ela produz nota corrompida, não erro.

#### A armadilha

**Um link com `Start == -1` não pode ser reescrito.** Depois da Task 63 devem ser poucos ou nenhum, mas o código **tem de checar** e reportar em vez de fatiar a partir de -1. Verifique com o `Kind` **e** com o offset; confiar só no `Kind` quebra no dia em que a Task 63 deixar um caso de fora.

#### Verificações além dos passos

- Alias preservado: `[[Civil/PONTO 03|Ponto 3 — Obrigações]]` continua com o mesmo alias?
- Âncora de heading e de bloco preservadas: `[[a#Seção]]` e `[[a#^bloco]]`?
- Link Markdown vira link Markdown, wikilink vira wikilink — a grafia não troca?
- Embed (`![[x]]`) continua embed?
- Duas ocorrências do mesmo link na mesma nota: as duas reescritas, e os offsets não se atropelam?
- Link que resolve por **alias** e não por caminho: é reescrito? Deve? **Decida e diga** — um link por alias continua válido depois do move, e reescrevê-lo mudaria o que o usuário escreveu sem necessidade.
- EOL e BOM preservados byte a byte.

**Prova de mutação obrigatória:** inverta a ordem das substituições (frente para trás) e confirme que um teste com duas ocorrências reprova. É a prova de que a ordem está sendo exercitada, não só escrita.

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

Grave em `.superpowers/sdd/task-64-report.md`: o que implementou; evidência de TDD (comando e saída do RED, comando e saída do GREEN); a prova de mutação com a **saída real colada**, nomeando o teste que reprovou; a tabela de verificações acima com o resultado **real** de cada uma, inclusive as que deram certo; arquivos alterados; o que ficou de fora e por quê; e `git status --porcelain`.

**Não escreva número que você não mediu** — escreva "não medido". **Prova de mutação no condicional não é prova:** use `scripts/mutate.ps1` e cole o que ele imprimiu. Rode `pwsh -File scripts/audit_reports.ps1 64` antes de entregar.

Responda com no máximo 15 linhas: status (`DONE` | `DONE_WITH_CONCERNS` | `BLOCKED` | `NEEDS_CONTEXT`), commit criado, resumo de teste em uma linha, e preocupações.

**Files:** Create `internal/writer/linkrewrite.go`, `linkrewrite_test.go`
**Commit:** `feat(writer): faithful link rewriting from Link.Raw`

---

