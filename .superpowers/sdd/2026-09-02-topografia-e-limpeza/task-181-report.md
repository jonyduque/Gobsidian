# Task 181 report — white-box tests for the metadata cache codec

## Status

DONE_WITH_CONCERNS

(concerns = um achado de teste real no codec, não corrigido por estar fora do
escopo desta tarefa de teste/doc — ver `## Concerns`)

## Progresso

- 14:59 iniciado; base commit confirmado 833906f2bc17dc4b260d3121aa7fbe992120e883
- 15:00 Step 1 (cobertura antes) medida
- 15:00 lido `internal/index/persist_codec.go` inteiro; confirmadas as correções do
  despacho (escritor.w é *bufio.Writer, leitor{b,i,err}, âncoras nas linhas citadas)
- 15:00 lido `internal/index/note.go` e `internal/parser/types.go` para os tipos de
  Note/Heading/Block/Link usados no teste de nota truncada
- 15:01 escrito `internal/index/persist_codec_test.go`, adaptando `roundTrip` para
  `bufio.NewWriter` + `Flush` (correção 1 do despacho)
- 15:01 achado: `TestCodecTagDesconhecidaERecusada` como escrito no brief falharia —
  a leitura da tag já passa pelo uvarint com limite=valMap, então a mensagem real é
  "acima do limite", não "tag de valor desconhecida" (o branch default do switch é
  código morto). Adaptado o teste para afirmar o comportamento real, com comentário;
  registrado como B21 em SUGESTOES.md e no bloco de cobertura de ESTADO.md — não
  mexido em persist_codec.go
- 15:01 `go vet ./internal/index/...` limpo
- 15:01 `go test -race ./internal/index/ -run TestCodec -v` — 8/8 PASS
- 15:01 `go test ./internal/index/` (suite inteira) — PASS
- 15:01 mutação 1 (profundidade): exit 0, teste FAIL sob mutação confirmado
- 15:01 mutação 2 (valInt64): exit 0, teste FAIL sob mutação confirmado
- 15:02 cobertura depois medida
- 15:03 `docs/ESTADO.md` e `docs/SUGESTOES.md` editados; UTF-8 validado nos dois
- 15:04 `pwsh -File scripts/verify.ps1 -SkipCross -SkipNet` — verde
- 15:07 commit criado
- 15:08 `pwsh -File scripts/audit_reports.ps1 181` rodado (relatório ainda
  incompleto; achados corrigidos na versão final)
- 15:11 relatório completo escrito; `audit_reports.ps1 181` re-rodado — 0
  achados novos no relatório, só os 14 pré-existentes do ledger antigo

## Commits

SHA: `b2561e0760cc037cffffc5e25494879c4f9c52fc`
Subject: `test(index): white-box tests for the metadata cache codec`

## Cobertura antes

