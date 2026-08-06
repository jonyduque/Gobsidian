# PRD — gobsidian

**Produto:** servidor MCP para cofres locais do Obsidian
**Linguagem:** Go 1.25+
**Status:** rascunho para implementação
**Última revisão:** 2026-07-25

---

## 1. Contexto e problema

O ecossistema MCP para Obsidian já existe, mas é dominado por implementações em Node/TypeScript que sofrem de três patologias recorrentes.

### 1.1 Processos órfãos

Servidores MCP em Node com frequência não tratam corretamente o encerramento do host. Quando o Claude Desktop é fechado abruptamente — ou reinicia após uma atualização, ou trava —, o processo filho sobrevive. O sintoma observado em campo é um acúmulo de `node.exe` consumindo memória e mantendo *handles* de arquivo abertos sobre o cofre, o que por sua vez interfere na sincronização do OneDrive e no próprio Obsidian.

A causa raiz raramente é a linguagem. É a ausência de três mecanismos que deveriam ser redundantes: detecção de EOF em stdin, tratamento de sinais e vigilância do processo pai. Servidores maduros implementam os três; a maioria não implementa nenhum.

### 1.2 Lentidão em cofres grandes

Duas decisões erradas se repetem. A primeira é reindexar o cofre inteiro a cada evento do sistema de arquivos, o que em um cofre sincronizado por nuvem significa reindexar dezenas de vezes por minuto. A segunda é fazer busca por varredura linear em disco a cada chamada, em vez de manter índice invertido.

O resultado é que operações que deveriam custar milissegundos custam segundos, e o custo cresce linearmente com o tamanho do cofre — exatamente o inverso do desejado, já que cofres crescem monotonicamente.

### 1.3 Parsing incompleto

Markdown do Obsidian não é CommonMark. A sintaxe adicional inclui:

- Wikilinks: `[[nota]]`, `[[nota|alias]]`, `[[nota#heading]]`, `[[nota#^bloco]]`
- Embeds: `![[nota]]`, `![[imagem.png]]`, `![[nota#seção]]`
- Identificadores de bloco: `^abc123` no fim de uma linha
- Tags inline: `#tag/aninhada`
- Propriedades (frontmatter YAML) com tipos específicos do Obsidian
- Campos inline do Dataview: `chave:: valor`
- Callouts: `> [!nota] Título`

Parsers genéricos ignoram tudo isso ou o interpretam errado. Servidores MCP então improvisam extração por expressão regular, que quebra dentro de blocos de código, dentro de código inline, em links escapados e em texto com colchetes literais. O resultado é um grafo de links silenciosamente incorreto.

### 1.4 Problemas de ambiente Windows subvalorizados

Cofres reais no Windows frequentemente vivem em pastas sincronizadas do OneDrive, com caminhos longos e casing inconsistente entre diretórios. Nenhuma dessas condições é exótica, e nenhuma é tratada pelos servidores existentes. Detalhamento completo em [`WINDOWS.md`](WINDOWS.md).

---

## 2. Objetivos

**O1 — Estabilidade de processo.** Zero processos órfãos, sob qualquer forma de encerramento do host, incluindo `taskkill /F` no processo pai.

**O2 — Performance previsível.** Operações de leitura em latência de milissegundos, independentemente do tamanho do cofre dentro da faixa alvo (até 20.000 notas).

**O3 — Fidelidade de parsing.** Grafo de links, tags, headings e blocos corretos, com paridade funcional ao *metadata cache* do próprio Obsidian.

**O4 — Escrita segura.** Nenhuma escrita pode corromper uma nota, deixar arquivo parcial em disco, ou perder conteúdo em caso de crash no meio da operação.

**O5 — Instalação trivial.** Um binário, uma linha de configuração JSON. Sem gerenciador de pacotes, sem ambiente virtual, sem etapa de build para o usuário final.

---

## 3. Não-objetivos

Delimitar o que o produto **não** faz é o que impede o escopo de explodir.

**N1 — Não é plugin do Obsidian.** `gobsidian` roda como processo externo independente e funciona com o Obsidian fechado. Não há integração com a API de plugins, não há UI dentro do Obsidian.

**N2 — Não replica o Dataview.** Campos inline são extraídos e expostos como metadados consultáveis. A linguagem de consulta do Dataview, com suas expressões e views, está fora de escopo.

