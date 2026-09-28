# Textos da CLI

Este documento reúne tudo o que o comando *gobsidian* escreve na tela para uma
pessoa ler, na ordem em que ela encontra cada tela, para que a redação possa
ser revista num lugar só. Cada texto aparece como sai num terminal UTF-8, mas
**sem as molduras** (os cantos e traços das caixas foram omitidos). O
comentário HTML no fim de cada linha, visível só no código-fonte deste arquivo,
diz de onde o texto vem: o nome da constante em internal/textos/textos.go, ou
arquivo:linha quando o texto está escrito direto no código. Verbos de
formatação (%s, %d, %q, %v, %w) são preenchidos na hora de imprimir e
precisam continuar no texto editado, na mesma ordem.

Este documento é derivado: onde divergir do código, o código vence. Log de
slog, que vai para o stderr em inglês de máquina, e a descrição das tools MCP,
que é lida por modelos e não pela CLI, ficam de fora.

## Como ler

| Na tela | Aqui |
|---|---|
| linha de sucesso (OK) | ✅ no começo da linha |
| aviso (Warn) | ⚠️ no começo da linha |
| falha (Err) | ❌ no começo da linha |
| informação (Info) | ℹ️ no começo da linha |
| etapa de relatório (Step) | ⏳ no começo da linha |
| passo de uma sequência (Passo), indentado e apagado | ▸ no começo da linha |
| resposta já dada (Resposta) | ✓ no começo da linha |
| linha de detalhe (Detail), indentada sob a anterior | item aninhado sob o item anterior |
| título de seção (Titulo, sai com ▪) e título de moldura | cabeçalho de quarto nível |
| rodapé de moldura | linha em itálico abaixo do bloco |
| pergunta, lista de seleção e botões | pergunta em negrito, opções em lista, teclas em itálico |
| pares campo/valor (Campos) | tabela de duas colunas |
| ajuda do comando e das flags | tabela flag, texto |
| dado do momento (caminho, número, nome de host) | ‹entre aspas angulares› |

Dentro de um texto, a marcação vai para o código como está e o console a
desenha: `**negrito**` sai em negrito, `*itálico*` em itálico e `` `código` ``
em destaque, sem as crases. Onde a saída não tem cor (redirecionada, pipe,
NO_COLOR), negrito e itálico somem e ficam só as palavras; as crases
continuam, porque são o que ainda separa um comando do texto em volta.

As regras de redação, aplicadas ao documento inteiro:

- Toda frase e todo título começam com maiúscula e terminam com ponto. A
  exceção é a linha que termina num erro embrulhado (%v ou %w no fim): o erro
  de dentro já traz a pontuação dele.
- Ponto e vírgula vira ponto e nova frase.
- Nome do produto em negrito: **gobsidian**. Dentro de um comando, entre
  crases: `gobsidian install`.
- Flag, comando e nome de arquivo entre crases: `--vault`, `.obsidian`.
- Termo técnico em inglês em itálico: *daemon*, *runtime*, *socket*,
  *symlink*, *handshake*, *casing*, *PATH*, *watcher*, *boot*, *commit*,
  *build*, *headings*, *backlinks*.

Num console que não aguenta UTF-8 (CP-850), os marcadores saem escritos
([OK], [!], [i], [*], [...]) e o acento cai na hora de imprimir; a redação é a
mesma. Os cabeçalhos de segundo e terceiro nível deste documento organizam as
telas e não aparecem na tela.

## *gobsidian* sem argumentos

Com o *gobsidian* já instalado, ou sem um terminal do outro lado, o comando
sozinho mostra a ajuda. Sem instalação e com terminal, ele se instala.

### Autoinstalação

- ⏳ O **gobsidian** ainda não está instalado nesta máquina. <!-- textos.AutoinstalarNaoInstalado -->
  - Vamos instalar em `%s`. <!-- textos.AutoinstalarDestino -->
  - Para ver apenas a ajuda, rode `gobsidian --help`. <!-- textos.AutoinstalarAjuda -->

Em seguida vem o fluxo de instalação (seção install abaixo).

### Ajuda (gobsidian --help)

#### Servidor MCP para cofres locais do Obsidian. <!-- textos.ResumoRaiz -->

**Uso:** <!-- textos.AjudaUso -->

**Comandos disponíveis:** <!-- textos.AjudaComandos -->

| Comando | Texto |
|---|---|
| completion | Gera o script de autocompletar do shell. <!-- textos.ResumoCompletion --> |
| doctor | Diagnostica o ambiente. <!-- textos.ResumoDoctor --> |
| help | Exibe a ajuda sobre qualquer comando. <!-- textos.ResumoHelp --> |
| index | Constrói o índice do cofre e exibe um resumo. <!-- textos.ResumoIndex --> |
| inspect | Exibe metadados, links e *backlinks* de uma nota. <!-- textos.ResumoInspect --> |
| install | Instala o **gobsidian**. <!-- textos.ResumoInstall --> |
| path | Acrescenta ou remove o diretório de instalação do *PATH* do usuário. <!-- textos.ResumoPath --> |
| search | Busca um texto no cofre. <!-- textos.ResumoSearch --> |
| update | Atualiza o **gobsidian**. <!-- textos.ResumoUpdate --> |
| config | Configura o **gobsidian** como MCP de hosts de IA. <!-- textos.ResumoConfig --> |
| vaults | Lista os cofres do Obsidian e os configurados nos hosts de IA. <!-- textos.ResumoVaults --> |
| version | Imprime versão. <!-- textos.ResumoVersion --> |
| serve (oculto) | Serve o cofre via MCP sobre stdio. <!-- textos.ResumoServe --> |
| daemon (oculto) | Roda o *daemon* de cofre compartilhado (uso interno da ponte). <!-- textos.ResumoDaemon --> |

**Flags:** <!-- textos.AjudaFlags -->

| Flag | Texto |
|---|---|
| -h, `--help` | Exibe a ajuda do **‹comando›**. <!-- textos.FlagHelp --> |

Use `gobsidian [comando] --help` para detalhes de um comando. <!-- textos.AjudaDetalhes -->

## install

### Ajuda

| Flag | Texto |
|---|---|
| (resumo) | Instala o **gobsidian**. <!-- textos.ResumoInstall --> |
| (descrição) | Instala o executável no perfil do usuário, sem elevação, pergunta se deve acrescentar o diretório ao *PATH* e registrar o servidor nos hosts de IA detectados. <!-- textos.DescricaoInstall --> |
| `--vault` | Cofre a servir (padrão: listar cofres registrados no Obsidian para escolha). <!-- textos.FlagInstallVault --> |
| `--install-dir` | Diretório onde o **gobsidian** será instalado (padrão: `‹diretório padrão›`). <!-- textos.FlagInstallDir, com internal/instalar DiretorioPadrao no meio --> |
| `--hosts` | Lista de hosts de IA (Claude, Codex, AGY, etc.) a configurar, separados por vírgula (`‹chaves dos hosts›`). `none` não configura nenhum. <!-- textos.FlagInstallHosts, com as chaves no meio --> |
| `--yes` | Não pergunta nada: instala, ajusta o *PATH* e configura todos os hosts de IA detectados. <!-- textos.FlagInstallYes --> |
| `--read-only` | Registra o servidor apenas para leitura. <!-- textos.FlagInstallReadOnly --> |
| `--no-path` | Não altera o *PATH*. <!-- textos.FlagInstallNoPath --> |

### Sem terminal

- ⚠️ A entrada não é um terminal interativo, então serão utilizadas as opções padrão. <!-- textos.InstallSemTerminal -->
  - Para escolher os cofres, rode `gobsidian install` em um terminal interativo. <!-- textos.InstallSemTerminalDica -->