```
go test ./internal/index/ -coverprofile=$TEMP/cov_antes.out
ok  	github.com/jonyd/gobsidian/internal/index	1.952s	coverage: 83.5% of statements

go tool cover -func=$TEMP/cov_antes.out | grep persist_codec.go
github.com/jonyd/gobsidian/internal/index/persist_codec.go:110:	uvarint				75.0%
github.com/jonyd/gobsidian/internal/index/persist_codec.go:118:	varint				75.0%
github.com/jonyd/gobsidian/internal/index/persist_codec.go:126:	fixed64				0.0%
github.com/jonyd/gobsidian/internal/index/persist_codec.go:135:	str				75.0%
github.com/jonyd/gobsidian/internal/index/persist_codec.go:143:	boolean				100.0%
github.com/jonyd/gobsidian/internal/index/persist_codec.go:156:	timeBlob			60.0%
github.com/jonyd/gobsidian/internal/index/persist_codec.go:175:	strSlice			100.0%
github.com/jonyd/gobsidian/internal/index/persist_codec.go:186:	headings			100.0%
github.com/jonyd/gobsidian/internal/index/persist_codec.go:202:	blocks				37.5%
github.com/jonyd/gobsidian/internal/index/persist_codec.go:228:	links				100.0%
github.com/jonyd/gobsidian/internal/index/persist_codec.go:246:	inline				27.3%
github.com/jonyd/gobsidian/internal/index/persist_codec.go:266:	value				65.8%
github.com/jonyd/gobsidian/internal/index/persist_codec.go:329:	note				100.0%
github.com/jonyd/gobsidian/internal/index/persist_codec.go:348:	asset				100.0%
github.com/jonyd/gobsidian/internal/index/persist_codec.go:361:	escreveIndexCache		85.7%
github.com/jonyd/gobsidian/internal/index/persist_codec.go:410:	falha				0.0%
github.com/jonyd/gobsidian/internal/index/persist_codec.go:416:	uvarint				57.1%
github.com/jonyd/gobsidian/internal/index/persist_codec.go:433:	uvarintLivre			62.5%
github.com/jonyd/gobsidian/internal/index/persist_codec.go:446:	varint				62.5%
github.com/jonyd/gobsidian/internal/index/persist_codec.go:459:	fixed64				0.0%
github.com/jonyd/gobsidian/internal/index/persist_codec.go:472:	str				77.8%
github.com/jonyd/gobsidian/internal/index/persist_codec.go:487:	boolean				100.0%
github.com/jonyd/gobsidian/internal/index/persist_codec.go:491:	timeBlob			61.5%
github.com/jonyd/gobsidian/internal/index/persist_codec.go:510:	strSlice			75.0%
github.com/jonyd/gobsidian/internal/index/persist_codec.go:529:	headings			75.0%
github.com/jonyd/gobsidian/internal/index/persist_codec.go:555:	blocks				33.3%
github.com/jonyd/gobsidian/internal/index/persist_codec.go:578:	links				75.0%
github.com/jonyd/gobsidian/internal/index/persist_codec.go:610:	inline				30.8%
github.com/jonyd/gobsidian/internal/index/persist_codec.go:630:	value				61.1%
github.com/jonyd/gobsidian/internal/index/persist_codec.go:690:	note				90.9%
github.com/jonyd/gobsidian/internal/index/persist_codec.go:740:	asset				83.3%
github.com/jonyd/gobsidian/internal/index/persist_codec.go:759:	leIndexCache			80.0%
```

## Testes

Antes deste commit, `internal/index/persist_codec_test.go` não existia — não há
uma versão RED intermediária do arquivo de teste em si (o arquivo nasceu junto
com os testes); a evidência RED→GREEN real é a das duas provas de mutação abaixo
(regra removida → teste nomeia a falha → regra restaurada → teste passa), que é
o padrão descrito em `docs/papeis/testador.md`.

```
$ go vet ./internal/index/...
(sem saída — limpo)

$ go test -race ./internal/index/ -run TestCodec -v
=== RUN   TestCodecValorRoundTripPorTipo
--- PASS: TestCodecValorRoundTripPorTipo (0.02s)
=== RUN   TestCodecStrSliceNilDistintoDeVazio
--- PASS: TestCodecStrSliceNilDistintoDeVazio (0.00s)
=== RUN   TestCodecTimeBlobPreservaZona
--- PASS: TestCodecTimeBlobPreservaZona (0.00s)
=== RUN   TestCodecValorTipoNaoSuportadoFalha
--- PASS: TestCodecValorTipoNaoSuportadoFalha (0.00s)
=== RUN   TestCodecValorProfundidadeAlemDoLimiteERecusada
--- PASS: TestCodecValorProfundidadeAlemDoLimiteERecusada (0.00s)
=== RUN   TestCodecTagDesconhecidaERecusada
--- PASS: TestCodecTagDesconhecidaERecusada (0.00s)
=== RUN   TestCodecStringAcimaDoLimiteERecusada
--- PASS: TestCodecStringAcimaDoLimiteERecusada (0.00s)
=== RUN   TestCodecNotaTruncadaEmCadaByteERecusada
--- PASS: TestCodecNotaTruncadaEmCadaByteERecusada (0.00s)
PASS
ok  	github.com/jonyd/gobsidian/internal/index	1.717s

$ go test ./internal/index/
ok  	github.com/jonyd/gobsidian/internal/index	1.737s
```

## Mutações

```
$ pwsh -File scripts/mutate.ps1 -Path internal/index/persist_codec.go -Anchor 'if profundidade > limiteValorProfund {' -Replacement 'if false {' -Test TestCodecValorProfundidadeAlemDoLimiteERecusada -Package ./internal/index/
Carregado em 460ms
[...] Mutando internal/index/persist_codec.go
      - if profundidade > limiteValorProfund {
      + if false {

[...] go test -race -run TestCodecValorProfundidadeAlemDoLimiteERecusada ./internal/index/
----------------------------------------------------------------------
--- FAIL: TestCodecValorProfundidadeAlemDoLimiteERecusada (0.00s)
    persist_codec_test.go:148: profundidade 70 devia ser recusada na leitura, tenho <nil>
FAIL
FAIL	github.com/jonyd/gobsidian/internal/index	0.687s
FAIL
----------------------------------------------------------------------
[OK] internal/index/persist_codec.go restaurado byte a byte (SHA-256 confere).

[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
EXIT=0
```

