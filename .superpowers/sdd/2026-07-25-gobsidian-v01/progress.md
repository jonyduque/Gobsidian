# gobsidian v0.1 — progresso SDD

Plano: docs/superpowers/plans/2026-07-25-gobsidian-v01.md
Baseline: db91011 (docs only)

Task 1: complete (commits db91011..5dbce1c, review clean)
Task 2: complete (commits 5dbce1c..b14f81d, review clean apos 2 fix passes)
Task 3: complete (commits b14f81d..0534b5f, review clean apos 1 fix pass)
Task 4: complete (commits 0534b5f..6a5c268, review Approved; 1 Important doc-only fechado depois)
Task 5: complete (commits 12d26f7..f1fb65a, review clean apos 2 fix passes)
Task 6: complete (commits 1a7786e..3453ddc, review Approved; 2 Important plan-mandated fechados)
Task 7: complete (commits 3453ddc..ff825e3, review Approved apos 3 fix passes)
Task 8: complete (commits ff825e3..6447833, review Approved apos 2 fix passes)
Task 9: complete (commits 38e6a74..18be197, 2 revisoes completas + 2 fix passes)
  ATENCAO: o ultimo fix pass NAO teve revisao fresca. Fechado por evidencia direta
  (20 sessoes limpas exit 0, mutacao %w->%s reprovando o teste novo). A revisao
  final de branch DEVE cobrir cmd/gobsidian/serve.go e internal/mcpsrv/convert.go.
Task 10: complete (commits 18be197..87201d1, 1 revisao completa + 1 fix pass)
  ATENCAO: fix pass sem revisao fresca. Fechado por evidencia (1 varredura medida,
  mutacao de marcador). Revisao final DEVE cobrir internal/doctor/.
Task 11: complete (commit 757d3b0, tag m0-lifecycle) — 100 ciclos, zero orfaos

Task 12: complete (commits d0143bc..dc786f3, 1 revisao completa + 1 fix pass + re-revisao Approved)

Task 13: complete (commits dc786f3..44512a7, 1 revisao completa + 2 fix passes + re-revisao Approved)

Task 14: complete (commits 44512a7..0d418db, 1 revisao completa + 1 fix pass)
  ATENCAO: o fix pass NAO teve revisao fresca. Fechado por evidencia direta (3 mutantes
  aplicados e capturados, parse de [[[a]] b](d.md) e de ![alt](x.png) reportados).
  A revisao final de branch DEVE cobrir internal/parser/ext_wikilink.go e ast.go.

Task 15/16/17: complete (commits 0d418db..f9e74d6, 1 revisao COMBINADA das tres +
  2 fix passes + re-revisao Approved com worktree isolado e 3 mutacoes independentes)

Task 27: complete (commit 64a2632, fsnotify facade with unified vault relevance filtering, review Approved)
Task 28: complete (commit 6fdecf8, single-ticker debouncer with dirty set coalescence, review Approved)
  SHA corrigido em 2026-08-02: o que estava registrado aqui nunca existiu neste
  repositorio — mesma classe do defeito ja anotado na Task 31. Achado pelo
  audit_reports.ps1 na conferencia que seguiu a reescrita de historico, e
  confirmado contra o bundle anterior a ela: o defeito e antigo, nao veio da
  reescrita.
Task 29: complete (commit 952b43b, real change verification with mtime+size diffing and index application, review Approved)
Task 30: complete (commit 3496610, reconciliation recovery on fsnotify overflow event, review Approved)
Task 31: complete (commits 1d4a65c..bce5fa1, xxhash rename correlation for fast path updates and backlink preservation, review Approved)
    Corrigido em 2026-07-29: a linha registrava 1d4a65c, que e a BASE da tarefa
    (o commit ANTERIOR a ela) e o commit final da Task 30. O trabalho da 31 e
    bce5fa1, e ate esta correcao nenhuma linha do ledger o referenciava. Antes
    disso a linha apontava para um SHA que nao existia no repositorio (nao
    reproduzido aqui de proposito: escrever um SHA morto no ledger faz o
    audit_reports.ps1 acusa-lo para sempre, e ruido permanente treina a ignorar
    a ferramenta). Um SHA que existe nao e um SHA que confere: o script ganhou
    SHA-NAO-CONFERE e SHA-REUSADO por causa deste caso.
Task 32: complete (commits 9dc9164..083fdeb, expose watcher metrics in vault_stats tool, review Approved)
Task 33: complete (commits 1ac2cbd..ab65c53, rename correlation single pass, zero asset/cloud reads, review Approved)
Task 34: complete (commits 1275c87..5afbcce, reconciliation tests that actually lose events, review Approved)
Task 35: complete (commits 02217e5..4f18c24, normalize byAlias keys so Remove clears what Build wrote, review Approved)
Task 36: complete (commits 0c6d025..c7161f5, MoveNote refreshes stat and matches Remove+Replace, review Approved)
Task 37: complete (commits b8af2d9..41c50a1, per-reason drop counters, coalesced count, and a real active flag, review Approved)
Task 38: complete (commits ba9a859..262cee1, reject debounce-ms below 1 instead of silently defaulting, review Approved)
Task 39: complete (commits 303fcd7..af86e83, cover handle release, channel close, and dynamic subdirectory watch, review Approved)
Task 40: complete (commits 4260436..7838773, stop swallowing Add errors, use errors.Is, drop committed scratch, review Approved)
Task 41: complete (commits caeb903..8657100, vault_stats reflects a note created while the server runs, review Approved)
Task 42: complete (commits e2c38e1..8ebeb6b, rewrite reports with real evidence; fix ledger, review Approved)
    Corrigido em 2026-07-29: registrava e2c38e1..e2c38e1, um intervalo que nao
    contem commit nenhum e cuja ponta e a entrada de ledger da Task 41. O
    trabalho da 42 e 34a37ba mais 8ebeb6b.

Task 43: complete (commits 5e834d8..a503d08, accept trailing whitespace on frontmatter delimiter, review Approved)
Task 44: complete (commits a931cb1..5e0988b, portuguese analyzer with dual indexing, review Approved)
Task 45: complete (commits 6efe9a0..e89e7f0, inverted index with incremental update, review Approved)
Task 46: complete (commits aa15982..b8dfb50, BM25 ranking with field weights, review Approved)
Task 47: complete (commits 229dfb9..c861ce1, snippets with term highlight, review Approved)
Task 48: complete (commits 71e9e05..5990011, vault_search tool with filters and phrase queries, review Approved)
Task 49: complete (commits 8faf689..0982e1e, on-disk cache with version header, review Approved)
Task 50: complete (commits ed1863d..efb10bc, close M3 with measured numbers and Q3 decision, review Approved)
Task 51: complete (commits 566e2c7..f66ec20, the inverted index never reached the watcher, review Approved)
Task 52: complete (commits 5a5f3ea..e9b1b85, measure Q3 on a corpus of distinct paths and a real p95, review Approved)
Task 53: complete (commits 79b39e1..43aa303, stop swallowing errors and name the tests for what they check, review Approved)
Task 54: complete (commits 5188726..e7825b3, per-path write lock, review Approved)
Task 55: complete (commits c135ee0..458a778, atomic write with fsync and rename retry, review Approved)
Task 56: complete (commits ebc647f..53ad63c, line-based Myers diff, review Approved)
Task 57: complete (commits 8a40a92..5d0f7c8, patch and append by heading, preserving EOL and BOM, review Approved)
- Task 58: Substituir Bloco por ^id | base 4341348 | review 4341348..6b1441d | review Approved; 0 Important doc-only fechados | commit 6b1441d ("feat(writer): replace block by id")
- Task 59: Write Tools (`note_create`, `note_append`, `note_patch`) | base 38e844f | review 38e844f..71e3653 | review Approved; 0 Important doc-only fechados | commit 71e3653 ("feat(mcpsrv): write tools with dry_run and expected_hash")
- Task 60: --read-only verificado no ListTools | base a57cd3d | review a57cd3d..7f8e0da | review Approved; 0 Important doc-only fechados | commit 7f8e0da ("feat(mcpsrv): write tools absent from ListTools under read-only")
- Task 61: A lacuna do RNF-04 — busca por frase | base 7aedd83 | review 7aedd83..1e0dfad | review Approved; 0 Important doc-only fechados | commit 1e0dfad ("perf(search): bring exact-phrase p95 within RNF-04")
- Task 62: Fechamento do M4 | base 5b824dc | review 5b824dc..c58f87b | review Approved; 0 Important doc-only fechados | commit c58f87b ("docs: close M4 with the RNF-11 gate output")
- Task 63: Offsets para LinkMarkdown no parser | base 5617075 | review 5617075..ccf1a4c | review Approved; 0 Important doc-only fechados | commit ccf1a4c ("feat(parser): byte offsets for markdown links and embeds")
- Task 64: internal/writer/linkrewrite.go | base ccf1a4c | review ccf1a4c..99d32d1 | review Approved; 0 Important doc-only fechados | commit 99d32d1 ("feat(writer): faithful link rewriting from Link.Raw")
- Task 65: Tool note_move | base 99d32d1 | review 99d32d1..02264d8 | review Approved; 0 Important doc-only fechados | commit 02264d8 ("feat(mcpsrv): note_move with faithful link rewriting")
- Task 66: Tool note_delete | base c50f8f3 | review c50f8f3..5a4f519 | review Approved; 0 Important doc-only fechados | commit 5a4f519 ("feat(mcpsrv): note_delete with trash and prior broken-link report")
- Task 67: Ancoras quebradas no relatorio de impacto | base d8cf22a | review d8cf22a..262589b | review Approved; 0 Important doc-only fechados | commit 262589b ("feat(service): broken anchors in the move and delete impact report")
- Task 68: Fechamento do M5 | base f64adbd | review f64adbd..c06ec08 | review Approved; 0 Important doc-only fechados | commit c06ec08 ("docs: close M5 and align RF-35 with the implemented contract")
- Task 75: Subcomandos index, search, inspect | base f0ca554 | review f0ca554..ac91156 | review Approved; 0 Important doc-only fechados | commit ac91156 ("feat(cmd): index, search and inspect subcommands (RF-52)")
- Task 69: Os quatro parametros de schema ignorados | base 62d3fef | review 62d3fef..d15fddf | review Approved; 0 Important doc-only fechados | commit d15fddf ("fix(mcpsrv,service): honour the four schema params the handlers dropped")
- Task 74: netcheck como analisador de go vet | base 8c87aa9 | review 8c87aa9..daab822 | review Approved; 0 Important doc-only fechados | commit daab822 ("ci: semantic net-import check as a go vet analyzer")
- Task 70: Gerador deterministico de cofre de 5.000 notas | base 9aa2495 | review 9aa2495..c0cfd9a | review Approved; 0 Important doc-only fechados | commit c0cfd9a ("tooling: deterministic 5000-note synthetic vault generator")
- Task 71: RNF-01, RNF-02, RNF-04 e RNF-07 medidos a 5.000 | base e23c1a5 | review e23c1a5..2da8b90 | review Approved; 0 Important doc-only fechados | commit 2da8b90 ("docs(operacao): measure RNF-01, RNF-02, RNF-04 and RNF-07 at 5000 notes")
- Task 72: A folga fina do RNF-04 em limit: 200 | base 7f90d87 | review 7f90d87..6cdc79e | review Approved; 0 Important doc-only fechados | commit 6cdc79e ("perf(search): cut snippet I/O cost for large result limits")
- Task 72 REABERTA e corrigida pelo modelo principal | base 6cdc79e | commit 22836a6 ("fix(service): a cancelled search must not return a partial page as success") | relatorio .superpowers/sdd/task-72-review-fix-report.md
  A revisao do lote M6 reprovou a 72. Cinco defeitos, cada um reproduzido: ctx cancelado
  devolvia 24 a 99 de 200 resultados com err == nil (regressao, o caminho sequencial
  anterior devolvia os 200); a prova de mutacao colada no relatorio da 72 e impossivel,
  porque mutate.ps1 roda com -race e a assercao vive atras de !raceEnabled — rodada de
  novo, ela sai MUTATE_EXIT=1; o teto de 60 ms nao era cobrado por etapa nenhuma;
  TestRNF04SnippetParity sobrevivia a inversao da ordem dos 200 resultados; e
  docs/OPERACAO.md nao foi atualizado, contra exigencia em negrito do plano.
  maxSnippetWorkers passou de 16 para 8 por varredura medida. tools/ entrou em $Alvos.
  RNF-04 para limit: 200 a 5.000 notas: 181,25 ms, NAO atingido (teto 100 ms).
  RNF-07 a 5.000 notas: RSS 67,08 MB quente e 112,96 MB frio, NAO atingido (teto 60 MB);
  a Task 71 o registrara como OK medindo runtime.MemStats.Alloc, que nao e RSS.
- Task 73: bench.yml no CI com deteccao de regressao | base 8679a70 | commits 718995f (workflow+comparador+benchmarks), 74c4ecd (1a referencia), be2d2b6 (referencia por mediana) | relatorio .superpowers/sdd/task-73-report.md
  Nao havia benchmark nenhum no repositorio; os seis foram escritos junto.
  Gate provado NO CI: run 30762162779 reprovou com +71,5%, +521,5% e +126,0%
  sob 120 ms injetados em Search. Duas rodadas limpas verdes: 30761791215 e
  30761852416. A primeira referencia (amostra unica) punha SearchFraseExata a
  0,7 ponto do gate em duas rodadas SEM mudanca de codigo; refeita sobre a
  mediana de tres rodadas, o pior desvio positivo caiu para +9,6%.
- ANTES da 73, o CI estava VERMELHO desde 2026-07-28, em todos os pushes.
  Cinco defeitos, commits c6a0baa e 0a8ba36, todos invisiveis para verify.ps1
  por ele so rodar em Windows: caminhos "C:\test\vault" fixos em filter_test;
  chmod 0400 num arquivo, que no Unix nao impede rename por cima; $env:TEMP
  inexistente em Linux no check_net.ps1 do vettool da Task 74; teto de tempo do
  TestBM25KernelLatency cobrado sob -race com numero medido na maquina errada;
  e TestWatcher_Burst com prazo fixo de 10 s num teste de convergencia.
  TestRun_OverflowSchedulesExactlyOne mantinha uma COPIA do laco de producao
  dentro do teste e afirmava sobre a reimplementacao. CI verde em 30761531221.
- Task 76: As duas lacunas de teste que atravessaram todos os marcos | base 458b03f | commits 12ad94f, 23ecc2e, 38a0cca | relatorio .superpowers/sdd/task-76-report.md
  O harness de orfaos tinha UM cenario e o EOF vencia sempre (100/100 nas duas
  rodadas anteriores). Agora sao tres, um por mecanismo, cada um desconectando os
  outros dois: stdin-eof, parent-death (keeper segura o stdin, o host morre) e
  signal (nada morre; stdin e CONIN$, servidor em CREATE_NEW_PROCESS_GROUP).
  O harness reprova se o reason= nao for o do mecanismo que o cenario nomeia.
  Provas por remocao: sem parent-watch, parent-death deixa 3/3 orfaos; sem
  watchSignals, signal deixa 3/3 sem reason= (o processo morre pelo handler padrao
  do SO, que e o falso-verde que o gate bloqueia). Com parent-watch removido, o
  cenario ANTIGO continuava verde — a medida exata da lacuna.
  100 ciclos por cenario, local e no CI: 100x o motivo certo, zero orfaos. O job
  orphans subiu de 15m11s para 49m8s.
  RNF-32: teste explicito de symlink, provado por mutacao (vault.New(root) ->
  vault.New(base) faz as duas asserçoes dispararem) e com o ramo de skip provado
  por injecao de ERROR_PRIVILEGE_NOT_HELD.
  Fora do escopo, corrigido junto: TestE2E_NoteMoveIsReflectedBySearchAndGraph
  tinha prazo fixo de 10 s e piscava no runner macOS; e teste de convergencia,
  passou para 60 s.
- Task 77: Fechamento do M6 e v1.0.0 | base 98b61a0 | commit fbd74ed | relatorio .superpowers/sdd/task-77-report.md
  Tabela de RNFs de docs/OPERACAO.md passou de 5 dos 22 para os 22, cada um com
  numero medido ou "nao medido". RNF-03 (p95 344,97 us) e RNF-05 (p95 533,68 us)
  medidos pela primeira vez e ATINGIDOS; RNF-06 medido em 20,35 ms de mediana
  contra alvo de 20 ms, NAO ATINGIDO. RNF-08 e RNF-09 nao medidos.
  TRES RNFs nao atingidos, todos medidos: RNF-04 (181,25 ms para limit: 200 a
  5.000 notas contra 100 ms), RNF-06 e RNF-07 (67,08 MB quente, 112,96 MB frio,
  contra 60 MB).
  Portao: verify.ps1 10/10; test_orphans 100 ciclos nos TRES cenarios, cada um
  com 100x o motivo do seu mecanismo e zero orfaos; check_tool_params limpo.
  audit_reports (40) e check_briefs (65) saem 1, e nenhum achado e das Tasks
  69-77 — todos vem de relatorios e briefs das Tasks 1 a 68.
  Tag v1.0.0 criada LOCALMENTE em ac4c2cf, o commit deste ledger.
  NAO publicada: nem a tag no remoto, nem release no GitHub. Binarios dos tres
  alvos em dist/, com versao embutida e SHA-256 no relatorio; so o de Windows
  foi executado. README NAO foi tocado — segue falando de v0.1, que e a unica
  afirmacao verdadeira hoje. A sequencia de publicacao esta no relatorio.

=== v1.0.1 PUBLICADA, 2026-08-03 ===
Tag em a875c57, release com quatro assets. Corrige o boot que impedia
conexao em cofres grandes.
Um segundo defeito foi achado ANTES do release, testando o encerramento no
meio da construcao: o encerramento levava ate 10 s, contra os 8 s que o
harness de orfaos usa para declarar orfao. runServe faz wg.Wait() DEPOIS de
lifecycle.Shutdown, entao a gravacao do cache parcial ficava fora de todo
orcamento. A primeira tentativa de conserto envolveu a gravacao num
WithTimeout e estava errada: SaveInvertedCache confere o context so na
ENTRADA (persist.go:51), entao o teto limitava COMECAR e nao terminar --
orcamento decorativo. O encerramento passou a sair sem gravar, registrando
quanto trabalho descarta; a gravacao periodica caiu de 2 min para 60 s por
ser a unica rede. Encerramento medido: 0,28 / 0,11 / 0,14 s.
Gate de release verde: verify 10/10 e os tres cenarios de orfaos 100x cada.
Instalador conferido de ponta a ponta: detectou a v1.0.0, atualizou para a
v1.0.1 com SHA-256 conferindo.

=== BOOT ASSINCRONO, 2026-08-03 (commit 94a7469) ===
Relato de uso: Claude Code recusava a conexao com timeout de 30 s no cofre
real de 109 MB / 3.153 notas. O boot tokenizava o cofre inteiro ANTES de
anunciar as tools: 1,3 s de indice de metadados e 219 s de indice invertido.
O host matava o processo antes de SaveInvertedCache, entao a tentativa
seguinte recomecava do zero -- impasse permanente, e nada no log dizia isso.
Medido no cofre real, cache frio: anuncio das tools de 220 s para 2,18 s;
busca utilizavel 206 s depois, em segundo plano. vault_search devolve
INDEX_BUILDING enquanto isso, e nao uma lista curta.
Um defeito foi introduzido e pego ANTES do commit: gravar caches parciais
somado a LoadInvertedCache nao conferir COBERTURA fazia um cache parcial
voltar do disco como completo, e a busca serviria um indice incompleto em
silencio. invertedCacheState compara NoteCount do cabecalho com o indice de
metadados: aceita, retoma ou reconstroi. Tres provas de mutacao verdes.
DOIS RNFs estavam documentados como atingidos medindo SUB-ETAPAS: RNF-01
(index_ms cobre so os metadados) e RNF-02 (96,94 ms sao LoadInvertedCache
isolado; o boot quente real levava 8,6 s contra teto de 300 ms). RNF-02
passou a constar como NAO ATINGIDO em docs/OPERACAO.md e no README.
gen_vault.ps1 ganhou -BodyKB: o cofre sintetico tinha 1,27 MB em 5.000 notas
e o real tem 109 MB em 3.148 -- o custo e por BYTES tokenizados.

