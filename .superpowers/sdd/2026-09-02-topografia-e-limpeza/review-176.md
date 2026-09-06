# Review — Task 176

## Progresso

- 11:08 -- Lidos `task-176-brief.md`, `task-176-report.md` e `review-6b70721..8a11cec.diff`.
- 11:08 -- Lidos `internal/boot/indice.go`, `cmd/gobsidian/cli_log.go`, `internal/config/config.go` (trecho de `CacheDir`).
- 11:08 -- `go build ./...` -- OK.
- 11:08 -- `grep -rn '"vault", ""' cmd/gobsidian/*.go` -- so `flags.go`, igual ao relatorio.
- 11:08 -- `go vet ./cmd/gobsidian/` -- OK.
- 11:08 -- `go test -race ./cmd/gobsidian/ -run TestIndexCmdJSONTrazOrigem -v` -- PASS.
- 11:09 -- `go test -race -count=1 ./cmd/gobsidian/...` -- PASS (suite inteira, sem cache).
- 11:09 -- `pwsh -File scripts/mutate.ps1 -Path cmd/gobsidian/index.go -Anchor 'Origin:     origem,' -Replacement 'Origin:     "build",' -Test TestIndexCmdJSONTrazOrigem -Package ./cmd/gobsidian/` -- exit 0, rodado por mim, nao copiado do relatorio.
- 11:09 -- `git status --porcelain`, `git show 022d2ef --stat`, `git show 8a11cec --stat`, `git log --oneline -5` -- dois commits confirmados, na ordem e com os assuntos exigidos; nenhum arquivo do escopo desta task fora dos dois commits.
- 11:09 -- Lido `TestIndexEInspectNaoAceitamFlagsQueIgnoram` (linha 205) inteiro -- confirmado que continua testando algo real apos o refactor (nem `flagsDeCofre` nem `flagsDeCache` registram `read-only`/`debounce-ms`/`max-results`).
- 11:10 -- Diff pre-existente das quatro flags em `search.go`, `serve.go`, `daemon.go`, `doctor.go`, `index.go`, `inspect.go` no commit-base `6b70721` (antes da Task 176) -- texto de ajuda identico em todos os pontos onde a flag ja existia; a afirmacao do relatorio ("nenhum texto divergia") confere.
- 11:12 -- Lidos `cmd/gobsidian/index.go` e `cmd/gobsidian/inspect.go` completos (pos-mudanca) e `docs/OPERACAO.md:2728-2762`, `README.md` trecho de flags e trecho novo sobre `origin`.
- 11:13 -- UTF-8 validado com `python -c "open(...,encoding='utf-8').read()"` em `README.md`, `docs/OPERACAO.md` e `task-176-report.md`.
- 11:14 -- `pwsh -File scripts/verify.ps1 -SkipCross -SkipNet` rodado por mim -- verde, mesmos 6 skips de plataforma que o relatorio lista.

## Spec compliance

**Veredito: APPROVED**

| Step | Exigido | Entregue | OK? |
|---|---|---|---|
| 1 | Teste `TestIndexCmdJSONTrazOrigem` criado, FAIL confirmado antes de qualquer mudanca | Criado, texto literal do brief; FAIL real colado (`unknown flag: --cache-dir`, nao "origin vazio" -- diferenca justificada e correta: a flag nao existia ainda) | Sim |
| 2 | `index`/`inspect` via `boot.AbrirIndice`; `Origin` no JSON; `con.Item("Origem: %s", origem)` no console de `index` | Feito nos dois arquivos; `inspect` corretamente descarta a origem (`idx, _, err`) porque nao tem "resumo" para anexar o campo | Sim |
| 3 | `flags.go` com `flagsDeCofre`/`flagsDeCache`; seis arquivos trocam registro manual pela chamada | Criado exatamente como o brief especifica; `doctor` so `flagsDeCofre`, confirmado por leitura do diff e por `grep` | Sim |
| 4 | `--cache-dir` com `t.TempDir()` nos dois testes de CLI existentes | Feito nos 4 `SetArgs` (`TestIndexCmd_StdoutAndJSON` x2, `TestInspectCmd_StdoutAndJSON` x2) | Sim |
| 5 | Prova de mutacao na ancora `Origin: origem,` (ajustada a `Origin:     origem,` pos-gofmt) | Rodada pelo implementador e por mim de forma independente; exit 0 nas duas vezes, mesma falha (`origin = "build", quero "cache"`) | Sim |
| 6 | README + OPERACAO.md documentam `--cache-dir`/`--log-level` em `index`/`inspect` e o campo `origin` | Feito; tabela do README atualizada nas duas linhas certas; secao nova em OPERACAO.md no mesmo estilo/local da secao da Task 175, sem repetir fatos que ja moram no README | Sim |
| 7 | `verify.ps1` verde; dois commits, nesta ordem, com estas mensagens | Dois commits confirmados por `git log`/`git show`; `verify.ps1` verde (rodado por mim de novo, `-SkipCross -SkipNet`) | Sim |

