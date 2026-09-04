# Task 159 — `internal/vaulttest`: um handle exclusivo que prova que trava, e um prazo unico

Status: **DONE_WITH_CONCERNS** (as ressalvas estao em "Desvios do brief" e "Fora do escopo"; nenhuma bloqueia)

Commit: **`c81b2b8`** — 29 arquivos, +775 / -338

## Progresso

- 01:53 Li o brief e `docs/papeis/implementador.md`; mapeei `boundedWait` (5 arquivos)
- 01:55 Criei `internal/vaulttest` (7 arquivos); testes do proprio pacote PASSAM
- 01:56 Prova de mutacao do helper feita (share 0 -> FILE_SHARE_READ => FAIL nomeado), restaurada
- 01:58 Migrados search (2 helpers), index (3 helpers), service (2 arquivos + guardas removidas)
- 01:59 Step 4: as duas travas de move_atomico removidas uma a uma => FAIL nomeado nas duas; restauradas
- 02:02 boundedWait -> vaulttest.Prazo em 5 arquivos; zero ocorrencias restantes. Migradas 3 copias
  a mais que o brief nao listava (build_cloudonly, snippet_cache, cloudonly_replace)
- 02:05 go test -race nos 8 pacotes: ok. GOOS=linux e GOOS=darwin go vet: exit 0
- 02:07 Docs: CLAUDE.md (grafo), ESTRUTURA.md (arvore), testador.md (secao nova). UTF-8 validado
- 02:12 verify.ps1 verde na segunda rodada (a primeira reprovou so no teto de latencia, por carga)
- 02:20 Relatorio escrito e commit feito

---

## Arquivos criados

| Arquivo | O que carrega |
|---|---|
| `internal/vaulttest/doc.go` | por que o pacote existe |
| `internal/vaulttest/exclusivo_windows.go` | `TravarExclusivo`, `TravarDiretorioExclusivo`, `abrirExclusivo` |
| `internal/vaulttest/exclusivo_other.go` | `//go:build !windows`, os dois com `t.Skip` |
| `internal/vaulttest/somentenuvem_windows.go` | `MarcarSomenteNuvem` com a prova de `vault.IsCloudOnly` |
| `internal/vaulttest/somentenuvem_other.go` | `//go:build !windows`, `t.Skip` |
| `internal/vaulttest/prazo.go` | `const Prazo = 5 * time.Second` |
| `internal/vaulttest/exclusivo_windows_test.go` | `package vaulttest_test`, os dois testes do brief |

Transcritos do brief sem variante. Unico ajuste mecanico: em `exclusivo_other.go` e
`somentenuvem_other.go` acrescentei um comentario de doc a cada funcao (o brief so
comentava a primeira; `golangci-lint`/`revive` cobra doc em identificador exportado).

## Arquivos modificados

`cmd/gobsidian/boot_indice_busca_windows_test.go`, `cmd/gobsidian/ponte_test.go`,
`cmd/gobsidian/serve_test.go`, `internal/daemon/daemon_test.go`,
`internal/index/build_cloudonly_windows_test.go`, `internal/index/build_descarte_test.go`,
`internal/index/build_descarte_unix_test.go`, `internal/index/build_descarte_windows_test.go`,
`internal/index/classify_cloudonly_windows_test.go`,
`internal/index/replace_duas_fases_windows_test.go`, `internal/ipc/ipc_test.go`,
`internal/ipc/maxresults_handshake_test.go`, `internal/search/cloudonly_update_windows_test.go`,
`internal/search/snippet_cache_windows_test.go`,
`internal/service/cloudonly_replace_windows_test.go`,
`internal/service/erro_engolido_windows_test.go`, `internal/service/move_atomico_windows_test.go`,
`internal/vault/walk_raiz_windows_test.go`, `CLAUDE.md`, `docs/ESTRUTURA.md`,
`docs/papeis/testador.md`.

Saldo: **+184 / -338 linhas** (`git diff --stat`), 21 arquivos, mais os 7 criados.

---

## Helpers apagados, com `arquivo:linha` original

