### Task 76: As duas lacunas de teste que atravessaram todos os marcos

**Onde encaixa.** Duas lacunas registradas há marcos e nunca fechadas. Esta tarefa é **projeto de teste**, não transcrição: o entregável é um cenário que hoje não existe.

**Lacuna 1 — a vigília do pai e os sinais nunca foram verificados ponta a ponta.** Registrada em `CLAUDE.md`: *"no harness de órfãos atual `stdin-eof` sempre vence (100/100 nas duas últimas rodadas), então vigília do pai e sinais seguem sem verificação ponta a ponta. Falta cenário em que stdin fica aberto e pai morre."*

Isto é sério porque a vigília do pai já falhou aqui, e feio: **comparar `(pid, creation time)` deixou 5 de 5 órfãos no primeiro teste ponta a ponta.** O Windows mantém PID e creation time consultáveis muito tempo depois da morte do processo; a correção foi usar `exitTime`. Em Unix, comparar o ppid capturado no startup, **não a constante 1** — sob Docker+tini, systemd ou s6 o reaper não é PID 1.

O que falta é um cenário em que **stdin fica aberto** e o processo pai morre, forçando o mecanismo de vigília a ser o que dispara. Enquanto `stdin-eof` vence, o resto é código não exercitado.

**A armadilha que decide se este teste vale alguma coisa:** *teste de mecanismo de recuperação que deixa o caminho normal ligado mede o caminho normal.* `TestOverflowReconciliationFull` injetava overflow com o watcher ativo; os eventos comuns aplicavam as mudanças e a reconciliação nunca era exercitada. Removido o reconciliador inteiro, o teste passava em 2,8 s — cobertura zero num requisito P0, através de uma revisão que o aprovou. **Teste de fallback desconecta o caminho principal, ou não é teste de fallback.**

Além disso: o gate de órfãos precisa **gatear em `reason=`**. Sem isso, servidor morrendo sozinho dá rodada verde sem mecanismo nenhum ter disparado — defeito real, achado na revisão do M0.

**Lacuna 2 — RNF-32, links simbólicos.** Registrada na auto-revisão do plano: *"tem teste apenas indireto na Task 7, porque criar symlink no Windows exige privilégio elevado e o teste falharia em máquina de desenvolvimento comum. `vault.Walk` usa `filepath.WalkDir`, que não segue symlinks por padrão — a propriedade vale por construção. Adicione um teste explícito em M6, marcado com `t.Skip` quando a criação de symlink falhar por permissão."*

#### Passos

1. Estenda `scripts/test_orphans.ps1` com um cenário `parent-death`: stdin **aberto**, pai morto. Confirme, pelo `reason=` registrado, que foi a vigília do pai que disparou — não `stdin-eof`.
2. Um cenário `signal` equivalente, em que nem stdin fecha nem o pai morre: o sinal é o que encerra.
3. O gate reprova se o `reason=` não for o esperado para o cenário. Cenário que encerra pelo motivo errado é cenário que não testou o mecanismo que nomeia.
4. Teste explícito de RNF-32, com `t.Skip` quando a criação de symlink falhar por permissão. **A mensagem do skip precisa dizer que pulou por permissão**, senão um skip permanente vira cobertura fantasma.

#### Verificações além dos passos

- Rode 100 ciclos de cada cenário e reporte a distribuição dos `reason=`. Se `stdin-eof` continuar vencendo em algum cenário que não é o dele, o cenário não está isolado.
- **Prove por remoção**: desligue a vigília do pai e confirme que o cenário `parent-death` deixa órfãos. Se ele continuar verde, ele não testa a vigília. Mesmo para o cenário de sinal.
- Para o RNF-32: rode uma vez com privilégio (se disponível) e confirme que o teste **executa** e passa; sem privilégio, confirme que ele **pula com a mensagem certa**. Um teste que só pula nunca foi verde.

#### Prova de mutação obrigatória

As duas provas por remoção acima, com a saída colada de cada uma.

`scripts/mutate.ps1` **não serve aqui**: ele roda teste Go com `-Test` e `-Package`, e o alvo desta prova não é teste Go. A prova é a remoção descrita acima, com a saída colada — mesma disciplina, ferramenta diferente.

#### Regras de execução

Idênticas às da Task 69. Atenção adicional: goroutine parada em `Read` não é desenrolável por cancelamento de context — é por isso que `watchStdin` fica fora do `WaitGroup`. Incluí-la trava `Wait()` quando sinal ou pai dispara primeiro.

#### Contrato de relatório

Grave em `.superpowers/sdd/task-76-report.md`: a distribuição de `reason=` em 100 ciclos por cenário; as duas provas por remoção, coladas; o resultado do teste de RNF-32 nos dois modos; e o que ficou de fora.

Responda com no máximo 15 linhas.

**Files:** Modify `scripts/test_orphans.ps1`, `internal/lifecycle/`, `internal/vault/walk_test.go`
**Commit:** `test(lifecycle,vault): exercise parent-death, signal and symlink paths`

---

