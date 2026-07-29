# gobsidian v0.1 — progresso SDD

Plano: docs/superpowers/plans/2026-07-25-gobsidian-v01.md
Baseline: e2c8ddf (docs only)

Task 1: complete (commits e2c8ddf..31f276d, review clean)
Task 2: complete (commits 31f276d..79bd036, review clean apos 2 fix passes)
Task 3: complete (commits 79bd036..d139e5d, review clean apos 1 fix pass)
Task 4: complete (commits d139e5d..d57d03c, review Approved; 1 Important doc-only fechado depois)
Task 5: complete (commits 8a3358c..150fd38, review clean apos 2 fix passes)
Task 6: complete (commits f0b6451..22cbe1b, review Approved; 2 Important plan-mandated fechados)
Task 7: complete (commits 22cbe1b..b85d695, review Approved apos 3 fix passes)
Task 8: complete (commits b85d695..5000de1, review Approved apos 2 fix passes)
Task 9: complete (commits 46ae951..1d93417, 2 revisoes completas + 2 fix passes)
  ATENCAO: o ultimo fix pass NAO teve revisao fresca. Fechado por evidencia direta
  (20 sessoes limpas exit 0, mutacao %w->%s reprovando o teste novo). A revisao
  final de branch DEVE cobrir cmd/gobsidian/serve.go e internal/mcpsrv/convert.go.
Task 10: complete (commits 1d93417..2281284, 1 revisao completa + 1 fix pass)
  ATENCAO: fix pass sem revisao fresca. Fechado por evidencia (1 varredura medida,
  mutacao de marcador). Revisao final DEVE cobrir internal/doctor/.
Task 11: complete (commit 67d7d69, tag m0-lifecycle) — 100 ciclos, zero orfaos

Task 12: complete (commits b6d4a7a..6eb612c, 1 revisao completa + 1 fix pass + re-revisao Approved)

Task 13: complete (commits 6eb612c..c80fc01, 1 revisao completa + 2 fix passes + re-revisao Approved)

Task 14: complete (commits c80fc01..8bcb706, 1 revisao completa + 1 fix pass)
  ATENCAO: o fix pass NAO teve revisao fresca. Fechado por evidencia direta (3 mutantes
  aplicados e capturados, parse de [[[a]] b](d.md) e de ![alt](x.png) reportados).
  A revisao final de branch DEVE cobrir internal/parser/ext_wikilink.go e ast.go.

Task 15/16/17: complete (commits 8bcb706..f1147ab, 1 revisao COMBINADA das tres +
  2 fix passes + re-revisao Approved com worktree isolado e 3 mutacoes independentes)

Task 27: complete (commit 69362c7, fsnotify facade with unified vault relevance filtering, review Approved)
Task 28: complete (commit cc08160, single-ticker debouncer with dirty set coalescence, review Approved)
Task 29: complete (commit 0bce29f, real change verification with mtime+size diffing and index application, review Approved)
Task 30: complete (commit 3496610, reconciliation recovery on fsnotify overflow event, review Approved)
Task 31: complete (commit 2a59d0c, xxhash rename correlation for fast path updates and backlink preservation, review Approved)
Task 32: complete (commits 56f8135..b5b2a9f, expose watcher metrics in vault_stats tool, review Approved)
Task 33: complete (commits e507be0..8daa35a, rename correlation single pass, zero asset/cloud reads, review Approved)
Task 34: complete (commits 0430807..f35edd4, reconciliation tests that actually lose events, review Approved)
Task 35: complete (commits da09dc7..f67257a, normalize byAlias keys so Remove clears what Build wrote, review Approved)
Task 36: complete (commits 98dd2d2..faa0674, MoveNote refreshes stat and matches Remove+Replace, review Approved)
Task 37: complete (commits a59e814..98eec3e, per-reason drop counters, coalesced count, and a real active flag, review Approved)

=== M0, M1 e M2 (Tasks 1-32) COMPLETAS! ===

Task 18: complete (commit 6c5a241) — revisao feita pelo modelo principal, Aprovada.
  48 pares sem orfao, todos codeblocks vazios com .md contendo a sintaxe de verdade,
  harness provado por adulteracao de golden, bytes de CRLF/BOM sobreviveram.
  1 Important roteado para a Task 19: a costura vault.StripBOM -> Parse nao e testada
  por ninguem, e o golden edge/bom.md fixa um estado de FALHA como contrato.

