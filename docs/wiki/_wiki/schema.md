# Schema deste wiki

Este arquivo torna o wiki autodescritivo: um agente que o leia consegue operar
aqui **sem nenhuma skill carregada**. Se as convenções mudarem, atualize este
arquivo — ele é a autoridade local.

## Identidade

| Campo | Valor |
|---|---|
| Raiz do wiki | `docs/wiki` |
| Raiz do repositório | `..`/`../..` (dois níveis acima) |
| Idioma da prosa | pt-BR |
| Identificadores e caminhos | sempre no original, nunca traduzidos |
| Modo de link | Markdown relativo padrão, com `.md` |
| Entry point | `Home.md` |
| Navegação | `_Sidebar.md` (gerado) |
| Detecção de mudança | `git diff` desde `Home.md:last_commit`; fallback `_wiki/manifest.json` |
| Configuração executável | `_wiki/config.json` |

## Este arquivo descreve; `config.json` executa

Este `schema.md` é **prosa**: ele documenta as convenções para quem lê. Nenhum
script o interpreta. O que as ferramentas de fato leem é `_wiki/config.json`:

```json
{
  "exclude": ["agent", "packages/legacy-*"],
  "size_targets": { "feature": 700 },
  "language": "pt-BR"
}
```

- `exclude` — globs de caminho relativo, **além** das exclusões padrão (harness
  de agente, dependências, build, e o que o `.gitignore` cobre).
- `size_targets` — alvo de palavras de **prosa** por `type` de página.

Se você mudar uma convenção aqui e ela tiver efeito em ferramenta, mude nos
dois lugares. Só este arquivo mudado é documentação que não vale.

## Dispensa de referência a código

O verificador (`wiki_doctor.py refs`) acusa token entre crases que parece
identificador e não existe em fonte nenhuma. Para dispensar, escreva na linha da
afirmação (ou na vizinha), **com motivo obrigatório**:

```markdown
<!-- wiki-refs: ignore max_results net.Dialer -- API que o projeto ainda não usa -->
```

Sem o `-- <motivo>` vira `DISPENSA-INVALIDA` e o token continua acusado. As
dispensas usadas são impressas a cada rodada, de propósito.

## Criar página

```powershell
python <skill>/scripts/wiki_new.py <repo> --type feature --title "X" --apply
```

Acerta pasta, `type`, `source_commit` e data. O corpo vem dos templates da
skill — não cole frontmatter à mão.

## Regra dura

**Toda página tem frontmatter YAML com `type`.** Só isso. Campo opcional
ausente, `type` desconhecido, chave extra e link quebrado **não são erro** — um
leitor permissivo nunca deve falhar por causa deles.

## Frontmatter

```yaml
---
title: "Nome de exibição"
type: "feature"
status: "active"            # draft | active | stale
description: "Uma frase."
source_paths:               # caminhos do repo que esta página documenta
  - src/middleware/auth.ts
source_commit: "<sha>"      # commit dos fontes quando a página foi escrita
tags: [auth]
language: "pt-BR"
updated_at: "2026-08-12"
---
```

`source_paths` é o eixo do wiki: é o que permite detectar página defasada e
código sem cobertura. Páginas abstratas podem tê-lo vazio.

## Tipos de página

| `type` | Para quê | Pasta |
|---|---|---|
| `index` | Entry point e catálogo | raiz (`Home.md`) |
| `overview` | O que é, por que existe, como rodar | `overview/` |
| `feature` | Uma capacidade ou área de implementação | `features/` |
| `flow` | Sequência, pipeline, ciclo de vida | `flows/` |
| `entity` | Modelo de dado, schema, tipo | `entities/` |
| `concept` | Ideia transversal, padrão | `concepts/` |
| `reference` | Página ancorada num caminho de código | `reference/` |
| `risk` | Invariante, contrato, área frágil | `risks/` |
| `decision` | ADR-lite: decisão e trade-offs | `decisions/` |
| `bug-fix` | Correção notável do histórico | `bug-fixes/` |
| `note` | Pergunta aberta | `notes/` |

## Links

Markdown relativo padrão, resolvido da pasta da página, **sempre com `.md`**:

```markdown
[Middleware de autenticação](../features/auth.md)
[Passos do login](../flows/login.md#passos)
```

Sem wikilinks `[[...]]`. Caminhos de código aparecem como texto
(`src/api/routes.ts:42`), não como link.

Link quebrado = conhecimento ainda não escrito. **Não remova.**

## `Home.md`

Responde, nesta ordem: o que é · como começar · por que existe · o que acontece
ao rodar · onde os dados ficam · peças importantes · o que não quebrar · por onde
ler primeiro.

Contém a seção de catálogo entre `<!-- wiki:catalog:start -->` e
`<!-- wiki:catalog:end -->`. **O miolo é gerado; o resto é prosa humana e não
deve ser sobrescrito.**

Frontmatter do `Home.md` carrega o checkpoint:

```yaml
last_commit: "<sha completo do HEAD no último ingest completo>"
```

## Sincronia

- **Repo mudou desde o último ingest?** `git diff --name-status -M <last_commit> HEAD`.
  Avance `last_commit` só após ingest completo.
- **Esta página defasou?** Compare `source_commit` da página com
  `git log -1 --format=%H -- <source_paths>`.
- Página defasada é **marcada** (`status: stale`) e anotada, nunca apagada.
- O ingest trabalha sobre `HEAD`; mudanças não commitadas são ignoradas.

## Escrita

Escreva para um recém-chegado inteligente. Termo leigo antes do termo técnico.
Estruture pela pergunta do leitor, não pela árvore de diretórios. Resuma em vez
de colar código. Separe fato (lido no código) de inferência (sua interpretação).
Páginas abaixo de ~500 palavras — crie outra em vez de inchar uma.

## Manutenção

Os scripts vivem na skill `codebase-wiki` (`scripts/`) e são todos *dry-run* por
padrão:

| Comando | Para quê |
|---|---|
| `wiki_map.py` | Digest estrutural do repo |
| `wiki_scan.py changes` | O que mudou desde o checkpoint |
| `wiki_scan.py coverage` | Código sem página / `source_paths` mortos |
| `wiki_index.py` | Regenera catálogo e `_Sidebar.md` |
| `wiki_doctor.py check` | Links quebrados, órfãs, defasadas, drift |
| `wiki_stubs.py` | Esqueletos para links quebrados |
| `wiki_export.py` | Bundle de arquivo único |