```
$ pwsh -File scripts/mutate.ps1 -Path internal/index/persist_codec.go -Anchor 'case valInt64:' -Replacement 'case valInt64 + 100:' -Test TestCodecValorRoundTripPorTipo -Package ./internal/index/
Carregado em 443ms
[...] Mutando internal/index/persist_codec.go
      - case valInt64:
      + case valInt64 + 100:

[...] go test -race -run TestCodecValorRoundTripPorTipo ./internal/index/
----------------------------------------------------------------------
--- FAIL: TestCodecValorRoundTripPorTipo (0.00s)
    persist_codec_test.go:47: lendo 1099511627776: index cache file corrupted: tag de valor desconhecida 3
FAIL
FAIL	github.com/jonyd/gobsidian/internal/index	0.732s
FAIL
----------------------------------------------------------------------
[OK] internal/index/persist_codec.go restaurado byte a byte (SHA-256 confere).

[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
EXIT=0
```

## Cobertura depois

```
go test ./internal/index/ -coverprofile=$TEMP/cov_depois.out
ok  	github.com/jonyd/gobsidian/internal/index	1.684s	coverage: 89.7% of statements

go tool cover -func=$TEMP/cov_depois.out | grep persist_codec.go
github.com/jonyd/gobsidian/internal/index/persist_codec.go:110:	uvarint				75.0%
github.com/jonyd/gobsidian/internal/index/persist_codec.go:118:	varint				75.0%
github.com/jonyd/gobsidian/internal/index/persist_codec.go:126:	fixed64				80.0%
github.com/jonyd/gobsidian/internal/index/persist_codec.go:135:	str				75.0%
github.com/jonyd/gobsidian/internal/index/persist_codec.go:143:	boolean				100.0%
github.com/jonyd/gobsidian/internal/index/persist_codec.go:156:	timeBlob			60.0%
github.com/jonyd/gobsidian/internal/index/persist_codec.go:175:	strSlice			100.0%
github.com/jonyd/gobsidian/internal/index/persist_codec.go:186:	headings			100.0%
github.com/jonyd/gobsidian/internal/index/persist_codec.go:202:	blocks				100.0%
github.com/jonyd/gobsidian/internal/index/persist_codec.go:228:	links				100.0%
github.com/jonyd/gobsidian/internal/index/persist_codec.go:246:	inline				100.0%
github.com/jonyd/gobsidian/internal/index/persist_codec.go:266:	value				92.1%
github.com/jonyd/gobsidian/internal/index/persist_codec.go:329:	note				100.0%
github.com/jonyd/gobsidian/internal/index/persist_codec.go:348:	asset				100.0%
github.com/jonyd/gobsidian/internal/index/persist_codec.go:361:	escreveIndexCache		85.7%
github.com/jonyd/gobsidian/internal/index/persist_codec.go:410:	falha				100.0%
github.com/jonyd/gobsidian/internal/index/persist_codec.go:416:	uvarint				100.0%
github.com/jonyd/gobsidian/internal/index/persist_codec.go:433:	uvarintLivre			100.0%
github.com/jonyd/gobsidian/internal/index/persist_codec.go:446:	varint				100.0%
github.com/jonyd/gobsidian/internal/index/persist_codec.go:459:	fixed64				62.5%
github.com/jonyd/gobsidian/internal/index/persist_codec.go:472:	str				100.0%
github.com/jonyd/gobsidian/internal/index/persist_codec.go:487:	boolean				100.0%
github.com/jonyd/gobsidian/internal/index/persist_codec.go:491:	timeBlob			84.6%
github.com/jonyd/gobsidian/internal/index/persist_codec.go:510:	strSlice			83.3%
github.com/jonyd/gobsidian/internal/index/persist_codec.go:529:	headings			83.3%
github.com/jonyd/gobsidian/internal/index/persist_codec.go:555:	blocks				83.3%
github.com/jonyd/gobsidian/internal/index/persist_codec.go:578:	links				83.3%
github.com/jonyd/gobsidian/internal/index/persist_codec.go:610:	inline				84.6%
github.com/jonyd/gobsidian/internal/index/persist_codec.go:630:	value				91.7%
github.com/jonyd/gobsidian/internal/index/persist_codec.go:690:	note				95.5%
github.com/jonyd/gobsidian/internal/index/persist_codec.go:740:	asset				83.3%
github.com/jonyd/gobsidian/internal/index/persist_codec.go:759:	leIndexCache			80.0%
```

