// Package textos guarda, num lugar so, tudo o que os comandos de CLI escrevem
// na tela.
//
// # Por que um arquivo so
//
// Ate 2026-09-16 cada mensagem morava ao lado do codigo que a imprimia. O
// efeito pratico: a saida do produto misturava portugues correto com portugues
// sem acento -- "permissão de leitura" numa linha e "caminho da raiz do cofre
// (obrigatorio)" na ajuda da flag logo abaixo --, e nao havia como revisar a
// redacao sem percorrer catorze arquivos. Aqui da para ler tudo de uma vez.
//
// # Por que constantes Go, e nao JSON ou .txt embutido
//
// Quase todo texto daqui carrega verbo de formatacao (%s, %d, %q). Constante
// Go deixa `go vet` conferir que o verbo casa com o argumento no ponto de
// chamada -- um catalogo carregado em tempo de execucao perde essa conferencia
// e ainda acrescenta um modo de falha novo, a chave que nao existe. O texto do
// servidor MCP (internal/mcpsrv/instrucoes.txt) e embutido porque e um bloco
// unico sem formatacao; este e o caso oposto.
//
// # UTF-8 por padrao
//
// Tudo aqui e escrito em portugues correto, com acento. Quem decide se o
// acento chega a tela e console.adaptarTexto, pela code page do console no
// Windows e pelo locale no resto -- onde nao aguenta, o acento cai na hora de
// imprimir. Escrever "instalacao" no codigo para agradar um CP-850 e o que
// fazia o produto inteiro falar errado.
//
// # O que NAO mora aqui
//
// Nome de comando e de flag (`install`, `--vault`), chave de host e valor de
// enum: sao interface de linha de comando, nao prosa -- mudam o contrato com
// quem automatiza, e nao a leitura de quem opera. Mensagem de log tambem nao:
// log sai por slog, em ingles de maquina, e nao passa pelo console.
package textos

// Ajuda dos comandos: o `Short` e o `Long` que o cobra mostra.
const (
	ResumoRaiz = "Servidor MCP para cofres locais do Obsidian"

	ResumoServe   = "Serve o cofre via MCP sobre stdio"
	ResumoDaemon  = "Roda o daemon de cofre compartilhado (uso interno da ponte)"
	ResumoDoctor  = "Diagnostica o ambiente: permissões, OneDrive, MAX_PATH, casing"
	ResumoIndex   = "Constrói o índice do cofre e exibe um resumo"
	ResumoInspect = "Exibe metadados, links e backlinks de uma nota"
	ResumoSearch  = "Executa busca por texto completo no cofre"
	ResumoVersion = "Imprime versão, commit e data de build"

	ResumoInstall    = "Instala o gobsidian, ajusta o PATH e configura os hosts de IA"
	DescricaoInstall = "Instala o executável no perfil do usuário -- nunca pede elevação --, " +
		"pergunta se deve acrescentar o diretório ao PATH e registrar o servidor " +
		"nos hosts de IA detectados, e limpa lixo comprovadamente órfão de execuções anteriores."

	ResumoPath   = "Acrescenta ou remove o diretório de instalação do PATH do usuário"
	ResumoVaults = "Configura os hosts de IA para um cofre, sem reinstalar o binário"

	ResumoUpdate    = "Atualiza o gobsidian para a última versão publicada"
	DescricaoUpdate = "Consulta a versão publicada, baixa o binário da plataforma corrente, " +
		"CONFERE o SHA-256 publicado e -- só então -- encerra os processos em execução " +
		"e troca o binário. Divergência de soma aborta sem instalar nada."
)

