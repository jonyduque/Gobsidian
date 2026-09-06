# Task 158 report

## Status
DONE

## Commit
3a1ab3e fix(service): a missing index is VAULT_UNAVAILABLE everywhere, and outline agrees with read

## Desvio do brief (por ruling do orquestrador, registrado no despacho)
O brief cita cinco sítios em `graph.go` (`:87,:303,:455,:556,:663`); o codigo
tem QUATRO: `LinkGraph` (85/87 apos delecao da Task 154 anterior no arquivo —
a numeracao mudou por causa de outros commits do lote, nao desta tarefa),
`TagList` (301/303), `ListNotes` (453/455), `NoteMetadata` (554/556).
`VaultStats` (`~662`) tem `if s.index == nil` com fallback para `vault.Walk` e
NAO retorna erro — deixado intocado, e a linha "Stats" foi retirada da tabela
do teste, conforme a instrucao do despacho. O metodo real e `ListNotes`
(nao `NoteList`), tipo de request `ListRequest` — usado corretamente no
teste.

## Evidencia de TDD

RED — quatro falhas, codigo INTERNAL nas quatro chamadas que devem virar
VAULT_UNAVAILABLE (Stats fora da tabela, como acordado):
```
$ go test ./internal/service/ -run TestServicoSemIndiceDevolveVaultUnavailable -v
=== RUN   TestServicoSemIndiceDevolveVaultUnavailable
    errors_test.go:53: LinkGraph sem indice: codigo = INTERNAL, quer VAULT_UNAVAILABLE (index not available)
    errors_test.go:53: TagList sem indice: codigo = INTERNAL, quer VAULT_UNAVAILABLE (index not available)
    errors_test.go:53: ListNotes sem indice: codigo = INTERNAL, quer VAULT_UNAVAILABLE (index not available)
    errors_test.go:53: Metadata sem indice: codigo = INTERNAL, quer VAULT_UNAVAILABLE (index not available)
--- FAIL: TestServicoSemIndiceDevolveVaultUnavailable (0.00s)
FAIL
FAIL	github.com/jonyd/gobsidian/internal/service	1.190s
```

GREEN — apos os quatro sitios em graph.go e o de outline.go:71:
```
$ go test ./internal/service/ -run TestServicoSemIndiceDevolveVaultUnavailable -v
=== RUN   TestServicoSemIndiceDevolveVaultUnavailable
--- PASS: TestServicoSemIndiceDevolveVaultUnavailable (0.00s)
PASS
ok  	github.com/jonyd/gobsidian/internal/service	0.794s
```

## Prova de mutacao (manual, ancora ambigua nos quatro sitios identicos)

Mutacao manual em UM sitio (`LinkGraph`, `graph.go:87`), diff:
```diff
 func (s *Service) LinkGraph(_ context.Context, req GraphRequest) (GraphResult, error) {
 	if s.index == nil {
-		return GraphResult{}, Errorf(CodeVaultUnavailable, "indice indisponivel")
+		return GraphResult{}, fmt.Errorf("indice indisponivel")
 	}
```

Teste reprova, nomeando o metodo e a linha da asserção:
```
$ go test ./internal/service/ -run TestServicoSemIndiceDevolveVaultUnavailable -v
=== RUN   TestServicoSemIndiceDevolveVaultUnavailable
    errors_test.go:53: LinkGraph sem indice: codigo = INTERNAL, quer VAULT_UNAVAILABLE (indice indisponivel)
--- FAIL: TestServicoSemIndiceDevolveVaultUnavailable (0.00s)
FAIL
FAIL	github.com/jonyd/gobsidian/internal/service	0.803s
```

Restaurado. `git diff --stat internal/service/graph.go` apos o restauro
mostrou só as quatro linhas do fix real (4 insertions/4 deletions),
confirmando que a mutacao temporaria nao deixou residuo — colado na secao
"Evidencia de TDD" acima (a saida GREEN final foi rodada depois do restauro).

## Verificacoes do brief
1. `grep -rn 'fmt.Errorf("index not available")' internal/service` -> vazio
   (o primeiro grep pegou uma citacao literal no COMENTARIO do proprio teste
   novo; reescrevi o comentario para nao conter a string literal, e o grep
   ficou limpo — exit 1).
2. `grep -n "CodeInternal" internal/service/outline.go` -> vazio (exit 1); a
   linha do `ReadAll` nao aparece mais.
3. `docs/TOOLS.md`: a linha de `VAULT_UNAVAILABLE` na tabela de codigos de
   erro agora cobre indice indisponivel e leitura de nota falha; a secao
   "Erros" de `note_outline` ganhou `VAULT_UNAVAILABLE` e nao promete
   `INTERNAL` para nota ilegivel.

`go build ./...`, `go vet ./...` limpos; `golangci-lint run ./internal/service/...`
-> `0 issues.`; `go test ./internal/service/... -race -count=1` ->
`ok  	github.com/jonyd/gobsidian/internal/service	58.546s`.

## verify.ps1
Rodado uma vez antes do primeiro commit do lote (154) — ver task-154-report.md
para a saida completa (13 etapas [OK], exit 0). Gate completo replanejado
para depois do ultimo commit do lote (156); entre commits, build+vet+test
por pacote bastou.

## O que ficou de fora
`write.go` tem varios `Errorf(CodeInternal, ...)` para falhas de I/O em
create/append/patch/move/delete — fora do escopo desta tarefa (o brief só
pede os quatro sitios de graph.go mais outline.go:71); nao tocados.

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
Apenas `internal/service/graph.go`, `internal/service/outline.go`,
`internal/service/errors_test.go` e `docs/TOOLS.md` entraram no commit
`3a1ab3e`.

## Progresso
- 01:15 iniciado Task 158: errors_test.go, graph.go, outline.go, service.go, vault.New conferidos
- 01:17 158: teste TestServicoSemIndiceDevolveVaultUnavailable escrito (sem a linha Stats, por ruling); RED confirmado (4 falhas INTERNAL)
- 01:20 158: quatro sitios de graph.go + outline.go:71 corrigidos; GREEN
- 01:22 158: mutacao manual em LinkGraph, teste reprova nomeado; restaurado, diff --stat confere so o fix real
- 01:24 158: grep index-not-available pegou comentario do proprio teste; comentario reescrito, grep limpo
- 01:26 158: docs/TOOLS.md atualizado (linha VAULT_UNAVAILABLE + Erros de note_outline)
- 01:30 158: go test ./internal/service/... -race verde (58.5s); commit 3a1ab3e sem bloqueio do doc-hook
