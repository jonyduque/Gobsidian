### Task 186: RF-63 nomeia as quatro formas; `[x](b.md#)` ganha a medição

**Files:**
- Modify: `docs/PRD.md:168` (linha da tabela RF-63) e `docs/PRD.md:208` (parágrafo "RF-63 vai além…")
- Modify: `docs/ESTADO.md:628-635` (item `[x](b.md#)`)

**Interfaces:** nenhuma. Docs apenas.

- [ ] **Step 1: Ler o que existe.** `sed -n 160,210p docs/PRD.md` e `sed -n 600,640p docs/ESTADO.md`. Confirmar que RF-63 hoje cita só `[[nota#heading]]` e `[[nota#^bloco]]`, e que o item `[x](b.md#)` diz "não foi medida em cofre real".

- [ ] **Step 2: Reescrever a linha RF-63** em `docs/PRD.md:168`. Manter o ID, a prioridade P1 e o formato da tabela. Texto:

```
| RF-63 | Validação da âncora em **todas** as formas que a carregam — `[[nota#heading]]`, `[[nota#^bloco]]`, `[texto](nota.md#heading)`, `![[nota#heading]]`/`![texto](nota.md#heading)` e as auto-referências `[[#heading]]`/`[texto](#heading)`, que resolvem para a própria nota — marcadas como âncora quebrada quando a nota resolve mas o alvo interno não existe. A separação do `#` é uma conta só, antes do percent-decode; `%23` nunca é separador | P1 |
```

- [ ] **Step 3: Acrescentar ao parágrafo de `docs/PRD.md:208`** ("RF-63 vai além do que o próprio Obsidian expõe…") uma frase final, sem alterar as existentes:

```
Até 2026-09-06 o requisito só nomeava o wikilink, e as formas Markdown e de auto-referência contavam como **alvo ausente** — um cofre real carregava 267 delas, outro 372 (medição em `docs/ESTADO.md`). O requisito nomeia as formas para que a próxima regressão tenha nome.
```

- [ ] **Step 4: Atualizar o item `[x](b.md#)` em `docs/ESTADO.md:628-635`.** Substituir a frase "Fidelidade mínima perdida numa forma que **não foi medida em cofre real**." por:

```
Fidelidade mínima perdida numa forma que, **medida em 2026-09-07 em cinco cofres reais** (Estudo, Jurisprudência, Oral, Revisão, _automacao), tem **zero** ocorrências internas: os dois únicos acertos de `\]\([^) ]*#\)` são URLs `http://...#mce_temp_url#` em Jurisprudência — externas, que `note_move` nunca reescreve. `[[b#]]` deu zero nos cinco.
```

Manter "**Parqueado por decisão**, não esquecido" e o restante do item. Trocar "cuja frequência é desconhecida" por "cuja frequência medida é zero".

- [ ] **Step 5: Validar encoding e referências.**

```bash
python -c "open('docs/PRD.md',encoding='utf-8').read()" && echo "[OK] UTF-8 valido"
python -c "open('docs/ESTADO.md',encoding='utf-8').read()" && echo "[OK] UTF-8 valido"
pwsh -File scripts/check_doc_refs.ps1; echo EXIT=$?
```

Esperado: os dois `[OK]`, `EXIT=0`.

- [ ] **Step 6: Commit** (mensagem em `.superpowers/sdd/2026-09-07-gates-e-rf63/commit-186.txt`):

```
docs(prd): RF-63 names every anchor form, and the empty anchor is measured

RF-63 named only the wikilink; the Markdown and self-reference forms were
counted as missing targets until 2026-09-06. The requirement now names
all of them so the next regression has a name.

The empty-anchor form `[x](b.md#)` stays parked, now with a measurement:
zero internal occurrences across five real vaults.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01P5wkw6PAdBFzF3uB1w1jNj
```

```bash
git add docs/PRD.md docs/ESTADO.md
git commit -F .superpowers/sdd/2026-09-07-gates-e-rf63/commit-186.txt
```

---

