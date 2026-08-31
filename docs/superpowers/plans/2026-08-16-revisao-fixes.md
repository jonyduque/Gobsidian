# Correções da revisão de 2026-08-15 (Tasks 104–123)

> **ENTREGUE. Este plano está fechado em 2026-08-31.** As Tasks 104, 105, 106 e
> 116 constam no ledger sob esta numeração; 110, 113, 118, 119, 120, 121, 122 e
> 123 foram entregues sob a numeração da auditoria de 2026-08-25; e 107, 109,
> 112 e 114 foram entregues em 2026-08-31, depois de uma auditoria achá-las **em
> estado nenhum** — nem feitas, nem abertas, nem BLOCKED.
>
> Os briefs abaixo ficam como registro do desenho e das decisões D-R-1 a D-R-8.
> **Não são fila.** Estado real: `.superpowers/sdd/2026-07-25-gobsidian-v01/progress.md`.
>
> Uma correção de rumo, para quem for reusar o desenho: o plano B da Task 109
> (*"se `jsonschema-go` não suportar `oneOf`, descreva as duas formas na
> descrição do campo"*) **teria quebrado todo cliente atual em silêncio** — o SDK
> valida a entrada contra o `InputSchema` antes de chamar `UnmarshalJSON`. Ver
> `docs/ARMADILHAS.md` § Contratos de API.

Base: `b2be492`. Fonte: `docs/REVISAO-2026-08-15.md`, 48 itens. Registro dos
achados, fechados e rejeitados: `docs/wiki/notes/achados-abertos.md`.

Alvo deste lote: as duas superfícies com falha **observada** (o incidente de
campo do Claude Desktop e a travessia de caminho na escrita), mais os gates que
deveriam ter pego uma delas e não pegaram, mais os achados baratos de alto
retorno. Desempenho, `os.Root`, daemon/IPC, Go 1.26 e recursos novos ficam
nomeados no fim como escopo posterior, sem brief.

---

## Fase 0 — o que o instrumento não viu, e por que

Escrito antes das tarefas, de propósito, como no M7. A revisão achou 48 itens
por leitura; a pergunta que importa para o processo não é quantos, é **quantos
gates existiam e passaram verdes por cima deles**.

| Defeito | Gate que deveria pegar | O que aconteceu |
|---|---|---|
| `tag_list.sort` e `tag_list.hierarchical` declarados e mortos (item 7) | `scripts/check_tool_params.ps1`, em `verify.ps1:189` | **Verde.** `[OK] todo parametro declarado e lido em algum lugar` sobre 69 parâmetros |
| Boot do índice de busca abre placeholder de nuvem (item 2) | `TestUpdateNaoAbreNotaSomenteNuvem` + helper `construirComoOBoot` | **Verde.** O helper chama `inv.Update`; a produção não chama |
| `daemon-idle` nunca roda no CI (item 20) | `.github/workflows/ci.yml` | Chama três cenários por nome; o quarto não está lá |
| `check_doc_refs`, `check_readme_anchors`, `check_tool_params` | CI | **Nenhum dos três está no CI.** Só existem dentro de `verify.ps1`, que nenhum job invoca |
| Travessia `..\` na escrita (item 1) | nenhum | Não há teste que peça escrita com separador do Windows |
| O produto perder a tarefa central para outro servidor MCP | nenhum | Nenhum gate pergunta se uma tool resolve a tarefa que ela existe para resolver |

Duas medições, para não ficar no argumento:

```
$ pwsh -File scripts/check_tool_params.ps1
[i] 12 structs de entrada, 69 parametros declarados.
[OK] todo parametro declarado e lido em algum lugar.
EXIT=0
```

```
$ grep -n "test_orphans" .github/workflows/ci.yml
122:  run: ./scripts/test_orphans.ps1 -Cycles 100 -Scenario stdin-eof
125:  run: ./scripts/test_orphans.ps1 -Cycles 100 -Scenario parent-death
133:  run: ./scripts/test_orphans.ps1 -Cycles 100 -Scenario signal
```

A causa do primeiro é mecânica e vale registrar aqui porque a Task 104 nasce
dela: `check_tool_params.ps1` casa **nome de campo no pacote inteiro**, não por
struct. `Sort` do `tagListInput` passa porque existe um `Sort` lido em
`tools_read.go:147`, que é de outra tool. `Hierarchical` passa porque é
repassado a `service.TagRequest` — e `service.TagList` o ignora. O próprio
cabeçalho do script admite o segundo limite ("NAO prova que o campo e honrado"),
mas o primeiro é um falso negativo puro, e é o que deixou três campos passarem.

**Por isso a Fase 0 vem antes de tudo.** Corrigir o item 7 com o gate quebrado
garante que o quarto campo morto escape do mesmo jeito.

---

## Decisões fechadas para a batelada inteira

Repetidas dentro de cada tarefa que as vincula, porque o brief é a unidade que
viaja. **Não re-litigar.**

**D-R-1. Estrutura em negrito: alternativa D agora, E como decisão separada.**
`note_outline` (Task 112) devolve o mapa da nota — headings ATX reais e,
rotulados à parte como candidatos, parágrafo que é só negrito, setext e
numeração hierárquica, com offsets. O parser **não** muda. A alternativa E
(pseudo-heading como extensão goldmark, com `Heading.Synthetic`) fica registrada
como decisão de produto pendente, fora deste lote. Quem executar não "aproveita
para" reconhecer negrito no parser.

**D-R-2. Item 1 se corrige por `vault.Resolve`; `os.Root` é fase própria.**
A escrita passa a ter um portão único (`Resolve`), e `vault.CanonicalPath`
deixa de ser construída por conversão fora do pacote `vault`. `os.Root` /
`os.OpenRoot` fecha o item 4 (symlink, TOCTOU) e é refatoração de `vault`,
`writer` e `index` — fase posterior, tarefa própria. Misturar as duas deixa a
travessia aberta durante uma refatoração longa.

**D-R-3. `offset` é sempre relativo ao byte 0 do arquivo.** Uma segunda origem
de coordenada (relativa à seção recortada) cria uma segunda linguagem de
endereçamento ao lado de heading e bloco, e as duas divergem no dia em que uma
ganhar tratamento que a outra não tem — é o argumento que a própria revisão usa
para rejeitar a alternativa F. Consequências obrigatórias, e são contrato:
`offset` é **mutuamente exclusivo** com `heading` e com `block_id`
(`INVALID_ARGUMENT` se vierem juntos), e com `offset` presente
`include_frontmatter` é **ignorado** — coordenada absoluta não admite um começo
móvel. Isso vai no `jsonschema` do campo, não só no `TOOLS.md`.

**D-R-4. Toda resposta que pode ser cortada declara como continuar.**
`total_size` (bytes do arquivo), `next_offset` (byte seguinte ao último
devolvido, ou ausente quando acabou) e `truncated` **correto** — verdadeiro
apenas quando o clamp disparou, nunca por uma faixa que mede exatamente
`max_bytes`. Teto sem par de continuação não é controle, é limite, e foi assim
que o incidente terminou.

**D-R-5. Campo de schema é implementado ou removido, na mesma tarefa.**
Não existe "fica para depois" para um campo que o modelo do outro lado lê para
decidir. Isso vale para `tag_list.sort`, `tag_list.hierarchical`, `max_results`
(que não é campo de schema — ver a Task 104)
e para todo campo que estas tarefas acrescentarem.

**D-R-6. `IndexCacheParserVersion` não sobe neste lote.** Nenhuma tarefa de 104
a 123 pode mudar o que o `parser` grava no cache de metadados. `note_outline`
calcula os candidatos sintéticos **no momento da chamada**, sobre os bytes da
nota, e não os persiste. Quem sentir necessidade de persistir parou na tarefa
errada: isso é a alternativa E, e é decisão de produto pendente (D-R-1).

**D-R-7. Prova de mutação sai do `scripts/mutate.ps1`, com a saída colada.**
Código de saída invertido de propósito: `0` = o teste reprovou sob mutação
(regra verificada), `1` = passou (regra escrita, não verificada), `2` =
inconclusivo. `EXIT` sozinho não diz **por quê** — três vezes neste projeto ele
enganou (Tasks 94, 101, 103). Cole a saída e diga qual asserção falhou. Prova
escrita no condicional ("se removermos X, o teste falharia") não é prova.

**D-R-8. Não escreva número que você não mediu.** Nenhum item deste lote é de
desempenho, então nenhuma tarefa daqui tem direito a citar ganho. Se uma
mudança parecer ter custo, meça com `benchstat -count=6` ou escreva **"não
medido"**.

---

## Armadilhas já pagas que valem para a batelada inteira

Cada uma aconteceu neste repositório. Repetidas nas tarefas que as vinculam.

- **Um teste que não pode falhar é pior que teste ausente.** Antes de dizer que
  testou: apague a regra, rode, confirme que um teste **nomeia** a falha,
  restaure. Três casos reais aqui, e a Fase 0 acrescentou o quarto.
- **Teste de fallback que deixa o caminho principal ligado mede o caminho
  principal.** Reincidiu duas vezes. Vale diretamente para a Task 115: o teste
  do placeholder de nuvem tem de exercitar **o laço de boot**, não um helper que
  imita o laço de boot.
- **Chave derivada calculada em dois lugares diverge**, e a divergência aparece
  no caminho menos usado. Vale para a Task 118 (três normalizações) e para a
  119 (`lowerPath` escrito num lugar e lido noutro).
- **Quem roda antes do guarda precisa do mesmo guarda.** `CorrelateRenames`
  abria o que `index.Replace` recusava. Vale para a Task 115.
- **Reparar metade do estado é pior que não reparar.** A reconciliação por
  overflow consertava um índice e deixava o outro obsoleto. Vale para a Task 113
  (se `Resolve` virar portão em `CreateNote` e não em `MoveNote`, o buraco
  continua aberto pelo caminho menos usado).
- **Conserto que remove uma parada abrupta abre caminho que ninguém executou.**
  Depois de remover um `panic`, um `Fatal` ou um `return` de erro, a pergunta é
  o que passa a rodar agora — e a resposta se acha percorrendo os chamadores a
  jusante. Vale para a Task 121 (`parser.Parse` deixando de mentir na assinatura
  torna alcançável o ramo de erro de `build.go:78`).
- **Campo de API com valor fixo mente sempre.** Vale para `total_size` e
  `next_offset`: devolver zero literal é pior que não devolver.
- **Script Python que edita arquivo versionado precisa de `newline=""` na
  leitura E na escrita**, e `str.replace` que não casa segue em silêncio. Use
  `Edit`, não script, para inserir código com escapes.
- **Pipe engole código de saída.** `cmd | tail` devolve o status do `tail`.
  Redirecione para arquivo e leia o `$?` do comando.
- **Não deixe sua deliberação no código.** `// better condition` está commitado
  em `read.go:244` e sai na Task 106.

## Regras de execução, válidas para as vinte tarefas

Rodar `pwsh -File scripts/verify.ps1` antes de dizer que acabou, e colar a
contagem de passos. Registrar no ledger
(`.superpowers/sdd/2026-07-25-gobsidian-v01/progress.md`) **antes** de reportar
conclusão; conferir todo SHA citado com `git cat-file -t`. Escopo não encolhe em
silêncio: se alguma parte não deu, entregue o resto inteiro e diga o que ficou
de fora e por quê — `BLOCKED` com motivo é resposta melhor que uma entrega que
parece completa. O relatório é o entregável, não o resumo dele: comando rodado,
saída real colada, prova de mutação no passado.

Antes de começar cada tarefa: `pwsh -File scripts/sdd.ps1 base <N>`. Depois:
`pwsh -File scripts/sdd.ps1 review <N>`.

---

## Ambiente de teste — o que já existe, e em que pacote

**Leia isto antes de escrever qualquer teste deste lote.** Nada aqui precisa ser
inventado; tudo já está no repositório. Inventar um segundo helper para o que já
existe é a divergência do `byAlias` aplicada a teste.

### `internal/service` tem DOIS pacotes de teste, e escolher errado não compila

| pacote | arquivo de referência | o que ele alcança |
|---|---|---|
| `package service` (interno) | `read_test.go` | tipos sem qualificação: `ReadRequest`, `CodeOf`, `CodeHashMismatch`, campos não exportados |
| `package service_test` (externo) | `search_test.go` | só a API exportada, via `service.` — mas tem o helper de busca |

**Regra prática:** teste que precisa de **busca, trecho ou índice invertido** vai
em `package service_test` e usa `createSearchService`. Todo o resto vai em
`package service` e usa `newTestService`.

### Helpers de `package service` (em `internal/service/read_test.go`)

```go
// Grava root/name com 0644. NAO cria subdiretório — para subpasta, use
// os.MkdirAll antes.
func writeFile(t *testing.T, root, name, content string)

// vault.New(root) + index.New() + Build. O índice invertido vai NIL, e as
// Options vão zeradas. Chamar de novo sobre a mesma raiz REINDEXA — é assim
// que se reindexa de forma determinística, nunca com time.Sleep.
func newTestService(t *testing.T, root string) *Service
```

`CodeOf(err) Code` (`errors.go:88`) extrai o código de um erro do serviço, ou
`INTERNAL` se ele não carregar um.

### Helper de `package service_test` (em `internal/service/search_test.go`)

```go
func createSearchService(t *testing.T, files map[string]string) (
    *service.Service, *vault.Vault, *index.Index, *search.Inverted)
```

Cria a raiz, grava cada arquivo (**criando subdiretórios**), constrói o índice
de metadados, tokeniza o cofre, **grava e recarrega o cache de busca**, e falha
se `inv.VindoDoCache()` for falso. Esse último passo não é cerimônia: `Postings`
tem dois ramos e o servidor sempre executa o que vem do cache — até 2026-08-13 os
testes de latência mediam o outro.

### Helpers de `internal/mcpsrv` (`package mcpsrv_test`)

```go
// defaults_test.go — servidor + sessão MCP em memória, com Cleanup registrado.
func novaSessao(t *testing.T, root string) (*mcp.ClientSession, context.Context)

// tools_write_test.go — servidor cru, com config à escolha.
func newTestServerWithConfig(t *testing.T, root string, cfg config.Config) *mcpsrv.Server

// e2e_move_test.go — chama a tool e desserializa StructuredContent no destino.
// FALHA o teste se res.IsError; para testar erro, use session.CallTool direto.
func chamaTool(ctx context.Context, t *testing.T, session *mcp.ClientSession,
    nome string, args map[string]any, destino any)
```

**Armadilha medida:** `newTestServerWithConfig` monta `search.NewInverted()`
**vazio**. `vault_search` por essa sessão devolve zero resultados sempre, e um
teste que contar hits ali estará medindo o vazio. Quando a asserção depender da
contagem, monte o servidor com o índice populado — a Task 120 traz o helper
literal.

### Helpers de `internal/index` (`package index_test`, em `classify_test.go`)

```go
func construirPorBuild(t *testing.T, root string) *index.Index    // caminho do boot
func construirPorReplace(t *testing.T, root string) *index.Index  // caminho do watcher
```

O terceiro caminho de produção — gravar e recarregar o cache de metadados — não
tem helper; a Task 118 escreve o dela com
`index.SaveIndexCache(ctx, cacheDir, vaultPath, ix)` e
`index.LoadIndexCache(ctx, cacheDir, vaultPath) (*Index, *CacheHeader, error)`.

### Convenções de teste que valem para o lote inteiro

- **Nunca `time.Sleep` para esperar indexação.** Reconstrua com
  `newTestService(t, root)`. Sleep mede a carga da máquina, e este projeto já
  pagou por isso.
- **Todo teste tem um controle.** Se a asserção é "X não acontece", exista um
  caso em que X **acontece** — senão uma implementação que recusa tudo passa.
- **`t.Skip` só depois de conferir que a condição não se montou**, e o relatório
  tem de dizer que o teste rodou de verdade. Skip incondicional é teste apagado.
- **Mensagem de falha carrega o valor observado.** `t.Fatalf("hits = %d, quer 3", n)`,
  nunca `t.Fatal("errado")`.
- **Não asserte texto de mensagem inteiro.** Asserte as substrings que o cliente
  precisa ver. Mensagem é reescrita; contrato é o que ela informa.

---

# Task 104 — o gate de parâmetro de tool que não pode reprovar

**Tier: modelo principal.** O entregável é projetar a checagem, não transcrevê-la.

#### Onde encaixa
Primeira tarefa do lote. A Task 120 corrige os campos mortos; esta corrige
o instrumento que os deixou passar. Nesta ordem, e não na inversa.

#### O que vincula esta tarefa
- **D-R-5**: campo de schema é implementado ou removido, na mesma tarefa.
- **D-R-7**: prova de mutação com a saída colada.
- **Um teste que não pode falhar é pior que teste ausente**, porque reporta
  cobertura que não existe. Este script *é* um teste, e é o caso literal.

#### A evidência medida do defeito
```
$ pwsh -File scripts/check_tool_params.ps1
[i] 12 structs de entrada, 69 parametros declarados.
[OK] todo parametro declarado e lido em algum lugar.
EXIT=0
```

Verde, com **dois** campos mortos no schema: `tag_list.sort` (declarado em
`tools_read.go:328`, o handler de `tag_list` em `:243-255` nunca o lê) e
`tag_list.hierarchical` (chega em `service.TagRequest`; `service/graph.go:180`
`TagList` nunca o consulta).

**Correção de uma versão anterior deste brief, que pedia três.** `max_results`
também é uma cadeia morta (`config.MaxResults` → `service.Options.MaxResults` →
nenhum leitor; `config.MaxResultsCeiling = 500` não é referenciado em lugar
nenhum), **mas ele nunca foi campo de schema de tool**: é flag de CLI, não vive
em nenhuma struct `*Input` de `internal/mcpsrv`, e portanto está fora do alcance
deste instrumento por construção. Quem executar não deve forçar o script a
enxergá-lo — isso o transformaria num grep de nomes, que é o defeito que ele
existe para corrigir. `max_results` é responsabilidade da Task 120.

#### A decisão que esta tarefa tem de acertar
São **dois** falsos negativos distintos, e o script só admite o segundo:

1. **Casamento por nome no pacote inteiro.** `Sort` do `tagListInput` passa
   porque `noteListInput` também tem um `Sort`, e esse é lido. A checagem tem de
   ser **por struct**: para cada campo de cada struct `*Input`, procurar
   `in.<Campo>` (ou o nome do receptor que aquele handler usa) **dentro do corpo
   do handler daquela tool**, não em qualquer lugar de `internal/mcpsrv`.
2. **Leitura que só repassa.** `Hierarchical` é lido pelo handler e entregue ao
   `service`, que o ignora. Este é o limite que o cabeçalho do script já
   declara, e a correção honesta não é adivinhar: é **seguir um nível**. Quando
   o campo é atribuído a um campo de struct de `internal/service`, exigir que
   esse campo apareça pelo menos uma vez em `internal/service` além da própria
   declaração e além da atribuição vinda de `mcpsrv`.

Se o segundo nível se mostrar frágil na implementação (encadeamento maior que
um salto), **pare e entregue só o primeiro**, com o segundo escrito como limite
conhecido no cabeçalho do script — e diga isso no relatório. Um gate que
reprova por engano ensina a ignorá-lo.

#### O que a primeira tentativa errou — não repita

A primeira entrega desta tarefa (2026-08-16) construiu o escopo por handler
corretamente e depois **o anulou**. Três buracos, todos verificados por execução:

**1. O disjunto que devolve o grep de nomes.** A decisão de nível 1 era:

```powershell
$level1Read = ($handlerBody -match "\b$pVar\.$NomeGo\b") -or ($handlerBody -match "\b$NomeGo\b")
```

O segundo disjunto casa o **nome nu em qualquer lugar** do corpo do handler, e
`-match` do PowerShell é **insensível a maiúsculas por padrão**. Medido: apagando
o único leitor real de `noteListInput.Sort` (`if in.Sort != ""`, `tools_read.go:147`),
o gate continuou reportando **2** campos mortos em vez de 3 — porque o corpo do
handler ainda contém `sort := "path"` e `Sort:` no literal de `index.Query`. O
campo passou a ser declarado e não lido, e o gate não viu.

`tag_list.sort` só é pego porque o handler de `tag_list` é curto e não contém a
palavra `sort` em lugar nenhum. **A regra entregue não é "o campo é lido"; é "o
nome não aparece neste handler"** — mais fraca, e ela falha exatamente no caso
que a tarefa existe para pegar.

**A decisão de nível 1 tem de ser só `$pVar.$NomeGo`**, sem disjunto de nome nu,
e com `-cmatch` para ser sensível a maiúsculas. Se algum campo legítimo passar a
reprovar por causa disso, ele é um achado de verdade — investigue antes de
afrouxar.

**2. O fallback que reinstala o comportamento antigo em silêncio.** Quando a
struct não é achada em `$StructHandlers`, o corpo vira `internal/mcpsrv` inteiro
concatenado. É a regra por pacote que esta tarefa remove, de volta como caminho
padrão, sem uma linha de aviso. Qualquer mudança no jeito de registrar handler
(um wrapper novo, uma renomeação) devolve o gate ao estado quebrado e nada diz.
**Struct sem handler resolvido é achado**, não fallback: reporte
`HANDLER-NAO-RESOLVIDO` e saia `1`.

**3. O nível 2 não tem escopo nenhum.** Ele varre `internal/` linha a linha
procurando o **nome nu** do campo. `Path`, `Limit`, `Content`, `Sort` aparecem
milhares de vezes; o nível 2 só dispara para um nome que não existe em lugar
nenhum de `internal/`, que foi por acidente o caso de `Hierarchical`. Ele precisa
seguir o campo até a struct de destino (`service.TagRequest.Hierarchical`) e
procurar `req.<Campo>` / `<recv>.<Campo>` **nos métodos que recebem aquela
struct** — ou ser declarado limite conhecido e não fingir cobertura.

Incoerência que denuncia as três: o nível 1 usa `-match` e o nível 2 usa
`-cmatch`. O que decide é o mais fraco.

Terceiro ponto, e não é opcional: o script hoje imprime só o veredito. Passa a
imprimir **a lista dos 69 pares struct/campo com onde cada um foi lido**, como
`check_doc_refs` faz com as dispensas. Lista de exceção que ninguém vê deixa de
ser revisada; lista de cobertura que ninguém vê é a mesma coisa.

#### Passos
1. `git log --oneline -3 -- scripts/check_tool_params.ps1` para saber o que a
   versão atual já tentou.
2. Reescrever o casamento para operar por struct. O parser de Go do script é
   textual; se a delimitação do corpo do handler por chaves se mostrar
   inconfiável, use `go/ast` num pequeno programa sob `tools/` em vez de
   PowerShell — mas então ele precisa de teste próprio, e isso é escopo maior:
   registre e decida.
3. Acrescentar o segundo nível (mcpsrv → service).
4. Imprimir a tabela de cobertura.
5. Rodar contra o HEAD atual e **exigir que ele reprove com os dois campos de
   schema** — `tagListInput.Sort` e `tagListInput.Hierarchical`. Três é o estado
   **sob mutação**, e `max_results` está fora do alcance do instrumento.

#### O que prova esta tarefa

**Duas coisas, e a segunda é a que a primeira tentativa não fez.**

**(a)** O script tem de sair `1` **hoje**, antes de qualquer correção da Task
120, nomeando os **dois** campos. Cole a saída literal.

**(b) O gate tem de pegar um campo morto que ele nunca viu.** Sair `1` sobre os
dois conhecidos não prova nada sobre a regra — prova que aqueles dois nomes não
aparecem naqueles dois handlers. O procedimento, e o resultado dele vai colado
no relatório:

1. Guarde `sha256sum internal/mcpsrv/tools_read.go`.
2. Apague o leitor de `noteListInput.Sort` em `internal/mcpsrv/tools_read.go:146-149`:
   ```go
   sort := "path"
   if in.Sort != "" {
       sort = in.Sort
   }
   ```
   vira
   ```go
   sort := "path"
   ```
3. Rode `pwsh -File scripts/check_tool_params.ps1`. Ele **tem** de reportar
   **3** campos, incluindo `noteListInput.Sort`. Se reportar 2, a regra continua
   sendo grep de nomes e a tarefa não está pronta.
4. Restaure o arquivo e **confira pelo SHA-256** que ele voltou byte a byte.
5. Repita com um campo de outra família — sugestão: `moveInput.UpdateLinks`,
   apagando a leitura em `tools_write.go`. Um só caso pode passar por acidente.

Cole os dois SHAs (antes e depois do restauro) e as duas saídas do gate.

Depois da Task 120, ele volta a sair `0` — e aí a prova é a inversa: mutação que
apaga a linha que **honra** um dos campos tem de fazer o script reprovar de novo.

#### Prova de mutação
```powershell
pwsh -File scripts/mutate.ps1 `
  -Path internal/service/graph.go `
  -Anchor 'tags := s.index.Tags(req.Prefix, req.MinCount)' `
  -Replacement 'tags := s.index.Tags(req.Prefix, req.MinCount) // req.Hierarchical descartado' `
  -Test TestTagListHierarquico -Package ./internal/service/
```
(Aplicável depois da Task 120; nesta tarefa a prova é a saída `1` do script
contra o HEAD.)

#### Contrato de relatório
Saída do script antes (EXIT=0, verde) e depois (EXIT=1, três campos nomeados),
literais. A tabela de cobertura impressa, ao menos as linhas dos dois campos.
Se o segundo nível ficou de fora, diga isso na primeira linha do relatório.

---

# Task 105 — o CI roda o que só existia no `verify.ps1`

**Tier: modelo barato.** Edição de YAML, sem projeto.

#### Onde encaixa
Segunda da Fase 0. Fecha a lacuna de processo que a revisão registra no item 20
e a que ela não registra.

#### O que vincula esta tarefa
- **Gate cujo padrão cobre parte do que ele aparenta cobrir é pior que gate
  ausente.** O CLAUDE.md conta a história de quando o padrão de
  `test_orphans.ps1` era `stdin-eof` e três mecanismos apareciam como
  verificados. A mesma forma se repetiu com o quarto cenário e com três
  scripts.
- **Não afirme estado que você não verificou.**

#### A evidência do defeito
```
$ grep -n "test_orphans" .github/workflows/ci.yml
122:  run: ./scripts/test_orphans.ps1 -Cycles 100 -Scenario stdin-eof
125:  run: ./scripts/test_orphans.ps1 -Cycles 100 -Scenario parent-death
133:  run: ./scripts/test_orphans.ps1 -Cycles 100 -Scenario signal
```

`daemon-idle` não está. E `check_doc_refs.ps1`, `check_readme_anchors.ps1` e
`check_tool_params.ps1` aparecem **só** em `scripts/verify.ps1` (linhas 189,
196, 197), que nenhum job do CI invoca.

#### A decisão que esta tarefa tem de acertar
`daemon-idle` é estruturalmente diferente dos outros três cenários: o daemon não
tem pai nem stdin de host, então a vigília do pai **não se aplica** — não a
ligue por consistência. Quem substitui é a ociosidade, com `--idle-seconds`
curto no cenário, e esse valor **não pode vazar** para
`daemon.DefaultIdleSeconds` (15 min).

Os três scripts de checagem vão num job próprio (`checagens`, `ubuntu-latest`,
`shell: pwsh`), não pendurados no job de teste: eles não dependem de plataforma
e um deles falhando não deve mascarar a matriz.

#### Passos
1. Acrescentar ao job `orphans` um quarto passo, `100 ciclos - daemon ocioso`,
   chamando `./scripts/test_orphans.ps1 -Cycles 100 -Scenario daemon-idle`.
   Conferir antes, lendo o script, qual é o nome exato do cenário e se ele exige
   parâmetro adicional.
2. Criar o job `checagens` com os três scripts.
3. Conferir que `check_doc_refs.ps1` e `check_readme_anchors.ps1` passam hoje
   num runner limpo — se algum reprovar, **não silencie**: reporte o achado e
   decida se entra nesta tarefa ou vira tarefa própria.

#### Contrato de relatório
O diff do `ci.yml`. A saída local dos três scripts com `EXIT=` de cada um,
capturado **sem pipe** (pipe engole código de saída). Se o `daemon-idle` local
demorar demais para 100 ciclos, rode com `-Cycles 20` localmente e diga que o
número do CI é 100 e não foi exercitado aqui.

---

# Task 106 — `note_read` com `offset`, `next_offset` e `total_size`

**Tier: modelo principal.** O entregável é o contrato, e ele tem quatro casos.

#### Onde encaixa
Primeira da Fase 1 e a mais importante do lote: é a alternativa **A** da
revisão, a que sozinha teria fechado o incidente de campo. As Tasks 107 a 112
dependem do contrato que esta fixa.

#### O que vincula esta tarefa
- **D-R-3**: `offset` é sempre relativo ao byte 0 do arquivo; mutuamente
  exclusivo com `heading` e `block_id`; com `offset` presente,
  `include_frontmatter` é ignorado, e isso vai no `jsonschema`.
- **D-R-4**: `total_size`, `next_offset` e `truncated` correto.
- **D-R-5**: campo declarado é campo honrado.
- **Campo de API com valor fixo mente sempre.** `alias_collisions` era `0`
  literal, aparecia na resposta e nunca foi verdade.
- **Não deixe sua deliberação no código.** O `// better condition` de
  `read.go:244` sai nesta tarefa.

#### A evidência do defeito, com duas observações de campo
Da sessão de 2026-08-15: uma nota de 255.568 caracteres
(`Livros/Direito Eleitoral/13 Registro de candidatura.md`), sem heading ATX. As
duas formas de recorte falharam, sobrou ler tudo, e o host recusou. O modelo
abandonou o gobsidian, leu a nota por outro servidor MCP, gravou em arquivo
temporário e fatiou por **posição de caractere** com Python — que é exatamente
a operação que `offset` oferece.

Segundo relato, outra nota (capítulo XI, ~56 KB): *"sem `max_bytes` ajustável
nem paginação"*. `max_bytes` **é** ajustável (`*int`, padrão 100.000 em
`tools_read.go:92`); o que falta é a continuação. Que a percepção tenha sido
"sem teto ajustável" é informação sobre o schema: um teto sem par de
continuação não se parece com um controle.

