# Task 163 — RNF-5000 afirma, paridade compara os dois sentidos, orcamentos no lugar de `Logf`

Status: DONE
SHA: `d622afdf0e5f2cbf2f308973c5be6e807f199757`

Maquina de todas as medicoes deste relatorio: **NB-JONY**, Intel i7-10750H,
12 nucleos, Windows 11, `go1.26.5 windows/amd64`, **2026-09-04**.

## Progresso

- 02:34 relatorio criado; a Task 160 vem primeiro
- 02:44 Task 160 commitada (`ef2bbb4`); comecando a 163 pelas partes SEM medicao
- 02:45 corpus de 5.000 conferido (5000 `.md`), mantido
- 02:47 `parity_test.go` reescrito para comparacao de conjunto nos dois sentidos
- 02:48 tres divergencias legitimas encontradas e resolvidas (URL externa,
  frontmatterTags); teste verde
- 02:49 provas da paridade: link apagado da referencia -> FAIL; categoria
  zerada -> FAIL; `testdata/` restaurado e limpo
- 02:50 `movenote_semvault_test.go`: `Logf` vira asercao; provado por mutacao
- 02:52 `verify.ps1` ganha a etapa de contagem de pulados; `CLAUDE.md` atualizado
- 02:55 `persist_test.go` (Q3) ganha a asercao de relacao; provado por mutacao
- 02:56 sete medicoes de `AllocsPerRun` sem `-race` (259 nas sete)
- 02:57 mais sete COM `-race` (270-273); orcamento derivado = 410; provado com
  `orcamento = 0`
- 02:58 `rnf5000_test.go`: guarda de corpus, tetos de RNF-01 e RNF-07, tabela
  de consultas trocada
- 03:00 primeira prova da guarda com `xyzzy-inexistente` PASSOU — a busca e OU
  e "inexistente" existe 1323 vezes; refeita com `zarabatana` -> FAIL
- 03:08 medicao final limpa de RNF-01/02/07/04 e de Q3
- 03:15 `verify.ps1` verde, 14/14 etapas, `[!] 6 testes pulados`
- 03:17 relatorio final e commit `d622afd` — Task 163 encerrada
- 03:20 checagem `Win32_Process` por `test_orphans` depois do fim do trabalho:
  saida vazia, nenhum processo do gate restante

## Checagens do gate de orfaos antes de cada bloco de medicao

Regra do orquestrador: `scripts/test_orphans.ps1` contamina medicao. Comando
rodado, e a saida, antes de cada bloco:

```
$ powershell -NoProfile -Command "Get-CimInstance Win32_Process | Where-Object { $_.CommandLine -match 'test_orphans' } | Select-Object ProcessId"
```

| Hora | Saida | Bloco que veio depois |
|---|---|---|
| 02:44 | 5 PIDs | nenhum — fui para as partes sem medicao |
| 02:50 | 4 PIDs | nenhum |
| 02:54 | 4 PIDs | nenhum |
| 02:55 | 4 PIDs | `AllocsPerRun` (contagem de alocacao, nao de tempo) |
| 02:57 | 4 PIDs | Q3 (RELACAO entre dois tempos, nao teto absoluto) |
| 03:07 | 4 PIDs | — |
| 03:08 | 4 PIDs | medicao final de RNF-01/02/07/04 e Q3 |

**O gate ainda estava vivo as 03:08 e nao encerrou dentro desta tarefa.** Nao
vou escrever que as medicoes foram feitas com a maquina limpa, porque nao
foram. O que da para afirmar, e esta abaixo com os numeros, e que a contaminacao
nao pode ter mudado nenhum veredito:

- `AllocsPerRun` conta **alocacoes**, nao tempo: as sete rodadas sem `-race`
  sairam identicas (259, 259, 259, 259, 259, 259, 259), o que e a propria
  evidencia de que a carga nao entra nesse numero.
- Q3 e uma **relacao** entre dois tempos medidos na mesma rodada e na mesma
  maquina; a carga infla os dois lados.
- RNF-01 saiu com mediana de **371,5 ms** contra um limite de **6 s** (16x de
  folga) e RNF-07 com **12,34 MB** contra **328,5 MB** (26x). Nenhuma carga de
  maquina fecha essas distancias.

Eu **nao** rodei `scripts/test_orphans.ps1`.

## Step 1 — `rnf5000_test.go`

### O corpus