// Ajuda das flags. O nome da flag fica no codigo; o que muda aqui e a frase.
const (
	FlagVault          = "caminho da raiz do cofre (obrigatório)"
	FlagFollowSymlinks = "segue symlink dentro do cofre; o padrão recusa, porque o confinamento não alcança o alvo"
	FlagCacheDir       = "diretório do cache de índice"
	FlagLogLevel       = "debug, info, warn ou error"
	FlagJSON           = "saída estruturada em formato JSON"
	FlagReadOnly       = "desabilita toda a superfície de escrita"
	FlagDebounce       = "janela de coalescência de eventos do watcher"
	FlagMaxResults     = "teto de resultados por consulta"
	FlagEagerSearch    = "carrega o índice de busca no boot em vez de esperar a primeira vault_search"
	FlagIdleSeconds    = "segundos sem cliente conectado antes do daemon encerrar (decisão 3 da Task 92; padrão 15 minutos)"

	FlagDoctorReadOnly = "não verifica permissão de escrita"
	FlagDoctorFix      = "além de diagnosticar, remove o lixo comprovadamente órfão do diretório de runtime e do cache"

	FlagSearchLimit = "limite máximo de resultados"

	FlagInstallVault    = "cofre a servir (padrão: perguntar, lendo o registro do Obsidian)"
	FlagInstallDir      = "onde pôr o binário (padrão: "
	FlagInstallHosts    = "hosts a configurar, separados por vírgula ("
	FlagInstallHostsFim = "); 'none' não configura nenhum"
	FlagInstallYes      = "não pergunta nada: instala, ajusta o PATH e configura os hosts detectados"
	FlagInstallReadOnly = "registra o servidor com --read-only"
	FlagInstallNoPath   = "não mexe no PATH"

	FlagPathAdd    = "acrescenta o diretório ao PATH"
	FlagPathRemove = "remove o diretório do PATH"

	FlagUpdateCheck = "só diz se há versão nova, sem baixar nem instalar"
	FlagUpdateYes   = "não pergunta antes de encerrar os processos em execução"
)

// Erros que o usuario le no terminal.
const (
	ErroIdleSeconds      = "--idle-seconds precisa ser >= 1 (recebido %d)"
	ErroResolvendoNota   = "resolvendo nota %q: %w"
	ErroNotaNaoIndexada  = "nota %q não encontrada no índice"
	ErroSemCofre         = "nenhum cofre encontrado; passe --vault com o caminho do cofre"
	ErroSelecaoCancelada = "seleção cancelada; nada foi alterado"
	ErroHostDesconhecido = "host desconhecido %q; conhecidos: %s"
	ErroAddOuRemove      = "escolha exatamente um: --add ou --remove"
	ErroSemManifesto     = "%w -- rode `gobsidian install` primeiro"
	ErroConsultarRelease = "consultando releases: %w"
	ErroAtivoAusente     = "o release %s não publica %q (plataforma %s/%s)"
	ErroTemporario       = "criando diretório temporário: %w"
	ErroGerandoScript    = "gerando o script de %s: %w"
)

// `gobsidian` sem argumento nenhum: a autoinstalacao da decisao D-11.
const (
	AutoinstalarNaoInstalado = "gobsidian ainda não está instalado nesta máquina"
	AutoinstalarDestino      = "este executável vai se instalar em %s"
	AutoinstalarAjuda        = "para só ver a ajuda, rode `gobsidian --help`"
)

// `version`, `index`, `inspect` e `search`.
const (
	CampoCommit = "commit"
	CampoBuild  = "build"

	IndexConcluido  = "Indexação concluída em %d ms"
	IndexTitulo     = "Índice"
	CampoOrigem     = "origem"
	CampoNotas      = "notas"
	CampoAnexos     = "anexos"
	CampoTags       = "tags"
	CampoTamanho    = "tamanho"
	NotaBytes       = "bytes"
	CampoTitulo     = "título"
	CampoHeadings   = "headings"
	CampoLinksSaida = "links de saída"
	CampoBacklinks  = "backlinks"

	SearchSemResultado = "Nenhum resultado para %q"
)

