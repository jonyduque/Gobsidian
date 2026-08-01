### Task 73: `bench.yml` no CI com detecção de regressão

**Onde encaixa.** H1 do M6. Depende das Tasks 70 e 71 — sem cofre determinístico e sem números de referência medidos, não há contra o que comparar.

**As decisões fechadas que te vinculam (nº 2 e nº 3 do M6):**

- **Regressão acima de 20% falha o build, e o número de referência é commitado.** Comparar contra "a última execução" torna degradação lenta invisível: 5% por semana nunca dispara e em dez semanas dobrou. A referência é um arquivo no repositório, atualizado por commit deliberado.
- **O número do CI não substitui a medição local.** Runner compartilhado tem variância alta. O CI guarda contra regressão de ordem de grandeza; o número que vai para `docs/OPERACAO.md` continua sendo medido em máquina nomeada.

#### Passos

1. `.github/workflows/bench.yml`. Gera o cofre com `gen_vault.ps1` e semente fixa, roda os benchmarks, compara com a referência.
2. A referência vive em arquivo versionado — sugestão: `docs/bench-baseline.json`, com o valor, a data e o runner que o produziu.
3. Comparação por benchmark, não agregada. Um agregado esconde a regressão de um caminho atrás da melhora de outro.
4. Falha acima de 20%. **Melhora acima de 20% não falha, mas avisa** — costuma significar que o benchmark parou de medir o que media, e este projeto já teve exatamente isso: uma medição de RNF-04 que reportava 0,58 ms porque chamava a camada errada.

#### Verificações além dos passos

- **Prove que o gate dispara.** Injete uma lentidão artificial acima de 20% num caminho medido, rode o workflow, confirme que o build falha, remova. Sem isso você escreveu o gate, não o verificou — e é literalmente a lição da Task 13 deste projeto, onde sete regras sobreviviam a mutantes com a suíte verde.
- **Prove que ele não dispara à toa.** Rode duas vezes sem mudança nenhuma e confirme verde nas duas. Um gate que pisca é um gate que alguém desliga.
- Confirme que `golangci-lint` no CI segue fixado em `v2.12.2`. O `go.mod` declara `go 1.25.0`, e um binário compilado com Go mais antigo recusa o config antes de analisar linha nenhuma.

#### Prova de mutação obrigatória

É a injeção de lentidão do primeiro item. Cole a saída do workflow falhando, com o percentual que ele reportou.

`scripts/mutate.ps1` **não serve aqui**: ele roda teste Go com `-Test` e `-Package`, e o alvo desta prova não é teste Go. A prova é a remoção descrita acima, com a saída colada — mesma disciplina, ferramenta diferente.

#### Regras de execução

Idênticas às da Task 69.

#### Contrato de relatório

Grave em `.superpowers/sdd/task-73-report.md`: o `bench.yml` inteiro; o baseline commitado; a saída do workflow falhando sob lentidão injetada, com o percentual; as duas execuções limpas em verde; e o que ficou de fora.

Responda com no máximo 15 linhas.

**Files:** Create `.github/workflows/bench.yml`, `docs/bench-baseline.json`
**Commit:** `ci: benchmark workflow with committed baseline and 20% regression gate`

---