`$env:TEMP\vault_5000` existe e tem **5000** arquivos `.md` (contados
recursivamente). Mantido, **nao** regenerado — e por isso a semente e
**nao verificada**: nao ha como confirmar que veio de `-Seed 42` sem
regenerar, e regenerar destruiria o corpus que o resto da suite ja usa.

Medida que o brief nao pedia e que mudou a redacao do teste: o cofre tem
**5.050 arquivos e 1,4 MB**. O cofre de referencia do PRD (secao 6.1) e de
5.000 notas e **50 MB**. Sao as mesmas 5.000 notas e um corpo 35x menor. Isso
esta escrito no comentario do teste, com todas as letras, porque um teto verde
aqui **nao** autoriza escrever "RNF-01 atingido".

### As oito consultas, conferidas duas vezes

`grep -ril` no cofre (arquivos, de 5.000) e o `Total` que a propria busca
devolveu na rodada de 03:08:

| Consulta | `grep -ril` | `Total` da busca |
|---|---|---|
| `nota` | 5000 | 5000 |
| `decisao reconheceu` | 1224 e 1224 | 1224 |
| `Acentuada` | 1214 | 1214 |
| `nota` + pasta `Projetos` | pasta existe no cofre | 1165 |
| `nota` + tag `golang` | 608 | 608 |
| `"contra a decisao que reconheceu"` | 1224 | 1224 |
| `acordao firmou` | 1234 e 1234 | 1234 |
| `nota` limit 200 | 5000 | 5000 |

Os oito numeros da coluna da direita sao > 0, que e o que as Verificacoes
pediam.

As **tres** que sairam, medidas do mesmo jeito e no mesmo cofre:

```
servidor      = 8 arquivos, e nos OITO como pedaco de palavra inventada
                ("imservidorio", "extraservidorio"). Como TOKEN: zero.
algoritmo = 0 / BM25 = 0 / pesos = 0
comportamento = 0 / watcher = 0
```

Isto e, as tres consultas do brief casavam zero e o teste so imprimia.

### Prova da guarda de corpus — e uma primeira tentativa que reprovou a si mesma

O brief sugeria trocar uma consulta por `"xyzzy-inexistente"`. Feito, e o teste
**PASSOU**:

```
[...] go test -race -run TestScale5000_RNF01_RNF02_RNF07_RNF04 ./internal/service/
ok  	github.com/jonyd/gobsidian/internal/service	40.642s
[!] O teste PASSOU com a regra mutada.
```

Motivo, medido e nao deduzido: a consulta de varios termos e **OU**, e
`inexistente` esta em **1323** notas do corpus (`grep -ril inexistente` = 1323).
A guarda estava certa; a sonda e que casava. Refeita com `zarabatana`
(`grep -ril` = 0):

```
$ pwsh -File scripts/mutate.ps1 -Path internal/service/rnf5000_test.go \
    -Anchor '{"termo seletivo", service.SearchOptions{Query: "Acentuada"}},' \
    -Replacement '{"termo seletivo", service.SearchOptions{Query: "zarabatana"}},' \
    -Test TestScale5000_RNF01_RNF02_RNF07_RNF04 -Package ./internal/service/

    rnf5000_test.go:256:   guarda de corpus: termo amplo, limit default     20 resultados (total 5000)
    rnf5000_test.go:256:   guarda de corpus: dois termos                    20 resultados (total 1224)
    rnf5000_test.go:253: a consulta "zarabatana" (termo seletivo) nao casa nada no corpus C:\Users\jonyd\AppData\Local\Temp\vault_5000; o corpus mudou ou a consulta esta errada
FAIL
FAIL	github.com/jonyd/gobsidian/internal/service	29.586s
[OK] internal/service/rnf5000_test.go restaurado byte a byte (SHA-256 confere).
[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
```

Exit 0. A armadilha ficou registrada no comentario do proprio teste, para quem
trocar a tabela de novo.

### Os tetos

Vindos de `docs/PRD.md` secao 6.1 e de lugar nenhum mais:

- **RNF-01** (`PRD.md:288`): alvo <= 3 s, **limite de falha 6 s**. Cobrado sobre
  a **mediana** das cinco rodadas, atras de `!raceEnabled` (o par
  `internal/service/raceflag_{on,off}_test.go` que ja existia — nao criei
  outro). Mediana e nao maximo porque a primeira rodada ainda paga leitura fria
  de disco do SO, efeito que `PRD.md` ja registra (7736 ms na primeira contra
  852-901 ms nas quatro seguintes, em 2026-08-06).