### Configuração existente

#### Configuração atual. <!-- textos.InstallConfigAtual -->

- `‹caminho de cada cofre já configurado›`

**Manter esta configuração?** <!-- textos.InstallManterConfig -->

- [•Sim•]  ⊏ Não ⊐ <!-- textos.BotaoSim, textos.BotaoNao; o botão em foco sai entre [• •], o outro entre ⊏ ⊐ -->

**[←]/[→]** *mover* · **[s]/[n]** *responder* · **[⏎ enter]** *confirmar* <!-- textos.TeclaMover, textos.TeclaResponder, textos.TeclaConfirmar -->

Sem terminal, a mesma pergunta é digitada:

**◆ Manter esta configuração? (S/n) ▸** <!-- textos.InstallManterConfig; sufixo textos.SufixoSimPadrao ou textos.SufixoNaoPadrao -->

- ✓ Sim <!-- textos.RespostaSim -->
- ✓ Não <!-- textos.RespostaNao -->
- ✓ Cancelado <!-- textos.RespostaCancelado -->

### Escolha dos cofres

- ⚠️ Não foi possível ler os cofres do Obsidian: %v <!-- textos.InstallRegistroIlegivel -->

**Quais cofres configurar?** <!-- textos.InstallQuaisCofres -->

- ▸ ▣ ‹nome do cofre›  *‹caminho do cofre›*
-   ■ ‹nome do cofre›  *‹caminho do cofre›*
-   ▢ ‹nome do cofre›  *‹caminho do cofre›*

▣ é a linha em foco, ■ a marcada e ▢ a vazia. Os cofres já configurados vêm
marcados. Um cofre configurado que o Obsidian não conhece aparece com o nome da
pasta.

**[↑]/[↓]** *mover* · **[ espaço ]** *marcar* · **[a]** *todos* · **[⏎ enter]** *confirmar* <!-- textos.TeclaMoverLista, textos.TeclaMarcar, textos.TeclaTodos, textos.TeclaConfirmar -->

Sem terminal, a lista é numerada e a escolha é digitada:

#### Quais cofres configurar? <!-- textos.InstallQuaisCofres -->

- ‹n›) ‹nome do cofre›  *‹caminho do cofre›* <!-- textos.InstallItemNumerado -->

**◆ Números separados por espaço, `*` para todos, vazio para nenhum. ▸** <!-- textos.InstallEscolhaDigitada -->

Com `--yes` e nenhum cofre configurado antes:

- ℹ️ Cofre: %s <!-- textos.InstallCofreEscolhido -->

### Escolha dos hosts

- ℹ️ Nenhum host de IA conhecido foi detectado. <!-- textos.InstallSemHosts -->

Com `--yes`:

#### Hosts de IA encontrados. <!-- textos.InstallHostsEncontrados -->

- ‹nome de cada host detectado›

Com terminal:

**Em quais hosts registrar?** <!-- textos.InstallQuaisHosts -->

- ▸ ▣ ‹nome do host›  *‹chave do host›*
-   ■ ‹nome do host›  *‹chave do host›*
-   ▢ ‹nome do host›  *‹chave do host›*

**[↑]/[↓]** *mover* · **[ espaço ]** *marcar* · **[a]** *todos* · **[⏎ enter]** *confirmar*

Sem terminal:

#### Em quais hosts registrar? <!-- textos.InstallQuaisHosts -->

- ‹n›) ‹nome do host›  *‹chave do host›* <!-- textos.InstallItemHost -->

**◆ Números separados por espaço, `*` para todos, vazio para nenhum. ▸** <!-- textos.InstallEscolhaDigitada -->

Os nomes dos hosts estão em textos (textos.HostNome*): Claude Desktop, Claude
Code (CLI), Gemini CLI, Antigravity, Antigravity IDE, Codex CLI, VS Code,
Cursor, Windsurf.

### Instalando

#### Instalando. <!-- textos.InstallTitulo -->

- ◔ Tomando a trava de instalação. <!-- textos.PassoTrava -->
- ◑ Procurando processos em execução. <!-- textos.PassoProcessos -->
- ◕ Limpando lixo de execuções anteriores. <!-- textos.PassoLimpeza -->
- ● Conferindo as chaves de cache. <!-- textos.PassoChaves -->
- ◔ Instalando o binário. <!-- textos.PassoBinario -->
- ◑ Ajustando o *PATH*. <!-- textos.PassoPath -->
- ◕ Configurando os hosts de IA. <!-- textos.PassoHosts -->
- ● Gravando o manifesto. <!-- textos.PassoManifesto -->

Se houver processos do **gobsidian** rodando:

**Encerrar estes processos do gobsidian para trocar o binário?** <!-- textos.InstallEncerrarProcessos -->

- pid ‹pid›  ‹papel›  ‹cofre› <!-- textos.InstallProcessoItem -->
- ⊏ Sim ⊐  [•Não•]

**[←]/[→]** *mover* · **[s]/[n]** *responder* · **[⏎ enter]** *confirmar*

Com `--yes`, a pergunta sai como aviso e segue sem esperar:

- ⚠️ Encerrar estes processos do **gobsidian** para trocar o binário? <!-- textos.InstallEncerrarProcessos -->
  - pid ‹pid›  ‹papel›  ‹cofre› <!-- textos.InstallProcessoItem -->
  - `--yes`: encerrando sem perguntar. <!-- textos.InstallYesEncerrando -->

Se a resposta for não:

- ⚠️ Instalação cancelada. Nada foi alterado. <!-- textos.InstallCancelado -->

### Resumo

- ✅ Instalado. <!-- textos.InstallConcluido -->

Bloco sem título, uma linha por item:

- Binário  %s <!-- textos.ResumoBinario -->
- Cofres   nenhum configurado <!-- textos.ResumoCofres -->
- Cofre    %s <!-- textos.ResumoCofre -->
- *PATH*     %s <!-- textos.ResumoPATH -->
- %-16s %s <!-- textos.ResumoHostOK: chave do host e o aviso abaixo -->
- %-16s FALHOU: %s <!-- textos.ResumoHostFalha -->

O valor da linha *PATH*, no Windows:

- Abra um terminal NOVO para carregar o *PATH* atualizado. <!-- textos.AvisoPathWindows -->

E fora do Windows:

- Abra um terminal NOVO, ou rode `source ~/.profile` para atualizar o *PATH*. <!-- textos.AvisoPathUnix -->

O aviso de cada host configurado com sucesso:

| Host | Aviso |
|---|---|
| Claude Desktop | Reinicie o Claude Desktop para carregar o servidor. <!-- textos.HostClaudeDesktop --> |
| Claude Code | Registrado no Claude Code. <!-- textos.HostClaudeCode --> |
| Gemini CLI | Registrado no Gemini CLI. <!-- textos.HostGeminiCLI --> |
| Antigravity | Reinicie o Antigravity. <!-- textos.HostAntigravity --> |
| Antigravity IDE | Reinicie o Antigravity IDE. <!-- textos.HostAntigravityIDE --> |
| Codex | Registrado em `~/.codex/config.toml`. <!-- textos.HostCodex --> |
| VS Code | Registrado na configuração de usuário do VS Code. <!-- textos.HostVSCode --> |
| Cursor | Reinicie o Cursor. <!-- textos.HostCursor --> |
| Windsurf | Reinicie o Windsurf. <!-- textos.HostWindsurf --> |

### Erros

- Nenhum cofre encontrado. Passe `--vault` com o caminho do cofre. <!-- textos.ErroSemCofre -->
- Seleção cancelada. Nada foi alterado. <!-- textos.ErroSelecaoCancelada -->
- Host desconhecido %q. Conhecidos: %s <!-- textos.ErroHostDesconhecido -->
- Escolha inválida: %q <!-- textos.ErroEscolhaInvalida -->
- Escolha fora da lista: %d <!-- textos.ErroEscolhaForaDaLista -->

