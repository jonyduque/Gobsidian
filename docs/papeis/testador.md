# Papel: testador

Você vai escrever ou avaliar testes.

**A premissa deste documento:** neste projeto, teste que não pode falhar já
custou mais caro que teste ausente — porque reporta cobertura que não existe, e
uma revisão aprovou justamente por causa dele.

---

## Seis testes reais que não podiam falhar

1. O teste de paridade passava **com referência vazia**: o guard checava se o
   diretório existia, e ele existia vazio.
2. `TestBuildBOM` afirmava que o heading *existia* e nunca o offset — enquanto o
   offset estava errado em 3 bytes.
3. As fixtures de exclusão usavam extensões que o filtro descartaria de qualquer
   jeito.

E o mais caro: `TestOverflowReconciliationFull` injetava overflow **com o
watcher ativo**; os eventos comuns aplicavam as três mudanças, e a reconciliação
nunca era exercida. Removido o reconciliador inteiro, o teste passava em 2,8 s —
cobertura zero num requisito P0, através de uma revisão que o aprovou.

**Teste de fallback desconecta o caminho principal, ou não é teste de fallback.**

### Mais três, de 2026-08-28, e as três passavam por um confundidor

1. **`TestBM25WeightHeadings`** comparava `"## civil / texto texto"` contra
   `"# Nota C / texto civil"`. As duas notas tinham **comprimentos diferentes**, e
   o BM25 normaliza por comprimento: a mais curta já pontuava mais **sem peso
   nenhum de heading**. Uma mutação que desligava a detecção inteira deixou o
   teste passando. Com o fixture equilibrado, sob a mesma mutação os dois scores
   saem idênticos — 0,2228 cada.
2. **O teste de peso de título** comparava uma nota titulada com o termo contra
   uma neutra. Mas **o frontmatter é tokenizado junto com o corpo**: um título
   "Ar puro" injeta uma ocorrência de "ar" nos tokens do documento, e a nota
   pontua mais mesmo com a regra de peso apagada. A regra teve de ir para um
   teste de UNIDADE; o de integração ficou só com a metade que ele consegue
   isolar.
3. **O teste do handshake de `max_results`** verificava a RECUSA por divergência.
   Com o campo deixando de ser lido, ele vinha como zero — que continua divergindo
   do que a ponte pediu, e a recusa continuava acontecendo. **Só o contrapeso**, o
   caso em que os dois valores são iguais e a conexão tem de ser ACEITA, pegava a
   mutação.

**O padrão comum:** o teste media a coisa certa por um caminho que outro fator já
determinava. Quando um teste passa, pergunte qual OUTRA diferença entre os dois
lados do cenário poderia explicar o resultado sozinha — e apague essa diferença.

---

## A prova de mutação

Regra que sobrevive a mutação não está verificada, está **escrita**. Na Task 13,
sete regras do módulo sobreviviam a mutantes com a suíte verde — inclusive a que
o comentário do próprio fix defendia. Ler o teste não acha isso.

```bash
pwsh -File scripts/mutate.ps1 -Path internal/watcher/apply.go `
  -Anchor 'if n, ok := idx.Get(path); ok {' `
  -Replacement 'if n, ok := idx.Get(path); ok && false {' `
  -Test TestApply -Package ./internal/watcher/
