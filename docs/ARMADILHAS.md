# ARMADILHAS.md — o que já custou caro neste projeto

Cada item aqui passou por revisão e só apareceu depois. Estão registrados para
não voltarem. **Regra nova entra com o defeito concreto que a originou** —
regra sem história é preferência, e preferência não merece espaço aqui.

Índice:

- [Ciclo de vida e encerramento](#ciclo-de-vida-e-encerramento)
- [Acesso a arquivo e confinamento](#acesso-a-arquivo-e-confinamento)
- [Índice, chaves derivadas e consistência](#índice-chaves-derivadas-e-consistência)
- [Watcher](#watcher)
- [Daemon e IPC](#daemon-e-ipc)
- [Medição e benchmark](#medição-e-benchmark)
- [Contratos de API](#contratos-de-api)
- [Ferramentas deste ambiente](#ferramentas-deste-ambiente)

---

## Ciclo de vida e encerramento

**`io.TeeReader` não propaga EOF.** Copia bytes; EOF não é byte. Usar
`mirrorReader`, que faz `dst.CloseWithError(err)`. Sem isso o monitor de stdin
do lifecycle fica inerte e `lc.Wait()` só retorna por acidente. O espelho é
**auxiliar**: falha de escrita nele não pode virar erro da leitura principal —
mataria sessão saudável por motivo que o cliente não pode agir.

**Vigília do pai precisa de `exitTime`, não só creation time.** O Windows
mantém PID e creation time consultáveis muito tempo depois da morte do
processo. Comparar `(pid, created)` nunca detecta pai morto — deixou 5 de 5
órfãos no primeiro teste ponta a ponta. Em Unix, comparar o ppid capturado no
startup, não a constante 1: sob Docker+tini, systemd ou s6 o reaper não é PID 1.

**Goroutine parada em `Read` não é desenrolável por cancelamento de context.**
Por isso `watchStdin` fica fora do `WaitGroup` — incluí-la trava `Wait()` quando
sinal ou pai dispara primeiro. Vigias de sinal e de pai entram, porque fazem
`select` em `ctx.Done()`.

**`ctx.Canceled` no retorno do serve loop é encerramento normal.** Duas
detecções de EOF independentes correm — SDK e lifecycle — e qual vence decide o
valor. Tratar como falha faz o host ver erro aleatório a cada desconexão limpa.

---

## Acesso a arquivo e confinamento

**Confinamento de caminho tem duas camadas.** `validateLocal` (léxica: NUL,
`..`, raiz, `IsLocal`, regra de plataforma) e `Canonicalize` (por componente,
via `filepath.Rel`). `filepath.IsLocal` barra nome de dispositivo —
`Resolve(root, "COM1")` escrevia em porta serial antes disso. Mas a regra de
ponto/espaço no fim de componente vale **só no Windows**: em Linux `Notas ` é
nome legal, e rejeitá-lo lá torna notas reais inalcançáveis.

**`CanonicalPath` não garante grafia do disco.** Esta camada não consulta disco;
preserva o que o chamador passou. Quem produz grafia real é `vault.Walk`.

**`d == nil` no callback do `WalkDir` é falha na própria raiz** — cofre
desmontado, share caído. Devolver `nil` ali faz a varredura reportar sucesso com
zero entradas, e o servidor afirma com confiança que o cofre está vazio. Cofre
inacessível e cofre vazio não podem produzir a mesma resposta.

**Placeholder de nuvem chega ao índice sem `note`, e o coletor desreferenciava.**
`build.go` manda `parsed{entry: e}` com `note` nil para anexo e para arquivo
somente-nuvem, de propósito: ler dispararia download síncrono. `insert` entrava
no ramo `r.entry.IsNote` — verdadeiro, é `.md` — e lia `r.note.Title`. Um único
`.md` não hidratado do OneDrive derrubava `index.Build` no boot.
**`FILE_ATTRIBUTE_OFFLINE` é gravável por `SetFileAttributes` e
`vault.IsCloudOnly` também o aceita** — é assim que se monta o caso em teste.

**O boot do índice de busca contornava a guarda que ele deveria respeitar.**
`Inverted.Update` tem a guarda de nuvem e o comentário certo — *"mora aqui, e
não nos chamadores, porque esta é a função que abre o arquivo"*. Mas
`buildInvertedIndex` não passava por ela: fazia `v.ReadAll` (um `os.ReadFile`
puro, sem consulta a `CloudOnly`) e `inv.Add` direto. Como `.md` somente-nuvem
**é** nota do índice de metadados, `idx.NotePaths()` devolvia placeholders, e um
cofre OneDrive sem cache válido baixava o cofre inteiro em segundo plano. Havia
**duas construções do mesmo índice**, uma com guarda (o CLI) e outra sem.
Agravante: o teste que "provava" o boot seguro usava um dublê chamado
`construirComoOBoot` que chamava `Update`, afirmando em comentário ser
"exatamente como buildInvertedIndex faz" — não era. **Teste que afirma sobre uma
reimplementação não afirma sobre produção.** Corrigido em 2026-08-26: o boot
chama `Update`, e o teste novo exercita `buildInvertedIndex` de verdade.

**Anexo era lido inteiro pelo índice de busca, e isso corrompia o ranking.**
`Inverted.Update` guardava só contra nuvem; todo `.png` ou `.mp4` seguia para
`os.ReadFile` e tokenização binária. O dano não era I/O: medido, um anexo
contribuía `DocLength=4`, e **`DocLength` é o divisor da normalização por
tamanho do BM25**. Pior, o índice do boot indexava só notas enquanto o mantido
por eventos indexava anexo também — duas construções respondendo diferente, a
mesma família do defeito de `DocLength`. Na reconciliação por overflow era pior
ainda: o atalho de mtime/tamanho consulta `idx.Get`, que resolve só notas, então
todo anexo falhava no atalho e era **re-lido a cada varredura**. Corrigido em
2026-08-26 com guarda de classe dentro de `Update`.

**Confinamento léxico não alcança symlink, e a camada que abria o arquivo não
exercia checagem nenhuma.** As duas camadas — `validateLocal` e `Canonicalize` —
são léxicas de propósito, e `path.go` documenta o limite delegando à "camada que
abre o arquivo". Essa camada era um `os.Open` puro. `cofre/nota.md ->
C:\qualquer\coisa.txt` passava nas duas, entrava no índice, e `note_read`
devolvia conteúdo arbitrário pelo canal MCP. **O RNF-32 estava publicado como
"Atingido"** com um único teste, o de symlink de *diretório* — que `WalkDir`
nunca atravessou. O de *arquivo* nunca teve teste e nunca funcionou. Corrigido
em 2026-08-26: `Walk` pula com descarte **registrado**, as três aberturas
recusam, e `--follow-symlinks` religa o comportamento antigo.

**`os.Root` foi implementado, medido e REJEITADO no mesmo dia** — não é dívida
esquecida, é decisão tomada. Ele funciona: `r.Open("link.md")` e
`r.Open("../fora.txt")` devolvem `path escapes from parent`, imposto pelo
runtime, e a janela TOCTOU fecha. O que o derrubou foi um custo que o plano não
previa. O plano mandava medir "syscalls extras por nota"; o bloqueio foi tempo
de vida de handle:

| Arranjo | Latência de leitura | Custo |
|---|---|---|
| `os.ReadFile` (hoje) | 155,2 µs | TOCTOU aberto |
| Root **cacheado** | 155,1 µs (idêntica) | **trava a pasta do cofre** |
| Root **por leitura** | 252,7 µs (**+63%**) | — |

Com o descritor da raiz aberto, o Windows recusa renomear, mover e apagar a
pasta do cofre enquanto o servidor roda — medido, não inferido. Numa ferramenta
de notas isso é regressão visível. E abrir o Root a cada leitura custa +63% num
caminho quente: `GenerateSnippet` abre um arquivo por hit, e é 21% do CPU da
busca. Reabrir exige caso novo — por exemplo, um vetor de escape que a guarda
`Lstat` não pegue.

**Camada de pré-filtro que abre arquivo derrota o filtro que ela precede.**
`CorrelateRenames` roda antes de `index.Replace` e chamava `vault.ReadAll` em
todo caminho do lote — anexo inclusive, placeholder de nuvem inclusive.
**Quem roda antes do guarda precisa do mesmo guarda.**

**Conserto que remove uma parada abrupta abre caminho que ninguém nunca
executou.** O fix do pânico de `index.Build` com placeholder de nuvem tornou o
boot possível — e, com ele, tornou alcançável
`NotePaths() -> Inverted.Update -> os.ReadFile`, que baixa todo placeholder do
cofre em segundo plano. Antes o processo morria antes de chegar lá. Depois de
remover um `panic`, um `Fatal` ou um `return` de erro que abortava cedo, a
pergunta é **o que passa a rodar agora que antes não rodava** — e a resposta se
acha percorrendo os chamadores a jusante, não relendo a função consertada.

---

## Índice, chaves derivadas e consistência

**Chave de mapa calculada em dois lugares diverge, e a divergência só aparece no
caminho menos usado.** `byAlias` era escrito minúsculo por `alias.go` no boot e
cru por `Replace`; `resolve.go` lia minúsculo. Enquanto o índice só era
construído no boot, os dois concordavam. Quando o watcher tornou `Replace` e
`Remove` alcançáveis, `Remove` passou a procurar uma chave que não existia:
`[[STJ]]` continuava resolvendo, com `state=ok`, para uma nota que já tinha
saído do índice. Toda chave derivada passa por **uma** função — `aliasKey(alias)`
— e todo acesso passa por ela, inclusive os que já estavam certos. Não é para
consertar os errados: é para tornar a próxima divergência impossível sem tocar
na função.

O mesmo padrão já apareceu em: `nomeChave` (resolução), a detecção de EOL
(`writer` × `vault`), o caminho do log do daemon (três contas, consolidadas em
`daemon.CaminhoDoLog` em 2026-08-26), e a versão do formato de cache de
metadados (duas constantes independentes guardando o mesmo portão).

**Índice reconstruído e índice recarregado do cache têm de responder igual.**
`DocLength` era derivado na leitura, somando as posições de cada termo — e um
token cuja forma reduzida difere da raiz entra em **duas** postings. Um
documento de 5 tokens que todos reduzem media 5 recém-construído e 10
recarregado. `DocLength` é o divisor da normalização por tamanho do BM25: o
mesmo cofre ranqueava diferente conforme o servidor tivesse acabado de indexar
ou de ler o cache, sem log e sem teste falhando. O que prova isso não é conferir
um valor escrito à mão — é **comparar as duas construções campo a campo**.

Junto veio o irmão: nota sem token nenhum não entrava em `docLengths`, logo não
contava em `DocCount`, logo o cabeçalho do cache declarava menos notas do que o
índice de metadados via, e **todo** boot concluía "cache parcial" e regravava o
cache inteiro.

**Reparar metade do estado é pior que não reparar.** A reconciliação por
overflow (RF-05, P0) reparava o índice de metadados e deixava o de busca
obsoleto. Como `service.Search` descarta a posting cujo caminho não está nos
metadados, uma nota movida durante o overflow devolvia **zero resultados** para
sempre. Teste de mecanismo que cruza estruturas afirma sobre o que o usuário
veria, não sobre cada estrutura em separado.

---

**Duas constantes independentes guardando o mesmo portão concordam por
coincidência, e divergir não quebra nada visível.** O cache de metadados era
conferido em dois lugares: `IndexCacheFormatVersion` em `LoadIndexCache`, e
`indexCacheCodecVers` no cabeçalho, dentro do decodificador. Tinham o mesmo
valor porque alguém digitou o mesmo número duas vezes. Subir **uma** — o que
qualquer mudança de layout exige — não quebra build nem teste: faz o leitor
recusar todo save que o próprio processo acabou de gravar, com **reconstrução
completa a cada boot e nenhum log dizendo por quê**. O sintoma parece
"cache lento", não "cache quebrado". Fechado em 2026-08-26 (achado B11) com
`indexCacheCodecVers = IndexCacheFormatVersion` — alias, não cópia, para que o
bump seja impossível de fazer pela metade. O teste que pega isso é um
round-trip `Save`→`Load`; ele não existia, e é o que faltava para o portão ter
dono.

---

**Teto de tamanho que não cabe no tipo do índice estoura em silêncio.**
`limitePosicoes` valia `4_000_000_000`, quase o dobro de `math.MaxInt32`, e os
índices dentro das fatias que ele dimensiona são `int32` (`termIni`, `postIni`,
`postPath`). Acima do teto, `int32(kPos)` dá a volta **sem erro**: o cache
decodifica "com sucesso" e serve as posições de outro termo. A mesma constante
dimensionava `make([]TokenPosition, totPos)` — 16 bytes cada — em **64 GB** a
partir de um cabeçalho adulterado, antes de qualquer verificação do corpo.
Fechado em 2026-08-27 (achado B1). A guarda é de **compilação**: converter uma
constante negativa para `uint` não compila, então um teto acima de `MaxInt32`
quebra o build naquela linha. Está no código-fonte, e não num teste, porque
teste que ninguém rodou não impede o commit.

**Duas funções com o mesmo nome em pacotes diferentes convidam a semânticas
diferentes.** `writer.DetectEOL` e `vault.DetectEOL` respondiam a mesma pergunta
por regras distintas: a primeira chamava o arquivo inteiro de CRLF se houvesse
QUALQUER quebra CRLF, a segunda usa o estilo PREDOMINANTE — e é a resposta da
segunda que o índice persiste em `Note.EOL`. Um arquivo com uma linha CRLF e mil
LF era LF para o índice e CRLF para a escrita. As duas concordavam na maioria dos
arquivos reais, que é o que fez isso sobreviver: **contas que divergem só na
borda são as que ninguém percebe**. Fechado em 2026-08-27 (achado M14);
`writer.DetectEOL` delega.

---

## Watcher

**A falha na raiz da varredura de diretório novo era engolida — a mesma
armadilha do `d == nil`, num segundo lugar.** `vault.Walk` distingue "raiz
ilegível" de "entrada ilegível" desde cedo (`walk.go:133`); `varreDiretorioNovo`
tratava as duas igual, logava e devolvia `nil`. Uma varredura que não conseguiu
ler a raiz reportava **sucesso com zero entradas** — indistinguível de "a
varredura não rodou", e produzindo exatamente o estado que a função existe para
impedir: um evento recebido, índice vazio, notas invisíveis até o próximo
reinício. No Windows a causa transitória clássica é o antivírus segurando o
diretório recém-movido. Corrigido em 2026-08-26: a raiz sobe erro, a entrada
individual continua sendo logada e pulada, e o `Run` **não morre** — agenda
reconciliação, que é o anteparo que já existia para eventos perdidos. A conta do
agendamento passou a ser uma só (`agendaReconciliacao`), partilhada com o
overflow.

**Watch em diretório novo não vê o que já está dentro dele.** Uma pasta que
chega ao cofre com arquivos entrega **um** evento — a criação do diretório — e
nenhum arquivo, porque eles existiam antes de o watch existir. Medido: 3 notas,
1 evento, 0 indexadas, invisíveis para todas as tools até o próximo reinício. É
o usuário arrastando uma pasta. Era também o que fazia `note_move` perder a
nota. **Todo `Add` de watch precisa ser seguido de varredura.**

**Feature P1 não tem direito de apagar dado P0.** O campo inline do Dataview
consumia o span do valor, e `fonte:: [[STJ]]` deixava de produzir link nenhum —
links que o commit anterior já coletava. Quando uma feature opcional muda o que
uma obrigatória já entregava, o A/B contra o commit anterior é o que revela.

---

## Daemon e IPC

**`.sock` órfão pode envenenar toda partida seguinte, para sempre.** Medido em
2026-08-26 na máquina do dono: três sessões MCP caíram para o modo em processo
em **toda** partida, ao longo de dias, pagando 10 s cada uma. Depois de remover
os arquivos órfãos, a mesma partida decidiu em **559 ms**.

**Não classifique estado de socket por errno.** Medido com
`net.Dial("unix", ...)` no Windows:

| Estado do caminho | errno |
|---|---|
| arquivo comum | `10061` ECONNREFUSED |
| socket órfão de dono morto à força | `10061` |
| caminho inexistente | `10061` |
| **diretório** | **`10022`** EINVAL |

Errnos diferentes descrevem o mesmo estado, e o mesmo errno descreve estados
diferentes. **O critério de "há daemon vivo" é handshake bem-sucedido**, nunca o
erro do dial. Conectar também não basta: um daemon com o laço de `Accept` morto
aceita no backlog do SO e nunca responde.

**Daemon que morre calado é indistinguível de daemon que não nasceu.** Dois
daemons registraram `"daemon iniciado"` e nada mais; a causa existia
(`vault.New` recusa caminho inexistente com mensagem específica) e nunca
chegava a lugar nenhum, porque o stderr de um processo detachado não tem leitor.
Todo caminho de saída loga a causa **antes** de sair. E log **ausente** contra
log **com uma linha** são dois defeitos diferentes: o primeiro é falha de spawn,
o segundo é morte na montagem.

**`StreamReader.Peek()` bloqueia.** Não é consulta sem espera, apesar do nome. O
harness o usava para ler o stderr da ponte, e o teste de prazo logo acima nunca
era alcançado: um ciclo ficou **15h44m** parado com o daemon vivo. Use
`ReadLineAsync` com `Wait` limitado. Gate que pode travar indefinidamente vira
gate que se aprende a pular.

**O daemon tem uma corrida residual conhecida.** O lock de inicialização
serializa quem disputa no mesmo instante, não quem chega atrasado: medido, dez
pontes sob carga produziram **dois daemons vivos** para o mesmo cofre.
Registrado nos limites conhecidos de `docs/OPERACAO.md`.

**`--cache-dir` explícito não é namespaced por cofre.** `defaultCacheDir` deriva
`.../gobsidian/<VaultKey>`, mas um caminho explícito é usado literal e os nomes
de arquivo são fixos. Dois cofres no mesmo diretório recusam o cache um do outro
(o cabeçalho guarda `VaultPath`) e **rebuildam a cada boot, sem aviso**.

---

## Medição e benchmark

**Cache ligado no harness que mede latência transforma o requisito em outra
pergunta.** Os laços de RNF-04 repetem a MESMA consulta 30 vezes, e `b.Loop` faz
o mesmo — então um cache de trecho ligado acerta 29 das 30, e o p95 passa a
descrever a consulta repetida, que nenhum usuário vê na primeira busca. Medido:
`limit: 200` deu 21,49 ms quente contra 25,98 ms frio. Todo harness de latência
desliga o cache por `semCacheDeTrecho`; quem mede repetição é um benchmark com
"Repetido" no nome.

**`GOGC` foi testado duas vezes e rejeitado nas duas** — não re-litigar sem dado
novo. `GOGC=off` deu `~ (p=0,093, n=6)`; `GOGC=400` deu −28,51% no benchmark mas
não significativo no boot real (12 partidas por braço, U de Mann-Whitney 88
contra região crítica 37/107) e com RSS pior. O que pagou foi
`debug.FreeOSMemory()` depois de o índice ficar pronto: −195 MB.

**Órfão vazado e ciclo que não mediu são coisas diferentes.** O harness contava
os dois como falha, e um commit **só de documentação** reprovou com 1 ciclo em
300 e os outros 299 limpos. Hoje `-MaxNaoMedidosPct` (padrão 2) tolera uma
fração, **sempre impressa**, e três coisas continuam reprovando com qualquer
teto: vazamento real, `reason=` errado, e **zero ciclos medidos**.

**Pipe engole código de saída.** `cmd | tail` devolve o status do `tail`, não do
`cmd`. Foi reportado `CI_EXIT=0` para uma rodada de CI que tinha **reprovado**.
Redirecione para arquivo e leia o `$?` do comando. Vale também para
`run_in_background` com `| tail`. <!-- check-doc-refs: ignore run_in_background -- parametro da ferramenta Bash do Claude Code, nao identificador deste codebase -->

**Não rode gate concorrente com medição.** O gate de órfãos rodando em paralelo
com medições teve seus processos mortos por uma rotina de limpeza, o que
produziria **falso verde** — o harness procura sobrevivente e não acha nenhum
quando outro mata por ele.

---

**Guarda que não pode mudar o resultado parece proteção e não é.** Duas apareceram
em 2026-08-27, e as duas foram mortas pela prova de mutação, não pela leitura.
`headingDoLink` nasceu com `if start < 0 { return "" }` para tratar
`offsetUnknown`: com `start = -1` todo heading tem `Start > -1`, o laço quebra na
primeira volta e o resultado já é `""`. E a correção do achado P11 prefixava a
raiz da varredura para alcançar caminhos além de MAX_PATH: `MkdirAll`,
`WriteFile` e `WalkDir` alcançaram 318 caracteres sem ela. **A mutação é o que
distingue proteção de decoração** — nos dois casos ela devolveu EXIT=1, "o teste
passou com a regra mutada".

**Mas a segunda dessas duas ensinou outra coisa, e maior: a prova de mutação vale
CONTRA O AMBIENTE em que rodou.** Descobriu-se em 2026-08-31 que a máquina tem
`LongPathsEnabled = 1` no registro do Windows, e que um caminho RELATIVO de 327
caracteres também passa — o `fixLongPath` do Go só trata caminho absoluto, então
quem respondia ali era o registro, e não o Go. A guarda pode não ser morta numa
máquina com a chave desligada. **EXIT=1 diz "este teste, nesta máquina, não
distingue"; não diz "a regra é inútil".** Quando a regra depende de configuração
da máquina, a prova de mutação precisa trazer a configuração ao lado do
resultado.

**RSS não mede quanto dado você guarda — mede a META de heap do GC.** O Go fixa a
meta ao fim de cada ciclo em ~2× o heap vivo daquele instante, e o RSS de um
processo em repouso é aproximadamente a meta que vigorava quando o último ciclo
terminou. Em 2026-08-27 isso produziu um resultado que parecia impossível: o
binário **sem** o campo `Context` consumia **3,6 MB a mais** de RSS que o binário
com ele, de forma reproduzível e com faixas disjuntas. `GODEBUG=gctrace=1` mostrou
a causa: os dois terminaram com heap vivo igual (15 contra 16 MB), o braço sem
contexto rodou um ciclo de GC a mais, disparado mais tarde e de um heap
marginalmente maior, e fixou meta de 33 MB contra 31 — o RSS seguiu a meta. No
cofre onde o campo acrescenta 6 MB de heap vivo, o efeito aparece limpo e na
direção esperada.

**E `runtime.MemStats.Alloc` não é heap vivo** — é `HeapAlloc`, heap vivo MAIS o
lixo ainda não coletado. Lê-lo como "quanto o índice ocupa" inverteu o sinal de uma
comparação nesta mesma sessão. Para heap vivo, o que serve é o número do meio no
`gctrace` (`antes->pico->vivo`), ou `runtime.GC()` seguido de `ReadMemStats`.

**Comparar duas variantes de desempenho em bateladas sequenciais mede a deriva da
máquina, não a diferença entre elas.** Em 2026-08-26 isto produziu **dois** números
errados na mesma sessão, publicados e depois retratados. O segundo dizia que o
contexto do backlink empurrava o RNF-02 de atingido para não atingido: medianas de
243 ms contra 323 ms, com as faixas quase disjuntas — parecia sinal forte. Refeito
com os binários construídos lado a lado e **uma rodada de cada por vez**, as três
variantes ficaram em 179 / 193 / 191 ms, com a diferença entre elas MENOR que a
variação dentro de qualquer uma.

**Alternar a ordem das bateladas não corrige isso** — foi o cuidado que eu tinha
tomado, e não bastou: as bateladas continuam separadas no tempo, e o cache de
arquivo do SO aquecendo, outro processo ou throttling térmico atingem cada uma de
forma diferente. O que corrige é **intercalar as execuções**, uma de cada variante
por vez, com um binário por variante construído antes de começar.

Sinal de alerta específico: se a variante medida PRIMEIRO for sistematicamente a
mais lenta, suspeite do cache de arquivo do SO antes de acreditar no resultado. O
primeiro boot contra um cofre de 1,3 GB mediu 4.534 ms; a repetição com o SO
quente, 765 ms.

---

## Contratos de API

**Handler que devolve `error` Go faz o SDK montar `IsError` sem
`StructuredContent`.** Devolver resultado de erro com `Out` zerado manda
`{"notes":0,...}` junto, e o cliente não distingue falha de cofre vazio no canal
que ele lê primeiro.

**Schema que promete e código que ignora é pior que parâmetro ausente.**
`note_list` declarava `fields` no schema e o descartava. O modelo do outro lado
pede três campos, recebe tudo, e não tem como saber que o pedido não fez nada —
o schema é justamente o que ele lê para decidir. Ou implemente, ou tire do
schema e da documentação.

**Campo de API com valor fixo mente sempre.** `alias_collisions` era
`Collisions: 0` literal. Aparecia na resposta e nunca foi verdade.

**E o valor fixo não precisa ser literal para mentir: basta não haver um lugar
onde escrevê-lo.** `Backlink.Context` existia no tipo, era documentado no próprio
campo como "texto ao redor da referência", e `docs/TOOLS.md` §`vault_backlinks`
**promete** que "backlinks traz origem e o contexto textual ao redor de cada
referência". Chegava vazio em toda resposta, desde sempre. A causa: **três**
construções de `Backlink` — `backlinks.go`, e duas em `update.go` — cada uma
escrevendo `Context: ""` à mão. Não havia um lugar para preencher; havia três
para esquecer. Corrigido em 2026-08-26 (achado A8) com `backlinkDe`, construtor
único, e `montarLinks`, único ponto que monta `[]ResolvedLink`.
**A normativa estava certa e o código é que mentia** — foi a doc que denunciou o
defeito, não o contrário.

**Comentário que descreve uma limitação some quando ela é removida; se não some,
vira desinformação com autoridade.** `parser/types.go` avisava que os offsets
`Start`/`End` "só são preenchidos para LinkWiki e LinkEmbed" e que link Markdown
fica em `offsetUnknown`, mandando checar `Kind` antes de reescrever. Sondado em
2026-08-26: **as quatro grafias trazem span correto**, embed Markdown incluído.
Era verdade quando foi escrito, deixou de ser, e o texto ficou desviando quem o
lesse de um caminho que já funcionava (achado B14). A armadilha real é outra e
continua: **`Raw` não cobre o mesmo trecho que `Start:End`** nas grafias
Markdown — nelas `Raw` traz só o destino. Quem reescreve deve fatiar por
`Start:End`; quem casa por `Raw` erra em Markdown.

**Erro de segurança usado como erro genérico ACUSA o cliente.**
`LinkGraph` e `NoteMetadata` embrulhavam qualquer falha de `ResolvePath` como
`PATH_OUTSIDE_VAULT`, inclusive nota que simplesmente não existe. O host lê esse
código como tentativa de escapar do cofre: errar um nome de nota passava a
acusar quem chamou de algo que ele não fez. Pior, `ResolvePath` **não verifica
confinamento** — ela só procura no índice —, então o código afirmava algo que a
função não tinha como saber. Fechado em 2026-08-27 (achado M2) com o sentinela
`index.ErrPathNotFound` e **uma** função de classificação,
`service.ErroDeResolucao`: havia seis chamadores e três respostas diferentes
para a mesma falha.

**Flag booleana ou inteira não distingue "omitida" de "definida com zero".**
`config.Flags` tem companheiros `ReadOnlySet` e `DebounceMSSet`. **Toda** chamada
a `config.Load` precisa preenchê-los com `cmd.Flags().Changed(nome)` — esquecer
em um subcomando faz a flag virar no-op silencioso. Vale também para
`service.Options.SnippetCacheEntries`, que é `*int` pelo mesmo motivo: nil usa o
padrão, zero explícito desliga o cache.

---

## Ferramentas deste ambiente

**`bash` no PATH é o do WSL.** Ele não enxerga `C:/Users/...` e responde
`No such file or directory` para um arquivo que existe. Use o bash do Git, e
converta `\` para `/` antes de passar caminho.

**Here-string do PowerShell (`@'...'@`) não funciona na ferramenta Bash**, e
backtick de continuação também não. Para multilinha ali, use heredoc.

**PowerShell: array de um elemento vira escalar.**
`$x = if ($c) { @() } else { @('-race') }` desenrola para string, e `@x` a
espalha caractere a caractere. Tipe explicitamente: `[string[]]$x = ...`.

**Script Python que edita arquivo versionado precisa de `newline=""` na leitura
E na escrita.** Modo texto converte o arquivo inteiro para CRLF no Windows, e o
`gofmt` reprova um `.go` que estava perfeitamente formatado. Custou dois commits.

**`str.replace` que não casa não falha — segue em silêncio.** Duas edições de
plano "deram certo" sem editar nada, e o ledger ficou duas tarefas desatualizado.
Toda edição por script leva `assert` do texto-âncora antes de substituir, e
conferência do resultado no disco depois. **Deleção por número de linha é pior
ainda**: um `while linha != '}'` parou no primeiro fecho interno e quebrou um
arquivo inteiro (2026-08-26).

**Ferramenta que reescreve `.md` pode gravar em cp1252.** Os docs deste projeto
são em português. Depois de qualquer reescrita:

```bash
python -c "open('ARQUIVO.md',encoding='utf-8').read()" && echo "[OK] UTF-8 valido"
```

Reparo de arquivo misto: decodificar byte a byte, tentando sequência UTF-8
válida primeiro e caindo pra cp1252 onde falhar. Transcodificar o arquivo
inteiro de uma vez duplica os acentos das regiões que já eram UTF-8.

**Dois agentes na mesma worktree colidem, e o estrago não fica na worktree.**
Três incidentes numa sessão: um `git add` de caminho explícito recolheu trabalho
não commitado de outro agente; um `Stop-Process -Name gobsidian -Force` de
rotina de limpeza matou **a sessão real de Claude do usuário**, porque não
filtrava por cofre; e o gate de órfãos rodando em paralelo com medições teve
processos mortos por essa limpeza. Regras: `git diff <caminho>` antes de
`git add <caminho>`; **matar sempre por PID que você mesmo lançou, nunca por
nome**; e não rodar gate concorrente com medição.

**`golangci-lint` local verde não significa CI verde.** O `go.mod` declara
`go 1.25.0`, e um binário compilado com Go mais antigo recusa o config antes de
analisar linha nenhuma. O CI fixa `v2.12.2` de propósito. Confira
`golangci-lint version` antes de confiar num zero.

**O extrator de briefs vaza até o fim do arquivo na última tarefa.** O `awk` do
`task-brief` só liga e desliga em cabeçalho casando `^#+[ \t]+Task[ \t]+[0-9]+`;
nenhum outro cabeçalho o interrompe. Medido: um brief saiu com 33.421 bytes
contra 3.625–4.425 das irmãs. Por isso existe a sentinela `# Task 000` no fim do
plano — **mova-a para depois da última tarefa** ao acrescentar tarefas.