| # | Helper | Origem | Virou |
|---|---|---|---|
| 1 | `marcarSomenteNuvem` | `internal/search/cloudonly_update_windows_test.go:30` | `vaulttest.MarcarSomenteNuvem` |
| 2 | `travarExclusivo` | `internal/search/cloudonly_update_windows_test.go:50` | `vaulttest.TravarExclusivo` |
| 3 | `travaExclusiva` | `internal/service/erro_engolido_windows_test.go:25` | `vaulttest.TravarExclusivo` |
| 4 | `travaLeitura` | `internal/index/replace_duas_fases_windows_test.go:22` | `vaulttest.TravarExclusivo` |
| 5 | `lockFileForTest` (windows) | `internal/index/build_descarte_windows_test.go:10` | delega a `vaulttest.TravarExclusivo` — ver ressalva D2 |
| 6 | `marcarSomenteNuvem` | `internal/index/classify_cloudonly_windows_test.go:24` | `vaulttest.MarcarSomenteNuvem` |
| 7 | `marcarOffline` | `cmd/gobsidian/boot_indice_busca_windows_test.go:28` | `vaulttest.MarcarSomenteNuvem` |
| 8 | `travarDiretorioExclusivo` | `internal/vault/walk_raiz_windows_test.go:19` | `vaulttest.TravarDiretorioExclusivo` |
| 9 | `boundedWait = 2s` | `cmd/gobsidian/serve_test.go:19` | `vaulttest.Prazo` |
| 10 | `boundedWait = 5s` | `internal/daemon/daemon_test.go:23` | `vaulttest.Prazo` |
| 11 | `boundedWait = 3s` | `internal/ipc/ipc_test.go:23` | `vaulttest.Prazo` |

Alem dos helpers **nomeados**, tres blocos **inline** que faziam a mesma coisa
sairam (o brief nao os listava, mas as Verificacoes exigem zero `CreateFile(` e
zero `FILE_ATTRIBUTE_OFFLINE` executavel fora de `vaulttest`):

| Origem | O que era |
|---|---|
| `internal/index/classify_cloudonly_windows_test.go:55-73` | `CreateFile` exclusivo + guarda de leitura, escritos a mao |
| `internal/service/move_atomico_windows_test.go:109-119` | `CreateFile` exclusivo **sem** guarda nenhuma |
| `internal/service/erro_engolido_windows_test.go:84-91` | `SetFileAttributes(OFFLINE)` a mao |
| `internal/service/cloudonly_replace_windows_test.go:40-49 e 63-73` | OFFLINE + `CreateFile` a mao |
| `internal/index/build_cloudonly_windows_test.go:37-46` | `SetFileAttributes(OFFLINE)` a mao |
| `internal/search/snippet_cache_windows_test.go:39-48` | `SetFileAttributes(OFFLINE)` a mao |

Contagem final: **8 helpers nomeados + 3 constantes `boundedWait` + 6 blocos inline
= 17 copias colapsadas em um pacote.**

## Guardas condicionais removidas

**`internal/service/move_atomico_windows_test.go`** (`TestMoveNaoReportaSucessoComNotaDuplicada`).
Era:

```go
if origemExiste && destinoExiste && err == nil { t.Errorf("SUCESSO reportado com a nota DUPLICADA: …") }
if origemExiste && destinoExiste && err != nil { t.Logf("estado duplicado, mas o erro foi reportado: %v", err) }
if err == nil && origemExiste { t.Error("MoveNote devolveu nil mas a origem continua no disco") }
```

Ficou: `t.Fatalf` se `!origemExiste || !destinoExiste` (guarda de montagem, com o
estado medido), depois `t.Fatalf` incondicional se `err == nil`. As duas assercoes
originais eram a mesma implicacao (`err == nil` ⇒ defeito) e ambas sobrevivem. O
`t.Logf` do meio nao era assercao e saiu — o estado que ele imprimia agora e
**afirmado** pela guarda acima.

O estado foi **medido**, nao suposto (sonda temporaria, 2026-09-04, esta maquina):

