### Task 185: `note_move` de uma nota que cita a si mesma pelo nome

Achado na rodada de correcao da Task 182, medido pelo implementador: uma nota `a.md` cujo corpo tem `[[a]]`, movida para `sub/a.md`, derruba `note_move` com `MoveNote: lendo nota "a.md": ... The system cannot find the file specified`. O mecanismo e o mesmo do F1 da revisao 182: a propria nota entra em `affectedNotes` como citante, `moverCorpo` renomeia antes, e o laco de reescrita le o caminho antigo. Pre-existente — nada da Task 182 toca link com alvo escrito —, mas o release sai com `note_move`, e o custo e um move pela metade.

**Files:**
- Modify: `internal/service/write.go` (laco de reescrita de citantes em `MoveNote`, ~linhas 481–524 e 624–632)
- Test: `internal/service/anchor_selfref_test.go` (caso novo ao lado dos da rodada 1)

**Interfaces:** nenhuma nova. Consome `index.Backlinks`, `writer.RewriteLinks`/`BuildLinkText` como hoje.

- [ ] **Step 1: Teste (falha antes)** — `a.md` com `# A

Veja [[a]].
`, `MoveNote` para `sub/a.md`: sem erro; `sub/a.md` existe; `a.md` nao existe; o corpo novo tem um link cujo `Resolved` (via `index`) e `sub/a.md` — afirme pelo indice, nao pela grafia, porque `[[a]]` continua resolvendo por nome e a reescrita pode ou nao mudar o texto.
- [ ] **Step 2: Rodar e ver falhar** — `go test ./internal/service -run TestMoveNote_NotaQueCitaASiMesma -v`; esperado: o ENOENT acima.
- [ ] **Step 3: Corrigir** — no laco de reescrita, quando `bl.From == canonicalFrom`, ler e gravar em `canonicalTo` (o arquivo ja esta la). Uma conta: a decisao de "onde a nota esta agora" fica numa variavel local usada nos dois pontos (leitura e escrita), nao em dois `if`. Comentario com o defeito.
- [ ] **Step 4: Rodar e ver passar**; rode tambem os dois testes da rodada 1 (`TestMoveNote_LinkSoDeAncoraNaoQuebraOMove`, `TestDeleteNote_NaoAcusaAPropriaNotaApagada`) e complete o discriminante que a revisao 182 pediu: o teste do F1 passa a afirmar que `[[a]]` dentro de `a.md` FOI reescrita/continua resolvendo apos o move (a metade que a rodada 1 nao pode fixar).
- [ ] **Step 5: Prova de mutacao** — remover o redirecionamento para `canonicalTo` -> o teste novo falha com o ENOENT. Saida colada.
- [ ] **Step 6: Gate e commit** — `verify.ps1` completo; `fix(write): note_move rewrites the moved note's own self-references at its new path`.

---

## Self-review

- Cobertura do pedido: sites -> medido, nao contam (Spec); falsos positivos de ancora -> Task 182 (e o defeito pre-existente de note_move que ela expos -> Task 185); tool de cofre inteiro -> Task 183; docs, build -> Task 184; release e commit -> orquestrador (tag `v1.5.0`, minor por `feat`).
- Tipos: `BrokenLinksRequest/BrokenLink/BrokenLinksResult` e `vaultBrokenLinksInput` aparecem so na Task 183 e no handler; `parser.Link.Anchor` e o unico contrato que a 182 muda e a 183 consome.
- Placeholders: nenhum "TBD"; os nomes de funcao de parse e de apoio de teste sao "leia antes" por serem conhecidos do repo, com o corpo do teste dado.
