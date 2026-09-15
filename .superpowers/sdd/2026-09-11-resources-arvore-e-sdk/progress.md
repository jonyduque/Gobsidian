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
- G1 (G1.0-G1.4): executado pelo orquestrador. `tools/sondahost` escrito, vet nos três GOOS e golangci-lint ok; smoke pela shell elevada: tudo ok. Registrado no config do Desktop (backup em %LOCALAPPDATA%\Temp\gobsidian-g1), dono reiniciou o Desktop, três processos da sonda rodaram; entrada removida do config e `sondahost limpar` apagou tudo, inclusive as três cópias em LocalCache.
  - G1.1: `%USERPROFILE%\.gobsidian-sondahost\run` ok (próprio e cruzado) nos três; Temp ok; controle em LOCALAPPDATA\gobsidian\run 1920/10022.
  - G1.2: trava da shell recusada nos três, trava livre tomada nos três.
  - G1.3: diretório novo na raiz de LOCALAPPDATA DESVIADO para Packages\Claude_pzs8sxrjxfjjc\LocalCache\Local nos três.
  - Consequência: G2 "só o socket" não é seguro por si; três caminhos escritos no plano, decisão do dono pendente. G2 fica BLOCKED nessa decisão; G3/G4/G5/G8 não dependem dela.
- Decisão do dono (2026-09-14, pergunta direta): G2 caminho 1 — diretório de runtime inteiro (socket, travas, log, presença, instalacao.lock) em %USERPROFILE%\.gobsidian\run; cache fica em %LOCALAPPDATA%\gobsidian. G2 desbloqueado; G2.3 (ponte nova recusa subir daemon com a trava antiga tomada) passa a ser obrigatório.
- verify.ps1 do G1, 1ª rodada: EXIT=1, só o teto de latência. Rodado sozinho em seguida: ok na 1ª, FAIL p95 23,8 ms contra 22 ms na 2ª — o mesmo ruído de carga medido em A/B na Parte I; esta mudança só toca tools/sondahost e docs. Refazer o verify antes do commit.
- verify.ps1 do G1, 2a rodada: EXIT=0, 23 etapas, Bateria completa (teto de latencia verde nesta rodada).
- G1: complete - 5458dad. Correcao de registro: a mensagem de 5458dad diz que G2 espera a decisao do dono; a decisao (caminho 1) ja estava tomada e esta registrada no plano e no ESTADO do mesmo commit. Causa: o script que trocava o paragrafo da mensagem nao achou a ancora e a cadeia seguiu para o commit sem parar. Nao reescrevi o commit.
- Proximo: G3, G4, G5, G8; depois G2 com o diretorio de runtime inteiro.
- G3, G4, G5: executados pelo orquestrador. RED: os testes novos de ipc reprovaram e os de ponte e doctor nao compilavam (sondarSocketFn, checkDiretorioDeSockets indefinidos). GREEN depois das edicoes. Um defeito achado no proprio GREEN: o nome do socket da sonda (sonda-<pid>-<nanos>.sock) era mais longo que o de um cofre e reprovou por limite de caminho AF_UNIX num t.TempDir(); o nome passou a ter o comprimento do socket real, e o teste usa os.MkdirTemp curto.
- Provas de mutacao (restauracao conferida com cmp contra copias): G3 sem a guarda em Listen -> TestListenNaoLimpaSocketOndeOProprioSocketNaoConecta FAIL; G4 com a sonda desligada -> TestPonteNaoIniciaDaemonOndeOSocketNaoConecta FAIL "a ponte iniciou um daemon..."; G5 ignorando a sonda -> TestCheckDiretorioDeSocketsAvisaOndeOSocketNaoConecta FAIL "status = 0, esperado Warn". Restaurados, os tres passam.
- Ruling R3 (G4.2): motivoDaQueda nao ganha caso de errors.Is para ErrDiretorioSemSocket -- seria codigo morto; o motivo entra como padrao da mesma funcao.
- verify de G3-G5, 1a rodada: EXIT=1, so o teto de latencia (TestRNF04SnippetConcurrencyLimit200). go list -deps ./internal/service/ nao contem ipc, doctor, daemon, instalar nem cmd/gobsidian -- os pacotes desta mudanca. A/B alternado contra 5458dad em worktree (removido): 5458dad ok, ok, FAIL 46,1 ms; arvore de trabalho ok, FAIL 28,3 ms, ok. Os dois lados reprovam uma vez em tres: e o teto oscilando com a carga desta maquina, nao a mudanca. Terceira vez hoje que o teto reprova um verify; fica registrado como divida a medir (o teto de 22 ms nao se sustenta com a maquina carregada). Refeito o verify completo antes do commit.
- Achado para G2.3, lendo internal/daemon/lock.go: a 'trava antiga' que o plano descreve nao existe para um daemon em regime -- `.sock.lock` so e segurada durante EnsureStarted e `.sock.listen.lock` so durante o Listen. O que um daemon vivo segura a vida inteira e a presenca (`daemon.<pid>.presenca`, trava de kernel) no diretorio de runtime dele. O sinal equivalente para 'ha daemon de versao anterior neste cofre' e presenca viva de papel daemon no diretorio ANTIGO. Tambem mapeado: todo caminho de runtime deriva de ipc.RuntimeDir (SocketPath -> lockPath, CaminhoDoLog, trava de escuta; instalar.DiretorioDeRuntime), entao a mudanca de diretorio em si e uma conta so, runtimeDirDoSistema no Windows. Revisar o texto de G2.3 no plano quando chegar la.
- verify de G3-G5, 2a rodada: EXIT=0, 23 etapas, Bateria completa.
- G3, G4, G5: complete - commit `fix(ipc): never trust "nobody listens" where this process cannot dial its own socket`. Proximo: G8.
- G8 (G8.1-G8.3): executado pelo orquestrador, codigo e testes preparados fora do repositorio enquanto o gate de G3-G5 rodava. RED: `ModoPonte` e `situacaoDoCofre` indefinidos. GREEN: testes novos e os de presenca, ponte e partida verdes; vet nos tres GOOS; golangci-lint 0 issues.
- Ruling R4 (G8.1): sem estado `decidindo` nem regravacao -- os tres registros de presenca ja rodam depois da decisao, entao o modo entra no proprio registro.
- Mutacoes (restauracao conferida com cmp): ponte como gravador -> TestGravaCacheSoDaemonEEmProcesso FAIL e dois subtestes de TestAvisoDeDuplicidadeContaSoGravadores FAIL; Modo fora do JSON -> TestPresencaGravaOModo FAIL `modo lido = ""`; SemPresenca sem tirar registrados -> TestSemPresencaTiraRegistradosEOProprio e TestProcessosDoSistemaAchaGobsidianSemPresenca FAIL; sem modo como gravador -> subteste `processo sem modo nao conta como gravador` FAIL. Restaurados, todos passam.
- doctor real com o binario novo nesta maquina: 28 presencas v1.8.1, todas `sem modo registrado (versao anterior)`; 1 processo sem presenca, pid 46888 C:\Program Files\gobsidian\gobsidian.exe (v1.5.1 do Claude Code). Observado e nao investigado: 24 processos serve v1.8.1 vivos para quatro cofres.
- verify de G8: EXIT=0, 23 etapas, Bateria completa.
- G8: complete - commit `fix(doctor): count only cache writers as duplicates, and show processes without presence`. Proximo: G2.
