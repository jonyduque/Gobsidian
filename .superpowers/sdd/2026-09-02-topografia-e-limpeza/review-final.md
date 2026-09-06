# Final whole-branch review — 6c5d1f1..5c613d5

## Progresso

- 15:30 — brief lido; arquivo criado
- 15:30 — final-log-stat lido (98 commits, 185 arquivos)
- 15:31 — `go build ./...` e `go vet ./...` exit 0
- 15:31 — `go list` imports coletados; grafo bate com CLAUDE.md linha a linha
- 15:32 — `go test -race -count=1 ./...` verde (todos os pacotes ok)
- 15:32 — verify.ps1 -SkipCross -SkipNet: `[OK] Bateria completa`, exit 0
- 15:33 — revisor.md e final-parked-items lidos
- 15:34 — diff de producao lido inteiro (5 858 linhas, 68 arquivos)
- 15:38 — TOOLS.md, README.md, ARCHITECTURE.md, ARMADILHAS.md, segregacao.md, verify.ps1 lidos
- 15:40 — checagens no fonte: schema servido, tag_list, travas do daemon, caches
- 15:42 — diff de testes lido (9 069 linhas, 104 arquivos)
- 15:44 — reproducao do achado F1 com o binario (`go run ... doctor --debounce-ms 0`)
- 15:45 — varredura de referencias obsoletas em `internal/`, `cmd/`, `docs/`
- 15:47 — achados redigidos; revisao fechada

---

## Verification

Todos os comandos rodaram em primeiro plano, na raiz do repositorio, em
`GOOS=windows`.

```
$ go build ./...            → exit 0, saida vazia
$ go vet ./...              → exit 0, saida vazia
```

```
$ go test -race -count=1 ./...
ok  	github.com/jonyd/gobsidian/cmd/gobsidian	8.912s
ok  	github.com/jonyd/gobsidian/internal/boot	2.435s
ok  	github.com/jonyd/gobsidian/internal/config	1.676s
ok  	github.com/jonyd/gobsidian/internal/console	1.701s
ok  	github.com/jonyd/gobsidian/internal/daemon	6.369s
ok  	github.com/jonyd/gobsidian/internal/doctor	36.744s
ok  	github.com/jonyd/gobsidian/internal/index	13.237s
ok  	github.com/jonyd/gobsidian/internal/ipc	1.744s
ok  	github.com/jonyd/gobsidian/internal/lifecycle	3.006s
ok  	github.com/jonyd/gobsidian/internal/mcpsrv	22.099s
ok  	github.com/jonyd/gobsidian/internal/parser	2.048s
ok  	github.com/jonyd/gobsidian/internal/search	28.499s
ok  	github.com/jonyd/gobsidian/internal/service	59.165s
?   	github.com/jonyd/gobsidian/internal/text	[no test files]
ok  	github.com/jonyd/gobsidian/internal/vault	29.924s
ok  	github.com/jonyd/gobsidian/internal/vaulttest	1.662s
ok  	github.com/jonyd/gobsidian/internal/watcher	24.362s
ok  	github.com/jonyd/gobsidian/internal/writer	4.898s
ok  	github.com/jonyd/gobsidian/tools/netcheck	23.227s
?   	github.com/jonyd/gobsidian/tools/netcheck/cmd/netcheck	[no test files]
```

```
$ pwsh -File scripts/verify.ps1 -SkipCross -SkipNet
[...] 1. go build
[OK] go build
[...] 2. go test -race
[OK] go test -race
[...] 3. contagem de testes pulados
[!] 6 testes pulados
     --- SKIP: TestAjudanteSeguraTrava (0.00s)
     --- SKIP: TestListenRestringePermissaoUnix (0.00s)
     --- SKIP: TestSignalCancelsContext (0.10s)
     --- SKIP: TestPerfilDeHeapServindo (0.00s)
     --- SKIP: TestWriteAtomicPreservaOModoDoAlvo (0.00s)
     --- SKIP: TestNew_FailsOnUnwatchablePath (0.00s)
[...] 4. go test (tetos de latencia, sem -race)
[OK] go test (tetos de latencia, sem -race)
[...] 5. go vet (windows)
[OK] go vet (windows)
[i] vet cruzado pulado (-SkipCross)
[...] 6. gofmt
[OK] gofmt
[...] 7. golangci-lint
[OK] golangci-lint
[...] 8. golangci-lint (linux)
[OK] golangci-lint (linux)
[i] check_net pulado (-SkipNet)
[...] 9. check_tool_params
[OK] check_tool_params
[...] 10. check_doc_refs
[OK] check_doc_refs
[...] 11. check_readme_anchors
[OK] check_readme_anchors

[OK] Bateria completa. Pode commitar.
VERIFY_EXIT=0
```