// `install`, `vaults` e `path`.
const (
	InstallSemTerminal      = "a entrada não é um terminal: usando as respostas padrão"
	InstallSemTerminalDica  = "para escolher os cofres, rode `gobsidian install` num terminal"
	InstallTitulo           = "Instalando"
	InstallCancelado        = "Instalação cancelada; nada foi alterado"
	InstallConcluido        = "Instalado"
	InstallRegistroIlegivel = "não foi possível ler o registro de cofres do Obsidian: %v"
	InstallCofreEscolhido   = "cofre: %s"
	InstallConfigAtual      = "Configuração atual"
	InstallManterConfig     = "Manter esta configuração?"
	InstallQuaisCofres      = "Quais cofres configurar?"
	InstallQuaisHosts       = "Em quais hosts registrar?"
	InstallSemHosts         = "nenhum host de IA conhecido foi detectado"
	InstallHostsEncontrados = "Hosts de IA encontrados"
	InstallEscolhaDigitada  = "Números separados por espaço, * para todos, vazio para nenhum"
	InstallItemNumerado     = "%d) %s  %s"
	InstallItemHost         = "%d) %s  (%s)"
	InstallYesEncerrando    = "--yes: encerrando sem perguntar"

	InstallHostsConfigurados = "Hosts configurados"
	InstallSemCofreNaLista   = "cofres   nenhum configurado"
	InstallCofreNaLista      = "cofre    %s"
	InstallHostNaLista       = "%-16s %s"
	InstallHostFalhou        = "%s não pode ser configurado"

	ResumoBinario   = "binário  %s"
	ResumoCofres    = "cofres   nenhum configurado"
	ResumoCofre     = "cofre    %s"
	ResumoPATH      = "PATH     %s"
	ResumoEncerrado = "encerrado pid %d (%s)"
	ResumoHostOK    = "%-16s %s"
	ResumoHostFalha = "%-16s FALHOU: %s"
	ResumoLimpeza   = "limpeza  %s trava(s), %s socket(s), %s presença(s), %s cache(s), %d KB"

	PathJaEstava   = "PATH já estava como você pediu"
	PathAtualizado = "PATH atualizado"
)

// `update`.
const (
	UpdateConsultando  = "Consultando a última versão publicada"
	UpdateInstalada    = "instalada: %s"
	UpdatePublicada    = "publicada: %s"
	UpdateJaAtual      = "Já está na última versão"
	UpdateHaVersaoNova = "Há versão nova: %s"
	UpdateComoInstalar = "rode `gobsidian update` para instalar"
	UpdateBaixando     = "Baixando %s e conferindo o SHA-256"
	UpdateHashDiverge  = "O binário baixado NÃO confere com a soma publicada"
	UpdateNadaMudou    = "nada foi instalado; sua instalação continua intacta"
	UpdateHashConfere  = "SHA-256 confere"
	UpdateTrocando     = "Trocando o binário"
	UpdateCancelado    = "Atualização cancelada; nada foi alterado"
	UpdateConcluido    = "Atualizado para %s"
	UpdateHostsSozinho = "os hosts reiniciam o servidor sozinhos; não há o que fazer à mão"
)

