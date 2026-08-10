# Task 81 — Título normalizado calculado na indexação, não por posição

**Tier: modelo barato.**

#### Onde encaixa
Depois da Task 80. Mede-se contra a baseline que ela deixou.

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
`bm25.go:170`, dentro do laço que roda **para cada posição de cada posting**:
```go
normTitle := Normalize(n.Title)
```
`CalculateBM25` cum = 1094,74 MB no perfil, ~137 MB por busca. O título não muda
entre posições da mesma nota.

#### A decisão que esta tarefa tem de acertar
`Note` ganha `TitleNorm string`, preenchido onde `Title` é preenchido.
`getFieldWeight` lê o campo.

**Uma função só produz a chave** (armadilha do `aliasKey`): `TitleNorm` é
escrito num lugar — o construtor de `Note` — e **nunca** recalculado no caminho
de leitura. `MoveNote` copia a struct, então o campo viaja junto; conferir isso
explicitamente, porque `MoveNote` já entrou fora do contrato uma vez.

#### O corpo do teste que não é óbvio
```go
// TestTitleNormAcompanhaOTitulo guarda a divergencia que um campo derivado
// SEMPRE convida: Title muda num caminho e TitleNorm nao muda no outro.
//
// Cobre os tres caminhos que publicam Note: Build, Replace e MoveNote.
func TestTitleNormAcompanhaOTitulo(t *testing.T) {
	root := t.TempDir()
	escreve(t, root, "a.md", "# Prescrição Intercorrente\n\ncorpo\n")
	v, idx := cofreIndexado(t, root)

	confere := func(quando string, p vault.CanonicalPath) {
		t.Helper()
		n, ok := idx.Get(p)
		if !ok {
			t.Fatalf("%s: nota %q sumiu do indice", quando, p)
		}
		if quer := search.Normalize(n.Title); n.TitleNorm != quer {
			t.Errorf("%s: TitleNorm=%q, quer %q (Title=%q)",
				quando, n.TitleNorm, quer, n.Title)
		}
	}
	confere("apos Build", "a.md")

	escreve(t, root, "a.md", "# Execução Fiscal\n\ncorpo\n")
	if err := idx.Replace(context.Background(), v, "a.md"); err != nil {
		t.Fatal(err)
	}
	confere("apos Replace", "a.md")

	if err := os.Rename(filepath.Join(root, "a.md"), filepath.Join(root, "b.md")); err != nil {
		t.Fatal(err)
	}
	idx.MoveNote(v, "a.md", "b.md")
	confere("apos MoveNote", "b.md")
}
```

#### Verificações além dos passos
Golden da Task 78 idêntico. `Normalize` de um título acentuado tem de dar o
mesmo antes e depois — é o caso que o golden `com-acento` cobre.

#### Prova de mutação
```
pwsh -File scripts/mutate.ps1 -Path internal/index/note.go `
  -Anchor 'TitleNorm: search.Normalize(title)' -Replacement 'TitleNorm: title' `
  -Test TestTitleNormAcompanhaOTitulo -Package ./internal/index/
```

#### Contrato de relatório
`benchstat` de `BenchmarkSearchLimit200` e `BenchmarkSearchTermoAmplo`.
Perfil `alloc_space` depois, mostrando `transform.Chain` fora do topo.

**Files:** `internal/index/note.go`, `internal/index/build.go`,
`internal/index/update.go`, `internal/search/bm25.go`, testes
**Commit:** `perf(search): precompute the normalized title at index time`

---

