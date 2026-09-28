# CLI das tools — desenho

**Data:** 2026-09-25, revisto em 2026-09-27. **Pedido do dono:** "todas as tools tenham um comando para uso por CLI" e "uma variável de ambiente para setar o vault padrão, fixado apenas para o CLI". Decisões tomadas na conversa: uso **pelos dois** (humano no terminal e agente/script pelo shell); variável **`GOBSIDIAN_VAULT`**; forma **agrupada** (`gobsidian note read ...`); abordagem **A** (a CLI chama a tool de verdade, em processo); `search`, `index` e `inspect` **mudam de lugar e passam a funcionar igual aos demais**.

**Revisão de 2026-09-27, decisão do dono:** "o comando `vaults` deve ser para listar os vaults; o comando `config` pode ficar para configurar MCPs". Com isso, o grupo `vault` do desenho original (`vault stats`, `vault search`, `vault broken-links`) sai: ele ficaria a uma letra de `vaults`, e um `s` separaria listar cofres de buscar num cofre. As três tools vão para a raiz. `search` continua onde sempre esteve.

## O que o usuário ganha

Cada uma das 14 tools do servidor MCP vira um comando. O resultado é **o mesmo** que o host recebe — mesmos padrões, mesmos tetos, `truncated`, códigos de erro —, porque a CLI chama a tool, não uma cópia dela.