Os seis pulados sao os legitimos de plataforma/ambiente; nenhum deles e o de
paridade. `verify.ps1` tem 14 etapas (13 `Invoke-Step` + a contagem manual de
pulados na linha 129) — a afirmacao do CLAUDE.md confere; a corrida acima
mostra 11 numeradas porque `-SkipCross` colapsa `go vet (linux)`+`(darwin)` e
`-SkipNet` retira `check_net`.

Reproducao do achado F1 (unico comando de produto que rodei):

```
$ go run ./cmd/gobsidian doctor --vault "<repo>/testdata" --debounce-ms 0
[!] --debounce-ms: valor invalido 0 (use um inteiro >= 1)
exit status 1
$ go run ./cmd/gobsidian doctor --vault "<repo>/testdata" --debounce-ms 500
[OK] raiz do cofre existe
[OK] permissao de leitura
     4 entradas na raiz
```

---

## Import graph

`go list -f '{{.ImportPath}} {{.Imports}}' ./internal/... ./cmd/...`, filtrado
para as arestas internas. **Bate com o bloco do CLAUDE.md linha a linha, sem
divergencia nenhuma.**

| Pacote | `go list` (arestas internas) | CLAUDE.md | Confere |
|---|---|---|---|
| `text` | — | folha | sim |
| `vault` | — | folha | sim |
| `config` | — | folha | sim |
| `console` | — | folha | sim |
| `lifecycle` | — | folha | sim |
| `parser` | text | text | sim |
| `ipc` | config | config | sim |
| `writer` | parser, text, vault | parser, text, vault | sim |
| `index` | parser, text, vault | parser, text, vault | sim |
| `search` | index, parser, text, vault | index, parser, text, vault | sim |
| `watcher` | index, search, vault | index, search, vault | sim |
| `service` | index, parser, search, vault, writer | idem | sim |
| `mcpsrv` | config, index, parser, service, vault | idem | sim |
| `boot` | config, index, lifecycle, search, service, vault, watcher | idem | sim |
| `daemon` | config, ipc, mcpsrv | config, ipc, mcpsrv | sim |
| `doctor` | config, daemon, ipc, vault | config, daemon, ipc, vault | sim |
| `vaulttest` | vault | vault (so-teste) | sim |
| `cmd/gobsidian` | boot, config, console, daemon, doctor, ipc, lifecycle, mcpsrv, search, service, vault | (nao tabelado) | — |

Conferido tambem o que o grafo promete e que `go list` sozinho nao mostra:

- Nenhum pacote de dominio importa `boot` — so `cmd/gobsidian`.
- `boot` nao importa `mcpsrv`. Confere.
- `lifecycle` continua folha depois de `boot → lifecycle`. Confere.
- `text` continua folha depois de ganhar `ChaveDeCaminho` (`path/filepath` e stdlib). Confere.
- `net` fora de `ipc`/`daemon`: `daemon` importa `net` (o `net.Listener` do `ipc.Listen`), `ipc` importa `net`. Nenhum outro. `check_net` foi pulado nesta corrida por `-SkipNet`, mas `go list` confirma a superficie.
- Tipos do SDK MCP: `github.com/modelcontextprotocol/go-sdk/mcp` aparece so em `mcpsrv` na producao; `daemon` o alcanca por `mcpsrv.Server`, nao pelo SDK. Confere.

---

## Findings

### Critical

Nenhum.

### Important

**F1** `cmd/gobsidian/doctor.go:69` — `doctor` declara `--debounce-ms` e nada
em `internal/doctor` le `cfg.DebounceMS`.

Mecanismo, fechado por leitura: `newDoctorCmd` registra a flag (`:69`) e
preenche `flags.DebounceMSSet` (`:24`); `config.Load` valida o valor e recusa
`0` quando a flag foi dada; `doctor.Run` → `platformChecks`/`checkDaemon*`
nunca consultam `DebounceMS`. `git grep -n "DebounceMS" internal/doctor` volta
vazio. O `doctor` nao tem watcher — nao ha o que coalescer.

