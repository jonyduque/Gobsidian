# SDD ledger — plan: docs/superpowers/plans/2026-09-11-resources-arvore-e-sdk.md

Spec: pedidos do dono de 2026-09-11 a 2026-09-14 — resources como árvore, SDK despinado, e, depois da investigação do daemon, "inclua no planejamento para implementação de tudo de uma vez". Execução autorizada em 2026-09-14 ("Sim." à pergunta "crie o ledger do plano e comece pela Parte I").

BASE inicial: 48a8fbf (docs(plan): the daemon Claude Desktop never reached, and why). Branch: master (o dono trabalha em master).

Os itens do plano não usam o número global de Task: são identificados pela parte (I1.1, G3.2...). `sdd.ps1 base/brief/review` recebem esse identificador como texto.

## Já entregue antes deste ledger

- Parte 0 (SDK v1.5.0 -> v1.7.0): complete — 9f25236. verify.ps1 22/22.
- Parte E (medições no Claude Desktop 1.52386.6): complete — 9f25236 (ESTADO, plano).
- Parte G0 (causa do daemon inalcançável) e o próprio plano G/H/I: complete — 48a8fbf. verify.ps1 22/22.

## Pre-flight

| Item | Produz vs consome | Achado |
|---|---|---|
| I1.1 | cria o desvio do diretório de runtime em `ipc`; G2.1 vai acrescentar `DiretorioDeSockets` | G2.1 precisa seguir o mesmo desvio — registrar em G2.1 quando chegar lá |
| I1.1 x testes | `ipc`, `daemon`, `doctor` têm arquivos `package X` e `package X_test` no mesmo diretório | um binário de teste por diretório: UM `TestMain` por diretório, não por pacote de teste |
| I1.2 | o plano descreve fotografia antes/depois do diretório real | a medir: processo real do usuário cria arquivo durante a rodada — ver ruling R1 |
| I1.3 | o plano pede limpeza de trava e presença órfãs | a medir se `instalar.Limpar` já cobre |

## Tasks
- Parte I: executada pelo orquestrador direto, sem subagente (três itens pequenos e acoplados ao gate). BASE=48a8fbf.
- Ruling R1 (I1.2): o gate NÃO fotografa o diretório real antes e depois, como o plano dizia. Motivo medido nesta sessão: o diretório de runtime da máquina do dono muda sozinho — 132 → 45 arquivos entre duas medições sem nenhum teste (não atribuído) e novas travas a cada partida de host. A isca (LOCALAPPDATA, XDG_RUNTIME_DIR, XDG_CACHE_HOME do processo `go test` apontando para diretório vazio) mede o mesmo sem ruído. Custo se errado: no macOS vazamento para o CACHE não cai na isca (os.UserCacheDir ignora env lá); registrado no próprio script.
- Ruling R2 (I1.1): o desvio é função exportada de `ipc` (`RodarComRuntimeIsolado`), e não variável não exportada como `instalar.raizDoCache`, porque cruza cinco pacotes. Protegido por regra nova no `check_test_isolation.ps1` (recusa chamada fora de `_test.go`), com três casos.
- I1.3: já coberto por `instalar.Limpar`, sem código. Medido: 62 arquivos no diretório real, `doctor` listou 16 travas e 1 presença removíveis.
- Evidência: RED (isca, antes) gate exit 1 com 15 travas + instalacao.lock + serve.53752.presenca; GREEN (isca, depois) exit 0; mutação A (sem TestMain do daemon) exit 1 com 4 travas, restaurado exit 0; mutação B (sem TestMain do ipc) TestSocketDeTesteCaiNoDesvio FAIL, restaurado PASS; sentinelas no diretório real sobreviveram à suíte (testes não apagam lá). check_gates 58 casos OK.
- Observado e não atribuído: `TestRNF04SnippetConcurrencyLimit200` reprovou (p95 80–594 ms, teto 22 ms) quando rodei `go test` de todos os pacotes em paralelo contra a isca, antes e depois da mudança. O verify roda esse teto numa etapa sozinha; conferir no resultado do verify.
- verify.ps1 da Parte I, 1ª rodada: EXIT=1, três etapas. (1) `check_doc_refs`: meu texto em ARMADILHAS.md citava `_test.go` entre crases; corrigido para prosa, check_doc_refs OK. (2) `go test -race`: DATA RACE em `internal/daemon` — a goroutine de `TestEnsureStartedPerdedorNuncaChamaIniciar` (lock_test.go:107) sobrevive a `m.Run` e lia `desvioDoRuntime` enquanto o `defer` de `RodarComRuntimeIsolado` o zerava. O comentário que eu escrevi afirmava que isso não acontecia. Conserto: nunca zerar (o processo sai com os.Exit logo depois). Depois: 5 rodadas de `go test -race` sobre os cinco pacotes, 0 corridas, 5×5 ok. (3) teto de latência `TestRNF04SnippetConcurrencyLimit200`: p95 31–38 ms contra 22 ms, sozinho. `go list -deps ./internal/service/` não contém `internal/ipc`, e esta parte não toca código que o pacote `service` compila — A/B contra 48a8fbf em worktree registrado abaixo.
- A/B do teto de latência, rodadas alternadas no mesmo minuto: 48a8fbf (worktree, removido depois) FAIL p95 24,3 ms e FAIL 48,3 ms; árvore de trabalho ok e ok. Ruído de carga da máquina, não regressão. Carga amostrada antes: msedge, node e MsMpEng no topo, 12 núcleos lógicos. Worktrees do Antigravity listados em `git worktree list` não são meus e não foram tocados.
- verify.ps1 da Parte I, 2ª rodada: disparado depois do conserto da corrida.
- verify.ps1 da Parte I, 2ª rodada: EXIT=0, 23 etapas, "Bateria completa. Pode commitar." (inclui `go test -race`, o teto de latência e `check_runtime_limpo` verdes; 6 testes pulados, informativo).
- Parte I (I1.1, I1.2, I1.3): complete — commit abaixo, `test: the suite stops writing into the user's runtime directory`. Próximo item da ordem: G1 (medições; precisa de uma reinicialização do Claude Desktop pelo dono).