```
SONDA: origemExiste=true destinoExiste=true err=a nota foi copiada para "destino.md" mas a
origem "origem.md" nao pode ser removida (…The process cannot access the file because it is
being used by another process.); a nota existe nos dois caminhos ate a origem ser liberada
```

**`internal/service/erro_engolido_windows_test.go`** (`TestDeleteToTrashNaoMenteQuandoORemoveFalha`).
Era:

```go
if errOrigem == nil && err == nil && res.Deleted { t.Errorf("Deleted=true com a nota ainda no caminho original…") }
if err != nil && !strings.Contains(strings.ToLower(err.Error()), ".trash") { t.Errorf("o erro nao explica…") }
```

Ficou: `t.Fatalf` se a origem sumiu (guarda de montagem), `t.Fatalf` se `err == nil`,
e a checagem do `.trash` agora **incondicional**. Nenhuma assercao perdida; as duas
deixaram de depender de o cenario ter se montado.

---

## Verificacao de referencias por helper apagado

**O MCP do `gopls` (`mcp__gopls__go_symbol_references`) nao esta no conjunto de
ferramentas deste agente** — confirmei com `ToolSearch`, que devolveu outras
ferramentas. Substitui por duas checagens que, em Go, nao podem deixar chamador
para tras:

1. `grep -rn "<identificador>" --include=*.go .` no repo inteiro, por helper, antes
   e depois de cada delecao.
2. Compilacao de **todos** os pacotes de teste: `go vet ./...` em `GOOS=windows`,
   `GOOS=linux` e `GOOS=darwin` — os tres exit 0. Um chamador nao migrado seria erro
   de compilacao, nao aviso. Isto ja pegou dois esquecimentos durante a tarefa
   (`ponte_test.go` sem o import, `move_atomico` com import ocioso na mutacao).

Estado final dos greps das Verificacoes:

```
$ grep -rn "CreateFile(" --include=*_test.go internal/ cmd/
(vazio)

$ grep -rn "boundedWait" --include=*.go .
(vazio)

$ grep -rn "FILE_ATTRIBUTE_OFFLINE" --include=*_test.go internal/ cmd/
internal/index/build_cloudonly_windows_test.go:27:   (comentario)
internal/search/snippet_cache_windows_test.go:21:    (comentario)
internal/vault/cloudonly_info_windows_test.go:25,41,42,49,81,82
```

`internal/vault/cloudonly_info_windows_test.go` e a excecao que o brief manda
manter: e `package vault` (interno) e importar `vaulttest` fecharia ciclo. Os
outros dois sao texto de comentario, nao chamada.

## `go list -deps`

```
$ go list -deps ./internal/vaulttest/ | grep gobsidian
github.com/jonyd/gobsidian/internal/vault
github.com/jonyd/gobsidian/internal/vaulttest

$ go list -f '{{.Imports}}' ./internal/vaulttest/
[github.com/jonyd/gobsidian/internal/vault golang.org/x/sys/windows os testing time]
```

`internal/vault` nao importa `internal/text` (conferido: `go list -f '{{.Imports}}'
./internal/vault/` devolve so stdlib mais `golang.org/x/sys/windows`), entao a
arvore fica em dois pacotes do projeto, como o brief exigia. `golang.org/x/sys`
ja estava em `go.mod` e ja e importado por producao — **nenhum `go mod tidy`,
nenhum `go get`, `go.mod`/`go.sum` intocados**.

---

## Prova 1 — o helper falha se a trava nao travar

Mutacao: em `exclusivo_windows.go`, `dwShareMode` de `0` para `windows.FILE_SHARE_READ`.

