# Re-review — Task 182, fix round 1 (8e684d0..c987ff1)

Revisor: agente `rev-182` (re-review). Somente leitura: nao rodei teste
(nenhuma duvida especifica exigiu), nao toquei working tree, index nem HEAD.
Tudo abaixo saiu do diff `review-8e684d0..c987ff1.diff` e do relatorio,
conferidos contra o codigo atual em `internal/index/update.go` e
`internal/service/write.go`.

## Veredictos por achado

### F1 (blocking) — ADDRESSED

`internal/service/write.go:481-484` (numeracao atual): guarda
`if rl.Target == "" { continue }` inserida dentro do `if rl.Resolved ==
canonicalFrom`, ANTES do calculo de `brokenAnchors` e `newTarget` — exatamente
o discriminante que a revisao exigiu (`rl.Target == ""`, nao `bl.From ==
canonicalFrom`).

Verificado por construcao que o guarda nao pode descartar um citante
LEGITIMO de OUTRA nota: um link com `Target == ""` so resolve com
`Resolved == origin` (a nota que CONTEM o link — regra da Task 182 em
`resolve.go`), entao `rl.Resolved == canonicalFrom && rl.Target == ""`
implica `bl.From == canonicalFrom`. Nunca dispara para `bl.From` diferente de
`canonicalFrom`. Efeito colateral bom, nao pedido explicitamente mas correto:
com `replacements` vazio para a auto-referencia, `affectedNotes[canonicalFrom]`
nunca e populado, entao o invariante "a origem nao entra em `diffs`" do
`DryRun` (linha ~540) tambem fica correto — o sintoma que a revisao apontou
(linha 66-68 do review original) desaparece pela mesma correcao, sem teste
dedicado a essa metade mas decorrencia direta e verificavel lendo o codigo.

Teste `TestMoveNote_LinkSoDeAncoraNaoQuebraOMove`
(`internal/service/anchor_selfref_test.go:265`) prova as duas metades do
discriminante: as tres formas de ancora saem intactas (RED comprovado por
ENOENT antes da correcao) E `ref.md`, que cita `[[a]]` de verdade, e reescrito
para `[[b]]` com `LinksUpdated == 1` — se o guarda fosse largo demais e
desligasse toda reescrita, esse assert reprovaria. O teste discrimina
corretamente.

**Limite do teste, honestidade da declaracao:** o teste evita colocar `[[a]]`
(auto-referencia com alvo escrito) DENTRO da nota movida porque isso dispara
um ENOENT pre-existente e nao relacionado — medido pelo implementador com uma
sonda descartada, saida colada (`MoveNote: lendo nota "a.md": ... cannot find
the file`). O comentario do teste (linhas 246-264 do diff) declara esse limite
explicitamente, nomeia o defeito, data a medicao (2026-09-06) e diz por que a
segunda metade do discriminante (`"[[a]]" dentro de a.md continua sendo
reescrita`) NAO E testavel aqui. Isso e exatamente o padrao "registrar o
limite, nao escondê-lo" que a casa cobra. Confirmado honesto.

### F2 (blocking) — ADDRESSED

`internal/index/update.go:479-484` (bloco novo, logo apos `n := &movida`):

```go
for i := range n.Links {
    if n.Links[i].Resolved == oldPath {
        n.Links[i].Resolved = newPath
    }
}
```

Local exato que a revisao pediu. Tracei a funcao inteira com esse bloco no
lugar: apos o loop, `n.Links[0].Resolved` (o `[[#Topo]]`) vira `newPath` antes
de `ix.notes[newPath] = n`. No Passo 6, `ix.backlinks[oldPath]` (que contem
`{From: oldPath}`) e movido para `ix.backlinks[newPath]` — a tentativa de
corrigir via `ix.notes[bl.From]` ainda falha (`bl.From == oldPath`, ja
apagado), mas isso deixou de importar: no Passo 7, o loop `for _, l := range
n.Links` ve `l.Resolved == newPath` (ja corrigido pelo bloco novo),
`bls := ix.backlinks[newPath]` acha a MESMA lista movida no Passo 6, e
`bl.From == oldPath` casa e vira `newPath` **in place** (mesma slice
compartilhada entre os dois mapas ate a reatribuicao). Estado final:
`Resolved == newPath` e `backlinks[newPath][0].From == newPath` — os dois
asserts de `TestMoveNote_LinkSoDeAncoraSegueANota` (`move_test.go:41-80`).

**O rewrite nao toca links que apontam para outro lugar:** o filtro e
`Resolved == oldPath`, e como caminho canonico e chave unica, isso so pode
casar com links de SAIDA da propria nota que resolviam para si mesma (anchor-
only, ou `[[a]]` por nome) — nunca um link de saida que aponta para outra
nota. Nenhum caso de sobre-escrita espuria encontrado.

