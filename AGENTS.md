# gobsidian — instruções para qualquer agente que mexa neste repositório

Servidor MCP em Go que expõe um cofre Obsidian local a hosts MCP. Roda como subprocesso sobre stdio, em Windows.

**Se você foi despachado para executar uma tarefa numerada, o seu brief é a fonte.** Ele fica em `.superpowers/sdd/2026-07-25-gobsidian-v01/task-<N>-brief.md` e é autocontido: carrega onde a tarefa encaixa, as decisões já fechadas, a evidência medida do defeito que ela corrige, as armadilhas que se aplicam, as verificações exigidas, e o contrato de relatório. **Leia o brief inteiro antes de escrever qualquer linha.** Não peça contexto adicional — ele não existe em lugar nenhum além do brief, deste arquivo e do `CLAUDE.md`.

Este arquivo é o contrato do **ambiente**. O brief é o contrato da **tarefa**. Onde os dois falarem do mesmo assunto, o brief vence.

---

## 1. Proibições absolutas

Violar qualquer uma destas causa dano que não aparece imediatamente. Não há exceção, não há "só desta vez".

**Nunca rode `git checkout`, `git restore`, `git stash`, `git clean` nem `git reset`.** Há trabalho não commitado neste repositório o tempo todo, e um subagente já destruiu trabalho exatamente assim. Para desfazer o que **você** escreveu, edite de volta ou apague o arquivo específico que você criou.

**Nunca rode `go mod tidy`.** Várias dependências estão fixadas sem importador ainda — `goldmark`, `yaml.v3`, `x/text`. `tidy` removeria todas, junto com o pin obrigatório do SDK de MCP, que é decisão fechada do PRD. Se o build reclamar de entrada faltando em `go.sum`, use `go get <caminho-do-pacote>@<versão-fixada>` — caminho do **pacote**, não do módulo. O Go informa o caminho certo na própria mensagem de erro.

**Nunca escreva em stdout a partir de código alcançável de `serve`.** stdout pertence ao JSON-RPC. Todo log vai para stderr via `log/slog`. Um `fmt.Println` ali corrompe a sessão, e o sintoma é o servidor sumir do host sem erro nenhum. Os comandos `doctor` e `version` imprimem em stdout **de propósito** — são CLI, não servidores.

**Nenhum pacote sob `internal/` ou `cmd/` importa `net` ou `net/*`.** `net/http` chega transitivamente pelo SDK e isso é esperado; o check de CI inspeciona os nossos pacotes. Verifique com `pwsh -File scripts/check_net.ps1`.

**Nenhum tipo do SDK de MCP cruza para fora de `internal/mcpsrv`.** `internal/service` fala tipos de domínio.

**Não commite o arquivo `task-<N>-base.txt` sozinho.** Ele fica sujo de propósito; o primeiro commit da tarefa o recolhe. Commitá-lo isolado move o HEAD e torna a base defasada, e regravar recursa.

---

## 2. Comandos

```bash
pwsh -File scripts/verify.ps1     # build, test -race, vet x3, gofmt, golangci-lint, check_net
pwsh -File scripts/build.ps1
```

**Verde obrigatório antes de qualquer commit.** `verify.ps1` cobre as oito etapas e para no primeiro erro. Ele existe porque a lista solta convida a rodar três dos cinco.

**O `golangci-lint` entrou na bateria em 2026-07-29** e não é mais comando separado: enquanto era, era um comando que alguém esquecia — as Tasks 33–42 fecharam com 22 achados (18 `errcheck`, 4 `revive`) e nenhum dos dez relatórios mencionava ter rodado o linter. **`go vet` não pega `errcheck`.** A etapa confere a versão antes de rodar, porque um binário compilado com Go mais antigo que o `go 1.25.0` do `go.mod` recusa o config antes de analisar linha nenhuma, e um zero local não diria nada sobre o CI.

### Prova de mutação — use a ferramenta, nunca a mão