=== v1.0.0 PUBLICADA, 2026-08-03 ===
Repositorio tornado PUBLICO. Release v1.0.0 publicada em 9220aba, com quatro
assets (tres binarios e SHA256SUMS.txt). Instalador de uma linha em install.ps1,
provado ponta a ponta contra a release real: baixa, confere SHA-256, instala,
detecta seis hosts e registra em Claude Desktop, Claude Code e Gemini CLI, com o
servidor conferido CONECTADO. Idempotente em duas rodadas seguidas.
Relatorio em .superpowers/sdd/task-77-release-report.md.
Quatro defeitos so apareceram com release de verdade: o workflow nunca publicava
SHA256SUMS.txt; a data de build usava %M/%D e carimbava lixo desde a v0.1.0; o
PowerShell come o separador -- quando escrito literalmente; e `claude mcp add`
recusa nome existente, o que quebrava a segunda rodada.
Antes de publicar, o historico foi reescrito para tirar dados pessoais: 289
commits, 374 SHAs remapeados em 86 arquivos pelo commit-map, nenhum ambiguo.
Backup em bundle antes, conferido. A conferencia achou que a Task 28 estava
registrada num SHA que nunca existiu — defeito anterior a reescrita, corrigido.

=== M6 FECHADO (Tasks 69-77), 2026-08-02 ===
As seis tarefas delegadas ao modelo barato voltaram com cinco corretas e a 72
reprovada na revisao (ver a entrada dela acima). Alem disso, a revisao encontrou
o CI VERMELHO desde 2026-07-28 em todos os pushes, com cinco defeitos que
verify.ps1 nao podia ver por rodar so em Windows. CI verde de novo em 30761531221.

=== REVISAO DO M2.1 (Tasks 33-42), 2026-07-29, pelo modelo principal ===
Nove provas de mutacao rodadas com scripts/mutate.ps1, TODAS saindo 0 (o teste
reprova sob mutacao): gate de classe da correlacao (33), idx.Replace e
idx.Remove dentro de Reconcile (34), aliasKey no Replace (35), byName no
MoveNote (36), contador coalesced e flag active (37), n < 1 na config (38),
idx.Replace no apply (41). Os oito testes por estrutura que a Task 36 exigia
existem, um por cada.
Gate de orfaos: 100/100 ciclos, zero orfaos, com "stdin-eof: 100x" nos motivos
— mecanismo confirmado, nao verde vazio. A Task 42 tinha afirmado isso sem
colar a saida; a afirmacao estava certa, o relatorio e que nao a sustentava.
CORRIGIDO NESTA REVISAO: golangci-lint reprovava com 22 achados (18 errcheck em
arquivos de teste novos, 4 revive), todos introduzidos pelas Tasks 33-42. A
Task 40 zerou o errcheck de producao e os testes novos reintroduziram o dele.
Nenhum relatorio das dez mencionava ter rodado o lint.
RESOLVIDO em 2026-07-29 (era: agent/, com 292 arquivos de assets de um plugin de
skills, estava na raiz do modulo e derrubava as SETE etapas do verify.ps1,
porque build, test, gofmt e check_net varrem a arvore inteira; o gopls tambem
reportava os .go de exemplo como erro do workspace, e a reacao natural de um
agente seria acrescentar as dependencias, violando a regra de nunca rodar
`go mod tidy` por outro caminho). Duas defesas independentes:
  - agent/go.mod tira o diretorio do grafo de pacotes do modulo pai. E o
    mecanismo do Go para isto; nao ha flag de exclusao. Se um plugin apagar
    esse arquivo, o sintoma sao as sete etapas reprovando de novo.
  - verify.ps1 e check_net.ps1 miram ./internal/... e ./cmd/... explicitamente,
    o que segura o proximo plugin que despejar assets na raiz. No check_net a
    correcao e mais que cosmetica: com ./... quebrado, o `go list` devolvia
    vazio e o script AVISAVA que a verificacao nao rodou — um gate que parou
    de gatear continuando a sair verde.
As skills instaladas (Go e Obsidian) ficam versionadas de proposito: skill que
so existe na maquina de quem instalou nao reduz o erro de mais ninguem.

=== M0, M1 e M2 (Tasks 1-42) COMPLETAS! ===

- **Lacuna conhecida**: Reconciliação por overflow não existe em darwin/BSD, porque o backend kqueue do fsnotify v1.10.1 nunca emite `ErrEventOverflow` (ARCHITECTURE §5.3).

Task 18: complete (commit 9547be3) — revisao feita pelo modelo principal, Aprovada.
  48 pares sem orfao, todos codeblocks vazios com .md contendo a sintaxe de verdade,
  harness provado por adulteracao de golden, bytes de CRLF/BOM sobreviveram.
  1 Important roteado para a Task 19: a costura vault.StripBOM -> Parse nao e testada
  por ninguem, e o golden edge/bom.md fixa um estado de FALHA como contrato.

Task 19-26: complete (commits 37fe3e1..3155eec, 11 commits). Revisadas depois:
    os 6 Important da revisao do M1 foram fechados e a paridade contra o dump
    real do metadataCache do Obsidian foi verificada. A linha antiga dizia "SEM
    NENHUMA REVISAO" e ficou aqui depois de deixar de ser verdade, contradizendo
    o cabecalho de M1 completa quatro linhas acima. Corrigido em 2026-07-29.
  Revisao feita pelo modelo principal em 2026-07-28, depois do fato. Achados:

  CRITICAL corrigidos (commit 069f125):
  - offset de BOM nunca somado de volta: toda leitura de secao em nota com BOM saia
    deslocada 3 bytes. Provado: com BOM devolvia "o

## Alvo

CONTEUDO-ESPERAD".
    TestBuildBOM existia e passava — afirmava presenca do heading, nunca o offset.
  - teste de paridade passava sem comparar nada: guard checava os.Stat do diretorio,
    que existia vazio, entao nao pulava; o laco iterava mapa vazio. Reportava a metrica
    do PRD 7 como atingida. Agora checa CONTEUDO e pula.

  IMPORTANT corrigidos em doc (commit 51c4a82):
  - OPERACAO.md 5 trazia medicoes fabricadas ("ex: 408ms", "tende a ficar ~30-45 MB").
  - README declarava "v0.1 publicada" sem tag, sem gate de orfaos, sem medicao.

  IMPORTANT abertos, registrados no plano (commit 3633507):
  - Task 21: reprocessLinksLocked sustenta a corretude de Remove e NENHUM teste o cobre.
    Sem ele, remover nota deixa links afirmando State=ok para arquivo inexistente.
    assertInvariant nao pega: estado obsoleto e internamente COERENTE.
  - Task 23: heading_level sem nenhuma cobertura; caminho AMBIGUOUS_HEADING idem.
  - Task 24: o parametro "fields" e declarado no schema de note_list e ignorado.
    Mais dois comentarios de deliberacao do modelo deixados no fonte ("Wait,").

  BEM COBERTO, verificado por mutacao: ordem de resolucao (alias vs nome vs anexo),
  validacao de ancora, tabela de casamento de frontmatter, limit/offset, remocao de
  contribuicoes no Replace, invariante de backlink nas DUAS direcoes, determinismo,
  cancelamento de ctx, arquivo ilegivel, offsets com frontmatter, anexos e cloud-only.
  Erro de tool devolve erro Go (sem saida zerada junto). Nada escreve em stdout.

=== M1: os 6 Important da revisao FECHADOS. Paridade VERIFICADA. ===
=== Gate de orfaos: RODADO em 2026-07-29, 100/100, zero orfaos. Tag: pendente. ===
    A linha antiga dizia "Falta: gates e tag" e sobreviveu ao gate ter rodado,
    contradizendo o cabecalho de Tasks 1-42 completas. Corrigido em 2026-07-29.

PARIDADE RODOU DE VERDADE (2026-07-28). O que o Obsidian real revelou:

- Obsidian NAO resolve link por alias. Apelidada.md declara aliases [P3, Terceiro] e
  [[Terceiro]] cai em unresolvedLinks. A premissa de RF-62 (P0) estava errada.
  DECISAO DO USUARIO: manter a resolucao por alias como DIVERGENCIA DELIBERADA.
  Registrado em docs/PRD.md junto da evidencia. Consequencia operacional: nossos
  backlinks incluem alias, o painel do Obsidian nao. Nao e bug a reportar.
- URL externa NAO entra no grafo do Obsidian, nem resolvida nem nao-resolvida.
  Confirmou que nosso broken_links estava inflado. Corrigido: estado LinkExternal.
  broken_links no cofre de teste caiu de 9 para 4 — que bate exatamente com a visao
  do Obsidian (5 nao resolvidos) menos Terceiro, que nos resolvemos por alias.
- Link interno inexistente (chapter17.xhtml) CONTA como quebrado. Nosso comportamento bate.

O dumper foi reescrito para schema 2 (resolvedLinks/unresolvedLinks). O schema 1 usava
getFirstLinkpathDest, que nao consulta aliases — comparar resolucao contra ele faria cada
alias virar divergencia falsa, e a reacao natural seria quebrar nosso resolvedor para casar
com o instrumento. A comparacao PULA em referencia schema 1, dizendo isso.

ARMADILHA da propria comparacao, achada por mutacao: comparar so presenca do alvo deixava
passar regressao que desvia UM link entre varios para o mesmo destino. Inverter o desempate
mandava [[PONTO 03]] para Penal/ e Civil/ continuava presente pelos outros cinco. Agora
compara CONTAGEM de arestas, com >= (nao ==), porque resolvemos aliases a mais.

CORRECOES DA REVISAO, todas com prova de mutacao:
- percent-encoding em link Markdown (net/url PROIBIDO — decodificador a mao, byte a byte)
- LinkExternal, para URL nao contar como quebrada
- AliasCollisions implementado (era zero literal em vault_stats)
- NotePaths separado de Paths: Paths tinha notas E anexos, Get so notas, e um unico .png
  no cofre derrubava o teste de propriedade com desreferencia nula
- TestLinkStateFollowsRemoveAndRecreate: reprocessLinksLocked sustentava Remove sem teste
- heading_level e AMBIGUOUS_HEADING cobertos
- note_list projeta (ListItem) e honra fields; antes devolvia *index.Note inteiro

BRIEFS AUTOCONTIDOS (commit cc8dfc3): as Tasks 19-26 do plano ganharam bloco proprio de
contexto — onde encaixa, decisoes fechadas que a vinculam, armadilhas ja pagas que se
aplicam, verificacoes alem dos passos, regras de execucao e contrato de relatorio.
O brief extraido pelo task-brief agora basta para executar: NAO e mais preciso injetar
- scripts/check_net.ps1:10 — segundo `go list` nao redireciona stderr; ruido inconsistente com a linha 4
- docs/ESTRUTURA.md — snippet go.mod diz `go 1.25`, disco diz `go 1.25.0` (cosmetico)

## Decisoes tomadas durante a execucao
- watchStdin NAO entra no WaitGroup. Goroutine parada em Read nao e desenrolavel por
  cancelamento de context; incluir no wg trava Wait() quando sinal ou pai dispara primeiro.
  Caminho de saida documentado: o encerramento do processo. Goroutines de sinal e de vigilia
  do pai ENTRAM no wg, porque ambas fazem select em ctx.Done().
- Constraint de ctx NARROWED (decisao do usuario): `ctx` so em funcoes que podem BLOQUEAR.
  Leitura de env, resolucao de caminho e calculo em memoria nao recebem ctx. Docs atualizados.
- config.Flags ganhou companheiros `ReadOnlySet` e `DebounceMSSet`, preenchidos por
  `cmd.Flags().Changed(nome)`. Sem eles, --read-only=false e --debounce-ms=0 sao inalcancaveis.
  TODA chamada a config.Load precisa preencher os companheiros das flags que o comando expoe.
- go.mod declara `go 1.25.0`, nao `1.24`: forcado por go-sdk@v1.5.0, que declara 1.25.0 no proprio go.mod.
  Pin do SDK (decisao D6) e inegociavel, entao a diretiva sobe. Docs atualizados.
- `go mod tidy` nao roda ate haver importadores: hoje removeria os pins, inclusive o do SDK.
- mcpsrv.Serve usa mcp.IOTransport, NAO StdioTransport — StdioTransport le os.Stdin direto e
  colide com o monitor de EOF do lifecycle. Plano corrigido na Task 9.

- Shutdown nao devolve valor. Falha de etapa e registrada, nunca fatal; unico exit != 0 e o
  guarda-chuva de hardLimit chamando os.Exit(1). Log e o unico observavel, entao os testes
  capturam o logger em buffer.
- watchParent: falha de consulta e AMBIGUA (limiar de 3 falhas consecutivas antes de disparar);
  identidade divergente e DEFINITIVA (dispara na hora). Nao unificar os dois caminhos.
- parent_unix compara ppid capturado no startup, nao a constante 1 — subreaper que nao e PID 1
  (Docker+tini, systemd, s6) quebraria a checagem antiga.
- vault/path: confinamento tem DUAS camadas. validateLocal (lexica: NUL, .., raiz, IsLocal,
  regra de plataforma) e Canonicalize (por componente, via filepath.Rel). Sentinelas distintas
  porque cada uma vira codigo de erro MCP diferente: ErrOutsideVault (travessia),
  ErrAbsolutePath (enraizado), ErrInvalidPath (malformado/dispositivo), ErrEmptyPath.
- Regra de ponto/espaco no fim de componente vive em path_windows.go, NAO em path.go.
  Em Linux/macOS `Notas ` e nome legal; rejeitar la tornaria notas reais inalcancaveis.
- CanonicalPath NAO garante a grafia do disco: esta camada nao consulta disco. Quem produz
  grafia real e vault.Walk; quem corrige grafia de entrada de tool e a resolucao do indice.
- vault.Walk usa walkRoot (= LongPath(root)) tanto no WalkDir quanto no Canonicalize.
  As duas PRECISAM usar a mesma raiz, senao filepath.Rel compara formas diferentes.
- d == nil no callback do WalkDir = falha na propria raiz -> ERRO, nunca `return nil`.
  Cofre inacessivel e cofre vazio nao podem produzir a mesma resposta.
- Entradas descartadas viram contador + amostra em Vault.SkippedEntries(), exposto depois
  em vault_stats. Nota descartada em silencio e nota inalcancavel E indiagnosticavel.