Pacote inteiro: 83,5% -> 89,7%.

## git show --stat HEAD

```
commit b2561e0760cc037cffffc5e25494879c4f9c52fc
Author: jonyduque <jonyduque@hotmail.com>
Date:   Sun Sep 6 15:07:52 2026 -0300

    test(index): white-box tests for the metadata cache codec
    ...

 docs/ESTADO.md                       |  49 +++++++
 docs/SUGESTOES.md                    |  16 +++
 internal/index/persist_codec_test.go | 239 +++++++++++++++++++++++++++++++++++
 3 files changed, 304 insertions(+)
```

Só teste e doc — nenhum arquivo de produto no commit.

## verify.ps1

```
pwsh -File scripts/verify.ps1 -SkipCross -SkipNet
...
[...] 11. check_readme_anchors
[OK] check_readme_anchors

[OK] Bateria completa. Pode commitar.
```

Todas as 11 etapas executadas (2 puladas por flag: vet cruzado e check_net) deram
`[OK]`; a etapa 3 (contagem de testes pulados) reportou 6 pulados pré-existentes,
sem relação com esta tarefa (informa, não reprova).

## Concerns

- `TestCodecTagDesconhecidaERecusada`, como escrito literalmente no brief da
  Task 181, esperava a mensagem "tag de valor desconhecida" para uma tag
  corrompida `valMap+1`. Isso não é o que o código faz: `leitor.value`
  (`persist_codec.go:638`) lê a tag via
  `l.uvarint(uint64(valMap), "tipo de valor")` — o MESMO `uvarint` que aplica
  `valMap` como limite superior. Uma tag `valMap+1` é recusada dentro desse
  `uvarint`, com a mensagem "acima do limite", ANTES de o `switch` (:642-688)
  rodar. O branch `default` (:684, "tag de valor desconhecida") é código morto
  para qualquer tag corrompida — o `switch` já cobre as doze tags válidas
  inteiras (`valNil`..`valMap`), então não existe tag alcançável que caia no
  `default`.
  O comportamento de segurança está correto (a tag inválida ainda é recusada,
  com `ErrIndexCacheCorrupted`); o que está morto é a mensagem específica e o
  fato de o teto de leitura da tag fazer dupla função (delimitar valores
  válidos E servir de guarda contra tag desconhecida) sem isso estar
  documentado no código. Mantive o teste, mudei a asserção para o texto real
  ("acima do limite") e deixei um comentário no teste explicando o mecanismo.
  Registrado como **B21** em `docs/SUGESTOES.md` e citado no bloco de
  cobertura de `docs/ESTADO.md`. Não mexi em `persist_codec.go` — está fora
  do escopo desta tarefa (`test:`), e a correção (se algum dia for feita: por
  exemplo, remover o `default` morto ou trocar o teto do `uvarint` para
  permitir passar por ele) é decisão de outra tarefa.

## audit_reports

