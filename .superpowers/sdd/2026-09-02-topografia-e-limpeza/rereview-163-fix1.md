# Re-revisão da Task 163 — fix round 1 (`15c66fb`)

Escopo: os dez itens despachados (B1, B2, N1, N2, N4, N5, N6, N7, N9 + a linha
de relatório da N3). Lido **só** o arquivo de diff
`review-fix1-b1fc99d..15c66fb.diff` (258 linhas) e o `## Fix round 1` de
`task-163-report.md`. Nada da árvore de trabalho.

## Progresso

- 23:24 lido o diff da fix round 1 inteiro (4 arquivos)
- 23:24 lido `## Fix round 1` de `task-163-report.md`, incluindo a prova de
  N6/N7 e a medição de `Projetos/`
- 23:25 `od -c` na linha 169 do diff — confirmado byte de controle (ver O1)
- 23:25 varredura de bytes não-ASCII/controle em **todas** as linhas `+` do diff
- 23:26 relatório escrito

## Os dez itens

**B1 — `internal/service/rnf5000_test.go:28-30` — ADDRESSED.** `~100 ms` virou
`~370 ms de mediana (medido em 2026-09-04: mediana de 371,5 ms)`. Bate com a
única medição colada no relatório (rodada de 03:08: `Mediana: 371.5013ms`), e a
data está no próprio texto. O número deixou de ser inventado e passou a ser
rastreável até uma saída colada.

**B2 — `internal/service/rnf5000_test.go:168-171` — REGRESSION.** A substância
está certa: o comentário parou de afirmar `-Seed 42` e passa a nomear a semente
como não verificada, com o motivo (diretório reaproveitado, não regenerado) —
exatamente o que o achado pedia, e coerente com a seção "Fora do escopo" do
relatório. **Mas o texto novo corrompeu o caminho.** Ver O1 abaixo: o comentário
agora diz `%TEMP%<VT>ault_5000` em vez de `%TEMP%\vault_5000`. O achado que
existia para tornar a procedência do corpus honesta acabou nomeando um diretório
que não existe.

**N1 — `internal/service/rnf5000_test.go:179` — ADDRESSED.** A linha do filtro de
pasta passou de `pasta existe / busca 1165` para `1165 .md sob Projetos/ / busca
1165` — as duas contagens independentes, como as outras seis linhas da tabela. O
número aparece no relatório como medição desta máquina, com o comando colado
(`find ./Projetos -type f -name '*.md' | wc -l` -> `1165`) e o caminho absoluto
confirmado por `pwd -W`. O `find .` de controle (5000) está junto. Confere.

**N2 — `internal/service/rnf5000_test.go:147-148` — ADDRESSED.** `"quer >= 5000"`
literal virou `"quer >= %d"` com `rnf5000Notas` passado como argumento. A
constante volta a ser a conta única do tamanho do cofre, inclusive no texto do
erro.

**N4 — `internal/index/parity_test.go:29-30` — ADDRESSED, com um trade-off que
vale registrar.** `slices.Compact` depois do `Sort`, nos dois lados, com o
comentário explicando a união `tags + frontmatterTags` e dizendo que nenhuma nota
do corpus de hoje dispara o caso. `Compact` remove duplicata **consecutiva**, e
vem depois do `Sort` — está correto. O trade-off está em O2.

**N5 — `internal/index/parity_test.go:56-60` — ADDRESSED.** `os.Stat` que falha
por qualquer motivo que não `fs.ErrNotExist` agora é `t.Fatalf` citando o erro e
o caminho; só a ausência pula. A mensagem falsa ("existe e não tem nota nenhuma"
para um erro de permissão) deixou de ser alcançável, e o comentário registra o
modo de falha que a motivou.

**N6 — `scripts/verify.ps1:206-219` — ADDRESSED, e provado.** Antes do
`-Tail 80`, a etapa despeja `Select-String -Pattern '^(--- FAIL|WARNING: DATA
RACE|FAIL\s)'` do log **inteiro**, com o número da linha, entre dois cabeçalhos.
A prova é uma falha injetada de verdade (`t.Fatal` em
`TestMarcadoresContinuamEmASCII`), não um raciocínio: a saída colada mostra
`97:--- FAIL: TestMarcadoresContinuamEmASCII` e `105:FAIL github.com/...`, e o
relatório aponta o que fecha o argumento — o rabo de 80 linhas era todo
`--- PASS`, então sem o despejo novo a linha que nomeia a falha não apareceria.
A injeção foi removida e a remoção conferida por `git diff --stat` e por `grep -rn
"INJECAO DELIBERADA"` vazio. O `unreachable-code` do `golangci-lint` na mesma
rodada está declarado como ruído da injeção.