O maquinário já existe dos dois lados: `vault.ReadRange(ctx, p, start, end)`
(`vault/vault.go:123`) valida a faixa e tem teto de alocação;
`service.ReadNote` já calcula `start`/`end` e já a chama; `note.Size` já está no
índice.

E o `Truncated` de hoje está errado por excesso:

```go
if req.MaxBytes > 0 && (end-start) == int64(req.MaxBytes) { // better condition
    res.Truncated = true
}
```

Uma seção que mede exatamente `max_bytes` e **não** foi cortada é reportada como
truncada.

#### A decisão que esta tarefa tem de acertar
Os quatro casos de `ReadNote`, e o que cada campo vale em cada um:

| caso | `start` | `end` | `total_size` | `next_offset` |
|---|---|---|---|---|
| nada (nota inteira) | 0 | `note.Size` | `note.Size` | `end` se clampou, senão ausente |
| `include_frontmatter:false` | `bodyOffset` | `note.Size` | `note.Size` | idem |
| `heading` / `block_id` | início da seção | fim da seção | `note.Size` | `end` se clampou, senão ausente |
| `offset:N` | `N` | `min(note.Size, N+max_bytes)` | `note.Size` | `end` se `end < note.Size`, senão ausente |

`total_size` é **sempre o tamanho do arquivo**, nunca o da faixa: é o número que
o cliente usa para saber quanto falta. `next_offset` é o byte seguinte ao último
devolvido, e **ausente** (não zero) quando não há mais nada — campo com valor
fixo mente sempre, e `0` aqui seria lido como "recomece do começo".

`truncated` passa a ser uma variável única, atribuída **no ponto onde o clamp
acontece**, e não recalculada por comparação depois. É a mesma lição do
`aliasKey`: um valor derivado em dois lugares diverge.

`offset` maior que `note.Size` é `INVALID_ARGUMENT`, não conteúdo vazio: vazio
é resposta legítima para uma nota vazia, e as duas não podem produzir a mesma
coisa. `offset` negativo, idem.

Em **lote** (`paths`), cada item carrega os próprios `total_size`,
`next_offset` e `truncated`. `ReadNotes` já monta `ReadNoteItem` por caminho
(`read.go:108-132`); os campos novos entram lá também, ou o lote fica sendo a
forma que não sabe continuar.

#### Passos
1. `service.ReadRequest`: campo `Offset *int64`. Ponteiro, não `int64` — flag
   inteira não distingue "omitida" de "definida com zero", e `offset:0` é um
   pedido legítimo e diferente de "sem offset" (que aceita
   `include_frontmatter`).
2. `service.ReadResult`: `TotalSize int64`, `NextOffset *int64`, `Truncated`
   corrigido.
3. `ReadNote`: validar exclusão mútua antes do `switch`; acrescentar o `case
   req.Offset != nil` ao `switch` de `read.go:166`, **antes** do
   `case !req.IncludeFrontmatter`, e clampar como os outros.
4. Uma variável `truncou bool` atribuída na linha do clamp
   (`read.go:229-231`). Apagar o bloco de `read.go:243-245` inteiro, comentário
   incluído.
5. `ReadBatchRequest` / `ReadNoteItem`: os três campos.
6. `mcpsrv/tools_read.go`: campo `Offset *int64` em `noteReadInput` com
   `jsonschema` que diz **as três regras** (origem no byte 0, exclusão mútua,
   `include_frontmatter` ignorado). Fiar em `ReadRequest` e em
   `ReadBatchRequest`.
7. `docs/TOOLS.md`: a tabela de `note_read` com os campos novos e o padrão de
   `max_bytes` (100.000) **declarado**, que hoje vive só no handler.

#### O teste que não é óbvio
```go
// TestReadNoteTruncatedNaoMenteQuandoAFaixaMedeExatamenteMaxBytes guarda o
// defeito do item 6: uma secao que mede exatamente max_bytes e NAO foi cortada
// era reportada como truncada.
//
// A fixture e montada ao contrario do habitual: o tamanho do corpo e escolhido
// DEPOIS, para casar exatamente com max_bytes. Um teste que so usa numeros
// redondos nunca encosta nesta condicao.
func TestReadNoteTruncatedNaoMenteQuandoAFaixaMedeExatamenteMaxBytes(t *testing.T) {
	corpo := strings.Repeat("a", 512)
	// ... monta cofre com uma nota cujo conteudo e exatamente `corpo` ...

	res, err := svc.ReadNote(ctx, service.ReadRequest{Path: "n.md", MaxBytes: 512})
	if err != nil {
		t.Fatalf("ReadNote: %v", err)
	}
	if res.Truncated {
		t.Fatalf("Truncated=true para faixa de %d bytes com max_bytes=%d: "+
			"nada foi cortado", len(res.Content), 512)
	}
	if res.NextOffset != nil {
		t.Fatalf("NextOffset=%d com o arquivo inteiro devolvido", *res.NextOffset)
	}

	// E o par: com max_bytes MENOR, truncated tem de ser verdadeiro e
	// next_offset tem de apontar para o byte seguinte.
	res2, err := svc.ReadNote(ctx, service.ReadRequest{Path: "n.md", MaxBytes: 511})
	if err != nil {
		t.Fatalf("ReadNote: %v", err)
	}
	if !res2.Truncated {
		t.Fatal("Truncated=false com 511 de 512 bytes devolvidos")
	}
	if res2.NextOffset == nil || *res2.NextOffset != 511 {
		t.Fatalf("NextOffset = %v, quer 511", res2.NextOffset)
	}
	if res2.TotalSize != 512 {
		t.Fatalf("TotalSize = %d, quer 512 (o arquivo, nao a faixa)", res2.TotalSize)
	}
}
```

```go
// TestReadNoteOffsetPaginaDoInicioAoFim prova a operacao que o incidente pediu:
// percorrer uma nota grande em pedacos, sem nunca pedir tudo.
//
// O laco e o teste. Se next_offset estiver errado por um byte, a concatenacao
// dos pedacos difere do arquivo — e essa e a unica asserção que pega erro de
// fencepost, que e o defeito esperado aqui.
func TestReadNoteOffsetPaginaDoInicioAoFim(t *testing.T) {
	conteudo := strings.Repeat("linha de texto qualquer\n", 1000) // 24.000 bytes
	// ... monta cofre ...

	var montado strings.Builder
	off := int64(0)
	for volta := 0; ; volta++ {
		if volta > 100 {
			t.Fatal("paginacao nao terminou em 100 voltas")
		}
		res, err := svc.ReadNote(ctx, service.ReadRequest{
			Path: "grande.md", Offset: &off, MaxBytes: 1000,
		})
		if err != nil {
			t.Fatalf("ReadNote(offset=%d): %v", off, err)
		}
		montado.WriteString(res.Content)
		if res.NextOffset == nil {
			break
		}
		off = *res.NextOffset
	}
	if montado.String() != conteudo {
		t.Fatalf("concatenacao dos pedacos difere do arquivo: %d bytes contra %d",
			montado.Len(), len(conteudo))
	}
}
```

#### Prova de mutação
```powershell
pwsh -File scripts/mutate.ps1 `
  -Path internal/service/read.go `
  -Anchor 'truncou = true' `
  -Replacement 'truncou = false' `
  -Test TestReadNoteTruncatedNaoMenteQuandoAFaixaMedeExatamenteMaxBytes `
  -Package ./internal/service/
```
e uma segunda, sobre a continuação:
```powershell
pwsh -File scripts/mutate.ps1 `
  -Path internal/service/read.go `
  -Anchor 'res.NextOffset = &fim' `
  -Replacement '_ = fim' `
  -Test TestReadNoteOffsetPaginaDoInicioAoFim `
  -Package ./internal/service/
```
As duas têm de sair `EXIT=0`. `EXIT=2` significa que a âncora ficou ambígua ou
que a mutação quebrou o build — falha de compilação não é cobertura; escolha
outra âncora.

#### Contrato de relatório
As duas saídas de `mutate.ps1`, literais. A resposta JSON de `note_read` com
`offset` numa nota real do `test-vault`, mostrando os três campos novos. O diff
de `docs/TOOLS.md`.

---

# Task 107 — `vault_search` devolve onde o casamento está no arquivo

**Tier: modelo barato.** A informação já existe nos dois lados; falta não jogá-la fora.

#### Onde encaixa
Segunda metade da alternativa **A**. Com a Task 106 no lugar, o modelo passa a
poder fazer `note_read(path, offset=match_offset-2000, max_bytes=8000)` — que é
o incidente inteiro resolvido com código que já existe.

#### O que vincula esta tarefa
- **D-R-4**: campo que permite continuar é contrato, não conveniência.
- **Campo de API com valor fixo mente sempre.**
- **Não afirme estado que você não verificou**: se o trecho vier vazio (nota
  somente-nuvem, nenhum termo casando), `match_offset` fica **ausente**, nunca
  zero.

#### A evidência do defeito
`search.Snippet` (`internal/search/snippet.go:19-22`) carrega `HighlightStart` e
`HighlightEnd`, e `GenerateSnippet` conhece também `winStart` e
`bestMatch.start`. `service/search.go:243-250` monta o `SearchHit` e **descarta
os três**. O `SearchHit` tem `snippet` e nada que diga onde ele está no arquivo.

#### A decisão que esta tarefa tem de acertar
`HighlightStart` é **relativo ao trecho** (`snippet.go:174` grava `relStart`).
O que o cliente precisa é absoluto no arquivo, porque é o que ele vai passar
para `note_read` — e D-R-3 diz que `offset` é sempre relativo ao byte 0.

Então `Snippet` ganha um campo novo, `MatchOffset int64`, **absoluto**, e não se
mexe no significado de `HighlightStart`/`HighlightEnd`, que já têm leitor. Dois
campos com o mesmo nome e origens diferentes é a divergência que este projeto já
pagou três vezes.

Devolver junto `snippet_offset` (o offset absoluto do primeiro byte do trecho) é
tentador e **fica de fora**: dois offsets na mesma resposta, um da janela e
outro do casamento, é uma pergunta a mais para o modelo do outro lado responder.
`match_offset` é o que ele quer — o ponto para onde navegar.

#### Passos
1. `search.Snippet`: campo `MatchOffset int64`, preenchido no mesmo ponto onde
   `HighlightStart` é calculado, com o valor **absoluto** (`bestMatch.start`, ou
   `winStart + relStart` — conferir qual dos dois é a origem certa lendo
   `snippet.go:150-180`, não presumindo).
2. `service.SearchHit`: `MatchOffset *int64` com `json:"match_offset,omitempty"`.
3. `montaSlot` (`search.go:231`) preenche a partir de `snip`. Preencher **só**
   quando `snip.Text != ""`.
4. `docs/TOOLS.md`: o campo, e a frase que ensina o encadeamento —
   `vault_search` → `match_offset` → `note_read(offset=...)`.

#### O teste que não é óbvio

Arquivo novo: `internal/service/match_offset_test.go`, **`package service_test`**
(precisa de busca). Depende da Task 106 estar aplicada — `ReadRequest.Offset`
vem de lá.

```go
package service_test

import (
	"context"
	"strings"
	"testing"

	"github.com/jonyd/gobsidian/internal/service"
)

// TestSearchMatchOffsetApontaParaOTermoNoArquivo prova o encadeamento inteiro,
// que e o que o incidente de 2026-08-15 pediu: buscar, pegar o offset, ler ali,
// achar o termo.
//
// A asserção NAO e sobre o VALOR do offset — numero conferido a mao vira
// tautologia na primeira mudanca de fixture. A asserção e que LER o arquivo
// naquele offset devolve o termo procurado.
//
// O enchimento antes do termo e obrigatorio: com o termo perto do inicio, um
// match_offset errado (zero, por exemplo) passaria por acidente.
func TestSearchMatchOffsetApontaParaOTermoNoArquivo(t *testing.T) {
	corpo := strings.Repeat("enchimento sem valor nenhum\n", 1500) +
		"a palavra procurada e xifopago aqui\n"

	svc, _, _, _ := createSearchService(t, map[string]string{
		"grande.md": corpo,
		"outra.md":  "nota sem o termo, so para o cofre nao ter uma nota so\n",
	})

	res, err := svc.Search(context.Background(), service.SearchOptions{
		Query: "xifopago",
		Limit: 5,
	})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(res.Hits) == 0 {
		t.Fatal("nenhum resultado para um termo que existe na nota")
	}

	h := res.Hits[0]
	if h.Path != "grande.md" {
		t.Fatalf("primeiro hit = %q, quer grande.md", h.Path)
	}
	if h.MatchOffset == nil {
		t.Fatal("MatchOffset ausente num hit que tem trecho")
	}
	if *h.MatchOffset == 0 {
		t.Fatal("MatchOffset = 0, e o termo esta a mais de 40 KB do inicio do arquivo")
	}

	lido, err := svc.ReadNote(context.Background(), service.ReadRequest{
		Path:     h.Path,
		Offset:   h.MatchOffset,
		MaxBytes: 64,
	})
	if err != nil {
		t.Fatalf("ReadNote(offset=%d): %v", *h.MatchOffset, err)
	}
	if !strings.HasPrefix(lido.Content, "xifopago") {
		t.Fatalf("ler em match_offset=%d deu %q, que nao comeca pelo termo",
			*h.MatchOffset, lido.Content)
	}
}

// TestSearchMatchOffsetDentroDosLimites e a propriedade que vale para TODO hit,
// e o controle do teste acima: um offset fora da faixa do arquivo nao e
// utilizavel, e devolver um numero grande seria tao ruim quanto devolver zero.
func TestSearchMatchOffsetDentroDosLimites(t *testing.T) {
	svc, _, _, _ := createSearchService(t, map[string]string{
		"a.md":         "alfa beta gama\n",
		"b.md":         strings.Repeat("beta\n", 200) + "alfa no fim\n",
		"sub/c.md":     "# Titulo\n\nalfa numa subpasta\n",
	})

	res, err := svc.Search(context.Background(), service.SearchOptions{Query: "alfa", Limit: 20})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(res.Hits) != 3 {
		t.Fatalf("hits = %d, quer 3 — a fixture nao monta o teste", len(res.Hits))
	}
	for _, h := range res.Hits {
		if h.Snippet == "" {
			t.Fatalf("%s: trecho vazio", h.Path)
		}
		if h.MatchOffset == nil {
			t.Fatalf("%s: hit COM trecho veio sem MatchOffset", h.Path)
		}
		// Le um byte no offset: se ele estiver fora da faixa, ReadNote falha, e
		// essa e a asserção — nao um numero escrito a mao.
		if _, err := svc.ReadNote(context.Background(), service.ReadRequest{
			Path: h.Path, Offset: h.MatchOffset, MaxBytes: 1,
		}); err != nil {
			t.Fatalf("%s: MatchOffset=%d nao e legivel: %v", h.Path, *h.MatchOffset, err)
		}
	}
}
```

#### Prova de mutação
```powershell
pwsh -File scripts/mutate.ps1 -Path internal/search/snippet.go -Anchor 'MatchOffset: bestMatch.start' -Replacement 'MatchOffset: 0' -Test TestSearchMatchOffsetApontaParaOTermoNoArquivo -Package ./internal/service/
```

#### Contrato de relatório
Saída de `mutate.ps1`. A resposta de `vault_search` seguida da de `note_read`
com o offset devolvido, contra o `test-vault`, coladas as duas.

---

# Task 108 — `HEADING_NOT_FOUND` que distingue três estados

**Tier: modelo barato.** Três ramos e três mensagens; a decisão está tomada abaixo.

#### Onde encaixa
Alternativa **C**. A mais barata do incidente, e a que economiza uma rodada
inteira do modelo do outro lado.

#### O que vincula esta tarefa
- **Um erro que ensina o caminho alternativo vale mais que um erro correto.**
- **`strings.Join(nil, ", ")` é `""`**, e uma mensagem que termina em
  "Disponiveis:" seguida de nada parece truncamento, não diagnóstico.

#### A evidência do defeito
Da sessão real de 2026-08-15:
```
note_read {"heading":"13.1.10 Substituição de candidatos", "path":"..."}
→ HEADING_NOT_FOUND: heading "13.1.10 Substituição de candidatos" nao encontrado. Disponiveis:
```

O modelo concluiu sozinho, por tentativa, que a nota não tinha heading nenhum, e
recomendou ao usuário voltar para o outro servidor.

O código (`service/read.go:180-187`):
```go
var alternatives []string
for _, h := range note.Headings {
    if req.HeadingLevel == 0 || h.Level == req.HeadingLevel {
        alternatives = append(alternatives, h.Text)
    }
}
return ReadResult{}, Errorf(CodeHeadingNotFound,
    "heading %q nao encontrado. Disponiveis: %s", req.Heading, strings.Join(alternatives, ", "))
```

O laço de alternativas aplica **o mesmo filtro de nível** que causou a falha —
então `heading_level: 3` numa nota só com `##` também produz lista vazia, com
headings existindo. São dois estados diferentes colapsados num.

#### A decisão que esta tarefa tem de acertar
Três estados, três mensagens, e a terceira é a que hoje não existe:

| estado | mensagem |
|---|---|
| `len(note.Headings) == 0` | `esta nota nao tem headings Markdown (#). Use note_outline para ver a estrutura, ou offset/max_bytes para recortar por posicao` |
| headings existem, nenhum casa o texto | lista os N disponíveis, **sem** aplicar o filtro de nível quando `HeadingLevel == 0` |
| headings existem, mas o filtro de nível excluiu todos | `nenhum heading de nivel N; a nota tem headings de nivel <lista dos níveis presentes>` |

A menção a `note_outline` só entra depois da Task 112. Se esta tarefa rodar
antes, cite `offset`/`max_bytes` e **deixe a pendência no plano, não no código**
— deliberação no código é o que a seção proibida do CLAUDE.md nomeia.

O mesmo tratamento vale para `writer.HeadingNotFoundError`
(`internal/writer/section.go:62-70`), que alimenta `note_append` e `note_patch`
e tem o mesmo `Alternatives` possivelmente vazio.

#### Passos
1. `service/read.go`: separar os três ramos antes de montar a mensagem.
2. Coletar os níveis presentes na nota quando o terceiro ramo dispara.
3. `writer/section.go`: mesmo tratamento no `Error()` de
   `HeadingNotFoundError`, para as tools de escrita.
4. `docs/TOOLS.md`: registrar as três mensagens como contrato, para elas não
   voltarem a divergir.

#### O teste que não é óbvio

Arquivo novo: `internal/service/heading_erro_test.go`, **`package service`**
(não precisa de busca).

```go
package service

import (
	"context"
	"strings"
	"testing"
)

// TestHeadingNotFoundDistingueOsTresEstados guarda o defeito exato do incidente
// de 2026-08-15: a mensagem que chegou ao modelo terminava em "Disponiveis:" e
// nada, o que parece truncamento e nao diagnostico.
//
// A asserção NAO e sobre o texto inteiro — mensagem e reescrita, e teste que
// fixa a frase vira atrito puro. A asserção e sobre o que TEM de aparecer para o
// modelo escolher a proxima ferramenta, e sobre o que NAO pode aparecer.
func TestHeadingNotFoundDistingueOsTresEstados(t *testing.T) {
	casos := []struct {
		nome       string
		conteudo   string
		pedido     ReadRequest
		querContem []string
		naoContem  []string
	}{
		{
			nome:     "nota sem heading algum",
			conteudo: "**13.1.10 Substituicao de candidatos**\n\ntexto corrido\n",
			pedido:   ReadRequest{Path: "n.md", Heading: "13.1.10 Substituicao de candidatos"},
			// Tem de dizer que a nota nao tem heading E oferecer a saida.
			querContem: []string{"nao tem headings", "offset"},
			// E NAO pode anunciar alternativas, porque nao ha nenhuma.
			naoContem: []string{"Disponiveis"},
		},
		{
			nome:       "headings existem, nenhum casa",
			conteudo:   "# Um\n\n## Dois\n\ntexto\n",
			pedido:     ReadRequest{Path: "n.md", Heading: "Tres"},
			querContem: []string{"Um", "Dois"},
			naoContem:  []string{"nao tem headings"},
		},
		{
			nome:     "o filtro de nivel excluiu todos",
			conteudo: "# Um\n\n## Dois\n\ntexto\n",
			pedido:   ReadRequest{Path: "n.md", Heading: "Um", HeadingLevel: 3},
			// Tem de dizer que o NIVEL e o culpado, e quais niveis existem.
			querContem: []string{"nivel"},
			naoContem:  []string{"nao tem headings"},
		},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			root := t.TempDir()
			writeFile(t, root, "n.md", c.conteudo)
			svc := newTestService(t, root)

			_, err := svc.ReadNote(context.Background(), c.pedido)
			if err == nil {
				t.Fatal("ReadNote devolveu sucesso para heading que nao existe")
			}
			if got := CodeOf(err); got != CodeHeadingNotFound {
				t.Fatalf("codigo = %v, quer %v", got, CodeHeadingNotFound)
			}

			msg := err.Error()
			for _, sub := range c.querContem {
				if !strings.Contains(msg, sub) {
					t.Errorf("a mensagem nao contem %q.\nMensagem: %s", sub, msg)
				}
			}
			for _, sub := range c.naoContem {
				if strings.Contains(msg, sub) {
					t.Errorf("a mensagem contem %q, que nao se aplica a este estado."+
						"\nMensagem: %s", sub, msg)
				}
			}
		})
	}
}

// TestHeadingNotFoundNaoTerminaEmListaVazia e a asserção mecanica que pega a
// regressao exata, independente de como a mensagem for redigida: nada pode
// anunciar uma lista com ':' e nao entregar nada depois.
func TestHeadingNotFoundNaoTerminaEmListaVazia(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "n.md", "**so negrito**\n\ntexto\n")
	svc := newTestService(t, root)

	_, err := svc.ReadNote(context.Background(), ReadRequest{Path: "n.md", Heading: "qualquer"})
	if err == nil {
		t.Fatal("ReadNote devolveu sucesso")
	}
	msg := strings.TrimSpace(err.Error())
	if strings.HasSuffix(msg, ":") {
		t.Fatalf("a mensagem termina em ':' e nada depois — parece truncamento: %q", msg)
	}
}

// TestHeadingEncontradoContinuaFuncionando e o controle. Sem ele, uma
// implementacao que recusasse TODO heading passaria nos dois testes acima.
func TestHeadingEncontradoContinuaFuncionando(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "n.md", "# Um\n\ntexto um\n\n## Dois\n\ntexto dois\n")
	svc := newTestService(t, root)

	res, err := svc.ReadNote(context.Background(), ReadRequest{Path: "n.md", Heading: "Dois"})
	if err != nil {
		t.Fatalf("heading que existe foi recusado: %v", err)
	}
	if !strings.Contains(res.Content, "texto dois") {
		t.Fatalf("conteudo = %q, quer a secao Dois", res.Content)
	}
}
```

O mesmo tratamento vale para `writer.HeadingNotFoundError`
(`internal/writer/section.go:62-70`), que alimenta `note_append` e `note_patch`.
O teste dele vive em `internal/writer` e tem a mesma forma: um caso com
`headings` vazio, um com headings que não casam, e o controle.

#### Prova de mutação
```powershell
pwsh -File scripts/mutate.ps1 -Path internal/service/read.go -Anchor 'if len(note.Headings) == 0 {' -Replacement 'if false {' -Test TestHeadingNotFoundDistingueOsTresEstados -Package ./internal/service/
```

#### Contrato de relatório
Saída de `mutate.ps1`. As três mensagens, coladas literalmente como o cliente as
receberia.

---

# Task 109 — `paths` aceitando objeto por item

**Tier: modelo barato.** Decodificação customizada, com o desenho fechado abaixo.

#### Onde encaixa
Alternativa **H**. Resolve o item 45: seis capítulos com seis seções diferentes
exigem hoje seis chamadas.

#### O que vincula esta tarefa
- **D-R-3** e **D-R-4**: o objeto por item pode sobrepor `offset` e `max_bytes`,
  e cada item devolve os próprios `total_size`/`next_offset`/`truncated`.
- **Schema que promete e código que ignora é pior que parâmetro ausente.**
- **Flag booleana ou inteira não distingue "omitida" de "definida com zero".**

#### A evidência do defeito
Relato do dono: *"heading só funciona por path único — pra 6 capítulos preciso
de 6 chamadas paralelas, não uma só com lista."*

Preciso sobre o mecanismo: `paths` **e** `heading` funcionam juntos —
`tools_read.go:96-104` repassa `Heading` a `ReadNotes`, que o aplica a **cada**
caminho (`read.go:110-118`). O que não existe é `heading` **diferente por
item**. `ReadBatchRequest` documenta a escolha: *"não há campo por-item porque o
schema de entrada é plano"*. A decisão é coerente e o custo caiu inteiro sobre o
caso mais comum — percorrer capítulos distintos de uma obra.

#### A decisão que esta tarefa tem de acertar
**Uma forma de entrada, não duas.** `paths` passa a aceitar string **ou**
objeto, misturados na mesma lista:

```json
["a.md", {"path": "b.md", "heading": "X"}, {"path": "c.md", "offset": 4000, "max_bytes": 2000}]
```

Os campos de topo (`heading`, `heading_level`, `block_id`, `max_bytes`,
`offset`, `include_frontmatter`) continuam sendo o **padrão**; o objeto
**sobrepõe** por item, campo a campo. A alternativa rejeitada é um `items:`
separado ao lado de `paths`: cria uma terceira forma de entrada e torna
**ternária** a exclusão mútua entre `path` e `paths`, que hoje é binária e
testada.

As regras de D-R-3 valem por item: um item que traga `offset` **e** `heading` é
`INVALID_ARGUMENT`, e o erro tem de dizer **qual índice da lista** falhou —
"item 3 de paths" e não "paths inválido".

Herança e sobreposição precisam distinguir "ausente" de "zero", e é aí que esta
tarefa erra se for descuidada: os campos do objeto por item são **ponteiros**.
`{"path":"b.md","max_bytes":0}` é um pedido explícito diferente de
`{"path":"b.md"}`. É a mesma armadilha de `ReadOnlySet`/`DebounceMSSet`, um
nível abaixo.

#### Passos
1. Tipo `noteReadAlvo` em `internal/mcpsrv` com `UnmarshalJSON` que aceita
   string ou objeto. O `jsonschema` do campo declara as duas formas — ou o
   modelo do outro lado não descobre a forma nova. Conferir o que a versão do
   `jsonschema-go` fixada no `go.mod` (v0.4.2) suporta antes de prometer
   `oneOf`; se ela não suportar, descreva as duas formas na descrição do campo e
   **diga isso no relatório**.
2. `service.ReadBatchRequest`: `Alvos []ReadAlvo` no lugar de `Paths []string`,
   com os campos de topo virando o padrão aplicado na montagem de cada
   `ReadRequest`.
3. Conferir que o teto `maxPathsPorLote` (`tools_read.go:83`) continua valendo,
   agora sobre a lista nova.
4. `docs/TOOLS.md`: a forma nova, com o exemplo dos seis capítulos.

#### O teste que não é óbvio

Arquivo novo: `internal/service/lote_por_item_test.go`, **`package service`**.

