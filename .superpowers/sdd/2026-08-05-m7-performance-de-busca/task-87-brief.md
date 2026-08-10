# Task 87 — Relatórios, ledger e medição de fechamento

**Tier: modelo forte.** O entregável são relatórios com evidência real, e o modo de falha de um modelo barato pedido a "escrever relatório com evidência" é
fabricá-la.

#### Onde encaixa
Fechamento. Não envia código.

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

#### O que entregar
- Tabela dos quatro RNF (02, 04, 06, 07) com o número **antes** e **depois**,
  medidos no cofre real, e a palavra "não atingido" onde for o caso.
- `docs/OPERACAO.md` atualizado; `README.md` com a contagem certa de requisitos
  não atingidos (já errou uma vez: dizia "três" com quatro na tabela).
- Ledger em `.superpowers/sdd/2026-07-25-gobsidian-v01/progress.md`.
- `pwsh -File scripts/audit_reports.ps1` sem achados nas seções novas.
- **Todo SHA citado conferido com `git cat-file -t`.** A Task 31 foi registrada
  em `14210ee`, que não existe.

#### Verificações além dos passos
- `git cat-file -t <sha>` para **cada** SHA citado, com a saída colada. A Task 31
  foi registrada em `14210ee`, que não existe no repositório.
- `audit_reports.ps1` rodado, e os achados da seção nova em zero — achados
  antigos em relatórios de outros marcos não contam e devem ser distinguidos.
- Validação UTF-8 de todo `.md` tocado.

#### Contrato de relatório
Esta tarefa **não tem prova de mutação**: não envia código.

**Files:** `docs/OPERACAO.md`, `README.md`, ledger
**Commit:** `docs(ledger): record M7 and the measured state of the four RNFs`

---

## Ordem de execução, e por quê

```
78 → 79 → 80 → 81 → 82 → 83 → 84 → 85 → 86 → 87
```

- **78 antes de tudo**: as seis otimizações são medidas contra o golden.
- **80 antes de 81**: o pool beneficia indexação também; medir na ordem inversa
  esconde o ganho do pool atrás do do título.
- **80, 81, 82, 83 tocam `bm25.go`/`analyzer.go`** — sequenciais, nunca em
  paralelo.
- **84 e 85 não conflitam** com nada acima nem entre si.
- **86 por último entre as de código**: maior risco, e quer o golden e os
  benchmarks já estáveis para atribuir regressão.

---

## Prompt de despacho

> **Lote M7 — performance de busca e leitura em lote. Tasks 78 a 87.**
>
> **O que torna este lote diferente:** seis das dez tarefas mudam como o score
> de busca é calculado. **Otimização que muda resultado é defeito, não
> trade-off.** A Task 78 congela o ranking num golden; toda tarefa seguinte tem
> de deixá-lo idêntico. **Nunca regenerar o golden com `-update` para fazer
> passar** — golden que muda exige explicação escrita e volta para revisão.
>
> **Estado inicial:** base `236b135`, árvore limpa, ledger em
> `.superpowers/sdd/2026-07-25-gobsidian-v01/progress.md`.
>
> **Ordem:** 78 → 79 → 80 → 81 → 82 → 83 → 84 → 85 → 86 → 87. As tarefas 80 a 83
> tocam os mesmos arquivos; **não paralelizar**.
>
> **Laço por tarefa:**
> ```
> pwsh -File scripts/sdd.ps1 base <N>      # ANTES de comecar
> pwsh -File scripts/sdd.ps1 brief <N>
> # executar
> pwsh -File scripts/verify.ps1
> pwsh -File scripts/sdd.ps1 review <N>
> # ledger ANTES de dizer que acabou
> ```
>
> **Aceitação por tarefa:**
> - **78** — os seis `.tsv` colados, e uma frase por consulta dizendo por que
>   aquela ordem está certa. Falha barata: golden gerado sobre corpus que não
>   tem o tamanho que o nome diz. O corpus afirma o próprio tamanho; conferir
>   que a asserção existe.
> - **79** — prova de disparo (inserir `create_dirs` na doc, ver o achado,
>   remover) **e** o volume total no repositório. Falha barata: checador que
>   dispara em prosa legítima vira ruído e para de ser lido.
> - **80** — `benchstat` das DUAS mudanças separadamente. Falha barata: pool sem
>   `Reset()`, que produz string errada só sob concorrência. Exigir o teste
>   rodado com `-race`.
> - **81** — falha barata: `TitleNorm` preenchido em `Build` e esquecido em
>   `Replace` ou `MoveNote`. O teste cobre os três; conferir que cobre.
> - **82** — falha barata: cache em variável de pacote, compartilhado entre
>   cofres. Exigir que esteja no `Inverted`.
> - **83** — **sem prova de mutação**, e o relatório tem de dizer isso. Falha
>   barata: reordenar o laço "para aproveitar a estrutura", quebrar o golden por
>   arredondamento e regenerá-lo.
> - **84** — falha barata: caminho que falha some da lista em vez de aparecer na
>   posição com erro. Exigir o teste dos dez com um inexistente.
> - **85** — falha barata: cache que confere versão e não cobertura, que já
>   aconteceu no cache de busca. Exigir o diferencial construído-vs-recarregado.
> - **86** — falha barata: chave do mapa reverso calculada em dois lugares.
>   Exigir que passe por uma função só, e o diferencial contra o caminho global.
> - **87** — sem prova de mutação; **dizer isso**. Todo SHA conferido com
>   `git cat-file -t`.
>
> **Decisões que não se re-litigam:** D-M7-1 a D-M7-5, na seção de decisões
> fechadas. Em especial: `~` no `benchstat` **reverte a mudança**, e nenhum teto
> de RNF é afrouxado nesta batelada.
>
> **Regras para quem orquestra:** revisor também erra — conferir a afirmação
> contra o código antes de aceitar o achado. Escopo não encolhe em silêncio:
> `BLOCKED` com motivo é resposta melhor que entrega que parece completa. Todo
> SHA que for para o ledger passa por `git cat-file -t`.
>
> **Gate final:**
> ```
> pwsh -File scripts/verify.ps1
> pwsh -File scripts/test_orphans.ps1 -Cycles 100
> pwsh -File scripts/audit_reports.ps1
> pwsh -File scripts/check_doc_refs.ps1
> ```
>
> **O que volta para quem pediu:** mudar o schema de `note_read` é contrato
> público — a Task 84 altera uma tool já publicada. Fechar a anotação do Q3 no
> PRD é decisão de projeto.

