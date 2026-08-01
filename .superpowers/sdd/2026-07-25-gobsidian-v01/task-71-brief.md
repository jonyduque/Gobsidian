### Task 71: RNF-01, RNF-02 e RNF-07 medidos na escala que o PRD nomeia

**Onde encaixa.** Depende da Task 70. Fecha a lacuna que `docs/OPERACAO.md` registra: os RNFs de escala foram medidos em cofre de **7 notas**, e medido em escala pequena é diferente de medido.

**Os alvos**, conforme `docs/PRD.md` §11:

| ID | Alvo | Medido hoje |
|---|---|---|
| RNF-01 | Indexação a frio ≤ 3 s | 5–8 ms **em 7 notas** |
| RNF-02 | Boot com cache válido ≤ 300 ms | 26,96 ms em 500 notas |
| RNF-07 | RSS em repouso ≤ 60 MB | 18,9–19,3 MB **em 7 notas** |

**A armadilha que te vincula, e é a mais cara deste projeto:** *não escreva número que você não mediu.* `docs/OPERACAO.md` chegou a trazer uma tabela de "Resultado da Medição v0.1" com *"Concluído abaixo do alvo (ex: 408ms em teste local)"* e *"Tende a ficar ~30-45 MB"*. O primeiro é exemplo, o segundo é expectativa; nenhum é medição. **Se não mediu, escreva "não medido"** — ninguém vai brigar com isso. Alvo não atingido e registrado é informação; alvo não medido apresentado como resultado é ficção com aparência de tabela.

#### Passos

1. Gere o cofre com `scripts/gen_vault.ps1 -Notes 5000`. Registre a semente e o tamanho real.
2. **RNF-01:** tempo de `index.Build` a frio, sem cache. Apague o `CacheDir` antes; um boot que leu cache não é boot a frio. Pelo menos 5 execuções; reporte mínimo, mediana e máximo — não uma execução.
3. **RNF-02:** tempo de boot com cache válido, medido depois de uma execução que o gravou.
4. **RNF-07:** RSS em repouso, depois de o servidor ficar pronto e ocioso. "Em repouso" quer dizer depois da indexação, não durante.
5. **RNF-04 em escala:** rode `TestRNF04VaultSearchLatencyP95` contra o cofre de 5.000 e registre por formato. O `limit: 200` já tem só ~20% de folga em 500 notas (ver Task 72); em 5.000 pode estourar, e **estourar e registrar é resposta certa**.
6. Atualize `docs/OPERACAO.md` com a data, a máquina nomeada, o número de núcleos e cada número medido.

#### Verificações além dos passos

- Confirme que o cofre tem mesmo 5.000 notas antes de medir: `idx.NoteCount() == 5000`. Uma medição sobre um cofre que não gerou por inteiro mede o que gerou.
- Onde um alvo **não** for atingido, escreva o número real e a diferença. Não afrouxe alvo, não use `t.Skip`, não arredonde para baixo.
- Se algum RNF não puder ser medido nesta máquina, escreva **"não medido"** e o motivo. Isso é entrega válida; número inventado não é.

#### Prova de mutação

Esta tarefa **não tem prova de mutação** — o entregável é medição, não regra. O que a substitui é a rastreabilidade: cada número do relatório precisa vir com o comando que o produziu e a saída colada. Não escreva no relatório um número que não esteja na saída colada logo acima dele.

#### Regras de execução

Idênticas às da Task 69.

#### Contrato de relatório

Grave em `.superpowers/sdd/task-71-report.md`: a semente e os números do cofre; para cada RNF, o comando, a saída colada e as 5 execuções; o diff de `docs/OPERACAO.md`; e o que ficou "não medido", com o motivo.

Responda com no máximo 15 linhas.

**Files:** Modify `docs/OPERACAO.md`; create measurement tests if needed
**Commit:** `docs(operacao): measure RNF-01, RNF-02, RNF-04 and RNF-07 at 5000 notes`

---

