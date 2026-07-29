### Task 43: Frontmatter — delimitador de fechamento com espaço no fim

Minor diferido do M1, promovido porque a busca o transforma em perda silenciosa de resultado.

#### Onde isto encaixa

Primeira tarefa do M3, e independente das outras sete. `internal/parser/frontmatter.go` está congelado por 48 golden files desde a Task 18; esta tarefa muda uma regra de aceitação e por isso mexe no corpus.

#### A evidência do defeito

`internal/parser/frontmatter.go:25-27` e `:43` — um espaço no fim da linha delimitadora **de fechamento** faz o parser cair no caminho "bloco não fechado", e o **YAML inteiro vira corpo**. Não é erro: é silêncio. A nota perde tags, aliases e todo campo de frontmatter, e continua sendo indexada como se não tivesse nenhum.

Obsidian e `gray-matter` toleram o espaço, então o arquivo é válido para o usuário e inválido só para nós. Ninguém edita frontmatter olhando espaço em fim de linha.

**Por que deixa de ser Minor no M3:** a busca filtra por tag e por campo de frontmatter (RF-21). Uma nota que perde metadados em silêncio deixa de ser encontrável por filtro, e o sintoma que chega ao usuário é *"a busca não acha"*, não *"o parser falhou"*. É perda de dado P0 com etiqueta de Minor.

#### O que implementar

Aceite espaço em branco à direita **na linha delimitadora**, de abertura e de fechamento. Nada mais muda: o delimitador continua sendo exatamente três hifens, ainda tem de estar na coluna 1, e a abertura ainda tem de ser a primeira linha do arquivo (depois do BOM, que `vault.StripBOM` já removeu).

**O offset do corpo tem de continuar exato.** `frontmatter.go` devolve o offset onde o corpo começa, e todo `Heading.Start` e `Block.Start` do parser é relativo a ele. Aceitar o espaço muda o comprimento da linha de fechamento — se o offset for calculado por soma de constantes em vez de pela posição real do fim da linha, ele passa a errar por exatamente o número de espaços. **Isto é o defeito de offset que este projeto já pagou duas vezes.** Invoque a skill `preventing-false-pass-and-offset-bugs` antes de escrever.

#### Armadilhas já pagas que se aplicam

- **`-update` de golden grava o que o código produz, não o que está certo.** O corpus vai mudar. Depois de gerar, **leia cada `.json` alterado** e confira contra o que você esperava **antes** de rodar. Aceitar a saída sem ler transforma a suíte na tautologia que fixa o bug de hoje como contrato de amanhã.
- **Um teste que não pode falhar.** Um golden novo com espaço no fim passa com o código velho se a asserção for só "parseou". Afirme os **campos** do frontmatter.

#### O teste que esta tarefa precisa

```go
func TestFrontmatterClosingDelimiterWithTrailingSpace(t *testing.T) {
	// O espaco depois dos tres hifens de FECHAMENTO e o defeito: sem a
	// correcao, o parser nao acha o fechamento, o bloco inteiro vira corpo,
	// e a nota perde tags e aliases em silencio.
	raw := []byte("---\ntitle: Prescricao\ntags: [civil]\n--- \n\n# Corpo\n\ntexto\n")

	note, err := parser.Parse(raw)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if len(note.Tags) != 1 || note.Tags[0] != "civil" {
		t.Fatalf("tags = %v, quer [civil] — o bloco YAML virou corpo", note.Tags)
	}
	if note.Frontmatter == nil || note.Frontmatter["title"] != "Prescricao" {
		t.Errorf("frontmatter = %v, quer title=Prescricao", note.Frontmatter)
	}

	// O offset do corpo tem de apontar para depois da linha de fechamento
	// INTEIRA, espaco incluso. Somar um comprimento constante erra por um.
	corpo := string(raw[note.BodyOffset:])
	if !strings.HasPrefix(corpo, "\n# Corpo") {
		t.Errorf("BodyOffset=%d aponta para %q; quer o inicio do corpo real",
			note.BodyOffset, corpo[:min(20, len(corpo))])
	}
}
```

Ajuste os nomes de campo à API real de `parser.Note` — o teste acima é o desenho, não a assinatura.

#### Verificações além dos passos

- Espaço no fim da linha de **abertura** também é aceito? Teste os dois.
- Tabulação no fim é aceita, ou só espaço? **Decida, implemente, e diga qual escolheu.**
- `--- x` (com conteúdo depois) continua **rejeitado**? Aceitar espaço não pode virar aceitar qualquer coisa.
- Uma nota **com BOM** e com espaço no delimitador ainda produz offset certo? É a composição de `vault.StripBOM` com `parser.Parse`, e o golden `edge/bom.md` existe justamente porque esse seam nunca foi testado dos dois lados juntos.
- Quantos goldens mudaram? Liste-os e diga o que mudou em cada um.

**Prova de mutação obrigatória:** reverta a aceitação do espaço, confirme que `TestFrontmatterClosingDelimiterWithTrailingSpace` reprova, restaure.

```bash
pwsh -File scripts/mutate.ps1 -Path internal/parser/frontmatter.go `
  -Anchor '<a linha que apara o espaco>' -Replacement '<a forma antiga>' `
  -Test TestFrontmatterClosingDelimiterWithTrailingSpace -Package ./internal/parser/
```

#### Regras de execução

- **O plano é a fonte.** Se um teste falhar por motivo que esta seção não explica, **pare e reporte**.
- **Nunca rode `git checkout`, `git restore`, `git stash`, `git clean` ou `git reset`.** **`go mod tidy` está proibido.**
- **Verde obrigatório antes do commit:** `pwsh -File scripts/verify.ps1` **e** `golangci-lint run ./internal/... ./cmd/...`. As Tasks 33–42 fecharam com 22 achados de lint porque ninguém rodou o linter — `go vet` não pega `errcheck` e o `verify.ps1` não roda `golangci-lint`.
- Commits em Conventional Commits, em inglês. Sem `helpers.go`, `utils.go`, `common.go`.

#### Contrato de relatório

`.superpowers/sdd/task-43-report.md`: o que implementou; RED e GREEN com comando e saída; a prova de mutação com saída colada; a tabela de verificações com resultado real; **a lista de goldens alterados e o que mudou em cada um, com a confirmação de que você leu os `.json` gerados**; arquivos alterados; preocupações.

Responda com no máximo 15 linhas.

**Files:** Modify `internal/parser/frontmatter.go`, `internal/parser/frontmatter_test.go`, goldens afetados em `testdata/parser/`
**Commit:** `fix(parser): accept trailing whitespace on the frontmatter delimiter`

---

