# SUGESTOES.md — Auditoria geral do projeto

Análise estática do projeto inteiro (bugs, falhas, contratos e performance), feita por
seis frentes paralelas de leitura por subsistema (`cmd`/`lifecycle`/`daemon`/`ipc`,
`vault`/`writer`/`config`/`console`/`doctor`, `index`/`text`, `search`,
`service`/`mcpsrv`, `parser`/`watcher`), com verificação cruzada dos achados de maior
severidade contra o fonte. **Nenhum código foi alterado.**

> **Estado em 2026-08-27: a seção de achados abaixo está DESATUALIZADA e não foi
> reescrita.** Ela ainda descreve como abertos: C1–C3, A1–A8, B1, M2–M5, M7, M14,
> M15, M17, P5, P6, P8, P10, P13, P14, B5, B6, B7, B9, B11, B14 e B17. **Todos
> foram fechados** nas Tasks 128–136, ou decididos pelo dono.
>
> Dois foram **REJEITADOS depois de verificados**, e isso importa mais que o
> fechamento: **M1** está prescrito ao contrário — pede que `note_delete` adote o
> critério de `note_move`, o que esconderia justamente as âncoras que quebram por
> causa do delete. **P11** parte de premissa falsa: o pacote `os` do Go já aplica
> o prefixo de caminho longo sozinho, e a correção foi reprovada pela prova de mutação. Os
> dois estão detalhados em `docs/OPERACAO.md`.
>
> **P1, P2 e P3 foram descongelados e implementados em 2026-08-28**, junto com a
> **Oportunidade 1** (BM25 em IDs densos), que a auditoria previa que os
> subsumiria — e subsumiu. Busca ficou 31% a 45% mais rápida e alocou 42% a 47%
> menos, com paridade de ranking verificada. Ver `docs/OPERACAO.md`.
>
> Quem quiser o estado real consulte o ledger em
> `.superpowers/sdd/2026-07-25-gobsidian-v01/progress.md`, que é a fonte, e
> `docs/ESTADO.md` para o quadro por severidade e as medições.
>
> O aviso está aqui em vez de a seção ter sido reescrita porque riscar dezenas de
> achados a mão é o tipo de edição em que um fica para trás — e um achado marcado
> como fechado sem ter sido é pior que um marcado como aberto tendo sido.
> Continuam **abertos e válidos**: 8 médios, 6 de desempenho e 8 baixos.

- Base da análise: HEAD `1cbb007` (merge Task 106).
- **Nada aqui é número medido.** Nenhum build, teste ou benchmark rodou nesta auditoria.
  Itens de performance descrevem **mecanismo e complexidade**; cada um exige baseline de
  `benchstat` antes de virar mudança, pela regra da casa. Onde a conclusão depende de um
  caminho não executado, está marcado SUSPEITA.
- Este arquivo **substitui** a versão anterior (2026-08-24): os itens dela foram
  re-verificados um a um contra o código atual e estão incorporados na seção de
  performance (P1–P14). Os itens confirmados como corrigidos desde então saíram da lista.
- Estado dos achados da revisão de 2026-08-15 conferido contra o ledger
  (`.superpowers/sdd/2026-07-25-gobsidian-v01/progress.md`) e contra o código atual.
- Complemento desta mesma auditoria (partida de várias instâncias e identidade da
  instância no protocolo): itens **M17**, **P15**, a seção **"Alternativas ao modelo
  de daemon"** e a **Fase 8** do plano.

---

## Como ler este relatório

| campo | valores |
|---|---|
| Severidade | Crítico · Alto · Médio · Baixo |
| Categoria | Bug · Falha-de-contrato · Concorrência · Performance · Higiene |
| Confiança | **CONFIRMADO** = mecanismo fechado por leitura, cadeia verificada até o chamador · **SUSPEITA** = plausível, exige teste/medição |

IDs (`C*`, `A*`, `M*`, `P*`, `B*`) são referenciados pelo plano no fim deste arquivo.

---

## Resumo executivo

- **3 críticos**, todos violações de regras não negociáveis na superfície de acesso a
  arquivo: boot do índice de busca hidrata placeholders OneDrive (C1); symlink dentro do
  cofre lê e indexa conteúdo fora dele (C2); anexos são lidos inteiros pelo índice de
  busca no watcher e na reconciliação (C3).
- **8 altos**: nota duplicada com sucesso reportado no move (A1), lock exclusivo durante
  I/O em toda atualização do watcher (A2), erro transitório que diverge os dois índices
  (A3), daemon surdo para sempre após erro de Accept (A4), socket vivo desvinculado (A5),
  primeira busca bloqueando sem prazo (A6), atalho do watcher sem consultar o índice de
  busca (A7) e campo de API sempre vazio contra promessa explícita do TOOLS.md (A8).
- **17 médios** (bugs, contratos que mentem e lacunas de contrato), **15 hipóteses de
  performance** com mecanismo identificado, **18 baixos** (higiene e latentes).
- Desde a revisão de 2026-08-15, **20+ achados foram corrigidos** (seção abaixo) — entre
  eles dois dos quatro críticos originais. Os gates de doc (`check_doc_refs`, <!-- check-doc-refs: ignore check_doc_refs check_readme_anchors -- scripts PowerShell em scripts/; o corpus deste checador e so .go -->
  `check_readme_anchors`, `check_tool_params`) e os **quatro** cenários de órfãos, <!-- check-doc-refs: ignore check_readme_anchors check_tool_params -- scripts PowerShell em scripts/; o corpus deste checador e so .go -->
  inclusive `daemon-idle`, agora rodam no CI (`ci.yml:65-71,139-153`) — aqueles buracos
  de processo estão fechados.
- **Identidade da instância (M17)**: toda instância responde o handshake com o mesmo
  `serverInfo` e sem `instructions` — para o modelo, duas instâncias são
  indistinguíveis. E a partida simultânea de várias instâncias paga uma varredura
  serial do cofre inteiro antes de anunciar qualquer tool (P15), além da tempestade de
  spawn do daemon (A4/A5).

---

## O que já foi corrigido desde a revisão de 2026-08-15 (não reabrir)

Verificado no código atual, com evidência:

| Achado antigo | Estado | Evidência |
|---|---|---|
| Escrita fora do confinamento (`..\..\x.md`) | CORRIGIDO (Task 113) | `vault.Resolve` é o portão em create/move/delete/trash (`service/write.go:106,428,437,730,741`); teste `write_traversal_test.go` |
| `expected_hash` contra hash do índice | CORRIGIDO (Task 116) | compara contra bytes lidos agora (`write.go:187-189,293-295`); `expected_hash_test.go` |
| `note_read` sem `offset`/paginação | CORRIGIDO (Task 106) | `offset`, `next_offset`, `total_size` em `read.go`; `Truncated` só no clamp real |
| `Truncated` falso-positivo em recorte exato | CORRIGIDO | comparador estrito (`read.go:254`) |
| Filtro de data descartado em silêncio | CORRIGIDO (Task 120) | `INVALID_ARGUMENT` (`tools_read.go:41-54`); `filtro_data_test.go` |
| `tag_list.sort`/`hierarchical` mortos | CORRIGIDO (Task 120) | lidos e consumidos (`tools_read.go:255-260`, `graph.go:243-251`) |
| `max_results` cadeia morta | CORRIGIDO (Task 120) | flag→config→clamp→`effective_limit`; flag registrada nos seis subcomandos | <!-- check-doc-refs: ignore max_results -- flag de CLI que define o teto; nunca foi campo de schema -->
| `FrontmatterErr` nunca lido | CORRIGIDO (Tasks 118/121) | exposto em metadata e contado em stats (`graph.go:491,595-597`); skips registrados |
| `Heading.Slug` pré-computado e nunca lido | CORRIGIDO (Task 118) | lido em `anchors.go:31`, `read.go:193`, `section.go:59`; guarda em `slug_persistido_test.go` |
| Placeholder `.md`: Note×Asset conforme construção | CORRIGIDO | `classeNotaSemLeitura` nos dois caminhos (`update.go:62-76`, `index.go:112`) |
| `link_graph` não determinístico, sem teto | CORRIGIDO (Task 122) | ordenação estável + `LimitEfetivo` (`graph_ordem_test.go`) |
| Campo inline apagando links | CORRIGIDO | avanço só sobre `"::"`/`"[chave::"`; guard A/B em `ext_inline_field_test.go:208-246` |
| Reconciliação reparando meio estado | CORRIGIDO | repara os DOIS índices; teste desconecta o caminho principal (`overflow_test.go`) |
| Pasta nova chegando com conteúdo invisível | SAUDÁVEL | `Add` seguido de varredura (`watcher.go:141-146`); `counters_test.go:363-424` |
| `--debounce-ms=0` aceito | CORRIGIDO | recusado na config (`config.go:177-188`) |
| Contadores de descarte agregados | SAUDÁVEL | desdobrados por motivo, atomics, publicados (`counters.go:21-43`) |
| Companheiros `*Set` esquecidos | CORRIGIDO | preenchidos nos seis subcomandos (`serve.go:34-38` etc.) |
| Doctor sai 0 com cofre inacessível | CORRIGIDO | checks halting + `ExitCode`=1 (`doctor.go:64-108`) |
| Cor decidida por `os.Stdout` global | CORRIGIDO | decisão pelo destino (`console.go:70-78,172-188`) |
| Gates de doc fora do CI; `daemon-idle` fora do CI | CORRIGIDO | `ci.yml:65-71,139-153`; também dentro do `verify.ps1:189-197` |

Dívida de processo da revisão anterior que continua válida: `scripts/measure.ps1`
(decisões de RSS: `GOGC`, `FreeOSMemory`, `maxSnippetWorkers`) continua fora de gate
nenhum — ver Plano, Fase 0.

---

## Achados

### Críticos

**C1 · Boot do índice de busca abre placeholder de nuvem — download síncrono em massa**
Bug/Falha-de-contrato · CONFIRMADO
`cmd/gobsidian/serve.go:281,325,331` — `buildInvertedIndex` chama `v.ReadAll(ctx, p)`
direto e insere com `inv.Add(...)`. A guarda de nuvem mora em `Inverted.Update`
(`internal/search/inverted.go:483-486`), e o boot **não passa por ela**. `v.ReadAll`
(`internal/vault/vault.go:163-171`) é um `os.ReadFile` puro, sem consulta a
`CloudOnly`; `idx.NotePaths()` (`index/query.go:135-145`) devolve placeholders, porque
`.md` somente-nuvem é nota do índice de metadados (`classify.go:39`). Num cofre
OneDrive com qualquer `.md` não hidratado, a reconstrução (sem cache válido, cache
corrompido ou troca de formato) dispara hidratação síncrona de cada placeholder em
segundo plano — a falha que o CLAUDE.md registra como "trava a máquina sem dizer por
quê". Agravante: o CLI usa o caminho protegido (`cmd/gobsidian/search.go:46-48` chama
`inv.Update`) — dois caminhos de construção, um com guarda, outro sem. O teste que
"prova" o boot seguro usa um dublê que chama `Update`
(`cloudonly_update_windows_test.go:132-143`, `construirComoOBoot`): produção chama
`Add`. O comentário de `inverted.go:466-469`, que lista o boot entre chamadores de
`Update`, está falso.

**C2 · Symlink de arquivo dentro do cofre lê e indexa conteúdo fora dele**
Bug/Segurança · CONFIRMADO
`internal/vault/vault.go:114-120` (`Open` usa `os.Open` puro);
`path.go:44-47` documenta o limite e delega à "camada que abre o arquivo", que não o
exerce; `walk.go:161-186` entrega symlink→arquivo à indexação. Grep repo-inteiro:
**zero usos de `os.Root`/`EvalSymlinks`**. Um `cofre/nota.md -> C:\Users\...\x.txt`
passa nas duas camadas léxicas, entra no índice, e `note_read`/`vault_search` devolvem
conteúdo arbitrário do disco pelo canal MCP. Mitigações parciais existentes: `WalkDir`
não desce symlink de diretório; `Rename` troca o link, não o alvo — o escape pleno é de
leitura + indexação. `os.Root` existe desde Go 1.24; o piso do projeto é 1.25.0.