```go
package service

import (
	"context"
	"strings"
	"testing"
)

// ptrDe existe porque os campos do objeto por item sao PONTEIROS, e escrever
// &valor em literal de struct nao compila para constante.
//
// Nome com sufixo de proposito: um helper chamado `ptr` no pacote de teste
// colide no dia em que a implementacao criar o dela.
func ptrDe[T any](v T) *T { return &v }

// TestReadNotesObjetoPorItemSobrepoeOTopo prova as duas metades do contrato: o
// item sem objeto HERDA os campos de topo, e o item com objeto sobrepoe SO o
// campo que ele traz.
//
// O erro previsivel aqui e sobrepor o REGISTRO INTEIRO em vez de campo a campo —
// e o sintoma seria max_bytes do topo sumindo no item que so pediu heading.
// A ultima asserção e a que pega isso; sem ela o teste passa com a versao errada.
func TestReadNotesObjetoPorItemSobrepoeOTopo(t *testing.T) {
	const secoes = "# Alfa\n\nconteudo de alfa bem mais longo que dez bytes\n\n" +
		"# Beta\n\nconteudo de beta bem mais longo que dez bytes\n"

	root := t.TempDir()
	writeFile(t, root, "a.md", secoes)
	writeFile(t, root, "b.md", secoes)
	svc := newTestService(t, root)

	out := svc.ReadNotes(context.Background(), ReadBatchRequest{
		Heading:  "Alfa", // padrao do topo
		MaxBytes: 10,     // padrao do topo
		Alvos: []ReadAlvo{
			{Path: "a.md"},                         // herda os dois
			{Path: "b.md", Heading: ptrDe("Beta")}, // sobrepoe SO o heading
		},
	})

	if len(out.Items) != 2 {
		t.Fatalf("items = %d, quer 2", len(out.Items))
	}
	for i, it := range out.Items {
		if it.Err != nil {
			t.Fatalf("item %d devolveu erro: %v", i, it.Err)
		}
	}
	if out.Items[0].Section == nil || out.Items[0].Section.Text != "Alfa" {
		t.Fatalf("item 0 nao herdou o heading do topo: %+v", out.Items[0].Section)
	}
	if out.Items[1].Section == nil || out.Items[1].Section.Text != "Beta" {
		t.Fatalf("item 1 nao sobrepos o heading: %+v", out.Items[1].Section)
	}
	if !out.Items[1].Truncated {
		t.Fatal("item 1 perdeu max_bytes=10 do topo ao sobrepor o heading — " +
			"a sobreposicao trocou o registro inteiro em vez de um campo")
	}
}

// TestReadNotesZeroExplicitoNaoEOmissao guarda a armadilha de ReadOnlySet e
// DebounceMSSet, um nivel abaixo: {"path":"a.md","max_bytes":0} e um pedido
// DIFERENTE de {"path":"a.md"}, e um campo nao-ponteiro nao distingue os dois.
func TestReadNotesZeroExplicitoNaoEOmissao(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "a.md", "corpo com bastante texto para ser cortado no meio\n")
	svc := newTestService(t, root)

	out := svc.ReadNotes(context.Background(), ReadBatchRequest{
		MaxBytes: 5,
		Alvos: []ReadAlvo{
			{Path: "a.md"},                     // herda 5
			{Path: "a.md", MaxBytes: ptrDe(0)}, // ZERO explicito: sem teto
		},
	})
	if len(out.Items[0].Content) != 5 {
		t.Fatalf("item 0: %d bytes, quer 5 (herdado do topo)", len(out.Items[0].Content))
	}
	if len(out.Items[1].Content) == 5 {
		t.Fatal("item 1: max_bytes=0 explicito foi tratado como omissao e herdou 5")
	}
}

// TestReadNotesItemInvalidoDizQualItem cobre a regra D-R-3 no nivel do item.
// "paths invalido" nao ajuda quem mandou seis capitulos numa chamada.
func TestReadNotesItemInvalidoDizQualItem(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "a.md", "# Alfa\n\ntexto\n")
	svc := newTestService(t, root)

	out := svc.ReadNotes(context.Background(), ReadBatchRequest{
		Alvos: []ReadAlvo{
			{Path: "a.md"},
			{Path: "a.md", Heading: ptrDe("Alfa"), Offset: ptrDe(int64(3))},
		},
	})
	if out.Items[0].Err != nil {
		t.Fatalf("item 0, que e valido, falhou: %v", out.Items[0].Err)
	}
	if out.Items[1].Err == nil {
		t.Fatal("item com offset E heading foi aceito")
	}
	if !strings.Contains(out.Items[1].Err.Error(), "1") {
		t.Errorf("o erro nao identifica o indice do item: %v", out.Items[1].Err)
	}
}

// TestReadNotesFormaAntigaContinuaValendo e o controle de compatibilidade: a
// lista so de strings, que e o que todo cliente manda hoje, nao pode quebrar.
func TestReadNotesFormaAntigaContinuaValendo(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "a.md", "alfa\n")
	writeFile(t, root, "b.md", "beta\n")
	svc := newTestService(t, root)

	out := svc.ReadNotes(context.Background(), ReadBatchRequest{
		Alvos: []ReadAlvo{{Path: "a.md"}, {Path: "b.md"}},
	})
	if len(out.Items) != 2 || out.Items[0].Err != nil || out.Items[1].Err != nil {
		t.Fatalf("lote simples quebrou: %+v", out.Items)
	}
}
```

E o teste de decodificação, que é onde a forma mista de fato entra. Arquivo
novo: `internal/mcpsrv/lote_misto_test.go`, **`package mcpsrv_test`**.

```go
package mcpsrv_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNoteReadAceitaListaMista prova o que o schema promete: string e objeto na
// MESMA lista. Um teste que so manda objetos nao prova a mistura, que e a forma
// que o cliente real vai usar.
func TestNoteReadAceitaListaMista(t *testing.T) {
	root := t.TempDir()
	conteudo := "# Alfa\n\ntexto de alfa\n\n# Beta\n\ntexto de beta\n"
	for _, n := range []string{"a.md", "b.md"} {
		if err := os.WriteFile(filepath.Join(root, n), []byte(conteudo), 0644); err != nil {
			t.Fatal(err)
		}
	}
	session, ctx := novaSessao(t, root)

	var out struct {
		Items []struct {
			Path    string `json:"path"`
			Content string `json:"content"`
			Err     any    `json:"err"`
		} `json:"items"`
	}
	chamaTool(ctx, t, session, "note_read", map[string]any{
		"paths": []any{
			"a.md",
			map[string]any{"path": "b.md", "heading": "Beta"},
		},
	}, &out)

	if len(out.Items) != 2 {
		t.Fatalf("items = %d, quer 2", len(out.Items))
	}
	if !strings.Contains(out.Items[0].Content, "texto de alfa") {
		t.Fatalf("item 0 (string) = %q", out.Items[0].Content)
	}
	if !strings.Contains(out.Items[1].Content, "texto de beta") {
		t.Fatalf("item 1 (objeto) nao recortou a secao: %q", out.Items[1].Content)
	}
	if strings.Contains(out.Items[1].Content, "texto de alfa") {
		t.Fatal("item 1 devolveu a nota inteira; o heading por item foi ignorado")
	}
}
```

#### Prova de mutação
```powershell
pwsh -File scripts/mutate.ps1 -Path internal/service/read.go -Anchor 'if alvo.Heading != nil {' -Replacement 'if false {' -Test TestReadNotesObjetoPorItemSobrepoeOTopo -Package ./internal/service/
```

#### Contrato de relatório
Saída de `mutate.ps1`. Uma chamada real de `note_read` com lista mista
(string + objeto) contra o `test-vault`, com a resposta colada.

---

# Task 110 — a âncora do trecho passa a ser escolhida

**Tier: modelo principal.** O entregável é o algoritmo, e há três defeitos empilhados.

#### Onde encaixa
Alternativa **B**. Sem ela, `vault_search` sobre documento longo devolve sempre
o começo do arquivo — e foi isso que o usuário viu.

#### O que vincula esta tarefa
- **D-R-8**: nenhum número de desempenho entra no relatório sem medição. Esta
  tarefa mexe no caminho quente da busca; se suspeitar de custo, meça com
  `benchstat -count=6` ou escreva "não medido".
- **Otimização que muda resultado é defeito** — e o inverso vale aqui: esta é
  mudança de **trecho**, deliberada, e o golden de ranking da Task 78
  (`testdata/ranking/*.tsv`) **não pode mudar**, porque o trecho não entra no
  score. Se ele mudar, você mexeu no ranking sem querer: **pare**.
- **Chave derivada calculada em dois lugares diverge**: o IDF tem de **viajar**
  de `CalculateBM25` até o trecho, nunca ser recalculado lá.

#### A evidência do defeito
`search/snippet.go:75-100`:
```go
for _, termStr := range queryTerms {      // ordem da CONSULTA
    for _, tok := range Analyze(termStr) {
        for _, t := range termsToSearch {
            posicoes := ix.Positions(t, string(cPath))
            if len(posicoes) > 0 {
                bestMatch = &matchPos{ start: posicoes[0].Start, ... }  // PRIMEIRA do documento
```

A consulta `"13.1.10 Substituição de candidatos"` tokeniza como
`13, 1, 10, substituicao, de, candidatos`. O primeiro termo é `13`; a primeira
ocorrência de `13` no documento é o título `**13**`, no offset ~0. O trecho
ancora ali e nunca chega perto da seção 13.1.10.

A variável se chama `bestMatch` e não há escolha nenhuma: é a primeira que
aparece.

#### A decisão que esta tarefa tem de acertar
Três mudanças, em ordem de retorno decrescente, e **cada uma é independente**:

1. **Consulta entre aspas ancora na posição do casamento da frase.** É a mais
   barata, a mais correta e a que fecha o caso observado.
   `matchPhraseInNote` (`service/search.go:411`) já **calcula** essa posição
   para decidir se o hit entra, e joga o valor fora. Passa a devolver
   `(bool, int64)`, e `service.Search` carrega o valor até `GenerateSnippet`
   como âncora sugerida. Âncora sugerida que não exista nas posições do índice é
   **ignorada**, não confiada.
2. **Sem frase, ancorar no termo de maior IDF.** O IDF já é calculado em
   `CalculateBM25` (`bm25.go:110-135`, mapa `termIDFs`) e descartado antes de
   chegar ao trecho. Ele precisa sobreviver — `search.Result` carrega o termo
   mais seletivo que casou naquele documento, ou `CalculateBM25` devolve o mapa
   junto. **Não recalcular** IDF dentro de `GenerateSnippet`.
3. **Janela deslizante que maximize termos distintos da consulta.** É o
   algoritmo padrão e o mais caro dos três. Se o orçamento da tarefa acabar,
   entregue 1 e 2 inteiros e **diga que a 3 ficou de fora** — escopo não encolhe
   em silêncio, mas encolher declaradamente é decisão legítima.

A variável passa a se chamar o que ela é. `bestMatch` só é `best` depois desta
tarefa.

#### Suspeita a testar nesta tarefa, e ela é da revisão
O segundo resultado da sessão (`Resumo - Claude - Direito Eleitoral.md`) muito
provavelmente **não contém** a frase `13.1.10 Substituição de candidatos` e
ainda assim passou pelo filtro de frase. Monte a fixture: um corpo com `13`,
`1`, `10` e `candidatos` espalhados e nunca contíguos, e afirme que
`matchPhraseInNote` devolve `false`. Se devolver `true`, você acabou de
reproduzir o defeito da Task 111 — registre com a saída e siga.

#### O teste que não é óbvio
```go
// TestSnippetAncoraNoTermoSeletivoENaoNoPrimeiroDaConsulta reproduz o incidente
// de 2026-08-15 em miniatura.
//
// A fixture e desenhada para que a ordem da consulta e a seletividade apontem
// para lugares OPOSTOS: "13" aparece no offset ~0 e mais 200 vezes; "xifopago"
// aparece uma vez so, muito adiante. Um trecho ancorado no primeiro termo da
// consulta cai no comeco do arquivo; um ancorado no mais seletivo cai na secao.
func TestSnippetAncoraNoTermoSeletivoENaoNoPrimeiroDaConsulta(t *testing.T) {
	var b strings.Builder
	b.WriteString("13 capitulo treze\n\n")
	for i := 0; i < 200; i++ {
		b.WriteString("13 mencao de enchimento sem valor nenhum aqui\n")
	}
	b.WriteString("\n13.1.10 xifopago de candidatos\n")
	b.WriteString("o texto da secao que o usuario queria ler\n")
	// ... monta cofre com essa nota ...

	res, err := svc.Search(ctx, service.SearchOptions{
		Query: "13 xifopago", Limit: 5, SnippetChars: 200,
	})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(res.Hits) == 0 {
		t.Fatal("nenhum resultado")
	}
	if !strings.Contains(res.Hits[0].Snippet, "xifopago") {
		t.Fatalf("o trecho nao contem o termo seletivo; ancorou no comeco do "+
			"arquivo. Trecho: %q", res.Hits[0].Snippet)
	}
}

// TestSnippetDeFraseContemAFrase e a asserção que o usuario faria: a busca por
// frase exata achou a nota certa e devolveu um trecho que nao contem a frase.
func TestSnippetDeFraseContemAFrase(t *testing.T) {
	// mesma fixture, consulta entre aspas
	res, _ := svc.Search(ctx, service.SearchOptions{
		Query: `"xifopago de candidatos"`, Limit: 5, SnippetChars: 200,
	})
	if len(res.Hits) == 0 {
		t.Fatal("busca por frase nao achou a nota que contem a frase")
	}
	if !strings.Contains(res.Hits[0].Snippet, "xifopago de candidatos") {
		t.Fatalf("trecho da busca por frase nao contem a frase: %q", res.Hits[0].Snippet)
	}
}
```

#### Prova de mutação
A mutação tem de provar que **voltar ao comportamento de hoje reprova o teste
novo**. Âncora dentro da função de escolha:
```powershell
pwsh -File scripts/mutate.ps1 -Path internal/search/snippet.go -Anchor 'if idf > melhorIDF {' -Replacement 'if false {' -Test TestSnippetAncoraNoTermoSeletivoENaoNoPrimeiroDaConsulta -Package ./internal/service/
```
Se sair `EXIT=2` porque a mutação deixou variável sem uso e quebrou o build,
troque por uma forma que compile (`if false && idf > melhorIDF {`) — falha de
compilação não é cobertura.

#### Contrato de relatório
Saída de `mutate.ps1`. Os dois trechos — antes e depois — colados para a mesma
consulta e a mesma nota. E `git diff --stat internal/service/testdata/ranking/`
vazio, que é a prova de que o score não se moveu.

---

# Task 111 — frase exata: adjacência verificada, não adivinhada

**Tier: modelo principal.** A decisão de mecanismo é o entregável.

#### Onde encaixa
Item 29. É correção, não desempenho, e explica o falso positivo suspeitado na
Task 110.

#### O que vincula esta tarefa
- **Um teste que não pode falhar é pior que teste ausente.** A regra de hoje é
  testada por `"foo bar"`, que passa; nenhum teste usa pontuação real.
- **Reparar metade do estado é pior que não reparar.** Corrigir o falso negativo
  (pontuação) e deixar o falso positivo (tokens só perto) é reparar metade.
- **D-R-6**: o formato do cache não muda neste lote.

#### A evidência do defeito
`service/search.go:432`:
```go
if posN.Start >= currEnd && posN.Start <= currEnd+1 {
```
Exige que o token seguinte comece no máximo **1 byte** depois do fim do
anterior. `"foo bar"` casa; `"foo  bar"` (dois espaços), `"foo, bar"`,
`"foo — bar"` (travessão, 3 bytes em UTF-8) e quebra de linha CRLF **não
casam**. E o inverso: nada impede que dois tokens não consecutivos no texto
casem, desde que estejam a um byte de distância — origem do falso positivo
suspeitado.

#### A decisão que esta tarefa tem de acertar
Duas saídas, e a escolhida é a primeira:

**Escolhida — verificar os bytes entre as duas posições.** Adjacência de token
significa: entre `currEnd` e `posN.Start` **não há letra nem dígito**. É exato,
não muda o formato do cache, e o custo é limitado. Regras:
- ler **uma vez por nota candidata**, não uma vez por par de posições;
- teto de bytes lidos por verificação — um par com distância absurda entre as
  posições é rejeitado **sem** ler;
- consulta sem aspas não paga nada: `matchPhraseInNote` já sai cedo com
  `len(phraseTokens) <= 1`;
- nota somente-nuvem **não é aberta** — a mesma regra não negociável de sempre.
  Sem os bytes, o hit de frase é **descartado**, não aceito por otimismo.

**Rejeitada para este lote — ordinal de token no índice.** É a resposta
estrutural: `TokenPosition{Start,End}` não sabe qual token é, e guardar o
ordinal (varint, delta-encoded) habilita frase exata, proximidade (`NEAR/3`) e
ranking por proximidade de uma vez. Custa **formato 7** do cache, e portanto
reconstrução do índice de busca em todo cofre no boot seguinte. Fica na Fase 6.

Se a leitura de bytes se mostrar cara na medição, o resultado **não** é voltar
ao `+1`: é levar a decisão de volta para quem pediu, com o número medido.

#### O teste que não é óbvio
```go
// TestMatchPhraseComPontuacaoReal cobre os dois lados do defeito de uma vez.
//
// Os casos "quer=true" sao os falsos NEGATIVOS de hoje: separadores que a regra
// de +1 byte rejeita. Os "quer=false" sao os falsos POSITIVOS: tokens perto, mas
// com outra palavra no meio ou fora de ordem.
func TestMatchPhraseComPontuacaoReal(t *testing.T) {
	casos := []struct {
		corpo string
		frase string
		quer  bool
	}{
		{"o foo bar aqui", "foo bar", true},
		{"o foo  bar aqui", "foo bar", true},   // dois espacos
		{"o foo, bar aqui", "foo bar", true},   // virgula
		{"o foo — bar aqui", "foo bar", true},  // travessao, 3 bytes
		{"o foo\r\nbar aqui", "foo bar", true}, // CRLF
		{"o foo.bar aqui", "foo bar", true},    // ponto
		{"o foo baz bar aqui", "foo bar", false},              // palavra no meio
		{"o bar foo aqui", "foo bar", false},                  // ordem invertida
		{"13 e 1 e 10 e candidatos", "13.1.10 candidatos", false}, // o caso do incidente
	}
	// ... para cada caso: indexa o corpo num cofre temporario, chama a busca por
	//     frase e compara com `quer` ...
}
```

O último caso é o que a Task 110 suspeitou. Se ele **passar antes** da correção,
o falso positivo estava confirmado; registre isso no relatório com a saída.

#### Prova de mutação
```powershell
pwsh -File scripts/mutate.ps1 -Path internal/service/search.go -Anchor 'if !separadorSemToken(' -Replacement 'if false && !separadorSemToken(' -Test TestMatchPhraseComPontuacaoReal -Package ./internal/service/
```

#### Contrato de relatório
Saída de `mutate.ps1`. A tabela de casos com o veredito **antes** e **depois**,
que é o que mostra quantos falsos negativos existiam. Se mediu custo, o
`benchstat`; se não mediu, escreva "não medido".

---

# Task 112 — `note_outline`, o mapa da nota

**Tier: modelo principal.** Tool nova, e a decisão de o que ela promete é o entregável.

#### Onde encaixa
Alternativa **D**, escolhida em D-R-1. Última da Fase 1, e a que fecha o
incidente para nota convertida — que é a maioria de um cofre de estudo.

#### O que vincula esta tarefa
- **D-R-1**: o parser **não** muda. `Heading.Synthetic` é a alternativa E, e é
  decisão de produto pendente.
- **D-R-6**: `IndexCacheParserVersion` não sobe. Os candidatos sintéticos são
  calculados **na chamada**, sobre os bytes da nota, e não são persistidos.
- **D-R-3**: os offsets que esta tool devolve são absolutos, iguais aos que
  `note_read(offset=)` aceita — ou ela não serve para nada.
- **D-R-5**: campo declarado é campo honrado.
- **Cofre inacessível e cofre vazio não podem produzir a mesma resposta** — aqui:
  nota ilegível e nota sem estrutura não podem.

#### O problema que ela resolve
`parseATXHeading` (`parser/headings.go:93`) só aceita ATX (`#`). Não aceita
parágrafo em negrito — `**13.1.10 Substituição de candidatos**`, que é o formato
da nota do incidente e o que qualquer conversão de PDF/DOCX/EPUB produz — nem
setext (`Título` seguido de `====` ou `----`), que é CommonMark válido.

Consequência: `note_read` por heading, `note_patch` por seção, âncora de
wikilink e o peso de heading do BM25 não funcionam em nenhuma nota convertida de
livro. `note_outline` não conserta nenhuma das quatro — conserta a pergunta
real, que é *"não sei onde está o que eu quero neste arquivo de 255 KB"*.

#### A decisão que esta tarefa tem de acertar
**A resposta separa o que é estrutura do que é palpite, e diz qual é qual.**

```json
{
  "path": "Livros/.../13 Registro de candidatura.md",
  "total_size": 255568,
  "headings": [ {"level":1, "text":"...", "start":0, "end":1200, "slug":"..."} ],
  "candidates": [
    {"kind":"strong_paragraph", "text":"13.1.10 Substituição de candidatos",
     "level":3, "start":198340, "end":204112},
    {"kind":"setext", "text":"...", "level":1, "start":0, "end":0}
  ],
  "truncated": false
}
```

Regras que não se negociam dentro da tarefa:

- `headings` vem do índice (`note.Headings`), sem reler o disco. `candidates` é
  calculado lendo a nota — e por isso a tool **recusa nota somente-nuvem**, como
  todas as outras que abrem arquivo.
- `candidates` **nunca** entra em `headings`, nem no índice, nem no cache. Uma
  tool que afirma estrutura que o arquivo não tem é a classe de defeito do item
  7, um nível acima.
- `level` de candidato com numeração hierárquica sai da profundidade da
  numeração: `13` → 1, `13.1` → 2, `13.1.10` → 3. Sem numeração, `level` é
  **ausente**, não 0 — zero literal mente.
- `end` de um candidato é o `start` do próximo candidato de nível menor ou
  igual, ou o fim do arquivo — a mesma regra de `closeSections`
  (`parser/headings.go:78`). **Reusar** a função, não reimplementá-la.
- Nota grande produz lista longa. Teto declarado no schema, `truncated` no
  retorno, e o teto **não é silencioso** — item 34 é exatamente esse defeito.
- Detecção de negrito acontece **fora de bloco de código cercado**. O laço de
  `ExtractHeadings` já rastreia cercas (`inFence`, `openFence`, `closesFence`);
  esta tarefa **reusa** essas funções. Se forem privadas ao pacote, ponha a
  detecção de candidatos dentro de `internal/parser` — não duplique a máquina de
  cercas, porque duas máquinas de cerca divergem.

#### Passos
1. `internal/parser/outline.go`: `DetectCandidates(body []byte, bodyOffset int64) []Candidate`,
   com as três formas e o rastreamento de cerca reusado.
2. `internal/service/outline.go`: `Outline(ctx, req)` — resolve o caminho, recusa
   somente-nuvem, lê a nota, monta `headings` do índice e `candidates` do parser.
3. `internal/mcpsrv/tools_read.go`: registrar a tool com o schema.
4. `docs/TOOLS.md`: contrato da tool, incluindo a frase que ensina o
   encadeamento — `note_outline` → `start` → `note_read(offset=start, max_bytes=…)`.
5. Voltar à Task 108 e acrescentar a menção a `note_outline` na mensagem de
   "nota sem headings".
6. Fixture nova: uma nota **realista** convertida — negrito como título, CRLF,
   sem heading ATX, dezenas de KB. As fixtures atuais são notas Obsidian
   idiomáticas, e nelas todo o produto funciona; a premissa "cofre Obsidian
   idiomático" nunca foi escrita como premissa, e por isso nunca foi
   questionada.
7. `README.md` e `docs/ESTRUTURA.md`: a tool nova entra na contagem de tools,
   que hoje é doze e passa a ser treze. `check_readme_anchors.ps1` confere as
   âncoras.

#### O teste que não é óbvio
```go
// TestOutlineNaoConfundeCandidatoComHeading e a asserção central da tool:
// ela pode ERRAR na deteccao de candidato sem causar dano, mas nao pode
// apresentar candidato como heading.
func TestOutlineNaoConfundeCandidatoComHeading(t *testing.T) {
	conteudo := "**13 Capitulo**\r\n\r\ntexto\r\n\r\n" +
		"**13.1.10 Substituicao de candidatos**\r\n\r\n" +
		"o texto da secao\r\n\r\n" +
		"```\r\n**isto esta dentro de bloco de codigo**\r\n```\r\n"
	// ... monta cofre ...

	out, err := svc.Outline(ctx, service.OutlineRequest{Path: "conv.md"})
	if err != nil {
		t.Fatalf("Outline: %v", err)
	}
	if len(out.Headings) != 0 {
		t.Fatalf("nota sem heading ATX devolveu %d headings: %+v",
			len(out.Headings), out.Headings)
	}
	if len(out.Candidates) != 2 {
		t.Fatalf("quer 2 candidatos (o de dentro da cerca nao conta), tem %d: %+v",
			len(out.Candidates), out.Candidates)
	}
	if out.Candidates[1].Level == nil || *out.Candidates[1].Level != 3 {
		t.Fatalf("13.1.10 devia dar nivel 3, deu %v", out.Candidates[1].Level)
	}
}

// TestOutlineOffsetAlimentaNoteRead prova que os dois lados falam a mesma
// coordenada. Sem isto, a tool devolve numeros bonitos e inuteis.
func TestOutlineOffsetAlimentaNoteRead(t *testing.T) {
	out, _ := svc.Outline(ctx, service.OutlineRequest{Path: "conv.md"})
	c := out.Candidates[1]
	lido, err := svc.ReadNote(ctx, service.ReadRequest{
		Path: "conv.md", Offset: &c.Start, MaxBytes: int(c.End - c.Start),
	})
	if err != nil {
		t.Fatalf("ReadNote(offset=%d): %v", c.Start, err)
	}
	if !strings.HasPrefix(lido.Content, "**13.1.10") {
		t.Fatalf("ler no offset do candidato deu %q", lido.Content[:60])
	}
}
```

#### Prova de mutação
```powershell
pwsh -File scripts/mutate.ps1 -Path internal/parser/outline.go -Anchor 'if inFence {' -Replacement 'if false {' -Test TestOutlineNaoConfundeCandidatoComHeading -Package ./internal/service/
```

#### Contrato de relatório
Saída de `mutate.ps1`. A resposta de `note_outline` sobre a fixture convertida,
colada inteira. E — se a nota do incidente estiver disponível no cofre real — a
resposta sobre ela, com a contagem de candidatos.

---

# Task 113 — `vault.Resolve` como portão único de toda escrita

**Tier: modelo principal.** É a superfície de segurança, e o modo de falha de
uma correção parcial é ficar verde com o buraco aberto.

#### Onde encaixa
Primeira da Fase 2 e o defeito mais grave do relatório: escrita arbitrária no
sistema de arquivos, pela tool que um modelo controla, no Windows — que é a
plataforma primária do projeto.

#### O que vincula esta tarefa
- **D-R-2**: a correção é `vault.Resolve`. `os.Root` é fase própria e **não**
  entra aqui.
- **Reparar metade do estado é pior que não reparar.** Se `Resolve` virar
  portão em `CreateNote` e não em `MoveNote`, o buraco continua aberto pelo
  caminho menos usado — que é exatamente como este projeto costuma perder um
  defeito.
- **Confinamento de caminho tem duas camadas**, e as duas são necessárias:
  `validateLocal` (léxica: NUL, `..`, raiz, `IsLocal`, regra de plataforma) e
  `Canonicalize` (por componente, via `filepath.Rel`).
- **`filepath.IsLocal` barra nome de dispositivo** — `Resolve(root, "COM1")`
  escrevia em porta serial antes disso.
- **A regra de ponto/espaço no fim de componente vale só no Windows**: em Linux
  `Notas ` é nome legal, e rejeitá-lo lá torna notas reais inalcançáveis. Isso
  já está resolvido dentro de `validatePlatformPath`; não reimplemente.

#### A evidência medida do defeito
Sonda executada em 2026-08-16, com raiz de cofre num `t.TempDir()`:

```
in="..\\..\\x.md"   check=<nil>                              clean="../../x.md"
                    abs="C:\Users\jonyd\AppData\Local\Temp\x.md"
                    vault.Resolve=caminho fora do cofre: "../../x.md" sobe acima da raiz do cofre
in="../../x.md"     check=caminho "../../x.md" fora do cofre clean="../../x.md"
in="..\\..\\Windows\\Temp\\x.md"
                    check=<nil>
                    abs="C:\Users\jonyd\AppData\Local\Temp\Windows\Temp\x.md"
```

