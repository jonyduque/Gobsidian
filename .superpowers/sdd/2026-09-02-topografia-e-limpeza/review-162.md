# Revisao Task 162

## Progresso

- 00:41 abri o arquivo de revisao antes de ler qualquer entrada
- 00:41 li o brief da Task 162
- 00:41 li o relatorio (EM ANDAMENTO, sem SHA nem evidencia) e o stat do commit d12f84f
- 00:42 li o diff inteiro (958 linhas): gerador, contexto, persist, bm25, classify e os seis .tsv
- 00:43 reli o relatorio: ele foi reescrito as 00:41 (15493 bytes); a primeira leitura pegou a versao curta
- 00:45 conferi ancoras de mutacao e TODAS as linhas citadas nas saidas coladas: batem com as assercoes reais
- 00:46 reproduzi o corpus ANTIGO via go test -overlay (sem tocar a arvore) e apliquei a mutacao do peso de heading: 2 dos 6 .tsv MUDARAM
- 00:47 conferi o ledger (linha 5209 registra 162) e escrevi os achados
- 00:50 review-162.md fechado: 1 BLOCKING, 3 NON-BLOCKING, 2 conferidos-sem-acao

## O que foi conferido, e como

Julgado sobre `review-a313f32..d12f84f.diff` (958 linhas), nao sobre a arvore de
trabalho — o mesmo implementador esta editando `internal/watcher/*_test.go` e
`internal/service/delete_test.go` para a Task 164.

Rodados por mim, pacote a pacote:

```
$ go test ./internal/service/ -run 'TestRanking' -v
--- PASS: TestRankingGolden (0.59s)
    --- PASS: TestRankingGolden/termo-amplo (0.00s)
    --- PASS: TestRankingGolden/dois-termos (0.00s)
    --- PASS: TestRankingGolden/frase-exata (0.00s)
    --- PASS: TestRankingGolden/com-acento (0.00s)
    --- PASS: TestRankingGolden/so-no-titulo (0.00s)
    --- PASS: TestRankingGolden/so-em-heading (0.00s)
ok  	github.com/jonyd/gobsidian/internal/service	1.236s

$ go test ./internal/search/ -run 'TestBM25' -v -count=1
--- PASS: TestBM25PesoDeCorpoEOMenorDosTres (0.02s)
(mais 9 PASS no arquivo)
ok  	github.com/jonyd/gobsidian/internal/search	1.666s

$ go test ./internal/search/ ./internal/index/
ok  	github.com/jonyd/gobsidian/internal/search
ok  	github.com/jonyd/gobsidian/internal/index
```

**As saidas coladas no relatorio sao reais.** Conferi cada linha citada contra o
arquivo de hoje, e todas apontam para a assercao que a mensagem diz:

| Citado no relatorio | O que ha naquela linha |
|---|---|
| `bm25_test.go:169` | `t.Errorf("titulo=%.6f nao ficou acima de heading=%.6f: ...` |
| `bm25_test.go:173` | `t.Errorf("heading=%.6f nao ficou acima de corpo=%.6f: ...` |
| `backlink_contexto_test.go:94` | `t.Errorf("kind=%v: Context %q nao traz %q, ...` |
| `backlink_heading_test.go:106` | `t.Errorf("contexto %q nao traz %q, ...` |
| `persist_test.go:196` | `t.Errorf("Backlinks(%s) divergiu: ...` |
| `ranking_golden_test.go:300` | `t.Errorf("ranking mudou. --- quer --- ...` |

E as ancoras das tres mutacoes existem: `internal/search/bm25.go:223`
(`return WeightTitle`), `:226` (`return WeightHeadings`),
`internal/index/contexto_link.go:53` (`Context: contextoDoLink(body, l),`).
`pesoDeCampo` (bm25.go:218-229) devolve mesmo `WeightBody` na primeira linha
quando a Note e nula — a premissa do Step 4 confere.

Nenhuma dep nova: `git diff a313f32..d12f84f -- go.mod go.sum` e vazio. Nenhum
`t.Skip` nos seis arquivos tocados. `git diff --stat` so tem `_test.go` e
`testdata/`. Working tree identica ao commit em `internal/`.

---

## Achados

