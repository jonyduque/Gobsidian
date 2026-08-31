# Tasks 107, 109, 112 e 114 — as quatro da revisão de 2026-08-15 que estavam em estado nenhum

**Data:** 2026-08-31 · **Gate:** `verify.ps1` 13 de 13

## Como elas foram achadas

Uma auditoria de documentação ia registrar que a revisão de 2026-08-15 tinha
fechado. O ledger registrava 104, 105, 106 e 116 como `complete`; outras oito
(110, 113, 118, 119, 120, 121, 122, 123) tinham sido entregues sob a numeração da
auditoria de 2026-08-25, e o código as confirmava.

Quatro não apareciam em estado nenhum — nem `complete`, nem `BLOCKED`, nem
abertas. Conferência **no código**:

```
match_offset / MatchOffset  -> zero referencias em internal/ e cmd/
noteReadInput.Paths         -> []string
note_outline                -> zero referencias
aliasKey                    -> func aliasKey(a string) string { return strings.ToLower(a) }
```

**Duas semanas e ~70 commits** com quatro tarefas fora de qualquer lista.
Registrado como armadilha em `docs/ARMADILHAS.md` § Ledger e rastreio de tarefa.

---

## Task 114 — a chave do índice normaliza; o caminho, não

### RED — a medição, ANTES de corrigir

O brief manda medir antes. Contra o HEAD (`f7de8e81`):

```
=== RUN   TestCaminhoUnicodeIdaEVolta/travessao                  PASS
=== RUN   TestCaminhoUnicodeIdaEVolta/nfc_ida_e_volta            PASS
=== RUN   TestCaminhoUnicodeIdaEVolta/nfd_ida_e_volta            PASS
=== RUN   TestCaminhoUnicodeIdaEVolta/cria_em_NFD,_le_em_NFC
    caminho_unicode_test.go:51: criou "Capítulo I.md" e ReadNote("Capítulo I.md")
        falhou: nota "Capítulo I.md" nao encontrada no indice
=== RUN   TestCaminhoUnicodeIdaEVolta/cria_em_NFC,_le_em_NFD
    caminho_unicode_test.go:51: (idem)
=== RUN   TestCaminhoUnicodeIdaEVolta/emoji_fora_do_BMP          PASS
--- FAIL: TestCaminhoUnicodeIdaEVolta (0.12s)
```

Os quatro casos de mesma forma passam; **os dois cruzados reprovam**. O defeito
existe.

### A correção

`internal/text.ParaNFC` (NFC puro, sem tocar em caixa nem acento) e
`internal/index/chave.go`, com uma conta por chave: `chaveDeCaminho`,
`chaveDeNomeDeArquivo` e `aliasKey`. Os sete pontos de `index.go`, `resolve.go` e
`update.go` passaram a chamá-las — **inclusive os que já estavam certos**, que é
o ponto da regra: não é consertar os errados, é tornar a próxima divergência
impossível sem tocar na função.

`text.Normalize` **não** serve aqui: ela remove acento, porque existe para BUSCA.
Aplicada a chave de índice faria `Capitulo` e `Capítulo` colidirem numa entrada.

### O EXIT=1, e o que ele revelou

A primeira prova de mutação **passou** com a regra mutada:

```
[!] O teste PASSOU com a regra mutada.
    TestCaminhoUnicodePorCaminhoCompletoComHomonimo nao consegue reprovar sem essa regra
```

`ResolvePath` tem três rotas — caminho exato, `lowerPath` e `byName`. Com a nota
na raiz do cofre, o nome de arquivo **é** o caminho inteiro, e `byName` — que
também normaliza — respondia pela regra mutada. Dois testes novos isolam as
rotas: homônimos em pastas diferentes tornam `byName` ambíguo e forçam o caminho
completo; nome nu em subpasta força `byName`.

### Erro próprio no caminho, que vale registrar

