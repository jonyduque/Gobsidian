### Task 184: Revisao de documentacao e preparo do release

**Files:**
- Modify: `docs/ESTADO.md` (marco: 2026-09-06, Tasks 182–183, `IndexCacheParserVersion` 2), `docs/OPERACAO.md` (se listar tools ou versao de cache), `docs/ARCHITECTURE.md` (se descrever resolucao de link/ancora — o texto tem de dizer que ancora vale para os tres tipos), `docs/TOOLS.md` (revisao: `vault_stats.broken_links` descreve o que conta; `link_graph.include_broken` cita a tool nova como o caminho de cofre inteiro), `docs/wiki/*` paginas que descrevem parser de links ou vault_stats (marque `status: stale` so se nao corrigir), `docs/ESTRUTURA.md` (`.github/workflows/release.yml` esta ausente da arvore — inclua; `internal/service/broken.go`)
- Verify: `pwsh -File scripts/check_doc_refs.ps1`, `pwsh -File scripts/check_readme_anchors.ps1`, UTF-8 de cada `.md`

**Interfaces:** nenhuma; le o diff das Tasks 182–183 (`git log -p <BASE>..HEAD`, BASE informado no despacho).

- [ ] **Step 1: Inventario** — `grep -rn "broken_links\|anchor_missing\|Anchor\|ancora" docs/ README.md` e listar no relatorio cada ocorrencia com "certo / precisa mudar / mudado".
- [ ] **Step 2: Editar** cada lugar listado como "precisa mudar". Sem numero que nao veio da medicao registrada no plano (secao Spec) ou do proprio diff; o resto e "nao medido".
- [ ] **Step 3: Gates de doc** — `check_doc_refs.ps1`, `check_readme_anchors.ps1`, UTF-8.
- [ ] **Step 4: Build** — `pwsh -File scripts/build.ps1`; cole a linha `[...] Compilando <versao> (<commit>)` e a saida de `gobsidian version` do binario gerado no relatorio.
- [ ] **Step 5: Commit** — `docs: anchor links, vault_broken_links, and the release workflow in the tree`. Nao cria tag; a tag e o push sao do orquestrador.

---

## Self-review

- Cobertura do pedido: sites -> medido, nao contam (Spec); falsos positivos de ancora -> Task 182; tool de cofre inteiro -> Task 183; docs, build -> Task 184; release e commit -> orquestrador (tag `v1.5.0`, minor por `feat`).
- Tipos: `BrokenLinksRequest/BrokenLink/BrokenLinksResult` e `vaultBrokenLinksInput` aparecem so na Task 183 e no handler; `parser.Link.Anchor` e o unico contrato que a 182 muda e a 183 consome.
- Placeholders: nenhum "TBD"; os nomes de funcao de parse e de apoio de teste sao "leia antes" por serem conhecidos do repo, com o corpo do teste dado.
