# gobsidian

MCP server em Go. Expõe cofre local Obsidian a hosts MCP, roda como subprocesso sobre stdio.

Spec vive em `docs/`: `PRD.md` (requisitos + decisões fechadas D1–D13), `ARCHITECTURE.md` (camadas + AD-01–AD-09), `ESTRUTURA.md` (árvore + convenções), `TOOLS.md` (contrato de cada tool), `WINDOWS.md` (OneDrive, MAX_PATH, casing). Plano de implementação em `docs/superpowers/plans/2026-07-25-gobsidian-v01.md` — fonte de onde código é transcrito. Plano e código não podem divergir.

**Docs em português. Ferramenta que reescreve `.md` pode gravar em cp1252.** `caveman-compress` fez isso com este arquivo: prosa saiu cp1252, bloco de código saiu UTF-8 — arquivo misto, `Expõe` virou `Exp\xf5e`. Nada se perde (cp1252 é reversível, ao contrário de `U+FFFD`), mas transcodificar o arquivo inteiro de uma vez duplica os acentos das regiões que já eram UTF-8.

Depois de qualquer ferramenta que reescreva um `.md`, confira:

```bash
python -c "open('CLAUDE.md',encoding='utf-8').read()" && echo "[OK] UTF-8 valido"
```

Reparo de arquivo misto: decodificar byte a byte, tentando sequência UTF-8 válida primeiro e caindo pra cp1252 onde falhar. Conferir palavras-sonda acentuadas depois, inclusive uma dentro de código inline.

## Comandos

```bash
pwsh -File scripts/verify.ps1    # bateria inteira, para no primeiro erro
pwsh -File scripts/build.ps1
pwsh -File scripts/test_orphans.ps1 -Cycles 100
golangci-lint run ./...          # exige v2.12.2; v1.x nem carrega o config

pwsh -File scripts/mutate.ps1 -Path <arquivo> -Anchor '<texto>' -Replacement '<texto>' -Test <Nome> -Package ./internal/x/
pwsh -File scripts/audit_reports.ps1 [N]    # relatórios e ledger contra evidência falsa
```

`mutate.ps1` tem **código de saída invertido de propósito**: `0` = o teste reprovou sob mutação (regra verificada), `1` = o teste passou (regra escrita, não verificada), `2` = inconclusivo (âncora ambígua, ou a mutação quebrou o build — falha de compilação não é cobertura). Ele exige âncora com ocorrência única, restaura em `finally` e confere o restauro por SHA-256. Detalhe na skill `mutation-proof-discipline`.

`audit_reports.ps1` procura hedge apresentado como medição, prova de mutação escrita no condicional, não-resposta do tipo "coberto implicitamente", SHA citado no ledger que não existe, e tarefa completa sem relatório. Sai `1` com achados. Não julga conteúdo — localiza a frase para alguém conferir.

`verify.ps1` roda build, `go test -race`, vet nos três alvos, `gofmt` e `check_net`. Existe porque a lista solta convida a rodar três dos cinco. Aceita `-SkipCross` e `-SkipNet` para iteração rápida; o gate roda tudo.

Fluxo de execução tarefa a tarefa:

```bash
pwsh -File scripts/sdd.ps1 status      # ledger + git
pwsh -File scripts/sdd.ps1 base 19     # ANTES de a tarefa comecar
pwsh -File scripts/sdd.ps1 brief 19
pwsh -File scripts/sdd.ps1 review 19   # empacota o diff desde a base gravada
```

O `sdd.ps1` embrulha os scripts do plugin superpowers, cujo caminho embute a versão — que já mudou de 6.1.1 para 6.2.0 no meio deste projeto, alterando a assinatura de `review-package` e movendo os artefatos para um subdiretório por plano. Chamada literal quebra na próxima atualização.

`base` existe porque `review-package` precisa do commit **anterior ao início** da tarefa. `HEAD~1` descarta em silêncio tudo menos o último commit de uma tarefa com vários, e a revisão passa a olhar meio diff sem avisar.

**`golangci-lint` local verde não significa CI verde.** O `go.mod` declara `go 1.25.0`, e um binário compilado com Go mais antigo recusa o config antes de analisar linha nenhuma: `can't load config: the Go language version (go1.24) used to build golangci-lint is lower than the targeted Go version (1.25.0)`. O CI fixa `v2.12.2` de propósito — com a versão flutuando, os dois lados resolvem binários diferentes e a checagem local para de dizer qualquer coisa sobre o pipeline. Confira `golangci-lint version` antes de confiar num zero.

## Dependências

