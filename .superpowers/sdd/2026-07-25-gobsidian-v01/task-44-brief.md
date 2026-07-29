### Task 44: `internal/search/analyzer.go` — normalização, tokenização e indexação dupla

RF-23, P0. É a base de tudo que vem depois, e o lugar onde uma escolha errada só aparece na qualidade do resultado.

#### Onde isto encaixa

Primeira peça do pacote `internal/search`, que não existe ainda. Ele é **folha**: não importa `index`, não importa `service`, não importa o SDK. Recebe texto, devolve tokens. Isso é o que o torna testável sem construir cofre nenhum, e o que impede a busca de virar dependência circular do índice.

#### O que já está fechado e vincula esta tarefa

- **Indexação dupla (D9).** Cada token produz a **forma crua normalizada** e, quando a redução produz algo diferente, também a **forma reduzida**. Ambas apontam para a mesma posting list. Nunca só a reduzida.
- **Normalização = remoção de acento + case folding.** `usucapiao` tem de encontrar `usucapião`; `Prescrição` tem de encontrar `prescrição`. `golang.org/x/text` já está fixado no `go.mod` **sem importador** — esta é a primeira tarefa a importá-lo. Se o build reclamar de `go.sum`, use `go get <caminho-do-pacote>@<versão-fixada>`; **`go mod tidy` está proibido**.
- **Redução deliberadamente conservadora:** plurais regulares e os sufixos verbais mais comuns. **Não é um stemmer completo e não deve virar um.** Se você se pegar acrescentando a décima regra, pare e reporte.
- **Sem remoção de stopwords.** Está prometido em `docs/TOOLS.md` ao chamador.
- **`ctx` não entra aqui.** Normalizar bytes em memória não bloqueia. Um `ctx` que nenhum corpo verifica ensina revisor a ignorar `ctx`.

#### A decisão que esta tarefa precisa tomar certo

**A posição do token é parte do token.** A Task 47 recorta trechos e a busca por frase (RF-24) casa sequência — as duas precisam do **offset em bytes** de cada ocorrência, relativo ao mesmo corpo que o parser usou. Emitir só a string do token torna as duas impossíveis sem reanalisar o texto.

```go
// Token e uma ocorrencia, nao uma palavra. Start e End sao offsets em BYTES
// no mesmo corpo que o parser indexou — o corpo DEPOIS de vault.StripBOM,
// como Heading.Start e Block.Start. Emitir so a forma perderia a posicao, e
// a Task 47 (trecho) e a busca por frase precisam dela.
type Token struct {
	Raw     string // forma crua normalizada: sem acento, minuscula
	Reduced string // forma reduzida; vazio quando a reducao nao muda nada
	Start   int64
	End     int64
}
```

`Reduced` vazio significa "a redução não produziu nada diferente" — **não** significa "não há forma reduzida". Distinga os dois, porque quem indexa vai iterar sobre isso e um zero-value ambíguo vira posting list duplicada.

#### Armadilhas já pagas que se aplicam

- **Offset em bytes, não em runes.** Português tem acento; `len(string)` em Go é bytes e `range` sobre string anda por rune. Misturar os dois produz trecho recortado no meio de um caractere — e o defeito só aparece em palavra acentuada, que é a maioria neste cofre.
- **Pergunte o que um valor zero significa.** `Start` zero é offset legítimo (primeiro byte). Não use zero como "não sei".
- **Um teste que não pode falhar.** Um teste de normalização que só afirma `len(tokens) > 0` passa com qualquer implementação.

#### O teste que sustenta a tarefa

```go
func TestAnalyzerDualIndexing(t *testing.T) {
	// "prescricoes" reduz para "prescricao"; "civil" nao reduz. As duas
	// formas do primeiro tem de sair, e a do segundo tem de sair com
	// Reduced vazio — nao com Reduced == Raw, que dobraria a posting list.
	toks := search.Analyze("Prescrições civil")

	if len(toks) != 2 {
		t.Fatalf("tokens = %d, quer 2: %+v", len(toks), toks)
	}

	if toks[0].Raw != "prescricoes" {
		t.Errorf("Raw = %q, quer %q — acento e caixa nao foram normalizados",
			toks[0].Raw, "prescricoes")
	}
	if toks[0].Reduced == "" || toks[0].Reduced == toks[0].Raw {
		t.Errorf("Reduced = %q; plural regular tem de reduzir e diferir da forma crua",
			toks[0].Reduced)
	}
	if toks[1].Reduced != "" {
		t.Errorf("Reduced = %q para termo que nao reduz; quer vazio, senao a "+
			"posting list recebe o mesmo termo duas vezes", toks[1].Reduced)
	}

	// Offsets em BYTES sobre o texto original, com acento. "Prescrições"
	// tem 11 runes e 13 bytes; um offset em runes recortaria no meio de "ç".
	if got := "Prescrições"[toks[0].Start:toks[0].End]; got != "Prescrições" {
		t.Errorf("fatia por offset = %q, quer o token inteiro — offset em rune, nao byte", got)
	}
}
```

#### Verificações além dos passos

Reporte o **resultado real** de cada uma:

- `usucapiao` e `usucapião` produzem o mesmo termo indexável? E `USUCAPIÃO`?
- Quantas regras de redução você implementou? Liste todas. Se passar de dez, pare e reporte.
- Um termo de arte que só se distingue pelo sufixo — teste com um par real do domínio jurídico — continua distinto depois da redução? **É o critério que justifica a indexação dupla existir.**
- Token com hífen, número, e `Art.` com ponto: como são tokenizados? Decida, documente, teste.
- Texto com BOM já removido por `vault.StripBOM` produz offsets alinhados com o corpo que o parser indexou?
- Quantas alocações por 10.000 tokens? Um número medido, ou **"não medido"**.

**Prova de mutação obrigatória:** desligue a emissão da forma reduzida e confirme que `TestAnalyzerDualIndexing` reprova nomeando o campo.

#### Regras de execução

Idênticas às da Task 43, incluindo `golangci-lint run ./internal/... ./cmd/...` antes do commit.

#### Contrato de relatório

`.superpowers/sdd/task-44-report.md`, no formato do contrato: RED e GREEN com saída, prova de mutação com saída, tabela de verificações com resultado real, a **lista completa das regras de redução**, e o número de alocações ou "não medido".

**Files:** Create `internal/search/analyzer.go`, `internal/search/analyzer_test.go`
**Interfaces:** Produces `search.Token`, `search.Analyze(string) []Token`
**Commit:** `feat(search): portuguese analyzer with dual indexing`

---

