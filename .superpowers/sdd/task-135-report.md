# Task 135 — A8 + B11 (+B14): o contexto do backlink, e o portão que tinha dois donos

**Status:** DONE. A correção está completa e provada. O contexto ficou em **80
bytes de cada lado** e o backlink ganhou o **título da seção de origem**
(`Heading`), derivado a custo zero de disco. Formato do cache **2 → 5** ao longo
da tarefa: 3 acrescentou o contexto, 4 cortou para 40 sobre uma medição que se
provou errada, 5 voltou para 80 com o número correto. Decisões do dono em
2026-08-26.

**Este relatório contém DUAS retratações de números que eu mesmo publiquei.** As
duas vieram do mesmo defeito de método: comparar variantes de desempenho em
bateladas sequenciais. Ver "Retratação" ao fim.
**Commit:** `7c24c67` — `fix(index,parser): fill backlink context, unify the cache version gate`

As duas foram juntas por necessidade, não por conveniência: implementar A8 exige
subir o formato do cache, e **é exatamente esse bump que torna o B11 perigoso**.

---

## A8 — um campo que prometia e nunca entregou

`index.Backlink` tinha o campo:

```go
Context string // texto ao redor da referencia
```

E `docs/TOOLS.md:164` **promete**, na normativa:

> `backlinks` traz origem e o contexto textual ao redor de cada referência.

Chegava vazio ao host em toda resposta de `vault_backlinks`, desde sempre.
`internal/service/graph.go:452` serializa `[]index.Backlink` direto para o JSON,
então o campo **aparecia** na resposta — sempre `""`.

### A causa não é um esquecimento, é a forma

Havia **três** construções de `Backlink`, e as três escreviam `Context: ""` à
mão: `backlinks.go:20`, `update.go:154` e `update.go:494`. Não havia um lugar
onde preencher o valor; havia três para esquecer. É o padrão que este projeto
nomeia como "uma conta por regra", e a violação estava em triplicata.

Colapsadas num construtor único:

```go
func backlinkDe(from vault.CanonicalPath, l ResolvedLink) Backlink
```

O mesmo valia um nível acima: `ResolvedLink{Link: l}` era montado à mão em
`index.go:133` (Build) e `update.go:142` (Replace). Agora existe `montarLinks`,
único ponto que monta `[]ResolvedLink`, e é lá que o contexto é recortado.

### Onde o corpo existe, e onde não existe

`contextoDoLink` precisa dos bytes da nota, e a struct `parsed` que atravessa o
canal do Build **não os carrega** — de propósito: ela cruza o cofre inteiro em
voo, e levar os bytes de cada nota até o coletor multiplicaria o pico de memória
pelo tamanho do cofre. Então o recorte acontece **no worker**, onde os bytes já
estão na mão, e só o contexto viaja.

Uma sutileza que teria produzido recorte deslocado: `ShiftOffsets` realinha os
offsets ao arquivo **bruto** quando há BOM, então `montarLinks` recebe `data`, e
não `body`. Recortar de `body` erraria `BOMLen` bytes em toda nota com BOM.

### Por que persistir, se `Resolved`/`Via`/`State` não são

O comentário do codec diz que esses três **não** são gravados porque são
recalculados na carga pelas mesmas funções que o Build chama, e persistir seria
uma segunda forma de calcular o mesmo dado.

O contexto é a exceção, e pelo **mesmo critério**: ele não é derivável do que o
índice guarda. Recalculá-lo exigiria reler o corpo de cada nota no boot — o custo
exato que este cache existe para não pagar. Deixá-lo de fora daria duas respostas
diferentes para `vault_backlinks` conforme o cache tenha sido usado ou não.

Daí o bump de formato: **2 para 3**.

---

## B11 — o portão com duas constantes

Duas constantes independentes guardavam o mesmo portão:

| onde | constante | conferida em |
|---|---|---|
| `persist.go:22` | `IndexCacheFormatVersion` | `LoadIndexCache` |
| `persist_codec.go:44` | `indexCacheCodecVers` | cabeçalho, no decodificador |