```
$ go test ./internal/vaulttest/ -run 'TestTravar' -v
=== RUN   TestTravarExclusivoBarraLeituraEDevolveNoCleanup
=== RUN   TestTravarExclusivoBarraLeituraEDevolveNoCleanup/travado
    exclusivo_windows_test.go:22: vaulttest: o handle exclusivo nao barrou a leitura de C:\Users\jonyd\AppData\Local\Temp\TestTravarExclusivoBarraLeituraEDevolveNoCleanup2492758218\001\a.md; a prova de 'nao abriu' seria vazia
--- FAIL: TestTravarExclusivoBarraLeituraEDevolveNoCleanup (0.00s)
    --- FAIL: TestTravarExclusivoBarraLeituraEDevolveNoCleanup/travado (0.00s)
=== RUN   TestTravarDiretorioExclusivoBarraListagem
=== RUN   TestTravarDiretorioExclusivoBarraListagem/travado
    exclusivo_windows_test.go:38: vaulttest: o handle exclusivo nao barrou a listagem de C:\Users\jonyd\AppData\Local\Temp\TestTravarDiretorioExclusivoBarraListagem2521590515\001\sub
--- FAIL: TestTravarDiretorioExclusivoBarraListagem (0.00s)
    --- FAIL: TestTravarDiretorioExclusivoBarraListagem/travado (0.00s)
FAIL
FAIL	github.com/jonyd/gobsidian/internal/vaulttest	0.551s
FAIL
```

Restaurado; depois da restauracao:

```
$ go test ./internal/vaulttest/ -run 'TestTravar'
ok  	github.com/jonyd/gobsidian/internal/vaulttest	0.527s
```

Observacao: a mutacao derrubou os **dois** testes, incluindo o de diretorio —
`FILE_SHARE_READ` no handle de diretorio tambem libera o `ReadDir`.

## Prova 2 (Step 4) — a guarda removida vale

O brief pedia comentar `vaulttest.TravarExclusivo(t, destino)` em
`move_atomico_windows_test.go`. **Nao existe trava sobre o destino nesse arquivo**
(ver desvio D1); as duas travas sao sobre a origem. Removi cada uma na vez dela.

**2a — a trava de `TestMoveNaoReportaSucessoComNotaDuplicada`** (`os.Open(origem)`,
que e a que a guarda `origemExiste && destinoExiste && err == nil` protegia):

```
$ go test ./internal/service/ -run 'TestMoveNaoReportaSucessoComNotaDuplicada' -v
=== RUN   TestMoveNaoReportaSucessoComNotaDuplicada
    move_atomico_windows_test.go:78: cenario invalido: com o handle aberto a nota devia ficar nos dois caminhos; origem existe=false, destino existe=true
--- FAIL: TestMoveNaoReportaSucessoComNotaDuplicada (0.01s)
FAIL
FAIL	github.com/jonyd/gobsidian/internal/service	0.821s
FAIL
```

Este e exatamente o ponto da tarefa: **antes**, sem trava, `origemExiste` era
`false`, as tres condicoes eram falsas, e o teste passava sem afirmar nada.

**2b — a trava de `TestMoveNaoReescreveCitantesAntesDeMoverOCorpo`**
(`vaulttest.TravarExclusivo`, migrada do `CreateFile` inline):

```
$ go test ./internal/service/ -run 'TestMoveNao' -v
=== RUN   TestMoveNaoReportaSucessoComNotaDuplicada
--- PASS: TestMoveNaoReportaSucessoComNotaDuplicada (0.01s)
=== RUN   TestMoveNaoReescreveCitantesAntesDeMoverOCorpo
    move_atomico_windows_test.go:121: MoveNote devolveu nil com a origem travada para leitura
--- FAIL: TestMoveNaoReescreveCitantesAntesDeMoverOCorpo (0.01s)
FAIL
FAIL	github.com/jonyd/gobsidian/internal/service	0.824s
FAIL
```

As duas restauradas; `grep -n MUTACAO internal/service/` devolve vazio.

---

## `go test -race` dos pacotes tocados

```
$ go test -race ./internal/vaulttest/ ./internal/search/ ./internal/service/ ./internal/index/ \
    ./internal/vault/ ./internal/daemon/ ./internal/ipc/ ./cmd/gobsidian/
ok  	github.com/jonyd/gobsidian/internal/vaulttest	2.143s
ok  	github.com/jonyd/gobsidian/internal/search	13.325s
ok  	github.com/jonyd/gobsidian/internal/service	58.159s
ok  	github.com/jonyd/gobsidian/internal/index	4.960s
ok  	github.com/jonyd/gobsidian/internal/vault	3.983s
ok  	github.com/jonyd/gobsidian/internal/daemon	7.619s
ok  	github.com/jonyd/gobsidian/internal/ipc	1.903s
ok  	github.com/jonyd/gobsidian/cmd/gobsidian	6.955s
```