Correcoes do orquestrador (nao sao defeito): nome real do teste de flags (`TestIndexEInspectNaoAceitamFlagsQueIgnoram`), alinhamento de gofmt na ancora, falha RED real ("unknown flag" em vez de "origin vazio") -- todas batem com o que o relatorio descreve e com o que eu conferi.

`[sem-doc]` no assunto do commit 2: e a valvula ja estabelecida no projeto (5 commits anteriores a usam) para um commit estrutural puro sem mudanca de doc/comportamento -- uso correto aqui, ja que a doc e o comportamento foram no commit 1.

## Code quality

**Veredito: APPROVED**

- `boot.AbrirIndice` (Task 174) ja trata o caso de erro ao gravar o cache: `log.Warn` e segue, nunca falha a chamada por causa disso -- `index.go`/`inspect.go` nao precisavam (e nao tem) tratamento extra para "diretorio de cache nao gravavel".
- `--cache-dir` vazio: `config.Load` preenche o default (`defaultCacheDir`, hash do caminho do cofre) antes de `AbrirIndice` ver o valor -- mesma conta que `search`/`serve` ja usam. Nao ha caminho onde `index`/`inspect` fiquem com `CacheDir` vazio de fato.
- Nenhum campo de valor fixo: `Origin` vem de `origem`, que vem do retorno real de `boot.AbrirIndice` -- nao ha string literal na struct populada de dados.
- `TestIndexEInspectNaoAceitamFlagsQueIgnoram` continua testando algo real: nem `flagsDeCofre` nem `flagsDeCache` registram `read-only`/`debounce-ms`/`max-results`, entao o teste ainda travaria uma regressao se alguem reintroduzisse essas flags.
- `flags.go`: nome e conteudo nao violam a regra "sem helpers.go/utils.go/common.go" (`docs/ESTRUTURA.md` linha ~257-261) -- a regra veta arquivos que sao um despejo por categoria sintatica; `flags.go` tem uma responsabilidade nomeada e unica (registrar os dois grupos de flags compartilhadas), e "arquivos por operacao" e exatamente o padrao que o resto de `cmd/gobsidian` ja segue (`index.go`, `search.go`, etc.).
- Import novo (`boot`) em `index.go`/`inspect.go`: justificado -- e o mesmo pacote que `search.go` (Task 175) e `serve.go` ja importam; nao e aresta nova no grafo do CLAUDE.md porque `cmd/gobsidian` ja importava `boot` transitivamente pela mesma familia de arquivos.
- `TestIndexCmdJSONTrazOrigem` prova cache vs build de verdade, e nao depende de timing: a segunda chamada usa o mesmo `--vault`/`--cache-dir` da primeira, e `carregarIndiceDoCache` decide por `VerifyFreshness` (contagem/tamanho/mtime dos arquivos), nao por relogio -- entre as duas chamadas do teste o cofre nao muda, entao a segunda sempre bate fresh. A prova de mutacao confirma que a asserção da segunda chamada de fato depende do valor de `origem` e nao de outra coisa.
- Nenhum comentario obsoleto encontrado dizendo que `index`/`inspect` constroem em processo -- os comentarios que sobraram ("index e um subcomando de CLI... escrita em stdout de proposito") continuam verdadeiros e nao mencionam a forma de abrir o indice.

## Findings

**N1** (nit) -- `cmd/gobsidian/index.go` e `cmd/gobsidian/inspect.go`: os dois arquivos duplicam o mesmo bloco `vault.New` + `loggerDeCLI` + `boot.AbrirIndice` + o comentario "e um subcomando de CLI... escrita em stdout de proposito" + o bloco `json.MarshalIndent`+`Fprintln`. Essa duplicacao ja existia antes desta task (e o mesmo padrao usado em `search.go`) e a Task 176 nao a piora nem a introduz -- so a repete nos dois arquivos que ja seguiam a convencao "arquivo por operacao". Extrair isso colidiria com a regra de nao ter `helpers.go`. Registrado como observacao, nao como bloqueio: se um dia o padrao se repetir uma quarta vez, vale nomear a operacao comum (ex.: `abrirCLI(cmd, flags) (*index.Index, string, *slog.Logger, error)`) em vez de uma quinta copia.