```
gobsidian note read "Civil/Dolo.md" --heading Conceito
gobsidian note list --folder Civil --tags penal
gobsidian note outline|metadata <nota>
gobsidian note create|append|patch|move|delete ...
gobsidian stats | search "dolo" | broken-links
gobsidian tag list --prefix direito
gobsidian graph <nota> --depth 2
gobsidian vaults                     # lista os cofres
gobsidian config                     # registra o servidor nos hosts de IA
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
| `vault_stats` | `stats` | — |
| `vault_search` | `search` | `query` |
| `vault_broken_links` | `broken-links` | — |
| `tag_list` | `tag list` | — |
| `link_graph` | `graph` | `path` |

`inspect` → `note metadata` e `index` → `stats`. Os dois antigos **saem**: dois nomes para o mesmo comando é a duplicação que o projeto persegue, e o dono autorizou mover. `index` existia também para aquecer o cache; isso continua acontecendo, porque todo comando abre o índice pela mesma conta (`boot.AbrirIndice`), que grava o cache. `search` fica com o mesmo nome e passa a ser a tool `vault_search`: as flags dele mudam para as da tool (`--limit` continua; `--json` continua; `--max-results` é flag de cofre).

## `vaults` e `config`

- **`config`** é o que `vaults` fazia até 2026-09-27: pergunta cofres e hosts e registra o **gobsidian** nos hosts de IA, sem reinstalar o binário. Mesmas flags, mesmo fluxo.
- **`vaults`** lista os cofres: cada cofre que o Obsidian conhece (`instalar.CofresDoObsidian`) e cada cofre já configurado num host (`instalar.ConfiguracaoAtual`), com nome, caminho, se está aberto no Obsidian e se está configurado. Não pede `--vault` e não abre cofre nenhum. Com `--json` ou pipe, sai uma lista JSON; no terminal, uma tabela.

## Arquitetura

**`mcpsrv` ganha duas funções, e continua sendo o único pacote com tipos do SDK:**

- `EsquemasDeEntrada() []EsquemaDeTool` — nome, descrição e o schema de entrada de cada tool, em tipos do domínio (nome do parâmetro, tipo, descrição, obrigatório, enum). Monta um servidor sem serviço só para ler o que foi registrado; nenhuma tool é chamada. É o que a árvore de comandos usa para `--help` e completação sem abrir o cofre.
- `ChamarLocal(ctx, svc, cfg, nome, argumentos map[string]any) (Resultado, error)` — monta o servidor sobre o serviço, conecta um cliente por transporte em memória (o mesmo que os testes de `mcpsrv` usam), chama a tool e devolve o `structuredContent` em JSON cru, ou o erro com o código de domínio. O servidor da CLI **não publica resources**: publicar exige percorrer o cofre inteiro, e nenhum comando os lê.

**`boot` ganha `AbrirServicoDeCLI`**: cofre, índice de metadados (pelo cache) e o serviço com a busca preguiçosa — a mesma montagem de `Montar`, sem watcher. O índice de busca só carrega quando a tool é `vault_search`, pelo `CarregarBusca` que o serviço já tem.

**`cmd/gobsidian` ganha:**

- `ferramentas.go` — a tabela da seção anterior e a construção dos comandos a partir dela e de `EsquemasDeEntrada()`.
- `ferramentas_flags.go` — schema → flags: `string` → texto, `integer` → inteiro, `boolean` → booleano, `array` de `string` → repetível (`--tags a --tags b` ou `--tags a,b`), `object` (o `frontmatter`) → `--frontmatter chave=valor` repetível. `oneOf` (o item de `paths` de `note_read`, string ou objeto) → os posicionais são as strings; a forma objeto só pela válvula `--args`. O nome da flag é o do parâmetro com `-` no lugar de `_`. Só a flag **passada** entra na chamada: flag omitida deixa a tool aplicar o padrão dela.
- `--args '<json>'` em todo comando: um objeto JSON mesclado **por baixo** das flags (flag vence). É a saída para o que uma flag não expressa, e o jeito de um script passar a entrada inteira.
- `--content -` lê o conteúdo do stdin, nas tools que escrevem: conteúdo de nota raramente cabe numa linha de comando.
- `ferramentas_texto.go` — um formatador legível por tool.

**Fluxo de uma chamada:** `carregarConfig` (Parte J: nome ou caminho) → `boot.AbrirServicoDeCLI` → `mcpsrv.ChamarLocal` → formatador.

Aresta nova no grafo: nenhuma. `cmd/gobsidian` já importa `boot`, `mcpsrv`, `service`, `config`, `instalar`; `boot` já importa `search` e `service`.

## `GOBSIDIAN_VAULT`

Padrão de `--vault` em **todo comando de CLI**: os 14 acima, `doctor`, `install`, `config`. Aceita nome ou caminho, pela regra da Parte J. `--vault` na linha de comando vence.

**`serve` e `daemon` a ignoram.** Um host MCP herda o ambiente do usuário; se `serve` lesse a variável, um `--vault` esquecido no config do host serviria em silêncio o cofre da variável — e dois hosts sem `--vault` serviriam o mesmo cofre por acidente. `serve` sem `--vault` continua sendo erro. Um teste prova as duas metades: CLI com a variável e sem flag abre o cofre da variável; `serve` com a variável e sem flag falha.

## Saída

- **Terminal:** formatador legível, com a mesma paleta e marcadores do resto da CLI (`console`).
- **Pipe ou arquivo:** o JSON da tool, uma linha, sem nada antes nem depois — é o que um script faz `| jq`.
- `--json` e `--texto` forçam um dos dois. Os dois juntos é erro de uso.
- Logs continuam em stderr e só acima de Warn sem `--log-level`, como hoje.

**Formatadores.** Um por tool: `note read` imprime o conteúdo; `note list`, `search`, `broken-links`, `tag list` imprimem lista com o `total` e um aviso quando `truncated`; `note metadata`, `note outline`, `stats` imprimem campos; `graph` imprime os nós e as arestas; as de escrita imprimem o marcador de estado e o resumo (com o diff no `--dry-run`). Tool sem formatador cai no JSON indentado — e um teste reprova se alguma das 14 não tiver o seu.

## Erros e códigos de saída

| Situação | Saída | Código |
|---|---|---|
| Sucesso | resultado | 0 |
| Erro da tool (`NOTE_NOT_FOUND`, `HASH_MISMATCH`...) | terminal: `❌ <mensagem>` com o código; JSON: `{"error":{"code":...,"message":...}}` em stdout | 1 |
| Uso errado (flag inválida, `--json` e `--texto` juntos, JSON inválido em `--args`, posicional faltando) | mensagem em stderr | 2 |
| Cofre não resolve (Parte J) | mensagem em stderr | 2 |
| Tool de escrita em modo somente leitura | mensagem em stderr dizendo isso | 2 |

Escrita respeita `--read-only` e `GOBSIDIAN_READ_ONLY` como o `serve`: as tools de escrita não existem nesse modo, e o comando diz isso em vez de "comando desconhecido".

## Concorrência com o daemon

Uma escrita pela CLI com o daemon servindo o mesmo cofre é uma escrita externa, igual a editar no Obsidian: o watcher do daemon a percebe, e `expected_hash` protege quem quer exclusão. A CLI grava o cache de índice como o `search` já grava hoje.

## O que prova que está pronto

- **Gate de cobertura:** teste que reprova se uma tool de `EsquemasDeEntrada()` não tem comando na tabela, se um comando da tabela não é tool, ou se um parâmetro do schema não virou flag nem posicional. Tool nova sem comando quebra o gate — é a invariante que vale um gate.
- **Paridade:** para cada tool, um teste roda o comando com `--json` contra um cofre de fixture e compara com o `structuredContent` da mesma chamada feita pelo transporte em memória. Iguais como JSON.
- **`GOBSIDIAN_VAULT`:** os dois testes da seção dela.
- **Códigos de saída:** um teste por linha da tabela.
- **Formatadores:** um teste por tool, com modo de saída fixado (a lição de 2026-09-16: teste que afirma desenho fixa o modo).
- **`vaults` e `config`:** `vaults` lista os cofres do registro e os configurados fora dele; `config` faz o que `vaults` fazia, com os testes de `vaults` migrados.
- Prova de mutação por regra, colada no ledger. `verify.ps1` verde.

## Fora do escopo

Modo interativo/REPL; saída em YAML ou CSV; falar com o daemon pelo socket (abordagem B, recusada: exigiria subir daemon pela CLI); `completion/complete` do MCP.

## Documentação que muda

`README.md` e `README.pt-BR.md` (referência da CLI), `docs/OPERACAO.md` (`GOBSIDIAN_VAULT`), `docs/ESTRUTURA.md` (arquivos novos), `docs/TOOLS.md` (uma linha: cada tool tem um comando), `docs/TEXTOS.md` (as telas novas).
