# Task 157 report

## Status
DONE

## Commit
b1f2cfe fix(search): the cache format version is one constant; magic and codec derive from it [sem-doc]

## Evidencia de TDD

RED (teste escrito, ainda com literais independentes, mas ja passa porque os
tres valem 6 hoje — o brief pede a prova via bump manual, nao um RED
tradicional):
```
$ go test ./internal/search/ -run TestVersaoDoCacheDeBuscaEUmaConta -v
=== RUN   TestVersaoDoCacheDeBuscaEUmaConta
--- PASS: TestVersaoDoCacheDeBuscaEUmaConta (0.00s)
PASS
ok  	github.com/jonyd/gobsidian/internal/search	0.692s
```

GREEN (apos o alias, mesmo teste):
```
$ go test ./internal/search/ -run TestVersaoDoCacheDeBuscaEUmaConta -v
=== RUN   TestVersaoDoCacheDeBuscaEUmaConta
--- PASS: TestVersaoDoCacheDeBuscaEUmaConta (0.00s)
PASS
ok  	github.com/jonyd/gobsidian/internal/search	0.714s
```

## Prova de mutacao (bump manual, conforme o brief)

ANTES do alias, bump `CacheFormatVersion` 6 -> 7 em `persist.go` (diff):
```diff
-	CacheFormatVersion   = 6
+	CacheFormatVersion   = 7
```
Saida — o teste reprova nas duas contas (Fatalf para na primeira asserção):
```
$ go test ./internal/search/ -run TestVersaoDoCacheDeBuscaEUmaConta -v
=== RUN   TestVersaoDoCacheDeBuscaEUmaConta
    persist_codec_test.go:281: cacheCodecVers = 6, CacheFormatVersion = 7: duas contas
--- FAIL: TestVersaoDoCacheDeBuscaEUmaConta (0.00s)
FAIL
FAIL	github.com/jonyd/gobsidian/internal/search	0.797s
FAIL
```
Restaurado (`CacheFormatVersion = 6`); `git diff --stat internal/search/persist.go`
vazio, confirmado antes de prosseguir.

DEPOIS do alias (`cacheCodecVers = CacheFormatVersion`,
`cacheMagic = fmt.Sprintf("GBS%d", CacheFormatVersion)`), o mesmo bump 6 -> 7:
tudo deriva, o teste PASSA, e a bateria `Cache|Persist` inteira tambem passa
(20 testes, cache escrito e lido pela versao 7 na mesma rodada):
```
$ go test ./internal/search -run 'Cache|Persist' -v
... (20 testes, todos PASS, ver lista completa abaixo em "Verificacoes")
PASS
ok  	github.com/jonyd/gobsidian/internal/search	0.963s
```
Restaurado (`CacheFormatVersion = 6`); `git diff --stat internal/search/persist.go`
vazio, confirmado de novo antes do commit.

## Verificacoes do brief
1. As duas saidas de bump (ANTES reprova, DEPOIS passa) coladas acima.
2. `grep -rn "GBS6\|cacheCodecVers = 6" internal/search` -> vazio (exit 1, sem
   correspondencia).
3. `go test ./internal/search -run 'Cache|Persist' -v` verde, 20 testes,
   incluindo `TestSaveAndLoadInvertedCache`, `TestSaveOverwritesMappedCache`,
   `TestCacheAnalyzerVersionMismatchDiscardsCache`, `TestTruncatedCacheRefused`,
   `TestEmptyVaultCacheDistinguishableFromMissing`, `TestCacheOutsideVault` —
   cache gravado e lido pela mesma versao continua carregando.

`go vet ./internal/search/...` limpo; `golangci-lint run ./internal/search/...`
-> `0 issues.`; `go test ./internal/search/...` completo (nao so o filtro)
verde em 5,625s.

## verify.ps1
Rodado uma vez, ANTES do primeiro commit do lote (154), cobrindo tambem esta
mudanca por estar no working tree naquele momento — ver task-154-report.md
para a saida completa (13 etapas [OK], exit 0). Rerun completo do gate
planejado para depois do ultimo commit do lote (156).

## O que ficou de fora
Nada. `internal/bincodec` fica adiado por decisao do dono (c2), conforme o
brief; este alias fecha o defeito imediato sem criar o pacote novo.

## git status --porcelain (apos o commit desta tarefa)
```
 M .superpowers/sdd/2026-07-25-gobsidian-v01/progress.md
 M "test-vault/test vault/.obsidian/community-plugins.json"
 D "test-vault/test vault/.obsidian/plugins/parity-dumper/main.js"
 D "test-vault/test vault/.obsidian/plugins/parity-dumper/manifest.json"
 M "test-vault/test vault/.obsidian/workspace.json"
?? .claude/skills/troglodita-commit/ ... (arquivos preexistentes do dono, inalterados)
?? Resume-Claude.ps1
?? "test-vault/test vault/..." (preexistentes)
```
Apenas `internal/search/persist_codec.go` e
`internal/search/persist_codec_test.go` entraram no commit `b1f2cfe`.

## Progresso
- 00:56 iniciado Task 157: TestVersaoDoCacheDeBuscaEUmaConta escrito, passa (fmt ja importado)
- 00:58 157: bump manual 6->7 ANTES do alias, teste reprova (colado); restaurado
- 01:00 157: alias implementado (cacheCodecVers deriva, cacheMagic vira var com fmt.Sprintf)
- 01:01 157: go build/vet OK; bump 6->7 DEPOIS do alias, bateria Cache|Persist (20 testes) passa; restaurado
- 01:03 157: grep GBS6/cacheCodecVers=6 vazio; lint 0 issues; go test ./internal/search/... verde
- 01:05 157: commit bloqueado por doc-hook (mudanca interna, sem contrato externo); commitado com [sem-doc] em b1f2cfe
- 01:49 fix round 1 (nao-bloqueante, da revisao do lote): docs/wiki/entities/formato-do-cache.md ainda dizia que so o cache de metadados usa alias; acrescentado paragrafo citando cacheCodecVers/cacheMagic de persist_codec.go, source_commit -> b1f2cfe, updated_at -> 2026-09-04, UTF-8 validado, check_doc_refs e check_readme_anchors verdes, commit 574eb21 docs(wiki): the search cache derives magic and codec version from CacheFormatVersion too
