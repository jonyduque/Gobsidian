# Review — Tasks 178 + 179

## Progresso

- 13:00 Lidos os briefs 178 e 179, o relatório combinado (`task-178-report.md`) e o diff `review-849dbb4..9803b01.diff` (3 commits: `364a3df`, `b6cf8be`, `9803b01`) na íntegra.
- 13:05 Lido `internal/service/graph.go` (`camposDeMetadata`, `incluidosPorPadrao`, o laço `includeSet`), `internal/service/errors.go` (`ValidarEnum`), `internal/mcpsrv/tools_read.go` (tag `jsonschema` de `Include`).
- 13:08 Lido `internal/service/write.go` inteiro no trecho de `PatchNote`/`AppendNote` para confirmar o achado B19 (`PatchNoteRequest.HeadingLevel` nunca lido; `AppendNoteRequest.HeadingLevel` sem teto).
- 13:09 Lido `scripts/check_tool_params.ps1` (cabeçalho) para confirmar a limitação de nível 2 citada no relatório e em B19.
- 13:11 `git log`, `git show <sha> --stat` e `git show 9803b01 -s --format=%B` nos três commits — confirmado HEAD = `9803b01`, mensagens batem com o relatório, rodapé `BREAKING CHANGE:` presente.
- 13:12 `go build ./...` limpo; `go vet ./internal/service/ ./internal/mcpsrv/` limpo.
- 13:13 `go test -race -count=1 ./internal/service/ ./internal/mcpsrv/` — `ok` nos dois pacotes.
- 13:14 `grep -rn '"hits"\|\.Hits\b' --include=*.go . docs/` — único resultado é a linha do próprio teste que confere a ausência da chave.
- 13:15 `grep -c 'minimum\|maximum' docs/TOOLS.md` — 1, e é a linha de prosa do parágrafo "Schemas servidos", não uma chave de bloco JSON.
- 13:16 Conferidos por grep todos os números citados no relatório contra o código: `LimitePadrao/LimiteTeto` (`errors.go:136-137`), `DefaultMaxResults/MaxResultsCeiling` (`config/defaults.go:9-10`), `DefaultSnippetChars/MaxSnippetChars` (`search/snippet.go:15-16`), `CandidatosPadrao/CandidatosTeto` (`outline.go:17-18`), `resources.go:66` (`Limit: 200` literal), clamp de `search.go:177-193`.
- 13:17 `pwsh -File scripts/verify.ps1 -SkipNet` completo, 14 passos, `[OK] Bateria completa. Pode commitar.`, exit 0.
- 13:18 Reproduzida a prova de mutação do Step 3 (`scripts/mutate.ps1` com a âncora e o teste do brief) de forma independente — FAIL sob mutação, exit 0, arquivo restaurado (`git status --short` limpo depois).
- 13:19 Conferido `docs/SUGESTOES.md` (P5 marcado feito, B19 completo) e `docs/ESTADO.md:160` (números "antes"/"depois" lado a lado) contra o texto atual do repositório (não contra o diff).
- 13:20 `python -c "open(...,encoding='utf-8').read()"` nos três `.md` editados — todos válidos.

## Spec compliance — Task 178

**APPROVED**

| Step | Requisito do brief | Verificado |
|---|---|---|
| 1 | Dois testes que falham hoje (`include=["headers"]`, `include=[""]`), um que passa (`include` válido) | RED colado no relatório bate com a saída que o teste produz nesta árvore; reproduzi o RED indiretamente via mutação (equivalente) |
| 2 | `camposDeMetadata` uma vez só, `includeSet` via `ValidarEnum`, tag `jsonschema` com os sete valores | `graph.go:518` e `tools_read.go:389` conferidos; sete valores na mesma ordem nos dois lugares |
| 3 | Prova de mutação | Reproduzida de forma independente — exit 0, FAIL sob mutação, arquivo restaurado |
| 4 | Commit 1 `fix:` só com os três arquivos do escopo | `364a3df` — `tools_read.go`, `graph.go`, `metadata_include_test.go`; nenhum `.md` |
| 5 | Tabela de auditoria completa, uma linha por propriedade | Quase completa — ver N2 (nit) |
| 6 | Reescrita do que era falso; nenhum `minimum`/`maximum` de bloco JSON restante; achado real vai para `SUGESTOES.md` | Confirmado por grep; B19 factualmente correto (verificado linha a linha) |

## Spec compliance — Task 179

**APPROVED**