**N3 — Não sincroniza.** Sincronização é responsabilidade do OneDrive, do Obsidian Sync, do Git ou do que o usuário já usa. `gobsidian` observa o sistema de arquivos e reage.

**N4 — Não edita semanticamente.** `gobsidian` não reescreve prosa, não corrige, não reformata, não normaliza. Insere e substitui em posições estruturais explicitamente identificadas.

**N5 — Não resolve conflito de escrita concorrente com o Obsidian.** Se o usuário editar a mesma nota no Obsidian no exato instante de uma escrita via MCP, a última escrita completa vence. O produto garante atomicidade, não serialização entre processos independentes.

**N6 — Não embute modelo de linguagem.** Sem embeddings, sem busca semântica na v1. Busca é léxica. Ver §8 sobre reavaliação futura.

---

## 4. Usuário-alvo e cenários

### Perfil

Usuário técnico que mantém um cofre grande e estruturado (milhares de notas), usa um cliente MCP diariamente como parte do fluxo de trabalho, e trata o cofre como base de conhecimento de longo prazo — não como bloco de notas descartável. Trabalha primariamente em Windows, com o cofre em pasta sincronizada.

### Cenários de uso

**C1 — Consulta de conhecimento acumulado.** *"O que eu já tenho anotado sobre prescrição intercorrente?"* Requer busca full-text com ranking, retornando trechos com contexto suficiente para o modelo decidir se vale ler a nota inteira.

**C2 — Inserção incremental em nota consolidada.** *"Adicione o resumo do capítulo 118 ao arquivo de resumos, sob o heading correto, sem tocar no resto."* Requer `note_patch`/`note_append` por heading, com resolução robusta de heading e falha explícita se o heading não existir.

**C3 — Auditoria de cobertura.** *"Quais capítulos entre 1 e 120 ainda não têm resumo neste arquivo?"* Requer leitura estruturada de headings, sem carregar o conteúdo integral.

**C4 — Reorganização com integridade referencial.** *"Mova estas notas de `Por ponto/` para `Por matéria/`."* Requer `note_move` com reescrita de todos os wikilinks que apontam para as notas movidas, incluindo links com alias e com âncora de heading.

**C5 — Diagnóstico de saúde do cofre.** *"Que links estão quebrados? Que notas estão órfãs?"* Requer grafo de links completo e correto.

---

## 5. Requisitos funcionais

Prioridades: **P0** bloqueia a v1.0; **P1** desejável na v1.0, aceitável na v1.1; **P2** pós-1.0.

### 5.1 Indexação

| ID | Requisito | Prio |
|---|---|---|
| RF-01 | Varredura completa do cofre no boot, paralelizada por *worker pool* limitado a `runtime.NumCPU()` | P0 |
| RF-02 | Índice em memória com frontmatter, tags, headings, blocos, links de saída e backlinks por nota | P0 |
| RF-03 | Reindexação incremental disparada por evento do sistema de arquivos, reparseando apenas os arquivos afetados | P0 |
| RF-04 | Debounce e coalescência de eventos, com janela padrão de 250 ms | P0 |
| RF-05 | Recuperação de `ErrEventOverflow` do watcher por meio de varredura completa de reconciliação | P0 |
| RF-06 | Persistência do índice em cache em disco, fora do cofre, com invalidação por mtime e tamanho | P1 |
| RF-07 | Respeito a `.gitignore` e a um `.gobsidianignore` opcional | P1 |
| RF-08 | Exclusão automática de `.obsidian/`, `.trash/`, `.git/` e diretórios de anexos configuráveis | P0 |

### 5.2 Parsing

| ID | Requisito | Prio |
|---|---|---|
| RF-10 | Frontmatter YAML, com tipagem preservada (string, número, booleano, lista, data) | P0 |
| RF-11 | Wikilinks em todas as formas: simples, com alias, com âncora de heading, com âncora de bloco | P0 |
| RF-12 | Embeds (`![[...]]`) distinguidos de links comuns no grafo | P0 |
| RF-13 | Links Markdown padrão (`[texto](destino)`) resolvidos quando apontam para notas do cofre | P0 |
| RF-14 | Tags inline hierárquicas (`#a/b/c`), com a hierarquia preservada | P0 |
| RF-15 | Identificadores de bloco (`^id`) com deslocamento de byte para leitura direta | P0 |
| RF-16 | Hierarquia de headings com nível, texto, deslocamento inicial e final | P0 |
| RF-17 | Supressão de falsos positivos dentro de blocos de código cercados, código inline e sequências escapadas | P0 |
| RF-18 | Campos inline do Dataview (`chave:: valor`) extraídos como metadados | P1 |
| RF-19 | Callouts (`> [!tipo]`) identificados como estrutura | P2 |

