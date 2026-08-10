# Task 88 — Índice de busca carregado sob demanda

**Tier: modelo barato.** O corpo do teste difícil está escrito abaixo.

#### Onde encaixa
Primeira da Parte II, e independente de todas as outras. É a que dá mais
resultado por linha mexida.

#### O que vincula esta tarefa

Repetido aqui de propósito: o brief é a unidade que viaja, e decisão citada por
código fica no preâmbulo, que não viaja com ela.

- **Medição com n maior ou igual a 3, uma mudança por vez.** Sem ganho medido, a
  mudança é revertida: código mais feio sem ganho é dívida pura.
- **`vault_search` responde `INDEX_BUILDING`, nunca lista vazia,** enquanto o
  índice não cobre o cofre. "Ainda não sei" e "não achei nada" pedem ações
  diferentes de quem chama.
- **Nenhum teto de RNF é afrouxado nesta batelada.**

#### A evidência medida do defeito
`cmd/gobsidian/serve.go:407` chama `prepararIndiceDeBusca` **incondicionalmente**,
numa goroutine, em toda partida. Uma sessão que nunca chama `vault_search` paga
o índice inteiro assim mesmo — e a maioria das sessões de assistente lê e
escreve nota sem nunca buscar. RSS em repouso hoje: 381,5 MB.

#### A decisão que esta tarefa tem de acertar
O carregamento passa a ser disparado pela **primeira chamada de
`vault_search`**, uma vez só. Até lá o índice fica marcado como em construção e
a tool responde `INDEX_BUILDING`.

**Pré-decidido:** a flag `--eager-search` liga o comportamento antigo, e o
padrão é preguiçoso. Quem roda o servidor num script que só busca quer o
carregamento adiantado; quem o roda como MCP quase nunca quer.

**O watcher continua começando na partida.** Só o carregamento do índice de
busca é adiado. Adiar o watcher faria eventos se perderem, e o único anteparo
seria a reindexação no boot seguinte.

#### Armadilhas já pagas que se aplicam
- **Teste de fallback que deixa o caminho principal ligado mede o caminho
  principal.**
- **`sync.Once` que envolve a chamada errada** carrega o erro para sempre: se a
  carga falhar, a próxima busca precisa poder tentar de novo. O `Once` é sobre
  "já disparei", não sobre "já consegui".

#### O corpo do teste que não é óbvio
```go
// TestBuscaPreguicosaCarregaUmaVezESoUmaVez guarda os dois defeitos que o
// adiamento introduz: carregar N vezes sob concorrencia, e nunca mais tentar
// depois de uma falha.
func TestBuscaPreguicosaCarregaUmaVezESoUmaVez(t *testing.T) {
	var cargas atomic.Int32
	svc := servicoComCargaPreguicosa(t, func() error {
		cargas.Add(1)
		return nil
	})

	// Vinte buscas concorrentes: uma carga, nao vinte.
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = svc.Search(context.Background(), service.SearchOptions{Query: "x"})
		}()
	}
	wg.Wait()
	if got := cargas.Load(); got != 1 {
		t.Errorf("carregou %d vezes sob concorrencia, quer 1", got)
	}

	// Falha na carga nao pode ser definitiva.
	var tentativas atomic.Int32
	svc2 := servicoComCargaPreguicosa(t, func() error {
		if tentativas.Add(1) == 1 {
			return errors.New("falha transitoria")
		}
		return nil
	})
	if _, err := svc2.Search(context.Background(), service.SearchOptions{Query: "x"}); err == nil {
		t.Fatal("primeira busca deveria propagar a falha da carga")
	}
	if _, err := svc2.Search(context.Background(), service.SearchOptions{Query: "x"}); err != nil {
		t.Errorf("segunda busca falhou (%v): o Once travou o erro para sempre", err)
	}
}
```

#### Verificações além dos passos
- RSS em repouso de uma instância que **nunca buscou**, medido em três partidas
  no cofre real, contra os 381,5 MB de hoje. É o número que justifica a tarefa.
- Tempo até a primeira busca responder — a carga entra nele agora. Se passar de
  3 s, dizer o número; não afrouxar nada.
- Golden da Task 78 idêntico.

#### Regras de execução
Rodar `pwsh -File scripts/verify.ps1` antes de dizer que acabou. Registrar no
ledger antes de reportar conclusão. Escopo não encolhe em silêncio: `BLOCKED`
com motivo é resposta melhor que entrega que parece completa.

#### Prova de mutação
```
pwsh -File scripts/mutate.ps1 -Path internal/service/search.go `
  -Anchor 'if err := s.garanteIndiceDeBusca(ctx); err != nil {' `
  -Replacement 'if err := error(nil); err != nil {' `
  -Test TestBuscaPreguicosaCarregaUmaVezESoUmaVez -Package ./internal/service/
```

#### Contrato de relatório
RSS em repouso antes e depois, três partidas cada. Tempo até a primeira busca.
Golden inalterado. Saída do `mutate.ps1` colada.

**Files:** `cmd/gobsidian/serve.go`, `internal/service/search.go`,
`internal/config/`, testes
**Commit:** `perf(serve): load the search index on first search, not at boot`

---