**C3 · Anexos são lidos inteiros pelo índice de busca — no watcher e na reconciliação**
Bug/Falha-de-contrato/Performance · CONFIRMADO
`internal/search/inverted.go:483-488` — `Update` guarda só contra nuvem; anexo local
segue para `os.ReadFile(abs)` e tokenização binária. Chamadores:
`watcher/apply.go:97-99` (todo evento não-rename, inclusive `.png`/`.mp4`, que o filtro
emite normalmente — `filter.go:57-60`) e `watcher/overflow.go:68-72`. Na reconciliação
é pior: o atalho de mtime/tamanho consulta `idx.Get` (`overflow.go:58`), que resolve só
notas — **todo anexo falha no atalho e é re-lído a cada varredura de overflow**, mesmo
intocado. Impacto: mídia grande lida sincronamente na goroutine única de `Apply`;
varredura pós-tempestade lê toda a mídia do cofre; binário entra no índice inflando
`DocCount`/`docLengths` (o divisor do BM25); e o índice construído no boot (só notas,
`serve.go:281`) ≠ índice mantido por eventos — a família do defeito DocLength já pago.
Viola diretamente "anexo é indexado por nome, NUNCA lido".

### Altos

**A1 · `note_move`: cópia+remove com erro descartado e reescrita antes da mudança, sem rollback**
Bug · CONFIRMADO · `service/write.go:553-595,609-636`
`_ = os.Remove(absFrom)` (:636): remove falhando (sharing violation do Obsidian é
rotina no Windows) ⇒ sucesso reportado com a nota **duplicada** nos dois caminhos. As
notas citantes são reescritas e gravadas ANTES da movimentação (:553-595 antes de
:615-634): falhando a cópia do corpo, todos os links apontam para destino inexistente,
persistidos em disco, sem compensação — o `raw` original de cada citante estava em mãos
no loop.

**A2 · `Replace` segura lock exclusivo durante I/O de disco**
Concorrência/Performance · CONFIRMADO · `index/update.go:25-26,39,59,78`
`ix.mu.Lock()` do início ao fim, incluindo `os.Stat`, `IsCloudOnly` e a leitura inteira
do arquivo. Todo `Get/List/Tags/ResolvePath/Backlinks/TotalSize` bloqueia atrás dessa
leitura a cada evento do watcher; em OneDrive hidratando, é espera de rede disfarçada
de disco. Contraria a declaração de `index.go:17` ("leituras são concorrentes").

**A3 · Erro transitório de `Replace` diverge os dois índices**
Bug · CONFIRMADO (frequência SUSPEITA) · `update.go:34,41-48,78-81` + `apply.go:94-101`
`Replace` remove TODAS as contribuições antigas ANTES do I/O; se `ReadAll` falha, a
nota fica fora dos metadados sem republish nem reprocessamento de citantes (links dela
ficam `state=ok` para caminho que `Get` não resolve mais). O `apply.go`, vendo erro,
pula `searchInv.Update`: busca mantém o documento velho, metadados não — a divergência
que a regra proíbe; `service.Search` descarta posting sem metadado e a nota some das
respostas até o próximo evento/reconciliação/boot.

**A4 · `acceptLoop` do daemon morre para sempre no primeiro erro transitório de `Accept`**
Bug/Concorrência · CONFIRMADO · `daemon/daemon.go:145-154`
Erro sem cancelamento ⇒ `log.Warn` + `return`. O processo segue vivo, socket bound,
ticker rodando: dials conectam, ninguém aceita. A ponte pendura a sessão; o probe de
`EnsureStarted` falha por prazo e dispara o A5. Sem classificação de erro, backoff ou
contador.

**A5 · `ipc.Listen` desvincula o socket sem provar que é órfão**
Bug · CONFIRMADO · `ipc/ipc.go:104-106,309-315` + `cmd/gobsidian/daemon.go:132`
`cleanupSocketFile` incondicional antes do `net.Listen`; sem dial de prova, sem
`pidVivo`, e o subcomando `daemon` não participa do lock de `EnsureStarted`. Um segundo
daemon remove o socket do vivo e binda no nome: o primeiro fica inalcançável (memória +
watcher rodando), e duas instâncias gravam concorrentemente no MESMO cache de busca
(`serve.go:284-288`) — corrupção por escrita intercalada. É a metade do incidente dos
dois daemons que a mitigação do lock (`lock.go:79-81`) não fecha: a janela estrutural
continua (boot > prazo 10 s + probe 200 ms ⇒ chegante tardio spawna duplicado e cai
aqui).

**A6 · Primeira `vault_search` bloqueia sem prazo; `INDEX_BUILDING` inalcançável durante a carga**
Falha-de-contrato/Concorrência · CONFIRMADO · `service/search_lazy.go:32-43`,
`search.go:113-127`, `servico.go:104-107,132-156`
`cargaUnica.fazer` segura o mutex enquanto carrega; concorrentes esperam em `mu.Lock()`
puro, sem `select` em `ctx.Done()` — prazos do host ignorados. Cache frio/corrompido ⇒
tokenização completa do cofre (minutos no cofre de referência) sem resposta nem erro.
`INDEX_BUILDING` só sai pela falha da carga (`search.go:114`) ou no modo eager; o
comentário de `servico.go:104-107` descreve apenas o modo eager. O lado bom: retentativa
após falha existe e é testada (`search_lazy_test.go:79-92`).

**A7 · Atalho de mtime/tamanho do `Apply` não consulta o índice de busca**
Bug · CONFIRMADO (gatilho depende de falha prévia) · `watcher/apply.go:85-92` vs `overflow.go:58`
Um único `searchInv.Update` falho (só `log.Warn`) deixa metadados em dia e posting
ausente; todo evento seguinte com mtime/tamanho iguais cai no `continue`. OneDrive
re-emite eventos de arquivos intocados rotineiramente. A `Reconcile` já aprendeu a
condição certa (`inv.HasDoc`); o `Apply` não a espelhou.

**A8 · `Backlink.Context` é sempre `""` contra promessa explícita do contrato**
Falha-de-contrato · CONFIRMADO · `index/backlinks.go:24`, `update.go:144,476`;
exposto em `graph.go:515`; promessa em `docs/TOOLS.md:164`
Os três únicos construtores gravam literal `""`. `note_metadata.backlinks` serializa o
campo direto ao host: "campo de API com valor fixo mente sempre" — o `alias_collisions:
0` em versão silenciosa, desta vez documentado.

### Médios — bugs e contratos

**M1 · `note_delete` reporta âncoras quebradas falsas** · Falha-de-contrato · **REJEITADO em 2026-08-27: prescrito ao contrário.** Ele manda `note_delete` adotar o critério de `note_move` (`write.go:479`), que reporta só âncoras **já** ausentes. No move a nota sobrevive com os headings; no delete ela some, e toda referência ancorada quebra — inclusive as que apontam para heading existente. `TOOLS.md` diz que a tool lista o que "passará a ter" links quebrados. Aplicar isto esconderia exatamente as âncoras que quebram por causa do delete.
`write.go:693-704` coleta qualquer link com âncora SEM checar estado; o critério certo
está no mesmo arquivo (`write.go:479`: `rl.State == index.LinkAnchorMissing`). Âncoras
existentes entram em `broken_anchors` — informação que o contrato diz mudar a decisão.

**M2 · Nota inexistente responde `PATH_OUTSIDE_VAULT` em `link_graph`/`note_metadata`**
Falha-de-contrato · CONFIRMADO · `graph.go:75-78,461-464`; raiz em `resolve.go:281`
`errors.New("not found")` genérico, sem sentinela; o handler não distingue e mapeia
tudo para o código errado. O modelo ramifica no código e recebe instrução errada.

**M3 · `note_metadata.headings` devolve só texto; contrato promete nível/slug/offsets**
Falha-de-contrato · CONFIRMADO · `graph.go:497-502` vs `docs/TOOLS.md:164`
O dado rico (`parser.Heading`) existe no índice e é descartado na projeção.

**M4 · `include=inline_fields` aceito e ignorado** · Falha-de-contrato · CONFIRMADO
`graph.go:489-516` não trata o valor listado em `docs/TOOLS.md:156`; o dado existe
(`Note.Inline`). Classe "schema promete, handler ignora".

**M5 · `link_graph` devolve menos que o prometido** · Falha-de-contrato · CONFIRMADO
`graph.go:29-46` vs `docs/TOOLS.md:206`: nós sem distância, arestas sem
alias/âncora/resolvido; arestas quebradas indistinguíveis. Dados disponíveis em
`Backlink`.

**M6 · `ResolvePath`: comentário promete caminho/nome/alias; corpo faz só exato+lowercase;
e é O(N) com erro morto** · Falha-de-contrato/Performance · CONFIRMADO
`resolve.go:256-288`: sem resolução por `byName`/`byAlias` (que `resolveTarget` sabe
fazer), embora seja a porta única de todas as tools; varredura do mapa inteiro
(:274-278) onde lookup direto responderia, e `ErrAmbiguousPath` nesse ramo é
inalcançável (chave única por construção, `index.go:163-169`).

**M7 · `Replace` reimplementa a publicação inline, contornando `publishNoteLocked`**
Higiene (Regra da chave única) · CONFIRMADO · `update.go:111-125` vs `index.go:140-169`
Seis derivações refazidas à mão (notes, lowerPath, byName, tags, citantes + aliases).
Hoje coincidem; amanhã, não necessariamente — o padrão exato que produziu o bug
`[[STJ]]`.

**M8 · `CloseWrite` declarado, exigido no handshake e nunca chamado; EOF corta resposta em voo**
Falha-de-contrato · CONFIRMADO · `ipc/ipc.go:71,179-183`; `ponte.go:169-176`
No EOF de stdin a ponte fecha a conexão inteira, descartando respostas do daemon em
voo — precisamente o que o meio-fechamento existiria para evitar.

**M9 · `--max-results` da ponte é no-op silencioso no modo daemon**
Falha-de-contrato · CONFIRMADO · `daemon/spawn.go:36-56`, `servico.go:128-131`,
`ipc/ipc.go:120-130`
A flag não é encaminhada ao spawn e não entra no handshake; quem vale é o cfg do
primeiro daemon. Mesma classe do defeito `ReadOnlySet`/`DebounceMSSet`.

**M10 · `findMarkdownLinkSpan` produz span errado em dois casos, silenciosamente**
Bug · CONFIRMADO (impacto depende das grafias no cofre) · `parser/ast.go:266-298,322-332`
(1) Link sem filhos de texto (`"[](alvo.md)"`) faz o fallback achar a PRIMEIRA `[](` do
corpo — span de outro link, com valores plausíveis e incorretos (pior que
`offsetUnknown`). (2) `[![alt](img.png)](dest.md)` casa o `'['` da imagem e para no
`)` interno — span truncado e sobreposto, alimentando reescrita/`note_move`. Nota:
fallback copia o corpo inteiro por chamada.

**M11 · Evento cuja máscara combina `Chmod` com `Write`/`Create` é descartado inteiro**
Bug (plataforma) · SUSPEITA · `watcher/filter.go:46-49`
A checagem de `Chmod` roda antes das demais e derruba o evento composto (padrão em
backends kqueue). Windows-first contém o dano; linux/darwin são alvos declarados e não
têm reconciliação por overflow (kqueue não emite `ErrEventOverflow`, `overflow.go:18-21`)
— conteúdo só entra no próximo boot lá.

**M12 · `WriteAtomic` sem fsync de diretório e perdendo o modo do arquivo**
Bug/Higiene · CONFIRMADO · `writer/atomic.go:94,113-115,126`
Temporário nasce 0600 e nada restaura o modo original (linux/darwin: toda nota
reescrita vira 0600); `Sync` cobre só o arquivo — a durabilidade prometida no doc
comment (:85) não cobre o rename em ext4/xfs.

**M13 · Espera real sem cancelamento efetivo** · Falha-de-contrato · CONFIRMADO
`writer/atomic.go:88,121-132` (WriteAtomic sem ctx; retries com sleep);
`vault/vault.go:124-127,163-166` (checagem só na entrada — `ReadAll` sobre placeholder
bloqueia na hidratação mesmo com ctx cancelado); `MoveNote(_ context.Context)` /
`DeleteNote(_ context.Context)` fazem I/O múltiplo ignorando cancelamento
(`write.go:415,667`). A forma `_` está conforme a regra; a espera ignorada, não.

**M14 · Detecção de EOL duplicada com semânticas diferentes**
Bug-latente/Higiene · CONFIRMADO · `writer/section.go:35-40` vs `vault/eol.go:51-59`
Writer: QUALQUER `\r\n` ⇒ CRLF; vault: maioria. Arquivo com 1 CRLF e 99 LF é "LF" para
o índice e "CRLF" para o patch de seção — conteúdo novo sai todo em CRLF dentro de
arquivo majoritariamente LF. Padrão "chave derivada em dois lugares", versão EOL.