```bash
pwsh -File scripts/mutate.ps1 -Path internal/watcher/apply.go `
  -Anchor 'if n, ok := idx.Get(path); ok {' `
  -Replacement 'if n, ok := idx.Get(path); ok && false {' `
  -Test TestApply -Package ./internal/watcher/
```

**O código de saída é invertido de propósito:**

| Saída | Significa | O que fazer |
|---|---|---|
| `0` | O teste **reprovou** sob mutação | É o que você quer. Cole a saída no relatório. |
| `1` | O teste **passou** sob mutação | A regra está escrita, não verificada. **Escreva o teste que falta.** |
| `2` | Inconclusivo | Âncora não casou, ou a mutação quebrou o build. Ajuste e rode de novo. |

O script exige âncora com ocorrência única, restaura em `finally` conferindo por SHA-256, e trata falha de compilação como inconclusivo em vez de contá-la como cobertura. Se `-Anchor` não casar ele sai `2` e diz isso — **copie o texto do arquivo, não digite de memória**.

### Conferência dos briefs, antes de despachar

```bash
pwsh -File scripts/check_briefs.ps1 <primeira> <ultima>
```

Só interessa a quem **escreve** as tarefas, não a quem as executa. O `task-brief` extrai de `### Task N` até o próximo `###`, então tudo que ficar sob o cabeçalho do marco não chega a brief nenhum — já produziu um lote em que a tarefa de fechamento saiu sem Regras de execução e sem Contrato de relatório. O script confere as seções exigidas e acusa brief que destoa em tamanho dos irmãos, que é o sintoma mais confiável.

### Auditoria do próprio relatório

```bash
pwsh -File scripts/audit_reports.ps1 <N>
```

Rode antes de entregar. Sai `1` quando encontra hedge apresentado como medição, prova de mutação escrita no condicional, não-resposta do tipo "coberto implicitamente", SHA inexistente, ou seção ausente.

---

## 3. Como uma tarefa é executada

1. Leia `.superpowers/sdd/2026-07-25-gobsidian-v01/task-<N>-brief.md` **inteiro**.
2. Implemente transcrevendo o desenho do brief. **O plano é a fonte** — não improvise uma variante. Se o código do plano não compilar, corrija o erro mecânico e **diga exatamente o que mudou**. Se um teste falhar por motivo que o brief não explica, **pare e reporte**; não ajuste a expectativa para o código passar.
3. Rode as provas de mutação que o brief exige, com `scripts/mutate.ps1`.
4. Rode `pwsh -File scripts/verify.ps1` até ficar verde.
5. Grave o relatório em `.superpowers/sdd/task-<N>-report.md`.
6. Rode `pwsh -File scripts/audit_reports.ps1 <N>` e resolva os achados.
7. Commite em Conventional Commits, em inglês, com a mensagem que o brief indica.

---

## 4. O que "pronto" significa

Oito tarefas deste projeto foram entregues como concluídas sem terem sido. Cada regra abaixo veio de uma dessas.

**Não escreva número que você não mediu.** Se não mediu, escreva **"não medido"** — ninguém vai brigar com isso. O que não pode é hedge (*tende a*, *aproximadamente*, *e.g.*, *ex:*, *deveria*) ao lado de algo apresentado como resultado. Uma tabela chamada "Resultado da Medição" com *"ex: 408ms em teste local"* já foi commitada aqui: o "ex:" fazia todo o trabalho.

**Não afirme estado que você não verificou.** O README declarou "v0.1 publicada" sem tag e sem release.

**Prova de mutação escrita no condicional não é prova.** *"Se removermos X, o teste falharia"* apareceu em dois relatórios, e uma das duas estava factualmente errada — a regra foi removida e a suíte continuou verde, deixando um requisito P0 com cobertura zero através de uma revisão que o aprovou. Prova real está no **passado** e traz a saída do `go test` colada.

**Um teste que não pode falhar é pior que teste ausente**, porque reporta cobertura que não existe. Antes de dizer que testou: mute a regra, confirme que um teste nomeia a falha, restaure.

