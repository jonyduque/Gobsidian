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
pwsh -File scripts/test_orphans.ps1 -Cycles 100    # os tres mecanismos; -Scenario isola um
golangci-lint run ./...          # exige v2.12.2; v1.x nem carrega o config

pwsh -File scripts/mutate.ps1 -Path <arquivo> -Anchor '<texto>' -Replacement '<texto>' -Test <Nome> -Package ./internal/x/
pwsh -File scripts/audit_reports.ps1 [N]    # relatórios e ledger contra evidência falsa
```

`mutate.ps1` tem **código de saída invertido de propósito**: `0` = o teste reprovou sob mutação (regra verificada), `1` = o teste passou (regra escrita, não verificada), `2` = inconclusivo (âncora ambígua, ou a mutação quebrou o build — falha de compilação não é cobertura). Ele exige âncora com ocorrência única, restaura em `finally` e confere o restauro por SHA-256. Detalhe na skill `mutation-proof-discipline`.

`audit_reports.ps1` procura hedge apresentado como medição, prova de mutação escrita no condicional, não-resposta do tipo "coberto implicitamente", SHA citado no ledger que não existe, e tarefa completa sem relatório. Sai `1` com achados. Não julga conteúdo — localiza a frase para alguém conferir.

`check_doc_refs.ps1` acha token entre crases que parece identificador de código e não existe em `.go` nenhum. `check_readme_anchors.ps1` confere que toda âncora do README resolve e que toda seção H2 é alcançável pela navegação. **Os dois entraram na bateria em 2026-08-11 porque até então não rodavam em lugar nenhum** — nem no `verify.ps1`, nem no CI. Três seções do README ficaram sem link por um marco inteiro por causa disso.

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

**Índice reconstruído e índice recarregado do cache têm de responder igual.** `DocLength` era derivado na leitura, somando as posições de cada termo — e um token cuja forma reduzida difere da raiz entra em **duas** postings. Um documento de 5 tokens que todos reduzem media 5 recém-construído e 10 recarregado. `DocLength` é o divisor da normalização por tamanho do BM25: o mesmo cofre ranqueava diferente conforme o servidor tivesse acabado de indexar ou de ler o cache, sem log e sem teste falhando. O que prova isso não é conferir um valor escrito à mão — é comparar as duas construções campo a campo. Junto veio o irmão: nota sem token nenhum não entrava em `docLengths`, logo não contava em `DocCount`, logo o cabeçalho do cache declarava menos notas do que o índice de metadados via, e **todo** boot concluía "cache parcial" e regravava o cache inteiro.

**Reparar metade do estado é pior que não reparar.** A reconciliação por overflow (RF-05, P0) reparava o índice de metadados e deixava o de busca obsoleto. Como `service.Search` descarta a posting cujo caminho não está nos metadados, uma nota movida durante o overflow devolvia **zero resultados** para sempre. Foi o segundo buraco no mesmo requisito: o primeiro era cobertura zero, este era cobertura que afirmava sobre **um** dos dois índices. Teste de mecanismo que cruza estruturas afirma sobre o que o usuário veria, não sobre cada estrutura em separado.

**Watch em diretório novo não vê o que já está dentro dele.** Uma pasta que chega ao cofre com arquivos entrega **um** evento — a criação do diretório — e nenhum arquivo, porque eles existiam antes de o watch existir. Medido: 3 notas, 1 evento, 0 indexadas, invisíveis para todas as tools até o próximo reinício. É o usuário arrastando uma pasta. Era também o que fazia `note_move` perder a nota: a tool cria o destino e renomeia para dentro em seguida, e quando o rename vence o registro do watch, o arquivo novo nunca é visto **enquanto a remoção do antigo é**. Todo `Add` de watch precisa ser seguido de varredura.

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

**M2.1 completa — Tasks 33 a 42.** A revisão do M2 (2026-07-28) encontrou três Critical, cinco Important e um lote de higiene, cada um reproduzido por mutação ou sonda, nenhum inferido:

- `CorrelateRenames` abria anexo e placeholder somente-nuvem, furando duas regras fechadas que `index.Replace` respeita.
- O reconciliador de overflow (P0, RF-05) tinha cobertura zero: removido inteiro, o teste continuava verde.
- Link resolvia para nota deletada, por divergência de caixa na chave de `byAlias`.

Três decisões foram fechadas antes de escrever as tarefas e não devem ser re-litigadas: `--debounce-ms=0` passa a ser **recusado na config**; `index.MoveNote` **fica**, pagando as dívidas que contraiu ao entrar fora do contrato; e os contadores de descarte são **publicados desdobrados por motivo** em `vault_stats`.

**As tarefas 19 a 42 do plano são autocontidas.** Cada uma carrega, dentro da própria seção, onde encaixa, as decisões fechadas que a vinculam, as armadilhas já pagas que se aplicam, verificações além dos passos, regras de execução e contrato de relatório. O brief extraído basta para executar — não é preciso injetar contexto acumulado no prompt. Foi pensado para delegar a modelo mais barato, com revisão feita pelo modelo principal. Para o M2.1: **33, 34, 35, 37, 38, 39, 40 e 41 vão para o modelo barato**; só 36 e 42 ficam com o principal.

A 33 e a 34 mudaram de lado depois que o corpo dos testes difíceis entrou no plano como código literal — transcrição de código completo roda bem no tier mais barato, e o que as tornava caras era ter de *projetar* o teste que não podia ser enganado. A 37 mudou pelo mesmo motivo: os quatro pontos onde ela erra (mapa compartilhado em vez de atomics separados, motivo reconstruído no chamador, `active` sem `defer`, watcher importando service) estão com o código na seção. A 36 fica com o principal porque o entregável dela é escrever os oito testes que faltam, um por estrutura, e isso é projeto e não transcrição. A 42 fica porque o entregável são relatórios com evidência real, e o modo de falha de um modelo barato pedido a "escrever relatórios com evidência" é fabricá-la.

O ledger fica em `.superpowers/sdd/2026-07-25-gobsidian-v01/progress.md`. O caminho plano antigo virou ponteiro: os dois derivaram e um tinha 16 tarefas enquanto o outro tinha 6.

**`.superpowers/` é versionado.** Era ignorado por inteiro, exceto um relatório que alguém forçou uma vez — então o ledger, a única coisa que atravessa sessões, existia só na cópia de trabalho. Remover a linha do `.gitignore` não bastou: havia um segundo `.gitignore` com `*` dentro de `.superpowers/sdd/`, que o `sdd-workspace` do plugin **recria**, e que negação no diretório pai não cancela. `sdd.ps1` o apaga a cada chamada. O arquivo `task-N-base.txt` fica sujo de propósito — commitá-lo move o HEAD e a base recursa.

**Formato do cache de busca é o 6, e não é `gob`** (formato 5 em 2026-08-03/04; formato 6 na Task 89). A extensão do arquivo continua `.gob` por compatibilidade de caminho; o conteúdo é um codec binário próprio em `internal/search/persist_codec.go`. O índice virou duas camadas: base imutável em arrays achatados vinda do cache (`soa.go`) mais um delta em mapas com o que mudou desde a partida. O formato 6 acrescentou uma arena mapeada em memória (`mmap.go`), que é o que permite várias instâncias compartilharem as páginas do índice. Medido no cofre real de 3.152 notas na virada do 4 para o 5: carregamento **5,59 s → 659 ms**, arquivo **482 → 67 MB**, boot quente **~7 s → 842 ms**. **Toda troca de formato reconstrói o cache de todo cofre no boot seguinte**, em segundo plano, com as outras onze tools respondendo desde o primeiro segundo — e isso vale para quem atualizar de uma v1.0.x, que grava `gob` e invalida o cache desta versão a cada alternância.

`GOGC` foi testado duas vezes e rejeitado nas duas — não re-litigar sem dado novo. `GOGC=off` deu `~ (p=0,093, n=6)`; `GOGC=400` deu −28,51% no benchmark mas não significativo no boot real (12 partidas por braço, U de Mann-Whitney 88 contra região crítica 37/107) e com RSS pior. O que pagou foi `debug.FreeOSMemory()` depois de o índice ficar pronto: −195 MB.

**O gate de órfãos cobre os quatro cenários, e o padrão roda os quatro.** `scripts/test_orphans.ps1 -Cycles 100` executa `stdin-eof`, `parent-death`, `signal` e `daemon-idle` em sequência, e cada um **reprova se o `reason=` não for o do mecanismo que ele nomeia** — encerrar pelo motivo certo por acidente não conta. `parent-death` desconecta o EOF (cadeia keeper → host → servidor, com o keeper segurando a ponta de escrita do pipe); `signal` deixa tudo vivo e só manda CTRL_BREAK; `daemon-idle` está detalhado adiante, junto com as armadilhas do próprio harness.

Isto esteve escrito aqui como lacuna aberta por mais tempo do que foi verdade. Os cenários existem desde 2026-08-02 e o CI chama os três explicitamente (`.github/workflows/ci.yml`), mas o padrão do script era `stdin-eof` — então quem rodava o comando documentado localmente via `[OK]` na tela depois de exercitar **um** dos três, e concluía que os três estavam verificados. O padrão passou a ser `all` por causa disso. Gate cujo padrão cobre parte do que ele aparenta cobrir é pior que gate ausente.

`docs/PRD.md` Q3 decidiu persistir **dois** caches, e desde a Task 85 os dois existem: o cache do índice de metadados entrou, e o boot com cache válido caiu de 1.192–1.396 ms para 371–472 ms num cofre real. A linha que esteve aqui dizendo que `index_cache.gob` "nunca foi implementado" ficou falsa nesse dia e sobreviveu a várias sessões.

**M7 completa — Tasks 78 a 93**, em duas partes. A Parte I (78–87) atacou a busca: `sync.Pool` de transformers, `TitleNorm` pré-computado, chave única `nomeChave` para resolução, e o corpus de contraste que tornou as perguntas de verificação respondíveis. Medido: busca **218,5 → 115,0 ms**, alocação **188,49 → 50,42 MiB** (−73%), **128,89k → 14,12k** allocs (−89%). A Task 82 foi **revertida**: `benchstat` deu `~`, e mudança sem ganho significativo é dívida pura.

A Parte II (88–93) atacou memória entre instâncias. Carga preguiçosa do índice de busca (`--eager-search` liga a antiga), arena mapeada em memória, o RNF-30 reformulado, o transporte IPC, e o daemon. Medido no cofre real de 4.513 notas, memória física agregada: **1 sessão** 579,1 → 244,6 → 223,6 MB; **3 sessões** 1.681,3 → 508,5 → 262,2 MB; **5 sessões** 2.916,4 → 773,4 → 229,4 MB (pré-M7 / sem daemon / com daemon). A coluna do daemon **não escala com N** — é a assinatura de um índice pago uma vez só.

**O RNF-30 mudou de redação, não de intenção** (Task 90, autorizado pelo dono em 2026-08-05). Era "nenhum socket"; é "nenhum socket que saia da máquina". `tools/netcheck` aceita `net.Dial`/`net.Listen` **apenas com a rede na constante literal `"unix"`** — rede vinda de variável é recusada. Escreva a string no lugar; guardá-la numa variável deixa o `check_net` vermelho, e isso é a regra funcionando.

**A escolha do transporte foi medida e não se re-litiga (D-M7-6).** AF_UNIX contra named pipe, ida e volta, 20.000 repetições: 25,7 contra 82,9 µs em 256 B, 23,0 contra 93,5 em 4 KB, 42,9 contra 110,0 em 64 KB. Está na biblioteca padrão e é o mesmo código nos três sistemas; build tag só para o caminho do socket e a limpeza.

**O daemon tem uma corrida residual conhecida.** O lock de inicialização serializa quem disputa no mesmo instante, não quem chega atrasado: medido no cofre real, dez pontes sob carga produziram **dois daemons vivos** para o mesmo cofre. O segundo dial depois de adquirir o lock reduziu a janela a milissegundos, mas não é exclusão mútua por construção. Registrado nos limites conhecidos de `docs/OPERACAO.md`.

**O gate de órfãos tem quatro cenários agora.** O quarto é `daemon-idle`, e ele é estruturalmente diferente dos outros três: o daemon não tem pai nem stdin de host, então a vigília do pai **não se aplica** — não a ligue por consistência. Quem substitui é a ociosidade, com padrão de 15 minutos (`daemon.DefaultIdleSeconds`); o cenário usa `--idle-seconds` curto, e esse valor não pode vazar para o padrão.

**`test_orphans.ps1` não compila — ele roda o que estiver em `bin/`.** Hoje ele recusa binário mais velho que o código e manda rodar `build.ps1`. Antes disso, um binário de quatro dias antes deu **três `[OK]`** nos cenários que não dependiam do código novo e 100 falhas de "daemon nao anunciou prontidao" no que dependia — mensagem que aponta para o daemon quando a causa era o subcomando não existir naquele build. A guarda fica **antes de qualquer despacho**: a primeira versão dela ficou junto da segunda definição de `$BinaryPath`, e o bloco de `daemon-idle` resolve o próprio caminho e **sai do script** antes de chegar lá — ela cobria três cenários e não o quarto, que é exatamente o defeito que ela existe para impedir. Só apareceu porque foi testada.

**`StreamReader.Peek()` bloqueia.** Não é consulta sem espera, apesar do nome. O harness o usava para ler o stderr da ponte, e o teste de prazo logo acima nunca era alcançado: um ciclo ficou **15h44m** parado com o daemon vivo e `ativos=1`. Use `ReadLineAsync` com `Wait` limitado. Gate que pode travar indefinidamente vira gate que se aprende a pular.

**Saída de CLI passa por `internal/console`.** Ele decide sobre cor **pelo destino**, não por `os.Stdout` global — senão `doctor > relatorio.txt` sai sujo, ou o erro no `stderr` do terminal sai sem cor. Os marcadores continuam ASCII e a cor só os reforça. `serve` não usa o pacote: stdout é do JSON-RPC. O pacote **não** se chama `utils` — a versão original dele vivia em `internal/utils/formatter.go`, que além de violar a regra de nome tinha `syscall.NewLazyDLL` sem build tag e reprovava `go vet` em linux e darwin, deixando o `check_net` cego de quebra.

**Órfão vazado e ciclo que não mediu são coisas diferentes.** O harness contava os dois como falha, e um commit **só de documentação** reprovou com 1 ciclo em 300 e os outros 299 limpos — o PID não apareceu em 20 s num runner carregado. Ciclo que não lançou não observou nada, nem sucesso nem vazamento; reprovar por causa dele mede a carga da máquina. Hoje `-MaxNaoMedidosPct` (padrão 2) tolera uma fração, **sempre impressa**, e três coisas continuam reprovando com qualquer teto: vazamento real, `reason=` errado, e **zero ciclos medidos**. Gate que reprova aleatoriamente ensina a re-rodar até ficar verde, e aí para de significar qualquer coisa.

**`check_doc_refs` dispensa por linha, não por lista global.** A diretiva é `<!-- check-doc-refs: ignore <tokens> -- <motivo> -->`, e o **motivo é obrigatório**: sem ele vira `DISPENSA-INVALIDA` e o token continua acusado. Uma lista global no topo do script dispensaria `helpers.go` em *todo* documento, inclusive num que passasse a afirmar, errado, que o arquivo existe. A dispensa mora colada à afirmação que a justifica, e as usadas saem impressas a cada rodada — lista de exceção que ninguém vê deixa de ser revisada.

**Dois agentes na mesma worktree colidem, e o estrago não fica na worktree.** Três incidentes numa sessão: um `git add` de caminho explícito recolheu trabalho não commitado de outro agente no mesmo arquivo; um `Stop-Process -Name gobsidian -Force` de rotina de limpeza matou **a sessão real de Claude do usuário**, porque não filtrava por cofre; e o gate de órfãos rodando em paralelo com medições teve seus processos mortos por essa mesma limpeza, o que produziria **falso verde** — o harness procura sobrevivente e não acha nenhum quando outro mata por ele. Regras: `git diff <caminho>` antes de `git add <caminho>`; matar sempre por PID que você mesmo lançou, nunca por nome; e não rodar gate concorrente com medição.

**Pipe engole código de saída.** `cmd | tail` devolve o status do `tail`, não do `cmd`. Reportei `CI_EXIT=0` para uma rodada de CI que tinha **reprovado**, e `EXIT=0` para um `mutate.ps1` inconclusivo. Redirecione para arquivo e leia o `$?` do comando, ou capture antes de canalizar. Vale também para `run_in_background` com `| tail`: nada aparece no arquivo até o processo terminar, porque o `tail` só escreve no fim.

**Função que lê `os.Stdin` direto não é testável de forma determinística.** `servePonteRemota` fazia isso: sob `go test` no Linux o stdin é `/dev/null` e devolve EOF na hora, então a ponte encerrava antes de o teste escrever, e o mesmo commit ficava verde no Windows e no macOS e vermelho no ubuntu. Passe `stdin` e `stdout` como parâmetros. O ganho não é só estabilidade — com o stdout na mão, o teste passou a conferir os bytes que atravessam, coisa que antes ele não fazia.

