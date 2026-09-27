# CLI das tools — desenho

**Data:** 2026-09-25. **Pedido do dono:** "todas as tools tenham um comando para uso por CLI" e "uma variável de ambiente para setar o vault padrão, fixado apenas para o CLI". Decisões tomadas na conversa: uso **pelos dois** (humano no terminal e agente/script pelo shell); variável **`GOBSIDIAN_VAULT`**; forma **agrupada** (`gobsidian note read ...`); abordagem **A** (a CLI chama a tool de verdade, em processo); `search`, `index` e `inspect` **mudam de lugar e passam a funcionar igual aos demais**.

## O que o usuário ganha

Cada uma das 14 tools do servidor MCP vira um comando. O resultado é **o mesmo** que o host recebe — mesmos padrões, mesmos tetos, `truncated`, códigos de erro —, porque a CLI chama a tool, não uma cópia dela.

```
gobsidian note read "Civil/Dolo.md" --heading Conceito
gobsidian note list --folder Civil --tags penal
gobsidian note outline|metadata <nota>
gobsidian note create|append|patch|move|delete ...
gobsidian vault stats | search "dolo" | broken-links
gobsidian tag list --prefix direito
gobsidian graph <nota> --depth 2
```

## Mapa tool → comando

| Tool | Comando | Posicional |
|---|---|---|
| `note_read` | `note read` | `path` (vários posicionais viram `paths`) |
| `note_list` | `note list` | — |
| `note_outline` | `note outline` | `path` |
| `note_metadata` | `note metadata` | `path` |
| `note_create` | `note create` | `path` |
| `note_append` | `note append` | `path` |
| `note_patch` | `note patch` | `path` |
| `note_move` | `note move` | `from`, `to` |
| `note_delete` | `note delete` | `path` |
| `vault_stats` | `vault stats` | — |
| `vault_search` | `vault search` | `query` |
| `vault_broken_links` | `vault broken-links` | — |
| `tag_list` | `tag list` | — |
| `link_graph` | `graph` | `path` |

`search` → `vault search`, `inspect` → `note metadata`, `index` → `vault stats`. Os três antigos **saem**: dois nomes para o mesmo comando é a duplicação que o projeto persegue, e o dono autorizou mover. `index` existia também para aquecer o cache; isso continua acontecendo, porque todo comando abre o índice pela mesma conta (`boot.AbrirIndice`), que grava o cache.

## Arquitetura

**`mcpsrv` ganha duas funções, e continua sendo o único pacote com tipos do SDK:**

- `EsquemasDeEntrada() []EsquemaDeTool` — nome, descrição e o schema de entrada de cada tool, em tipos do domínio (nome do parâmetro, tipo, descrição, obrigatório, enum). Monta um `Server` sem serviço só para ler o que foi registrado; nenhuma tool é chamada. É o que a árvore de comandos usa para `--help` e completação sem abrir o cofre.
- `ChamarLocal(ctx, srv, nome, argumentos map[string]any) (Resultado, error)` — conecta um cliente ao servidor por transporte em memória (o mesmo que os testes de `mcpsrv` usam), chama a tool e devolve o `structuredContent` em JSON cru, ou o erro com o código de domínio.

**`cmd/gobsidian` ganha:**

- `ferramentas.go` — a tabela da seção anterior e a construção dos comandos a partir dela e de `EsquemasDeEntrada()`.
- `ferramentas_flags.go` — schema → flags: `string` → texto, `integer` → inteiro, `boolean` → booleano, `array` de `string` → repetível (`--tags a --tags b` ou `--tags a,b`), `object` (o `frontmatter` de `note_list`) → `--frontmatter chave=valor` repetível. `oneOf` (o item de `paths` de `note_read`, string ou objeto) → os posicionais são as strings; a forma objeto só pela válvula `--args`.
- `--args '<json>'` em todo comando: um objeto JSON mesclado **por baixo** das flags (flag vence). É a saída para o que uma flag não expressa, e o jeito de um script passar a entrada inteira.
- `ferramentas_texto.go` — um formatador legível por tool.

**Fluxo de uma chamada:** `carregarConfig` (Parte J: nome ou caminho) → `boot` abre cofre e índice como o `search` de hoje (sem watcher) → `service.New` → `mcpsrv.New` → `ChamarLocal` → formatador. A busca carrega o índice invertido só quando a tool é `vault_search` (o carregamento preguiçoso que o `service` já tem).

