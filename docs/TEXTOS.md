# Textos da CLI

Este documento reúne tudo o que o comando gobsidian escreve na tela para uma
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

Num console que não aguenta UTF-8 (CP-850), os marcadores saem escritos
([OK], [!], [i], [*], [...]) e o acento cai na hora de imprimir; a redação é a
mesma. Os cabeçalhos de segundo e terceiro nível deste documento organizam as
telas e não aparecem na tela.

## gobsidian sem argumentos

Com o gobsidian já instalado, ou sem um terminal do outro lado, o comando
sozinho mostra a ajuda. Sem instalação e com terminal, ele se instala.

### Autoinstalação

- ⏳ O **gobsidian** ainda não está instalado nesta máquina <!-- textos.AutoinstalarNaoInstalado -->
  - vamos instalar em `%s` <!-- textos.AutoinstalarDestino -->
  - Para ver apenas a ajuda, rode `gobsidian --help` <!-- textos.AutoinstalarAjuda -->

Em seguida vem o fluxo de instalação (seção install abaixo).

### Ajuda (gobsidian --help)

#### Servidor MCP para cofres locais do Obsidian <!-- textos.ResumoRaiz -->

**Uso:** <!-- internal/console/cobra.go:26 -->

**Comandos disponiveis:** <!-- internal/console/cobra.go:29 -->

| Comando | Texto |
|---|---|
| completion | Gera o script de autocompletar do shell. <!-- internal/console/cobra.go:110 --> |
| doctor | Diagnostica o ambiente. <!-- textos.ResumoDoctor --> |
| help | Exibe a ajuda sobre qualquer comando. <!-- internal/console/cobra.go:108 --> |
| index | Constrói o índice do cofre e exibe um resumo. <!-- textos.ResumoIndex --> |
| inspect | Exibe metadados, links e backlinks de uma nota. <!-- textos.ResumoInspect --> |
| install | Instala o gobsidian. <!-- textos.ResumoInstall --> |
| path | Acrescenta ou remove o diretório de instalação do *PATH* do usuário. <!-- textos.ResumoPath --> |
| search | Busca um texto no cofre <!-- textos.ResumoSearch --> |
| serve (oculto) | Serve o cofre via MCP sobre stdio <!-- textos.ResumoServe --> |
| path | Acrescenta ou remove o diretório de instalação do *PATH* do usuário. <!-- textos.ResumoPath --> |
| search | Busca um texto no cofre. <!-- textos.ResumoSearch --> |
| update | Atualiza o **gobsidian**. <!-- textos.ResumoUpdate --> |
| vaults | Configura o **gobisidian** como MCP de hosts de IA. <!-- textos.ResumoVaults --> |
| version | Imprime versão. <!-- textos.ResumoVersion --> |
| daemon (oculto) | Roda o daemon de cofre compartilhado (uso interno da ponte). <!-- textos.ResumoDaemon --> |

**Flags:** <!-- internal/console/cobra.go:32 -->

| Flag | Texto |
|---|---|
| -h, --help | Exibe a ajuda do **gobsidian**. <!-- internal/console/cobra.go:116 --> |

Use "gobsidian [comando] --help" para detalhes de um comando. <!-- internal/console/cobra.go:35 -->

## install

### Ajuda

| Flag | Texto |
|---|---|
| (resumo) | Instala o **gobsidian**, ajusta o *PATH* e configura os hosts de IA. <!-- textos.ResumoInstall --> |
| (descrição) | Instala o executável no perfil do usuário, sem elevação, pergunta se deve acrescentar o diretório ao *PATH* e registrar o servidor nos hosts de IA detectados. <!-- textos.DescricaoInstall --> |
| --vault | Cofre a servir (padrão: listar cofres registrados no Obsidian para escolha). <!-- textos.FlagInstallVault --> |
| --install-dir | Diretório onde o **gobsidian** será instalado (padrão: `‹diretório padrão›`). <!-- textos.FlagInstallDir + internal/instalar DiretorioPadrao + ")" em cmd/gobsidian/install.go:42 --> |
| --hosts | Lista de hosts de IA (Claude, Codex, AGY, etc.) a configurar, separados por vírgula (`‹chaves dos hosts›`); 'none' não configura nenhum. <!-- textos.FlagInstallHosts + chaves + textos.FlagInstallHostsFim, cmd/gobsidian/install.go:44 --> |
| --yes | Não pergunta nada: instala, ajusta o PATH e configura todos os hosts de IA detectados. <!-- textos.FlagInstallYes --> |
| --read-only | Registra o servidor apenas para leitura. <!-- textos.FlagInstallReadOnly --> |
| --no-path | Não altera o *PATH* <!-- textos.FlagInstallNoPath --> |

### Sem terminal

- ⚠️ A entrada não é um terminal interativo, então serão utilizdas as opções padrões. <!-- textos.InstallSemTerminal -->
  - Para escolher os cofres, rode `gobsidian install` em um terminal interativo. <!-- textos.InstallSemTerminalDica -->

### Configuração existente

#### Configuração atual <!-- textos.InstallConfigAtual -->

- `‹caminho de cada cofre já configurado›`

**Manter esta configuração?** <!-- textos.InstallManterConfig -->

- [•Sim•] <!-- internal/console/confirmar.go:143 -->
- ⊏ Não ⊐ <!-- internal/console/confirmar.go:143 -->

**[←]/[→]** *mover* · **[s]/[n]** *responder* · **[⏎ enter]** *confirmar* <!-- internal/console/confirmar.go:145 -->

Sem terminal, a mesma pergunta é digitada:

**◆ Manter esta configuração? (S/n) ▸** <!-- textos.InstallManterConfig; sufixo S/n e s/N em cmd/gobsidian/install.go:605 e :607 -->

- ✓ Sim <!-- cmd/gobsidian/install.go:620 -->
- ✓ Não <!-- cmd/gobsidian/install.go:620 -->
- ✓ Cancelado <!-- cmd/gobsidian/install.go:592 -->

### Escolha dos cofres

- ⚠️ Não foi possível ler os cofres do Obsidian: %v <!-- textos.InstallRegistroIlegivel -->

**Quais cofres configurar?** <!-- textos.InstallQuaisCofres -->

- ▸ ▣ ‹nome do cofre› *‹caminho do cofre›*  <!-- cmd/gobsidian/install.go:259 -->
-   ▢ ‹nome do cofre› *‹caminho do cofre›*  <!-- cmd/gobsidian/install.go:262 -->
-   ■ ‹nome do cofre› *‹caminho do cofre›*  <!-- cmd/gobsidian/install.go:269 -->

**[↑]/[↓]** *mover* · **[ espaço ]** *marcar* · **[a]** *todos* · **[⏎ enter]** *confirmar* <!-- internal/console/selecao.go:248 -->

Sem terminal, a lista é numerada e a escolha é digitada:

#### Quais cofres configurar? <!-- textos.InstallQuaisCofres -->

- ‹n›) <nome> *‹caminho›* `‹nota›` <!-- textos.InstallItemNumerado: "%d) %s  %s" -->

**◆ Números separados por espaço, * para todos, vazio para nenhum ▸** <!-- textos.InstallEscolhaDigitada -->

Com --yes e nenhum cofre configurado antes:

- ℹ️ Cofre: %s <!-- textos.InstallCofreEscolhido -->

### Escolha dos hosts

- ℹ️ Nenhum host de IA conhecido foi detectado <!-- textos.InstallSemHosts -->

Com --yes:

#### Hosts de IA encontrados <!-- textos.InstallHostsEncontrados -->

- ‹nome de cada host detectado›

Com terminal:

**Em quais hosts registrar?** <!-- textos.InstallQuaisHosts -->

- ▸ ▣ ‹nome do host›  *‹chave do host›*
-   ▢ ‹nome do host›  *‹chave do host›*
-   ■ ‹nome do host›  *‹chave do host›*

*↑↓ mover · espaco marcar · a todos · ⏎ confirmar* <!-- internal/console/selecao.go:248 -->

Sem terminal:

#### Em quais hosts registrar? <!-- textos.InstallQuaisHosts -->

- ‹n›. ‹nome do host›  *‹chave›* <!-- textos.InstallItemHost: "%d) %s  (%s)" -->

**◆ Números separados por espaço, * para todos, vazio para nenhum (\*) ▸** <!-- textos.InstallEscolhaDigitada -->

Os nomes dos hosts estão no código, fora de textos: Claude Desktop, Claude Code
(CLI), Gemini CLI, Antigravity, Antigravity IDE, Codex CLI, VS Code, Cursor,
Windsurf. <!-- internal/hosts/hosts.go:102 a :215 -->

### Instalando

#### Instalando <!-- textos.InstallTitulo -->

- ◔ Tomando a trava de instalação <!-- textos.PassoTrava -->
- ◑ Procurando processos em execução <!-- textos.PassoProcessos -->
- ◕ Limpando lixo de execuções anteriores <!-- textos.PassoLimpeza -->
- ● Conferindo as chaves de cache <!-- textos.PassoChaves -->
- ◔ Instalando o binário <!-- textos.PassoBinario -->
- ◑ Ajustando o PATH <!-- textos.PassoPath -->
- ◕ Configurando os hosts de IA <!-- textos.PassoHosts -->
- ● Gravando o manifesto <!-- textos.PassoManifesto -->