Escrevi as constantes `nfc` e `nfd` como literais Unicode e **as duas saíram em
NFC** — 14 bytes cada. Os dois testes novos eram tautológicos e passavam por
isso. Uma sonda que imprimiu o comprimento em bytes pegou. O arquivo usa escapes
Go agora (`í` e `́`), que é o que o brief pedia desde o começo — e o
motivo de pedir era exatamente este.

### GREEN — depois da correção

```
--- PASS: TestCaminhoUnicodeIdaEVolta (0.07s)
--- PASS: TestCaminhoUnicodeSobreviveAoMove (0.01s)
--- PASS: TestCaminhoUnicodePorCaminhoCompletoComHomonimo (0.01s)
--- PASS: TestCaminhoUnicodePorNomeNuEmSubpasta (0.01s)
```

### Provas de mutação

```
- return strings.ToLower(text.ParaNFC(filepath.ToSlash(path)))
+ return strings.ToLower(filepath.ToSlash(path))
--- FAIL: TestCaminhoUnicodePorCaminhoCompletoComHomonimo (0.02s)
    gravou "a/Capítulo I.md" e ReadNote("a/Capítulo I.md") falhou
[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.

- return text.ParaNFC(base)
+ return base
--- FAIL: TestCaminhoUnicodePorNomeNuEmSubpasta (0.01s)
    gravou livro/"Capítulo I.md" e ReadNote("Capítulo I.md") falhou
[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
```

### O que fica aberto

O disco pode ter as **duas** formas ao mesmo tempo. Duas notas cujos nomes só
diferem na normalização são dois arquivos para o sistema de arquivos e **uma
chave só** para o índice; hoje a segunda ganha o lugar da primeira em
`lowerPath`. **Não medido** em cofre real. Registrado nas dívidas de `ESTADO.md`.

---

## Task 107 — `match_offset`

**RED.** Contra o HEAD, `TestSearchMatchOffsetApontaParaOTermoNoArquivo` nem
compilava: `SearchHit` não tinha o campo. O estado vermelho das Tasks 107, 109 e
112 é o mesmo — o entregável não existia, conferido por busca no código antes de
começar (`match_offset`, `Paths []string`, `note_outline`). **GREEN** é a saída
colada em cada seção, e as provas de mutação são o que separa "passa" de
"verifica".

`Snippet.MatchOffset` sai de `diskMatchStart`, **não** de `winStart+relStart`: o
segundo é o início do trecho depois do aparo de UTF-8 parcial, não o da
ocorrência. `SearchHit.MatchOffset` é `*int64` com `omitempty`, preenchido só
quando há trecho — zero é offset válido (o início do arquivo), e devolvê-lo num
hit sem ocorrência localizada mandaria o cliente ler o começo de uma nota
qualquer acreditando estar indo ao termo.

O caso "sem trecho" tem fixture real: a nota sai do **disco** entre o índice e o
recorte (Obsidian a trava, OneDrive a evita, o usuário a apaga).

### Provas de mutação

```
- MatchOffset: diskMatchStart,
+ MatchOffset: 0,
--- FAIL: match_offset_test.go:49: MatchOffset = 0, e o termo esta a mais de 40 KB do inicio
[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.

- if snip.Text != "" {
+ if true {
--- FAIL: match_offset_test.go:140: hit SEM trecho veio com match_offset=0
[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
```

---

## Task 109 — objeto por item em `paths`

`service.ReadAlvo` com campos **ponteiro** — `{"max_bytes":0}` pede "sem teto" e
é diferente de omitir — e `pedidoDoAlvo` como conta única da herança.

### A descoberta que muda a avaliação do brief

O brief autorizava, como plano B: *"se ela não suportar `oneOf`, descreva as duas
formas na descrição do campo e diga isso no relatório."*

**O plano B teria quebrado todo cliente atual, em silêncio.** O SDK valida a
entrada contra o `InputSchema` **antes** de chamar `UnmarshalJSON`
(`mcp/tool.go:94`, `resolved.Validate(&v)`). Um schema que descrevesse só o
objeto reprovaria a lista de strings na validação, com o `UnmarshalJSON` nunca
chamado.