RED colado (`Resolved="a.md" State=ok, quer "b/a.md"/ok` e o aviso de
backlink morto) e GREEN colado. Mutation proof 2 remove exatamente esse bloco
e reproduz o mesmo RED. Prova real.

### F3 (should-fix) — ADDRESSED

`internal/service/write.go:707-709` (numeracao atual): `if bl.From ==
canonical { continue }` no topo do laco de `DeleteNote`, discriminante
correto para este caso (nota apagada nao existe mais, entao nenhum link dela
sobrevive — ao contrario de F1, onde a nota continua existindo no caminho
novo). RED/GREEN colados, mutation proof 3 remove o guarda e reproduz o
mesmo RED (`BrokenLinks` e `BrokenAnchors` acusando `a.md`).

### F4 — ADDRESSED

Secao "Concerns" do relatorio reescrita: a frase "parity_test verde => nao ha
divergencia medida" foi substituida por uma que nomeia a assimetria de
`assertGraphMatches` e diz explicitamente "nao medido contra cofre real...
nem a favor nem contra". Corresponde ao texto que F4 pedia.

### Nits F6, F7 — ADDRESSED

F6: `internal/index/resolve_anchor_test.go` agora usa `v, err := vault.New(root)`
com `t.Fatalf` no erro (era `v, _ :=`). F7: `![[#Topo]]` entrou na tabela como
caso 3, `note.Links` agora e 7 (era 6), coberto por `TestAncoraNaMesmaNota`
GREEN colado.

`strings.Cut` em `ast.go`: `target, anchor, _ = strings.Cut(s, "#")` e
equivalente ao `IndexByte` anterior — quando nao acha `'#'`,
`strings.Cut` devolve `(s, "", false)`, igual ao `return s, ""` que existia.
Ordem (split ANTES de `PercentDecode`) preservada — o refactor e so dentro do
corpo de `splitAnchor`, nada mudou em quem a chama. A prova de mutacao extra
da rodada anterior (inverter a ordem) continua valendo porque a ordem em si
nao foi tocada.

### Parqueados (nao contam como pendentes) — confirmados fora de escopo

F5 (hook `pre_commit_docs.ps1`), F8 (`[x](b.md#)` perde `#`), linha de
`docs/TOOLS.md` sobre aresta `source == target` — nenhum foi tocado nesta
rodada, e o relatorio declara os tres explicitamente como nao endereçados
com o motivo (ticket propria / registrado sem mudar / Task 183-184). Correto,
consistente com o que a revisao original pediu.

## Novos achados na inspecao do diff — nenhum blocking ou should-fix

Nenhuma quebra nova encontrada. Dois pontos menores, sem acao pedida:

- O comentario novo em `update.go:479` cita "passo 1b" pelo nome de secao
  ("logo abaixo") em vez de linha — estilo ja usado no arquivo (outros
  comentarios tambem citam passos por numero), consistente, nao e achado.
- O bloco de `write.go:481-484` duplica, em comentario, a explicacao de por
  que `bl.From == canonicalFrom` seria o discriminante errado — a mesma
  explicacao que ja esta no `move_test.go`/relatorio. Redundante entre
  comentario de codigo e teste, mas nao e defeito: o comentario de codigo e o
  que sobrevive quando o teste for lido isoladamente.

## Verificacoes de disciplina

- Todos os tres testes novos (F1/F3 em `anchor_selfref_test.go`, F2 em
  `move_test.go`) tem prova de mutacao com saida colada, no passado, restauro
  por SHA-256 confirmado, `EXIT=0`. Nenhuma prova escrita no condicional.
- Nenhum numero nao medido citado como medido nesta rodada; o achado
  pre-existente (`[[a]]` derruba `note_move`) e apresentado como medido com a
  sonda colada, e a sonda foi descartada apos a medicao (declarado).
- `verify.ps1` tail colado, 14 etapas, `[OK] Bateria completa. Pode
  commitar.`, `EXIT=0`. Etapa 3 (testes pulados) mencionada como inalterada,
  sem alegar medicao que nao foi feita ("nao medi se essa contagem mudou").
- Ledger: relatorio declara que a linha da Task 182 foi anexada ao
  `progress.md` sem commitar, com o motivo (evitar conflito com o
  orquestrador). Aceitavel, declarado.

## Veredito final

**All findings addressed: yes.** F1, F2 e F3 corrigidos com o discriminante
exato que a revisao pediu, testes que discriminam corretamente (inclusive
provando que o guarda de F1 nao desliga reescrita de citante real), limite
honesto declarado no comentario do teste, mutation proofs reais para as tres,
F4 e os dois nits tambem endereçados, e `verify.ps1` verde. Nenhuma quebra
nova encontrada na inspecao do diff.