- Regras de isNoise para desktop.ini/Thumbs.db/.DS_Store/*.tmp sao DEFENSIVAS: o filtro de
  extensao ja as descarta. Marcadas como tal no codigo; nao fingir cobertura para elas.
- io.TeeReader NAO propaga EOF: copia bytes, e EOF nao e byte. Usar mirrorReader com
  dst.CloseWithError(err). Sem isso o monitor de stdin do lifecycle fica inerte e
  lc.Wait() so retorna por acidente, via a etapa close-pipe.
- Espelho de stdin e AUXILIAR: falha de escrita nele nao pode virar erro da leitura
  principal, senao mata sessao saudavel por motivo que o cliente nao pode agir.
- serveErr tem capacidade 1 e recebe um valor so. Quem consumir primeiro marca
  serveReturned; a etapa in-flight nao pode tentar ler de novo.
- ctx.Canceled no retorno do serve loop e encerramento NORMAL, nao falha. Duas deteccoes
  de EOF independentes (SDK e lifecycle) correm, e qual vence decide o valor que chega.
- Handler que devolve error Go -> SDK monta IsError SEM StructuredContent. Devolver
  resultado de erro com Out zerado faz VAULT_UNAVAILABLE parecer cofre vazio.
- toolErr precisa de %w, nao %s: e o unico ponto onde erro de dominio cruza a fronteira MCP.
- doctor: marcador por status precisa ser DISTINTO. [OK] ok, [*] aviso, [!] falha.
  Relatorio impresso e a unica coisa que a pessoa le, e ela roda o comando ja confusa.
- doctor halta por PROPRIEDADE (flag halting em rootExists e readable), nao por nome de
  verificacao. Raiz que existe mas nao le produz a mesma cascata de falhas derivadas.
- doctor varre o cofre UMA vez e distribui os fatos. Cinco varreduras em cofre de 20k notas
  era o custo antes.

## DEFEITO REAL ACHADO PELO GATE DE M0 (o mais importante ate agora)
O primeiro teste ponta a ponta deixou 5 de 5 orfaos, contra codigo nao regredido.
Os TRES mecanismos falhavam juntos:
- parent_windows.go comparava so (pid, creation) e ignorava exitTime. Windows mantem PID e
  creation time consultaveis por muito tempo depois da morte -> vigilia nunca detectava.
  CORRIGIDO: campo `exited` vindo de exitTime != 0; sameProcess devolve false, o que cai no
  ramo de divergencia de identidade e dispara na hora (morte confirmada nao e transitoria).
- stdin-eof nao disparava porque o harness dava CONSOLE ao filho, nao pipe. Console nao da EOF.
  CORRIGIDO: harness agora usa ProcessStartInfo com RedirectStandardInput, como um host real.
- sinais nao disparam: taskkill /F sem /T nao levanta evento de console. Esperado.
O script tinha ainda um bug de aspas (PowerShell 7 duplo-escapava) que fazia o teste passar
SEM NUNCA LANCAR O SERVIDOR. Verde provando nada.

## LACUNA CONHECIDA PARA M6
stdin-eof vence sempre no harness atual, entao parent-watch e sinais seguem NAO verificados
ponta a ponta. Fix 1 tem cobertura so unitaria. M6 deve acrescentar um cenario onde o stdin
fica aberto e o pai morre — que e exatamente o caso que a vigilia do pai existe para cobrir.
Regressao comprovada: desligar stdin-eof -> 5/5 orfaos; religar -> 10/10 limpos.

=== DIVIDA DE REVISAO DE M0: PAGA (2026-07-27) ===
Plano: docs/superpowers/plans/2026-07-26-m0-review-fixes.md, tasks R1-R4.
Workspace com o detalhe: .superpowers/sdd/2026-07-26-m0-review-fixes/

As tres tasks que haviam fechado sem revisao fresca (9, 10 e 11) foram revisadas.
Achados: 3 Important, 7 Minor. Todos os Important fechados. Commits b930976..4ff5c2d.

Defeitos reais que a revisao tardia pegou, e que os gates existentes NAO pegavam:
- doctor saia 0 com cofre inacessivel. scanVault fundia cancelamento de contexto
  (aviso legitimo) com falha de raiz (unidade desconectada, share caido, pasta movida
  pela nuvem). Corrigido com scanStatus. O exit code e a unica parte do relatorio que
  um script de setup consegue ler.
- O gate de orfaos imprimia os `reason=` mas nao gateava neles: um servidor que morresse
  sozinho logo apos o startup deixava zero orfaos e rodada verde, sem nenhum mecanismo
  de encerramento ter disparado. Mesma classe do bug de aspas de 2026-07-26.
  Tambem: `-Cycles 0` imprimia [OK] sem lancar nada; e um ciclo que estourava o poll do
  PID deixava um servidor invisivel para a varredura final — orfao real nao reportado.
- cmd/gobsidian nao tinha teste NENHUM. As duas propriedades do fix da Task 9
  (espelho quebrado nao mata sessao saudavel; desconexao limpa sai 0) viviam so numa
  observacao manual que nao existia mais em lugar nenhum.

LICAO DE PROCESSO (custou a revisao final para aparecer):
Corrigir o plano ANTES de despachar o primeiro fixer nao basta. O fix loop muda o
codigo DEPOIS disso, e o plano fica para tras de novo. A ressincronizacao plano<->codigo
tem que acontecer TAMBEM no fim do plano, com comparacao mecanica (extrair o bloco do
plano e diffar contra o disco), nao por leitura.

DIVIDA ABERTA, NAO CAUSADA POR ESTE TRABALHO — MERECE TASK PROPRIA:
golangci-lint reporta 39 issues, identico em GOOS=linux e GOOS=windows
(revive 32, errcheck 5, errorlint 1, contextcheck 1). Verificado no commit de
bootstrap 8f951e5: ja falhava la. O job `lint` do CI esta VERMELHO DESDE O BOOTSTRAP,
e o job `lint-windows` novo ficara vermelho pelas mesmas 39.
Uma delas e excecao DOCUMENTADA no CLAUDE.md e precisa de nolint/exclusao, nao de
mudanca de codigo:
  cmd/gobsidian/serve.go:133 — "Shutdown should pass the context parameter (contextcheck)"
  lifecycle.Shutdown bloqueia e nao recebe ctx de proposito.

LACUNA DE M6 CONFIRMADA E INALTERADA: stdin-eof venceu 100/100 nas duas rodadas do gate,
entao a vigilia do pai e os sinais seguem sem verificacao ponta a ponta.

=== DIVIDA DO LINT: PAGA (2026-07-27, commit d0143bc) ===
golangci-lint: 0 issues em GOOS=linux, darwin e windows. Eram 39, identicas nos
tres alvos, vermelhas desde o commit de bootstrap 8f951e5.

A DECISAO que o usuario tomou (nao era conserto, era escolha de desenho):
  cmd/gobsidian/serve.go:133 — contextcheck em lifecycle.Shutdown
Em vez de nolint ou exclusao, refatoramos: Shutdown agora RECEBE ctx e descarta
so o cancelamento, via context.WithoutCancel (Go 1.21+). Isso ELIMINA a excecao
de ctx do CLAUDE.md em vez de documenta-la — uma regra sem excecao e mais facil
de fazer valer que uma com uma.
Por que WithoutCancel e a resposta certa: o context raiz JA esta cancelado
quando Shutdown roda (e o cancelamento dele que traz o processo ate ali), entao
derivar os orcamentos das etapas dele faria cada etapa NASCER EXPIRADA e ser
abandonada. WithoutCancel preserva valores e joga fora so o cancelamento.

ARMADILHA que quase passou, e vale lembrar:
Meu primeiro teste da refatoracao afirmava "a etapa rodou" sob raiz cancelada.
A mutacao NAO o reprovou — step.Fn e lancada incondicionalmente numa goroutine,
entao ela roda de qualquer jeito, e Deadline() reporta now+budget mesmo num
context ja morto. O teste testava a coisa errada. A diferenca real e se Shutdown
ESPERA pela etapa ou a abandona; so uma etapa LENTA (200ms) expoe isso.
Teste final: TestShutdownIgnoresParentCancellation, provado por mutacao
(base := ctx faz reprovar com "abandonada por estouro de orcamento").

Demais 38: errcheck (5, escritas em stdout de doctor/version — descarte
explicito, porque se o relatorio nao sai nao sobra canal para reclamar),
errorlint (1, vault.go comparava != io.EOF), unused-parameter (2: watchStdin
ganhou `_`, fechando um Minor diferido da revisao de M0; VaultStats idem ate M1
ler os campos), e 30 comentarios de doc + 4 de pacote.

Plano v0.1, CLAUDE.md e docs/ESTRUTURA.md ressincronizados com a assinatura
nova. O bloco do Shutdown no plano foi verificado byte-a-byte contra o disco,
nao por leitura — a licao da revisao final aplicada de imediato.

- parser.SplitFrontmatter devolve offset relativo AO SLICE RECEBIDO, nao ao arquivo.
  A funcao exige entrada sem BOM, entao quem tem o arquivo soma len(bom) quando
  vault.StripBOM reportou true. Nao somar desloca toda leitura de secao em 3 bytes,
  em silencio, so em notas com BOM.
- Heading.Start/Block.Start/Link.Start sao relativos ao mesmo buffer, ja com bodyOffset
  somado por Parse (Task 18). A origem esta escrita em types.go porque um ParsedNote
  sozinho nao permite descobri-la.
- Task 25 (paridade) tem checklist de frontmatter a confirmar contra o Obsidian:
  delimitador de fechamento com espaco final, e chave duplicada no YAML. NAO decidir
  por palpite — as duas fazem uma nota que o Obsidian indexa perder metadados aqui.

ARMADILHA de ferramenta, custou um commit extra:
Script Python que le com encoding=utf-8 e escreve de volta converte o arquivo inteiro
para CRLF no Windows (modo texto traduz 
 para os.linesep). gofmt reprova o .go
resultante. Usar newline="" na leitura E na escrita ao editar arquivo versionado.

- ExtractHeadings: Heading.Start e o inicio da LINHA, nao do '#'. Um heading aceita ate
  tres espacos de indentacao e replace_heading_and_section precisa consumi-los junto.
- Cerca de codigo: o fechamento precisa do MESMO caractere e de comprimento >= o da
  abertura, sem info string. Descartar o comprimento faz um bloco de 4 crases ser fechado
  por uma linha de 3, e a partir dali a hierarquia inteira sai errada. Dentro de uma cerca
  a unica pergunta e se a linha a fecha.
- Hashes de fechamento ("## Titulo ##") so sao removidos quando precedidos de espaco ou
  quando sao todo o conteudo. Sem isso "# Notas sobre C#" vira "Notas sobre C" e a secao
  fica inenderecavel por note_read, note_patch e ancora de wikilink.
- bytes.TrimRight(line, CR) no scan de headings e DEFENSIVO, nao load-bearing: os tres
  consumidores do texto ja passam por TrimSpace. Nao remover esperando teste reprovar.
- Setext heading ("Titulo
======") NAO e reconhecido. Fora do escopo da Task 13, mas o
  Obsidian renderiza — Task 18 (corpus) e Task 25 (paridade) precisam saber.
- heading_level so desempata headings de niveis DIFERENTES. Dois "## Dup" no mesmo nivel
  continuam ambiguos; a task de resolucao de ancora precisa de outro criterio.

LICAO de revisao, vale para as proximas:
Bateria de mutacao achou o que leitura nao acha. Sete regras do modulo sobreviviam a
mutantes com a suite verde — inclusive a que o comentario do proprio fix defendia.
Pedir ao revisor que mute cada regra nomeada, nao so que leia.

- Wikilink com TRES colchetes: o parser RECUSA em vez de chutar. "[[[a]] b](d.md)" fazia o
  gatilho disparar na posicao 0, consumir "[[[a]]" e DESTRUIR o link Markdown para d.md.
  Recusar faz o goldmark reoferecer o gatilho um byte adiante, onde a analise e inequivoca.
- Link.Start/End usam offsetUnknown (-1), nao zero, quando o parser nao sabe a posicao.
  Zero e posicao legitima; reescrever a partir dela sobrescreveria o inicio da nota.
  So LinkWiki e LinkEmbed de wikilink tem offsets reais hoje; fechar isso e M5.
- *gast.Image e coletado como LinkEmbed. Sem isso "![alt](x.png)" fica invisivel enquanto
  "![[x.png]]" e visto — a mesma nota perde metade dos anexos conforme a grafia.
- goldmark.Markdown de pacote e seguro para Parse concorrente (verificado: 64 goroutines
  sob -race). O worker pool de indexacao depende disso.
- A suspensao de wikilink em contexto de codigo foi verificada em 12 contextos, incluindo
  cerca de 4 crases com linha de 3 dentro, span inline de multiplas crases, HTML e
  comentario. O argumento de usar goldmark em vez de regex se sustenta.

ARMADILHA de ferramenta, repetida duas vezes:
Escrever "
" dentro de string Python em heredoc pode virar quebra de linha real e
corromper a linha do plano. Conferir com `sed -n 'N,Mp' arquivo | cat -A` depois de
editar snippet que contenha escapes.

- Campo inline do Dataview NAO consome o valor. Consumir fazia "fonte:: [[STJ]]",
  "capa:: ![[img.png]]" e "fonte:: [STJ](stj.md)" perderem TODO link — dado P0 apagado por
  feature P1. O parser avanca so sobre "chave::" e o goldmark analisa o resto.
- Forma simples do campo inline e de LINHA: a chave comeca no inicio da linha (depois de
  marcador de lista ou prefixo de citacao). Sem isso, como o valor nao e mais consumido, o
  parser redispara no segundo "::" e inventa Inline["1 b"] em "a:: 1 b:: 2".
  Forma entre colchetes vale em qualquer posicao — e assim que o Dataview permite varios.
- Forma entre colchetes recusa quando o "]" e seguido de "(" ou "[": senao
  "[Nota:: veja](destino.md)" consome o campo e o link Markdown some. Mesma falha da Task 14.
- Block id precisa estar na ULTIMA linha do bloco. Aceitar no meio dava faixas sobrepostas
  e replace_block sobrescrevendo linhas nao referenciadas.
- Block.Start em item de lista cai DEPOIS do "- ", de proposito: replace_block responde so
  pelo texto, nao por reemitir marcador e indentacao (que definem aninhamento).
  LIMITE conhecido, documentado em types.go e fixado por teste: em bloco de varias linhas
  os prefixos das linhas de continuacao ficam DENTRO da faixa. M4 precisa reemiti-los.
- Tag precedida de *, _ ou ~ vale. Sao Po/Pc em Unicode, entao checagem por categoria os
  deixava de fora e "**#civil**" sumia — idioma comum que o Obsidian indexa.
- Segmento vazio nao chega em Tags: colapsar sequencias de "/" e tirar das pontas.
  tag_list hierarchical divide por "/" e nos sem nome colidem entre si.

======================================================================
CONTEUDO INCORPORADO DO LEDGER PLANO-ESCOPADO (consolidado 2026-07-28)

Existiam DOIS ledgers: este caminho e
.superpowers/sdd/2026-07-25-gobsidian-v01/progress.md, criado quando o
plugin superpowers passou a escrever por plano. O plano-escopado tinha 6
tarefas e a triagem de Minor; este tinha as 16 e as decisoes. Uma sessao
que lesse so um dos dois re-despacharia trabalho feito ou perderia a
triagem. Agora ha um so, no caminho plano-escopado, e o caminho plano
virou ponteiro.

Remote: https://github.com/jonyduque/Gobsidian.git (adicionado 2026-07-27)

## Minor diferidos de M1 (triagem na revisao de marco)

- frontmatter.go:25-27,:43 — espaco no fim da linha delimitadora rejeita o bloco.
  No FECHAMENTO e grave: cai no caminho "nao fechado" e o YAML INTEIRO vira corpo,
  perdendo metadados em silencio. Obsidian e gray-matter toleram.
- slug.go:41-44 — Slug funde headings distintos: "1.2 Escopo" == "12 Escopo";
  C++/C#/C viram todos "c". Deliberado no brief (e o que faz "Art. 1.234"
  funcionar), mas a resolucao de link (Task 20) precisa decidir o desempate.
- slug.go:33-44 — heading so de simbolo vira slug vazio; "## Deploy" colide.
- slug.go:37 — `!lastSpace && b.Len() > 0`: segunda condicao implicada pela
  primeira. Codigo verbatim do brief.
- frontmatter.go:46,:55,:61 — contrato nil-vs-vazio nao documentado.

## Fatos que vinculam as proximas tasks

- internal/parser e FOLHA: nao importa vault, service nem SDK. Confirmado.
- go.sum ja tinha os hashes de yaml.v3 e x/text; nenhum `go get` foi preciso.
  `go mod tidy` continua proibido.
- BOM e removido por vault.StripBOM (eol.go:54), NAO pelo parser. A fiacao
  parser<->vault so acontece na Task 19 — quem a escrever precisa chamar
  StripBOM antes, ou o frontmatter fica INVISIVEL (nao malformado).
- Task 18 constroi o corpus de golden files: e o que trava o parser inteiro.
- Task 25 precisa de UMA RODADA MANUAL do plugin do Obsidian para gerar a
  referencia de paridade. Nao automatizavel — agendar antes de chegar nela.

## LICAO DE PROCESSO, terceira ocorrencia da MESMA classe

Commits de ressincronizacao ate agora: 4ff5c2d, c77760f, f357448.
Corrigir o plano ANTES de despachar o fixer e necessario mas NAO SUFICIENTE:
o fix loop muda o codigo depois disso, e o plano fica para tras de novo.
A conferencia tem que acontecer TAMBEM no fim de cada task, e MECANICAMENTE —
extrair os blocos go do plano e comparar com os arquivos do disco, nao ler.
Script usado nesta task: extrair todos os blocos com regex e casar por
igualdade exata contra cada arquivo. Ancorar por "package parser" NAO funciona:
o plano tem 68 blocos go e varios comecam igual.

## CI: primeira rodada real (2026-07-27) — DOIS FATOS

1. O job `orphans` PASSOU num runner Windows do GitHub, 16m11s. O gate de
   release deixou de existir so na maquina de um desenvolvedor.

2. Os DOIS jobs de lint reprovaram, com 0 achados locais:
     can't load config: the Go language version (go1.24) used to build
     golangci-lint is lower than the targeted Go version (1.25.0)
   golangci-lint-action@v6 fixa golangci-lint v1.64.8, compilado com Go 1.24,
   e ele recusa um go.mod que declara 1.25.0 — o nosso, forcado pelo pin do SDK
   (PRD D6). Tambem nao le config `version: "2"`.
   Local eu tinha v2.12.2. A verificacao local NAO representava o CI, e eu
   afirmei "lint zerado nos tres alvos" com base nela. Verdade local, conclusao
   errada sobre o pipeline.
   CORRIGIDO em 2dcc4f0: action v8 com `version: v2.12.2` FIXA. Fixar e o ponto:
   com a versao flutuando, os dois lados resolvem binarios diferentes e o
   verde local para de dizer qualquer coisa. Registrado tambem no CLAUDE.md.
   Depois do fix: lint e lint-windows verdes.

Commits de ressincronizacao plano<->codigo ate agora: 4ff5c2d, c77760f, f357448,
e o do job orphans. Quatro, todas da mesma classe.

## 2026-07-28 — FECHAMENTO DO M1, e o defeito que quase saiu com ele

### Gate de orfaos: VERDE

100/100 ciclos, zero orfaos, `stdin-eof: 100x`. Confirma pela terceira rodada
seguida que a vigilia do pai e os sinais SEGUEM SEM VERIFICACAO ponta a ponta —
stdin-eof sempre vence. Lacuna do M6, ja registrada, agora com terceira
evidencia.

### DEFEITO P0 achado ao medir, nao ao revisar

Fui rodar `scripts/measure.ps1` contra o cofre de teste do usuario e o servidor
NAO SUBIU:

    panic: parse "gobsidian://test vault/Origem.md": invalid character " " in host name
        .../go-sdk@v1.5.0/mcp/server.go:523

`registerResources` montava `"gobsidian://" + string(n.Path)`. Depois de `//`
vem a AUTORIDADE, nao o caminho: o primeiro segmento virava nome de host, e
host nao aceita espaco. Pasta com espaco e o caso COMUM num cofre do Obsidian.
O panic e no boot, dentro de AddResource, ANTES de qualquer tool ser anunciada.

Escapar sem mexer na barra nao resolve: `%20` tambem e ilegal em host
(`invalid URL escape "%20"`). A TERCEIRA barra e o que carrega a correcao.

**Por que nenhum teste pegou.** `TestResources` ja exercitava AddResource com
indice real e passava — o cofre dele tem `A.md`. Nenhum caminho de teste tinha
espaco. E `newTestServer` passa indice NULO, com o que ListNotes nao devolve
nada e o laco de registro nunca roda: a maior parte da bateria de mcpsrv nunca
chegava perto de AddResource.

Isto NAO e transcricao errada. `docs/TOOLS.md` e o AD-08 do ARCHITECTURE.md
descreviam a forma quebrada — e o AD-08 usa `Civil/PONTO 03.md` como exemplo,
que e justamente um caso que estoura. A especificacao estava errada, o codigo
foi fiel a ela, e a revisao leu os dois e concordou.

Corrigido em 827d96f. Plano ganhou CORRECAO OBRIGATORIA na Task 24.
`parser.percentDecode` foi EXPORTADA como `parser.PercentDecode` para o mcpsrv
reusar em vez de manter segunda copia das regras de escape invalido.
Prova de mutacao: tirar a terceira barra faz o teste novo estourar por panic.

### Documentacao que prometia o que o codigo nao emite

`docs/OPERACAO.md` §3 listava campos de `vault_stats` que NAO EXISTEM:
TotalNotes, TotalBytes, TotalLinks, TotalAliases, TotalTags, LoadTimeMs,
CloudOnlyFiles. Zero de sete existem. §5 mandava rodar
`gobsidian index --vault --stats`, subcomando inexistente.
Mesma classe do `Collisions: 0`, pelo lado da documentacao.
Corrigido em 95c4308, conferido campo a campo contra `service.StatsResult`.

### Medicao (escala pequena, POR DECISAO do usuario)

`scripts/measure.ps1` criado. Le `index_ms` do log de boot — campo novo — e
amostra WorkingSet64; reporta o MAIOR RSS, nao o ultimo.

Cofre de 7 notas, 3 execucoes, maquina de referencia de 12 nucleos:
  RNF-01 indexacao a frio : 5-8 ms      (alvo <= 3000)
  RNF-07 RSS em repouso   : 18,9-19,3 MB (alvo <= 60)

NAO valida o orcamento. 7 notas nao exercitam RNF-01. O que estabelece e o
PISO: ~19 MB e o custo do processo com indice praticamente vazio, restando
~40 MB para o indice de 5.000 notas. Registrado assim em OPERACAO.md.

### M2 PREPARADO PARA DELEGACAO

Tasks 27-32 escritas no plano, autossuficientes, mesmo formato das 19-26:
onde encaixa / o que ja esta fechado / armadilhas ja pagas / verificacoes alem
dos passos / regras de execucao / contrato de relatorio.

  27 fachada fsnotify + filtro     28 debounce e coalescencia
  29 mudanca real + aplicacao      30 overflow -> reconciliacao
  31 rename por xxhash             32 contadores em vault_stats

Tres decisoes de projeto tomadas no plano, com o motivo, em vez de deixadas
para o executor:
- `vault.Classify` EXPORTADA e consumida por Walk E pelo filtro. Duas copias
  das regras de exclusao divergem, e a divergencia e invisivel nos dois
  sentidos. Exige prova de mutacao cruzada.
- Debouncer com TIQUE UNICO + conjunto sujo, nao mapa caminho->timer como diz
  ARCHITECTURE §5.3. Timer por caminho aloca por arquivo sob rajada do OneDrive
  e sofre inanicao num arquivo escrito continuamente. §5.3 sera atualizado.
- Correlacao de rename LIMITADA A NOTAS. `index.Asset` nao guarda Hash, e
  acrescentar obrigaria a LER todo anexo no build — o que viola a regra fechada
  de que anexo e indexado por nome, nunca lido.

Uma incerteza deixada explicita em vez de chutada: `docs/WINDOWS.md` §4.1
afirma que fsnotify no Windows e recursivo por diretorio observado. NAO
verificado contra a v1.10.1 fixada. A Task 27 manda MEDIR e corrigir o doc se
estiver errado — se nao for recursivo, o escopo dela cresce bastante.

=== M2 FECHADO em 2026-07-29, tag m2-watcher (1fba32b) ===
Gate de orfaos 100/100 com mecanismo confirmado, golangci-lint zerado, sete
etapas do verify.ps1 verdes. Tags do projeto: m0-lifecycle (M0), v0.1.0 (M1),
m2-watcher (M2).

=== M3 (BUSCA) ESCRITO: Tasks 43-50, prontas para delegar ===
43 frontmatter com espaco no delimitador   44 analisador com indexacao dupla
45 indice invertido                        46 BM25 com pesos de campo
47 trecho com destaque                     48 tool vault_search
49 cache em disco + decisao da Q3          50 fechamento do M3

DECISAO FECHADA em 2026-07-29 — COLISAO DE SLUG DE HEADING E ACEITA.
`Slug` funde headings distintos por projeto (e o que faz "Art. 1.234" funcionar)
e a BUSCA NAO DESAMBIGUA: duas secoes com o mesmo slug sao duas ocorrencias,
cada uma com seu offset. `matched_headings` traz o TEXTO do heading, nao o slug,
porque o slug nao e identificador unico. Nao acrescentar sufixo de desempate ao
Slug: mudaria a resolucao de ancora do M1, congelada por golden e por paridade.
Isso fecha a pergunta que os Minor diferidos de M1 deixaram aberta em slug.go.

MINOR DE M1 PROMOVIDO: o de frontmatter.go virou a Task 43. Espaco no fim do
delimitador de FECHAMENTO faz o YAML inteiro virar corpo, perdendo metadados em
silencio. Era Minor porque afeta poucas notas; deixa de ser no M3 porque a busca
filtra por tag e por campo de frontmatter, e o sintoma vira "a busca nao acha".
Os demais Minor de M1 (slug vazio, condicao redundante, contrato nil-vs-vazio)
carregam para o M4 sem relacao com busca.

=== REVISAO DO M3 (Tasks 43-50), 2026-07-29, pelo modelo principal ===
Gate estava verde: verify.ps1 7/7 e golangci-lint 0 issues, primeira vez sem
precisar conserto. Nenhum dos dois pega o que segue.

CRITICAL: o indice de busca NUNCA e atualizado pelo watcher em producao.
serve.go constroi inv e passa a service.New, mas watcher.New nao o recebe, o
Watcher nao tem campo de search, e Run chama Apply sem o variadico — entao
searchInv e nil sempre e o bloco de atualizacao em apply.go e codigo morto.
Nota criada/editada/removida com o servidor rodando nao entra/sai da busca ate
reiniciar. Os dois call sites de teste tambem nao passam inv: cobertura zero.
O parametro `inv ...*search.Inverted` e o que tornou isso silencioso —
esquecer de passar COMPILA. O brief da Task 45 pedia o teste ponta a ponta que
teria pegado; nao foi feito, e as tres lentes aprovaram. Vira a Task 51.

CRITICAL: a medicao da Q3 mediu UMA nota rotulada como 100. persist_test.go
inseria 100x o MESMO caminho; o log do proprio teste dizia "Notes: 1" ao lado
de "100 notas". E a comparacao que decide a pergunta — custo de RECONSTRUIR a
busca a partir dos metadados carregados — nunca foi feita; mediu-se Save/Load
do cache, o outro lado. Q3 REABERTA no PRD 11. RNF-02 e RNF-04 retirados do
OPERACAO.md e marcados "nao medido": numero falso esperando tarefa e numero
falso no repositorio. O p95 de RNF-04 tambem nao era p95 — uma chamada unica.
Vira a Task 52.

MECANICO (Task 53): TestInvertedConcurrencyRace(_ *testing.T) nao pode falhar
(descarta o t); tres `_ = searchInv.Update(...)` e um `_ = SaveInvertedCache`
engolindo erro; nota ilegivel descartada em silencio no rebuild do boot; as
cinco constantes do BM25 fixadas por ordenacao e nao por valor (perturbacao
pequena sobrevive: k1->0.9, b->0.5, headings->1.5 todos passam), entao os
testes precisam dizer no NOME o que verificam de fato; sete constantes e nao
cinco; verificacao de query vazia mais fraca que a pedida; pacote de revisao de
intervalo vazio.

DOIS ACHADOS DA REVISAO ESTAVAM ERRADOS, e a correcao e nos docs:
- watcher importar search NAO e violacao: ARCHITECTURE 5.3 poe search.Update no
  pipeline, logo depois de index.Replace.
- search importar index NAO e violacao: 6.2 exige pesos por campo, que precisam
  de titulo e headings.
O paragrafo do ARCHITECTURE 1 que dizia "as camadas abaixo nao se conhecem",
sem excecao, contradizia 5.3 e 6.2 e foi corrigido com as duas excecoes
nomeadas. O "e folha" do brief da Task 44 tambem: vale para analyzer.go e
persist.go, nao para bm25.go e snippet.go.

O QUE PASSOU: mutacoes que rodei e sairam 0 — espaco no delimitador (43), peso
do titulo (46), offset do BOM (47), termo orfao (45). As Tasks 43 e 47 fizeram
o dever de offset: o teste le do DISCO e reprova sem o ajuste de BOM. Os nove
parametros de schema e os seis campos de retorno da Task 48 tem teste nomeado
cada um. Ledger sem SHA fantasma, reusado ou intervalo vazio nas 43-50.

=== M3.1 ESCRITO: Tasks 51-53, prontas para delegar ===

=== REVISAO DO M3.1 (Tasks 51-53), 2026-07-29, pelo modelo principal ===
Tasks 51 e 53 CORRETAS, verificadas por mutacao: w.inv->nil e
searchInv.Remove->_ = path ambas reprovam TestWatcherUpdatesSearchIndex, entao
a fiacao E a metade da REMOCAO estao cobertas. O variadico morreu. Os testes do
BM25 foram renomeados como pedido: TestBM25ParamB virou
TestBM25DocumentLengthNormalization — o nome agora diz o que verifica.

Task 52 mediu bem e decidiu por outro critério. O corpus esta amarrado por
QUATRO asserçoes (NoteCount, header.NoteCount, loadedInv.DocCount,
rebuiltInv.DocCount) e as DUAS medicoes da Q3 foram feitas. Mas o plano
pre-comprometia o critério — "se (b) couber em 300 ms, persista SO os
metadados, porque e o formato mais barato de versionar" — e (b) deu 106,58 ms,
que cabe. A decisao entregue foi persistir ambos, por velocidade (~80 ms de
boot). DECISAO MANTIDA por escolha de quem pediu, em 2026-07-29. O que era
defeito era o silencio: nem o relatorio nem o PRD diziam que havia regra.

RNF-04 REMEDIDO pelo modelo principal. A medicao da Task 52 chamava
search.CalculateBM25 direto e reportou 0,58 ms — aritmetica honesta sobre a
fatia errada, rotulada com o nome do todo, e consultando termos que casavam UMA
nota em 500 (dai "Minimo: 0s, Mediana: 0s" ao lado de um p95). Medido por
service.Search, com trecho e filtros, POR FORMATO de consulta:
  termo amplo 14,9ms | dois termos 17,7ms | seletivo 5,7ms | pasta 17,4ms
  tag 11,9ms | FRASE EXATA 174,2ms | trecho 1000 24,5ms | limit 200 85,0ms
RNF-04 ATINGIDO EM 7 DE 8 FORMATOS. NAO atingido em frase exata: 174 ms contra
100 ms. Registrado no OPERACAO.md como falha, com teto de 250 ms no teste —
guarda contra piorar sem afrouxar os outros sete nem esconder com t.Skip.
Fechar a lacuna e a Task 61. Minha suposicao de que limit=200 estouraria estava
ERRADA: cabe (85 ms).
Asserçao de tempo nao vale sob -race (frase exata: 174 ms -> 1,00 s). Guardada
atras de constante com build tag em arquivo separado; medicao continua nos dois.

OPERACAO.md: duas afirmacoes ficaram para tras do M2 INTEIRO — "a v0.1 nao tem
um watcher de arquivos ativo" e "no futuro, overflow indicara problemas com o
watcher". As duas eram verdade quando escritas e deixaram de ser nas Tasks 27 e
30. Corrigidas, e a tabela de vault_stats ganhou os dez campos de watcher.

HARNESS: golangci-lint entrou no verify.ps1 como oitava etapa, com conferencia
da versao v2.12.2. Enquanto era comando separado, era comando que alguem
esquecia — 22 achados nas Tasks 33-42, dez relatorios, nenhum mencionando o
lint. `go vet` NAO pega errcheck. AGENTS.md 4 ganhou as quatro licoes de
medicao; GEMINI.md ganhou a regra de escrita atomica inline, porque o M4 e o
marco que pode destruir dados.

=== M4 (ESCRITA) ESCRITO: Tasks 54-62, prontas para delegar ===
54 trava por caminho          55 escrita atomica + GATE RNF-11 (1.000 crashes)
56 diff de Myers              57 patch/append por heading (offset com BOM+CRLF)
58 replace_block por ^id      59 tools com dry_run e expected_hash
60 read-only fora do ListTools 61 lacuna do RNF-04 (frase exata)
62 fechamento do M4
RNF-11 e o critério de bloqueio: zero notas corrompidas em 1.000 iteracoes de
crash injetado. Nenhuma outra tarefa fecha antes da 55 passar.

=== REVISAO DO M4 (Tasks 54-62), 2026-07-30, pelo modelo principal ===
Gate verde nas OITO etapas, arvore limpa, nove relatorios, nove tarefas no
ledger. RNF-11 e honesto: o crash mata um SUBPROCESSO de verdade (TestMain
despachando por env), nao uma goroutine. Sondei quantas iteracoes exercitam a
escrita: 340 de 1000 terminaram com o conteudo novo. Minha suspeita de que o
kill em 0-39 ms sempre chegasse antes da escrita comecar estava ERRADA.
Task 61 e o melhor trabalho do marco: pprof rodado, causa raiz nomeada
(matchPhraseInNote chamava Postings por candidato), p95 da frase exata de
174,2 ms para 22,1 ms, e reproduz aqui em 33,2 ms. Dentro do alvo de 100 ms.
Task 60 afirma a LISTA do ListTools nome por nome, nos dois modos.

QUATRO ACHADOS, TODOS CORRIGIDOS pelo modelo principal em 2026-07-30 (delegar
nao compensava: a unica decisao de projeto era onde por a varredura, e ela e a
maior parte do valor):

1. CORRIDA NO VARREDOR (Important). WriteAtomic chamava CleanStaleTempFiles(dir)
   no inicio, e o glob apaga TODOS os .gobsidian-tmp-* do diretorio — inclusive
   o de outra escrita em voo. A trava do writer e por CAMINHO de proposito
   (Task 54), entao duas notas na mesma pasta escrevem em paralelo; o recurso
   compartilhado e o DIRETORIO e a trava nao o cobre.
   Medido: os.Remove sobre arquivo com handle aberto FALHA no Windows com
   sharing violation, e o erro era engolido — a corrida ficava mascarada. Em
   Linux e macOS o unlink sucede por POSIX, a outra escrita segue gravando num
   inode desvinculado, e o rename falha com ENOENT. O projeto faz vet nos tres
   alvos e o M6 publica binario para os tres.
   CORRECAO: a varredura saiu de WriteAtomic. WriteAtomic ja removia o proprio
   temporario no defer em qualquer falha normal; o unico caso que sobrava era
   processo morto, que nao roda defer. Novo SweepStaleTempFiles(ctx, root) roda
   NO BOOT, onde nao ha escrita em voo — e resolve o achado 3 de graca.
   Regressao: TestWriteAtomicConcurrentSameDirectory, 8 escritores x 25
   escritas na mesma pasta. ATENCAO: no Windows ele passa dos dois jeitos
   (sharing violation mascara); reprova de verdade em Linux e macOS. Esta
   escrito no comentario do teste.

2. ASSERCAO QUE NAO PODIA FALHAR (Important). O teste do RNF-11 chamava
   CleanStaleTempFiles e DEPOIS afirmava que nao havia sobras: a linha de cima
   garantia a de baixo. Medido: com a limpeza neutralizada, o teste REPROVA —
   havia temporarios reais sendo mascarados.
   CORRECAO: conta os orfaos ANTES da varredura, afirma que a varredura removeu
   todos, e afirma zero DEPOIS. E reprova se os orfaos forem ZERO: 92 orfaos em
   1.000 iteracoes sao as que morreram entre criar o temporario e renomear, que
   e exatamente a janela que a atomicidade cobre. Zero significaria que o crash
   nunca caiu nessa janela e o teste nao exercitou o que promete.

3. TEMPORARIO ORFAO FICAVA NO COFRE (Minor). A varredura preguicosa so rodava
   na proxima escrita naquela pasta; se ela nunca fosse escrita de novo, o
   temporario ficava para sempre. Resolvido pela varredura de boot, e agora
   documentado no OPERACAO.md com a linha de log que aparece.

4. O AUDITOR PAROU DE GATEAR EM CINCO TAREFAS (Minor). O ledger passou a usar
   um segundo formato de linha nas Tasks 58-62 ("- Task NN: Titulo | base X |
   review Y..Z | commit W"), que o regex do audit_reports.ps1 nao casava. As
   tres checagens de SHA simplesmente NAO RODARAM naquelas cinco linhas, sem
   aviso. CORRECAO: o auditor aceita os dois formatos. Provado por sonda —
   troquei o SHA de review da Task 60 por "deadbee" e ele acusou SHA-FANTASMA na
   linha 80; restaurado.

=== M5 (REFATORACAO DO COFRE) ESCRITO: Tasks 63-68, prontas para delegar ===
63 offsets de link Markdown no parser   64 writer/linkrewrite.go
65 tool note_move                       66 tool note_delete
67 ancoras quebradas no impacto         68 fechamento do M5

Segundo marco que pode destruir dados, e mais insidioso que o M4: uma escrita
malfeita aqui nao corrompe UMA nota, reescreve links em DEZENAS que o usuario
nao pediu para tocar.

QUATRO DECISOES FECHADAS em 2026-07-30:
1. LINK MARKDOWN E REESCRITO. Contradicao de contrato resolvida a favor do
   TOOLS.md: RF-35 diz "todos os wikilinks", o TOOLS.md promete preservar "a
   escolha entre wikilink e link Markdown", e o TOOLS.md e o que o modelo do
   outro lado le. Um note_move que deixa [texto](Civil/PONTO 03.md) apontando
   para o caminho antigo quebra em silencio. Por isso a Task 63 existe: o
   parser so preenche Start/End para LinkWiki e LinkEmbed, e o comentario em
   types.go JA atribuia essa lacuna ao M5. RF-35 e corrigido na Task 68.
2. A REESCRITA PASSA PELOS Links DA NOTA DE ORIGEM, nao pelo Backlink.
   index.Backlink nao tem Raw nem offsets — o laco obvio,
   for bl := range idx.Backlinks(alvo), nao consegue reescrever fielmente.
3. SEM TRANSACAO ENTRE ARQUIVOS, E O ARQUIVO MOVE POR ULTIMO. Falha antes do
   move deixa links reescritos apontando para arquivo que ainda existe no lugar
   antigo: inconsistente, mas nada quebrado, e reexecutar conserta.
4. .trash/ JA ESTA EM excludedDirs (walk.go:31). A nota sai do indice pelo
   watcher sozinha. NAO acrescentar caso especial.

DEFEITO MEU, PEGO ANTES DE DESPACHAR: o `task-brief` extrai de "### Task N" ate
o proximo "###". Tudo sob o cabecalho "## M5" — as quatro decisoes acima e seis
regras compartilhadas — NAO chegava a brief nenhum. Os briefs saiam com 22 a 71
linhas e o da 67 tinha ZERO das quatro decisoes; nenhuma das seis secoes tinha
Regras de execucao nem Contrato de relatorio. Corrigido: cada secao repete o que
a vincula, mais as regras e o contrato proprios. Conferido por grep depois de
regerar, nao suposto. A skill gobsidian-execution ganhou o aviso, porque o mesmo
provavelmente valeu para o M4 e passou por sorte.

## Revisao do M5 — pendencias fechadas em 2026-08-01

Fechadas as duas pendencias que a revisao do M5 deixou abertas. Ambas rederam
defeito real, nenhum deles inferido: cada um esta reproduzido, corrigido e
provado por mutacao.

**1. O teto do RNF-04 nao era cobrado em lugar nenhum.** Existiam DUAS
assercoes de RNF-04. A do `internal/search` media `CalculateBM25` direto com um
termo presente em 5 de 500 notas — `Mediana: 0s, p95: 1,02 ms` contra teto de
100 ms, um teste que nao podia falhar — e era a unica que o gate rodava. A do
`internal/service`, que mede pela camada que o RNF nomeia, e pulada sob `-race`,
e `-race` era o unico modo em que `verify.ps1` rodava testes. O gate cobrava a
tautologia e pulava a medicao.

- `TestRNF04SearchLatencyPercentile` virou `TestBM25KernelLatency`, com consulta
  que casa as 500 notas (mediana 4,1 ms), teto medido de 80 ms e falha explicita
  se a mediana for zero. Mutacao: voltando a consulta seletiva, reprova
  (`a consulta casou 1 notas de 500`) — `[OK] VERIFICADA`.
- `verify.ps1` ganhou a etapa `go test (RNF-04, sem -race)`. Sao ~7 s.
- A instabilidade que motivou a investigacao foi reproduzida: quatro copias do
  binario de teste ao mesmo tempo dao p95 de 100,6 / 102,9 / 107,4 ms em tres de
  quatro, contra 81 ms ocioso. Uma delas estourava por 0,6%. A medicao agora
  repete ate 3 vezes antes de reprovar — pico de carga nao sobrevive a tres
  rodadas, regressao sobrevive a todas. Mutacao: baixando o teto do formato
  `limit: 200` para 10 ms, reprova depois das tres rodadas — `[OK] VERIFICADA`.
- Repetir NAO cria folga. `limit: 200` segue com ~20% dela, registrado em
  `docs/OPERACAO.md` como lacuna de M6, nao como alvo atingido com conforto.

**2. O teste ponta a ponta de `note_move` (Task 65) foi escrito — e achou uma
corrida de dados em producao.** `internal/mcpsrv/e2e_move_test.go` monta a pilha
como `cmd/gobsidian` monta, move pela tool com o servidor no ar e confirma que
`vault_search` e `link_graph` respondem pelo caminho novo.

Na primeira execucao sob `-race`: `index.MoveNote` escrevendo `n.Path` contra
`service.Search` lendo `note.Path`. Nao e trava esquecida — `MoveNote` segura
`ix.mu.Lock()` inteiro. O mutex protege o MAPA; o `*Note` escapa por `Get` e por
`List`, e o chamador le os campos depois de soltar o `RLock`.
`reprocessLinksLocked` tinha o mesmo defeito e alcance muito maior: reescreve os
`Links` de todas as notas, a cada evento do watcher, nao so em move.

Invariante agora explicita e por uma funcao so (`mutarNotaLocked`): **`*Note`
publicado em `ix.notes` e imutavel; quem muda uma nota troca a entrada por uma
copia.** `Build` e `Replace` ja obedeciam sem que ninguem tivesse escrito a
regra. Mutacao: desligando o `w.Run`, o teste reprova
(`vault_search ainda responde por "origem/alvo.md"`) — `[OK] VERIFICADA`.

**3. Achado nao previsto: o criterio de bloqueio do M4 nao verificava nada.**
Aparecido quando o gate ficou vermelho por outro motivo.
`TestRNF11NoCorruptionUnder1000Crashes` matava o filho de 0 a 39 ms contados do
`cmd.Start()`. Mas o filho e o proprio binario de teste, e paga init antes de
chegar em `WriteAtomic`. Medido: escrita nao interrompida leva mediana de
**47,2 ms sem `-race` e 1,077 s com**. A janela cabia inteira dentro do init —
sob `-race`, em 100% das 1.000 iteracoes. O teste relatava "0 corrompidas em
1.000 iteracoes" sem ter escrito um byte.

Quem denunciou foi a guarda `orfaos == 0`, escrita junto com o teste e correta
desde entao. Corrigido sincronizando com a escrita: o filho avisa em stdout
imediatamente antes de `WriteAtomic`, e o pai mata de 0 a 9,95 ms depois do
aviso. Iteracoes em pool, porque sequencial sob `-race` daria ~18 min.

  antes: 0 orfaos, 69 s
  depois: 381 orfaos sob `-race` (280 sem), 0 corrompidas, ~20 s

Mutacao, possivel pela primeira vez: trocando `os.CreateTemp` por
`os.OpenFile(alvo, O_TRUNC)`, o teste reprova com **7 de 1.000 iteracoes
corrompidas**, truncadas a 0 bytes — `[OK] VERIFICADA`. A mesma mutacao passava
verde antes.

**Commits:** `9d7f9dd`, `efee542`, `a08a660`, `9dc6f00`. Bateria `verify.ps1`
verde nas nove etapas depois do ultimo.

**M6 ainda NAO preparado.** Antes de escrever tarefa, a Fase 2 pendente: o
padrao "parametro de schema que o codigo ignora" apareceu cinco vezes nesta
revisao e merece instrumento, nao mais uma linha em `AGENTS.md`.

## M6 preparado para delegacao — 2026-08-01

Tasks 69 a 77 escritas autocontidas em `docs/superpowers/plans/2026-07-25-gobsidian-v01.md`,
commit `4463092`. Briefs gerados e conferidos por `check_briefs.ps1`: as nove
carregam Verificacoes, Regras de execucao, Contrato de relatorio, clausula de
mutacao e linha de Files, e as nove passam do piso relativo de tamanho.

**Fase 2 do preparo (melhoria de harness) rendeu instrumento novo.** O padrao
"parametro de schema que o codigo ignora" tinha aparecido CINCO vezes e sempre
fora pego por pessoa lendo codigo. `scripts/check_tool_params.ps1` (commit
`fa65e5f`) o mecaniza: todo campo de struct `*Input` em `internal/mcpsrv` tem
de ter o identificador Go aparecendo alem da propria declaracao. Nenhum gate
pegava porque nenhum podia — o campo E usado, pelo decodificador de JSON, o que
satisfaz todo analisador de uso.

**Achou quatro instancias novas na primeira execucao**, todas com contrato
documentado em `docs/TOOLS.md` que o codigo nao honra: `note_metadata.include`,
`link_graph.direction`, `link_graph.include_broken`, `link_graph.include_embeds`.
Viraram a Task 69. O checador NAO entrou no `verify.ps1` ainda, de proposito:
entraria vermelho. Entra junto com a correcao, na propria 69.

**Divisao para delegacao.** Ao modelo barato: **69, 70, 71, 72, 74, 75** — as
seis sao transcricao ou medicao com passos fechados. Ficam com o principal:
**73** (o gate de bench precisa julgar o que conta como regressao e provar que
dispara), **76** (o entregavel e projetar um cenario de teste que hoje nao
existe, e o modo de falha e escrever teste que nao pode falhar) e **77** (o
entregavel sao relatorios com evidencia real, e o modo de falha de um modelo
barato pedido a "escrever relatorios com evidencia" e fabrica-la).

## Desempenho do boot com cache quente, e tres defeitos de producao — 2026-08-04

Trabalho fora do plano numerado, pedido em sessao: performance do boot com cache
quente. Rendeu seis commits, e a investigacao achou tres defeitos que nenhum
gate via.

**Formato do cache: `gob` -> codec binario proprio (`a132cb9`, `d814d0c`,
`cddd511`).** Formato 1 -> 5. Medido no cofre real de 3.152 notas / 109 MB,
mesma maquina, mesmos dados:

| Medida | Antes (gob) | Depois | Fator |
|---|---|---|---|
| Arquivo | 505.643.791 B | 70.084.435 B | 7,2x |
| Carregamento | 5,59 s | 659,2 ms (±23%, n=6) | 8,5x |
| Bytes alocados | 3,69 GB | 389,8 MiB | 9,7x |
| Alocacoes | 13.035.004 | 291.104 | 44,8x |
| RSS em repouso | 782,8 MB | 381,5 MB | 2,1x |
| Boot quente (busca utilizavel) | ~7 s | 842 ms | 8,3x |

Pecas: tabela de caminhos (286,3 MB eram a mesma string repetida 2,96 milhoes de
vezes), posicoes em varint sobre delta, codec sem reflexao, arrays achatados com
busca binaria no lugar de 126 mil mapas internos, e `FreeOSMemory` depois da
montagem. Detalhe e todas as medicoes em `docs/OPERACAO.md`.

**GOGC foi testado duas vezes e rejeitado nas duas.** `GOGC=off` deu `~
(p=0,093, n=6)`. `GOGC=400` deu -28,51% (p=0,002) no benchmark, mas o benchmark
nao e o boot: nele o heap vivo e o dobro do de uma partida real. No boot real,
12 partidas por braco, mediana 1217 -> 1147 ms com U de Mann-Whitney = 88 contra
regiao critica de 37/107 — nao significativo. E o RSS ficou pior. Fica
registrado para nao ser re-litigado.

**RNF-02 segue NAO ATINGIDO**: 832-1183 ms contra teto de 300 ms. Esta 6,7x mais
perto e o que sobra e construcao de mapas Go, nao serializacao.

### Tres defeitos de producao achados no caminho

Nenhum era hipotetico, nenhum tinha teste, todos davam resultado errado em
silencio.

1. **`DocLength` divergia entre indice construido e recarregado** (`cddd511`).
   Era derivado somando postings, e um token cuja forma reduzida difere da raiz
   entra em DUAS. Medido: 5 recem-construido, 10 recarregado. E o divisor da
   normalizacao por tamanho do BM25 — o mesmo cofre ranqueava diferente
   conforme o servidor tivesse acabado de indexar ou de ler o cache. Junto:
   nota vazia nunca contava como coberta, entao TODO boot concluia "cache
   parcial" e regravava o cache inteiro (3,3-3,9 s por partida).

2. **A reconciliacao por overflow reparava so o indice de metadados**
   (`ef3d4da`). `service.Search` descarta posting cujo caminho nao esta nos
   metadados, entao uma nota movida durante o overflow devolvia ZERO resultados
   para sempre. Segundo buraco no RF-05 depois do do M2.1: aquele era cobertura
   zero, este era cobertura que afirmava sobre metade da resposta.

3. **Diretorio novo nao era varrido** (`58658fc`). Pasta que chega ao cofre ja
   com arquivos dentro entregava UM evento e nenhum arquivo: "eventos
   recebidos=1, notas no indice=0". As notas ficavam invisiveis para todas as
   tools ate o proximo reinicio. E o usuario arrastando uma pasta para o cofre —
   e era tambem o que fazia `note_move` perder a nota quando o rename vencia o
   registro do watch, o que reprovava `TestE2E_NoteMove...` em 3 de 10 rodadas
   da suite sob `-race`.

### Dois testes que mediam a maquina, nao o mecanismo

- `TestE2E_NoteMove...` tinha prazo de convergencia de 60 s e contexto de sessao
  de 30 s escrito a mao: o laco sobrevivia ao proprio transporte e a falha saia
  como "connection closed", culpando a conexao (`bd7efaf`).
- `TestCounters_ReconciledUpdatedAndRemoved` rodava o watcher inteiro, entao o
  caminho normal competia com o reconciliador. Sonda com 300 ms de vantagem ao
  caminho normal: reprovava 3 de 3 (`1f610a5`). Mesma armadilha de
  `TestOverflowReconciliationFull`.

### Estado

`verify.ps1` verde nos 10 passos. Binario conferido contra o cofre real: boot
905 ms, busca do cache 842 ms, `doctor` apto. **Todo cofre reconstroi o cache no
proximo boot** — o formato mudou e caches antigos sao recusados como versao
incompativel, com a reconstrucao em segundo plano.

Pendente: reamostrar a suite sob `-race` depois de `58658fc` para confirmar que
o e2e parou de reprovar, e o gate de orfaos desta rodada.

## Task 78 (M7) — golden de ranking de busca — 2026-08-05

`TestRankingGolden` (`internal/service/ranking_golden_test.go`) e seis
`testdata/ranking/*.tsv` (caminho + score com 6 casas), commitados em
`61d2ab7`. Base contra a qual as seis otimizacoes seguintes de M7 (tasks 79-87)
sao medidas: qualquer uma que mude a ordem tem de justificar por escrito, nao
regenerar com `-update` para fazer passar.

**O corpo literal do brief nao gerava cobertura real, e foi corrigido antes de
commitar.** O template original dava as 300 notas o mesmo titulo, heading e
frase final palavra por palavra — "algoritmo BM25 com pesos" e "intercorrente"
identicos em toda nota. Gerado com `-update` e lido antes de aceitar (regra do
CLAUDE.md), os arquivos `frase-exata.tsv` e `so-no-titulo.tsv` saiam com
exatamente 20 linhas: nao porque so 20 notas casavam, mas porque as 300 casavam
empatadas e o Limit padrao cortava ali. As duas perguntas que a propria tarefa
manda conferir — "a frase exata casa uma so?", "a nota com o termo no titulo
vem antes da que so tem no corpo?" — nao tinham como ser respondidas: nenhuma
nota tinha o termo so no corpo, nenhuma tinha a frase incompleta. Um golden que
empata em tudo passa e nao cobre nem peso de titulo nem casamento de frase
unico — o proprio "golden que passa com o corpus errado" que a tarefa nomeia
como modo de falha, so que produzido pelo corpus que o brief mandava
transcrever, nao por erro de transcricao.

Corrigido com duas notas de contraste, resto do template inalterado: `n0150` e
a UNICA nota com a sequencia exata "algoritmo BM25 com pesos" (as outras 299
tem as mesmas palavras fora de ordem); `n0250` e a UNICA nota com
"intercorrente" no corpo e nao no titulo, contra cinco notas (`n0005`..`n0045`)
que tem o termo so no titulo. Golden confirma o esperado: as 5 notas de titulo
empatam em 6.619287, `n0250` fica abaixo em 4.221181 — peso de titulo (3x)
batendo peso de corpo, visivel no arquivo, nao so inferido.

Efeito colateral honesto, nao um defeito: `n0150` tem frase de fechamento mais
curta que as outras 299 (9 tokens contra 12), entao a normalizacao por tamanho
do BM25 da a ela um score levemente mais alto em TODA consulta, inclusive as
que nao tem nada a ver com a frase — e por isso ela aparece no topo de
`termo-amplo.tsv`, `dois-termos.tsv`, `com-acento.tsv` e `so-em-heading.tsv`. E
o normalizador de tamanho funcionando, nao um vies do corpus a esconder.

`scripts/verify.ps1 -SkipCross -SkipNet` verde nos 7 passos.
`go test -run TestRankingGolden ./internal/service/` verde lendo do disco (sem
`-update`). Sem prova de mutacao: tarefa nao toca codigo de producao.

## Task 79 (M7) — checador de artefato citado na doc que nao existe no codigo — 2026-08-05

`scripts/check_doc_refs.ps1`, commitado em `cfb3d60`. Varre `docs/*.md` (nao
recursivo) e `README.md` por token entre crases que parece identificador de
codigo — nome de arquivo `.go`/`.gob`, `snake_case`, `CamelCase()` — e falta
em todo `.go` do repositorio. `.go` confere presenca em disco pelo nome-base;
`.gob`/`snake_case`/`CamelCase()` conferem substring no corpus de `.go`
(producao+teste), porque sao artefato de dado ou identificador, nao arquivo
versionado — arquivo-fonte real nunca aparece por nome dentro de outro `.go`,
entao testar os dois do mesmo jeito deu 21 falso-positivo na primeira rodada.

Achou os dois casos reais do repositorio hoje: `index_cache.gob`
(`docs/PRD.md` Q3, decidido e nunca implementado — so `inverted_cache.gob`
existe) e `total_bytes` (`docs/TOOLS.md` promete no retorno de `note_read`,
`service.ReadResult` nao tem o campo nem `path`). Volume total: 14 achados,
os outros 12 sao ruido explicado no relatorio (arquivo interno do `fsnotify`
vendorizado, campo Go mostrado em minusculo numa tabela, nome proibido citado
como exemplo de convencao, sysctl do Linux).

Bug de parsing PowerShell no caminho: `` `{3}` `` dentro de string de aspas
duplas faz o segundo backtick escapar a aspa de fechamento (`` `" ``), e o
parser consome o resto do arquivo procurando a proxima aspa — corrigido com
crase dupla (` `` ` = crase literal). `-match`/`-notmatch` do PowerShell sao
insensiveis a maiuscula por padrao; sem `(?-i)` nos padroes e `-cnotmatch` na
busca, `SNAKE_CASE` casava com constante GRITANTE do Windows
(`ERROR_SHARING_VIOLATION`). Os dois foram achados olhando o volume antes de
aceitar a regra, como o brief manda.

Prova de disparo (sem mutacao — entregavel e PowerShell): inserido
`` `create_dirs` `` em `docs/TOOLS.md`, achado foi de 14 para 15; removido,
voltou a 14; `git status --porcelain docs/TOOLS.md` vazio.

`scripts/verify.ps1 -SkipCross -SkipNet` verde nos 7 passos. Relatorio
completo em `.superpowers/sdd/task-79-report.md`.

## Task 80 (M7) — Normalize pool com reutilizacao de transformers — 2026-08-05

`sync.Pool` de transformadores de normalizacao Unicode para eliminar 80,45%
de alocacao em `internal/search/analyzer.go:26`, que reconstroía o pipeline
`transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)` a
cada chamada. Commit `ffa0bb5`.

Mudança 1 (pool): Pool + Reset() antes de usar.
- Benchmark BenchmarkSearchLimit200: 188.51Mi → 54.88Mi (−71% bytes), 128.91k → 46.38k (−64% allocs)
- Golden de ranking idêntico (`TestRankingGolden` passou)
- Teste concorrente `TestNormalizeNaoVazaEstadoEntreUsos` adicionado (32 goroutines × 1000 voltas)

Mudança 2 (ASCII atalho): Atalho se string sem byte >= 0x80. Resultado em benchstat:
`~` (sem diferença significativa). Revertida conforme brief (dívida sem ganho).

**Correcao da observacao original sobre a prova de mutacao (revisor, 2026-08-05).**
O relatorio da tarefa registrou aqui que o `exit 1` de `mutate.ps1 -Anchor 't.Reset()'`
provava que "o transformador do x/text e robusto contra estado residual". Isso fecha a
lacuna com prosa, e estava errado nos dois sentidos.

O motivo real do `exit 1`: **`transform.String` chama `Reset` internamente**, entao o
`Reset` explicito e redundante e remove-lo nao muda comportamento nenhum. A ancora do
brief nomeava uma regra que nao existe como regra separada — defeito do plano, corrigido
em `f1a5d15`.

A regra que o pool realmente promete e **uma instancia por goroutine em voo**, e ela ESTA
verificada. Mutacao feita a mao pelo revisor (trocar o pool por um transformer
compartilhado, restaurado com SHA-256 conferido):

```
WARNING: DATA RACE
golang.org/x/text/transform.(*chain).Transform
  internal/search/analyzer.go:42   Normalize
  internal/search/analyzer_test.go:147  TestNormalizeNaoVazaEstadoEntreUsos
FAIL
```

Numeros do benchmark reconferidos pelo revisor no mesmo commit:
197.650.000 B/op / 128.894 allocs -> 57.521.018 B/op / 46.351 allocs.

`scripts/verify.ps1 -SkipCross -SkipNet` verde. Relatorio em
`.superpowers/sdd/2026-08-05-m7-performance-de-busca/task-80-report.md`.

## Task 81 (M7) — titulo normalizado na indexacao — 2026-08-05

`index.Note` ganhou `TitleNorm`, e `getFieldWeight` deixou de chamar
`Normalize(n.Title)` **por posicao de cada posting**. Commit `30f7739`.

Medido pelo revisor, nao pelo executor — a tarefa nao entregou relatorio, so
codigo:

```
antes  (f1a5d15)   57.505.556 B/op   46.317 allocs/op
depois (30f7739)   53.121.232 B/op   14.116 allocs/op
                       -7,6%            -69,5%
```

Acumulado das tres otimizacoes de busca (78 nao conta, e teste):

```
baseline (56cebb3)   197.650.000 B/op   128.894 allocs/op
agora    (30f7739)    53.121.232 B/op    14.116 allocs/op
                          -73%              -89%
```

`TitleNorm` e escrito em DOIS lugares — `Build` (`index.go:102`) e `Replace`
(`update.go:84`) — ambos pela mesma funcao `normalizeTitleForNote`, que e a
regra desta tarefa. `MoveNote` copia a struct e o campo viaja junto, o que e
correto: ele nao muda o titulo.

Prova de mutacao rodada pelo revisor:

```
Anchor:      TitleNorm:   normalizeTitleForNote(r.note.Title),
Replacement: TitleNorm:   r.note.Title,
Test:        TestTitleNormAcompanhaOTitulo
FAIL -> [OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
```

`TestRankingGolden` identico. `verify.ps1 -SkipCross -SkipNet` verde.

**Mudanca estrutural que nao estava no brief e nao veio em relatorio:** a tarefa
criou `internal/text` (36 linhas) e moveu `Normalize` para la, porque `index`
precisa normalizar e `search` ja importa `index` — importar de volta seria
ciclo. A solucao esta certa e o pacote tem nome legitimo, mas foi decisao de
arquitetura tomada em silencio. Encontrada pelo revisor ao investigar um alerta
de ciclo do gopls, que era estado velho: `go build`, `go vet` e os testes dos
dois pacotes passam limpos.

## Task 82 (M7) — cache de avgdl — REVERTIDA POR MEDICAO — 2026-08-05

Implementada em `78fc6b6`, revertida em seguida. **A reversao e o resultado da
tarefa, nao uma falha dela**: o codigo estava correto e testado.

Medido pelo revisor, `benchstat`, n=6 dos dois lados, cofre sintetico de 5.000
notas, `BenchmarkSearchLimit200`:

```
                  |    sec/op    |    sec/op     vs base         |
SearchLimit200-12   126.5m +/-24%   141.6m +/-22%  ~ (p=0.065 n=6)
                  |     B/op     |     B/op      vs base              |
SearchLimit200-12   50.67Mi +/- 0%   50.60Mi +/- 0%  -0.14% (p=0.002 n=6)
                  |  allocs/op  |  allocs/op   vs base         |
SearchLimit200-12   14.13k +/- 1%   14.14k +/- 1%  ~ (p=0.699 n=6)
```

`~` em tempo e em alocacoes. Os -0,14% de bytes sao significativos e
irrelevantes: e a fatia de 5.000 caminhos que deixou de ser alocada.

**D-M7-3 do plano manda reverter com `~`**, e a razao escrita e "codigo mais
feio sem ganho e divida pura" — aqui, ~50 linhas com dupla checagem sob lock.

O erro foi de quem escolheu o alvo, nao de quem executou. O revisor leu no
perfil que `idx.Paths()` alocava e ordenava por busca, e superestimou: o laco
custa ~250 us contra ~140 ms de busca, **0,2%**. Otimizacao de nao-gargalo.

### O achado que FICA (`9f49a74`, tambem revertido junto, mas a licao vale)

O teste `TestAvgdlInvalidaComAGeracao`, transcrito literalmente do brief,
**nao podia falhar**. Ele comparava `antes[0].Score` com `depois[0].Score` e
exigia que diferissem — mas o corpus passa de duas para tres notas, entao o IDF
se move sozinho e os scores diferem servindo um avgdl obsoleto ou nao.
Medido: removendo `gen == ix.avgdlGen` da condicao de cache, o teste continuava
verde.

Corrigido durante a revisao para afirmar sobre `GetAvgdl` diretamente, e ai a
mutacao reprovava. **Terceiro defeito de desenho do proprio plano exposto pela
execucao** — corpus uniforme na 78, ancora errada na 80, asserçao que nao isola
na 82. Os tres foram achados rodando a prova de mutacao, nunca lendo o teste.

**Se alguem quiser reabrir:** a mudanca e O(N) -> O(1) por consulta e poderia
pagar num cofre muito maior. RNF-09 (20.000 notas) nunca foi medido. Reabrir
exige a medicao, nao o argumento.

## Task 83 (M7) — fetch postings once per query — 2026-08-05

Commit `98c8d0a`. Pre-fetch postings para todos os termos unicos antes do
processamento, reaproveitando na iteracao de pontuacao e no calculo de IDF.
Profile: `Postings` com 371.40 MB foi alocado duas vezes por consulta, uma
vez no laco de scoring (line 61) e de novo para `docsWithTerm` de IDF (lines
102, 106).

**Sem prova de mutacao**: a tarefa remove trabalho duplicado sem criar
invariante nova. O golden inalterado prova que nada quebrou. Golden test
`TestRankingGolden` passou verde nos seis casos; os seis `.tsv` ficaram
identicos (gofmt e go test rodaram limpo).

Medicao de desempenho, `BenchmarkSearchLimit200`, cofre sintetico 5.000 notas,
n=6, antes e depois:

```
Baseline (commit 71b0ea3):
BenchmarkSearchLimit200-12   122264078 ns/op   53176260 B/op   14244 allocs/op
BenchmarkSearchLimit200-12   131448925 ns/op   53095009 B/op   14092 allocs/op
BenchmarkSearchLimit200-12   131386267 ns/op   53144963 B/op   14142 allocs/op
BenchmarkSearchLimit200-12   136534400 ns/op   53126513 B/op   14127 allocs/op
BenchmarkSearchLimit200-12   134491167 ns/op   53125110 B/op   14131 allocs/op
BenchmarkSearchLimit200-12   129407612 ns/op   53129185 B/op   14124 allocs/op

Apos (commit 98c8d0a):
BenchmarkSearchLimit200-12   118616567 ns/op   52900094 B/op   14234 allocs/op
BenchmarkSearchLimit200-12   115683900 ns/op   52872166 B/op   14116 allocs/op
BenchmarkSearchLimit200-12   114568644 ns/op   52870880 B/op   14104 allocs/op
BenchmarkSearchLimit200-12   113407289 ns/op   52884251 B/op   14119 allocs/op
BenchmarkSearchLimit200-12   114283722 ns/op   52866808 B/op   14106 allocs/op
BenchmarkSearchLimit200-12   115384167 ns/op   52873995 B/op   14121 allocs/op
```

**Numero corrigido pelo revisor.** O relatorio estimou "~6-7% de melhora"
olhando as medianas. Passando as MESMAS medicoes pelo `benchstat`, o ganho e
maior:

```
                  |   sec/op    |   sec/op     vs base               |
SearchLimit200-12   131.4m +/-7%   115.0m +/-3%  -12.51% (p=0.002 n=6)
                  |     B/op     |     B/op      vs base              |
SearchLimit200-12   50.67Mi +/-0%   50.42Mi +/-0%  -0.48% (p=0.002 n=6)
                  |  allocs/op  |  allocs/op   vs base         |
SearchLimit200-12   14.13k +/- 1%   14.12k +/- 1%  ~ (p=0.240 n=6)
```

O revisor mediu em paralelo e obteve `~ (p=0.485)` com `+/-7%` de variacao — a
maquina estava sob carga desta sessao. As duas medicoes sao honestas; a do
executor tem mais poder estatistico (`+/-3%`). **Primeira vez nesta batelada em
que conferir mudou a conclusao A FAVOR do delegado**: ele subestimou o proprio
ganho, nao inflou.

Fica registrado que o ganho e de TEMPO, nao de alocacao: bytes caem 0,48% e a
contagem de alocacoes nao muda. O que some e varredura repetida de fatia grande
por consulta.

Contraste deliberado com a Task 82, revertida no mesmo dia: la o `~` era em
tempo E em allocs, e o custo eram ~50 linhas com dupla checagem sob lock. Aqui
ha diferenca significativa na metrica que importa, por 8 linhas e um mapa. A
D-M7-3 nao foi excepcionada — ela dispara em `~`, e aqui nao houve `~`.

## Task 84 (M7) — `note_read` aceitando varios caminhos — 2026-08-05

Commit `301aaff`. `note_read` ganha `paths []string` ao lado de `path`; um
fluxo de pesquisa que lia dez notas com dez idas e voltas de protocolo (para
3,5 ms de trabalho — p95 medido em 345 us) agora paga uma so.

Quatro decisoes do brief, todas mantidas sem re-litigar: `path` e `paths`
juntos e `INVALID_ARGUMENT`, nao precedencia silenciosa; falha parcial nao
derruba o lote — cada item carrega o proprio erro NA MESMA POSICAO; `max_bytes`
vale por nota, nao pelo lote; `paths` recusa acima de 50 itens.

`service.ReadNotes` (`internal/service/read.go`) escreve cada `ReadNoteItem`
por indice — `out[i] = ReadNoteItem{Path: p, Err: err}` — nao por `append`,
entao um item que falha ocupa a MESMA posicao do caminho que o gerou, com o
proprio erro. `ReadNoteItem.MarshalJSON` traduz o `error` Go (sem campo
exportado) para `{"code","message"}`, porque sem isso o cliente veria
`"error":{}` e perderia o motivo.

Em `internal/mcpsrv/tools_read.go`, os dois erros de validacao (campos juntos,
lote acima do teto) sao montados a mao via `noteReadValidationError`, e NAO
devolvidos como `error` Go — devolver `error` faz o SDK descartar
`StructuredContent` por inteiro (ver `toolErr`), e um erro de validacao de
lote ainda se beneficia de `Out` estruturado (`items` de um elemento,
com o proprio erro) que o cliente pode inspecionar por campo. Confirmado por
teste: `res.StructuredContent` nao-nulo nos dois casos, ao contrario dos erros
normais de `note_read` (ex.: `NOTE_NOT_FOUND`), que continuam sem
`StructuredContent`, como antes.

Duas provas de mutacao, uma por regra:

```
pwsh -File scripts/mutate.ps1 -Path internal/mcpsrv/tools_read.go `
  -Anchor 'if len(req.Paths) > maxPathsPorLote {' -Replacement 'if false {' `
  -Test TestNoteReadRecusaLoteAcimaDoTeto -Package ./internal/mcpsrv/
[OK] internal/mcpsrv/tools_read.go restaurado byte a byte (SHA-256 confere).
[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.

pwsh -File scripts/mutate.ps1 -Path internal/service/read.go `
  -Anchor 'out[i] = ReadNoteItem{Path: p, Err: err}' -Replacement 'continue' `
  -Test TestNoteReadMantemPosicaoNoErroParcial -Package ./internal/service/
[OK] internal/service/read.go restaurado byte a byte (SHA-256 confere).
[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
```

A segunda prova o que importava de verdade: com `continue` no lugar do item
de erro, `Items[1]` ficava com `Path` vazio e `Err` nulo — nao a lista
encolhendo (o slice e pre-alocado por indice, `make([]ReadNoteItem,
len(paths))`), mas um item fantasma indistinguivel de dado real na mesma
posicao. Sem checar `Items[1].Path` explicitamente (nao so `len(Items)`), a
mutacao passaria pelo teste sem ser notada.

`docs/TOOLS.md` atualizado: `paths`, `items`, e a nota de que `INVALID_ARGUMENT`
sai com `structuredContent` preenchido ao contrario dos demais erros da tool.

`scripts/check_doc_refs.ps1`: 14 achados, 1 deles em `docs/TOOLS.md` (`total_bytes`,
linha 93) — o mesmo achado ja registrado no ledger da Task 79 (14 achados
naquele commit). O `docs/TOOLS.md` de antes desta tarefa citava `total_bytes`
DUAS vezes (linhas 91 e 94), sob a mesma explicacao repetida — medido direto
contra o corpus antes de editar: 2 achados so daquele arquivo, 15 no total.
Este trabalho consolidou as duas mencoes numa so, voltando ao volume de 14
que a Task 79 tinha registrado — reduziu achado, nao acrescentou. O brief
desta tarefa citava "13 achados conhecidos hoje"; a medicao direta (script
rodado contra o commit anterior a esta tarefa, `607ab88`) nao bate com esse
numero — o valor real medido era 15, nao 13. Registrado aqui para quem ler
o brief depois nao presumir 13 como fato conferido.

`scripts/check_tool_params.ps1`: `[OK] todo parametro declarado e lido em
algum lugar` — 12 structs, 69 parametros, `Paths` entre eles.

`pwsh -File scripts/verify.ps1 -SkipCross -SkipNet`: verde nos 7 passos
(build, `go test -race`, testes de teto de latencia sem `-race`, `go vet`
Windows, `gofmt`, `golangci-lint`, `check_tool_params`) depois de corrigir um
achado real do `golangci-lint` (`context-as-argument` em
`connectTestSession`, helper de teste — `ctx` precisa vir primeiro).

Quatro testes que o brief exigiu, todos verdes: `TestNoteReadBatchKeepsFailedItemAtPosition`
(dez caminhos, um inexistente, posicao preservada), `TestNoteReadPathAndPathsMutuallyExclusive`,
`TestNoteReadRecusaLoteAcimaDoTeto` (51 caminhos), `TestReadNotesMaxBytesPerNote`
(duas notas de mesmo tamanho, `max_bytes` identico em ambas — prova de que o
teto nao e compartilhado pelo lote).

## Task 85 (M7) — cache do indice de metadados, `index_cache.gob` — 2026-08-06

Commit `4d97943b8ab18000ecff8e6503a379ffdd872b20`. Fecha `docs/PRD.md` Q3: o
indice de metadados era reconstruido por varredura e parse do cofre inteiro
em TODA partida, mesmo com o cache de busca quente — so `inverted_cache.gob`
existia. `internal/index/persist.go` (novo) e `persist_codec.go` (novo)
implementam o segundo cache que a decisao de 2026-07-29 ja previa e nunca
tinha sido escrito.

**Decisao que a tarefa tinha que acertar: reaproveitar a TECNICA do codec de
busca, nao importar codigo.** `search` importa `index`, nao o contrario, entao
nao ha como `index` importar tipos privados de `persist_codec.go` de `search`
— um segundo arquivo era inevitavel. O que teria sido errado era um TERCEIRO
formato: `escritor`/`leitor` em `internal/index/persist_codec.go` usam a mesma
tecnica (varint, string por comprimento+bytes, portao de versao antes de
qualquer campo de layout), documentado no topo do arquivo com o motivo de nao
ser codigo compartilhado.

**Divergencia deliberada do codec de busca: um CRC32 (IEEE) de rodape.** O
cache de busca confia nos totais declarados e nos limites por campo pra pegar
corrupcao — suficiente quando o pior caso e busca com menos resultado. Aqui o
pior caso e outro (o proprio brief: "um cache de metadados errado serve nota
errada, nao nota lenta"): um byte trocado dentro do PAYLOAD de uma string
(nao um comprimento) nao violaria limite nenhum e decodificaria "com sucesso"
um titulo ou caminho corrompido. O checksum fecha isso de forma
deterministica, nao probabilistica — testado contra o binario real, nao so a
suite (ver abaixo).

**`byAlias`, backlinks e a resolucao de cada link NAO sao persistidos.** Sao
recalculados no load chamando as MESMAS tres funcoes que `Build` chama depois
de indexar — `buildAliasMap`, `resolveAllLinks`, `buildBacklinks`. A licao ja
paga neste projeto (`byAlias` minusculo numa via e cru na outra, `[[STJ]]`
resolvendo para nota apagada) e que toda chave derivada calculada em dois
lugares diverge, e a divergencia aparece no caminho menos usado. Persistir os
tres separadamente seria exatamente essa segunda via. Para viabilizar isso,
`insert()` (`internal/index/index.go`, chamado por `Build`) foi refatorado
para dividir a publicacao nos indices derivados (`notes`/`assets`,
`lowerPath`, `byName`, `tags`) em
`publishNoteLocked`/`publishAssetLocked`/`publishNameLocked`
(`internal/index/index.go`) — o carregamento do cache chama os MESMOS
metodos, em vez de reescrever a logica de povoamento numa segunda funcao.

**O cabecalho confere COBERTURA, nao so versao** — a regra que o cache de
busca aprendeu na marra. `LoadIndexCache` compara `h.NoteCount`/`h.AssetCount`
contra o indice DECODIFICADO, nao contra o que o cabecalho promete, e recusa
com `ErrIndexCachePartial` se divergir.

**Invalidacao por mtime e tamanho por arquivo** (`Index.VerifyFreshness`,
`internal/index/persist.go`): uma varredura leve (sem parse, sem leitura de
conteudo — so `vault.Walk`, que so faz `Stat`) confere que TODO arquivo do
disco bate em tamanho e mtime com o que o cache tem, e que a contagem de
arquivos e a mesma dos dois lados (pega adicao pura, que nao diverge em
tamanho/mtime de nada porque nao ha entrada pra comparar). Qualquer
divergencia cai pra `idx.Build` como antes; nao ha reparo parcial —
reconciliar arquivo a arquivo e trabalho do watcher (RF-05), nao do boot.

`cmd/gobsidian/serve.go`: `carregarIndiceDoCache` tenta o cache antes de
`idx.Build`; `indexMS` continua medindo exatamente o mesmo trecho de antes
(RNF-01), so que agora pode terminar em 200ms em vez de 900ms. Campo novo no
log "servidor pronto": `index_origin` (`cache` ou `build`).

**Teste diferencial, no molde do que pegou o defeito do `DocLength` no cache
de busca:** `TestIndiceDeMetadadosRecarregadoEIdentico`
(`internal/index/persist_test.go`) constroi um indice do zero, salva, carrega,
e compara CAMPO A CAMPO — `Paths()`, `NoteCount()`, `AssetCount()`,
`TotalSize()`, `Tags("", 0)`, e por caminho `Get()` (via `reflect.DeepEqual`
na `*Note` inteira), `Backlinks()`, `ResolvePath()` do nome curto. O corpus
cobre as seis armadilhas que o brief exigiu na mesma fixture: nota com alias
(`aliases: [P3]`), nota com backlink (`Origem.md` recebe dois links da mesma
origem), nota com ancora quebrada (`[[Origem#NaoExiste]]`), nota vazia (zero
bytes — testa nil vs fatia/mapa vazio em quase todo campo de `Note` ao mesmo
tempo), anexo, e nome que colide em caixa
(`ResolvePath("civil/ponto 03.md")` contra `Civil/PONTO 03.md`, batendo nos
dois indices). `Resolved`/`Via`/`State` de cada link NAO sao persistidos e
batem depois do load mesmo assim — prova de que recalcular pela mesma funcao
funciona.

Achado durante o design, corrigido antes de virar bug: `yaml.v3` decodifica
inteiro de frontmatter como Go `int` (64 bits nesta plataforma), nunca
`int64`, mas `query.go` faz asserção de tipo `.(int)` sobre o Frontmatter — um
codec que decodificasse de volta como `int64` faria `note_list` filtrar por
numero de forma silenciosamente diferente entre indice recem-construido e
recarregado do cache. O codec de valor generico
(`internal/index/persist_codec.go`, tags `valInt`/`valInt64` separadas)
preserva o tipo exato. `time.Time` de frontmatter (datas do YAML) e gravado
via `MarshalBinary`/`UnmarshalBinary`, nao um par (offset, UTC assumido) —
preserva fuso explicito (`+05:00` etc.), que `time.Parse` produz com
`Location` != UTC e que `reflect.DeepEqual` distingue.

Prova de mutacao, exatamente a que o brief pediu:

```
pwsh -File scripts/mutate.ps1 -Path internal/index/persist.go `
  -Anchor 'if h.NoteCount != idx.NoteCount() {' -Replacement 'if false {' `
  -Test TestCacheDeMetadadosParcialERecusado -Package ./internal/index/
[...] Mutando internal/index/persist.go
      - if h.NoteCount != idx.NoteCount() {
      + if false {
[...] go test -race -run TestCacheDeMetadadosParcialERecusado ./internal/index/
----------------------------------------------------------------------
--- FAIL: TestCacheDeMetadadosParcialERecusado (0.04s)
    persist_test.go:276: LoadIndexCache deveria recusar um cabecalho que declara mais notas do que o corpo traz
FAIL
FAIL	github.com/jonyd/gobsidian/internal/index	0.942s
FAIL
----------------------------------------------------------------------
[OK] internal/index/persist.go restaurado byte a byte (SHA-256 confere).
[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
```

**Medicao: nao foi possivel confirmar o cofre real exato (3.149 notas, 109 MB)
citado nas secoes anteriores de `OPERACAO.md` dentro desta tarefa** — pedido
enviado ao team-lead, sem resposta a tempo. A medicao abaixo e num cofre
SINTETICO gerado na mesma escala (`scripts/gen_vault.ps1 -Notes 3149
-BodyKB 35 -Seed 42`: 3.149 notas, 50 anexos, 107,93 MB), registrado como tal
em vez de apresentado como o mesmo dado. `docs/OPERACAO.md` e `docs/PRD.md`
Q3 tem a tabela completa e a mesma ressalva.

`index_ms`, cinco partidas sem cache (cache apagado antes de cada uma,
equivalente ao comportamento de antes desta tarefa): **7736, 886, 877, 901,
852 ms** — a primeira paga leitura fria de disco do SO para os 108 MB; as
quatro seguintes ficam em 852–901 ms, a mesma faixa ja registrada pro cofre
real (832–1183 ms).

`index_ms`, cinco partidas com cache presente e valido: **208, 236, 213, 213,
282 ms.**

**RNF-02 (≤ 300 ms) passa a ser atingido nesta escala** — as cinco partidas
com cache ficaram todas abaixo do teto, de 3 a 4 vezes mais rapido que sem
cache. Precisa ser remedido no cofre real para fechar com certeza, porque o
sintetico e aproximacao, nao o mesmo dado.

Tres cenarios verificados contra o BINARIO REAL, nao so a suite:

1. **Com cache:** `index_ms` cai de ~870 ms pra ~230 ms — tabelas acima.
2. **Cache apagado e servidor reiniciado:** reconstroi sem erro,
   `notes=3149 assets=50` batendo com a varredura, regrava o cache no fim.
3. **Um byte corrompido no meio do arquivo** (flip de um byte no offset
   451328 de 902657, bem no meio): log real da partida —
   `msg="cache de indice de metadados descartado" err="index cache file
   corrupted"`, seguido de `index_ms=868` (a reconstrucao, nao um indice
   com metadado corrompido servido em silencio).

`golangci-lint run ./internal/index/... ./cmd/...`: `0 issues.` — um achado
real corrigido no caminho (`revive`/stutter: `index.IndexCacheHeader`
renomeado pra `index.CacheHeader`, espelhando `search.CacheHeader`).

`pwsh -File scripts/verify.ps1 -SkipCross -SkipNet`: verde nos 7 passos.

`go test -count=1 ./internal/index/ ./internal/search/ ./internal/service/
./cmd/...`: verde. `go test -race -count=1 ./internal/index/`: verde.
`go test -count=1 -run TestRankingGolden ./internal/service/`: verde,
golden de ranking IDENTICO — nenhuma reordenacao de acumulacao de ponto
flutuante, nenhum `-update`.

O que ficou de fora: remedir no cofre real (bloqueado por nao ter o caminho
a tempo); nao foi implementada reconciliacao parcial (reparar so as notas
que mudaram em vez de reconstruir tudo) — fora do escopo do brief, que so
pedia invalidacao binaria (usar o cache inteiro ou reconstruir inteiro), e a
reconciliacao arquivo-a-arquivo ja e trabalho do watcher (RF-05).

## Task 85 — remedicao no cofre REAL pelo revisor — 2026-08-06

A tarefa mediu num cofre sintetico de mesma escala, e disse isso. O caminho do
cofre real chegou tarde na caixa dela. Remedido aqui, seis partidas, todas com
`index_origin=cache`:

```
451  472  408  411  371  458  ms
```

Baseline no MESMO cofre, antes da tarefa (`0174778`): 1267, 1396, 1192 ms.
**Queda de ~66%.**

**RNF-02 segue NAO ATINGIDO**: 371-472 ms contra teto de 300 ms.

O sintetico dizia 208-282 ms e "requisito atingido". O real diz que nao. A
diferenca e o OneDrive: `VerifyFreshness` percorre todo arquivo do cofre para
conferir mtime e tamanho, e e essa varredura que sobra — nao a decodificacao do
cache. Cofre sintetico e cofre real nao sao intercambiaveis para este numero, e
a tarefa acertou em marcar a ressalva em vez de fechar o requisito.

`docs/OPERACAO.md` e `README.md` corrigidos com o numero do cofre real.

Conferido tambem que o caminho do cofre — que carrega nome de empregador — nao
vazou para commit nenhum: `git log -p` dos dois commits da tarefa, zero
ocorrencias.

**Corroborado por uma segunda medicao independente**, feita pela propria
Task 85 no mesmo cofre e na mesma janela de tempo (a mensagem com o caminho
chegou tarde, mas nao tarde demais pra remedir antes de reportar): cinco
partidas com cache, **547, 397, 494, 443, 438 ms** — faixa proxima da do
revisor (371-472 ms), mesma conclusao (RNF-02 nao atingido, ~3x mais rapido
que sem cache). As duas medicoes concordam no que importa: o numero do cofre
sintetico nao serve pra fechar este requisito.

## Colisao de escrita entre revisor e executor no mesmo worktree — 2026-08-06

Registrado como defeito de PROCESSO, nao de codigo. Nada se perdeu; a atribuicao
e que ficou errada.

O revisor e o executor da Task 85 editaram `docs/OPERACAO.md` ao mesmo tempo, no
mesmo worktree. O revisor commitou com `git add docs/OPERACAO.md`, caminho
explicito — e mesmo assim levou junto a secao "Remedicao no cofre real" que o
executor tinha acabado de gravar em disco e ainda nao commitado.

Resultado: `6f5a842` mudou 78 linhas em `OPERACAO.md` quando a edicao do revisor
tinha cerca de 40. A mensagem do commit descreve so a metade dele. O conteudo
final ficou consistente **por acidente** — as duas secoes eram complementares e
chegaram a mesma conclusao. Se tivessem se contradito, o commit teria gravado a
contradicao sem ninguem notar.

**A licao contradiz uma defesa que este projeto ja tinha.** A regra escrita e
"nunca `git add -A`, adicione por caminho explicito". Caminho explicito **nao
basta** quando outro processo escreve no mesmo arquivo: `git add <arquivo>`
estagia o arquivo inteiro, incluindo o que nao e seu.

Regra nova, para o resto desta batelada e para qualquer sessao com subagentes
no mesmo worktree:

- **`git diff <caminho>` antes de `git add <caminho>`**, e conferir que o diff
  e so o que voce fez. E a unica forma de distinguir "meu arquivo" de "meu
  arquivo mais o de outro".
- Enquanto um executor estiver rodando, o revisor **nao edita** os mesmos
  arquivos que a tarefa dele toca. Medir e conferir, sim; escrever, depois.

O executor percebeu a colisao sozinho, removeu a propria secao duplicada do
ledger e deixou so uma nota de corroboracao — comportamento certo, e foi ele
quem tornou a colisao visivel.

## Task 86 — re-resolucao de links dirigida, nao global — 2026-08-06

`reprocessLinksLocked` rodava em todo evento do watcher, sobre todas as
notas — 20,35 ms de mediana contra o teto de 20 ms do RNF-06, medido num
cofre de 5.000 notas. Substituido por um indice reverso, `citantesPorNome`
(`internal/index/index.go`), que mapeia a chave normalizada de um alvo de
link (`nomeChave`, `internal/index/resolve.go`) para as notas que o citam —
resolvido OU quebrado, porque um `[[foo]]` sem arquivo tem de ficar
descobrivel sob "foo" para que criar `foo.md` conserte o link.
`Replace`/`Remove`/`MoveNote` (`internal/index/update.go`) agora reprocessam
so as notas citantes das chaves que uma mudanca de identidade pode afetar
(caminho e aliases), em vez de varrer o cofre inteiro.

**Decisao de risco.** Esta e a area onde um erro nao aparece em teste
nenhum e sai como resposta confiante: link com `state=ok` apontando para
nota que nao existe mais, exatamente a classe de defeito que a divergencia
de `byAlias` produziu em 2026-07-28. Guarda: escrita e leitura de
`citantesPorNome` passam SEMPRE por `nomeChave` — nunca o alvo cru — mesma
disciplina de `aliasKey`.

**RNF-06 ATINGIDO.** Mediana 20,35 ms → **334,87 µs** (p95 544,87 µs),
medido por `TestRNF03_RNF05_RNF06` (`internal/service/rnf_leves_test.go`) no
cofre sintetico de 5.000 notas. Essa mesma medicao precisou de correcao: com
`lote=1` a chamada individual ficou rapida demais para o relogio do Windows
resolver (mediana virava `0s`), o mesmo problema que RNF-03 e RNF-05 ja
tinham corrigido com lote — RNF-06 foi ajustado para `lote=20`, mesma tecnica,
nao afrouxamento de guarda.

**`BenchmarkReplaceSingleFile`** (`internal/index/bench_test.go`, criado
nesta tarefa — nao havia benchmark de reindexacao de arquivo unico antes
dela), `-count=6` no cofre de 5.000 notas:

```
                     │ antes (task86-bench-before.txt) │        depois (task86-bench-after.txt)         │
                     │              sec/op              │   sec/op     vs base                            │
ReplaceSingleFile-12          19658.0µ ± 16%                332.9µ ± 30%  -98.31% (p=0.002 n=6)
```

**Diferencial contra o caminho global — TestReresolucaoDirigidaIgualAGlobal**
(`internal/index/reindex_test.go`): sequencia criar / renomear / apagar /
criar de novo com alias, aplicada em paralelo a dois indices — um so com a
re-resolucao dirigida (via `Replace`/`Remove`/`MoveNote` normais), o outro
com uma passada global forcada por cima a cada evento (`resolveAllLinks` +
`buildBacklinks`, as MESMAS funcoes que `Build` e `LoadIndexCache` usam — o
caminho global fica so no teste, como oraculo, nao no produto). Comparados
campo a campo: `Get()` de cada nota (Resolved/Via/State de cada link) e
`Backlinks()` de cada caminho. Identicos. `TestReresolucaoDirigidaCobreAliases`
prova a decisao de que alias conta: criar uma nota com `aliases: [STJ]` faz
um `[[STJ]]` que ja estava quebrado noutra nota passar a resolver.

**Duas provas de mutacao, as duas saida `0` (regra verificada):**

```
pwsh -File scripts/mutate.ps1 -Path internal/index/resolve.go `
  -Anchor 'for _, alias := range n.Aliases {' -Replacement 'for _, alias := range []string(nil) {' `
  -Test TestReresolucaoDirigidaCobreAliases -Package ./internal/index/
```
```
--- FAIL: TestReresolucaoDirigidaCobreAliases (0.01s)
    reindex_test.go:60: link [[STJ]] depois de criar Tribunal.md com alias STJ = {...State:target_missing},
    quer State=LinkOK Resolved=Tribunal.md Via=ViaAlias
[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
```

```
pwsh -File scripts/mutate.ps1 -Path internal/index/update.go `
  -Anchor 'ix.citantesPorNome[nomeChave(alvo)]' -Replacement 'ix.citantesPorNome[alvo]' `
  -Test TestReresolucaoDirigidaIgualAGlobal -Package ./internal/index/
```
```
--- FAIL: TestReresolucaoDirigidaIgualAGlobal (0.03s)
    reindex_test.go:160: Citante.md: link 0 (alvo "Original") diverge — dirigido={Resolved:"Renomeada.md" Via:name State:ok},
    global={Resolved:"" Via: State:target_missing}
    reindex_test.go:160: Citante.md: link 1 (alvo "Apelido") diverge — dirigido={Resolved:"" Via: State:target_missing},
    global={Resolved:"Renomeada.md" Via:alias State:ok}
    reindex_test.go:160: Nova.md: link 0 (alvo "Original") diverge — dirigido={Resolved:"Renomeada.md" Via:name State:ok},
    global={Resolved:"" Via: State:target_missing}
    reindex_test.go:160: Renomeada.md: 2 backlinks no dirigido, 1 no global
[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
```

A segunda prova reproduz, ao vivo, exatamente o padrao de defeito que a
tarefa foi escrita para nunca reintroduzir: link com `state=ok` apontando
para caminho errado assim que a chave de escrita deixa de ser normalizada.

**Verificacao:** `go test -count=1` e `go test -race -count=1` verdes em
`internal/index`, `internal/watcher`, `internal/service`, `internal/mcpsrv`.
`TestRankingGolden` identico — nenhuma reordenacao de acumulacao de ponto
flutuante, nenhum `-update`. `pwsh -File scripts/verify.ps1 -SkipCross
-SkipNet`: `[OK] Bateria completa`, incluindo `golangci-lint` na v2.12.2
(mesma do CI).

**Escopo cumprido integralmente.** Nenhuma reordenacao da resolucao de link
foi necessaria alem do que a tarefa pediu.

Commit: `d6fb7d0278ba50122875213644fd7d528764d44a` — "perf(index): re-resolve
only the links a change can affect", `git cat-file -t` confirma `commit`.

## Task 87 — fechamento da Parte I do M7: relatorios, ledger e medicao final — 2026-08-09

Nao envia codigo. Mede o efeito acumulado das Tasks 78 a 86, corrige
`docs/OPERACAO.md` e `README.md`, e fecha esta parte do marco.

**Sem prova de mutacao: esta tarefa nao altera regra de codigo nenhuma**, entao
nao ha regra para provar por mutacao.

### O que ja tinha numero e nao foi remedido

Instrucao explicita do lote: nao remedir o que ja tem numero, a menos que haja
motivo para suspeitar dele. Nenhum dos dois casos abaixo deu esse motivo.

- **RNF-02** (boot com cache valido, teto 300 ms): **371-472 ms** num cofre
  real de 4.165 notas, medido pela Task 85 (`4d97943`) e corroborado de forma
  independente (`6f5a842`). Nenhuma das Tasks 78-86 tocou
  `Index.VerifyFreshness` nem `LoadIndexCache`, os dois pontos que a analise
  anterior aponta como o custo residual. **NAO ATINGIDO.**
- **RNF-06** (reindexacao de arquivo unico, teto 20 ms): **334,87 us**
  mediana, p95 544,87 us, commit `d6fb7d0`, com duas provas de mutacao ja
  coladas na secao "Task 86" deste mesmo ledger. **Atingido.**

### O que esta tarefa mediu: RNF-04 e RNF-07, cofre sintetico de 5.000 notas

A tabela de RNF-04 (`limit: 200` e mais sete formatos) e a de RNF-07 (RSS
quente/fria) desta batelada foram estabelecidas no cofre sintetico gerado por
`scripts/gen_vault.ps1 -Notes 5000 -Seed 42` (2026-08-01), nao no cofre real —
diferente de RNF-02, que so tem numero no cofre real. Remedir a mesma tabela
exige o mesmo cofre.

**Achado antes de remedir:** `%TEMP%\vault_5000` ja existia na maquina, mas
com 3 anexos e 6,7 MB — nao bate com o cofre documentado (50 anexos, 1,27 MB,
10.101 links, 1.518 quebrados). Regenerado com o comando exato ja documentado:

```
pwsh -File scripts/gen_vault.ps1 -Out "$env:TEMP\vault_5000" -Notes 5000 -Seed 42
```
```
[OK] Cofre sintético gerado em C:\Users\jonyd\AppData\Local\Temp\vault_5000
[*] Notas: 5000
[*] Anexos: 50
[*] Tamanho total: 1.27 MB (1329475 bytes)
[*] Links totais: 10101
[*] Links quebrados: 1518
```

Confere com a geracao de 2026-08-01 (mesma semente, mesmo script).

**RNF-04**, via `TestScale5000_RNF01_RNF02_RNF07_RNF04`
(`internal/service/rnf5000_test.go`), que chama `svc.Search` — a mesma funcao
que `internal/mcpsrv/tools_read.go` chama para a tool `vault_search`. Tres
rodadas independentes, sem `-race`:

```
go test ./internal/service/ -run TestScale5000_RNF01_RNF02_RNF07_RNF04 -v -count=1
```
```
=== RNF-04 (Latencia vault_search p95 5.000 notas) ===
  termo amplo, limit default     mediana 19.75ms    p95 29.517ms
  dois termos                    mediana 9.9605ms   p95 14.2779ms
  termo seletivo                 mediana 5.7207ms   p95 11.989ms
  filtro de pasta                mediana 20.5042ms  p95 25.7847ms
  filtro de tag                  mediana 20.5481ms  p95 31.3671ms
  frase exata                    mediana 13.0583ms  p95 19.7018ms
  trecho maximo                  mediana 7.9574ms   p95 13.2298ms
  limit maximo do schema         mediana 107.7772ms p95 122.5453ms
--- PASS: TestScale5000_RNF01_RNF02_RNF07_RNF04 (11.19s)
```

Rodadas 2 e 3 (so a linha de `limit: 200`, a unica que nao fecha): p95
120,0577 ms e 119,1827 ms. As outras sete linhas variaram entre 8 e 34 ms nas
tres rodadas, todas dentro do teto de 100 ms.

**Sete formatos de oito atingidos, com mais folga que antes** (o pior deles,
"termo amplo", tinha p95 de 94,54 ms nas Tasks anteriores a este lote; hoje
29-33 ms). `limit: 200` caiu de 181,25 ms para a faixa 119-123 ms — queda de
~33%, e **segue NAO ATINGIDO** contra o teto de 100 ms.

**RNF-07**, via `WorkingSet64` do processo real, mesmo cofre. `measure.ps1`
nao aceita `--cache-dir`, e medir frio vs quente exige controlar se o cache ja
existe antes do boot — usado um script local equivalente (mesma sequencia de
handshake MCP, `SettleMs=8000`, 5 amostras a 200ms, reporta o pico), nao
commitado, so chamando o binario compilado com `--cache-dir` explicito.

Cache quente (3 partidas):

```
notes=5000 assets=50 index_ms=280 index_origin=cache RSS_pico_MB=38.10
notes=5000 assets=50 index_ms=283 index_origin=cache RSS_pico_MB=37.95
notes=5000 assets=50 index_ms=267 index_origin=cache RSS_pico_MB=38.10
```

Cache frio (3 partidas, cache apagado antes de cada uma):

```
notes=5000 assets=50 index_ms=508 index_origin=build RSS_pico_MB=54.82
notes=5000 assets=50 index_ms=513 index_origin=build RSS_pico_MB=54.76
notes=5000 assets=50 index_ms=538 index_origin=build RSS_pico_MB=54.69
```

**RNF-07 estava NAO ATINGIDO (67,08 MB quente / 112,96 MB frio, 2026-08-01) e
agora esta ATINGIDO nos dois regimes**: 37,95-38,10 MB quente (37% de folga
sobre o teto de 60 MB) e 54,69-54,82 MB frio (9% de folga).

### Tabela de fechamento — os quatro RNFs

| RNF | Metrica (alvo) | Antes | Depois (2026-08-09) | Estado |
|---|---|---|---|---|
| **RNF-02** | Boot com cache valido (<= 300 ms) | 1192-1396 ms sem `index_cache` | **371-472 ms** (cofre real, 4.165 notas; nao remedido — numero da Task 85, 2026-08-06) | **NAO ATINGIDO** |
| **RNF-04** | `vault_search` p95 (<= 100 ms) | `limit: 200` em 181,25 ms; 7 outros formatos ja atingidos | 7 de 8 formatos atingidos (7-33 ms); `limit: 200` em **119-123 ms** | **Parcial (1 de 8 formatos NAO ATINGIDO)** |
| **RNF-06** | Reindexacao de arquivo unico (<= 20 ms) | mediana 20,35 ms, p95 30,14 ms | **334,87 us** mediana, p95 544,87 us (Task 86, nao remedido) | **Atingido** |
| **RNF-07** | RSS em repouso (<= 60 MB) | 67,08 MB quente / 112,96 MB frio | **37,95-38,10 MB** quente / **54,69-54,82 MB** frio | **Atingido** |

Dos quatro RNFs que fechavam o M6 como nao atingidos, dois seguem nao
totalmente atingidos (RNF-02, e RNF-04 num unico formato de oito) e dois
passaram a ser atingidos (RNF-06 pela Task 86, RNF-07 como efeito colateral
das otimizacoes de busca das Tasks 78-85). Nenhum teto foi afrouxado.

### Correcao de doc encontrada nesta tarefa

A tabela "Tabela completa dos RNFs" em `docs/OPERACAO.md` ainda listava RNF-06
como NAO ATINGIDO com o numero antigo (20,35 ms) — a Task 86 tinha corrigido o
codigo e registrado o resultado no ledger, mas nao tinha atualizado essa
tabela, que e a que `README.md` referencia. Corrigido junto com as linhas de
RNF-04 e RNF-07.

`README.md` dizia "Quatro requisitos nao estao atingidos" — hoje sao dois
(RNF-02 e RNF-04 parcial); a tabela e o texto de aviso foram atualizados para
bater, e a contagem foi conferida contando as linhas com ❌/⚠️ na tabela
(2), nao assumida.

### Verificacao

```
pwsh -File scripts/verify.ps1 -SkipCross -SkipNet
```
```
[OK] go build
[OK] go test -race
[OK] go test (tetos de latencia, sem -race)
[OK] go vet (windows)
[OK] gofmt
[OK] golangci-lint
[OK] check_tool_params
[OK] Bateria completa. Pode commitar.
```

`golangci-lint version` confirmado em `2.12.2` (mesma do CI) antes de aceitar
o zero acima.

`pwsh -File scripts/check_doc_refs.ps1`: 10 achados, todos pre-existentes em
`ARCHITECTURE.md`, `ESTRUTURA.md`, `TOOLS.md` e `WINDOWS.md` — nenhum em
`docs/OPERACAO.md` nem `README.md`, os dois arquivos que esta tarefa tocou.

`pwsh -File scripts/audit_reports.ps1`: 45 achados, todos pre-existentes em
relatorios de tarefas anteriores (Tasks 1-79) e em tres linhas do ledger de
2026-07 (duas de SHA-nao-confere nas Tasks 4 e 6, uma de SHA-fantasma com um
identificador de exemplo que nao corresponde a commit nenhum) — nenhum deles
introduzido por esta tarefa. O script nao verifica `docs/OPERACAO.md` nem
`README.md` (so `task-*-report.md` e `progress.md`), entao a tabela e o aviso
atualizados nesses dois arquivos nao passam por ele; a evidencia deles esta
colada acima e neste ledger.

Todo SHA citado neste registro conferido com `git cat-file -t`:

```
$ git cat-file -t 4d97943
commit
$ git cat-file -t 6f5a842
commit
$ git cat-file -t d6fb7d0
commit
```

Validacao UTF-8 dos dois arquivos `.md` tocados:

```
$ python -c "open('docs/OPERACAO.md',encoding='utf-8').read()"
[OK] UTF-8 valido
$ python -c "open('README.md',encoding='utf-8').read()"
[OK] UTF-8 valido README
```

### Escopo

Cumprido integralmente: os quatro RNFs tem numero antes/depois, `docs/OPERACAO.md`
e `README.md` corrigidos, contagem de requisitos nao atingidos conferida por
contagem real (nao suposta), e a correcao de doc que a Task 86 tinha deixado
pendente (RNF-06 na tabela principal) foi paga junto.

Esta tarefa nao produz commit de codigo — so documentacao e ledger, no unico
commit que segue esta entrada no historico, mensagem "docs(ledger): record M7
and the measured state of the four RNFs".

## Task 88 (M7) — indice de busca sob demanda — 2026-08-10

`cmd/gobsidian/serve.go` chamava `prepararIndiceDeBusca` incondicionalmente em
toda partida. Uma sessao que nunca chama `vault_search` pagava o indice inteiro
assim mesmo — e a maioria das sessoes de assistente le e escreve nota sem nunca
buscar. Cada sessao de host MCP abre um processo desses.

Agora a carga e disparada pela PRIMEIRA chamada de `vault_search`. Ate la o
indice fica marcado como em construcao e a tool responde `INDEX_BUILDING`, nunca
lista vazia. `--eager-search` liga o comportamento antigo, para quem roda num
script que so busca.

O watcher continua comecando na partida. So o carregamento do indice de busca e
adiado — adiar o watcher faria eventos se perderem, e o unico anteparo seria a
reindexacao no boot seguinte.

### Medicao — feita pelo REVISOR, nao pelo executor

O executor ficou bloqueado esperando o caminho do cofre real, que carrega nome
de empregador e nao entra em artefato versionado. Duas mensagens com o caminho
nao chegaram a caixa dele. O revisor mediu com o codigo dele, compilado da
arvore antes do commit.

Cofre real de 4.490 notas, tres partidas, **sem nenhuma chamada a
`vault_search`**:

```
RSS 165,7 MB   index_ms=1850   <- cache de metadados desatualizado, reconstruiu
RSS 125,2 MB   index_ms=622
RSS 125,0 MB   index_ms=507
```

Baseline no mesmo cofre e maquina, cache quente, antes da mudanca:

```
RSS 501,9 / 502,1 / 501,9 MB
```

**~502 MB -> ~125 MB numa instancia que nunca busca: queda de 75%.**

A primeira amostra e descartada de proposito: o cofre mudou entre as medicoes e
aquela partida pagou uma reconstrucao do cache de metadados. As duas seguintes
sao o numero em repouso.

Isto ataca o problema que motivou a Parte II: cada sessao de host MCP abre um
processo, e duas sessoes do Claude custavam cerca de 1 GB. Duas sessoes que so
leem e escrevem nota passam a custar ~250 MB.

### Medicao — tempo ate a primeira busca (executor, corpus sintetico)

Medido depois do desbloqueio, sem o cofre real: um corpus sintetico de escala
parecida (`scripts/gen_vault.ps1 -Notes 4490 -BodyKB 30`, 132,1 MB, mesma
ordem de grandeza do cofre real) serve para este numero — o brief pede o
tempo, nao o cofre especifico.

Sequencia: boot com `--cache-dir` vazio para aquecer os dois caches (ignorado
na medicao — e o caminho de construcao do zero, 13,4 s), depois tres partidas
contra o MESMO cache ja quente, cada uma com uma unica chamada de
`vault_search` logo apos o handshake MCP, cronometrada do envio da requisicao
ate a linha de resposta:

```
index_ms=367 (cache)   PRIMEIRA_BUSCA_MS=848
index_ms=345 (cache)   PRIMEIRA_BUSCA_MS=700
index_ms=365 (cache)   PRIMEIRA_BUSCA_MS=703
```

**700-848 ms, bem abaixo do teto de 3 s.** Ordem de grandeza da referencia do
revisor para a carga do indice de busca sozinha (648-676 ms) mais o
round-trip do JSON-RPC e a geracao de snippets da propria busca — a inferencia
registrada acima batia com o numero real.

### Prova de mutacao (rodada pelo revisor)

```
Anchor:      if err := s.garanteIndiceDeBusca(ctx); err != nil {
Replacement: if err := error(nil); err != nil {
Test:        TestBuscaPreguicosaCarregaUmaVezESoUmaVez
--- FAIL: carregou 0 vezes sob concorrencia, quer 1
--- FAIL: segunda busca falhou: o Once travou o erro para sempre
[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
```

O teste cobre os DOIS defeitos que o adiamento introduz: carregar N vezes sob
concorrencia, e nunca mais tentar depois de uma falha. `Once` e sobre "ja
disparei", nao sobre "ja consegui".

`TestRankingGolden` identico. `verify.ps1 -SkipCross -SkipNet` verde.

## Task 89 (M7) — arena de posicoes mapeada do arquivo — 2026-08-10

Commit: `b92d58a0da17a98ad9605e535f2de971a031900f`
("perf(search): map the position array from the cache file")

```
$ git cat-file -t b92d58a
commit
```

Formato do cache: 5 -> 6 (`GBS5` -> `GBS6`). `escreveCache` grava, depois do
corpo em varint de sempre, uma secao fixa de posicoes em 16 bytes cada
(alinhada em 8) e um rodape de 24 bytes no fim do arquivo (assinatura +
offset da secao fixa + contagem). O varint continua sendo o que a
decodificacao integral usa; a secao fixa so existe para quem mapeia.
`internal/search/mmap.go` (guarda `dentroDoCofre` + orquestracao) e
`mmap_windows.go`/`mmap_unix.go` (build tag, `syscall` da std lib, sem
dependencia nova) fazem o mapeamento. `LoadInvertedCache` tenta mapear
primeiro; qualquer recusa cai pro `os.ReadFile` + decodificacao integral —
nunca mapeia lixo.

### A decisao que a tarefa tinha de acertar: opcao (a), medida

Pre-decidido: comecar pela opcao (a) — segunda secao fixa, custa disco — e
medir. Se o RSS agregado de tres instancias que buscaram nao caisse pelo
menos 30%, parar e reportar, sem partir pra opcao (b) (mapear o arquivo
comprimido, decodificar sob demanda).

**Bloqueio e o que mudou entre as duas mensagens do brief.** O primeiro envio
do caminho do cofre real nao chegou pela caixa do executor duas vezes — mesmo
problema que ja tinha custado a medicao da Task 88 ao revisor. A segunda
mensagem trouxe o caminho E um alerta: uma sessao real do usuario roda
`C:\Program Files\gobsidian\gobsidian.exe` v1.0.1 (formato `gob`, anterior ao
codec binario) contra o `--cache-dir` PADRAO do cofre real, e as duas versoes
ficavam disputando o mesmo arquivo — cada uma achava o formato da outra
incompativel, reconstruia, e a seguinte reconstruia de novo. Medir ali daria
numero de reconstrucao, nao de repouso. A correcao: `--cache-dir` dedicado
(`%TEMP%\gob89_cache_old` e `%TEMP%\gob89_cache_new`, fora do cofre e fora do
caminho padrao), pre-populado por UMA partida ignorada na medicao, e as
partidas medidas todas com `--eager-search` (Task 88) em vez de handshake MCP
manual — carrega o indice do mesmo jeito que uma busca real carregaria.

**Baseline "antes" medido de verdade, nao extrapolado.** Em vez de supor que
tres instancias do binario anterior custariam ~3x uma, foi criado um worktree
detached em `db68915` (HEAD antes desta tarefa comecar), compilado, e as
mesmas tres partidas rodadas contra um cache formato-5 pre-populado no cofre
real.

### Medicao — RSS, cofre real de 4.490 notas

Tres partidas por cenario, `--eager-search`, cache aquecido por uma partida
anterior ignorada. Metrica via `Win32_PerfFormattedData_PerfProc_Process`
(`WorkingSet`, `WorkingSetPrivate`), casada por `IDProcess`:

```
=== UMA instancia, OLD (formato 5, sem mmap) ===
PID 37232: WorkingSet=584,9 MB   WorkingSet-Private=574,0 MB

=== UMA instancia, NEW (formato 6, mmap) ===
PID 62028: WorkingSet=244,3 MB   WorkingSet-Private=129,9 MB

=== TRES instancias, OLD (formato 5, sem mmap) ===
PID 38560: WS=585,0  WSPriv=574,1
PID 56656: WS=584,9  WSPriv=574,0
PID 67972: WS=584,6  WSPriv=573,7
TOTAL WS: 1754,5 MB   TOTAL WS-Private: 1721,9 MB

=== TRES instancias, NEW (formato 6, mmap) ===
PID 36228: WS=244,5  WSPriv=130,1
PID 66632: WS=244,2  WSPriv=129,9
PID 69196: WS=244,0  WSPriv=129,6
TOTAL WS: 732,7 MB   TOTAL WS-Private: 389,6 MB
```

**Queda no agregado de tres instancias: 58,2% em Working Set total, 77,4% em
Working Set-Private.** Acima do criterio de parada de 30% — nao foi preciso
considerar a opcao (b).

**Working Set-Private e a metrica certa** (confirmado pelo revisor antes de
eu medir, e o motivo bate com o que a medicao mostrou): Working Set total
conta pagina mapeada residente INTEIRA em cada processo que a tem residente,
mesmo quando o cache de paginas do SO a compartilha entre processos —
por isso ele so caiu 58% enquanto o WS-Private caiu 77%.

**Ressalva que fica registrada, e nao escondida:** a queda percentual por
instancia e IGUAL com uma ou com tres instancias (77,4% nos dois casos), e
3 x (WS-Private de uma instancia) bate com o WS-Private agregado de tres —
nao ha queda ADICIONAL mensuravel por instancia extra alem da primeira. Isso
e esperado dado como o Windows contabiliza pagina mapeada de arquivo em modo
leitura: ela sai do "Private" por ser file-backed e reclamavel,
INDEPENDENTE de quantos processos a tem mapeada — os contadores de processo
do Windows nao expoem uma metrica de paginas de quadro fisico UNICAS entre N
processos (o equivalente do PSS do Linux). O que esta medido e provado e que
o array deixou de ser heap privado sempre residente, o que ja e um ganho real
e mensuravel; que as paginas residentes sao efetivamente as MESMAS entre os
processos (compartilhamento fisico estrito, e nao so "igualmente barata
cada copia") precisaria de RAMMap/VMMap para confirmar — **nao medido nesta
tarefa**.

### Custo em disco — reportado com o mesmo destaque do ganho

```
Formato 5 (antes):  108.492.193 B (103,5 MB)
Formato 6 (depois): 573.554.656 B (546,9 MB)
```

**5,3x maior, +443,5 MB** — mais do que os ~291 MB estimados no brief, porque
o cofre real (uso diario do usuario) cresceu desde que aquele numero foi
escrito: o array de posicoes hoje tem mais entradas do que as 18.229.295
registradas antes. Troco pre-decidido (disco por memoria compartilhada), so
que o disco pago e maior do que a estimativa original.

### Custo da reconstrucao pelo bump de formato

Medido no cofre real, tokenizacao completa das 4.490 notas: **157,3 s**
(binario `db68915`, grava formato 5) contra **157,9 s** (binario desta
tarefa, grava formato 6) — diferenca e ruido; escrever a secao fixa adicional
nao aparece no tempo de construcao, dominado pela tokenizacao. Ambos em
segundo plano, com as outras onze tools respondendo desde o primeiro segundo.
Registrado tambem em `docs/OPERACAO.md`, que e onde a decisao "toda troca de
formato reconstroi" ja estava documentada para a troca 1 -> 5.

### Rename atomico sobre arquivo mapeado — testado antes de confiar

`os.Rename` do salvamento atomico falha no Windows se o arquivo de destino
ainda esta mapeado neste processo. Confirmado experimentalmente, nao supondo:
`TestSaveOverwritesMappedCache` primeiro tenta `os.Remove` no arquivo com
`loaded` ainda mapeado e confere que FALHA (condicionado a
`runtime.GOOS=="windows"`), so depois exercita `SaveInvertedCache` por cima
do mesmo arquivo e confere que passa. `promoverArenaSePresente` copia `pos`
para o heap e desmapeia antes do rename — so dispara no caminho raro de cache
PARCIAL retomado e depois regravado; no caminho comum (cache completo)
`SaveInvertedCache` nunca roda de novo no mesmo processo.

### Verificacao

```
$ go test -race -count=1 ./internal/search/ ./internal/service/
ok  	github.com/jonyd/gobsidian/internal/search	12.339s
ok  	github.com/jonyd/gobsidian/internal/service	66.881s

$ go test -count=1 -run TestRankingGolden -v ./internal/service/...
--- PASS: TestRankingGolden (0.58s)
    (6 subtestes, todos PASS)

$ go vet ./... (windows, linux, darwin) — limpo nos tres
$ golangci-lint version
golangci-lint has version 2.12.2 ... — confere com o que o CI fixa

$ pwsh -File scripts/verify.ps1 -SkipNet
[OK] go build
[OK] go test -race
[OK] go test (tetos de latencia, sem -race)
[OK] go vet (windows)
[OK] go vet (linux)
[OK] go vet (darwin)
[OK] gofmt
[OK] golangci-lint
[OK] check_tool_params
[OK] Bateria completa. Pode commitar.
```

Nota lateral sobre `go vet`: `mmap_windows.go` tem uma conversao
`uintptr -> unsafe.Pointer` que e o idioma padrao de mapeamento de arquivo no
Windows em Go puro sem cgo (o proprio `cmd/go/internal/mmap/mmap_windows.go`
do toolchain usa o mesmo padrao) e que `go vet` marca por padrao como
"possible misuse". Reinterpretado via ponteiro-para-ponteiro (documentado no
codigo) em vez de suprimir a checagem globalmente — `go vet` roda limpo nos
tres alvos sem tocar em `verify.ps1` nem no config do `golangci-lint`.

### Prova de mutacao

```
pwsh -File scripts/mutate.ps1 -Path internal/search/mmap.go `
  -Anchor 'if dentroDoCofre(caminhoCache, vaultPath) {' -Replacement 'if false {' `
  -Test TestRecusaMapearCacheDentroDoCofre -Package ./internal/search/

[...] Mutando internal/search/mmap.go
      - if dentroDoCofre(caminhoCache, vaultPath) {
      + if false {
[...] go test -race -run TestRecusaMapearCacheDentroDoCofre ./internal/search/
--- FAIL: TestRecusaMapearCacheDentroDoCofre (0.02s)
    mmap_test.go:66: tentaAbrirArena mapeou um cache dentro do cofre; a guarda nao disparou
FAIL
[OK] internal/search/mmap.go restaurado byte a byte (SHA-256 confere).
[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
```
Exit code do `mutate.ps1`: `0`.

### Escopo

Cumprido integralmente, incluindo a medicao real no cofre de 4.490 notas que
tinha ficado bloqueada. Nao foi preciso a opcao (b): a opcao (a) sozinha
excedeu o criterio de parada de 30%. Arquivos deste worktree de teste
(`gob89-old-baseline`, cache dirs temporarios) removidos apos a medicao —
nao entraram em commit nenhum.

## Task 89 — compartilhamento entre processos, medido pelo revisor — 2026-08-10

O relatorio da tarefa registrou, honestamente, que a queda de WS-Private era
identica com 1 e com 3 instancias, e que **provar que as paginas sao fisicamente
as MESMAS entre processos precisaria de RAMMap/VMMap — nao medido**. A ressalva
estava certa: o Windows tira pagina file-backed do "Private" por ela ser
file-backed, compartilhada ou nao, e nao tem equivalente do PSS do Linux.

Fechado aqui por outra via: **memoria livre do sistema**, que nao depende de
contador por processo. Cofre real de 4.490 notas, cache formato 6 em diretorio
dedicado, todas as instancias com `--eager-search`:

```
0 instancias   11452 MB livres
1 instancia    consumiu 239 MB     soma dos WS  284 MB
2 instancias   consumiu 381 MB     soma dos WS  536 MB
3 instancias   consumiu 513 MB     soma dos WS  788 MB
```

A soma dos Working Sets chega a 788 MB e o sistema perdeu 513 MB. O custo
MARGINAL de cada instancia extra e **132-142 MB de memoria fisica**, contra
~252 MB de Working Set reportado por instancia. A diferenca sao as paginas
mapeadas, contadas em cada processo e residentes uma vez so.

Sem compartilhamento, tres instancias consumiriam ~717 MB (3 x 239). Consumiram
513 MB. **O compartilhamento e real e esta medido**, sem precisar de RAMMap.

Metodo, para quem repetir: `Win32_OperatingSystem.FreePhysicalMemory` antes de
subir qualquer instancia e depois de cada uma. Contador por processo nao serve
para esta pergunta — foi exatamente o que a ressalva da tarefa apontou.

## Task 90 — RNF-30 reformulado, antes de abrir o primeiro socket — 2026-08-10

Reabre uma decisao fechada, com autorizacao explicita do dono do projeto em
2026-08-05, para desbloquear o IPC local via socket AF_UNIX das Tasks 91 e 92.
Formulacao antiga: "nenhuma requisicao de rede... nenhum socket de saida em
nenhuma circunstancia." Nova: **"nenhum socket que saia da maquina."** Um
socket de dominio Unix nao atravessa rede nenhuma — o kernel resolve um
caminho de arquivo especial —, entao a garantia contra exfiltracao continua
de pe sob a formulacao nova; o que muda e o mecanismo de IPC permitido, nao a
superficie de exposicao. Texto completo, com data e autorizacao, em
`docs/PRD.md` §6.4.

A regra deixou de ser "nenhum import de `net`" e passou a ter duas camadas:

1. **Importacao.** `net/http` e qualquer `net/*` continuam banidos. O pacote
   `net` em si passou a ser permitido — sem ele nao ha como pedir um socket
   Unix.
2. **Chamada.** Dentro de `net`, so `net.Dial` e `net.Listen` sao aceitos, e
   so quando o primeiro argumento (a rede) e a constante literal `"unix"`.
   Rede vinda de variavel e recusada mesmo que o valor em tempo de execucao
   seja `"unix"` — o analisador nao pode provar isso estaticamente, entao nao
   aceita. Qualquer outra chamada do pacote `net` (`DialTCP`, `ListenTCP`,
   `DialUDP`, `LookupHost` etc.) e banida por padrao: o par Dial/Listen com
   `"unix"` e a unica porta aberta.

Implementado em `tools/netcheck/netcheck.go` via `go/analysis` + `go/types`
(nao so `go/ast` — precisa da informacao de tipo para distinguir constante de
variavel). `scripts/check_net.ps1` teve sua checagem textual de retrocompat
ajustada no mesmo sentido: so flagra `net/*`, nao mais `net` puro.

### Prova de disparo — as tres regras, plantadas e removidas

Plantado cada caso em `internal/config/zz_netcheck_scratch.go` (fora de
qualquer commit), rodado `scripts/check_net.ps1`, removido, rodado de novo.

**1. `net.Dial("tcp", ...)` — reprova:**
```
[!] netcheck reprovou em GOOS=windows
     internal\config\zz_netcheck_scratch.go:6:9: rede proibida: net.Dial so aceita a constante "unix"
```
**Removido — passa:**
```
[OK] Nenhum pacote de internal/ ou cmd/ importa net/* ou abre socket que saia da maquina (verificado via netcheck vettool em windows, linux, darwin)
```

**2. `net.Dial(rede, ...)` com `rede := "unix"` (variavel, nao literal) — reprova:**
```
[!] netcheck reprovou em GOOS=windows
     internal\config\zz_netcheck_scratch.go:7:9: rede proibida: net.Dial so aceita a constante "unix"
```
**Removido — passa:**
```
[OK] Nenhum pacote de internal/ ou cmd/ importa net/* ou abre socket que saia da maquina (verificado via netcheck vettool em windows, linux, darwin)
```

**3. `http.Get(...)` — reprova:**
```
[!] netcheck reprovou em GOOS=windows
     internal\config\zz_netcheck_scratch.go:3:8: pacote de rede proibido: net/http
```
**Removido — passa:**
```
[OK] Nenhum pacote de internal/ ou cmd/ importa net/* ou abre socket que saia da maquina (verificado via netcheck vettool em windows, linux, darwin)
```

`git status --porcelain internal/config` vazio depois do ultimo removido —
nao sobrou scratch em nenhum commit.

### Prova de mutacao

A primeira tentativa, com o corpo do `if` referenciando so `call.Pos()`,
saiu **inconclusiva**: `if false {` deixava `ehConstante`, `valorDe` e `arg0`
sem uso nenhum no resto da funcao, e o `go test` reprovava por "declared and
not used" — build quebrado, nao cobertura. Corrigido referenciando os tres de
novo dentro do proprio corpo do `if` (na mensagem do `Reportf`, com
`arg0.Pos()`), que fica parado mas presente apos a mutacao trocar so a linha
da condicao.

```
pwsh -File scripts/mutate.ps1 -Path tools/netcheck/netcheck.go `
  -Anchor 'if !ehConstante(arg0) || valorDe(arg0) != "unix" {' -Replacement 'if false {' `
  -Test TestNetcheckRecusaRedeVariavel -Package ./tools/netcheck/

[...] Mutando tools/netcheck/netcheck.go
      - if !ehConstante(arg0) || valorDe(arg0) != "unix" {
      + if false {
[...] go test -race -run TestNetcheckRecusaRedeVariavel ./tools/netcheck/
--- FAIL: TestNetcheckRecusaRedeVariavel (3.71s)
    analysistest.go:713: redevar/redevar.go:7: no diagnostic was reported matching `rede proibida: net.Dial so aceita a constante "unix"`
FAIL
[OK] tools/netcheck/netcheck.go restaurado byte a byte (SHA-256 confere).
[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
```
Exit code do `mutate.ps1`: `0`.

### Gate completo

`pwsh -File scripts/verify.ps1 -SkipCross` verde nas 8 etapas, incluindo
`golangci-lint` (confirmado `v2.12.2`, igual ao fixado no CI) e `check_net
(RNF-30)`.

### Escopo

Cumprido integralmente. Nenhum socket foi aberto nesta tarefa — so a
garantia e o analisador que a Task 91 vai depender de manter verde. Commit
`a46175cbdd3648d87dcfd731f22421ea1c0b3939` (`git cat-file -t` confirma
`commit`).