Se houver processos do gobsidian rodando:

**Encerrar estes processos do gobsidian para trocar o binario?** <!-- internal/instalar/instalar.go:279 -->

- pid ‹pid›  ‹papel›  ‹cofre› <!-- internal/instalar/instalar.go:277 -->
- ⊏ Sim ⊐ [•Não•] <!-- internal/console/confirmar.go:143 -->

*←→ mover · s/n responder · ⏎ confirmar* <!-- internal/console/confirmar.go:145 -->

Com --yes, a pergunta sai como aviso e segue sem esperar:

- ⚠️ Encerrar estes processos do **gobsidian** para trocar o binário? <!-- internal/instalar/instalar.go:279 -->
  - pid ‹pid›  ‹papel›  ‹cofre› <!-- internal/instalar/instalar.go:277 -->
  - --yes: encerrando sem perguntar <!-- textos.InstallYesEncerrando -->

Se a resposta for não:

- ⚠️ Instalação cancelada. Nada foi alterado. <!-- textos.InstallCancelado -->

### Resumo

- ✅ Instalado <!-- textos.InstallConcluido -->

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

- Abra um terminal NOVO, ou rode o comando para atualizar o *PATH*. <!-- internal/instalar/path_other.go:116 -->

O aviso de cada host configurado com sucesso:

| Host | Aviso |
|---|---|
| Claude Desktop | Reinicie o Claude Desktop para carregar o servidor. <!-- textos.HostClaudeDesktop --> |
| Claude Code | Registrado no Claude Code. <!-- textos.HostClaudeCode --> |
| Gemini CLI | Registrado no Gemini CLI. <!-- textos.HostGeminiCLI --> |
| Antigravity | Reinicie o Antigravity. <!-- textos.HostAntigravity --> |
| Antigravity IDE | Reinicie o Antigravity IDE. <!-- textos.HostAntigravityIDE --> |
| Codex | Registrado em ~/.codex/config.toml. <!-- textos.HostCodex --> |
| VS Code | Registrado na configuração de usuário do VS Code. <!-- textos.HostVSCode --> |
| Cursor | Reinicie o Cursor. <!-- textos.HostCursor --> |
| Windsurf | Reinicie o Windsurf. <!-- textos.HostWindsurf --> |

### Erros

- nenhum cofre encontrado; passe --vault com o caminho do cofre <!-- textos.ErroSemCofre -->
- seleção cancelada; nada foi alterado <!-- textos.ErroSelecaoCancelada -->
- host desconhecido %q; conhecidos: %s <!-- textos.ErroHostDesconhecido -->
- escolha invalida: %q <!-- internal/console/selecao.go:345 -->
- escolha fora da lista: %d <!-- internal/console/selecao.go:348 -->

Os erros de internal/instalar e internal/hosts que podem chegar aqui estão na
última seção.

## vaults

### Ajuda

| Flag | Texto |
|---|---|
| (resumo) | Configura os hosts de IA para um cofre, sem reinstalar o binário <!-- textos.ResumoVaults --> |

As flags são as mesmas de install (--vault, --install-dir, --hosts, --yes,
--read-only, --no-path), com o mesmo texto.

### Na tela

A escolha de cofres e de hosts é a mesma de install. Depois:

- ✅ Configuração mantida; nada foi alterado <!-- textos.VaultsMantido -->

ou:

- ✅ Hosts configurados <!-- textos.InstallHostsConfigurados -->
  - cofres   nenhum configurado <!-- textos.InstallSemCofreNaLista -->
  - cofre    %s <!-- textos.InstallCofreNaLista -->
  - %-16s %s <!-- textos.InstallHostNaLista: chave do host e o aviso da tabela de install -->
- ⚠️ %s não pode ser configurado <!-- textos.InstallHostFalhou -->
  - ‹erro do host›

### Erros

- %w -- rode `gobsidian install` primeiro <!-- textos.ErroSemManifesto; o %w é "nao ha manifesto de instalacao", internal/instalar/manifesto.go:50 -->

## update

### Ajuda

| Flag | Texto |
|---|---|
| (resumo) | Atualiza o gobsidian para a última versão publicada <!-- textos.ResumoUpdate --> |
| (descrição) | Consulta a versão publicada, baixa o binário da plataforma corrente, CONFERE o SHA-256 publicado e -- só então -- encerra os processos em execução e troca o binário. Divergência de soma aborta sem instalar nada. <!-- textos.DescricaoUpdate --> |
| --check | só diz se há versão nova, sem baixar nem instalar <!-- textos.FlagUpdateCheck --> |
| --yes | não pergunta antes de encerrar os processos em execução <!-- textos.FlagUpdateYes --> |

### Na tela

- ⏳ Consultando a última versão publicada <!-- textos.UpdateConsultando -->
- ℹ️ instalada: %s <!-- textos.UpdateInstalada -->
- ℹ️ publicada: %s <!-- textos.UpdatePublicada -->
- ✅ Já está na última versão <!-- textos.UpdateJaAtual -->

Com --check e versão nova:

- ⚠️ Há versão nova: %s <!-- textos.UpdateHaVersaoNova -->
  - rode `gobsidian update` para instalar <!-- textos.UpdateComoInstalar -->

Sem --check:

- ⏳ Baixando %s e conferindo o SHA-256 <!-- textos.UpdateBaixando -->
- ❌ O binário baixado NÃO confere com a soma publicada <!-- textos.UpdateHashDiverge -->
  - nada foi instalado; sua instalação continua intacta <!-- textos.UpdateNadaMudou -->
- ✅ SHA-256 confere <!-- textos.UpdateHashConfere -->
- ⏳ Trocando o binário <!-- textos.UpdateTrocando -->

Seguem os passos, a pergunta sobre encerrar processos e o resumo, iguais aos de
install. Se a resposta for não:

- ⚠️ Atualização cancelada; nada foi alterado <!-- textos.UpdateCancelado -->

No fim:

- ✅ Atualizado para %s <!-- textos.UpdateConcluido -->
- (o resumo de install)
  - os hosts reiniciam o servidor sozinhos; não há o que fazer à mão <!-- textos.UpdateHostsSozinho -->

### Erros

- consultando releases: %w <!-- textos.ErroConsultarRelease -->
- o release %s não publica %q (plataforma %s/%s) <!-- textos.ErroAtivoAusente -->
- criando diretório temporário: %w <!-- textos.ErroTemporario -->
- %w -- rode `gobsidian install` primeiro <!-- textos.ErroSemManifesto -->

Os erros de internal/selfupdate (download, soma) estão na última seção.

## path

### Ajuda

| Flag | Texto |
|---|---|
| (resumo) | Acrescenta ou remove o diretório de instalação do PATH do usuário <!-- textos.ResumoPath --> |
| --add | acrescenta o diretório ao PATH <!-- textos.FlagPathAdd --> |
| --remove | remove o diretório do PATH <!-- textos.FlagPathRemove --> |

### Na tela

- ✅ PATH já estava como você pediu <!-- textos.PathJaEstava -->
  - ‹diretório›
- ✅ PATH atualizado <!-- textos.PathAtualizado -->
  - ‹diretório›
  - ‹aviso de PATH da plataforma, o mesmo do resumo de install›

### Erros

- escolha exatamente um: --add ou --remove <!-- textos.ErroAddOuRemove -->

## doctor

### Ajuda

| Flag | Texto |
|---|---|
| (resumo) | Diagnostica o ambiente: permissões, OneDrive, MAX_PATH, casing <!-- textos.ResumoDoctor --> |
| --vault | nome do cofre no Obsidian ou caminho da raiz (obrigatório) <!-- textos.FlagVault --> |
| --follow-symlinks | segue symlink dentro do cofre; o padrão recusa, porque o confinamento não alcança o alvo <!-- textos.FlagFollowSymlinks --> |
| --read-only | não verifica permissão de escrita <!-- textos.FlagDoctorReadOnly --> |
| --max-results | teto de resultados por consulta <!-- textos.FlagMaxResults --> |
| --fix | além de diagnosticar, remove o lixo comprovadamente órfão do diretório de runtime e do cache <!-- textos.FlagDoctorFix --> |

### Verificações

#### Diagnóstico do ambiente <!-- textos.DoctorTitulo -->

As verificações saem em quatro grupos, cada um com seu título:

#### Cofre <!-- internal/doctor/doctor.go:44 -->

#### Cache e disco <!-- internal/doctor/doctor.go:45 -->

#### Daemon <!-- internal/doctor/doctor.go:46 -->

#### Windows <!-- internal/doctor/doctor.go:47 -->

Cada verificação é uma linha: o marcador do resultado e o nome. O detalhe vai
na mesma linha, apagado, quando o resultado é OK e o detalhe é curto; senão vai
na linha de baixo, indentado. Abaixo de cada nome, cada detalhe possível vem
precedido do marcador com que sai.

**Atenção:** os nomes e os detalhes existem como constantes em textos, mas o
código ainda imprime os literais de internal/doctor. O texto abaixo é o das
constantes (com acento); onde o literal em uso difere, a diferença está na
última seção. Editar a constante hoje não muda a tela.

