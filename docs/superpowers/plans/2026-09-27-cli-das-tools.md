# CLI das tools, `vaults` e `config` — plano

**Desenho:** [`docs/superpowers/specs/2026-09-25-cli-das-tools-design.md`](../specs/2026-09-25-cli-das-tools-design.md), revisto em 2026-09-27.
**Pedido do dono (2026-09-27):** "Segue como sugerido, faça tudo na ordem sugerida até encerrar." A ordem: desenho, plano, `vaults`/`config`, depois as 14 tools. Um commit por parte, `verify.ps1` verde antes de cada um.
**Ledger:** `.superpowers/sdd/2026-09-27-cli-das-tools/progress.md`.

Executado pelo orquestrador, sem subagente: as duas partes mexem nos mesmos arquivos de `cmd/gobsidian` e na árvore de comandos inteira, e dividir entre implementadores seria conflito de edição, não paralelismo.

---

## Parte K — `vaults` lista, `config` configura

Decisão do dono: `vaults` lista os cofres; `config` configura os hosts de IA (o que `vaults` fazia).

- [x] K1 — `newConfigCmd` é o `newVaultsCmd` de hoje, renomeado, com `textos.ResumoConfig`. Os testes que exercitam o fluxo de `vaults` (`install_manter_test.go`, a lista de comandos de `install_test.go`) passam a nomear `config`.
- [x] K2 — `newVaultsCmd` novo: une `instalar.CofresDoObsidian` e `instalar.ConfiguracaoAtual` por caminho canônico (`instalar.CaminhoCanonicoDeCofre`, a mesma conta de `escolherCofres`), e mostra nome (`filepath.Base`), caminho, aberto no Obsidian, configurado em host. Terminal: tabela numa moldura; `--json` ou saída que não é terminal: lista JSON `[{"name","path","open","configured","in_obsidian"}]`. Sem cofre nenhum: `ℹ️` dizendo isso, código 0.
- [x] K3 — A montagem da lista é uma função pura (`listarCofres(doObsidian, configurados)`), testada sem tocar o registro nem os hosts da máquina — a lição de 2026-09-08, quando um teste reescreveu os hosts reais. O comando lê pelas variáveis injetáveis que `install.go` já tem (`configuracaoAtualFn`) e por uma nova para o registro.
- [x] K4 — Textos em `textos` (`ResumoConfig`, `ResumoVaultsLista`, `VaultsTitulo`, `VaultsNenhum`, `VaultsAberto`, `VaultsConfigurado`, `VaultsForaDoObsidian`) e em `docs/TEXTOS.md`.
- [x] K5 — Docs: README (en e pt-BR), `docs/OPERACAO.md` onde cita `vaults`, `docs/ESTRUTURA.md`.
- [x] K6 — Mutação: a união por caminho canônico (um cofre configurado com outra grafia vira duas linhas), o cofre configurado fora do Obsidian (some da lista), a saída JSON fora do terminal.
- [x] K7 — `verify.ps1`, ledger, commit.

## Parte L — as 14 tools pela CLI

- [ ] L1 — `mcpsrv`: `NovoParaCLI` (tools sem resources), `EsquemasDeEntrada()` e `ChamarLocal`. `EsquemaDeTool`/`Parametro` em tipos do domínio: nome, descrição, tipo (`string`, `integer`, `boolean`, `array`, `object`), tipo do item de array, enum, obrigatório, e se o item aceita objeto (`oneOf`). `ChamarLocal` devolve `Resultado{JSON json.RawMessage}` ou `*ErroDeTool{Codigo, Mensagem}`; o código sai do prefixo `CODIGO: ` que `toolErr` já escreve.
- [ ] L2 — `boot.AbrirServicoDeCLI(ctx, cfg, log) (*service.Service, func(), error)`: cofre, `AbrirIndice`, `search.NewInverted` marcado em construção e `CarregarBusca` preguiçoso, como `Montar` faz sem `EagerSearch`. O `func()` fecha o índice invertido.
- [ ] L3 — `cmd/gobsidian/ferramentas.go`: a tabela tool → caminho de comando → posicionais; grupos `note` e `tag`; `stats`, `search`, `broken-links`, `graph` na raiz. Cada comando é montado do schema: flags (`ferramentas_flags.go`), `--args`, `--json`, `--texto`, as flags de cofre e de cache. `note read` aceita N posicionais: um vira `path`, dois ou mais viram `paths`.
- [ ] L4 — Saída: terminal → formatador; senão → JSON numa linha. Erro da tool: terminal `❌` com mensagem e código; JSON `{"error":{"code","message"}}` em stdout; código 1.
- [ ] L5 — Códigos de saída: `erroComCodigo` com o código; `main` sai com ele. Uso errado (flag do cobra, posicional, `--json` com `--texto`, `--args` inválido) e cofre que não resolve saem 2. Tool de escrita com `--read-only`/`GOBSIDIAN_READ_ONLY`: 2, dizendo que a tool não existe nesse modo.
- [ ] L6 — `GOBSIDIAN_VAULT`: `flagsDeCofre` ganha o padrão da variável para os comandos de CLI; `serve` e `daemon` registram sem ele. Teste das duas metades.
- [ ] L7 — `ferramentas_texto.go`: um formatador por tool, sobre os tipos de `service` decodificados do JSON.
- [ ] L8 — Saem `index.go`, `inspect.go` e o `search.go` antigo, com os testes que só os exercitavam (`cli_subcommands_test.go`, `index_origem_test.go`, `search_cache_test.go`); o que eles provavam e ainda vale (o cache gravado pela CLI, a origem do índice) é reescrito sobre o comando novo quando ainda for comportamento do produto.
- [ ] L9 — Testes: gate de cobertura (tool ↔ comando ↔ flag), paridade por tool (comando `--json` × chamada em memória), `GOBSIDIAN_VAULT` nas duas metades, um por linha da tabela de códigos, formatador por tool com modo fixado e um teste que reprova tool sem formatador.
- [ ] L10 — Textos em `textos` e `docs/TEXTOS.md`; README (en e pt-BR), `docs/OPERACAO.md`, `docs/ESTRUTURA.md`, `docs/TOOLS.md`.
- [ ] L11 — Mutação por regra (flag omitida não viaja; flag vence `--args`; `note read` com N posicionais vira `paths`; JSON fora do terminal; erro de tool sai 1 e uso sai 2; escrita recusada em somente leitura; `serve` ignora `GOBSIDIAN_VAULT`; gate de cobertura acusa parâmetro sem flag).
- [ ] L12 — `verify.ps1`, ledger, commit.

## Riscos

| Risco | Mitigação |
|---|---|
| O schema que o SDK devolve no `ListTools` do cliente muda de forma entre versões | `EsquemasDeEntrada` decodifica por JSON num tipo próprio de `mcpsrv`, e o gate de cobertura quebra se um parâmetro deixar de virar flag |
| Nome de flag da tool colidir com flag de cofre (`--limit` não colide; `--offset`, `--folder` também não) | O gate de cobertura recusa colisão com as flags de cofre, cache e saída |
| `search` mudar de flags quebra script de quem já usava | `search "x" --limit N --json` continua funcionando; o que muda é a forma do JSON, que passa a ser a da tool — registrado no README |