Os erros de internal/instalar e internal/hosts que podem chegar aqui estão na
última seção.

## config

Até 2026-09-27 este comando se chamava `vaults`.

### Ajuda

| Flag | Texto |
|---|---|
| (resumo) | Configura o **gobsidian** como MCP de hosts de IA. <!-- textos.ResumoConfig --> |

As flags são as mesmas de install (`--vault`, `--install-dir`, `--hosts`,
`--yes`, `--read-only`, `--no-path`), com o mesmo texto.

### Na tela

A escolha de cofres e de hosts é a mesma de install. Depois:

- ✅ Configuração mantida. Nada foi alterado. <!-- textos.VaultsMantido -->

ou:

- ✅ Hosts configurados. <!-- textos.InstallHostsConfigurados -->
  - Nenhum cofre configurado. <!-- textos.InstallSemCofreNaLista -->
  - Cofre    %s <!-- textos.InstallCofreNaLista -->
  - %-16s %s <!-- textos.InstallHostNaLista: chave do host e o aviso da tabela de install -->
- ⚠️ %s não pode ser configurado. <!-- textos.InstallHostFalhou -->
  - ‹erro do host›

### Erros

- %w -- Rode `gobsidian install` primeiro. <!-- textos.ErroSemManifesto; o %w é textos.ErroManifestoAusente -->
- Não há manifesto de instalação <!-- textos.ErroManifestoAusente -->

## vaults

### Ajuda

| Flag | Texto |
|---|---|
| (resumo) | Lista os cofres do Obsidian e os configurados nos hosts de IA. <!-- textos.ResumoVaults --> |
| `--json` | Força a saída em JSON, mesmo no terminal. <!-- textos.FlagJSON --> |
| `--texto` | Força a saída para ler, mesmo fora do terminal. <!-- textos.FlagTexto --> |

### Na tela

Fora do terminal (pipe ou arquivo), sai a lista em JSON numa linha, sem texto nenhum.

#### Cofres. <!-- textos.VaultsTitulo -->

- **‹nome do cofre›**  ‹estado›
  - ‹caminho do cofre›

O estado junta, separados por vírgula:

- aberto no Obsidian <!-- textos.VaultsAberto -->
- configurado <!-- textos.VaultsConfigurado -->
- fora do Obsidian <!-- textos.VaultsForaDoObsidian -->

*Rode `gobsidian config` para registrar um cofre nos hosts de IA.* <!-- textos.VaultsRodape -->

Sem cofre nenhum:

- ℹ️ Nenhum cofre encontrado no Obsidian nem nos hosts de IA. <!-- textos.VaultsNenhum -->

### Erros

- `--json` e `--texto` não podem ser usados juntos. <!-- textos.ErroJSONETexto -->

## update

### Ajuda

| Flag | Texto |
|---|---|
| (resumo) | Atualiza o **gobsidian**. <!-- textos.ResumoUpdate --> |
| (descrição) | Consulta a versão publicada, baixa o binário da plataforma corrente, confere o SHA-256 publicado e só então encerra os processos em execução e troca o binário. Divergência de soma aborta sem instalar nada. <!-- textos.DescricaoUpdate --> |
| `--check` | Só diz se há versão nova, sem baixar nem instalar. <!-- textos.FlagUpdateCheck --> |
| `--yes` | Não pergunta antes de encerrar os processos em execução. <!-- textos.FlagUpdateYes --> |

### Na tela

- ⏳ Consultando a última versão publicada. <!-- textos.UpdateConsultando -->
- ℹ️ Versão instalada: %s <!-- textos.UpdateInstalada -->
- ℹ️ Versão publicada: %s <!-- textos.UpdatePublicada -->
- ✅ Já está atualizado. <!-- textos.UpdateJaAtual -->

Com `--check` e versão nova:

- ⚠️ Há versão nova: %s <!-- textos.UpdateHaVersaoNova -->
  - Rode `gobsidian update` para instalar. <!-- textos.UpdateComoInstalar -->

Sem `--check`:

- ⏳ Baixando %s e conferindo o SHA-256. <!-- textos.UpdateBaixando -->
- ❌ O binário baixado NÃO confere com a soma publicada. <!-- textos.UpdateHashDiverge -->
  - Nada foi instalado. Sua instalação continua intacta. <!-- textos.UpdateNadaMudou -->
- ✅ SHA-256 confere. <!-- textos.UpdateHashConfere -->
- ⏳ Trocando o binário. <!-- textos.UpdateTrocando -->

Seguem os passos, a pergunta sobre encerrar processos e o resumo, iguais aos de
install. Se a resposta for não:

- ⚠️ Atualização cancelada. Nada foi alterado. <!-- textos.UpdateCancelado -->

No fim:

- ✅ Atualizado para %s. <!-- textos.UpdateConcluido -->
- (o resumo de install)
  - Os hosts reiniciam o servidor sozinhos. Não precisa reiniciar manualmente. <!-- textos.UpdateHostsSozinho -->

### Erros

- Consultando *releases*: %w <!-- textos.ErroConsultarRelease -->
- O *release* %s não publica %q (plataforma %s/%s). <!-- textos.ErroAtivoAusente -->
- Criando diretório temporário: %w <!-- textos.ErroTemporario -->
- %w -- Rode `gobsidian install` primeiro. <!-- textos.ErroSemManifesto -->

Os erros de internal/selfupdate (download, soma) estão na última seção.

## path

### Ajuda

| Flag | Texto |
|---|---|
| (resumo) | Acrescenta ou remove o diretório de instalação do *PATH* do usuário. <!-- textos.ResumoPath --> |
| `--add` | Acrescenta o diretório ao *PATH*. <!-- textos.FlagPathAdd --> |
| `--remove` | Remove o diretório do *PATH*. <!-- textos.FlagPathRemove --> |

### Na tela

- ✅ *PATH* já estava como você pediu. <!-- textos.PathJaEstava -->
  - ‹diretório›
- ✅ *PATH* atualizado. <!-- textos.PathAtualizado -->
  - ‹diretório›
  - ‹aviso de *PATH* da plataforma, o mesmo do resumo de install›

### Erros

- Escolha exatamente um: `--add` ou `--remove`. <!-- textos.ErroAddOuRemove -->

## doctor

### Ajuda

| Flag | Texto |
|---|---|
| (resumo) | Diagnostica o ambiente. <!-- textos.ResumoDoctor --> |
| `--vault` | Nome do cofre no Obsidian ou caminho da raiz (obrigatório). <!-- textos.FlagVault --> |
| `--follow-symlinks` | Segue *symlink* dentro do cofre. <!-- textos.FlagFollowSymlinks --> |
| `--read-only` | Não verifica permissão de escrita. <!-- textos.FlagDoctorReadOnly --> |
| `--max-results` | Teto de resultados por consulta. <!-- textos.FlagMaxResults --> |
| `--fix` | Além de diagnosticar, remove o lixo de *runtime* e do cache. <!-- textos.FlagDoctorFix --> |

### Verificações

#### Diagnóstico do ambiente. <!-- textos.DoctorTitulo -->

As verificações saem em quatro grupos, cada um com seu título:

#### Cofre. <!-- textos.GrupoCofre -->

#### Cache e disco. <!-- textos.GrupoCache -->

#### *Daemon*. <!-- textos.GrupoDaemon -->

#### Windows. <!-- textos.GrupoWindows -->

Cada verificação é uma linha: o marcador do resultado e o nome. O detalhe vai
na mesma linha, apagado, quando o resultado é OK e o detalhe é curto; senão vai
na linha de baixo, indentado. Abaixo de cada nome, cada detalhe possível vem
precedido do marcador com que sai.

