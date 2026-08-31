---
title: Gates e scripts
type: reference
status: active
description: O que cada script em scripts/ verifica, e o que acontece quando ele não roda.
source_paths:
  - scripts/verify.ps1
  - scripts/test_orphans.ps1
  - scripts/mutate.ps1
  - scripts/audit_reports.ps1
  - scripts/measure.ps1
  - scripts/bench_compare.ps1
  - scripts/gen_vault.ps1
  - .github/workflows/ci.yml
  - .github/workflows/bench.yml
source_commit: f7de8e81
tags: [ci, gates, qualidade]
language: pt-BR
updated_at: '2026-08-31'
---

# Gates e scripts

`scripts/` tem 18 arquivos PowerShell. Estes são os que decidem se algo entra.

## `verify.ps1` — a bateria

Build, `go test -race`, vet nos três alvos, `gofmt`, `check_net`. Para no
primeiro erro. Existe porque a lista solta convida a rodar três dos cinco.

## `test_orphans.ps1` — o gate de release

`-Cycles 100` roda **quatro** cenários: `stdin-eof`, `parent-death`, `signal`,
`daemon-idle`. Cada um **reprova se o `reason=` não for o do mecanismo que ele
nomeia** — encerrar pelo motivo certo por acidente não conta.

Três armadilhas do próprio harness, todas pagas:

- **Ele não compila** — roda o que estiver em `bin/`. Hoje recusa binário mais
  velho que o código. A guarda fica **antes de qualquer despacho**: a primeira
  versão dela cobria três cenários e não o quarto, que é exatamente o defeito que
  ela existe para impedir.
- **`StreamReader.Peek()` bloqueia**, apesar do nome. Um ciclo ficou 15h44m parado
  por causa disso. Use `ReadLineAsync` com `Wait` limitado.
- **Órfão vazado e ciclo que não mediu são coisas diferentes.** Um commit só de
  documentação reprovou com 1 ciclo em 300 não medido. Hoje `-MaxNaoMedidosPct`
  (padrão 2) tolera uma fração, sempre impressa. Continuam reprovando com
  qualquer teto: vazamento real, `reason=` errado, e **zero ciclos medidos**.

## `mutate.ps1` — prova de cobertura

**Código de saída invertido de propósito:** `0` = o teste reprovou sob mutação
(regra verificada), `1` = o teste passou (regra escrita, não verificada), `2` =
inconclusivo (âncora ambígua, ou a mutação quebrou o build — falha de compilação
não é cobertura).

Exige âncora com ocorrência única, restaura em `finally` e confere o restauro por
SHA-256.

> **Regra que sobrevive a mutação não está verificada, está escrita.** Na Task 13,
> sete regras sobreviviam a mutantes com a suíte verde — inclusive a que o
> comentário do próprio fix defendia.

## `audit_reports.ps1` — contra evidência falsa

Procura hedge apresentado como medição, prova de mutação escrita no condicional
(o tempo verbal é o sinal: prova real está no passado e traz saída colada),
não-resposta do tipo "coberto implicitamente", SHA citado no ledger que não
existe, e tarefa completa sem relatório. Sai `1` com achados.

## `check_doc_refs.ps1` e `check_readme_anchors.ps1`

O primeiro acha token entre crases que parece identificador e não existe em `.go`
nenhum. O segundo confere que toda âncora do README resolve.

**Os dois entraram na bateria em 2026-08-11 porque até então não rodavam em lugar
nenhum** — nem no `verify.ps1`, nem no CI. Três seções do README ficaram sem link
por um marco inteiro por causa disso.

A dispensa do `check_doc_refs` é **por linha**, e o motivo é obrigatório:
`<!-- check-doc-refs: ignore <tokens> -- <motivo> -->`. Sem motivo vira
`DISPENSA-INVALIDA`. Lista global no topo do script dispensaria o token em *todo*
documento.

## `measure.ps1` — RNF-01 e RNF-07

Mede indexação a frio (do log `servidor pronto`) e **heap vivo** em dois estados
nomeados: `pronto` (índice de metadados carregado, nenhuma busca) e `servindo`
(uma `vault_search` já respondida, portanto o índice invertido carregado).
Encerra fechando o stdin, que é o caminho que um host MCP usa.

Existe porque uma versão de `docs/OPERACAO.md` trazia "ex: 408ms em teste local" e
"tende a ficar ~30-45 MB" — um exemplo e uma expectativa, apresentados como
resultado.

**Três correções de 2026-08-30, cada uma vinda de um número publicado errado:**

- **Mede heap vivo, não RSS.** O RSS do Go segue o **goal** do GC, que fica em
  torno de 2× o heap vivo do último ciclo — ele mede a política do coletor, não o
  volume de dado. O heap vivo sai do terceiro número de `A->B->C MB` no
  `gctrace`, que o script liga com `GODEBUG=gctrace=1` e não exige mudança
  nenhuma no produto.
- **Força `GOBSIDIAN_NO_DAEMON=1`.** Havendo daemon, o processo que o script
  media era a **ponte** (~15 MB) — não o servidor.
- **Confere `index_origin`.** Um número publicado saiu contaminado por uma
  execução que reconstruiu o índice em vez de carregar o cache; o script agora
  avisa.

> **`measure.ps1` não está em gate nenhum** — nem no `verify.ps1`, nem no CI. É a
> mesma forma de `check_doc_refs` antes de 2026-08-11. Segue registrado nas
> dívidas abertas de `docs/ESTADO.md`.

## `bench_compare.ps1` e `gen_vault.ps1`

`gen_vault.ps1 -Notes 5000 -Seed 42` gera um cofre sintético **determinístico**.
Sem cofre determinístico não há contra o que comparar: o corpus mudaria a cada
rodada e o delta mediria o corpus, não o código.

`bench_compare.ps1` compara contra `docs/bench-baseline.json` usando a **mediana**
de 5 amostras, com tolerância de 20%. Antes ele deixava a última amostra vencer em
silêncio, e o gate reprovou duas vezes sem mudança de código.

O JSON registra a máquina **e o toolchain** (`"ubuntu-latest ... go 1.25"`).
Comparar número de runner diferente não mede regressão — mede a diferença entre
os runners.

## O CI

`.github/workflows/ci.yml`: matriz de 3 SOs (vet + test -race), `fmt` (gofmt +
vet cruzado), `netcheck`, `lint` e `lint-windows` (sem o qual todo arquivo
`//go:build windows` ficava sem análise), e `orphans`.

`.github/workflows/bench.yml` roda num **único** runner, com o cofre sintético de
semente fixa, e compara contra `docs/bench-baseline.json`.

> O job `orphans` chama **três** dos quatro cenários. `daemon-idle` está de fora.

<!-- wiki-refs: ignore source_paths -- conceito do wiki, nao simbolo do projeto -->
Os dois arquivos de workflow estão no `source_paths` desta página, então mexer
neles a marca como defasada.

## Ver também

- [Como rodar](../overview/como-rodar.md) · [Achados em aberto](../notes/achados-abertos.md)