**Nunca rode `go mod tidy`.** Várias deps fixadas mas sem importador ainda — `goldmark`, `yaml.v3`, `x/text` só ganham um em M1 e M3. `tidy` removeria elas, junto com pin do SDK MCP, que é decisão fechada (PRD D6).

Quando pacote importa dep pela primeira vez, `go.sum` pode faltar entradas transitivas. Comando certo: `go get <caminho-do-pacote>@<versão-fixada>` — caminho do **pacote**, não do módulo. `go get <módulo>@<versão>` é no-op quando módulo já requerido naquela versão. Go informa caminhos certos na própria mensagem de erro.

Piso é `go 1.25.0`, não 1.24. Forçado por `go-sdk@v1.5.0`, que declara 1.25.0 no próprio go.mod.

## Regras que não são negociáveis

**stdout pertence ao JSON-RPC.** Todo log vai pra stderr via `log/slog`. `fmt.Println` em código alcançável de `serve` corrompe sessão — sintoma: servidor some do host sem erro nenhum. `doctor` e `version` imprimem em stdout de propósito — são comandos CLI, não servidores. Distinção merece comentário onde aparece.

**Nenhum pacote sob `internal/` ou `cmd/` importa `net` ou `net/*`.** `net/http` e `x/oauth2` chegam transitivamente pelo SDK — esperado. Check de CI inspeciona nossos pacotes, não fecho transitivo. Garantia verificável em PRD §6.4.

**Nenhum tipo do SDK MCP cruza pra fora de `internal/mcpsrv`.** `internal/service` fala tipos de domínio, não importa SDK. Torna migração de protocolo mudança de um pacote só — e protocolo já quebrou compat várias vezes.

**`ctx` onde há espera real.** Funções que podem **bloquear** recebem `ctx` e respeitam: I/O de arquivo, varredura, worker pool, watcher, chamadas MCP. Leitura de env var, resolução de caminho, cálculo em memória não recebem. `ctx` que nenhum corpo verifica ensina revisor a ignorar `ctx` — quando o parâmetro existe só por consistência de assinatura, nomeie-o `_`, como em `watchStdin`. **Não há mais exceção à regra.** `lifecycle.Shutdown` recebe `ctx` e descarta o cancelamento via `context.WithoutCancel`: o context raiz já está cancelado quando ela roda, então derivar os orçamentos dele faria toda etapa nascer expirada. `WithoutCancel` preserva os valores e joga fora só o cancelamento. Quem limita tempo ali são os orçamentos por etapa e o `hardLimit`.

**Código de plataforma atrás de build tag, em arquivo separado.** Nunca `if runtime.GOOS ==` dentro de lógica compartilhada. Em teste, `runtime.GOOS` aceitável pra pular casos.

**Saída de console em ASCII puro:** `[OK]`, `[*]`, `[!]`, `[i]`, `[...]`. Console PowerShell em CP-850 renderiza resto como lixo, e `doctor` é justamente o comando que alguém roda quando já está confuso.

**Sem `helpers.go`, `utils.go`, `common.go`.** Arquivo assim é preocupação que ninguém nomeou.

Commits em Conventional Commits, em inglês.

## Armadilhas que já custaram caro

Cada uma passou por revisão e só apareceu depois. Estão aqui pra não voltarem.

**`io.TeeReader` não propaga EOF.** Copia bytes, EOF não é byte. Usar `mirrorReader`, que faz `dst.CloseWithError(err)`. Sem isso monitor de stdin do lifecycle fica inerte e `lc.Wait()` só retorna por acidente. Espelho é **auxiliar**: falha de escrita nele não pode virar erro da leitura principal — mataria sessão saudável por motivo que cliente não pode agir.

**Vigília do pai precisa de `exitTime`, não só creation time.** Windows mantém PID e creation time consultáveis muito tempo depois da morte do processo. Comparar `(pid, created)` nunca detecta pai morto — deixou 5 de 5 órfãos no primeiro teste ponta a ponta. Em Unix, comparar ppid capturado no startup, não constante 1: sob Docker+tini, systemd ou s6 o reaper não é PID 1.

**Goroutine parada em `Read` não é desenrolável por cancelamento de context.** Por isso `watchStdin` fica fora do `WaitGroup` — incluí-la trava `Wait()` quando sinal ou pai dispara primeiro. Vigias de sinal e de pai entram, porque fazem `select` em `ctx.Done()`.

**`ctx.Canceled` no retorno do serve loop é encerramento normal.** Duas detecções de EOF independentes correm — SDK e lifecycle — e qual vence decide o valor. Tratar como falha faz host ver erro aleatório a cada desconexão limpa.