`checkWriteAllowed` (`internal/service/write.go:75-83`) é uma checagem caseira:
```go
if strings.Contains(path, "../") || strings.HasPrefix(path, "/") || (len(path) > 1 && path[1] == ':')
```
Em Linux a barra invertida não é separador, então `Contains("../")` pega o caso
equivalente — é bug **específico de Windows**. Depois disso, `CreateNote:92` e
`MoveNote:420` fazem `vault.CanonicalPath(cleanPath)`, construção direta do
tipo, **forjando a prova** que o comentário de `CanonicalPath` promete ("o tipo
e a prova de que o confinamento ja rodou"). `WriteAtomic` cria o temporário em
`filepath.Dir(target)` e renomeia — os dois fora do cofre.

`AppendNote`, `PatchNote` e `DeleteNote` **não** têm o buraco: passam por
`index.ResolvePath`, que só resolve o que já está indexado. São `CreateNote` e o
**destino** de `MoveNote`.

#### A decisão que esta tarefa tem de acertar
1. **`Resolve` como portão único.** `CreateNote` e `MoveNote` (destino) chamam
   `vault.Resolve(s.vault.Root(), req.Path)` e usam o `CanonicalPath` que ele
   devolve. `checkWriteAllowed` fica com uma responsabilidade só — o modo
   somente-leitura — e o nome muda para dizer isso (`checkWritable`, ou o que
   couber).
2. **`CanonicalPath` deixa de ser construível fora de `vault`.** As duas
   conversões de `write.go` saem. As de `write.go:712,720` (caminho da lixeira)
   também passam a derivar de `Resolve` ou de uma função de `vault` que produza
   o caminho da lixeira — não por conversão.
3. **O mapeamento de erro é contrato.** `vault.Resolve` devolve quatro
   sentinelas de propósito, e cada uma vira um código MCP diferente:
   `ErrOutsideVault` → `PATH_OUTSIDE_VAULT`, `ErrAbsolutePath` →
   `PATH_OUTSIDE_VAULT` (ou código próprio, se já existir),
   `ErrEmptyPath`/`ErrInvalidPath` → `INVALID_ARGUMENT`. Colapsar as quatro numa
   torna a mensagem a única informação útil, que é o que o comentário de
   `path.go:27` diz.
4. **Uma checagem estrutural, não uma lista de nomes.** Não acrescente
   `strings.Contains(path, "..\\")` ao `checkWriteAllowed`. Isso conserta o caso
   e deixa a categoria aberta — e é a diferença entre este item e o item 4.

#### O que a revisão pede e esta tarefa executa
*"Um teste de mutação vale aqui: apague `checkWriteAllowed` inteira e veja
quantos testes falham — a suspeita é que nenhum cobre `..\`."* Faça isso
**antes** de corrigir, e cole o resultado. Se nenhum teste falhar, você acabou
de medir a cobertura real da superfície de escrita, e esse número vai no
relatório.

#### O teste que não é óbvio
```go
// TestEscritaRecusaTravessiaComSeparadorDoWindows e o teste que faltava.
//
// A forma com barra invertida NAO e coberta hoje, e e a unica que escapa: em
// Linux a barra invertida nao e separador, entao o mesmo caso de teste roda nos
// dois sistemas e so significa alguma coisa no Windows. Ele fica sem
// runtime.GOOS de proposito — a asserção "recusou" vale nos dois, e no Linux
// ela e trivialmente verdadeira. Um teste que so roda no Windows e um teste que
// ninguem ve reprovar.
func TestEscritaRecusaTravessiaComSeparadorDoWindows(t *testing.T) {
	fora := filepath.Join(t.TempDir(), "fora")
	if err := os.MkdirAll(fora, 0o755); err != nil {
		t.Fatal(err)
	}
	// ... monta cofre em <tmp>/cofre, service em modo escrita ...

	casos := []string{
		`..\..\x.md`,
		`..\x.md`,
		`sub\..\..\x.md`,
		`../../x.md`,
		`/etc/passwd`,
		`C:\Windows\Temp\x.md`,
		"COM1",
		"nota\x00.md",
	}
	for _, c := range casos {
		t.Run(c, func(t *testing.T) {
			_, err := svc.CreateNote(ctx, service.CreateNoteRequest{Path: c, Content: "x"})
			if err == nil {
				t.Fatalf("CreateNote(%q) devolveu sucesso", c)
			}
			// A asserção mais forte: nada foi escrito FORA do cofre.
			// Sem isto, um erro devolvido depois da escrita passaria.
			entradas, _ := os.ReadDir(filepath.Dir(cofreRoot))
			for _, e := range entradas {
				if strings.HasSuffix(e.Name(), ".md") {
					t.Fatalf("CreateNote(%q) gravou %q fora do cofre", c, e.Name())
				}
			}
			// E o mesmo pelo destino de note_move.
			_, err = svc.MoveNote(ctx, service.MoveNoteRequest{From: "existente.md", To: c})
			if err == nil {
				t.Fatalf("MoveNote(to=%q) devolveu sucesso", c)
			}
		})
	}
}
```

A segunda asserção — varrer o diretório **pai** do cofre — é o que distingue
este teste de um que só confere o erro devolvido. `WriteAtomic` grava o
temporário antes do rename; um erro devolvido depois disso deixaria arquivo
fora do cofre com o teste verde.

#### Prova de mutação
Duas, e as duas são obrigatórias — uma por tool, porque a armadilha aqui é
consertar uma e não a outra:
```powershell
pwsh -File scripts/mutate.ps1 -Path internal/service/write.go -Anchor '_, canonical, err := vault.Resolve(s.vault.Root(), req.Path)' -Replacement 'canonical, err := vault.CanonicalPath(filepath.ToSlash(filepath.Clean(req.Path))), error(nil)' -Test TestEscritaRecusaTravessiaComSeparadorDoWindows -Package ./internal/service/

pwsh -File scripts/mutate.ps1 -Path internal/service/write.go -Anchor '_, canonicalTo, err := vault.Resolve(s.vault.Root(), req.To)' -Replacement 'canonicalTo, err := vault.CanonicalPath(filepath.ToSlash(filepath.Clean(req.To))), error(nil)' -Test TestEscritaRecusaTravessiaComSeparadorDoWindows -Package ./internal/service/
```

#### Contrato de relatório
A medição de cobertura **antes** (quantos testes falham ao apagar
`checkWriteAllowed`). As duas saídas de `mutate.ps1`. A saída de
`scripts/check_net.ps1` e de `verify.ps1` inteiro — esta tarefa mexe na camada
que todo o resto usa.

---

# Task 114 — a fixture Unicode que não existe

**Tier: modelo barato.** Fixture e asserção; o desenho está fechado abaixo.

#### Onde encaixa
Logo depois da 113, porque é a mesma superfície e porque a 113 é o que faz
`validatePlatformPath` — a única regra que hoje olha a forma do componente —
passar a rodar em `note_create` e `note_move`.

#### O que vincula esta tarefa
- **`CanonicalPath` não garante a grafia do disco.** Esta camada não consulta o
  disco; preserva o que o chamador passou.
- **Chave derivada calculada em dois lugares diverge.**
- **Não afirme estado que você não verificou** — em especial: não afirme que o
  produto "suporta" uma forma Unicode que nenhum teste exercita.

#### O que a revisão diz, e o que está confirmado
**O travessão em si é seguro.** `—` (U+2014) é só bytes UTF-8, sem significado
para `filepath.Clean`, para `WriteAtomic` ou para o NTFS. Não é caractere
reservado do Windows (`< > : " / \ | ? *`), e `validatePlatformPath` recusa
ponto e espaço no fim de componente, não travessão.

**O risco real é normalização Unicode, e ele existe.** Confirmado por leitura:
as quatro derivações de chave do índice aplicam **só caixa**, nunca NFC/NFD:

| ponto | derivação |
|---|---|
| `index.publishNameLocked:163` | `strings.ToLower(string(path))` |
| `index.aliasKey` | `strings.ToLower(alias)` |
| `index.nomeChave` | `strings.ToLower(...)` |
| `index.ResolvePath:271` | `strings.ToLower(filepath.ToSlash(input))` |

Enquanto `text.Normalize` (`internal/text/normalize.go:27`) e `parser.Slug`
(`internal/parser/slug.go:19`) aplicam `NFD → remove marcas → NFC`. As duas
formas convivem no mesmo produto, para propósitos diferentes, sem que nada diga
onde uma acaba.

Consequência: `Capítulo` gravado em NFD e pedido em NFC são **strings
diferentes**, e `ResolvePath` devolve "not found" para uma nota que existe. Num
cofre só de Windows quase nunca aparece; aparece em cofre sincronizado com macOS
(que historicamente normaliza para NFD) — que é o cenário OneDrive que o projeto
suporta. E este é um cofre em português, onde acento é a regra.

#### A decisão que esta tarefa tem de acertar
**Esta tarefa mede antes de corrigir.** O entregável primeiro é a fixture que
responde se o defeito existe no produto como ele está:

`note_create` seguido de `note_read` com o mesmo caminho contendo
(a) travessão, (b) acento em NFC, (c) o mesmo acento em NFD, (d) emoji fora do
BMP. E o cruzado, que é o caso real: **criar em NFD e ler em NFC**, e vice-versa.

Se o cruzado falhar — e a leitura do código diz que vai —, a correção é
normalizar a chave numa **única** função, do jeito que `aliasKey` já é. Regras:
- a forma canônica escolhida é **NFC**, porque é o que a maioria dos clientes
  envia e o que o Go emite por padrão;
- a normalização vale para a **chave**, nunca para o `CanonicalPath` guardado —
  o caminho gravado continua sendo a grafia do disco, senão o servidor passa a
  abrir arquivo que não existe;
- **uma** função (`chaveDeCaminho(path)`), e **todo** acesso passa por ela,
  inclusive os que já estavam certos. Não é para consertar os errados: é para
  tornar a próxima divergência impossível sem tocar na função.

Se a correção se mostrar maior que a tarefa comporta, **entregue a fixture com
os casos marcados como falhando** (`t.Skip` com o motivo escrito, ou lista de
casos conhecidos), registre no ledger e abra tarefa própria. Fixture que
documenta o defeito vale mais que correção apressada na camada de chaves do
índice.

#### O teste que não é óbvio

Arquivo novo: `internal/service/caminho_unicode_test.go`, **`package service`**.

```go
package service

import (
	"context"
	"testing"
)

// TestCaminhoUnicodeIdaEVolta e o teste CRUZADO: o que importa nao e criar e ler
// com a MESMA string, e sim criar numa forma de normalizacao e ler na outra.
//
// Criar e ler com a mesma string passa hoje e nao prova nada — as duas pontas
// derivam a chave do mesmo texto. O cruzado e o que um cofre sincronizado com
// macOS produz, e este e um cofre em portugues, onde acento e a regra.
//
// A reindexacao entre escrever e ler e feita construindo um Service NOVO sobre a
// mesma raiz — NUNCA por time.Sleep esperando watcher, que mede a maquina.
func TestCaminhoUnicodeIdaEVolta(t *testing.T) {
	const nfc = "Cap\u00edtulo I.md"  // í precomposto, U+00ED
	const nfd = "Capi\u0301tulo I.md" // i + acento combinante, U+0301

	casos := []struct {
		nome string
		cria string
		le   string
	}{
		{"travessao", "Nota \u2014 com travessao.md", "Nota \u2014 com travessao.md"},
		{"nfc ida e volta", nfc, nfc},
		{"nfd ida e volta", nfd, nfd},
		{"cria em NFD, le em NFC", nfd, nfc},
		{"cria em NFC, le em NFD", nfc, nfd},
		{"emoji fora do BMP", "Nota \U0001F600.md", "Nota \U0001F600.md"},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			root := t.TempDir()
			svc := newTestService(t, root)

			if _, err := svc.CreateNote(context.Background(), CreateNoteRequest{
				Path:    c.cria,
				Content: "conteudo",
			}); err != nil {
				t.Fatalf("CreateNote(%q): %v", c.cria, err)
			}

			// Reindexa de forma deterministica: Service novo sobre a mesma raiz.
			svc = newTestService(t, root)

			res, err := svc.ReadNote(context.Background(), ReadRequest{Path: c.le})
			if err != nil {
				t.Fatalf("criou %q e ReadNote(%q) falhou: %v", c.cria, c.le, err)
			}
			if res.Content != "conteudo" {
				t.Fatalf("conteudo = %q, quer %q", res.Content, "conteudo")
			}
		})
	}
}

// TestCaminhoUnicodeSobreviveAoMove cobre a segunda superficie pelo mesmo
// mecanismo: note_move constroi CanonicalPath para o DESTINO, e e o outro lugar
// onde a chave nasce.
func TestCaminhoUnicodeSobreviveAoMove(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "origem.md", "conteudo\n")
	svc := newTestService(t, root)

	const destino = "Cap\u00edtulo \u2014 I.md"
	if _, err := svc.MoveNote(context.Background(), MoveNoteRequest{
		From: "origem.md", To: destino,
	}); err != nil {
		t.Fatalf("MoveNote para %q: %v", destino, err)
	}

	svc = newTestService(t, root)
	if _, err := svc.ReadNote(context.Background(), ReadRequest{Path: destino}); err != nil {
		t.Fatalf("moveu para %q e ReadNote falhou: %v", destino, err)
	}
}
```

**Nota sobre o que medir antes de corrigir.** Rode este arquivo **contra o HEAD**
antes de tocar em `internal/index`. Os casos cruzados provavelmente reprovam, e a
tabela de veredito antes/depois é o entregável desta tarefa — é ela que diz se o
defeito existe. Se os cruzados reprovarem e a correção da camada de chaves não
couber, **entregue a fixture com os cruzados marcados e abra tarefa própria**:
fixture que documenta o defeito vale mais que correção apressada na chave do
índice.

#### Prova de mutação
Se a correção de chave entrar:
```powershell
pwsh -File scripts/mutate.ps1 -Path internal/index/index.go -Anchor 'lower := chaveDeCaminho(string(path))' -Replacement 'lower := strings.ToLower(string(path))' -Test TestCaminhoUnicodeIdaEVolta -Package ./internal/service/
```

#### Contrato de relatório
A tabela de casos com o veredito **antes** da correção — é a medição que diz se
o defeito existe. Depois, a mesma tabela verde e a saída de `mutate.ps1`. Se a
correção ficou de fora, diga na primeira linha e registre a tarefa nova.

---

# Task 115 — o laço de boot do índice de busca passa pelo guarda

**Tier: modelo principal.** O entregável é fazer o teste exercitar produção, e
esse é o defeito de verdade.

#### Onde encaixa
O segundo crítico. É a armadilha que o CLAUDE.md registra ("trocar um crash por
download síncrono em massa... trava a máquina do usuário sem dizer por quê")
**aberta em produção**.

#### O que vincula esta tarefa
- **Quem roda antes do guarda precisa do mesmo guarda.** `CorrelateRenames`
  abria anexo e placeholder somente-nuvem, furando duas regras que
  `index.Replace` respeita. É literalmente a mesma forma.
- **Teste de mecanismo que deixa o caminho normal ligado mede o caminho
  normal.** Aqui é pior: o teste chama um helper que **imita** produção.
- **`FILE_ATTRIBUTE_OFFLINE` é gravável por `SetFileAttributes` e
  `vault.IsCloudOnly` também o aceita** — é assim que se monta o caso em teste.
  `FILE_ATTRIBUTE_RECALL_ON_DATA_ACCESS` não é gravável, e o teste que tentava
  está pulado.
- **Conserto que remove uma parada abrupta abre caminho que ninguém executou**:
  foi exatamente isso que criou este defeito.

#### A evidência do defeito
`cmd/gobsidian/serve.go:323`:
```go
data, err := v.ReadAll(ctx, p)          // nao consulta IsCloudOnly
body, _ := vault.StripBOM(data)
inv.Add(string(p), search.Analyze(string(body)))
```

O comentário de `Inverted.Update` (`internal/search/inverted.go:462-467`) afirma:
*"Sao tres pontos de chamada hoje — o laco de boot em buildInvertedIndex, o
watcher em Apply e a reconciliacao por overflow"*. Verificado por grep, os
chamadores de `inv.Update` são `watcher/apply.go:58`, `watcher/apply.go:98`,
`watcher/overflow.go:69` e `cmd/gobsidian/search.go:46`. **`buildInvertedIndex`
não está na lista.** A guarda mora em `Update`; o laço de boot não passa por
`Update`.

E o teste que prova a regra tem o helper `construirComoOBoot`
(`internal/search/cloudonly_update_windows_test.go:132`), documentado como
*"tokeniza o cofre exatamente como buildInvertedIndex faz: um Update por
caminho"* — e ele chama `inv.Update`, que a produção não chama. Teste que não
pode falhar pelo caminho que ele nomeia, com a asserção certa apontada para o
código errado.

Amplificação verificada nesta sessão, e ela piora o quadro: no modo padrão
(preguiçoso), `opts.CarregarBusca` é `prepararIndiceDeBusca`
(`cmd/gobsidian/servico.go:149`), que inclui `buildInvertedIndex`. Então o laço
de boot roda **dentro da primeira `vault_search`** — o download em massa
acontece com o cliente esperando a resposta.

#### A decisão que esta tarefa tem de acertar
1. **O laço de boot chama `inv.Update`.** Não uma cópia do guarda no laço:
   guarda em chamador é a próxima divergência esperando acontecer, que é a
   lição do `aliasKey` e do `index.Classify`. Conferir o que se perde na troca —
   `Update` faz `os.ReadFile(abs)` e o laço faz `v.ReadAll(ctx, p)`, e
   `ReadAll` pode ter teto de alocação ou checagem de ctx que `Update` não tem.
   Se tiver, o certo é `Update` ganhar isso, não o laço manter o próprio ReadAll.
2. **O teste passa a exercitar `buildInvertedIndex`, e o helper morre.**
   `buildInvertedIndex` é privada de `cmd/gobsidian`; o teste vai para
   `cmd/gobsidian`, ou a função se move para um pacote testável. **Não** deixe o
   helper vivo "por enquanto" — ele é a razão de o defeito ter sobrevivido.
3. **A asserção é sobre o que o usuário veria**, não sobre a estrutura: o
   placeholder **não foi aberto**. A forma de afirmar isso sem depender de
   contador interno é conferir que o arquivo continua com
   `FILE_ATTRIBUTE_OFFLINE` depois da construção, e que `inv.HasDoc(caminho)` é
   verdadeiro com `DocLength == 0` — coberto e vazio, que é o contrato que
   `Update` documenta.

#### Armadilha específica desta tarefa
`Add(path, nil)` e não um `return` seco: a nota **tem** de contar como coberta.
Fora de `docLengths` ela não entra em `DocCount`, o cabeçalho do cache declara
menos notas do que o índice de metadados enxerga, e `invertedCacheState` conclui
"cache parcial" em **todo** boot, regravando o cache inteiro. É a mesma
armadilha que a nota sem token nenhum já custou. Se você trocar o laço por
`Update`, isso vem de graça — se você copiar o guarda, provavelmente não vem.

#### O teste que não é óbvio
```go
//go:build windows

// TestBuildInvertedIndexNaoAbrePlaceholderDeNuvem exercita a FUNCAO DE
// PRODUCAO, e essa e a diferenca em relacao ao teste que existia.
//
// O teste antigo chamava um helper que imitava o laco de boot chamando
// inv.Update — e a producao nao chamava Update. A asserção estava certa e
// apontada para o codigo errado.
func TestBuildInvertedIndexNaoAbrePlaceholderDeNuvem(t *testing.T) {
	// FILE_ATTRIBUTE_OFFLINE e gravavel por SetFileAttributes, e
	// vault.IsCloudOnly o aceita. FILE_ATTRIBUTE_RECALL_ON_DATA_ACCESS nao e
	// gravavel — o teste que tentava usa-lo esta pulado ate hoje.
	// ... monta cofre com nuvem.md marcada OFFLINE e normal.md sem marca ...

	inv := search.NewInverted()
	buildInvertedIndex(context.Background(), v, idx, inv, cfg, log)

	if !inv.HasDoc("nuvem.md") {
		t.Fatal("placeholder ficou FORA do indice: DocCount vai divergir do " +
			"indice de metadados e todo boot vai reconstruir o cache")
	}
	if n := inv.DocLength("nuvem.md"); n != 0 {
		t.Fatalf("DocLength(nuvem.md) = %d: o placeholder FOI lido e tokenizado", n)
	}
	if !vault.IsCloudOnly(filepath.Join(root, "nuvem.md")) {
		t.Fatal("o atributo OFFLINE sumiu do arquivo — ele foi hidratado")
	}
	if inv.DocLength("normal.md") == 0 {
		t.Fatal("a nota normal nao foi indexada: o teste esta medindo nada")
	}
}
```

A última asserção é o controle. Sem ela, um `buildInvertedIndex` que não faz
coisa alguma passa no teste.

#### Prova de mutação
```powershell
pwsh -File scripts/mutate.ps1 -Path internal/search/inverted.go -Anchor 'if vault.IsCloudOnly(abs) {' -Replacement 'if false {' -Test TestBuildInvertedIndexNaoAbrePlaceholderDeNuvem -Package ./cmd/gobsidian/
```
`EXIT=0` obrigatório. Essa é a prova de que o teste novo alcança o guarda pelo
caminho de produção — que é a coisa que o teste antigo não fazia.

#### Contrato de relatório
Saída de `mutate.ps1`. A confirmação de que o helper `construirComoOBoot` foi
**removido** (`git diff --stat`), e a saída do teste antigo se ele foi
convertido em vez de apagado.

---

# Task 116 — `expected_hash` contra os bytes lidos

**Tier: modelo barato.** Uma linha de código; o teste é o trabalho.

#### Onde encaixa
Terceiro crítico. O controle otimista de concorrência falha exatamente no
cenário para o qual foi escrito.

#### O que vincula esta tarefa
- **Chave derivada calculada em dois lugares diverge**, e a divergência aparece
  no caminho menos usado. Aqui são duas fontes para o mesmo valor: o disco e o
  índice.
- **Não afirme estado que você não verificou.**

#### A evidência do defeito
`write.go:166-175` e `:273-281`, os dois idênticos:
```go
raw, err := os.ReadFile(absPath)                 // bytes do disco AGORA
currentHash := fmt.Sprintf("%016x", note.Hash)   // hash do INDICE
if req.ExpectedHash != "" && currentHash != req.ExpectedHash { ... }
```

O hash comparado vem do índice de metadados, não de `raw`. Na janela do debounce
(padrão 250 ms) mais o tempo de `Replace`, uma edição externa já está no disco e
ainda não no índice: a checagem passa, e `note_append`/`note_patch` sobrescrevem
a edição do usuário.

`index.Note.Hash` está documentado como *"xxhash do conteudo BRUTO do arquivo,
com frontmatter e BOM"* (`index/note.go:100`) e é produzido por
`xxhash.Sum64(data)` em `build.go:89` e `update.go:98`. Então
`xxhash.Sum64(raw)` está a uma linha e produz **o mesmo valor** quando disco e
índice concordam — o que torna a correção segura para todo cliente que hoje usa
o campo.

#### A decisão que esta tarefa tem de acertar
Trocar a origem do `currentHash` para `xxhash.Sum64(raw)` nos dois pontos, e
**derivá-lo numa função só** — as duas cópias são idênticas hoje e vão divergir
amanhã. A função vive perto de onde `raw` é lido, e devolve a string formatada.

Segunda coisa, e é o que faz a mudança valer: a **resposta** de `note_append` e
`note_patch` devolve o hash **novo**, do conteúdo gravado. Sem isso, o cliente
que quiser encadear duas edições tem de esperar o índice atualizar — e é a
mesma janela, do outro lado. `note_read` já devolve `Hash`; conferir se ele vem
do índice também e, se vier, decidir e registrar (não mudar por reflexo:
`note_read` não escreve, e ler o disco duas vezes por leitura é custo real).

#### O teste que não é óbvio

Arquivo novo: `internal/service/expected_hash_test.go`, **`package service`**.

O ponto de partida do teste é que `ReadNote` devolve `Hash` **do índice**
(`read.go:239`, `fmt.Sprintf("%016x", note.Hash)`). É exatamente o valor que o
cliente teria em mãos, e é o que torna a condição montável sem tocar em
`index` direto.

```go
package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// TestExpectedHashPegaEdicaoExternaAindaNaoIndexada e o cenario para o qual o
// campo foi escrito, e o unico que os testes de hoje nao cobrem.
//
// O truque e NAO deixar o indice ver a edicao: escrever no disco por baixo e nao
// reindexar. Um teste que reindexa antes de chamar a escrita mede o caso facil,
// em que disco e indice concordam — e esse ja passava com o codigo defeituoso.
func TestExpectedHashPegaEdicaoExternaAindaNaoIndexada(t *testing.T) {
	casos := []struct {
		nome    string
		chamada func(svc *Service, hash string) error
	}{
		{
			nome: "note_append",
			chamada: func(svc *Service, hash string) error {
				_, err := svc.AppendNote(context.Background(), AppendNoteRequest{
					Path: "nota.md", Content: "acrescentado\n", ExpectedHash: hash,
				})
				return err
			},
		},
		{
			nome: "note_patch",
			chamada: func(svc *Service, hash string) error {
				_, err := svc.PatchNote(context.Background(), PatchNoteRequest{
					Path: "nota.md", Heading: "Secao", Mode: "replace_section",
					Content: "trocado\n", ExpectedHash: hash,
				})
				return err
			},
		},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			root := t.TempDir()
			writeFile(t, root, "nota.md", "# Secao\n\noriginal\n")
			svc := newTestService(t, root)

			// O hash que o cliente teria em maos: o do estado indexado.
			antes, err := svc.ReadNote(context.Background(), ReadRequest{Path: "nota.md"})
			if err != nil {
				t.Fatalf("ReadNote inicial: %v", err)
			}
			hashDoCliente := antes.Hash
			if hashDoCliente == "" {
				t.Fatal("ReadNote devolveu Hash vazio; o teste nao tem o que comparar")
			}

			// Edicao externa DIRETO no disco, sem reindexar. E a janela do debounce.
			const editado = "# Secao\n\neditado por fora\n"
			abs := filepath.Join(root, "nota.md")
			if err := os.WriteFile(abs, []byte(editado), 0644); err != nil {
				t.Fatalf("escrevendo por fora: %v", err)
			}

			// Confere que a CONDICAO se montou: o indice ainda tem o hash velho.
			// Sem esta guarda, um teste que reindexou por acidente passaria por
			// engano — a licao da Task 103.
			depois, err := svc.ReadNote(context.Background(), ReadRequest{Path: "nota.md"})
			if err != nil {
				t.Fatalf("ReadNote apos edicao externa: %v", err)
			}
			if depois.Hash != hashDoCliente {
				t.Fatalf("o indice ja absorveu a edicao (%s -> %s); a condicao "+
					"deste teste nao se montou", hashDoCliente, depois.Hash)
			}

			err = c.chamada(svc, hashDoCliente)
			if err == nil {
				t.Fatal("a escrita aceitou expected_hash obsoleto e sobrescreveu a " +
					"edicao externa — o controle otimista falhou no seu unico caso")
			}
			if got := CodeOf(err); got != CodeHashMismatch {
				t.Fatalf("codigo = %v, quer %v", got, CodeHashMismatch)
			}

			// A prova de que nada foi gravado. Sem ela, um erro devolvido DEPOIS
			// da escrita passaria.
			atual, err := os.ReadFile(abs)
			if err != nil {
				t.Fatalf("relendo: %v", err)
			}
			if string(atual) != editado {
				t.Fatalf("o arquivo foi alterado apesar do erro:\n%q", atual)
			}
		})
	}
}

// TestExpectedHashCorretoAindaPassa e o controle. Sem ele, uma implementacao que
// recusasse TODA escrita com expected_hash passaria no teste acima.
func TestExpectedHashCorretoAindaPassa(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "nota.md", "# Secao\n\noriginal\n")
	svc := newTestService(t, root)

	res, err := svc.ReadNote(context.Background(), ReadRequest{Path: "nota.md"})
	if err != nil {
		t.Fatalf("ReadNote: %v", err)
	}
	if _, err := svc.AppendNote(context.Background(), AppendNoteRequest{
		Path: "nota.md", Content: "acrescentado\n", ExpectedHash: res.Hash,
	}); err != nil {
		t.Fatalf("AppendNote com expected_hash CORRETO foi recusado: %v", err)
	}
}
```

#### Prova de mutação
```powershell
pwsh -File scripts/mutate.ps1 -Path internal/service/write.go -Anchor 'currentHash := hashDoConteudo(raw)' -Replacement 'currentHash := fmt.Sprintf("%016x", note.Hash)' -Test TestExpectedHashPegaEdicaoExternaAindaNaoIndexada -Package ./internal/service/
```

#### Contrato de relatório
Saída de `mutate.ps1`, e a saída do `go test -run ...` mostrando que o teste
**rodou** (não pulou). Se ele pulou, a tarefa não está pronta.

---

# Task 117 — a carga do índice de busca respeita o prazo de quem espera

**Tier: modelo principal.** É concorrência, e o modo de falha é um deadlock novo.

#### Onde encaixa
Fecha a Fase 2. Item 15 da revisão, mais o que esta sessão verificou por cima
dele — que é a parte que muda a prioridade.

#### O que vincula esta tarefa
- **Goroutine parada em `Read` não é desenrolável por cancelamento de context.**
  A carga não é um `Read`, mas o princípio se aplica: cancelar o `ctx` de quem
  **espera** não pode cancelar a carga que outro disparou.
- **`ctx` onde há espera real.** Esta é espera real, e hoje o `ctx` não é
  respeitado.
- **Não afirme estado que você não verificou**: o comentário de
  `servico.go:105` afirma um comportamento que o código não entrega.

#### A evidência do defeito
`search_lazy.go:32-42`:
```go
func (c *cargaUnica) fazer(f func() error) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	...
}
```
`c.mu.Lock()` não é cancelável. A primeira busca dispara a carga com **o ctx
dela**; as concorrentes ficam presas no mutex ignorando os próprios prazos.

O que a revisão não diz, e esta sessão verificou: `opts.CarregarBusca` é
`prepararIndiceDeBusca` (`cmd/gobsidian/servico.go:149-155`), que inclui
`buildInvertedIndex`. E em `service/search.go`, a ordem é:

```go
if err := s.garanteIndiceDeBusca(ctx); err != nil { ... }   // linha 110 — bloqueia
if s.inverted != nil && s.inverted.Building() { ... }        // linha 122 — INDEX_BUILDING
```

Então, com cache ausente, a **primeira** `vault_search` bloqueia pela
tokenização inteira do cofre — 219 s medidos num cofre de 109 MB, número que
está no comentário de `servico.go:88` — e `INDEX_BUILDING` é **inalcançável**
durante ela. Isso contradiz diretamente o comentário de `servico.go:105`:
*"Quem consulta a busca nesse intervalo recebe INDEX_BUILDING, e nao zero
resultados"*.

#### A decisão que esta tarefa tem de acertar
1. **Porta em vez de mutex.** `cargaUnica` ganha um `chan struct{}` fechado
   quando a carga termina, e quem chega durante ela faz
   `select { case <-pronto: ...; case <-ctx.Done(): return ctx.Err() }`.
   Quem **dispara** continua sendo um só, e a retentativa em caso de falha
   continua valendo — a propriedade que `sync.Once` não dá e que o comentário de
   `search_lazy.go:14-22` explica. Não a perca.
2. **O ctx da carga não é o ctx do disparador.** Hoje é, e isso já está
   documentado como escolha (`servico.go:135-140`: se aquele cliente desistir, a
   próxima busca retoma do que `HasDoc` cobriu). Com a porta, a escolha fica
   ruim: um cliente que desiste cancelaria a carga de todos os que esperam.
   A carga passa a rodar com `context.WithoutCancel` do ctx que a disparou — o
   mesmo idioma que `lifecycle.Shutdown` já usa e que o CLAUDE.md registra — ou
   com o ctx de boot, se ele estiver ao alcance. **Decida e escreva por quê.**
3. **`INDEX_BUILDING` volta a ser alcançável.** Com a porta, quem espera e
   estoura o próprio prazo recebe `INDEX_BUILDING` (não `DEADLINE_EXCEEDED`
   cru): é o código que o cliente sabe interpretar, e a mensagem diz quanto já
   foi coberto. `buildInvertedIndex` já conta `feitas` e `len(caminhos)`;
   expor esse par é o que transforma "tente de novo" em informação.

#### Armadilha específica desta tarefa
O teste óbvio — duas goroutines, uma com ctx cancelado — passa mesmo com o
código de hoje se a carga for rápida. O teste **tem** de segurar a carga por um
tempo controlado, com um `CarregadorBusca` de teste que espera um canal, e não
por `time.Sleep`. Teste de concorrência com sleep mede a máquina.

#### O teste que não é óbvio
```go
// TestBuscaConcorrenteRespeitaOProprioPrazo prova o item 15: hoje quem chega
// durante a carga fica preso no mutex ignorando o proprio deadline.
//
// O carregador de teste BLOQUEIA num canal ate o teste liberar. Sem isso, a
// carga termina rapido demais e o teste passa com o codigo defeituoso — que e
// exatamente a forma de teste que este projeto ja pagou tres vezes.
func TestBuscaConcorrenteRespeitaOProprioPrazo(t *testing.T) {
	libera := make(chan struct{})
	comecou := make(chan struct{})
	svc := service.New(v, idx, inv, nil, service.Options{
		CarregarBusca: func(ctx context.Context) error {
			close(comecou)
			<-libera
			return nil
		},
	})

	// Disparador: fica preso na carga.
	go func() { _, _ = svc.Search(context.Background(), service.SearchOptions{Query: "x"}) }()
	<-comecou

	// Segundo cliente, com prazo curto. Ele NAO pode esperar a carga inteira.
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	inicio := time.Now()
	_, err := svc.Search(ctx, service.SearchOptions{Query: "x"})
	decorrido := time.Since(inicio)

	if err == nil {
		t.Fatal("a segunda busca devolveu sucesso enquanto a carga estava presa")
	}
	if decorrido > 2*time.Second {
		t.Fatalf("a segunda busca esperou %v com prazo de 100ms: ficou presa no mutex", decorrido)
	}
	if got := codigoDe(err); got != service.CodeIndexBuilding {
		t.Fatalf("codigo = %q, quer INDEX_BUILDING — e o codigo que o cliente sabe tratar", got)
	}

	close(libera)
	// E a prova de que o cancelamento do segundo NAO matou a carga do primeiro.
	// ... espera a carga terminar e afirma que uma terceira busca funciona ...
}
```

A última parte é a metade que um teste apressado esquece, e é a que guarda a
decisão 2.

#### Prova de mutação
```powershell
pwsh -File scripts/mutate.ps1 -Path internal/service/search_lazy.go -Anchor 'case <-ctx.Done():' -Replacement 'case <-neverChan:' -Test TestBuscaConcorrenteRespeitaOProprioPrazo -Package ./internal/service/
```
(Se a mutação não compilar, ancore no `select` inteiro ou troque o corpo do
`case` — falha de compilação não é cobertura.)

#### Contrato de relatório
Saída de `mutate.ps1`. Saída de `go test -race` do pacote `service` inteiro —
esta tarefa mexe em concorrência e um `-race` limpo é parte do entregável, não
um extra. E o comentário de `servico.go:105` corrigido ou confirmado: se
`INDEX_BUILDING` passou a ser alcançável, ele volta a ser verdade; se não
passou, o comentário muda.

---

# Task 118 — `h.Slug` nos três pontos, e uma normalização em vez de três

**Tier: modelo barato.** O achado mais barato do relatório inteiro.

#### Onde encaixa
Primeira da Fase 3. Trabalho já pago, já persistido em disco, e nunca lido.

#### O que vincula esta tarefa
- **Chave derivada calculada em dois lugares diverge**, e a divergência aparece
  no caminho menos usado. Aqui são **três** implementações da mesma
  normalização, e o valor certo já está guardado no campo ao lado.
- **D-R-6**: `IndexCacheParserVersion` não sobe. Esta tarefa **não** pode mudar
  o que `Slug` produz — só quem o consome.
- **`-update` de golden grava o que o código produz, não o que está certo.**
  Se algum golden se mover nesta tarefa, algo mudou de comportamento: **pare**.

#### A evidência do defeito
`ExtractHeadings` (`parser/headings.go:60-67`) preenche `Slug: Slug(title)` em
todo heading, com `Text: title` — os dois do **mesmo** valor, então
`h.Slug == Slug(h.Text)` por construção. `persist_codec.go:186` **grava esse
slug no cache de metadados**. E nenhum leitor o usa; todos recomputam:

| ponto | o que faz | contexto |
|---|---|---|
| `index/anchors.go:31` | `parser.Slug(h.Text)` dentro do laço | por link com âncora, em `resolveAllLinks`, no boot |
| `service/read.go:172` | `parser.Slug(h.Text)` dentro do laço | por `note_read` com `heading` |
| `writer/section.go:59` | `parser.Slug(h.Text)` dentro do laço | por `note_append`/`note_patch` com `heading` |

`Slug` constrói `transform.Chain(norm.NFD, runes.Remove(...), norm.NFC)` a cada
chamada, **sem pool**. `resolveAllLinks` no boot é O(links × headings)
construções de chain — num cofre jurídico cheio de `[[nota#Art. 5]]`, é caro.

Sobra o problema de fundo, que é o que dá o segundo passo desta tarefa:

| função | pool | onde |
|---|---|---|
| `text.Normalize` | **sim** (Task 78) | `internal/text/normalize.go:27` |
| `index.normalizeString` | não | `internal/index/query.go:41` |
| `parser.Slug` | não | `internal/parser/slug.go:19` |

Verificado: `index.normalizeString` e `text.Normalize` fazem **exatamente** a
mesma coisa (`NFD → remove Mn → NFC → ToLower`); a única diferença é o pool.
`parser.Slug` compartilha o prefixo e depois remove pontuação e colapsa espaços.

#### A decisão que esta tarefa tem de acertar
Três passos, e o terceiro é opcional:

1. **Trocar `parser.Slug(h.Text)` por `h.Slug` nos três pontos.** Uma linha
   cada. Cuidado: o **argumento** de comparação (`parser.Slug(req.Heading)`,
   `parser.Slug(link.Anchor)`, `parser.Slug(headingQuery)`) continua sendo
   calculado — é entrada do cliente, não tem slug pré-computado.
2. **`index.normalizeString` passa a chamar `text.Normalize`.** Elimina a
   segunda implementação e ganha o pool de graça. Verificar antes, com um teste
   de equivalência sobre um corpus de strings acentuadas, que as duas produzem o
   mesmo — não presumir por leitura.
3. **`parser.Slug` reusar o pool** de `internal/text`: extrair a chain para uma
   função exportada de `text` que devolva a string normalizada sem `ToLower`, ou
   dar a `text` um `NormalizeKeepCase`. Se isso arrastar dependência de `parser`
   para `text` de forma que crie ciclo, **pare no passo 2** e registre.

**Nenhum dos três pode mudar o valor produzido.** É por isso que o passo 1 é
seguro: `h.Slug` foi computado por `Slug(h.Text)` na indexação, com a mesma
função. O risco real é um `Heading` construído por outro caminho com `Slug`
vazio — daí a mutação abaixo.

#### O teste que não é óbvio

Arquivo novo: `internal/index/slug_persistido_test.go`, **`package index_test`**
(é o pacote de `classify_test.go`, de onde vêm os dois primeiros helpers).

```go
package index_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/parser"
)

// construirPorCache monta o indice pelo TERCEIRO caminho de producao: gravar o
// cache de metadados e recarrega-lo. E o caminho do boot quente, e e o que ja
// escondeu uma divergencia — DocLength media 5 construido e 10 recarregado.
func construirPorCache(t *testing.T, root string) *index.Index {
	t.Helper()
	origem := construirPorBuild(t, root)

	cacheDir := t.TempDir()
	if err := index.SaveIndexCache(context.Background(), cacheDir, root, origem); err != nil {
		t.Fatalf("SaveIndexCache: %v", err)
	}
	recarregado, _, err := index.LoadIndexCache(context.Background(), cacheDir, root)
	if err != nil {
		t.Fatalf("LoadIndexCache: %v", err)
	}
	if recarregado == nil {
		t.Fatal("LoadIndexCache devolveu indice nulo")
	}
	return recarregado
}

func escreverFixtureDeHeadings(t *testing.T, root string) {
	t.Helper()
	// Acento, pontuacao, caixa, e o caso do '#' final — que ja custou caro e
	// tem tratamento proprio em parseATXHeading.
	arquivos := map[string]string{
		"a.md": "# Capítulo 118\n\ntexto\n\n## Artigo 5º — parágrafo único\n\nmais\n",
		"b.md": "## Notas sobre C#\n\ntexto\n\n### Seção: Ação & Órgão\n\nmais\n",
		"c.md": "# Titulo Simples\n\ntexto\n\n## outro em minuscula\n\nmais\n",
	}
	for nome, conteudo := range arquivos {
		if err := os.WriteFile(filepath.Join(root, nome), []byte(conteudo), 0644); err != nil {
			t.Fatalf("escrevendo %s: %v", nome, err)
		}
	}
}

// TestSlugPersistidoBateComORecomputado guarda o unico risco de trocar
// parser.Slug(h.Text) por h.Slug: um Heading que chegue ao indice com Slug vazio
// ou desatualizado.
//
// Ele varre as TRES fontes de Heading — Build, cache recarregado e Replace —
// porque o defeito so aparece na fonte que ninguem lembrou de conferir. E a
// mesma licao do DocLength construido contra recarregado.
func TestSlugPersistidoBateComORecomputado(t *testing.T) {
	root := t.TempDir()
	escreverFixtureDeHeadings(t, root)

	fontes := []struct {
		nome string
		idx  *index.Index
	}{
		{"Build", construirPorBuild(t, root)},
		{"cache recarregado", construirPorCache(t, root)},
		{"Replace", construirPorReplace(t, root)},
	}

	for _, f := range fontes {
		t.Run(f.nome, func(t *testing.T) {
			caminhos := f.idx.NotePaths()
			if len(caminhos) != 3 {
				t.Fatalf("NotePaths = %d, quer 3 — a fonte nao construiu o cofre", len(caminhos))
			}
			vistos := 0
			for _, p := range caminhos {
				n, ok := f.idx.Get(p)
				if !ok {
					t.Fatalf("NotePaths devolveu %q e Get nao resolve", p)
				}
				for _, h := range n.Headings {
					vistos++
					if quer := parser.Slug(h.Text); h.Slug != quer {
						t.Errorf("%s: heading %q tem Slug %q, recomputado da %q",
							p, h.Text, h.Slug, quer)
					}
					if h.Slug == "" {
						t.Errorf("%s: heading %q tem Slug vazio", p, h.Text)
					}
				}
			}
			// Controle: sem esta linha, uma fonte que devolvesse zero headings
			// passaria verde, que e a forma exata do teste que nao pode falhar.
			if vistos != 6 {
				t.Fatalf("%d headings conferidos, quer 6", vistos)
			}
		})
	}
}
```

E o teste de equivalência do passo 2 (consolidar `index.normalizeString` em
`text.Normalize`). Arquivo novo: `internal/index/normalizacao_equivalente_test.go`,
**`package index`** (interno — `normalizeString` não é exportada).

```go
package index

import (
	"testing"

	"github.com/jonyd/gobsidian/internal/text"
)

// TestNormalizeStringEquivaleATextNormalize e o que autoriza apagar a segunda
// implementacao. Conferir por LEITURA que as duas fazem a mesma coisa nao basta:
// a chain e igual hoje e a divergencia apareceria na proxima edicao de uma delas.
func TestNormalizeStringEquivaleATextNormalize(t *testing.T) {
	entradas := []string{
		"", "a", "A", "ÁÉÍÓÚÃÕÇ", "Prescrição Intercorrente",
		"Notas sobre C#", "Artigo 5º — parágrafo único",
		"MAIÚSCULA com Acento", "sem acento nenhum aqui",
		"Cap\u00edtulo", "Capi\u0301tulo", // NFC e NFD do mesmo texto
		"emoji \U0001F600 no meio", "  espaços  nas  bordas  ",
	}
	for _, e := range entradas {
		if got, quer := normalizeString(e), text.Normalize(e); got != quer {
			t.Errorf("normalizeString(%q) = %q, text.Normalize da %q", e, got, quer)
		}
	}
}
```

**Se este teste reprovar, pare.** As duas implementações divergem, e trocar uma
pela outra mudaria comportamento do filtro de frontmatter — o que é uma mudança
de resultado disfarçada de higiene, e vira decisão de quem revisa.

#### Prova de mutação
```powershell
pwsh -File scripts/mutate.ps1 -Path internal/parser/headings.go -Anchor 'Slug:      Slug(title),' -Replacement 'Slug:      "",' -Test TestSlugPersistidoBateComORecomputado -Package ./internal/index/
```

#### Contrato de relatório
Saída de `mutate.ps1`. `git diff --stat` mostrando que nenhum golden se moveu.
Se mediu o efeito no boot, o `benchstat`; se não mediu, **escreva "não
medido"** — D-R-8 vale aqui, e a tentação de escrever um ganho é grande.

---

# Task 119 — `ResolvePath` em O(1), e a ambiguidade de caixa detectada onde ela nasce

**Tier: modelo principal.** Semântica de índice, e a correção óbvia esconde um segundo defeito.

#### Onde encaixa
Item 5. `ResolvePath` roda em **toda** tool de leitura e de escrita.

#### O que vincula esta tarefa
- **Chave derivada calculada em dois lugares diverge.**
- **Campo de API com valor fixo mente sempre** — aqui é o irmão: um erro que
  nunca é devolvido, tratado por quatro tools.
- **Cofre inacessível e cofre vazio não podem produzir a mesma resposta.** Aqui:
  "nota não existe" e "duas notas competem pelo mesmo nome" não podem.

#### A evidência do defeito
`index/resolve.go:271-286`:
```go
lower := strings.ToLower(filepath.ToSlash(input))
var matches []vault.CanonicalPath
for lowerPath, realPath := range ix.lowerPath {
    if lowerPath == lower { matches = append(matches, realPath) }
}
if len(matches) > 1 { return "", ErrAmbiguousPath }
```

`lowerPath` é um mapa chaveado pela forma minúscula, e igualdade de chave casa
**no máximo uma** entrada. Três consequências:

1. o laço é varredura de N entradas onde `ix.lowerPath[lower]` responde em O(1);
2. `len(matches) > 1` **nunca** é verdade, e `ErrAmbiguousPath` por esse ramo é
   código morto — quatro tools tratam um erro que não chega;
3. a ambiguidade real de caixa é perdida **na escrita**, em
   `publishNameLocked:163-164` (`ix.lowerPath[lower] = path` sobrescreve). Num
   cofre Linux/macOS com `Nota.md` e `nota.md`, uma das duas fica inalcançável
   por nome, **em silêncio**.

#### A decisão que esta tarefa tem de acertar
A correção fácil (trocar o laço por `ix.lowerPath[lower]`) resolve 1 e deixa 2 e
3 exatamente como estão — e aí `ErrAmbiguousPath` vira código morto declarado em
vez de código morto acidental, que é pior porque parece intencional.

O desenho que resolve os três: **`lowerPath` passa a ser `map[string][]CanonicalPath`**.
`publishNameLocked` anexa em vez de sobrescrever; `ResolvePath` faz uma consulta
O(1) e devolve `ErrAmbiguousPath` quando a lista tem mais de um. O caminho de
remoção (`Remove`, `Replace`) tem de **tirar a entrada certa da lista**, e é aí
que esta tarefa erra se for descuidada — é literalmente a forma do defeito do
`byAlias`, que já custou uma nota deletada continuar resolvendo com `state=ok`.

Duas coisas que vêm junto e não são opcionais:

- **Toda escrita e toda leitura de `lowerPath` passa por uma função.** Não é
  para consertar os errados: é para tornar a próxima divergência impossível sem
  tocar na função. `byAlias` já tem `aliasKey`; esta ganha a irmã.
- **`vault_stats` conta as colisões.** O campo existe no espírito de
  `alias_collisions` — e `alias_collisions` era `Collisions: 0` literal, o que
  é a armadilha a não repetir. Se o número for exposto, ele é **contado**.

#### O teste que não é óbvio
```go
// TestResolvePathDetectaColisaoDeCaixa e o teste que so significa alguma coisa
// num sistema de arquivos sensivel a caixa.
//
// Ele NAO usa runtime.GOOS para pular: monta as duas entradas direto no indice,
// sem tocar no disco, e por isso roda identico nos tres sistemas. Um teste que
// pula no Windows e um teste que o desenvolvedor deste projeto nunca ve rodar.
func TestResolvePathDetectaColisaoDeCaixa(t *testing.T) {
	ix := index.NewParaTeste(t)
	ix.InserirParaTeste("Nota.md")
	ix.InserirParaTeste("nota.md")

	_, err := ix.ResolvePath("NOTA.md")
	if !errors.Is(err, index.ErrAmbiguousPath) {
		t.Fatalf("ResolvePath com duas notas competindo devolveu %v, quer ErrAmbiguousPath", err)
	}
	// O caminho exato continua resolvendo: ambiguidade e so quando ha DUVIDA.
	if p, err := ix.ResolvePath("Nota.md"); err != nil || p != "Nota.md" {
		t.Fatalf("ResolvePath exato = (%q, %v), quer (Nota.md, nil)", p, err)
	}
	// E a metade que o byAlias ja custou caro: remover uma NAO pode deixar a
	// chave apontando para a que saiu, nem apagar a que ficou.
	ix.RemoverParaTeste("Nota.md")
	p, err := ix.ResolvePath("NOTA.md")
	if err != nil {
		t.Fatalf("depois de remover uma das duas, ResolvePath falhou: %v", err)
	}
	if p != "nota.md" {
		t.Fatalf("ResolvePath = %q depois de remover Nota.md; a entrada velha sobreviveu", p)
	}
}
```

#### Prova de mutação
Duas:
```powershell
pwsh -File scripts/mutate.ps1 -Path internal/index/resolve.go -Anchor 'if len(candidatos) > 1 {' -Replacement 'if false {' -Test TestResolvePathDetectaColisaoDeCaixa -Package ./internal/index/

pwsh -File scripts/mutate.ps1 -Path internal/index/index.go -Anchor 'ix.lowerPath[lower] = append(ix.lowerPath[lower], path)' -Replacement 'ix.lowerPath[lower] = []vault.CanonicalPath{path}' -Test TestResolvePathDetectaColisaoDeCaixa -Package ./internal/index/
```

#### Contrato de relatório
As duas saídas de `mutate.ps1`. `go test -race ./internal/index/ ./internal/service/`.
Se houve efeito medido em latência de tool (a varredura O(N) sai do caminho de
toda tool), `benchstat`; senão, "não medido".

---

# Task 120 — schema honesto: data, teto de trecho, `tag_list`, `max_results`

**Tier: modelo barato.** Quatro correções da mesma família, com decisão tomada em cada uma.

#### Onde encaixa
Depois da Task 104, e **só** depois: o gate corrigido é o que impede a quinta
recorrência. Itens 7, 8 e 34.

#### O que vincula esta tarefa
- **D-R-5**: campo é implementado ou removido do schema, na mesma tarefa.
- **Schema que promete e código que ignora é pior que parâmetro ausente.** O
  modelo do outro lado pede, recebe outra coisa, e não tem como saber — o
  schema é justamente o que ele lê para decidir. Já custou caro três vezes:
  `note_list.fields`, `note_append.ensure_blank_line`, `vault_stats.include_health`.
- **Campo de API com valor fixo mente sempre.**

#### Os quatro defeitos, e a decisão de cada um

**(a) `modified_after` / `modified_before` descartados em silêncio.**
`tools_read.go:41-49`:
```go
if t, err := time.Parse(time.RFC3339, in.ModifiedAfter); err == nil { modAfter = &t }
```
`modified_after: "2026-01-01"` — a forma que qualquer modelo escreve primeiro —
não parseia, o filtro **some**, e a busca devolve o cofre inteiro como se
filtrado.
**Decisão: aceitar as duas formas e recusar o resto.** `time.RFC3339` primeiro,
`2006-01-02` em seguida (interpretado como meia-noite UTC, e isso vai no
`jsonschema`); erro de parse vira `INVALID_ARGUMENT` com as duas formas na
mensagem. Silêncio nunca.

**(b) `snippet_chars` clampado em silêncio.**
`service/search.go:138-140` corta em `search.MaxSnippetChars` (1000). O modelo
pediu 4000 e recebeu 1000, sem aviso, e `tools_read.go:269` é um `*int` **sem
descrição**.
**Decisão: declarar o teto no `jsonschema`** (`maximum`, se a versão fixada do
`jsonschema-go` suportar; descrição explícita se não) **e devolver o valor
efetivo na resposta** (`snippet_chars_effective`, ou o nome que couber). As duas
coisas, não uma: schema sem retorno deixa o cliente adivinhar se o clamp
disparou, e retorno sem schema faz ele descobrir tarde.

**(c) `tag_list.sort` declarado e nunca lido.**
`tools_read.go:328` declara; o handler (`:243-255`) passa só `Prefix`, `MinCount`
e `Hierarchical`.
**Decisão: implementar.** `index.Tags` já devolve `[]TagCount`; ordenar por
`name` (padrão) ou `count` é `slices.SortStableFunc` com desempate estável por
nome — desempate obrigatório, senão duas chamadas idênticas devolvem ordens
diferentes, que é o item 21 em miniatura.

**(d) `tag_list.hierarchical` chega ao `service` e morre lá.**
`service/graph.go:180-186` nunca consulta `req.Hierarchical` — devolve lista
plana sempre.
**Decisão: implementar.** Tag do Obsidian é hierárquica por `/`
(`projeto/ativo/2026`); a árvore agrupa por segmento e a contagem de um nó
**inclui** os filhos, com a contagem própria também exposta — senão o total não
fecha e o cliente não sabe qual dos dois números leu.

**(e) `max_results`, a cadeia morta.**
`config.MaxResults` → `service.Options.MaxResults` → **nenhum leitor**.
`config.MaxResultsCeiling = 500` não é referenciado em lugar nenhum.
**Decisão: implementar como teto de `limit`**, que é o que o nome promete: toda
tool que aceita `limit` clampa em `opts.MaxResults`, e o clamp **não é
silencioso** (mesma regra de (b)). Se o dono preferir remover a flag inteira,
é decisão dele e não desta tarefa — mas **uma das duas**, e o relatório diz
qual. `MaxResultsCeiling` é o teto do que a flag aceita, e passa a ser validado
em `config.Load`.

#### O que esta tarefa tem de rodar no fim
```
pwsh -File scripts/check_tool_params.ps1
```
Com a Task 104 no lugar, ele tem de sair `0` **depois** desta tarefa e ter saído
`1` **antes**. Cole os dois.

#### O teste que não é óbvio

São quatro campos e quatro testes. **Nenhum pode ficar de fora** — a armadilha
desta tarefa é implementar três e esquecer o quarto.

#### (a) Filtro de data — `internal/mcpsrv/filtro_data_test.go`, `package mcpsrv_test`

O parse mora em `mcpsrv`, mas a asserção que pega o defeito é a **contagem**, e
`newTestServerWithConfig` monta o índice invertido **vazio**. Por isso este
arquivo traz o próprio helper, que popula a busca.

```go
package mcpsrv_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jonyd/gobsidian/internal/config"
	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/mcpsrv"
	"github.com/jonyd/gobsidian/internal/search"
	"github.com/jonyd/gobsidian/internal/service"
	"github.com/jonyd/gobsidian/internal/vault"
)

// sessaoComBusca e novaSessao COM o indice invertido populado.
//
// newTestServerWithConfig monta search.NewInverted() VAZIO, entao vault_search
// por aquela sessao devolve zero hits sempre — e um teste que contasse hits ali
// estaria medindo o vazio, nao o filtro.
func sessaoComBusca(t *testing.T, root string) (*mcp.ClientSession, context.Context) {
	t.Helper()
	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	idx := index.New()
	if err := idx.Build(context.Background(), v); err != nil {
		t.Fatalf("idx.Build: %v", err)
	}
	inv := search.NewInverted()
	for _, p := range idx.NotePaths() {
		data, err := v.ReadAll(context.Background(), p)
		if err != nil {
			t.Fatalf("ReadAll %s: %v", p, err)
		}
		body, _ := vault.StripBOM(data)
		inv.Add(string(p), search.Analyze(string(body)))
	}
	svc := service.New(v, idx, inv, nil, service.Options{})
	srv := mcpsrv.New(context.Background(), svc, config.Defaults(),
		slog.New(slog.NewTextHandler(io.Discard, nil)))

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	t.Cleanup(cancel)
	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	go func() { _ = srv.Connect(ctx, serverTransport) }()
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "0.0.1"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })
	return session, ctx
}

// TestFiltroDeDataInvalidoNaoViraSilencio guarda o item 8: a data que nao
// parseia fazia o filtro SUMIR, e a busca devolvia o cofre inteiro como se
// filtrado.
//
// A asserção que pega isso nao e "devolveu erro" — e a CONTAGEM. Um filtro que
// vira no-op devolve o mesmo numero de hits que nenhum filtro, e e assim que o
// defeito se esconde de quem le a resposta.
func TestFiltroDeDataInvalidoNaoViraSilencio(t *testing.T) {
	root := t.TempDir()
	for _, nome := range []string{"a.md", "b.md", "c.md"} {
		if err := os.WriteFile(filepath.Join(root, nome),
			[]byte("# Nota\n\npalavra comum aqui\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	session, ctx := sessaoComBusca(t, root)

	// buscar devolve (numeroDeHits, houveErro). -1 hits significa erro.
	buscar := func(t *testing.T, extra map[string]any) (int, bool) {
		t.Helper()
		args := map[string]any{"query": "comum"}
		for k, v := range extra {
			args[k] = v
		}
		res, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name: "vault_search", Arguments: args,
		})
		if err != nil {
			t.Fatalf("CallTool: %v", err)
		}
		if res.IsError {
			return -1, true
		}
		var out struct {
			Hits []struct {
				Path string `json:"path"`
			} `json:"hits"`
		}
		bruto, err := json.Marshal(res.StructuredContent)
		if err != nil {
			t.Fatalf("resposta nao serializa: %v", err)
		}
		if err := json.Unmarshal(bruto, &out); err != nil {
			t.Fatalf("resposta ilegivel: %v\n%s", err, bruto)
		}
		return len(out.Hits), false
	}

	// Controle 1: sem filtro nenhum, tres notas. Se isto falhar, a fixture nao
	// monta o teste e o resto nao significa nada.
	if n, _ := buscar(t, nil); n != 3 {
		t.Fatalf("sem filtro: hits = %d, quer 3", n)
	}

	// Controle 2: o filtro FUNCIONA. Data no futuro exclui todas.
	if n, _ := buscar(t, map[string]any{"modified_after": "2100-01-01T00:00:00Z"}); n != 0 {
		t.Fatalf("modified_after no futuro: hits = %d, quer 0 — o filtro nao e aplicado", n)
	}

	// As duas formas que TEM de ser aceitas, as duas permissivas.
	for _, forma := range []string{"2000-01-01", "2000-01-01T00:00:00Z"} {
		t.Run("aceita "+forma, func(t *testing.T) {
			n, erro := buscar(t, map[string]any{"modified_after": forma})
			if erro {
				t.Fatalf("forma valida %q foi rejeitada", forma)
			}
			if n != 3 {
				t.Fatalf("hits = %d, quer 3", n)
			}
		})
	}

	// A forma que NAO pode passar em silencio. Sem a correcao, ela devolve 3 —
	// o cofre inteiro, com aparencia de filtrado.
	t.Run("recusa data invalida", func(t *testing.T) {
		n, erro := buscar(t, map[string]any{"modified_after": "ontem"})
		if !erro {
			t.Fatalf("modified_after=%q foi aceito e devolveu %d hits: o filtro "+
				"sumiu e a busca respondeu como se filtrada", "ontem", n)
		}
	})

	// E o par, para modified_before nao ficar de fora.
	t.Run("recusa modified_before invalido", func(t *testing.T) {
		if n, erro := buscar(t, map[string]any{"modified_before": "semana passada"}); !erro {
			t.Fatalf("modified_before invalido foi aceito e devolveu %d hits", n)
		}
	})
}
```

#### (b) e (c) `tag_list.sort` e `tag_list.hierarchical` — `internal/service/tag_list_test.go`, `package service`

```go
package service

import (
	"context"
	"testing"
)

func cofreDeTags(t *testing.T) *Service {
	t.Helper()
	root := t.TempDir()
	// projeto/ativo aparece 3x, projeto/ativo/2026 2x, zzz 1x. As contagens sao
	// diferentes entre si DE PROPOSITO: ordenar por nome e por contagem tem de
	// produzir ordens distintas, senao o teste passa com qualquer implementacao.
	writeFile(t, root, "a.md", "---\ntags: [projeto/ativo, projeto/ativo/2026, zzz]\n---\n\ntexto\n")
	writeFile(t, root, "b.md", "---\ntags: [projeto/ativo, projeto/ativo/2026]\n---\n\ntexto\n")
	writeFile(t, root, "c.md", "---\ntags: [projeto/ativo]\n---\n\ntexto\n")
	return newTestService(t, root)
}

// TestTagListOrdenacao prova que o campo `sort` deixou de ser decorativo.
//
// As duas ordens tem de ser DIFERENTES entre si — se a fixture produzisse a
// mesma ordem nos dois modos, o teste passaria com o handler ignorando o campo,
// que e exatamente o estado de hoje.
func TestTagListOrdenacao(t *testing.T) {
	svc := cofreDeTags(t)

	porNome, err := svc.TagList(context.Background(), TagRequest{Sort: "name", MinCount: 1})
	if err != nil {
		t.Fatalf("TagList(name): %v", err)
	}
	porContagem, err := svc.TagList(context.Background(), TagRequest{Sort: "count", MinCount: 1})
	if err != nil {
		t.Fatalf("TagList(count): %v", err)
	}
	if len(porNome.Tags) == 0 || len(porNome.Tags) != len(porContagem.Tags) {
		t.Fatalf("listas de tamanhos diferentes: %d e %d", len(porNome.Tags), len(porContagem.Tags))
	}

	// Por nome: crescente.
	for i := 1; i < len(porNome.Tags); i++ {
		if porNome.Tags[i-1].Tag > porNome.Tags[i].Tag {
			t.Fatalf("sort=name fora de ordem em %d: %q depois de %q",
				i, porNome.Tags[i].Tag, porNome.Tags[i-1].Tag)
		}
	}
	// Por contagem: decrescente, com desempate ESTAVEL por nome. Sem o
	// desempate, duas chamadas identicas devolvem ordens diferentes — o item 21
	// em miniatura.
	for i := 1; i < len(porContagem.Tags); i++ {
		a, b := porContagem.Tags[i-1], porContagem.Tags[i]
		if a.Count < b.Count {
			t.Fatalf("sort=count fora de ordem em %d: %d depois de %d", i, b.Count, a.Count)
		}
		if a.Count == b.Count && a.Tag > b.Tag {
			t.Fatalf("empate em %d nao desempatou por nome: %q depois de %q", i, b.Tag, a.Tag)
		}
	}

	// E a asserção que prova que `sort` MUDOU alguma coisa.
	if porNome.Tags[0].Tag == porContagem.Tags[0].Tag &&
		porNome.Tags[len(porNome.Tags)-1].Tag == porContagem.Tags[len(porContagem.Tags)-1].Tag {
		t.Fatal("sort=name e sort=count devolveram a mesma ordem; o campo foi ignorado")
	}
}

// TestTagListHierarquico prova o segundo campo morto. A asserção e sobre a
// FORMA da resposta: hierarchical=true nao pode devolver a mesma lista plana.
func TestTagListHierarquico(t *testing.T) {
	svc := cofreDeTags(t)

	plana, err := svc.TagList(context.Background(), TagRequest{MinCount: 1})
	if err != nil {
		t.Fatalf("TagList plana: %v", err)
	}
	arvore, err := svc.TagList(context.Background(), TagRequest{MinCount: 1, Hierarchical: true})
	if err != nil {
		t.Fatalf("TagList hierarquica: %v", err)
	}

	// A raiz "projeto" nao existe como tag literal em nota nenhuma: ela so
	// aparece se a arvore foi de fato montada.
	temRaizProjeto := false
	for _, n := range arvore.Tags {
		if n.Tag == "projeto" {
			temRaizProjeto = true
			// A contagem do no INCLUI os filhos: 3 notas usam projeto/*.
			if n.Count != 3 {
				t.Errorf("no raiz projeto tem Count=%d, quer 3 (incluindo filhos)", n.Count)
			}
		}
	}
	if !temRaizProjeto {
		t.Fatal("hierarchical=true devolveu lista plana: nao ha no raiz 'projeto'")
	}
	if len(arvore.Tags) == len(plana.Tags) {
		t.Fatal("hierarchical=true devolveu o mesmo numero de entradas que a lista plana")
	}
}
```

**Decisão de forma que a tarefa tem de tomar e registrar:** `TagResult.Tags` é
`[]index.TagCount`, que não tem filhos. A árvore precisa de um tipo novo
(`TagNode` com `Children`) ou de um campo paralelo (`Tree`). Escolha **um**,
escreva no `docs/TOOLS.md`, e não devolva os dois — duas representações do mesmo
dado divergem. O teste acima assume um nó raiz na mesma lista; se você escolher
`Tree` separado, **ajuste o teste e diga isso no relatório**.

#### (d) `max_results` — `internal/service/max_results_test.go`, `package service_test`

```go
package service_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/jonyd/gobsidian/internal/service"
)

// TestMaxResultsClampaLimit prova que a cadeia config -> Options -> uso deixou
// de morrer no meio. Hoje Options.MaxResults nao tem leitor nenhum.
//
// createSearchService nao aceita Options, entao o Service e montado aqui — o que
// tambem deixa explicito QUAL opcao esta sob teste.
func TestMaxResultsClampaLimit(t *testing.T) {
	arquivos := map[string]string{}
	for i := 0; i < 30; i++ {
		arquivos[fmt.Sprintf("n%02d.md", i)] = "palavra comum nesta nota\n"
	}
	_, v, idx, inv := createSearchService(t, arquivos)

	svc := service.New(v, idx, inv, nil, service.Options{MaxResults: 5})

	res, err := svc.Search(context.Background(), service.SearchOptions{
		Query: "comum",
		Limit: 100, // acima do teto
	})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(res.Hits) != 5 {
		t.Fatalf("hits = %d, quer 5 (limit=100 clampado por MaxResults=5)", len(res.Hits))
	}

	// Controle: sem teto configurado, o limite pedido vale.
	semTeto := service.New(v, idx, inv, nil, service.Options{})
	res2, err := semTeto.Search(context.Background(), service.SearchOptions{
		Query: "comum", Limit: 100,
	})
	if err != nil {
		t.Fatalf("Search sem teto: %v", err)
	}
	if len(res2.Hits) != 30 {
		t.Fatalf("sem teto: hits = %d, quer 30", len(res2.Hits))
	}

	// E o clamp nao pode ser silencioso (mesma regra de snippet_chars).
	if res.LimitEfetivo != 5 {
		t.Fatalf("LimitEfetivo = %d, quer 5 — o clamp nao foi anunciado na resposta",
			res.LimitEfetivo)
	}
}

// TestSnippetCharsClampAnunciado e o item 34: o modelo pediu 4000, recebeu 1000,
// e nao teve como saber.
func TestSnippetCharsClampAnunciado(t *testing.T) {
	svc, _, _, _ := createSearchService(t, map[string]string{
		"a.md": "palavra comum e muito texto depois dela para o trecho ter de onde sair\n",
	})
	res, err := svc.Search(context.Background(), service.SearchOptions{
		Query: "comum", Limit: 5, SnippetChars: 4000,
	})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if res.SnippetCharsEfetivo == 0 {
		t.Fatal("a resposta nao diz qual snippet_chars foi usado")
	}
	if res.SnippetCharsEfetivo >= 4000 {
		t.Fatalf("SnippetCharsEfetivo = %d; o teto de MaxSnippetChars nao foi aplicado",
			res.SnippetCharsEfetivo)
	}
}
```

**Os nomes `LimitEfetivo` e `SnippetCharsEfetivo` são propostas, não contrato.**
Escolha os nomes JSON no `docs/TOOLS.md` primeiro, ajuste o teste, e mantenha os
dois com a mesma convenção — dois campos da mesma família com nomes de estilos
diferentes é a próxima divergência.

#### Prova de mutação
Uma por campo. As quatro obrigatórias, porque a armadilha desta tarefa é
implementar três e esquecer a quarta:
```powershell
pwsh -File scripts/mutate.ps1 -Path internal/mcpsrv/tools_read.go -Anchor 'return noteReadValidationError(fmt.Sprintf("modified_after invalido' -Replacement 'modAfter = nil; _ = fmt.Sprintf("' -Test TestFiltroDeDataInvalidoNaoViraSilencio -Package ./internal/mcpsrv/
pwsh -File scripts/mutate.ps1 -Path internal/service/graph.go -Anchor 'if req.Hierarchical {' -Replacement 'if false {' -Test TestTagListHierarquico -Package ./internal/service/
pwsh -File scripts/mutate.ps1 -Path internal/service/graph.go -Anchor 'ordenarTags(tags, req.Sort)' -Replacement '_ = req.Sort' -Test TestTagListOrdenacao -Package ./internal/service/
pwsh -File scripts/mutate.ps1 -Path internal/service/search.go -Anchor 'if s.opts.MaxResults > 0 && opts.Limit > s.opts.MaxResults {' -Replacement 'if false {' -Test TestMaxResultsClampaLimit -Package ./internal/service/
```

#### Contrato de relatório
As quatro saídas de `mutate.ps1`. `check_tool_params.ps1` antes e depois. O
`docs/TOOLS.md` atualizado nos quatro campos.

---

# Task 121 — `FrontmatterErr` ligado, `Parse` que não mente, `Build` que conta o que descarta

**Tier: modelo barato.** Três defeitos da mesma família: informação produzida e jogada fora.

#### Onde encaixa
Itens 14 e 18. Uma nota some do índice, de todas as tools, e nada registra.

#### O que vincula esta tarefa
- **`SkippedEntries` foi criado exatamente para isto**, e `vault.recordSkip`
  existe e não é usado em `index.Build`.
- **Conserto que remove uma parada abrupta abre caminho que ninguém executou.**
  Fazer `parser.Parse` deixar de mentir na assinatura torna alcançável o ramo de
  `build.go:78` — que hoje é código morto. Percorra os chamadores a jusante
  antes de mexer.
- **Não deixe sua deliberação no código.**

#### A evidência do defeito
`parser.Parse:31` preenche `note.FrontmatterErr` (`parser/types.go:136`). Grep no
repositório inteiro: **nenhum leitor**. `index.Note` não tem o campo,
`note_metadata` não o expõe, `vault_stats` não o conta. Uma nota com YAML
malformado perde tags, aliases e título **silenciosamente**.

E `parser.Parse` devolve `(*ParsedNote, error)` e **nunca** devolve erro
não-nil — os únicos `return` da função são `return note, nil` (`parser.go:51`).
`build.go:78` tem um `if err != nil { continue }` inalcançável, e
`write.go:178,284` fazem `parsed, _ :=`. A assinatura mente.

`index.Build` (`build.go:71-81`) tem `continue` seco em erro de leitura **e** de
parse. A nota some do índice e nada registra.

#### A decisão que esta tarefa tem de acertar
1. **`index.Note` ganha `FrontmatterErr string`**, propagado em `insert`
   (`index.go:120-135`) **e** em `Replace` (`update.go:95-108`) — os dois, ou é o
   item 16 outra vez, com um campo derivado novo que não chega ao caminho menos
   usado. `persist_codec` grava e lê o campo; isso **é** mudança de formato do
   cache de metadados: confira qual constante de versão governa esse cache e
   suba-a, se for a de metadados. D-R-6 proíbe subir a **do parser**; a de
   metadados é outra coisa e é decisão desta tarefa — **escreva qual você
   subiu e por quê**.
2. **`note_metadata` expõe `frontmatter_err`** (omitempty). **`vault_stats`
   conta** quantas notas têm o campo preenchido. Contar não é opcional: sem o
   contador, o usuário só descobre nota a nota.
3. **`parser.Parse` perde o `error` do retorno**, e os três chamadores se
   ajustam. Antes de fazer: rode `grep -rn "parser.Parse(" --include=*.go` e
   confira que **todos** os chamadores tratam o retorno novo. O ramo morto de
   `build.go:78` sai junto.
4. **`index.Build` registra o descarte** por `vault.recordSkip`, com o motivo, e
   o número aparece em `vault_stats` como os outros descartes — **desdobrado por
   motivo**, que é a decisão já fechada no M2.1 e vale igual aqui.

#### O teste que não é óbvio

Arquivo novo: `internal/service/frontmatter_err_test.go`, **`package service`**.

```go
package service

import (
	"context"
	"strings"
	"testing"
)

// TestNotaComFrontmatterQuebradoNaoSomeEmSilencio cobre as tres pontas de uma
// vez: o campo chega ao indice, note_metadata o expoe, e vault_stats o CONTA.
//
// A asserção sobre o contador e a que importa: sem ela, uma nota quebrada em mil
// e indistinguivel de zero notas quebradas, que e o estado de hoje.
func TestNotaComFrontmatterQuebradoNaoSomeEmSilencio(t *testing.T) {
	// YAML invalido de verdade: lista aberta que a linha seguinte nao fecha.
	const quebrada = "---\ntags: [a, b\ntitulo: sem fechar\n---\n\n# Corpo\n\ntexto util\n"
	const boa = "---\ntags: [x]\n---\n\n# Boa\n\ntexto\n"

	root := t.TempDir()
	writeFile(t, root, "quebrada.md", quebrada)
	writeFile(t, root, "boa.md", boa)
	svc := newTestService(t, root)

	md, err := svc.NoteMetadata(context.Background(), MetadataRequest{Path: "quebrada.md"})
	if err != nil {
		t.Fatalf("NoteMetadata: %v", err)
	}
	if md.FrontmatterErr == "" {
		t.Fatal("frontmatter malformado nao chegou a note_metadata: a nota perdeu " +
			"tags, aliases e titulo em silencio")
	}

	// O corpo continua util — frontmatter quebrado NAO invalida a nota, e o
	// comentario de parser.Parse:29 diz exatamente isso.
	if len(md.Headings) == 0 {
		t.Fatal("os headings do corpo sumiram junto com o frontmatter")
	}

	// A nota boa nao pode ganhar o campo.
	boaMD, err := svc.NoteMetadata(context.Background(), MetadataRequest{Path: "boa.md"})
	if err != nil {
		t.Fatalf("NoteMetadata(boa): %v", err)
	}
	if boaMD.FrontmatterErr != "" {
		t.Fatalf("nota com frontmatter valido ganhou FrontmatterErr=%q", boaMD.FrontmatterErr)
	}

	st, err := svc.VaultStats(context.Background(), StatsRequest{IncludeHealth: true})
	if err != nil {
		t.Fatalf("VaultStats: %v", err)
	}
	if st.FrontmatterErrors == nil {
		t.Fatal("vault_stats nao reporta o contador de frontmatter quebrado")
	}
	if *st.FrontmatterErrors != 1 {
		t.Fatalf("vault_stats conta %d notas com frontmatter quebrado, quer 1",
			*st.FrontmatterErrors)
	}
}

// TestContadorDeFrontmatterZeraQuandoNaoHaQuebrada e o controle: sem ele, um
// contador que devolvesse 1 fixo passaria no teste acima.
func TestContadorDeFrontmatterZeraQuandoNaoHaQuebrada(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "boa.md", "---\ntags: [x]\n---\n\n# Boa\n\ntexto\n")
	svc := newTestService(t, root)

	st, err := svc.VaultStats(context.Background(), StatsRequest{IncludeHealth: true})
	if err != nil {
		t.Fatalf("VaultStats: %v", err)
	}
	if st.FrontmatterErrors == nil {
		t.Fatal("com include_health o contador tem de existir, mesmo sendo zero")
	}
	if *st.FrontmatterErrors != 0 {
		t.Fatalf("contador = %d num cofre sem nota quebrada", *st.FrontmatterErrors)
	}
}

// TestContadorDeFrontmatterAusenteSemIncludeHealth e a outra metade da mesma
// regra, e ela ja esta escrita em StatsResult: nil = nao pedido, &0 = pedido e
// nao ha nenhum. Um zero que significa as duas coisas nao informa nada.
func TestContadorDeFrontmatterAusenteSemIncludeHealth(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "boa.md", "---\ntags: [x]\n---\n\n# Boa\n\ntexto\n")
	svc := newTestService(t, root)

	st, err := svc.VaultStats(context.Background(), StatsRequest{})
	if err != nil {
		t.Fatalf("VaultStats: %v", err)
	}
	if st.FrontmatterErrors != nil {
		t.Fatalf("sem include_health o contador veio preenchido (%d): o cliente "+
			"nao distingue 'nao ha' de 'nao perguntei'", *st.FrontmatterErrors)
	}
}
```

**`FrontmatterErrors` é `*int`, e não `int`**, pela regra que `StatsResult` já
documenta nos campos `Orphans`, `BrokenLinks` e `BrokenAnchor`: `nil` = não
pedido, `&0` = pedido e não há nenhum. Não repita `Collisions`, que é `int` e
era `0` literal.

E o teste do descarte silencioso de `index.Build`, arquivo novo
`internal/index/build_descarte_test.go`, **`package index_test`**:

```go
package index_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jonyd/gobsidian/internal/vault"
)

// TestBuildRegistraArquivoIlegivel prova que a nota que some do indice deixa
// rastro. Hoje build.go:73 faz `continue` seco: a nota desaparece de todas as
// tools e NADA registra.
//
// Montar um arquivo ilegivel de forma portavel e o problema deste teste. A
// forma que funciona nos tres sistemas e um DIRETORIO com nome .md: ReadAll
// falha, e o caminho de erro e o mesmo.
func TestBuildRegistraArquivoIlegivel(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "boa.md"), []byte("# Boa\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "ilegivel.md"), 0755); err != nil {
		t.Fatal(err)
	}

	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	idx := construirPorBuild(t, root)

	if idx.NoteCount() != 1 {
		t.Fatalf("NoteCount = %d, quer 1 (so a boa)", idx.NoteCount())
	}
	pulados := v.SkippedEntries()
	if len(pulados) == 0 {
		t.Fatal("o arquivo ilegivel sumiu do indice sem deixar registro em " +
			"SkippedEntries — cofre com nota perdida responde igual a cofre limpo")
	}
}
```

**Confira a API antes de escrever:** o nome exato do acessor de descartes em
`vault` (`SkippedEntries` ou equivalente) e sua assinatura. Se `vault.Walk` já
pula diretório antes de chegar em `Build`, monte a condição de outro jeito e
**diga qual** no relatório — o que não vale é entregar um teste que passa porque
a condição nunca se montou.

#### Prova de mutação
```powershell
pwsh -File scripts/mutate.ps1 -Path internal/index/update.go -Anchor 'FrontmatterErr: note.FrontmatterErr,' -Replacement 'FrontmatterErr: "",' -Test TestNotaComFrontmatterQuebradoNaoSomeEmSilencio -Package ./internal/service/
```
E, se o cache de metadados guardar o campo, a segunda prova é a paridade
construído-contra-recarregado, que já tem teste próprio no projeto — rode-o e
cole.

#### Contrato de relatório
Saída de `mutate.ps1`. Qual versão de cache subiu e por quê (ou por que nenhuma
precisou). A resposta de `vault_stats` com o contador novo.

---

# Task 122 — `link_graph` determinístico, com teto

**Tier: modelo barato.** Item 21, e é ordenação.

#### Onde encaixa
Fase 3. Todo o resto do projeto ordena deterministicamente; esta tool não.

#### O que vincula esta tarefa
- **Duas chamadas idênticas têm de devolver a mesma coisa na mesma ordem** —
  `Postings`, `Paths`, `List` e `affectedKeys` já o fazem, e o desempate estável
  por caminho é o padrão do projeto.
- **Teto declarado, nunca silencioso** (a lição do item 34).

#### A evidência do defeito
`service/graph.go:155-160`:
```go
for _, v := range nodesMap { res.Nodes = append(res.Nodes, v) }
for _, e := range edgesMap { res.Edges = append(res.Edges, e) }
```
Iteração de mapa, sem ordenar. E `limit` não tem teto — `depth` tem, 3.

#### A decisão que esta tarefa tem de acertar
Ordenar `Nodes` por `path` e `Edges` por `(source, target, kind)` — a tripla,
não só a origem, senão duas arestas entre os mesmos nós continuam podendo
trocar de lugar. `slices.SortFunc`, não `sort.Slice` (a Task 123 faria a troca
de qualquer jeito).

Teto de `limit`: usar o mesmo `MaxResults` que a Task 120 implementa, ou uma
constante própria declarada no schema. **Uma das duas, e declarada** — e quando
o teto corta, o retorno diz que cortou.

#### O teste que não é óbvio

Arquivo novo: `internal/service/graph_ordem_test.go`, **`package service`**.

```go
package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"testing"
)

// cofreInterligado monta 12 notas com ligacoes cruzadas. O numero importa: com
// duas ou tres, a iteracao de mapa do Go coincide com frequencia alta e o teste
// passa com o codigo desordenado.
func cofreInterligado(t *testing.T) *Service {
	t.Helper()
	root := t.TempDir()
	for i := 0; i < 12; i++ {
		var corpo bytes.Buffer
		fmt.Fprintf(&corpo, "# Nota %02d\n\n", i)
		for j := 0; j < 12; j++ {
			if j != i {
				fmt.Fprintf(&corpo, "liga para [[n%02d]]\n", j)
			}
		}
		writeFile(t, root, fmt.Sprintf("n%02d.md", i), corpo.String())
	}
	return newTestService(t, root)
}

// TestLinkGraphOrdemEstavel roda a MESMA consulta 50 vezes e exige resposta byte
// a byte identica.
//
// Uma chamada so nao pega ordem de mapa. 50 voltas e o que torna a falha
// confiavel; com menos, o teste passa as vezes — que e a pior especie.
func TestLinkGraphOrdemEstavel(t *testing.T) {
	svc := cofreInterligado(t)

	consulta := GraphRequest{Path: "n00.md", Direction: "both", Depth: 2, Limit: 100}

	serializa := func(t *testing.T) []byte {
		t.Helper()
		res, err := svc.LinkGraph(context.Background(), consulta)
		if err != nil {
			t.Fatalf("LinkGraph: %v", err)
		}
		if len(res.Nodes) < 5 || len(res.Edges) < 5 {
			t.Fatalf("grafo pequeno demais para o teste significar algo: %d nos, %d arestas",
				len(res.Nodes), len(res.Edges))
		}
		bruto, err := json.Marshal(res)
		if err != nil {
			t.Fatalf("Marshal: %v", err)
		}
		return bruto
	}

	primeira := serializa(t)
	for i := 1; i <= 50; i++ {
		outra := serializa(t)
		if !bytes.Equal(primeira, outra) {
			t.Fatalf("volta %d devolveu ordem diferente:\n%s\n%s", i, primeira, outra)
		}
	}
}

// TestLinkGraphLimitTemTeto guarda a segunda metade do item 21: `depth` tem teto
// (3) e `limit` nao tem nenhum.
func TestLinkGraphLimitTemTeto(t *testing.T) {
	svc := cofreInterligado(t)

	res, err := svc.LinkGraph(context.Background(), GraphRequest{
		Path: "n00.md", Direction: "both", Depth: 3, Limit: 1000000,
	})
	if err != nil {
		t.Fatalf("LinkGraph: %v", err)
	}
	// O cofre tem 12 notas, entao o teto nao pode ser medido pelo tamanho do
	// resultado. A asserção e sobre o VALOR EFETIVO anunciado na resposta.
	if res.LimitEfetivo == 0 {
		t.Fatal("a resposta nao diz qual limit foi usado")
	}
	if res.LimitEfetivo >= 1000000 {
		t.Fatalf("LimitEfetivo = %d: limit continua sem teto", res.LimitEfetivo)
	}
}
```

Como na Task 120, **`LimitEfetivo` é proposta, não contrato**: escolha o nome
JSON no `docs/TOOLS.md` primeiro e use a mesma convenção das outras tools que
anunciam clamp. Se a Task 120 já tiver fixado uma convenção, siga a dela.

**Ordenação:** `Nodes` por `path`; `Edges` pela tripla `(source, target, kind)`.
A tripla, não só a origem — duas arestas entre os mesmos nós com `kind`
diferente continuariam trocando de lugar. `slices.SortFunc`, não `sort.Slice`
(a Task 123 faria a troca de qualquer jeito).

#### Prova de mutação
```powershell
pwsh -File scripts/mutate.ps1 -Path internal/service/graph.go -Anchor 'slices.SortFunc(res.Nodes,' -Replacement 'noop(res.Nodes,' -Test TestLinkGraphOrdemEstavel -Package ./internal/service/
```
Se `noop` não existir, ancore na função de comparação e faça-a devolver `0`
sempre — uma comparação constante desordena sem quebrar o build.

#### Contrato de relatório
Saída de `mutate.ps1`. E a saída do teste rodado com `-count=5`, porque um teste
de estabilidade que roda uma vez não é teste de estabilidade.

---

# Task 123 — higiene: deliberação commitada, `go.mod`, e o que o `go fix` mecaniza

**Tier: modelo barato.** Diff auditável, sem decisão de projeto.

#### Onde encaixa
Última da Fase 3. Junta os itens de higiene do relatório num commit legível.

#### O que vincula esta tarefa
- **Não deixe sua deliberação no código.** Comentário explica por que o código é
  assim; raciocínio sobre o que fazer não é comentário.
- **Nunca rode `go mod tidy`.** Várias deps estão fixadas sem importador ainda;
  `tidy` removeria o pin do SDK MCP, que é decisão fechada (PRD D6). As linhas
  `// indirect` se corrigem **à mão**.
- **`gofmt` reprova `.go` que estava perfeito** se um script de edição converter
  o arquivo para CRLF. Use `Edit`, não script.

#### O que entra, item a item

**(a) Deliberação commitada, quatro pontos.**

Os dois que a revisão lista, mais dois que ela não viu porque não olhou
`_test.go` nem comentários de histórico:

`internal/service/read_test.go:30,35` — dois comentários que são pergunta, não
explicação:
```go
// How to build an index for tests? We should probably just use index.Build
// I need to use the interface Index if we didn't add the methods to Index yet.
// We'll update the Index interface in service.go.
```
Este é o helper que **todas** as tarefas deste lote usam. Deliberação em inglês,
no primeiro arquivo que um executor novo abre.

`internal/service/graph.go:401` — `// VaultStats was relocated from service.go`.
Comentário de histórico: o `git log` já sabe disso, e o comentário passa a mentir
no dia em que a função se mover de novo.

Os dois da revisão:
`service/read.go:244` — `// better condition`. Já sai na Task 106; se sobrou,
sai aqui.
`index/note.go:142`:
```go
_ = text.Normalize("") // Garante que text é usado; mutação deve remover text.Normalize
```
Andaime de teste de mutação rodando **em produção, uma vez por nota indexada**.
O objetivo é legítimo e o lugar não é: se a preocupação é a mutação quebrar a
compilação em vez de o teste, isso pertence à escolha da âncora no `mutate.ps1`,
não ao caminho quente. Remova a linha e **registre no plano** qual âncora
substitui a garantia — não deixe a preocupação sem resposta, senão a próxima
prova de mutação sobre `normalizeTitleForNote` sai `EXIT=2`.

**(b) `go.mod` marca todas as dependências como `// indirect`**, incluindo
`cobra`, `goldmark`, `go-sdk` e `fsnotify`, que são importadas diretamente.
Consequência do `tidy` proibido. Cosmético, mas engana qualquer leitura do
arquivo. Corrigir à mão, uma vez, e conferir com `go build ./...` e
`go list -m all` — **sem** rodar `tidy`.

**(c) `golang.org/x/tools` é a única dependência direta declarada**, e serve só
ao `tools/netcheck`. Ela entra no grafo de quem fizer `go install` do produto.
Mover `tools/` para módulo próprio (`tools/go.mod`) tira `x/tools` do módulo
principal. **Conferir antes** o que isso quebra: `scripts/check_net.ps1` chama o
analisador, e o CI o roda — se o caminho de invocação mudar, os dois mudam
junto, e essa é a parte que pode não caber na tarefa. Se não couber, **entregue
(a) e (b) e diga que (c) ficou de fora**.

**(d) `interface{}` → `any`** em `text/normalize.go:19` e
`mcpsrv/tools_read.go:266,302`.

**(e) `sort.Slice` / `sort.Strings` → `slices.SortFunc` / `slices.Sort`** nos 12
lugares. `service/write.go:528` e `search/bm25.go:158` estão entre eles.

**(f) `index/query.go:377`**: `cmp = int(a.Size - b.Size)` trunca `int64` para
`int` — inofensivo em 64 bits, quebra em 32. `cmp.Compare(a.Size, b.Size)` é o
idioma de Go 1.21+.

**(g) `index/query.go:372`**: `switch strings.ToLower(q.Sort)` está **dentro** do
comparador passado a `SortStableFunc` — O(n log n) alocações de string para um
valor constante. Içar para fora e escolher a função de comparação uma vez.
(`strings.ToLower(q.Order)`, na linha 369, já está certo.)

#### A ferramenta que faz metade disso
`go fix` mecaniza (d), (e) e laços contados. O diff é auditável, e é preferível
à edição manual em 12 lugares. **Leia o diff** antes de commitar: `go fix` não
sabe que `sort.Slice` com comparador que aloca é o item (g), e vai deixar (g)
intacto.

#### O que prova esta tarefa
Não há mutação a fazer — nenhum item muda comportamento. O que prova é:
- `pwsh -File scripts/verify.ps1` verde, com a contagem de passos;
- `go test ./...` e `go test -race ./...` verdes;
- `git diff --stat` mostrando que nenhum golden e nenhum testdata se moveu;
- `golangci-lint version` **conferido** antes de confiar num zero: o CI fixa
  `v2.12.2`, e binário compilado com Go mais antigo recusa o config antes de
  analisar linha nenhuma.

#### Contrato de relatório
As quatro saídas acima, literais. Se (c) ficou de fora, diga na primeira linha.
E a âncora de mutação que substitui o andaime removido em (a) — sem isso, a
remoção troca um problema por outro.

---

# Task 124 — observabilidade do daemon: quem morre diz por quê

**Tier: modelo barato.** Sem decisão de projeto; o desenho está aqui inteiro.

#### Onde encaixa
Primeira da Fase 4, **antes** da 125 e da 126. Vem primeiro de propósito: a 126
mexe na lógica de partida, e mexer nela enquanto as falhas são mudas repete a
investigação que originou este lote.

#### A evidência que originou esta tarefa (2026-08-26, cofres reais do dono)
Dois daemons na máquina do dono morreram deixando **uma linha** no log e nada
mais — `%LOCALAPPDATA%\gobsidian\run\<VaultKey>.sock.log`:

```
time=2026-08-24T12:56:46.418-03:00 level=INFO msg="daemon iniciado" vault="C:\\Users\\jonyd\\Obsidian\\Revis<U+FFFD>o" socket=...7a43b2b161338f9a.sock read_only=false ociosidade_s=900
(fim do arquivo)
```

O outro, `4568ecbd07c39faa.sock.log`, repetiu a mesma linha duas vezes (24/08
12:56 e 19:15) para `C:\Users\jonyd\Obsidian\Jurisprudencia` — caminho **sem
acento, que não existe no disco**; só `Jurisprudência` existe. O daemon subiu,
falhou ao indexar um caminho inexistente e saiu sem registrar nada.

Do lado do host o sintoma não ajuda em nada — `mcp-server-gobsidian-jurisprudencia.log`:

```
[gobsidian-jurisprudencia] [info] Server transport closed unexpectedly, this is
likely due to the process exiting early.
[gobsidian-jurisprudencia] [error] Couldn't start for Cowork and Code sessions.
```

Um cofre inexistente produziu, ao longo de dois dias, **zero** mensagens
acionáveis em três lugares diferentes.

#### O que vincula esta tarefa
- **stdout pertence ao JSON-RPC.** Todo log vai para stderr via `log/slog`. No
  daemon o stderr é redirecionado para `<socket>.log` pelo spawner; é lá que a
  mensagem tem de cair.
- **Não afirme estado que você não verificou.** O relatório desta tarefa cola o
  conteúdo real do arquivo de log produzido pelo teste, não a descrição dele.
- **Mensagem host-facing não leva caminho absoluto** (B9); absoluto só no `slog`.

#### O que entra

**(a) Todo caminho de saída do daemon loga a causa antes de sair.**
Em `internal/daemon/daemon.go` e `cmd/gobsidian/daemon.go`: falha de
`index.Build`, de `ipc.Listen`, de leitura de config, de montagem do serviço —
cada uma emite `log.Error` com a causa e o campo que a identifica, **antes** do
`return`/`os.Exit`, e o processo sai com código diferente de zero.

**(b) Cofre inválido falha alto, nomeando o caminho.**
Caminho inexistente, inacessível, ou que não é diretório: erro na entrada, com o
caminho na mensagem de log. Vale para o daemon **e** para `serveEmProcesso`, que
hoje morre igualmente calado.

**(c) O errno numérico entra no log de dial.**
`cmd/gobsidian/ponte.go`, nos três pontos de queda: além da mensagem do Go,
registrar o número, via `errors.As` para `syscall.Errno`. A prosa do Windows não
distingue casos que se comportam de forma diferente — foi preciso reverter a
mensagem para descobrir que `An invalid argument was supplied` é `10022` e
`actively refused` é `10061`, e essa distinção decide a Task 126.

#### O que prova esta tarefa
RED que falha **hoje**, em `internal/daemon`:

```go
func TestDaemonComCofreInexistenteRegistraCausa(t *testing.T) {
	dir := t.TempDir()
	inexistente := filepath.Join(dir, "cofre-que-nao-existe")
	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	err := runDaemon(context.Background(), config.Config{VaultPath: inexistente}, log)
	if err == nil {
		t.Fatal("runDaemon devolveu nil para cofre inexistente")
	}
	saida := buf.String()
	if !strings.Contains(saida, "level=ERROR") {
		t.Errorf("nenhum log de ERROR antes da saida; log foi:\n%s", saida)
	}
	if !strings.Contains(saida, inexistente) {
		t.Errorf("o log nao nomeia o caminho do cofre; log foi:\n%s", saida)
	}
}
```

Provas de mutação exigidas, com a saída de `scripts/mutate.ps1` colada:
1. Remover o `log.Error` do ramo de saída por cofre inválido ⇒ o teste acima
   reprova pelo nome e pela linha.
2. Trocar o `return err` por `return nil` nesse mesmo ramo ⇒ reprova.

#### Verificações
Além dos passos:
1. `runDaemon` (`cmd/gobsidian/daemon.go:131`) hoje devolve erro em
   `ipc.Listen` (`:133-135`) **sem logar**, e a montagem do serviço mais adiante
   devolve o erro de `vault.New` do mesmo jeito. Confira que **todos** os
   `return` de erro anteriores e posteriores ao `log.Info("daemon iniciado")`
   (`:139`) passaram a logar.
2. `vault.New` **já valida** existência e tipo do caminho
   (`internal/vault/vault.go:90-95`). Esta tarefa **não** acrescenta validação —
   ela faz o erro que já existe chegar ao log. Se você se pegar escrevendo
   `os.Stat` novo, parou no lugar errado.
3. Rode o binário contra um caminho inexistente e **leia o arquivo**
   `<socket>.log` que ele produz. Uma linha só significa que a tarefa não está
   pronta.
4. Nenhum log novo em stdout — stdout pertence ao JSON-RPC.

#### Regras de execução
- Gate: `pwsh -File scripts/verify.ps1` verde, com a contagem de passos colada.
- Nunca `git checkout`, `git restore`, `git stash`, `git clean` nem `git reset`.
- Nunca `go mod tidy`.
- `golangci-lint version` conferido antes de confiar num zero — o CI fixa
  `v2.12.2`.
- Se um teste falhar por motivo que este brief não explica, **pare e reporte**
  `BLOCKED`; não ajuste a expectativa para o código passar.

#### Comando de mutação
Copie a âncora **do arquivo**, não de memória — âncora digitada sai `EXIT=2`:

```bash
pwsh -File scripts/mutate.ps1 -Path cmd/gobsidian/daemon.go `
  -Anchor 'return fmt.Errorf("abrindo socket do daemon: %w", err)' `
  -Replacement 'return nil' `
  -Test TestDaemonComCofreInexistenteRegistraCausa -Package ./cmd/gobsidian/
```

`0` = o teste reprovou sob mutação (é o que se quer). `1` = a regra está escrita
e não verificada. `2` = âncora ambígua ou build quebrado.

**Files:** `cmd/gobsidian/daemon.go`, `cmd/gobsidian/ponte.go`,
`cmd/gobsidian/servico.go`, `internal/daemon/daemon.go`, mais o `_test.go` novo.

#### Contrato de relatório
Status; commit; RED com a saída falhando e GREEN com a saída passando; as duas
provas de mutação com `EXIT=0` e o texto do `mutate.ps1`; `verify.ps1` com a
contagem de passos; e **o conteúdo literal do arquivo de log** que o teste
produziu, para que se veja a mensagem que passou a existir.

---

# Task 125 — `doctor` diagnostica o daemon

**Tier: modelo barato.** A lista de checagens está fechada abaixo.

#### Onde encaixa
Depois da 124, que cria as mensagens que este comando vai ler.

#### Por que existe
`doctor` é o comando que alguém roda quando **já está confuso**, e hoje ele não
diz uma palavra sobre a metade do sistema que quebra: nem socket, nem lock, nem
log, nem daemon vivo. `internal/doctor/checks.go` tem `checkCacheDir` e nenhum
equivalente para o runtime.

O diagnóstico que originou esta tarefa custou quatro camadas de investigação
manual — processos e parentesco, `%LOCALAPPDATA%\gobsidian\run\`, os logs do
host em `%LOCALAPPDATA%\Claude\Logs\mcp-server-*.log`, e uma reprodução com
stderr capturado. **Todas as quatro respostas estavam disponíveis para um
programa.**

#### O que vincula esta tarefa
- **Saída de console em ASCII puro:** `[OK]`, `[*]`, `[!]`, `[i]`, `[...]`.
  Console PowerShell em CP-850 renderiza o resto como lixo, e este é justamente
  o comando de quem já está perdido.
- **`doctor` e `version` imprimem em stdout de propósito** — são CLI, não
  servidores. A distinção merece comentário onde aparecer.
- **Cor decidida pelo destino, não por `os.Stdout` global** (`internal/console`),
  senão `doctor > relatorio.txt` sai sujo.
- **`doctor` sai 1 nos checks halting** — corrigido antes, não regredir.

#### As checagens que entram
Uma por linha, cada uma com nome estável:

1. **`socket_path`** — o caminho derivado de `ipc.SocketPath(cfg.VaultPath)`, e
   **o que existe lá**, distinguindo: ausente · socket (reparse point) · arquivo
   comum · **diretório** · outro. A distinção não é acadêmica: medido em
   2026-08-26, `net.Dial("unix", …)` devolve `10061` (refused) para arquivo
   comum, socket órfão **e** caminho inexistente, mas `10022`
   (`An invalid argument was supplied`) para **diretório**. Só o `10022` estava
   nos logs do dono, e nenhum dos reprodutores conhecidos o explica — o check
   existe para que a próxima ocorrência se explique sozinha.
2. **`daemon_vivo`** — tenta `DialAndHandshake` com prazo curto. `[OK]` com o
   PID quando responde; `[i]` quando não há daemon (não é erro); `[!]` quando o
   arquivo existe e o dial falha, **imprimindo o errno numérico**.
3. **`daemon_log`** — existência, tamanho, idade e as **três últimas linhas** de
   `<socket>.log`. É onde a Task 124 passou a escrever a causa.
4. **`locks_orfaos`** — `*.sock.lock` cujo PID não corresponde a processo vivo.
   Medido: cinco deles na máquina do dono, de 15 a 19/08, todos com PID morto.
5. **`grafia_do_cofre`** — confere que `cfg.VaultPath` existe **e** compara com
   a grafia real do disco. `C:\Users\jonyd\Obsidian\Jurisprudencia` não existe;
   `Jurisprudência` existe; nada dizia isso a ninguém.

Nenhuma delas é halting salvo a 5 quando o cofre não existe — aí já é falha de
cofre, que o `doctor` hoje trata.

#### O que prova esta tarefa
Teste por checagem, com o estado montado em `t.TempDir()`: socket ausente;
arquivo comum no lugar do socket; **diretório** no lugar do socket; lock com PID
morto; cofre com grafia divergente. Cada um afirma o **marcador e o texto**, não
só o status.

Prova de mutação: apagar a distinção diretório × arquivo comum na checagem 1
(devolver o mesmo texto para os dois) ⇒ o teste do diretório reprova.

#### Verificações
Além dos passos:
1. `doctor > relatorio.txt` sai **sem** códigos de cor — a decisão é pelo
   destino, não por `os.Stdout` global.
2. Marcadores ASCII puros. Nenhum caractere fora de ASCII na saída.
3. `ExitCode` continua 1 nos checks halting e 0 no resto: daemon ausente é
   `[i]`, não falha.
4. A checagem `socket_path` não pode **abrir** o socket para classificá-lo —
   `os.Lstat` e atributos bastam, e abrir muda o que se está diagnosticando.
5. `gobsidian doctor --vault <cofre real>` roda sem daemon vivo e com daemon
   vivo, e diz coisas diferentes nos dois.

#### Regras de execução
- Gate: `pwsh -File scripts/verify.ps1` verde, contagem de passos colada.
- Nunca `git checkout`, `git restore`, `git stash`, `git clean` nem `git reset`.
- Nunca `go mod tidy`.
- `doctor` imprime em **stdout de propósito** — é CLI, não servidor. A distinção
  merece comentário onde aparecer.
- Escopo não encolhe em silêncio: se uma das cinco checagens não couber,
  entregue as outras quatro e **diga qual ficou de fora e por quê**.

#### Comando de mutação
A âncora está no código que você vai escrever; copie a linha literal depois de
escrevê-la. O alvo é a distinção diretório × arquivo comum:

```bash
pwsh -File scripts/mutate.ps1 -Path internal/doctor/checks.go `
  -Anchor '<a linha que devolve o texto do caso DIRETORIO>' `
  -Replacement '<o mesmo texto do caso ARQUIVO COMUM>' `
  -Test TestCheckSocketPathDistingueDiretorio -Package ./internal/doctor/
```

**Files:** `internal/doctor/checks.go`, `internal/doctor/doctor.go`, os
`_test.go` correspondentes, e `docs/OPERACAO.md` (a seção de diagnóstico).

#### Contrato de relatório
A saída **literal** de `gobsidian doctor --vault <fixture>` em dois estados —
um saudável e um com socket quebrado — coladas lado a lado. Mais a prova de
mutação e o `verify.ps1`.

---

# Task 126 — `EnsureStarted` decide por handshake, nunca por errno

**Tier: modelo principal.** Concorrência, orçamento e uma correção de premissa
do próprio plano.

#### Onde encaixa
Depois da 124 e da 125. Fecha, junto com o **item 11** da Fase 4, a cadeia que
produziu o incidente de campo.

#### A evidência (2026-08-26, máquina do dono)
Nas três sessões MCP, em **toda** partida, ao longo de dias:

```
time=2026-08-25T18:26:36 msg="socket do daemon indisponivel; tentando iniciar o daemon"
     err="... connect: An invalid argument was supplied."
time=2026-08-25T18:26:46 msg="nao foi possivel iniciar o daemon; servindo em processo"
     err="socket do daemon nao respondeu em 10s: ..."
time=2026-08-25T18:26:48 msg="servidor pronto" vault=...\Estudo notes=2557 index_origin=cache
```

Dez segundos de pena por sessão, sempre terminando em `servindo em processo`, e
**nenhuma linha `"daemon iniciado"`** nos logs de runtime naquele dia: o daemon
não morreu, não nasceu. Depois de remover os `.sock` órfãos, a mesma partida no
mesmo cofre com o mesmo binário passou a decidir em **559 ms**, com
`conectado ao daemon recem-iniciado via socket`.

#### Correção de premissa — leia antes de codar
O item 11 da Fase 4 dizia: *"discar antes de desvincular — `ECONNREFUSED`
significa órfão e libera o unlink, sucesso significa abortar."*

**A primeira metade está errada como critério.** Medido em 2026-08-26 com
`net.Dial("unix", …)` no Windows: `ECONNREFUSED` (`10061`) é o que devolvem
**arquivo comum, socket órfão de dono morto à força, e caminho inexistente** —
os três. E o erro que apareceu em produção na máquina do dono foi `10022`, que
nenhum desses três reproduz. Classificar por errno decide certo nos casos que já
conhecemos e erra em silêncio no que apareceu de verdade.

**O critério é comportamental: só um handshake bem-sucedido prova daemon vivo.**
Qualquer falha de dial — refused, invalid argument, timeout — significa "não
está servindo", e libera o unlink.

#### O que entra
1. **Espera escalonada no lugar do probe único.** N tentativas de dial dentro do
   orçamento total (ex.: 250 ms × 40 = 10 s), falhando só no fim. Hoje o
   orçamento inteiro é gasto num `select` só, então o chegante tardio nunca vê o
   daemon que subiu 300 ms depois dele.
2. **A decisão é o handshake.** `EnsureStarted` só declara sucesso quando
   `DialAndHandshake` completa. Conectar não basta: um daemon com o `acceptLoop`
   morto (item 10) aceita no backlog do SO e nunca responde.
3. **O errno vai para o log** em toda tentativa falha (produzido na Task 124(c);
   aqui, consumir).
4. **`cmd/gobsidian/daemon.go` participa do mesmo lock** (`daemon/adquirirLock`)
   antes do `Listen`, com o segundo dial idempotente que a ponte já faz.

#### O que prova esta tarefa
- **RED 1:** listener que aceita a conexão e **nunca responde** ao handshake ⇒
  `EnsureStarted` não pode declarar sucesso. Falha hoje.
- **RED 2:** socket que só passa a responder após 2 s ⇒ com orçamento de 10 s, a
  ponte **conecta**, em vez de cair para o modo em processo. Falha hoje.
- **RED 3 (tempestade):** ≥10 pontes simultâneas sobre cofre frio ⇒ exatamente
  **um** daemon, e todas as sessões listam tools dentro do prazo.

Provas de mutação:
1. Trocar o critério de handshake por "dial conectou" ⇒ RED 1 reprova.
2. Remover a espera escalonada (voltar ao probe único) ⇒ RED 2 e RED 3 reprovam.

**Cuidado com o harness:** `StreamReader.Peek()` bloqueia apesar do nome, e já
deixou um ciclo 15h44m parado. Use `ReadLineAsync` com `Wait` limitado. Gate que
pode travar indefinidamente vira gate que se aprende a pular.

#### Verificações
Além dos passos:
1. O fallback em processo **continua obrigatório** nos três pontos (decisão 2 da
   Task 91, mantida pela 92). Esta tarefa muda quando ele dispara, nunca se ele
   existe. Um daemon quebrado não pode transformar a ferramenta em nada.
2. A goroutine vigia de `ln.Close()` (`daemon.go:140-143`) fica como está.
3. Meça o tempo de decisão da ponte antes e depois, no mesmo cofre e com cache
   quente. Número não medido não entra no relatório — escreva "não medido".
4. Rode `pwsh -File scripts/test_orphans.ps1 -Cycles 100` (os quatro cenários, o
   padrão) e confira que `daemon-idle` continua reprovando por `reason=` errado.
5. Não rode o gate de órfãos concorrente com a medição de tempo: um mata os
   processos do outro e produz falso verde.

#### Regras de execução
- Gate: `pwsh -File scripts/verify.ps1` verde, contagem de passos colada.
- Nunca `git checkout`, `git restore`, `git stash`, `git clean` nem `git reset`.
- Nunca `go mod tidy`.
- Matar processo **sempre por PID que você mesmo lançou**, nunca por nome:
  `Stop-Process -Name gobsidian` já matou a sessão real do dono.
- Asserção de tempo atrás de build tag `//go:build race`, em arquivo separado.
- `pipe engole código de saída`: `cmd | tail` devolve o status do `tail`.
  Redirecione para arquivo e leia o `$?` do comando.

#### Comando de mutação
```bash
pwsh -File scripts/mutate.ps1 -Path internal/daemon/lock.go `
  -Anchor '<a linha que exige handshake bem-sucedido>' `
  -Replacement '<a versao que aceita dial conectado>' `
  -Test TestEnsureStartedNaoAceitaDaemonQueNaoResponde -Package ./internal/daemon/
```

Copie as duas âncoras **do arquivo** depois de escrever o código. A segunda
mutação (remover a espera escalonada) usa a mesma ferramenta, com a âncora do
laço de tentativas.

**Files:** `internal/daemon/lock.go`, `internal/ipc/ipc.go`,
`cmd/gobsidian/daemon.go`, `cmd/gobsidian/ponte.go`, os `_test.go`
correspondentes, e `docs/OPERACAO.md` (limites conhecidos: a corrida residual).

#### Contrato de relatório
Os três RED com saída falhando e passando; as duas mutações com `EXIT=0`; o
tempo medido de decisão da ponte **antes e depois** (a referência é 10 s →
559 ms, medida no cofre Estudo, 2.557 notas, cache quente); `verify.ps1`.

---

## Identidade do cofre — verificado, NÃO virou tarefa

Estava planejado como Task 127 e foi **retirado depois de conferir o código**,
em 2026-08-26. Fica registrado para não ser re-proposto.

**O problema observado.** `config.VaultKey` é
`xxhash.Sum64String(strings.ToLower(vaultPath))`, sem normalização Unicode. Na
máquina do dono isso produziu, para o mesmo cofre pretendido, duas instâncias
completas — socket, cache e daemon próprios para cada grafia:

| Grafia | VaultKey | Existe no disco? |
|---|---|---|
| `...\Obsidian\Jurisprudência` | `d34d3da9c925ef62` | sim |
| `...\Obsidian\Jurisprudencia` | `4568ecbd07c39faa` | **não** |
| `...\Obsidian\Revisão` | `1f213394ace393eb` | sim |
| `...\Obsidian\Revis<U+FFFD>o` | `7a43b2b161338f9a` | **não** |

As duas grafias inexistentes vieram do config do host — uma com acento removido,
outra com caractere de substituição, resíduo de round-trip de encoding.

**Por que não virou tarefa.** A correção planejada era "recusar alto caminho que
não existe". **Isso já está implementado**: `vault.New`
(`internal/vault/vault.go:90-95`) devolve `raiz do cofre inacessivel %q` para
caminho ausente e `raiz do cofre nao e diretorio: %q` para arquivo. O erro
existe, é específico, e nomeia o caminho.

O que faltava não era a validação — era **o erro chegar a alguém**. Ele é
devolvido e descartado sem log, e é por isso que o daemon morria mudo. Isso é a
Task 124(a)/(b), e a grafia real do disco aparece na checagem `grafia_do_cofre`
da Task 125. Uma tarefa própria duplicaria as duas.

**Decisão do dono, registrada:** **não** normalizar Unicode em `VaultKey`. NFC
resolveria acento composto × pré-composto e **não** resolveria acento ausente,
que é o caso que de fato ocorreu, ao custo de invalidar todo cache existente.
A chave é sensível a grafia por construção, e isso é aceitável dado que
`vault.New` recusa e o `doctor` mostra. Quem quiser reabrir precisa de caso novo
em que a divergência ocorra com **duas grafias que existem no disco**.

---

# Task 000 — sentinela de fim dos briefs (não é tarefa; não execute)

Existe por uma razão mecânica, medida em 2026-08-26. O `task-brief` do plugin
superpowers extrai com um `awk` que só liga e desliga em cabeçalho casando
`^#+[ 	]+Task[ 	]+[0-9]+`. **Nenhum outro cabeçalho interrompe a extração** —
nem `#`, nem `##`. Consequência: a ÚLTIMA tarefa de qualquer plano vaza até o
fim do arquivo.

Medido: sem esta sentinela, `task-127-brief.md` saiu com **33.421 bytes** e
engoliu sete cabeçalhos — as fases seguintes, a ordem de execução, os tiers, o
prompt de despacho, a auditoria e as pendências. As irmãs têm 3.625 a 4.425
bytes. Um implementador delegado receberia o plano inteiro no lugar da tarefa
dele, e o `check_briefs.ps1` acusaria o brief que destoa em tamanho — que é
exatamente o sintoma que ele existe para pegar.

Não numere uma tarefa real como 000. Ao acrescentar tarefas ao fim deste plano,
**mova esta seção para depois da última** e confira o tamanho do brief recém-
extraído contra o das irmãs antes de despachar.

---

# Fases seguintes — escopo nomeado, sem brief

O detalhe destas fases se escreve quando elas chegarem, com o mesmo padrão
autocontido. Aqui fica o escopo, a ordem interna e o que cada uma depende.

## Fase 4 — daemon e IPC (itens 10, 11, 12, 13)

Ordem: **11 antes de tudo**, porque é o único que muda o estado observável do
sistema.

- **11.** `ipc.Listen` (`internal/ipc/ipc.go:104`) chama `cleanupSocketFile`
  **incondicionalmente** antes do `net.Listen`. Um daemon vivo tem seu arquivo
  removido; ele fica com o descritor ligado a um inode sem nome, invisível para
  toda ponte futura, e sem ociosidade se tiver cliente. Padrão consagrado: discar
  antes de desvincular.
  **Correção de 2026-08-26, medida:** o critério NÃO é `ECONNREFUSED`. Arquivo
  comum, socket órfão de dono morto à força e caminho inexistente devolvem os
  três o mesmo `10061`; e o erro que apareceu em produção foi `10022`, que
  nenhum deles reproduz. O critério é **handshake bem-sucedido** — ver Task 126,
  que detalha a medição.
  **Correção da revisão:** ela diz *"o lock não é a causa: o `Listen` é"*, e isso
  é exclusão indevida. São duas causas que compõem. `daemon/lock.go:65-79`
  registra o incidente medido — dez pontes sob carga, dois daemons vivos, quase
  um minuto entre os dois "daemon iniciado", a segunda ponte nunca viu a
  primeira. A janela do lock explica o **lançamento**; o `Listen` explica a
  **tomada do nome e o órfão**. As duas entram, e `docs/OPERACAO.md` (limites
  conhecidos) se atualiza com o que sobrar.
- **10.** `acceptLoop` (`daemon/daemon.go:152`) faz `return` em qualquer erro de
  `Accept` sem `ctx` cancelado. `EMFILE`/`ENFILE` é transitório e clássico: o
  daemon segue vivo no ticker e nunca mais aceita ninguém. Distinguir temporário
  de terminal, com backoff.
- **13.** O handshake confere `ReadOnly` e `VaultKey`. `DebounceMS`, `CacheDir`,
  `EagerSearch` (e `MaxResults`, depois da Task 120) são os da **primeira**
  ponte. Uma segunda ponte com `--debounce-ms=1000` conecta e é servida com 250.
  Ou os campos que mudam comportamento entram no `HandshakeConfig`, ou a ponte
  **avisa** que a configuração dela foi ignorada.
- **12.** `CloseWrite` é declarado (`ipc.go:71`), exigido no handshake
  (`ipc.go:182` recusa a conexão sem ele) e chamado só pelo dublê de teste. Em
  `servePonteRemota` (`cmd/gobsidian/ponte.go:168-176`) o shutdown faz
  `conn.Close()`, matando a resposta em voo. **Nota de precisão:** quando esse
  caminho dispara, o stdin do host já deu EOF e o host está indo embora, então a
  resposta perdida em geral não tem leitor — o defeito real é o contrato morto,
  não o dano observado. Ou `CloseWrite` passa a ser usado no shutdown, ou some
  do tipo e do handshake.

## Fase 5 — `os.Root` (item 4)

A mudança de arquitetura de maior retorno vista no projeto, e **não é Go 1.26**:
`os.Root`/`os.OpenRoot` está disponível desde **1.24**, dois ciclos atrás.
Confinamento imposto pelo SO, resistente a symlink e a TOCTOU, com `Root.Open`,
`Root.Create`, `Root.Stat`, `Root.Remove`, `Root.OpenFile`.

Fecha o item 4 — hoje `Walk` usa `filepath.WalkDir` (Lstat), um symlink chamado
`nota.md` apontando para fora entra no índice como nota, `note_read` devolve
conteúdo arbitrário e `note_patch` escreve nele — e torna o item 1
**estruturalmente impossível**, em vez de corrigido caso a caso pela Task 113.

Depende da Task 113 estar pronta (D-R-2). Compõe com o item 43 (`Root.OpenFile`
aceita as mesmas flags de Windows).

## Fase 6 — desempenho, e nada aqui entra sem `benchstat`

Ordem por retorno esperado, e **todos são hipóteses** — a própria revisão diz
que nenhum número dela é medido.

- **36 (+22).** Pontuar o BM25 em espaço de IDs densos. `CalculateBM25` trabalha
  em espaço de `string`: `docTermFreqs[p.Path] = make(map[int]float64)` é **um
  mapa por documento** — numa consulta que casa 3.000 notas são 3.001 mapas. A
  Task 88 já construiu o que falta (`baseSoA`, `idPorCaminho`, `caminhos[]`,
  `docLen[]`) e o scorer nunca adotou. Subsume o item 22 (`avgdl` O(N do cofre)
  por consulta, `bm25.go:91`) como efeito colateral. Complicação honesta: a
  camada delta ainda é mapa de string.
- **23.** `getFieldWeight` roda por **posição**, não por posting — `idx.Get`
  (mapa + `RLock`) e `strings.Contains(n.TitleNorm, term)` são invariantes na
  posição. Junto vem uma **imprecisão de relevância**: `Contains` é substring,
  não termo — buscar `ar` dá peso de título (3.0) a uma nota chamada "Barra".
  Essa metade é correção, não desempenho, e pode subir de fase.
- **27.** `Analyze` (`search/analyzer.go:78`) sem pré-alocação nem caminho rápido
  ASCII. É o custo dominante da construção do índice de busca.
- **28.** `GenerateSnippet` re-tokeniza termos já tokenizados
  (`snippet.go:78`): com `limit: 200`, são 200 × (#termos) análises redundantes.
- **25.** `IsCloudOnly` é um syscall por arquivo em `vault/walk.go:185`, e
  `d.Info()` na linha 174 já traz os mesmos bits no Windows.
- **37.** SoA no índice de metadados. O maior em memória e o mais caro; só depois
  do 36.
- **38.** APIs em estilo `append` (`PositionsInto`) para o `variablemake` do 1.26
  ter o que empilhar. **Antes de mexer:** rodar `-gcflags=-m` e listar o que já
  passou a ser empilhado sozinho — otimizar o que o compilador resolveu é dívida
  pura.
- **Ordinal de token no índice** (formato 7). Habilita frase exata sem ler bytes
  (substitui a Task 111), proximidade `NEAR/3` e ranking por proximidade.
- **Busca por prefixo e aproximada.** `baseSoA` guarda `termos` **ordenados**, o
  que dá busca por prefixo em O(log n) de graça. É a lacuna funcional mais
  visível para o usuário.

## Fase 7 — `netcheck` e Go 1.26

Ordem **obrigatória**: 19 antes de 42.

- **19.** `tools/netcheck/netcheck.go:74` exige `sel.X` ser um `*ast.Ident` que
  resolve para o pacote `net`. Escapam `d := net.Dialer{}; d.DialContext(...)`,
  `(&net.Dialer{}).Dial(...)`, `net.ListenConfig{}.Listen(...)`,
  `syscall.Socket` e `golang.org/x/sys/unix` — e `x/sys` **já está no módulo**.
  A regra RNF-30 é verificável para a forma que o código usa hoje, não para a
  categoria. Banir os **tipos** `net.Dialer`/`net.ListenConfig`/`net.Resolver` e
  as chamadas de socket via `syscall`.
- **42.** Só então adotar `Dialer.DialUnix`/`ListenUnix` (Go 1.26), que põem a
  rede no **nome do método** — a regra vira "só estes dois métodos", impossível
  de furar por variável. Elimina o workaround de `internal/ipc/dialUnix`
  (~22 linhas). Adotar antes de corrigir o analisador faria a chamada nova
  passar pelo guarda **por acidente, não por permissão**.
- **41.** `errors.AsType[T]` em 9 pontos, 8 concentrados em `service/write.go`.
  Exige o directive em `go 1.26.0`.
- **43.** `os.OpenFile` com `FILE_FLAG_SEQUENTIAL_SCAN` no laço de boot e
  `FILE_FLAG_RANDOM_ACCESS` em `vault.ReadRange`. É a única recomendação da
  seção cujo ganho não se justifica por mecanismo — prototipar e medir antes de
  assumir o custo de trocar 14 `os.ReadFile` por `OpenFile` atrás de build tag.
- **Directive.** `go 1.25.0` continua bastando **hoje**: o toolchain 1.26.5 já
  entrega runtime e compilador via `GOTOOLCHAIN=auto`. Passa a ser insuficiente
  no momento em que 41, 42 ou 43 forem adotados — `go build` passa e
  `go vet` reprova (`errors.AsType requires go1.26 or later`), e o CI roda
  `go vet` em três SOs. Se o `bench.yml` subir para 1.26, o campo `runner` de
  `docs/bench-baseline.json` é regravado — não porque os números mudaram (o A/B
  da revisão mostra que quase não mudam), mas porque o campo passaria a
  descrever uma coisa e o gate mediria outra.

## Fase 8 — lifecycle e medição

- **39.** `context.WithCancelCause` no lifecycle. Hoje o motivo vive em
  `l.reason` e quem está a jusante só vê `context.Canceled` — é por isso que
  `shutdownExitCode` precisa adivinhar. O texto de `reason=` é contrato do
  harness de órfãos: a causa **acompanha**, não substitui. Junto e independente
  de versão: `signals.go:29` faz `case <-ch:` e descarta o `os.Signal` recebido
  — qual sinal chegou é diagnóstico grátis, e vai num campo novo, **nunca** no
  `reason`.
- **40.** Gate de goroutine vazada, via `GOEXPERIMENT=goroutineleakprofile`
  (1.26). O gate de órfãos prova que **o processo** morre nos quatro mecanismos;
  não prova nada sobre goroutine presa antes disso — e o desenho tem goroutines
  deliberadamente fora do `WaitGroup` (`watchStdin`, as duas cópias de
  `servePonteRemota`). Converte "não vazou processo" em "não vazou nada".
- **44.** `testing.T.ArtifactDir()` e a flag `-artifacts`, encaixando na cultura
  de "a saída real colada é o entregável". `runtime/metrics`:
  `/sched/goroutines` por estado e `/sched/goroutines-created` em `vault_stats`,
  que hoje expõe só `NumGoroutine`.
- **`measure.ps1` num gate.** Ele existe, é bom, e não está em gate nenhum — nem
  no `verify.ps1`, nem no CI. É a mesma forma de `check_doc_refs` antes de
  2026-08-11. Três decisões continuam sem medição e **nenhuma é respondível pela
  bateria de benchmark que existe**, porque exigem variante compilada ou RSS de
  servidor de vida longa: `GOGC` no boot real, `debug.FreeOSMemory()` (−195 MB) e
  `maxSnippetWorkers = 8`.
- **`GOGC` não se re-litiga sem dado novo de RSS.** A medição da revisão
  reproduz a metade que nunca esteve em disputa (o benchmark melhora com
  `GOGC=400`) e não toca nas duas que decidiram (boot real e RSS). O dado novo é
  que `GOGC=off` **piora a busca em 39,55% (p=0,002)**, o que **reforça** a
  rejeição registrada.

## Fase 9 — recursos escolhidos pelo dono

- **`vault_lint` (item 48).** Metade da máquina já existe: `vault_stats` com
  `include_health` já percorre o cofre contando órfãs, links quebrados e âncoras
  quebradas. O que falta é a regra ser **do usuário**: frontmatter obrigatório
  por pasta, padrão de nome, heading exigido, tag obrigatória, limite de órfãs —
  declarativas, rodando sobre o índice em memória, sem I/O, devolvendo violações
  com caminho e offset. O argumento não é conveniência: o script externo precisa
  **reler o cofre inteiro do disco** para responder o que o servidor já tem em
  RAM, e não tem acesso aos links resolvidos nem aos backlinks, que são
  justamente o que uma regra estrutural quer checar.
- **Lixeira: listar e restaurar.** `note_delete --to_trash` move para `.trash/`,
  que é diretório **excluído da varredura** — então nem `note_list` a vê. Não há
  tool para listar nem para restaurar; o usuário precisa do Explorer. A tool de
  listagem tem de varrer o disco, não o índice, precisamente porque `.trash` não
  está indexado.
- **`note_rename` distinto de `note_move`.** Renomear é o caso comum e hoje paga
  a semântica inteira de move — **inclusive o item 9**, que é o defeito de
  atomicidade: `write.go:596-618` lê o arquivo inteiro, `WriteAtomic` no destino
  e `_ = os.Remove(absFrom)` com **erro descartado**. Se o remove falha (arquivo
  travado pelo Obsidian no Windows, cenário comum), a tool devolve sucesso com a
  nota duplicada nos dois caminhos, e o watcher indexa as duas. Pior: as notas
  citantes já foram reescritas antes (linha 535) — se a leitura da origem falhar
  depois, o retorno é erro **com os links já apontando para um destino que não
  existe**, sem rollback e sem registro do que ficou pela metade.
  **`os.Rename` resolve o corpo**; o lote de reescritas precisa de plano de
  compensação ou de uma ordem que falhe cedo. O item 9 é pré-requisito desta
  tarefa, não consequência dela.

## Itens que ficam fora, e por quê

- **Alternativa E (pseudo-heading no parser).** Decisão de produto, D-R-1.
  Resolve mais que a D — conserta `note_read`, `note_patch`, âncora de wikilink e
  peso de BM25 de uma vez — e custa o corpus de 48 golden files,
  `IndexCacheParserVersion++` (reconstrói o cache de **todo** cofre no boot
  seguinte) e a pergunta de o que `replace_heading_and_section` faz com uma seção
  que não tem linha de heading. `Synthetic` precisaria aparecer em
  `note_metadata`, ou a tool passa a afirmar estrutura que o arquivo não tem.
- **Alternativa F (`note_read` com `contains`/`from_text`).** Segunda linguagem
  de endereçamento ao lado de heading e bloco; as duas divergem no dia em que uma
  ganhar tratamento de acento que a outra não tem. Só se D e E forem recusadas —
  e D foi escolhida.
- **`Service.Index` como abstração.** 12 métodos vazando `*index.Note`,
  `index.Query`, `index.TagCount`, `index.Backlink`: `service` não pode ser
  testado sem `index`, e a fronteira não protege nada. Ou estreitar para o que
  `service` de fato usa, ou apagar a interface. É refatoração ampla, sem defeito
  observado atrás dela — fica registrada, não agendada.
- **`sync.RWMutex` único do índice de metadados.** `Replace` segura o lock
  exclusivo durante `os.Stat` + `ReadAll` + `parser.Parse` — I/O de disco
  **dentro** do lock de escrita, e toda leitura concorrente para. Ler e parsear
  fora do lock e só publicar dentro dele é mudança pequena com efeito direto na
  latência sob edição ativa. Entra na Fase 6 se alguém medir; hoje é hipótese.
- **Nomear SQLite/FTS5 como alternativa rejeitada** no PRD. O projeto
  reimplementou BM25, índice invertido, codec binário próprio, arena mmap, cache
  de duas camadas e LRU de trecho; a decisão não está registrada em lugar nenhum
  e é a primeira pergunta de qualquer revisor externo. É trabalho de documento,
  não de código — vai junto da próxima edição do PRD.
- **Prompts MCP.** `prompts/` é a superfície do protocolo feita para "resuma
  este cofre", "encontre notas órfãs". Recurso novo, sem defeito atrás.

---

# Ordem de execução, e por quê

```
Fase 0   104 → 105                       instrumento antes de conserto
Fase 1   106 → 107 → 108 → 109 → 110 → 111 → 112     o incidente
Fase 2   113 → 114 → 115 → 116 → 117     os críticos
Fase 3   118 → 119 → 120 → 121 → 122 → 123           barato e de alto retorno
```

**Por que a Fase 1 vem antes dos críticos.** É o único bloco com falha
**observada em produção**, agora por dois relatos independentes, e o custo de
A + C + H é quase nulo. Enquanto não for feito, o produto perde para o
concorrente na tarefa central, e o usuário descobre isso sozinho — que foi o
que aconteceu. Os itens 1, 2 e 3 são graves e **nenhum tem exploração
observada**; a travessia exige que alguém peça uma escrita com `..\`.

Quem discordar dessa ordem tem um argumento legítimo (o item 1 é escrita
arbitrária no sistema de arquivos) e a inversão é barata: 113 é autocontida e
pode subir para logo depois da 105 sem tocar em nada da Fase 1. **É decisão do
dono, e trocar a ordem não invalida nenhum brief.**

Dependências que **não** se invertem:
- 104 antes de 120 — o gate corrigido é o que impede a quinta recorrência.
- 106 antes de 107, 109 e 112 — as três dependem do contrato de `offset`.
- 108 depois de 112, ou com a menção a `note_outline` pendente.
- 110 antes de 111 — a 110 monta a fixture que confirma o falso positivo.
- 113 antes da Fase 5 (`os.Root`).
- 119 antes de 120(e), se `max_results` for clampar em algo que passa por
  `ResolvePath`.

# Tiers, e por que cada um

**Modelo barato: 105, 107, 108, 109, 114, 116, 118, 120, 121, 122, 123.**
Todas têm o teste difícil escrito literalmente no brief, a decisão fechada no
texto, e nenhuma exige projetar o que não pode ser enganado. Transcrição de
código completo roda bem no tier mais barato — foi o que a experiência das
Tasks 33 e 34 mostrou.

**Modelo principal: 104, 106, 110, 111, 112, 113, 115, 117, 119.**
- **104** e **115**: o entregável é fazer um instrumento **passar a poder
  reprovar**, e o modo de falha de um modelo barato aqui é declarar que passou.
- **106**, **112**, **119**: o entregável é o contrato/semântica, com quatro
  casos que interagem.
- **110**, **111**: algoritmo, com decisão de mecanismo dentro.
- **113**: superfície de segurança; correção parcial fica verde com o buraco
  aberto.
- **117**: concorrência; o modo de falha é um deadlock novo que o teste fácil
  não pega.

# Prompt de despacho

O brief da seção da tarefa **basta**: ele carrega onde encaixa, o que o vincula,
as armadilhas aplicáveis, a decisão a acertar, os passos, o código de teste
completo, a prova de mutação e o contrato de relatório. **Não injete contexto
acumulado da sessão** — as tarefas foram escritas para não precisar dele.

## O prompt do orquestrador

Escrito depois de a primeira rodada de despacho sair sem ele. O prompt do
implementador diz o que ELE faz; este diz o que **você** faz. Sem esta metade,
o orquestrador improvisa a parte cara — decidir se aceita, e o que fazer quando
não aceita.

```text
Você é o orquestrador de um lote de tarefas de um plano existente. Você NÃO
implementa: você despacha, verifica o que volta, decide, e integra.

O PLANO
docs/superpowers/plans/2026-08-16-revisao-fixes.md. Leia dele, agora:
  - "Decisões fechadas para a batelada inteira" (D-R-1 a D-R-8);
  - "Ordem de despacho do lote barato" (a tabela de dependências);
  - "O que conferir na volta, antes de aceitar".
Não leia os briefs das tarefas — eles são para quem implementa. Você só precisa
saber o que cada tarefa TOCA, para não colidir duas na mesma superfície.

1. DESPACHAR
Uma tarefa por agente, e uma worktree por agente. NUNCA dois agentes na mesma
árvore de trabalho: neste projeto isso já custou trabalho não commitado recolhido
por um `git add` alheio, e uma rotina de limpeza que matou a sessão real do
usuário. Se você não puder isolar em worktree, despache UMA de cada vez.

Antes de despachar em paralelo, confira a interseção de arquivos. Duas tarefas
que editam o mesmo arquivo vão em sequência, não em paralelo — e a segunda só
sai depois de a primeira estar mesclada, não só entregue.

O que você manda para cada agente é o prompt da seção "O prompt, literal" com o
cabeçalho específico da tarefa. Não resuma o prompt e não injete contexto da sua
sessão: os briefs foram escritos para não precisarem dele.

Diga a cada agente qual é o estado herdado dos gates. Hoje: verify.ps1 reprova
só em check_tool_params, e isso pertence à Task 120. Sem essa frase, um agente
embrulha uma falha NOVA num "conforme esperado" e você não vê.

2. VERIFICAR O QUE VOLTA — não acredite no relatório
O relatório é uma alegação. Estes são os passos, e cada um já pegou algo real
nesta batelada:

  a) Rode VOCÊ MESMO cada `mutate.ps1` que o relatório cita, e confira o EXIT.
     Só 0 serve. Prova de mutação colada não vale sem re-execução — uma já veio
     factualmente errada aqui, e outra citava um SHA que não era do arquivo.
  b) Confira todo SHA citado com `git cat-file -t`. Um relatório desta batelada
     trouxe um SHA-256 que não correspondia a nenhuma versão do arquivo, com o
     "antes" e o "depois" iguais entre si e errados os dois.
  c) `pwsh -File scripts/verify.ps1` na árvore mesclada, não na worktree isolada.
     É a integração que interessa.
  d) `pwsh -File scripts/audit_reports.ps1 <N>` — ele procura hedge apresentado
     como medição, prova escrita no condicional, e tarefa completa sem relatório.
  e) Confira que o ledger MOVEU. Tarefa que não está nele não está feita: a
     próxima sessão tem o ledger, não o seu contexto.
  f) LEIA O DIFF DOS TESTES, não só o resultado. A falha mais barata para um
     modelo pressionado é enfraquecer a asserção até ela passar.
  g) Compare o que o relatório diz que mudou com `git diff --stat` da base real
     da tarefa (`git merge-base`). Diff contra o SEU HEAD mostra como "removido"
     tudo o que você commitou depois de o agente ramificar — isso é artefato,
     não reversão. Confirme antes de acusar.