**M15 · Busca por frase exata falha com pontuação/CRLF entre palavras**
Bug · CONFIRMADO (regra por bytes) · `service/search.go:458-476` (`:465`)
Adjacência exige `posN.Start <= currEnd+1` (1 byte). `"foo, bar"` (vírgula+espaço),
travessão (3 bytes) e CRLF não casam. A regra certa é ordinal (posição de token), que
o `TokenPosition` não carrega — ver Oportunidades. Laço interno linear sobre listas
ordenadas (busca binária possível).

**M16 · Trecho ancorado na primeira ocorrência do PRIMEIRO termo da consulta**
Falha-de-contrato/qualidade · CONFIRMADO · `search/snippet.go:77-103`
`bestMatch = posicoes[0]` do primeiro termo com posting, na ordem da CONSULTA: buscar
`"13.1.10 Substituição de candidatos"` ancora em `13` (topo do documento) e o trecho
não contém a frase. IDF já calculado em `CalculateBM25` e descartado; a posição da
frase já calculada em `matchPhraseInNote` e jogada fora. Foi a causa direta do
incidente de campo de 2026-08-15 (itens 30–35 da revisão anterior); a parte de
`offset`/`next_offset` foi corrigida (Task 106), a ancoragem não.

**M17 · Toda instância responde o handshake idêntica — nada diz a qual cofre ela atende**
Falha-de-contrato (lacuna) · CONFIRMADO · `mcpsrv/server.go:39`
`mcp.NewServer(&mcp.Implementation{Name: "gobsidian", Version: Version}, nil)` sem
campo `instructions`; resources genéricos ("Nota do cofre", `resources.go:57`); URIs
`gobsidian:///<caminho-relativo>` sem identificador de cofre (`uri.go:25`). O nome do
socket deriva do cofre (`ipc.go:84`) mas é invisível ao modelo. No padrão MCP a
declaração é DINÂMICA — `initialize` (serverInfo + campo opcional `instructions`) e
`tools/list`; **não existe manifest.json obrigatório nem servido**. Quem registra N
instâncias (uma por cofre) não tem como o modelo saber em qual cofre está trabalhando,
além do rótulo da entrada no host. Direção: preencher `instructions` com a identidade
do cofre (nome via `filepath.Base` ou rótulo configurável — nunca caminho absoluto,
ver B9), expor uma tool `vault_info` (nome, contagem de notas, read-only) e, <!-- check-doc-refs: ignore vault_info -- tool PROPOSTA pelo M17; a ausencia dela e o achado -->
opcionalmente, sufixar `serverInfo.Name` com a chave do cofre. Plano na Fase 8.

### Performance — hipóteses com mecanismo (nenhuma medida; baseline obrigatório)

**P1 · `avgdl` recalculado por consulta, com sort jogado fora** · CONFIRMADO
`search/bm25.go:91-101` + `index/index.go:176-189`: `idx.Paths()` aloca e ORDENA tudo
(notas+anexos; anexos não têm `DocLength`>0 — trabalho morto além do sort) e cada
`DocLength` pega RLock individual. Número quase constante entre mutações: somar
incrementalmente no invertido (`Add`/`Remove`/sombra) ou memoizar por generation.

**P2 · `getFieldWeight` roda por OCORRÊNCIA, refazendo `idx.Get`** · CONFIRMADO
`bm25.go:80-83,168-198`: `Get` (RLock+mapa), `Contains(n.TitleNorm, term)` (substring,
não termo — "Barra" dá peso de título para busca `ar`) e varredura linear de headings
por posição. Içar por posting; headings estão ordenados por `Start` (busca binária).

**P3 · `docTermFreqs` é `map[string]map[int]float64`** · CONFIRMADO
`bm25.go:42,76-82`: um mapa por documento candidato, hash duplo por incremento. Termos
de consulta são poucos: slice paralela indexada por `queryIdx` elimina o mapa interno.

**P4 · Família `query.go`: custo constante dentro de laços quentes** · CONFIRMADO
`strings.ToLower(q.Sort)` DENTRO do comparador (:362; o `q.Order` vizinho :359 já foi
içado); `ToLower(t)` por tag por chamada (:180,214,245); `reflect.DeepEqual` no filtro
de frontmatter (:6,114,119) para tipos que comparação tipada resolve; e o sort roda
sob RLock (alonga retenção). Mais `cmp = int(a.Size - b.Size)` (:370) — transborda só
em builds 32-bit, mas `cmp.Compare` é o idioma.

**P5 · Resposta de busca serializa os mesmos hits duas vezes** · CONFIRMADO
`service/search.go:97-98,314-321,369-375`: `Hits` (omitempty) e `Results` recebem o
MESMO slice — página dobrada no fio em TODAS as buscas (até 200 × 1000 chars × 2).
Decisão de compatibilidade necessária para remover.

**P6 · `TotalSize()` é O(n) sob RLock por chamada** · CONFIRMADO (impacto delimitado)
`index/index.go:82-93`; chamadores: `vault_stats` (`graph.go:621`) e CLI. Total
incremental em insert/remove elimina a varredura.

**P7 · `escritaCache` materializa a seção fixa inteira em RAM a cada gravação parcial**
CONFIRMADO · `search/persist_codec.go:201,263-284`; gravações de 60 s durante retomada
(`serve.go:334-337`). No cofre de referência (~18,2 mi posições × 16 B ≈ 291 MB, número
documentado no próprio código, não medido nesta auditoria) o buffer vive inteiro na
heap durante cada salvamento, ao lado do índice vivo — durante justamente o regime em
que a arena mapeada existe para economizar RAM. Escrita sequencial em blocos resolve.

**P8 · Deduplicação de citantes por busca linear, por link** · CONFIRMADO
`index/update.go:362-365`: O(citantes da chave) por link; redundante — o consumidor já
deduplica (`:410-419`). Alvo-hub paga soma quadrática no Build.

**P9 · Correlação de renames lê toda candidata quando há qualquer remoção**
CONFIRMADO · `watcher/rename.go:56-69`: sem pré-filtro por tamanho (os tamanhos das
remoções já estão no índice, :39). Esvaziar lixeira enquanto chega lote grande lê
todas as candidatas, serializado.

**P10 · `GenerateSnippet` re-tokeniza os termos da consulta por hit** · CONFIRMADO
`snippet.go:78` dentro do laço; `rawTerms` já vem analisado de `Search`
(`service/search.go:168`). Memoizar uma vez por busca ou aceitar tokens prontos.

**P11 · `SweepStaleTempFiles` varre sem prefixo LongPath e pula diretórios profundos em silêncio** — **REJEITADO em 2026-08-27: a premissa é falsa.** O pacote `os` do Go aplica o prefixo de caminho longo sozinho (`fixLongPath`), e `MkdirAll`, `WriteFile` e `WalkDir` alcançaram 318 caracteres sem ele. A correção foi escrita e a prova de mutação a reprovou. A metade que estava certa — descarte silencioso de erro de subárvore — foi corrigida. Ver `docs/OPERACAO.md`.
CONFIRMADO (alcance depende do cofre) · `writer/atomic.go:57,69`; chamador
`servico.go:80` passa root cru. Contraste: `vault.Walk` usa `walkRoot` prefixado.
Além do limiar Win32, o sweep "sucede" sem descer — temporários órfãos sobrevivem lá
para sempre.

**P12 · `Analyze` sem pré-alocação nem caminho rápido ASCII** · CONFIRMADO
`search/analyzer.go:78,95-108`: `var tokens []Token` cresce por append (documento de
100 KB ⇒ dezenas de milhares de tokens, ~20 realocações com cópia) e decodifica runa a
runa sem tabela para `r < utf8.RuneSelf`. É custo da CONSTRUÇÃO do índice (boot e
retomadas). Item 27 da revisão anterior, re-verificado hoje.

**P13 · `Search` faz `index.Get` duas vezes por hit da página** · CONFIRMADO
`service/search.go:201` (filtro) e `:254` (`montaSlot`). Carregar a nota no primeiro
Get e transportá-la resolve; cuidado com a corrida documentada de nota removida entre
as fases (o comentário de `montaSlot` existe por isso).

**P14 · `IsCloudOnly` custa um syscall por entrada da varredura, evitável**
CONFIRMADO · `vault/walk.go:174,185`: `d.Info()` já foi obtido em :174; no Windows ele
vem preenchido do `FindFirstFile` — os mesmos bits, sem syscall, via
`info.Sys().(*syscall.Win32FileAttributeData)`. (Em `Inverted.Update` o syscall é
necessário — não há `FileInfo` em mãos.)

**P15 · Varredura de temporários no caminho crítico do boot, em toda instância**
Performance · CONFIRMADO · `cmd/gobsidian/servico.go:80-84`
`SweepStaleTempFiles` percorre o cofre inteiro (`WalkDir`, `writer/atomic.go:55-80`)
ANTES de montar watcher e serviço — recuperação de crash que não é prerequisito para
responder o initialize. Numa partida simultânea de N instâncias são N varreduras
seriais competindo por disco antes do primeiro byte; contribui direto para a demora de
listar ferramentas. Direção: SOBREPOR à carga do índice de metadados (goroutine
iniciada junto, `join` antes de servir) — preserva a garantia documentada em
`atomic.go:20-32` ("o único lugar sem escrita em voo é o boot") e troca a soma serial
pelo máximo das duas etapas. Variante posterior com guarda (sweep pós-announce pulando
caminhos sob trava do writer, `lock.go`) exige teste da corrida ENOENT documentada —
não fazer sem ele. Detalhe no plano, Fase 2.4.

### Baixos — higiene e latentes

- **B1** `limitePosicoes = 4_000_000_000` excede o `int32` de `postIni` — cache adversário
  trunca fronteiras em silêncio; e `make([]TokenPosition, totPos)` perto do teto pede
  dezenas de GB. `persist_codec.go:75,506,536,559`. CONFIRMADO (mecanismo).
- **B2** Janela latente: `Postings` devolve janelas da arena mapeada iteradas pelo BM25
  FORA do lock; `promoverArenaSePresente` desmapeia depois de soltar o lock
  (`mmap.go:167-184`). Hoje bloqueado só pelo gate `Building()` — dependência não
  documentada. CONFIRMADO (latente).
- **B3** Contrato de erro de `GenerateSnippet` é morto — todos os ramos devolvem `nil`;
  chamador ignora (`snippet.go:32,54,65,108,163`; `search.go:258`). CONFIRMADO.
- **B4** `note_list` sem teto de limit (500 não aplicado — `tools_read.go:136-171`,
  `query.go:396`); enums de schema são strings livres com fallback silencioso;
  `note_patch` mode inválido responde INTERNAL em vez de INVALID_ARGUMENT
  (`write.go:371-372`). CONFIRMADO.
- **B5** `note_delete to_trash` descarta o erro do remove original (`write.go:761`) —
  nota travada pelo Obsidian ⇒ `Deleted=true` com a nota em dois lugares. CONFIRMADO.
- **B6** Travas do move adquiridas em ordem fixa from→to sem ordenação global — deadlock
  AB-BA teórico entre moves opostos (`write.go:609-612`, `lock.go:37-61`). SUSPEITA.
- **B7** `IsCloudOnly` é fail-open: erro de `GetFileAttributes` ⇒ `false` (hidratado) —
  o pré-filtro deixa de proteger exatamente quando falha (`cloud_windows.go:23-26`).
  Cobertura de atributos, por outro lado, completa. CONFIRMADO (política a decidir).
- **B8** `UnifiedDiff` emite cabeçalho inválido para ranges vazios no topo
  (`diff.go:200,222`) — só atinge consumidor que aplicar o patch. SUSPEITA.
- **B9** Mensagens de erro vazam caminhos absolutos da máquina ao host
  (`vault/path.go:115,120` → `mapVaultErr`; `write.go:124,605`). CONFIRMADO.
- **B10** Deliberação commitada remanescente: andaime de mutação rodando em produção
  UMA VEZ POR NOTA (`index/note.go:143`, via `normalizeTitleForNote` — chamado em
  `index.go:123`, `update.go:92`, `classify.go:60`, `persist_codec.go:691`);
  perguntas em inglês em `read_test.go:29-37`; narrativas datadas em
  `graph.go:408-416,584-587`, `write.go:217-223`, `server.go:66-70`. CONFIRMADO.