```

**O código de saída é invertido de propósito:**

| Saída | Significa | O que fazer |
|---|---|---|
| `0` | O teste **reprovou** sob mutação | É o que você quer. Cole a saída no relatório. |
| `1` | O teste **passou** sob mutação | A regra está escrita, não verificada. **Escreva o teste que falta.** |
| `2` | Inconclusivo | Âncora ambígua, ou a mutação quebrou o build. Falha de compilação **não é cobertura**. |

O script exige âncora com ocorrência única, restaura em `finally` conferindo por
SHA-256, e trata falha de compilação como inconclusivo. Se `-Anchor` não casar
ele sai `2` — **copie o texto do arquivo, não digite de memória**.

**Prova de mutação escrita no condicional não é prova.** *"Se removermos X, o
teste falharia"* apareceu em dois relatórios, e uma das duas estava
factualmente errada — a regra foi removida e a suíte continuou verde. O tempo
verbal é o sinal: prova real está no **passado** e traz a saída colada.

Detalhe adicional na skill `mutation-proof-discipline`.

---

## Antes de dizer que testou

**Apague a regra, rode, confirme que um teste nomeia a falha, restaure.** Se nada
falhar, você escreveu a regra, não a verificou.

Outras armadilhas de teste que já ocorreram:

- **`-update` de golden grava o que o código produz, não o que está certo.**
  Aceitar a saída sem ler transforma a suíte em tautologia que fixa o bug de hoje
  como contrato de amanhã. Depois de gerar, leia cada `.json` e confira contra o
  que você esperava **antes** de rodar.
- **Asserção de tempo atrás de build tag `//go:build race`, em arquivo
  separado.** O detector multiplica latência por 2×–6×; teto cobrado sob `-race`
  reprova por motivo que não é o do produto.
- **Função que lê `os.Stdin` direto não é testável de forma determinística.**
  `servePonteRemota` fazia isso: sob `go test` no Linux o stdin é `/dev/null` e
  devolve EOF na hora, então o mesmo commit ficava verde no Windows e no macOS e
  vermelho no ubuntu. Passe `stdin` e `stdout` como parâmetros — e o ganho não é
  só estabilidade: com o stdout na mão, o teste passou a conferir os bytes que
  atravessam.
- **Teste sob carga paralela pode estourar prazo sem que nada esteja errado.**
  `TestPastaQueChegaComArquivosDentro` passa isolado em 2,6 s e já estourou 60 s
  dentro de `go test -race ./...`. Antes de declarar regressão, rode isolado.

---

## Regra de gate

Os gates deste projeto — `scripts/pre_commit_docs.ps1`, `scripts/audit_reports.ps1`
— são código que decide `allow`/`deny` sobre texto, e mentem do mesmo jeito que
um teste que não pode falhar: aceitando o que deviam recusar sem que nada
avise. Em 2026-09-07 a revisão final de um hook que já "tinha teste" achou
sete achados só nele e no auditor — três deles bypasses inteiros (comentário
de shell, commit encadeado, `--amend`). Cada regra de gate vem, no mesmo commit, com **três casos em
`scripts/check_gates.ps1`**:

1. **O que a regra deve recusar** — a entrada que motivou a regra, literal
   (`git commit -F msg # was: -m "wip [sem-doc]"`).
2. **O que a regra deve aceitar** — a entrada legítima mais parecida com a
   recusada (`git commit -m "fix: issue #12 [sem-doc]"`, o `#` dentro de
   aspas).
3. **O mecanismo inverso** — a entrada que a implementação ingênua da regra
   quebraria. Remover cerca de código para não contar `# comentário` como
   cabeçalho é a regra; cerca **nunca fechada** engolindo os cabeçalhos reais
   que vêm depois é o inverso, e foi o achado N2.

Os três usam a mesma fixture mínima que faz o `allow` só poder vir da regra
sob teste (um `.go` em stage sem doc, para o hook). Regra com um caso só
tem o caso que o autor imaginou; é o terceiro que pega o que ele não
imaginou.

**Todo ramo do hook é alcançável por `-Simular`.** A exceção de `--amend` não
era: vivia fora do caminho simulado, `check_gates` não a exercitava, e ela
era um bypass inteiro (`git commit --amend -m "..."` com `.go` em stage e doc
nenhuma → `allow`). Ramo que só roda quando o Claude Code chama o hook de
verdade é ramo sem teste. Se a simulação não alcança, a simulação está
incompleta — não o caso.

**Prova de mutação de gate é a mesma dos testes de Go:** volte a linha antiga
(`$Filter = 'task-*-report.md'`), rode `check_gates.ps1`, cole os casos que
reprovaram **pelo nome**, restaure, rode de novo. Medido em 2026-09-07 para o
glob do auditor:

```
[!] -Task final-fix acha final-fix-report.md -> 0 SECAO-AUSENTE: esperado '0', obtido 'erro'
[!] sem -Task, todos os cinco *-report.md da fixture sao vistos: esperado '5', obtido '4'
[!] check_gates: 2 de 26 casos reprovados
```

Ver também [`../ARMADILHAS.md`](../ARMADILHAS.md), que traz o mecanismo de cada
defeito histórico — vários deles só são testáveis se você souber como montar a
condição (por exemplo: `FILE_ATTRIBUTE_OFFLINE` é gravável, e é assim que se
simula um placeholder de nuvem).

---

## Handle exclusivo e placeholder de nuvem: `internal/vaulttest`

As condições de ambiente que só o sistema operacional cria — um arquivo que
**não pode ser aberto**, um diretório que **não pode ser listado**, um
placeholder de nuvem — vêm de `internal/vaulttest`, e de nenhum outro lugar:

| Precisa de… | Chame |
|---|---|
| Arquivo que `os.ReadFile` não abre | `vaulttest.TravarExclusivo(t, abs)` |
| Diretório que `os.ReadDir` não lista | `vaulttest.TravarDiretorioExclusivo(t, abs)` |
| Placeholder somente-nuvem | `vaulttest.MarcarSomenteNuvem(t, abs)` |
| Prazo de espera de algo assíncrono | `vaulttest.Prazo` |

Fora do Windows os três primeiros fazem `t.Skip` com o motivo — share mode e
`FILE_ATTRIBUTE_OFFLINE` são semântica do NTFS.

**A regra que o pacote existe para impor: o helper prova a condição antes de
devolver, e por isso a asserção do chamador é incondicional.** `TravarExclusivo`
confere que a leitura de fato falha; `MarcarSomenteNuvem` confere que
`vault.IsCloudOnly` de fato responde verdadeiro. Nunca escreva
`if origemExiste && destinoExiste && err == nil { …asserções… }`: uma guarda
assim faz o teste passar em silêncio exatamente quando o cenário não se montou,
que é quando ele mais precisava falhar.

Isso veio de um defeito medido. Na base `86f07e5`,
`git grep -n 'CreateFile(' -- '*_test.go'` devolvia **oito** sítios: cinco
helpers nomeados e três blocos inline dentro do próprio teste. Desses oito,
**quatro** conferiam que a trava travava — três com `t.Fatal`
(`classify_cloudonly`, `cloudonly_update`, `cloudonly_replace`) e um com
`t.Skip` (`walk_raiz`, que desistia do cenário em vez de acusar). Entre os
**cinco helpers nomeados**, só o `travarExclusivo` de `internal/search`
conferia e falhava; o `travarDiretorioExclusivo` de `internal/vault` conferia e
pulava, e os outros três não conferiam nada. Um handle que pede só
`GENERIC_READ` não barra o `os.ReadFile` — precisa de
`GENERIC_READ|GENERIC_WRITE` com `dwShareMode = 0` —, e os helpers que não
conferiam sustentavam duas asserções condicionais em `internal/service`.

Cópia local nova de `windows.CreateFile` ou de `SetFileAttributes` em `_test.go`
é regressão desta tarefa. A única exceção é `internal/vault/cloudonly_info_windows_test.go`,
que é `package vault` (interno) e não pode importar `vaulttest` sem ciclo.

---

## Onde os testes moram

- Testes em tabela; golden files com `-update` (**regenerar e olhar são passos
  diferentes**).
- `testdata/parser/` — 48 golden files do parser e das quatro extensões.
- `testdata/parity/` — corpus de paridade contra o `metadataCache` real do
  Obsidian.
- `testdata/vault_small/` — cofre de fixture.
- Cofre sintético de benchmark: `scripts/gen_vault.ps1 -Notes 5000 -Seed 42`.
  Sem cofre, o benchmark **pula**, e o comparador trata benchmark ausente como
  erro — um bench que mede corpus vazio reporta número ótimo e some com a
  regressão que existia para pegar.

A skill `preventing-false-pass-and-offset-bugs` cobre offset e falso-PASS em
detalhe.