Grupo Cofre:

- Raiz do cofre existe. <!-- textos.CheckRaizExiste -->
  - ✅ Pelo nome %q, resolvido para %s. <!-- textos.DetPeloNome -->
  - ⚠️ Varredura interrompida: %v <!-- textos.DetVarreduraInterrompida -->
  - ❌ %q: %v <!-- textos.DetCaminhoEErroCitado -->
    - Existe(m) ao lado, com grafia diferente: %s <!-- textos.DetGrafiaVizinha -->
  - ❌ %q existe mas não é um diretório. <!-- textos.DetNaoEhDiretorio -->
- Permissão de leitura. <!-- textos.CheckLeitura -->
  - ✅ %d entradas na raiz. <!-- textos.DetEntradasRaiz -->
  - ❌ Não foi possível listar %q: %v <!-- textos.DetNaoListou -->
- Permissão de escrita. <!-- textos.CheckEscrita -->
  - ❌ Não foi possível escrever em %q: %v (⚠️ com `--read-only`) <!-- textos.DetNaoEscreveu -->
- `.obsidian` presente. <!-- textos.CheckObsidian -->
  - ⚠️ Pasta `.obsidian` ausente: configurações, temas e plugins do Obsidian não serão detectados. <!-- textos.DetObsidianAusente -->
  - ⚠️ Não foi possível verificar %q: %v <!-- textos.DetNaoVerificou -->
  - ⚠️ %q existe mas não é um diretório. <!-- textos.DetNaoEhDiretorio -->
- Contagem de notas. <!-- textos.CheckNotas -->
  - ✅ %d notas. <!-- textos.DetNotas -->
  - ⚠️ Nenhuma nota encontrada. <!-- textos.DetSemNotas -->
  - ❌ Cofre inacessível durante a varredura: %v <!-- textos.DetCofreInacessivel -->
- Comprimento de caminho. <!-- textos.CheckCaminho -->
  - ✅ Maior caminho: %d caracteres. <!-- textos.DetMaiorCaminho -->
  - ⚠️ %d caracteres. Acima do limiar de %d: %s <!-- textos.DetCaminhoLongo -->

Grupo Cache e disco:

- Diretório de cache. <!-- textos.CheckCache -->
  - ✅ Nenhum diretório de cache configurado. <!-- textos.DetSemCacheDir -->
  - ⚠️ Não foi possível criar %q: %v <!-- textos.DetNaoCriouCache -->
- Espaço em disco. <!-- textos.CheckEspaco -->
  - ✅ %d MB livres. (⚠️ ou ❌ quando pouco) <!-- textos.DetEspacoLivre -->
  - ⚠️ Não foi possível medir espaço livre: %v <!-- textos.DetNaoMediuEspaco -->

Grupo *Daemon*:

- Caminho do *socket* do *daemon*. <!-- textos.CheckSocket -->
  - ✅ %s -- %s. <!-- textos.DetSocketEClasse -->
  - ⚠️ Não foi possível derivar: %v <!-- textos.DetNaoDerivou -->
- Diretório de *sockets* aceita conexão. <!-- textos.CheckDiretorioSockets -->
  - ✅ Vale neste processo. Num host, o servidor roda noutro contexto, e a prova lá é a linha `conectado ao daemon` no log dele. <!-- textos.DetSondaOK -->
  - ⚠️ %v -- A ponte deste contexto vai servir em processo em vez de usar o *daemon*. <!-- textos.DetSondaFalhou -->
- *Daemon* respondendo. <!-- textos.CheckDaemonVivo -->
  - ✅ *Handshake* completo. <!-- textos.DetHandshakeOK -->
  - ✅ Nenhum *daemon* rodando (a ponte servirá em processo). <!-- textos.DetSemDaemon -->
  - ⚠️ Arquivo existe mas o *handshake* falhou: %v <!-- textos.DetHandshakeFalhou -->
  - ⚠️ (caminho indisponível) <!-- textos.DetCaminhoIndisponivel -->
- Log do *daemon*. <!-- textos.CheckLogDaemon -->
  - ✅ Ainda não existe (nenhum *daemon* rodou para este cofre). <!-- textos.DetLogAusente -->
  - ✅ %s (%d bytes, última escrita há %s). <!-- textos.DetLogResumo -->
    - | ‹uma das três últimas linhas do log›
  - ⚠️ %s: %v <!-- textos.DetCaminhoEErro -->
  - ⚠️ Não foi possível derivar: %v <!-- textos.DetNaoDerivou -->
- Travas de *daemon* em uso. <!-- textos.CheckTravas -->
  - ✅ Nenhuma trava em uso. <!-- textos.DetSemTravas -->
  - ✅ Diretório de *runtime* ainda não existe. <!-- textos.DetRuntimeAusente -->
  - ✅ %d em %s: %s <!-- textos.DetTravasEmUso -->
  - Cada trava da lista: %s (PID %d) <!-- textos.DetTravaComPID -->
  - Cada trava da lista: %s (não foi possível consultar: %v) <!-- textos.DetTravaIlegivel -->
  - ⚠️ %s: %v <!-- textos.DetCaminhoEErro -->
  - ⚠️ Não foi possível derivar: %v <!-- textos.DetNaoDerivou -->

O segundo %s de "Caminho do *socket* do *daemon*" diz o que existe no caminho:

- ausente <!-- textos.ClasseAusente -->
- *socket* <!-- textos.ClasseSocket -->
- DIRETÓRIO (nenhum *daemon* consegue usar este caminho) <!-- textos.ClasseDiretorio -->
- *symlink* <!-- textos.ClasseSymlink -->
- arquivo comum de %d bytes (resíduo, nenhum *daemon* escuta aqui) <!-- textos.ClasseArquivo -->
- outro (modo=%v) <!-- textos.ClasseOutro -->
- inacessível (%v) <!-- textos.ClasseInacessivel -->

Grupo Windows (só no Windows):

- Caminhos longos habilitados. <!-- textos.CheckCaminhosLongos -->
  - ⚠️ `LongPathsEnabled` != 1 no registro e há caminho de %d caracteres: %s <!-- textos.DetLongPaths -->
- Arquivos somente-nuvem. <!-- textos.CheckSomenteNuvem -->
  - ⚠️ %d nota(s) ainda não baixada(s) pelo sincronizador de nuvem. <!-- textos.DetSomenteNuvem -->
- Colisões de *casing*. <!-- textos.CheckCasing -->
  - ⚠️ %d colisão(ões): %s <!-- textos.DetColisoes -->

As verificações que dependem da varredura (contagem de notas, comprimento de
caminho e as três do Windows) podem sair também com "Varredura interrompida: %v"
ou "Cofre inacessível durante a varredura: %v".

Depois dos grupos, a contagem:

- ℹ️ %d verificações: %s, %s, %s <!-- textos.DoctorResumo -->

Cada %s é um número seguido do marcador do estado, o mesmo que abre as linhas:
✅, ⚠️ e ❌. Onde o console não aguenta emoji, aviso e falha seriam os dois
"[!]", e a contagem sai por extenso:

- ok <!-- textos.DoctorResumoOK -->
- aviso <!-- textos.DoctorResumoAviso -->
- avisos <!-- textos.DoctorResumoAvisos -->
- falha <!-- textos.DoctorResumoFalha -->
- falhas <!-- textos.DoctorResumoFalhas -->

### Processos

- ⚠️ Processos e lixo: diretório de *runtime* indisponível (%v). <!-- textos.DoctorRuntimeIndisponivel -->
- ⚠️ Processos do **gobsidian**: %v <!-- textos.DoctorProcessosErro -->
- ✅ Processos do **gobsidian**. <!-- textos.DoctorProcessos -->
  - Nenhum rodando. <!-- textos.DoctorNenhumProcesso -->
