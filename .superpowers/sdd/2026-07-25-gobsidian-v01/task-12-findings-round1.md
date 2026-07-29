# Task 12 — achados da revisão, rodada 1

Spec passou: os seis steps estão lá, `slug.go` e `frontmatter.go` são
byte-idênticos ao brief, `go.mod`/`go.sum` intocados, pacote é folha de
verdade (sem `vault`, sem `service`, sem SDK), e o revisor confirmou que cinco
das suas sete provas de mutação são sólidas e bem isoladas.

Sua decisão de acrescentar comentários de doc ao `types.go` foi **aprovada** e
verificada como forçada, não preferência: remover um deles reproduz o achado do
revive exatamente. Quatro dos seis carregam informação que o nome não dá.

O que reprovou é força de cobertura — e é a entrega desta task, porque o parser
é a folha em cima da qual todo o resto de M1 é construído.

**O plano já foi corrigido** (commit `a1ec366`): os dois testes defeituosos
nasceram no código verbatim dele, não na sua transcrição. Transcreva de lá.

---

## 1. Important (bloqueador) — `bodyOffset` é asserido em 1 dos 5 subtestes

`internal/parser/frontmatter_test.go:59`

```go
if tt.wantOffset != 0 && off != tt.wantOffset {
```

Só o caso `presente` define `wantOffset` não-zero, então o offset — o valor que
o próprio brief chama de "o que desloca todo heading, bloco e link do produto
inteiro se estiver errado" — é conferido uma vez em cinco.

O revisor aplicou esta mutação e **a suíte inteira passou**:

```go
if fmEnd == 0 {
	return data[firstNL+1 : firstNL+1+fmEnd], body, 0   // sobrevive
}
```

Cenário: nota `---\n---\n# Corpo\n`. O offset certo é 8; o mutado devolve 0.
Todo heading, bloco e link de toda nota com frontmatter vazio sai 8 bytes
deslocado, e `note_read` de uma seção devolve texto começando cedo demais.
Frontmatter vazio não é exótico: o Obsidian deixa `---\n---` para trás toda vez
que alguém adiciona uma propriedade e depois a apaga.

**Correção (já no plano):** tirar a guarda `!= 0` e declarar `wantOffset` nos
cinco casos — zero é a resposta **certa** em três deles, e é justamente por isso
que pular a asserção neles deixa passar o bug.

## 2. Important (bloqueador) — o teste de tipos carrega uma data e nunca a confere

`internal/parser/frontmatter_test.go:67`

O fixture termina em `data: 2026-07-25` e nada assere isso. O revisor confirmou
que `yaml.v3` de fato devolve `time.Time` ali, então há um tipo real a
proteger. Esta mutação **passa a suíte inteira**:

```go
for k, v := range out {
	if t, ok := v.(time.Time); ok {
		out[k] = t.Format("2006-01-02")   // sobrevive
	}
}
```

Cenário: em M3, uma consulta de metadados faz comparação de intervalo de datas
contra o que agora é string; a ordenação degrada silenciosamente para
lexicográfica e nenhum teste do repositório dispara.

Note que **esta é exatamente a garantia que a sua Mutação 7 alegava cobrir**. A
mutação "unmarshal devolve nil" zera *tudo*, então qualquer asserção a pega —
ela não distingue um decoder que preserva tipos de um que os colapsa. É o tipo
de prova fraca que este projeto já pagou caro para aprender a reconhecer.

**Correção (já no plano):** uma asserção `got["data"].(time.Time)`. O import de
`time` também já está no plano.

## 3. Contrato do BOM — documentar, não duplicar

`internal/parser/frontmatter.go:16`

`bytes.HasPrefix(data, fmDelim)` falha em `\xEF\xBB\xBF---`. Com BOM, o
frontmatter não fica malformado: fica **invisível**. `FrontmatterErr` não
dispara, tags/aliases/title somem em silêncio, e as linhas `---` viram conteúdo
da nota.

Isto **não** é para você consertar no parser. `internal/vault` já tem
`StripBOM` (`eol.go:54`), e duplicar a lógica criaria dois lugares tratando BOM
— exatamente o tipo de divergência que dá bug seis meses depois.

O que falta é o **contrato estar escrito**. Hoje nada no parser diz que ele
espera entrada já sem BOM, e a fiação parser↔vault só acontece na Task 19: quem
a escrever não tem como saber. Windows PowerShell 5.1 emite BOM por padrão em
`Out-File`/`Set-Content`, e este é um projeto que roda em Windows e distribui
scripts PowerShell.

**Correção:** um comentário em `SplitFrontmatter` dizendo que ela exige entrada
já sem BOM, nomeando `vault.StripBOM` como o produtor dessa garantia, e dizendo
o que acontece se alguém esquecer (frontmatter fica invisível, não malformado).
Comentário só — nenhuma mudança de comportamento, e o parser continua sem
importar `vault`.

---

## Fora do escopo desta rodada — NÃO corrija

Registrados no ledger para a revisão de marco decidir. Mexer neles agora torna
a rodada não-verificável:

- **Espaço em branco no fim da linha delimitadora** (`frontmatter.go:25-27`,
  `:43`) rejeita o bloco. No fechamento é pior: cai no caminho de "não fechado"
  e o **YAML inteiro vira corpo**. Obsidian e gray-matter toleram.
- **`Slug` funde headings distintos**: `"1.2 Escopo"` e `"12 Escopo"` colidem;
  `C++`/`C#`/`C` viram todos `"c"`. Deliberado no brief — é o que faz
  `Art. 1.234` funcionar — mas a resolução de link precisa decidir o desempate.
- **Headings só de símbolo** viram slug vazio; `## 🚀 Deploy` colide com
  `## Deploy`.
- `slug.go:37` — `!lastSpace && b.Len() > 0`: a segunda condição é implicada
  pela primeira. Vem do código verbatim do brief.
- Contrato nil-vs-vazio de `SplitFrontmatter`/`DecodeFrontmatter` não
  documentado.

---

## O que esta rodada precisa provar

Sem `-timeout` na linha de comando, e com a mutação que o teste **deveria**
pegar, não uma que apenas o deixe vermelho:

- **Achado 1:** a mutação `return ..., body, 0` no ramo de frontmatter vazio
  precisa reprovar, nomeando o subteste.
- **Achado 2:** a mutação que formata `time.Time` como string precisa reprovar.
- **Regressão:** `go test -race ./...` limpo, `gofmt -l .` vazio, e
  `golangci-lint` em zero nos alvos que você conseguir rodar.

Se alguma mutação **não** reprovar, diga isso e reestruture o teste — não
reporte verde. Foi essa a diferença entre as suas cinco provas boas e a
Mutação 7.