Tinham o mesmo valor porque alguém digitou o mesmo número duas vezes. **Subir uma
sem a outra não quebra build nem teste**: faz o leitor recusar todo save que o
próprio processo acabou de gravar — reconstrução completa a cada boot, e nenhum
log dizendo por quê. O sintoma de campo pareceria "cache lento", não "cache
quebrado".

Fechado com um alias, não uma cópia:

```go
indexCacheCodecVers = IndexCacheFormatVersion
```

Para que o bump seja **impossível de fazer pela metade**.

---

## B14 — de brinde, porque a sondagem passou por cima dele

`parser/types.go` avisava que `Start`/`End` "só são preenchidos para LinkWiki e
LinkEmbed", e que link Markdown fica em `offsetUnknown`. Como o contexto depende
justamente desses offsets, sondei antes de confiar:

```
kind=wikilink target="wiki"    start=  6 end= 14 raw="[[wiki]]"   span="[[wiki]]"
kind=markdown target="alvo.md" start= 20 end= 35 raw="alvo.md"    span="[alvo](alvo.md)"
kind=embed    target="e.png"   start= 40 end= 50 raw="![[e.png]]" span="![[e.png]]"
kind=embed    target="x.png"   start= 53 end= 66 raw="x.png"      span="![alt](x.png)"
```

**As quatro grafias trazem span correto**, embed Markdown incluído — o caso que o
comentário nomeava. Era verdade quando foi escrito e deixou de ser.

A armadilha real é outra, e continua: **`Raw` não cobre o mesmo trecho que
`Start:End`** em Markdown — ali `Raw` traz só o destino. Comentário reescrito com
a tabela medida.

`offsetUnknown` **continua caminho vivo** (três saídas em `ast.go`), então a
guarda `Start < 0` em `contextoDoLink` é necessária, não decorativa.

---

## Provas de mutação

Três, todas `EXIT=0`, e cada uma reproduz um sintoma **de campo** distinto.

### 1. O contexto é de verdade recortado

```
-Anchor 'trecho := strings.TrimSpace(string(body[ini:fim]))'
-Replacement '...; trecho = ""'

--- FAIL: TestBacklinkTrazContexto
    backlink_contexto_test.go:49: kind=wikilink: Context vazio — o campo continua sendo promessa nao cumprida
    backlink_contexto_test.go:49: kind=markdown: Context vazio — o campo continua sendo promessa nao cumprida
    backlink_contexto_test.go:67: os dois backlinks tem o MESMO contexto "": o recorte ignora a posicao do link
    backlink_contexto_test.go:77: nenhum contexto trouxe a vizinhanca da referencia wikilink; contextos="", ""
```

É o defeito A8 exato, reproduzido.

### 2. B11 — a prova central

```
-Anchor 'indexCacheCodecVers = IndexCacheFormatVersion'
-Replacement 'indexCacheCodecVers = 2'

--- FAIL: TestContextoSobreviveAoCacheDeMetadados
    backlink_contexto_test.go:116: LoadIndexCache recusou um cache que ESTE processo
    acabou de gravar: index cache version mismatch
        e o sintoma do B11 — as duas constantes de versao divergiram
```

**Antes deste teste, essa mutação passaria em silêncio.** É o que faltava para o
portão ter dono.

### 3. O contexto atravessa o disco

```
-Anchor 'e.str(rl.Context)'  -Replacement 'e.str("")'

--- FAIL: TestContextoSobreviveAoCacheDeMetadados
    backlink_contexto_test.go:126: backlink 0: contexto "" depois do cache,
    "# Origem O acordao do RESP 1234 foi superado pelo entendimento de [[Alvo]]
     quanto a prescricao. Ver tambem o resumo em [o alvo](Alvo.md) para o historico" antes
        o cache degrada a resposta: mesma pergunta, respostas diferentes conforme houve boot frio
```

---

## O custo, medido