RF-17 merece ênfase: é a diferença entre um grafo de links correto e um grafo plausível. É também o requisito que inviabiliza extração por regex e obriga o uso de um parser real.

### 5.2.1 Resolução de referências

Um wikilink não é um caminho, e resolvê-lo como se fosse produz um grafo que diverge do Obsidian de forma silenciosa. Estes requisitos existem separados do parsing porque são resolução, não sintaxe: dependem do conteúdo do cofre inteiro, não da nota que está sendo parseada.

| ID | Requisito | Prio |
|---|---|---|
| RF-60 | Indexação de anexos (`.png`, `.jpg`, `.jpeg`, `.gif`, `.webp`, `.svg`, `.pdf`, `.mp3`, `.mp4`, `.wav`, `.canvas`) por caminho, tamanho e mtime, sem leitura de conteúdo | P0 |
| RF-61 | Resolução de embed para anexo, de modo que `![[diagrama.png]]` não seja contabilizado como link quebrado | P0 |
| RF-62 | Resolução de wikilink pelo campo `aliases` do frontmatter da nota alvo — **divergência deliberada do Obsidian**, ver abaixo | P0 |
| RF-63 | Validação da âncora: `[[nota#heading]]` e `[[nota#^bloco]]` marcados como âncora quebrada quando a nota resolve mas o alvo interno não existe | P1 |

RF-60 e RF-61 andam juntos. Sem indexar os anexos, todo embed de imagem vira link quebrado, `vault_stats` reporta centenas de falsos positivos e a métrica de saúde do cofre deixa de ter uso. O custo é baixo — o índice guarda apenas a entrada de diretório, nunca os bytes.

**RF-62 é uma divergência deliberada, e a premissa original estava errada.**

Este requisito foi escrito assumindo que o Obsidian resolve `[[P3]]` quando alguma nota declara `aliases: [P3]`. A rodada de paridade contra o `metadataCache` real mostrou que **não resolve**: uma nota com `aliases: [P3, Terceiro]` foi corretamente registrada com esses aliases, e `[[Terceiro]]` apareceu em `unresolvedLinks`. O alias alimenta o seletor rápido e o autocomplete — que ao inserir escreve `[[Nota Real|Alias]]` — mas um `[[Alias]]` digitado à mão não vira aresta no grafo dele.

A decisão, tomada com a evidência na mão, é **manter a resolução por alias**. Um alias declarado é uma intenção explícita do autor de que aquele nome se refere àquela nota, e honrá-la encontra backlinks reais que o Obsidian perde. A regra de comparação assimétrica de §7 já cobre o caso: nossa saída precisa conter tudo o que o Obsidian encontrou, e encontrar mais é o produto.

A consequência precisa ser sabida por quem opera: `note_metadata` e `link_graph` mostram backlinks que o painel do Obsidian não mostra. Isso é uma diferença observável, não um bug a ser reportado.

O que **não** muda: alias continua sendo *fallback*, nunca *override*. Se existe `P3.md` e outra nota declara `aliases: [P3]`, `[[P3]]` aponta para o arquivo. A paridade confirmou essa precedência nos dois lados.

RF-63 vai além do que o próprio Obsidian expõe na interface, e é deliberado: uma âncora quebrada é exatamente o tipo de erro que aparece depois de renomear um heading, e é invisível até alguém clicar no link.

### 5.3 Busca

| ID | Requisito | Prio |
|---|---|---|
| RF-20 | Busca full-text com ranking BM25 sobre corpo e título | P0 |
| RF-21 | Filtros combináveis: pasta, tag, campo de frontmatter, intervalo de datas | P0 |
| RF-22 | Trechos de resultado com destaque de termos e contexto configurável | P0 |
| RF-23 | Normalização de acentos, *case folding* e indexação dupla — forma crua e forma reduzida — para português | P0 |
| RF-24 | Busca exata por frase entre aspas | P1 |
| RF-25 | Consulta estruturada apenas por metadados, servida do índice em memória sem tocar o índice de texto | P0 |
| RF-26 | Busca por similaridade semântica via embeddings locais | P2 |