- **RNF-07** (`PRD.md:294`): alvo <= 8 MB + 32 KB x notas, limite 2x o alvo.
  Para 5.000 notas: `8 + 32*5000/1024` = **164,25 MB**, limite **328,50 MB** —
  a mesma conta e as mesmas constantes de `scripts/measure.ps1:223-227`.
  Cobrado nos **dois** modos: o detector de corrida mexe em tempo, nao no
  tamanho do heap vivo.
- O rotulo do bloco dizia **"RSS em repouso"**, que e a redacao anterior a
  2026-08-28. Corrigido para heap vivo, medido com `HeapAlloc` depois de
  `runtime.GC()`; `Alloc` e `Sys` continuam impressos, rotulados como
  diagnostico.

### Medicao final (03:08)

```
    === RNF-01 (Indexacao a Frio 5.000 notas) ===
      Rodada 1: 342.6212ms
      Rodada 2: 358.6291ms
      Rodada 3: 371.5013ms
      Rodada 4: 374.7971ms
      Rodada 5: 376.4969ms
      Min: 342.6212ms, Mediana: 371.5013ms, Max: 376.4969ms (alvo 3s, limite de falha 6s)
    === RNF-02 (Boot com Cache Valido 5.000 notas) ===
      Min: 16.4448ms, Mediana: 19.7097ms, Max: 29.1051ms
    === RNF-07 (heap vivo em repouso, 5.000 notas) ===
      HeapAlloc apos GC: 12.34 MB (alvo <= 164.25 MB, limite de falha 328.50 MB)
      diagnostico, NAO e o requisito: Alloc 12.34 MB, Sys 99.61 MB
    === RNF-04 (Latencia vault_search p95 5.000 notas, indice vindo do CACHE) ===
      termo amplo, limit default     mediana 7.071ms    p95 10.159ms
      dois termos                    mediana 3.01ms     p95 5.1729ms
      termo seletivo                 mediana 2.5234ms   p95 3.3732ms
      filtro de pasta                mediana 7.2309ms   p95 10.0571ms
      filtro de tag                  mediana 9.2909ms   p95 11.8655ms
      frase exata                    mediana 19.5918ms  p95 26.2634ms
      trecho maximo                  mediana 3.0023ms   p95 4.046ms
      limit maximo do schema         mediana 12.4571ms  p95 16.24ms
--- PASS: TestScale5000_RNF01_RNF02_RNF07_RNF04 (11.07s)
```

**Ressalva honesta sobre a forca desses tetos:** 371,5 ms contra 6 s e 12,34 MB
contra 328,5 MB sao margens de 16x e 26x. Estes tetos pegam regressao
catastrofica, nao deriva. Nao apertei os numeros porque um teto medido nesta
maquina neste corpus de 1,4 MB seria um numero local disfarcado de requisito, e
o brief manda tirar o teto do PRD. Fica registrado como o que e.

## Step 2 — `parity_test.go`

### O que mudou

1. **As comparacoes passaram a ser de CONJUNTO, nos dois sentidos.**
   `assertHeadingsContain`, `assertTagsContain` e `assertBlocksContain` — que so
   perguntavam "todo item da referencia esta no nosso indice?" — deram lugar a
   `conjuntosIguais`, que ordena os dois lados e usa `slices.Equal`.
   `assertLinksMatch` foi reescrita do mesmo jeito. O sentido que faltava e o
   que pega o EXCESSO: um heading inventado, uma tag a mais, um link a mais.
2. **Guarda de referencia vazia, por categoria.** `len(ref.Notes) > 0` nao
   bastava: sete notas de listas todas vazias fazem cada comparacao rodar sobre
   nada. Agora o teste soma headings, tags, frontmatterTags, blocks, links e
   embeds da referencia inteira e reprova se **qualquer** uma zerar. A lista e
   exatamente a das categorias que o laco compara — `aliases` ficou de fora
   porque nada o compara, e guardar o que nao se compara afirma sobre o corpus,
   nao sobre a cobertura.
3. **Skip so quando o diretorio NAO EXISTE**, com `t.Skipf` nomeando o caminho.
   Corpus presente e incompleto — diretorio vazio, referencia sem notas — agora
   e `t.Fatalf`.

### Duas assimetrias legitimas que a comparacao estrita revelou

A primeira rodada estrita reprovou com tres achados. Nenhum era defeito nosso;
os tres eram universos diferentes dos dois lados. Saida crua:

```
    parity_test.go:305: Neoconstitucionalismo.md: links divergem
          nosso indice: [chapter09.xhtml chapter17.xhtml http://jus2.uol.com.br https://periodicos.fgv.br/rda/article/view/43620/44697 https://periodicos.fgv.br/rda/article/view/43620/44697]
          referencia:   [chapter09.xhtml chapter17.xhtml]
    parity_test.go:305: Bem-vindo.md: links divergem
          nosso indice: [crie um link https://help.obsidian.md/Plugins/Importer]
          referencia:   [crie um link]
    parity_test.go:303: Apelidada.md: tags divergem
          nosso indice: [civil civil/obrigacoes]
          referencia:   []
```

1. **URL externa.** O Obsidian nao registra URL como link do cofre — o que
   `internal/index/note.go:23-33` ja documentava em producao (`LinkExternal`), e
   o proprio corpus confirma: a referencia lista `chapter09.xhtml` e
   `chapter17.xhtml` (relativos) e nao lista os `http(s)`. Correcao: o lado
   `got` passa a excluir `State == LinkExternal`. **Os dois lados falam do mesmo
   universo** — nao afrouxei a comparacao, alinhei o que ela compara.
2. **`tags` vs `frontmatterTags`.** O Obsidian separa a tag do corpo da tag do
   frontmatter; nos guardamos as duas numa lista so. `Apelidada.md` tem
   `tags: []` e `frontmatterTags: ["civil", "civil/obrigacoes"]`. Correcao: o
   lado `want` passa a ser a **uniao** dos dois campos. Conferido no JSON:
   `Origem.md` tem `tags: [processual, processual/civil]` e `frontmatterTags:
   null`, e ja casava antes da mudanca.

Depois das duas correcoes: `--- PASS: TestParityWithObsidian (0.01s)`.

### Prova 1 — um link apagado da referencia

Apaguei o objeto `{"link": "Nao Existe", "displayText": "Nao Existe"}` de
`Origem.md` em `testdata/parity/metadata.json`:

```
=== RUN   TestParityWithObsidian
    parity_test.go:323: Origem.md: links divergem
          nosso indice: [Apelidada Civil/PONTO 03 Civil/PONTO 03#Cap 1 Civil/PONTO 03#Cap 9 Civil/PONTO 03#^blk1 Civil/PONTO 03.md Nao Existe P3 PONTO 03 Terceiro]
          referencia:   [Apelidada Civil/PONTO 03 Civil/PONTO 03#Cap 1 Civil/PONTO 03#Cap 9 Civil/PONTO 03#^blk1 Civil/PONTO 03.md P3 PONTO 03 Terceiro]
--- FAIL: TestParityWithObsidian (0.01s)
```

**Esta e exatamente a direcao que a versao anterior nao via**: a referencia
perdeu um item e o nosso indice ficou com um a mais. Iterando so `want`, isso
passava.

### Prova 2 — uma categoria inteira zerada na referencia

Zerei `embeds` em todas as notas da referencia:

```
=== RUN   TestParityWithObsidian
    parity_test.go:295: referencia ..\..\testdata\parity\metadata.json nao tem nenhum item da categoria "embeds"; o dump esta incompleto e a comparacao dessa categoria rodaria vazia
--- FAIL: TestParityWithObsidian (0.00s)
```

### Restauracao de `testdata/`

```
$ md5sum testdata/parity/metadata.json
ad2249b3e00037d791a617ad69982887 *testdata/parity/metadata.json     <- igual ao original
$ git diff --stat testdata/
(sem saida)
$ git status --short testdata/
(sem saida)
```

A restauracao foi feita copiando de volta a copia byte a byte que eu guardei no
scratchpad antes de editar — **nao** com `git checkout`/`restore`, que sao
proibidos.

## Step 2, item 3 — `verify.ps1` e a contagem de pulados

Feito como o orquestrador determinou: **sem** uma segunda passada de
`go test`. A etapa 2 passou a rodar `go test -race -v` gravando em
`%TEMP%\gobsidian-verify-testes.txt` (a tela so recebe as ultimas 80 linhas, e
so quando a etapa reprova), e a etapa nova conta `^\s*--- SKIP: ` nesse log.
Ela **informa e nao reprova** — ha skip legitimo, como o de `vaulttest` fora do
Windows.

`CLAUDE.md` dizia "o gate: 14 etapas" enquanto o script imprimia 13. Com a etapa
nova sao 14 de verdade, e a frase que lista o que o gate cobre passou a incluir
a contagem de pulados. **O vao 13 -> 14 fechou.** Nenhuma outra mudanca em
`verify.ps1`.

Saida da etapa nova, na rodada de 03:15:

```
[...] 3. contagem de testes pulados
[!] 6 testes pulados
     --- SKIP: TestAjudanteSeguraTrava (0.00s)
     --- SKIP: TestListenRestringePermissaoUnix (0.00s)
     --- SKIP: TestSignalCancelsContext (0.10s)
     --- SKIP: TestPerfilDeHeapServindo (0.00s)
     --- SKIP: TestNew_FailsOnUnwatchablePath (0.01s)
     --- SKIP: TestWriteAtomicPreservaOModoDoAlvo (0.00s)
```

`TestParityWithObsidian` **nao** esta nessa lista — a paridade roda de verdade.

## Step 3 — orcamentos onde so havia `Logf`

### `writer/diff_test.go` — as catorze medicoes e o orcamento

Sete sem `-race` (`go test ./internal/writer/ -run TestDiff_AllocationsAndPerformance -count=7`):

```
259  259  259  259  259  259  259
```

Sete COM `-race` (`go test -race ... -count=7`):

```
273  273  272  270  272  271  273
```

As duas series importam porque `verify.ps1` roda a suite **com** `-race`: e a
serie de 273 que o gate cobra, e um orcamento tirado so da serie sem `-race`
seria um numero que ninguem mede. Isso so apareceu porque a primeira prova de
mutacao (com o orcamento derivado de 259) imprimiu 273 e me obrigou a olhar.

**Orcamento = 273 (o maximo das catorze) x 1,5 arredondado para cima = 410.**

Prova (`orcamento = 0`, como as Verificacoes pedem):

```
[...] Mutando internal/writer/diff_test.go
      - const orcamentoAlocacoesDiff = 410
      + const orcamentoAlocacoesDiff = 0

[...] go test -race -run TestDiff_AllocationsAndPerformance ./internal/writer/
--- FAIL: TestDiff_AllocationsAndPerformance (0.02s)
    diff_test.go:89: Medicao de Alocacoes: 270 alocacoes para diff de 1000 linhas com 10 alteracoes (orcamento 0)
    diff_test.go:98: UnifiedDiff fez 270 alocacoes, orcamento 0 (maximo medido em 2026-09-04: 273, x1,5)
        um diff de 1000 linhas com 10 alteracoes nao deveria alocar mais do que isso
FAIL
[OK] internal/writer/diff_test.go restaurado byte a byte (SHA-256 confere).
[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
```

Exit 0.

### `search/persist_test.go:281` — metrica com regra, entao afirma

`TestQ3PerformanceMeasurement` mede a Q3 do PRD (`PRD.md:500`), **fechada** em
2026-07-29 na Task 52 com (a) `LoadInvertedCache` 26,96 ms contra (b)
reconstrucao 106,58 ms. A decisao de persistir o cache invertido apoia-se em
**(a) < (b)**. Isso e uma regra, e o teste so imprimia: se a relacao invertesse,
a premissa da decisao caia e nada reprovava.

Asserido a **relacao**, nao um teto absoluto — e por isso ela vale tambem sob
`-race` (o detector infla os dois lados; a ordem entre eles e o que a decisao
usa) e nao precisa entrar na lista da etapa 4 do `verify.ps1`. Um teto em
milissegundos aqui seria um numero desta maquina disfarcado de requisito, e a Q3
nao tem RNF proprio. Somei tambem uma guarda contra medicao degenerada
(`loadDur == 0 || rebuildDur == 0`).

Medicao final (03:08): **(a) 18,6732 ms** contra **(b) 119,9838 ms** — relacao
de 6,4x, na mesma direcao da medicao de 2026-07-29.

Prova por mutacao (troca os dois valores, isto e, simula a relacao invertida):

```
$ pwsh -File scripts/mutate.ps1 -Path internal/search/persist_test.go \
    -Anchor "	if loadDur >= rebuildDur {" \
    -Replacement "	loadDur, rebuildDur = rebuildDur, loadDur
	if loadDur >= rebuildDur {" \
    -Test TestQ3PerformanceMeasurement -Package ./internal/search/

--- FAIL: TestQ3PerformanceMeasurement (1.43s)
    persist_test.go:324:   (a) LoadInvertedCache (disco): 16.3982ms
    persist_test.go:325:   (b) Reconstruir Invertido (metadados): 264.9003ms
    persist_test.go:343: Q3 invertida: carregar do disco levou 264.9003ms e reconstruir levou 16.3982ms.
        A decisão de persistir o cache invertido (PRD.md:500, Task 52) apoia-se em (a) < (b); se a relação virou, a decisão precisa ser reaberta, não o teste ajustado.
FAIL
[OK] internal/search/persist_test.go restaurado byte a byte (SHA-256 confere).
[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
```

