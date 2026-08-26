# Papel: testador

Você vai escrever ou avaliar testes.

**A premissa deste documento:** neste projeto, teste que não pode falhar já
custou mais caro que teste ausente — porque reporta cobertura que não existe, e
uma revisão aprovou justamente por causa dele.

---

## Três testes reais que não podiam falhar

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

Ver também [`../ARMADILHAS.md`](../ARMADILHAS.md), que traz o mecanismo de cada
defeito histórico — vários deles só são testáveis se você souber como montar a
condição (por exemplo: `FILE_ATTRIBUTE_OFFLINE` é gravável, e é assim que se
simula um placeholder de nuvem).

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
