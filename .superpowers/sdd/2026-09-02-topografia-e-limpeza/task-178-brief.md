### Task 178: `note_metadata` recusa `include` desconhecido; `TOOLS.md` diz o que o schema serve

**Files:**
- Modify: `internal/service/graph.go:567-577` (`includeSet` via `ValidarEnum`), `:519-521` (comentário de `MetadataRequest.Include`)
- Modify: `internal/mcpsrv/tools_read.go:378-381` (tag `jsonschema` de `Include` lista os sete valores)
- Create: `internal/service/metadata_include_test.go`
- Modify: `docs/TOOLS.md:13,:62,:228-238,:481` e cada `minimum`/`maximum` (`:54,:55,:94,:96` e os demais que `grep -n 'minimum\|maximum' docs/TOOLS.md` listar)

**Interfaces:**
- Consumes: `ValidarEnum(campo, valor, padrao string, aceitos ...string) (string, error)` (`errors.go:166`; `""` devolve `padrao`); `Errorf(CodeInvalidArgument, …)`; `newTestService(t, root)` (`read_test.go:22`); a tabela de defaults da Task 168 (para o `:13`).
- Produces: `var camposDeMetadata = []string{"frontmatter", "tags", "headings", "blocks", "links", "backlinks", "inline_fields"}` em `graph.go`.

- [ ] **Step 1: Testes — falham porque hoje qualquer string é aceita**

`internal/service/metadata_include_test.go`:

```go
package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func cofreComUmaNota(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "n.md"), []byte("---\nk: v\n---\n# N\n\n#tag texto\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestNoteMetadataIncludeInvalidoEInvalidArgument(t *testing.T) {
	svc := newTestService(t, cofreComUmaNota(t))
	_, err := svc.NoteMetadata(context.Background(), MetadataRequest{
		Path: "n.md", Include: []string{"headers"},
	})
	var se *Error
	if !errors.As(err, &se) || se.Code != CodeInvalidArgument {
		t.Fatalf("include=[\"headers\"]: quero INVALID_ARGUMENT, tenho %v", err)
	}
}

func TestNoteMetadataIncludeVazioEInvalidArgument(t *testing.T) {
	svc := newTestService(t, cofreComUmaNota(t))
	_, err := svc.NoteMetadata(context.Background(), MetadataRequest{
		Path: "n.md", Include: []string{""},
	})
	var se *Error
	if !errors.As(err, &se) || se.Code != CodeInvalidArgument {
		t.Fatalf("include=[\"\"]: quero INVALID_ARGUMENT, tenho %v", err)
	}
}

func TestNoteMetadataIncludeValidoContinuaAceito(t *testing.T) {
	svc := newTestService(t, cofreComUmaNota(t))
	res, err := svc.NoteMetadata(context.Background(), MetadataRequest{
		Path: "n.md", Include: []string{"tags", "inline_fields"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Tags) != 1 {
		t.Fatalf("tags = %v, quero [tag]", res.Tags)
	}
}
```

`newTestService` está em `read_test.go`, `package service` (caixa-branca) — por isso este arquivo também é `package service`. Run: `go test ./internal/service/ -run TestNoteMetadataInclude -v` — Expected: os dois primeiros FAIL (`err == nil`), o terceiro PASS.

- [ ] **Step 2: A lista, uma vez, e o laço com `ValidarEnum`**

`internal/service/graph.go`, acima de `NoteMetadata`:

```go
// camposDeMetadata sao os valores aceitos em MetadataRequest.Include. A lista
// mora aqui, no service, porque a validacao e o schema (mcpsrv) precisam
// concordar e so um dos dois pode ser a fonte.
var camposDeMetadata = []string{"frontmatter", "tags", "headings", "blocks", "links", "backlinks", "inline_fields"}

// incluidosPorPadrao e o que note_metadata devolve quando include e omitido:
// blocks e inline_fields ficam de fora porque sao os dois campos que crescem
// com o tamanho da nota.
var incluidosPorPadrao = []string{"frontmatter", "tags", "headings", "links", "backlinks"}
```

Substituir `:567-577`:

```go
	includeSet := make(map[string]bool, len(camposDeMetadata))
	if len(req.Include) == 0 {
		for _, c := range incluidosPorPadrao {
			includeSet[c] = true
		}
	} else {
		for _, inc := range req.Include {
			v, err := ValidarEnum("include", inc, "", camposDeMetadata...)
			if err != nil {
				return MetadataResult{}, err
			}
			if v == "" {
				return MetadataResult{}, Errorf(CodeInvalidArgument,
					"include = \"\" invalido; aceitos: %s", strings.Join(camposDeMetadata, ", "))
			}
			includeSet[v] = true
		}
	}
```

`ValidarEnum` devolve o `padrao` (aqui `""`) para valor vazio sem erro — por isso o segundo teste e o `if v == ""`. Run: `go test ./internal/service/ -run 'TestNoteMetadataInclude|TestService_' -v` — Expected: PASS em todos.

`internal/mcpsrv/tools_read.go:380`: `Include []string \`json:"include,omitempty" jsonschema:"campos a devolver; aceitos: frontmatter, tags, headings, blocks, links, backlinks, inline_fields; omitido devolve frontmatter, tags, headings, links, backlinks"\``. Run: `go test ./internal/mcpsrv/ -run TestNoteMetadata_IncludeParameter` — Expected: PASS.

- [ ] **Step 3: Prova de mutação**

```powershell
pwsh -File scripts/mutate.ps1 -Path internal/service/graph.go -Anchor '"backlinks", "inline_fields"}' -Replacement '"backlinks", "inline_fields", "headers"}' -Test TestNoteMetadataIncludeInvalidoEInvalidArgument -Package ./internal/service/
```