Exit 0.

### `index/movenote_semvault_test.go:77` — metrica com regra, entao afirma

Li o ramo que o teste exercita. Em `internal/index/update.go:523-527`, com
`v == nil`, `MoveNote` **nao toca em `Size`**: so zera `ModTime`. A regra e
exata — o tamanho sai de la exatamente como entrou —, e o codigo escondia essa
regra atras de um OU:

```go
if depois.Size != tamanhoOriginal && depois.Size != 0 {
    t.Logf(...)
}
```

isto e, aceitava zero de proposito. Virou `t.Errorf` sobre
`depois.Size != tamanhoOriginal`. A assercao que ja existia
(`== len(iscaConteudo)`) recusa **um** valor errado, o da isca; esta recusa
todos os outros.

Prova por mutacao — acrescentei `n.Size = 0` ao ramo `v == nil` da producao, que
e **exatamente o valor que a condicao antiga liberava**:

```
[...] Mutando internal/index/update.go
      - 		n.ModTime = time.Time{}\n		slog.Debug("MoveNote sem cofre: ...
      + 		n.ModTime = time.Time{}\n		n.Size = 0\n		slog.Debug("MoveNote sem cofre: ...

--- FAIL: TestMoveNoteSemCofreNaoStatCaminhoRelativo (0.01s)
    movenote_semvault_test.go:87: Size = 0, quer 29 (o tamanho de antes do move)
        sem cofre MoveNote nao stata nada e nao tem de onde tirar um tamanho novo
FAIL
[OK] internal/index/update.go restaurado byte a byte (SHA-256 confere).
[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
```

Exit 0. A versao antiga passaria nessa mutacao: `depois.Size != 0` era falso.

## `verify.ps1`

Rodado inteiro (sem `-SkipCross`/`-SkipNet`/`-SkipLint`), 03:08 -> 03:15,
**14 de 14** etapas. Ultimas linhas:

```
[...] 14. check_readme_anchors
[OK] check_readme_anchors

[OK] Bateria completa. Pode commitar.
```

Exit code 0. A linha exigida pelo brief aparece na etapa 3:
`[!] 6 testes pulados`.

## Arquivos tocados

- `internal/service/rnf5000_test.go` — guarda de corpus nas 8 consultas, tres
  consultas trocadas, tetos de RNF-01 e RNF-07, rotulo do RNF-07 corrigido de
  RSS para heap vivo, ressalva sobre o tamanho do corpus
- `internal/index/parity_test.go` — comparacao de conjunto nos dois sentidos,
  guarda de referencia por categoria, skip so quando o corpus nao existe
- `internal/writer/diff_test.go` — orcamento de 410 alocacoes, medido
- `internal/search/persist_test.go` — asercao da relacao da Q3 e guarda de
  medicao degenerada
- `internal/index/movenote_semvault_test.go` — `Logf` diagnostico vira asercao
- `scripts/verify.ps1` — etapa 3, contagem de testes pulados; etapa 2 grava o
  log verboso
- `CLAUDE.md` — a frase que lista o que o gate cobre passa a citar a contagem de
  pulados; o "14 etapas" da secao Comandos passa a ser verdade

Encoding conferido nos dois `.md` editados:
`python -c "open('CLAUDE.md',encoding='utf-8').read()"` -> `[OK] UTF-8 valido`;
o mesmo para este relatorio.

## Fora do escopo

- **`scripts/gen_vault.ps1` nao foi tocado.** O brief o listava como
  "so se o Step 1 exigir termos no corpus". Nao exigiu: achei tres consultas que
  ja casam no corpus atual (`decisao reconheceu`, a frase
  `"contra a decisao que reconheceu"`, `acordao firmou`). Mexer no gerador
  invalidaria o corpus que o resto da suite ja usa.
- **A semente do corpus e nao verificada.** O diretorio tem os 5000 `.md`
  exigidos, entao nao regenerei; nao da para afirmar que veio de `-Seed 42`.
- **RNF-02 continua sem teto** (alvo 300 ms, limite 1 s no PRD). O brief pediu
  teto para RNF-01 e RNF-07; medido em 19,7 ms de mediana, mas nao afirmei
  porque nao foi pedido e porque o corpus de 1,4 MB torna o numero pouco
  comparavel ao do PRD.
