# Task 85 — Cache do índice de metadados

**Tier: modelo forte.** É o maior ganho absoluto e o maior risco: um cache de metadados errado serve nota errada, não nota lenta.

#### Onde encaixa
Independente das otimizações de busca. Maior ganho absoluto do lote.

#### O que vincula esta tarefa

Repetido aqui de propósito: o brief é a unidade que viaja, e decisão citada por
código fica no preâmbulo, que não viaja com ela.

- **Otimização que muda resultado é defeito, não trade-off.** O golden de
  ranking da Task 78 (`testdata/ranking/*.tsv`, teste `TestRankingGolden` em
  `internal/service/`) tem de ficar **idêntico**. Golden que muda exige
  explicação escrita e volta para revisão. **Nunca regenerar com `-update` para
  fazer passar** — `-update` grava o que o código produz, não o que está certo.
- **Ordem de acumulação de ponto flutuante não muda.** `CalculateBM25` soma
  `score += idf * tfScore` num laço. Reordenar a iteração muda o arredondamento
  e faz o golden falhar por motivo legítimo; a reação previsível é regenerar, o
  que apaga o gate. Se parecer necessário reordenar, **pare** e escreva por quê.
- **`benchstat` com `-count=6`, uma mudança por vez.** Baseline antes, mudança,
  baseline depois. `~` (sem diferença significativa) **reverte a mudança**:
  código mais feio sem ganho é dívida pura. Colar a saída, não o resumo dela.
- **Teto de latência não é afirmado sob `-race`** (custa 2× a 6×). Asserção de
  tempo fica atrás da constante `raceEnabled`, padrão já existente em
  `internal/service` e `internal/search`.
- **Nenhum teto de RNF é afrouxado nesta batelada.** RNF-04 está em 181 ms
  contra alvo de 100 ms. Alvo não atingido e registrado é informação; alvo
  afrouxado é ficção.

#### Armadilhas já pagas que se aplicam
- **Teste de fallback que deixa o caminho principal ligado mede o caminho
  principal.** Reincidiu duas vezes neste projeto.
- **Chave derivada calculada em dois lugares diverge**, e a divergência aparece
  no caminho menos usado — `[[STJ]]` continuou resolvendo, com `state=ok`, para
  uma nota já removida. Toda chave passa por **uma** função.
- **Campo com valor fixo mente sempre.** `alias_collisions` era `0` literal.
- **Prova de mutação escrita no condicional não é prova.** Tempo verbal no
  passado, com a saída colada.
- **Script Python que edita `.go` converte a sequencia de escape de quebra
  de linha numa quebra literal**, e corrompe a string Go.
  Use `Edit`, não script, para inserir código com escapes.

#### Regras de execução
Rodar `pwsh -File scripts/verify.ps1` antes de dizer que acabou. Registrar no
ledger (`.superpowers/sdd/2026-07-25-gobsidian-v01/progress.md`) **antes** de
reportar conclusão. Escopo não encolhe em silêncio: se alguma parte não deu,
entregue o resto inteiro e diga o que ficou de fora e por quê — `BLOCKED` com
motivo é resposta melhor que entrega que parece completa.

#### A evidência medida do defeito
```
msg="servidor pronto" notes=3149 index_ms=905
```
905 ms, toda partida, varrendo e parseando o cofre inteiro. **RNF-02 não
atingido**: 832–1183 ms contra teto de 300 ms.


#### A decisão que esta tarefa tem de acertar
**Reaproveitar o codec da Task do formato 5**, não inventar outro. O
`persist_codec.go` já resolve tabela de strings, varint com delta, totais
adiantados e portão de versão. Um segundo formato binário no mesmo projeto é
duas cópias da mesma regra, e a que diverge é a menos usada.

**O cabeçalho confere cobertura, não só versão.** A lição já paga: um cache
parcial passou por completo porque `LoadInvertedCache` conferia versão e não
contagem.

**Invalidação é por mtime e tamanho por arquivo**, com o mesmo raciocínio já
escrito em `ARCHITECTURE.md` §.

#### Armadilhas específicas desta tarefa
- **Nota vazia nunca contava como coberta** e fazia todo boot achar o cache
  parcial. O equivalente aqui é qualquer nota que não gere entrada.
- **`DocLength` divergia entre construído e recarregado.** O mesmo teste
  diferencial vale: índice construído do zero e índice carregado do cache têm de
  responder **igual** em `Get`, `Backlinks`, `Tags`, `ResolvePath` e `Paths`.

#### O corpo do teste que não é óbvio
Diferencial, no molde do que pegou o defeito do `DocLength`:
```go
// TestIndiceDeMetadadosRecarregadoEIdentico compara os dois caminhos de
// construcao campo a campo, em vez de conferir valores escritos a mao.
// Valor escrito a mao codifica o mesmo engano do codigo; o caminho de
// construcao do zero e o que ja estava certo.
func TestIndiceDeMetadadosRecarregadoEIdentico(t *testing.T) {
	// ... corpus com: nota com alias, nota com backlink, nota com anchor
	// quebrada, nota VAZIA, anexo, e nome que colide em caixa.
	// Construir do zero -> salvar -> carregar -> comparar:
	//   Paths(), NoteCount(), AssetCount(), TotalSize(), Tags(""),
	//   e por caminho: Get(), Backlinks(), ResolvePath() do nome curto.
}
```

#### Verificações além dos passos
- O diferencial acima cobre nota vazia, alias, âncora quebrada, anexo e colisão
  de caixa. Faltando qualquer um deles, o teste passa e a cobertura não existe.
- Boot real com o cache presente: o log tem de dizer que ele foi usado, e
  `notes=` tem de bater com a varredura.
- Apagar o cache e reiniciar: reconstrói sem erro.
- Corromper um byte no meio do arquivo: recusa como corrompido, não decodifica.

#### Prova de mutação
```
pwsh -File scripts/mutate.ps1 -Path internal/index/persist.go `
  -Anchor 'if h.NoteCount != idx.NoteCount() {' -Replacement 'if false {' `
  -Test TestCacheDeMetadadosParcialERecusado -Package ./internal/index/
```
É a regra que o cache de busca aprendeu na marra: conferir versão e não
cobertura deixou um cache parcial passar por completo.

#### Contrato de relatório
`index_ms` medido em cinco partidas, antes e depois, no cofre real.
Dizer se RNF-02 passou a ser atingido — e **se não passou, dizer o número**.
Prova de mutação da checagem de cobertura do cabeçalho.

**Files:** `internal/index/persist.go` (novo), `cmd/gobsidian/serve.go`,
`docs/PRD.md` (fechar a anotação), `docs/OPERACAO.md`, testes
**Commit:** `perf(index): persist the metadata index and load it at boot`

---

