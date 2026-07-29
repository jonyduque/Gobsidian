### Task 45: `internal/search/inverted.go` — dicionário de termos e posting lists

RF-20, P0. E o ponto onde a busca passa a acompanhar o watcher.

#### Onde isto encaixa

Consome os tokens da Task 44. É o que a Task 46 pontua e a Task 49 persiste. E é aqui que a busca se liga ao ciclo incremental que o M2 construiu: `docs/ARCHITECTURE.md` §5.3 já descreve `search.Update(nota)` no pipeline, depois de `index.Replace`.

#### O que já está fechado e vincula esta tarefa

- **Escritas são serializadas na goroutine do watcher.** O índice invertido é escrito de lá e lido pela goroutine que atende MCP. `sync.RWMutex` como no `internal/index`, e prove com `go test -race`.
- **A `generation` do `internal/index` é a única.** Não crie um segundo contador.
- **Remoção tem de ser completa.** `index.Remove` já limpa backlinks, tags, aliases e `byName`. O análogo aqui: um termo que só ocorria na nota removida sai do dicionário, não fica com posting list vazia. Uma posting list vazia é o `alias_collisions: 0` desta camada — ocupa espaço, aparece na contagem, e nunca é verdade.
- **`ctx` onde bloqueia.** Indexar um lote grande percorre muitas notas; receba `ctx` e respeite-o. Inserir um token em memória não bloqueia.

#### A decisão que esta tarefa precisa tomar certo

**Uma posting list por termo, compartilhada pelas duas formas.** A forma crua e a forma reduzida do mesmo token apontam para **a mesma** lista, não para duas. Duas listas dobram o índice e fazem a mesma nota ser contada duas vezes no BM25 da Task 46 — o que infla o score de todo termo que reduz, silenciosamente.

**A posting list guarda posições, não só documentos.** A Task 47 precisa do offset para recortar e a busca por frase precisa da sequência.

#### Armadilhas já pagas que se aplicam

- **Determinismo sob paralelismo.** A ordem de iteração de mapa em Go é aleatória por projeto. Se o resultado depender de ordem, ordene explicitamente. Um teste que passa dez vezes e falha na décima primeira é o que essa aleatoriedade produz.
- **Entrada descartada em silêncio.** Se uma nota for pulada na indexação — parse falhou, arquivo sumiu —, **conte e reporte por motivo**, como `vault.SkippedEntries()` e como os contadores do watcher.
- **Chave derivada calculada em dois lugares diverge.** O termo indexado e o termo consultado passam pela **mesma** função de análise. Se a consulta normalizar por um caminho e a indexação por outro, nada casa e o sintoma é busca vazia, não erro. Foi assim que `byAlias` quebrou.

#### O teste que sustenta a tarefa

```go
func TestInvertedRemoveLeavesNoEmptyPosting(t *testing.T) {
	ix := search.NewInverted()
	ix.Add("a.md", search.Analyze("prescricao intercorrente"))
	ix.Add("b.md", search.Analyze("prescricao civil"))

	ix.Remove("a.md")

	// "prescricao" continua, porque b.md ainda o tem.
	if got := ix.Postings("prescricao"); len(got) != 1 || got[0].Path != "b.md" {
		t.Errorf("postings de termo compartilhado = %+v, quer so b.md", got)
	}
	// "intercorrente" so existia em a.md: o TERMO tem de sair do dicionario,
	// nao ficar com lista vazia. Lista vazia ocupa espaco, entra na contagem
	// de termos, e nunca corresponde a nada.
	if ix.HasTerm("intercorrente") {
		t.Errorf("termo orfao continua no dicionario com posting list vazia")
	}
	if n := ix.TermCount(); n != 2 { // prescricao, civil
		t.Errorf("TermCount = %d, quer 2 — termo orfao esta sendo contado", n)
	}
}
```

#### Verificações além dos passos

- Um termo que reduz — `prescrições` — produz **uma** posting list ou duas? Conte, não observe.
- Reindexar a mesma nota duas vezes duplica as posições?
- `go test -race` acusa corrida entre `Add` na goroutine do watcher e `Postings` na do MCP? Escreva o teste que faz as duas ao mesmo tempo.
- Uma nota removida e recriada com conteúdo diferente deixa resíduo do conteúdo antigo?
- Quantos termos e quanta memória num cofre de teste? Números medidos, com a contagem de notas ao lado, ou **"não medido"**.
- **Ponta a ponta com o watcher:** escreva uma nota nova com o servidor rodando e confirme que ela passa a ser encontrável. É o análogo do teste que provou que o M2 existe.

**Prova de mutação obrigatória:** desligue a remoção do termo órfão do dicionário e confirme que `TestInvertedRemoveLeavesNoEmptyPosting` reprova.

#### Regras de execução e contrato de relatório

Idênticos aos da Task 43. Relatório em `.superpowers/sdd/task-45-report.md`.

**Files:** Create `internal/search/inverted.go`, `internal/search/inverted_test.go`; Modify `internal/watcher/apply.go` (ligar `search.Update` depois de `index.Replace`), `docs/ARCHITECTURE.md` §5.3
**Commit:** `feat(search): inverted index with incremental update`

---