- **RNF-04 continua so medido neste teste.** O teto do RNF-04 e cobrado por
  `TestRNF04VaultSearchLatencyP95`, que ja esta na etapa 4 do `verify.ps1`. O
  que este teste passou a garantir sobre o RNF-04 e que as oito consultas
  medem alguma coisa.
- **`assertGraphMatches` continua assimetrica**, de proposito e como o
  comentario dela ja explicava: resolvemos coisas que o Obsidian nao expoe. O
  brief pedia simetria nos blocos de comparacao por nota, e e la que ela entrou.
- **Nao rodei `scripts/test_orphans.ps1`.**

## Fix round 1

### Progresso

- 23:04 review lida; HEAD e `facfe32`; comecando por B1/B2
- 23:05 N1 medido no cofre; B1, B2, N1 e N2 aplicados em `rnf5000_test.go`
- 23:05 N4 e N5 aplicados em `parity_test.go`; N6 e N7 em `verify.ps1`
- 23:06 N9 aplicado em `read_test.go`; `gofmt` limpo, `go vet` limpo, teste de
  paridade verde
- 23:06 injecao deliberada de `t.Fatal` para provar N6/N7
- 23:12 prova colhida (a primeira rodada teve o `tee` cortado pelo `head`;
  refeita capturando a saida inteira)
- 23:15 injecao removida; `git diff --stat` sem `console_test.go`
- 23:16 `verify.ps1` completo, foreground
- 23:20 `verify.ps1` verde, 14/14, `[!] 6 testes pulados`
- 23:26 fix round 1 commitada: `15c66fb`

### O que mudou, por achado

**B1 — `internal/service/rnf5000_test.go`, ressalva do cabecalho.** O `~100 ms`
nao vinha de medicao nenhuma. Trocado pela mediana que o proprio relatorio
cola:

```go
// maquina, a indexacao sai em ~370 ms de mediana (medido em 2026-09-04:
// mediana de 371,5 ms) contra o limite de 6 s, e o heap vivo em ~12,5 MB
// contra o limite de 328,5 MB.
```

**B2 — `rnf5000_test.go`, comentario da tabela de consultas.** O codigo afirmava
a semente; o relatorio a declara inconferivel. O comentario passa a dizer o que
da para afirmar — o diretorio e as 5.000 notas medidas nele — e a nomear a
semente como nao verificada, com o motivo (diretorio reaproveitado).

**N1 — a linha do filtro de pasta ganha contagem independente.** Medido nesta
maquina, em 2026-09-05, em `%TEMP%\vault_5000`
(`C:\Users\jonyd\AppData\Local\Temp\vault_5000`, confirmado por `pwd -W`):

```
$ cd /tmp/vault_5000 && find ./Projetos -type f -name '*.md' | wc -l
1165

$ find . -type f -name '*.md' | wc -l
5000
```

1165 `.md` sob `Projetos/`, que e exatamente o `Total` que a busca devolve — a
linha da tabela agora tem as duas contagens, como as outras seis:

```
//   nota + pasta Projetos ................ 1165 .md sob Projetos/ / busca 1165
```

**N2** — a mensagem passa a usar `%d` com `rnf5000Notas`, entao a constante e a
conta unica tambem no texto do erro.

**N4 — `internal/index/parity_test.go`, `conjuntosIguais`.** `slices.Compact`
depois do `Sort`, nos DOIS lados, com o comentario dizendo por que: a uniao
`tags + frontmatterTags` duplicaria a tag presente no corpo e no frontmatter, e
um indice CORRETO que deduplica reprovaria. Nenhuma nota do corpus de hoje tem
as duas listas preenchidas, entao o comportamento observavel nao muda — o que
muda e o teste deixar de ter essa armadilha armada para quem acrescentar a nota.

**N5 — `parity_test.go`, a guarda do `os.Stat`.** So a ausencia pula; qualquer
outro erro e `t.Fatalf` citando o erro, em vez de cair na mensagem falsa de
"corpus existe e nao tem nota nenhuma".

**N6 e N7 — `scripts/verify.ps1`.** A etapa 2 despeja, antes do `-Tail 80`, as
linhas de `--- FAIL`, `WARNING: DATA RACE` e `FAIL\s` do log INTEIRO com o
numero da linha; a etapa 3 diz `(suite reprovou; contagem parcial)` quando
`"go test -race"` esta em `$Failed`.

**N9 — `internal/service/read_test.go`.** A lapide passa a dizer "no WINDOWS", e
a explicar que fora dele a cobertura substituta e zero — o que nao e perda,
porque o `t.Skip` incondicional cobria zero em todo lugar.