- ✅ %d processo(s) do **gobsidian**. <!-- textos.DoctorProcessosContagem -->

Bloco sem título, uma linha por cofre: ‹cofre›  ‹n› ‹modo›, ...  ‹versões›.
O cofre pode ser:

- (sem cofre registrado) <!-- textos.DoctorSemCofre -->

E o modo, no singular e no plural:

- *daemon* <!-- textos.DoctorModoDaemon -->
- *daemons* <!-- textos.DoctorModoDaemons -->
- ponte <!-- textos.DoctorModoPonte -->
- pontes <!-- textos.DoctorModoPontes -->
- servidor em processo <!-- textos.DoctorModoEmProcesso -->
- servidores em processo <!-- textos.DoctorModoEmProcessos -->
- modo não registrado <!-- textos.DoctorModoNaoRegistrado -->

*Um servidor por sessão do host. O daemon é um só por cofre.* <!-- textos.DoctorProcessosRodape -->

- ⚠️ %d processos gravam o cache do mesmo cofre. <!-- textos.DoctorGravadoresDuplos -->
  - %s -- gravadores: %s. Encerre os extras. Pontes não gravam e ficam fora desta conta. <!-- textos.DoctorGravadoresDetalhe; a lista é textos.DoctorPid separado por vírgula -->
  - %s -- %s sem modo registrado (versão anterior): não dá para saber se gravam. <!-- textos.DoctorSemModoDetalhe -->

### Processos sem presença

- Processos do **gobsidian** sem presença: não verificado nesta plataforma. (linha de detalhe) <!-- textos.DoctorSemPresencaPlataforma -->
- ⚠️ Processos do **gobsidian** sem presença: %v <!-- textos.DoctorSemPresencaErro -->
- ✅ Processos do **gobsidian** sem presença. <!-- textos.DoctorSemPresencaNenhum -->
  - Nenhum. <!-- textos.DoctorSemPresencaVazio -->
- ⚠️ %d processo(s) do **gobsidian** sem presença. <!-- textos.DoctorSemPresencaContagem -->

Bloco sem título: ‹quantidade›  ‹executável›, e abaixo "pid ‹n›, ‹n›" <!-- textos.DoctorPids -->

*Binário anterior à presença, ou de outra instalação. O doctor não sabe o modo nem o cofre deles.* <!-- textos.DoctorSemPresencaRodape -->

### Binário dos hosts

- Binário dos hosts: sem instalação registrada, nada a comparar. (linha de detalhe) <!-- textos.DoctorHostsSemManifesto -->
- ⚠️ Binário dos hosts: %v <!-- textos.DoctorHostsErro -->
- ✅ Binário dos hosts. <!-- textos.DoctorHostsOK -->
  - Toda entrada do **gobsidian** nos configs de arquivo roda %s. <!-- textos.DoctorHostsDetalhe -->
- ⚠️ %d entrada(s) de host rodam outro binário. <!-- textos.DoctorHostsOutroBinario -->

Bloco sem título: ‹host›  ‹chave›  ‹versão›  ‹comando›. A versão pode ser:

- versão não medida <!-- textos.DoctorVersaoNaoMedida -->
- comando não encontrado <!-- textos.DoctorComandoAusente -->

*O instalado é %s (%s). `gobsidian install` reconfigura. Claude Code, Gemini CLI, Codex e VS Code guardam a configuração no próprio CLI e não entram aqui.* <!-- textos.DoctorHostsRodape -->

### Chaves de cache

- ⚠️ Chaves de cache: %v <!-- textos.DoctorChavesErro -->
- ⚠️ %d cache(s) sob chave superada. <!-- textos.DoctorChavesSuperada -->

Bloco sem título: ‹chave antiga› -> ‹chave nova›  ‹cofre›

*Rode `gobsidian update` para renomear com tudo encerrado.* <!-- textos.DoctorChavesRodape -->

### Lixo

- ⚠️ Lixo do diretório de *runtime*: %v <!-- textos.DoctorLixoErro -->
- ✅ Lixo de execuções anteriores. <!-- textos.DoctorLixoNenhum -->
  - Nada a remover. <!-- textos.DoctorLixoNadaARemover -->
- ⚠️ Lixo de execuções anteriores. <!-- textos.DoctorLixoTitulo -->

| Campo | Valor |
|---|---|
| Travas <!-- textos.DoctorLixoTravas --> | ‹n› |
| *Sockets* <!-- textos.DoctorLixoSockets --> | ‹n› |
| Presenças <!-- textos.DoctorLixoPresencas --> | ‹n› |
| Caches <!-- textos.DoctorLixoCaches --> | ‹n›  de cofre inexistente <!-- textos.DoctorLixoCachesNota --> |
| Logs rotacionados <!-- textos.DoctorLixoLogs --> | ‹n› |
| Total <!-- textos.DoctorLixoTotal --> | %d KB <!-- textos.DoctorLixoKB -->  removível <!-- textos.DoctorLixoRemovivel --> / removido <!-- textos.DoctorLixoRemovido --> |

  - Rode `gobsidian doctor --fix` para remover, ou `gobsidian update`, que já limpa. <!-- textos.DoctorLixoComoLimpar -->
  - Não removido: %s <!-- textos.DoctorLixoNaoRemovido -->

### Fim do relatório

- ❌ Há falhas bloqueantes acima. <!-- textos.DoctorFalhasAcima -->
- ✅ Ambiente apto. <!-- textos.DoctorAmbienteApto -->

## version

#### **gobsidian** ‹versão› <!-- textos.VersionTitulo -->

| Campo | Valor |
|---|---|
| *Commit* <!-- textos.CampoCommit --> | ‹commit› |
| *Build* <!-- textos.CampoBuild --> | ‹data de build› |

## search, index e inspect

### Flags de cofre e de cache

Registradas em todo comando que abre um cofre (serve, daemon, doctor, index,
search, inspect); as duas últimas, nos que leem ou gravam o cache.

| Flag | Texto |
|---|---|
| `--vault` | Nome do cofre no Obsidian ou caminho da raiz (obrigatório). <!-- textos.FlagVault --> |
| `--follow-symlinks` | Segue *symlink* dentro do cofre. <!-- textos.FlagFollowSymlinks --> |
| `--cache-dir` | Diretório do cache de índice. <!-- textos.FlagCacheDir --> |
| `--log-level` | Nível de log: `debug`, `info`, `warn` ou `error`. <!-- textos.FlagLogLevel --> |

### index

| Flag | Texto |
|---|---|
| (resumo) | Constrói o índice do cofre e exibe um resumo. <!-- textos.ResumoIndex --> |
| `--json` | Força a saída em JSON, mesmo no terminal. <!-- textos.FlagJSON --> |

- ✅ Indexação concluída em %d ms. <!-- textos.IndexConcluido -->

#### Índice. <!-- textos.IndexTitulo -->

| Campo | Valor |
|---|---|
| Origem <!-- textos.CampoOrigem --> | ‹cache ou build› |
| Notas <!-- textos.CampoNotas --> | ‹n› |
| Anexos <!-- textos.CampoAnexos --> | ‹n› |
| Tags <!-- textos.CampoTags --> | ‹n› |
| Tamanho <!-- textos.CampoTamanho --> | ‹n›  bytes <!-- textos.NotaBytes --> |

### search

