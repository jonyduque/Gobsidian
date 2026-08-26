# gobsidian

Servidor MCP em Go que expõe um cofre Obsidian local a hosts MCP. Roda como
subprocesso sobre stdio, em Windows.

Este arquivo é o **índice** e as **regras que não se negociam**. Tudo o mais está
nos documentos abaixo — cada fato mora num lugar só, porque duas cópias do mesmo
fato divergem e a menos consultada é a que fica errada.

---

## Índice da documentação

### Normativa — onde divergir de qualquer outra coisa, esta vence

| Documento | O que responde |
|---|---|
| [`docs/PRD.md`](docs/PRD.md) | Requisitos, prioridades, RNFs e as decisões fechadas D1–D13 |
| [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) | Camadas, fluxos e as decisões AD-01–AD-09 |
| [`docs/TOOLS.md`](docs/TOOLS.md) | Contrato de cada tool: schema, retorno, erros |
| [`docs/ESTRUTURA.md`](docs/ESTRUTURA.md) | Árvore autoritativa de arquivos e convenções |
| [`docs/WINDOWS.md`](docs/WINDOWS.md) | OneDrive, MAX_PATH, casing, fsnotify |
| [`docs/OPERACAO.md`](docs/OPERACAO.md) | Como operar, medições publicadas e limites conhecidos |

### Por papel — leia o do trabalho que você vai fazer

| Papel | Documento |
|---|---|
| Escrever código | [`docs/papeis/implementador.md`](docs/papeis/implementador.md) |
| Revisar código de outro | [`docs/papeis/revisor.md`](docs/papeis/revisor.md) |
| Escrever ou avaliar teste | [`docs/papeis/testador.md`](docs/papeis/testador.md) |
| Otimizar desempenho | [`docs/papeis/desempenho.md`](docs/papeis/desempenho.md) |
| Escrever documentação | [`docs/papeis/documentador.md`](docs/papeis/documentador.md) |
| Escrever, despachar e auditar tarefas | [`docs/papeis/orquestrador.md`](docs/papeis/orquestrador.md) |

### Histórica — o porquê

| Documento | O que responde |
|---|---|
| [`docs/ARMADILHAS.md`](docs/ARMADILHAS.md) | Todo defeito que já custou caro aqui, com o mecanismo. **Releia antes de commitar.** |
| [`docs/ESTADO.md`](docs/ESTADO.md) | Marcos, medições, formato de cache, gates e dívidas abertas |
| [`docs/SUGESTOES.md`](docs/SUGESTOES.md) | Auditoria de 2026-08-25, com as decisões do dono registradas |
| [`docs/REVISAO-2026-08-15.md`](docs/REVISAO-2026-08-15.md) | Revisão anterior, com código e trade-offs |

### Derivada

`docs/wiki/Home.md` explica **o código** e responde oito perguntas na ordem em
que um recém-chegado as faz. Cita a normativa; não a recopia. Página com
`status: stale` é ponto de partida, não resposta.

**Plano e ledger:** `docs/superpowers/plans/` e
`.superpowers/sdd/<marco>/progress.md`. O ledger é o que a próxima sessão tem no
lugar do seu contexto.

---

## Estrutura do projeto