| Step | Requisito do brief | Verificado |
|---|---|---|
| 1 | Teste do contrato falha enquanto `hits` existe, lendo JSON bruto (não struct) | `search_sem_hits_test.go` usa `map[string]json.RawMessage`, exatamente como o brief pede |
| 2 | Campo `Hits` apagado da struct e das três atribuições | `search.go` diff: `Hits` some da struct e das três `return SearchResult{...}` |
| 3 | Testes migrados para `Results` sem afrouxar | `match_offset_test.go`: renomeação 1:1, mesma asserção. `max_results_test.go`: a checagem OR (`!=5 && !=5`) vira checagem única sobre `Results`, comportamento equivalente (o teste roda depois da migração completa, então `Hits` não existe mais para a OR fazer sentido) — não é afrouxamento, é a mesma regra sem o campo morto. `filtro_data_test.go`: struct perde `Hits`, fallback `n==0 && len(Results)>0` vira `n := len(out.Results)` direto — mesma asserção final |
| 4 | Prova manual de mutação (reintroduzir `Hits`) com FAIL colado; grep final vazio | FAIL colado no relatório é consistente com o comportamento do teste (bati a lógica do teste linha a linha); `grep -rn '"hits"\|\.Hits\b'` rodado por mim aqui: único resultado é a assertiva do próprio teste |
| 5 | M3 medido depois, publicado em `ESTADO.md` ao lado do antes, com comando reproduzível | `ESTADO.md:160` traz antes e depois lado a lado, com o comando completo incluindo `--max-results 200` e a justificativa do porquê a flag é necessária |
| 6 | `TOOLS.md`, `SUGESTOES.md` atualizados; `verify.ps1` verde; commit com `BREAKING CHANGE:` | Todos conferidos; `feat(service)!:` com corpo que diz o que quebra (campo `hits` sumiu) e o que fazer (ler `results`) |

Regras de execução de ambas as tasks: nenhum `git checkout/restore/stash/clean/reset` nos commits (não há evidência de uso); nenhum outro campo de `SearchResponse` mudou (`Total`, `Truncated`, `SnippetCharsEfetivo`, `LimitEfetivo`, `TrechosIndisponiveis` intactos, conferido no diff de `search.go`); Conventional Commits em inglês nos três commits.

## Code quality

**APPROVED**, com dois achados não bloqueantes (N1, N2).

- `ValidarEnum("include", inc, "", camposDeMetadata...)`: para `inc == ""` devolve `padrao` (`""`) sem erro — por isso o `if v == ""` explícito logo depois é necessário e correto; testei a leitura contra `errors.go:165-174` e bate exatamente com o que o brief previa. `[""]` é rejeitado (via este `if`); `["frontmatter","frontmatter"]` seria aceito silenciosamente (mapa idempotente) — comportamento razoável, o brief não pede rejeição de duplicata.
- Mensagem de erro (`"include = %q invalido; aceitos: %s"`, dentro de `ValidarEnum`, e a mensagem custom para `""`) lista os valores aceitos nos dois casos.
- B19 é factualmente correto: `PatchNoteRequest.HeadingLevel` (`write.go:80`) não aparece em nenhum ponto do corpo de `PatchNote` (`write.go:265-266` em diante) — confirmei com leitura direta da função, sem grep enganoso. `check_tool_params.ps1` de fato documenta a limitação de nível 2 no próprio cabeçalho (linhas 19-27), e `AppendNoteRequest.HeadingLevel` é lido em `write.go:244`, mascarando o campo morto de `PatchNoteRequest` como o relatório descreve.
- `snippet_chars` clampa em 1000 (`search/snippet.go:15-16` + `search.go:189-193`) e `vault_search`'s `limit` clampa em 200 antes do teto administrativo (`search.go:177-181`) — ambos conferidos por leitura direta, batendo com o texto novo de `TOOLS.md`.
- Nenhum comentário obsoleto mencionando `hits` sobrou em código de produção.
- O corpo do commit `9803b01` diz o que quebra (`o campo hits, cópia byte a byte de results, some`) e o remédio (`leia results`) — suficiente, embora sucinto.

### N1 — should-fix (não bloqueante): `include` tem a mesma lista em dois lugares sem guarda de sincronia

`internal/service/graph.go:518` (`camposDeMetadata`, slice Go) e `internal/mcpsrv/tools_read.go:389` (tag `jsonschema`, string livre) citam os mesmos sete valores hoje, na mesma ordem — conferido manualmente. Mas nada testa que ficam iguais: a tag é texto livre, não gerado a partir de `camposDeMetadata`. Uma futura mudança em um dos dois (por exemplo, adicionar um oitavo campo a `camposDeMetadata` sem atualizar a description) não quebraria nenhum teste — só divergiria a documentação servida do comportamento real. O comentário em `graph.go:515-517` ("a validação e o schema precisam concordar e só um dos dois pode ser a fonte") registra a intenção mas não a impõe.