3. DECIDIR — três saídas, e o critério é o tamanho
  ACEITAR: o entregável funciona, as mutações passam, o escopo foi respeitado.
  CORRIGIR NO LUGAR: defeito pequeno, mecânico, que você conserta em minutos sem
    redesenhar nada — um caractere corrompido num comentário, um nome duplicado,
    um campo novo não documentado, uma linha de ledger desatualizada. Conserte,
    e DIGA no seu relatório o que consertou e de quem era.
  DEVOLVER: o entregável não faz o que a tarefa existe para fazer, ou a prova não
    prova. Aí você escreve um prompt de retrabalho — ver o passo 5.

O critério não é gravidade da consequência, é quanto projeto a correção exige.
Reescrever a decisão central de um script é devolução. Trocar um nome é correção.

4. INTEGRAR
Mescle na ordem em que despachou. Espere conflito no ledger — todas as tarefas
escrevem no mesmo ponto dele — e resolva por UNIÃO: nenhum bloco de tarefa se
exclui. Depois de resolver, confira `grep -c "<<<<<<<"` no arquivo E no commit:
é possível commitar marcador de conflito sem perceber, e já aconteceu.

Só rode a bateria completa DEPOIS de tudo mesclado. Duas tarefas verdes em
isolamento podem não compilar juntas.