`setSchema` (`mcp/server.go:426`) só infere quando `Tool.InputSchema` é nil.
Então `schemaDoNoteRead` **infere** a partir das tags — a conta única continua
sendo o struct — e remenda **uma** propriedade com `oneOf`. Escrever o schema
inteiro à mão seria a segunda conta.

Isto está fixado por prova de mutação e registrado em `docs/ARMADILHAS.md`:

```
- {Type: "string"},
+
--- FAIL: lote_misto_test.go:76: note_read devolveu erro
[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.

- if alvo.Heading != nil {
+ if false {
--- FAIL: lote_por_item_test.go:53: item 1 nao sobrepos o heading
[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
```

---

## Task 112 — `note_outline`

`parser.DetectCandidates` **reusa** `openFence`, `closesFence` e `closeSections`.
`Candidate.Level` é ponteiro: sem numeração hierárquica não há nível a afirmar.

Uma decisão que o brief não fixava: candidato **sem** numeração entra no cálculo
de `End` como nível **6**, o mais profundo. Com zero, um `**Introdução**`
engoliria um `**13.1 Algo**` que viesse depois, porque `2 <= 0` é falso.

A mensagem de `HEADING_NOT_FOUND` numa nota sem heading nenhum passou a apontar
`note_outline` — passo 5 do brief.

### Provas de mutação

```
- inFence, fence = true, f
+ inFence, fence = false, f
--- FAIL: quer 2 candidatos (o de dentro da cerca nao conta), tem 3
[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.

- Headings:   note.Headings,
+ Headings:   append(..., parser.Heading{Text: candidatos[0].Text}),
--- FAIL  [OK] a regra esta verificada.

- truncado = true
+ truncado = false
--- FAIL  [OK] a regra esta verificada.
```

*(A mutação que o brief sugeria, `if inFence {` para `if false {`, sai
**EXIT=2**: `inFence` fica sem leitor e o pacote não compila. Mutação que quebra
a compilação não prova cobertura. Trocada pela equivalente acima.)*

---

## A chamada real, contra o `test-vault`

Nota convertida de 30.586 bytes: negrito no lugar de heading, CRLF, bloco de
código cercado.

```
== note_outline ==
   headings=[]  candidates=2  truncated=False
   strong_paragraph level=1    start=0      end=30586  '13 Registro de candidatura'
   strong_paragraph level=3    start=15234  end=30586  '13.1.10 Substituicao de candidatos'

== vault_search "xifopago" ==
   convertida-107-112.md        match_offset=15284

== note_read(offset=15284) = o match_offset da busca ==
   content='xifopago aparece nesta secao e em nenhum'

== note_read com lista MISTA (string + objeto) ==
   item 0: '**13 Registro de candidatura**'                     <- string, herda max_bytes=30
   item 1: '**13.1.10 Substituicao de candidatos**\r\n\r\no t'  <- objeto, sobrepoe offset e max_bytes
```

**A chamada real achou um defeito que o teste unitário não pegava.** A primeira
execução devolveu `"headings": null`. Slice nil vira `null` no JSON, e aí "esta
nota não tem heading nenhum" lê igual a "não sei dizer" — que são exatamente as
duas respostas que a tool existe para separar. O teste não pegava porque compara
`len()`, e `len(nil)` é zero. Corrigido, com `TestOutlineListasVaziasNaoSaoNulas`
fixando.

---

## Documentação

`docs/TOOLS.md` (contrato de `note_outline`, `match_offset`, objeto por item),
`docs/ESTRUTURA.md` (quatro arquivos novos), `docs/ESTADO.md`,
`docs/ARMADILHAS.md` (duas armadilhas novas), `README.md`, e o wiki: doze tools
viraram treze, e as **seis páginas `stale`** foram conferidas contra o código e
reativadas. Nenhuma página do wiki está `stale`.