```
cmd/gobsidian/     entrypoint fino e subcomandos: serve, doctor, index, search,
                   inspect, daemon (oculto). ponte.go escolhe daemon vs
                   em-processo; servico.go monta o Service que serve e daemon
                   compartilham
internal/
  config/          struct de configuração, flags cobra, defaults, VaultKey
  lifecycle/       stdin-eof, sinais, vigília do PID pai, shutdown com orçamento
  vault/           raiz, caminho canônico e confinamento, walk com exclusões,
                   EOL, detecção de somente-nuvem
  parser/          goldmark + extensões [[wikilink]], ^blockid, #tag,
                   campo::inline; headings com offsets de byte
  index/           Note, RWMutex, build com worker pool, update incremental,
                   backlinks, resolve, alias, query
  search/          analyzer, índice invertido base/delta, BM25, trecho,
                   cache binário formato 6 + arena mmap
  watcher/         fsnotify, debounce, filtro de relevância, apply,
                   rename por hash, reconciliação pós-overflow
  writer/          lock por caminho canônico, temp+sync+rename atômico,
                   edição sob heading/bloco, reescrita de link, diff
  service/         fachada das tools em tipos de domínio; erros em errors.go
  mcpsrv/          ÚNICO pacote onde tipos do SDK de MCP existem
  console/         marcadores ASCII e cor decidida pelo destino de saída
  ipc/             transporte local: socket, saudação, handshake
  daemon/          N conexões sobre um índice; lock anti-corrida; spawn
  doctor/          diagnóstico de ambiente e do runtime do daemon
  text/            normalização
docs/              normativa, papéis, história, wiki
testdata/          golden files do parser, cofre pequeno, corpus de paridade
tools/             netcheck (analisador da RNF-30); parity-dumper (plugin de
                   dev do Obsidian, não é produto)
scripts/           gates e utilitários PowerShell — ver Comandos
.claude/           skills, workflows e settings do projeto (fonte única;
                   .agents/ é espelho gerado e não versionado)
.superpowers/sdd/  briefs e ledger
```

Grafo de dependências, acíclico e conferido em revisão:
`mcpsrv → service → {index, search, writer, vault, parser}`; `index → {parser,
vault}`; `search → parser`; `writer → vault`; `watcher → {vault, index, search}`;
`doctor → {vault, config, ipc, daemon}`. `parser` e `vault` são folhas.
**Aresta nova precisa de justificativa — folhas não ganham import.**

---

## Comandos

```bash
pwsh -File scripts/verify.ps1              # o gate: 14 etapas, para no primeiro erro
pwsh -File scripts/build.ps1               # build com versão via ldflags
pwsh -File scripts/test_orphans.ps1        # os quatro cenários de encerramento
pwsh -File scripts/mutate.ps1 ...          # prova de mutação — ver papeis/testador.md
pwsh -File scripts/audit_reports.ps1 <N>   # auditoria do próprio relatório
pwsh -File scripts/check_briefs.ps1 <a> <b>  # antes de despachar tarefas
pwsh -File scripts/sdd.ps1 status          # ledger + git
pwsh -File scripts/measure.ps1 -Vault <x>  # RNF-01 e RNF-07 contra um cofre real
pwsh -File scripts/gen_vault.ps1 -Notes 5000 -Seed 42 -Out <x>   # cofre de bench
```

**`verify.ps1` verde é obrigatório antes de qualquer commit.** Ele existe porque
a lista solta convida a rodar três dos cinco: cobre build, `go test -race`, tetos
de latência, `go vet` nos três GOOS, `gofmt`, `golangci-lint` (Windows e Linux),
`check_net`, `check_tool_params`, `check_doc_refs` e `check_readme_anchors`.
Aceita `-SkipCross` e `-SkipNet` para iteração rápida; o gate roda tudo.

---

## Regras que não são negociáveis

**stdout pertence ao JSON-RPC.** Todo log vai para stderr via `log/slog`. Um
`fmt.Println` em código alcançável de `serve` corrompe a sessão — o sintoma é o
servidor sumir do host sem erro nenhum. `doctor` e `version` imprimem em stdout
**de propósito**: são comandos CLI, não servidores. A distinção merece comentário
onde aparece.

**Nenhum socket que saia da máquina (RNF-30).** Nenhum pacote sob `internal/` ou
`cmd/` importa `net/*` — `net/http` e `x/oauth2` chegam transitivamente pelo SDK,
e isso é esperado. O pacote `net` em si é permitido **só** para
`net.Dial`/`net.Listen` com a rede na constante literal `"unix"` — o IPC local do
daemon, reaberto com autorização do dono em 2026-08-05 (Task 90). Rede vinda de
variável é recusada por `tools/netcheck`. Redação normativa em `PRD.md` §6.4.

