# Task 135 — A8 + B11 (+B14): o contexto do backlink, e o portão que tinha dois donos

**Status:** DONE_WITH_CONCERNS — a correção está completa e provada; **o custo em
disco é de +67% no cache de metadados, medido, e merece decisão do dono.**
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

### O que NÃO foi medido, e importa

**O boot completo deste cofre contra o teto de 300 ms do RNF-02.** Medi
`LoadIndexCache` isolado, não o boot. O RNF-02 foi publicado num cofre sintético
de 3.149 notas (208–282 ms); este tem 5.686 notas. Se ele já estoura o teto
**hoje**, é condição preexistente e não desta tarefa — mas eu **não verifiquei**,
e não afirmo em nenhuma das duas direções.

Não medi porque o cofre é um dos que o dono usa em sessão viva, e subir `serve`
contra ele arriscaria conflitar com as pontes em execução.

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
