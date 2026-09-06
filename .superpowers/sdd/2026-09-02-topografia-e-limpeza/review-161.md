# Revisao da Task 161 — auditoria dos treze testes reportados pelo sweep

Alvo: commits `b3bcba0` + `ebc0f12` sobre a base `15c66fb`.
Modo: somente leitura. Nenhuma edicao, nenhum gate, nenhuma mutacao aplicada por
mim — as mutacoes foram conferidas por leitura do produto citado, e os testes
alvo foram rodados isolados, por pacote.

## Progresso

- 23:58 — inicio da revisao da Task 161; arquivo de progresso criado antes de ler qualquer entrada
- 23:58 — brief `task-161-brief.md` lido (treze sitios, protocolo de mutacao, contrato de relatorio)
- 23:59 — relatorio `task-161-report.md` lido: catorze linhas, saidas coladas, tres defeitos fora de escopo
- 23:59 — diff `review-15c66fb..ebc0f12.diff` lido na integra (8 arquivos, todos `_test.go`)
- 23:59 — `git log`/`git show --stat` dos dois commits; mensagens em Conventional Commits, ingles, sem caractere solto
- 23:59 — `git diff --name-only 15c66fb ebc0f12 | grep -v '_test.go$' | grep -v '.md$'` → vazio
- 00:00 — produto conferido: `bm25.go:167,193`, `graph.go:462,481,667`, `vault/path.go:85,111-123`, `inverted.go` Remove/DocCount
- 00:00 — produto conferido: `write.go:22,185,288`, `persist_codec.go:195`, `index/update.go:126`, `config.go:74`
- 00:00 — cobertura substituta do teste apagado lida em `internal/config/config_test.go:198-212`
- 00:00 — `go test` isolado: `TestServeNaoEscreveNoStdout` PASS; `TestFilter_OutsideVaultIsDropped` ok; `TestBM25TermoEmTodasAsNotasAindaPontua|TestInvertedConcurrencyRace|TestPoolReuse` ok
- 00:01 — `docs/papeis/revisor.md` e `docs/papeis/testador.md` lidos; rubrica aplicada linha a linha
- 00:01 — `gofmt -l` nos oito arquivos tocados: limpo. Imports novos (`math`, `fmt`) presentes; `strings` e `path/filepath` ja existiam
- 00:02 — escrita do parecer

---

## O que foi conferido, por linha da tabela

A coluna "regra de produto" e a mutacao colada foram cruzadas com o codigo real.
Nenhuma mutacao da tabela e cosmetica: cada uma apaga a regra que a linha nomeia.

| # | Regra nomeada | Sitio de produto conferido | Mutacao remove a regra? | Assercao nova pega? |
|---|---|---|---|---|
| 1 | leitura devolve conteudo, nao envelope | `bm25.go:193` `if !math.IsNaN(score) && score > 0` | sim — `if false` zera `results` e `vault_search` volta `"results":[]` | sim, `wantIn` + `Fatalf` para caso de sucesso sem `wantIn` |
| 2 | `note_list` clampa `limit` antes do indice | `graph.go:462` `q.Limit = ComTeto(q.Limit)` | sim — `q.Limit = req.Query.Limit` entrega 100000 ao `index.List` | sim, 501 notas e `len(res.Notes) != 500`, com `res.Total == 501` de controle |
| 3 | `vault_stats` conta pelo ramo `s.index == nil` | `graph.go:663,667` `out.Notes++` dentro do ramo | sim | mantido — ja reprovava |
| 4 | o `1 +` do idf | `bm25.go:167` `math.Log(1.0 + (N-d+0.5)/(d+0.5))` | sim — com d==N==2 o idf vira `log(0.2) < 0`, cai no `idf <= 0` de `:185`, `res` fica vazio | sim, `len(res) != 2` e `t.Fatalf` |
| 5 | cache derivado fora do cofre | `config.go` `defaultCacheDir` | sim | apagado; substituto real conferido em `config_test.go:198-212`, que chama `config.Load` |
| 6 | `Normalize` tira acento e baixa caixa | `analyzer.go` | sim | mantido — ja reprovava |
| 7 | `AppendNote` recusa hash divergente | `write.go:185` | sim | mantido |
| 8 | `PatchNote` recusa hash divergente | `write.go:288` (bloco 287-290) | sim | mantido |
| 9 | hash do indice == hash que o escritor aceita | `update.go:126` vs `write.go:22-24`, publicado em `graph.go:481` | sim, mutando so o lado do servico | mantido; a hipotese de tautologia do brief e refutada com razao |
| 10 | `Slug` sobrevive ao codec | `persist_codec.go:195` `e.str(h.Slug)` | sim | mantido |
| 11 | — | — | — | `TestSubcommands_FlagsSetPopulated` ausente do worktree; `cli_subcommands_test.go:163-172` e o teste do `inspect --json`, como o relatorio diz |
| 12 | `serve`/`daemon` nao escrevem no writer do cobra | `config.go:74` `"...use --vault"` | sim, nos tres eixos testados | sim, `strings.Contains(err, "--vault")` distingue o erro de configuracao de `unknown command` |
| 13 | confinamento por raiz | `vault/path.go:111` `filepath.Rel(root, abs)` + `:119` | sim — alargar a raiz um nivel faz o `"../"` sumir | sim, com dois `t.TempDir()` irmaos a `Canonicalize` passa a aceitar e o teste reprova |
| 14 | `Remove` tira a nota da contagem | `inverted.go` `Remove` → `sombrearLocked`+`removeLocked`; `DocCount` = `len(ix.docLengths)` sem base | sim | sim, `10` (Remove inerte) e `0` (Add inerte) caem fora de `{6,7}` |