Expected: exit 0 (com `"headers"` na lista o teste falha). Colar.

- [ ] **Step 4: Commit 1**

```bash
git add internal/service/graph.go internal/service/metadata_include_test.go internal/mcpsrv/tools_read.go
git commit -m "fix(service): note_metadata rejects unknown include values"
```

- [ ] **Step 5: Auditoria do que `TOOLS.md` promete contra o que o schema serve**

Contexto que o implementador não tem como saber sozinho: `jsonschema-go v0.4.2` (`jsonschema.For[T]`) lê a tag `jsonschema:"…"` **só como `description`**. Nenhum `enum`, `minimum`, `maximum` ou `default` é emitido; a única tool que remenda o schema à mão é `note_read` (`schemaDoNoteRead`, `alvo_note_read.go:71`). Logo toda linha de `TOOLS.md` que mostra `"minimum": 1` ou `"enum": [...]` descreve um schema que o host **não** recebe.

Montar a tabela abaixo (no relatório e, resumida, no topo da seção de contratos de `TOOLS.md`) — uma linha por propriedade numérica ou enumerada de cada tool:

| Tool | Propriedade | `TOOLS.md` diz | Código faz (clamp / rejeita / ignora / onde) | Schema servido |
|---|---|---|---|---|
| vault_search | limit | default 50, min 1, max 500 | clamp a `max_results`, `effective_limit` (`search.go:…`) | só description |
| … | … | … | … | … |

Fonte de "Código faz": `grep -n "Clamp\|clamp\|LimitePadrao\|ValidarEnum" internal/service/*.go` e a tabela de defaults da Task 168. Fonte de "TOOLS.md diz": `grep -n 'default\|minimum\|maximum\|enum' docs/TOOLS.md`.

- [ ] **Step 6: Reescrever o que estava falso**

- `TOOLS.md:13`: "O padrão de `limit` é 20 em `vault_search` e 100 nas demais listas (`LimitePadrao`); o teto é 500" — conferir os dois números na tabela da Task 168 antes de escrever; se divergirem, a tabela vence.
- `TOOLS.md:62`: "e o padrão é 50" → o valor real de `limit` de `vault_search` (20) e do teto `max_results` (o valor de `config.Defaults()`; escrever o número que `grep -n MaxResults internal/config/defaults.go` mostrar).
- `TOOLS.md:481`: "limite configurável (padrão: 200…)" → "limite fixo de 200 (`resources.go:66`); não há flag nem parâmetro que o mude".
- `TOOLS.md:228-238` (schema de `include`): trocar o bloco JSON com `enum`/`default` por prosa: "`include`: lista de strings; aceitos `frontmatter`, `tags`, `headings`, `blocks`, `links`, `backlinks`, `inline_fields`; valor fora da lista devolve `INVALID_ARGUMENT`; omitido devolve `frontmatter`, `tags`, `headings`, `links`, `backlinks`."
- Cada `"minimum"`/`"maximum"` em bloco JSON de schema: remover a chave e dizer em prosa o que o código faz — "valores acima do teto são clampados e `effective_limit` informa o valor usado" onde há clamp; "valores fora da faixa devolvem `INVALID_ARGUMENT`" onde há `ValidarEnum`/rejeição; e onde o código **ignora**, escrever isso e abrir uma linha em `docs/SUGESTOES.md` (não corrigir código nesta Task: é `docs:`).
- Parágrafo novo no topo da seção de contratos: "Os schemas servidos trazem só `type` e `description` (limitação de `jsonschema-go v0.4.2`; a exceção é `note_read`). Limites e enumerações são aplicados pelo servidor, não pelo host: um valor inválido volta como `INVALID_ARGUMENT`, um valor acima do teto volta clampado com `effective_*`."

Run: `pwsh -File scripts/verify.ps1` (inclui `check_tool_params` e `check_doc_refs`) — Expected: verde. Se `check_tool_params` reclamar de uma propriedade que a tabela mostra como "ignora", é achado real — vai para o relatório e para `SUGESTOES.md`, não se apaga a linha do doc.

- [ ] **Step 7: Commit 2**

```bash
git add docs/TOOLS.md docs/SUGESTOES.md
git commit -m "docs(tools): describe the schema the server actually serves; limits and enums are enforced server-side"
```

#### Verificações

- Step 1 FAIL→PASS colado (os dois FAIL e o PASS de controle).
- `mutate.ps1` exit 0 colado.
- Tabela do Step 5 completa no relatório, uma linha por propriedade.
- `grep -c 'minimum\|maximum' docs/TOOLS.md` antes e depois; depois deve ser `0` ou cada ocorrência restante está em prosa, não em bloco JSON — listar.
- `verify.ps1` verde depois de cada commit.

#### Regras de execução

- Nunca `git checkout/restore/stash/clean/reset`.
- Commit 1 é `fix:` (comportamento); commit 2 é `docs:` — nenhum `.go` no segundo.
- Os limites **não** entram no schema servido. Ruling do orquestrador registrado no ledger: a validação do SDK transformaria clamp em erro duro e contradiria `effective_limit`. Se o implementador discordar, escreve no relatório; não remenda `schemaDo*`.
- Não inventar número: cada default e teto escrito em `TOOLS.md` vem de um `grep` colado no relatório.

#### Comando de mutação

Step 3 (`mutate.ps1`, âncora `"backlinks", "inline_fields"}` em `internal/service/graph.go`).

#### Contrato de relatório

`task-178-report.md`: status, dois SHAs, saídas dos Steps 1 e 3, a tabela do Step 5, os `grep` que fundamentam cada número escrito, última linha do `verify.ps1`.

---