Grupo Cofre:

- raiz do cofre existe <!-- textos.CheckRaizExiste; em uso: internal/doctor/checks.go:38 -->
  - ✅ pelo nome %q, resolvido para %s <!-- internal/doctor/checks.go:87 -->
  - ⚠️ varredura interrompida: %v <!-- textos.DetVarreduraInterrompida; em uso: internal/doctor/checks.go:44 -->
  - ❌ %q: %v <!-- internal/doctor/checks.go:59 -->
    - existe(m) ao lado, com grafia diferente: %s <!-- textos.DetGrafiaVizinha; em uso: internal/doctor/checks.go:61 -->
  - ❌ %q existe mas não é um diretório <!-- textos.DetNaoEhDiretorio; em uso: internal/doctor/checks.go:73 -->
- permissão de leitura <!-- textos.CheckLeitura; em uso: internal/doctor/checks.go:92 -->
  - ✅ %d entradas na raiz <!-- textos.DetEntradasRaiz; em uso: internal/doctor/checks.go:106 -->
  - ❌ não foi possível listar %q: %v <!-- textos.DetNaoListou; em uso: internal/doctor/checks.go:103 -->
- permissão de escrita <!-- textos.CheckEscrita; em uso: internal/doctor/checks.go:115 -->
  - ❌ não foi possível escrever em %q: %v (⚠️ com --read-only) <!-- textos.DetNaoEscreveu; em uso: internal/doctor/checks.go:136 -->
- .obsidian presente <!-- textos.CheckObsidian; em uso: internal/doctor/checks.go:152 -->
  - ⚠️ pasta .obsidian ausente: configurações, temas e plugins do Obsidian não serão detectados <!-- textos.DetObsidianAusente; em uso: internal/doctor/checks.go:165 -->
  - ⚠️ não foi possível verificar %q: %v <!-- textos.DetNaoVerificou; em uso: internal/doctor/checks.go:171 -->
  - ⚠️ %q existe mas não é um diretório <!-- textos.DetNaoEhDiretorio; em uso: internal/doctor/checks.go:177 -->
- contagem de notas <!-- textos.CheckNotas; em uso: internal/doctor/checks.go:188 -->
  - ✅ %d notas <!-- textos.DetNotas; em uso: internal/doctor/checks.go:196 -->
  - ⚠️ nenhuma nota .md encontrada <!-- textos.DetSemNotas; em uso: internal/doctor/checks.go:194 -->
  - ❌ cofre inacessível durante a varredura: %v <!-- textos.DetCofreInacessivel; em uso: internal/doctor/checks.go:327 -->
- comprimento de caminho <!-- textos.CheckCaminho; em uso: internal/doctor/checks.go:206 -->
  - ✅ maior caminho: %d caracteres <!-- textos.DetMaiorCaminho; em uso: internal/doctor/checks.go:218 -->
  - ⚠️ %d caracteres, acima do limiar de %d: %s <!-- textos.DetCaminhoLongo; em uso: internal/doctor/checks.go:215 -->

Grupo Cache e disco:

- diretório de cache <!-- textos.CheckCache; em uso: internal/doctor/checks.go:224 -->
  - ✅ nenhum diretório de cache configurado <!-- textos.DetSemCacheDir; em uso: internal/doctor/checks.go:234 -->
  - ⚠️ não foi possível criar %q: %v <!-- textos.DetNaoCriouCache; em uso: internal/doctor/checks.go:241 -->
- espaço em disco <!-- textos.CheckEspaco; em uso: internal/doctor/checks.go:251 -->
  - ✅ %d MB livres (⚠️ ou ❌ quando pouco) <!-- textos.DetEspacoLivre; em uso: internal/doctor/checks.go:262 -->
  - ⚠️ não foi possível medir espaço livre: %v <!-- textos.DetNaoMediuEspaco; em uso: internal/doctor/checks.go:259 -->

Grupo Daemon:

- caminho do socket do daemon <!-- textos.CheckSocket; em uso: internal/doctor/daemon.go:79 -->
  - ✅ %s -- %s <!-- textos.DetSocketEClasse; em uso: internal/doctor/daemon.go:87 -->
  - ⚠️ não foi possível derivar: %v <!-- textos.DetNaoDerivou; em uso: internal/doctor/daemon.go:83 -->
- diretório de sockets aceita conexão <!-- textos.CheckDiretorioSockets; em uso: internal/doctor/daemon.go:113 -->
  - ✅ vale neste processo; num host o servidor roda noutro contexto, e a prova lá é a linha `conectado ao daemon` no log dele <!-- textos.DetSondaOK; em uso: internal/doctor/daemon.go:117 -->
  - ⚠️ %v -- a ponte deste contexto vai servir em processo em vez de usar o daemon <!-- textos.DetSondaFalhou; em uso: internal/doctor/daemon.go:115 -->
- daemon respondendo <!-- textos.CheckDaemonVivo; em uso: internal/doctor/daemon.go:130 -->
  - ✅ handshake completo <!-- textos.DetHandshakeOK; em uso: internal/doctor/daemon.go:135 -->
  - ✅ nenhum daemon rodando (a ponte servirá em processo) <!-- textos.DetSemDaemon; em uso: internal/doctor/daemon.go:148 -->
  - ⚠️ arquivo existe mas o handshake falhou: %v <!-- textos.DetHandshakeFalhou; em uso: internal/doctor/daemon.go:154 -->
- log do daemon <!-- textos.CheckLogDaemon; em uso: internal/doctor/daemon.go:164 -->
  - ✅ ainda não existe (nenhum daemon rodou para este cofre) <!-- textos.DetLogAusente; em uso: internal/doctor/daemon.go:174 -->
  - ✅ %s (%d bytes, última escrita há %s) <!-- textos.DetLogResumo; em uso: internal/doctor/daemon.go:183 -->
    - | ‹uma das três últimas linhas do log› <!-- internal/doctor/daemon.go:185 -->
  - ⚠️ %s: %v <!-- textos.DetCaminhoEErro; em uso: internal/doctor/daemon.go:176 -->
  - ⚠️ não foi possível derivar: %v <!-- textos.DetNaoDerivou; em uso: internal/doctor/daemon.go:168 -->
- travas de daemon em uso <!-- textos.CheckTravas; em uso: internal/doctor/daemon.go:206 -->
  - ✅ nenhuma trava em uso <!-- textos.DetSemTravas; em uso: internal/doctor/daemon.go:248 -->
  - ✅ diretório de runtime ainda não existe <!-- textos.DetRuntimeAusente; em uso: internal/doctor/daemon.go:217 -->
  - ✅ %d em %s: %s <!-- textos.DetTravasEmUso; em uso: internal/doctor/daemon.go:253 -->
  - cada trava da lista: %s (PID %d) <!-- textos.DetTravaComPID; em uso: internal/doctor/daemon.go:241 -->
  - cada trava da lista: %s (não foi possível consultar: %v) <!-- textos.DetTravaIlegivel; em uso: internal/doctor/daemon.go:230 -->
  - ⚠️ %s: %v <!-- textos.DetCaminhoEErro; em uso: internal/doctor/daemon.go:219 -->
  - ⚠️ não foi possível derivar: %v <!-- textos.DetNaoDerivou; em uso: internal/doctor/daemon.go:210 -->

O segundo %s de "caminho do socket do daemon" diz o que existe no caminho:

- ausente <!-- textos.ClasseAusente; em uso: internal/doctor/daemon.go:55 -->
- socket <!-- textos.ClasseSocket; em uso: internal/doctor/daemon.go:62 -->
- DIRETÓRIO (nenhum daemon consegue usar este caminho) <!-- textos.ClasseDiretorio; em uso: internal/doctor/daemon.go:64 -->
- symlink <!-- textos.ClasseSymlink; em uso: internal/doctor/daemon.go:66 -->
- arquivo comum de %d bytes (resíduo; nenhum daemon escuta aqui) <!-- textos.ClasseArquivo; em uso: internal/doctor/daemon.go:68 -->
- outro (modo=%v) <!-- textos.ClasseOutro; em uso: internal/doctor/daemon.go:70 -->
- inacessível (%v) <!-- textos.ClasseInacessivel; em uso: internal/doctor/daemon.go:57 -->

Grupo Windows (só no Windows):

- caminhos longos habilitados <!-- textos.CheckCaminhosLongos; em uso: internal/doctor/checks_windows.go:65 -->
  - ⚠️ LongPathsEnabled != 1 no registro e há caminho de %d caracteres: %s <!-- textos.DetLongPaths; em uso: internal/doctor/checks_windows.go:78 -->
- arquivos somente-nuvem <!-- textos.CheckSomenteNuvem; em uso: internal/doctor/checks_windows.go:107 -->
  - ⚠️ %d nota(s) ainda não baixada(s) pelo sincronizador de nuvem <!-- textos.DetSomenteNuvem; em uso: internal/doctor/checks_windows.go:116 -->
- colisões de casing <!-- textos.CheckCasing; em uso: internal/doctor/checks_windows.go:127 -->
  - ⚠️ %d colisão(ões): %s <!-- textos.DetColisoes; em uso: internal/doctor/checks_windows.go:136 -->

