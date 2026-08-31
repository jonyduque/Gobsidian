---
title: Wiki — Gobsidian
type: index
status: active
language: pt-BR
last_commit: f7de8e818fba6a831cd07592f15fd978332618ba
updated_at: '2026-08-31'
---

# Gobsidian

> Wiki do codebase. Fonte da verdade é o repositório; estas páginas são a
> camada derivada que explica o porquê.
>
> A **especificação** vive em `docs/` — `PRD.md` (requisitos e decisões
> fechadas), `ARCHITECTURE.md`, `TOOLS.md` (contrato de cada tool),
> `ESTRUTURA.md`, `WINDOWS.md`, `OPERACAO.md`. Este wiki não os substitui:
> ele explica o código que os implementa.

## O que é isto?

Um servidor **MCP** (Model Context Protocol) escrito em Go que expõe um cofre
local do **Obsidian** — uma pasta de arquivos `.md` — a um host de IA como o
Claude Desktop.

Ele roda como subprocesso, conversa por **stdio** (JSON-RPC no stdout), e
oferece 12 ferramentas: 7 de leitura e 5 de escrita. O ponto do produto é que
o modelo do outro lado consegue buscar, ler seções específicas e editar notas
sem que ninguém precise despejar o cofre inteiro no contexto.

## Como começar

- **Comando principal:** `gobsidian serve --vault "C:\caminho\do\cofre"`
- **Precisa de:** Go 1.25.0 ou mais novo (o toolchain instalado hoje é 1.26.5).
  Nenhum serviço externo, nenhum banco, nenhuma rede.
- **Saída esperada:** no stderr, `servidor pronto vault=... notes=N index_ms=…`.
  No stdout, só JSON-RPC — **um byte estranho ali corrompe a sessão**.
- **Primeiros arquivos para ler:** `cmd/gobsidian/main.go`,
  `cmd/gobsidian/servico.go` (a sequência de boot), `internal/service/service.go`
  (a fachada que cada tool chama).
- **Uma primeira mudança segura:** acrescentar um campo a uma resposta de tool
  em `internal/service/` e expô-lo em `internal/mcpsrv/`.
- **Uma mudança tentadora e perigosa:** `go mod tidy`. Ele remove dependências
  fixadas que ainda não têm importador. Ver [Regras não negociáveis](risks/regras-nao-negociaveis.md).

Detalhes em [Como rodar](overview/como-rodar.md).

## Por que existe?

Um cofre real tem milhares de notas e dezenas de MB. Colar isso num prompt é
impossível, e deixar o modelo abrir arquivo por arquivo é lento e caro.

O servidor mantém em memória **metadados e offsets de byte, nunca conteúdo** —
é essa escolha que faz ler uma seção de 2 KB numa nota de 500 KB custar 2 KB, e
é ela que sustenta o orçamento de memória do produto.

## O que acontece quando eu rodo?

Resumo; detalhe em [Fluxo de boot](flows/boot.md).

1. Resolve a configuração (flag > variável de ambiente > default).
2. Tenta falar com um **daemon** já rodando para aquele cofre por socket Unix.
   Se não houver, tenta iniciar um; se não conseguir, serve neste processo.
3. Monta o **índice de metadados** — do cache em disco, se ele bater com o
   disco; senão varre e parseia o cofre.
4. Registra o **watcher** de arquivos e anuncia as tools.
5. O **índice de busca** é carregado em segundo plano, ou só na primeira busca.

## Onde os dados são gravados?

Nunca dentro do cofre, exceto as notas que o usuário mandou escrever.
Ver [Onde ficam os dados](overview/onde-ficam-os-dados.md).

| O quê | Onde |
|---|---|
| Cache do índice de metadados | `<UserCacheDir>/gobsidian/<hash-do-cofre>/index_cache.gob` |
| Cache do índice de busca | mesma pasta, `inverted_cache.gob` |
| Socket e lock do daemon | diretório de runtime do usuário, `<hash-do-cofre>.sock` |
| Lixeira | `.trash/` **dentro** do cofre |

## Quais são as peças importantes?

- [Camadas e fronteiras](concepts/camadas-e-fronteiras.md) — quem pode importar o quê.
- [Os dois índices](concepts/os-dois-indices.md) — metadados e busca, e por que são separados.
- [Busca](features/busca.md) · [Escrita](features/escrita.md) · [Watcher](features/watcher.md)
- [Daemon e ponte](features/daemon-e-ponte.md) — como N sessões compartilham um índice.
- [As 12 tools](features/tools-mcp.md)

## O que eu não devo quebrar?