**Schema que promete e código que ignora é pior que parâmetro ausente.** `note_list` declarava `fields` no schema e o descartava. O modelo do outro lado pede três campos, recebe tudo, e não tem como saber que o pedido não fez nada. Ou implemente, ou tire do schema e da documentação.

**Campo de API com valor fixo mente sempre.** `alias_collisions` era `Collisions: 0` literal. Aparecia na resposta e nunca foi verdade.

**Não deixe sua deliberação no código.** Comentários começando com `Wait,`, `For the sake of simplicity` e `Actually` já foram commitados; um documentava um defeito como se fosse decisão. Comentário explica **por que** o código é assim; raciocínio sobre o que fazer não é comentário.

**Escopo não encolhe em silêncio.** Se alguma parte não deu para fazer, entregue o resto inteiro e **diga o que ficou de fora e por quê**. Reduzir escopo é decisão de quem pediu. `BLOCKED` com o motivo é resposta melhor que uma entrega que parece completa.

**Meça pela camada que o requisito nomeia.** RNF-04 diz *"latência de `vault_search`"*. A primeira medição chamou `search.CalculateBM25` direto e reportou **0,58 ms**; medida por `service.Search`, com trecho e filtros, a mesma coisa dá **entre 6 e 174 ms**. Não foi número inventado — foi aritmética honesta sobre a fatia errada, rotulada com o nome do todo. Antes de escrever o número, pergunte por qual função o requisito passaria se um usuário o exercitasse.

**Distribuição com mediana zero está medindo o relógio, não o código.** A mesma medição publicou `Mínimo: 0s, Mediana: 0s` ao lado de um p95, porque consultava termos que casavam **uma** nota em 500 — a consulta mais seletiva possível. Se metade das amostras dá zero, a carga não existe. E p95 sobre formatos de custo diferente não é p95 de nada: vira o percentil do formato mais caro presente, e muda quando a proporção muda. **Meça por formato.**

**Asserção de tempo não sobrevive ao `-race`.** O detector multiplica a latência por 2× a 6× — aqui, busca por frase foi de 174 ms para 1,00 s. O gate roda `-race`, então um teto de tempo cobrado sob ele reprova por motivo que não é o do produto. Guarde a asserção atrás de constante com build tag (`//go:build race`), em arquivo separado, e **continue registrando a medição nos dois modos**. A condição pertence à compilação, não ao corpo do teste — mesma razão do código de plataforma.

**Alvo não atingido e registrado é informação; alvo escondido é dívida.** Quando o número estoura, não use `t.Skip` e não afrouxe o alvo dos outros casos. Faça o teste cobrar um **teto medido** naquele caso, escreva ao lado que o alvo do RNF não está atingido, e registre a lacuna. O teto guarda contra piorar enquanto a lacuna espera tarefa.

**O relatório é o entregável, não o resumo dele.** Comando rodado, saída real colada, prova de mutação. "Testes passam" não é evidência; a saída do teste é.

### Forma do relatório

- **Status** — `DONE` | `DONE_WITH_CONCERNS` | `BLOCKED` | `NEEDS_CONTEXT`
- **Commit** — SHA curto e assunto
- **Evidência de TDD** — o comando do RED e sua saída falhando, depois o comando do GREEN e sua saída passando. Não "segui TDD".
- **Prova de mutação** — por regra reivindicada: o que foi mutado, qual teste reprovou **pelo nome e pela linha**, a saída colada, e a confirmação de restauro
- **As verificações do brief** — cada uma com o resultado **real**, inclusive as que deram certo
- **O que ficou de fora** — e por quê. Vazio é resposta aceitável; ausente não é.
- **`git status --porcelain`** — sem arquivo estranho, nada do usuário tocado

---

## 5. Regras de código que a revisão cobra

**`ctx` onde há espera real, e respeitado de verdade.** Funções que podem **bloquear** recebem `ctx` e verificam: I/O de arquivo, varredura, worker pool, watcher, chamadas MCP. Leitura de env var, resolução de caminho e cálculo em memória **não** recebem. `context.Background()` dentro de `internal/` é defeito. Quando o parâmetro existe só por consistência de assinatura, nomeie-o `_`.