- **B11** Duas constantes independentes guardam o MESMO gate de versão do cache de
  metadados (`persist.go:22` × `persist_codec.go:44`) — subir uma e não a outra faz o
  leitor recusar TODO save novo, rebuild a cada boot, sem log. CONFIRMADO.
- **B12** `resolveAllLinks` muta Note publicada fora de `mutarNotaLocked`; seguro hoje
  por ordem de chamadas, não por construção (`resolve.go:59-69`). CONFIRMADO.
- **B13** `index/assets.go` é um arquivo-inteiro-comentário; package doc já vive em <!-- check-doc-refs: ignore index/assets.go -- FECHADO em 2026-08-28: o arquivo foi apagado, e e por isso que a referencia nao resolve -->
  `alias.go`. **FECHADO em 2026-08-28** — apagado.
- **B14** Comentário de `parser/types.go:103-108` afirma que offsets de link Markdown
  ficam em `offsetUnknown`; código e teste dizem o contrário
  (`ast.go:45-58`, `markdown_link_test.go:71-82`). CONFIRMADO.
- **B15** Remove+create com bytes idênticos no mesmo lote correlaciona como rename e
  transfere a entrada via `MoveNote` — resolução afirma equivalência não pedida
  (`rename.go:76-94`). SUSPEITA (frequência).
- **B16** Dívidas residuais de `MoveNote` no índice: `!hasNote` retorna em silêncio
  ("moveu"="não havia nada"); ramo `v==nil` faz `Stat` de caminho RELATIVO contra o
  CWD (`update.go:489-492,510-516`). Inalcançável hoje pelos chamadores atuais.
  CONFIRMADO (latente).
- **B17** Dry-run engole erro de leitura (`fromRaw, _ :=` em `write.go:514`) — diff
  vazio apresentado como resultado. CONFIRMADO.
- **B18** Nota documental: `internal/ipc` e `internal/daemon` importam `net` — conforme
  a RNF-30 reformulada (Task 90: "nenhum socket que saia da máquina"; gate
  `tools/netcheck` com `"unix"` literal cumprido), mas destoante da redação antiga da
  regra que proíbe `net` sob `internal/`. Alinhar o texto para a próxima revisão não
  apontar falsa violação. CONFIRMADO (fato; prevalência é decisão do dono).

---

## Oportunidades maiores (decisão de arquitetura; da revisão de 2026-08-15, ainda abertas)

Registradas sem re-análise profunda nesta rodada — a revisão anterior as detalha com
código e trade-offs (`docs/REVISAO-2026-08-15.md`):

1. **BM25 em espaço de IDs densos** (item 36): usa o `baseSoA` que já atribui IDs
   int32; subsume P1–P3 de brinde e tira milhares de mapas do grafo do GC por consulta.
   Maior ganho de desempenho pendente.
2. **Ordinal de token no índice** (estrutura do item 29): habilita frase exata correta
   (M15), proximidade e ranking por proximidade.
3. **SoA no índice de metadados** (item 37) e **buffers do chamador** (item 38):
   memória e escape analysis; só depois de medir.
4. **Busca por prefixo/aproximada**: os termos do `baseSoA` já estão ordenados —
   `sort.SearchStrings` dá autocomplete/prefixo de graça.
5. **`netcheck` verificável por nome de método** + `Dialer.DialUnix` (itens 19/42),
   nesta ordem: corrigir o analisador antes de adotar.
6. **Recursos de produto** (itens 30–35, 45–48): `note_outline` (estrutura sintética <!-- check-doc-refs: ignore note_outline -- tool proposta na Fase 9; ainda nao existe, e esse e o ponto -->
   rotulada), restauração de lixeira, `vault_lint`, prompts MCP, `vault_stats` <!-- check-doc-refs: ignore vault_lint -- tool proposta na Fase 9; ainda nao existe, e esse e o ponto -->
   expondo o estado do índice de busca (`Building()` existe e não é publicado).

---

## Alternativas ao modelo de daemon e partida de várias instâncias

O que o daemon compra hoje — números **medidos pelo projeto** (CLAUDE.md, cofre real
de 4.513 notas, memória física agregada):

| sessões (mesmo cofre) | pré-M7 | sem daemon (mmap) | com daemon |
|---|---|---|---|
| 1 | 579,1 MB | 244,6 MB | 223,6 MB |
| 3 | 1.681,3 MB | 508,5 MB | 262,2 MB |
| 5 | 2.916,4 MB | 773,4 MB | 229,4 MB |

Duas leituras: o **mmap sozinho já captura ~73% do ganho** (o page cache do SO
compartilha as páginas entre processos sem protocolo nenhum); e o daemon só vence
decisivamente com **várias sessões do MESMO cofre** — para cofres diferentes ele nada
compra (um daemon por cofre; memória soma igual). O que ele custa é a classe inteira
de defeitos desta auditoria: A4 (daemon surdo), A5 (socket vivo desvinculado), M8
(CloseWrite), M9 (flags no-op) e a corrida residual do lock. E o sintoma que motivou a
pergunta tem cadeia própria: N instâncias subindo juntas estouram o prazo do probe do
`EnsureStarted` → spawn duplicado → `cleanupSocketFile` rouba o socket do daemon vivo
(A5) → pontes conectam num daemon que nunca responde → o initialize pendura e o host
não lista ferramentas. (`INDEX_BUILDING` afeta só busca; listagem nunca depende dela.)
A varredura serial do boot soma-se no mesmo instante (P15).

**(a) Sem daemon — mmap puro, sessões autônomas.** Toda a superfície de socket/proxy/
handshake morre por construção. Custo: memória volta a escalar (~100–160 MB/sessão
além das páginas compartilhadas); watchers duplicam por sessão (CPU/IO ×N, correto mas
desperdiçado); e surge risco novo que hoje o daemon abafa — **duas instâncias frias do
mesmo cofre constroem e gravam concorrentemente o MESMO cache de busca** (saves
parciais em `buildInvertedIndex`, `serve.go:284-288`). Exige arbitração por lock de
escritor ("primeira constrói, demais esperam/adotam") antes de virar padrão.

**(b) Publicador único + leitores finos (single-writer).** Um processo pequeno de fundo
detém watch + construção e publica GERAÇÕES do cache sob lock; as sessões viram
servidores sem estado que mapeiam o snapshot publicado e conferem geração por
requisição. Sem socket RPC nem proxy de bytes — as classes A4/A5/M8/M9 morrem; o
watcher não multiplica; e a carga bloqueante (A6) desaparece porque sempre há snapshot
quente. Custos: defasagem limitada ao intervalo de publicação, escrita amplificada do
cache, política de invalidação por geração. É a arquitetura que ataca a demora de
partida pela raiz.

**(c) Daemon multi-cofre.** Um processo hospeda N cofres roteados pela `VaultKey` que
o handshake JÁ carrega (`ipc.go:127-130`). Menos processos e watchers quando muitos
cofres simultâneos; custos: raio de explosão concentrado (uma queda derruba todos),
flags por cofre generalizam o M9, memória soma igual (índices distintos).

**Enquadramento da decisão:** as escolhas D-M7-* estão fechadas e não se re-litigam
sem dado novo. **O dado novo foi medido em 2026-08-25** — ver abaixo.

### Medição do cenário real do dono (2026-08-25)

Binário `v1.2.1-32-g1cbb007`, cofres reais, cache quente por cofre, `--read-only`,
cache isolado fora do de produção. Cada sessão faz o handshake MCP completo **e uma
`vault_search`** — o índice de busca é preguiçoso, então medir uma sessão que nunca
buscou responde outra pergunta. Pico de `WorkingSet64`, 8 amostras, após 4 s de
acomodação. Agregado = soma de todos os processos (pontes + daemons).

Cofres: TJSP 192 (6.793 notas, 231 MB, zero placeholders), Estudo (2.557), Revisão
(1.275), Jurisprudência (1.254), Oral (78).

| Cenário | Sem daemon | Com daemon | Efeito |
|---|---|---|---|
| **A — 5 cofres × 1 sessão** (o uso real do dono) | 5 proc, **565,5 MB** | 10 proc, **649,6 MB** | **+84,1 MB (+14,9%)** |
| **B — TJSP 192 × 3 sessões** (mesmo cofre) | 3 proc, **782,6 MB** | 4 proc, **316,4 MB** | **−466,2 MB (−59,6%)** |

O mecanismo aparece por processo: no braço A, cada daemon pesa o mesmo que a sessão
autônoma pesava (149,7 × 149,5; 261,4 × 261,1; 80,5 × 80,1) e as cinco pontes
acrescentam ~16,4 MB **cada**. Um daemon por cofre não compartilha nada com os outros
cofres — o daemon vira custo puro. No braço B o mesmo mecanismo trabalha a favor: um
índice pago uma vez, três pontes de ~15 MB.

**Conclusão:** o daemon entrega exatamente o que foi projetado para entregar, e a carga
de trabalho do dono é a que ele não atende. A superfície de defeito que ele carrega
(A4, A5, M8, M9, corrida residual do lock) está sendo paga sem contrapartida no
cenário A.

Observação de campo, mesma data: com seis sessões reais vivas na máquina do dono
(v1.2.1, 3 cofres × 2 sessões, 247 min de vida), **nenhum processo daemon existia**.
Ou o daemon expirou por ociosidade sob sessões conectadas, ou as pontes caíram no
fallback em processo. Diagnosticar qual é pré-requisito para qualquer decisão de
migração — se for a segunda, o braço B nunca acontece em produção.

Custo a frio, medido de passagem (cache vazio, mesma metodologia): TJSP 192 **1.417 MB**,
Estudo **830 MB**, Jurisprudência **328 MB**, Revisão **130 MB**, Oral **71 MB**.
Contra os mesmos cofres com cache quente: 261 / 149 / 80 / 45 / 30 MB. O cache de busca
vale 5,4× de RSS no cofre grande.

### RNF-07 é medido por um caminho que exclui o índice de busca

`scripts/measure.ps1` emite `initialize` + `vault_stats` e mede. Desde a carga
preguiçosa (Task 88), uma sessão que nunca buscou **nunca carregou o invertido** — então
o número publicado descreve um servidor que ainda não buscou. A/B com uma única variável
(uma `vault_search`), cache quente, mesmo binário:

| Cofre | A: só `vault_stats` | B: + uma `vault_search` | Delta |
|---|---|---|---|
| Oral (78 notas) | 22,3 MB | 29,5 MB | +7,2 MB (1,32×) |
| Estudo (2.557 notas) | **53,1 MB** | **149,0 MB** | **+95,9 MB (2,81×)** |

`docs/OPERACAO.md:193` registra RNF-07 (≤ 60 MB) como **Atingido** com 37,95–38,10 MB.
Pelo protocolo A, Estudo dá 53,1 MB e passa; pelo protocolo B dá 149,0 MB, **2,5× o
alvo**. Não é número inventado — é a lição "meça pela camada que o requisito nomeia"
incidindo sobre o próprio RNF-07: a mesma Task 88 que melhorou o número é a que tirou
o índice de busca de dentro dele. Decisão do dono: ou o RNF-07 passa a nomear o estado
"já atendeu uma busca" (e o alvo é re-negociado com o número real), ou a tabela declara
explicitamente que mede o servidor antes da primeira busca. O que não pode continuar é
**Atingido** sem essa qualificação.

---

# PLANO DE CORREÇÃO E IMPLEMENTAÇÃO

## Princípios (não negociáveis neste plano)

1. **Teste antes de consertar.** Cada item entra por um RED que reprova pelo nome no
   caminho REAL (não por dublê), GREEN mínimo, e prova de mutação via
   `scripts/mutate.ps1` com saída colada no relatório da tarefa. Regra escrita sob
   suíte verde não está verificada.
2. **Nenhuma mudança de performance sem baseline.** Rodar a bateria atual
   (`go test -bench . -benchmem -count≥6` nos pacotes tocados + `benchstat`) ANTES,
   declarar a regra de decisão antes de medir, e aceitar "~" como resposta final
   (mudança sem ganho significativo é dívida pura — precedente Task 82).
3. **Uma conta só por regra.** Guardas e derivações moram num lugar (lições
   `aliasKey`, `publishNameLocked`, EOL). Correção que cria segunda implementação
   paralela não é aceita.