**A conta de `{6,7}` da linha 14 foi refeita e fecha.** O escritor so testa
`ctx.Done()` no topo do laco, entao a ultima volta e sempre completa; as dez
ultimas voltas cobrem os dez residuos `i%10` exatamente uma vez, e dez inteiros
consecutivos contem tres ou quatro multiplos de tres. Nao ha janela de
flakiness: os leitores nao escrevem, e o corte por timeout nao parte uma
iteracao ao meio.

**Restricoes globais, todas satisfeitas.** `git diff --name-only 15c66fb ebc0f12`
filtrado por nao-`_test.go` e nao-`.md` volta vazio. Nenhuma aresta de import
nova de producao (o diff nao toca producao); os imports acrescentados sao
`math` em `bm25_test.go` e `fmt` em `limites_enums_test.go`, ambos usados.
`strings` (console_saida) e `path/filepath` (filter) ja estavam la. `gofmt -l`
limpo nos oito. Mensagens dos dois commits em Conventional Commits, ingles, sem
caractere solto. `b3bcba0` traz nove arquivos — oito `_test.go` e o proprio
relatorio; `ebc0f12` traz so o relatorio. Nenhum vestigio de `git add -A`: o
`task-163-report.md` que aparece no intervalo vem de `9629584`/`50fb0bc`, dois
commits de docs anteriores a esta tarefa, e nao dos commits auditados.

**Nada do que a casa proibe entrou.** Nenhum `t.Skip` novo, nenhum `time.Sleep`
como espera, nenhuma copia local de helper que `internal/vaulttest` ja fornece,
nenhum numero afirmado sem medicao — inclusive o `32 goroutines x 1000 voltas`
citado no comentario de `pool_test.go`, que confere com
`analyzer_test.go:138-139` (`goroutines = 32`, `voltas = 1000`). A unica
assercao guardada por condicao no diff e a guarda **contraria**
(`len(tt.wantIn) == 0 → t.Fatalf`), que existe justamente para impedir vacuidade
futura.

---

## Achados

### NON-BLOCKING

`internal/search/pool_test.go:11` — **o nome `TestPoolReuse` continua prometendo
a cobertura que o comentario novo desmente.** O comentario agora diz, corretamente,
que o teste afirma so o resultado de `Normalize` e nao mede alocacao nenhuma; o
nome, que e o que aparece na saida do `go test` e em qualquer varredura por
nome, continua dizendo "reuse". A premissa do `testador.md` e cobertura
reportada que nao existe, e um nome tambem reporta. Fix: renomear para algo como
`TestNormalizeAcentosECaixa` e deixar o ponteiro para
`TestNormalizeNaoVazaEstadoEntreUsos` no comentario.

`internal/mcpsrv/tools_read_test.go:80-147` — **so um dos seis casos de sucesso
teve a mutacao colada.** A prova de `bm25.go → if false` exercita `vault_search`;
`note_read`, `note_list`, `note_metadata`, `link_graph` e `tag_list` ganharam
`wantIn` sem uma mutacao que mostre cada um reprovando. As assercoes sao
plausiveis e o fixture as sustenta (`A.md` tem `tags: [a]`, `Text A`, `[[B]]`),
e a guarda `len(tt.wantIn) == 0` fecha a vacuidade estrutural — mas o brief pede
prova por sitio, e o relatorio nao diz que cinco dos seis ficaram sem ela. Fix:
uma mutacao por tool restante, ou uma frase no relatorio declarando o alcance
real da prova.