RF-23 não é opcional para um cofre em português. Buscar por "usucapiao" precisa encontrar "usucapião", e buscar por "Prescrição" precisa encontrar "prescrição".

A redução morfológica é onde a escolha fica delicada. Stemming agressivo — Snowball português, RSLP — eleva o recall e destrói a precisão em corpus técnico, fundindo termos de arte que só se distinguem pelo sufixo. Ignorá-lo faz "prescrições" deixar de encontrar "prescrição", o que é um buraco óbvio.

A solução adotada é indexação dupla: cada token entra no índice na forma crua normalizada e, quando a redução produz algo diferente, também na forma reduzida, ambas apontando para a mesma posting list. A consulta casa qualquer uma das duas; a busca por frase exata (RF-24) usa apenas a forma crua. O recall vem do stem, a precisão vem de o termo original continuar presente e pontuar mais alto. O custo é em torno de 30% no tamanho do índice de termos — irrelevante nesta escala.

A redução em si é deliberadamente conservadora: plurais regulares e os sufixos verbais mais comuns. Não é um stemmer completo, e não deve virar um.

### 5.4 Escrita

| ID | Requisito | Prio |
|---|---|---|
| RF-30 | Criação de nota, falhando se o caminho já existir, com frontmatter opcional | P0 |
| RF-31 | Anexação ao fim da nota ou ao fim de uma seção identificada por heading | P0 |
| RF-32 | Substituição do conteúdo sob um heading, preservando o heading e as subseções fora do alvo | P0 |
| RF-33 | Substituição de bloco identificado por `^id` | P1 |
| RF-34 | Escrita atômica: arquivo temporário no mesmo volume seguido de rename | P0 |
| RF-35 | Movimentação e renomeação com reescrita de todos os links (wikilinks e links Markdown) apontando para a nota, preservando alias e âncora | P0 |
| RF-36 | Exclusão com relatório prévio de links que ficarão quebrados | P1 |
| RF-37 | Modo *dry-run* em todas as tools de escrita, retornando o diff sem tocar o disco | P1 |
| RF-38 | Preservação do estilo de fim de linha original do arquivo (CRLF/LF) | P0 |

RF-38 evita que cada escrita produza um diff de arquivo inteiro em cofres versionados por Git — problema clássico em Windows.

### 5.5 Ciclo de vida

| ID | Requisito | Prio |
|---|---|---|
| RF-40 | Encerramento ao detectar EOF em stdin | P0 |
| RF-41 | Tratamento de `SIGINT` e `SIGTERM` (e equivalentes no Windows) com *graceful shutdown* | P0 |
| RF-42 | Vigilância do PID pai, com encerramento se o pai deixar de existir | P0 |
| RF-43 | Encerramento por inatividade após intervalo configurável, desligado por padrão | P2 |
| RF-44 | Liberação de todos os *handles* de arquivo e do watcher no encerramento | P0 |

RF-40 a RF-42 são três mecanismos redundantes para o mesmo objetivo. A redundância é deliberada: cada um falha em cenários diferentes, e o custo de implementar os três é desprezível comparado ao custo do sintoma.

### 5.6 CLI e diagnóstico

| ID | Requisito | Prio |
|---|---|---|
| RF-50 | Subcomando `serve` (stdio) | P0 |
| RF-51 | Subcomando `doctor` com verificação de ambiente | P0 |
| RF-52 | Subcomandos `index`, `search`, `inspect` para uso fora do MCP | P1 |
| RF-53 | Log estruturado em stderr via `log/slog`, nível configurável | P0 |
| RF-54 | Transporte HTTP/SSE além de stdio | P2 |
| RF-55 | Flag global `--read-only`, que remove as tools de escrita da lista anunciada ao host, não apenas as rejeita em tempo de chamada | P0 |

RF-53 tem uma restrição não negociável: **stdout pertence ao protocolo JSON-RPC**. Qualquer byte escrito em stdout que não seja uma mensagem MCP válida corrompe a sessão. Todo log vai para stderr, sem exceção.