As verificações que dependem da varredura (contagem de notas, comprimento de
caminho e as três do Windows) podem sair também com "varredura interrompida: %v"
ou "cofre inacessível durante a varredura: %v". <!-- internal/doctor/checks.go:321 e :327 -->

Depois dos grupos, a contagem:

- ℹ️ %d verificações: %s, %s, %s <!-- textos.DoctorResumo -->

Cada %s é um número seguido de uma destas palavras:

- ok <!-- textos.DoctorResumoOK -->
- aviso <!-- textos.DoctorResumoAviso -->
- avisos <!-- textos.DoctorResumoAvisos -->
- falha <!-- textos.DoctorResumoFalha -->
- falhas <!-- textos.DoctorResumoFalhas -->

### Processos

- ⚠️ processos e lixo: diretório de runtime indisponível (%v) <!-- textos.DoctorRuntimeIndisponivel -->
- ⚠️ processos do gobsidian: %v <!-- textos.DoctorProcessosErro -->
- ✅ processos do gobsidian <!-- textos.DoctorProcessos -->
  - nenhum rodando <!-- textos.DoctorNenhumProcesso -->
- ✅ %d processo(s) do gobsidian <!-- textos.DoctorProcessosContagem -->

Bloco sem título, uma linha por cofre: ‹cofre›  ‹n› ‹modo›, ...  ‹versões›.
O cofre pode ser:

- (sem cofre registrado) <!-- textos.DoctorSemCofre -->

E o modo, no singular e no plural:

- daemon <!-- textos.DoctorModoDaemon -->
- daemons <!-- textos.DoctorModoDaemons -->
- ponte <!-- textos.DoctorModoPonte -->
- pontes <!-- textos.DoctorModoPontes -->
- servidor em processo <!-- textos.DoctorModoEmProcesso -->
- servidores em processo <!-- textos.DoctorModoEmProcessos -->
- modo não registrado <!-- textos.DoctorModoNaoRegistrado -->

*um servidor por sessão do host; o daemon é um só por cofre* <!-- textos.DoctorProcessosRodape -->

- ⚠️ %d processos gravam o cache do mesmo cofre <!-- textos.DoctorGravadoresDuplos -->
  - %s -- gravadores: %s; encerre os extras. Pontes não gravam e ficam fora desta conta <!-- textos.DoctorGravadoresDetalhe; a lista é "pid %d, pid %d", cmd/gobsidian/doctor.go:415 -->
  - %s -- %s sem modo registrado (versão anterior): não dá para saber se gravam <!-- textos.DoctorSemModoDetalhe -->

### Processos sem presença

- processos do gobsidian sem presença: não verificado nesta plataforma (linha de detalhe) <!-- textos.DoctorSemPresencaPlataforma -->
- ⚠️ processos do gobsidian sem presença: %v <!-- textos.DoctorSemPresencaErro -->
- ✅ processos do gobsidian sem presença <!-- textos.DoctorSemPresencaNenhum -->
  - nenhum <!-- textos.DoctorSemPresencaVazio -->
- ⚠️ %d processo(s) do gobsidian sem presença <!-- textos.DoctorSemPresencaContagem -->

Bloco sem título: ‹quantidade›  ‹executável›, e abaixo "pid ‹n›, ‹n›" <!-- cmd/gobsidian/doctor.go:409 -->

*binário anterior à presença, ou de outra instalação -- o doctor não sabe o modo nem o cofre deles* <!-- textos.DoctorSemPresencaRodape -->

### Binário dos hosts

- binário dos hosts: sem instalação registrada, nada a comparar (linha de detalhe) <!-- textos.DoctorHostsSemManifesto -->
- ⚠️ binário dos hosts: %v <!-- textos.DoctorHostsErro -->
- ✅ binário dos hosts <!-- textos.DoctorHostsOK -->
  - toda entrada do gobsidian nos configs de arquivo roda %s <!-- textos.DoctorHostsDetalhe -->
- ⚠️ %d entrada(s) de host rodam outro binário <!-- textos.DoctorHostsOutroBinario -->

Bloco sem título: ‹host›  ‹chave›  ‹versão›  ‹comando›. A versão pode ser:

- versão não medida <!-- textos.DoctorVersaoNaoMedida -->
- comando não encontrado <!-- textos.DoctorComandoAusente -->

*o instalado é %s (%s); `gobsidian install` reconfigura. Claude Code, Gemini CLI, Codex e VS Code guardam a config no próprio CLI e não entram aqui* <!-- textos.DoctorHostsRodape -->

### Chaves de cache

- ⚠️ chaves de cache: %v <!-- textos.DoctorChavesErro -->
- ⚠️ %d cache(s) sob chave superada <!-- textos.DoctorChavesSuperada -->

Bloco sem título: ‹chave antiga› -> ‹chave nova›  ‹cofre› <!-- cmd/gobsidian/doctor.go:141 -->

*rode `gobsidian update` para renomear com tudo encerrado* <!-- textos.DoctorChavesRodape -->

### Lixo

- ⚠️ lixo do diretório de runtime: %v <!-- textos.DoctorLixoErro -->
- ✅ lixo de execuções anteriores <!-- textos.DoctorLixoNenhum -->
  - nada a remover <!-- textos.DoctorLixoNadaARemover -->
- ⚠️ lixo de execuções anteriores <!-- textos.DoctorLixoTitulo -->

| Campo | Valor |
|---|---|
| travas <!-- textos.DoctorLixoTravas --> | ‹n› |
| sockets <!-- textos.DoctorLixoSockets --> | ‹n› |
| presenças <!-- textos.DoctorLixoPresencas --> | ‹n› |
| caches <!-- textos.DoctorLixoCaches --> | ‹n›  de cofre inexistente <!-- textos.DoctorLixoCachesNota --> |
| logs rotacionados <!-- textos.DoctorLixoLogs --> | ‹n› |
| total <!-- textos.DoctorLixoTotal --> | %d KB <!-- cmd/gobsidian/doctor.go:177 -->  removível <!-- textos.DoctorLixoRemovivel --> / removido <!-- textos.DoctorLixoRemovido --> |

  - rode `gobsidian doctor --fix` para remover, ou `gobsidian update`, que já limpa <!-- textos.DoctorLixoComoLimpar -->
  - não removido: %s <!-- textos.DoctorLixoNaoRemovido -->

### Fim do relatório

- ❌ Há falhas bloqueantes acima <!-- textos.DoctorFalhasAcima -->
- ✅ Ambiente apto <!-- textos.DoctorAmbienteApto -->

## version

#### gobsidian ‹versão› <!-- cmd/gobsidian/main.go:100 -->

| Campo | Valor |
|---|---|
| commit <!-- textos.CampoCommit --> | ‹commit› |
| build <!-- textos.CampoBuild --> | ‹data de build› |

## search, index e inspect

### Flags de cofre e de cache

Registradas em todo comando que abre um cofre (serve, daemon, doctor, index,
search, inspect); as duas últimas, nos que leem ou gravam o cache.

| Flag | Texto |
|---|---|
| --vault | nome do cofre no Obsidian ou caminho da raiz (obrigatório) <!-- textos.FlagVault --> |
| --follow-symlinks | segue symlink dentro do cofre; o padrão recusa, porque o confinamento não alcança o alvo <!-- textos.FlagFollowSymlinks --> |
| --cache-dir | diretório do cache de índice <!-- textos.FlagCacheDir --> |
| --log-level | debug, info, warn ou error <!-- textos.FlagLogLevel --> |

### index

| Flag | Texto |
|---|---|
| (resumo) | Constrói o índice do cofre e exibe um resumo <!-- textos.ResumoIndex --> |
| --json | saída estruturada em formato JSON <!-- textos.FlagJSON --> |

- ✅ Indexação concluída em %d ms <!-- textos.IndexConcluido -->

#### Índice <!-- textos.IndexTitulo -->

| Campo | Valor |
|---|---|
| origem <!-- textos.CampoOrigem --> | ‹cache ou build› |
| notas <!-- textos.CampoNotas --> | ‹n› |
| anexos <!-- textos.CampoAnexos --> | ‹n› |
| tags <!-- textos.CampoTags --> | ‹n› |
| tamanho <!-- textos.CampoTamanho --> | ‹n›  bytes <!-- textos.NotaBytes --> |

### search

| Flag | Texto |
|---|---|
| (resumo) | Executa busca por texto completo no cofre <!-- textos.ResumoSearch --> |
| --json | saída estruturada em formato JSON <!-- textos.FlagJSON --> |
| --limit | limite máximo de resultados <!-- textos.FlagSearchLimit --> |
| --max-results | teto de resultados por consulta <!-- textos.FlagMaxResults --> |

- ℹ️ Nenhum resultado para %q <!-- textos.SearchSemResultado -->

#### %d de %d para %q <!-- cmd/gobsidian/search.go:89 -->

- ‹caminho da nota›  ‹pontuação›
  - ... ‹trecho› ... <!-- cmd/gobsidian/search.go:86 -->

### inspect

| Flag | Texto |
|---|---|
| (resumo) | Exibe metadados, links e backlinks de uma nota <!-- textos.ResumoInspect --> |
| --json | saída estruturada em formato JSON <!-- textos.FlagJSON --> |

#### ‹caminho da nota›