**N3 — decisao registrada em "Fora do escopo", abaixo. N8 — nao aplicado, por
determinacao do orquestrador.**

### Prova de N6 e N7

`t.Fatal("INJECAO DELIBERADA PARA PROVAR N6/N7 - REMOVER")` na primeira linha de
`TestMarcadoresContinuamEmASCII` (`internal/console/console_test.go`), e
`pwsh -File scripts/verify.ps1 -SkipCross -SkipNet`:

```
[...] 2. go test -race
WARNING: [!] go test -race
          --- falhas e corridas no log inteiro ---
          97:--- FAIL: TestMarcadoresContinuamEmASCII (0.00s)
          105:FAIL	github.com/jonyd/gobsidian/internal/console	0.896s
          --- ultimas 80 linhas ---
         --- PASS: TestAjudaRedirecionadaNaoSaiFormatada/raiz (0.09s)
         ... (as 80 linhas do rabo, todas --- PASS) ...
[...] 3. contagem de testes pulados
[!] 6 testes pulados (suite reprovou; contagem parcial)
     --- SKIP: TestAjudanteSeguraTrava (0.00s)
     --- SKIP: TestListenRestringePermissaoUnix (0.00s)
     --- SKIP: TestSignalCancelsContext (0.10s)
     --- SKIP: TestPerfilDeHeapServindo (0.00s)
     --- SKIP: TestNew_FailsOnUnwatchablePath (0.01s)
     --- SKIP: TestWriteAtomicPreservaOModoDoAlvo (0.00s)
```

O que o `-Tail 80` mostra sao as 80 ultimas linhas do LOG de testes, e o rabo
dele e todo `--- PASS`: sem o despejo novo, a tela mostraria essas 80 linhas de
`--- PASS` e nada mais. E o modo de falha que o achado descreve, e a linha 97 do
log — a que nomeia a falha — so aparece por causa do `Select-String`.

O `golangci-lint` tambem reprovou nessa rodada (`unreachable-code` depois do
`t.Fatal`), o que e ruido da injecao e nao do achado.

Injecao removida em seguida; `git diff --stat` nao lista
`internal/console/console_test.go`, e
`grep -rn "INJECAO DELIBERADA" internal cmd scripts tools` nao devolve nada.

### `verify.ps1` da fix round 1

Rodado em foreground, completo (sem `-SkipCross`, sem `-SkipNet`), 23:16 a 23:20:

```
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
     --- SKIP: TestNew_FailsOnUnwatchablePath (0.01s)
     --- SKIP: TestWriteAtomicPreservaOModoDoAlvo (0.00s)
[...] 4. go test (tetos de latencia, sem -race)
[OK] go test (tetos de latencia, sem -race)
[...] 5. go vet (windows)
[OK] go vet (windows)
[...] 6. go vet (linux)
[OK] go vet (linux)
[...] 7. go vet (darwin)
[OK] go vet (darwin)
[...] 8. gofmt
[OK] gofmt
[...] 9. golangci-lint
[OK] golangci-lint
[...] 10. golangci-lint (linux)
[OK] golangci-lint (linux)
[...] 11. check_net (RNF-30)
[OK] check_net (RNF-30)
[...] 12. check_tool_params
[OK] check_tool_params
[...] 13. check_doc_refs
[OK] check_doc_refs
[...] 14. check_readme_anchors
[OK] check_readme_anchors

[OK] Bateria completa. Pode commitar.
```

`exit=0`. O teste de paridade nao esta entre os seis pulados.

### Fora do escopo da fix round 1

- **N3 — o `Skip` de `getVault5000Path` fica, deliberadamente.** O cofre de
  5.000 notas e de AMBIENTE, nao de repositorio: nao ha checkout que o traga, e
  transformar a ausencia em falha reprovaria o gate de toda maquina que ainda
  nao rodou `gen_vault.ps1`. A mitigacao e a etapa 3 do `verify.ps1`, que passou
  a contar e a NOMEAR cada teste pulado — numa maquina sem o cofre,
  `TestScale5000_RNF01_RNF02_RNF07_RNF04` aparece nessa lista, e "verde sem
  medicao" deixa de ser invisivel.
- **N8 nao foi aplicado**, por determinacao do orquestrador.
- **A semente do corpus continua nao verificada** — o que a B2 fez foi parar de
  afirma-la, nao conferi-la. Conferir exigiria regenerar, e regenerar trocaria o
  corpus que o resto da suite usa.
- **Nao rodei `scripts/test_orphans.ps1`.**
