## Progresso

- 10:23 — Leitura do brief e arquivos existentes
- 10:24 — Edições de RF-63 em docs/PRD.md (linhas 168 e 208)
- 10:25 — Edição de [x](b.md#) em docs/ESTADO.md (linhas 628-635)
- 10:26 — Validações de UTF-8 e referências
- 10:27 — Criação e execução do commit

## Status

DONE

## Commits

c642e18 docs(prd): RF-63 names every anchor form, and the empty anchor is measured

## Diff

```
commit c642e183c7dad9f792762778804fb94418618b1a
Author: jonyduque <jonyduque@hotmail.com>
Date:   Mon Sep 7 10:25:05 2026 -0300

    docs(prd): RF-63 names every anchor form, and the empty anchor is measured
    
    RF-63 named only the wikilink; the Markdown and self-reference forms were
    counted as missing targets until 2026-09-06. The requirement now names
    all of them so the next regression has a name.
    
    The empty-anchor form `[x](b.md#)` stays parked, now with a measurement:
    zero internal occurrences across five real vaults.
    
    Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
    Claude-Session: https://claude.ai/code/session_01P5wkw6PAdBFzF3uB1w1jNj

 docs/ESTADO.md | 4 ++--
 docs/PRD.md    | 4 ++--
 2 files changed, 4 insertions(+), 4 deletions(-)
```

### Diff detalhado

```
diff --git a/docs/ESTADO.md b/docs/ESTADO.md
index 7730bd7..2495b0a 100644
--- a/docs/ESTADO.md
+++ b/docs/ESTADO.md
@@ -628,9 +628,9 @@ o quarto, que é exatamente o defeito que ela existe para impedir.
 - **`[x](b.md#)` — âncora vazia depois do `#` — perde o `#` na reescrita.**
   `splitAnchor` devolve `("b.md", "")` e `anchorMarkdown` devolve `""` para
   âncora vazia, então um `note_move` reescreve `[x](b.md#)` como `[x](c.md)`.
-  Fidelidade mínima perdida numa forma que **não foi medida em cofre real**.
+  Fidelidade mínima perdida numa forma que, **medida em 2026-09-07 em cinco cofres reais** (Estudo, Jurisprudência, Oral, Revisão, _automacao), tem **zero** ocorrências internas: os dois únicos acertos de `\]\([^) ]*#\)` são URLs `http://...#mce_temp_url#` em Jurisprudência — externas, que `note_move` nunca reescreve. `[[b#]]` deu zero nos cinco.
   **Parqueado por decisão**, não esquecido: não vale um ramo a mais no formatador
-  por uma forma cuja frequência é desconhecida. Se aparecer, o conserto é
+  por uma forma cuja frequência medida é zero. Se aparecer, o conserto é
   distinguir "sem âncora" de "âncora vazia" no `parser.Link`, que hoje são a
   mesma coisa.
 
diff --git a/docs/PRD.md b/docs/PRD.md
index 5b571a0..2961a98 100644
--- a/docs/PRD.md
+++ b/docs/PRD.md
@@ -165,7 +165,7 @@ Um wikilink não é um caminho, e resolvê-lo como se fosse produz um grafo que
 | RF-60 | Indexação de anexos (`.png`, `.jpg`, `.jpeg`, `.gif`, `.webp`, `.svg`, `.pdf`, `.mp3`, `.mp4`, `.wav`, `.canvas`) por caminho, tamanho e mtime, sem leitura de conteúdo | P0 |
 | RF-61 | Resolução de embed para anexo, de modo que `![[diagrama.png]]` não seja contabilizado como link quebrado | P0 |
 | RF-62 | Resolução de wikilink pelo campo `aliases` do frontmatter da nota alvo — **divergência deliberada do Obsidian**, ver abaixo | P0 |
-| RF-63 | Validação da âncora: `[[nota#heading]]` e `[[nota#^bloco]]` marcados como âncora quebrada quando a nota resolve mas o alvo interno não existe | P1 |
+| RF-63 | Validação da âncora em **todas** as formas que a carregam — `[[nota#heading]]`, `[[nota#^bloco]]`, `[texto](nota.md#heading)`, `![[nota#heading]]`/`![texto](nota.md#heading)` e as auto-referências `[[#heading]]`/`[texto](#heading)`, que resolvem para a própria nota — marcadas como âncora quebrada quando a nota resolve mas o alvo interno não existe. A separação do `#` é uma conta só, antes do percent-decode; `%23` nunca é separador | P1 |
 | RF-64 | Chaves de resolução normalizadas para **NFC**, de modo que uma nota gravada em NFD seja encontrada por um pedido em NFC e vice-versa | P0 |
 | RF-65 | Colisão de chave de resolução — por caixa ou por normalização — devolve ambiguidade nomeada, nunca um dos candidatos escolhido em silêncio | P0 |
 | RF-66 | Resolução por nome de arquivo insensível a maiúsculas, igual à resolução por caminho completo | P1 |
@@ -205,7 +205,7 @@ A consequência precisa ser sabida por quem opera: `note_metadata` e `link_graph
 
 O que **não** muda: alias continua sendo *fallback*, nunca *override*. Se existe `P3.md` e outra nota declara `aliases: [P3]`, `[[P3]]` aponta para o arquivo. A paridade confirmou essa precedência nos dois lados.
 
-RF-63 vai além do que o próprio Obsidian expõe na interface, e é deliberado: uma âncora quebrada é exatamente o tipo de erro que aparece depois de renomear um heading, e é invisível até alguém clicar no link.
+RF-63 vai além do que o próprio Obsidian expõe na interface, e é deliberado: uma âncora quebrada é exatamente o tipo de erro que aparece depois de renomear um heading, e é invisível até alguém clicar no link. Até 2026-09-06 o requisito só nomeava o wikilink, e as formas Markdown e de auto-referência contavam como **alvo ausente** — um cofre real carregava 267 delas, outro 372 (medição em `docs/ESTADO.md`). O requisito nomeia as formas para que a próxima regressão tenha nome.
 
 ### 5.3 Busca
```

## Verification

[OK] UTF-8 valido (docs/PRD.md)
[OK] UTF-8 valido (docs/ESTADO.md)

```
Carregado em 962ms
Carregamento: 1023,4 ms 

[i] corpus: 383 arquivos .go.

[i] 37 dispensa(s) em uso -- nao contam como achado:
  docs\ARCHITECTURE.md:394: `backend_inotify.go` -- arquivos do fsnotify v1.10.1, dependencia externa
  docs\ARCHITECTURE.md:394: `backend_windows.go` -- arquivos do fsnotify v1.10.1, dependencia externa
  docs\ARMADILHAS.md:351: `run_in_background` -- parametro da ferramenta Bash do Claude Code, nao identificador deste codebase
  docs\ESTRUTURA.md:268: `interfaces.go` -- exemplos de categoria sintatica que o projeto evita, nao arquivos reais
  docs\ESTRUTURA.md:268: `helpers.go` -- exemplos de categoria sintatica que o projeto evita, nao arquivos reais
  docs\ESTRUTURA.md:270: `helpers.go` -- a propria frase afirma que NAO existem; e a regra, nao uma referencia
  docs\ESTRUTURA.md:270: `utils.go` -- a propria frase afirma que NAO existem; e a regra, nao uma referencia
  docs\ESTRUTURA.md:316: `snake_case.go` -- exemplo da convencao de nome, nao arquivo
  docs\OPERACAO.md:1526: `cmd/gobsidian/servico.go` -- registro historico de 2026-08-28: o arquivo existia entao, e a Task 174 o moveu para internal/boot/montar.go
  docs\REVISAO-2026-08-15.md:24: `max_results` -- flag de CLI; a frase existe para dizer que NAO e campo de schema
  docs\REVISAO-2026-08-15.md:202: `max_results` -- flag de CLI que nunca foi campo de schema; a Task 120 decide implementar ou remover
  docs\REVISAO-2026-08-15.md:575: `next_offset` -- campo que a Task 106 vai criar; a ausencia dele E o achado
  docs\REVISAO-2026-08-15.md:734: `next_offset` -- campo que a Task 106 vai criar; a ausencia dele E o achado
  docs\REVISAO-2026-08-15.md:790: `vault_lint` -- tool proposta na Fase 9; ainda nao existe, e esse E o ponto
  docs\REVISAO-2026-08-15.md:811: `next_offset` -- campo que a Task 106 vai criar; a ausencia dele E o achado
  docs\REVISAO-2026-08-15.md:851: `from_text` -- alternativa F, REJEITADA; o nome nunca deve existir no codigo
  docs\REVISAO-2026-08-15.md:869: `next_offset` -- campo que a Task 106 vai criar; a ausencia dele E o achado
  docs\REVISAO-2026-08-15.md:938: `note_rename` -- tool proposta na Fase 9; ainda nao existe, e esse E o ponto
  docs\REVISAO-2026-08-15.md:1378: `check_doc_refs` -- scripts PowerShell em scripts/; o corpus deste checador e so .go
  docs\REVISAO-2026-08-15.md:1378: `check_readme_anchors` -- scripts PowerShell em scripts/; o corpus deste checador e so .go
  docs\REVISAO-2026-08-15.md:1422: `vault_lint` -- tool proposta na Fase 9; ainda nao existe, e esse E o ponto
  docs\SUGESTOES.md:85: `check_doc_refs` -- scripts PowerShell em scripts/; o corpus deste checador e so .go
  docs\SUGESTOES.md:86: `check_readme_anchors` -- scripts PowerShell em scripts/; o corpus deste checador e so .go
  docs\SUGESTOES.md:86: `check_tool_params` -- scripts PowerShell em scripts/; o corpus deste checador e so .go
  docs\SUGESTOES.md:109: `max_results` -- flag de CLI que define o teto; nunca foi campo de schema
  docs\SUGESTOES.md:356: `vault_info` -- tool PROPOSTA pelo M17; a ausencia dela e o achado
  docs\SUGESTOES.md:488: `index/assets.go` -- FECHADO em 2026-08-28: o arquivo foi apagado, e e por isso que a referencia nao resolve
  docs\SUGESTOES.md:574: `vault_lint` -- tool proposta na Fase 9; ainda nao existe, e esse e o ponto
  docs\SUGESTOES.md:974: `assets.go` -- arquivo apagado; a referencia e historica
  docs\SUGESTOES.md:1064: `vault_info` -- tool PROPOSTA pelo M17; a ausencia dela e o achado
  docs\SUGESTOES.md:1073: `vault_info` -- tool PROPOSTA pelo M17; a ausencia dela e o achado
  docs\SUGESTOES.md:1435: `vault_info` -- tool PROPOSTA pelo M17; a ausencia dela e o achado
  docs\SUGESTOES.md:1435: `vault_info` -- tool PROPOSTA pelo M17; a ausencia dela e o achado
  docs\TOOLS.md:64: `max_results` -- flag de CLI da configuracao que define o teto maximo do limit
  docs\TOOLS.md:119: `total_bytes` -- este bloco existe para dizer que o campo NAO existe no retorno
  docs\WINDOWS.md:157: `max_user_watches` -- sysctl do Linux (fs.inotify.max_user_watches), nome externo
  README.md:76: `node_modules` -- diretorio do Node, citado para dizer que o instalador nao cria um

[OK] nenhum token entre crases parece citar artefato ausente do codigo.
EXIT=0
```

## Concerns

Nenhuma. Todas as edições foram feitas exatamente conforme o brief, validações passam, e o commit foi bem-sucedido.