| Campo | Valor |
|---|---|
| título <!-- textos.CampoTitulo --> | ‹título› |
| tamanho <!-- textos.CampoTamanho --> | ‹n›  bytes <!-- textos.NotaBytes --> |
| tags <!-- textos.CampoTags --> | ‹tags›  (%d) <!-- cmd/gobsidian/inspect.go:115 --> |
| headings <!-- textos.CampoHeadings --> | ‹headings›  (%d) <!-- cmd/gobsidian/inspect.go:121 --> |
| links de saída <!-- textos.CampoLinksSaida --> | ‹n› |
| backlinks <!-- textos.CampoBacklinks --> | ‹backlinks›  (%d) <!-- cmd/gobsidian/inspect.go:128 --> |

#### Erros

- resolvendo nota %q: %w <!-- textos.ErroResolvendoNota -->
- nota %q não encontrada no índice <!-- textos.ErroNotaNaoIndexada -->

## serve e daemon

Os dois não escrevem nada para uma pessoa durante o serviço: o stdout de serve
pertence ao JSON-RPC, e o que eles têm a dizer vai para o log. O que um humano
lê deles é a ajuda e os erros de partida.

### Ajuda

**Atenção:** as flags --read-only, --debounce-ms, --eager-search e
--idle-seconds têm constante em textos, mas serve.go e daemon.go ainda
registram literais sem acento. O texto abaixo é o das constantes; editar a
constante hoje não muda a tela.

| Flag | Texto |
|---|---|
| serve (resumo) | Serve o cofre via MCP sobre stdio <!-- textos.ResumoServe --> |
| daemon (resumo) | Roda o daemon de cofre compartilhado (uso interno da ponte) <!-- textos.ResumoDaemon --> |
| --read-only | desabilita toda a superfície de escrita <!-- textos.FlagReadOnly; em uso: cmd/gobsidian/serve.go:46 e cmd/gobsidian/daemon.go:61 --> |
| --debounce-ms | janela de coalescência de eventos do watcher <!-- textos.FlagDebounce; em uso: cmd/gobsidian/serve.go:47 e cmd/gobsidian/daemon.go:62 --> |
| --max-results | teto de resultados por consulta <!-- textos.FlagMaxResults; em uso: literal em cmd/gobsidian/serve.go:48 e cmd/gobsidian/daemon.go:63 --> |
| --eager-search | carrega o índice de busca no boot em vez de esperar a primeira vault_search <!-- textos.FlagEagerSearch; em uso: cmd/gobsidian/serve.go:50 e cmd/gobsidian/daemon.go:65 --> |
| --idle-seconds (daemon) | segundos sem cliente conectado antes do daemon encerrar (decisão 3 da Task 92; padrão 15 minutos) <!-- textos.FlagIdleSeconds; em uso: cmd/gobsidian/daemon.go:67 --> |

### Erros

- --idle-seconds precisa ser >= 1 (recebido %d) <!-- textos.ErroIdleSeconds -->
- resolvendo caminho do log do daemon: %w <!-- textos.ErroLogCaminho; em uso: cmd/gobsidian/daemon.go:142 -->
- criando diretório do log do daemon: %w <!-- textos.ErroLogDiretorio; em uso: cmd/gobsidian/daemon.go:145 -->
- abrindo log do daemon %s: %w <!-- textos.ErroLogAbrir; em uso: cmd/gobsidian/daemon.go:150 -->
- abrindo socket do daemon: %w <!-- textos.ErroSocketDaemon; em uso: cmd/gobsidian/daemon.go:210 -->

## completion

| Comando | Texto |
|---|---|
| completion (resumo) | Gera o script de autocompletar do shell <!-- internal/console/cobra.go:110 --> |
| completion ‹shell› (resumo) | Gera o script de autocompletar para %s <!-- textos.CompletarResumoShell --> |
| completion ‹shell› (descrição) | Gera o script de autocompletar para %s.<br><br>%s <!-- textos.CompletarDescricao --> |

Os subcomandos bash, zsh, fish e powershell são do cobra, com a ajuda dele, em
inglês. O segundo %s da descrição é, para cada shell que o cobra não cobre:

**Atenção:** estes textos têm constante em textos, mas o código imprime os
literais de cmd/gobsidian/completion_extra.go, sem acento em "saída" e
"diretório". Editar a constante hoje não muda a tela.

nushell <!-- textos.ShellNushell; em uso: cmd/gobsidian/completion_extra.go:29 -->

```
Acrescente ao seu config.nu:

  let gobsidian_completer = {|spans| gobsidian _carapace nushell ...$spans | from json }
  $env.config.completions.external = { enable: true, completer: $gobsidian_completer }
```

elvish <!-- textos.ShellElvish; em uso: cmd/gobsidian/completion_extra.go:32 -->

```
Acrescente ao seu rc.elv:

  eval (gobsidian completion elvish | slurp)
```

ion <!-- textos.ShellIon; em uso: cmd/gobsidian/completion_extra.go:33 -->

```
Acrescente ao seu initrc:

  eval $(gobsidian completion ion)
```

oil <!-- textos.ShellOil; em uso: cmd/gobsidian/completion_extra.go:34 -->

```
Acrescente ao seu oshrc:

  source <(gobsidian completion oil)
```

tcsh <!-- textos.ShellTcsh; em uso: cmd/gobsidian/completion_extra.go:35 -->

```
Acrescente ao seu .tcshrc:

  eval `gobsidian completion tcsh`
```

xonsh <!-- textos.ShellXonsh; em uso: cmd/gobsidian/completion_extra.go:36 -->

```
Acrescente ao seu .xonshrc:

  exec($(gobsidian completion xonsh))
```

cmd_clink <!-- textos.ShellClink; em uso: cmd/gobsidian/completion_extra.go:37 -->

```
Salve a saída em um arquivo .lua dentro do diretório de scripts do clink.
```

bash_ble <!-- textos.ShellBashBLE; em uso: cmd/gobsidian/completion_extra.go:38 -->

```
Acrescente ao seu .bashrc, DEPOIS de carregar o ble.sh:

  source <(gobsidian completion bash_ble)
```

### O que o shell mostra ao lado de cada valor

| Valor | Descrição |
|---|---|
| none (em --hosts) | não configura nenhum host <!-- textos.CompletarNenhumHost --> |
| ‹cofre› (em --vault) | cofre do Obsidian <!-- textos.CompletarCofre --> |
| ‹cofre aberto› (em --vault) | aberto agora <!-- textos.CompletarCofreAberto --> |
| debug (em --log-level) | tudo, inclusive o que só interessa depurando <!-- textos.CompletarLogDebug --> |
| info | o padrão <!-- textos.CompletarLogInfo --> |
| warn | só o que pede atenção <!-- textos.CompletarLogWarn --> |
| error | só falha <!-- textos.CompletarLogError --> |

### Erros

- gerando o script de %s: %w <!-- textos.ErroGerandoScript -->

## Erros genéricos

Todo erro que um comando devolve sai numa linha de falha, no stderr:

- ❌ %v <!-- cmd/gobsidian/main.go:41 -->

Os erros de --vault dado pelo nome do cofre, que qualquer comando que abre um
cofre pode devolver:

- o nome %q casa mais de um cofre do Obsidian: %s; passe o caminho da raiz em --vault <!-- textos.ErroCofreAmbiguo -->
- o nome %q é o cofre %s do Obsidian e também a pasta %s no diretório atual; passe o caminho da raiz em --vault <!-- textos.ErroCofreNomeEPasta -->
- %q não é uma pasta e o Obsidian não tem registro de cofres (%s); passe o caminho da raiz do cofre em --vault <!-- textos.ErroCofreSemRegistro -->
- nenhum cofre do Obsidian se chama %q; conhecidos: %s. Passe o nome de um deles ou o caminho da raiz em --vault <!-- textos.ErroCofreDesconhecido -->
  - quando não há nenhum conhecido, o segundo %s é: nenhum <!-- textos.CofreNenhumConhecido -->

Os erros de configuração (--vault ausente, --log-level, --debounce-ms,
--max-results e as variáveis GOBSIDIAN_*) estão em internal/config, fora de
textos; ver a seção seguinte. As mensagens de erro próprias do cobra (comando
desconhecido, flag desconhecida) saem em inglês e não são do projeto.

## Textos fora de internal/textos

A regra do pacote textos é que tudo o que o produto escreve na tela mora num
arquivo só. Os textos abaixo estão escritos direto no código. Na coluna
Situação, "duplicata" quer dizer que a constante existe em textos mas o código
não a usa: a tela mostra o literal, e editar a constante não muda nada.
Contagem: 245 linhas nas tabelas abaixo, cobrindo 250 literais em 32 arquivos.

Literais só de formato (alinhamento de colunas, como "  %-24s %-34s %s"),
nomes de flag e de comando, e mensagens de slog ficaram de fora.

### internal/console

