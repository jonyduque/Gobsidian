### Task 49: Cache de índice em disco — e a decisão da Q3 por medição

RF-06 e RNF-02 (boot com cache válido ≤ 300 ms). Fecha a Q3 do PRD §11.

#### Onde isto encaixa

Última tarefa de código do M3. Até aqui o servidor reconstrói tudo no boot; esta tarefa persiste, e decide **o que** persistir.

#### A pergunta que esta tarefa responde, e que não pode ser respondida por opinião

**Q3 do PRD §11:** o cache guarda também o índice de busca, ou apenas o índice de metadados?

Serializar o índice invertido inteiro é o que faz RNF-02 ser atingível, mas é o formato mais caro de versionar — qualquer mudança no analisador invalida tudo.

**O plano não decide isto. A medição decide.** O procedimento é fixo:

1. Meça o tempo de reconstruir o índice invertido a partir do índice de metadados **já carregado**, num cofre de tamanho realista. Registre a contagem de notas ao lado.
2. Se esse tempo, somado ao de carregar o índice de metadados, couber em 300 ms: **persista só os metadados**. É o formato mais barato de versionar, e a busca se reconstrói.
3. Se não couber: persista os dois, e registre o custo de versionamento como dívida.
4. **Escreva o número medido no relatório e no PRD §11, fechando a Q3 com a data.**

**Não escreva número que você não mediu.** Uma tabela de "Resultado da Medição" com `"ex: 408ms em teste local"` já foi commitada neste projeto — o `"ex:"` fazia todo o trabalho. Se não conseguir medir, o resultado é `BLOCKED` com o motivo, não uma estimativa.

#### O que já está fechado

- **O cabeçalho carrega `format_version`, `parser_version` e `analyzer_version`.** Cache lido com qualquer um deles diferente é **descartado**, não migrado. Um cache aceito com analisador diferente produz busca que não acha o que existe, sem erro nenhum.
- **O cache fica FORA do cofre** (PRD D1). `config.CacheDir` já deriva o caminho de um hash do caminho absoluto do cofre.
- **Cofre inacessível e cache inválido não podem produzir a mesma resposta.** Cache inválido reindexar; cofre inacessível é erro.
- **`ctx` onde bloqueia.** Escrever e ler o cache é I/O.

#### Armadilhas já pagas que se aplicam

- **Pergunte o que um valor zero significa.** Um cache de cofre vazio é legítimo. "Zero notas no cache" e "cache ausente" precisam ser distinguíveis, ou o boot de um cofre vazio vira reindexação a cada vez — ou pior, um cofre cheio com cache corrompido passa por vazio.
- **Escrita atômica.** Um cache escrito pela metade, porque o processo morreu no meio, tem de ser recusado no próximo boot, não lido parcialmente. Escreva em temporário e renomeie. O prefixo `.gobsidian-tmp-` já é ignorado pelo filtro do watcher — é para isso que ele existe.
- **`-update` de golden grava o que o código produz.** Se houver golden de formato de cache, leia o que gerou.

#### Verificações além dos passos

- **RNF-02 medido:** boot com cache válido, em milissegundos, com a contagem de notas ao lado. Número medido ou **"não medido"**.
- O tempo de reconstruir a busca a partir dos metadados, medido — **é a resposta da Q3**.
- Cache com `analyzer_version` diferente é descartado? Prove mudando a versão e confirmando reindexação.
- Cache truncado no meio é recusado?
- Cache de cofre vazio é distinguível de cache ausente?
- O cache está fora do cofre? Confirme o caminho real.
- Escrita interrompida deixa o cache anterior intacto?

**Prova de mutação obrigatória:** desligue a checagem de `analyzer_version` e confirme que um teste nomeado reprova.

#### Regras de execução e contrato de relatório

Idênticos aos da Task 43. Relatório em `.superpowers/sdd/task-49-report.md`, com **os dois números medidos**, o diff do PRD §11 fechando a Q3, e a prova de mutação.

**Files:** Create `internal/index/cache.go`, `internal/search/persist.go` e testes; Modify `cmd/gobsidian/serve.go`, `docs/PRD.md` §11
**Commit:** `feat(index): on-disk cache with version header`

---