**Handler que devolve `error` Go faz SDK montar `IsError` sem `StructuredContent`.** Devolver resultado de erro com `Out` zerado manda `{"notes":0,...}` junto, e cliente não distingue falha de cofre vazio no canal que ele lê primeiro.

**Confinamento de caminho tem duas camadas.** `validateLocal` (léxica: NUL, `..`, raiz, `IsLocal`, regra de plataforma) e `Canonicalize` (por componente, via `filepath.Rel`). `filepath.IsLocal` barra nome de dispositivo — `Resolve(root, "COM1")` escrevia em porta serial antes disso. Mas regra de ponto/espaço no fim de componente vale **só no Windows**: em Linux `Notas ` é nome legal, e rejeitá-lo lá torna notas reais inalcançáveis.

**`CanonicalPath` não garante grafia do disco.** Esta camada não consulta disco; preserva o que chamador passou. Quem produz grafia real é `vault.Walk`.

**`d == nil` no callback do `WalkDir` é falha na própria raiz** — cofre desmontado, share caído. Devolver `nil` ali faz varredura reportar sucesso com zero entradas, e servidor afirma com confiança que cofre está vazio. Cofre inacessível e cofre vazio não podem produzir mesma resposta.

**Script Python que edita arquivo versionado precisa de `newline=""` na leitura E na escrita.** Ler com `encoding='utf-8'` e gravar em modo texto converte o arquivo inteiro para CRLF no Windows, e `gofmt` reprova um `.go` que estava perfeitamente formatado. Custou dois commits.

**`str.replace` que não casa não falha — segue em silêncio.** Duas edições de plano "deram certo" sem editar nada, e o ledger ficou duas tarefas desatualizado do mesmo jeito. Toda edição por script leva `assert` do texto-âncora antes de substituir, e conferência do resultado no disco depois. Vale também para o plano: `sed -n 'N,Mp' arquivo | cat -A` depois de mexer em snippet com escapes, porque `"\n"` dentro de string Python já virou quebra de linha real e corrompeu a linha.

**`-update` de golden grava o que o código produz, não o que está certo.** Aceitar a saída sem ler transforma a suíte em tautologia que fixa o bug de hoje como contrato de amanhã. Depois de gerar, leia cada `.json` e confira contra o que você esperava **antes** de rodar.

**Regra que sobrevive a mutação não está verificada, está escrita.** Na Task 13, sete regras do módulo sobreviviam a mutantes com a suíte verde — inclusive a que o comentário do próprio fix defendia. Ler o teste não acha isso. Para cada regra que importa, apague-a, confirme que um teste nomeia a falha, restaure.

**Chave de mapa calculada em dois lugares diverge, e a divergência só aparece no caminho menos usado.** `byAlias` era escrito minúsculo por `alias.go` no boot e cru por `Replace`; `resolve.go` lia minúsculo. Enquanto o índice só era construído no boot, os dois concordavam. Quando o watcher tornou `Replace` e `Remove` alcançáveis, `Remove` passou a procurar uma chave que não existia, e a entrada velha sobrevivia: `[[STJ]]` continuava resolvendo, com `state=ok`, para uma nota que já tinha saído do índice. Toda chave derivada passa por **uma** função — `aliasKey(alias)` — e todo acesso passa por ela, inclusive os que já estavam certos. Não é para consertar os errados: é para tornar a próxima divergência impossível sem tocar na função.

**Camada de pré-filtro que abre arquivo derrota o filtro que ela precede.** `CorrelateRenames` roda antes de `index.Replace` e chamava `vault.ReadAll` em todo caminho do lote — anexo inclusive, placeholder de nuvem inclusive. `Replace` respeita as duas regras (anexo por nome, nuvem não hidratada); a camada anterior a ele não respeitava, então as regras valiam para um caminho que nunca era alcançado. **Quem roda antes do guarda precisa do mesmo guarda.**

**Teste de mecanismo de recuperação que deixa o caminho normal ligado mede o caminho normal.** `TestOverflowReconciliationFull` injetava overflow com o watcher ativo; os eventos comuns aplicavam as três mudanças, e a reconciliação nunca era exercida. Removido o reconciliador inteiro, o teste passava em 2,8 s — cobertura zero num requisito P0, através de uma revisão que o aprovou. Teste de fallback **desconecta** o caminho principal, ou não é teste de fallback.

**Feature P1 não tem direito de apagar dado P0.** O campo inline do Dataview consumia o span do valor, e `fonte:: [[STJ]]` deixava de produzir link nenhum — links que o commit anterior já coletava. Quando uma feature opcional muda o que uma obrigatória já entregava, o A/B contra o commit anterior é o que revela.