| Local | Texto | Situação |
|---|---|---|
| internal/console/cobra.go:26 | Uso: | sem constante |
| internal/console/cobra.go:29 | Comandos disponiveis: | sem constante; sem acento |
| internal/console/cobra.go:32 | Flags: | sem constante |
| internal/console/cobra.go:35 | Use "%s [comando] --help" para detalhes de um comando. | sem constante |
| internal/console/cobra.go:108 | Exibe a ajuda sobre qualquer comando | sem constante |
| internal/console/cobra.go:110 | Gera o script de autocompletar do shell | sem constante |
| internal/console/cobra.go:116 | Exibe a ajuda do ‹nome do comando› | sem constante |
| internal/console/confirmar.go:143 | Sim | sem constante |
| internal/console/confirmar.go:143 | Não | sem constante |
| internal/console/confirmar.go:145 | ←→ mover · s/n responder · ⏎ confirmar | sem constante (as setas, o separador e o ⏎ são glifos; "mover", "s/n responder" e "confirmar" são texto) |
| internal/console/selecao.go:248 | ↑↓ mover · espaco marcar · a todos · ⏎ confirmar | sem constante; sem acento em "espaço" |
| internal/console/selecao.go:26 | a entrada nao e um terminal interativo | erro interno, não chega à tela hoje |
| internal/console/selecao.go:33 | selecao cancelada | erro interno, trocado por ErroSelecaoCancelada antes de chegar à tela |
| internal/console/selecao.go:345 | escolha invalida: %q | sem constante; sem acento |
| internal/console/selecao.go:348 | escolha fora da lista: %d | sem constante |
| internal/console/estilo.go:82 | glifos em texto: "enter", "setas" (conjunto ASCII do rodapé) | sem constante |

### cmd/gobsidian

| Local | Texto | Situação |
|---|---|---|
| cmd/gobsidian/main.go:41 | %v (a linha de falha de todo erro) | formato |
| cmd/gobsidian/main.go:100 | gobsidian ‹versão› (título de version) | sem constante |
| cmd/gobsidian/install.go:42 | ")" que fecha a ajuda de --install-dir | metade do texto em textos, metade aqui |
| cmd/gobsidian/install.go:44 | ", " entre as chaves de --hosts | formato |
| cmd/gobsidian/install.go:259 | (aberto agora) | sem constante; textos.CompletarCofreAberto tem o mesmo sentido, sem parênteses |
| cmd/gobsidian/install.go:262 | (ja configurado) | sem constante; sem acento |
| cmd/gobsidian/install.go:269 | (ja configurado, fora do Obsidian) | sem constante; sem acento |
| cmd/gobsidian/install.go:592 | cancelado | sem constante |
| cmd/gobsidian/install.go:605 | s/N | sem constante |
| cmd/gobsidian/install.go:607 | S/n | sem constante |
| cmd/gobsidian/install.go:618 | s, sim, y, yes (respostas aceitas) | sem constante |
| cmd/gobsidian/install.go:620 | sim | sem constante |
| cmd/gobsidian/install.go:620 | nao | sem constante; sem acento |
| cmd/gobsidian/serve.go:46 | desabilita toda a superficie de escrita | duplicata de textos.FlagReadOnly; sem acento |
| cmd/gobsidian/serve.go:47 | janela de coalescencia de eventos do watcher | duplicata de textos.FlagDebounce; sem acento |
| cmd/gobsidian/serve.go:48 | teto de resultados por consulta | duplicata de textos.FlagMaxResults |
| cmd/gobsidian/serve.go:50 | carrega o indice de busca no boot em vez de esperar a primeira vault_search | duplicata de textos.FlagEagerSearch; sem acento |
| cmd/gobsidian/daemon.go:61 | desabilita toda a superficie de escrita | duplicata de textos.FlagReadOnly; sem acento |
| cmd/gobsidian/daemon.go:62 | janela de coalescencia de eventos do watcher | duplicata de textos.FlagDebounce; sem acento |
| cmd/gobsidian/daemon.go:63 | teto de resultados por consulta | duplicata de textos.FlagMaxResults |
| cmd/gobsidian/daemon.go:65 | carrega o indice de busca no boot em vez de esperar a primeira vault_search | duplicata de textos.FlagEagerSearch; sem acento |
| cmd/gobsidian/daemon.go:67 | segundos sem cliente conectado antes do daemon encerrar (decisao 3 da Task 92; padrao 15 minutos) | duplicata de textos.FlagIdleSeconds; sem acento |
| cmd/gobsidian/daemon.go:142 | resolvendo caminho do log do daemon: %w | duplicata de textos.ErroLogCaminho |
| cmd/gobsidian/daemon.go:145 | criando diretorio do log do daemon: %w | duplicata de textos.ErroLogDiretorio; sem acento |
| cmd/gobsidian/daemon.go:150 | abrindo log do daemon %s: %w | duplicata de textos.ErroLogAbrir |
| cmd/gobsidian/daemon.go:210 | abrindo socket do daemon: %w | duplicata de textos.ErroSocketDaemon |
| cmd/gobsidian/completion_extra.go:29 | Acrescente ao seu config.nu: ... (nushell) | duplicata de textos.ShellNushell |
| cmd/gobsidian/completion_extra.go:32 | Acrescente ao seu rc.elv: ... (elvish) | duplicata de textos.ShellElvish |
| cmd/gobsidian/completion_extra.go:33 | Acrescente ao seu initrc: ... (ion) | duplicata de textos.ShellIon |
| cmd/gobsidian/completion_extra.go:34 | Acrescente ao seu oshrc: ... (oil) | duplicata de textos.ShellOil |
| cmd/gobsidian/completion_extra.go:35 | Acrescente ao seu .tcshrc: ... (tcsh) | duplicata de textos.ShellTcsh |
| cmd/gobsidian/completion_extra.go:36 | Acrescente ao seu .xonshrc: ... (xonsh) | duplicata de textos.ShellXonsh |
| cmd/gobsidian/completion_extra.go:37 | Salve a saida em um arquivo .lua dentro do diretorio de scripts do clink. | duplicata de textos.ShellClink; sem acento |
| cmd/gobsidian/completion_extra.go:38 | Acrescente ao seu .bashrc, DEPOIS de carregar o ble.sh: ... (bash_ble) | duplicata de textos.ShellBashBLE |
| cmd/gobsidian/search.go:86 | ... ‹trecho› ... | formato com reticências |
| cmd/gobsidian/search.go:89 | %d de %d para %q | sem constante |
| cmd/gobsidian/inspect.go:115 | (%d) | formato |
| cmd/gobsidian/inspect.go:121 | (%d) | formato |
| cmd/gobsidian/inspect.go:128 | (%d) | formato |
| cmd/gobsidian/doctor.go:141 | %s -> %s  %s | formato |
| cmd/gobsidian/doctor.go:177 | %d KB | sem constante |
| cmd/gobsidian/doctor.go:287 e :289 | %d %s (número e palavra da contagem) | formato |
| cmd/gobsidian/doctor.go:409 | pid ‹n›, ‹n› | sem constante |
| cmd/gobsidian/doctor.go:415 | pid %d | sem constante |

### internal/doctor

Todos os nomes e detalhes abaixo têm constante em textos (Check*, Det*,
Classe*) que o código não usa, exceto onde a Situação diz "sem constante".

