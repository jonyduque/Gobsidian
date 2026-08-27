# Task 136 — B1, o bloco de contratos, M14 e os achados de desempenho

**Status:** DONE
**Commit:** `08e12fc` — `fix(service,index,search): close B1, M2-M5, M14 and the measurable perf items`

Onze achados fechados. **Dois rejeitados depois de verificados** — e essa metade
é a que vale mais, porque implementá-los teria piorado o produto.

---

## B1 — um teto que não cabia no tipo que ele dimensiona

`limitePosicoes = 4_000_000_000`, quase o **dobro** de `math.MaxInt32`.

Duas falhas na mesma constante, e a primeira é a séria:

**1. Estouro silencioso.** `termIni`, `postIni` e `postPath` são `[]int32` e
guardam índices dentro das fatias que esses tetos dimensionam. Acima de
`MaxInt32`, `int32(kPos)` dá a volta **sem erro**: o cache decodifica "com
sucesso" e serve as posições de outro termo. Não há sintoma — há resultado
errado.

**2. 64 GB de alocação.** `make([]TokenPosition, totPos)` a 16 bytes por posição,
executado a partir do **cabeçalho**, antes de qualquer verificação do corpo. Um
byte trocado por corrupção de disco derrubava o processo por OOM em vez de
devolver "cache corrompido" e reconstruir — que é exatamente o que os tetos
existem para fazer.

O cofre de referência tem **17,8 milhões** de posições. O teto passou a acompanhar
`limitePostings`: 200 milhões, 11× de folga, 3,2 GB no pior caso.

### A guarda é de compilação, e isso é deliberado

```go
const _ = uint(math.MaxInt32-limiteTermos) +
	uint(math.MaxInt32-limitePostings) +
	uint(math.MaxInt32-limitePosicoes) +
	uint(math.MaxInt32-limiteCaminhos)
```

Converter constante negativa para `uint` não compila. Restaurando o valor antigo:

```
internal\search\persist_codec.go:102:7: constant -1852516353 overflows uint
```

**Está no código-fonte e não num teste porque teste que ninguém rodou não impede
o commit.** Os testes existem também, e nomeiam a regra para quem só ler testes.

O terceiro teste exercita o caminho real — cabeçalho declarando 3 bilhões de
posições — e confere que a mensagem **nomeia o campo adulterado**:

```
cache file corrupted: totalPosicoes = 3000000000, acima do limite de 200000000
```

Sem essa asserção, uma mudança de layout faria o teste corromper outro varint,
o cache seria recusado por outro motivo, e ele continuaria verde cobrindo uma
regra que deixou de exercitar.

---

## M2 — um erro de segurança usado como erro genérico

`LinkGraph` e `NoteMetadata` embrulhavam **qualquer** falha de `ResolvePath` como
`PATH_OUTSIDE_VAULT`. O host lê esse código como tentativa de escapar do cofre:
**errar um nome de nota passava a acusar o cliente de algo que ele não fez.**

Pior: `ResolvePath` **não verifica confinamento**. Ela só procura no índice.
O código afirmava algo que a função não tinha como saber.

Havia seis chamadores e **três** classificações diferentes para a mesma falha —
`read.go` distinguia ambiguidade, `../` e não-encontrado; `write.go` distinguia
ambiguidade e devolvia `NOTE_NOT_FOUND`; `graph.go` respondia `PATH_OUTSIDE_VAULT`
para tudo. Agora há o sentinela `index.ErrPathNotFound` e **uma** função,
`service.ErroDeResolucao`.

O contrapeso está no teste: caminho com `../` **continua** classificado como saída
do cofre. Sem isso, "corrigir" o M2 seria só apagar o sinal.

---

## M3, M4, M5 — a normativa estava certa, de novo

Os três são o mesmo defeito do A8: `docs/TOOLS.md` prometia, o código entregava
menos.