Quatro regras que derrubam o produto em silêncio se violadas — stdout, rede,
tipos do SDK e `ctx`. Estão em [Regras não negociáveis](risks/regras-nao-negociaveis.md).

E [Armadilhas já pagas](risks/armadilhas-pagas.md): defeitos que passaram por
revisão e só apareceram em produção. Estão escritos para não voltarem.

## Por onde eu começo a ler?

1. Esta página.
2. [Camadas e fronteiras](concepts/camadas-e-fronteiras.md) — o mapa mental.
3. [Fluxo de boot](flows/boot.md) — o que roda, em que ordem.
4. A feature que você vai tocar.
5. [Regras não negociáveis](risks/regras-nao-negociaveis.md) antes de commitar.

A auditoria de 2026-08-25 está **fechada** (3 achados rejeitados com
fundamento); da revisão de 2026-08-15 **quatro tarefas nunca foram entregues**.
As duas listas, com a conferência feita no código, estão em
[Achados da revisão e da auditoria](notes/achados-abertos.md); o resto vive em
`docs/ESTADO.md` § Dívidas abertas.

**Seis páginas estão marcadas `status: stale`** — os arquivos que elas citam
mudaram e elas não foram conferidas nesta passagem. Página stale é ponto de
partida, não resposta.

---

## Catálogo

<!-- wiki:catalog:start -->
### Visão geral

- [Como rodar](overview/como-rodar.md) — Subcomandos, flags, configuração e a bateria de verificação.
- [Onde ficam os dados](overview/onde-ficam-os-dados.md) — Caches, socket, lock, lixeira e temporários — o que o servidor grava e onde.

### Funcionalidades

- [As 12 tools MCP](features/tools-mcp.md) — Superfície pública do servidor — 7 tools de leitura, 5 de escrita, e os resources.
- [Busca](features/busca.md) — Tokenização, ranking BM25, recorte de trecho e o cache de duas camadas.
- [Daemon e ponte](features/daemon-e-ponte.md) — Como N sessões do host compartilham um índice — socket AF_UNIX, handshake e ociosidade.
- [Escrita](features/escrita.md) — As cinco tools de escrita, gravação atômica, travas por caminho e preservação de EOL/BOM.
- [Parser](features/parser.md) — Markdown do Obsidian — frontmatter, headings com offsets e quatro extensões goldmark.
- [Watcher](features/watcher.md) — Fachada sobre fsnotify — filtro, debounce, correlação de rename e reconciliação.

### Fluxos

- [Encerramento](flows/encerramento.md) — Os quatro mecanismos que garantem que o processo morre, e o desligamento por etapas.
- [Fluxo de boot](flows/boot.md) — O que roda entre o host lançar o processo e as tools serem anunciadas.

### Entidades

- [Formato do cache de busca](entities/formato-do-cache.md) — Codec binário próprio, arrays achatados e arena mapeada em memória.
- [Nota, anexo e caminho canônico](entities/note-e-caminho.md) — Os tipos centrais do domínio e as regras de confinamento de caminho.

### Conceitos

- [Camadas e fronteiras](concepts/camadas-e-fronteiras.md) — Quem pode importar o quê, e por que cada fronteira existe.
- [Os dois índices](concepts/os-dois-indices.md) — Metadados e busca são estruturas separadas, com custos e ciclos de vida diferentes.

### Referência

- [Gates e scripts](reference/gates-e-scripts.md) — O que cada script em scripts/ verifica, e o que acontece quando ele não roda.

### Riscos

- [Armadilhas já pagas](risks/armadilhas-pagas.md) — Defeitos que passaram por revisão e só apareceram depois. Estão aqui para não voltarem.
- [Regras não negociáveis](risks/regras-nao-negociaveis.md) — O que derruba o produto em silêncio se for violado.

### Decisões

- [Decisões fechadas](decisions/decisoes-fechadas.md) — Escolhas com trade-off registrado, que não devem ser re-litigadas sem dado novo.

### Notas

- [Achados da revisão e da auditoria](notes/achados-abertos.md) — A auditoria fechou; da revisão restam quatro tarefas. Registra também os três rejeitados com fundamento.
- [Incidente de campo de 2026-08-15](notes/incidente-de-campo-2026-08-15.md) — Sessao real do Claude Desktop em que o modelo abandonou o gobsidian e usou outro servidor MCP, e os cinco defeitos que a causaram.
- [Medições de 2026-08-15](notes/medicoes-2026-08-15.md) — A/B de toolchain 1.25.6 x 1.26.5 e re-aferição do GOGC — as únicas medições da revisão de 2026-08-15.
<!-- wiki:catalog:end -->