5. DEVOLVER UMA TAREFA
O prompt de retrabalho tem quatro partes, nesta ordem:
  - POR QUE FOI REPROVADA, com a evidência que você mesmo produziu: o comando
    que rodou, a saída literal, e o que ela contradiz. Nunca "não ficou bom".
  - O QUE ENTREGAR, item a item, com o que NÃO fazer junto.
  - A PROVA que a primeira tentativa não fez, com o procedimento passo a passo.
  - O corpo padrão de "O prompt, literal" com o número da tarefa.
Acrescente a seção "O que a primeira tentativa errou" ao brief da tarefa NO
PLANO, não só no prompt. O prompt é descartável; o plano é o que a próxima
tentativa lê.

6. O SEU RELATÓRIO
Ao fim do lote, entregue: o que cada tarefa mudou, as provas de mutação que VOCÊ
re-executou, o verify.ps1 da árvore integrada, o que você corrigiu no lugar e de
quem era, o que devolveu e por quê, e o que ficou pendente para o próximo lote.
Se você corrigiu algo de um agente e não disse, o próximo lote repete o defeito.

O QUE NUNCA FAZER
  - Aceitar prova de mutação sem re-executar.
  - Despachar em paralelo duas tarefas que editam o mesmo arquivo.
  - Deixar o escopo crescer: um agente que "aproveitou para" consertar outra
    coisa entregou uma tarefa diferente da que você pediu. Registre e avalie
    separado; não mescle por conveniência.
  - Commitar em master. Branch primeiro.
  - Dizer que o lote acabou com o ledger desatualizado.