- **M3** — `headings` devolvia `[]string` com só o texto. Sem slug não dá para
  montar a âncora; sem offsets não dá para planejar leitura seletiva. Que é
  literalmente o que o contrato diz que o campo serve para fazer.
- **M4** — `inline_fields` estava no enum, era aceito, e descartado.
- **M5** — nós sem distância, arestas sem alias/âncora/resolvido.

**M5 trouxe uma decisão que não era óbvia.** Alias e âncora mudam a *identidade*
da aresta: `A->B#Prescricao` e `A->B#Honorarios` são referências diferentes. A
chave de deduplicação era `origem+destino`, então as duas colapsavam numa aresta
só — e publicar a âncora de uma delas seria escolher arbitrariamente qual contar,
que é a mesma classe de "campo de API com valor fixo". A chave passou a incluir
tipo, alias e âncora, montada por uma função só, porque os três pontos que
inserem em `edgesMap` precisam concordar.

**Nenhum teste cobria nenhum dos três.** A suíte inteira passou verde antes da
correção — o que é o próprio achado.

---

## M14 — duas contas de EOL com semânticas diferentes

`writer.DetectEOL` chamava o arquivo inteiro de CRLF se houvesse **qualquer**
quebra CRLF. `vault.DetectEOL` usa o estilo **predominante** — e é a resposta
dela que o índice persiste em `Note.EOL`.

Um arquivo com uma linha CRLF e mil LF era **LF para o índice e CRLF para a
escrita**.

As duas concordavam na maioria dos arquivos reais, e é isso que fez o defeito
sobreviver: **contas que divergem só na borda são as que ninguém percebe.**

O teste compara as duas em seis cenários, mas isso sozinho não bastaria: alinhar
as duas na regra *errada* passaria. Por isso há um segundo teste que nomeia a
regra escolhida — uma linha CRLF não torna o arquivo CRLF.

---

## Desempenho: um ganho grande, dois dentro do ruído, um não medido

Tudo intercalado, com binários lado a lado e **uma rodada de cada por vez**.

| Achado | Medida | Veredito |
|---|---|---|
| **P6** — `TotalSize` memorizado contra `generation` | **47.155 → 68,55 ns/op** | **688×** |
| **P8** — laço quadrático de citantes removido | 4,905 → 4,865 s (Build de 8.000 notas, alvo-hub), n=5 | **−0,8%, dentro do ruído** |
| **P10+P13** — termos analisados uma vez; um `index.Get` por hit | 22,28 → 22,10 ms, n=8 | **dentro do ruído** |
| **P14** — atributos de nuvem lidos do `FileInfo` que já está em mãos | **não medido em relógio** | trabalho removido é certo pelo código |

**P8 e P10/P13 ficam pela forma, não pela velocidade.** O laço quadrático era
real — 32 milhões de comparações a 8.000 notas — e some dentro de um Build de
4,9 s dominado por I/O e parse. Chamar isso de otimização seria a mesma classe de
afirmação que esta sessão já retratou duas vezes.

**P6 escolheu memorização contra `generation`, e não um total mantido a cada
insert e remoção.** A segunda forma é mais rápida e teria de ser mantida em cinco
lugares — e o quinto é o que esquece. O teste cobre as **duas** direções, porque
o risco da correção é maior que o do defeito: memorizar sem invalidar congela
`vault_stats` no valor do boot, em silêncio.

**P14 mudou uma regra de segurança do produto, então foi testado como tal.** Se
`info.Sys()` não trouxesse `*syscall.Win32FileAttributeData`, a versão nova
devolveria `false` por não saber, e a detecção de somente-nuvem sumiria sem nada
falhar — o índice passaria a **abrir placeholders**, disparando download síncrono
do cofre inteiro no boot. Dois testes: um compara as duas funções com
`FILE_ATTRIBUTE_OFFLINE` posto à mão, outro atravessa o `Walk` de verdade, porque
o que importa é o `fs.FileInfo` que o `WalkDir` entrega, não um do `Lstat` do
teste. Os dois passam: o `FileInfo` **carrega** os atributos.

