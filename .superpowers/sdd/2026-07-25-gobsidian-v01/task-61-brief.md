### Task 61: A lacuna do RNF-04 — busca por frase

Promovida da revisão do M3.1. **Independente das outras oito; pode rodar em qualquer ponto do marco.**

#### A evidência medida

Medido em 2026-07-29 por `TestRNF04VaultSearchLatencyP95`, 500 notas distintas, 30 consultas por formato, através de `service.Search`:

```
termo amplo, limit default     p95  14,9 ms
dois termos                    p95  17,7 ms
termo seletivo                 p95   5,7 ms
filtro de pasta                p95  17,4 ms
filtro de tag                  p95  11,9 ms
frase exata                    p95 174,2 ms   <- 1,7x o alvo
trecho de 1000 chars           p95  24,5 ms
limit: 200 (maximo do schema)  p95  85,0 ms
```

**RNF-04 é atingido em sete de oito formatos e falha em um: frase exata, 174 ms contra 100 ms.** O teste hoje cobra um **teto de 250 ms** nesse formato — guarda contra piorar sem afrouxar os outros sete e sem esconder a lacuna.

**A hipótese está registrada e não confirmada:** a busca por frase percorre as posições da forma crua em todas as postings do termo mais raro, e o caminho de termo solto não faz isso. **Confirme ou refute antes de otimizar** — otimizar o lugar errado é trabalho jogado fora, e o instrumento é `pprof`, não leitura. A skill `golang-benchmark` cobre a metodologia.

#### O que implementar

Depois de confirmar onde o tempo está: reduza o p95 da frase exata para dentro dos 100 ms, **ou** reporte que não é possível sem mudar o formato do índice e registre a lacuna com o motivo. As duas são respostas; "otimizei um pouco" não é.

Baixe o teto do teste para o novo p95 medido mais folga. **Não** apague a linha nem o formato da tabela: o que ele mede continua sendo o pior caso.

#### Verificações além dos passos

- Onde está o tempo? Perfil de CPU, com a função dominante nomeada. **Medido, não suposto.**
- O p95 novo, com a mesma metodologia (`service.Search`, 500 notas, 30 por formato) — comparável com o antigo ou não é evidência de melhora.
- Os outros sete formatos **não** pioraram? Cole a tabela inteira, antes e depois.
- A busca por frase continua correta? Frase exata tem de casar sequência, não os termos soltos — otimização que quebra isso troca latência por resposta errada. O teste de correção da Task 48 continua verde?
- `docs/OPERACAO.md` atualizado com o número novo, ou com a lacuna registrada se ela persistir.

**Prova de mutação obrigatória:** confirme que o teste de correção de frase exata (Task 48) ainda reprova se a busca por frase passar a casar termos soltos. É o que impede a otimização de virar regressão de comportamento.

**Files:** Modify `internal/search/`, `internal/service/search_test.go`, `docs/OPERACAO.md`
**Commit:** `perf(search): bring exact-phrase p95 within RNF-04`

---