**N7 — `scripts/verify.ps1:243-248` — ADDRESSED, e provado na mesma rodada.**
`if ($Failed -contains "go test -race")` imprime `[!] 6 testes pulados (suite
reprovou; contagem parcial)`; o ramo normal fica intacto. A linha aparece na
prova injetada, e a rodada verde do fim (23:16-23:20) mostra a mesma etapa sem o
sufixo — as duas metades do condicional exercitadas. O nome comparado
(`"go test -race"`) casa exatamente o `-Name` do `Invoke-Step`, e `$Failed` é a
mesma variável de escopo de script que `Invoke-Step` alimenta.

**N9 — `internal/service/read_test.go:88-93` — ADDRESSED.** A lápide passou a
dizer "existe de verdade, NO WINDOWS, em cloudonly_replace_windows_test.go", e
acrescenta que o arquivo é `//go:build windows`, que fora do Windows a cobertura
substituta é zero, e por que isso não é perda (o `t.Skip` incondicional cobria
zero em todo lugar). A frase deixou de prometer mais do que entrega.

**N3 — linha de relatório — ADDRESSED.** Está em "Fora do escopo da fix round 1",
com a decisão e o motivo: o cofre de 5.000 notas é de ambiente, nenhum checkout o
traz, e transformar a ausência em falha reprovaria o gate de toda máquina que
ainda não rodou `gen_vault.ps1`; a mitigação é a etapa 3, que passou a **nomear**
cada teste pulado, de modo que numa máquina sem o cofre o
`TestScale5000_RNF01_RNF02_RNF07_RNF04` aparece na lista. É a decisão do
orquestrador, registrada onde o próximo leitor a encontra. Não re-litigado.

## Itens abertos

**O1. `internal/service/rnf5000_test.go:169` (linha 169 do diff) — o caminho do
cofre no comentário carrega um byte de controle e nomeia um diretório inexistente
— BLOCANTE — CONFIRMADO por `od -c`**

O texto commitado é:

```
+  \t   /   /       %   T   E   M   P   %  \v   a   u   l   t   _   5   0   0   0
```

Isto é, `%TEMP%` seguido de **0x0B (vertical tab)** e `ault_5000`. O `\v` de
`\vault_5000` foi interpretado como escape de tabulação vertical por alguma
etapa da edição, e o que ficou no arquivo `.go` é um caractere de controle cru
dentro de um comentário, com o `\v` do caminho comido.

Que não é escolha deliberada está provado pelo próprio relatório, que escreve o
caminho certo três vezes na seção da N1 (`%TEMP%\vault_5000`,
`C:\Users\jonyd\AppData\Local\Temp\vault_5000`) e no bloco citado da B2 não
mostra esta linha.

Por que o gate não pegou: `gofmt` preserva bytes de comentário, `go vet` não
inspeciona texto de comentário, e `golangci-lint` com a config atual também não —
a rodada verde de 23:16-23:20 é real e não contradiz o achado.

O dano é exatamente o que a B2 vinha consertar: o comentário que passou a ser a
fonte de verdade sobre **qual** diretório foi medido agora nomeia um caminho que
ninguém consegue abrir, e quem for reproduzir as contagens não acha o cofre.

**Fix:** escrever `%TEMP%\vault_5000` literal na linha 169 (e usar uma ferramenta
de edição que não interprete `\v` — o `Edit` direto, não uma substituição por
regex nem uma string entre aspas duplas). Vale um `grep -nP '[^\x09\x20-\x7e]'`
nos arquivos tocados antes de recommitar, restrito a bytes de controle: o mesmo
comando que usei aqui não acusou nenhum outro no diff inteiro.

**O2. `internal/index/parity_test.go:29-30` — o `Compact` nos dois lados descarta
a multiplicidade em TODAS as cinco categorias, não só em `tags` — NÃO BLOCANTE —
CONFIRMADO**

`conjuntosIguais` é a função única de comparação de `headings`, `tags`, `blocks`,
`links` e `embeds`. Com `Compact` dentro dela, uma nota com dois headings
idênticos (`## Notas` duas vezes) ou dois links para o mesmo alvo deixa de ter a
contagem comparada: se perdêssemos um dos dois, os dois lados agora colapsam para
um item só e o teste passa. Antes desta fix round, essa divergência reprovava.