Fix concreto: um teste em `internal/mcpsrv` que extrai a `jsonschema` tag de `noteMetadataInput.Include` via reflection e compara (como conjunto) com `service.camposDeMetadata` exportado, ou — mais simples — gerar a string da tag a partir da lista em tempo de teste e comparar. Não bloqueante porque o brief não pediu essa guarda e os dois lugares estão de fato sincronizados agora.

### N2 — nit: tabela do Step 5 não é 100% exaustiva

O brief pede "uma linha por propriedade numérica ou enumerada de cada tool". Faltam duas linhas sem claim falso a corrigir, mas que são propriedades numéricas do schema: `note_read.max_bytes` (`TOOLS.md:100`, default 100000) e `note_list.offset` (`TOOLS.md:211`, default 0). Nenhuma das duas jamais teve `minimum`/`maximum` declarado, então a omissão não introduz erro de documentação — só deixa a tabela do relatório menos completa do que o brief pediu literalmente. Não bloqueante.

## Verified claims

```
$ go build ./...
(sem saída, sem erro)

$ go vet ./internal/service/ ./internal/mcpsrv/
(sem saída, sem erro)

$ go test -race -count=1 ./internal/service/ ./internal/mcpsrv/
ok  	github.com/jonyd/gobsidian/internal/service	35.395s
ok  	github.com/jonyd/gobsidian/internal/mcpsrv	8.667s

$ grep -rn '"hits"\|\.Hits\b' --include=*.go . docs/
./internal/mcpsrv/search_sem_hits_test.go:32:	if _, tem := out["hits"]; tem {

$ grep -c 'minimum\|maximum' docs/TOOLS.md
1
$ grep -n 'minimum\|maximum' docs/TOOLS.md
36:**Schemas servidos.** [...] Nenhum `enum`, `minimum`, `maximum` ou `default` chega ao host [...]

$ grep -n "MaxResults " internal/config/defaults.go ; grep -n "DefaultMaxResults\|MaxResultsCeiling" internal/config/defaults.go
9:	DefaultMaxResults = 50
21:		MaxResults:  DefaultMaxResults,
(MaxResultsCeiling = 500, mesma linha :10)

$ grep -n "DefaultSnippetChars\|MaxSnippetChars" internal/search/snippet.go
15:	DefaultSnippetChars = 240
16:	MaxSnippetChars     = 1000

$ grep -n "CandidatosPadrao\|CandidatosTeto" internal/service/outline.go
17:	CandidatosPadrao = 200
18:	CandidatosTeto   = 1000

$ sed -n '66p' internal/mcpsrv/resources.go
		Query: index.Query{Limit: 200, Sort: "modified", Order: "desc"},

$ pwsh -File scripts/verify.ps1 -SkipNet
[...]
[OK] Bateria completa. Pode commitar.
(exit code 0; 6 testes pulados, mesma lista do relatório: TestAjudanteSeguraTrava,
 TestListenRestringePermissaoUnix, TestSignalCancelsContext, TestPerfilDeHeapServindo,
 TestWriteAtomicPreservaOModoDoAlvo, TestNew_FailsOnUnwatchablePath)

$ pwsh -File scripts/mutate.ps1 -Path internal/service/graph.go \
    -Anchor '"backlinks", "inline_fields"}' \
    -Replacement '"backlinks", "inline_fields", "headers"}' \
    -Test TestNoteMetadataIncludeInvalidoEInvalidArgument -Package ./internal/service/
[...]
--- FAIL: TestNoteMetadataIncludeInvalidoEInvalidArgument (0.01s)
    metadata_include_test.go:27: include=["headers"]: quero INVALID_ARGUMENT, tenho <nil>
FAIL
[OK] internal/service/graph.go restaurado byte a byte (SHA-256 confere).
[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
Exit code: 0
$ git status --short internal/service/graph.go
(vazio — restaurado)

$ git log --oneline -4
9803b01 feat(service)!: vault_search returns results only
b6cf8be docs(tools): describe the schema the server actually serves; limits and enums are enforced server-side
364a3df fix(service): note_metadata rejects unknown include values
849dbb4 docs(sdd): record the Task 177 review, the fix round and the orphans gate log

$ git show 9803b01 -s --format=%B
feat(service)!: vault_search returns results only

BREAKING CHANGE: the hits field, a byte-for-byte copy of results that
doubled the payload, is gone. Read results.
[...]

$ python -c "open('docs/TOOLS.md',encoding='utf-8').read()" && echo OK
OK
$ python -c "open('docs/ESTADO.md',encoding='utf-8').read()" && echo OK
OK
$ python -c "open('docs/SUGESTOES.md',encoding='utf-8').read()" && echo OK
OK
```