4. **Escopo não encolhe em silêncio.** O que ficar de fora, declarado no relatório.
5. Gate de toda tarefa: `pwsh -File scripts/verify.ps1` verde + `golangci-lint run ./...`.

---

## Fase 0 — Baseline e testes que provam os defeitos (zero mudança de produção)

**Objetivo:** tornar os críticos e altos reprováveis por teste antes de tocar neles.

1. Rodar e arquivar baseline: `verify.ps1` completo, bateria de benchmarks de
   `internal/{search,service,index}` e `scripts/test_orphans.ps1 -Cycles 100`.
2. Escrever os testes que faltam (todos devem falhar HOJE):
   - **T-C1/C3** (`internal/search`, build tag windows): montar cofre com `.md`
     somente-nuvem via `FILE_ATTRIBUTE_OFFLINE` (gravável por `SetFileAttributes`,
     aceito por `IsCloudOnly`) e anexo local grande; exercitar **`buildInvertedIndex`
     de verdade** e `watcher.Apply` + `Reconcile`; afirmar que NEHNUM arquivo
     somente-nuvem nem anexo foi aberto (contador/interceptação de open), e que o
     invertido registra ambos como entradas vazias. Hoje: falha (boot abre, Apply lê
     anexo, Reconcile re-lê anexo).
   - **T-C2** (`internal/vault`): fixture com symlink de arquivo apontando para fora do
     cofre; afirmar que Walk não indexa e que Open/ReadAll recusam. Hoje: falha.
   - **T-A3** (`internal/watcher` ou `internal/index`): forçar falha de `ReadAll` no
     segundo Replace de uma nota; afirmar que metadados e busca respondem IGUAL depois
     do erro (nota continua visível OU sai dos dois). Hoje: falha.
   - **T-A7** (`internal/watcher`): remover a posting da busca manualmente, emitir
     evento com mtime/tamanho iguais, afirmar que o índice de busca se recompõe.
     Hoje: falha.
   - **T-M15** (`internal/service`): frases com `"foo, bar"`, `"foo — bar"`, CRLF entre
     palavras. Hoje: falham.
3. Entregar a Fase 0 como tarefa própria (relatório com as saídas vermelhas coladas).

**Critério de pronto:** cinco testes reprovando pelos motivos certos, baseline
arquivada, nenhum arquivo de produção alterado.

---

## Fase 1 — Guardas de acesso a arquivo (C1, C2, C3)

### 1.1 C3+C1 juntos: a guarda mora DENTRO de `Inverted.Update` (uma vez, três chamadores curados)

Passos:
1. Em `internal/search/inverted.go` (`Update`, ao lado da guarda `IsCloudOnly` que já
   existe em :483): acrescentar a guarda de classe —
   `if vault.Classify(path) != vault.ClassNote { ix.Add(string(path), nil); return nil }`.
   Anexo passa a entrar por nome (vazio), igual ao placeholder de nuvem. Decidir e
   registrar: entrada-vazia para anexo é o mesmo tratamento do cloud-only.
2. Rotejar o boot pelo caminho guardado — escolher UMA forma e apagar a outra conta:
   (a) `buildInvertedIndex` chama `inv.Update(ctx, v, p)` por nota (simples; repete a
   checagem `HasDoc` já feita no laço de retomada — inofensivo), ou (b) extrair o miolo
   comum ler+tokenizar numa função interna usada por `Update` e pelo boot, com as
   guardas NA função comum. Preferir (b): torna impossível o próximo chamador repetir
   o C1.
3. Corrigir o comentário falso de `inverted.go:466-469` para descrever os chamadores
   reais.
4. Apontar o dublê `construirComoOBoot` (`cloudonly_update_windows_test.go:132-143`)
   para exercitar o código de produção (função comum ou `buildInvertedIndex`), ou
   substituí-lo pelos testes T-C1/C3 da Fase 0.
5. Prova de mutação: remover a guarda de classe de `Update` ⇒ T-C3 reprova; remover a
   de nuvem ⇒ T-C1 reprova. Remover o roteamento do boot ⇒ T-C1 (variante boot)
   reprova.
6. Verificar efeito colateral: `Reconcile` deixa de re-ler anexos (atalho passa a bater
   para anexo? Não — `idx.Get` continua não resolvendo anexo; o atalho de `overflow.go`
   precisa do complemento do item 1.4 abaixo). Registrar no relatório o que mudou na
   contagem de eventos processados.

### 1.2 C2: confinamento contra symlink — duas opções, decisão antes de codar

- **Opção mínima (S)**: (i) `vault.Walk` pula entradas com `d.Type()&fs.ModeSymlink != 0`
  registrando skip (como faz com ruído); (ii) centralizar a abertura: `Open`/
  `ReadRange`/`ReadAll`/`WriteAtomic` resolvem o componente final com `filepath.EvalSymlinks`
  e recusam alvo fora da raiz (uma função única em `vault`, usada por todos). Testes
  T-C2 ficam verdes; mutação: remover a checagem ⇒ T-C2 reprova. Custo baixo; janela
  TOCTOU residual documentada.
- **Opção estrutural (M/L)**: migrar aberturas para `os.OpenRoot(root)` (Go ≥1.24):
  `Root.Open/Create/Stat/Remove/OpenFile` tornam o escape impossível por construção,
  eliminam TOCTOU e a classe inteira (inclui futuros vetores 8.3/junction). Toca em
  `vault.Open/ReadRange/ReadAll`, `writer.WriteAtomic` (criar temporário + rename
  dentro do root — renomear via `Root.Rename`?) e no sweep de temporários. Exige ADR
  novo (AD-10) e revisão dos testes de plataforma.
- **Recomendação**: Opção mínima AGORA (fecha o buraco com risco pequeno), Opção
  estrutural como tarefa de arquitetura própria com medição de impacto em boot
  (syscalls extras por nota) antes de adotar.

### 1.3 A4: classificar erros de Accept

1. No ramo de erro (`daemon.go:147-153`): sair só quando
   `errors.Is(err, net.ErrClosed)` ou `ctx.Err() != nil`; caso contrário, logar com
   contador, dormir backoff curto (começar 50 ms, teto 1 s) e `continue`.
2. Teste: listener que falha N vezes (injeção) e volta — afirmar que conexões seguintes
   são aceitas; mutação: remover o backoff/continue ⇒ teste reprova.
3. Cuidado: manter a goroutine vigia de `ln.Close()` como está (:140-143).

### 1.4 A5: provar órfão antes de unlink + lock único para `runDaemon`

1. `ipc.Listen`: antes de `cleanupSocketFile`, tentar `DialAndHandshake` com prazo
   curto (~250 ms). Sucesso ⇒ retornar erro "daemon já ativo em <socket>" (sem tocar
   no arquivo). Falha de conexão ⇒ unlink seguro e seguir. Alternativa complementar:
   gravar PID ao lado do socket e usar `pidVivo` (já existe em `daemon/`) — preferir o
   dial-probe: funciona nos três SOs sem arquivo extra.
2. `cmd/gobsidian/daemon.go`: adquirir o MESMO lock de `EnsureStarted`
   (`daemon/adquirirLock`) antes de `Listen`, com o segundo-dial idempotente que o
   caminho de ponte já faz — fecha a janela do chegante tardio por construção nos
   casos manuais.
3. Testes: (i) socket ocupado por listener vivo ⇒ `Listen` falha sem remover o arquivo
   (afirmar que o arquivo sobreviveu e o primeiro daemon continua aceitando);
   (ii) socket órfão (arquivo sem ouvinte) ⇒ limpado e bindado. Mutações: remover o
   probe ⇒ (i) reprova.
4. Fechar o item do OPERACAO.md sobre a corrida residual quando os dois mecanismos
   estiverem no lugar (atualizar o doc na mesma tarefa).

---

## Fase 2 — Daemon/IPC restantes (A6, M8, M9)

### 2.1 A6: espera cancelável com INDEX_BUILDING honesto

1. Trocar o mutex-do-carregamento por porta `chan struct{}` fechada na conclusão:
   quem chega durante a carga faz `select { case <-porta: case <-ctx.Done(): return
   INDEX_BUILDING }` — prazo do chamador respeitado, código de erro honesto.
   Manter a semântica de retentativa pós-falha (não marcar pronta em erro).
2. O PRIMEIRO chamador continua disparando e esperando a carga com o próprio ctx —
   decidir se recebe INDEX_BUILDING após orçamento (ex.: 2 s) em vez de bloquear
   minutos; recomenda-se sim: devolver INDEX_BUILDING com `Retryable` e deixar o host
   reconsultar (a carga continua em background — requer desacoplar o f() da vida do
   primeiro chamador: goroutine dona da carga, referenciada pela porta).
3. Corrigir o comentário de `servico.go:104-107` para descrever os dois modos.
4. Testes: (i) chamador com ctx de 100 ms durante carga longa recebe INDEX_BUILDING e
   a carga prossegue; (ii) após conclusão, buscas normais; (iii) falha da carga permite
   nova tentativa (já existe — manter verde). Mutação: fechar a porta antes do fim ⇒
   (i) reprova.

### 2.2 M8: meio-fechamento no EOF

1. Na cópia host→daemon (`ponte.go`), no EOF: `conn.CloseWrite()` e continuar drenando
   daemon→host até EOF/erro ou teto de drenagem (ex.: 5 s) antes do `Close` total.
2. Teste com o dublê `duplexPipe` existente (`ponte_test.go:244-257`): pedido seguido
   de EOF imediato; afirmar que a resposta chega. Hoje deve falhar; mutação: remover o
   `CloseWrite` ⇒ reprova.

### 2.3 M9: `--max-results` atravessa a fronteira

1. Encaminhar `--max-results` em `SpawnDetached` (`spawn.go:36-56`).
2. Acrescentar `MaxResults int` ao `HandshakeConfig` + verificação no handshake
   (recusa `ErrConfigMismatch` em divergência), espelhando `ReadOnly`.
3. Teste: ponte com `--max-results=50` contra daemon com padrão ⇒ handshake recusa OU
   (alternativa de produto) documento declara que o primeiro daemon fixa o parâmetro —
   escolher UMA e registrar a decisão.

### 2.4 Partida rápida de várias instâncias (P15 + prazos do EnsureStarted)

1. Sobrepor a varredura de temporários à carga do índice de metadados: goroutine
   iniciada junto em `montarServico`, `join` antes de servir — preserva a garantia
   "nenhuma escrita em voo" (`atomic.go:20-32`) e troca soma serial por máximo das
   duas etapas. Teste: boot com cofre fixture grande; afirmar que o sweep rodou
   (temporário órfão removido) e que o announce não esperou a varredura acabar.
2. `EnsureStarted` paciente: substituir o probe único por espera no lock até o socket
   aparecer, com N tentativas de dial espaçadas dentro do orçamento total (ex.: 10 s
   em tentativas de 250 ms); falhar só depois do orçamento. Fecha a metade da
   tempestade de partida que spawna duplicados — a outra metade é o unlink do A5
   (Fase 1.4).
3. Teste da tempestade: ≥10 pontes simultâneas sobre cofre frio — afirmar EXATAMENTE
   um daemon e todas as sessões listando tools dentro de prazo. Mutação: remover a
   espera do EnsureStarted ⇒ teste reprova com dois daemons.

---

## Fase 3 — Consistência dos dois índices e escrita (A2, A3, A7, A1, B5, B17, M12, M13)

### 3.1 Redesenho do `Replace` em duas fases (cura A2 + A3 juntos)

1. Nova ordem dentro de `Replace`: (i) FORA do lock — `os.Stat`, `IsCloudOnly`,
   `ReadAll`, parse; (ii) SOB lock — construir a `Note`, remover contribuições antigas,
   publicar novas, reprocessar citantes dirigido. A janela entre as fases precisa de
   política explícita (re-checar existência sob lock; se sumiu, tratar como Remove).
2. Caminho de erro: NUNCA deixar a nota fora dos metadados sem republicar o estado
   anterior (guardar as contribuições removidas e republish em caso de erro de
   leitura/parse — ou só remover depois da leitura bem-sucedida, o que a nova ordem já
   dá de graça).
3. Publicação exclusivamente via `publishNoteLocked` + loop de `aliasKey` (cura M7 no
   mesmo movimento — apagar o bloco duplicado de `update.go:111-125`).