---

## 6. Requisitos não-funcionais

### 6.1 Performance

Cofre de referência para todas as medições: 5.000 notas Markdown, 50 MB de texto, SSD NVMe, Windows 11, Go 1.24.

| ID | Métrica | Alvo | Limite de falha |
|---|---|---|---|
| RNF-01 | Indexação a frio | ≤ 3 s | 6 s |
| RNF-02 | Boot com cache válido | ≤ 300 ms | 1 s |
| RNF-03 | `note_read`, p95 | ≤ 15 ms | 50 ms |
| RNF-04 | `vault_search` full-text, p95 | ≤ 100 ms | 300 ms |
| RNF-05 | `note_list` com filtro de metadados, p95 | ≤ 10 ms | 30 ms |
| RNF-06 | Reindexação de arquivo único | ≤ 20 ms | 100 ms |
| RNF-07 | RSS em repouso | ≤ 60 MB | 150 MB |
| RNF-08 | CPU em repouso | < 0,5 % | 2 % |
| RNF-09 | Escalabilidade | linear até 20.000 notas | — |

### 6.2 Confiabilidade

| ID | Requisito |
|---|---|
| RNF-10 | Zero processos órfãos em 100 ciclos de start/kill do host — critério de bloqueio de release |
| RNF-11 | Zero notas corrompidas em teste de crash injetado durante escrita, 1.000 iterações |
| RNF-12 | Índice degradado nunca produz resultado incorreto: em caso de inconsistência detectada, reconstrói |
| RNF-13 | Falha de uma tool jamais derruba o servidor; erros são retornados como resultado MCP |

### 6.3 Compatibilidade

| ID | Requisito |
|---|---|
| RNF-20 | Windows 10+ como plataforma de primeira classe; macOS 13+ e Linux com kernel 5.x+ como plataformas suportadas |
| RNF-21 | Cofres em OneDrive, Dropbox e Google Drive, incluindo arquivos somente-nuvem |
| RNF-22 | Caminhos acima de 260 caracteres no Windows |
| RNF-23 | Nomes de arquivo com acentuação e espaços |
| RNF-24 | Versão de protocolo MCP fixada em `2025-11-25`, com negociação de fallback para `2025-06-18`, `2025-03-26` e `2024-11-05` |

**Sobre RNF-20.** "Primeira classe" e "suportada" são níveis diferentes de garantia, e a distinção é deliberada. Nos três sistemas o código compila, a suíte de testes roda no CI e há binário de release. Apenas no Windows rodam também os testes que dependem de comportamento de plataforma: ciclo de órfãos com `taskkill /F`, `ERROR_SHARING_VIOLATION` no rename, arquivos somente-nuvem do OneDrive, `MAX_PATH` e colisão de casing. Prometer paridade de verificação nos três seria promessa não cumprida.

**Sobre RNF-24.** A versão `2025-11-25` é a última estável com suporte pleno no SDK Go oficial e a que os hosts instalados negociam hoje. A revisão `2026-07-28` remove o handshake `initialize` e a sessão de protocolo, o que é uma mudança estrutural, não incremental — entra em uma versão posterior, atrás da camada de adaptação de `internal/mcpsrv` (ARCHITECTURE §2.3).

### 6.4 Segurança

| ID | Requisito |
|---|---|
| RNF-30 | Nenhuma requisição de rede. O código do produto não abre socket de saída em nenhuma circunstância |
| RNF-31 | Todo caminho recebido de uma tool é resolvido e verificado como interno ao cofre; travessia rejeitada |
| RNF-32 | Links simbólicos que apontem para fora do cofre não são seguidos |
| RNF-33 | Modo `--read-only` que desabilita toda a superfície de escrita |

RNF-30 é uma propriedade de produto, não apenas técnica: o cofre pode conter material confidencial, e a garantia de que o servidor não exfiltra precisa ser verificável, não apenas afirmada.

A formulação exige cuidado. O SDK oficial de MCP importa `net/http` para o transporte HTTP/SSE, e essa importação entra no grafo de dependências mesmo quando só o transporte stdio é usado. Uma regra de CI do tipo "nenhum pacote `net/*` no grafo" falharia permanentemente e acabaria desabilitada — pior que não existir.