`GOOS=linux go vet ./...` e `GOOS=darwin go vet ./...`: **exit 0, saida vazia** nos
dois (os `_other.go` compilam).

## `verify.ps1`

Primeira rodada reprovou **uma** etapa, a 3 (tetos de latencia), em
`TestRNF04SnippetConcurrencyLimit200` — p95 27,7 ms contra teto de 22 ms, em tres
rodadas. Rodado isolado logo em seguida, **passa** (p95 13,9 ms na segunda rodada
interna). E o caso que `docs/papeis/testador.md` ja registra: "teste sob carga
paralela pode estourar prazo sem que nada esteja errado" — a maquina vinha de
`go test -race` nos oito pacotes. O teste **nao toca** codigo nem teste que esta
tarefa alterou (o arquivo `internal/service/search_test.go` nao foi modificado).

Segunda rodada, com a maquina livre:

```
[...] 1. go build
[OK] go build
[...] 2. go test -race
[OK] go test -race
[...] 3. go test (tetos de latencia, sem -race)
[OK] go test (tetos de latencia, sem -race)
[...] 4. go vet (windows)
[OK] go vet (windows)
[...] 5. go vet (linux)
[OK] go vet (linux)
[...] 6. go vet (darwin)
[OK] go vet (darwin)
[...] 7. gofmt
[OK] gofmt
[...] 8. golangci-lint
[OK] golangci-lint
[...] 9. golangci-lint (linux)
[OK] golangci-lint (linux)
[...] 10. check_net (RNF-30)
[OK] check_net (RNF-30)
[...] 11. check_tool_params
[OK] check_tool_params
[...] 12. check_doc_refs
[OK] check_doc_refs
[...] 13. check_readme_anchors
[OK] check_readme_anchors

[OK] Bateria completa. Pode commitar.
```

Nota factual: o gate imprimiu **13** etapas numeradas, nao 14 como o `CLAUDE.md`
diz. Nao investiguei; anotado em "Fora do escopo".

## Encoding

```
$ python -c "open('CLAUDE.md',encoding='utf-8').read()"                 -> [OK]
$ python -c "open('docs/ESTRUTURA.md',encoding='utf-8').read()"          -> [OK]
$ python -c "open('docs/papeis/testador.md',encoding='utf-8').read()"    -> [OK]
$ python -c "open('.superpowers/.../task-159-report.md',encoding='utf-8').read()" -> [OK]
```

---

## Desvios do brief (todos deliberados, nenhum silencioso)

**D1 — Step 4 pedia comentar `vaulttest.TravarExclusivo(t, destino)` em
`move_atomico_windows_test.go`. Essa linha nao existe.** As duas travas do arquivo
sao sobre a **origem**, com semanticas diferentes de proposito:
`TestMoveNaoReportaSucessoComNotaDuplicada` usa `os.Open` (compartilha leitura,
bloqueia so a remocao — e o cenario da nota duplicada) e
`TestMoveNaoReescreveCitantesAntesDeMoverOCorpo` usa handle exclusivo (bloqueia a
leitura). **Nao migrei a primeira para `vaulttest.TravarExclusivo`**: com share
mode 0 o `os.ReadFile` da origem falha antes da copia, o destino nunca e criado e
o cenario da duplicacao deixa de existir. Provei as duas travas separadamente
(Prova 2a e 2b) em vez da unica que o brief descrevia.