4. Testes: T-A3 da Fase 0 fica verde; teste novo de concorrência: leitores em loop
   (`Get`/`List`) durante Replace com arquivo grande — sem starvation mensurável além
   do ruído (asserção de teto atrás de build tag race, como manda a casa); mutações:
   (i) remover republish de erro ⇒ T-A3 reprova; (ii) voltar a publicar inline ⇒ teste
   de paridade de publicação (comparar índices construídos pelas duas rotas campo a
   campo) reprova.

### 3.2 A7: atalho do `Apply` espelha a condição da `Reconcile`

1. `apply.go:85-92`: pular só quando
   `n.ModTime.Equal(...) && n.Size == ... && (searchInv == nil || searchInv.HasDoc(string(path)))`.
2. T-A7 fica verde; mutação: remover o `HasDoc` ⇒ reprova.

### 3.3 A1 (+B5, B17): move/delete/dry-run honestos

1. `MoveNote`: mover o CORPO primeiro — `os.Rename(absFrom, absTo)` (mesmo volume é o
   caso comum; fallback: `WriteAtomic` + `os.Remove` COM o erro verificado; remove
   falhando ⇒ erro da chamada com resultado indicando "destino criado, origem
   intacta"). Só DEPOIS reescrever citantes; em erro de reescrita, registrar as já
   reescritas no resultado (lista + diff) para compensação consciente — rollback
   automático opcional usando o `raw` em mãos.
2. Ordem das travas: adquirir por chave normalizada menor-primeiro (cura B6 de graça).
3. `DeleteNote to_trash`: verificar o erro do remove (:761) e mapear para FILE_LOCKED
   com mensagem informando a cópia na lixeira (B5).
4. Dry-run: propagar erro de leitura (`write.go:514`) em vez de diff vazio (B17).
5. Testes: duplicata (remove injetado falhando) reprova; rollback/registro das citantes
   em falha; trash travada; dry-run com arquivo ilegível. Mutações correspondentes.

### 3.4 M12: durabilidade e modo

1. `WriteAtomic`: preservar modo (`os.Stat` do alvo antes; `tmpFile.Chmod(mode)` antes
   do rename; fallback 0644/-umask quando não existir).
2. fsync do diretório após rename: unix via `OpenFile(dir, O_RDONLY)`+`Sync` (melhor
   esforço documentado); Windows: sem equivalente barato — documentar no doc comment
   que a garantia declarada vale para unix, ajustando o texto de :85 para não prometer
   além do entregue.
3. M13 parcial: acrescentar `ctx` a `WriteAtomic` e checar entre retries
   (`select ctx.Done()/time.After`); chamadores atualizados (assim Move/Delete ganham
   cancelamento real onde hoje há `_`).
4. Cancelamento de `ReadAll` durante hidratação: padrão goroutine-abre+`select` em
   `ctx.Done()` fechando o handle — tarefa separada com teste de timeout, pois toca o
   caminho quente de leitura.

### 3.5 M14: uma detecção de EOL

Apagar `writer.DetectEOL/NormalizeEOL` (`section.go:35-40`) e consumir
`vault.DetectEOL/NormalizeEOL` nos pontos de `write.go:227,345-346`. Teste de
paridade: mesmo arquivo misto decide igual no índice e no patch (golden com 1 CRLF +
99 LF). Mutação: restaurar a heurística do writer ⇒ reprova.

---

## Fase 4 — Contratos que mentem (A8, M1–M6, M11, B4, B14, B18, higiene B10/B11/B13/B16)

Decisões de produto curtas, depois execução mecânica. Para cada campo: **ou implementa,
ou tira do schema e do TOOLS.md** — nunca deixar prometido e ignorado.

1. **A8 `Backlink.Context`**: preencher na resolução — o offset Start/End do link e o
   corpo já lidos no parse permitem recortar ±N bytes; persistir no codec (bump de
   versão do cache de METADADOS — usar a constante ÚNICA do B11). Ou remover campo +
   promessa. Recomenda-se implementar: backlink sem contexto obriga abrir a origem.
2. **M1**: extrair predicado único `ancoraQuebrada(rl)` (usado por move :479 e delete
   :693). Teste: âncora existente não aparece em broken_anchors do delete; mutação:
   voltar ao critério sem State ⇒ reprova.
3. **M2**: sentinela `index.ErrNotFound` (e revisitar `ErrAmbiguousPath` morto do M6);
   mapeamento único erro→Code em `service/errors.go`; `link_graph`/`note_metadata`
   passam a responder NOTE_NOT_FOUND. Teste por tool.
4. **M3/M4/M5**: projetar tipos wire ricos — `headings[] {level,text,slug,start,end}`,
   `inline_fields{}` de `Note.Inline`, `edges[]{source,target,kind,alias,anchor,resolved}`
   + `nodes[].distance`. Atualizar TOOLS.md na mesma tarefa. Testes golden por retorno.
5. **M6**: implementar as etapas anunciadas no próprio comentário — resolução por
   `byName` e `byAlias` com o tiebreak existente — OU corrigir comentário + docs para
   "caminho canônico ou grafia de caminho". Substituir a varredura O(N) por lookup
   direto `ix.lowerPath[lower]` (o ramo `len(matches)>1` morre junto — documentar que
   ambiguidade real exigiria multimap, decisão explícita).
6. **M11**: dropar só `e.Op == fsnotify.Chmod` exato; máscaras compostas priorizam
   Create/Write/Remove/Rename. Teste com máscara combinada.
7. **B4**: clampear `limit` ao teto 500 no boundary mcpsrv; validar enums na entrada
   (INVALID_ARGUMENT) e, se o SDK permitir, declará-los no jsonschema.
8. **B14**: reescrever o comentário de `types.go:103-108` para o contrato real
   (preenchido quando determinável; `offsetUnknown` só no fallback).
9. **B10**: apagar `note.go:143` (+ ajustar a prova de mutação que o motivava — âncora
   diferente em `mutate.ps1`), os comentários-pergunta de `read_test.go:29-37` e
   reescrever as narrativas datadas como justificativas presentes.
10. **B11**: uma constante exportada (`IndexCacheFormatVersion`) consumida pelo codec.
11. **B13**: apagar `assets.go`. **FEITO em 2026-08-28.** <!-- check-doc-refs: ignore assets.go -- arquivo apagado; a referencia e historica -->
12. **B16**: `MoveNote` devolve `(bool moved, error)`; caller deixa de contar às cegas;
    exigir `*vault.Vault` não-nil (remover ramo `v==nil`).
13. **B18**: decisão do dono sobre a redação da regra; atualizar AGENTS.md/CLAUDE.md
    para a redação RNF-30 vigente, mantendo o netcheck como garantia mecânica.

---

## Fase 5 — Qualidade da busca (M15, M16)

### 5.1 M15: adjacência ordinal para frase

1. Curto prazo (sem formato novo): casar por sobreposição de intervalos — o token
   seguinte deve começar em `>= currEnd` e a lacuna não pode conter letra/dígito…
   impossível saber sem re-tokenizar. Alternativa honesta de curto prazo: tolerância de
   gap configurável (≤ 8 bytes) documentada como aproximativa — decide-se com o dono.
2. Estrutural: ordinal do token em `TokenPosition` (varint delta-encoded cabe no
   formato 6 sem inflar; bump de `CacheFormatVersion`) — frase exata correta de uma
   vez e base para proximidade. Ver Oportunidade 2.
3. Laço interno: listas ordenadas ⇒ busca binária.
4. Testes da Fase 0 (T-M15) definem o verde; corpus com pontuação real (vírgula,
   travessão, CRLF).

### 5.2 M16: ancoragem do trecho

1. Passar a melhor âncora por seletividade: `CalculateBM25` já calcula IDF por termo —
   devolver `termIDFs` (ou o termo de menor df) e usar a primeira ocorrência DESSE
   termo como âncora inicial.
2. Quando a consulta é frase (`isPhrase`), usar a posição calculada por
   `matchPhraseInNote` (hoje descartada) como âncora definitiva.
3. Melhoria adicional (opcional, mesma tarefa): janela deslizante maximizando densidade
   de termos distintos.
4. Teste: nota longa estilo livro (fixture do incidente: termo numérico no topo, frase
   procurada muito adiante) — trecho devolvido CONTÉM a frase. Golden.

---

## Fase 6 — Performance medida (P1–P14; o P15 está na Fase 2.4), nesta ordem

Disciplina: baseline por item (benchstat, n≥6, braços intercalados em máquina ruidosa),
regra de decisão declarada antes, "~" encerra o item sem mudança.

| # | Mudança | Verificação |
|---|---|---|
| P4 | içar `switch ToLower(q.Sort)`; comparador escolhido uma vez; `cmp.Compare(a.Size,b.Size)`; `ToLower` de tags hoisted; substituir `reflect.DeepEqual` por comparação tipada (int/bool/string/float64) | `BenchmarkSearch*`/`List` existentes; mutação trivial |
| P1 | soma incremental de `docLen` no invertido (`Add`/`Remove`/sombra) + `avgdl` O(1); fallback `idx==nil` mantém comportamento | `BenchmarkSearchLimit200Cache` deve melhorar escalando com cofre; teste de paridade avgdl construído×recarregado |
| P2 | içar `idx.Get` por posting; título por limite de token (corrige substring de graça); headings por busca binária | bench de termo comum + golden de ranking (pesos não podem mudar além do título-token) |
| P3 | `docTermFreqs` → slices paralelas por `queryIdx` | mesmo golden de scores; bench allocs/op |
| P10 | memoizar análise dos termos uma vez por busca; passar tokens ao `GenerateSnippet` | bench de limit 200 |
| P12 | `make([]Token, 0, len(text)/8+1)` + tabela ASCII 128 no laço de `Analyze` | `BenchmarkIndexBuild`; golden de tokens byte-a-byte (offsets não podem mudar) |
| P7 | gravar `fixedSec` em blocos reutilizáveis (cursor sequencial já existe) | teste round-trip do codec inalterado; medir pico de RSS com `measure.ps1` se disponível |
| P6 | total incremental de tamanho em insert/remove/move | paridade TotalSize construído×incremental em teste |
| P8 | soltar o `Contains` de `registrarCitantesLocked` (consumidor deduplica) | suíte de índice inteira verde = semântica preservada |
| P9 | pré-filtro por tamanho na correlação de renames | `rename_test.go` + novo caso lote grande com remoção |
| P13 | transportar a `*Note` do filtro para `montaSlot` (uma leitura do índice por hit), preservando a corrida documentada | suíte de service verde; bench leve |
| P11 | prefixar LongPath no sweep + contar/logar diretórios não-descidos | teste com árvore profunda (skip declarado) |
| P14 | atributos via `info.Sys().(*syscall.Win32FileAttributeData)` na varredura (windows, build tag) | `walk_windows_test.go` + teste de atributo OFFLINE |
| P5 | REMOVER `Hits` OU `Results` — decisão de compatibilidade com hosts existente ANTES; alternativa: manter alias documentado por uma versão | schema/TOOLS.md atualizados |

Dependências: P1/P2/P3 tornam-se parciais se a Oportunidade 1 (IDs densos) for aprovada
— decidir ANTES de gastar os três itens.

---

## Fase 7 — Latentes e baixos restantes (B1–B3, B7–B9)

- **B1**: `limitePosicoes ≤ math.MaxInt32` + validação de bytes totais antes do `make`;
  teste com header adversário (cache rejeitado com erro claro).
- **B2**: copiar `Positions` em `Postings` quando a arena estiver mapeada (custo só no
  ramo mapeado) OU documentar+afirmar o invariante "save só pré-ready; Postings só
  pós-ready" nos dois lados. Preferir a cópia: invariante que não se testa não dura.
- **B3**: propagar o `readErr` do fallback como erro (política de trecho vazio explícita
  no chamador) ou mudar a assinatura — escolher e alinhar o `_` de `search.go:258`.
- **B7**: política explícita para atributo indisponível (log+contador; fail-open só
  documentado ou fail-closed com custo de re-checagem — decisão do dono).
- **B8**: `start-1` no lado vazio do hunk; teste com patch GNU aplicando o diff.
- **B9**: formatar erros host-facing com caminho relativo/canônico; absoluto só no slog.

---

## Fase 8 — Identidade da instância no protocolo (M17)

Objetivo: o modelo saber, sem consultar nada externo, qual cofre aquela instância
atende. Três peças independentes, na ordem de custo:

1. **`instructions` no initialize** — preencher as opções do `NewServer`
   (`server.go:39`) com: *"Este servidor atende o cofre Obsidian '<NOME>' (somente
   leitura | leitura e escrita)."* Decisão PRÉVIA obrigatória sobre a fonte do nome:
   `filepath.Base(vaultPath)` ou flag nova `--label`; **nunca caminho absoluto no fio**
   (mesma decisão do B9). Uma linha de código + golden.