**Nenhum tipo do SDK MCP cruza para fora de `internal/mcpsrv`.**
`internal/service` fala tipos de domínio. Torna migração de protocolo mudança de
um pacote só — e o protocolo já quebrou compatibilidade várias vezes.

**`ctx` onde há espera real.** Funções que podem **bloquear** recebem `ctx` e o
respeitam. Leitura de env var, resolução de caminho e cálculo em memória não
recebem. Quando o parâmetro existe só por consistência de assinatura,
nomeie-o `_`. `lifecycle.Shutdown` é o caso especial documentado: recebe `ctx` e
descarta o cancelamento via `context.WithoutCancel`, porque o context raiz já
está cancelado quando ela roda.

**Anexo é indexado por nome, nunca lido. Arquivo somente-nuvem nunca é aberto.**
Abrir dispara download síncrono. E **quem roda antes do guarda precisa do mesmo
guarda**.

**Uma conta por regra.** Toda chave derivada, todo caminho derivado, toda decisão
de formato passa por **uma** função — inclusive nos pontos que já estavam certos.

**Código de plataforma atrás de build tag, em arquivo separado.** Nunca
`if runtime.GOOS ==` dentro de lógica compartilhada.

**Saída de console em ASCII puro:** `[OK]`, `[*]`, `[!]`, `[i]`, `[...]`.

**Sem `helpers.go`, `utils.go`, `common.go`.**

**Nunca `git checkout`, `git restore`, `git stash`, `git clean` nem `git reset`.**
Há trabalho não commitado neste repositório o tempo todo, e um subagente já
destruiu trabalho exatamente assim.

**Nunca `go mod tidy`.** Várias deps estão fixadas sem importador ainda —
`goldmark`, `yaml.v3`, `x/text`. `tidy` removeria todas, junto com o pin do SDK
MCP, que é decisão fechada (PRD D6). Se faltar entrada em `go.sum`, use
`go get <caminho-do-pacote>@<versão-fixada>` — caminho do **pacote**, não do
módulo. Piso é `go 1.25.0`, forçado por `go-sdk@v1.5.0`.

**Commits em Conventional Commits, em inglês.**

---

## Quando uma tarefa está pronta

Oito tarefas deste projeto foram entregues como concluídas sem terem sido. Cada
regra abaixo veio de uma dessas.

- **Não escreva número que você não mediu.** Se não mediu, escreva **"não
  medido"** — ninguém vai brigar com isso.
- **Não afirme estado que você não verificou.**
- **Um teste que não pode falhar é pior que teste ausente.** Antes de dizer que
  testou: apague a regra, rode, confirme que um teste nomeia a falha, restaure.
- **Prova de mutação escrita no condicional não é prova.** Prova real está no
  passado e traz a saída colada.
- **Schema que promete e código que ignora é pior que parâmetro ausente.**
- **Campo de API com valor fixo mente sempre.**
- **Não deixe sua deliberação no código.**
- **Escopo não encolhe em silêncio.** `BLOCKED` com o motivo é resposta melhor
  que uma entrega que parece completa.
- **Registre no ledger antes de dizer que acabou.**
- **O relatório é o entregável, não o resumo dele.** "Testes passam" não é
  evidência; a saída do teste é.

O detalhe de cada uma, com o defeito que a originou, está em
[`docs/ARMADILHAS.md`](docs/ARMADILHAS.md) e nos documentos de papel.

---

## Editando estas instruções

Regra nova entra **com o defeito concreto que a originou** — regra sem história é
preferência. Conteúdo novo vai para o documento do papel a que pertence, ou para
`ARMADILHAS.md`; este arquivo só cresce quando surge uma regra que vale para
todos os papéis.

Plano e código não podem divergir: mudança de nome, contrato ou comportamento
atualiza `docs/`, o plano e os briefs no mesmo PR. Depois de editar qualquer
`.md`, valide o encoding:

```bash
python -c "open('CLAUDE.md',encoding='utf-8').read()" && echo "[OK] UTF-8 valido"
```