// `doctor`: o relatorio, os processos e o lixo.
const (
	DoctorTitulo       = "Diagnóstico do ambiente"
	DoctorFalhasAcima  = "Há falhas bloqueantes acima"
	DoctorAmbienteApto = "Ambiente apto"
	DoctorResumo       = "%d verificações: %s, %s, %s"
	DoctorResumoOK     = "ok"
	DoctorResumoAviso  = "aviso"
	DoctorResumoAvisos = "avisos"
	DoctorResumoFalha  = "falha"
	DoctorResumoFalhas = "falhas"

	DoctorRuntimeIndisponivel = "processos e lixo: diretório de runtime indisponível (%v)"
	DoctorProcessosErro       = "processos do gobsidian: %v"
	DoctorProcessos           = "processos do gobsidian"
	DoctorNenhumProcesso      = "nenhum rodando"
	DoctorProcessosContagem   = "%d processo(s) do gobsidian"
	DoctorProcessosRodape     = "um servidor por sessão do host; o daemon é um só por cofre"
	DoctorGravadoresDuplos    = "%d processos gravam o cache do mesmo cofre"
	DoctorGravadoresDetalhe   = "%s -- gravadores: %s; encerre os extras. Pontes não gravam e ficam fora desta conta"
	DoctorSemModoDetalhe      = "%s -- %s sem modo registrado (versão anterior): não dá para saber se gravam"

	DoctorSemPresencaPlataforma = "processos do gobsidian sem presença: não verificado nesta plataforma"
	DoctorSemPresencaErro       = "processos do gobsidian sem presença: %v"
	DoctorSemPresencaNenhum     = "processos do gobsidian sem presença"
	DoctorSemPresencaVazio      = "nenhum"
	DoctorSemPresencaContagem   = "%d processo(s) do gobsidian sem presença"
	DoctorSemPresencaRodape     = "binário anterior à presença, ou de outra instalação -- o doctor não sabe o modo nem o cofre deles"

	DoctorHostsSemManifesto = "binário dos hosts: sem instalação registrada, nada a comparar"
	DoctorHostsErro         = "binário dos hosts: %v"
	DoctorHostsOK           = "binário dos hosts"
	DoctorHostsDetalhe      = "toda entrada do gobsidian nos configs de arquivo roda %s"
	DoctorHostsOutroBinario = "%d entrada(s) de host rodam outro binário"
	DoctorHostsRodape       = "o instalado é %s (%s); `gobsidian install` reconfigura. Claude Code, Gemini CLI, Codex e VS Code guardam a config no próprio CLI e não entram aqui"
	DoctorVersaoNaoMedida   = "versão não medida"
	DoctorComandoAusente    = "comando não encontrado"

	DoctorChavesErro     = "chaves de cache: %v"
	DoctorChavesSuperada = "%d cache(s) sob chave superada"
	DoctorChavesRodape   = "rode `gobsidian update` para renomear com tudo encerrado"

	DoctorLixoErro         = "lixo do diretório de runtime: %v"
	DoctorLixoNenhum       = "lixo de execuções anteriores"
	DoctorLixoNadaARemover = "nada a remover"
	DoctorLixoTitulo       = "lixo de execuções anteriores"
	DoctorLixoRemovivel    = "removível"
	DoctorLixoRemovido     = "removido"
	DoctorLixoTravas       = "travas"
	DoctorLixoSockets      = "sockets"
	DoctorLixoPresencas    = "presenças"
	DoctorLixoCaches       = "caches"
	DoctorLixoCachesNota   = "de cofre inexistente"
	DoctorLixoLogs         = "logs rotacionados"
	DoctorLixoTotal        = "total"
	DoctorLixoComoLimpar   = "rode `gobsidian doctor --fix` para remover, ou `gobsidian update`, que já limpa"
	DoctorLixoNaoRemovido  = "não removido: %s"

	DoctorModoDaemon        = "daemon"
	DoctorModoDaemons       = "daemons"
	DoctorModoPonte         = "ponte"
	DoctorModoPontes        = "pontes"
	DoctorModoEmProcesso    = "servidor em processo"
	DoctorModoEmProcessos   = "servidores em processo"
	DoctorModoNaoRegistrado = "modo não registrado"
	DoctorSemCofre          = "(sem cofre registrado)"
)