### Custo e tiers

| Tier | Tasks | Por quê |
|---|---|---|
| Barato | 79, 80, 81, 82, 83 | corpo dos testes difíceis está escrito; é transcrição |
| Forte | 78, 84, 85, 86, 87 | teste que ainda não existe, contrato público, ou relatório com evidência |

Estimativa: 10 tarefas, ~2 invocações cada com revisão, ~20 no total.

**A que eu não delegaria sem ler a saída inteira: a 86.** Ela mexe em resolução
de link, o defeito que ela pode introduzir resolve para a nota errada com
`state=ok`, e este projeto já pagou exatamente esse.

---

# Parte II — Custo por instância (Tasks 88–93)

Cada sessão de host MCP abre **um** processo `gobsidian serve`. É o transporte
stdio: o host cria o subprocesso e fala JSON-RPC pelo pipe. O servidor não tem
como recusar — quando ele existe, o host já o criou.

Medido em 2026-08-05 na máquina do projeto, com duas sessões do Claude vivas:

```
PID 24988  v1.0.0  943,8 MB   pai: claude vivo
PID 54892  v1.0.1  584,7 MB   pai: claude vivo
```

Nenhum é órfão. É o custo normal, multiplicado. O build atual mede **381,5 MB**
em repouso no mesmo cofre, então parte disso se resolve reinstalando; o resto é
estrutural, e é o que estas seis tarefas atacam.

## A medição que decide o transporte

Feita antes de escrever as tarefas, não depois. Eco de ida e volta, 20.000
repetições por tamanho, após aquecimento, em windows/amd64, 12 núcleos:

| Transporte | 256 B | 4 KB | 64 KB |
|---|---|---|---|
| **AF_UNIX** (`net.Dial("unix")`) | **25,7 µs** | **23,0 µs** | **42,9 µs** |
| named pipe (`go-winio`, config padrão) | 82,9 µs | 93,5 µs | 110,0 µs |

**AF_UNIX ganha em todos os tamanhos, por 3 a 4x**, está na biblioteca padrão e
é o mesmo código nos três sistemas. Windows suporta AF_UNIX desde a versão 10
1803 (abril de 2018).

**Decisão D-M7-6: AF_UNIX nos três sistemas, sem compilação condicional para o
transporte.** Duas razões, e a segunda é mais forte que a primeira:

1. Ganhou a medição, e sem trazer dependência nova.
2. **A escolha quase não importa.** A ida e volta custa ~25 µs contra uma busca
   que leva 90 a 200 ms — quatro ordens de grandeza. Mesmo que o named pipe
   ganhasse, o critério certo seria complexidade e dependência, e AF_UNIX vence
   os dois. Otimizar o transporte aqui seria ajustar 0,02% do tempo.

Ressalva honesta: uma execução, uma máquina, `go-winio` com configuração padrão.
Um named pipe ajustado pode fechar parte da distância. Isso não muda a decisão,
porque a decisão não depende da margem.

Build tag continua existindo para **o caminho do socket e a limpeza dele**, não
para o transporte — Windows deixa um arquivo que precisa ser removido, Linux
poderia usar namespace abstrato. Ver Task 91.

---