Cofre real, 5.686 notas, 42.329 links, 109 MB. Cache gravado com e sem o campo,
medindo o mesmo cofre nas duas rodadas (a segunda por mutação do codec, com o
arquivo restaurado byte a byte depois — SHA-256 conferido antes e depois).

| | formato 2 | formato 3 |
|---|---|---|
| arquivo | 19,53 MB | **32,62 MB** |
| delta | — | **+13,09 MB (+67%)** |
| `LoadIndexCache`, mediana de 5 | 275,5 ms | 282,2 ms |
| amostras | 258–292 ms | 236–450 ms |

**O custo de disco é real e está medido: +309 bytes por link.**

**O custo de tempo NÃO é distinguível de ruído nesta amostra.** As distribuições
se sobrepõem, e a rodada *com* contexto produziu as duas amostras mais rápidas
(236 e 241 ms). O delta mediano de +6,7 ms é o que este formato acrescenta.

### O boot completo, medido depois (RNF-02)

O relatório fechou com isto em aberto; o dono mandou medir. Feito no mesmo cofre,
com `index_ms` da linha `servidor pronto`, em processo (`GOBSIDIAN_NO_DAEMON=1`),
somente-leitura e com `--cache-dir` próprio — **para não gravar formato 3 no cache
que as sessões vivas do dono leem**: o binário instalado é anterior, recusaria, e
todas elas reconstruiriam o índice no próximo boot. O binário foi remontado para o
formato 2 por mutação do codec, medido, e o arquivo restaurado byte a byte
(SHA-256 conferido: `7cca05addc8b4116` antes e depois).

| | formato 2 | formato 3 |
|---|---|---|
| boot quente, mediana de 5 | **891 ms** | **921 ms** |
| amostras | 810 / 871 / 891 / 930 / 1079 ms | 872 / 887 / 921 / 988 / 1034 ms |
| boot frio, n=1 | 1741 ms | 2326 ms |

**RNF-02 estourado nas duas, e já estava.** 891 ms contra 300 ms **com o formato
antigo** — condição preexistente nesta escala, não regressão desta tarefa. O
RNF-02 já é publicado como NÃO ATINGIDO em `OPERACAO.md` desde 2026-08-06; o que
esta medição acrescenta é a escala (5.686 notas: 810–1079 ms, contra os 371–472 ms
publicados) e a prova de que o formato 3 não é a causa.

**O delta de +30 ms na mediana não é distinguível de ruído.** As faixas se
sobrepõem quase inteiras, e a do formato 2 é a **mais larga** das duas — 269 ms
contra 162 ms. Uma amostra do formato 2 (1079 ms) estourou até o limite de falha
de 1 s, e nenhuma do formato 3: isso sozinho mostra o tamanho do ruído. Remover o
contexto não devolveria o RNF-02.

**O boot frio tem uma amostra só de cada e não sustenta conclusão.** A diferença
está na direção que se esperaria — `index_ms` inclui `SaveIndexCache` e o formato
3 grava 13 MB a mais —, mas com n=1 é hipótese, não medida. As duas passam no alvo
de 3 s do RNF-01.

### Correção: fora do OneDrive o custo aparece, e cruza a linha

A conclusão acima — "não distinguível de ruído" — **vale para aquele cofre e eu a
generalizei demais.** Escrevi que "remover o contexto não devolveria o RNF-02".
Num cofre em OneDrive isso é verdade; não é uma afirmação sobre o formato 3.

O dono mandou medir fora do OneDrive. `Obsidian\Jurisprudência`, **1.254 notas,
90 anexos**, disco local. Duas bateladas de cada formato, em ordens alternadas,
agrupadas — **n=13 cada**:

| | formato 2 | formato 3 |
|---|---|---|
| cache | 9,76 MB | **19,05 MB (+95%)** |
| boot quente, mediana de 13 | **243 ms** | **323 ms** |
| faixa | 214–433 ms | 284–415 ms |
| acima do teto de 300 ms | **3 de 13** | **10 de 13** |