A regra verificável é outra, em três partes:

1. Nenhum pacote sob `internal/` ou `cmd/` importa `net`, `net/http` ou qualquer pacote de rede. Verificado por análise estática do grafo de importação **do nosso código**, não do fechamento transitivo.
2. Nenhuma chamada a `net.Dial`, `http.Get`, `http.Client` ou equivalente no código do produto. Verificado por `go vet` com um analisador próprio.
3. RF-54 (transporte HTTP/SSE) fica fora da v1. O único transporte construído é stdio.

O resultado é honesto: `net/http` está compilado no binário porque o SDK o carrega, e nunca é exercitado. Auditável em um comando, e é a garantia que dá para sustentar sem manter um fork podado do SDK.

---

## 7. Métricas de sucesso

**Funcional.** Paridade do grafo de links com o *metadata cache* do Obsidian em um corpus de teste de 500 notas contendo todos os casos de borda documentados: divergência zero.

O *metadata cache* do Obsidian vive no IndexedDB do Electron e não é um arquivo que se possa ler de fora. A referência é obtida uma única vez por um plugin de desenvolvimento descartável, em `tools/parity-dumper/`, que serializa `app.metadataCache` do corpus para JSON e o versiona em `testdata/parity/`. O plugin não é parte do produto e não é distribuído; existe para que a métrica seja o comportamento real do Obsidian, e não a interpretação da documentação dele. Regenerar a referência é uma operação manual, feita quando o Obsidian mudar de comportamento de forma relevante.

**Performance.** Todos os alvos de RNF-01 a RNF-08 atingidos em benchmark automatizado no CI.

**Estabilidade.** RNF-10 satisfeito. Nenhum processo órfão em execução repetida do ciclo de encerramento abrupto.

**Adoção pessoal.** O critério honesto: substituir integralmente o servidor MCP de Obsidian atualmente em uso, sem que nenhuma operação do fluxo de trabalho precise recorrer a contornos manuais como cópia de arquivo para sandbox antes da leitura.

---

## 8. Riscos e mitigações

| Risco | Impacto | Probabilidade | Mitigação |
|---|---|---|---|
| Eventos espúrios do OneDrive causando reindexação constante | Alto | Alta | Debounce agressivo; ignorar eventos que não alterem mtime nem tamanho; detectar e ignorar arquivos de estado do OneDrive |
| Arquivos somente-nuvem bloqueando a leitura por download síncrono | Alto | Média | Detectar atributo de *reparse point*; indexar apenas metadados sem forçar hidratação; expor o estado na tool e no `doctor` |
| Divergência do parser em relação ao Obsidian | Médio | Média | Corpus de teste com casos de borda extraídos da documentação oficial; testes de golden file |
| API do SDK Go do MCP em evolução, com quebras entre versões | Médio | Média | Fixar versão exata no `go.mod`; isolar o SDK atrás de uma camada de adaptação interna |
| Buffer overflow do `ReadDirectoryChangesW` em cofres muito ativos | Médio | Baixa | Tratar `ErrEventOverflow` com varredura de reconciliação (RF-05) |
| Reescrita de links em `note_move` corrompendo notas | Alto | Baixa | Dry-run obrigatório em testes; escrita atômica por arquivo; reversão em caso de falha parcial |
| Consumo de memória em cofres muito grandes | Médio | Baixa | Índice armazena deslocamentos, não conteúdo; corpo lido do disco sob demanda |

O primeiro risco da tabela é o mais provável de se materializar e o mais chato de diagnosticar. Merece instrumentação dedicada desde o primeiro dia: contador de eventos recebidos, coalescidos e efetivamente processados, exposto por `vault_stats`.

---

## 9. Marcos

### M0 — Esqueleto (1 semana) — **Concluído em 2026-07-28**

Servidor MCP em stdio respondendo a `initialize`. Uma tool trivial (`vault_stats`) devolvendo contagem de arquivos. Ciclo de vida completo (RF-40 a RF-44) implementado e testado antes de qualquer outra coisa. `gobsidian doctor` funcional.

Encerrar o ciclo de vida primeiro é deliberado: é o requisito que define o produto, e postergá-lo garante que ele nunca fique bom.