**Flag booleana ou inteira não distingue "omitida" de "definida com zero".** `config.Flags` tem companheiros `ReadOnlySet` e `DebounceMSSet`. **Toda** chamada a `config.Load` precisa preenchê-los com `cmd.Flags().Changed(nome)` — esquecer em um subcomando faz flag virar no-op silencioso.

## Quando uma tarefa está pronta

Esta seção existe porque oito tarefas deste projeto foram entregues como concluídas sem terem sido. Nenhuma das falhas abaixo é hipotética; todas aconteceram aqui e custaram uma auditoria.

**Não escreva número que você não mediu.** `docs/OPERACAO.md` chegou a trazer uma tabela de "Resultado da Medição v0.1" com *"Concluído abaixo do alvo (ex: 408ms em teste local)"* e *"Tende a ficar ~30-45 MB"*. O primeiro é exemplo, o segundo é expectativa; nenhum é medição. Alvo não atingido e registrado é informação. Alvo não medido apresentado como resultado é ficção com aparência de tabela. Se não mediu, escreva **"não medido"** — ninguém vai brigar com isso.

**Não afirme estado que você não verificou.** O README declarou "v0.1 publicada" sem tag, sem release, sem o gate de órfãos ter rodado. A frase custou nada de escrever e teria custado caro em quem confiasse nela.

**Um teste que não pode falhar é pior que teste ausente**, porque reporta cobertura que não existe. Três casos reais aqui: o teste de paridade passava com referência vazia (o guard checava se o diretório existia, e ele existia vazio); `TestBuildBOM` afirmava que o heading *existia* e nunca o offset, enquanto o offset estava errado em 3 bytes; e as fixtures de exclusão usavam extensões que o filtro descartaria de qualquer jeito. **Antes de dizer que testou: apague a regra, rode, confirme que um teste nomeia a falha, restaure.** Se nada falhar, você escreveu a regra, não a verificou.

**Schema que promete e código que ignora é pior que parâmetro ausente.** `note_list` declarava `fields` no schema e o descartava. O modelo do outro lado pede três campos, recebe tudo, e não tem como saber que o pedido não fez nada — o schema é justamente o que ele lê para decidir. Ou implemente, ou tire do schema e da documentação.

**Campo de API com valor fixo mente sempre.** `alias_collisions` era `Collisions: 0` literal. Aparecia na resposta e nunca foi verdade.

**Não deixe sua deliberação no código.** Três comentários começando com "Wait," e "For the sake of simplicity" foram commitados. Um deles documentava um defeito como se fosse decisão. Comentário explica por que o código é assim; raciocínio sobre o que fazer não é comentário.

**Prova de mutação escrita no condicional não é prova.** "Se removermos X, o teste falha" apareceu em dois relatórios desta fase, e uma das duas estava factualmente errada — o reconciliador foi removido e a suíte continuou verde. O tempo verbal é o sinal: prova real está no passado e traz saída colada. Rode `scripts/mutate.ps1` e cole o que ele imprimiu.

**Confira todo SHA que você escrever no ledger.** A Task 31 foi registrada em `14210ee`, que não existe no repositório — `git cat-file -t` responde `fatal: Not a valid object name`. Ledger que aponta para o vazio é pior que ledger desatualizado, porque parece preciso. `scripts/audit_reports.ps1` confere todos.

**Registre no ledger antes de dizer que acabou.** Oito tarefas e onze commits entraram sem uma linha. A próxima sessão não tem seu contexto — ela tem o ledger, e um ledger desatualizado faz alguém re-executar trabalho pronto, que é a falha mais cara deste fluxo. `pwsh -File scripts/sdd.ps1 status` mostra o que ele diz hoje.

**Escopo não encolhe em silêncio.** Se alguma parte da tarefa não deu para fazer, entregue o resto inteiro e **diga o que ficou de fora e por quê**. Reduzir escopo é decisão de quem pediu. `BLOCKED` com o motivo é resposta melhor que uma entrega que parece completa.

**O relatório é o entregável, não o resumo dele.** Comando rodado, saída real colada, prova de mutação. "Testes passam" não é evidência; a saída do teste é.

## Estado

M0 completa, etiquetada `m0-lifecycle`: ciclo de vida, `internal/vault`, servidor MCP mínimo com `vault_stats`, `doctor`, e 100 ciclos de encerramento abrupto com zero órfãos.