---

## Os dois rejeitados

### M1 estava prescrito ao contrário

Ele manda `note_delete` adotar o critério de `note_move` (`write.go:479`), que
reporta só âncoras **já** ausentes.

No **move** a nota sobrevive com os headings, então só as âncoras já quebradas
interessam. No **delete** a nota some, e **toda** referência ancorada quebra —
inclusive as que apontam para heading existente. `docs/TOOLS.md` diz que a tool
lista o que *"passará a ter"* links quebrados.

Aplicar M1 esconderia exatamente as âncoras que quebram por causa do delete.

### P11 partia de premissa falsa

Ele dizia que a varredura de temporários pulava, em silêncio, subárvores além de
MAX_PATH por falta do prefixo de caminho longo.

Sondado: **o pacote `os` do Go aplica o prefixo sozinho** (`fixLongPath`).

```
MkdirAll SEM prefixo (310 chars): OK
WriteFile SEM prefixo (318 chars): OK
WalkDir SEM prefixo alcancou o arquivo profundo: true
```

A correção **chegou a ser escrita** — `LongPathSempre` para a raiz — e a prova de
mutação a reprovou com EXIT=1: trocá-la por `vault.LongPath`, que para raiz curta
é identidade e portanto o comportamento antigo, deixou o teste passando. Guarda
morta, removida junto com a função que nasceu para ela.

**A metade que P11 acertou ficou:** a varredura descartava erro de subárvore em
silêncio, então "varri e não achei nada" e "não consegui entrar em trinta
diretórios" davam a mesma resposta — nenhuma. `SweepResult` passou a carregar
removidos, não-removidos e inacessíveis, e o boot loga os três.

---

## A prova de mutação matou uma guarda minha, de novo

`headingDoLink` (Task 135) e o prefixo do P11 (esta). Nos dois casos a mutação
devolveu EXIT=1 — "o teste passou com a regra mutada" — e a leitura seguinte
mostrou que a regra não fazia nada.

Nos dois casos meu primeiro instinto foi consertar a **mutação**. Nos dois a
mutação estava certa. Registrado em `ARMADILHAS.md`: **é a mutação que distingue
proteção de decoração.**

---

## Provas de mutação

Todas `EXIT=0`, salvo a de B1, que é de compilação.

```
B1   (compilacao)  limitePosicoes = 4_000_000_000
                   -> constant -1852516353 overflows uint

M2   CodeNoteNotFound -> CodePathOutsideVault em ErroDeResolucao
M3   res.Headings = n.Headings -> nil
M4   res.InlineFields = n.Inline -> nil
M5a  Distance: curr.Depth -> Distance: 0
M5b  chaveDaAresta sem alias e ancora
       -> "nenhuma aresta trouxe o alias da referencia"
          arestas: [{... Alias: Anchor:Secao ...}]   (as duas colapsaram)
M14  writer.DetectEOL volta a regra antiga
       -> writer.DetectEOL = "\r\n", vault.DetectEOL = "\n"
P6   memorizacao sem conferir a geracao
P8   append duplicado na lista de citantes
```

`M14` e `P11` exigiram manter o import vivo na mutação (`_ = vault.EOLLF`),
porque a primeira versão quebrava a compilação e devolvia EXIT=2 —
**inconclusivo**, que o script distingue de reprovação de propósito.

---

## Verificações

1. `pwsh -File scripts/verify.ps1`: **bateria completa, EXIT=0.**
2. `golangci-lint run ./...`: **0 issues** — depois de corrigir três que a
   correção introduziu, incluindo `citantesAtuais` virando função órfã quando o
   P8 removeu suas duas chamadas.
3. `go test -race ./...`: verde. **`TestDebounce_Coalescence` reprovou uma vez**
   durante a sessão, com a máquina carregada pelas medições, e passou em três
   execuções isoladas e na repetição do pacote. É teste sensível a tempo sob
   carga; não foi investigado além disso.