| Flag | Texto |
|---|---|
| (resumo) | Busca um texto no cofre. <!-- textos.ResumoSearch --> |
| `--json` | Força a saída em JSON, mesmo no terminal. <!-- textos.FlagJSON --> |
| `--limit` | Limite máximo de resultados. <!-- textos.FlagSearchLimit --> |
| `--max-results` | Teto de resultados por consulta. <!-- textos.FlagMaxResults --> |

- ℹ️ Nenhum resultado para %q. <!-- textos.SearchSemResultado -->

#### %d de %d para %q. <!-- textos.SearchTitulo -->

- ‹caminho da nota›  ‹pontuação›
  - ... ‹trecho› ...

### inspect

| Flag | Texto |
|---|---|
| (resumo) | Exibe metadados, links e *backlinks* de uma nota. <!-- textos.ResumoInspect --> |
| `--json` | Força a saída em JSON, mesmo no terminal. <!-- textos.FlagJSON --> |

#### ‹caminho da nota›

| Campo | Valor |
|---|---|
| Título <!-- textos.CampoTitulo --> | ‹título› |
| Tamanho <!-- textos.CampoTamanho --> | ‹n›  bytes <!-- textos.NotaBytes --> |
| Tags <!-- textos.CampoTags --> | ‹tags›  (%d) |
| *Headings* <!-- textos.CampoHeadings --> | ‹headings›  (%d) |
| Links de saída <!-- textos.CampoLinksSaida --> | ‹n› |
| *Backlinks* <!-- textos.CampoBacklinks --> | ‹backlinks›  (%d) |

#### Erros

- Resolvendo nota %q: %w <!-- textos.ErroResolvendoNota -->
- Nota %q não encontrada no índice. <!-- textos.ErroNotaNaoIndexada -->

## serve e daemon

Os dois não escrevem nada para uma pessoa durante o serviço: o stdout de serve
pertence ao JSON-RPC, e o que eles têm a dizer vai para o log. O que um humano
lê deles é a ajuda e os erros de partida.

### Ajuda

| Flag | Texto |
|---|---|
| serve (resumo) | Serve o cofre via MCP sobre stdio. <!-- textos.ResumoServe --> |
| daemon (resumo) | Roda o *daemon* de cofre compartilhado (uso interno da ponte). <!-- textos.ResumoDaemon --> |
| `--read-only` | Desabilita toda a superfície de escrita. <!-- textos.FlagReadOnly --> |
| `--debounce-ms` | Janela de coalescência de eventos do *watcher*. <!-- textos.FlagDebounce --> |
| `--max-results` | Teto de resultados por consulta. <!-- textos.FlagMaxResults --> |
| `--eager-search` | Carrega o índice de busca no *boot* em vez de esperar a primeira `vault_search`. <!-- textos.FlagEagerSearch --> |
| `--idle-seconds` (daemon) | Segundos sem cliente conectado antes de o *daemon* encerrar (padrão: 15 minutos). <!-- textos.FlagIdleSeconds --> |

### Erros

- `--idle-seconds` precisa ser >= 1 (recebido %d). <!-- textos.ErroIdleSeconds -->
- Resolvendo caminho do log do *daemon*: %w <!-- textos.ErroLogCaminho -->
- Criando diretório do log do *daemon*: %w <!-- textos.ErroLogDiretorio -->
- Abrindo log do *daemon* %s: %w <!-- textos.ErroLogAbrir -->
- Abrindo *socket* do *daemon*: %w <!-- textos.ErroSocketDaemon -->

## completion

| Comando | Texto |
|---|---|
| completion (resumo) | Gera o script de autocompletar do shell. <!-- textos.ResumoCompletion --> |
| completion ‹shell› (resumo) | Gera o script de autocompletar para %s. <!-- textos.CompletarResumoShell --> |
| completion ‹shell› (descrição) | Gera o script de autocompletar para %s.<br><br>%s <!-- textos.CompletarDescricao --> |

Os subcomandos bash, zsh, fish e powershell são do cobra, com a ajuda dele, em
inglês. O segundo %s da descrição é, para cada shell que o cobra não cobre, o
texto abaixo. Ele sai como está, sem desenhar a marcação: o que vem depois dos
dois-pontos é para copiar, e a crase do tcsh faz parte do comando.

nushell <!-- textos.ShellNushell -->

```
Acrescente ao seu config.nu:

  let gobsidian_completer = {|spans| gobsidian _carapace nushell ...$spans | from json }
  $env.config.completions.external = { enable: true, completer: $gobsidian_completer }
```

elvish <!-- textos.ShellElvish -->

```
Acrescente ao seu rc.elv:

  eval (gobsidian completion elvish | slurp)
```

ion <!-- textos.ShellIon -->

```
Acrescente ao seu initrc:

  eval $(gobsidian completion ion)
```

oil <!-- textos.ShellOil -->

```
Acrescente ao seu oshrc:

  source <(gobsidian completion oil)
```

tcsh <!-- textos.ShellTcsh -->

```
Acrescente ao seu .tcshrc:

  eval `gobsidian completion tcsh`
```

xonsh <!-- textos.ShellXonsh -->

```
Acrescente ao seu .xonshrc:

  exec($(gobsidian completion xonsh))
```

cmd_clink <!-- textos.ShellClink -->

```
Salve a saída em um arquivo .lua dentro do diretório de scripts do clink.
```

bash_ble <!-- textos.ShellBashBLE -->

```
Acrescente ao seu .bashrc, DEPOIS de carregar o ble.sh:

  source <(gobsidian completion bash_ble)
```

### O que o shell mostra ao lado de cada valor

O shell mostra estes textos sem desenhar marcação nenhuma.

| Valor | Descrição |
|---|---|
| none (em `--hosts`) | Não configura nenhum host. <!-- textos.CompletarNenhumHost --> |
| ‹cofre› (em `--vault`) | Cofre do Obsidian. <!-- textos.CompletarCofre --> |
| ‹cofre aberto› (em `--vault`) | Aberto agora. <!-- textos.CompletarCofreAberto --> |
| debug (em `--log-level`) | Tudo, inclusive o que só interessa depurando. <!-- textos.CompletarLogDebug --> |
| info | O padrão. <!-- textos.CompletarLogInfo --> |
| warn | Só o que pede atenção. <!-- textos.CompletarLogWarn --> |
| error | Só falha. <!-- textos.CompletarLogError --> |

### Erros

- Gerando o script de %s: %w <!-- textos.ErroGerandoScript -->

## Erros genéricos

Todo erro que um comando devolve sai numa linha de falha, no stderr:

- ❌ %v

Os erros de `--vault` dado pelo nome do cofre, que qualquer comando que abre um
cofre pode devolver:

- O nome %q casa mais de um cofre do Obsidian: %s. Passe o caminho da raiz em `--vault`. <!-- textos.ErroCofreAmbiguo -->
- O nome %q é o cofre %s do Obsidian e também a pasta %s no diretório atual. Passe o caminho da raiz em `--vault`. <!-- textos.ErroCofreNomeEPasta -->
- %q não é uma pasta e o Obsidian não tem registro de cofres (%s). Passe o caminho da raiz do cofre em `--vault`. <!-- textos.ErroCofreSemRegistro -->
- Nenhum cofre do Obsidian se chama %q. Conhecidos: %s. Passe o nome de um deles ou o caminho da raiz em `--vault`. <!-- textos.ErroCofreDesconhecido -->
  - Quando não há nenhum conhecido, o segundo %s é: nenhum <!-- textos.CofreNenhumConhecido -->

Os erros de configuração (`--vault` ausente, `--log-level`, `--debounce-ms`,
`--max-results` e as variáveis GOBSIDIAN_*) estão em internal/config, fora de
textos; ver a seção seguinte. As mensagens de erro próprias do cobra (comando
desconhecido, flag desconhecida) saem em inglês e não são do projeto.

## Textos fora de internal/textos