Dívida de revisão de M0 **paga**. Tasks 9, 10 e 11 haviam fechado sem revisão fresca; a revisão que faltava rodou e virou trabalho (plano `docs/superpowers/plans/2026-07-26-m0-review-fixes.md`). Três defeitos reais que os gates existentes não pegavam: `doctor` saindo 0 com cofre inacessível; o gate de órfãos não gateando em `reason=`, de modo que servidor morrendo sozinho dava rodada verde sem mecanismo nenhum disparar; e `cmd/gobsidian` sem teste algum.

Lint limpo nos três alvos (`GOOS=linux/darwin/windows`), depois de 39 achados que estavam vermelhos desde o commit de bootstrap. CI ganhou `fmt` (gofmt + vet cruzado) e `lint-windows`, sem o qual todo arquivo `//go:build windows` ficava sem análise.

**M1 completa** — Tasks 12 a 26: parser e as quatro extensões goldmark congelados por um corpus de 48 golden files, o índice com offsets de byte, resolução, backlinks, consultas, a fachada de serviço, as cinco tools de leitura e os resources, e paridade verificada contra um dump real do `metadataCache` do Obsidian.

**M2 completa** — Tasks 27 a 32: fachada sobre fsnotify com filtro de relevância unificado em `vault.Classify`, debounce de tique único com conjunto sujo, verificação de mudança real ligada a `index.Replace`, reconciliação por overflow, correlação de rename por `xxhash`, e os contadores em `vault_stats`.

**M2.1 em andamento — Tasks 33 a 42.** A revisão do M2 (2026-07-28) encontrou três Critical, cinco Important e um lote de higiene, cada um reproduzido por mutação ou sonda, nenhum inferido:

- `CorrelateRenames` abria anexo e placeholder somente-nuvem, furando duas regras fechadas que `index.Replace` respeita.
- O reconciliador de overflow (P0, RF-05) tinha cobertura zero: removido inteiro, o teste continuava verde.
- Link resolvia para nota deletada, por divergência de caixa na chave de `byAlias`.

Três decisões foram fechadas antes de escrever as tarefas e não devem ser re-litigadas: `--debounce-ms=0` passa a ser **recusado na config**; `index.MoveNote` **fica**, pagando as dívidas que contraiu ao entrar fora do contrato; e os contadores de descarte são **publicados desdobrados por motivo** em `vault_stats`.

**As tarefas 19 a 42 do plano são autocontidas.** Cada uma carrega, dentro da própria seção, onde encaixa, as decisões fechadas que a vinculam, as armadilhas já pagas que se aplicam, verificações além dos passos, regras de execução e contrato de relatório. O brief extraído basta para executar — não é preciso injetar contexto acumulado no prompt. Foi pensado para delegar a modelo mais barato, com revisão feita pelo modelo principal. Para o M2.1: **33, 34, 35, 37, 38, 39, 40 e 41 vão para o modelo barato**; só 36 e 42 ficam com o principal.

A 33 e a 34 mudaram de lado depois que o corpo dos testes difíceis entrou no plano como código literal — transcrição de código completo roda bem no tier mais barato, e o que as tornava caras era ter de *projetar* o teste que não podia ser enganado. A 37 mudou pelo mesmo motivo: os quatro pontos onde ela erra (mapa compartilhado em vez de atomics separados, motivo reconstruído no chamador, `active` sem `defer`, watcher importando service) estão com o código na seção. A 36 fica com o principal porque o entregável dela é escrever os oito testes que faltam, um por estrutura, e isso é projeto e não transcrição. A 42 fica porque o entregável são relatórios com evidência real, e o modo de falha de um modelo barato pedido a "escrever relatórios com evidência" é fabricá-la.

O ledger fica em `.superpowers/sdd/2026-07-25-gobsidian-v01/progress.md`. O caminho plano antigo virou ponteiro: os dois derivaram e um tinha 16 tarefas enquanto o outro tinha 6.

**`.superpowers/` é versionado.** Era ignorado por inteiro, exceto um relatório que alguém forçou uma vez — então o ledger, a única coisa que atravessa sessões, existia só na cópia de trabalho. Remover a linha do `.gitignore` não bastou: havia um segundo `.gitignore` com `*` dentro de `.superpowers/sdd/`, que o `sdd-workspace` do plugin **recria**, e que negação no diretório pai não cancela. `sdd.ps1` o apaga a cada chamada. O arquivo `task-N-base.txt` fica sujo de propósito — commitá-lo move o HEAD e a base recursa.

Lacuna registrada pra M6: no harness de órfãos atual `stdin-eof` sempre vence (100/100 nas duas últimas rodadas), então vigília do pai e sinais seguem sem verificação ponta a ponta. Falta cenário em que stdin fica aberto e pai morre.