// Os nomes das verificações do `doctor`: o que abre cada linha do relatório.
const (
	CheckRaizExiste       = "raiz do cofre existe"
	CheckLeitura          = "permissão de leitura"
	CheckEscrita          = "permissão de escrita"
	CheckObsidian         = ".obsidian presente"
	CheckNotas            = "contagem de notas"
	CheckCaminho          = "comprimento de caminho"
	CheckCache            = "diretório de cache"
	CheckEspaco           = "espaço em disco"
	CheckCaminhosLongos   = "caminhos longos habilitados"
	CheckSomenteNuvem     = "arquivos somente-nuvem"
	CheckCasing           = "colisões de casing"
	CheckSocket           = "caminho do socket do daemon"
	CheckDiretorioSockets = "diretório de sockets aceita conexão"
	CheckDaemonVivo       = "daemon respondendo"
	CheckLogDaemon        = "log do daemon"
	CheckTravas           = "travas de daemon em uso"
)

// Os detalhes de cada verificação: o número ou o caminho que torna o resultado
// acionável.
const (
	DetVarreduraInterrompida = "varredura interrompida: %v"
	DetCofreInacessivel      = "cofre inacessível durante a varredura: %v"
	DetNaoEhDiretorio        = "%q existe mas não é um diretório"
	DetGrafiaVizinha         = "\n     existe(m) ao lado, com grafia diferente: %s"
	DetNaoListou             = "não foi possível listar %q: %v"
	DetEntradasRaiz          = "%d entradas na raiz"
	DetNaoEscreveu           = "não foi possível escrever em %q: %v"
	DetObsidianAusente       = "pasta .obsidian ausente: configurações, temas e plugins do Obsidian não serão detectados"
	DetNaoVerificou          = "não foi possível verificar %q: %v"
	DetSemNotas              = "nenhuma nota .md encontrada"
	DetNotas                 = "%d notas"
	DetCaminhoLongo          = "%d caracteres, acima do limiar de %d: %s"
	DetMaiorCaminho          = "maior caminho: %d caracteres"
	DetSemCacheDir           = "nenhum diretório de cache configurado"
	DetNaoCriouCache         = "não foi possível criar %q: %v"
	DetEspacoLivre           = "%d MB livres"
	DetNaoMediuEspaco        = "não foi possível medir espaço livre: %v"

	DetLongPaths    = "LongPathsEnabled != 1 no registro e há caminho de %d caracteres: %s"
	DetSomenteNuvem = "%d nota(s) ainda não baixada(s) pelo sincronizador de nuvem"
	DetColisoes     = "%d colisão(ões): %s"

	DetNaoDerivou      = "não foi possível derivar: %v"
	DetSondaFalhou     = "%v -- a ponte deste contexto vai servir em processo em vez de usar o daemon"
	DetSondaOK         = "vale neste processo; num host o servidor roda noutro contexto, e a prova lá é a linha `conectado ao daemon` no log dele"
	DetHandshakeOK     = "handshake completo"
	DetSemDaemon       = "nenhum daemon rodando (a ponte servirá em processo)"
	DetHandshakeFalhou = "arquivo existe mas o handshake falhou: %v"
	DetLogAusente      = "ainda não existe (nenhum daemon rodou para este cofre)"
	DetLogResumo       = "%s (%d bytes, última escrita há %s)"
	DetCaminhoEErro    = "%s: %v"
	DetRuntimeAusente  = "diretório de runtime ainda não existe"
	DetSemTravas       = "nenhuma trava em uso"
	DetTravasEmUso     = "%d em %s: %s"
	DetTravaIlegivel   = "%s (não foi possível consultar: %v)"
	DetTravaComPID     = "%s (PID %d)"
	DetSocketEClasse   = "%s -- %s"
)

// O que existe no caminho do socket. O texto é o diagnóstico: "ausente" e
// "socket" são os dois estados saudáveis, e o resto é resíduo que impede o
// daemon de subir.
const (
	ClasseAusente     = "ausente"
	ClasseSocket      = "socket"
	ClasseDiretorio   = "DIRETÓRIO (nenhum daemon consegue usar este caminho)"
	ClasseSymlink     = "symlink"
	ClasseArquivo     = "arquivo comum de %d bytes (resíduo; nenhum daemon escuta aqui)"
	ClasseOutro       = "outro (modo=%v)"
	ClasseInacessivel = "inacessível (%v)"
)

