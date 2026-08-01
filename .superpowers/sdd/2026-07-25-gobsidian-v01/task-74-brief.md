### Task 74: `netcheck` como `go vet -vettool` no CI

**Onde encaixa.** H4 do M6, segunda metade da garantia de RNF-30.

**A regra que isto protege, e ela não é negociável:** *nenhum pacote sob `internal/` ou `cmd/` importa `net` ou `net/*`.* `net/http` e `x/oauth2` chegam transitivamente pelo SDK MCP — isso é esperado e permitido. O check inspeciona **os nossos pacotes**, não o fecho transitivo.

**O que já existe.** `scripts/check_net.ps1`, rodando como etapa do `verify.ps1`. Ele é textual. Esta tarefa acrescenta a análise semântica, que pega o que o texto não pega — import com apelido, por exemplo.

#### Passos

1. Um analisador `golang.org/x/tools/go/analysis` que reprova import de `net` ou `net/*` em pacote sob `internal/` ou `cmd/`.
2. Empacotado como `vettool`, rodável por `go vet -vettool=<binário> ./internal/... ./cmd/...`.
3. No CI, e também no `verify.ps1`.

**Atenção às dependências.** Se `golang.org/x/tools` ainda não tem importador, `go.sum` pode faltar entradas transitivas. O comando certo é `go get <caminho-do-pacote>@<versão>` — caminho do **pacote**, não do módulo; `go get <módulo>@<versão>` é no-op quando o módulo já está requerido naquela versão. **Nunca rode `go mod tidy`**: várias deps estão fixadas sem importador de propósito, e `tidy` removeria elas junto com o pin do SDK MCP, que é decisão fechada (PRD D6).

#### Verificações além dos passos

- **Prove que reprova.** Acrescente `import "net/http"` a um pacote sob `internal/`, rode, confirme a reprovação, remova. Cole a saída.
- **Prove que reprova import com apelido** — `import foo "net"`. É o caso que o check textual não pega, e é a razão de esta tarefa existir.
- **Prove que NÃO reprova o import transitivo do SDK.** Se ele reprovar isso, o projeto inteiro para de compilar no CI e alguém vai desligar o check.
- Roda nos três alvos: `GOOS=windows`, `linux`, `darwin`. Sem `lint-windows`, todo arquivo `//go:build windows` fica sem análise — já aconteceu aqui.

#### Prova de mutação obrigatória

Os três primeiros itens acima, cada um com a saída colada.

`scripts/mutate.ps1` **não serve aqui**: ele roda teste Go com `-Test` e `-Package`, e o alvo desta prova não é teste Go. A prova é a remoção descrita acima, com a saída colada — mesma disciplina, ferramenta diferente.

#### Regras de execução

Idênticas às da Task 69.

#### Contrato de relatório

Grave em `.superpowers/sdd/task-74-report.md`: o código do analisador; as três provas com saída colada; o diff do CI e do `verify.ps1`; a confirmação de que `go.mod` e `go.sum` mudaram só pelo que foi acrescentado (`git diff go.mod go.sum` colado).

Responda com no máximo 15 linhas.

**Files:** Create `internal/netcheck/` or `tools/netcheck/`; modify `.github/workflows/`, `scripts/verify.ps1`
**Commit:** `ci: semantic net-import check as a go vet analyzer`

---