| Local | Texto | Situação |
|---|---|---|
| internal/doctor/doctor.go:44 | Cofre | sem constante (título de grupo) |
| internal/doctor/doctor.go:45 | Cache e disco | sem constante (título de grupo) |
| internal/doctor/doctor.go:46 | Daemon | sem constante (título de grupo) |
| internal/doctor/doctor.go:47 | Windows | sem constante (título de grupo; fora do Windows o grupo não tem verificação e não aparece) |
| internal/doctor/checks.go:38 | raiz do cofre existe | duplicata de textos.CheckRaizExiste |
| internal/doctor/checks.go:44 | varredura interrompida: %v | duplicata de textos.DetVarreduraInterrompida |
| internal/doctor/checks.go:59 | %q: %v | sem constante |
| internal/doctor/checks.go:61 | \n     existe(m) ao lado, com grafia diferente: %s | duplicata de textos.DetGrafiaVizinha |
| internal/doctor/checks.go:73 | %q existe mas não é um diretório | duplicata de textos.DetNaoEhDiretorio |
| internal/doctor/checks.go:87 | pelo nome %q, resolvido para %s | sem constante |
| internal/doctor/checks.go:92 | permissão de leitura | duplicata de textos.CheckLeitura |
| internal/doctor/checks.go:95 | varredura interrompida: %v | duplicata de textos.DetVarreduraInterrompida |
| internal/doctor/checks.go:103 | não foi possível listar %q: %v | duplicata de textos.DetNaoListou |
| internal/doctor/checks.go:106 | %d entradas na raiz | duplicata de textos.DetEntradasRaiz |
| internal/doctor/checks.go:115 | permissão de escrita | duplicata de textos.CheckEscrita |
| internal/doctor/checks.go:118 | varredura interrompida: %v | duplicata de textos.DetVarreduraInterrompida |
| internal/doctor/checks.go:136 | não foi possível escrever em %q: %v | duplicata de textos.DetNaoEscreveu |
| internal/doctor/checks.go:152 | .obsidian presente | duplicata de textos.CheckObsidian |
| internal/doctor/checks.go:155 | varredura interrompida: %v | duplicata de textos.DetVarreduraInterrompida |
| internal/doctor/checks.go:165 | pasta .obsidian ausente: configurações, temas e plugins do Obsidian não serão detectados | duplicata de textos.DetObsidianAusente |
| internal/doctor/checks.go:171 | não foi possível verificar %q: %v | duplicata de textos.DetNaoVerificou |
| internal/doctor/checks.go:177 | %q existe mas não é um diretório | duplicata de textos.DetNaoEhDiretorio |
| internal/doctor/checks.go:188 | contagem de notas | duplicata de textos.CheckNotas |
| internal/doctor/checks.go:194 | nenhuma nota .md encontrada | duplicata de textos.DetSemNotas |
| internal/doctor/checks.go:196 | %d notas | duplicata de textos.DetNotas |
| internal/doctor/checks.go:206 | comprimento de caminho | duplicata de textos.CheckCaminho |
| internal/doctor/checks.go:215 | %d caracteres, acima do limiar de %d: %s | duplicata de textos.DetCaminhoLongo |
| internal/doctor/checks.go:218 | maior caminho: %d caracteres | duplicata de textos.DetMaiorCaminho |
| internal/doctor/checks.go:224 | diretório de cache | duplicata de textos.CheckCache |
| internal/doctor/checks.go:227 | varredura interrompida: %v | duplicata de textos.DetVarreduraInterrompida |
| internal/doctor/checks.go:234 | nenhum diretório de cache configurado | duplicata de textos.DetSemCacheDir |
| internal/doctor/checks.go:241 | não foi possível criar %q: %v | duplicata de textos.DetNaoCriouCache |
| internal/doctor/checks.go:251 | espaço em disco | duplicata de textos.CheckEspaco |
| internal/doctor/checks.go:254 | varredura interrompida: %v | duplicata de textos.DetVarreduraInterrompida |
| internal/doctor/checks.go:259 | não foi possível medir espaço livre: %v | duplicata de textos.DetNaoMediuEspaco |
| internal/doctor/checks.go:262 | %d MB livres | duplicata de textos.DetEspacoLivre |
| internal/doctor/checks.go:321 | varredura interrompida: %v | duplicata de textos.DetVarreduraInterrompida |
| internal/doctor/checks.go:327 | cofre inacessível durante a varredura: %v | duplicata de textos.DetCofreInacessivel |
| internal/doctor/checks.go:372 | abrindo cofre: %w | sem constante (chega à tela dentro do detalhe acima) |
| internal/doctor/checks_windows.go:65 | caminhos longos habilitados | duplicata de textos.CheckCaminhosLongos |
| internal/doctor/checks_windows.go:78 | LongPathsEnabled != 1 no registro e há caminho de %d caracteres: %s | duplicata de textos.DetLongPaths |
| internal/doctor/checks_windows.go:107 | arquivos somente-nuvem | duplicata de textos.CheckSomenteNuvem |
| internal/doctor/checks_windows.go:116 | %d nota(s) ainda não baixada(s) pelo sincronizador de nuvem | duplicata de textos.DetSomenteNuvem |
| internal/doctor/checks_windows.go:127 | colisões de casing | duplicata de textos.CheckCasing |
| internal/doctor/checks_windows.go:136 | %d colisão(ões): %s | duplicata de textos.DetColisoes |
| internal/doctor/checks_windows.go:148 | resolvendo %q: %w | sem constante (chega à tela dentro de "não foi possível medir espaço livre") |
| internal/doctor/checks_windows.go:152 | convertendo %q: %w | sem constante (idem) |
| internal/doctor/checks_windows.go:157 | GetDiskFreeSpaceEx(%q): %w | sem constante (idem) |
| internal/doctor/checks_other.go:34 | statfs(%q): %w | sem constante (idem) |
| internal/doctor/daemon.go:55 | ausente | duplicata de textos.ClasseAusente |
| internal/doctor/daemon.go:57 | inacessível (%v) | duplicata de textos.ClasseInacessivel |
| internal/doctor/daemon.go:62 | socket | duplicata de textos.ClasseSocket |
| internal/doctor/daemon.go:64 | DIRETÓRIO (nenhum daemon consegue usar este caminho) | duplicata de textos.ClasseDiretorio |
| internal/doctor/daemon.go:66 | symlink | duplicata de textos.ClasseSymlink |
| internal/doctor/daemon.go:68 | arquivo comum de %d bytes (resíduo; nenhum daemon escuta aqui) | duplicata de textos.ClasseArquivo |
| internal/doctor/daemon.go:70 | outro (modo=%v) | duplicata de textos.ClasseOutro |
| internal/doctor/daemon.go:79 | caminho do socket do daemon | duplicata de textos.CheckSocket |
| internal/doctor/daemon.go:83 | nao foi possivel derivar: %v | duplicata de textos.DetNaoDerivou; sem acento |
| internal/doctor/daemon.go:87 | %s -- %s | duplicata de textos.DetSocketEClasse |
| internal/doctor/daemon.go:113 | diretório de sockets aceita conexão | duplicata de textos.CheckDiretorioSockets |
| internal/doctor/daemon.go:115 | %v -- a ponte deste contexto vai servir em processo em vez de usar o daemon | duplicata de textos.DetSondaFalhou |
| internal/doctor/daemon.go:117 | vale neste processo; num host o servidor roda noutro contexto, e a prova lá é a linha ... no log dele | duplicata de textos.DetSondaOK |
| internal/doctor/daemon.go:130 | daemon respondendo | duplicata de textos.CheckDaemonVivo |
| internal/doctor/daemon.go:135 | handshake completo | duplicata de textos.DetHandshakeOK |
| internal/doctor/daemon.go:140 | (caminho indisponível) | sem constante |
| internal/doctor/daemon.go:148 | nenhum daemon rodando (a ponte servirá em processo) | duplicata de textos.DetSemDaemon |
| internal/doctor/daemon.go:154 | arquivo existe mas o handshake falhou: %v | duplicata de textos.DetHandshakeFalhou |
| internal/doctor/daemon.go:164 | log do daemon | duplicata de textos.CheckLogDaemon |
| internal/doctor/daemon.go:168 | nao foi possivel derivar: %v | duplicata de textos.DetNaoDerivou; sem acento |
| internal/doctor/daemon.go:174 | ainda não existe (nenhum daemon rodou para este cofre) | duplicata de textos.DetLogAusente |
| internal/doctor/daemon.go:176 | %s: %v | duplicata de textos.DetCaminhoEErro |
| internal/doctor/daemon.go:183 | %s (%d bytes, última escrita há %s) | duplicata de textos.DetLogResumo |
| internal/doctor/daemon.go:185 | \n      \| ‹linha do log› | formato |
| internal/doctor/daemon.go:206 | travas de daemon em uso | duplicata de textos.CheckTravas |
| internal/doctor/daemon.go:210 | nao foi possivel derivar: %v | duplicata de textos.DetNaoDerivou; sem acento |
| internal/doctor/daemon.go:217 | diretório de runtime ainda não existe | duplicata de textos.DetRuntimeAusente |
| internal/doctor/daemon.go:219 | %s: %v | duplicata de textos.DetCaminhoEErro |
| internal/doctor/daemon.go:230 | %s (não foi possível consultar: %v) | duplicata de textos.DetTravaIlegivel |
| internal/doctor/daemon.go:241 | %s (PID %d) | duplicata de textos.DetTravaComPID |
| internal/doctor/daemon.go:248 | nenhuma trava em uso | duplicata de textos.DetSemTravas |
| internal/doctor/daemon.go:253 | %d em %s: %s | duplicata de textos.DetTravasEmUso |

### internal/instalar

Estes chegam à tela como a linha de falha de install, update, vaults ou path,
ou dentro de um aviso do doctor.