**Camada de pré-filtro que abre arquivo derrota o filtro que ela precede.** Quem roda **antes** do guarda precisa do mesmo guarda. Anexo é indexado por nome, **nunca lido**; arquivo somente-nuvem **nunca é aberto**, porque abrir dispara download síncrono.

**Chave derivada calculada em dois lugares diverge**, e a divergência só aparece no caminho menos usado. Toda chave passa por **uma** função, inclusive nos pontos que já estavam certos — o objetivo não é consertar os errados, é tornar a próxima divergência impossível sem tocar na função.

**Teste de mecanismo de recuperação que deixa o caminho normal ligado mede o caminho normal.** Teste de fallback **desconecta** o caminho principal, ou não é teste de fallback.

**Pergunte o que um valor zero significa.** Zero é offset válido, debounce válido, contagem válida. Onde também puder significar "ausente" ou "omitido", os dois precisam ser distinguíveis — `offsetUnknown = -1`, `ReadOnlySet`, um erro em vez de uma varredura vazia.

**Código de plataforma atrás de build tag, em arquivo separado.** Nunca `if runtime.GOOS ==` dentro de lógica compartilhada. Em teste, `runtime.GOOS` é aceitável para pular casos.

**Saída de console em ASCII puro:** `[OK]`, `[*]`, `[!]`, `[i]`, `[...]`. Console PowerShell em CP-850 renderiza o resto como lixo.

**Sem `helpers.go`, `utils.go`, `common.go`.** Arquivo assim é preocupação que ninguém nomeou.

---

## 6. Armadilhas de ferramenta neste ambiente

Todas já custaram tempo aqui. O sintoma de cada uma aponta para o lugar errado.