O achado N4 que despachei oferecia as duas formas ("nos dois lados dentro de
`conjuntosIguais`, ou só no lado das tags"), e o implementador escolheu uma
delas — então isto **não** é desvio do que foi pedido. Registro porque a troca
tem um lado que o meu achado não pesou: o próprio arquivo, em
`parity_test.go:111-115`, declara que a contagem de arestas é carga útil ("A
contagem importa, nao so a presenca do alvo... Provado por mutacao — a versao
anterior desta comparacao passava"). `assertGraphMatches` continua contando, o
que cobre o caso resolvido e só na direção `>=`, e só quando `ref.HasGraph()`.

**Fix (se o orquestrador quiser a forma estrita):** tirar o `Compact` de
`conjuntosIguais` e aplicá-lo só na chamada de `tags`, onde a união é a causa da
duplicata — um `slices.Compact(slices.Sorted(...))` no argumento `want` daquela
linha, deixando as outras quatro categorias comparando multiplicidade.

## Verificações que o despacho pediu explicitamente

- **Todo número novo nos comentários aparece nas medições do relatório.** As duas
  entradas novas: `371,5 ms` (relatório, medição de 03:08, saída colada) e `1165`
  (relatório, fix round 1, comando e saída colados, com o `find .` de controle em
  5000). Confere. Os números pré-existentes que a fix round não tocou continuam
  batendo (5.050 arquivos / 1,4 MB; 164,25 / 328,50 MB pela fórmula de
  `PRD.md:294`; 273 x 1,5 = 410). Uma observação sem consequência, que **não**
  conto como item aberto: o `~12,5 MB` do heap vivo é um arredondamento para cima
  do `12,34 MB` medido — pré-existente, sob "~", e na direção conservadora.
- **A saída do `verify.ps1` continua ASCII puro.** Varri todas as linhas `+` do
  diff por byte fora de `\x09\x20-\x7e`. Sete acusadas, e **nenhuma** no bloco do
  `scripts/verify.ps1`: seis são travessões em comentário de código Go (uso
  corrente do repositório, e a regra é sobre marcador de console), e a sétima é o
  O1. As três strings novas que chegam à tela —
  `     --- falhas e corridas no log inteiro ---`,
  `     --- ultimas 80 linhas ---` e
  `(suite reprovou; contagem parcial)` — são ASCII, sem acento inclusive nas
  palavras que o pediriam ("ultimas", "suite").
- **A prova de N6/N7 mostra as duas coisas.** As linhas do `Select-String` com
  número de linha (`97:--- FAIL: ...`, `105:FAIL ...`) entre os dois cabeçalhos,
  **e** a linha `[!] 6 testes pulados (suite reprovou; contagem parcial)` — as
  duas na mesma rodada, com falha injetada de verdade e removida depois, com a
  remoção conferida por dois comandos. O relatório ainda registra que a primeira
  tentativa de colher a prova saiu truncada pelo `head` e foi refeita, o que é o
  tipo de detalhe que costuma sumir.

---

Verdict: NOT APPROVED — 2 itens abertos (1 blocante: O1; 1 não blocante: O2).

---

## Round 2

### Progresso

- 00:08 despacho da round 2 lido; diff `review-fix2-4ffaf50..44f2700.diff` (132
  linhas) lido inteiro; conta `## Fix round 2` do relatorio lida
- 00:08 O1 verificado por varredura de bytes de controle no diff e por `cat -A`
  no blob commitado
- 00:09 O2 verificado por lista de call sites no blob e por contagem
  independente de duplicatas no corpus de paridade
- 00:09 escopo do intervalo `4ffaf50..44f2700` conferido; secao escrita

### O que eu rodei

Somente leitura, sobre o diff e sobre blobs do git — nenhum gate, nenhuma
escrita em codigo.

```
$ grep -nP '[\x00-\x08\x0b\x0c\x0e-\x1f\x7f]' review-fix2-4ffaf50..44f2700.diff
121:-	// %TEMP%ault_5000 (5.000 notas .md; a semente que o gerou NAO foi
```

Uma unica ocorrencia, e ela e a linha `-` (o texto REMOVIDO). A linha `+` nao
aparece na varredura. Confirmado no blob commitado, onde `cat -A` mostraria VT
como `^K`:

```
$ git show 44f2700:internal/service/rnf5000_test.go | grep -n 'TEMP' | cat -A
223:^I// %TEMP%\vault_5000 (5.000 notas .md; a semente que o gerou NAO foi$
```

Call sites no blob commitado:

```
$ git show 44f2700:internal/index/parity_test.go | grep -n 'conjuntosIguais\|semRepetidos\|slices.Compact'
158:func conjuntosIguais(t *testing.T, path, campo string, got, want []string) {
180:func semRepetidos(xs []string) []string {
183:	return slices.Compact(out)
261:	conjuntosIguais(t, path, "links", gotNormais, querNormais)
262:	conjuntosIguais(t, path, "embeds", gotEmbeds, querEmbeds)
346:	conjuntosIguais(t, path, "headings", ...)
352:	conjuntosIguais(t, path, "tags", semRepetidos(note.Tags), semRepetidos(...))
354:	conjuntosIguais(t, path, "blocks", blocosComparaveis(note.Blocks), want.Blocks)
```

Contagem independente de duplicatas em `testdata/parity/metadata.json` do blob
`44f2700`, por nota e por categoria (`headings`, `links`, `embeds`, `tags`,
`frontmatterTags`, `blocks`):

```
notas: 7 categorias com duplicata: 0
notas com tags E frontmatterTags: 0
```

Confirma as duas afirmacoes do relatorio: tirar o `Compact` das outras quatro
categorias nao pode reprovar o corpus de hoje, e a armadilha das tags continua
desarmada mas ainda nao disparavel pelo corpus — que e por isso que a prova do
O2 teve de ser feita contra a funcao, e foi, nas duas direcoes.

Escopo do intervalo:

```
$ git log --oneline 4ffaf50..44f2700
44f2700 docs(sdd): record the fix round 2 commit SHA in the Task 163 report
b19316d test: the rnf5000 vault path is a backslash, not a vertical tab; parity compacts only tag sets

$ git diff --stat 4ffaf50 44f2700
 .../task-163-report.md            | 185 +++++++++++++++++++++
 internal/index/parity_test.go     |  41 +++--
 internal/service/rnf5000_test.go  |   2 +-
```

Dois arquivos `_test.go` e o relatorio. Dentro do escopo permitido a 163.
Nao ha `internal/index/scratch*` — o rascunho da prova do O2 foi apagado, como o
relatorio diz.

### Achados

O1 — ADDRESSED. `internal/service/rnf5000_test.go:223` le
`// %TEMP%\vault_5000`, com barra invertida de verdade: o 0x0B nao aparece na
varredura da linha `+` do diff nem no blob commitado, e `cat -A` no blob mostra
`\v` como dois caracteres (`\` + `v`), nao como `^K`. A varredura do diff
inteiro devolve uma unica linha, a `-`, que e o texto removido. O relatorio
nomeia a causa (string Python nao-raw) e registra a varredura dos cinco
arquivos da round 1 — nao a repeti nos arquivos fora deste diff, e o lead
informou ter confirmado zero bytes de controle nos dois arquivos em HEAD.

O2 — ADDRESSED. `internal/index/parity_test.go:158-166`: `conjuntosIguais`
clona, ordena e compara com `slices.Equal`, sem `Compact` — multiconjunto de
novo. O unico `slices.Compact` do arquivo esta em `semRepetidos`
(`parity_test.go:180-184`), e o unico ponto de chamada de `semRepetidos` e a
comparacao de tags (`parity_test.go:352-353`), nos dois lados da uniao. As
outras quatro categorias — links (:261), embeds (:262), headings (:346) e
blocks (:354) — passam direto por `conjuntosIguais` e voltam a enxergar
multiplicidade. Os dois comentarios cobrem os dois erros possiveis: o de
`conjuntosIguais` diz por que a deduplicacao nao mora nela, com o mecanismo (um
de dois links identicos perdido passaria), e o de `semRepetidos` diz para nao
generalizar. A prova esta no passado e nas duas direcoes — com o conserto o
link perdido e PEGO, com o codigo da round 1 devolvido ele PASSA — e o subteste
de tags passa nas duas rodadas, o que mostra que o O2 nao desfez o N4.

Nao verificado por mim: a linha `verify.ps1 verde, 14/14, [!] 6 testes pulados`
do Progresso do implementador. O despacho e so-leitura e sobre o arquivo de
diff; nao rodei gate nenhum.

Verdict: APPROVED — 0 itens abertos.