### M1 — Leitura (2 semanas) — **v0.1, primeiro release utilizável — Concluído em 2026-07-28**

Parser completo (RF-10 a RF-17). Resolução de referências (RF-60 a RF-63). Índice em memória construído no boot. `note_read`, `note_list`, `note_metadata`, `link_graph`, `tag_list`. Testes de golden file do parser e de paridade com o Obsidian.

**Sem watcher e sem busca.** Reindexar no boot basta para validar tudo o que existe até aqui, e o custo de reiniciar o host quando o cofre muda é aceitável em troca de chegar a algo utilizável três semanas antes.

Ao fim de M1 o produto já substitui a parte de leitura do fluxo de trabalho atual. Esse é o critério de corte: v0.1 não é uma demonstração, é o que passa a ser usado todo dia. O que vem depois é construído sobre uma fundação que já está sob uso real, com os defeitos que só o uso real revela já visíveis.

### M2 — Watcher (meia semana)

`fsnotify` com filtro, debounce e coalescência. Reindexação incremental. Recuperação de `ErrEventOverflow` por varredura de reconciliação. Contadores expostos em `vault_stats`.

Curto porque o índice e o parser já estão corretos e testados; o watcher é apenas a fonte dos eventos que disparam o que já funciona.

### M3 — Busca (1 semana)

Índice invertido, BM25, filtros, analisador de português com indexação dupla. `vault_search`. Cache de índice em disco (RF-06).

### M4 — Escrita (2 semanas)

Escritas atômicas. `note_create`, `note_append`, `note_patch`. Preservação de fim de linha e de BOM. Dry-run. Serialização de escrita por caminho. Teste de crash injetado.

### M5 — Refatoração do cofre (1 semana)

`note_move` com reescrita de links. `note_delete` com relatório de impacto. Detecção de links e âncoras quebradas.

### M6 — Endurecimento (1 semana) — **v1.0**

Suíte de benchmark no CI com verificação de regressão. Teste de 100 ciclos de encerramento abrupto. Analisador de importações de rede no CI. Documentação de operação. Release binário para Windows, macOS e Linux.

---

## 10. Decisões fechadas

| # | Questão | Decisão |
|---|---|---|
| D1 | Local do cache de índice (RF-06) | Fora do cofre, em `%LOCALAPPDATA%\gobsidian\<hash-do-caminho-do-cofre>\`. O cache é derivado e descartável; portabilidade entre máquinas não vale o custo de o Obsidian indexá-lo e o OneDrive sincronizá-lo |
| D2 | `note_patch` com heading inexistente | Falha por padrão. Criar o heading exige `create_if_missing: true` explícito |
| D3 | Escrita em lote transacional | Fora de escopo. Não há transação entre arquivos no sistema de arquivos, e prometer uma seria mentira. Reavaliar pós-1.0 |
| D4 | Múltiplos cofres por instância | Um cofre por instância. Cofres separados viram entradas separadas no `claude_desktop_config.json` |
| D5 | Nome do produto | `gobsidian`. Módulo `github.com/jonyd/gobsidian`, binário `gobsidian.exe` |
| D6 | Versão de protocolo MCP | `2025-11-25` fixada, com fallback negociado. Migração para `2026-07-28` atrás de `internal/mcpsrv`, pós-1.0 |
| D7 | Garantia de ausência de rede | Análise estática sobre `internal/` e `cmd/`, não sobre o fechamento transitivo. RF-54 fora da v1. Ver §6.4 |
| D8 | Referência de paridade com o Obsidian | Plugin descartável em `tools/parity-dumper/` serializa `app.metadataCache` uma vez para `testdata/parity/` |
| D9 | Redução morfológica na busca | Indexação dupla: forma crua normalizada e forma reduzida conservadora, na mesma posting list. Ver §5.3 |
| D10 | Escopo de plataforma | Windows de primeira classe; macOS e Linux compilam, testam no CI e têm release. Ver §6.3 |
| D11 | Ordem de entrega | M0 e M1 primeiro, entregues como v0.1 utilizável. Watcher, busca e escrita depois. Ver §9 |
| D12 | Anexos, aliases e âncoras | Todos indexados e resolvidos. Ver §5.2.1 |
| D13 | Esquema de URI dos resources | `gobsidian://`, não `obsidian://`. O segundo é o esquema real do aplicativo Obsidian, registrado no sistema operacional, e reusá-lo para outra semântica é colisão certa |