### B1 — BLOCKING

`internal/service/ranking_golden_test.go:88` e `:218-219` — e tambem a mensagem
do commit `d12f84f` ("Erasing the heading weight changed no byte of any .tsv")
e `task-162-report.md` ("Antes do conserto do corpus esta mesma mutacao deixava
os seis `.tsv` intactos") — **afirmam um estado que nao foi verificado, e que e
falso quando se verifica.**

O texto de `:88`:

> Os goldens congelavam o desempate, não o ranking: apagar o
> peso de heading não mudava um byte de nenhum `.tsv`.

E o de `:218-219`:

> apagar o peso de heading não mudava um byte de arquivo nenhum, e os seis
> subtestes continuavam verdes.

Reproduzi o corpus ANTIGO sem tocar na arvore, com `go test -overlay`: o
`ranking_golden_test.go` de `a313f32` recuperado por `git show`, com a linha do
golden redirecionada para um diretorio de scratch e a escrita forcada, e uma
copia de `bm25.go` com `return WeightHeadings` trocado por `return WeightBody`.
Duas execucoes, so o `bm25.go` mudando entre elas:

```
$ GOLDEN_OUT=$S/goldA go test -overlay=base.json ./internal/service/ -run TestRankingGolden
ok  	github.com/jonyd/gobsidian/internal/service	1.410s
$ GOLDEN_OUT=$S/goldB go test -overlay=mut.json  ./internal/service/ -run TestRankingGolden
ok  	github.com/jonyd/gobsidian/internal/service	1.284s

$ diff -r $S/goldA $S/goldB
diff -r goldA/dois-termos.tsv goldB/dois-termos.tsv
1,20c1,20
< pasta00/n0150.md	0.006210
< pasta00/n0000.md	0.006116
  (... as 20 linhas)
---
> pasta00/n0150.md	0.005949
> pasta00/n0000.md	0.005841
  (... as 20 linhas, mesma sequencia, so a coluna de score)
diff -r goldA/so-em-heading.tsv goldB/so-em-heading.tsv
1,20c1,20
< pasta00/n0150.md	0.002685
< pasta00/n0000.md	0.002613
  (... as 20 linhas)
---
> pasta00/n0150.md	0.002121
> pasta00/n0000.md	0.002033
  (... as 20 linhas, mesma sequencia, so a coluna de score)
```

**Dois dos seis `.tsv` mudavam, 40 linhas ao todo, e o `TestRankingGolden`
antigo REPROVAVA com essa mutacao.** `dois-termos` mudava porque o heading
antigo era `## Execução fiscal` e "execucao" e um dos dois termos daquela
consulta — ele estava no heading, nao so no corpo.

O que era verdade — e e o achado da tarefa, intacto — e que a **ordem** nao
mudava: nas duas execucoes as 20 linhas saem na mesmissima sequencia, porque com
299 notas identicas o peso e fator comum e o que ordena e o desempate por
caminho mais o fato de `n0150` ser tres tokens mais curta. O golden congelava o
desempate. Ele so nao era inerte ao peso de heading.

Isto e BLOCKING porque a frase e uma afirmacao medivel, escrita como medida, em
**duas docstrings permanentes** e na mensagem de commit — os tres lugares que
sobrevivem a tarefa. Um leitor futuro que confiar nela vai acreditar que o
golden antigo nao via peso de campo nenhum, e vai desenhar a proxima prova em
cima disso. E o exato padrao do `CLAUDE.md`: "Não afirme estado que você não
verificou."

**Fix:** trocar a clausula final nos tres textos por algo como — "apagar o peso
de heading nao movia UMA LINHA de lugar em `.tsv` nenhum: mudava a coluna de
score de `so-em-heading` e de `dois-termos` (o heading antigo era `## Execução
fiscal`, e `execucao` e termo daquela consulta), e deixava a sequencia inteira
identica. O golden congelava o desempate, nao o ranking." A mensagem de commit
nao pode ser reescrita sem reescrever o commit; um paragrafo no relatorio
corrigindo-a resolve, e as duas docstrings vao no proximo commit.

### N1 — NON-BLOCKING

`internal/service/ranking_golden_test.go:139-142` — o comentario diz, sobre as
notas sem `i%7 == 0`:

> As demais notas trazem só a forma reduzida, no corpo.

Nao trazem. A forma reduzida "prescricao" so entra no corpo quando `i%3 == 0`.
Uma nota com `i%7 != 0` **e** `i%3 != 0` — pouco mais da metade do corpus — nao
tem nem a forma crua nem a reduzida.

**Fix:** "As notas com `i%3 == 0` trazem a forma reduzida no corpo; e o
contraste que `com-acento` mede."

### N2 — NON-BLOCKING

`internal/service/ranking_golden_test.go:106-108` e
`internal/service/testdata/ranking/so-em-heading.tsv` — o comentario diz que
`so-em-heading` "é o único golden que depende do peso de heading". Verdadeiro
para os **valores**, e a mutacao colada no relatorio prova isso. Mas como
"fiscal" aparece **exatamente uma vez, sempre num heading, nas 28 notas que
casam**, `WeightHeadings` e fator comum a todas: a **ordem** daquele golden e
decidida so pela normalizacao por comprimento. A mutacao do relatorio confirma —
o "tem" colado traz `n0000, n0044, n0121` na mesma sequencia do "quer", so com
outros scores.

Nao e defeito: o golden compara a linha inteira, entao ele pega. Vale registrar
para que ninguem depois leia a ORDEM de `so-em-heading` como prova de peso de
heading, e para nao ficar surpreso quando a mutacao mudar so a segunda coluna.

**Fix:** uma frase na docstring — "o que a ordem deste golden mede e o
comprimento; o peso de heading aparece na coluna de score, porque ele e fator
comum as 28 notas que casam."

### N3 — NON-BLOCKING

`internal/service/ranking_golden_test.go:148-165` (o corpo do gerador) —
`n0000` e simultaneamente (a) a nota **unicamente mais curta** do corpus
(`20 + (0*7)%60 + 0/60 = 20`, e nenhum outro `i` da 20) e (b) membro de **todas**
as classes de congruencia: `0 % 3 == 0 % 4 == 0 % 5 == 0 % 7 == 0 % 11 == 0`.
Ela tem heading `Rito fiscal`, titulo `Prescrição 0000`, `prescricao`,
`execucao`, e o menor comprimento possivel.

Resultado: `n0000` e o primeiro de `com-acento`, `dois-termos` e
`so-em-heading` — tres dos seis goldens. A forma "a nota mais curta ganha", que
e o que esta tarefa saiu para eliminar, sobrevive no posto 1 de metade dos
goldens. A diferenca real e que agora ela ganha por acumular tudo, e os postos
2..20 discriminam de verdade — as assercoes do `conferirDiscriminacao` passam
com folga.

**Fix (opcional, barato):** deslocar a fase do enchimento, por exemplo
`20 + ((i*7+29)%60) + i/60`, para que `i == 0` deixe de ser o minimo. Regenerar
os seis `.tsv`. Custo: um commit de testdata.

### N4 — conferido, sem acao

O gerador **nao** segue a formula literal do brief: o enchimento e
`20 + (i*7)%60 + i/60`, e nao `20 + (i*7)%60`. O desvio esta declarado no
relatorio e na docstring `:94-101`, com o motivo (periodo 60 contra 300 notas, e
3, 4 e 5 dividem 60, entao as cinco notas de cada classe concordavam tambem em
tf) e com o FAIL que o revelou colado. **Confirmei o motivo de forma
independente**, no proprio golden: em `termo-amplo.tsv` as cinco notas
`i ≡ 43 (mod 60)` saem com cinco scores distintos —

```
pasta03/n0043.md	0.003212
pasta03/n0103.md	0.003206
pasta03/n0163.md	0.003200
pasta03/n0223.md	0.003194
pasta03/n0283.md	0.003189
```

— exatamente o quinteto que empataria sem o `+ i/60`. O "Medido:" da linha :101
e verdadeiro.

As demais formulas do brief estao ao pe da letra: `1 + i%4` para "nota",
`i%3`/`i%5` para `prescricao`/`execucao`, `i%7` para o titulo acentuado, `i%11`
para o heading. Confirmei pelos proprios goldens: os 20 caminhos de
`so-em-heading.tsv` sao todos multiplos de 11, e os 20 de `com-acento.tsv` todos
multiplos de 7. Sem `rand`.

### N5 — conferido, sem acao

Os quatro pontos que o brief nomeia estao cumpridos:

- **Contexto de backlink:** nos DOIS arquivos a assercao pede uma palavra
  ANTES e uma DEPOIS do link, ambas fora dos colchetes e na linha do link
  (`acordao`/`prescricao` e `resumo`/`historico` em `backlink_contexto_test.go`;
  `esquerda`/`direita` em `backlink_heading_test.go`). O FAIL da mutacao
  `Context: l.Raw` esta colado para os dois, com as seis mensagens.
- **persist:** segunda origem (`Penal/Citante.md`), guarda de fixture exigindo
  duas origens distintas, e `ordenarBacklinks` (`slices.SortFunc` + `cmp.Or`,
  chave TOTAL) nas duas fatias antes do `DeepEqual`. `cmp` e o pacote da stdlib;
  go-cmp continua indireto e nao foi promovido.
- **bm25:** indice real via `createVaultWithNotes`, escada `titulo > heading >
  corpo` com as tres notas de mesmo multiconjunto de tokens, mais a guarda de
  `DocLength` igual que impede a normalizacao de explicar a ordem sozinha. Duas
  mutacoes coladas, exit 0 nas duas.
- **`classify_test.go` fora do brief:** justificado — `ordenarBacklinks` ja
  morava la e duplica-la quebrava a compilacao do pacote de teste. Reforcar a
  existente e "uma conta por regra". A chave passou a ser total, o que so
  fortalece `compararConstrucoes`.

A excecao de `frase-exata` no `conferirDiscriminacao` (`:227-233`) e legitima:
aquela consulta casa uma nota entre 300 por construcao, e a assercao que a
substitui (`len(got) == 1`, com Fatalf) e real, nao vacua.

---

## Veredictos

**Spec: APPROVED**

Os cinco steps do brief foram entregues, e o contrato de relatorio foi cumprido
por inteiro: status, SHA, `head -2` dos seis `.tsv` (o brief dizia quatro; ha
seis, e o implementador corrigiu o brief em vez de entregar quatro), as duas
saidas de `mutate.ps1` do Step 1 e do Step 4 — mais uma terceira, do Step 2, que
o brief pedia como prova manual —, o FAIL do Step 2, a prova do Step 3 por
`-count=20` com e sem ordenacao, o `verify.ps1` de 14 etapas com exit 0, e o
`git diff --stat`. Os tres desvios do brief (enchimento `+ i/60`, o nome
`TestBM25PesoDeCorpoEOMenorDosTres` no lugar de `TestBM25PesoDeHeading`, e
`classify_test.go` fora da lista de arquivos) estao declarados com o motivo, e
os tres motivos se sustentam — o segundo em especial: escrito ao pe da letra o
Step 4 seria duplicata de `TestBM25WeightHeadings`, que ja existe em
`bm25_test.go:94`. Escopo nao encolheu em silencio; encolheu em lugar nenhum.
Verifiquei por amostragem que as saidas coladas nao sao inventadas: todas as
linhas citadas apontam para a assercao que a mensagem nomeia, e as tres ancoras
de mutacao existem no produto.

**Quality: NOT APPROVED**

O trabalho e bom — o corpus discrimina de verdade (conferi que `so-em-heading` e
`com-acento` respeitam `i%11` e `i%7`, e que o quinteto `i ≡ 43 (mod 60)` de
`termo-amplo` saiu com cinco scores distintos), as assercoes novas sao todas
falsificaveis, e a `conferirDiscriminacao` e uma guarda que impede a regressao
de voltar em silencio. O que reprova e B1: a frase "apagar o peso de heading nao
mudava um byte de nenhum `.tsv`" foi escrita como fato medido em duas docstrings
permanentes e na mensagem de commit, e nao e verdade — reproduzi o corpus antigo
por overlay e a mutacao mudou 40 linhas em dois dos seis arquivos, e o teste
antigo reprovava. O achado real da tarefa nao depende dessa frase: o que o
golden antigo congelava era o desempate, e a ORDEM de fato nao se movia.
Corrigir e trocar uma clausula. Mas uma tarefa cujo tema declarado e "teste que
nomeia uma regra e exercita outra coisa" nao pode fechar com uma medicao
inventada na propria docstring que explica o conserto — e o defeito que ela veio
cacar, escrito na explicacao. N1 (comentario que descreve mal o proprio corpus)
anda junto: sao os dois lugares onde o texto afirma mais do que o codigo faz. N2
e N3 sao registro, nao correcao obrigatoria.

## Round 1

### Progresso

- 01:13 abri a secao Round 1 antes de ler o diff da correcao
- 01:13 c572e83 foi emendado em 4c6da0a (mesma arvore, +1 linha no relatorio); o diff de codigo e identico
- 01:14 rodei a mutacao do peso de heading contra os goldens NOVOS via overlay: so-em-heading FAIL (a prova segue viva)
- 01:15 conferi o relatorio (secao "Fix round 1"), o assunto do commit e o comprimento das linhas; fechei o Round 1

### Achado por achado

#### B1 — ADDRESSED

`internal/service/ranking_golden_test.go:86-96` (docstring de `corpusGolden`) e
`:236-241` (docstring de `conferirDiscriminacao`).

O enunciado novo e verdadeiro contra a medicao que eu mesmo rodei. `:88-96`
agora diz que a primeira versao afirmava o que nao tinha medido, e escreve o
resultado real:

> apagar o peso de heading não movia UMA LINHA de lugar em `.tsv` nenhum, e
> mudava a coluna de score de `so-em-heading` e de `dois-termos` — 40 linhas ao
> todo. O segundo entrava porque o heading antigo era `## Execução fiscal` e
> "execucao" é um dos dois termos daquela consulta.

Isso bate frase a frase com o `diff -r goldA goldB` do Round 0: dois arquivos,
40 linhas, so a segunda coluna, sequencia identica. `:239-241` recebeu a mesma
correcao ("não movia UMA LINHA de lugar em arquivo nenhum — só reescrevia a
coluna de score de `so-em-heading` e de `dois-termos`"). A mensagem de commit de
`c572e83` carrega o mesmo enunciado, e `task-162-report.md:343-374` abre uma
secao que corrige explicitamente as duas ocorrencias do relatorio e a de
`d12f84f`, dizendo por que aquela mensagem nao pode ser reescrita e assumindo a
inferencia ("foi inferida de 'as 300 notas sao identicas, entao o peso e fator
comum', que explica a ordem e nao explica a coluna de score"). E o conserto que
a correcao pedia, nos quatro lugares.

#### N1 — ADDRESSED

`internal/service/ranking_golden_test.go:157-159` — "As demais notas trazem só a
forma reduzida, no corpo" virou "As notas com i%3 == 0 trazem a forma reduzida
no corpo; é o contraste que `com-acento` mede." Confere com o gerador: a
`prescricao` do corpo entra em `:183-185`, sob `if i%3 == 0`.

#### N2 — ADDRESSED

`internal/service/ranking_golden_test.go:130-134` — acrescentado que "fiscal"
ocorre exatamente uma vez, sempre num heading, em todas as notas que casam, de
modo que `WeightHeadings` e fator comum e aparece na coluna de score, nao na
sequencia, e que mutar o peso "muda a segunda coluna deste `.tsv` inteiro e não
move nenhuma linha — é assim que ele reprova". A saida re-rodada em
`task-162-report.md:432-478` confirma no proprio texto: `quer` e `tem` com a
mesma sequencia de 20 caminhos.

#### N3 — ADDRESSED

`internal/service/ranking_golden_test.go:190` — `enchimento(20 + (i*7)%60 +
i/60)` virou `enchimento(20 + (i*7+29)%60 + i/60)`, com o motivo em `:100-107`.

Conferi as tres consequencias, nao so a formula:

- **O minimo saiu de `i == 0`.** `7i + 29 ≡ 0 (mod 60)` da `i ≡ 13 (mod 60)`, e
  com o `+ i/60` o unico minimo absoluto (20 tokens) e `i == 13`. `n0000` passou
  a 49 tokens de enchimento.
- **`n0013` nao carrega termo especial e nao venceu nada.** `13 % 3 == 1`,
  `13 % 5 == 3`, `13 % 7 == 6`, `13 % 11 == 2`. Rodei
  `grep -l n0013 internal/service/testdata/ranking/*.tsv`: **nao aparece em
  `.tsv` nenhum**. A nota mais curta do corpus deixou de ser candidata.
- **`n0000` nao e mais o primeiro de golden algum.** Primeiras linhas hoje:

  ```
  com-acento.tsv     pasta08/n0168.md	1.705403
  dois-termos.tsv    pasta05/n0245.md	3.758047
  frase-exata.tsv    pasta00/n0150.md	5.738904
  so-em-heading.tsv  pasta06/n0176.md	4.059071
  so-no-titulo.tsv   pasta05/n0005.md	7.070732
  termo-amplo.tsv    pasta09/n0039.md	0.003200
  ```

  Em `com-acento` ele caiu para 6o, em `dois-termos` para 3o, em
  `so-em-heading` para 14o. A forma "a nota mais curta ganha" saiu dos tres
  goldens onde estava.

O topo de `termo-amplo` melhorou de brinde: `n0039` tem 22 tokens de enchimento
e `tf("nota") = 1 + 39%4 = 4`, entao a primeira posicao passou a ser decidida
por frequencia contra comprimento, e nao so por comprimento.

### Os seis goldens continuam discriminando, e nada foi enfraquecido

**Nenhuma assercao mudou.** Isolei do diff toda linha `+`/`-` que nao e
comentario nem `.tsv`, e ha exatamente uma:

```
-		corpo.WriteString(enchimento(20 + (i*7)%60 + i/60))
+		corpo.WriteString(enchimento(20 + (i*7+29)%60 + i/60))
```

`conferirDiscriminacao` esta intacta — as tres guardas (`frase-exata` com
`len == 1`, `got[0].Score > got[1].Score`, e o piso de metade de scores
distintos) sao as mesmas, e as seis passam:

```
$ go test ./internal/service/ -run TestRankingGolden -v -count=1
--- PASS: TestRankingGolden (0.75s)
    --- PASS: TestRankingGolden/termo-amplo (0.00s)
    --- PASS: TestRankingGolden/dois-termos (0.00s)
    --- PASS: TestRankingGolden/frase-exata (0.00s)
    --- PASS: TestRankingGolden/com-acento (0.00s)
    --- PASS: TestRankingGolden/so-no-titulo (0.00s)
    --- PASS: TestRankingGolden/so-em-heading (0.00s)
```

**A prova de mutacao segue viva sobre os goldens novos.** Re-rodei eu mesmo, por
overlay, sem tocar a arvore (so `bm25.go` trocado, `return WeightHeadings` ->
`return WeightBody`):

```
$ go test -overlay=mutonly.json ./internal/service/ -run TestRankingGolden -count=1
--- FAIL: TestRankingGolden (0.53s)
    --- FAIL: TestRankingGolden/so-em-heading (0.00s)
FAIL	github.com/jonyd/gobsidian/internal/service	1.160s
```

As regras que o Round 0 tinha conferido continuam valendo apos a regeneracao:
os 20 caminhos de `so-em-heading.tsv` sao todos multiplos de 11 (176, 22, 253,
99, 220, 297, 143, 66, 33, 264, 110, 187, 77, 0, 154, 231, 44, 121, 198, 275) e
os 20 de `com-acento.tsv` todos multiplos de 7 (inclusive os novos 56, 133, 14,
91, 245). `so-no-titulo.tsv` mantem as cinco notas de titulo antes de `n0250`,
que so tem o termo no corpo, e `frase-exata.tsv` continua com uma linha so. O
quinteto `i ≡ 43 (mod 60)` de `termo-amplo` continua com cinco scores distintos
(0.003051 / 0.003046 / 0.003041 / 0.003036 / 0.003030), entao o `+ i/60` ainda
faz o que a docstring diz.

### Assunto do commit

`test(ranking): the old golden was order-inert, not byte-inert; n0000 stops
being the shortest note` — Conventional Commits, ingles, e nomeia as duas
metades da rodada. Bate com o ruling.

Nota de rastreio: `c572e83` foi emendado em `4c6da0a`, que e o HEAD. As arvores
sao identicas em `internal/`; a emenda so acrescentou a linha de Progresso das
01:12 ao relatorio. O diff de codigo que julguei e o mesmo nos dois.

### Achados novos desta rodada — todos NON-BLOCKING

**R1** — `task-162-report.md:21-23` — `## Status` continua **"DONE. SHA
`d12f84f`."**, que agora nomeia so metade da entrega. A linha de Progresso das
01:12 explica bem por que o SHA da rodada nao pode ser escrito dentro do proprio
commit, mas o Status como esta afirma um SHA unico para um trabalho que tem
dois. **Fix:** "DONE. SHAs: `d12f84f` (rodada 0) + o commit desta rodada (a
mensagem ao lead e `git log` o trazem)."

**R2** — `task-162-report.md:54-108` — o bloco `head -2` dos seis `.tsv` e o
`so-no-titulo.tsv inteiro` mostram os numeros da rodada 0
(`pasta00/n0000.md 1.716162`, `7.022125`, ...), que nenhum arquivo do repo tem
mais; e a secao "Uma tentativa que reprovou" descreve como vigente a formula
`20 + (i*7)%60 + i/60`. Sao evidencia historica legitima, e a secao
`### N3` (`:389-431`) traz o `head -3` atual dos seis, que confere com o disco —
mas nada marca os blocos antigos como superados, e quem cruzar o relatorio com
os arquivos encontra seis discrepancias antes de chegar na secao que as explica.
**Fix:** uma linha sob cada bloco — "Valores da rodada 0; a rodada 1 regenerou
os seis, ver `### N3`."

**R3** — `internal/service/ranking_golden_test.go:239` — a linha tem 147 colunas
contra o embrulho de ~78 do arquivo inteiro (e a unica acima de 90 que nao e
codigo). Nasceu da reescrita do B1: o texto novo foi emendado no fim da frase
sem re-embrulhar o paragrafo. `gofmt` nao re-embrulha comentario, entao o
`verify.ps1` passou verde e vai continuar passando. **Fix:** quebrar em tres
linhas de ~78.

### Veredictos

**Spec: APPROVED**

Os quatro achados foram atacados no lugar certo e pelo mecanismo certo, nao por
enfraquecimento. B1 esta corrigido nos quatro lugares onde a frase falsa morava
— as duas docstrings, a mensagem do commit novo, e um paragrafo do relatorio que
assume a inferencia e explica por que a mensagem de `d12f84f` fica como esta. N1
e N2 sao trocas de texto que conferi contra o codigo que descrevem. N3 aplicou o
ruling literal (`20 + ((i*7+29)%60) + i/60`), e as tres consequencias que
importavam se verificam: o minimo mudou para `i == 13`, `n0013` nao carrega
termo especial e nao aparece em `.tsv` nenhum, e `n0000` deixou de encabecar os
tres goldens que encabecava. O relatorio traz a rodada em secao propria, com o
`head -3` atual dos seis `.tsv` (bate com o disco), a mutacao re-rodada e o
`verify.ps1` de 14 etapas com exit 0.

**Quality: APPROVED**

Exatamente uma linha executavel mudou no arquivo de teste — a formula do
enchimento —, e conferi isso isolando do diff toda linha `+`/`-` que nao e
comentario. Nenhuma guarda de `conferirDiscriminacao` foi afrouxada para os
goldens novos passarem: as tres continuam la e os seis subtestes passam, e
re-rodei por conta propria a mutacao do peso de heading contra os goldens novos
para confirmar que a prova nao morreu na regeneracao — `so-em-heading` reprova.
Amostrei as invariantes do corpus depois do `-update` (multiplos de 11 em
`so-em-heading`, de 7 em `com-acento`, o contraste titulo-vs-corpo em
`so-no-titulo`, a nota unica em `frase-exata`, o quinteto `i ≡ 43 (mod 60)` ainda
distinto) e todas se mantem. R1, R2 e R3 sao arrumacao de relatorio e uma linha
mal embrulhada; nenhum deles muda o que o teste cobre, e nenhum justifica
segurar a tarefa. Zero blocking.
