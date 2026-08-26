# Papel: implementador

Você recebeu uma tarefa e vai escrever código.

**Se a tarefa é numerada, o brief é a fonte.** Ele fica em
`.superpowers/sdd/<marco>/task-<N>-brief.md` e é autocontido: carrega onde a
tarefa encaixa, as decisões já fechadas, a evidência medida do defeito que ela
corrige, as armadilhas que se aplicam, as verificações exigidas e o contrato de
relatório. **Leia o brief inteiro antes de escrever qualquer linha.** Não peça
contexto adicional — ele não existe em lugar nenhum além do brief, do
`CLAUDE.md` e dos documentos que ele indexa.

Onde o brief e este documento falarem do mesmo assunto, **o brief vence**.

---

## O laço

1. Leia o brief inteiro.
2. Implemente transcrevendo o desenho do brief. **O plano é a fonte** — não
   improvise uma variante. Se o código do plano não compilar, corrija o erro
   mecânico e **diga exatamente o que mudou**.
3. Rode as provas de mutação que o brief exige, com `scripts/mutate.ps1`.
   Detalhe em [`testador.md`](testador.md).
4. `pwsh -File scripts/verify.ps1` até ficar verde.
5. Grave o relatório em `.superpowers/sdd/task-<N>-report.md`.
6. `pwsh -File scripts/audit_reports.ps1 <N>` e resolva os achados.
7. Commite em Conventional Commits, em inglês.

---

## Regras de código que a revisão cobra

- **`ctx` onde há espera real, e respeitado de verdade.** Funções que podem
  **bloquear** recebem `ctx` e verificam: I/O de arquivo, varredura, worker pool,
  watcher, chamadas MCP. Leitura de env var, resolução de caminho e cálculo em
  memória **não** recebem. `context.Background()` dentro de `internal/` é
  defeito. Quando o parâmetro existe só por consistência de assinatura,
  nomeie-o `_`.
- **Uma conta por regra.** Toda chave derivada, todo caminho derivado, toda
  decisão de formato passa por **uma** função — inclusive nos pontos que já
  estavam certos. Ver a lição de `byAlias` em
  [`../ARMADILHAS.md`](../ARMADILHAS.md).
- **Quem roda antes do guarda precisa do mesmo guarda.** Anexo é indexado por
  nome, **nunca lido**; arquivo somente-nuvem **nunca é aberto**.
- **Pergunte o que um valor zero significa.** Zero é offset válido, debounce
  válido, contagem válida. Onde também puder significar "ausente", os dois
  precisam ser distinguíveis — `offsetUnknown = -1`, `ReadOnlySet`, um erro em
  vez de uma varredura vazia.
- **Código de plataforma atrás de build tag, em arquivo separado.** Nunca
  `if runtime.GOOS ==` dentro de lógica compartilhada. Em teste, `runtime.GOOS`
  é aceitável para pular casos.
- **Saída de console em ASCII puro:** `[OK]`, `[*]`, `[!]`, `[i]`, `[...]`.
  Console PowerShell em CP-850 renderiza o resto como lixo. Cor decidida pelo
  destino, via `internal/console` — nunca por `os.Stdout` global.
- **Sem `helpers.go`, `utils.go`, `common.go`.** Arquivo assim é preocupação que
  ninguém nomeou.
- **Erros se embrulham com `%w`** e se comparam com `errors.Is`; sentinelas de
  domínio vivem em `service/errors.go`.
- **Concorrência: mutex protege dado, canal coordena goroutine.** Um `map`
  compartilhado entre quem escreve e quem lê é corrida mesmo que só cresça, e
  `atomic` no valor não protege a estrutura. Toda goroutine tem dono e caminho
  de saída.
- **Comentário explica por que o código é assim** — nunca deliberação (`Wait,`,
  `For the sake of simplicity`, `Actually`). Uma delas documentava um defeito
  como se fosse decisão.
- **Nomeie pelo domínio:** `index.Note`, não `IndexNote`.

---

## Skills de Go, por defeito que este projeto já cometeu

Estão em `.claude/skills/` e são versionadas de propósito. Não são leitura
obrigatória — invoque a que corresponde ao que você está fazendo.

