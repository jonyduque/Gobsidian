### Task 46: `internal/search/bm25.go` — ranking

RF-20, P0. `k1=1.2`, `b=0.75`, pesos: título 3×, headings 2×, corpo 1×.

#### Onde isto encaixa

Consome as posting lists da Task 45. É o que decide a ordem que o usuário vê, e o lugar onde um erro não quebra nada — só piora o resultado, em silêncio.

#### O que já está fechado

- **Os parâmetros são `k1=1.2` e `b=0.75`.** São o padrão da literatura e não são para ajustar por intuição. Se quiser mudá-los, isso é uma medição com corpus, não uma tarefa desta.
- **Os pesos de campo são 3× / 2× / 1×** para título, headings e corpo.
- **A forma crua pontua mais alto que a reduzida.** É o que o PRD §5.3 chama de "o recall vem do stem, a precisão vem de o termo original continuar presente e pontuar mais alto". Se as duas pontuarem igual, a indexação dupla vira só recall e a precisão que ela deveria preservar some.

#### A armadilha específica desta tarefa

**Peso de campo e parâmetro de fórmula são números que ninguém nota estarem errados.** Um `b` trocado por `0.5`, ou o peso de título aplicado ao corpo, produz resultado plausível — ordenado, com scores decrescentes, sem erro. **Cada constante precisa de um teste que reprove se ela mudar.** É o mesmo defeito do campo de API com valor fixo, pelo outro lado: aqui o valor varia, mas ninguém verifica se varia certo.

#### O teste que sustenta a tarefa

```go
func TestBM25FieldWeightsAreApplied(t *testing.T) {
	// Duas notas, mesmo termo, mesma frequencia, mesma extensao. A unica
	// diferenca e ONDE o termo aparece. Se os pesos nao forem aplicados, os
	// scores empatam — e um empate aqui e indistinguivel de "funciona",
	// porque a lista sai ordenada de qualquer jeito.
	noTitulo := scoreDe(t, "prescricao", nota{Titulo: "prescricao", Corpo: "texto texto"})
	noCorpo := scoreDe(t, "prescricao", nota{Titulo: "outra", Corpo: "prescricao texto"})

	if noTitulo <= noCorpo {
		t.Fatalf("titulo=%.4f corpo=%.4f — o peso 3x do titulo nao esta sendo aplicado",
			noTitulo, noCorpo)
	}
	// A razao nao e exatamente 3 (BM25 satura), mas tem de ser claramente > 1.
	if r := noTitulo / noCorpo; r < 1.5 {
		t.Errorf("razao titulo/corpo = %.2f; peso 3x deveria separar bem mais", r)
	}
}
```

#### Verificações além dos passos

- Existe teste que reprova se `k1` mudar? E se `b` mudar? E cada um dos três pesos? **Liste os cinco com o nome do teste ao lado.**
- A forma crua pontua acima da reduzida para a mesma consulta? Prove com um caso.
- Um termo que aparece em **todas** as notas ainda ordena de forma útil? (É o caso do termo de arte frequente, e é por isso que não há stopwords.)
- Nota vazia ou sem o termo produz score zero, e não `NaN`? Divisão por comprimento médio com zero notas é o caminho para `NaN`, e `NaN` ordena de forma imprevisível.
- Determinismo: duas notas com score idêntico saem sempre na mesma ordem? Se o desempate for a ordem do mapa, não é.

**Prova de mutação obrigatória, cinco vezes:** mude `k1`, `b`, e cada um dos três pesos, um de cada vez, e confirme que um teste nomeado reprova em cada caso. **Cinco saídas coladas.** Uma constante que sobrevive à mutação está escrita, não verificada.

#### Regras de execução e contrato de relatório

Idênticos aos da Task 43. Relatório em `.superpowers/sdd/task-46-report.md`, com **a tabela das cinco constantes**, uma linha por constante, dizendo o que foi mutado e qual teste reprovou por nome.

**Files:** Create `internal/search/bm25.go`, `internal/search/bm25_test.go`
**Commit:** `feat(search): BM25 ranking with field weights`

---