## 11. Questões em aberto

**Q1.** Busca semântica (RF-26) justifica embutir um modelo de embeddings, com o custo em tamanho de binário e em tempo de indexação? *Reavaliar após a v1.0, com dados de uso real de `vault_search`. O critério é concreto: se as consultas que falham em `vault_search` forem majoritariamente de paráfrase, sim; se forem de sintaxe de consulta, não.*

**Q2.** Quando migrar para o protocolo `2026-07-28`? A revisão remove o `initialize` e a sessão, e deprecia Roots, Sampling e Logging. *Gatilho: quando o Claude Desktop instalado passar a negociá-la e o SDK Go marcá-la como estável. A janela de depreciação anunciada é de 12 meses, o que dá folga.*

**Q3.** O cache de índice deve guardar também o índice de busca, ou apenas o índice de metadados? **FECHADA em 2026-07-29 na Task 52 com medição real.** Medição em corpus de 500 notas distintas (`idx.NoteCount() == 500`): (a) `LoadInvertedCache` do disco: 26,96 ms; (b) Reconstrução do índice invertido a partir do `index` de metadados já carregado: 106,58 ms. **Decisão:** Persistir ambos os caches (`index_cache.gob` e `inverted_cache.gob`). O carregamento do cache invertido do disco é ~4x mais rápido do que a reconstrução em memória (26,96 ms vs 106,58 ms), economizando ~80 ms no boot frio e mantendo o RNF-02 (≤ 300 ms) com ampla folga.

> **Anotação de 2026-08-04, duas correções de fato.** Nenhuma delas reabre a decisão; as duas existem porque a redação acima afirma coisas que hoje são falsas.
>
> 1. **`index_cache.gob` nunca foi implementado.** Só `inverted_cache.gob` é gravado — `grep index_cache` não acha nada no código, e o diretório de cache de um cofre real tem um arquivo só. O índice de metadados é reconstruído por varredura a cada partida, em ~900 ms num cofre de 109 MB. A decisão continua fechada como está escrita; o que falta é implementá-la ou revisá-la, e o custo real de não tê-la está medido.
> 2. **"Mantendo o RNF-02 com ampla folga" não se sustenta fora do corpus de 500 notas.** Os 26,96 ms medem `LoadInvertedCache` isolado num corpus sintético. Num cofre real de 3.149 notas o boot com cache válido leva **832–1183 ms** contra o teto de 300 ms — RNF-02 **não atingido**, e registrado assim em [`OPERACAO.md`](OPERACAO.md). Era ~7 s antes da troca do formato do cache.
>
> **Anotação de 2026-08-06, Task 85: `index_cache.gob` implementado.** Fecha o ponto 1 acima. `internal/index/persist.go` e `persist_codec.go` persistem o índice de metadados com o mesmo tipo de codec binário do índice de busca (varint, tabela ordenada, portão de versão), mais um CRC32 de rodapé — divergência deliberada do codec de busca, justificada no comentário do arquivo: um cache de metadados errado serve nota errada, não nota lenta, então um byte corrompido no meio do payload de uma string não pode decodificar "com sucesso". `cmd/gobsidian/serve.go` tenta o cache antes de `idx.Build`, e só aceita quando `Index.VerifyFreshness` confirma mtime e tamanho iguais aos do disco, para cada arquivo — sem isso, um cache com uma nota editada offline serviria metadados desatualizados sem aviso. Medido num cofre **sintético** de mesma escala do cofre real citado no ponto 2 (3.149 notas, 50 anexos, 107,93 MB, `scripts/gen_vault.ps1 -Notes 3149 -BodyKB 35 -Seed 42`) — não foi possível confirmar o cofre real exato usado no ponto 2 dentro desta tarefa: cinco partidas sem cache **852–7736 ms** (a primeira paga leitura fria de disco do SO, as quatro seguintes 852–901 ms), cinco partidas com cache **208–282 ms**. **RNF-02 passa a ser atingido nesta escala** — as cinco partidas com cache ficaram abaixo do teto de 300 ms. Ver `docs/OPERACAO.md` para a tabela completa e a ressalva sobre o cofre não ser idêntico ao medido no ponto 2.