**`bash` no PATH é o do WSL.** Ele não enxerga `C:/Users/...` e responde `No such file or directory` para um arquivo que existe. Use o bash do Git (`C:\Program Files\Git\bin\bash.exe`), e converta `\` para `/` antes de passar caminho — o bash consome contrabarra como escape.

**Here-string do PowerShell (`@'...'@`) não funciona na ferramenta Bash.** O `@` entra no texto. Para mensagem de commit multilinha ali, use heredoc: `git commit -F - <<'EOF' ... EOF`.

**PowerShell: array de um elemento vira escalar.** `$x = if ($c) { @() } else { @('-race') }` desenrola para string, e `@x` a espalha caractere a caractere — o `go test` vai procurar pacotes `r`, `a`, `c`, `e`. Tipe explicitamente: `[string[]]$x = ...`.

**PowerShell: `Write-Output` dentro de pipeline atribuído não imprime.** Capture primeiro, imprima depois.

**Script Python que edita arquivo versionado precisa de `newline=""` na leitura E na escrita.** Modo texto converte o arquivo inteiro para CRLF e o `gofmt` reprova um `.go` perfeitamente formatado. Já custou dois commits. Em PowerShell, o equivalente seguro é `[System.IO.File]::ReadAllBytes` / `WriteAllBytes`.

**`str.replace` que não casa não falha — segue em silêncio.** Toda edição por script leva `assert` do texto-âncora antes de substituir, e conferência do resultado no disco depois.

**Ferramenta que reescreve `.md` pode gravar em cp1252.** Os docs deste projeto são em português. Depois de qualquer reescrita, confira:

```bash
python -c "open('ARQUIVO.md',encoding='utf-8').read()" && echo "[OK] UTF-8 valido"
```

---

## 6.1 Skills de Go instaladas neste repositório, e quando cada uma serve

Estão em `.claude/skills/` e são versionadas de propósito: uma skill que só existe na máquina de quem instalou não reduz o erro de mais ninguém. Não são leitura obrigatória — invoque a que corresponde ao que você está fazendo. O mapa abaixo é por **defeito que este projeto já cometeu**, não por assunto.

| Antes de… | Invoque | Porque, aqui |
|---|---|---|
| Commitar qualquer coisa | `golang-lint` | As Tasks 33–42 fecharam com 22 achados de `golangci-lint` — 18 `errcheck` e 4 `revive` — e nenhum dos dez relatórios mencionava ter rodado o linter. **`go vet` não pega `errcheck` e o `verify.ps1` não roda `golangci-lint`.** |
| Passar ou receber `ctx` | `golang-context` | `context.Background()` dentro de `internal/` já passou por revisão aqui, numa função que lê arquivo. |
| Mexer em goroutine, canal, `atomic` ou mutex | `golang-concurrency` | Um `map` compartilhado entre a goroutine que escreve e a que lê é corrida mesmo que só cresça, e `atomic` no valor não protege a estrutura. |
| Escrever ou revisar teste | `golang-testing` **e** `mutation-proof-discipline` | A skill de Go dá a forma; a deste projeto dá a prova. Teste que não pode falhar é o defeito mais caro daqui. |
| Comparar, embrulhar ou engolir erro | `golang-error-handling` | `err == fsnotify.ErrEventOverflow` com comparação de string ao lado, e um comentário afirmando usar `errors.Is` sem usar. |
| Valor zero, nil, `defer` em laço | `golang-safety` | Zero é offset válido, debounce válido e contagem válida — onde também puder significar "ausente", os dois precisam ser distinguíveis. |
| Nomear pacote, tipo, construtor, teste | `golang-naming` | Sem `helpers.go`, `utils.go`, `common.go`. |
| Navegar código que você não escreveu | `golang-gopls` | Encontra chamadores antes de mudar assinatura. Uma mudança de wiring já foi propagada para `serve` e esquecida em `doctor`. |

`golang-how-to` é orquestradora: em dúvida sobre qual carregar, comece por ela.

**Onde uma skill de Go conflitar com este arquivo ou com o `CLAUDE.md`, este projeto vence.** As skills descrevem Go idiomático em geral; as regras daqui vieram de defeitos concretos e algumas são deliberadamente mais estritas — a de `ctx`, por exemplo, proíbe o parâmetro decorativo que muito código Go aceita.

**A skill `obsidian` não serve para o servidor.** Ela é sobre escrever *plugins* do Obsidian em TypeScript — Vault API, `processFrontMatter`, ESLint, submissão à comunidade. O gobsidian **lê** um cofre em Go e nunca roda dentro do Obsidian. A única parte deste repositório em que ela ajuda é `tools/parity-dumper/`, que é um plugin de verdade, usado para despejar o `metadataCache` e gerar a referência de paridade da Task 25.

## 7. Onde as coisas ficam

| Caminho | O quê |
|---|---|
| `docs/superpowers/plans/2026-07-25-gobsidian-v01.md` | O plano. Tarefas 12 a 42. |
| `.superpowers/sdd/2026-07-25-gobsidian-v01/progress.md` | O ledger. É o que a próxima sessão tem no lugar do seu contexto. |
| `.superpowers/sdd/2026-07-25-gobsidian-v01/task-<N>-brief.md` | O brief autocontido da tarefa |
| `.superpowers/sdd/task-<N>-report.md` | Onde você grava o relatório |
| `.claude/skills/` | Skills do projeto — **fonte única**. `.agents/` é espelho gerado e não versionado; não o edite. |
| `.claude/workflows/` | Workflows |
| `docs/PRD.md` | Requisitos, prioridades, decisões fechadas D1–D13 |
| `docs/ARCHITECTURE.md` | Camadas, fluxos, decisões AD-01–AD-09 |
| `docs/TOOLS.md` | Contrato de cada tool: schema, retorno, erros |
| `docs/WINDOWS.md` | OneDrive, MAX_PATH, casing, fsnotify |

`CLAUDE.md` tem o histórico completo das armadilhas e o estado do projeto. Este arquivo é o subconjunto necessário para executar sem lê-lo inteiro.

## 7.1 O MCP do `gopls`

Está disponível como `mcp__gopls__*`. Use-o em vez de `grep` para qualquer pergunta sobre **símbolo**: o `grep` acha texto, o `gopls` acha referência resolvida, e a diferença aparece em nome comum, em método de interface e em `_test.go` de outro pacote.

| Ferramenta | Use quando | O caso concreto deste projeto |
|---|---|---|
| `go_symbol_references` | **Antes de mudar qualquer assinatura.** | Uma mudança de wiring foi propagada para `serve` e esquecida em `doctor`, e só a revisão pegou. Uma chamada aqui teria listado os dois. |
| `go_search` | Achar onde algo vive sem saber o pacote | Busca difusa; `foo` casa `Foo`, `fooBar`, `pkg.Baz`. |
| `go_package_api` | Descobrir o que um pacote expõe antes de importá-lo | `internal/index` e `internal/vault` têm superfície grande; ler a API é mais barato que abrir os arquivos. |
| `go_file_context` | Entender um arquivo que você não escreveu | Traz o contexto resolvido, não o texto. |
| `go_diagnostics` | Depois de editar, antes de rodar teste | Erro de compilação em segundos, sem esperar `go build`. |
| `go_workspace` | Confirmar em que módulo você está | |
| `go_vulncheck` | Só quando perguntado sobre CVE | **Não aja sozinho no que ele sugerir.** Atualizar dependência aqui é decisão fechada do PRD, e várias estão fixadas de propósito. |
| `go_rename_symbol` | Renomear em todo o workspace | **Leia o parágrafo abaixo antes.** |

### As quatro coisas que dão errado com ele

**`go_rename_symbol` escreve nos arquivos.** É a única ferramenta desta lista que muda o repositório, e **você não pode desfazê-la com `git checkout` ou `git reset`** — os dois estão proibidos aqui. Antes de renomear: rode `go_symbol_references` e olhe a lista. Depois de renomear: rode `git diff` e leia. E lembre que ele **não** renomeia dentro de `.md` — o plano, o `docs/` e os briefs continuam citando o nome antigo, e plano e código não podem divergir.

**Diagnóstico limpo não é gate verde.** `go_diagnostics` e `golangci-lint` são conjuntos diferentes de análise. O gate deste projeto é `pwsh -File scripts/verify.ps1` mais `golangci-lint run ./internal/... ./cmd/...`, e as Tasks 33–42 fecharam com 22 achados de lint que nada além do `golangci-lint` reportou. **Diagnóstico limpo permite continuar; só o gate permite commitar.**

**O diagnóstico é do workspace inteiro, não do seu arquivo.** Ele reporta erro em código que você não escreveu. Já aconteceu: `agent/` guarda assets de plugin com `.go` de exemplo cujos imports não resolvem, e o `gopls` os reportava como erro do workspace. A reação natural — acrescentar a dependência — viola a regra de nunca rodar `go mod tidy`, chegando por outro caminho. Hoje `agent/go.mod` os tira do grafo; se erros assim voltarem, **confira de qual diretório vêm antes de tentar consertá-los**.

**Ele responde sobre o código, não sobre o contrato.** `go_symbol_references` diz quem chama uma função; não diz se `docs/TOOLS.md` promete um campo que ninguém preenche. Esse defeito já passou por aqui duas vezes e nenhuma ferramenta de símbolo o pega.

---

## 8. Quando parar e reportar em vez de continuar

Escalar é mais barato para todo mundo do que uma entrega que parece completa.

- Um teste falha por motivo que o brief não explica.
- O brief manda fazer algo que contradiz este arquivo ou o `CLAUDE.md`.
- A correção exigiria mexer em arquivo que o brief não lista.
- Você precisaria rodar um comando proibido para prosseguir.
- `scripts/mutate.ps1` sai `1` e você não consegue escrever o teste que falta.
- Você não entendeu o que a tarefa pede.

Nesses casos: `BLOCKED`, com o que tentou, o que aconteceu, e o que precisa ser decidido.
