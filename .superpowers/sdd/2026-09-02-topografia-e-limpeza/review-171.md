# Revisao da Task 171 — `4d7f397`

Base `8a70044`. Revisor somente-leitura: nada foi editado, commitado ou revertido.
`verify.ps1` e `test_orphans.ps1` NAO foram rodados aqui (o implementador ja os
rodou); o que rodei esta em "Verified myself".

## Spec compliance

**Veredito: OK**

| Requisito do brief | Estado |
|---|---|
| `atomic.go`, `syncdir_unix.go`, `syncdir_windows.go` movidos para `vault` | OK — `syncdir_*` como rename (97% / 98%); `atomic.go` explicado em N4 |
| `atomic_test.go`, `durabilidade_test.go`, `sweep_profundo_windows_test.go` movidos | OK — renames de 84% / 88% / 92% |
| `package writer` → `package vault`, `writer_test` → `vault_test` | OK nos seis |
| `const TempFilePrefix = ".gobsidian-tmp-"` em `vault` | OK — `internal/vault/atomic.go:14` |
| `ReplaceFile(ctx, target, func(*os.File) error)` extraida | OK — `internal/vault/atomic.go:127` |
| Docstring de `ReplaceFile` mantem os cinco passos numerados | OK — `atomic.go:104-108`, passo 2 reescrito para o callback |
| Comportamento preservado (temp no mesmo dir, chmod do alvo, Sync, Close, rename 10x/10ms, fsync do dir, cleanup no defer) | OK — corpo identico ao antigo linha a linha; a unica troca e `tmpFile.Write(data)` → `escrever(tmpFile)` e o texto do erro ("escrevendo dados no temporario" → "escrevendo no temporario"), sem nenhum dependente (grep) |
| `WriteAtomic` como wrapper de `ReplaceFile` | OK — `atomic.go:194-199` |
| `SweepResult` e `SweepStaleTempFiles` movidos, corpo intacto | OK — diff mostra bloco identico |
| `writer/atomic.go` so com os quatro encaminhadores | OK — 50 linhas, `const`/`type =`/duas funcoes |
| Marcador `// Deprecated:` | Substituido pela prosa, **por ruling do orquestrador** — nao contabilizado como defeito |
| `walk.go` usa `TempFilePrefix` | OK — `internal/vault/walk.go:73` |
| Comentario de `longpath_windows.go:26` corrigido | OK |
| `TestWalkIgnoraTemporarioDeBinarioAntigo` existe e pode falhar | OK — `walk_test.go:118-139`; prova de mutacao com a saida colada |
| `TestReplaceFileCallbackFalhaNaoTocaOAlvo` existe e pode falhar | OK — `atomic_test.go:228-264`; duas provas de mutacao coladas |
| Provas no passado, com `EXIT=0` colado | OK — tres provas, todas com a saida do `mutate.ps1` |
| Chamadores (`service/write.go`, `cmd/gobsidian/servico.go`) intocados | OK — nenhum dos dois esta no `git show --name-only` |
| `vault` continua folha | OK — medido, sem `internal/*` |
| `writer` sem aresta nova | OK — `parser`, `text`, `vault`, os mesmos de antes |
| Codigo de plataforma atras de build tag em arquivo separado | OK — `syncdir_unix.go` `//go:build !windows`, `syncdir_windows.go` `//go:build windows`; nenhum `runtime.GOOS` novo em logica compartilhada |
| `CLAUDE.md`, `ESTRUTURA.md`, `ARCHITECTURE.md` atualizados | OK — os tres, e todos UTF-8 valido |
| Commit com os 13 caminhos explicitos, nada do dono | OK — `--name-only` da exatamente os 13; `progress.md`, `test-vault/`, `.claude/skills/` e `Resume-Claude.ps1` continuam fora do commit, no working tree |
| Conventional Commits, em ingles | OK — `refactor(vault): atomic replace, write and temp sweep move to vault; writer forwards` |
| Timestamps do relatorio plausiveis | OK — commit em `2026-09-06 07:25:29 -0300`; a linha "07:25 — commit `4d7f397`" bate |

Fora de escopo, sem contaminacao: nenhum `net/*` novo (`vault` e `writer` so
trazem stdlib + `golang.org/x/sys/windows`, que ja estava la), nenhum tipo do SDK
MCP fora de `mcpsrv`, nenhum `fmt.Print*` novo em codigo alcancavel de `serve` (o
unico `fmt.Print` e o do processo auxiliar em `atomic_test.go`, que ja existia e
tem o comentario que explica por que nao fere a regra).