Task 19-26: implementadas (commits fbc192f..66ea24a, 11 commits) SEM NENHUMA REVISAO.
  Revisao feita pelo modelo principal em 2026-07-28, depois do fato. Achados:

  CRITICAL corrigidos (commit 1619a37):
  - offset de BOM nunca somado de volta: toda leitura de secao em nota com BOM saia
    deslocada 3 bytes. Provado: com BOM devolvia "o

## Alvo

CONTEUDO-ESPERAD".
    TestBuildBOM existia e passava — afirmava presenca do heading, nunca o offset.
  - teste de paridade passava sem comparar nada: guard checava os.Stat do diretorio,
    que existia vazio, entao nao pulava; o laco iterava mapa vazio. Reportava a metrica
    do PRD 7 como atingida. Agora checa CONTEUDO e pula.

  IMPORTANT corrigidos em doc (commit 77b88d4):
  - OPERACAO.md 5 trazia medicoes fabricadas ("ex: 408ms", "tende a ficar ~30-45 MB").
  - README declarava "v0.1 publicada" sem tag, sem gate de orfaos, sem medicao.

  IMPORTANT abertos, registrados no plano (commit 191f980):
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

=== M1: os 6 Important da revisao FECHADOS. Paridade VERIFICADA. Falta: gates e tag ===

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

BRIEFS AUTOCONTIDOS (commit dc6bbea): as Tasks 19-26 do plano ganharam bloco proprio de
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
Achados: 3 Important, 7 Minor. Todos os Important fechados. Commits e74e375..44672fd.

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
bootstrap 1de565c: ja falhava la. O job `lint` do CI esta VERMELHO DESDE O BOOTSTRAP,
e o job `lint-windows` novo ficara vermelho pelas mesmas 39.
Uma delas e excecao DOCUMENTADA no CLAUDE.md e precisa de nolint/exclusao, nao de
mudanca de codigo:
  cmd/gobsidian/serve.go:133 — "Shutdown should pass the context parameter (contextcheck)"
  lifecycle.Shutdown bloqueia e nao recebe ctx de proposito.

LACUNA DE M6 CONFIRMADA E INALTERADA: stdin-eof venceu 100/100 nas duas rodadas do gate,
entao a vigilia do pai e os sinais seguem sem verificacao ponta a ponta.

=== DIVIDA DO LINT: PAGA (2026-07-27, commit b6d4a7a) ===
golangci-lint: 0 issues em GOOS=linux, darwin e windows. Eram 39, identicas nos
tres alvos, vermelhas desde o commit de bootstrap 1de565c.

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

Commits de ressincronizacao ate agora: 44672fd, a1ec366, 2b4967f.
Corrigir o plano ANTES de despachar o fixer e necessario mas NAO SUFICIENTE:
o fix loop muda o codigo depois disso, e o plano fica para tras de novo.
A conferencia tem que acontecer TAMBEM no fim de cada task, e MECANICAMENTE —
extrair os blocos ```go do plano e comparar com os arquivos do disco, nao ler.
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
   CORRIGIDO em 50529ef: action v8 com `version: v2.12.2` FIXA. Fixar e o ponto:
   com a versao flutuando, os dois lados resolvem binarios diferentes e o
   verde local para de dizer qualquer coisa. Registrado tambem no CLAUDE.md.
   Depois do fix: lint e lint-windows verdes.

Commits de ressincronizacao plano<->codigo ate agora: 44672fd, a1ec366, 2b4967f,
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

Corrigido em 9a03781. Plano ganhou CORRECAO OBRIGATORIA na Task 24.
`parser.percentDecode` foi EXPORTADA como `parser.PercentDecode` para o mcpsrv
reusar em vez de manter segunda copia das regras de escape invalido.
Prova de mutacao: tirar a terceira barra faz o teste novo estourar por panic.

### Documentacao que prometia o que o codigo nao emite

`docs/OPERACAO.md` §3 listava campos de `vault_stats` que NAO EXISTEM:
TotalNotes, TotalBytes, TotalLinks, TotalAliases, TotalTags, LoadTimeMs,
CloudOnlyFiles. Zero de sete existem. §5 mandava rodar
`gobsidian index --vault --stats`, subcomando inexistente.
Mesma classe do `Collisions: 0`, pelo lado da documentacao.
Corrigido em 55966ce, conferido campo a campo contra `service.StatsResult`.

### Medicao (escala pequena, POR DECISAO do usuario)

`scripts/measure.ps1` criado. Le `index_ms` do log de boot — campo novo — e
amostra WorkingSet64; reporta o MAIOR RSS, nao o ultimo.

Cofre de 7 notas, 3 execucoes, maquina de referencia/12 nucleos:
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