**Aqui o formato 3 empurra o RNF-02 de atingido para não atingido.** As caudas se
sobrepõem, mas a separação é forte onde importa: **9 das 13 amostras do formato 2
ficam abaixo da MENOR amostra do formato 3** (284 ms), e nenhuma do formato 3
desce abaixo disso.

O cofre em OneDrive escondeu o efeito porque lá o ruído tem ~270 ms de largura
contra um efeito de ~90 ms — e o RNF-02 já estava 3× estourado nos dois formatos,
o que tornava a comparação acadêmica. No cofre local a métrica vive **em cima da
linha**, e é onde 80 ms decidem.

### A ordem das bateladas quase produziu o número errado

A primeira batelada do formato 3 tocou 1,3 GB com o cache de arquivo do SO **frio**
— boot frio de 4.534 ms, contra 765 ms na repetição com o SO quente. A batelada do
formato 2 rodou depois, com tudo aquecido. Comparar aquelas duas daria um custo
muito maior do que o real.

Foi o que me obrigou a repetir: cada formato medido duas vezes, em ordens
diferentes, e as bateladas agrupadas. **O primeiro par de números teria passado por
medição.**

### O que continua não medido

**Onde estão os ~600 ms restantes.** `LoadIndexCache` isolado mede 275–282 ms
neste mesmo cofre e o boot mede ~900 ms, então a maior parte do tempo está fora do
codec. `VerifyFreshness` faz `Stat` em cada um dos 5.686 arquivos, num cofre em
OneDrive, e é o suspeito óbvio — **mas suspeito não é medida**, e não fiz nenhuma
para confirmá-lo.

---

## Decisão que fica com o dono

**+67% de cache é o preço de honrar um contrato que a normativa já publicava.** O
tamanho é governado por uma constante única, `contextoBytes = 80`, em
`internal/index/contexto_link.go` — deixada num lugar só justamente para poder ser
mudada sem caçar ocorrências. Metade dela é aproximadamente metade do custo.

Existe uma alternativa que **não** implementei, e registro para não parecer que
não foi considerada: **não persistir nada e recortar o contexto na hora da
consulta**, lendo o corpo só das notas de origem que a resposta realmente cita —
tipicamente poucas. Custo em disco: zero. Custos do outro lado: uma leitura de
disco por backlink devolvido, e o choque com a regra "arquivo somente-nuvem nunca
é aberto" — uma origem em placeholder do OneDrive dispararia download síncrono ou
voltaria sem contexto, e aí o campo volta a ser inconstante, que é o defeito que
esta tarefa fecha. Foi por isso que segui pelo cache; a troca continua disponível
se o disco pesar mais que a leitura.

Uma observação de produto que a medição expôs: **em nota curta, o contexto é a
nota inteira** — 80 bytes de cada lado cobrem tudo. Não é defeito (a promessa é
limitada e cumprida), mas explica por que dois backlinks da mesma nota curta
trazem trechos muito parecidos.

---

## Retratação: o custo de boot que eu publiquei duas vezes não existe

Publiquei que o formato 3 empurrava o RNF-02 de atingido para não atingido no
cofre local — mediana de 243 ms contra 323 ms, faixas quase disjuntas. **Está
errado.**

Aquelas bateladas rodaram em sequência, com a máquina carregada pelas próprias
medições. Eu tinha alternado a ORDEM das bateladas, achando que bastava. **Não
basta**: as bateladas continuam separadas no tempo, e a deriva de carga entre elas
vira "diferença entre formatos".

Refeito com os três binários construídos lado a lado e **uma rodada de cada por
vez**, dez vezes, cada um com seu `--cache-dir`, máquina em repouso:

| variante | cache | mediana de 10 | faixa | acima do teto de 300 ms |
|---|---|---|---|---|
| sem contexto | 9,76 MB | 179 ms | 163–237 ms | **0 de 10** |
| contexto de 80 | 19,05 MB | 193 ms | 159–231 ms | **0 de 10** |
| contexto de 40 + heading | **16,95 MB** | 191 ms | 147–245 ms | **0 de 10** |

