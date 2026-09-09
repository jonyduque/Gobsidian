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
| [`docs/segregacao.md`](docs/segregacao.md) | Como o código se agrupa por função e por proximidade; os candidatos a extração, medidos, com decisão pendente |
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
                   em-processo; a montagem que serve e daemon compartilham
                   mora em internal/boot. index, inspect e search também
                   passam por lá para abrir o índice (boot.AbrirIndice; search
                   também chama boot.PrepararBusca), sem montar watcher nem
                   Service
internal/
  config/          struct de configuração, flags cobra, defaults, VaultKey
  lifecycle/       stdin-eof, sinais, vigília do PID pai, shutdown com orçamento
  vault/           raiz, caminho canônico e confinamento, walk com exclusões,
                   EOL, detecção de somente-nuvem, temp+sync+rename atômico
                   (ReplaceFile/WriteAtomic) e varredura de temporários órfãos
  vaulttest/       apoio a teste: condições de ambiente do Windows; só _test.go
                   importa
  parser/          goldmark + extensões [[wikilink]], ^blockid, #tag,
                   campo::inline; headings com offsets de byte; candidatos a
                   título (negrito, setext) que NUNCA viram Heading
  index/           Note, RWMutex, build com worker pool, update incremental,
                   backlinks, resolve, alias, query; chave.go é a conta única
                   das chaves derivadas (NFC + caixa)
  search/          analyzer, índice invertido base/delta, BM25, trecho,
                   cache binário formato 6 + arena mmap
  watcher/         fsnotify, debounce, filtro de relevância, apply,
                   rename por hash, reconciliação pós-overflow
  writer/          lock por caminho canônico, edição sob heading/bloco,
                   reescrita de link, diff
  service/         fachada das tools em tipos de domínio; erros em errors.go
  mcpsrv/          ÚNICO pacote onde tipos do SDK de MCP existem
  console/         marcadores ASCII e cor decidida pelo destino de saída
  boot/            monta cofre, índice, busca, watcher e Service; a vigília do
                   host (pipe espelhado + lifecycle) e os passos de shutdown
                   compartilhados; serve, ponte e daemon chamam
  ipc/             transporte local: socket, saudação, handshake
  daemon/          N conexões sobre um índice; spawn; posse por trava do
                   kernel (flock / LockFileEx), nunca por arquivo com PID
  doctor/          diagnóstico de ambiente e do runtime do daemon
  instalar/        o instalador dentro do binário: presença por trava de
                   kernel, trava global de instalação, limpeza de lixo órfão,
                   manifesto, PATH e a sequência de troca do binário
  hosts/           os 9 hosts MCP: onde mora o config de cada um e como fundir
                   a entrada preservando o resto do JSON do usuário
  selfupdate/      release, SHA-256 e download; ÚNICO pacote com net/http
  text/            normalização
docs/              normativa, papéis, história, wiki
testdata/          golden files do parser, cofre pequeno, corpus de paridade
bootstrap/         install.sh, install.ps1 e install.nu — baixam o executável
                   e o rodam, e nada mais (126 linhas, contra as 1.819
                   apagadas: install.ps1 com 729 e installer/ com 1.090)
tools/             netcheck (analisador da RNF-30); parity-dumper (plugin de
                   dev do Obsidian, não é produto)
scripts/           gates e utilitários PowerShell — ver Comandos
.claude/           skills, workflows e settings do projeto (fonte única;
                   .agents/ é espelho gerado e não versionado)
