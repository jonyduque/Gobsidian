### Task 63: Offsets para `LinkMarkdown` no parser

Pré-requisito do marco. Sem ela, `note_move` reescreve wikilink e deixa link Markdown apontando para o caminho antigo.

#### O que vincula esta tarefa

- **O sentinela é `-1`, não zero.** Zero é posição legítima — o primeiro byte. Onde o offset continuar desconhecido depois desta tarefa, ele continua `-1`, para que um fatiamento estoure alto em vez de sobrescrever o início da nota em silêncio.
- **O parser é folha e está congelado por 48 golden files.** Mudar o que ele emite muda o corpus; `-update` grava o que o código produz, não o que está certo.
- **Link Markdown é reescrito no M5** (decisão fechada): `docs/TOOLS.md` promete preservar a escolha entre wikilink e link Markdown, e é o contrato que o modelo do outro lado lê. É por isso que esta tarefa existe.

#### A evidência do defeito

`internal/parser/types.go` documenta o sentinela e o motivo: a AST do goldmark não entrega o span completo de `[texto](destino)` num nó só. `Start` e `End` ficam em `offsetUnknown = -1` para `LinkMarkdown` e para embeds em grafia Markdown.

**O sentinela é -1 e não zero, de propósito** — zero é posição legítima, e usá-lo para "não sei" repete o defeito que `ReadOnlySet` corrigiu. Com -1, um fatiamento estoura alto em vez de sobrescrever o início da nota em silêncio. **Preserve essa propriedade:** onde o offset continuar desconhecido depois desta tarefa, ele continua -1.

#### O que implementar

Preencha `Start` e `End` para `LinkMarkdown` e para embeds em grafia Markdown. O span vai do `[` (ou `![`) ao `)` final, inclusive.

**A parte que erra é o link aninhado.** `[[[a]] b](d.md)` já é caso de teste deste projeto — a Task 14 o cita. O `]` que fecha o texto não é o primeiro `]` encontrado. Use a estrutura da AST para achar os limites, não varredura de caractere; se a AST não bastar, varra com contagem de profundidade e **teste o aninhado**.

**Se algum caso continuar sem offset confiável, deixe -1 e diga qual.** Metade dos offsets certos e metade errados é pior que metade ausente: com -1 quem reescreve pula; com offset errado, escreve no lugar errado.

#### Verificações além dos passos

- `[texto](destino.md)` simples: `raw[Start:End]` devolve o link inteiro?
- `![alt](img.png)`: idem?
- `[[[a]] b](d.md)`: o span vai até o `)` certo?
- Link com parênteses no destino — `[x](a(b).md)` — e com título: `[x](a.md "t")`?
- Nota **com BOM**: os offsets já vêm com `bodyOffset` somado, como o comentário do campo promete? É a mesma composição que a Task 43 e a 47 exercitaram.
- Quantos goldens do corpus mudaram? Liste-os e **confirme que leu cada `.json`** — `-update` grava o que o código produz, não o que está certo.
- Sobrou algum `Kind` com offset -1? Liste quais e por quê.

**Prova de mutação obrigatória:**

```bash
pwsh -File scripts/mutate.ps1 -Path internal/parser/<arquivo>.go `
  -Anchor '<a atribuicao de Start para LinkMarkdown>' -Replacement '<offsetUnknown>' `
  -Test TestLinkMarkdownOffsets -Package ./internal/parser/
```

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

Grave em `.superpowers/sdd/task-63-report.md`: o que implementou; evidência de TDD (comando e saída do RED, comando e saída do GREEN); a prova de mutação com a **saída real colada**, nomeando o teste que reprovou; a tabela de verificações acima com o resultado **real** de cada uma, inclusive as que deram certo; arquivos alterados; o que ficou de fora e por quê; e `git status --porcelain`.

**Não escreva número que você não mediu** — escreva "não medido". **Prova de mutação no condicional não é prova:** use `scripts/mutate.ps1` e cole o que ele imprimiu. Rode `pwsh -File scripts/audit_reports.ps1 63` antes de entregar.

Responda com no máximo 15 linhas: status (`DONE` | `DONE_WITH_CONCERNS` | `BLOCKED` | `NEEDS_CONTEXT`), commit criado, resumo de teste em uma linha, e preocupações.

**Files:** Modify `internal/parser/` e testes, goldens afetados
**Commit:** `feat(parser): byte offsets for markdown links and embeds`

---