```

## O prompt, literal

Troque `<N>` pelo número da tarefa. Nada mais muda entre despachos.

```text
Você vai executar UMA tarefa de um plano existente, num repositório Go em Windows.

ANTES DE TOCAR EM QUALQUER ARQUIVO, leia nesta ordem:
  1. CLAUDE.md na raiz do repositório — inteiro. Ele tem as regras que não são
     negociáveis e as armadilhas que já custaram caro neste projeto.
  2. docs/superpowers/plans/2026-08-16-revisao-fixes.md, três partes apenas:
     - a seção "Decisões fechadas para a batelada inteira" (D-R-1 a D-R-8);
     - a seção "Ambiente de teste — o que já existe, e em que pacote";
     - a seção "# Task <N>" inteira.
  Não leia as outras tarefas. Elas não te vinculam.

Depois rode, e cole a saída no seu relatório:
  pwsh -File scripts/sdd.ps1 base <N>

ESCOPO
Execute a Task <N> e nada além dela. Se encontrar outro defeito pelo caminho,
REGISTRE no relatório e NÃO conserte — escopo que cresce em silêncio custa mais
caro que escopo que encolhe. Se alguma parte da tarefa não der para fazer,
entregue o resto inteiro e diga o que ficou de fora e por quê. "BLOCKED: <motivo>"
é resposta melhor que uma entrega que parece completa.