A regra do pacote textos é que tudo o que o produto escreve na tela mora num
arquivo só. O que sobra fora dele são as mensagens de erro de contexto dos
pacotes de baixo nível: "lendo %s: %w", "gravando manifesto: %w". Elas quase
sempre saem no MEIO de uma linha, embrulhadas por outra, e por isso começam
com minúscula e não levam ponto; a primeira letra da linha de falha vira
maiúscula na hora de imprimir (cmd/gobsidian/main.go, comMaiuscula). config e
selfupdate são folhas do grafo e não podem importar textos.

Todas estão com acento e seguem as outras regras de redação. Editar uma delas
é editar o literal no arquivo da coluna Local.

Contagem: 107 literais, levantados de fmt.Errorf e errors.New em
2026-09-27. Literais só de formato (alinhamento de colunas, "%s: %v"),
nomes de flag e de comando, e mensagens de slog ficaram de fora.

### internal/instalar

| Local | Texto | Situação |
|---|---|---|
| internal/instalar/cofres.go:42 | lendo %s: %w | sem constante; embrulha outro erro e sai no meio da linha |
| internal/instalar/cofres.go:47 | %s não é um JSON válido: %w | sem constante; embrulha outro erro e sai no meio da linha |
| internal/instalar/instalar.go:127 | instalação recusada pelo usuário | sem constante |
| internal/instalar/instalar.go:152 | resolvendo o diretório de *runtime*: %w | sem constante; embrulha outro erro e sai no meio da linha |
| internal/instalar/instalar.go:180 | limpando: %w | sem constante; embrulha outro erro e sai no meio da linha |
| internal/instalar/instalar.go:192 | migrando chaves de cache: %w | sem constante; embrulha outro erro e sai no meio da linha |
| internal/instalar/instalar.go:210 | ajustando o *PATH*: %w | sem constante; embrulha outro erro e sai no meio da linha |
| internal/instalar/instalar.go:268 | listando processos: %w | sem constante; embrulha outro erro e sai no meio da linha |
| internal/instalar/instalar.go:309 | resolvendo o executável corrente: %w | sem constante; embrulha outro erro e sai no meio da linha |
| internal/instalar/instalar.go:320 | criando %s: %w | sem constante; embrulha outro erro e sai no meio da linha |
| internal/instalar/instalar.go:335 | tirando o binário antigo do caminho: %w | sem constante; embrulha outro erro e sai no meio da linha |
| internal/instalar/instalar.go:364 | abrindo %s: %w | sem constante; embrulha outro erro e sai no meio da linha |
| internal/instalar/instalar.go:372 | criando %s: %w | sem constante; embrulha outro erro e sai no meio da linha |
| internal/instalar/instalar.go:376 | copiando para %s: %w | sem constante; embrulha outro erro e sai no meio da linha |
| internal/instalar/instalar.go:379 | fechando %s: %w | sem constante; embrulha outro erro e sai no meio da linha |
| internal/instalar/instalar.go:387 | abrindo %s para somar: %w | sem constante; embrulha outro erro e sai no meio da linha |
| internal/instalar/instalar.go:392 | somando %s: %w | sem constante; embrulha outro erro e sai no meio da linha |
| internal/instalar/limpeza.go:132 | lendo raiz do cache %s: %w | sem constante; embrulha outro erro e sai no meio da linha |
| internal/instalar/limpeza.go:165 | lendo diretório de *runtime* %s: %w | sem constante; embrulha outro erro e sai no meio da linha |
| internal/instalar/manifesto.go:61 | lendo manifesto: %w | sem constante; embrulha outro erro e sai no meio da linha |
| internal/instalar/manifesto.go:65 | manifesto ilegível em %s: %w | sem constante; embrulha outro erro e sai no meio da linha |
| internal/instalar/manifesto.go:74 | criando diretório do manifesto: %w | sem constante; embrulha outro erro e sai no meio da linha |
| internal/instalar/manifesto.go:78 | serializando manifesto: %w | sem constante; embrulha outro erro e sai no meio da linha |
| internal/instalar/manifesto.go:81 | gravando manifesto: %w | sem constante; embrulha outro erro e sai no meio da linha |
| internal/instalar/manifesto.go:107 | resolvendo o executável corrente: %w | sem constante; embrulha outro erro e sai no meio da linha |
| internal/instalar/manifesto.go:112 | consultando o executável corrente: %w | sem constante; embrulha outro erro e sai no meio da linha |
| internal/instalar/migracao.go:70 | lendo raiz do cache %s: %w | sem constante; embrulha outro erro e sai no meio da linha |
| internal/instalar/path_other.go:28 | resolvendo o home do usuário: %w | sem constante; embrulha outro erro e sai no meio da linha |
| internal/instalar/path_other.go:46 | lendo %s: %w | sem constante; embrulha outro erro e sai no meio da linha |
| internal/instalar/path_other.go:58 | abrindo %s: %w | sem constante; embrulha outro erro e sai no meio da linha |
| internal/instalar/path_other.go:62 | gravando em %s: %w | sem constante; embrulha outro erro e sai no meio da linha |
| internal/instalar/path_other.go:79 | lendo %s: %w | sem constante; embrulha outro erro e sai no meio da linha |
| internal/instalar/path_other.go:103 | lendo %s: %w | sem constante; embrulha outro erro e sai no meio da linha |
| internal/instalar/path_other.go:111 | gravando %s: %w | sem constante; embrulha outro erro e sai no meio da linha |
| internal/instalar/path_windows.go:35 | abrindo HKCU\%s: %w | sem constante; embrulha outro erro e sai no meio da linha |
| internal/instalar/path_windows.go:41 | lendo o *PATH* do usuário: %w | sem constante; embrulha outro erro e sai no meio da linha |
| internal/instalar/path_windows.go:63 | gravando o *PATH* do usuário: %w | sem constante; embrulha outro erro e sai no meio da linha |
| internal/instalar/path_windows.go:72 | abrindo HKCU\%s: %w | sem constante; embrulha outro erro e sai no meio da linha |
| internal/instalar/path_windows.go:81 | lendo o *PATH* do usuário: %w | sem constante; embrulha outro erro e sai no meio da linha |
| internal/instalar/path_windows.go:102 | gravando o *PATH* do usuário: %w | sem constante; embrulha outro erro e sai no meio da linha |
| internal/instalar/presenca.go:73 | criando diretório de *runtime*: %w | sem constante; embrulha outro erro e sai no meio da linha |
| internal/instalar/presenca.go:79 | travando presença: %w | sem constante; embrulha outro erro e sai no meio da linha |
| internal/instalar/presenca.go:85 | presença %s já está travada por outro processo | sem constante |
| internal/instalar/presenca.go:97 | serializando presença: %w | sem constante; embrulha outro erro e sai no meio da linha |
| internal/instalar/presenca.go:101 | gravando presença: %w | sem constante; embrulha outro erro e sai no meio da linha |
| internal/instalar/presenca.go:223 | lendo diretório de *runtime* %s: %w | sem constante; embrulha outro erro e sai no meio da linha |
| internal/instalar/processos.go:34 | listagem de processos não verificada nesta plataforma | sem constante |
| internal/instalar/processos_windows.go:31 | listando processos: %w | sem constante; embrulha outro erro e sai no meio da linha |
| internal/instalar/processos_windows.go:49 | percorrendo processos: %w | sem constante; embrulha outro erro e sai no meio da linha |
| internal/instalar/trava_global.go:48 | travando instalação: %w | sem constante; embrulha outro erro e sai no meio da linha |
| internal/instalar/trava_global.go:51 | já há uma instalação em curso (%s) | sem constante |

### internal/hosts