## Code quality

**Veredito: OK**

O `git mv` preservou os comentarios historicos inteiros — os achados P11, M12 e
M13, o paragrafo do glob por diretorio de 2026-07-30, a sondagem de MAX_PATH de
2026-08-27. Nada de "movi e resumi". A docstring de `ReplaceFile` ganhou o
paragrafo que explica **por que** o callback existe (os dois caches em
streaming, com o numero medido `24,95 MiB`), e nao apenas que existe.

Os dois testes novos sao testes de verdade: o de `walk` usa o **literal** na
fixture, com o comentario dizendo por que; e o desvio deliberado do brief
(`.gobsidian-tmp-abc123` → `.gobsidian-tmp-abc123.md`) esta certo e o relatorio
explica o mecanismo — sem a extensao `.md`, o filtro de extensao descartaria o
arquivo com ou sem `isNoise`, e a prova de mutacao teria saido 1. Isso e
exatamente a checagem que o brief nao fez e o implementador fez.

O relatorio e honesto no que interessa: declara o bloqueio do gate em vez de
esconde-lo, declara que a tarefa **nao** foi test-first em vez de inventar um
RED, e declara uma divergencia pre-existente da `ARCHITECTURE.md` que nao
corrigiu.

## Findings

**Bloqueantes: 0.**

**N1 — should-fix — `docs/ARCHITECTURE.md:421` (e `:405-414`)**
§5.5 diz "Backoff exponencial, tres tentativas, 50 ms iniciais". O codigo faz 10
tentativas com 10 ms fixos (`internal/vault/atomic.go:176-177`), e ja fazia em
`8a70044` — conferi: a linha e byte a byte identica na base, entao **nao e
regressao desta PR**, e o relatorio a declara. Anoto porque a §5.5 tambem nao
menciona o `fsync` do diretorio nem o `Chmod` para o modo do alvo, que sao as
duas metades do achado M12 e estao no codigo movido — a secao que descreve a
escrita atomica ficou tres fatos atras do codigo justo na PR que a mudou de
pacote. Correcao: tarefa propria que reescreva §5.5 com os quatro numeros
medidos e os dois passos faltantes. Nao dobrar dentro da PR2 em silencio.

**N2 — should-fix — `internal/index/persist.go:114`, `internal/search/persist.go:69`**
Os dois caches montam o nome do temporario com o literal
(`".gobsidian-tmp-index-cache-*.gob"` e `".gobsidian-tmp-cache-*.gob"`), nao com
`vault.TempFilePrefix`. Sao pre-existentes — conferi em `8a70044`, identicos — e
sao **precisamente** os dois arquivos que a Task 172 migra para `ReplaceFile`,
que e o que faria o literal sumir. O risco e que a Task 172 troque so as chamadas
`writer.WriteAtomic` de `service/write.go` e `servico.go` e deixe estas duas de
fora: o brief da 172 nao as nomeia, o relatorio da 171 nao as menciona, e a regra
"uma conta por regra" continuaria com duas copias do prefixo no repositorio, num
lugar onde a varredura do boot as apaga se o cache morar dentro do cofre.
Correcao: o brief da Task 172 tem de citar `index/persist.go:114` e
`search/persist.go:69` por caminho e linha, e a verificacao da 172 deve ser
`grep -rn '\.gobsidian-tmp-' --include=*.go internal cmd` devolvendo **so**
`internal/vault/atomic.go:14` e as fixtures de `walk_test.go`.

**N3 — nit — `docs/ESTRUTURA.md:47` e `:117`**
A arvore lista `vault/ignore.go` e `writer/writer.go`; nenhum dos dois existe
(`ls internal/vault/*.go`, `ls internal/writer/*.go`). Pre-existente e fora das
linhas que esta PR alterou, mas esta nos dois blocos que ela editou, no documento
que o `CLAUDE.md` chama de "arvore autoritativa". Correcao: apagar as duas linhas
numa tarefa de documentacao.

**N4 — nit — relatorio, secao "Grep de chamadores"**
O padrao do grep (`writer\.WriteAtomic\|writer\.SweepStaleTempFiles`) devolve 7
linhas, e a prosa do relatorio esta correta para esse padrao (6 chamadas +
`servico.go:103`). A superficie real de encaminhador em uso e de **8** linhas:
falta `cmd/gobsidian/servico.go:73`, `res writer.SweepResult`, que a Task 172
tambem precisa migrar. Sem impacto nesta PR; a 172 nao pode se guiar pelo numero
7.