Falha concreta, reproduzida acima: `gobsidian doctor --debounce-ms 0` **sai com
codigo 1** por causa de um parametro que nao tem efeito nenhum sobre o
diagnostico; e `--debounce-ms 500` e aceito em silencio prometendo uma janela
que nao existe. E exatamente a classe que a Task 176 removeu de `index` e
`inspect` ("Schema que promete e codigo que ignora e pior que parametro
ausente"), deixada de pe no terceiro subcomando. Pior: o `README.md:206` desta
mesma branch passa a **documentar** a flag como parte do contrato de `doctor`,
e o teste de regressao `TestIndexEInspectNaoAceitamFlagsQueIgnoram`
(`cmd/gobsidian/cli_subcommands_test.go:190`) cobre `index`, `inspect` e
`search` — nao `doctor`.

Conserta: apagar a linha `:69` (e a linha `:24`, que fica sem alvo), tirar
`doctor` da coluna Subcommands de `--debounce-ms` no `README.md:206`, e
acrescentar `{"doctor", newDoctorCmd}` ao laco do teste com a flag
`debounce-ms`. CONFIRMADO.

**F2** `internal/mcpsrv/tools_read.go:327` e `:363` — a mudanca de semantica de
`tags` (Task 180, `18d9da4`, marcado `!` como quebra de compatibilidade) foi
documentada no `TOOLS.md` mas nao chegou a tag `jsonschema`, que e o que o host
recebe.

Mecanismo: `18d9da4` fez `tags` casar hierarquicamente e dobrar caixa/NFC nas
duas tools. O `TOOLS.md:818` passou a mostrar, no bloco de `vault_search`,
`"description": "Notas que contenham TODAS as tags. A tag pedida casa a si
mesma e suas subtags; '#' inicial é opcional; comparação insensível a caixa e a
forma Unicode (NFC)."`, e o `TOOLS.md:911` mostra a mesma prosa para
`note_list.tags`. No codigo, `vaultSearchInput.Tags` (`:327`) continua com
`jsonschema:"Notas que contenham TODAS as tags."` e `noteListInput.Tags`
(`:363`) **nao tem tag `jsonschema` nenhuma**.

Por que isso e um contrato que mente, e nao so um comentario velho: o proprio
`TOOLS.md:802`, acrescentado nesta branch, declara que "o schema que o host
efetivamente recebe traz só `type` e `description`" — ou seja, `description` e
a unica prosa que atravessa. Os blocos JSON abaixo dela exibem, para estes dois
campos, um `description` que o servidor nao envia. O modelo do outro lado le o
schema para decidir o que pedir: ele nao tem como saber que `#projeto` passou a
casar `projeto/alpha`, nem que `#Ação` e `#ação` sao a mesma tag — e essa e
justamente a mudanca de comportamento que a Task 180 introduziu.

Que isto e lapso e nao decisao: a mesma leva atualizou a tag de
`tagListInput.Prefix` no codigo (`aa8ec0b`, `:378`) e criou
`internal/mcpsrv/metadata_include_schema_test.go`, que amarra a tag
`jsonschema` de `Include` a `service.CamposDeMetadata` justamente para as duas
listas nao divergirem. Nada equivalente existe para `tags`, e
`scripts/check_tool_params.ps1` nao pega: ele confere que o parametro e **lido**
pelo handler, nunca o texto da descricao (ver o proprio cabecalho do script).

Conserta: por a prosa de `TOOLS.md:818`/`:911` nas duas tags `jsonschema`
(`tools_read.go:327` e `:363`); a coerencia das duas com a semantica de
`index.ChaveDeTag` nao tem teste barato, mas a divergencia texto-a-texto entre
`TOOLS.md` e a tag pode ser fechada do mesmo jeito que a de `Include`.
CONFIRMADO.

**F3** `docs/segregacao.md:101-112` e `:134` — o candidato 3 ("Chave derivada —
a tag escapa da conta única") descreve, como pendente, exatamente o trabalho que
a Task 180 concluiu.

Mecanismo: o texto diz que "`index/query.go` tem 10 `strings.ToLower` soltos …
parte é comparação de tag … que baixa caixa sem normalizar Unicode", e fecha
(`:134`) com "o 3 porque falta a medição que diria se vale". Depois de
`18d9da4`, a comparacao de tag passa por `index.ChaveDeTag` (NFC + caixa) em
`query.go`, `index.go`, `update.go` e `service/graph.go`, e sobraram **tres**
`strings.ToLower` em `query.go` (`:263`, `:424`, `:428`) — os tres sao
parametro de consulta (`mode`, `order`, `sort`), que o proprio documento
classifica como "o que está certo".

Falha concreta: o `CLAUDE.md` indexa este arquivo como "os candidatos a
extração, medidos, com decisão pendente". A proxima sessao que o abrir vai
replanejar — e possivelmente remedir — trabalho ja entregue. O candidato 2, no
mesmo arquivo e na mesma leva, **recebeu** a nota "Extraído nas Tasks 171 e 172"
(`:98-99`); o 3 nao recebeu nada.

Conserta: uma nota analoga sob o candidato 3 dizendo que a Task 180 fechou a
conta com `index.ChaveDeTag`, e o ajuste da frase de `:134`. O `:87` do mesmo
arquivo ainda nomeia `writer/atomic.go:141`, apagado nesta branch (ver Stale
references). CONFIRMADO.

### Nits

**N1** `internal/daemon/lock.go:121` — `EhArquivoDeTrava` promete no doc comment
"diz se um nome de arquivo no diretorio de runtime e **uma das duas travas de
daemon**", mas o corpo e `strings.HasSuffix(nome, ".lock")`: qualquer arquivo
`.lock` no diretorio de runtime passa, e `sufixoTravaDeEscuta` (`:116`) fica
declarada ao lado sem participar do predicado, o que sugere ao leitor uma conta
que nao esta ali. Sem consequencia hoje — `ipc.runtimeDir` devolve
`%LocalAppData%/gobsidian/run`, so do produto — e `TestEhArquivoDeTravaCobreAsDuasTravas`
ja fixa `"abc.lock.txt" → false`. Ou o comentario passa a dizer "qualquer
`.lock`", ou o predicado compara os dois sufixos.

**N2** `internal/mcpsrv/tools_read.go:296` — `tag_list.min_count` ainda aplica o
padrao (`minCount := 1`) na borda MCP, contra a regra que a Task 168 estabeleceu
e aplicou a `vault_search`, `note_list` e `link_graph` ("quem aplica o padrao e
o service, que e tambem o caminho da CLI"). Sem diferenca observavel hoje: toda
chave de `ix.tags` tem pelo menos um caminho, entao `minCount` 0 e 1 devolvem o
mesmo conjunto nos dois ramos. Fica como inconsistencia da regra, nao como
defeito.

**N3** `internal/search/persist_test.go:5091` (`TestQ3PerformanceMeasurement`) —
a assercao nova `loadDur >= rebuildDur` e uma comparacao de relogio de parede
dentro da suite que roda com `-race`. O comentario defende a escolha (relacao,
nao teto) e a margem medida e de ~4x (26,96 ms contra 106,58 ms), mas e o unico
teste da branch cujo veredito depende de escalonamento. Se aparecer
intermitencia, o suspeito e este.

**N4** `internal/search/persist_codec.go:54` — `cacheMagic` deixou de ser
`const` e virou `var cacheMagic = fmt.Sprintf("GBS%d", CacheFormatVersion)`.
A conta unica e o ganho, e `TestVersaoDoCacheDeBuscaEUmaConta` a prende; o custo
e que um identificador de formato passou a ser gravavel em tempo de execucao.
Uma alternativa sem esse custo seria manter o `const` e deixar o teste comparar
com o `Sprintf`.

**N5** `internal/boot/montar.go:85-87` — o comentario afirma "as duas metades do
boot continuam antes de o servidor servir, e o join abaixo acontece antes de
`watcher.New` — nada que escreva chegou a existir ainda", mas `AbrirIndice`
(chamada entre o `go SweepStaleTempFiles` e o join) grava o cache de metadados
por `vault.ReplaceFile`, que cria um `.gobsidian-tmp-*` no `CacheDir`. Inocuo na
pratica — o `CacheDir` padrao fica **fora** do cofre, e a varredura so alcanca
`cfg.VaultPath` — e a ordem e anterior a esta branch (era a mesma em
`cmd/gobsidian/servico.go`). Vale a ressalva porque nada impede
`--cache-dir <dentro do cofre>`.

---

## Stale references

Documentacao e comentarios que nomeiam arquivo ou funcao que esta branch
renomeou ou apagou. Os itens ja listados em `final-parked-items.txt` e no bloco
"Known / already-parked" do brief **nao** estao repetidos aqui; o que segue e a
mesma classe, encontrada a mais.

**Codigo de producao**

| Arquivo:linha | Diz | Deveria dizer |
|---|---|---|
| `internal/index/persist.go:65` | "ver o comentário de `invertedCacheState` em `cmd/gobsidian/serve.go`" | `estadoDoCache`, em `internal/boot/busca.go` |
| `internal/search/inverted.go:86` | "o laço de `buildInvertedIndex`" | `construirBusca`, em `internal/boot/busca.go` |
| `internal/search/inverted.go:601` | "o laco de boot em `buildInvertedIndex`" | idem |
| `internal/search/inverted.go:615` | "`invertedCacheState`" | `boot.estadoDoCache` |
| `internal/search/mmap.go:158-159` | "`invertedCacheState` em `cmd/gobsidian/serve.go`" e "`buildInvertedIndex`" | `boot.estadoDoCache` / `boot.construirBusca` |

`inverted.go:86` e comentario **novo**, escrito pela Task 170, que ficou obsoleto
quatro tasks depois, na 174 — vale como sinal de que a renomeacao nao varreu
comentario.

**Testes**

| Arquivo:linha | Nome morto |
|---|---|
| `internal/search/cloudonly_update_windows_test.go:88,146,171,175` | `buildInvertedIndex`, `invertedCacheState` |
| `internal/search/persist_test.go:54,99` | `buildInvertedIndex (em cmd/gobsidian/serve.go)` |
| `internal/search/update_bench_test.go:16` | `buildInvertedIndex em ...` |
| `internal/service/search_lazy_test.go:25` | `prepararIndiceDeBusca` → `boot.PrepararBusca` |

**Documentacao**

| Arquivo:linha | Diz | Deveria dizer |
|---|---|---|
| `docs/segregacao.md:87` | `writer/atomic.go:141` | `internal/vault/atomic.go` (ver F3) |
| `docs/segregacao.md:101-112,134` | candidato 3 em aberto | fechado pela Task 180 (F3) |
| `docs/SUGESTOES.md:311,317,413,437` | `writer/atomic.go:…` | `internal/vault/atomic.go` |
| `docs/SUGESTOES.md:775` | `writer.WriteAtomic` | `vault.WriteAtomic` |
| `docs/SUGESTOES.md:924-925` | M14: "consumir `vault.DetectEOL/NormalizeEOL`" | `vault.NormalizeEOL` foi apagada na Task 166; a recomendacao, como escrita, nao e mais executavel |
| `docs/ESTADO.md:199` | tabela de cobertura de 2026-09-02 nomeando `construirServico`, `carregarIndiceDoCache`, `prepararIndiceDeBusca`, `buildInvertedIndex`, `CleanStaleTempFiles` | as linhas vizinhas `:202` e `:203` trazem a nota "Depois das Tasks 171-172, … não existem mais"; a `:199` nao traz nenhuma, e `CleanStaleTempFiles` nao existe mais em lugar nenhum |
| `docs/OPERACAO.md:884,981,1657,1679,1696` | `prepararIndiceDeBusca`, `buildInvertedIndex`, `invertedCacheState` | fora do diff desta revisao (excluido do escopo), mas ficou obsoleto **por causa** desta branch |
| `docs/wiki/flows/boot.md:8,25,29,83` | `cmd/gobsidian/servico.go`, `construirServico`, `invertedCacheState` | `internal/boot` (parcialmente parked) |
| `docs/wiki/features/escrita.md:26,46` | `writer.WriteAtomic`, `CleanStaleTempFiles` | `vault.WriteAtomic`; `CleanStaleTempFiles` foi apagada |
| `docs/wiki/overview/onde-ficam-os-dados.md:92,99` | `writer.WriteAtomic`, `writer.SweepStaleTempFiles` | `vault.*` |
| `docs/wiki/concepts/camadas-e-fronteiras.md:78` | "`service.Index` é uma interface de 12 métodos" | removida na Task 173 |
| `docs/wiki/decisions/decisoes-fechadas.md:10` | `source_paths: cmd/gobsidian/servico.go` | `internal/boot/montar.go` |

`scripts/check_doc_refs.ps1` passou verde nesta corrida: ele confere caminho de
arquivo citado, e a maioria das entradas acima cita **nome de funcao**, que ele
nao segue. As que citam caminho (`writer/atomic.go`, `cmd/gobsidian/servico.go`)
estao em documentos que o script nao varre ou atras de um marcador
`check-doc-refs: ignore`.

---

## O que foi conferido e esta certo

Registrado porque a ausencia de achado aqui e informacao, nao silencio:

- **`uma conta por regra`** — as contas novas da branch (`text.ChaveDeCaminho`,
  `index.ChaveDeTag`, `service.formatarHash`, `ipc.EhDesconexaoLimpa`,
  `vault.FalhaNaRaiz`, `vault.ReplaceFile`, `daemon.CaminhoDoLog`,
  `index.PathsComTags`, `service.resultadoVazio`/`pagina`,
  `boot.VigiarHost`/`AbrirIndice`/`PrepararBusca`) foram seguidas ate cada
  chamador. Nao achei um segundo sitio derivando a mesma coisa a mao, com a
  excecao registrada em N2.
- **`index.publishNoteLocked` (dedupe de tag por ultimo elemento)** — correto:
  `Replace` chama `removeContributionsLocked` antes de republicar, e as tags de
  uma nota sao anexadas em sequencia, entao o duplicado que a guarda precisa
  pegar e sempre adjacente.
- **`candidatosPorTagLocked`** — `slices.DeleteFunc` opera sobre fatia recem
  alocada por `ordenado`, e `casam` nunca aliasa a fatia guardada em `ix.tags`
  (`append(nil, v...)` sempre copia). A saida sai ordenada nos dois modos, que e
  a pre-condicao de `slices.BinarySearch` em `matchesSearchFilters`.
- **`vault_search` sem `tag_mode`** — o `"all"` fixo em `service/search.go:238` bate
  com o schema ("Notas que contenham TODAS as tags") e com o `searchMetadataOnly`,
  cujo `index.Query` deixa `TagMode` no zero-valor, que `candidatosPorTagLocked`
  trata como `all`. Nao ha campo de API com valor fixo mentindo aqui.
- **`Search` e os padroes crus da Task 168** — `opts.Limit <= 0 → 20` acontece
  em `service.Search` antes de qualquer uso, entao `pagina(offset, 0, total)`
  nao e alcancavel. `note_list`/`link_graph` idem, por `ComTeto`/`ValidarEnum`.
- **`SaveIndexCache`/`SaveInvertedCache` sobre `vault.ReplaceFile`** — o
  `os.MkdirAll(cacheDir)` continua antes; `promoverArenaSePresente` foi movida
  para antes de `ExportForCache`, o que preserva a razao original (o rename do
  Windows falha com o alvo mapeado).
- **stdout/stderr** — nenhum `fmt.Print*` novo alcancavel de `serve`;
  `loggerDeCLI` escreve em `cmd.ErrOrStderr()`.
- **Testes** — li o diff de teste inteiro. O padrao dominante da branch e o
  oposto do defeito: assercao que passou a poder falhar
  (`TestBM25TermoEmTodasAsNotasAindaPontua`, `TestInvertedConcurrencyRace`,
  `TestReconcileCtxCanceladoParaAntesDeProcessarTudo`,
  `TestFilter_OutsideVaultIsDropped`, as tres guardas `len(data.Edges) == 0` em
  `schema_params_test.go`), com a mutacao colada no comentario e no passado.
  Nao achei teste novo que nao possa falhar.

---

## Verdict

**CHANGES_REQUIRED** — nao ha defeito de comportamento, corrida ou perda de
dado, e o gate esta verde; mas F1 e um parametro de CLI que reprova a execucao
sem ter efeito (a classe que a propria branch removeu de `index` e `inspect`) e
F2 e uma mudanca de contrato quebrante que chegou ao `TOOLS.md` e nao ao schema
que o host le — as duas cabem na leva de correcao ja prevista, junto com F3 e a
lista de referencias obsoletas.