TESTES
O código de teste da seção "O teste que não é óbvio" é para ser transcrito, não
reprojetado. Ele já usa os helpers que existem no repositório, e a seção
"Ambiente de teste" diz em que PACOTE cada arquivo tem de ficar — errar o pacote
não compila. Se algum identificador do teste não existir no código, confira o
nome real antes de inventar um: a seção diz onde procurar.

O que NÃO fazer nos testes:
  - não use time.Sleep para esperar indexação; reconstrua com newTestService;
  - não apague uma asserção que falha — ela é o defeito, e é o que a tarefa
    existe para consertar;
  - não regenere golden com -update para fazer passar; -update grava o que o
    código produz, não o que está certo;
  - não afrouxe um teto de requisito.

PROVA DE MUTAÇÃO — obrigatória
Rode o comando `scripts/mutate.ps1` que o brief traz e COLE A SAÍDA LITERAL.
O código de saída é invertido de propósito:
  0 = o teste reprovou sob mutação  -> a regra está VERIFICADA
  1 = o teste passou                -> a regra está escrita, não verificada
  2 = inconclusivo                  -> âncora ambígua, ou a mutação quebrou o
                                       build; falha de compilação NÃO é cobertura
Só 0 é aceitável. Com 1, o teste não cobre o que diz cobrir — conserte o teste.
Com 2, escolha outra âncora ou uma mutação que compile.
O código de saída diz o que aconteceu, nunca por quê: diga também QUAL asserção
falhou. Prova escrita no futuro do pretérito ("se removêssemos X, falharia") não
é prova.

ANTES DE DIZER QUE ACABOU
  pwsh -File scripts/verify.ps1        # a bateria inteira; cole a contagem de passos
  golangci-lint version                # confira que é v2.12.2 antes de confiar num zero
Registre a tarefa no ledger .superpowers/sdd/2026-07-25-gobsidian-v01/progress.md
ANTES de reportar conclusão. Confira todo SHA que escrever com `git cat-file -t`.

CUIDADOS DE AMBIENTE (Windows)
  - Não rode `go mod tidy`. Nunca. Várias dependências estão fixadas sem
    importador e tidy as removeria, junto com um pin que é decisão fechada.
  - Pipe engole código de saída: `cmd | tail` devolve o status do tail.
    Redirecione para arquivo e leia o código do comando.
  - Para inserir código Go com escapes, use a ferramenta de edição, não script
    Python: "\n" dentro de string Python vira quebra de linha real e corrompe a
    linha. Se usar script mesmo assim, abra com newline="" na leitura E na
    escrita, ou o arquivo inteiro vira CRLF e o gofmt reprova um .go que estava
    perfeito.
  - Não mate processo por nome (`Stop-Process -Name gobsidian`): isso já matou a
    sessão real do usuário. Mate por PID que você mesmo lançou.

RELATÓRIO — é o entregável, não o resumo dele
Entregue, nesta ordem:
  1. O que mudou, por arquivo, em uma linha cada.
  2. A saída literal de cada `mutate.ps1` exigido, com o EXIT e a asserção que
     falhou.
  3. A saída de `verify.ps1` (contagem de passos e o código de saída).
  4. O que o contrato de relatório da tarefa pedir a mais.
  5. O que ficou de fora, se ficou.
"Testes passam" não é evidência; a saída do teste é. Número que você não mediu
não entra — escreva "não medido", que ninguém vai brigar com isso.
```

## Ordem de despacho do lote barato

As onze de tier barato, com o que cada uma exige estar pronto antes:

| Task | depende de | por quê |
|---|---|---|
| 105 | — | só YAML de CI |
| 116 | — | independente |
| 118 | — | independente |
| 121 | — | independente |
| 122 | — | independente |
| 123 | 106 | o `// better condition` sai lá; se sobrar, sai aqui |
| 108 | — (112 opcional) | a menção a `note_outline` só entra depois da 112 |
| 107 | **106** | o teste usa `ReadRequest.Offset` |
| 109 | **106** | o objeto por item sobrepõe `offset` |
| 114 | **113** | é a 113 que faz `validatePlatformPath` rodar na escrita |
| 120 | **104** | o gate corrigido é o que impede a quinta recorrência |

Despachar 105, 116, 118, 121 e 122 **em paralelo** é seguro: tocam arquivos
disjuntos. As outras esperam a dependência.

**Nunca despache duas tarefas na mesma worktree ao mesmo tempo.** Já aconteceu
aqui três vezes numa sessão: um `git add` de caminho explícito recolheu trabalho
não commitado de outro agente, uma rotina de limpeza matou a sessão real do
usuário, e o gate de órfãos rodando em paralelo com medições teve os processos
mortos por essa limpeza — o que produziria falso verde. Uma worktree por tarefa,
ou uma tarefa por vez.

## O que conferir na volta, antes de aceitar

Cada item abaixo já falhou neste projeto:

1. **Rode você mesmo o `mutate.ps1` que o relatório cita** e confira que o EXIT
   bate. Prova de mutação falsa apareceu duas vezes, e uma estava factualmente
   errada — o reconciliador foi removido e a suíte continuou verde.
2. **`git cat-file -t <sha>`** para todo SHA citado no ledger. A Task 31 foi
   registrada num commit que não existe.
3. **`pwsh -File scripts/audit_reports.ps1`** — ele procura hedge apresentado
   como medição, prova escrita no condicional, "coberto implicitamente", SHA
   inexistente e tarefa completa sem relatório.
4. **Confira que o teste RODOU**, e não pulou. Um `t.Skip` que dispara sempre é
   um teste apagado com aparência de verde.
5. **`git diff --stat` nos golden e testdata.** Se algum se moveu numa tarefa
   que não devia mexer em comportamento, pare e pergunte por quê.
6. **Leia o diff dos testes**, não só o resultado. A forma de falha mais barata
   para um modelo pressionado é enfraquecer a asserção até ela passar.

# Auditoria de cada entrega

Antes de aceitar qualquer tarefa como concluída, rodar você mesmo:

```powershell
pwsh -File scripts/audit_reports.ps1        # hedge apresentado como medicao, SHA inexistente, tarefa sem relatorio
pwsh -File scripts/verify.ps1               # a bateria inteira
```

E, para cada prova de mutação citada no relatório, **rodar de novo** o comando
`mutate.ps1` colado e conferir que o `EXIT` bate. Prova de mutação escrita no
condicional apareceu duas vezes neste projeto, e uma das duas estava
factualmente errada.

# Pendências abertas deste lote

Decisões de produto que **não** são de implementação e ficam para o dono:

- **Alternativa E** (pseudo-heading no parser) — depois de a Task 112 mostrar
  quantos candidatos sintéticos aparecem num cofre real. O número decide.
- **`max_results`**: implementar como teto (Task 120e) ou remover a flag. A
  tarefa implementa; se o dono preferir remover, é uma linha e um `TOOLS.md`.
- **Ordem entre Fase 1 e Fase 2** — a travessia de caminho é mais grave e menos
  observada. Inverter é barato.
- **Item 9 (`note_move` sem atomicidade)** ficou na Fase 9, atrelado a
  `note_rename`. Se o dono quiser antes, é tarefa autocontida e não depende de
  nada deste lote.