**N5 — nit — `internal/vault/atomic.go:104-125`**
A docstring de `ReplaceFile` nao diz o contrato do callback: que ele **nao** deve
fechar nem renomear o `*os.File`, e que qualquer buffer intermediario precisa ter
sido drenado quando o callback retorna. Fechar por dentro faz o `Sync()` da linha
172 devolver `file already closed` e a escrita falha alto, sem tocar o alvo — a
falha e segura, mas a mensagem nao aponta para a causa. Correcao: uma frase na
docstring ("o callback nao fecha o arquivo; quem escreve por buffer da Flush
antes de retornar").

## Verified myself

```
$ git show --name-only --format="" 4d7f397        # 13 caminhos, nenhum chamador
CLAUDE.md  docs/ARCHITECTURE.md  docs/ESTRUTURA.md
internal/vault/{atomic.go,atomic_test.go,durabilidade_test.go,longpath_windows.go,
  sweep_profundo_windows_test.go,syncdir_unix.go,syncdir_windows.go,walk.go,walk_test.go}
internal/writer/atomic.go
$ git show --format="%ci" -s 4d7f397
2026-09-06 07:25:29 -0300

$ go list -f '{{.ImportPath}}: {{.Imports}}' ./internal/vault/ ./internal/writer/
.../internal/vault: [bytes context errors fmt golang.org/x/sys/windows io io/fs
  log/slog os path path/filepath strings sync sync/atomic syscall time]
.../internal/writer: [bytes context errors fmt .../internal/parser .../internal/text
  .../internal/vault sort strings sync]
                                            -> vault continua folha; writer sem aresta nova

$ go build ./...   -> BUILD OK
$ go vet ./...     -> VET OK
$ GOOS=linux go build ./...   -> LINUX BUILD OK
$ GOOS=darwin go build ./...  -> DARWIN BUILD OK
$ GOOS=linux go vet ./internal/vault/ ./internal/writer/  -> LINUX VET OK
$ gofmt -l internal/vault internal/writer  -> (vazio)

$ go test -race -count=1 ./internal/vault/ ./internal/writer/
ok  github.com/jonyd/gobsidian/internal/vault   18.472s
ok  github.com/jonyd/gobsidian/internal/writer   3.398s
                                            (coerente com os 45,628s de -count=3 do relatorio)

$ grep -rn '\.gobsidian-tmp-' --include=*.go internal cmd tools
internal/index/persist.go:114   <- N2, pre-existente (identico em 8a70044)
internal/search/persist.go:69   <- N2, pre-existente (identico em 8a70044)
internal/vault/atomic.go:14     <- a constante
internal/vault/walk_test.go:57  <- fixture literal, deliberada
internal/vault/walk_test.go:121 <- fixture literal, deliberada
                                   nenhum literal sobrou em codigo de producao de vault/writer

$ grep -rn "writer\.WriteAtomic\|writer\.SweepStaleTempFiles\|writer\.SweepResult" \
    --include=*.go internal cmd | grep -v _test
service/write.go:161,260,389,640,846 | servico.go:73 (tipo), :78 (chamada), :103 (comentario)
                                            6 chamadas intocadas; ver N4

$ grep -rn "Deprecated:" internal/writer/ internal/vault/
internal/writer/atomic.go:17   <- so dentro da prosa que explica a ausencia; nao e
                                  marcador (nao inicia paragrafo, comentario solto)

$ python -c "open('X',encoding='utf-8').read()"
[OK] CLAUDE.md  [OK] docs/ARCHITECTURE.md  [OK] docs/ESTRUTURA.md  [OK] task-171-report.md

$ git status --porcelain   -> progress.md, test-vault/, .claude/skills/, Resume-Claude.ps1
                              e os arquivos soltos do ledger continuam FORA do commit
```

**Provas de mutacao — nao reexecutadas** (o revisor e somente-leitura e
`mutate.ps1` edita o arquivo em disco). Verifiquei a evidencia de outro jeito: as
tres saidas coladas citam `walk_test.go:138`, `atomic_test.go:250` e
`atomic_test.go:263`, e as tres linhas nos arquivos commitados sao exatamente os
`t.Fatalf` correspondentes (`sed -n '135,140p'` e `sed -n '246,265p'`). Linha
inventada nao acerta tres de tres.

**Nao rodei** `scripts/verify.ps1` nem `scripts/test_orphans.ps1`, por instrucao.
A ultima linha `[OK] Bateria completa. Pode commitar.` esta no relatorio e nao foi
reconferida aqui; o que rodei — build nos tres GOOS, `vet`, `gofmt`, `-race` nos
dois pacotes — e consistente com ela.