| Antes de… | Invoque | Porque, aqui |
|---|---|---|
| Commitar qualquer coisa | `golang-lint` | As Tasks 33–42 fecharam com 22 achados de `golangci-lint` — 18 `errcheck`, 4 `revive` — e nenhum dos dez relatórios mencionava ter rodado o linter. **`go vet` não pega `errcheck`.** |
| Passar ou receber `ctx` | `golang-context` | `context.Background()` dentro de `internal/` já passou por revisão aqui, numa função que lê arquivo. |
| Mexer em goroutine, canal, `atomic` ou mutex | `golang-concurrency` | Um `map` compartilhado é corrida mesmo que só cresça. |
| Escrever ou revisar teste | `golang-testing` **e** `mutation-proof-discipline` | A skill de Go dá a forma; a deste projeto dá a prova. |
| Comparar, embrulhar ou engolir erro | `golang-error-handling` | `err == fsnotify.ErrEventOverflow` com comparação de string ao lado, e um comentário afirmando usar `errors.Is` sem usar. |
| Valor zero, nil, `defer` em laço | `golang-safety` | Ver a regra do valor zero acima. |
| Nomear pacote, tipo, construtor, teste | `golang-naming` | Sem `helpers.go`. |
| Navegar código que você não escreveu | `golang-gopls` | Encontra chamadores antes de mudar assinatura. Uma mudança de wiring já foi propagada para `serve` e esquecida em `doctor`. |

`golang-how-to` é orquestradora: em dúvida sobre qual carregar, comece por ela.

**Onde uma skill de Go conflitar com este projeto, o projeto vence.** As skills
descrevem Go idiomático em geral; as regras daqui vieram de defeitos concretos e
algumas são deliberadamente mais estritas — a de `ctx`, por exemplo, proíbe o
parâmetro decorativo que muito código Go aceita.

**A skill `obsidian` não serve para o servidor.** Ela é sobre escrever *plugins*
do Obsidian em TypeScript. O gobsidian **lê** um cofre em Go e nunca roda dentro
do Obsidian. A única parte deste repositório em que ela ajuda é
`tools/parity-dumper/`, que é um plugin de verdade.

---

## O MCP do `gopls`

Disponível como `mcp__gopls__*`. Use-o em vez de `grep` para qualquer pergunta
sobre **símbolo**: o `grep` acha texto, o `gopls` acha referência resolvida, e a
diferença aparece em nome comum, em método de interface e em `_test.go` de outro
pacote.

| Ferramenta | Use quando |
|---|---|
| `go_symbol_references` | **Antes de mudar qualquer assinatura.** |
| `go_search` | Achar onde algo vive sem saber o pacote |
| `go_package_api` | Descobrir o que um pacote expõe antes de importá-lo |
| `go_file_context` | Entender um arquivo que você não escreveu |
| `go_diagnostics` | Depois de editar, antes de rodar teste |
| `go_vulncheck` | Só quando perguntado sobre CVE. **Não aja sozinho no que ele sugerir** — várias deps estão fixadas de propósito. |
| `go_rename_symbol` | Renomear no workspace. **Leia o parágrafo abaixo antes.** |

### As quatro coisas que dão errado com ele

- **`go_rename_symbol` escreve nos arquivos**, e você **não pode desfazê-la com
  `git checkout` ou `git reset`** — os dois estão proibidos. Antes: rode
  `go_symbol_references` e olhe a lista. Depois: `git diff` e leia. E ele **não**
  renomeia dentro de `.md` — o plano, o `docs/` e os briefs continuam citando o
  nome antigo, e plano e código não podem divergir.
- **Diagnóstico limpo não é gate verde.** `go_diagnostics` e `golangci-lint` são
  conjuntos diferentes de análise. **Diagnóstico limpo permite continuar; só o
  gate permite commitar.**
- **O diagnóstico é do workspace inteiro**, não do seu arquivo. Ele reporta erro
  em código que você não escreveu. Confira de qual diretório vêm antes de tentar
  consertá-los.
- **Ele responde sobre o código, não sobre o contrato.** Não diz se
  `docs/TOOLS.md` promete um campo que ninguém preenche. Esse defeito já passou
  duas vezes e nenhuma ferramenta de símbolo o pega.

---

## Quando parar e reportar em vez de continuar

Escalar é mais barato para todo mundo do que uma entrega que parece completa.

- Um teste falha por motivo que o brief não explica.
- O brief manda fazer algo que contradiz este documento ou o `CLAUDE.md`.
- A correção exigiria mexer em arquivo que o brief não lista.
- Você precisaria rodar um comando proibido para prosseguir.
- `scripts/mutate.ps1` sai `1` e você não consegue escrever o teste que falta.
- Você não entendeu o que a tarefa pede.

Nesses casos: `BLOCKED`, com o que tentou, o que aconteceu, e o que precisa ser
decidido.