Aresta nova no grafo: nenhuma. `cmd/gobsidian` já importa `boot`, `mcpsrv`, `service`, `config`, `instalar`.

## `GOBSIDIAN_VAULT`

Padrão de `--vault` em **todo comando de CLI**: os 14 acima, `doctor`, `install`, `vaults`. Aceita nome ou caminho, pela regra da Parte J. `--vault` na linha de comando vence.

**`serve` e `daemon` a ignoram.** Um host MCP herda o ambiente do usuário; se `serve` lesse a variável, um `--vault` esquecido no config do host serviria em silêncio o cofre da variável — e dois hosts sem `--vault` serviriam o mesmo cofre por acidente. `serve` sem `--vault` continua sendo erro. Um teste prova as duas metades: CLI com a variável e sem flag abre o cofre da variável; `serve` com a variável e sem flag falha.

## Saída

- **Terminal:** formatador legível, com a mesma paleta e marcadores do resto da CLI (`console`).
- **Pipe ou arquivo:** o JSON da tool, uma linha, sem nada antes nem depois — é o que um script faz `| jq`.
- `--json` e `--texto` forçam um dos dois. Os dois juntos é erro de uso.
- Logs continuam em stderr e só acima de Warn sem `--log-level`, como hoje.

**Formatadores.** Um por tool, testado contra o mesmo resultado que o JSON: `note read` imprime o conteúdo; `note list`, `vault search`, `vault broken-links`, `tag list` imprimem tabela ou lista com o `total` e um aviso quando `truncated`; `note metadata`, `note outline`, `vault stats` imprimem campos; `graph` imprime os nós por distância e as arestas; as de escrita imprimem o marcador de estado e o resumo (com o diff no `--dry-run`). Tool sem formatador cai no JSON indentado — e um teste reprova se alguma das 14 não tiver o seu.

## Erros e códigos de saída

| Situação | Saída | Código |
|---|---|---|
| Sucesso | resultado | 0 |
| Erro da tool (`NOTE_NOT_FOUND`, `HASH_MISMATCH`...) | terminal: `❌ <mensagem>` + código; JSON: `{"error":{"code":...,"message":...}}` em stdout | 1 |
| Uso errado (flag inválida, `--json` e `--texto` juntos, JSON inválido em `--args`) | mensagem em stderr | 2 |
| Cofre não resolve (Parte J) | mensagem em stderr | 2 |

Escrita respeita `--read-only` e `GOBSIDIAN_READ_ONLY` como o `serve`: as tools de escrita não existem nesse modo, e o comando diz isso em vez de "comando desconhecido".

## Concorrência com o daemon

Uma escrita pela CLI com o daemon servindo o mesmo cofre é uma escrita externa, igual a editar no Obsidian: o watcher do daemon a percebe, e `expected_hash` protege quem quer exclusão. A CLI grava o cache de índice como o `search` já grava hoje; o `doctor` a conta como processo em processo (`G8`).

## O que prova que está pronto

- **Gate de cobertura:** teste que reprova se uma tool de `EsquemasDeEntrada()` não tem comando na tabela, se um comando da tabela não é tool, ou se um parâmetro do schema não virou flag. Tool nova sem comando quebra o gate — é a invariante que vale um gate.
- **Paridade:** para cada tool, um teste roda o comando com `--json` contra um cofre de fixture e compara com o `structuredContent` da mesma chamada feita pelo transporte em memória. Iguais como JSON (mesma estrutura e valores).
- **`GOBSIDIAN_VAULT`:** os dois testes da seção dela.
- **Códigos de saída:** um teste por linha da tabela.
- **Formatadores:** um teste por tool, com modo de saída fixado (a lição de 2026-09-16: teste que afirma desenho fixa o modo).
- Prova de mutação por regra, colada no ledger. `verify.ps1` verde.

## Fora do escopo

Modo interativo/REPL; saída em YAML ou CSV; falar com o daemon pelo socket (abordagem B, recusada: exigiria subir daemon pela CLI); `completion/complete` do MCP.

## Documentação que muda

`README.md` e `README.pt-BR.md` (referência da CLI), `docs/OPERACAO.md` (`GOBSIDIAN_VAULT`), `docs/ESTRUTURA.md` (arquivos novos), `docs/TOOLS.md` (uma linha: cada tool tem um comando), `docs/TEXTOS.md` (as telas novas).