// Os passos que a instalação anuncia antes de cada etapa começar, e o aviso de
// PATH que ela imprime no fim. Ver instalar.Sistema.Passo: quem anuncia é o
// pacote, e o texto é o que o usuário lê esperando.
const (
	PassoTrava     = "tomando a trava de instalação"
	PassoProcessos = "procurando processos em execução"
	PassoLimpeza   = "limpando lixo de execuções anteriores"
	PassoChaves    = "conferindo as chaves de cache"
	PassoBinario   = "instalando o binário"
	PassoPath      = "ajustando o PATH"
	PassoHosts     = "configurando os hosts de IA"
	PassoManifesto = "gravando o manifesto"

	AvisoPathWindows = "abra um terminal NOVO para o PATH atualizado valer; a sessão atual mantém o PATH antigo"
)

// O que cada host pede depois de ser configurado. Sai no resumo da instalação,
// uma linha por host.
const (
	HostClaudeDesktop  = "Reinicie o Claude Desktop para carregar o servidor."
	HostClaudeCode     = "Registrado no Claude Code."
	HostGeminiCLI      = "Registrado no Gemini CLI."
	HostAntigravity    = "Reinicie o Antigravity."
	HostAntigravityIDE = "Reinicie o Antigravity IDE."
	HostCodex          = "Registrado em ~/.codex/config.toml."
	HostVSCode         = "Registrado na configuração de usuário do VS Code."
	HostCursor         = "Reinicie o Cursor."
	HostWindsurf       = "Reinicie o Windsurf."
)

// Erros do `daemon` que chegam ao terminal de quem o roda à mão. O log do
// daemon é outra coisa: sai por slog e não passa por aqui.
const (
	ErroLogCaminho   = "resolvendo caminho do log do daemon: %w"
	ErroLogDiretorio = "criando diretório do log do daemon: %w"
	ErroLogAbrir     = "abrindo log do daemon %s: %w"
	ErroSocketDaemon = "abrindo socket do daemon: %w"
)

// Como carregar o autocompletar em cada shell que o cobra não cobre sozinho.
const (
	ShellNushell = "Acrescente ao seu config.nu:\n\n" +
		"  let gobsidian_completer = {|spans| gobsidian _carapace nushell ...$spans | from json }\n" +
		"  $env.config.completions.external = { enable: true, completer: $gobsidian_completer }"
	ShellElvish  = "Acrescente ao seu rc.elv:\n\n  eval (gobsidian completion elvish | slurp)"
	ShellIon     = "Acrescente ao seu initrc:\n\n  eval $(gobsidian completion ion)"
	ShellOil     = "Acrescente ao seu oshrc:\n\n  source <(gobsidian completion oil)"
	ShellTcsh    = "Acrescente ao seu .tcshrc:\n\n  eval `gobsidian completion tcsh`"
	ShellXonsh   = "Acrescente ao seu .xonshrc:\n\n  exec($(gobsidian completion xonsh))"
	ShellClink   = "Salve a saída em um arquivo .lua dentro do diretório de scripts do clink."
	ShellBashBLE = "Acrescente ao seu .bashrc, DEPOIS de carregar o ble.sh:\n\n  source <(gobsidian completion bash_ble)"
)

// Autocompletar: o que o shell mostra ao lado de cada valor.
const (
	CompletarNenhumHost  = "não configura nenhum host"
	CompletarCofre       = "cofre do Obsidian"
	CompletarCofreAberto = "aberto agora"
	CompletarLogDebug    = "tudo, inclusive o que só interessa depurando"
	CompletarLogInfo     = "o padrão"
	CompletarLogWarn     = "só o que pede atenção"
	CompletarLogError    = "só falha"
	CompletarResumoShell = "Gera o script de autocompletar para %s"
	CompletarDescricao   = "Gera o script de autocompletar para %s.\n\n%s"
)