**D2 — `lockFileForTest` e cross-platform e nao virou chamada direta a
`vaulttest`.** O chamador (`internal/index/build_descarte_test.go`) nao tem build
tag, e o lado `!windows` (`build_descarte_unix_test.go`) usa `os.Chmod(0000)`, que
nao tem equivalente em `vaulttest` — e nao deve ter, porque `TravarExclusivo` fora
do Windows faz `t.Skip` e o teste perderia cobertura em Linux e macOS. Solucao: o
lado Windows virou tres linhas delegando a `vaulttest.TravarExclusivo` (o
`CreateFile` a mao sumiu, que era o objetivo) e a assinatura dos dois lados perdeu
o `func()` de retorno em favor de `t.Cleanup`. **Isso obrigou a tocar dois
arquivos que o brief nao lista**: `build_descarte_unix_test.go` (mesma mudanca de
assinatura) e `build_descarte_test.go` (`unlock := …; defer unlock()` virou
`lockFileForTest(t, …)`). Ganho real: a copia Windows **nao conferia nada** — se o
share mode nao barrasse a leitura, `TestBuildRegistraArquivoIlegivel` exercitava um
cofre de dois arquivos legiveis e passava.

**D3 — Migrei tres arquivos alem dos listados**, porque as Verificacoes do brief
exigem zero `CreateFile(`/`FILE_ATTRIBUTE_OFFLINE` fora de `vaulttest` e eles
tinham blocos inline: `internal/index/build_cloudonly_windows_test.go`,
`internal/search/snippet_cache_windows_test.go` e
`internal/service/cloudonly_replace_windows_test.go`.

**D4 — `TravarDiretorioExclusivo` faz `t.Fatalf` onde o helper original de
`internal/vault` fazia `t.Skip`.** O original pulava com "handle exclusivo NAO
impediu ReadDir nesta maquina; o cenario nao se reproduz"; o do brief reprova.
Segui o brief — e a tese da tarefa (trava que nao trava e defeito, nao condicao de
ambiente) —, mas registro a troca: se alguma maquina de CI deixar o `ReadDir`
passar, `TestWalkNaoEngoleRaizQueExisteMasNaoLe` fica **vermelho** em vez de
pulado. Nesta maquina passa.

**D5 — `internal/ipc/ipc_test.go:68` afrouxou.** `TestDialAndHandshakeSocketAusente`
afirma `elapsed > boundedWait` como teto de "quanto tempo desistir de um socket
ausente"; o teto sobe de 3 s para 5 s por causa do `Prazo` unico. E uma assercao de
tempo real, nao so um limite de espera — foi a unica que encontrei nessa categoria.
Passa (o dial desiste em milissegundos), mas o teto ficou mais frouxo do que era.

**D6 — `internal/vault/walk_raiz_windows_test.go` virou `package vault_test`**, como
a decisao 3 do orquestrador mandava; nao havia identificador nao exportado. O
comentario de dominio que morava no helper (Lstat passa, ReadDir falha com
`ERROR_SHARING_VIOLATION`, e a relacao com `FalhaNaRaiz`) foi movido para cima do
teste em vez de perdido.

---

## Fora do escopo — visto e nao tocado

- `CLAUDE.md` diz que `verify.ps1` tem **14 etapas**; a saida numera **13**. Nao
  investiguei qual das duas esta errada.
- `docs/ESTRUTURA.md:229-230` tem um grafo obsoleto por outros motivos — a decisao 5
  do orquestrador manda deixar quieto, e deixei.
- `.superpowers/sdd/2026-07-25-gobsidian-v01/progress.md` aparece modificado no `git
  status` e **nao foi tocado por mim**; nao entrou no commit.
- Ha mais consumidores de `golang.org/x/sys/windows` em `_test.go` que nao sao
  desta familia: `internal/lifecycle/parent_identity_windows_test.go` (identidade de
  processo) e `internal/vault/cloudonly_info_windows_test.go` (interno, exceção
  documentada). Nenhum dos dois pertence a `vaulttest`.
- `internal/search/cloudonly_update_windows_test.go` guarda, em comentario, uma
  falha nao reproduzida de 2026-08-12 cuja causa nunca foi identificada. Continua
  la, intacta.

## Commit

`c81b2b8` — `test(vaulttest): one exclusive-handle helper that proves it locks, and one wait budget`.
Arquivos adicionados por caminho explicito; nenhum
`git add -A`, nenhum `git checkout/restore/stash/clean/reset` foi executado nesta
tarefa.