| Local | Texto | Situação |
|---|---|---|
| internal/instalar/instalar.go:277 | pid %d  %s  %s | sem constante (itens da pergunta de encerrar) |
| internal/instalar/instalar.go:279 | Encerrar estes processos do gobsidian para trocar o binario? | sem constante; sem acento |
| internal/instalar/path_other.go:116 | abra um terminal NOVO, ou rode `source ~/.profile`; fish e nushell leem outro arquivo e precisam da linha a mao | sem constante (o par Windows é textos.AvisoPathWindows); sem acento |
| internal/instalar/cofres.go:42 | lendo %s: %w | sem constante |
| internal/instalar/cofres.go:47 | %s nao e um JSON valido: %w | sem constante; sem acento |
| internal/instalar/instalar.go:152 | resolvendo o diretorio de runtime: %w | sem constante; sem acento |
| internal/instalar/instalar.go:180 | limpando: %w | sem constante |
| internal/instalar/instalar.go:192 | migrando chaves de cache: %w | sem constante |
| internal/instalar/instalar.go:210 | ajustando o PATH: %w | sem constante |
| internal/instalar/instalar.go:268 | listando processos: %w | sem constante |
| internal/instalar/instalar.go:309 | resolvendo o executavel corrente: %w | sem constante; sem acento |
| internal/instalar/instalar.go:320 | criando %s: %w | sem constante |
| internal/instalar/instalar.go:335 | tirando o binario antigo do caminho: %w | sem constante; sem acento |
| internal/instalar/instalar.go:364 | abrindo %s: %w | sem constante |
| internal/instalar/instalar.go:372 | criando %s: %w | sem constante |
| internal/instalar/instalar.go:376 | copiando para %s: %w | sem constante |
| internal/instalar/instalar.go:379 | fechando %s: %w | sem constante |
| internal/instalar/instalar.go:387 | abrindo %s para somar: %w | sem constante |
| internal/instalar/instalar.go:392 | somando %s: %w | sem constante |
| internal/instalar/limpeza.go:132 | lendo raiz do cache %s: %w | sem constante |
| internal/instalar/limpeza.go:165 | lendo diretorio de runtime %s: %w | sem constante; sem acento |
| internal/instalar/limpeza.go:234 | %s: %v (item de "não removido") | formato |
| internal/instalar/manifesto.go:50 | nao ha manifesto de instalacao | sem constante; sem acento (vira o %w de textos.ErroSemManifesto) |
| internal/instalar/manifesto.go:59 | lendo manifesto: %w | sem constante |
| internal/instalar/manifesto.go:63 | manifesto ilegivel em %s: %w | sem constante; sem acento |
| internal/instalar/manifesto.go:72 | criando diretorio do manifesto: %w | sem constante; sem acento |
| internal/instalar/manifesto.go:76 | serializando manifesto: %w | sem constante |
| internal/instalar/manifesto.go:79 | gravando manifesto: %w | sem constante |
| internal/instalar/manifesto.go:105 | resolvendo o executavel corrente: %w | sem constante; sem acento |
| internal/instalar/manifesto.go:110 | consultando o executavel corrente: %w | sem constante; sem acento |
| internal/instalar/migracao.go:70 | lendo raiz do cache %s: %w | sem constante |
| internal/instalar/path_other.go:26 | resolvendo o home do usuario: %w | sem constante; sem acento |
| internal/instalar/path_other.go:44 | lendo %s: %w | sem constante |
| internal/instalar/path_other.go:56 | abrindo %s: %w | sem constante |
| internal/instalar/path_other.go:60 | gravando em %s: %w | sem constante |
| internal/instalar/path_other.go:77 | lendo %s: %w | sem constante |
| internal/instalar/path_other.go:101 | lendo %s: %w | sem constante |
| internal/instalar/path_other.go:109 | gravando %s: %w | sem constante |
| internal/instalar/path_windows.go:35 | abrindo HKCU\%s: %w | sem constante |
| internal/instalar/path_windows.go:41 | lendo o PATH do usuario: %w | sem constante; sem acento |
| internal/instalar/path_windows.go:63 | gravando o PATH do usuario: %w | sem constante; sem acento |
| internal/instalar/path_windows.go:72 | abrindo HKCU\%s: %w | sem constante |
| internal/instalar/path_windows.go:81 | lendo o PATH do usuario: %w | sem constante; sem acento |
| internal/instalar/path_windows.go:102 | gravando o PATH do usuario: %w | sem constante; sem acento |
| internal/instalar/processos_windows.go:31 | listando processos: %w | sem constante |
| internal/instalar/processos_windows.go:49 | percorrendo processos: %w | sem constante |
| internal/instalar/trava_global.go:48 | travando instalacao: %w | sem constante; sem acento |
| internal/instalar/trava_global.go:51 | ja ha uma instalacao em curso (%s) | sem constante; sem acento |
| internal/instalar/presenca.go:73, :79, :85, :97, :101 | criando diretorio de runtime / travando presenca / presenca %s ja esta travada por outro processo / serializando presenca / gravando presenca | sem constante; sem acento; em serve vão para o log (cmd/gobsidian/serve.go:135), não à tela |

### internal/hosts

| Local | Texto | Situação |
|---|---|---|
| internal/hosts/hosts.go:102 | Claude Desktop | nome do host, sem constante |
| internal/hosts/hosts.go:116 | Claude Code (CLI) | nome do host, sem constante |
| internal/hosts/hosts.go:135 | Gemini CLI | nome do host, sem constante |
| internal/hosts/hosts.go:150 | Antigravity | nome do host, sem constante |
| internal/hosts/hosts.go:162 | Antigravity IDE | nome do host, sem constante |
| internal/hosts/hosts.go:174 | Codex CLI | nome do host, sem constante |
| internal/hosts/hosts.go:187 | VS Code | nome do host, sem constante |
| internal/hosts/hosts.go:204 | Cursor | nome do host, sem constante |
| internal/hosts/hosts.go:215 | Windsurf | nome do host, sem constante |
| internal/hosts/hosts.go:58 | %s: %w (%s) | formato (erro do CLI do host) |
| internal/hosts/hosts.go:256 | host %s nao sabe se configurar | sem constante; sem acento |
| internal/hosts/merge.go:96 | criando diretorio de %s: %w | sem constante; sem acento |
| internal/hosts/merge.go:108 | lendo %s: %w | sem constante |
| internal/hosts/merge.go:120 | %s nao e um JSON valido; nada foi alterado (backup em %s%s): %w | sem constante; sem acento |
| internal/hosts/merge.go:128 | mcpServers de %s nao e um objeto; nada foi alterado: %w | sem constante; sem acento |
| internal/hosts/merge.go:142 | serializando a entrada %q: %w | sem constante |
| internal/hosts/merge.go:149 | serializando mcpServers: %w | sem constante |
| internal/hosts/merge.go:155 | serializando %s: %w | sem constante |
| internal/hosts/merge.go:161 | gravando %s: %w | sem constante |
| internal/hosts/merge.go:171 | gravando backup de %s: %w | sem constante |
| internal/hosts/merge.go:188 | serializando a definicao para o VS Code: %w | sem constante; sem acento |
| internal/hosts/merge.go:210 | lendo %s: %w | sem constante |

### internal/selfupdate

| Local | Texto | Situação |
|---|---|---|
| internal/selfupdate/selfupdate.go:62 | host fora da lista permitida | sem constante |
| internal/selfupdate/selfupdate.go:67 | SHA-256 do arquivo baixado nao confere com o publicado | sem constante; sem acento (sai como a linha de falha depois de textos.UpdateHashDiverge) |
| internal/selfupdate/selfupdate.go:84 | %w: %q (permitidos: %s) | sem constante |
| internal/selfupdate/selfupdate.go:129 | lendo resposta da API de releases: %w | sem constante |
| internal/selfupdate/selfupdate.go:132 | a API devolveu um release sem tag | sem constante |
| internal/selfupdate/selfupdate.go:160 | release %s nao tem o ativo %q | sem constante; sem acento |
| internal/selfupdate/selfupdate.go:175 | criando diretorio de destino: %w | sem constante; sem acento |
| internal/selfupdate/selfupdate.go:179 | criando temporario de download: %w | sem constante; sem acento |
| internal/selfupdate/selfupdate.go:187 | gravando download: %w | sem constante |
| internal/selfupdate/selfupdate.go:190 | fechando download: %w | sem constante |
| internal/selfupdate/selfupdate.go:195 | %w: esperado %s, obtido %s | sem constante |
| internal/selfupdate/selfupdate.go:199 | movendo download para %s: %w | sem constante |
| internal/selfupdate/selfupdate.go:216 | release %s nao publica %s; sem ele nao ha o que conferir | sem constante; sem acento |
| internal/selfupdate/selfupdate.go:226 | lendo %s: %w | sem constante |
| internal/selfupdate/selfupdate.go:239 | %s nao lista %q | sem constante; sem acento |
| internal/selfupdate/transporte_http.go:48 | montando requisicao para %s: %w | sem constante; sem acento |
| internal/selfupdate/transporte_http.go:58 | buscando %s: %w | sem constante |
| internal/selfupdate/transporte_http.go:62 | buscando %s: status %d | sem constante |

### internal/config

Chegam à tela como a linha de falha de qualquer comando que abre um cofre.

| Local | Texto | Situação |
|---|---|---|
| internal/config/config.go:88 | caminho do cofre nao informado: use --vault | sem constante; sem acento |
| internal/config/config.go:92 | resolvendo caminho do cofre %q: %w | sem constante |
| internal/config/config.go:99 | GOBSIDIAN_LOG_LEVEL: %w | formato |
| internal/config/config.go:107 | --log-level: %w | formato |
| internal/config/config.go:116 | GOBSIDIAN_READ_ONLY: %w | formato |
| internal/config/config.go:127 | GOBSIDIAN_DEBOUNCE_MS: %w | formato |
| internal/config/config.go:133 | --debounce-ms: %w | formato |
| internal/config/config.go:141 | GOBSIDIAN_MAX_RESULTS: %w | formato |
| internal/config/config.go:147 | --max-results: %w | formato |
| internal/config/config.go:174 | nivel de log desconhecido: %q (use debug, info, warn ou error) | sem constante; sem acento |
| internal/config/config.go:189 | valor desconhecido: %q (use 1, true, t, yes, y, 0, false, f, no ou n) | sem constante |
| internal/config/config.go:198 | valor invalido %q (use um inteiro >= 1): %w | sem constante; sem acento |
| internal/config/config.go:214 | valor invalido %d (use um inteiro >= 1) | sem constante; sem acento |
| internal/config/config.go:222 | valor invalido %q (use um inteiro de 1 a %d): %w | sem constante; sem acento |
| internal/config/config.go:232 | valor invalido %d (deve ser entre 1 e %d) | sem constante; sem acento |