**Os três passam no RNF-02 nesse cofre.** As medianas diferem em 14 ms — menos que
a variação dentro de uma única variante, cuja faixa mais larga tem 74 ms.

**O que sobrevive à medição é o tamanho do cache**, que é determinístico: 9,76 →
19,05 → 16,95 MB. Foi isso, e não o tempo, que justificou cortar para 40.

Foi o **segundo** número errado da sessão pela mesma causa. O primeiro — "o delta
não é distinguível de ruído" no cofre em OneDrive — estava certo por acaso: lá o
RNF-02 já estava 3× estourado nos dois formatos, o que tornava a comparação
irrelevante de qualquer jeito. Registrado em `ARMADILHAS.md`.

## O heading, e por que ele custa zero

`Backlink.Heading` é o título da seção em que a referência está. Não é persistido:
`Note.Headings` já vai para o cache com offsets, e `Link.Start` vem da **mesma
origem de offset** (dito no comentário de `parser.Heading`). Então o título é
derivado na hora de montar o backlink.

O critério é o mesmo que mantém `Resolved`/`Via`/`State` fora do disco e que
colocou `Context` dentro: **derivável do que o índice já guarda fica de fora.** O
contexto precisa dos bytes do corpo, que o cache não tem; o heading não precisa.

`TestContextoSobreviveAoCacheDeMetadados` passou a conferir o `Heading` no
round-trip, e a falhar se ele vier vazio ANTES do cache — senão compararia `""`
com `""` e passaria sem exercitar nada.

### A guarda que a mutação matou

`headingDoLink` nasceu com `if start < 0 { return "" }` para tratar
`offsetUnknown`. A prova de mutação devolveu **EXIT=1**: o teste passou sem ela.
A mutação não estava errada — **a guarda era código morto.** Com `start = -1`,
todo heading tem `Start > -1`, o laço quebra na primeira volta, e o resultado já é
`""`. Removida, com o motivo escrito no lugar.

## Provas de mutação do corte e do heading

```
-Anchor 'if h.Start > start {' -Replacement 'if false {'
  (TestBacklinkTrazHeadingDaSecao)                                   -> EXIT=0
    "nenhum backlink na secao "Prescricao"; vistos=map[Honorarios:true]"
    "os dois backlinks caíram no MESMO heading: o calculo ignora a posicao do link"

-Anchor 'if h.Start > start {' -Replacement 'if false {'
  (TestHeadingVazioAntesDoPrimeiroTitulo)                            -> EXIT=0
    "Heading = "Depois", queria vazio: a referencia esta ANTES do primeiro
     titulo, e devolver "Depois" seria apontar uma secao que vem depois dela"

-Anchor 'const contextoBytes = 40' -Replacement 'const contextoBytes = 80'
  (TestContextoCurtoNaoEstouraOOrcamento)                            -> EXIT=0
    "contexto tem 168 bytes, teto 96: contextoBytes voltou a crescer"
```

A mesma âncora serve aos dois primeiros e eles reprovam por motivos **diferentes**
— um perde a variação por seção, o outro passa a apontar uma seção posterior à
referência.

## Verificações

1. `go test -race ./internal/index/ ./internal/parser/ ./internal/service/`:
   **os três verdes** (índice 5,9 s; parser 2,2 s; service 89,8 s).
2. `pwsh -File scripts/verify.ps1`: **bateria completa, EXIT=0** — build,
   `go test -race`, tetos de latência, `go vet` nos três GOOS, `gofmt`,
   `golangci-lint` (Windows e Linux), `check_net`, `check_tool_params`,
   `check_doc_refs`, `check_readme_anchors`.
3. Nenhuma mudança em `docs/TOOLS.md`: **a normativa já estava certa**, e era o
   código que a contradizia. Foi a doc que denunciou o defeito, não o contrário.
   Registrado em `ARMADILHAS.md`; formato e medições em `ESTADO.md`.