| Local | Texto | Situação |
|---|---|---|
| internal/hosts/hosts.go:58 | %s: %w (%s) | sem constante |
| internal/hosts/hosts.go:256 | host %s não sabe se configurar | sem constante |
| internal/hosts/merge.go:96 | criando diretório de %s: %w | sem constante; embrulha outro erro e sai no meio da linha |
| internal/hosts/merge.go:108 | lendo %s: %w | sem constante; embrulha outro erro e sai no meio da linha |
| internal/hosts/merge.go:120 | %s não é um JSON válido. Nada foi alterado (*backup* em %s%s): %w | sem constante; embrulha outro erro e sai no meio da linha |
| internal/hosts/merge.go:128 | `mcpServers` de %s não é um objeto. Nada foi alterado: %w | sem constante; embrulha outro erro e sai no meio da linha |
| internal/hosts/merge.go:142 | serializando a entrada %q: %w | sem constante; embrulha outro erro e sai no meio da linha |
| internal/hosts/merge.go:149 | serializando `mcpServers`: %w | sem constante; embrulha outro erro e sai no meio da linha |
| internal/hosts/merge.go:155 | serializando %s: %w | sem constante; embrulha outro erro e sai no meio da linha |
| internal/hosts/merge.go:161 | gravando %s: %w | sem constante; embrulha outro erro e sai no meio da linha |
| internal/hosts/merge.go:171 | gravando *backup* de %s: %w | sem constante; embrulha outro erro e sai no meio da linha |
| internal/hosts/merge.go:188 | serializando a definição para o VS Code: %w | sem constante; embrulha outro erro e sai no meio da linha |
| internal/hosts/merge.go:210 | lendo %s: %w | sem constante; embrulha outro erro e sai no meio da linha |

### internal/selfupdate

| Local | Texto | Situação |
|---|---|---|
| internal/selfupdate/selfupdate.go:62 | host fora da lista permitida | folha: não importa textos |
| internal/selfupdate/selfupdate.go:67 | SHA-256 do arquivo baixado não confere com o publicado | folha: não importa textos |
| internal/selfupdate/selfupdate.go:84 | %w: %q (permitidos: %s) | folha: não importa textos |
| internal/selfupdate/selfupdate.go:129 | lendo resposta da API de *releases*: %w | folha: não importa textos; embrulha outro erro e sai no meio da linha |
| internal/selfupdate/selfupdate.go:132 | a API devolveu um *release* sem *tag* | folha: não importa textos |
| internal/selfupdate/selfupdate.go:160 | *release* %s não tem o ativo %q | folha: não importa textos |
| internal/selfupdate/selfupdate.go:175 | criando diretório de destino: %w | folha: não importa textos; embrulha outro erro e sai no meio da linha |
| internal/selfupdate/selfupdate.go:179 | criando temporário de *download*: %w | folha: não importa textos; embrulha outro erro e sai no meio da linha |
| internal/selfupdate/selfupdate.go:187 | gravando *download*: %w | folha: não importa textos; embrulha outro erro e sai no meio da linha |
| internal/selfupdate/selfupdate.go:190 | fechando *download*: %w | folha: não importa textos; embrulha outro erro e sai no meio da linha |
| internal/selfupdate/selfupdate.go:195 | %w: esperado %s, obtido %s | folha: não importa textos |
| internal/selfupdate/selfupdate.go:199 | movendo *download* para %s: %w | folha: não importa textos; embrulha outro erro e sai no meio da linha |
| internal/selfupdate/selfupdate.go:216 | *release* %s não publica %s. Sem ele, não há o que conferir | folha: não importa textos |
| internal/selfupdate/selfupdate.go:226 | lendo %s: %w | folha: não importa textos; embrulha outro erro e sai no meio da linha |
| internal/selfupdate/selfupdate.go:239 | %s não lista %q | folha: não importa textos |
| internal/selfupdate/transporte_http.go:48 | montando requisição para %s: %w | folha: não importa textos; embrulha outro erro e sai no meio da linha |
| internal/selfupdate/transporte_http.go:58 | buscando %s: %w | folha: não importa textos; embrulha outro erro e sai no meio da linha |
| internal/selfupdate/transporte_http.go:62 | buscando %s: status %d | folha: não importa textos |

### internal/config

| Local | Texto | Situação |
|---|---|---|
| internal/config/config.go:88 | caminho do cofre não informado: use `--vault` | folha: não importa textos |
| internal/config/config.go:92 | resolvendo caminho do cofre %q: %w | folha: não importa textos; embrulha outro erro e sai no meio da linha |
| internal/config/config.go:99 | GOBSIDIAN_LOG_LEVEL: %w | folha: não importa textos; embrulha outro erro e sai no meio da linha |
| internal/config/config.go:107 | --log-level: %w | folha: não importa textos; embrulha outro erro e sai no meio da linha |
| internal/config/config.go:116 | GOBSIDIAN_READ_ONLY: %w | folha: não importa textos; embrulha outro erro e sai no meio da linha |
| internal/config/config.go:127 | GOBSIDIAN_DEBOUNCE_MS: %w | folha: não importa textos; embrulha outro erro e sai no meio da linha |
| internal/config/config.go:133 | --debounce-ms: %w | folha: não importa textos; embrulha outro erro e sai no meio da linha |
| internal/config/config.go:141 | GOBSIDIAN_MAX_RESULTS: %w | folha: não importa textos; embrulha outro erro e sai no meio da linha |
| internal/config/config.go:147 | --max-results: %w | folha: não importa textos; embrulha outro erro e sai no meio da linha |
| internal/config/config.go:174 | nível de log desconhecido: %q (use `debug`, `info`, `warn` ou `error`) | folha: não importa textos |
| internal/config/config.go:189 | valor desconhecido: %q (use 1, true, t, yes, y, 0, false, f, no ou n) | folha: não importa textos |
| internal/config/config.go:198 | valor inválido %q (use um inteiro >= 1): %w | folha: não importa textos; embrulha outro erro e sai no meio da linha |
| internal/config/config.go:214 | valor inválido %d (use um inteiro >= 1) | folha: não importa textos |
| internal/config/config.go:222 | valor inválido %q (use um inteiro de 1 a %d): %w | folha: não importa textos; embrulha outro erro e sai no meio da linha |
| internal/config/config.go:232 | valor inválido %d (deve ser entre 1 e %d) | folha: não importa textos |

### internal/doctor

| Local | Texto | Situação |
|---|---|---|
| internal/doctor/checks.go:373 | abrindo cofre: %w | sem constante; embrulha outro erro e sai no meio da linha |
| internal/doctor/checks_other.go:34 | statfs(%q): %w | sem constante; embrulha outro erro e sai no meio da linha |
| internal/doctor/checks_windows.go:149 | resolvendo %q: %w | sem constante; embrulha outro erro e sai no meio da linha |
| internal/doctor/checks_windows.go:153 | convertendo %q: %w | sem constante; embrulha outro erro e sai no meio da linha |
| internal/doctor/checks_windows.go:158 | GetDiskFreeSpaceEx(%q): %w | sem constante; embrulha outro erro e sai no meio da linha |

### internal/console

| Local | Texto | Situação |
|---|---|---|
| internal/console/bruto_outros.go:16 | plataforma sem modo bruto de terminal | erro interno, trocado antes de chegar à tela |
| internal/console/bruto_unix.go:35 | a entrada não é um terminal | erro interno, trocado antes de chegar à tela |
| internal/console/bruto_windows.go:40 | a entrada não é um console | erro interno, trocado antes de chegar à tela |
| internal/console/selecao.go:28 | a entrada não é um terminal interativo | erro interno, trocado antes de chegar à tela |
| internal/console/selecao.go:35 | seleção cancelada | erro interno, trocado antes de chegar à tela |