2. **Tool `vault_info`** — identidade consultável a qualquer momento (o <!-- check-doc-refs: ignore vault_info -- tool PROPOSTA pelo M17; a ausencia dela e o achado -->
   `instructions` só chega uma vez): `{vault_name, notes, assets, read_only,
   index_origin}`. Preferir tool a resource: a lista de resources já é grande e cada
   nota vira URI; schema honesto desde o dia um (lição M3/M4 — só prometer campo com
   leitor). Dados todos já existem (`NoteCount`, `AssetCount`, `cfg.ReadOnly`,
   origem cache/construção já logada).
3. **Sufixo em `serverInfo.Name`** — `"gobsidian[<VaultKey>]"`: telemetria e hosts
   passam a distinguir instâncias sem tocar em mais nada. Cosmético, mas barato.

Verificações: golden do initialize afirmando name/instructions; teste da `vault_info` <!-- check-doc-refs: ignore vault_info -- tool PROPOSTA pelo M17; a ausencia dela e o achado -->
contra cofre fixture com contagens reais; mutação — apagar o preenchimento de
`instructions` ⇒ golden reprova pelo nome. Atualizar `docs/TOOLS.md` e README na mesma
tarefa (nova tool = contrato novo).

---

## Ordem sugerida de execução (resumo)

| Ordem | Itens | Por quê |
|---|---|---|
| 0 | Fase 0 | provas vermelhas + baseline; nada muda |
| 1 | C3+C1, C2(opção S) | regras não negociáveis, maior risco do usuário |
| 2 | A4, A5 | daemon zumbi/órfão afeta toda sessão |
| 3 | A2+A3+A7, A1+B5+B17, M12(+ctx) | consistência dos índices e escrita honesta |
| 4 | A6, M8, M9 | experiência da primeira busca e do modo daemon |
| 5 | A8, M1–M6, M11, B4, B10–B14, B16 | contratos honestos; mecânica de baixo risco |
| 6 | M15, M16 | qualidade da busca (dependem da decisão do ordinal) |
| 7 | Fase 6 (P4→P14) | performance com medição |
| 8 | B1–B3, B7–B9 | latentes |
| 9 | M17 (Fase 8) | barato, alto valor no uso multi-instância; pode entrar antes, é independente |
| — | Alternativas ao daemon | não é tarefa: decisão D-M7; medir (a)/(b) no cenário real antes de migrar |

## Altos em aberto — alternativas de solução (2026-08-26)

Escrito depois de fechar os três críticos. Cada mecanismo abaixo foi
**reconferido no código nesta data**, não copiado da auditoria; onde a
verificação mudou o achado, está dito.

Cada item traz alternativas reais com o preço de cada uma. Onde há recomendação,
ela vem com o motivo — e onde a decisão depende de um número que ninguém mediu,
isso está escrito em vez de um palpite.

---

### Bloco 1 — consistência dos dois índices (A2, A3, A7, + M7)

Os três são **o mesmo defeito visto de três ângulos**, e é por isso que o plano
os junta.

**Mecanismo, conferido em `internal/index/update.go:24-50`:**

```go
func (ix *Index) Replace(ctx, v, path) error {
	ix.mu.Lock()
	defer ix.mu.Unlock()                       // <- lock do inicio ao fim

	chaves := ix.removeContributionsLocked(path)   // <- remove ANTES do I/O
	atomic.AddUint64(&ix.generation, 1)

	abs := v.Abs(path)
	info, err := os.Stat(abs)                  // <- I/O sob lock exclusivo
	...
```

