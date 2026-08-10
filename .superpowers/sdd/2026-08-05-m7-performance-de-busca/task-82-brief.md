# Task 82 — `avgdl` em cache, invalidado por geração

**Tier: modelo barato.**

#### Onde encaixa
Depois da 81. Toca `bm25.go`, como a 81 — sequencial, nunca em paralelo.

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
`bm25.go:83`, por busca:
```go
for _, p := range idx.Paths() {
    if dl := ix.DocLength(string(p)); dl > 0 { sumDocLen += float64(dl) }
}
```
`idx.Paths()` aloca fatia de N elementos **e ordena** (`index.go`, `slices.Sort`).
Depois são N chamadas a `DocLength`, **cada uma pegando RLock**. Tudo para uma
constante do corpus.

#### A decisão que esta tarefa tem de acertar
O cache é chaveado por `idx.Generation()`, que **já existe** e sobe a cada
mutação. Guardar `(geracao, avgdl)`; recalcular quando a geração diferir.

**O cache fica no `Inverted`, não em variável de pacote.** Estado global entre
cofres é como dois cofres passam a compartilhar `avgdl` — e o servidor pode
servir mais de um índice no mesmo processo em teste.

**`avgdl` obsoleto muda score sem erro.** É o modo de falha, e o golden não o
pega se a geração não mudar no teste: o teste tem de **forçar** a mudança.

#### O corpo do teste que não é óbvio
```go
// TestAvgdlInvalidaComAGeracao guarda o defeito que o cache INTRODUZ.
//
// avgdl obsoleto nao levanta erro: muda o score em silencio. O teste adiciona
// uma nota MUITO longa depois da primeira busca — se o cache nao invalidar, o
// avgdl continua o da corpus pequeno e a normalizacao por tamanho fica errada.
func TestAvgdlInvalidaComAGeracao(t *testing.T) {
	root := t.TempDir()
	escreve(t, root, "a.md", "# A\n\nprescricao\n")
	escreve(t, root, "b.md", "# B\n\nprescricao prescricao\n")
	svc, v, idx, inv := servicoCompleto(t, root)

	antes, err := svc.Search(context.Background(), service.SearchOptions{Query: "prescricao"})
	if err != nil {
		t.Fatal(err)
	}

	// Nota longa: muda avgdl de forma detectavel.
	longa := "# C\n\n" + strings.Repeat("palavra ", 5000) + "prescricao\n"
	escreve(t, root, "c.md", longa)
	if err := idx.Replace(context.Background(), v, "c.md"); err != nil {
		t.Fatal(err)
	}
	if err := inv.Update(context.Background(), v, "c.md"); err != nil {
		t.Fatal(err)
	}

	depois, err := svc.Search(context.Background(), service.SearchOptions{Query: "prescricao"})
	if err != nil {
		t.Fatal(err)
	}
	if len(antes.Results) == 0 || len(depois.Results) == 0 {
		t.Fatal("busca vazia nos dois lados nao prova nada")
	}
	if antes.Results[0].Score == depois.Results[0].Score {
		t.Errorf("score de %q identico (%.6f) depois de o corpus mudar de "+
			"tamanho: avgdl nao foi invalidado",
			antes.Results[0].Path, antes.Results[0].Score)
	}
}
```

#### Verificações além dos passos
- Golden da Task 78 idêntico.
- O teste acima rodado **duas vezes seguidas** no mesmo processo: cache que
  guarda a geração errada acerta na primeira e erra na segunda.
- `go test -race ./internal/search/`: o cache é lido e escrito sob o mesmo
  `RWMutex` do índice, e o detector é quem diz se ficou.

#### Prova de mutação
```
pwsh -File scripts/mutate.ps1 -Path internal/search/bm25.go `
  -Anchor 'if ix.avgdlGen == ger {' -Replacement 'if true {' `
  -Test TestAvgdlInvalidaComAGeracao -Package ./internal/search/
```

#### Contrato de relatório
`benchstat` das quatro `BenchmarkSearch*`. Dizer explicitamente quantas
aquisições de RLock por busca foram eliminadas (é N, com N = notas do cofre).

**Files:** `internal/search/bm25.go`, `internal/search/inverted.go`, testes
**Commit:** `perf(search): cache avgdl and invalidate it by index generation`

---