`internal/config/config_test.go:203-208` — **a cobertura substituta citada para
o teste apagado tem um `return` silencioso.** Quando `filepath.Rel` erra
(volumes diferentes), o subteste `derived_path_is_not_inside_the_vault` retorna
sem afirmar nada — a forma exata que `docs/papeis/testador.md` proibe em
`internal/vaulttest`. Nesta maquina ele nao e vacuo (a saida FAIL colada na
linha 5 do relatorio prova que ele dispara), e o arquivo esta fora da lista dos
treze, entao isto nao e regressao desta tarefa. Fix, quando alguem passar por
la: trocar o `return` por uma assercao explicita de que o erro de `Rel` foi por
volume diferente.

`internal/search/bm25_test.go:251-253` — **a assercao de ordenacao vive num
confundidor conhecido.** `a.md` ("de de de", len 3) contra `b.md` ("de de",
len 2): o `tf` maior de `a.md` e a normalizacao por comprimento do BM25 puxam
para lados opostos, e com `ParamK1 = 1.2` / `ParamB = 0.75` a conta sai 1,507
contra 1,457 — passa, por 3%. E o padrao do `testador.md` §"Mais tres". Nao e
grave porque essa linha nao e a que mata a mutacao do idf (quem mata e o
`len(res) != 2` com `t.Fatalf`), e o teste roda verde aqui. Fix opcional:
equilibrar os comprimentos, ou afirmar so o que a regra exige.

`internal/service/limites_enums_test.go:452-456` — **501 arquivos escritos por
execucao** num pacote que ja leva 64 s sob `-race`. O custo e o preco de tornar
o clamp observavel, e nao ha caminho mais barato; fica registrado para quem
investigar tempo de suite depois.

### BLOCKING

Nenhum.

---

## Vereditos

**Spec: APPROVED**

O brief pedia um veredito por sitio, com mutacao aplicada e resultado, mais o
diff que os vereditos exigem — e e isso que esta entregue. Os treze sitios estao
cobertos: `write_test.go` virou tres linhas por ter tres sitios, o que da as
catorze linhas com a explicacao no proprio relatorio, e `cli_subcommands_test.go:168`
esta com o veredito correto de "ja removido em `b8ed7f6`", que eu confirmei
(`TestSubcommands_FlagsSetPopulated` nao existe no worktree, e a linha 168 hoje
pertence ao teste do `inspect --json`). Cada "consertado" traz a saida do teste
reprovando sob mutacao, no passado e colada, e cada "mantido" traz a saida do
FAIL que justifica manter; o unico "apagado" nomeia o teste substituto, que eu
li e que de fato chama `config.Load`. As tres regras de execucao foram
respeitadas: nenhuma linha de producao no commit, staging por caminho explicito,
e os tres defeitos de produto encontrados no caminho ficaram registrados como
fora de escopo em vez de consertados por conta propria — inclusive o mais
interessante, a regra de travessia escrita duas vezes em `vault/path.go`, que e
o que explica por que a primeira mutacao da linha 13 sobreviveu. O relatorio
tambem contesta a premissa do brief onde ela estava errada (a suposta tautologia
do `write_test.go`), com o mecanismo verificado ate os dois sitios independentes
— que e exatamente o que `docs/papeis/revisor.md` pede de quem revisa, e aqui
partiu de quem implementou.

**Quality: APPROVED**

Conferi o produto por tras de cada mutacao em vez de acreditar na tabela, e as
seis que eu podia derrubar por leitura se sustentam: o `1 +` do idf com d==N
manda o termo para o `idf <= 0` e esvazia o resultado; `q.Limit = req.Query.Limit`
entrega 100000 ao indice sobre 501 notas; alargar a raiz de `filepath.Rel` um
nivel faz o `"../"` desaparecer e o confinamento nunca rodar; `Remove` inerte
deixa `DocCount` em 10, que cai fora de `{6,7}` — e a conta de `{6,7}` fecha
por argumento, nao por chute, porque a ultima volta do escritor e sempre
completa. Os consertos melhoram a suite na direcao certa: assercao sobre
conteudo em vez de envelope nao-nulo, fixture grande o bastante para o teto ser
observavel, raizes reais em vez de `C:\test\vault` com drive diferente, valor
exato em vez de `< 0` sobre `len(map)`, e uma guarda nova que faz um caso de
sucesso sem expectativa reprovar na hora. Os comentarios acrescentados dizem o
que foi medido e o que o teste **nao** cobre, com o numero conferido (o
`32 x 1000` bate com `analyzer_test.go`). Os quatro achados nao-bloqueantes sao
polimento — um nome de teste desalinhado do proprio comentario, cinco assercoes
novas sem mutacao individual, um `return` silencioso num arquivo de terceiros e
uma assercao de ordenacao com margem estreita —, nenhum deles capaz de tornar
verde um teste que deveria reprovar.