.superpowers/sdd/  briefs e ledger
```

Grafo de dependências, acíclico e **re-extraído dos imports de produção em
2026-09-06** — `GOOS=windows go list -f '{{.Imports}}'` pacote a pacote, que NÃO
enxerga arquivo `_test.go`. Três linhas mudaram desde 2026-09-02: a do `writer`,
a do `boot`, que é nova — e a do `boot` ganhou `lifecycle` no mesmo dia
(Task 177) —, e a do `service`, que ganhou `text` em 2026-09-06 (Task 183).
As justificativas estão logo abaixo do bloco:

```
text  vault  config  console  lifecycle      folhas
parser   → text
ipc      → config
writer   → parser, text, vault
index    → parser, text, vault
search   → index, parser, text, vault
watcher  → index, search, vault
service  → index, parser, search, text, vault, writer
mcpsrv   → config, index, parser, service, vault
boot     → config, index, lifecycle, search, service, vault, watcher
daemon   → config, ipc, lifecycle, mcpsrv
doctor   → config, daemon, ipc, vault
hosts      → (folha)
selfupdate → (folha)
instalar   → config, daemon, hosts, ipc, search
```

Os três últimos são de 2026-09-08 (plano do instalador, tasks 195–201), e as
justificativas de cada aresta:

`daemon → lifecycle` é de 2026-09-09 e a justificativa é a mesma de
`boot → lifecycle`: **o mesmo andaime escrito duas vezes**. `lifecycle.Esperar`
registra no log que uma espera de encerramento começou e quanto levou, e as
cinco esperas do produto — a de conexões em voo dentro de `daemon.Run`, e as
quatro em `cmd/gobsidian` — precisam da mesma linha, com o mesmo formato. Sem
ela, a última linha do log não diz qual espera não voltou, que é exatamente o
que faltou para determinar a causa do PID 42856 em 2026-09-07. `lifecycle`
continua folha, e continua sem saber quando encerrar: quem decide é quem chama.

`hosts` é **folha**: ele recebe a raiz do sistema de arquivos e o comando a
registrar por parâmetro. Escreve JSON genérico e não sabe o que é uma tool —
por isso não importa `mcpsrv` nem `service`, e não vai importar.

`selfupdate` também é **folha** — nenhuma aresta interna. Ele é o **único
pacote do produto com `net/http`**, sob a segunda exceção nomeada da RNF-30
(PRD §6.4, decisão D-13 do dono em 2026-09-08), e a lista de hosts que pode
alcançar é uma constante dele. `serve`, `daemon` e as tools **não o importam**:
servir um cofre nunca toca a rede, e é `cmd/gobsidian/update.go` o único
chamador.

(A primeira redação deste bloco escrevia `selfupdate → config`. Os imports
reais dizem folha; `GOOS=windows go list -f '{{.Imports}}'` foi rodado antes de
o parágrafo ficar.)

`instalar → daemon` reusa `TentarTravar`, a primitiva de trava de kernel que já
existe — a presença de processos e a trava global são a MESMA primitiva
respondendo duas perguntas diferentes. Escrever uma segunda trava seria duas
contas da mesma regra. `instalar → ipc` é o diretório de runtime
(`ipc.RuntimeDir`) e a sonda de ouvinte (`ipc.AlguemEscuta`), as duas contas do
`ipc`. `instalar → search` é `LerCabecalhoDoCache`: decidir se um cache ficou
órfão exige o caminho do cofre, que só o cabeçalho do cache guarda, e o formato
dele é de `search`. `instalar → config` é `VaultKey` e a raiz do cache.

`instalar` é importado **só** por `cmd/gobsidian`, como `boot`. E `doctor` NÃO
importa `instalar`: quem relata processos e lixo é o comando, não o pacote de
diagnóstico — `doctor` responde sobre o AMBIENTE.

`boot` é de 2026-09-06 (Task 174) e nasceu sem aresta nova nenhuma: as seis
eram exatamente as que `cmd/gobsidian` já tinha. A sequência de boot — cofre,
varredura de temporários, índice de metadados, índice de busca, watcher,
Service — vivia em `cmd/gobsidian`, onde nenhum teste de pacote a alcançava.

`boot → lifecycle` é a sétima, de 2026-09-06 (Task 177), e a justificativa é o
mesmo **andaime escrito duas vezes**: pipe, `mirrorReader` e `lifecycle.New`,
nessa ordem — a ordem é o que faz o EOF do host chegar ao monitor de stdin —
estavam em `serveEmProcesso` e em `servePonteRemota`, e os passos `close-pipe` e
`watcher` estavam nos três pontos de saída. `boot.VigiarHost` e
`(*Componentes).PassoWatcher` são a conta única disso. `lifecycle` continua
folha. `boot` continua sem importar `mcpsrv`: quem monta não decide como o host
conversa. E não decide **quando** encerrar — `lifecycle.Shutdown` é chamada por
`cmd/gobsidian`, com os passos que só quem serve conhece (`in-flight`,
`half-close`, `close-conn`). E nenhum pacote de domínio importa `boot` — só
`cmd/gobsidian`.

`writer → text` é de 2026-09-06 (Task 169) e a justificativa é **uma conta por
regra**: a trava por caminho do `writer` e a chave `lowerPath` do `index` são a
mesma derivação, e eram duas — a do `writer` só baixava caixa, então o índice
tratava as duas grafias Unicode de um nome como uma nota e o locker as tratava
como dois arquivos. A conta mudou-se para `text.ChaveDeCaminho` porque `writer`
não importa `index` e não vai importar. `text` continua folha: ganhou
`path/filepath`, que é stdlib.

`service → text` é de 2026-09-06 (Task 183): o filtro `prefix` de
`vault_broken_links` compara caminho de origem com o prefixo pedido, e
comparar caminho é `text.ChaveDeCaminho` — a mesma conta do `writer`
(Task 169) e do `index`. Uma comparação local faria `Sub/` e `sub/` serem
duas pastas aqui e uma lá. `text` continua folha; o grafo continua acíclico.

Quatro arestas existem **só em teste**, e ficam fora do grafo acima de
propósito — teste pode montar o mundo inteiro sem que isso vire acoplamento do
produto:

```
mcpsrv  → search, watcher     (só em _test)
daemon  → service, vault      (só em _test)
```

`internal/vaulttest` é o pacote de apoio a teste: ele importa `vault` — e só
`vault` —, e essa aresta **é do próprio pacote**, então `go list -f
'{{.Imports}}' ./internal/vaulttest` a mostra como importaria qualquer outra.
Isso vale **com `GOOS=windows`**: o corpo real mora nos arquivos `_windows.go`,
e em `GOOS=linux` o mesmo comando devolve só `[testing time]`, porque os
`_other.go` apenas fazem `t.Skip` (medido em 2026-09-05). O
que é exclusivo de teste são os imports **de** `vaulttest`: nenhum arquivo de
produção o importa, só arquivos `_test.go` de `index`, `search`, `service`,
`vault`, `watcher`, `daemon`, `ipc`, `boot` e `cmd/gobsidian` — a lista saiu de
`git grep -l "internal/vaulttest" -- '*_test.go' | cut -d/ -f1-2 | sort -u`
(medido em 2026-09-06; `watcher` entrou na Task 164 e `boot` na Task 174). O
comando devolve uma décima linha, `internal/vaulttest`: é o teste externo do
próprio pacote, não um importador. Que não há aresta de produção também é medido, e é o outro
comando: `git grep -l "internal/vaulttest" -- '*.go' ':!*_test.go'` volta vazio.
Por isso ele fica fora do grafo de produção acima, e por isso não pode ganhar
import de `index`, `search`, `service`, `parser` ou `boot`: seria ciclo no teste
externo desses pacotes.

```
vaulttest → vault             (pacote só de teste; ninguém em produção o importa)
```

A versão anterior deste bloco somava as duas listas numa só e, com isso,
atribuía ao `daemon` um conhecimento de `service` e de `vault` que ele não tem —
ele fala com `mcpsrv` e mais nada do domínio. Antes dela, outra omitia `text`
inteiro e chamava `parser` de folha quando ele já importava `text`.

**Aresta nova precisa de justificativa — folha não ganha import.** E o parágrafo
que descreve o grafo não vale mais que os imports: dizer "conferido" não é
conferir, e as duas versões anteriores diziam.

---

## Comandos

```bash
pwsh -File scripts/verify.ps1              # o gate: 15 etapas, para no primeiro erro
pwsh -File scripts/build.ps1               # build com versão via ldflags
pwsh -File scripts/test_orphans.ps1        # os quatro cenários de encerramento
pwsh -File scripts/mutate.ps1 ...          # prova de mutação — ver papeis/testador.md
pwsh -File scripts/audit_reports.ps1 <N>   # auditoria do próprio relatório (N ou nome: final-fix)
pwsh -File scripts/check_gates.ps1         # o hook e o auditor contra os bypasses que já aceitaram
pwsh -File scripts/check_briefs.ps1 <a> <b>  # antes de despachar tarefas
pwsh -File scripts/sdd.ps1 status          # ledger + git
pwsh -File scripts/measure.ps1 -Vault <x>  # RNF-01 e RNF-07 contra um cofre real
pwsh -File scripts/gen_vault.ps1 -Notes 5000 -Seed 42 -Out <x>   # cofre de bench
```

**`verify.ps1` verde é obrigatório antes de qualquer commit.** Ele existe porque
a lista solta convida a rodar três dos cinco: cobre build, `go test -race`, a
contagem de testes pulados, tetos de latência, `go vet` nos três GOOS, `gofmt`,
`golangci-lint` (Windows e Linux), `check_net`, `check_tool_params`,
`check_doc_refs`, `check_readme_anchors` e `check_gates`. A contagem de pulados **informa e não
reprova** — há skip legítimo, como o de `vaulttest` fora do Windows —, mas um
teste que pula não cobre nada, e o de paridade pulava sem que o gate dissesse.
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
- **Gate que aceita o que devia recusar é pior que gate ausente.** O hook de
  pré-commit aceitou `[sem-doc]` num comentário de shell e num `--amend`; o
  auditor aceitou `# comment` colado como seção. Regra de gate nova entra com
  três casos em `check_gates.ps1`: o que recusa, o que aceita, o inverso.

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