**N2** (nit) -- `docs/OPERACAO.md`, secao nova da Task 176: a frase "As quatro flags que `index`, `inspect`, `search`, `serve`, `daemon` e `doctor` registravam" lista seis subcomandos mas fala de "quatro flags"; fica correta so porque a proxima oracao esclarece que `--cache-dir`/`--log-level` sao "em quem le indice" (excluindo `doctor`) -- mas um leitor apressado pode ler a primeira oracao como "todos os seis registravam as quatro" e achar que `doctor` tinha `--cache-dir`. Nao e factualmente errado (a frase se corrige na mesma sentenca), so exige atencao. Sem acao necessaria.

Nenhum finding blocking ou should-fix.

## Verified claims

- `go build ./...` -- limpo.
- `grep -rn '"vault", ""' cmd/gobsidian/*.go` -- so `cmd/gobsidian/flags.go:13`.
- `go vet ./cmd/gobsidian/` -- limpo.
- `go test -race ./cmd/gobsidian/ -run TestIndexCmdJSONTrazOrigem -v`:
  ```
  === RUN   TestIndexCmdJSONTrazOrigem
  --- PASS: TestIndexCmdJSONTrazOrigem (0.11s)
  PASS
  ok  	github.com/jonyd/gobsidian/cmd/gobsidian	2.585s
  ```
- `go test -race -count=1 ./cmd/gobsidian/...`:
  ```
  ok  	github.com/jonyd/gobsidian/cmd/gobsidian	3.810s
  ```
- `pwsh -File scripts/mutate.ps1 -Path cmd/gobsidian/index.go -Anchor 'Origin:     origem,' -Replacement 'Origin:     "build",' -Test TestIndexCmdJSONTrazOrigem -Package ./cmd/gobsidian/`:
  ```
  --- FAIL: TestIndexCmdJSONTrazOrigem (0.11s)
      index_origem_test.go:42: segunda execucao: origin = "build", quero "cache"
  FAIL
  FAIL	github.com/jonyd/gobsidian/cmd/gobsidian	1.583s
  FAIL
  ----------------------------------------------------------------------
  [OK] cmd/gobsidian/index.go restaurado byte a byte (SHA-256 confere).

  [OK] O teste REPROVOU com a regra mutada -- a regra esta verificada.
  EXIT=0
  ```
- `git log --oneline -5`:
  ```
  8a11cec refactor(cli): register shared vault flags once [sem-doc]
  022d2ef feat(cli): index and inspect load from the cache
  6b70721 docs(sdd): record the Task 175 review and its package
  8d3626e feat(cli): search loads the index and the search cache like serve does
  ebf29ad docs(sdd): record the Task 174 review, the fix round and the orphans gate logs
  ```
- `git show 022d2ef --stat` / `git show 8a11cec --stat`: escopo de arquivos de cada commit confere com o que o relatorio descreve (comportamento+doc no 1, estrutura pura no 2).
- `pwsh -File scripts/verify.ps1 -SkipCross -SkipNet`:
  ```
  [...] 3. contagem de testes pulados
  [!] 6 testes pulados
       --- SKIP: TestAjudanteSeguraTrava (0.00s)
       --- SKIP: TestListenRestringePermissaoUnix (0.00s)
       --- SKIP: TestSignalCancelsContext (0.10s)
       --- SKIP: TestPerfilDeHeapServindo (0.00s)
       --- SKIP: TestWriteAtomicPreservaOModoDoAlvo (0.00s)
       --- SKIP: TestNew_FailsOnUnwatchablePath (0.01s)
  ...
  [OK] Bateria completa. Pode commitar.
  ```
- `python -c "open('README.md',encoding='utf-8').read()"`, mesmo para `docs/OPERACAO.md` e `task-176-report.md` -- todos OK, sem excecao.
- `git status --porcelain` -- so arquivos preexistentes do dono e artefatos de outras tasks/relatorios (`commit-176-*.txt` incluidos, seguindo o padrao ja usado por `commit-169-*.txt` etc.); nada do escopo desta task ficou fora dos dois commits.