Rodado com este relatório ainda incompleto (antes de escrever as seções finais);
os 4 achados sobre este arquivo (`SECAO-AUSENTE` x3, `CURTO`) referem-se a essa
versão intermediária e foram corrigidos ao escrever esta versão completa. Os 14
achados sobre `.superpowers/sdd/2026-07-25-gobsidian-v01/progress.md` são
pré-existentes, do ledger antigo, sem relação com a Task 181 — confirmado contra
o aviso do despacho ("14 achados pré-existentes do ledger antigo são conhecidos;
só achados novos importam").

## Fix round 1

Revisão em `.superpowers/sdd/2026-09-02-topografia-e-limpeza/review-181.md`:
APPROVED_WITH_NITS, dois nits (N1, N2).

### Progresso

- 15:19 lida a revisão; N1 (par valSliceNil/valSlice a 0% de cobertura em
  `TestCodecValorRoundTripPorTipo`) e N2 (truncação sem `errors.Is`) confirmados
  contra `persist_codec.go:294-299` e `:659-660`
- 15:20 `persist_codec_test.go` editado: `[]any(nil)` acrescentado a `casos`
  (com comentário citando N1) e `errors.Is(l.err, ErrIndexCacheCorrupted)`
  acrescentado ao loop de truncação (N2)
- 15:20 `go vet ./internal/index/...` limpo; `go test -race -run TestCodec -v`
  — 8/8 PASS
- 15:20 mutação (N1): branch de escrita de `valSliceNil` forçado a emitir
  `valSlice` de tamanho 0 — exit 0, teste FAIL sob mutação confirmado
- 15:20 `go test ./internal/index/` (suite inteira) — PASS
- 15:20 cobertura remedida
- 15:21 `docs/ESTADO.md`: linhas `escritor.value`, `leitor.value` e o total do
  pacote atualizadas com os números remedidos; UTF-8 validado
- 15:21–15:26 `pwsh -File scripts/verify.ps1 -SkipCross -SkipNet` — verde
- 15:26 commit criado
- 15:2x `pwsh -File scripts/audit_reports.ps1 181` rodado de novo

### N1 — teste

```
$ go test -race ./internal/index/ -run TestCodecValorRoundTripPorTipo -v
=== RUN   TestCodecValorRoundTripPorTipo
--- PASS: TestCodecValorRoundTripPorTipo (0.02s)
PASS
ok  	github.com/jonyd/gobsidian/internal/index	1.636s
```

### N1 — mutação

```
$ pwsh -File scripts/mutate.ps1 -Path internal/index/persist_codec.go -Anchor 'e.uvarint(uint64(valSliceNil))' -Replacement 'e.uvarint(uint64(valSlice)); e.uvarint(0)' -Test TestCodecValorRoundTripPorTipo -Package ./internal/index/
Carregado em 412ms
[...] Mutando internal/index/persist_codec.go
      - e.uvarint(uint64(valSliceNil))
      + e.uvarint(uint64(valSlice)); e.uvarint(0)

[...] go test -race -run TestCodecValorRoundTripPorTipo ./internal/index/
----------------------------------------------------------------------
--- FAIL: TestCodecValorRoundTripPorTipo (0.03s)
    persist_codec_test.go:73: []interface {}(nil) voltou []interface {}{}
FAIL
FAIL	github.com/jonyd/gobsidian/internal/index	0.667s
FAIL
----------------------------------------------------------------------
[OK] internal/index/persist_codec.go restaurado byte a byte (SHA-256 confere).

[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
EXIT=0
```

Não achado B22: o par `valSliceNil`/`valSlice` round-tripa corretamente — o
`[]any(nil)` volta `[]any(nil)`, distinto de `[]any{}`, exatamente como o
código promete. Nenhuma entrada nova em `docs/SUGESTOES.md` foi necessária.

### N2 — teste

```
$ go test -race ./internal/index/ -run TestCodecNotaTruncadaEmCadaByteERecusada -v
=== RUN   TestCodecNotaTruncadaEmCadaByteERecusada
--- PASS: TestCodecNotaTruncadaEmCadaByteERecusada (0.00s)
PASS
ok  	github.com/jonyd/gobsidian/internal/index	1.6s
```

### Cobertura depois (rodada de correção)

```
go test ./internal/index/ -coverprofile=$TEMP/cov_depois2.out
ok  	github.com/jonyd/gobsidian/internal/index	1.653s	coverage: 89.8% of statements

go tool cover -func=$TEMP/cov_depois2.out | grep persist_codec.go
... (linhas iguais às do Step 4 original, exceto:)
github.com/jonyd/gobsidian/internal/index/persist_codec.go:266:	value				97.4%
github.com/jonyd/gobsidian/internal/index/persist_codec.go:630:	value				94.4%
```

`escritor.value`: 92,1% → 97,4%. `leitor.value`: 91,7% → 94,4%. Pacote inteiro:
89,7% → 89,8%.

### git show --stat HEAD

```
commit 7d4d684f64bea17ca5ae2a25764c20f6e524060d
    test(index): the codec round-trip covers the typed-nil slice, truncation asserts the sentinel

 docs/ESTADO.md                       | 12 +++++++-----
 internal/index/persist_codec_test.go | 11 +++++++++++
 2 files changed, 18 insertions(+), 5 deletions(-)
```

Só teste e doc — nenhum arquivo de produto.

### verify.ps1

```
pwsh -File scripts/verify.ps1 -SkipCross -SkipNet
...
[OK] Bateria completa. Pode commitar.
```

Todas as 11 etapas `[OK]` (2 puladas por flag), mesmos 6 pulados pré-existentes
na etapa 3.

### Concerns (rodada de correção)

Nenhum achado novo. O `[]any(nil)` round-tripa corretamente — não há B22 a
registrar.

### audit_reports (rodada de correção)

Pendente de colar após rodar novamente com este relatório completo — ver seção
seguinte se atualizada.