- **A2**: `os.Stat`, `IsCloudOnly` e a leitura inteira acontecem com o lock
  exclusivo tomado. Todo `Get`/`List`/`Backlinks`/`TotalSize` espera atrás
  disso a cada evento do watcher. Em OneDrive hidratando, é espera de **rede**
  disfarçada de disco. Contraria o que `index.go:17` declara ("leituras são
  concorrentes").
- **A3**: as contribuições antigas saem **antes** de o arquivo ser lido. Se
  `ReadAll` falha, a nota fica fora dos metadados sem republish; `apply.go`,
  vendo erro, pula `searchInv.Update`. Busca mantém o documento velho,
  metadados não — e `service.Search` descarta posting sem metadado, então **a
  nota some das respostas** até o próximo evento, reconciliação ou boot.
- **A7**: o atalho de mtime/tamanho do `Apply` consulta só `idx.Get`. A
  `Reconcile` já aprendeu a condição certa — conferido em `overflow.go:58`:
  `if n, ok := idx.Get(e.Path); ok && (inv == nil || inv.HasDoc(...))`. O
  `Apply` não a espelhou. Um `searchInv.Update` falho deixa metadados em dia e
  posting ausente, e **todo evento seguinte com mtime/tamanho iguais cai no
  `continue`** — e o OneDrive re-emite eventos de arquivos intocados como
  rotina.

#### Alternativa 1 — redesenho em duas fases (I/O fora do lock)

Fase 1 sem lock: `Stat`, `IsCloudOnly`, `ReadAll`, parse. Fase 2 sob lock:
construir a `Note`, remover contribuições antigas, publicar as novas,
reprocessar citantes.

- **Vantagem:** cura A2 e A3 de uma vez, pela raiz. Remover só depois de a
  leitura ter dado certo elimina a janela de A3 **por construção**, não por
  tratamento de erro. E **M7 sai junto**: publicar exclusivamente via
  `publishNoteLocked` apaga as seis derivações refeitas à mão em
  `update.go:111-125` — o padrão que gerou o bug `[[STJ]]`.
- **Desvantagem:** abre uma janela nova entre as fases. O arquivo pode sumir ou
  mudar entre a leitura e a publicação, e isso **exige política explícita**
  (re-checar existência sob lock; se sumiu, tratar como `Remove`). É a parte
  que precisa de teste, não de comentário.
- **Risco:** `Replace` é o ponto de chegada do watcher e roda em todo evento.
  Errar a janela troca um defeito raro por um frequente.

#### Alternativa 2 — só reordenar (remover depois de ler), mantendo o lock

Mantém `Lock()` no topo, mas move `removeContributionsLocked` para depois da
leitura bem-sucedida.

- **Vantagem:** conserta A3 com um diff pequeno e de baixo risco. Nenhuma
  janela nova. Cabe num modelo barato.
- **Desvantagem:** **não** conserta A2 — o I/O continua sob lock exclusivo, e a
  contenção que a auditoria descreve permanece inteira.
- **Quando escolher:** se você quiser fechar a divergência (que é dado
  invisível para o usuário) sem abrir a frente de concorrência agora.

#### Alternativa 3 — RWMutex com upgrade / cópia-na-escrita

Índice imutável trocado por ponteiro atômico; escritores constroem a versão
nova fora do lock.

- **Vantagem:** leitura nunca bloqueia. Resolve A2 no limite.
- **Desvantagem:** reescreve `internal/index` inteiro, e o índice tem
  `backlinks`, `byName`, `byAlias`, `lowerPath` e `citantesPorNome` —
  copiar tudo a cada evento troca contenção por alocação. **Sem baseline, isso
  é troca no escuro**, e a regra da casa proíbe.
- **Veredito:** fora de escopo hoje. Só faz sentido se a medição mostrar que a
  contenção de A2 é real no cofre do dono — e ela **não foi medida**.

#### Sobre A7, separadamente

Uma linha: `apply.go:85-92` passa a exigir
`(searchInv == nil || searchInv.HasDoc(string(path)))` no atalho, espelhando o
`overflow.go:58`.

- **Vantagem:** trivial, e é "uma conta por regra" — hoje há duas cópias da
  condição, e a que está errada é a mais usada.
- **Desvantagem:** nenhuma identificada. O custo é uma consulta em memória.
- **Alternativa descartada:** extrair a condição para uma função compartilhada
  entre `Apply` e `Reconcile`. Mais correto em princípio, mas os dois laços têm
  formas diferentes (um itera evento, outro entrada de varredura), e forçar uma
  assinatura comum acrescenta mais do que remove.

**Recomendação do bloco:** A7 primeiro (isolado, trivial, e sozinho já corta o
modo de falha mais provável em OneDrive). Depois Alternativa 1 para A2+A3+M7,
com a política da janela escrita no brief e testada.

---

### Bloco 2 — escrita que reporta sucesso sem ter feito (A1, + B5, B17, B6)

**Mecanismo, conferido em `internal/service/write.go`:** a ordem é
`WriteAtomic(absRef, ...)` para cada citante (`:581`), depois
`WriteAtomic(absTo, fromRaw)` (`:626`), depois `_ = os.Remove(absFrom)`
(`:636`).

Duas consequências independentes:

1. `_ = os.Remove` — remove falhando (sharing violation do Obsidian é rotina no
   Windows) ⇒ **sucesso reportado com a nota duplicada** nos dois caminhos.
2. Os citantes são reescritos **antes** de o corpo se mover. Falhando a cópia,
   todos os links apontam para destino inexistente, **persistidos em disco**,
   sem compensação — com o `raw` original em mãos no loop.

#### Alternativa 1 — mover o corpo primeiro, citantes depois

`os.Rename(absFrom, absTo)` (mesmo volume é o caso comum; fallback
`WriteAtomic` + `os.Remove` **com o erro conferido**). Só depois reescrever
citantes.

- **Vantagem:** elimina os dois defeitos com uma reordenação. `os.Rename` é
  atômico no mesmo volume, então "nota duplicada" deixa de ser possível ali.
- **Desvantagem:** inverte quem fica inconsistente quando a segunda etapa falha
  — agora o corpo moveu e alguns citantes ficaram apontando para o caminho
  antigo. É **menos grave** (link quebrado é visível e recuperável; nota
  duplicada é silenciosa), mas não é nada.
- **Mitigação:** registrar no resultado quais citantes já foram reescritos,
  com diff, para compensação consciente.

#### Alternativa 2 — rollback automático dos citantes

O `raw` original de cada citante já está em mãos no loop; guardar e reescrever
de volta em caso de falha.

- **Vantagem:** o move vira tudo-ou-nada do ponto de vista do usuário.
- **Desvantagem:** **o rollback também pode falhar**, e aí o estado é pior que
  o de qualquer uma das alternativas — parcialmente revertido, sem registro. E
  guardar o `raw` de N citantes é memória proporcional ao grafo; uma nota-hub
  com centenas de citantes é caso real.
- **Veredito:** só sobre a Alternativa 1, e como opção, não como padrão.

#### Alternativa 3 — journal de operação

Gravar a intenção antes, aplicar, apagar o journal.

- **Vantagem:** recuperável até depois de crash.
- **Desvantagem:** formato novo, ponto de recuperação novo no boot, e uma
  classe inteira de defeitos nova. Desproporcional para uma tool de move.

**Relacionados que se consertam no mesmo movimento:**

- **B5** — `note_delete to_trash` descarta o erro do remove (`:761`). Nota
  travada pelo Obsidian ⇒ `Deleted=true` com a nota em **dois** lugares.
  Mesmo remédio: conferir o erro e mapear para `FILE_LOCKED`, dizendo que a
  cópia na lixeira existe.
- **B17** — dry-run engole erro de leitura (`fromRaw, _ :=`, `:514`) e apresenta
  **diff vazio como resultado**. Uma linha.
- **B6** — travas do move em ordem fixa `from→to`. Ordenar por chave
  normalizada (menor primeiro) elimina o deadlock AB-BA **de graça**, dentro da
  mesma tarefa.

**Recomendação:** Alternativa 1 + B5 + B17 + B6 numa tarefa só. Rollback
(Alternativa 2) fica como decisão sua, separada.

---

### Bloco 3 — daemon e IPC (A4, A5, + M8, M9)

**Isto saiu do hipotético.** Até 2026-08-26 o daemon não subia na máquina do
dono; hoje há três vivos. A superfície passou a ser exercitada de verdade.

**A5, conferido em `internal/ipc/ipc.go:104`:** `cleanupSocketFile(path)` roda
**incondicionalmente** antes do `net.Listen`. Sem dial de prova, sem `pidVivo`,
e o subcomando `daemon` não participa do lock de `EnsureStarted`. Um segundo
daemon remove o socket do vivo e binda no nome: o primeiro fica inalcançável, e
**duas instâncias gravam concorrentemente no mesmo cache de busca** — corrupção
por escrita intercalada.

**A4, conferido em `internal/daemon/daemon.go:145-155`:**

```go
conn, err := d.ln.Accept()
if err != nil {
	if ctx.Err() != nil { return }        // encerramento normal
	d.log.Warn("aceitar conexao ... falhou", "err", err)
	return                                 // <- morre para sempre
}
```

Processo segue vivo, socket bound, ticker rodando: dials conectam, ninguém
aceita.

#### A5 — Alternativa 1: dial de prova antes do unlink

Antes de `cleanupSocketFile`, tentar `DialAndHandshake` com prazo curto.
Sucesso ⇒ erro "daemon já ativo", sem tocar no arquivo. Falha ⇒ unlink e segue.

- **Vantagem:** funciona nos três SOs sem arquivo extra, e o critério é
  **comportamental** — que é o que a medição de hoje exige. Errno **não serve**:
  medido, arquivo comum, socket órfão e caminho inexistente devolvem os três
  `10061`, e o que apareceu em produção foi `10022`.
- **Desvantagem:** custa um dial (~250 ms de prazo) em toda partida de daemon.
  Medido: um socket que responde faz ida e volta em 25,7 µs, então o custo real
  só aparece quando **não** há daemon — que é justamente o caso comum.
- **Mitigação:** prazo curto, e o caminho "arquivo ausente" nem chega a discar.

#### A5 — Alternativa 2: PID ao lado do socket + `pidVivo`

- **Vantagem:** decide sem I/O de rede; `daemon.PIDVivo` já existe e já paga a
  armadilha do Windows (PID consultável depois da morte).
- **Desvantagem:** arquivo a mais para ficar órfão — e órfão de arquivo de
  estado é exatamente o defeito que se está consertando. Além disso, PID vivo
  **não prova** que o processo está servindo aquele socket.
- **Veredito:** complementar, não substituto.

#### A5 — Alternativa 3: as duas, em camadas

Dial de prova decide; o PID entra só no diagnóstico (`doctor`).

- **Vantagem:** o `doctor` da Task 125 já lê locks órfãos com PID morto — o
  investimento está feito.
- **Desvantagem:** duas fontes de verdade sobre "há daemon vivo", e a regra da
  casa é uma conta por regra. **Só aceitável se o PID for explicitamente
  informativo**, nunca decisório.

#### A4 — Alternativa 1: classificar erro e continuar com backoff

Sair só em `net.ErrClosed` ou `ctx.Err() != nil`; qualquer outro ⇒ log com
contador, backoff (50 ms → teto 1 s) e `continue`.

- **Vantagem:** `EMFILE`/`ENFILE` é transitório e clássico; hoje ele mata o
  daemon em silêncio.
- **Desvantagem:** um erro **permanente** não classificado vira laço quente com
  backoff — consome CPU para sempre em vez de morrer. Precisa de teto de
  tentativas consecutivas, ou o conserto troca "daemon surdo" por "daemon
  girando".

#### A4 — Alternativa 2: morrer, mas alto

Manter o `return`, mas com `log.Error` e código de saída diferente de zero,
para a ponte re-spawnar.

- **Vantagem:** diff mínimo, e compõe com a Task 124 (que já fez todo caminho
  de saída logar).
- **Desvantagem:** o daemon morre em erro transitório que se resolveria sozinho,
  e cada morte custa uma reconstrução de índice a quem reconectar.

**Recomendação do bloco:** A5 pela Alternativa 1 (o critério comportamental é o
que a medição sustenta), A4 pela Alternativa 1 **com teto de tentativas
consecutivas** — sem o teto ela é meio conserto.

**Relacionados:**

- **M8** — `CloseWrite` é declarado, exigido no handshake e **nunca chamado**;
  no EOF a ponte fecha a conexão inteira, descartando resposta em voo. Duas
  saídas: usar no shutdown, ou **tirar do tipo e do handshake**. Contrato morto
  é pior que contrato ausente, e a segunda opção é legítima.
- **M9** — `--max-results` é no-op silencioso no modo daemon: não é encaminhado
  no spawn nem entra no handshake, então vale o cfg do **primeiro** daemon.
  Duas saídas: entrar no `HandshakeConfig` (recusando divergência, como
  `ReadOnly`), ou **documentar** que o primeiro daemon fixa o parâmetro. A
  segunda é honesta e mais barata; a primeira é o que a simetria pede. Escolher
  uma e registrar.

---

### Bloco 4 — contrato (A6, A8)

Os dois já têm decisão sua tomada; ficam aqui as alternativas para o registro.

**A6 — primeira `vault_search` bloqueia sem prazo.** `cargaUnica.fazer` segura
o mutex durante a carga; concorrentes esperam em `mu.Lock()` puro, **sem
`select` em `ctx.Done()`**. Cache frio ou corrompido ⇒ tokenização completa do
cofre sem resposta nem erro.

- **Decidido:** porta `chan struct{}` no lugar do mutex, com orçamento no
  primeiro chamador devolvendo `INDEX_BUILDING` com `Retryable`.
- **Vantagem:** prazo do host respeitado, código de erro honesto, e a carga
  continua em segundo plano.
- **Desvantagem:** exige desacoplar a carga da vida do primeiro chamador —
  goroutine dona, referenciada pela porta. É a parte de projeto, e é onde um
  erro vira carga órfã ou carga dupla.
- **Alternativa mais barata, não escolhida:** só tornar a espera cancelável
  (concorrentes respeitam ctx; o primeiro segue bloqueando). Menor risco, mas
  deixa o caso pior — o primeiro chamador — sem conserto.

**A8 — `Backlink.Context` é sempre `""`.** Conferido: os três construtores
gravam literal, e `note_metadata.backlinks` serializa direto ao host, contra
promessa explícita de `docs/TOOLS.md:164`.

- **Alternativa 1 — implementar.** O offset `Start`/`End` do link e o corpo já
  lido no parse permitem recortar ±N bytes.
  - *Vantagem:* backlink sem contexto obriga o modelo a abrir a origem — é uma
    chamada a mais por backlink, e o campo existe justamente para evitá-la.
  - *Desvantagem:* **exige bump do formato do cache de metadados**, e aí é
    obrigatório usar a constante única do **B11** — hoje são duas constantes
    independentes guardando o mesmo portão (`persist.go:22` ×
    `persist_codec.go:44`), e subir uma sem a outra faz o leitor recusar todo
    save novo, com rebuild a cada boot e **sem log**.
- **Alternativa 2 — remover o campo e a promessa.**
  - *Vantagem:* honesto, imediato, zero formato novo.
  - *Desvantagem:* perde-se uma economia real de chamadas.

**Recomendação:** implementar, **junto** do B11, numa tarefa só — porque o bump
de formato é o gatilho que torna o B11 perigoso, e consertar os dois separados
deixa a janela aberta exatamente no commit que a usa.

---

## Decisões fechadas em 2026-08-25 (dono do projeto) — não re-litigar

As nove pendências que este relatório listava como bloqueantes foram decididas. O que
está aqui é o contrato; divergir dele exige dado novo, não preferência.

| # | Decisão | O que isso manda fazer |
|---|---|---|
| **P5** | Remover `Hits`; `results` fica | Apagar o campo `Hits` de `SearchResponse` (`service/search.go:97`) e as três atribuições (`:176,220,315,338,370`). Atualizar `TOOLS.md:62`, que hoje diz *"contendo `results` (e `hits`)"*. Grep em goldens e CLI antes — reprovar a suíte por um campo que ninguém lê seria custo sem ganho. |
| **C2** | Flag agora, `os.Root` depois | **(F)** `--follow-symlinks`, padrão **recusa**, com o skip visível via `RecordSkip`/`doctor` — o usuário recebe a informação em vez de a nota sumir. **(L)** migrar `Open`/`ReadRange`/`ReadAll` (`vault.go:114,124,163`), `WriteAtomic` e o sweep para `os.OpenRoot` como tarefa de arquitetura própria, com AD-10 e medição do custo de syscall no boot. Não fazer (L) dentro da tarefa de (F). |
| **M15** | Não mexer; documentar a limitação | A tolerância de gap em bytes está **recusada**: trocaria falso-negativo silencioso por falso-positivo silencioso, e o próprio plano admite que não dá para distinguir sem re-tokenizar. Registrar em `TOOLS.md` que frase exata casa adjacência por byte e não atravessa pontuação nem CRLF. O ordinal de token (Oportunidade 2) segue aberto como trabalho estrutural. |
| **Oport. 1** | Medir antes de decidir | Baseline + perfil de alocação do caminho BM25, para saber que fração do custo de `vault_search` está nos mapas que os IDs densos eliminariam. **P1, P2 e P3 ficam congelados até o número existir** — fazê-los antes é arriscar jogar os três fora. |
| **A6** | Orçamento no primeiro chamador | Porta `chan struct{}` no lugar do mutex de `cargaUnica`; concorrentes fazem `select` em `ctx.Done()`; o primeiro chamador recebe `INDEX_BUILDING` com `Retryable` após orçamento em vez de bloquear minutos. Exige desacoplar `f()` da vida do primeiro chamador — goroutine dona da carga. Preservar a retentativa pós-falha, que já é testada. |
| **B7** | Fail-open **instrumentado** | Manter `return false` em erro de `GetFileAttributes` (`cloud_windows.go:23-26`), mas com contador e log por motivo. A política definitiva se decide com o número: hoje ninguém sabe se o ramo dispara uma vez por ano ou mil vezes por boot, e sem isso a escolha é palpite. |
| **M17 + B9** | `--label`, com fallback para o basename | Uma decisão só para os dois itens. `instructions` e `vault_info` usam o rótulo; sem flag, `filepath.Base(vaultPath)`. **Caminho absoluto nunca vai no fio** — nem no `instructions`, nem na `vault_info`, nem em mensagem de erro host-facing (B9); absoluto só no `slog`. | <!-- check-doc-refs: ignore vault_info -- tool PROPOSTA pelo M17; a ausencia dela e o achado -->
| **Daemon** | Medir o cenário real antes | Rodar N cofres × M sessões na configuração atual antes de considerar (a), (b) ou (c). As decisões D-M7-* seguem fechadas até o dado existir. |
| **B18** | Não era decisão | A redação vigente já estava em `PRD.md:265`, `ARCHITECTURE.md:654` e `OPERACAO.md:205`, e o `tools/netcheck` já a implementa. Quem divergia era o `CLAUDE.md` e o `AGENTS.md`, **corrigidos em 2026-08-25**. |

Consequência de acoplamento: com M15 decidido por (c), o bump de `CacheFormatVersion`
perde o seu motivo principal. O `Backlink.Context` (A8) passa a ser o único candidato a
justificar troca de formato do cache de METADADOS — decidir lá, não aqui, e usar a
constante única do B11.

---

## Verificado e saudável (para não re-auditar sem motivo)

stdout purity em serve (único `os.Stdout` é o JSON-RPC e as cópias da ponte); SDK
confinado a `mcpsrv`; zero `context.Background()` em produção sob `internal/`; shutdown
com `context.WithoutCancel` e orçamentos por etapa; vigília do pai correta nos dois
SOs (exitTime/ppid capturado); `ctx.Canceled` tratado como encerramento normal nos três
loops; ociosidade do daemon resetada sob mutex; FDs fechados nos caminhos de erro de
dial/handshake/mmap; pool de transformers com Get/Put/Reset; build tags de race
presentes; codec de busca formato 6 com portão de versão; design base/delta com sombra
e contadores exatos; cache de trechos LRU com chave por hash+ocorrência+teto, nil =
desligado; `alias_collisions` honesto; backlinks com delta em Replace/Remove;
`citantesPorNome` cobrindo links quebrados; classify fonte única via `vault.Classify`;
round-trip completo do codec de metadados (nil×vazio distinguível); `d == nil` tratado
como falha de raiz no walk e no sweep; console ASCII com cor pelo destino; doctor
halting com exit code correto; companheiros `*Set` em todos os subcomandos; reconciliação
por overflow repara os dois índices e seu teste desconecta o caminho principal; Add de
watch seguido de varredura; debounce com tique único e zero recusado; contadores
desdobrados por motivo; correlação de renames nunca abre anexo nem nuvem no pré-filtro.
